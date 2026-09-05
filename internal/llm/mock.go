package llm

import (
	"context"
	"strings"
	"time"
)

// MockProvider 是用于开发和测试的确定性 Provider。
type MockProvider struct{}

// NewMockProvider 返回一个 MockProvider。
func NewMockProvider() *MockProvider {
	return &MockProvider{}
}

// Chat 实现 Provider。若没有任何消息则返回就绪提示；
// 否则回显最后一条 user 消息的内容。
func (p *MockProvider) Chat(_ context.Context, req ChatRequest) (string, error) {
	if len(req.Messages) == 0 {
		return "Agent is ready.", nil
	}

	for i := len(req.Messages) - 1; i >= 0; i-- {
		if req.Messages[i].Role == "user" {
			return "Echo: " + req.Messages[i].Content, nil
		}
	}

	return "Agent is ready.", nil
}

// ChatStream 实现 Provider：把回显结果逐字符推送，模拟真实流式输出。
func (p *MockProvider) ChatStream(ctx context.Context, req ChatRequest, onDelta StreamFunc) (string, error) {
	full, _ := p.Chat(ctx, req)

	var b strings.Builder
	for _, r := range full {
		select {
		case <-ctx.Done():
			return b.String(), ctx.Err()
		default:
		}
		s := string(r)
		b.WriteString(s)
		if err := onDelta(s); err != nil {
			return b.String(), err
		}
		time.Sleep(15 * time.Millisecond)
	}
	return b.String(), nil
}
