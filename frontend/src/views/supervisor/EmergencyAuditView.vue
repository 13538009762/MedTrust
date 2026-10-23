<template>
  <div class="page-container">
    <div class="page-header flex-between">
      <div>
        <h2 class="page-title">紧急访问 (Break-Glass) 监管审核台</h2>
        <p class="page-sub">监管人员为唯一的医疗合规审判者。审查急救原因、患者知情异议反馈，执行合规结案或阶梯式违规惩戒与复权治理</p>
      </div>
      <div class="header-actions">
        <el-button type="warning" plain @click="openGeneralLiftModal">
          受限医生复权管理
        </el-button>
      </div>
    </div>

    <el-card shadow="hover" class="box-card">
      <el-table :data="events" v-loading="loading" stripe :row-class-name="tableRowClassName" style="width: 100%">
        <el-table-column prop="event_no" label="事件流水单号" width="160" />
        <el-table-column prop="doctor_name" label="调阅医生" width="120" />
        <el-table-column prop="source_hospital_name" label="调阅方医院" width="150" />
        <el-table-column prop="patient_name" label="涉及患者" width="100" />
        <el-table-column prop="emergency_reason" label="急救原因" width="110">
          <template #default="{ row }">
            <el-tag size="small" type="danger">{{ row.emergency_reason }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="patient_feedback" label="患者知情确认" width="145">
          <template #default="{ row }">
            <el-tag v-if="row.patient_feedback === 'CONFIRMED'" type="success" size="small" effect="plain">正常抢救 (无异议)</el-tag>
            <el-tag v-else-if="row.patient_feedback === 'OBJECTED'" type="danger" size="small" effect="dark">
              ⚠️ 提出异议投诉
            </el-tag>
            <el-tag v-else type="info" size="small">等待确认</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="audit_status" label="监管审核状态" width="140">
          <template #default="{ row }">
            <el-tag v-if="row.audit_status === 'CLOSED_APPROVED'" type="success" size="small">正常结案</el-tag>
            <el-tag v-else-if="row.audit_status === 'CLOSED_VIOLATION'" type="danger" size="small">确认违规</el-tag>
            <el-tag v-else-if="row.audit_status === 'UNDER_INVESTIGATION'" type="warning" size="small">待调查</el-tag>
            <el-tag v-else type="primary" size="small">待审核</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="审核判定与处置" width="200" fixed="right">
          <template #default="{ row }">
            <div class="action-btns">
              <el-button
                type="primary"
                size="small"
                @click="openAuditModal(row)"
              >
                审查裁决
              </el-button>
              <el-button
                v-if="row.audit_status === 'CLOSED_VIOLATION'"
                type="warning"
                size="small"
                plain
                @click="openLiftModal(row)"
              >
                解除限制
              </el-button>
            </div>
          </template>
        </el-table-column>
      </el-table>
    </el-card>

    <!-- 监管裁决对话框 -->
    <el-dialog v-model="dialogVisible" title="紧急调阅合规审查与裁决" width="580px">
      <div v-if="currentEvent" class="audit-info-box">
        <p><strong>事件单号：</strong><span class="mono font-bold">{{ currentEvent.event_no }}</span></p>
        <p><strong>调阅医生：</strong>{{ currentEvent.doctor_name }} ({{ currentEvent.source_hospital_name }})</p>
        <p><strong>就诊患者：</strong>{{ currentEvent.patient_name }}</p>
        <p><strong>医生抢救说明：</strong>{{ getCleanDoctorDesc(currentEvent.description) }}</p>
        <p><strong>患者反馈结论：</strong>
          <el-tag size="small" :type="currentEvent.patient_feedback === 'OBJECTED' ? 'danger' : 'success'">
            {{ currentEvent.patient_feedback === 'OBJECTED' ? '⚠️ 已提出异议申诉' : (currentEvent.patient_feedback === 'CONFIRMED' ? '已确认无异议 (正常抢救)' : '等待患者核验') }}
          </el-tag>
        </p>
        <el-alert
          v-if="currentEvent.patient_feedback === 'OBJECTED' || (currentEvent.description && currentEvent.description.includes('|'))"
          type="error"
          title="【患者异议投诉预警】该紧急事件已被患者提出异议！"
          :description="`患者陈述理由: ${getExtractedPatientObjection(currentEvent)}`"
          show-icon
          :closable="false"
          class="mt-2 mb-2"
        />
        <el-divider />
      </div>

      <el-form :model="form" label-width="110px">
        <el-form-item label="合规审查判定" required>
          <el-select v-model="form.audit_status" style="width: 100%;">
            <el-option label="合规结案 · 真实急危重症抢救，符合避险合规要求" value="CLOSED_APPROVED" />
            <el-option label="补充证据 · 存在存疑，责令医生补充出具急救处置证据" value="UNDER_INVESTIGATION" />
            <el-option label="违规处罚 · 判定属于违规越权套取隐私，予以惩戒" value="CLOSED_VIOLATION" />
          </el-select>
        </el-form-item>
        <el-form-item label="阶梯惩戒措施" v-if="form.audit_status === 'CLOSED_VIOLATION'" required>
          <el-select v-model="form.punishment" style="width: 100%;">
            <el-option label="限制执业 · 暂停该医生跨院调阅与急救破窗权限 (RESTRICTED)" value="RESTRICTED" />
            <el-option label="封禁停用 · 情节恶劣严重，彻底停用医生账号 (DISABLED)" value="DISABLED" />
          </el-select>
        </el-form-item>
        <el-form-item label="监管处置意见" required>
          <el-input v-model="form.audit_comment" type="textarea" :rows="3" placeholder="填写监管审核处理结论，将作为不可篡改证据记录上链..." />
        </el-form-item>
      </el-form>

      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="submitting" @click="submitAudit">提交裁决并上链固化</el-button>
      </template>
    </el-dialog>

    <!-- 一键解除医生权限限制（复权审批）弹窗 -->
    <el-dialog v-model="liftDialogVisible" title="解除医生权限限制（复权审批）" width="540px">
      <el-alert
        title="复权机制说明"
        type="info"
        description="当受限医生整改期满或申诉复核通过后，监管部门可解除其访问控制限制，将其账号状态由 RESTRICTED 恢复为 NORMAL 正常，并在区块链上记存复权审计日志。"
        show-icon
        :closable="false"
        class="mb-3"
      />
      <el-form :model="liftForm" label-width="110px">
        <el-form-item label="选择医生" required>
          <el-select
            v-if="!selectedFromRow"
            v-model="liftForm.doctor_id"
            placeholder="请选择需要解除限制的医生"
            style="width: 100%;"
            filterable
          >
            <el-option
              v-for="doc in doctorList"
              :key="doc.id"
              :label="`${doc.real_name || doc.username} (${doc.hospital_name || '医院'} · 状态: ${doc.status})`"
              :value="doc.id"
            />
          </el-select>
          <el-input v-else v-model="liftDoctorDisplay" disabled />
        </el-form-item>
        <el-form-item label="复权处置意见" required>
          <el-input
            v-model="liftForm.comment"
            type="textarea"
            :rows="3"
            placeholder="请输入复权审查理由，该决议将写入区块链存证（例如：经监管复核，医生已完成整改且急救合规培训考核合格，解除限制恢复正常）"
          />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="liftDialogVisible = false">取消</el-button>
        <el-button type="success" :loading="lifting" @click="submitLiftRestriction">
          确认解除限制并恢复正常
        </el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { ElMessage } from 'element-plus'
import api from '../../api/client'

const events = ref([])
const loading = ref(false)
const dialogVisible = ref(false)
const currentEvent = ref<any>(null)
const submitting = ref(false)

const form = ref({
  audit_status: 'CLOSED_APPROVED',
  punishment: 'RESTRICTED',
  audit_comment: '经监管人员独立调查调阅明细，本次抢救记录符合突发抢救规范，准予结案。',
})

onMounted(() => {
  loadEvents()
})

async function loadEvents() {
  loading.value = true
  try {
    const res: any = await api.get('/supervisor/emergency-events')
    if (res.code === 200) {
      events.value = res.data
    }
  } finally {
    loading.value = false
  }
}

function getCleanDoctorDesc(desc?: string): string {
  if (!desc) return '急重症突发昏迷急救，启动 Break-Glass 紧急访问通道调阅'
  return desc.split(/\|\s*\[(?:患者反馈|患者异议|患者陈述)[:：]?/)[0].trim()
}

function getExtractedPatientObjection(ev?: any): string {
  if (!ev) return ''
  if (ev.patient_comment && ev.patient_comment.trim()) {
    return ev.patient_comment.trim()
  }
  if (ev.description && ev.description.includes('|')) {
    const parts = ev.description.split(/\|\s*\[(?:患者反馈|患者异议|患者陈述)[:：]?\s*/)
    if (parts.length > 1) {
      return parts[1].replace(/\]\s*$/, '').trim()
    }
  }
  return '当时我意识完全清醒，并未发生急重抢救，涉嫌违规调阅个人病历。'
}

function openAuditModal(row: any) {
  currentEvent.value = row
  if (row.patient_feedback === 'OBJECTED') {
    form.value.audit_status = 'CLOSED_VIOLATION'
    form.value.audit_comment = '经核查患者投诉，医生未能提供客观抢救单据，判定属于滥用 Break-Glass 机制，处罚限制跨院权限。'
  }
  dialogVisible.value = true
}

async function submitAudit() {
  if (!currentEvent.value) return
  submitting.value = true
  try {
    const res: any = await api.post(`/supervisor/emergency-events/${currentEvent.value.event_no}/audit`, {
      audit_status: form.value.audit_status,
      audit_comment: form.value.audit_comment,
      punishment: form.value.audit_status === 'CLOSED_VIOLATION' ? form.value.punishment : 'NORMAL',
    })
    if (res.code === 200) {
      ElMessage.success('审核判定已成功提交并在 Fabric 链上固化，账号惩戒措施即刻生效！')
      dialogVisible.value = false
      loadEvents()
    }
  } catch (err: any) {
    ElMessage.error(err?.message || '审核提交失败')
  } finally {
    submitting.value = false
  }
}

const liftDialogVisible = ref(false)
const selectedFromRow = ref(false)
const liftDoctorDisplay = ref('')
const lifting = ref(false)
const doctorList = ref<any[]>([])

const liftForm = ref({
  doctor_id: undefined as number | undefined,
  comment: '经监管部门复核申诉与整改情况，急救合规考核合格，准予解除权限限制恢复正常执业状态。',
})

async function loadDoctorList() {
  try {
    const res: any = await api.get('/system/doctors')
    if (res.code === 200) {
      doctorList.value = res.data || []
    }
  } catch (err) {
    console.warn('loadDoctorList failed', err)
  }
}

function openLiftModal(row: any) {
  selectedFromRow.value = true
  liftForm.value.doctor_id = row.doctor_id
  liftDoctorDisplay.value = `${row.doctor_name} (工号/ID: ${row.doctor_id} · ${row.source_hospital_name})`
  liftForm.value.comment = '经监管复核，医生已补交客观急救材料并通过合规审查，同意解除权限限制。'
  liftDialogVisible.value = true
}

async function openGeneralLiftModal() {
  selectedFromRow.value = false
  await loadDoctorList()
  const restricted = doctorList.value.find(d => d.status === 'RESTRICTED')
  if (restricted) {
    liftForm.value.doctor_id = restricted.id
  } else if (doctorList.value.length > 0) {
    liftForm.value.doctor_id = doctorList.value[0].id
  }
  liftForm.value.comment = '经监管复核，医生已完成合规培训，解除权限限制。'
  liftDialogVisible.value = true
}

async function submitLiftRestriction() {
  if (!liftForm.value.doctor_id) {
    ElMessage.warning('请选择需要解除限制的医生')
    return
  }
  lifting.value = true
  try {
    const res: any = await api.post('/supervisor/lift-doctor-restriction', {
      doctor_id: liftForm.value.doctor_id,
      comment: liftForm.value.comment || '经监管部门审查准予解除惩戒限制。'
    })
    if (res.code === 200) {
      ElMessage.success('已成功解除该医生的限制，状态恢复为 NORMAL 正常！区块链已记录复权存证。')
      liftDialogVisible.value = false
      loadEvents()
      loadDoctorList()
    }
  } catch (err: any) {
    ElMessage.error(err?.message || '解除限制失败')
  } finally {
    lifting.value = false
  }
}

function tableRowClassName({ row }: { row: any }) {
  if (row.patient_feedback === 'OBJECTED') {
    return 'objected-row'
  }
  return ''
}
</script>

<style scoped>
:deep(.objected-row) {
  background-color: #fff1f2 !important;
}
:deep(.objected-row td) {
  background-color: #fff1f2 !important;
}
.page-container {
  max-width: 1300px;
  margin: 0 auto;
}
.page-header {
  margin-bottom: 20px;
}
.flex-between {
  display: flex;
  justify-content: space-between;
  align-items: center;
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
.audit-info-box p {
  line-height: 1.8;
  font-size: 14px;
}
.action-btns {
  display: flex;
  align-items: center;
  gap: 8px;
}
</style>
