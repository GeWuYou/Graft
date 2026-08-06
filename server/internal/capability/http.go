package capability

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"

	"graft/server/internal/container"
	generated "graft/server/internal/contract/openapi/generated"
	"graft/server/internal/httpx"
	"graft/server/internal/moduleapi"
	"graft/server/internal/redisx"
)

const capabilityObservationTTL = 5 * time.Minute

// RegisterRoutes 注册平台能力快照只读路由；快照由 coordinator 统一观测和投影。
func RegisterRoutes(group *gin.RouterGroup, resolver container.Resolver, coordinator *Coordinator, authorizer gin.HandlerFunc) error {
	if group == nil || resolver == nil || coordinator == nil {
		return fmt.Errorf("capability route dependencies are unavailable")
	}
	group.GET("/platform/capabilities", httpx.RequestIDMiddleware(), authorizer, func(ctx *gin.Context) {
		observations, err := coordinator.Observe(ctx.Request.Context())
		if err != nil {
			httpx.AbortAppError(ctx, nil, nil, err)
			return
		}
		items := make([]generated.PlatformCapability, 0, len(observations))
		for _, entry := range coordinator.RegistryEntries() {
			observation := observations[entry.Descriptor.Key]
			var summary *string
			if observation.Summary != "" {
				value := observation.Summary
				summary = &value
			}
			items = append(items, generated.PlatformCapability{Key: entry.Descriptor.Key, Category: generated.CapabilityCategory(entry.Descriptor.Category), Impact: generated.CapabilityImpact(entry.Descriptor.Impact), Status: generated.CapabilityStatus(observation.Status), Summary: summary, ObservedAt: observation.ObservedAt, ExpiresAt: observation.ExpiresAt, Stale: observation.Stale})
		}
		httpx.WriteSuccess(ctx, http.StatusOK, generated.PlatformCapabilitiesResponse{Items: items, ObservedAt: time.Now().UTC()})
	})
	return nil
}

// NewRuntimeCoordinator 构造 core 能力 coordinator，并通过容器延迟解析模块 observation source。
func NewRuntimeCoordinator(db *sql.DB, redisClient *redis.Client, resolver container.Resolver) (*Coordinator, error) {
	entries := []Entry{
		{Descriptor: moduleapi.CapabilityDescriptor{Key: "postgresql", Category: moduleapi.CapabilityCategoryInfrastructure, Impact: moduleapi.CapabilityImpactPlatform, StaleAfter: capabilityObservationTTL}, Provider: databaseProvider{db: db}},
		{Descriptor: moduleapi.CapabilityDescriptor{Key: "redis", Category: moduleapi.CapabilityCategoryInfrastructure, Impact: moduleapi.CapabilityImpactFeature, StaleAfter: capabilityObservationTTL}, Provider: redisProvider{reporter: redisx.NewHealthReporter(redisClient)}},
		{Descriptor: moduleapi.CapabilityDescriptor{Key: "outbound-network", Category: moduleapi.CapabilityCategoryIntegration, Impact: moduleapi.CapabilityImpactFeature, StaleAfter: capabilityObservationTTL}, Provider: resolverProvider{resolver: resolver}},
	}
	registry, err := NewRegistry(entries)
	if err != nil {
		return nil, err
	}
	return NewCoordinator(registry), nil
}

type databaseProvider struct{ db *sql.DB }

func (p databaseProvider) Observe(ctx context.Context) (moduleapi.CapabilityObservation, error) {
	if p.db == nil {
		return moduleapi.CapabilityObservation{Status: moduleapi.CapabilityStatusUnknown, Summary: "Database handle is unavailable"}, nil
	}
	started := time.Now()
	if err := p.db.PingContext(ctx); err != nil {
		return moduleapi.CapabilityObservation{Status: moduleapi.CapabilityStatusUnavailable, Summary: "Database ping failed"}, nil
	}
	return moduleapi.CapabilityObservation{Status: moduleapi.CapabilityStatusHealthy, Summary: fmt.Sprintf("Database ping succeeded in %dms", time.Since(started).Milliseconds())}, nil
}

type redisProvider struct{ reporter redisx.HealthReporter }

func (p redisProvider) Observe(ctx context.Context) (moduleapi.CapabilityObservation, error) {
	if p.reporter == nil {
		return moduleapi.CapabilityObservation{Status: moduleapi.CapabilityStatusDisabled, Summary: "Redis is not configured"}, nil
	}
	report, err := p.reporter.Report(ctx)
	if err != nil || !report.Reachable {
		return moduleapi.CapabilityObservation{Status: moduleapi.CapabilityStatusUnavailable, Summary: "Redis ping failed"}, nil
	}
	return moduleapi.CapabilityObservation{Status: moduleapi.CapabilityStatusHealthy, Summary: "Redis ping succeeded"}, nil
}

type resolverProvider struct{ resolver container.Resolver }

func (p resolverProvider) Observe(ctx context.Context) (moduleapi.CapabilityObservation, error) {
	if p.resolver == nil {
		return moduleapi.CapabilityObservation{Status: moduleapi.CapabilityStatusUnsupported, Summary: "Outbound network source is unavailable"}, nil
	}
	value, err := p.resolver.Resolve((*moduleapi.CapabilityObservationSource)(nil))
	if err != nil {
		return moduleapi.CapabilityObservation{Status: moduleapi.CapabilityStatusUnknown, Summary: "Outbound network observation is not available"}, nil
	}
	source, ok := value.(moduleapi.CapabilityObservationSource)
	if !ok || source == nil {
		return moduleapi.CapabilityObservation{Status: moduleapi.CapabilityStatusUnknown, Summary: "Outbound network observation is not available"}, nil
	}
	return source.ObserveCapability(ctx)
}
