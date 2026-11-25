import { describe, it, expect } from 'vitest'
import { validatePasswordStrength } from '../passwordPolicy'

describe('passwordPolicy frontend-backend contract alignment', () => {
  it('rejects passwords shorter than 8 chars or lacking letter/digit mix', () => {
    expect(validatePasswordStrength('123456')).toContain('8')
    expect(validatePasswordStrength('12345678')).toBeTruthy()
    expect(validatePasswordStrength('abcdefgh')).toContain('字母和数字')
    expect(validatePasswordStrength('admin123')).toContain('弱密码')
  })

  it('accepts strong alphanumeric passwords >= 8 chars', () => {
    expect(validatePasswordStrength('MedTrust2026')).toBeNull()
    expect(validatePasswordStrength('a1b2c3d4')).toBeNull()
  })
})
