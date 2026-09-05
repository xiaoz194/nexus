package handler

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"

	"nexus/internal/service"
)

// MessageHandler 提供消息发送与历史查询接口。
type MessageHandler struct {
	svc *service.ChatService
}

// NewMessageHandler 构造一个 MessageHandler。
func NewMessageHandler(svc *service.ChatService) *MessageHandler {
	return &MessageHandler{svc: svc}
}

type sendMessageRequest struct {
	Content string `json:"content" binding:"required"`
}

// Send 处理 POST /agents/:agent_id/conversations/:conversation_id/messages。
func (h *MessageHandler) Send(c *gin.Context) {
	var req sendMessageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		errorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	msg, err := h.svc.SendMessage(
		c.Request.Context(),
		currentUserID(c),
		c.Param("agent_id"),
		c.Param("conversation_id"),
		req.Content,
	)
	if err != nil {
		handleServiceError(c, err)
		return
	}
	c.JSON(http.StatusCreated, msg)
}

// SendStream 处理 POST .../messages/stream，以 SSE 流式返回 assistant 回复。
// 事件格式：
//
//	data: {"delta":"片段"}      —— 每生成一块文本
//	data: {"done":true,"message":{...}} —— 结束，附完整落库的 assistant 消息
//	data: {"error":"..."}      —— 出错（若尚未开始输出，则改为普通 JSON 错误响应）
func (h *MessageHandler) SendStream(c *gin.Context) {
	var req sendMessageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		errorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		errorResponse(c, http.StatusInternalServerError, "streaming unsupported")
		return
	}

	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no") // 关闭 nginx 缓冲，确保逐块下发

	// writeEvent 写一条 SSE data 事件并立即刷新。
	writeEvent := func(v any) {
		b, _ := json.Marshal(v)
		fmt.Fprintf(c.Writer, "data: %s\n\n", b)
		flusher.Flush()
	}

	started := false // 是否已向客户端写过任何 delta（决定出错时的呈现方式）
	onDelta := func(delta string) error {
		started = true
		writeEvent(gin.H{"delta": delta})
		return nil
	}

	msg, err := h.svc.SendMessageStream(
		c.Request.Context(),
		currentUserID(c),
		c.Param("agent_id"),
		c.Param("conversation_id"),
		req.Content,
		onDelta,
	)
	if err != nil {
		if !started {
			// 尚未写出任何内容，仍可返回带正确状态码的 JSON 错误。
			handleServiceError(c, err)
			return
		}
		// 已在流式输出中途出错，只能作为事件下发。
		writeEvent(gin.H{"error": err.Error()})
		return
	}

	writeEvent(gin.H{"done": true, "message": msg})
}

// History 处理 GET /agents/:agent_id/conversations/:conversation_id/messages。
func (h *MessageHandler) History(c *gin.Context) {
	msgs, err := h.svc.History(currentUserID(c), c.Param("agent_id"), c.Param("conversation_id"))
	if err != nil {
		handleServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, msgs)
}
