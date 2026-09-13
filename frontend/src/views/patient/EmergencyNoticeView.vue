<template>
  <div class="page-container">
    <div class="page-header">
      <div>
        <h2 class="page-title">🚨 紧急访问知情与异议申诉中心</h2>
        <p class="page-sub">当医生在未事先取得授权的情况下触发 Break-Glass 抢救调阅您的病历时，系统在此实时知情通知，支持确认或提出异议</p>
      </div>
    </div>

    <el-card shadow="hover" class="box-card">
      <div v-if="events.length">
        <div
          v-for="ev in events"
          :key="ev.id"
          class="event-card"
          :class="ev.patient_feedback === 'PENDING' ? 'pending-border' : ''"
        >
          <div class="event-header">
            <span class="event-no">事件流水单号: <strong>{{ ev.event_no }}</strong></span>
            <el-tag :type="ev.patient_feedback === 'PENDING' ? 'danger' : 'info'" effect="dark">
              {{ ev.patient_feedback === 'PENDING' ? '待您知情确认' : (ev.patient_feedback === 'CONFIRMED' ? '已确认无异议' : '已提出异议') }}
            </el-tag>
          </div>

          <div class="event-body">
            <p><strong>调阅医生：</strong>{{ ev.doctor_name }} ({{ ev.source_hospital_name }})</p>
            <p><strong>调阅病历编号：</strong>{{ ev.record_no }}</p>
            <p><strong>紧急原因：</strong><el-tag size="small" type="danger">{{ ev.emergency_reason }}</el-tag></p>
            <p><strong>医生急救临床说明：</strong>{{ ev.description }}</p>
            <p><strong>上链存证 TxID：</strong><code class="tx-hash">{{ ev.fabric_tx_id }}</code></p>
            <p><strong>调阅时间：</strong>{{ ev.created_at ? ev.created_at.substring(0, 16).replace('T', ' ') : '' }}</p>
          </div>

          <div v-if="ev.patient_feedback === 'PENDING'" class="event-actions">
            <el-button type="success" @click="handleFeedback(ev.event_no, 'CONFIRMED')">确认属于正常抢救 (无异议)</el-button>
            <el-button type="danger" plain @click="openObjectionModal(ev)">对本次调阅提出异议投诉</el-button>
          </div>
        </div>
      </div>
      <el-empty v-else description="暂无紧急访问事件通知，您的医疗数据安全在控" />
    </el-card>

    <el-dialog v-model="dialogVisible" title="提出紧急调阅异议投诉" width="500px">
      <el-form label-width="80px">
        <el-form-item label="异议理由" required>
          <el-input v-model="comment" type="textarea" :rows="3" placeholder="请阐明异议理由（如：当时我意识清醒，并未发生急重抢救，涉嫌虚假理由调阅）..." />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="danger" @click="submitObjection">提交监管审核</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { ElMessage } from 'element-plus'
import api from '../../api/client'

const events = ref<any[]>([])
const dialogVisible = ref(false)
const targetEventNo = ref('')
const comment = ref('当时我意识完全清醒，医生未经我同意擅自调阅，涉嫌违规套取隐私。')

onMounted(() => {
  loadEvents()
})

async function loadEvents() {
  try {
    const res: any = await api.get('/supervisor/emergency-events')
    if (res.code === 200) {
      events.value = res.data
    }
  } catch (err) {
    console.error(err)
  }
}

async function handleFeedback(eventNo: string, feedback: string, cmt: string = '') {
  try {
    const res: any = await api.post('/access/break-glass/patient-feedback', {
      event_no: eventNo,
      feedback: feedback,
      comment: cmt,
    })
    if (res.code === 200) {
      ElMessage.success('反馈已成功提交至监管后台')
      loadEvents()
    }
  } catch (err: any) {
    ElMessage.error(err?.message || '反馈提交失败')
  }
}

function openObjectionModal(ev: any) {
  targetEventNo.value = ev.event_no
  dialogVisible.value = true
}

function submitObjection() {
  dialogVisible.value = false
  handleFeedback(targetEventNo.value, 'OBJECTED', comment.value)
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
</style>
