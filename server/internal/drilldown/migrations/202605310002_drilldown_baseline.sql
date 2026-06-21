CREATE TABLE "system_drilldown_scope" (
  "id" BIGSERIAL PRIMARY KEY,
  "module" VARCHAR(64) NOT NULL,
  "scope" VARCHAR(128) NOT NULL,
  "name" VARCHAR(255) NOT NULL,
  "description" TEXT NOT NULL DEFAULT '',
  "target_type" VARCHAR(64) NOT NULL,
  "target_module" VARCHAR(64) NOT NULL,
  "target_page" VARCHAR(128) NOT NULL,
  "filter_payload" JSONB NULL,
  "enabled" BOOLEAN NOT NULL DEFAULT TRUE,
  "sort_order" INTEGER NOT NULL DEFAULT 0,
  "created_at" TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  "updated_at" TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX "uq_system_drilldown_scope_module_scope"
  ON "system_drilldown_scope" ("module", "scope");
CREATE INDEX "idx_system_drilldown_scope_module_enabled"
  ON "system_drilldown_scope" ("module", "enabled");
CREATE INDEX "idx_system_drilldown_scope_target"
  ON "system_drilldown_scope" ("target_module", "target_page");

COMMENT ON TABLE "system_drilldown_scope" IS '系统钻取范围元数据注册表';
COMMENT ON COLUMN "system_drilldown_scope"."id" IS '系统钻取范围主键';
COMMENT ON COLUMN "system_drilldown_scope"."module" IS '声明钻取范围的模块标识';
COMMENT ON COLUMN "system_drilldown_scope"."scope" IS '模块内稳定钻取范围标识';
COMMENT ON COLUMN "system_drilldown_scope"."name" IS '钻取范围展示名称';
COMMENT ON COLUMN "system_drilldown_scope"."description" IS '钻取范围业务说明';
COMMENT ON COLUMN "system_drilldown_scope"."target_type" IS '钻取目标类型，用于区分页面或模块内资源';
COMMENT ON COLUMN "system_drilldown_scope"."target_module" IS '钻取目标所属模块标识';
COMMENT ON COLUMN "system_drilldown_scope"."target_page" IS '钻取目标页面标识';
COMMENT ON COLUMN "system_drilldown_scope"."filter_payload" IS '预留钻取筛选载荷 JSON，当前不作为查询权威';
COMMENT ON COLUMN "system_drilldown_scope"."enabled" IS '是否启用钻取范围，true 表示可用，false 表示停用';
COMMENT ON COLUMN "system_drilldown_scope"."sort_order" IS '钻取范围展示排序值，数值越小越靠前';
COMMENT ON COLUMN "system_drilldown_scope"."created_at" IS '钻取范围注册记录创建时间';
COMMENT ON COLUMN "system_drilldown_scope"."updated_at" IS '钻取范围注册记录更新时间';

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
  ('audit', 'failed_operations', '失败操作', '用于安全审计概览与日志页之间的业务钻取语义。', 'log_query', 'audit', 'audit_logs', NULL, TRUE, 110),
  ('audit', 'high_risk_operations', '高风险操作', '用于安全审计概览与日志页之间的业务钻取语义。', 'log_query', 'audit', 'audit_logs', NULL, TRUE, 120),
  ('audit', 'sensitive_operations', '敏感操作', '用于安全审计概览与日志页之间的业务钻取语义。', 'log_query', 'audit', 'audit_logs', NULL, TRUE, 130),
  ('audit', 'auth_failures', '认证失败', '用于安全审计概览与日志页之间的业务钻取语义。', 'log_query', 'audit', 'audit_logs', NULL, TRUE, 140),
  ('audit', 'permission_denials', '权限拒绝', '用于安全审计概览与日志页之间的业务钻取语义。', 'log_query', 'audit', 'audit_logs', NULL, TRUE, 150),
  ('audit', 'rbac_changes', '权限配置变更', '用于安全审计概览与日志页之间的业务钻取语义。', 'log_query', 'audit', 'audit_logs', NULL, TRUE, 160),
  ('audit', 'critical_security', '关键安全事件', '用于安全审计概览与日志页之间的业务钻取语义。', 'log_query', 'audit', 'audit_logs', NULL, TRUE, 170);
