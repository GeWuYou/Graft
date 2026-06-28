# AI代码生成与Review规范

## 1. 目标

本规范定义 `Graft` 中 AI agent 参与代码生成、修改、评审与多 agent 协作时的执行边界。

目标：

- 限制 AI 在 authority 不清、风险过高或跨切片过大的场景下擅自扩张。
- 统一 closeout 证据，避免“改了什么、风险在哪、怎么回滚”不可复核。
- 给单 agent 与多 agent 协作提供相同的 review 清单。

## 2. 默认工作方式

- 先识别 authority owner，再冻结改动范围。
- 只在当前 owned scope 内修改；发现 authority 在上游时，先升级任务范围，不做下游补丁伪完成。
- 没有明确要求时，不做顺手修复、顺手重命名、顺手升级依赖或顺手整理结构。
- 以最小闭环完成当前切片：规则、实现、必要测试、closeout 证据。

## 3. Agent 禁区

以下场景默认属于 agent no-go area，未经明确授权不得直接落地：

- 跨模块重构
- 跨 `server` / `web` / OpenAPI / shared contract 的连锁改动
- 自动生成或自动改写数据库迁移
- 依赖升级、锁文件批量刷新、工具链版本漂移
- 大范围 rename wave、目录迁移、批量移动文件
- 以“统一风格”为名的仓库级格式化或整理 import

这些场景如确有必要，必须先形成明确设计、范围、验证与回滚路径，再进入实施。

## 4. 禁止机会主义修复

AI 在当前任务中不得：

- 顺手修 unrelated warning、lint、注释、命名或历史遗留
- 把局部问题扩成跨切片整治
- 借用户没限制的空档做个人偏好的架构调整
- 在评审阶段夹带功能修改

例外只允许在以下同时满足时发生：

- 与当前 authority 修复直接相关
- 不扩大 task class
- 有直接验证
- closeout 明确列出

## 5. 禁止 TODO 泄漏

完成态代码不得新增以下内容作为未交付占位：

- 无责任人的 `TODO`
- 无退出条件的 `FIXME`
- “后续再补测试 / 审计 / 权限”的占位注释
- 伪实现、空分支、静默 fallback

若必须保留临时标记，至少满足：

- 使用统一格式
- 写明责任边界
- 写明清理触发条件
- 本次改动已具备最小可接受行为

## 6. Closeout 最低要求

AI 参与的任务 closeout 必须包含：

- 变更摘要
- 风险摘要
- 验证结果
- 回滚思路

推荐结构：

```text
AI closeout:
- summary: <改了什么>
- risk: <主要风险或 not-applicable>
- validation: <运行的命令或未运行原因>
- rollback: <如何回退或 not-applicable>
```

如果存在兼容桥接、批量操作、安全影响或多 agent 协作，closeout 还应补充对应专项证据。

## 7. 多 Agent 协作规则

多 agent 协作时，主 agent 负责：

- 提供启动收据与 inherited context package
- 分配不重叠 owned scope
- 汇总验证与最终 closeout
- 拒绝把未知来源改动打包进同一结论

子 agent 必须：

- 只在分配范围内工作
- 发现 authority 越界时立即上报
- 不假设其他 agent 已经处理测试、迁移、注释或回滚
- 输出可供主 agent 复核的差异、风险与验证

## 8. Review 清单

### 8.1 单 Agent Review

- authority owner 是否确认清楚
- 是否严格留在 owned scope
- 是否存在机会主义修复
- 是否新增 TODO 泄漏、伪实现或静默 fallback
- closeout 是否给出 summary / risk / validation / rollback

### 8.2 多 Agent Review

- 各 agent 的 owned scope 是否明确且不重叠
- 是否有人越界修改上游 authority 或共享契约
- 汇总结果是否遗漏冲突、风险或未验证项
- 主 agent 是否明确列出每个子切片的验证状态
- 是否把“另一位 agent 会处理”当成未完成工作的借口

## 9. 证据要求

AI 生成、修改或 review 任务的 closeout 至少记录：

```text
ai review evidence:
- authority_owner: confirmed | escalated | unknown
- scope_discipline: clean | mixed | escalated
- opportunistic_fix: none | included-with-justification
- todo_leakage: none | temporary-with-expiry
- validation: <command or reason>
- rollback: documented | not-applicable
- multi_agent: no | yes
```

若为多 agent 任务，另补：

```text
multi-agent evidence:
- slices: <count>
- overlap: none | detected
- inherited_context: complete | incomplete
- integration_validation: done | partial | not-run
```

## 10. 适合进入 CI 的规则

适合结构化检查或 PR 模板门禁的规则：

- closeout 包含 summary / risk / validation / rollback 字段
- 禁止新增裸 `TODO` / `FIXME`
- 禁止无授权的大范围 rename、锁文件刷新或依赖升级
- 多 agent 任务必须声明 slices 与验证状态

更适合留在文档 / review 的规则：

- 这次跨模块重构是否真的必要
- owned scope 是否划分合理
- 回滚方案是否足够现实
- 机会主义修复是否真的与 authority 修复强相关
