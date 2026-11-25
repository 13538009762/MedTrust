<template>
  <div class="page-container">
    <div class="page-header flex justify-between items-center flex-wrap gap-2">
      <div>
        <h2 class="page-title">紧急访问知情与异议申诉中心</h2>
        <p class="page-sub">当医生在急危重症抢救中紧急破窗调阅您的病历时，系统将在此实时通知，您可核实确认或提交异议申诉</p>
      </div>
      <el-button plain :loading="loadingEvents" @click="loadEvents">刷新通知</el-button>
    </div>

    <el-card shadow="hover" class="box-card" v-loading="loadingEvents" element-loading-text="正在加载紧急访问通知...">
      <el-result
        v-if="loadError && !loadingEvents"
        icon="error"
        title="紧急访问通知加载失败"
        :sub-title="loadError"
      >
        <template #extra>
          <el-button type="primary" @click="loadEvents">重新加载</el-button>
        </template>
      </el-result>
      <div v-else-if="events.length">
        <div
          v-for="ev in events"
          :key="ev.id"
          class="event-card"
          :class="ev.patient_feedback === 'PENDING' ? 'pending-border' : ''"
        >
          <div class="event-header">
            <span class="event-no">事件流水单号: <strong>{{ ev.event_no }}</strong></span>
            <el-tag
              :type="ev.patient_feedback === 'PENDING' ? 'warning' : (ev.patient_feedback === 'CONFIRMED' ? 'success' : 'danger')"
              :effect="ev.patient_feedback === 'OBJECTED' ? 'plain' : 'dark'"
            >
              {{ ev.patient_feedback === 'PENDING' ? '待您知情确认' : (ev.patient_feedback === 'CONFIRMED' ? '已确认无异议 (正常抢救)' : '已提出异议投诉') }}
            </el-tag>
          </div>

          <div class="event-body">
            <p><strong>调阅医生：</strong>{{ ev.doctor_name }} ({{ ev.source_hospital_name }})</p>
            <p><strong>调阅病历编号：</strong><code class="mono font-bold">{{ ev.record_no }}</code></p>
            <p><strong>紧急原因：</strong><el-tag size="small" type="danger" effect="plain">{{ formatEmergencyReason(ev.emergency_reason) }}</el-tag></p>
            <p><strong>医生急救临床说明：</strong>{{ getCleanDoctorDesc(ev.description) }}</p>

            <!-- 独立醒目的异议申诉记录区块 (严禁与医生说明粘连) -->
            <div v-if="getExtractedObjection(ev)" class="objection-alert-card mt-2 mb-2">
              <div class="oac-title">
                <el-icon class="mr-1 text-red-600"><WarningFilled /></el-icon>
                <span>我的异议申诉记录：</span>
              </div>
              <p class="oac-text">{{ getExtractedObjection(ev) }}</p>
            </div>

            <p v-if="ev.audit_status">
              <strong>监管合规裁决：</strong>
              <el-tag size="small" :type="ev.audit_status === 'CLOSED_APPROVED' ? 'success' : (ev.audit_status === 'CLOSED_VIOLATION' ? 'danger' : 'warning')">
                {{ ev.audit_status === 'CLOSED_APPROVED' ? '合规正常结案' : (ev.audit_status === 'CLOSED_VIOLATION' ? '判定违规 (已惩戒医生)' : '调查核实中') }}
              </el-tag>
            </p>
            <p><strong>上链存证 TxID：</strong><HashCapsule :value="ev.fabric_tx_id" :prefix-len="10" :suffix-len="8" label="上链存证 TxID" /></p>
            <p><strong>调阅时间：</strong><span class="mono">{{ formatEventTime(ev.created_at) }}</span></p>
          </div>

          <div v-if="ev.patient_feedback === 'PENDING'" class="event-actions">
            <el-button
              type="success"
              :loading="submittingEventNo === ev.event_no"
              @click="confirmNormalRescue(ev)"
            >
              确认属于正常抢救 (无异议)
            </el-button>
            <el-button
              type="danger"
              plain
              :disabled="submittingEventNo === ev.event_no"
              @click="openObjectionModal(ev)"
            >
              对本次调阅提出异议投诉
            </el-button>
          </div>
        </div>
      </div>
      <el-empty v-else-if="!loadingEvents" description="暂无紧急访问事件通知，您的医疗数据安全在控" />
    </el-card>

    <el-dialog v-model="dialogVisible" title="提出紧急调阅异议投诉" width="min(500px, 92vw)">
      <el-form label-width="80px">
        <el-form-item label="异议理由" required>
          <el-input v-model="comment" type="textarea" :rows="3" placeholder="请如实填写您的异议说明（例如：本人当时未在该院就诊、意识清醒未被告知等）..." />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="danger" :loading="submittingObjection" :disabled="!comment.trim()" @click="submitObjection">提交监管审核</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, onUnmounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import HashCapsule from '@/components/common/HashCapsule.vue'
import { listEmergencyEventsApi, submitBreakGlassFeedbackApi } from '../../api'
import { useSSE } from '@/composables/useSSE'

const { addEventListener: addSSEListener } = useSSE()
let removeSSEListener: (() => void) | null = null

const events = ref<any[]>([])
const loadingEvents = ref(false)
const loadError = ref('')
const dialogVisible = ref(false)
const targetEventNo = ref('')
const comment = ref('')
const submittingEventNo = ref('')
const submittingObjection = ref(false)

onMounted(() => {
  loadEvents()
  removeSSEListener = addSSEListener((event) => {
    if (event.type === 'break_glass') {
      loadEvents()
    }
  })
})

onUnmounted(() => {
  if (removeSSEListener) {
    removeSSEListener()
    removeSSEListener = null
  }
})

async function loadEvents() {
  loadingEvents.value = true
  loadError.value = ''
  try {
    const res: any = await listEmergencyEventsApi()
    if (res.code === 200) {
      events.value = Array.isArray(res.data) ? res.data : []
    } else {
      loadError.value = res?.message || '获取紧急访问记录失败，请稍后重试'
    }
  } catch (err: any) {
    loadError.value = err?.message || '网络连接异常，无法获取紧急访问通知'
  } finally {
    loadingEvents.value = false
  }
}

async function confirmNormalRescue(ev: any) {
  try {
    await ElMessageBox.confirm(
      `确认 ${ev.doctor_name || '该医生'} 于 ${formatEventTime(ev.created_at)} 对病历 ${ev.record_no} 的紧急调阅属于正常抢救且无异议吗？提交后不可撤销。`,
      '确认正常抢救无异议',
      {
        confirmButtonText: '确认无异议',
        cancelButtonText: '取消',
        type: 'info',
      }
    )
  } catch {
    return
  }
  await handleFeedback(ev.event_no, 'CONFIRMED')
}

async function handleFeedback(eventNo: string, feedback: string, cmt: string = ''): Promise<boolean> {
  submittingEventNo.value = eventNo
  try {
    const res: any = await submitBreakGlassFeedbackApi({
      event_no: eventNo,
      feedback: feedback,
      comment: cmt,
    })
    if (res.code === 200) {
      ElMessage.success('反馈已成功提交至监管后台')
      loadEvents()
      return true
    } else {
      ElMessage.error(res?.message || '反馈提交失败')
      return false
    }
  } catch (err: any) {
    if (!err?.__toastShown) {
      ElMessage.error(err?.message || '反馈提交失败')
    }
    return false
  } finally {
    submittingEventNo.value = ''
  }
}

function openObjectionModal(ev: any) {
  targetEventNo.value = ev.event_no
  comment.value = ''
  dialogVisible.value = true
}

function getCleanDoctorDesc(desc?: string): string {
  if (!desc) return '（医生未填写补充说明）'
  return desc.split(/\|\s*\[(?:患者反馈|患者异议|患者陈述)[:：]?/)[0].trim() || '（医生未填写补充说明）'
}

function getExtractedObjection(ev: any): string {
  if (ev.patient_comment && ev.patient_comment.trim()) {
    return ev.patient_comment.trim()
  }
  if (ev.description && ev.description.includes('|')) {
    const parts = ev.description.split(/\|\s*\[(?:患者反馈|患者异议|患者陈述)[:：]?\s*/)
    if (parts.length > 1) {
      return parts[1].replace(/\]\s*$/, '').trim()
    }
  }
  return ''
}

function formatEmergencyReason(reason?: string): string {
  if (!reason) return '急危重症抢救'
  if (reason === 'COMA' || reason.includes('COMA')) return '急性昏迷 / 意识障碍抢救'
  if (reason === 'CARDIAC_ARREST' || reason === 'RESCUE') return '心搏骤停 / 急性心梗急救'
  if (reason === 'SEVERE_TRAUMA' || reason === 'SHOCK' || reason === 'CRITICAL') return '严重创伤 / 休克急救'
  if (reason === 'ACUTE_STROKE') return '急性脑卒中绿色通道'
  if (reason === 'EMERGENCY') return '急危重症抢救生命关键期'
  if (reason === 'OTHER') return '其他危及生命的紧急医学场景'
  return reason
}

function formatEventTime(t?: string): string {
  if (!t) return '-'
  return t.replace('T', ' ').replace(/\s+/g, ' ').substring(0, 16)
}

async function submitObjection() {
  const trimmed = comment.value.trim()
  if (!trimmed) {
    ElMessage.warning('请填写您的异议理由后再提交')
    return
  }
  submittingObjection.value = true
  try {
    const ok = await handleFeedback(targetEventNo.value, 'OBJECTED', trimmed)
    if (ok) {
      dialogVisible.value = false
    }
  } finally {
    submittingObjection.value = false
  }
}
</script>

<style scoped>
.page-container {
  max-width: 1100px;
  margin: 0 auto;
}
.page-header {
  margin-bottom: 20px;
}
.page-title {
  font-size: 22px;
  font-weight: 800;
  color: #0f172a;
}
.page-sub {
  font-size: 13px;
  color: #64748b;
}
.box-card {
  border-radius: 14px;
}
.event-card {
  border: 1px solid #e2e8f0;
  border-radius: 12px;
  padding: 18px;
  margin-bottom: 16px;
  background: #ffffff;
  transition: all 0.3s;
}
.pending-border {
  border-left: 5px solid #ef4444;
  background: #fef2f2;
}
.event-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 12px;
}
.event-no {
  font-size: 14px;
  color: #475569;
}
.event-body p {
  line-height: 1.8;
  font-size: 14px;
  color: #1e293b;
}
.tx-hash {
  color: #8b5cf6;
}
.event-actions {
  margin-top: 14px;
  display: flex;
  gap: 12px;
}

.objection-alert-card {
  background: #fef2f2;
  border: 1px solid #fecaca;
  border-left: 4px solid #ef4444;
  border-radius: 8px;
  padding: 10px 14px;
}
.oac-title {
  display: flex;
  align-items: center;
  font-weight: 700;
  color: #b91c1c;
  font-size: 13px;
  margin-bottom: 4px;
}
.oac-text {
  color: #991b1b;
  font-size: 13px;
  margin: 0;
  line-height: 1.5;
}

</style>
