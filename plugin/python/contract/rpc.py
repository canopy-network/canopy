"""
Skeleton RPC server for the Canopy Python plugin.

This file ships the plugin's own HTTP server, but it registers NO routes by default. It is a blank
canvas that plugin builders extend with their own custom, chain-specific RPC endpoints. Matches
Go's contract/rpc.go skeleton structure.

DESIGN
------
Canopy core only exposes a single, generic, read-only transport over the unix socket:
`Plugin.query_state(height, read)` (see contract/plugin.py). It is DETACHED (not tied to any
in-flight tx/block lifecycle) and READ-ONLY: it returns raw key/value state at a historical height
(0 = latest committed). Canopy never needs to know about chain-specific endpoints.

The plugin process owns its HTTP server entirely. Builders may register as many routes as they want
and decode their own keys/protobufs into whatever response shapes they like, each handler backed by
the detached, read-only `query_state()` path. No routes are registered here by default.

For a complete, worked example (faucet/reward records) showing how to add routes + handlers to this
skeleton, see TUTORIAL.md ("Custom RPC endpoints").

The server runs in a background thread (Python stdlib http.server, no extra dependencies). Handlers
call the async `query_state` safely via `asyncio.run_coroutine_threadsafe` against the plugin's
event loop.

EXAMPLE — registering a custom route backed by query_state
----------------------------------------------------------
A builder would extend the handler below with their own path, e.g.::

    # 1) import the proto request types and your own keys/messages
    from .proto import PluginStateReadRequest, PluginKeyRead
    from .contract import key_for_myrecord, unmarshal
    from .proto import MyRecord

    # 2) route the path inside PluginRPCHandler.do_GET
    #    if parsed.path == "/v1/query/myrecords":
    #        self._handle_query_myrecords(parse_qs(parsed.query))

    # 3) implement the handler using the detached, read-only query_state() path
    #    def _handle_query_myrecords(self, query: dict) -> None:
    #        coro = self.plugin.query_state(
    #            0,  # height (0 = latest committed)
    #            PluginStateReadRequest(
    #                keys=[PluginKeyRead(query_id=random.getrandbits(64), key=key_for_myrecord(addr))]
    #            ),
    #        )
    #        future = asyncio.run_coroutine_threadsafe(coro, self.plugin._loop)
    #        resp = future.result(timeout=15.0)
    #        ...  # decode resp, unmarshal(MyRecord, value), and self._write_json(...)
"""

import asyncio
import json
import logging
import threading
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
from typing import Optional
from urllib.parse import urlparse, parse_qs

from .plugin import Plugin, PLUGIN_BUILD
from .proto import PluginStateReadRequest, PluginKeyRead, PluginRangeRead
from .proto import Faucet, Reward
from .contract import key_for_faucet, key_for_reward, FAUCET_PREFIX, REWARD_PREFIX, unmarshal

logger = logging.getLogger(__name__)


class PluginRPCHandler(BaseHTTPRequestHandler):
    """HTTP request handler for the plugin's custom, chain-specific RPC endpoints.

    No routes are registered by default: every request returns 404. Builders add their own routes
    here (see the example in this module's docstring and TUTORIAL.md). The owning Plugin instance is
    injected as a class attribute by start_rpc_server() so handlers can reach query_state().
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

    def _handle_query_faucets(self, query: dict) -> None:
        plugin = self.plugin
        if not plugin:
            self._write_json_error(500, "plugin not initialized")
            return

        height = int(query.get("height", [0])[0])
        addresses = query.get("address")

        if addresses:
            addr_bytes = bytes.fromhex(addresses[0])
            key = key_for_faucet(addr_bytes)
            request = PluginStateReadRequest(keys=[PluginKeyRead(query_id=0, key=key)])
        else:
            request = PluginStateReadRequest(
                ranges=[PluginRangeRead(query_id=0, prefix=FAUCET_PREFIX, limit=0, reverse=False)]
            )

        coro = plugin.query_state(height, request)
        future = asyncio.run_coroutine_threadsafe(coro, plugin._loop)

        try:
            resp = future.result(timeout=15.0)
        except Exception as e:
            self._write_json_error(500, str(e))
            return

        if addresses:
            faucet = None
            if resp.results and resp.results[0].entries:
                faucet_bytes = resp.results[0].entries[0].value
                if faucet_bytes:
                    faucet_pb = Faucet()
                    faucet_pb.ParseFromString(faucet_bytes)
                    faucet = {
                        "recipientAddress": faucet_pb.recipient_address.hex(),
                        "totalAmount": faucet_pb.total_amount,
                        "count": faucet_pb.count,
                    }
            self._write_json({
                "faucet": faucet or {"recipientAddress": addresses[0], "totalAmount": 0, "count": 0},
                "height": height,
            })
        else:
            faucets = []
            for result in resp.results:
                for entry in result.entries:
                    if entry.value:
                        faucet_pb = Faucet()
                        faucet_pb.ParseFromString(entry.value)
                        faucets.append({
                            "recipientAddress": faucet_pb.recipient_address.hex(),
                            "totalAmount": faucet_pb.total_amount,
                            "count": faucet_pb.count,
                        })
            self._write_json({"faucets": faucets, "count": len(faucets), "height": height})

    def _handle_query_rewards(self, query: dict) -> None:
        plugin = self.plugin
        if not plugin:
            self._write_json_error(500, "plugin not initialized")
            return

        height = int(query.get("height", [0])[0])
        addresses = query.get("address")

        if addresses:
            addr_bytes = bytes.fromhex(addresses[0])
            key = key_for_reward(addr_bytes)
            request = PluginStateReadRequest(keys=[PluginKeyRead(query_id=0, key=key)])
        else:
            request = PluginStateReadRequest(
                ranges=[PluginRangeRead(query_id=0, prefix=REWARD_PREFIX, limit=0, reverse=False)]
            )

        coro = plugin.query_state(height, request)
        future = asyncio.run_coroutine_threadsafe(coro, plugin._loop)

        try:
            resp = future.result(timeout=15.0)
        except Exception as e:
            self._write_json_error(500, str(e))
            return

        if addresses:
            reward = None
            if resp.results and resp.results[0].entries:
                reward_bytes = resp.results[0].entries[0].value
                if reward_bytes:
                    reward_pb = Reward()
                    reward_pb.ParseFromString(reward_bytes)
                    reward = {
                        "recipientAddress": reward_pb.recipient_address.hex(),
                        "lastAdminAddress": reward_pb.last_admin_address.hex(),
                        "totalAmount": reward_pb.total_amount,
                        "count": reward_pb.count,
                    }
            self._write_json({
                "reward": reward or {"recipientAddress": addresses[0], "lastAdminAddress": "", "totalAmount": 0, "count": 0},
                "height": height,
            })
        else:
            rewards = []
            for result in resp.results:
                for entry in result.entries:
                    if entry.value:
                        reward_pb = Reward()
                        reward_pb.ParseFromString(entry.value)
                        rewards.append({
                            "recipientAddress": reward_pb.recipient_address.hex(),
                            "lastAdminAddress": reward_pb.last_admin_address.hex(),
                            "totalAmount": reward_pb.total_amount,
                            "count": reward_pb.count,
                        })
            self._write_json({"rewards": rewards, "count": len(rewards), "height": height})

    def log_message(self, format: str, *args) -> None:  # noqa: A002 (http.server API)
        """Route default access logging through the module logger at debug level."""
        logger.debug("plugin RPC: %s", format % args)


def start_rpc_server(plugin: Plugin) -> Optional[ThreadingHTTPServer]:
    """Launch the plugin's own HTTP server. By default it registers NO routes.

    Builders are free to register any number of routes on PluginRPCHandler; each handler should use
    the detached, read-only query_state() path to fetch state snapshots from Canopy. Matches Go's
    StartRPCServer. See TUTORIAL.md for a worked example.

    The server runs in a daemon background thread so it does not block the plugin's event loop. The
    running ThreadingHTTPServer is returned so callers can shut it down if desired.
    """
    addr = plugin.config.rpc_address
    # if no address is configured, the RPC server is disabled
    if not addr:
        logger.info("plugin RPC server disabled (no rpc_address configured)")
        return None

    # The custom RPC server is OPTIONAL. Parsing the address or binding the port (e.g. already in
    # use) can fail; a failure here must NOT crash the plugin — log it and continue without an RPC
    # server so plugins that don't use this feature are unaffected.
    try:
        # resolve host/port from the configured listen address (e.g. "0.0.0.0:50010")
        host, _, port_str = addr.rpartition(":")
        if not host:
            host = "0.0.0.0"
        port = int(port_str)

        # bind the plugin to a dedicated handler subclass so each request can reach query_state()
        handler_cls = type("BoundPluginRPCHandler", (PluginRPCHandler,), {"plugin": plugin})
        server = ThreadingHTTPServer((host, port), handler_cls)
    except (OSError, ValueError) as exc:
        logger.warning(f"plugin RPC server disabled (failed to start on {addr!r}): {exc}")
        return None

    # log the build marker so the running version is obvious in the log; no routes are registered
    logger.info(f"plugin RPC server ({PLUGIN_BUILD}) listening on {addr}")
    logger.info("plugin RPC routes registered: none (skeleton — add your own; see TUTORIAL.md)")

    thread = threading.Thread(target=server.serve_forever, name="plugin-rpc", daemon=True)
    thread.start()
    return server
