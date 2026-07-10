package task

import (
	"context"
	"errors"
	"strconv"
	"strings"

	"graft/server/internal/moduleapi"
	"graft/server/internal/realtime"
	"graft/server/internal/realtimeauth"
)

const taskRealtimeTopicPrefix = "task:"

// SetRealtime provides the shared realtime primitives after module assembly.
func (r *Runtime) SetRealtime(tickets realtimeauth.Service, hub realtime.Hub, issuers realtime.TopicIssuerRegistry) {
	if r == nil {
		return
	}
	r.mu.Lock()
	r.realtimeTickets, r.realtimeHub, r.topicIssuers = tickets, hub, issuers
	r.mu.Unlock()
}

// RegisterRealtimeTopics makes this runtime the owner of task:{id} subscriptions.
func (r *Runtime) RegisterRealtimeTopics() error {
	if r == nil || r.topicIssuers == nil {
		return errors.New("task realtime topic issuer registry is unavailable")
	}
	return r.topicIssuers.Register(taskRealtimeTopicPrefix, r)
}

// IssueSubscription authorizes the consumer-owned resource before issuing a unified websocket ticket.
func (r *Runtime) IssueSubscription(ctx context.Context, request realtime.SubscriptionRequest) (realtime.SubscriptionResponse, error) {
	taskID, err := parseTaskRealtimeTopic(request.Topic)
	if err != nil {
		return realtime.SubscriptionResponse{}, realtime.ErrTopicNotFound
	}
	if request.RequestAuth.User == nil {
		return realtime.SubscriptionResponse{}, realtime.ErrTopicForbidden
	}
	task, err := r.GetTask(ctx, taskID)
	if err != nil {
		return realtime.SubscriptionResponse{}, realtime.ErrTopicNotFound
	}
	if err := r.AuthorizeOwner(ctx, request.RequestAuth.User, moduleapi.TaskOwnerActionView, task.Owner); err != nil {
		return realtime.SubscriptionResponse{}, realtime.ErrTopicForbidden
	}
	r.mu.RLock()
	tickets := r.realtimeTickets
	r.mu.RUnlock()
	if tickets == nil {
		return realtime.SubscriptionResponse{}, realtime.ErrTopicConflict
	}
	issued, err := (realtime.TicketIssuer{Tickets: tickets}).IssueTopicTicket(ctx, request)
	if err != nil {
		return realtime.SubscriptionResponse{}, realtime.ErrTopicConflict
	}
	return realtime.SubscriptionResponse{Topic: realtime.NormalizeTopic(request.Topic), Ticket: issued.Ticket, WebSocketURL: realtime.BuildTopicWebSocketURL(realtime.NormalizeTopic(request.Topic), issued.Ticket), ExpiresAt: issued.ExpiresAt}, nil
}

func taskRealtimeTopic(taskID uint64) string {
	return taskRealtimeTopicPrefix + strconv.FormatUint(taskID, 10)
}

func parseTaskRealtimeTopic(topic string) (uint64, error) {
	value := strings.TrimPrefix(realtime.NormalizeTopic(topic), taskRealtimeTopicPrefix)
	if value == "" || taskRealtimeTopicPrefix+value != realtime.NormalizeTopic(topic) {
		return 0, errors.New("invalid task realtime topic")
	}
	id, err := strconv.ParseUint(value, 10, 64)
	if err != nil || id == 0 {
		return 0, errors.New("invalid task realtime id")
	}
	return id, nil
}

// publishTask emits only after the caller has persisted the corresponding fact.
func (r *Runtime) publishTask(taskID uint64, eventType string) {
	r.mu.RLock()
	hub := r.realtimeHub
	r.mu.RUnlock()
	if hub != nil && taskID != 0 {
		hub.Publish(taskRealtimeTopic(taskID), map[string]any{"type": eventType, "task_id": taskID})
	}
}
