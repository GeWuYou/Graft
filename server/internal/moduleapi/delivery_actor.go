package moduleapi

import (
	"errors"
	"strings"
)

// ErrDeliveryActorInvalid 表示内部交付调用方未提供完整的已认证主体身份。
var ErrDeliveryActorInvalid = errors.New("delivery actor is invalid")

// DeliveryActor 表示既有认证边界已经验证的部署交付调用方。
//
// 该值对象只传递稳定主体标识及其类型，不负责认证、授权或凭据解析。
// 调用方不得从 HTTP 请求字段、回执内容或 Agent 提交内容推导此值；receipt
// 只能作为投递证据，不能据此激活 Agent 身份世代。
type DeliveryActor struct {
	ID   string
	Type string
}

// ValidateDeliveryActor 校验已经认证的部署交付调用方是否具备最小稳定身份。
func ValidateDeliveryActor(actor DeliveryActor) error {
	if strings.TrimSpace(actor.ID) == "" || strings.TrimSpace(actor.Type) == "" {
		return ErrDeliveryActorInvalid
	}
	return nil
}
