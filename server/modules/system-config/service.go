// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

package systemconfig

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"graft/server/internal/configregistry"
	"graft/server/internal/moduleapi"
	"graft/server/internal/scheduler"
	systemconfigstore "graft/server/modules/system-config/store"
)

var (
	errDefinitionNotFound = errors.New("system config definition not found")
	errInvalidConfigValue = errors.New("invalid system config value")
	errSensitiveConfig    = errors.New("sensitive system config cannot be resolved as default config")
)

// ValueSnapshot is the service read model for one effective config value.
type ValueSnapshot struct {
	Definition     configregistry.Definition
	EffectiveValue json.RawMessage
	DefaultValue   json.RawMessage
	OverrideValue  json.RawMessage
	HasOverride    bool
	Status         ValueStatus
	CreatedAt      *time.Time
	CreatedBy      *uint64
	UpdatedAt      *time.Time
	UpdatedBy      *uint64
	UpdatedByName  string
	Masked         bool
}

// ValueStatus describes whether a config item uses its module default or a user override.
type ValueStatus string

const (
	// ValueStatusDefault means no user override exists and the module default is effective.
	ValueStatusDefault ValueStatus = "default"
	// ValueStatusModified means a stored user override is effective.
	ValueStatusModified ValueStatus = "modified"
)

// Service merges module-registered definitions with user overrides.
type Service struct {
	registry *configregistry.Registry
	store    systemconfigstore.Repository
	users    moduleapi.UserService
}

// NewService creates the system configuration service boundary.
func NewService(registry *configregistry.Registry, store systemconfigstore.Repository, users moduleapi.UserService) (*Service, error) {
	if registry == nil {
		return nil, errors.New("config registry is unavailable")
	}
	if store == nil {
		return nil, errors.New("system config store is unavailable")
	}
	return &Service{registry: registry, store: store, users: users}, nil
}

func (s *Service) setUserService(users moduleapi.UserService) {
	if s == nil || users == nil {
		return
	}
	s.users = users
}

// List returns all registered definitions merged with user overrides.
func (s *Service) List(ctx context.Context) ([]ValueSnapshot, error) {
	definitions := s.registry.Items()
	items := make([]ValueSnapshot, 0, len(definitions))
	for _, definition := range definitions {
		item, err := s.snapshot(ctx, definition)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, nil
}

// Get returns one effective config value by registered definition key.
func (s *Service) Get(ctx context.Context, key string) (ValueSnapshot, error) {
	definition, ok := s.registry.Get(key)
	if !ok {
		return ValueSnapshot{}, errDefinitionNotFound
	}
	return s.snapshot(ctx, definition)
}

// ResolveDefaultConfig exposes effective object values for scheduler job definitions.
func (s *Service) ResolveDefaultConfig(ctx context.Context, key string) (string, error) {
	item, err := s.Get(ctx, key)
	if err != nil {
		return "", err
	}
	if item.Definition.Sensitive {
		return "", fmt.Errorf("%w: %s", errSensitiveConfig, item.Definition.Key)
	}
	return string(item.EffectiveValue), nil
}

// IsBooleanConfigEnabled 返回跨模块布尔配置开关的有效值。
//
// 调用方负责传入已注册的布尔配置 key 与显式 fallback；当配置不存在、类型不是布尔值、读取失败或有效值不是合法
// JSON boolean 时，System Config 按 moduleapi.SystemConfigResolver 约定返回 fallback。
func (s *Service) IsBooleanConfigEnabled(ctx context.Context, key string, fallback bool) bool {
	item, err := s.Get(ctx, key)
	if err != nil || item.Definition.Type != configregistry.ValueTypeBoolean {
		return fallback
	}
	var value bool
	if err := json.Unmarshal(item.EffectiveValue, &value); err != nil {
		return fallback
	}
	return value
}

// Update stores a user override for one registered definition key.
func (s *Service) Update(ctx context.Context, key string, value json.RawMessage, userID *uint64) (ValueSnapshot, error) {
	definition, ok := s.registry.Get(key)
	if !ok {
		return ValueSnapshot{}, errDefinitionNotFound
	}
	if err := validateValueForDefinition(definition, value); err != nil {
		return ValueSnapshot{}, err
	}
	if _, err := s.store.SetOverride(ctx, definition.Key, value, userID); err != nil {
		return ValueSnapshot{}, err
	}
	return s.snapshot(ctx, definition)
}

// Reset deletes the user override for one registered definition key.
func (s *Service) Reset(ctx context.Context, key string) (ValueSnapshot, error) {
	definition, ok := s.registry.Get(key)
	if !ok {
		return ValueSnapshot{}, errDefinitionNotFound
	}
	if err := s.store.DeleteOverride(ctx, definition.Key); err != nil {
		return ValueSnapshot{}, err
	}
	return s.snapshot(ctx, definition)
}

func (s *Service) snapshot(ctx context.Context, definition configregistry.Definition) (ValueSnapshot, error) {
	override, err := s.store.GetOverride(ctx, definition.Key)
	hasOverride := true
	if errors.Is(err, systemconfigstore.ErrOverrideNotFound) {
		hasOverride = false
	} else if err != nil {
		return ValueSnapshot{}, err
	}

	effective := definition.DefaultValue
	var overrideValue json.RawMessage
	if hasOverride {
		effective = override.Value
		overrideValue = override.Value
	}

	item := ValueSnapshot{
		Definition:     definition.Snapshot(),
		EffectiveValue: cloneRawMessage(effective),
		DefaultValue:   cloneRawMessage(definition.DefaultValue),
		OverrideValue:  cloneRawMessage(overrideValue),
		HasOverride:    hasOverride,
		Status:         ValueStatusDefault,
		Masked:         definition.Sensitive,
	}
	if hasOverride {
		createdAt := override.CreatedAt
		updatedAt := override.UpdatedAt
		item.Status = ValueStatusModified
		item.CreatedAt = &createdAt
		item.CreatedBy = cloneUint64Pointer(override.CreatedBy)
		item.UpdatedAt = &updatedAt
		item.UpdatedBy = cloneUint64Pointer(override.UpdatedBy)
		item.UpdatedByName = s.usernameForOverride(ctx, override.UpdatedBy)
	}
	if definition.Sensitive {
		item.EffectiveValue = nil
		item.DefaultValue = nil
		item.OverrideValue = nil
	}
	return item, nil
}

func (s *Service) usernameForOverride(ctx context.Context, userID *uint64) string {
	if s == nil || s.users == nil || userID == nil {
		return ""
	}
	user, err := s.users.GetUserByID(ctx, *userID)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(user.Username)
}

func validateValueForDefinition(definition configregistry.Definition, value json.RawMessage) error {
	if len(value) == 0 {
		return fmt.Errorf("%w: value is required", errInvalidConfigValue)
	}
	var decoded any
	if err := json.Unmarshal(value, &decoded); err != nil {
		return fmt.Errorf("%w: %v", errInvalidConfigValue, err)
	}

	expected := configregistry.InvalidJSONShape(decoded, definition.Type)
	if expected != "" {
		return fmt.Errorf("%w: %s must be %s", errInvalidConfigValue, definition.Key, expected)
	}
	if len(definition.Schema) > 0 {
		if err := validateSchemaValue(definition, value); err != nil {
			return fmt.Errorf("%w: %s %v", errInvalidConfigValue, definition.Key, err)
		}
	}
	return nil
}

func validateSchemaValue(definition configregistry.Definition, value json.RawMessage) error {
	switch definition.Type {
	case configregistry.ValueTypeObject:
		return scheduler.ValidateConfigJSON(string(definition.Schema), string(value))
	case configregistry.ValueTypeString,
		configregistry.ValueTypeNumber,
		configregistry.ValueTypeInteger,
		configregistry.ValueTypeBoolean:
		return scheduler.ValidateScalarConfigJSON(string(definition.Schema), string(value), string(definition.Type))
	default:
		return nil
	}
}

func cleanKey(key string) string {
	return strings.TrimSpace(key)
}

func cloneRawMessage(raw json.RawMessage) json.RawMessage {
	if len(raw) == 0 {
		return nil
	}
	cloned := make(json.RawMessage, len(raw))
	copy(cloned, raw)
	return cloned
}

func cloneUint64Pointer(value *uint64) *uint64 {
	if value == nil {
		return nil
	}
	cloned := *value
	return &cloned
}
