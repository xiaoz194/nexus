import { defineStore } from 'pinia'
import { computed, ref } from 'vue'
import type { Agent, AgentInput } from '../types'
import * as agentsApi from '../api/agents'
import { useToastStore } from './toast'

export const useAgentsStore = defineStore('agents', () => {
  const toast = useToastStore()

  const agents = ref<Agent[]>([])
  const selectedId = ref<string | null>(null)
  const loading = ref(false)

  const selected = computed(() => agents.value.find((a) => a.id === selectedId.value) ?? null)

  async function load() {
    loading.value = true
    try {
      agents.value = await agentsApi.listAgents()
      // 若当前选中的 Agent 已不存在，则清除选中。
      if (selectedId.value && !agents.value.some((a) => a.id === selectedId.value)) {
        selectedId.value = null
      }
    } catch (e) {
      toast.error((e as Error).message)
    } finally {
      loading.value = false
    }
  }

  function select(id: string | null) {
    selectedId.value = id
  }

  async function create(input: AgentInput): Promise<Agent | null> {
    try {
      const agent = await agentsApi.createAgent(input)
      agents.value.push(agent)
      selectedId.value = agent.id
      return agent
    } catch (e) {
      toast.error((e as Error).message)
      return null
    }
  }

  async function update(id: string, input: AgentInput): Promise<boolean> {
    try {
      const updated = await agentsApi.updateAgent(id, input)
      const i = agents.value.findIndex((a) => a.id === id)
      if (i !== -1) agents.value[i] = updated
      return true
    } catch (e) {
      toast.error((e as Error).message)
      return false
    }
  }

  async function remove(id: string): Promise<boolean> {
    try {
      await agentsApi.deleteAgent(id)
      agents.value = agents.value.filter((a) => a.id !== id)
      if (selectedId.value === id) selectedId.value = null
      return true
    } catch (e) {
      toast.error((e as Error).message)
      return false
    }
  }

  return { agents, selectedId, selected, loading, load, select, create, update, remove }
})
