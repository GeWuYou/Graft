-- 执行诊断投影：为任务与阶段补充有界失败阶段、失败分类，配合 failure_message 承载脱敏诊断摘要。
-- 新增列全部可空且不带约束；历史回放与未失败的行保持 NULL，不重写任何既有结算事实。
ALTER TABLE tasks
  ADD COLUMN failure_stage varchar(32) NULL,
  ADD COLUMN failure_class varchar(32) NULL;

ALTER TABLE task_stages
  ADD COLUMN failure_stage varchar(32) NULL,
  ADD COLUMN failure_class varchar(32) NULL;

COMMENT ON COLUMN tasks.failure_stage IS '任务失败时的有界执行阶段归属，未归类时为空';
COMMENT ON COLUMN tasks.failure_class IS '任务失败的稳定恢复分类，未归类时为空';
COMMENT ON COLUMN task_stages.failure_stage IS '阶段失败时的有界执行阶段归属，未归类时为空';
COMMENT ON COLUMN task_stages.failure_class IS '阶段失败的稳定恢复分类，未归类时为空';
