/**
 * verify.ts is a small, self-contained, dependency-free client library for trustlessly verifying the
 * merkle proofs returned by the plugin's custom RPC endpoints (e.g. /v1/query/faucets?address=).
 *
 * It does NOT import or run a Canopy node: it reconstructs the Sparse-Merkle-Tree root from the proof
 * and checks membership / non-membership against a state root the caller has authenticated
 * independently (via the quorum certificate for the height). It is a faithful port of the Go verifier
 * in plugin/go/tutorial/verify (which itself ports Canopy's store/smt.go), using the exact same key
 * encoding and hashing (sha256), so proofs produced by the node verify here without running it.
 *
 * Node leaf keys are sha256(rawKey) and leaf values are sha256(value); internal node values are the
 * hash of (leftKey || leftValue || rightKey || rightValue).
 *
 * Trust model: the ONLY thing you must trust is the state root for the height (authenticated out of
 * band). The value, the proof, and existence are all VERIFIED here against that root — so a lying or
 * buggy RPC node can at most make verification FAIL; it can never forge an accepted answer.
 */

import { createHash } from 'crypto';

// maxKeyBitLength is the leaf key bit length: 20-byte (sha256-truncated) leaf keys.
const maxKeyBitLength = 160;
// leftChild is the 0-bit branch; the right (1-bit) branch is the implicit `else` everywhere below.
const leftChild = 0;

// rootKeyBytes is the arbitrary root key (midpoint between min and max), matching store.RootKey.
const rootKeyBytes: Uint8Array = new Uint8Array([
    0x7f, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255,
    255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255
]);

// Bytes is the shape accepted everywhere a raw byte-string is used (growable key storage uses number[],
// while proof/hash data arrives as Uint8Array/Buffer).
type Bytes = Uint8Array | number[];

// ProofNode is a single sparse-merkle-tree proof node (the leaf being proven, followed by the sibling
// nodes along the path to the root). It mirrors the `proof` array shape returned by the plugin RPC.
export interface ProofNode {
    key: Uint8Array;
    value: Uint8Array;
    bitmask: number;
}

// hash() is the global hashing algorithm (sha256), matching lib/crypto.Hash.
function hash(msg: Bytes): Buffer {
    return createHash('sha256').update(Buffer.from(msg)).digest();
}

// concatBytes() returns a freshly-allocated concatenation of the parts.
function concatBytes(...parts: Bytes[]): Buffer {
    return Buffer.concat(parts.map((p) => Buffer.from(p)));
}

// bytesEqual() compares two byte-strings for exact equality.
function bytesEqual(a: Bytes, b: Bytes): boolean {
    if (a.length !== b.length) {
        return false;
    }
    for (let i = 0; i < a.length; i++) {
        if (a[i] !== b[i]) {
            return false;
        }
    }
    return true;
}

// keyString() maps a key's raw bytes to a stable map-key string (mirrors Go's string([]byte)).
function keyString(b: Bytes): string {
    return Buffer.from(b).toString('hex');
}

// len8() returns the minimum number of bits required to represent an 8-bit value (bits.Len8).
function len8(x: number): number {
    let n = 0;
    let v = x & 0xff;
    while (v > 0) {
        n++;
        v >>= 1;
    }
    return n;
}

// leadingZeros8() returns the number of leading zero bits in an 8-bit value (bits.LeadingZeros8).
function leadingZeros8(x: number): number {
    return 8 - len8(x);
}

// ============================ key ============================
//
// Key is a byte representation of a left-to-right bit-string where the final byte stores the number of
// left-padding zeroes in the preceding data byte. Faithful port of store's key type.
class Key {
    key: number[] = [];
    bitCount = 0;

    greatestCommonPrefix(bitPos: number, gcp: Key, current: Key): number {
        const totalBits = current.totalBits();
        for (; bitPos < totalBits; bitPos++) {
            const bit1 = this.bitAt(bitPos);
            const bit2 = current.bitAt(bitPos);
            if (bit1 !== bit2) {
                break;
            }
            gcp.addBit(bit1);
        }
        return bitPos;
    }

    bitAt(bitPos: number): number {
        const byteIndex = Math.floor(bitPos / 8);
        const byt = this.key[byteIndex];
        const size = this.size();
        const pos = bitPos % 8;
        if (byteIndex !== size - 2) {
            const bitIndex = 7 - pos;
            return this.bitAtIndex(byt, bitIndex);
        }
        const leftPadding = this.key[size - 1];
        if ((leftPadding === 0 && byt === 0) || pos < leftPadding) {
            return 0;
        }
        const lastByteBitsCount = ((this.totalBits() - 1) % 8) + 1;
        const shift = lastByteBitsCount - pos - 1;
        return (byt >> shift) & 1;
    }

    addBit(bit: number): void {
        let bitPos = this.totalBits() % 8;
        let size = this.size();
        if (bitPos === 0 && this.bitCount !== 0) {
            bitPos = 8;
        }
        let lastByte = this.key[size - 2];
        let leftPadding = this.key[size - 1];
        if (this.bitCount !== 0 && lastByte === 0) {
            leftPadding++;
        }
        if (bitPos === 8) {
            this.key.push(0);
            lastByte = 0;
            leftPadding = 0;
            size = size + 1;
        }
        this.bitCount++;
        this.key[size - 2] = ((lastByte << 1) | bit) & 0xff;
        this.key[size - 1] = leftPadding & 0xff;
    }

    bytes(): number[] {
        return this.key;
    }

    fromBytes(data: Bytes): Key {
        this.key = Array.from(data);
        return this;
    }

    totalBits(): number {
        if (this.bitCount === 0) {
            const size = this.size();
            if (size === 0) {
                this.key = [0, 0];
                return 0;
            }
            const leadingZeroes = this.key[size - 1];
            const fullBits = (size - 2) * 8;
            let bitLen = len8(this.key[size - 2]);
            if (bitLen === 0) {
                bitLen = 1;
            }
            this.bitCount = fullBits + leadingZeroes + bitLen;
        }
        return this.bitCount;
    }

    size(): number {
        return this.key.length;
    }

    bitAtIndex(b: number, index: number): number {
        return (b >> index) & 1;
    }

    equals(k2: Key): boolean {
        return bytesEqual(this.key, k2.key);
    }
}

// newNodeKey() builds a Key from raw data truncated/encoded to bitCount bits (mirrors store.newNodeKey).
function newNodeKey(data: Bytes, bitCount: number): Key {
    const numBytes = Math.floor((bitCount + 7) / 8);
    const keyBytes: number[] = new Array<number>(numBytes).fill(0);
    for (let i = 0; i < numBytes && i < data.length; i++) {
        keyBytes[i] = data[i];
    }
    let lastByteBits = bitCount % 8;
    if (lastByteBits === 0 && bitCount > 0) {
        lastByteBits = 8;
    }
    keyBytes[numBytes - 1] = keyBytes[numBytes - 1] >> (8 - lastByteBits);
    let leftPadding = leadingZeros8(keyBytes[numBytes - 1]) - (8 - lastByteBits);
    if (keyBytes[numBytes - 1] === 0) {
        leftPadding--;
    }
    keyBytes.push(leftPadding & 0xff);
    const k = new Key();
    k.key = keyBytes;
    k.bitCount = bitCount;
    return k;
}

// repeatByte() returns a number[] of length n filled with b (mirrors bytes.Repeat).
function repeatByte(b: number, n: number): number[] {
    return new Array<number>(n).fill(b);
}

// ============================ node ============================

// SMTNode mirrors the fields of store's node used during verification.
interface SMTNode {
    key: Key;
    value?: Bytes;
    leftChildKey?: Bytes;
    rightChildKey?: Bytes;
}

function newNode(): SMTNode {
    return { key: new Key() };
}

function copyNode(x: SMTNode): SMTNode {
    return {
        key: x.key,
        value: x.value,
        leftChildKey: x.leftChildKey,
        rightChildKey: x.rightChildKey
    };
}

// ============================ smt (verification-only) ============================

class SMT {
    store: Map<string, SMTNode>;
    keyBitLength: number;
    root: SMTNode;
    minKey: Key;
    maxKey: Key;
    // traversal state
    current: SMTNode;
    target: SMTNode;
    traversed: SMTNode[];
    gcp: Key;
    bitPos: number;
    pathBit: number;

    constructor(rootKey: Bytes, keyBitLen: number, store: Map<string, SMTNode>) {
        this.store = store;
        this.keyBitLength = keyBitLen;
        this.minKey = newNodeKey(repeatByte(0, 20), keyBitLen);
        this.maxKey = newNodeKey(repeatByte(255, 20), keyBitLen);
        const rKey = newNodeKey(Array.from(rootKey), keyBitLen);
        this.root = this.getNode(rKey.bytes());
        if (this.root.leftChildKey == null) {
            this.initializeTree(rKey);
        }
        // fields are initialized by reset(); seed them first to satisfy strict property init
        this.current = this.root;
        this.target = this.root;
        this.traversed = [];
        this.gcp = new Key();
        this.bitPos = 0;
        this.pathBit = 0;
        this.reset();
    }

    initializeTree(rootKey: Key): void {
        const minNode: SMTNode = {
            key: newNodeKey(repeatByte(0, 20), this.keyBitLength),
            value: repeatByte(0, 20)
        };
        const maxNode: SMTNode = {
            key: newNodeKey(repeatByte(255, 20), this.keyBitLength),
            value: repeatByte(255, 20)
        };
        this.setNode(minNode);
        this.setNode(maxNode);
        this.root = {
            key: rootKey,
            leftChildKey: minNode.key.bytes(),
            rightChildKey: maxNode.key.bytes()
        };
        this.updateParentValue(this.root);
        this.setNode(this.root);
    }

    updateParentValue(parent: SMTNode): void {
        const lChild = this.getNode(parent.leftChildKey as Bytes);
        const rChild = this.getNode(parent.rightChildKey as Bytes);
        const input = concatBytes(
            lChild.key.bytes(),
            lChild.value as Bytes,
            rChild.key.bytes(),
            rChild.value as Bytes
        );
        parent.value = hash(input);
        if (bytesEqual(parent.key.bytes(), this.root.key.bytes())) {
            this.root = copyNode(parent);
        }
    }

    getNode(k: Bytes): SMTNode {
        const n = this.store.get(keyString(k));
        if (n != null) {
            return n;
        }
        return newNode();
    }

    setNode(n: SMTNode): void {
        this.store.set(keyString(n.key.bytes()), n);
    }

    reset(): void {
        this.current = copyNode(this.root);
        this.traversed = [];
        this.resetGCP();
    }

    resetGCP(): void {
        this.gcp = new Key();
        this.pathBit = 0;
        this.bitPos = 0;
    }

    // traverse() navigates the tree downward to locate the target or its closest position.
    traverse(): void {
        for (;;) {
            this.traversed.push(copyNode(this.current));
            this.pathBit = this.target.key.bitAt(this.bitPos);
            let currentKey: Bytes;
            if (this.pathBit === leftChild) {
                currentKey = this.current.leftChildKey as Bytes;
            } else {
                currentKey = this.current.rightChildKey as Bytes;
            }
            this.current = this.getNode(currentKey);
            this.current.key.fromBytes(currentKey);
            this.bitPos = this.target.key.greatestCommonPrefix(
                this.bitPos,
                this.gcp,
                this.current.key
            );
            if (!this.current.key.equals(this.gcp) || this.target.key.equals(this.gcp)) {
                return;
            }
        }
    }

    // validateTarget() rejects reserved keys (root, min, max).
    validateTarget(n: SMTNode): boolean {
        if (bytesEqual(this.root.key.bytes(), n.key.bytes())) {
            return false;
        }
        if (bytesEqual(this.minKey.bytes(), n.key.bytes())) {
            return false;
        }
        if (bytesEqual(this.maxKey.bytes(), n.key.bytes())) {
            return false;
        }
        return true;
    }
}

// verifyProof() reconstructs the root hash from the proof and compares it against the provided root,
// then confirms membership (value matches) or non-membership (target absent). Faithful port of
// store.SMT.VerifyProof. Any malformed/adversarial input is rejected (returns false) rather than
// throwing — the whole body runs inside verifyKeyProof's try/catch guard.
function verifyProof(
    k: Uint8Array,
    v: Uint8Array,
    validateMembership: boolean,
    root: Uint8Array,
    proof: ProofNode[],
    keyBitLength: number
): boolean {
    const proofLen = proof.length;
    if (proofLen < 2) {
        return false;
    }
    let h: Bytes = proof[0].value;
    let currentKey = new Key().fromBytes(proof[0].key);
    const memStore = new Map<string, SMTNode>();
    const s = new SMT(rootKeyBytes, keyBitLength, memStore);
    // set the node being proven (leaf or potential location)
    s.setNode({ value: proof[0].value, key: currentKey });
    // reconstruct the tree bottom-up using the proof siblings
    for (let i = 1; i < proofLen; i++) {
        let parentNode: SMTNode;
        if (proof[i].bitmask === leftChild) {
            h = hash(concatBytes(proof[i].key, proof[i].value, currentKey.bytes(), h));
            parentNode = {
                key: new Key(),
                leftChildKey: proof[i].key,
                rightChildKey: currentKey.bytes()
            };
        } else {
            h = hash(concatBytes(currentKey.bytes(), h, proof[i].key, proof[i].value));
            parentNode = {
                key: new Key(),
                leftChildKey: currentKey.bytes(),
                rightChildKey: proof[i].key
            };
        }
        const nodeKey = new Key().fromBytes(proof[i].key);
        const gcp = new Key();
        // NOTE: faithfully mirrors store.VerifyProof (the first comparison is a no-op there too).
        nodeKey.greatestCommonPrefix(0, gcp, currentKey);
        currentKey = gcp;
        parentNode.value = h;
        parentNode.key = currentKey;
        s.setNode(parentNode);
        if (i === proofLen - 1) {
            s.root = parentNode;
        }
    }
    // compare the reconstructed root against the provided (trusted) root
    if (!bytesEqual(h, root)) {
        return false;
    }
    // traverse the reconstructed tree to confirm presence/absence of the target
    s.target = { key: newNodeKey(hash(k), s.keyBitLength) };
    if (!s.validateTarget(s.target)) {
        return false;
    }
    s.reset();
    s.traverse();
    const nodeExists = s.target.key.equals(s.gcp);
    if ((!nodeExists && validateMembership) || (nodeExists && !validateMembership)) {
        return false;
    }
    if (!nodeExists && !validateMembership) {
        return true;
    }
    // membership: the proven value must hash to the leaf value
    return bytesEqual(proof[0].value, hash(v));
}

/**
 * verifyKeyProof verifies a membership (exists=true) or non-membership (exists=false) proof for the
 * given raw state key and committed value against a trusted state root. It returns true when the proof
 * is valid for the claimed existence, and false for ANY mismatch: a tampered value, a tampered proof
 * node, a wrong/forged root, or a malformed proof. It never throws on adversarial input.
 *
 *   - rawKey: the raw (length-prefixed) state key, e.g. KeyForFaucet(addr). The verifier hashes it
 *     itself (leaf keys are sha256(rawKey)); a trustless caller should recompute it rather than trust
 *     the node's echoed key.
 *   - value: the raw committed bytes the proof commits to (ignored when exists=false).
 *   - exists: whether this should be a membership (true) or non-membership (false) proof.
 *   - root: the trusted state root the proof must reconstruct to.
 *   - proof: the proof nodes from the RPC response.
 */
export function verifyKeyProof(
    rawKey: Uint8Array,
    value: Uint8Array,
    exists: boolean,
    root: Uint8Array,
    proof: ProofNode[]
): boolean {
    // The proof is fully attacker-controlled: a malicious server can send malformed keys/bitmasks that
    // drive the faithful SMT port into out-of-range indexing. Treat any such throw as a failed proof
    // rather than crashing the client (defense against a client-side DoS).
    try {
        return verifyProof(rawKey, value, exists, root, proof, maxKeyBitLength);
    } catch {
        return false;
    }
}
