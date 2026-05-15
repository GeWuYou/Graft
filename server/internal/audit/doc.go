// Package audit 提供审计记录写入的最小应用层封装。
//
// 该包负责把 HTTP 中间件和 event bus 路径产生的审计输入收敛为统一写入
// 行为，但不承担路由装配、权限、查询 DSL 或归档策略。
package audit
