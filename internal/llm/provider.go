package llm

import "context"

// ChatMessage 是传给 Provider 的单条消息。
type ChatMessage struct {
	Role    string
	Content string
}

// ChatRequest 是 Provider.Chat 调用的入参。
type ChatRequest struct {
	SystemPrompt string
	Messages     []ChatMessage
	Model        string
}

// StreamFunc 在流式生成时被逐块回调，delta 为本次新增的文本片段。
// 返回非 nil 错误会中止流式。
type StreamFunc func(delta string) error

// Provider 抽象一个 LLM 后端。
type Provider interface {
	// Chat 一次性返回完整回复。
	Chat(ctx context.Context, req ChatRequest) (string, error)
	// ChatStream 流式生成，每块调用 onDelta，并在结束时返回拼接后的完整回复。
	ChatStream(ctx context.Context, req ChatRequest, onDelta StreamFunc) (string, error)
}
