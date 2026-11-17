<script setup lang="ts">
import { computed, ref, type CSSProperties } from 'vue'
import {
  Clock,
  Document,
  Key,
  Lock,
  OfficeBuilding,
  Plus,
  View,
  WarningFilled
} from '@element-plus/icons-vue'
import type { MedicalRecordRow } from '@/types/clinical'
import ChainVerifyBadge from '@/components/common/ChainVerifyBadge.vue'
import HashCapsule from '@/components/common/HashCapsule.vue'
import {
  buildHistoryClueSnippet,
  cleanClinicalText,
  cleanDurationText,
  computeVisitGapLabel,
  filterTimelineRecords,
  formatEncounterType,
  formatOnsetTime,
  formatRelativeVisitTime,
  formatTime,
  isRecordAccessible,
  resolveChainVerifyStatus,
  sortRecordsChronologically,
  type TimelineFilterMode
} from '@/utils/timelineFormatters'

interface Props {
  records: MedicalRecordRow[]
  loading?: boolean
  compact?: boolean
  maxHeight?: string
  allowAdopt?: boolean
  allowRequestAccess?: boolean
  hospitalMap?: Record<number, string>
  departmentMap?: Record<number, string>
  currentHospitalId?: number
  emptyDescription?: string
}

const props = withDefaults(defineProps<Props>(), {
  loading: false,
  compact: false,
  maxHeight: 'none',
  allowAdopt: false,
  allowRequestAccess: false,
  hospitalMap: () => ({}),
  departmentMap: () => ({}),
  currentHospitalId: undefined,
  emptyDescription: '该患者暂无可展示的历史就诊时间线记录'
})

const emit = defineEmits<{
  (e: 'view-detail', record: MedicalRecordRow): void
  (e: 'request-consent', record: MedicalRecordRow): void
  (e: 'adopt-history', snippet: string): void
}>()

const filterMode = ref<TimelineFilterMode>('ALL')
const expandedIds = ref<Record<number, boolean>>({})

const sortedRecords = computed<MedicalRecordRow[]>(() => sortRecordsChronologically(props.records))

const accessibleCount = computed<number>(
  () => sortedRecords.value.filter((r) => isRecordAccessible(r)).length
)

const lockedCount = computed<number>(
  () => sortedRecords.value.filter((r) => !isRecordAccessible(r)).length
)

const tamperedCount = computed<number>(
  () => sortedRecords.value.filter((r) => Boolean(r.is_tampered)).length
)

const visibleRecords = computed<MedicalRecordRow[]>(() =>
  filterTimelineRecords(sortedRecords.value, filterMode.value)
)

const scrollContainerStyle = computed<CSSProperties>(() => {
  if (!props.maxHeight || props.maxHeight === 'none') return {}
  return { maxHeight: props.maxHeight, overflowY: 'auto' }
})

function resolveHospitalName(rec: MedicalRecordRow): string {
  if (rec.hospital_name) return rec.hospital_name
  if (rec.hospital_id && props.hospitalMap[rec.hospital_id]) {
    return props.hospitalMap[rec.hospital_id]
  }
  return rec.hospital_id ? `合作医疗机构 #${rec.hospital_id}` : '本院接诊机构'
}

function resolveDepartmentName(rec: MedicalRecordRow): string {
  if (rec.department_name) return rec.department_name
  if (rec.department_id && props.departmentMap[rec.department_id]) {
    return props.departmentMap[rec.department_id]
  }
  return '临床接诊科室'
}

function isCrossHospital(rec: MedicalRecordRow): boolean {
  if (!props.currentHospitalId || !rec.hospital_id) return false
  return rec.hospital_id !== props.currentHospitalId
}

function getNodeStatusClass(rec: MedicalRecordRow): string {
  if (rec.is_tampered) return 'is-tampered'
  if (!isRecordAccessible(rec)) return 'is-locked'
  if (rec.verified) return 'is-verified'
  return 'is-pending'
}

function getGapFromNext(index: number): string {
  const list = visibleRecords.value
  if (index >= list.length - 1) return ''
  return computeVisitGapLabel(list[index].created_at, list[index + 1].created_at)
}

function isExpanded(id: number): boolean {
  return Boolean(expandedIds.value[id])
}

function toggleExpand(id: number): void {
  expandedIds.value = {
    ...expandedIds.value,
    [id]: !expandedIds.value[id]
  }
}

function handleAdopt(rec: MedicalRecordRow): void {
  const snippet = buildHistoryClueSnippet(rec, resolveHospitalName(rec))
  if (snippet) emit('adopt-history', snippet)
}

function formatPhenomenon(rec: MedicalRecordRow): string {
  return cleanClinicalText(rec.chief_complaint || rec.symptoms) || '未登记明确主诉症状'
}

function formatDiagnosis(rec: MedicalRecordRow): string {
  return cleanClinicalText(rec.diagnosis || rec.initial_diagnosis) || '待确诊 / 随访评估中'
}

function formatEtiology(rec: MedicalRecordRow): string {
  return cleanClinicalText(rec.etiology || rec.diagnostic_basis) || '暂未记录明确病因及病理依据'
}

function formatExamAndTreatment(rec: MedicalRecordRow): string {
  const segments: string[] = []
  if (rec.exam_items) segments.push(`检查：${rec.exam_items}`)
  if (rec.exam_result) segments.push(`结果：${cleanClinicalText(rec.exam_result)}`)
  if (rec.treatment_plan) segments.push(`处置：${cleanClinicalText(rec.treatment_plan)}`)
  return segments.join(' ｜ ')
}
</script>

<template>
  <div class="pct-root" :class="{ 'is-compact': compact }" v-loading="loading">
    <!-- 顶部统计与视图筛选条 -->
    <div v-if="sortedRecords.length > 0" class="pct-toolbar">
      <div class="pct-stats">
        <span class="pct-stat-chip">
          共 <strong>{{ sortedRecords.length }}</strong> 次就诊
        </span>
        <span class="pct-stat-chip is-success">
          已授权 <strong>{{ accessibleCount }}</strong>
        </span>
        <span v-if="lockedCount > 0" class="pct-stat-chip is-warning">
          受控待授权 <strong>{{ lockedCount }}</strong>
        </span>
        <span v-if="tamperedCount > 0" class="pct-stat-chip is-danger">
          核验异常 <strong>{{ tamperedCount }}</strong>
        </span>
      </div>

      <div class="pct-filters" role="group" aria-label="时间线授权筛选">
        <button
          type="button"
          class="pct-filter-btn"
          :class="{ 'is-active': filterMode === 'ALL' }"
          @click="filterMode = 'ALL'"
        >
          全部
        </button>
        <button
          type="button"
          class="pct-filter-btn"
          :class="{ 'is-active': filterMode === 'ACCESSIBLE' }"
          @click="filterMode = 'ACCESSIBLE'"
        >
          可阅明文
        </button>
        <button
          v-if="lockedCount > 0"
          type="button"
          class="pct-filter-btn"
          :class="{ 'is-active': filterMode === 'LOCKED' }"
          @click="filterMode = 'LOCKED'"
        >
          受控锁定
        </button>
      </div>
    </div>

    <!-- 空状态 -->
    <div v-if="!loading && visibleRecords.length === 0" class="pct-empty">
      <el-icon :size="26" class="pct-empty-icon"><Document /></el-icon>
      <p class="pct-empty-title">
        {{ sortedRecords.length > 0 ? '当前筛选条件下暂无病例节点' : emptyDescription }}
      </p>
      <p class="pct-empty-sub">
        {{ sortedRecords.length > 0 ? '请切换上方授权筛选查看其他历史节点' : '若患者存在跨机构就诊记录，可在右侧跨机构查询发起调阅' }}
      </p>
    </div>

    <!-- 纵向病例时间线列表 -->
    <div v-else class="pct-scroll-area" :style="scrollContainerStyle">
      <div class="pct-timeline">
        <div
          v-for="(rec, idx) in visibleRecords"
          :key="rec.id"
          class="pct-node"
          :class="getNodeStatusClass(rec)"
        >
          <!-- 左侧时间轴脊柱与节点圆点 -->
          <div class="pct-rail">
            <span class="pct-dot" />
            <span v-if="idx < visibleRecords.length - 1" class="pct-line" />
          </div>

          <!-- 右侧病历内容主卡片 -->
          <div class="pct-card">
            <!-- 头部：时间 / 机构 / 链上校验徽章 -->
            <div class="pct-card-head">
              <div class="pct-head-left">
                <div class="pct-date-line">
                  <el-icon class="pct-meta-icon"><Clock /></el-icon>
                  <span class="pct-date-text">{{ formatTime(rec.created_at) }}</span>
                  <span class="pct-relative-tag">{{ formatRelativeVisitTime(rec.created_at) }}</span>
                  <el-tag size="small" effect="plain" type="info" class="pct-type-tag">
                    {{ formatEncounterType(rec.encounter_type) }}
                  </el-tag>
                  <el-tag
                    v-if="rec.status === 'IN_PROGRESS'"
                    size="small"
                    effect="light"
                    type="warning"
                  >
                    接诊未完结
                  </el-tag>
                </div>

                <div class="pct-org-line">
                  <el-icon class="pct-meta-icon"><OfficeBuilding /></el-icon>
                  <span class="pct-org-name">{{ resolveHospitalName(rec) }}</span>
                  <span class="pct-sep">·</span>
                  <span>{{ resolveDepartmentName(rec) }}</span>
                  <template v-if="rec.doctor_name">
                    <span class="pct-sep">·</span>
                    <span>主诊 {{ rec.doctor_name }}</span>
                  </template>
                  <span v-if="isCrossHospital(rec)" class="pct-cross-pill">跨院协同</span>
                </div>
              </div>

              <div class="pct-head-right">
                <ChainVerifyBadge
                  :status="resolveChainVerifyStatus(rec)"
                  size="small"
                />
              </div>
            </div>

            <!-- 防篡改安全阻断告警 -->
            <div v-if="rec.is_tampered" class="pct-tamper-banner">
              <el-icon class="pct-banner-icon"><WarningFilled /></el-icon>
              <div>
                <div class="pct-banner-title">链上存证指纹校验不一致（零信任阻断）</div>
                <div class="pct-banner-desc">
                  {{ rec.tamper_reason || '本地病历摘要与 Fabric 区块存证或 IPFS 内容哈希存在冲突，该节点仅作审计阻断提示，禁止直接引用。' }}
                </div>
              </div>
            </div>

            <!-- 未授权跨院受控锁定视图（绝不泄露任何明文） -->
            <div v-if="!isRecordAccessible(rec)" class="pct-locked-box">
              <div class="pct-locked-info">
                <el-icon class="pct-locked-icon"><Lock /></el-icon>
                <div>
                  <div class="pct-locked-title">跨院隐私受控节点 · 未取得本次就诊授权</div>
                  <div class="pct-locked-desc">
                    依据零信任策略，诊断结论、主诉症状、病因及用药信息已做服务端脱敏掩码保护。
                  </div>
                </div>
              </div>

              <div v-if="allowRequestAccess" class="pct-locked-actions">
                <el-button
                  size="small"
                  type="primary"
                  plain
                  :icon="Key"
                  @click="emit('request-consent', rec)"
                >
                  申请调阅授权
                </el-button>
              </div>
            </div>

            <!-- 已授权明文临床结构化要素 -->
            <div v-else class="pct-clinical-body">
              <!-- 传染病防护风险横幅 -->
              <div v-if="rec.infection_alert?.has_risk" class="pct-infection-strip">
                <el-icon><WarningFilled /></el-icon>
                <span>职业暴露防护提示：{{ rec.infection_alert.summary }}</span>
              </div>

              <!-- 发病时序与基础体征条 -->
              <div
                v-if="rec.onset_time || rec.duration || rec.vital_signs"
                class="pct-vital-row"
              >
                <span v-if="rec.onset_time" class="pct-mono-chip">
                  发病：{{ formatOnsetTime(rec.onset_time) }}
                </span>
                <span v-if="rec.duration" class="pct-mono-chip">
                  病程：{{ cleanDurationText(rec.duration) }}
                </span>
                <span v-if="rec.vital_signs" class="pct-mono-chip is-vital">
                  体征：{{ rec.vital_signs }}
                </span>
              </div>

              <!-- 四维核心诊疗信息表：现象 / 病因 / 诊断 / 处置 -->
              <div class="pct-clinical-grid">
                <div class="pct-clinical-row">
                  <span class="pct-field-label">临床现象</span>
                  <div class="pct-field-val" :class="{ 'is-clamped': compact && !isExpanded(rec.id) }">
                    {{ formatPhenomenon(rec) }}
                    <span
                      v-if="rec.present_illness && rec.present_illness !== rec.chief_complaint"
                      class="pct-sub-note"
                    >
                      （现病史：{{ cleanClinicalText(rec.present_illness) }}）
                    </span>
                  </div>
                </div>

                <div class="pct-clinical-row">
                  <span class="pct-field-label">病因诱因</span>
                  <div class="pct-field-val" :class="{ 'is-clamped': compact && !isExpanded(rec.id) }">
                    {{ formatEtiology(rec) }}
                  </div>
                </div>

                <div class="pct-clinical-row">
                  <span class="pct-field-label">临床诊断</span>
                  <div class="pct-field-val is-diagnosis">
                    {{ formatDiagnosis(rec) }}
                  </div>
                </div>

                <div v-if="formatExamAndTreatment(rec)" class="pct-clinical-row">
                  <span class="pct-field-label">检查处置</span>
                  <div class="pct-field-val" :class="{ 'is-clamped': compact && !isExpanded(rec.id) }">
                    {{ formatExamAndTreatment(rec) }}
                  </div>
                </div>
              </div>

              <div v-if="compact" class="pct-expand-row">
                <button
                  type="button"
                  class="pct-link-btn"
                  @click="toggleExpand(rec.id)"
                >
                  {{ isExpanded(rec.id) ? '收起详细文本' : '展开全部文本' }}
                </button>
              </div>
            </div>

            <!-- 底部：链上哈希存证与快捷动作 -->
            <div class="pct-card-foot">
              <div class="pct-foot-meta">
                <HashCapsule
                  :value="rec.fabric_tx_id"
                  :prefix-len="8"
                  :suffix-len="6"
                  label="TxID"
                  empty-text="待上链"
                />
                <span v-if="rec.block_height && rec.block_height > 0" class="pct-block-chip">
                  区块 #{{ rec.block_height }}
                </span>
                <span v-if="rec.files && rec.files.length > 0" class="pct-file-chip">
                  {{ rec.files.length }} 份影像附件
                </span>
              </div>

              <div class="pct-foot-actions">
                <el-button
                  v-if="allowAdopt && isRecordAccessible(rec) && !rec.is_tampered"
                  size="small"
                  plain
                  type="primary"
                  :icon="Plus"
                  @click="handleAdopt(rec)"
                >
                  引用至现病史
                </el-button>
                <el-button
                  v-if="isRecordAccessible(rec)"
                  size="small"
                  text
                  type="primary"
                  :icon="View"
                  @click="emit('view-detail', rec)"
                >
                  完整档案
                </el-button>
              </div>
            </div>
          </div>

          <!-- 相邻节点就诊时间间隔标识 -->
          <div v-if="getGapFromNext(idx)" class="pct-gap-badge">
            {{ getGapFromNext(idx) }}
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.pct-root {
  display: flex;
  flex-direction: column;
  gap: 10px;
  font-family: var(--mt-font-sans);
  color: var(--mt-text-primary);
}

.pct-toolbar {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  padding: 8px 10px;
  background: var(--mt-bg-canvas);
  border: 1px solid var(--mt-border-base);
  border-radius: var(--mt-radius-sm);
}

.pct-stats {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 6px;
}

.pct-stat-chip {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  padding: 2px 8px;
  font-size: var(--mt-text-xs);
  color: var(--mt-text-regular);
  background: var(--mt-bg-surface);
  border: 1px solid var(--mt-border-base);
  border-radius: var(--mt-radius-xs);
  font-variant-numeric: tabular-nums;
}

.pct-stat-chip.is-success {
  color: var(--mt-success);
  background: var(--mt-success-soft);
  border-color: var(--mt-success-border);
}

.pct-stat-chip.is-warning {
  color: var(--mt-warning);
  background: var(--mt-warning-soft);
  border-color: var(--mt-warning-border);
}

.pct-stat-chip.is-danger {
  color: var(--mt-danger);
  background: var(--mt-danger-soft);
  border-color: var(--mt-danger-border);
}

.pct-filters {
  display: inline-flex;
  border: 1px solid var(--mt-border-base);
  border-radius: var(--mt-radius-xs);
  overflow: hidden;
  background: var(--mt-bg-surface);
}

.pct-filter-btn {
  border: none;
  background: transparent;
  padding: 4px 10px;
  font-size: var(--mt-text-xs);
  color: var(--mt-text-secondary);
  cursor: pointer;
  transition: background-color 0.15s ease, color 0.15s ease;
}

.pct-filter-btn + .pct-filter-btn {
  border-left: 1px solid var(--mt-border-base);
}

.pct-filter-btn:hover {
  color: var(--mt-text-primary);
  background: var(--mt-bg-hover);
}

.pct-filter-btn.is-active {
  background: var(--mt-primary-soft);
  color: var(--mt-primary);
  font-weight: 600;
}

.pct-empty {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  text-align: center;
  padding: 28px 16px;
  background: var(--mt-bg-canvas);
  border: 1px dashed var(--mt-border-strong);
  border-radius: var(--mt-radius-sm);
}

.pct-empty-icon {
  color: var(--mt-text-muted);
  margin-bottom: 8px;
}

.pct-empty-title {
  margin: 0 0 4px;
  font-size: var(--mt-text-sm);
  font-weight: 600;
  color: var(--mt-text-regular);
}

.pct-empty-sub {
  margin: 0;
  font-size: var(--mt-text-xs);
  color: var(--mt-text-secondary);
}

.pct-scroll-area {
  padding-right: 4px;
  overscroll-behavior: contain;
  scrollbar-width: thin;
  scrollbar-color: #cbd5e1 transparent;
}

.pct-timeline {
  display: flex;
  flex-direction: column;
}

.pct-node {
  position: relative;
  display: grid;
  grid-template-columns: 18px minmax(0, 1fr);
  column-gap: 10px;
  padding-bottom: 14px;
}

.pct-node:last-child {
  padding-bottom: 2px;
}

.pct-rail {
  position: relative;
  display: flex;
  flex-direction: column;
  align-items: center;
}

.pct-dot {
  width: 10px;
  height: 10px;
  border-radius: 50%;
  margin-top: 14px;
  background: var(--mt-primary);
  border: 2px solid var(--mt-bg-surface);
  box-shadow: 0 0 0 1px var(--mt-primary);
  flex-shrink: 0;
  z-index: 1;
}

.pct-node.is-locked .pct-dot {
  background: var(--mt-warning);
  box-shadow: 0 0 0 1px var(--mt-warning);
}

.pct-node.is-tampered .pct-dot {
  background: var(--mt-danger);
  box-shadow: 0 0 0 1px var(--mt-danger);
}

.pct-node.is-pending .pct-dot {
  background: var(--mt-text-muted);
  box-shadow: 0 0 0 1px var(--mt-border-strong);
}

.pct-line {
  width: 2px;
  flex: 1;
  background: var(--mt-border-base);
  margin-top: 4px;
  margin-bottom: -14px;
}

.pct-card {
  background: var(--mt-bg-surface);
  border: 1px solid var(--mt-border-base);
  border-left: 3px solid var(--mt-primary);
  border-radius: var(--mt-radius-sm);
  padding: 10px 12px;
  transition: border-color 0.15s ease;
}

.pct-node.is-locked .pct-card {
  background: var(--mt-bg-canvas);
  border-left-color: var(--mt-warning);
}

.pct-node.is-tampered .pct-card {
  border-color: var(--mt-danger-border);
  border-left-color: var(--mt-danger);
}

.pct-node.is-pending .pct-card {
  border-left-color: var(--mt-border-strong);
}

.pct-card-head {
  display: flex;
  flex-wrap: wrap;
  justify-content: space-between;
  align-items: flex-start;
  gap: 8px;
  padding-bottom: 8px;
  border-bottom: 1px solid var(--mt-border-base);
}

.pct-head-left {
  display: flex;
  flex-direction: column;
  gap: 4px;
  min-width: 0;
}

.pct-date-line,
.pct-org-line {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 6px;
}

.pct-meta-icon {
  color: var(--mt-text-secondary);
  font-size: 13px;
  flex-shrink: 0;
}

.pct-date-text {
  font-family: var(--mt-font-mono);
  font-size: var(--mt-text-xs);
  font-weight: 600;
  color: var(--mt-text-primary);
}

.pct-relative-tag {
  font-size: 11px;
  padding: 1px 6px;
  border-radius: var(--mt-radius-xs);
  background: var(--mt-primary-soft);
  color: var(--mt-primary);
  font-weight: 600;
}

.pct-type-tag {
  height: 20px;
  line-height: 18px;
}

.pct-org-line {
  font-size: var(--mt-text-xs);
  color: var(--mt-text-secondary);
}

.pct-org-name {
  color: var(--mt-text-regular);
  font-weight: 600;
}

.pct-sep {
  color: var(--mt-border-strong);
}

.pct-cross-pill {
  padding: 0 5px;
  font-size: 11px;
  border-radius: var(--mt-radius-xs);
  background: var(--mt-bg-subtle);
  border: 1px solid var(--mt-border-base);
  color: var(--mt-text-regular);
}

.pct-tamper-banner {
  display: flex;
  align-items: flex-start;
  gap: 8px;
  margin-top: 8px;
  padding: 8px 10px;
  background: var(--mt-danger-soft);
  border: 1px solid var(--mt-danger-border);
  border-radius: var(--mt-radius-xs);
  color: #991b1b;
}

.pct-banner-icon {
  color: var(--mt-danger);
  margin-top: 2px;
  flex-shrink: 0;
}

.pct-banner-title {
  font-size: var(--mt-text-xs);
  font-weight: 700;
}

.pct-banner-desc {
  font-size: 11px;
  line-height: 1.45;
  margin-top: 2px;
}

.pct-locked-box {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
  margin-top: 8px;
  padding: 10px 12px;
  background: var(--mt-warning-soft);
  border: 1px solid var(--mt-warning-border);
  border-radius: var(--mt-radius-xs);
}

.pct-locked-info {
  display: flex;
  align-items: flex-start;
  gap: 8px;
  min-width: 0;
  flex: 1;
}

.pct-locked-icon {
  color: var(--mt-warning);
  margin-top: 2px;
  flex-shrink: 0;
}

.pct-locked-title {
  font-size: var(--mt-text-xs);
  font-weight: 700;
  color: #92400e;
}

.pct-locked-desc {
  font-size: 11px;
  color: #b45309;
  line-height: 1.45;
  margin-top: 2px;
}

.pct-clinical-body {
  display: flex;
  flex-direction: column;
  gap: 8px;
  margin-top: 8px;
}

.pct-infection-strip {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 4px 8px;
  background: var(--mt-danger-soft);
  border: 1px solid var(--mt-danger-border);
  border-radius: var(--mt-radius-xs);
  color: #991b1b;
  font-size: 11px;
  font-weight: 600;
}

.pct-vital-row {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}

.pct-mono-chip {
  font-family: var(--mt-font-mono);
  font-size: 11px;
  padding: 2px 6px;
  background: var(--mt-bg-subtle);
  border: 1px solid var(--mt-border-base);
  border-radius: var(--mt-radius-xs);
  color: var(--mt-text-regular);
}

.pct-mono-chip.is-vital {
  background: var(--mt-primary-soft);
  border-color: var(--mt-primary-border);
  color: var(--mt-primary-hover);
}

.pct-clinical-grid {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.pct-clinical-row {
  display: grid;
  grid-template-columns: 64px minmax(0, 1fr);
  gap: 8px;
  align-items: baseline;
}

.pct-field-label {
  font-size: 11px;
  font-weight: 600;
  color: var(--mt-text-secondary);
  background: var(--mt-bg-subtle);
  padding: 2px 6px;
  border-radius: var(--mt-radius-xs);
  text-align: center;
  white-space: nowrap;
}

.pct-field-val {
  font-size: var(--mt-text-xs);
  line-height: 1.55;
  color: var(--mt-text-regular);
  word-break: break-word;
}

.pct-field-val.is-diagnosis {
  font-size: var(--mt-text-sm);
  font-weight: 700;
  color: var(--mt-text-primary);
}

.pct-field-val.is-clamped {
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
}

.pct-sub-note {
  color: var(--mt-text-secondary);
}

.pct-expand-row {
  display: flex;
  justify-content: flex-end;
}

.pct-link-btn {
  border: none;
  background: transparent;
  padding: 0;
  font-size: 11px;
  color: var(--mt-primary);
  cursor: pointer;
}

.pct-link-btn:hover {
  color: var(--mt-primary-hover);
  text-decoration: underline;
}

.pct-card-foot {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  margin-top: 8px;
  padding-top: 8px;
  border-top: 1px dashed var(--mt-border-base);
}

.pct-foot-meta {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 6px;
}

.pct-block-chip,
.pct-file-chip {
  font-size: 11px;
  padding: 1px 6px;
  border-radius: var(--mt-radius-xs);
  background: var(--mt-bg-subtle);
  color: var(--mt-text-secondary);
  font-family: var(--mt-font-mono);
}

.pct-foot-actions {
  display: flex;
  align-items: center;
  gap: 4px;
}

.pct-gap-badge {
  grid-column: 2;
  justify-self: flex-start;
  margin-top: 6px;
  padding: 1px 8px;
  font-size: 11px;
  color: var(--mt-text-secondary);
  background: var(--mt-bg-subtle);
  border: 1px solid var(--mt-border-base);
  border-radius: var(--mt-radius-xs);
}

.pct-root.is-compact .pct-card {
  padding: 8px 10px;
}

.pct-root.is-compact .pct-clinical-row {
  grid-template-columns: 58px minmax(0, 1fr);
}
</style>
