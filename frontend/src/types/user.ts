/**
 * 用户与鉴权领域模型定义
 */

export type UserRole = 'doctor' | 'patient' | 'admin' | 'supervisor'
export type UserStatus = 'NORMAL' | 'RESTRICTED' | 'DISABLED'

export interface User {
  id: number
  user_no: string
  username: string
  real_name: string
  role: UserRole
  hospital_id: number
  department_id: number
  title?: string
  phone?: string
  id_card?: string
  medical_key?: string
  medical_key_hash?: string
  must_change_pwd?: boolean
  status: UserStatus
  hospital_name?: string
  department_name?: string
  has_medical_key?: boolean
  avatar?: string
  created_at?: string
  updated_at?: string
}

export interface LoginReq {
  username: string
  password: string
  captcha_id?: string
  captcha_token?: string
  captcha_answer?: string
}

export interface LoginResp {
  token: string
  user: User
  must_change_pwd?: boolean
}

export interface CaptchaResp {
  captcha_id?: string
  captcha_image?: string
  captcha_img?: string
  captcha_token?: string
}

export interface ChangePasswordReq {
  old_password: string
  new_password: string
}

export interface SetMedicalKeyReq {
  medical_key: string
}

export interface CreateUserReq {
  username: string
  password?: string
  user_no?: string
  real_name: string
  role: UserRole
  hospital_id?: number
  department_id?: number
  title?: string
  phone?: string
  id_card?: string
  status?: UserStatus
}

export interface UpdateUserStatusReq {
  status: UserStatus
}

export interface ResetPasswordReq {
  new_password?: string
}
