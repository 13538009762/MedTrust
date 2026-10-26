<template>
  <el-dialog
    v-model="visible"
    title="📋 跨院既往病史 SOAP 结构化提炼与临床决策支持 (Doctor-in-the-Loop)"
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
              {{ patientInfo?.gender === 'FEMALE' ? '女' : '男' }} / {{ patientInfo?.age || 45 }}岁
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
          <el-icon :size="20" color="#0284c7"><MagicStick /></el-icon>
          <span class="shc-title">跨院病历多维演进分析与 SOAP 结构化提炼</span>
          <el-tag size="small" type="success" effect="dark" class="ml-auto">
            {{ diagnosticResult.post_verification_guard || '🛡️ 已通过 Go 授权范围后置核验 (仅聚合已授权病历)' }}
          </el-tag>
        </div>
        <div class="shc-body">
          {{ diagnosticResult.analysis_summary }}
        </div>
      </div>

      <!-- 选项卡导航切换各维度分析 -->
      <el-tabs v-model="activeTab" class="diagnostic-tabs">
        <!-- 维度 1: SOAP 接诊结构化汇总 (核心主视图) -->
        <el-tab-pane label="📋 SOAP 接诊结构化汇总" name="soap">
          <div class="tab-pane-box">
            <div class="pane-tip-row flex-between mb-3">
              <span>由医疗专业智能引擎对跨院已授权病程记录进行四段式客观提炼（严格遵循 Doctor-in-the-Loop 辅助诊疗与临床决策合规准则）：</span>
              <el-button type="primary" size="small" :icon="DocumentCopy" @click="adoptSOAP(diagnosticResult.soap_summary)">
                一键复制 SOAP 至病历草稿
              </el-button>
            </div>

            <div class="soap-grid-layout">
              <div class="soap-box s-box">
                <div class="soap-box-header">
                  <span class="soap-letter">S</span>
                  <span class="soap-title">患者主诉与既往病史 (Subjective)</span>
                </div>
                <div class="soap-box-body">
                  {{ diagnosticResult.soap_summary?.subjective || '暂无既往主诉' }}
                </div>
              </div>

              <div class="soap-box o-box">
                <div class="soap-box-header">
                  <span class="soap-letter">O</span>
                  <span class="soap-title">跨院客观检查与生命体征 (Objective)</span>
                </div>
                <div class="soap-box-body">
                  {{ diagnosticResult.soap_summary?.objective || '化验基本正常' }}
                </div>
              </div>

              <div class="soap-box a-box">
                <div class="soap-box-header">
                  <span class="soap-letter">A</span>
                  <span class="soap-title">各院既有临床确诊客观汇总 (Assessment)</span>
                </div>
                <div class="soap-box-body">
                  {{ diagnosticResult.soap_summary?.assessment || '暂无跨院确诊记录' }}
                </div>
                <div class="soap-box-tip text-muted">
                  ⚠️ 客观摘录各院既有病历确诊结论，AI 不做任何新疾病确诊，供接诊医生鉴别衔接。
                </div>
              </div>

              <div class="soap-box p-box">
                <div class="soap-box-header">
                  <span class="soap-letter">P</span>
                  <span class="soap-title">历史处置与已用药物记录 (Plan)</span>
                </div>
                <div class="soap-box-body">
                  {{ diagnosticResult.soap_summary?.plan || '暂无历史处方' }}
                </div>
                <div class="soap-box-tip text-muted">
                  ⚠️ 仅供用药史追溯与防范重复用药，当前处方用药由接诊医生亲自开具。
                </div>
              </div>
            </div>
          </div>
        </el-tab-pane>

        <!-- 维度 2: 隐匿过敏与高危线索 -->
        <el-tab-pane label="⚠️ 隐匿过敏与高危线索" name="clues">
          <div class="tab-pane-box">
            <div class="pane-tip-row mb-3">
              <span>从跨院已授权自由文本主诉、病程与现病史中自动抽取的口语化隐匿风险线索：</span>
            </div>
            <div class="clues-grid">
              <div
                v-for="(clue, cIdx) in (diagnosticResult.hidden_allergy_clues || [])"
                :key="cIdx"
                class="clue-item-card"
                :class="clue.clue_type.includes('过敏') ? 'clue-danger' : (clue.clue_type.includes('正常') ? 'clue-success' : 'clue-warning')"
              >
                <div class="clue-top-row">
                  <span class="clue-badge">{{ clue.clue_type }}</span>
                  <span class="clue-source font-mono text-muted text-xs">来源: {{ clue.source_hospital }} ({{ clue.source_record_no }})</span>
                </div>
                <div class="clue-text">{{ clue.content }}</div>
              </div>
            </div>
          </div>
        </el-tab-pane>

        <!-- 维度 3: 跨院记录语义矛盾提示 -->
        <el-tab-pane label="🔍 跨院记录语义矛盾提示" name="contradictions">
          <div class="tab-pane-box">
            <div class="pane-tip-row mb-3">
              <span>通过跨机构病历交叉比对，识别不同医院之间关于慢病、过敏或用药依从性的自述差异：</span>
            </div>
            <div class="contradictions-grid">
              <div
                v-for="(ct, ctIdx) in (diagnosticResult.semantic_contradictions || [])"
                :key="ctIdx"
                class="ct-card"
                :class="ct.contradiction_type.includes('正常') ? 'ct-success' : 'ct-warning'"
              >
                <div class="ct-top-row">
                  <span class="ct-tag font-bold">{{ ct.contradiction_type }}</span>
                  <el-tag size="small" :type="ct.contradiction_type.includes('正常') ? 'success' : 'warning'">
                    {{ ct.contradiction_type.includes('正常') ? '比对一致' : '需当面核实' }}
                  </el-tag>
                </div>
                <div class="ct-desc mt-2">{{ ct.description }}</div>
                <div class="ct-advice-row mt-2" v-if="ct.verification_advice">
                  <span class="advice-label font-bold">🩺 接诊核查建议：</span>
                  <span class="advice-text">{{ ct.verification_advice }}</span>
                </div>
              </div>
            </div>
          </div>
        </el-tab-pane>

        <!-- 维度 2: 跨期病历时序演变与时间间隔分析 -->
        <el-tab-pane label="跨期时序演变与发病间隔" name="timeline">
          <div class="tab-pane-box">
            <div class="timeline-evolution-card mb-3">
              <div class="tec-title">
                <el-icon><Timer /></el-icon>
                <span>发病时间间隔规律与动态特征</span>
              </div>
              <div class="tec-desc">
                {{ diagnosticResult.symptom_correlation?.progression_pattern || '就诊档案记录了完整的病程发展时空链条。' }}
              </div>
            </div>

            <!-- 垂直时序演变链 -->
            <div class="encounter-timeline-wrapper">
              <div
                v-for="(t, idx) in diagnosticResult.timeline_correlation"
                :key="t.record_id || idx"
                class="timeline-node-card"
              >
                <div class="tnc-left">
                  <div class="tnc-dot" :class="t.evolutionStatus === '频发加剧' ? 'danger' : 'normal'"></div>
                  <div v-if="idx < diagnosticResult.timeline_correlation.length - 1" class="tnc-line"></div>
                </div>
                <div class="tnc-right">
                  <div class="tnc-header-row">
                    <span class="tnc-date font-mono">{{ t.visit_date }}</span>
                    <span class="tnc-days-ago text-muted">({{ t.days_ago }}天前)</span>
                    <el-tag size="small" :type="getEvolutionTagType(t.evolutionStatus)" effect="plain">
                      {{ t.evolutionStatus || '就诊记录' }}
                    </el-tag>
                    <span class="tnc-interval-chip font-bold">{{ t.interval_from_prev }}</span>
                    <span class="tnc-hosp ml-auto text-muted">{{ t.hospital_name }} · {{ t.department_name }}</span>
                  </div>

                  <div class="tnc-body-grid">
                    <div class="t-cell">
                      <span class="t-label">主诉病状:</span>
                      <span class="t-val">{{ t.chief_complaint || '—' }}</span>
                    </div>
                    <div class="t-cell">
                      <span class="t-label">临床确诊:</span>
                      <span class="t-val font-bold text-primary">{{ t.diagnosis || '—' }}</span>
                    </div>
                    <div class="t-cell full-width" v-if="t.treatment_plan">
                      <span class="t-label">使用处方:</span>
                      <span class="t-val font-mono text-muted">{{ t.treatment_plan }}</span>
                    </div>
                    <div class="t-cell full-width" v-if="t.exam_summary">
                      <span class="t-label">检验/检查:</span>
                      <span class="t-val text-muted">{{ t.exam_summary }}</span>
                    </div>
                  </div>
                </div>
              </div>

              <!-- 当前拟接诊节点 -->
              <div class="timeline-node-card current-encounter">
                <div class="tnc-left">
                  <div class="tnc-dot active"></div>
                </div>
                <div class="tnc-right">
                  <div class="tnc-header-row">
                    <span class="tnc-date font-mono text-purple font-bold">本次就诊 (当前输入)</span>
                    <span class="tnc-interval-chip current font-bold">
                      经治医生正在采集临床表现
                    </span>
                  </div>
                  <div class="tnc-body-grid">
                    <div class="t-cell full-width">
                      <span class="t-label">当前主诉:</span>
                      <span class="t-val font-bold">{{ currentSymptoms || '（就诊主诉录入中）' }}</span>
                    </div>
                    <div class="t-cell full-width" v-if="presentIllness">
                      <span class="t-label">现病史特点:</span>
                      <span class="t-val">{{ presentIllness }}</span>
                    </div>
                  </div>
                </div>
              </div>
            </div>
          </div>
        </el-tab-pane>

        <!-- 维度 3: 症状、既往用药与化验单交叉比对 -->
        <el-tab-pane label="用药史与化验单比对" name="correlation">
          <div class="tab-pane-box">
            <!-- 既往用药与疗效反应 -->
            <div class="detail-section-card mb-3">
              <div class="dsc-header">
                <el-icon color="#0284c7"><FolderChecked /></el-icon>
                <span>过往使用药物、治疗反应与耐药机制分析</span>
              </div>
              <div class="dsc-body">
                <div class="med-history-tags mb-2" v-if="diagnosticResult.medication_correlation?.past_medications?.length">
                  <span class="mht-label">既往病历使用处方药品清单：</span>
                  <el-tag
                    v-for="med in diagnosticResult.medication_correlation.past_medications"
                    :key="med"
                    type="info"
                    size="small"
                    effect="light"
                    class="mr-1 mb-1"
                  >
                    💊 {{ med }}
                  </el-tag>
                </div>
                <div class="dsc-text-block">
                  <strong>用药疗效与病情反复评估：</strong>
                  <p>{{ diagnosticResult.medication_correlation?.drug_response_analysis }}</p>
                </div>
                <div class="dsc-text-block alert-block mt-2">
                  <strong>用药配伍禁忌与安全提示：</strong>
                  <p>{{ diagnosticResult.medication_correlation?.interactions_and_risks }}</p>
                </div>
              </div>
            </div>

            <!-- 历史医技与化验单异常指标对照表 -->
            <div class="detail-section-card">
              <div class="dsc-header">
                <el-icon color="#059669"><Collection /></el-icon>
                <span>历史医技检查与检验化验单异常指标对照</span>
              </div>
              <div class="dsc-body">
                <el-table
                  :data="diagnosticResult.lab_correlation?.abnormal_indicators || []"
                  size="small"
                  border
                  stripe
                  class="lab-table"
                >
                  <el-table-column prop="item_name" label="检查/化验项目" min-width="180" />
                  <el-table-column prop="history_value" label="历史回传测量结果" min-width="220" />
                  <el-table-column prop="status" label="临床状态" width="100">
                    <template #default="{ row }">
                      <el-tag
                        size="small"
                        :type="row.status === 'ABNORMAL' ? 'danger' : (row.status === 'WARNING' ? 'warning' : 'success')"
                      >
                        {{ row.status === 'ABNORMAL' ? '异常' : (row.status === 'WARNING' ? '提示' : '正常') }}
                      </el-tag>
                    </template>
                  </el-table-column>
                  <el-table-column prop="clinical_alert" label="对本次诊断之提示意义" min-width="260" />
                </el-table>
                <div class="lab-summary-text mt-2 text-muted">
                  <el-icon><InfoFilled /></el-icon>
                  <span>{{ diagnosticResult.lab_correlation?.summary }}</span>
                </div>
              </div>
            </div>
          </div>
        </el-tab-pane>

        <!-- 维度 4: 既往处置追溯与接诊核查线索（严格遵循非诊断、非处方决策合规红线） -->
        <el-tab-pane label="既往处置追溯与核查线索" name="treatment">
          <div class="tab-pane-box">
            <div class="treatment-overview-card mb-3">
              <div class="toc-header">
                <span class="toc-title">既往跨院处置记录归纳与接诊核查要点（仅供病历文书整理参考，AI 不下达诊断或处方）</span>
                <el-button
                  type="primary"
                  plain
                  size="small"
                  :icon="Select"
                  @click="adoptTreatment(diagnosticResult.recommended_treatment)"
                >
                  引用既往处置摘要至病史备注
                </el-button>
              </div>
              <div class="toc-text font-mono" style="white-space: pre-wrap;">
                {{ diagnosticResult.recommended_treatment?.treatment_plan_text }}
              </div>
            </div>

            <!-- 既往用药记录摘录表 -->
            <div class="rx-recommend-table-box mb-3">
              <div class="rtt-header">
                <span>💊 既往已授权病历记载药物摘录（仅供防重复用药与禁忌核对，非 AI 处方建议）：</span>
              </div>
              <el-table
                :data="diagnosticResult.recommended_treatment?.prescriptions || []"
                size="small"
                border
                class="rx-table"
              >
                <el-table-column prop="category" label="既往用药分类" width="140" />
                <el-table-column prop="drug_name" label="历史病历记载药品及规格" min-width="220" />
                <el-table-column prop="usage" label="历史记录用法与核对提示" min-width="220" />
              </el-table>
            </div>

            <!-- 建议医技检查开单 -->
            <div v-if="diagnosticResult.recommended_treatment?.suggested_orders?.length" class="suggested-orders-box">
              <span class="sob-label">🔬 建议补充开立辅助检查：</span>
              <div class="sob-chips">
                <el-tag
                  v-for="order in diagnosticResult.recommended_treatment.suggested_orders"
                  :key="order"
                  type="primary"
                  effect="light"
                  class="mr-2"
                >
                  {{ order }}
                </el-tag>
              </div>
            </div>
          </div>
        </el-tab-pane>

        <!-- 维度 5: 提示词审查与外部大模型 API 配置指南 -->
        <el-tab-pane label="AI提示词与API配置" name="prompt">
          <div class="tab-pane-box">
            <el-alert
              type="info"
              show-icon
              :closable="false"
              class="mb-3"
              title="大模型接口集成说明：当前可无缝调用本地内置临床知识推理引擎，亦支持填入您的 OpenAI / DeepSeek / Qwen 等大模型 API Key 随时接入真实通用大模型。"
            />

            <div class="api-guide-card mb-3">
              <div class="agc-title">⚙️ 如何配置您的外部大模型 API：</div>
              <div class="agc-body">
                <p>您只需在系统配置文件 <code>backend/config/config.yaml</code> 或环境变量中设置如下参数，后端即可自动切换至外部大模型：</p>
                <pre class="code-block font-mono">
# 配置文件: backend/config/config.yaml
ai:
  enabled: true
  api_url: "https://api.deepseek.com/v1"  # 外部大模型 API 端点
  api_key: "sk-your-real-api-key-here"    # 填入您的 API Key
  model: "deepseek-chat"                  # 模型名称 (如 deepseek-chat, gpt-4o, qwen-turbo)
  temperature: 0.2
                </pre>
              </div>
            </div>

            <div class="prompt-preview-card">
              <div class="ppc-header">
                <span>🔍 发送给大模型的完整结构化临床推理 Prompt：</span>
                <el-button size="small" link type="primary" :icon="CopyDocument" @click="copyPrompt">
                  复制完整提示词
                </el-button>
              </div>
              <pre class="prompt-text-pre font-mono">{{ diagnosticResult.raw_prompt }}</pre>
            </div>
          </div>
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
          AI 辅助诊断仅供经治医师参考，最终诊断与用药处方权依法归属执业医师 (Doctor-in-the-Loop)
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
            重新推理分析
          </el-button>
          <el-button
            type="success"
            :icon="DocumentCopy"
            :disabled="!diagnosticResult?.soap_summary"
            @click="adoptAllToRecord"
          >
            一键采纳 SOAP 与临床线索至病历草稿
          </el-button>
        </div>
      </div>
    </template>
  </el-dialog>
</template>

<script setup lang="ts">
import { ref, watch, computed } from 'vue'
import {
  MagicStick, CircleCheckFilled, Document, Cpu, Loading,
  Timer, FolderChecked, Collection, InfoFilled,
  Select, CopyDocument, WarningFilled, Refresh, DocumentCopy
} from '@element-plus/icons-vue'
import { ElMessage } from 'element-plus'
import { diagnosticAssistApi } from '../api'

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
      ElMessage.warning(res.message || 'AI 辅助诊断返回格式异常')
    }
  } catch (err: any) {
    clearInterval(stepTimer)
    console.error('[AI Diagnostic Assist Error]:', err)
    ElMessage.error(err?.response?.data?.message || err?.message || '调用 AI 辅助诊断服务失败')
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

function getEvolutionTagType(status?: string) {
  if (status === '频发加剧' || status === '急性恶化') return 'danger'
  if (status === '间歇复发') return 'warning'
  if (status === '平稳代偿') return 'success'
  return 'info'
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
  margin-bottom: 14px;
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

.tab-pane-box {
  background: #ffffff;
  border-radius: 10px;
  padding: 16px;
  border: 1px solid #e2e8f0;
}

.pane-tip-row {
  font-size: 13px;
  color: #475569;
  margin-bottom: 14px;
}

/* 疾病预测卡片 */
.predicted-diseases-grid {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.disease-card {
  border-radius: 10px;
  padding: 14px 16px;
  border: 1px solid #e2e8f0;
  background: #ffffff;
  transition: transform 0.2s ease, box-shadow 0.2s ease;
}

.disease-card:hover {
  transform: translateY(-2px);
  box-shadow: 0 4px 14px rgba(0, 0, 0, 0.08);
}

.disease-card.risk-critical {
  border-left: 4px solid #ef4444;
  background: #fffdfd;
}

.disease-card.risk-high {
  border-left: 4px solid #f59e0b;
}

.disease-card.risk-medium {
  border-left: 4px solid #3b82f6;
}

.disease-card.risk-low {
  border-left: 4px solid #10b981;
}

.dc-header {
  display: flex;
  align-items: center;
  gap: 10px;
}

.dc-rank-badge {
  background: #ede9fe;
  color: #7c3aed;
  font-size: 11px;
  font-weight: 800;
  padding: 2px 8px;
  border-radius: 6px;
}

.disease-name {
  font-size: 15px;
  font-weight: 800;
  color: #0f172a;
}

.icd-code {
  font-size: 12px;
  color: #64748b;
  background: #f1f5f9;
  padding: 2px 6px;
  border-radius: 4px;
  margin-left: 6px;
}

.dc-prob-box {
  margin-left: auto;
  display: flex;
  align-items: center;
  gap: 8px;
}

.prob-num {
  font-size: 18px;
  font-weight: 800;
  color: #7c3aed;
}

.dc-progress-bar {
  margin: 10px 0 12px 0;
}

.dc-content {
  display: flex;
  flex-direction: column;
  gap: 6px;
  font-size: 13px;
  line-height: 1.5;
}

.content-row {
  display: flex;
  gap: 8px;
}

.row-label {
  font-weight: 700;
  color: #475569;
  flex-shrink: 0;
  width: 90px;
}

.row-val {
  color: #1e293b;
}

.exam-chips-box {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}

.exam-chip {
  background: #f1f5f9;
  border: 1px solid #cbd5e1;
  padding: 2px 8px;
  border-radius: 6px;
  font-size: 11px;
  color: #334155;
}

.dc-footer-action {
  margin-top: 10px;
  padding-top: 10px;
  border-top: 1px dashed #e2e8f0;
  display: flex;
  justify-content: flex-end;
}

/* 时间轴部分 */
.timeline-evolution-card {
  background: #f8fafc;
  border: 1px solid #e2e8f0;
  border-radius: 8px;
  padding: 10px 14px;
}

.tec-title {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 13px;
  font-weight: 700;
  color: #0284c7;
  margin-bottom: 4px;
}

.tec-desc {
  font-size: 12px;
  color: #334155;
  line-height: 1.5;
}

.encounter-timeline-wrapper {
  display: flex;
  flex-direction: column;
}

.timeline-node-card {
  display: flex;
  gap: 14px;
}

.tnc-left {
  display: flex;
  flex-direction: column;
  align-items: center;
  width: 20px;
}

.tnc-dot {
  width: 14px;
  height: 14px;
  border-radius: 50%;
  background: #3b82f6;
  border: 2px solid #ffffff;
  box-shadow: 0 0 0 2px #93c5fd;
  flex-shrink: 0;
  margin-top: 4px;
}

.tnc-dot.danger {
  background: #ef4444;
  box-shadow: 0 0 0 2px #fca5a5;
}

.tnc-dot.active {
  background: #8b5cf6;
  box-shadow: 0 0 0 3px #ddd6fe;
}

.tnc-line {
  width: 2px;
  flex: 1;
  background: #cbd5e1;
  margin: 4px 0;
}

.tnc-right {
  flex: 1;
  background: #ffffff;
  border: 1px solid #e2e8f0;
  border-radius: 8px;
  padding: 10px 14px;
  margin-bottom: 12px;
}

.timeline-node-card.current-encounter .tnc-right {
  border-color: #8b5cf6;
  background: #faf5ff;
}

.tnc-header-row {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 8px;
  flex-wrap: wrap;
}

.tnc-interval-chip {
  font-size: 11px;
  background: #fef3c7;
  color: #b45309;
  padding: 2px 6px;
  border-radius: 4px;
}

.tnc-interval-chip.current {
  background: #ede9fe;
  color: #7c3aed;
}

.tnc-body-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 6px 14px;
  font-size: 12px;
}

.t-cell.full-width {
  grid-column: 1 / -1;
}

.t-label {
  color: #64748b;
  font-weight: 600;
  margin-right: 6px;
}

/* 用药与化验卡片 */
.detail-section-card {
  border: 1px solid #e2e8f0;
  border-radius: 8px;
  padding: 12px 14px;
  background: #f8fafc;
}

.dsc-header {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 13px;
  font-weight: 700;
  color: #1e293b;
  margin-bottom: 10px;
}

.dsc-text-block {
  font-size: 12px;
  line-height: 1.6;
  color: #334155;
}

.dsc-text-block.alert-block {
  background: #fff7ed;
  border-left: 3px solid #f97316;
  padding: 6px 10px;
  border-radius: 4px;
}

/* 处置建议 */
.treatment-overview-card {
  background: #f0fdf4;
  border: 1px solid #bbf7d0;
  border-radius: 8px;
  padding: 12px 16px;
}

.toc-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 8px;
}

.toc-title {
  font-size: 14px;
  font-weight: 700;
  color: #166534;
}

.toc-text {
  font-size: 13px;
  line-height: 1.6;
  color: #14532d;
}

/* 提示词卡片 */
.prompt-text-pre {
  background: #0f172a;
  color: #f8fafc;
  padding: 14px;
  border-radius: 8px;
  font-size: 11px;
  line-height: 1.5;
  max-height: 280px;
  overflow-y: auto;
  white-space: pre-wrap;
}

.code-block {
  background: #1e293b;
  color: #38bdf8;
  padding: 10px;
  border-radius: 6px;
  font-size: 11px;
  line-height: 1.4;
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

/* ── SOAP 结构化网格与线索样式 ── */
.soap-grid-layout {
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  gap: 16px;
}

.soap-box {
  background: #ffffff;
  border-radius: 10px;
  border: 1px solid #e2e8f0;
  padding: 16px;
  box-shadow: 0 1px 4px rgba(0, 0, 0, 0.03);
}

.soap-box.s-box {
  border-left: 4px solid #6366f1;
}

.soap-box.o-box {
  border-left: 4px solid #0284c7;
}

.soap-box.a-box {
  border-left: 4px solid #059669;
}

.soap-box.p-box {
  border-left: 4px solid #7c3aed;
}

.soap-box-header {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 10px;
}

.soap-letter {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 26px;
  height: 26px;
  border-radius: 6px;
  background: #f1f5f9;
  font-weight: 800;
  font-size: 14px;
  color: #1e293b;
}

.s-box .soap-letter { background: #ede9fe; color: #6366f1; }
.o-box .soap-letter { background: #e0f2fe; color: #0284c7; }
.a-box .soap-letter { background: #d1fae5; color: #059669; }
.p-box .soap-letter { background: #f3e8ff; color: #7c3aed; }

.soap-title {
  font-weight: 700;
  font-size: 14px;
  color: #0f172a;
}

.soap-box-body {
  font-size: 13px;
  color: #334155;
  line-height: 1.6;
  white-space: pre-wrap;
}

.soap-box-tip {
  font-size: 11px;
  margin-top: 8px;
}

/* 隐匿线索网格 */
.clues-grid {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.clue-item-card {
  background: #ffffff;
  border-radius: 8px;
  padding: 14px;
  border: 1px solid #e2e8f0;
}

.clue-item-card.clue-danger {
  border-left: 4px solid #ef4444;
  background: #fffafa;
}

.clue-item-card.clue-warning {
  border-left: 4px solid #f59e0b;
  background: #fffdfa;
}

.clue-item-card.clue-success {
  border-left: 4px solid #10b981;
}

.clue-top-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.clue-badge {
  font-weight: 700;
  font-size: 13px;
  color: #1e293b;
}

.clue-text {
  font-size: 13px;
  color: #475569;
  margin-top: 6px;
  line-height: 1.5;
}

/* 语义矛盾网格 */
.contradictions-grid {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.ct-card {
  background: #ffffff;
  border-radius: 8px;
  padding: 14px;
  border: 1px solid #e2e8f0;
}

.ct-card.ct-warning {
  border-left: 4px solid #f59e0b;
  background: #fffdfa;
}

.ct-card.ct-success {
  border-left: 4px solid #10b981;
}

.ct-top-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.ct-tag {
  font-size: 14px;
  color: #1e293b;
}

.ct-desc {
  font-size: 13px;
  color: #334155;
  line-height: 1.5;
}

.ct-advice-row {
  background: #f8fafc;
  border-radius: 6px;
  padding: 8px 12px;
  font-size: 12px;
}

.advice-label {
  color: #0369a1;
}

.advice-text {
  color: #0f172a;
}
</style>
