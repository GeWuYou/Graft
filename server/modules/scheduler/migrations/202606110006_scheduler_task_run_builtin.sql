-- Copyright (c) 2025-2026 GeWuYou
-- SPDX-License-Identifier: Apache-2.0

ALTER TABLE "scheduler_task_runs"
  ADD COLUMN IF NOT EXISTS "task_builtin" boolean NOT NULL DEFAULT false;

COMMENT ON COLUMN "scheduler_task_runs"."task_builtin" IS '运行时任务实例是否为系统内置任务快照';
