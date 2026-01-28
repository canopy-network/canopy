# Tutorial: Implementing New Transaction Types

This tutorial walks you through implementing two custom transaction types for the Canopy Python plugin:
- **Faucet**: A test transaction that mints tokens to any address (no balance check)
- **Reward**: A transaction that mints tokens to a recipient (admin pays fee)

## Prerequisites

- Python 3.9 or later
- `protoc` compiler installed (or use `grpcio-tools`)
- The python plugin base code from `plugin/python`
- Canopy node (running with the plugin will be explained in Step 7)

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
message MessageFaucet {
  // signer_address: the address signing this transaction (for auth)
  bytes signer_address = 1; // @gotags: json:"signerAddress"
  // recipient_address: who receives the tokens
  bytes recipient_address = 2; // @gotags: json:"recipientAddress"
  // amount: tokens to mint
  uint64 amount = 3;
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

```python
def _check_message_faucet(self, msg: MessageFaucet) -> PluginCheckResponse:
    """CheckMessageFaucet statelessly validates a 'faucet' message."""
    # Check signer address (must be exactly 20 bytes)
    if len(msg.signer_address) != 20:
        raise err_invalid_address()

    # Check recipient address (must be exactly 20 bytes)
    if len(msg.recipient_address) != 20:
        raise err_invalid_address()

    # Check amount (must be greater than 0)
    if msg.amount == 0:
        raise err_invalid_amount()

    # Return authorized signers (signer must sign)
    response = PluginCheckResponse()
    response.recipient = msg.recipient_address
    response.authorized_signers.append(msg.signer_address)
    return response
```

### CheckMessageReward Implementation

```python
def _check_message_reward(self, msg: MessageReward) -> PluginCheckResponse:
    """CheckMessageReward statelessly validates a 'reward' message."""
    # Check admin address (must be exactly 20 bytes)
    if len(msg.admin_address) != 20:
        raise err_invalid_address()

    # Check recipient address (must be exactly 20 bytes)
    if len(msg.recipient_address) != 20:
        raise err_invalid_address()

    # Check amount (must be greater than 0)
    if msg.amount == 0:
        raise err_invalid_amount()

    # Return authorized signers (admin must sign)
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

The faucet transaction mints tokens without requiring the signer to have any balance:

```python
async def _deliver_message_faucet(self, msg: MessageFaucet) -> PluginDeliverResponse:
    """DeliverMessageFaucet handles a 'faucet' message (mints tokens to recipient - no fee, no balance check)."""
    if not self.plugin or not self.config:
        raise PluginError(1, "plugin", "plugin or config not initialized")

    # Generate query ID
    recipient_query_id = random.randint(0, 2**53)

    # Calculate key
    recipient_key = key_for_account(msg.recipient_address)

    # Read current recipient state
    response = await self.plugin.state_read(
        self,
        PluginStateReadRequest(
            keys=[
                PluginKeyRead(query_id=recipient_query_id, key=recipient_key),
            ]
        ),
    )

    # Check for internal error
    if response.HasField("error"):
        result = PluginDeliverResponse()
        result.error.CopyFrom(response.error)
        return result

    # Get recipient bytes
    recipient_bytes = None
    for resp in response.results:
        if resp.query_id == recipient_query_id and resp.entries:
            recipient_bytes = resp.entries[0].value

    # Unmarshal recipient account (or create new if doesn't exist)
    recipient_account = unmarshal(Account, recipient_bytes) if recipient_bytes else Account()

    # Mint tokens to recipient
    recipient_account.amount += msg.amount

    # Marshal updated state
    recipient_bytes_new = marshal(recipient_account)

    # Write state changes
    write_resp = await self.plugin.state_write(
        self,
        PluginStateWriteRequest(
            sets=[
                PluginSetOp(key=recipient_key, value=recipient_bytes_new),
            ],
        ),
    )

    result = PluginDeliverResponse()
    if write_resp.HasField("error"):
        result.error.CopyFrom(write_resp.error)
    return result
```

### DeliverMessageReward Implementation

The reward transaction mints tokens to a recipient, with the admin paying the transaction fee:

```python
async def _deliver_message_reward(self, msg: MessageReward, fee: int) -> PluginDeliverResponse:
    """DeliverMessageReward handles a 'reward' message (mints tokens to recipient)."""
    if not self.plugin or not self.config:
        raise PluginError(1, "plugin", "plugin or config not initialized")

    # Generate query IDs
    admin_query_id = random.randint(0, 2**53)
    recipient_query_id = random.randint(0, 2**53)
    fee_query_id = random.randint(0, 2**53)

    # Calculate keys
    admin_key = key_for_account(msg.admin_address)
    recipient_key = key_for_account(msg.recipient_address)
    fee_pool_key = key_for_fee_pool(self.config.chain_id)

    # Read current state
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

    # Check for internal error
    if response.HasField("error"):
        result = PluginDeliverResponse()
        result.error.CopyFrom(response.error)
        return result

    # Parse results by query_id
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

    # Unmarshal accounts
    admin_account = unmarshal(Account, admin_bytes) if admin_bytes else Account()
    recipient_account = unmarshal(Account, recipient_bytes) if recipient_bytes else Account()
    fee_pool = unmarshal(Pool, fee_pool_bytes) if fee_pool_bytes else Pool()

    # Admin must have enough to pay the fee
    if admin_account.amount < fee:
        raise err_insufficient_funds()

    # Apply state changes
    admin_account.amount -= fee  # Admin pays fee
    recipient_account.amount += msg.amount  # Mint tokens to recipient
    fee_pool.amount += fee

    # Marshal updated state
    admin_bytes_new = marshal(admin_account)
    recipient_bytes_new = marshal(recipient_account)
    fee_pool_bytes_new = marshal(fee_pool)

    # Write state changes
    if admin_account.amount == 0:
        # Delete drained admin account
        write_resp = await self.plugin.state_write(
            self,
            PluginStateWriteRequest(
                sets=[
                    PluginSetOp(key=fee_pool_key, value=fee_pool_bytes_new),
                    PluginSetOp(key=recipient_key, value=recipient_bytes_new),
                ],
                deletes=[PluginDeleteOp(key=admin_key)],
            ),
        )
    else:
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

    result = PluginDeliverResponse()
    if write_resp.HasField("error"):
        result.error.CopyFrom(write_resp.error)
    return result
```

## Step 7: Running Canopy with the Plugin

To run Canopy with the Python plugin enabled, you need to configure the `plugin` field in your Canopy configuration file.

### 1. Locate your config.json

The configuration file is typically located at `~/.canopy/config.json`. If it doesn't exist, start Canopy once to generate the default configuration:

```bash
~/go/bin/canopy start
# Stop it after it generates the config (Ctrl+C)
```

### 2. Enable the Python plugin

Edit `~/.canopy/config.json` and add or modify the `plugin` field to `"python"`:

```json
{
  "plugin": "python",
  ...
}
```

**Note**: The `plugin` field should be at the top level of the JSON configuration.

### 3. Start Canopy

```bash
~/go/bin/canopy start
```

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

Run the RPC tests from the `tutorial` directory:

```bash
cd plugin/python/tutorial
pip install -r requirements.txt
python rpc_test.py
```

Or using make:

```bash
cd plugin/python/tutorial
make install
make test
```

### Test Prerequisites

1. **Canopy node must be running** with the Python plugin enabled (see Step 7)

2. **Plugin must have the new transaction types registered** (faucet, reward)

### What the Tests Do

1. **Create test accounts** - Creates two new accounts in the Canopy keystore
2. **Faucet test** - Mints tokens to account 1 using the faucet transaction
3. **Send test** - Sends tokens from account 1 to account 2
4. **Reward test** - Account 2 rewards tokens back to account 1
5. **Balance verification** - Confirms balances changed as expected

## Transaction Signing Details

When submitting signed transactions to the RPC endpoint (`/v1/tx`), the signature must be computed over the protobuf-encoded transaction with the signature field omitted.

Key points:
- Canopy uses BLS12-381 signatures (96-byte G2 signatures)
- Use the `blspy` library with `BasicSchemeMPL` for signing
- Sign the deterministically marshaled protobuf bytes of the Transaction (without signature field)
- For plugin-only message types (faucet, reward), use `msgTypeUrl` and `msgBytes` fields for exact byte control

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

# Terminal 2: Run the tests
cd ~/canopy/plugin/python/tutorial
pip install -r requirements.txt
python rpc_test.py
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
