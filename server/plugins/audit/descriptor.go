package audit

import "graft/server/internal/plugin"

// NewDescriptor exposes the audit plugin's stable metadata and builder.
func NewDescriptor() plugin.Descriptor {
	instance, err := NewPlugin(nil)
	if err != nil {
		instance = &Plugin{}
	}

	return plugin.Descriptor{
		ID:            instance.Name(),
		PluginVersion: instance.Version(),
		Dependencies:  append([]string(nil), instance.DependsOn()...),
		Builder: plugin.BuilderFunc(func(ctx plugin.BuildContext) (plugin.Plugin, error) {
			return NewPlugin(ctx.Stores.Audit())
		}),
	}
}
