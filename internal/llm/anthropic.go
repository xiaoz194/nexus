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

// anthropicVersion 是 Anthropic Messages API 要求的版本头。
const anthropicVersion = "2023-06-01"

// AnthropicConfig 配置一个 Anthropic 原生 Provider。
type AnthropicConfig struct {
	BaseURL   string        // API 根地址，如 https://api.anthropic.com；也兼容以 /v1 结尾的中转地址
	APIKey    string        // 密钥（同时用于 x-api-key 与 Bearer）
	Model     string        // 默认模型；Agent 未指定 model_name 时使用
	MaxTokens int           // Anthropic 强制要求；<=0 时由上层回退
	Timeout   time.Duration // 非流式 HTTP 超时（流式不设硬超时，靠 ctx 取消）
}

// AnthropicProvider 通过 Anthropic 原生 /v1/messages 接口调用 Claude，
// 系统提示词以真正的顶层 system 参数下发（遵循度高于塞进消息内容）。
type AnthropicProvider struct {
	endpoint  string
	apiKey    string
	model     string
	maxTokens int
	client    *http.Client // 非流式：带超时
	streamer  *http.Client // 流式：不设超时，由请求 ctx 控制
}

// NewAnthropicProvider 构造一个 AnthropicProvider。
func NewAnthropicProvider(cfg AnthropicConfig) *AnthropicProvider {
	timeout := cfg.Timeout
	if timeout <= 0 {
		timeout = 60 * time.Second
	}
	// 智能拼接端点：已带 /v1 的中转地址只补 /messages，否则补 /v1/messages。
	base := strings.TrimRight(cfg.BaseURL, "/")
	endpoint := base + "/v1/messages"
	if strings.HasSuffix(base, "/v1") {
		endpoint = base + "/messages"
	}
	return &AnthropicProvider{
		endpoint:  endpoint,
		apiKey:    cfg.APIKey,
		model:     cfg.Model,
		maxTokens: cfg.MaxTokens,
		client:    &http.Client{Timeout: timeout},
		streamer:  &http.Client{}, // 流式请求可能持续很久，超时交给 ctx
	}
}

type anthropicMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type anthropicRequest struct {
	Model     string             `json:"model"`
	MaxTokens int                `json:"max_tokens"`
	System    string             `json:"system,omitempty"`
	Messages  []anthropicMessage `json:"messages"`
	Stream    bool               `json:"stream,omitempty"`
}

type anthropicResponse struct {
	Content []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"content"`
	Error *struct {
		Type    string `json:"type"`
		Message string `json:"message"`
	} `json:"error"`
}

// anthropicStreamEvent 是流式 SSE 中每个 data 块的结构。
type anthropicStreamEvent struct {
	Type  string `json:"type"`
	Delta struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"delta"`
	Error *struct {
		Type    string `json:"type"`
		Message string `json:"message"`
	} `json:"error"`
}

// resolveModel 选定实际使用的模型名。
func (p *AnthropicProvider) resolveModel(req ChatRequest) (string, error) {
	model := req.Model
	if model == "" {
		model = p.model
	}
	if model == "" {
		return "", fmt.Errorf("未指定模型名（Agent.model_name 与配置 llm.model 均为空）")
	}
	return model, nil
}

// maxTokensOrDefault 返回有效的 max_tokens（Anthropic 强制要求 > 0）。
func (p *AnthropicProvider) maxTokensOrDefault() int {
	if p.maxTokens > 0 {
		return p.maxTokens
	}
	return 4096
}

// buildRequestBody 构造 Anthropic 请求体。系统提示词作为顶层 system 下发。
func (p *AnthropicProvider) buildRequestBody(req ChatRequest, model string, stream bool) ([]byte, error) {
	msgs := make([]anthropicMessage, 0, len(req.Messages))
	for _, m := range req.Messages {
		msgs = append(msgs, anthropicMessage{Role: m.Role, Content: m.Content})
	}
	return json.Marshal(anthropicRequest{
		Model:     model,
		MaxTokens: p.maxTokensOrDefault(),
		System:    strings.TrimSpace(req.SystemPrompt),
		Messages:  msgs,
		Stream:    stream,
	})
}

// newRequest 构造带 Anthropic 鉴权头的 POST 请求。
func (p *AnthropicProvider) newRequest(ctx context.Context, body []byte) (*http.Request, error) {
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, p.endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-api-key", p.apiKey)
	httpReq.Header.Set("anthropic-version", anthropicVersion)
	// 同时带 Bearer，兼容部分中转网关；原生 Anthropic 会忽略它。
	httpReq.Header.Set("Authorization", "Bearer "+p.apiKey)
	return httpReq, nil
}

// Chat 实现 Provider：一次性返回完整回复。
func (p *AnthropicProvider) Chat(ctx context.Context, req ChatRequest) (string, error) {
	model, err := p.resolveModel(req)
	if err != nil {
		return "", err
	}

	body, err := p.buildRequestBody(req, model, false)
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

	var parsed anthropicResponse
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return "", fmt.Errorf("解析 LLM 响应失败(status=%d): %s", resp.StatusCode, truncate(strings.TrimSpace(string(raw)), 300))
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		if parsed.Error != nil && parsed.Error.Message != "" {
			return "", fmt.Errorf("LLM 返回错误(status=%d): %s", resp.StatusCode, parsed.Error.Message)
		}
		return "", fmt.Errorf("LLM 返回非 2xx(status=%d): %s", resp.StatusCode, truncate(strings.TrimSpace(string(raw)), 300))
	}

	var b strings.Builder
	for _, block := range parsed.Content {
		if block.Type == "text" {
			b.WriteString(block.Text)
		}
	}
	return b.String(), nil
}

// ChatStream 实现 Provider：以 SSE 流式读取，每块回调 onDelta，返回完整回复。
func (p *AnthropicProvider) ChatStream(ctx context.Context, req ChatRequest, onDelta StreamFunc) (string, error) {
	model, err := p.resolveModel(req)
	if err != nil {
		return "", err
	}

	body, err := p.buildRequestBody(req, model, true)
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
		var parsed anthropicResponse
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
			// 只关心 data: 行，忽略 event: 行与空行。
			if line != "" && strings.HasPrefix(line, "data:") {
				data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
				var ev anthropicStreamEvent
				if json.Unmarshal([]byte(data), &ev) != nil {
					continue // 跳过无法解析的行（如心跳）
				}
				switch ev.Type {
				case "content_block_delta":
					if ev.Delta.Type == "text_delta" && ev.Delta.Text != "" {
						full.WriteString(ev.Delta.Text)
						if derr := onDelta(ev.Delta.Text); derr != nil {
							return full.String(), derr
						}
					}
				case "error":
					if ev.Error != nil && ev.Error.Message != "" {
						return full.String(), fmt.Errorf("LLM 流式返回错误: %s", ev.Error.Message)
					}
					return full.String(), fmt.Errorf("LLM 流式返回错误")
				case "message_stop":
					return full.String(), nil
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
