package build

import (
	"errors"
	"fmt"
	"strings"
)

var errBuildIntentUnavailable = errors.New("build template or driver is unavailable")

// TemplateVersion 是 Build-owned 的不可变模板版本描述。
type TemplateVersion struct {
	Ref         string
	Version     string
	DriverRefs  []string
	SourceKinds []string
}

// DriverVersion 是 Build-owned 的驱动版本及其模板兼容性描述。
type DriverVersion struct {
	Ref          string
	Version      string
	TemplateRefs []string
	Platforms    []string
}

// IntentResolver 解析并规范化模板版本与驱动版本的兼容关系。
type IntentResolver interface {
	ResolveBuildIntent(templateRef, driverRef string) (TemplateVersion, DriverVersion, error)
}

type buildIntentRegistry struct {
	templates map[string]TemplateVersion
	drivers   map[string]DriverVersion
	aliases   map[string]string
}

func newBuiltinBuildIntentRegistry() *buildIntentRegistry {
	r := &buildIntentRegistry{templates: make(map[string]TemplateVersion), drivers: make(map[string]DriverVersion), aliases: make(map[string]string)}
	template := TemplateVersion{Ref: "oci-dockerfile/default@v1", Version: "v1", DriverRefs: []string{"docker-engine@v1", "docker-buildx@v1"}, SourceKinds: []string{"application_workspace"}}
	driver := DriverVersion{Ref: "docker-engine@v1", Version: "v1", TemplateRefs: []string{template.Ref}, Platforms: []string{"linux/amd64"}}
	buildx := DriverVersion{Ref: "docker-buildx@v1", Version: "v1", TemplateRefs: []string{template.Ref}, Platforms: []string{"linux/amd64", "linux/arm64"}}
	r.templates[template.Ref], r.drivers[driver.Ref], r.drivers[buildx.Ref] = template, driver, buildx
	// 旧 API 引用是同一内置资源的公开别名；冻结计划只保存规范化后的版本引用。
	r.aliases["oci-dockerfile/default"], r.aliases["docker-engine"] = template.Ref, driver.Ref
	return r
}

func (r *buildIntentRegistry) ResolveBuildIntent(templateRef, driverRef string) (TemplateVersion, DriverVersion, error) {
	if r == nil {
		return TemplateVersion{}, DriverVersion{}, errBuildIntentUnavailable
	}
	templateRef, driverRef = r.canonical(templateRef), r.canonical(driverRef)
	template, templateOK := r.templates[templateRef]
	driver, driverOK := r.drivers[driverRef]
	if !templateOK || !driverOK || !containsString(template.DriverRefs, driver.Ref) || !containsString(driver.TemplateRefs, template.Ref) {
		return TemplateVersion{}, DriverVersion{}, fmt.Errorf("%w: template %q and driver %q are incompatible", errBuildIntentUnavailable, templateRef, driverRef)
	}
	return template, driver, nil
}

func (r *buildIntentRegistry) canonical(ref string) string {
	ref = strings.TrimSpace(ref)
	if canonical, ok := r.aliases[ref]; ok {
		return canonical
	}
	return ref
}

func containsString(values []string, wanted string) bool {
	for _, value := range values {
		if value == wanted {
			return true
		}
	}
	return false
}
