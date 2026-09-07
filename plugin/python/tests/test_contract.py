"""
Unit tests for the Contract class.

Covers the current `contract.contract` API: lifecycle hooks, stateless message
validation for the base 'send' transaction, and the on-chain AI 'predict' transaction.
"""

import pytest

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
