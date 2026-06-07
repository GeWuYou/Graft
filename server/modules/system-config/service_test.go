package systemconfig

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"graft/server/internal/configregistry"
	systemconfigstore "graft/server/modules/system-config/store"
)

func TestServiceMasksSensitiveValues(t *testing.T) {
	service := newTestService(t, configregistry.Definition{
		Key:          "auth.jwt_secret",
		Module:       "auth",
		Group:        "security",
		Title:        "JWT Secret",
		Type:         configregistry.ValueTypeString,
		DefaultValue: json.RawMessage(`"secret"`),
		Sensitive:    true,
	})

	item, err := service.Get(context.Background(), "auth.jwt_secret")
	if err != nil {
		t.Fatalf("get sensitive config: %v", err)
	}
	if !item.Masked || item.EffectiveValue != nil || item.DefaultValue != nil {
		t.Fatalf("sensitive values must be masked, got %#v", item)
	}
}

func TestServiceListsDefaultsAndStoresOverridesOnly(t *testing.T) {
	repo := newMemoryRepo()
	service := newTestServiceWithRepo(t, repo, configregistry.Definition{
		Key:          "scheduler.timeout",
		Module:       "scheduler",
		Group:        "runtime",
		Title:        "Timeout",
		Type:         configregistry.ValueTypeString,
		DefaultValue: json.RawMessage(`"30s"`),
	})

	assertDefaultVisibleWithoutOverride(t, service, repo)
	assertUpdateStoresOneOverride(t, service, repo)
	assertResetDeletesOverride(t, service, repo)
}

func TestServiceResolveDefaultConfigReturnsEffectiveOverride(t *testing.T) {
	repo := newMemoryRepo()
	service := newTestServiceWithRepo(t, repo, configregistry.Definition{
		Key:          "httpx.access-log-retention-cleanup",
		Module:       "core.httpx",
		Group:        "log.retention",
		Title:        "Access log retention cleanup",
		Type:         configregistry.ValueTypeObject,
		DefaultValue: json.RawMessage(`{"retentionDays":30,"batchSize":1000}`),
	})
	if _, err := service.Update(
		context.Background(),
		"httpx.access-log-retention-cleanup",
		json.RawMessage(`{"retentionDays":45,"batchSize":2000}`),
	); err != nil {
		t.Fatalf("update override: %v", err)
	}

	value, err := service.ResolveDefaultConfig(context.Background(), "httpx.access-log-retention-cleanup")
	if err != nil {
		t.Fatalf("resolve default config: %v", err)
	}
	if value != `{"retentionDays":45,"batchSize":2000}` {
		t.Fatalf("expected effective override, got %s", value)
	}
}

func TestServiceResolveDefaultConfigRejectsSensitiveDefinitions(t *testing.T) {
	service := newTestService(t, configregistry.Definition{
		Key:          "auth.jwt_secret",
		Module:       "auth",
		Group:        "security",
		Title:        "JWT Secret",
		Type:         configregistry.ValueTypeString,
		DefaultValue: json.RawMessage(`"secret"`),
		Sensitive:    true,
	})

	if _, err := service.ResolveDefaultConfig(context.Background(), "auth.jwt_secret"); !errors.Is(err, errSensitiveConfig) {
		t.Fatalf("expected sensitive default config error, got %v", err)
	}
}

func assertDefaultVisibleWithoutOverride(t *testing.T, service *Service, repo *memoryRepo) {
	t.Helper()

	items, err := service.List(context.Background())
	if err != nil {
		t.Fatalf("list default config: %v", err)
	}
	if len(items) != 1 || items[0].HasOverride || string(items[0].EffectiveValue) != `"30s"` {
		t.Fatalf("expected listed module default without override, got %#v", items)
	}
	if len(repo.values) != 0 {
		t.Fatalf("expected list to avoid copying defaults into overrides, got %d rows", len(repo.values))
	}

	item, err := service.Get(context.Background(), "scheduler.timeout")
	if err != nil {
		t.Fatalf("get default config: %v", err)
	}
	if item.HasOverride || string(item.EffectiveValue) != `"30s"` {
		t.Fatalf("expected get to return module default without override, got %#v", item)
	}
}

func assertUpdateStoresOneOverride(t *testing.T, service *Service, repo *memoryRepo) {
	t.Helper()

	item, err := service.Update(context.Background(), "scheduler.timeout", json.RawMessage(`"60s"`))
	if err != nil {
		t.Fatalf("update override: %v", err)
	}
	if !item.HasOverride || string(item.EffectiveValue) != `"60s"` {
		t.Fatalf("expected effective override, got %#v", item)
	}
	if len(repo.values) != 1 {
		t.Fatalf("expected only one override row, got %d", len(repo.values))
	}
}

func assertResetDeletesOverride(t *testing.T, service *Service, repo *memoryRepo) {
	t.Helper()

	item, err := service.Reset(context.Background(), "scheduler.timeout")
	if err != nil {
		t.Fatalf("reset override: %v", err)
	}
	if item.HasOverride || string(item.EffectiveValue) != `"30s"` {
		t.Fatalf("expected reset to return module default without override, got %#v", item)
	}
	if len(repo.values) != 0 {
		t.Fatalf("expected reset to delete override row, got %d rows", len(repo.values))
	}
}

func TestServiceRejectsMismatchedValueType(t *testing.T) {
	service := newTestService(t, configregistry.Definition{
		Key:          "audit.retention_days",
		Module:       "audit",
		Group:        "retention",
		Title:        "Retention Days",
		Type:         configregistry.ValueTypeInteger,
		DefaultValue: json.RawMessage(`30`),
	})

	if _, err := service.Update(context.Background(), "audit.retention_days", json.RawMessage(`"30"`)); err == nil {
		t.Fatal("expected value type error")
	}
}

func TestServiceRejectsObjectValueOutsideSchemaConstraints(t *testing.T) {
	service := newTestService(t, configregistry.Definition{
		Key:          "httpx.access-log-retention-cleanup",
		Module:       "core.httpx",
		Group:        "log.retention",
		Title:        "Access Log Retention Cleanup",
		Type:         configregistry.ValueTypeObject,
		Schema:       json.RawMessage(`{"type":"object","properties":{"retentionDays":{"type":"integer","minimum":1,"maximum":365},"batchSize":{"type":"integer","minimum":1,"maximum":10000}},"additionalProperties":false}`),
		DefaultValue: json.RawMessage(`{"retentionDays":30,"batchSize":1000}`),
	})

	if _, err := service.Update(context.Background(), "httpx.access-log-retention-cleanup", json.RawMessage(`{"retentionDays":366,"batchSize":1000}`)); err == nil {
		t.Fatal("expected schema validation error")
	}
	if _, err := service.Update(context.Background(), "httpx.access-log-retention-cleanup", json.RawMessage(`{"retentionDays":30,"batchSize":1000,"extra":true}`)); err == nil {
		t.Fatal("expected additional property validation error")
	}
}

func TestToItemIncludesLocalizationMetadataAndStructuredSchema(t *testing.T) {
	item := toItem(ValueSnapshot{
		Definition: configregistry.Definition{
			Key:            "httpx.access-log-retention-cleanup",
			Module:         "core.httpx",
			Group:          "log.retention",
			GroupKey:       "systemConfig.groups.coreHttpxLogRetention",
			GroupLabel:     "core.httpx / log.retention",
			Title:          "Access log retention cleanup",
			TitleKey:       "systemConfig.items.accessLogRetentionCleanup.title",
			Description:    "Default cleanup configuration for access-log retention jobs.",
			DescriptionKey: "systemConfig.items.accessLogRetentionCleanup.description",
			Tags:           []string{"httpx", "log.retention"},
			Type:           configregistry.ValueTypeObject,
			Schema: json.RawMessage(
				`{"type":"object","properties":{"retentionDays":{"type":"integer","title":"Log retention days","x-i18n":{"titleKey":"systemConfig.fields.retentionDays.title","unitKey":"systemConfig.units.days"}}}}`,
			),
			DefaultValue: json.RawMessage(`{"retentionDays":30}`),
		},
		DefaultValue:   json.RawMessage(`{"retentionDays":30}`),
		EffectiveValue: json.RawMessage(`{"retentionDays":30}`),
	})

	if item.GroupKey == nil || *item.GroupKey != "systemConfig.groups.coreHttpxLogRetention" {
		t.Fatalf("expected group key in response, got %#v", item.GroupKey)
	}
	if item.TitleKey == nil || *item.TitleKey != "systemConfig.items.accessLogRetentionCleanup.title" {
		t.Fatalf("expected title key in response, got %#v", item.TitleKey)
	}
	if item.Tags == nil || len(*item.Tags) != 2 || (*item.Tags)[0] != "httpx" {
		t.Fatalf("expected tags in response, got %#v", item.Tags)
	}
	properties, ok := item.ConfigSchema["properties"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected structured config schema properties, got %#v", item.ConfigSchema)
	}
	retentionDays, ok := properties["retentionDays"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected retentionDays schema, got %#v", properties)
	}
	i18nExtension, ok := retentionDays["x-i18n"].(map[string]interface{})
	if !ok || i18nExtension["unitKey"] != "systemConfig.units.days" {
		t.Fatalf("expected x-i18n unit metadata, got %#v", retentionDays)
	}
}

func newTestService(t *testing.T, definition configregistry.Definition) *Service {
	t.Helper()
	return newTestServiceWithRepo(t, newMemoryRepo(), definition)
}

func newTestServiceWithRepo(t *testing.T, repo *memoryRepo, definition configregistry.Definition) *Service {
	t.Helper()
	registry := configregistry.NewRegistry()
	if err := registry.Register(definition); err != nil {
		t.Fatalf("register definition: %v", err)
	}
	service, err := NewService(registry, repo)
	if err != nil {
		t.Fatalf("create service: %v", err)
	}
	return service
}

type memoryRepo struct {
	values map[string]json.RawMessage
}

func newMemoryRepo() *memoryRepo {
	return &memoryRepo{values: make(map[string]json.RawMessage)}
}

func (r *memoryRepo) GetOverride(_ context.Context, key string) (systemconfigstore.Override, error) {
	value, ok := r.values[key]
	if !ok {
		return systemconfigstore.Override{}, systemconfigstore.ErrOverrideNotFound
	}
	return systemconfigstore.Override{Key: key, Value: cloneRawMessage(value)}, nil
}

func (r *memoryRepo) SetOverride(_ context.Context, key string, value json.RawMessage) (systemconfigstore.Override, error) {
	if len(value) == 0 {
		return systemconfigstore.Override{}, errors.New("value is required")
	}
	r.values[key] = cloneRawMessage(value)
	return systemconfigstore.Override{Key: key, Value: cloneRawMessage(value)}, nil
}

func (r *memoryRepo) DeleteOverride(_ context.Context, key string) error {
	delete(r.values, key)
	return nil
}
