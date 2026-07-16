// Package openapi 保存由仓库根 OpenAPI 规范生成的非运行时 Go 契约产物。
//
// 该边界与模块运行时装配、手写 HTTP DTO 真相和 handler 生命周期所有权隔离。这里的生成代码只能用于编译、测试或契约对比，
// 未经单独批准不得成为隐式运行时真相。
package openapi
