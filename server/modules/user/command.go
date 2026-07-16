package user

// CreateUserCommand 是创建受管理用户的业务输入；调用方负责在服务边界完成权限校验。
type CreateUserCommand struct {
	Username string
	Display  string
	Password string
	ActorID  uint64
}

// UpdateUserCommand 是更新受管理用户资料的业务输入。
type UpdateUserCommand struct {
	ID       uint64
	Username string
	Display  string
	ActorID  uint64
}

// UpdateUserStatusCommand 是更新受管理用户状态的业务输入；状态迁移规则由 user 服务统一执行。
type UpdateUserStatusCommand struct {
	ID      uint64
	Status  string
	ActorID uint64
}
