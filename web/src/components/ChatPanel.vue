<script setup lang="ts">
import { ref, watch } from 'vue'
import { useConversationsStore } from '../stores/conversations'
import { useChatStore } from '../stores/chat'
import MessageBubble from './MessageBubble.vue'

const conversations = useConversationsStore()
const chat = useChatStore()

const draft = ref('')
const scrollEl = ref<HTMLElement | null>(null)

// stick=true 时视图跟随最新消息；用户手动往上滚离底部则暂停跟随（不打断阅读），
// 滚回底部又自动恢复。滚动写入按帧节流，避免流式每吐一帧都强制回流。
let stick = true
let scrollScheduled = false

function onScroll() {
  const el = scrollEl.value
  if (!el) return
  stick = el.scrollHeight - el.scrollTop - el.clientHeight < 80
}

function scheduleScroll() {
  if (scrollScheduled || !stick) return
  scrollScheduled = true
  requestAnimationFrame(() => {
    scrollScheduled = false
    const el = scrollEl.value
    if (el && stick) el.scrollTop = el.scrollHeight
  })
}

// 消息数量变化或最后一条内容增长（流式）时，按需跟随滚动到底部。
watch(
  () => {
    const last = chat.messages[chat.messages.length - 1]
    return `${chat.messages.length}:${last ? last.content.length : 0}`
  },
  scheduleScroll,
)

// 切换会话时重新固定到底部。
watch(
  () => conversations.selectedId,
  () => {
    stick = true
  },
)

async function onSend() {
  const content = draft.value.trim()
  if (!content || chat.sending) return
  draft.value = ''
  stick = true // 自己发消息时强制回到底部
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

      <div
        ref="scrollEl"
        class="flex-1 space-y-4 overflow-y-auto px-6 py-4"
        @scroll.passive="onScroll"
      >
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
            v-if="chat.sending"
            type="button"
            class="rounded-lg bg-red-600 px-4 py-2 text-sm text-white hover:bg-red-700"
            @click="chat.stop()"
          >
            停止
          </button>
          <button
            v-else
            type="submit"
            :disabled="!draft.trim()"
            class="rounded-lg bg-blue-600 px-4 py-2 text-sm text-white hover:bg-blue-700 disabled:opacity-50"
          >
            发送
          </button>
        </form>
      </div>
    </template>
  </div>
</template>
