# 服务端 API 边界与兼容治理规范

## 1. 目标

这份文档定义 `Graft` 在 `server` 侧设计和演进 API 时的 AI Guardrail。

本规范关注：

- `Entity / DTO / VO / Request / Response` 的边界
- OpenAPI authority
- 兼容、废弃、共享契约演进
- review 证据与 CI 适配规则

本规范不是：

- OpenAPI 生成教程
- handler/service/store 分层总设计
- 前端页面契约细节文档

## 2. Authority 与边界总则

### 2.1 Canonical Authority

HTTP API 对外契约 authority 应保持单一。

默认 authority 链：

- request / response 语义定义
- canonical OpenAPI source
- 生成或派生契约
- handler 装配
- `web` 或其它 consumer

规则：

- OpenAPI 是对外 API 契约 authority
- `server` 内部结构体、Ent entity、service model 都不是对外契约 authority
- `web` 是下游 consumer，不得反向定义服务端响应真义

### 2.2 边界对象定义

术语在本仓库中应严格区分：

- `Entity`
  - 持久化实体或 ORM 对象，服务于数据库读写
- `Request`
  - 入口参数模型，服务于 handler 绑定、校验和默认值治理
- `DTO`
  - 跨层传输模型，服务于 service/query/use-case 与 adapter 之间的稳定数据搬运
- `VO`
  - 面向输出视图的结构，服务于最终响应装配
- `Response`
  - 真正对外返回的 HTTP 契约结构

禁止把这些术语混用成“反正都是 struct”。

## 3. Entity / DTO / VO / Request / Response Guardrail

### 3.1 Request 边界

`Request` 只负责：

- 输入字段
- 校验规则
- 默认值
- 与 transport 相关的绑定语义

禁止：

- 在 `Request` 中混入数据库实体字段全集
- 让 `Request` 承担内部 service 输出语义
- 直接把 `Request` 下传到 repository 作为通用查询对象而失去边界

评审必须检查：

- 输入校验是否只定义入口语义
- 默认值是否清晰、可复现
- 是否把内部字段泄露为可输入参数

### 3.2 Entity 边界

`Entity` 是数据库访问模型，不是 API 输出模型。

强制规则：

- 不得直接把 Ent entity 暴露到 HTTP `Response`
- 不得让 OpenAPI 以 Ent entity 作为 schema authority
- 不得把实体字段变更当作 API 自动兼容

原因：

- Entity 包含数据库技术细节、内部字段、加载策略和未来演进噪声
- 直接暴露会让数据库重构变成 API 破坏风险

CI 适合做：

- 阻断明显的 `handler -> ent entity -> JSON` 直接返回模式

### 3.3 DTO 边界

`DTO` 用于稳定跨层传输，不应退化成：

- Entity 别名
- Response 镜像
- 任意字段的大杂烩

使用 DTO 的场景：

- service 输出需要脱离数据库实体
- 多 repository 结果需要组合后再传给 adapter
- 领域逻辑输出和 HTTP 输出并不完全一致

评审必须确认：

- DTO 是否确实隔离了内部模型与外部契约
- DTO 是否具有清晰 owner

### 3.4 VO / Response 边界

`VO` 或 `Response` 应面向输出语义，而不是数据库列名。

规则：

- Response 字段必须服务调用方理解
- 输出结构不得被内部表结构绑死
- 分页响应、错误响应、列表项响应都应有稳定语义边界

允许：

- `VO` 作为内部输出组装对象，再映射为 `Response`

禁止：

- 直接把内部 DTO 原样透出，对外声称“以后再收敛”

## 4. OpenAPI Authority Guardrail

### 4.1 Authority 规则

对外 API 的 schema、字段名、必填语义、枚举语义、废弃标记，必须由 canonical OpenAPI source 驱动。

禁止：

- handler 注释、Go struct tag、前端推断结果各自维护一套 API 真值
- 下游 generated artifact 反过来变成 authority

### 4.2 变更要求

当 API 契约变化影响 consumer 时，必须：

- 先更新 OpenAPI authority
- 再同步派生产物
- 再修改 handler / service / web consumer

不能只改运行时实现而跳过 authority。

### 4.3 Review 证据

评审必须确认：

- 契约变更是否先落在 OpenAPI authority
- 生成产物是否只是同步结果
- 是否存在服务端实现与 OpenAPI 漂移

CI 适合做：

- authority 文件变更与派生产物同步检查
- OpenAPI drift 检查

### 4.4 Realtime API Authority

统一 realtime 订阅也属于 OpenAPI authority 管理范围。

规则：

* 普通订阅类实时能力必须先经过 HTTP 票据签发接口，再进入统一 realtime gateway
* realtime gateway 的 canonical authority 是统一 `/ws` WebSocket 路径和统一 SSE 路径，而不是各模块长期各自维护订阅型连接入口
* topic、ticket、权限、resource scope 的语义必须先落 OpenAPI source，再同步到 runtime 和 `web`
* WebSocket 与 SSE 必须复用同一 topic issuer、授权、一次性 ticket、资源范围校验和断开清理；SSE 仅承载服务端单向事件，WebSocket 继续承载双向会话
* shell 这类双向交互终端能力可以保留独立专用通道，但普通 topic 订阅不再新增模块私有 WS 或 SSE 路径

## 5. 兼容与废弃治理

### 5.1 兼容不是默认答案

新增别名字段、兼容 response、双写双读、老新字段并存前，必须先回答：

- canonical authority 是什么
- 为什么不能直接修 authority
- 哪些 consumer 还依赖旧契约
- 清理触发条件是什么

禁止默认接受：

- “前端还在用，所以后端先兼容一下”
- “为了不改调用方，先多回一个字段”
- “字段先保留，之后再说”

### 5.2 废弃规则

废弃必须显式，不允许静默废弃。

至少要记录：

- 被废弃字段/接口
- 替代方案
- 开始废弃的版本或阶段
- 预期移除条件

允许的表达形式：

- OpenAPI `deprecated`
- 文档中的废弃说明
- response 兼容期说明

### 5.3 兼容期 Guardrail

兼容期内也必须满足：

- canonical 字段只有一个
- 旧字段只是临时桥接，不得继续扩展语义
- 新旧字段返回值必须可解释，不得出现互相矛盾

评审必须确认：

- 是否真的需要兼容期
- 兼容期是否可结束
- 是否存在第二套长期真值风险

### 5.3A 兼容桥接放置规则

服务端 API 兼容桥接必须放在最靠近 transport / response 的边界，默认规则如下：

- 旧 request 字段、旧 query/path 参数兼容：
  - 放在 OpenAPI + Request 绑定层
  - 如需下传，先映射到 canonical DTO / use-case 输入
  - 不把新旧字段同时沉到 Entity、repository 或持久化模型
- 旧 response 字段、兼容 view shape：
  - 放在 VO / Response 装配层或最终 transport adapter
  - service / DTO 内部保持 canonical 语义
  - 不把兼容字段扩散成新的内部模型真值
- 旧 path / endpoint：
  - 放在 OpenAPI、router、handler 边界
  - service/use-case 继续只暴露 canonical 行为入口
- 共享兼容 metadata：
  - authority 仍留在 canonical OpenAPI source 或 module contract
  - bridge 只是在边界临时转译，不生成第二套长期 DTO 目录或 compatibility model

以下做法默认禁止：

- 为单个 API rename / evolution 提前引入通用 compatibility adapter framework
- 因一个 response 字段或 action key 清理就新增 task-store、UpgradeTask 或后台升级编排层
- 把 transport 兼容桥接伪装成 repository upgrade、Entity versioning 或跨模块共享 persistence abstraction

### 5.4 Release-Time Config Compatibility Governance

当变更的是稳定 runtime config key、默认值语义或 operator-facing 配置口径时，按 release governance 额外检查：

- 稳定配置变更必须先分类为以下之一：
  - `additive`
  - `default-change`
  - `rename`
  - `semantic-change`
  - `removal`
- patch release 不允许静默执行以下任一行为：
  - rename stable config key
  - removal stable config key
  - reinterpret stable config semantics
  - 在没有显式升级说明时把稳定默认值变成高风险行为变化
- minor release 若必须发生 `rename`、`semantic-change` 或 `removal`，至少记录：
  - canonical owner
  - deprecated_in
  - removal_target
  - replacement
  - operator action required
  - release-notes required
  - upgrade-notes required
- startup deprecation warning、legacy key alias bridge、config rewrite helper 不是默认承诺；只有 authority 不能直接修复时，
  才允许作为受控兼容例外记录。
- 若确需受控兼容例外，至少记录：
  - canonical key
  - legacy key
  - why direct repair is deferred
  - affected consumers
  - expiry trigger
  - validation expectation

## 6. 共享契约演进 Guardrail

共享契约包括但不限于：

- 分页响应
- 错误响应
- 枚举值
- 过滤/排序参数
- 模块级公共 response item

错误响应额外遵循：

- 内部 typed error 使用稳定 code、message key 与明确安全的 public data 描述展示语义
- cause 只保留在服务端 error chain 与原因日志中，不进入 message、data、details 或其它公开字段
- handler 不把 `err.Error()`、SQL、Docker stderr、路径或依赖响应原文写入统一错误 envelope
- 未登记的 internal error 由 HTTP 最后边界返回 `COMMON_INTERNAL_ERROR`，并仅在尚未 reported 时补一条原因日志

演进规则：

- 优先做向后兼容的增量变更
- 删除、重命名、语义重解释默认视为高风险
- 共用结构一旦被多个模块消费，就必须有明确 owner

当共享契约变化时，必须检查：

- `server` authority
- OpenAPI authority
- 生成契约
- `web` consumer 或其它下游

不得只在某一侧“先适配一下”。

## 7. Review 证据要求

### 7.1 必须提供证据的场景

- 新增或修改对外 API schema
- response 字段新增、删除、重命名
- 分页/错误/枚举等共享契约变更
- 引入兼容字段或废弃字段
- 声称“不会影响 consumer”的变更

### 7.2 最低证据包

最低证据包至少包含：

- authority owner 在哪里
- Request / DTO / VO / Response 各自边界
- 是否涉及 OpenAPI 变更
- 是否涉及兼容期或废弃
- 下游 consumer 影响面

### 7.3 可接受的证据形式

- OpenAPI diff
- 契约结构 diff
- PR 说明中的边界说明
- consumer 影响清单

## 8. CI 适合规则与文档规则

### 8.1 适合进入 CI 的规则

- 阻断 Ent entity 直接暴露为 HTTP response
- OpenAPI authority 与生成产物 drift 检查
- response/request 目录与命名约定检查
- `deprecated` 标记存在性检查

### 8.2 当前仅适合文档与评审的规则

- DTO 是否真的必要
- VO/Response 是否表达了正确业务语义
- 兼容期是否合理
- 废弃窗口是否足够清晰
- 共享契约 owner 是否选对

## 9. 评审清单

评审服务端 API 变更时至少检查：

- 是否直接暴露 Ent entity
- Request、DTO、VO、Response 是否混层
- OpenAPI 是否保持 authority
- 兼容是否有明确理由与退出条件
- 是否显式标记废弃
- 共享契约变更是否同步检查下游
- 是否制造第二套长期真值

## 10. 违规处理

若当前切片无法按本规范直接修复，必须记录：

- 违反哪条 guardrail
- canonical authority 在哪里
- 为什么这次不能向上修
- 兼容桥接影响哪些 consumer
- 何时删除桥接

不得把“兼容一下”当成默认结案。

## 11. 落地要求

后续若仓库引入 AI Guardrail 自动检查，本规范优先落地为：

- Ent entity 暴露检测
- OpenAPI authority drift 检测
- Request/Response 命名与目录规则
- deprecated 标记与兼容说明模板

在没有自动化前，本规范仍作为 code review 与设计评审阻断依据。

## 12. 消除性操作契约

本节是资源删除、关系解除、凭据撤销、不可逆销毁和外部资源清理的 HTTP 契约 authority。领域是否允许删除、资源级授权、审计事件和持久化写入仍由 owning module 负责；不得据此新增跨模块通用 `DeleteService`。

### 12.1 操作类型与 HTTP 形态

| 语义 | HTTP 形态 | 成功结果 | 重试 authority |
| --- | --- | --- | --- |
| 普通资源软删除 | `DELETE /resources/{id}` | `204 No Content` | 同授权作用域 tombstone |
| 关系解除 | `DELETE /resources/{id}/relationships/{relationId}` | `204 No Content` | 关系 tombstone 或当前无绑定事实 |
| 不可逆数据库硬删除 | `POST /resources/{id}/deletions` | `200` operation result 或 `202` Task receipt | 持久化 `Idempotency-Key` receipt |
| 外部资源销毁 | `POST .../remove|destroy|untag` | `202 Accepted` + canonical Task receipt | Task operation identity 与 receipt |

硬删除表达的是不可逆命令，不使用 HTTP DELETE 假装成无回执的资源状态转换。相同 `Idempotency-Key` 与相同规范化输入必须返回原 operation/task receipt；相同 key 绑定不同输入返回 `409`。receipt 不能随着目标业务记录一起被删除。

### 12.2 软删除 tombstone 幂等

软删除必须同时满足：

1. 第一次合法 `DELETE` 返回 204。
2. 目标已由同一授权作用域软删除时，再次 `DELETE` 仍返回 204，不重复执行领域副作用。
3. 资源从未存在，或不属于调用者可操作作用域时返回 404，避免泄露跨作用域存在性。
4. `GET`、列表、候选读取、计数和关联读取默认只查询 `deleted_at = 0`；已删除 tombstone 对这些读取表现为 404 或不可见。

因此 DELETE 的幂等成功不意味着 tombstone 重新对查询可见。任何“删除成功后刷新仍出现”的实现都违反本节。

### 12.3 批量提交模型

- 所有批量消除请求必须有明确 `maxItems`，拒绝空集合和重复 ID，并在可信后端边界执行逐项资源授权。
- 普通资源默认允许 `partial`：合法项可提交，失败项在 200 结果中逐项返回稳定 code。
- role、permission、access binding、credential grant 等安全敏感对象必须使用 `atomic`：先完成全部存在性、生命周期和授权校验，再在一个数据库事务中写入；任一失败时零提交并返回统一 4xx error envelope。
- 会触发外部副作用、跨 authority owner 或无法回滚的批量操作不得伪装成数据库原子事务，应提交 Task 并返回 202。

同步批量成功结果统一使用：

```json
{
  "operation_id": "op_01J...",
  "summary": {"requested": 3, "succeeded": 2, "failed": 1},
  "results": [
    {"id": "user1", "status": "deleted"},
    {"id": "user2", "status": "failed", "code": "FORBIDDEN"},
    {"id": "user3", "status": "deleted"}
  ]
}
```

`results` 必须保持请求顺序并为每个请求 ID 返回恰好一项。`summary.requested = summary.succeeded + summary.failed`。幂等 no-op 可使用 `unchanged` 状态并计入 succeeded，不得凭空省略结果项。

### 12.4 OpenAPI destructive metadata

已迁移的消除性 operation 必须声明 `x-graft-destructive`：

```yaml
x-graft-destructive:
  kind: resource_delete
  effect: soft_delete
  execution: synchronous
  retry:
    mode: tombstone_idempotent
  result:
    status: 204
  authorization:
    owner_check: true
  audit:
    required: true
  confirmation:
    required: false
  exposure:
    mcp: false
```

批量 operation 还必须声明 `batch.mode` 与 `batch.max_items`。`exposure.mcp: true` 必须同时存在合法 `x-graft-mcp`；false 时不得留下 MCP capability metadata。该字段是生成和治理输入，不代替后端认证、授权或确认执行。

迁移期间不得把目标 metadata 提前写到仍使用旧语义的 operation。最终收敛必须让完整消除性操作清单具备 metadata coverage，并删除所有旧 action path、局部 envelope 和兼容 alias。
