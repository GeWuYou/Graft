---
name: graft-frontend-behavior-investigation
description: Evidence-first runtime investigation workflow for Vue frontend behavior whose real event, lifecycle, state, route, query, or request trigger chain cannot be proven by static analysis. Use structured, scoped probes, pause for user reproduction, reconstruct the causal timeline, verify the root cause, then clean up and add a durable regression test.
---

# Graft Frontend Behavior Investigation

Use this skill when a frontend behavior has multiple plausible trigger paths and source inspection cannot prove which
path occurred during the user's operation. It is an investigation protocol, not a general logging or automatic browser
testing skill. Follow root `AGENTS.md`, `web/AGENTS.md`, relevant frontend governance, and the normal validation entrypoint.

## Non-negotiable evidence gate

Start with a short static reconnaissance. Record source-backed facts, competing hypotheses, and the exact evidence gap.
Enter runtime mode only when at least one of these is true:

- two or more reasonable, mutually distinguishable trigger paths remain;
- lifecycle, watcher, route, store, query, or request multiplicity is not source-provable;
- two independent static passes still produce only a possible cause;
- the user explicitly asks for runtime evidence instead of a guess.

Ask `Evidence sufficient?` before changing behavior. Answer `YES` only when the triggering source is unique (or all
competitors are ruled out), the ordering does not depend on an unobserved runtime event or async boundary, and the
proposed fix is justified by source/tests. `YES` permits normal fixing. `NO` requires instrumentation and forbids a
business fix based on speculation.

Never label a hypothesis as a conclusion. Keep an evidence ledger with `fact`, `hypothesis`, `evidence`, `verified
cause`, `rejected hypothesis`, and `unknown` entries. If logs do not distinguish candidates, add the smallest missing
probe and repeat; do not infer the missing event.

## Probe layers

Classify every observation before adding it:

- **L0 Investigation Foundation (permanent):** event schema, investigation/session IDs, sequence and elapsed time,
  parent linkage, gating, field allowlists, sanitization, and logger transport. Reuse the repository's debug/logger
  infrastructure; do not create a second logger or trace authority.
- **L1 Reusable Debug Probe (conditional):** a bounded router, request, query, lifecycle, or Pinia probe that describes
  a stable technical chain across modules. It may be retained only after Probe Promotion Review (normally two
  independent scenarios; a single investigation may qualify when cross-module stability is already clear).
- **L2 Case Instrumentation (temporary default):** page-specific handlers, watcher bodies, query keys, state fields,
  or component instance markers. Tag with a case ID and remove after root-cause confirmation unless promoted.

Do not build a probe catalog speculatively in advance. Let the current evidence gap determine the minimum L2 points;
promote only proven, useful technical observations.

## Investigation workflow

1. **Recon and authority:** confirm the frontend entry page, component, route, store, query/mutation, request boundary,
   existing debug switch, and test seam. Read references only as needed: `references/evidence-gate.md`,
   `references/event-schema.md`, `references/vue-probes.md`, and `references/promotion-cleanup.md`.
2. **Behavior contract:** describe normal result, anomalous result, start/end of the user action, and scenarios that
   differ by one variable.
3. **Candidate graph:** map `USER_ACTION -> UI_EVENT -> STATE_CHANGE -> WATCHER/ROUTE -> LIFECYCLE/STORE/QUERY -> API`.
   For each edge state what would prove it and what would reject it.
4. **Instrumentation plan:** choose only probes needed to distinguish the candidates. Define an investigation ID,
   event names, parent IDs, field summaries, redactions, and a temporary-file manifest before editing.
5. **Scoped instrumentation:** preserve behavior, timing, lifecycle, request wire shape, and cache semantics. Use the
   existing logger/debug gate and structured event helper. Never add blanket `console.log`, global monkey patches, or a
   second Axios/query client.
6. **Mandatory user handoff:** after instrumentation, stop modifying code and stop reasoning from assumed outcomes.
   Give the user minimal scenarios, collection start/stop instructions, environment identity, and the exact marker to
   return. Do not run real browser interactions unless separately authorized under `graft-web-browser-agent`.
7. **Runtime reconstruction:** when logs arrive, verify completeness, order by sequence/time, rebuild parent-child and
   sync/async edges, correlate request signatures and component instance generations, then update the evidence ledger.
8. **Root-cause decision:** call a cause verified only when one complete runtime chain explains the behavior and rivals
   are rejected or no longer plausible. Otherwise report `unknown` and design another minimal probe.
9. **Fix and regression:** implement the smallest authority-level fix only after verification, validate at the durable
   module/component/router/query/API seam, and add a permanent regression test appropriate to the behavior.
10. **Promotion and cleanup:** review each L2 observation for L1 promotion, remove all case markers and subscriptions,
    verify no secret-bearing fields or temporary IDs remain, and run frontend checks.

## Structured event contract

Use the repository logger through one investigation helper (the helper may be introduced as L0 only when the task
requires it). Events are JSON-shaped even if the existing transport renders one-line text:

```text
marker, schemaVersion, investigationId, eventId, parentEventId,
seq, timestamp, elapsedMs, phase, component, event, source,
asyncBoundary, route, stateSummary, queryKeySummary, requestSummary
```

`phase` is one of `USER_ACTION`, `UI_EVENT`, `STATE_CHANGE`, `WATCHER_TRIGGER`, `ROUTE_NAVIGATION`, `LIFECYCLE`,
`STORE_ACTION`, `QUERY`, `API_REQUEST`, `ASYNC_BOUNDARY`, or `ERROR`. Explicit parent IDs are preferred over implicit
async context. Backend request/trace IDs are additional fields, never replacements for `investigationId`.

Summarize and allowlist payloads. Never emit tokens, passwords, cookies, authorization headers, full response bodies,
or unbounded user input. Record method/path/status/duration, parameter keys and safe summaries, route name/path, query
key structure, and error class/code only. Default output is off and debug-level; bound event count and field length.

## Output and handoff protocol

When evidence is insufficient, output exactly these sections: `Current conclusion`, `Confirmed facts`, `Hypotheses`,
`Evidence gap`, `Instrumentation added`, `User scenarios`, and `Return format`. State explicitly that no business fix
was made and that analysis is paused awaiting logs.

Each scenario must change one variable only (for example overlay close versus Cancel), specify when collection starts,
what to wait for, what unrelated actions to avoid, and request the complete contiguous investigation-marker output plus
scenario name and runtime identity.

After logs are returned, output: `Runtime Timeline`, `Verified Root Cause`, `Supporting Evidence`, `Rejected Hypotheses`,
`Remaining Unknowns`, `Fix Plan`, `Cleanup/Promotion Review`, and `Regression Test Plan`. Do not output `Verified Root
Cause` when the returned evidence is incomplete or ambiguous.

## Boundaries and closeout

This skill observes and establishes causality. It does not default to refactoring, permanent observability, broad source
instrumentation, automatic browser operation, or replacing unit/component/E2E testing. Browser artifacts are optional
inspection evidence and do not replace event logs or `bun run check`.

Case instrumentation must be reversible and tagged `FRONTEND-INVESTIGATION-TEMP:<case-id>`. Before closeout, remove
case-specific probes, subscriptions, flags, and metadata; retain only L0 or an approved L1. A promotion review must
check cross-module stability, technical (not business-specific) semantics, noise/performance, sanitization, off-switch,
owner, tests, and duplication with existing infrastructure. Run `bun run check` plus focused tests; use the repository's
frontend browser skill only when authorized.
