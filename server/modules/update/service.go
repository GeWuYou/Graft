package update

import (
	"context"
	"sync"
	"time"

	"graft/server/internal/buildinfo"
)

const releaseDiscoveryFailedMessage = "release discovery failed"
const discoveryCacheStaleAfter = 24 * time.Hour

// Status 是 Update Center 的只读发现快照。
type Status struct {
	CurrentVersion   string              `json:"current_version"`
	Channel          string              `json:"channel"`
	Latest           *Release            `json:"latest,omitempty"`
	Profile          InstallationProfile `json:"installation_profile"`
	CheckedAt        *time.Time          `json:"checked_at,omitempty"`
	LastSuccessfulAt *time.Time          `json:"last_successful_at,omitempty"`
	CacheStale       bool                `json:"cache_stale"`
	CheckError       string              `json:"check_error,omitempty"`
}

// Service 持有最近检查快照，并在配置缓存时恢复最近一次成功的已验证结果。
type Service struct {
	provider         ReleaseProvider
	cache            DiscoveryCache
	profile          func() InstallationProfile
	current          func() buildinfo.Info
	mu               sync.RWMutex
	latest           *Release
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
	return &Service{provider: provider, cache: cache, profile: runtimeInstallationProfile, current: buildinfo.Current}
}

// Status 返回当前进程已知的发布发现状态。
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
	status := Status{CurrentVersion: info.Version, Channel: channel, Profile: s.profile(), CheckError: s.checkError}
	if s.latest != nil {
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
		s.store(ctx, nil, now, false, "current build is not a release semantic version")
		return s.Status()
	}
	releases, err := s.provider.List(ctx)
	if err != nil {
		// 发布源错误可能包含上游 URL 或响应细节，状态 API 只暴露稳定失败语义；保留已验证快照供运维判断。
		s.store(ctx, nil, now, false, releaseDiscoveryFailedMessage)
		return s.Status()
	}
	latest, found := SelectLatest(current, releases)
	if found {
		s.store(ctx, &latest, now, true, "")
	} else {
		s.store(ctx, nil, now, true, "")
	}
	return s.Status()
}

func (s *Service) store(ctx context.Context, latest *Release, checkedAt time.Time, successful bool, checkError string) {
	s.mu.Lock()
	if successful {
		s.latest = latest
		s.lastSuccessfulAt = &checkedAt
	}
	s.checkedAt = &checkedAt
	s.checkError = checkError
	snapshot := DiscoverySnapshot{Latest: s.latest, LastSuccessfulAt: s.lastSuccessfulAt, LastAttemptAt: s.checkedAt, CheckError: s.checkError}
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
		s.latest, s.lastSuccessfulAt, s.checkedAt, s.checkError = snapshot.Latest, snapshot.LastSuccessfulAt, snapshot.LastAttemptAt, snapshot.CheckError
		s.mu.Unlock()
	})
}
