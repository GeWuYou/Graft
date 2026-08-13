# Promotion Review And Cleanup

Before deleting a case probe, classify it as L2 or candidate L1. Promote only when the observation describes a stable
technical chain across modules, is bounded and low-noise, has safe summaries, an off-switch, a clear owner, focused
tests, and no overlap with existing logger/debug/request/query infrastructure.

Default threshold is two independent scenarios. A single investigation may qualify early when cross-module stability
is already obvious; this requires explicit owner, tests, review evidence, and a written reason for the exception.

Case probes use `FRONTEND-INVESTIGATION-TEMP:<case-id>`. The cleanup manifest lists every changed file, symbol,
subscription, flag, and metadata field. Remove all case-specific calls and subscriptions, then scan for the marker,
case ID, unsafe fields, and bare `console.*`. Verify the investigation flag is off and run focused tests plus `bun run
check`. L0 remains; only an approved L1 survives.
