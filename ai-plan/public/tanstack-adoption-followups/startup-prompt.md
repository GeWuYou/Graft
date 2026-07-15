Continue the `tanstack-adoption-followups` topic.

- governance source: root `AGENTS.md`
- task class: `web`
- recovery source: `parent topic`
- recovery entry: `ai-plan/public/tanstack-adoption-followups/README.md`
- local execution truth: `web/AGENTS.md` and `ai-plan/AGENTS.md`

First rerun the startup preflight, then read the topic README, tracking, and trace. Consume the existing Work Contract;
do not recreate the topic, design, roadmap, or ADR.

Topic objective:

- Complete only evidence-backed P1 Query migrations while retaining module API ownership and Query's server-state boundary.

Guardrails:

1. One module group per batch, with focused tests before broad validation.
2. URL filter state and UI-local state remain outside Query cache.
3. Table, Virtual, Router, and Form require a separately recorded go/no-go decision; Router remains out of scope.
