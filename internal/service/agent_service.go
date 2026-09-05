package service

import (
	"errors"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"nexus/internal/model"
)

// AgentService 处理 Agent 的业务逻辑。
type AgentService struct {
	db *gorm.DB
}

// NewAgentService 构造一个 AgentService。
func NewAgentService(db *gorm.DB) *AgentService {
	return &AgentService{db: db}
}

// CreateAgentInput 承载创建 Agent 时接受的字段。
type CreateAgentInput struct {
	Name          string
	Description   string
	SystemPrompt  string
	ModelProvider string
	ModelName     string
	BaseURL       string
	APIKey        string
	MaxTokens     int
	Temperature   *float64
}

// UpdateAgentInput 承载更新 Agent 时接受的字段。为 nil 的
// 指针字段保持不变。
type UpdateAgentInput struct {
	Name          *string
	Description   *string
	SystemPrompt  *string
	ModelProvider *string
	ModelName     *string
	BaseURL       *string
	APIKey        *string
	MaxTokens     *int
	Temperature   *float64
}

// Create 插入一个新 Agent（归属于 userID），并为可选字段应用默认值。
func (s *AgentService) Create(userID string, in CreateAgentInput) (*model.Agent, error) {
	agent := &model.Agent{
		ID:            uuid.NewString(),
		UserID:        userID,
		Name:          in.Name,
		Description:   in.Description,
		SystemPrompt:  in.SystemPrompt,
		ModelProvider: firstNonEmpty(in.ModelProvider, "mock"),
		ModelName:     in.ModelName,
		BaseURL:       in.BaseURL,
		APIKey:        in.APIKey,
		MaxTokens:     in.MaxTokens,
		Temperature:   0.7,
	}
	if in.Temperature != nil {
		agent.Temperature = *in.Temperature
	}

	if err := s.db.Create(agent).Error; err != nil {
		return nil, err
	}
	return agent, nil
}

// List 返回该用户的全部 Agent，按创建时间排序。
func (s *AgentService) List(userID string) ([]model.Agent, error) {
	var agents []model.Agent
	if err := s.db.Where("user_id = ?", userID).Order("created_at ASC").Find(&agents).Error; err != nil {
		return nil, err
	}
	return agents, nil
}

// Get 按 ID 返回单个 Agent（须归属于 userID）。查不到即 ErrAgentNotFound，
// 既覆盖「不存在」也覆盖「属于他人」（越权）。
func (s *AgentService) Get(userID, id string) (*model.Agent, error) {
	var agent model.Agent
	if err := s.db.First(&agent, "id = ? AND user_id = ?", id, userID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrAgentNotFound
		}
		return nil, err
	}
	return &agent, nil
}

// Update 修改已有 Agent（须归属于 userID）并返回最新副本。
func (s *AgentService) Update(userID, id string, in UpdateAgentInput) (*model.Agent, error) {
	agent, err := s.Get(userID, id)
	if err != nil {
		return nil, err
	}

	if in.Name != nil {
		agent.Name = *in.Name
	}
	if in.Description != nil {
		agent.Description = *in.Description
	}
	if in.SystemPrompt != nil {
		agent.SystemPrompt = *in.SystemPrompt
	}
	if in.ModelProvider != nil {
		agent.ModelProvider = *in.ModelProvider
	}
	if in.ModelName != nil {
		agent.ModelName = *in.ModelName
	}
	if in.BaseURL != nil {
		agent.BaseURL = *in.BaseURL
	}
	if in.APIKey != nil {
		agent.APIKey = *in.APIKey
	}
	if in.MaxTokens != nil {
		agent.MaxTokens = *in.MaxTokens
	}
	if in.Temperature != nil {
		agent.Temperature = *in.Temperature
	}

	if err := s.db.Save(agent).Error; err != nil {
		return nil, err
	}
	return agent, nil
}

// Delete 在单个事务内删除 Agent（须归属于 userID）及其全部会话和消息。
func (s *AgentService) Delete(userID, id string) error {
	if _, err := s.Get(userID, id); err != nil {
		return err
	}

	return s.db.Transaction(func(tx *gorm.DB) error {
		var convIDs []string
		if err := tx.Model(&model.Conversation{}).
			Where("agent_id = ?", id).
			Pluck("id", &convIDs).Error; err != nil {
			return err
		}

		if len(convIDs) > 0 {
			if err := tx.Where("conversation_id IN ?", convIDs).
				Delete(&model.Message{}).Error; err != nil {
				return err
			}
		}

		if err := tx.Where("agent_id = ?", id).
			Delete(&model.Conversation{}).Error; err != nil {
			return err
		}

		if err := tx.Where("id = ?", id).Delete(&model.Agent{}).Error; err != nil {
			return err
		}

		return nil
	})
}

func firstNonEmpty(v, fallback string) string {
	if v != "" {
		return v
	}
	return fallback
}

// firstPositive 返回第一个大于 0 的值；若都不满足则返回最后一个。
func firstPositive(vals ...int) int {
	for _, v := range vals {
		if v > 0 {
			return v
		}
	}
	if len(vals) == 0 {
		return 0
	}
	return vals[len(vals)-1]
}
