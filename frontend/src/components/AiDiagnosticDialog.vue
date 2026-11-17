<template>
  <el-dialog
    v-model="visible"
    title="跨院既往病史 SOAP 客观摘要（经治医生复核）"
    width="980px"
    class="ai-diagnostic-dialog"
    :close-on-click-modal="false"
    destroy-on-close
    append-to-body
  >
    <!-- 弹窗顶栏：患者档案与授权调阅状态摘要 -->
    <div class="ai-header-strip">
      <div class="patient-brief-box">
        <div class="patient-avatar-circle">
          {{ (patientInfo?.real_name || patientInfo?.username || '患').slice(0, 1) }}
        </div>
        <div class="patient-text-meta">
          <div class="p-name-row">
            <span class="p-name font-bold">{{ patientInfo?.real_name || patientInfo?.username || '就诊患者' }}</span>
            <el-tag size="small" type="primary" effect="plain">
              {{ formatGender(patientInfo?.gender || diagnosticResult?.patient_gender) }} / {{ (patientInfo?.age || diagnosticResult?.patient_age) ? `${patientInfo?.age || diagnosticResult?.patient_age}岁` : '年龄未登记' }}
            </el-tag>
            <span class="p-id-card font-mono text-muted">档案号: {{ patientInfo?.user_no || '—' }}</span>
          </div>
          <div class="p-auth-stats">
            <span class="stat-pill success">
              <el-icon><CircleCheckFilled /></el-icon>
              已授权调阅 <strong>{{ diagnosticResult?.authorized_record_count || authCount }}</strong> 份跨期病历
            </span>
            <span class="stat-pill info">
              <el-icon><Document /></el-icon>
              关联 <strong>{{ diagnosticResult?.total_exam_order_count || examCount }}</strong> 份医技化验单
            </span>
          </div>
        </div>
      </div>

      <div class="engine-badge-box">
        <div class="engine-tag" :class="diagnosticResult?.engine_mode === 'EXTERNAL_API' ? 'llm' : 'knowledge'">
          <el-icon><Cpu /></el-icon>
          <span>{{ diagnosticResult?.engine_mode === 'EXTERNAL_API' ? '外部大模型 API' : 'MedTrust 跨院病历语义提炼引擎' }}</span>
        </div>
        <div class="model-caption font-mono">
          {{ diagnosticResult?.model_name || 'MedTrust-SOAP-Extractor-V2' }}
        </div>
      </div>
    </div>

    <!-- 正在推理中的动态流水线动画 -->
    <div v-if="loading" class="ai-loading-pipeline">
      <div class="pipeline-spinner-ring">
        <el-icon class="is-loading" :size="48" color="#0284c7"><Loading /></el-icon>
      </div>
      <div class="pipeline-title">正在执行跨机构病历授权核验、SOAP 摘要生成与隐匿线索抽取...</div>
      <div class="pipeline-steps-row">
        <div class="step-node" :class="{ done: pipelineStep >= 1, active: pipelineStep === 1 }">
          <div class="step-dot">1</div>
          <span class="step-text">调阅已授权跨期病历档案与化验单</span>
        </div>
        <div class="step-line" :class="{ filled: pipelineStep >= 2 }"></div>
        <div class="step-node" :class="{ done: pipelineStep >= 2, active: pipelineStep === 2 }">
          <div class="step-dot">2</div>
          <span class="step-text">测算发病时间间隔与演化周期</span>
        </div>
        <div class="step-line" :class="{ filled: pipelineStep >= 3 }"></div>
        <div class="step-node" :class="{ done: pipelineStep >= 3, active: pipelineStep === 3 }">
          <div class="step-dot">3</div>
          <span class="step-text">比对既往用药史、抽取隐匿过敏线索</span>
        </div>
        <div class="step-line" :class="{ filled: pipelineStep >= 4 }"></div>
        <div class="step-node" :class="{ done: pipelineStep >= 4, active: pipelineStep === 4 }">
          <div class="step-dot">4</div>
          <span class="step-text">Go授权范围后置校验与提炼SOAP摘要</span>
        </div>
      </div>
    </div>

    <!-- 诊断结果主体展示区 -->
    <div v-else-if="diagnosticResult" class="diagnostic-result-container">
      <!-- 1. 综合诊断简述卡片 -->
      <div class="summary-hero-card">
        <div class="shc-header">
          <el-icon :size="20" color="#0284c7"><DocumentChecked /></el-icon>
          <span class="shc-title">跨院病历多维演进分析与 SOAP 结构化提炼</span>
          <el-tag size="small" type="success" effect="dark" class="ml-auto">
            {{ diagnosticResult.post_verification_guard || '已通过 Go 授权范围后置核验（仅聚合已授权病历）' }}
          </el-tag>
        </div>
        <div class="shc-body">
          {{ diagnosticResult.analysis_summary }}
        </div>
      </div>

      <!-- 选项卡导航切换各维度分析 (WP-16: 各 tab 内容拆分至 ai-diagnostic/ 子面板) -->
      <el-tabs v-model="activeTab" class="diagnostic-tabs">
        <el-tab-pane label="SOAP 接诊结构化汇总" name="soap">
          <SoapSummaryPanel :result="diagnosticResult" @adopt-soap="adoptSOAP" />
        </el-tab-pane>

        <el-tab-pane label="隐匿过敏与高危线索" name="clues">
          <HiddenCluesPanel :result="diagnosticResult" />
        </el-tab-pane>

        <el-tab-pane label="跨院记录语义矛盾提示" name="contradictions">
          <ContradictionPanel :result="diagnosticResult" />
        </el-tab-pane>

        <el-tab-pane label="跨期时序演变与发病间隔" name="timeline">
          <TimelineCorrelationPanel
            :result="diagnosticResult"
            :current-symptoms="currentSymptoms"
            :present-illness="presentIllness"
          />
        </el-tab-pane>

        <el-tab-pane label="用药史与化验单比对" name="correlation">
          <MedicationCorrelationPanel :result="diagnosticResult" />
        </el-tab-pane>

        <el-tab-pane label="既往处置追溯与核查线索" name="treatment">
          <TreatmentTracePanel :result="diagnosticResult" @adopt-treatment="adoptTreatment" />
        </el-tab-pane>

        <el-tab-pane label="AI提示词与API配置" name="prompt">
          <PromptConfigPanel :result="diagnosticResult" @copy-prompt="copyPrompt" />
        </el-tab-pane>
      </el-tabs>

      <!-- 底部临床安全防范与决策支持准则卡片 -->
      <div v-if="diagnosticResult.clinical_warnings?.length" class="clinical-warnings-box mt-3">
        <div class="cwb-header">
          <el-icon color="#ef4444"><WarningFilled /></el-icon>
          <span class="font-bold text-danger">经治医生临床安全防范与合规审方准则</span>
        </div>
        <ul class="cwb-list">
          <li v-for="(w, idx) in diagnosticResult.clinical_warnings" :key="idx">
            {{ w }}
          </li>
        </ul>
      </div>
    </div>

    <!-- 弹窗底部操作工具栏 -->
    <template #footer>
      <div class="ai-dialog-footer">
        <span class="footer-hint text-muted">
          <el-icon><InfoFilled /></el-icon>
          仅作历史病历客观汇总，不构成临床诊断或处方依据；最终诊断与处方权归属执业医师
        </span>
        <div class="footer-btn-group">
          <el-button @click="visible = false">关 闭</el-button>
          <el-button
            type="primary"
            plain
            :icon="Refresh"
            :loading="loading"
            @click="runDiagnosticAnalysis"
          >
            重新提取摘要
          </el-button>
          <el-button
            type="success"
            :icon="DocumentCopy"
            :disabled="!diagnosticResult?.soap_summary"
            @click="adoptAllToRecord"
          >
            引用历史摘要至草稿
          </el-button>
        </div>
      </div>
    </template>
  </el-dialog>
</template>

<script setup lang="ts">
import { ref, watch, computed } from 'vue'
import {
  DocumentChecked, CircleCheckFilled, Document, Cpu, Loading,
  InfoFilled, WarningFilled, Refresh, DocumentCopy
} from '@element-plus/icons-vue'
import { ElMessage } from 'element-plus'
import { diagnosticAssistApi } from '../api'
import SoapSummaryPanel from './ai-diagnostic/SoapSummaryPanel.vue'
import HiddenCluesPanel from './ai-diagnostic/HiddenCluesPanel.vue'
import ContradictionPanel from './ai-diagnostic/ContradictionPanel.vue'
import TimelineCorrelationPanel from './ai-diagnostic/TimelineCorrelationPanel.vue'
import MedicationCorrelationPanel from './ai-diagnostic/MedicationCorrelationPanel.vue'
import TreatmentTracePanel from './ai-diagnostic/TreatmentTracePanel.vue'
import PromptConfigPanel from './ai-diagnostic/PromptConfigPanel.vue'

const props = defineProps<{
  modelValue: boolean
  patient: any
  currentSymptoms?: string
  presentIllness?: string
  vitalSigns?: string
  departmentName?: string
}>()

const emit = defineEmits<{
  (e: 'update:modelValue', val: boolean): void
  (e: 'adopt-diagnosis', item: any): void
  (e: 'adopt-treatment', plan: any): void
  (e: 'adopt-soap', soap: any): void
}>()

const visible = computed({
  get: () => props.modelValue,
  set: (val) => emit('update:modelValue', val),
})

const loading = ref(false)
const pipelineStep = ref(1)
const activeTab = ref('soap')
const diagnosticResult = ref<any>(null)
const authCount = ref(4)
const examCount = ref(2)

const patientInfo = computed(() => props.patient || {})

function formatGender(gender?: string) {
  if (!gender) return '性别未登记'
  const g = String(gender).trim().toUpperCase()
  if (g === 'FEMALE' || g === '女' || g === 'F') return '女'
  if (g === 'MALE' || g === '男' || g === 'M') return '男'
  return gender
}

watch(
  () => props.modelValue,
  (val) => {
    if (val) {
      activeTab.value = 'soap'
      runDiagnosticAnalysis()
    }
  }
)

async function runDiagnosticAnalysis() {
  if (!props.patient?.id) {
    ElMessage.warning('尚未指定有效的就诊患者')
    return
  }

  loading.value = true
  pipelineStep.value = 1
  diagnosticResult.value = null

  // 动画流水线定时推进
  const stepTimer = setInterval(() => {
    if (pipelineStep.value < 4) {
      pipelineStep.value++
    }
  }, 350)

  try {
    const res: any = await diagnosticAssistApi({
      patient_id: props.patient.id,
      current_symptoms: props.currentSymptoms || '',
      present_illness: props.presentIllness || '',
      vital_signs: props.vitalSigns || '',
      department_name: props.departmentName || '全科/综合门诊',
    })

    clearInterval(stepTimer)
    pipelineStep.value = 4

    if (res.code === 200 && res.data) {
      diagnosticResult.value = res.data
    } else {
      ElMessage.warning(res.message || '临床摘要检索返回格式异常')
    }
  } catch (err: any) {
    clearInterval(stepTimer)
    console.error('[AI Diagnostic Assist Error]:', err)
    ElMessage.error(err?.response?.data?.message || err?.message || '调用临床线索聚合服务失败')
  } finally {
    loading.value = false
  }
}

function adoptSOAP(soap: any) {
  emit('adopt-soap', soap)
  if (navigator.clipboard && soap) {
    const text = `【S 主诉与既往病史】\n${soap.subjective || ''}\n\n【O 客观检查体征】\n${soap.objective || ''}\n\n【A 各院确诊归纳】\n${soap.assessment || ''}\n\n【P 历史处方用药】\n${soap.plan || ''}`
    navigator.clipboard.writeText(text).catch(() => {})
  }
  ElMessage.success('已采纳跨院 SOAP 摘要至病历草稿并复制到剪贴板')
}

function adoptTreatment(treatment: any) {
  emit('adopt-treatment', treatment)
  ElMessage.success('已引用既往处置摘要供医师核对参考')
}

function adoptAllToRecord() {
  if (!diagnosticResult.value?.soap_summary) {
    ElMessage.warning('暂无有效的 SOAP 摘要可采纳')
    return
  }
  emit('adopt-soap', diagnosticResult.value.soap_summary)
  if (diagnosticResult.value.recommended_treatment) {
    emit('adopt-treatment', diagnosticResult.value.recommended_treatment)
  }
  ElMessage.success('已采纳跨院 SOAP 结构化摘要与既往用药线索至当前病历草稿')
  visible.value = false
}

function copyPrompt() {
  if (!diagnosticResult.value?.raw_prompt) return
  navigator.clipboard.writeText(diagnosticResult.value.raw_prompt).then(() => {
    ElMessage.success('已复制 AI 临床推理完整提示词到剪贴板')
  })
}
</script>

<style scoped>
.ai-diagnostic-dialog :deep(.el-dialog__body) {
  padding: 12px 20px 20px 20px;
  background: #f8fafc;
}

/* 顶部患者与状态横幅 */
.ai-header-strip {
  background: #ffffff;
  border-radius: 12px;
  padding: 14px 18px;
  border: 1px solid #e2e8f0;
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 14px;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.04);
}

.patient-brief-box {
  display: flex;
  align-items: center;
  gap: 14px;
}

.patient-avatar-circle {
  width: 44px;
  height: 44px;
  border-radius: 8px;
  background: #0284c7;
  color: #ffffff;
  font-weight: 700;
  font-size: 18px;
  display: flex;
  align-items: center;
  justify-content: center;
  box-shadow: 0 2px 6px rgba(2, 132, 199, 0.25);
}

.patient-text-meta {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.p-name-row {
  display: flex;
  align-items: center;
  gap: 10px;
}

.p-name {
  font-size: 16px;
  color: #1e293b;
}

.p-auth-stats {
  display: flex;
  align-items: center;
  gap: 10px;
}

.stat-pill {
  font-size: 12px;
  display: flex;
  align-items: center;
  gap: 4px;
  padding: 2px 8px;
  border-radius: 6px;
}

.stat-pill.success {
  background: #ecfdf5;
  color: #047857;
}

.stat-pill.info {
  background: #eff6ff;
  color: #1d4ed8;
}

.engine-badge-box {
  text-align: right;
  display: flex;
  flex-direction: column;
  align-items: flex-end;
  gap: 4px;
}

.engine-tag {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 4px 10px;
  border-radius: 20px;
  font-size: 12px;
  font-weight: 600;
}

.engine-tag.knowledge {
  background: #f0f9ff;
  color: #0369a1;
  border: 1px solid #bae6fd;
}

.engine-tag.llm {
  background: #ecfdf5;
  color: #059669;
  border: 1px solid #a7f3d0;
}

.model-caption {
  font-size: 11px;
  color: #94a3b8;
}

/* 流水线加载动画 */
.ai-loading-pipeline {
  background: #ffffff;
  border-radius: 12px;
  padding: 48px 24px;
  text-align: center;
  border: 1px solid #e2e8f0;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 16px;
}

.pipeline-title {
  font-size: 16px;
  font-weight: 700;
  color: #334155;
}

.pipeline-steps-row {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-top: 16px;
  max-width: 820px;
  width: 100%;
  justify-content: center;
}

.step-node {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 6px;
  width: 160px;
}

.step-dot {
  width: 28px;
  height: 28px;
  border-radius: 50%;
  background: #e2e8f0;
  color: #64748b;
  font-size: 13px;
  font-weight: 700;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: all 0.3s ease;
}

.step-node.active .step-dot {
  background: #0284c7;
  color: #ffffff;
  box-shadow: 0 0 0 4px #e0f2fe;
}

.step-node.done .step-dot {
  background: #0d9488;
  color: #ffffff;
}

.step-text {
  font-size: 11px;
  color: #64748b;
  text-align: center;
  line-height: 1.3;
}

.step-line {
  flex: 1;
  height: 3px;
  background: #e2e8f0;
  margin-bottom: 22px;
  border-radius: 2px;
  transition: background 0.3s ease;
}

.step-line.filled {
  background: #0d9488;
}

/* 结论横幅 */
.summary-hero-card {
  background: #f0f9ff;
  border: 1px solid #bae6fd;
  border-left: 4px solid #0284c7;
  border-radius: 8px;
  padding: 14px 18px;
}

.shc-header {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 8px;
}

.shc-title {
  font-size: 14px;
  font-weight: 700;
  color: #0369a1;
}

.shc-body {
  font-size: 13px;
  line-height: 1.6;
  color: #334155;
}

/* 标签面板通用容器 */
.diagnostic-tabs :deep(.el-tabs__header) {
  margin-bottom: 12px;
}

/* 警告框 */
.clinical-warnings-box {
  background: #fef2f2;
  border: 1px solid #fecaca;
  border-radius: 8px;
  padding: 10px 14px;
}

.cwb-header {
  display: flex;
  align-items: center;
  gap: 6px;
  margin-bottom: 4px;
  font-size: 13px;
}

.cwb-list {
  margin: 0;
  padding-left: 18px;
  font-size: 12px;
  color: #991b1b;
  line-height: 1.5;
}

/* 底部栏 */
.ai-dialog-footer {
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.footer-hint {
  font-size: 12px;
  display: flex;
  align-items: center;
  gap: 4px;
}

.footer-btn-group {
  display: flex;
  gap: 10px;
}
</style>
