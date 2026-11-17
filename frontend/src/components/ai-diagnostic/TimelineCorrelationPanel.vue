<template>
  <!-- 跨期病历时序演变与时间间隔分析 — WP-16 从 AiDiagnosticDialog 抽取 -->
  <div class="tab-pane-box">
    <div class="timeline-evolution-card mb-3">
      <div class="tec-title">
        <el-icon><Timer /></el-icon>
        <span>发病时间间隔规律与动态特征</span>
      </div>
      <div class="tec-desc">
        {{ result.symptom_correlation?.progression_pattern || '就诊档案记录了完整的病程发展时空链条。' }}
      </div>
    </div>

    <!-- 垂直时序演变链 -->
    <div class="encounter-timeline-wrapper">
      <div
        v-for="(t, idx) in result.timeline_correlation"
        :key="t.record_id || idx"
        class="timeline-node-card"
      >
        <div class="tnc-left">
          <div class="tnc-dot" :class="t.evolutionStatus === '频发加剧' ? 'danger' : 'normal'"></div>
          <div v-if="idx < result.timeline_correlation.length - 1" class="tnc-line"></div>
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
</template>

<script setup lang="ts">
import { Timer } from '@element-plus/icons-vue'

defineProps<{
  result: any
  currentSymptoms?: string
  presentIllness?: string
}>()

function getEvolutionTagType(status?: string) {
  if (status === '频发加剧' || status === '急性恶化') return 'danger'
  if (status === '间歇复发') return 'warning'
  if (status === '平稳代偿') return 'success'
  return 'info'
}
</script>

<style scoped>
.tab-pane-box {
  background: #ffffff;
  border-radius: 10px;
  padding: 16px;
  border: 1px solid #e2e8f0;
}

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
</style>