<template>
  <div class="user-mgmt-page">
    <!-- 顶部状态统计看板 -->
    <div class="stats-grid mb-4">
      <div class="stat-card blue">
        <div class="stat-icon"><el-icon><User /></el-icon></div>
        <div class="stat-info">
          <div class="stat-value">{{ users.length }}</div>
          <div class="stat-label">全部注册用户</div>
        </div>
      </div>
      <div class="stat-card cyan">
        <div class="stat-icon"><el-icon><FirstAidKit /></el-icon></div>
        <div class="stat-info">
          <div class="stat-value">{{ doctorCount }}</div>
          <div class="stat-label">临床执业医生</div>
        </div>
      </div>
      <div class="stat-card green">
        <div class="stat-icon"><el-icon><UserFilled /></el-icon></div>
        <div class="stat-info">
          <div class="stat-value">{{ patientCount }}</div>
          <div class="stat-label">就诊患者主体</div>
        </div>
      </div>
      <div class="stat-card purple">
        <div class="stat-icon"><el-icon><Management /></el-icon></div>
        <div class="stat-info">
          <div class="stat-value">{{ supervisorCount + adminCount }}</div>
          <div class="stat-label">监管审计与管理员</div>
        </div>
      </div>
    </div>

    <!-- 标题与新增卡片栏 -->
    <div class="hero-header">
      <div class="header-left">
        <div class="header-icon-ring">
          <el-icon><User /></el-icon>
        </div>
        <div>
          <h1 class="page-title">系统全域用户与角色权限管理</h1>
          <p class="page-sub">面向平台全体用户（医生、患者、监管审计员、管理员）的全量身份核验、详细电子档案、账号状态处置与资质管控</p>
        </div>
      </div>
      <div class="header-right">
        <el-button type="primary" size="large" :icon="Plus" @click="openCreateDialog">
          新增系统用户
        </el-button>
        <el-button :icon="Refresh" circle size="large" @click="loadUsers" title="刷新列表" />
      </div>
    </div>

    <el-card shadow="never" class="mgmt-card">
      <!-- 角色分类标签页与搜索工具条 -->
      <div class="toolbar-wrapper">
        <el-radio-group v-model="activeRoleTab" size="default" class="role-tabs-group">
          <el-radio-button value="ALL">全部人员 ({{ users.length }})</el-radio-button>
          <el-radio-button value="doctor">医生群体 ({{ doctorCount }})</el-radio-button>
          <el-radio-button value="patient">就诊患者 ({{ patientCount }})</el-radio-button>
          <el-radio-button value="supervisor">监管审计员 ({{ supervisorCount }})</el-radio-button>
          <el-radio-button value="admin">系统管理员 ({{ adminCount }})</el-radio-button>
        </el-radio-group>

        <div class="filter-controls">
          <el-select v-model="statusFilter" placeholder="账号状态" style="width: 140px;" clearable>
            <el-option label="全部状态" value="" />
            <el-option label="正常 (NORMAL)" value="NORMAL" />
            <el-option label="已限制 (RESTRICTED)" value="RESTRICTED" />
            <el-option label="已停用 (DISABLED)" value="DISABLED" />
          </el-select>

          <el-input
            v-model="searchKeyword"
            placeholder="搜索姓名、登录名、手机、身份证或卡号..."
            style="width: 320px;"
            clearable
            :prefix-icon="Search"
          />
        </div>
      </div>

      <!-- 用户主表格 -->
      <el-table :data="filteredUsers" v-loading="loading" stripe style="width: 100%" class="user-table">
        <el-table-column prop="user_no" label="业务编号/卡号" width="150">
          <template #default="{ row }">
            <span class="mono font-semibold">{{ row.user_no }}</span>
          </template>
        </el-table-column>

        <el-table-column label="用户身份" width="170">
          <template #default="{ row }">
            <div class="user-cell">
              <div class="user-avatar-tag" :class="row.role">
                {{ (row.real_name || row.username || '用').charAt(0) }}
              </div>
              <div class="user-name-col">
                <span class="user-real-name font-bold">{{ row.real_name || row.username }}</span>
                <span class="user-username text-xs text-gray-500">@{{ row.username }}</span>
              </div>
            </div>
          </template>
        </el-table-column>

        <el-table-column prop="role" label="系统角色" width="130">
          <template #default="{ row }">
            <el-tag size="small" :type="getRoleTag(row.role)" effect="plain">
              {{ formatRoleLabel(row.role) }}
            </el-tag>
          </template>
        </el-table-column>

        <el-table-column prop="phone" label="联系手机号" width="130">
          <template #default="{ row }">
            <span class="mono">{{ maskPhone(row.phone) }}</span>
          </template>
        </el-table-column>

        <el-table-column prop="id_card" label="居民身份证号" width="185">
          <template #default="{ row }">
            <span class="mono">{{ maskIdCard(row.id_card) }}</span>
          </template>
        </el-table-column>

        <el-table-column label="机构 / 科室 / 职权定位" min-width="220">
          <template #default="{ row }">
            <div v-if="row.role === 'doctor'">
              <span class="text-sm font-medium text-slate-800">{{ row.hospital_name || formatHospName(row.hospital_id) }}</span>
              <div class="text-xs text-blue-600 mt-0.5">
                {{ row.department_name || '临床科室' }} · {{ row.title || '主治医师' }}
              </div>
            </div>
            <div v-else-if="row.role === 'patient'">
              <el-tag size="small" type="success" effect="plain">平台实名就诊人</el-tag>
              <span class="text-xs text-gray-500 ml-2">具有全量健康档案与跨院调阅知情授权权利</span>
            </div>
            <div v-else-if="row.role === 'supervisor'">
              <el-tag size="small" type="warning" effect="plain">卫健委联合监管委派</el-tag>
              <span class="text-xs text-gray-500 ml-2">跨机构急救破窗与合规穿透审计权限</span>
            </div>
            <div v-else>
              <el-tag size="small" type="danger" effect="plain">平台超级管理员</el-tag>
            </div>
          </template>
        </el-table-column>

        <el-table-column prop="status" label="账号状态" width="125" align="center">
          <template #default="{ row }">
            <el-tag v-if="row.status === 'NORMAL'" type="success" size="small">正常 (NORMAL)</el-tag>
            <el-tag v-else-if="row.status === 'RESTRICTED'" type="warning" size="small">已限制 (RESTRICTED)</el-tag>
            <el-tag v-else type="danger" size="small">已禁用 (DISABLED)</el-tag>
          </template>
        </el-table-column>

        <el-table-column label="管理与处置操作" width="260" fixed="right">
          <template #default="{ row }">
            <!-- 1. 临床医生 (doctor): 资质档案 | 限制跨院/恢复正常 | 停用账号 -->
            <template v-if="row.role === 'doctor'">
              <el-button
                size="small"
                type="primary"
                link
                :icon="View"
                @click="openUserDrawer(row)"
              >
                资质档案
              </el-button>
              <el-button
                v-if="row.status === 'RESTRICTED'"
                size="small"
                type="success"
                link
                @click="changeStatus(row.id, 'NORMAL')"
              >
                恢复正常
              </el-button>
              <el-button
                v-else
                size="small"
                type="warning"
                link
                :disabled="row.status === 'DISABLED'"
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
              <el-button
                v-else
                size="small"
                type="success"
                link
                @click="changeStatus(row.id, 'NORMAL')"
              >
                启用账号
              </el-button>
            </template>

            <!-- 2. 就诊患者 (patient): 严格禁止查看病历档案与限制跨院，仅脱敏核验与启停用 -->
            <template v-else-if="row.role === 'patient'">
              <el-button
                size="small"
                type="info"
                link
                :icon="User"
                @click="openUserDrawer(row)"
              >
                脱敏身份核验
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
              <el-button
                v-else
                size="small"
                type="success"
                link
                @click="changeStatus(row.id, 'NORMAL')"
              >
                恢复账号
              </el-button>
            </template>

            <!-- 3. 管理员 / 监管员 (admin / supervisor): 权限审计详情，主管理员账号受保护禁止停用 -->
            <template v-else>
              <el-button
                size="small"
                type="primary"
                link
                :icon="View"
                @click="openUserDrawer(row)"
              >
                权限审计详情
              </el-button>
              <el-button
                size="small"
                type="danger"
                link
                :disabled="row.username === 'admin' || row.id === auth.user?.id"
                :title="row.username === 'admin' ? '主管理员账号受核心保护，禁止停用' : ''"
                @click="changeStatus(row.id, row.status === 'DISABLED' ? 'NORMAL' : 'DISABLED')"
              >
                {{ row.status === 'DISABLED' ? '启用账号' : '停用账号' }}
              </el-button>
            </template>
          </template>
        </el-table-column>
      </el-table>
    </el-card>

    <!-- 用户详细电子档案抽屉 (查看平台用户完整档案信息，不只是医生) -->
    <el-drawer
      v-model="drawerVisible"
      title="平台用户全景电子档案卡"
      size="540px"
      direction="rtl"
    >
      <div v-if="selectedUser" class="drawer-user-body">
        <!-- 头部身份卡 -->
        <div class="user-hero-box mb-4">
          <div class="hero-avatar" :class="selectedUser.role">
            {{ (selectedUser.real_name || selectedUser.username || '用').charAt(0) }}
          </div>
          <div class="hero-details">
            <div class="hero-name-row">
              <span class="name">{{ selectedUser.real_name || selectedUser.username }}</span>
              <el-tag :type="getRoleTag(selectedUser.role)" size="small" class="ml-2">
                {{ formatRoleLabel(selectedUser.role) }}
              </el-tag>
              <el-tag
                :type="selectedUser.status === 'NORMAL' ? 'success' : selectedUser.status === 'RESTRICTED' ? 'warning' : 'danger'"
                size="small"
                effect="plain"
                class="ml-1"
              >
                {{ selectedUser.status }}
              </el-tag>
            </div>
            <div class="hero-user-no text-xs text-gray-500 mt-1">
              业务工号/卡号：<span class="mono">{{ selectedUser.user_no }}</span>
            </div>
            <div class="hero-username text-xs text-gray-500">
              登录账户名：<span class="mono font-semibold">@{{ selectedUser.username }}</span>
            </div>
          </div>
        </div>

        <el-divider content-position="left">实名身份与联系信息</el-divider>
        <div class="detail-info-grid">
          <div class="grid-item">
            <span class="lbl">真实姓名：</span>
            <span class="val font-semibold">{{ selectedUser.real_name || '-' }}</span>
          </div>
          <div class="grid-item">
            <span class="lbl">联系电话：</span>
            <span class="val mono">{{ maskPhone(selectedUser.phone) }}</span>
          </div>
          <div class="grid-item full">
            <span class="lbl">居民身份证号：</span>
            <span class="val mono font-semibold text-slate-800">{{ maskIdCard(selectedUser.id_card) }}</span>
          </div>
          <div class="grid-item full">
            <span class="lbl">实名核验状态：</span>
            <el-tag v-if="selectedUser.id_card" type="success" size="small" effect="plain">
              已登记实名身份凭证（脱敏保护）
            </el-tag>
            <el-tag v-else type="info" size="small" effect="plain">
              未登记身份证件信息
            </el-tag>
          </div>
        </div>

        <!-- 针对医生角色的专属执业档案 -->
        <div v-if="selectedUser.role === 'doctor'">
          <el-divider content-position="left">医疗机构执业与职称凭证</el-divider>
          <div class="detail-info-grid">
            <div class="grid-item">
              <span class="lbl">执业医疗机构：</span>
              <span class="val font-medium text-blue-700">{{ selectedUser.hospital_name || formatHospName(selectedUser.hospital_id) }}</span>
            </div>
            <div class="grid-item">
              <span class="lbl">执业科室：</span>
              <span class="val">{{ selectedUser.department_name || '临床科室' }}</span>
            </div>
            <div class="grid-item">
              <span class="lbl">临床职称：</span>
              <span class="val">{{ selectedUser.title || '主治医师' }}</span>
            </div>
            <div class="grid-item">
              <span class="lbl">机构代码：</span>
              <span class="val mono">HOSP_00{{ selectedUser.hospital_id }}</span>
            </div>
            <div class="grid-item full">
              <span class="lbl">临床安全资质：</span>
              <el-tag type="primary" size="small" effect="plain" class="mr-1">具本院接诊处方权</el-tag>
              <el-tag type="warning" size="small" effect="plain" class="mr-1">支持跨机构知情申请</el-tag>
              <el-tag type="danger" size="small" effect="plain">支持急诊破窗调阅</el-tag>
            </div>
          </div>
        </div>

        <!-- 针对患者角色的专属健康档案 -->
        <div v-else-if="selectedUser.role === 'patient'">
          <el-divider content-position="left">就诊人健康档案与授权主权</el-divider>
          <div class="detail-info-grid">
            <div class="grid-item">
              <span class="lbl">就诊人主索引：</span>
              <span class="val mono text-green-700 font-bold">EMPI-{{ selectedUser.id }}</span>
            </div>
            <div class="grid-item">
              <span class="lbl">电子就诊卡号：</span>
              <span class="val mono font-semibold">{{ selectedUser.user_no }}</span>
            </div>
            <div class="grid-item full">
              <span class="lbl">病历调阅专属密钥：</span>
              <el-tag size="small" type="success" effect="plain">已启用单向加密保护（仅患者本人可见与重置）</el-tag>
            </div>
            <div class="grid-item full">
              <span class="lbl">跨院数据主权：</span>
              <div class="text-xs text-gray-600 mt-1 leading-relaxed">
                患者对自身所有在各级医疗机构生成的电子病历拥有 100% 自主授权主权。外院医生调阅时必须通过该患者显式在线知情授权或现场提供此专属密钥，或由系统在急救破窗后自动通知患者行使异议与审计回溯追责权。
              </div>
            </div>
          </div>
        </div>

        <!-- 针对监管角色的监管审计权限 -->
        <div v-else-if="selectedUser.role === 'supervisor'">
          <el-divider content-position="left">监管合规与审计授权范围</el-divider>
          <div class="detail-info-grid">
            <div class="grid-item full">
              <span class="lbl">委派监管机构：</span>
              <span class="val font-semibold text-purple-700">国家卫生健康委员会 / 省级医疗数据安全监管专班</span>
            </div>
            <div class="grid-item full">
              <span class="lbl">审计权限范围：</span>
              <el-tag type="warning" size="small" effect="plain" class="mr-1">全网调阅操作审计</el-tag>
              <el-tag type="danger" size="small" effect="plain" class="mr-1">急危重症破窗闭环复核</el-tag>
              <el-tag type="success" size="small" effect="plain">区块链存证指纹核验</el-tag>
            </div>
          </div>
        </div>

        <el-divider content-position="left">系统安全与账户运维</el-divider>
        <div class="detail-info-grid">
          <div class="grid-item">
            <span class="lbl">账户注册时间：</span>
            <span class="val text-xs text-gray-500">{{ formatTime(selectedUser.created_at) }}</span>
          </div>
          <div class="grid-item">
            <span class="lbl">区块链用户标识：</span>
            <span class="val mono text-xs text-gray-600">UID: {{ selectedUser.id }}</span>
          </div>
        </div>

        <!-- 底部快捷处置操作按钮 -->
        <div class="drawer-actions mt-4">
          <el-button
            v-if="selectedUser.status !== 'NORMAL'"
            type="success"
            @click="changeStatus(selectedUser.id, 'NORMAL')"
          >
            解封账号 (恢复正常)
          </el-button>
          <el-button
            v-if="selectedUser.role === 'doctor' && selectedUser.status !== 'RESTRICTED'"
            type="warning"
            @click="changeStatus(selectedUser.id, 'RESTRICTED')"
          >
            限制跨院执业 (RESTRICTED)
          </el-button>
          <el-button
            v-if="selectedUser.status !== 'DISABLED'"
            type="danger"
            :disabled="selectedUser.username === 'admin' || selectedUser.id === auth.user?.id"
            @click="changeStatus(selectedUser.id, 'DISABLED')"
          >
            停用此账号 (DISABLED)
          </el-button>
        </div>
      </div>
    </el-drawer>

    <!-- 新增全类用户对话框 -->
    <el-dialog
      v-model="dialogVisible"
      :title="`管理员录入开设系统账号（${getDialogRoleText(form.role)}）`"
      width="640px"
    >
      <el-alert
        :title="getDialogAlertTitle(form.role)"
        type="info"
        :description="getDialogAlertDesc(form.role)"
        show-icon
        :closable="false"
        style="margin-bottom: 16px;"
      />
      <el-form :model="form" label-width="110px">
        <el-row :gutter="12">
          <el-col :span="12">
            <el-form-item label="开设角色类型" required>
              <el-select v-model="form.role" style="width: 100%;">
                <el-option label="执业医生 (doctor)" value="doctor" />
                <el-option label="就诊患者 (patient)" value="patient" />
                <el-option label="监管审计员 (supervisor)" value="supervisor" />
                <el-option label="系统管理员 (admin)" value="admin" />
              </el-select>
            </el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item label="真实姓名" required>
              <el-input v-model="form.real_name" placeholder="如：陈晓华" />
            </el-form-item>
          </el-col>
        </el-row>

        <el-row :gutter="12">
          <el-col :span="12">
            <el-form-item label="登录账号名" required>
              <el-input v-model="form.username" placeholder="如 doc_c 或 pat_zhang" />
            </el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item label="初始密码">
              <el-input v-model="form.password" placeholder="留空使用系统预设，或填≥8位含字母数字密码" show-password />
            </el-form-item>
          </el-col>
        </el-row>

        <el-row :gutter="12">
          <el-col :span="12">
            <el-form-item label="居民身份证" required>
              <el-input v-model="form.id_card" placeholder="18位身份证号" />
            </el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item label="联系电话" required>
              <el-input v-model="form.phone" placeholder="11位手机号" />
            </el-form-item>
          </el-col>
        </el-row>

        <!-- 医生专属表单项（动态读取真实医院与科室字典） -->
        <template v-if="form.role === 'doctor'">
          <el-row :gutter="12">
            <el-col :span="12">
              <el-form-item label="执业机构" required>
                <el-select v-model="form.hospital_id" style="width: 100%;" filterable @change="onCreateHospitalChange">
                  <el-option
                    v-for="h in hospitalsList"
                    :key="h.id"
                    :label="h.name"
                    :value="h.id"
                  />
                </el-select>
              </el-form-item>
            </el-col>
            <el-col :span="12">
              <el-form-item label="执业科室">
                <el-select v-model="form.department_id" style="width: 100%;" filterable clearable placeholder="请选择科室">
                  <el-option
                    v-for="d in filteredCreateDepartments"
                    :key="d.id"
                    :label="d.name"
                    :value="d.id"
                  />
                </el-select>
              </el-form-item>
            </el-col>
          </el-row>

          <el-form-item label="临床/医技职称">
            <el-select v-model="form.title" style="width: 100%;">
              <el-option label="主任医师" value="主任医师" />
              <el-option label="副主任医师" value="副主任医师" />
              <el-option label="主治医师" value="主治医师" />
              <el-option label="主管检验技师" value="主管检验技师" />
              <el-option label="副主任放射技师" value="副主任放射技师" />
              <el-option label="住院医师" value="住院医师" />
            </el-select>
          </el-form-item>
        </template>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="creatingUser" @click="submitAddUser">确认录入开设账号</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { Plus, User, Search, Refresh, View, FirstAidKit, UserFilled, Management } from '@element-plus/icons-vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { listUsersApi, updateUserStatusApi, createUserApi, listHospitalsApi, listDepartmentsApi } from '../../api'
import { useAuthStore } from '../../stores/auth'
import { validatePasswordStrength, isErrorToastShown, extractApiErrorMessage } from '@/utils/passwordPolicy'

const auth = useAuthStore()
const users = ref<any[]>([])
const hospitalsList = ref<any[]>([])
const departmentsList = ref<any[]>([])
const loading = ref(false)
const creatingUser = ref(false)
const dialogVisible = ref(false)
const drawerVisible = ref(false)
const selectedUser = ref<any>(null)

const activeRoleTab = ref('ALL')
const searchKeyword = ref('')
const statusFilter = ref('')

const form = ref({
  username: '',
  password: '',
  real_name: '',
  role: 'doctor',
  hospital_id: 1,
  department_id: 2,
  title: '主治医师',
  phone: '',
  id_card: '',
})

const doctorCount = computed(() => users.value.filter(u => u.role === 'doctor').length)
const patientCount = computed(() => users.value.filter(u => u.role === 'patient').length)
const supervisorCount = computed(() => users.value.filter(u => u.role === 'supervisor').length)
const adminCount = computed(() => users.value.filter(u => u.role === 'admin').length)

const filteredUsers = computed(() => {
  let list = users.value
  if (activeRoleTab.value !== 'ALL') {
    list = list.filter(u => u.role === activeRoleTab.value)
  }
  if (statusFilter.value) {
    list = list.filter(u => u.status === statusFilter.value)
  }
  if (searchKeyword.value.trim()) {
    const kw = searchKeyword.value.trim().toLowerCase()
    list = list.filter(u =>
      (u.username && u.username.toLowerCase().includes(kw)) ||
      (u.real_name && u.real_name.toLowerCase().includes(kw)) ||
      (u.phone && u.phone.includes(kw)) ||
      (u.id_card && u.id_card.includes(kw)) ||
      (u.user_no && u.user_no.toLowerCase().includes(kw))
    )
  }
  return list
})

const filteredCreateDepartments = computed(() => {
  if (!form.value.hospital_id) return departmentsList.value
  const matched = departmentsList.value.filter(d => d.hospital_id === form.value.hospital_id)
  return matched.length > 0 ? matched : departmentsList.value
})

function onCreateHospitalChange() {
  const firstDept = filteredCreateDepartments.value[0]
  form.value.department_id = firstDept ? firstDept.id : undefined
}

function maskPhone(phone?: string) {
  if (!phone) return '-'
  const s = String(phone).trim()
  if (s.length < 7) return s
  return `${s.slice(0, 3)}****${s.slice(-4)}`
}

function maskIdCard(id?: string) {
  if (!id) return '-'
  const s = String(id).trim()
  if (s.length < 10) return s
  return `${s.slice(0, 6)}********${s.slice(-4)}`
}

async function loadDictData() {
  try {
    const [hRes, dRes]: any = await Promise.all([listHospitalsApi(), listDepartmentsApi()])
    if (hRes?.code === 200 && Array.isArray(hRes.data)) {
      hospitalsList.value = hRes.data
    }
    if (dRes?.code === 200 && Array.isArray(dRes.data)) {
      departmentsList.value = dRes.data
    }
  } catch (err) {
    console.warn('加载医院/科室字典失败:', err)
  }
}

onMounted(() => {
  loadUsers()
  loadDictData()
})

async function loadUsers() {
  loading.value = true
  try {
    const res: any = await listUsersApi()
    if (res.code === 200) {
      users.value = res.data || []
    }
  } catch (err: unknown) {
    if (!isErrorToastShown(err)) {
      ElMessage.error(extractApiErrorMessage(err, '获取用户列表失败'))
    }
  } finally {
    loading.value = false
  }
}

function formatHospName(hId?: number) {
  if (!hId) return '-'
  const matched = hospitalsList.value.find(h => h.id === hId)
  if (matched?.name) return matched.name
  if (hId === 1) return '第一人民医院'
  if (hId === 2) return '省立中心医院'
  if (hId === 3) return '协和医学中心'
  return `机构 #${hId}`
}

function formatRoleLabel(role: string) {
  if (role === 'doctor') return '临床医生'
  if (role === 'patient') return '就诊患者'
  if (role === 'supervisor') return '监管审计员'
  if (role === 'admin') return '系统管理员'
  return role
}

function getRoleTag(role: string) {
  if (role === 'doctor') return 'primary'
  if (role === 'patient') return 'success'
  if (role === 'supervisor') return 'warning'
  return 'danger'
}

function formatTime(t?: string) {
  if (!t) return '-'
  return t.substring(0, 19).replace('T', ' ')
}

function openUserDrawer(row: any) {
  selectedUser.value = row
  drawerVisible.value = true
}

function openCreateDialog() {
  const defaultHospId = hospitalsList.value[0]?.id ?? 1
  const defaultDept = departmentsList.value.find(d => d.hospital_id === defaultHospId) || departmentsList.value[0]
  form.value = {
    username: '',
    password: '',
    real_name: '',
    role: activeRoleTab.value !== 'ALL' ? activeRoleTab.value : 'doctor',
    hospital_id: defaultHospId,
    department_id: defaultDept?.id ?? 2,
    title: '主治医师',
    phone: '',
    id_card: '',
  }
  dialogVisible.value = true
}

function getDialogRoleText(role: string) {
  if (role === 'doctor') return '执业医生建档'
  if (role === 'patient') return '患者实名建档'
  if (role === 'supervisor') return '监管审计员授权'
  return '系统管理员'
}

function getDialogAlertTitle(role: string) {
  if (role === 'doctor') return '医生执业账号专设机制'
  if (role === 'patient') return '就诊患者实名开卡机制'
  if (role === 'supervisor') return '监管合规委派机制'
  return '系统运维管理账号'
}

function getDialogAlertDesc(role: string) {
  if (role === 'doctor') {
    return '根据卫健可信医疗数据规范，临床医生账号涉及处方权与跨机构急救病历解密，必须由管理员核验执业医师资格证后在此录入开设。'
  }
  if (role === 'patient') {
    return '就诊患者账号基于真实居民身份证建档，关联统一电子健康卡与全网病历档案，支持个人自主知情同意与授权控制。'
  }
  if (role === 'supervisor') {
    return '监管人员账号具备全网就诊与跨机构调阅审计日志穿透核查权限，仅限各级卫生健康委员会授权人员开设。'
  }
  return '系统管理员账号具备平台全局用户维护、安全策略与机构权限配置权。'
}

async function changeStatus(id: number, status: string) {
  const targetUser = users.value.find(u => u.id === id) || selectedUser.value
  if (status === 'DISABLED' || status === 'RESTRICTED') {
    const actionLabel = status === 'DISABLED' ? '停用该账号' : '限制该医生跨院与急救破窗权限'
    const userNameText = targetUser ? `${targetUser.real_name || targetUser.username}（@${targetUser.username}）` : `用户 #${id}`
    try {
      await ElMessageBox.confirm(
        `确认对 ${userNameText} 执行「${actionLabel}」操作吗？变更后权限将立即生效。`,
        '高危账号状态变更确认',
        {
          confirmButtonText: '确认执行',
          cancelButtonText: '取消',
          type: 'warning',
        }
      )
    } catch {
      return
    }
  }

  try {
    const res: any = await updateUserStatusApi(id, status)
    if (res.code === 200) {
      ElMessage.success('账号状态变更成功！')
      if (selectedUser.value && selectedUser.value.id === id) {
        selectedUser.value.status = status
      }
      loadUsers()
    } else {
      ElMessage.error(res?.message || '账号状态变更失败')
    }
  } catch (err: unknown) {
    if (!isErrorToastShown(err)) {
      ElMessage.error(extractApiErrorMessage(err, '变更失败'))
    }
  }
}

async function submitAddUser() {
  if (!form.value.username || !form.value.real_name || !form.value.id_card || !form.value.phone) {
    ElMessage.warning('请填写完整的用户名、姓名、身份证号与联系电话')
    return
  }
  if (form.value.password && form.value.password.trim()) {
    const pwdErr = validatePasswordStrength(form.value.password)
    if (pwdErr) {
      ElMessage.warning(pwdErr)
      return
    }
  }
  creatingUser.value = true
  try {
    const res: any = await createUserApi(form.value)
    if (res.code === 200) {
      ElMessage.success(form.value.password?.trim() ? '用户账号创建成功！' : '用户账号创建成功，已启用系统默认初始密码')
      dialogVisible.value = false
      loadUsers()
    } else {
      ElMessage.error(res?.message || '创建失败')
    }
  } catch (err: unknown) {
    if (!isErrorToastShown(err)) {
      ElMessage.error(extractApiErrorMessage(err, '创建失败'))
    }
  } finally {
    creatingUser.value = false
  }
}
</script>

<style scoped>
.user-mgmt-page {
  max-width: 1400px;
  margin: 0 auto;
  padding: 16px;
}

.stats-grid {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 10px;
}

.stat-card {
  background: #ffffff;
  border-radius: 6px;
  padding: 8px 14px;
  min-height: 44px;
  max-height: 48px;
  display: flex;
  align-items: center;
  gap: 10px;
  border: 1px solid #e2e8f0;
  box-shadow: none;
}

.stat-card.blue { border-left: 3px solid #0284c7; }
.stat-card.cyan { border-left: 3px solid #0369a1; }
.stat-card.green { border-left: 3px solid #059669; }
.stat-card.purple { border-left: 3px solid #475569; }

.stat-icon { font-size: 16px; color: #0284c7; display: flex; align-items: center; }
.stat-info { display: flex; align-items: baseline; justify-content: space-between; width: 100%; gap: 8px; }
.stat-value { font-size: 16px; font-weight: 700; color: #0f172a; font-family: var(--mt-font-mono, monospace); line-height: 1; order: 2; }
.stat-label { font-size: 12px; color: #475569; font-weight: 600; margin-top: 0; order: 1; }

.hero-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 12px;
}

.header-left {
  display: flex;
  align-items: center;
  gap: 12px;
}

.header-icon-ring {
  width: 36px;
  height: 36px;
  border-radius: 6px;
  background: #f0f9ff;
  border: 1px solid #bae6fd;
  color: #0284c7;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 18px;
}

.page-title {
  font-size: 18px;
  font-weight: 700;
  color: #0f172a;
}

.page-sub {
  font-size: 12px;
  color: #64748b;
  margin-top: 2px;
}

.mgmt-card {
  border-radius: 8px;
  border: 1px solid #e2e8f0;
}

.toolbar-wrapper {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 16px;
  flex-wrap: wrap;
  gap: 12px;
}

.filter-controls {
  display: flex;
  align-items: center;
  gap: 12px;
}

.user-cell {
  display: flex;
  align-items: center;
  gap: 10px;
}

.user-avatar-tag {
  width: 34px;
  height: 34px;
  border-radius: 8px;
  background: #f1f5f9;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 16px;
  flex-shrink: 0;
}

.user-avatar-tag.doctor { background: #eff6ff; }
.user-avatar-tag.patient { background: #ecfdf5; }
.user-avatar-tag.supervisor { background: #fef3c7; }
.user-avatar-tag.admin { background: #fee2e2; }

.user-name-col {
  display: flex;
  flex-direction: column;
}

.user-real-name {
  color: #1e293b;
  font-size: 0.92rem;
}

.mono {
  font-family: 'Consolas', monospace;
}

/* 抽屉样式 */
.drawer-user-body {
  padding: 8px 16px 24px;
}

.user-hero-box {
  display: flex;
  align-items: center;
  gap: 16px;
  padding: 16px;
  background: #f8fafc;
  border: 1px solid #e2e8f0;
  border-radius: 10px;
}

.hero-avatar {
  width: 44px;
  height: 44px;
  border-radius: 6px;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 20px;
  background: #f0f9ff;
  color: #0284c7;
  border: 1px solid #bae6fd;
}

.hero-name-row {
  display: flex;
  align-items: center;
}

.hero-name-row .name {
  font-size: 18px;
  font-weight: 700;
  color: #0f172a;
}

.detail-info-grid {
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  gap: 12px 16px;
  font-size: 0.88rem;
  background: #fdfdfd;
  padding: 12px 14px;
  border-radius: 8px;
  border: 1px solid #f1f5f9;
}

.grid-item {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.grid-item.full {
  grid-column: span 2;
}

.grid-item .lbl {
  color: #64748b;
  font-size: 0.82rem;
}

.grid-item .val {
  color: #1e293b;
}

.drawer-actions {
  display: flex;
  flex-wrap: wrap;
  gap: 10px;
  padding-top: 16px;
}

/* 修复表格右侧固定列 fixed="right" 样式污染与绿色垂直割裂线 */
:deep(.el-table__fixed-right) {
  background: #ffffff !important;
  box-shadow: -4px 0 12px rgba(15, 23, 42, 0.05) !important;
}
:deep(.el-table__fixed-right::before) {
  display: none !important;
}
:deep(.el-table__fixed-right-patch) {
  background: #ffffff !important;
  border-bottom: 1px solid #f1f5f9 !important;
}
:deep(.el-table__fixed-right .el-table__cell) {
  background: #ffffff !important;
}
:deep(.el-table__body tr:hover > td.el-table__fixed-right-column) {
  background-color: #f8fafc !important;
}
</style>
