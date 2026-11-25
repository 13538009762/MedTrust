<template>
  <el-container class="medtrust-layout-wrapper">
    <el-aside :width="isCollapse ? '64px' : '250px'" class="medtrust-sidebar">
      <div class="sidebar-brand">
        <img src="/emblem.png" class="brand-logo" alt="MedTrust 3D Emblem" />
        <div v-if="!isCollapse" class="brand-info">
          <span class="brand-title">MedTrust</span>
          <span class="brand-badge">医联体可信共享 2.0</span>
        </div>
      </div>

      <el-menu
        :default-active="activeMenu"
        router
        :collapse="isCollapse"
        :collapse-transition="false"
        :default-openeds="['/doctor/query']"
        class="medtrust-menu"
      >
        <!-- 医生 / 医技菜单 -->
        <template v-if="auth.user?.role === 'doctor'">
          <template v-if="isLabTechnician">
            <div class="menu-category" v-if="!isCollapse">医技检查工作台</div>
            <el-menu-item index="/doctor/lab-center">
              <el-icon><Tickets /></el-icon>
              <span>医技检查中心</span>
            </el-menu-item>
          </template>
          <template v-else>
            <div class="menu-category" v-if="!isCollapse">医生临床工作台</div>
            <el-menu-item index="/doctor/records">
              <el-icon><FolderAdd /></el-icon>
              <span>就诊记录与初诊录入</span>
            </el-menu-item>
            <el-menu-item index="/doctor/patient-search">
              <el-icon><FirstAidKit /></el-icon>
              <span>患者检索与急救调阅</span>
            </el-menu-item>
            <el-menu-item index="/doctor/lab-center">
              <el-icon><Tickets /></el-icon>
              <span>医技检查中心</span>
            </el-menu-item>

            <!-- 病历查询折叠子目录 -->
            <el-sub-menu index="/doctor/query">
              <template #title>
                <el-icon><Search /></el-icon>
                <span>跨院病历查询</span>
              </template>
              <el-menu-item index="/doctor/query/authorized">
                <el-icon><CircleCheckFilled /></el-icon>
                <span>已获授权外院病历</span>
              </el-menu-item>
              <el-menu-item index="/doctor/query/local">
                <el-icon><OfficeBuilding /></el-icon>
                <span>本院病历</span>
              </el-menu-item>
              <el-menu-item index="/doctor/query/pending">
                <el-icon><Lock /></el-icon>
                <span>待申请外院病历</span>
              </el-menu-item>
              <el-menu-item index="/doctor/query/all">
                <el-icon><Compass /></el-icon>
                <span>全网联合检索</span>
              </el-menu-item>
            </el-sub-menu>
          </template>
        </template>

        <!-- 患者菜单 -->
        <template v-if="auth.user?.role === 'patient'">
          <div class="menu-category" v-if="!isCollapse">患者健康中心</div>
          <el-menu-item index="/patient/records">
            <el-icon><Document /></el-icon>
            <span>我的电子健康档案</span>
          </el-menu-item>
          <el-menu-item index="/patient/auth" class="menu-item-with-badge">
            <el-icon><Key /></el-icon>
            <span class="menu-label-text">授权策略管理</span>
            <span v-if="pendingCount > 0" class="nav-pill-badge">{{ pendingCount }}</span>
          </el-menu-item>
          <el-menu-item index="/patient/emergency">
            <el-icon><Bell /></el-icon>
            <span>紧急访问知情面板</span>
          </el-menu-item>
        </template>

        <!-- 监管人员菜单 -->
        <template v-if="auth.user?.role === 'supervisor'">
          <div class="menu-category" v-if="!isCollapse">医疗监管审计中心</div>
          <el-menu-item index="/supervisor/overview">
            <el-icon><DataLine /></el-icon>
            <span>全网存证监控大屏</span>
          </el-menu-item>
          <el-menu-item index="/supervisor/emergency">
            <el-icon><WarningFilled /></el-icon>
            <span>紧急访问事件审核台</span>
          </el-menu-item>
          <el-menu-item index="/supervisor/audits">
            <el-icon><Monitor /></el-icon>
            <span>可信审计日志溯源</span>
          </el-menu-item>
          <el-menu-item index="/supervisor/verify">
            <el-icon><CircleCheck /></el-icon>
            <span>动态防篡改核验</span>
          </el-menu-item>
        </template>

        <!-- 管理员菜单 -->
        <template v-if="auth.user?.role === 'admin'">
          <div class="menu-category" v-if="!isCollapse">系统运维控制台</div>
          <el-menu-item index="/admin/users">
            <el-icon><User /></el-icon>
            <span>用户账号与状态惩戒</span>
          </el-menu-item>
          <el-menu-item index="/admin/hospitals">
            <el-icon><OfficeBuilding /></el-icon>
            <span>医院与科室字典</span>
          </el-menu-item>
          <el-menu-item index="/supervisor/overview">
            <el-icon><DataLine /></el-icon>
            <span>存证数据大屏</span>
          </el-menu-item>
        </template>

        <!-- 通用个人中心 / 我的 -->
        <div class="menu-category" v-if="!isCollapse">账户与个人中心</div>
        <el-menu-item index="/personal">
          <el-icon><UserFilled /></el-icon>
          <span>我的信息与设置</span>
        </el-menu-item>
      </el-menu>
    </el-aside>

    <el-container class="main-container">
      <el-header class="medtrust-header">
        <div class="header-left">
          <el-icon class="collapse-btn" @click="isCollapse = !isCollapse">
            <component :is="isCollapse ? Expand : Fold" />
          </el-icon>
          <span class="logo-text full-title">基于区块链的跨机构医疗数据可信共享平台</span>
          <span class="logo-text short-title">MedTrust 可信共享平台</span>
        </div>

        <div class="header-right">
          <!-- 联盟链与 IPFS 底层实时双状态指示灯 -->
          <el-tooltip :content="chainTooltip" placement="bottom">
            <div
              class="header-chain-badge"
              :class="[chainBadgeClass, { 'is-clickable': canOpenChainOverview }]"
              @click="onChainBadgeClick"
              :title="canOpenChainOverview ? '点击查看底层联盟链运行状态' : '底层联盟链运行状态'"
            >
              <span class="mini-pulse-dot" :class="chainDotClass"></span>
              <span class="chain-badge-text">{{ chainBadgeText }}</span>
              <span v-if="chainStatus.latest_block" class="chain-block-num">#{{ chainStatus.latest_block }}</span>
            </div>
          </el-tooltip>

          <el-tooltip :content="ipfsTooltip" placement="bottom">
            <div class="header-ipfs-badge" :class="{ 'ipfs-offline': !ipfsOnline }">
              <span class="mini-pulse-dot" :class="ipfsOnline ? 'dot-green' : 'dot-amber'"></span>
              <span class="ipfs-badge-text">{{ ipfsOnline ? 'IPFS 密文节点 (就绪)' : 'IPFS 存储状态待确认' }}</span>
              <span class="ipfs-badge-short">{{ ipfsOnline ? 'IPFS (就绪)' : 'IPFS (待确认)' }}</span>
            </div>
          </el-tooltip>

          <!-- 实时消息通知信道状态微件 (实时通道) -->
          <el-tooltip :content="isChannelOnline ? '实时双向业务通知通道已连接 (SSE 在线)' : '实时通知通道连接中或已断开，系统将自动退避重连'" placement="bottom">
            <div class="header-sse-badge" :class="isChannelOnline ? 'sse-online' : 'sse-offline'">
              <span class="sse-pulse-dot" :class="isChannelOnline ? 'dot-green' : 'dot-amber'"></span>
              <span class="sse-badge-text">{{ isChannelOnline ? '实时通道已就绪' : '实时通道重连中' }}</span>
              <span class="sse-badge-short">{{ isChannelOnline ? '实时通道' : '重连中' }}</span>
            </div>
          </el-tooltip>

          <!-- 角色标签 -->
          <el-tag
            :type="getRoleTagType(auth.user?.role)"
            effect="dark"
            class="role-badge"
          >
            {{ getRoleName(auth.user?.role) }}
          </el-tag>

          <!-- 账号执业/监管状态标签：仅当受限或停用时展示惩戒告警，避免在正常状态下与角色“执业医生”徽标叠词重复 -->
          <el-tag
            v-if="auth.user?.status === 'RESTRICTED' || auth.user?.status === 'DISABLED'"
            :type="auth.user?.status === 'RESTRICTED' ? 'danger' : 'info'"
            :effect="auth.user?.status === 'RESTRICTED' ? 'dark' : 'light'"
            class="header-user-status-tag"
          >
            <span class="status-mini-dot" :class="auth.user?.status === 'RESTRICTED' ? 'dot-red-mini' : 'dot-gray-mini'"></span>
            {{ auth.user?.status === 'RESTRICTED' ? '监管受限' : '已停用' }}
          </el-tag>

          <!-- 用户真实姓名与所属机构科室常驻展示 -->
          <div class="user-profile clickable" @click="router.push('/personal')" title="点击查看并维护个人信息">
            <el-avatar size="small" :src="auth.user?.avatar" :style="{ backgroundColor: '#0284c7' }">
              {{ (auth.user?.real_name || '用').charAt(0) }}
            </el-avatar>
            <div class="user-meta-stacked">
              <span class="user-name">{{ auth.user?.real_name || auth.user?.username }}</span>
              <span class="user-affiliation">{{ userAffiliationDisplay }}</span>
            </div>
          </div>

          <el-button link type="danger" @click="onLogout" title="退出系统" class="logout-btn">
            <el-icon :size="18"><SwitchButton /></el-icon>
          </el-button>
        </div>
      </el-header>

      <el-main class="medtrust-main">
        <div class="medtrust-main-inner">
          <router-view v-slot="{ Component }">
            <transition name="fade-slide" mode="out-in">
              <component :is="Component" :key="$route.path" />
            </transition>
          </router-view>
        </div>
      </el-main>

      <!-- 全局受控 AI 助手悬浮窗 -->
      <AiAssistantDialog />
    </el-container>
  </el-container>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { Expand, Fold, SwitchButton } from '@element-plus/icons-vue'
import { useAuthStore } from '../stores/auth'
import AiAssistantDialog from '../components/AiAssistantDialog.vue'
import { getBlockchainStatusApi, listPendingAccessRequestsApi } from '../api'
import { useSSE } from '../composables/useSSE'

const isCollapse = ref(false)
const route = useRoute()
const router = useRouter()
const auth = useAuthStore()
const pendingCount = ref(0)

const { isConnected: isSSEConnected, connect: connectSSE, disconnect: disconnectSSE, addEventListener: addSSEListener } = useSSE()

interface BlockchainStatus {
  mode: string
  connected: boolean
  network: string
  chaincode: string
  peer: string
  latest_block: number
  message: string
}

const chainStatusLoaded = ref(false)
const chainStatus = ref<BlockchainStatus>({
  mode: 'fabric',
  connected: false,
  network: 'medicalchannel',
  chaincode: 'medical_contract',
  peer: 'peer0.org1.example.com',
  latest_block: 0,
  message: '正在检测联盟链状态...',
})

const displayChannelName = computed(() => {
  const net = chainStatus.value.network
  if (!net || net === 'medchannel') return 'medicalchannel'
  return net
})

const isChannelOnline = computed(() => isSSEConnected.value)

const ipfsOnline = computed(() => {
  return chainStatusLoaded.value && (chainStatus.value.connected || chainStatus.value.mode === 'mock')
})

const ipfsTooltip = computed(() => {
  if (!chainStatusLoaded.value) return '正在检测密文存储网关状态...'
  return ipfsOnline.value
    ? '密文存储网关可用，病历附件经 AES-256-GCM 加密后写入 IPFS 寻址'
    : '后端存储网关当前未连接，密文上传/读取可能受影响'
})

const canOpenChainOverview = computed(() => {
  return auth.user?.role === 'supervisor' || auth.user?.role === 'admin'
})

async function fetchChainStatus() {
  try {
    const res: any = await getBlockchainStatusApi()
    const st = (res && res.data !== undefined) ? res.data : res
    if (st && st.mode) {
      chainStatus.value = st
    }
    chainStatusLoaded.value = true
  } catch {
    chainStatusLoaded.value = true
    chainStatus.value = {
      mode: 'fabric',
      connected: false,
      network: 'medicalchannel',
      chaincode: 'medical_contract',
      peer: 'peer0.org1.example.com',
      latest_block: 0,
      message: 'Fabric Unavailable: Connection Failed',
    }
  }
}

const chainBadgeText = computed(() => {
  if (chainStatus.value.mode === 'mock') {
    return 'Mock 账本 (容灾模式)'
  }
  if (chainStatus.value.connected) {
    return `Fabric (${displayChannelName.value}) 在线`
  }
  return `Fabric 节点未连接`
})

const chainTooltip = computed(() => {
  const net = displayChannelName.value
  const cc = chainStatus.value.chaincode === 'medical' ? 'medical_contract' : (chainStatus.value.chaincode || 'medical_contract')
  const heightText = chainStatus.value.latest_block ? `#${chainStatus.value.latest_block}` : '待同步'
  return `运行模式: ${chainStatus.value.mode} | 通道: ${net} | 合约: ${cc} | 节点: ${chainStatus.value.peer || '-'} | 最新高度: ${heightText}`
})

const chainBadgeClass = computed(() => {
  if (chainStatus.value.mode === 'fabric' && chainStatus.value.connected) return 'badge-fabric-ok'
  if (chainStatus.value.mode === 'mock') return 'badge-mock-dev'
  return 'badge-fabric-err'
})

const chainDotClass = computed(() => {
  if (chainStatus.value.mode === 'fabric' && chainStatus.value.connected) return 'dot-green'
  if (chainStatus.value.mode === 'mock') return 'dot-amber'
  return 'dot-red'
})

function onChainBadgeClick() {
  if (auth.user?.role === 'supervisor' || auth.user?.role === 'admin') {
    router.push('/supervisor/overview')
  }
}

const activeMenu = computed(() => {
  if (route.path === '/doctor/cross-query' || route.path === '/doctor/query') {
    return '/doctor/query/all'
  }
  if (route.path === '/doctor/query/same') {
    return '/doctor/query/local'
  }
  return route.path
})

async function checkPendingConsent() {
  if (auth.user?.role === 'patient') {
    try {
      const res: any = await listPendingAccessRequestsApi()
      if (res.code === 200) {
        pendingCount.value = res.data.length
      }
    } catch {}
  }
}

function handleWindowResize() {
  if (window.innerWidth < 1200) {
    isCollapse.value = true
  }
}

let removeSSEListener: (() => void) | null = null

onMounted(() => {
  handleWindowResize()
  window.addEventListener('resize', handleWindowResize)
  checkPendingConsent()
  fetchChainStatus()

  // 启动 SSE 实时业务通知通道
  const token = auth.token
  if (token) {
    connectSSE(token)
  }

  // 注册全局事件监听器：当收到新的调阅申请时，实时刷新待处理徽章
  removeSSEListener = addSSEListener((event) => {
    if (event.type === 'access_request' && auth.user?.role === 'patient') {
      checkPendingConsent()
    }
  })
})

onUnmounted(() => {
  window.removeEventListener('resize', handleWindowResize)
  if (removeSSEListener) {
    removeSSEListener()
    removeSSEListener = null
  }
  disconnectSSE()
})

watch(() => route.path, () => {
  checkPendingConsent()
  fetchChainStatus()
  const token = auth.token
  if (token && !isSSEConnected.value) {
    connectSSE(token)
  }
})

function getRoleTagType(role?: string) {
  if (role === 'doctor') return 'primary'
  if (role === 'patient') return 'success'
  if (role === 'supervisor') return 'warning'
  if (role === 'admin') return 'danger'
  return 'info'
}

function getRoleName(role?: string) {
  if (role === 'doctor') return '执业医生'
  if (role === 'patient') return '患者中心'
  if (role === 'supervisor') return '监管审计'
  if (role === 'admin') return '系统管理'
  return '用户'
}

import { ElMessage, ElMessageBox } from 'element-plus'

const isLabTechnician = computed(() => {
  return auth.user?.role === 'doctor' && Boolean(auth.user?.username?.startsWith('tech_'))
})

const userAffiliationDisplay = computed(() => {
  const u = auth.user
  if (!u) return ''
  if (u.role === 'doctor') {
    const hosp = u.hospital_name || (u.hospital_id === 1 ? '第一人民医院' : (u.hospital_id === 2 ? '省立中心医院' : (u.hospital_id === 3 ? '协和医学中心' : '医疗机构')))
    const dept = u.department_name || (u.department_id === 1 || u.department_id === 3 ? '心血管内科' : (u.department_id === 2 || u.department_id === 4 ? '急诊科' : '临床科室'))
    return `${hosp} · ${dept}`
  }
  if (u.role === 'patient') {
    return '国家健康档案持有者'
  }
  if (u.role === 'supervisor') {
    return '国家卫健委 · 医疗监管审计中心'
  }
  if (u.role === 'admin') {
    return 'MedTrust 平台 · 运维技术部'
  }
  return u.hospital_name || '医疗联盟'
})

async function onLogout() {
  try {
    await ElMessageBox.confirm('确定要退出当前登录账号吗？', '退出系统确认', {
      confirmButtonText: '确认退出',
      cancelButtonText: '取消',
      type: 'info',
    })
  } catch {
    return
  }
  disconnectSSE()
  try {
    await auth.logout()
    ElMessage.success('已安全退出登录')
  } catch (err) {
    console.warn('Logout error:', err)
  } finally {
    router.push('/login')
  }
}
</script>

<style scoped>
.medtrust-layout-wrapper {
  height: 100vh;
  width: 100vw;
  display: flex;
  background: #f8fafc;
}
.medtrust-sidebar {
  background: #ffffff;
  border-right: 1px solid rgba(226, 232, 240, 0.8);
  display: flex;
  flex-direction: column;
  transition: width 0.3s cubic-bezier(0.2, 0.8, 0.2, 1);
  box-shadow: 2px 0 10px rgba(0, 0, 0, 0.02);
}
.sidebar-brand {
  height: 64px;
  display: flex;
  align-items: center;
  padding: 0 16px;
  gap: 12px;
  border-bottom: 1px solid rgba(226, 232, 240, 0.8);
}
.brand-logo {
  width: 36px;
  height: 36px;
  object-fit: contain;
  filter: drop-shadow(0 2px 8px rgba(14, 165, 233, 0.3));
  transition: all 0.3s cubic-bezier(0.4, 0, 0.2, 1);
  flex-shrink: 0;
}
.brand-logo:hover {
  transform: scale(1.08) rotate(3deg);
  filter: drop-shadow(0 4px 12px rgba(14, 165, 233, 0.5));
}
.brand-info {
  display: flex;
  align-items: center;
  gap: 8px;
}
.brand-title {
  font-size: 20px;
  font-weight: 800;
  color: var(--mt-primary, #0284c7);
  letter-spacing: -0.4px;
}
.brand-badge {
  font-size: 10px;
  background: var(--mt-primary-soft, #f0f9ff);
  color: var(--mt-primary, #0284c7);
  border: 1px solid var(--mt-primary-border, #bae6fd);
  padding: 2px 6px;
  border-radius: 4px;
  font-weight: 700;
}
.menu-category {
  font-size: 11px;
  color: #94a3b8;
  font-weight: 700;
  padding: 16px 20px 6px;
  text-transform: uppercase;
}
.medtrust-menu {
  border-right: none;
  flex: 1;
}
.medtrust-menu :deep(.el-menu-item.menu-item-with-badge) {
  display: flex !important;
  align-items: center !important;
  justify-content: space-between !important;
  padding-right: 16px !important;
}
.menu-label-text {
  flex: 1;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.nav-pill-badge {
  display: inline-flex !important;
  align-items: center !important;
  justify-content: center !important;
  background-color: #dc2626;
  color: #ffffff;
  font-size: 11px;
  font-weight: 700;
  min-width: 18px;
  height: 18px;
  line-height: 18px;
  padding: 0 6px;
  border-radius: 4px;
  margin-left: auto;
  flex-shrink: 0;
}
.main-container {
  flex: 1;
  display: flex;
  flex-direction: column;
  min-width: 0;
}
.medtrust-header {
  height: 60px;
  background: #ffffff;
  border-bottom: 1px solid var(--mt-border-base, #e2e8f0);
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0 20px;
  overflow: hidden !important;
  white-space: nowrap !important;
}
.header-left {
  display: flex;
  align-items: center;
  gap: 12px;
  min-width: 0;
  flex-shrink: 1;
}
.collapse-btn {
  font-size: 20px;
  cursor: pointer;
  color: #64748b;
  transition: color 0.2s;
  flex-shrink: 0;
}
.collapse-btn:hover {
  color: var(--mt-primary, #0284c7);
}
.logo-text {
  font-size: 16px;
  font-weight: 700;
  color: #0f172a;
  white-space: nowrap !important;
  overflow: hidden;
  text-overflow: ellipsis;
}
.logo-text.full-title {
  display: inline-block;
}
.logo-text.short-title {
  display: none;
}
.header-right {
  display: flex;
  align-items: center;
  gap: 10px;
  flex-shrink: 0;
  overflow: hidden !important;
  white-space: nowrap !important;
  scrollbar-width: none !important;
  -ms-overflow-style: none !important;
}
.header-right::-webkit-scrollbar {
  display: none !important;
  width: 0 !important;
  height: 0 !important;
}
.header-right > * {
  flex-shrink: 0;
}
.ipfs-badge-short,
.sse-badge-short {
  display: none;
}
.ipfs-badge-text,
.sse-badge-text {
  display: inline;
}

.header-chain-badge {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  padding: 4px 10px;
  border-radius: 6px;
  font-size: 12px;
  font-weight: 600;
  cursor: default;
  transition: all 0.2s ease;
  user-select: none;
  white-space: nowrap;
}
.header-chain-badge.is-clickable {
  cursor: pointer;
}
.badge-fabric-ok {
  background: var(--mt-success-soft, #ecfdf5);
  color: var(--mt-success, #059669);
  border: 1px solid var(--mt-success-border, #a7f3d0);
}
.badge-fabric-ok:hover {
  background: #d1fae5;
}
.badge-mock-dev {
  background: var(--mt-warning-soft, #fffbeb);
  color: var(--mt-warning, #d97706);
  border: 1px solid var(--mt-warning-border, #fde68a);
}
.badge-mock-dev:hover {
  background: #fef3c7;
}
.badge-fabric-err {
  background: var(--mt-danger-soft, #fef2f2);
  color: var(--mt-danger, #dc2626);
  border: 1px solid var(--mt-danger-border, #fecaca);
}
.badge-fabric-err:hover {
  background: #fee2e2;
}

.mini-pulse-dot {
  width: 7px;
  height: 7px;
  border-radius: 50%;
  display: inline-block;
}
.dot-green {
  background-color: #059669;
}
.dot-amber {
  background-color: #d97706;
}
.dot-red {
  background-color: #dc2626;
}

.chain-badge-text {
  letter-spacing: 0.2px;
  white-space: nowrap;
}
.chain-block-num {
  font-family: var(--mt-font-mono, monospace) !important;
  font-variant-numeric: tabular-nums;
  font-weight: 700;
  padding: 1px 6px;
  border-radius: 4px;
  background: rgba(15, 23, 42, 0.06);
  white-space: nowrap;
  margin-left: 2px;
}

.header-sse-badge {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 4px 10px;
  border-radius: 6px;
  font-size: 11.5px;
  font-weight: 600;
  user-select: none;
  transition: all 0.2s ease;
}
.sse-online {
  background: var(--mt-success-soft, #ecfdf5);
  color: var(--mt-success, #059669);
  border: 1px solid var(--mt-success-border, #a7f3d0);
}
.sse-offline {
  background: #f8fafc;
  color: #64748b;
  border: 1px solid #cbd5e1;
}
.sse-pulse-dot {
  width: 7px;
  height: 7px;
  border-radius: 50%;
}

.header-ipfs-badge {
  display: flex;
  align-items: center;
  gap: 7px;
  padding: 4px 10px;
  border-radius: 6px;
  font-size: 12px;
  font-weight: 600;
  background: var(--mt-primary-soft, #f0f9ff);
  color: var(--mt-primary, #0284c7);
  border: 1px solid var(--mt-primary-border, #bae6fd);
  transition: all 0.2s ease;
  user-select: none;
}
.header-ipfs-badge.ipfs-offline {
  background: #fffbeb;
  color: #b45309;
  border-color: #fde68a;
}
.header-ipfs-badge:hover {
  background: #e0f2fe;
}
.ipfs-badge-text {
  letter-spacing: 0.2px;
}

.header-user-status-tag {
  font-weight: 700;
  display: inline-flex;
  align-items: center;
  gap: 4px;
}

.status-mini-dot {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  display: inline-block;
}
.dot-green-mini {
  background: #059669;
}
.dot-red-mini {
  background: #ffffff;
}
.dot-gray-mini {
  background: #94a3b8;
}

.status-warning-tag {
  font-weight: 700;
}
.user-profile {
  display: flex;
  align-items: center;
  gap: 8px;
}
.user-profile.clickable {
  cursor: pointer;
  padding: 4px 8px;
  border-radius: 6px;
  transition: all 0.2s ease;
}
.user-profile.clickable:hover {
  background: var(--mt-primary-soft, #f0f9ff);
}

.user-meta-stacked {
  display: flex;
  flex-direction: column;
  line-height: 1.25;
}
.user-name {
  font-size: 13px;
  font-weight: 700;
  color: #1e293b;
}
.user-affiliation {
  font-size: 11px;
  color: #64748b;
  max-width: 170px;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.medtrust-main {
  flex: 1;
  padding: 0;
  overflow-y: auto;
  overflow-x: hidden;
  background: #f8fafc;
}
.medtrust-main-inner {
  padding: 24px 24px 100px 24px;
  min-height: 100%;
  overflow-x: clip;
}

/* GPU 硬件加速丝滑路由切换动效 */
.fade-slide-enter-active,
.fade-slide-leave-active {
  transition: opacity 0.25s cubic-bezier(0.2, 0.8, 0.2, 1),
              transform 0.25s cubic-bezier(0.2, 0.8, 0.2, 1);
  will-change: transform, opacity;
}
.fade-slide-enter-from {
  opacity: 0;
  transform: translate3d(0, 14px, 0);
}
.fade-slide-leave-to {
  opacity: 0;
  transform: translate3d(0, -14px, 0);
}

@media (max-width: 1480px) {
  /* 优先保留关键的“监管受限”状态告警徽标，仅折叠区块号等次要技术指标 */
  .chain-block-num {
    display: none !important;
  }
}

@media (max-width: 1320px) {
  .logo-text.full-title {
    display: none !important;
  }
  .logo-text.short-title {
    display: inline-block !important;
  }
  .user-affiliation {
    display: none !important;
  }
  .header-ipfs-badge {
    display: none !important;
  }
  .header-right {
    gap: 8px !important;
  }
}

@media (max-width: 1180px) {
  .ipfs-badge-text {
    display: none !important;
  }
  .ipfs-badge-short {
    display: inline !important;
  }
  .sse-badge-text {
    display: none !important;
  }
  .sse-badge-short {
    display: inline !important;
  }
}

@media (max-width: 1024px) {
  .role-badge {
    display: none !important;
  }
  .header-chain-badge {
    padding: 4px 8px !important;
    font-size: 11px !important;
  }
  .header-ipfs-badge {
    padding: 4px 8px !important;
    font-size: 11px !important;
  }
  .header-sse-badge {
    padding: 4px 8px !important;
    font-size: 11px !important;
  }
  .medtrust-header {
    padding: 0 12px !important;
  }
}
</style>
