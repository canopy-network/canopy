"""
Unit tests for Contract class.

Tests transaction validation, processing logic, and error conditions.
"""

import pytest
from unittest.mock import Mock, AsyncMock

from plugin.core import Contract, ContractOptions
from plugin.config import Config
from plugin.core.exceptions import PluginException


class TestContract:
    """Test cases for Contract class."""
    
    @pytest.fixture
    def config(self):
        """Create test configuration."""
        return Config()
    
    @pytest.fixture
    def mock_plugin(self):
        """Create mock socket client plugin."""
        plugin = Mock()
        plugin.state_read = AsyncMock()
        plugin.state_write = AsyncMock()
        return plugin
    
    @pytest.fixture
    def contract(self, config, mock_plugin):
        """Create contract instance for testing."""
        options = ContractOptions(
            config=config,
            plugin=mock_plugin,
            fsm_id=1
        )
        return Contract(options)
    
    def test_genesis(self, contract):
        """Test genesis method."""
        result = contract.genesis({})
        assert result.error is None
    
    def test_begin_block(self, contract):
        """Test begin block method."""
        result = contract.begin_block({})
        assert result.error is None
    
    def test_end_block(self, contract):
        """Test end block method.""" 
        result = contract.end_block({})
        assert result.error is None
    
    def test_is_message_send_dict(self, contract):
        """Test _is_message_send with dict format."""
        msg = {
            'from_address': b'a' * 20,
            'to_address': b'b' * 20,
            'amount': 1000
        }
        assert contract._is_message_send(msg) is True
    
    def test_is_message_send_invalid(self, contract):
        """Test _is_message_send with invalid message."""
        msg = {'invalid': 'message'}
        assert contract._is_message_send(msg) is False
    
    def test_check_message_send_valid(self, contract):
        """Test _check_message_send with valid message."""
        msg = {
            'from_address': b'a' * 20,
            'to_address': b'b' * 20, 
            'amount': 1000
        }
        result = contract._check_message_send(msg)
        
        assert result.error is None
        assert result.recipient == b'b' * 20
        assert result.authorized_signers == [b'a' * 20]
    
    def test_check_message_send_invalid_from_address(self, contract):
        """Test _check_message_send with invalid from address."""
        msg = {
            'from_address': b'short',  # Not 20 bytes
            'to_address': b'b' * 20,
            'amount': 1000
        }
        result = contract._check_message_send(msg)
        
        assert result.error is not None
        assert result.error['code'] == 12  # Invalid address error code
    
    def test_check_message_send_invalid_to_address(self, contract):
        """Test _check_message_send with invalid to address."""
        msg = {
            'from_address': b'a' * 20,
            'to_address': b'short',  # Not 20 bytes
            'amount': 1000
        }
        result = contract._check_message_send(msg)
        
        assert result.error is not None
        assert result.error['code'] == 12  # Invalid address error code
    
    def test_check_message_send_invalid_amount(self, contract):
        """Test _check_message_send with invalid amount."""
        msg = {
            'from_address': b'a' * 20,
            'to_address': b'b' * 20,
            'amount': 0  # Invalid amount
        }
        result = contract._check_message_send(msg)
        
        assert result.error is not None
        assert result.error['code'] == 13  # Invalid amount error code
    
    def test_generate_query_ids(self, contract):
        """Test _generate_query_ids method."""
        ids = contract._generate_query_ids()
        
        assert 'from_query_id' in ids
        assert 'to_query_id' in ids
        assert 'fee_query_id' in ids
        
        # All IDs should be different
        assert ids['from_query_id'] != ids['to_query_id']
        assert ids['to_query_id'] != ids['fee_query_id']
        assert ids['from_query_id'] != ids['fee_query_id']


@pytest.mark.asyncio
class TestContractAsync:
    """Async test cases for Contract class."""
    
    @pytest.fixture
    def config(self):
        """Create test configuration."""
        return Config()
    
    @pytest.fixture
    def mock_plugin(self):
        """Create mock socket client plugin."""
        plugin = Mock()
        plugin.state_read = AsyncMock(return_value={
            'error': None,
            'results': [{'entries': [{'value': b'test_data'}]}]
        })
        plugin.state_write = AsyncMock(return_value={'error': None})
        return plugin
    
    @pytest.fixture
    def contract(self, config, mock_plugin):
        """Create contract instance for testing."""
        options = ContractOptions(
            config=config,
            plugin=mock_plugin,
            fsm_id=1
        )
        return Contract(options)
    
    async def test_check_tx_no_plugin(self, config):
        """Test check_tx without plugin."""
        contract = Contract(ContractOptions(config=config))
        
        request = {'tx': {'fee': 1000, 'msg': {}}}
        result = await contract.check_tx(request)
        
        assert result.error is not None
        assert 'Plugin or config not initialized' in result.error['msg']