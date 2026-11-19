"""
Self-contained, dependency-free client library for trustlessly verifying the merkle proofs returned
by the plugin's custom RPC endpoints (e.g. /v1/query/faucets?address=).

It does NOT import or run a Canopy node: it reconstructs the sparse-merkle-tree (SMT) root from the
proof and checks membership / non-membership against a state root the caller has authenticated
independently (via the quorum certificate for the height).

This is a faithful port of Canopy's optimized SMT proof verification (store/smt.go) and the Go
tutorial's `verify` package (plugin/go/tutorial/verify). It reconstructs the tree from a proof using
the exact same key encoding and hashing (sha256) as the node, so proofs produced by the node verify
here without importing or running it.

Node leaf keys are sha256(rawKey) and leaf values are sha256(value); internal node values are the
hash of (leftKey || leftValue || rightKey || rightValue).

Trust model: the ONLY thing you must trust is the state root for the height (authenticated out of
band). The value, the proof, and existence are all VERIFIED here against that root — so a lying or
buggy RPC node can at most make verification FAIL; it can never forge an accepted answer.

Uses only the Python standard library (hashlib).
"""

from __future__ import annotations

import hashlib
from dataclasses import dataclass, field
from typing import Dict, List, Optional

# maxKeyBitLength is the bit length of 20-byte (sha256-truncated) leaf keys.
MAX_KEY_BIT_LENGTH: int = 160
LEFT_CHILD: int = 0
RIGHT_CHILD: int = 1

# rootKeyBytes is the arbitrary root key (midpoint between min and max), matching store.RootKey.
_ROOT_KEY_BYTES: bytes = bytes([0x7F]) + bytes([0xFF] * 31)


class _InvalidProofError(Exception):
    """Raised internally for a structurally invalid proof; surfaced as verification failure."""


@dataclass
class ProofNode:
    """A single sparse-merkle-tree proof node (the leaf being proven, followed by the sibling nodes
    along the path to the root). Mirrors the `proof` array shape returned by the plugin RPC."""

    key: bytes
    value: bytes
    bitmask: int


def _hash(msg: bytes) -> bytes:
    """The global hashing algorithm (sha256), matching lib/crypto.Hash."""
    return hashlib.sha256(msg).digest()


def _leading_zeros_8(b: int) -> int:
    """Number of leading zero bits in an 8-bit value (bits.LeadingZeros8)."""
    return 8 - (b & 0xFF).bit_length()


def _len_8(b: int) -> int:
    """Minimum number of bits to represent an 8-bit value (bits.Len8)."""
    return (b & 0xFF).bit_length()


# ============================ key ============================
#
# Key is a byte representation of a left-to-right bit-string where the final byte stores the number
# of left-padding zeroes in the preceding data byte. Faithful port of store's `key` type.


class _Key:
    __slots__ = ("key", "bit_count", "length")

    def __init__(self) -> None:
        self.key: bytearray = bytearray()
        self.bit_count: int = 0
        self.length: int = 0

    @classmethod
    def new_node_key(cls, data: bytes, bit_count: int) -> "_Key":
        k = cls()
        num_bytes = (bit_count + 7) // 8
        buf = bytearray(num_bytes)
        n = min(len(data), num_bytes)
        buf[:n] = data[:n]
        last_byte_bits = bit_count % 8
        if last_byte_bits == 0 and bit_count > 0:
            last_byte_bits = 8
        buf[num_bytes - 1] >>= 8 - last_byte_bits
        left_padding = _leading_zeros_8(buf[num_bytes - 1]) - (8 - last_byte_bits)
        if buf[num_bytes - 1] == 0:
            left_padding -= 1
        buf.append(left_padding & 0xFF)
        k.key = buf
        k.bit_count = bit_count
        return k

    def from_bytes(self, data: bytes) -> "_Key":
        # Mirrors Go's fromBytes: replace the backing bytes, leave bit_count/length lazily computed.
        self.key = bytearray(data)
        return self

    def greatest_common_prefix(self, bit_pos: int, gcp: "_Key", current: "_Key") -> int:
        total_bits = current.total_bits()
        while bit_pos < total_bits:
            bit1 = self.bit_at(bit_pos)
            bit2 = current.bit_at(bit_pos)
            if bit1 != bit2:
                break
            gcp.add_bit(bit1)
            bit_pos += 1
        return bit_pos

    def bit_at(self, bit_pos: int) -> int:
        byte_index = bit_pos // 8
        byt = self.key[byte_index]
        size, pos = self.size(), bit_pos % 8
        if byte_index != size - 2:
            bit_index = 7 - pos
            return self._bit_at_index(byt, bit_index)
        left_padding = self.key[size - 1]
        if (left_padding == 0 and byt == 0) or pos < left_padding:
            return 0
        last_byte_bits_count = ((self.total_bits() - 1) % 8) + 1
        shift = last_byte_bits_count - pos - 1
        return (byt >> shift) & 1

    def add_bit(self, bit: int) -> None:
        bit_pos, size = self.total_bits() % 8, self.size()
        if bit_pos == 0 and self.bit_count != 0:
            bit_pos = 8
        last_byte, left_padding = self.key[size - 2], self.key[size - 1]
        if self.bit_count != 0 and last_byte == 0:
            left_padding += 1
        if bit_pos == 8:
            self.key.append(0)
            last_byte, bit_pos, left_padding = 0, 0, 0
            self.length, size = self.length + 1, size + 1
        self.bit_count += 1
        self.key[size - 2] = ((last_byte << 1) | bit) & 0xFF
        self.key[size - 1] = left_padding & 0xFF

    def bytes(self) -> bytes:
        return bytes(self.key)

    def total_bits(self) -> int:
        if self.bit_count == 0:
            size = self.size()
            if size == 0:
                self.key, self.length = bytearray([0, 0]), 2
                return 0
            leading_zeroes = self.key[size - 1]
            data_bytes = self.key[: size - 1]
            full_bits = (size - 2) * 8
            bit_len = _len_8(data_bytes[size - 2])
            if bit_len == 0:
                bit_len = 1
            self.bit_count = full_bits + leading_zeroes + bit_len
        return self.bit_count

    def size(self) -> int:
        if self.length == 0:
            self.length = len(self.key)
        return self.length

    @staticmethod
    def _bit_at_index(b: int, index: int) -> int:
        return (b >> index) & 1

    def equals(self, k2: "_Key") -> bool:
        return bytes(self.key) == bytes(k2.key)


# ============================ node ============================


class _Node:
    """Mirrors the fields of store's node/lib.Node used during verification."""

    __slots__ = ("key", "value", "left_child_key", "right_child_key", "bitmask")

    def __init__(
        self,
        key: Optional[_Key] = None,
        value: bytes = b"",
        left_child_key: Optional[bytes] = None,
        right_child_key: Optional[bytes] = None,
        bitmask: int = 0,
    ) -> None:
        self.key: _Key = key if key is not None else _Key()
        self.value: bytes = value
        self.left_child_key: Optional[bytes] = left_child_key
        self.right_child_key: Optional[bytes] = right_child_key
        self.bitmask: int = bitmask

    def copy(self) -> "_Node":
        # Shallow copy (shares the key reference), matching Go's `c := *x`.
        n = _Node()
        n.key = self.key
        n.value = self.value
        n.left_child_key = self.left_child_key
        n.right_child_key = self.right_child_key
        n.bitmask = self.bitmask
        return n


# ============================ smt (verification-only) ============================


@dataclass
class _NodeList:
    nodes: List[_Node] = field(default_factory=list)


class _SMT:
    def __init__(self, root_key: bytes, key_bit_len: int, store: Dict[bytes, _Node]) -> None:
        self.store: Dict[bytes, _Node] = store
        self.key_bit_length: int = key_bit_len
        self.min_key: _Key = _Key.new_node_key(bytes(20), key_bit_len)
        self.max_key: _Key = _Key.new_node_key(bytes([255] * 20), key_bit_len)
        # traversal state
        self.root: _Node = _Node()
        self.current: _Node = _Node()
        self.target: _Node = _Node()
        self.traversed: _NodeList = _NodeList()
        self.gcp: _Key = _Key()
        self.bit_pos: int = 0
        self.path_bit: int = 0

        r_key = _Key.new_node_key(bytes(root_key), key_bit_len)
        self.root = self.get_node(r_key.bytes())
        if self.root.left_child_key is None:
            self._initialize_tree(r_key)
        self.reset()

    def _initialize_tree(self, root_key: _Key) -> None:
        min_node = _Node(key=_Key.new_node_key(bytes(20), self.key_bit_length), value=bytes(20))
        max_node = _Node(
            key=_Key.new_node_key(bytes([255] * 20), self.key_bit_length),
            value=bytes([255] * 20),
        )
        self.set_node(min_node)
        self.set_node(max_node)
        self.root = _Node(
            key=root_key,
            left_child_key=min_node.key.bytes(),
            right_child_key=max_node.key.bytes(),
        )
        self._update_parent_value(self.root)
        self.set_node(self.root)

    def _update_parent_value(self, parent: _Node) -> None:
        l_child = self.get_node(parent.left_child_key)
        r_child = self.get_node(parent.right_child_key)
        parent.value = _hash(
            l_child.key.bytes() + l_child.value + r_child.key.bytes() + r_child.value
        )
        if parent.key.bytes() == self.root.key.bytes():
            self.root = parent.copy()

    def get_node(self, k: Optional[bytes]) -> _Node:
        if k is not None:
            n = self.store.get(bytes(k))
            if n is not None:
                return n
        return _Node()

    def set_node(self, n: _Node) -> None:
        self.store[n.key.bytes()] = n

    def reset(self) -> None:
        self.current = self.root.copy()
        self.traversed = _NodeList()
        self._reset_gcp()

    def _reset_gcp(self) -> None:
        self.gcp = _Key()
        self.path_bit, self.bit_pos = 0, 0

    def traverse(self) -> None:
        # Termination bound: a well-formed tree is acyclic, so a root-to-leaf path can visit at most
        # as many nodes as exist in the reconstructed store. An attacker-controlled proof can produce
        # a shape that descends forever (e.g. into a non-existent child that normalizes to a reserved
        # border key). The reference Go verifier escapes such input via an out-of-range panic that its
        # caller recovers into `false`; we bound the traversal explicitly and reject instead, so a
        # malformed proof is rejected rather than looping/OOMing the client.
        max_steps = len(self.store) + 1
        steps = 0
        while True:
            steps += 1
            if steps > max_steps:
                raise _InvalidProofError("merkle proof traversal did not terminate")
            self.traversed.nodes.append(self.current.copy())
            self.path_bit = self.target.key.bit_at(self.bit_pos)
            if self.path_bit == LEFT_CHILD:
                current_key = self.current.left_child_key
            else:
                current_key = self.current.right_child_key
            self.current = self.get_node(current_key)
            self.current.key.from_bytes(current_key if current_key is not None else b"")
            self.bit_pos = self.target.key.greatest_common_prefix(
                self.bit_pos, self.gcp, self.current.key
            )
            if not self.current.key.equals(self.gcp) or self.target.key.equals(self.gcp):
                return

    def validate_target(self, n: _Node) -> None:
        if self.root.key.bytes() == n.key.bytes():
            raise _InvalidProofError("reserved key")
        if self.min_key.bytes() == n.key.bytes():
            raise _InvalidProofError("reserved key")
        if self.max_key.bytes() == n.key.bytes():
            raise _InvalidProofError("reserved key")


def _verify_proof(
    k: bytes,
    v: bytes,
    validate_membership: bool,
    root: bytes,
    proof: List[ProofNode],
    key_bit_length: int,
) -> bool:
    # The proof is fully attacker-controlled: a malicious server can send malformed keys/bitmasks
    # that drive the faithful SMT port into out-of-range indexing. Treat any such error as a failed
    # proof rather than crashing the client (defense against a client-side DoS).
    try:
        proof_len = len(proof)
        if proof_len < 2:
            return False
        h = proof[0].value
        current_key = _Key().from_bytes(proof[0].key)
        mem_store: Dict[bytes, _Node] = {}
        s = _SMT(_ROOT_KEY_BYTES, key_bit_length, mem_store)
        # set the node being proven (leaf or potential location)
        s.set_node(_Node(value=proof[0].value, key=current_key))
        # reconstruct the tree bottom-up using the proof siblings
        for i in range(1, proof_len):
            if proof[i].bitmask == LEFT_CHILD:
                h = _hash(proof[i].key + proof[i].value + current_key.bytes() + h)
                parent_node = _Node(
                    left_child_key=proof[i].key, right_child_key=current_key.bytes()
                )
            else:
                h = _hash(current_key.bytes() + h + proof[i].key + proof[i].value)
                parent_node = _Node(
                    left_child_key=current_key.bytes(), right_child_key=proof[i].key
                )
            node_key = _Key().from_bytes(proof[i].key)
            gcp = _Key()
            # NOTE: faithfully mirrors store.VerifyProof (its first comparison is a no-op too, so the
            # `else` branch is always taken).
            node_key.greatest_common_prefix(0, gcp, current_key)
            current_key = gcp
            parent_node.value = h
            parent_node.key = current_key
            s.set_node(parent_node)
            if i == proof_len - 1:
                s.root = parent_node
        # compare the reconstructed root against the provided (trusted) root
        if h != root:
            return False
        # traverse the reconstructed tree to confirm presence/absence of the target
        s.target = _Node(key=_Key.new_node_key(_hash(k), s.key_bit_length))
        s.validate_target(s.target)
        s.reset()
        s.traverse()
        node_exists = s.target.key.equals(s.gcp)
        if (not node_exists and validate_membership) or (node_exists and not validate_membership):
            return False
        if not node_exists and not validate_membership:
            return True
        # membership: the proven value must hash to the leaf value
        return proof[0].value == _hash(v)
    except _InvalidProofError:
        return False
    except Exception:
        # Any out-of-range indexing / malformed-input error is a rejected proof, never a crash.
        return False


def verify_key_proof(
    raw_key: bytes,
    value: bytes,
    exists: bool,
    root: bytes,
    proof: List[ProofNode],
) -> bool:
    """Verify a membership (exists=True) or non-membership (exists=False) proof for the given raw
    state key and committed value against a trusted state root.

    Returns True when the proof is valid for the claimed existence, and False for ANY mismatch: a
    tampered value, a tampered proof node, a wrong/forged root, or a malformed proof. It never raises
    on adversarial input.

    - raw_key: the raw (length-prefixed) state key, e.g. key_for_faucet(addr). The verifier hashes it
      itself (leaf keys are sha256(raw_key)); a trustless caller should recompute it rather than
      trust the node's echoed key.
    - value: the raw committed bytes the proof commits to (ignored when exists=False).
    - exists: whether this should be a membership (True) or non-membership (False) proof.
    - root: the trusted state root the proof must reconstruct to.
    - proof: the proof nodes from the RPC response.
    """
    return _verify_proof(raw_key, value, exists, root, proof, MAX_KEY_BIT_LENGTH)
