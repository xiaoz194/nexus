import { defineStore } from 'pinia'
import { computed, ref } from 'vue'
import type { Conversation } from '../types'
import * as convApi from '../api/conversations'
import { useToastStore } from './toast'

export const useConversationsStore = defineStore('conversations', () => {
  const toast = useToastStore()

  const conversations = ref<Conversation[]>([])
  const selectedId = ref<string | null>(null)
  const loading = ref(false)
  // 这些会话所属的 Agent（用于拼接消息路径）。
  const agentId = ref<string | null>(null)

  const selected = computed(
    () => conversations.value.find((c) => c.id === selectedId.value) ?? null,
  )

  // 切换 Agent 时重置，强化隔离。
  function reset() {
    conversations.value = []
    selectedId.value = null
    agentId.value = null
  }

  async function loadFor(id: string) {
    agentId.value = id
    selectedId.value = null
    loading.value = true
    try {
      conversations.value = await convApi.listConversations(id)
    } catch (e) {
      conversations.value = []
      toast.error((e as Error).message)
    } finally {
      loading.value = false
    }
  }

  function select(id: string | null) {
    selectedId.value = id
  }

  async function create(title?: string): Promise<Conversation | null> {
    if (!agentId.value) return null
    try {
      const conv = await convApi.createConversation(agentId.value, title)
      conversations.value.push(conv)
      selectedId.value = conv.id
      return conv
    } catch (e) {
      toast.error((e as Error).message)
      return null
    }
  }

  async function rename(id: string, title: string): Promise<boolean> {
    if (!agentId.value) return false
    const t = title.trim()
    if (!t) return false
    try {
      const conv = await convApi.updateConversation(agentId.value, id, t)
      const idx = conversations.value.findIndex((c) => c.id === id)
      if (idx !== -1) conversations.value[idx] = conv
      return true
    } catch (e) {
      toast.error((e as Error).message)
      return false
    }
  }

  async function remove(id: string): Promise<boolean> {
    if (!agentId.value) return false
    try {
      await convApi.deleteConversation(agentId.value, id)
      conversations.value = conversations.value.filter((c) => c.id !== id)
      if (selectedId.value === id) selectedId.value = null
      return true
    } catch (e) {
      toast.error((e as Error).message)
      return false
    }
  }

  return {
    conversations,
    selectedId,
    selected,
    loading,
    agentId,
    reset,
    loadFor,
    select,
    create,
    rename,
    remove,
  }
})
