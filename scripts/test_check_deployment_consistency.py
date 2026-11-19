"""Regression checks for the post-deploy consistency checker."""
import argparse
import json
import sys
import tempfile
import threading
import unittest
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
from pathlib import Path
from unittest.mock import patch

sys.path.insert(0, str(Path(__file__).resolve().parent))

import check_deployment_consistency as checker
from check_deployment_consistency import ConsistencyError


GOOD_HTML = (
    "<!doctype html><html><head>"
    '<meta charset="utf-8">'
    '<meta name="revision" content="abc123def456">'
    '<link rel="stylesheet" href="/assets/app.css">'
    '<link rel="modulepreload" href="/assets/preload.js">'
    "</head><body>"
    '<script type="module" src="/assets/app.js"></script>'
    "</body></html>"
)


class DeploymentServer:
    """Ephemeral in-process HTTP server with per-path responses."""

    def __init__(self, routes: dict[str, tuple[int, str, bytes]]):
        outer = self
        self.routes = routes

        class Handler(BaseHTTPRequestHandler):
            def do_GET(self):
                route = outer.routes.get(self.path)
                if route is None:
                    self.send_response(404)
                    self.send_header("Content-Type", "text/plain")
                    self.end_headers()
                    self.wfile.write(b"not found")
                    return
                status, content_type, body = route
                self.send_response(status)
                self.send_header("Content-Type", content_type)
                self.send_header("Content-Length", str(len(body)))
                self.end_headers()
                self.wfile.write(body)

            def log_message(self, *args):
                pass

        self.server = ThreadingHTTPServer(("127.0.0.1", 0), Handler)
        self.thread = threading.Thread(target=self.server.serve_forever, daemon=True)

    @property
    def base_url(self):
        return f"http://127.0.0.1:{self.server.server_address[1]}"

    def __enter__(self):
        self.thread.start()
        return self

    def __exit__(self, *exc_info):
        self.server.shutdown()
        self.server.server_close()


def good_routes(**overrides):
    routes = {
        "/": (200, "text/html", GOOD_HTML.encode("utf-8")),
        "/assets/app.js": (200, "text/javascript", b"console.log('app')"),
        "/assets/app.css": (200, "text/css", b"body{}"),
        "/assets/preload.js": (200, "application/javascript", b"export {}"),
    }
    routes.update(overrides)
    return routes


def check_args(**overrides):
    values = {"base_url": None, "expect_revision": None, "expect_source_digest": None,
              "acceptable_status": [], "timeout": 5.0, "insecure": False}
    values.update(overrides)
    return argparse.Namespace(**values)


def run(routes=None, **overrides):
    with DeploymentServer(good_routes() if routes is None else routes) as server:
        return checker.check(check_args(base_url=server.base_url, **overrides))


class HappyPathTests(unittest.TestCase):
    def test_consistent_deployment_passes_with_asset_and_marker_evidence(self):
        report = run(expect_revision="abc123def456")
        self.assertEqual(report["verdict"], "pass")
        self.assertEqual(report["failures"], [])
        self.assertEqual(len(report["assets"]), 3)
        kinds = {asset["url"].rsplit("/", 1)[-1]: asset["kind"] for asset in report["assets"]}
        self.assertEqual(kinds, {"app.js": "js", "app.css": "css", "preload.js": "js"})
        self.assertEqual(report["revision"], {"declared": "abc123def456",
                                              "expected": "abc123def456",
                                              "verified": True})
        self.assertTrue(report["html"]["sha256"])
        for asset in report["assets"]:
            self.assertTrue(asset["ok"])
            self.assertTrue(asset["sha256"])

    def test_main_passes_with_exit_zero(self):
        import io
        from contextlib import redirect_stdout
        with DeploymentServer(good_routes()) as server:
            buffer = io.StringIO()
            with redirect_stdout(buffer):
                checker.main(["--base-url", server.base_url,
                              "--expect-revision", "abc123def456"])
            report = json.loads(buffer.getvalue())
        self.assertEqual(report["verdict"], "pass")

    def test_relative_asset_urls_resolve_against_the_base(self):
        html = ('<html><head><link rel="stylesheet" href="nested/style.css"></head>'
                "<body><script src='nested/app.js'></script></body></html>")
        routes = {
            "/": (200, "text/html", html.encode("utf-8")),
            "/nested/app.js": (200, "text/javascript", b"a"),
            "/nested/style.css": (200, "text/css", b"b"),
        }
        report = run(routes=routes)
        self.assertEqual(report["verdict"], "pass")
        self.assertEqual({tuple(asset["url"].rsplit("/", 2)[-2:]) for asset in report["assets"]},
                         {("nested", "app.js"), ("nested", "style.css")})


class FalseSuccessPreventionTests(unittest.TestCase):
    def test_html_200_but_js_404_fails(self):
        routes = good_routes(**{"/assets/app.js": (404, "text/plain", b"nope")})
        report = run(routes=routes)
        self.assertEqual(report["verdict"], "fail")
        self.assertTrue(any("HTTP 404" in failure and "/assets/app.js" in failure
                            for failure in report["failures"]))

    def test_error_page_without_entry_assets_fails(self):
        routes = {"/": (200, "text/html", b"<html><body>Service unavailable</body></html>")}
        report = run(routes=routes)
        self.assertEqual(report["verdict"], "fail")
        self.assertTrue(any("no entry JS/CSS assets" in failure for failure in report["failures"]))

    def test_wrong_asset_content_type_fails(self):
        routes = good_routes(**{"/assets/app.js": (200, "text/html", b"console.log(1)")})
        report = run(routes=routes)
        self.assertEqual(report["verdict"], "fail")
        self.assertTrue(any("content type text/html" in failure for failure in report["failures"]))

    def test_html_non_html_content_type_fails(self):
        routes = good_routes(**{"/": (200, "application/json", b"{}")})
        report = run(routes=routes)
        self.assertEqual(report["verdict"], "fail")
        self.assertTrue(any("not text/html" in failure for failure in report["failures"]))


class AcceptableStatusContractTests(unittest.TestCase):
    def test_503_fails_by_default(self):
        routes = good_routes(**{"/assets/app.js": (503, "text/javascript", b"later")})
        report = run(routes=routes)
        self.assertEqual(report["verdict"], "fail")
        self.assertEqual(report["acceptable_status"], [])
        self.assertTrue(any("HTTP 503" in failure for failure in report["failures"]))

    def test_503_passes_only_when_explicitly_acceptable(self):
        routes = good_routes(**{"/assets/app.js": (503, "text/javascript", b"later")})
        report = run(routes=routes, acceptable_status=[503])
        self.assertEqual(report["verdict"], "pass")
        self.assertEqual(report["acceptable_status"], [503])

    def test_503_entry_html_is_unacceptable_by_default(self):
        routes = good_routes(**{"/": (503, "text/html", GOOD_HTML.encode("utf-8"))})
        report = run(routes=routes)
        self.assertEqual(report["verdict"], "fail")
        self.assertTrue(any("entry HTML" in failure and "HTTP 503" in failure
                            for failure in report["failures"]))

    def test_4xx_never_becomes_acceptable(self):
        routes = good_routes(**{"/assets/app.js": (404, "text/plain", b"nope")})
        report = run(routes=routes, acceptable_status=[404])
        self.assertEqual(report["verdict"], "fail")

    def test_unreachable_deployment_reports_before_any_verdict(self):
        with tempfile.TemporaryDirectory() as temporary:
            with self.assertRaises(ConsistencyError):
                checker.check(check_args(
                    base_url=f"http://127.0.0.1:1/{temporary}", timeout=1.0))


class RevisionContractTests(unittest.TestCase):
    def test_revision_mismatch_fails(self):
        report = run(expect_revision="ffffffffffff")
        self.assertEqual(report["verdict"], "fail")
        self.assertFalse(report["revision"]["verified"])
        self.assertTrue(any("does not match the expected" in failure
                            for failure in report["failures"]))

    def test_missing_revision_marker_with_expectation_fails_explicitly(self):
        html = '<html><head><script src="/assets/app.js"></script></head></html>'
        routes = good_routes(**{"/": (200, "text/html", html.encode("utf-8"))})
        report = run(routes=routes, expect_revision="abc123def456")
        self.assertEqual(report["verdict"], "fail")
        self.assertTrue(any("declares no revision marker" in failure
                            for failure in report["failures"]))

    def test_no_expectation_reports_declared_values_without_failure(self):
        report = run()
        self.assertEqual(report["verdict"], "pass")
        self.assertIsNone(report["revision"]["expected"])
        self.assertEqual(report["revision"]["declared"], "abc123def456")

    def test_source_digest_marker_is_verified(self):
        html = GOOD_HTML.replace(
            '<meta name="revision" content="abc123def456">',
            '<meta name="source-digest" content="deadbeef">')
        routes = good_routes(**{"/": (200, "text/html", html.encode("utf-8"))})
        report = run(routes=routes, expect_source_digest="deadbeef")
        self.assertEqual(report["verdict"], "pass")
        self.assertEqual(report["source_digest"]["verified"], True)

    def test_missing_source_digest_marker_fails_when_expected(self):
        report = run(expect_source_digest="deadbeef")
        self.assertEqual(report["verdict"], "fail")
        self.assertTrue(any("source digest marker" in failure
                            for failure in report["failures"]))

    def test_marker_parsing_accepts_alternative_marker_names(self):
        html = ('<html><head><meta name="graft-revision" content="alt-rev">'
                '<script src="/assets/app.js"></script></head></html>')
        routes = good_routes(**{"/": (200, "text/html", html.encode("utf-8"))})
        report = run(routes=routes, expect_revision="alt-rev")
        self.assertEqual(report["verdict"], "pass")
        self.assertEqual(report["revision"]["declared"], "alt-rev")


if __name__ == "__main__":
    unittest.main()
