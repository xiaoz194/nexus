<script setup lang="ts">
import { ref } from 'vue'
import { useAuthStore } from '../stores/auth'
import { useToastStore } from '../stores/toast'

const auth = useAuthStore()
const toast = useToastStore()

const mode = ref<'login' | 'register'>('login')
const username = ref('')
const password = ref('')
const displayName = ref('')
const submitting = ref(false)

function toggleMode() {
  mode.value = mode.value === 'login' ? 'register' : 'login'
}

async function onSubmit() {
  const u = username.value.trim()
  if (!u || !password.value) {
    toast.error('请输入用户名和密码')
    return
  }
  if (mode.value === 'register' && password.value.length < 6) {
    toast.error('密码至少 6 位')
    return
  }

  submitting.value = true
  try {
    const ok =
      mode.value === 'login'
        ? await auth.login(u, password.value)
        : await auth.register(u, password.value, displayName.value.trim() || undefined)
    if (!ok) return
    // 成功后 App.vue 的门禁会自动切到主界面。
  } finally {
    submitting.value = false
  }
}
</script>

<template>
  <div class="flex h-screen w-screen items-center justify-center bg-gray-100">
    <div class="w-80 rounded-2xl bg-white p-8 shadow-sm">
      <h1 class="mb-1 text-center text-xl font-semibold text-gray-800">Nexus</h1>
      <p class="mb-6 text-center text-sm text-gray-400">
        {{ mode === 'login' ? '登录以继续' : '注册一个新账号' }}
      </p>

      <form class="space-y-3" @submit.prevent="onSubmit">
        <input
          v-model="username"
          type="text"
          autocomplete="username"
          placeholder="用户名"
          class="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm focus:border-blue-500 focus:outline-none"
        />
        <input
          v-model="password"
          type="password"
          :autocomplete="mode === 'login' ? 'current-password' : 'new-password'"
          placeholder="密码"
          class="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm focus:border-blue-500 focus:outline-none"
        />
        <input
          v-if="mode === 'register'"
          v-model="displayName"
          type="text"
          placeholder="昵称（可选）"
          class="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm focus:border-blue-500 focus:outline-none"
        />

        <button
          type="submit"
          :disabled="submitting"
          class="w-full rounded-lg bg-blue-600 px-4 py-2 text-sm text-white hover:bg-blue-700 disabled:opacity-50"
        >
          {{ submitting ? '请稍候…' : mode === 'login' ? '登录' : '注册' }}
        </button>
      </form>

      <p class="mt-4 text-center text-xs text-gray-400">
        {{ mode === 'login' ? '还没有账号？' : '已有账号？' }}
        <button class="text-blue-500 hover:underline" @click="toggleMode">
          {{ mode === 'login' ? '去注册' : '去登录' }}
        </button>
      </p>
    </div>
  </div>
</template>
