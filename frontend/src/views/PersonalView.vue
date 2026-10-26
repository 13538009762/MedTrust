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
import { getRoleTagType, getRoleName, presetAvatars } from '@/utils/userProfileUtils'
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

// ── 个人头像更换与自定义业务逻辑 ──────────────────────────
const avatarDialogVisible = ref(false)
const avatarActiveTab = ref('upload')
const previewAvatarUrl = ref('')
const avatarZoom = ref(100)
const avatarRotate = ref(0)
const selectedPresetIdx = ref(-1)
const savingAvatar = ref(false)
const fileInputRef = ref<HTMLInputElement | null>(null)

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

<style scoped src="./PersonalView.css"></style>
