package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"nexus/internal/service"
)

// errorResponse 输出统一格式的错误响应体。
func errorResponse(c *gin.Context, status int, message string) {
	c.JSON(status, gin.H{"error": message})
}

// handleServiceError 将 service 层的哨兵错误映射为 HTTP 响应。
func handleServiceError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrAgentNotFound):
		errorResponse(c, http.StatusNotFound, err.Error())
	case errors.Is(err, service.ErrConversationNotFound):
		errorResponse(c, http.StatusNotFound, err.Error())
	case errors.Is(err, service.ErrProviderNotImplemented):
		errorResponse(c, http.StatusNotImplemented, "Provider not implemented")
	case errors.Is(err, service.ErrLLMNotConfigured):
		errorResponse(c, http.StatusBadRequest, "该 Agent 未配置 api_key，请在编辑中填写")
	case errors.Is(err, service.ErrUnauthorized):
		errorResponse(c, http.StatusUnauthorized, "未登录或登录已失效")
	case errors.Is(err, service.ErrUserExists):
		errorResponse(c, http.StatusConflict, "用户名已存在")
	case errors.Is(err, service.ErrInvalidCredentials):
		errorResponse(c, http.StatusUnauthorized, "用户名或密码错误")
	case errors.Is(err, service.ErrInvalidInput):
		errorResponse(c, http.StatusBadRequest, "用户名不能为空，密码至少 6 位")
	default:
		errorResponse(c, http.StatusInternalServerError, err.Error())
	}
}
