package schema

import (
	"time"

	"entgo.io/ent"
	entsql "entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// AuthPersonalAccessToken 定义用户为 MCP 等受控自动化客户端签发的个人 API Token。
type AuthPersonalAccessToken struct {
	ent.Schema
}

// Annotations 返回个人 API Token 表映射及数据库注释配置。
func (AuthPersonalAccessToken) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "auth_personal_access_tokens"},
		schema.Comment("个人 API 令牌表（认证模块）"),
		entsql.WithComments(true),
	}
}

// Fields 返回个人 API Token 的验证、作用域与生命周期字段。
func (AuthPersonalAccessToken) Fields() []ent.Field {
	return []ent.Field{
		field.Uint64("user_id").
			Comment("关联用户资料的稳定标识").
			Immutable(),
		field.String("name").
			Comment("用户标记令牌用途的名称").
			NotEmpty(),
		field.String("token_prefix").
			Comment("仅用于识别令牌的公开前缀，不包含完整密钥").
			NotEmpty().
			Immutable(),
		field.String("secret_hash").
			Comment("个人 API 令牌明文的 SHA-256 摘要").
			NotEmpty().
			Unique().
			Immutable().
			Sensitive(),
		field.JSON("scopes", []string{}).
			Comment("允许调用的精确权限代码列表，只能收窄用户 RBAC 权限"),
		field.Time("expires_at").
			Comment("个人 API 令牌失效时间"),
		field.Time("revoked_at").
			Comment("令牌撤销时间，为空表示未主动撤销").
			Optional().
			Nillable(),
		field.Time("last_used_at").
			Comment("最近一次通过令牌完成认证的时间").
			Optional().
			Nillable(),
		field.Time("created_at").
			Comment("令牌创建时间").
			Immutable().
			Default(time.Now),
		field.Time("updated_at").
			Comment("令牌最近更新时间").
			Default(time.Now).
			UpdateDefault(time.Now),
		field.Int64("deleted_at").
			Comment("软删除时间戳，0 表示当前记录仍有效").
			Default(0),
	}
}

// Indexes 为认证查找和所有者列表查询声明所需索引。
func (AuthPersonalAccessToken) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("secret_hash").Unique(),
		index.Fields("user_id", "deleted_at", "created_at"),
	}
}
