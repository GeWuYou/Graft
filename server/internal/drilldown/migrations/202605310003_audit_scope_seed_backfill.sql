-- Copyright (c) 2025-2026 GeWuYou
-- SPDX-License-Identifier: Apache-2.0

INSERT INTO "system_drilldown_scope" (
  "module",
  "scope",
  "name",
  "description",
  "target_type",
  "target_module",
  "target_page",
  "filter_payload",
  "enabled",
  "sort_order"
) VALUES
  (
    'audit',
    'failed_operations',
    '失败操作',
    '用于安全审计概览与日志页之间的业务钻取语义。',
    'log_query',
    'audit',
    'audit_logs',
    NULL,
    TRUE,
    110
  ),
  (
    'audit',
    'high_risk_operations',
    '高风险操作',
    '用于安全审计概览与日志页之间的业务钻取语义。',
    'log_query',
    'audit',
    'audit_logs',
    NULL,
    TRUE,
    120
  ),
  (
    'audit',
    'sensitive_operations',
    '敏感操作',
    '用于安全审计概览与日志页之间的业务钻取语义。',
    'log_query',
    'audit',
    'audit_logs',
    NULL,
    TRUE,
    130
  ),
  (
    'audit',
    'auth_failures',
    '认证失败',
    '用于安全审计概览与日志页之间的业务钻取语义。',
    'log_query',
    'audit',
    'audit_logs',
    NULL,
    TRUE,
    140
  ),
  (
    'audit',
    'permission_denials',
    '权限拒绝',
    '用于安全审计概览与日志页之间的业务钻取语义。',
    'log_query',
    'audit',
    'audit_logs',
    NULL,
    TRUE,
    150
  ),
  (
    'audit',
    'rbac_changes',
    '权限配置变更',
    '用于安全审计概览与日志页之间的业务钻取语义。',
    'log_query',
    'audit',
    'audit_logs',
    NULL,
    TRUE,
    160
  ),
  (
    'audit',
    'critical_security',
    '关键安全事件',
    '用于安全审计概览与日志页之间的业务钻取语义。',
    'log_query',
    'audit',
    'audit_logs',
    NULL,
    TRUE,
    170
  )
ON CONFLICT ("module", "scope") DO UPDATE SET
  "name" = EXCLUDED."name",
  "description" = EXCLUDED."description",
  "target_type" = EXCLUDED."target_type",
  "target_module" = EXCLUDED."target_module",
  "target_page" = EXCLUDED."target_page",
  "enabled" = EXCLUDED."enabled",
  "sort_order" = EXCLUDED."sort_order",
  "updated_at" = NOW();
