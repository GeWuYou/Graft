# Event Schema And Safety

Use one root event for each scenario and explicit `parentEventId` for every downstream edge. Keep `seq` monotonic per
investigation and include `elapsedMs` so sync, microtask, timer, and network boundaries can be compared.

Recommended safe summaries:

- route name/path and navigation outcome;
- component name, instance token, mount generation, lifecycle event;
- store ID/action and selected mutation fields;
- watcher source label, flush mode, and old/new value summaries;
- query key structure, invalidate/refetch intent, request method/path/status/duration;
- backend request ID and error class/code.

Do not record secrets, auth headers, cookies, passwords, complete response bodies, or arbitrary form input. Apply field
allowlists, length bounds, hashing/truncation, and the existing logger's redaction policy before transport.

Default the investigation flag off. Bound event count and payload size. Do not use a global monkey patch or a new
network client merely to obtain correlation.
