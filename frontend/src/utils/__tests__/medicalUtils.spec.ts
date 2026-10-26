import { describe, it, expect } from 'vitest'
import {
  escapeHtml,
  highlightText,
  extractAlertTags,
  parseMedicalExamResult
} from '../medicalReportParser'
import { scanTextForInfections } from '../infectionGuard'

describe('medicalReportParser unit tests', () => {
  it('escapeHtml should escape HTML entities properly', () => {
    expect(escapeHtml('<script>alert("xss")</script>')).toBe(
      '&lt;script&gt;alert(&quot;xss&quot;)&lt;/script&gt;'
    )
    expect(escapeHtml("Tom & Jerry 'cat'")).toBe(
      'Tom &amp; Jerry &#039;cat&#039;'
    )
  })

  it('highlightText should wrap keyword in mark tag safely', () => {
    const raw = '患者出现心肌缺血症状'
    const highlighted = highlightText(raw, '心肌缺血')
    expect(highlighted).toContain('<mark class="med-kw-mark">心肌缺血</mark>')
  })

  it('extractAlertTags should identify abnormal danger tags', () => {
    const text = '患者急性心肌梗死合并室性早搏，建议立即急诊救治'
    const result = extractAlertTags(text)
    expect(result.isAbnormal).toBe(true)
    expect(result.tags.length).toBeGreaterThan(0)
    const tagTexts = result.tags.map(t => t.text)
    expect(tagTexts.some(t => t.includes('心梗'))).toBe(true)
    expect(result.tags[0].level).toBe(3)
  })

  it('parseMedicalExamResult should parse structured report', () => {
    const raw = `
【12导联心电图】
检查所见: 窦性心律，HR 78bpm，PR间期0.16s
诊断结论: ST段轻度下移0.05mV，T波倒置
建议: 结合临床随访
    `
    const parsed = parseMedicalExamResult(raw, 'ST段')
    expect(parsed.hasItems).toBe(true)
    expect(parsed.sections.length).toBeGreaterThan(0)
    expect(parsed.sections[0].conclusion).toContain('ST段轻度下移')
    expect(parsed.sections[0].hasKeywordMatch).toBe(true)
  })
})

describe('infectionGuard unit tests', () => {
  it('scanTextForInfections should detect high-risk infectious diseases', () => {
    const report = '患者HIV抗体确证试验阳性，合并持续低热与口腔真菌感染'
    const alert = scanTextForInfections(report)
    expect(alert.has_risk).toBe(true)
    expect(alert.diseases.length).toBeGreaterThan(0)
    expect(alert.diseases[0].name).toContain('艾滋病')
  })

  it('scanTextForInfections should return no risk for normal report', () => {
    const report = '常规体检，心肺未见明显异常，腹软无压痛'
    const alert = scanTextForInfections(report)
    expect(alert.has_risk).toBe(false)
    expect(alert.diseases.length).toBe(0)
  })
})
