package httpx

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"graft/server/internal/apperror"
	"graft/server/internal/contract/errorcode"
	messagecontract "graft/server/internal/contract/message"
	"graft/server/internal/moduleapi"
)

const (
	agentExecutionClaimPath              = "/agent/v1/execution-leases/claim"
	agentExecutionRenewPath              = "/agent/v1/execution-leases/:leaseID/renew"
	agentExecutionLogsPath               = "/agent/v1/execution-leases/:leaseID/logs"
	agentExecutionMaterialPath           = "/agent/v1/execution-leases/:leaseID/material"
	agentExecutionResultPath             = "/agent/v1/execution-leases/:leaseID/results"
	agentExecutionReceiptPath            = "/agent/v1/execution-leases/:leaseID/receipts"
	maxAgentExecutionRequestBytes  int64 = 1 << 20
	agentExecutionLongPollWindow         = 20 * time.Second
	agentExecutionLongPollInterval       = 500 * time.Millisecond
)

// ConfigureExecutionRoutes 将 Task Runtime 的外部执行 capability 接入 Agent 专用 listener。
// Runtime Target binding reader 只用于把请求中的 capability 限定在证书身份已登记的能力集合内。
func (s *AgentServer) ConfigureExecutionRoutes(gateway moduleapi.RuntimeAgentExecutionGateway, bindings moduleapi.RuntimeTargetAgentBindingReader) error {
	if s == nil || s.engine == nil {
		return errors.New("agent mTLS server is unavailable")
	}
	if gateway == nil || bindings == nil {
		return errors.New("agent execution gateway and binding reader are required")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.server != nil {
		return errors.New("agent mTLS server is already running")
	}
	if s.executionRoutesConfigured {
		return errors.New("agent execution routes are already configured")
	}
	s.engine.POST(agentExecutionClaimPath, agentExecutionClaimHandler(gateway, bindings, s.logger))
	s.engine.POST(agentExecutionRenewPath, agentExecutionRenewHandler(gateway, bindings, s.logger))
	s.engine.POST(agentExecutionLogsPath, agentExecutionLogsHandler(gateway, bindings, s.logger))
	s.engine.POST(agentExecutionMaterialPath, agentExecutionMaterialHandler(gateway, bindings, s.logger))
	s.engine.POST(agentExecutionResultPath, agentExecutionResultHandler(gateway, bindings, s.logger))
	s.engine.POST(agentExecutionReceiptPath, agentExecutionReceiptHandler(gateway, bindings, s.logger))
	s.executionRoutesConfigured = true
	return nil
}

type agentExecutionClaimRequest struct {
	ProviderID        string `json:"provider_id"`
	Capability        string `json:"capability"`
	CapabilityVersion string `json:"capability_version"`
}

type agentExecutionLeaseResponse struct {
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

func agentExecutionClaimHandler(gateway moduleapi.RuntimeAgentExecutionGateway, bindings moduleapi.RuntimeTargetAgentBindingReader, logger *zap.Logger) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		identity, ok := AgentMTLSIdentityFromGinContext(ctx)
		if !ok || agentLedgerHasForbiddenHeaders(ctx) {
			abortAgentLedgerIdentity(ctx, logger)
			return
		}
		var request agentExecutionClaimRequest
		if err := decodeAgentExecutionJSON(ctx, &request); err != nil || !agentCapabilityAllowed(ctx, bindings, identity, request.ProviderID, request.Capability, request.CapabilityVersion) {
			abortInvalidAgentLedgerRequest(ctx, logger)
			return
		}
		lease, err := longPollAgentExecution(ctx.Request.Context(), gateway, moduleapi.ExternalExecutionClaimRequest{RuntimeTargetID: identity.TargetID, ProviderID: strings.TrimSpace(request.ProviderID), Capability: strings.TrimSpace(request.Capability), CapabilityVersion: strings.TrimSpace(request.CapabilityVersion)})
		if errors.Is(err, moduleapi.ErrExternalExecutionNotFound) {
			ctx.Status(http.StatusNoContent)
			return
		}
		if err != nil {
			abortAgentExecutionFailure(ctx, logger)
			return
		}
		ctx.JSON(http.StatusOK, marshalAgentExecutionLease(lease))
	}
}

func longPollAgentExecution(ctx context.Context, gateway moduleapi.RuntimeAgentExecutionGateway, request moduleapi.ExternalExecutionClaimRequest) (moduleapi.ExternalExecutionLease, error) {
	deadline := time.NewTimer(agentExecutionLongPollWindow)
	defer deadline.Stop()
	for {
		lease, err := gateway.ClaimExternalExecution(ctx, request)
		if !errors.Is(err, moduleapi.ErrExternalExecutionNotFound) {
			return lease, err
		}
		poll := time.NewTimer(agentExecutionLongPollInterval)
		select {
		case <-ctx.Done():
			poll.Stop()
			return moduleapi.ExternalExecutionLease{}, ctx.Err()
		case <-deadline.C:
			poll.Stop()
			return moduleapi.ExternalExecutionLease{}, moduleapi.ErrExternalExecutionNotFound
		case <-poll.C:
		}
	}
}

type agentExecutionHandleRequest struct {
	FenceToken string `json:"fence_token"`
}

func agentExecutionRenewHandler(gateway moduleapi.RuntimeAgentExecutionGateway, bindings moduleapi.RuntimeTargetAgentBindingReader, logger *zap.Logger) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		identity, ok := AgentMTLSIdentityFromGinContext(ctx)
		if !ok || !agentIdentityBindingActive(ctx, bindings, identity) || agentLedgerHasForbiddenHeaders(ctx) {
			abortAgentLedgerIdentity(ctx, logger)
			return
		}
		var request agentExecutionHandleRequest
		if err := decodeAgentExecutionJSON(ctx, &request); err != nil || strings.TrimSpace(ctx.Param("leaseID")) == "" {
			abortInvalidAgentLedgerRequest(ctx, logger)
			return
		}
		handle := moduleapi.ExternalExecutionLeaseHandle{LeaseID: ctx.Param("leaseID"), FenceToken: request.FenceToken}
		if !agentExecutionHandleAllowed(ctx, gateway, bindings, identity, handle) {
			abortAgentLedgerIdentity(ctx, logger)
			return
		}
		lease, err := gateway.RenewExternalExecution(ctx.Request.Context(), handle)
		if err != nil {
			abortAgentExecutionFailure(ctx, logger)
			return
		}
		ctx.JSON(http.StatusOK, marshalAgentExecutionLease(lease))
	}
}

type agentExecutionLogsRequest struct {
	FenceToken string                   `json:"fence_token"`
	Entries    []moduleapi.TaskLogEntry `json:"entries"`
}

func agentExecutionLogsHandler(gateway moduleapi.RuntimeAgentExecutionGateway, bindings moduleapi.RuntimeTargetAgentBindingReader, logger *zap.Logger) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		identity, ok := AgentMTLSIdentityFromGinContext(ctx)
		if !ok || !agentIdentityBindingActive(ctx, bindings, identity) || agentLedgerHasForbiddenHeaders(ctx) {
			abortAgentLedgerIdentity(ctx, logger)
			return
		}
		var request agentExecutionLogsRequest
		if err := decodeAgentExecutionJSON(ctx, &request); err != nil {
			abortInvalidAgentLedgerRequest(ctx, logger)
			return
		}
		handle := moduleapi.ExternalExecutionLeaseHandle{LeaseID: ctx.Param("leaseID"), FenceToken: request.FenceToken}
		if !agentExecutionHandleAllowed(ctx, gateway, bindings, identity, handle) {
			abortAgentLedgerIdentity(ctx, logger)
			return
		}
		if err := gateway.AppendExternalExecutionLogs(ctx.Request.Context(), moduleapi.ExternalExecutionLogBatch{Handle: handle, Entries: request.Entries}); err != nil {
			abortAgentExecutionFailure(ctx, logger)
			return
		}
		ctx.Status(http.StatusNoContent)
	}
}

func agentExecutionMaterialHandler(gateway moduleapi.RuntimeAgentExecutionGateway, bindings moduleapi.RuntimeTargetAgentBindingReader, logger *zap.Logger) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		identity, ok := AgentMTLSIdentityFromGinContext(ctx)
		if !ok || !agentIdentityBindingActive(ctx, bindings, identity) || agentLedgerHasForbiddenHeaders(ctx) {
			abortAgentLedgerIdentity(ctx, logger)
			return
		}
		var request agentExecutionHandleRequest
		if err := decodeAgentExecutionJSON(ctx, &request); err != nil || strings.TrimSpace(ctx.Param("leaseID")) == "" {
			abortInvalidAgentLedgerRequest(ctx, logger)
			return
		}
		handle := moduleapi.ExternalExecutionLeaseHandle{LeaseID: ctx.Param("leaseID"), FenceToken: request.FenceToken}
		if !agentExecutionHandleAllowed(ctx, gateway, bindings, identity, handle) {
			abortAgentLedgerIdentity(ctx, logger)
			return
		}
		material, err := gateway.ResolveExternalExecutionMaterial(ctx.Request.Context(), handle)
		if err != nil {
			abortAgentExecutionFailure(ctx, logger)
			return
		}
		ctx.JSON(http.StatusOK, gin.H{"protocol": material.Protocol, "payload": material.Payload})
	}
}

type agentExecutionResultRequest struct {
	FenceToken string          `json:"fence_token"`
	Protocol   string          `json:"protocol"`
	Payload    json.RawMessage `json:"payload"`
}

func agentExecutionResultHandler(gateway moduleapi.RuntimeAgentExecutionGateway, bindings moduleapi.RuntimeTargetAgentBindingReader, logger *zap.Logger) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		identity, ok := AgentMTLSIdentityFromGinContext(ctx)
		if !ok || !agentIdentityBindingActive(ctx, bindings, identity) || agentLedgerHasForbiddenHeaders(ctx) {
			abortAgentLedgerIdentity(ctx, logger)
			return
		}
		var request agentExecutionResultRequest
		if err := decodeAgentExecutionJSON(ctx, &request); err != nil {
			abortInvalidAgentLedgerRequest(ctx, logger)
			return
		}
		handle := moduleapi.ExternalExecutionLeaseHandle{LeaseID: ctx.Param("leaseID"), FenceToken: request.FenceToken}
		if !agentExecutionHandleAllowed(ctx, gateway, bindings, identity, handle) {
			abortAgentLedgerIdentity(ctx, logger)
			return
		}
		if err := gateway.RecordExternalExecutionResult(ctx.Request.Context(), moduleapi.ExternalExecutionResult{Handle: handle, Protocol: strings.TrimSpace(request.Protocol), Payload: request.Payload}); err != nil {
			abortAgentExecutionFailure(ctx, logger)
			return
		}
		ctx.Status(http.StatusNoContent)
	}
}

type agentExecutionReceiptRequest struct {
	FenceToken      string `json:"fence_token"`
	Outcome         string `json:"outcome"`
	FailureCode     string `json:"failure_code"`
	IntegritySHA256 string `json:"integrity_sha256"`
}

func agentExecutionReceiptHandler(gateway moduleapi.RuntimeAgentExecutionGateway, bindings moduleapi.RuntimeTargetAgentBindingReader, logger *zap.Logger) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		identity, ok := AgentMTLSIdentityFromGinContext(ctx)
		if !ok || !agentIdentityBindingActive(ctx, bindings, identity) || agentLedgerHasForbiddenHeaders(ctx) {
			abortAgentLedgerIdentity(ctx, logger)
			return
		}
		var request agentExecutionReceiptRequest
		if err := decodeAgentExecutionJSON(ctx, &request); err != nil {
			abortInvalidAgentLedgerRequest(ctx, logger)
			return
		}
		handle := moduleapi.ExternalExecutionLeaseHandle{LeaseID: ctx.Param("leaseID"), FenceToken: request.FenceToken}
		if !agentExecutionHandleAllowed(ctx, gateway, bindings, identity, handle) {
			abortAgentLedgerIdentity(ctx, logger)
			return
		}
		settlement, err := gateway.SettleExternalExecution(ctx.Request.Context(), moduleapi.ExternalExecutionReceipt{Handle: handle, Outcome: moduleapi.ExternalReceiptOutcome(strings.TrimSpace(request.Outcome)), FailureCode: strings.TrimSpace(request.FailureCode), IntegritySHA256: strings.TrimSpace(request.IntegritySHA256)})
		if err != nil {
			abortAgentExecutionFailure(ctx, logger)
			return
		}
		ctx.JSON(http.StatusOK, settlement)
	}
}

func abortAgentExecutionFailure(ctx *gin.Context, logger *zap.Logger) {
	AbortAppError(ctx, nil, logger, apperror.New(apperror.Descriptor{Kind: apperror.KindInternal, Code: errorcode.CommonInternalError, MessageKey: messagecontract.CommonInternalError}))
}

func decodeAgentExecutionJSON(ctx *gin.Context, destination any) error {
	if ctx == nil || ctx.Request == nil || ctx.Request.Body == nil || !agentLedgerJSONContentType(ctx.GetHeader("Content-Type")) {
		return errors.New("agent execution request content type is invalid")
	}
	ctx.Request.Body = http.MaxBytesReader(ctx.Writer, ctx.Request.Body, maxAgentExecutionRequestBytes)
	decoder := json.NewDecoder(ctx.Request.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(destination); err != nil {
		return err
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return errors.New("agent execution request must contain one JSON value")
	}
	return nil
}

func marshalAgentExecutionLease(lease moduleapi.ExternalExecutionLease) agentExecutionLeaseResponse {
	return agentExecutionLeaseResponse{ID: lease.ID, TaskID: lease.TaskID, StageID: lease.StageID, Attempt: lease.Attempt, ExecutorType: string(lease.ExecutorType), RuntimeTargetID: lease.RuntimeTargetID, ProviderID: lease.ProviderID, Capability: lease.Capability, CapabilityVersion: lease.CapabilityVersion, Protocol: lease.Protocol, OperationID: lease.OperationID, PayloadSHA256: lease.PayloadSHA256, Input: lease.Input, FenceToken: lease.FenceToken, State: string(lease.State), LeaseTTLMS: lease.LeaseTTL.Milliseconds(), LeaseExpiresAt: lease.LeaseExpiresAt.UTC(), AbsoluteDeadlineAt: lease.AbsoluteDeadlineAt.UTC(), CancellationRequested: lease.CancellationRequested}
}

func agentCapabilityAllowed(ctx *gin.Context, bindings moduleapi.RuntimeTargetAgentBindingReader, identity AgentMTLSIdentity, providerID, capability, capabilityVersion string) bool {
	binding, err := bindings.ReadAgentBinding(ctx.Request.Context(), identity.TargetID, identity.AgentID)
	if err != nil || !agentIdentityMatchesBinding(identity, binding) ||
		binding.ProviderID != strings.TrimSpace(providerID) || binding.CapabilityVersion != strings.TrimSpace(capabilityVersion) {
		return false
	}
	for _, candidate := range binding.Capabilities {
		if candidate == strings.TrimSpace(capability) {
			return true
		}
	}
	return false
}

func agentIdentityBindingActive(ctx *gin.Context, bindings moduleapi.RuntimeTargetAgentBindingReader, identity AgentMTLSIdentity) bool {
	binding, err := bindings.ReadAgentBinding(ctx.Request.Context(), identity.TargetID, identity.AgentID)
	return err == nil && agentIdentityMatchesBinding(identity, binding)
}

func agentIdentityMatchesBinding(identity AgentMTLSIdentity, binding moduleapi.RuntimeTargetAgentBinding) bool {
	return binding.Status == moduleapi.RuntimeTargetAgentStatusActive && binding.IdentityID == identity.IdentityID &&
		binding.TargetID == identity.TargetID && binding.AgentID == identity.AgentID && binding.Generation == identity.Generation &&
		binding.CertificateSerial == identity.CertificateSerial && binding.PublicKeyFingerprint == identity.PublicKeyFingerprint
}

func agentExecutionHandleAllowed(ctx *gin.Context, gateway moduleapi.RuntimeAgentExecutionGateway, bindings moduleapi.RuntimeTargetAgentBindingReader, identity AgentMTLSIdentity, handle moduleapi.ExternalExecutionLeaseHandle) bool {
	lease, err := gateway.InspectExternalExecution(ctx.Request.Context(), handle)
	return err == nil && lease.RuntimeTargetID == identity.TargetID &&
		agentCapabilityAllowed(ctx, bindings, identity, lease.ProviderID, lease.Capability, lease.CapabilityVersion)
}
