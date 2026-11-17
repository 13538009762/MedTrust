<template>
  <div ref="pageRootRef" class="page-container">
    <!-- 视图一：跨院病例检索与多维筛选视图 (QUERY) -->
    <div v-if="currentView === 'QUERY'" class="query-main-wrapper">
      <CrossQueryHeaderBar
        :scope-meta="scopeMeta"
        :is-from-encounter="isFromEncounter"
        :has-encounter-draft="hasEncounterDraft"
        :has-id-card-query="hasIdCardQuery"
        :patient-name="routePatientName"
        :id-card="routeIdCard"
        @return-encounter="returnToEncounter"
        @back-records="router.push('/doctor/records')"
        @open-cross-ai="openCrossAIDiagnostic"
        @refresh="refreshData"
      />

    <!-- 检索与分支快捷导航区 -->
    <PatientDirectorySearchPanel
      v-model:current-scope="currentScope"
      v-model:selected-hospital="selectedHospital"
      v-model:selected-data-type="selectedDataType"
      v-model:search-keyword="searchKeyword"
      :stats-count="statsCount"
      :loading="loading"
      @search="searchCrossRecords"
    />

    <!-- 该患者做过的所有检查单与医学影像概览专区 (等待路由入场过渡完成后挂载，防止阻塞入场动画帧) -->
    <CrossExamOverviewCard
      v-if="contentReady"
      :results="results"
      :search-keyword="searchKeyword"
      :patient-name="routePatientName"
      :user-id="auth.user?.id"
      :user-hospital-id="auth.user?.hospital_id"
      :get-parsed-exam="getParsedExam"
      @access="handleAccess"
    />

    <!-- 跨院记录结果列表 (含多维筛选与自定义列；入场期间显示轻量骨架占位以保持 60/120fps 丝滑过渡) -->
    <CrossRecordResultTable
      v-if="contentReady"
      v-model:quick-classify-mode="quickClassifyMode"
      v-model:filter-auth-status="filterAuthStatus"
      v-model:filter-clinical-status="filterClinicalStatus"
      v-model:filter-tamper-status="filterTamperStatus"
      v-model:filter-time-range="filterTimeRange"
      v-model:current-page="currentPage"
      v-model:page-size="pageSize"
      :results="results"
      :filtered-results="filteredResults"
      :paginated-results="paginatedResults"
      :loading="loading"
      :table-max-height="tableMaxHeight"
      :is-wide-screen="isWideScreen"
      :user-id="auth.user?.id"
      :user-hospital-id="auth.user?.hospital_id"
      :search-keyword="searchKeyword"
      :active-filter-count="activeFilterCount"
      :table-columns="tableColumns"
      :visible-columns-count="visibleColumnsCount"
      :format-patient-name="formatPatientName"
      :mask-i-d-card="maskIDCard"
      :get-parsed-exam="getParsedExam"
      :get-exam-table-summary="getExamTableSummary"
      :is-column-visible="isColumnVisible"
      @access="handleAccess"
      @break-glass="bgFlow.openBreakGlassDirect"
      @reset-filters="resetAllFilters"
      @save-columns="saveColumnsConfig"
      @select-all-columns="selectAllColumns"
      @reset-columns="resetColumns"
      @size-change="handleSizeChange"
      @current-change="handleCurrentChange"
    />
    <el-card v-else shadow="hover" class="box-card table-card">
      <template #header>
        <div class="table-card-header">
          <div class="tch-left">
            <span class="font-bold">跨机构就诊与医技病历列表</span>
            <el-tag type="info" size="small" effect="plain" class="ml-2">
              正在载入...
            </el-tag>
          </div>
          <span class="cross-table-loading-tip">正在载入多维检索工作台...</span>
        </div>
      </template>
      <div class="cross-table-placeholder-body">
        <el-skeleton :rows="6" animated />
      </div>
    </el-card>
    </div>
    <!-- 视图二：跨院医疗数据已核准解密放行全景工作台 (DETAIL) -->
    <div v-else-if="currentView === 'DETAIL'" :class="['inpage-workstation-wrapper', { 'tampered-watermark-overlay': releasedRecord?.is_tampered }]">
      <!-- 顶部工作台粘性控制栏 (轻量纤细版 48px) -->
      <div class="ws-page-header-card sticky-header">
        <div class="ws-header-left">
          <el-button :icon="ArrowLeft" plain size="small" @click="currentView = 'QUERY'" class="back-btn">返回检索</el-button>
          <el-button
            v-if="isFromEncounter || hasEncounterDraft"
            type="primary"
            size="small"
            :icon="ArrowLeft"
            @click="returnToEncounter"
            class="back-btn draft-back-btn"
          >
            返回接诊录入
          </el-button>
          <div class="ws-header-title-box">
            <span class="ws-page-title">跨院解密就诊全景病历</span>
            <el-tag
              v-if="releasedRecord?.is_tampered"
              size="small"
              type="danger"
              effect="dark"
              class="tamper-tag-glow"
            >
              <el-icon class="mr-1"><WarningFilled /></el-icon> 数据已被恶意篡改 · 存证不匹配
            </el-tag>
            <el-tag v-else-if="releasedRecord?.verified" size="small" type="success" effect="light">已放行解密 (存证一致)</el-tag>
            <el-tag v-else size="small" type="warning" effect="light">已放行解密 (存证待复核)</el-tag>
            <div v-if="releasedRecord" class="ws-record-chip">
              <span class="chip-lbl">单号</span>
              <span class="chip-val mono">{{ releasedRecord.record_no }}</span>
              <el-tooltip content="复制单号" placement="top">
                <el-icon class="copy-icon" @click.stop="copyToClipboard(releasedRecord.record_no, '就诊单号')"><CopyDocument /></el-icon>
              </el-tooltip>
            </div>
          </div>
        </div>
        <div class="ws-header-right">
          <InfectionSafetyAlert
            v-if="releasedRecord?.infection_alert?.has_risk"
            :alert="releasedRecord.infection_alert!"
            mode="badge"
          />
          <el-button type="primary" size="small" :icon="View" @click="openPdfPreview(releasedRecord)">查阅红头 PDF</el-button>
        </div>
      </div>

      <RecordDetailWorkstation
        :record="releasedRecord"
        variant="cross"
        :user-id="auth.user?.id"
        :user-hospital-id="auth.user?.hospital_id"
        :is-break-glass-release="isBreakGlassRelease"
        :token="auth.token"
        :format-patient-name="formatPatientName"
        :mask-i-d-card="maskIDCard"
        :format-hosp-name="formatHospName"
        :format-record-status="formatRecordStatus"
        @view-image="openImageModal"
        @view-pdf="openPdfPreview"
        @download-image="downloadImageFile"
        @download-record="downloadRecordFile"
        @copy-text="copyToClipboard"
        @back="currentView = 'QUERY'"
      />
    </div>
    <!-- 调阅网关三态弹窗 (知情同意 / 现场密钥解锁 / Break-Glass) -->
    <AccessRequestAndKeyUnlockDialog
      v-if="hasOpenedBreakGlass"
      v-model:visible="bgFlow.breakGlassVisible.value"
      v-model:access-channel="bgFlow.accessChannel.value"
      v-model:patient-key="bgFlow.patientKey.value"
      v-model:captcha-answer="bgFlow.unlockCaptchaAnswer.value"
      :risk-decision-state="bgFlow.riskDecisionState.value"
      :risk-decision-req-no="bgFlow.riskDecisionReqNo.value"
      :intercept-risk-eval="bgFlow.interceptRiskEval.value"
      :display-risk-score="bgFlow.displayRiskScore.value"
      :consent-form="bgFlow.consentForm.value"
      :bg-form="bgFlow.bgForm.value"
      :current-target="currentTarget"
      :captcha-img="bgFlow.unlockCaptchaImg.value"
      :captcha-token="bgFlow.unlockCaptchaToken.value"
      :unlocking-by-key="bgFlow.unlockingByKey.value"
      :submitting-consent="bgFlow.submittingConsent.value"
      :submitting-b-g="bgFlow.submittingBG.value"
      :restricted="auth.user?.status === 'RESTRICTED'"
      :format-hosp-name="formatHospName"
      @unlock-by-key="bgFlow.handleUnlockByKey"
      @fetch-captcha="bgFlow.fetchUnlockCaptcha"
      @submit-consent="bgFlow.submitConsentRequest"
      @submit-break-glass="bgFlow.submitBreakGlass"
      @enter-detail="enterDetailFromAllowed"
    />

    <!-- PDF 电子病历规范预览弹窗 -->
    <RedHeaderPdfPreviewDialog
      v-if="hasOpenedPdf"
      v-model:visible="pdfPreviewVisible"
      :record="previewingRecord"
      :format-hosp-name="formatHospName"
      :mask-i-d-card="maskIDCard"
      @download-record="downloadRecordFile"
      @verify-integrity="verifyRecordIntegrity"
    />

    <!-- 影像大图全屏查阅弹窗 -->
    <el-dialog
      v-if="hasOpenedImage"
      v-model="imageModalVisible"
      :title="imageModalTitle || '医学检验/影像检查原图'"
      width="860px"
      top="4vh"
    >
      <div class="text-center p-3">
        <img
          :src="imageModalUrl"
          style="max-width: 100%; max-height: 72vh; border-radius: 8px; box-shadow: 0 4px 16px rgba(0,0,0,0.15);"
          alt="医学图像"
        />
      </div>
      <template #footer>
        <el-button @click="imageModalVisible = false">关闭预览</el-button>
        <el-button type="primary" :icon="Download" @click="downloadImageFile()">
          下载原始影像文件
        </el-button>
      </template>
    </el-dialog>

    <!-- 链上完整性防篡改核验弹窗 (密码学链上背书核验, WP-16 迁入子组件) -->
    <ChainIntegrityVerifyDialog
      ref="integrityDialog"
      @download="downloadRecordFile($event || previewingRecord?.id)"
    />

    <!-- AI 跨期全景病程与疾病关联推演工作台弹窗 -->
    <AiDiagnosticDialog
      v-if="hasOpenedCrossAI"
      v-model="crossAIDiagnosticVisible"
      :patient="crossTargetPatient"
      :current-symptoms="''"
      :present-illness="''"
      :department-name="'跨机构联合就诊中心'"
    />
  </div>
</template>

<script setup lang="ts">
// WP-16 拆分编排器：模板引用子组件展示，业务逻辑保留在视图层与 composables 中，文案与交互保持原样。
import { ref, computed, watch, onMounted, onBeforeUnmount } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { WarningFilled, View, Download, ArrowLeft, CopyDocument } from '@element-plus/icons-vue'
import { ElMessage } from 'element-plus'
import { getRecordDetailApi } from '../../api'
import { useAuthStore } from '../../stores/auth'
import { formatPatientName, formatRecordStatus, formatHospName, maskIDCard } from '../../utils/recordFormatters'
import InfectionSafetyAlert from '../../components/InfectionSafetyAlert.vue'
import AiDiagnosticDialog from '../../components/AiDiagnosticDialog.vue'
import PatientDirectorySearchPanel from './components/PatientDirectorySearchPanel.vue'
import CrossRecordResultTable from './components/CrossRecordResultTable.vue'
import CrossExamOverviewCard from './components/CrossExamOverviewCard.vue'
import CrossQueryHeaderBar from './components/CrossQueryHeaderBar.vue'
import AccessRequestAndKeyUnlockDialog from './components/AccessRequestAndKeyUnlockDialog.vue'
import RecordDetailWorkstation from './components/RecordDetailWorkstation.vue'
import RedHeaderPdfPreviewDialog from './components/RedHeaderPdfPreviewDialog.vue'
import ChainIntegrityVerifyDialog from './components/ChainIntegrityVerifyDialog.vue'
import { useBreakGlassFlow } from '@/composables/useBreakGlassFlow'
import { useCrossQueryScope } from '@/composables/useCrossQueryScope'
import { useCrossRecordQuery } from '@/composables/useCrossRecordQuery'
import { useCrossRecordFilters } from '@/composables/useCrossRecordFilters'
import { useCrossTableColumns } from '@/composables/useCrossTableColumns'
import { useTamperWarning } from '@/composables/useTamperWarning'
import { useRecordFileActions } from '@/composables/useRecordFileActions'
import { useViewportMetrics } from '@/composables/useViewportMetrics'
import { useCrossAIDiagnostic } from '@/composables/useCrossAIDiagnostic'
import type { MedicalRecordRow } from '@/types/clinical'

const currentView = ref<'QUERY' | 'DETAIL'>('QUERY')

const route = useRoute()
const router = useRouter()
const auth = useAuthStore()

const isFromEncounter = computed(() => route.query.from === 'encounter_create')

const hasEncounterDraft = computed(() => {
  try {
    const draft = sessionStorage.getItem('medtrust_encounter_draft')
    return !!draft
  } catch {
    return false
  }
})

const hasIdCardQuery = computed(() => !!route.query.id_card)
const routePatientName = computed(() => (route.query.patient_name as string) || undefined)
const routeIdCard = computed(() => (route.query.id_card as string) || undefined)

function returnToEncounter() {
  router.push({
    path: '/doctor/records',
    query: {
      restore_encounter: '1'
    }
  })
}

// ── 路由入场过渡与重型子区协同解锁，保证 fade-slide 入场全帧丝滑 ────────────────
const pageRootRef = ref<HTMLElement | null>(null)
const contentReady = ref(false)
let resolveEnterSettled: (() => void) | null = null
const enterSettledPromise = new Promise<void>((resolve) => {
  resolveEnterSettled = resolve
})
let settleTimer: ReturnType<typeof setTimeout> | null = null
let activeTransitionEl: HTMLElement | null = null

function parseDurationMs(raw: string): number {
  const token = raw.trim()
  const val = parseFloat(token) || 0
  return token.endsWith('ms') ? val : val * 1000
}

function cleanupEnterListeners() {
  if (settleTimer !== null) {
    clearTimeout(settleTimer)
    settleTimer = null
  }
  if (activeTransitionEl) {
    activeTransitionEl.removeEventListener('transitionend', onRootTransitionEnd)
    activeTransitionEl = null
  }
  if (resolveEnterSettled) {
    const fn = resolveEnterSettled
    resolveEnterSettled = null
    fn()
  }
}

function markEnterSettled() {
  cleanupEnterListeners()
  if (!contentReady.value) {
    contentReady.value = true
  }
}

function onRootTransitionEnd(ev: TransitionEvent) {
  if (ev.target === activeTransitionEl && (ev.propertyName === 'opacity' || ev.propertyName === 'transform')) {
    markEnterSettled()
  }
}

onMounted(() => {
  const el = pageRootRef.value
  if (!el || !el.classList.contains('fade-slide-enter-active')) {
    markEnterSettled()
    return
  }
  const durations = (window.getComputedStyle(el).transitionDuration || '0s')
    .split(',')
    .map(parseDurationMs)
  const maxDurMs = Math.max(0, ...durations)
  if (maxDurMs <= 0) {
    markEnterSettled()
    return
  }
  activeTransitionEl = el
  el.addEventListener('transitionend', onRootTransitionEnd)
  settleTimer = setTimeout(markEnterSettled, Math.min(maxDurMs + 60, 400))
})

onBeforeUnmount(() => {
  cleanupEnterListeners()
})

// ── 分支范围元信息与数据获取/筛选/表格列 (composables) ────────────────
const { currentScope, scopeMeta } = useCrossQueryScope(route, router)

const {
  statsCount, selectedHospital, selectedDataType, searchKeyword, results, loading,
  getParsedExam, getExamTableSummary, searchCrossRecords, refreshData,
} = useCrossRecordQuery(route, currentScope, {
  waitForEnterTransition: () => enterSettledPromise,
})

const {
  quickClassifyMode, filterAuthStatus, filterClinicalStatus, filterTamperStatus, filterTimeRange,
  activeFilterCount, resetAllFilters, filteredResults,
  currentPage, pageSize, paginatedResults, handleSizeChange, handleCurrentChange,
} = useCrossRecordFilters(results, getParsedExam, searchKeyword, route)

const { tableColumns, saveColumnsConfig, isColumnVisible, resetColumns, selectAllColumns, visibleColumnsCount } = useCrossTableColumns()

const { isWideScreen, tableMaxHeight } = useViewportMetrics()

// ── AI 跨期全景病程推演与关联分析 ────────────────────────
const { crossAIDiagnosticVisible, crossTargetPatient, openCrossAIDiagnostic } = useCrossAIDiagnostic(route, results)

// ── 跨院调阅网关三态流转 (composable) ────────────────────────
const currentTarget = ref<MedicalRecordRow | null>(null)
const releasedRecord = ref<MedicalRecordRow | null>(null)
const isBreakGlassRelease = ref(false)

const { promptTamperedRecordWarning } = useTamperWarning()

const {
  pdfPreviewVisible, previewingRecord, imageModalVisible, imageModalUrl, imageModalTitle,
  openImageModal, downloadImageFile, openPdfPreview, downloadRecordFile, copyToClipboard,
} = useRecordFileActions(releasedRecord)

const bgFlow = useBreakGlassFlow(currentTarget, {
  onReleased: (record, via) => {
    releasedRecord.value = record
    isBreakGlassRelease.value = via === 'BREAK_GLASS'
    currentView.value = 'DETAIL'
    if (record.is_tampered) {
      promptTamperedRecordWarning(record)
    }
  },
  onAllowedDirect: (record, isEmergency) => {
    releasedRecord.value = record
    isBreakGlassRelease.value = isEmergency
    currentView.value = 'DETAIL'
    if (record.is_tampered) {
      promptTamperedRecordWarning(record)
    }
  },
  onRefresh: refreshData,
})

// ── 弹窗惰性挂载标记 (首次打开后保持挂载以完整保留关闭过渡) ────────────────
const hasOpenedBreakGlass = ref(false)
const hasOpenedPdf = ref(false)
const hasOpenedImage = ref(false)
const hasOpenedCrossAI = ref(false)

watch(() => bgFlow.breakGlassVisible.value, (val) => {
  if (val) hasOpenedBreakGlass.value = true
}, { immediate: true })
watch(pdfPreviewVisible, (val) => {
  if (val) hasOpenedPdf.value = true
}, { immediate: true })
watch(imageModalVisible, (val) => {
  if (val) hasOpenedImage.value = true
}, { immediate: true })
watch(crossAIDiagnosticVisible, (val) => {
  if (val) hasOpenedCrossAI.value = true
}, { immediate: true })

// ── 链上完整性核验弹窗 (子组件 expose 的 verifyRecordIntegrity) ────────────────
const integrityDialog = ref<InstanceType<typeof ChainIntegrityVerifyDialog> | null>(null)

function verifyRecordIntegrity(recordId: number) {
  integrityDialog.value?.verifyRecordIntegrity(recordId)
}

async function enterDetailFromAllowed() {
  if (!currentTarget.value) return
  bgFlow.breakGlassVisible.value = false
  try {
    const recRes = await getRecordDetailApi(currentTarget.value.id)
    if (recRes.code === 200) {
      releasedRecord.value = recRes.data as MedicalRecordRow
      currentView.value = 'DETAIL'
      if (releasedRecord.value?.is_tampered) {
        promptTamperedRecordWarning(releasedRecord.value)
      }
    }
  } catch (err: unknown) {
    const e = err as { message?: string }
    ElMessage.error(e?.message || '获取病历详情失败')
  }
}

async function handleAccess(row: MedicalRecordRow) {
  currentTarget.value = row
  const isSelf = row.doctor_id === auth.user?.id
  const isSameHosp = row.hospital_id === auth.user?.hospital_id

  // 1. 若当前已具备调阅权（本人责任病历、同院、跨院已授权或24小时破窗有效期内），优先直接放行展示
  if (row.has_access || isSelf || isSameHosp) {
    try {
      const recRes = await getRecordDetailApi(row.id)
      if (recRes.code === 200) {
        releasedRecord.value = recRes.data as MedicalRecordRow
        isBreakGlassRelease.value = (row.access_type === 'BREAK_GLASS')
        currentView.value = 'DETAIL'
        if (releasedRecord.value?.is_tampered) {
          promptTamperedRecordWarning(releasedRecord.value)
        } else {
          ElMessage.success('【调阅放行】已核验权限凭据，解密病历成功')
        }
        return
      }
    } catch (err: unknown) {
      const e = err as { message?: string }
      console.warn('Direct record fetch failed, falling back to gateway access evaluation', e)
    }
  }

  // 2. 跨院未授权记录，向安全规则与风险网关提交调阅申请
  await bgFlow.submitAccessRequest(row)
}
</script>

<style scoped src="./CrossQueryView.css"></style>
