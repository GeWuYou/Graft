package systemconfig

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync/atomic"
	"time"

	"go.uber.org/zap"
	"graft/server/internal/cachex"
	"graft/server/internal/cachex/keys"

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
	systemConfigSnapshotCacheName = "system-config-snapshot"
	systemConfigSnapshotLoadTTL   = 2 * time.Second
)

type snapshotInvalidationAction string

const (
	snapshotInvalidationActionUpdate snapshotInvalidationAction = "update"
	snapshotInvalidationActionReset  snapshotInvalidationAction = "reset"
)

type snapshotInvalidationSource string

const (
	snapshotInvalidationSourceLocal  snapshotInvalidationSource = "local"
	snapshotInvalidationSourceManual snapshotInvalidationSource = "manual"
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
	registry      *configregistry.Registry
	store         systemconfigstore.Repository
	users         moduleapi.UserService
	cache         *cachex.Cache
	snapshotStats snapshotCacheStats

	logger *zap.Logger
}

// ServiceOptions configures module-local cache wiring and optional service collaborators.
type ServiceOptions struct {
	Users  moduleapi.UserService
	Cache  *cachex.Cache
	Logger *zap.Logger
}

// NewService creates a system configuration service. Registry, store, and cache must be non-nil; returns an error if any required dependency is missing.
func NewService(registry *configregistry.Registry, store systemconfigstore.Repository, options ServiceOptions) (*Service, error) {
	if registry == nil {
		return nil, errors.New("config registry is unavailable")
	}
	if store == nil {
		return nil, errors.New("system config store is unavailable")
	}
	if options.Cache == nil {
		return nil, errors.New("system config snapshot cache is unavailable")
	}
	return &Service{
		registry: registry,
		store:    store,
		users:    options.Users,
		cache:    options.Cache,
		logger:   options.Logger,
	}, nil
}

type overrideSnapshotCache struct {
	overrides map[string]systemconfigstore.Override
}

// SnapshotCacheDebugState exposes local snapshot-cache observability without leaking storage access.
type SnapshotCacheDebugState struct {
	Cached                  bool
	CachedOverrideCount     int
	HitCount                uint64
	MissCount               uint64
	LoadCount               uint64
	LoadErrorCount          uint64
	LoadSharedCount         uint64
	LastLoadedOverrideCount int64
	InvalidateCount         uint64
	LastLoadAt              *time.Time
	LastInvalidateAt        *time.Time
	LastInvalidationKey     string
	LastInvalidationAction  string
	LastInvalidationSource  string
	LastInvalidationAt      *time.Time
}

type snapshotInvalidationObservation struct {
	Key       string
	Action    snapshotInvalidationAction
	Source    snapshotInvalidationSource
	UpdatedAt time.Time
}

type snapshotCacheStats struct {
	hitCount                atomic.Uint64
	missCount               atomic.Uint64
	loadCount               atomic.Uint64
	loadErrorCount          atomic.Uint64
	loadSharedCount         atomic.Uint64
	lastLoadedOverrideCount atomic.Int64
	invalidateCount         atomic.Uint64

	lastLoadAtUnixNano       atomic.Int64
	lastInvalidateAtUnixNano atomic.Int64
	lastInvalidation         atomic.Pointer[snapshotInvalidationObservation]
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
	s.invalidateSnapshotCacheForKey(definition.Key, snapshotInvalidationActionUpdate)
	return s.Get(ctx, definition.Key)
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
	s.invalidateSnapshotCacheForKey(definition.Key, snapshotInvalidationActionReset)
	return s.Get(ctx, definition.Key)
}

// SnapshotCacheDebugState returns read-only observability for the unified local snapshot path.
func (s *Service) SnapshotCacheDebugState() SnapshotCacheDebugState {
	if s == nil {
		return SnapshotCacheDebugState{}
	}
	cache, err := s.cachedSnapshot(context.Background())
	state := s.snapshotStats.snapshot()
	state.Cached = err == nil && cache != nil
	if cache != nil {
		state.CachedOverrideCount = len(cache.overrides)
	}
	return state
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
	if cache, ok := s.readWarmOverrideSnapshot(ctx); ok {
		return cache, nil
	}
	s.snapshotStats.recordMiss()
	item, err := s.sharedLoadOverrideSnapshot(ctx)
	if err != nil {
		s.snapshotStats.recordLoadError()
		s.logInvalidationWarning("load system-config snapshot cache", err)
		return nil, err
	}

	return s.decodeOrRebuildOverrideSnapshot(ctx, item)
}

func (s *Service) readWarmOverrideSnapshot(ctx context.Context) (*overrideSnapshotCache, bool) {
	if cache, err := s.cachedSnapshot(ctx); err != nil {
		s.snapshotStats.recordLoadError()
		s.logInvalidationWarning("read system-config snapshot cache", err)
	} else if cache != nil {
		s.snapshotStats.recordHit()
		return cache, true
	}

	return nil, false
}

func (s *Service) sharedLoadOverrideSnapshot(ctx context.Context) (cachex.Item, error) {
	return s.cache.GetOrLoad(ctx, systemConfigSnapshotKey(), func(loadCtx context.Context) (cachex.Item, error) {
		payload, ok, err := s.readSharedCachedSnapshot(loadCtx)
		if ok || err != nil {
			return payload, err
		}

		return s.buildOverrideSnapshotCacheItem(loadCtx)
	})
}

func (s *Service) readSharedCachedSnapshot(ctx context.Context) (cachex.Item, bool, error) {
	cache, err := s.cachedSnapshot(ctx)
	if err != nil {
		s.logInvalidationWarning("recheck system-config snapshot cache", err)
		return cachex.Item{}, false, nil
	}
	if cache == nil {
		return cachex.Item{}, false, nil
	}

	payload, err := marshalOverrideSnapshotCache(cache)
	if err != nil {
		return cachex.Item{}, false, err
	}
	return cachex.NewItem(payload, 0), true, nil
}

func (s *Service) buildOverrideSnapshotCacheItem(ctx context.Context) (cachex.Item, error) {
	overrides, err := s.listOverridesForSnapshotLoad(ctx)
	if err != nil {
		return cachex.Item{}, err
	}
	cache := buildOverrideSnapshotCache(overrides)
	payload, err := marshalOverrideSnapshotCache(cache)
	if err != nil {
		return cachex.Item{}, err
	}
	s.snapshotStats.recordLoad(len(overrides))
	s.logSnapshotDebug("system-config snapshot cache loaded", zap.Int("overrideCount", len(overrides)))
	return cachex.NewItem(payload, 0), nil
}

func (s *Service) decodeOrRebuildOverrideSnapshot(ctx context.Context, item cachex.Item) (*overrideSnapshotCache, error) {
	cache, err := unmarshalOverrideSnapshotCache(item.Value)
	if err != nil {
		s.snapshotStats.recordLoadError()
		s.logInvalidationWarning("decode system-config snapshot cache", err)
		if invalidateErr := s.deleteSnapshotCache(); invalidateErr != nil {
			s.logInvalidationWarning("delete corrupt system-config snapshot cache", invalidateErr)
		}
		return s.reloadOverrideSnapshotAfterCacheEviction(ctx)
	}
	return cache, nil
}

func (s *Service) cachedSnapshot(ctx context.Context) (*overrideSnapshotCache, error) {
	if s == nil || s.cache == nil {
		return nil, nil
	}
	item, ok, err := s.cache.Get(ctx, systemConfigSnapshotKey())
	if err != nil || !ok {
		return nil, err
	}
	cache, decodeErr := unmarshalOverrideSnapshotCache(item.Value)
	return cache, decodeErr
}

func (s *Service) invalidateSnapshotCacheForKey(key string, action snapshotInvalidationAction) {
	s.invalidateSnapshotCacheWithObservation(snapshotInvalidationObservation{
		Key:       key,
		Action:    action,
		Source:    snapshotInvalidationSourceLocal,
		UpdatedAt: time.Now().UTC(),
	})
}

func (s *Service) invalidateSnapshotCacheWithObservation(observation snapshotInvalidationObservation) {
	if s == nil {
		return
	}
	if err := s.deleteSnapshotCache(); err != nil {
		s.logInvalidationWarning("delete system-config snapshot cache", err)
	}
	s.snapshotStats.recordInvalidation(observation)
	s.logSnapshotInvalidation(observation)
}

func (s *Service) reloadOverrideSnapshotAfterCacheEviction(ctx context.Context) (*overrideSnapshotCache, error) {
	overrides, err := s.listOverridesForSnapshotLoad(ctx)
	if err != nil {
		s.snapshotStats.recordLoadError()
		s.logInvalidationWarning("reload system-config snapshot cache after decode failure", err)
		return nil, err
	}
	cache := buildOverrideSnapshotCache(overrides)
	payload, marshalErr := marshalOverrideSnapshotCache(cache)
	if marshalErr != nil {
		return nil, marshalErr
	}
	if err := s.cache.Set(ctx, systemConfigSnapshotKey(), cachex.NewItem(payload, 0)); err != nil {
		s.snapshotStats.recordLoadError()
		s.logInvalidationWarning("persist rebuilt system-config snapshot cache", err)
		return nil, err
	}
	s.snapshotStats.recordLoad(len(overrides))
	s.logSnapshotDebug("system-config snapshot cache rebuilt after decode failure", zap.Int("overrideCount", len(overrides)))
	return cache, nil
}

func (s *Service) listOverridesForSnapshotLoad(ctx context.Context) ([]systemconfigstore.Override, error) {
	detachedCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), systemConfigSnapshotLoadTTL)
	defer cancel()

	return s.store.ListOverrides(detachedCtx)
}

func (s *Service) deleteSnapshotCache() error {
	if s == nil || s.cache == nil {
		return nil
	}
	return s.cache.Delete(context.Background(), systemConfigSnapshotKey())
}

// buildOverrideSnapshotCache 从覆盖值列表构建快照缓存，按 key 索引并深拷贝所有覆盖值。
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

// cloneRawMessage returns a deep copy of the given JSON raw message, or nil if the input is empty.
func cloneRawMessage(raw json.RawMessage) json.RawMessage {
	if len(raw) == 0 {
		return nil
	}
	cloned := make(json.RawMessage, len(raw))
	copy(cloned, raw)
	return cloned
}

// cloneUint64Pointer returns a pointer to a copy of the given uint64, or nil if the given pointer is nil.
func cloneUint64Pointer(value *uint64) *uint64 {
	if value == nil {
		return nil
	}
	cloned := *value
	return &cloned
}

// cloneOverride 创建 Override 的一个深副本，以防止对共享数据的意外修改。
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

// marshalOverrideSnapshotCache 将覆盖快照缓存中的覆盖映射序列化为 JSON 字节。若缓存为 nil，返回错误。
func marshalOverrideSnapshotCache(cache *overrideSnapshotCache) ([]byte, error) {
	if cache == nil {
		return nil, errors.New("system config snapshot cache is unavailable")
	}
	return json.Marshal(cache.overrides)
}

// unmarshalOverrideSnapshotCache 从 JSON 载体重建覆盖快照缓存结构。
// unmarshalOverrideSnapshotCache 将 JSON 载体反序列化为 override 快照缓存。如果载体为空或 JSON 无效，返回错误；否则解析 override 映射并返回深度克隆后的缓存。
func unmarshalOverrideSnapshotCache(payload []byte) (*overrideSnapshotCache, error) {
	if len(payload) == 0 {
		return nil, errors.New("system config snapshot cache returned empty payload")
	}
	var overrides map[string]systemconfigstore.Override
	if err := json.Unmarshal(payload, &overrides); err != nil {
		return nil, err
	}
	cache := &overrideSnapshotCache{
		overrides: make(map[string]systemconfigstore.Override, len(overrides)),
	}
	for key, override := range overrides {
		cache.overrides[key] = cloneOverride(override)
	}
	return cache, nil
}

// SystemConfigSnapshotKey 构建系统配置有效覆盖快照的缓存键。
func systemConfigSnapshotKey() keys.Key {
	return keys.MustNew("system-config", "snapshot", "effective-overrides")
}

func (s *Service) logInvalidationWarning(msg string, err error) {
	if s == nil || s.logger == nil || err == nil {
		return
	}
	s.logger.Warn(msg, zap.Error(err))
}

func (s *Service) logSnapshotInvalidation(observation snapshotInvalidationObservation) {
	if s == nil || s.logger == nil {
		return
	}
	fields := []zap.Field{zap.String("source", string(observation.Source))}
	if observation.Key != "" {
		fields = append(fields, zap.String("key", observation.Key))
	}
	if observation.Action != "" {
		fields = append(fields, zap.String("action", string(observation.Action)))
	}
	if !observation.UpdatedAt.IsZero() {
		fields = append(fields, zap.Time("updatedAt", observation.UpdatedAt))
	}
	s.logger.Debug("system-config snapshot cache invalidated", fields...)
}

func (s *Service) logSnapshotDebug(msg string, fields ...zap.Field) {
	if s == nil || s.logger == nil {
		return
	}
	s.logger.Debug(msg, fields...)
}

func (s *snapshotCacheStats) recordHit() {
	s.hitCount.Add(1)
}

func (s *snapshotCacheStats) recordMiss() {
	s.missCount.Add(1)
}

func (s *snapshotCacheStats) recordLoad(overrideCount int) {
	s.loadCount.Add(1)
	if overrideCount < 0 {
		overrideCount = 0
	}
	s.lastLoadedOverrideCount.Store(int64(overrideCount))
	s.lastLoadAtUnixNano.Store(time.Now().UTC().UnixNano())
}

func (s *snapshotCacheStats) recordLoadError() {
	s.loadErrorCount.Add(1)
}

func (s *snapshotCacheStats) recordInvalidation(observation snapshotInvalidationObservation) {
	s.invalidateCount.Add(1)
	if observation.UpdatedAt.IsZero() {
		observation.UpdatedAt = time.Now().UTC()
	}
	s.lastInvalidateAtUnixNano.Store(time.Now().UTC().UnixNano())
	copied := observation
	s.lastInvalidation.Store(&copied)
}

func (s *snapshotCacheStats) snapshot() SnapshotCacheDebugState {
	state := SnapshotCacheDebugState{
		HitCount:                s.hitCount.Load(),
		MissCount:               s.missCount.Load(),
		LoadCount:               s.loadCount.Load(),
		LoadErrorCount:          s.loadErrorCount.Load(),
		LoadSharedCount:         s.loadSharedCount.Load(),
		LastLoadedOverrideCount: s.lastLoadedOverrideCount.Load(),
		InvalidateCount:         s.invalidateCount.Load(),
	}
	if lastLoadAt := s.lastLoadAtUnixNano.Load(); lastLoadAt > 0 {
		loadedAt := time.Unix(0, lastLoadAt).UTC()
		state.LastLoadAt = &loadedAt
	}
	if lastInvalidateAt := s.lastInvalidateAtUnixNano.Load(); lastInvalidateAt > 0 {
		invalidatedAt := time.Unix(0, lastInvalidateAt).UTC()
		state.LastInvalidateAt = &invalidatedAt
	}
	if lastInvalidation := s.lastInvalidation.Load(); lastInvalidation != nil {
		state.LastInvalidationKey = lastInvalidation.Key
		state.LastInvalidationAction = string(lastInvalidation.Action)
		state.LastInvalidationSource = string(lastInvalidation.Source)
		if !lastInvalidation.UpdatedAt.IsZero() {
			updatedAt := lastInvalidation.UpdatedAt.UTC()
			state.LastInvalidationAt = &updatedAt
		}
	}
	return state
}
