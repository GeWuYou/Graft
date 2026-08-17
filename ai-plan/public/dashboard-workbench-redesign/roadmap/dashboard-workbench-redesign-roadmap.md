# Dashboard Workbench Redesign Roadmap

## Phase 1: Preview And Acceptance

- Implement the development-only fixed-scenario workbench in the real application shell.
- Validate five-state semantics, responsive information hierarchy, theme tokens, and i18n.
- Capture desktop and full-page inspection artifacts, then complete at least one visual refinement.
- Keep the formal homepage and all production contracts unchanged.

## Phase 2: Production Web Adoption

- [x] Introduce the accepted fixed-slot presentation projection over the existing dashboard summary.
- [x] Keep widget load status, display state, priority, and business presentation status separate.
- [x] Collapse normal health facts and keep optional-source failure visible without overstating impact.
- Remove the development preview when the formal homepage passes equivalent validation and human acceptance.

## Phase 3: Typed Attention Contract

- Add typed attention items, evidence state, server generation time, and source coverage to the OpenAPI authority.
- Refactor server loaders to return payload, control, metrics, and attention facts separately.
- Authorize actions at the server boundary and execute loaders under a bounded concurrent budget.
- Add container-owned attention contributions rather than deriving container severity in the page.

## Phase 4: Bounded Personalization

- Keep a fixed homepage information budget as module count grows.
- Add stable issue/resource identity and de-duplication before role or user policy.
- Introduce `server/modules/dashboard` only if persisted policy or personalization becomes a real product requirement.
