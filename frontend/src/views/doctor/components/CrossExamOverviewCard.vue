<template>
  <!-- 该患者做过的所有检查单与医学影像概览专区 (WP-16: 从 CrossQueryView 迁入, 支持跨机构调阅与统一安全评估) -->
  <el-card v-if="patientExamRecords.length > 0 && searchKeyword.trim()" shadow="hover" class="box-card mb-4 patient-exams-card">
    <template #header>
      <div class="pec-header">
        <div class="pec-title">
          <strong>
            <template v-if="patientName">
              【{{ patientName }}】关联全部外院/全网检查单与影像报告 (共 {{ displayedExamRecords.length }} 项)
            </template>
            <template v-else>
              检索关键词【{{ searchKeyword }}】匹配的外院/全网检查单与报告 (共 {{ displayedExamRecords.length }} 项)
            </template>
          </strong>
          <el-tag size="small" type="success" effect="dark" class="ml-2">跨机构可信存证</el-tag>
        </div>

        <!-- 快捷视图控制与异常筛选工具栏 -->
        <div class="pec-toolbar">
          <el-switch
            v-model="filterOnlyAbnormal"
            inline-prompt
            active-text="仅看阳性/异常"
            inactive-text="全部报告"
            size="small"
            class="mr-2"
          />
          <el-button
            size="small"
            :type="expandAllFindings ? 'primary' : 'default'"
            plain
            @click.stop="toggleAllFindings"
          >
            {{ expandAllFindings ? '收起全部测量明细' : '展开全部测量明细' }}
          </el-button>
        </div>
      </div>
    </template>

    <div class="pec-grid">
      <div
        v-for="exam in displayedExamRecords"
        :key="'exam-' + exam.id"
        class="pec-item-card"
        :class="{
          'has-access': exam.has_access || exam.doctor_id === userId || exam.hospital_id === userHospitalId,
          'is-abnormal-card': getParsedExam(exam).isAnyAbnormal
        }"
        @click="$emit('access', exam)"
      >
        <!-- 1. 卡片顶部状态区：类别、账本一致性、异常总览标签 -->
        <div class="pec-item-top">
          <div class="top-tags-left">
            <el-tag size="small" :type="exam.data_type === 'IMAGE' ? 'warning' : (exam.data_type === 'REPORT' ? 'success' : 'primary')" effect="dark">
              {{ exam.data_type === 'IMAGE' ? '影像切片' : (exam.data_type === 'REPORT' ? '检验化验' : '临床检查') }}
            </el-tag>
            <el-tag v-if="exam.is_tampered" size="small" type="danger" effect="dark">篡改告警</el-tag>
            <el-tag v-else size="small" type="success" effect="light">账本一致</el-tag>
          </div>

          <!-- 右侧：报告级异常/阳性快速指征徽章 -->
          <div class="top-tags-right">
            <template v-for="tag in getParsedExam(exam).summaryTags.slice(0, 2)" :key="tag.text">
              <el-tag size="small" :type="tag.type" effect="plain" class="clinical-summary-tag">
                {{ tag.text }}
              </el-tag>
            </template>
          </div>
        </div>

        <!-- 2. 卡片项目标题与就诊患者 -->
        <div class="pec-item-name">
          <span v-if="exam.patient_name" class="font-bold text-indigo-700 mr-1">[{{ exam.patient_name }}]</span>
          <span v-if="exam.exam_items && !exam.exam_items.includes('未取得患者授权')" v-html="highlightText(exam.exam_items, searchKeyword)"></span>
          <span v-else-if="exam.symptoms && !exam.symptoms.includes('未取得患者授权')" v-html="highlightText(exam.symptoms, searchKeyword)"></span>
          <span v-else-if="exam.diagnosis && !exam.diagnosis.includes('未授权')" v-html="highlightText(exam.diagnosis, searchKeyword)"></span>
          <span v-else>
            {{ exam.data_type === 'IMAGE' ? '跨院医学影像数据' : '跨院临床检查单据' }} ({{ exam.record_no }})
          </span>
        </div>

        <!-- 3. 医疗机构与开具医生元信息 -->
        <div class="pec-item-meta-bar">
          <span><strong>{{ exam.hospital_name }}</strong> · {{ exam.department_name || '临床/医技科室' }}</span>
          <span>{{ formatTime(exam.created_at) }} · {{ exam.doctor_name }} 医生</span>
        </div>

        <!-- 4. 核心内容区：优先呈现核心诊断结论，分层收起参数明细 -->
        <div class="pec-item-body">
          <template v-if="exam.has_access || exam.doctor_id === userId || exam.hospital_id === userHospitalId">
            <!-- 情况 A：存在医技回传报告与结论 (exam_result) -->
            <div v-if="exam.exam_result" class="structured-exam-container">
              <!-- 结构化子项列表（例如将心电图与胸部CT完全拆分呈现） -->
              <div
                v-for="(sec, sIdx) in getParsedExam(exam).sections"
                :key="sec.id"
                class="exam-sub-section"
                :class="{
                  'section-matched': sec.hasKeywordMatch,
                  'section-abnormal': sec.isAbnormal
                }"
              >
                <!-- 子项标头 -->
                <div class="sub-sec-head">
                  <div class="sub-sec-title">
                    <strong v-html="highlightText(sec.title, searchKeyword)"></strong>
                    <span v-if="sec.subtitle" class="sub-sec-sub" v-html="highlightText(sec.subtitle, searchKeyword)"></span>
                  </div>
                  <div class="sub-sec-tags">
                    <span v-if="sec.hasKeywordMatch && searchKeyword.trim()" class="match-badge">命中搜索</span>
                    <template v-for="t in sec.tags" :key="t.text">
                      <span class="sub-tag-pill" :class="t.type">{{ t.text }}</span>
                    </template>
                  </div>
                </div>

                <!-- 优先大字高亮显示：核心诊断与结论 (Conclusion) -->
                <div class="sub-sec-conclusion" :class="{ 'is-warning-bg': sec.isAbnormal }">
                  <div class="conclusion-badge">
                    <span class="pulse-indicator" v-if="sec.isAbnormal"></span>
                    <span>核心结论</span>
                  </div>
                  <div class="conclusion-text" v-html="highlightText(sec.conclusion, searchKeyword)"></div>
                </div>

                <!-- 处置/随访建议（若有） -->
                <div v-if="sec.suggestion" class="sub-sec-suggestion">
                  <span class="sug-icon">建议：</span>
                  <span class="sug-text" v-html="highlightText(sec.suggestion, searchKeyword)"></span>
                </div>

                <!-- 过程测量与详细所见（默认折叠，按需展开） -->
                <div v-if="sec.findings && sec.findings.length" class="sub-sec-findings-container">
                  <div
                    class="findings-toggle-btn"
                    @click.stop="toggleFindingExpand(`${exam.id}-${sIdx}`)"
                  >
                    <span class="toggle-icon">{{ isFindingExpanded(`${exam.id}-${sIdx}`) ? '▾' : '▸' }}</span>
                    <span>{{ isFindingExpanded(`${exam.id}-${sIdx}`) ? '收起测量所见与检查参数' : `查看测量参数与检查所见 (${sec.findings.length}项)` }}</span>
                  </div>

                  <div v-if="isFindingExpanded(`${exam.id}-${sIdx}`)" class="findings-list-panel">
                    <div
                      v-for="(finding, fIdx) in sec.findings"
                      :key="fIdx"
                      class="finding-row"
                      v-html="highlightText(finding, searchKeyword)"
                    ></div>
                    <div v-if="sec.issuer" class="issuer-row">
                      <span>报告医师：{{ sec.issuer }}</span>
                    </div>
                  </div>
                </div>
              </div>
            </div>

            <!-- 情况 B：无医技报告，直接展示门诊临床诊断与主诉 -->
            <div v-else-if="exam.diagnosis || exam.symptoms" class="pure-diagnosis-box">
              <div class="pure-diag-head">
                <strong>临床确诊 / 主诉：</strong>
              </div>
              <div class="pure-diag-text font-bold" v-html="highlightText(exam.diagnosis || exam.symptoms || '', searchKeyword)"></div>
              <div v-if="exam.treatment_plan" class="pure-diag-plan">
                <span>处置方案：{{ exam.treatment_plan }}</span>
              </div>
            </div>
          </template>

          <!-- 情况 C：跨机构未授权脱敏遮罩保护 -->
          <div v-else class="masked-lock-box">
            <div class="lock-head">
              <span><strong>跨院临床数据已实施零信任访问保护</strong></span>
            </div>
            <div class="lock-desc">
              就诊记录包含 <strong>{{ exam.exam_items || '专科检查项目' }}</strong>。请点击卡片发起知情授权申请，或由患者输入现场密钥即时解密调阅。
            </div>
          </div>
        </div>

        <!-- 5. 卡片底部：放行权限状态与调阅按钮 -->
        <div class="pec-item-bottom">
          <span v-if="exam.doctor_id === userId || exam.hospital_id === userHospitalId" class="access-tag green">
            本院/本人放行
          </span>
          <span v-else-if="exam.access_type === 'ALL_DOCTORS'" class="access-tag green">
            全体医生可见
          </span>
          <span v-else-if="exam.access_type === 'BREAK_GLASS'" class="access-tag red">
            24h破窗放行中
          </span>
          <span v-else-if="exam.has_access" class="access-tag green">
            患者已授权
          </span>
          <span v-else class="access-tag orange">
            需安全网关核准
          </span>
          <el-button type="primary" size="small" link class="view-btn">安全调阅</el-button>
        </div>
      </div>
    </div>
  </el-card>
</template>

<script setup lang="ts">
// WP-16: 检查单与医学影像概览专区 (含阳性/异常筛选与测量明细展开状态), 从 CrossQueryView 迁入
import { ref, computed } from 'vue'
import { formatTime } from '@/utils/recordFormatters'
import { highlightText, type ParsedMedicalReport } from '@/utils/medicalReportParser'
import type { MedicalRecordRow } from '@/types/clinical'

const props = defineProps<{
  results: MedicalRecordRow[]
  searchKeyword: string
  patientName: string | undefined
  userId: number | undefined
  userHospitalId: number | undefined
  getParsedExam: (exam: MedicalRecordRow) => ParsedMedicalReport
}>()

defineEmits<{
  (e: 'access', row: MedicalRecordRow): void
}>()

const patientExamRecords = computed(() => {
  return props.results.filter((r: MedicalRecordRow) =>
    r.data_type === 'REPORT' ||
    r.data_type === 'IMAGE' ||
    r.need_exam ||
    (r.exam_items && r.exam_items.trim() !== '') ||
    (r.exam_result && r.exam_result.trim() !== '')
  )
})

// 检查报告结构化展开、过滤与状态管理
const filterOnlyAbnormal = ref(false)
const expandAllFindings = ref(false)
const expandedFindings = ref<Record<string, boolean>>({})

const displayedExamRecords = computed(() => {
  let list = patientExamRecords.value
  if (filterOnlyAbnormal.value) {
    list = list.filter((exam: MedicalRecordRow) => {
      const p = props.getParsedExam(exam)
      return p.isAnyAbnormal
    })
  }
  return list
})

function isFindingExpanded(key: string): boolean {
  if (expandAllFindings.value) return true
  return !!expandedFindings.value[key]
}

function toggleFindingExpand(key: string) {
  expandedFindings.value[key] = !isFindingExpanded(key)
}

function toggleAllFindings() {
  expandAllFindings.value = !expandAllFindings.value
  expandedFindings.value = {}
}
</script>

<style scoped src="../CrossQueryView.css"></style>