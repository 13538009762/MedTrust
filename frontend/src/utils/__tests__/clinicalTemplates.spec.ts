import { describe, it, expect } from 'vitest'
import { clinicalTemplates, getIcdCode } from '../clinicalTemplates'

describe('clinicalTemplates unit tests', () => {
  it('should have 5 pre-configured clinical templates', () => {
    expect(clinicalTemplates.length).toBe(5)
    expect(clinicalTemplates[0].icd).toBe('I25.1')
    expect(clinicalTemplates[0].dept).toBe('心血管内科门诊')
  })

  it('getIcdCode should return correct ICD-10 code for diagnosis', () => {
    expect(getIcdCode('冠状动脉硬化')).toBe('I25.1')
    expect(getIcdCode('原发性高血压')).toBe('I10')
    expect(getIcdCode('急性上呼吸道感染')).toBe('J06.9')
    expect(getIcdCode('急性胸痛查因')).toBe('R07.4')
    expect(getIcdCode('慢性浅表性胃炎')).toBe('K29.7')
    expect(getIcdCode('2型糖尿病')).toBe('E11.9')
    expect(getIcdCode('左下肢骨折')).toBe('S82.9')
    expect(getIcdCode('未知综合征')).toBe('Z00.0')
    expect(getIcdCode()).toBe('Z00.0')
  })
})
