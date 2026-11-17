/**
 * 临床就诊业务领域共享类型定义 (WP-16)
 * 供医生端跨院检索 / 就诊记录 / AI 辅助诊断等视图与子组件共用，
 * 取代各巨型视图中散落的 ref<any[]> 与 : any 注解。
 */

// ==================== 患者与用户 ====================

/** 患者档案下拉/选择候选项 (listPatientsApi 返回行) */
export interface PatientOption {
  id: number
  user_no?: string
  real_name?: string
  username?: string
  gender?: string
  age?: number
  id_card?: string
  infection_alert?: InfectionAlert | null
  [key: string]: any
}

import type { InfectionRiskAlert } from '../utils/infectionGuard'

/** 跨机构传染病职业暴露预警 (getPatientInfectionRisksApi 返回)
 *  与 utils/infectionGuard.InfectionRiskAlert 结构对齐，保证可直接传给 InfectionSafetyAlert */
export type InfectionAlert = InfectionRiskAlert

// ==================== 就诊病历记录 ====================

/** 病历/就诊记录行 (listRecordsApi / getRecordDetailApi 返回) */
export interface MedicalRecordRow {
  id: number
  record_no: string
  patient_id?: number
  patient_name?: string
  patient_id_card?: string
  patient_phone?: string
  patient_gender?: string
  patient_age?: number
  hospital_id?: number
  hospital_name?: string
  department_name?: string
  doctor_id?: number
  doctor_name?: string
  encounter_type?: string
  status?: string
  data_type?: string
  access_type?: string
  has_access?: boolean
  is_tampered?: boolean
  tamper_reason?: string
  current_hash?: string
  chain_hash?: string
  need_exam?: boolean
  exam_items?: string
  exam_result?: string
  exam_doctor?: string
  exam_time?: string
  chief_complaint?: string
  symptoms?: string
  present_illness?: string
  onset_time?: string
  onsetTime?: string
  duration?: string
  initial_diagnosis?: string
  diagnostic_basis?: string
  diagnosis?: string
  etiology?: string
  treatment_plan?: string
  soap_content?: string
  vital_signs?: string
  created_at?: string
  fabric_tx_id?: string
  block_height?: number
  encrypted_dek?: string
  iv_nonce?: string
  ipfs_cid?: string
  file_hash?: string
  files?: MedicalFileItem[]
  infection_alert?: InfectionAlert | null
  [key: string]: any
}

/** 病历归档附件 (IPFS 存证文件) */
export interface MedicalFileItem {
  id: number
  file_name?: string
  file_type?: string
  file_size?: number
  file_hash?: string
  ipfs_cid?: string
  [key: string]: any
}

// ==================== 表格自定义列 ====================

/** 跨院检索表格自定义显示列配置项 (localStorage 持久化) */
export interface TableColumnItem {
  key: string
  label: string
  visible: boolean
  minWidth?: string
  width?: string
  fixed?: string | boolean
}

// ==================== 风控评估与调阅网关 ====================

/** 风控规则命中因子 */
export interface RiskFactor {
  rule_id: string
  factor_name: string
  weight: number
  score: number
  triggered: boolean
  description: string
}

/** 动态访问风险评估报告 (RULE_ENGINE_WEIGHTED) */
export interface RiskEvaluation {
  strategy?: string
  level?: 'LOW' | 'MEDIUM' | 'HIGH' | string
  total_score?: number
  factors?: RiskFactor[]
  [key: string]: any
}

/** 风控终审三态 */
export type RiskDecisionState = 'ALLOWED' | 'PENDING_CONFIRM' | 'REJECTED'

/** 调阅通道：知情同意/现场密钥 或 危重急救破窗 */
export type AccessChannel = 'CONSENT' | 'BREAK_GLASS'

/** 知情同意申请表单 */
export interface ConsentFormState {
  purpose: string
  scope_type: 'SINGLE' | 'ALL'
  days: number
}

/** Break-Glass 紧急破窗申请表单 */
export interface BreakGlassFormState {
  emergency_reason: string
  description: string
  doctor_confirmed: boolean
}

// ==================== 就诊录入工作流 ====================

/** 临床决策路径：直接确诊 / 开立医技检查 */
export type EncounterMode = 'DIRECT' | 'EXAM'

/** 接诊初诊基本信息表单 */
export interface EncounterInitForm {
  patient_id: number | undefined
  encounter_type: string
  department_name: string
  onset_time: string
  duration: string
  chief_complaint: string
  present_illness: string
  initial_diagnosis: string
  diagnostic_basis: string
  exam_reason: string
}

/** 生命体征测量 */
export interface VitalSigns {
  temperature: string
  blood_pressure: string
  heart_rate: string
  spo2: string
}

/** 确诊与处置表单 (直接确诊 / 最终确诊共用结构) */
export interface DiagnosisFormState {
  diagnosis: string
  etiology: string
  treatment_plan: string
  soap_content: string
}

/** sessionStorage 接诊草稿结构 (medtrust_encounter_draft) */
export interface EncounterDraft {
  selectedPatientInfo: PatientOption | null
  selectedPatientInfectionAlert: InfectionAlert | null
  initForm: Partial<EncounterInitForm>
  vitals: Partial<VitalSigns>
  encounterMode: EncounterMode
  directForm: Partial<DiagnosisFormState>
  selectedExamList: string[]
  timestamp: number
}

// ==================== 处方与检查单 ====================

/** 规范处方一键录入模板 */
export interface PrescriptionPreset {
  key: string
  label: string
  type: 'primary' | 'success' | 'warning' | 'danger' | 'info'
  diagnosis: string
  etiology: string
  plan: string
}

/** 处置方案结构化条目 (parsePlanItems 输出) */
export interface PlanItem {
  id: number
  text: string
  isMed: boolean
}

/** 解析后的生命体征 (parseVitals 输出) */
export interface ParsedVitals {
  temperature: string
  blood_pressure: string
  heart_rate: string
  spo2: string
  raw: string
}

// ==================== 跨院检索筛选状态 ====================

/** 跨院检索多维细筛组合状态 */
export interface CrossQueryFilterState {
  quickClassifyMode: 'ALL' | 'ATTENTION' | 'ACCESSIBLE' | 'NEED_AUTH' | 'EXAM_ONLY'
  filterAuthStatus: 'ALL' | 'SELF' | 'HOSPITAL' | 'AUTHORIZED' | 'PENDING' | 'BREAK_GLASS'
  filterClinicalStatus: 'ALL' | 'ABNORMAL' | 'NORMAL'
  filterTamperStatus: 'ALL' | 'SAFE' | 'TAMPERED'
  filterTimeRange: 'ALL' | '3D' | '7D' | '30D' | '90D'
}

// ==================== 链上完整性校验结果 ====================

/** 链上存证哈希比对行 */
export interface ChainHashEntry {
  record_id: number
  current_hash: string
  chain_hash: string
  is_consistent: boolean
}

/** IntegrityVerifyResult 结构 (verifyRecordApi 返回 data) */
export interface IntegrityVerifyResult {
  current_hash: string
  chain_hash: string
  is_consistent: boolean
  verify_time: string
  ipfs_cid?: string
  ipfs_hash?: string
  fabric_tx_id?: string
  block_height?: number
  details?: ChainHashEntry[]
  [key: string]: unknown
}

// ==================== AI 辅助诊断弹窗输入目标患者结构 ====================

/** AI 辅助诊断弹窗输入目标患者结构 */
export interface CrossTargetPatient {
  id: number
  real_name: string
  user_no?: string
  gender?: string
  age?: number
  [key: string]: unknown
}