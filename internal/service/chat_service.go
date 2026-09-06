package service

import (
	"context"
	"errors"
	"log"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"nexus/internal/llm"
	"nexus/internal/model"
)

// LLMDefaults 是 Agent 未在页面填写 base_url/api_key/model 时的回退默认值，
// 取自 config.yml 的 llm 段。
type LLMDefaults struct {
	BaseURL   string
	APIKey    string
	Model     string
	MaxTokens int
	Timeout   time.Duration
}

// ChatService 处理会话内的消息收发与历史读取，
// 并强制保证 Agent 隔离。
type ChatService struct {
	db       *gorm.DB
	mock     llm.Provider
	defaults LLMDefaults
}

// NewChatService 构造一个 ChatService。mock 为内置回显 Provider；
// provider=openai 的 Agent 会按自身 base_url/api_key/model 动态构造真实 Provider。
func NewChatService(db *gorm.DB, mock llm.Provider, defaults LLMDefaults) *ChatService {
	return &ChatService{db: db, mock: mock, defaults: defaults}
}

// providerFor 依据 Agent 的配置返回对应的 LLM Provider。
func (s *ChatService) providerFor(agent *model.Agent) (llm.Provider, error) {
	switch agent.ModelProvider {
	case "mock":
		return s.mock, nil
	case "openai":
		// 页面按 Agent 配置优先，留空则回退到 config.yml 的默认值。
		apiKey := firstNonEmpty(agent.APIKey, s.defaults.APIKey)
		if apiKey == "" {
			return nil, ErrLLMNotConfigured
		}
		return llm.NewOpenAIProvider(llm.OpenAIConfig{
			BaseURL: firstNonEmpty(agent.BaseURL, s.defaults.BaseURL),
			APIKey:  apiKey,
			Model:   firstNonEmpty(agent.ModelName, s.defaults.Model),
			Timeout: s.defaults.Timeout,
		}), nil
	case "anthropic":
		// Claude 原生接口：系统提示词以顶层 system 参数下发。
		apiKey := firstNonEmpty(agent.APIKey, s.defaults.APIKey)
		if apiKey == "" {
			return nil, ErrLLMNotConfigured
		}
		return llm.NewAnthropicProvider(llm.AnthropicConfig{
			BaseURL:   firstNonEmpty(agent.BaseURL, s.defaults.BaseURL),
			APIKey:    apiKey,
			Model:     firstNonEmpty(agent.ModelName, s.defaults.Model),
			MaxTokens: firstPositive(agent.MaxTokens, s.defaults.MaxTokens, 4096),
			Timeout:   s.defaults.Timeout,
		}), nil
	default:
		return nil, ErrProviderNotImplemented
	}
}

// preparedChat 是发送前的公共准备结果：选定的 Provider、Agent 及构造好的
// Provider 请求（历史 + 本次待发送的 user 消息，均未落库）。
// SendMessage 与 SendMessageStream 共用。
type preparedChat struct {
	provider llm.Provider
	agent    model.Agent
	req      llm.ChatRequest
}

// prepare 完成发送前的公共步骤：校验会话归属、加载 Agent、选择 Provider、
// 加载已有历史并把「本次待发送的 user 消息」在内存中追加到请求末尾。
//
// 注意：这里**不落库**任何消息。user 与 assistant 消息只有在 LLM 调用成功后
// 才由 persistExchange 一起入库（见下）。这样任何失败都不留痕，前端可以安全地
// 「重试」同一条消息而不会产生重复的 user 消息。
func (s *ChatService) prepare(userID, agentID, conversationID, content string) (*preparedChat, error) {
	// 1. 加载 Agent 并校验归属（须属于 userID）。这一处即锁住整条链——
	//    会话必属于某 Agent，Agent 必属于该用户。
	var agent model.Agent
	if err := s.db.First(&agent, "id = ? AND user_id = ?", agentID, userID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrAgentNotFound
		}
		return nil, err
	}

	// 2. 以 Agent 为范围查询会话。
	var conv model.Conversation
	if err := s.db.First(&conv, "id = ? AND agent_id = ?", conversationID, agentID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrConversationNotFound
		}
		return nil, err
	}

	// 按 Agent 自身配置选择/构造 LLM 后端。
	provider, err := s.providerFor(&agent)
	if err != nil {
		return nil, err
	}

	// 3. 加载已有历史（尚不含本次 user 消息）。
	var history []model.Message
	if err := s.db.Where("conversation_id = ?", conversationID).
		Order("created_at ASC").
		Find(&history).Error; err != nil {
		return nil, err
	}

	// 4. 构造 Provider 请求：历史 + 本次 user 消息（仅内存，未落库）。
	chatMessages := make([]llm.ChatMessage, 0, len(history)+1)
	for _, m := range history {
		chatMessages = append(chatMessages, llm.ChatMessage{Role: m.Role, Content: m.Content})
	}
	chatMessages = append(chatMessages, llm.ChatMessage{Role: model.RoleUser, Content: content})

	return &preparedChat{
		provider: provider,
		agent:    agent,
		req: llm.ChatRequest{
			SystemPrompt: agent.SystemPrompt,
			Messages:     chatMessages,
			Model:        agent.ModelName,
		},
	}, nil
}

// persistExchange 在一个事务里把本轮 user 与 assistant 消息一起落库，
// 并返回 assistant 消息。仅在 LLM 调用成功后调用。显式设置时间戳以保证
// user 严格早于 assistant（避免同毫秒下 created_at 并列导致排序错乱）。
func (s *ChatService) persistExchange(conversationID, userContent, assistantContent string) (*model.Message, error) {
	now := time.Now()
	userMsg := &model.Message{
		ID:             uuid.NewString(),
		ConversationID: conversationID,
		Role:           model.RoleUser,
		Content:        userContent,
		CreatedAt:      now,
	}
	assistantMsg := &model.Message{
		ID:             uuid.NewString(),
		ConversationID: conversationID,
		Role:           model.RoleAssistant,
		Content:        assistantContent,
		CreatedAt:      now.Add(time.Millisecond),
	}
	if err := s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(userMsg).Error; err != nil {
			return err
		}
		return tx.Create(assistantMsg).Error
	}); err != nil {
		return nil, err
	}
	return assistantMsg, nil
}

// SendMessage 调用 LLM，成功后把 user 与 assistant 消息一起落库并返回
// assistant 回复。会话必须归属于指定的 Agent。失败时不落库任何消息，便于重试。
func (s *ChatService) SendMessage(ctx context.Context, userID, agentID, conversationID, content string) (*model.Message, error) {
	p, err := s.prepare(userID, agentID, conversationID, content)
	if err != nil {
		return nil, err
	}

	started := time.Now()
	reply, err := p.provider.Chat(ctx, p.req)
	if err != nil {
		// 失败不落库，允许前端原样重试。
		log.Printf("LLM 调用失败 provider=%s model=%s 耗时=%s: %v", p.agent.ModelProvider, p.agent.ModelName, time.Since(started).Round(time.Millisecond), err)
		return nil, err
	}

	return s.persistExchange(conversationID, content, reply)
}

// SendMessageStream 保存 user 消息，流式调用 LLM，每块通过 onDelta 回调，
// 流结束后把完整回复落库并返回。会话必须归属于指定的 Agent。
func (s *ChatService) SendMessageStream(ctx context.Context, userID, agentID, conversationID, content string, onDelta func(string) error) (*model.Message, error) {
	p, err := s.prepare(userID, agentID, conversationID, content)
	if err != nil {
		return nil, err
	}

	started := time.Now()
	reply, err := p.provider.ChatStream(ctx, p.req, onDelta)
	if err != nil {
		// 客户端主动中断（前端点「停止」→ 断开连接 → ctx 取消）：把已生成的
		// 部分回复当作一次「提前停止」落库，让它刷新后仍在。ChatStream 在 ctx 取消
		// 时会连同已累积文本一起返回，故此处 reply 可能非空。
		if errors.Is(err, context.Canceled) {
			if reply == "" {
				return nil, err // 还没吐出任何字就停了，不落库
			}
			log.Printf("流式被客户端中断，保存部分回复 provider=%s model=%s 长度=%d", p.agent.ModelProvider, p.agent.ModelName, len(reply))
			return s.persistExchange(conversationID, content, reply)
		}
		// 其余为真正失败：不落库，允许前端原样重试。
		log.Printf("LLM 流式调用失败 provider=%s model=%s 耗时=%s: %v", p.agent.ModelProvider, p.agent.ModelName, time.Since(started).Round(time.Millisecond), err)
		return nil, err
	}

	return s.persistExchange(conversationID, content, reply)
}

// History 返回某会话（以 Agent 为范围）的全部消息，
// 按创建时间升序排列。
func (s *ChatService) History(userID, agentID, conversationID string) ([]model.Message, error) {
	// 先校验 Agent 归属（须属于 userID），再校验会话归属于该 Agent。
	var agent model.Agent
	if err := s.db.First(&agent, "id = ? AND user_id = ?", agentID, userID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrAgentNotFound
		}
		return nil, err
	}
	var conv model.Conversation
	if err := s.db.First(&conv, "id = ? AND agent_id = ?", conversationID, agentID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrConversationNotFound
		}
		return nil, err
	}

	var msgs []model.Message
	if err := s.db.Where("conversation_id = ?", conversationID).
		Order("created_at ASC").
		Find(&msgs).Error; err != nil {
		return nil, err
	}
	return msgs, nil
}
