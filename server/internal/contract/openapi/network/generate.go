// Package networkopenapi 持有由 canonical OpenAPI 派生的 Network 连通性类型。
package networkopenapi

//go:generate go tool oapi-codegen --include-operation-ids getPlatformConnectivityTargets,getPlatformConnectivityCustomTargets,postPlatformConnectivityCustomTarget,deletePlatformConnectivityCustomTarget,getPlatformConnectivityLatest,getPlatformConnectivityAggregate,postPlatformConnectivityRun,postPlatformConnectivityBatchRun,getPlatformConnectivityHistory,getPlatformConnectivityReport,getPlatformConnectivityTrace,getPlatformConnectivityExport --generate types --package networkopenapi -o zz_generated.connectivity.go ../../../../../openapi/openapi.yaml
