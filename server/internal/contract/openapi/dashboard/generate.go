package dashboardopenapi

//go:generate go tool oapi-codegen --include-operation-ids getDashboardSummary,getDashboardWidget --generate types --package dashboardopenapi -o zz_generated.dashboard.go ../../../../../openapi/openapi.yaml
