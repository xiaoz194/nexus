<script setup lang="ts">
import { useAgentsStore } from '../stores/agents'
import { useConversationsStore } from '../stores/conversations'
import type { Conversation } from '../types'

const agents = useAgentsStore()
const store = useConversationsStore()

async function onCreate() {
  await store.create()
}

async function onDelete(conv: Conversation) {
  if (!confirm(`删除会话「${conv.title}」及其全部消息？`)) return
  await store.remove(conv.id)
}
</script>

<template>
  <div class="flex h-full w-72 flex-col border-r border-gray-200 bg-white">
    <div class="flex items-center justify-between px-4 py-3">
      <h2 class="text-base font-semibold text-gray-800">会话</h2>
      <button
        v-if="agents.selectedId"
        class="rounded-lg bg-blue-600 px-2 py-1 text-xs text-white hover:bg-blue-700"
        @click="onCreate"
      >
        + 新建
      </button>
    </div>

    <div class="flex-1 overflow-y-auto px-2 pb-2">
      <p v-if="!agents.selectedId" class="px-2 py-4 text-sm text-gray-400">
        请先在左侧选择一个 Agent。
      </p>
      <template v-else>
        <p v-if="store.loading" class="px-2 py-4 text-sm text-gray-400">加载中…</p>
        <p v-else-if="store.conversations.length === 0" class="px-2 py-4 text-sm text-gray-400">
          该 Agent 还没有会话。
        </p>

        <button
          v-for="conv in store.conversations"
          :key="conv.id"
          class="group mb-1 flex w-full items-center justify-between rounded-lg px-3 py-2 text-left transition"
          :class="
            conv.id === store.selectedId
              ? 'bg-blue-100 text-blue-900'
              : 'hover:bg-gray-100 text-gray-700'
          "
          @click="store.select(conv.id)"
        >
          <span class="truncate text-sm">{{ conv.title }}</span>
          <span
            class="ml-2 text-xs text-gray-400 opacity-0 hover:text-red-600 group-hover:opacity-100"
            @click.stop="onDelete(conv)"
          >
            删除
          </span>
        </button>
      </template>
    </div>
  </div>
</template>
