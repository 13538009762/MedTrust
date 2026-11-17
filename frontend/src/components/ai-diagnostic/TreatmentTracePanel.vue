<template>
  <!-- 既往处置追溯与接诊核查线索（严格遵循非诊断、非处方决策合规红线）— WP-16 从 AiDiagnosticDialog 抽取 -->
  <div class="tab-pane-box">
    <div class="treatment-overview-card mb-3">
      <div class="toc-header">
        <span class="toc-title">既往跨院处置记录归纳与接诊核查要点（仅供病历文书整理参考，AI 不下达诊断或处方）</span>
        <el-button
          type="primary"
          plain
          size="small"
          :icon="Select"
          @click="$emit('adopt-treatment', result.recommended_treatment)"
        >
          引用既往处置摘要至病史备注
        </el-button>
      </div>
      <div class="toc-text font-mono" style="white-space: pre-wrap;">
        {{ result.recommended_treatment?.treatment_plan_text }}
      </div>
    </div>

    <!-- 既往用药记录摘录表 -->
    <div class="rx-recommend-table-box mb-3">
      <div class="rtt-header">
        <span>既往已授权病历记载药物摘录（仅供防重复用药与禁忌核对，非处方建议）：</span>
      </div>
      <el-table
        :data="result.recommended_treatment?.prescriptions || []"
        size="small"
        border
        class="rx-table"
      >
        <el-table-column prop="category" label="既往用药分类" width="140" />
        <el-table-column prop="drug_name" label="历史病历记载药品及规格" min-width="220" />
        <el-table-column prop="usage" label="历史记录用法与核对提示" min-width="220" />
      </el-table>
    </div>

    <!-- 关联既往检查项目参考 -->
    <div v-if="result.recommended_treatment?.suggested_orders?.length" class="suggested-orders-box">
      <span class="sob-label">关联既往检查项目参考（供医生核对是否需复查）：</span>
      <div class="sob-chips">
        <el-tag
          v-for="order in result.recommended_treatment.suggested_orders"
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
</template>

<script setup lang="ts">
import { Select } from '@element-plus/icons-vue'

defineProps<{
  result: any
}>()

defineEmits<{
  (e: 'adopt-treatment', treatment: any): void
}>()
</script>

<style scoped>
.tab-pane-box {
  background: #ffffff;
  border-radius: 10px;
  padding: 16px;
  border: 1px solid #e2e8f0;
}

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
</style>