import { defineStore } from 'pinia'
import { ref } from 'vue'
import type { Message, Role } from '../types'
import * as msgApi from '../api/messages'
import { useToastStore } from './toast'

// 消息 + 客户端投递状态，用于乐观渲染。
export interface ChatMessage extends Message {
  pending?: boolean
  failed?: boolean
  streaming?: boolean // assistant 消息正在逐块接收中
}

let tmpSeq = 0

export const useChatStore = defineStore('chat', () => {
  const toast = useToastStore()

  const messages = ref<ChatMessage[]>([])
  const loading = ref(false)
  const sending = ref(false)
  const agentId = ref<string | null>(null)
  const conversationId = ref<string | null>(null)

  function reset() {
    messages.value = []
    agentId.value = null
    conversationId.value = null
  }

  async function loadFor(aId: string, cId: string) {
    agentId.value = aId
    conversationId.value = cId
    loading.value = true
    try {
      messages.value = await msgApi.getHistory(aId, cId)
    } catch (e) {
      messages.value = []
      toast.error((e as Error).message)
    } finally {
      loading.value = false
    }
  }

  // runExchange 跑一轮「流式请求 + 乐观状态收尾」，send 与 retry 共用。
  // userRef / assistantRef 必须是 messages 数组里的响应式代理（改它们才会触发渲染）。
  async function runExchange(
    aId: string,
    cId: string,
    content: string,
    userRef: ChatMessage,
    assistantRef: ChatMessage,
  ) {
    sending.value = true
    try {
      const reply = await msgApi.sendMessageStream(aId, cId, content, {
        onDelta: (delta) => {
          assistantRef.content += delta
        },
      })
      userRef.pending = false
      userRef.failed = false
      // 用服务端落库的真实消息（含最终 id/时间）替换占位内容。
      assistantRef.id = reply.id
      assistantRef.content = reply.content
      assistantRef.created_at = reply.created_at
      assistantRef.streaming = false
      assistantRef.failed = false
    } catch (e) {
      userRef.pending = false
      userRef.failed = true
      // 移除空的 assistant 占位气泡；若已收到部分内容则标记失败保留。
      if (assistantRef.content) {
        assistantRef.streaming = false
        assistantRef.failed = true
      } else {
        const i = messages.value.indexOf(assistantRef)
        if (i !== -1) messages.value.splice(i, 1)
      }
      toast.error((e as Error).message)
    } finally {
      sending.value = false
    }
  }

  async function send(content: string) {
    const aId = agentId.value
    const cId = conversationId.value
    if (!aId || !cId) return

    // 乐观渲染 user 消息。
    const optimistic: ChatMessage = {
      id: `tmp-${++tmpSeq}`,
      conversation_id: cId,
      role: 'user' as Role,
      content,
      created_at: new Date().toISOString(),
      pending: true,
    }
    // assistant 占位气泡，随流式 delta 增量填充。
    const assistant: ChatMessage = {
      id: `tmp-a-${++tmpSeq}`,
      conversation_id: cId,
      role: 'assistant' as Role,
      content: '',
      created_at: new Date().toISOString(),
      streaming: true,
    }
    messages.value.push(optimistic, assistant)

    // push 进 ref 数组后，须通过数组返回的响应式代理来修改；
    // 直接改上面的原始对象引用不会触发重新渲染。
    const userRef = messages.value[messages.value.length - 2]
    const assistantRef = messages.value[messages.value.length - 1]
    await runExchange(aId, cId, content, userRef, assistantRef)
  }

  // retry 重发一条发送失败的 user 消息（后端失败时不落库，故重试不会产生重复）。
  // 就地复用该 user 气泡与其后的 assistant 气泡，避免消息位置跳动。
  async function retry(msg: ChatMessage) {
    if (sending.value) return
    const aId = agentId.value
    const cId = conversationId.value
    if (!aId || !cId) return

    const i = messages.value.indexOf(msg)
    if (i === -1 || messages.value[i].role !== 'user') return

    const userRef = messages.value[i]
    userRef.pending = true
    userRef.failed = false

    // 复用紧随其后的 assistant 气泡；若不存在（上次失败时空占位已被移除）则新插一个。
    let assistantRef = messages.value[i + 1]
    if (assistantRef && assistantRef.role === 'assistant') {
      assistantRef.content = ''
      assistantRef.failed = false
      assistantRef.streaming = true
    } else {
      const placeholder: ChatMessage = {
        id: `tmp-a-${++tmpSeq}`,
        conversation_id: cId,
        role: 'assistant' as Role,
        content: '',
        created_at: new Date().toISOString(),
        streaming: true,
      }
      messages.value.splice(i + 1, 0, placeholder)
      assistantRef = messages.value[i + 1]
    }

    await runExchange(aId, cId, userRef.content, userRef, assistantRef)
  }

  return { messages, loading, sending, agentId, conversationId, reset, loadFor, send, retry }
})
