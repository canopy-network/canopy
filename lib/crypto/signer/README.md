# Native transaction signer

This package is the intentionally small, offline signing boundary for native
Canopy transfers. It can be imported by custody software or used through the
`signer` executable. Both paths call the same implementation.

The package performs no network or filesystem operations. It constructs the
same protobuf transaction accepted by `POST /v1/tx`, signs the canonical bytes
returned by `Transaction.GetSignBytes`, and immediately verifies the result.

## Security boundary

The online system supplies these non-secret fields:

- recipient;
- amount and fee in integer `uCNPY`;
- recent chain height;
- network ID and chain ID;
- a stable Unix-microsecond timestamp;
- optional memo.

The offline system must display or otherwise authorize those fields before
signing. The `inspect` command independently derives them from a signed
transaction and verifies its signature.

The signer deliberately does not:

- contact an RPC server;
- choose a recipient, amount, or fee;
- decide whether a height is current;
- decide whether a fee satisfies current governance parameters;
- broadcast a transaction;
- determine whether a submitted transaction committed.

Those stateful checks remain with the synchronized node. A native transaction's
`createdHeight` must normally be within the protocol acceptance range when it is
submitted.

## Package use

Pass an existing Canopy `crypto.PrivateKeyI`, such as a key returned by the
encrypted keystore:

```go
request := signer.SendRequest{
    Recipient:     "502c0b3d6ccd1c6f164aa5536b2ba2cb9e80c711",
    AmountUCNPY:   1_000_000,
    FeeUCNPY:      10_000,
    CreatedHeight: 2_044_370,
    NetworkID:     1,
    ChainID:       1,
    Time:          1_786_569_463_230_678,
}

tx, err := signer.SignSend(request, key)
if err != nil {
    return err
}

details, err := signer.Inspect(tx)
```

`SendRequest.Time` is required. Persist the signed transaction and its hash in
the withdrawal record before broadcast; every retry must reuse those exact
signed bytes rather than call `SignSend` again. The memo values `RLP` and
`RLP.V2` are reserved for Ethereum transaction wrappers and are rejected by the
native signer.

## Command-line use

Build the standalone executable:

```sh
go build -o signer ./cmd/signer
```

Prepare `request.json`. All integers are quoted decimal strings to preserve
full `uint64` precision across languages:

```json
{
  "recipient": "502c0b3d6ccd1c6f164aa5536b2ba2cb9e80c711",
  "amountUCNPY": "1000000",
  "feeUCNPY": "10000",
  "createdHeight": "2044370",
  "networkID": "1",
  "chainID": "1",
  "time": "1786569463230678",
  "memo": "withdrawal-123"
}
```

Sign using an existing encrypted Canopy `keystore.json`. The key selector may
be an address or nickname. The password is read from a terminal without echo;
for an automated offline process it may be supplied on standard input.

```sh
signer sign-send \
  --keystore-dir /secure/canopy \
  --key withdrawal \
  request.json > signed.json
```

Only the bare signed transaction is written to standard output, allowing the
calling custody system to store or broadcast it directly. Every retry must use
that same output rather than invoke the signer again:

```sh
curl -X POST "$CANOPY_RPC/v1/tx" \
  -H 'Content-Type: application/json' \
  --data-binary @signed.json
```

Before releasing it from the offline environment, inspect and verify it:

```sh
signer inspect signed.json
```

The resulting audit summary contains the native transaction hash, sender,
recipient, amount, fee, height, IDs, memo, and timestamp. `inspect` exits with a
non-zero status if any signed byte was altered.

Never provide a password or raw private key as a command-line argument or an
environment variable. Both commonly leak through process inspection, logs, or
diagnostic capture.
