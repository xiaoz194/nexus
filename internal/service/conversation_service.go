package service

import (
	"errors"
	"strings"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"nexus/internal/model"
)

// ConversationService 处理会话的业务逻辑，始终以 Agent
// 为范围来保证隔离。
type ConversationService struct {
	db *gorm.DB
}

// NewConversationService 构造一个 ConversationService。
func NewConversationService(db *gorm.DB) *ConversationService {
	return &ConversationService{db: db}
}

// Create 在指定 Agent 下新增一个会话（Agent 须归属于 userID）。
func (s *ConversationService) Create(userID, agentID, title string) (*model.Conversation, error) {
	if err := s.ensureAgentOwned(userID, agentID); err != nil {
		return nil, err
	}

	conv := &model.Conversation{
		ID:      uuid.NewString(),
		AgentID: agentID,
		Title:   firstNonEmpty(title, "新会话"),
	}
	if err := s.db.Create(conv).Error; err != nil {
		return nil, err
	}
	return conv, nil
}

// List 返回指定 Agent 下的全部会话（Agent 须归属于 userID）。
func (s *ConversationService) List(userID, agentID string) ([]model.Conversation, error) {
	if err := s.ensureAgentOwned(userID, agentID); err != nil {
		return nil, err
	}

	var convs []model.Conversation
	if err := s.db.Where("agent_id = ?", agentID).
		Order("created_at ASC").
		Find(&convs).Error; err != nil {
		return nil, err
	}
	return convs, nil
}

// Update 重命名指定会话（Agent 须归属于 userID）。标题去空白后不可为空。
func (s *ConversationService) Update(userID, agentID, conversationID, title string) (*model.Conversation, error) {
	if err := s.ensureAgentOwned(userID, agentID); err != nil {
		return nil, err
	}
	title = strings.TrimSpace(title)
	if title == "" {
		return nil, ErrInvalidInput
	}

	var conv model.Conversation
	if err := s.db.First(&conv, "id = ? AND agent_id = ?", conversationID, agentID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrConversationNotFound
		}
		return nil, err
	}
	if err := s.db.Model(&conv).Update("title", title).Error; err != nil {
		return nil, err
	}
	return &conv, nil
}

// Delete 在单个事务内删除会话（Agent 须归属于 userID）及其全部消息。
func (s *ConversationService) Delete(userID, agentID, conversationID string) error {
	// 先校验 Agent 归属，再用双重条件校验会话归属。
	if err := s.ensureAgentOwned(userID, agentID); err != nil {
		return err
	}
	var conv model.Conversation
	err := s.db.First(&conv, "id = ? AND agent_id = ?", conversationID, agentID).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrConversationNotFound
		}
		return err
	}

	return s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("conversation_id = ?", conversationID).
			Delete(&model.Message{}).Error; err != nil {
			return err
		}
		if err := tx.Where("id = ? AND agent_id = ?", conversationID, agentID).
			Delete(&model.Conversation{}).Error; err != nil {
			return err
		}
		return nil
	})
}

// ensureAgentOwned 校验该 Agent 存在且归属于 userID。
// 不存在或属于他人（越权）都返回 ErrAgentNotFound。
func (s *ConversationService) ensureAgentOwned(userID, agentID string) error {
	var count int64
	if err := s.db.Model(&model.Agent{}).
		Where("id = ? AND user_id = ?", agentID, userID).
		Count(&count).Error; err != nil {
		return err
	}
	if count == 0 {
		return ErrAgentNotFound
	}
	return nil
}
