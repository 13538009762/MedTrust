import { describe, it, expect } from 'vitest'
import {
  evaluateTemperature,
  evaluateBloodPressure,
  evaluateHeartRate,
  evaluateSpo2,
  evaluateVitalSummaryTags,
} from '../vitalSignsEvaluator'

describe('vitalSignsEvaluator clinical safety tests', () => {
  it('flags hypertensive crisis and shock as critical instead of normal', () => {
    expect(evaluateBloodPressure('185/115').level).toBe('critical')
    expect(evaluateBloodPressure('80/50').level).toBe('critical')
    expect(evaluateBloodPressure('145/92').level).toBe('elevated')
    expect(evaluateBloodPressure('120/78').level).toBe('normal')
    expect(evaluateBloodPressure('').level).toBe('unknown')
  })

  it('evaluates temperature, heart rate, and SpO2 thresholds accurately', () => {
    expect(evaluateTemperature('39.5').level).toBe('critical')
    expect(evaluateTemperature('37.8').level).toBe('elevated')
    expect(evaluateTemperature('36.5').level).toBe('normal')
    expect(evaluateTemperature('').level).toBe('unknown')

    expect(evaluateHeartRate('138').level).toBe('critical')
    expect(evaluateHeartRate('52').level).toBe('elevated')
    expect(evaluateHeartRate('76').level).toBe('normal')

    expect(evaluateSpo2('86').level).toBe('critical')
    expect(evaluateSpo2('92').level).toBe('elevated')
    expect(evaluateSpo2('98').level).toBe('normal')
  })

  it('returns empty tags for missing vitals and dynamic tags for parsed emergency vitals', () => {
    expect(evaluateVitalSummaryTags('')).toEqual([])
    const tags = evaluateVitalSummaryTags('BP: 185/115 mmHg, HR: 132 bpm, SpO2: 88%, T: 39.2℃')
    expect(tags.every((t) => t.level === 'critical')).toBe(true)
    expect(tags.length).toBe(4)
  })
})
