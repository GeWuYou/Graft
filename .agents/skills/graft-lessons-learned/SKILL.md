---
name: graft-lessons-learned
description: "Capture reusable engineering lessons at Graft closeout and promote only stable, verifiable rules."
---

# Graft Lessons Learned

Use this skill to turn corrected, reusable mistakes into project knowledge without polluting environment facts,
overloading [AGENTS.md](../../../AGENTS.md), or creating duplicate guidance.

## 2. When To Use

Use this skill near task closeout when one or more of these happened:

- the user pointed out that the implementation direction was wrong
- the task exposed a repeatable anti-pattern
- the fix can be generalized into a correct pattern plus enforcement
- the same mistake is likely to recur in future `web`, `server`, or governance work
- a corrected pattern now deserves promotion into design docs or [AGENTS.md](../../../AGENTS.md)

## 3. When Not To Use

Do not use this skill for:

- one-off local preferences without reuse value
- unverified guesses
- subjective taste feedback that cannot be enforced
- temporary workarounds or unstable implementation notes
- environment facts, toolchain facts, or runtime context that belong in `.ai/environment/**`

## 4. Governance Levels

Levels describe governance destination, not severity.

- `L1`
  - lesson only
  - record the experience in `ai-plan/lessons/*`
- `L2`
  - reusable pattern
  - record in `ai-plan/lessons/*`
  - update `ai-plan/design/*` when the lesson maps to a page pattern, component pattern, interface pattern, or architecture pattern
- `L3`
  - promoted hard rule
  - the detailed explanation must already exist in lesson/design material
  - add a short enforceable rule to the correct [AGENTS.md](../../../AGENTS.md)
  - violations should be treated as task-quality failure

## 5. Write Threshold

Create or update a lesson only when at least 2 of these are true:

1. the user explicitly identified the implementation as wrong, off-direction, inconsistent, or low-quality
2. the task exposed a recognizable anti-pattern
3. the same problem could recur in other tasks, pages, modules, or workflows
4. the fix can be written as a correct pattern plus anti-pattern plus enforcement
5. the issue relates to existing page templates, design rules, contracts, architecture boundaries, or governance rules
6. without capture, the agent is likely to repeat the same mistake

If the threshold is not met, closeout must report:

```text
Experience capture:
- result: none
- reason: no reusable lesson found
```

## 6. Promotion Threshold

Promote a lesson into the correct [AGENTS.md](../../../AGENTS.md) only when all of these are true:

1. the rule applies across multiple tasks, pages, modules, or long-running workflows
2. the rule is actionable and verifiable, not just aesthetic preference
3. the rule can be written briefly as a hard constraint
4. detailed explanation already exists in `ai-plan/lessons/*` or `ai-plan/design/*`
5. the rule does not conflict with existing [AGENTS.md](../../../AGENTS.md)
6. ownership is clear: root, `web`, `server`, or another local governance file
7. violating the rule would predictably cause quality regressions or repeat rework

Never:

- promote every lesson into [AGENTS.md](../../../AGENTS.md)
- place long examples or case studies inside [AGENTS.md](../../../AGENTS.md)
- promote subjective taste with no enforcement path
- skip the lesson/design layer and jump straight to hard rules

## 7. File Routing

Route captured experience by ownership:

- `.agents/skills/graft-lessons-learned/SKILL.md`
  - process and thresholds
- [ai-plan/lessons/index.md](../../../ai-plan/lessons/index.md)
  - first-entry index
- [ai-plan/lessons/web-ui.md](../../../ai-plan/lessons/web-ui.md)
  - TDesign, layout, empty states, tables, forms, drawers, detail panels, theme compatibility
- [ai-plan/lessons/backend.md](../../../ai-plan/lessons/backend.md)
  - plugin boundaries, contracts, Ent, migrations, error handling, backend validation
- [ai-plan/lessons/governance.md](../../../ai-plan/lessons/governance.md)
  - closeout, validation, commit, multi-worktree, AI collaboration, documentation governance
- [ai-plan/lessons/deprecated.md](../../../ai-plan/lessons/deprecated.md)
  - superseded and deprecated lessons
- `ai-plan/design/*`
  - only when the lesson becomes a reusable design or architecture pattern
- [web/AGENTS.md](../../../web/AGENTS.md), [server/AGENTS.md](../../../server/AGENTS.md), root [AGENTS.md](../../../AGENTS.md)
  - only for `L3` hard rules

Do not route design or governance lessons into `.ai/environment/**`.

Read [the entry format](references/lesson-format.md) only when writing or updating a lesson.

## 9. Index Maintenance

[ai-plan/lessons/index.md](../../../ai-plan/lessons/index.md) is the first lookup entry for future tasks.

Maintain these sections:

- `Active Lessons`
- `Promoted Rules`
- `Deprecated / Superseded`

Requirements:

- update the index whenever a lesson is added, updated, promoted, superseded, or deprecated
- search the index and target category file before creating a new lesson
- update an existing lesson when it already covers the same pattern

## 10. Closeout Requirements

When the task exposes a reusable correction, use this `Experience Capture Check` at closeout:

1. did the user point out an implementation mistake or dissatisfaction?
2. did the task expose a reusable anti-pattern?
3. can the fix be expressed as a correct pattern, anti-pattern, and enforcement rule?
4. does an equivalent or near-equivalent lesson already exist?
5. should the result be `add`, `update`, `merge`, `promote`, or `none`?
6. should a design document also be updated?
7. does the lesson meet [AGENTS.md](../../../AGENTS.md) promotion threshold?
8. was [ai-plan/lessons/index.md](../../../ai-plan/lessons/index.md) updated?
9. does closeout explicitly report the experience-capture result?

Closeout must include:

```text
Experience capture:
- result: none | added | updated | promoted | deprecated
- lessons: ...
- design docs: ...
- AGENTS rules: yes/no
- reason: ...
```

## 11. Deduplication Rules

- search [ai-plan/lessons/index.md](../../../ai-plan/lessons/index.md) first
- search the category file next
- if an existing lesson already covers the pattern, update it instead of creating a near-duplicate
- if multiple narrow lessons are accumulating around one pattern, consolidate them into a broader lesson or design pattern

## 12. Deprecation Rules

- mark old entries `superseded` when a newer lesson replaces them
- move obsolete or no-longer-valid guidance into [ai-plan/lessons/deprecated.md](../../../ai-plan/lessons/deprecated.md)
- preserve replacement links in both the lesson body and the index

## 13. Anti-Bloat Rules

- do not record every preference
- every lesson must have a clear scope
- “looks better” alone is not enough
- lessons without enforcement cannot be promoted to `L3`
- when similar lessons exceed 3 entries in the same area, consider merging them into one reusable pattern
- each new [AGENTS.md](../../../AGENTS.md) hard rule should stay short, normally within 5 lines
- detailed rationale belongs in lessons/design, not in [AGENTS.md](../../../AGENTS.md)

## 15. Safety Rules

- do not create `.ai/skills/graft-lessons-learned/SKILL.md`
- do not treat `.ai/environment/**` as design-governance storage
- do not let lessons become a task diary
- do not promote unstable or low-confidence guidance into hard rules
