# Evidence Gate

## Ledger

| Claim type | Allowed source | Agent wording |
| --- | --- | --- |
| Fact | source, existing test, returned runtime event | “confirmed” |
| Hypothesis | candidate path not yet observed | “H1/H2; possible” |
| Evidence | event or test that supports/rejects a claim | cite event IDs/seq |
| Verified cause | complete runtime chain with rivals rejected | “verified” |
| Rejected hypothesis | contradictory or absent expected evidence | explain why |
| Unknown | still unobserved or ambiguous | keep investigation open |

The evidence gate is closed (`NO`) if any candidate edge has no discriminating observation, if a lifecycle instance or
request source is ambiguous, or if the conclusion relies on assumed user timing. A fix may begin only after the gate is
open (`YES`).

## Required candidate table

```text
H1 | predicted events | probe that distinguishes it | status
H2 | predicted events | probe that distinguishes it | status
```

Absence of a log is evidence only when the probe was enabled, the collection window covers the operation, and the
transport is known to be complete.
