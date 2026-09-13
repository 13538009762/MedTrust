<template>
  <div class="page-container">
    <div class="page-header">
      <div>
        <h2 class="page-title">🛡️ 全局不可篡改审计日志溯源</h2>
        <p class="page-sub">系统内所有上传、授权、调阅、Break-Glass、核验全生命周期均自动记账并锚定至 Fabric 联盟链</p>
      </div>
    </div>

    <el-card shadow="hover" class="box-card">
      <div class="filter-row">
        <el-select v-model="filterOp" placeholder="操作类型过滤" clearable style="width: 180px;" @change="loadLogs">
          <el-option label="全部类型" value="" />
          <el-option label="UPLOAD (病历上传)" value="UPLOAD" />
          <el-option label="ACCESS (跨院调阅)" value="ACCESS" />
          <el-option label="BREAK_GLASS (紧急访问)" value="BREAK_GLASS" />
          <el-option label="AUTHORIZE (患者授权)" value="AUTHORIZE" />
          <el-option label="VERIFY (防篡改核验)" value="VERIFY" />
        </el-select>
        <el-select v-model="filterRisk" placeholder="风险等级" clearable style="width: 150px;" @change="loadLogs">
          <el-option label="全部风险" value="" />
          <el-option label="LOW (低风险)" value="LOW" />
          <el-option label="MEDIUM (中风险)" value="MEDIUM" />
          <el-option label="HIGH (高风险)" value="HIGH" />
        </el-select>
        <el-button type="primary" plain @click="loadLogs">刷新流水</el-button>
      </div>

      <el-table :data="logs" v-loading="loading" stripe style="width: 100%">
        <el-table-column prop="log_id" label="日志追踪 UUID" width="190" />
        <el-table-column prop="created_at" label="操作时间戳" width="170">
          <template #default="{ row }">
            {{ row.created_at ? row.created_at.substring(0, 19).replace('T', ' ') : '' }}
          </template>
        </el-table-column>
        <el-table-column prop="user_name" label="操作主体" width="120" />
        <el-table-column prop="operation_type" label="操作类型" width="140">
          <template #default="{ row }">
            <el-tag size="small">{{ row.operation_type }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="target_id" label="目标编号" width="150" />
        <el-table-column prop="result" label="执行结果" width="110">
          <template #default="{ row }">
            <el-tag size="small" :type="row.result === 'SUCCESS' ? 'success' : 'danger'">
              {{ row.result }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="risk_level" label="风险评分评级" width="120">
          <template #default="{ row }">
            <el-tag size="small" :type="row.risk_level === 'LOW' ? 'success' : (row.risk_level === 'MEDIUM' ? 'warning' : 'danger')">
              {{ row.risk_level }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="fabric_tx_id" label="Fabric 存证 TxID" min-width="180">
          <template #default="{ row }">
            <span class="tx-hash">{{ row.fabric_tx_id ? row.fabric_tx_id.substring(0, 18) + '...' : '' }}</span>
          </template>
        </el-table-column>
      </el-table>
    </el-card>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import api from '../../api/client'

const logs = ref([])
const loading = ref(false)
const filterOp = ref('')
const filterRisk = ref('')

onMounted(() => {
  loadLogs()
})

async function loadLogs() {
  loading.value = true
  try {
    const res: any = await api.get('/audit-logs', {
      params: {
        operation_type: filterOp.value,
        risk_level: filterRisk.value,
      }
    })
    if (res.code === 200) {
      logs.value = res.data
    }
  } finally {
    loading.value = false
  }
}
</script>

<style scoped>
.page-container {
  max-width: 1300px;
  margin: 0 auto;
}
.page-header {
  margin-bottom: 20px;
}
.page-title {
  font-size: 22px;
  font-weight: 800;
  color: #0f172a;
}
.page-sub {
  font-size: 13px;
  color: #64748b;
}
.box-card {
  border-radius: 14px;
}
.filter-row {
  display: flex;
  gap: 12px;
  margin-bottom: 16px;
}
.tx-hash {
  font-family: monospace;
  color: #8b5cf6;
}
</style>
