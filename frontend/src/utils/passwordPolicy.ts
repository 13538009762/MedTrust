/**
 * 统一密码强度校验（与后端 backend/pkg/security/password.go 保持严格一致）：
 * - 长度 >= 8 个字符
 * - 必须同时包含英文字母与数字
 * - 不能属于常见弱口令黑名单
 */
const COMMON_WEAK_PASSWORDS = new Set([
  '123456',
  '12345678',
  '123456789',
  'password',
  'admin123',
  'admin',
  'root',
  'qwerty',
])

export function validatePasswordStrength(password: string): string | null {
  const trimmed = (password || '').trim()
  if (trimmed.length < 8) {
    return '密码长度不能少于 8 个字符'
  }
  if (COMMON_WEAK_PASSWORDS.has(trimmed.toLowerCase())) {
    return '所选密码过于简单，属于常见弱密码字典，请使用更复杂的密码组合'
  }
  const hasLetter = /[A-Za-z]/.test(trimmed)
  const hasDigit = /\d/.test(trimmed)
  if (!hasLetter || !hasDigit) {
    return '密码必须同时包含英文字母和数字组合'
  }
  return null
}

/**
 * 统一判断 API 错误是否已在全局拦截器中弹出过提示，避免重复弹窗
 */
export function isErrorToastShown(err: unknown): boolean {
  return Boolean(err && typeof err === 'object' && (err as { __toastShown?: boolean }).__toastShown)
}

/**
 * 安全提取接口错误文案（兼容拦截器解包后的 { code, message } 与原始 AxiosError）
 */
export function extractApiErrorMessage(err: unknown, fallback = '操作失败，请稍后重试'): string {
  if (!err || typeof err !== 'object') return fallback
  const e = err as { message?: string; response?: { data?: { message?: string } } }
  return e.response?.data?.message || e.message || fallback
}
