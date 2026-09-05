package llm

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// OpenAIConfig 配置一个 OpenAI 兼容的 Provider。
type OpenAIConfig struct {
	BaseURL string        // API 根地址，如 https://api.deepseek.com（会自动补 /chat/completions）
	APIKey  string        // Bearer 密钥
	Model   string        // 默认模型；Agent 未指定 model_name 时使用
	Timeout time.Duration // 非流式 HTTP 超时（流式不设硬超时，靠 ctx 取消）
}

// OpenAIProvider 通过 OpenAI 兼容的 /chat/completions 接口调用大模型，
// 可适配 DeepSeek、通义千问、Kimi、智谱、本地 vLLM 等。
type OpenAIProvider struct {
	endpoint string
	apiKey   string
	model    string
	client   *http.Client // 非流式：带超时
	streamer *http.Client // 流式：不设超时，由请求 ctx 控制
}

// NewOpenAIProvider 构造一个 OpenAIProvider。
func NewOpenAIProvider(cfg OpenAIConfig) *OpenAIProvider {
	timeout := cfg.Timeout
	if timeout <= 0 {
		timeout = 60 * time.Second
	}
	base := strings.TrimRight(cfg.BaseURL, "/")
	return &OpenAIProvider{
		endpoint: base + "/chat/completions",
		apiKey:   cfg.APIKey,
		model:    cfg.Model,
		client:   &http.Client{Timeout: timeout},
		streamer: &http.Client{}, // 流式请求可能持续很久，超时交给 ctx
	}
}

type openaiMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type openaiRequest struct {
	Model    string          `json:"model"`
	Messages []openaiMessage `json:"messages"`
	Stream   bool            `json:"stream,omitempty"`
}

type openaiResponse struct {
	Choices []struct {
		Message openaiMessage `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error"`
}

// streamChunk 是流式响应中每个 SSE data 块的结构。
type streamChunk struct {
	Choices []struct {
		Delta struct {
			Content string `json:"content"`
		} `json:"delta"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error"`
}

// resolveModel 选定实际使用的模型名。
func (p *OpenAIProvider) resolveModel(req ChatRequest) (string, error) {
	model := req.Model
	if model == "" {
		model = p.model
	}
	if model == "" {
		return "", fmt.Errorf("未指定模型名（Agent.model_name 与配置 llm.model 均为空）")
	}
	return model, nil
}

// buildMessages 转换历史消息。系统提示词不单独发 system 角色（部分网关/Claude 类
// 模型会拒绝 messages 中的 system），而是并入首条消息内容，兼容性最好。
func buildMessages(req ChatRequest) []openaiMessage {
	msgs := make([]openaiMessage, 0, len(req.Messages)+1)
	for _, m := range req.Messages {
		msgs = append(msgs, openaiMessage{Role: m.Role, Content: m.Content})
	}
	if sp := strings.TrimSpace(req.SystemPrompt); sp != "" {
		if len(msgs) > 0 {
			msgs[0].Content = sp + "\n\n" + msgs[0].Content
		} else {
			msgs = append(msgs, openaiMessage{Role: "user", Content: sp})
		}
	}
	return msgs
}

// newRequest 构造带鉴权头的 POST 请求。
func (p *OpenAIProvider) newRequest(ctx context.Context, body []byte) (*http.Request, error) {
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, p.endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+p.apiKey)
	return httpReq, nil
}

// Chat 实现 Provider：一次性返回完整回复。
func (p *OpenAIProvider) Chat(ctx context.Context, req ChatRequest) (string, error) {
	model, err := p.resolveModel(req)
	if err != nil {
		return "", err
	}

	body, err := json.Marshal(openaiRequest{Model: model, Messages: buildMessages(req)})
	if err != nil {
		return "", err
	}
	httpReq, err := p.newRequest(ctx, body)
	if err != nil {
		return "", err
	}

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("请求 LLM 失败: %w", err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	if err := guardNonJSON(resp, raw, p.endpoint); err != nil {
		return "", err
	}

	var parsed openaiResponse
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return "", fmt.Errorf("解析 LLM 响应失败(status=%d): %s", resp.StatusCode, truncate(strings.TrimSpace(string(raw)), 300))
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		if parsed.Error != nil && parsed.Error.Message != "" {
			return "", fmt.Errorf("LLM 返回错误(status=%d): %s", resp.StatusCode, parsed.Error.Message)
		}
		return "", fmt.Errorf("LLM 返回非 2xx(status=%d): %s", resp.StatusCode, truncate(strings.TrimSpace(string(raw)), 300))
	}

	if len(parsed.Choices) == 0 {
		return "", fmt.Errorf("LLM 响应无 choices")
	}
	return parsed.Choices[0].Message.Content, nil
}

// ChatStream 实现 Provider：以 SSE 流式读取，每块回调 onDelta，返回完整回复。
func (p *OpenAIProvider) ChatStream(ctx context.Context, req ChatRequest, onDelta StreamFunc) (string, error) {
	model, err := p.resolveModel(req)
	if err != nil {
		return "", err
	}

	body, err := json.Marshal(openaiRequest{Model: model, Messages: buildMessages(req), Stream: true})
	if err != nil {
		return "", err
	}
	httpReq, err := p.newRequest(ctx, body)
	if err != nil {
		return "", err
	}
	httpReq.Header.Set("Accept", "text/event-stream")

	resp, err := p.streamer.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("请求 LLM 失败: %w", err)
	}
	defer resp.Body.Close()

	// 非 2xx：读出错误体（此时通常是 JSON 错误，而非 SSE 流）。
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		raw, _ := io.ReadAll(resp.Body)
		if err := guardNonJSON(resp, raw, p.endpoint); err != nil {
			return "", err
		}
		var parsed openaiResponse
		if json.Unmarshal(raw, &parsed) == nil && parsed.Error != nil && parsed.Error.Message != "" {
			return "", fmt.Errorf("LLM 返回错误(status=%d): %s", resp.StatusCode, parsed.Error.Message)
		}
		return "", fmt.Errorf("LLM 返回非 2xx(status=%d): %s", resp.StatusCode, truncate(strings.TrimSpace(string(raw)), 300))
	}

	var full strings.Builder
	reader := bufio.NewReader(resp.Body)
	for {
		select {
		case <-ctx.Done():
			return full.String(), ctx.Err()
		default:
		}

		line, err := reader.ReadString('\n')
		if len(line) > 0 {
			line = strings.TrimSpace(line)
			if line != "" && strings.HasPrefix(line, "data:") {
				data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
				if data == "[DONE]" {
					break
				}
				var chunk streamChunk
				if json.Unmarshal([]byte(data), &chunk) != nil {
					continue // 跳过无法解析的行（如心跳）
				}
				if chunk.Error != nil && chunk.Error.Message != "" {
					return full.String(), fmt.Errorf("LLM 流式返回错误: %s", chunk.Error.Message)
				}
				for _, ch := range chunk.Choices {
					if ch.Delta.Content == "" {
						continue
					}
					full.WriteString(ch.Delta.Content)
					if derr := onDelta(ch.Delta.Content); derr != nil {
						return full.String(), derr
					}
				}
			}
		}

		if err == io.EOF {
			break
		}
		if err != nil {
			return full.String(), fmt.Errorf("读取流式响应失败: %w", err)
		}
	}

	return full.String(), nil
}

// guardNonJSON 检测非 JSON 响应（如 HTML 网页），多为 base_url 配错。
func guardNonJSON(resp *http.Response, raw []byte, endpoint string) error {
	contentType := resp.Header.Get("Content-Type")
	trimmed := strings.TrimSpace(string(raw))
	if strings.Contains(contentType, "json") || strings.Contains(contentType, "event-stream") {
		return nil
	}
	if strings.HasPrefix(trimmed, "{") || strings.HasPrefix(trimmed, "[") {
		return nil
	}
	return fmt.Errorf(
		"LLM 返回了非 JSON 响应(status=%d, content-type=%s)，请检查 base_url 是否为正确的 OpenAI 兼容 API 地址（应是厂商 API 根地址，如 https://api.deepseek.com，而非前端/网页地址）；实际请求了 %s，收到: %s",
		resp.StatusCode, contentType, endpoint, truncate(trimmed, 200))
}

// truncate 截断过长的原始响应，避免错误信息刷屏。
func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
