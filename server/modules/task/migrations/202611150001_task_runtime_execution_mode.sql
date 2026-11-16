CREATE TABLE task_runtime_execution_mode (
  id SMALLINT PRIMARY KEY CHECK (id = 1),
  mode TEXT NOT NULL CHECK (mode IN ('local', 'agent')),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);
COMMENT ON TABLE task_runtime_execution_mode IS '任务运行时执行模式绑定，切换前必须收敛已有工作';
COMMENT ON COLUMN task_runtime_execution_mode.id IS '全库唯一的运行时模式绑定标识';
COMMENT ON COLUMN task_runtime_execution_mode.mode IS '有限副作用由本机任务执行器或独立代理执行';
COMMENT ON COLUMN task_runtime_execution_mode.updated_at IS '执行模式最近完成受控切换的时间';
INSERT INTO task_runtime_execution_mode (id, mode) VALUES (1, 'agent');
