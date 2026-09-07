"""
Contract implementation for Canopy blockchain plugin.

This file contains the base contract implementation that handles the 'send' transaction.
Matches Go's contract/contract.go structure.
"""

import json
import random
import struct
from typing import Optional, Dict, Any, Union, Protocol, TYPE_CHECKING

UINT64_MAX = (1 << 64) - 1

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
    MessagePredict,
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
from .proto import account_pb2, event_pb2, plugin_pb2, tx_pb2
from google.protobuf import any_pb2
from .ai_model import get_model

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
    "name": "python_plugin_contract",
    "id": 1,
    "version": 1,
    "supported_transactions": ["send", "predict"],
    "transaction_type_urls": [
        "type.googleapis.com/types.MessageSend",
        "type.googleapis.com/types.MessagePredict",
    ],
    "event_type_urls": [],
    # Include google/protobuf/any.proto first as it's a dependency of event.proto and tx.proto
    "file_descriptor_protos": [
        any_pb2.DESCRIPTOR.serialized_pb,
        account_pb2.DESCRIPTOR.serialized_pb,
        event_pb2.DESCRIPTOR.serialized_pb,
        plugin_pb2.DESCRIPTOR.serialized_pb,
        tx_pb2.DESCRIPTOR.serialized_pb,
    ],
}


# State key prefixes (matching Go)
ACCOUNT_PREFIX = b"\x01"
POOL_PREFIX = b"\x02"
# Predict results (AI/ML inference output from G2 ZeroPerceptron)
PREDICT_PREFIX = b"\x03"
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


def key_for_predict_log(address: bytes, seq: int) -> bytes:
    """Generate state database key for an AI prediction result.

    Namespace: 0x03 + address + sequence (big-endian)."""
    return join_len_prefix(PREDICT_PREFIX, address, format_uint64(seq))


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

            # Get the message and handle by type
            type_url = request.tx.msg.type_url

            # Check for minimum fee (predict uses its own fee threshold)
            if type_url.endswith("/types.MessagePredict"):
                if request.tx.fee < min_fees.predict_fee:
                    raise err_tx_fee_below_state_limit()
            elif request.tx.fee < min_fees.send_fee:
                raise err_tx_fee_below_state_limit()

            if type_url.endswith("/types.MessageSend"):
                msg = MessageSend()
                msg.ParseFromString(request.tx.msg.value)
                return self._check_message_send(msg)
            elif type_url.endswith("/types.MessagePredict"):
                msg = MessagePredict()
                msg.ParseFromString(request.tx.msg.value)
                return self._check_message_predict(msg)
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
            # Get the message and handle by type
            type_url = request.tx.msg.type_url
            if type_url.endswith("/types.MessageSend"):
                msg = MessageSend()
                msg.ParseFromString(request.tx.msg.value)
                return await self._deliver_message_send(msg, request.tx.fee, request.tx.memo)
            elif type_url.endswith("/types.MessagePredict"):
                msg = MessagePredict()
                msg.ParseFromString(request.tx.msg.value)
                return await self._deliver_message_predict(msg, request.tx.fee, request.tx.memo)
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

    async def _deliver_message_send(self, msg: MessageSend, fee: int, memo: str) -> PluginDeliverResponse:
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

        if msg.amount > UINT64_MAX - fee:
            raise err_invalid_amount()

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

        if fee_pool.amount > UINT64_MAX - fee or (
            from_key != to_key and to_account.amount > UINT64_MAX - msg.amount
        ):
            raise err_invalid_amount()

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

        # Retain drained accounts only when they carry nonce state or core will advance the nonce after RLP.V2 delivery.
        sets = [
            PluginSetOp(key=fee_pool_key, value=fee_pool_bytes_new),
            PluginSetOp(key=to_key, value=to_bytes_new),
        ]
        deletes = []
        if from_account.amount == 0 and from_account.nonce == 0 and memo != "RLP.V2":
            deletes.append(PluginDeleteOp(key=from_key))
        else:
            sets.append(PluginSetOp(key=from_key, value=from_bytes_new))
        write_resp = await self.plugin.state_write(
            self,
            PluginStateWriteRequest(
                sets=sets,
                deletes=deletes,
            ),
        )

        result = PluginDeliverResponse()
        if write_resp.HasField("error"):
            result.error.CopyFrom(write_resp.error)
        return result

    def _check_message_predict(self, msg: MessagePredict) -> PluginCheckResponse:
        """CheckMessagePredict statelessly validates a 'predict' message.

        The message carries multimodal features (text / numbers / timestamp)
        that the G2 ZeroPerceptron model consumes on-chain."""
        # Check sender address (must be exactly 20 bytes)
        if len(msg.from_address) != 20:
            raise err_invalid_address()

        # Features may be empty (pure text-event) but never negative:
        # model normalizes it. We only reject absurd sizes to keep
        # transaction size bounded.
        if len(msg.features) > 256:
            raise PluginError(1, "plugin", "features too large")

        # Return authorized signers (sender must sign)
        response = PluginCheckResponse()
        response.authorized_signers.append(msg.from_address)
        return response

    async def _deliver_message_predict(self, msg: MessagePredict, fee: int, memo: str) -> PluginDeliverResponse:
        """DeliverMessagePredict runs on-chain AI inference.

        Features from the message are passed through the G2 28D encoder
        (text/numbers/timestamp modalities with octonion cross7 fusion)
        and the ZeroPerceptron backbone produces a 16-class decision.

        The result (class + confidence) is written into state under
        PREDICT_PREFIX (b'\\x03') keyed by sender address + nonce.
        """
        if not self.plugin or not self.config:
            raise PluginError(1, "plugin", "plugin or config not initialized")

        # Read sender account (for fee deduction + nonce)
        from_key = key_for_account(msg.from_address)
        fee_pool_key = key_for_fee_pool(self.config.chain_id)
        from_query_id = random.randint(0, 2**53)
        fee_query_id = random.randint(0, 2**53)

        response = await self.plugin.state_read(
            self,
            PluginStateReadRequest(
                keys=[
                    PluginKeyRead(query_id=fee_query_id, key=fee_pool_key),
                    PluginKeyRead(query_id=from_query_id, key=from_key),
                ]
            ),
        )
        if response.HasField("error"):
            result = PluginDeliverResponse()
            result.error.CopyFrom(response.error)
            return result

        from_bytes = None
        fee_pool_bytes = None
        for resp in response.results:
            if resp.query_id == from_query_id:
                from_bytes = resp.entries[0].value if resp.entries else None
            elif resp.query_id == fee_query_id:
                fee_pool_bytes = resp.entries[0].value if resp.entries else None

        from_account = unmarshal(Account, from_bytes) if from_bytes else Account()
        fee_pool = unmarshal(Pool, fee_pool_bytes) if fee_pool_bytes else Pool()

        # Must be able to pay the predict fee
        if from_account.amount < fee:
            raise err_insufficient_funds()

        # Run the G2 ZeroPerceptron inference.
        # MessagePredict carries only numeric features; a 28-D vector is
        # consumed directly (raw mode). Shorter vectors (7-D numeric or
        # empty) are encoded via the G2 multimodal numeric encoder path.
        model = get_model()
        model.reset()
        features = list(msg.features)
        if len(features) == 28:
            result = model.predict_raw(features)
        else:
            result = model.predict_from_event(
                text="",
                numbers=features,
                timestamp=0.0,
            )

        # Deduct fee -> pool, keep nonce
        from_account.amount -= fee
        fee_pool.amount += fee

        # Persist prediction log under 0x03 namespace
        seq = from_account.nonce
        predict_key = key_for_predict_log(msg.from_address, seq)
        log_value = json.dumps({
            "y": result["y"],
            "class": result.get("class", ""),
            "top_probs": result.get("probs", [])[:3],
            "top3": result.get("top3", []),
            "feature_importance": result.get("feature_importance", []),
            "temperature": result.get("temperature", 1.5),
            "height": 0,
        }).encode("utf-8")

        sets = [
            PluginSetOp(key=from_key, value=marshal(from_account)),
            PluginSetOp(key=fee_pool_key, value=marshal(fee_pool)),
            PluginSetOp(key=predict_key, value=log_value),
        ]
        write_resp = await self.plugin.state_write(
            self,
            PluginStateWriteRequest(sets=sets, deletes=[]),
        )

        result = PluginDeliverResponse()
        result.events.extend([])
        if write_resp.HasField("error"):
            result.error.CopyFrom(write_resp.error)
        return result
