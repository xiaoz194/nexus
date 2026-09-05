// 类型与 Go 后端 JSON 对齐（字段名用 snake_case）。

export interface Agent {
  id: string
  name: string
  description: string
  system_prompt: string
  model_provider: string
  model_name: string
  base_url: string
  api_key: string
  max_tokens: number
  temperature: number
  created_at: string
  updated_at: string
}

export interface Conversation {
  id: string
  agent_id: string
  title: string
  created_at: string
  updated_at: string
}

export type Role = 'user' | 'assistant'

export interface Message {
  id: string
  conversation_id: string
  role: Role
  content: string
  created_at: string
}

// 登录用户（后端 publicUser，不含密码哈希）。
export interface User {
  id: string
  username: string
  display_name: string
  created_at: string
}

// 创建/更新 Agent 的请求体。
export interface AgentInput {
  name: string
  description?: string
  system_prompt?: string
  model_provider?: string
  model_name?: string
  base_url?: string
  api_key?: string
  max_tokens?: number
  temperature?: number
}
