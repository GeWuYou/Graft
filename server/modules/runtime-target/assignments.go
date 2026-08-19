package runtimetarget

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
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

const (
	defaultAssignmentCandidateLimit    = 20
	maxAssignmentCandidateSearchLength = 128
)

type runtimeTargetUserAssignmentHTTP struct {
	TargetID  uint64 `json:"target_id"`
	UserID    uint64 `json:"user_id"`
	CreatedAt string `json:"created_at"`
	CreatedBy uint64 `json:"created_by"`
	Username  string `json:"username,omitempty"`
	Display   string `json:"display,omitempty"`
	Status    string `json:"status,omitempty"`
}

type runtimeTargetUserAssignmentRequestHTTP struct {
	UserID uint64 `json:"user_id"`
}

type runtimeTargetUserAssignmentReplaceRequestHTTP struct {
	UserIDs  []uint64 `json:"user_ids"`
	Revision uint64   `json:"revision"`
}

type runtimeTargetAssignmentBatchRequestHTTP struct {
	TargetIDs []uint64 `json:"target_ids"`
	UserIDs   []uint64 `json:"user_ids"`
	Action    string   `json:"action"`
}

type runtimeTargetAssignmentBatchResultHTTP struct {
	Targets int `json:"targets"`
	Users   int `json:"users"`
}

type runtimeTargetAssignmentCandidateHTTP struct {
	ID       uint64 `json:"id"`
	Username string `json:"username"`
	Display  string `json:"display"`
	Status   string `json:"status"`
}

type runtimeTargetAssignmentListHTTP struct {
	Items    []runtimeTargetUserAssignmentHTTP `json:"items"`
	Revision uint64                            `json:"revision"`
}

type runtimeTargetAssignmentCandidateListHTTP struct {
	Items  []runtimeTargetAssignmentCandidateHTTP `json:"items"`
	Total  int                                    `json:"total"`
	Limit  int                                    `json:"limit"`
	Offset int                                    `json:"offset"`
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
	revision, err := m.repository.UserAssignmentRevision(c.Request.Context(), target.ID)
	if err != nil {
		httpx.AbortAppError(c, m.i18n, m.runtimeLogger, err)
		return
	}
	summaries, err := m.assignmentUserSummaryMap(c.Request.Context(), assignmentUserIDs(items))
	if err != nil {
		httpx.AbortAppError(c, m.i18n, m.runtimeLogger, err)
		return
	}
	response := assignmentHTTPItems(items, summaries)
	httpx.WriteSuccess(c, http.StatusOK, runtimeTargetAssignmentListHTTP{Items: response, Revision: revision})
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
	summaries, err := m.assignmentUserSummaryMap(c.Request.Context(), []uint64{request.UserID})
	if err != nil {
		httpx.AbortAppError(c, m.i18n, m.runtimeLogger, err)
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
	httpx.WriteSuccess(c, http.StatusOK, toAssignmentHTTP(assignment, summaries[assignment.UserID]))
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
	httpx.WriteSuccess(c, http.StatusOK, runtimeTargetAssignmentCandidateListHTTP{Items: response, Total: total, Limit: query.Limit, Offset: query.Offset})
}

func (m *Module) handleReplaceAssignments(c *gin.Context) {
	target, ok := m.readTarget(c)
	if !ok {
		return
	}
	var request runtimeTargetUserAssignmentReplaceRequestHTTP
	if err := c.ShouldBindJSON(&request); err != nil || invalidAssignmentReplaceRequest(request) {
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
	summaries, err := m.assignmentUserSummaryMap(c.Request.Context(), userIDs)
	if err != nil {
		httpx.AbortAppError(c, m.i18n, m.runtimeLogger, err)
		return
	}
	items, revision, err := m.replaceUserAssignments(c.Request.Context(), target, userIDs, request.Revision, actorID)
	if err != nil {
		m.writeAssignmentReplaceError(c, err)
		return
	}
	response := assignmentHTTPItems(items, summaries)
	httpx.WriteSuccess(c, http.StatusOK, runtimeTargetAssignmentListHTTP{Items: response, Revision: revision})
}

func invalidAssignmentReplaceRequest(request runtimeTargetUserAssignmentReplaceRequestHTTP) bool {
	return len(request.UserIDs) > 10000 || request.Revision == 0
}

//nolint:cyclop,gocognit,gocyclo // validates one bounded request before entering the shared transaction.
func (m *Module) handleBatchAssignments(c *gin.Context) {
	var request runtimeTargetAssignmentBatchRequestHTTP
	if err := c.ShouldBindJSON(&request); err != nil || len(request.TargetIDs) == 0 || len(request.TargetIDs) > 1000 || len(request.UserIDs) == 0 || len(request.UserIDs) > 10000 || (request.Action != string(store.AssignmentBatchGrant) && request.Action != string(store.AssignmentBatchRevoke)) {
		httpx.AbortLocalizedError(c, m.i18n, http.StatusBadRequest, messagecontract.CommonInvalidArgument.String(), nil)
		return
	}
	actorID, ok := assignmentActorID(c)
	if !ok {
		httpx.AbortLocalizedError(c, m.i18n, http.StatusUnauthorized, messagecontract.AuthTokenMissing.String(), nil)
		return
	}
	targetIDs := uniqueAssignmentIDs(request.TargetIDs)
	userIDs, ok := m.resolveAssignmentUserIDs(c, request.UserIDs)
	if !ok {
		return
	}
	action := store.AssignmentBatchAction(request.Action)
	err := m.repository.RunInTransaction(c.Request.Context(), func(txCtx context.Context, tx *sql.Tx) error {
		if err := m.repository.ApplyAssignmentBatch(txCtx, targetIDs, userIDs, action, actorID); err != nil {
			return err
		}
		for _, targetID := range targetIDs {
			target, err := m.repository.Get(txCtx, targetID)
			if err != nil {
				return err
			}
			payload := moduleapi.AuditEvent{Kind: moduleapi.AuditEventKindDomain, Action: "runtime_target.assignment." + request.Action, ResourceType: "runtime_target", ResourceID: strconv.FormatUint(targetID, 10), ResourceName: strings.TrimSpace(target.DisplayName), StatusCode: http.StatusOK, Success: true, Metadata: map[string]any{"user_count": len(userIDs)}}
			envelope, err := httpx.NewAuditEvent(moduleID, payload)
			if err != nil {
				return err
			}
			if _, err = m.events.PublishTx(txCtx, tx, envelope, event.PublishOptions{Delivery: event.DeliveryDurable}); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		m.writeAssignmentReplaceError(c, err)
		return
	}
	httpx.WriteSuccess(c, http.StatusOK, runtimeTargetAssignmentBatchResultHTTP{Targets: len(targetIDs), Users: len(userIDs)})
}

func (m *Module) writeAssignmentReplaceError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, store.ErrAssignmentRevisionConflict):
		httpx.AbortLocalizedError(c, m.i18n, http.StatusConflict, "common.conflict", nil)
	case errors.Is(err, store.ErrNotFound):
		httpx.AbortLocalizedError(c, m.i18n, http.StatusNotFound, "common.not_found", nil)
	default:
		httpx.AbortAppError(c, m.i18n, m.runtimeLogger, err)
	}
}

func (m *Module) assignmentCandidateQuery(c *gin.Context) (moduleapi.UserCandidateQuery, bool) {
	query := moduleapi.UserCandidateQuery{Search: strings.TrimSpace(c.Query("search")), Limit: defaultAssignmentCandidateLimit}
	if len(query.Search) > maxAssignmentCandidateSearchLength {
		httpx.AbortLocalizedError(c, m.i18n, http.StatusBadRequest, messagecontract.CommonInvalidArgument.String(), nil)
		return moduleapi.UserCandidateQuery{}, false
	}
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

func uniqueAssignmentIDs(ids []uint64) []uint64 {
	seen := make(map[uint64]struct{}, len(ids))
	uniqueIDs := make([]uint64, 0, len(ids))
	for _, id := range ids {
		if _, exists := seen[id]; exists {
			continue
		}
		seen[id] = struct{}{}
		uniqueIDs = append(uniqueIDs, id)
	}
	return uniqueIDs
}

func (m *Module) replaceUserAssignments(ctx context.Context, target store.Target, userIDs []uint64, expectedRevision, actorID uint64) ([]store.UserAssignment, uint64, error) {
	var items []store.UserAssignment
	var revision uint64
	err := m.repository.RunInTransaction(ctx, func(txCtx context.Context, tx *sql.Tx) error {
		var err error
		items, revision, err = m.repository.ReplaceUserAssignmentsTx(txCtx, target.ID, userIDs, expectedRevision, actorID)
		if err != nil {
			return err
		}
		return m.publishAssignmentReplacementAuditTx(txCtx, tx, target, len(items))
	})
	return items, revision, err
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

func (m *Module) assignmentUserSummaryMap(ctx context.Context, userIDs []uint64) (map[uint64]moduleapi.UserAccountSummary, error) {
	if len(userIDs) == 0 {
		return map[uint64]moduleapi.UserAccountSummary{}, nil
	}
	if m.userSummaries == nil {
		return nil, errors.New("runtime target user summary reader is unavailable")
	}
	items, err := m.userSummaries.ListUserSummariesByIDs(ctx, uniqueAssignmentIDs(userIDs))
	if err != nil {
		return nil, fmt.Errorf("list runtime target assignment user summaries: %w", err)
	}
	summaries := make(map[uint64]moduleapi.UserAccountSummary, len(items))
	for _, item := range items {
		summaries[item.ID] = item
	}
	return summaries, nil
}

func assignmentUserIDs(items []store.UserAssignment) []uint64 {
	userIDs := make([]uint64, 0, len(items))
	for _, item := range items {
		userIDs = append(userIDs, item.UserID)
	}
	return userIDs
}

func assignmentHTTPItems(items []store.UserAssignment, summaries map[uint64]moduleapi.UserAccountSummary) []runtimeTargetUserAssignmentHTTP {
	response := make([]runtimeTargetUserAssignmentHTTP, 0, len(items))
	for _, item := range items {
		response = append(response, toAssignmentHTTP(item, summaries[item.UserID]))
	}
	return response
}

func toAssignmentHTTP(item store.UserAssignment, summary moduleapi.UserAccountSummary) runtimeTargetUserAssignmentHTTP {
	return runtimeTargetUserAssignmentHTTP{
		TargetID: item.TargetID, UserID: item.UserID,
		CreatedAt: item.CreatedAt.UTC().Format("2006-01-02T15:04:05Z07:00"), CreatedBy: item.CreatedBy,
		Username: summary.Username, Display: summary.Display, Status: summary.Status,
	}
}
