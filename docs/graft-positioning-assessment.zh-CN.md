# Graft 定位与 Docker 可靠性评估

评估日期为 2026-11-15。Graft 的调查基线为 `3989727b06bf446a03bcabceba0246c43bf3f181`，升级运行镜像源码为 `c274ac9236a240cf209638434fd370692e204a2d`，Arcane 固定比较版本为 `v3.0.0`。目标用户为个人与小团队的 Docker/Compose 运维人员。本报告提出后续产品取舍，并保留依赖升级阶段的历史隔离证据。三项既有断点及复验中发现的 Compose 标签缺口的最终修复复验源码为 `fbe7b39e225d45fb834d81b4b738602ebc604355`，最新运行结论见“既有断点修复复验”。

## 后续批准的实施方向

用户于 2026-11-15 批准单机 Docker/Compose 优先、后续容器化建站的演进方案。
工程治理与本机执行迁移覆盖 Build 和平台自更新，决策见
[ADR-029](../ai-plan/design/decisions/ADR-029-local-application-operations.md)，
阶段与实际进度分别见[路线](../ai-plan/roadmap/MVP实施计划.md)和
[主题归档](../ai-plan/public/archive/containerized-platform-governance/README.md)。
以下隔离复验是所列历史源码的运行证据，新执行边界的验收另行绑定交付 revision。

## 当前判断

Graft 已拥有应用工作区、Compose 生命周期、Docker 资源清单、日志、终端、权限、任务与审计等实现。依赖升级阶段的隔离基线复现了 Agent 健康、读取和 Shell 正常、写操作 Task 停在 ready 的问题，真实 mTLS 请求与源码将原因定位到执行入口的身份字段不匹配。该边界、Compose 导入的 SQL 参数顺序和 Agent `--once` 成功后退出的行为现已修复。复验进一步发现直接使用 Compose SDK 时遗漏 CLI 准备的归属标签，导致 stop Task 成功而容器仍运行，该缺口也已修复。相同私有引擎中的镜像拉取、容器启停重启、Compose 部署和相对挂载均通过，导入的四项身份字段经 API 与 PostgreSQL 读回一致。执行中断后 Task 收敛为 `needs_attention`，标记 `agent_execution_interrupted`，已产生的应用容器保留且没有重复启动，完整自动恢复仍需独立验收。现有能力覆盖面可以支撑一个单机 Docker/Compose 运维工具，后续功能收拢应依据安装成本、诊断和恢复证据推进。实现证据见 [应用模块](../server/modules/project)、[容器模块](../server/modules/container) 和 [任务模块](../server/modules/task)。

建议下一阶段围绕“安装并连接本机 Docker、导入或创建 Compose 应用、部署和更新、查看日志、定位失败”五个用户任务安排工作。将本机运维作为默认交付目标，远程控制、构建平台与平台自更新分别设置独立的可用性证据和维护预算。

Arcane 的默认安装将管理程序、数据库文件和本机 Docker 连接放在一套配置内，远程主机再引入 Agent。Graft 的默认 Compose 同时依赖 PostgreSQL、Redis、server、web 和 Runtime Agent，Agent 激活还涉及外部 Vault PKI、双 TLS listener、delivery 与配置材料。这个差异直接影响首次操作所需的步骤与可失败边界。[Arcane 安装文档](https://getarcane.app/docs/get-started/installation)、[Graft Compose](../compose.yml)、[Agent 部署说明](../server/agents/docker-runtime-agent/README.md)

## 功能对照

表中“源码实现”表示存在可追踪的生产入口。“部分”表示能力边界或交付范围仍有限。“待运行验证”表示本轮尚未建立该功能的真实环境结果。Arcane 的对应能力以固定版本源码与官方文档共同核对，官方文档后续可能更新。

| 用户能力 | Arcane v3.0.0 对应能力 | Graft 当前实现与边界 | 本轮运行证据 | 建议 |
| --- | --- | --- | --- | --- |
| 首次安装与本机连接 | 默认单管理容器，本机 socket，内置 SQLite，可选 PostgreSQL | Compose 提供 PG/Redis/server/web/Agent，初始材料准备分散 | 基线默认样例环境 config validate 通过，完整交付依赖 fixture 准备 | 优先简化首次可用路径与配置诊断 |
| Compose 创建、编辑与部署 | 项目文件夹、YAML/.env 编辑、Up/Down/Redeploy | 应用 ID、工作区、配置快照、生命周期配置、Task/Agent 执行均有源码实现 | 历史两版 up 超时，断点修复后 create 201、up 与 stop Task 成功，相对挂载内容正确 | 保留，继续完善失败诊断与恢复 |
| 现有 Compose 导入 | 从项目目录发现与管理已有文件 | 目录来源、inspect、冲突检查、import、运行时候选有实现 | 历史两版 import 500，修复后导入 200，API 与 PostgreSQL 四字段读回一致，重复导入 409 且记录保持 | 保留，完善导入冲突的用户反馈 |
| 模板 | 模板及模板注册源 | 版本化模板、发布、撤回、克隆与创建入口有实现 | 源码实现，模板全流程待验证 | 保留最常用模板，注册源扩展延后 |
| 单容器管理 | 清单、详情与生命周期 | 清单/详情在后端读 Docker，启停删除通过 Task/Agent | 历史两版读取通过而写任务超时，修复后启停重启 Task 与实际引擎状态均通过 | 保留，优先改善操作失败定位 |
| 镜像、网络、卷 | 资源管理、清理、拉取 | 资源清单、详情及受控写操作有实现 | 历史两版拉取 Task 超时，修复后拉取成功，其余资源管理待验证 | 保留高频操作，危险操作维持确认和审计 |
| 日志、终端、事件 | 实时日志、资源信号、终端 | 日志快照、实时订阅、Shell ticket/WebSocket 与事件有实现 | 两版日志和真实 WebSocket Shell 命令通过，事件流待验证 | 保留，按实际故障路径提升诊断能力 |
| 指标与运行健康 | CPU/内存/网络等运行指标 | 容器采样、运行目标与平台监控有实现 | 源码实现，连续运行准确性待验证 | 保留支撑运维任务的指标 |
| 远程多主机 | 远程 Agent 主动连接管理器，聚合多主机资源 | 有 Agent 身份、generation、ledger 与租约协议，当前 Docker discovery/读取仍绑定本地 socket | 两版 Agent 身份与重启 receipt 通过，通用远程资源闭环待验证 | 延后，先完成单机路径 |
| 镜像构建与仓库 | 构建、仓库与推送 | Build 规划、快照、Registry 凭据、Moby/ORAS 执行有实现 | 源码实现，本轮构建发布全流程待验证 | 延后独立产品线，保留已有数据和边界 |
| Git 同步与 GitOps | 拉取、推送、双向同步及生命周期钩子 | 当前 creation method 为 blank/template/import，现行路由缺少持续 Git 同步闭环 | 当前源码未证实等价能力 | 按明确使用需求再评估 |
| 应用镜像自动更新 | 镜像发现、延迟、标签策略和 Compose 更新 | 已有平台版本检查与平台自更新控制器，应用 redeploy 使用显式生命周期策略 | 部分，持续应用自动更新缺少证据 | 优先手动预览和可恢复更新 |
| 备份与恢复 | 应用卷备份、计划、S3 与管理器备份 | platform-backup 生成 Graft 配置元数据和 PostgreSQL dump，保存摘要与恢复证据 | 平台备份有源码实现，应用卷备份缺少证据 | 明确备份对象，先演练平台恢复 |
| 认证与权限 | 角色权限、OIDC、passkey/MFA、恢复 | 本地认证、会话、个人令牌、RBAC 与强制首次改密有实现 | 源码实现，认证扩展缺少证据 | 保留现有安全底座，认证扩展延后 |
| 通知 | 多通道通知及运行事件 | 当前通知中心与来源去重、站内投递有实现 | 源码实现，外部通道未证实 | 保留站内关键故障通知 |
| 安全 | 漏洞扫描与安全功能 | security 模块聚合 RBAC 姿态和审计信息 | 安全概览有源码实现，镜像 CVE 扫描未证实 | 明确安全概览职责，扫描延后 |
| API 与平台能力 | REST/OpenAPI、CLI 与运维配置 | OpenAPI、模块显式装配、权限、Task、调度与审计有实现 | 服务端与 Web 完成态检查通过，运行限制见隔离结果 | 保留，控制平台抽象的新增范围 |
| Podman、Swarm 与集群 | 存在专门模块及管理入口 | 部分容器元数据识别包含编排器来源，生产 discovery 当前仅本机 Docker | 部分，专用管理闭环未证实 | 延后独立提供方扩展 |

Arcane 对照入口包括 [v3.0.0 模块树](https://github.com/getarcaneapp/arcane/tree/v3.0.0/backend/internal)、[Compose 项目](https://getarcane.app/docs/docker/projects)、[远程 Agent](https://getarcane.app/docs/remote/agents)、[Git Sync](https://getarcane.app/docs/docker/git-sync)、[自动更新](https://getarcane.app/docs/docker/auto-updates)、[备份](https://getarcane.app/docs/docker/backups)。

Graft 的特定边界可从 [当前创建方式](../server/modules/project/service_source_import.go)、[本地 discovery](../server/modules/runtime-target/discovery.go)、[通知配置](../server/modules/notification/config.go)、[备份工件](../server/modules/backup/artifact_writer.go)、[安全概览](../server/modules/security/module.go) 和 [平台更新](../server/modules/update) 核对。

## 反复修复背后的机制

### 历史 P0，执行入口缺少完整受信身份，现已修复

隔离基线的镜像拉取请求返回 HTTP 202，Task 保持 `ready`、Stage 保持 `pending`，外部执行 lease 为空。Agent 已完成 mTLS、健康检查与重启后的 receipt 推进，其 binding 包含四项 capability 和 `docker/v1`。使用同一真实证书发起无任务的 `oci-build` claim，仍返回 HTTP 400 `COMMON_INVALID_ARGUMENT`。这个最小探针避开了认领写操作的副作用。

mTLS 中间件从 URI SAN 解析 TargetID、AgentID，并添加证书序列号与公钥指纹。它写入请求上下文时，IdentityID 仍为空，Generation 仍为零。执行入口随后用 `agentIdentityMatchesBinding` 要求这些值与已激活绑定的 IdentityID、Generation 完全相等。正常激活绑定具有非空 IdentityID 和正 generation，因而 claim 被稳定拒绝，Task 从未进入 provider 执行。Ledger 路径通过 Runtime Target authority 使用证书证据解析已保存身份，可以完成健康与 receipt 流程。这解释了同一个 Agent 健康与任务停滞同时出现的机制。[证书身份提取](../server/internal/httpx/agent_mtls.go)、[执行入口比较](../server/internal/httpx/agent_execution_routes.go)、[Ledger 入口](../server/internal/httpx/agent_ledger_routes.go)

Git blame 显示，严格的完整身份比较在 `fb0d29a14c` 的 2026-08-21 Application/Container 执行迁移中引入，证书身份生产者从 2026-08-10 的 `8ac11bc5f2` 起继续只提供证书可验证字段。执行路由测试直接向 Gin 注入身份，mTLS 测试则明确验证 Generation 为零，缺少把真实中间件、当前 binding 与 claim handler 串起来的正向回归。[执行迁移提交](https://github.com/GeWuYou/Graft/commit/fb0d29a14c)、[执行路由测试](../server/internal/httpx/agent_execution_routes_test.go)、[mTLS 测试](../server/internal/httpx/agent_mtls_test.go)

修复在 HTTP 执行路由组新增受信身份解析，依据已验证证书的 TargetID、AgentID、序列号与公钥指纹匹配当前 active binding，并校验撤销状态、有效期、非空 IdentityID 和正 Generation。解析后的完整身份统一供 claim、renew、material、logs、result 与 receipt 使用，长轮询的每次重新领取前再核对当前绑定。真实证书中间件回归覆盖完整执行链与过期、撤销、证书和主体不匹配等拒绝条件，真实运行结果见下文复验。受信绑定读取与 Task 领取属于既有的两个事务边界，当前修复覆盖入口与每轮领取前的验证，两次操作之间并发轮换的原子性仍需单独评估。[修复后的身份边界](../server/internal/httpx/agent_execution_identity.go)、[中间件到执行链回归](../server/internal/httpx/agent_execution_mtls_test.go)

以上因果链保留为历史基线证据。断点与 Compose 归属标签修复后的真实镜像拉取、容器生命周期、Compose 部署与停止已经通过，后续架构取舍可据此继续评估。修复跟踪见 [执行身份问题 #342](https://github.com/GeWuYou/Graft/issues/342)。

### Docker 依赖组合存在真实编译约束

提交 `7b1551f5d7280b48b9f3f30bae6a86a819fa15b6` 曾将 Docker CLI 29.2.0、compose-go 2.14.0 和 go-archive 0.3.3 回退到旧组合。积压 PR #336 再次单独提升 archive 等包后，旧 CLI 的构建代码引用了已经消失的 `archive.Gzip`。这个问题能由编译日志直接证实，后续应将 Compose、CLI、Moby、BuildKit 和 archive 按上游发布依赖图迁移并验证。Compose v5.7.1 的发布源码已经使用 CLI 29.9.3 与新的 Moby 模块。[历史回退提交](https://github.com/GeWuYou/Graft/commit/7b1551f5d7280b48b9f3f30bae6a86a819fa15b6)、[PR #336](https://github.com/GeWuYou/Graft/pull/336)、[Compose v5.7.1 依赖](https://github.com/docker/compose/blob/v5.7.1/go.mod)

### “读取在线”与“能够执行”由不同路径证明

后端通过本地 Unix socket 获取容器清单、日志和运行目标事实。写操作需要 Task 外部执行租约、有效 Agent generation、capability、交付与 mTLS，再由 Agent 的 Moby/Compose 提供方执行。因而清单可读只证明后端的 Docker 连接可用。Agent 未交付、离线、配置路径错误或能力未激活时，操作仍可能在另一个边界失败。根 Compose 同时保留后端与 Agent 的 socket，迁移还留有显式的读取路径。这些均为当前源码事实，具体现场故障需要绑定其请求与 Task 结果。[读取入口](../server/modules/container/docker_runtime.go)、[容器 Task](../server/modules/container/task_lifecycle.go)、[Agent 执行边界](../ai-plan/design/decisions/ADR-026-docker-runtime-agent-execution-boundary.md)

### 快速启动的默认配置缺少 Agent 激活闭环

`compose.env.example` 默认将 bootstrap TLS、Agent TLS 和 Credential Vault 设为 false，根 Compose 却默认启动 Agent。Agent README 要求首次运行前准备 Vault PKI、证书和 delivery 产物。本轮对默认样例环境补入合成密码和 JWT 后运行配置校验，退出码为零。该结果覆盖配置语法与解析，Agent 配置和信任材料的来源仍需额外完成。后续优先把首次安装的可执行步骤、就绪判断和失败提示收敛到同一入口，并用全新数据目录验证。[默认环境模板](../compose.env.example)、[快速开始](../README.zh-CN.md)、[Agent README](../server/agents/docker-runtime-agent/README.md)

### Claim 拒绝没有进入执行就绪与用户诊断

`pollExecutionCapabilities` 遇到普通 claim 错误后继续循环，`runExecutionCycle` 在身份准备、mTLS client 或 journal 恢复失败时也直接返回当前状态。Agent 的 healthcheck 校验本地证书、generation、过期时间和信任束，缺少 claim 接受状态。因此已经持有有效证书的 Agent 可以保持容器 healthy，同时持续收到 claim 400，公共操作 Task 长时间停留 ready。该状态在本轮真实基线中已经观察到。[执行循环与 healthcheck](../server/agents/docker-runtime-agent/internal/execution.go)

后续建议增加执行 admission 的有界诊断。以稳定阶段码、失败类别和关联标识投影当前可执行性，把正常无工作 claim 的 204 作为执行通道验收证据之一，并区分无工作、身份拒绝和传输失败。相应状态需要进入 Runtime Target capability 与操作结果的用户反馈。诊断输出应有去重和频率上限，并校验凭据与 provider 详情脱敏。

### 提供方诊断信息在返回路径上被压缩

Agent 的 `loadComposeProject` 将 SDK stdout/stderr 设为 `io.Discard`，多数提供方错误被映射到 `provider_operation_failed`。ADR-026 限制原始 provider stderr 进入 Task 日志与 receipt，目的是保护凭据、路径与边界。当前设计同时压缩了运维人员区分配置解析、拉取、挂载和启动错误所需的信息。建议补充有界的结构化失败阶段、稳定诊断码和受控脱敏详情，并提供与 Task 关联的诊断入口。该建议需要独立契约设计与隐私测试。[提供方实现](../server/agents/docker-runtime-agent/internal/provider.go)、[外部执行决策](../ai-plan/design/decisions/ADR-026-docker-runtime-agent-execution-boundary.md)

### 既有 fixture 的通过范围有限

原 conformance runner 验证 enrollment、证书签发、mTLS、ledger 和重启身份。它的八个阶段没有从公共容器或应用 API 触发 Docker 生命周期。当前 driver 还会调用本地 discovery，原 Compose 的 driver 服务缺少 socket 挂载。本轮隔离 overlay 为 Backend、Agent 和 driver 明确配置相同的私有引擎，并新增实际 API 操作与引擎状态核对。[原 runner](../tests/conformance/docker-runtime-agent/run.sh)、[driver 的目标解析](../server/modules/runtime-target/conformance_driver.go)、[隔离 overlay](../tests/conformance/docker-runtime-agent/compose.isolated.yml)

### 历史 P1，Compose 导入参数顺序错位，现已修复

基线的 managed create 返回 HTTP 201，随后针对另一个独立工作区的 inspect 成功，import 返回 HTTP 500 `COMMON_INTERNAL_ERROR`。关联请求日志给出的具体错误为 `upsert application` 的第六个绑定参数无法编码为 PostgreSQL varchar，传入值是 Runtime Target 数值 ID。

`upsertApplication` 的前八个参数按 application ID、adapter、application name、workspace、display name、runtime target、Compose name、Compose name source 排列。`applicationsUpsertSQL` 的对应列却把 Compose name 和 source 放在 display name 与 runtime target 之前。因此数值 runtime target 被绑定到 `compose_project_name_source`。独立的 `createApplication` 路径按 SQL 列顺序传参，可以成功创建。Git blame 将这处 SQL 与既有参数顺序的分离定位到 2026-07-17 的 `f5fea4eaf2`。基线回归只统计占位符和列的数量。[SQL 参数与列定义](../server/modules/project/store/sql_repository.go)、[占位符回归](../server/modules/project/store/sql_repository_review_regressions_test.go)

修复将 upsert 参数恢复为 SQL 列的有序映射，直接仓储回归核对 22 项字段与参数、可空值及 ON CONFLICT 更新列的映射。真实 PostgreSQL 中的目录 inspect、import 和持久化读回已经通过，runtime target、Compose name/source、display name 均正确。公共导入会先拒绝已注册工作区，重复导入返回 409 且原记录保持一致。ON CONFLICT 更新分支的证据来自直接仓储回归，真实 PostgreSQL 的更新执行仍待单独覆盖。该缺陷发生在导入持久化阶段，独立于 Agent 的执行身份问题。修复跟踪见 [导入问题 #343](https://github.com/GeWuYou/Graft/issues/343)。

### Compose SDK 归属标签遗漏，复验发现后已修复

首次修复快照 `00e985dd4c251e8ee9a0cafb15283606f8469456` 使 Compose up 开始实际创建容器，但探针最初只凭 stop Task 的 `success` 判断停止完成。随后独立 inspect 发现容器仍在运行，重新提交 stop 也得到相同结果。实际容器只有 SDK 生成的 config-hash、container-number 等标签，缺少 Project、Service 和 Oneoff 归属字段。该证据证明既有探针对引擎最终状态的覆盖不足。[提供方加载边界](../server/agents/docker-runtime-agent/internal/provider.go)、[强化后的真实状态断言](../scripts/validate_docker_isolated.py)

`loadComposeDefinition` 原先直接返回 `LoadProject` 结果。Compose CLI 在其加载阶段准备的服务 CustomLabels 没有由嵌入式调用者补齐，SDK 按项目标签查找停止对象时得到空集合，因而 stop 可以成功返回而没有停止容器。升级前的 v2 调用路径也存在这一加载缺口。修复在受控加载边界补齐项目、服务、工作区、配置文件与普通容器标识，归属标签优先于用户声明的冲突值。最小回归先复现 Task 成功与引擎状态不一致，再要求 up 的 Running=true、stop 的 Running=false，以及容器 restart 的 StartedAt 实际变化。[标签回归](../server/agents/docker-runtime-agent/internal/provider_compose_labels_test.go)、[问题 #347](https://github.com/GeWuYou/Graft/issues/347)

最终快照验证了新建 Compose 资源的核心归属标签、实际启停和最终引擎状态。嵌入式 SDK 的 ComposeVersion 当前为空字符串，版本标签沿用上游字段，运行源码身份由构建记录和镜像源码标签证明。标签修复对新建或经过受控重建的资源生效。现场旧版创建的无归属标签实例需要在核对应用归属、工作区和持久卷后，由操作人安排受控重建。

首次快照的中断场景曾观察到 `provider_operation_failed`，最终快照则得到 `needs_attention` 与 `agent_execution_interrupted`。这两次观察分别属于其冻结源码，前一次失败码的具体机制仍缺少独立归因证据。最终中断结论以下文最新运行记录为准。

### 历史 Agent 单次运行退出缺陷，现已修复

隔离基线中，Agent 已完成 enrollment、mTLS 连接和健康就绪，`compose run ... --once` 仍持续运行。`RunCLI` 调用 `runOnce` 成功后继续执行函数末尾的 `<-sigCtx.Done()`，使“运行一次并退出”的 CLI 契约无法完成。原 conformance runner 的首次和重启阶段均依赖这个退出行为，因而可能停在已健康的 Agent 进程。该问题属于基线实现缺陷。本轮隔离 harness 改为启动普通 Agent 服务、等待 Backend-owned bootstrap 证据、明确停止并重启，再验证相同身份与 receipt 推进。该真实重启流程已在基线证明同一身份与递增 receipt。后续修复移除了生命周期返回后的额外信号等待，并以子进程回归覆盖成功退出、配置失败、常驻模式信号退出及 wait-for-delivery 的既有优先级。真实一次性 Agent 在 1.79 秒内退出 0，身份保持一致且 ledger receipt 推进。原 runner 同步改为读取 driver 保存的完整重启基线，保留两次实际单次执行。修复跟踪见 [单次退出问题 #346](https://github.com/GeWuYou/Graft/issues/346)、[Agent 入口](../server/agents/docker-runtime-agent/internal/main.go)、[原验证阶段](../tests/conformance/docker-runtime-agent/run.sh)

## 历史修复与本轮核查点

这些提交涉及不同失败边界，后续核查需要分别复现其触发条件。历史提交的存在证明修复意图与代码变化，当前用户操作的可靠性由本轮运行证据判断。

| 时间与提交 | 已处理的边界 | 本轮核查点 |
| --- | --- | --- |
| 2026-06-23，`f5aa0f16f` | server 按挂载 socket 的 GID 建组后降权 | Agent 的独立 socket GID、读取与执行权限是否一致 |
| 2026-07-18，`67c6722e3` | root-init 多行 command 参数传递，CI 可写工作区与失败日志 | 全新部署目录、目录 ownership 和实际相对挂载 |
| 2026-08-10，`5f7f26d90` | pending enrollment 权威与本地 Vault 材料 UID | 正式 Compose 的首次 delivery 是否有完整可执行入口 |
| 2026-08-11，`4c4c71662` | CA fingerprint 变化后的 generation 轮换与重新登记 | Vault 重建、已有 Agent state 和重新交付之间的恢复顺序 |
| 2026-08-11，`93fddf582` | 本地 Gateway 独立 dotenv 与 Agent 配置落点 | 常规开发、Agent 调试和正式部署三种配置的实际选择 |
| 2026-08-21，`fb0d29a14c` | 将 Application/Container 写操作迁到 Agent，并增加完整绑定身份比较 | 证书中间件的部分身份与执行消费者的新要求之间缺少受信解析和端到端正向验证 |
| 2026-08-31，`7b1551f5d` | 回退不兼容 Docker SDK 依赖组合 | 新版 SDK 成组迁移后的编译与真实 Compose 操作 |

提交记录可分别从 [socket 权限修复](https://github.com/GeWuYou/Graft/commit/f5aa0f16f70cfa43e534a15570a891eaa6146240)、[工作区启动修复](https://github.com/GeWuYou/Graft/commit/67c6722e3140223fbc1bd920c43a199e08d1aaf3)、[交付归属修复](https://github.com/GeWuYou/Graft/commit/5f7f26d90e7eac8df6aff93ae84c4e0076e38c06)、[信任轮换修复](https://github.com/GeWuYou/Graft/commit/4c4c71662c20d5418094623e68e917fcdf224bb1) 和 [开发配置修复](https://github.com/GeWuYou/Graft/commit/93fddf582b8853bd5b47fe05b5fe1724515cfd57) 查看。

## 依赖升级阶段的隔离运行验证

运行环境采用 task-owned Compose project、独立 PostgreSQL/Redis/Vault 数据和证书、Docker 29.9.3 的专用 daemon。只有私有 socket 卷交付给 Graft 组件，应用工作区在 daemon、Backend 和 Agent 内使用同一绝对路径。原本的本地发布验证脚本会共享宿主 socket，本轮增加了渲染后拓扑校验与十三个针对性回归检查。[原本地验证脚本](../scripts/local_graft_validation.py)、[本轮验证入口](../scripts/validate_docker_isolated.py)、[隔离边界回归](../scripts/test_validate_docker_isolated.py)

基线与升级后的镜像需要记录准确源码 SHA 和不可变镜像 ID。真实 API 探针覆盖镜像拉取、容器启停与重启、日志、WebSocket Shell 命令、Compose 创建部署和相对 bind mount、目录导入，以及材料缺失、权限拒绝、Agent 离线和执行中断。

本轮已观察基线 Agent 的 enrollment、mTLS、相同身份重启与 ledger receipt 推进，容器清单、日志和真实 WebSocket Shell 命令已有通过记录。镜像拉取和生命周期的首次请求因探针缺少必需的 Idempotency-Key 返回 400，该项归入探针修复。补齐请求契约后，基线镜像拉取获 202，但 90 秒后 Task 仍停在 ready。真实 mTLS 无任务 claim 的 400 及身份解析源码构成上文 P0 发现的证据。基线和升级版本的同场景结果见下表。两版的真实 mTLS 无任务 claim 均返回 400，表现与同一身份解析缺陷一致。该阶段通过 Python 编译、十三个隔离、生命周期和 API 契约回归、实际 Compose 渲染与 host socket 排除检查。以上结果证明测试边界和脚本结构，真实业务场景的通过情况由 `api-evidence.json` 和 `trust-evidence.json` 的实际记录决定。

| 场景 | 基线观察 | 升级后同场景 |
| --- | --- | --- |
| 默认样例配置 | config validate 退出 0，覆盖语法与解析 | 相同校验退出 0 |
| enrollment、mTLS 与重启 | 身份保持一致，receipt 推进 | 身份保持一致，receipt 推进 |
| 镜像拉取、容器启停重启 | HTTP 202，Task 90 秒后仍 ready，Stage pending，无 lease | 相同操作接受后超时，既有身份 gate 阻断 |
| 容器清单、日志与 Shell | HTTP 读取与真实 WebSocket 命令输出通过 | 相同读取与真实 Shell 命令通过 |
| managed create 与 up | 创建返回 201，up 接受后停在 ready | 创建返回 201，up 接受后超时，容器未出现 |
| 相对 bind mount 与执行中断 | 身份 gate 阻断执行，实际 provider 副作用待验证 | 同一 gate 阻断，provider 副作用待验证 |
| inspect 与 import | inspect 成功，import 因 SQL 绑定顺序返回 500 | 同样在 import 持久化返回 500 |
| 材料缺失 | 删除 Compose 文件后请求返回 409，前置冲突阻断 | 相同请求返回 409 |
| socket 权限 | UID 10001 新连接被拒绝，既有 API 连接仍可读取 | 相同新连接拒绝，既有 API 仍为 200 |
| Agent 离线 | 操作返回 500，容器 StartedAt 保持一致 | 相同操作返回 500，StartedAt 保持一致 |

fixture 的 Agent 依照既有 bootstrap-secret 契约以 root 运行。权限探针使用 Backend 的实际 UID 10001，新的 curl Unix socket 连接在 chmod 前返回 OK，chmod 000 后退出码为 7。同一时刻已建立连接的 API 读取仍返回 200，记录了既有连接的继续可用性。探针最后将 socket 权限恢复为 660。Agent 的非 root/GID 拒绝应单独证明。Backend 自身运行在外层 fixture，而其 Docker 观察对象位于私有 daemon，平台自更新需要另建整套部署位于私有引擎内的验收。

## 既有断点修复复验

2026-11-15 使用最终修复源码 `fbe7b39e225d45fb834d81b4b738602ebc604355` 的冻结 Git 导出构建 Backend、Agent 与 conformance driver。三份 CGO-disabled 二进制分别进入原有运行镜像阶段，镜像记录源码标签、不可变镜像 ID 与二进制 SHA-256。沿用专用 Docker daemon、独立 PostgreSQL/Redis/Vault 和相同工作区挂载，通过真实公共 API 发起操作，并核对 Task 终态和引擎状态。enrollment 与普通服务重启保留同一身份，ledger receipt 从 1 推进到 2。

| 场景 | 修复后的实际结果 |
| --- | --- |
| Agent 单次退出 | `--once` 在 1.79 秒内退出 0，身份保持一致，receipt 继续推进 |
| 镜像拉取 | HTTP 202，Task `success` |
| 容器停止、启动和重启 | 均返回 HTTP 202，三个 Task 均 `success`，引擎 running 状态分别为 false、true、true，重启前后 StartedAt 实际变化 |
| 清单、日志与 Shell | 清单和日志 HTTP 200，真实 WebSocket Shell 命令输出通过 |
| Compose 创建、部署与停止 | managed create HTTP 201，up 与 stop Task 均 `success`，up 后 Running=true、stop 后 Running=false，全部探针结束后的独立 inspect 仍为 false |
| 相对路径挂载 | 应用容器读取 `./probe.txt` 挂载内容，与写入工作区的内容一致 |
| Compose 导入 | inspect 成功，import HTTP 200，API 和 PostgreSQL 读回的 display name、runtime target、Compose name/source 四字段一致 |
| 重复导入 | HTTP 409，持久化字段保持一致。ON CONFLICT 更新分支由直接仓储回归覆盖 |
| Compose 材料缺失 | HTTP 202 后 Task `failed`，稳定失败码为 `invalid_execution_intent`。历史基线的前置 409 结果单独保留于上表 |
| 执行中断 | 观察到提供方容器后杀死 Agent，再恢复 Agent。Task 最终 `needs_attention`，失败码 `agent_execution_interrupted`，容器 ID 和 StartedAt 保持一致，已产生的容器保留 |
| Backend socket 权限 | UID 10001 新连接在 chmod 前成功，权限收紧后 curl 退出 7，既有 API 连接仍返回 200 |
| Agent 离线 | 操作先返回 HTTP 202，探针在有界等待后主动取消，Task 最终 `cancelled`，容器 StartedAt 保持一致 |

执行中断场景记录为 `observed`，当前证据覆盖需人工处理的终态、保留副作用与恢复后的容器身份。完整自动恢复、重新执行和副作用清理需要额外验收。离线场景包含探针主动取消，其结果支持本次离线请求无重启副作用及取消收敛。材料缺失场景记录了接受与异步失败的实际边界。

最终独立引擎核对还确认 Compose 资源的 Project、Service、Oneoff 归属标签与预期一致，八个外部执行租约均已进入 `settled`。脱敏事实保存在任务本地的 `api-evidence.json`、`trust-evidence.json` 与 `engine-final-evidence.json`，构建身份保存在 `images.json`。这些结果覆盖本表明确列出的操作，后续产品路线仍属于评估提案。当前修复入口与验证工具见 [执行身份解析](../server/internal/httpx/agent_execution_identity.go)、[导入存储](../server/modules/project/store/sql_repository.go)、[Agent CLI](../server/agents/docker-runtime-agent/internal/main.go) 和 [隔离验收入口](../scripts/validate_docker_isolated.py)。

## PR 审查后的验证工具修复

PR #344 审查发现，隔离工具最初的 `stop` 会停止 Vault dev。Vault dev 的 PKI 和 AppRole 保存于进程内存，具名卷上的 `.initialized` 与旧证书继续存在，再次启动的开发实例与保留材料脱节。当前 Vault authority 的完整性需要实际登录和 PKI 端点验证。[隔离工具](../scripts/validate_docker_isolated.py)、[Vault fixture 初始化](../tests/conformance/docker-runtime-agent/vault-init.sh)

修复后的逻辑停止保留运行中的任务专用 Vault，停止其余声明的 fixture 服务。`up` 在迁移和 Backend 启动前使用已交付 AppRole 发起真实登录，读取当前 PKI CA 并与交付 CA 比对。`destroy` 统一删除全部注册服务与卷。帮助信息与 fixture 指南明确了保留依赖的生命周期。[停止与销毁指南](../tests/conformance/docker-runtime-agent/README.md)

真实复验使用源码为 `c274ac9236a240cf209638434fd370692e204a2d` 的既有 Graft 镜像与修订后的工具。初次启动证明 AppRole 登录与 CA 一致，逻辑停止后只保留 Vault，恢复后重新登录通过且 CA 指纹一致。随后明确停止 Vault 造成状态丢失，工具在 Backend、Agent 启动前拒绝旧材料。该复验项目随后已经销毁。工具 SHA-256 为 `2f9194dcee7126ceb7d25103c74a160f2988a3e7a1fb655d56bb08c2c085de4b`，安全汇总记录在版本清单的 `stop_resume_authority_verification` 字段中。此次复验覆盖隔离工具的停止、恢复与 authority 检查。

## 三条路线的取舍

| 路线 | 适用需求 | 收益 | 主要成本与验收要求 | 判断 |
| --- | --- | --- | --- | --- |
| 继续完善当前 Agent | 确有远程主机、信任隔离与离线执行需求 | 复用租约、fence、身份和 Task 中断保护 | 补齐读取和交互传输、自动交付、证书轮换、诊断与远程工作区，持续维护多进程协议 | 当前单机操作复验通过，继续依据诊断与恢复成本判断远程扩展收益 |
| 本机直接执行 | 当前个人与小团队的单机 Docker/Compose 运维 | 减少首次启动步骤和本机操作跨进程边界，读写使用同一目标 | 明确后端 socket 权限，保留受控异步任务、取消与审计，迁移未完成任务，重做可恢复性验证 | 在当前可执行基线上进行有界对比，再决定是否迁移 |
| 缩减 Docker 管理职责 | 核心用户实际只需要后台框架或只读看板 | 可降低持续运行时维护投入 | 失去部署和故障处理闭环，需要重新证明后台平台的需求和差异化 | 与本次选择的运维用户目标匹配度较低 |

当前 Agent 已通过本轮列出的真实单机操作。本机直连方案可在现有容器/应用服务的执行边界做最小试验，比较安装步骤、首次成功率、错误可定位程度和中断恢复行为。Agent 协议及现有运行记录需要明确的数据与任务迁移策略。产品方案收敛后再修改当前 ADR 与模块边界。

## 分阶段收拢建议

| 阶段 | 用户收益 | 保留与调整范围 | 数据与契约处理 | 完成依据 |
| --- | --- | --- | --- | --- |
| 第一阶段，修复执行与单机闭环 | 写操作开始执行，全新安装后能运行一个真实 Compose 应用 | 已修复受信身份解析，保留应用、容器、日志、终端与审计，继续简化首次可用步骤 | 记录部署配置迁移、旧 Agent generation 和未完成 Task 的处理规则 | 新目录安装、已有应用导入、相对挂载与启停全部通过，错误能定位到失败边界 |
| 第二阶段，可靠更新 | 更新失败可定位、可恢复，用户能判断是否完成 | 优先手动 redeploy、镜像变更预览、平台备份与恢复演练 | 保留历史 Task/审计，明确备份包含的数据、恢复版本和回滚限制 | 拉取失败、启动失败、服务中断与恢复演练具有可复现证据 |
| 第三阶段，扩展按需 | 有实际需求时扩展远程与构建 | 将远程 Agent、多平台构建、GitOps、外部通知、镜像扫描和集群提供方分别延后评估 | 每项扩展声明 owner、稳定契约、迁移和停止维护办法 | 由真实任务证明收益，并给出维护预算与完整用户旅程 |

建议退出面向多个未来提供方的预先扩展承诺、脱离当前运维任务的独立平台展示、缺少故障证据的重复补丁循环。保留已有历史数据与已验证能力，对具体模块的删除或下线另行列出消费者、迁移与回滚证据。

## 修复后的后续工作

1. 执行身份、导入绑定和单次退出三项明确断点，以及复验发现的 Compose 归属标签缺口均已修复，真实单机操作复验结果见上文。补充无工作 claim 的独立 HTTP 204 证据，以及绑定轮换与领取事务之间的并发边界验收。
2. 将材料缺失、执行失败和 Agent 离线的稳定阶段码与 Task 关联信息投影到用户可见诊断，减少只显示通用失败码的排障成本。
3. 对执行中断后的容器保留、Task `needs_attention`、继续执行与人工处置建立明确恢复规则，补齐自动恢复验收。平台备份恢复和全新默认安装仍需要独立运行证据。
4. 在已能执行的基线上对比本机执行简化方案的安装步骤、故障定位和恢复成本，再决定产品边界调整。

## 验证范围与已完成检查

服务端切片与后续可空枚举修复分别通过仓库 backend 验证入口。Web 切片在 Node 26.13.1、Bun 1.4.4 下完成 `bun run check`，312 个测试文件、2175 个测试通过，生产构建零 warning，frozen lockfile 安装通过。隔离 harness 的十三个针对性回归、Compose 渲染和 AI 治理检查通过。后续断点修复在 Go 1.27.4 下再次通过完整 `go run ./cmd/graft validate backend`，二十三个隔离工具回归通过。真实 Docker 写路径已通过上文列出的成功场景，执行中断的完整自动恢复、默认安装闭环和其他未列出的能力仍待验收。此次修复覆盖服务端身份、存储、Agent 生命周期与隔离验证工具，沿用现有数据模型和公共 API。远端 CI、合并和原依赖 PR 的实际处理结果由负责会话在交付回执中记录。

## 依赖升级与交付记录

完整版本记录包含服务端模块变更、前端声明版本、Actions 的旧新 SHA、基础镜像与兼容性例外，可查阅 [依赖版本清单](dependency-upgrade-2026-11-15.json)。版本清单已与服务端和 Web 完成态 manifest 复核。当前已确定 Go 1.27.4、Compose 5.7.1、Node 26.13.1、Bun 1.4.4、golangci-lint 2.15.0、Alpine 3.24.4。基础服务保留已有大版本，根部署 PostgreSQL 16.16、fixture PostgreSQL 17.12、Redis 7.4.12、Vault 1.21.4。

服务端同时将 98 个 OpenAPI 源文件中的 266 处可空声明迁到 OpenAPI 3.1 的类型联合或对应组合表达，补齐新版 kin-openapi 的校验要求，并同步 bundle、服务端绑定、嵌入式契约与前端派生消费面。

PR 审查还发现，初轮 nullable 结构比对遗漏了 enum 对 null 的额外约束。针对七个源文件中的十个可空枚举字段，行为回归先复现新版校验拒绝 null，再将表示修正为受约束字符串枚举与 null 的 `anyOf`。全部十个字段的 null、已声明字符串及非法字符串/布尔值均有校验，修正后的 98 文件基线结构比对与服务端生成、backend 验证和 OpenAPI freshness 检查通过。修复提交为 `3944f0e5a4b86ac76729cd292251fd632c65e108`，具体字段及对照证据见版本清单的 `nullable_enum_repair`。

前端升级已经识别 TDesign TreeNodeValue 的 bigint 扩展、DropdownItem 的事件参数变化、markdown-it 15 原生 Token 类型、vue-i18n 11 消息类型变化及 Volar URL 字面量解析问题。相关修复与验证由 Web 切片记录。TypeScript 主编译器保留 6.0.3，同时保留独立 7.0.2 别名检查，原因是现行 vue-tsc 和 ESLint parser 仍依赖 JavaScript compiler API。jscpd 保留 4.3.0。用相同的 715 个源文件和相同阈值比较后，5.4.3 的分词与跨格式分析报告了 25 处新克隆，4.3.0 报告零克隆。两个版本都支持现有忽略标记，差异需要独立审查扫描基线并完成迁移。`@babel/runtime-corejs3` 保留 Babel 7 兼容线并提升到 7.29.14。tvision-color 1.6.0 产物导入的 Babel 7 路径在 Babel 8.1.2 中已移除，完整构建已复现失败。移除 override 会恢复上游固定的 7.18.9，因此继续使用已提升补丁版本的原有覆盖机制，待上游更新运行时或重建产物后清理。服务端保留上游稳定 Compose/BuildKit/containerd 依赖图要求的 `containerd/platforms v1.0.0-rc.5` 和既有 gorilla/websocket 伪版本，具体 owner、版本与原因记录在清单中。

环境 inventory 通过仓库 collector 和 generator 刷新，记录当前全局 Go 1.26.11、Node 24.19.0 和 Bun 1.3.14。最新工具链的执行使用 task-local 安装和 CI 固定版本，其实际验证结果由对应切片保存。

积压依赖 PR 的替代范围为 #314、#319、#320、#322、#325、#332、#334、#335、#336、#337、#338。对应更新的集成、远端验证、合并与 PR 处理由负责会话跟进，并在交付回执记录实际结果。维护交付跟踪见 [问题 #341](https://github.com/GeWuYou/Graft/issues/341)。
