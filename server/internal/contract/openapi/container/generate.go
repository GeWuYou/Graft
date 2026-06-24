package containeropenapi

//go:generate go tool oapi-codegen --include-operation-ids getContainers,getContainerDashboardSummary,getContainer,getContainerMountUsage,postContainerMountUsageRefresh,getContainerLogs,postContainerShellSession,getContainerShellWebSocket,postContainerStart,postContainerStop,postContainerRestart,postContainerRemove,postContainerBatchActions --generate types --package containeropenapi -o zz_generated.container.go ../../../../../openapi/openapi.yaml
