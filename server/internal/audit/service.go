package audit

import (
	"context"
	"errors"
	"strings"
	"time"

	"graft/server/internal/store"
)

// RecordInput 描述一次统一审计写入所需的最小业务输入。
type RecordInput struct {
	OperatorID    *uint64
	OperatorName  string
	Action        string
	ResourceType  string
	ResourceID    string
	RequestMethod string
	RequestPath   string
	IP            string
	UserAgent     string
	Success       bool
	ErrorMessage  string
	CreatedAt     time.Time
}

// Service 负责把统一审计输入写入稳定仓储边界。
//
// Service 只做轻量输入规范化与默认时间补齐，不引入额外生命周期资源。
type Service struct {
	repo store.AuditRepository
}

// NewService 创建最小审计写入服务。
func NewService(repo store.AuditRepository) (*Service, error) {
	if repo == nil {
		return nil, errors.New("audit repository is required")
	}

	return &Service{repo: repo}, nil
}

// Record 写入一条统一审计记录。
func (s *Service) Record(ctx context.Context, input RecordInput) (store.AuditLog, error) {
	if s == nil || s.repo == nil {
		return store.AuditLog{}, errors.New("audit service is unavailable")
	}
	action := strings.TrimSpace(input.Action)
	if action == "" {
		return store.AuditLog{}, errors.New("audit action is required")
	}
	if input.CreatedAt.IsZero() {
		input.CreatedAt = time.Now().UTC()
	}

	return s.repo.CreateAuditLog(ctx, store.CreateAuditLogInput{
		OperatorID:    input.OperatorID,
		OperatorName:  strings.TrimSpace(input.OperatorName),
		Action:        action,
		ResourceType:  strings.TrimSpace(input.ResourceType),
		ResourceID:    strings.TrimSpace(input.ResourceID),
		RequestMethod: strings.TrimSpace(input.RequestMethod),
		RequestPath:   strings.TrimSpace(input.RequestPath),
		IP:            strings.TrimSpace(input.IP),
		UserAgent:     strings.TrimSpace(input.UserAgent),
		Success:       input.Success,
		ErrorMessage:  strings.TrimSpace(input.ErrorMessage),
		CreatedAt:     input.CreatedAt,
	})
}
