// Package conversation 注册会话相关路由（均需登录，且按 Agent 归属隔离）。
package conversation

import (
	"github.com/gin-gonic/gin"

	"nexus/internal/handler"
)

// Register 注册某 Agent 下会话的增删查路由。
func Register(rg *gin.RouterGroup, h *handler.ConversationHandler) {
	rg.POST("/agents/:agent_id/conversations", h.Create)
	rg.GET("/agents/:agent_id/conversations", h.List)
	rg.DELETE("/agents/:agent_id/conversations/:conversation_id", h.Delete)
}
