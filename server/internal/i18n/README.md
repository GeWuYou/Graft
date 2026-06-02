# i18n

## 用途

`i18n` 为 `server` 提供统一的语言解析、消息查找、注册期 registry 与 freeze 能力。

## 职责边界

这个模块负责：

* 解析请求或会话偏好的 locale
* 维护平台级公共消息 key 与 server 侧 fallback 文案
* 在模块 `Register` 阶段接收显式 message registration
* 负责 duplicate-key 与 locale 校验
* 在运行期切换到 freeze 后只保留只读 lookup
* 按默认语言和回退语言输出稳定 `key + fallback` 解析结果

这个模块不负责：

* 业务模块完整的多语言资源管理后台
* 前端页面翻译加载
* 前端菜单标题、权限展示名或页面 copy 的长期真相
* 复杂 ICU 模板或运行时热更新
* `message key -> error code` 契约映射
* 在 facade 外暴露未来可能引入的第三方 i18n 类型

## 主要入口

* `doc.go`：包职责说明
* `service.go`：locale 解析、消息查找与回退逻辑
* `service_test.go`：registry / freeze / fallback 行为断言

## 关键依赖

* 由 `server/internal/app` 在 runtime 装配阶段创建
* 依赖 `server/internal/config` 提供默认语言与支持语言配置
* 供 `httpx`、core 与模块通过 `module.Context` 共享使用

## 维护提示

新增平台级错误消息时，应优先增加稳定 `message_key`，再补充不同语言文案；
不要直接把面向用户的长文本散落到各模块路由处理函数里。
模块消息资源应通过 facade 注册，不要在模块内维护平行的消息目录真值。
当跨边界响应同时带有 `messageKey/title_key` 与 `message/title` 时，`server` 侧文案只承担兼容回退，
不能把本模块扩张成 UI copy center。
