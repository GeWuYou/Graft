---
name: graft-navigation-route-governance
description: Classify Graft navigation, UI routes, object capabilities, and global pages before changing menus, route metadata, bootstrap navigation, or page entry points. Use for new or moved menu items, UI route migrations, global pages, and navigation design review.
---

# Graft Navigation Route Governance

Use this skill before changing a Graft menu item, UI route, route-registration metadata, global entry point, or object-detail action. The canonical authority is `ai-plan/design/architecture/导航与资源路由信息架构规范.md`; this skill applies it and does not create a second navigation taxonomy.

## Required Classification

Classify the proposed capability before editing any menu or UI route:

1. Identify the user-managed object and its lifecycle owner.
2. Classify it as exactly one of: domain object, shared resource, platform capability, object-detail capability, or global page.
3. For a navigable domain object or platform capability, choose one canonical domain: Application, Infrastructure, Build, Resources, Observability, Security, or Platform.
4. Choose a `/<domain>/<resource>` UI URL for visible navigation entries. The resource must be stable; Runtime technology and creation source must not become URL hierarchy.
5. Record the menu parent, route owner, permission owner, and whether it is a visible entry, an object-detail tab/action, or a global entry.

Do not render an empty domain group. Build and Resources remain canonical domains even when they have no current authorized visible entries.

## Placement Rules

Apply these rules in order:

- A user-managed deliverable such as Project or Template belongs to Application. A Template Repository is a shared Resource.
- A host, container, cluster, network, storage, or runtime environment belongs to Infrastructure.
- A reusable credential, registry, secret, config, or repository belongs to Resources and is referenced rather than run.
- Logs, metrics, alerts, traces, events, diagnostics, dependencies, and runtime status belong to Observability.
- Users, roles, permissions, tokens, API keys, and audit belong to Security.
- Settings, automation, announcements, plugins, license, and platform administration belong to Platform.
- Terminal, files, logs, exec, shell, and console are object capabilities. Place them in the owning Host, Project, or Container detail instead of creating a first-level menu.
- Notification Center is a global page, not a visible navigation domain entry.
- Docker, Compose, Kubernetes, Helm, OCI, Git, SSH, and creation sources are implementation or pipeline attributes. They must not become navigation domains or URL hierarchy.

Visible entry URLs use these domain slugs: `applications`, `infrastructure`, `build`, `resources`, `observability`, `security`, and `platform`. Domain groups themselves remain graph-only and have no route. Menu-external global pages may use their own dedicated URLs.

## Unresolved Placement

Stop before adding a menu item, UI route, alias, redirect, compatibility node, or placeholder when any of these remain unresolved:

- the user-managed object or lifecycle owner;
- whether the feature is a shared resource, an object capability, a global page, or a visible entry;
- the canonical domain or menu parent; or
- the stable resource boundary for the UI URL.

Report the blocker in this form and request a decision before continuing:

```text
Navigation placement blocked: <feature> has no resolved <object/domain/resource boundary>.
Reason: <why the authority and current evidence do not determine placement>.
Decision needed: choose one of <concrete option A> or <concrete option B>, including the owning object/domain and canonical UI URL.
```

Do not infer a temporary Runtime-, Source-, or implementation-based location merely to unblock implementation. Escalate to the design authority when a new stable domain is genuinely required.

## Change Checklist

For a resolved change, update the highest canonical owner first. When a server menu contract, OpenAPI/typed contract, and web bootstrap consume the same navigation semantics, classify the work as cross-boundary and follow their respective governance documents. UI route migration must not retain aliases or redirects unless a separately approved compatibility exception records the canonical authority, consumers, expiry, and validation.
