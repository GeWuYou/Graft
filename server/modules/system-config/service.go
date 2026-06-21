// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

package systemconfig

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"golang.org/x/sync/singleflight"

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

const (
	systemConfigSnapshotInvalidationChannel = "graft:system-config:snapshot:invalidate"
	systemConfigInvalidationPublishTimeout  = 1 * time.Second
	systemConfigInvalidationShutdownTimeout = 1 * time.Second
)

type snapshotInvalidationAction string

const (
	snapshotInvalidationActionUpdate snapshotInvalidationAction = "update"
	snapshotInvalidationActionReset  snapshotInvalidationAction = "reset"
)

type snapshotInvalidationSignal struct {
	Source    string                     `json:"source"`
	Key       string                     `json:"key"`
	Action    snapshotInvalidationAction `json:"action"`
	UpdatedAt time.Time                  `json:"updated_at"`
}

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

	snapshotMu    sync.RWMutex
	snapshotCache *overrideSnapshotCache
	snapshotGroup singleflight.Group

	invalidationMu     sync.Mutex
	invalidationBroker invalidationBroker
	invalidationSub    invalidationSubscription
	invalidationCancel context.CancelFunc
	invalidationDone   chan struct{}
	logger             *zap.Logger
	instanceID         string
}

// NewService creates the system configuration service boundary.
func NewService(registry *configregistry.Registry, store systemconfigstore.Repository, users moduleapi.UserService) (*Service, error) {
	if registry == nil {
		return nil, errors.New("config registry is unavailable")
	}
	if store == nil {
		return nil, errors.New("system config store is unavailable")
	}
	return &Service{
		registry:   registry,
		store:      store,
		users:      users,
		instanceID: uuid.NewString(),
	}, nil
}

type overrideSnapshotCache struct {
	overrides map[string]systemconfigstore.Override
}

type invalidationBroker interface {
	Publish(ctx context.Context, channel string, payload string) error
	Subscribe(ctx context.Context, channel string) (invalidationSubscription, error)
}

type invalidationSubscription interface {
	Channel() <-chan *redis.Message
	Close() error
}

type redisInvalidationBroker struct {
	client *redis.Client
}

type redisPubSubSubscription struct {
	pubsub *redis.PubSub
}

func (s *Service) setUserService(users moduleapi.UserService) {
	if s == nil || users == nil {
		return
	}
	s.users = users
}

// List returns all registered definitions merged with user overrides.
func (s *Service) List(ctx context.Context) ([]ValueSnapshot, error) {
	cache, err := s.loadOverrideSnapshot(ctx)
	if err != nil {
		return nil, err
	}
	definitions := s.registry.Items()
	items := make([]ValueSnapshot, 0, len(definitions))
	for _, definition := range definitions {
		item, err := s.snapshotFromCache(ctx, definition, cache)
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
	cache, err := s.loadOverrideSnapshot(ctx)
	if err != nil {
		return ValueSnapshot{}, err
	}
	return s.snapshotFromCache(ctx, definition, cache)
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
	s.invalidateSnapshotCache()
	item, err := s.Get(ctx, definition.Key)
	s.publishSnapshotInvalidation(ctx, definition.Key, snapshotInvalidationActionUpdate)
	return item, err
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
	s.invalidateSnapshotCache()
	item, err := s.Get(ctx, definition.Key)
	s.publishSnapshotInvalidation(ctx, definition.Key, snapshotInvalidationActionReset)
	return item, err
}

func (s *Service) startInvalidationSync(ctx context.Context, broker invalidationBroker, logger *zap.Logger) {
	if s == nil {
		return
	}
	if ctx == nil {
		ctx = context.Background()
	}

	s.invalidationMu.Lock()
	s.logger = logger
	s.invalidationBroker = broker
	if broker == nil || s.invalidationCancel != nil {
		s.invalidationMu.Unlock()
		return
	}

	runCtx, cancel := context.WithCancel(context.WithoutCancel(ctx))
	sub, err := broker.Subscribe(runCtx, systemConfigSnapshotInvalidationChannel)
	if err != nil {
		s.invalidationMu.Unlock()
		cancel()
		s.logInvalidationWarning("subscribe system-config invalidation", err)
		return
	}

	done := make(chan struct{})
	s.invalidationCancel = cancel
	s.invalidationSub = sub
	s.invalidationDone = done
	s.invalidationMu.Unlock()

	go s.runInvalidationSubscriber(runCtx, sub, done)
}

func (s *Service) stopInvalidationSync() {
	if s == nil {
		return
	}

	s.invalidationMu.Lock()
	cancel := s.invalidationCancel
	sub := s.invalidationSub
	done := s.invalidationDone
	s.invalidationCancel = nil
	s.invalidationSub = nil
	s.invalidationDone = nil
	s.invalidationMu.Unlock()

	if cancel != nil {
		cancel()
	}
	if sub != nil {
		_ = sub.Close()
	}
	if done == nil {
		return
	}

	select {
	case <-done:
	case <-time.After(systemConfigInvalidationShutdownTimeout):
		s.logInvalidationWarning("shutdown system-config invalidation subscriber", errors.New("timed out waiting for subscriber to stop"))
	}
}

func (s *Service) snapshotFromCache(
	ctx context.Context,
	definition configregistry.Definition,
	cache *overrideSnapshotCache,
) (ValueSnapshot, error) {
	override, hasOverride := cache.overrides[definition.Key]
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

func (s *Service) loadOverrideSnapshot(ctx context.Context) (*overrideSnapshotCache, error) {
	if cache := s.cachedSnapshot(); cache != nil {
		return cache, nil
	}

	resultCh := s.snapshotGroup.DoChan("override-snapshot", func() (any, error) {
		if cache := s.cachedSnapshot(); cache != nil {
			return cache, nil
		}
		overrides, err := s.store.ListOverrides(context.WithoutCancel(ctx))
		if err != nil {
			return nil, err
		}
		cache := buildOverrideSnapshotCache(overrides)
		s.snapshotMu.Lock()
		s.snapshotCache = cache
		s.snapshotMu.Unlock()
		return cache, nil
	})

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case result := <-resultCh:
		if result.Err != nil {
			return nil, result.Err
		}
		cache, ok := result.Val.(*overrideSnapshotCache)
		if !ok || cache == nil {
			return nil, errors.New("system config snapshot cache returned unexpected value")
		}
		return cache, nil
	}
}

func (s *Service) cachedSnapshot() *overrideSnapshotCache {
	if s == nil {
		return nil
	}
	s.snapshotMu.RLock()
	defer s.snapshotMu.RUnlock()
	return s.snapshotCache
}

func (s *Service) invalidateSnapshotCache() {
	if s == nil {
		return
	}
	s.snapshotMu.Lock()
	s.snapshotCache = nil
	s.snapshotMu.Unlock()
}

func buildOverrideSnapshotCache(overrides []systemconfigstore.Override) *overrideSnapshotCache {
	cache := &overrideSnapshotCache{
		overrides: make(map[string]systemconfigstore.Override, len(overrides)),
	}
	for _, override := range overrides {
		cache.overrides[override.Key] = cloneOverride(override)
	}
	return cache
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

func cloneOverride(value systemconfigstore.Override) systemconfigstore.Override {
	return systemconfigstore.Override{
		Key:       value.Key,
		Value:     cloneRawMessage(value.Value),
		CreatedAt: value.CreatedAt,
		CreatedBy: cloneUint64Pointer(value.CreatedBy),
		UpdatedAt: value.UpdatedAt,
		UpdatedBy: cloneUint64Pointer(value.UpdatedBy),
	}
}

func newRedisInvalidationBroker(client *redis.Client) invalidationBroker {
	if client == nil {
		return nil
	}
	return redisInvalidationBroker{client: client}
}

func (b redisInvalidationBroker) Publish(ctx context.Context, channel string, payload string) error {
	return b.client.Publish(ctx, channel, payload).Err()
}

func (b redisInvalidationBroker) Subscribe(ctx context.Context, channel string) (invalidationSubscription, error) {
	pubsub := b.client.Subscribe(ctx, channel)
	if _, err := pubsub.Receive(ctx); err != nil {
		_ = pubsub.Close()
		return nil, err
	}
	return redisPubSubSubscription{pubsub: pubsub}, nil
}

func (s redisPubSubSubscription) Channel() <-chan *redis.Message {
	return s.pubsub.Channel()
}

func (s redisPubSubSubscription) Close() error {
	return s.pubsub.Close()
}

func (s *Service) publishSnapshotInvalidation(ctx context.Context, key string, action snapshotInvalidationAction) {
	broker := s.currentInvalidationBroker()
	if broker == nil {
		return
	}

	signal := snapshotInvalidationSignal{
		Source:    s.instanceID,
		Key:       key,
		Action:    action,
		UpdatedAt: time.Now().UTC(),
	}
	payload, err := json.Marshal(signal)
	if err != nil {
		s.logInvalidationWarning("marshal system-config invalidation signal", err)
		return
	}

	if ctx == nil {
		ctx = context.Background()
	}
	publishCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), systemConfigInvalidationPublishTimeout)
	defer cancel()

	if err := broker.Publish(publishCtx, systemConfigSnapshotInvalidationChannel, string(payload)); err != nil {
		s.logInvalidationWarning("publish system-config invalidation", err)
	}
}

func (s *Service) currentInvalidationBroker() invalidationBroker {
	if s == nil {
		return nil
	}
	s.invalidationMu.Lock()
	defer s.invalidationMu.Unlock()
	return s.invalidationBroker
}

func (s *Service) runInvalidationSubscriber(ctx context.Context, sub invalidationSubscription, done chan struct{}) {
	defer close(done)
	for {
		select {
		case <-ctx.Done():
			return
		case message, ok := <-sub.Channel():
			if !ok {
				return
			}
			s.handleInvalidationMessage(message)
		}
	}
}

func (s *Service) handleInvalidationMessage(message *redis.Message) {
	if s == nil || message == nil {
		return
	}

	var signal snapshotInvalidationSignal
	if err := json.Unmarshal([]byte(message.Payload), &signal); err != nil {
		s.logInvalidationWarning("decode system-config invalidation signal", err)
		return
	}
	if signal.Source == s.instanceID {
		return
	}
	s.invalidateSnapshotCache()
}

func (s *Service) logInvalidationWarning(msg string, err error) {
	if s == nil || s.logger == nil || err == nil {
		return
	}
	s.logger.Warn(msg, zap.Error(err))
}
