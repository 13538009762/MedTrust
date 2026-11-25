<template>
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
          <div class="hero-brand-pill">
            <img src="/emblem.png" class="hero-pill-icon" alt="Fabric & IPFS" />
            <span class="pill-dot"></span>
            <span>基于 Fabric 联盟链 · IPFS 密文存证 · 跨机构可信共享</span>
          </div>
          <h1 class="hero-title">
            可信存证 · 动态访问 · <span class="text-gradient">临床协同摘要</span>
          </h1>
          <p class="hero-subtitle">
            面向跨医院医疗协同共享场景，构建链下 IPFS 密文存储与 Fabric 账本锚定协同机制，打造轻量严密的急诊破窗紧急访问闭环。
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
            <div class="intro-card-left">
              <h3>医疗数据可信共享平台 2.0</h3>
              <p>
                面向第一人民医院（总院）、省立中心医院（跨域互认中心）、协和医学中心（远程协作分部）等多方参与的医联体协作网络。通过 AES-256-GCM 客户端加密与 IPFS 结合，有效规避账本膨胀；以统一安全网关收口权限校验，提供合规可溯源的急救访问闭环。
              </p>
            </div>
            <img src="/emblem.png" class="intro-card-emblem" alt="MedTrust 3D Shield" />
          </div>
        </div>
      </div>

      <!-- 右侧登录/注册舱 (极光白亚克力材质) -->
      <div class="login-panel">
        <div class="login-glass-card">
          <transition name="form-fade" mode="out-in">
            <!-- 登录模式 -->
            <div v-if="mode === 'login'" key="login-mode">
              <div class="card-header">
                <div class="card-header-emblem-wrap">
                  <img src="/emblem.png" class="card-header-emblem" alt="MedTrust" />
                </div>
                <h2>欢迎登录</h2>
                <p>基于数字身份的医疗可信访问网关</p>
                <div class="security-badges-row">
                  <span class="sec-badge"><el-icon><CircleCheck /></el-icon> 密码学可信鉴权</span>
                  <span class="sec-badge"><el-icon><KeyIcon /></el-icon> X.509 机构认证</span>
                </div>
              </div>

              <form class="login-form" @submit.prevent="handleLogin">
                <el-form-item>
                  <el-input
                    v-model="username"
                    placeholder="请输入账号名 / 执业医生证号 / 档案账号"
                    :prefix-icon="User"
                    size="large"
                    clearable
                  />
                </el-form-item>

                <el-form-item>
                  <el-input
                    v-model="password"
                    type="password"
                    placeholder="请输入系统登录密码"
                    :prefix-icon="Lock"
                    size="large"
                    show-password
                  />
                </el-form-item>

                <el-form-item>
                  <div class="captcha-wrapper">
                    <el-input
                      v-model="captchaAnswer"
                      placeholder="请输入图形安全计算验证码"
                      :prefix-icon="Finished"
                      size="large"
                      class="captcha-input"
                      autocomplete="off"
                    />
                    <div class="captcha-img-container" @click="fetchCaptcha" title="点击刷新验证码">
                      <img v-if="captchaImg" :src="captchaImg" alt="验证码" class="captcha-img" />
                      <span v-else-if="captchaLoading" class="captcha-loading">
                        <el-icon class="is-loading" style="margin-right: 2px;"><Loading /></el-icon> 加载中...
                      </span>
                      <span v-else class="captcha-loading" style="color: #64748b; font-size: 12px; display: inline-flex; align-items: center; gap: 4px;">
                        <el-icon><Refresh /></el-icon> 点击重试
                      </span>
                    </div>
                  </div>
                </el-form-item>

                <div class="form-aux-row">
                  <el-checkbox v-model="rememberUsername" size="small">记住账号</el-checkbox>
                  <el-link type="info" :underline="false" class="forgot-link" @click="handleForgotPwd">忘记密码 / 资质申诉？</el-link>
                </div>

                <el-button
                  type="primary"
                  class="login-btn"
                  :loading="loading"
                  native-type="submit"
                >
                  安 全 登 录
                </el-button>
              </form>

              <div class="register-footer">
                <span>还没有健康档案账户？</span>
                <el-link class="purple-link" :underline="false" @click="switchToRegister">
                  立即自主注册建档
                </el-link>
              </div>
              <div class="doctor-tip-footer">
                <span>执业医生账号涉及临床资质核验，须由医院管理员录入开设</span>
              </div>
            </div>

            <!-- 患者自主建档注册模式 -->
            <div v-else key="register-mode">
              <div class="card-header">
                <div class="card-header-emblem-wrap">
                  <img src="/emblem.png" class="card-header-emblem" alt="MedTrust" />
                </div>
                <h2>患者自主建档注册</h2>
                <p>开设个人医疗可信共享健康档案账号</p>
              </div>

              <el-alert
                title="特别规范提示：仅限就诊患者自主建档"
                type="info"
                description="临床执业医生及监管专员账号需严格核验医师执业证并经卫健审批，必须由医疗机构管理员统一开设，不支持公共注册。"
                show-icon
                :closable="false"
                class="mb-3"
              />

              <el-form :model="regForm" label-position="top" class="login-form" @submit.prevent="handleRegister">
                <el-row :gutter="10">
                  <el-col :span="12">
                    <el-form-item label="真实姓名" required class="reg-form-item">
                      <el-input
                        v-model="regForm.real_name"
                        placeholder="就诊人真实姓名"
                        :prefix-icon="User"
                      />
                    </el-form-item>
                  </el-col>
                  <el-col :span="12">
                    <el-form-item label="登录账号名" required class="reg-form-item">
                      <el-input
                        v-model="regForm.username"
                        placeholder="登录账号名 (如 pat_wang)"
                        :prefix-icon="Edit"
                      />
                    </el-form-item>
                  </el-col>
                </el-row>

                <el-form-item label="居民身份证号" required class="reg-form-item">
                  <el-input
                    v-model="regForm.id_card"
                    placeholder="18位居民身份证号（跨院精准防重名索引）"
                    :prefix-icon="CreditCard"
                    maxlength="18"
                  />
                  <div class="field-tip">用于跨医院就诊调阅时精准唯一定位健康档案，杜绝同名混淆</div>
                </el-form-item>

                <el-form-item label="联系手机号" required class="reg-form-item">
                  <el-input
                    v-model="regForm.phone"
                    placeholder="11位手机号码（接收急诊调阅即时通知）"
                    :prefix-icon="Iphone"
                    maxlength="11"
                  />
                  <div class="field-tip">当发生急诊抢救紧急破窗调阅时，系统将实时向您推送知情预警</div>
                </el-form-item>

                <el-row :gutter="10">
                  <el-col :span="12">
                    <el-form-item label="登录密码" required class="reg-form-item">
                      <el-input
                        v-model="regForm.password"
                        type="password"
                        placeholder="设置登录密码 (≥8位，含字母与数字)"
                        :prefix-icon="Lock"
                        show-password
                      />
                    </el-form-item>
                  </el-col>
                  <el-col :span="12">
                    <el-form-item label="确认密码" required class="reg-form-item">
                      <el-input
                        v-model="regForm.confirm_password"
                        type="password"
                        placeholder="确认登录密码"
                        :prefix-icon="Lock"
                        show-password
                      />
                    </el-form-item>
                  </el-col>
                </el-row>

                <el-button
                  type="primary"
                  class="login-btn"
                  :loading="regLoading"
                  native-type="submit"
                >
                  立 即 完 成 自 主 建 档
                </el-button>
              </el-form>

              <div class="register-footer">
                <span>已有健康档案账号？</span>
                <el-link class="purple-link" :underline="false" @click="mode = 'login'">
                  返回系统登录
                </el-link>
              </div>
            </div>
          </transition>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, watch } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { 
  User, Lock, Lock as KeyIcon, CircleCheck, Share,
  CreditCard, Iphone, Edit, Finished, Loading, Refresh
} from '@element-plus/icons-vue'
import { ElMessage, ElNotification, ElMessageBox } from 'element-plus'
import { getCaptchaApi, loginApi, registerApi } from '../api'
import { useAuthStore } from '../stores/auth'
import { validatePasswordStrength, isErrorToastShown, extractApiErrorMessage } from '../utils/passwordPolicy'

const mode = ref<'login' | 'register'>('login')
const username = ref('')
const password = ref('')
const loading = ref(false)
const rememberUsername = ref(false)
const captchaImg = ref('')
const captchaToken = ref('')
const captchaAnswer = ref('')
const captchaLoading = ref(false)

async function fetchCaptcha() {
  if (captchaLoading.value) return
  captchaLoading.value = true
  try {
    const res: any = await getCaptchaApi()
    if (res.code === 200 && res.data) {
      captchaImg.value = res.data.captcha_img
      captchaToken.value = res.data.captcha_token
      captchaAnswer.value = ''
    }
  } catch (err) {
    console.error('Failed to fetch captcha', err)
  } finally {
    captchaLoading.value = false
  }
}
const regLoading = ref(false)

const router = useRouter()
const route = useRoute()
const auth = useAuthStore()

onMounted(async () => {
  await fetchCaptcha()
  const savedUser = localStorage.getItem('medtrust_remembered_username')
  if (savedUser) {
    username.value = savedUser
    rememberUsername.value = true
  }
})

// 监听路由进入/回到登录页时自动刷新验证码
watch(
  () => route.fullPath,
  (newPath) => {
    if (newPath && newPath.includes('/login')) {
      fetchCaptcha()
    }
  },
  { immediate: true }
)

// 当从患者自主建档切回系统登录时，若无验证码也自动加载
watch(mode, (newMode) => {
  if (newMode === 'login' && !captchaImg.value) {
    fetchCaptcha()
  }
})

function handleForgotPwd() {
  ElMessageBox.alert(
    '根据国家医疗机构网络安全合规规范与等级保护（等保三级）要求，MedTrust 平台不设公开密码重置通道。临床医护人员、医技科室及监管专员如遗忘密码或出现数字证书异常，请携带有效执业证件至医院信息网络中心或医务部申请重置密码与资质复核；就诊患者可使用建档身份证件至院区便民服务台办理档案关联凭证重置。',
    '账户安全与密码申诉说明',
    {
      confirmButtonText: '已悉知',
      type: 'info',
      customClass: 'security-notice-box'
    }
  )
}

const regForm = ref({
  real_name: '',
  username: '',
  id_card: '',
  phone: '',
  password: '',
  confirm_password: '',
})

const features = [
  {
    title: '链下密文存储与 IPFS',
    desc: '医疗影像及病历附件经客户端本地对称加密后推至分布式 IPFS 节点，杜绝明文上云与网络账本膨胀。',
    icon: CircleCheck,
    bg: '#f0f9ff',
  },
  {
    title: 'Fabric 联盟链不可篡改锚定',
    desc: '核心数据指纹、智能合约授权凭证及生命周期事件均实时上链存证，确保医疗全链路可溯源抗抵赖。',
    icon: Share,
    bg: '#f0f9ff',
  },
  {
    title: '急诊抢救破窗放行',
    desc: '急危重症患者抢救时启动紧急访问通道，即时解密放行，并同步上链记录生成闭环监管审计工单。',
    icon: KeyIcon,
    bg: '#fef2f2',
  },
  {
    title: '受控医疗数据检索助理',
    desc: '严守医疗数据权限与合规边界，仅在当前登录身份获权范围内辅助检索病历与汇总临床线索。',
    icon: CircleCheck,
    bg: '#f0f9ff',
  },
]

function switchToRegister() {
  regForm.value = {
    real_name: '',
    username: '',
    id_card: '',
    phone: '',
    password: '',
    confirm_password: '',
  }
  mode.value = 'register'
}

async function handleLogin() {
  if (!username.value || !password.value) {
    ElMessage.warning('请输入用户名和密码')
    return
  }
  if (!captchaAnswer.value.trim()) {
    ElMessage.warning('请输入安全验证码计算结果，防止撞库爆破')
    return
  }

  loading.value = true
  try {
    const res: any = await loginApi({
      username: username.value,
      password: password.value,
      captcha_answer: captchaAnswer.value.trim(),
      captcha_token: captchaToken.value,
    })

    if (res.code === 200) {
      if (rememberUsername.value) {
        localStorage.setItem('medtrust_remembered_username', username.value)
      } else {
        localStorage.removeItem('medtrust_remembered_username')
      }

      auth.setAuth(res.data.token, res.data.user)
      ElMessage.success(`登录成功！欢迎回来，${res.data.user.real_name}`)

      const redirectTarget = typeof route.query.redirect === 'string' ? route.query.redirect : ''
      if (redirectTarget && redirectTarget.startsWith('/') && !redirectTarget.startsWith('//') && !redirectTarget.startsWith('/login')) {
        router.push(redirectTarget)
        return
      }

      const role = res.data.user.role
      if (role === 'doctor') {
        if (res.data.user.username?.startsWith('tech_')) {
          router.push('/doctor/lab-center')
        } else {
          router.push('/doctor/records')
        }
      } else if (role === 'patient') {
        router.push('/patient/records')
      } else if (role === 'supervisor') {
        router.push('/supervisor/overview')
      } else if (role === 'admin') {
        router.push('/admin/users')
      } else {
        router.push('/')
      }
    }
  } catch (err: unknown) {
    if (!isErrorToastShown(err)) {
      ElMessage.error(extractApiErrorMessage(err, '登录失败，请检查账号密码'))
    }
    fetchCaptcha()
  } finally {
    loading.value = false
  }
}

async function handleRegister() {
  if (!regForm.value.real_name) {
    ElMessage.warning('请填写真实姓名')
    return
  }
  if (!regForm.value.username) {
    ElMessage.warning('请设置登录账号名')
    return
  }
  if (!regForm.value.id_card || regForm.value.id_card.length < 15) {
    ElMessage.warning('请输入正确的18位居民身份证号（防重名建档必需）')
    return
  }
  if (!regForm.value.phone || regForm.value.phone.length < 11) {
    ElMessage.warning('请输入正确的11位手机号码')
    return
  }
  const pwdError = validatePasswordStrength(regForm.value.password)
  if (pwdError) {
    ElMessage.warning(pwdError)
    return
  }
  if (regForm.value.password !== regForm.value.confirm_password) {
    ElMessage.warning('两次输入的密码不一致，请核对')
    return
  }

  regLoading.value = true
  try {
    const res: any = await registerApi({
      real_name: regForm.value.real_name,
      username: regForm.value.username,
      id_card: regForm.value.id_card,
      phone: regForm.value.phone,
      password: regForm.value.password,
    })

    if (res.code === 200) {
      ElNotification({
        title: '建档注册成功',
        message: `恭喜患者 ${res.data?.real_name}，您的健康档案编号为 ${res.data?.user_no}，请直接登录！`,
        type: 'success',
        duration: 5000
      })
      // 自动切回登录并填充，同时刷新验证码避免长时间填表导致旧验证码过期
      username.value = regForm.value.username
      password.value = regForm.value.password
      mode.value = 'login'
      fetchCaptcha()
    }
  } catch (err: unknown) {
    if (!isErrorToastShown(err)) {
      ElMessage.error(extractApiErrorMessage(err, '注册失败'))
    }
  } finally {
    regLoading.value = false
  }
}
</script>

<style scoped>
.login-container {
  min-height: 100vh;
  width: 100vw;
  background-image: url('/images/v1.png');
  background-size: cover;
  background-position: center;
  background-repeat: no-repeat;
  display: flex;
  flex-direction: column;
  position: relative;
  overflow-x: hidden;
}

.top-nav {
  height: 72px;
  padding: 22px 72px 0;
  max-width: 1720px;
  width: 100%;
  margin: 0 auto;
  box-sizing: border-box;
  display: flex;
  align-items: center;
  justify-content: space-between;
  background: transparent;
  border: none;
  z-index: 10;
}

.logo-section {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-left: 20px;
}

.app-logo {
  width: 40px;
  height: 40px;
  border-radius: 6px;
  box-shadow: 0 1px 3px rgba(15, 23, 42, 0.08);
  object-fit: contain;
}

.logo-text {
  display: flex;
  align-items: center;
  gap: 10px;
}

.brand {
  font-size: 22px;
  font-weight: 800;
  color: var(--mt-primary, #0284c7);
  letter-spacing: -0.4px;
}

.divider {
  color: #94a3b8;
  font-weight: 300;
  font-size: 16px;
}

.app-name {
  font-size: 16px;
  font-weight: 650;
  color: #1e293b;
  letter-spacing: 0.2px;
}

.main-content {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 24px 72px 36px;
  max-width: 1720px;
  margin: 0 auto;
  width: 100%;
  box-sizing: border-box;
  gap: 52px;
}

.hero-area {
  flex: 1;
  max-width: 760px;
  margin-left: 20px;
}

.hero-brand-pill {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  padding: 6px 14px 6px 10px;
  background: #ffffff;
  border: 1px solid var(--mt-primary-border, #bae6fd);
  border-radius: 6px;
  font-size: 12.5px;
  font-weight: 600;
  color: var(--mt-primary, #0284c7);
  margin-bottom: 18px;
  box-shadow: var(--mt-shadow-sm, 0 1px 2px rgba(15, 23, 42, 0.04));
}

.hero-pill-icon {
  width: 20px;
  height: 20px;
  object-fit: contain;
}

.pill-dot {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: var(--mt-success, #059669);
  display: inline-block;
}

.hero-title {
  font-size: 38px;
  font-weight: 800;
  color: #0f172a;
  line-height: 1.25;
  margin-bottom: 16px;
  letter-spacing: -0.5px;
}

.text-gradient {
  color: var(--mt-primary, #0284c7);
}

.hero-subtitle {
  font-size: 15.5px;
  color: #334155;
  line-height: 1.65;
  margin-bottom: 28px;
}

.feature-list {
  display: flex;
  flex-direction: column;
  gap: 16px;
  margin-bottom: 28px;
}

.feature-item {
  display: flex;
  align-items: flex-start;
  gap: 14px;
}

.feature-icon {
  width: 38px;
  height: 38px;
  border-radius: 6px;
  display: flex;
  align-items: center;
  justify-content: center;
  color: var(--mt-primary, #0284c7);
  font-size: 18px;
  flex-shrink: 0;
  border: 1px solid var(--mt-primary-border, #bae6fd);
}

.feature-info h3 {
  font-size: 15.5px;
  font-weight: 700;
  color: #0f172a;
  margin: 0 0 4px 0;
}

.feature-info p {
  font-size: 13px;
  color: #475569;
  margin: 0;
  line-height: 1.5;
}

.system-intro-card {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 20px;
  padding: 18px 22px;
  background: #ffffff;
  border: 1px solid var(--mt-border-base, #e2e8f0);
  border-left: 4px solid var(--mt-primary, #0284c7);
  border-radius: 8px;
  box-shadow: var(--mt-shadow-sm, 0 1px 2px rgba(15, 23, 42, 0.04));
}

.intro-card-left {
  flex: 1;
}

.system-intro-card h3 {
  font-size: 15px;
  font-weight: 700;
  color: #0f172a;
  margin-bottom: 6px;
}

.system-intro-card p {
  font-size: 13px;
  color: #475569;
  line-height: 1.55;
  margin: 0;
}

.intro-card-emblem {
  width: 64px;
  height: 64px;
  object-fit: contain;
  flex-shrink: 0;
}

.login-panel {
  flex-shrink: 0;
}

.login-glass-card {
  width: clamp(400px, 32vw, 460px);
  padding: 32px 30px 26px;
  background: #ffffff !important;
  border: 1px solid var(--mt-border-base, #e2e8f0) !important;
  box-shadow: 0 4px 12px rgba(15, 23, 42, 0.06) !important;
  border-radius: 8px;
  box-sizing: border-box;
}

.card-header-emblem-wrap {
  margin-bottom: 10px;
  display: flex;
  align-items: center;
}

.card-header-emblem {
  width: 42px;
  height: 42px;
  object-fit: contain;
}

.card-header h2 {
  font-size: 22px;
  font-weight: 700;
  color: #0f172a;
  margin: 0 0 4px 0;
}

.card-header p {
  font-size: 13px;
  color: #64748b;
  margin: 0 0 10px 0;
}

.security-badges-row {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 20px;
}

.sec-badge {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  padding: 3px 8px;
  background: var(--mt-primary-soft, #f0f9ff);
  border: 1px solid var(--mt-primary-border, #bae6fd);
  border-radius: 4px;
  font-size: 11.5px;
  font-weight: 600;
  color: var(--mt-primary, #0284c7);
}

.sec-badge .el-icon {
  font-size: 13px;
}

.form-aux-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-top: -4px;
  margin-bottom: 16px;
  padding: 0 2px;
}

.forgot-link {
  font-size: 12.5px;
  color: #64748b !important;
  cursor: pointer;
  transition: color 0.2s ease;
}

.forgot-link:hover {
  color: var(--mt-primary, #0284c7) !important;
}

.mb-3 {
  margin-bottom: 14px;
}

.reg-form-item {
  margin-bottom: 12px;
}

.field-tip {
  font-size: 11px;
  color: #94a3b8;
  line-height: 1.3;
  margin-top: 2px;
}

.login-btn {
  width: 100%;
  height: 42px;
  border-radius: 6px;
  font-size: 14px;
  font-weight: 700;
  letter-spacing: 1px;
  background: var(--mt-primary, #0284c7) !important;
  border: 1px solid var(--mt-primary, #0284c7) !important;
  box-shadow: none !important;
  color: #ffffff !important;
  transition: background-color 0.2s ease;
  margin-top: 8px;
}

.login-btn:hover {
  background: var(--mt-primary-hover, #0369a1) !important;
  border-color: var(--mt-primary-hover, #0369a1) !important;
}

.register-footer {
  display: flex;
  justify-content: center;
  align-items: center;
  gap: 8px;
  margin-top: 16px;
  font-size: 13px;
  color: #64748b;
}

.doctor-tip-footer {
  text-align: center;
  margin-top: 8px;
  font-size: 11px;
  color: #94a3b8;
}

.purple-link {
  color: var(--mt-primary, #0284c7) !important;
  font-weight: 600;
  cursor: pointer;
}

.purple-link:hover {
  color: var(--mt-primary-hover, #0369a1) !important;
  text-decoration: underline;
}

/* 切换动画 */
.form-fade-enter-active,
.form-fade-leave-active {
  transition: opacity 0.25s ease, transform 0.25s ease;
}

.form-fade-enter-from {
  opacity: 0;
  transform: translateX(12px);
}

.form-fade-leave-to {
  opacity: 0;
  transform: translateX(-12px);
}

.captcha-wrapper {
  display: flex;
  align-items: center;
  gap: 12px;
  width: 100%;
}

.captcha-input {
  flex: 1;
}

.captcha-img-container {
  width: 124px;
  height: 40px;
  cursor: pointer;
  border-radius: 6px;
  overflow: hidden;
  border: 1px solid #e2e8f0;
  display: flex;
  align-items: center;
  justify-content: center;
  background: #f8fafc;
  transition: all 0.2s ease;
  user-select: none;
  flex-shrink: 0;
}

.captcha-img-container:hover {
  border-color: #10b981;
  box-shadow: 0 0 8px rgba(16, 185, 129, 0.2);
  transform: translateY(-1px);
}

.captcha-img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.captcha-loading {
  font-size: 12px;
  color: #10b981;
  font-weight: 500;
}

.reg-form-item :deep(.el-form-item__label) {
  padding-bottom: 4px;
  font-size: 12px;
  font-weight: 600;
  color: #334155;
  line-height: 1.3;
}

@media (max-width: 1080px) {
  .top-nav {
    padding: 16px 24px 0;
  }
  .logo-section {
    margin-left: 0;
  }
  .main-content {
    flex-direction: column-reverse;
    justify-content: flex-start;
    padding: 16px 24px 32px;
    gap: 24px;
  }
  .hero-area {
    margin-left: 0;
    max-width: 100%;
    width: 100%;
  }
  .login-panel {
    width: 100%;
    display: flex;
    justify-content: center;
  }
  .login-glass-card {
    width: min(460px, 100%);
  }
}

@media (max-width: 640px) {
  .top-nav {
    padding: 12px 16px 0;
    height: auto;
  }
  .app-name,
  .divider {
    display: none;
  }
  .main-content {
    padding: 12px 16px 24px;
  }
  .login-glass-card {
    padding: 22px 18px 20px;
  }
  .hero-title {
    font-size: 26px;
  }
  .system-intro-card {
    flex-direction: column;
    align-items: flex-start;
  }
}

</style>
