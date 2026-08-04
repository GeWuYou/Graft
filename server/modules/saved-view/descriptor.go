// Package savedview 提供分页列表使用的通用私有保存视图能力；模块只保存消费者状态，不解释查询语义。
package savedview

import (
	"database/sql"
	"fmt"

	"graft/server/internal/module"
	"graft/server/modules/saved-view/store"
)

const moduleID = "saved-view"

// NewModuleSpec 创建 savedview 模块的编译期描述符。
func NewModuleSpec() module.Spec {
	return module.Spec{
		ID:            moduleID,
		Dependencies:  nil,
		MigrationPath: []string{"modules/saved-view/migrations"},
		Builder: module.BuilderFunc(func(ctx module.BuildContext) (module.Module, error) {
			db, err := module.ResolveService[*sql.DB](ctx.Services, (*sql.DB)(nil))
			if err != nil {
				return nil, fmt.Errorf("resolve sql db: %w", err)
			}
			repository, err := store.NewSQLRepository(db)
			if err != nil {
				return nil, fmt.Errorf("build saved view repository: %w", err)
			}
			return NewModule(NewService(repository)), nil
		}),
	}
}
