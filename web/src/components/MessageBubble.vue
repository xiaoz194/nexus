<script setup lang="ts">
import { computed } from 'vue'
import type { ChatMessage } from '../stores/chat'

const props = defineProps<{ message: ChatMessage; retrying?: boolean }>()
const emit = defineEmits<{ (e: 'retry'): void }>()

const isUser = computed(() => props.message.role === 'user')

function formatTime(iso: string): string {
  const d = new Date(iso)
  if (Number.isNaN(d.getTime())) return ''
  return d.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })
}
</script>

<template>
  <div class="flex" :class="isUser ? 'justify-end' : 'justify-start'">
    <div class="max-w-[75%]">
      <div
        class="whitespace-pre-wrap break-words rounded-2xl px-4 py-2 text-sm"
        :class="[
          isUser ? 'bg-blue-600 text-white' : 'bg-white text-gray-800 border border-gray-200',
          message.failed ? 'opacity-60 ring-1 ring-red-400' : '',
        ]"
      >
        <template v-if="message.streaming && !message.content">
          <span class="inline-flex gap-1 py-1 align-middle">
            <span class="h-1.5 w-1.5 animate-bounce rounded-full bg-gray-400 [animation-delay:-0.3s]"></span>
            <span class="h-1.5 w-1.5 animate-bounce rounded-full bg-gray-400 [animation-delay:-0.15s]"></span>
            <span class="h-1.5 w-1.5 animate-bounce rounded-full bg-gray-400"></span>
          </span>
        </template>
        <template v-else>{{ message.content
          }}<span v-if="message.streaming" class="ml-0.5 inline-block animate-pulse">▋</span></template>
      </div>
      <div
        class="mt-1 flex items-center gap-2 text-[11px] text-gray-400"
        :class="isUser ? 'justify-end' : 'justify-start'"
      >
        <span>{{ isUser ? '我' : '助手' }}</span>
        <span v-if="message.pending">发送中…</span>
        <span v-else-if="message.streaming">生成中…</span>
        <template v-else-if="message.failed">
          <span class="text-red-500">发送失败</span>
          <button
            v-if="isUser"
            type="button"
            :disabled="retrying"
            class="text-blue-500 underline-offset-2 hover:underline disabled:opacity-50"
            @click="emit('retry')"
          >
            重试
          </button>
        </template>
        <span v-else>{{ formatTime(message.created_at) }}</span>
      </div>
    </div>
  </div>
</template>
