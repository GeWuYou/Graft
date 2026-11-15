# Lesson Entry Format

Read only when a reusable lesson meets the write threshold.

## 8. Lesson Format

Use this format for every lesson entry:

```md
## LESSON-<AREA>-<TOPIC>-<NUMBER>：<标题>

- Status: active | superseded | deprecated
- Level: L1 | L2 | L3
- Applies to:
  - ...
- Source:
  - ...
- Problem:
  ...
- Correct pattern:
  ...
- Anti-pattern:
  ...
- Enforcement:
  ...
- Promotion:
  - AGENTS.md: yes/no
  - Design doc: yes/no
- Related:
  - ...
- Updated at:
  YYYY-MM-DD
```

Rules:

- IDs must stay stable
- `active` entries are default reference material
- `superseded` entries must link to the replacement lesson
- `deprecated` entries must not remain default guidance
- `Problem` must describe the real failure mode
- `Correct pattern` must be implementable
- `Anti-pattern` must state what not to do
- `Enforcement` must explain how to check compliance
