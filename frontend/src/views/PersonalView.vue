<template>
  <div class="personal-container">
    <!-- Hero 渐变横幅 -->
    <div class="hero-header">
      <div class="header-left">
        <div class="header-icon-ring">
          <el-icon><User /></el-icon>
        </div>
        <div>
          <h1 class="page-title">个人中心 (My Profile)</h1>
          <p class="page-sub">查看并维护您的个人身份档案、医疗执业资质以及跨院防重名唯一标识</p>
        </div>
      </div>
      <div class="header-right-chip">
        <span class="chain-status-dot"></span>
        <span>联盟链身份已锚定生效</span>
      </div>
    </div>

    <!-- 个人资料主卡片 -->
    <div class="profile-card" v-loading="loading">
      <!-- 1. 顶部身份横幅 -->
      <div class="profile-hero-section">
        <!-- 头像区 -->
        <div class="avatar-col" @click="handleAvatarClick" title="更换个人头像">
          <div class="user-avatar-circle" :style="{ background: avatarGradient }">
            <span class="user-avatar-text">{{ userInitials }}</span>
            <div class="avatar-hover-overlay">
              <el-icon class="camera-icon"><Camera /></el-icon>
              <span class="change-hint-text">更换头像</span>
            </div>
          </div>
          <el-button link type="primary" size="small" class="change-avatar-btn">
            更换头像
          </el-button>
        </div>

        <!-- 核心身份信息区 -->
        <div class="identity-col">
          <div class="identity-name-row">
            <h2 class="user-title-name">{{ auth.user?.real_name || '未填写真实姓名' }}</h2>
            <el-tag size="default" :type="getRoleTagType(auth.user?.role)" effect="dark" class="role-badge">
              {{ getRoleName(auth.user?.role) }}
            </el-tag>
            <el-tag size="small" :type="auth.user?.status === 'NORMAL' ? 'success' : 'danger'" effect="light" class="status-tag">
              <span class="status-dot" :class="{ 'dot-danger': auth.user?.status !== 'NORMAL' }"></span>
              {{ auth.user?.status === 'NORMAL' ? '正常在网' : '受到监管惩戒' }}
            </el-tag>
          </div>

          <div class="identity-sub-row">
            <span class="identity-chip">
              <span class="chip-label">登录账号:</span>
              <span class="chip-val font-mono">@{{ auth.user?.username || '—' }}</span>
            </span>
            <span class="identity-chip">
              <span class="chip-label">电子工号/档案号:</span>
              <code class="emp-code">{{ auth.user?.user_no || '—' }}</code>
            </span>
            <span v-if="auth.user?.hospital_name" class="identity-chip">
              <span class="chip-label">归属机构:</span>
              <span class="chip-val">{{ auth.user?.hospital_name }}</span>
            </span>
          </div>

          <!-- 资质与科室标签栏 -->
          <div class="dept-bar">
            <span class="dept-label">资质/专业科室:</span>
            <div class="dept-pill">
              <span class="dept-name">{{ auth.user?.department_name || (auth.user?.role === 'patient' ? '全民健康服务' : '行政管理科室') }}</span>
              <el-tag size="small" type="info" effect="plain" class="title-tag">
                {{ auth.user?.title || (auth.user?.role === 'patient' ? '实名电子就诊人' : '医疗业务人员') }}
              </el-tag>
            </div>
          </div>
        </div>

        <!-- 右侧操作按钮组 -->
        <div class="actions-col">
          <el-button type="primary" :icon="Edit" class="btn-action-primary" @click="openEditDialog">
            编辑个人信息
          </el-button>
          <el-button type="success" :icon="Key" class="btn-action-primary" @click="openMedicalKeyDialog">
            修改病历调阅密钥
          </el-button>
          <el-button type="warning" plain :icon="Lock" class="btn-action-secondary" @click="openPassDialog">
            修改登录密码
          </el-button>
        </div>
      </div>

      <el-divider class="profile-divider" />

      <!-- 2. 双栏多维结构化资料卡片 -->
      <div class="info-blocks-grid">
        <!-- 卡片 1: 医疗机构与专业资质 -->
        <div class="info-block-card">
          <div class="block-card-header">
            <span class="block-title">医疗机构与专业资质</span>
          </div>
          <div class="block-card-body">
            <div class="field-item">
              <span class="field-label">所属医疗机构</span>
              <div class="field-val font-bold">
                {{ auth.user?.hospital_name || '跨医疗机构统一联合网络' }}
              </div>
            </div>

            <div class="field-item">
              <span class="field-label">临床就诊科室</span>
              <div class="field-val">
                <el-tag size="small" type="primary" effect="plain">
                  {{ auth.user?.department_name || (auth.user?.role === 'patient' ? '门急诊全科' : '综合业务管理中心') }}
                </el-tag>
              </div>
            </div>

            <div class="field-item">
              <span class="field-label">系统安全角色</span>
              <div class="field-val">
                <el-tag size="default" :type="getRoleTagType(auth.user?.role)" effect="light">
                  {{ getRoleName(auth.user?.role) }}
                </el-tag>
              </div>
            </div>

            <div class="field-item">
              <span class="field-label">临床职称 / 身份</span>
              <div class="field-val font-bold">
                {{ auth.user?.title || (auth.user?.role === 'patient' ? '实名制就诊人 (持身份证建档)' : '系统工作人员') }}
              </div>
            </div>

            <div class="field-item">
              <span class="field-label">联盟链节点 MSP</span>
              <div class="field-val font-mono text-muted text-xs">
                MedTrust-Hosp{{ auth.user?.hospital_id || 1 }}MSP::Peer0
              </div>
            </div>
          </div>
        </div>

        <!-- 卡片 2: 身份认证与安全资料 -->
        <div class="info-block-card">
          <div class="block-card-header">
            <span class="block-title">身份认证与跨院防重名索引</span>
          </div>
          <div class="block-card-body">
            <div class="field-item">
              <span class="field-label">居民身份证号</span>
              <div class="field-val contact-val">
                <span class="font-mono text-primary font-bold">
                  {{ auth.user?.id_card || '尚未录入' }}
                </span>
                <el-button
                  v-if="auth.user?.id_card"
                  link
                  type="primary"
                  size="small"
                  :icon="CopyDocument"
                  @click="copyText(auth.user?.id_card, '身份证号')"
                  title="复制身份证号"
                />
              </div>
            </div>

            <div class="field-item">
              <span class="field-label">关联联系手机</span>
              <div class="field-val contact-val">
                <span class="font-mono font-bold">
                  {{ auth.user?.phone || '尚未绑定' }}
                </span>
                <el-button
                  v-if="auth.user?.phone"
                  link
                  type="primary"
                  size="small"
                  :icon="CopyDocument"
                  @click="copyText(auth.user?.phone, '手机号')"
                  title="复制手机号"
                />
              </div>
            </div>

            <div class="field-item">
              <span class="field-label">跨院病历调阅专属密钥</span>
              <div class="field-val contact-val">
                <span class="font-mono font-bold" :style="{ color: showKeyPlain ? '#059669' : '#64748b' }">
                  {{ showKeyPlain ? (auth.user?.medical_key || '123456') : '••••••' }}
                </span>
                <el-button
                  link
                  type="primary"
                  size="small"
                  :icon="showKeyPlain ? View : Hide"
                  @click="showKeyPlain = !showKeyPlain"
                  :title="showKeyPlain ? '隐藏密钥' : '显示明文密钥'"
                />
                <el-button
                  type="success"
                  link
                  size="small"
                  @click="openMedicalKeyDialog"
                >
                  修改密钥
                </el-button>
              </div>
            </div>

            <div class="field-item">
              <span class="field-label">跨院防重名索引</span>
              <div class="field-val">
                <el-tag size="small" type="success" effect="plain">
                  已开启跨机构精准唯一匹配
                </el-tag>
              </div>
            </div>

            <div class="field-item">
              <span class="field-label">破窗知情推送</span>
              <div class="field-val text-muted text-xs">
                当发生 Break-Glass 紧急访问时实时推送知情通知
              </div>
            </div>

            <div class="field-item">
              <span class="field-label">首次系统建档时间</span>
              <div class="field-val text-muted font-mono text-xs">
                2026-09-01 00:00:00 (电子病历存证生效)
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- 业务数据统计卡片 -->
    <div class="stats-section">
      <div class="stats-header-bar">
        <h3 class="stats-title">医疗数据与业务互联统计</h3>
      </div>
      <div class="stats-grid">
        <el-card class="stat-card" shadow="hover">
          <template #header>
            <div class="stat-card-header">
              <span>{{ auth.user?.role === 'doctor' ? '累计开具存证病历' : (auth.user?.role === 'patient' ? '个人健康档案总数' : '全网存证病历资产') }}</span>
            </div>
          </template>
          <div class="stat-value text-purple">{{ auth.user?.role === 'doctor' ? '4' : (auth.user?.role === 'patient' ? '4' : '12') }} <span class="stat-unit">份</span></div>
        </el-card>

        <el-card class="stat-card" shadow="hover">
          <template #header>
            <div class="stat-card-header">
              <span>{{ auth.user?.role === 'patient' ? '有效授权策略凭证' : '跨院调阅与安全协同' }}</span>
            </div>
          </template>
          <div class="stat-value text-green">{{ auth.user?.role === 'patient' ? '1' : '3' }} <span class="stat-unit">项</span></div>
        </el-card>

        <el-card class="stat-card" shadow="hover">
          <template #header>
            <div class="stat-card-header">
              <span>{{ auth.user?.role === 'patient' ? '收到抢救破窗预警' : 'Break-Glass 破窗急救' }}</span>
            </div>
          </template>
          <div class="stat-value text-amber">1 <span class="stat-unit">次</span></div>
        </el-card>
      </div>
    </div>

    <!-- 3. 编辑个人信息弹窗 (Edit Profile Dialog) -->
    <el-dialog 
      title="编辑个人档案与联系信息" 
      v-model="editVisible" 
      width="540px"
      class="personal-edit-dialog"
      :close-on-click-modal="false"
    >
      <el-alert
        title="提示：更新手机号或身份证号后，跨机构就诊检索与急救调阅推送将立即同步更新生效。"
        type="info"
        show-icon
        :closable="false"
        class="mb-3"
      />

      <el-form :model="editForm" label-position="top" class="edit-form">
        <el-form-item label="真实姓名" required>
          <el-input v-model="editForm.real_name" placeholder="请输入真实姓名" :prefix-icon="User" />
        </el-form-item>

        <el-form-item label="18位居民身份证号（跨院防同名混淆精准主键）" required>
          <el-input 
            v-model="editForm.id_card" 
            placeholder="请输入18位身份证号" 
            :prefix-icon="CreditCard"
            maxlength="18" 
          />
          <div class="form-tip">用于跨医院就诊调阅时精准唯一定位健康档案，杜绝同名混淆</div>
        </el-form-item>

        <el-form-item label="11位手机号码（接收 Break-Glass 抢救知情推送）" required>
          <el-input 
            v-model="editForm.phone" 
            placeholder="请输入11位手机号" 
            :prefix-icon="Iphone"
            maxlength="11" 
          />
          <div class="form-tip">发生紧急破窗调阅时，系统网关将向该手机号发送安全预警</div>
        </el-form-item>

        <el-form-item v-if="auth.user?.role === 'doctor' || auth.user?.role === 'admin'" label="专业职称 / 资质级别">
          <el-input v-model="editForm.title" placeholder="如：主任医师、副主任医师、主治医师" />
        </el-form-item>
      </el-form>

      <template #footer>
        <el-button @click="editVisible = false">取 消</el-button>
        <el-button type="primary" :loading="savingProfile" @click="handleSaveProfile">
          保 存 并 同 步 生 效
        </el-button>
      </template>
    </el-dialog>

    <!-- 4. 修改登录密码弹窗 (Change Password Dialog) -->
    <el-dialog 
      title="修改登录密码" 
      v-model="passVisible" 
      width="460px"
      class="personal-pass-dialog"
      :close-on-click-modal="false"
    >
      <el-form :model="passForm" label-position="top">
        <el-form-item label="原登录密码" required>
          <el-input 
            v-model="passForm.old_password" 
            type="password" 
            placeholder="请输入当前正在使用的登录密码" 
            show-password 
            :prefix-icon="Lock"
          />
        </el-form-item>

        <el-form-item label="设置新密码 (≥6位)" required>
          <el-input 
            v-model="passForm.new_password" 
            type="password" 
            placeholder="请输入新的登录密码" 
            show-password 
            :prefix-icon="Lock"
          />
        </el-form-item>

        <el-form-item label="确认新密码" required>
          <el-input 
            v-model="passForm.confirm_password" 
            type="password" 
            placeholder="请再次输入新密码进行确认" 
            show-password 
            :prefix-icon="Lock"
          />
        </el-form-item>
      </el-form>

      <template #footer>
        <el-button @click="passVisible = false">取 消</el-button>
        <el-button type="primary" :loading="savingPass" @click="handleSavePassword">
          确 认 更 改 密 码
        </el-button>
      </template>
    </el-dialog>

    <!-- 5. 修改跨院病历调阅专属密钥弹窗 (Change Medical Key Dialog) -->
    <el-dialog 
      title="修改跨院病历调阅专属密钥" 
      v-model="medicalKeyVisible" 
      width="480px"
      class="personal-pass-dialog"
      :close-on-click-modal="false"
    >
      <el-alert
        title="关于跨院病历调阅专属密钥"
        type="success"
        show-icon
        :closable="false"
        class="mb-3"
        description="当您在外院就医时，主治医生发起跨院调阅时，您可直接向医生提供此专属密钥或在医生电脑现场输入，即可立即授权解锁您的健康病历，无需在线等待推送审批。"
      />

      <el-form :model="medicalKeyForm" label-position="top">
        <el-form-item label="原调阅密钥 (初始默认密码为 123456)">
          <el-input 
            v-model="medicalKeyForm.old_key" 
            type="password" 
            placeholder="若首次修改可留空或输入 123456" 
            show-password 
            :prefix-icon="Key"
          />
        </el-form-item>

        <el-form-item label="设置新专属调阅密钥 (建议 6 位数字或口令)" required>
          <el-input 
            v-model="medicalKeyForm.new_key" 
            type="password" 
            placeholder="请输入新的专属调阅密钥（≥4位）" 
            show-password 
            :prefix-icon="Key"
          />
        </el-form-item>

        <el-form-item label="确认新专属调阅密钥" required>
          <el-input 
            v-model="medicalKeyForm.confirm_key" 
            type="password" 
            placeholder="请再次输入新专属密钥进行确认" 
            show-password 
            :prefix-icon="Key"
          />
        </el-form-item>
      </el-form>

      <template #footer>
        <el-button @click="medicalKeyVisible = false">取 消</el-button>
        <el-button type="success" :loading="savingMedicalKey" @click="handleSaveMedicalKey">
          确 认 更 改 专 属 密 钥
        </el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'
import { 
  User, Edit, Lock, Camera, CopyDocument, CreditCard, Iphone, Key, View, Hide 
} from '@element-plus/icons-vue'
import { ElMessage } from 'element-plus'
import api from '../api/client'
import { useAuthStore } from '../stores/auth'

const auth = useAuthStore()
const loading = ref(false)

// 头像渐变色与首字母计算
const userInitials = computed(() => {
  const name = auth.user?.real_name || auth.user?.username || '用'
  return name.slice(0, 1).toUpperCase()
})

const avatarGradient = computed(() => {
  const role = auth.user?.role
  if (role === 'doctor') return 'linear-gradient(135deg, #8b5cf6 0%, #6366f1 100%)'
  if (role === 'patient') return 'linear-gradient(135deg, #10b981 0%, #059669 100%)'
  if (role === 'supervisor') return 'linear-gradient(135deg, #f59e0b 0%, #d97706 100%)'
  if (role === 'admin') return 'linear-gradient(135deg, #ef4444 0%, #dc2626 100%)'
  return 'linear-gradient(135deg, #3b82f6 0%, #1d4ed8 100%)'
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
  if (role === 'patient') return '居民患者'
  if (role === 'supervisor') return '卫健监管专员'
  if (role === 'admin') return '系统管理员'
  return '系统用户'
}

function handleAvatarClick() {
  ElMessage.info('系统当前已根据您的真实姓名生成了专属区块链数字身份印记头像')
}

function copyText(text?: string, label = '内容') {
  if (!text) return
  navigator.clipboard.writeText(text).then(() => {
    ElMessage.success(`已复制${label}到剪贴板: ${text}`)
  }).catch(() => {
    ElMessage.info(`内容为: ${text}`)
  })
}

// ── 编辑个人资料 ─────────────────────────────
const editVisible = ref(false)
const savingProfile = ref(false)
const editForm = ref({
  real_name: '',
  phone: '',
  id_card: '',
  title: '',
})

function openEditDialog() {
  if (!auth.user) return
  editForm.value = {
    real_name: auth.user.real_name || '',
    phone: auth.user.phone || '',
    id_card: (auth.user as any).id_card || '',
    title: auth.user.title || '',
  }
  editVisible.value = true
}

async function handleSaveProfile() {
  if (!editForm.value.real_name.trim()) {
    ElMessage.warning('真实姓名不能为空')
    return
  }
  if (!editForm.value.phone.trim()) {
    ElMessage.warning('手机号不能为空')
    return
  }

  savingProfile.value = true
  try {
    const res: any = await api.put('/auth/profile', editForm.value)
    if (res.code === 200) {
      ElMessage.success('个人档案信息更新成功！')
      // 同步更新 Pinia 状态及 LocalStorage
      auth.setUser(res.data)
      editVisible.value = false
    }
  } catch (err: any) {
    ElMessage.error(err?.message || '更新个人资料失败')
  } finally {
    savingProfile.value = false
  }
}

// ── 修改密码 ───────────────────────────────
const passVisible = ref(false)
const savingPass = ref(false)
const passForm = ref({
  old_password: '',
  new_password: '',
  confirm_password: '',
})

function openPassDialog() {
  passForm.value = {
    old_password: '',
    new_password: '',
    confirm_password: '',
  }
  passVisible.value = true
}

async function handleSavePassword() {
  if (!passForm.value.old_password) {
    ElMessage.warning('请输入当前原密码')
    return
  }
  if (!passForm.value.new_password || passForm.value.new_password.length < 6) {
    ElMessage.warning('新密码长度不能少于 6 位')
    return
  }
  if (passForm.value.new_password !== passForm.value.confirm_password) {
    ElMessage.warning('两次输入的新密码不一致，请核对')
    return
  }

  savingPass.value = true
  try {
    const res: any = await api.put('/auth/password', {
      old_password: passForm.value.old_password,
      new_password: passForm.value.new_password,
    })
    if (res.code === 200) {
      ElMessage.success('登录密码已成功更新，请牢记新密码！')
      passVisible.value = false
    }
  } catch (err: any) {
    ElMessage.error(err?.message || '修改密码失败')
  } finally {
    savingPass.value = false
  }
}

// ── 跨院病历调阅专属密钥 ───────────────────
const showKeyPlain = ref(false)
const medicalKeyVisible = ref(false)
const savingMedicalKey = ref(false)
const medicalKeyForm = ref({
  old_key: '',
  new_key: '',
  confirm_key: '',
})

function openMedicalKeyDialog() {
  medicalKeyForm.value = {
    old_key: '',
    new_key: '',
    confirm_key: '',
  }
  medicalKeyVisible.value = true
}

async function handleSaveMedicalKey() {
  if (!medicalKeyForm.value.new_key) {
    ElMessage.warning('请输入新的专属调阅密钥')
    return
  }
  if (medicalKeyForm.value.new_key.length < 4) {
    ElMessage.warning('专属调阅密钥长度不能少于 4 位')
    return
  }
  if (medicalKeyForm.value.new_key !== medicalKeyForm.value.confirm_key) {
    ElMessage.warning('两次输入的新专属密钥不一致，请核对')
    return
  }

  savingMedicalKey.value = true
  try {
    const res: any = await api.put('/auth/medical-key', {
      old_key: medicalKeyForm.value.old_key,
      new_key: medicalKeyForm.value.new_key,
    })
    if (res.code === 200) {
      ElMessage.success(res.message || '病历调阅专属密钥已成功更新！')
      if (auth.user) {
        auth.user.medical_key = medicalKeyForm.value.new_key
      }
      medicalKeyVisible.value = false
    }
  } catch (err: any) {
    ElMessage.error(err?.message || err?.response?.data?.message || '更新调阅密钥失败')
  } finally {
    savingMedicalKey.value = false
  }
}
</script>

<style scoped>
.personal-container {
  padding: 8px 4px;
}

/* ── Hero 渐变横幅 ────────────────────────────── */
.hero-header {
  background: linear-gradient(135deg, #6366f1 0%, #8b5cf6 50%, #a855f7 100%);
  border-radius: 16px;
  padding: 24px 30px;
  margin-bottom: 24px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  box-shadow: 0 8px 24px rgba(99, 102, 241, 0.18);
}

.header-left {
  display: flex;
  align-items: center;
  gap: 16px;
}

.header-icon-ring {
  width: 48px;
  height: 48px;
  border-radius: 50%;
  background: rgba(255, 255, 255, 0.22);
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 24px;
  color: #fff;
  flex-shrink: 0;
  backdrop-filter: blur(4px);
}

.page-title {
  margin: 0 0 4px;
  font-size: 1.45rem;
  font-weight: 800;
  color: #fff;
}

.page-sub {
  margin: 0;
  font-size: 0.88rem;
  color: rgba(255, 255, 255, 0.9);
}

.header-right-chip {
  display: flex;
  align-items: center;
  gap: 8px;
  background: rgba(255, 255, 255, 0.18);
  backdrop-filter: blur(8px);
  padding: 6px 14px;
  border-radius: 20px;
  color: #fff;
  font-size: 12.5px;
  font-weight: 600;
  border: 1px solid rgba(255, 255, 255, 0.25);
}

.chain-status-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background: #4ade80;
  box-shadow: 0 0 8px #4ade80;
}

/* ── 个人主卡片 ──────────────────────────────── */
.profile-card {
  margin-bottom: 24px;
  padding: 24px 28px;
  background: #ffffff;
  border-radius: 14px;
  border: 1px solid rgba(226, 232, 240, 0.9);
  box-shadow: 0 4px 18px rgba(0, 0, 0, 0.03);
}

.profile-hero-section {
  display: flex;
  align-items: center;
  gap: 24px;
  flex-wrap: wrap;
}

.avatar-col {
  flex-shrink: 0;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 6px;
  cursor: pointer;
}

.user-avatar-circle {
  width: 84px;
  height: 84px;
  border-radius: 50%;
  position: relative;
  display: flex;
  align-items: center;
  justify-content: center;
  box-shadow: 0 4px 16px rgba(139, 92, 246, 0.28);
  border: 3px solid #ffffff;
  transition: transform 0.25s ease, box-shadow 0.25s ease;
}

.avatar-col:hover .user-avatar-circle {
  transform: scale(1.05);
  box-shadow: 0 6px 20px rgba(139, 92, 246, 0.4);
}

.user-avatar-text {
  font-size: 30px;
  font-weight: 800;
  color: #ffffff;
}

.avatar-hover-overlay {
  position: absolute;
  inset: 0;
  background: rgba(0, 0, 0, 0.52);
  backdrop-filter: blur(2px);
  color: #ffffff;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 2px;
  opacity: 0;
  transition: opacity 0.2s ease;
  border-radius: 50%;
}

.avatar-col:hover .avatar-hover-overlay {
  opacity: 1;
}

.camera-icon {
  font-size: 20px;
}

.change-hint-text {
  font-size: 11px;
  font-weight: 600;
}

.change-avatar-btn {
  font-size: 12px;
  font-weight: 600;
  padding: 0;
}

/* ── 身份文字信息区 ────────────────────────────── */
.identity-col {
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: 10px;
  min-width: 280px;
}

.identity-name-row {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 10px;
}

.user-title-name {
  margin: 0;
  font-size: 22px;
  font-weight: 700;
  color: #0f172a;
}

.role-badge {
  font-weight: 600;
  border-radius: 6px;
  font-size: 12px;
  padding: 0 8px;
  height: 24px;
  line-height: 24px;
}

.status-tag {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  font-size: 12px;
  font-weight: 600;
  border-radius: 12px;
  padding: 0 10px;
  height: 24px;
}

.status-dot {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: #10b981;
  display: inline-block;
}

.dot-danger {
  background: #ef4444;
}

.identity-sub-row {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 16px;
  font-size: 13px;
}

.identity-chip {
  display: inline-flex;
  align-items: center;
  gap: 6px;
}

.chip-label {
  color: #64748b;
  font-weight: 500;
}

.chip-val {
  color: #1e293b;
  font-weight: 600;
}

.font-mono {
  font-family: 'JetBrains Mono', Consolas, Monaco, monospace;
}

.emp-code {
  background: #f1f5f9;
  padding: 2px 8px;
  border-radius: 6px;
  font-family: 'JetBrains Mono', Consolas, Monaco, monospace;
  font-weight: 600;
  color: #6366f1;
  border: 1px solid #e2e8f0;
}

.dept-bar {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 8px;
  font-size: 13px;
}

.dept-label {
  color: #64748b;
  font-weight: 500;
}

.dept-pill {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  background: linear-gradient(135deg, rgba(99, 102, 241, 0.07), rgba(16, 185, 129, 0.07));
  border: 1px solid rgba(99, 102, 241, 0.2);
  padding: 3px 12px;
  border-radius: 16px;
}

.dept-avatar {
  font-size: 13px;
}

.dept-name {
  font-weight: 600;
  color: #1e293b;
}

.title-tag {
  height: 20px;
  line-height: 18px;
  padding: 0 6px;
  font-size: 11px;
}

/* ── 右侧操作区 ──────────────────────────────── */
.actions-col {
  display: flex;
  align-items: center;
  gap: 10px;
  align-self: flex-start;
  padding-top: 4px;
}

.btn-action-primary,
.btn-action-secondary {
  border-radius: 8px;
  font-weight: 600;
  padding: 8px 16px;
}

.profile-divider {
  margin: 20px 0;
  border-color: #f1f5f9;
}

/* ── 结构化双栏资料卡片 ────────────────────────── */
.info-blocks-grid {
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  gap: 20px;
}

@media (max-width: 880px) {
  .info-blocks-grid {
    grid-template-columns: 1fr;
    gap: 16px;
  }
}

.info-block-card {
  background: #fafafa;
  border: 1px solid #e2e8f0;
  border-radius: 12px;
  padding: 18px 22px;
  display: flex;
  flex-direction: column;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.02);
  transition: border-color 0.2s, box-shadow 0.2s;
}

.info-block-card:hover {
  border-color: #c7d2fe;
  box-shadow: 0 4px 14px rgba(99, 102, 241, 0.08);
}

.block-card-header {
  display: flex;
  align-items: center;
  gap: 8px;
  padding-bottom: 12px;
  border-bottom: 1px solid #e2e8f0;
  margin-bottom: 4px;
}

.block-icon {
  font-size: 17px;
}

.block-title {
  font-size: 14.5px;
  font-weight: 700;
  color: #1e293b;
}

.block-card-body {
  display: flex;
  flex-direction: column;
}

.field-item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 14px;
  padding: 10px 0;
  border-bottom: 1px dashed #e2e8f0;
  font-size: 13.5px;
  min-height: 42px;
}

.field-item:last-child {
  border-bottom: none;
  padding-bottom: 2px;
}

.field-label {
  color: #64748b;
  font-weight: 500;
  flex-shrink: 0;
  min-width: 110px;
}

.field-val {
  color: #1e293b;
}

.contact-val {
  display: inline-flex;
  align-items: center;
  gap: 8px;
}

.font-bold {
  font-weight: 600;
}

.text-primary {
  color: #4f46e5;
}

.text-muted {
  color: #94a3b8;
}

.text-xs {
  font-size: 12px;
}

/* ── 统计面板 ────────────────────────────────── */
.stats-section {
  margin-top: 24px;
}

.stats-header-bar {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 14px;
}

.stats-bar-icon {
  font-size: 18px;
}

.stats-title {
  font-size: 16px;
  font-weight: 700;
  color: #1e293b;
  margin: 0;
}

.stats-grid {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 18px;
}

@media (max-width: 768px) {
  .stats-grid {
    grid-template-columns: 1fr;
  }
}

.stat-card {
  border-radius: 12px;
  border: 1px solid #e2e8f0;
}

.stat-card-header {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 13.5px;
  font-weight: 600;
  color: #475569;
}

.stat-value {
  font-size: 32px;
  font-weight: 800;
  font-family: 'JetBrains Mono', Consolas, sans-serif;
  margin-top: 4px;
}

.stat-unit {
  font-size: 14px;
  font-weight: normal;
  color: #94a3b8;
}

.text-purple {
  color: #8b5cf6;
}

.text-green {
  color: #10b981;
}

.text-amber {
  color: #f59e0b;
}

.form-tip {
  font-size: 11px;
  color: #6366f1;
  margin-top: 4px;
  line-height: 1.3;
}
</style>
