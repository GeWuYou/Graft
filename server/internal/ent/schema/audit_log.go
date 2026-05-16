package schema

import (
	"time"

	"entgo.io/ent"
	entsql "entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// AuditLog 定义当前 MVP 阶段的最小审计记录持久化模型。
//
// 该模型只收敛请求级自动审计和业务主动审计都需要的公共字段，不在 schema
// 层提前固化查询 DSL、归档分区或审计分析语义。
type AuditLog struct {
	ent.Schema
}

// Annotations 返回 audit_logs 表名映射。
func (AuditLog) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "audit_logs"},
	}
}

// Fields 返回最小审计记录字段定义。
func (AuditLog) Fields() []ent.Field {
	return []ent.Field{
		field.Uint64("operator_id").
			Optional().
			Nillable(),
		field.String("operator_name").
			Default(""),
		field.String("action").
			NotEmpty(),
		field.String("resource_type").
			Default(""),
		field.String("resource_id").
			Default(""),
		field.String("request_method").
			Default(""),
		field.String("request_path").
			Default(""),
		field.String("ip").
			Default(""),
		field.String("user_agent").
			Default(""),
		field.Bool("success").
			Default(false),
		field.String("error_message").
			Default(""),
		field.Time("created_at").
			Immutable().
			Default(time.Now),
	}
}

// Indexes 返回最小审计表当前需要的辅助索引。
func (AuditLog) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("created_at"),
		index.Fields("action"),
		index.Fields("operator_id"),
	}
}
