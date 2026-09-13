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
          <el-menu-item index="/doctor/lab-center">
            <el-icon><Tickets /></el-icon>
            <span>医技检查中心</span>
          </el-menu-item>

          <!-- 病例查询折叠子目录 -->
          <el-sub-menu index="/doctor/query">
            <template #title>
              <el-icon><Search /></el-icon>
              <span>病例查询</span>
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

          <div class="user-profile clickable" @click="router.push('/personal')" title="点击查看并维护个人信息">
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
import { ref, computed, onMounted, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { Expand, Fold, SwitchButton } from '@element-plus/icons-vue'
import { useAuthStore } from '../stores/auth'
import AiAssistantDialog from '../components/AiAssistantDialog.vue'
import api from '../api/client'

const isCollapse = ref(false)
const route = useRoute()
const router = useRouter()
const auth = useAuthStore()
const pendingCount = ref(0)

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
})

watch(() => route.path, () => {
  checkPendingConsent()
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
