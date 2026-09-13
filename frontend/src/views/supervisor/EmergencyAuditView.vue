<template>
  <div class="page-container">
    <div class="page-header">
      <div>
        <h2 class="page-title">🚨 紧急访问 (Break-Glass) 监管审核台</h2>
        <p class="page-sub">监管人员为唯一的医疗合规审判者。审查急救原因、患者知情异议反馈，执行合规结案或阶梯式违规惩戒</p>
      </div>
    </div>

    <el-card shadow="hover" class="box-card">
      <el-table :data="events" v-loading="loading" stripe style="width: 100%">
        <el-table-column prop="event_no" label="事件流水单号" width="160" />
        <el-table-column prop="doctor_name" label="调阅医生" width="120" />
        <el-table-column prop="source_hospital_name" label="调阅方医院" width="150" />
        <el-table-column prop="patient_name" label="涉及患者" width="100" />
        <el-table-column prop="emergency_reason" label="急救原因" width="110">
          <template #default="{ row }">
            <el-tag size="small" type="danger">{{ row.emergency_reason }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="patient_feedback" label="患者知情确认" width="130">
          <template #default="{ row }">
            <el-tag v-if="row.patient_feedback === 'CONFIRMED'" type="success" size="small">无异议</el-tag>
            <el-tag v-else-if="row.patient_feedback === 'OBJECTED'" type="danger" size="small">⚠️ 提出异议</el-tag>
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
        <el-table-column label="审核判定" width="120" fixed="right">
          <template #default="{ row }">
            <el-button
              type="primary"
              size="small"
              @click="openAuditModal(row)"
            >
              审查裁决
            </el-button>
          </template>
        </el-table-column>
      </el-table>
    </el-card>

    <!-- 监管裁决对话框 -->
    <el-dialog v-model="dialogVisible" title="紧急调阅合规审查与裁决" width="580px">
      <div v-if="currentEvent" class="audit-info-box">
        <p><strong>事件单号：</strong>{{ currentEvent.event_no }}</p>
        <p><strong>调阅医生：</strong>{{ currentEvent.doctor_name }} ({{ currentEvent.source_hospital_name }})</p>
        <p><strong>就诊患者：</strong>{{ currentEvent.patient_name }}</p>
        <p><strong>医生抢救说明：</strong>{{ currentEvent.description }}</p>
        <p><strong>患者反馈结论：</strong>
          <el-tag size="small" :type="currentEvent.patient_feedback === 'OBJECTED' ? 'danger' : 'success'">
            {{ currentEvent.patient_feedback }}
          </el-tag>
        </p>
        <el-divider />
      </div>

      <el-form :model="form" label-width="110px">
        <el-form-item label="合规审查判定" required>
          <el-select v-model="form.audit_status" style="width: 100%;">
            <el-option label="CLOSED_APPROVED - 真实急救调阅，予以合规正常结案" value="CLOSED_APPROVED" />
            <el-option label="UNDER_INVESTIGATION - 存在疑问，要求补充临床病历佐证" value="UNDER_INVESTIGATION" />
            <el-option label="CLOSED_VIOLATION - 判定属于虚假救治违规套取，予以处罚" value="CLOSED_VIOLATION" />
          </el-select>
        </el-form-item>
        <el-form-item label="阶梯惩戒措施" v-if="form.audit_status === 'CLOSED_VIOLATION'" required>
          <el-select v-model="form.punishment" style="width: 100%;">
            <el-option label="RESTRICTED - 限制该医生账号跨院访问及紧急访问权限" value="RESTRICTED" />
            <el-option label="DISABLED - 情节极其恶劣，彻底停用封禁医生账号" value="DISABLED" />
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
</script>

<style scoped>
.page-container {
  max-width: 1300px;
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
.audit-info-box p {
  line-height: 1.8;
  font-size: 14px;
}
</style>
