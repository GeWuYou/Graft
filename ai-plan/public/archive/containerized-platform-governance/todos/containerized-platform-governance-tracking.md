# Containerized Platform Governance Tracking

## Work Contract

```yaml
version: 1
kind: refactor
scope: long-running
authority_summary: Root AGENTS, domain AGENTS, ADR-029 and module contracts
requires:
  design: true
  topic: true
  roadmap: true
  adr: true
execution:
  engine: graft-multi-agent-loop
  dispatch_skill: graft-multi-agent-task
bootstrap:
  targets:
    - topic
    - design
    - roadmap
    - adr
closeout:
  archive: true
  lessons_review: true
```

## Closeout

2026-11-16，四批实现、独立评审修正、实际运行与完整 CI 验收完成，按 Work Contract 归档。
PR [#351](https://github.com/GeWuYou/Graft/pull/351) 由负责会话继续合并、资源释放与最终报告。
发布和生产部署属于独立交付操作。

## Checklist

- [x] 工程定位统一、根 Bun workspace、定向验证、Turbo 缓存与 CI 单次测试流。
- [x] 本机 Docker/Compose Task 执行、持久化资源协调、取消与中断处置。
- [x] 本机认证构建发布、独立平台更新 Controller 与跨重启回执。
- [x] 模式迁移预检、实际执行就绪诊断、公开契约与应用操作入口。

## Runtime Evidence

- 本地生产源码 976144490bf02ccfab35d7d8092881a4d9238350 的全新官方 Compose 安装通过 24 项必需 API 与实际引擎场景。head 795c9a8345720679bc0d9c8cdd0b09b662ae6184 的完整 CI 对私有 daemon 与官方安装分别执行并通过 24 项场景。
- Build Task 10 成功，认证 Registry 的匿名 401、认证 200、上传快照、公开 OCI artifact 与 publication 均完成。
  采用 1d9703272 的契约消费修正读取真实 manifest config digest，绑定 Engine Image ID，提取上传 context 的 /probe 并核对内容摘要。
- 更新故障场景验证配置快照失败的负回执，正常 Update API 使 Task 12 成功、Backup 1 AVAILABLE，server 从 beta.1 替换为 beta.2。
  Task、Backup API 与 Controller 回执中的配置及数据库备份摘要、字节数和执行 attempt 完整绑定。
- 数据库 dump 在新的私有 PostgreSQL 数据库恢复成功，核对 users=1、roles=8、runtime_targets=1、applications=3、build_v2_artifacts=1、workspace snapshots=1、migration revisions=151，演练数据库随后删除。
- [完整 CI run 43794608388](https://github.com/GeWuYou/Graft/actions/runs/43794608388) 对 head 795c9a8345720679bc0d9c8cdd0b09b662ae6184 全部通过，包含源码门禁、PostgreSQL 迁移、私有与官方安装、认证 Build、完整 Update API、身份受控清理和 Required checks 汇总。

## Operational Boundaries

- 每个数据库由一个活跃 Task Runtime 持有数据库会话执行锁，执行所有权失效会停止 admission。
- Controller 使用短生命周期 digest-pinned 镜像，root 身份仅增加 CHOWN、DAC_OVERRIDE，根文件系统只读、网络关闭、禁止提权，其余 capability 全部删除。
- 两个测试版本来自同一 Git revision，更新证据覆盖现有迁移图入口调用。PostgreSQL 迁移门禁已通过。含新 DDL 的跨版本平台更新需独立版本场景验收。
- Secret Scan 保留完整提交历史扫描。唯一精确例外绑定 Postgres 检测器、合成 URI 规范化 Raw SHA-256、引入提交 d8172f6eafac645051c832e81e8b0ba22d153bb6、文件与第 45 行。
  该已知测试值可有 DNS 验证错误，Verified=true 及其他身份的结果继续阻断；当前测试在运行时组合该值。
- 应用数据备份、域名 HTTPS、宿主机管理等后续能力继续由独立路线推进。

## Preparation Evidence

定向检查覆盖 consumer 选择、删除文件、部分暂存和重复命令去重。Turbo 缓存覆盖源码、契约、配置、锁、环境与工具链变化，并恢复完整 dist。
同输入的直接构建、冷 Turbo 构建与稳定命中测量分别为 81.28 秒、83.39 秒、0.16 秒，完整 CI 的耗时单独统计。
八项远端评审意见及其中的 recovery 子问题均已修复，历史 Agent 指纹、共享镜像标签、busy retry 和短暂状态写失败分别有直接回归。
