// Package router 组装 HTTP 路由：在此集中挂载 /api/v1 分组与鉴权中间件，
// 各业务的具体路由注册下放到 internal/router 下的子包（auth/agent/...）。
package router

import (
	"github.com/gin-gonic/gin"

	"nexus/internal/handler"
	"nexus/internal/router/agent"
	"nexus/internal/router/auth"
	"nexus/internal/router/conversation"
	"nexus/internal/router/message"
	"nexus/internal/service"
)

// Handlers 汇集各业务 handler，供路由注册使用。
type Handlers struct {
	Auth         *handler.AuthHandler
	Agent        *handler.AgentHandler
	Conversation *handler.ConversationHandler
	Message      *handler.MessageHandler
}

// New 构建并返回配置好路由的 gin 引擎。userSvc 供鉴权中间件校验会话。
func New(h Handlers, userSvc *service.UserService) *gin.Engine {
	r := gin.Default()

	v1 := r.Group("/api/v1")

	// 免鉴权：注册 / 登录 / 登出。
	auth.RegisterPublic(v1, h.Auth)

	// 以下均需登录：会话中间件注入 user_id，各接口按用户隔离。
	authed := v1.Group("")
	authed.Use(handler.AuthRequired(userSvc))

	auth.RegisterAuthed(authed, h.Auth)
	agent.Register(authed, h.Agent)
	conversation.Register(authed, h.Conversation)
	message.Register(authed, h.Message)

	return r
}
