// Package message 注册消息相关路由（均需登录，含普通与流式发送）。
package message

import (
	"github.com/gin-gonic/gin"

	"nexus/internal/handler"
)

// Register 注册某会话下消息的发送、流式发送与历史查询路由。
func Register(rg *gin.RouterGroup, h *handler.MessageHandler) {
	rg.POST("/agents/:agent_id/conversations/:conversation_id/messages", h.Send)
	rg.POST("/agents/:agent_id/conversations/:conversation_id/messages/stream", h.SendStream)
	rg.GET("/agents/:agent_id/conversations/:conversation_id/messages", h.History)
}
