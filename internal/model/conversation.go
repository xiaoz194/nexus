package model

import "time"

// Conversation 表示归属于单个 Agent 的对话会话。
type Conversation struct {
	ID        string    `gorm:"primaryKey" json:"id"`
	AgentID   string    `gorm:"not null;index" json:"agent_id"`
	Title     string    `gorm:"default:新会话" json:"title"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
