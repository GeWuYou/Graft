package auditopenapi

//go:generate go tool oapi-codegen --include-operation-ids getAuditLogs,postAuditLogDeletion,getAuditLogSavedViews,postAuditLogSavedView,putAuditLogSavedView,deleteAuditLogSavedView,getAuditLogDetail,getAuditIncident,getAuditVisibilityPolicy,putAuditVisibilityPolicy,putAuditVisibilityOverride,putAuditVisibilityOverridesBatch,deleteAuditVisibilityOverride --generate types --package auditopenapi -o zz_generated.audit.go ../../../../../openapi/openapi.yaml
