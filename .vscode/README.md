# Local VS Code tasks

`.vscode/tasks.json` 是开发者本机配置，仓库不会跟踪它。首次使用或切换 PostgreSQL 拓扑时，从以下模板中选择一份复制为 `.vscode/tasks.json`：

```bash
# 使用 server/.env 中已有的共享 PostgreSQL
cp .vscode/tasks.shared-postgres.example.json .vscode/tasks.json

# 使用 127.0.0.1:15432 上由开发命令管理的独立 PostgreSQL
cp .vscode/tasks.isolated-postgres.example.json .vscode/tasks.json
```

两份模板使用相同的任务标签，均兼容 `.vscode/launch.json` 中的 `fullstack: dev`；区别仅在 Docker Runtime Agent 开发命令显式选择 `shared` 或 `isolated` 数据库模式。
