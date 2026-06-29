package audit

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	auditstore "graft/server/modules/audit/store"
)

// normalizeCandidateAction 返回候选记录的规范化动作标识。
// normalizeCandidateAction 返回候选审计记录的规范化动作标识，优先使用事件类型。
func normalizeCandidateAction(candidate auditstore.AuditCandidate) string {
	if eventType := strings.TrimSpace(candidate.EventType); eventType != "" {
		return eventType
	}

	return strings.TrimSpace(candidate.Action)
}

// candidateMetadata 规范化候选审计记录的元数据，并补充统一字段、会话 ID、策略规则和兼容旧字段别名。
func candidateMetadata(candidate auditstore.AuditCandidate, decision auditstore.AuditPolicyDecision) any {
	metadata := decodeCandidateMetadata(candidate.Metadata)
	resolved := resolveCandidateMetadataFields(candidate, metadata)

	applyCanonicalCandidateMetadata(metadata, candidate, resolved)
	if sessionID := firstNonEmptyTrimmed(strings.TrimSpace(candidate.SessionID), stringValue(metadata["sessionId"]), stringValue(metadata["session_id"])); sessionID != "" {
		metadata["sessionId"] = sessionID
		metadata["session_id"] = sessionID
	}
	if decision.Rule != nil {
		metadata["policy_rule_id"] = decision.Rule.ID
		metadata["policy_rule_name"] = decision.Rule.Name
		metadata["policy_effect"] = decision.Rule.Effect
	}
	applyLegacyCandidateMetadataAliases(metadata, resolved)
	return metadata
}

type resolvedCandidateMetadata struct {
	requestMethod string
	requestPath   string
	requestID     string
	traceID       string
	eventType     string
	targetType    string
	targetID      string
	status        int
	actorID       string
	actorType     string
}

// resolveCandidateMetadataFields 从候选记录及其元数据别名中解析标准化字段。
// 解析请求方法、路径、请求 ID、追踪 ID、事件类型、目标类型与 ID、状态码以及 actor 信息，并在存在 actor ID 时默认将 actorType 设为 "user"。
func resolveCandidateMetadataFields(candidate auditstore.AuditCandidate, metadata map[string]any) resolvedCandidateMetadata {
	actorID := ""
	if candidate.ActorUserID != nil {
		actorID = strconv.FormatUint(*candidate.ActorUserID, 10)
	}

	resolved := resolvedCandidateMetadata{
		requestMethod: firstNonEmptyTrimmed(strings.TrimSpace(candidate.RequestMethod), stringValue(metadata["method"]), stringValue(metadata["request_method"])),
		requestPath:   firstNonEmptyTrimmed(strings.TrimSpace(candidate.RequestPath), stringValue(metadata["route"]), stringValue(metadata["path"]), stringValue(metadata["request_path"])),
		requestID:     firstNonEmptyTrimmed(strings.TrimSpace(candidate.RequestID), stringValue(metadata["requestId"]), stringValue(metadata["request_id"])),
		eventType:     firstNonEmptyTrimmed(strings.TrimSpace(candidate.EventType), stringValue(metadata["eventType"]), stringValue(metadata["event_type"])),
		targetType:    firstNonEmptyTrimmed(strings.TrimSpace(candidate.TargetType), stringValue(metadata["targetType"]), stringValue(metadata["target_type"])),
		targetID:      firstNonEmptyTrimmed(strings.TrimSpace(candidate.ResourceID), stringValue(metadata["targetId"]), stringValue(metadata["target_id"])),
		status:        firstNonZeroInt(candidate.StatusCode, intValue(metadata["status"]), intValue(metadata["status_code"])),
		actorID:       firstNonEmptyTrimmed(actorID, stringValue(metadata["actorId"]), stringValue(metadata["actor_id"])),
		actorType:     firstNonEmptyTrimmed(stringValue(metadata["actorType"]), stringValue(metadata["actor_type"])),
	}
	resolved.traceID = firstNonEmptyTrimmed(strings.TrimSpace(candidate.TraceID), stringValue(metadata["traceId"]), stringValue(metadata["trace_id"]), resolved.requestID)
	if resolved.actorType == "" && resolved.actorID != "" {
		resolved.actorType = "user"
	}

	return resolved
}

// applyCanonicalCandidateMetadata 将候选审计信息写入统一的规范化元数据字段中。
// 其中包括审计来源、请求标识、追踪标识、请求方法、路径、路由、状态码，以及可选的 actor、事件和目标信息。
func applyCanonicalCandidateMetadata(metadata map[string]any, candidate auditstore.AuditCandidate, resolved resolvedCandidateMetadata) {
	metadata["auditSource"] = string(candidate.Source)
	metadata["requestId"] = resolved.requestID
	metadata["traceId"] = resolved.traceID
	metadata["method"] = resolved.requestMethod
	metadata["path"] = resolved.requestPath
	metadata["route"] = resolved.requestPath
	metadata["status"] = resolved.status
	assignOptionalMetadataString(metadata, "actorId", resolved.actorID)
	assignOptionalMetadataString(metadata, "actorType", resolved.actorType)
	assignOptionalMetadataString(metadata, "eventType", resolved.eventType)
	assignOptionalMetadataString(metadata, "targetType", resolved.targetType)
	assignOptionalMetadataString(metadata, "targetId", resolved.targetID)
}

// applyLegacyCandidateMetadataAliases 为元数据写入旧字段兼容别名。
// 它将规范字段复制到 audit_source、request_method、request_path、status_code 和 trace_id，并在值存在时补充 event_type、target_type、target_id。
func applyLegacyCandidateMetadataAliases(metadata map[string]any, resolved resolvedCandidateMetadata) {
	metadata["audit_source"] = metadata["auditSource"]
	metadata["request_method"] = metadata["method"]
	metadata["request_path"] = metadata["path"]
	metadata["status_code"] = metadata["status"]
	metadata["trace_id"] = metadata["traceId"]
	assignOptionalMetadataString(metadata, "event_type", resolved.eventType)
	assignOptionalMetadataString(metadata, "target_type", resolved.targetType)
	assignOptionalMetadataString(metadata, "target_id", resolved.targetID)
}

// assignOptionalMetadataString 在值不为空时写入元数据字段。
func assignOptionalMetadataString(metadata map[string]any, key string, value string) {
	if value != "" {
		metadata[key] = value
	}
}

// decodeCandidateMetadata 将原始候选元数据解码为映射。
// 空输入、解码失败或解码结果为空时，返回空映射。
func decodeCandidateMetadata(raw json.RawMessage) map[string]any {
	if len(raw) == 0 {
		return map[string]any{}
	}

	var metadata map[string]any
	if err := json.Unmarshal(raw, &metadata); err != nil || metadata == nil {
		return map[string]any{}
	}

	return metadata
}

// classifyCandidateRiskLevel 根据候选审计记录的规范化内容计算风险等级。
// 它会先规范化动作与元数据，再推导审计结果并据此完成风险分级。
func classifyCandidateRiskLevel(candidate auditstore.AuditCandidate) auditstore.AuditRiskLevel {
	record := auditstore.AuditLog{
		Action:       normalizeCandidateAction(candidate),
		Success:      candidate.Success,
		ResourceType: candidate.ResourceType,
	}
	record.Metadata = mustMarshalMetadata(candidateMetadata(candidate, auditstore.AuditPolicyDecision{}))
	record.RequestPath = candidate.RequestPath
	record.StatusCode = candidate.StatusCode
	record.Result = classifyCandidateResult(record, decodeCandidateMetadata(record.Metadata))
	return classifyCandidateAuditRiskLevel(record)
}

// mustMarshalMetadata 将值序列化为 JSON。
//
// @returns 序列化后的 JSON；当序列化失败时返回空对象 `{}`。
func mustMarshalMetadata(value any) json.RawMessage {
	payload, err := json.Marshal(value)
	if err != nil {
		return json.RawMessage([]byte("{}"))
	}
	return payload
}

// classifyCandidateResult 根据成功标记、状态码和错误元数据推导审计结果。
// 当记录成功、被拒绝、出现系统错误或显式错误信息时，分别返回对应结果。
//
// @param record 审计记录。
// @param metadata 解析后的元数据。
// @returns 审计结果。
func classifyCandidateResult(record auditstore.AuditLog, metadata map[string]any) auditstore.AuditResult {
	if record.Success {
		return auditstore.AuditResultSuccess
	}

	statusCode := record.StatusCode
	if statusCode == 0 {
		if raw, ok := metadata["status_code"]; ok {
			switch typed := raw.(type) {
			case float64:
				statusCode = int(typed)
			case int:
				statusCode = typed
			}
		}
	}
	if statusCode == http.StatusForbidden {
		return auditstore.AuditResultDenied
	}

	if errorKind, _ := metadata["error_kind"].(string); statusCode >= http.StatusInternalServerError || strings.TrimSpace(errorKind) == "system" {
		return auditstore.AuditResultError
	}
	if errorText, _ := metadata["error"].(string); strings.TrimSpace(errorText) != "" {
		return auditstore.AuditResultError
	}

	return auditstore.AuditResultFailed
}

// classifyCandidateAuditRiskLevel 根据审计记录的结果、资源类型和操作动作计算风险等级。
// 当结果为 Error 或 Denied 时返回 Critical；当资源类型为 container 或 container_batch 且动作为 ops.container.action.* 时返回 High。
// classifyCandidateAuditRiskLevel 根据审计记录的结果、资源类型和动作标识计算风险等级。
// @return 计算得到的审计风险等级。
func classifyCandidateAuditRiskLevel(record auditstore.AuditLog) auditstore.AuditRiskLevel {
	action := strings.ToLower(strings.TrimSpace(record.Action))
	resourceType := strings.ToLower(strings.TrimSpace(record.ResourceType))

	if record.Result == auditstore.AuditResultError || record.Result == auditstore.AuditResultDenied {
		return auditstore.AuditRiskLevelCritical
	}
	if (resourceType == "container" || resourceType == "container_batch") && strings.HasPrefix(action, "ops.container.action.") {
		return auditstore.AuditRiskLevelHigh
	}
	if containsAny(action, []string{"reset_password", "update_permission", "update_role", "assign_role", "token_revoke"}) {
		return auditstore.AuditRiskLevelCritical
	}
	if record.Result == auditstore.AuditResultFailed || containsAny(action, []string{"delete", "reset", "grant", "assign", "revoke", "remove", "replace", "update_role", "update_permission"}) {
		return auditstore.AuditRiskLevelHigh
	}
	if containsAny(action, []string{"login_failed", "login", "permission", "role", "auth"}) {
		return auditstore.AuditRiskLevelMedium
	}

	return auditstore.AuditRiskLevelLow
}

// containsAny 检查源字符串是否包含任一关键字。
// @returns 若 source 包含 keywords 中任一子串则为 `true`，否则为 `false`。
func containsAny(source string, keywords []string) bool {
	for _, keyword := range keywords {
		if strings.Contains(source, keyword) {
			return true
		}
	}
	return false
}

// stringValue 提取并清理字符串值。
// 当 value 为字符串时返回去除首尾空白后的内容；否则返回空字符串。
func stringValue(value any) string {
	typed, ok := value.(string)
	if !ok {
		return ""
	}
	return strings.TrimSpace(typed)
}

// intValue 将支持的数值类型转换为 int。
// 对于 int、int32、int64 和 float64，返回其对应的 int 值；其他类型返回 0。
func intValue(value any) int {
	switch typed := value.(type) {
	case int:
		return typed
	case int32:
		return int(typed)
	case int64:
		return int(typed)
	case float64:
		return int(typed)
	default:
		return 0
	}
}

// firstNonZeroInt 返回提供的整数中第一个非零值。
// @returns 第一个非零整数；如果全部为 0，则返回 0。
func firstNonZeroInt(values ...int) int {
	for _, value := range values {
		if value != 0 {
			return value
		}
	}
	return 0
}

// sanitizeMetadata 将输入元数据规范化并清洗为可序列化的 JSON。
// 当输入为 nil 时，返回空对象 `{}`。若归一化或序列化失败，则返回带上下文的错误。
// @param input 待规范化的元数据输入。
// @returns 清洗后的 JSON 元数据；当输入为 nil 时返回空对象 `{}`。
func sanitizeMetadata(input any) (json.RawMessage, error) {
	if input == nil {
		return json.RawMessage([]byte("{}")), nil
	}

	payload, err := normalizeMetadataValue(input)
	if err != nil {
		return nil, fmt.Errorf("normalize metadata value: %w", err)
	}

	sanitized := sanitizeMetadataValue(payload)
	if sanitized == nil {
		sanitized = map[string]any{}
	}

	encoded, err := json.Marshal(sanitized)
	if err != nil {
		return nil, fmt.Errorf("marshal metadata value: %w", err)
	}

	return json.RawMessage(encoded), nil
}

// normalizeMetadataValue 将输入规范化为可按 JSON 解码的通用值。
// 对于 json.RawMessage 和 []byte，会直接解码其内容；对其他类型，会先序列化再解码。
// 当输入为空字节时，返回空的对象映射。解析失败时返回带上下文的错误。
func normalizeMetadataValue(input any) (any, error) {
	switch typed := input.(type) {
	case json.RawMessage:
		if len(typed) == 0 {
			return map[string]any{}, nil
		}
		var decoded any
		if err := json.Unmarshal(typed, &decoded); err != nil {
			return nil, fmt.Errorf("unmarshal metadata raw message: %w", err)
		}
		return decoded, nil
	case []byte:
		if len(typed) == 0 {
			return map[string]any{}, nil
		}
		var decoded any
		if err := json.Unmarshal(typed, &decoded); err != nil {
			return nil, fmt.Errorf("unmarshal metadata bytes: %w", err)
		}
		return decoded, nil
	default:
		encoded, err := json.Marshal(typed)
		if err != nil {
			return nil, fmt.Errorf("marshal metadata input: %w", err)
		}
		var decoded any
		if err := json.Unmarshal(encoded, &decoded); err != nil {
			return nil, fmt.Errorf("unmarshal metadata payload: %w", err)
		}
		return decoded, nil
	}
}

// parseOptionalUint64Param 解析可选的 uint64 路径参数。
//
// @param ginParamGetter 提供参数读取方法的对象。
// @param key 参数名。
// @returns 解析后的数值、是否存在该参数，以及解析错误。若参数为空，返回 0、false 和 nil。
func parseOptionalUint64Param(ginParamGetter interface{ Param(string) string }, key string) (uint64, bool, error) {
	value := strings.TrimSpace(ginParamGetter.Param(key))
	if value == "" {
		return 0, false, nil
	}
	parsed, err := strconv.ParseUint(value, 10, 64)
	if err != nil {
		return 0, false, err
	}
	return parsed, true, nil
}
