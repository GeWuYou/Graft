package update

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"graft/server/internal/moduleapi"
)

// LegacyCutoverPurgeMarker is written only by the beta schema-v1 cutover.
// The next forward migration deletes rows carrying this marker and their
// operation-linked diagnostics; ordinary runner failures are never selected.
const LegacyCutoverPurgeMarker = "PLATFORM_UPDATE_LEGACY_CUTOVER"

type legacyCutoverOperation struct {
	operationID string
	taskID      uint64
}

// CutoverV1 validates and removes a legacy state volume before SQL migration.
// It deliberately preserves Task and Backup facts by cancelling them through
// their owning capabilities before marking the Update operation for purge.
//
//nolint:cyclop,gocognit,gocyclo,nestif // cutover 必须在一个 fail-closed 流程中完成状态判定、事实保留和文件清理。
func CutoverV1(ctx context.Context, root string, db *sql.DB, tasks moduleapi.TaskService, backups moduleapi.BackupService) error {
	if ctx == nil {
		ctx = context.Background()
	}
	root = filepath.Clean(strings.TrimSpace(root))
	if root == "." || !filepath.IsAbs(root) {
		return errors.New("update state root must be absolute")
	}
	currentPath := filepath.Join(root, "current.json")
	contents, err := os.ReadFile(currentPath)
	if errors.Is(err, os.ErrNotExist) {
		if removeErr := os.RemoveAll(filepath.Join(root, "events")); removeErr != nil {
			return fmt.Errorf("remove legacy update events: %w", removeErr)
		}
		return nil
	}
	if err != nil {
		return fmt.Errorf("read update state snapshot: %w", err)
	}
	var raw struct {
		SchemaVersion int         `json:"schema_version"`
		OperationID   string      `json:"operation_id"`
		RunnerID      string      `json:"runner_id"`
		Operation     string      `json:"operation"`
		Phase         RunnerPhase `json:"phase"`
		Progress      int         `json:"progress"`
		SourceVersion string      `json:"source_version"`
		TargetVersion string      `json:"target_version"`
		Strategy      string      `json:"deployment_strategy"`
		Revision      uint64      `json:"revision"`
	}
	if err := json.Unmarshal(contents, &raw); err != nil {
		return fmt.Errorf("decode legacy update state: %w", err)
	}
	if raw.SchemaVersion != 1 {
		if raw.SchemaVersion == runnerStateSchemaVersion {
			store, storeErr := NewFileRunnerStateStore(root)
			if storeErr != nil {
				return fmt.Errorf("prepare schema-v2 state validation: %w", storeErr)
			}
			if _, storeErr = store.Read(); storeErr != nil {
				return fmt.Errorf("validate schema-v2 update state: %w", storeErr)
			}
			return nil
		}
		return errors.New("unsupported update state schema version")
	}
	if !runnerOperationID.MatchString(raw.OperationID) || !runnerOperationID.MatchString(raw.RunnerID) || raw.Operation != "self_update" || raw.Progress < 0 || raw.Progress > 100 || raw.Revision == 0 || !validRunnerPhase(raw.Phase) || strings.TrimSpace(raw.SourceVersion) == "" || strings.TrimSpace(raw.TargetVersion) == "" || !validDeploymentStrategy(DeploymentStrategy(raw.Strategy)) {
		return errors.New("legacy update state is unsafe")
	}
	if db == nil || tasks == nil || backups == nil {
		return errors.New("legacy update cutover dependencies are unavailable")
	}
	rows, err := db.QueryContext(ctx, `SELECT operation_id, task_id FROM update_operations WHERE operation_id = $1 AND finished_at IS NULL`, raw.OperationID)
	if err != nil {
		return fmt.Errorf("list active update operations for cutover: %w", err)
	}
	operations := make([]legacyCutoverOperation, 0)
	for rows.Next() {
		var operation legacyCutoverOperation
		if err := rows.Scan(&operation.operationID, &operation.taskID); err != nil {
			_ = rows.Close()
			return fmt.Errorf("scan active update operation for cutover: %w", err)
		}
		operations = append(operations, operation)
	}
	if err := rows.Err(); err != nil {
		_ = rows.Close()
		return fmt.Errorf("iterate active update operations for cutover: %w", err)
	}
	_ = rows.Close()
	for _, operation := range operations {
		if err := tasks.Cancel(ctx, operation.taskID); err != nil {
			return fmt.Errorf("cancel legacy update task %d: %w", operation.taskID, err)
		}
		if err := backups.CancelRunnerHandoff(ctx, operation.operationID, operation.taskID); err != nil {
			return fmt.Errorf("cancel legacy backup handoff %s: %w", operation.operationID, err)
		}
		if _, err := db.ExecContext(ctx, `UPDATE update_operations SET failure_code = $1, updated_at = CURRENT_TIMESTAMP WHERE operation_id = $2 AND finished_at IS NULL`, LegacyCutoverPurgeMarker, operation.operationID); err != nil {
			return fmt.Errorf("mark legacy update operation %s for purge: %w", operation.operationID, err)
		}
	}
	if err := os.Remove(currentPath); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("remove legacy update snapshot: %w", err)
	}
	if err := os.RemoveAll(filepath.Join(root, "events")); err != nil {
		return fmt.Errorf("remove legacy update events: %w", err)
	}
	return nil
}
