package httpx

import (
	"encoding/json"
	"errors"
	"io"
	"mime"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"graft/server/internal/apperror"
	"graft/server/internal/contract/errorcode"
	"graft/server/internal/contract/httpheader"
	messagecontract "graft/server/internal/contract/message"
	"graft/server/internal/moduleapi"
)

const agentLedgerSnapshotPath = "/agent/v1/ledger-snapshot"
const agentTelemetryReportsPath = "/agent/v1/telemetry-reports"
const maxAgentTelemetryReportBytes int64 = 1 << 20

// ConfigureLedgerRoutes 注册 Agent mTLS listener 的唯一 ledger surface。
// 该方法必须在 listener 启动前调用，且不向主 HTTP router 注册任何 Agent 路由。
func (s *AgentServer) ConfigureLedgerRoutes(reader moduleapi.RuntimeTargetAgentLedgerReader) error {
	if s == nil || s.engine == nil {
		return errors.New("agent mTLS server is unavailable")
	}
	if reader == nil {
		return errors.New("runtime target agent ledger reader is required")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.server != nil {
		return errors.New("agent mTLS server is already running")
	}
	if s.ledgerRoutesConfigured {
		return errors.New("agent mTLS ledger routes are already configured")
	}
	s.engine.Use(agentLedgerNoStore())
	s.engine.GET(agentLedgerSnapshotPath, agentLedgerSnapshotHandler(reader, s.logger))
	s.engine.POST(agentTelemetryReportsPath, agentTelemetryReportHandler(reader, s.logger))
	s.ledgerRoutesConfigured = true
	return nil
}

func agentLedgerNoStore() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		ctx.Header("Cache-Control", "no-store")
		ctx.Next()
	}
}

func agentLedgerSnapshotHandler(reader moduleapi.RuntimeTargetAgentLedgerReader, runtimeLogger *zap.Logger) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		identity, ok := AgentMTLSIdentityFromGinContext(ctx)
		if !ok || agentLedgerHasForbiddenHeaders(ctx) {
			abortAgentLedgerIdentity(ctx, runtimeLogger)
			return
		}
		snapshot, err := reader.IssueLedgerSnapshot(ctx.Request.Context(), identity.AgentIdentity)
		if err != nil {
			abortAgentLedgerIdentity(ctx, runtimeLogger)
			return
		}
		ctx.JSON(http.StatusOK, snapshot)
	}
}

type agentTelemetryReportRequest struct {
	SnapshotID            string `json:"snapshot_id"`
	SnapshotDigest        string `json:"snapshot_digest"`
	ObservedAt            string `json:"observed_at"`
	ExpiresAt             string `json:"expires_at"`
	Available             bool   `json:"available"`
	ImplementationVersion string `json:"implementation_version"`
	Diagnostic            string `json:"diagnostic"`
}

func agentTelemetryReportHandler(reader moduleapi.RuntimeTargetAgentLedgerReader, runtimeLogger *zap.Logger) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		identity, ok := AgentMTLSIdentityFromGinContext(ctx)
		if !ok || agentLedgerHasForbiddenHeaders(ctx) {
			abortAgentLedgerIdentity(ctx, runtimeLogger)
			return
		}
		request, err := decodeAgentTelemetryReport(ctx)
		if err != nil {
			abortInvalidAgentLedgerRequest(ctx, runtimeLogger)
			return
		}
		if err := reader.SubmitTelemetryReport(ctx.Request.Context(), moduleapi.RuntimeTargetTelemetryReport{IdentityID: identity.IdentityID, TargetID: identity.TargetID, AgentID: identity.AgentID, Generation: identity.Generation, CertificateSerial: identity.CertificateSerial, PublicKeyFingerprint: identity.PublicKeyFingerprint, SnapshotID: request.SnapshotID, SnapshotDigest: request.SnapshotDigest, ObservedAt: request.ObservedAt, ExpiresAt: request.ExpiresAt, Available: request.Available, ImplementationVersion: request.ImplementationVersion, Diagnostic: request.Diagnostic}); err != nil {
			abortAgentLedgerIdentity(ctx, runtimeLogger)
			return
		}
		ctx.Status(http.StatusNoContent)
	}
}

type parsedAgentTelemetryReport struct {
	SnapshotID, SnapshotDigest, ImplementationVersion, Diagnostic string
	ObservedAt, ExpiresAt                                         time.Time
	Available                                                     bool
}

//nolint:cyclop // 内容类型、大小、单值 JSON 与时间字段必须在同一请求边界严格拒绝。
func decodeAgentTelemetryReport(ctx *gin.Context) (parsedAgentTelemetryReport, error) {
	if ctx == nil || ctx.Request == nil || ctx.Request.Body == nil || !agentLedgerJSONContentType(ctx.GetHeader("Content-Type")) {
		return parsedAgentTelemetryReport{}, errors.New("agent telemetry request content type is invalid")
	}
	ctx.Request.Body = http.MaxBytesReader(ctx.Writer, ctx.Request.Body, maxAgentTelemetryReportBytes)
	decoder := json.NewDecoder(ctx.Request.Body)
	decoder.DisallowUnknownFields()
	var payload agentTelemetryReportRequest
	if err := decoder.Decode(&payload); err != nil {
		return parsedAgentTelemetryReport{}, err
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return parsedAgentTelemetryReport{}, errors.New("agent telemetry request must contain one JSON value")
	}
	observedAt, err := time.Parse(time.RFC3339, payload.ObservedAt)
	if err != nil {
		return parsedAgentTelemetryReport{}, err
	}
	expiresAt, err := time.Parse(time.RFC3339, payload.ExpiresAt)
	if err != nil {
		return parsedAgentTelemetryReport{}, err
	}
	if strings.TrimSpace(payload.SnapshotID) == "" || strings.TrimSpace(payload.SnapshotDigest) == "" {
		return parsedAgentTelemetryReport{}, errors.New("agent telemetry request is incomplete")
	}
	return parsedAgentTelemetryReport{SnapshotID: payload.SnapshotID, SnapshotDigest: payload.SnapshotDigest, ObservedAt: observedAt, ExpiresAt: expiresAt, Available: payload.Available, ImplementationVersion: payload.ImplementationVersion, Diagnostic: payload.Diagnostic}, nil
}

func agentLedgerJSONContentType(value string) bool {
	mediaType, _, err := mime.ParseMediaType(value)
	return err == nil && strings.EqualFold(mediaType, "application/json")
}
func agentLedgerHasForbiddenHeaders(ctx *gin.Context) bool {
	return ctx.GetHeader(httpheader.Authorization.String()) != "" || ctx.GetHeader(httpheader.Cookie.String()) != ""
}
func abortAgentLedgerIdentity(ctx *gin.Context, logger *zap.Logger) {
	AbortAppError(ctx, nil, logger, apperror.New(apperror.Descriptor{Kind: apperror.KindUnauthenticated, Code: errorcode.AuthTokenInvalid, MessageKey: messagecontract.AuthTokenInvalid}))
}
func abortInvalidAgentLedgerRequest(ctx *gin.Context, logger *zap.Logger) {
	AbortAppError(ctx, nil, logger, apperror.New(apperror.Descriptor{Kind: apperror.KindInvalidArgument, Code: errorcode.CommonInvalidArgument, MessageKey: messagecontract.CommonInvalidArgument}))
}
