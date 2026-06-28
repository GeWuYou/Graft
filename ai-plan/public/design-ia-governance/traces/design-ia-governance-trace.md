# Design IA Governance Trace

## 2026-06-28 Topic initialization

- 建立 active topic：`ai-plan/public/design-ia-governance/README.md`
- 建立 startup prompt：`ai-plan/public/design-ia-governance/startup-prompt.md`
- 建立 tracking：`ai-plan/public/design-ia-governance/todos/design-ia-governance-tracking.md`
- 建立 trace：`ai-plan/public/design-ia-governance/traces/design-ia-governance-trace.md`
- 目标明确为 `ai-plan/design/**` 内容本身的 IA 治理，而不是继续只做外围治理基线
- 当前第一批次定义为：
  - design inventory
  - 分类矩阵
  - 目标目录骨架
  - README 责任模型

## 2026-06-28 Batch 1 completed: design inventory and target IA skeleton

- 盘点 `ai-plan/design/**` 当前文档总量：
  - `46` 个 Markdown 文档
  - 其中 `33` 个仍位于 `ai-plan/design/` 根层
  - 已存在子目录仅有：
    - `decisions/`
    - `release/`
    - `graft-design-system/`
- 产出 topic-local 执行文档：
  - `ai-plan/public/design-ia-governance/design/phase-1-design-inventory-and-target-ia.md`
- 收敛的 target IA：
  - `architecture/`
  - `governance/ai/`
  - `governance/backend/`
  - `governance/frontend/`
  - `governance/platform/`
  - `domains/<domain>/`
  - 保留 `decisions/`、`release/`、`graft-design-system/`
- 明确 README 责任模型：
  - root `design/README.md` 负责目录路由
  - 一级目录 README 负责边界定义
  - 二级目录 README 负责域内 authority 与文档入口说明
- authority decision：
  - 本批次只需更新 topic-local recovery 与 design note
  - 共享 router、catalog、validator 现阶段无需同步
- 下一批方向：
  - 在 `ai-plan/design/**` 下创建目标目录与 README 骨架
  - 仍不批量移动 design 文档

## 2026-06-28 Batch 2 completed: target design directories and router readmes

- 在 `ai-plan/design/**` 下建立目标目录骨架：
  - `architecture/`
  - `governance/ai/`
  - `governance/backend/`
  - `governance/frontend/`
  - `governance/platform/`
  - `domains/compose/`
  - `domains/container/`
  - `domains/notification/`
  - `domains/audit/`
- 新增或补齐 router README：
  - `ai-plan/design/README.md`
  - `ai-plan/design/architecture/README.md`
  - `ai-plan/design/governance/**/README.md`
  - `ai-plan/design/domains/**/README.md`
  - `ai-plan/design/decisions/README.md`
  - `ai-plan/design/release/README.md`
- README 责任保持为目录路由与边界定义，不复制现有 design 正文。
- 保留已有目录 `decisions/`、`release/`、`graft-design-system/`，本批次不移动 existing design docs。
- `compose-project-management` 与其他 active topic 的 recovery entry 未改动。
- 下一批方向：
  - 迁移 low-coupling design docs 到 `architecture/`、`governance/`、`release/` 等目标目录
  - 在最小范围内修复引用，不扩大到 domain-heavy 文档迁移

## 2026-06-28 Batch 3 completed: low-coupling design-doc migration and path repair

- 使用 `git mv` 迁移低耦合 canonical design docs：
  - `architecture/`：
    - `项目设计.md`
    - `模块与依赖注入设计.md`
    - `前端架构设计.md`
  - `governance/ai/`：
    - `AI任务追踪与恢复设计.md`
    - `AI工具与MCP接入治理规范.md`
    - `AI代码生成与Review规范.md`
    - `代码注释与模块文档规范.md`
    - `CodeGraph-MCP-辅助开发规范.md`
  - `governance/backend/`：
    - `后端安全与信任边界治理规范.md`
    - `后端查询与数据库访问治理规范.md`
    - `后端测试与可维护性治理规范.md`
    - `服务端API边界与兼容治理规范.md`
    - `数据库表设计与迁移规范.md`
  - `governance/frontend/`：
    - `前端视觉设计规范.md`
    - `分页列表页统一规范与收敛计划.md`
    - `TDesign-MCP-辅助开发规范.md`
  - `governance/platform/`：
    - `共享资产复用治理规范.md`
    - `本地化与i18n治理规范.md`
    - `服务端Locale资源归属与迁移设计.md`
    - `系统配置模型与渲染设计.md`
    - `缓存治理与系统配置读取加速规范.md`
    - `契约治理与魔法值治理规范.md`
- 在最小必要范围内修复 repository path 引用：
  - root / subdomain governance docs
  - `ai-plan/public/**` active 与 archive recovery materials
  - `.agents/skills/**` 与 `plugins/graft-frontend-vibe-toolchain/skills/**`
  - `scripts/validate_ai_governance.py`
- `compose-project-management` recovery docs 已同步新 canonical 设计路径，保持 startability。
- 验证结果显示一个额外 authority owner：
  - `.ai/registries/cross-boundary-assets.yaml` 仍保留旧 design authority path
  - 该 path drift 会导致 `python3 scripts/validate_ai_governance.py` 失败
  - 因该 registry 不在当前 batch 继承 owned scope 内，本轮不越界修复，改为新增一个最小 follow-up batch
- 本批次未移动 domain-heavy docs：
  - `通知中心设计.md`
  - `公告中心设计.md`
  - 容器 / compose / access-log / 日志治理文档
- 下一批方向：
  - 先同步 shared-asset registry path
  - 再迁移 domain-oriented design docs 并收敛剩余 cross-reference repair

## 2026-06-28 Batch 3b completed: shared-asset registry path sync

- 修复 `.ai/registries/cross-boundary-assets.yaml` 中三处 stale low-coupling design authority path：
  - `graft-shared-asset-reuse-skill` example 指向 `ai-plan/design/governance/platform/共享资产复用治理规范.md`
  - `cache-governance-design` canonical path 指向 `ai-plan/design/governance/platform/缓存治理与系统配置读取加速规范.md`
  - `graft-cache-governance-skill` example 指向 `ai-plan/design/governance/platform/缓存治理与系统配置读取加速规范.md`
- 同步 `scripts/plugin_residual/allowlist.json` 中已迁移 low-coupling design truth 的 allowlist path：
  - `项目设计.md`
  - `模块与依赖注入设计.md`
  - `契约治理与魔法值治理规范.md`
- 修复同批次遗留的下游文档引用：
  - `ai-plan/lessons/governance.md`
  - `ai-plan/lessons/index.md`
  - `ai-plan/lessons/web-ui.md`
  - `server/internal/app/README.md`
  - `server/internal/store/README.md`
  - `server/internal/module/README.md`
  - `server/internal/container/README.md`
  - `server/internal/eventbus/README.md`
- 验证恢复通过：
  - `git diff --check`
  - `python3 scripts/validate_ai_governance.py`
  - `python3 scripts/validate_ai_plan_structure.py`
- 下一批方向：
  - 进入 Batch 4，迁移 domain-oriented design docs 并收敛剩余 cross-reference repair

## 2026-06-28 Batch 4 completed: domain design-doc migration and path repair

- 使用 `git mv` 迁移剩余 domain-oriented canonical design docs：
  - `domains/notification/`
    - `通知中心设计.md`
    - `公告中心设计.md`
  - `domains/container/`
    - `容器管理设计.md`
    - `容器Dashboard汇总与实时一致性升级设计.md`
    - `容器资源状态与订阅治理设计.md`
    - `容器运行时事件能力设计.md`
  - `domains/compose/`
    - `Compose项目管理设计.md`
  - `domains/audit/`
    - `日志治理开发规范.md`
    - `Access-Log-Authority-Contract.md`
    - `Access-Log-Explorer-Authority.md`
    - `Access-Log-Retention-Authority.md`
- 在最小必要范围内修复 downstream canonical path 消费方：
  - root `AGENTS.md`
  - `.agents/skills/graft-cache-governance/SKILL.md`
  - `scripts/plugin_residual/allowlist.json`
  - `ai-plan/public/compose-project-management/**`
  - 仍引用这些 authority 的 archived recovery materials
- 同步 moved-doc 自引用与 sibling domain cross-links，避免保留已失效的旧根层路径。
- 更新 `ai-plan/design/README.md`、`ai-plan/design/domains/README.md` 与四个 domain README，使 `design/` 根层回到 router-only 角色，并把 `domains/*/README.md` 升级为当前 canonical 文档入口。
- 验证恢复通过：
  - `git diff --check`
  - `python3 scripts/validate_ai_plan_structure.py`
  - `python3 scripts/validate_ai_governance.py`
- 下一批方向：
  - 进入 Batch 5，完成 archive / naming / governance sync closeout

## Loop Batch State

```json
{
  "loop_mode": "topic-completion-loop",
  "completed_batches": [
    "phase-1-batch-1-design-inventory-and-target-ia-skeleton",
    "phase-1-batch-2-create-target-design-directories-and-readmes",
    "phase-1-batch-3-migrate-low-coupling-design-docs",
    "phase-1-batch-3b-sync-shared-asset-registry-paths",
    "phase-1-batch-4-migrate-domain-design-docs-and-fix-references"
  ],
  "pending_batches": [
    "phase-1-batch-5-design-archive-naming-and-governance-sync-closeout"
  ],
  "current_batch": "phase-1-batch-4-migrate-domain-design-docs-and-fix-references",
  "next_batch": "phase-1-batch-5-design-archive-naming-and-governance-sync-closeout",
  "closeout_status": "batch-4-complete"
}
```
