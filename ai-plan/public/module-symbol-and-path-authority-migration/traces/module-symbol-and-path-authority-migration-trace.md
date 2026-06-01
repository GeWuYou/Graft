# Module Symbol And Path Authority Migration Trace

## Summary

- Re-ran startup preflight from root `AGENTS.md`.
- Kept the task classified as `cross-boundary`.
- Treated `module-oriented-modular-monolith` as archive-ready parent evidence rather than continuing that wording-only topic in place.
- Opened a new bounded topic at `ai-plan/public/module-symbol-and-path-authority-migration/**`.
- Attempted Batch 2 as a tiny descriptor-local candidate slice, then rejected it after direct package validation showed the target names are package-scoped within each plugin package.
- Completed Batch 3 by reconciling the recovery state, clearing stale pending/current batch markers, and running the terminal archive-readiness check.
- Verified the Batch 1 authority inventory against:
  - `server/internal/plugin/**`
  - `server/internal/pluginregistry/**`
  - `server/plugins/*/descriptor.go`
  - `server/internal/app/runtime.go`
- Confirmed the minimum additional authority file needed by Batch 1 is `server/internal/app/runtime.go` because it is the immediate runtime consumer of the registry/plugin exported symbols under review.
- Kept the round doc-only; no exported symbol, package/path, generator, or runtime rename landed in Batch 1.

## Batch 2 Slice

- attempted target files:
  - `server/plugins/auth/descriptor.go`
  - `server/plugins/user/descriptor.go`
  - `server/plugins/rbac/descriptor.go`
  - `server/plugins/audit/descriptor.go`
  - `server/plugins/monitor/descriptor.go`
- attempted candidate names:
  - `pluginID`
  - `pluginVersion`
  - `pluginDependencies`
- validation outcome:
  - the names are referenced by other files in the same package, including runtime files and tests
  - they are package-scoped plugin-local authorities, not descriptor-local-only names
  - the attempted rename was reverted and no Go code change landed in Batch 2
- left unchanged by design:
  - exported symbols in `server/internal/plugin/**`
  - runtime-consumer names in `server/internal/app/runtime.go`
  - generator constants and generated output under `server/internal/pluginregistry/**`
  - package paths, physical directories, import paths, and migration strings
  - `server/plugins/scheduler/descriptor.go` because this batch was pre-approved only for the five target files

## Batch 1 Findings

### Exported Symbol Class

- `server/internal/plugin/plugin.go`
  - `Plugin`
  - `Descriptor`
- `server/internal/plugin/runtime_metadata.go`
  - `OrderedPluginDescriptors`
- `server/internal/pluginregistry/registry.go`
  - `BuildPlugins`

Verdict:

- these names are real authority surfaces and must not be renamed as wording cleanup
- any rename must be handled as an explicit API/runtime migration slice with direct Go validation

### Package/Path Class

- `graft/server/internal/plugin`
- `graft/server/internal/pluginregistry`
- `server/plugins/<name>`
- migration path strings under `plugins/<name>/migrations`

Verdict:

- these are physical/import-path authorities, not comment-level wording drift
- Batch 1 keeps them deferred because changing them would widen into filesystem, imports, generator outputs, and migration-path semantics

### Import/Generator Class

- `server/internal/pluginregistry/cmd/pluginregistrygen/main.go`
  - `pluginsDirName`
  - generated alias suffix `plugin`
  - generated `[]plugin.Descriptor` references
- `server/internal/pluginregistry/generated.go`

Verdict:

- generator constants and generated output shape are tightly coupled to real package and directory truth
- they should only move when the corresponding authority migration is accepted, not ahead of it

### Descriptor-Local Class

- `server/plugins/*/descriptor.go`
  - local names such as `pluginID`, `pluginVersion`, `pluginDependencies`

Verdict:

- the initial Batch 1 assumption was too narrow
- direct Batch 2 validation showed these names are package-scoped, not file-local
- future work must either keep them unchanged or treat them as a wider plugin-package rename class with matching validation

### Runtime-Consumer Class

- `server/internal/app/runtime.go`
  - consumes `pluginregistry.OrderedDescriptors`
  - consumes `pluginregistry.BuildPlugins`
  - constructs `plugin.NewRuntimeMetadata`
  - stores and serves `plugin.RuntimeMetadata.OrderedPluginDescriptors`

Verdict:

- runtime-consumer edits cannot lead the migration; they must follow exported authority changes
- Batch 1 records the dependency direction so later slices do not patch consumers first

## Validation Record

- executed:
  - `git diff --check`
- executed for Batch 2:
  - `cd server && go run ./cmd/graft validate backend --stage lint`
  - `cd server && go test ./plugins/auth ./plugins/user ./plugins/rbac ./plugins/audit ./plugins/monitor`
  - `cd server && go build ./cmd/graft`
- Batch 2 observed failures:
  - the narrow `go test` failed when those names were renamed only in `descriptor.go`
  - that failure is the authority evidence for keeping the candidate deferred
- web validation status:
  - `cd web && bun run check` is required before any future cross-boundary slice can be marked `archive-ready`
  - this topic should be downgraded from `archive-ready` if a later batch changes `web` or shared contract consumers without that frontend validation

## Scope Guard

- no compatibility alias, adapter, or fallback layer was introduced
- no physical path rename was attempted
- no archived topic files were edited to continue the old loop in place

## Final Closeout

- result: `archive-ready`
- reason:
  - the bounded topic answered the authority question it opened
  - the only approved tiny migration candidate was directly invalidated by package-scoped Go validation
  - no additional safe batch remains inside the current owned scope without broadening into a different rename class
- follow-up policy:
  - any future rename attempt must open a new bounded topic at the true authority owner instead of resuming this loop
