/**
 * 访问控制、授权凭证与调阅申请模型定义
 */

export interface Authorization {
  id: number
  auth_no: string
  patient_id: number
  auth_target_type: 'ALL_DOCTORS' | 'HOSPITAL' | 'DOCTOR'
  auth_target_id: number
  scope_type: 'ALL' | 'SINGLE'
  record_id: number
  start_time: string
  end_time: string
  status: 'ACTIVE' | 'REVOKING' | 'REVOKE_PENDING' | 'REVOKED' | 'REVOKE_FAILED' | 'NEEDS_RECONCILIATION'
  fabric_tx_id?: string
  revoke_tx_id?: string
  revoke_error?: string
  operator_id?: number
  target_name?: string
  created_at?: string
  updated_at?: string
}

export interface AccessRequest {
  id: number
  request_no: string
  doctor_id: number
  record_id: number
  patient_id: number
  scope_type: 'ALL' | 'SINGLE'
  purpose: string
  status: 'PENDING' | 'APPROVED' | 'REJECTED'
  decision?: string
  doctor_name?: string
  patient_name?: string
  hospital_name?: string
  record_no?: string
  created_at?: string
  updated_at?: string
}

export interface SetRecordAccessPolicyReq {
  policy_type: 'ALL_DOCTORS' | 'HOSPITAL' | 'DOCTOR' | 'PRIVATE' | string
  auth_target_id?: number
  days?: number
}

export interface SetAllAccessPolicyReq {
  policy_type: 'ALL_DOCTORS' | 'HOSPITAL' | 'DOCTOR' | 'PRIVATE' | string
  auth_target_id?: number
  days?: number
}

export interface CreateAccessRequestReq {
  doctor_id?: number
  record_id?: number
  patient_id?: number
  scope_type?: 'ALL' | 'SINGLE'
  purpose?: string
  days?: number
}

export interface BatchAccessRequestReq {
  doctor_id?: number
  patient_id: number
  record_ids: number[]
  purpose?: string
  days?: number
}

export interface KeyUnlockVerifyReq {
  patient_id: number
  medical_key: string
  captcha_token?: string
  captcha_answer?: string
  purpose?: string
  days?: number
}

export interface KeyUnlockVerifyRecordReq {
  record_id: number
  medical_key: string
  captcha_token?: string
  captcha_answer?: string
  purpose?: string
  days?: number
}

export interface RevokeAuthorizationReq {
  reason?: string
}
