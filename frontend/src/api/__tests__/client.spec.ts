import { describe, it, expect, vi, beforeEach } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'
import axios from 'axios'

const mockRouter = vi.hoisted(() => ({
  currentRoute: { value: { name: 'doctor-records' } },
  replace: vi.fn(),
  push: vi.fn(),
}))

vi.mock('@/router', () => ({ default: mockRouter }))
vi.mock('../router', () => ({ default: mockRouter }))
vi.mock('element-plus', () => ({
  ElMessage: { error: vi.fn(), success: vi.fn(), warning: vi.fn(), info: vi.fn() },
}))

import api from '../client'
import { useAuthStore } from '@/stores/auth'

describe('API client 401 handler', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    const mem: Record<string, string> = {}
    vi.stubGlobal('sessionStorage', {
      getItem: (k: string) => mem[k] ?? null,
      setItem: (k: string, v: string) => {
        mem[k] = v
      },
      removeItem: (k: string) => {
        delete mem[k]
      },
      clear: () => {
        for (const k of Object.keys(mem)) delete mem[k]
      },
    })
    setActivePinia(createPinia())
  })

  it('clears Pinia auth.token and sessionStorage on 401 to avoid /login redirect loop', async () => {
    const auth = useAuthStore()
    auth.setAuth('expired-jwt-token', {
      id: 1,
      user_no: 'DOC01',
      username: 'doc_li',
      real_name: '李医生',
      role: 'doctor',
      hospital_id: 1,
      department_id: 1,
      status: 'NORMAL',
    })
    expect(auth.token).toBe('expired-jwt-token')
    expect(sessionStorage.getItem('medtrust_token')).toBe('expired-jwt-token')

    api.defaults.adapter = async () => {
      throw new axios.AxiosError('Unauthorized', 'ERR_BAD_REQUEST', undefined, undefined, {
        status: 401,
        statusText: 'Unauthorized',
        headers: {},
        config: {} as any,
        data: { code: 401, message: '登录已过期' },
      })
    }

    await expect(api.get('/medical-records')).rejects.toMatchObject({ code: 401 })

    expect(auth.token).toBe('')
    expect(auth.user).toBeNull()
    expect(sessionStorage.getItem('medtrust_token')).toBeNull()
    expect(mockRouter.replace).toHaveBeenCalledWith('/login')
  })
})
