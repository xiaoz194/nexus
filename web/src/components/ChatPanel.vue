<script setup lang="ts">
import { nextTick, ref, watch } from 'vue'
import { useConversationsStore } from '../stores/conversations'
import { useChatStore } from '../stores/chat'
import MessageBubble from './MessageBubble.vue'

const conversations = useConversationsStore()
const chat = useChatStore()

const draft = ref('')
const scrollEl = ref<HTMLElement | null>(null)

async function scrollToBottom() {
  await nextTick()
  const el = scrollEl.value
  if (el) el.scrollTop = el.scrollHeight
}

// 始终把视图固定到最新消息：新消息追加时，以及流式增量填充最后一条时都跟随滚动。
watch(
  () => {
    const last = chat.messages[chat.messages.length - 1]
    return `${chat.messages.length}:${last ? last.content.length : 0}`
  },
  scrollToBottom,
)

async function onSend() {
  const content = draft.value.trim()
  if (!content || chat.sending) return
  draft.value = ''
  await chat.send(content)
}
</script>

<template>
  <div class="flex h-full flex-1 flex-col bg-gray-100">
    <div v-if="!conversations.selectedId" class="flex flex-1 items-center justify-center">
      <p class="text-sm text-gray-400">选择或新建一个会话开始对话。</p>
    </div>

    <template v-else>
      <div class="border-b border-gray-200 bg-white px-6 py-3">
        <h2 class="truncate text-base font-semibold text-gray-800">
          {{ conversations.selected?.title }}
        </h2>
      </div>

      <div ref="scrollEl" class="flex-1 space-y-4 overflow-y-auto px-6 py-4">
        <p v-if="chat.loading" class="text-center text-sm text-gray-400">加载历史中…</p>
        <p
          v-else-if="chat.messages.length === 0"
          class="mt-8 text-center text-sm text-gray-400"
        >
          还没有消息，发送第一条试试。
        </p>
        <MessageBubble
          v-for="m in chat.messages"
          :key="m.id"
          :message="m"
          :retrying="chat.sending"
          @retry="chat.retry(m)"
        />
      </div>

      <div class="border-t border-gray-200 bg-white px-4 py-3">
        <form class="flex items-end gap-2" @submit.prevent="onSend">
          <textarea
            v-model="draft"
            rows="1"
            placeholder="输入消息，Enter 发送，Shift+Enter 换行"
            class="max-h-40 flex-1 resize-none rounded-lg border border-gray-300 px-3 py-2 text-sm focus:border-blue-500 focus:outline-none"
            @keydown.enter.exact.prevent="onSend"
          />
          <button
            type="submit"
            :disabled="!draft.trim() || chat.sending"
            class="rounded-lg bg-blue-600 px-4 py-2 text-sm text-white hover:bg-blue-700 disabled:opacity-50"
          >
            {{ chat.sending ? '发送中' : '发送' }}
          </button>
        </form>
      </div>
    </template>
  </div>
</template>
