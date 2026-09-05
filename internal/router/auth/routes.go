// Package auth 注册认证相关路由（注册/登录/登出/当前用户）。
package auth

import (
	"github.com/gin-gonic/gin"

	"nexus/internal/handler"
)

// RegisterPublic 注册免鉴权的认证路由。
func RegisterPublic(rg *gin.RouterGroup, h *handler.AuthHandler) {
	rg.POST("/auth/register", h.Register)
	rg.POST("/auth/login", h.Login)
	rg.POST("/auth/logout", h.Logout)
}

// RegisterAuthed 注册需登录后才可访问的认证路由。
func RegisterAuthed(rg *gin.RouterGroup, h *handler.AuthHandler) {
	rg.GET("/auth/me", h.Me)
}
