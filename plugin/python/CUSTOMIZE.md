# Custom Transaction Guide - Python

This guide shows how to add a custom "reward" transaction to the Python plugin.

## Step 1: Register the Transaction

In `contract/contract.py`:

```python
CONTRACT_CONFIG = {
    "name": "send",
    "id": 1,
    "version": 1,
    "supported_transactions": ["send", "reward"],  # Add "reward"
}
```

## Step 2: Add CheckTx Validation

CheckTx performs stateless validation and returns authorized signers.

```python
# In contract.py - add to check_tx
async def check_tx(self, request: PluginCheckRequest) -> PluginCheckResponse:
    # ... existing fee validation ...

    if request.tx.msg.type_url.endswith("/types.MessageSend"):
        msg = MessageSend()
        msg.ParseFromString(request.tx.msg.value)
        return self._check_message_send(msg)
    elif request.tx.msg.type_url.endswith("/types.MessageReward"):
        msg = MessageReward()
        msg.ParseFromString(request.tx.msg.value)
        return self._check_message_reward(msg)  # Add this
    else:
        raise err_invalid_message_cast()

# Add validation function
def _check_message_reward(self, msg: MessageReward) -> PluginCheckResponse:
    if len(msg.admin_address) != 20:
        raise err_invalid_address()
    if len(msg.recipient_address) != 20:
        raise err_invalid_address()
    if msg.amount == 0:
        raise err_invalid_amount()

    response = PluginCheckResponse()
    response.recipient = msg.recipient_address
    response.authorized_signers.append(msg.admin_address)
    return response
```

## Step 3: Add DeliverTx Execution

DeliverTx reads state, applies the transaction, and writes state changes.

The "reward" transaction mints new tokens to a recipient.

```python
# In contract.py - add to deliver_tx
async def deliver_tx(self, request: PluginDeliverRequest) -> PluginDeliverResponse:
    if request.tx.msg.type_url.endswith("/types.MessageSend"):
        msg = MessageSend()
        msg.ParseFromString(request.tx.msg.value)
        return await self._deliver_message_send(msg, request.tx.fee)
    elif request.tx.msg.type_url.endswith("/types.MessageReward"):
        msg = MessageReward()
        msg.ParseFromString(request.tx.msg.value)
        return await self._deliver_message_reward(msg, request.tx.fee)  # Add this
    else:
        raise err_invalid_message_cast()

# Add execution function
async def _deliver_message_reward(self, msg: MessageReward, fee: int) -> PluginDeliverResponse:
    admin_query_id = random.randint(0, 2**53)
    recipient_query_id = random.randint(0, 2**53)
    fee_query_id = random.randint(0, 2**53)

    admin_key = key_for_account(msg.admin_address)
    recipient_key = key_for_account(msg.recipient_address)
    fee_pool_key = key_for_fee_pool(self.config.chain_id)

    # Read state
    response = await self.plugin.state_read(
        self,
        PluginStateReadRequest(
            keys=[
                PluginKeyRead(query_id=admin_query_id, key=admin_key),
                PluginKeyRead(query_id=recipient_query_id, key=recipient_key),
                PluginKeyRead(query_id=fee_query_id, key=fee_pool_key),
            ]
        ),
    )

    if response.HasField("error"):
        result = PluginDeliverResponse()
        result.error.CopyFrom(response.error)
        return result

    # Parse results
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

    # Unmarshal
    admin = unmarshal(Account, admin_bytes) if admin_bytes else Account()
    recipient = unmarshal(Account, recipient_bytes) if recipient_bytes else Account()
    fee_pool = unmarshal(Pool, fee_pool_bytes) if fee_pool_bytes else Pool()

    # Admin must have enough to pay fee
    if admin.amount < fee:
        raise err_insufficient_funds()

    # Apply changes
    admin.amount -= fee              # Admin pays fee
    recipient.amount += msg.amount   # Recipient gets reward (minted tokens)
    fee_pool.amount += fee

    # Write state
    admin_bytes_new = marshal(admin)
    recipient_bytes_new = marshal(recipient)
    fee_pool_bytes_new = marshal(fee_pool)

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

## Adding New State Keys

If your transaction needs new state storage:

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

## Build

```bash
make dev        # Install with dev dependencies
make proto      # Regenerate protobuf bindings
make test       # Run pytest suite
make validate   # Run format + lint + type-check + tests
```
