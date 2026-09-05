package model

import "time"

// Agent 表示一个 AI Agent 的配置。
type Agent struct {
	ID string `gorm:"primaryKey" json:"id"`
	// UserID 是该 Agent 的归属用户（owner）。所有查询都以它为范围，
	// Agent 下的会话/消息通过链式从属自动隔离。
	UserID       string `gorm:"index;not null" json:"user_id"`
	Name         string `gorm:"not null" json:"name"`
	Description  string `gorm:"type:text" json:"description"`
	SystemPrompt string `gorm:"type:text" json:"system_prompt"`
	// ModelProvider：mock（本地回显）、openai（OpenAI 兼容真实大模型）或 anthropic（Claude 原生）。
	ModelProvider string `gorm:"not null" json:"model_provider"`
	ModelName     string `json:"model_name"`
	// BaseURL / APIKey：provider=openai/anthropic 时使用，页面按 Agent 配置；留空回退到 config.yml 的 llm 默认值。
	BaseURL string `gorm:"type:text" json:"base_url"`
	APIKey  string `gorm:"type:text" json:"api_key"`
	// MaxTokens：provider=anthropic 的必填项（Anthropic 强制）；<=0 时回退到 config 默认值。
	MaxTokens   int       `json:"max_tokens"`
	Temperature float64   `gorm:"default:0.7" json:"temperature"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}
