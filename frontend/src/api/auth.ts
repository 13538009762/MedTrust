import client from './client'
import type {
  ApiEnvelope,
  User,
  LoginReq,
  LoginResp,
  CaptchaResp,
} from '@/types'

export function loginApi(data: LoginReq): Promise<ApiEnvelope<LoginResp>> {
  return client.post('/auth/login', data)
}

export function registerApi(data: Record<string, any>): Promise<ApiEnvelope<any>> {
  return client.post('/auth/register', data)
}

export function logoutApi(): Promise<ApiEnvelope<any>> {
  return client.post('/auth/logout')
}

export function createSSETicketApi(): Promise<ApiEnvelope<{ ticket: string; expires_in: number }>> {
  return client.post('/auth/ticket')
}

export function getCaptchaApi(): Promise<ApiEnvelope<CaptchaResp>> {
  return client.get('/auth/captcha')
}

export function getMeApi(): Promise<ApiEnvelope<User>> {
  return client.get('/auth/me')
}

export function updatePasswordApi(data: { old_password?: string; new_password: string }): Promise<ApiEnvelope<any>> {
  return client.put('/auth/password', data)
}

export function updateMedicalKeyApi(data: { old_key?: string; new_key?: string; medical_key?: string }): Promise<ApiEnvelope<any>> {
  return client.put('/auth/medical-key', data)
}

export function updateProfileApi(data: Record<string, any>): Promise<ApiEnvelope<any>> {
  return client.put('/auth/profile', data)
}

export function updateAvatarApi(data: { avatar: string } | string): Promise<ApiEnvelope<any>> {
  const payload = typeof data === 'string' ? { avatar: data } : data
  return client.put('/auth/avatar', payload)
}
