# Compose Runner Fixture

此 fixture 使用两个本地版本 image reference 验证 server 与 web 会一起从 `v1.0.0` 切换到 `v1.1.0`。
它不构建、不拉取或启动 Docker 容器；Go 测试通过受限 `ComposeRunnerActions` adapter 记录确定性命令 trace，覆盖真实 runner 的输入校验、固定阶段顺序、receipt 写入和 migration 后 `NEEDS_ATTENTION` 边界。

可用以下只读 Compose 解析确认 fixture 语法：

```sh
docker compose --env-file source.env -f compose.yml config
docker compose --env-file target.env -f compose.yml config
```
