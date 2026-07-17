package contract

// ApplicationTemplateDefinitionSchemaVersionCurrent 是当前通用模板定义快照版本。
// 适配器负责解释 definition，而不是让通用表暴露 Compose 或 Provider 专属列。
const ApplicationTemplateDefinitionSchemaVersionCurrent = 1

// ApplicationTemplateCategory 是模板目录使用的受控分类。
type ApplicationTemplateCategory string

// String 返回数据库与 HTTP 契约使用的稳定分类值。
func (value ApplicationTemplateCategory) String() string { return string(value) }

const (
	// ApplicationTemplateCategoryDatabase 表示数据库模板。
	ApplicationTemplateCategoryDatabase ApplicationTemplateCategory = "database"
	// ApplicationTemplateCategoryCache 表示缓存模板。
	ApplicationTemplateCategoryCache ApplicationTemplateCategory = "cache"
	// ApplicationTemplateCategoryMQ 表示消息队列模板。
	ApplicationTemplateCategoryMQ ApplicationTemplateCategory = "mq"
	// ApplicationTemplateCategoryProxy 表示代理模板。
	ApplicationTemplateCategoryProxy ApplicationTemplateCategory = "proxy"
	// ApplicationTemplateCategoryStorage 表示存储模板。
	ApplicationTemplateCategoryStorage ApplicationTemplateCategory = "storage"
	// ApplicationTemplateCategoryMonitoring 表示监控模板。
	ApplicationTemplateCategoryMonitoring ApplicationTemplateCategory = "monitoring"
	// ApplicationTemplateCategoryLogging 表示日志模板。
	ApplicationTemplateCategoryLogging ApplicationTemplateCategory = "logging"
	// ApplicationTemplateCategoryCICD 表示持续集成与交付模板。
	ApplicationTemplateCategoryCICD ApplicationTemplateCategory = "cicd"
	// ApplicationTemplateCategoryAI 表示 AI 服务模板。
	ApplicationTemplateCategoryAI ApplicationTemplateCategory = "ai"
	// ApplicationTemplateCategoryOther 表示无法归入其它受控类别的模板。
	ApplicationTemplateCategoryOther ApplicationTemplateCategory = "other"
)

// Valid 报告分类是否属于稳定的目录枚举。
func (value ApplicationTemplateCategory) Valid() bool {
	switch value {
	case ApplicationTemplateCategoryDatabase, ApplicationTemplateCategoryCache, ApplicationTemplateCategoryMQ,
		ApplicationTemplateCategoryProxy, ApplicationTemplateCategoryStorage, ApplicationTemplateCategoryMonitoring,
		ApplicationTemplateCategoryLogging, ApplicationTemplateCategoryCICD, ApplicationTemplateCategoryAI,
		ApplicationTemplateCategoryOther:
		return true
	default:
		return false
	}
}
