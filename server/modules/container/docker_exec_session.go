// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

package container

import (
	"context"
	"errors"
	"io"
	"strings"
	"sync"

	dockertypes "github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"

	"graft/server/modules/container/terminal"
)

type dockerExecClient interface {
	ContainerExecCreate(context.Context, string, container.ExecOptions) (container.ExecCreateResponse, error)
	ContainerExecAttach(context.Context, string, container.ExecAttachOptions) (dockertypes.HijackedResponse, error)
	ContainerExecResize(context.Context, string, container.ResizeOptions) error
}

type dockerExecSession struct {
	client      dockerExecClient
	containerID string
	command     string

	mu       sync.Mutex
	started  bool
	execID   string
	stream   *dockertypes.HijackedResponse
	outputCh chan []byte
	errorCh  chan error
	closeCh  chan struct{}
	closeOnce sync.Once
	done      chan struct{}
}

func newDockerExecSession(client dockerExecClient, containerID string, command string) *dockerExecSession {
	return &dockerExecSession{
		client:      client,
		containerID: strings.TrimSpace(containerID),
		command:     strings.TrimSpace(command),
		outputCh:    make(chan []byte, 32),
		errorCh:     make(chan error, 2),
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

	options := container.ExecOptions{
		Cmd:          []string{s.command},
		AttachStdin:  true,
		AttachStdout: true,
		AttachStderr: true,
		Tty:          true,
		ConsoleSize:  consoleSize(size),
	}
	created, err := s.client.ContainerExecCreate(ctx, s.containerID, options)
	if err != nil {
		return mapDockerShellError(err)
	}
	if strings.TrimSpace(created.ID) == "" {
		return errShellSessionFailed
	}
	stream, err := s.client.ContainerExecAttach(ctx, created.ID, container.ExecAttachOptions{
		Tty:         true,
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
	return mapDockerShellError(s.client.ContainerExecResize(ctx, execID, container.ResizeOptions{
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
		s.stream = nil
		s.mu.Unlock()
		close(s.closeCh)
		if stream != nil {
			_ = stream.CloseWrite()
			stream.Close()
		}
		close(s.outputCh)
		close(s.errorCh)
	})
	select {
	case <-ctx.Done():
		closeErr = ctx.Err()
	default:
	}
	return closeErr
}

func (s *dockerExecSession) copyOutput() {
	s.mu.Lock()
	stream := s.stream
	s.mu.Unlock()
	if stream == nil || stream.Reader == nil {
		s.pushError(errShellSessionFailed)
		return
	}
	buffer := make([]byte, 8192)
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
				close(s.done)
				return
			}
			s.pushError(mapDockerShellError(err))
			close(s.done)
			return
		}
	}
}

func (s *dockerExecSession) pushError(err error) {
	select {
	case <-s.closeCh:
		return
	case s.errorCh <- err:
	}
}

func consoleSize(size terminal.Size) *[2]uint {
	if size.Cols == 0 || size.Rows == 0 {
		return nil
	}
	return &[2]uint{size.Rows, size.Cols}
}
