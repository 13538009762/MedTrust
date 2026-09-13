import axios from 'axios'
import { ElMessage } from 'element-plus'
import router from '../router'

const api = axios.create({
  baseURL: 'http://127.0.0.1:8080/api/v1',
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
        sessionStorage.removeItem('medtrust_token')
        sessionStorage.removeItem('medtrust_user')
        ElMessage.error(error.response.data?.message || '登录已过期，请重新登录')
        router.push('/login')
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
