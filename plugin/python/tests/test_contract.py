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