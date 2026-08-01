package update

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// RunnerStateRoot 是官方 Compose 状态卷在 runner 和 server 中的固定挂载点。
// runner 是唯一写入者；server 只读取该目录并将终态投影到业务历史。
const RunnerStateRoot = "/var/lib/graft/update-state"

const runnerStateSchemaVersion = 1

const (
	runnerStateServerUID                       = 10001
	runnerStateServerGID                       = 10001
	runnerStateDirectoryPermission os.FileMode = 0o750
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

// FileRunnerStateStore 原子替换当前快照，并追加可校验关联的事件记录。
type FileRunnerStateStore struct {
	root             string
	enforceOwnership bool
	renameFile       func(string, string) error
	mu               sync.Mutex
}

// NewFileRunnerStateStore 创建状态卷适配器。仅官方 RunnerStateRoot 强制 runner 写入后归属 server 运行用户；其他绝对目录仅用于本地测试。
func NewFileRunnerStateStore(root string) (*FileRunnerStateStore, error) {
	root = filepath.Clean(strings.TrimSpace(root))
	if !filepath.IsAbs(root) {
		return nil, errors.New("runner state root must be absolute")
	}
	return &FileRunnerStateStore{root: root, enforceOwnership: root == RunnerStateRoot, renameFile: os.Rename}, nil
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
	// 官方状态卷不保存秘密，server 以非 root 用户只读挂载，故 runner 写入必须转交给该用户。
	if err := os.MkdirAll(filepath.Join(s.root, "events"), runnerStateDirectoryPermission); err != nil {
		return fmt.Errorf("create runner state directory: %w", err)
	}
	if err := s.assignServerOwnership(s.root, "root"); err != nil {
		return fmt.Errorf("assign runner state root owner: %w", err)
	}
	if err := s.assignServerOwnership(filepath.Join(s.root, "events"), "event"); err != nil {
		return fmt.Errorf("assign runner state event owner: %w", err)
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
	if err := validateRunnerState(next); err != nil {
		return err
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
	event := append(payload, '\n')
	eventPath := filepath.Join(s.root, "events", fmt.Sprintf("%020d.json", next.Revision))
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

func (s *FileRunnerStateStore) assignServerOwnership(path, kind string) error {
	if s == nil || !s.enforceOwnership {
		return nil
	}
	if err := os.Chown(path, runnerStateServerUID, runnerStateServerGID); err != nil {
		return fmt.Errorf("assign runner state %s owner: %w", kind, err)
	}
	return nil
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

//nolint:cyclop // 单个快照的全部不变量必须在读取边界一次性验证。
func validateRunnerState(value RunnerState) error {
	if value.SchemaVersion != runnerStateSchemaVersion || !runnerOperationID.MatchString(value.OperationID) || !runnerOperationID.MatchString(value.RunnerID) || strings.TrimSpace(value.SourceVersion) == "" || strings.TrimSpace(value.TargetVersion) == "" || !validDeploymentStrategy(DeploymentStrategy(value.Strategy)) || value.Operation != "self_update" || !validRunnerPhase(value.Phase) || value.Progress < 0 || value.Progress > 100 || value.Revision == 0 || value.StartedAt.IsZero() || value.UpdatedAt.IsZero() {
		return errors.New("runner state is invalid")
	}
	if isTerminalRunnerPhase(value.Phase) != (value.FinishedAt != nil) {
		return errors.New("runner state terminal timestamp is invalid")
	}
	return nil
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
