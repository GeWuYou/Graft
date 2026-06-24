// Package realtime provides the bounded HTTP and websocket surfaces for unified topic subscriptions.
package realtime

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	messagecontract "graft/server/internal/contract/message"
	openapigen "graft/server/internal/contract/openapi/generated"
	"graft/server/internal/httpx"
	"graft/server/internal/i18n"
)

// HTTPRegistration defines dependencies for issuing realtime subscription tickets over HTTP.
type HTTPRegistration struct {
	I18n     *i18n.Service
	Registry TopicIssuerRegistry
}

// RegisterSubscriptionRoutes 注册用于签发实时主题订阅票据的 HTTP 端点。
// 当 router 或 registration.Registry 为空时返回错误。
// 成功时挂载 `POST /realtime/subscriptions`，并在请求处理过程中按主题解析、路由到对应的订阅签发器，最后返回签发结果。
//
// @param router 用于注册路由的 HTTP 路由器。
// @param registration 订阅路由所需的依赖。
// @returns 注册失败时返回错误；成功时返回 nil。
func RegisterSubscriptionRoutes(router gin.IRouter, registration HTTPRegistration) error {
	if router == nil {
		return errors.New("realtime router is unavailable")
	}
	if registration.Registry == nil {
		return errors.New("realtime subscription registry is unavailable")
	}

	router.POST("/realtime/subscriptions", func(ctx *gin.Context) {
		var request SubscriptionRequestPayload
		if !bindSubscriptionRequest(ctx, registration.I18n, &request) {
			return
		}

		topic := NormalizeTopic(request.Topic)
		if topic == "" {
			httpx.WriteLocalizedError(ctx, registration.I18n, http.StatusBadRequest, messagecontract.CommonInvalidArgument.String(), map[string]any{
				"field": "topic",
			})
			return
		}

		issuer, ok := registration.Registry.Resolve(topic)
		if !ok {
			httpx.WriteLocalizedError(ctx, registration.I18n, http.StatusNotFound, messagecontract.CommonInvalidArgument.String(), map[string]any{
				"topic": topic,
			})
			return
		}

		response, err := issuer.IssueSubscription(ctx.Request.Context(), BuildSubscriptionRequest(ctx.Request.Context(), topic))
		if err != nil {
			writeSubscriptionError(ctx, registration.I18n, err, topic)
			return
		}

		httpx.WriteSuccess(ctx, http.StatusOK, response)
	})

	return nil
}

// SubscriptionRequestPayload is the OpenAPI-generated request shape for subscription issuance.
type SubscriptionRequestPayload = openapigen.RealtimeSubscriptionRequest

// bindSubscriptionRequest 绑定并校验订阅请求的 JSON 请求体。
//
// 当 ctx 或 request 为空，或请求体绑定失败时返回 false；绑定成功时返回 true。
// 绑定失败时会写入一个带有 body 字段的本地化 400 错误响应。
func bindSubscriptionRequest(ctx *gin.Context, localizer *i18n.Service, request *SubscriptionRequestPayload) bool {
	if ctx == nil || request == nil {
		return false
	}
	if err := ctx.ShouldBindJSON(request); err != nil {
		httpx.WriteLocalizedError(ctx, localizer, http.StatusBadRequest, messagecontract.CommonInvalidArgument.String(), map[string]any{
			"field": "body",
		})
		return false
	}
	return true
}

// writeSubscriptionError 将领域错误映射为本地化的 HTTP 错误响应并写回客户端。
// 它会根据错误类型选择相应的状态码与消息键，并在错误体中包含 topic 字段。
func writeSubscriptionError(ctx *gin.Context, localizer *i18n.Service, err error, topic string) {
	status := http.StatusInternalServerError
	messageKey := messagecontract.CommonInternalError.String()

	switch {
	case errors.Is(err, ErrTopicRequired):
		status = http.StatusBadRequest
		messageKey = messagecontract.CommonInvalidArgument.String()
	case errors.Is(err, ErrTopicNotFound):
		status = http.StatusNotFound
		messageKey = messagecontract.CommonInvalidArgument.String()
	case errors.Is(err, ErrTopicForbidden):
		status = http.StatusForbidden
		messageKey = messagecontract.AuthForbidden.String()
	case errors.Is(err, ErrTopicConflict):
		status = http.StatusConflict
		messageKey = messagecontract.CommonInternalError.String()
	}

	httpx.WriteLocalizedError(ctx, localizer, status, messageKey, map[string]any{
		"topic": topic,
	})
}
