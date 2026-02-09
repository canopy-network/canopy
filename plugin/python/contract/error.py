"""Contract level PluginErrors matching Go implementation."""

from typing import Any

DEFAULT_MODULE = "plugin"


class PluginError(Exception):
    """Plugin error with code, module, and message."""

    def __init__(self, code: int, module: str, msg: str):
        super().__init__(msg)
        self.code = code
        self.module = module
        self.msg = msg

    def __str__(self) -> str:
        return f"\nModule:  {self.module}\nCode:    {self.code}\nMessage: {self.msg}"


# Error factory functions (matching Go error.go)

def err_plugin_timeout() -> PluginError:
    return PluginError(1, DEFAULT_MODULE, "a plugin timeout occurred")


def err_marshal(err: Any) -> PluginError:
    return PluginError(2, DEFAULT_MODULE, f"marshal() failed with err: {err}")


def err_unmarshal(err: Any) -> PluginError:
    return PluginError(3, DEFAULT_MODULE, f"unmarshal() failed with err: {err}")


def err_failed_plugin_read(err: Any) -> PluginError:
    return PluginError(4, DEFAULT_MODULE, f"a plugin read failed with err: {err}")


def err_failed_plugin_write(err: Any) -> PluginError:
    return PluginError(5, DEFAULT_MODULE, f"a plugin write failed with err: {err}")


def err_invalid_plugin_resp_id() -> PluginError:
    return PluginError(6, DEFAULT_MODULE, "plugin response id is invalid")


def err_unexpected_fsm_to_plugin(t: Any) -> PluginError:
    return PluginError(7, DEFAULT_MODULE, f"unexpected FSM to plugin: {t}")


def err_invalid_fsm_to_plugin_message(t: Any) -> PluginError:
    return PluginError(8, DEFAULT_MODULE, f"invalid FSM to plugin: {t}")


def err_insufficient_funds() -> PluginError:
    return PluginError(9, DEFAULT_MODULE, "insufficient funds")


def err_from_any(err: Any) -> PluginError:
    return PluginError(10, DEFAULT_MODULE, f"fromAny() failed with err: {err}")


def err_invalid_message_cast() -> PluginError:
    return PluginError(11, DEFAULT_MODULE, "the message cast failed")


def err_invalid_address() -> PluginError:
    return PluginError(12, DEFAULT_MODULE, "address is invalid")


def err_invalid_amount() -> PluginError:
    return PluginError(13, DEFAULT_MODULE, "amount is invalid")


def err_tx_fee_below_state_limit() -> PluginError:
    return PluginError(14, DEFAULT_MODULE, "tx.fee is below state limit")
