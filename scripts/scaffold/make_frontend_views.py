import os

base = r'e:\EnglishEncoding\competition\last\MedTrust\frontend\src'

def write_file(rel_path, content):
    full_path = os.path.join(base, rel_path)
    os.makedirs(os.path.dirname(full_path), exist_ok=True)
    with open(full_path, 'w', encoding='utf-8') as out:
        out.write(content.strip() + '\n')
    print('Wrote:', rel_path)

# 1. router/index.ts
write_file('router/index.ts', """import { createRouter, createWebHistory } from 'vue-router'
import { useAuthStore } from '../stores/auth'

const router = createRouter({
  history: createWebHistory(),
  routes: [
    {
      path: '/login',
      name: 'login',
      component: () => import('../views/LoginView.vue'),
    },
    {
      path: '/',
      component: () => import('../layouts/MainLayout.vue'),
      meta: { requiresAuth: true },
      children: [
        // 医生工作台
        { path: '', redirect: '/doctor/records' },
        { path: 'doctor/records', name: 'doctor-records', component: () => import('../views/doctor/DoctorRecordsView.vue') },
        { path: 'doctor/cross-query', name: 'doctor-cross-query', component: () => import('../views/doctor/CrossQueryView.vue') },

        // 患者中心
        { path: 'patient/records', name: 'patient-records', component: () => import('../views/patient/PatientRecordsView.vue') },
        { path: 'patient/auth', name: 'patient-auth', component: () => import('../views/patient/AuthManagerView.vue') },
        { path: 'patient/emergency', name: 'patient-emergency', component: () => import('../views/patient/EmergencyNoticeView.vue') },

        // 管理员控制台
        { path: 'admin/users', name: 'admin-users', component: () => import('../views/admin/UserManagementView.vue') },
        { path: 'admin/hospitals', name: 'admin-hospitals', component: () => import('../views/admin/HospitalManagementView.vue') },

        // 监管审计看板
        { path: 'supervisor/overview', name: 'supervisor-overview', component: () => import('../views/supervisor/OverviewView.vue') },
        { path: 'supervisor/emergency', name: 'supervisor-emergency', component: () => import('../views/supervisor/EmergencyAuditView.vue') },
        { path: 'supervisor/audits', name: 'supervisor-audits', component: () => import('../views/supervisor/AuditLogView.vue') },
        { path: 'supervisor/verify', name: 'supervisor-verify', component: () => import('../views/supervisor/VerifyView.vue') },
      ],
    },
  ],
})

router.beforeEach((to, _from, next) => {
  const auth = useAuthStore()
  if (to.meta.requiresAuth && !auth.token) {
    next({ name: 'login' })
    return
  }
  if (to.name === 'login' && auth.token) {
    if (auth.user?.role === 'doctor') next('/doctor/records')
    else if (auth.user?.role === 'patient') next('/patient/records')
    else if (auth.user?.role === 'supervisor') next('/supervisor/overview')
    else if (auth.user?.role === 'admin') next('/admin/users')
    else next('/doctor/records')
    return
  }
  next()
})

export default router
""")

# 2. layouts/MainLayout.vue
write_file('layouts/MainLayout.vue', """<template>
  <el-container class="medtrust-layout-wrapper">
    <el-aside :width="isCollapse ? '64px' : '250px'" class="medtrust-sidebar">
      <div class="sidebar-brand">
        <img src="/favicon.png" class="brand-logo" alt="Logo" />
        <div v-if="!isCollapse" class="brand-info">
          <span class="brand-title">MedTrust</span>
          <span class="brand-badge">可信共享 2.0</span>
        </div>
      </div>

      <el-menu
        :default-active="route.path"
        router
        :collapse="isCollapse"
        :collapse-transition="false"
        class="medtrust-menu"
      >
        <!-- 医生菜单 -->
        <template v-if="auth.user?.role === 'doctor'">
          <div class="menu-category" v-if="!isCollapse">医生临床工作台</div>
          <el-menu-item index="/doctor/records">
            <el-icon><FolderAdd /></el-icon>
            <span>病历录入与上链</span>
          </el-menu-item>
          <el-menu-item index="/doctor/cross-query">
            <el-icon><Search /></el-icon>
            <span>跨院调阅 (含Break-Glass)</span>
          </el-menu-item>
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
          <!-- 角色标签 -->
          <el-tag
            :type="getRoleTagType(auth.user?.role)"
            effect="dark"
            class="role-badge"
          >
            {{ getRoleName(auth.user?.role) }}
          </el-tag>

          <!-- 账号状态惩戒警示 -->
          <el-tag
            v-if="auth.user?.status === 'RESTRICTED'"
            type="danger"
            effect="plain"
            class="status-warning-tag"
          >
            ⚠️ 账号已被监管限制跨院访问
          </el-tag>

          <div class="user-profile">
            <el-avatar size="small" :style="{ backgroundColor: '#8b5cf6' }">
              {{ (auth.user?.real_name || '用').charAt(0) }}
            </el-avatar>
            <span class="user-name">{{ auth.user?.real_name }}</span>
            <span v-if="auth.user?.hospital_name" class="user-hosp">({{ auth.user?.hospital_name }})</span>
          </div>

          <el-button link type="danger" @click="onLogout" title="退出系统">
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
    </el-container>
  </el-container>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { Expand, Fold, SwitchButton } from '@element-plus/icons-vue'
import { useAuthStore } from '../stores/auth'
import AiAssistantDialog from '../components/AiAssistantDialog.vue'

const isCollapse = ref(false)
const route = useRoute()
const router = useRouter()
const auth = useAuthStore()

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

function onLogout() {
  auth.logout()
  router.push('/login')
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
  width: 32px;
  height: 32px;
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
.status-warning-tag {
  font-weight: 700;
}
.user-profile {
  display: flex;
  align-items: center;
  gap: 8px;
}
.user-name {
  font-size: 14px;
  font-weight: 600;
  color: #1e293b;
}
.user-hosp {
  font-size: 12px;
  color: #64748b;
}
.medtrust-main {
  flex: 1;
  padding: 24px;
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
</style>
""")

# 3. views/LoginView.vue
write_file('views/LoginView.vue', """<template>
  <div class="login-container">
    <!-- 顶部导航 -->
    <div class="top-nav">
      <div class="logo-section">
        <img src="/favicon.png" class="app-logo" alt="MedTrust Logo" />
        <div class="logo-text">
          <span class="brand">MedTrust</span>
          <span class="divider">|</span>
          <span class="app-name">医疗数据区块链可信共享平台 2.0</span>
        </div>
      </div>
    </div>

    <!-- 主体区域 -->
    <div class="main-content">
      <!-- 左侧 Hero 说明区 -->
      <div class="hero-area">
        <div class="hero-content">
          <h1 class="hero-title">
            可信存证 · 动态访问 · <span class="text-gradient">AI 智能辅助</span>
          </h1>
          <p class="hero-subtitle">
            面向跨医院医疗协同共享场景，构建链下 IPFS 密文存储与 Fabric 账本锚定协同机制，打造轻量严密的 Break-Glass 紧急访问闭环。
          </p>

          <div class="feature-list">
            <div class="feature-item" v-for="(f, i) in features" :key="i">
              <div class="feature-icon" :style="{ background: f.bg }">
                <el-icon><component :is="f.icon" /></el-icon>
              </div>
              <div class="feature-info">
                <h3>{{ f.title }}</h3>
                <p>{{ f.desc }}</p>
              </div>
            </div>
          </div>

          <!-- 左下角系统介绍卡片 -->
          <div class="system-intro-card">
            <h3>医疗数据可信共享平台</h3>
            <p>
              模拟医院A、医院B、医院C三方参与的联盟网络。通过 AES-256-GCM 本地加密与 IPFS 结合，有效规避账本膨胀；以统一安全网关收口权限校验，提供合规可溯源的急救访问闭环。
            </p>
          </div>
        </div>
      </div>

      <!-- 右侧登录舱 (极光白亚克力材质) -->
      <div class="login-panel">
        <div class="login-glass-card">
          <div class="card-header">
            <h2>欢迎登录</h2>
            <p>基于数字身份的医疗可信访问网关</p>
          </div>

          <!-- 快速填充演示栏 (专为答辩 8 分钟流程准备) -->
          <div class="demo-account-bar">
            <div class="demo-title">⚡ 快速登录演示账号 (密码均为 123456)</div>
            <div class="demo-buttons">
              <el-button size="small" type="primary" plain @click="fillAccount('doc_a')">李医生(医院A)</el-button>
              <el-button size="small" type="primary" plain @click="fillAccount('doc_b')">王医生(医院B)</el-button>
              <el-button size="small" type="success" plain @click="fillAccount('pat_zhang')">患者张三</el-button>
              <el-button size="small" type="warning" plain @click="fillAccount('supervisor')">监管专员</el-button>
              <el-button size="small" type="danger" plain @click="fillAccount('admin')">系统管理员</el-button>
            </div>
          </div>

          <el-form class="login-form" @submit.prevent="handleLogin">
            <el-form-item>
              <el-input
                v-model="username"
                placeholder="请输入用户名 (如 doc_a, doc_b, supervisor)"
                :prefix-icon="User"
                size="large"
              />
            </el-form-item>

            <el-form-item>
              <el-input
                v-model="password"
                type="password"
                placeholder="请输入密码 (预置 123456)"
                :prefix-icon="Lock"
                size="large"
                show-password
              />
            </el-form-item>

            <el-button
              type="primary"
              class="login-btn"
              :loading="loading"
              native-type="submit"
            >
              进 入 系 统
            </el-button>
          </el-form>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { User, Lock, Lock as KeyIcon, ShieldCheck, Share, MagicStick } from '@element-plus/icons-vue'
import { ElMessage } from 'element-plus'
import api from '../api/client'
import { useAuthStore } from '../stores/auth'

const username = ref('doc_a')
const password = ref('123456')
const loading = ref(false)
const router = useRouter()
const auth = useAuthStore()

const features = [
  {
    title: '链下密文 + 链上凭证',
    desc: 'AES-256-GCM 加密，IPFS 寻址，Fabric 固化 SHA-256 指纹',
    icon: KeyIcon,
    bg: '#ede9fe',
  },
  {
    title: '复合访问控制体系',
    desc: 'RBAC角色 + ABAC属性 + 患者显式知情授权过滤',
    icon: ShieldCheck,
    bg: '#dcfce7',
  },
  {
    title: 'Break-Glass 紧急访问',
    desc: '昏迷危重抢救放行，双向知情通知与监管闭环追责',
    icon: Share,
    bg: '#fef3c7',
  },
  {
    title: '受控 AI Agent 辅助',
    desc: 'Tool Calling 透传 Token，严禁直连库，保障安全底线',
    icon: MagicStick,
    bg: '#e0e7ff',
  },
]

function fillAccount(acc: string) {
  username.value = acc
  password.value = '123456'
  ElMessage.success(`已自动填充账号: ${acc}，点击登录即可进入`)
}

async function handleLogin() {
  if (!username.value || !password.value) {
    ElMessage.warning('请输入用户名和密码')
    return
  }

  loading.value = true
  try {
    const res: any = await api.post('/auth/login', {
      username: username.value,
      password: password.value,
    })

    if (res.code === 200) {
      auth.setAuth(res.data.token, res.data.user)
      ElMessage.success(`欢迎回来，${res.data.user.real_name}`)
      
      const role = res.data.user.role
      if (role === 'doctor') router.push('/doctor/records')
      else if (role === 'patient') router.push('/patient/records')
      else if (role === 'supervisor') router.push('/supervisor/overview')
      else if (role === 'admin') router.push('/admin/users')
      else router.push('/doctor/records')
    } else {
      ElMessage.error(res.message || '登录失败')
    }
  } catch (err: any) {
    ElMessage.error(err?.message || '登录请求失败')
  } finally {
    loading.value = false
  }
}
</script>

<style scoped>
.login-container {
  --el-color-primary: #8b5cf6;
  --el-color-primary-light-3: #a78bfa;
  --el-color-primary-light-5: #c4b5fd;
  --el-color-primary-light-7: #ddd6fe;
  --el-color-primary-light-8: #ede9fe;
  --el-color-primary-light-9: #f5f3ff;
  --el-color-primary-dark-2: #7c3aed;

  width: 100vw;
  min-height: 100vh;
  background: #f8fafc url('/images/v1.png') no-repeat center center;
  background-size: cover;
  display: flex;
  flex-direction: column;
  color: #1e293b;
  position: relative;
}
.top-nav {
  padding: 28px 64px;
  display: flex;
  align-items: center;
}
.logo-section {
  display: flex;
  align-items: center;
  gap: 12px;
}
.app-logo {
  width: 40px;
  height: 40px;
}
.brand {
  font-size: 26px;
  font-weight: 900;
  color: #8b5cf6;
  letter-spacing: -1px;
}
.divider {
  color: #cbd5e1;
  margin: 0 12px;
}
.app-name {
  font-size: 15px;
  color: #475569;
  font-weight: 600;
}
.main-content {
  flex: 1;
  display: flex;
  padding: 0 80px;
  align-items: center;
  justify-content: space-between;
  max-width: 1600px;
  margin: 0 auto;
  width: 100%;
}
.hero-area {
  flex: 1;
  max-width: 620px;
}
.hero-title {
  font-size: clamp(30px, 3.8vw, 48px);
  font-weight: 850;
  color: #0f172a;
  margin-bottom: 20px;
  line-height: 1.15;
  letter-spacing: -1px;
}
.text-gradient {
  color: #7c3aed;
}
.hero-subtitle {
  font-size: 16px;
  color: #334155;
  line-height: 1.6;
  margin-bottom: 40px;
}
.feature-list {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 16px;
  margin-bottom: 40px;
}
.feature-item {
  display: flex;
  align-items: center;
  gap: 14px;
  background: rgba(255, 255, 255, 0.45);
  backdrop-filter: blur(8px);
  padding: 14px;
  border-radius: 14px;
  border: 1px solid rgba(255, 255, 255, 0.6);
}
.feature-icon {
  width: 44px;
  height: 44px;
  border-radius: 12px;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 20px;
  color: #7c3aed;
  flex-shrink: 0;
}
.feature-info h3 {
  font-size: 15px;
  font-weight: 700;
  margin: 0 0 4px 0;
  color: #0f172a;
}
.feature-info p {
  font-size: 12px;
  color: #475569;
  margin: 0;
  line-height: 1.35;
}
.system-intro-card {
  padding: 20px;
  background: rgba(255, 255, 255, 0.5);
  backdrop-filter: blur(12px);
  border: 1px solid rgba(255, 255, 255, 0.7);
  border-radius: 16px;
  box-shadow: 0 10px 25px rgba(0, 0, 0, 0.02);
}
.system-intro-card h3 {
  font-size: 15px;
  font-weight: 700;
  color: #1e293b;
  margin-bottom: 8px;
}
.system-intro-card p {
  font-size: 13px;
  color: #475569;
  line-height: 1.5;
  margin: 0;
}
.login-panel {
  flex-shrink: 0;
  margin-right: 40px;
}
.login-glass-card {
  width: clamp(340px, 28vw, 420px);
  padding: 36px 30px;
  background: rgba(255, 255, 255, 0.55) !important;
  backdrop-filter: blur(24px) !important;
  border: 1px solid rgba(255, 255, 255, 0.8) !important;
  box-shadow: 0 20px 40px rgba(139, 92, 246, 0.08) !important;
  border-radius: 24px;
}
.card-header h2 {
  font-size: 28px;
  font-weight: 800;
  color: #0f172a;
  margin: 0 0 6px 0;
}
.card-header p {
  font-size: 14px;
  color: #64748b;
  margin-bottom: 24px;
}
.demo-account-bar {
  background: rgba(237, 233, 254, 0.6);
  border: 1px dashed rgba(139, 92, 246, 0.4);
  border-radius: 12px;
  padding: 12px;
  margin-bottom: 24px;
}
.demo-title {
  font-size: 12px;
  font-weight: 700;
  color: #7c3aed;
  margin-bottom: 8px;
}
.demo-buttons {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}
.login-btn {
  width: 100%;
  height: 50px;
  border-radius: 10px;
  font-size: 16px;
  font-weight: 700;
  letter-spacing: 2px;
  background: linear-gradient(135deg, #a78bfa 0%, #8b5cf6 100%) !important;
  border: none;
  box-shadow: 0 8px 16px rgba(139, 92, 246, 0.25) !important;
  color: white !important;
  transition: all 0.3s cubic-bezier(0.4, 0, 0.2, 1);
  margin-top: 10px;
}
.login-btn:hover {
  transform: translateY(-2px);
  box-shadow: 0 12px 24px rgba(139, 92, 246, 0.35) !important;
}
</style>
""")

print("Views part 1 created successfully!")
