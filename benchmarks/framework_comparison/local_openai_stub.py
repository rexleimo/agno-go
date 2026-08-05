#!/usr/bin/env python3
"""Serve a deterministic OpenAI-compatible chat completion for local benchmarks."""

from __future__ import annotations

import argparse
import json
import time
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer


class StubHandler(BaseHTTPRequestHandler):
    server_version = "HNOBenchmarkStub/1.0"
    protocol_version = "HTTP/1.1"

    def do_GET(self) -> None:  # noqa: N802 - stdlib handler API
        if self.path == "/health":
            self._send_json({"status": "ok"})
            return
        self.send_error(404)

    def do_POST(self) -> None:  # noqa: N802 - stdlib handler API
        if self.path != "/v1/chat/completions":
            self.send_error(404)
            return
        length = int(self.headers.get("Content-Length", "0"))
        if length:
            self.rfile.read(length)
        delay = self.server.delay_ms / 1000
        if delay:
            time.sleep(delay)
        self.server.request_count += 1
        body = {
            "id": f"stub-{self.server.request_count}",
            "object": "chat.completion",
            "created": int(time.time()),
            "model": self.server.model,
            "choices": [
                {
                    "index": 0,
                    "message": {"role": "assistant", "content": self.server.response_text},
                    "finish_reason": "stop",
                }
            ],
            "usage": {"prompt_tokens": 1, "completion_tokens": 1, "total_tokens": 2},
        }
        self._send_json(body)

    def log_message(self, format: str, *args: object) -> None:
        return

    def _send_json(self, body: dict) -> None:
        encoded = json.dumps(body).encode("utf-8")
        self.send_response(200)
        self.send_header("Content-Type", "application/json")
        self.send_header("Content-Length", str(len(encoded)))
        self.send_header("Connection", "keep-alive")
        self.end_headers()
        self.wfile.write(encoded)


class StubServer(ThreadingHTTPServer):
    daemon_threads = True
    allow_reuse_address = True
    request_queue_size = 256

    def __init__(self, address: tuple[str, int], model: str, response_text: str, delay_ms: float):
        super().__init__(address, StubHandler)
        self.model = model
        self.response_text = response_text
        self.delay_ms = delay_ms
        self.request_count = 0

    def handle_error(self, request, client_address) -> None:
        return


def main() -> None:
    parser = argparse.ArgumentParser()
    parser.add_argument("--host", default="127.0.0.1")
    parser.add_argument("--port", type=int, default=18081)
    parser.add_argument("--model", default="stub-model")
    parser.add_argument("--response", default="LOCAL_MODEL_OK")
    parser.add_argument("--delay-ms", type=float, default=0)
    args = parser.parse_args()
    server = StubServer((args.host, args.port), args.model, args.response, args.delay_ms)
    print(f"stub listening on http://{args.host}:{args.port}", flush=True)
    try:
        server.serve_forever()
    except KeyboardInterrupt:
        pass
    finally:
        server.shutdown()
        server.server_close()


if __name__ == "__main__":
    main()
