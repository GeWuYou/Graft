package monitor

//go:generate go tool oapi-codegen --include-operation-ids getMonitorServerStatus,getMonitorRequestPerformance --generate types --package monitor -o zz_generated.types.go ../../../../../openapi/openapi.yaml
