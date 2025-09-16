"""
State key generation functions for the Canopy blockchain plugin.

Provides functions for generating state database keys with proper prefixes.
"""

from typing import Union
from ..proto_utils import join_len_prefix, format_uint64


# State key prefixes
ACCOUNT_PREFIX = b"\x01"  # store key prefix for accounts
POOL_PREFIX = b"\x02"  # store key prefix for pools
PARAMS_PREFIX = b"\x07"  # store key prefix for governance parameters

# Type definitions
Address = Union[bytes, str]
ChainId = Union[int, str]


def key_for_account(address: Address) -> bytes:
    """
    Generate state database key for an account.

    Args:
        address: Account address (must be 20 bytes when converted to bytes)

    Returns:
        State key bytes

    Raises:
        ValueError: If address is invalid
    """
    address_bytes = address if isinstance(address, bytes) else address.encode()
    return join_len_prefix(ACCOUNT_PREFIX, address_bytes)


def key_for_fee_params() -> bytes:
    """
    Generate state database key for governance controlled fee parameters.

    Returns:
        Fee parameters key bytes
    """
    suffix = b"/f/"
    return join_len_prefix(PARAMS_PREFIX, suffix)


def key_for_fee_pool(chain_id: ChainId) -> bytes:
    """
    Generate state database key for fee pool.

    Args:
        chain_id: Chain identifier

    Returns:
        Fee pool key bytes
    """
    chain_id_bytes = format_uint64(chain_id)
    return join_len_prefix(POOL_PREFIX, chain_id_bytes)


