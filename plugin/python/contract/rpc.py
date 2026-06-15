"""
Custom RPC server for the Canopy Python plugin.

This file contains an EXAMPLE HTTP server that demonstrates how a plugin builder exposes their own
custom RPC endpoints for their chain. Matches Go's contract/rpc.go structure.

Canopy core only exposes a single, generic, read-only transport over the unix socket:
`Plugin.query_state(height, read)`, which returns raw key/value state at a historical height. The
plugin process owns its HTTP server entirely, so builders may register as many routes as they want
and decode their own keys/protobufs into whatever response shapes they like. Canopy never needs to
know about chain-specific endpoints.

The endpoints below are intentionally plugin-specific (faucet and reward records) so they showcase
data that does NOT exist in the Canopy node's own RPC. Account/pool queries already exist in core,
so they make poor examples of a *custom* endpoint.

The server runs in a background thread (Python stdlib http.server, no extra dependencies) and calls
the async `query_state` safely via `asyncio.run_coroutine_threadsafe` against the plugin's event
loop.
"""

import asyncio
import json
import logging
import random
import threading
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
from typing import List, Optional
from urllib.parse import urlparse, parse_qs

from .proto import (
    Faucet,
    Reward,
    PluginStateReadRequest,
    PluginKeyRead,
    PluginRangeRead,
    PluginStateEntry,
)
from .contract import (
    key_for_faucet,
    faucet_prefix,
    key_for_reward,
    reward_prefix,
    unmarshal,
)
from .plugin import Plugin, PLUGIN_BUILD

logger = logging.getLogger(__name__)

# How long an RPC handler waits on the detached query coroutine before giving up
_QUERY_TIMEOUT_SEC = 15.0


def _run_query(plugin: Plugin, height: int, request: PluginStateReadRequest):
    """Run the plugin's async query_state from a (synchronous) RPC handler thread.

    The plugin owns an asyncio event loop on another thread; we schedule the coroutine onto it and
    block for the result, matching Go's synchronous QueryState call from its HTTP handlers.
    """
    if plugin._loop is None:
        raise RuntimeError("plugin event loop is not running")
    coro = plugin.query_state(height, request)
    future = asyncio.run_coroutine_threadsafe(coro, plugin._loop)
    return future.result(timeout=_QUERY_TIMEOUT_SEC)


def _query_value(plugin: Plugin, height: int, key: bytes) -> Optional[bytes]:
    """Perform a single-key detached read and return the raw value bytes (None means 'not found')."""
    resp = _run_query(
        plugin,
        height,
        PluginStateReadRequest(keys=[PluginKeyRead(query_id=random.getrandbits(64), key=key)]),
    )
    if not resp.results or not resp.results[0].entries:
        return None
    return resp.results[0].entries[0].value


def _query_range(plugin: Plugin, height: int, prefix: bytes) -> List[PluginStateEntry]:
    """Perform a detached range read over a key prefix and return the entries."""
    resp = _run_query(
        plugin,
        height,
        PluginStateReadRequest(ranges=[PluginRangeRead(query_id=random.getrandbits(64), prefix=prefix)]),
    )
    if not resp.results:
        return []
    return list(resp.results[0].entries)


def _faucet_to_json(faucet: Faucet) -> dict:
    """Shape a Faucet record into a JSON-friendly dict (hex-encoding addresses)."""
    return {
        "recipientAddress": faucet.recipient_address.hex(),
        "totalAmount": faucet.total_amount,
        "count": faucet.count,
    }


def _reward_to_json(reward: Reward) -> dict:
    """Shape a Reward record into a JSON-friendly dict (hex-encoding addresses)."""
    return {
        "recipientAddress": reward.recipient_address.hex(),
        "lastAdminAddress": reward.last_admin_address.hex(),
        "totalAmount": reward.total_amount,
        "count": reward.count,
    }


def _parse_height(query: dict) -> int:
    """Read the optional 'height' query parameter, defaulting to 0 (latest committed)."""
    raw = query.get("height", [None])[0]
    if raw is None:
        return 0
    try:
        return int(raw)
    except (ValueError, TypeError):
        return 0


class PluginRPCHandler(BaseHTTPRequestHandler):
    """HTTP request handler exposing the plugin's custom, chain-specific RPC endpoints.

    The owning Plugin instance is injected as a class attribute by start_rpc_server().
    """

    plugin: Optional[Plugin] = None

    def do_GET(self) -> None:  # noqa: N802 (http.server API)
        parsed = urlparse(self.path)
        query = parse_qs(parsed.query)
        if parsed.path == "/v1/query/faucets":
            self._handle_query_faucets(query)
        elif parsed.path == "/v1/query/rewards":
            self._handle_query_rewards(query)
        else:
            self._write_json_error(404, "not found")

    def _handle_query_faucets(self, query: dict) -> None:
        """Return faucet records. With ?address=<hex> it returns a single recipient's record;
        otherwise it returns every faucet record via a range read over the faucet prefix.
        """
        height = _parse_height(query)
        addr_hex = query.get("address", [None])[0]
        # optional single-record lookup by recipient address
        if addr_hex:
            try:
                address = bytes.fromhex(addr_hex)
            except ValueError:
                address = b""
            if len(address) != 20:
                self._write_json_error(400, "address must be a 20-byte hex string")
                return
            try:
                value = _query_value(self.plugin, height, key_for_faucet(address))
            except Exception as err:
                self._write_json_error(500, str(err))
                return
            faucet = unmarshal(Faucet, value) or Faucet()
            self._write_json({"faucet": _faucet_to_json(faucet), "height": height})
            return
        # otherwise return all faucet records via a range read
        try:
            entries = _query_range(self.plugin, height, faucet_prefix())
        except Exception as err:
            self._write_json_error(500, str(err))
            return
        faucets = []
        for entry in entries:
            faucet = unmarshal(Faucet, entry.value) or Faucet()
            faucets.append(_faucet_to_json(faucet))
        self._write_json({"faucets": faucets, "count": len(faucets), "height": height})

    def _handle_query_rewards(self, query: dict) -> None:
        """Return reward records. With ?address=<hex> it returns a single recipient's record;
        otherwise it returns every reward record via a range read over the reward prefix.
        """
        height = _parse_height(query)
        addr_hex = query.get("address", [None])[0]
        # optional single-record lookup by recipient address
        if addr_hex:
            try:
                address = bytes.fromhex(addr_hex)
            except ValueError:
                address = b""
            if len(address) != 20:
                self._write_json_error(400, "address must be a 20-byte hex string")
                return
            try:
                value = _query_value(self.plugin, height, key_for_reward(address))
            except Exception as err:
                self._write_json_error(500, str(err))
                return
            reward = unmarshal(Reward, value) or Reward()
            self._write_json({"reward": _reward_to_json(reward), "height": height})
            return
        # otherwise return all reward records via a range read
        try:
            entries = _query_range(self.plugin, height, reward_prefix())
        except Exception as err:
            self._write_json_error(500, str(err))
            return
        rewards = []
        for entry in entries:
            reward = unmarshal(Reward, entry.value) or Reward()
            rewards.append(_reward_to_json(reward))
        self._write_json({"rewards": rewards, "count": len(rewards), "height": height})

    def _write_json(self, body: dict, status: int = 200) -> None:
        """Write a JSON success response."""
        data = json.dumps(body).encode("utf-8")
        self.send_response(status)
        self.send_header("Content-Type", "application/json")
        self.send_header("Content-Length", str(len(data)))
        self.end_headers()
        self.wfile.write(data)

    def _write_json_error(self, status: int, message: str) -> None:
        """Write a JSON error response with the given status code."""
        data = json.dumps({"error": message}).encode("utf-8")
        self.send_response(status)
        self.send_header("Content-Type", "application/json")
        self.send_header("Content-Length", str(len(data)))
        self.end_headers()
        self.wfile.write(data)

    def log_message(self, format: str, *args) -> None:  # noqa: A002 (http.server API)
        """Route default access logging through the module logger at debug level."""
        logger.debug("plugin RPC: %s", format % args)


def start_rpc_server(plugin: Plugin) -> Optional[ThreadingHTTPServer]:
    """Launch the plugin's own HTTP server exposing custom, chain-specific RPC endpoints.

    Builders are free to register any number of routes; each handler uses the detached, read-only
    query_state() path to fetch state snapshots from Canopy. Matches Go's StartRPCServer.

    The server runs in a daemon background thread so it does not block the plugin's event loop. The
    running ThreadingHTTPServer is returned so callers can shut it down if desired.
    """
    addr = plugin.config.rpc_address
    # if no address is configured, the RPC server is disabled
    if not addr:
        logger.info("plugin RPC server disabled (no rpc_address configured)")
        return None

    # resolve host/port from the configured listen address (e.g. "0.0.0.0:50010")
    host, _, port_str = addr.rpartition(":")
    if not host:
        host = "0.0.0.0"
    port = int(port_str)

    # bind the plugin to a dedicated handler subclass so each request can reach query_state()
    handler_cls = type("BoundPluginRPCHandler", (PluginRPCHandler,), {"plugin": plugin})
    server = ThreadingHTTPServer((host, port), handler_cls)

    # log the build marker and the registered routes so the running version is obvious in the log
    logger.info(f"plugin RPC server ({PLUGIN_BUILD}) listening on {addr}")
    logger.info("plugin RPC routes registered: GET /v1/query/faucets, GET /v1/query/rewards")

    thread = threading.Thread(target=server.serve_forever, name="plugin-rpc", daemon=True)
    thread.start()
    return server
