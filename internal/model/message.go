package model

import "time"

// 消息角色。
const (
	RoleUser      = "user"
	RoleAssistant = "assistant"
)

// Message 表示一条会话内的消息。
type Message struct {
	ID             string    `gorm:"primaryKey" json:"id"`
	ConversationID string    `gorm:"not null;index" json:"conversation_id"`
	Role           string    `gorm:"not null" json:"role"`
	Content        string    `gorm:"type:text;not null" json:"content"`
	CreatedAt      time.Time `json:"created_at"`
}
