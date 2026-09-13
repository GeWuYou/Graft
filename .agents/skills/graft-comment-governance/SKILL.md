---
name: graft-comment-governance
description: "Review or edit meaningful Chinese comments in handwritten Graft Go, TypeScript, and Vue code, including code-task closeout."
---

# Graft Comment Governance

Apply [the comment standard](../../../ai-plan/design/governance/ai/代码注释与模块文档规范.md) and [root governance](../../../AGENTS.md). Reuse this task's startup, subdomain reading, and validation evidence.

Review changed handwritten Go, TypeScript, and Vue files at code-task closeout. Inspect implementation, callers, tests, and existing comments; generated, third-party, migration, and build artifacts are exempt unless explicitly included. Stale comments are defects.

## Value Gate

Before adding a comment, answer both questions:

1. Can a maintainer infer this information from the code and types alone? If yes, do not comment it.
2. Does it explain a design reason, constraint, business rule, algorithm, or external-system behavior? If no, do not comment it.

Remove or rewrite comments that merely translate names, types, templates, or framework lifecycle calls. Do not introduce a comment-count or documentation-coverage target.

## Language Rules

### Go

* Write Chinese Go doc comments for every handwritten exported package, type, interface, function, method, const, and var; start each comment with its identifier.
* Document public responsibility, error semantics, lifecycle, concurrency, registration order, compatibility, or caller-visible constraints. Use independent lines for non-obvious locking, retry, cache, transaction, and shutdown decisions.
* Do not turn every parameter or field into a prose copy of its type. Comment fields only when their lifecycle, nullability, ownership, or sharing semantics are non-obvious.

### TypeScript

* Use `/** ... */` TSDoc/JSDoc for public interfaces, exported APIs, generic contracts, and asynchronous behavior only when types do not convey their business meaning or constraints.
* Describe validation preconditions, fallback or failure behavior, ownership, and stable semantics. Do not document obvious primitive fields, return types, or local variable names.

### Vue

* Give non-trivial SFCs a component-responsibility comment in `<script setup>` that identifies its data boundary or integration behavior.
* Comment Props only for UI/business meaning that types cannot express. Comment reactive state, watchers, subscriptions, and lifecycle hooks only for the reason, ownership, cleanup obligation, or external behavior.
* Never annotate template elements, translate `onMounted`, or explain self-evident refs and computed values.

## Delegation and Closeout

A small comment change can be handled locally. For useful disjoint batches, follow root delegation policy and the [host mapping](../graft-multi-agent-batch/references/host-delegation.md): choose a verified same/lower model, or inherit the parent when comparison is unavailable. Comment generation has no special model tier.

Report changed paths, meaningful additions/updates/removals or `none-needed`, and any exemptions or semantic gaps. Formatting and comment-presence checks do not prove quality.

Comment review creates no separate commit requirement. Include accepted comments with their owning task's normal validated commit through `graft-commit` when its trigger and ownership rules hold. A worker without commit authority returns the reviewed diff to its coordinator; do not demand a comment-only commit or fabricate a blocker because a legal commit trigger is absent.
