"""
Socket communication exceptions for the Canopy blockchain plugin.

Provides socket-specific error handling for protocol communication and marshaling operations.
"""

from typing import Union, Optional
from ..core.exceptions import PluginException, PluginErrorCode


class MarshalError(PluginException):
    """Marshal operation failed error."""

    def __init__(self, original_error: Union[Exception, str]):
        message = f"marshal() failed with err: {original_error}"
        super().__init__(message, code=PluginErrorCode.MARSHAL, module="plugin")


class UnmarshalError(PluginException):
    """Unmarshal operation failed error."""

    def __init__(self, original_error: Union[Exception, str]):
        message = f"unmarshal() failed with err: {original_error}"
        super().__init__(message, code=PluginErrorCode.UNMARSHAL, module="plugin")


class SocketTimeoutError(PluginException):
    """Socket request timeout error."""

    def __init__(self, request_type: str = "request", timeout: Optional[float] = None):
        if timeout:
            message = f"{request_type} timed out after {timeout}s"
        else:
            message = f"{request_type} timed out"
        super().__init__(message, code=PluginErrorCode.PLUGIN_TIMEOUT, module="socket")


class SocketConnectionError(PluginException):
    """Socket connection error."""

    def __init__(self, message: str = "Socket connection failed"):
        super().__init__(message, code=1, module="socket")


class InvalidSocketResponseError(PluginException):
    """Invalid socket response error."""

    def __init__(self, expected: str, received: Optional[str] = None):
        if received:
            message = f"Expected {expected} response, but received {received}"
        else:
            message = f"No {expected} response received"
        super().__init__(message, code=1, module="socket")