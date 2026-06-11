-- Copyright (c) 2025-2026 GeWuYou
-- SPDX-License-Identifier: Apache-2.0

ALTER TABLE "scheduler_task_runs"
  ADD COLUMN IF NOT EXISTS "task_name_key" character varying NOT NULL DEFAULT '';

COMMENT ON COLUMN "scheduler_task_runs"."task_name_key" IS '运行任务名称国际化键，来源于 Job Definition 标题键';
