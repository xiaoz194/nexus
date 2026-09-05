<script setup lang="ts">
import { useToastStore } from '../stores/toast'

const toast = useToastStore()
</script>

<template>
  <div class="pointer-events-none fixed bottom-4 right-4 z-50 flex flex-col gap-2">
    <div
      v-for="t in toast.items"
      :key="t.id"
      class="pointer-events-auto flex max-w-sm items-start gap-2 rounded-lg px-4 py-2 text-sm text-white shadow-lg"
      :class="t.kind === 'error' ? 'bg-red-600' : 'bg-gray-800'"
    >
      <span class="flex-1">{{ t.text }}</span>
      <button class="opacity-70 hover:opacity-100" @click="toast.dismiss(t.id)">✕</button>
    </div>
  </div>
</template>
