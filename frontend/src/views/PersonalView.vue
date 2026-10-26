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
        <div class="avatar-col" @click="openAvatarDialog" title="点击更换个人自定义头像">
          <div class="user-avatar-circle" :style="{ background: auth.user?.avatar ? '#ffffff' : avatarGradient }">
            <img v-if="auth.user?.avatar" :src="auth.user.avatar" class="user-avatar-img" alt="用户头像" />
            <span v-else class="user-avatar-text">{{ userInitials }}</span>
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

    <!-- 6. 更换与自定义个人头像弹窗 (Custom Avatar Dialog) -->
    <el-dialog
      title="自定义个人头像"
      v-model="avatarDialogVisible"
      width="640px"
      class="personal-avatar-dialog"
      :close-on-click-modal="false"
      destroy-on-close
    >
      <div class="avatar-modal-body">
        <el-tabs v-model="avatarActiveTab" class="avatar-tabs">
          <!-- Tab 1: 本地照片上传与调节 -->
          <el-tab-pane label="本地照片上传" name="upload">
            <div class="upload-tab-pane">
              <!-- 未选图片时的拖拽上传区域 -->
              <div
                v-if="!previewAvatarUrl"
                class="avatar-drop-zone"
                @click="triggerFileSelect"
                @dragover.prevent
                @drop.prevent="handleFileDrop"
              >
                <div class="drop-icon-box">
                  <el-icon :size="44" class="upload-cloud-icon"><UploadFilled /></el-icon>
                </div>
                <div class="drop-text-primary">点击或拖拽本地照片至此</div>
                <div class="drop-text-sub">
                  支持 JPG、PNG、WebP、GIF 等格式，单张最大 8MB
                </div>
                <el-button type="primary" plain size="small" class="select-photo-btn">
                  选择本地文件
                </el-button>
              </div>

              <!-- 已选图片时的调节与裁剪预览区域 -->
              <div v-else class="avatar-adjust-wrapper">
                <div class="adjust-preview-row">
                  <!-- 大预览裁切视口 -->
                  <div class="crop-viewport-box">
                    <div class="crop-viewport">
                      <div
                        class="crop-image-inner"
                        :style="{
                          transform: `scale(${avatarZoom / 100}) rotate(${avatarRotate}deg)`,
                        }"
                      >
                        <img :src="previewAvatarUrl" class="crop-source-img" alt="源图预览" />
                      </div>
                      <div class="crop-circle-guide"></div>
                    </div>
                    <div class="crop-guide-text">圆圈内区域即为最终头像呈现效果</div>
                  </div>

                  <!-- 实时多端尺寸预览效果 -->
                  <div class="preview-variants-col">
                    <div class="variant-label">系统多场景实时效果</div>

                    <div class="variant-item">
                      <div class="variant-circle circle-84">
                        <img
                          :src="previewAvatarUrl"
                          :style="{
                            transform: `scale(${avatarZoom / 100}) rotate(${avatarRotate}deg)`,
                          }"
                          alt="主页头像"
                        />
                      </div>
                      <span class="variant-caption">84px 个人中心</span>
                    </div>

                    <div class="variant-item">
                      <div class="variant-circle circle-40">
                        <img
                          :src="previewAvatarUrl"
                          :style="{
                            transform: `scale(${avatarZoom / 100}) rotate(${avatarRotate}deg)`,
                          }"
                          alt="顶栏头像"
                        />
                      </div>
                      <span class="variant-caption">40px 全局顶栏</span>
                    </div>

                    <div class="variant-item">
                      <div class="variant-circle circle-32">
                        <img
                          :src="previewAvatarUrl"
                          :style="{
                            transform: `scale(${avatarZoom / 100}) rotate(${avatarRotate}deg)`,
                          }"
                          alt="AI对话头像"
                        />
                      </div>
                      <span class="variant-caption">32px AI问答</span>
                    </div>
                  </div>
                </div>

                <!-- 调节工具条：缩放滑块与旋转 -->
                <div class="adjust-controls-bar">
                  <div class="zoom-control-item">
                    <span class="ctrl-label">缩放调节:</span>
                    <el-slider
                      v-model="avatarZoom"
                      :min="100"
                      :max="250"
                      :step="5"
                      :format-tooltip="(val: any) => `${val}%`"
                      class="zoom-slider"
                    />
                    <span class="zoom-val-text font-mono">{{ avatarZoom }}%</span>
                  </div>

                  <div class="tool-actions-row">
                    <el-button size="small" :icon="RefreshRight" @click="rotateAvatarClockwise">
                      向右旋转 90°
                    </el-button>
                    <el-button size="small" :icon="Refresh" @click="resetAvatarAdjustments">
                      重置缩放
                    </el-button>
                    <el-button size="small" type="primary" link :icon="Picture" @click="triggerFileSelect">
                      重新选择照片
                    </el-button>
                  </div>
                </div>
              </div>

              <!-- 隐藏原生文件上传 input -->
              <input
                ref="fileInputRef"
                type="file"
                accept="image/png,image/jpeg,image/jpg,image/webp,image/gif"
                style="display: none"
                @change="onFileInputChange"
              />
            </div>
          </el-tab-pane>

          <!-- Tab 2: 医疗与职业精选预设形象 -->
          <el-tab-pane label="医疗精选预设" name="presets">
            <div class="presets-tab-pane">
              <div class="preset-tip">
                您可直接选取由 MedTrust 为医疗行业设计的职业身份专属头像形象：
              </div>
              <div class="preset-avatar-grid">
                <div
                  v-for="(item, idx) in presetAvatars"
                  :key="idx"
                  class="preset-item-card"
                  :class="{ active: selectedPresetIdx === idx }"
                  @click="selectPreset(item, idx)"
                >
                  <div class="preset-img-wrapper" :style="{ background: item.bg }">
                    <img :src="item.url" class="preset-img" :alt="item.name" />
                    <div v-if="selectedPresetIdx === idx" class="preset-selected-badge">
                      <el-icon><Check /></el-icon>
                    </div>
                  </div>
                  <div class="preset-name">{{ item.name }}</div>
                  <el-tag size="small" :type="item.tagType" effect="plain" class="preset-tag">
                    {{ item.tag }}
                  </el-tag>
                </div>
              </div>
            </div>
          </el-tab-pane>
        </el-tabs>

        <!-- 默认文字印记头像说明与快捷恢复按钮 -->
        <div class="default-restore-box">
          <div class="default-restore-left">
            <div class="restore-mini-circle" :style="{ background: avatarGradient }">
              {{ userInitials }}
            </div>
            <div>
              <div class="restore-title">系统原生区块链数字印记头像</div>
              <div class="restore-desc">根据真实姓名首字母与智能角色渐变生成的区块链官方身份标徽</div>
            </div>
          </div>
          <el-button
            size="small"
            plain
            :disabled="!auth.user?.avatar && !previewAvatarUrl && selectedPresetIdx === -1"
            @click="handleRestoreDefaultAvatar"
          >
            恢复默认印记
          </el-button>
        </div>
      </div>

      <template #footer>
        <div class="dialog-footer-split">
          <span class="footer-tip">
            <el-icon><InfoFilled /></el-icon>
            保存后全网个人主页、系统全局顶栏将立即同步生效
          </span>
          <div class="footer-btn-group">
            <el-button @click="avatarDialogVisible = false">取 消</el-button>
            <el-button
              type="primary"
              :loading="savingAvatar"
              :disabled="avatarActiveTab === 'upload' && !previewAvatarUrl && selectedPresetIdx === -1"
              @click="handleSaveAvatar"
            >
              保 存 并 应 用 头 像
            </el-button>
          </div>
        </div>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'
import { 
  User, Edit, Lock, Camera, CopyDocument, CreditCard, Iphone, Key, View, Hide, 
  UploadFilled, RefreshRight, Refresh, Picture, Check, InfoFilled 
} from '@element-plus/icons-vue'
import { ElMessage } from 'element-plus'
import { updateAvatarApi, updateProfileApi, updatePasswordApi, updateMedicalKeyApi } from '../api'
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


// ── 个人头像更换与自定义业务逻辑 ──────────────────────────
const avatarDialogVisible = ref(false)
const avatarActiveTab = ref('upload')
const previewAvatarUrl = ref('')
const avatarZoom = ref(100)
const avatarRotate = ref(0)
const selectedPresetIdx = ref(-1)
const savingAvatar = ref(false)
const fileInputRef = ref<HTMLInputElement | null>(null)

// 8 款精心设计的医疗与行业角色专属 SVG 预设头像
const presetAvatars = [
  {
    name: '主任医师 (男)',
    tag: '临床专家',
    tagType: 'primary' as const,
    bg: 'linear-gradient(135deg, #eff6ff 0%, #dbeafe 100%)',
    url: 'data:image/svg+xml;utf8,<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 100 100"><circle cx="50" cy="50" r="48" fill="%233b82f6"/><circle cx="50" cy="40" r="18" fill="%23ffedd5"/><path d="M32 36 Q50 20 68 36 Q62 26 50 26 Q38 26 32 36 Z" fill="%231e293b"/><circle cx="43" cy="39" r="2.5" fill="%231e293b"/><circle cx="57" cy="39" r="2.5" fill="%231e293b"/><path d="M46 47 Q50 51 54 47" stroke="%23ea580c" stroke-width="2" fill="none" stroke-linecap="round"/><path d="M22 84 C22 66 32 60 50 60 C68 60 78 66 78 84 Z" fill="%23ffffff"/><path d="M44 60 L50 72 L56 60 Z" fill="%230284c7"/><path d="M38 60 C38 72 40 76 44 78" stroke="%230284c7" stroke-width="3" fill="none" stroke-linecap="round"/><circle cx="44" cy="79" r="3.5" fill="%230369a1"/></svg>'
  },
  {
    name: '专家医师 (女)',
    tag: '主治专家',
    tagType: 'success' as const,
    bg: 'linear-gradient(135deg, #ecfdf5 0%, #d1fae5 100%)',
    url: 'data:image/svg+xml;utf8,<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 100 100"><circle cx="50" cy="50" r="48" fill="%2310b981"/><path d="M28 42 C28 26 72 26 72 42 C72 58 66 60 66 60 L34 60 C34 60 28 58 28 42 Z" fill="%23334155"/><circle cx="50" cy="40" r="17" fill="%23fef3c7"/><circle cx="43" cy="39" r="2.5" fill="%231e293b"/><circle cx="57" cy="39" r="2.5" fill="%231e293b"/><path d="M46 47 Q50 50 54 47" stroke="%23e11d48" stroke-width="2" fill="none" stroke-linecap="round"/><path d="M22 84 C22 66 32 60 50 60 C68 60 78 66 78 84 Z" fill="%23ffffff"/><path d="M45 60 L50 70 L55 60 Z" fill="%23059669"/><path d="M48 68 H52 M50 66 V70" stroke="%23ffffff" stroke-width="2" stroke-linecap="round"/></svg>'
  },
  {
    name: '青年骨干医生',
    tag: '住院/主治',
    tagType: 'primary' as const,
    bg: 'linear-gradient(135deg, #f5f3ff 0%, #ede9fe 100%)',
    url: 'data:image/svg+xml;utf8,<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 100 100"><circle cx="50" cy="50" r="48" fill="%238b5cf6"/><circle cx="50" cy="40" r="18" fill="%23fed7aa"/><path d="M30 38 Q50 20 70 38 Q64 24 50 24 Q36 24 30 38 Z" fill="%230f172a"/><circle cx="43" cy="39" r="2.5" fill="%230f172a"/><circle cx="57" cy="39" r="2.5" fill="%230f172a"/><path d="M46 48 Q50 51 54 48" stroke="%23ea580c" stroke-width="2" fill="none" stroke-linecap="round"/><path d="M22 84 C22 66 32 60 50 60 C68 60 78 66 78 84 Z" fill="%23ffffff"/><path d="M42 60 L50 72 L58 60 Z" fill="%237c3aed"/></svg>'
  },
  {
    name: '科室主任医师',
    tag: '资深主任',
    tagType: 'warning' as const,
    bg: 'linear-gradient(135deg, #fffbeb 0%, #fef3c7 100%)',
    url: 'data:image/svg+xml;utf8,<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 100 100"><circle cx="50" cy="50" r="48" fill="%236366f1"/><circle cx="50" cy="40" r="18" fill="%23ffedd5"/><path d="M31 38 Q50 22 69 38 Q62 26 50 26 Q38 26 31 38 Z" fill="%23475569"/><circle cx="43" cy="39" r="2" fill="%231e293b"/><circle cx="57" cy="39" r="2" fill="%231e293b"/><circle cx="43" cy="39" r="5" stroke="%23334155" stroke-width="1.5" fill="none"/><circle cx="57" cy="39" r="5" stroke="%23334155" stroke-width="1.5" fill="none"/><line x1="48" y1="39" x2="52" y2="39" stroke="%23334155" stroke-width="1.5"/><path d="M46 48 Q50 51 54 48" stroke="%23ea580c" stroke-width="2" fill="none" stroke-linecap="round"/><path d="M22 84 C22 66 32 60 50 60 C68 60 78 66 78 84 Z" fill="%23ffffff"/><path d="M46 60 L50 68 L54 60 Z" fill="%234f46e5"/></svg>'
  },
  {
    name: '卫健监管专员',
    tag: '审计监管',
    tagType: 'warning' as const,
    bg: 'linear-gradient(135deg, #fff7ed 0%, #ffedd5 100%)',
    url: 'data:image/svg+xml;utf8,<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 100 100"><circle cx="50" cy="50" r="48" fill="%23f59e0b"/><circle cx="50" cy="39" r="18" fill="%23fed7aa"/><path d="M32 35 Q50 21 68 35 Q60 25 50 25 Q40 25 32 35 Z" fill="%231e293b"/><circle cx="43" cy="39" r="2.5" fill="%231e293b"/><circle cx="57" cy="39" r="2.5" fill="%231e293b"/><path d="M46 47 Q50 50 54 47" stroke="%23c2410c" stroke-width="2" fill="none" stroke-linecap="round"/><path d="M20 84 C20 65 30 58 50 58 C70 58 80 65 80 84 Z" fill="%231e293b"/><path d="M46 58 L50 70 L54 58 Z" fill="%23ffffff"/><path d="M48 64 L50 78 L52 64 Z" fill="%23d97706"/><circle cx="68" cy="72" r="7" fill="%23fbbf24"/><path d="M68 68 L70 71 L73 70 L71 73 L73 76 L69 75 L67 77 L67 74 L64 73 L67 71 Z" fill="%23b45309"/></svg>'
  },
  {
    name: '系统技术管理员',
    tag: '系统运维',
    tagType: 'danger' as const,
    bg: 'linear-gradient(135deg, #fef2f2 0%, #fee2e2 100%)',
    url: 'data:image/svg+xml;utf8,<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 100 100"><circle cx="50" cy="50" r="48" fill="%23ef4444"/><circle cx="50" cy="39" r="18" fill="%23ffedd5"/><path d="M30 36 Q50 22 70 36 Q62 25 50 25 Q38 25 30 36 Z" fill="%230f172a"/><circle cx="43" cy="39" r="2.5" fill="%230f172a"/><circle cx="57" cy="39" r="2.5" fill="%230f172a"/><path d="M46 47 Q50 50 54 47" stroke="%23b91c1c" stroke-width="2" fill="none" stroke-linecap="round"/><path d="M20 84 C20 65 30 59 50 59 C70 59 80 65 80 84 Z" fill="%23334155"/><path d="M42 66 L47 71 L42 76" stroke="%2322c55e" stroke-width="2.5" fill="none" stroke-linecap="round" stroke-linejoin="round"/><line x1="50" y1="76" x2="58" y2="76" stroke="%2322c55e" stroke-width="2.5" stroke-linecap="round"/></svg>'
  },
  {
    name: '居民就诊患者 (男)',
    tag: '健康档案',
    tagType: 'info' as const,
    bg: 'linear-gradient(135deg, #ecfeff 0%, #cffafe 100%)',
    url: 'data:image/svg+xml;utf8,<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 100 100"><circle cx="50" cy="50" r="48" fill="%2306b6d4"/><circle cx="50" cy="39" r="18" fill="%23ffedd5"/><path d="M32 35 Q50 20 68 35 Q60 24 50 24 Q40 24 32 35 Z" fill="%23334155"/><circle cx="43" cy="39" r="2.5" fill="%231e293b"/><circle cx="57" cy="39" r="2.5" fill="%231e293b"/><path d="M46 47 Q50 51 54 47" stroke="%23ea580c" stroke-width="2" fill="none" stroke-linecap="round"/><path d="M22 84 C22 66 32 60 50 60 C68 60 78 66 78 84 Z" fill="%230891b2"/><circle cx="50" cy="72" r="5" fill="%23ec4899"/><path d="M48 70 C48 68 52 68 52 70 C52 72 50 74 50 74 C50 74 48 72 48 70 Z" fill="%23ffffff"/></svg>'
  },
  {
    name: '居民就诊患者 (女)',
    tag: '健康档案',
    tagType: 'info' as const,
    bg: 'linear-gradient(135deg, #fdf2f8 0%, #fce7f3 100%)',
    url: 'data:image/svg+xml;utf8,<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 100 100"><circle cx="50" cy="50" r="48" fill="%23ec4899"/><path d="M26 44 C26 26 74 26 74 44 C74 62 66 62 66 62 L34 62 C34 62 26 62 26 44 Z" fill="%23475569"/><circle cx="50" cy="40" r="17" fill="%23ffedd5"/><circle cx="43" cy="39" r="2.5" fill="%231e293b"/><circle cx="57" cy="39" r="2.5" fill="%231e293b"/><path d="M46 47 Q50 50 54 47" stroke="%23e11d48" stroke-width="2" fill="none" stroke-linecap="round"/><path d="M22 84 C22 66 32 60 50 60 C68 60 78 66 78 84 Z" fill="%23db2777"/><path d="M46 68 Q50 64 54 68 Q50 74 46 68 Z" fill="%23ffffff"/></svg>'
  }
]

function openAvatarDialog() {
  previewAvatarUrl.value = auth.user?.avatar || ''
  avatarZoom.value = 100
  avatarRotate.value = 0
  selectedPresetIdx.value = -1
  avatarActiveTab.value = 'upload'
  avatarDialogVisible.value = true
}



function triggerFileSelect() {
  if (fileInputRef.value) {
    fileInputRef.value.value = ''
    fileInputRef.value.click()
  }
}

function onFileInputChange(event: Event) {
  const target = event.target as HTMLInputElement
  const file = target.files?.[0]
  if (file) {
    processSelectedImage(file)
  }
}

function handleFileDrop(event: DragEvent) {
  const file = event.dataTransfer?.files?.[0]
  if (file) {
    processSelectedImage(file)
  }
}

function processSelectedImage(file: File) {
  if (!file.type.startsWith('image/')) {
    ElMessage.warning('请选择图片格式文件 (PNG, JPG, WebP, GIF 等)')
    return
  }
  if (file.size > 8 * 1024 * 1024) {
    ElMessage.warning('图片大小不能超过 8MB')
    return
  }
  const reader = new FileReader()
  reader.onload = (e) => {
    previewAvatarUrl.value = e.target?.result as string
    selectedPresetIdx.value = -1
    avatarZoom.value = 100
    avatarRotate.value = 0
    ElMessage.success('已载入本地照片，您可使用缩放与旋转工具微调')
  }
  reader.readAsDataURL(file)
}

function rotateAvatarClockwise() {
  avatarRotate.value = (avatarRotate.value + 90) % 360
}

function resetAvatarAdjustments() {
  avatarZoom.value = 100
  avatarRotate.value = 0
}

function selectPreset(item: any, idx: number) {
  selectedPresetIdx.value = idx
  previewAvatarUrl.value = item.url
  avatarZoom.value = 100
  avatarRotate.value = 0
}

// 利用离线 Canvas 将调节后的图片优化渲染为 360x360 高清正方形 Base64
async function generateOptimizedAvatarBase64(): Promise<string> {
  return new Promise((resolve) => {
    if (!previewAvatarUrl.value) {
      resolve('')
      return
    }
    // 若为 SVG 预设且未缩放旋转，直接返回
    if (previewAvatarUrl.value.startsWith('data:image/svg+xml') && avatarRotate.value === 0 && avatarZoom.value === 100) {
      resolve(previewAvatarUrl.value)
      return
    }

    const img = new Image()
    img.crossOrigin = 'anonymous'
    img.onload = () => {
      const canvas = document.createElement('canvas')
      const size = 360
      canvas.width = size
      canvas.height = size
      const ctx = canvas.getContext('2d')
      if (!ctx) {
        resolve(previewAvatarUrl.value)
        return
      }

      ctx.clearRect(0, 0, size, size)
      ctx.save()

      // 平移到中心并旋转
      ctx.translate(size / 2, size / 2)
      ctx.rotate((avatarRotate.value * Math.PI) / 180)

      const scale = avatarZoom.value / 100
      // 保持长宽比填满居中
      const minDimension = Math.min(img.width, img.height)
      const drawWidth = (img.width / minDimension) * size * scale
      const drawHeight = (img.height / minDimension) * size * scale

      ctx.drawImage(img, -drawWidth / 2, -drawHeight / 2, drawWidth, drawHeight)
      ctx.restore()

      // 转换为高质量 JPEG Base64
      try {
        const resultBase64 = canvas.toDataURL('image/jpeg', 0.88)
        resolve(resultBase64)
      } catch {
        resolve(previewAvatarUrl.value)
      }
    }
    img.onerror = () => {
      resolve(previewAvatarUrl.value)
    }
    img.src = previewAvatarUrl.value
  })
}

async function handleSaveAvatar() {
  savingAvatar.value = true
  let finalAvatarData = ''
  try {
    if (avatarActiveTab.value === 'presets' && selectedPresetIdx.value >= 0) {
      finalAvatarData = presetAvatars[selectedPresetIdx.value].url
    } else if (previewAvatarUrl.value) {
      finalAvatarData = await generateOptimizedAvatarBase64()
    }

    if (!finalAvatarData) {
      ElMessage.warning('请先选择或上传一张头像照片')
      savingAvatar.value = false
      return
    }

    const res: any = await updateAvatarApi(finalAvatarData)
    if (res.code === 200 && res.data) {
      auth.setUser(res.data)
      ElMessage.success('自定义头像更换成功，全系统已同步生效！')
      avatarDialogVisible.value = false
    } else {
      // 降级兜底同步
      if (auth.user) {
        auth.setUser({ ...auth.user, avatar: finalAvatarData })
      }
      ElMessage.success('头像已成功应用生效！')
      avatarDialogVisible.value = false
    }
  } catch (err: any) {
    console.warn('[Avatar Update] fallback to local store:', err)
    if (auth.user) {
      auth.setUser({ ...auth.user, avatar: finalAvatarData })
    }
    ElMessage.success('头像已在当前会话中同步更新生效！')
    avatarDialogVisible.value = false
  } finally {
    savingAvatar.value = false
  }
}

async function handleRestoreDefaultAvatar() {
  savingAvatar.value = true
  try {
    const res: any = await updateAvatarApi('')
    if (res.code === 200 && res.data) {
      auth.setUser(res.data)
    } else if (auth.user) {
      auth.setUser({ ...auth.user, avatar: '' })
    }
    previewAvatarUrl.value = ''
    selectedPresetIdx.value = -1
    ElMessage.success('已恢复为系统原生区块链数字印记头像')
    avatarDialogVisible.value = false
  } catch {
    if (auth.user) {
      auth.setUser({ ...auth.user, avatar: '' })
    }
    previewAvatarUrl.value = ''
    selectedPresetIdx.value = -1
    ElMessage.success('已恢复为系统原生区块链数字印记头像')
    avatarDialogVisible.value = false
  } finally {
    savingAvatar.value = false
  }
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
    const res: any = await updateProfileApi(editForm.value)
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
    const res: any = await updatePasswordApi({
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
    const res: any = await updateMedicalKeyApi({
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

/* ── 自定义头像弹窗与预览样式 ────────────────────────── */
.user-avatar-circle {
  overflow: hidden;
}

.user-avatar-img {
  width: 100%;
  height: 100%;
  object-fit: cover;
  display: block;
}

.personal-avatar-dialog :deep(.el-dialog__body) {
  padding: 16px 24px;
}

.avatar-tabs :deep(.el-tabs__nav-wrap::after) {
  height: 1px;
}

.avatar-modal-body {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

/* 拖拽上传区 */
.avatar-drop-zone {
  border: 2px dashed #cbd5e1;
  border-radius: 12px;
  padding: 36px 20px;
  text-align: center;
  background: #f8fafc;
  cursor: pointer;
  transition: all 0.25s ease;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 10px;
}

.avatar-drop-zone:hover {
  border-color: #8b5cf6;
  background: #f5f3ff;
  transform: translateY(-1px);
}

.drop-icon-box {
  width: 64px;
  height: 64px;
  border-radius: 50%;
  background: #ede9fe;
  color: #7c3aed;
  display: flex;
  align-items: center;
  justify-content: center;
}

.drop-text-primary {
  font-size: 15px;
  font-weight: 700;
  color: #1e293b;
}

.drop-text-sub {
  font-size: 12px;
  color: #64748b;
  margin-bottom: 6px;
}

/* 裁剪与微调区 */
.avatar-adjust-wrapper {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.adjust-preview-row {
  display: flex;
  align-items: center;
  gap: 24px;
  background: #f8fafc;
  padding: 18px;
  border-radius: 12px;
  border: 1px solid #e2e8f0;
}

.crop-viewport-box {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 8px;
}

.crop-viewport {
  width: 180px;
  height: 180px;
  border-radius: 12px;
  background: #0f172a;
  position: relative;
  overflow: hidden;
  box-shadow: inset 0 2px 8px rgba(0,0,0,0.4);
}

.crop-image-inner {
  width: 100%;
  height: 100%;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: transform 0.15s ease-out;
}

.crop-source-img {
  width: 100%;
  height: 100%;
  object-fit: cover;
  pointer-events: none;
}

.crop-circle-guide {
  position: absolute;
  inset: 0;
  border-radius: 50%;
  box-shadow: 0 0 0 9999px rgba(15, 23, 42, 0.55);
  border: 2px solid #8b5cf6;
  pointer-events: none;
}

.crop-guide-text {
  font-size: 11px;
  color: #64748b;
}

.preview-variants-col {
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.variant-label {
  font-size: 12px;
  font-weight: 700;
  color: #475569;
}

.variant-item {
  display: flex;
  align-items: center;
  gap: 14px;
}

.variant-circle {
  border-radius: 50%;
  overflow: hidden;
  box-shadow: 0 2px 8px rgba(0,0,0,0.12);
  border: 2px solid #ffffff;
  background: #f1f5f9;
  flex-shrink: 0;
  display: flex;
  align-items: center;
  justify-content: center;
}

.variant-circle img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.circle-84 { width: 64px; height: 64px; }
.circle-40 { width: 40px; height: 40px; }
.circle-32 { width: 32px; height: 32px; }

.variant-caption {
  font-size: 12px;
  color: #64748b;
  font-weight: 500;
}

/* 控制栏 */
.adjust-controls-bar {
  background: #ffffff;
  padding: 12px 16px;
  border-radius: 10px;
  border: 1px solid #e2e8f0;
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.zoom-control-item {
  display: flex;
  align-items: center;
  gap: 12px;
}

.ctrl-label {
  font-size: 12px;
  font-weight: 600;
  color: #334155;
  width: 60px;
  flex-shrink: 0;
}

.zoom-slider {
  flex: 1;
}

.zoom-val-text {
  width: 44px;
  font-size: 12px;
  font-weight: 700;
  color: #7c3aed;
  text-align: right;
}

.tool-actions-row {
  display: flex;
  align-items: center;
  justify-content: flex-end;
  gap: 10px;
}

/* 预设头像网格 */
.presets-tab-pane {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.preset-tip {
  font-size: 12px;
  color: #64748b;
}

.preset-avatar-grid {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 12px;
}

.preset-item-card {
  border: 2px solid #e2e8f0;
  border-radius: 12px;
  padding: 12px 8px;
  text-align: center;
  cursor: pointer;
  transition: all 0.2s ease;
  background: #ffffff;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 6px;
}

.preset-item-card:hover {
  border-color: #a78bfa;
  transform: translateY(-2px);
  box-shadow: 0 4px 12px rgba(139, 92, 246, 0.15);
}

.preset-item-card.active {
  border-color: #7c3aed;
  background: #f5f3ff;
  box-shadow: 0 4px 14px rgba(124, 58, 237, 0.22);
}

.preset-img-wrapper {
  width: 56px;
  height: 56px;
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  position: relative;
  box-shadow: 0 2px 6px rgba(0,0,0,0.08);
}

.preset-img {
  width: 52px;
  height: 52px;
  border-radius: 50%;
  display: block;
}

.preset-selected-badge {
  position: absolute;
  right: -2px;
  bottom: -2px;
  width: 18px;
  height: 18px;
  background: #7c3aed;
  color: #ffffff;
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 11px;
  border: 2px solid #ffffff;
}

.preset-name {
  font-size: 12px;
  font-weight: 700;
  color: #1e293b;
}

.preset-tag {
  font-size: 10px;
  height: 18px;
  padding: 0 6px;
}

/* 默认印记恢复栏 */
.default-restore-box {
  display: flex;
  align-items: center;
  justify-content: space-between;
  background: #f8fafc;
  border: 1px dashed #cbd5e1;
  padding: 12px 16px;
  border-radius: 10px;
  margin-top: 4px;
}

.default-restore-left {
  display: flex;
  align-items: center;
  gap: 12px;
}

.restore-mini-circle {
  width: 36px;
  height: 36px;
  border-radius: 50%;
  color: #ffffff;
  font-weight: 800;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 15px;
  flex-shrink: 0;
  box-shadow: 0 2px 6px rgba(0,0,0,0.15);
}

.restore-title {
  font-size: 13px;
  font-weight: 700;
  color: #1e293b;
}

.restore-desc {
  font-size: 11px;
  color: #64748b;
}

/* 底部按钮栏 */
.dialog-footer-split {
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.footer-tip {
  font-size: 12px;
  color: #64748b;
  display: flex;
  align-items: center;
  gap: 4px;
}

.footer-btn-group {
  display: flex;
  align-items: center;
  gap: 10px;
}

</style>
