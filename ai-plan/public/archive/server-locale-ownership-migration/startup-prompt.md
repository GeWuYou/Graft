Start the next delegated round under the same topic-completion-loop.

Round context:
- governance source: root `AGENTS.md`
- task class: `server`
- recovery source: `parent topic`
- recovery entry: `ai-plan/public/server-locale-ownership-migration/README.md`
- design authority:
  - `ai-plan/design/本地化与i18n治理规范.md`
  - `ai-plan/design/服务端Locale资源归属与迁移设计.md`
  - `ai-plan/design/模块与依赖注入设计.md`
- AI skills:
  - `$graft-localization-governance`
  - `$graft-multi-agent-loop`

Topic objective:
- Continue the server locale ownership migration under `topic-completion-loop` until the topic reaches `archive-ready`, becomes `blocked`, or new bounded batches must be defined.

Locked architecture decisions:
1. `server/internal/i18n` owns only i18n infrastructure:
   - YAML parsing
   - locale/key validation
   - duplicate checks
   - registry construction
   - `Lookup` / `Message`
   - `Freeze`
   - diagnostics
2. Resource ownership:
   - `core` / `display` stay in `server/internal/i18n/locales/*.yaml`
   - business module copy lives in `server/modules/<module>/locales/*.yaml`
   - module-runtime copy lives in `server/internal/moduleruntime/locales/*.yaml`
3. `server/internal/i18n` must not reverse import `server/modules/*`
4. Owner packages may `go:embed locales/*.yaml`, but only expose read-only embedded resource descriptors
5. Runtime preregisters locale resources before `Freeze` and before module `Register`
6. `registerMessages()` remains key-existence validation only, not a loader
7. compile-time registry / equivalent descriptor carries locale providers or lists
8. Disabled modules' locale resources still register by default
9. `system-config` is module-owned locale
10. No `go-i18n`, no stable-key change, no HTTP wire-shape change, no long-lived dual-source compatibility
