/**
 * 密码学哈希、交易 ID 与 IPFS CID 截断及剪贴板安全工具函数
 */
import { ElMessage } from 'element-plus'

/**
 * 首尾保留、中间省略号截断展示哈希字符串。
 * 空值或无效输入安全返回 '—'。
 */
export function formatHashMiddle(
  val?: string | null,
  prefixLen = 8,
  suffixLen = 6
): string {
  if (val === null || val === undefined) return '—'
  const trimmed = String(val).trim()
  if (!trimmed || trimmed === '-' || trimmed === '—') return '—'
  if (trimmed.length <= prefixLen + suffixLen + 3) {
    return trimmed
  }
  return `${trimmed.slice(0, prefixLen)}...${trimmed.slice(-suffixLen)}`
}

/**
 * 复制文本至系统剪贴板并给出轻提示。
 */
export async function copyTextToClipboard(
  text?: string | null,
  label = '凭证内容'
): Promise<boolean> {
  if (!text) return false
  const value = String(text)
  try {
    if (navigator.clipboard && window.isSecureContext) {
      await navigator.clipboard.writeText(value)
    } else {
      const textArea = document.createElement('textarea')
      textArea.value = value
      textArea.style.position = 'fixed'
      textArea.style.opacity = '0'
      document.body.appendChild(textArea)
      textArea.select()
      const ok = document.execCommand('copy')
      document.body.removeChild(textArea)
      if (!ok) throw new Error('execCommand copy failed')
    }
    ElMessage.success(`${label}已复制到剪贴板`)
    return true
  } catch {
    ElMessage.warning('复制失败，请手动选中复制')
    return false
  }
}
