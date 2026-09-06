package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"nexus/internal/service"
)

// ConversationHandler 提供会话相关接口。
type ConversationHandler struct {
	svc *service.ConversationService
}

// NewConversationHandler 构造一个 ConversationHandler。
func NewConversationHandler(svc *service.ConversationService) *ConversationHandler {
	return &ConversationHandler{svc: svc}
}

type createConversationRequest struct {
	Title string `json:"title"`
}

// Create 处理 POST /agents/:agent_id/conversations。
func (h *ConversationHandler) Create(c *gin.Context) {
	var req createConversationRequest
	// 请求体可选：有则解析，允许空 body。
	if c.Request.ContentLength != 0 {
		if err := c.ShouldBindJSON(&req); err != nil {
			errorResponse(c, http.StatusBadRequest, err.Error())
			return
		}
	}

	conv, err := h.svc.Create(currentUserID(c), c.Param("agent_id"), req.Title)
	if err != nil {
		handleServiceError(c, err)
		return
	}
	c.JSON(http.StatusCreated, conv)
}

// List 处理 GET /agents/:agent_id/conversations。
func (h *ConversationHandler) List(c *gin.Context) {
	convs, err := h.svc.List(currentUserID(c), c.Param("agent_id"))
	if err != nil {
		handleServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, convs)
}

type updateConversationRequest struct {
	Title string `json:"title" binding:"required"`
}

// Update 处理 PUT /agents/:agent_id/conversations/:conversation_id（重命名）。
func (h *ConversationHandler) Update(c *gin.Context) {
	var req updateConversationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		errorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	conv, err := h.svc.Update(currentUserID(c), c.Param("agent_id"), c.Param("conversation_id"), req.Title)
	if err != nil {
		handleServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, conv)
}

// Delete 处理 DELETE /agents/:agent_id/conversations/:conversation_id。
func (h *ConversationHandler) Delete(c *gin.Context) {
	err := h.svc.Delete(currentUserID(c), c.Param("agent_id"), c.Param("conversation_id"))
	if err != nil {
		handleServiceError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}
