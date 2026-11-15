# Graft 定位与 Docker 可靠性评估

评估日期为 2026-11-15。Graft 的调查基线为 `3989727b06bf446a03bcabceba0246c43bf3f181`，升级运行镜像源码为 `c274ac9236a240cf209638434fd370692e204a2d`，Arcane 固定比较版本为 `v3.0.0`。目标用户为个人与小团队的 Docker/Compose 运维人员。本报告提出后续产品取舍，依赖升级与验证属于本轮交付。

## 当前判断

Graft 已拥有应用工作区、Compose 生命周期、Docker 资源清单、日志、终端、权限、任务与审计等实现。本轮在隔离基线复现了 Agent 健康、读取和 Shell 正常、写操作 Task 停在 ready 的问题。真实 mTLS 请求与源码共同指向执行入口的身份字段不匹配。应先修复这一处受信身份解析边界并重跑相同操作，再根据安装成本、诊断和恢复证据评估功能收拢。现有能力覆盖面可以支撑一个单机 Docker/Compose 运维工具。实现证据见 [应用模块](../server/modules/project)、[容器模块](../server/modules/container) 和 [任务模块](../server/modules/task)。

建议下一阶段围绕“安装并连接本机 Docker、导入或创建 Compose 应用、部署和更新、查看日志、定位失败”五个用户任务安排工作。将本机运维作为默认交付目标，远程控制、构建平台与平台自更新分别设置独立的可用性证据和维护预算。

Arcane 的默认安装将管理程序、数据库文件和本机 Docker 连接放在一套配置内，远程主机再引入 Agent。Graft 的默认 Compose 同时依赖 PostgreSQL、Redis、server、web 和 Runtime Agent，Agent 激活还涉及外部 Vault PKI、双 TLS listener、delivery 与配置材料。这个差异直接影响首次操作所需的步骤与可失败边界。[Arcane 安装文档](https://getarcane.app/docs/get-started/installation)、[Graft Compose](../compose.yml)、[Agent 部署说明](../server/agents/docker-runtime-agent/README.md)

## 功能对照

表中“源码实现”表示存在可追踪的生产入口。“部分”表示能力边界或交付范围仍有限。“待运行验证”表示本轮尚未建立该功能的真实环境结果。Arcane 的对应能力以固定版本源码与官方文档共同核对，官方文档后续可能更新。

| 用户能力 | Arcane v3.0.0 对应能力 | Graft 当前实现与边界 | 本轮运行证据 | 建议 |
| --- | --- | --- | --- | --- |
| 首次安装与本机连接 | 默认单管理容器，本机 socket，内置 SQLite，可选 PostgreSQL | Compose 提供 PG/Redis/server/web/Agent，初始材料准备分散 | 基线默认样例环境 config validate 通过，完整交付依赖 fixture 准备 | 优先简化首次可用路径与配置诊断 |
| Compose 创建、编辑与部署 | 项目文件夹、YAML/.env 编辑、Up/Down/Redeploy | 应用 ID、工作区、配置快照、生命周期配置、Task/Agent 执行均有源码实现 | 两版 managed create 201，up Task 接受后超时 | 保留，优先修复执行入口并验证真实部署 |
| 现有 Compose 导入 | 从项目目录发现与管理已有文件 | 目录来源、inspect、冲突检查、import、运行时候选有实现 | 两版 inspect 成功，import 持久化返回 500，已定位 SQL 参数顺序 | 保留，修复有序参数绑定并验证真实 PostgreSQL |
| 模板 | 模板及模板注册源 | 版本化模板、发布、撤回、克隆与创建入口有实现 | 源码实现，模板全流程待验证 | 保留最常用模板，注册源扩展延后 |
| 单容器管理 | 清单、详情与生命周期 | 清单/详情在后端读 Docker，启停删除通过 Task/Agent | 两版清单和日志通过，写 Task 接受后超时 | 保留，优先修复受信身份解析 |
| 镜像、网络、卷 | 资源管理、清理、拉取 | 资源清单、详情及受控写操作有实现 | 两版拉取均接受为 Task 后超时，其余资源管理待验证 | 保留高频操作，危险操作维持确认和审计 |
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

### P0，执行入口要求的身份字段没有从受信绑定解析

隔离基线的镜像拉取请求返回 HTTP 202，Task 保持 `ready`、Stage 保持 `pending`，外部执行 lease 为空。Agent 已完成 mTLS、健康检查与重启后的 receipt 推进，其 binding 包含四项 capability 和 `docker/v1`。使用同一真实证书发起无任务的 `oci-build` claim，仍返回 HTTP 400 `COMMON_INVALID_ARGUMENT`。这个最小探针避开了认领写操作的副作用。

mTLS 中间件从 URI SAN 解析 TargetID、AgentID，并添加证书序列号与公钥指纹。它写入请求上下文时，IdentityID 仍为空，Generation 仍为零。执行入口随后用 `agentIdentityMatchesBinding` 要求这些值与已激活绑定的 IdentityID、Generation 完全相等。正常激活绑定具有非空 IdentityID 和正 generation，因而 claim 被稳定拒绝，Task 从未进入 provider 执行。Ledger 路径通过 Runtime Target authority 使用证书证据解析已保存身份，可以完成健康与 receipt 流程。这解释了同一个 Agent 健康与任务停滞同时出现的机制。[证书身份提取](../server/internal/httpx/agent_mtls.go)、[执行入口比较](../server/internal/httpx/agent_execution_routes.go)、[Ledger 入口](../server/internal/httpx/agent_ledger_routes.go)

Git blame 显示，严格的完整身份比较在 `fb0d29a14c` 的 2026-08-21 Application/Container 执行迁移中引入，证书身份生产者从 2026-08-10 的 `8ac11bc5f2` 起继续只提供证书可验证字段。执行路由测试直接向 Gin 注入身份，mTLS 测试则明确验证 Generation 为零，缺少把真实中间件、当前 binding 与 claim handler 串起来的正向回归。[执行迁移提交](https://github.com/GeWuYou/Graft/commit/fb0d29a14c)、[执行路由测试](../server/internal/httpx/agent_execution_routes_test.go)、[mTLS 测试](../server/internal/httpx/agent_mtls_test.go)

建议的最小修复由 Runtime Target 身份与 HTTP 执行边界共同拥有。依据已验证证书的 TargetID、AgentID、序列号与公钥指纹，读取并验证 active binding，再从该受信记录取得 IdentityID 与 Generation，统一供 claim、renew、material、result 与 receipt 授权使用。请求 header 和 payload 继续只提供操作输入。验收应覆盖真实中间件下有效证书领取成功、无工作返回 204、旧证书或 retired generation 被拒绝，并使用本轮相同镜像拉取与容器生命周期重现观察修复效果。

本轮保留该基线缺陷作为评估发现。它为下一步提供了可定位的小范围修复机制，后续架构取舍应在修复复验后继续评估。修复与真实场景验收跟踪见 [执行身份问题 #342](https://github.com/GeWuYou/Graft/issues/342)。

### Docker 依赖组合存在真实编译约束

提交 `7b1551f5d7280b48b9f3f30bae6a86a819fa15b6` 曾将 Docker CLI 29.2.0、compose-go 2.14.0 和 go-archive 0.3.3 回退到旧组合。积压 PR #336 再次单独提升 archive 等包后，旧 CLI 的构建代码引用了已经消失的 `archive.Gzip`。这个问题能由编译日志直接证实，后续应将 Compose、CLI、Moby、BuildKit 和 archive 按上游发布依赖图迁移并验证。Compose v5.7.1 的发布源码已经使用 CLI 29.9.3 与新的 Moby 模块。[历史回退提交](https://github.com/GeWuYou/Graft/commit/7b1551f5d7280b48b9f3f30bae6a86a819fa15b6)、[PR #336](https://github.com/GeWuYou/Graft/pull/336)、[Compose v5.7.1 依赖](https://github.com/docker/compose/blob/v5.7.1/go.mod)

### “读取在线”与“能够执行”由不同路径证明

后端通过本地 Unix socket 获取容器清单、日志和运行目标事实。写操作需要 Task 外部执行租约、有效 Agent generation、capability、交付与 mTLS，再由 Agent 的 Moby/Compose 提供方执行。因而清单可读只证明后端的 Docker 连接可用。Agent 未交付、离线、配置路径错误或能力未激活时，操作仍可能在另一个边界失败。根 Compose 同时保留后端与 Agent 的 socket，迁移还留有显式的读取路径。这些均为当前源码事实，具体现场故障需要绑定其请求与 Task 结果。[读取入口](../server/modules/container/docker_runtime.go)、[容器 Task](../server/modules/container/task_lifecycle.go)、[Agent 执行边界](../ai-plan/design/decisions/ADR-026-docker-runtime-agent-execution-boundary.md)

### 快速启动的默认配置缺少 Agent 激活闭环

`compose.env.example` 默认将 bootstrap TLS、Agent TLS 和 Credential Vault 设为 false，根 Compose 却默认启动 Agent。Agent README 要求首次运行前准备 Vault PKI、证书和 delivery 产物。本轮对默认样例环境补入合成密码和 JWT 后运行配置校验，退出码为零。该结果覆盖配置语法与解析，Agent 配置和信任材料的来源仍需额外完成。后续优先把首次安装的可执行步骤、就绪判断和失败提示收敛到同一入口，并用全新数据目录验证。[默认环境模板](../compose.env.example)、[快速开始](../README.zh-CN.md)、[Agent README](../server/agents/docker-runtime-agent/README.md)

### Claim 拒绝没有进入执行就绪与用户诊断

`pollExecutionCapabilities` 遇到普通 claim 错误后继续循环，`runExecutionCycle` 在身份准备、mTLS client 或 journal 恢复失败时也直接返回当前状态。Agent 的 healthcheck 校验本地证书、generation、过期时间和信任束，缺少 claim 接受状态。因此已经持有有效证书的 Agent 可以保持容器 healthy，同时持续收到 claim 400，公共操作 Task 长时间停留 ready。该状态在本轮真实基线中已经观察到。[执行循环与 healthcheck](../server/agents/docker-runtime-agent/internal/execution.go)

建议在 P0 修复验收中增加执行 admission 的有界诊断。以稳定阶段码、失败类别和关联标识投影当前可执行性，把正常无工作 claim 的 204 作为执行通道验收证据之一，并区分无工作、身份拒绝和传输失败。相应状态需要进入 Runtime Target capability 与操作结果的用户反馈。诊断输出应有去重和频率上限，并校验凭据与 provider 详情脱敏。

### 提供方诊断信息在返回路径上被压缩

Agent 的 `loadComposeProject` 将 SDK stdout/stderr 设为 `io.Discard`，多数提供方错误被映射到 `provider_operation_failed`。ADR-026 限制原始 provider stderr 进入 Task 日志与 receipt，目的是保护凭据、路径与边界。当前设计同时压缩了运维人员区分配置解析、拉取、挂载和启动错误所需的信息。建议补充有界的结构化失败阶段、稳定诊断码和受控脱敏详情，并提供与 Task 关联的诊断入口。该建议需要独立契约设计与隐私测试。[提供方实现](../server/agents/docker-runtime-agent/internal/provider.go)、[外部执行决策](../ai-plan/design/decisions/ADR-026-docker-runtime-agent-execution-boundary.md)

### 既有 fixture 的通过范围有限

原 conformance runner 验证 enrollment、证书签发、mTLS、ledger 和重启身份。它的八个阶段没有从公共容器或应用 API 触发 Docker 生命周期。当前 driver 还会调用本地 discovery，原 Compose 的 driver 服务缺少 socket 挂载。本轮隔离 overlay 为 Backend、Agent 和 driver 明确配置相同的私有引擎，并新增实际 API 操作与引擎状态核对。[原 runner](../tests/conformance/docker-runtime-agent/run.sh)、[driver 的目标解析](../server/modules/runtime-target/conformance_driver.go)、[隔离 overlay](../tests/conformance/docker-runtime-agent/compose.isolated.yml)

### P1，Compose 导入的 SQL 列与参数顺序错位

基线的 managed create 返回 HTTP 201，随后针对另一个独立工作区的 inspect 成功，import 返回 HTTP 500 `COMMON_INTERNAL_ERROR`。关联请求日志给出的具体错误为 `upsert application` 的第六个绑定参数无法编码为 PostgreSQL varchar，传入值是 Runtime Target 数值 ID。

`upsertApplication` 的前八个参数按 application ID、adapter、application name、workspace、display name、runtime target、Compose name、Compose name source 排列。`applicationsUpsertSQL` 的对应列却把 Compose name 和 source 放在 display name 与 runtime target 之前。因此数值 runtime target 被绑定到 `compose_project_name_source`。独立的 `createApplication` 路径按 SQL 列顺序传参，可以成功创建。Git blame 将这处 SQL 与既有参数顺序的分离定位到 2026-07-17 的 `f5fea4eaf2`。现有回归只统计占位符和列的数量。[SQL 参数与列定义](../server/modules/project/store/sql_repository.go)、[占位符回归](../server/modules/project/store/sql_repository_review_regressions_test.go)

建议先对齐 upsert 的有序字段与参数，并以真实 PostgreSQL 完成目录 inspect、import 和持久化读回，校验 runtime target、Compose name/source、display name 全部保持正确。该缺陷发生在导入持久化阶段，独立于 Agent 的执行身份问题。本轮作为已复现的既有问题保留，后续修复跟踪见 [导入问题 #343](https://github.com/GeWuYou/Graft/issues/343)。

### Agent 单次运行模式在成功后仍等待信号

隔离基线中，Agent 已完成 enrollment、mTLS 连接和健康就绪，`compose run ... --once` 仍持续运行。`RunCLI` 调用 `runOnce` 成功后继续执行函数末尾的 `<-sigCtx.Done()`，使“运行一次并退出”的 CLI 契约无法完成。原 conformance runner 的首次和重启阶段均依赖这个退出行为，因而可能停在已健康的 Agent 进程。该问题属于基线实现缺陷。本轮隔离 harness 改为启动普通 Agent 服务、等待 Backend-owned bootstrap 证据、明确停止并重启，再验证相同身份与 receipt 推进。该真实重启流程已在基线证明同一身份与递增 receipt。CLI 本身的修复建议为成功单次执行后返回，并补进程退出回归。[Agent 入口](../server/agents/docker-runtime-agent/internal/main.go)、[原验证阶段](../tests/conformance/docker-runtime-agent/run.sh)

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

## 隔离运行验证

运行环境采用 task-owned Compose project、独立 PostgreSQL/Redis/Vault 数据和证书、Docker 29.9.3 的专用 daemon。只有私有 socket 卷交付给 Graft 组件，应用工作区在 daemon、Backend 和 Agent 内使用同一绝对路径。原本的本地发布验证脚本会共享宿主 socket，本轮增加了渲染后拓扑校验与十个针对性回归检查。[原本地验证脚本](../scripts/local_graft_validation.py)、[本轮验证入口](../scripts/validate_docker_isolated.py)、[隔离边界回归](../scripts/test_validate_docker_isolated.py)

基线与升级后的镜像需要记录准确源码 SHA 和不可变镜像 ID。真实 API 探针覆盖镜像拉取、容器启停与重启、日志、WebSocket Shell 命令、Compose 创建部署和相对 bind mount、目录导入，以及材料缺失、权限拒绝、Agent 离线和执行中断。

本轮已观察基线 Agent 的 enrollment、mTLS、相同身份重启与 ledger receipt 推进，容器清单、日志和真实 WebSocket Shell 命令已有通过记录。镜像拉取和生命周期的首次请求因探针缺少必需的 Idempotency-Key 返回 400，该项归入探针修复。补齐请求契约后，基线镜像拉取获 202，但 90 秒后 Task 仍停在 ready。真实 mTLS 无任务 claim 的 400 及身份解析源码构成上文 P0 发现的证据。基线和升级版本的同场景结果见下表。两版的真实 mTLS 无任务 claim 均返回 400，表现与同一身份解析缺陷一致。当前已经通过 Python 编译、十个隔离、生命周期和 API 契约回归、实际 Compose 渲染与 host socket 排除检查。以上结果证明测试边界和脚本结构，真实业务场景的通过情况由 `api-evidence.json` 和 `trust-evidence.json` 的实际记录决定。

| 场景 | 基线观察 | 升级后同场景 |
| --- | --- | --- |
| 默认样例配置 | config validate 退出 0，覆盖语法与解析 | 相同校验退出 0 |
| enrollment、mTLS 与重启 | 身份保持一致，receipt 推进 | 身份保持一致，receipt 推进 |
| 镜像拉取、容器启停重启 | HTTP 202，Task 90 秒后仍 ready，Stage pending，无 lease | 相同操作接受后超时，既有身份 gate 待修复 |
| 容器清单、日志与 Shell | HTTP 读取与真实 WebSocket 命令输出通过 | 相同读取与真实 Shell 命令通过 |
| managed create 与 up | 创建返回 201，up 接受后停在 ready | 创建返回 201，up 接受后超时，容器未出现 |
| 相对 bind mount 与执行中断 | 身份 gate 阻断执行，实际 provider 副作用待验证 | 同一 gate 阻断，provider 副作用待验证 |
| inspect 与 import | inspect 成功，import 因 SQL 绑定顺序返回 500 | 同样在 import 持久化返回 500 |
| 材料缺失 | 删除 Compose 文件后请求返回 409，前置冲突阻断 | 相同请求返回 409 |
| socket 权限 | UID 10001 新连接被拒绝，既有 API 连接仍可读取 | 相同新连接拒绝，既有 API 仍为 200 |
| Agent 离线 | 操作返回 500，容器 StartedAt 保持一致 | 相同操作返回 500，StartedAt 保持一致 |

fixture 的 Agent 依照既有 bootstrap-secret 契约以 root 运行。权限探针使用 Backend 的实际 UID 10001，新的 curl Unix socket 连接在 chmod 前返回 OK，chmod 000 后退出码为 7。同一时刻已建立连接的 API 读取仍返回 200，记录了既有连接的继续可用性。探针最后将 socket 权限恢复为 660。Agent 的非 root/GID 拒绝应单独证明。Backend 自身运行在外层 fixture，而其 Docker 观察对象位于私有 daemon，平台自更新需要另建整套部署位于私有引擎内的验收。

## 三条路线的取舍

| 路线 | 适用需求 | 收益 | 主要成本与验收要求 | 判断 |
| --- | --- | --- | --- | --- |
| 继续完善当前 Agent | 确有远程主机、信任隔离与离线执行需求 | 复用租约、fence、身份和 Task 中断保护 | 补齐读取和交互传输、自动交付、证书轮换、诊断与远程工作区，持续维护多进程协议 | 先完成 P0 身份修复与操作复验，再判断远程扩展收益 |
| 本机直接执行 | 当前个人与小团队的单机 Docker/Compose 运维 | 减少首次启动步骤和本机操作跨进程边界，读写使用同一目标 | 明确后端 socket 权限，保留受控异步任务、取消与审计，迁移未完成任务，重做可恢复性验证 | 在 P0 修复后的可用基线上进行有界对比，再决定是否迁移 |
| 缩减 Docker 管理职责 | 核心用户实际只需要后台框架或只读看板 | 可降低持续运行时维护投入 | 失去部署和故障处理闭环，需要重新证明后台平台的需求和差异化 | 与本次选择的运维用户目标匹配度较低 |

P0 修复应先证明当前 Agent 能完成真实操作。本机直连方案随后可在现有容器/应用服务的执行边界做最小试验，比较安装步骤、首次成功率、错误可定位程度和中断恢复行为。Agent 协议及现有运行记录需要明确的数据与任务迁移策略。产品方案收敛后再修改当前 ADR 与模块边界。

## 分阶段收拢建议

| 阶段 | 用户收益 | 保留与调整范围 | 数据与契约处理 | 完成依据 |
| --- | --- | --- | --- | --- |
| 第一阶段，修复执行与单机闭环 | 写操作开始执行，全新安装后能运行一个真实 Compose 应用 | 先修复受信身份解析，保留应用、容器、日志、终端与审计，再简化首次可用步骤 | 记录部署配置迁移、旧 Agent generation 和未完成 Task 的处理规则 | 新目录安装、已有应用导入、相对挂载与启停全部通过，错误能定位到失败边界 |
| 第二阶段，可靠更新 | 更新失败可定位、可恢复，用户能判断是否完成 | 优先手动 redeploy、镜像变更预览、平台备份与恢复演练 | 保留历史 Task/审计，明确备份包含的数据、恢复版本和回滚限制 | 拉取失败、启动失败、服务中断与恢复演练具有可复现证据 |
| 第三阶段，扩展按需 | 有实际需求时扩展远程与构建 | 将远程 Agent、多平台构建、GitOps、外部通知、镜像扫描和集群提供方分别延后评估 | 每项扩展声明 owner、稳定契约、迁移和停止维护办法 | 由真实任务证明收益，并给出维护预算与完整用户旅程 |

建议退出面向多个未来提供方的预先扩展承诺、脱离当前运维任务的独立平台展示、缺少故障证据的重复补丁循环。保留已有历史数据与已验证能力，对具体模块的删除或下线另行列出消费者、迁移与回滚证据。

## 已知问题的修复顺序

1. 先修复执行入口的受信身份解析，使用相同证书和 public API 重跑镜像拉取与容器生命周期，补齐真实中间件到 active binding 的正向与拒绝回归，同时验证可见的脱敏 admission 诊断与无工作 claim 204。
2. 对齐 import upsert 有序参数，并在真实 PostgreSQL 中验证持久化字段、冲突处理与已有应用重导入。
3. 修复 Agent `--once` 的正常退出，更新原 conformance runner 的重启验证基线读取，补进程级退出证据。
4. 在可执行操作的基线上验证 Compose 相对挂载、执行中断恢复和平台备份恢复，再比较本机执行简化方案的收益与维护成本。

## 验证范围与已完成检查

服务端切片已经通过仓库 backend 验证入口。Web 切片在 Node 26.13.1、Bun 1.4.4 下完成 `bun run check`，312 个测试文件、2175 个测试通过，生产构建零 warning，frozen lockfile 安装通过。隔离 harness 的十个针对性回归、Compose 渲染和 AI 治理检查通过。真实 Docker 写路径的运行结论以上文隔离证据为准，P0 身份拒绝仍限制了 provider 操作的验收。远端 CI、合并和原依赖 PR 的实际处理结果由负责会话在交付回执中记录。

## 依赖升级与交付记录

完整版本记录包含服务端模块变更、前端声明版本、Actions 的旧新 SHA、基础镜像与兼容性例外，可查阅 [依赖版本清单](dependency-upgrade-2026-11-15.json)。版本清单已与服务端和 Web 完成态 manifest 复核。当前已确定 Go 1.27.4、Compose 5.7.1、Node 26.13.1、Bun 1.4.4、golangci-lint 2.15.0、Alpine 3.24.4。基础服务保留已有大版本，根部署 PostgreSQL 16.16、fixture PostgreSQL 17.12、Redis 7.4.12、Vault 1.21.4。

服务端同时将 98 个 OpenAPI 源文件中的 266 处可空声明迁到 OpenAPI 3.1 的类型联合或对应组合表达，补齐新版 kin-openapi 的校验要求，并同步 bundle、服务端绑定、嵌入式契约与前端派生消费面。

前端升级已经识别 TDesign TreeNodeValue 的 bigint 扩展、DropdownItem 的事件参数变化、markdown-it 15 原生 Token 类型、vue-i18n 11 消息类型变化及 Volar URL 字面量解析问题。相关修复与验证由 Web 切片记录。TypeScript 主编译器保留 6.0.3，同时保留独立 7.0.2 别名检查，原因是现行 vue-tsc 和 ESLint parser 仍依赖 JavaScript compiler API。jscpd 保留 4.3.0。用相同的 715 个源文件和相同阈值比较后，5.4.3 的分词与跨格式分析报告了 25 处新克隆，4.3.0 报告零克隆。两个版本都支持现有忽略标记，差异需要独立审查扫描基线并完成迁移。`@babel/runtime-corejs3` 保留 Babel 7 兼容线并提升到 7.29.14。tvision-color 1.6.0 产物导入的 Babel 7 路径在 Babel 8.1.2 中已移除，完整构建已复现失败。移除 override 会恢复上游固定的 7.18.9，因此继续使用已提升补丁版本的原有覆盖机制，待上游更新运行时或重建产物后清理。服务端保留上游稳定 Compose/BuildKit/containerd 依赖图要求的 `containerd/platforms v1.0.0-rc.5` 和既有 gorilla/websocket 伪版本，具体 owner、版本与原因记录在清单中。

环境 inventory 通过仓库 collector 和 generator 刷新，记录当前全局 Go 1.26.11、Node 24.19.0 和 Bun 1.3.14。最新工具链的执行使用 task-local 安装和 CI 固定版本，其实际验证结果由对应切片保存。

积压依赖 PR 的替代范围为 #314、#319、#320、#322、#325、#332、#334、#335、#336、#337、#338。对应更新的集成、远端验证、合并与 PR 处理由负责会话跟进，并在交付回执记录实际结果。维护交付跟踪见 [问题 #341](https://github.com/GeWuYou/Graft/issues/341)。
