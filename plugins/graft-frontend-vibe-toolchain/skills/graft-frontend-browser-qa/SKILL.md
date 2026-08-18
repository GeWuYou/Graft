---
name: graft-frontend-browser-qa
description: Verify Graft frontend changes with repository-approved browser inspection, screenshots, DOM checks, accessibility-oriented review, and frontend validation. Use as the Graft replacement for generic Frontend Testing Debugging, Playwright, Screenshot, or AccessLint skills, without adding Playwright test dependencies or changing web/package.json.
---

# Graft Frontend Browser QA

Use this skill after or during Graft web UI changes when behavior, layout, visibility, responsiveness, authentication, or browser evidence matters.

## Workflow

1. Start from repository governance. For web changes, `bun run check` is the completion entrypoint.
2. Run browser QA only in the developer-owned primary checkout after a user or developer approves a runtime proven to
   serve the branch and HEAD under review. Numbered agent worktrees must not start services or use browser agents,
   Playwright, screenshots, or browser interaction; record primary-checkout browser QA as follow-up rather than
   blocking a validated scoped worktree commit.
3. Use `$graft-web-browser-agent` for eligible primary-checkout interaction, authentication, screenshots, DOM text
   snapshots, and simple click/fill/wait checks.
4. Use Playwright MCP only as an exploratory browser aid when it is already configured; do not add a Playwright test dependency or generate a new test baseline.
5. Inspect console errors, failed network requests, broken auth flows, layout overlap, unreadable text, missing affordances, and focus/keyboard traps.
6. Check desktop and mobile-sized viewports when the changed surface is responsive or visually material. Exercise the longest realistic labels, IDs, validation messages, empty states, and loading replacements so they do not clip, overlap, or cause disruptive layout shifts.
7. Keep evidence auditable: record the branch/HEAD, command or browser path, page/surface inspected, and any validation gaps.

## Accessibility-Oriented Review

- Verify controls have visible labels; icon-only controls have accessible names; decorative icons do not create duplicate announcements; and focus order follows the visual workflow.
- Verify keyboard users can reach and operate changed controls, receive a visible unobscured focus indicator, and have an alternative to any drag, hover-only, or pointer-only action.
- Verify form fields retain specific adjacent errors and recovery guidance. Multi-error submission should preserve field errors and direct users to a useful error summary or the first invalid field; status messages must not steal focus.
- Verify disabled, loading, success, empty, and error states communicate both the current status and the next viable action without relying on color alone.
- Verify fixed bars, dialogs, drawers, tables, tags, and long user-provided values remain readable and operable at the reviewed viewport sizes; truncation must not hide essential information without a usable disclosure.
- Prefer TDesign-native controls and semantics before custom DOM.
- Treat screenshot review as evidence, not as a replacement for `bun run check`.

## Boundaries

- Do not modify runtime code just to satisfy a visual preference without tying it to the user request or repository docs.
- Do not create new test runners, install packages, or change browser tooling ownership from a QA request alone.
- If browser verification cannot run, state the expected command/tool and the concrete blocker.
