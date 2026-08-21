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
	"strings"
	"time"
)

const (
	executionClaimPath      = "/agent/v1/execution-leases/claim"
	executionRenewPath      = "/agent/v1/execution-leases/"
	executionLogsSuffix     = "/logs"
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
	FenceToken      string `json:"fence_token"`
	Outcome         string `json:"outcome"`
	FailureCode     string `json:"failure_code,omitempty"`
	IntegritySHA256 string `json:"integrity_sha256"`
}

type executionSettlement struct {
	TaskID     uint64 `json:"TaskID"`
	StageID    uint64 `json:"StageID"`
	Status     string `json:"Status"`
	Idempotent bool   `json:"Idempotent"`
}

type executionResult struct {
	Outcome     string
	FailureCode string
}

type executionOperation func(context.Context, executionLease) executionResult

func claimExecutionLease(ctx context.Context, client *http.Client, agentURL string, providerID, capability, capabilityVersion string) (executionLease, error) {
	var lease executionLease
	status, err := requestJSON(ctx, client, http.MethodPost, strings.TrimRight(agentURL, "/")+executionClaimPath, executionClaimRequest{ProviderID: providerID, Capability: capability, CapabilityVersion: capabilityVersion}, &lease)
	if err != nil {
		return executionLease{}, err
	}
	if status == http.StatusNoContent {
		return executionLease{}, errExecutionNoWork
	}
	if status == http.StatusUnauthorized || status == http.StatusForbidden {
		return executionLease{}, errAgentIdentityRejected
	}
	if status != http.StatusOK || strings.TrimSpace(lease.ID) == "" {
		return executionLease{}, errors.New("agent execution claim rejected")
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

func settleExecution(ctx context.Context, client *http.Client, agentURL string, lease executionLease, outcome, failureCode string) (executionSettlement, error) {
	var settlement executionSettlement
	status, err := requestJSON(ctx, client, http.MethodPost, strings.TrimRight(agentURL, "/")+executionRenewPath+lease.ID+executionReceiptsSuffix, executionReceipt{FenceToken: lease.FenceToken, Outcome: outcome, FailureCode: failureCode, IntegritySHA256: leaseIntegrity(lease)}, &settlement)
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
	return executeLeaseWithOperation(ctx, client, c, lease, func(context.Context, executionLease) executionResult {
		// Batch 3 只升格 transport 与 fencing；Provider Docker 操作由后续迁移批次接入。
		return executionResult{Outcome: "needs_attention", FailureCode: "provider_execution_not_migrated"}
	})
}

func executeLeaseWithOperation(ctx context.Context, client *http.Client, c config, lease executionLease, operation executionOperation) error {
	renewed, err := renewExecutionLease(ctx, client, c.AgentURL, lease)
	if err != nil {
		return err
	}
	lease = renewed
	if err := appendExecutionLogs(ctx, client, c.AgentURL, lease, []executionLogEntry{{Stream: "agent", Level: "info", Line: "execution lease accepted"}}); err != nil {
		return err
	}
	if lease.CancellationRequested {
		return settleObservedCancellation(ctx, client, c, lease)
	}
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
			_, err := settleExecution(ctx, client, c.AgentURL, lease, result.Outcome, result.FailureCode)
			return err
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

func settleObservedCancellation(ctx context.Context, client *http.Client, c config, lease executionLease) error {
	if err := appendExecutionLogs(ctx, client, c.AgentURL, lease, []executionLogEntry{{Stream: "agent", Level: "info", Line: "cancellation observed"}}); err != nil {
		return err
	}
	_, err := settleExecution(ctx, client, c.AgentURL, lease, "success", "")
	return err
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

func runExecutionLoop(ctx context.Context, c config, state persistedState) error {
	for {
		var ready bool
		state, ready = ensureExecutionIdentity(ctx, c, state)
		if !ready {
			if err := waitExecutionPoll(ctx, c.PollInterval); err != nil {
				return err
			}
			continue
		}
		client, err := newMTLSClient(c, state)
		if err != nil {
			if waitErr := waitExecutionPoll(ctx, c.PollInterval); waitErr != nil {
				return waitErr
			}
			continue
		}
		state = pollExecutionCapabilities(ctx, client, c, state)
		if err := waitExecutionPoll(ctx, c.PollInterval); err != nil {
			return err
		}
	}
}

func ensureExecutionIdentity(ctx context.Context, c config, state persistedState) (persistedState, bool) {
	if state.CertificatePEM != "" && !state.ExpiresAt.Before(time.Now().UTC().Add(c.RenewBefore)) {
		return state, true
	}
	refreshed, err := enrollAndReconnect(ctx, c)
	if err != nil {
		return state, false
	}
	return refreshed, true
}

func pollExecutionCapabilities(ctx context.Context, client *http.Client, c config, state persistedState) persistedState {
	for _, capability := range c.Capabilities {
		lease, err := claimExecutionLease(ctx, client, c.AgentURL, c.ProviderID, capability, c.CapabilityVersion)
		if errors.Is(err, errExecutionNoWork) {
			continue
		}
		if errors.Is(err, errAgentIdentityRejected) {
			return persistedState{}
		}
		if err != nil {
			return state
		}
		if err := executeLease(ctx, client, c, lease); err != nil {
			if errors.Is(err, errAgentIdentityRejected) {
				return persistedState{}
			}
			return state
		}
	}
	return state
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
	if err != nil || state.CertificatePEM == "" || state.Generation < 1 || state.TargetID != c.TargetID || state.AgentID != c.AgentID || state.Identity != stableSPIFFE(c.TargetID, c.AgentID) || state.ExpiresAt.Before(time.Now().UTC().Add(c.RenewBefore)) {
		return fmt.Errorf("runtime agent is not ready")
	}
	if !stateTrustBundleMatches(c.TrustBundle, c.StateDir) {
		return fmt.Errorf("runtime agent trust bundle is stale")
	}
	return nil
}
