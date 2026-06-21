package i18n

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"path"
	"slices"
	"strings"

	"gopkg.in/yaml.v3"
)

// EmbeddedLocaleResource carries owner-declared raw locale data while keeping
// YAML parsing and registry validation inside server/internal/i18n.
type EmbeddedLocaleResource struct {
	Namespace Namespace
	Locale    LocaleTag
	Source    string
	Data      []byte
}

var localeResourcePatterns = [...]string{
	"locales/*.yaml",
	"locales/modules/*.yaml",
}

const yamlMappingPairWidth = 2

func (s *Service) registerEmbeddedCatalogs() error {
	return s.registerLocaleResources(embeddedLocaleFS)
}

// EmbeddedLocaleResources returns deferred embedded locale resources that
// runtime should pre-register before module Register.
func EmbeddedLocaleResources() ([]EmbeddedLocaleResource, error) {
	return loadEmbeddedLocaleResources(embeddedLocaleFS)
}

// EmbeddedLocaleResourcesFromFS converts owner-local {locale}.yaml resources
// into read-only descriptors without exposing loader internals outside i18n.
func EmbeddedLocaleResourcesFromFS(fsys fs.FS, namespace Namespace) ([]EmbeddedLocaleResource, error) {
	namespace = Namespace(strings.TrimSpace(string(namespace)))
	if namespace == "" {
		return nil, errors.New("embedded locale resource namespace is required")
	}
	if fsys == nil {
		return nil, nil
	}

	matches, err := fs.Glob(fsys, "*.yaml")
	if err != nil {
		return nil, fmt.Errorf("glob owner-local locale resources: %w", err)
	}
	slices.Sort(matches)

	resources := make([]EmbeddedLocaleResource, 0, len(matches))
	for _, name := range matches {
		locale, err := parseOwnerLocaleResourceName(name)
		if err != nil {
			return nil, fmt.Errorf("parse owner-local locale resource %q: %w", name, err)
		}
		data, err := fs.ReadFile(fsys, name)
		if err != nil {
			return nil, fmt.Errorf("read owner-local locale resource %q: %w", name, err)
		}
		resources = append(resources, EmbeddedLocaleResource{
			Namespace: namespace,
			Locale:    locale,
			Source:    name,
			Data:      append([]byte(nil), data...),
		})
	}

	return resources, nil
}

// RegisterEmbeddedLocaleResources registers raw embedded locale resources while
// reusing the canonical RegisterMessages validation path.
func (s *Service) RegisterEmbeddedLocaleResources(resources []EmbeddedLocaleResource) error {
	registrations, err := loadEmbeddedLocaleRegistrations(resources)
	if err != nil {
		return err
	}
	return s.registerRegistrations(registrations)
}

func (s *Service) registerLocaleResources(fsys fs.FS) error {
	registrations, err := loadLocaleRegistrations(fsys)
	if err != nil {
		return err
	}

	return s.registerRegistrations(registrations)
}

func (s *Service) registerRegistrations(registrations []Registration) error {
	for _, registration := range registrations {
		if err := s.RegisterMessages(registration); err != nil {
			return fmt.Errorf(
				"register locale resource %q for %q: %w",
				registration.Namespace,
				registration.Locale,
				err,
			)
		}
	}

	return nil
}

func loadEmbeddedLocaleResources(fsys fs.FS) ([]EmbeddedLocaleResource, error) {
	if fsys == nil {
		return nil, nil
	}

	matches, err := fs.Glob(fsys, "locales/modules/*.yaml")
	if err != nil {
		return nil, fmt.Errorf("glob embedded module locale resources: %w", err)
	}
	slices.Sort(matches)

	resources := make([]EmbeddedLocaleResource, 0, len(matches))
	for _, name := range matches {
		registration, err := loadLocaleRegistration(fsys, name)
		if err != nil {
			return nil, err
		}
		data, err := fs.ReadFile(fsys, name)
		if err != nil {
			return nil, fmt.Errorf("read locale resource %q: %w", name, err)
		}
		resources = append(resources, EmbeddedLocaleResource{
			Namespace: registration.Namespace,
			Locale:    registration.Locale,
			Source:    name,
			Data:      append([]byte(nil), data...),
		})
	}

	return resources, nil
}

func loadEmbeddedLocaleRegistrations(resources []EmbeddedLocaleResource) ([]Registration, error) {
	registrations := make([]Registration, 0, len(resources))
	for _, resource := range resources {
		source := strings.TrimSpace(resource.Source)
		if source == "" {
			source = fmt.Sprintf("%s.%s", resource.Namespace, resource.Locale)
		}
		namespace := Namespace(strings.TrimSpace(string(resource.Namespace)))
		if namespace == "" {
			return nil, fmt.Errorf("embedded locale resource %q: namespace is required", source)
		}
		locale := LocaleTag(strings.TrimSpace(string(resource.Locale)))
		if locale == "" {
			return nil, fmt.Errorf("embedded locale resource %q: locale is required", source)
		}

		messages, err := parseFlatYAMLMessages(source, resource.Data)
		if err != nil {
			return nil, err
		}
		registrations = append(registrations, Registration{
			Namespace: namespace,
			Locale:    locale,
			Messages:  messages,
		})
	}

	return registrations, nil
}

func loadLocaleRegistrations(fsys fs.FS) ([]Registration, error) {
	if fsys == nil {
		return nil, nil
	}

	var matches []string
	for _, pattern := range localeResourcePatterns {
		patternMatches, err := fs.Glob(fsys, pattern)
		if err != nil {
			return nil, fmt.Errorf("glob locale resources %q: %w", pattern, err)
		}
		matches = append(matches, patternMatches...)
	}
	slices.Sort(matches)

	registrations := make([]Registration, 0, len(matches))
	for _, name := range matches {
		registration, err := loadLocaleRegistration(fsys, name)
		if err != nil {
			return nil, err
		}

		registrations = append(registrations, registration)
	}

	return registrations, nil
}

func loadLocaleRegistration(fsys fs.FS, resourcePath string) (Registration, error) {
	namespace, locale, err := parseLocaleResourceName(path.Base(resourcePath))
	if err != nil {
		return Registration{}, fmt.Errorf("parse locale resource %q: %w", resourcePath, err)
	}

	content, err := fs.ReadFile(fsys, resourcePath)
	if err != nil {
		return Registration{}, fmt.Errorf("read locale resource %q: %w", resourcePath, err)
	}

	messages, err := parseFlatYAMLMessages(resourcePath, content)
	if err != nil {
		return Registration{}, err
	}

	return Registration{
		Namespace: namespace,
		Locale:    locale,
		Messages:  messages,
	}, nil
}

func parseLocaleResourceName(filename string) (Namespace, LocaleTag, error) {
	name := strings.TrimSpace(filename)
	if name == "" {
		return "", "", errors.New("locale resource filename is required")
	}
	if !strings.HasSuffix(name, ".yaml") {
		return "", "", fmt.Errorf("locale resource %q must end with .yaml", name)
	}

	stem := strings.TrimSuffix(name, ".yaml")
	separator := strings.LastIndex(stem, ".")
	if separator <= 0 || separator == len(stem)-1 {
		return "", "", fmt.Errorf("locale resource %q must match {namespace}.{locale}.yaml", name)
	}

	namespace := Namespace(strings.TrimSpace(stem[:separator]))
	locale := LocaleTag(strings.TrimSpace(stem[separator+1:]))
	if namespace == "" || locale == "" {
		return "", "", fmt.Errorf("locale resource %q must match {namespace}.{locale}.yaml", name)
	}

	return namespace, locale, nil
}

func parseOwnerLocaleResourceName(filename string) (LocaleTag, error) {
	name := strings.TrimSpace(filename)
	if name == "" {
		return "", errors.New("locale resource filename is required")
	}
	if !strings.HasSuffix(name, ".yaml") {
		return "", fmt.Errorf("locale resource %q must end with .yaml", name)
	}

	locale := strings.TrimSpace(strings.TrimSuffix(name, ".yaml"))
	if locale == "" {
		return "", fmt.Errorf("locale resource %q must match {locale}.yaml", name)
	}

	return LocaleTag(locale), nil
}

func parseFlatYAMLMessages(resourcePath string, content []byte) ([]MessageResource, error) {
	root, err := decodeLocaleDocument(resourcePath, content)
	if err != nil {
		return nil, err
	}
	if root == nil {
		return nil, nil
	}
	if root.Kind != yaml.MappingNode {
		return nil, fmt.Errorf("locale resource %q must be a flat key-value mapping", resourcePath)
	}

	return collectFlatYAMLMessages(resourcePath, root)
}

func decodeLocaleDocument(resourcePath string, content []byte) (*yaml.Node, error) {
	var document yaml.Node
	decoder := yaml.NewDecoder(bytes.NewReader(content))
	if err := decoder.Decode(&document); err != nil {
		if errors.Is(err, io.EOF) {
			return nil, nil
		}
		return nil, fmt.Errorf("decode locale resource %q: %w", resourcePath, err)
	}

	root := &document
	if document.Kind == yaml.DocumentNode {
		if len(document.Content) == 0 {
			return nil, nil
		}
		root = document.Content[0]
	}
	if root.Kind == 0 {
		return nil, nil
	}

	return root, nil
}

func collectFlatYAMLMessages(resourcePath string, root *yaml.Node) ([]MessageResource, error) {
	if root == nil {
		return nil, fmt.Errorf("locale resource %q must be a flat key-value mapping", resourcePath)
	}
	if len(root.Content)%yamlMappingPairWidth != 0 {
		return nil, fmt.Errorf("locale resource %q contains a malformed mapping entry", resourcePath)
	}

	seenKeys := make(map[string]struct{}, len(root.Content)/yamlMappingPairWidth)
	messages := make([]MessageResource, 0, len(root.Content)/yamlMappingPairWidth)
	for index := 0; index < len(root.Content); index += yamlMappingPairWidth {
		message, err := parseFlatYAMLPair(resourcePath, root.Content[index], root.Content[index+1], seenKeys)
		if err != nil {
			return nil, err
		}
		messages = append(messages, message)
	}

	return messages, nil
}

func parseFlatYAMLPair(
	resourcePath string,
	keyNode *yaml.Node,
	valueNode *yaml.Node,
	seenKeys map[string]struct{},
) (MessageResource, error) {
	if keyNode.Kind != yaml.ScalarNode {
		return MessageResource{}, fmt.Errorf("locale resource %q contains a non-scalar key", resourcePath)
	}

	key := strings.TrimSpace(keyNode.Value)
	if key == "" {
		return MessageResource{}, fmt.Errorf("locale resource %q contains an empty message key", resourcePath)
	}
	if _, exists := seenKeys[key]; exists {
		return MessageResource{}, fmt.Errorf("locale resource %q contains a duplicate message key %q", resourcePath, key)
	}
	if valueNode.Kind != yaml.ScalarNode {
		return MessageResource{}, fmt.Errorf("locale resource %q key %q must map to a scalar string", resourcePath, key)
	}
	if strings.TrimSpace(valueNode.Value) == "" {
		return MessageResource{}, fmt.Errorf("locale resource %q key %q must not have an empty message text", resourcePath, key)
	}

	seenKeys[key] = struct{}{}
	return MessageResource{
		Key:  MessageKey(key),
		Text: valueNode.Value,
	}, nil
}
