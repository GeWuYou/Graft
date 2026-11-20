#!/usr/bin/env python3
"""Serve the built web production bundle with a minimal backend reverse proxy.

Mirrors the graft-web production proxy surface (the /api prefix, websocket
paths and root-level HTTP paths from web/vite.config.ts) so browser
acceptance suites drive the production bundle against a real backend without
a container image. Standard library only; run as a foreground process.
"""
from __future__ import annotations

import argparse
from http.client import HTTPConnection
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
import os
from pathlib import Path
import socket
import sys
import threading
from urllib.parse import urlsplit

API_PREFIX = "/api"
WEBSOCKET_PATHS = ("/ws",)
ROOT_HTTP_PROXY_PATHS = (
    "/healthz",
    "/docs",
    "/openapi.json",
    "/openapi.yaml",
    "/mcp/docs",
    "/mcp/docs.json",
    "/sse",
)
HOP_BY_HOP_HEADERS = (
    "connection",
    "keep-alive",
    "proxy-authenticate",
    "proxy-authorization",
    "te",
    "trailers",
    "transfer-encoding",
    "upgrade",
)
DEFAULT_CONTENT_TYPES = {
    ".html": "text/html; charset=utf-8",
    ".js": "text/javascript; charset=utf-8",
    ".mjs": "text/javascript; charset=utf-8",
    ".css": "text/css; charset=utf-8",
    ".json": "application/json",
    ".svg": "image/svg+xml",
    ".png": "image/png",
    ".jpg": "image/jpeg",
    ".jpeg": "image/jpeg",
    ".gif": "image/gif",
    ".webp": "image/webp",
    ".ico": "image/x-icon",
    ".woff": "font/woff",
    ".woff2": "font/woff2",
    ".ttf": "font/ttf",
    ".map": "application/json",
    ".txt": "text/plain; charset=utf-8",
}


def is_proxied_path(path: str) -> bool:
    """Classify a request path as backend-proxied or static bundle content."""
    if path == API_PREFIX or path.startswith(API_PREFIX + "/"):
        return True
    for prefix in WEBSOCKET_PATHS + ROOT_HTTP_PROXY_PATHS:
        if path == prefix or path.startswith(prefix + "/"):
            return True
    return False


def is_websocket_upgrade(headers) -> bool:
    return "websocket" in headers.get("Upgrade", "").lower()


def filter_hop_by_hop(headers) -> list[tuple[str, str]]:
    """Filter hop-by-hop headers from either an HTTPMessage or a pair iterable."""
    items = headers.items() if hasattr(headers, "items") else headers
    filtered = []
    for name, value in items:
        if str(name).lower() in HOP_BY_HOP_HEADERS:
            continue
        filtered.append((name, value))
    return filtered


class DistProxyHandler(BaseHTTPRequestHandler):
    """Static file handler with an HTTP proxy fallback for backend paths."""

    server_version = "graft-browser-dist/1.0"
    protocol_version = "HTTP/1.1"

    @property
    def dist_dir(self) -> Path:
        return Path(self.server.dist_dir)  # type: ignore[attr-defined]

    @property
    def backend_origin(self) -> str:
        return self.server.backend_origin  # type: ignore[attr-defined]

    def do_GET(self) -> None:  # noqa: N802 - http.server naming
        self._handle(upgrade_allowed=True)

    def do_HEAD(self) -> None:  # noqa: N802 - http.server naming
        self._handle(upgrade_allowed=False)

    def do_POST(self) -> None:  # noqa: N802 - http.server naming
        self._handle(upgrade_allowed=False)

    def do_PUT(self) -> None:  # noqa: N802 - http.server naming
        self._handle(upgrade_allowed=False)

    def do_PATCH(self) -> None:  # noqa: N802 - http.server naming
        self._handle(upgrade_allowed=False)

    def do_DELETE(self) -> None:  # noqa: N802 - http.server naming
        self._handle(upgrade_allowed=False)

    def do_OPTIONS(self) -> None:  # noqa: N802 - http.server naming
        self._handle(upgrade_allowed=False)

    def _handle(self, upgrade_allowed: bool) -> None:
        path = urlsplit(self.path).path
        if is_proxied_path(path):
            if upgrade_allowed and is_websocket_upgrade(self.headers):
                self._relay_websocket()
            else:
                self._proxy_request()
        elif self.command in ("GET", "HEAD"):
            self._serve_static(path)
        else:
            self._write_plain(405, "method not allowed for static content")

    def _proxy_request(self) -> None:
        origin = urlsplit(self.backend_origin)
        length = int(self.headers.get("Content-Length") or 0)
        body = self.rfile.read(length) if length else None
        connection = HTTPConnection(origin.hostname or "127.0.0.1", origin.port or 80, timeout=120)
        try:
            connection.request(
                self.command,
                self.path,
                body=body,
                headers=dict(filter_hop_by_hop(self.headers)),
            )
            response = connection.getresponse()
            self.send_response(response.status)
            content_length: int | None = None
            for name, value in filter_hop_by_hop(response.getheaders()):
                if name.lower() == "content-length":
                    content_length = int(value)
                self.send_header(name, value)
            if content_length is None and not response.will_close:
                # Streaming responses (SSE heartbeats, long polls) have no known
                # length; forward bytes as they arrive and close when the
                # backend finishes instead of buffering the whole stream.
                self.send_header("Connection", "close")
                self.close_connection = True
            self.end_headers()
            if self.command == "HEAD":
                return
            while True:
                # read1 returns as soon as any bytes are available so streaming
                # responses (SSE heartbeats) reach the browser without waiting
                # for the backend to close or fill the full chunk size.
                chunk = response.read1(65536)
                if not chunk:
                    break
                self.wfile.write(chunk)
                self.wfile.flush()
        except OSError as error:
            self._write_plain(502, f"backend unreachable: {error}")
        finally:
            connection.close()

    def _relay_websocket(self) -> None:
        """Tunnel a websocket upgrade to the backend with raw socket relaying."""
        origin = urlsplit(self.backend_origin)
        try:
            upstream = socket.create_connection(
                (origin.hostname or "127.0.0.1", origin.port or 80), timeout=30
            )
        except OSError as error:
            self._write_plain(502, f"backend unreachable: {error}")
            return
        raw_headers = f"{self.requestline}\r\n"
        raw_headers += "".join(f"{name}: {value}\r\n" for name, value in self.headers.items())
        raw_headers += "\r\n"
        try:
            upstream.sendall(raw_headers.encode("latin-1"))
            downstream = self.connection
            # A blind TCP tunnel: the backend's 101 response is relayed verbatim,
            # preserving its headers, and both directions pump raw bytes.
            stop = threading.Event()

            def pump(source, target) -> None:
                try:
                    while not stop.is_set():
                        chunk = source.recv(65536)
                        if not chunk:
                            break
                        target.sendall(chunk)
                except OSError:
                    pass
                finally:
                    stop.set()
                    try:
                        target.shutdown(socket.SHUT_WR)
                    except OSError:
                        pass

            client_to_backend = threading.Thread(target=pump, args=(downstream, upstream), daemon=True)
            backend_to_client = threading.Thread(target=pump, args=(upstream, downstream), daemon=True)
            client_to_backend.start()
            backend_to_client.start()
            client_to_backend.join(timeout=3600)
        finally:
            stop.set()
            upstream.close()

    def _serve_static(self, path: str) -> None:
        relative = path.lstrip("/") or "index.html"
        candidate = (self.dist_dir / relative).resolve()
        try:
            candidate.relative_to(self.dist_dir.resolve())
        except ValueError:
            self._write_plain(403, "forbidden path")
            return
        if not candidate.is_file():
            candidate = self.dist_dir / "index.html"  # SPA history-mode fallback
        if not candidate.is_file():
            self._write_plain(404, "dist bundle is missing; run the production build first")
            return
        payload = candidate.read_bytes()
        suffix = candidate.suffix.lower()
        self.send_response(200)
        self.send_header("Content-Type", DEFAULT_CONTENT_TYPES.get(suffix, "application/octet-stream"))
        self.send_header("Content-Length", str(len(payload)))
        self.send_header("Cache-Control", "no-store")
        self.end_headers()
        if self.command != "HEAD":
            self.wfile.write(payload)

    def _write_plain(self, status: int, message: str) -> None:
        payload = (message + "\n").encode("utf-8")
        self.send_response(status)
        self.send_header("Content-Type", "text/plain; charset=utf-8")
        self.send_header("Content-Length", str(len(payload)))
        self.end_headers()
        if self.command != "HEAD":
            self.wfile.write(payload)

    def log_message(self, format: str, *args) -> None:  # noqa: A002 - stdlib signature
        pass  # keep attempt output clean; failures surface through the suites


def serve_forever(dist_dir: Path, backend_origin: str, host: str, port: int) -> None:
    server = ThreadingHTTPServer((host, port), DistProxyHandler)
    server.daemon_threads = True
    server.dist_dir = dist_dir  # type: ignore[attr-defined]
    server.backend_origin = backend_origin  # type: ignore[attr-defined]
    print(f"dist server ready on http://{host}:{port} (backend {backend_origin})", flush=True)
    try:
        server.serve_forever()
    except KeyboardInterrupt:
        pass
    finally:
        server.server_close()


def main(argv: list[str] | None = None) -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--dist", type=Path, default=Path("web/dist"),
                        help="Built web bundle directory (default web/dist).")
    parser.add_argument("--backend", default="http://127.0.0.1:8080",
                        help="Origin of the real backend (default http://127.0.0.1:8080).")
    parser.add_argument("--host", default="127.0.0.1")
    parser.add_argument("--port", type=int, default=4173)
    args = parser.parse_args(argv)
    if not args.dist.is_dir():
        print(f"dist directory does not exist: {args.dist}", file=sys.stderr)
        return 2
    serve_forever(args.dist, args.backend, args.host, args.port)
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
