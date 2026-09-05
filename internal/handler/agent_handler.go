package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"nexus/internal/service"
)

// AgentHandler 提供 Agent 的增删改查接口。
type AgentHandler struct {
	svc *service.AgentService
}

// NewAgentHandler 构造一个 AgentHandler。
func NewAgentHandler(svc *service.AgentService) *AgentHandler {
	return &AgentHandler{svc: svc}
}

type createAgentRequest struct {
	Name          string   `json:"name" binding:"required"`
	Description   string   `json:"description"`
	SystemPrompt  string   `json:"system_prompt"`
	ModelProvider string   `json:"model_provider"`
	ModelName     string   `json:"model_name"`
	BaseURL       string   `json:"base_url"`
	APIKey        string   `json:"api_key"`
	MaxTokens     int      `json:"max_tokens"`
	Temperature   *float64 `json:"temperature"`
}

type updateAgentRequest struct {
	Name          *string  `json:"name"`
	Description   *string  `json:"description"`
	SystemPrompt  *string  `json:"system_prompt"`
	ModelProvider *string  `json:"model_provider"`
	ModelName     *string  `json:"model_name"`
	BaseURL       *string  `json:"base_url"`
	APIKey        *string  `json:"api_key"`
	MaxTokens     *int     `json:"max_tokens"`
	Temperature   *float64 `json:"temperature"`
}

// Create 处理 POST /agents。
func (h *AgentHandler) Create(c *gin.Context) {
	var req createAgentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		errorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	agent, err := h.svc.Create(currentUserID(c), service.CreateAgentInput{
		Name:          req.Name,
		Description:   req.Description,
		SystemPrompt:  req.SystemPrompt,
		ModelProvider: req.ModelProvider,
		ModelName:     req.ModelName,
		BaseURL:       req.BaseURL,
		APIKey:        req.APIKey,
		MaxTokens:     req.MaxTokens,
		Temperature:   req.Temperature,
	})
	if err != nil {
		handleServiceError(c, err)
		return
	}

	c.JSON(http.StatusCreated, agent)
}

// List 处理 GET /agents。
func (h *AgentHandler) List(c *gin.Context) {
	agents, err := h.svc.List(currentUserID(c))
	if err != nil {
		handleServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, agents)
}

// Get 处理 GET /agents/:agent_id。
func (h *AgentHandler) Get(c *gin.Context) {
	agent, err := h.svc.Get(currentUserID(c), c.Param("agent_id"))
	if err != nil {
		handleServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, agent)
}

// Update 处理 PUT /agents/:agent_id。
func (h *AgentHandler) Update(c *gin.Context) {
	var req updateAgentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		errorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	agent, err := h.svc.Update(currentUserID(c), c.Param("agent_id"), service.UpdateAgentInput{
		Name:          req.Name,
		Description:   req.Description,
		SystemPrompt:  req.SystemPrompt,
		ModelProvider: req.ModelProvider,
		ModelName:     req.ModelName,
		BaseURL:       req.BaseURL,
		APIKey:        req.APIKey,
		MaxTokens:     req.MaxTokens,
		Temperature:   req.Temperature,
	})
	if err != nil {
		handleServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, agent)
}

// Delete 处理 DELETE /agents/:agent_id。
func (h *AgentHandler) Delete(c *gin.Context) {
	if err := h.svc.Delete(currentUserID(c), c.Param("agent_id")); err != nil {
		handleServiceError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}
