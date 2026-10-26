/**
 * 统一通用响应与基础数据类型
 */

export interface ApiEnvelope<T = any> {
  code: number
  message: string
  data: T
}

export interface PaginationParams {
  page?: number
  page_size?: number
}

export interface PaginationResult<T> {
  total: number
  page: number
  page_size: number
  list: T[]
}
