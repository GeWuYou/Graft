package scheduleropenapi

//go:generate go tool oapi-codegen --include-operation-ids getScheduledTasks,getScheduledTaskSavedViews,postScheduledTaskSavedView,putScheduledTaskSavedView,deleteScheduledTaskSavedView,getScheduledTaskJobDefinitions,getScheduledTaskJobDefinition,postScheduledTask,getScheduledTask,putScheduledTask,deleteScheduledTask,postScheduledTaskEnable,postScheduledTaskDisable,getScheduledTaskRuns,getScheduledTaskRun,postScheduledTaskRun,postScheduledTaskAction --generate types --package scheduleropenapi -o zz_generated.scheduler.go ../../../../../openapi/openapi.yaml
