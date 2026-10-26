/**
 * 电子病历与附件领域模型定义
 */

export interface MedicalFile {
  id: number
  record_id: number
  file_name: string
  file_size: number
  file_type: string
  ipfs_hash: string
  encrypted_key?: string
  checksum_sha256?: string
  is_compressed?: boolean
  created_at?: string
}

export interface MedicalRecord {
  id: number
  record_no: string
  patient_id: number
  doctor_id: number
  hospital_id: number
  department_id: number
  title: string
  diagnosis: string
  chief_complaint?: string
  data_type: string // OUTPATIENT, INPATIENT, EXAM, SURGERY
  ipfs_cid?: string
  file_hash?: string
  encrypted_symmetric_key?: string
  key_derivation_algorithm?: string
  fabric_tx_id?: string
  integrity_status?: string
  is_tampered?: boolean
  patient_name?: string
  patient_id_card?: string
  patient_phone?: string
  doctor_name?: string
  hospital_name?: string
  department_name?: string
  files?: MedicalFile[]
  created_at?: string
  updated_at?: string
}

export interface MedicalExamOrder {
  id: number
  order_no: string
  record_id: number
  patient_id: number
  hospital_id: number
  order_type: string
  item_name: string
  status: string // PENDING, PROCESSING, COMPLETED
  priority: string
  result_summary?: string
  doctor_name?: string
  patient_name?: string
  hospital_name?: string
  created_at?: string
  updated_at?: string
}

export interface RecordQueryParams {
  patient_id?: number
  hospital_id?: number
  exclude_hospital_id?: number
  only_cross?: boolean
  only_cross_accessed?: boolean
  data_type?: string
  keyword?: string
}

export interface InfectionRiskItem {
  name: string
  level: 'HIGH' | 'MEDIUM' | 'LOW'
  code: string
  detected_term: string
  advice: string
}

export interface InfectionRiskSummary {
  has_risk: boolean
  patient_name: string
  record_no: string
  diseases: InfectionRiskItem[]
  notice: string
}
