"""
Ortoplex Gate — G2 ZeroPerceptron RPC endpoints for Canopy plugin.

Implements custom, chain-specific RPC routes backed by the detached, read-only
query_state() path. Exposes 4 endpoints:

  GET /v1/predict?text=...&numbers=[...] — run on-chain inference
  GET /v1/dashboard — read aggregated metrics (predictions, accuracy, revenue)
  GET /v1/models — list model versions + active model
  GET /v1/feedback?address=...&seq=... — export learning signal logs

Each handler uses PluginStateReadRequest to query historical state at height 0
(latest committed). Handlers are completely detached from block lifecycle and
fully deterministic.

DESIGN: Builders may extend with additional routes; all state queries go through
the plugin's event loop via asyncio.run_coroutine_threadsafe() for thread safety.
"""

import asyncio
import json
import logging
import random
import threading
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
from typing import Optional, Dict, Any
from urllib.parse import urlparse, parse_qs

from .plugin import Plugin, PLUGIN_BUILD
from .proto import PluginStateReadRequest, PluginKeyRead
from .contract import (
    key_for_predict_log,
    key_for_feedback_log,
    key_for_dashboard,
    key_for_model_registry,
    key_for_active_model,
    unmarshal,
)
from .ai_model import get_model

logger = logging.getLogger(__name__)


class PluginRPCHandler(BaseHTTPRequestHandler):
    """HTTP request handler for Ortoplex Gate G2 ZeroPerceptron endpoints.

    Routes:
      GET /v1/predict?text=...&numbers=[...] → {"y": int, "class": str, "probs": [...]}
      GET /v1/dashboard → {"total_predictions": ..., "accuracy": ..., ...}
      GET /v1/models → {"active": int, "versions": [...]}
      GET /v1/feedback?address=...&seq=... → {"predict_seq": ..., "correct": bool, ...}
    """

    plugin: Optional[Plugin] = None

    def do_GET(self) -> None:  # noqa: N802 (http.server API)
        parsed = urlparse(self.path)
        path = parsed.path
        query = parse_qs(parsed.query)

        try:
            if path == "/v1/predict":
                self._handle_predict(query)
            elif path == "/v1/dashboard":
                self._handle_dashboard(query)
            elif path == "/v1/models":
                self._handle_models(query)
            elif path == "/v1/feedback":
                self._handle_feedback(query)
            elif path == "/v1/health":
                self._write_json({"status": "ok"}, 200)
            else:
                self._write_json_error(404, "not found")
        except Exception as e:
            logger.exception(f"RPC handler error: {e}")
            self._write_json_error(500, str(e))

    def _handle_predict(self, query: Dict[str, list]) -> None:
        """
        GET /v1/predict?text=<text>&numbers=[<n1>,<n2>,...] → inference result.

        Query params:
          text (optional): event text for encoding
          numbers (optional): JSON array of floats for numeric encoding
          timestamp (optional): unix timestamp for temporal encoding

        Response:
          {
            "y": int,
            "class": str,
            "logits": [...],
            "probs": [...],
            "top3": [{"class": str, "prob": float}, ...],
            "feature_importance": [...],
            "temperature": float,
            "vector": [...],  # 42D encoded vector
          }
        """
        text = query.get("text", [""])[0]
        timestamp = float(query.get("timestamp", [0.0])[0]) if query.get("timestamp") else 0.0

        numbers = []
        if "numbers" in query:
            try:
                numbers = json.loads(query["numbers"][0])
                if not isinstance(numbers, list):
                    numbers = []
            except (json.JSONDecodeError, ValueError):
                numbers = []

        model = get_model()
        model.reset()
        result = model.predict_from_event(text=text, numbers=numbers, timestamp=timestamp)

        self._write_json(result, 200)

    def _handle_dashboard(self, query: Dict[str, list]) -> None:
        """
        GET /v1/dashboard → read on-chain dashboard metrics.

        Dashboard is a single JSON record under DASHBOARD_PREFIX (namespace 107)
        that aggregates:
          - total_predictions: cumulative count
          - class_counts: {class_id: count}
          - accuracy: correct_feedback / total_feedback
          - revenue: total fees collected
          - total_feedback, correct_feedback: on-chain learning signals
          - total_staked, total_markets, resolved_markets: prediction market stats
          - total_rewards: cumulative reward payouts
          - total_models, active_model: model registry stats

        Response: {"dashboard": {...}} or error
        """
        if not self.plugin:
            self._write_json_error(500, "plugin not initialized")
            return

        dashboard_key = key_for_dashboard()
        query_id = random.randint(0, 2**53)

        coro = self.plugin.query_state(
            0,  # height 0 = latest committed
            PluginStateReadRequest(
                keys=[PluginKeyRead(query_id=query_id, key=dashboard_key)]
            ),
        )

        try:
            future = asyncio.run_coroutine_threadsafe(coro, self.plugin._loop)
            resp = future.result(timeout=15.0)

            dashboard = {}
            for result in resp.results:
                if result.query_id == query_id and result.entries:
                    dashboard = json.loads(result.entries[0].value.decode("utf-8"))

            self._write_json({"dashboard": dashboard}, 200)
        except asyncio.TimeoutError:
            self._write_json_error(504, "query_state timeout")
        except Exception as e:
            logger.exception(f"dashboard query error: {e}")
            self._write_json_error(500, str(e))

    def _handle_models(self, query: Dict[str, list]) -> None:
        """
        GET /v1/models → read model registry + active model pointer.

        Queries:
          - Active model version (MODEL_PREFIX 105, key='/active/')
          - Optionally list model versions if /v1/models?list=1

        Response: {"active": int, "versions": [...]}
        """
        if not self.plugin:
            self._write_json_error(500, "plugin not initialized")
            return

        active_key = key_for_active_model()
        query_id = random.randint(0, 2**53)

        coro = self.plugin.query_state(
            0,
            PluginStateReadRequest(
                keys=[PluginKeyRead(query_id=query_id, key=active_key)]
            ),
        )

        try:
            future = asyncio.run_coroutine_threadsafe(coro, self.plugin._loop)
            resp = future.result(timeout=15.0)

            active_model = None
            for result in resp.results:
                if result.query_id == query_id and result.entries:
                    active_model = int(result.entries[0].value.decode("utf-8"))

            response = {
                "active": active_model,
                "versions": [],  # TODO: scan MODEL_PREFIX range if list=1
            }
            self._write_json(response, 200)
        except asyncio.TimeoutError:
            self._write_json_error(504, "query_state timeout")
        except Exception as e:
            logger.exception(f"models query error: {e}")
            self._write_json_error(500, str(e))

    def _handle_feedback(self, query: Dict[str, list]) -> None:
        """
        GET /v1/feedback?address=<hex>&seq=<uint64> → export learning signal.

        Feedback logs are stored under FEEDBACK_PREFIX (namespace 101)
        keyed by address + predict_seq. Each entry tracks:
          - predict_seq: original prediction sequence number
          - correct: whether the prediction was correct
          - actual_class: ground truth class (0-15)
          - height: block height recorded

        Query params:
          address (required): 20-byte hex address
          seq (required): prediction sequence number

        Response: {"feedback": {...}} or error
        """
        if not self.plugin:
            self._write_json_error(500, "plugin not initialized")
            return

        address_str = query.get("address", [""])[0]
        seq_str = query.get("seq", ["0"])[0]

        if not address_str or not seq_str:
            self._write_json_error(400, "missing address or seq")
            return

        try:
            address_bytes = bytes.fromhex(address_str)
            if len(address_bytes) != 20:
                raise ValueError("address must be 20 bytes")
            seq = int(seq_str)
        except (ValueError, TypeError) as e:
            self._write_json_error(400, f"invalid address/seq: {e}")
            return

        feedback_key = key_for_feedback_log(address_bytes, seq)
        query_id = random.randint(0, 2**53)

        coro = self.plugin.query_state(
            0,
            PluginStateReadRequest(
                keys=[PluginKeyRead(query_id=query_id, key=feedback_key)]
            ),
        )

        try:
            future = asyncio.run_coroutine_threadsafe(coro, self.plugin._loop)
            resp = future.result(timeout=15.0)

            feedback = None
            for result in resp.results:
                if result.query_id == query_id and result.entries:
                    feedback = json.loads(result.entries[0].value.decode("utf-8"))

            if feedback is None:
                self._write_json_error(404, f"feedback not found for seq={seq}")
                return

            self._write_json({"feedback": feedback}, 200)
        except asyncio.TimeoutError:
            self._write_json_error(504, "query_state timeout")
        except Exception as e:
            logger.exception(f"feedback query error: {e}")
            self._write_json_error(500, str(e))

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
    """Launch the plugin's HTTP server with Ortoplex Gate G2 endpoints.

    Registers 4 routes:
      - GET /v1/predict — on-chain inference
      - GET /v1/dashboard — aggregated metrics
      - GET /v1/models — model registry
      - GET /v1/feedback — learning signal export

    The server runs in a daemon background thread. Returns the ThreadingHTTPServer
    instance for shutdown control.
    """
    addr = plugin.config.rpc_address
    if not addr:
        logger.info("plugin RPC server disabled (no rpc_address configured)")
        return None

    try:
        host, _, port_str = addr.rpartition(":")
        if not host:
            host = "0.0.0.0"
        port = int(port_str)

        handler_cls = type("BoundPluginRPCHandler", (PluginRPCHandler,), {"plugin": plugin})
        server = ThreadingHTTPServer((host, port), handler_cls)
    except (OSError, ValueError) as exc:
        logger.warning(f"plugin RPC server disabled (failed to start on {addr!r}): {exc}")
        return None

    logger.info(f"plugin RPC server ({PLUGIN_BUILD}) listening on {addr}")
    logger.info("plugin RPC routes registered: /v1/predict, /v1/dashboard, /v1/models, /v1/feedback (Ortoplex Gate G2 ZeroPerceptron)")

    thread = threading.Thread(target=server.serve_forever, name="plugin-rpc", daemon=True)
    thread.start()
    return server
