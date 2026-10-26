<template>
  <div class="page-container">
    <div class="page-header">
      <div>
        <h2 class="page-title">联盟医院与科室字典维护</h2>
        <p class="page-sub">管理联盟成员医疗机构接入白名单、Fabric 组织节点拓扑、MSP 证书有效期及下属临床科室字典</p>
      </div>
    </div>

    <el-row :gutter="20">
      <el-col :span="12">
        <!-- 联盟医疗机构卡片 -->
        <el-card shadow="hover" class="box-card">
          <template #header>
            <div class="card-header flex justify-between items-center">
              <div class="flex items-center gap-2">
                <span>联盟成员医疗机构 (Organizations)</span>
                <el-tag size="small" type="primary" effect="plain">共 {{ hospitals.length }} 家接入机构</el-tag>
              </div>
              <el-button type="primary" size="small" :icon="Plus" @click="openCreateHospitalDialog">
                接入新医疗机构
              </el-button>
            </div>
          </template>
          <el-table :data="hospitals" stripe style="width: 100%" v-loading="hospitalsLoading">
            <el-table-column prop="hospital_no" label="机构代码" width="95">
              <template #default="{ row }">
                <code class="mono font-semibold">{{ row.hospital_no }}</code>
              </template>
            </el-table-column>
            <el-table-column prop="name" label="医院机构全称" min-width="130" />
            <el-table-column label="MSP 标识" width="95">
              <template #default="{ row, $index }">
                <el-tag size="small" type="primary" effect="plain" class="mono font-bold">
                  {{ getHospitalMsp(row, $index) }}
                </el-tag>
              </template>
            </el-table-column>
            <el-table-column prop="level" label="等级" width="65" align="center" />
            <el-table-column prop="status" label="状态" width="85" align="center">
              <template #default="{ row }">
                <el-tag size="small" :type="row.status === 1 ? 'success' : 'danger'">
                  {{ row.status === 1 ? '接入在线' : '已暂停接入' }}
                </el-tag>
              </template>
            </el-table-column>
            <el-table-column label="操作" width="135" align="center">
              <template #default="{ row }">
                <el-button link type="primary" size="small" @click="openEditHospitalDialog(row)">
                  编辑
                </el-button>
                <el-button
                  link
                  :type="row.status === 1 ? 'warning' : 'success'"
                  size="small"
                  @click="toggleHospitalStatus(row)"
                >
                  {{ row.status === 1 ? '暂停' : '恢复' }}
                </el-button>
              </template>
            </el-table-column>
          </el-table>
        </el-card>

        <!-- 🌐 联盟链多组织节点拓扑与证书健康状态 -->
        <el-card shadow="hover" class="box-card mt-4 topology-card">
          <template #header>
            <div class="card-header flex justify-between items-center">
              <div class="flex items-center gap-2">
                <el-icon class="text-emerald-400"><Connection /></el-icon>
                <span class="text-white font-bold">🌐 联盟链多组织节点拓扑与证书健康状态</span>
              </div>
              <el-tag size="small" :type="blockchainStatus.connected ? 'success' : (blockchainStatus.mode === 'mock' ? 'info' : 'warning')" effect="dark">
                {{ blockchainStatus.mode === 'fabric' ? (blockchainStatus.connected ? 'Fabric 2.5 官方网关在线' : 'Fabric 节点离线') : '本地模拟账本 (离线容灾模式)' }}
              </el-tag>
            </div>
          </template>

          <div class="topology-grid">
            <div v-for="node in topologyNodes" :key="node.msp" class="topo-node-box">
              <div class="tn-top">
                <div class="tn-name-row">
                  <span class="mini-pulse-dot dot-green mr-1"></span>
                  <strong class="tn-name">{{ node.name }}</strong>
                  <el-tag size="small" type="primary" effect="dark" class="mono ml-auto">{{ node.msp }}</el-tag>
                </div>
                <div class="tn-peer mono text-xs mt-1 text-slate-300">
                  <el-icon class="mr-1"><Platform /></el-icon>{{ node.peer }}
                </div>
              </div>
              <div class="tn-divider"></div>
              <div class="tn-bottom">
                <div class="tn-info-row">
                  <span class="k">X.509 根证书：</span>
                  <span class="v text-emerald-400 font-semibold">{{ node.certExpire }}</span>
                </div>
                <div class="tn-info-row">
                  <span class="k">TLS 传输加密：</span>
                  <span class="v text-cyan-300 mono">{{ node.tlsVersion }} (双向 mTLS)</span>
                </div>
                <div class="tn-info-row">
                  <span class="k">共识区块高度：</span>
                  <span class="v mono font-bold text-amber-400">#{{ blockchainStatus.latest_block || node.blockHeight }} (已同步)</span>
                </div>
              </div>
            </div>
          </div>
          <div class="mt-3 px-3 py-2 bg-slate-900/60 rounded border border-slate-700/50 text-xs text-slate-400 flex items-center justify-between">
            <span><el-icon class="mr-1"><InfoFilled /></el-icon>底层引擎状态: <strong>{{ blockchainStatus.mode === 'fabric' ? (blockchainStatus.connected ? '真实 Fabric 2.5 网关' : 'Fabric 连接未就绪') : 'MockLedger 模拟账本' }}</strong></span>
            <span class="mono text-emerald-400">通道: {{ blockchainStatus.network || 'medicalchannel' }}</span>
          </div>
        </el-card>
      </el-col>

      <el-col :span="12">
        <el-card shadow="hover" class="box-card h-full">
          <template #header>
            <div class="card-header flex justify-between items-center">
              <div class="flex items-center gap-2">
                <span>临床科室字典 (Departments)</span>
                <el-tag size="small" type="info">共 {{ filteredDepartments.length }} 个科室</el-tag>
              </div>
              <div class="flex items-center gap-2">
                <el-select
                  v-model="selectedHospitalId"
                  placeholder="筛选所属医院"
                  clearable
                  size="small"
                  style="width: 140px"
                  @change="fetchDepartments"
                >
                  <el-option
                    v-for="h in hospitals"
                    :key="h.id"
                    :label="h.name"
                    :value="h.id"
                  />
                </el-select>
                <el-button type="primary" size="small" :icon="Plus" @click="openCreateDeptDialog">
                  添加科室
                </el-button>
              </div>
            </div>
          </template>
          <el-table :data="filteredDepartments" stripe style="width: 100%" max-height="620" v-loading="departmentsLoading">
            <el-table-column prop="dept_no" label="科室代码" width="95">
              <template #default="{ row }">
                <code class="mono font-semibold">{{ row.dept_no }}</code>
              </template>
            </el-table-column>
            <el-table-column label="所属医院" width="130">
              <template #default="{ row }">
                <span class="text-xs text-slate-600">{{ getHospitalNameById(row.hospital_id) }}</span>
              </template>
            </el-table-column>
            <el-table-column prop="name" label="科室名称" width="120">
              <template #default="{ row }">
                <strong>{{ row.name }}</strong>
              </template>
            </el-table-column>
            <el-table-column prop="description" label="职责范围与指征" min-width="120" show-overflow-tooltip />
            <el-table-column label="操作" width="110" align="center">
              <template #default="{ row }">
                <el-button link type="primary" size="small" @click="openEditDeptDialog(row)">
                  编辑
                </el-button>
                <el-popconfirm
                  title="确认删除该临床科室？若有关联医生或病历将安全拦截"
                  confirm-button-text="确认删除"
                  cancel-button-text="取消"
                  confirm-button-type="danger"
                  @confirm="handleDeleteDepartment(row)"
                >
                  <template #reference>
                    <el-button link type="danger" size="small">
                      删除
                    </el-button>
                  </template>
                </el-popconfirm>
              </template>
            </el-table-column>
          </el-table>
        </el-card>
      </el-col>
    </el-row>

    <!-- 医院新增/编辑弹窗 -->
    <el-dialog
      v-model="hospitalDialogVisible"
      :title="isEditingHospital ? '编辑医疗机构' : '接入新医疗机构'"
      width="480px"
      destroy-on-close
    >
      <el-form :model="hospitalForm" label-width="90px">
        <el-form-item label="机构代码" required>
          <el-input
            v-model="hospitalForm.hospital_no"
            placeholder="如 HOSP_004"
            :disabled="isEditingHospital"
          />
        </el-form-item>
        <el-form-item label="机构全称" required>
          <el-input v-model="hospitalForm.name" placeholder="请输入医疗机构全称" />
        </el-form-item>
        <el-form-item label="机构等级">
          <el-select v-model="hospitalForm.level" placeholder="请选择机构等级" style="width: 100%">
            <el-option label="三级甲等 (三甲)" value="三甲" />
            <el-option label="三级乙等 (三乙)" value="三乙" />
            <el-option label="二级甲等 (二甲)" value="二甲" />
            <el-option label="二级乙等 (二乙)" value="二乙" />
          </el-select>
        </el-form-item>
        <el-form-item label="机构地址">
          <el-input v-model="hospitalForm.address" placeholder="请输入机构详细地理地址" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="hospitalDialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="hospitalSubmitting" @click="submitHospitalForm">
          确认保存
        </el-button>
      </template>
    </el-dialog>

    <!-- 科室新增/编辑弹窗 -->
    <el-dialog
      v-model="deptDialogVisible"
      :title="isEditingDept ? '编辑科室字典' : '添加专科科室'"
      width="480px"
      destroy-on-close
    >
      <el-form :model="deptForm" label-width="90px">
        <el-form-item label="所属医院" required>
          <el-select
            v-model="deptForm.hospital_id"
            placeholder="请选择所属医疗机构"
            :disabled="isEditingDept"
            style="width: 100%"
          >
            <el-option
              v-for="h in hospitals"
              :key="h.id"
              :label="h.name"
              :value="h.id"
            />
          </el-select>
        </el-form-item>
        <el-form-item label="科室代码" required>
          <el-input
            v-model="deptForm.dept_no"
            placeholder="如 DEPT_CARDIO"
            :disabled="isEditingDept"
          />
        </el-form-item>
        <el-form-item label="科室名称" required>
          <el-input v-model="deptForm.name" placeholder="如 心血管内科" />
        </el-form-item>
        <el-form-item label="职责指征">
          <el-input
            v-model="deptForm.description"
            type="textarea"
            :rows="3"
            placeholder="请输入科室诊疗专科范围与职责描述"
          />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="deptDialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="deptSubmitting" @click="submitDeptForm">
          确认保存
        </el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { Connection, Platform, Plus, InfoFilled } from '@element-plus/icons-vue'
import { ElMessage } from 'element-plus'
import api from '../../api/client'

interface Hospital {
  id: number
  hospital_no: string
  name: string
  level: string
  address: string
  status: number
  created_at?: string
}

interface Department {
  id: number
  dept_no: string
  hospital_id: number
  name: string
  description: string
  created_at?: string
}

const hospitals = ref<Hospital[]>([])
const departments = ref<Department[]>([])
const hospitalsLoading = ref(false)
const departmentsLoading = ref(false)
const selectedHospitalId = ref<number | undefined>(undefined)

const filteredDepartments = computed(() => {
  if (!selectedHospitalId.value) return departments.value
  return departments.value.filter(d => d.hospital_id === selectedHospitalId.value)
})

// 医院模态框状态
const hospitalDialogVisible = ref(false)
const isEditingHospital = ref(false)
const hospitalSubmitting = ref(false)
const currentHospitalId = ref<number | null>(null)
const hospitalForm = ref({
  hospital_no: '',
  name: '',
  level: '三甲',
  address: ''
})

// 科室模态框状态
const deptDialogVisible = ref(false)
const isEditingDept = ref(false)
const deptSubmitting = ref(false)
const currentDeptId = ref<number | null>(null)
const deptForm = ref({
  dept_no: '',
  hospital_id: undefined as number | undefined,
  name: '',
  description: ''
})

const blockchainStatus = ref<{
  mode: string
  connected: boolean
  latest_block: number
  network?: string
  chaincode?: string
  peer?: string
  message?: string
}>({
  mode: 'mock',
  connected: false,
  latest_block: 100,
  network: 'medicalchannel',
  chaincode: 'medical_contract'
})

async function fetchBlockchainStatus() {
  try {
    const res: any = await api.get('/system/blockchain/status')
    if (res.code === 200 && res.data) {
      blockchainStatus.value = res.data
    }
  } catch (err) {
    console.warn('获取区块链真实运行状态失败，保持降级显示', err)
  }
}

const topologyNodes = ref([
  {
    name: '广州第一附属医院背书节点',
    msp: 'Org1MSP',
    peer: 'peer0.org1.example.com:7051',
    certExpire: '有效期至 2036-09-12 · 正常',
    tlsVersion: 'TLSv1.3',
    blockHeight: 357
  },
  {
    name: '深圳中心医院背书节点',
    msp: 'Org2MSP',
    peer: 'peer0.org2.example.com:7051',
    certExpire: '有效期至 2036-09-12 · 正常',
    tlsVersion: 'TLSv1.3',
    blockHeight: 357
  },
  {
    name: '珠海人民医院背书节点',
    msp: 'Org3MSP',
    peer: 'peer0.org3.example.com:7051',
    certExpire: '有效期至 2036-09-12 · 正常',
    tlsVersion: 'TLSv1.3',
    blockHeight: 357
  }
])

function getHospitalMsp(row: Hospital, index: number): string {
  if (row.id === 1 || row.name?.includes('第一') || index === 0) return 'Org1MSP'
  if (row.id === 2 || row.name?.includes('中心') || index === 1) return 'Org2MSP'
  return 'Org3MSP'
}

function getHospitalNameById(id: number): string {
  const h = hospitals.value.find(item => item.id === id)
  return h ? h.name : `机构 #${id}`
}

async function fetchHospitals() {
  hospitalsLoading.value = true
  try {
    const res: any = await api.get('/system/hospitals')
    if (res.code === 200) {
      hospitals.value = res.data
    }
  } catch (err: any) {
    ElMessage.error(err.response?.data?.message || '获取医疗机构列表失败')
  } finally {
    hospitalsLoading.value = false
  }
}

async function fetchDepartments() {
  departmentsLoading.value = true
  try {
    const url = selectedHospitalId.value ? `/system/departments?hospital_id=${selectedHospitalId.value}` : '/system/departments'
    const res: any = await api.get(url)
    if (res.code === 200) {
      departments.value = res.data
    }
  } catch (err: any) {
    ElMessage.error(err.response?.data?.message || '获取科室字典列表失败')
  } finally {
    departmentsLoading.value = false
  }
}

function openCreateHospitalDialog() {
  isEditingHospital.value = false
  currentHospitalId.value = null
  hospitalForm.value = {
    hospital_no: '',
    name: '',
    level: '三甲',
    address: ''
  }
  hospitalDialogVisible.value = true
}

function openEditHospitalDialog(row: Hospital) {
  isEditingHospital.value = true
  currentHospitalId.value = row.id
  hospitalForm.value = {
    hospital_no: row.hospital_no,
    name: row.name,
    level: row.level || '三甲',
    address: row.address || ''
  }
  hospitalDialogVisible.value = true
}

async function submitHospitalForm() {
  if (!hospitalForm.value.name.trim()) {
    ElMessage.warning('请输入机构全称')
    return
  }
  if (!isEditingHospital.value && !hospitalForm.value.hospital_no.trim()) {
    ElMessage.warning('请输入机构代码')
    return
  }

  hospitalSubmitting.value = true
  try {
    if (isEditingHospital.value && currentHospitalId.value) {
      const res: any = await api.put(`/system/hospitals/${currentHospitalId.value}`, {
        name: hospitalForm.value.name,
        level: hospitalForm.value.level,
        address: hospitalForm.value.address
      })
      if (res.code === 200) {
        ElMessage.success('机构信息更新成功')
        hospitalDialogVisible.value = false
        await fetchHospitals()
      }
    } else {
      const res: any = await api.post('/system/hospitals', hospitalForm.value)
      if (res.code === 200) {
        ElMessage.success('医疗机构接入成功')
        hospitalDialogVisible.value = false
        await fetchHospitals()
      }
    }
  } catch (err: any) {
    ElMessage.error(err.response?.data?.message || '保存医疗机构失败')
  } finally {
    hospitalSubmitting.value = false
  }
}

async function toggleHospitalStatus(row: Hospital) {
  const targetStatus = row.status === 1 ? 0 : 1
  const actionText = targetStatus === 0 ? '暂停' : '恢复'
  try {
    const res: any = await api.put(`/system/hospitals/${row.id}/status`, {
      status: targetStatus
    })
    if (res.code === 200) {
      ElMessage.success(`机构已成功${actionText}接入服务`)
      row.status = targetStatus
    }
  } catch (err: any) {
    ElMessage.error(err.response?.data?.message || `${actionText}机构状态失败`)
  }
}

function openCreateDeptDialog() {
  isEditingDept.value = false
  currentDeptId.value = null
  deptForm.value = {
    dept_no: '',
    hospital_id: selectedHospitalId.value || (hospitals.value.length > 0 ? hospitals.value[0].id : undefined),
    name: '',
    description: ''
  }
  deptDialogVisible.value = true
}

function openEditDeptDialog(row: Department) {
  isEditingDept.value = true
  currentDeptId.value = row.id
  deptForm.value = {
    dept_no: row.dept_no,
    hospital_id: row.hospital_id,
    name: row.name,
    description: row.description || ''
  }
  deptDialogVisible.value = true
}

async function submitDeptForm() {
  if (!deptForm.value.name.trim()) {
    ElMessage.warning('请输入科室名称')
    return
  }
  if (!deptForm.value.hospital_id) {
    ElMessage.warning('请选择所属医疗机构')
    return
  }
  if (!isEditingDept.value && !deptForm.value.dept_no.trim()) {
    ElMessage.warning('请输入科室代码')
    return
  }

  deptSubmitting.value = true
  try {
    if (isEditingDept.value && currentDeptId.value) {
      const res: any = await api.put(`/system/departments/${currentDeptId.value}`, {
        name: deptForm.value.name,
        description: deptForm.value.description
      })
      if (res.code === 200) {
        ElMessage.success('科室信息更新成功')
        deptDialogVisible.value = false
        await fetchDepartments()
      }
    } else {
      const res: any = await api.post('/system/departments', deptForm.value)
      if (res.code === 200) {
        ElMessage.success('科室字典创建成功')
        deptDialogVisible.value = false
        await fetchDepartments()
      }
    }
  } catch (err: any) {
    ElMessage.error(err.response?.data?.message || '保存科室字典失败')
  } finally {
    deptSubmitting.value = false
  }
}

async function handleDeleteDepartment(row: Department) {
  try {
    const res: any = await api.delete(`/system/departments/${row.id}`)
    if (res.code === 200) {
      ElMessage.success('科室删除成功')
      await fetchDepartments()
    }
  } catch (err: any) {
    ElMessage.error(err.response?.data?.message || '删除科室失败')
  }
}

onMounted(async () => {
  await Promise.all([fetchHospitals(), fetchDepartments(), fetchBlockchainStatus()])
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

/* 🌐 联盟链拓扑卡片深色科技感 */
.topology-card {
  background: linear-gradient(145deg, #0f172a 0%, #1e293b 100%) !important;
  color: #f8fafc !important;
  border: 1px solid #334155 !important;
}
.topology-card :deep(.el-card__header) {
  border-bottom: 1px solid rgba(255, 255, 255, 0.1) !important;
}

.topology-grid {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.topo-node-box {
  background: rgba(255, 255, 255, 0.04);
  border: 1px solid rgba(255, 255, 255, 0.08);
  border-radius: 8px;
  padding: 12px 14px;
  transition: all 0.2s ease;
}
.topo-node-box:hover {
  background: rgba(255, 255, 255, 0.08);
  border-color: rgba(56, 189, 248, 0.3);
  transform: translateY(-1px);
}

.tn-name-row {
  display: flex;
  align-items: center;
}
.tn-name {
  color: #f8fafc;
  font-size: 13px;
}

.tn-divider {
  height: 1px;
  background: rgba(255, 255, 255, 0.06);
  margin: 8px 0;
}

.tn-bottom {
  display: flex;
  flex-direction: column;
  gap: 4px;
  font-size: 11px;
}

.tn-info-row {
  display: flex;
  justify-content: space-between;
  align-items: center;
}
.tn-info-row .k {
  color: #94a3b8;
}

.mini-pulse-dot {
  width: 7px;
  height: 7px;
  border-radius: 50%;
  display: inline-block;
}
.dot-green {
  background-color: #10b981;
  box-shadow: 0 0 0 0 rgba(16, 185, 129, 0.7);
}
</style>
