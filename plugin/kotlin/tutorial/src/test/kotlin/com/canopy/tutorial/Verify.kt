package com.canopy.tutorial

import java.security.MessageDigest

/*
 * Verify.kt is a self-contained, dependency-free port of Canopy's optimized Sparse Merkle Tree proof
 * verification (store/smt.go). It reconstructs the tree from a proof and checks membership or
 * non-membership against a trusted root, using the exact same key encoding and hashing (sha256) as
 * the node, so proofs produced by the node verify here without importing or running it.
 *
 * It does NOT import or run a Canopy node: it reconstructs the sparse-merkle-tree root from the proof
 * and checks membership / non-membership against a state root the caller has authenticated
 * independently (via the quorum certificate for the height).
 *
 * Node leaf keys are sha256(rawKey) and leaf values are sha256(value); internal node values are the
 * hash of (leftKey || leftValue || rightKey || rightValue).
 *
 * Trust model: the ONLY thing you must trust is the state root for the height (authenticated out of
 * band). The value, the proof, and existence are all VERIFIED here against that root — so a lying or
 * buggy RPC node can at most make verification FAIL; it can never forge an accepted answer.
 */

/**
 * ProofNode is a single sparse-merkle-tree proof node (the leaf being proven, followed by the sibling
 * nodes along the path to the root). It mirrors the `proof` array shape returned by the plugin RPC.
 */
data class ProofNode(val key: ByteArray, val value: ByteArray, val bitmask: Int)

/**
 * verifyKeyProof verifies a membership (exists=true) or non-membership (exists=false) proof for the
 * given raw state key and committed value against a trusted state root. It returns true when the
 * proof is valid for the claimed existence, and false for ANY mismatch: a tampered value, a tampered
 * proof node, a wrong/forged root, or a malformed proof. It never throws on adversarial input.
 *
 *  - rawKey: the raw (length-prefixed) state key, e.g. keyForFaucet(addr). The verifier hashes it
 *    itself (leaf keys are sha256(rawKey)); a trustless caller should recompute it rather than trust
 *    the node's echoed key.
 *  - value: the raw committed bytes the proof commits to (ignored when exists=false).
 *  - exists: whether this should be a membership (true) or non-membership (false) proof.
 *  - root: the trusted state root the proof must reconstruct to.
 *  - proof: the proof nodes from the RPC response.
 */
fun verifyKeyProof(rawKey: ByteArray, value: ByteArray, exists: Boolean, root: ByteArray, proof: List<ProofNode>): Boolean =
    verifyProof(rawKey, value, exists, root, proof, MAX_KEY_BIT_LENGTH)

private const val MAX_KEY_BIT_LENGTH = 160 // 20-byte (sha256-truncated) leaf keys
private const val LEFT_CHILD = 0
private const val RIGHT_CHILD = 1

// rootKeyBytes is the arbitrary root key (midpoint between min and max), matching store.RootKey.
private val ROOT_KEY_BYTES = byteArrayOf(
    0x7F, 0xFF.toByte(), 0xFF.toByte(), 0xFF.toByte(), 0xFF.toByte(), 0xFF.toByte(), 0xFF.toByte(), 0xFF.toByte(),
    0xFF.toByte(), 0xFF.toByte(), 0xFF.toByte(), 0xFF.toByte(), 0xFF.toByte(), 0xFF.toByte(), 0xFF.toByte(), 0xFF.toByte(),
    0xFF.toByte(), 0xFF.toByte(), 0xFF.toByte(), 0xFF.toByte(), 0xFF.toByte(), 0xFF.toByte(), 0xFF.toByte(), 0xFF.toByte(),
    0xFF.toByte(), 0xFF.toByte(), 0xFF.toByte(), 0xFF.toByte(), 0xFF.toByte(), 0xFF.toByte(), 0xFF.toByte(), 0xFF.toByte()
)

// hash() is the global hashing algorithm (sha256), matching lib/crypto.Hash.
private fun hash(msg: ByteArray): ByteArray = MessageDigest.getInstance("SHA-256").digest(msg)

// concat() returns a freshly-allocated concatenation of the parts.
private fun concat(vararg parts: ByteArray): ByteArray {
    var total = 0
    for (p in parts) total += p.size
    val out = ByteArray(total)
    var offset = 0
    for (p in parts) {
        System.arraycopy(p, 0, out, offset, p.size)
        offset += p.size
    }
    return out
}

// leadingZeros8 mirrors math/bits.LeadingZeros8 (8 for 0, 0 for 0xFF).
private fun leadingZeros8(b: Int): Int = Integer.numberOfLeadingZeros(b and 0xFF) - 24

// len8 mirrors math/bits.Len8 (0 for 0, 8 for 0xFF).
private fun len8(b: Int): Int = 32 - Integer.numberOfLeadingZeros(b and 0xFF)

// IntRef is a tiny mutable holder used to mirror Go's `*int` pass-by-reference in greatestCommonPrefix.
private class IntRef(var value: Int)

// skey derives a stable, collision-free map key from raw bytes (ISO-8859-1 maps bytes 1:1 to chars),
// mirroring Go's `string(k)` map keys.
private fun skey(b: ByteArray): String = String(b, Charsets.ISO_8859_1)

// ============================ node ============================

// Node mirrors the fields of store's node used during verification.
private class Node {
    var key: Key? = null
    var value: ByteArray? = null
    var leftChildKey: ByteArray? = null
    var rightChildKey: ByteArray? = null
    var bitmask: Int = 0

    // copyNode() is a shallow copy (shares Key pointer and value slice), mirroring Go's node.copy().
    fun copyNode(): Node {
        val c = Node()
        c.key = key
        c.value = value
        c.leftChildKey = leftChildKey
        c.rightChildKey = rightChildKey
        c.bitmask = bitmask
        return c
    }
}

private fun newNode(): Node {
    val n = Node()
    n.key = Key()
    return n
}

// ============================ smt (verification-only) ============================

private class Smt(rootKey: ByteArray, val keyBitLength: Int, val store: HashMap<String, Node>) {
    lateinit var root: Node
    val minKey: Key = newNodeKey(ByteArray(20) { 0 }, keyBitLength)
    val maxKey: Key = newNodeKey(ByteArray(20) { 0xFF.toByte() }, keyBitLength)

    // traversal state
    var current: Node? = null
    var target: Node? = null
    var traversed: MutableList<Node> = mutableListOf()
    var gcp: Key = Key()
    var bitPos: Int = 0
    var pathBit: Int = 0

    init {
        val rKey = newNodeKey(rootKey, keyBitLength)
        root = getNode(rKey.bytes())
        if (root.leftChildKey == null) {
            initializeTree(rKey)
        }
        reset()
    }

    // initializeTree() ensures the tree always has a root with two border children.
    private fun initializeTree(rootKey: Key) {
        val minNode = Node()
        minNode.key = newNodeKey(ByteArray(20) { 0 }, keyBitLength)
        minNode.value = ByteArray(20) { 0 }
        val maxNode = Node()
        maxNode.key = newNodeKey(ByteArray(20) { 0xFF.toByte() }, keyBitLength)
        maxNode.value = ByteArray(20) { 0xFF.toByte() }
        setNode(minNode)
        setNode(maxNode)
        val r = Node()
        r.key = rootKey
        r.leftChildKey = minNode.key!!.bytes()
        r.rightChildKey = maxNode.key!!.bytes()
        root = r
        updateParentValue(root)
        setNode(root)
    }

    // updateParentValue() recomputes a parent's value from its children (left/right key+value).
    private fun updateParentValue(parent: Node) {
        val lChild = getNode(parent.leftChildKey!!)
        val rChild = getNode(parent.rightChildKey!!)
        val lk = lChild.key!!.bytes()
        val lv = lChild.value!!
        val rk = rChild.key!!.bytes()
        val rv = rChild.value!!
        val input = ByteArray(lk.size + lv.size + rk.size + rv.size)
        var offset = 0
        System.arraycopy(lk, 0, input, offset, lk.size); offset += lk.size
        System.arraycopy(lv, 0, input, offset, lv.size); offset += lv.size
        System.arraycopy(rk, 0, input, offset, rk.size); offset += rk.size
        System.arraycopy(rv, 0, input, offset, rv.size)
        parent.value = hash(input)
        if (parent.key!!.bytes().contentEquals(root.key!!.bytes())) {
            root = parent.copyNode()
        }
    }

    fun getNode(k: ByteArray): Node = store[skey(k)] ?: newNode()

    fun setNode(n: Node) {
        store[skey(n.key!!.bytes())] = n
    }

    fun reset() {
        current = root.copyNode()
        traversed = mutableListOf()
        resetGCP()
    }

    private fun resetGCP() {
        gcp = Key()
        pathBit = 0
        bitPos = 0
    }

    // traverse() navigates the tree downward to locate the target or its closest position.
    fun traverse() {
        while (true) {
            traversed.add(current!!.copyNode())
            pathBit = target!!.key!!.bitAt(bitPos)
            val currentKey: ByteArray = when (pathBit) {
                LEFT_CHILD -> current!!.leftChildKey ?: ByteArray(0)
                else -> current!!.rightChildKey ?: ByteArray(0)
            }
            current = getNode(currentKey)
            current!!.key!!.fromBytes(currentKey)
            val bp = IntRef(bitPos)
            target!!.key!!.greatestCommonPrefix(bp, gcp, current!!.key!!)
            bitPos = bp.value
            if (!current!!.key!!.equalsKey(gcp) || target!!.key!!.equalsKey(gcp)) {
                return
            }
        }
    }

    // validateTargetOk() returns false for reserved keys (root, min, max).
    fun validateTargetOk(n: Node): Boolean {
        if (root.key!!.bytes().contentEquals(n.key!!.bytes())) return false
        if (minKey.bytes().contentEquals(n.key!!.bytes())) return false
        if (maxKey.bytes().contentEquals(n.key!!.bytes())) return false
        return true
    }
}

// verifyProof() reconstructs the root hash from the proof and compares it against the provided root,
// then confirms membership (value matches) or non-membership (target absent). Faithful port of
// store.SMT.VerifyProof.
//
// The proof is fully attacker-controlled: a malicious server can send malformed keys/bitmasks that
// drive the faithful SMT port into out-of-range indexing. Treat any such exception as a failed proof
// rather than crashing the client (defense against a client-side DoS).
private fun verifyProof(
    k: ByteArray,
    v: ByteArray,
    validateMembership: Boolean,
    root: ByteArray,
    proof: List<ProofNode>,
    keyBitLength: Int
): Boolean {
    return try {
        val proofLen = proof.size
        if (proofLen < 2) return false
        var h = proof[0].value
        var currentKey = Key().fromBytes(proof[0].key)
        val memStore = HashMap<String, Node>()
        val s = Smt(ROOT_KEY_BYTES, keyBitLength, memStore)
        // set the node being proven (leaf or potential location)
        val leaf = Node()
        leaf.value = proof[0].value
        leaf.key = currentKey
        s.setNode(leaf)
        // reconstruct the tree bottom-up using the proof siblings
        for (i in 1 until proofLen) {
            val parentNode = Node()
            if (proof[i].bitmask == LEFT_CHILD) {
                h = hash(concat(proof[i].key, proof[i].value, currentKey.bytes(), h))
                parentNode.leftChildKey = proof[i].key
                parentNode.rightChildKey = currentKey.bytes()
            } else {
                h = hash(concat(currentKey.bytes(), h, proof[i].key, proof[i].value))
                parentNode.leftChildKey = currentKey.bytes()
                parentNode.rightChildKey = proof[i].key
            }
            val nodeKey = Key().fromBytes(proof[i].key)
            val gcp = Key()
            // NOTE: faithfully mirrors store.VerifyProof (the first comparison is a no-op there too).
            if (currentKey.totalBits() < currentKey.totalBits()) {
                currentKey.greatestCommonPrefix(IntRef(0), gcp, nodeKey)
            } else {
                nodeKey.greatestCommonPrefix(IntRef(0), gcp, currentKey)
            }
            currentKey = gcp
            parentNode.value = h
            parentNode.key = currentKey
            s.setNode(parentNode)
            if (i == proofLen - 1) {
                s.root = parentNode
            }
        }
        // compare the reconstructed root against the provided (trusted) root
        if (!h.contentEquals(root)) return false
        // traverse the reconstructed tree to confirm presence/absence of the target
        val target = Node()
        target.key = newNodeKey(hash(k), s.keyBitLength)
        s.target = target
        if (!s.validateTargetOk(target)) return false
        s.reset()
        s.traverse()
        val nodeExists = target.key!!.equalsKey(s.gcp)
        if ((!nodeExists && validateMembership) || (nodeExists && !validateMembership)) return false
        if (!nodeExists && !validateMembership) return true
        // membership: the proven value must hash to the leaf value
        proof[0].value.contentEquals(hash(v))
    } catch (e: Throwable) {
        false
    }
}

// ============================ key ============================
//
// Key is a ByteArray representation of a left-to-right bit-string where the final byte stores the
// number of left-padding zeroes in the preceding data byte. Faithful port of store's key type.

private class Key {
    var keyBytes: ByteArray = ByteArray(0)
    var bitCount: Int = 0
    var length: Int = 0

    fun greatestCommonPrefix(bitPos: IntRef, gcp: Key, current: Key) {
        val totalBits = current.totalBits()
        while (bitPos.value < totalBits) {
            val bit1 = bitAt(bitPos.value)
            val bit2 = current.bitAt(bitPos.value)
            if (bit1 != bit2) break
            gcp.addBit(bit1)
            bitPos.value++
        }
    }

    fun bitAt(bitPos: Int): Int {
        val byteIndex = bitPos / 8
        val byt = keyBytes[byteIndex].toInt() and 0xFF
        val size = size()
        val pos = bitPos % 8
        if (byteIndex != size - 2) {
            val bitIndex = 7 - pos
            return bitAtIndex(byt, bitIndex)
        }
        val leftPadding = keyBytes[size - 1].toInt() and 0xFF
        if ((leftPadding == 0 && byt == 0) || pos < leftPadding) {
            return 0
        }
        val lastByteBitsCount = ((totalBits() - 1) % 8) + 1
        val shift = lastByteBitsCount - pos - 1
        return (byt ushr shift) and 1
    }

    fun addBit(bit: Int) {
        var bitPos = totalBits() % 8
        var size = size()
        if (bitPos == 0 && bitCount != 0) {
            bitPos = 8
        }
        var lastByte = keyBytes[size - 2].toInt() and 0xFF
        var leftPadding = keyBytes[size - 1].toInt() and 0xFF
        if (bitCount != 0 && lastByte == 0) {
            leftPadding++
        }
        if (bitPos == 8) {
            keyBytes = keyBytes + byteArrayOf(0)
            lastByte = 0
            leftPadding = 0
            length += 1
            size += 1
        }
        bitCount++
        keyBytes[size - 2] = (((lastByte shl 1) or bit) and 0xFF).toByte()
        keyBytes[size - 1] = (leftPadding and 0xFF).toByte()
    }

    fun bytes(): ByteArray = keyBytes

    fun fromBytes(data: ByteArray): Key {
        keyBytes = data
        return this
    }

    fun totalBits(): Int {
        if (bitCount == 0) {
            val size = size()
            if (size == 0) {
                keyBytes = byteArrayOf(0, 0)
                length = 2
                return 0
            }
            val leadingZeroes = keyBytes[size - 1].toInt() and 0xFF
            val fullBits = (size - 2) * 8
            var bitLen = len8(keyBytes[size - 2].toInt())
            if (bitLen == 0) {
                bitLen = 1
            }
            bitCount = fullBits + leadingZeroes + bitLen
        }
        return bitCount
    }

    fun size(): Int {
        if (length == 0) {
            length = keyBytes.size
        }
        return length
    }

    fun bitAtIndex(b: Int, index: Int): Int = (b ushr index) and 1

    fun equalsKey(k2: Key): Boolean = keyBytes.contentEquals(k2.keyBytes)
}

// newNodeKey builds a Key from `data` holding `bitCount` bits, matching store's newNodeKey.
private fun newNodeKey(data: ByteArray, bitCount: Int): Key {
    val numBytes = (bitCount + 7) / 8
    val buf = ByteArray(numBytes)
    System.arraycopy(data, 0, buf, 0, minOf(data.size, numBytes))
    var lastByteBits = bitCount % 8
    if (lastByteBits == 0 && bitCount > 0) {
        lastByteBits = 8
    }
    buf[numBytes - 1] = ((buf[numBytes - 1].toInt() and 0xFF) ushr (8 - lastByteBits)).toByte()
    var leftPadding = leadingZeros8(buf[numBytes - 1].toInt()) - (8 - lastByteBits)
    if ((buf[numBytes - 1].toInt() and 0xFF) == 0) {
        leftPadding--
    }
    val k = Key()
    k.keyBytes = buf + byteArrayOf(leftPadding.toByte())
    k.bitCount = bitCount
    return k
}
