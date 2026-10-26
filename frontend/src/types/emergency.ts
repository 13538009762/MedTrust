/**
 * 紧急救治 (Break-Glass) 与事后监管模型定义
 */

export interface EmergencyAccessEvent {
  id: number
  event_no: string
  doctor_id: number
  patient_id: number
  record_id: number
  source_hospital_id: number
  target_hospital_id: number
  emergency_reason: string
  description: string
  doctor_confirmed: boolean
  audit_status: 'PENDING_AUDIT' | 'APPROVED' | 'REJECTED' | 'CLOSED_NORMAL' | 'CLOSED_VIOLATION'
  supervisor_id?: number
  audit_comment?: string
  patient_feedback?: 'PENDING' | 'ACKNOWLEDGED' | 'OBJECTED'
  patient_comment?: string
  fabric_tx_id?: string
  doctor_name?: string
  patient_name?: string
  record_no?: string
  source_hospital_name?: string
  target_hospital_name?: string
  rule_pre_screen?: string
  rule_pre_screen_desc?: string
  audit_comparison_summary?: string
  created_at?: string
  updated_at?: string
}

export interface EmergencyUnlockReq {
  doctor_id?: number
  patient_id: number
  record_ids?: number[]
  emergency_reason: string
  description: string
  doctor_confirmed: boolean
}

export interface EmergencySingleUnlockReq {
  record_id: number
  emergency_reason: string
  description: string
  doctor_confirmed: boolean
}

export interface PatientEmergencyFeedbackReq {
  patient_feedback: 'ACKNOWLEDGED' | 'OBJECTED'
  patient_comment?: string
}

export interface AuditEmergencyReq {
  audit_status?: string
  decision?: string
  audit_comment?: string
  comment?: string
  punishment?: 'NORMAL' | 'RESTRICTED' | 'DISABLED' | string
}

export interface LiftDoctorRestrictionReq {
  doctor_id: number
  comment?: string
}
