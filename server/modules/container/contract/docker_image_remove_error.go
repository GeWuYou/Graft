package contract

// DockerImageRemoveErrorCode 表示 Docker 镜像删除逐项结果的稳定错误码。
// 该契约由 Docker daemon 的实际响应驱动，调用方不得根据镜像列表快照推断失败原因。
type DockerImageRemoveErrorCode string

// String 返回供 HTTP 响应、审计与客户端映射使用的稳定错误码值。
func (c DockerImageRemoveErrorCode) String() string {
	return string(c)
}

const (
	// DockerImageMultipleTagsError 表示同一 Image ID 仍被多个 Repository:Tag 引用。
	DockerImageMultipleTagsError DockerImageRemoveErrorCode = "IMAGE_REFERENCED_BY_MULTIPLE_TAGS"
	// DockerImageInUseError 表示镜像仍被一个或多个容器引用。
	DockerImageInUseError DockerImageRemoveErrorCode = "IMAGE_IN_USE"
	// DockerImageNotFoundError 表示删除时 Docker daemon 未找到指定镜像。
	DockerImageNotFoundError DockerImageRemoveErrorCode = "IMAGE_NOT_FOUND"
	// DockerRuntimeUnavailable 表示 Docker runtime 或其 socket 当前不可用。
	DockerRuntimeUnavailable DockerImageRemoveErrorCode = "DOCKER_RUNTIME_UNAVAILABLE"
	// DockerTimeout 表示 Docker 删除操作在运行时边界超时。
	DockerTimeout DockerImageRemoveErrorCode = "DOCKER_TIMEOUT"
	// DockerCommunicationError 表示与 Docker daemon 的通信未完成。
	DockerCommunicationError DockerImageRemoveErrorCode = "DOCKER_COMMUNICATION_ERROR"
	// DockerImageRemoveUnknown 表示无法安全归类的 Docker 删除失败。
	DockerImageRemoveUnknown DockerImageRemoveErrorCode = "UNKNOWN"
)
