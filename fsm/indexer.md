Indexer Blob

Overview
This package includes helpers for producing an "indexer blob" that bundles the
current and previous block's state data into a single protobuf response. The
blob is intended for indexers that want to hydrate state with one request.

Height Semantics
- The `height` parameter is the state version (the same value returned by
  `/v1/query/height`).
- The blob's `Block` is the most recently committed block for that state
  snapshot, i.e. `block_height = height - 1`.
- Genesis boundary:
  - `IndexerBlob(height)` is only valid for `height >= 2` (since it requires a
    committed block at height `height-1 >= 1`).
  - `IndexerBlobs(height)` returns `Previous=nil` for `height <= 2`.

What's inside
- Block bytes (protobuf)
- Raw state bytes for accounts, pools, validators, non-signers, dex batches
- Pre-marshaled structures for dex prices, orders, params, and double signers
- Committee data, subsidized committees, retired committees
- Supply bytes

Usage
- fsm.StateMachine.IndexerBlob(height) returns the protobuf-ready structure
  for a specific height.
- fsm.StateMachine.IndexerBlobs(height) returns current and previous blobs.
