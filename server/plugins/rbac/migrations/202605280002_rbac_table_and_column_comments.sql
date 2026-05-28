COMMENT ON TABLE "roles" IS '角色信息表（RBAC 插件）';
COMMENT ON COLUMN "roles"."id" IS '主键 ID';
COMMENT ON COLUMN "roles"."name" IS '角色标识名称，用于唯一识别角色';
COMMENT ON COLUMN "roles"."display" IS '角色显示名称';
COMMENT ON COLUMN "roles"."description" IS '角色描述';
COMMENT ON COLUMN "roles"."builtin" IS '是否为系统内置角色';
COMMENT ON COLUMN "roles"."created_at" IS '创建时间';
COMMENT ON COLUMN "roles"."updated_at" IS '更新时间';
COMMENT ON COLUMN "roles"."created_by" IS '创建人用户 ID，0 表示系统';
COMMENT ON COLUMN "roles"."updated_by" IS '最后更新人用户 ID，0 表示系统';
COMMENT ON COLUMN "roles"."deleted_at" IS '软删除时间戳，0 表示未删除';
COMMENT ON COLUMN "roles"."deleted_by" IS '删除人用户 ID，0 表示未删除';

COMMENT ON TABLE "permissions" IS '权限点信息表（RBAC 插件）';
COMMENT ON COLUMN "permissions"."id" IS '主键 ID';
COMMENT ON COLUMN "permissions"."code" IS '权限点编码，采用点分层级格式';
COMMENT ON COLUMN "permissions"."display" IS '权限点显示名称';
COMMENT ON COLUMN "permissions"."description" IS '权限点描述';
COMMENT ON COLUMN "permissions"."category" IS '权限类别：api 表示接口权限';
COMMENT ON COLUMN "permissions"."created_at" IS '创建时间';
COMMENT ON COLUMN "permissions"."updated_at" IS '更新时间';
COMMENT ON COLUMN "permissions"."created_by" IS '创建人用户 ID，0 表示系统';
COMMENT ON COLUMN "permissions"."updated_by" IS '最后更新人用户 ID，0 表示系统';
COMMENT ON COLUMN "permissions"."deleted_at" IS '软删除时间戳，0 表示未删除';
COMMENT ON COLUMN "permissions"."deleted_by" IS '删除人用户 ID，0 表示未删除';

COMMENT ON TABLE "user_roles" IS '用户与角色关联表（RBAC 插件）';
COMMENT ON COLUMN "user_roles"."id" IS '主键 ID';
COMMENT ON COLUMN "user_roles"."created_at" IS '创建时间';
COMMENT ON COLUMN "user_roles"."role_id" IS '角色 ID';
COMMENT ON COLUMN "user_roles"."user_id" IS '用户 ID';

COMMENT ON TABLE "role_permissions" IS '角色与权限关联表（RBAC 插件）';
COMMENT ON COLUMN "role_permissions"."id" IS '主键 ID';
COMMENT ON COLUMN "role_permissions"."created_at" IS '创建时间';
COMMENT ON COLUMN "role_permissions"."permission_id" IS '权限 ID';
COMMENT ON COLUMN "role_permissions"."role_id" IS '角色 ID';
