#!/usr/bin/env python3
"""Create or reconcile the current branch pull request for Graft."""

from __future__ import annotations

import argparse
import json
import os
from pathlib import Path
import subprocess
import sys
from typing import Any

MANAGED_BLOCK_START = "<!-- graft-pr-create:managed-start -->"
MANAGED_BLOCK_END = "<!-- graft-pr-create:managed-end -->"
DEFAULT_GH_TIMEOUT_SECONDS = 60


class PrCreateError(RuntimeError):
    """Raised when the PR workflow must fail closed."""

    def __init__(self, message: str, diagnostics: list[str] | None = None):
        super().__init__(message)
        self.diagnostics = diagnostics or []


def run_command(args: list[str]) -> str:
    """Run a command and return stdout."""
    process = subprocess.run(args, capture_output=True, text=True, check=False)
    if process.returncode != 0:
        raise PrCreateError(
            f"Command failed: {' '.join(args)}",
            diagnostics=[process.stderr.strip() or process.stdout.strip()],
        )
    return process.stdout.strip()


def run_gh_json(args: list[str], payload: dict[str, Any] | None = None) -> Any:
    """Run gh and parse JSON stdout."""
    env = os.environ.copy()
    env.setdefault("GH_PAGER", "")
    process = subprocess.run(
        ["gh", *args],
        input=json.dumps(payload) if payload is not None else None,
        capture_output=True,
        text=True,
        check=False,
        env=env,
        timeout=DEFAULT_GH_TIMEOUT_SECONDS,
    )
    if process.returncode != 0:
        raise PrCreateError(
            f"gh command failed: {' '.join(args)}",
            diagnostics=[process.stderr.strip() or process.stdout.strip()],
        )
    stdout = process.stdout.strip()
    if not stdout:
        return None
    return json.loads(stdout)


def get_current_branch() -> str:
    branch = run_command(["git", "branch", "--show-current"])
    if not branch:
        raise PrCreateError("Detached HEAD is not supported for graft-pr-create.")
    return branch


def get_status_lines() -> list[str]:
    output = run_command(["git", "status", "--short"])
    return [line for line in output.splitlines() if line.strip()]


def get_upstream_branch(branch: str) -> tuple[str, str]:
    remote = run_command(["git", "config", "--get", f"branch.{branch}.remote"])
    merge_ref = run_command(["git", "config", "--get", f"branch.{branch}.merge"])
    if not remote or not merge_ref:
        raise PrCreateError(
            f"Branch '{branch}' has no configured upstream.",
            diagnostics=["Run $graft-push first so the PR source branch is explicit."],
        )
    if not merge_ref.startswith("refs/heads/"):
        raise PrCreateError(
            f"Branch '{branch}' has unsupported upstream ref '{merge_ref}'.",
        )
    return remote, merge_ref.removeprefix("refs/heads/")


def get_head_subject() -> str:
    return run_command(["git", "log", "-1", "--pretty=%s"])


def load_body_file(path: str | None) -> str:
    if not path:
        return ""
    return Path(path).read_text(encoding="utf-8")


def get_repo_info() -> dict[str, Any]:
    repo = run_gh_json(
        [
            "repo",
            "view",
            "--json",
            "nameWithOwner,defaultBranchRef,mergeCommitAllowed,rebaseMergeAllowed,squashMergeAllowed,viewerDefaultMergeMethod,url",
        ]
    )
    graphql = run_gh_json(
        [
            "api",
            "graphql",
            "-f",
            (
                'query=query { repository(owner:"GeWuYou", name:"Graft") { autoMergeAllowed '
                "branchProtectionRules(first: 50) { nodes { pattern requiresStatusChecks "
                "requiresStrictStatusChecks requiresApprovingReviews } } } }"
            ),
        ]
    )
    repo_data = graphql["data"]["repository"]
    return {
        "name_with_owner": repo["nameWithOwner"],
        "default_branch": repo["defaultBranchRef"]["name"],
        "viewer_default_merge_method": repo["viewerDefaultMergeMethod"],
        "merge_commit_allowed": bool(repo["mergeCommitAllowed"]),
        "rebase_merge_allowed": bool(repo["rebaseMergeAllowed"]),
        "squash_merge_allowed": bool(repo["squashMergeAllowed"]),
        "auto_merge_allowed": bool(repo_data["autoMergeAllowed"]),
        "branch_protection_rules": repo_data["branchProtectionRules"]["nodes"],
        "url": repo["url"],
    }


def branch_has_protection(repo_info: dict[str, Any], branch: str) -> bool:
    for rule in repo_info["branch_protection_rules"]:
        pattern = rule["pattern"]
        if pattern == branch or pattern == "*":
            if (
                rule["requiresStatusChecks"]
                or rule["requiresStrictStatusChecks"]
                or rule["requiresApprovingReviews"]
            ):
                return True
    return False


def get_open_pull_requests(branch: str) -> list[dict[str, Any]]:
    payload = run_gh_json(
        [
            "pr",
            "list",
            "--head",
            branch,
            "--state",
            "open",
            "--json",
            "number,title,url,body,baseRefName,headRefName",
        ]
    )
    return list(payload or [])


def render_managed_block(
    *,
    head_branch: str,
    base_branch: str,
    repo_info: dict[str, Any],
    diagnostics: list[str],
    extra_body: str,
) -> str:
    lines = [
        MANAGED_BLOCK_START,
        "graft-pr-create managed metadata",
        f"- repository: {repo_info['name_with_owner']}",
        f"- head: {head_branch}",
        f"- base: {base_branch}",
    ]
    if diagnostics:
        lines.append("- diagnostics:")
        lines.extend(f"  - {item}" for item in diagnostics)
    else:
        lines.append("- diagnostics: none")
    if extra_body.strip():
        lines.append("- closeout:")
        lines.extend(f"  {line}" if line else "  " for line in extra_body.strip().splitlines())
    lines.append(MANAGED_BLOCK_END)
    return "\n".join(lines)


def stable_managed_diagnostics(items: list[str]) -> list[str]:
    """Keep only diagnostics that should persist in the managed PR body."""
    unstable_prefixes = (
        "no open PR matched branch ",
    )
    return [item for item in items if not item.startswith(unstable_prefixes)]


def merge_body(existing_body: str, managed_block: str) -> str:
    existing_body = existing_body or ""
    if MANAGED_BLOCK_START in existing_body and MANAGED_BLOCK_END in existing_body:
        start_index = existing_body.index(MANAGED_BLOCK_START)
        end_index = existing_body.index(MANAGED_BLOCK_END) + len(MANAGED_BLOCK_END)
        merged = existing_body[:start_index].rstrip()
        suffix = existing_body[end_index:].lstrip()
        parts = [part for part in [merged, managed_block, suffix] if part]
        return "\n\n".join(parts)
    if not existing_body.strip():
        return managed_block
    return f"{existing_body.rstrip()}\n\n{managed_block}"


def choose_merge_method(repo_info: dict[str, Any]) -> str:
    method = repo_info["viewer_default_merge_method"]
    allowed = {
        "MERGE": repo_info["merge_commit_allowed"],
        "REBASE": repo_info["rebase_merge_allowed"],
        "SQUASH": repo_info["squash_merge_allowed"],
    }
    if method in allowed and allowed[method]:
        return method
    for fallback in ("MERGE", "SQUASH", "REBASE"):
        if allowed.get(fallback):
            return fallback
    raise PrCreateError("No allowed merge method is available for auto-merge.")


def write_temp_body(content: str) -> Path:
    path = Path("/tmp/graft-pr-create-body.md")
    path.write_text(content, encoding="utf-8")
    return path


def update_pull_request(pr_number: int, *, base_branch: str | None, title: str | None, body: str | None) -> None:
    fields: list[str] = []
    if base_branch is not None:
        fields.extend(["--base", base_branch])
    if title is not None:
        fields.extend(["--title", title])
    if body is not None:
        body_path = write_temp_body(body)
        fields.extend(["--body-file", str(body_path)])
    if not fields:
        return
    run_command(["gh", "pr", "edit", str(pr_number), *fields])


def create_pull_request(*, title: str, body: str, base_branch: str, head_branch: str) -> dict[str, Any]:
    body_path = write_temp_body(body)
    run_command(
        [
            "gh",
            "pr",
            "create",
            "--title",
            title,
            "--body-file",
            str(body_path),
            "--base",
            base_branch,
            "--head",
            head_branch,
        ]
    )
    payload = run_gh_json(
        [
            "pr",
            "view",
            head_branch,
            "--json",
            "number,url,title,body,baseRefName,headRefName",
        ]
    )
    return dict(payload)


def enable_auto_merge(pr_number: int, merge_method: str) -> None:
    run_command(["gh", "pr", "merge", str(pr_number), f"--{merge_method.lower()}", "--auto"])


def build_result(
    *,
    action: str,
    pr: dict[str, Any] | None,
    head_branch: str,
    base_branch: str,
    auto_merge: str,
    diagnostics: list[str],
    status_lines: list[str],
) -> dict[str, Any]:
    return {
        "action": action,
        "pr_number": pr["number"] if pr else None,
        "pr_url": pr["url"] if pr else None,
        "pr_title": pr["title"] if pr else None,
        "head_branch": head_branch,
        "base_branch": base_branch,
        "auto_merge": auto_merge,
        "diagnostics": diagnostics,
        "working_tree_dirty": bool(status_lines),
        "working_tree_status": status_lines,
    }


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--dry-run", action="store_true", help="Print the planned action without changing GitHub state.")
    parser.add_argument("--base", help="Override the default PR base branch.")
    parser.add_argument("--title", help="Override the PR title.")
    parser.add_argument("--body-file", help="Inject additional managed-block body content from a file.")
    parser.add_argument("--enable-auto-merge", action="store_true", help="Request auto-merge enablement.")
    parser.add_argument(
        "--confirm-automerge",
        action="store_true",
        help="Second confirmation required before auto-merge can be enabled.",
    )
    parser.add_argument("--format", choices=("text", "json"), default="text")
    parser.add_argument("--json-output", help="Write the JSON result to a file.")
    return parser.parse_args()


def maybe_write_json(path: str | None, payload: dict[str, Any]) -> None:
    if not path:
        return
    Path(path).write_text(json.dumps(payload, indent=2, ensure_ascii=False) + "\n", encoding="utf-8")


def render_text(result: dict[str, Any]) -> str:
    lines = [
        f"Action: {result['action']}",
        f"Head: {result['head_branch']}",
        f"Base: {result['base_branch']}",
        f"Auto-merge: {result['auto_merge']}",
    ]
    if result["pr_number"] is not None:
        lines.append(f"PR: #{result['pr_number']} {result['pr_url']}")
    if result["diagnostics"]:
        lines.append("Diagnostics:")
        lines.extend(f"- {item}" for item in result["diagnostics"])
    return "\n".join(lines)


def main() -> int:
    args = parse_args()
    try:
        branch = get_current_branch()
        status_lines = get_status_lines()
        repo_info = get_repo_info()
        if branch == repo_info["default_branch"]:
            raise PrCreateError("Current branch matches the repository default branch; refusing to create a PR.")
        _, upstream_branch = get_upstream_branch(branch)
        if upstream_branch != branch:
            raise PrCreateError(
                f"Current branch '{branch}' pushes to '{upstream_branch}', which is not supported for PR creation.",
            )
        base_branch = args.base or repo_info["default_branch"]
        existing_prs = get_open_pull_requests(branch)
        if len(existing_prs) > 1:
            raise PrCreateError(
                f"Multiple open PRs matched branch '{branch}'.",
                diagnostics=[f"Matched PR numbers: {', '.join(str(pr['number']) for pr in existing_prs)}"],
            )

        extra_body = load_body_file(args.body_file)
        diagnostics: list[str] = []
        action = "dry_run" if args.dry_run else "created"
        pr: dict[str, Any] | None = None

        if existing_prs:
            pr = existing_prs[0]
            action = "reused"
            if pr["headRefName"] != branch:
                raise PrCreateError(
                    f"Open PR #{pr['number']} head branch '{pr['headRefName']}' does not match current branch '{branch}'."
                )
            if pr["baseRefName"] != base_branch:
                diagnostics.append(f"existing PR base is '{pr['baseRefName']}', target base is '{base_branch}'")
        else:
            diagnostics.append(f"no open PR matched branch '{branch}'")

        managed_block = render_managed_block(
            head_branch=branch,
            base_branch=base_branch,
            repo_info=repo_info,
            diagnostics=stable_managed_diagnostics(diagnostics),
            extra_body=extra_body,
        )
        body = merge_body(pr["body"] if pr else "", managed_block)
        title = args.title or (pr["title"] if pr else get_head_subject())

        if not args.dry_run:
            if pr is None:
                pr = create_pull_request(title=title, body=body, base_branch=base_branch, head_branch=branch)
                action = "created"
            else:
                updates_needed = (
                    body != (pr.get("body") or "")
                    or (args.title is not None and args.title != pr["title"])
                    or pr["baseRefName"] != base_branch
                )
                if updates_needed:
                    update_pull_request(
                        pr["number"],
                        base_branch=base_branch if pr["baseRefName"] != base_branch else None,
                        title=args.title if args.title is not None else None,
                        body=body if body != (pr.get("body") or "") else None,
                    )
                    pr["body"] = body
                    pr["baseRefName"] = base_branch
                    if args.title is not None:
                        pr["title"] = args.title
                    action = "updated"
        else:
            if pr is None:
                action = "dry_run"

        auto_merge = "not_requested"
        if args.enable_auto_merge:
            if not args.confirm_automerge:
                raise PrCreateError(
                    "Refusing to enable auto-merge without --confirm-automerge.",
                    diagnostics=["Auto-merge requires both --enable-auto-merge and --confirm-automerge."],
                )
            if not repo_info["auto_merge_allowed"]:
                raise PrCreateError("Repository auto-merge is disabled on GitHub.")
            if not branch_has_protection(repo_info, base_branch):
                auto_merge = "would_enable"
                diagnostics.append(
                    f"base branch '{base_branch}' has no detected protection or required checks; auto-merge was not enabled"
                )
            elif args.dry_run:
                auto_merge = "would_enable"
            else:
                if pr is None:
                    raise PrCreateError("Cannot enable auto-merge because no PR was created or resolved.")
                enable_auto_merge(pr["number"], choose_merge_method(repo_info))
                auto_merge = "enabled"

        result = build_result(
            action=action,
            pr=pr,
            head_branch=branch,
            base_branch=base_branch,
            auto_merge=auto_merge,
            diagnostics=diagnostics,
            status_lines=status_lines,
        )
        maybe_write_json(args.json_output, result)
        if args.format == "json":
            print(json.dumps(result, indent=2, ensure_ascii=False))
        else:
            print(render_text(result))
        return 0
    except PrCreateError as error:
        result = {
            "action": "blocked",
            "pr_number": None,
            "pr_url": None,
            "head_branch": None,
            "base_branch": None,
            "auto_merge": "blocked",
            "diagnostics": [str(error), *error.diagnostics],
        }
        maybe_write_json(args.json_output, result)
        if args.format == "json":
            print(json.dumps(result, indent=2, ensure_ascii=False))
        else:
            print(render_text(result))
        return 1


if __name__ == "__main__":
    sys.exit(main())
