# Lessons Learned

`ai-plan/lessons/` stores reusable project experience learned through real tasks, corrections, and closeouts.

It exists to help future contributors and AI agents avoid repeating the same mistakes without overloading `AGENTS.md`
or abusing `.ai/environment/**`.

## Purpose

Use this directory to capture:

- corrected implementation patterns
- repeatable anti-patterns
- enforcement-ready checks
- promotion history into design docs or `AGENTS.md`

Do not use it for:

- environment facts
- one-off personal preferences
- speculative rules without evidence
- task diaries

## Governance Levels

Levels describe governance destination, not severity.

- `L1`
  - lesson only
- `L2`
  - lesson plus reusable pattern, with optional promotion into design docs
- `L3`
  - lesson/design explanation plus promoted hard rule in the correct `AGENTS.md`

## File Layout

- [index.md](./index.md)
  - first-entry lesson index
- [web-ui.md](./web-ui.md)
  - frontend UI, TDesign, layout, theme, empty state, and page-pattern lessons
- [backend.md](./backend.md)
  - backend architecture, contract, plugin, migration, and validation lessons
- [governance.md](./governance.md)
  - closeout, commit, validation, collaboration, and worktree governance lessons
- [deprecated.md](./deprecated.md)
  - deprecated and superseded entries

## Lifecycle

1. closeout identifies whether the task produced a reusable lesson
2. the lesson is added or updated in the correct area file
3. `index.md` is updated
4. if the lesson expresses a reusable pattern, design docs may be updated
5. only stable, enforceable, cross-task rules may be promoted to `AGENTS.md`

## Write Threshold

Add or update a lesson only when at least 2 of these are true:

1. the user explicitly pointed out a wrong or low-quality implementation
2. the task exposed a reusable anti-pattern
3. the same issue could recur in future tasks
4. the fix can be written as correct pattern + anti-pattern + enforcement
5. the lesson relates to existing design, architecture, or governance truth
6. without capture, the agent is likely to repeat the mistake

## Promotion Rules

Promote into `AGENTS.md` only when:

- the rule is cross-task and stable
- the rule is short and enforceable
- the lesson/design layer already explains the why
- ownership is clear
- violating the rule should count as an unacceptable task result

## Anti-Bloat Rules

- search the index before creating a lesson
- update existing lessons when possible
- every lesson must declare scope and enforcement
- “just prettier” is not enough
- a lesson without verification guidance cannot become `L3`
- if one pattern accumulates more than 3 narrow lessons, consolidate it
- keep `AGENTS.md` additions short; keep long explanations here

## Closeout Contract

Task closeout should report:

```text
Experience capture:
- result: none | added | updated | promoted | deprecated
- lessons: ...
- design docs: ...
- AGENTS rules: yes/no
- reason: ...
```
