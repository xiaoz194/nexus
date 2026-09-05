<script setup lang="ts">
import { onMounted, onUnmounted, watch } from 'vue'
import { useAuthStore } from './stores/auth'
import { useAgentsStore } from './stores/agents'
import { useConversationsStore } from './stores/conversations'
import { useChatStore } from './stores/chat'
import AgentList from './components/AgentList.vue'
import ConversationList from './components/ConversationList.vue'
import ChatPanel from './components/ChatPanel.vue'
import LoginView from './components/LoginView.vue'
import Toast from './components/Toast.vue'

const auth = useAuthStore()
const agents = useAgentsStore()
const conversations = useConversationsStore()
const chat = useChatStore()

// 任意请求返回 401 时就地登出（跳回登录页）。
function onUnauthorized() {
  auth.clear()
}

onMounted(async () => {
  window.addEventListener('nexus:unauthorized', onUnauthorized)
  await auth.fetchMe()
})

onUnmounted(() => {
  window.removeEventListener('nexus:unauthorized', onUnauthorized)
})

// 登录后加载 Agent 列表；登出后清空所有业务状态（隔离）。
watch(
  () => auth.user,
  (user) => {
    if (user) {
      agents.load()
    } else {
      chat.reset()
      conversations.reset()
      agents.select(null)
      agents.agents = []
    }
  },
)

// 切换 Agent 时重新加载其会话并清空聊天（隔离）。
watch(
  () => agents.selectedId,
  (id) => {
    chat.reset()
    if (id) {
      conversations.loadFor(id)
    } else {
      conversations.reset()
    }
  },
)

// 选中会话时加载其消息历史。
watch(
  () => conversations.selectedId,
  (convId) => {
    const agentId = agents.selectedId
    if (agentId && convId) {
      chat.loadFor(agentId, convId)
    } else {
      chat.reset()
    }
  },
)
</script>

<template>
  <!-- 首次会话探测未完成前显示占位，避免登录页/主界面闪烁。 -->
  <div v-if="!auth.ready" class="flex h-screen w-screen items-center justify-center bg-gray-100">
    <p class="text-sm text-gray-400">加载中…</p>
  </div>

  <LoginView v-else-if="!auth.user" />

  <div v-else class="flex h-screen w-screen overflow-hidden">
    <AgentList />
    <ConversationList />
    <ChatPanel />
  </div>

  <!-- Toast 常驻，登录页也能弹错。 -->
  <Toast />
</template>
