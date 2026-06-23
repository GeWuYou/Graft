package container

import (
	"context"
	"errors"
	"io"
	"strings"
	"sync"

	mobyclient "github.com/moby/moby/client"

	"graft/server/modules/container/terminal"
)

const (
	dockerExecOutputBuffer = 32
	dockerExecErrorBuffer  = 2
	dockerExecReadBuffer   = 8192
)

type dockerExecClient interface {
	ContainerExecCreate(context.Context, string, mobyclient.ExecCreateOptions) (mobyclient.ExecCreateResult, error)
	ContainerExecAttach(context.Context, string, mobyclient.ExecAttachOptions) (mobyclient.HijackedResponse, error)
	ContainerExecResize(context.Context, string, mobyclient.ExecResizeOptions) error
}

type dockerExecSession struct {
	client      dockerExecClient
	containerID string
	command     string

	mu         sync.Mutex
	started    bool
	execID     string
	stream     *mobyclient.HijackedResponse
	outputCh   chan []byte
	errorCh    chan error
	closeCh    chan struct{}
	closeOnce  sync.Once
	done       chan struct{}
	finishOnce sync.Once
}

// newDockerExecSession creates a new Docker exec session with the given client, container ID, and command.
func newDockerExecSession(client dockerExecClient, containerID string, command string) *dockerExecSession {
	return &dockerExecSession{
		client:      client,
		containerID: strings.TrimSpace(containerID),
		command:     strings.TrimSpace(command),
		outputCh:    make(chan []byte, dockerExecOutputBuffer),
		errorCh:     make(chan error, dockerExecErrorBuffer),
		closeCh:     make(chan struct{}),
		done:        make(chan struct{}),
	}
}

func (s *dockerExecSession) Start(ctx context.Context, size terminal.Size) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.started {
		return nil
	}
	if s.client == nil || s.containerID == "" || s.command == "" {
		return errShellSessionFailed
	}

	options := mobyclient.ExecCreateOptions{
		Cmd:          []string{s.command},
		AttachStdin:  true,
		AttachStdout: true,
		AttachStderr: true,
		TTY:          true,
		ConsoleSize:  consoleSize(size),
	}
	created, err := s.client.ContainerExecCreate(ctx, s.containerID, options)
	if err != nil {
		return mapDockerShellError(err)
	}
	if strings.TrimSpace(created.ID) == "" {
		return errShellSessionFailed
	}
	stream, err := s.client.ContainerExecAttach(ctx, created.ID, mobyclient.ExecAttachOptions{
		TTY:         true,
		ConsoleSize: consoleSize(size),
	})
	if err != nil {
		return mapDockerShellError(err)
	}

	s.execID = created.ID
	s.stream = &stream
	s.started = true
	go s.copyOutput()
	return nil
}

func (s *dockerExecSession) Write(_ context.Context, data []byte) error {
	s.mu.Lock()
	stream := s.stream
	s.mu.Unlock()
	if stream == nil || stream.Conn == nil {
		return errShellSessionFailed
	}
	if len(data) == 0 {
		return nil
	}
	_, err := stream.Conn.Write(data)
	if err != nil {
		return mapDockerShellError(err)
	}
	return nil
}

func (s *dockerExecSession) Resize(ctx context.Context, size terminal.Size) error {
	s.mu.Lock()
	execID := s.execID
	s.mu.Unlock()
	if execID == "" {
		return nil
	}
	if size.Cols == 0 || size.Rows == 0 {
		return nil
	}
	return mapDockerShellError(s.client.ContainerExecResize(ctx, execID, mobyclient.ExecResizeOptions{
		Height: size.Rows,
		Width:  size.Cols,
	}))
}

func (s *dockerExecSession) Output() <-chan []byte {
	return s.outputCh
}

func (s *dockerExecSession) Errors() <-chan error {
	return s.errorCh
}

func (s *dockerExecSession) Close(ctx context.Context) error {
	var closeErr error
	s.closeOnce.Do(func() {
		s.mu.Lock()
		stream := s.stream
		started := s.started
		s.stream = nil
		s.mu.Unlock()
		close(s.closeCh)
		if stream != nil {
			_ = stream.CloseWrite()
			stream.Close()
		}
		if !started {
			s.finish()
			return
		}
		select {
		case <-s.done:
		case <-ctx.Done():
			closeErr = ctx.Err()
		}
	})
	return closeErr
}

func (s *dockerExecSession) copyOutput() {
	defer s.finish()
	s.mu.Lock()
	stream := s.stream
	s.mu.Unlock()
	if stream == nil || stream.Reader == nil {
		s.pushError(errShellSessionFailed)
		return
	}
	buffer := make([]byte, dockerExecReadBuffer)
	for {
		n, err := stream.Reader.Read(buffer)
		if n > 0 {
			chunk := append([]byte(nil), buffer[:n]...)
			select {
			case <-s.closeCh:
				return
			case s.outputCh <- chunk:
			}
		}
		if err != nil {
			if errors.Is(err, io.EOF) {
				s.pushError(nil)
				return
			}
			s.pushError(mapDockerShellError(err))
			return
		}
	}
}

func (s *dockerExecSession) finish() {
	s.finishOnce.Do(func() {
		close(s.done)
		close(s.outputCh)
		close(s.errorCh)
	})
}

func (s *dockerExecSession) pushError(err error) {
	select {
	case <-s.closeCh:
		return
	case s.errorCh <- err:
	}
}

// consoleSize 将终端大小转换为 Docker exec 支持的控制台大小格式。
// 如果行数或列数为零，返回 nil；否则返回指向包含 [行数, 列数] 的数组指针。
func consoleSize(size terminal.Size) mobyclient.ConsoleSize {
	if size.Cols == 0 || size.Rows == 0 {
		return mobyclient.ConsoleSize{}
	}
	return mobyclient.ConsoleSize{Height: size.Rows, Width: size.Cols}
}
