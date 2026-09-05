import { api, ApiError, BASE } from './client'
import type { Message } from '../types'

export const getHistory = (agentId: string, conversationId: string) =>
  api.get<Message[]>(`/agents/${agentId}/conversations/${conversationId}/messages`)

// 后端只返回 assistant 回复；user 消息由服务端保存。
export const sendMessage = (agentId: string, conversationId: string, content: string) =>
  api.post<Message>(`/agents/${agentId}/conversations/${conversationId}/messages`, { content })

// 流式回调。onDelta 每收到一块文本片段触发一次。
export interface StreamHandlers {
  onDelta: (delta: string) => void
}

// sendMessageStream 以 SSE 流式发送消息：逐块回调 onDelta，
// 结束时 resolve 出完整落库的 assistant 消息。
// 浏览器原生 EventSource 只支持 GET，这里用 fetch + ReadableStream 读 SSE。
export async function sendMessageStream(
  agentId: string,
  conversationId: string,
  content: string,
  handlers: StreamHandlers,
): Promise<Message> {
  let res: Response
  try {
    res = await fetch(
      `${BASE}/agents/${agentId}/conversations/${conversationId}/messages/stream`,
      {
        method: 'POST',
        headers: { 'Content-Type': 'application/json', Accept: 'text/event-stream' },
        body: JSON.stringify({ content }),
        // 流式 fetch 绕过了 client.ts，需单独带上会话 cookie。
        credentials: 'include',
      },
    )
  } catch {
    throw new ApiError(0, '无法连接到服务器，请确认后端已启动')
  }

  // 出错时后端在开始流式前会返回普通 JSON 错误（正确状态码）。
  if (!res.ok || !res.body) {
    let message = `请求失败 (${res.status})`
    try {
      const data = await res.json()
      if (data && typeof data === 'object' && 'error' in data) message = String(data.error)
    } catch {
      /* ignore */
    }
    throw new ApiError(res.status, message)
  }

  const reader = res.body.getReader()
  const decoder = new TextDecoder()
  let buffer = ''
  let finalMessage: Message | null = null
  let streamError: string | null = null

  // 处理单条完整的 SSE 事件（可能含多行，取其中的 data: 行）。
  const handleEvent = (raw: string) => {
    for (const line of raw.split('\n')) {
      const trimmed = line.trimStart()
      if (!trimmed.startsWith('data:')) continue
      const payload = trimmed.slice(5).trim()
      if (!payload) continue
      let evt: { delta?: string; done?: boolean; message?: Message; error?: string }
      try {
        evt = JSON.parse(payload)
      } catch {
        continue
      }
      if (typeof evt.delta === 'string') handlers.onDelta(evt.delta)
      if (evt.error) streamError = evt.error
      if (evt.done && evt.message) finalMessage = evt.message
    }
  }

  for (;;) {
    const { done, value } = await reader.read()
    if (done) break
    buffer += decoder.decode(value, { stream: true })
    // SSE 事件以空行分隔。
    let sep: number
    while ((sep = buffer.indexOf('\n\n')) !== -1) {
      const event = buffer.slice(0, sep)
      buffer = buffer.slice(sep + 2)
      handleEvent(event)
    }
  }
  if (buffer.trim()) handleEvent(buffer)

  if (streamError) throw new ApiError(500, streamError)
  if (!finalMessage) throw new ApiError(0, '流式响应异常结束')
  return finalMessage
}
