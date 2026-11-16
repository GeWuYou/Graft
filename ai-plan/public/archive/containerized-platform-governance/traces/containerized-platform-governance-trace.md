# Containerized Platform Governance Trace

## 2026-11-15

- User approved governance and local runtime migration including Build and platform self-update.
- Synchronized baseline is 346854ff5daae3f804f8fad706202c9e983d2067.
- ADR-029 records local/agent selection, Task ownership, independent update Controller and explicit cutover preflight.
- Engineering changes are under implementation. Docker operation, migration and recovery acceptance awaits integrated source.
- Governance scoped tests passed for local scope planning, workflow aggregation and same-stream Go reporting.
- Turbo hash invalidation covered source, generated contract, configuration, environment file, lock, build variables and toolchain identity.
- Direct/cold/hit frontend build measurement was 81.28s / 83.39s / 0.16s on the original production source and Linux x64, Bun 1.3.14, Node 24.19.0.
- Shared dependency reconciliation retained existing version sets except two old root-tool patch nodes unified to existing frontend patches. Exact decisions are in development validation.

- Four validation skill routes now consume root phase timing; product commands remain independently usable. Integrated CI and runtime acceptance remain pending.

- Independent review corrected local consumer-family validation and deleted-source selection. A real helper subprocess fixture verified repository-root test report output.

- Added required private-daemon and frozen official-install CI wiring using source-bound server/web artifacts and strict required-scenario evidence. Runtime results await integrated CI.

- Extended runtime CI to same-commit versioned server binaries, production Controller and standard-Go TLS release helper. Required Update API evidence binds binary hashes, backup bytes and migrated replacement Task receipts; runtime execution awaits integrated source.

## 2026-11-16

- Full integration validation exposed two previously hoisted tooling imports. Root Redocly and Web yaml declarations now match their actual consumers at existing locked versions.
- Clean task dependency install, direct bundle generation and full Web check passed all source gates, production build and 312 files / 2175 tests. Combined diagnostic UI verification remains separately scoped.

- Extended required CI to authenticated Build API publication before Update API, with locked bcrypt helper provenance, Task/artifact/publication/engine checks and explicit secret-exclusion evidence. Runtime results await integrated execution.

- Aligned CI proof validation with completed-install markers, published manifest config-to-image binding and uploaded-context probe hashes. Success and backup-failure migration boundaries are explicit; same-source versioned update covers the existing migration graph invocation.


## 2026-11-16 Closeout

- 完成前四批实现并修复实际运行暴露的注册、公开响应投影、Build 计划校验、Controller 意图与文件权限、持久化 catalog 和恢复状态边界。
- 生产源码 976144490 配合验收读取修正 1d9703272，实际通过两组 24 场景、Build Task 10、Update Task 12 与 Backup 1 的内容及身份核对。
- 新私有 PostgreSQL 数据库恢复 dump 后，users=1、roles=8、runtime_targets=1、applications=3、build_v2_artifacts=1、workspace snapshots=1、migration revisions=151。恢复验证数据库已删除。
- head 795c9a8345720679bc0d9c8cdd0b09b662ae6184 的完整 CI run 43794608388 成功，运行验收、清理和 Required checks 均为 success。
- 本轮 lessons review 新增 L1 LESSON-BACKEND-ENTRYPOINT-EVIDENCE-001，记录正常用户入口、实际 PostgreSQL/SDK 效果、公开消费契约与 owner 历史编码夹具的验证方法，并同步 lessons index。治理级别保持 L1。
- 按原 Work Contract archive=true 移入 public/archive，并移除 active router。源码已实现、运行与 CI 已验收，PR #351 的合并和资源释放继续由负责会话执行。
