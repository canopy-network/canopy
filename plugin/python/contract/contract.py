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
    MessageFeedback,
    MessageStake,
    MessageCreateMarket,
    MessageResolveMarket,
    MessageClaimReward,
    MessageRegisterModel,
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
    "supported_transactions": ["send", "predict", "feedback", "stake", "create_market", "resolve_market", "claim_reward", "register_model"],
    "transaction_type_urls": [
        "type.googleapis.com/types.MessageSend",
        "type.googleapis.com/types.MessagePredict",
        "type.googleapis.com/types.MessageFeedback",
        "type.googleapis.com/types.MessageStake",
        "type.googleapis.com/types.MessageCreateMarket",
        "type.googleapis.com/types.MessageResolveMarket",
        "type.googleapis.com/types.MessageClaimReward",
        "type.googleapis.com/types.MessageRegisterModel",
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
# Feedback log (on-chain learning signals: correct/incorrect predictions)
FEEDBACK_PREFIX = b"\x04"
# Prediction markets (QARD staking pools)
MARKET_PREFIX = b"\x05"
# Stakes (user positions in prediction markets)
STAKE_PREFIX = b"\x06"
PARAMS_PREFIX = b"\x07"
# Market ID counter (monotonic sequence for market creation)
MARKET_COUNTER_PREFIX = b"\x08"
# Model versioning registry (G2 ZeroPerceptron model versions)
MODEL_PREFIX = b"\x09"
# Model version counter (monotonic sequence for model registration)
MODEL_COUNTER_PREFIX = b"\x0a"


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


def key_for_feedback_log(address: bytes, seq: int) -> bytes:
    """Generate state database key for a feedback entry.



    Namespace: 0x04 + address + sequence (big-endian)."""
    return join_len_prefix(FEEDBACK_PREFIX, address, format_uint64(seq))


def key_for_market_counter() -> bytes:
    """Generate state database key for the market ID counter.

    Namespace: 0x08 (single global counter)."""
    return join_len_prefix(MARKET_COUNTER_PREFIX, b"/c/")


def key_for_market(market_id: int) -> bytes:
    """Generate state database key for a prediction market.

    Namespace: 0x05 + market_id (big-endian)."""
    return join_len_prefix(MARKET_PREFIX, format_uint64(market_id))


def key_for_stake(address: bytes, market_id: int) -> bytes:
    """Generate state database key for a user's stake in a market.

    Namespace: 0x06 + address + market_id (big-endian)."""
    return join_len_prefix(STAKE_PREFIX, address, format_uint64(market_id))


def key_for_model_counter() -> bytes:
    """Generate state database key for the model version counter.

    Namespace: 0x0a (single global counter)."""
    return join_len_prefix(MODEL_COUNTER_PREFIX, b"/c/")


def key_for_model_registry(version: int) -> bytes:
    """Generate state database key for a model version record.

    Namespace: 0x09 + version (big-endian)."""
    return join_len_prefix(MODEL_PREFIX, format_uint64(version))


def key_for_active_model() -> bytes:
    """Generate state database key for the active model version.

    Namespace: 0x09 + '/active/' (single global pointer)."""
    return join_len_prefix(MODEL_PREFIX, b"/active/")


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
            elif type_url.endswith("/types.MessageFeedback"):
                msg = MessageFeedback()
                msg.ParseFromString(request.tx.msg.value)
                return self._check_message_feedback(msg)
            elif type_url.endswith("/types.MessageStake"):
                msg = MessageStake()
                msg.ParseFromString(request.tx.msg.value)
                return self._check_message_stake(msg)
            elif type_url.endswith("/types.MessageCreateMarket"):
                msg = MessageCreateMarket()
                msg.ParseFromString(request.tx.msg.value)
                return self._check_message_create_market(msg)
            elif type_url.endswith("/types.MessageResolveMarket"):
                msg = MessageResolveMarket()
                msg.ParseFromString(request.tx.msg.value)
                return self._check_message_resolve_market(msg)
            elif type_url.endswith("/types.MessageClaimReward"):
                msg = MessageClaimReward()
                msg.ParseFromString(request.tx.msg.value)
                return self._check_message_claim_reward(msg)
            elif type_url.endswith("/types.MessageRegisterModel"):
                msg = MessageRegisterModel()
                msg.ParseFromString(request.tx.msg.value)
                return self._check_message_register_model(msg)
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
            elif type_url.endswith("/types.MessageFeedback"):
                msg = MessageFeedback()
                msg.ParseFromString(request.tx.msg.value)
                return await self._deliver_message_feedback(msg, request.tx.fee, request.tx.memo)
            elif type_url.endswith("/types.MessageStake"):
                msg = MessageStake()
                msg.ParseFromString(request.tx.msg.value)
                return await self._deliver_message_stake(msg, request.tx.fee, request.tx.memo)
            elif type_url.endswith("/types.MessageCreateMarket"):
                msg = MessageCreateMarket()
                msg.ParseFromString(request.tx.msg.value)
                return await self._deliver_message_create_market(msg, request.tx.fee, request.tx.memo)
            elif type_url.endswith("/types.MessageResolveMarket"):
                msg = MessageResolveMarket()
                msg.ParseFromString(request.tx.msg.value)
                return await self._deliver_message_resolve_market(msg, request.tx.fee, request.tx.memo)
            elif type_url.endswith("/types.MessageClaimReward"):
                msg = MessageClaimReward()
                msg.ParseFromString(request.tx.msg.value)
                return await self._deliver_message_claim_reward(msg, request.tx.fee, request.tx.memo)
            elif type_url.endswith("/types.MessageRegisterModel"):
                msg = MessageRegisterModel()
                msg.ParseFromString(request.tx.msg.value)
                return await self._deliver_message_register_model(msg, request.tx.fee, request.tx.memo)
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

        Features from the message are passed through the G2 42D encoder
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
        # MessagePredict carries only numeric features; a 42-D vector is
        # consumed directly (raw mode). Shorter vectors (7-D numeric or
        # empty) are encoded via the G2 multimodal numeric encoder path.
        model = get_model()
        model.reset()
        features = list(msg.features)
        if len(features) == 42:
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

    def _check_message_stake(self, msg: MessageStake) -> PluginCheckResponse:
        """CheckMessageStake statelessly validates a 'stake' message.

        Users lock QARD into a prediction market pool to earn rewards
        from correct predictions."""
        # Check sender address (must be exactly 20 bytes)
        if len(msg.from_address) != 20:
            raise err_invalid_address()

        # Check amount (must be greater than 0)
        if msg.amount == 0:
            raise err_invalid_amount()

        # Outcome must be a valid class (0-15)
        if msg.outcome > 15:
            raise PluginError(1, "plugin", "invalid outcome class")

        # Return authorized signers (sender must sign)
        response = PluginCheckResponse()
        response.authorized_signers.append(msg.from_address)
        return response

    async def _deliver_message_stake(self, msg: MessageStake, fee: int, memo: str) -> PluginDeliverResponse:
        """DeliverMessageStake locks QARD into a prediction market.

        The stake is recorded under STAKE_PREFIX (b'\\x06') keyed by
        sender address + market_id. The market's total pool is updated
        under MARKET_PREFIX (b'\\x05')."""
        if not self.plugin or not self.config:
            raise PluginError(1, "plugin", "plugin or config not initialized")

        # Read sender account, fee pool, market, and existing stake
        from_key = key_for_account(msg.from_address)
        fee_pool_key = key_for_fee_pool(self.config.chain_id)
        market_key = key_for_market(msg.market_id)
        stake_key = key_for_stake(msg.from_address, msg.market_id)

        from_query_id = random.randint(0, 2**53)
        fee_query_id = random.randint(0, 2**53)
        market_query_id = random.randint(0, 2**53)
        stake_query_id = random.randint(0, 2**53)

        response = await self.plugin.state_read(
            self,
            PluginStateReadRequest(
                keys=[
                    PluginKeyRead(query_id=fee_query_id, key=fee_pool_key),
                    PluginKeyRead(query_id=from_query_id, key=from_key),
                    PluginKeyRead(query_id=market_query_id, key=market_key),
                    PluginKeyRead(query_id=stake_query_id, key=stake_key),
                ]
            ),
        )
        if response.HasField("error"):
            result = PluginDeliverResponse()
            result.error.CopyFrom(response.error)
            return result

        from_bytes = None
        fee_pool_bytes = None
        market_bytes = None
        stake_bytes = None
        for resp in response.results:
            if resp.query_id == from_query_id:
                from_bytes = resp.entries[0].value if resp.entries else None
            elif resp.query_id == fee_query_id:
                fee_pool_bytes = resp.entries[0].value if resp.entries else None
            elif resp.query_id == market_query_id:
                market_bytes = resp.entries[0].value if resp.entries else None
            elif resp.query_id == stake_query_id:
                stake_bytes = resp.entries[0].value if resp.entries else None

        from_account = unmarshal(Account, from_bytes) if from_bytes else Account()
        fee_pool = unmarshal(Pool, fee_pool_bytes) if fee_pool_bytes else Pool()
        market = json.loads(market_bytes.decode("utf-8")) if market_bytes else None
        stake = json.loads(stake_bytes.decode("utf-8")) if stake_bytes else None

        # Market must exist
        if market is None:
            raise PluginError(1, "plugin", "market not found")

        # Market must be open (not resolved)
        if market.get("resolved", False):
            raise PluginError(1, "plugin", "market already resolved")

        # Must be able to pay fee + stake
        if from_account.amount < fee + msg.amount:
            raise err_insufficient_funds()

        # Deduct fee + stake from sender
        from_account.amount -= fee + msg.amount
        fee_pool.amount += fee

        # Update market pool (JSON keys are strings after serialization)
        market["total_pool"] = market.get("total_pool", 0) + msg.amount
        outcome_key = str(msg.outcome)
        market["outcome_pools"][outcome_key] = market["outcome_pools"].get(outcome_key, 0) + msg.amount

        # Update or create stake record
        if stake is None:
            stake = {
                "market_id": int(msg.market_id),
                "outcome": int(msg.outcome),
                "amount": int(msg.amount),
                "claimed": False,
            }
        else:
            stake["amount"] = stake.get("amount", 0) + msg.amount
            stake["outcome"] = int(msg.outcome)

        sets = [
            PluginSetOp(key=from_key, value=marshal(from_account)),
            PluginSetOp(key=fee_pool_key, value=marshal(fee_pool)),
            PluginSetOp(key=market_key, value=json.dumps(market).encode("utf-8")),
            PluginSetOp(key=stake_key, value=json.dumps(stake).encode("utf-8")),
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

    def _check_message_create_market(self, msg: MessageCreateMarket) -> PluginCheckResponse:
        """CheckMessageCreateMarket statelessly validates a 'create_market' message.

        Users create a new prediction market for the G2 model to predict on."""
        # Check sender address (must be exactly 20 bytes)
        if len(msg.from_address) != 20:
            raise err_invalid_address()

        # Question must not be empty
        if not msg.question:
            raise PluginError(1, "plugin", "question cannot be empty")

        # Resolution height must be in the future
        if msg.resolution_height == 0:
            raise PluginError(1, "plugin", "resolution height must be > 0")

        # Initial stake must be > 0
        if msg.stake == 0:
            raise err_invalid_amount()

        # Return authorized signers (sender must sign)
        response = PluginCheckResponse()
        response.authorized_signers.append(msg.from_address)
        return response

    async def _deliver_message_create_market(self, msg: MessageCreateMarket, fee: int, memo: str) -> PluginDeliverResponse:
        """DeliverMessageCreateMarket creates a new prediction market.

        A new market ID is allocated from the monotonic counter (0x08),
        and the market record is stored under MARKET_PREFIX (b'\\x05')."""
        if not self.plugin or not self.config:
            raise PluginError(1, "plugin", "plugin or config not initialized")

        # Read sender account, fee pool, and market counter
        from_key = key_for_account(msg.from_address)
        fee_pool_key = key_for_fee_pool(self.config.chain_id)
        counter_key = key_for_market_counter()

        from_query_id = random.randint(0, 2**53)
        fee_query_id = random.randint(0, 2**53)
        counter_query_id = random.randint(0, 2**53)

        response = await self.plugin.state_read(
            self,
            PluginStateReadRequest(
                keys=[
                    PluginKeyRead(query_id=fee_query_id, key=fee_pool_key),
                    PluginKeyRead(query_id=from_query_id, key=from_key),
                    PluginKeyRead(query_id=counter_query_id, key=counter_key),
                ]
            ),
        )
        if response.HasField("error"):
            result = PluginDeliverResponse()
            result.error.CopyFrom(response.error)
            return result

        from_bytes = None
        fee_pool_bytes = None
        counter_bytes = None
        for resp in response.results:
            if resp.query_id == from_query_id:
                from_bytes = resp.entries[0].value if resp.entries else None
            elif resp.query_id == fee_query_id:
                fee_pool_bytes = resp.entries[0].value if resp.entries else None
            elif resp.query_id == counter_query_id:
                counter_bytes = resp.entries[0].value if resp.entries else None

        from_account = unmarshal(Account, from_bytes) if from_bytes else Account()
        fee_pool = unmarshal(Pool, fee_pool_bytes) if fee_pool_bytes else Pool()

        # Must be able to pay fee + initial stake
        if from_account.amount < fee + msg.stake:
            raise err_insufficient_funds()

        # Allocate new market ID
        market_id = int.from_bytes(counter_bytes, "big") if counter_bytes else 0
        new_market_id = market_id + 1

        # Deduct fee + stake from sender
        from_account.amount -= fee + msg.stake
        fee_pool.amount += fee

        # Create market record
        market = {
            "id": new_market_id,
            "creator": msg.from_address.hex(),
            "question": msg.question,
            "resolution_height": int(msg.resolution_height),
            "total_pool": int(msg.stake),
            "outcome_pools": {},
            "resolved": False,
            "actual_class": None,
            "resolver": None,
        }

        market_key = key_for_market(new_market_id)
        sets = [
            PluginSetOp(key=from_key, value=marshal(from_account)),
            PluginSetOp(key=fee_pool_key, value=marshal(fee_pool)),
            PluginSetOp(key=counter_key, value=new_market_id.to_bytes(8, "big")),
            PluginSetOp(key=market_key, value=json.dumps(market).encode("utf-8")),
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

    def _check_message_resolve_market(self, msg: MessageResolveMarket) -> PluginCheckResponse:
        """CheckMessageResolveMarket statelessly validates a 'resolve_market' message.

        The market creator or designated resolver settles the market based
        on the actual outcome, distributing rewards to correct stakers."""
        # Check sender address (must be exactly 20 bytes)
        if len(msg.from_address) != 20:
            raise err_invalid_address()

        # Actual class must be valid (0-15)
        if msg.actual_class > 15:
            raise PluginError(1, "plugin", "invalid actual class")

        # Return authorized signers (sender must sign)
        response = PluginCheckResponse()
        response.authorized_signers.append(msg.from_address)
        return response

    async def _deliver_message_resolve_market(self, msg: MessageResolveMarket, fee: int, memo: str) -> PluginDeliverResponse:
        """DeliverMessageResolveMarket resolves a prediction market.

        The market is marked as resolved with the actual outcome class.
        Rewards are distributed proportionally to stakers who predicted
        the correct outcome."""
        if not self.plugin or not self.config:
            raise PluginError(1, "plugin", "plugin or config not initialized")

        # Read sender account, fee pool, and market
        from_key = key_for_account(msg.from_address)
        fee_pool_key = key_for_fee_pool(self.config.chain_id)
        market_key = key_for_market(msg.market_id)

        from_query_id = random.randint(0, 2**53)
        fee_query_id = random.randint(0, 2**53)
        market_query_id = random.randint(0, 2**53)

        response = await self.plugin.state_read(
            self,
            PluginStateReadRequest(
                keys=[
                    PluginKeyRead(query_id=fee_query_id, key=fee_pool_key),
                    PluginKeyRead(query_id=from_query_id, key=from_key),
                    PluginKeyRead(query_id=market_query_id, key=market_key),
                ]
            ),
        )
        if response.HasField("error"):
            result = PluginDeliverResponse()
            result.error.CopyFrom(response.error)
            return result

        from_bytes = None
        fee_pool_bytes = None
        market_bytes = None
        for resp in response.results:
            if resp.query_id == from_query_id:
                from_bytes = resp.entries[0].value if resp.entries else None
            elif resp.query_id == fee_query_id:
                fee_pool_bytes = resp.entries[0].value if resp.entries else None
            elif resp.query_id == market_query_id:
                market_bytes = resp.entries[0].value if resp.entries else None

        from_account = unmarshal(Account, from_bytes) if from_bytes else Account()
        fee_pool = unmarshal(Pool, fee_pool_bytes) if fee_pool_bytes else Pool()
        market = json.loads(market_bytes.decode("utf-8")) if market_bytes else None

        # Market must exist
        if market is None:
            raise PluginError(1, "plugin", "market not found")

        # Market must not already be resolved
        if market.get("resolved", False):
            raise PluginError(1, "plugin", "market already resolved")

        # Only the creator can resolve
        if market.get("creator") != msg.from_address.hex():
            raise PluginError(1, "plugin", "only market creator can resolve")

        # Must be able to pay fee
        if from_account.amount < fee:
            raise err_insufficient_funds()

        # Deduct fee
        from_account.amount -= fee
        fee_pool.amount += fee

        # Resolve market
        market["resolved"] = True
        market["actual_class"] = int(msg.actual_class)
        market["resolver"] = msg.from_address.hex()

        sets = [
            PluginSetOp(key=from_key, value=marshal(from_account)),
            PluginSetOp(key=fee_pool_key, value=marshal(fee_pool)),
            PluginSetOp(key=market_key, value=json.dumps(market).encode("utf-8")),
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

    def _check_message_claim_reward(self, msg: MessageClaimReward) -> PluginCheckResponse:
        """CheckMessageClaimReward statelessly validates a 'claim_reward' message.

        Stakers who predicted correctly can claim their share of the reward pool."""
        # Check sender address (must be exactly 20 bytes)
        if len(msg.from_address) != 20:
            raise err_invalid_address()

        # Return authorized signers (sender must sign)
        response = PluginCheckResponse()
        response.authorized_signers.append(msg.from_address)
        return response

    async def _deliver_message_claim_reward(self, msg: MessageClaimReward, fee: int, memo: str) -> PluginDeliverResponse:
        """DeliverMessageClaimReward pays out rewards to correct stakers.

        The reward is calculated as the staker's share of the winning
        outcome pool, proportional to their stake."""
        if not self.plugin or not self.config:
            raise PluginError(1, "plugin", "plugin or config not initialized")

        # Read sender account, fee pool, market, and stake
        from_key = key_for_account(msg.from_address)
        fee_pool_key = key_for_fee_pool(self.config.chain_id)
        market_key = key_for_market(msg.market_id)
        stake_key = key_for_stake(msg.from_address, msg.market_id)

        from_query_id = random.randint(0, 2**53)
        fee_query_id = random.randint(0, 2**53)
        market_query_id = random.randint(0, 2**53)
        stake_query_id = random.randint(0, 2**53)

        response = await self.plugin.state_read(
            self,
            PluginStateReadRequest(
                keys=[
                    PluginKeyRead(query_id=fee_query_id, key=fee_pool_key),
                    PluginKeyRead(query_id=from_query_id, key=from_key),
                    PluginKeyRead(query_id=market_query_id, key=market_key),
                    PluginKeyRead(query_id=stake_query_id, key=stake_key),
                ]
            ),
        )
        if response.HasField("error"):
            result = PluginDeliverResponse()
            result.error.CopyFrom(response.error)
            return result

        from_bytes = None
        fee_pool_bytes = None
        market_bytes = None
        stake_bytes = None
        for resp in response.results:
            if resp.query_id == from_query_id:
                from_bytes = resp.entries[0].value if resp.entries else None
            elif resp.query_id == fee_query_id:
                fee_pool_bytes = resp.entries[0].value if resp.entries else None
            elif resp.query_id == market_query_id:
                market_bytes = resp.entries[0].value if resp.entries else None
            elif resp.query_id == stake_query_id:
                stake_bytes = resp.entries[0].value if resp.entries else None

        from_account = unmarshal(Account, from_bytes) if from_bytes else Account()
        fee_pool = unmarshal(Pool, fee_pool_bytes) if fee_pool_bytes else Pool()
        market = json.loads(market_bytes.decode("utf-8")) if market_bytes else None
        stake = json.loads(stake_bytes.decode("utf-8")) if stake_bytes else None

        # Market must exist and be resolved
        if market is None:
            raise PluginError(1, "plugin", "market not found")
        if not market.get("resolved", False):
            raise PluginError(1, "plugin", "market not resolved yet")

        # Stake must exist and not be claimed
        if stake is None:
            raise PluginError(1, "plugin", "no stake found")
        if stake.get("claimed", False):
            raise PluginError(1, "plugin", "reward already claimed")

        # Must be able to pay fee
        if from_account.amount < fee:
            raise err_insufficient_funds()

        # Only correct stakers get rewards
        actual_class = market.get("actual_class")
        if stake.get("outcome") != actual_class:
            raise PluginError(1, "plugin", "stake was not correct")

        # Calculate reward: share of winning pool proportional to stake
        # (JSON keys are strings after serialization)
        winning_pool = market.get("outcome_pools", {}).get(str(actual_class), 0)
        if winning_pool <= 0:
            raise PluginError(1, "plugin", "winning pool is empty")

        stake_amount = stake.get("amount", 0)
        reward = (stake_amount * market.get("total_pool", 0)) // winning_pool

        # Deduct fee, add reward
        from_account.amount -= fee
        from_account.amount += reward
        fee_pool.amount += fee

        # Mark stake as claimed
        stake["claimed"] = True

        sets = [
            PluginSetOp(key=from_key, value=marshal(from_account)),
            PluginSetOp(key=fee_pool_key, value=marshal(fee_pool)),
            PluginSetOp(key=stake_key, value=json.dumps(stake).encode("utf-8")),
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

    def _check_message_register_model(self, msg: MessageRegisterModel) -> PluginCheckResponse:
        """CheckMessageRegisterModel statelessly validates a 'register_model' message.

        Governance registers a new G2 ZeroPerceptron model version with
        its weights hash, accuracy, and input dimension."""
        # Check sender address (must be exactly 20 bytes)
        if len(msg.from_address) != 20:
            raise err_invalid_address()

        # Version must be > 0
        if msg.version == 0:
            raise PluginError(1, "plugin", "version must be > 0")

        # Weights hash must not be empty
        if not msg.weights_hash:
            raise PluginError(1, "plugin", "weights hash cannot be empty")

        # Accuracy must be in [0, 1]
        if msg.accuracy < 0.0 or msg.accuracy > 1.0:
            raise PluginError(1, "plugin", "accuracy must be in [0, 1]")

        # Input dimension must be valid (28 or 42)
        if msg.in_dim not in (28, 42):
            raise PluginError(1, "plugin", "in_dim must be 28 or 42")

        # Number of classes must be > 0
        if msg.n_classes == 0:
            raise PluginError(1, "plugin", "n_classes must be > 0")

        # Return authorized signers (sender must sign)
        response = PluginCheckResponse()
        response.authorized_signers.append(msg.from_address)
        return response

    async def _deliver_message_register_model(self, msg: MessageRegisterModel, fee: int, memo: str) -> PluginDeliverResponse:
        """DeliverMessageRegisterModel registers a new model version.

        The model record (version, weights hash, accuracy, input dim,
        class count, description) is stored under MODEL_PREFIX (b'\\x09')
        keyed by version. The active model pointer is updated to the
        highest registered version."""
        if not self.plugin or not self.config:
            raise PluginError(1, "plugin", "plugin or config not initialized")

        # Read sender account, fee pool, model counter, and active model
        from_key = key_for_account(msg.from_address)
        fee_pool_key = key_for_fee_pool(self.config.chain_id)
        counter_key = key_for_model_counter()
        active_key = key_for_active_model()

        from_query_id = random.randint(0, 2**53)
        fee_query_id = random.randint(0, 2**53)
        counter_query_id = random.randint(0, 2**53)
        active_query_id = random.randint(0, 2**53)

        response = await self.plugin.state_read(
            self,
            PluginStateReadRequest(
                keys=[
                    PluginKeyRead(query_id=fee_query_id, key=fee_pool_key),
                    PluginKeyRead(query_id=from_query_id, key=from_key),
                    PluginKeyRead(query_id=counter_query_id, key=counter_key),
                    PluginKeyRead(query_id=active_query_id, key=active_key),
                ]
            ),
        )
        if response.HasField("error"):
            result = PluginDeliverResponse()
            result.error.CopyFrom(response.error)
            return result

        from_bytes = None
        fee_pool_bytes = None
        counter_bytes = None
        active_bytes = None
        for resp in response.results:
            if resp.query_id == from_query_id:
                from_bytes = resp.entries[0].value if resp.entries else None
            elif resp.query_id == fee_query_id:
                fee_pool_bytes = resp.entries[0].value if resp.entries else None
            elif resp.query_id == counter_query_id:
                counter_bytes = resp.entries[0].value if resp.entries else None
            elif resp.query_id == active_query_id:
                active_bytes = resp.entries[0].value if resp.entries else None

        from_account = unmarshal(Account, from_bytes) if from_bytes else Account()
        fee_pool = unmarshal(Pool, fee_pool_bytes) if fee_pool_bytes else Pool()

        # Must be able to pay fee
        if from_account.amount < fee:
            raise err_insufficient_funds()

        # Allocate new model version from counter
        model_version = int.from_bytes(counter_bytes, "big") if counter_bytes else 0
        new_version = model_version + 1

        # Deduct fee -> pool
        from_account.amount -= fee
        fee_pool.amount += fee

        # Create model registry record
        model_record = {
            "version": new_version,
            "weights_hash": msg.weights_hash,
            "accuracy": float(msg.accuracy),
            "in_dim": int(msg.in_dim),
            "n_classes": int(msg.n_classes),
            "description": msg.description,
            "registered_by": msg.from_address.hex(),
            "height": 0,
        }

        model_key = key_for_model_registry(new_version)
        sets = [
            PluginSetOp(key=from_key, value=marshal(from_account)),
            PluginSetOp(key=fee_pool_key, value=marshal(fee_pool)),
            PluginSetOp(key=counter_key, value=new_version.to_bytes(8, "big")),
            PluginSetOp(key=model_key, value=json.dumps(model_record).encode("utf-8")),
            # Update active model pointer to the latest version
            PluginSetOp(key=active_key, value=str(new_version).encode("utf-8")),
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

    def _check_message_feedback(self, msg: MessageFeedback) -> PluginCheckResponse:
        """CheckMessageFeedback statelessly validates a 'feedback' message.

        The message carries an on-chain learning signal: whether a previous
        prediction was correct. This enables the G2 model to improve over time."""
        # Check sender address (must be exactly 20 bytes)
        if len(msg.from_address) != 20:
            raise err_invalid_address()

        # Return authorized signers (sender must sign)
        response = PluginCheckResponse()
        response.authorized_signers.append(msg.from_address)

        return response

    async def _deliver_message_feedback(self, msg: MessageFeedback, fee: int, memo: str) -> PluginDeliverResponse:
        """DeliverMessageFeedback stores an on-chain learning signal.

        The feedback (correct/incorrect + actual class) is persisted under
        FEEDBACK_PREFIX (b'\\x04') keyed by sender address + predict_seq.
        This data can be exported off-chain for retraining / governance."""
        if not self.plugin or not self.config:
            raise PluginError(1, "plugin", "plugin or config not initialized")

        # Read sender account (for fee deduction)
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

        # Must be able to pay the feedback fee (same as send fee)
        if from_account.amount < fee:
            raise err_insufficient_funds()

        # Deduct fee -> pool
        from_account.amount -= fee
        fee_pool.amount += fee

        # Persist feedback log under 0x04 namespace
        feedback_key = key_for_feedback_log(msg.from_address, msg.predict_seq)
        feedback_value = json.dumps({
            "predict_seq": int(msg.predict_seq),
            "correct": bool(msg.correct),
            "actual_class": int(msg.actual_class),
            "height": 0,
        }).encode("utf-8")

        sets = [
            PluginSetOp(key=from_key, value=marshal(from_account)),
            PluginSetOp(key=fee_pool_key, value=marshal(fee_pool)),
            PluginSetOp(key=feedback_key, value=feedback_value),
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
