#!/usr/bin/env python3
"""Post-deploy consistency check for a deployed Graft web stack.

Fetches the deployed HTML entry and every referenced entry JS/CSS asset so
an "HTML 200 but JS 404" deployment can never read as a success. Status
codes, Content-Type bindings and revision/source-digest declarations are
verified as one verdict. The acceptable-status contract is explicit and
empty by default: no 5xx (including 503) is acceptable unless it is named
via --acceptable-status.
"""
from __future__ import annotations

import argparse
import datetime as dt
import hashlib
import json
import ssl
import sys
from html.parser import HTMLParser
from urllib import error as urlerror
from urllib import request as urlrequest
from urllib.parse import urljoin

REPORT_SCHEMA_VERSION = 1
REVISION_MARKERS = ("revision", "graft-revision", "source-revision")
SOURCE_DIGEST_MARKERS = ("source-digest", "source_digest", "graft-source-digest")
JS_MEDIA_TYPES = frozenset({"text/javascript", "application/javascript"})
CSS_MEDIA_TYPES = frozenset({"text/css"})


class ConsistencyError(RuntimeError):
    """Unrecoverable input or network failure reported before any verdict."""


def utc_now() -> str:
    return dt.datetime.now(dt.timezone.utc).strftime("%Y-%m-%dT%H:%M:%SZ")


class EntryHtmlParser(HTMLParser):
    """Collects entry asset references and revision marker declarations."""

    def __init__(self) -> None:
        super().__init__(convert_charrefs=True)
        self.scripts: list[str] = []
        self.stylesheets: list[str] = []
        self.preloads: list[str] = []
        self.meta: dict[str, str] = {}

    def handle_starttag(self, tag, attrs):
        attributes = {key.lower(): (value or "") for key, value in attrs}
        if tag == "script" and attributes.get("src"):
            self.scripts.append(attributes["src"])
        elif tag == "link":
            rel = attributes.get("rel", "").lower()
            href = attributes.get("href")
            if not href:
                return
            if "stylesheet" in rel:
                self.stylesheets.append(href)
            elif "modulepreload" in rel:
                self.preloads.append(href)
        elif tag == "meta" and attributes.get("name") and attributes.get("content"):
            self.meta[attributes["name"].lower()] = attributes["content"]


def fetch(url: str, timeout: float, insecure: bool) -> dict:
    context = None
    if insecure and url.startswith("https://"):
        context = ssl.create_default_context()
        context.check_hostname = False
        context.verify_mode = ssl.CERT_NONE
    request = urlrequest.Request(url, headers={"Accept": "*/*"})
    try:
        with urlrequest.urlopen(request, timeout=timeout, context=context) as response:
            return {"status": response.status,
                    "content_type": response.headers.get("Content-Type", ""),
                    "body": response.read()}
    except urlerror.HTTPError as exc:
        return {"status": exc.code,
                "content_type": exc.headers.get("Content-Type", "") if exc.headers else "",
                "body": exc.read() if exc.fp else b""}


def status_failure(kind: str, url: str, status: int, acceptable: set[int]) -> str | None:
    """The acceptable-status contract: 2xx passes; 5xx only when explicitly
    listed; everything else fails. Default contract accepts no 5xx at all."""
    if 200 <= status < 300:
        return None
    if status >= 500 and status in acceptable:
        return None
    contract = f" (acceptable only because of --acceptable-status {status})" \
        if status >= 500 else ""
    return f"{kind} {url} returned HTTP {status}{contract}"


def media_type(content_type: str) -> str:
    return content_type.split(";")[0].strip().lower()


def collect_assets(html: str, base_url: str) -> tuple[list[dict], dict[str, str]]:
    parser = EntryHtmlParser()
    parser.feed(html)
    parser.close()
    assets = []
    for href, kind in ([(item, "js") for item in parser.scripts]
                       + [(item, "css") for item in parser.stylesheets]
                       + [(item, "js") for item in parser.preloads]):
        if not href or href.startswith(("data:", "javascript:", "#")):
            continue
        assets.append({"url": urljoin(base_url, href), "kind": kind, "spec": href})
    markers: dict[str, str] = {}
    for name in REVISION_MARKERS:
        if name in parser.meta:
            markers["revision"] = parser.meta[name]
            break
    for name in SOURCE_DIGEST_MARKERS:
        if name in parser.meta:
            markers["source_digest"] = parser.meta[name]
            break
    return assets, markers


def check(args: argparse.Namespace) -> dict:
    base_url = args.base_url.rstrip("/") + "/"
    acceptable = set(args.acceptable_status)
    failures: list[str] = []
    try:
        html_response = fetch(base_url, args.timeout, args.insecure)
    except (urlerror.URLError, OSError, ValueError) as exc:
        raise ConsistencyError(f"entry HTML {base_url} is unreachable: {exc}") from exc
    html_status = html_response["status"]
    html_type = media_type(html_response["content_type"])
    html_report = {"url": base_url, "status": html_status, "content_type": html_type,
                   "sha256": hashlib.sha256(html_response["body"]).hexdigest()}
    reason = status_failure("entry HTML", base_url, html_status, acceptable)
    if reason:
        failures.append(reason)
    if html_type != "text/html":
        failures.append(f"entry HTML content type {html_type or 'missing'} is not text/html")

    assets, markers = collect_assets(html_response["body"].decode("utf-8", "replace"), base_url)
    if not assets:
        failures.append(
            "no entry JS/CSS assets discovered in the HTML; an SPA entry must reference "
            "at least one script or stylesheet (an error page must not read as a pass)")

    asset_reports = []
    for asset in assets:
        try:
            response = fetch(asset["url"], args.timeout, args.insecure)
        except (urlerror.URLError, OSError, ValueError) as exc:
            failures.append(f"entry asset {asset['url']} is unreachable: {exc}")
            asset_reports.append({"url": asset["url"], "kind": asset["kind"],
                                  "status": None, "content_type": None,
                                  "sha256": None, "ok": False})
            continue
        asset_type = media_type(response["content_type"])
        expected_types = JS_MEDIA_TYPES if asset["kind"] == "js" else CSS_MEDIA_TYPES
        asset_ok = True
        accepted_status = False
        reason = status_failure("entry asset", asset["url"], response["status"], acceptable)
        if reason:
            failures.append(reason)
            asset_ok = False
        elif 200 > response["status"] or response["status"] >= 300:
            # The status passed only through the explicit --acceptable-status
            # contract: it is an error page by definition, so it is accepted
            # but never content-verified.
            accepted_status = True
        if response["status"] in range(200, 300) and asset_type not in expected_types:
            expected_display = " or ".join(sorted(expected_types))
            failures.append(f"entry asset {asset['url']} ({asset['kind']}) has content type "
                            f"{asset_type or 'missing'}, expected {expected_display}")
            asset_ok = False
        asset_reports.append({"url": asset["url"], "kind": asset["kind"], "spec": asset["spec"],
                              "status": response["status"], "content_type": asset_type,
                              "sha256": hashlib.sha256(response["body"]).hexdigest(),
                              "accepted_status": accepted_status,
                              "ok": asset_ok})

    def verify_marker(markers_key: str, marker_names: tuple[str, ...],
                      expected: str | None, label: str) -> dict:
        declared = markers.get(markers_key)
        if expected is None:
            return {"declared": declared, "expected": None,
                    "verified": None,
                    "note": ("pass --expect-" + label + " to verify; HTML declares "
                             "no " + label + " marker") if declared is None
                            else "declared value reported, not verified"}
        if declared is None:
            failures.append(
                f"HTML declares no {label} marker (looked for meta "
                f"{', '.join(marker_names)}); deployment revision consistency cannot be "
                "verified, so the check fails instead of guessing")
            return {"declared": None, "expected": expected, "verified": False}
        verified = declared == expected
        if not verified:
            failures.append(f"HTML {label} marker {declared!r} does not match the "
                            f"expected {expected!r}")
        return {"declared": declared, "expected": expected, "verified": verified}

    revision = verify_marker("revision", REVISION_MARKERS, args.expect_revision, "revision")
    source_digest = verify_marker("source_digest", SOURCE_DIGEST_MARKERS,
                                  args.expect_source_digest, "source digest")
    verdict = "fail" if failures else "pass"
    return {
        "schema_version": REPORT_SCHEMA_VERSION,
        "mode": "post-deploy-consistency",
        "generated_at": utc_now(),
        "base_url": base_url,
        "html": html_report,
        "assets": asset_reports,
        "revision": revision,
        "source_digest": source_digest,
        "acceptable_status": sorted(acceptable),
        "failures": failures,
        "verdict": verdict,
    }


def build_parser() -> argparse.ArgumentParser:
    parser = argparse.ArgumentParser(description=__doc__, epilog=(
        "Default acceptable-status contract: none. Any 5xx (including 503) fails "
        "unless explicitly listed with --acceptable-status; 4xx always fails."))
    parser.add_argument("--base-url", required=True, help="Deployed web entry base URL")
    parser.add_argument("--expect-revision",
                        help="Git revision the deployment must declare in its HTML")
    parser.add_argument("--expect-source-digest",
                        help="Source digest the deployment must declare in its HTML")
    parser.add_argument("--acceptable-status", action="append", type=int, default=[],
                        metavar="STATUS",
                        help="A 5xx status that this check may accept (repeatable)")
    parser.add_argument("--timeout", type=float, default=10.0,
                        help="Per-request timeout in seconds")
    parser.add_argument("--insecure", action="store_true",
                        help="Disable TLS verification (explicit opt-in only)")
    return parser


def main(argv: list[str] | None = None) -> None:
    args = build_parser().parse_args(argv)
    report = check(args)
    print(json.dumps(report, indent=2, ensure_ascii=False))
    if report["verdict"] == "fail":
        raise SystemExit(1)


if __name__ == "__main__":
    try:
        main()
    except ConsistencyError as exc:
        print(f"check-deployment-consistency: {exc}", file=sys.stderr)
        raise SystemExit(1) from None
    except (OSError, ValueError, KeyError) as exc:
        print(f"check-deployment-consistency: {type(exc).__name__}: {exc}", file=sys.stderr)
        raise SystemExit(1) from exc
