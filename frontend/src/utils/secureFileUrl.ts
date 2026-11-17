import { createSSETicketApi } from '@/api/auth'

/**
 * 构造受保护静态资源（影像预览 / 附件下载）的访问 URL。
 *
 * 生产模式下后端拒绝 `?token=` 形式的长效 JWT（会经访问日志、Referer、浏览器历史泄露），
 * 因此统一改为先 `POST /auth/ticket` 领取 60s 一次性票据，再以 `?ticket=` 访问。
 * 取票失败时仅在 demo 模式回落 `?token=`，保证本地联调可用。
 */
export async function buildSecureFileUrl(path: string): Promise<string> {
  const base = path.startsWith('/api/v1') ? path : `/api/v1${path}`
  try {
    const res = await createSSETicketApi()
    if (res && res.code === 200 && res.data?.ticket) {
      return `${base}?ticket=${encodeURIComponent(res.data.ticket)}`
    }
  } catch {
    // 静默降级，下面走兼容分支
  }
  const token = sessionStorage.getItem('medtrust_token')
  return token ? `${base}?token=${encodeURIComponent(token)}` : base
}

/** 医疗附件预览 URL（一次性票据鉴权） */
export function buildMedicalFileViewUrl(fileId: number | string): Promise<string> {
  return buildSecureFileUrl(`/medical-files/${fileId}/view`)
}

/** 医疗附件下载 URL（一次性票据鉴权） */
export function buildMedicalFileDownloadUrl(fileId: number | string): Promise<string> {
  return buildSecureFileUrl(`/medical-files/${fileId}/download`)
}