---
name: graft-frontend-browser-qa
description: Verify Graft frontend changes with repository-approved browser inspection, screenshots, DOM checks, accessibility-oriented review, and frontend validation. Use as the Graft replacement for generic Frontend Testing Debugging, Playwright, Screenshot, or AccessLint skills, without adding Playwright test dependencies or changing web/package.json.
---

# Graft Frontend Browser QA

Use this skill after or during Graft web UI changes when behavior, layout, visibility, responsiveness, authentication, or browser evidence matters.

## Workflow

1. Start from repository governance. For web changes, `bun run check` is the completion entrypoint.
2. Use `$graft-web-browser-agent` for local Graft browser interaction, authentication, screenshots, DOM text snapshots, and simple click/fill/wait checks.
3. Use Playwright MCP only as an exploratory browser aid when it is already configured; do not add a Playwright test dependency or generate a new test baseline.
4. Inspect console errors, failed network requests, broken auth flows, layout overlap, unreadable text, missing affordances, and focus/keyboard traps.
5. Check desktop and mobile-sized viewports when the changed surface is responsive or visually material.
6. Keep evidence auditable: record the command or browser path used, the page/surface inspected, and any validation gaps.

## Accessibility-Oriented Review

- Verify controls have visible labels, predictable focus order, usable disabled/loading/empty/error states, and color-independent status signals.
- Prefer TDesign-native controls and semantics before custom DOM.
- Treat screenshot review as evidence, not as a replacement for `bun run check`.

## Boundaries

- Do not modify runtime code just to satisfy a visual preference without tying it to the user request or repository docs.
- Do not create new test runners, install packages, or change browser tooling ownership from a QA request alone.
- If browser verification cannot run, state the expected command/tool and the concrete blocker.
