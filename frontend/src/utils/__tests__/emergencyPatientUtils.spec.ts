import { describe, it, expect } from 'vitest'
import {
  hasRealAllergy,
  formatProtectionLevel,
  formatInfectionSummary
} from '../emergencyPatientUtils'

describe('emergencyPatientUtils unit tests', () => {
  it('hasRealAllergy should identify true allergy vs safe negative expressions', () => {
    expect(hasRealAllergy()).toBe(false)
    expect(hasRealAllergy('')).toBe(false)
    expect(hasRealAllergy('未发现明确药物过敏史')).toBe(false)
    expect(hasRealAllergy('否认药物过敏史')).toBe(false)
    expect(hasRealAllergy('青霉素过敏')).toBe(true)
    expect(hasRealAllergy({ allergies: '头孢类过敏，皮疹' })).toBe(true)
    expect(hasRealAllergy({ allergies: '暂无' })).toBe(false)
    expect(hasRealAllergy({ has_allergies: true })).toBe(true)
    expect(hasRealAllergy({ has_allergies: false })).toBe(false)
  })

  it('formatProtectionLevel should format various levels to X级防护', () => {
    expect(formatProtectionLevel()).toBe('二级防护')
    expect(formatProtectionLevel({ level: '三级' })).toBe('三级防护')
    expect(formatProtectionLevel({ level: '一级防护' })).toBe('一级防护')
    expect(formatProtectionLevel({ risk_level: 'CRITICAL' })).toBe('三级防护')
  })

  it('formatInfectionSummary should replace aggressive terminology', () => {
    expect(formatInfectionSummary()).toBe('既往/长期携带病史')
    expect(formatInfectionSummary('患者必定携带乙肝病毒')).toBe('患者长期携带乙肝病毒')
    expect(formatInfectionSummary('必定/长期携带病史')).toBe('既往/长期携带病史')
  })
})
