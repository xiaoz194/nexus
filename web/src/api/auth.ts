import { api } from './client'
import type { User } from '../types'

export const register = (username: string, password: string, displayName?: string) =>
  api.post<User>('/auth/register', { username, password, display_name: displayName })

export const login = (username: string, password: string) =>
  api.post<User>('/auth/login', { username, password })

export const logout = () => api.post<void>('/auth/logout')

export const me = () => api.get<User>('/auth/me')
