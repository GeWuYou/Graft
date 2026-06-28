# Design Inventory And Target IA

## Purpose

This note is the execution artifact for `phase-1-batch-1-design-inventory-and-target-ia-skeleton`.

It defines:

- the current inventory summary for `ai-plan/design/**`
- the recommended target information architecture for the repository's current stage
- the classification direction for existing design documents
- the README responsibility model for the target directories
- the phased migration batches for later rounds

This note is topic-local on purpose. It records the migration plan before any broad repository-wide design-doc moves.

## Current Inventory Summary

### Inventory snapshot

- total Markdown documents under `ai-plan/design/**`: `46`
- current root-level documents under `ai-plan/design/`: `33`
- existing subdirectories:
  - `ai-plan/design/decisions/`: `2` ADRs
  - `ai-plan/design/release/`: `6` release-policy documents
  - `ai-plan/design/graft-design-system/`: `5` frontend page-template documents

### Current shape diagnosis

- The directory is still mostly flat, which keeps file discovery cheap at small scale but makes long-term ownership and scan cost worse.
- Some classification already exists:
  - `decisions/` is the ADR convergence gate.
  - `release/` is a bounded release-policy cluster.
  - `graft-design-system/` is a bounded frontend template cluster.
- Most remaining documents mix several kinds of repository truth in one flat namespace:
  - architecture baseline
  - AI / docs governance
  - backend governance
  - frontend governance
  - cross-cutting platform governance
  - domain capability design
  - authority sub-slices for one domain

### Practical document families in the current set

1. Repository foundation and platform baseline
   - `项目设计.md`
   - `模块与依赖注入设计.md`
   - `前端架构设计.md`

2. AI / documentation / workflow governance
   - `AI任务追踪与恢复设计.md`
   - `AI工具与MCP接入治理规范.md`
   - `AI代码生成与Review规范.md`
   - `代码注释与模块文档规范.md`

3. Frontend design and frontend governance
   - `前端视觉设计规范.md`
   - `分页列表页统一规范与收敛计划.md`
   - `TDesign-MCP-辅助开发规范.md`
   - `graft-design-system/**`

4. Backend and cross-boundary engineering governance
   - `契约治理与魔法值治理规范.md`
   - `数据库表设计与迁移规范.md`
   - `后端查询与数据库访问治理规范.md`
   - `服务端API边界与兼容治理规范.md`
   - `后端安全与信任边界治理规范.md`
   - `后端测试与可维护性治理规范.md`
   - `本地化与i18n治理规范.md`
   - `服务端Locale资源归属与迁移设计.md`
   - `CodeGraph-MCP-辅助开发规范.md`
   - `共享资产复用治理规范.md`

5. Platform capability governance and subsystem design
   - `系统配置模型与渲染设计.md`
   - `缓存治理与系统配置读取加速规范.md`
   - `日志治理开发规范.md`
   - `release/**`

6. Domain capability design
   - `通知中心设计.md`
   - `公告中心设计.md`
   - `容器管理设计.md`
   - `容器Dashboard汇总与实时一致性升级设计.md`
   - `容器资源状态与订阅治理设计.md`
   - `容器运行时事件能力设计.md`
   - `Compose项目管理设计.md`
   - `Access-Log-Authority-Contract.md`
   - `Access-Log-Explorer-Authority.md`
   - `Access-Log-Retention-Authority.md`

## Target IA Recommendation

### Design goals

- Keep `ai-plan/design/**` as repository-wide design truth instead of turning it into a topic archive.
- Reduce flat-directory sprawl without forcing a whole-repo rename in one batch.
- Make ownership visible from the path before a contributor opens the document.
- Keep the first migration stage cheap:
  - minimal directory set
  - explicit README routing
  - phased moves
  - no frontmatter rollout
  - no graph/projection system

### Recommended target structure

```text
ai-plan/design/
  README.md
  architecture/
    README.md
  governance/
    README.md
    ai/
      README.md
    backend/
      README.md
    frontend/
      README.md
    platform/
      README.md
  domains/
    README.md
    compose/
      README.md
    container/
      README.md
    notification/
      README.md
    audit/
      README.md
  release/
    README.md
  decisions/
    README.md
  graft-design-system/
    README.md
```

### Why this structure is the right size for the current stage

- `architecture/`
  - for repository-wide structural truth that explains the baseline architecture, module model, and frontend shell/module structure
- `governance/`
  - for long-lived rules and guardrails rather than capability design
- `governance/ai/`
  - isolates AI / docs workflow governance so future AI-specific rules do not keep polluting the root
- `governance/backend/`
  - groups server-side and cross-boundary engineering guardrails that are primarily backend-authority driven
- `governance/frontend/`
  - groups frontend design/governance rules and TDesign workflow guidance
- `governance/platform/`
  - groups cross-cutting platform rules that are neither purely backend nor purely frontend, such as cache, shared asset, and localization governance
- `domains/`
  - separates capability design from generic governance
- `domains/<domain>/`
  - keeps capability-specific documents together without inventing a deep taxonomy too early
- `release/`
  - stays as an existing stable cluster
- `decisions/`
  - stays as the convergence gate defined by ADR-001
- `graft-design-system/`
  - stays as a bounded template library because it behaves like a reusable design pattern pack rather than ordinary governance prose

### Explicit non-goals for this IA

- no conversion of every design document into ADR form
- no forced English rename of all existing Chinese filenames in this phase
- no whole-repository metadata rollout
- no docs-site navigation model
- no third-level directory taxonomy unless a later batch proves it is needed

## README Responsibility Model

### `ai-plan/design/README.md`

Must define:

- what kinds of documents belong in `design/`
- the directory map for `architecture/`, `governance/`, `domains/`, `release/`, `decisions/`, and `graft-design-system/`
- a short routing rule for deciding whether a new document belongs in design truth, roadmap truth, or topic-local docs
- the migration note that some files may temporarily remain at root during phased convergence

Must not become:

- a duplicate of root `AGENTS.md`
- a second active-topic router
- a full inventory dump of every paragraph in every design doc

### `ai-plan/design/architecture/README.md`

Must define:

- repository-wide baseline architecture scope
- which kinds of structural docs belong here
- when a document should move to `domains/` instead

### `ai-plan/design/governance/README.md`

Must define:

- that this branch holds long-lived rules, guardrails, and authority models
- the split between `ai/`, `backend/`, `frontend/`, and `platform/`
- when a rule should stay in one sub-README rather than at this level

### `ai-plan/design/governance/ai/README.md`

Must define:

- AI workflow, MCP, task recovery, and AI review governance scope
- why these documents are repository-wide design truth rather than topic-local notes

### `ai-plan/design/governance/backend/README.md`

Must define:

- server-authority-driven guardrails
- backend/cross-boundary rule types that belong here
- when a capability-specific backend doc should live under `domains/`

### `ai-plan/design/governance/frontend/README.md`

Must define:

- frontend page, visual, and TDesign design-governance scope
- relation between prose governance docs and `graft-design-system/**` reusable templates

### `ai-plan/design/governance/platform/README.md`

Must define:

- cross-cutting platform governance that is neither one product domain nor one pure frontend/backend cluster
- examples such as cache, localization, shared-asset governance, and system-config shape

### `ai-plan/design/domains/README.md`

Must define:

- that each child directory maps to one capability domain or bounded product area
- that cross-domain baseline rules do not belong here
- that domain READMEs should summarize local authority and reference the contained design docs

### `ai-plan/design/domains/<domain>/README.md`

Must define:

- domain intent
- canonical documents in that domain folder
- nearby cross-domain governance docs that contributors usually need to read with the domain docs

### Existing retained directories

- `ai-plan/design/release/README.md`
  - should explain release-policy scope only
- `ai-plan/design/decisions/README.md`
  - should explain ADR purpose and naming rules only
- `ai-plan/design/graft-design-system/README.md`
  - should remain a template index, not a general frontend-governance router

## Classification Suggestions For Existing Documents

### Keep in place in this batch

- `ai-plan/design/decisions/ADR-001-ai-plan-authority-and-metadata-model.md`
  - keep under `decisions/`
- `ai-plan/design/decisions/ADR-002-ai-plan-lifecycle-and-archive-model.md`
  - keep under `decisions/`
- `ai-plan/design/release/*.md`
  - keep under `release/`
- `ai-plan/design/graft-design-system/*.md`
  - keep under `graft-design-system/`

### Recommended future placement matrix

| Current document | Recommended target path |
| --- | --- |
| `项目设计.md` | `ai-plan/design/architecture/项目设计.md` |
| `模块与依赖注入设计.md` | `ai-plan/design/architecture/模块与依赖注入设计.md` |
| `前端架构设计.md` | `ai-plan/design/architecture/前端架构设计.md` |
| `AI任务追踪与恢复设计.md` | `ai-plan/design/governance/ai/AI任务追踪与恢复设计.md` |
| `AI工具与MCP接入治理规范.md` | `ai-plan/design/governance/ai/AI工具与MCP接入治理规范.md` |
| `AI代码生成与Review规范.md` | `ai-plan/design/governance/ai/AI代码生成与Review规范.md` |
| `代码注释与模块文档规范.md` | `ai-plan/design/governance/ai/代码注释与模块文档规范.md` |
| `后端安全与信任边界治理规范.md` | `ai-plan/design/governance/backend/后端安全与信任边界治理规范.md` |
| `后端查询与数据库访问治理规范.md` | `ai-plan/design/governance/backend/后端查询与数据库访问治理规范.md` |
| `后端测试与可维护性治理规范.md` | `ai-plan/design/governance/backend/后端测试与可维护性治理规范.md` |
| `服务端API边界与兼容治理规范.md` | `ai-plan/design/governance/backend/服务端API边界与兼容治理规范.md` |
| `数据库表设计与迁移规范.md` | `ai-plan/design/governance/backend/数据库表设计与迁移规范.md` |
| `前端视觉设计规范.md` | `ai-plan/design/governance/frontend/前端视觉设计规范.md` |
| `分页列表页统一规范与收敛计划.md` | `ai-plan/design/governance/frontend/分页列表页统一规范与收敛计划.md` |
| `TDesign-MCP-辅助开发规范.md` | `ai-plan/design/governance/frontend/TDesign-MCP-辅助开发规范.md` |
| `CodeGraph-MCP-辅助开发规范.md` | `ai-plan/design/governance/ai/CodeGraph-MCP-辅助开发规范.md` |
| `共享资产复用治理规范.md` | `ai-plan/design/governance/platform/共享资产复用治理规范.md` |
| `本地化与i18n治理规范.md` | `ai-plan/design/governance/platform/本地化与i18n治理规范.md` |
| `服务端Locale资源归属与迁移设计.md` | `ai-plan/design/governance/platform/服务端Locale资源归属与迁移设计.md` |
| `系统配置模型与渲染设计.md` | `ai-plan/design/governance/platform/系统配置模型与渲染设计.md` |
| `缓存治理与系统配置读取加速规范.md` | `ai-plan/design/governance/platform/缓存治理与系统配置读取加速规范.md` |
| `契约治理与魔法值治理规范.md` | `ai-plan/design/governance/platform/契约治理与魔法值治理规范.md` |
| `通知中心设计.md` | `ai-plan/design/domains/notification/通知中心设计.md` |
| `公告中心设计.md` | `ai-plan/design/domains/notification/公告中心设计.md` |
| `容器管理设计.md` | `ai-plan/design/domains/container/容器管理设计.md` |
| `容器Dashboard汇总与实时一致性升级设计.md` | `ai-plan/design/domains/container/容器Dashboard汇总与实时一致性升级设计.md` |
| `容器资源状态与订阅治理设计.md` | `ai-plan/design/domains/container/容器资源状态与订阅治理设计.md` |
| `容器运行时事件能力设计.md` | `ai-plan/design/domains/container/容器运行时事件能力设计.md` |
| `Compose项目管理设计.md` | `ai-plan/design/domains/compose/Compose项目管理设计.md` |
| `日志治理开发规范.md` | `ai-plan/design/domains/audit/日志治理开发规范.md` |
| `Access-Log-Authority-Contract.md` | `ai-plan/design/domains/audit/Access-Log-Authority-Contract.md` |
| `Access-Log-Explorer-Authority.md` | `ai-plan/design/domains/audit/Access-Log-Explorer-Authority.md` |
| `Access-Log-Retention-Authority.md` | `ai-plan/design/domains/audit/Access-Log-Retention-Authority.md` |

### Notes on borderline classifications

- `契约治理与魔法值治理规范.md`
  - It is cross-boundary and affects both `server` and `web`, but it behaves as a generic repository guardrail rather than one product domain design, so `governance/platform/` is the best current fit.
- `日志治理开发规范.md`
  - The filename sounds generic, but the current companion Access Log authority documents make it behave like a bounded audit-domain cluster instead of repository-wide logging-only baseline truth.
- `代码注释与模块文档规范.md`
  - It affects `server`, `web`, and AI-authored docs behavior. For the current stage it fits best beside AI workflow governance because that is where repository-wide documentation-generation behavior is already anchored.
- `CodeGraph-MCP-辅助开发规范.md`
  - It could live under generic tool governance or developer workflow governance. It fits better under `governance/ai/` because the current authority is AI-assisted discovery workflow rather than runtime architecture.

## Phased Migration Batches

### Batch 2: create target directories and README skeletons

Scope:

- add `ai-plan/design/README.md`
- add `architecture/`, `governance/`, `governance/ai/`, `governance/backend/`, `governance/frontend/`, `governance/platform/`, `domains/`, and selected domain README skeletons
- add `release/README.md` and `decisions/README.md` only if missing
- do not move existing design docs yet, except where a README path or tiny anchor file is necessary

Acceptance:

- directory map exists
- README responsibility model becomes repo-visible
- no broad content migration yet

### Batch 3: migrate low-coupling design docs

Scope:

- move architecture docs
- move AI governance docs
- move frontend governance docs
- move platform governance docs with low reference churn
- fix local references caused by those moves

Acceptance:

- the highest-signal governance docs leave the flat root
- references remain navigable
- no domain-heavy document cluster moves yet

### Batch 4: migrate domain design docs and fix references

Scope:

- move `notification`, `container`, `compose`, and `audit` domain clusters
- add or refine domain README summaries
- repair cross-links and mention any domain documents that still need to stay split

Acceptance:

- product/domain documents are clustered by capability
- cross-domain references are repaired
- no unresolved authority ambiguity is hidden by the new paths

### Batch 5: archive naming and governance sync closeout

Scope:

- verify remaining root-level docs are intentional
- decide whether any transitional anchor notes are still needed
- update topic docs for archive-readiness or bounded next work
- sync any minimal shared governance docs only if the final IA must become repository-visible outside the topic

Acceptance:

- the migration leaves a stable IA, not a half-finished experiment
- recovery docs and any required shared docs describe the new steady state

## Minimal Repo-Visible Anchor Decision

For this batch, no shared router or validator update is required.

Reason:

- the target IA remains a planning artifact until Batch 2 creates real directory and README anchors under `ai-plan/design/**`
- `ai-plan/public/README.md` already lists this active topic, so the work is discoverable through the normal recovery path
- the bounded structure guard does not claim to validate every active topic's inner design notes

The first repository-wide visible anchor should be created in Batch 2 by adding README and directory skeleton files under `ai-plan/design/**`.
