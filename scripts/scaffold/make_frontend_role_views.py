import os

base = r'e:\EnglishEncoding\competition\last\MedTrust\frontend\src'

def write_file(rel_path, content):
    full_path = os.path.join(base, rel_path)
    os.makedirs(os.path.dirname(full_path), exist_ok=True)
    with open(full_path, 'w', encoding='utf-8') as out:
        out.write(content.strip() + '\n')
    print('Wrote:', rel_path)

# 1. views/doctor/DoctorRecordsView.vue
write_file('views/doctor/DoctorRecordsView.vue', """<template>
  <div class="page-container">
    <div class="page-header">
      <div>
        <h2 class="page-title">🩺 门诊电子病历录入与区块链存证</h2>
        <p class="page-sub">医生录入患者诊断，附件自动执行 AES-256-GCM 链下加密与 IPFS 存储，并将明文 SHA-256 存证锚定至 Fabric 联盟链</p>
      </div>
      <el-button type="primary" :icon="Plus" @click="openUploadDialog">新增就诊病历上链</el-button>
    </div>

    <!-- 病历列表表格 -->
    <el-card shadow="hover" class="box-card">
      <div class="table-toolbar">
        <el-input
          v-model="searchKeyword"
          placeholder="搜索病历编号或诊断结论..."
          clearable
          style="width: 320px;"
          @clear="loadRecords"
          @keyup.enter="loadRecords"
        >
          <template #prefix><el-icon><Search /></el-icon></template>
        </el-input>
        <el-button type="primary" plain @click="loadRecords">查询</el-button>
      </div>

      <el-table :data="records" v-loading="loading" style="width: 100%" stripe>
        <el-table-column prop="record_no" label="病历编号" width="160" />
        <el-table-column prop="patient_name" label="就诊患者" width="100" />
        <el-table-column prop="hospital_name" label="开具机构" width="140" />
        <el-table-column prop="data_type" label="类型" width="90">
          <template #default="{ row }">
            <el-tag size="small" :type="row.data_type === 'EMR' ? 'primary' : 'warning'">{{ row.data_type }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="diagnosis" label="临床诊断结论" min-width="200" show-overflow-tooltip />
        <el-table-column prop="fabric_tx_id" label="Fabric 存证 TxID" min-width="180">
          <template #default="{ row }">
            <el-tooltip :content="row.fabric_tx_id" placement="top">
              <span class="tx-hash">{{ row.fabric_tx_id ? row.fabric_tx_id.substring(0, 18) + '...' : '未上链' }}</span>
            </el-tooltip>
          </template>
        </el-table-column>
        <el-table-column prop="block_height" label="区块高度" width="100">
          <template #default="{ row }">
            <el-tag size="small" type="success">#{{ row.block_height }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="created_at" label="就诊时间" width="160">
          <template #default="{ row }">
            {{ row.created_at ? row.created_at.substring(0, 16).replace('T', ' ') : '' }}
          </template>
        </el-table-column>
        <el-table-column label="操作" width="120" fixed="right">
          <template #default="{ row }">
            <el-button link type="primary" @click="viewDetail(row)">详情</el-button>
          </template>
        </el-table-column>
      </el-table>
    </el-card>

    <!-- 新增病历上链对话框 -->
    <el-dialog v-model="dialogVisible" title="新增就诊档案 (AES-256-GCM 链下加密 + 链上存证)" width="600px">
      <el-form :model="form" label-width="110px">
        <el-form-item label="就诊患者" required>
          <el-select v-model="form.patient_id" placeholder="选择患者" style="width: 100%;">
            <el-option label="张三 (ID: 4)" :value="4" />
            <el-option label="李四 (ID: 5)" :value="5" />
          </el-select>
        </el-form-item>
        <el-form-item label="数据类型" required>
          <el-radio-group v-model="form.data_type">
            <el-radio value="EMR">门诊病历 (EMR)</el-radio>
            <el-radio value="REPORT">检验检查单 (REPORT)</el-radio>
            <el-radio value="IMAGE">医学影像切片 (IMAGE)</el-radio>
          </el-radio-group>
        </el-form-item>
        <el-form-item label="临床诊断结论" required>
          <el-input v-model="form.diagnosis" type="textarea" :rows="3" placeholder="填写患者主诉、现病史、既往过敏史及诊断意见..." />
        </el-form-item>
        <el-form-item label="附件文件" required>
          <input type="file" ref="fileInput" @change="onFileSelected" />
          <div class="form-tip">系统将自动对原始明文计算 SHA-256 指纹并在内存执行对称加密后推送至 IPFS</div>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="uploading" @click="submitUpload">立即加密上链</el-button>
      </template>
    </el-dialog>

    <!-- 详情弹窗 -->
    <el-dialog v-model="detailVisible" title="病历与存证详情" width="550px">
      <div v-if="selectedRecord" class="detail-box">
        <p><strong>病历单号：</strong>{{ selectedRecord.record_no }}</p>
        <p><strong>就诊患者：</strong>{{ selectedRecord.patient_name }}</p>
        <p><strong>开具机构：</strong>{{ selectedRecord.hospital_name }}</p>
        <p><strong>开具医生：</strong>{{ selectedRecord.doctor_name }}</p>
        <p><strong>诊断意见：</strong>{{ selectedRecord.diagnosis }}</p>
        <el-divider />
        <p><strong>Fabric TxID：</strong><span class="tx-hash">{{ selectedRecord.fabric_tx_id }}</span></p>
        <p><strong>区块高度：</strong>#{{ selectedRecord.block_height }}</p>
        <div v-if="selectedRecord.files && selectedRecord.files.length">
          <p><strong>关联文件：</strong>{{ selectedRecord.files[0].file_name }} ({{ selectedRecord.files[0].file_type }})</p>
          <p><strong>IPFS CID：</strong><code>{{ selectedRecord.files[0].ipfs_cid }}</code></p>
          <p><strong>明文 SHA-256：</strong><code>{{ selectedRecord.files[0].file_hash }}</code></p>
        </div>
      </div>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { Plus, Search } from '@element-plus/icons-vue'
import { ElMessage } from 'element-plus'
import api from '../../api/client'
import { useAuthStore } from '../../stores/auth'

const auth = useAuthStore()
const records = ref([])
const loading = ref(false)
const searchKeyword = ref('')

const dialogVisible = ref(false)
const uploading = ref(false)
const detailVisible = ref(false)
const selectedRecord = ref<any>(null)
const selectedFile = ref<File | null>(null)
const fileInput = ref<HTMLInputElement | null>(null)

const form = ref({
  patient_id: 4,
  data_type: 'EMR',
  diagnosis: '',
})

onMounted(() => {
  loadRecords()
})

async function loadRecords() {
  loading.value = true
  try {
    const res: any = await api.get('/medical-records', {
      params: { keyword: searchKeyword.value }
    })
    if (res.code === 200) {
      records.value = res.data
    }
  } catch (err) {
    console.error(err)
  } finally {
    loading.value = false
  }
}

function openUploadDialog() {
  form.value.diagnosis = ''
  selectedFile.value = null
  dialogVisible.value = true
}

function onFileSelected(e: any) {
  if (e.target.files && e.target.files.length) {
    selectedFile.value = e.target.files[0]
  }
}

async function submitUpload() {
  if (!form.value.diagnosis) {
    ElMessage.warning('请填写诊断结论')
    return
  }
  if (!selectedFile.value) {
    ElMessage.warning('请选择需要加密上传的文件')
    return
  }

  uploading.value = true
  const formData = new FormData()
  formData.append('patient_id', String(form.value.patient_id))
  formData.append('data_type', form.value.data_type)
  formData.append('diagnosis', form.value.diagnosis)
  formData.append('file', selectedFile.value)

  try {
    const res: any = await api.post('/medical-records/upload', formData, {
      headers: { 'Content-Type': 'multipart/form-data' }
    })
    if (res.code === 200) {
      ElMessage.success('病历已完成 AES-256-GCM 链下加密并成功锚定至 Fabric 联盟链！')
      dialogVisible.value = false
      loadRecords()
    }
  } catch (err: any) {
    ElMessage.error(err?.message || '上链失败')
  } finally {
    uploading.value = false
  }
}

function viewDetail(row: any) {
  selectedRecord.value = row
  detailVisible.value = true
}
</script>

<style scoped>
.page-container {
  max-width: 1300px;
  margin: 0 auto;
}
.page-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 20px;
}
.page-title {
  font-size: 22px;
  font-weight: 800;
  color: #0f172a;
  margin-bottom: 4px;
}
.page-sub {
  font-size: 13px;
  color: #64748b;
}
.box-card {
  border-radius: 14px;
  border: 1px solid rgba(226, 232, 240, 0.8);
}
.table-toolbar {
  display: flex;
  gap: 12px;
  margin-bottom: 16px;
}
.tx-hash {
  font-family: monospace;
  color: #8b5cf6;
  font-weight: 600;
}
.form-tip {
  font-size: 12px;
  color: #94a3b8;
  margin-top: 4px;
}
.detail-box p {
  line-height: 2;
  font-size: 14px;
}
</style>
""")

# 2. views/doctor/CrossQueryView.vue
write_file('views/doctor/CrossQueryView.vue', """<template>
  <div class="page-container">
    <div class="page-header">
      <div>
        <h2 class="page-title">🔍 跨医疗机构病历检索与紧急访问 (Break-Glass)</h2>
        <p class="page-sub">医生检索其他医院病历时，需通过统一权限网关校验。无授权时阻断并提供【申请紧急访问】入口</p>
      </div>
    </div>

    <!-- 检索筛选区 -->
    <el-card shadow="hover" class="box-card mb-4">
      <div class="search-form">
        <el-select v-model="selectedHospital" placeholder="目标医疗机构" style="width: 200px;">
          <el-option label="所有机构" :value="0" />
          <el-option label="第一人民医院 (Hosp A)" :value="1" />
          <el-option label="省立中心医院 (Hosp B)" :value="2" />
          <el-option label="协和医学中心 (Hosp C)" :value="3" />
        </el-select>
        <el-input
          v-model="searchKeyword"
          placeholder="输入患者姓名、病历编号或诊断关键词检索..."
          style="width: 360px;"
          @keyup.enter="searchCrossRecords"
        />
        <el-button type="primary" :icon="Search" @click="searchCrossRecords">跨院检索</el-button>
      </div>
    </el-card>

    <!-- 跨院记录结果列表 -->
    <el-card shadow="hover" class="box-card">
      <el-table :data="results" v-loading="loading" stripe style="width: 100%">
        <el-table-column prop="record_no" label="病历单号" width="160" />
        <el-table-column prop="patient_name" label="就诊患者" width="100" />
        <el-table-column prop="hospital_name" label="归属医疗机构" width="160" />
        <el-table-column prop="doctor_name" label="开具医生" width="120" />
        <el-table-column prop="data_type" label="类别" width="100">
          <template #default="{ row }">
            <el-tag size="small">{{ row.data_type }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="created_at" label="就诊日期" width="160">
          <template #default="{ row }">
            {{ row.created_at ? row.created_at.substring(0, 10) : '' }}
          </template>
        </el-table-column>
        <el-table-column label="调阅操作" width="160" fixed="right">
          <template #default="{ row }">
            <el-button type="primary" size="small" @click="handleAccess(row)">申请调阅病历</el-button>
          </template>
        </el-table-column>
      </el-table>
    </el-card>

    <!-- 权限拦截与 Break-Glass 紧急申请入口对话框 -->
    <el-dialog v-model="breakGlassVisible" title="⚠️ 安全网关访问控制拦截" width="620px">
      <div class="intercept-notice">
        <el-alert
          title="未取得患者显式知情授权 (HTTP 403 Forbidden)"
          type="error"
          description="系统基于统一权限网关检测：目标病历属于其他医疗机构，且患者当前未对您建立有效的授权策略。"
          show-icon
          :closable="false"
        />
      </div>

      <div class="emergency-guide">
        <h4>🚨 是否属于急诊危重抢救场景？</h4>
        <p>
          根据《医疗数据可信共享实施规范》，若患者处于<strong>突发休克、严重昏迷或危及生命</strong>的极端急救场景，医生经身份核验并签署临床法律责任声明后，系统允许触发 <strong>Break-Glass 紧急访问机制</strong>。
        </p>
      </div>

      <el-form :model="bgForm" label-width="110px" class="bg-form">
        <el-form-item label="紧急原因" required>
          <el-select v-model="bgForm.emergency_reason" style="width: 100%;">
            <el-option label="COMA - 患者严重休克昏迷，无法表达意愿" value="COMA" />
            <el-option label="RESCUE - 急诊抢救生命关键期，急需用药与过敏史" value="RESCUE" />
            <el-option label="CRITICAL - 突发急性危重病综合救治" value="CRITICAL" />
            <el-option label="OTHER - 其他危及生命的紧急医学场景" value="OTHER" />
          </el-select>
        </el-form-item>
        <el-form-item label="临床急救说明" required>
          <el-input
            v-model="bgForm.description"
            type="textarea"
            :rows="3"
            placeholder="请详细录入急救诊断情况、抢救必要性说明，此说明将直接作为不可篡改证据固化上链..."
          />
        </el-form-item>
        <el-form-item>
          <el-checkbox v-model="bgForm.doctor_confirmed">
            <span class="confirm-text">我确认本次紧急调阅仅用于患者紧急临床救治，并承担相应法律责任</span>
          </el-checkbox>
        </el-form-item>
      </el-form>

      <template #footer>
        <el-button @click="breakGlassVisible = false">放弃调阅</el-button>
        <el-button
          type="danger"
          :disabled="!bgForm.doctor_confirmed || !bgForm.description"
          :loading="submittingBG"
          @click="submitBreakGlass"
        >
          确认申请 Break-Glass 抢救放行
        </el-button>
      </template>
    </el-dialog>

    <!-- 解密放行展示详情框 -->
    <el-dialog v-model="recordModalVisible" title="✅ 医疗数据已核准解密放行" width="650px">
      <div v-if="releasedRecord" class="released-box">
        <el-alert
          v-if="isBreakGlassRelease"
          title="⚠️ 本次调阅为 Break-Glass 紧急放行，事件单已写入联盟链，已同步通知患者与监管部门"
          type="warning"
          show-icon
          class="mb-3"
          :closable="false"
        />
        <div class="content-card">
          <p><strong>病历单号：</strong>{{ releasedRecord.record_no }}</p>
          <p><strong>就诊患者：</strong>{{ releasedRecord.patient_name }}</p>
          <p><strong>开具机构：</strong>{{ releasedRecord.hospital_name }}</p>
          <p><strong>开具医生：</strong>{{ releasedRecord.doctor_name }}</p>
          <p class="highlight-diag"><strong>临床诊断：</strong>{{ releasedRecord.diagnosis }}</p>
          <el-divider />
          <p><strong>Fabric TxID：</strong><code class="tx-hash">{{ releasedRecord.fabric_tx_id }}</code></p>
          <div v-if="releasedRecord.files && releasedRecord.files.length">
            <p><strong>IPFS 密文 CID：</strong><code>{{ releasedRecord.files[0].ipfs_cid }}</code></p>
            <p><strong>明文哈希：</strong><code>{{ releasedRecord.files[0].file_hash }}</code></p>
          </div>
        </div>
      </div>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { Search } from '@element-plus/icons-vue'
import { ElMessage, ElNotification } from 'element-plus'
import api from '../../api/client'
import { useAuthStore } from '../../stores/auth'

const auth = useAuthStore()
const selectedHospital = ref(0)
const searchKeyword = ref('')
const results = ref([])
const loading = ref(false)

const breakGlassVisible = ref(false)
const currentTarget = ref<any>(null)
const submittingBG = ref(false)

const recordModalVisible = ref(false)
const releasedRecord = ref<any>(null)
const isBreakGlassRelease = ref(false)

const bgForm = ref({
  emergency_reason: 'COMA',
  description: '患者严重休克昏迷送医，急需调阅既往严重药物过敏史及基础心脑血管病史。',
  doctor_confirmed: true,
})

onMounted(() => {
  searchCrossRecords()
})

async function searchCrossRecords() {
  loading.value = true
  try {
    const res: any = await api.get('/medical-records', {
      params: {
        hospital_id: selectedHospital.value || undefined,
        keyword: searchKeyword.value,
      }
    })
    if (res.code === 200) {
      results.value = res.data
    }
  } catch (err) {
    console.error(err)
  } finally {
    loading.value = false
  }
}

async function handleAccess(row: any) {
  currentTarget.value = row
  try {
    const res: any = await api.post('/access/requests', {
      patient_id: row.patient_id,
      record_id: row.id,
      purpose: '跨院协同门诊诊疗调阅',
    })

    if (res.data?.allowed) {
      ElMessage.success('授权校验通过，直接放行')
      isBreakGlassRelease.value = false
      releasedRecord.value = row
      recordModalVisible.value = true
    } else {
      // 弹出 Break-Glass 申请
      breakGlassVisible.value = true
    }
  } catch (err: any) {
    breakGlassVisible.value = true
  }
}

async function submitBreakGlass() {
  if (!currentTarget.value) return
  submittingBG.value = true
  try {
    const res: any = await api.post('/access/break-glass', {
      record_id: currentTarget.value.id,
      emergency_reason: bgForm.value.emergency_reason,
      description: bgForm.value.description,
      doctor_confirmed: bgForm.value.doctor_confirmed,
    })

    if (res.code === 200) {
      breakGlassVisible.value = false
      ElNotification({
        title: 'Break-Glass 紧急访问已核准放行',
        message: `事件单号: ${res.data.event.event_no} 已全量上链存证，已向患者与监管端推送通知。`,
        type: 'success',
      })
      isBreakGlassRelease.value = true
      releasedRecord.value = res.data.record
      recordModalVisible.value = true
    } else {
      ElMessage.error(res.message || '申请被拒绝')
    }
  } catch (err: any) {
    ElMessage.error(err?.message || '紧急访问申请失败')
  } finally {
    submittingBG.value = false
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
.search-form {
  display: flex;
  gap: 12px;
}
.mb-4 {
  margin-bottom: 16px;
}
.mb-3 {
  margin-bottom: 12px;
}
.intercept-notice {
  margin-bottom: 16px;
}
.emergency-guide {
  background: #fef2f2;
  border: 1px solid #fecaca;
  padding: 14px;
  border-radius: 10px;
  margin-bottom: 18px;
}
.emergency-guide h4 {
  color: #dc2626;
  margin-bottom: 6px;
}
.emergency-guide p {
  font-size: 13px;
  color: #7f1d1d;
  line-height: 1.5;
}
.confirm-text {
  font-weight: 700;
  color: #dc2626;
}
.tx-hash {
  font-family: monospace;
  color: #8b5cf6;
}
.content-card p {
  line-height: 2;
  font-size: 14px;
}
.highlight-diag {
  background: #ede9fe;
  padding: 8px 12px;
  border-radius: 8px;
  color: #6d28d9;
}
</style>
""")

# 3. views/patient/PatientRecordsView.vue
write_file('views/patient/PatientRecordsView.vue', """<template>
  <div class="page-container">
    <div class="page-header">
      <div>
        <h2 class="page-title">📁 我的电子健康档案</h2>
        <p class="page-sub">查看您在联盟链医疗机构建立的所有门诊病历、检验单与影像数据</p>
      </div>
    </div>

    <el-card shadow="hover" class="box-card">
      <el-timeline v-if="records.length">
        <el-timeline-item
          v-for="rec in records"
          :key="rec.id"
          :timestamp="rec.created_at ? rec.created_at.substring(0, 16).replace('T', ' ') : ''"
          placement="top"
          type="primary"
        >
          <el-card class="timeline-card">
            <div class="card-head">
              <span class="rec-title">{{ rec.hospital_name }} · {{ rec.doctor_name }} ({{ rec.data_type }})</span>
              <el-tag size="small" type="success">区块 #{{ rec.block_height }}</el-tag>
            </div>
            <p class="diag-text"><strong>诊断结论：</strong>{{ rec.diagnosis }}</p>
            <div class="meta-row">
              <span>单号: <code>{{ rec.record_no }}</code></span>
              <span>存证 TxID: <code class="tx-hash">{{ rec.fabric_tx_id }}</code></span>
            </div>
          </el-card>
        </el-timeline-item>
      </el-timeline>
      <el-empty v-else description="暂无就诊记录" />
    </el-card>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import api from '../../api/client'

const records = ref([])

onMounted(async () => {
  try {
    const res: any = await api.get('/medical-records')
    if (res.code === 200) {
      records.value = res.data
    }
  } catch (err) {
    console.error(err)
  }
})
</script>

<style scoped>
.page-container {
  max-width: 1000px;
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
.timeline-card {
  border-radius: 12px;
}
.card-head {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 10px;
}
.rec-title {
  font-size: 15px;
  font-weight: 700;
  color: #1e293b;
}
.diag-text {
  font-size: 14px;
  color: #334155;
  margin-bottom: 12px;
}
.meta-row {
  display: flex;
  gap: 20px;
  font-size: 12px;
  color: #64748b;
}
.tx-hash {
  color: #8b5cf6;
}
</style>
""")

# 4. views/patient/AuthManagerView.vue
write_file('views/patient/AuthManagerView.vue', """<template>
  <div class="page-container">
    <div class="page-header">
      <div>
        <h2 class="page-title">🔑 患者自主知情授权策略管理</h2>
        <p class="page-sub">由患者完全自主控制数据共享权限，支持按医生或医院授权，生效即上链，支持一键撤销</p>
      </div>
      <el-button type="primary" :icon="Plus" @click="dialogVisible = true">新增授权策略</el-button>
    </div>

    <el-card shadow="hover" class="box-card">
      <el-table :data="authorizations" v-loading="loading" stripe style="width: 100%">
        <el-table-column prop="auth_no" label="授权流水号" width="160" />
        <el-table-column prop="auth_target_type" label="授权对象类型" width="120" />
        <el-table-column prop="target_name" label="被授权方" width="180" />
        <el-table-column prop="scope_type" label="授权范围" width="100">
          <template #default="{ row }">
            <el-tag size="small" :type="row.scope_type === 'ALL' ? 'success' : 'info'">{{ row.scope_type }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="end_time" label="失效时间" width="160">
          <template #default="{ row }">
            {{ row.end_time ? row.end_time.substring(0, 10) : '' }}
          </template>
        </el-table-column>
        <el-table-column prop="status" label="状态" width="100">
          <template #default="{ row }">
            <el-tag size="small" :type="row.status === 'ACTIVE' ? 'success' : 'danger'">
              {{ row.status === 'ACTIVE' ? '生效中' : '已撤销' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="fabric_tx_id" label="上链 TxID" min-width="180">
          <template #default="{ row }">
            <span class="tx-hash">{{ row.fabric_tx_id ? row.fabric_tx_id.substring(0, 18) + '...' : '' }}</span>
          </template>
        </el-table-column>
        <el-table-column label="操作" width="100" fixed="right">
          <template #default="{ row }">
            <el-popconfirm
              v-if="row.status === 'ACTIVE'"
              title="确定立即撤销此授权？撤销行为将记录至区块链。"
              @confirm="revokeAuth(row.id)"
            >
              <template #reference>
                <el-button link type="danger">撤销授权</el-button>
              </template>
            </el-popconfirm>
            <span v-else class="text-muted">已作废</span>
          </template>
        </el-table-column>
      </el-table>
    </el-card>

    <el-dialog v-model="dialogVisible" title="新增数据访问授权" width="500px">
      <el-form :model="form" label-width="100px">
        <el-form-item label="授权目标" required>
          <el-select v-model="form.auth_target_type" style="width: 100%;">
            <el-option label="指定医生 (DOCTOR)" value="DOCTOR" />
            <el-option label="指定医院机构 (HOSPITAL)" value="HOSPITAL" />
          </el-select>
        </el-form-item>
        <el-form-item label="被授权对象" required>
          <el-select v-if="form.auth_target_type === 'DOCTOR'" v-model="form.auth_target_id" style="width: 100%;">
            <el-option label="李建国医生 (第一人民医院)" :value="1" />
            <el-option label="王明德医生 (省立中心医院)" :value="2" />
            <el-option label="陈晓华医生 (协和医学中心)" :value="3" />
          </el-select>
          <el-select v-else v-model="form.auth_target_id" style="width: 100%;">
            <el-option label="第一人民医院 (HOSP_A)" :value="1" />
            <el-option label="省立中心医院 (HOSP_B)" :value="2" />
            <el-option label="协和医学中心 (HOSP_C)" :value="3" />
          </el-select>
        </el-form-item>
        <el-form-item label="授权范围" required>
          <el-radio-group v-model="form.scope_type">
            <el-radio value="ALL">全部历史病历</el-radio>
            <el-radio value="SINGLE">指定特定单次就诊</el-radio>
          </el-radio-group>
        </el-form-item>
        <el-form-item label="有效期限(天)">
          <el-input-number v-model="form.days" :min="1" :max="365" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="submitting" @click="submitCreate">确认授权上链</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { Plus } from '@element-plus/icons-vue'
import { ElMessage } from 'element-plus'
import api from '../../api/client'

const authorizations = ref([])
const loading = ref(false)
const dialogVisible = ref(false)
const submitting = ref(false)

const form = ref({
  auth_target_type: 'DOCTOR',
  auth_target_id: 1,
  scope_type: 'ALL',
  days: 30,
})

onMounted(() => {
  loadAuths()
})

async function loadAuths() {
  loading.value = true
  try {
    const res: any = await api.get('/authorizations')
    if (res.code === 200) {
      authorizations.value = res.data
    }
  } finally {
    loading.value = false
  }
}

async function submitCreate() {
  submitting.value = true
  try {
    const res: any = await api.post('/authorizations', {
      auth_target_type: form.value.auth_target_type,
      auth_target_id: form.value.auth_target_id,
      scope_type: form.value.scope_type,
    })
    if (res.code === 200) {
      ElMessage.success('授权策略已成功固化至联盟链！')
      dialogVisible.value = false
      loadAuths()
    }
  } catch (err: any) {
    ElMessage.error(err?.message || '授权创建失败')
  } finally {
    submitting.value = false
  }
}

async function revokeAuth(id: number) {
  try {
    const res: any = await api.delete(`/authorizations/${id}`)
    if (res.code === 200) {
      ElMessage.success('授权已成功撤销')
      loadAuths()
    }
  } catch (err: any) {
    ElMessage.error(err?.message || '撤销失败')
  }
}
</script>

<style scoped>
.page-container {
  max-width: 1200px;
  margin: 0 auto;
}
.page-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
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
.tx-hash {
  font-family: monospace;
  color: #8b5cf6;
}
.text-muted {
  color: #94a3b8;
  font-size: 12px;
}
</style>
""")

# 5. views/patient/EmergencyNoticeView.vue
write_file('views/patient/EmergencyNoticeView.vue', """<template>
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
""")

print("Part A (Doctor & Patient views) created successfully!")
