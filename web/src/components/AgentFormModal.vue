<script setup lang="ts">
import { reactive, watch } from 'vue'
import type { Agent, AgentInput } from '../types'

const props = defineProps<{
  open: boolean
  // 编辑时为已有 Agent；新建时为 null。
  agent: Agent | null
}>()

const emit = defineEmits<{
  (e: 'submit', input: AgentInput): void
  (e: 'close'): void
}>()

const form = reactive<AgentInput>({
  name: '',
  description: '',
  system_prompt: '',
  // openai = OpenAI 兼容真实大模型；anthropic = Claude 原生；mock = 本地回显。
  model_provider: 'openai',
  model_name: 'deepseek-chat',
  base_url: 'https://api.deepseek.com',
  api_key: '',
  max_tokens: 4096,
  temperature: 0.7,
})

// 每次弹窗打开时重置表单。
watch(
  () => props.open,
  (open) => {
    if (!open) return
    if (props.agent) {
      form.name = props.agent.name
      form.description = props.agent.description
      form.system_prompt = props.agent.system_prompt
      form.model_provider = props.agent.model_provider
      form.model_name = props.agent.model_name
      form.base_url = props.agent.base_url
      form.api_key = props.agent.api_key
      form.max_tokens = props.agent.max_tokens
      form.temperature = props.agent.temperature
    } else {
      form.name = ''
      form.description = ''
      form.system_prompt = ''
      form.model_provider = 'openai'
      form.model_name = 'deepseek-chat'
      form.base_url = 'https://api.deepseek.com'
      form.api_key = ''
      form.max_tokens = 4096
      form.temperature = 0.7
    }
  },
)

function submit() {
  if (!form.name.trim()) return
  emit('submit', { ...form, name: form.name.trim() })
}
</script>

<template>
  <div
    v-if="open"
    class="fixed inset-0 z-40 flex items-center justify-center bg-black/40 p-4"
    @click.self="emit('close')"
  >
    <div class="w-full max-w-lg rounded-xl bg-white p-6 shadow-xl">
      <h2 class="mb-4 text-lg font-semibold">{{ agent ? '编辑 Agent' : '新建 Agent' }}</h2>

      <form class="space-y-3" @submit.prevent="submit">
        <label class="block">
          <span class="text-sm text-gray-600">名称 <span class="text-red-500">*</span></span>
          <input
            v-model="form.name"
            type="text"
            required
            class="mt-1 w-full rounded-lg border border-gray-300 px-3 py-2 text-sm focus:border-blue-500 focus:outline-none"
            placeholder="例如：翻译官"
          />
        </label>

        <label class="block">
          <span class="text-sm text-gray-600">描述</span>
          <input
            v-model="form.description"
            type="text"
            class="mt-1 w-full rounded-lg border border-gray-300 px-3 py-2 text-sm focus:border-blue-500 focus:outline-none"
          />
        </label>

        <label class="block">
          <span class="text-sm text-gray-600">系统提示词</span>
          <textarea
            v-model="form.system_prompt"
            rows="3"
            class="mt-1 w-full rounded-lg border border-gray-300 px-3 py-2 text-sm focus:border-blue-500 focus:outline-none"
          />
        </label>

        <div class="flex gap-3">
          <label class="block flex-1">
            <span class="text-sm text-gray-600">Provider</span>
            <select
              v-model="form.model_provider"
              class="mt-1 w-full rounded-lg border border-gray-300 bg-white px-3 py-2 text-sm focus:border-blue-500 focus:outline-none"
            >
              <option value="openai">openai（OpenAI 兼容）</option>
              <option value="anthropic">anthropic（Claude 原生）</option>
              <option value="mock">mock（本地回显）</option>
            </select>
          </label>
          <label class="block flex-1">
            <span class="text-sm text-gray-600">Model</span>
            <input
              v-model="form.model_name"
              type="text"
              placeholder="如 deepseek-chat"
              class="mt-1 w-full rounded-lg border border-gray-300 px-3 py-2 text-sm focus:border-blue-500 focus:outline-none"
            />
          </label>
          <label class="block w-28">
            <span class="text-sm text-gray-600">Temperature</span>
            <input
              v-model.number="form.temperature"
              type="number"
              step="0.1"
              min="0"
              max="2"
              class="mt-1 w-full rounded-lg border border-gray-300 px-3 py-2 text-sm focus:border-blue-500 focus:outline-none"
            />
          </label>
        </div>

        <!-- provider=openai/anthropic 时按 Agent 配置真实大模型连接。 -->
        <template v-if="form.model_provider !== 'mock'">
          <label class="block">
            <span class="text-sm text-gray-600">Base URL</span>
            <input
              v-model="form.base_url"
              type="text"
              :placeholder="
                form.model_provider === 'anthropic'
                  ? '如 https://api.anthropic.com'
                  : '如 https://api.deepseek.com'
              "
              class="mt-1 w-full rounded-lg border border-gray-300 px-3 py-2 text-sm focus:border-blue-500 focus:outline-none"
            />
            <span class="mt-1 block text-xs text-gray-400">
              <template v-if="form.model_provider === 'anthropic'">
                Claude 原生地址，会自动追加 /v1/messages。原生填 https://api.anthropic.com，中转填 .../v1。
              </template>
              <template v-else>
                OpenAI 兼容地址，会自动追加 /chat/completions。DeepSeek/通义/Kimi/智谱/本地 vLLM 换这里即可。
              </template>
            </span>
          </label>

          <label class="block">
            <span class="text-sm text-gray-600">API Key</span>
            <input
              v-model="form.api_key"
              type="password"
              autocomplete="off"
              placeholder="sk-..."
              class="mt-1 w-full rounded-lg border border-gray-300 px-3 py-2 text-sm focus:border-blue-500 focus:outline-none"
            />
          </label>

          <!-- Anthropic 强制要求 max_tokens。 -->
          <label v-if="form.model_provider === 'anthropic'" class="block">
            <span class="text-sm text-gray-600">Max Tokens</span>
            <input
              v-model.number="form.max_tokens"
              type="number"
              min="1"
              step="1"
              placeholder="留空用默认 4096"
              class="mt-1 w-full rounded-lg border border-gray-300 px-3 py-2 text-sm focus:border-blue-500 focus:outline-none"
            />
            <span class="mt-1 block text-xs text-gray-400">
              单次回复最大 token，Anthropic 必填项；留空/0 用默认 4096。
            </span>
          </label>
        </template>

        <div class="mt-5 flex justify-end gap-2">
          <button
            type="button"
            class="rounded-lg px-4 py-2 text-sm text-gray-600 hover:bg-gray-100"
            @click="emit('close')"
          >
            取消
          </button>
          <button
            type="submit"
            :disabled="!form.name.trim()"
            class="rounded-lg bg-blue-600 px-4 py-2 text-sm text-white hover:bg-blue-700 disabled:opacity-50"
          >
            保存
          </button>
        </div>
      </form>
    </div>
  </div>
</template>
