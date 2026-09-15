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
        { path: 'doctor/records', name: 'doctor-records', component: () => import('../views/doctor/DoctorRecordsView.vue') },
        { path: 'doctor/patient-search', name: 'doctor-patient-search', component: () => import('../views/doctor/PatientEmergencySearchView.vue') },
        { path: 'doctor/lab-center', name: 'doctor-lab-center', component: () => import('../views/doctor/LabCenterView.vue') },
        { path: 'doctor/cross-query', redirect: to => ({ path: '/doctor/query/all', query: to.query }) },
        { path: 'doctor/query', redirect: '/doctor/query/all' },
        { path: 'doctor/query/:scope', name: 'doctor-query-scope', component: () => import('../views/doctor/CrossQueryView.vue') },

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

        // 个人中心 (通用)
        { path: 'personal', name: 'personal', component: () => import('../views/PersonalView.vue') },
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
  if (to.name === 'login') {
    if (to.query.auto_login || to.query.force_login || to.query.username) {
      next()
      return
    }
    if (auth.token) {
      if (auth.user?.role === 'doctor') {
        if (auth.user.username?.startsWith('tech_')) next('/doctor/lab-center')
        else next('/doctor/records')
      }
      else if (auth.user?.role === 'patient') next('/patient/records')
      else if (auth.user?.role === 'supervisor') next('/supervisor/overview')
      else if (auth.user?.role === 'admin') next('/admin/users')
      else next('/doctor/records')
      return
    }
  }
  next()
})

export default router
