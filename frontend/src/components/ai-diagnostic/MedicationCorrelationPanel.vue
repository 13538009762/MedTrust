<template>
  <!-- 症状、既往用药与化验单交叉比对 — WP-16 从 AiDiagnosticDialog 抽取 -->
  <div class="tab-pane-box">
    <!-- 既往用药与疗效反应 -->
    <div class="detail-section-card mb-3">
      <div class="dsc-header">
        <el-icon color="#0284c7"><FolderChecked /></el-icon>
        <span>过往使用药物、治疗反应与耐药机制分析</span>
      </div>
      <div class="dsc-body">
        <div class="med-history-tags mb-2" v-if="result.medication_correlation?.past_medications?.length">
          <span class="mht-label">既往病历使用处方药品清单：</span>
          <el-tag
            v-for="med in result.medication_correlation.past_medications"
            :key="med"
            type="info"
            size="small"
            effect="light"
            class="mr-1 mb-1"
          >
            {{ med }}
          </el-tag>
        </div>
        <div class="dsc-text-block">
          <strong>用药疗效与病情反复评估：</strong>
          <p>{{ result.medication_correlation?.drug_response_analysis }}</p>
        </div>
        <div class="dsc-text-block alert-block mt-2">
          <strong>用药配伍禁忌与安全提示：</strong>
          <p>{{ result.medication_correlation?.interactions_and_risks }}</p>
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
          :data="result.lab_correlation?.abnormal_indicators || []"
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
          <span>{{ result.lab_correlation?.summary }}</span>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { FolderChecked, Collection, InfoFilled } from '@element-plus/icons-vue'

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
</style>