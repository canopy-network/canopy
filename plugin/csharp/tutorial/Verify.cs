// Verify.cs is a small, self-contained, dependency-free client library for trustlessly verifying the
// merkle proofs returned by the plugin's custom RPC endpoints (e.g. /v1/query/faucets?address=).
//
// It does NOT import or run a Canopy node: it reconstructs the sparse-merkle-tree (SMT) root from the
// proof and checks membership / non-membership against a state root the caller has authenticated
// independently (via the quorum certificate for the height).
//
// It is a faithful C# port of the Go reference verifier (plugin/go/tutorial/verify/{smt.go,verify.go})
// using the exact same key encoding and hashing (sha256), so proofs produced by the node verify here
// without importing or running it: leaf keys are sha256(rawKey), leaf values are sha256(value), and an
// internal node value is hash(leftKey || leftValue || rightKey || rightValue).
//
// Trust model: the ONLY thing you must trust is the state root for the height (authenticated out of
// band). The value, the proof, and existence are all VERIFIED here against that root — so a lying or
// buggy RPC node can at most make verification FAIL; it can never forge an accepted answer.

#nullable disable

using System;
using System.Collections.Generic;
using System.Security.Cryptography;

namespace CanopyPlugin.Tutorial
{
    /// <summary>
    /// ProofNode is a single sparse-merkle-tree proof node (the leaf being proven, followed by the
    /// sibling nodes along the path to the root). It mirrors the <c>proof</c> array shape returned by
    /// the plugin RPC.
    /// </summary>
    public sealed class ProofNode
    {
        public byte[] Key { get; set; }
        public byte[] Value { get; set; }
        public int Bitmask { get; set; }

        public ProofNode(byte[] key, byte[] value, int bitmask)
        {
            Key = key;
            Value = value;
            Bitmask = bitmask;
        }
    }

    /// <summary>
    /// Verifier exposes the public merkle-proof verification API.
    /// </summary>
    public static class Verifier
    {
        private const int MaxKeyBitLength = 160; // 20-byte (sha256-truncated) leaf keys
        private const int LeftChild = 0;
        private const int RightChild = 1;

        // RootKeyBytes is the arbitrary root key (midpoint between min and max), matching store.RootKey.
        private static readonly byte[] RootKeyBytes =
        {
            0x7F, 255, 255, 255, 255, 255, 255, 255,
            255, 255, 255, 255, 255, 255, 255, 255,
            255, 255, 255, 255, 255, 255, 255, 255,
            255, 255, 255, 255, 255, 255, 255, 255,
        };

        /// <summary>
        /// VerifyKeyProof verifies a membership (exists=true) or non-membership (exists=false) proof for
        /// the given raw state key and committed value against a trusted state root. It returns true when
        /// the proof is valid for the claimed existence, and false for ANY mismatch: a tampered value, a
        /// tampered proof node, a wrong/forged root, or a malformed proof. It never throws on adversarial
        /// input.
        /// </summary>
        /// <param name="rawKey">The raw (length-prefixed) state key, e.g. KeyForFaucet(addr). The verifier
        /// hashes it itself (leaf keys are sha256(rawKey)); a trustless caller should recompute it rather
        /// than trust the node's echoed key.</param>
        /// <param name="value">The raw committed bytes the proof commits to (ignored when exists=false).</param>
        /// <param name="exists">Whether this should be a membership (true) or non-membership (false) proof.</param>
        /// <param name="root">The trusted state root the proof must reconstruct to.</param>
        /// <param name="proof">The proof nodes from the RPC response.</param>
        public static bool VerifyKeyProof(byte[] rawKey, byte[] value, bool exists, byte[] root, IReadOnlyList<ProofNode> proof)
        {
            return VerifyProof(rawKey, value, exists, root, proof, MaxKeyBitLength);
        }

        // ============================ hashing / byte helpers ============================

        // hash() is the global hashing algorithm (sha256), matching lib/crypto.Hash.
        private static byte[] Hash(byte[] msg) => SHA256.HashData(msg);

        // concat() returns a freshly-allocated concatenation of the parts.
        private static byte[] Concat(params byte[][] parts)
        {
            int total = 0;
            foreach (var p in parts) total += p.Length;
            var outb = new byte[total];
            int offset = 0;
            foreach (var p in parts)
            {
                Array.Copy(p, 0, outb, offset, p.Length);
                offset += p.Length;
            }
            return outb;
        }

        private static byte[] Repeat(byte b, int n)
        {
            var a = new byte[n];
            if (b != 0)
            {
                for (int i = 0; i < n; i++) a[i] = b;
            }
            return a;
        }

        private static bool BytesEqual(byte[] a, byte[] b) => a.AsSpan().SequenceEqual(b);

        // bits.LeadingZeros8 equivalent: number of leading zero bits in an 8-bit value (8 for zero).
        private static int LeadingZeros8(byte x)
        {
            if (x == 0) return 8;
            int n = 0;
            for (int i = 7; i >= 0; i--)
            {
                if ((x & (1 << i)) != 0) break;
                n++;
            }
            return n;
        }

        // bits.Len8 equivalent: minimum number of bits to represent an 8-bit value (0 for zero).
        private static int Len8(byte x)
        {
            int n = 0;
            while (x != 0)
            {
                n++;
                x >>= 1;
            }
            return n;
        }

        private static byte[] AppendByte(byte[] arr, byte b)
        {
            var res = new byte[arr.Length + 1];
            Array.Copy(arr, res, arr.Length);
            res[arr.Length] = b;
            return res;
        }

        // ============================ verifyProof ============================

        // verifyProof() reconstructs the root hash from the proof and compares it against the provided
        // root, then confirms membership (value matches) or non-membership (target absent). Faithful port
        // of store.SMT.VerifyProof.
        private static bool VerifyProof(byte[] k, byte[] v, bool validateMembership, byte[] root, IReadOnlyList<ProofNode> proof, int keyBitLength)
        {
            // The proof is fully attacker-controlled: a malicious server can send malformed keys/bitmasks
            // that drive the faithful SMT port into out-of-range indexing / null access. Treat any such
            // exception as a failed proof rather than crashing the client (defense against a client-side DoS).
            try
            {
                int proofLen = proof.Count;
                if (proofLen < 2)
                {
                    return false;
                }
                byte[] h = proof[0].Value;
                var currentKey = new Key().FromBytes(proof[0].Key);
                var memStore = new Dictionary<string, Node>();
                var s = Smt.NewSmt(RootKeyBytes, keyBitLength, memStore);
                // set the node being proven (leaf or potential location)
                s.SetNode(new Node { Value = proof[0].Value, Key = currentKey });
                // reconstruct the tree bottom-up using the proof siblings
                for (int i = 1; i < proofLen; i++)
                {
                    Node parentNode;
                    if (proof[i].Bitmask == LeftChild)
                    {
                        h = Hash(Concat(proof[i].Key, proof[i].Value, currentKey.Bytes(), h));
                        parentNode = new Node { LeftChildKey = proof[i].Key, RightChildKey = currentKey.Bytes() };
                    }
                    else
                    {
                        h = Hash(Concat(currentKey.Bytes(), h, proof[i].Key, proof[i].Value));
                        parentNode = new Node { LeftChildKey = currentKey.Bytes(), RightChildKey = proof[i].Key };
                    }
                    var nodeKey = new Key().FromBytes(proof[i].Key);
                    var gcp = new Key();
                    int bitPos = 0;
                    // NOTE: faithfully mirrors store.VerifyProof (the first comparison is a no-op there too).
                    if (currentKey.TotalBits() < currentKey.TotalBits())
                    {
                        currentKey.GreatestCommonPrefix(ref bitPos, gcp, nodeKey);
                    }
                    else
                    {
                        nodeKey.GreatestCommonPrefix(ref bitPos, gcp, currentKey);
                    }
                    currentKey = gcp;
                    parentNode.Value = h;
                    parentNode.Key = currentKey;
                    s.SetNode(parentNode);
                    if (i == proofLen - 1)
                    {
                        s.Root = parentNode;
                    }
                }
                // compare the reconstructed root against the provided (trusted) root
                if (!BytesEqual(h, root))
                {
                    return false;
                }
                // traverse the reconstructed tree to confirm presence/absence of the target
                s.Target = new Node { Key = Key.NewNodeKey(Hash(k), s.KeyBitLength) };
                if (s.IsReservedTarget(s.Target))
                {
                    return false;
                }
                s.Reset();
                s.Traverse();
                bool nodeExists = s.Target.Key.Equals(s.Gcp);
                if ((!nodeExists && validateMembership) || (nodeExists && !validateMembership))
                {
                    return false;
                }
                if (!nodeExists && !validateMembership)
                {
                    return true;
                }
                // membership: the proven value must hash to the leaf value
                return BytesEqual(proof[0].Value, Hash(v));
            }
            catch
            {
                return false;
            }
        }

        // ============================ node ============================

        // Node mirrors the fields of store's node used during verification. It is a reference type so
        // that copy() shares the Key reference (matching Go's shallow struct copy semantics).
        private sealed class Node
        {
            public Key Key;
            public byte[] Value;
            public byte[] LeftChildKey;
            public byte[] RightChildKey;

            public Node Copy() => (Node)MemberwiseClone();
        }

        // ============================ smt (verification-only) ============================

        private sealed class Smt
        {
            public Dictionary<string, Node> Store;
            public int KeyBitLength;
            public Node Root;
            public Key MinKey;
            public Key MaxKey;

            // traversal state
            public Node Current;
            public Node Target;
            public List<Node> Traversed;
            public Key Gcp;
            public int BitPos;
            public int PathBit;

            private static string KeyStr(byte[] k) => k == null ? string.Empty : Convert.ToHexString(k);

            private static Node NewNode() => new Node { Key = new Key() };

            // NewSmt() creates a verification SMT over an in-memory node map, initializing an empty tree
            // with the reserved min/max border nodes (mirrors store.NewSMT).
            public static Smt NewSmt(byte[] rootKey, int keyBitLen, Dictionary<string, Node> store)
            {
                var s = new Smt
                {
                    Store = store,
                    KeyBitLength = keyBitLen,
                    MinKey = Key.NewNodeKey(Repeat(0, 20), keyBitLen),
                    MaxKey = Key.NewNodeKey(Repeat(255, 20), keyBitLen),
                };
                var rKey = Key.NewNodeKey((byte[])rootKey.Clone(), keyBitLen);
                s.Root = s.GetNode(rKey.Bytes());
                if (s.Root.LeftChildKey == null)
                {
                    s.InitializeTree(rKey);
                }
                s.Reset();
                return s;
            }

            // InitializeTree() ensures the tree always has a root with two border children.
            private void InitializeTree(Key rootKey)
            {
                var minNode = new Node { Key = Key.NewNodeKey(Repeat(0, 20), KeyBitLength), Value = Repeat(0, 20) };
                var maxNode = new Node { Key = Key.NewNodeKey(Repeat(255, 20), KeyBitLength), Value = Repeat(255, 20) };
                SetNode(minNode);
                SetNode(maxNode);
                Root = new Node
                {
                    Key = rootKey,
                    LeftChildKey = minNode.Key.Bytes(),
                    RightChildKey = maxNode.Key.Bytes(),
                };
                UpdateParentValue(Root);
                SetNode(Root);
            }

            // UpdateParentValue() recomputes a parent's value from its children (left/right key+value).
            private void UpdateParentValue(Node parent)
            {
                var lChild = GetNode(parent.LeftChildKey);
                var rChild = GetNode(parent.RightChildKey);
                var input = new byte[lChild.Key.Size() + lChild.Value.Length + rChild.Key.Size() + rChild.Value.Length];
                int offset = 0;
                offset += CopyInto(input, offset, lChild.Key.Bytes());
                offset += CopyInto(input, offset, lChild.Value);
                offset += CopyInto(input, offset, rChild.Key.Bytes());
                CopyInto(input, offset, rChild.Value);
                parent.Value = Hash(input);
                if (BytesEqual(parent.Key.Bytes(), Root.Key.Bytes()))
                {
                    Root = parent.Copy();
                }
            }

            private static int CopyInto(byte[] dst, int offset, byte[] src)
            {
                int n = Math.Min(dst.Length - offset, src.Length);
                Array.Copy(src, 0, dst, offset, n);
                return n;
            }

            public Node GetNode(byte[] k)
            {
                if (k != null && Store.TryGetValue(KeyStr(k), out var n) && n != null)
                {
                    return n;
                }
                return NewNode();
            }

            public void SetNode(Node n) => Store[KeyStr(n.Key.Bytes())] = n;

            public void Reset()
            {
                Current = Root.Copy();
                Traversed = new List<Node>();
                ResetGcp();
            }

            private void ResetGcp()
            {
                Gcp = new Key();
                PathBit = 0;
                BitPos = 0;
            }

            // Traverse() navigates the tree downward to locate the target or its closest position.
            public void Traverse()
            {
                while (true)
                {
                    byte[] currentKey = null;
                    Traversed.Add(Current.Copy());
                    PathBit = Target.Key.BitAt(BitPos);
                    switch (PathBit)
                    {
                        case LeftChild:
                            currentKey = Current.LeftChildKey;
                            break;
                        case RightChild:
                            currentKey = Current.RightChildKey;
                            break;
                    }
                    Current = GetNode(currentKey);
                    Current.Key.FromBytes(currentKey);
                    Target.Key.GreatestCommonPrefix(ref BitPos, Gcp, Current.Key);
                    if (!Current.Key.Equals(Gcp) || Target.Key.Equals(Gcp))
                    {
                        return;
                    }
                }
            }

            // IsReservedTarget() reports whether the target is a reserved key (root, min, max).
            public bool IsReservedTarget(Node n)
            {
                var nb = n.Key.Bytes();
                if (BytesEqual(Root.Key.Bytes(), nb)) return true;
                if (BytesEqual(MinKey.Bytes(), nb)) return true;
                if (BytesEqual(MaxKey.Bytes(), nb)) return true;
                return false;
            }
        }

        // ============================ key ============================
        //
        // Key is a byte[] representation of a left-to-right bit-string where the final byte stores the
        // number of left-padding zeroes in the preceding data byte. Faithful port of store's key type.
        private sealed class Key
        {
            private byte[] _key;
            private int _bitCount;
            private int _length;

            public Key()
            {
                _key = Array.Empty<byte>();
            }

            public static Key NewNodeKey(byte[] data, int bitCount)
            {
                int numBytes = (bitCount + 7) / 8;
                var key = new byte[numBytes];
                Array.Copy(data, 0, key, 0, Math.Min(numBytes, data.Length));
                int lastByteBits = bitCount % 8;
                if (lastByteBits == 0 && bitCount > 0)
                {
                    lastByteBits = 8;
                }
                key[numBytes - 1] >>= (8 - lastByteBits);
                int leftPadding = LeadingZeros8(key[numBytes - 1]) - (8 - lastByteBits);
                if (key[numBytes - 1] == 0)
                {
                    leftPadding--;
                }
                var k = new Key
                {
                    _key = AppendByte(key, (byte)leftPadding),
                    _bitCount = bitCount,
                };
                return k;
            }

            public void GreatestCommonPrefix(ref int bitPos, Key gcp, Key current)
            {
                int totalBits = current.TotalBits();
                for (; bitPos < totalBits; bitPos++)
                {
                    int bit1 = BitAt(bitPos);
                    int bit2 = current.BitAt(bitPos);
                    if (bit1 != bit2)
                    {
                        break;
                    }
                    gcp.AddBit(bit1);
                }
            }

            public int BitAt(int bitPos)
            {
                int byteIndex = bitPos / 8;
                byte byt = _key[byteIndex];
                int size = Size();
                int pos = bitPos % 8;
                if (byteIndex != size - 2)
                {
                    int bitIndex = 7 - pos;
                    return BitAtIndex(byt, bitIndex);
                }
                int leftPadding = _key[size - 1];
                if ((leftPadding == 0 && byt == 0) || pos < leftPadding)
                {
                    return 0;
                }
                int lastByteBitsCount = ((TotalBits() - 1) % 8) + 1;
                int shift = lastByteBitsCount - pos - 1;
                return (byt >> shift) & 1;
            }

            public void AddBit(int bit)
            {
                int bitPos = TotalBits() % 8;
                int size = Size();
                if (bitPos == 0 && _bitCount != 0)
                {
                    bitPos = 8;
                }
                byte lastByte = _key[size - 2];
                int leftPadding = _key[size - 1];
                if (_bitCount != 0 && lastByte == 0)
                {
                    leftPadding++;
                }
                if (bitPos == 8)
                {
                    _key = AppendByte(_key, 0);
                    lastByte = 0;
                    bitPos = 0;
                    leftPadding = 0;
                    _length = _length + 1;
                    size = size + 1;
                }
                _bitCount++;
                _key[size - 2] = (byte)((lastByte << 1) | bit);
                _key[size - 1] = (byte)leftPadding;
            }

            public byte[] Bytes() => _key;

            public Key FromBytes(byte[] data)
            {
                _key = data;
                return this;
            }

            public int TotalBits()
            {
                if (_bitCount == 0)
                {
                    int size = Size();
                    if (size == 0)
                    {
                        _key = new byte[] { 0, 0 };
                        _length = 2;
                        return 0;
                    }
                    int leadingZeroes = _key[size - 1];
                    int fullBits = (size - 2) * 8;
                    int bitLen = Len8(_key[size - 2]);
                    if (bitLen == 0)
                    {
                        bitLen = 1;
                    }
                    _bitCount = fullBits + leadingZeroes + bitLen;
                }
                return _bitCount;
            }

            public int Size()
            {
                if (_length == 0)
                {
                    _length = _key.Length;
                }
                return _length;
            }

            private static int BitAtIndex(byte b, int index) => (b >> index) & 1;

            public bool Equals(Key k2) => BytesEqual(_key, k2._key);
        }
    }
}
