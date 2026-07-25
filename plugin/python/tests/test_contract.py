import pytest
from unittest.mock import AsyncMock, MagicMock
from google.protobuf.any_pb2 import Any

from contract import Contract, Config, PluginError
from contract.proto import (
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
    PluginStateReadResponse,
    PluginStateWriteRequest,
    PluginStateWriteResponse,
    PluginReadResult,
    PluginStateEntry,
    PluginSetOp,
    PluginDeleteOp,
    Transaction,
    Account,
    Pool,
    FeeParams,
)

@pytest.fixture
def config():
    return Config(chain_id=1, data_dir_path="/tmp/plugin/")

@pytest.fixture
def mock_plugin():
    plugin = MagicMock()
    
    async def side_effect_read(contract, request):
        results = []
        for key_read in request.keys:
            val = b""
            # Because of join_len_prefix, the prefixes are length-prefixed:
            # - ACCOUNT_PREFIX (b"\x01") -> b"\x01\x01..."
            # - POOL_PREFIX (b"\x02")    -> b"\x01\x02..."
            # - PARAMS_PREFIX (b"\x07")  -> b"\x01\x07..."
            if key_read.key.startswith(b"\x01\x01"):
                acc = Account(amount=10000)
                val = acc.SerializeToString()
            elif key_read.key.startswith(b"\x01\x02"):
                pool = Pool(amount=500)
                val = pool.SerializeToString()
            elif key_read.key.startswith(b"\x01\x07"):
                params = FeeParams(send_fee=100)
                val = params.SerializeToString()
            results.append(PluginReadResult(
                query_id=key_read.query_id,
                entries=[PluginStateEntry(value=val)]
            ))
        return PluginStateReadResponse(results=results)

    plugin.state_read = AsyncMock(side_effect=side_effect_read)
    plugin.state_write = AsyncMock(return_value=PluginStateWriteResponse())
    return plugin

@pytest.fixture
def contract(config, mock_plugin):
    return Contract(config=config, plugin=mock_plugin, fsm_id=1)

class TestContract:
    def test_genesis(self, contract):
        req = PluginGenesisRequest()
        resp = contract.genesis(req)
        assert isinstance(resp, PluginGenesisResponse)

    def test_begin_block(self, contract):
        req = PluginBeginRequest()
        resp = contract.begin_block(req)
        assert isinstance(resp, PluginBeginResponse)

    def test_end_block(self, contract):
        req = PluginEndRequest()
        resp = contract.end_block(req)
        assert isinstance(resp, PluginEndResponse)

    @pytest.mark.asyncio
    async def test_check_tx_valid(self, contract):
        msg = MessageSend(from_address=b'a'*20, to_address=b'b'*20, amount=500)
        any_msg = Any()
        any_msg.Pack(msg)
        
        tx = Transaction(fee=200, msg=any_msg)
        req = PluginCheckRequest(tx=tx)
        
        resp = await contract.check_tx(req)
        assert not resp.HasField("error")
        assert resp.recipient == b'b'*20
        assert list(resp.authorized_signers) == [b'a'*20]

    @pytest.mark.asyncio
    async def test_check_tx_insufficient_fee(self, contract):
        msg = MessageSend(from_address=b'a'*20, to_address=b'b'*20, amount=500)
        any_msg = Any()
        any_msg.Pack(msg)
        
        # Fee is 50, but min is 100
        tx = Transaction(fee=50, msg=any_msg)
        req = PluginCheckRequest(tx=tx)
        
        resp = await contract.check_tx(req)
        assert resp.HasField("error")
        assert resp.error.code == 14  # CodeTxFeeBelowLimit

    @pytest.mark.asyncio
    async def test_check_tx_invalid_address(self, contract):
        # Invalid address length
        msg = MessageSend(from_address=b'a'*10, to_address=b'b'*20, amount=500)
        any_msg = Any()
        any_msg.Pack(msg)
        
        tx = Transaction(fee=200, msg=any_msg)
        req = PluginCheckRequest(tx=tx)
        
        resp = await contract.check_tx(req)
        assert resp.HasField("error")
        assert resp.error.code == 12  # CodeInvalidAddress

    @pytest.mark.asyncio
    async def test_check_tx_invalid_amount(self, contract):
        # Amount is 0
        msg = MessageSend(from_address=b'a'*20, to_address=b'b'*20, amount=0)
        any_msg = Any()
        any_msg.Pack(msg)
        
        tx = Transaction(fee=200, msg=any_msg)
        req = PluginCheckRequest(tx=tx)
        
        resp = await contract.check_tx(req)
        assert resp.HasField("error")
        assert resp.error.code == 13  # CodeInvalidAmount

    @pytest.mark.asyncio
    async def test_deliver_tx_valid(self, contract, mock_plugin):
        msg = MessageSend(from_address=b'a'*20, to_address=b'b'*20, amount=500)
        any_msg = Any()
        any_msg.Pack(msg)
        
        tx = Transaction(fee=200, msg=any_msg)
        req = PluginDeliverRequest(tx=tx)
        
        resp = await contract.deliver_tx(req)
        assert not resp.HasField("error")
        assert mock_plugin.state_write.called

    @pytest.mark.asyncio
    async def test_deliver_tx_insufficient_funds(self, contract, mock_plugin):
        msg = MessageSend(from_address=b'a'*20, to_address=b'b'*20, amount=50000) # exceeds 10000 balance
        any_msg = Any()
        any_msg.Pack(msg)
        
        tx = Transaction(fee=200, msg=any_msg)
        req = PluginDeliverRequest(tx=tx)
        
        resp = await contract.deliver_tx(req)
        assert resp.HasField("error")
        assert resp.error.code == 9  # CodeInsufficientFunds (err_insufficient_funds is code 9)