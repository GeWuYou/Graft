package healthopenapi

// ServerInterface is the minimal generated handler contract for the core healthz route.
type ServerInterface interface {
	// GetHealthz 处理核心 healthz 探针，并保持运行时健康检查的最小生成契约边界。
	GetHealthz()
}
