// WP-16: 影像/PDF 预览与下载、剪贴板复制等文件操作 (从 CrossQueryView 迁入, 行为保持原样)
import { ref, type Ref } from 'vue'
import { ElMessage } from 'element-plus'
import { downloadRecordApi, getPatientInfectionRisksApi } from '@/api'
import { buildMedicalFileViewUrl, buildMedicalFileDownloadUrl, buildSecureFileUrl } from '@/utils/secureFileUrl'
import type { MedicalRecordRow } from '@/types/clinical'

export function useRecordFileActions(releasedRecord: Ref<MedicalRecordRow | null>) {
  const pdfPreviewVisible = ref(false)
  const previewingRecord = ref<MedicalRecordRow | null>(null)

  const imageModalVisible = ref(false)
  const imageModalUrl = ref('')
  const imageModalTitle = ref('')
  const imageModalFileId = ref<number | null>(null)

  async function openImageModal(file: { id: number; file_name?: string }) {
    imageModalFileId.value = file.id
    imageModalUrl.value = await buildMedicalFileViewUrl(file.id)
    imageModalTitle.value = file.file_name || '医学检查图像'
    imageModalVisible.value = true
  }

  // 一次性票据只能消费一次：下载时重新申领，不能复用 <img> 已用过的 URL。
  // 传入 file 时下载该附件；不传则下载当前大图弹窗对应附件。
  async function downloadImageFile(file?: { id: number; file_name?: string }) {
    const fileId = file?.id ?? imageModalFileId.value
    if (fileId === null || fileId === undefined) return
    const url = await buildMedicalFileDownloadUrl(fileId)
    downloadFileDirect(url, file?.file_name || imageModalTitle.value)
  }

  function downloadFileDirect(url: string, fileName?: string) {
    const a = document.createElement('a')
    a.href = url
    a.download = fileName || 'medical_file'
    a.target = '_blank'
    document.body.appendChild(a)
    a.click()
    document.body.removeChild(a)
  }

  async function openPdfPreview(rec: MedicalRecordRow | null) {
    if (!rec) return
    previewingRecord.value = rec
    if (!previewingRecord.value.infection_alert && previewingRecord.value.patient_id) {
      try {
        const riskRes = await getPatientInfectionRisksApi(previewingRecord.value.patient_id)
        if (riskRes.code === 200 && riskRes.data?.has_risk) {
          previewingRecord.value.infection_alert = riskRes.data as unknown as typeof previewingRecord.value.infection_alert
        }
      } catch {}
    }
    pdfPreviewVisible.value = true
  }

  function downloadRecordFile(recordId?: number) {
    if (!recordId) return
    downloadRecordApi(recordId)
      .then((res) => {
        const blob = new Blob([res], { type: 'application/pdf;charset=utf-8' })
        const link = document.createElement('a')
        link.href = window.URL.createObjectURL(blob)
        const rNo = previewingRecord.value?.record_no || releasedRecord.value?.record_no || recordId
        link.download = `临床就诊规范归档凭据_${rNo}.pdf`
        document.body.appendChild(link)
        link.click()
        document.body.removeChild(link)
        window.URL.revokeObjectURL(link.href)
        ElMessage.success('已成功下载解密归档凭据！')
      })
      .catch(async () => {
        window.open(await buildSecureFileUrl(`/medical-records/${recordId}/download`), '_blank')
      })
  }

  function copyToClipboard(text: string, label = '交易哈希') {
    if (!text) return
    if (navigator.clipboard) {
      navigator.clipboard.writeText(text).then(() => {
        ElMessage.success(`已复制${label}到剪贴板`)
      }).catch(() => {
        fallbackCopy(text, label)
      })
    } else {
      fallbackCopy(text, label)
    }
  }

  function fallbackCopy(text: string, label: string) {
    const textArea = document.createElement('textarea')
    textArea.value = text
    document.body.appendChild(textArea)
    textArea.select()
    try {
      document.execCommand('copy')
      ElMessage.success(`已复制${label}到剪贴板`)
    } catch (err) {
      ElMessage.error('复制失败，请手动复制')
    }
    document.body.removeChild(textArea)
  }

  return {
    pdfPreviewVisible,
    previewingRecord,
    imageModalVisible,
    imageModalUrl,
    imageModalTitle,
    openImageModal,
    downloadImageFile,
    downloadFileDirect,
    openPdfPreview,
    downloadRecordFile,
    copyToClipboard,
  }
}