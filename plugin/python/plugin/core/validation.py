"""
Data validation and normalization functions for the Canopy blockchain plugin.

Provides functions for validating and converting address and amount formats.
"""

from typing import Union

# Type definitions
Address = Union[bytes, str]
Amount = Union[int, str]


def validate_address(address: Address) -> bool:
    """
    Validate that an address is exactly 20 bytes.
    Used in transaction validation.

    Args:
        address: Address to validate

    Returns:
        True if address is valid (exactly 20 bytes)
    """
    try:
        if isinstance(address, str):
            # Handle hex strings
            if address.startswith("0x"):
                address_bytes = bytes.fromhex(address[2:])
            else:
                address_bytes = address.encode()
        else:
            address_bytes = address

        return len(address_bytes) == 20
    except Exception:
        import traceback

        traceback.print_exc()
        return False


def validate_amount(amount: Amount) -> bool:
    """
    Validate that an amount is greater than 0.
    Used in transaction validation.

    Args:
        amount: Amount to validate

    Returns:
        True if amount is valid (greater than 0)
    """
    try:
        if isinstance(amount, str):
            amount_int = int(amount)
        else:
            amount_int = amount

        return isinstance(amount_int, int) and amount_int > 0
    except (ValueError, TypeError):
        return False


def normalize_address(address: Address) -> bytes:
    """
    Convert various address formats to bytes.

    Args:
        address: Address in various formats

    Returns:
        Bytes representation of the address

    Raises:
        ValueError: If address cannot be converted or is invalid length
    """
    if not validate_address(address):
        raise ValueError("Invalid address: must be exactly 20 bytes")

    if isinstance(address, str):
        if address.startswith("0x"):
            return bytes.fromhex(address[2:])
        else:
            return address.encode()

    return address


def normalize_amount(amount: Amount) -> int:
    """
    Convert various amount formats to int for arithmetic.

    Args:
        amount: Amount in various formats

    Returns:
        Int representation of the amount

    Raises:
        ValueError: If amount cannot be converted or is invalid
    """
    if not validate_amount(amount):
        raise ValueError("Invalid amount: must be greater than 0")

    if isinstance(amount, str):
        return int(amount)

    return amount