package apperror

import (
	"errors"

	"graft/server/internal/contract/errorcode"
	messagecontract "graft/server/internal/contract/message"
)

// Kind 标识应用错误对应的稳定 HTTP 语义类别。
type Kind string

const (
	// KindInvalidArgument 表示请求参数不满足调用约束。
	KindInvalidArgument Kind = "invalid_argument"
	// KindUnauthenticated 表示请求缺少有效认证主体。
	KindUnauthenticated Kind = "unauthenticated"
	// KindForbidden 表示已认证主体无权执行操作。
	KindForbidden Kind = "forbidden"
	// KindNotFound 表示目标资源不存在或不可见。
	KindNotFound Kind = "not_found"
	// KindConflict 表示请求与当前资源状态冲突。
	KindConflict Kind = "conflict"
	// KindInternal 表示不应向调用方暴露 cause 的内部系统失败。
	KindInternal Kind = "internal"
)

// Descriptor 保存 HTTP 边界可以安全公开的稳定错误元数据。
// PublicData 必须由错误 owner 明确确认不含内部 cause、凭证或运行时细节。
type Descriptor struct {
	Kind       Kind
	Code       errorcode.Code
	MessageKey messagecontract.Key
	PublicData any
}

// Error 把稳定展示元数据与内部 cause 保存在同一条可解包错误链中。
type Error struct {
	descriptor Descriptor
	cause      error
}

// New 创建没有更底层 cause 的 typed application error。
func New(descriptor Descriptor) error {
	return &Error{descriptor: descriptor}
}

// Wrap 将 cause 包装为 typed application error，并保留 errors.Is/As 语义。
func Wrap(cause error, descriptor Descriptor) error {
	if cause == nil {
		return New(descriptor)
	}
	return &Error{descriptor: descriptor, cause: cause}
}

// Error 返回内部控制流使用的错误文本；HTTP 边界不得直接公开该值。
func (e *Error) Error() string {
	if e == nil {
		return ""
	}
	if e.cause != nil {
		return e.cause.Error()
	}
	if e.descriptor.MessageKey != "" {
		return e.descriptor.MessageKey.String()
	}
	return e.descriptor.Code.String()
}

// Unwrap 返回原始 cause，使标准库错误匹配可以穿过 typed boundary。
func (e *Error) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.cause
}

// Descriptor 返回该错误允许 HTTP 边界消费的稳定展示元数据。
func (e *Error) Descriptor() Descriptor {
	if e == nil {
		return Descriptor{}
	}
	return e.descriptor
}

// Describe 从错误链读取最接近调用方的 typed application descriptor。
func Describe(err error) (Descriptor, bool) {
	var appErr *Error
	if !errors.As(err, &appErr) || appErr == nil {
		return Descriptor{}, false
	}
	return appErr.Descriptor(), true
}

type reportedError struct {
	cause error
}

func (e reportedError) Error() string {
	return e.cause.Error()
}

func (e reportedError) Unwrap() error {
	return e.cause
}

// MarkReported 返回不可变 reported wrapper，避免后续边界重复记录同一错误。
func MarkReported(err error) error {
	if err == nil || IsReported(err) {
		return err
	}
	return reportedError{cause: err}
}

// IsReported 判断错误链是否已经由拥有足够语义的边界记录。
func IsReported(err error) bool {
	var reported reportedError
	return errors.As(err, &reported)
}
