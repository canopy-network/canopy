"""
Unit tests for the Contract class.

Covers the current `contract.contract` API: lifecycle hooks, stateless message
validation for the base 'send' transaction, and the on-chain AI 'predict' transaction.
"""

import asyncio
import json
import unittest

import pytest
from google.protobuf.any_pb2 import Any

from contract.contract import (
    Contract,
    key_for_account,
    key_for_dashboard,
    key_for_fee_params,
)

from contract.contract import Contract
from contract.plugin import Config
from contract.error import PluginError
from contract.proto import (
    MessageSend,
    MessagePredict,
    MessageFeedback,
    MessageStake,
    MessageCreateMarket,
    MessageResolveMarket,
    MessageClaimReward,
    MessageRegisterModel,
    PluginGenesisRequest,
    PluginBeginRequest,
    PluginEndRequest,
    PluginCheckRequest,
    PluginDeliverRequest,
    PluginStateReadRequest,
    PluginStateReadResponse,
    PluginStateWriteRequest,
    PluginStateWriteResponse,
    PluginKeyRead,
    PluginReadResult,
    PluginStateEntry,
    PluginSetOp,
    PluginDeleteOp,
    Transaction,
    Account,
    Pool,
    FeeParams,
)

# Error codes (see contract/error.py)
CODE_INVALID_ADDRESS = 12
CODE_INVALID_AMOUNT = 13

ADDR_A = b"a" * 20
ADDR_B = b"b" * 20
ADDR_SHORT = b"short"


@pytest.fixture
def config():
    """Default plugin configuration."""
    return Config()


@pytest.fixture
def contract(config):
    """Contract instance with config but no live plugin (stateless tests)."""
    return Contract(config=config)


class TestContractLifecycle:
    """Lifecycle hooks should succeed without error."""

    def test_genesis(self, contract):
        result = contract.genesis(PluginGenesisRequest())
        assert not result.HasField("error")

    def test_begin_block(self, contract):
        result = contract.begin_block(PluginBeginRequest())
        assert not result.HasField("error")

    def test_end_block(self, contract):
        result = contract.end_block(PluginEndRequest())
        assert not result.HasField("error")


class TestCheckMessageSend:
    """Stateless validation of the base 'send' message."""

    def test_valid(self, contract):
        msg = MessageSend(from_address=ADDR_A, to_address=ADDR_B, amount=1000)
        result = contract._check_message_send(msg)

        assert not result.HasField("error")
        assert result.recipient == ADDR_B
        assert list(result.authorized_signers) == [ADDR_A]

    def test_invalid_from_address(self, contract):
        msg = MessageSend(from_address=ADDR_SHORT, to_address=ADDR_B, amount=1000)
        with pytest.raises(PluginError) as exc:
            contract._check_message_send(msg)
        assert exc.value.code == CODE_INVALID_ADDRESS

    def test_invalid_to_address(self, contract):
        msg = MessageSend(from_address=ADDR_A, to_address=ADDR_SHORT, amount=1000)
        with pytest.raises(PluginError) as exc:
            contract._check_message_send(msg)
        assert exc.value.code == CODE_INVALID_ADDRESS

    def test_invalid_amount(self, contract):
        msg = MessageSend(from_address=ADDR_A, to_address=ADDR_B, amount=0)
        with pytest.raises(PluginError) as exc:
            contract._check_message_send(msg)
        assert exc.value.code == CODE_INVALID_AMOUNT


class TestCheckMessagePredict:
    """Stateless validation of the on-chain AI 'predict' message."""

    def test_valid_42d(self, contract):
        msg = MessagePredict(from_address=ADDR_A, features=[0.1] * 42)
        result = contract._check_message_predict(msg)

        assert not result.HasField("error")
        assert list(result.authorized_signers) == [ADDR_A]

    def test_valid_7d(self, contract):
        msg = MessagePredict(from_address=ADDR_A, features=[1.0, 2.0, 3.0])
        result = contract._check_message_predict(msg)



        assert not result.HasField("error")
        assert list(result.authorized_signers) == [ADDR_A]

    def test_invalid_from_address(self, contract):
        msg = MessagePredict(from_address=ADDR_SHORT, features=[0.1] * 42)
        with pytest.raises(PluginError) as exc:
            contract._check_message_predict(msg)
        assert exc.value.code == CODE_INVALID_ADDRESS

    def test_features_too_large(self, contract):
        msg = MessagePredict(from_address=ADDR_A, features=[0.0] * 257)
        with pytest.raises(PluginError) as exc:
            contract._check_message_predict(msg)
        assert exc.value.code == 1
        assert "features too large" in exc.value.msg


class TestCheckMessageFeedback:
    """Stateless validation of the on-chain learning 'feedback' message."""

    def test_valid(self, contract):
        msg = MessageFeedback(from_address=ADDR_A, predict_seq=5, correct=True, actual_class=3)
        result = contract._check_message_feedback(msg)

        assert not result.HasField("error")
        assert list(result.authorized_signers) == [ADDR_A]

    def test_invalid_from_address(self, contract):
        msg = MessageFeedback(from_address=ADDR_SHORT, predict_seq=5, correct=True)
        with pytest.raises(PluginError) as exc:
            contract._check_message_feedback(msg)
        assert exc.value.code == CODE_INVALID_ADDRESS


class TestCheckMessageStake:
    """Stateless validation of the prediction market 'stake' message."""

    def test_valid(self, contract):
        msg = MessageStake(from_address=ADDR_A, market_id=1, amount=1000, outcome=3)
        result = contract._check_message_stake(msg)

        assert not result.HasField("error")
        assert list(result.authorized_signers) == [ADDR_A]

    def test_invalid_from_address(self, contract):
        msg = MessageStake(from_address=ADDR_SHORT, market_id=1, amount=1000, outcome=3)
        with pytest.raises(PluginError) as exc:
            contract._check_message_stake(msg)
        assert exc.value.code == CODE_INVALID_ADDRESS

    def test_invalid_amount(self, contract):
        msg = MessageStake(from_address=ADDR_A, market_id=1, amount=0, outcome=3)
        with pytest.raises(PluginError) as exc:
            contract._check_message_stake(msg)
        assert exc.value.code == CODE_INVALID_AMOUNT

    def test_invalid_outcome(self, contract):
        msg = MessageStake(from_address=ADDR_A, market_id=1, amount=1000, outcome=16)
        with pytest.raises(PluginError) as exc:
            contract._check_message_stake(msg)
        assert exc.value.code == 1
        assert "invalid outcome class" in exc.value.msg


class TestCheckMessageCreateMarket:
    """Stateless validation of the prediction market 'create_market' message."""

    def test_valid(self, contract):
        msg = MessageCreateMarket(from_address=ADDR_A, question="BTC up?", resolution_height=1000, stake=5000)
        result = contract._check_message_create_market(msg)

        assert not result.HasField("error")
        assert list(result.authorized_signers) == [ADDR_A]

    def test_invalid_from_address(self, contract):
        msg = MessageCreateMarket(from_address=ADDR_SHORT, question="BTC up?", resolution_height=1000, stake=5000)
        with pytest.raises(PluginError) as exc:
            contract._check_message_create_market(msg)
        assert exc.value.code == CODE_INVALID_ADDRESS

    def test_empty_question(self, contract):
        msg = MessageCreateMarket(from_address=ADDR_A, question="", resolution_height=1000, stake=5000)
        with pytest.raises(PluginError) as exc:
            contract._check_message_create_market(msg)
        assert exc.value.code == 1
        assert "question cannot be empty" in exc.value.msg

    def test_zero_resolution_height(self, contract):
        msg = MessageCreateMarket(from_address=ADDR_A, question="BTC up?", resolution_height=0, stake=5000)
        with pytest.raises(PluginError) as exc:
            contract._check_message_create_market(msg)
        assert exc.value.code == 1
        assert "resolution height must be > 0" in exc.value.msg

    def test_zero_stake(self, contract):
        msg = MessageCreateMarket(from_address=ADDR_A, question="BTC up?", resolution_height=1000, stake=0)
        with pytest.raises(PluginError) as exc:
            contract._check_message_create_market(msg)
        assert exc.value.code == CODE_INVALID_AMOUNT


class TestCheckMessageResolveMarket:
    """Stateless validation of the prediction market 'resolve_market' message."""

    def test_valid(self, contract):
        msg = MessageResolveMarket(from_address=ADDR_A, market_id=1, actual_class=3)
        result = contract._check_message_resolve_market(msg)

        assert not result.HasField("error")
        assert list(result.authorized_signers) == [ADDR_A]

    def test_invalid_from_address(self, contract):
        msg = MessageResolveMarket(from_address=ADDR_SHORT, market_id=1, actual_class=3)
        with pytest.raises(PluginError) as exc:
            contract._check_message_resolve_market(msg)
        assert exc.value.code == CODE_INVALID_ADDRESS

    def test_invalid_actual_class(self, contract):
        msg = MessageResolveMarket(from_address=ADDR_A, market_id=1, actual_class=16)
        with pytest.raises(PluginError) as exc:
            contract._check_message_resolve_market(msg)
        assert exc.value.code == 1
        assert "invalid actual class" in exc.value.msg


class TestCheckMessageClaimReward:
    """Stateless validation of the prediction market 'claim_reward' message."""

    def test_valid(self, contract):
        msg = MessageClaimReward(from_address=ADDR_A, market_id=1)
        result = contract._check_message_claim_reward(msg)

        assert not result.HasField("error")
        assert list(result.authorized_signers) == [ADDR_A]

    def test_invalid_from_address(self, contract):
        msg = MessageClaimReward(from_address=ADDR_SHORT, market_id=1)
        with pytest.raises(PluginError) as exc:
            contract._check_message_claim_reward(msg)
        assert exc.value.code == CODE_INVALID_ADDRESS


class TestCheckTx:
    """check_tx wiring guards."""

    async def test_check_tx_without_plugin(self, config):
        """check_tx must fail gracefully when no plugin is wired in."""
        contract = Contract(config=config)  # plugin is None
        result = await contract.check_tx(PluginCheckRequest())

        assert result.HasField("error")
        assert "plugin or config not initialized" in result.error.msg


# ---------------------------------------------------------------------------
# DeliverTx end-to-end tests (mock plugin with in-memory state)
# ---------------------------------------------------------------------------

import json as _json
from contract.contract import (
    key_for_account,
    key_for_fee_pool,
    key_for_fee_params,
    key_for_market,
    key_for_market_counter,
    key_for_stake,
    key_for_model_counter,
    key_for_model_registry,
    key_for_active_model,
    marshal,
)


class MockPlugin:
    """In-memory mock of the Canopy Plugin state_read/state_write API."""

    def __init__(self, state: dict):
        self.state = state

    async def state_read(self, contract, request: PluginStateReadRequest) -> PluginStateReadResponse:
        response = PluginStateReadResponse()
        for key_read in request.keys:
            result = PluginReadResult(query_id=key_read.query_id)
            value = self.state.get(key_read.key)
            if value is not None:
                result.entries.append(PluginStateEntry(key=key_read.key, value=value))
            response.results.append(result)
        return response

    async def state_write(self, contract, request: PluginStateWriteRequest) -> PluginStateWriteResponse:
        for set_op in request.sets:
            self.state[set_op.key] = set_op.value
        for delete_op in request.deletes:
            self.state.pop(delete_op.key, None)
        return PluginStateWriteResponse()


def _make_deliver_contract(initial_balance: int = 1_000_000) -> Contract:
    """Build a Contract wired to a MockPlugin with a funded account + fee params."""
    cfg = Config(chain_id=1)
    state: dict = {}

    # Funded account for ADDR_A
    acct = Account(address=ADDR_A, amount=initial_balance, nonce=0)
    state[key_for_account(ADDR_A)] = marshal(acct)

    # Fee pool (chain_id=1)
    pool = Pool(id=1, amount=0)
    state[key_for_fee_pool(1)] = marshal(pool)

    # Fee params (send_fee=10, predict_fee=100)
    fee_params = FeeParams(send_fee=10, predict_fee=100)
    state[key_for_fee_params()] = marshal(fee_params)

    plugin = MockPlugin(state)
    contract = Contract(config=cfg, plugin=plugin, fsm_id=1)
    return contract


def _deliver_request(msg, type_url: str, fee: int = 10) -> PluginDeliverRequest:
    """Wrap a message in a PluginDeliverRequest with the given type URL."""
    from google.protobuf import any_pb2

    any_msg = any_pb2.Any()
    any_msg.type_url = type_url
    any_msg.value = msg.SerializeToString()

    tx = Transaction(
        message_type=type_url.split("/")[-1],
        fee=fee,
        memo="",
        nonce=0,
    )
    tx.msg.CopyFrom(any_msg)

    return PluginDeliverRequest(tx=tx, height=1)


class TestDeliverMarketplace:
    """Full end-to-end marketplace cycle: create_market → stake → resolve → claim."""

    async def test_full_cycle(self):
        contract = _make_deliver_contract(initial_balance=1_000_000)

        # 1. Create market (fee=10, stake=5000)
        create_msg = MessageCreateMarket(
            from_address=ADDR_A,
            question="BTC up?",
            resolution_height=1000,
            stake=5000,
        )
        resp = await contract.deliver_tx(_deliver_request(create_msg, "type.googleapis.com/types.MessageCreateMarket"))
        assert not resp.HasField("error"), resp.error

        # Market ID 1 should exist in state
        market_bytes = contract.plugin.state.get(key_for_market(1))
        assert market_bytes is not None
        market = _json.loads(market_bytes.decode("utf-8"))
        assert market["id"] == 1
        assert market["creator"] == ADDR_A.hex()
        assert market["total_pool"] == 5000
        assert market["resolved"] is False

        # Counter should be 1
        counter_bytes = contract.plugin.state.get(key_for_market_counter())
        assert int.from_bytes(counter_bytes, "big") == 1

        # 2. Stake on outcome 3 (fee=10, amount=2000)
        stake_msg = MessageStake(from_address=ADDR_A, market_id=1, amount=2000, outcome=3)
        resp = await contract.deliver_tx(_deliver_request(stake_msg, "type.googleapis.com/types.MessageStake"))
        assert not resp.HasField("error"), resp.error

        # Market pool updated (JSON keys are strings)
        market_bytes = contract.plugin.state.get(key_for_market(1))
        market = _json.loads(market_bytes.decode("utf-8"))
        assert market["total_pool"] == 7000
        assert market["outcome_pools"]["3"] == 2000

        # Stake record created
        stake_bytes = contract.plugin.state.get(key_for_stake(ADDR_A, 1))
        assert stake_bytes is not None
        stake = _json.loads(stake_bytes.decode("utf-8"))
        assert stake["outcome"] == 3
        assert stake["amount"] == 2000
        assert stake["claimed"] is False

        # 3. Resolve market with actual_class=3 (fee=10)
        resolve_msg = MessageResolveMarket(from_address=ADDR_A, market_id=1, actual_class=3)
        resp = await contract.deliver_tx(_deliver_request(resolve_msg, "type.googleapis.com/types.MessageResolveMarket"))
        assert not resp.HasField("error"), resp.error

        market_bytes = contract.plugin.state.get(key_for_market(1))
        market = _json.loads(market_bytes.decode("utf-8"))
        assert market["resolved"] is True
        assert market["actual_class"] == 3
        assert market["resolver"] == ADDR_A.hex()

        # 4. Claim reward (fee=10)
        claim_msg = MessageClaimReward(from_address=ADDR_A, market_id=1)
        resp = await contract.deliver_tx(_deliver_request(claim_msg, "type.googleapis.com/types.MessageClaimReward"))
        assert not resp.HasField("error"), resp.error

        # Stake marked as claimed
        stake_bytes = contract.plugin.state.get(key_for_stake(ADDR_A, 1))
        stake = _json.loads(stake_bytes.decode("utf-8"))
        assert stake["claimed"] is True

        # Account balance: 1_000_000 - 10 - 5000 - 10 - 2000 - 10 - 10 + reward
        # reward = (2000 * 7000) // 2000 = 7000
        acct_bytes = contract.plugin.state.get(key_for_account(ADDR_A))
        acct = Account.FromString(acct_bytes)
        expected = 1_000_000 - 10 - 5000 - 10 - 2000 - 10 - 10 + 7000
        assert acct.amount == expected, f"expected {expected}, got {acct.amount}"

    async def test_stake_missing_market_fails(self):
        contract = _make_deliver_contract()

        stake_msg = MessageStake(from_address=ADDR_A, market_id=99, amount=1000, outcome=3)
        resp = await contract.deliver_tx(_deliver_request(stake_msg, "type.googleapis.com/types.MessageStake"))

        assert resp.HasField("error")
        assert "market not found" in resp.error.msg

    async def test_resolve_non_creator_fails(self):
        contract = _make_deliver_contract()

        # Create market as ADDR_A
        create_msg = MessageCreateMarket(
            from_address=ADDR_A,
            question="BTC up?",
            resolution_height=1000,
            stake=5000,
        )
        resp = await contract.deliver_tx(_deliver_request(create_msg, "type.googleapis.com/types.MessageCreateMarket"))
        assert not resp.HasField("error"), resp.error

        # Try to resolve as ADDR_B (not creator)
        resolve_msg = MessageResolveMarket(from_address=ADDR_B, market_id=1, actual_class=3)
        resp = await contract.deliver_tx(_deliver_request(resolve_msg, "type.googleapis.com/types.MessageResolveMarket"))

        assert resp.HasField("error")
        assert "only market creator can resolve" in resp.error.msg

    async def test_claim_wrong_outcome_fails(self):
        contract = _make_deliver_contract()

        # Create market
        create_msg = MessageCreateMarket(
            from_address=ADDR_A,
            question="BTC up?",
            resolution_height=1000,
            stake=5000,
        )
        resp = await contract.deliver_tx(_deliver_request(create_msg, "type.googleapis.com/types.MessageCreateMarket"))
        assert not resp.HasField("error"), resp.error

        # Stake on outcome 3
        stake_msg = MessageStake(from_address=ADDR_A, market_id=1, amount=2000, outcome=3)
        resp = await contract.deliver_tx(_deliver_request(stake_msg, "type.googleapis.com/types.MessageStake"))
        assert not resp.HasField("error"), resp.error

        # Resolve with actual_class=5 (wrong)
        resolve_msg = MessageResolveMarket(from_address=ADDR_A, market_id=1, actual_class=5)
        resp = await contract.deliver_tx(_deliver_request(resolve_msg, "type.googleapis.com/types.MessageResolveMarket"))
        assert not resp.HasField("error"), resp.error

        # Claim should fail
        claim_msg = MessageClaimReward(from_address=ADDR_A, market_id=1)
        resp = await contract.deliver_tx(_deliver_request(claim_msg, "type.googleapis.com/types.MessageClaimReward"))

        assert resp.HasField("error")
        assert "stake was not correct" in resp.error.msg

    async def test_claim_twice_fails(self):
        contract = _make_deliver_contract()

        # Full cycle
        create_msg = MessageCreateMarket(
            from_address=ADDR_A,
            question="BTC up?",
            resolution_height=1000,
            stake=5000,
        )
        resp = await contract.deliver_tx(_deliver_request(create_msg, "type.googleapis.com/types.MessageCreateMarket"))
        assert not resp.HasField("error"), resp.error

        stake_msg = MessageStake(from_address=ADDR_A, market_id=1, amount=2000, outcome=3)
        resp = await contract.deliver_tx(_deliver_request(stake_msg, "type.googleapis.com/types.MessageStake"))
        assert not resp.HasField("error"), resp.error

        resolve_msg = MessageResolveMarket(from_address=ADDR_A, market_id=1, actual_class=3)
        resp = await contract.deliver_tx(_deliver_request(resolve_msg, "type.googleapis.com/types.MessageResolveMarket"))
        assert not resp.HasField("error"), resp.error

        claim_msg = MessageClaimReward(from_address=ADDR_A, market_id=1)
        resp = await contract.deliver_tx(_deliver_request(claim_msg, "type.googleapis.com/types.MessageClaimReward"))
        assert not resp.HasField("error"), resp.error

        # Second claim should fail
        resp = await contract.deliver_tx(_deliver_request(claim_msg, "type.googleapis.com/types.MessageClaimReward"))
        assert resp.HasField("error")
        assert "reward already claimed" in resp.error.msg

    async def test_insufficient_funds_fails(self):
        contract = _make_deliver_contract(initial_balance=100)

        create_msg = MessageCreateMarket(
            from_address=ADDR_A,
            question="BTC up?",
            resolution_height=1000,
            stake=5000,
        )
        resp = await contract.deliver_tx(_deliver_request(create_msg, "type.googleapis.com/types.MessageCreateMarket"))

        assert resp.HasField("error")
        assert "insufficient funds" in resp.error.msg


class TestCheckMessageRegisterModel:
    """Stateless validation of the model versioning 'register_model' message."""

    def test_valid(self, contract):
        msg = MessageRegisterModel(
            from_address=ADDR_A,
            version=1,
            weights_hash="abc123",
            accuracy=0.95,
            in_dim=42,
            n_classes=16,
            description="42D krypto",
        )
        result = contract._check_message_register_model(msg)

        assert not result.HasField("error")
        assert list(result.authorized_signers) == [ADDR_A]

    def test_invalid_from_address(self, contract):
        msg = MessageRegisterModel(
            from_address=ADDR_SHORT,
            version=1,
            weights_hash="abc123",
            accuracy=0.95,
            in_dim=42,
            n_classes=16,
        )
        with pytest.raises(PluginError) as exc:
            contract._check_message_register_model(msg)
        assert exc.value.code == CODE_INVALID_ADDRESS

    def test_zero_version(self, contract):
        msg = MessageRegisterModel(
            from_address=ADDR_A,
            version=0,
            weights_hash="abc123",
            accuracy=0.95,
            in_dim=42,
            n_classes=16,
        )
        with pytest.raises(PluginError) as exc:
            contract._check_message_register_model(msg)
        assert exc.value.code == 1
        assert "version must be > 0" in exc.value.msg

    def test_empty_weights_hash(self, contract):
        msg = MessageRegisterModel(
            from_address=ADDR_A,
            version=1,
            weights_hash="",
            accuracy=0.95,
            in_dim=42,
            n_classes=16,
        )
        with pytest.raises(PluginError) as exc:
            contract._check_message_register_model(msg)
        assert exc.value.code == 1
        assert "weights hash cannot be empty" in exc.value.msg

    def test_invalid_accuracy(self, contract):
        msg = MessageRegisterModel(
            from_address=ADDR_A,
            version=1,
            weights_hash="abc123",
            accuracy=1.5,
            in_dim=42,
            n_classes=16,
        )
        with pytest.raises(PluginError) as exc:
            contract._check_message_register_model(msg)
        assert exc.value.code == 1
        assert "accuracy must be in [0, 1]" in exc.value.msg

    def test_invalid_in_dim(self, contract):
        msg = MessageRegisterModel(
            from_address=ADDR_A,
            version=1,
            weights_hash="abc123",
            accuracy=0.95,
            in_dim=64,
            n_classes=16,
        )
        with pytest.raises(PluginError) as exc:
            contract._check_message_register_model(msg)
        assert exc.value.code == 1
        assert "in_dim must be 28 or 42" in exc.value.msg

    def test_zero_n_classes(self, contract):
        msg = MessageRegisterModel(
            from_address=ADDR_A,
            version=1,
            weights_hash="abc123",
            accuracy=0.95,
            in_dim=42,
            n_classes=0,
        )
        with pytest.raises(PluginError) as exc:
            contract._check_message_register_model(msg)
        assert exc.value.code == 1
        assert "n_classes must be > 0" in exc.value.msg


class TestDeliverModelRegistry:
    """Full end-to-end model versioning: register_model → registry + active pointer."""

    async def test_register_first_model(self):
        contract = _make_deliver_contract()

        msg = MessageRegisterModel(
            from_address=ADDR_A,
            version=1,
            weights_hash="abc123",
            accuracy=0.95,
            in_dim=42,
            n_classes=16,
            description="42D krypto",
        )
        resp = await contract.deliver_tx(_deliver_request(msg, "type.googleapis.com/types.MessageRegisterModel"))
        assert not resp.HasField("error"), resp.error

        # Model version 1 should exist in registry
        model_bytes = contract.plugin.state.get(key_for_model_registry(1))
        assert model_bytes is not None
        model = _json.loads(model_bytes.decode("utf-8"))
        assert model["version"] == 1
        assert model["weights_hash"] == "abc123"
        assert model["accuracy"] == 0.95
        assert model["in_dim"] == 42
        assert model["n_classes"] == 16
        assert model["description"] == "42D krypto"
        assert model["registered_by"] == ADDR_A.hex()

        # Counter should be 1
        counter_bytes = contract.plugin.state.get(key_for_model_counter())
        assert int.from_bytes(counter_bytes, "big") == 1

        # Active model pointer should be 1
        active_bytes = contract.plugin.state.get(key_for_active_model())
        assert active_bytes.decode("utf-8") == "1"

        # Account balance: 1_000_000 - 10 (fee)
        acct_bytes = contract.plugin.state.get(key_for_account(ADDR_A))
        acct = Account.FromString(acct_bytes)
        assert acct.amount == 1_000_000 - 10

    async def test_register_second_model_updates_active(self):
        contract = _make_deliver_contract()

        # Register v1
        msg1 = MessageRegisterModel(
            from_address=ADDR_A,
            version=1,
            weights_hash="abc123",
            accuracy=0.90,
            in_dim=28,
            n_classes=16,
            description="28D",
        )
        resp = await contract.deliver_tx(_deliver_request(msg1, "type.googleapis.com/types.MessageRegisterModel"))
        assert not resp.HasField("error"), resp.error

        # Register v2
        msg2 = MessageRegisterModel(
            from_address=ADDR_A,
            version=2,
            weights_hash="def456",
            accuracy=0.95,
            in_dim=42,
            n_classes=16,
            description="42D krypto",
        )
        resp = await contract.deliver_tx(_deliver_request(msg2, "type.googleapis.com/types.MessageRegisterModel"))
        assert not resp.HasField("error"), resp.error

        # Both versions in registry
        model1_bytes = contract.plugin.state.get(key_for_model_registry(1))
        model1 = _json.loads(model1_bytes.decode("utf-8"))
        assert model1["version"] == 1
        assert model1["in_dim"] == 28

        model2_bytes = contract.plugin.state.get(key_for_model_registry(2))
        model2 = _json.loads(model2_bytes.decode("utf-8"))
        assert model2["version"] == 2
        assert model2["in_dim"] == 42

        # Counter should be 2
        counter_bytes = contract.plugin.state.get(key_for_model_counter())
        assert int.from_bytes(counter_bytes, "big") == 2

        # Active model pointer should be 2 (latest)
        active_bytes = contract.plugin.state.get(key_for_active_model())
        assert active_bytes.decode("utf-8") == "2"

    async def test_register_insufficient_funds_fails(self):
        contract = _make_deliver_contract(initial_balance=5)

        msg = MessageRegisterModel(
            from_address=ADDR_A,
            version=1,
            weights_hash="abc123",
            accuracy=0.95,
            in_dim=42,
            n_classes=16,
        )
        resp = await contract.deliver_tx(_deliver_request(msg, "type.googleapis.com/types.MessageRegisterModel"))

        assert resp.HasField("error")
        assert "insufficient funds" in resp.error.msg


class TestDashboardOnChain(unittest.TestCase):
    """Test the on-chain dashboard aggregate record (0x0b)."""

    def setUp(self):
        self.plugin = MockPlugin({})
        self.contract = Contract(config=Config(chain_id=1), plugin=self.plugin)
        # Seed fee params
        fee_params = FeeParams(send_fee=10, predict_fee=20)
        self.plugin.state[key_for_fee_params()] = fee_params.SerializeToString()
        # Seed sender account with 1M QARD
        self.sender = bytes(range(20))
        self.sender_account = Account(amount=1_000_000, nonce=0)
        self.plugin.state[key_for_account(self.sender)] = self.sender_account.SerializeToString()

    def _make_predict_tx(self):
        msg = MessagePredict(from_address=self.sender, features=[1.0]*42)
        tx = Transaction(msg=Any(type_url='type.googleapis.com/types.MessagePredict', value=msg.SerializeToString()), fee=20)
        return tx

    def test_dashboard_created_after_predict(self):
        """Dashboard record is created after a predict tx."""
        tx = self._make_predict_tx()
        resp = asyncio.get_event_loop().run_until_complete(self.contract.deliver_tx(PluginDeliverRequest(tx=tx)))
        self.assertFalse(resp.HasField('error'))
        dashboard_key = key_for_dashboard()
        self.assertIn(dashboard_key, self.plugin.state)
        dashboard = json.loads(self.plugin.state[dashboard_key].decode('utf-8'))
        self.assertEqual(dashboard['total_predictions'], 1)
        self.assertEqual(dashboard['revenue'], 20)
        self.assertIn('class_counts', dashboard)
        self.assertIn('accuracy', dashboard)
        self.assertEqual(dashboard['accuracy'], 0.0)

    def test_dashboard_accumulates_predictions(self):
        """Dashboard accumulates multiple predictions."""
        for _ in range(3):
            tx = self._make_predict_tx()
            resp = asyncio.get_event_loop().run_until_complete(self.contract.deliver_tx(PluginDeliverRequest(tx=tx)))
            self.assertFalse(resp.HasField('error'))
        dashboard_key = key_for_dashboard()
        dashboard = json.loads(self.plugin.state[dashboard_key].decode('utf-8'))
        self.assertEqual(dashboard['total_predictions'], 3)
        self.assertEqual(dashboard['revenue'], 60)

    def test_dashboard_feedback_updates_accuracy(self):
        """Feedback updates accuracy in dashboard."""
        # First predict
        tx = self._make_predict_tx()
        resp = asyncio.get_event_loop().run_until_complete(self.contract.deliver_tx(PluginDeliverRequest(tx=tx)))
        self.assertFalse(resp.HasField('error'))
        # Send feedback (correct)
        msg = MessageFeedback(from_address=self.sender, predict_seq=0, correct=True, actual_class=1)
        tx = Transaction(msg=Any(type_url='type.googleapis.com/types.MessageFeedback', value=msg.SerializeToString()), fee=10)
        resp = asyncio.get_event_loop().run_until_complete(self.contract.deliver_tx(PluginDeliverRequest(tx=tx)))
        self.assertFalse(resp.HasField('error'))
        dashboard_key = key_for_dashboard()
        dashboard = json.loads(self.plugin.state[dashboard_key].decode('utf-8'))
        self.assertEqual(dashboard['total_feedback'], 1)
        self.assertEqual(dashboard['correct_feedback'], 1)
        self.assertEqual(dashboard['accuracy'], 1.0)

    def test_dashboard_market_creation(self):
        """Dashboard tracks market creation."""
        msg = MessageCreateMarket(from_address=self.sender, question='Will G2 predict?', resolution_height=100, stake=100)
        tx = Transaction(msg=Any(type_url='type.googleapis.com/types.MessageCreateMarket', value=msg.SerializeToString()), fee=10)
        resp = asyncio.get_event_loop().run_until_complete(self.contract.deliver_tx(PluginDeliverRequest(tx=tx)))
        self.assertFalse(resp.HasField('error'))
        dashboard_key = key_for_dashboard()
        dashboard = json.loads(self.plugin.state[dashboard_key].decode('utf-8'))
        self.assertEqual(dashboard['total_markets'], 1)
        self.assertEqual(dashboard['total_staked'], 100)

    def test_dashboard_model_registration(self):
        """Dashboard tracks model registration."""
        msg = MessageRegisterModel(from_address=self.sender, version=1, weights_hash='abc123', accuracy=0.95, in_dim=42, n_classes=16, description='test')
        tx = Transaction(msg=Any(type_url='type.googleapis.com/types.MessageRegisterModel', value=msg.SerializeToString()), fee=10)
        resp = asyncio.get_event_loop().run_until_complete(self.contract.deliver_tx(PluginDeliverRequest(tx=tx)))
        self.assertFalse(resp.HasField('error'))
        dashboard_key = key_for_dashboard()
        dashboard = json.loads(self.plugin.state[dashboard_key].decode('utf-8'))
        self.assertEqual(dashboard['total_models'], 1)
        self.assertEqual(dashboard['active_model'], 1)

if __name__ == '__main__':
    unittest.main()
