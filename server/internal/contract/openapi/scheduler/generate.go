package scheduleropenapi

//go:generate go tool oapi-codegen --include-operation-ids getScheduledTasks,postScheduledTask,getScheduledTask,postScheduledTaskUpdate,postScheduledTaskDelete,postScheduledTaskEnable,postScheduledTaskDisable,getScheduledTaskRuns,getScheduledTaskRun,postScheduledTaskRun --generate types --package scheduleropenapi -o zz_generated.scheduler.go ../../../../../openapi/openapi.yaml
