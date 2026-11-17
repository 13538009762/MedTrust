<template>
  <div class="doctor-records-container">
    <!-- 视图一：就诊记录列表与统计看板 (LIST) -->
    <div v-if="currentView === 'LIST'" class="records-list-wrapper">
      <EncounterTriageQueuePanel
        v-model:active-status-filter="activeStatusFilter"
        v-model:search-keyword="searchKeyword"
        v-model:current-page="currentPage"
        v-model:page-size="pageSize"
        :records="records"
        :filtered-records="filteredRecords"
        :paged-records="pagedRecords"
        :loading="loading"
        :waiting-exam-count="waitingExamCount"
        :exam-completed-count="examCompletedCount"
        :completed-count="completedCount"
        :is-wide-screen="isWideScreen"
        :has-saved-draft="hasSavedDraft"
        :draft-patient-name="draftPatientName"
        :format-patient-name="formatPatientName"
        @create-encounter="openInitialDialog"
        @restore-draft="checkAndRestoreDraft(false)"
        @refresh="fetchRecords"
        @open-final="openFinalDialog"
        @view-detail="handleViewDetail"
      />
    </div>

    <!-- 视图二：接诊初诊工作台 (CREATE) —— 子组件承载全部模板 (WP-16) -->
    <div v-else-if="currentView === 'CREATE'">
      <EncounterCreateDialog
        :patient-list="patientList"
        :sel-pid="initForm.patient_id"
        :si="selectedPatientInfo"
        :ia="selectedPatientInfectionAlert"
        v-model:m="encounterMode"
        v-model:exam-list="selectedExamList"
        :init-form="initForm"
        :vitals="vitals"
        :direct-form="directForm"
        :presets="prescriptionPresets"
        :submitting="submitting"
        :doctor-info="auth.user?.real_name || '陈建明' + ' (' + (auth.user?.title || '主任医师') + ' · ' + (auth.user?.department_name || '心血管内科') + ')'"
        :timeline-records="patientTimelineRecords"
        :timeline-loading="patientTimelineLoading"
        :current-hospital-id="auth.user?.hospital_id"
        @cancel="handleCancelEncounter()"
        @change-patient="changeSelectedPatient()"
        @quick-fill-vitals="quickFillVitals()"
        @open-ai-diagnostic-assist="openAIDiagnosticAssist()"
        @generate-direct-soap="generateDirectSOAP()"
        @apply-prescription-preset="(p: PrescriptionPreset) => applyPrescriptionPreset(p, 'DIRECT')"
        @fill-direct-preset="fillQuickDirectPreset()"
        @submit-direct="submitDirectEncounter()"
        @submit-exam="submitInitialEncounter()"
        @open-infection-guide="openCreateInfectionGuide()"
        @cross-query-patient="goToCrossQueryForPatient()"
        @on-select-patient="onPatientSelect"
        @on-pick-quick-patient="pickQuickPatient"
      />
    </div>

    <!-- 视图三：最终确诊与规范 SOAP 病历生成存证工作台 (FINAL) -->
    <div v-else-if="currentView === 'FINAL'" class="inpage-workstation-wrapper">
      <FinalDiagnosisDialog
        :record="activeRecordForFinal"
        :form-state="finalForm"
        :submitting-final="submittingFinal"
        :presets="prescriptionPresets as PrescriptionPreset[]"
        @back="currentView = 'LIST'"
        @apply-prescription-preset="applyFinalPreset"
        @generate-soap-record="generateSOAPRecord()"
        @submit-final="submitFinalEncounter()"
      />
    </div>

    <!-- 视图四：全景病历详情调阅工作台 (DETAIL) -->
    <div v-else-if="currentView === 'DETAIL'" :class="['inpage-workstation-wrapper', { 'tampered-watermark-overlay': detailRecord?.is_tampered }]">
      <div class="ws-page-header-card sticky-header">
        <div class="ws-header-left">
          <el-button :icon="ArrowLeft" plain size="small" @click="currentView = 'LIST'" class="back-btn">返回列表</el-button>
          <div class="ws-header-title-box">
            <span class="ws-page-title">临床就诊全景病历</span>
            <el-tag v-if="detailRecord?.is_tampered" size="small" type="danger" effect="dark" class="tamper-tag-glow">
              <el-icon class="mr-1"><WarningFilled /></el-icon> 底层数据库已被恶意篡改 · 存证不匹配
            </el-tag>
            <el-tag v-else-if="detailRecord?.verified" size="small" type="success" effect="light">已存证核验</el-tag>
            <el-tag v-else size="small" type="warning" effect="light">存证待复核</el-tag>
            <div v-if="detailRecord" class="ws-record-chip">
              <span class="chip-lbl">单号</span>
              <span class="chip-val mono">{{ detailRecord.record_no }}</span>
              <el-tooltip content="复制单号" placement="top">
                <el-icon class="copy-icon" @click.stop="copyToClipboard(detailRecord.record_no, '就诊单号')"><CopyDocument /></el-icon>
              </el-tooltip>
            </div>
          </div>
        </div>
        <div class="ws-header-right">
          <InfectionSafetyAlert
            v-if="detailRecord?.infection_alert?.has_risk" :alert="detailRecord.infection_alert" mode="badge"
          />
          <el-button type="primary" size="small" :icon="View" @click="handlePdfPreview(detailRecord)">查阅红头 PDF</el-button>
        </div>
      </div>

      <RecordDetailWorkstation
        :record="detailRecord" variant="local"
        :user-id="auth.user?.id" :user-hospital-id="auth.user?.hospital_id" :token="auth.token"
        :format-patient-name="formatPatientName" :mask-i-d-card="maskID"
        :format-hosp-name="formatHospName" :format-record-status="formatRecordStatus"
        @view-image="imageModalImageClick"
        @view-pdf="handlePdfPreview($event)"
        @download-image="downloadImageFile"
        @download-record="downloadRecordFile"
        @copy-text="(t: string, l: string) => copyToClipboard(t, l)"
        @back="currentView = 'LIST'"
      />
    </div>

    <!-- PDF 电子病历规范预览弹窗 -->
    <RedHeaderPdfPreviewDialog
      v-model:visible="pdfPreviewVisible" :record="previewingRecord"
      :format-hosp-name="formatHospName" :mask-i-d-card="maskID"
      @download-record="downloadRecordFile" @verify-integrity="triggerVerifyRecordIntegrity"
    />

    <!-- 影像大图全屏查阅弹窗 -->
    <el-dialog
      v-model="imageModalVisible" :title="imageModalTitle || '医学检验/影像检查原图'"
      width="860px" top="4vh"
    >
      <div class="text-center p-3">
        <img :src="imageModalUrl"
          style="max-width: 100%; max-height: 72vh; border-radius: 8px; box-shadow: 0 4px 16px rgba(0,0,0,0.15);" alt="医学图像" />
      </div>
      <template #footer>
        <el-button @click="imageModalVisible = false">关闭预览</el-button>
        <el-button type="primary" :icon="Download" @click="downloadImageFile()">下载原始影像文件</el-button>
      </template>
    </el-dialog>

    <!-- 链上完整性防篡改核验弹窗 -->
    <RecordIntegrityDialog
      v-model="integrityDialogVisible" :result="integrityResult" :loading="verifyingIntegrity"
      @download="downloadRecordFile(Number($event) ?? Number(previewingRecord?.id))"
    />

    <!-- 跨院既往病史 SOAP 简明汇总与隐匿线索提取弹窗 -->
    <AiDiagnosticDialog
      v-model="aiDiagnosticVisible" :patient="selectedPatientInfo"
      :current-symptoms="initForm.chief_complaint" :present-illness="initForm.present_illness"
      :vital-signs="formatVitalSignsText()" :department-name="initForm.department_name"
      @adopt-soap="handleAdoptSOAPFromAI"
    />
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch, onMounted, onUnmounted } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { View, Download, ArrowLeft, CopyDocument, WarningFilled } from '@element-plus/icons-vue'
import { ElMessage } from 'element-plus'
import { useAuthStore } from '../../stores/auth'
import AiDiagnosticDialog from '../../components/AiDiagnosticDialog.vue'
import RecordIntegrityDialog from '../../components/RecordIntegrityDialog.vue'
import InfectionSafetyAlert from '../../components/InfectionSafetyAlert.vue'
import EncounterTriageQueuePanel from './components/EncounterTriageQueuePanel.vue'
import FinalDiagnosisDialog from './components/FinalDiagnosisDialog.vue'
import RecordDetailWorkstation from './components/RecordDetailWorkstation.vue'
import RedHeaderPdfPreviewDialog from './components/RedHeaderPdfPreviewDialog.vue'
import EncounterCreateDialog from './components/EncounterCreateDialog.vue'
import { useEncounterWorkflow } from '@/composables/useEncounterWorkflow'
import { useDoctorRecordList } from '@/composables/useDoctorRecordList'
import { useDoctorRecordActions } from '@/composables/useDoctorRecordActions'
import { usePrescriptionPresets } from '@/composables/usePrescriptionPresets'
import type { MedicalRecordRow, PrescriptionPreset } from '@/types/clinical'

// ── 视图编排 ──
const currentView = ref<'LIST' | 'CREATE' | 'FINAL' | 'DETAIL'>('LIST')

// ── 屏幕宽度 ──
const windowWidth = ref(window.innerWidth)
function handleResize() {
  windowWidth.value = window.innerWidth
}
const isWideScreen = computed(() => windowWidth.value >= 1200)

// ── Router & Auth ──
const router = useRouter()
const route = useRoute()
const auth = useAuthStore()

// ── 列表 composable ──
const list = useDoctorRecordList()
const {
  loading, records, activeStatusFilter, searchKeyword,
  waitingExamCount, examCompletedCount, completedCount,
  currentPage, pageSize, filteredRecords, pagedRecords,
  formatPatientName, formatRecordStatus, fetchRecords, detailRecord, openViewDetail,
} = list

// ── 接诊工作流 composable ──
const wf = useEncounterWorkflow({
  setView: (view) => { currentView.value = view },
  refreshRecords: fetchRecords,
})
const {
  draftPatientName, hasSavedDraft, updateDraftState, saveEncounterDraft, checkAndRestoreDraft,
  selectedPatientInfo, selectedPatientInfectionAlert, patientList,
  patientTimelineRecords, patientTimelineLoading, selectedExamList,
  submitting, submittingFinal, encounterMode, initForm, vitals, directForm, finalForm, activeRecordForFinal,
  fetchPatients, onPatientSelect, pickQuickPatient, changeSelectedPatient,
  quickFillVitals, formatVitalSignsText, generateDirectSOAP, generateSOAPRecord,
  openInitialDialog, handleCancelEncounter,
  submitDirectEncounter, submitInitialEncounter, openFinalDialog, submitFinalEncounter,
} = wf
// pickQuickPatient handled via @on-pick-quick-patient event from EncounterCreateDialog

// ── 文件操作 composable ──
const fileActions = useDoctorRecordActions(detailRecord)
const {
  pdfPreviewVisible, previewingRecord,
  imageModalVisible, imageModalUrl, imageModalTitle,
  openImageModal, downloadImageFile, copyToClipboard, openPdfPreview: _openPdfPreview,
  promptTamperedRecordWarning,
  integrityDialogVisible, verifyingIntegrity, integrityResult, triggerVerifyRecordIntegrity,
  downloadRecordFile,
} = fileActions

// ── 处方预设 composable ──
const presets = usePrescriptionPresets(directForm, finalForm, generateDirectSOAP, generateSOAPRecord)
const { prescriptionPresets, applyPrescriptionPreset, fillQuickDirectPreset } = presets

// ── AI 辅助诊断 ──
const aiDiagnosticVisible = ref(false)

function openAIDiagnosticAssist() {
  if (!initForm.value.patient_id && !selectedPatientInfo.value?.id) {
    ElMessage.warning('请先在第 1 步选择或核验就诊患者')
    return
  }
  aiDiagnosticVisible.value = true
}

function handleAdoptSOAPFromAI(soap: { subjective?: string; s?: string; objective?: string; o?: string; assessment?: string; a?: string; plan?: string; p?: string }) {
  if (!soap) return
  const s = soap.subjective || soap.s || ''
  const o = soap.objective || soap.o || ''
  const a = soap.assessment || soap.a || ''
  const p = soap.plan || soap.p || ''

  if (s) initForm.value.chief_complaint = s
  if (o) initForm.value.diagnostic_basis = o

  if (encounterMode.value === 'DIRECT') {
    if (a) directForm.value.diagnosis = a
    if (p) directForm.value.treatment_plan = p
    generateDirectSOAP()
  } else {
    if (a) initForm.value.initial_diagnosis = a
  }
  ElMessage.success('已采纳跨院 SOAP 摘要结构化信息回填至接诊记录')
}

// ── 传染性预警引用 ──
const createInfectionAlertRef = ref<{ openGuideModal?: () => void } | null>(null)
function openCreateInfectionGuide() {
  createInfectionAlertRef.value?.openGuideModal?.()
}

// ── 格式化 helpers ──
function maskID(idCard?: string) {
  if (!idCard || idCard.length < 8) return '已脱敏保护'
  return idCard.substring(0, 6) + '********' + idCard.substring(idCard.length - 4)
}

function formatHospName(hId?: number) {
  if (hId === 1) return '第一人民医院'
  if (hId === 2) return '第二人民医院'
  if (hId === 3) return '第三人民医院'
  return '医疗中心'
}

// ── 跨院跳转 ──
function goToCrossQueryForPatient() {
  if (!selectedPatientInfo.value) {
    ElMessage.warning('请先在左侧选择就诊患者')
    return
  }
  const idCard = selectedPatientInfo.value.id_card || selectedPatientInfo.value.real_name
  saveEncounterDraft()
  ElMessage.info({
    message: `已暂存患者【${selectedPatientInfo.value.real_name}】初诊录入内容，调阅病历后点击左上角即可返回继续编辑`,
    duration: 3500
  })
  router.push({
    path: '/doctor/cross-query',
    query: {
      id_card: idCard,
      patient_name: selectedPatientInfo.value.real_name,
      patient_id: selectedPatientInfo.value.id,
      scope: 'all',
      from: 'encounter_create'
    }
  })
}

// ── 事件代理 ──
function handleViewDetail(row: MedicalRecordRow) {
  openViewDetail(row, (v) => { currentView.value = v }, promptTamperedRecordWarning)
}

function handlePdfPreview(rec: MedicalRecordRow | null) {
  _openPdfPreview(rec)
}

interface ImageFileArg { id: number; file_name?: string }
function imageModalImageClick(file: ImageFileArg) {
  openImageModal(file)
}

// 终诊弹窗专用的处方预设调用
function applyFinalPreset(preset: PrescriptionPreset, target: string) {
  applyPrescriptionPreset(preset, target || 'FINAL')
}

watch([activeStatusFilter, searchKeyword], () => {
  currentPage.value = 1
})

onMounted(() => {
  handleResize()
  window.addEventListener('resize', handleResize)
  fetchRecords()
  fetchPatients()
  updateDraftState()

  if (route.query.restore_encounter === '1') {
    const ok = checkAndRestoreDraft(false)
    if (ok) {
      router.replace({ path: '/doctor/records' })
    }
  }
})

watch(() => route.query.restore_encounter, (val) => {
  if (val === '1') {
    const ok = checkAndRestoreDraft(false)
    if (ok) {
      router.replace({ path: '/doctor/records' })
    }
  }
})

onUnmounted(() => {
  window.removeEventListener('resize', handleResize)
})
</script>

<style scoped src="./DoctorRecordsView.css"></style>
