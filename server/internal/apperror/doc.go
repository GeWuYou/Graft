// Package apperror 定义服务端内部错误分类、公开展示元数据与单次记录标记。
//
// 该包不负责 HTTP 写出或日志输出；调用方应分别通过 httpx 与 logger 边界
// 消费同一个 error chain，避免把 transport 或 observability 职责混入控制流。
package apperror
