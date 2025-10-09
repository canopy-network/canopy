"""
Core plugin exceptions for contract validation and execution.

Provides standardized error codes and messages for contract-related operations.
"""

from typing import Union, Optional
from ..proto import PluginError, PluginCheckResponse, PluginDeliverResponse

DEFAULT_MODULE = "plugin"


class PluginException(Exception):
    """
    Base exception for all plugin errors.

    Follows Python best practices while maintaining protobuf error code compatibility.
    """

    def __init__(self, message: str, code: int = 1, module: str = DEFAULT_MODULE):
        super().__init__(message)
        self.code = code
        self.module = module
        self.msg = message  # For protobuf compatibility

    def to_protobuf(self) -> PluginError:
        """Convert to protobuf PluginError for FSM communication."""
        error = PluginError()
        error.code = self.code
        error.module = self.module
        error.msg = self.msg
        return error


class ValidationError(PluginException):
    """Base class for validation errors."""

    pass


class InvalidAddressError(ValidationError):
    """Invalid address format error."""

    def __init__(self, address: Optional[bytes] = None):
        address_str = address.hex() if address else "unknown"
        message = f"Invalid address format: {address_str}"
        super().__init__(
            message, code=PluginErrorCode.INVALID_ADDRESS, module="contract"
        )


class InvalidAmountError(ValidationError):
    """Invalid amount error."""

    def __init__(self, amount: Optional[Union[int, str]] = None):
        message = (
            f"Invalid amount: {amount}" if amount is not None else "Invalid amount"
        )
        super().__init__(
            message, code=PluginErrorCode.INVALID_AMOUNT, module="contract"
        )


class InsufficientFundsError(PluginException):
    """Insufficient funds error."""

    def __init__(self, required: Optional[int] = None, available: Optional[int] = None):
        if required is not None and available is not None:
            message = f"Insufficient funds: required {required}, available {available}"
        else:
            message = "Insufficient funds"
        super().__init__(
            message, code=PluginErrorCode.INSUFFICIENT_FUNDS, module="contract"
        )


class FeeBelowLimitError(ValidationError):
    """Transaction fee below state minimum error."""

    def __init__(self, fee: Optional[int] = None, minimum: Optional[int] = None):
        if fee is not None and minimum is not None:
            message = f"Transaction fee {fee} is below state minimum {minimum}"
        else:
            message = "Transaction fee is below state minimum"
        super().__init__(
            message, code=PluginErrorCode.TX_FEE_BELOW_STATE_LIMIT, module="contract"
        )


class UnsupportedMessageTypeError(PluginException):
    """Unsupported message type error."""

    def __init__(self, message_type: str):
        message = f"Unsupported message type: {message_type}"
        super().__init__(message, code=1, module="contract")


class PluginNotInitializedError(PluginException):
    """Plugin or configuration not initialized error."""

    def __init__(self) -> None:
        super().__init__("Plugin or config not initialized", code=1, module="contract")


class ParameterError(PluginException):
    """Parameter-related errors."""

    def __init__(self, message: str):
        super().__init__(message, code=1, module="contract")


class PluginErrorCode:
    """Error code constants for plugin errors."""

    PLUGIN_TIMEOUT = 1
    MARSHAL = 2
    UNMARSHAL = 3
    FAILED_PLUGIN_READ = 4
    FAILED_PLUGIN_WRITE = 5
    INVALID_PLUGIN_RESP_ID = 6
    UNEXPECTED_FSM_TO_PLUGIN = 7
    INVALID_FSM_TO_PLUGIN_MESSAGE = 8
    INSUFFICIENT_FUNDS = 9
    FROM_ANY = 10
    INVALID_MESSAGE_CAST = 11
    INVALID_ADDRESS = 12
    INVALID_AMOUNT = 13
    TX_FEE_BELOW_STATE_LIMIT = 14


# Response Helper Functions


def create_error_response_from_exception(
    response_class: type, exception: PluginException
) -> object:
    """
    Create a response with error from a PluginException.

    Args:
        response_class: The response class to instantiate
        exception: PluginException to convert to response

    Returns:
        Response instance with error field set
    """
    response = response_class()
    response.error.CopyFrom(exception.to_protobuf())
    return response


def create_check_error_response_from_exception(
    exception: PluginException,
) -> PluginCheckResponse:
    """Create PluginCheckResponse with error from exception."""
    return create_error_response_from_exception(PluginCheckResponse, exception)


def create_deliver_error_response_from_exception(
    exception: PluginException,
) -> PluginDeliverResponse:
    """Create PluginDeliverResponse with error from exception."""
    return create_error_response_from_exception(PluginDeliverResponse, exception)