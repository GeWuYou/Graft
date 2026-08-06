# Task 提交生命周期实施计划

## 目标

将 Task 创建前的 Reservation 重构为独立的 TaskSubmission 聚合，使每个 Submission 最终可物化为 Task 或以 discarded/expired 终结，且 Worker 永远不消费未物化提交。

## 分批路线

1. `submission-authority-and-contract`
   - 设计、ADR、状态 wire contract、SubmissionPolicy 与通用 Materialize capability。
2. `submission-persistence-and-owner-lock`
   - Submission migration、owner advisory-lock protocol、统一 version CAS、expiry recovery、Task ready state。
3. `build-submission-materializer`
   - Build Snapshot writer、单库原子物化、幂等/网络失败测试。
4. `legacy-reservation-migration-and-rollout`
   - `activation_required` legacy adapter、grace expiry、feature rollout 和 rollback guard。
5. `cross-boundary-projection-and-closeout`
   - OpenAPI/generated/web projection、全链路验证、兼容桥清理条件。

## Acceptance

- 任意 Submission 最终成为 activated、discarded 或 expired，不存在永久 reserved。
- Worker 仅领取 `ready` Task。
- Snapshot、Task、Submission 终态和 owner claim 在本地路径上原子一致。
- 相同幂等请求、Materialize 重试、Discard 重试和 Expirer 并发均无重复 Task 或 claim 泄漏。
