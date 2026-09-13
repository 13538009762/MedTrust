import { defineStore } from 'pinia'
import { ref } from 'vue'

export interface User {
  id: number
  user_no: string
  username: string
  real_name: string
  role: 'doctor' | 'patient' | 'admin' | 'supervisor'
  hospital_id: number
  department_id: number
  title: string
  phone: string
  id_card?: string
  medical_key?: string
  status: 'NORMAL' | 'RESTRICTED' | 'DISABLED'
  hospital_name?: string
  department_name?: string
}

// 清理历史遗留的全局 localStorage 登录信息，杜绝跨标签页账号冲突
try {
  localStorage.removeItem('medtrust_token')
  localStorage.removeItem('medtrust_user')
} catch {}

function getStoredToken(): string {
  return sessionStorage.getItem('medtrust_token') || ''
}

function getStoredUser(): User | null {
  const raw = sessionStorage.getItem('medtrust_user')
  if (!raw) return null
  try {
    return JSON.parse(raw)
  } catch {
    return null
  }
}

export const useAuthStore = defineStore('auth', () => {
  const token = ref<string>(getStoredToken())
  const user = ref<User | null>(getStoredUser())

  function setToken(newToken: string) {
    token.value = newToken
    sessionStorage.setItem('medtrust_token', newToken)
  }

  function setUser(newUser: User | null) {
    user.value = newUser
    if (newUser) {
      sessionStorage.setItem('medtrust_user', JSON.stringify(newUser))
    } else {
      sessionStorage.removeItem('medtrust_user')
      sessionStorage.removeItem('medtrust_token')
    }
  }

  function setAuth(newToken: string, newUser: User) {
    setToken(newToken)
    setUser(newUser)
  }

  function logout() {
    token.value = ''
    user.value = null
    sessionStorage.removeItem('medtrust_token')
    sessionStorage.removeItem('medtrust_user')
  }

  return {
    token,
    user,
    setAuth,
    setToken,
    setUser,
    logout,
  }
})

