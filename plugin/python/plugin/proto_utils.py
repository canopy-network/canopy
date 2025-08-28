"""
Protobuf utility functions for marshaling and unmarshaling.

Provides utilities for working with protobuf messages and binary data.
"""

import struct
from typing import Union, Optional, Any, Dict, cast, Protocol
from .proto import MessageSend, Transaction


class MessageProtocol(Protocol):
    """Protocol for objects that can be marshaled."""
    def SerializeToString(self) -> bytes: ...


def marshal(message: MessageProtocol) -> bytes:
    """
    Marshal object to protobuf bytes.
    
    Args:
        message: Protobuf message to marshal
        
    Returns:
        Serialized bytes
        
    Raises:
        ValueError: If marshaling fails
    """
    try:
        if hasattr(message, 'SerializeToString'):
            return message.SerializeToString()
        else:
            # Fallback for simple dict-like objects
            if hasattr(message, 'to_dict'):
                import json
                return json.dumps(message.to_dict()).encode()
            else:
                import json
                return json.dumps(message).encode()
    except Exception as err:
        raise ValueError(f"Marshal failed: {err}")


def unmarshal(message_type: Any, data: Optional[bytes]) -> Optional[Any]:
    """
    Unmarshal bytes to protobuf message.
    
    Args:
        message_type: Protobuf message type class
        data: Bytes to unmarshal
        
    Returns:
        Unmarshaled message or None if data is empty
        
    Raises:
        ValueError: If unmarshaling fails
    """
    try:
        if not data:
            return None
            
        if hasattr(message_type, 'FromString'):
            return message_type.FromString(data)
        else:
            # Fallback for simple cases
            import json
            return json.loads(data.decode())
    except Exception as err:
        raise ValueError(f"Unmarshal failed: {err}")


def from_any(any_message: Dict[str, Any]) -> Union[MessageSend, Transaction]:
    """
    Convert protobuf Any type to concrete message.
    
    Args:
        any_message: Any message dict with type_url and value
        
    Returns:
        Concrete message instance
        
    Raises:
        ValueError: If conversion fails or unknown type
    """
    try:
        if not any_message:
            raise ValueError("Any message is null or undefined")
        
        # Handle both typeUrl and type_url field names
        type_url = any_message.get('typeUrl') or any_message.get('type_url')
        value = any_message.get('value')
        
        if not type_url:
            raise ValueError("Any message missing type URL")
        
        if not value:
            raise ValueError("Any message missing value")
        
        # Extract message type from URL
        type_name = type_url.split('/')[-1] if '/' in type_url else type_url
        
        # Map known message types
        if type_name in ('MessageSend', 'types.MessageSend'):
            msg = unmarshal(MessageSend, value)
            if not msg:
                raise ValueError("Failed to unmarshal MessageSend")
            return msg
        elif type_name in ('Transaction', 'types.Transaction'):
            msg = unmarshal(Transaction, value)
            if not msg:
                raise ValueError("Failed to unmarshal Transaction")
            return msg
        else:
            raise ValueError(f"Unknown message type in Any: {type_name}")
    except Exception as err:
        raise ValueError(f"FromAny failed: {err}")


def join_len_prefix(*items: Optional[bytes]) -> bytes:
    """
    Join byte arrays with length prefixes.
    
    Args:
        *items: Byte arrays to join
        
    Returns:
        Concatenated bytes with length prefixes
    """
    result = bytearray()
    
    for item in items:
        if not item:
            continue
        
        # Write length byte (max 255 bytes per item)
        if len(item) > 255:
            raise ValueError(f"Item too long: {len(item)} bytes (max 255)")
        
        result.append(len(item))
        result.extend(item)
    
    return bytes(result)


def format_uint64(value: Union[int, str]) -> bytes:
    """
    Format uint64 as big-endian bytes.
    
    Args:
        value: Integer value to format
        
    Returns:
        8-byte big-endian representation
    """
    if isinstance(value, str):
        value = int(value)
    
    if not isinstance(value, int) or value < 0 or value >= (1 << 64):
        raise ValueError(f"Invalid uint64 value: {value}")
    
    return struct.pack('>Q', value)