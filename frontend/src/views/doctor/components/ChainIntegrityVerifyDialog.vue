<template>
  <!-- 链上完整性防篡改核验弹窗 (WP-16: 从 CrossQueryView 迁入, 双哈希对比展示复用共享的 RecordIntegrityDialog) -->
  <RecordIntegrityDialog
    v-model="integrityDialogVisible"
    :result="integrityResult"
    :loading="verifyingIntegrity"
    @download="$emit('download', $event)"
  />
</template>

<script setup lang="ts">
// WP-16: 链上完整性防篡改核验弹窗与校验结果处理逻辑, 从 CrossQueryView 迁入 (文案/接口调用保持原样)
import { ref } from 'vue'
import { ElMessage } from 'element-plus'
import { verifyRecordApi } from '../../../api'
import RecordIntegrityDialog from '../../../components/RecordIntegrityDialog.vue'
import type { IntegrityVerifyResult } from '@/types/clinical'

defineEmits<{
  (e: 'download', recordId?: number): void
}>()

// 链上完整性验真报告弹窗状态
const integrityDialogVisible = ref(false)
const verifyingIntegrity = ref(false)
const integrityResult = ref<IntegrityVerifyResult | null>(null)

async function verifyRecordIntegrity(recordId: number) {
  if (!recordId) return
  verifyingIntegrity.value = true
  integrityResult.value = null
  integrityDialogVisible.value = true
  try {
    const res = await verifyRecordApi(recordId)
    if (res.code === 200 && res.data) {
      integrityResult.value = res.data as unknown as IntegrityVerifyResult
    } else {
      ElMessage.error(res.message || '核验异常')
    }
  } catch (err: unknown) {
    const e = err as { message?: string }
    ElMessage.error('链上完整性核验失败: ' + (e.message || '网络错误'))
  } finally {
    verifyingIntegrity.value = false
  }
}

defineExpose({ verifyRecordIntegrity })
</script>