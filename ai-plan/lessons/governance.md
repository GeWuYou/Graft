# Governance Lessons

## LESSON-GOVERNANCE-PORTABLE-WSL-MCP-001：共享 MCP 指引不得固化本机 WSL 发行版

- Status: active
- Level: L2
- Applies to:
  - Windows Codex host 通过 WSL 启动用户级 MCP Server 的共享安装指引
  - `.ai/environment/**` 之外的仓库级工具配置示例
- Source:
  - PR #284 的 GitHub MCP 启动命令可移植性审查
- Problem:
  共享文档若写死 `Ubuntu-24.04` 或另一台机器的 distro 名，会把本机环境事实伪装成团队前提；使用其它默认
  distro 的贡献者会在 MCP 启动前失败，即使所需工具已经安装。
- Correct pattern:
  共享示例通过 `wsl.exe -- <command>` 进入当前用户的默认 distro。只有工具确实位于非默认 distro 时，用户才在
  个人 Codex 配置中根据 `wsl.exe --list --quiet` 的结果设置 `--distribution <name>`；该名称不回写仓库共享文档。
- Anti-pattern:
  - 在共享 MCP 命令中固定 `wsl.exe -d Ubuntu-24.04` 或其它设备专属 distro
  - 把 WindowsApps 绝对路径、用户名或个人客户端配置提交为仓库真值
  - 为了兼容单台机器而维护多份平行启动命令
- Enforcement:
  `scripts/validate_ai_governance.py` 要求 GitHub MCP 示例使用默认 WSL launcher，并拒绝共享命令中的硬编码
  distro；对应单元测试必须覆盖一个固定 distro 的负向样例。
- Promotion:
  - AGENTS.md: no
  - Design doc: yes
- Related:
  - `ai-plan/design/governance/ai/AI工具与MCP接入治理规范.md`
  - `scripts/validate_ai_governance.py`
  - `scripts/test_validate_ai_governance.py`
- Updated at:
  2026-08-17

## LESSON-GOVERNANCE-COMMENT-VALUE-001：代码注释必须记录无法从实现推导的决策

- Status: active
- Level: L3
- Applies to:
  - 手写 Go、TypeScript 与 Vue 代码
  - AI 生成、重构或评审涉及注释的任务
  - 代码任务的 closeout 注释审查
- Source:
  - 用户指出 AI 不能以 GoDoc、TSDoc 或 Vue 注释格式为由机械堆叠注释
  - 注释治理基础能力建设
- Problem:
  仅要求 exported symbol、TSDoc 或组件“有注释”会驱动 AI 翻译变量、类型、模板和框架调用。此类注释增加维护成本，
  在实现变化后还会成为误导性的第二份行为说明。
- Correct pattern:
  先判断信息能否由代码与类型推导；不能时，再判断它是否解释设计原因、约束、业务规则、算法原因或外部系统行为。只为
  这些高价值信息添加或更新中文注释。Go 导出符号仍以标识符名开头；TypeScript 使用标准 TSDoc/JSDoc；非平凡 Vue SFC
  只说明组件职责、数据边界和非显然的生命周期/资源语义。每个手写 Go、TS、Vue 改动在 closeout 前执行
  `graft-comment-governance` 增量审查。
- Anti-pattern:
  - 为字段名、原始类型、局部变量、模板元素或 `onMounted` 等框架调用写翻译式注释
  - 用注释数量、覆盖率或仅有 lint 通过证明注释质量
  - 保留已经与实现、调用方或测试冲突的历史注释
  - 用新的 skill 或脚本另建一套与设计规范分离的注释真值
- Enforcement:
  代码评审和 closeout 使用 `graft-comment-governance` 的价值门检查新增、更新和保留的注释；审查变更后的实现而非仅检查注释
  是否存在。生成、第三方、迁移和构建产物按现有豁免边界处理。
- Promotion:
  - AGENTS.md: yes
  - Design doc: yes
- Related:
  - `ai-plan/design/governance/ai/代码注释与模块文档规范.md`
  - `.agents/skills/graft-comment-governance/SKILL.md`
  - `AGENTS.md`
  - `server/AGENTS.md`
  - `web/AGENTS.md`
- Updated at:
  2026-07-16

## LESSON-GOVERNANCE-SCHEMA-AUTHORITY-001：动态配置必须消费 schema 与 i18n authority

- Status: active
- Level: L2
- Applies to:
  - 后端声明 JSON Schema、配置 schema 或表单 schema 的跨边界功能
  - `web` 动态表单、配置弹窗和 schema-form 共享组件
  - 需要同时提供字段本地化、约束校验和后端错误详情的管理页
  - `server/internal/configregistry.Definition` 暴露给 `web` System Config 页的配置元数据
- Source:
  - Scheduled Task 日志保留配置校验修复
  - 用户指出 `batchSize` 超过上限时后端应返回详细错误，前端也应读取后端真值做动态校验和本地化
  - Notification System Config 出现后端英文 fallback，暴露后端动态 i18n key 未进入前端门禁
- Problem:
  后端 schema 已经声明字段、上限和 `x-i18n` 元数据，但前端动态表单若只读取部分字段，或另写一套本地校验/本地文案，
  会导致约束语义漂移。典型表现是 `InputNumber` 传了 `max` 仍允许提交超限值，后端只返回笼统 400，用户无法知道哪个
  字段、哪个约束、实际值和允许值是什么。系统配置类页面还会出现另一种漂移：后端 `Definition` 已给出
  `domain_key`、`group_key`、`title_key`、`description_key` 或 JSON Schema `x-i18n`，但前端 catalog 缺 key 时页面
  回退到后端英文 fallback，普通源码字面量扫描也不会发现动态拼接出的 key。
- Correct pattern:
  后端 schema 是字段、约束和字段级本地化元数据的 authority。后端必须在持久化或 handler 执行前按同一 schema 校验，
  并返回结构化错误详情，例如 `field`、`reason_code`、`constraint`、`minimum`、`maximum`、`expected`、`actual`。
  前端动态表单应消费同一份 schema 渲染输入限制、字段标题和提交前校验；错误文案用通用 reason code 模板加 schema
  字段 `x-i18n` 标题生成，只有字段确实需要特殊文案时才扩展 schema 元数据。系统配置展示还必须把后端
  `Definition` 的 domain/group/item i18n key 以及 schema `x-i18n` key 纳入 `web` locale catalog 和 `lint:i18n`
  required-key 检查；fallback 字段只能兜底未知项，不能成为长期展示真相。
- Anti-pattern:
  - 后端 schema 声明约束，但前端只把它当展示 JSON
  - 为某个字段在前端硬编码最大值、本地字段名或专用错误文案
  - 依赖数据库、repository 或任务 handler 兜底拒绝明显违反 schema 的值
  - 后端返回只有 `invalid_request` 的笼统 400，而没有字段级结构化详情
  - 同时维护 `x-i18n` 和旧式字段本地化 key 作为长期平行真相
  - 只校验前端源码里显式调用的 `t('...')`，漏掉后端动态生成的 System Config 展示 key
- Enforcement:
  修改 schema 驱动配置表单时，检查后端校验、前端提交前校验、字段 i18n 和错误详情是否都来自同一 schema authority。
  测试至少覆盖一个越界值不会持久化/不会触发 handler、一个前端提交被拦截、以及一个字段标题从 `x-i18n` 解析。
  若前端保留 legacy key 兼容分支，测试 fixture 和正常生产路径必须使用 canonical schema 元数据。新增或修改
  `configregistry.Definition` 时，后端测试应确认注册的 display key 在 `zh-CN` / `en-US` 都有 message resource；
  `web` 的 `lint:i18n` 应把这些后端 key 对照前端 locale catalog。
- Promotion:
  - AGENTS.md: no
  - Design doc: no
- Related:
  - `ai-plan/design/governance/platform/契约治理与魔法值治理规范.md`
  - `server/internal/scheduler/config_schema.go`
  - `server/internal/configregistry/definition.go`
  - `web/scripts/check-i18n-governance.ts`
  - `web/src/shared/schema-form/config-schema.ts`
- Updated at:
  2026-06-10

## LESSON-GOVERNANCE-BROWSER-BACKEND-001：浏览器验收需要真实后端时不要停在 mock 登录页

- Status: active
- Level: L1
- Applies to:
  - `web` 页面浏览器验收
  - 需要登录、bootstrap、动态菜单或真实 API 契约的 Playwright 检查
  - 使用 `graft-web-browser-agent` 或本地 Playwright 脚本做 UI 交互验证的任务
- Source:
  - Scheduled Task 配置弹窗 UX 重构浏览器验收
  - `web` 以 `dev:mock` 启动后，登录页可渲染但 `/api/auth/login` 未形成可用会话，导致目标页面无法进入
- Problem:
  `web` 的 mock dev server 不一定覆盖当前页面所需的认证、bootstrap、动态菜单和业务 API。浏览器验收如果只停留在
  mock 登录页，会把真实 UI 检查误判为“需要继续找选择器”或“页面不可达”，浪费调试时间。
- Correct pattern:
  当目标页面依赖登录态或后端动态契约时，直接启动真实后端：
  `cd server && go run ./cmd/graft dev`。确认后端已注册目标 API 后，再通过 `web/.env.development`
  的 `VITE_API_TARGET` 代理访问页面。若当前目录的 `temp` 下提供本地凭据，可用于本机浏览器验收，但不要把凭据写入代码、
  文档正文或测试 fixture。
- Anti-pattern:
  在 `dev:mock` 登录页反复尝试账号密码、改前端选择器或伪造业务结论，而没有先确认认证接口、bootstrap 和目标业务 API
  是否由 mock server 真实承载。
- Enforcement:
  浏览器验收前先检查页面是否需要认证或真实后端数据；若需要，确认 `server` 已启动并能看到目标路由，例如
  `/api/auth/login`、`/api/auth/bootstrap` 和相关业务 API。若登录仍失败，优先检查后端启动状态和代理配置，而不是继续调整
  UI 选择器。
- Promotion:
  - AGENTS.md: no
  - Design doc: no
- Related:
  - `.agents/skills/graft-web-browser-agent/SKILL.md`
  - `README.md`
  - `web/.env.development`
- Updated at:
  2026-06-07

## LESSON-GOVERNANCE-RELEASE-LINT-001：Release tag checkout 不应 prune lint 基准分支

- Status: active
- Level: L1
- Applies to:
  - GitHub Actions release workflow 的 tag checkout
  - 依赖 remote-tracking ref 计算 changed-file lint 基准的 CI 任务
- Source:
  - `v0.11.0-beta.31` 的 Backend Release Validation 失败
- Problem:
  `actions/checkout` 在 tag 场景会收窄 remote fetch refspec。随后使用 `git fetch --prune` 获取
  `origin/main` 时，Git 可能把该 remote-tracking ref 视为不受当前 refspec 管理而删除，令后续 lint 基准失效。
- Correct pattern:
  使用不带 `--prune` 的显式完全限定 refspec，例如
  `git fetch --no-tags origin '+refs/heads/main:refs/remotes/origin/main'`，并在写入 lint 基准前通过
  `git rev-parse --verify 'refs/remotes/origin/main^{commit}'` 验证引用存在。
- Anti-pattern:
  - 在 tag checkout 后使用 `git fetch --prune origin main:refs/remotes/origin/main`
  - 未验证 base ref 是否存在就将其传给 changed-file lint
- Enforcement:
  修改 release workflow 的 lint base 解析时，在 tag-only fetch refspec 的临时仓库中验证目标
  remote-tracking ref 存在并可计算 merge-base。
- Promotion:
  - AGENTS.md: no
  - Design doc: no
- Related:
  - `.github/workflows/publish.yml`
  - `server/internal/cli/validate_backend_lint.go`
- Updated at:
  2026-08-01

## LESSON-GOVERNANCE-EXTERNAL-SKILL-001：外部 skill 应按现有能力边界转换并路由

- Status: active
- Level: L2
- Applies to:
  - 引入或参考第三方 AI skill、plugin、设计工作流或资料包
  - `.agents/skills/**` 与项目级 Codex plugin 的能力演进
- Source:
  - UI/UX Pro Max 适配任务中，用户明确拒绝将外部能力压缩为一个“上帝 skill”
- Problem:
  直接 vendoring 外部 skill 或把其所有能力堆入一个本地入口，会带入无关技术栈、数据、脚本和第二套工作流，
  并模糊现有 skill 的 authority 与职责。
- Correct pattern:
  固定外部对照版本并仅作开发期阅读；按请求意图提取可复用原则，分别收口到已有的 intake、构建、QA、资产/动效
  等窄 skill。外部资料只提供启发式输入，项目设计文档、既有 skill、运行时框架和验证入口继续是 authority。
- Anti-pattern:
  - 整包复制外部 skill、搜索脚本、数据集、字体或框架特定实现
  - 新建覆盖设计、生成、验证和资产决策的通用“上帝 skill”
  - 让外部生成的设计系统、目录或验证流程成为第二套长期真值
- Enforcement:
  外部前端 skill 先经过 `graft-frontend-skill-intake` 路由；变更项目级 plugin 时运行 manifest、skill 结构和
  AI governance 校验，并确认没有引入运行时依赖、个人配置或未经授权的技术栈。
- Promotion:
  - AGENTS.md: no
  - Design doc: no
- Related:
  - `plugins/graft-frontend-vibe-toolchain/skills/graft-frontend-skill-intake/`
  - `.agents/skills/graft-web-vibe-coding/SKILL.md`
  - `ai-plan/design/governance/ai/AI工具与MCP接入治理规范.md`
- Updated at:
  2026-08-17
