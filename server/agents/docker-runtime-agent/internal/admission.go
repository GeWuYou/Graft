package agent

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// claimAdmissionClass 是 Agent 领取工作可观察状态的稳定三态分类。
// 空值表示尚未观察到领取结果；该分类进入遥测 Diagnostic 与本地 healthcheck，
// 后端据此区分无工作、身份拒绝与传输失败。
type claimAdmissionClass string

const (
	claimAdmissionNoWork           claimAdmissionClass = "no_work"
	claimAdmissionIdentityRejected claimAdmissionClass = "identity_rejected"
	claimAdmissionTransportFailed  claimAdmissionClass = "transport_failed"
)

const (
	admissionStateFile           = "admission.json"
	admissionTelemetryMinSpacing = 30 * time.Second
	// admissionTelemetryRefreshInterval 重报同值结论的间隔；必须小于后端回执的最长新鲜窗口，
	// 否则持续的传输失败会在回执过期后被误读为就绪。
	admissionTelemetryRefreshInterval = 2 * time.Minute
	// admissionPersistenceInterval 是持续结论刷新本地状态文件时间戳的有界间隔；
	// 必须小于 admissionHealthcheckWindow，持续拒绝才会一直让本地 readiness 失败。
	admissionPersistenceInterval = 30 * time.Second
	// admissionHealthcheckWindow 决定 identity_rejected 是否仍代表当前状态；
	// 持续被拒的 Agent 按 admissionPersistenceInterval 刷新时间戳。
	admissionHealthcheckWindow = time.Minute
)

// claimAdmission 记录最近一次领取结论的稳定分类、HTTP 状态与观察时间。
type claimAdmission struct {
	Class      claimAdmissionClass `json:"class"`
	HTTPStatus int                 `json:"http_status,omitempty"`
	At         time.Time           `json:"at"`
}

// claimAdmissionError 把一次领取结论绑定到稳定分类，供领取循环记录 admission 状态。
type claimAdmissionError struct {
	class  claimAdmissionClass
	status int
	err    error
}

func (e *claimAdmissionError) Error() string { return e.err.Error() }
func (e *claimAdmissionError) Unwrap() error { return e.err }

// admissionTracker 保存进程内最近领取状态，并在变化时持久化到 Agent 私有状态目录，
// 使 healthcheck 与离线诊断可以读取同一份 admission 事实。
type admissionTracker struct {
	mu sync.Mutex
	// dir 为空表示禁用持久化；current/reported/lastSent 仍按进程内语义工作。
	dir       string
	current   claimAdmission
	reported  claimAdmission
	lastSent  time.Time
	persisted time.Time
	validNow  func() time.Time
	loadError error
}

func newAdmissionTracker(dir string) *admissionTracker {
	tracker := &admissionTracker{dir: dir, validNow: time.Now}
	if admission, err := loadAdmissionState(dir); err == nil {
		tracker.current = admission
		tracker.persisted = admission.At
	} else {
		tracker.loadError = err
	}
	return tracker
}

// observe 记录最近一次领取结论。分类或状态变化立即落盘；持续存在的非空结论按
// 有界间隔刷新落盘时间戳，保证本地 healthcheck 不会把持续拒绝误读为过期状态。
func (t *admissionTracker) observe(class claimAdmissionClass, status int) {
	if t == nil {
		return
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	now := t.validNow().UTC()
	next := claimAdmission{Class: class, HTTPStatus: status, At: now}
	unchanged := t.current.Class == next.Class && t.current.HTTPStatus == next.HTTPStatus
	t.current = next
	if unchanged && (class == "" || now.Sub(t.persisted) < admissionPersistenceInterval) {
		// 空结论没有可刷新的本地事实；未变化的拒绝结论等待有界刷新间隔。
		return
	}
	t.persisted = now
	if err := saveAdmissionState(t.dir, next); err != nil {
		t.loadError = err
	}
}

// clear 在领取成功后移除遗留的拒绝记录，避免过期拒绝误导 readiness。
func (t *admissionTracker) clear() {
	if t == nil {
		return
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	t.current = claimAdmission{}
	t.persisted = time.Time{}
	if err := removeAdmissionState(t.dir); err != nil {
		t.loadError = err
	}
}

// shouldReport 预览当前 admission 是否应当随遥测上报。变化结论使用最短上报间隔防抖；
// 同值结论按更长的刷新间隔重报，让后端在旧回执过期前拿到新鲜证据。
// 拒绝状态恢复为空时同样上报一次，让后端的新鲜回执尽快替换过期拒绝证据。
// 去重状态只在遥测成功后由 commitReport 推进，失败的提交会在下一轮重试。
func (t *admissionTracker) shouldReport() (claimAdmission, bool) {
	t.mu.Lock()
	defer t.mu.Unlock()
	now := t.validNow()
	if t.current.Class == "" {
		return claimAdmission{}, t.reported.Class != ""
	}
	minSpacing := admissionTelemetryMinSpacing
	if t.reported.Class == t.current.Class && t.reported.HTTPStatus == t.current.HTTPStatus {
		minSpacing = admissionTelemetryRefreshInterval
	}
	if !t.lastSent.IsZero() && now.Sub(t.lastSent) < minSpacing {
		return claimAdmission{}, false
	}
	return t.current, true
}

// commitReport 在遥测提交成功后推进去重状态。
func (t *admissionTracker) commitReport(reported claimAdmission) {
	if t == nil {
		return
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	t.reported = reported
	t.lastSent = t.validNow()
}

// currentAdmissionClass 返回当前已观察的领取结论分类；未观察到时为空。
func (t *admissionTracker) currentAdmissionClass() claimAdmissionClass {
	if t == nil {
		return ""
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.current.Class
}

func admissionStatePath(dir string) string { return filepath.Join(dir, admissionStateFile) }

func loadAdmissionState(dir string) (claimAdmission, error) {
	payload, err := os.ReadFile(admissionStatePath(dir))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return claimAdmission{}, nil
		}
		return claimAdmission{}, fmt.Errorf("read admission state: %w", err)
	}
	var admission claimAdmission
	if err := json.Unmarshal(payload, &admission); err != nil {
		return claimAdmission{}, fmt.Errorf("decode admission state: %w", err)
	}
	switch admission.Class {
	case "", claimAdmissionNoWork, claimAdmissionIdentityRejected, claimAdmissionTransportFailed:
	default:
		return claimAdmission{}, errors.New("admission state class is invalid")
	}
	return admission, nil
}

func saveAdmissionState(dir string, admission claimAdmission) error {
	if dir == "" {
		return nil
	}
	if err := os.MkdirAll(dir, stateDirectoryMode); err != nil {
		return fmt.Errorf("create admission state directory: %w", err)
	}
	payload, err := json.Marshal(admission)
	if err != nil {
		return errors.New("encode admission state")
	}
	path := admissionStatePath(dir)
	temporary, err := os.CreateTemp(dir, ".admission-*")
	if err != nil {
		return fmt.Errorf("create admission state: %w", err)
	}
	name := temporary.Name()
	if err = temporary.Chmod(persistedStateFilePerm); err == nil {
		_, err = temporary.Write(payload)
	}
	if closeErr := temporary.Close(); err == nil {
		err = closeErr
	}
	if err == nil {
		err = os.Rename(name, path)
	} else {
		_ = os.Remove(name)
	}
	if err != nil {
		return fmt.Errorf("persist admission state: %w", err)
	}
	return nil
}

func removeAdmissionState(dir string) error {
	if dir == "" {
		return nil
	}
	err := os.Remove(admissionStatePath(dir))
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("remove admission state: %w", err)
	}
	return nil
}

// admissionReadiness 描述本地 healthcheck 观察到的领取状态；
// identity_rejected 在窗口内表示当前仍然被拒，readiness 必须失败。
func admissionReadiness(dir string, now time.Time) (claimAdmission, bool, error) {
	admission, err := loadAdmissionState(dir)
	if err != nil {
		return claimAdmission{}, false, err
	}
	if admission.Class == claimAdmissionIdentityRejected && now.Sub(admission.At) <= admissionHealthcheckWindow {
		return admission, false, nil
	}
	return admission, true, nil
}
