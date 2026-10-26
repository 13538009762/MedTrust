import { describe, it, expect } from 'vitest'
import {
  formatFileSize,
  formatEncounterType,
  formatTime,
  formatTxId,
  cleanClinicalText,
  getRiskScoreText,
  getRiskRulesTooltip,
  cleanDurationText,
  formatOnsetTime,
  getDisplayEtiology,
  getBarcodeData
} from '../recordFormatters'

describe('recordFormatters unit tests', () => {
  it('formatFileSize should format bytes correctly', () => {
    expect(formatFileSize()).toBe('2.8 KB')
    expect(formatFileSize(500)).toBe('500 B')
    expect(formatFileSize(2048)).toBe('2.0 KB')
    expect(formatFileSize(2097152)).toBe('2.00 MB')
  })

  it('formatEncounterType should map encounter types', () => {
    expect(formatEncounterType('OUTPATIENT')).toBe('普通门诊')
    expect(formatEncounterType('EMERGENCY')).toBe('急诊救治')
    expect(formatEncounterType('INPATIENT')).toBe('住院就诊')
    expect(formatEncounterType()).toBe('门诊')
  })

  it('formatTime should format ISO time string', () => {
    expect(formatTime()).toBe('-')
    expect(formatTime('2026-05-01T14:30:00Z')).toBe('2026-05-01 14:30')
  })

  it('formatTxId should strip 0x prefix', () => {
    expect(formatTxId()).toBe('-')
    expect(formatTxId('0x123abc')).toBe('123abc')
    expect(formatTxId('456def')).toBe('456def')
  })

  it('cleanClinicalText should clean redundant words', () => {
    expect(cleanClinicalText('持续持续发热')).toBe('持续发热')
    expect(cleanClinicalText('持续时间约 持续2天')).toBe('持续约 2天')
  })

  it('getRiskScoreText should map risk scores', () => {
    expect(getRiskScoreText(75)).toBe('高风险需二次确认')
    expect(getRiskScoreText(45)).toBe('中风险需核准')
    expect(getRiskScoreText(10)).toBe('低风险常规调阅')
  })

  it('getRiskRulesTooltip should return detailed or inferred rules', () => {
    expect(getRiskRulesTooltip({ risk_detail: 'custom detail' })).toContain('custom detail')
    expect(getRiskRulesTooltip({ risk_score: 70 })).toContain('高频访问预警')
    expect(getRiskRulesTooltip({ risk_score: 40 })).toContain('跨合作机构调阅')
    expect(getRiskRulesTooltip({ risk_score: 15 })).toContain('同院经治医生随访')
  })

  it('cleanDurationText should clean prefix labels', () => {
    expect(cleanDurationText('持续时间：3天')).toBe('3天')
    expect(cleanDurationText('病程: 1周')).toBe('1周')
    expect(cleanDurationText()).toBe('发作性')
  })

  it('formatOnsetTime should format onset timestamp', () => {
    expect(formatOnsetTime()).toBe('接诊前')
    expect(formatOnsetTime('2026-05-01T10:00:00')).toBe('2026-05-01 10:00:00')
  })

  it('getDisplayEtiology should provide appropriate etiology', () => {
    expect(getDisplayEtiology({ etiology: '明确外伤' })).toBe('明确外伤')
    expect(getDisplayEtiology({ diagnosis: '急性细菌性肺炎' })).toContain('急性细菌性感染')
    expect(getDisplayEtiology({ diagnosis: '冠心病心绞痛' })).toContain('冠状动脉粥样硬化')
    expect(getDisplayEtiology({ diagnosis: '原发性高血压' })).toContain('外周阻力血管收缩')
    expect(getDisplayEtiology({ diagnosis: '支气管哮喘伴咳嗽' })).toContain('气道高反应性')
  })

  it('getBarcodeData should return valid SVG bars and total width', () => {
    const res = getBarcodeData('REC202609001')
    expect(res.bars.length).toBeGreaterThan(0)
    expect(res.totalWidth).toBeGreaterThan(0)
  })
})
