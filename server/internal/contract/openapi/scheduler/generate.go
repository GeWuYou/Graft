package scheduleropenapi

//go:generate go tool oapi-codegen --include-operation-ids getScheduledTasks,getScheduledTask,getScheduledTaskRuns,postScheduledTaskRun --generate types --package scheduleropenapi -o zz_generated.scheduler.go ../../../../../openapi/openapi.yaml
