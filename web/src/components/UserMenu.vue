<script setup lang="ts">
import { computed, onBeforeUnmount, ref } from 'vue'
import { useAuthStore } from '../stores/auth'

const auth = useAuthStore()

const open = ref(false)
const rootEl = ref<HTMLElement | null>(null)

const displayName = computed(
  () => auth.user?.display_name || auth.user?.username || '',
)

// 头像首字母：优先昵称，回退用户名；取首个字符大写（中文直接取首字）。
const initial = computed(() => {
  const s = displayName.value.trim()
  return s ? s[0].toUpperCase() : '?'
})

// 依据用户名稳定地挑一个背景色，让不同用户头像有区分度。
const palette = [
  'bg-blue-500',
  'bg-emerald-500',
  'bg-violet-500',
  'bg-amber-500',
  'bg-rose-500',
  'bg-cyan-500',
  'bg-indigo-500',
]
const avatarColor = computed(() => {
  const key = auth.user?.username || ''
  let sum = 0
  for (let i = 0; i < key.length; i++) sum += key.charCodeAt(i)
  return palette[sum % palette.length]
})

const joinedAt = computed(() => {
  if (!auth.user?.created_at) return ''
  const d = new Date(auth.user.created_at)
  if (Number.isNaN(d.getTime())) return ''
  return d.toLocaleDateString('zh-CN')
})

function toggle() {
  open.value = !open.value
  if (open.value) {
    document.addEventListener('click', onDocClick, true)
  } else {
    document.removeEventListener('click', onDocClick, true)
  }
}

function close() {
  open.value = false
  document.removeEventListener('click', onDocClick, true)
}

// 点击组件外部关闭下拉。
function onDocClick(e: MouseEvent) {
  if (rootEl.value && !rootEl.value.contains(e.target as Node)) close()
}

async function onLogout() {
  close()
  if (!confirm('确定要登出吗？')) return
  await auth.logout()
}

onBeforeUnmount(() => {
  document.removeEventListener('click', onDocClick, true)
})
</script>

<template>
  <div ref="rootEl" class="relative">
    <button
      class="flex w-full items-center gap-2 rounded-lg px-2 py-1.5 text-left transition hover:bg-gray-100"
      @click="toggle"
    >
      <span
        class="flex h-8 w-8 shrink-0 items-center justify-center rounded-full text-sm font-semibold text-white"
        :class="avatarColor"
      >
        {{ initial }}
      </span>
      <span class="min-w-0 flex-1 truncate text-sm font-medium text-gray-800" :title="displayName">
        {{ displayName }}
      </span>
      <svg
        class="h-4 w-4 shrink-0 text-gray-400 transition"
        :class="{ 'rotate-180': open }"
        viewBox="0 0 20 20"
        fill="currentColor"
      >
        <path
          fill-rule="evenodd"
          d="M5.23 7.21a.75.75 0 011.06.02L10 11.17l3.71-3.94a.75.75 0 111.08 1.04l-4.25 4.5a.75.75 0 01-1.08 0l-4.25-4.5a.75.75 0 01.02-1.06z"
          clip-rule="evenodd"
        />
      </svg>
    </button>

    <!-- 下拉框 -->
    <div
      v-if="open"
      class="absolute left-0 top-full z-20 mt-1 w-60 overflow-hidden rounded-xl border border-gray-200 bg-white shadow-lg"
    >
      <!-- 个人信息 -->
      <div class="flex items-center gap-3 px-4 py-3">
        <span
          class="flex h-10 w-10 shrink-0 items-center justify-center rounded-full text-base font-semibold text-white"
          :class="avatarColor"
        >
          {{ initial }}
        </span>
        <div class="min-w-0">
          <p class="truncate text-sm font-semibold text-gray-800" :title="displayName">
            {{ displayName }}
          </p>
          <p class="truncate text-xs text-gray-500">@{{ auth.user?.username }}</p>
        </div>
      </div>

      <div v-if="joinedAt" class="border-t border-gray-100 px-4 py-2">
        <p class="text-xs text-gray-400">注册于 {{ joinedAt }}</p>
      </div>

      <div class="border-t border-gray-100 py-1">
        <button
          class="flex w-full items-center gap-2 px-4 py-2 text-left text-sm text-red-600 transition hover:bg-red-50"
          @click="onLogout"
        >
          <svg class="h-4 w-4" viewBox="0 0 20 20" fill="currentColor">
            <path
              fill-rule="evenodd"
              d="M3 4.25A2.25 2.25 0 015.25 2h5.5A2.25 2.25 0 0113 4.25v2a.75.75 0 01-1.5 0v-2a.75.75 0 00-.75-.75h-5.5a.75.75 0 00-.75.75v11.5c0 .414.336.75.75.75h5.5a.75.75 0 00.75-.75v-2a.75.75 0 011.5 0v2A2.25 2.25 0 0110.75 18h-5.5A2.25 2.25 0 013 15.75V4.25z"
              clip-rule="evenodd"
            />
            <path
              fill-rule="evenodd"
              d="M6 10a.75.75 0 01.75-.75h6.638L11.23 7.29a.75.75 0 111.04-1.08l3 2.875a.75.75 0 010 1.08l-3 2.875a.75.75 0 11-1.04-1.08l2.158-1.96H6.75A.75.75 0 016 10z"
              clip-rule="evenodd"
            />
          </svg>
          退出登录
        </button>
      </div>
    </div>
  </div>
</template>
