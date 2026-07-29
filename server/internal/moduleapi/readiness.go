package moduleapi

// ReadinessState 描述一项诊断条件的当前观测结果，而不推断其显示颜色或业务阻断含义。
type ReadinessState string

const (
	// ReadinessStatePassed 表示条件已满足。
	ReadinessStatePassed ReadinessState = "passed"
	// ReadinessStateWarning 表示条件可继续但需要操作者注意。
	ReadinessStateWarning ReadinessState = "warning"
	// ReadinessStateFailed 表示条件未满足。
	ReadinessStateFailed ReadinessState = "failed"
	// ReadinessStateUnavailable 表示当前无法可靠观察条件。
	ReadinessStateUnavailable ReadinessState = "unavailable"
)

// ReadinessSeverity 供所有平台诊断消费者一致地呈现风险优先级。
type ReadinessSeverity string

const (
	// ReadinessSeveritySuccess 表示满足条件。
	ReadinessSeveritySuccess ReadinessSeverity = "success"
	// ReadinessSeverityInfo 表示提示性信息。
	ReadinessSeverityInfo ReadinessSeverity = "info"
	// ReadinessSeverityWarning 表示需要注意但未阻断的情况。
	ReadinessSeverityWarning ReadinessSeverity = "warning"
	// ReadinessSeverityCritical 表示当前操作不应继续。
	ReadinessSeverityCritical ReadinessSeverity = "critical"
)

// ReadinessEvidenceState 描述一条底层观测是否符合预期。
type ReadinessEvidenceState string

const (
	// ReadinessEvidencePassed 表示观测符合预期。
	ReadinessEvidencePassed ReadinessEvidenceState = "passed"
	// ReadinessEvidenceFailed 表示观测不符合预期。
	ReadinessEvidenceFailed ReadinessEvidenceState = "failed"
	// ReadinessEvidenceUnavailable 表示当前无法获得观测。
	ReadinessEvidenceUnavailable ReadinessEvidenceState = "unavailable"
)

// ReadinessActionType 限定服务端可授权给诊断界面的操作类别。
type ReadinessActionType string

const (
	// ReadinessActionCommand 表示可复制或执行的受控命令。
	ReadinessActionCommand ReadinessActionType = "command"
	// ReadinessActionDocumentation 表示受控文档路径。
	ReadinessActionDocumentation ReadinessActionType = "documentation"
	// ReadinessActionNavigate 表示应用内或已验证的导航目标。
	ReadinessActionNavigate ReadinessActionType = "navigate"
	// ReadinessActionCopy 表示可复制的非敏感值。
	ReadinessActionCopy ReadinessActionType = "copy"
	// ReadinessActionRecheck 表示重新执行只读检查。
	ReadinessActionRecheck ReadinessActionType = "recheck"
)

// ReadinessEvidence 是一条可结构化展示的诊断观测；敏感证据只能由拥有相应权限的调用方接收。
type ReadinessEvidence struct {
	Code      string                 `json:"code"`
	State     ReadinessEvidenceState `json:"state"`
	LabelKey  string                 `json:"label_key"`
	Value     string                 `json:"value,omitempty"`
	Expected  string                 `json:"expected,omitempty"`
	Sensitive bool                   `json:"sensitive,omitempty"`
}

// ReadinessAction 是服务端授权的诊断后续操作，界面不得从失败文本自行推导动作。
type ReadinessAction struct {
	ID       string              `json:"id"`
	Type     ReadinessActionType `json:"type"`
	LabelKey string              `json:"label_key"`
	Target   string              `json:"target,omitempty"`
	Params   map[string]string   `json:"params,omitempty"`
}

// ReadinessCheck 是具有稳定顺序、本地化键和结构化证据的平台诊断项。
type ReadinessCheck struct {
	ID         string              `json:"id"`
	Order      int                 `json:"order"`
	State      ReadinessState      `json:"state"`
	Severity   ReadinessSeverity   `json:"severity"`
	Blocking   bool                `json:"blocking"`
	TitleKey   string              `json:"title_key"`
	SummaryKey string              `json:"summary_key"`
	DetailKey  string              `json:"detail_key,omitempty"`
	Params     map[string]string   `json:"params,omitempty"`
	Evidence   []ReadinessEvidence `json:"evidence"`
	Actions    []ReadinessAction   `json:"actions"`
}

// Readiness 汇总某业务能力返回的有序平台诊断结果。
type Readiness struct {
	Overall    string           `json:"overall"`
	ReadyCount int              `json:"ready_count"`
	TotalCount int              `json:"total_count"`
	NextAction *ReadinessAction `json:"next_action,omitempty"`
	Checks     []ReadinessCheck `json:"checks"`
}
