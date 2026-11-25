<template>
  <div class="page-container">
    <div class="page-header flex-between">
      <div>
        <h2 class="page-title">紧急访问 (Break-Glass) 监管审核台</h2>
        <p class="page-sub">监管人员为唯一的医疗合规审判者。审查急救原因、患者知情异议反馈，执行合规结案或阶梯式违规惩戒与复权治理</p>
      </div>
      <div class="header-actions flex items-center gap-2">
        <el-button plain :loading="loading" @click="loadEvents">刷新工单</el-button>
        <el-button type="warning" plain @click="openGeneralLiftModal">
          受限医生复权管理
        </el-button>
      </div>
    </div>

    <el-card shadow="hover" class="box-card">
      <div class="mb-3 flex items-center justify-between flex-wrap gap-2">
        <el-radio-group v-model="statusFilter" size="small">
          <el-radio-button value="ALL">全部事件 ({{ events.length }})</el-radio-button>
          <el-radio-button value="PENDING">待审核 ({{ pendingAuditCount }})</el-radio-button>
          <el-radio-button value="OBJECTED">患者异议 ({{ objectedCount }})</el-radio-button>
          <el-radio-button value="CLOSED">已结案 ({{ closedCount }})</el-radio-button>
        </el-radio-group>
      </div>
      <el-table :data="filteredEvents" v-loading="loading" stripe :row-class-name="tableRowClassName" style="width: 100%" empty-text="当前筛选条件下暂无紧急访问工单">
        <el-table-column prop="event_no" label="事件流水单号" width="160" />
        <el-table-column prop="created_at" label="发生时间" width="155">
          <template #default="{ row }">
            <span class="mono text-xs">{{ row.created_at ? String(row.created_at).replace('T', ' ').slice(0, 16) : '-' }}</span>
          </template>
        </el-table-column>
        <el-table-column prop="doctor_name" label="调阅医生" width="120" />
        <el-table-column prop="source_hospital_name" label="调阅方医院" width="150" />
        <el-table-column prop="patient_name" label="涉及患者" width="100" />
        <el-table-column prop="emergency_reason" label="急救原因" width="110">
          <template #default="{ row }">
            <el-tag size="small" type="danger">{{ row.emergency_reason }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="rule_pre_screen" label="规则初筛结论" width="160">
          <template #default="{ row }">
            <el-tag
              size="small"
              :type="row.rule_pre_screen === 'RULE_WARN_SUSPICIOUS' ? 'warning' : 'success'"
              :effect="row.rule_pre_screen === 'RULE_WARN_SUSPICIOUS' ? 'dark' : 'light'"
            >
              {{ row.rule_pre_screen === 'RULE_WARN_SUSPICIOUS' ? '非典型破窗需复核' : '符合危急重症指征' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="patient_feedback" label="患者知情确认" width="145">
          <template #default="{ row }">
            <el-tag v-if="row.patient_feedback === 'CONFIRMED'" type="success" size="small" effect="plain">正常抢救 (无异议)</el-tag>
            <el-tag v-else-if="row.patient_feedback === 'OBJECTED'" type="danger" size="small" effect="dark">
              提出异议投诉
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
    <el-dialog v-model="dialogVisible" title="紧急调阅合规审查与裁决" width="min(580px, 92vw)">
      <div v-if="currentEvent" class="audit-info-box">
        <p><strong>事件单号：</strong><span class="mono font-bold">{{ currentEvent.event_no }}</span></p>
        <p><strong>调阅医生：</strong>{{ currentEvent.doctor_name }} ({{ currentEvent.source_hospital_name }})</p>
        <p><strong>就诊患者：</strong>{{ currentEvent.patient_name }}</p>
        <p><strong>医生抢救说明：</strong>{{ getCleanDoctorDesc(currentEvent.description) }}</p>
        <p><strong>第一道关卡 (规则初筛)：</strong>
          <el-tag
            size="small"
            :type="currentEvent.rule_pre_screen === 'RULE_WARN_SUSPICIOUS' ? 'warning' : 'success'"
            :effect="currentEvent.rule_pre_screen === 'RULE_WARN_SUSPICIOUS' ? 'dark' : 'plain'"
          >
            {{ currentEvent.rule_pre_screen_desc || (currentEvent.rule_pre_screen === 'RULE_WARN_SUSPICIOUS' ? '[规则初筛：体征/场景不典型，建议重点复核]' : '[规则初筛：符合急诊危急重症指征]') }}
          </el-tag>
        </p>
        <p><strong>患者反馈结论：</strong>
          <el-tag size="small" :type="currentEvent.patient_feedback === 'OBJECTED' ? 'danger' : (currentEvent.patient_feedback === 'CONFIRMED' ? 'success' : 'info')">
            {{ currentEvent.patient_feedback === 'OBJECTED' ? '已提出异议申诉' : (currentEvent.patient_feedback === 'CONFIRMED' ? '已确认无异议 (正常抢救)' : '等待患者核验') }}
          </el-tag>
        </p>

        <!-- 第二道关卡：当存在异议或存疑时展示医患陈述争议焦点对比卡 -->
        <div v-if="currentEvent.audit_comparison_summary" class="ai-dispute-box mt-3 mb-3">
          <div class="ai-dispute-header">
            <el-icon color="#0284c7"><DocumentChecked /></el-icon>
            <span class="font-bold text-slate-900 ml-1">医患陈述争议焦点对比卡 (辅助审查参考)</span>
          </div>
          <pre class="ai-dispute-content font-mono">{{ currentEvent.audit_comparison_summary }}</pre>
        </div>

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
    <el-dialog v-model="liftDialogVisible" title="解除医生权限限制（复权审批）" width="min(540px, 92vw)">
      <el-alert
        title="复权机制说明"
        type="info"
        description="当受限医生整改期满或申诉复核通过后，监管部门可解除其访问控制限制，将其账号状态由受限恢复为正常，并在区块链上记存复权审计日志。"
        show-icon
        :closable="false"
        class="mb-3"
      />
      <el-form :model="liftForm" label-width="110px">
        <el-form-item label="选择医生" required>
          <el-select
            v-if="!selectedFromRow"
            v-model="liftForm.doctor_id"
            :placeholder="restrictedDoctorList.length ? '请选择需要解除限制的受限医生' : '当前暂无受限或停用医生'"
            style="width: 100%;"
            filterable
            no-data-text="当前暂无受限医生"
          >
            <el-option
              v-for="doc in restrictedDoctorList"
              :key="doc.id"
              :label="`${doc.real_name || doc.username} (${doc.hospital_name || '医院'} · ${formatDoctorStatus(doc.status)})`"
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
import { ref, computed, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { DocumentChecked } from '@element-plus/icons-vue'
import {
  listEmergencyEventsApi,
  auditEmergencyEventApi,
  listDoctorsApi,
  liftDoctorRestrictionApi
} from '../../api'

const events = ref<any[]>([])
const statusFilter = ref<'ALL' | 'PENDING' | 'OBJECTED' | 'CLOSED'>('ALL')
const loading = ref(false)

const pendingAuditCount = computed(() =>
  events.value.filter(e => !e.audit_status || e.audit_status === 'PENDING' || e.audit_status === 'UNDER_INVESTIGATION').length
)
const objectedCount = computed(() =>
  events.value.filter(e => e.patient_feedback === 'OBJECTED').length
)
const closedCount = computed(() =>
  events.value.filter(e => e.audit_status === 'CLOSED_APPROVED' || e.audit_status === 'CLOSED_VIOLATION').length
)
const filteredEvents = computed(() => {
  if (statusFilter.value === 'PENDING') {
    return events.value.filter(e => !e.audit_status || e.audit_status === 'PENDING' || e.audit_status === 'UNDER_INVESTIGATION')
  }
  if (statusFilter.value === 'OBJECTED') {
    return events.value.filter(e => e.patient_feedback === 'OBJECTED')
  }
  if (statusFilter.value === 'CLOSED') {
    return events.value.filter(e => e.audit_status === 'CLOSED_APPROVED' || e.audit_status === 'CLOSED_VIOLATION')
  }
  return events.value
})
const dialogVisible = ref(false)
const currentEvent = ref<any>(null)
const submitting = ref(false)

const form = ref({
  audit_status: 'CLOSED_APPROVED',
  punishment: 'RESTRICTED',
  audit_comment: '',
})

onMounted(() => {
  loadEvents()
})

async function loadEvents() {
  loading.value = true
  try {
    const res: any = await listEmergencyEventsApi()
    if (res.code === 200) {
      events.value = res.data
    }
  } finally {
    loading.value = false
  }
}

function getCleanDoctorDesc(desc?: string): string {
  if (!desc) return '（医生未填写补充说明）'
  return desc.split(/\|\s*\[(?:患者反馈|患者异议|患者陈述)[:：]?/)[0].trim() || '（医生未填写补充说明）'
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
  return '（患者未提交文字说明）'
}

function openAuditModal(row: any) {
  currentEvent.value = row
  // 每次打开审查弹窗均强制重置表单，严禁上一条工单的违规判定与评语残留导致误惩
  form.value = {
    audit_status: row.patient_feedback === 'OBJECTED' ? 'UNDER_INVESTIGATION' : 'CLOSED_APPROVED',
    punishment: 'RESTRICTED',
    audit_comment: '',
  }
  dialogVisible.value = true
}

async function submitAudit() {
  if (!currentEvent.value) return
  if (!form.value.audit_comment.trim()) {
    ElMessage.warning('请填写监管处置意见，该结论将固化上链留档')
    return
  }

  if (form.value.audit_status === 'CLOSED_VIOLATION') {
    const punishText = form.value.punishment === 'DISABLED' ? '停用封禁医生账号 (DISABLED)' : '限制跨院与急救破窗权限 (RESTRICTED)'
    try {
      await ElMessageBox.confirm(
        `确认对事件 ${currentEvent.value.event_no} 作出「违规处罚」裁决，并对医生 ${currentEvent.value.doctor_name || ''} 执行【${punishText}】吗？该裁决将永久写入区块链存证。`,
        '高危惩戒裁决二次确认',
        {
          confirmButtonText: '确认处罚并上链',
          cancelButtonText: '取消核对',
          type: 'warning',
        }
      )
    } catch {
      return
    }
  }

  submitting.value = true
  try {
    const res: any = await auditEmergencyEventApi(currentEvent.value.event_no, {
      audit_status: form.value.audit_status,
      audit_comment: form.value.audit_comment.trim(),
      punishment: form.value.audit_status === 'CLOSED_VIOLATION' ? form.value.punishment : 'NORMAL',
    })
    if (res.code === 200) {
      ElMessage.success(
        form.value.audit_status === 'CLOSED_VIOLATION'
          ? '违规处罚裁决已提交并在 Fabric 链上固化，惩戒措施即刻生效！'
          : '监管审核结论已成功提交并在 Fabric 链上固化！'
      )
      dialogVisible.value = false
      loadEvents()
    } else {
      ElMessage.error(res?.message || '审核提交失败')
    }
  } catch (err: any) {
    if (!err?.__toastShown) {
      ElMessage.error(err?.message || '审核提交失败')
    }
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
    const res: any = await listDoctorsApi()
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

const restrictedDoctorList = computed(() =>
  doctorList.value.filter(d => d.status === 'RESTRICTED' || d.status === 'DISABLED')
)

function formatDoctorStatus(st?: string): string {
  if (st === 'RESTRICTED') return '跨院受限'
  if (st === 'DISABLED') return '已停用'
  return '正常'
}

async function openGeneralLiftModal() {
  selectedFromRow.value = false
  await loadDoctorList()
  liftForm.value.doctor_id = restrictedDoctorList.value[0]?.id
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
    const res: any = await liftDoctorRestrictionApi({
      doctor_id: liftForm.value.doctor_id!,
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
.ai-dispute-box {
  background: #fdf4ff;
  border: 1px solid #e9d5ff;
  border-radius: 8px;
  padding: 12px 14px;
}
.ai-dispute-header {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 13px;
  margin-bottom: 8px;
}
.ai-dispute-content {
  background: #ffffff;
  border: 1px solid #f3e8ff;
  border-radius: 6px;
  padding: 10px 12px;
  font-size: 12px;
  line-height: 1.6;
  color: #334155;
  white-space: pre-wrap;
  word-break: break-word;
  margin: 0;
}
</style>
