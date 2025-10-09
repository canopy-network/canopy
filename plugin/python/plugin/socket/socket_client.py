"""
Unix socket client that communicates with Canopy FSM using length-prefixed protobuf messages.

Provides async socket communication with automatic reconnection and message handling.
"""

import asyncio
import os
import socket
import struct
import logging
from enum import Enum
from typing import Optional, Dict, Any, cast
from dataclasses import dataclass

from ..config import Config
from ..core import (
    Contract,
    CONTRACT_CONFIG,
    ContractOptions,
)
from ..core.exceptions import (
    PluginException,
)
from .exceptions import (
    SocketTimeoutError,
    SocketConnectionError,
    InvalidSocketResponseError,
)
from ..proto import (
    FSMToPlugin,
    PluginBeginResponse,
    PluginCheckResponse,
    PluginConfig,
    PluginDeliverResponse,
    PluginEndResponse,
    PluginGenesisResponse,
    PluginStateReadRequest,
    PluginStateReadResponse,
    PluginStateWriteRequest,
    PluginStateWriteResponse,
    PluginToFSM,
)
from ..proto_utils import marshal


class ResponseType(Enum):
    """Enum for response types to ensure type safety."""

    GENESIS = "genesis"
    BEGIN = "begin"
    CHECK = "check"
    DELIVER = "deliver"
    END = "end"


RESPONSE_TYPE_MAP = {
    ResponseType.GENESIS.value: PluginGenesisResponse,
    ResponseType.BEGIN.value: PluginBeginResponse,
    ResponseType.CHECK.value: PluginCheckResponse,
    ResponseType.DELIVER.value: PluginDeliverResponse,
    ResponseType.END.value: PluginEndResponse,
}


@dataclass
class SocketClientOptions:
    """Socket client configuration options."""

    config: Config
    reconnect_interval: float = 3.0
    request_timeout: float = 10.0
    connection_timeout: float = 5.0


class SocketClient:
    """
    Unix socket client for FSM communication with automatic reconnection and message handling.
    """

    def __init__(self, options: SocketClientOptions):
        """Initialize socket client with configuration."""
        self.config = options.config
        self.logger = logging.getLogger("SocketClient")
        self.socket_path = os.path.join(self.config.data_dir_path, "plugin.sock")
        self.reconnect_interval = options.reconnect_interval
        self.request_timeout = options.request_timeout
        self.connection_timeout = options.connection_timeout

        self._sock: Optional[socket.socket] = None
        self._reader: Optional[asyncio.StreamReader] = None
        self._writer: Optional[asyncio.StreamWriter] = None
        self._pending: Dict[int, asyncio.Future] = {}
        self._request_contracts: Dict[int, Contract] = {}
        self._listen_task: Optional[asyncio.Task] = None
        self._message_tasks: set[asyncio.Task] = set()

        self._is_connected = False
        self._is_reconnecting = False
        self._message_id_counter = 1

        # Logger will use the root logger configuration from main.py
        self.logger.setLevel(logging.DEBUG)

    async def start(self) -> None:
        """Start the socket client and connect to FSM."""
        await self._connect_with_retry()
        await self._handshake()
        self._listen_task = asyncio.create_task(self._listen_for_messages())
        self.logger.info("Socket client started and connected to FSM")

    async def close(self) -> None:
        """Close the socket connection gracefully."""
        self._is_connected = False

        # Cancel the listening task
        if self._listen_task and not self._listen_task.done():
            self._listen_task.cancel()
            try:
                await self._listen_task
            except asyncio.CancelledError:
                pass

        # Cancel and cleanup any running message handling tasks
        if self._message_tasks:
            self.logger.debug(
                f"Cancelling {len(self._message_tasks)} message handling tasks"
            )
            for task in self._message_tasks.copy():
                if not task.done():
                    task.cancel()

            # Wait for all tasks to complete or be cancelled
            if self._message_tasks:
                try:
                    await asyncio.gather(*self._message_tasks, return_exceptions=True)
                except Exception as err:
                    self.logger.debug(f"Error during task cleanup: {err}")

        if self._writer:
            self._writer.close()
            await self._writer.wait_closed()

        if self._sock:
            self._sock.close()

        self.logger.info("Socket client closed")

    async def state_read(
        self, contract: Contract, request: PluginStateReadRequest
    ) -> PluginStateReadResponse:
        """
        Send state read request to FSM.

        Args:
            contract: Contract instance making the request
            request: State read request data

        Returns:
            State read response
        """
        # Request is already in protobuf format
        proto_request = request

        # Ensure fsm_id is an int for socket operations
        if not isinstance(contract.fsm_id, int):
            raise ValueError(
                f"Contract fsm_id must be int for socket operations, got {type(contract.fsm_id)}"
            )

        fsm_id = contract.fsm_id

        # Create future before setting up request to avoid race condition
        future: asyncio.Future[PluginStateReadResponse] = asyncio.Future()
        self._pending[fsm_id] = future
        self._request_contracts[fsm_id] = contract

        plugin_message = PluginToFSM()
        plugin_message.id = fsm_id
        plugin_message.state_read.CopyFrom(proto_request)

        # self.logger.debug(f"state_read pending set {self._pending}")
        # self.logger.debug(f"plugin_message {plugin_message}")

        try:
            self.logger.debug(f"0x{fsm_id:x}: state_read")
            await self._send_message(plugin_message)

            # Wait for response with timeout
            response = await asyncio.wait_for(future, timeout=self.request_timeout)

            # Return protobuf response directly
            if response.HasField("state_read"):
                return response.state_read
            else:
                raise InvalidSocketResponseError("state_read")

        except asyncio.TimeoutError:
            self.logger.error(f"State read request 0x{fsm_id:x} timed out")
            raise SocketTimeoutError("State read request", self.request_timeout)
        except Exception:
            # If there's an error, make sure to clean up and mark future as failed
            if fsm_id in self._pending and not future.done():
                future.cancel()
            raise
        finally:
            # Clean up request state
            self._pending.pop(fsm_id, None)
            self._request_contracts.pop(fsm_id, None)

    async def state_write(
        self, contract: Contract, request: PluginStateWriteRequest
    ) -> PluginStateWriteResponse:
        """
        Send state write request to FSM.

        Args:
            contract: Contract instance making the request
            request: State write request data

        Returns:
            State write response
        """
        # Request is already in protobuf format
        proto_request = request

        # Ensure fsm_id is an int for socket operations
        if not isinstance(contract.fsm_id, int):
            raise ValueError(
                f"Contract fsm_id must be int for socket operations, got {type(contract.fsm_id)}"
            )

        fsm_id = contract.fsm_id

        # Create future before setting up request to avoid race condition
        future: asyncio.Future[PluginStateWriteResponse] = asyncio.Future()
        self._pending[fsm_id] = future
        self._request_contracts[fsm_id] = contract

        plugin_message = PluginToFSM()
        plugin_message.id = fsm_id
        plugin_message.state_write.CopyFrom(proto_request)

        try:
            self.logger.debug(f"0x{fsm_id:x}: state_write")
            await self._send_message(plugin_message)

            # Wait for response with timeout
            response = await asyncio.wait_for(future, timeout=self.request_timeout)

            # Return protobuf response directly
            if response.HasField("state_write"):
                return response.state_write
            else:
                raise InvalidSocketResponseError("state_write")

        except asyncio.TimeoutError:
            self.logger.error(f"State write request 0x{fsm_id:x} timed out")
            raise SocketTimeoutError("State write request", self.request_timeout)
        except Exception:
            # If there's an error, make sure to clean up and mark future as failed
            if fsm_id in self._pending and not future.done():
                future.cancel()
            raise
        finally:
            # Clean up request state
            self._pending.pop(fsm_id, None)
            self._request_contracts.pop(fsm_id, None)

    async def _connect_with_retry(self) -> None:
        """Connect to Unix socket with retry logic."""
        if self._is_reconnecting:
            return

        self._is_reconnecting = True

        while not self._is_connected:
            try:
                await self._attempt_connection()
                self._is_reconnecting = False
                return
            except Exception as err:
                self.logger.warning(f"Error connecting to plugin socket: {err}")
                await asyncio.sleep(self.reconnect_interval)

        self._is_reconnecting = False

    async def _attempt_connection(self) -> None:
        """Attempt a single connection with proper error handling."""
        try:
            self._reader, self._writer = await asyncio.wait_for(
                asyncio.open_unix_connection(self.socket_path),
                timeout=self.connection_timeout,
            )

            self._is_connected = True
            self.logger.info(f"Connection established to {self.socket_path}")

        except asyncio.TimeoutError as err:
            if self._writer:
                self._writer.close()
                await self._writer.wait_closed()
            self._is_connected = False
            raise SocketTimeoutError(
                "Connection attempt", self.connection_timeout
            ) from err
        except Exception as err:
            if self._writer:
                self._writer.close()
                await self._writer.wait_closed()
            self._is_connected = False
            raise SocketConnectionError(f"Connection failed: {err}") from err

    async def _handshake(self) -> None:
        """Perform initial handshake with FSM."""
        plugin_config = PluginConfig()
        plugin_config.name = CONTRACT_CONFIG["name"]
        plugin_config.id = CONTRACT_CONFIG["id"]
        plugin_config.version = CONTRACT_CONFIG["version"]

        # Set supported transactions
        supported_transactions = cast(list, CONTRACT_CONFIG["supported_transactions"])
        for tx_type in supported_transactions:
            plugin_config.supported_transactions.append(tx_type)

        message_id = self._get_next_message_id()

        plugin_message = PluginToFSM()
        plugin_message.id = message_id
        plugin_message.config.CopyFrom(plugin_config)

        await self._send_message(plugin_message)
        self.logger.info("Plugin config sent")

    async def _listen_for_messages(self) -> None:
        """Listen for inbound messages from FSM with proper buffer management."""
        if not self._reader:
            raise SocketConnectionError("No reader available for listening")

        try:
            while self._is_connected:
                try:
                    # Read length prefix (4 bytes, big-endian) with timeout
                    length_data = await asyncio.wait_for(
                        self._reader.readexactly(4), timeout=self.request_timeout
                    )
                    message_length = struct.unpack(">I", length_data)[0]

                    # Read message data with timeout
                    message_data = await asyncio.wait_for(
                        self._reader.readexactly(message_length),
                        timeout=self.request_timeout,
                    )

                    # Handle the message concurrently
                    task = asyncio.create_task(
                        self._handle_inbound_message(message_data)
                    )
                    self._message_tasks.add(task)
                    task.add_done_callback(self._message_tasks.discard)

                except asyncio.TimeoutError:
                    # Timeout is normal, just continue listening
                    if not self._is_connected:
                        break
                    continue

        except asyncio.IncompleteReadError:
            self.logger.info("Connection closed by FSM")
        except asyncio.CancelledError:
            self.logger.info("Message listening cancelled")
        except Exception as err:
            self.logger.error(f"Error reading from socket: {err}")
        finally:
            self._is_connected = False
            # Cancel any pending requests
            for future in self._pending.values():
                if not future.done():
                    future.cancel()

    async def _handle_inbound_message(self, message_data: bytes) -> None:
        """Handle inbound protobuf message from FSM."""
        try:
            # Decode FSMToPlugin message
            fsm_message = FSMToPlugin()
            fsm_message.ParseFromString(message_data)

            # self.logger.info(f"handle inbound message pending {self._pending}")

            # Check if this is a response to our request
            if fsm_message.id in self._pending:
                # self.logger.debug(f"Handling FSM response for id: 0x{fsm_message.id:x}")
                future = self._pending.pop(fsm_message.id, None)
                if future and not future.done():
                    future.set_result(fsm_message)
            else:
                # self.logger.debug(f"Handling FSM request for id: 0x{fsm_message.id:x}")
                await self._handle_fsm_request(fsm_message)

        except Exception as err:
            self.logger.error(f"Failed to handle inbound FSM message: {err}")

    async def _handle_fsm_request(self, message: FSMToPlugin) -> None:
        """Handle new request from FSM."""
        try:
            contract = self._create_contract_instance(message.id)
            response = await self._process_request_message(message, contract)

            if response:
                await self._send_response_to_fsm(message.id, response)

        except Exception as err:
            await self._send_error_response(message.id, err)

    async def _process_request_message(
        self, message: FSMToPlugin, contract: Contract
    ) -> Optional[Dict[str, Any]]:
        """Process specific request message types."""

        # Handle different message types
        if message.HasField("config"):
            self.logger.debug(f"0x{message.id:x}: config")
            return None  # No response needed

        if message.HasField("genesis"):
            self.logger.info(f"0x{message.id:x}: genesis")
            result = contract.genesis(message.genesis)
            return {"genesis": result}

        if message.HasField("begin"):
            self.logger.debug(f"0x{message.id:x}: begin_block H:{message.begin.height}")
            result = contract.begin_block(message.begin)
            return {"begin": result}

        if message.HasField("check"):
            self.logger.debug(f"0x{message.id:x}: check_tx")
            result = await contract.check_tx(message.id, message.check)
            return {"check": result}

        if message.HasField("deliver"):
            self.logger.debug(f"0x{message.id:x}: deliver_tx")
            result = await contract.deliver_tx(message.deliver)
            return {"deliver": result}

        if message.HasField("end"):
            self.logger.debug(f"0x{message.id:x}: end_block H:{message.end.height}")
            result = contract.end_block(message.end)
            return {"end": result}

        return None

    async def _send_response_to_fsm(
        self, request_id: int, response: Dict[str, Any]
    ) -> None:
        """Send response back to FSM."""
        plugin_message = PluginToFSM()
        plugin_message.id = request_id

        # Find the response type
        response_type = next(iter(response.keys()))
        response_data = response[response_type]

        # Get the appropriate response class and plugin message field
        response_class = RESPONSE_TYPE_MAP[response_type]
        plugin_response = getattr(plugin_message, response_type)
        plugin_response.CopyFrom(response_class())

        # Handle special case for check response
        if response_type == ResponseType.CHECK.value:
            if response_data.recipient:
                plugin_response.recipient = response_data.recipient
            if response_data.authorized_signers:
                plugin_response.authorized_signers.extend(
                    response_data.authorized_signers
                )

        # Set error if present (common to all response types)
        if response_data.HasField("error"):
            plugin_response.error.CopyFrom(response_data.error)

        await self._send_message(plugin_message)

    async def _send_error_response(self, request_id: int, error: Exception) -> None:
        """Send error response to FSM."""
        try:
            plugin_message = PluginToFSM()
            plugin_message.id = request_id

            # Create error message
            plugin_error = plugin_message.error
            if isinstance(error, PluginException):
                plugin_error.code = error.code
                plugin_error.module = error.module
                plugin_error.msg = error.msg
            else:
                plugin_error.code = 1
                plugin_error.module = "socket_client"
                plugin_error.msg = str(error)

            await self._send_message(plugin_message)
            self.logger.error(
                f"Sent error response for request 0x{request_id:x}: {error}"
            )
        except Exception as send_error:
            self.logger.error(
                f"Failed to send error response for request 0x{request_id:x}: {send_error}"
            )

    async def _send_message(self, message: PluginToFSM) -> None:
        """Send protobuf message to FSM with length prefix."""
        if not self._writer:
            raise SocketConnectionError("No writer available for sending")

        if not self._is_connected:
            raise SocketConnectionError("Socket not connected")

        # Serialize message to bytes
        message_data = marshal(message)

        # Send length prefix (4 bytes, big-endian) followed by message
        length_prefix = struct.pack(">I", len(message_data))
        self._writer.write(length_prefix + message_data)

        # Add timeout to drain operation to prevent blocking
        try:
            await asyncio.wait_for(self._writer.drain(), timeout=self.request_timeout)
        except asyncio.TimeoutError as err:
            self.logger.error("Message send timeout - connection may be blocked")
            self._is_connected = False
            raise SocketTimeoutError("Message send", self.request_timeout) from err

    def _create_contract_instance(self, fsm_id: int) -> Contract:
        """Create contract instance for request processing."""
        options = ContractOptions(config=self.config, plugin=self, fsm_id=fsm_id)
        return Contract(options)

    def _get_next_message_id(self) -> int:
        """Get next unique message ID."""
        current = self._message_id_counter
        self._message_id_counter += 1
        return current
