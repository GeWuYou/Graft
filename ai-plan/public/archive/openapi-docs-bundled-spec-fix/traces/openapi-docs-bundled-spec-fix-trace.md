# OpenAPI Docs Bundled Spec Fix Trace

## 2026-05-24

- 按 root `AGENTS.md` 完成 startup preflight，本轮任务维持 `cross-boundary`。
- 确认当前 `/openapi.json` 真实来源是 `openapi/openapi.yaml` 经 `kin-openapi` 加载后直接 `MarshalJSON()`，不是 bundled artifact。
- 确认根文档仍然包含外部文件 `$ref`：
  - `./paths/**`
  - `./components/**`
- 确认 `openapi/paths/**` 本身有真实 `get` / `post` operation，因此 `/docs` 只显示 tags / models 的直接原因不是 spec 缺 operation，而是浏览器端无法解析仓库文件系统里的 fragment。
- 评估现有工具链后，没有发现现成 bundle 命令或 checked-in bundle artifact。
- 复用当前依赖树里的 Redocly bundling 能力生成 `openapi/dist/openapi.bundle.json`，避免新增新的 OpenAPI bundler 依赖。
- `server/internal/app/openapi_docs.go` 改为读取 bundled JSON，并在启动时校验：
  - bundle 文件可加载
  - bundle 文件可通过 OpenAPI 校验
  - bundle 文件不再包含 `./paths/` 或 `./components/`
- `/openapi.yaml` 继续返回原始拆分源文档入口。
- `/docs` 继续读取 `/openapi.json`，不改路由位置、不改 Scalar 主题、不扩展到 `web` 菜单。

## Validation Intent

- `git diff --check`
- `cd server && go test ./...`
- `cd server && go run ./cmd/graft validate backend`
- 文件级检查 bundled JSON：
  - 存在真实 operation
  - 不再包含 `./paths/`
  - 不再包含 `./components/`

## Known Risks

- `openapi/dist/openapi.bundle.json` 仍然可能与 `openapi/openapi.yaml` 漂移。
- 本轮没有把 stale check 接入 CI / hooks。
