# ADR-006: Platform Self Update Compose Runner

- Status: accepted
- Date: 2026-07-22
- Scope: `server/modules/update/**`, Compose update execution, and release delivery artifacts

## Context

官方 Compose 部署中的 server 自身会在 `docker compose up -d` 期间被 recreate。server 镜像还以非 root 用户运行，不带 Docker Compose CLI，且容器内看到的路径不能可靠地解释为 Docker daemon 所在宿主机路径。因此 server 进程直接执行 update 无法可靠等待健康检查、写入最终结果或安全定位 Compose 根目录。

持久 Update Agent 虽能解决这些问题，却引入第二个长期版本、生命周期和信任边界，超出当前平台能力的必要复杂度。

## Decision

采用短生命周期、receipt-writing 的 Compose runner：

- server 在完整 preflight 后通过 Docker socket 启动 runner；runner 不提供 API、不保存 Graft 业务状态、不常驻。
- runner 只接受 digest-pinned target、已验证的 official Compose root、固定的 Compose command allowlist 和受限 receipt 输出位置。
- Compose runner 只接受由 Deployment Runtime freeze 的 `DeploymentContext`。`GRAFT_DEPLOYMENT_COMPOSE_ROOT` 已设置时拥有最高优先级：它必须是非空宿主机绝对路径，并以相同绝对路径挂载进 runner；显式空值或路径校验失败时不得回退到另一种 runtime 或自动发现。只有变量未设置时，且 Docker API 可用，Deployment Runtime 才通过 Container 提供的当前 server 原始 inspect facts（Compose labels、config files 和 bind mounts）生成候选并要求管理员确认后才能执行；发现不可用时必须 fail closed。ADR-008 拥有 Deployment Runtime 的解析边界与配置语义。
- 自动发现的候选使用 opaque candidate key 暴露给 Web；server 在启动升级时重新发现并校验 key 与当前 Docker runtime facts 的一致性。候选路径只属于当前操作，不持久化到 System Config、数据库或其他存储；Docker daemon 返回的 Linux host path 是 runner 执行和挂载的唯一权威路径。
- runner 顺序执行独立 backup capability、pull、显式 bootstrap migration、受控 server/web recreate、health check，然后写 durable receipt。`BackupCompletion` 是 `compose-runner/v1` receipt 的必填、版本化证据白名单字段；它只承载 operation/task 绑定的 SHA-256 与字节数。Backup 动作失败、未产生该字段或字段不通过绑定校验时，runner 必须写入失败 receipt，不能把更新标记为成功。Task Runtime 和 update history 之后消费 receipt，形成最终审计事实。
- runner 不替换容器内 binary、不使用 mutable `latest`/`beta` tag 作为目标，也不实现自动 schema rollback。

### Protocol And Receipt Boundary

runner input and receipt use a versioned schema. Input contains only operation identity, Task ID, target immutable
server/web references, the server-validated host absolute Compose root, socket location and preflight evidence. It is
passed inline through the Docker API and never requires the server container to write into the discovered host path. It never contains `.env`,
database URLs, dump contents, or arbitrary commands. Receipt contains only operation identity, migration-started
evidence, terminal result, stable failure code, and the versioned `BackupCompletion` integrity evidence required after
backup validation succeeds. That evidence is limited to the operation/task binding, SHA-256 digests, and byte counts;
it never contains backup storage references. A receipt that advances beyond the backup boundary must include valid
`BackupCompletion`, while a backup failure receipt may omit it. The runner image itself must be a separately published,
digest-pinned release asset. Canonical `release-manifest.json` includes `runners.compose.image`, `digest`, and
`reference`; its companion checksum is verified before the manifest is consumed. The runner coordinate is the official
sibling `ghcr.io/<owner>/graft-updater` of the server image. An unpinned local image, a mutable tag, or a
missing runner declaration is not an execution authority. The same release workflow builds the runner from
`server/runner/compose/Dockerfile`; its Buildx output is the only publication authority. That emitted immutable digest
is recorded in the manifest and must match the constructed GHCR reference. Neither an externally injected digest nor a
locally built substitute may stand in for that workflow output.

The fixed order is `backup -> compose pull -> bootstrap migrate up -> recreate server/web -> Docker health ->
/healthz -> receipt`. A failure before migration starts may restore the configuration snapshot and old image
references and records `RECOVERED`. Once migration has started, any failed verification records
`NEEDS_ATTENTION`; neither runner nor server performs a database rollback or restore.

A successful terminal receipt always includes the Backup capability's completion evidence. The protocol does not
permit a receipt to advance beyond backup validation, or a successful terminal result, when that evidence is missing
or invalid. The retained runner container is labeled with the operation identity and protocol version; the server reads
the marker-bounded receipt from its Docker logs through the Docker API, validates the labels and operation binding, and
only then settles the operation and removes the completed runner. A missing, malformed, or mismatched log receipt is not
success evidence.

Task Runtime, rather than Update, must own authenticated receipt settlement after server recreation. Update may
submit an operation Task and consume its public settlement capability, but must not write `tasks`, `task_stages` or
`task_events` directly. Backup remains owner of backup metadata; runner handoff records only backup evidence through
the narrow Backup capability and never exposes its storage references through Update HTTP responses.

## Consequences

正面影响：server 被 recreate 后仍能完成健康验证；运行时镜像由 immutable digest 确定；未填写宿主路径时仍能基于 Docker runtime facts 提供可确认的升级路径；无须维护 agent-to-server 协议或 agent 升级链。

代价和约束：Docker socket 是高权限信任边界，官方 Compose 安装是唯一可执行 MVP profile；自动发现依赖当前 server 容器的 labels 和 bind mounts，候选缺失或不一致时仍需 operator 指引；runner crash 通过保留容器日志中的 receipt、Task Runtime 重试/失败语义和 operator recovery 处理，不承诺透明自动恢复。

## Rejected Alternatives

- 容器内下载并替换 binary：破坏 immutable image 和 Compose deployment model。
- server 直接运行 `docker compose`：无法跨自身 recreate 可靠完成生命周期，且路径与权限不成立。
- 常驻 Update Agent：当前阶段没有足以抵消新增版本兼容、部署和 crash-recovery 成本的通用执行需求。
