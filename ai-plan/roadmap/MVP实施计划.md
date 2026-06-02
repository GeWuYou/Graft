# 后端主导的 MVP 闭环收敛计划

## 1. 当前阶段目标

当前阶段统一采用：

**后端补齐 MVP 闭环 + 前端收敛真实契约**

目标不是扩展平台能力面，而是在当前已有 `Runtime`、插件系统、CLI、Ent + Atlas、Redis/PostgreSQL、
`menu / permission / cron registry`、`user` 与 `rbac` 最小实现的基础上，补齐后端最小闭环，并让 `web`
围绕真实后端契约完成壳层收敛。

当前阶段完成标准：

* 平台可以稳定启动
* 插件可以注册并按依赖顺序运行
* `auth + user + rbac + audit + scheduler` 形成最小后端闭环
* 前端可以接通真实登录、刷新、当前用户、菜单、路由、权限链路
* 插件式扩展路径已经稳定，且没有为了赶功能而破坏边界

---

## 2. 当前阶段范围

### 包含

* 核心运行时与显式 CLI
* 轻量 DI / 服务注册
* `auth` 插件最小认证与会话生命周期能力
* `user` 插件最小用户资料与用户管理能力
* `rbac` 插件最小授权能力
* 最小进程内 `event bus`
* 审计日志最小版
* 定时任务最小版
* Vue 3 + TDesign 管理后台壳的真实契约接入

### 不包含

* 运维类插件
* 第三方插件分发
* 热插拔
* 复杂工作流
* AI 业务功能
* 大规模 CRUD 扩展
* 高级权限系统
* 中台化能力扩展

---

## 3. 后端实施顺序

### 阶段一：已基本落地的内核与注册中心基线

已具备或基本具备：

* `app`
* `config`
* `logger`
* `database`
* `http`
* `container`
* `module manager`
* Ent baseline and repository / store factory boundary
* `menu registry`
* `permission registry`
* `cron registry`

约束：

* schema 变更基线使用 Ent + Atlas versioned migrations
* 迁移执行通过显式 CLI 步骤完成，不在应用启动流程中隐式执行

### 阶段二：补齐 MVP 闭环缺口

本阶段必须补齐：

* `event bus`
* `audit`
* `scheduler`

验收：

* `Runtime` 统一注入 `event bus`
* 模块可以通过依赖注入获取 `event bus`
* `audit` 同时支持 HTTP middleware 自动审计与 event bus 主动审计
* `scheduler` 通过仓库内封装接口运行，而不是让业务直接依赖 `robfig/cron`

### 阶段三：稳定真实后端契约

本阶段后端优先稳定这些契约：

* `POST /api/auth/login`
* `POST /api/auth/refresh`
* 当前用户接口
* 菜单元信息
* 权限判定语义
* 稳定错误响应 `message_key + message + locale`

验收：

* 登录、刷新、当前用户、菜单、权限守卫链路可直接支持前端接入
* 不再把当前阶段重点继续转向 session 治理的功能面扩张

---

## 4. Event Bus 方向

当前阶段采用：

**自研最小进程内 event bus**

约束：

* 放置于 `server/internal/eventbus`
* 只支持 MVP 所需能力：
  * `Subscribe`
  * `Publish`
  * handler 注册
  * 同步或简单异步派发
  * handler panic recover
  * 错误日志记录
* 目标是模块解耦，不是分布式消息系统
* 不引入 Kafka、RabbitMQ、NATS 等 MQ
* 设计必须避免与未来 MQ 替换路径形成强耦合
* `Runtime` 启动时统一注入 event bus
* 模块通过依赖注入获取 event bus

推荐边界：

* 公开接口只表达事件发布与订阅
* 不提前暴露 ack、retry、dead-letter、partition、consumer-group 等分布式语义
* handler 注册属于模块 `Register` 阶段
* bus 生命周期由 `Runtime` 持有，模块只消费其稳定接口

---

## 5. Audit 模块方向

当前阶段采用：

**业务贴合型自研审计最小实现**

约束：

* 新增 `server/internal/audit`
* 新增 `server/modules/audit`
* 仅记录：
  * `operator_id`
  * `operator_name`
  * `action`
  * `resource_type`
  * `resource_id`
  * `request_method`
  * `request_path`
  * `ip`
  * `user_agent`
  * `success`
  * `error_message`
  * `created_at`
* 支持：
  * HTTP middleware 自动审计
  * event bus 主动审计事件

职责边界：

* middleware 负责标准请求级自动审计
* event bus 负责明确业务语义或非 HTTP 入口动作的主动审计
* middleware 不负责理解全部业务语义
* event 审计不替代通用请求级落盘

明确不做：

* 审计检索 DSL
* 审计回放
* 审计归档
* 风险分析

---

## 6. Scheduler 模块方向

当前阶段采用：

**以 `robfig/cron/v3` 为底层，但通过仓库内封装隔离实现**

约束：

* 新增 `server/internal/scheduler`
* 新增 `server/modules/scheduler`
* 业务代码禁止直接依赖 `github.com/robfig/cron/v3`
* 必须通过自定义 `Scheduler` 接口隔离底层实现
* 需要支持：
  * `RegisterJob`
  * `Start`
  * `Stop`
  * `RemoveJob`
* MVP 仅支持：
  * 进程内调度
  * cron 表达式
  * 固定 interval

职责边界：

* `cron registry` 负责模块注册任务声明
* `scheduler` 负责把任务声明装配成实际运行中的调度器
* `Runtime` 通过模块生命周期统一启动和关闭调度器

明确不做：

* 分布式调度
* 持久化任务恢复
* DAG 工作流
* 可视化编排
* 多节点抢占

---

## 7. Web 当前阶段策略

当前阶段 `web` 的目标不是扩展后台业务页面，而是：

**收敛 starter 壳层 + 接通真实后端契约 + 稳定基础前端治理能力**

当前阶段 `web` 的核心目标仅包括：

* 登录
* token refresh
* 当前用户获取
* 动态菜单
* 动态路由
* permission guard
* API client
* 错误处理
* starter 壳层收敛
* 与真实后端 `menu / permission / api` 契约对齐

明确禁止：

* 大规模 CRUD 页面扩展
* 假数据驱动开发
* mock 优先开发
* 演示型 dashboard 扩展
* 与真实后端脱节的静态菜单
* 自定义复杂状态管理体系
* 提前做高级页面组件库
* 提前做低代码 / 工作流 / UI 编排系统
* 复杂可视化平台建设

当前阶段 `web` 不以这些指标衡量进度：

* 页面数量
* CRUD 数量
* UI 丰富程度

而以这些标准判断完成度：

* 是否完成真实登录链路
* 是否完成 token refresh
* 是否能获取当前用户并恢复登录态
* 是否能动态装配菜单与路由
* 是否完成 permission guard
* 是否能稳定处理未授权与错误响应
* 是否完成真实后端契约联调

当前阶段前后端协作采用：

**后端主导契约稳定，前端围绕真实契约收敛**

而不是前后端并行自由扩展业务功能。

---

## 8. 测试清单

必须覆盖：

* 插件缺失依赖时报错
* 插件循环依赖时报错
* 插件注册顺序正确
* event bus 的 handler panic recover 与错误日志路径
* `audit` 的 middleware 自动审计路径
* `audit` 的 event bus 主动审计路径
* `scheduler` 的任务注册、启动、停止、移除路径
* 登录后动态菜单正确
* 未授权路由不可访问
* 登录、刷新、当前用户、菜单、路由、权限守卫链路可联调

---

## 9. 当前阶段必须完成

* `event bus` 最小实现与 runtime 注入
* `audit` 最小实现
* `scheduler` 最小实现
* `auth + user + rbac + audit + scheduler` 在同一 runtime 内形成最小闭环
* 前端真实登录、刷新、当前用户、菜单、路由、权限链路接通
* starter 壳层完成与真实后端契约一致的基础收敛

---

## 10. 明确延期到后续版本

* 完整用户管理 CRUD
* 完整角色管理 CRUD
* 完整权限管理 CRUD
* 审计日志高级查询与分析
* 调度任务持久化恢复与分布式化
* 更丰富的 session 治理与审计联动扩展
* 高级权限模型与复杂授权管理后台
* dashboard、美化型展示页与复杂可视化工作台
* 低代码、工作流、UI 编排能力
* 运维类插件与其它更大范围插件族

---

## 11. 代码实施前的前置条件

编码前先确认这些文档已经稳定：

* 插件契约
* DI 职责边界
* 前端目录与模块规则
* 当前阶段范围

如果这些边界还在频繁变化，就不应该开始大规模实现或功能面扩张。
