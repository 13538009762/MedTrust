<template>
  <!-- WP-16: 跨院检索页头与业务提示条 (从 CrossQueryView 迁入, 文案保持原样) -->
  <div>
    <div class="page-header">
      <div class="header-main-row">
        <div>
          <div class="header-badge-row">
            <el-button
              v-if="isFromEncounter || hasEncounterDraft"
              type="primary"
              :icon="ArrowLeft"
              class="top-left-back-btn"
              @click="$emit('return-encounter')"
            >
              返回接诊录入
            </el-button>
            <el-button
              v-else-if="hasIdCardQuery"
              type="default"
              plain
              :icon="ArrowLeft"
              class="top-left-back-btn"
              @click="$emit('back-records')"
            >
              返回门诊列表
            </el-button>
            <h2 class="page-title">{{ scopeMeta.title }}</h2>
            <el-tag :type="scopeMeta.tagType" effect="dark" class="scope-header-tag">
              {{ scopeMeta.tag }}
            </el-tag>
          </div>
          <p class="page-sub">{{ scopeMeta.sub }}</p>
        </div>
        <div class="header-action-group">
          <el-button type="danger" plain @click="router.push('/doctor/patient-search')">
            <el-icon style="margin-right: 4px;"><FirstAidKit /></el-icon>
            患者全景与急救速查
          </el-button>
          <el-button :icon="Refresh" circle @click="$emit('refresh')" title="刷新列表数据" />
        </div>
      </div>
    </div>

    <!-- 快捷调阅患者提示条 (由门诊工作台跳转时展示) -->
    <el-alert
      v-if="hasIdCardQuery"
      type="success"
      show-icon
      class="mb-3 patient-route-alert"
      :closable="false"
    >
      <template #title>
        <div class="alert-title-row" style="display: flex; align-items: center; justify-content: space-between; width: 100%;">
          <span>
            <strong>已自动定位目标患者：</strong>
            姓名：<strong style="color: #047857;">{{ patientName || '患者' }}</strong> |
            身份证号：<code style="font-weight: bold; background: #ecfdf5; padding: 2px 6px; border-radius: 4px;">{{ idCard }}</code>。
            系统已自动全网检索其所有就诊档案，请在下方列表核验并选择<strong>【申请调阅 (知情同意/破窗)】</strong>或直接查阅。
          </span>
          <div class="alert-btn-group" style="display: flex; gap: 8px; align-items: center; flex-shrink: 0; margin-left: 16px;">
            <el-button
              size="small"
              class="btn-cross-ai-diagnose"
              :icon="DocumentChecked"
              @click="$emit('open-cross-ai')"
            >
              跨院病历 SOAP 汇总与线索
            </el-button>
          </div>
        </div>
      </template>
    </el-alert>

    <!-- 录入草稿保留提示条 (无目标卡号参数但存在草稿时) -->
    <el-alert
      v-if="!hasIdCardQuery && hasEncounterDraft"
      type="warning"
      show-icon
      class="mb-3 draft-notice-alert"
      :closable="false"
    >
      <template #title>
        <div style="display: flex; align-items: center; justify-content: space-between; width: 100%;">
          <span>检测到您有一份尚未完成的接诊录入草稿正在妥善暂存中，查阅既往病历后可随时点击返回继续填写。</span>
          <el-button size="small" type="primary" :icon="ArrowLeft" @click="$emit('return-encounter')">
            返回继续录入病历
          </el-button>
        </div>
      </template>
    </el-alert>

    <!-- 业务规范与当前分支放行规则说明条 -->
    <el-alert
      :title="scopeMeta.alertTitle"
      :type="scopeMeta.alertType"
      :closable="false"
      show-icon
      class="mb-3 rule-alert"
    >
      <template #default>
        <div class="rule-alert-content">
          <span>{{ scopeMeta.alertText }}</span>
        </div>
      </template>
    </el-alert>
  </div>
</template>

<script setup lang="ts">
// WP-16: 检索页头/快捷调阅提示/草稿提示/业务规范说明条, 从 CrossQueryView 迁入
import { useRouter } from 'vue-router'
import { Refresh, ArrowLeft, DocumentChecked, FirstAidKit } from '@element-plus/icons-vue'
import type { CrossScopeMeta } from '@/composables/useCrossQueryScope'

defineProps<{
  scopeMeta: CrossScopeMeta
  isFromEncounter: boolean
  hasEncounterDraft: boolean
  hasIdCardQuery: boolean
  patientName: string | undefined
  idCard: string | undefined
}>()

defineEmits<{
  (e: 'return-encounter'): void
  (e: 'back-records'): void
  (e: 'open-cross-ai'): void
  (e: 'refresh'): void
}>()

const router = useRouter()
</script>

<style scoped src="../CrossQueryView.css"></style>