import type { MedicalRecordRow } from '@/types/clinical'
import {
  cleanClinicalText,
  cleanDurationText,
  formatEncounterType,
  formatOnsetTime,
  formatTime
} from './recordFormatters'

export type TimelineFilterMode = 'ALL' | 'ACCESSIBLE' | 'LOCKED'
export type ChainStatusKind = 'verified' | 'tampered' | 'pending'

export interface TimelineRecordApiResponse {
  code?: number
  data?: MedicalRecordRow[] | { items?: MedicalRecordRow[]; list?: MedicalRecordRow[] }
}

/**
 * 从后端 /medical-records 响应中安全解包病历列表（兼容直出数组与分页对象）
 */
export function unwrapTimelineRecordsResponse(
  res: TimelineRecordApiResponse | null | undefined
): MedicalRecordRow[] {
  if (!res || res.code !== 200) return []
  const raw = res.data
  const list = Array.isArray(raw) ? raw : (raw?.items ?? raw?.list ?? [])
  return Array.isArray(list) ? list : []
}

/**
 * 零信任访问权限判断：判断当前医生是否持有该历史病历节点的明文调阅权
 */
export function isRecordAccessible(rec: {
  has_access?: boolean
  diagnosis?: string
  chief_complaint?: string
}): boolean {
  if (rec.has_access === false) return false
  if (rec.diagnosis?.includes('未取得患者授权')) return false
  if (rec.chief_complaint?.includes('受控访问限制')) return false
  return true
}

/**
 * 统一映射区块链完整性校验状态（无 Emoji）
 */
export function resolveChainVerifyStatus(rec: {
  is_tampered?: boolean
  verified?: boolean
}): ChainStatusKind {
  if (rec.is_tampered) return 'tampered'
  if (rec.verified) return 'verified'
  return 'pending'
}

/**
 * 安全解析病历时间戳（毫秒）
 */
export function parseRecordTimestamp(rec: { created_at?: string; id?: number }): number {
  if (rec.created_at) {
    const ts = Date.parse(rec.created_at)
    if (!Number.isNaN(ts)) return ts
  }
  return Number(rec.id || 0)
}

/**
 * 按就诊时间倒序排列时间线（纯函数，不修改原数组）
 */
export function sortRecordsChronologically(records: readonly MedicalRecordRow[]): MedicalRecordRow[] {
  return [...records].sort((a, b) => {
    const diff = parseRecordTimestamp(b) - parseRecordTimestamp(a)
    if (diff !== 0) return diff
    return Number(b.id || 0) - Number(a.id || 0)
  })
}

/**
 * 按授权可见性过滤时间线节点
 */
export function filterTimelineRecords(
  records: readonly MedicalRecordRow[],
  mode: TimelineFilterMode
): MedicalRecordRow[] {
  if (mode === 'ACCESSIBLE') return records.filter((r) => isRecordAccessible(r))
  if (mode === 'LOCKED') return records.filter((r) => !isRecordAccessible(r))
  return [...records]
}

/**
 * 计算就诊日期距今相对时间文案
 */
export function formatRelativeVisitTime(dateStr?: string, nowMs: number = Date.now()): string {
  if (!dateStr) return '时间未登记'
  const ts = Date.parse(dateStr)
  if (Number.isNaN(ts)) return '时间格式异常'
  const diffDays = Math.floor((nowMs - ts) / (1000 * 60 * 60 * 24))
  if (diffDays <= 0) return '今日就诊'
  if (diffDays === 1) return '昨日'
  if (diffDays < 30) return `${diffDays}天前`
  if (diffDays < 365) return `约${Math.floor(diffDays / 30)}个月前`
  return `约${Math.floor(diffDays / 365)}年前`
}

/**
 * 计算相邻两次就诊之间的时间间隔描述（供纵向时间轴脊柱显示）
 */
export function computeVisitGapLabel(newerDateStr?: string, olderDateStr?: string): string {
  if (!newerDateStr || !olderDateStr) return ''
  const newerTs = Date.parse(newerDateStr)
  const olderTs = Date.parse(olderDateStr)
  if (Number.isNaN(newerTs) || Number.isNaN(olderTs)) return ''
  const diffDays = Math.max(0, Math.round((newerTs - olderTs) / (1000 * 60 * 60 * 24)))
  if (diffDays === 0) return '同日复诊/连续就诊'
  if (diffDays < 30) return `间隔 ${diffDays} 天`
  if (diffDays < 365) return `间隔约 ${Math.round(diffDays / 30)} 个月`
  return `间隔约 ${(diffDays / 365).toFixed(1)} 年`
}

/**
 * 提取患者历史病历关键临床摘要，用于接诊医生一键引用到当前现病史
 * 安全防线：严禁引用未授权或已被篡改的病历节点
 */
export function buildHistoryClueSnippet(
  rec: MedicalRecordRow,
  hospitalName?: string
): string {
  if (!isRecordAccessible(rec) || rec.is_tampered) return ''

  const dateText = formatTime(rec.created_at)
  const orgText = hospitalName || rec.hospital_name || '既往就诊机构'
  const encounterText = formatEncounterType(rec.encounter_type)
  const diagnosisText = cleanClinicalText(rec.diagnosis || rec.initial_diagnosis) || '待定'
  const phenomenonText = cleanClinicalText(rec.chief_complaint || rec.symptoms)
  const etiologyText = cleanClinicalText(rec.etiology || rec.diagnostic_basis)
  const treatmentText = cleanClinicalText(rec.treatment_plan)

  const parts: string[] = [
    `[${dateText} · ${orgText} · ${encounterText}]`,
    `诊断：${diagnosisText}`
  ]
  if (phenomenonText) parts.push(`症状表现：${phenomenonText}`)
  if (etiologyText) parts.push(`病因/诱因：${etiologyText}`)
  if (treatmentText) parts.push(`既往处置：${treatmentText}`)
  return parts.join('；')
}

export {
  cleanClinicalText,
  cleanDurationText,
  formatEncounterType,
  formatOnsetTime,
  formatTime
}
