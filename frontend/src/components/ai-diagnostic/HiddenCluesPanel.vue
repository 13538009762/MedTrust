<template>
  <!-- 维度 2: 隐匿过敏与高危线索 — WP-16 从 AiDiagnosticDialog 抽取 -->
  <div class="tab-pane-box">
    <div class="pane-tip-row mb-3">
      <span>从跨院已授权自由文本主诉、病程与现病史中自动抽取的口语化隐匿风险线索：</span>
    </div>
    <div class="clues-grid">
      <div
        v-for="(clue, cIdx) in (result.hidden_allergy_clues || [])"
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
</template>

<script setup lang="ts">
defineProps<{
  result: any
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
</style>