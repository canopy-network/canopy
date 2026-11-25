// Package verify is a small, self-contained, dependency-free client library for trustlessly verifying
// the merkle proofs returned by the plugin's custom RPC endpoints (e.g. /v1/query/faucets?address=).
//
// It does NOT import or run a Canopy node: it reconstructs the sparse-merkle-tree root from the proof
// and checks membership / non-membership against a state root the caller has authenticated
// independently (via the quorum certificate for the height). See smt.go for the proof core.
//
// Trust model: the ONLY thing you must trust is the state root for the height (authenticated out of
// band). The value, the proof, and existence are all VERIFIED here against that root — so a lying or
// buggy RPC node can at most make verification FAIL; it can never forge an accepted answer.
package verify

// ProofNode is a single sparse-merkle-tree proof node (the leaf being proven, followed by the sibling
// nodes along the path to the root). It mirrors the `proof` array shape returned by the plugin RPC.
type ProofNode struct {
	Key     []byte
	Value   []byte
	Bitmask int32
}

// VerifyKeyProof verifies a membership (exists=true) or non-membership (exists=false) proof for the
// given raw state key and committed value against a trusted state root. It returns (true, nil) when
// the proof is valid for the claimed existence, and (false, ...) for ANY mismatch: a tampered value,
// a tampered proof node, a wrong/forged root, or a malformed proof. It never panics on adversarial
// input.
//
//   - rawKey: the raw (length-prefixed) state key, e.g. KeyForFaucet(addr). The verifier hashes it
//     itself (leaf keys are sha256(rawKey)); a trustless caller should recompute it rather than trust
//     the node's echoed key.
//   - value: the raw committed bytes the proof commits to (ignored when exists=false).
//   - exists: whether this should be a membership (true) or non-membership (false) proof.
//   - root: the trusted state root the proof must reconstruct to.
//   - proof: the proof nodes from the RPC response.
func VerifyKeyProof(rawKey, value []byte, exists bool, root []byte, proof []ProofNode) (bool, error) {
	return verifyProof(rawKey, value, exists, root, proof, maxKeyBitLength)
}
