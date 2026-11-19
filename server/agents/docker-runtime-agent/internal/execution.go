package agent

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"graft/server/internal/dockerexecution"
	"graft/server/internal/moduleapi"
)

const (
	executionClaimPath      = "/agent/v1/execution-leases/claim"
	executionRenewPath      = "/agent/v1/execution-leases/"
	executionLogsSuffix     = "/logs"
	executionMaterialSuffix = "/material"
	executionResultsSuffix  = "/results"
	executionReceiptsSuffix = "/receipts"
	executionRenewalDivisor = 2
)

var errExecutionNoWork = errors.New("no execution lease available")
var errAgentIdentityRejected = errors.New("agent identity rejected")

type executionLease struct {
	ID                    string          `json:"lease_id"`
	TaskID                uint64          `json:"task_id"`
	StageID               uint64          `json:"stage_id"`
	Attempt               int             `json:"attempt"`
	ExecutorType          string          `json:"executor_type"`
	RuntimeTargetID       int64           `json:"runtime_target_id"`
	ProviderID            string          `json:"provider_id"`
	Capability            string          `json:"capability"`
	CapabilityVersion     string          `json:"capability_version"`
	Protocol              string          `json:"protocol"`
	OperationID           string          `json:"operation_id"`
	PayloadSHA256         string          `json:"payload_sha256"`
	Input                 json.RawMessage `json:"input"`
	FenceToken            string          `json:"fence_token"`
	State                 string          `json:"state"`
	LeaseTTLMS            int64           `json:"lease_ttl_ms"`
	LeaseExpiresAt        time.Time       `json:"lease_expires_at"`
	AbsoluteDeadlineAt    time.Time       `json:"absolute_deadline_at"`
	CancellationRequested bool            `json:"cancellation_requested"`
}

type executionLogEntry struct {
	Stream string `json:"stream"`
	Level  string `json:"level"`
	Line   string `json:"line"`
}

type executionReceipt struct {
	FenceToken        string `json:"fence_token"`
	Outcome           string `json:"outcome"`
	FailureCode       string `json:"failure_code,omitempty"`
	FailureStage      string `json:"failure_stage,omitempty"`
	FailureClass      string `json:"failure_class,omitempty"`
	DiagnosticSummary string `json:"diagnostic_summary,omitempty"`
	IntegritySHA256   string `json:"integrity_sha256"`
}

type executionSettlement struct {
	TaskID     uint64 `json:"TaskID"`
	StageID    uint64 `json:"StageID"`
	Status     string `json:"Status"`
	Idempotent bool   `json:"Idempotent"`
}

type executionMaterial = dockerexecution.Material
type executionResult = dockerexecution.Result

type executionOperation func(context.Context, executionLease) executionResult

func claimExecutionLease(ctx context.Context, client *http.Client, agentURL string, providerID, capability, capabilityVersion string) (executionLease, error) {
	var lease executionLease
	status, err := requestJSON(ctx, client, http.MethodPost, strings.TrimRight(agentURL, "/")+executionClaimPath, executionClaimRequest{ProviderID: providerID, Capability: capability, CapabilityVersion: capabilityVersion}, &lease)
	if err != nil {
		return executionLease{}, &claimAdmissionError{class: claimAdmissionTransportFailed, err: err}
	}
	if status == http.StatusNoContent {
		return executionLease{}, &claimAdmissionError{class: claimAdmissionNoWork, status: status, err: errExecutionNoWork}
	}
	if status == http.StatusBadRequest || status == http.StatusUnauthorized || status == http.StatusForbidden {
		return executionLease{}, &claimAdmissionError{class: claimAdmissionIdentityRejected, status: status, err: errAgentIdentityRejected}
	}
	if status != http.StatusOK || strings.TrimSpace(lease.ID) == "" {
		return executionLease{}, &claimAdmissionError{class: claimAdmissionTransportFailed, status: status, err: errors.New("agent execution claim rejected")}
	}
	return lease, nil
}

type executionClaimRequest struct {
	ProviderID        string `json:"provider_id"`
	Capability        string `json:"capability"`
	CapabilityVersion string `json:"capability_version"`
}

func renewExecutionLease(ctx context.Context, client *http.Client, agentURL string, lease executionLease) (executionLease, error) {
	var renewed executionLease
	status, err := requestJSON(ctx, client, http.MethodPost, strings.TrimRight(agentURL, "/")+executionRenewPath+lease.ID+"/renew", executionHandleRequest{FenceToken: lease.FenceToken}, &renewed)
	if err != nil {
		return executionLease{}, err
	}
	if status == http.StatusUnauthorized || status == http.StatusForbidden {
		return executionLease{}, errAgentIdentityRejected
	}
	if status != http.StatusOK {
		return executionLease{}, errors.New("agent execution renewal rejected")
	}
	return renewed, nil
}

func appendExecutionLogs(ctx context.Context, client *http.Client, agentURL string, lease executionLease, entries []executionLogEntry) error {
	status, err := requestJSON(ctx, client, http.MethodPost, strings.TrimRight(agentURL, "/")+executionRenewPath+lease.ID+executionLogsSuffix, executionLogsRequest{FenceToken: lease.FenceToken, Entries: entries}, nil)
	if err != nil {
		return err
	}
	if status == http.StatusUnauthorized || status == http.StatusForbidden {
		return errAgentIdentityRejected
	}
	if status != http.StatusNoContent {
		return errors.New("agent execution log rejected")
	}
	return nil
}

func resolveExecutionMaterial(ctx context.Context, client *http.Client, agentURL string, lease executionLease) (executionMaterial, error) {
	var material executionMaterial
	status, err := requestJSON(ctx, client, http.MethodPost, strings.TrimRight(agentURL, "/")+executionRenewPath+lease.ID+executionMaterialSuffix, executionHandleRequest{FenceToken: lease.FenceToken}, &material)
	if err != nil {
		return executionMaterial{}, err
	}
	if status == http.StatusUnauthorized || status == http.StatusForbidden {
		return executionMaterial{}, errAgentIdentityRejected
	}
	if status != http.StatusOK || strings.TrimSpace(material.Protocol) == "" || len(material.Payload) == 0 {
		return executionMaterial{}, errors.New("agent execution material rejected")
	}
	return material, nil
}

type executionResultRequest struct {
	FenceToken string          `json:"fence_token"`
	Protocol   string          `json:"protocol"`
	Payload    json.RawMessage `json:"payload"`
}

func submitExecutionResult(ctx context.Context, client *http.Client, agentURL string, lease executionLease, result executionResult) error {
	request := executionResultRequest{FenceToken: lease.FenceToken, Protocol: result.Protocol, Payload: result.Payload}
	status, err := requestJSON(ctx, client, http.MethodPost, strings.TrimRight(agentURL, "/")+executionRenewPath+lease.ID+executionResultsSuffix, request, nil)
	if err != nil {
		return err
	}
	if status == http.StatusUnauthorized || status == http.StatusForbidden {
		return errAgentIdentityRejected
	}
	if status != http.StatusNoContent {
		return errors.New("agent execution result rejected")
	}
	return nil
}

func settleExecution(ctx context.Context, client *http.Client, agentURL string, journal executionJournal) (executionSettlement, error) {
	var settlement executionSettlement
	receipt := executionReceipt{FenceToken: journal.Lease.FenceToken, Outcome: journal.Outcome, FailureCode: journal.FailureCode, FailureStage: journal.FailureStage, FailureClass: journal.FailureClass, DiagnosticSummary: journal.FailureSummary, IntegritySHA256: leaseIntegrity(journal.Lease)}
	status, err := requestJSON(ctx, client, http.MethodPost, strings.TrimRight(agentURL, "/")+executionRenewPath+journal.Lease.ID+executionReceiptsSuffix, receipt, &settlement)
	if err != nil {
		return executionSettlement{}, err
	}
	if status == http.StatusUnauthorized || status == http.StatusForbidden {
		return executionSettlement{}, errAgentIdentityRejected
	}
	if status != http.StatusOK {
		return executionSettlement{}, errors.New("agent execution receipt rejected")
	}
	return settlement, nil
}

type executionHandleRequest struct {
	FenceToken string `json:"fence_token"`
}
type executionLogsRequest struct {
	FenceToken string              `json:"fence_token"`
	Entries    []executionLogEntry `json:"entries"`
}

func executeLease(ctx context.Context, client *http.Client, c config, lease executionLease) error {
	return executeLeaseWithOperation(ctx, client, c, lease, func(operationCtx context.Context, operationLease executionLease) executionResult {
		return executeProviderOperation(operationCtx, client, c, operationLease)
	})
}

func executeLeaseWithOperation(ctx context.Context, client *http.Client, c config, lease executionLease, operation executionOperation) error {
	renewed, err := renewExecutionLease(ctx, client, c.AgentURL, lease)
	if err != nil {
		return err
	}
	lease = renewed
	journal, terminal, err := prepareExecutionJournal(c.StateDir, lease)
	if err != nil {
		return err
	}
	if terminal {
		return replayTerminalExecution(ctx, client, c, journal)
	}
	if err := appendExecutionLogs(ctx, client, c.AgentURL, lease, []executionLogEntry{{Stream: "system", Level: "info", Line: "execution lease accepted"}}); err != nil {
		return err
	}
	if lease.CancellationRequested {
		return settleObservedCancellation(ctx, client, c, lease)
	}
	journal.Lease = lease
	journal.Phase = executionJournalRunning
	if err := saveExecutionJournal(c.StateDir, journal); err != nil {
		return err
	}
	if err := appendExecutionLogs(ctx, client, c.AgentURL, lease, []executionLogEntry{{Stream: "system", Level: "info", Line: "provider operation started"}}); err != nil {
		return err
	}
	return waitForProviderOperation(ctx, client, c, lease, journal, operation)
}

func prepareExecutionJournal(stateDir string, lease executionLease) (executionJournal, bool, error) {
	journal, exists, err := executionJournalForLease(stateDir, lease.ID)
	if err != nil {
		return executionJournal{}, false, err
	}
	if !exists {
		journal = executionJournal{Lease: lease, Phase: executionJournalPrepared}
		return journal, false, saveExecutionJournal(stateDir, journal)
	}
	journal.Lease = lease
	if journal.Phase == executionJournalTerminal {
		return journal, true, nil
	}
	if journal.Phase != executionJournalRunning {
		return journal, false, nil
	}
	journal.Phase, journal.Outcome, journal.FailureCode, journal.FailureClass = executionJournalTerminal, "needs_attention", failureInterrupted, string(moduleapi.ExecutionFailureClassUnknown)
	return journal, true, saveExecutionJournal(stateDir, journal)
}

func waitForProviderOperation(ctx context.Context, client *http.Client, c config, lease executionLease, journal executionJournal, operation executionOperation) error {
	operationCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	resultCh := make(chan executionResult, 1)
	operationLease := lease
	go func() { resultCh <- operation(operationCtx, operationLease) }()
	ticker := time.NewTicker(executionRenewInterval(lease))
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case result := <-resultCh:
			return completeProviderOperation(ctx, client, c, lease, journal, result)
		case <-ticker.C:
			renewed, err := renewExecutionLease(ctx, client, c.AgentURL, lease)
			if err != nil {
				cancel()
				return err
			}
			lease = renewed
			if lease.CancellationRequested {
				cancel()
				return settleObservedCancellation(ctx, client, c, lease)
			}
		}
	}
}

func completeProviderOperation(ctx context.Context, client *http.Client, c config, lease executionLease, journal executionJournal, result executionResult) error {
	if result.Outcome == "success" && lease.Capability == buildExecutionCapability {
		if result.Protocol != buildExecutionResultProtocol || len(result.Payload) == 0 {
			return errors.New("provider execution result is unavailable")
		}
		// Build 结果只在有效 fence 下瞬时交付；未交付成功前不得把 journal 写成可重放的终态。
		if err := submitExecutionResult(ctx, client, c.AgentURL, lease, result); err != nil {
			return err
		}
	}
	journal.Lease, journal.Phase, journal.Outcome, journal.FailureCode = lease, executionJournalTerminal, result.Outcome, result.FailureCode
	journal.FailureStage, journal.FailureClass, journal.FailureSummary = result.FailureStage, string(result.FailureClass), result.FailureSummary
	if err := saveExecutionJournal(c.StateDir, journal); err != nil {
		return err
	}
	if err := appendExecutionLogs(ctx, client, c.AgentURL, lease, []executionLogEntry{{Stream: "system", Level: "info", Line: "provider operation completed"}}); err != nil {
		return err
	}
	return replayTerminalExecution(ctx, client, c, journal)
}

func settleObservedCancellation(ctx context.Context, client *http.Client, c config, lease executionLease) error {
	if err := appendExecutionLogs(ctx, client, c.AgentURL, lease, []executionLogEntry{{Stream: "system", Level: "info", Line: "cancellation observed"}}); err != nil {
		return err
	}
	journal := executionJournal{Lease: lease, Phase: executionJournalTerminal, Outcome: "success"}
	if err := saveExecutionJournal(c.StateDir, journal); err != nil {
		return err
	}
	return replayTerminalExecution(ctx, client, c, journal)
}

func replayTerminalExecution(ctx context.Context, client *http.Client, c config, journal executionJournal) error {
	if _, err := settleExecution(ctx, client, c.AgentURL, journal); err != nil {
		return err
	}
	return removeExecutionJournal(c.StateDir, journal.Lease.ID)
}

func executionJournalForLease(stateDir, leaseID string) (executionJournal, bool, error) {
	journals, err := loadExecutionJournals(stateDir)
	if err != nil {
		return executionJournal{}, false, err
	}
	for _, journal := range journals {
		if journal.Lease.ID == leaseID {
			return journal, true, nil
		}
	}
	return executionJournal{}, false, nil
}

func executionRenewInterval(lease executionLease) time.Duration {
	interval := time.Duration(lease.LeaseTTLMS) * time.Millisecond / executionRenewalDivisor
	if interval <= 0 {
		interval = time.Second
	}
	return interval
}

func leaseIntegrity(lease executionLease) string {
	digest := sha256.Sum256([]byte(strings.Join([]string{lease.OperationID, lease.PayloadSHA256, lease.Protocol}, "\n")))
	return hex.EncodeToString(digest[:])
}

func requestJSON(ctx context.Context, client *http.Client, method, endpoint string, body any, destination any) (int, error) {
	var reader io.Reader
	if body != nil {
		payload, err := json.Marshal(body)
		if err != nil {
			return 0, errors.New("encode agent transport request")
		}
		reader = strings.NewReader(string(payload))
	}
	request, err := http.NewRequestWithContext(ctx, method, endpoint, reader)
	if err != nil {
		return 0, errors.New("create agent transport request")
	}
	request.Header.Set("Content-Type", "application/json")
	response, err := client.Do(request)
	if err != nil {
		return 0, errors.New("agent transport request failed")
	}
	defer closeResponse(response.Body)
	if destination != nil && response.StatusCode == http.StatusOK {
		if err := json.NewDecoder(io.LimitReader(response.Body, maxResponseBytes)).Decode(destination); err != nil {
			return response.StatusCode, errors.New("decode agent transport response")
		}
	}
	return response.StatusCode, nil
}

func runExecutionLoop(ctx context.Context, c config, state persistedState, tracker *admissionTracker) error {
	for {
		state = runExecutionCycle(ctx, c, state, tracker)
		if err := waitExecutionPoll(ctx, c.PollInterval); err != nil {
			return err
		}
	}
}

func runExecutionCycle(ctx context.Context, c config, state persistedState, tracker *admissionTracker) persistedState {
	refreshed, ready := ensureExecutionIdentity(ctx, c, state, tracker)
	if !ready {
		return state
	}
	client, err := newMTLSClient(c, refreshed)
	if err != nil {
		return refreshed
	}
	if err := recoverExecutionJournals(ctx, client, c); err != nil {
		if errors.Is(err, errAgentIdentityRejected) {
			return persistedState{}
		}
		return refreshed
	}
	return pollExecutionCapabilities(ctx, client, c, refreshed, tracker)
}

func ensureExecutionIdentity(ctx context.Context, c config, state persistedState, tracker *admissionTracker) (persistedState, bool) {
	if state.CertificatePEM != "" && !state.ExpiresAt.Before(time.Now().UTC().Add(c.RenewBefore)) {
		return state, true
	}
	refreshed, err := enrollAndReconnect(ctx, c, tracker)
	if err != nil {
		return state, false
	}
	return refreshed, true
}

type claimResult struct {
	lease executionLease
	err   error
}

func pollExecutionCapabilities(ctx context.Context, client *http.Client, c config, state persistedState, tracker *admissionTracker) persistedState {
	results := make(chan claimResult, len(c.Capabilities))
	var claims sync.WaitGroup
	for _, capability := range c.Capabilities {
		claims.Add(1)
		go func(capability string) {
			defer claims.Done()
			lease, err := claimExecutionLease(ctx, client, c.AgentURL, c.ProviderID, capability, c.CapabilityVersion)
			results <- claimResult{lease: lease, err: err}
		}(capability)
	}
	go func() {
		claims.Wait()
		close(results)
	}()
	outcome := collectClaimResults(ctx, client, c, results)
	// 拒绝或传输失败优先记录；无工作不算失败，只要本轮证明过领取成功就清空遗留结论。
	// 领取结论必须先于执行等待落盘，长时间执行的租约不得延迟本地 admission 证据。
	if claimAdmissionRank(outcome.cycle.Class) >= claimAdmissionBlockedRank || !outcome.admitted {
		tracker.observe(outcome.cycle.Class, outcome.cycle.HTTPStatus)
	} else {
		tracker.clear()
	}
	reportAdmissionTelemetry(ctx, client, c, tracker)
	outcome.executions.Wait()
	if outcome.identityRejected.Load() {
		return persistedState{}
	}
	return state
}

// claimCycleOutcome 汇合同一轮的领取事实；executions 在调用方持久化 admission 结论后才等待。
type claimCycleOutcome struct {
	cycle            claimAdmission
	admitted         bool
	identityRejected *atomic.Bool
	executions       *sync.WaitGroup
}

// collectClaimResults 汇合同一轮的并发领取结论，并启动已领取租约的执行。
// 结论按拒绝 > 传输失败 > 无工作合并，避免后到的 204 掩盖已被拒绝的能力。
func collectClaimResults(ctx context.Context, client *http.Client, c config, results <-chan claimResult) claimCycleOutcome {
	var executions sync.WaitGroup
	var identityRejected atomic.Bool
	outcome := claimCycleOutcome{identityRejected: &identityRejected, executions: &executions}
	for result := range results {
		var admission *claimAdmissionError
		if errors.As(result.err, &admission) && claimAdmissionRank(admission.class) > claimAdmissionRank(outcome.cycle.Class) {
			outcome.cycle = claimAdmission{Class: admission.class, HTTPStatus: admission.status}
		}
		if errors.Is(result.err, errAgentIdentityRejected) {
			identityRejected.Store(true)
			continue
		}
		if result.err != nil {
			continue
		}
		outcome.admitted = true
		executions.Add(1)
		go func(lease executionLease) {
			defer executions.Done()
			if errors.Is(executeLease(ctx, client, c, lease), errAgentIdentityRejected) {
				identityRejected.Store(true)
			}
		}(result.lease)
	}
	return outcome
}

const (
	claimAdmissionRankUnset            = 0
	claimAdmissionRankNoWork           = 1
	claimAdmissionRankTransportFailed  = 2
	claimAdmissionRankIdentityRejected = 3
)

// claimAdmissionBlockedRank 标记会阻断领取的结论等级；无工作不属于阻断状态。
const claimAdmissionBlockedRank = claimAdmissionRankTransportFailed

// claimAdmissionRank 决定同一轮并发领取结论的合并优先级。
func claimAdmissionRank(class claimAdmissionClass) int {
	switch class {
	case claimAdmissionIdentityRejected:
		return claimAdmissionRankIdentityRejected
	case claimAdmissionTransportFailed:
		return claimAdmissionRankTransportFailed
	case claimAdmissionNoWork:
		return claimAdmissionRankNoWork
	default:
		return claimAdmissionRankUnset
	}
}

// reportAdmissionTelemetry 在领取结论变化时提交一次受限遥测；
// 去重与最小上报间隔保证 admission 状态不会跟随轮询节奏刷屏，
// 提交成功才推进去重状态，失败时下一轮按间隔重试。
func reportAdmissionTelemetry(ctx context.Context, client *http.Client, c config, tracker *admissionTracker) {
	if tracker == nil {
		return
	}
	admission, report := tracker.shouldReport()
	if !report {
		return
	}
	snapshot, err := fetchLedgerSnapshot(ctx, client, c.AgentURL)
	if err != nil {
		return
	}
	if err := submitTelemetry(ctx, client, c.AgentURL, snapshot, string(admission.Class)); err != nil {
		return
	}
	tracker.commitReport(admission)
}

func recoverExecutionJournals(ctx context.Context, client *http.Client, c config) error {
	journals, err := loadExecutionJournals(c.StateDir)
	if err != nil {
		return err
	}
	for _, journal := range journals {
		if err := executeLease(ctx, client, c, journal.Lease); err != nil {
			return err
		}
	}
	return nil
}

func waitExecutionPoll(ctx context.Context, interval time.Duration) error {
	timer := time.NewTimer(interval)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func healthcheck(configPath string) error {
	c, err := loadConfig(configPath)
	if err != nil {
		return err
	}
	state, err := loadState(c.StateDir)
	if !runtimeAgentIdentityReady(c, state, err) {
		return fmt.Errorf("runtime agent is not ready")
	}
	if !stateTrustBundleMatches(c.TrustBundle, c.StateDir) {
		return fmt.Errorf("runtime agent trust bundle is stale")
	}
	return reportHealthcheckAdmission(c, time.Now().UTC())
}

// runtimeAgentIdentityReady 判断本地证书身份是否满足 readiness 基线。
func runtimeAgentIdentityReady(c config, state persistedState, loadErr error) bool {
	if loadErr != nil || state.CertificatePEM == "" || state.Generation < 1 {
		return false
	}
	return state.TargetID == c.TargetID && state.AgentID == c.AgentID &&
		state.Identity == stableSPIFFE(c.TargetID, c.AgentID) &&
		!state.ExpiresAt.Before(time.Now().UTC().Add(c.RenewBefore))
}

// reportHealthcheckAdmission 把领取结论写入 readiness 结论；持续身份拒绝使 healthcheck 失败。
func reportHealthcheckAdmission(c config, now time.Time) error {
	admission, ready, err := admissionReadiness(c.StateDir, now)
	if err != nil {
		return fmt.Errorf("runtime agent admission state is unreadable")
	}
	if !ready {
		return fmt.Errorf("runtime agent claim admission is %s", describeClaimAdmission(admission))
	}
	if admission.Class != "" {
		_, _ = fmt.Fprintf(os.Stdout, "graft-docker-runtime-agent: last claim admission %s\n", describeClaimAdmission(admission))
	}
	return nil
}

// describeClaimAdmission 生成单行、无路径与凭据的 admission 可观察描述。
func describeClaimAdmission(admission claimAdmission) string {
	if admission.HTTPStatus > 0 {
		return fmt.Sprintf("%s (HTTP %d at %s)", admission.Class, admission.HTTPStatus, admission.At.UTC().Format(time.RFC3339))
	}
	if admission.At.IsZero() {
		return string(admission.Class)
	}
	return fmt.Sprintf("%s (at %s)", admission.Class, admission.At.UTC().Format(time.RFC3339))
}
