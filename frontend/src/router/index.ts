import { createRouter, createWebHistory } from 'vue-router'
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
        // 医生临床与医技工作台
        { path: '', redirect: '/doctor/records' },
        { path: 'doctor/records', name: 'doctor-records', meta: { roles: ['doctor'] }, component: () => import('../views/doctor/DoctorRecordsView.vue') },
        { path: 'doctor/patient-search', name: 'doctor-patient-search', meta: { roles: ['doctor'] }, component: () => import('../views/doctor/PatientEmergencySearchView.vue') },
        { path: 'doctor/lab-center', name: 'doctor-lab-center', meta: { roles: ['doctor'] }, component: () => import('../views/doctor/LabCenterView.vue') },
        { path: 'doctor/cross-query', redirect: to => ({ path: '/doctor/query/all', query: to.query }) },
        { path: 'doctor/query', redirect: '/doctor/query/all' },
        { path: 'doctor/query/:scope', name: 'doctor-query-scope', meta: { roles: ['doctor'] }, component: () => import('../views/doctor/CrossQueryView.vue') },

        // 患者中心
        { path: 'patient/records', name: 'patient-records', meta: { roles: ['patient'] }, component: () => import('../views/patient/PatientRecordsView.vue') },
        { path: 'patient/auth', name: 'patient-auth', meta: { roles: ['patient'] }, component: () => import('../views/patient/AuthManagerView.vue') },
        { path: 'patient/emergency', name: 'patient-emergency', meta: { roles: ['patient'] }, component: () => import('../views/patient/EmergencyNoticeView.vue') },

        // 管理员控制台
        { path: 'admin/users', name: 'admin-users', meta: { roles: ['admin'] }, component: () => import('../views/admin/UserManagementView.vue') },
        { path: 'admin/hospitals', name: 'admin-hospitals', meta: { roles: ['admin'] }, component: () => import('../views/admin/HospitalManagementView.vue') },

        // 监管审计看板（存证监控大屏后端同步授权 admin 查阅）
        { path: 'supervisor/overview', name: 'supervisor-overview', meta: { roles: ['supervisor', 'admin'] }, component: () => import('../views/supervisor/OverviewView.vue') },
        { path: 'supervisor/emergency', name: 'supervisor-emergency', meta: { roles: ['supervisor'] }, component: () => import('../views/supervisor/EmergencyAuditView.vue') },
        { path: 'supervisor/audits', name: 'supervisor-audits', meta: { roles: ['supervisor'] }, component: () => import('../views/supervisor/AuditLogView.vue') },
        { path: 'supervisor/verify', name: 'supervisor-verify', meta: { roles: ['supervisor'] }, component: () => import('../views/supervisor/VerifyView.vue') },

        // 个人中心 (通用)
        { path: 'personal', name: 'personal', component: () => import('../views/PersonalView.vue') },
      ],
    },
    // 404 兜底路由：访问无效路径自动回到当前角色首页或登录页
    {
      path: '/:pathMatch(.*)*',
      redirect: () => {
        const auth = useAuthStore()
        return auth.token ? homePathForRole(auth.user?.role, auth.user?.username) : '/login'
      },
    },
  ],
})

/** 依据角色返回对应工作台落地路径（登录成功与越权兜底共用） */
function homePathForRole(role?: string, username?: string): string {
  if (role === 'doctor') {
    return username?.startsWith('tech_') ? '/doctor/lab-center' : '/doctor/records'
  }
  if (role === 'patient') return '/patient/records'
  if (role === 'supervisor') return '/supervisor/overview'
  if (role === 'admin') return '/admin/users'
  return '/personal'
}

router.beforeEach((to, _from, next) => {
  const auth = useAuthStore()
  if (to.meta.requiresAuth && !auth.token) {
    next({ name: 'login' })
    return
  }
  if (to.name === 'login') {
    if (to.query.force_login) {
      next()
      return
    }
    if (auth.token) {
      next(homePathForRole(auth.user?.role, auth.user?.username))
      return
    }
  }
  // 角色准入：路由声明了 meta.roles 时，非授权角色一律重定向到自己的工作台。
  // 注意这只是 UX 层防护，数据安全仍由后端 RBAC 兜底（后端拒绝时接口返回 403）。
  const allowedRoles = to.meta.roles as string[] | undefined
  if (allowedRoles && allowedRoles.length > 0 && auth.token) {
    const role = auth.user?.role
    if (!role || !allowedRoles.includes(role)) {
      next(homePathForRole(role, auth.user?.username))
      return
    }
  }
  next()
})

export default router
