package update

import (
	"context"
	"os"
	"sync"
	"time"

	"graft/server/internal/buildinfo"
	"graft/server/internal/moduleapi"
)

const releaseDiscoveryFailedMessage = "release discovery failed"
const discoveryCacheStaleAfter = 24 * time.Hour

// Status 是 Update Center 的只读发现快照。
type Status struct {
	CurrentVersion    string              `json:"current_version"`
	Channel           string              `json:"channel"`
	UpdatePolicy      UpdatePolicy        `json:"update_policy,omitempty"`
	PolicyInitialized bool                `json:"policy_initialized"`
	AvailableReleases []Release           `json:"available_releases"`
	Latest            *Release            `json:"latest,omitempty"`
	Profile           InstallationProfile `json:"installation_profile"`
	CheckedAt         *time.Time          `json:"checked_at,omitempty"`
	LastSuccessfulAt  *time.Time          `json:"last_successful_at,omitempty"`
	CacheStale        bool                `json:"cache_stale"`
	CheckError        string              `json:"check_error,omitempty"`
}

// withoutComposeCandidates 为只读调用方移除仅供升级管理员确认的宿主机候选路径。
// 它只操作当前响应副本，不能影响本进程后续升级预检使用的 Docker 发现结果。
func (s Status) withoutComposeCandidates() Status {
	s.Profile.ComposeCandidates = []ComposeRootCandidate{}
	return s
}

// Service 持有最近检查快照，并在配置缓存时恢复最近一次成功的已验证结果。
type Service struct {
	provider         ReleaseProvider
	cache            DiscoveryCache
	profile          func() InstallationProfile
	runtimeReader    moduleapi.UpdateComposeRuntimeReader
	current          func() buildinfo.Info
	mu               sync.RWMutex
	latest           *Release
	catalog          []Release
	checkedAt        *time.Time
	lastSuccessfulAt *time.Time
	checkError       string
	loadOnce         sync.Once
}

// NewService 创建只读更新发现服务。
func NewService(provider ReleaseProvider) *Service {
	return NewServiceWithCache(provider, nil)
}

// NewServiceWithCache 创建带有 Update 自有持久 catalog 快照的发现服务。
func NewServiceWithCache(provider ReleaseProvider, cache DiscoveryCache) *Service {
	service := &Service{provider: provider, cache: cache, current: buildinfo.Current}
	service.profile = func() InstallationProfile {
		return DetectInstallationProfileWithComposeReader(context.Background(), os.Getenv, os.LookupEnv, os.Executable, service.runtimeReader)
	}
	return service
}

// Status 返回当前进程已知的发布发现状态。
//
//nolint:cyclop // 快照新鲜度、策略解析和缓存目录投影都是独立的响应不变量。
func (s *Service) Status() Status {
	s.loadCachedSnapshot()
	info := s.current()
	current, err := ParseVersion(info.Version)
	channel := "unknown"
	if err == nil {
		if current.IsPrerelease() {
			channel = "beta"
		} else {
			channel = "stable"
		}
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	policy, initialized := configuredUpdatePolicy()
	catalog := append([]Release(nil), s.catalog...)
	if len(catalog) == 0 && s.latest != nil {
		catalog = []Release{*s.latest}
	}
	status := Status{CurrentVersion: info.Version, Channel: channel, UpdatePolicy: policy, PolicyInitialized: initialized, AvailableReleases: catalog, Profile: s.profile(), CheckError: s.checkError}
	if initialized {
		if selected, found := SelectLatestForPolicy(current, policy, catalog); found {
			status.Latest = &selected
		}
	} else if s.latest != nil {
		copied := *s.latest
		status.Latest = &copied
	}
	if s.checkedAt != nil {
		copied := *s.checkedAt
		status.CheckedAt = &copied
	}
	if s.lastSuccessfulAt != nil {
		copied := *s.lastSuccessfulAt
		status.LastSuccessfulAt = &copied
		status.CacheStale = time.Since(copied) > discoveryCacheStaleAfter
	} else {
		status.CacheStale = true
	}
	if status.CheckError != "" {
		status.CacheStale = true
	}
	return status
}

// Check 刷新上游 Release 目录；无有效当前 SemVer 时保留错误而不伪造可升级版本。
func (s *Service) Check(ctx context.Context) Status {
	s.loadCachedSnapshot()
	info := s.current()
	current, err := ParseVersion(info.Version)
	now := time.Now().UTC()
	if err != nil {
		s.store(ctx, nil, nil, now, false, "current build is not a release semantic version")
		return s.Status()
	}
	releases, err := s.provider.List(ctx)
	if err != nil {
		// 发布源错误可能包含上游 URL 或响应细节，状态 API 只暴露稳定失败语义；保留已验证快照供运维判断。
		s.store(ctx, nil, nil, now, false, releaseDiscoveryFailedMessage)
		return s.Status()
	}
	latest, found := SelectLatest(current, releases)
	if found {
		s.store(ctx, &latest, releases, now, true, "")
	} else {
		s.store(ctx, nil, releases, now, true, "")
	}
	return s.Status()
}

func (s *Service) store(ctx context.Context, latest *Release, catalog []Release, checkedAt time.Time, successful bool, checkError string) {
	s.mu.Lock()
	if successful {
		s.latest = latest
		s.catalog = append([]Release(nil), catalog...)
		s.lastSuccessfulAt = &checkedAt
	}
	s.checkedAt = &checkedAt
	s.checkError = checkError
	snapshot := DiscoverySnapshot{Latest: s.latest, Catalog: s.catalog, LastSuccessfulAt: s.lastSuccessfulAt, LastAttemptAt: s.checkedAt, CheckError: s.checkError}
	s.mu.Unlock()
	if s.cache != nil {
		_ = s.cache.Save(ctx, snapshot)
	}
}

func (s *Service) loadCachedSnapshot() {
	if s == nil || s.cache == nil {
		return
	}
	s.loadOnce.Do(func() {
		snapshot, err := s.cache.Load(context.Background())
		if err != nil {
			s.mu.Lock()
			s.checkError = "load cached release catalog: " + err.Error()
			s.mu.Unlock()
			return
		}
		s.mu.Lock()
		s.latest, s.catalog, s.lastSuccessfulAt, s.checkedAt, s.checkError = snapshot.Latest, snapshot.Catalog, snapshot.LastSuccessfulAt, snapshot.LastAttemptAt, snapshot.CheckError
		s.mu.Unlock()
	})
}
