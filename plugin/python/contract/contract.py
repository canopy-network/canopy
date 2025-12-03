"""
Contract implementation for Canopy blockchain plugin.

This file contains the base contract implementation that handles the 'send' transaction.
Matches Go's contract/contract.go structure.
"""

import random
import struct
from typing import Optional, Dict, Any, Union, Protocol, TYPE_CHECKING

if TYPE_CHECKING:
    from .plugin import Plugin, Config

# Import proto types
from .proto import (
    PluginCheckRequest,
    PluginCheckResponse,
    PluginDeliverRequest,
    PluginDeliverResponse,
    PluginGenesisRequest,
    PluginGenesisResponse,
    PluginBeginRequest,
    PluginBeginResponse,
    PluginEndRequest,
    PluginEndResponse,
    MessageSend,
    PluginKeyRead,
    PluginStateReadRequest,
    PluginStateWriteRequest,
    PluginSetOp,
    PluginDeleteOp,
    PluginFSMConfig,
    FeeParams,
    Account,
    Pool,
)

from .error import (
    PluginError,
    err_invalid_address,
    err_invalid_amount,
    err_insufficient_funds,
    err_tx_fee_below_state_limit,
    err_invalid_message_cast,
    err_unmarshal,
)


# Plugin configuration (matching Go's ContractConfig)
CONTRACT_CONFIG = {
    "name": "send",
    "id": 1,
    "version": 1,
    "supported_transactions": ["send"],
}


# State key prefixes (matching Go)
ACCOUNT_PREFIX = b"\x01"
POOL_PREFIX = b"\x02"
PARAMS_PREFIX = b"\x07"


# Key generation functions (from keys.py)

def join_len_prefix(*items: Optional[bytes]) -> bytes:
    """Join byte arrays with length prefixes."""
    result = bytearray()
    for item in items:
        if not item:
            continue
        if len(item) > 255:
            raise ValueError(f"Item too long: {len(item)} bytes (max 255)")
        result.append(len(item))
        result.extend(item)
    return bytes(result)


def format_uint64(value: Union[int, str]) -> bytes:
    """Format uint64 as big-endian bytes."""
    if isinstance(value, str):
        value = int(value)
    if not isinstance(value, int) or value < 0 or value >= (1 << 64):
        raise ValueError(f"Invalid uint64 value: {value}")
    return struct.pack('>Q', value)


def key_for_account(address: bytes) -> bytes:
    """Generate state database key for an account."""
    return join_len_prefix(ACCOUNT_PREFIX, address)


def key_for_fee_params() -> bytes:
    """Generate state database key for fee parameters."""
    return join_len_prefix(PARAMS_PREFIX, b"/f/")


def key_for_fee_pool(chain_id: int) -> bytes:
    """Generate state database key for fee pool."""
    return join_len_prefix(POOL_PREFIX, format_uint64(chain_id))


# Proto marshal/unmarshal utilities

def marshal(message: Any) -> bytes:
    """Marshal object to protobuf bytes."""
    try:
        if hasattr(message, 'SerializeToString'):
            return message.SerializeToString()
        raise ValueError("Message does not support serialization")
    except Exception as err:
        raise err_unmarshal(err)


def unmarshal(message_type: Any, data: Optional[bytes]) -> Optional[Any]:
    """Unmarshal bytes to protobuf message."""
    if not data:
        return None
    try:
        if hasattr(message_type, 'FromString'):
            return message_type.FromString(data)
        raise ValueError("Message type does not support deserialization")
    except Exception as err:
        raise err_unmarshal(err)


class Contract:
    """
    Contract defines the smart contract that implements the extended logic of the nested chain.
    Matches Go's Contract struct.
    """

    def __init__(
        self,
        config: Optional["Config"] = None,
        fsm_config: Optional[PluginFSMConfig] = None,
        plugin: Optional["Plugin"] = None,
        fsm_id: Optional[int] = None,
    ):
        self.config = config
        self.fsm_config = fsm_config
        self.plugin = plugin
        self.fsm_id = fsm_id

    def genesis(self, request: PluginGenesisRequest) -> PluginGenesisResponse:
        """Genesis implements logic to import a json file to create the state at height 0."""
        return PluginGenesisResponse()

    def begin_block(self, request: PluginBeginRequest) -> PluginBeginResponse:
        """BeginBlock is code that is executed at the start of applying the block."""
        return PluginBeginResponse()

    async def check_tx(self, request: PluginCheckRequest) -> PluginCheckResponse:
        """CheckTx is code that is executed to statelessly validate a transaction."""
        try:
            if not self.plugin or not self.config:
                raise PluginError(1, "plugin", "plugin or config not initialized")

            # Validate fee - read fee params from state
            resp = await self.plugin.state_read(
                self,
                PluginStateReadRequest(
                    keys=[PluginKeyRead(query_id=random.randint(0, 2**53), key=key_for_fee_params())]
                ),
            )

            if resp.HasField("error"):
                response = PluginCheckResponse()
                response.error.CopyFrom(resp.error)
                return response

            # Convert bytes into fee parameters
            if not resp.results or not resp.results[0].entries:
                raise PluginError(1, "plugin", "Fee parameters not found")

            fee_params_bytes = resp.results[0].entries[0].value
            min_fees = unmarshal(FeeParams, fee_params_bytes)
            if not min_fees:
                raise PluginError(1, "plugin", "Failed to decode fee parameters")

            # Check for minimum fee
            if request.tx.fee < min_fees.send_fee:
                raise err_tx_fee_below_state_limit()

            # Get the message
            if request.tx.msg.type_url.endswith("/types.MessageSend"):
                msg = MessageSend()
                msg.ParseFromString(request.tx.msg.value)
                return self._check_message_send(msg)
            else:
                raise err_invalid_message_cast()

        except PluginError as e:
            response = PluginCheckResponse()
            response.error.code = e.code
            response.error.module = e.module
            response.error.msg = e.msg
            return response
        except Exception as err:
            response = PluginCheckResponse()
            response.error.code = 1
            response.error.module = "plugin"
            response.error.msg = str(err)
            return response

    async def deliver_tx(self, request: PluginDeliverRequest) -> PluginDeliverResponse:
        """DeliverTx is code that is executed to apply a transaction."""
        try:
            # Get the message
            if request.tx.msg.type_url.endswith("/types.MessageSend"):
                msg = MessageSend()
                msg.ParseFromString(request.tx.msg.value)
                return await self._deliver_message_send(msg, request.tx.fee)
            else:
                raise err_invalid_message_cast()

        except PluginError as e:
            response = PluginDeliverResponse()
            response.error.code = e.code
            response.error.module = e.module
            response.error.msg = e.msg
            return response
        except Exception as err:
            response = PluginDeliverResponse()
            response.error.code = 1
            response.error.module = "plugin"
            response.error.msg = str(err)
            return response

    def end_block(self, request: PluginEndRequest) -> PluginEndResponse:
        """EndBlock is code that is executed at the end of applying a block."""
        return PluginEndResponse()

    def _check_message_send(self, msg: MessageSend) -> PluginCheckResponse:
        """CheckMessageSend statelessly validates a 'send' message."""
        # Check sender address (must be exactly 20 bytes)
        if len(msg.from_address) != 20:
            raise err_invalid_address()

        # Check recipient address (must be exactly 20 bytes)
        if len(msg.to_address) != 20:
            raise err_invalid_address()

        # Check amount (must be greater than 0)
        if msg.amount == 0:
            raise err_invalid_amount()

        # Return authorized signers (sender must sign)
        response = PluginCheckResponse()
        response.recipient = msg.to_address
        response.authorized_signers.append(msg.from_address)
        return response

    async def _deliver_message_send(self, msg: MessageSend, fee: int) -> PluginDeliverResponse:
        """DeliverMessageSend handles a 'send' message."""
        if not self.plugin or not self.config:
            raise PluginError(1, "plugin", "plugin or config not initialized")

        # Generate query IDs
        from_query_id = random.randint(0, 2**53)
        to_query_id = random.randint(0, 2**53)
        fee_query_id = random.randint(0, 2**53)

        # Calculate keys
        from_key = key_for_account(msg.from_address)
        to_key = key_for_account(msg.to_address)
        fee_pool_key = key_for_fee_pool(self.config.chain_id)

        # Get the from and to accounts
        response = await self.plugin.state_read(
            self,
            PluginStateReadRequest(
                keys=[
                    PluginKeyRead(query_id=fee_query_id, key=fee_pool_key),
                    PluginKeyRead(query_id=from_query_id, key=from_key),
                    PluginKeyRead(query_id=to_query_id, key=to_key),
                ]
            ),
        )

        # Check for internal error
        if response.HasField("error"):
            result = PluginDeliverResponse()
            result.error.CopyFrom(response.error)
            return result

        # Get the from bytes and to bytes
        from_bytes = None
        to_bytes = None
        fee_pool_bytes = None

        for resp in response.results:
            if resp.query_id == from_query_id:
                from_bytes = resp.entries[0].value if resp.entries else None
            elif resp.query_id == to_query_id:
                to_bytes = resp.entries[0].value if resp.entries else None
            elif resp.query_id == fee_query_id:
                fee_pool_bytes = resp.entries[0].value if resp.entries else None

        # Add fee to amount to deduct
        amount_to_deduct = msg.amount + fee

        # Convert bytes to account structures
        from_account = unmarshal(Account, from_bytes) if from_bytes else Account()
        to_account = unmarshal(Account, to_bytes) if to_bytes else Account()
        fee_pool = unmarshal(Pool, fee_pool_bytes) if fee_pool_bytes else Pool()

        # Check sufficient funds
        if from_account.amount < amount_to_deduct:
            raise err_insufficient_funds()

        # For self-transfer, use same account data
        if from_key == to_key:
            to_account = from_account

        # Subtract from sender
        from_account.amount -= amount_to_deduct

        # Add the fee to the fee pool
        fee_pool.amount += fee

        # Add to recipient
        to_account.amount += msg.amount

        # Convert accounts to bytes
        from_bytes_new = marshal(from_account)
        to_bytes_new = marshal(to_account)
        fee_pool_bytes_new = marshal(fee_pool)

        # Execute writes to database
        if from_account.amount == 0:
            # If sender account is drained, delete it
            write_resp = await self.plugin.state_write(
                self,
                PluginStateWriteRequest(
                    sets=[
                        PluginSetOp(key=fee_pool_key, value=fee_pool_bytes_new),
                        PluginSetOp(key=to_key, value=to_bytes_new),
                    ],
                    deletes=[PluginDeleteOp(key=from_key)],
                ),
            )
        else:
            write_resp = await self.plugin.state_write(
                self,
                PluginStateWriteRequest(
                    sets=[
                        PluginSetOp(key=fee_pool_key, value=fee_pool_bytes_new),
                        PluginSetOp(key=to_key, value=to_bytes_new),
                        PluginSetOp(key=from_key, value=from_bytes_new),
                    ],
                ),
            )

        result = PluginDeliverResponse()
        if write_resp.HasField("error"):
            result.error.CopyFrom(write_resp.error)
        return result
