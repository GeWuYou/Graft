package runtimetarget

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	messagecontract "graft/server/internal/contract/message"
	"graft/server/internal/event"
	"graft/server/internal/httpx"
	"graft/server/internal/moduleapi"
	store "graft/server/modules/runtime-target/store"
)

const defaultAssignmentCandidateLimit = 20

type runtimeTargetUserAssignmentHTTP struct {
	TargetID  uint64 `json:"target_id"`
	UserID    uint64 `json:"user_id"`
	CreatedAt string `json:"created_at"`
	CreatedBy uint64 `json:"created_by"`
}

type runtimeTargetUserAssignmentRequestHTTP struct {
	UserID uint64 `json:"user_id"`
}

type runtimeTargetUserAssignmentReplaceRequestHTTP struct {
	UserIDs []uint64 `json:"user_ids"`
}

type runtimeTargetAssignmentCandidateHTTP struct {
	ID       uint64 `json:"id"`
	Username string `json:"username"`
	Display  string `json:"display"`
	Status   string `json:"status"`
}

func (m *Module) handleListAssignments(c *gin.Context) {
	target, ok := m.readTarget(c)
	if !ok {
		return
	}
	items, err := m.repository.ListUserAssignments(c.Request.Context(), target.ID)
	if err != nil {
		httpx.AbortAppError(c, m.i18n, m.runtimeLogger, err)
		return
	}
	response := make([]runtimeTargetUserAssignmentHTTP, 0, len(items))
	for _, item := range items {
		response = append(response, toAssignmentHTTP(item))
	}
	httpx.WriteSuccess(c, http.StatusOK, map[string]any{"items": response})
}

func (m *Module) handleGrantAssignment(c *gin.Context) {
	target, ok := m.readTarget(c)
	if !ok {
		return
	}
	var request runtimeTargetUserAssignmentRequestHTTP
	if err := c.ShouldBindJSON(&request); err != nil || request.UserID == 0 {
		httpx.AbortLocalizedError(c, m.i18n, http.StatusBadRequest, messagecontract.CommonInvalidArgument.String(), nil)
		return
	}
	if m.users == nil {
		httpx.AbortAppError(c, m.i18n, m.runtimeLogger, errors.New("runtime target user identity provider is unavailable"))
		return
	}
	if _, err := m.users.GetCurrentUserByID(c.Request.Context(), request.UserID); err != nil {
		m.writeAssignmentUserError(c, err)
		return
	}
	actorID, ok := assignmentActorID(c)
	if !ok {
		httpx.AbortLocalizedError(c, m.i18n, http.StatusUnauthorized, messagecontract.AuthTokenMissing.String(), nil)
		return
	}
	assignment, err := m.repository.GrantUserAssignment(c.Request.Context(), target.ID, request.UserID, actorID)
	if err != nil {
		httpx.AbortAppError(c, m.i18n, m.runtimeLogger, err)
		return
	}
	httpx.WriteSuccess(c, http.StatusOK, toAssignmentHTTP(assignment))
}

func (m *Module) handleAssignmentCandidates(c *gin.Context) {
	if _, ok := m.readTarget(c); !ok {
		return
	}
	query, ok := m.assignmentCandidateQuery(c)
	if !ok {
		return
	}
	if m.candidates == nil {
		httpx.AbortAppError(c, m.i18n, m.runtimeLogger, errors.New("runtime target user candidate reader is unavailable"))
		return
	}
	items, total, err := m.candidates.ListUserCandidates(c.Request.Context(), query)
	if err != nil {
		httpx.AbortAppError(c, m.i18n, m.runtimeLogger, err)
		return
	}
	response := make([]runtimeTargetAssignmentCandidateHTTP, 0, len(items))
	for _, item := range items {
		response = append(response, runtimeTargetAssignmentCandidateHTTP{ID: item.ID, Username: item.Username, Display: item.Display, Status: item.Status})
	}
	httpx.WriteSuccess(c, http.StatusOK, map[string]any{"items": response, "total": total, "limit": query.Limit, "offset": query.Offset})
}

func (m *Module) handleReplaceAssignments(c *gin.Context) {
	target, ok := m.readTarget(c)
	if !ok {
		return
	}
	var request runtimeTargetUserAssignmentReplaceRequestHTTP
	if err := c.ShouldBindJSON(&request); err != nil || len(request.UserIDs) > 10000 {
		httpx.AbortLocalizedError(c, m.i18n, http.StatusBadRequest, messagecontract.CommonInvalidArgument.String(), nil)
		return
	}
	actorID, ok := assignmentActorID(c)
	if !ok {
		httpx.AbortLocalizedError(c, m.i18n, http.StatusUnauthorized, messagecontract.AuthTokenMissing.String(), nil)
		return
	}
	userIDs, ok := m.resolveAssignmentUserIDs(c, request.UserIDs)
	if !ok {
		return
	}
	items, err := m.replaceUserAssignments(c.Request.Context(), target, userIDs, actorID)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			httpx.AbortLocalizedError(c, m.i18n, http.StatusNotFound, "common.not_found", nil)
			return
		}
		httpx.AbortAppError(c, m.i18n, m.runtimeLogger, err)
		return
	}
	response := make([]runtimeTargetUserAssignmentHTTP, 0, len(items))
	for _, item := range items {
		response = append(response, toAssignmentHTTP(item))
	}
	httpx.WriteSuccess(c, http.StatusOK, map[string]any{"items": response})
}

func (m *Module) assignmentCandidateQuery(c *gin.Context) (moduleapi.UserCandidateQuery, bool) {
	query := moduleapi.UserCandidateQuery{Search: strings.TrimSpace(c.Query("search")), Limit: defaultAssignmentCandidateLimit}
	if raw := strings.TrimSpace(c.Query("limit")); raw != "" {
		limit, err := strconv.Atoi(raw)
		if err != nil || limit < 1 || limit > 100 {
			httpx.AbortLocalizedError(c, m.i18n, http.StatusBadRequest, messagecontract.CommonInvalidArgument.String(), nil)
			return moduleapi.UserCandidateQuery{}, false
		}
		query.Limit = limit
	}
	if raw := strings.TrimSpace(c.Query("offset")); raw != "" {
		offset, err := strconv.Atoi(raw)
		if err != nil || offset < 0 {
			httpx.AbortLocalizedError(c, m.i18n, http.StatusBadRequest, messagecontract.CommonInvalidArgument.String(), nil)
			return moduleapi.UserCandidateQuery{}, false
		}
		query.Offset = offset
	}
	return query, true
}

func (m *Module) resolveAssignmentUserIDs(c *gin.Context, requestedIDs []uint64) ([]uint64, bool) {
	if m.users == nil {
		httpx.AbortAppError(c, m.i18n, m.runtimeLogger, errors.New("runtime target user identity provider is unavailable"))
		return nil, false
	}
	seen := make(map[uint64]struct{}, len(requestedIDs))
	userIDs := make([]uint64, 0, len(requestedIDs))
	for _, userID := range requestedIDs {
		if userID == 0 {
			httpx.AbortLocalizedError(c, m.i18n, http.StatusBadRequest, messagecontract.CommonInvalidArgument.String(), nil)
			return nil, false
		}
		if _, exists := seen[userID]; exists {
			continue
		}
		if _, err := m.users.GetCurrentUserByID(c.Request.Context(), userID); err != nil {
			m.writeAssignmentUserError(c, err)
			return nil, false
		}
		seen[userID] = struct{}{}
		userIDs = append(userIDs, userID)
	}
	return userIDs, true
}

func (m *Module) replaceUserAssignments(ctx context.Context, target store.Target, userIDs []uint64, actorID uint64) ([]store.UserAssignment, error) {
	var items []store.UserAssignment
	err := m.repository.RunInTransaction(ctx, func(txCtx context.Context, tx *sql.Tx) error {
		var err error
		items, err = m.repository.ReplaceUserAssignmentsTx(txCtx, target.ID, userIDs, actorID)
		if err != nil {
			return err
		}
		return m.publishAssignmentReplacementAuditTx(txCtx, tx, target, len(items))
	})
	return items, err
}

func (m *Module) publishAssignmentReplacementAuditTx(ctx context.Context, tx *sql.Tx, target store.Target, assignmentCount int) error {
	payload := moduleapi.AuditEvent{Kind: moduleapi.AuditEventKindDomain, Action: "runtime_target.assignment.replace", ResourceType: "runtime_target", ResourceID: strconv.FormatUint(target.ID, 10), ResourceName: strings.TrimSpace(target.DisplayName), StatusCode: http.StatusOK, Success: true, Metadata: map[string]any{"assignment_count": assignmentCount}}
	envelope, err := httpx.NewAuditEvent(moduleID, payload)
	if err != nil {
		return err
	}
	_, err = m.events.PublishTx(ctx, tx, envelope, event.PublishOptions{Delivery: event.DeliveryDurable})
	return err
}

func (m *Module) handleRevokeAssignment(c *gin.Context) {
	target, ok := m.readTarget(c)
	if !ok {
		return
	}
	userID, err := strconv.ParseUint(strings.TrimSpace(c.Param("userId")), 10, 64)
	if err != nil || userID == 0 {
		httpx.AbortLocalizedError(c, m.i18n, http.StatusBadRequest, messagecontract.CommonInvalidArgument.String(), nil)
		return
	}
	actorID, ok := assignmentActorID(c)
	if !ok {
		httpx.AbortLocalizedError(c, m.i18n, http.StatusUnauthorized, messagecontract.AuthTokenMissing.String(), nil)
		return
	}
	if err := m.repository.RevokeUserAssignment(c.Request.Context(), target.ID, userID, actorID); err != nil {
		if errors.Is(err, store.ErrNotFound) {
			httpx.AbortLocalizedError(c, m.i18n, http.StatusNotFound, "common.not_found", nil)
			return
		}
		httpx.AbortAppError(c, m.i18n, m.runtimeLogger, err)
		return
	}
	httpx.WriteSuccess[any](c, http.StatusOK, nil)
}

func (m *Module) writeAssignmentUserError(c *gin.Context, err error) {
	if errors.Is(err, moduleapi.ErrUserNotFound) {
		httpx.AbortLocalizedError(c, m.i18n, http.StatusNotFound, "common.not_found", nil)
		return
	}
	httpx.AbortAppError(c, m.i18n, m.runtimeLogger, err)
}

func assignmentActorID(c *gin.Context) (uint64, bool) {
	auth, ok := moduleapi.RequestAuthContextFromContext(c.Request.Context())
	if !ok || auth.User == nil || auth.User.ID == 0 {
		return 0, false
	}
	return auth.User.ID, true
}

func toAssignmentHTTP(item store.UserAssignment) runtimeTargetUserAssignmentHTTP {
	return runtimeTargetUserAssignmentHTTP{TargetID: item.TargetID, UserID: item.UserID, CreatedAt: item.CreatedAt.UTC().Format("2006-01-02T15:04:05Z07:00"), CreatedBy: item.CreatedBy}
}
