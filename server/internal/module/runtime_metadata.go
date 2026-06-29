package module

import "graft/server/internal/buildinfo"

// DescriptorSnapshot 是暴露给运行时模块消费的稳定描述符元数据快照。
//
// 它只包含模块运行期观测需要的 canonical metadata，避免模块直接依赖
// compile-time registry 或构造器内部实现。
type DescriptorSnapshot struct {
	Name      string
	DependsOn []string
}

// RuntimeMetadata 暴露 core 运行时编排后可安全共享给模块的元数据表面。
//
// 当前只承载按 canonical 依赖顺序排列的模块描述符快照，供模块进行
// 观测或诊断，不提供 registry 级构造能力。
type RuntimeMetadata struct {
	orderedModuleDescriptors []DescriptorSnapshot
	buildInfo                buildinfo.Info
}

// NewRuntimeMetadata 根据模块定义和当前构建信息构造 RuntimeMetadata，并规范化构建标识。
// descriptors 提供模块定义；currentBuildInfo 提供当前构建信息。
func NewRuntimeMetadata(descriptors []Spec, currentBuildInfo buildinfo.Info) RuntimeMetadata {
	return RuntimeMetadata{
		orderedModuleDescriptors: collectDescriptorSnapshots(descriptors, func(descriptor Spec) (string, []string) {
			return descriptor.Name(), descriptor.DependsOn()
		}),
		buildInfo: buildinfo.Normalize(currentBuildInfo),
	}
}

// OrderedModuleDescriptors 返回运行时可见的 canonical 有序描述符快照。
func (m RuntimeMetadata) OrderedModuleDescriptors() []DescriptorSnapshot {
	return collectDescriptorSnapshots(m.orderedModuleDescriptors, func(descriptor DescriptorSnapshot) (string, []string) {
		return descriptor.Name, descriptor.DependsOn
	})
}

// BuildInfo 返回运行时可见的 canonical 构建身份快照。
func (m RuntimeMetadata) BuildInfo() buildinfo.Info {
	return buildinfo.Normalize(m.buildInfo)
}

// newDescriptorSnapshot 创建一个 DescriptorSnapshot，并复制依赖项列表以避免共享底层数组。
func newDescriptorSnapshot(name string, dependsOn []string) DescriptorSnapshot {
	return DescriptorSnapshot{
		Name:      name,
		DependsOn: append([]string(nil), dependsOn...),
	}
}

// project 用于从每个输入对象提取名称和依赖项。
func collectDescriptorSnapshots[T any](
	descriptors []T,
	project func(T) (string, []string),
) []DescriptorSnapshot {
	snapshots := make([]DescriptorSnapshot, 0, len(descriptors))
	for _, descriptor := range descriptors {
		name, dependsOn := project(descriptor)
		snapshots = append(snapshots, newDescriptorSnapshot(name, dependsOn))
	}
	return snapshots
}
