import axios from 'axios'
import { ElMessage } from 'element-plus'
import router from '../router'
import { useAuthStore } from '../stores/auth'

declare module 'axios' {
  export interface AxiosRequestConfig {
    /** 静默模式：为 true 时不触发全局 ElMessage 错误弹窗（如后台轮询或 SSE 重连） */
    silent?: boolean
  }
}

const api = axios.create({
  baseURL: import.meta.env.VITE_API_BASE_URL || '/api/v1',
  timeout: 15000,
})

let isHandling401 = false
let reset401Timer: ReturnType<typeof setTimeout> | null = null

function markToastShown<T>(payload: T): T {
  if (payload && typeof payload === 'object') {
    try {
      Object.defineProperty(payload, '__toastShown', {
        value: true,
        enumerable: false,
        configurable: true,
      })
    } catch {
      ;(payload as Record<string, unknown>).__toastShown = true
    }
  }
  return payload
}

api.interceptors.request.use((config) => {
  const token = sessionStorage.getItem('medtrust_token')
  if (token) {
    config.headers.Authorization = `Bearer ${token}`
  }
  return config
})

api.interceptors.response.use(
  (response) => {
    return response.data
  },
  (error) => {
    const isSilent = Boolean(error?.config?.silent)

    if (error.response) {
      const data = error.response.data
      if (error.response.status === 401) {
        // 同步清空 Pinia 内存态与 sessionStorage，避免 router.beforeEach 读取残留 auth.token 陷入登录重定向死循环
        try {
          useAuthStore().clearAuth()
        } catch {
          sessionStorage.removeItem('medtrust_token')
          sessionStorage.removeItem('medtrust_user')
        }
        if (!isHandling401) {
          isHandling401 = true
          if (!isSilent) {
            ElMessage.error(data?.message || '登录已过期，请重新登录')
          }
          if (router.currentRoute.value.name !== 'login') {
            const currentFullPath = router.currentRoute.value.fullPath
            if (currentFullPath && currentFullPath !== '/' && !currentFullPath.startsWith('/login')) {
              router.replace({ path: '/login', query: { redirect: currentFullPath } })
            } else {
              router.replace('/login')
            }
          }
          if (reset401Timer) clearTimeout(reset401Timer)
          reset401Timer = setTimeout(() => {
            isHandling401 = false
            reset401Timer = null
          }, 1500)
        }
        return Promise.reject(markToastShown(data))
      } else if (error.response.status === 403) {
        // 403 留给业务层做定制引导或权限提示
        return Promise.reject(data)
      } else {
        if (!isSilent) {
          ElMessage.error(data?.message || '请求服务失败，请稍后重试')
          return Promise.reject(markToastShown(data))
        }
        return Promise.reject(data)
      }
    }

    // 主动取消的请求不弹错误提示
    if (axios.isCancel(error) || error?.code === 'ERR_CANCELED') {
      return Promise.reject(error)
    }

    if (!isSilent) {
      const isTimeout = error?.code === 'ECONNABORTED' || String(error?.message || '').toLowerCase().includes('timeout')
      ElMessage.error(isTimeout ? '请求响应较慢，请稍后重试' : '网络连接异常，请检查网络后重试')
      return Promise.reject(markToastShown(error))
    }
    return Promise.reject(error)
  }
)

export default api
