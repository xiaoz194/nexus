<script setup lang="ts">
import { ref } from 'vue'
import { useAgentsStore } from '../stores/agents'
import type { Agent, AgentInput } from '../types'
import AgentFormModal from './AgentFormModal.vue'
import UserMenu from './UserMenu.vue'

const store = useAgentsStore()

const modalOpen = ref(false)
const editing = ref<Agent | null>(null)

function openCreate() {
  editing.value = null
  modalOpen.value = true
}

function openEdit(agent: Agent) {
  editing.value = agent
  modalOpen.value = true
}

async function onSubmit(input: AgentInput) {
  const ok = editing.value
    ? await store.update(editing.value.id, input)
    : !!(await store.create(input))
  if (ok) modalOpen.value = false
}

async function onDelete(agent: Agent) {
  if (!confirm(`删除 Agent「${agent.name}」？其所有会话和消息都会被清除。`)) return
  await store.remove(agent.id)
}
</script>

<template>
  <div class="flex h-full w-64 flex-col border-r border-gray-200 bg-gray-50">
    <!-- 左上角当前用户头像 + 下拉菜单 -->
    <div class="border-b border-gray-200 px-2 py-2">
      <UserMenu />
    </div>

    <div class="flex items-center justify-between px-4 py-3">
      <h1 class="text-base font-semibold text-gray-800">Agents</h1>
      <button
        class="rounded-lg bg-blue-600 px-2 py-1 text-xs text-white hover:bg-blue-700"
        @click="openCreate"
      >
        + 新建
      </button>
    </div>

    <div class="flex-1 overflow-y-auto px-2 pb-2">
      <p v-if="store.loading" class="px-2 py-4 text-sm text-gray-400">加载中…</p>
      <p v-else-if="store.agents.length === 0" class="px-2 py-4 text-sm text-gray-400">
        还没有 Agent，点击「新建」创建一个。
      </p>

      <button
        v-for="agent in store.agents"
        :key="agent.id"
        class="group mb-1 flex w-full flex-col rounded-lg px-3 py-2 text-left transition"
        :class="
          agent.id === store.selectedId
            ? 'bg-blue-100 text-blue-900'
            : 'hover:bg-gray-100 text-gray-700'
        "
        @click="store.select(agent.id)"
      >
        <span class="truncate text-sm font-medium">{{ agent.name }}</span>
        <span v-if="agent.description" class="truncate text-xs text-gray-500">
          {{ agent.description }}
        </span>
        <span class="mt-1 flex gap-3 text-xs opacity-0 group-hover:opacity-100">
          <span class="text-gray-500 hover:text-blue-600" @click.stop="openEdit(agent)">编辑</span>
          <span class="text-gray-500 hover:text-red-600" @click.stop="onDelete(agent)">删除</span>
        </span>
      </button>
    </div>

    <AgentFormModal
      :open="modalOpen"
      :agent="editing"
      @submit="onSubmit"
      @close="modalOpen = false"
    />
  </div>
</template>
