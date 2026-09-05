import { api } from './client'
import type { Agent, AgentInput } from '../types'

export const listAgents = () => api.get<Agent[]>('/agents')
export const getAgent = (id: string) => api.get<Agent>(`/agents/${id}`)
export const createAgent = (input: AgentInput) => api.post<Agent>('/agents', input)
export const updateAgent = (id: string, input: AgentInput) => api.put<Agent>(`/agents/${id}`, input)
export const deleteAgent = (id: string) => api.del(`/agents/${id}`)
