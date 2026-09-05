package config

import (
	"fmt"
	"os"

	"github.com/goccy/go-yaml"
)

// Config 是从 config.yml 加载的运行时配置。
type Config struct {
	Server   ServerConfig   `yaml:"server"`
	Database DatabaseConfig `yaml:"database"`
	LLM      LLMConfig      `yaml:"llm"`
	Auth     AuthConfig     `yaml:"auth"`
}

// AuthConfig 是登录会话相关配置。
type AuthConfig struct {
	SessionTTLHours int  `yaml:"session_ttl_hours"` // 会话有效期（小时），默认 168=7天
	CookieSecure    bool `yaml:"cookie_secure"`     // 生产（HTTPS）置 true，dev 保持 false
}

// ServerConfig 是 HTTP 服务配置。
type ServerConfig struct {
	Port string `yaml:"port"`
}

// DatabaseConfig 是数据库连接配置。
type DatabaseConfig struct {
	Type string `yaml:"type"`
	DSN  string `yaml:"dsn"`
}

// LLMConfig 配置大模型服务（OpenAI 兼容 / Anthropic 原生），
// 作为 Agent 未在页面填写时的回退默认值。
type LLMConfig struct {
	BaseURL        string `yaml:"base_url"`
	APIKey         string `yaml:"api_key"`
	Model          string `yaml:"model"`
	MaxTokens      int    `yaml:"max_tokens"` // 仅 anthropic provider 需要（Anthropic 强制）
	TimeoutSeconds int    `yaml:"timeout_seconds"`
}

// Load 从 config.yml（或 CONFIG_FILE 指定的路径）读取配置并应用默认值。
func Load() (*Config, error) {
	path := os.Getenv("CONFIG_FILE")
	if path == "" {
		path = "config.yml"
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("读取配置文件 %s 失败: %w", path, err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("解析配置文件 %s 失败: %w", path, err)
	}

	cfg.applyDefaults()
	return &cfg, nil
}

// applyDefaults 为缺省字段填入合理默认值。
func (c *Config) applyDefaults() {
	if c.Server.Port == "" {
		c.Server.Port = "8080"
	}
	if c.Database.Type == "" {
		c.Database.Type = "postgres"
	}
	if c.LLM.TimeoutSeconds == 0 {
		c.LLM.TimeoutSeconds = 60
	}
	if c.LLM.MaxTokens == 0 {
		c.LLM.MaxTokens = 4096
	}
	if c.Auth.SessionTTLHours == 0 {
		c.Auth.SessionTTLHours = 168 // 7 天
	}
}
