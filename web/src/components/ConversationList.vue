<script setup lang="ts">
import { nextTick, ref } from 'vue'
import type { ComponentPublicInstance } from 'vue'
import { useAgentsStore } from '../stores/agents'
import { useConversationsStore } from '../stores/conversations'
import type { Conversation } from '../types'

const agents = useAgentsStore()
const store = useConversationsStore()

// 行内重命名状态：同一时刻只有一个会话处于编辑态。
const editingId = ref<string | null>(null)
const editText = ref('')
const editInput = ref<HTMLInputElement | null>(null)

// 用函数 ref 而非字符串 ref：v-for 内的字符串 ref 在 Vue 3 里会被收集成数组，
// 这里同一时刻只有一个输入框，用函数 ref 直接拿到该元素更稳妥。
function setEditInput(el: Element | ComponentPublicInstance | null) {
  editInput.value = el as HTMLInputElement | null
}

async function startRename(conv: Conversation) {
  editingId.value = conv.id
  editText.value = conv.title
  await nextTick()
  editInput.value?.focus()
  editInput.value?.select()
}

// 提交重命名。Enter 与 blur 都会触发，靠 editingId 判断避免重复提交。
async function commitRename(conv: Conversation) {
  if (editingId.value !== conv.id) return
  const t = editText.value.trim()
  editingId.value = null
  if (t && t !== conv.title) await store.rename(conv.id, t)
}

function cancelRename() {
  editingId.value = null
}

// 新建后立即进入编辑态，用户可直接给会话起名（不改就保留「新会话」）。
async function onCreate() {
  const conv = await store.create()
  if (conv) startRename(conv)
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

        <div
          v-for="conv in store.conversations"
          :key="conv.id"
          class="group mb-1 flex w-full cursor-pointer items-center justify-between rounded-lg px-3 py-2 text-left transition"
          :class="
            conv.id === store.selectedId
              ? 'bg-blue-100 text-blue-900'
              : 'hover:bg-gray-100 text-gray-700'
          "
          @click="store.select(conv.id)"
        >
          <input
            v-if="editingId === conv.id"
            :ref="setEditInput"
            v-model="editText"
            type="text"
            class="min-w-0 flex-1 rounded border border-blue-400 bg-white px-1.5 py-0.5 text-sm text-gray-800 focus:outline-none"
            @click.stop
            @keydown.enter.prevent="commitRename(conv)"
            @keydown.esc="cancelRename"
            @blur="commitRename(conv)"
          />
          <span v-else class="truncate text-sm">{{ conv.title }}</span>

          <span
            v-if="editingId !== conv.id"
            class="ml-2 flex shrink-0 gap-2 text-xs opacity-0 group-hover:opacity-100"
          >
            <span class="text-gray-400 hover:text-blue-600" @click.stop="startRename(conv)">
              重命名
            </span>
            <span class="text-gray-400 hover:text-red-600" @click.stop="onDelete(conv)">
              删除
            </span>
          </span>
        </div>
      </template>
    </div>
  </div>
</template>
