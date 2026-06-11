-- Copyright (c) 2025-2026 GeWuYou
-- SPDX-License-Identifier: Apache-2.0

CREATE UNIQUE INDEX IF NOT EXISTS "notification_deliveries_event_recipient"
  ON "notification_deliveries" ("event_id", "recipient_user_id");
