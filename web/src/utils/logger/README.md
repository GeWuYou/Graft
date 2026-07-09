# Logger

`web/src/utils/logger` 提供前端统一日志基础设施。

## 目的

- 为 `web` 提供稳定、可治理、可扩展的日志入口
- 将业务代码与底层 `consola` 实现隔离
- 为未来 `sentry`、`remote`、`telemetry` transport 预留边界

## 边界

- 业务代码只能通过 `createLogger()` 获取 logger
- 不允许业务代码直接 `import consola`
- `LoggerCore` 负责级别判断、`Error` 归一化、context 合并和 `LogEvent` 构造
- transport 只负责输出 `LogEvent`

## 推荐用法

```ts
const requestLogger = createLogger('request');
const authLogger = requestLogger.child('auth');

authLogger.error(error, {
  requestId,
});
```

## 注意事项

- `moduleName` 必须稳定、短小、能力导向
- `meta` / `context` 默认要求可 JSON 序列化
- 禁止输出 `token`、`password`、`Authorization`、`cookie` 等敏感信息
- `logger` 负责调试和排障，不替代 `MessagePlugin` 等用户提示机制
- 长期保留的调试日志必须挂到 `web/src/shared/debug/**` 的 namespaced debug flag 下，不直接把 `logger.debug` 散落到页面或 store
- `window.__GRAFT_DEBUG__` 只作为开发者控制台入口；真实调试状态由 shell-owned debug store 持有
