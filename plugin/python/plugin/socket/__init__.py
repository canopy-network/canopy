"""
Socket communication module for the Canopy blockchain plugin.

Provides Unix socket client functionality and socket-specific exception handling.
"""

from .socket_client import SocketClient, SocketClientOptions
from .exceptions import (
    MarshalError,
    UnmarshalError,
    SocketTimeoutError,
    SocketConnectionError,
    InvalidSocketResponseError,
)

__all__ = [
    "SocketClient",
    "SocketClientOptions",
    "MarshalError",
    "UnmarshalError",
    "SocketTimeoutError",
    "SocketConnectionError",
    "InvalidSocketResponseError",
]