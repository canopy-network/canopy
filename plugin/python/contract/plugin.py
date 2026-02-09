"""
Plugin communication with Canopy FSM via Unix socket.

This file contains boilerplate logic to interact with the Canopy FSM via socket file.
Matches Go's contract/plugin.go structure.
"""

import asyncio
import json
import logging
import os
import struct
from dataclasses import dataclass
from pathlib import Path
from typing import Optional, Dict, Any, Set

# Import proto types
from .proto import (
    FSMToPlugin,
    PluginToFSM,
    PluginConfig,
    PluginFSMConfig,
    PluginStateReadRequest,
    PluginStateReadResponse,
    PluginStateWriteRequest,
    PluginStateWriteResponse,
)

from .error import (
    PluginError,
    err_plugin_timeout,
    err_failed_plugin_read,
    err_failed_plugin_write,
    err_invalid_plugin_resp_id,
    err_unexpected_fsm_to_plugin,
    err_invalid_fsm_to_plugin_message,
)
from .contract import Contract, CONTRACT_CONFIG

logger = logging.getLogger(__name__)

# Socket path name (matching Go)
SOCKET_PATH = "plugin.sock"


@dataclass
class Config:
    """Plugin configuration matching Go's Config struct."""
    chain_id: int = 1
    data_dir_path: str = "/tmp/plugin/"

    def __post_init__(self) -> None:
        if not isinstance(self.chain_id, int) or self.chain_id < 1:
            raise ValueError(f"Invalid chain_id: {self.chain_id}")
        if not isinstance(self.data_dir_path, str) or not self.data_dir_path.strip():
            raise ValueError(f"Invalid data_dir_path: {self.data_dir_path}")


def default_config() -> Config:
    """Return the default configuration (matching Go's DefaultConfig)."""
    return Config(chain_id=1, data_dir_path="/tmp/plugin/")


def new_config_from_file(filepath: str) -> Config:
    """Load configuration from JSON file (matching Go's NewConfigFromFile)."""
    try:
        config_data = json.loads(Path(filepath).read_text(encoding="utf-8"))
        return Config(
            chain_id=config_data.get("chainId", 1),
            data_dir_path=config_data.get("dataDirPath", "/tmp/plugin/"),
        )
    except Exception as err:
        raise ValueError(f"Failed to load config from {filepath}: {err}")


class Plugin:
    """
    Plugin defines the 'VM-less' extension of the Finite State Machine.
    Matches Go's Plugin struct.
    """

    def __init__(self, config: Config):
        self.config = config
        self.fsm_config: Optional[PluginFSMConfig] = None
        self.plugin_config = PluginConfig()
        self._reader: Optional[asyncio.StreamReader] = None
        self._writer: Optional[asyncio.StreamWriter] = None
        self._pending: Dict[int, asyncio.Future] = {}
        self._request_contracts: Dict[int, Contract] = {}
        self._listen_task: Optional[asyncio.Task] = None
        self._message_tasks: Set[asyncio.Task] = set()
        self._is_connected = False

        # Setup plugin config
        self.plugin_config.name = CONTRACT_CONFIG["name"]
        self.plugin_config.id = CONTRACT_CONFIG["id"]
        self.plugin_config.version = CONTRACT_CONFIG["version"]
        for tx_type in CONTRACT_CONFIG["supported_transactions"]:
            self.plugin_config.supported_transactions.append(tx_type)
        for url in CONTRACT_CONFIG.get("transaction_type_urls", []):
            self.plugin_config.transaction_type_urls.append(url)
        for url in CONTRACT_CONFIG.get("event_type_urls", []):
            self.plugin_config.event_type_urls.append(url)
        for fd in CONTRACT_CONFIG.get("file_descriptor_protos", []):
            self.plugin_config.file_descriptor_protos.append(fd)

    async def start(self) -> None:
        """Start the plugin - connect to socket and begin listening."""
        sock_path = os.path.join(self.config.data_dir_path, SOCKET_PATH)

        # Connect to the socket with retry (matching Go's polling loop)
        while True:
            try:
                self._reader, self._writer = await asyncio.open_unix_connection(sock_path)
                self._is_connected = True
                logger.info(f"Connected to plugin socket: {sock_path}")
                break
            except Exception as err:
                logger.warning(f"Failed to connect to plugin socket: {err}")
                await asyncio.sleep(1.0)

        # Begin listening service
        self._listen_task = asyncio.create_task(self._listen_for_inbound())

        # Execute handshake
        await self._handshake()

    async def close(self) -> None:
        """Close the plugin gracefully."""
        self._is_connected = False

        if self._listen_task and not self._listen_task.done():
            self._listen_task.cancel()
            try:
                await self._listen_task
            except asyncio.CancelledError:
                pass

        for task in self._message_tasks.copy():
            if not task.done():
                task.cancel()

        if self._message_tasks:
            await asyncio.gather(*self._message_tasks, return_exceptions=True)

        if self._writer:
            self._writer.close()
            await self._writer.wait_closed()

        logger.info("Plugin closed")

    async def _handshake(self) -> None:
        """Handshake sends the contract configuration to the FSM and awaits a reply."""
        logger.info("Handshaking with FSM")

        # Send config to FSM
        message = PluginToFSM()
        message.id = 0
        message.config.CopyFrom(self.plugin_config)

        await self._send_proto_msg(message)

        # Wait for response - the FSM will send back its config
        # This will be handled in _listen_for_inbound

    async def state_read(self, contract: Contract, request: PluginStateReadRequest) -> PluginStateReadResponse:
        """StateRead sends a state read request to FSM and waits for response."""
        if contract.fsm_id is None:
            raise PluginError(1, "plugin", "Contract fsm_id is not set")

        fsm_id = contract.fsm_id

        # Create future for response
        future: asyncio.Future = asyncio.Future()
        self._pending[fsm_id] = future
        self._request_contracts[fsm_id] = contract

        # Send request
        message = PluginToFSM()
        message.id = fsm_id
        message.state_read.CopyFrom(request)

        try:
            logger.debug(f"0x{fsm_id:x}: state_read")
            await self._send_proto_msg(message)

            # Wait for response with timeout
            response = await asyncio.wait_for(future, timeout=10.0)

            if response.HasField("state_read"):
                return response.state_read
            else:
                raise err_unexpected_fsm_to_plugin(type(response))

        except asyncio.TimeoutError:
            raise err_plugin_timeout()
        finally:
            self._pending.pop(fsm_id, None)
            self._request_contracts.pop(fsm_id, None)

    async def state_write(self, contract: Contract, request: PluginStateWriteRequest) -> PluginStateWriteResponse:
        """StateWrite sends a state write request to FSM and waits for response."""
        if contract.fsm_id is None:
            raise PluginError(1, "plugin", "Contract fsm_id is not set")

        fsm_id = contract.fsm_id

        # Create future for response
        future: asyncio.Future = asyncio.Future()
        self._pending[fsm_id] = future
        self._request_contracts[fsm_id] = contract

        # Send request
        message = PluginToFSM()
        message.id = fsm_id
        message.state_write.CopyFrom(request)

        try:
            logger.debug(f"0x{fsm_id:x}: state_write")
            await self._send_proto_msg(message)

            # Wait for response with timeout
            response = await asyncio.wait_for(future, timeout=10.0)

            if response.HasField("state_write"):
                return response.state_write
            else:
                raise err_unexpected_fsm_to_plugin(type(response))

        except asyncio.TimeoutError:
            raise err_plugin_timeout()
        finally:
            self._pending.pop(fsm_id, None)
            self._request_contracts.pop(fsm_id, None)

    async def _listen_for_inbound(self) -> None:
        """ListenForInbound routes inbound requests from the plugin."""
        if not self._reader:
            raise PluginError(1, "plugin", "No reader available")

        try:
            while self._is_connected:
                try:
                    # Read length prefix (4 bytes, big-endian)
                    length_data = await asyncio.wait_for(
                        self._reader.readexactly(4), timeout=10.0
                    )
                    message_length = struct.unpack(">I", length_data)[0]

                    # Read message data
                    message_data = await asyncio.wait_for(
                        self._reader.readexactly(message_length), timeout=10.0
                    )

                    # Handle message concurrently
                    task = asyncio.create_task(self._handle_inbound_message(message_data))
                    self._message_tasks.add(task)
                    task.add_done_callback(self._message_tasks.discard)

                except asyncio.TimeoutError:
                    if not self._is_connected:
                        break
                    continue

        except asyncio.IncompleteReadError:
            logger.info("Connection closed by FSM")
        except asyncio.CancelledError:
            logger.info("Message listening cancelled")
        except Exception as err:
            logger.error(f"Error reading from socket: {err}")
        finally:
            self._is_connected = False
            for future in self._pending.values():
                if not future.done():
                    future.cancel()

    async def _handle_inbound_message(self, message_data: bytes) -> None:
        """Handle inbound protobuf message from FSM."""
        try:
            msg = FSMToPlugin()
            msg.ParseFromString(message_data)

            # Check if this is a response to our request
            if msg.id in self._pending:
                logger.debug(f"Received FSM response for id: 0x{msg.id:x}")
                future = self._pending.pop(msg.id, None)
                if future and not future.done():
                    future.set_result(msg)
            else:
                # This is a new request from FSM
                await self._handle_fsm_request(msg)

        except Exception as err:
            logger.error(f"Failed to handle inbound FSM message: {err}")

    async def _handle_fsm_request(self, msg: FSMToPlugin) -> None:
        """Handle new request from FSM."""
        try:
            # Create a new contract instance for this request
            contract = Contract(
                config=self.config,
                fsm_config=self.fsm_config,
                plugin=self,
                fsm_id=msg.id,
            )

            response: Optional[PluginToFSM] = None

            # Route the message
            if msg.HasField("config"):
                logger.info("Received FSM config response")
                self.fsm_config = msg.config
                return  # No response needed

            elif msg.HasField("genesis"):
                logger.info("Received genesis request from FSM")
                result = contract.genesis(msg.genesis)
                response = PluginToFSM()
                response.id = msg.id
                response.genesis.CopyFrom(result)

            elif msg.HasField("begin"):
                logger.debug(f"Received begin request from FSM (H:{msg.begin.height})")
                result = contract.begin_block(msg.begin)
                response = PluginToFSM()
                response.id = msg.id
                response.begin.CopyFrom(result)

            elif msg.HasField("check"):
                logger.debug("Received check request from FSM")
                result = await contract.check_tx(msg.check)
                response = PluginToFSM()
                response.id = msg.id
                response.check.CopyFrom(result)

            elif msg.HasField("deliver"):
                logger.debug("Received deliver request from FSM")
                result = await contract.deliver_tx(msg.deliver)
                response = PluginToFSM()
                response.id = msg.id
                response.deliver.CopyFrom(result)

            elif msg.HasField("end"):
                logger.debug(f"Received end request from FSM (H:{msg.end.height})")
                result = contract.end_block(msg.end)
                response = PluginToFSM()
                response.id = msg.id
                response.end.CopyFrom(result)

            else:
                raise err_invalid_fsm_to_plugin_message(type(msg))

            if response:
                await self._send_proto_msg(response)

        except Exception as err:
            logger.error(f"Error handling FSM request: {err}")

    async def _send_proto_msg(self, message: PluginToFSM) -> None:
        """Send length-prefixed protobuf message to FSM."""
        if not self._writer:
            raise err_failed_plugin_write("No writer available")

        try:
            # Serialize message
            message_data = message.SerializeToString()

            # Send length prefix (4 bytes, big-endian) + message
            length_prefix = struct.pack(">I", len(message_data))
            self._writer.write(length_prefix + message_data)
            await self._writer.drain()

        except Exception as err:
            raise err_failed_plugin_write(err)


async def start_plugin(config: Config) -> Plugin:
    """
    Start the plugin (matching Go's StartPlugin).

    Creates and starts a plugin that connects to the FSM socket.
    """
    plugin = Plugin(config)
    await plugin.start()
    return plugin
