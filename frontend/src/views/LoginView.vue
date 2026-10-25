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
            <div class="intro-card-left">
              <h3>医疗数据可信共享平台 2.0</h3>
              <p>
                面向广州第一人民医院（总院）、深圳第二人民医院（跨域互认中心）、北京朝阳医院（远程协作分部）等多方参与的医联体协作网络。通过 AES-256-GCM 客户端加密与 IPFS 结合，有效规避账本膨胀；以统一安全网关收口权限校验，提供合规可溯源的急救访问闭环。
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
              </div>

              <!-- 医联体跨机构协作通道 · 一键免密快捷登入 -->
              <div class="demo-account-bar">
                <div class="demo-title-row">
                  <span class="demo-pulse-dot"></span>
                  <span class="demo-title">医联体跨机构协作通道 · 一键免密快捷登入 (Quick Access)</span>
                </div>

                <!-- 6 大核心标准内置账号快速卡片网格 -->
                <div class="defense-accounts-grid">
                  <button
                    v-for="acc in defenseAccounts"
                    :key="acc.username"
                    type="button"
                    :class="['defense-acc-btn', username === acc.username ? 'active-acc' : '']"
                    @click="fillAccount(acc.username)"
                    :title="`${acc.name} (${acc.hospital} · ${acc.dept})`"
                  >
                    <div class="dab-top">
                      <span class="dab-name">{{ acc.name }}</span>
                      <el-tag size="small" :type="acc.tagType" effect="dark" class="dab-tag">{{ acc.roleLabel }}</el-tag>
                    </div>
                    <div class="dab-sub">{{ acc.dept }}</div>
                    <div class="dab-id"><code>{{ acc.username }}</code></div>
                  </button>
                </div>

                <!-- 可折叠展开更多辅助测试账号 -->
                <div class="extra-accounts-wrapper">
                  <div class="extra-toggle" @click="showExtraAccounts = !showExtraAccounts">
                    <span>{{ showExtraAccounts ? '收起协作医疗机构与医技账号' : '切换更多协作医疗机构与医技检查中心账号' }}</span>
                    <el-icon class="extra-arrow" :class="{ 'is-open': showExtraAccounts }"><ArrowDown /></el-icon>
                  </div>
                  <div v-show="showExtraAccounts" class="extra-accounts-content">
                    <div class="demo-tabs">
                      <button
                        type="button"
                        :class="['demo-tab-item', activeDemoHosp === 'HOSP_A' ? 'active' : '']"
                        @click="activeDemoHosp = 'HOSP_A'"
                      >第一医院</button>
                      <button
                        type="button"
                        :class="['demo-tab-item', activeDemoHosp === 'HOSP_B' ? 'active' : '']"
                        @click="activeDemoHosp = 'HOSP_B'"
                      >第二医院</button>
                      <button
                        type="button"
                        :class="['demo-tab-item', activeDemoHosp === 'HOSP_C' ? 'active' : '']"
                        @click="activeDemoHosp = 'HOSP_C'"
                      >第三医院</button>
                      <button
                        type="button"
                        :class="['demo-tab-item', activeDemoHosp === 'OTHERS' ? 'active' : '']"
                        @click="activeDemoHosp = 'OTHERS'"
                      >患者/监管</button>
                    </div>

                    <div v-if="activeDemoHosp === 'HOSP_A'" class="demo-buttons">
                      <el-button size="small" type="primary" plain @click="fillAccount('doc_li')">李主任 (心内科)</el-button>
                      <el-button size="small" type="warning" plain @click="fillAccount('tech_lab_a')">李主管 (检验科)</el-button>
                      <el-button size="small" type="warning" plain @click="fillAccount('tech_pacs_a')">王医生 (影像科)</el-button>
                    </div>
                    <div v-else-if="activeDemoHosp === 'HOSP_B'" class="demo-buttons">
                      <el-button size="small" type="primary" plain @click="fillAccount('doc_wang')">王医生 (心内科)</el-button>
                      <el-button size="small" type="warning" plain @click="fillAccount('doc_chen')">陈医生 (急诊科)</el-button>
                      <el-button size="small" type="warning" plain @click="fillAccount('tech_lab_b')">刘主管 (检验科)</el-button>
                    </div>
                    <div v-else-if="activeDemoHosp === 'HOSP_C'" class="demo-buttons">
                      <el-button size="small" type="primary" plain @click="fillAccount('doc_c')">陈医生 (神经内科)</el-button>
                      <el-button size="small" type="warning" plain @click="fillAccount('tech_lab_c')">孙主管 (检验科)</el-button>
                    </div>
                    <div v-else class="demo-buttons">
                      <el-button size="small" type="success" plain @click="fillAccount('pat_zhang')">患者张伟/张三</el-button>
                      <el-button size="small" type="success" plain @click="fillAccount('pat_qian')">患者钱小芳</el-button>
                      <el-button size="small" type="info" plain @click="fillAccount('sup_zhao')">监管赵主任</el-button>
                      <el-button size="small" type="danger" plain @click="fillAccount('admin')">系统管理员</el-button>
                    </div>
                  </div>
                </div>
              </div>

              <form class="login-form" @submit.prevent="handleLogin">
                <el-form-item>
                  <el-input
                    v-model="username"
                    placeholder="请输入用户名 (如 doc_a, pat_zhang)"
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
                  />
                </el-form-item>

                <el-form-item>
                  <div class="captcha-wrapper">
                    <el-input
                      v-model="captchaAnswer"
                      placeholder="请输入安全计算验证码"
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

                <el-button
                  type="primary"
                  class="login-btn"
                  :loading="loading"
                  native-type="submit"
                >
                  进 入 系 统
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

              <el-form :model="regForm" class="login-form" @submit.prevent="handleRegister">
                <el-row :gutter="10">
                  <el-col :span="12">
                    <el-form-item class="reg-form-item">
                      <el-input
                        v-model="regForm.real_name"
                        placeholder="就诊人真实姓名"
                        :prefix-icon="User"
                      />
                    </el-form-item>
                  </el-col>
                  <el-col :span="12">
                    <el-form-item class="reg-form-item">
                      <el-input
                        v-model="regForm.username"
                        placeholder="登录账号名 (如 pat_wang)"
                        :prefix-icon="Edit"
                      />
                    </el-form-item>
                  </el-col>
                </el-row>

                <el-form-item class="reg-form-item">
                  <el-input
                    v-model="regForm.id_card"
                    placeholder="18位居民身份证号（跨院精准防重名索引）"
                    :prefix-icon="CreditCard"
                    maxlength="18"
                  />
                  <div class="field-tip">用于跨医院就诊调阅时精准唯一定位健康档案，杜绝同名混淆</div>
                </el-form-item>

                <el-form-item class="reg-form-item">
                  <el-input
                    v-model="regForm.phone"
                    placeholder="11位手机号码（接收急诊调阅即时通知）"
                    :prefix-icon="Iphone"
                    maxlength="11"
                  />
                  <div class="field-tip">当发生 Break-Glass 紧急抢救破窗调阅时，系统将向该手机推送知情预警</div>
                </el-form-item>

                <el-row :gutter="10">
                  <el-col :span="12">
                    <el-form-item class="reg-form-item">
                      <el-input
                        v-model="regForm.password"
                        type="password"
                        placeholder="设置登录密码 (≥6位)"
                        :prefix-icon="Lock"
                        show-password
                      />
                    </el-form-item>
                  </el-col>
                  <el-col :span="12">
                    <el-form-item class="reg-form-item">
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
  User, Lock, Lock as KeyIcon, CircleCheck, Share, MagicStick, 
  CreditCard, Iphone, Edit, ArrowDown, Finished, Loading, Refresh
} from '@element-plus/icons-vue'
import { ElMessage, ElNotification } from 'element-plus'
import api from '../api/client'
import { useAuthStore } from '../stores/auth'

const mode = ref<'login' | 'register'>('login')
const username = ref('doc_li')
const password = ref('123456')
const loading = ref(false)
const captchaImg = ref('')
const captchaToken = ref('')
const captchaAnswer = ref('')
const captchaLoading = ref(false)

async function fetchCaptcha() {
  if (captchaLoading.value) return
  captchaLoading.value = true
  try {
    const res: any = await api.get('/auth/captcha')
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
const activeDemoHosp = ref('HOSP_A')
const showExtraAccounts = ref(false)

const router = useRouter()
const route = useRoute()
const auth = useAuthStore()

// 医联体跨机构协作通道内置核心账号
const defenseAccounts = [
  { username: 'doc_li', name: '李建国主任', roleLabel: '总院区 (Org1)', tagType: 'primary', hospital: '广州第一附属医院', dept: '心内科' },
  { username: 'doc_wang', name: '王明德医生', roleLabel: '分院区 (Org2)', tagType: 'primary', hospital: '深圳中心医院', dept: '心内科' },
  { username: 'doc_chen', name: '陈建明医生', roleLabel: '深圳急诊 (Org2)', tagType: 'warning', hospital: '深圳中心医院', dept: '急诊科' },
  { username: 'pat_zhang', name: '患者张三/张伟', roleLabel: '患者主权端', tagType: 'success', hospital: '患者个人中心', dept: '档案主权' },
  { username: 'sup_zhao', name: '监管赵主任', roleLabel: '合规监管端', tagType: 'danger', hospital: '国家卫健委', dept: '审计中心' },
  { username: 'admin', name: '系统管理员', roleLabel: '平台运维端', tagType: 'info', hospital: 'MedTrust 平台', dept: '技术主管' },
]

const accountDescMap: Record<string, string> = {
  'doc_li': '广州第一附属医院 (Org1 节点) · 李建国主任 (心内科主任医师)',
  'doc_wang': '深圳中心医院 (Org2 节点) · 王明德医生 (心内科主治医师)',
  'doc_chen': '深圳中心医院 (Org2 节点) · 陈建明医生 (急诊科主治医师)',
  'pat_zhang': '就诊患者 · 张三/张伟 (健康档案主权持有者)',
  'sup_zhao': '国家卫生健康委 · 赵主任 (医疗监管与防篡改审计专员)',
  'admin': 'MedTrust 平台 · 系统管理员 (平台技术主管)',
  'doc_a': '第一人民医院 (Org1) · 李建国医生 (心内科主任医师)',
  'tech_lab_a': '第一人民医院 (Org1) · 李主管 (医学检验科主管技师)',
  'tech_pacs_a': '第一人民医院 (Org1) · 王医生 (放射影像中心主治医师)',
  'doc_b': '省立中心医院 (Org2) · 王明德医生 (骨创伤科主治医师)',
  'tech_lab_b': '省立中心医院 (Org2) · 刘主管 (医学检验科主管技师)',
  'tech_pacs_b': '省立中心医院 (Org2) · 赵医生 (放射影像中心副主任技师)',
  'doc_c': '协和医学中心 (Org3) · 陈晓华医生 (神经内科副主任医师)',
  'tech_lab_c': '协和医学中心 (Org3) · 孙主管 (医学检验科主管技师)',
  'tech_pacs_c': '协和医学中心 (Org3) · 钱医生 (放射影像中心主治医师)',
  'pat_qian': '患者钱小芳 (就诊档案持有者)',
  'supervisor': '卫健监管专员 (医疗合规全链路督察)'
}

onMounted(async () => {
  await fetchCaptcha()
  const autoUser = (route.query.auto_login || route.query.username) as string
  if (autoUser) {
    username.value = autoUser
    password.value = '123456'
    if (route.query.auto_login) {
      await handleLogin()
    }
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

function fillAccount(acc: string) {
  username.value = acc
  password.value = '123456'
  const desc = accountDescMap[acc] || acc
  ElMessage.info(`已快捷载入：${desc}`)
  if (!captchaImg.value) {
    fetchCaptcha()
  }
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
    bg: '#ede9fe',
  },
  {
    title: 'Fabric 联盟链不可篡改锚定',
    desc: '核心数据指纹、智能合约授权凭证及生命周期事件均实时上链存证，确保医疗全链路可溯源抗抵赖。',
    icon: Share,
    bg: '#e0e7ff',
  },
  {
    title: 'Break-Glass 破窗急救放行',
    desc: '急危重症患者抢救时启动紧急访问通道，毫秒级解密放行，并即刻异步上链记录生成闭环审计工单。',
    icon: KeyIcon,
    bg: '#fee2e2',
  },
  {
    title: '受控医疗 AI 智能助理',
    desc: '严守工程与合规边界，AI 仅作为自然语言意图转换器，携带当前 JWT 经由 Go 统一网关鉴权交互。',
    icon: MagicStick,
    bg: '#fdf4ff',
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
    const res: any = await api.post('/auth/login', {
      username: username.value,
      password: password.value,
      captcha_answer: captchaAnswer.value.trim(),
      captcha_token: captchaToken.value,
    })

    if (res.code === 200) {
      auth.setAuth(res.data.token, res.data.user)
      ElMessage.success(`登录成功！欢迎回来，${res.data.user.real_name}`)

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
  } catch (err: any) {
    ElMessage.error(err?.message || '登录失败，请检查账号密码')
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
  if (!regForm.value.password || regForm.value.password.length < 6) {
    ElMessage.warning('登录密码长度不能少于 6 位')
    return
  }
  if (regForm.value.password !== regForm.value.confirm_password) {
    ElMessage.warning('两次输入的密码不一致，请核对')
    return
  }

  regLoading.value = true
  try {
    const res: any = await api.post('/auth/register', {
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
      // 自动切回登录并填充
      username.value = regForm.value.username
      password.value = regForm.value.password
      mode.value = 'login'
    }
  } catch (err: any) {
    ElMessage.error(err?.message || '注册失败')
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
  border-radius: 11px;
  box-shadow: 0 4px 14px rgba(14, 165, 233, 0.25);
  transition: transform 0.3s cubic-bezier(0.4, 0, 0.2, 1);
  object-fit: contain;
}
.app-logo:hover {
  transform: scale(1.08);
}

.logo-text {
  display: flex;
  align-items: center;
  gap: 10px;
}

.brand {
  font-size: 22px;
  font-weight: 850;
  color: #4f46e5;
  letter-spacing: -0.5px;
  text-shadow: 0 1px 2px rgba(255, 255, 255, 0.85);
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
  text-shadow: 0 1px 2px rgba(255, 255, 255, 0.85);
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
  gap: 9px;
  padding: 7px 16px 7px 12px;
  background: rgba(255, 255, 255, 0.75);
  backdrop-filter: blur(12px);
  border: 1px solid rgba(99, 102, 241, 0.25);
  border-radius: 30px;
  font-size: 13px;
  font-weight: 600;
  color: #6366f1;
  margin-bottom: 20px;
  box-shadow: 0 4px 14px rgba(99, 102, 241, 0.08);
}

.hero-pill-icon {
  width: 22px;
  height: 22px;
  object-fit: contain;
  filter: drop-shadow(0 2px 4px rgba(14, 165, 233, 0.3));
}

.pill-dot {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: #10b981;
  box-shadow: 0 0 6px #10b981;
  display: inline-block;
}

.hero-title {
  font-size: 42px;
  font-weight: 850;
  color: #0f172a;
  line-height: 1.25;
  margin-bottom: 18px;
  letter-spacing: -0.6px;
}

.text-gradient {
  background: linear-gradient(135deg, #6366f1 0%, #8b5cf6 50%, #ec4899 100%);
  -webkit-background-clip: text;
  -webkit-text-fill-color: transparent;
}

.hero-subtitle {
  font-size: 16.5px;
  color: #334155;
  line-height: 1.7;
  margin-bottom: 34px;
}

.feature-list {
  display: flex;
  flex-direction: column;
  gap: 20px;
  margin-bottom: 34px;
}

.feature-item {
  display: flex;
  align-items: flex-start;
  gap: 16px;
}

.feature-icon {
  width: 42px;
  height: 42px;
  border-radius: 12px;
  display: flex;
  align-items: center;
  justify-content: center;
  color: #6366f1;
  font-size: 20px;
  flex-shrink: 0;
  box-shadow: 0 4px 12px rgba(99, 102, 241, 0.1);
}

.feature-info h3 {
  font-size: 16.5px;
  font-weight: 750;
  color: #0f172a;
  margin: 0 0 6px 0;
}

.feature-info p {
  font-size: 13.5px;
  color: #475569;
  margin: 0;
  line-height: 1.55;
}

.system-intro-card {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 24px;
  padding: 22px 28px;
  background: rgba(255, 255, 255, 0.65);
  backdrop-filter: blur(14px);
  border: 1px solid rgba(255, 255, 255, 0.85);
  border-radius: 20px;
  box-shadow: 0 14px 32px rgba(99, 102, 241, 0.05);
}

.intro-card-left {
  flex: 1;
}

.system-intro-card h3 {
  font-size: 16px;
  font-weight: 750;
  color: #0f172a;
  margin-bottom: 8px;
}

.system-intro-card p {
  font-size: 13.5px;
  color: #475569;
  line-height: 1.6;
  margin: 0;
}

.intro-card-emblem {
  width: 72px;
  height: 72px;
  object-fit: contain;
  flex-shrink: 0;
  filter: drop-shadow(0 6px 16px rgba(14, 165, 233, 0.35));
  animation: emblemFloat 4s ease-in-out infinite;
}

@keyframes emblemFloat {
  0%, 100% {
    transform: translateY(0px) rotate(0deg);
  }
  50% {
    transform: translateY(-6px) rotate(2deg);
  }
}

.login-panel {
  flex-shrink: 0;
}

.login-glass-card {
  width: clamp(380px, 32vw, 460px);
  padding: 32px 28px;
  background: rgba(255, 255, 255, 0.6) !important;
  backdrop-filter: blur(24px) !important;
  border: 1px solid rgba(255, 255, 255, 0.8) !important;
  box-shadow: 0 20px 40px rgba(139, 92, 246, 0.08) !important;
  border-radius: 24px;
  box-sizing: border-box;
}

.card-header-emblem-wrap {
  margin-bottom: 12px;
  display: flex;
  align-items: center;
}

.card-header-emblem {
  width: 46px;
  height: 46px;
  object-fit: contain;
  filter: drop-shadow(0 4px 12px rgba(14, 165, 233, 0.35));
  transition: transform 0.3s cubic-bezier(0.4, 0, 0.2, 1);
}

.card-header-emblem:hover {
  transform: scale(1.1) rotate(5deg);
}

.card-header h2 {
  font-size: 26px;
  font-weight: 800;
  color: #0f172a;
  margin: 0 0 6px 0;
}

.card-header p {
  font-size: 13px;
  color: #64748b;
  margin-bottom: 18px;
}

.mb-3 {
  margin-bottom: 14px;
}

.demo-account-bar {
  background: rgba(237, 233, 254, 0.7);
  border: 1px solid rgba(139, 92, 246, 0.35);
  border-radius: 14px;
  padding: 10px 12px;
  margin-bottom: 16px;
  box-shadow: 0 4px 12px rgba(139, 92, 246, 0.06);
}

.demo-title-row {
  display: flex;
  align-items: center;
  gap: 6px;
  margin-bottom: 8px;
}

.demo-pulse-dot {
  width: 7px;
  height: 7px;
  border-radius: 50%;
  background: #10b981;
  box-shadow: 0 0 0 2px rgba(16, 185, 129, 0.2);
  display: inline-block;
  animation: pulse 2s infinite;
}

.demo-title {
  font-size: 11px;
  font-weight: 700;
  color: #6d28d9;
}

.defense-accounts-grid {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 6px;
  margin-bottom: 8px;
}

.defense-acc-btn {
  background: #ffffff;
  border: 1px solid rgba(226, 232, 240, 0.9);
  border-radius: 8px;
  padding: 6px 8px;
  text-align: left;
  cursor: pointer;
  transition: all 0.2s cubic-bezier(0.4, 0, 0.2, 1);
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.defense-acc-btn:hover {
  transform: translateY(-1px);
  border-color: #8b5cf6;
  box-shadow: 0 3px 8px rgba(139, 92, 246, 0.15);
}

.defense-acc-btn.active-acc {
  background: #f5f3ff;
  border-color: #7c3aed;
  box-shadow: 0 0 0 2px rgba(124, 58, 237, 0.2);
}

.dab-top {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 4px;
}

.dab-name {
  font-size: 11.5px;
  font-weight: 700;
  color: #1e293b;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.dab-tag {
  font-size: 9px;
  height: 16px;
  line-height: 16px;
  padding: 0 4px;
  border-radius: 4px;
}

.dab-sub {
  font-size: 10px;
  color: #64748b;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.dab-id {
  font-size: 10px;
  color: #7c3aed;
  font-weight: 600;
}

.extra-accounts-wrapper {
  border-top: 1px dashed rgba(167, 139, 250, 0.4);
  padding-top: 6px;
  margin-top: 4px;
}

.extra-toggle {
  display: flex;
  align-items: center;
  justify-content: space-between;
  font-size: 10.5px;
  color: #7c3aed;
  cursor: pointer;
  padding: 2px 4px;
  border-radius: 4px;
  user-select: none;
}

.extra-toggle:hover {
  background: rgba(255, 255, 255, 0.6);
}

.extra-arrow {
  transition: transform 0.25s ease;
}

.extra-arrow.is-open {
  transform: rotate(180deg);
}

.extra-accounts-content {
  margin-top: 6px;
}

.demo-tabs {
  display: flex;
  gap: 3px;
  margin-bottom: 8px;
  background: rgba(241, 245, 249, 0.9);
  padding: 3px;
  border-radius: 8px;
}

.demo-tab-item {
  flex: 1;
  border: none;
  background: transparent;
  padding: 4px 4px;
  font-size: 11px;
  font-weight: 600;
  color: #64748b;
  border-radius: 6px;
  cursor: pointer;
  transition: all 0.2s ease;
  white-space: nowrap;
  text-align: center;
}

.demo-tab-item:hover {
  color: #475569;
}

.demo-tab-item.active {
  background: white;
  color: #7c3aed;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.1);
}

.demo-buttons {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}

.demo-btn-box {
  display: inline-flex;
  align-items: center;
  background: white;
  padding: 1px 4px;
  border-radius: 6px;
  border: 1px solid rgba(226, 232, 240, 0.9);
  box-shadow: 0 1px 2px rgba(0, 0, 0, 0.03);
}

.demo-btn-box .el-button--link {
  padding: 0 2px;
  margin-left: 2px;
  font-size: 11px;
}

.concurrency-banner {
  margin-bottom: 8px;
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
  height: 46px;
  border-radius: 10px;
  font-size: 15px;
  font-weight: 700;
  letter-spacing: 2px;
  background: linear-gradient(135deg, #a78bfa 0%, #8b5cf6 100%) !important;
  border: none;
  box-shadow: 0 8px 16px rgba(139, 92, 246, 0.25) !important;
  color: white !important;
  transition: all 0.3s cubic-bezier(0.4, 0, 0.2, 1);
  margin-top: 8px;
}

.login-btn:hover {
  transform: translateY(-2px);
  box-shadow: 0 12px 24px rgba(139, 92, 246, 0.35) !important;
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
  color: #7c3aed !important;
  font-weight: 600;
  cursor: pointer;
}

.purple-link:hover {
  color: #6d28d9 !important;
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

</style>
