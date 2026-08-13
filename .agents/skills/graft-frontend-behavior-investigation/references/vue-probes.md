# Vue Probe Selection

Choose probes from the candidate graph, not from a checklist.

| Suspected edge | Minimal probe |
| --- | --- |
| remount or unexpected initialization | target `setup`, `onMounted`, `onBeforeUnmount`, `onUnmounted`; instance token + generation |
| overlay/cancel difference | the two handlers, drawer close callback, propagation/default flags |
| watcher side effect | watcher boundary, source label, flush mode, summarized old/new values |
| route-induced refresh | local `push/replace`, filtered navigation guards, `afterEach` outcome |
| Pinia duplication | target store `$onAction`; scoped `$subscribe` for selected fields |
| query invalidation/refetch | target query/mutation call site and filtered cache event |
| unknown request origin | module queryFn/mutationFn and the shared request boundary |

Subscriptions and probes must be scoped, retain unsubscribe handles, and be removed during cleanup. Do not log every
`computed` evaluation or reactive mutation unless it is a proven candidate edge.
