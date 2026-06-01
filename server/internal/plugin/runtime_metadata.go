package plugin

// DescriptorSnapshot 是暴露给运行时模块消费的稳定描述符元数据快照。
//
// 它只包含模块运行期观测需要的 canonical metadata，避免模块直接依赖
// compile-time registry 或构造器内部实现。
type DescriptorSnapshot struct {
	Name      string
	Version   string
	DependsOn []string
}

// RuntimeMetadata 暴露 core 运行时编排后可安全共享给模块的元数据表面。
//
// 当前只承载按 canonical 依赖顺序排列的模块描述符快照，供模块进行
// 观测或诊断，不提供 registry 级构造能力。
type RuntimeMetadata struct {
	orderedModuleDescriptors []DescriptorSnapshot
}

// NewRuntimeMetadata 从有序描述符集合构造运行时元数据快照。
func NewRuntimeMetadata(descriptors []Descriptor) RuntimeMetadata {
	snapshots := make([]DescriptorSnapshot, 0, len(descriptors))
	for _, descriptor := range descriptors {
		snapshots = append(snapshots, DescriptorSnapshot{
			Name:      descriptor.Name(),
			Version:   descriptor.Version(),
			DependsOn: append([]string(nil), descriptor.DependsOn()...),
		})
	}

	return RuntimeMetadata{orderedModuleDescriptors: snapshots}
}

// OrderedPluginDescriptors 返回运行时可见的 canonical 有序描述符快照。
func (m RuntimeMetadata) OrderedPluginDescriptors() []DescriptorSnapshot {
	snapshots := make([]DescriptorSnapshot, 0, len(m.orderedModuleDescriptors))
	for _, descriptor := range m.orderedModuleDescriptors {
		snapshots = append(snapshots, DescriptorSnapshot{
			Name:      descriptor.Name,
			Version:   descriptor.Version,
			DependsOn: append([]string(nil), descriptor.DependsOn...),
		})
	}

	return snapshots
}
