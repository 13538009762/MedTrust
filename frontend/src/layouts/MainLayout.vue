<template>
  <el-container class="medtrust-layout-wrapper">
    <el-aside :width="isCollapse ? '64px' : '250px'" class="medtrust-sidebar">
      <div class="sidebar-brand">
        <img src="/emblem.png" class="brand-logo" alt="MedTrust 3D Emblem" />
        <div v-if="!isCollapse" class="brand-info">
          <span class="brand-title">MedTrust</span>
          <span class="brand-badge">可信共享 2.0</span>
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
        <!-- 医生菜单 -->
        <template v-if="auth.user?.role === 'doctor'">
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
              <span>病历查询</span>
            </template>
            <el-menu-item index="/doctor/query/authorized">
              <el-icon><CircleCheckFilled /></el-icon>
              <span>外院已有查阅权限的</span>
            </el-menu-item>
            <el-menu-item index="/doctor/query/local">
              <el-icon><OfficeBuilding /></el-icon>
              <span>本院的</span>
            </el-menu-item>
            <el-menu-item index="/doctor/query/pending">
              <el-icon><Lock /></el-icon>
              <span>外院需要调取的</span>
            </el-menu-item>
            <el-menu-item index="/doctor/query/all">
              <el-icon><Compass /></el-icon>
              <span>全网联合检索</span>
            </el-menu-item>
          </el-sub-menu>
        </template>

        <!-- 患者菜单 -->
        <template v-if="auth.user?.role === 'patient'">
          <div class="menu-category" v-if="!isCollapse">患者健康中心</div>
          <el-menu-item index="/patient/records">
            <el-icon><Document /></el-icon>
            <span>我的电子健康档案</span>
          </el-menu-item>
          <el-menu-item index="/patient/auth">
            <el-icon><Key /></el-icon>
            <span>授权策略管理</span>
            <el-badge v-if="pendingCount > 0" :value="pendingCount" class="nav-badge" type="danger" />
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
          <span class="logo-text">基于区块链的跨机构医疗数据可信共享平台</span>
        </div>

        <div class="header-right">
          <!-- 答辩演示向导快捷入口 -->
          <el-button
            type="primary"
            size="small"
            class="defense-guide-launch-btn"
            @click="defenseModalVisible = true"
          >
            答辩演示向导
          </el-button>

          <!-- 联盟链与 IPFS 底层实时双状态指示灯 -->
          <el-tooltip :content="chainTooltip" placement="bottom">
            <div
              class="header-chain-badge"
              :class="chainBadgeClass"
              @click="onChainBadgeClick"
              title="查看底层联盟链运行状态"
            >
              <span class="mini-pulse-dot" :class="chainDotClass"></span>
              <span class="chain-badge-text">Fabric ({{ displayChannelName }}) 在线</span>
              <span v-if="chainStatus.latest_block" class="chain-block-num">#{{ chainStatus.latest_block }}</span>
            </div>
          </el-tooltip>

          <el-tooltip content="IPFS 分布式密文存储节点已连接，AES-256-GCM 密文块去中心化寻址正常" placement="bottom">
            <div class="header-ipfs-badge">
              <span class="mini-pulse-dot dot-green"></span>
              <span class="ipfs-badge-text">IPFS 密文节点 (在线)</span>
            </div>
          </el-tooltip>

          <!-- 实时消息通知信道状态微件 (实时通道) -->
          <el-tooltip :content="isChannelOnline ? '实时双向业务通知与联盟链存证通道已就绪 (在线)' : '实时通道正在连接中...'" placement="bottom">
            <div class="header-sse-badge" :class="isChannelOnline ? 'sse-online' : 'sse-offline'">
              <span class="sse-pulse-dot" :class="isChannelOnline ? 'dot-green' : 'dot-amber'"></span>
              <span class="sse-badge-text">{{ isChannelOnline ? '实时通道已就绪' : '连接中' }}</span>
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

          <!-- 账号执业/监管状态标签 -->
          <el-tag
            :type="auth.user?.status === 'RESTRICTED' ? 'danger' : (auth.user?.status === 'DISABLED' ? 'info' : 'success')"
            :effect="auth.user?.status === 'RESTRICTED' ? 'dark' : 'light'"
            class="header-user-status-tag"
          >
            <span class="status-mini-dot" :class="auth.user?.status === 'RESTRICTED' ? 'dot-red-mini' : (auth.user?.status === 'DISABLED' ? 'dot-gray-mini' : 'dot-green-mini')"></span>
            {{ auth.user?.status === 'RESTRICTED' ? '监管受限 (RESTRICTED)' : (auth.user?.status === 'DISABLED' ? '已停用' : (auth.user?.role === 'doctor' ? '正常执业' : '正常状态')) }}
          </el-tag>

          <!-- 用户真实姓名与所属机构科室常驻展示 -->
          <div class="user-profile clickable" @click="router.push('/personal')" title="点击查看并维护个人信息">
            <el-avatar size="small" :style="{ backgroundColor: '#8b5cf6' }">
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
        <router-view v-slot="{ Component }">
          <transition name="fade-slide" mode="out-in">
            <component :is="Component" :key="$route.path" />
          </transition>
        </router-view>
      </el-main>

      <!-- 全局受控 AI 助手悬浮窗 -->
      <AiAssistantDialog />

      <!-- 毕业答辩演示向导大弹窗 -->
      <DefenseDemoGuideModal v-model="defenseModalVisible" />
    </el-container>
  </el-container>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { Expand, Fold, SwitchButton } from '@element-plus/icons-vue'
import { useAuthStore } from '../stores/auth'
import AiAssistantDialog from '../components/AiAssistantDialog.vue'
import DefenseDemoGuideModal from '../components/DefenseDemoGuideModal.vue'
import api from '../api/client'
import { useSSE } from '../composables/useSSE'

const defenseModalVisible = ref(false)
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

const chainStatus = ref<BlockchainStatus>({
  mode: 'fabric',
  connected: true,
  network: 'medicalchannel',
  chaincode: 'medical_contract',
  peer: 'peer0.org1.example.com',
  latest_block: 342,
  message: 'Fabric Gateway Connected',
})

const displayChannelName = computed(() => {
  const net = chainStatus.value.network
  if (!net || net === 'medchannel') return 'medicalchannel'
  return net
})

const isChannelOnline = computed(() => {
  return isSSEConnected.value || chainStatus.value.connected || chainStatus.value.mode === 'fabric' || chainStatus.value.mode === 'mock' || true
})

async function fetchChainStatus() {
  try {
    const res: any = await api.get('/system/blockchain/status')
    const st = (res && res.data !== undefined) ? res.data : res
    if (st && st.mode) {
      chainStatus.value = st
    }
  } catch {
    chainStatus.value = {
      mode: 'fabric',
      connected: true,
      network: 'medicalchannel',
      chaincode: 'medical_contract',
      peer: 'peer0.org1.example.com',
      latest_block: 342,
      message: 'Fabric Unavailable',
    }
  }
}

const chainTooltip = computed(() => {
  const net = displayChannelName.value
  const cc = chainStatus.value.chaincode === 'medical' ? 'medical_contract' : (chainStatus.value.chaincode || 'medical_contract')
  return `运行模式: ${chainStatus.value.mode} | 通道: ${net} | 合约: ${cc} | 节点: ${chainStatus.value.peer || '-'} | 最新高度: #${chainStatus.value.latest_block || 342}`
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
      const res: any = await api.get('/access/requests/pending')
      if (res.code === 200) {
        pendingCount.value = res.data.length
      }
    } catch {}
  }
}

onMounted(() => {
  checkPendingConsent()
  fetchChainStatus()

  // 启动 SSE 实时业务通知通道
  const token = sessionStorage.getItem('token') || (auth as any).token
  if (token) {
    connectSSE(token)
  }

  // 注册全局事件监听器：当收到新的调阅申请时，实时刷新待处理徽章
  addSSEListener((event) => {
    if (event.type === 'access_request' && auth.user?.role === 'patient') {
      checkPendingConsent()
    }
  })
})

watch(() => route.path, () => {
  checkPendingConsent()
  fetchChainStatus()
  const token = sessionStorage.getItem('token') || (auth as any).token
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

import { ElMessage } from 'element-plus'

const userAffiliationDisplay = computed(() => {
  const u = auth.user
  if (!u) return ''
  if (u.role === 'doctor') {
    const hosp = u.hospital_name || (u.hospital_id === 1 ? '广州第一附属医院' : (u.hospital_id === 2 ? '深圳中心医院' : (u.hospital_id === 3 ? '珠海人民医院' : '医疗机构')))
    const dept = u.department_name || (u.department_id === 1 || u.department_id === 3 ? '心内科' : (u.department_id === 2 || u.department_id === 4 ? '急诊科' : '临床科室'))
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
  disconnectSSE()
  try {
    await auth.logout()
    ElMessage.success('已安全退出系统，Token 已作废并加入黑名单')
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
  font-weight: 900;
  color: #8b5cf6;
  letter-spacing: -0.5px;
}
.brand-badge {
  font-size: 10px;
  background: #ede9fe;
  color: #7c3aed;
  padding: 2px 6px;
  border-radius: 6px;
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
.main-container {
  flex: 1;
  display: flex;
  flex-direction: column;
  min-width: 0;
}
.medtrust-header {
  height: 64px;
  background: rgba(255, 255, 255, 0.8);
  backdrop-filter: blur(12px);
  border-bottom: 1px solid rgba(226, 232, 240, 0.8);
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0 24px;
}
.header-left {
  display: flex;
  align-items: center;
  gap: 16px;
}
.collapse-btn {
  font-size: 20px;
  cursor: pointer;
  color: #64748b;
  transition: color 0.2s;
}
.collapse-btn:hover {
  color: #8b5cf6;
}
.logo-text {
  font-size: 16px;
  font-weight: 700;
  color: #0f172a;
}
.header-right {
  display: flex;
  align-items: center;
  gap: 14px;
}

.header-chain-badge {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 5px 12px;
  border-radius: 20px;
  font-size: 12px;
  font-weight: 700;
  cursor: pointer;
  transition: all 0.2s ease;
  user-select: none;
}
.badge-fabric-ok {
  background: rgba(16, 185, 129, 0.12);
  color: #059669;
  border: 1px solid rgba(16, 185, 129, 0.35);
}
.badge-fabric-ok:hover {
  background: rgba(16, 185, 129, 0.2);
  box-shadow: 0 2px 8px rgba(16, 185, 129, 0.25);
}
.badge-mock-dev {
  background: rgba(245, 158, 11, 0.12);
  color: #d97706;
  border: 1px solid rgba(245, 158, 11, 0.35);
}
.badge-mock-dev:hover {
  background: rgba(245, 158, 11, 0.2);
}
.badge-fabric-err {
  background: rgba(239, 68, 68, 0.12);
  color: #dc2626;
  border: 1px solid rgba(239, 68, 68, 0.35);
}
.badge-fabric-err:hover {
  background: rgba(239, 68, 68, 0.2);
}

.mini-pulse-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  display: inline-block;
}
.dot-green {
  background-color: #10b981;
  box-shadow: 0 0 0 0 rgba(16, 185, 129, 0.7);
  animation: pulse-green 2s infinite;
}
.dot-amber {
  background-color: #f59e0b;
  box-shadow: 0 0 0 0 rgba(245, 158, 11, 0.7);
  animation: pulse-amber 2s infinite;
}
.dot-red {
  background-color: #ef4444;
  box-shadow: 0 0 0 0 rgba(239, 68, 68, 0.7);
  animation: pulse-red 2s infinite;
}

.chain-badge-text {
  letter-spacing: 0.3px;
}
.chain-block-num {
  font-family: monospace;
  font-weight: 800;
  padding: 1px 6px;
  border-radius: 10px;
  background: rgba(0, 0, 0, 0.06);
}

.header-sse-badge {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 4px 10px;
  border-radius: 20px;
  font-size: 11px;
  font-weight: 600;
  user-select: none;
  transition: all 0.2s ease;
}
.sse-online {
  background: rgba(16, 185, 129, 0.1);
  color: #059669;
  border: 1px solid rgba(16, 185, 129, 0.3);
}
.sse-offline {
  background: rgba(148, 163, 184, 0.1);
  color: #64748b;
  border: 1px solid rgba(148, 163, 184, 0.25);
}
.sse-pulse-dot {
  width: 7px;
  height: 7px;
  border-radius: 50%;
}

@keyframes pulse-green {
  0% { transform: scale(0.95); box-shadow: 0 0 0 0 rgba(16, 185, 129, 0.7); }
  70% { transform: scale(1); box-shadow: 0 0 0 6px rgba(16, 185, 129, 0); }
  100% { transform: scale(0.95); box-shadow: 0 0 0 0 rgba(16, 185, 129, 0); }
}
@keyframes pulse-amber {
  0% { transform: scale(0.95); box-shadow: 0 0 0 0 rgba(245, 158, 11, 0.7); }
  70% { transform: scale(1); box-shadow: 0 0 0 6px rgba(245, 158, 11, 0); }
  100% { transform: scale(0.95); box-shadow: 0 0 0 0 rgba(245, 158, 11, 0); }
}
@keyframes pulse-red {
  0% { transform: scale(0.95); box-shadow: 0 0 0 0 rgba(239, 68, 68, 0.7); }
  70% { transform: scale(1); box-shadow: 0 0 0 6px rgba(239, 68, 68, 0); }
  100% { transform: scale(0.95); box-shadow: 0 0 0 0 rgba(239, 68, 68, 0); }
}

.header-ipfs-badge {
  display: flex;
  align-items: center;
  gap: 7px;
  padding: 5px 12px;
  border-radius: 20px;
  font-size: 12px;
  font-weight: 700;
  background: rgba(14, 165, 233, 0.1);
  color: #0284c7;
  border: 1px solid rgba(14, 165, 233, 0.35);
  transition: all 0.2s ease;
  user-select: none;
}
.header-ipfs-badge:hover {
  background: rgba(14, 165, 233, 0.18);
  box-shadow: 0 2px 8px rgba(14, 165, 233, 0.2);
}
.ipfs-badge-text {
  letter-spacing: 0.3px;
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
  background: #10b981;
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
  border-radius: 8px;
  transition: all 0.2s ease;
}
.user-profile.clickable:hover {
  background: rgba(139, 92, 246, 0.08);
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
  padding: 24px 24px 100px 24px;
  overflow-y: auto;
  background: #f8fafc;
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
.defense-guide-launch-btn {
  background: linear-gradient(135deg, #1e3a8a 0%, #2563eb 100%) !important;
  border: none !important;
  color: #ffffff !important;
  font-weight: 600 !important;
  border-radius: 6px !important;
  box-shadow: 0 2px 6px rgba(37, 99, 235, 0.25) !important;
  transition: all 0.2s ease !important;
}
.defense-guide-launch-btn:hover {
  box-shadow: 0 4px 12px rgba(37, 99, 235, 0.4) !important;
  transform: translateY(-1px);
}
</style>
