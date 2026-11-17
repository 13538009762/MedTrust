<template>
  <!-- 病历归档附件与规范 EMR 凭据上传/查阅面板 (WP-16: 从 DoctorRecordsView DETAIL 左侧卡片 3 抽取)
       展示 IPFS 存证附件列表与大图/红头 PDF 查阅、下载入口，行为与原文件一致。 -->
  <div class="bento-card files-card">
    <div class="files-card-header">
      <div class="fch-title">
        <el-icon color="#0284c7"><Files /></el-icon>
        <span>临床归档凭据与附件</span>
      </div>
      <el-tag size="small" type="info" effect="plain" class="fch-tag">IPFS 存证</el-tag>
    </div>

    <div class="archive-files-section">
      <div v-if="record?.files && record.files.length" class="file-cards-list">
        <div v-for="file in getDistinctFiles(record.files)" :key="file.id" class="file-item-bento">
          <!-- 顶部行：图标 + 文件名 + 类型与大小 -->
          <div class="fib-top-row">
            <div class="fib-icon">
              <el-icon v-if="file.file_type === 'pdf'" color="#dc2626"><Document /></el-icon>
              <el-icon v-else-if="isImageFileType(file.file_type)" color="#0284c7"><Picture /></el-icon>
              <el-icon v-else color="#64748b"><Files /></el-icon>
            </div>
            <div class="fib-meta">
              <div class="fib-name-box">
                <span class="fib-name" :title="file.file_name || '临床就诊规范归档凭据.pdf'">{{ file.file_name || '临床就诊规范归档凭据.pdf' }}</span>
              </div>
              <div class="fib-tags">
                <el-tag size="small" type="primary" effect="light" class="type-tag">{{ (file.file_type || 'PDF').toUpperCase() }}</el-tag>
                <el-tag size="small" type="success" effect="plain" class="size-tag">{{ formatFileSize(file.file_size) }}</el-tag>
              </div>
            </div>
          </div>

          <!-- 中间行：IPFS CID 存证哈希 -->
          <div class="fib-cid-row">
            <span class="cid-lbl">IPFS CID：</span>
            <code class="cid-val mono" :title="file.ipfs_cid">{{ shortHash(file.ipfs_cid, 10, 8) }}</code>
            <el-tooltip content="复制 IPFS CID" placement="top">
              <el-icon class="copy-icon ml-1" @click.stop="$emit('copy-text', file.ipfs_cid || '', 'IPFS CID')"><CopyDocument /></el-icon>
            </el-tooltip>
          </div>

          <!-- 底部行：独立操作按钮条 -->
          <div class="fib-actions-row">
            <template v-if="isImageFileType(file.file_type)">
              <el-button type="primary" size="small" plain :icon="View" class="fib-btn" @click="$emit('view-image', file)">查阅大图</el-button>
              <el-button type="success" size="small" plain :icon="Download" class="fib-btn" @click="$emit('download-image', file)">下载原图</el-button>
            </template>
            <template v-else>
              <el-button type="primary" size="small" plain :icon="View" class="fib-btn" @click="$emit('view-pdf', record)">查阅红头 PDF</el-button>
              <el-button type="success" size="small" plain :icon="Download" class="fib-btn" @click="$emit('download-record', record.id)">下载凭据</el-button>
            </template>
          </div>

          <!-- 略缩图（如果是图片） -->
          <div v-if="isImageFileType(file.file_type)" class="fib-thumb-box mt-2">
            <img
              :src="thumbUrl(file.id)"
              class="preview-thumbnail"
              @click="$emit('view-image', file)"
              title="点击全屏查阅大图"
            />
          </div>
        </div>
      </div>

      <div v-else class="empty-files-bento">
        <div class="efb-title font-medium text-slate-800">标准规范电子病历归档凭证 (PDF)</div>
        <div class="efb-desc text-xs text-gray-500 mt-1">已自动生成全量 SOAP 规范电子病历并完成区块链存证</div>
        <div class="efb-btns mt-2">
          <el-button type="primary" size="small" plain :icon="View" class="fib-btn" @click="$emit('view-pdf', record)">
            查阅红头 PDF
          </el-button>
          <el-button type="success" size="small" plain :icon="Download" class="fib-btn" @click="$emit('download-record', record?.id)">
            下载加密凭据
          </el-button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, watch } from 'vue'
import { View, Download, Document, Picture, Files, CopyDocument } from '@element-plus/icons-vue'
import { formatFileSize } from '@/utils/recordFormatters'
import { buildMedicalFileViewUrl } from '@/utils/secureFileUrl'
import type { MedicalRecordRow, MedicalFileItem } from '@/types/clinical'

const props = defineProps<{
  record: MedicalRecordRow | null
  token: string
}>()

defineEmits<{
  (e: 'view-image', file: MedicalFileItem): void
  (e: 'view-pdf', record: MedicalRecordRow | null): void
  (e: 'download-image', file: MedicalFileItem): void
  (e: 'download-record', recordId?: number): void
  (e: 'copy-text', text: string, label: string): void
}>()

// 略缩图需在模板中同步取值，故在 record 变化时批量预领一次性票据
const thumbUrls = ref<Record<number, string>>({})

watch(
  () => props.record,
  async (rec) => {
    thumbUrls.value = {}
    if (!rec?.files?.length) return
    const images = rec.files.filter((f) => isImageFileType(f.file_type))
    const entries = await Promise.all(
      images.map(async (f) => [f.id, await buildMedicalFileViewUrl(f.id)] as const)
    )
    thumbUrls.value = Object.fromEntries(entries)
  },
  { immediate: true }
)

function thumbUrl(fileId: number): string {
  return thumbUrls.value[fileId] || ''
}

function getDistinctFiles(files: MedicalFileItem[]) {
  if (!files || !files.length) return []
  const seen = new Set<string>()
  return files.filter((f: MedicalFileItem) => {
    const key = f.ipfs_cid || String(f.id)
    if (seen.has(key)) return false
    seen.add(key)
    return true
  })
}

function isImageFileType(ft?: string) {
  if (!ft) return false
  const lower = ft.toLowerCase()
  return ['png', 'jpg', 'jpeg', 'gif', 'webp', 'svg'].includes(lower) || lower.includes('png') || lower.includes('jpg') || lower.includes('jpeg')
}

function shortHash(hash?: string, left = 12, right = 10) {
  if (!hash) return '-'
  if (hash.length <= left + right) return hash
  return `${hash.slice(0, left)}...${hash.slice(-right)}`
}
</script>
<style scoped src="@/styles/clinical-workstation.css"></style>
