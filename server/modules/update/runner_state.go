package update

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

// RunnerStateRoot 是官方 Compose 状态卷在 runner 和 server 中的固定挂载点。
// runner 是唯一写入者；server 只读取该目录并将终态投影到业务历史。
const RunnerStateRoot = "/var/lib/graft/update-state"

const runnerStateSchemaVersion = 1
const runnerStateEventSchemaVersion = 1
const maxRunnerStateEventReplay = 100

const (
	runnerStateServerUID                       = 10001
	runnerStateServerGID                       = 10001
	runnerStateDirectoryPermission os.FileMode = 0o755
	runnerStateFilePermission      os.FileMode = 0o600
)

// RunnerPhase 是 runner 控制面唯一的活动升级阶段枚举。
type RunnerPhase string

//nolint:revive // 同一枚举块的文档集中说明全部 runner 阶段，避免逐项重复相同语义。
const (
	// RunnerPhaseReady 到 RunnerPhaseRollback 是 runner 控制面稳定阶段值。
	RunnerPhaseReady         RunnerPhase = "READY"
	RunnerPhasePreflight     RunnerPhase = "PREFLIGHT"
	RunnerPhaseBackup        RunnerPhase = "BACKUP"
	RunnerPhasePullImages    RunnerPhase = "PULL_IMAGES"
	RunnerPhaseStopServices  RunnerPhase = "STOP_SERVICES"
	RunnerPhaseApplyUpdate   RunnerPhase = "APPLY_UPDATE"
	RunnerPhaseMigration     RunnerPhase = "MIGRATION"
	RunnerPhaseStartServices RunnerPhase = "START_SERVICES"
	RunnerPhaseHealthCheck   RunnerPhase = "HEALTH_CHECK"
	RunnerPhaseSuccess       RunnerPhase = "SUCCESS"
	RunnerPhaseFailed        RunnerPhase = "FAILED"
	RunnerPhaseRollback      RunnerPhase = "ROLLBACK"
)

// RunnerState 是状态卷中唯一的当前操作快照。它不包含命令输出、宿主机路径、凭证或备份内容。
type RunnerState struct {
	SchemaVersion int            `json:"schema_version"`
	OperationID   string         `json:"operation_id"`
	Operation     string         `json:"operation"`
	RunnerID      string         `json:"runner_id"`
	SourceVersion string         `json:"source_version"`
	TargetVersion string         `json:"target_version"`
	Strategy      string         `json:"deployment_strategy"`
	Phase         RunnerPhase    `json:"phase"`
	Progress      int            `json:"progress"`
	Message       string         `json:"message"`
	StartedAt     time.Time      `json:"started_at"`
	UpdatedAt     time.Time      `json:"updated_at"`
	FinishedAt    *time.Time     `json:"finished_at,omitempty"`
	Error         string         `json:"error,omitempty"`
	Revision      uint64         `json:"revision"`
	Digest        string         `json:"digest"`
	Receipt       *RunnerReceipt `json:"receipt,omitempty"`
}

// RunnerStateStore 持久化由 runner 独占的活动升级状态。
type RunnerStateStore interface {
	Read() (RunnerState, error)
	Write(RunnerState) error
}

// RunnerStateEventReader 读取按 operation 隔离的受控节点事件；事件是重连回放事实，不能由 Docker 日志替代。
type RunnerStateEventReader interface {
	ReadEvents(operationID string, afterRevision uint64, limit int) ([]RunnerOperationEvent, error)
}

// RunnerOperationEvent 是可回放的升级节点记录，只保留 allowlisted 阶段和消息码。
type RunnerOperationEvent struct {
	OperationID string      `json:"operation_id"`
	Revision    uint64      `json:"revision"`
	Phase       RunnerPhase `json:"phase"`
	Message     string      `json:"message"`
	OccurredAt  time.Time   `json:"occurred_at"`
}

type runnerStateEventRecord struct {
	SchemaVersion int `json:"schema_version"`
	RunnerOperationEvent
	Digest string `json:"digest"`
}

// FileRunnerStateStore 原子替换当前快照，并追加可校验关联的事件记录。
type FileRunnerStateStore struct {
	root             string
	enforceOwnership bool
	renameFile       func(string, string) error
	chown            func(string, int, int) error
	mu               sync.Mutex
}

// NewFileRunnerStateStore 创建状态卷适配器。仅官方 RunnerStateRoot 强制 runner 写入后归属 server 运行用户；其他绝对目录仅用于本地测试。
func NewFileRunnerStateStore(root string) (*FileRunnerStateStore, error) {
	root = filepath.Clean(strings.TrimSpace(root))
	if !filepath.IsAbs(root) {
		return nil, errors.New("runner state root must be absolute")
	}
	return &FileRunnerStateStore{root: root, enforceOwnership: root == RunnerStateRoot, renameFile: os.Rename, chown: os.Chown}, nil
}

// Read 读取并校验最近一次原子写入的快照。
func (s *FileRunnerStateStore) Read() (RunnerState, error) {
	if s == nil {
		return RunnerState{}, errors.New("runner state store is unavailable")
	}
	contents, err := os.ReadFile(filepath.Join(s.root, "current.json"))
	if err != nil {
		return RunnerState{}, err
	}
	var state RunnerState
	if err := json.Unmarshal(contents, &state); err != nil {
		return RunnerState{}, fmt.Errorf("decode runner state: %w", err)
	}
	if err := validateRunnerState(state); err != nil {
		return RunnerState{}, err
	}
	digest := state.Digest
	state.Digest = ""
	payload, err := json.Marshal(state)
	if err != nil {
		return RunnerState{}, fmt.Errorf("encode runner state integrity: %w", err)
	}
	want := sha256.Sum256(payload)
	if digest != hex.EncodeToString(want[:]) {
		return RunnerState{}, errors.New("runner state integrity check failed")
	}
	state.Digest = digest
	return state, nil
}

// Write 仅供一次性 runner 调用；它拒绝阶段倒退并持久化快照与事件。
//
//nolint:cyclop,gocyclo,gocognit // 状态完整性、互斥和原子发布是同一个不可拆分的写入事务。
func (s *FileRunnerStateStore) Write(next RunnerState) error {
	if s == nil {
		return errors.New("runner state store is unavailable")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := validateRunnerState(next); err != nil {
		return err
	}
	// 目录保持 runner 所有且可遍历，单个 JSON 文件才转交给 server 用户；这样只需 CAP_CHOWN 仍可持续发布后续 revision。
	if err := s.prepareRunnerWritableDirectory(s.root, "root"); err != nil {
		return err
	}
	eventsRoot := filepath.Join(s.root, "events")
	if err := os.MkdirAll(eventsRoot, runnerStateDirectoryPermission); err != nil {
		return fmt.Errorf("create runner state directory: %w", err)
	}
	if err := s.prepareRunnerWritableDirectory(eventsRoot, "event"); err != nil {
		return err
	}
	eventDirectory := filepath.Join(eventsRoot, next.OperationID)
	if err := os.MkdirAll(eventDirectory, runnerStateDirectoryPermission); err != nil {
		return fmt.Errorf("create runner operation event directory: %w", err)
	}
	if err := s.prepareRunnerWritableDirectory(eventDirectory, "operation event"); err != nil {
		return err
	}
	previous, err := s.Read()
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	if err == nil {
		if previous.OperationID != next.OperationID && !isTerminalRunnerPhase(previous.Phase) {
			return errors.New("another runner operation is active")
		}
		if previous.OperationID == next.OperationID && (next.Revision <= previous.Revision || phaseOrdinal(next.Phase) < phaseOrdinal(previous.Phase)) {
			return errors.New("runner state transition is invalid")
		}
	}
	next.Digest = ""
	payload, err := json.Marshal(next)
	if err != nil {
		return fmt.Errorf("encode runner state: %w", err)
	}
	digest := sha256.Sum256(payload)
	next.Digest = hex.EncodeToString(digest[:])
	payload, err = json.Marshal(next)
	if err != nil {
		return fmt.Errorf("encode runner state with digest: %w", err)
	}
	temporary := filepath.Join(s.root, ".current.json.tmp")
	if err := os.WriteFile(temporary, payload, runnerStateFilePermission); err != nil {
		return fmt.Errorf("write runner state: %w", err)
	}
	if err := s.assignServerOwnership(temporary, "state"); err != nil {
		return fmt.Errorf("assign runner state owner: %w", err)
	}
	event, err := marshalRunnerStateEvent(newRunnerOperationEvent(next))
	if err != nil {
		return err
	}
	event = append(event, '\n')
	eventPath := filepath.Join(eventDirectory, fmt.Sprintf("%020d.json", next.Revision))
	eventTemporary := eventPath + ".tmp"
	if err := os.WriteFile(eventTemporary, event, runnerStateFilePermission); err != nil {
		return fmt.Errorf("write runner state event: %w", err)
	}
	if err := s.assignServerOwnership(eventTemporary, "event"); err != nil {
		return fmt.Errorf("assign runner state event owner: %w", err)
	}
	if err := s.renameFile(eventTemporary, eventPath); err != nil {
		return fmt.Errorf("publish runner state event: %w", err)
	}
	if err := s.renameFile(temporary, filepath.Join(s.root, "current.json")); err != nil {
		return fmt.Errorf("publish runner state: %w", err)
	}
	return nil
}

// ReadEvents 返回 revision 严格大于游标的已校验事件，供 HTTP 回放和 realtime 断线补偿共用。
//
//nolint:cyclop,gocognit,gocyclo // 目录边界、文件名、operation 绑定和完整性验证必须依次成立，不能把受控日志退化为原始文件遍历。
func (s *FileRunnerStateStore) ReadEvents(operationID string, afterRevision uint64, limit int) ([]RunnerOperationEvent, error) {
	if s == nil || !runnerOperationID.MatchString(operationID) || limit < 1 || limit > maxRunnerStateEventReplay {
		return nil, errors.New("runner state event query is invalid")
	}
	directory := filepath.Join(s.root, "events", operationID)
	entries, err := os.ReadDir(directory)
	if errors.Is(err, os.ErrNotExist) {
		return []RunnerOperationEvent{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read runner state events: %w", err)
	}
	events := make([]RunnerOperationEvent, 0, min(limit, len(entries)))
	for _, entry := range entries {
		if entry.IsDir() || strings.HasSuffix(entry.Name(), ".tmp") {
			continue
		}
		revision, ok := parseRunnerStateEventRevision(entry.Name())
		if !ok || revision <= afterRevision {
			continue
		}
		// #nosec G304 -- entry.Name 已经通过固定宽度 revision 文件名校验，目录由安全 operation ID 构成。
		contents, readErr := os.ReadFile(filepath.Join(directory, entry.Name()))
		if readErr != nil {
			return nil, fmt.Errorf("read runner state event: %w", readErr)
		}
		event, decodeErr := unmarshalRunnerStateEvent(contents)
		if decodeErr != nil {
			return nil, decodeErr
		}
		if event.OperationID != operationID || event.Revision != revision {
			return nil, errors.New("runner state event binding is invalid")
		}
		events = append(events, event)
		if len(events) == limit {
			break
		}
	}
	return events, nil
}

func (s *FileRunnerStateStore) assignServerOwnership(path, kind string) error {
	if s == nil || !s.enforceOwnership {
		return nil
	}
	if err := s.chown(path, runnerStateServerUID, runnerStateServerGID); err != nil {
		return fmt.Errorf("assign runner state %s owner: %w", kind, err)
	}
	return nil
}

func (s *FileRunnerStateStore) prepareRunnerWritableDirectory(path, kind string) error {
	if s == nil || !s.enforceOwnership {
		return nil
	}
	if err := s.chown(path, 0, 0); err != nil {
		return fmt.Errorf("assign runner state %s directory owner: %w", kind, err)
	}
	if err := os.Chmod(path, runnerStateDirectoryPermission); err != nil {
		return fmt.Errorf("assign runner state %s directory permission: %w", kind, err)
	}
	return nil
}

func newRunnerOperationEvent(state RunnerState) RunnerOperationEvent {
	return RunnerOperationEvent{OperationID: state.OperationID, Revision: state.Revision, Phase: state.Phase, Message: state.Message, OccurredAt: state.UpdatedAt}
}

func marshalRunnerStateEvent(event RunnerOperationEvent) ([]byte, error) {
	record := runnerStateEventRecord{SchemaVersion: runnerStateEventSchemaVersion, RunnerOperationEvent: event}
	if err := validateRunnerStateEvent(record); err != nil {
		return nil, err
	}
	payload, err := json.Marshal(record)
	if err != nil {
		return nil, fmt.Errorf("encode runner state event: %w", err)
	}
	digest := sha256.Sum256(payload)
	record.Digest = hex.EncodeToString(digest[:])
	payload, err = json.Marshal(record)
	if err != nil {
		return nil, fmt.Errorf("encode runner state event with digest: %w", err)
	}
	return payload, nil
}

func unmarshalRunnerStateEvent(contents []byte) (RunnerOperationEvent, error) {
	var record runnerStateEventRecord
	if err := json.Unmarshal(contents, &record); err != nil {
		return RunnerOperationEvent{}, fmt.Errorf("decode runner state event: %w", err)
	}
	if err := validateRunnerStateEvent(record); err != nil {
		return RunnerOperationEvent{}, err
	}
	digest := record.Digest
	record.Digest = ""
	payload, err := json.Marshal(record)
	if err != nil {
		return RunnerOperationEvent{}, fmt.Errorf("encode runner state event integrity: %w", err)
	}
	want := sha256.Sum256(payload)
	if digest != hex.EncodeToString(want[:]) {
		return RunnerOperationEvent{}, errors.New("runner state event integrity check failed")
	}
	return record.RunnerOperationEvent, nil
}

func validateRunnerStateEvent(record runnerStateEventRecord) error {
	event := record.RunnerOperationEvent
	if record.SchemaVersion != runnerStateEventSchemaVersion || !runnerOperationID.MatchString(event.OperationID) || event.Revision == 0 || !validRunnerPhase(event.Phase) || !validRunnerStateMessage(event.Message) || event.OccurredAt.IsZero() {
		return errors.New("runner state event is invalid")
	}
	return nil
}

func parseRunnerStateEventRevision(name string) (uint64, bool) {
	if len(name) != len("00000000000000000000.json") || !strings.HasSuffix(name, ".json") {
		return 0, false
	}
	revision, err := strconv.ParseUint(strings.TrimSuffix(name, ".json"), 10, 64)
	return revision, err == nil && revision > 0
}

// NewRunnerState 返回具有单调 revision 的新状态转换。
//
//nolint:revive // 状态转换同时需要冻结输入、执行器身份、阶段、进度、受控消息、错误和上一个快照。
func NewRunnerState(input RunnerInput, runnerID string, phase RunnerPhase, progress int, message, failure string, previous RunnerState) RunnerState {
	now := time.Now().UTC()
	state := RunnerState{SchemaVersion: runnerStateSchemaVersion, OperationID: input.OperationID, Operation: "self_update", RunnerID: runnerID, SourceVersion: input.SourceVersion, TargetVersion: input.TargetVersion, Strategy: string(input.Preflight.DeploymentStrategy), Phase: phase, Progress: progress, Message: message, Error: failure, StartedAt: now, UpdatedAt: now, Revision: 1}
	if previous.OperationID == input.OperationID {
		state.StartedAt, state.Revision = previous.StartedAt, previous.Revision+1
		state.SourceVersion, state.TargetVersion, state.Strategy = previous.SourceVersion, previous.TargetVersion, previous.Strategy
	}
	if isTerminalRunnerPhase(phase) {
		state.FinishedAt = &now
	}
	return state
}

//nolint:cyclop,gocyclo // 单个快照的全部不变量必须在读取边界一次性验证。
func validateRunnerState(value RunnerState) error {
	if value.SchemaVersion != runnerStateSchemaVersion || !runnerOperationID.MatchString(value.OperationID) || !runnerOperationID.MatchString(value.RunnerID) || strings.TrimSpace(value.SourceVersion) == "" || strings.TrimSpace(value.TargetVersion) == "" || !validDeploymentStrategy(DeploymentStrategy(value.Strategy)) || value.Operation != "self_update" || !validRunnerPhase(value.Phase) || !validRunnerStateMessage(value.Message) || !validRunnerStateFailure(value.Error) || value.Progress < 0 || value.Progress > 100 || value.Revision == 0 || value.StartedAt.IsZero() || value.UpdatedAt.IsZero() {
		return errors.New("runner state is invalid")
	}
	if isTerminalRunnerPhase(value.Phase) != (value.FinishedAt != nil) {
		return errors.New("runner state terminal timestamp is invalid")
	}
	return nil
}

func validRunnerStateMessage(message string) bool {
	switch message {
	case "runner_starting", "runner_accepted", "checking_environment", "creating_backup", "pulling_images", "verifying_images", "stopping_services", "applying_update", "running_migrations", "starting_services", "checking_health", "update_completed", "update_failed", "rollback_completed":
		return true
	default:
		return false
	}
}

func validRunnerStateFailure(failure string) bool {
	switch failure {
	case "", runnerFailureInvalidInput, runnerFailureBackup, runnerFailurePull, runnerFailureImageVerify, runnerFailureStopServices, runnerFailureMigration, runnerFailureRecreate, runnerFailureDockerHealth, runnerFailureHealthz:
		return true
	default:
		return false
	}
}

func validRunnerPhase(value RunnerPhase) bool { return phaseOrdinal(value) >= 0 }

func isTerminalRunnerPhase(value RunnerPhase) bool {
	return value == RunnerPhaseSuccess || value == RunnerPhaseFailed || value == RunnerPhaseRollback
}

func phaseOrdinal(value RunnerPhase) int {
	for index, phase := range []RunnerPhase{RunnerPhaseReady, RunnerPhasePreflight, RunnerPhaseBackup, RunnerPhasePullImages, RunnerPhaseStopServices, RunnerPhaseApplyUpdate, RunnerPhaseMigration, RunnerPhaseStartServices, RunnerPhaseHealthCheck, RunnerPhaseSuccess, RunnerPhaseFailed, RunnerPhaseRollback} {
		if value == phase {
			return index
		}
	}
	return -1
}
