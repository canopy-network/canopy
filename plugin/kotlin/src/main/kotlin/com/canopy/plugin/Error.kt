package com.canopy.plugin

/**
 * Plugin error with code, module, and message
 * Matches Go implementation error codes
 */
data class PluginError(
    val code: Int,
    val module: String,
    val msg: String
) : Exception("Module: $module, Code: $code, Message: $msg")

private const val DEFAULT_MODULE = "plugin"

fun ErrPluginTimeout() = PluginError(1, DEFAULT_MODULE, "a plugin timeout occurred")

fun ErrMarshal(err: Exception) = PluginError(2, DEFAULT_MODULE, "marshal() failed with err: ${err.message}")

fun ErrUnmarshal(err: Exception) = PluginError(3, DEFAULT_MODULE, "unmarshal() failed with err: ${err.message}")

fun ErrFailedPluginRead(err: Exception) = PluginError(4, DEFAULT_MODULE, "a plugin read failed with err: ${err.message}")

fun ErrFailedPluginWrite(err: Exception) = PluginError(5, DEFAULT_MODULE, "a plugin write failed with err: ${err.message}")

fun ErrInvalidPluginRespId() = PluginError(6, DEFAULT_MODULE, "plugin response id is invalid")

fun ErrUnexpectedFSMToPlugin(type: String) = PluginError(7, DEFAULT_MODULE, "unexpected FSM to plugin: $type")

fun ErrInvalidFSMToPluginMessage(type: String) = PluginError(8, DEFAULT_MODULE, "invalid FSM to plugin: $type")

fun ErrInsufficientFunds() = PluginError(9, DEFAULT_MODULE, "insufficient funds")

fun ErrFromAny(err: Exception) = PluginError(10, DEFAULT_MODULE, "fromAny() failed with err: ${err.message}")

fun ErrInvalidMessageCast() = PluginError(11, DEFAULT_MODULE, "the message cast failed")

fun ErrInvalidAddress() = PluginError(12, DEFAULT_MODULE, "address is invalid")

fun ErrInvalidAmount() = PluginError(13, DEFAULT_MODULE, "amount is invalid")

fun ErrTxFeeBelowStateLimit() = PluginError(14, DEFAULT_MODULE, "tx.fee is below state limit")
