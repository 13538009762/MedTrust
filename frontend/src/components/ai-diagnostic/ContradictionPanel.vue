<template>
  <!-- 维度 3: 跨院记录语义矛盾提示 — WP-16 从 AiDiagnosticDialog 抽取 -->
  <div class="tab-pane-box">
    <div class="pane-tip-row mb-3">
      <span>通过跨机构病历交叉比对，识别不同医院之间关于慢病、过敏或用药依从性的自述差异：</span>
    </div>
    <div v-if="(result.semantic_contradictions || []).length > 0" class="contradictions-grid">
      <div
        v-for="(ct, ctIdx) in (result.semantic_contradictions || [])"
        :key="ctIdx"
        class="ct-card"
        :class="String(ct.contradiction_type || '').includes('正常') ? 'ct-success' : 'ct-warning'"
      >
        <div class="ct-top-row">
          <span class="ct-tag font-bold">{{ ct.contradiction_type || '语义差异提示' }}</span>
          <el-tag size="small" :type="String(ct.contradiction_type || '').includes('正常') ? 'success' : 'warning'">
            {{ String(ct.contradiction_type || '').includes('正常') ? '比对一致' : '需当面核实' }}
          </el-tag>
        </div>
        <div class="ct-desc mt-2">{{ ct.description }}</div>
        <div class="ct-advice-row mt-2" v-if="ct.verification_advice">
          <span class="advice-label font-bold">接诊核查建议：</span>
          <span class="advice-text">{{ ct.verification_advice }}</span>
        </div>
      </div>
    </div>
    <el-empty v-else description="跨院历史病历比对一致，暂未发现明显语义矛盾" :image-size="60" />
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