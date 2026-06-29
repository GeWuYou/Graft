package scheduler

import (
	"database/sql"
	"time"

	"graft/server/internal/cronx"
)

func scanTaskRun(scanner rowScanner) (TaskRun, error) {
	var run TaskRun
	var id int64
	var triggerType string
	var status string
	var jobCategory string
	var resultSummary string
	var resultJSON string
	var errorMessage string
	var finishedAt sql.NullTime
	var durationMS sql.NullInt64
	if err := scanner.Scan(
		&id,
		&run.TaskKey,
		&run.JobKey,
		&run.TaskTitle,
		&run.TaskTitleKey,
		&run.JobTitle,
		&run.JobTitleKey,
		&run.JobShortTitle,
		&run.JobShortTitleKey,
		&jobCategory,
		&run.ModuleKey,
		&run.TaskBuiltin,
		&triggerType,
		&status,
		&resultSummary,
		&resultJSON,
		&errorMessage,
		&run.StartedAt,
		&finishedAt,
		&durationMS,
		&run.CreatedAt,
	); err != nil {
		return TaskRun{}, err
	}
	runID, err := taskRunIDFromSQL(id)
	if err != nil {
		return TaskRun{}, err
	}
	run.ID = runID
	run.TriggerType = TriggerType(triggerType)
	run.Status = RunStatus(status)
	run.JobCategory = cronx.JobCategory(jobCategory)
	run.Result = resultSummary
	run.ResultJSON = defaultJSONObject(resultJSON)
	run.ErrorMessage = errorMessage
	if finishedAt.Valid {
		finished := finishedAt.Time
		run.FinishedAt = &finished
	}
	if durationMS.Valid {
		duration := durationMS.Int64
		run.DurationMS = &duration
	}
	return run, nil
}

func scanTaskDefinition(scanner rowScanner) (TaskDefinition, error) {
	var task TaskDefinition
	var id int64
	var deletedAt sql.NullInt64
	if err := scanner.Scan(
		&id,
		&task.TaskKey,
		&task.JobKey,
		&task.TitleKey,
		&task.Title,
		&task.DescriptionKey,
		&task.Description,
		&task.CronExpression,
		&task.Enabled,
		&task.Builtin,
		&task.ConfigJSON,
		&task.ConfigSource,
		&task.CreatedAt,
		&task.UpdatedAt,
		&deletedAt,
	); err != nil {
		return TaskDefinition{}, err
	}
	taskID, err := taskRunIDFromSQL(id)
	if err != nil {
		return TaskDefinition{}, err
	}
	task.ID = taskID
	if task.ConfigSource == "" {
		if task.Builtin {
			task.ConfigSource = taskConfigSourceSystem
		} else {
			task.ConfigSource = taskConfigSourceUser
		}
	}
	if deletedAt.Valid && deletedAt.Int64 > 0 {
		deleted := time.Unix(deletedAt.Int64, 0).UTC()
		task.DeletedAt = &deleted
	}
	return task, nil
}

func scanJobDefinition(scanner rowScanner) (JobDefinition, error) {
	var definition JobDefinition
	var id int64
	var category string
	var deletedAt sql.NullInt64
	if err := scanner.Scan(
		&id,
		&definition.JobKey,
		&definition.ModuleKey,
		&category,
		&definition.TitleKey,
		&definition.Title,
		&definition.ShortTitleKey,
		&definition.ShortTitle,
		&definition.DescriptionKey,
		&definition.Description,
		&definition.ConfigSchema,
		&definition.DefaultConfig,
		&definition.DefaultCron,
		&definition.DefaultEnabled,
		&definition.Enabled,
		&definition.CreatedAt,
		&definition.UpdatedAt,
		&deletedAt,
	); err != nil {
		return JobDefinition{}, err
	}
	definitionID, err := taskRunIDFromSQL(id)
	if err != nil {
		return JobDefinition{}, err
	}
	definition.ID = definitionID
	definition.Category = cronx.JobCategory(category)
	if deletedAt.Valid && deletedAt.Int64 > 0 {
		deleted := time.Unix(deletedAt.Int64, 0).UTC()
		definition.DeletedAt = &deleted
	}
	return definition, nil
}
