import { defineStore } from 'pinia'
import { ref } from 'vue'
import type { User } from '../types'
import * as authApi from '../api/auth'
import { useToastStore } from './toast'

export const useAuthStore = defineStore('auth', () => {
  const toast = useToastStore()

  const user = ref<User | null>(null)
  const ready = ref(false) // 首次 me() 探测是否完成（未完成前不渲染主界面/登录页）

  // fetchMe 启动时探测已有 cookie 会话；未登录（401）静默置空，不弹错。
  async function fetchMe() {
    try {
      user.value = await authApi.me()
    } catch {
      user.value = null
    } finally {
      ready.value = true
    }
  }

  async function login(username: string, password: string): Promise<boolean> {
    try {
      user.value = await authApi.login(username, password)
      return true
    } catch (e) {
      toast.error((e as Error).message)
      return false
    }
  }

  async function register(
    username: string,
    password: string,
    displayName?: string,
  ): Promise<boolean> {
    try {
      user.value = await authApi.register(username, password, displayName)
      return true
    } catch (e) {
      toast.error((e as Error).message)
      return false
    }
  }

  async function logout() {
    try {
      await authApi.logout()
    } catch {
      /* 即便请求失败也本地登出 */
    }
    user.value = null
  }

  // clear 供 401 拦截时把登录态就地失效（无需再请求后端）。
  function clear() {
    user.value = null
  }

  return { user, ready, fetchMe, login, register, logout, clear }
})
