<template>
  <!-- 维度 1: SOAP 接诊结构化汇总 (核心主视图) — WP-16 从 AiDiagnosticDialog 抽取 -->
  <div class="tab-pane-box">
    <div class="pane-tip-row flex-between mb-3">
      <span>由医疗专业智能引擎对跨院已授权病程记录进行四段式客观提炼（严格遵循 Doctor-in-the-Loop 辅助诊疗与临床决策合规准则）：</span>
      <el-button
        type="primary"
        size="small"
        :icon="DocumentCopy"
        :disabled="!result.soap_summary"
        @click="$emit('adopt-soap', result.soap_summary)"
      >
        引用 SOAP 摘要至草稿
      </el-button>
    </div>

    <div class="soap-grid-layout">
      <div class="soap-box s-box">
        <div class="soap-box-header">
          <span class="soap-letter">S</span>
          <span class="soap-title">患者主诉与既往病史 (Subjective)</span>
        </div>
        <div class="soap-box-body">
          {{ result.soap_summary?.subjective || '暂无既往主诉' }}
        </div>
      </div>

      <div class="soap-box o-box">
        <div class="soap-box-header">
          <span class="soap-letter">O</span>
          <span class="soap-title">跨院客观检查与生命体征 (Objective)</span>
        </div>
        <div class="soap-box-body">
          {{ result.soap_summary?.objective || '暂无跨院客观检查与生命体征记录' }}
        </div>
      </div>

      <div class="soap-box a-box">
        <div class="soap-box-header">
          <span class="soap-letter">A</span>
          <span class="soap-title">各院既有临床确诊客观汇总 (Assessment)</span>
        </div>
        <div class="soap-box-body">
          {{ result.soap_summary?.assessment || '暂无跨院确诊记录' }}
        </div>
        <div class="soap-box-tip text-muted">
          说明：客观摘录各院既有病历确诊结论，系统不做新疾病确诊，供接诊医生鉴别衔接。
        </div>
      </div>

      <div class="soap-box p-box">
        <div class="soap-box-header">
          <span class="soap-letter">P</span>
          <span class="soap-title">历史处置与已用药物记录 (Plan)</span>
        </div>
        <div class="soap-box-body">
          {{ result.soap_summary?.plan || '暂无历史处方' }}
        </div>
        <div class="soap-box-tip text-muted">
          说明：仅供用药史追溯与防范重复用药，当前处方用药由接诊医生亲自开具。
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { DocumentCopy } from '@element-plus/icons-vue'

defineProps<{
  result: any
}>()

defineEmits<{
  (e: 'adopt-soap', soap: any): void
}>()
</script>

<style scoped>
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
</style>