package container

import (
	"context"
	"sync"

	"graft/server/modules/container/terminal"
)

// runtimeLease 延迟关闭共享运行时，并在退休后拒绝新操作，避免调用方使用已关闭的客户端。
type runtimeLease struct {
	runtime Runtime
	mu      sync.Mutex
	refs    int
	retired bool
	closed  bool
}

func newRuntimeLease(runtime Runtime) *runtimeLease {
	return &runtimeLease{runtime: runtime}
}

func (l *runtimeLease) acquire() (func(), error) {
	l.mu.Lock()
	if l.retired || l.closed {
		l.mu.Unlock()
		return nil, errRuntimeDisabled
	}
	l.refs++
	l.mu.Unlock()
	return l.release, nil
}

func (l *runtimeLease) release() {
	l.mu.Lock()
	l.refs--
	closeNow := l.retired && l.refs == 0 && !l.closed
	if closeNow {
		l.closed = true
	}
	l.mu.Unlock()
	if closeNow {
		_ = l.runtime.Close()
	}
}

func (l *runtimeLease) retire() error {
	l.mu.Lock()
	l.retired = true
	closeNow := l.refs == 0 && !l.closed
	if closeNow {
		l.closed = true
	}
	l.mu.Unlock()
	if closeNow {
		return l.runtime.Close()
	}
	return nil
}

func (l *runtimeLease) Info(ctx context.Context) (RuntimeInfo, error) {
	done, err := l.acquire()
	if err != nil {
		return RuntimeInfo{}, err
	}
	defer done()
	return l.runtime.Info(ctx)
}
func (l *runtimeLease) List(ctx context.Context, q ListQuery) ([]Summary, error) {
	done, err := l.acquire()
	if err != nil {
		return nil, err
	}
	defer done()
	return l.runtime.List(ctx, q)
}
func (l *runtimeLease) Detail(ctx context.Context, ref Ref) (Detail, error) {
	done, err := l.acquire()
	if err != nil {
		return Detail{}, err
	}
	defer done()
	return l.runtime.Detail(ctx, ref)
}
func (l *runtimeLease) Mounts(ctx context.Context, ref Ref) ([]Mount, error) {
	done, err := l.acquire()
	if err != nil {
		return nil, err
	}
	defer done()
	return l.runtime.Mounts(ctx, ref)
}
func (l *runtimeLease) MountUsage(ctx context.Context, ref Ref, id string) (MountUsage, error) {
	done, err := l.acquire()
	if err != nil {
		return MountUsage{}, err
	}
	defer done()
	return l.runtime.MountUsage(ctx, ref, id)
}
func (l *runtimeLease) Logs(ctx context.Context, ref Ref, q LogQuery) (Logs, error) {
	done, err := l.acquire()
	if err != nil {
		return Logs{}, err
	}
	defer done()
	return l.runtime.Logs(ctx, ref, q)
}
func (l *runtimeLease) StreamLogs(ctx context.Context, ref Ref, q LogQuery, emit func(LogChunk) error) error {
	done, err := l.acquire()
	if err != nil {
		return err
	}
	defer done()
	return l.runtime.StreamLogs(ctx, ref, q, emit)
}
func (l *runtimeLease) Shell(ctx context.Context, ref Ref, command string) (terminal.Session, error) {
	done, err := l.acquire()
	if err != nil {
		return nil, err
	}
	session, err := l.runtime.Shell(ctx, ref, command)
	if err != nil {
		done()
		return nil, err
	}
	return &leasedSession{Session: session, release: done}, nil
}
func (l *runtimeLease) Start(ctx context.Context, ref Ref) (ActionResult, error) {
	done, err := l.acquire()
	if err != nil {
		return ActionResult{}, err
	}
	defer done()
	return l.runtime.Start(ctx, ref)
}
func (l *runtimeLease) Stop(ctx context.Context, ref Ref) (ActionResult, error) {
	done, err := l.acquire()
	if err != nil {
		return ActionResult{}, err
	}
	defer done()
	return l.runtime.Stop(ctx, ref)
}
func (l *runtimeLease) Restart(ctx context.Context, ref Ref) (ActionResult, error) {
	done, err := l.acquire()
	if err != nil {
		return ActionResult{}, err
	}
	defer done()
	return l.runtime.Restart(ctx, ref)
}
func (l *runtimeLease) Remove(ctx context.Context, ref Ref, opts RemoveOptions) (ActionResult, error) {
	done, err := l.acquire()
	if err != nil {
		return ActionResult{}, err
	}
	defer done()
	return l.runtime.Remove(ctx, ref, opts)
}
func (l *runtimeLease) Close() error { return l.retire() }

func (l *runtimeLease) CollectStatsSnapshots(ctx context.Context) ([]StatsSnapshot, error) {
	runtime, ok := l.runtime.(StatsCollectorRuntime)
	if !ok {
		return nil, nil
	}
	done, err := l.acquire()
	if err != nil {
		return nil, err
	}
	defer done()
	return runtime.CollectStatsSnapshots(ctx)
}

func (l *runtimeLease) StreamRuntimeEvents(ctx context.Context, emit func(RuntimeEventCandidate) error) error {
	runtime, ok := l.runtime.(RuntimeEventSource)
	if !ok {
		return nil
	}
	done, err := l.acquire()
	if err != nil {
		return err
	}
	defer done()
	return runtime.StreamRuntimeEvents(ctx, emit)
}

func (l *runtimeLease) ListDockerImages(ctx context.Context) (DockerImageListResult, error) {
	r, ok := l.runtime.(DockerResourceReader)
	if !ok {
		return DockerImageListResult{}, errUnsupportedContainerRuntime
	}
	done, err := l.acquire()
	if err != nil {
		return DockerImageListResult{}, err
	}
	defer done()
	return r.ListDockerImages(ctx)
}
func (l *runtimeLease) ReadDockerImage(ctx context.Context, id string) (DockerImage, error) {
	r, ok := l.runtime.(DockerResourceReader)
	if !ok {
		return DockerImage{}, errUnsupportedContainerRuntime
	}
	done, err := l.acquire()
	if err != nil {
		return DockerImage{}, err
	}
	defer done()
	return r.ReadDockerImage(ctx, id)
}
func (l *runtimeLease) ListDockerNetworks(ctx context.Context) ([]DockerNetwork, error) {
	r, ok := l.runtime.(DockerResourceReader)
	if !ok {
		return nil, errUnsupportedContainerRuntime
	}
	done, err := l.acquire()
	if err != nil {
		return nil, err
	}
	defer done()
	return r.ListDockerNetworks(ctx)
}
func (l *runtimeLease) ReadDockerNetwork(ctx context.Context, id string) (DockerNetwork, error) {
	r, ok := l.runtime.(DockerResourceReader)
	if !ok {
		return DockerNetwork{}, errUnsupportedContainerRuntime
	}
	done, err := l.acquire()
	if err != nil {
		return DockerNetwork{}, err
	}
	defer done()
	return r.ReadDockerNetwork(ctx, id)
}
func (l *runtimeLease) CreateDockerNetwork(ctx context.Context, command DockerNetworkCreateCommand) (DockerNetworkActionResult, error) {
	r, ok := l.runtime.(interface {
		CreateDockerNetwork(context.Context, DockerNetworkCreateCommand) (DockerNetworkActionResult, error)
	})
	if !ok {
		return DockerNetworkActionResult{}, errUnsupportedContainerRuntime
	}
	done, err := l.acquire()
	if err != nil {
		return DockerNetworkActionResult{}, err
	}
	defer done()
	return r.CreateDockerNetwork(ctx, command)
}
func (l *runtimeLease) RemoveDockerNetwork(ctx context.Context, id, confirmation string) (DockerNetworkActionResult, error) {
	r, ok := l.runtime.(interface {
		RemoveDockerNetwork(context.Context, string, string) (DockerNetworkActionResult, error)
	})
	if !ok {
		return DockerNetworkActionResult{}, errUnsupportedContainerRuntime
	}
	done, err := l.acquire()
	if err != nil {
		return DockerNetworkActionResult{}, err
	}
	defer done()
	return r.RemoveDockerNetwork(ctx, id, confirmation)
}
func (l *runtimeLease) ListDockerVolumes(ctx context.Context) ([]DockerVolume, error) {
	r, ok := l.runtime.(DockerResourceReader)
	if !ok {
		return nil, errUnsupportedContainerRuntime
	}
	done, err := l.acquire()
	if err != nil {
		return nil, err
	}
	defer done()
	return r.ListDockerVolumes(ctx)
}
func (l *runtimeLease) ReadDockerVolume(ctx context.Context, id string) (DockerVolume, error) {
	r, ok := l.runtime.(DockerResourceReader)
	if !ok {
		return DockerVolume{}, errUnsupportedContainerRuntime
	}
	done, err := l.acquire()
	if err != nil {
		return DockerVolume{}, err
	}
	defer done()
	return r.ReadDockerVolume(ctx, id)
}
func (l *runtimeLease) RemoveDockerVolume(ctx context.Context, id string, force bool) error {
	r, ok := l.runtime.(interface {
		RemoveDockerVolume(context.Context, string, bool) error
	})
	if !ok {
		return errUnsupportedContainerRuntime
	}
	done, err := l.acquire()
	if err != nil {
		return err
	}
	defer done()
	return r.RemoveDockerVolume(ctx, id, force)
}
func (l *runtimeLease) PullDockerImage(ctx context.Context, ref string, emit func(DockerImagePullEvent) error) error {
	r, ok := l.runtime.(DockerImageWriter)
	if !ok {
		return errUnsupportedContainerRuntime
	}
	done, err := l.acquire()
	if err != nil {
		return err
	}
	defer done()
	return r.PullDockerImage(ctx, ref, emit)
}
func (l *runtimeLease) TagDockerImage(ctx context.Context, source, target string) error {
	r, ok := l.runtime.(DockerImageWriter)
	if !ok {
		return errUnsupportedContainerRuntime
	}
	done, err := l.acquire()
	if err != nil {
		return err
	}
	defer done()
	return r.TagDockerImage(ctx, source, target)
}
func (l *runtimeLease) UntagDockerImage(ctx context.Context, ref string) error {
	r, ok := l.runtime.(DockerImageWriter)
	if !ok {
		return errUnsupportedContainerRuntime
	}
	done, err := l.acquire()
	if err != nil {
		return err
	}
	defer done()
	return r.UntagDockerImage(ctx, ref)
}
func (l *runtimeLease) RemoveDockerImage(ctx context.Context, id string, force bool) error {
	r, ok := l.runtime.(DockerImageWriter)
	if !ok {
		return errUnsupportedContainerRuntime
	}
	done, err := l.acquire()
	if err != nil {
		return err
	}
	defer done()
	return r.RemoveDockerImage(ctx, id, force)
}

type leasedSession struct {
	terminal.Session
	release func()
	once    sync.Once
}

func (s *leasedSession) Close(ctx context.Context) error {
	err := s.Session.Close(ctx)
	s.once.Do(s.release)
	return err
}

var _ Runtime = (*runtimeLease)(nil)
