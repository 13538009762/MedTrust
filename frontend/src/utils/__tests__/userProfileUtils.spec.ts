import { describe, it, expect } from 'vitest'
import { getRoleTagType, getRoleName, presetAvatars } from '../userProfileUtils'

describe('userProfileUtils unit tests', () => {
  it('getRoleTagType maps role to appropriate tag styling', () => {
    expect(getRoleTagType('doctor')).toBe('primary')
    expect(getRoleTagType('patient')).toBe('success')
    expect(getRoleTagType('supervisor')).toBe('warning')
    expect(getRoleTagType('admin')).toBe('danger')
    expect(getRoleTagType()).toBe('info')
  })

  it('getRoleName maps role to correct localized display name', () => {
    expect(getRoleName('doctor')).toBe('执业医生')
    expect(getRoleName('patient')).toBe('居民患者')
    expect(getRoleName('supervisor')).toBe('卫健监管专员')
    expect(getRoleName('admin')).toBe('系统管理员')
    expect(getRoleName('guest')).toBe('系统用户')
  })

  it('presetAvatars contains 8 valid SVG presets', () => {
    expect(presetAvatars.length).toBe(8)
    presetAvatars.forEach(av => {
      expect(av.name).toBeTruthy()
      expect(av.url).toContain('data:image/svg+xml')
    })
  })
})
