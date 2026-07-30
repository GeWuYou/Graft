package moduleapi

import "context"

// DeploymentRuntime 是部署运行时信息的唯一可信来源；消费者不得重新解释环境变量或 Docker facts。
type DeploymentRuntime interface {
	Current(context.Context) DeploymentContext
	Freeze(context.Context, DeploymentFreezeRequest) (DeploymentSnapshot, error)
}

// DeploymentFreezeRequest 指定一次受控操作选择的候选；空 CandidateKey 只适用于唯一确认的候选。
type DeploymentFreezeRequest struct {
	CandidateKey string
}

// DeploymentDiagnostic 是不可执行或需要确认时的稳定诊断投影。
type DeploymentDiagnostic struct {
	Code    string
	Message string
}

// DeploymentComposeCandidate 是由 Deployment Runtime 解释后的 Compose 候选。
type DeploymentComposeCandidate struct {
	key         string
	root        string
	configFiles []string
	projectName string
	confidence  string
	warnings    []string
}

// Key 返回仅供服务端操作确认使用的不透明候选标识。
func (c DeploymentComposeCandidate) Key() string { return c.key }

// Root 返回只可作为 Docker daemon 宿主机路径使用的 Compose 根目录。
func (c DeploymentComposeCandidate) Root() string { return c.root }

// ConfigFiles 返回复制后的 Compose 配置文件序列。
func (c DeploymentComposeCandidate) ConfigFiles() []string {
	return append([]string(nil), c.configFiles...)
}

// ProjectName 返回 Docker Compose 项目名。
func (c DeploymentComposeCandidate) ProjectName() string { return c.projectName }

// Confidence 返回 Discovery 对候选事实的置信度。
func (c DeploymentComposeCandidate) Confidence() string { return c.confidence }

// Warnings 返回复制后的管理员确认或诊断提示。
func (c DeploymentComposeCandidate) Warnings() []string { return append([]string(nil), c.warnings...) }

// NewDeploymentComposeCandidate 为 Deployment Runtime 实现创建不可变 Compose 候选值。
func NewDeploymentComposeCandidate(key, root string, configFiles []string, projectName, confidence string, warnings []string) DeploymentComposeCandidate {
	return DeploymentComposeCandidate{key: key, root: root, configFiles: append([]string(nil), configFiles...), projectName: projectName, confidence: confidence, warnings: append([]string(nil), warnings...)}
}

// DeploymentContext 是描述当前部署发现结果的不可变值对象。
type DeploymentContext struct {
	mode                        string
	composeRootSource           string
	composeConfirmationRequired bool
	composeCandidates           []DeploymentComposeCandidate
	diagnostics                 []DeploymentDiagnostic
}

// Mode 返回经 Deployment Runtime 解释后的部署运行时类型。
func (c DeploymentContext) Mode() string { return c.mode }

// ComposeRootSource 返回 Compose 根目录的解析来源。
func (c DeploymentContext) ComposeRootSource() string { return c.composeRootSource }

// IsComposeConfirmationRequired 表示操作开始前必须提交本次候选的确认标识。
func (c DeploymentContext) IsComposeConfirmationRequired() bool { return c.composeConfirmationRequired }

// ComposeCandidates 返回复制后的当前 Compose 候选。
func (c DeploymentContext) ComposeCandidates() []DeploymentComposeCandidate {
	return append([]DeploymentComposeCandidate(nil), c.composeCandidates...)
}

// Diagnostics 返回当前解析不可用或需确认的结构化诊断。
func (c DeploymentContext) Diagnostics() []DeploymentDiagnostic {
	return append([]DeploymentDiagnostic(nil), c.diagnostics...)
}

// IsAvailable 表示当前上下文能为一次受控 Compose 操作提供候选。
func (c DeploymentContext) IsAvailable() bool {
	return len(c.diagnostics) == 0 && len(c.composeCandidates) > 0
}

// NewDeploymentContext 创建不可变的部署发现结果。
func NewDeploymentContext(mode, composeRootSource string, confirmationRequired bool, candidates []DeploymentComposeCandidate, diagnostics []DeploymentDiagnostic) DeploymentContext {
	return DeploymentContext{mode: mode, composeRootSource: composeRootSource, composeConfirmationRequired: confirmationRequired, composeCandidates: append([]DeploymentComposeCandidate(nil), candidates...), diagnostics: append([]DeploymentDiagnostic(nil), diagnostics...)}
}

// DeploymentSnapshot 是当前运行时事实校验完成后冻结的不可变操作级部署上下文。
type DeploymentSnapshot struct {
	context     DeploymentContext
	candidate   DeploymentComposeCandidate
	fingerprint string
}

// Context 返回冻结时的不可变部署上下文。
func (s DeploymentSnapshot) Context() DeploymentContext { return s.context }

// Candidate 返回冻结时已经校验的 Compose 候选。
func (s DeploymentSnapshot) Candidate() DeploymentComposeCandidate { return s.candidate }

// Fingerprint 返回用于检测操作前后部署事实漂移的摘要。
func (s DeploymentSnapshot) Fingerprint() string { return s.fingerprint }

// NewDeploymentSnapshot 创建不可变的操作级部署快照。
func NewDeploymentSnapshot(context DeploymentContext, candidate DeploymentComposeCandidate, fingerprint string) DeploymentSnapshot {
	return DeploymentSnapshot{context: context, candidate: candidate, fingerprint: fingerprint}
}
