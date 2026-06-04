package logger

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"
)

const (
	// FieldOccurredAt stores the canonical UTC occurrence time for one app-log record.
	FieldOccurredAt = "occurred_at"
	// FieldSeverity stores the canonical app-log severity.
	FieldSeverity = "severity"
	// FieldMessage stores the canonical sanitized app-log message.
	FieldMessage = "message"
	// FieldFields stores bounded structured app-log attributes.
	FieldFields = "fields"

	appLogDefaultPageSize = 20
	appLogMaxPageSize     = 100
)

const (
	// AppLogSortFieldOccurredAt orders App Log records by canonical occurrence time.
	AppLogSortFieldOccurredAt AppLogSortField = "occurred_at"
	// AppLogSortFieldSeverity orders App Log records by severity.
	AppLogSortFieldSeverity AppLogSortField = "severity"
	// AppLogSortFieldComponent orders App Log records by component.
	AppLogSortFieldComponent AppLogSortField = "component"
)

const (
	// AppLogSortOrderAsc orders App Log records from low to high.
	AppLogSortOrderAsc AppLogSortOrder = "asc"
	// AppLogSortOrderDesc orders App Log records from high to low.
	AppLogSortOrderDesc AppLogSortOrder = "desc"
)

var (
	errAppLogStorageModeRequired    = errors.New("app log storage mode is required")
	errAppLogRetentionOwnerRequired = errors.New("app log retention owner is required")
	// ErrAppLogNotFound reports that a requested app-log record is outside the retained dataset.
	ErrAppLogNotFound = errors.New("app log not found")
)

var forbiddenAppLogPersistedFields = []string{
	"action",
	"actor_id",
	"actor_type",
	"audit_id",
	"authorization",
	"client_ip",
	"cookie",
	"decision",
	"ip",
	"path",
	"permission",
	"policy",
	"request_size",
	"resource_id",
	"resource_type",
	"response_size",
	"security_event_id",
	"session_id",
	"status",
	"status_code",
	"user_agent",
	"user_id",
	"username",
}

// AppLogSeverity describes the canonical persisted app-log severity surface.
type AppLogSeverity string

// AppLogSortField constrains App Log Explorer supported sort fields.
type AppLogSortField string

// AppLogSortOrder constrains App Log Explorer supported sort directions.
type AppLogSortOrder string

// AppLogSorter describes one validated App Log sort instruction.
type AppLogSorter struct {
	Field AppLogSortField
	Order AppLogSortOrder
}

const (
	// AppLogSeverityDebug persists debug-level runtime diagnostics.
	AppLogSeverityDebug AppLogSeverity = "debug"
	// AppLogSeverityInfo persists normal runtime progress events.
	AppLogSeverityInfo AppLogSeverity = "info"
	// AppLogSeverityWarn persists degraded but recoverable runtime states.
	AppLogSeverityWarn AppLogSeverity = "warn"
	// AppLogSeverityError persists runtime failures that require investigation.
	AppLogSeverityError AppLogSeverity = "error"
)

// AppLogStorageMode describes how the current repository authority stores App Log truth.
type AppLogStorageMode string

const (
	// AppLogStorageModeProcessOutput keeps App Log truth on the current process logger output only.
	AppLogStorageModeProcessOutput AppLogStorageMode = "process_output_only"
	// AppLogStorageModeRepositoryDurableStore persists App Log truth into logger-owned repository storage.
	AppLogStorageModeRepositoryDurableStore AppLogStorageMode = "repository_durable_store"
)

// AppLogRetentionOwner describes who currently owns App Log retention policy.
type AppLogRetentionOwner string

const (
	// AppLogRetentionOwnerNone means the repository runtime owns no retention policy while App Log stays on process output.
	AppLogRetentionOwnerNone AppLogRetentionOwner = "none"
	// AppLogRetentionOwnerLogger means logger owns the durable App Log retention cleanup lifecycle.
	AppLogRetentionOwnerLogger AppLogRetentionOwner = "server_internal_logger"
)

// AppLogStoragePolicy captures the current canonical App Log storage and retention authority.
type AppLogStoragePolicy struct {
	Mode           AppLogStorageMode
	RetentionOwner AppLogRetentionOwner
	DefaultWindow  time.Duration
}

// DefaultAppLogStoragePolicy returns the approved repository-owned durable App Log policy.
func DefaultAppLogStoragePolicy(defaultWindow time.Duration) AppLogStoragePolicy {
	return AppLogStoragePolicy{
		Mode:           AppLogStorageModeRepositoryDurableStore,
		RetentionOwner: AppLogRetentionOwnerLogger,
		DefaultWindow:  defaultWindow,
	}
}

// Validate ensures the storage policy keeps retention ownership aligned with persistence authority.
func (p AppLogStoragePolicy) Validate() error {
	if strings.TrimSpace(string(p.Mode)) == "" {
		return errAppLogStorageModeRequired
	}
	if strings.TrimSpace(string(p.RetentionOwner)) == "" {
		return errAppLogRetentionOwnerRequired
	}

	switch p.Mode {
	case AppLogStorageModeProcessOutput:
		if p.RetentionOwner != AppLogRetentionOwnerNone {
			return fmt.Errorf("app log process-output mode requires retention owner %q", AppLogRetentionOwnerNone)
		}
		if p.DefaultWindow != 0 {
			return errors.New("app log process-output mode does not allow a repository retention window")
		}
	case AppLogStorageModeRepositoryDurableStore:
		if p.RetentionOwner == AppLogRetentionOwnerNone {
			return errors.New("repository durable app-log storage requires a retention owner")
		}
		if p.DefaultWindow <= 0 {
			return errors.New("repository durable app-log storage requires a positive retention window")
		}
	default:
		return fmt.Errorf("unsupported app log storage mode %q", p.Mode)
	}

	return nil
}

// AppLogRecord defines the canonical persisted App Log field set.
type AppLogRecord struct {
	ID         uint64
	OccurredAt time.Time
	Severity   AppLogSeverity
	Component  string
	Message    string
	Operation  string
	RequestID  string
	TraceID    string
	Route      string
	Method     string
	Error      string
	Fields     map[string]string
}

// CreateAppLogInput describes one canonical App Log record before repository normalization.
type CreateAppLogInput = AppLogRecord

// AppLogListQuery describes the logger-owned App Log read model.
type AppLogListQuery struct {
	Page         int
	PageSize     int
	Severity     AppLogSeverity
	Component    string
	Operation    string
	RequestID    string
	TraceID      string
	Route        string
	Method       string
	Error        string
	Message      string
	Keyword      string
	OccurredFrom *time.Time
	OccurredTo   *time.Time
	Sorters      []AppLogSorter
}

// AppLogListResult carries a paginated logger-owned App Log query result.
type AppLogListResult struct {
	Items    []AppLogRecord
	Total    int64
	Page     int
	PageSize int
}

// Normalize sanitizes one canonical App Log persisted record shape without widening authority.
func (r AppLogRecord) Normalize() (AppLogRecord, error) {
	normalized := newNormalizedAppLogRecord(r)
	if err := validateNormalizedAppLogRecord(normalized); err != nil {
		return AppLogRecord{}, err
	}

	fields, err := normalizeAppLogRecordFields(r.Fields)
	if err != nil {
		return AppLogRecord{}, err
	}
	normalized.Fields = fields

	return normalized, nil
}

// Validate ensures the severity remains inside the canonical App Log surface.
func (s AppLogSeverity) Validate() error {
	switch s {
	case AppLogSeverityDebug, AppLogSeverityInfo, AppLogSeverityWarn, AppLogSeverityError:
		return nil
	default:
		return fmt.Errorf("unsupported app log severity %q", s)
	}
}

func normalizeAppLogListQuery(query AppLogListQuery) AppLogListQuery {
	query.Page = normalizePositivePage(query.Page)
	query.PageSize = normalizeAppLogPageSize(query.PageSize)
	query.Component = sanitizeComponent(query.Component)
	query.Operation = sanitizeString(query.Operation)
	query.RequestID = sanitizeString(query.RequestID)
	query.TraceID = sanitizeString(query.TraceID)
	query.Route = sanitizeString(query.Route)
	query.Method = sanitizeString(query.Method)
	query.Error = sanitizeString(query.Error)
	query.Message = sanitizeString(query.Message)
	query.Keyword = sanitizeString(query.Keyword)
	if err := query.Severity.Validate(); err != nil {
		query.Severity = ""
	}
	return query
}

func normalizePositivePage(page int) int {
	if page < 1 {
		return 1
	}
	return page
}

func normalizeAppLogPageSize(pageSize int) int {
	switch {
	case pageSize < 1:
		return appLogDefaultPageSize
	case pageSize > appLogMaxPageSize:
		return appLogMaxPageSize
	default:
		return pageSize
	}
}

// IsForbiddenAppLogPersistedField reports whether one field belongs to another authority boundary.
func IsForbiddenAppLogPersistedField(key string) bool {
	normalized := sanitizeFieldKey(key)
	if normalized == "" {
		return false
	}

	return slices.Contains(forbiddenAppLogPersistedFields, normalized)
}

func newNormalizedAppLogRecord(r AppLogRecord) AppLogRecord {
	return AppLogRecord{
		ID:         r.ID,
		OccurredAt: r.OccurredAt.UTC(),
		Severity:   r.Severity,
		Component:  sanitizeComponent(r.Component),
		Message:    sanitizeMessage(r.Message),
		Operation:  sanitizeFieldValue(FieldOperation, r.Operation).(string),
		RequestID:  sanitizeFieldValue(FieldRequestID, r.RequestID).(string),
		TraceID:    sanitizeFieldValue(FieldTraceID, r.TraceID).(string),
		Route:      sanitizeFieldValue(FieldRoute, r.Route).(string),
		Method:     sanitizeFieldValue(FieldMethod, r.Method).(string),
		Error:      sanitizeFieldValue(FieldError, r.Error).(string),
	}
}

func validateNormalizedAppLogRecord(record AppLogRecord) error {
	if record.OccurredAt.IsZero() {
		return errors.New("app log record occurred_at is required")
	}
	if err := record.Severity.Validate(); err != nil {
		return err
	}
	if record.Component == "" {
		return errors.New("app log record component is required")
	}
	if record.Message == "" {
		return errors.New("app log record message is required")
	}

	return nil
}

func normalizeAppLogRecordFields(fields map[string]string) (map[string]string, error) {
	normalized := make(map[string]string, len(fields))
	for key, value := range fields {
		normalizedKey := sanitizeFieldKey(key)
		if normalizedKey == "" {
			continue
		}
		if err := validateAppLogRecordFieldKey(normalizedKey); err != nil {
			return nil, err
		}
		if sanitized, ok := sanitizeFieldValue(normalizedKey, value).(string); ok && sanitized != "" {
			normalized[normalizedKey] = sanitized
		}
	}

	return normalized, nil
}

func validateAppLogRecordFieldKey(key string) error {
	if IsForbiddenAppLogPersistedField(key) {
		return fmt.Errorf("app log persisted field %q is forbidden", key)
	}
	if isAppLogTopLevelField(key) {
		return fmt.Errorf("app log persisted field %q collides with a canonical top-level field", key)
	}

	return nil
}

func isAppLogTopLevelField(key string) bool {
	switch key {
	case FieldOccurredAt,
		FieldSeverity,
		FieldComponent,
		FieldMessage,
		FieldOperation,
		FieldRequestID,
		FieldTraceID,
		FieldRoute,
		FieldMethod,
		FieldError,
		FieldFields:
		return true
	default:
		return false
	}
}

// AppLogRepository owns durable persistence for canonical App Log truth.
type AppLogRepository interface {
	CreateAppLog(context.Context, CreateAppLogInput) (AppLogRecord, error)
	DeleteAppLogsBefore(context.Context, time.Time) (int64, error)
	ListAppLogs(context.Context, AppLogListQuery) (AppLogListResult, error)
	GetAppLogByID(context.Context, uint64) (AppLogRecord, error)
}
