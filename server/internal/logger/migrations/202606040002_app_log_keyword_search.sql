-- Copyright (c) 2025-2026 GeWuYou
-- SPDX-License-Identifier: Apache-2.0

CREATE INDEX "idx_app_logs_keyword_search"
ON "app_logs"
USING GIN (
  to_tsvector(
    'simple',
    "component" || ' ' || COALESCE("operation", '') || ' ' || "message" || ' ' || COALESCE("error", '')
  )
);
