import axios from 'axios'
import { ElMessage } from 'element-plus'
import router from '../router'
import { useAuthStore } from '../stores/auth'

const api = axios.create({
  baseURL: import.meta.env.VITE_API_BASE_URL || '/api/v1',
  timeout: 10000,
})

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
    if (error.response) {
      if (error.response.status === 401) {
        // 同步清空 Pinia 内存态与 sessionStorage，避免 router.beforeEach 读取残留 auth.token 陷入登录重定向死循环
        try {
          useAuthStore().clearAuth()
        } catch {
          sessionStorage.removeItem('medtrust_token')
          sessionStorage.removeItem('medtrust_user')
        }
        ElMessage.error(error.response.data?.message || '登录已过期，请重新登录')
        if (router.currentRoute.value.name !== 'login') {
          router.replace('/login')
        }
      } else if (error.response.status === 403) {
        // 留给业务层或直接提示
        return Promise.reject(error.response.data)
      } else {
        ElMessage.error(error.response.data?.message || '请求服务失败')
      }
      return Promise.reject(error.response.data)
    }
    ElMessage.error('网络连接异常或后端服务未启动')
    return Promise.reject(error)
  }
)

export default api
