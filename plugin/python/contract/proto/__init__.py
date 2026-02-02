"""
Protobuf bindings for Canopy Plugin Protocol

This module contains the generated protobuf classes for communication
between the plugin and FSM.
"""

# Import generated protobuf classes
from .account_pb2 import Account, Pool  # type: ignore[attr-defined]
from .event_pb2 import Event, EventCustom  # type: ignore[attr-defined]
from .tx_pb2 import Transaction, MessageSend, FeeParams, Signature  # type: ignore[attr-defined]

# Import plugin proto classes
from .plugin_pb2 import (  # type: ignore[attr-defined]
    # Plugin communication types
    FSMToPlugin,
    PluginToFSM,
    PluginConfig,
    PluginFSMConfig,
    # Request/Response types
    PluginGenesisRequest,
    PluginGenesisResponse,
    PluginBeginRequest,
    PluginBeginResponse,
    PluginCheckRequest,
    PluginCheckResponse,
    PluginDeliverRequest,
    PluginDeliverResponse,
    PluginEndRequest,
    PluginEndResponse,
    PluginError,
    # State management types
    PluginStateReadRequest,
    PluginStateReadResponse,
    PluginStateWriteRequest,
    PluginStateWriteResponse,
    PluginKeyRead,
    PluginRangeRead,
    PluginReadResult,
    PluginSetOp,
    PluginDeleteOp,
    PluginStateEntry,
)

__all__ = [
    # Account types
    "Account",
    "Pool",
    # Event types
    "Event",
    "EventCustom",
    # Transaction types
    "Transaction",
    "MessageSend",
    "FeeParams",
    "Signature",
    # Plugin communication types
    "FSMToPlugin",
    "PluginToFSM",
    "PluginConfig",
    "PluginFSMConfig",
    # Request/Response types
    "PluginGenesisRequest",
    "PluginGenesisResponse",
    "PluginBeginRequest",
    "PluginBeginResponse",
    "PluginCheckRequest",
    "PluginCheckResponse",
    "PluginDeliverRequest",
    "PluginDeliverResponse",
    "PluginEndRequest",
    "PluginEndResponse",
    "PluginError",
    # State management types
    "PluginStateReadRequest",
    "PluginStateReadResponse",
    "PluginStateWriteRequest",
    "PluginStateWriteResponse",
    "PluginKeyRead",
    "PluginRangeRead",
    "PluginReadResult",
    "PluginSetOp",
    "PluginDeleteOp",
    "PluginStateEntry",
]
