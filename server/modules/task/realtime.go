package task

import (
	"context"
	"errors"
	"strconv"
	"strings"

	"graft/server/internal/moduleapi"
	"graft/server/internal/realtime"
	"graft/server/internal/realtimeauth"
	taskcontract "graft/server/modules/task/contract"
)

// SetRealtime 在模块装配完成后注入 realtime 票据、消息 hub 和主题注册器；写入受互斥锁保护。
func (r *Runtime) SetRealtime(tickets realtimeauth.Service, hub realtime.Hub, issuers realtime.TopicIssuerRegistry) {
	if r == nil {
		return
	}
	r.mu.Lock()
	r.realtimeTickets, r.realtimeHub, r.topicIssuers = tickets, hub, issuers
	r.mu.Unlock()
}

// RegisterRealtimeTopics 将 task:{id} 订阅主题的所有权注册到该 Runtime。
func (r *Runtime) RegisterRealtimeTopics() error {
	if r == nil || r.topicIssuers == nil {
		return errors.New("task realtime topic issuer registry is unavailable")
	}
	return r.topicIssuers.Register(taskcontract.TaskRealtimeTopicPrefix.String(), r)
}

// IssueSubscription 先校验任务所有者的查看权限，再签发统一 websocket 票据；任务不存在或无权访问时不泄露资源存在性。
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

// taskRealtimeTopic 构造指定任务的实时主题名称。
func taskRealtimeTopic(taskID uint64) string {
	return taskcontract.TaskRealtimeTopicPrefix.String() + strconv.FormatUint(taskID, 10)
}

// parseTaskRealtimeTopic 从实时主题中解析任务 ID，并验证主题格式和 ID 的有效性。
// 返回解析出的任务 ID；主题格式无效或任务 ID 无效时返回错误。
func parseTaskRealtimeTopic(topic string) (uint64, error) {
	prefix := taskcontract.TaskRealtimeTopicPrefix.String()
	value := strings.TrimPrefix(realtime.NormalizeTopic(topic), prefix)
	if value == "" || prefix+value != realtime.NormalizeTopic(topic) {
		return 0, errors.New("invalid task realtime topic")
	}
	id, err := strconv.ParseUint(value, 10, 64)
	if err != nil || id == 0 {
		return 0, errors.New("invalid task realtime id")
	}
	return id, nil
}

// publishTask 只在调用方已持久化对应事实后发布 realtime 通知，避免客户端看到无法从数据库重放的状态。
func (r *Runtime) publishTask(taskID uint64, eventType taskcontract.RealtimeEvent) {
	r.mu.RLock()
	hub := r.realtimeHub
	r.mu.RUnlock()
	if hub != nil && taskID != 0 {
		hub.Publish(taskRealtimeTopic(taskID), map[string]any{"type": eventType.String(), "task_id": taskID})
	}
}
