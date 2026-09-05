import { api } from './client'
import type { Conversation } from '../types'

export const listConversations = (agentId: string) =>
  api.get<Conversation[]>(`/agents/${agentId}/conversations`)

export const createConversation = (agentId: string, title?: string) =>
  api.post<Conversation>(`/agents/${agentId}/conversations`, { title: title ?? '' })

export const deleteConversation = (agentId: string, conversationId: string) =>
  api.del(`/agents/${agentId}/conversations/${conversationId}`)
