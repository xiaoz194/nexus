// Package agent 注册 Agent 相关路由（均需登录）。
package agent

import (
	"github.com/gin-gonic/gin"

	"nexus/internal/handler"
)

// Register 注册 Agent 的增删改查路由。
func Register(rg *gin.RouterGroup, h *handler.AgentHandler) {
	rg.POST("/agents", h.Create)
	rg.GET("/agents", h.List)
	rg.GET("/agents/:agent_id", h.Get)
	rg.PUT("/agents/:agent_id", h.Update)
	rg.DELETE("/agents/:agent_id", h.Delete)
}
