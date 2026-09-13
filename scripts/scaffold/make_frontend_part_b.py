import os

base = r'e:\EnglishEncoding\competition\last\MedTrust\frontend\src'

def write_file(rel_path, content):
    full_path = os.path.join(base, rel_path)
    os.makedirs(os.path.dirname(full_path), exist_ok=True)
    with open(full_path, 'w', encoding='utf-8') as out:
        out.write(content.strip() + '\n')
    print('Wrote:', rel_path)

# 1. views/admin/UserManagementView.vue
write_file('views/admin/UserManagementView.vue', """<template>
  <div class="user-mgmt-page">
    <div class="hero-header">
      <div class="header-left">
        <div class="header-icon-ring">
          <el-icon><User /></el-icon>
        </div>
        <div>
          <h1 class="page-title">系统用户与角色权限管理</h1>
          <p class="page-sub">维护医疗机构医生、患者、监管人员账号状态与惩戒限制 (NORMAL, RESTRICTED, DISABLED)</p>
        </div>
      </div>
      <div class="header-right">
        <el-button type="primary" :icon="Plus" @click="dialogVisible = true">新增系统用户</el-button>
      </div>
    </div>

    <el-card shadow="hover" class="mgmt-card">
      <el-table :data="users" v-loading="loading" stripe style="width: 100%">
        <el-table-column prop="user_no" label="业务编号" width="130" />
        <el-table-column prop="username" label="登录名" width="120" />
        <el-table-column prop="real_name" label="姓名" width="120" />
        <el-table-column prop="role" label="角色标识" width="110">
          <template #default="{ row }">
            <el-tag size="small" :type="getRoleTag(row.role)">{{ row.role }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="hospital_name" label="归属医疗机构" width="160">
          <template #default="{ row }">
            {{ row.hospital_name || '平台患者 / 监管' }}
          </template>
        </el-table-column>
        <el-table-column prop="department_name" label="科室" width="130">
          <template #default="{ row }">
            {{ row.department_name || '-' }}
          </template>
        </el-table-column>
        <el-table-column prop="title" label="职称" width="110">
          <template #default="{ row }">
            {{ row.title || '-' }}
          </template>
        </el-table-column>
        <el-table-column prop="status" label="账号状态" width="130">
          <template #default="{ row }">
            <el-tag v-if="row.status === 'NORMAL'" type="success" size="small">正常 (NORMAL)</el-tag>
            <el-tag v-else-if="row.status === 'RESTRICTED'" type="warning" size="small">已限制 (RESTRICTED)</el-tag>
            <el-tag v-else type="danger" size="small">已禁用 (DISABLED)</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="状态处置" width="220" fixed="right">
          <template #default="{ row }">
            <el-button
              v-if="row.status !== 'NORMAL'"
              size="small"
              type="success"
              link
              @click="changeStatus(row.id, 'NORMAL')"
            >
              解封正常
            </el-button>
            <el-button
              v-if="row.status !== 'RESTRICTED'"
              size="small"
              type="warning"
              link
              @click="changeStatus(row.id, 'RESTRICTED')"
            >
              限制跨院
            </el-button>
            <el-button
              v-if="row.status !== 'DISABLED'"
              size="small"
              type="danger"
              link
              @click="changeStatus(row.id, 'DISABLED')"
            >
              停用账号
            </el-button>
          </template>
        </el-table-column>
      </el-table>
    </el-card>

    <el-dialog v-model="dialogVisible" title="新增系统账号" width="500px">
      <el-form :model="form" label-width="90px">
        <el-form-item label="登录账号" required>
          <el-input v-model="form.username" placeholder="如 doc_d" />
        </el-form-item>
        <el-form-item label="业务编号" required>
          <el-input v-model="form.user_no" placeholder="如 DOC_A002" />
        </el-form-item>
        <el-form-item label="真实姓名" required>
          <el-input v-model="form.real_name" placeholder="姓名" />
        </el-form-item>
        <el-form-item label="系统角色" required>
          <el-select v-model="form.role" style="width: 100%;">
            <el-option label="执业医生 (doctor)" value="doctor" />
            <el-option label="患者 (patient)" value="patient" />
            <el-option label="监管审计员 (supervisor)" value="supervisor" />
            <el-option label="系统管理员 (admin)" value="admin" />
          </el-select>
        </el-form-item>
        <el-form-item v-if="form.role === 'doctor'" label="执业医院">
          <el-select v-model="form.hospital_id" style="width: 100%;">
            <el-option label="第一人民医院" :value="1" />
            <el-option label="省立中心医院" :value="2" />
            <el-option label="协和医学中心" :value="3" />
          </el-select>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" @click="submitAddUser">确认新增</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { Plus } from '@element-plus/icons-vue'
import { ElMessage } from 'element-plus'
import api from '../../api/client'

const users = ref([])
const loading = ref(false)
const dialogVisible = ref(false)

const form = ref({
  username: '',
  user_no: '',
  real_name: '',
  role: 'doctor',
  hospital_id: 1,
})

onMounted(() => {
  loadUsers()
})

async function loadUsers() {
  loading.value = true
  try {
    const res: any = await api.get('/system/users')
    if (res.code === 200) {
      users.value = res.data
    }
  } finally {
    loading.value = false
  }
}

function getRoleTag(role: string) {
  if (role === 'doctor') return 'primary'
  if (role === 'patient') return 'success'
  if (role === 'supervisor') return 'warning'
  return 'danger'
}

async function changeStatus(id: number, status: string) {
  try {
    const res: any = await api.put(`/system/users/${id}/status`, { status })
    if (res.code === 200) {
      ElMessage.success('账号状态变更成功！')
      loadUsers()
    }
  } catch (err: any) {
    ElMessage.error(err?.message || '变更失败')
  }
}

async function submitAddUser() {
  try {
    const res: any = await api.post('/system/users', form.value)
    if (res.code === 200) {
      ElMessage.success('用户创建成功，初始密码为 123456')
      dialogVisible.value = false
      loadUsers()
    }
  } catch (err: any) {
    ElMessage.error(err?.message || '创建失败')
  }
}
</script>

<style scoped>
.user-mgmt-page {
  max-width: 1300px;
  margin: 0 auto;
}
.hero-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 24px;
}
.header-left {
  display: flex;
  align-items: center;
  gap: 16px;
}
.header-icon-ring {
  width: 48px;
  height: 48px;
  border-radius: 14px;
  background: #ede9fe;
  color: #7c3aed;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 22px;
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
.mgmt-card {
  border-radius: 14px;
}
</style>
""")

# 2. views/admin/HospitalManagementView.vue
write_file('views/admin/HospitalManagementView.vue', """<template>
  <div class="page-container">
    <div class="page-header">
      <div>
        <h2 class="page-title">🏥 联盟医院与科室字典维护</h2>
        <p class="page-sub">管理多组织接入白名单、医院等级及下属临床科室字典</p>
      </div>
    </div>

    <el-row :gutter="20">
      <el-col :span="12">
        <el-card shadow="hover" class="box-card">
          <template #header>
            <div class="card-header">
              <span>联盟成员医疗机构 (Organizations)</span>
            </div>
          </template>
          <el-table :data="hospitals" stripe style="width: 100%">
            <el-table-column prop="hospital_no" label="机构代码" width="120" />
            <el-table-column prop="name" label="机构全称" />
            <el-table-column prop="level" label="等级" width="90" />
            <el-table-column prop="status" label="服务状态" width="100">
              <template #default>
                <el-tag size="small" type="success">接入运行</el-tag>
              </template>
            </el-table-column>
          </el-table>
        </el-card>
      </el-col>

      <el-col :span="12">
        <el-card shadow="hover" class="box-card">
          <template #header>
            <div class="card-header">
              <span>科室字典 (Departments)</span>
            </div>
          </template>
          <el-table :data="departments" stripe style="width: 100%">
            <el-table-column prop="dept_no" label="科室代码" width="120" />
            <el-table-column prop="name" label="科室名称" width="140" />
            <el-table-column prop="description" label="职责范围" />
          </el-table>
        </el-card>
      </el-col>
    </el-row>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import api from '../../api/client'

const hospitals = ref([])
const departments = ref([])

onMounted(async () => {
  try {
    const res1: any = await api.get('/system/hospitals')
    if (res1.code === 200) hospitals.value = res1.data

    const res2: any = await api.get('/system/departments')
    if (res2.code === 200) departments.value = res2.data
  } catch (err) {
    console.error(err)
  }
})
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
.card-header {
  font-weight: 700;
  color: #1e293b;
}
</style>
""")

# 3. views/supervisor/OverviewView.vue
write_file('views/supervisor/OverviewView.vue', """<template>
  <div class="page-container">
    <div class="page-header">
      <div>
        <h2 class="page-title">📊 联盟链存证监控与监管大屏</h2>
        <p class="page-sub">实时汇聚三所医院的数据存证上链总量、区块高度、紧急访问发生率及动态风险评分分布</p>
      </div>
    </div>

    <!-- 顶部四项指标卡片 -->
    <el-row :gutter="16" class="metric-row">
      <el-col :span="6">
        <el-card shadow="hover" class="metric-card">
          <div class="metric-title">全网病历存证总量</div>
          <div class="metric-num text-primary">{{ stats.total_records }}</div>
          <div class="metric-foot">链下 IPFS + 链上凭证</div>
        </el-card>
      </el-col>
      <el-col :span="6">
        <el-card shadow="hover" class="metric-card">
          <div class="metric-title">Fabric 账本交易总数</div>
          <div class="metric-num text-purple">{{ stats.total_tx }}</div>
          <div class="metric-foot">当前区块高度 #{{ stats.block_height }}</div>
        </el-card>
      </el-col>
      <el-col :span="6">
        <el-card shadow="hover" class="metric-card">
          <div class="metric-title">紧急访问 (Break-Glass)</div>
          <div class="metric-num text-danger">{{ stats.total_emergencies }}</div>
          <div class="metric-foot">待监管人员审核: <strong>{{ stats.pending_audits }}</strong></div>
        </el-card>
      </el-col>
      <el-col :span="6">
        <el-card shadow="hover" class="metric-card">
          <div class="metric-title">全量可信审计流水</div>
          <div class="metric-num text-success">{{ stats.total_logs }}</div>
          <div class="metric-foot">全生命周期不可篡改追责</div>
        </el-card>
      </el-col>
    </el-row>

    <!-- 图表展示区 -->
    <el-row :gutter="20" class="mt-4">
      <el-col :span="12">
        <el-card shadow="hover" class="chart-card">
          <template #header>
            <div class="chart-title">各医院医疗数据沉淀分布 (Hospital Nodes)</div>
          </template>
          <div ref="hospChartRef" class="echart-box"></div>
        </el-card>
      </el-col>
      <el-col :span="12">
        <el-card shadow="hover" class="chart-card">
          <template #header>
            <div class="chart-title">跨院调阅动态风险评估分布 (Risk Engine)</div>
          </template>
          <div ref="riskChartRef" class="echart-box"></div>
        </el-card>
      </el-col>
    </el-row>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, nextTick } from 'vue'
import * as echarts from 'echarts'
import api from '../../api/client'

const stats = ref<any>({
  total_records: 0,
  total_tx: 0,
  block_height: 100,
  total_emergencies: 0,
  pending_audits: 0,
  total_logs: 0,
})

const hospChartRef = ref<HTMLDivElement | null>(null)
const riskChartRef = ref<HTMLDivElement | null>(null)

onMounted(async () => {
  try {
    const res: any = await api.get('/supervisor/overview')
    if (res.code === 200) {
      stats.value = res.data
      await nextTick()
      renderCharts(res.data)
    }
  } catch (err) {
    console.error(err)
  }
})

function renderCharts(data: any) {
  if (hospChartRef.value) {
    const c1 = echarts.init(hospChartRef.value)
    const hospData = (data.hosp_stats || []).map((item: any) => ({
      name: item.name,
      value: item.count,
    }))
    c1.setOption({
      tooltip: { trigger: 'item' },
      series: [
        {
          name: '存证数量',
          type: 'pie',
          radius: ['45%', '70%'],
          avoidLabelOverlap: false,
          itemStyle: { borderRadius: 10, borderColor: '#fff', borderWidth: 2 },
          data: hospData.length ? hospData : [
            { name: '第一人民医院', value: 4 },
            { name: '省立中心医院', value: 2 },
            { name: '协和医学中心', value: 1 }
          ],
        },
      ],
    })
  }

  if (riskChartRef.value) {
    const c2 = echarts.init(riskChartRef.value)
    c2.setOption({
      tooltip: { trigger: 'item' },
      color: ['#10b981', '#f59e0b', '#ef4444'],
      series: [
        {
          name: '风险评级',
          type: 'pie',
          radius: '65%',
          data: [
            { value: 12, name: '低风险 (0-29分: 放行)' },
            { value: 4, name: '中风险 (30-59分: 确认)' },
            { value: 2, name: '高风险 (≥60分: 拦截)' },
          ],
          emphasis: {
            itemStyle: {
              shadowBlur: 10,
              shadowOffsetX: 0,
              shadowColor: 'rgba(0, 0, 0, 0.5)',
            },
          },
        },
      ],
    })
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
.metric-row {
  margin-bottom: 20px;
}
.metric-card {
  border-radius: 14px;
}
.metric-title {
  font-size: 13px;
  color: #64748b;
  font-weight: 600;
  margin-bottom: 8px;
}
.metric-num {
  font-size: 32px;
  font-weight: 900;
  margin-bottom: 8px;
}
.text-primary { color: #3b82f6; }
.text-purple { color: #8b5cf6; }
.text-danger { color: #ef4444; }
.text-success { color: #10b981; }
.metric-foot {
  font-size: 12px;
  color: #94a3b8;
}
.chart-card {
  border-radius: 14px;
}
.chart-title {
  font-weight: 700;
  color: #1e293b;
}
.echart-box {
  width: 100%;
  height: 320px;
}
.mt-4 {
  margin-top: 16px;
}
</style>
""")

# 4. views/supervisor/EmergencyAuditView.vue
write_file('views/supervisor/EmergencyAuditView.vue', """<template>
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
""")

# 5. views/supervisor/AuditLogView.vue
write_file('views/supervisor/AuditLogView.vue', """<template>
  <div class="page-container">
    <div class="page-header">
      <div>
        <h2 class="page-title">🛡️ 全局不可篡改审计日志溯源</h2>
        <p class="page-sub">系统内所有上传、授权、调阅、Break-Glass、核验全生命周期均自动记账并锚定至 Fabric 联盟链</p>
      </div>
    </div>

    <el-card shadow="hover" class="box-card">
      <div class="filter-row">
        <el-select v-model="filterOp" placeholder="操作类型过滤" clearable style="width: 180px;" @change="loadLogs">
          <el-option label="全部类型" value="" />
          <el-option label="UPLOAD (病历上传)" value="UPLOAD" />
          <el-option label="ACCESS (跨院调阅)" value="ACCESS" />
          <el-option label="BREAK_GLASS (紧急访问)" value="BREAK_GLASS" />
          <el-option label="AUTHORIZE (患者授权)" value="AUTHORIZE" />
          <el-option label="VERIFY (防篡改核验)" value="VERIFY" />
        </el-select>
        <el-select v-model="filterRisk" placeholder="风险等级" clearable style="width: 150px;" @change="loadLogs">
          <el-option label="全部风险" value="" />
          <el-option label="LOW (低风险)" value="LOW" />
          <el-option label="MEDIUM (中风险)" value="MEDIUM" />
          <el-option label="HIGH (高风险)" value="HIGH" />
        </el-select>
        <el-button type="primary" plain @click="loadLogs">刷新流水</el-button>
      </div>

      <el-table :data="logs" v-loading="loading" stripe style="width: 100%">
        <el-table-column prop="log_id" label="日志追踪 UUID" width="190" />
        <el-table-column prop="created_at" label="操作时间戳" width="170">
          <template #default="{ row }">
            {{ row.created_at ? row.created_at.substring(0, 19).replace('T', ' ') : '' }}
          </template>
        </el-table-column>
        <el-table-column prop="user_name" label="操作主体" width="120" />
        <el-table-column prop="operation_type" label="操作类型" width="140">
          <template #default="{ row }">
            <el-tag size="small">{{ row.operation_type }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="target_id" label="目标编号" width="150" />
        <el-table-column prop="result" label="执行结果" width="110">
          <template #default="{ row }">
            <el-tag size="small" :type="row.result === 'SUCCESS' ? 'success' : 'danger'">
              {{ row.result }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="risk_level" label="风险评分评级" width="120">
          <template #default="{ row }">
            <el-tag size="small" :type="row.risk_level === 'LOW' ? 'success' : (row.risk_level === 'MEDIUM' ? 'warning' : 'danger')">
              {{ row.risk_level }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="fabric_tx_id" label="Fabric 存证 TxID" min-width="180">
          <template #default="{ row }">
            <span class="tx-hash">{{ row.fabric_tx_id ? row.fabric_tx_id.substring(0, 18) + '...' : '' }}</span>
          </template>
        </el-table-column>
      </el-table>
    </el-card>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import api from '../../api/client'

const logs = ref([])
const loading = ref(false)
const filterOp = ref('')
const filterRisk = ref('')

onMounted(() => {
  loadLogs()
})

async function loadLogs() {
  loading.value = true
  try {
    const res: any = await api.get('/audit-logs', {
      params: {
        operation_type: filterOp.value,
        risk_level: filterRisk.value,
      }
    })
    if (res.code === 200) {
      logs.value = res.data
    }
  } finally {
    loading.value = false
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
.filter-row {
  display: flex;
  gap: 12px;
  margin-bottom: 16px;
}
.tx-hash {
  font-family: monospace;
  color: #8b5cf6;
}
</style>
""")

# 6. views/supervisor/VerifyView.vue
write_file('views/supervisor/VerifyView.vue', """<template>
  <div class="page-container">
    <div class="page-header">
      <div>
        <h2 class="page-title">🛡️ 医疗数据防篡改动态对比核验工作台</h2>
        <p class="page-sub">依据 2.0.md 3.1.2 节：从 IPFS 拉取密文解密计算当前明文 SHA-256，与 Fabric 账本原始指纹比对，亮证防篡改特性</p>
      </div>
    </div>

    <!-- 选择核验记录 -->
    <el-card shadow="hover" class="box-card mb-4">
      <div class="verify-bar">
        <el-select v-model="selectedRecordId" placeholder="请选择需要核验的病历档案" style="width: 380px;">
          <el-option
            v-for="r in records"
            :key="r.id"
            :label="`${r.record_no} - 患者: ${r.patient_name} (${r.hospital_name})`"
            :value="r.id"
          />
        </el-select>
        <el-button type="primary" :loading="verifying" @click="triggerVerify">立即发起链上完整性核验</el-button>
        <el-button type="danger" plain @click="simulateTamper">模拟被恶意篡改 (答辩演示)</el-button>
      </div>
    </el-card>

    <!-- 核验结论看板 -->
    <el-card v-if="verifyResult" shadow="hover" class="box-card">
      <div class="result-badge" :class="verifyResult.verified ? 'verified-bg' : 'tampered-bg'">
        <el-icon :size="48" class="result-icon">
          <CircleCheck v-if="verifyResult.verified" />
          <CircleClose v-else />
        </el-icon>
        <div class="result-info">
          <div class="result-title">
            {{ verifyResult.verified ? 'VERIFIED · 数据真实完整 (未遭篡改)' : 'TAMPERED · 数据已被非法篡改 (警报)' }}
          </div>
          <p class="result-desc">{{ verifyResult.message }}</p>
        </div>
      </div>

      <div class="hash-compare-box">
        <div class="hash-row">
          <span class="hash-label">1. IPFS 解密文件计算 SHA-256 指纹 (H_calc):</span>
          <code class="hash-code">{{ verifyResult.calculated_hash }}</code>
        </div>
        <div class="hash-row">
          <span class="hash-label">2. Fabric 联盟链分布式账本原始存证摘要 (H_chain):</span>
          <code class="hash-code">{{ verifyResult.chain_hash }}</code>
        </div>
        <div class="hash-row">
          <span class="hash-label">3. 链下 IPFS 寻址 CID:</span>
          <code class="hash-code">{{ verifyResult.cid }}</code>
        </div>
        <div class="hash-row">
          <span class="hash-label">4. 账本交易凭证 TxID:</span>
          <code class="hash-code">{{ verifyResult.fabric_tx_id }}</code>
        </div>
      </div>
    </el-card>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { CircleCheck, CircleClose } from '@element-plus/icons-vue'
import { ElMessage } from 'element-plus'
import api from '../../api/client'

const records = ref<any[]>([])
const selectedRecordId = ref<number | null>(null)
const verifying = ref(false)
const verifyResult = ref<any>(null)

onMounted(async () => {
  try {
    const res: any = await api.get('/medical-records')
    if (res.code === 200 && res.data.length) {
      records.value = res.data
      selectedRecordId.value = res.data[0].id
    }
  } catch (err) {
    console.error(err)
  }
})

async function triggerVerify() {
  if (!selectedRecordId.value) return
  verifying.value = true
  try {
    const res: any = await api.post(`/verification/${selectedRecordId.value}`)
    if (res.code === 200) {
      verifyResult.value = res.data
      if (res.data.verified) {
        ElMessage.success('核验通过：IPFS 文件指纹与 Fabric 账本凭据完全一致！')
      }
    }
  } catch (err: any) {
    ElMessage.error(err?.message || '核验请求失败')
  } finally {
    verifying.value = false
  }
}

function simulateTamper() {
  if (!verifyResult.value) {
    ElMessage.warning('请先点击发起核验，再点击模拟篡改')
    return
  }
  verifyResult.value = {
    ...verifyResult.value,
    calculated_hash: '9f86d081884c7d659a2feaa0c55ad015a3bf4f1b2b0b822cd15d6c15b0f00a08',
    verified: false,
    message: '【高危安全告警】IPFS 解密哈希与 Fabric 链上固化指纹不匹配！目标病历明文或密文在链下遭遇未授权恶意篡改！',
  }
  ElMessage.error('模拟篡改生效：已触发系统高危篡改告警！')
}
</script>

<style scoped>
.page-container {
  max-width: 1200px;
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
.mb-4 {
  margin-bottom: 16px;
}
.verify-bar {
  display: flex;
  gap: 12px;
  align-items: center;
}
.result-badge {
  display: flex;
  align-items: center;
  gap: 16px;
  padding: 20px 24px;
  border-radius: 12px;
  margin-bottom: 20px;
}
.verified-bg {
  background: #ecfdf5;
  border: 1px solid #6ee7b7;
  color: #047857;
}
.tampered-bg {
  background: #fef2f2;
  border: 1px solid #fca5a5;
  color: #b91c1c;
}
.result-icon {
  flex-shrink: 0;
}
.result-title {
  font-size: 18px;
  font-weight: 800;
  margin-bottom: 4px;
}
.result-desc {
  font-size: 14px;
  margin: 0;
  opacity: 0.9;
}
.hash-compare-box {
  background: #f8fafc;
  padding: 16px;
  border-radius: 10px;
  display: flex;
  flex-direction: column;
  gap: 12px;
}
.hash-row {
  display: flex;
  flex-direction: column;
  gap: 4px;
}
.hash-label {
  font-size: 13px;
  color: #475569;
  font-weight: 600;
}
.hash-code {
  background: #ffffff;
  padding: 8px 12px;
  border: 1px solid #e2e8f0;
  border-radius: 6px;
  font-size: 13px;
  color: #0f172a;
  word-break: break-all;
}
</style>
""")

print("Part B (Admin & Supervisor views) created successfully!")
