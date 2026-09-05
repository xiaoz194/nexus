package service

import "errors"

// 各 service 通用的哨兵错误，由 handler 层
// 映射为对应的 HTTP 状态码。
var (
	ErrAgentNotFound          = errors.New("agent not found")
	ErrConversationNotFound   = errors.New("conversation not found")
	ErrProviderNotImplemented = errors.New("provider not implemented")
	ErrLLMNotConfigured       = errors.New("llm not configured: 缺少 api_key")

	// 认证相关。
	ErrUnauthorized       = errors.New("unauthorized")            // 未登录 / 会话失效
	ErrUserExists         = errors.New("username already exists") // 注册时用户名重复
	ErrInvalidCredentials = errors.New("invalid credentials")    // 用户名或密码错误
	ErrInvalidInput       = errors.New("invalid input")          // 用户名/密码不满足要求
)
