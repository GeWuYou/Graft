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

## 6. 共享契约演进 Guardrail

共享契约包括但不限于：

- 分页响应
- 错误响应
- 枚举值
- 过滤/排序参数
- 模块级公共 response item

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
