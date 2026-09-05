import { defineStore } from 'pinia'
import { ref } from 'vue'

export interface ToastItem {
  id: number
  kind: 'error' | 'info'
  text: string
}

let seq = 0

export const useToastStore = defineStore('toast', () => {
  const items = ref<ToastItem[]>([])

  function push(kind: ToastItem['kind'], text: string) {
    const id = ++seq
    items.value.push({ id, kind, text })
    setTimeout(() => dismiss(id), 4000)
  }

  const error = (text: string) => push('error', text)
  const info = (text: string) => push('info', text)

  function dismiss(id: number) {
    items.value = items.value.filter((t) => t.id !== id)
  }

  return { items, error, info, dismiss }
})
