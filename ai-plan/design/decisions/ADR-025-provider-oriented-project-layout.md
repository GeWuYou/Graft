# ADR-025: Provider-Oriented Project Layout And Legacy Migration

- Status: accepted
- Date: 2026-08-10
- Scope: `server` project organization, provider boundaries, independent agents, runners, integrations, deployment topology, and conformance fixtures
- Supersedes: none; this ADR extends the existing module-oriented modular monolith decisions

## Context

Graft is adding runtime-target, build, credential, deployment and Agent capabilities. Docker and Compose are currently
visible in several implementation paths, but they are external implementations rather than platform business authority.
Continuing to place new code by vendor or by the first feature would make later Kubernetes, Vault, BuildKit or third-party
integrations depend on Docker-shaped assumptions.

The repository already has a stable compile-time module model, `internal/moduleapi`, explicit Runtime lifecycle and a
Task/Submission execution boundary. The layout change must preserve those authorities and must not create a dynamic plugin
platform or a second dependency-injection system.

## Decision

Adopt the target organization and rules in
[`项目文件组织与扩展点设计.md`](../architecture/项目文件组织与扩展点设计.md).

1. Core and Application code depend only on provider-neutral contracts and platform runtime services.
2. `internal/moduleapi` remains the authority for cross-module business capabilities. A future `internal/ports` boundary,
   when justified, is reserved for external capability ports and is not a replacement or general shared bucket.
3. Providers are compile-time registered implementations of those ports. Runtime discovery, hot loading, reflection-based
   plugin systems, marketplaces and provider-owned schedulers are prohibited for the current architecture.
4. Provider-specific SDK/CLI/protocol wrappers are adapters. Business-system connections such as GitHub, GitLab, Jira and
   Slack are Integrations, not Runtime Providers.
5. Builder/Runtime Agents are independent deployable units under `server/agents/**`; Runner remains an external-side-effect
   executor governed by the existing Task/Submission lifecycle.
6. Existing paths, including `server/runner/compose/**`, remain current or legacy-frozen until a capability migration has
   one owner, one implementation, conformance evidence and a deletion decision.

## Consequences

New work can add Docker, Kubernetes, Vault, AWS or third-party implementations without changing Core/Application policy.
The repository temporarily contains current and target concepts, but it does not contain duplicate implementations merely
to satisfy directory shape. Each physical move is a separate authority-first migration slice with its own validation.

This decision does not create directories, Go packages, APIs, migrations, deployment files or test fixtures by itself.

## Rejected Alternatives

- Immediate mass `git mv` of `internal/**`, `runner/**` and module implementations: too broad and would mix authority repair
  with structural churn.
- A generic `internal/ports`/`internal/common` catch-all: duplicates `moduleapi` and weakens ownership.
- Runtime `.so` loading, reflection discovery or a Provider marketplace: conflicts with compile-time modular monolith
  governance and increases deployment/security/versioning complexity.
- Treating GitHub, Jira or Slack as Runtime Providers: confuses external business integrations with platform execution
  capability.
