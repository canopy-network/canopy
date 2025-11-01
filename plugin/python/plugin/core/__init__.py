"""
Core plugin functionality.

Contains the main contract logic, state key management, validation, and core exceptions.
"""

from .contract import Contract, CONTRACT_CONFIG, ContractOptions
from . import keys
from . import validation
from .exceptions import (
    PluginException,
    ValidationError,
    InvalidAddressError,
    InvalidAmountError,
    InsufficientFundsError,
    FeeBelowLimitError,
    UnsupportedMessageTypeError,
    PluginNotInitializedError,
    ParameterError,
    PluginErrorCode,
    create_error_response_from_exception,
    create_check_error_response_from_exception,
    create_deliver_error_response_from_exception,
)

__all__ = [
    "Contract",
    "CONTRACT_CONFIG", 
    "ContractOptions",
    "keys",
    "validation",
    "PluginException",
    "ValidationError",
    "InvalidAddressError",
    "InvalidAmountError",
    "InsufficientFundsError",
    "FeeBelowLimitError",
    "UnsupportedMessageTypeError",
    "PluginNotInitializedError",
    "ParameterError",
    "PluginErrorCode",
    "create_error_response_from_exception",
    "create_check_error_response_from_exception",
    "create_deliver_error_response_from_exception",
]