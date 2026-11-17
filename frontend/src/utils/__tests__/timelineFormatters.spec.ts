import { describe, it, expect } from 'vitest'
import type { MedicalRecordRow } from '../../types/clinical'
import {
  buildHistoryClueSnippet,
  computeVisitGapLabel,
  filterTimelineRecords,
  formatRelativeVisitTime,
  isRecordAccessible,
  resolveChainVerifyStatus,
  sortRecordsChronologically,
  unwrapTimelineRecordsResponse
} from '../timelineFormatters'

function makeRecord(overrides: Partial<MedicalRecordRow> = {}): MedicalRecordRow {
  return {
    id: 1,
    record_no: 'MR-2026-001',
    created_at: '2026-11-10T09:00:00Z',
    encounter_type: 'OUTPATIENT',
    diagnosis: '原发性高血压病1级',
    chief_complaint: '间断头晕3天',
    etiology: '劳累伴盐摄入偏高',
    treatment_plan: '低盐饮食，氨氯地平5mg qd',
    verified: true,
    is_tampered: false,
    has_access: true,
    ...overrides
  }
}

describe('timelineFormatters unit tests', () => {
  it('isRecordAccessible enforces zero-trust access boundaries', () => {
    expect(isRecordAccessible(makeRecord())).toBe(true)
    expect(isRecordAccessible(makeRecord({ has_access: false }))).toBe(false)
    expect(isRecordAccessible(makeRecord({ diagnosis: '【未取得患者授权】' }))).toBe(false)
    expect(isRecordAccessible(makeRecord({ chief_complaint: '【受控访问限制·未授权掩码】' }))).toBe(false)
  })

  it('resolveChainVerifyStatus prioritizes tampered over verified', () => {
    expect(resolveChainVerifyStatus(makeRecord({ verified: true, is_tampered: true }))).toBe('tampered')
    expect(resolveChainVerifyStatus(makeRecord({ verified: true, is_tampered: false }))).toBe('verified')
    expect(resolveChainVerifyStatus(makeRecord({ verified: false, is_tampered: false }))).toBe('pending')
  })

  it('sortRecordsChronologically sorts descending without mutating input', () => {
    const input = [
      makeRecord({ id: 1, created_at: '2026-09-01T10:00:00Z' }),
      makeRecord({ id: 3, created_at: '2026-11-01T10:00:00Z' }),
      makeRecord({ id: 2, created_at: '2026-10-01T10:00:00Z' })
    ]
    const sorted = sortRecordsChronologically(input)
    expect(sorted.map((r) => r.id)).toEqual([3, 2, 1])
    expect(input.map((r) => r.id)).toEqual([1, 3, 2])
  })

  it('filterTimelineRecords splits accessible vs locked records cleanly', () => {
    const records = [
      makeRecord({ id: 1, has_access: true }),
      makeRecord({ id: 2, has_access: false })
    ]
    expect(filterTimelineRecords(records, 'ALL').map((r) => r.id)).toEqual([1, 2])
    expect(filterTimelineRecords(records, 'ACCESSIBLE').map((r) => r.id)).toEqual([1])
    expect(filterTimelineRecords(records, 'LOCKED').map((r) => r.id)).toEqual([2])
  })

  it('formatRelativeVisitTime & computeVisitGapLabel handle valid and invalid dates safely', () => {
    const now = Date.parse('2026-11-14T12:00:00Z')
    expect(formatRelativeVisitTime('2026-11-14T08:00:00Z', now)).toBe('今日就诊')
    expect(formatRelativeVisitTime('2026-11-13T08:00:00Z', now)).toBe('昨日')
    expect(formatRelativeVisitTime('2026-11-01T08:00:00Z', now)).toBe('13天前')
    expect(formatRelativeVisitTime(undefined, now)).toBe('时间未登记')

    expect(computeVisitGapLabel('2026-11-14T08:00:00Z', '2026-11-02T08:00:00Z')).toBe('间隔 12 天')
    expect(computeVisitGapLabel('2026-11-14T08:00:00Z', '2026-11-14T07:00:00Z')).toBe('同日复诊/连续就诊')
    expect(computeVisitGapLabel(undefined, '2026-11-01T00:00:00Z')).toBe('')
  })

  it('buildHistoryClueSnippet blocks tampered or locked nodes and formats valid nodes', () => {
    expect(buildHistoryClueSnippet(makeRecord({ has_access: false }))).toBe('')
    expect(buildHistoryClueSnippet(makeRecord({ is_tampered: true }))).toBe('')

    const snippet = buildHistoryClueSnippet(makeRecord(), '协和医院')
    expect(snippet).toContain('协和医院')
    expect(snippet).toContain('诊断：原发性高血压病1级')
    expect(snippet).toContain('症状表现：间断头晕3天')
    expect(snippet).toContain('病因/诱因：劳累伴盐摄入偏高')
  })

  it('unwrapTimelineRecordsResponse supports plain arrays and paginated envelopes', () => {
    const r = makeRecord()
    expect(unwrapTimelineRecordsResponse({ code: 200, data: [r] })).toEqual([r])
    expect(unwrapTimelineRecordsResponse({ code: 200, data: { items: [r] } })).toEqual([r])
    expect(unwrapTimelineRecordsResponse({ code: 200, data: { list: [r] } })).toEqual([r])
    expect(unwrapTimelineRecordsResponse({ code: 500, data: [r] })).toEqual([])
    expect(unwrapTimelineRecordsResponse(undefined)).toEqual([])
  })
})
