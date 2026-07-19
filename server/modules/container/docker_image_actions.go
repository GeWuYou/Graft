package container

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"strings"
	"syscall"
	"unicode"

	cerrdefs "github.com/containerd/errdefs"
	mobyclient "github.com/moby/moby/client"
)

const (
	maxDockerImageReferenceLength = 512
	maxDockerImageProgressLength  = 1024
)

var (
	errInvalidDockerImageReference   = errors.New("invalid docker image reference")
	errDockerImageInUse              = errors.New("docker image is in use")
	errDockerImagePullFailed         = errors.New("docker image pull failed")
	errDockerImageTagFailed          = errors.New("docker image tag failed")
	errDockerImageRemoveFailed       = errors.New("docker image remove failed")
	errDockerImageMultipleTags       = errors.New("docker image is referenced by multiple tags")
	errDockerImageNotFound           = errors.New("docker image not found")
	errDockerImageRuntimeUnavailable = errors.New("docker image runtime unavailable")
	errDockerImageTimeout            = errors.New("docker image remove timeout")
	errDockerImageCommunication      = errors.New("docker image communication failed")
	errDockerImageTagNotAssociated   = errors.New("docker image tag does not reference image")
)

// DockerImagePullEvent 是发送给 API 消费方的脱敏拉取进度事件。
// Docker 守护进程原始错误文本可能包含仓库细节，因此不会越过模块边界。
type DockerImagePullEvent struct {
	Status   string `json:"status"`
	ID       string `json:"id,omitempty"`
	Progress string `json:"progress,omitempty"`
	Error    bool   `json:"error,omitempty"`
}

// DockerImageActionResult 是非流式镜像写操作的规范化结果。
type DockerImageActionResult struct {
	ID         string
	Action     string
	MessageKey string
}

// DockerImageWriter 表示支持镜像生命周期写操作的 Docker 运行时边界。
type DockerImageWriter interface {
	PullDockerImage(context.Context, string, func(DockerImagePullEvent) error) error
	TagDockerImage(context.Context, string, string) error
	UntagDockerImage(context.Context, string) error
	RemoveDockerImage(context.Context, string, bool) error
}

type dockerImageWriterClient interface {
	ImagePull(context.Context, string, mobyclient.ImagePullOptions) (mobyclient.ImagePullResponse, error)
	ImageTag(context.Context, mobyclient.ImageTagOptions) (mobyclient.ImageTagResult, error)
	ImageRemove(context.Context, string, mobyclient.ImageRemoveOptions) (mobyclient.ImageRemoveResult, error)
}

// PullDockerImage 通过 Docker 守护进程拉取镜像，仓库凭据始终由守护进程持有。
func (r *DockerRuntime) PullDockerImage(ctx context.Context, reference string, emit func(DockerImagePullEvent) error) error {
	if err := validateDockerImageReference(reference); err != nil {
		return err
	}
	if emit == nil {
		return errDockerImagePullFailed
	}
	client, ok := r.client.(dockerImageWriterClient)
	if !ok {
		return errUnsupportedContainerRuntime
	}
	stream, err := client.ImagePull(ctx, reference, mobyclient.ImagePullOptions{})
	if err != nil {
		return mapDockerImagePullError(err)
	}
	defer func() {
		_ = stream.Close()
	}()

	return emitDockerImagePullEvents(stream, emit)
}

func emitDockerImagePullEvents(stream io.Reader, emit func(DockerImagePullEvent) error) error {
	decoder := json.NewDecoder(bufio.NewReader(stream))
	for {
		var raw struct {
			Status   string `json:"status"`
			ID       string `json:"id"`
			Progress string `json:"progress"`
			Error    string `json:"error"`
		}
		if err := decoder.Decode(&raw); err != nil {
			if errors.Is(err, io.EOF) {
				return nil
			}
			return errDockerImagePullFailed
		}
		event := DockerImagePullEvent{Status: safeDockerImageProgress(raw.Status), ID: safeDockerImageProgress(raw.ID), Progress: safeDockerImageProgress(raw.Progress)}
		if strings.TrimSpace(raw.Error) != "" {
			event.Status = "error"
			event.Progress = ""
			event.Error = true
		}
		if err := emit(event); err != nil {
			return err
		}
		if event.Error {
			return errDockerImagePullFailed
		}
	}
}

// TagDockerImage 为已有本地镜像应用经过校验的目标标签。
func (r *DockerRuntime) TagDockerImage(ctx context.Context, source, target string) error {
	if err := validateDockerImageReference(source); err != nil {
		return err
	}
	if err := validateDockerImageReference(target); err != nil {
		return err
	}
	client, ok := r.client.(dockerImageWriterClient)
	if !ok {
		return errUnsupportedContainerRuntime
	}
	if _, err := client.ImageTag(ctx, mobyclient.ImageTagOptions{Source: source, Target: target}); err != nil {
		return mapDockerImageTagError(err)
	}
	return nil
}

// UntagDockerImage 移除一个完整的 Repository:Tag 引用，不会强制删除镜像。
func (r *DockerRuntime) UntagDockerImage(ctx context.Context, reference string) error {
	if err := validateDockerImageReference(reference); err != nil {
		return err
	}
	client, ok := r.client.(dockerImageWriterClient)
	if !ok {
		return errUnsupportedContainerRuntime
	}
	if _, err := client.ImageRemove(ctx, reference, mobyclient.ImageRemoveOptions{Force: false, PruneChildren: false}); err != nil {
		return mapDockerImageRemoveError(err)
	}
	return nil
}

// RemoveDockerImage 删除镜像；除非显式 force，守护进程会拒绝删除被容器引用或多标签引用的镜像。
func (r *DockerRuntime) RemoveDockerImage(ctx context.Context, id string, force bool) error {
	if err := validateDockerImageReference(id); err != nil {
		return err
	}
	client, ok := r.client.(dockerImageWriterClient)
	if !ok {
		return errUnsupportedContainerRuntime
	}
	if _, err := client.ImageRemove(ctx, id, mobyclient.ImageRemoveOptions{Force: force, PruneChildren: false}); err != nil {
		return mapDockerImageRemoveError(err)
	}
	return nil
}

func validateDockerImageReference(value string) error {
	if value == "" || len(value) > maxDockerImageReferenceLength || strings.Contains(value, "..") || strings.IndexFunc(value, func(r rune) bool {
		return unicode.IsSpace(r) || unicode.IsControl(r)
	}) >= 0 {
		return errInvalidDockerImageReference
	}
	return nil
}

func safeDockerImageProgress(value string) string {
	value = strings.TrimSpace(value)
	if len(value) > maxDockerImageProgressLength {
		return value[:maxDockerImageProgressLength]
	}
	return value
}

func mapDockerImagePullError(err error) error {
	if mapped := mapDockerError(err); mapped != errRuntimeDaemonUnavailable {
		return mapped
	}
	return errDockerImagePullFailed
}

func mapDockerImageTagError(err error) error {
	if mapped := mapDockerError(err); mapped != errRuntimeDaemonUnavailable {
		return mapped
	}
	return errDockerImageTagFailed
}

func mapDockerImageRemoveError(err error) error {
	message := strings.ToLower(err.Error())
	switch {
	case strings.Contains(message, "image is referenced in multiple repositories"), strings.Contains(message, "image is referenced in multiple tags"):
		return wrapDockerImageRemoveError(errDockerImageMultipleTags, err)
	case strings.Contains(message, "being used"), strings.Contains(message, "image is in use"):
		return wrapDockerImageRemoveError(errDockerImageInUse, err)
	case cerrdefs.IsNotFound(err), strings.Contains(message, "no such image"), strings.Contains(message, "image not found"):
		return wrapDockerImageRemoveError(errDockerImageNotFound, err)
	case errors.Is(err, context.DeadlineExceeded), errors.Is(err, context.Canceled), dockerImageRemoveNetworkTimeout(err):
		return wrapDockerImageRemoveError(errDockerImageTimeout, err)
	case dockerImageRemoveRuntimeUnavailable(err, message):
		return wrapDockerImageRemoveError(errDockerImageRuntimeUnavailable, err)
	case dockerImageRemoveCommunicationError(err, message):
		return wrapDockerImageRemoveError(errDockerImageCommunication, err)
	default:
		return wrapDockerImageRemoveError(errDockerImageRemoveFailed, err)
	}
}

func wrapDockerImageRemoveError(category error, cause error) error {
	return fmt.Errorf("%w: %v", category, cause)
}

func dockerImageRemoveNetworkTimeout(err error) bool {
	var networkError net.Error
	return errors.As(err, &networkError) && networkError.Timeout()
}

func dockerImageRemoveRuntimeUnavailable(err error, message string) bool {
	var errno syscall.Errno
	return errors.Is(err, os.ErrNotExist) ||
		(errors.As(err, &errno) && (errno == syscall.ENOENT || errno == syscall.ECONNREFUSED)) ||
		strings.Contains(message, "cannot connect") ||
		strings.Contains(message, "connection refused") ||
		strings.Contains(message, "is the docker daemon running")
}

func dockerImageRemoveCommunicationError(err error, message string) bool {
	var networkError net.Error
	var errno syscall.Errno
	return (errors.As(err, &networkError) && !networkError.Timeout()) ||
		(errors.As(err, &errno) && (errno == syscall.ECONNRESET || errno == syscall.EPIPE)) ||
		strings.Contains(message, "connection reset") ||
		strings.Contains(message, "broken pipe")
}

var _ DockerImageWriter = (*DockerRuntime)(nil)
