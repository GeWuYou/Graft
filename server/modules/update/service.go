package update

import (
	"context"
	"sync"
	"time"

	"graft/server/internal/buildinfo"
)

// Status 是 Update Center 的只读发现快照。
type Status struct {
	CurrentVersion string              `json:"current_version"`
	Channel        string              `json:"channel"`
	Latest         *Release            `json:"latest,omitempty"`
	Profile        InstallationProfile `json:"installation_profile"`
	CheckedAt      *time.Time          `json:"checked_at,omitempty"`
	CheckError     string              `json:"check_error,omitempty"`
}

// Service 持有无持久化的最近检查快照；失去进程时会安全重新发现上游发布目录。
type Service struct {
	provider   ReleaseProvider
	profile    func() InstallationProfile
	current    func() buildinfo.Info
	mu         sync.RWMutex
	latest     *Release
	checkedAt  *time.Time
	checkError string
}

// NewService 创建只读更新发现服务。
func NewService(provider ReleaseProvider) *Service {
	return &Service{provider: provider, profile: runtimeInstallationProfile, current: buildinfo.Current}
}

// Status 返回当前进程已知的发布发现状态。
func (s *Service) Status() Status {
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
	return status
}

// Check 刷新上游 Release 目录；无有效当前 SemVer 时保留错误而不伪造可升级版本。
func (s *Service) Check(ctx context.Context) Status {
	info := s.current()
	current, err := ParseVersion(info.Version)
	now := time.Now().UTC()
	if err != nil {
		s.store(nil, now, "current build is not a release semantic version")
		return s.Status()
	}
	releases, err := s.provider.List(ctx)
	if err != nil {
		s.store(nil, now, err.Error())
		return s.Status()
	}
	latest, found := SelectLatest(current, releases)
	if found {
		s.store(&latest, now, "")
	} else {
		s.store(nil, now, "")
	}
	return s.Status()
}

func (s *Service) store(latest *Release, checkedAt time.Time, checkError string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.latest = latest
	s.checkedAt = &checkedAt
	s.checkError = checkError
}
