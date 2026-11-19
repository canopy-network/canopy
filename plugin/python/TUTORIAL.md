# Tutorial: Implementing New Transaction Types

This tutorial walks you through implementing two custom transaction types for the Canopy Python plugin:
- **Faucet**: A test transaction that mints tokens to any address (no balance check)
- **Reward**: A transaction that mints tokens to a recipient (admin pays fee)

## Prerequisites

- Go 1.26.0 or higher (required to build Canopy)
- Python 3.9 or later
- `protoc` compiler installed (or use `grpcio-tools`)
- The python plugin base code from `plugin/python`

## Step 0: Build Canopy

Before working with plugins, build the Canopy binary from the repository root:

```bash
make build/canopy
```

This installs the `canopy` binary to your Go bin directory (`~/go/bin/canopy`).

## Step 1: Define the Protobuf Messages

Add the new message types to `contract/proto/tx.proto`:

```protobuf
// Example: MessageReward mints tokens to a recipient
message MessageReward {
  // admin_address: the admin authorizing the reward
  bytes admin_address = 1; // @gotags: json:"adminAddress"
  // recipient_address: who receives the reward
  bytes recipient_address = 2; // @gotags: json:"recipientAddress"
  // amount: tokens to mint
  uint64 amount = 3;
}

// MessageFaucet is a test-only transaction that mints tokens to any address
// No balance check required - just mints tokens for testing purposes
// NOTE: there is intentionally NO `amount` field. The minted amount is operator-controlled via
// config.json `pluginConfig.faucetAmount` (see Step 6b), not supplied per-transaction over RPC.
message MessageFaucet {
  // signer_address: the address signing this transaction (for auth)
  bytes signer_address = 1; // @gotags: json:"signerAddress"
  // recipient_address: who receives the tokens
  bytes recipient_address = 2; // @gotags: json:"recipientAddress"
}
```

## Step 2: Regenerate Python Protobuf Code

Run the generation command:

```bash
cd plugin/python
make proto
```

This creates the Python classes for `MessageReward` and `MessageFaucet` in `contract/proto/tx_pb2.py`.

## Step 3: Update Proto Imports

Update `contract/proto/__init__.py` to export the new message types:

```python
from .tx_pb2 import Transaction, MessageSend, MessageReward, MessageFaucet, FeeParams, Signature

__all__ = [
    # ... existing exports ...
    "MessageReward",
    "MessageFaucet",
]
```

## Step 4: Register the Transaction Types

Update `contract/contract.py` to register the new transaction types in `CONTRACT_CONFIG`:

```python
CONTRACT_CONFIG = {
    "name": "python_plugin_contract",
    "id": 1,
    "version": 1,
    "supported_transactions": ["send", "reward", "faucet"],  # Add here
    "transaction_type_urls": [
        "type.googleapis.com/types.MessageSend",
        "type.googleapis.com/types.MessageReward",  # Add here
        "type.googleapis.com/types.MessageFaucet",  # Add here
    ],
    "event_type_urls": [],
    "custom_state_prefixes": [FAUCET_PREFIX, REWARD_PREFIX],  # Add here
    "file_descriptor_protos": [
        any_pb2.DESCRIPTOR.serialized_pb,
        account_pb2.DESCRIPTOR.serialized_pb,
        event_pb2.DESCRIPTOR.serialized_pb,
        plugin_pb2.DESCRIPTOR.serialized_pb,
        tx_pb2.DESCRIPTOR.serialized_pb,
    ],
}
```

**Important**: The order of `supported_transactions` must match the order of `transaction_type_urls`.

**Important**: Declare every custom record prefix in `custom_state_prefixes`. Canopy reserves single-byte prefixes 1-15 and panics at handshake — before processing any block — if a declared prefix collides with that range, so use prefixes outside 1-15 (e.g. 100, 101) for your own records.

## Step 5: Add CheckTx Validation

Add cases in the `check_tx` method:

```python
async def check_tx(self, request: PluginCheckRequest) -> PluginCheckResponse:
    # ... existing fee validation ...

    type_url = request.tx.msg.type_url
    if type_url.endswith("/types.MessageSend"):
        msg = MessageSend()
        msg.ParseFromString(request.tx.msg.value)
        return self._check_message_send(msg)
    elif type_url.endswith("/types.MessageReward"):
        msg = MessageReward()
        msg.ParseFromString(request.tx.msg.value)
        return self._check_message_reward(msg)  # Add this
    elif type_url.endswith("/types.MessageFaucet"):
        msg = MessageFaucet()
        msg.ParseFromString(request.tx.msg.value)
        return self._check_message_faucet(msg)  # Add this
    else:
        raise err_invalid_message_cast()
```

### CheckMessageFaucet Implementation

Add this method inside the `Contract` class in `contract/contract.py`, after the existing `_check_message_send` method:

```python
def _check_message_faucet(self, msg: MessageFaucet) -> PluginCheckResponse:
    """
    CheckMessageFaucet statelessly validates a 'faucet' message.
    
    This is called during mempool validation BEFORE the transaction is included in a block.
    Faucet is a test transaction that mints tokens to any address without balance checks.
    
    Args:
        msg: The faucet message containing signer_address and recipient_address

    Returns:
        PluginCheckResponse with authorized signers set
        
    Raises:
        PluginError: If any validation fails (invalid address)
    """
    # Validate signer address - all Canopy addresses are exactly 20 bytes.
    # This prevents malformed addresses from entering the mempool.
    if len(msg.signer_address) != 20:
        raise err_invalid_address()

    # Validate recipient address - same 20-byte requirement.
    # The recipient will receive the minted tokens.
    if len(msg.recipient_address) != 20:
        raise err_invalid_address()

    # NOTE: there is no amount to validate here — the faucet mints a fixed, operator-controlled
    # amount (pluginConfig.faucetAmount), which is validated once at startup (see Step 6b).

    # Build and return the successful check response:
    # - recipient: who receives funds (used for indexing/notifications)
    # - authorized_signers: list of addresses that MUST sign this transaction.
    #   The FSM will verify ALL addresses in this list have valid BLS signatures.
    #   For faucet, only the signer needs to authorize the mint request.
    response = PluginCheckResponse()
    response.recipient = msg.recipient_address
    response.authorized_signers.append(msg.signer_address)
    return response
```

### CheckMessageReward Implementation

Add this method inside the `Contract` class in `contract/contract.py`, after `_check_message_faucet`:

```python
def _check_message_reward(self, msg: MessageReward) -> PluginCheckResponse:
    """
    CheckMessageReward statelessly validates a 'reward' message.
    
    Rewards allow an admin to mint tokens to any recipient address.
    The admin pays the transaction fee but the recipient gets the tokens.
    
    Args:
        msg: The reward message containing admin_address, recipient_address, and amount
        
    Returns:
        PluginCheckResponse with authorized signers set
        
    Raises:
        PluginError: If any validation fails (invalid address or amount)
    """
    # Validate admin address - the admin is the authority who can mint rewards.
    # In production, you might check against a whitelist of admin addresses.
    if len(msg.admin_address) != 20:
        raise err_invalid_address()

    # Validate recipient address - who will receive the minted tokens.
    if len(msg.recipient_address) != 20:
        raise err_invalid_address()

    # Validate amount - must be positive to be meaningful.
    if msg.amount == 0:
        raise err_invalid_amount()

    # Build and return the successful check response:
    # - authorized_signers: the ADMIN must sign to authorize this mint.
    #   Unlike faucet, the admin (not recipient) must sign, making this
    #   suitable for controlled token distribution.
    response = PluginCheckResponse()
    response.recipient = msg.recipient_address
    response.authorized_signers.append(msg.admin_address)
    return response
```

## Step 6: Add DeliverTx Execution

Add cases in the `deliver_tx` method:

```python
async def deliver_tx(self, request: PluginDeliverRequest) -> PluginDeliverResponse:
    type_url = request.tx.msg.type_url
    if type_url.endswith("/types.MessageSend"):
        msg = MessageSend()
        msg.ParseFromString(request.tx.msg.value)
        return await self._deliver_message_send(msg, request.tx.fee)
    elif type_url.endswith("/types.MessageReward"):
        msg = MessageReward()
        msg.ParseFromString(request.tx.msg.value)
        return await self._deliver_message_reward(msg, request.tx.fee)  # Add this
    elif type_url.endswith("/types.MessageFaucet"):
        msg = MessageFaucet()
        msg.ParseFromString(request.tx.msg.value)
        return await self._deliver_message_faucet(msg)  # Add this (no fee for faucet)
    else:
        raise err_invalid_message_cast()
```

### DeliverMessageFaucet Implementation

Add this async method inside the `Contract` class in `contract/contract.py`, after the existing `_deliver_message_send` method:

The faucet transaction mints tokens without requiring the signer to have any balance:

```python
async def _deliver_message_faucet(self, msg: MessageFaucet) -> PluginDeliverResponse:
    """
    DeliverMessageFaucet handles a faucet message by minting tokens to the recipient.
    
    This is called AFTER CheckTx passes and the transaction is included in a block.
    Unlike CheckTx, DeliverTx CAN read and write blockchain state.
    Faucet is special: it mints tokens without requiring any existing balance (for testing).
    
    Args:
        msg: The faucet message containing recipient_address (the minted amount comes from
            pluginConfig.faucetAmount, not from the message)
        
    Returns:
        PluginDeliverResponse with empty error field on success
    """
    # Guard clause: verify the plugin infrastructure is initialized.
    # plugin is the connection to Canopy FSM; config holds chain settings.
    if not self.plugin or not self.config:
        raise PluginError(1, "plugin", "plugin or config not initialized")

    # Generate a unique query ID to correlate request/response in batch reads.
    # When reading multiple keys, each gets a query_id so we can match results.
    # Use random number in safe JavaScript integer range for compatibility.
    recipient_query_id = random.randint(0, 2**53)

    # Generate the state key for the recipient's account.
    # key_for_account creates a length-prefixed key: [prefix][address]
    # This ensures unique keys in the key-value store.
    recipient_key = key_for_account(msg.recipient_address)

    # Request the current state of the recipient's account from the FSM.
    # state_read sends a request over the Unix socket to the Canopy FSM,
    # which reads from the blockchain's state database.
    response = await self.plugin.state_read(
        self,
        PluginStateReadRequest(
            keys=[
                PluginKeyRead(query_id=recipient_query_id, key=recipient_key),
            ]
        ),
    )

    # Check for application-level errors from the FSM read operation.
    # HasField checks if the optional error field is populated in protobuf.
    if response.HasField("error"):
        result = PluginDeliverResponse()
        result.error.CopyFrom(response.error)  # Copy error to response
        return result

    # Extract the recipient's current account bytes from the response.
    # Results are returned with their query_id so we can match them.
    # If the account doesn't exist yet, recipient_bytes will be None.
    recipient_bytes = None
    for resp in response.results:
        if resp.query_id == recipient_query_id and resp.entries:
            recipient_bytes = resp.entries[0].value

    # Unmarshal the protobuf Account message.
    # If bytes are None, create a new empty Account with default values (amount=0).
    recipient_account = unmarshal(Account, recipient_bytes) if recipient_bytes else Account()

    # CORE LOGIC: Add the faucet amount to the recipient's balance.
    # This is where tokens are "minted" - we simply increase the balance.
    # No balance check needed because faucet creates tokens from nothing.
    # The amount is NOT taken from the message; it comes from the operator-controlled
    # pluginConfig.faucetAmount, decoded + validated once at startup (see Step 6b).
    faucet_amount = self.plugin.settings.faucet_amount
    recipient_account.amount += faucet_amount

    # Marshal the updated account back to protobuf bytes for storage.
    recipient_bytes_new = marshal(recipient_account)

    # Write the updated state back to the blockchain via the FSM.
    # sets contains key-value pairs to write; deletes would remove keys.
    # This persists the recipient's new balance to the blockchain.
    write_resp = await self.plugin.state_write(
        self,
        PluginStateWriteRequest(
            sets=[
                PluginSetOp(key=recipient_key, value=recipient_bytes_new),
            ],
        ),
    )

    # Build the response, copying any error from the write operation.
    result = PluginDeliverResponse()
    if write_resp.HasField("error"):
        result.error.CopyFrom(write_resp.error)
    # Empty error field means success
    return result
```

### DeliverMessageReward Implementation

Add this async method inside the `Contract` class in `contract/contract.py`, after `_deliver_message_faucet`:

The reward transaction mints tokens to a recipient, with the admin paying the transaction fee:

```python
async def _deliver_message_reward(self, msg: MessageReward, fee: int) -> PluginDeliverResponse:
    """
    DeliverMessageReward handles a reward message by minting tokens to the recipient.
    
    The admin authorizes this transaction and pays the transaction fee.
    This demonstrates a more complex DeliverTx with multiple account updates.
    
    Args:
        msg: The reward message containing admin_address, recipient_address, and amount
        fee: The transaction fee that the admin must pay
        
    Returns:
        PluginDeliverResponse with empty error field on success
    """
    # Guard clause: verify the plugin infrastructure is initialized.
    if not self.plugin or not self.config:
        raise PluginError(1, "plugin", "plugin or config not initialized")

    # Generate unique query IDs for each key to correlate responses with requests.
    # This is necessary because results may come back in any order.
    admin_query_id = random.randint(0, 2**53)
    recipient_query_id = random.randint(0, 2**53)
    fee_query_id = random.randint(0, 2**53)

    # Calculate the state database keys for each entity we need to read/write.
    # Each key type has a unique prefix to avoid collisions in the key-value store.
    admin_key = key_for_account(msg.admin_address)        # Admin's account (pays fee)
    recipient_key = key_for_account(msg.recipient_address) # Recipient's account (gets tokens)
    fee_pool_key = key_for_fee_pool(self.config.chain_id)  # Fee pool for this chain

    # Batch read all three accounts in a single round-trip to the FSM.
    # This is more efficient than making three separate read requests.
    response = await self.plugin.state_read(
        self,
        PluginStateReadRequest(
            keys=[
                PluginKeyRead(query_id=fee_query_id, key=fee_pool_key),
                PluginKeyRead(query_id=admin_query_id, key=admin_key),
                PluginKeyRead(query_id=recipient_query_id, key=recipient_key),
            ]
        ),
    )

    # Check for application-level errors from the FSM read operation.
    if response.HasField("error"):
        result = PluginDeliverResponse()
        result.error.CopyFrom(response.error)
        return result

    # Extract each account's bytes from the response, matching by query_id.
    # None means the account doesn't exist yet (new account).
    admin_bytes = None
    recipient_bytes = None
    fee_pool_bytes = None

    for resp in response.results:
        if resp.query_id == admin_query_id:
            admin_bytes = resp.entries[0].value if resp.entries else None
        elif resp.query_id == recipient_query_id:
            recipient_bytes = resp.entries[0].value if resp.entries else None
        elif resp.query_id == fee_query_id:
            fee_pool_bytes = resp.entries[0].value if resp.entries else None

    # Unmarshal the protobuf messages using the appropriate type schemas.
    # If bytes are None, create empty objects with default values (amount=0).
    admin_account = unmarshal(Account, admin_bytes) if admin_bytes else Account()
    recipient_account = unmarshal(Account, recipient_bytes) if recipient_bytes else Account()
    fee_pool = unmarshal(Pool, fee_pool_bytes) if fee_pool_bytes else Pool()

    # BUSINESS LOGIC: Verify admin has sufficient funds to pay the transaction fee.
    # This is a critical check - without it, admins could spam free transactions.
    if admin_account.amount < fee:
        raise err_insufficient_funds()

    # CORE STATE CHANGES: Update balances for all three entities.
    # 1. Deduct fee from admin's balance
    admin_account.amount -= fee  # Admin pays the transaction fee
    # 2. Mint new tokens to recipient (this increases total supply!)
    recipient_account.amount += msg.amount  # Mint tokens (created from nothing)
    # 3. Add fee to the pool for validator rewards
    fee_pool.amount += fee

    # Marshal all updated accounts to protobuf bytes for storage.
    admin_bytes_new = marshal(admin_account)
    recipient_bytes_new = marshal(recipient_account)
    fee_pool_bytes_new = marshal(fee_pool)

    # Write all state changes atomically.
    # Special case: if admin's balance is now zero, delete their account to save space.
    # This is a common pattern - zero-balance accounts are removed from state.
    if admin_account.amount == 0:
        # Admin account is empty - delete it instead of storing zeros.
        write_resp = await self.plugin.state_write(
            self,
            PluginStateWriteRequest(
                sets=[
                    PluginSetOp(key=fee_pool_key, value=fee_pool_bytes_new),
                    PluginSetOp(key=recipient_key, value=recipient_bytes_new),
                ],
                deletes=[PluginDeleteOp(key=admin_key)],  # Remove empty account
            ),
        )
    else:
        # Admin still has balance - update all three accounts.
        write_resp = await self.plugin.state_write(
            self,
            PluginStateWriteRequest(
                sets=[
                    PluginSetOp(key=fee_pool_key, value=fee_pool_bytes_new),
                    PluginSetOp(key=admin_key, value=admin_bytes_new),
                    PluginSetOp(key=recipient_key, value=recipient_bytes_new),
                ],
            ),
        )

    # Build the response, copying any error from the write operation.
    result = PluginDeliverResponse()
    if write_resp.HasField("error"):
        result.error.CopyFrom(write_resp.error)
    # Empty error field means success
    return result
```

## Step 5b: Expose Custom RPC Endpoints

A plugin can serve its own RPC endpoints for chain-specific data. Canopy core only exposes a single, generic, read-only transport over the unix socket: `Plugin.query_state(height, read)`, which returns raw key/value state at a historical height (`0` = latest committed). The plugin process owns its HTTP server entirely, so you can register as many routes as you want and decode your own keys/protobufs into any response shape. Canopy never needs to know about your endpoints.

> Note: account and pool queries already exist in the Canopy node's own RPC (`/v1/query/account`, `/v1/query/pool`), so they make poor examples of a *custom* endpoint. This tutorial exposes faucet and reward data instead, which only the plugin knows about.

### Persist queryable records during DeliverTx

For data to be queryable, it has to live in state. The `_deliver_message_faucet` and `_deliver_message_reward` handlers above persist a small record alongside the balance update:

- A `Faucet` record per recipient (`recipientAddress`, `totalAmount`, `count`), stored under prefix `b"\x64"` (100) via `key_for_faucet(addr)`.
- A `Reward` record per recipient (`recipientAddress`, `lastAdminAddress`, `totalAmount`, `count`), stored under prefix `b"\x65"` (101) via `key_for_reward(addr)`.

> **Important — avoid prefix collisions:** the plugin reads and writes Canopy's FSM keyspace directly (that's why `send` works on real accounts at prefix `1`). Canopy reserves single-byte prefixes `1–15` for its own state (e.g. `3` = validators, `4` = committees). Your plugin-specific records must use prefixes outside that range — otherwise a range/list scan over your prefix will return core records (validators, committees, …) that fail to decode as your type. We use `100`/`101` here.

These `Faucet`/`Reward` messages and the `key_for_faucet`/`key_for_reward`/`faucet_prefix`/`reward_prefix` helpers live in `contract/proto/tx.proto` and `contract/contract.py`.

### The detached, read-only query

The framework adds `Plugin.query_state(height, read)` (`contract/plugin.py`), which mirrors `state_read` but is **not** tied to an in-flight tx/block lifecycle. It allocates its own RANDOM request id (instead of reusing an FSM request id), so it is safe to call from custom RPC handlers:

```python
async def query_state(self, height, request):
    request_id = random.getrandbits(64)            # fresh random id (detached)
    future = asyncio.Future()
    self._pending[request_id] = future
    message = PluginToFSM()
    message.id = request_id
    message.query.height = height
    message.query.read.CopyFrom(request)
    await self._send_proto_msg(message)
    response = await asyncio.wait_for(future, timeout=10.0)
    return response.query.read
```

This relies on the new `PluginQueryRequest`/`PluginQueryResponse` messages and the `query = 10` fields added to both oneofs in `contract/proto/plugin.proto`.

### Register the endpoints

The base plugin already ships a **skeleton** `contract/rpc.py`. Its `start_rpc_server(plugin)` runs the plugin's HTTP server using the Python standard library `http.server` in a background daemon thread (no extra dependencies), but it registers **no routes by default** — it's a blank canvas. It is also already started from `main.py`:

```python
plugin = await start_plugin(default_config())
start_rpc_server(plugin)  # skeleton: registers no routes by default
```

So your job here is to **add your faucet/reward routes and handlers to the existing skeleton**, not to create the file. Add route dispatch in `PluginRPCHandler.do_GET` and implement the handlers (add as many routes as you like):

```python
def do_GET(self):
    parsed = urlparse(self.path)
    query = parse_qs(parsed.query)
    # GET /v1/query/faucets[?address=<hex>][&height=<uint64>]
    if parsed.path == "/v1/query/faucets":
        self._handle_query_faucets(query)
    # GET /v1/query/rewards[?address=<hex>][&height=<uint64>]
    elif parsed.path == "/v1/query/rewards":
        self._handle_query_rewards(query)
    else:
        self._write_json_error(404, "not found")
```

Because the HTTP server runs on a background thread while the plugin owns an asyncio event loop, each handler schedules the async `query_state` coroutine onto the plugin's loop and blocks for the result:

```python
future = asyncio.run_coroutine_threadsafe(plugin.query_state(height, request), plugin._loop)
result = future.result(timeout=15)
```

Each handler then decodes the raw bytes into the plugin's own `Faucet`/`Reward` types:

- Without `?address`, it does a **range read** over the record prefix (`faucet_prefix()` / `reward_prefix()`) and returns every record.
- With `?address=<hex>`, it does a **single-key read with a proof** via the new `query_state_with_proof(height, key)` helper (`key_for_faucet(addr)` / `key_for_reward(addr)`) and returns just that recipient's record together with the data needed to verify it trustlessly (see "Return a verifiable proof" below).

> **Important — the range prefix must be length-prefixed.** Stored keys are length-prefixed (`key_for_faucet(addr)` = `join_len_prefix(FAUCET_PREFIX, addr)`), so the prefix you pass to a **range read** must be length-prefixed too — that's exactly what `faucet_prefix()` returns (`join_len_prefix(FAUCET_PREFIX)`, i.e. `b"\x01\x64"`), **not** the raw byte `b"\x64"`. Passing the raw prefix byte misaligns Canopy's length-prefixed iterator and crashes the node with `panic: corrupt or incomplete key`. (Single-key reads via `key_for_faucet(addr)` are already length-prefixed, so they're unaffected.)

### Return a verifiable proof (recommended)

A plain read answers "what does this node *say* the value is?" — the client has to trust the node. The single-recipient endpoint does better: it returns a **sparse-merkle-tree (SMT) proof** so the client can verify the value **trustlessly**, without trusting the node that served it.

The framework adds `Plugin.query_state_with_proof(height, key)` (`contract/plugin.py`), the proof-carrying sibling of `query_state`. It sends a single-key read with `prove=True` and returns a small object with:

- `value` (bytes) — the record's raw value (empty when the key doesn't exist)
- `exists` (bool) — whether the key currently holds a non-empty value (membership vs. non-membership proof)
- `proof` (list of `ProofNode`, each `{key, value, bitmask}`) — the leaf being proven plus the sibling nodes along the path to the root
- `state_root` (bytes) — the state commitment root the proof commits to, at the resolved height
- `height` (int) — the concrete committed height the query was resolved at (so `height=0` "latest" still tells the client which root to trust)

The by-address faucet handler calls this helper and shapes a JSON object:

```python
def _handle_query_faucets(self, query):
    addr_hex = (query.get("address") or [""])[0]
    if not addr_hex:
        # no address → fall back to the existing range-read list endpoint
        return self._handle_query_faucets_list(query)
    height = int((query.get("height") or ["0"])[0])  # 0 = latest
    key = key_for_faucet(hex_to_bytes(addr_hex))

    future = asyncio.run_coroutine_threadsafe(
        self.plugin.query_state_with_proof(height, key), self.plugin._loop
    )
    sp = future.result(timeout=15)  # StateProof

    # `faucet` is a convenience, already-decoded view of `value`; a trustless client should still
    # verify `value` + `proof` against `stateRoot` itself rather than trust this decoded field.
    faucet = unmarshal(Faucet, sp.value) if sp.exists else Faucet()
    self._write_json({
        "height": sp.height,
        "stateRoot": sp.state_root.hex(),
        "key": key.hex(),
        "exists": sp.exists,
        "value": sp.value.hex(),
        "proof": [
            {"key": n.key.hex(), "value": n.value.hex(), "bitmask": n.bitmask}
            for n in sp.proof
        ],
        "faucet": {
            "recipientAddress": faucet.recipient_address.hex(),
            "totalAmount": faucet.total_amount,
            "count": faucet.count,
        },
    })
```

**How a client verifies trustlessly:** it authenticates `stateRoot` independently — it does **not** take the node's word for it — by checking the quorum certificate for that `height` (the root is signed by the validator set). It then recomputes the root from `value` + `proof` and checks it equals the authenticated `stateRoot`. If they match, the value is provably the committed state; if a node lies about `value` or `proof`, the recomputed root simply won't match and verification **fails** — a dishonest node can only make the check fail, it can never forge a value that verifies. (When `exists` is `false`, the proof is a *non-membership* proof: it proves the key is genuinely absent.)

The `faucet` field is a **convenience view only** — the already-decoded record so callers that trust the node can skip the proof math. A trustless client should ignore it and verify `value` + `proof` against `stateRoot` itself. The **reward** endpoint follows the exact same pattern through its own key builder (`key_for_reward(addr)`), returning a `reward` convenience field alongside the identical `height`/`stateRoot`/`key`/`exists`/`value`/`proof` fields.

The server is already started from `main.py` (no change needed):

```python
plugin = await start_plugin(default_config())
start_rpc_server(plugin)
```

The listen address comes from the `rpc_address` config field (default `0.0.0.0:50010`). The RPC server is optional and non-fatal: set `rpc_address` to empty to disable it, and a bind failure (e.g. port already in use) is logged without crashing the plugin.

### Query the endpoints

After faucet/reward transactions have been included in blocks:

```bash
# all faucet records
curl 'http://localhost:50010/v1/query/faucets'
# {"faucets":[{"recipientAddress":"...","totalAmount":1000000000,"count":1}],"count":1,"height":0}

# a single recipient's faucet record — now returns a verifiable proof
curl 'http://localhost:50010/v1/query/faucets?address=<recipient-hex>'
# {
#   "height": 42,
#   "stateRoot": "9f2c…",                       // authenticate this via the quorum certificate
#   "key": "0164…",                             // the length-prefixed state key that was proven
#   "exists": true,
#   "value": "0a14…",                           // raw record bytes the proof commits to
#   "proof": [                                  // leaf + sibling nodes up to the root
#     {"key": "…", "value": "…", "bitmask": 0},
#     {"key": "…", "value": "…", "bitmask": 1}
#   ],
#   "faucet": {                                 // convenience view only — verify value+proof instead
#     "recipientAddress": "<recipient-hex>",
#     "totalAmount": 1000000000,
#     "count": 1
#   }
# }

# all reward records (optionally at a historical height)
curl 'http://localhost:50010/v1/query/rewards?height=42'

# a single recipient's reward record — same proof shape, with a `reward` convenience field
curl 'http://localhost:50010/v1/query/rewards?address=<recipient-hex>'
```

## Step 6b: Configure the faucet amount via `pluginConfig`

The faucet amount is **operator-controlled**, not supplied per-transaction. It is read from the
node's `config.json` `pluginConfig` field, which Canopy forwards verbatim to the plugin during the
handshake. `pluginConfig` is intentionally typed as *any* JSON on the node side, so each plugin can
define its own schema and validate it at startup.

### How it flows

1. **Node config** (`~/.canopy/config.json`) carries an opaque `pluginConfig` JSON value:

   ```json
   {
     "plugin": "python",
     "pluginConfig": { "faucetAmount": 1000000000 }
   }
   ```

2. **Canopy core** (`lib/config.go` `MainConfig.PluginConfig`) reads it as raw JSON and forwards it
   to the plugin in the handshake response (`PluginFSMConfig.plugin_config`). Core never inspects it.

3. **The plugin** decodes + validates it at startup. The base SDK (`contract/plugin.py`) defines the
   expected shape and the single validation entry point:

   ```python
   @dataclass
   class Config:
       chain_id: int = 1
       data_dir_path: str = "/tmp/plugin/"
       rpc_address: str = "0.0.0.0:50010"
       # opaque pluginConfig mirrored from the node config.json (any JSON)
       plugin_config: Any = None

   @dataclass
   class PluginSettings:
       # the typed, validated shape the plugin expects pluginConfig to take
       faucet_amount: int = 0

   def parse_plugin_config(raw: bytes) -> PluginSettings:
       """Decode + validate the forwarded JSON at startup; a malformed blob fails the handshake
       (and the plugin exits), satisfying 'validate it is in the expected form'."""
       ...
   ```

   The handshake (the `config` branch of `_handle_fsm_request` in `contract/plugin.py`) calls
   `parse_plugin_config(msg.config.plugin_config)` and stores the result on `self.settings`, so
   handlers can read `self.plugin.settings.faucet_amount`. An invalid blob is logged and the plugin
   process exits (fail fast).

### Require a non-zero faucet amount (tutorial validation)

Because this plugin's faucet is useless without a configured amount, extend `parse_plugin_config` to
reject a missing/zero `faucetAmount` so the plugin fails fast at startup:

```python
def parse_plugin_config(raw: bytes) -> PluginSettings:
    settings = PluginSettings()
    if not raw:
        raise err_invalid_plugin_config('missing pluginConfig (expected { "faucetAmount": <uint64> })')
    try:
        data = json.loads(raw)
    except (ValueError, TypeError) as err:
        raise err_invalid_plugin_config(str(err))
    if not isinstance(data, dict):
        raise err_invalid_plugin_config("expected a JSON object")
    settings.faucet_amount = int(data.get("faucetAmount", 0) or 0)
    if settings.faucet_amount == 0:
        raise err_invalid_plugin_config("faucetAmount must be greater than zero")
    return settings
```

The tutorial test (`tutorial/rpc_test.py`) asserts the faucet mints exactly this configured amount
(`FAUCET_CONFIGURED_AMOUNT`), proving the amount comes from `pluginConfig` and not from the RPC
request.

## Step 7: Running Canopy with the Plugin

To run Canopy with the Python plugin enabled, you need to configure the `plugin` field in your Canopy configuration file.

### 1. Locate your config.json

The configuration file is typically located at `~/.canopy/config.json`. If it doesn't exist, start Canopy once to generate the default configuration:

```bash
canopy start
# Stop it after it generates the config (Ctrl+C)
```

> **Note**: If your Go bin directory is not in your PATH, use `~/go/bin/canopy` instead of `canopy`.

### 2. Enable the Python plugin

Edit `~/.canopy/config.json` and add or modify the `plugin` field to `"python"`:

```json
{
  "plugin": "python",
  "pluginConfig": { "faucetAmount": 1000000000 },
  ...
}
```

**Note**: The `plugin` field should be at the top level of the JSON configuration.

**Note**: `pluginConfig` is an opaque (any) JSON value that Canopy forwards to the plugin unchanged. For this tutorial the Python plugin expects `{ "faucetAmount": <uint64> }` and validates it at startup (see Step 6b); the plugin will refuse to start if it is missing or `faucetAmount` is zero.

### 3. Start Canopy

```bash
canopy start
```

> **Note**: If your Go bin directory is not in your PATH, use `~/go/bin/canopy start` instead.

> **Warning**: You may see error logs about the plugin failing to start on the first attempt. This is normal - Canopy will retry and the plugin should start successfully within a few seconds, then begin producing blocks.

Canopy will automatically start the Python plugin and connect to it.

### 4. Verify the plugin is running

Check the plugin logs:

```bash
tail -f /tmp/plugin/python-plugin.log
```

### Step 7b: Running with Docker (Alternative)

Instead of running Canopy and the plugin locally, you can use Docker to run everything in a container.

#### 1. Build the Docker image

From the repository root:

```bash
make docker/plugin PLUGIN=python
```

This creates a `canopy-python` image containing both Canopy and the Python plugin pre-configured.

#### 2. Run the container

```bash
make docker/run-python
```

Or with a custom volume mount for persistent data:

```bash
docker run -v ~/.canopy:/root/.canopy canopy-python
```

#### 3. Expose RPC ports (for running tests)

To run tests against the containerized Canopy, expose the RPC ports:

```bash
docker run -p 50002:50002 -p 50003:50003 -v ~/.canopy:/root/.canopy canopy-python
```

| Port | Service |
|------|---------|
| 50002 | RPC API (transactions, queries) |
| 50003 | Admin RPC (keystore operations) |

Now you can run tests from your host machine that connect to `localhost:50002` and `localhost:50003`.

#### 4. View logs inside the container

```bash
# Get the container ID
docker ps

# View Canopy logs
docker exec -it <container_id> tail -f /root/.canopy/logs/log

# View plugin logs
docker exec -it <container_id> tail -f /tmp/plugin/python-plugin.log
```

#### 5. Interactive shell (for debugging)

To inspect the container or debug issues:

```bash
docker run -it --entrypoint /bin/sh canopy-python
```

## Step 8: Testing

Run the integration tests from the `tutorial` directory. `make test` runs the transaction tests **and** the custom RPC endpoints test:

```bash
cd plugin/python/tutorial
make test
```

This is equivalent to running, from the `tutorial` directory (transactions first, then the custom RPC endpoints test):

```bash
cd plugin/python/tutorial
pip install -r requirements.txt
python rpc_test.py && python rpc_test.py custom
```

### Test Prerequisites

1. **Canopy node must be running** with the Python plugin enabled (see Step 7)

2. **Plugin must have the new transaction types registered** (faucet, reward)

3. **The plugin's RPC server must be reachable** on port `50010` (Step 5b) for the custom RPC test

### What the Tests Do

`python rpc_test.py` exercises the transaction flow:

1. **Create test accounts** - Creates two new accounts in the Canopy keystore
2. **Faucet test** - Mints tokens to account 1 using the faucet transaction
3. **Send test** - Sends tokens from account 1 to account 2
4. **Reward test** - Account 2 rewards tokens back to account 1
5. **Balance verification** - Confirms balances changed as expected

`python rpc_test.py custom` then verifies the custom RPC endpoints (Step 5b):

1. **Submit faucet/reward transactions** and wait for inclusion
2. **Query `/v1/query/faucets` and `/v1/query/rewards`** (both the list and single-recipient forms)
3. **Validate the returned records' structure** (valid hex addresses, `count >= 1`, `totalAmount >= 1`), which also guards against prefix-collision regressions

## Transaction Signing Details

When submitting signed transactions to the RPC endpoint (`/v1/tx`), the signature must be computed over the protobuf-encoded transaction with the signature field omitted.

Key points:
- Canopy uses BLS12-381 signatures (96-byte G2 signatures)
- Use the `blspy` library with `BasicSchemeMPL` for signing
- Encode every protobuf `bytes` field as a hexadecimal string in the message JSON
- Sign the deterministically marshaled protobuf bytes of the Transaction (without signature field)
- Submit native and plugin-only message types as structured `msg` JSON through `/v1/tx`
- The JSON hex encoding does not change the raw bytes stored in the protobuf message

See `rpc_test.py` in `plugin/python/tutorial` for the complete signing implementation.

## Common Issues

### "message name faucet is unknown"
- Make sure `CONTRACT_CONFIG["supported_transactions"]` includes `"faucet"`
- Ensure `CONTRACT_CONFIG["transaction_type_urls"]` includes the type URL
- Restart the plugin after making changes

### Invalid signature errors
- Ensure you're signing the protobuf bytes, not JSON
- Verify the transaction structure matches Canopy's `lib.Transaction`
- Use `BasicSchemeMPL` from `blspy` (not `AugSchemeMPL` or `PopSchemeMPL`)
- Check that the address derivation (SHA256 → first 20 bytes) matches

### Balance not updating
- Wait for block finalization (at least 6-12 seconds)
- Check plugin logs
- Verify the transaction was included in a block (check `/v1/query/txs-by-sender`)

## Project Structure

After implementation, your files should look like:

```
plugin/python/
├── contract/
│   ├── contract.py       # Updated with reward/faucet handlers
│   ├── proto/
│   │   ├── tx.proto      # Updated with MessageReward/MessageFaucet
│   │   ├── tx_pb2.py     # Regenerated
│   │   └── __init__.py   # Updated exports
│   └── ...
├── tutorial/             # Test project for verifying implementation
│   ├── proto/
│   │   ├── tx.proto      # Pre-defined with faucet/reward messages
│   │   └── tx_pb2.py     # Pre-generated
│   ├── rpc_test.py       # RPC test suite
│   ├── main.py
│   ├── requirements.txt
│   └── Makefile
├── TUTORIAL.md           # This file
└── ...
```

## Running the Tests

After implementing the new transaction types and starting Canopy with the plugin:

```bash
# Terminal 1: Start Canopy with the plugin
cd ~/canopy
~/go/bin/canopy start

# Terminal 2: Run the tests (transactions + custom RPC endpoints)
cd ~/canopy/plugin/python/tutorial
make test
```

The test will:
1. Create two new accounts in the keystore
2. Use faucet to mint 1000 tokens to account 1
3. Send 100 tokens from account 1 to account 2
4. Use reward to mint 50 tokens from account 2 to account 1
5. Verify all transactions were included in blocks

## Adding New State Keys

If your transaction needs new state storage, add key generation functions:

```python
# In contract.py
VALIDATOR_PREFIX = b"\x03"
DELEGATION_PREFIX = b"\x04"

def key_for_validator(address: bytes) -> bytes:
    """Generate state database key for a validator."""
    return join_len_prefix(VALIDATOR_PREFIX, address)

def key_for_delegation(delegator: bytes, validator: bytes) -> bytes:
    """Generate state database key for a delegation."""
    return join_len_prefix(DELEGATION_PREFIX, delegator, validator)
```
