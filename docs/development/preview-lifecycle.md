# 预览实例生命周期

根 AGENTS.md 持有授权与 worktree 边界：worktree 内禁止启动长驻服务，预览实例属于开发者主检出工作；
本页定义 `scripts/preview.py` 的实例身份、注册表、端口与销毁纪律，`scripts/preview_recovery.py`
定义只读对账。

## 实例身份与注册表

- 实例身份由「检出路径 + 任务 ID」确定性推导：`slug(任务ID)-<sha256 前 12 位>`。同一检出同一任务
  在所有命令中解析出同一身份，无需共享句柄。
- 注册表存放在 Git common 目录（worktree 丢失后仍可对账）：`<git-common-dir>/graft-preview/registry.json`，
  原子写入，权限 0600。实例目录、`.env`、compose 副本与日志同样位于该目录下 `instances/<identity>/`。
- `GRAFT_PREVIEW_STATE_DIR` 可覆盖状态根目录（单元测试与隔离演练用）；覆盖后 recovery 的任务归属证明
  按设计失败关闭，只有从注册检出运行或仓库绑定注册表才可获销毁建议。

## 生命周期命令

```bash
just preview-start --task my-task --image-tag candidate-v1.2.3-abc
just preview-status --task my-task        # 或 --identity <identity> / --all
just preview-logs --task my-task --tail 200
just preview-stop --task my-task          # 保留容器与数据卷
just preview-destroy --task my-task --confirm-instance <status 输出的完整身份>
```

- `start` 生成实例 `.env`（GRAFT_IMAGE_TAG、随机 POSTGRES_PASSWORD / GRAFT_AUTH_JWT_SECRET、
  每实例命名的 build-snapshot/update-state 卷、实例独立的备份目录）并复制 compose 文件；已注册实例收敛启动
  （复用保留端口与已存密钥，不重新生成凭据）；注册态为 running 但健康探测失败时（容器被外部停止或删除），
  start 会回落到 compose up 重建栈而不是虚报成功。注册表读写段由 git-common-dir 下的文件锁串行化，
  端口分配与容量预留都在锁内完成；调用 Compose 时宿主 shell 导出的保留键（注册端口、镜像 tag、
  实例卷、备份路径等）会被剥离，Compose 只读实例 `.env`，导出的部署设置无法破坏预览隔离。
  `--image-tag` 必须是裸 tag（不含 `/` 或 `:`）：compose.yml 已固定镜像名，
  传入完整镜像引用会造成双重引用，脚本直接拒绝。`--env KEY=VALUE` 可追加变量，保留键不可覆盖。
- `status` 输出纯 JSON：URL、实时健康探测、日志位置、源码摘要（注册时 compose 摘要、当前实例摘要、
  检出 compose 摘要三向对比的 drift 结论）。
- `stop` 仅 `compose stop`：容器与数据保留，可再次 `start` 收敛。`destroy` 要求
  `--confirm-instance` 与注册身份逐字一致：先 `compose down --volumes --remove-orphans`，再在特权容器
  （栈自身的 postgres 镜像短驻运行）内直接删除容器属主数据（apps/、backups/、.data/ 等以 UID 10001 或
  root 写入的目录），宿主只移除开发者属主的剩余文件（compose.yml、.env、logs），目录不会留下 root
  属主残留；随后把注册项连同 `evidence_sha256` 归档进 `archive/` 与 `archive/manifest.json`。清理失败时
  注册项保留并标记 `cleanup_failed`，阻塞解除后可原样重试 destroy，不会留下脱离注册的半清理状态。
  若实例目录或其文件已不是开发者属主（旧版清理遗留的 root 属主残留），destroy 先用一次特权运行把属主
  归还给调用者（数字 uid:gid 的 chown 加 u+rwX），再走正常清理路径。
  `status` 把不可读的实例目录与不可读的检出 compose 文件分别报告为独立异常，互不影响对方的摘要结论。

## 端口与容量

- `GRAFT_PREVIEW_PORT_BASE`（默认 38000）起在 64 个端口内分配。分配时跳过已注册实例的端口；遇到非注册的
  外部监听端口直接拒绝（提示显式 `--port`），不静默换位。显式 `--port` 与注册端口或被占端口冲突同样拒绝。
- `PREVIEW_MAX_RUNNING`（默认 3）限制同时 running 的实例数；达到上限后新实例与已停止实例的重启
  都必须先 stop/destroy 一个。

## 只读对账（recovery）

```bash
just preview-recovery                       # 等价 python3 -B scripts/preview_recovery.py scan
just preview-recovery --identity <identity> \
  --claim-ownership <identity> --claim-task-id <identity>=<任务ID>
```

- 扫描永不写入；输出 JSON：每个实例的 checkout 是否存在、实例文件是否缺失、未注册实例目录、
  归档证据（manifest 摘要与内部 evidence 摘要双重校验）、异常列表与 verdict。发现异常时退出码为 1。
- 销毁建议仅在归属证明通过时给出：命令本身在注册检出内运行，或显式 identity+task 声明且注册表为
  仓库绑定位置。建议命令始终使用 `destroy --identity <identity> --confirm-instance <identity>`，
  worktree 已丢失时无法从 `--task` 重新推导身份。健康的 running 实例不会得到销毁建议。
- 归档证据用于事后审计：清理记录被篡改、manifest 列项缺失或出现未登记归档文件，都在对账中显式报告。

## 主检出后续（worktree 内不执行）

worktree 只做静态检查与单元测试；真实预览冒烟在合并后的主检出执行：

```bash
python3 -B scripts/preview.py start --task smoke --image-tag <裸 tag，例如 candidate-v1.2.3-abc>
python3 -B scripts/preview.py status --task smoke
python3 -B scripts/preview.py destroy --task smoke --confirm-instance <identity>
```

预览所需镜像来自 release-candidate 工作流产物，见
[发布与部署一致性](../operations/release-deployment-consistency.md)。
