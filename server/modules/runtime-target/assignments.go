package runtimetarget

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	messagecontract "graft/server/internal/contract/message"
	"graft/server/internal/httpx"
	"graft/server/internal/moduleapi"
	store "graft/server/modules/runtime-target/store"
)

type runtimeTargetUserAssignmentHTTP struct {
	TargetID  uint64 `json:"target_id"`
	UserID    uint64 `json:"user_id"`
	CreatedAt string `json:"created_at"`
	CreatedBy uint64 `json:"created_by"`
}

type runtimeTargetUserAssignmentRequestHTTP struct {
	UserID uint64 `json:"user_id"`
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
