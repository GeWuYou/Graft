package storeent

import (
	"database/sql"
	"fmt"
	"strings"

	entsql "entgo.io/ent/dialect/sql"
	"go.uber.org/zap"

	"graft/server/internal/logger"
	ent "graft/server/modules/user/ent"
)

// Runtime 持有单进程内 user 模块共享 Ent 客户端的装配关系。
//
// 客户端复用 core 拥有的 *sql.DB 连接池，连接池关闭责任仍归 core；本运行时只构建模块本地 Ent 表面。
type Runtime struct {
	client *ent.Client
	db     *sql.DB
}

// NewRuntime 基于共享 SQL 连接池构建 user 模块的 Ent 运行时。
func NewRuntime(sqlDB *sql.DB, runtimeLogger *zap.Logger) (*Runtime, error) {
	if sqlDB == nil {
		return nil, fmt.Errorf("user storeent runtime requires a non-nil sql db")
	}
	if runtimeLogger == nil {
		return nil, fmt.Errorf("user storeent runtime requires a non-nil logger")
	}

	driver := entsql.OpenDB("postgres", sqlDB)
	categoryLog := logger.Category(runtimeLogger, logger.CategoryDatabaseEnt)
	return &Runtime{
		db: sqlDB,
		client: ent.NewClient(
			ent.Driver(driver),
			ent.Log(func(args ...any) {
				if !categoryLog.Enabled(logger.TraceLevel) {
					return
				}
				message := strings.TrimSpace(fmt.Sprint(args...))
				if message == "" {
					return
				}

				categoryLog.Trace("ent debug",
					zap.String("module", "user"),
					zap.String("component", "ent"),
					zap.String("message", message),
				)
			}),
		),
	}, nil
}

// NewUserRepository 基于共享 Ent 客户端构建模块拥有的用户仓储。
func (r *Runtime) NewUserRepository() (*userRepository, error) {
	return newUserRepository(r.client, r.db)
}

// Client 返回共享 Ent 客户端，仅供仍需直接访问客户端的受限模块内部场景使用。
func (r *Runtime) Client() *ent.Client {
	return r.client
}
