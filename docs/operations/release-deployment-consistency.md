# 发布与部署一致性 Runbook

本 runbook 覆盖两条线：release candidate 产物流（`.github/workflows/release-candidate.yml`）与
部署后一致性检查（`scripts/check_deployment_consistency.py`）。稳定版 `v*` 标签仍由 semantic-release
（`release.yml`）唯一拥有；candidate 流不创建 `v*` 标签，不会触发 `publish.yml`。

## Release Candidate 流程

1. 触发（需要 semver 版本号与渠道）：

   ```bash
   gh workflow run release-candidate.yml -f version=1.2.3 -f channel=beta
   # 或 stable
   ```

2. 构建与扫描：workflow 从 dispatch SHA 构建带版本链接信息的 server 二进制与 web release 分发，
   构建镜像（`org.opencontainers.image.revision` 标签绑定源码 SHA），并用固定版本 trivy 扫描两遍：
   非阻断全量扫描（`ignore-unfixed: false`）记录包含未修复项的完整报告，上传为
   `release-candidate-unfixed-scan` artifact，其 SHA-256 摘要绑定进 release.json 供发布决策审计；
   阻断门禁（`ignore-unfixed: true`，HIGH,CRITICAL，exit-code 1）只让存在修复方案的发现拦截
   candidate。未修复项不拦截发布，但扫描证据缺失同样使 candidate 注册失败，不会静默放行。
3. 推送与登记：镜像以 `candidate-v<version>-<sha>` 推到 ghcr，`release.json` candidate artifact
   绑定 run_id、run_attempt、SHA、version、channel 与两个镜像的 `sha256:` 摘要并上传。
4. 汇总门禁（`Required checks passed`）：needs 结果校验 + Actions API jobs 列表逐 job 校验
   （GET 固定、绑定当前 run/attempt/SHA、fail-closed），再校验 candidate artifact 与本次运行绑定一致。
5. 幂等打标签：`stable` 渠道创建 `rc-v<version>`，`beta` 渠道创建 `rc-v<version>-beta`。

   - 同名标签指向同一 SHA：视为已处理，成功退出（幂等重跑）。
   - 同名标签指向不同 SHA：硬失败并给出错误说明；不允许覆盖。
   - 标签经 `git ls-remote` peeled 比较判定，轻量与附注标签同等对待。

重跑纪律：workflow 按 ref 分组并发、不取消进行中的运行；失败的 candidate 修复输入后重新 dispatch，
已存在的同名同 SHA 标签不会重复创建。

## 部署后一致性检查

目标：杜绝「HTML 200 但 JS 404」被读成部署成功。检查覆盖入口 HTML 与其引用的全部
entry JS/CSS 资产（`script[src]`、`link rel=stylesheet`、`link rel=modulepreload`）。

```bash
just check-deployment --base-url https://<deployment-host>/
just check-deployment --base-url http://127.0.0.1:3000/ --expect-revision <git-sha>
just check-deployment --base-url http://127.0.0.1:3000/ --expect-source-digest <digest>
```

判定契约：

- 状态码：2xx 通过；任何 5xx（含 503）默认失败——可接受状态必须逐个显式传入
  `--acceptable-status <code>`；4xx 永不接受。默认契约是"没有可接受的 5xx"。
- Content-Type：HTML 必须为 `text/html`；JS 资产须为 `text/javascript` / `application/javascript`；
  CSS 资产须为 `text/css`。
- entry 资产：HTML 中未发现任何 entry JS/CSS 资产（例如错误页）直接失败。
- revision / source digest：HTML `meta`（`revision` / `graft-revision` / `source-revision`、
  `source-digest` 等名称）声明的值与 `--expect-revision` / `--expect-source-digest` 比对；
  给出期望值而 HTML 未声明对应标记时 fail-closed（不猜测）。当前 web 入口尚未注入 revision 标记，
  该契约需要在 web 仓库注入标记后启用；在此之前 checker 会如实报告"已声明但未校验"。
- 每个抓取体的 SHA-256 写入 JSON 报告；`verdict: fail` 时退出码为 1。

## 标准部署验收顺序

1. 部署完成后先做常规健康检查（server `/healthz`、web `/`）。
2. 运行 `just check-deployment --base-url <部署地址> --expect-revision <部署的源码 SHA>`（标记可用后）。
3. 预览实例冒烟见 [预览实例生命周期](../development/preview-lifecycle.md)；两者都通过才算部署验收完成。
4. 检查结果作为交付证据留存（JSON 输出归档），失败原因逐条列出，不接受"部分通过"。
