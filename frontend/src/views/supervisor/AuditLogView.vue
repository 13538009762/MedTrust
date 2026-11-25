<template>
  <div class="page-container">
    <div class="page-header">
      <div>
        <h2 class="page-title">全局不可篡改审计日志溯源</h2>
        <p class="page-sub">系统内所有上传、授权、调阅、Break-Glass、核验全生命周期均自动记账并锚定至 Fabric 联盟链</p>
      </div>
    </div>

    <el-card shadow="hover" class="box-card">
      <div class="filter-row flex-wrap">
        <el-input
          v-model="searchKeyword"
          placeholder="搜索操作主体、目标编号或日志 UUID..."
          clearable
          style="width: min(260px, 100%);"
        />
        <el-select v-model="filterOp" placeholder="操作类型过滤" clearable style="width: 210px;" @change="loadLogs">
          <el-option label="全部类型" value="" />
          <el-option label="病历上传 (UPLOAD)" value="UPLOAD" />
          <el-option label="跨院调阅 (ACCESS)" value="ACCESS" />
          <el-option label="患者授权核准 (GRANT)" value="GRANT" />
          <el-option label="紧急破窗访问 (BREAK_GLASS)" value="BREAK_GLASS" />
          <el-option label="防篡改核验 (VERIFY)" value="VERIFY" />
          <el-option label="监管合规裁决 (AUDIT_CLOSE)" value="AUDIT_CLOSE" />
          <el-option label="解除医生受限 (UNRESTRICT_DOCTOR)" value="UNRESTRICT_DOCTOR" />
          <el-option label="安全登录认证 (LOGIN)" value="LOGIN" />
        </el-select>
        <el-select v-model="filterRisk" placeholder="风险等级" clearable style="width: 150px;" @change="loadLogs">
          <el-option label="全部风险" value="" />
          <el-option label="低风险 (LOW)" value="LOW" />
          <el-option label="中风险 (MEDIUM)" value="MEDIUM" />
          <el-option label="高风险 (HIGH)" value="HIGH" />
        </el-select>
        <el-button type="primary" plain :loading="loading" @click="loadLogs">刷新流水</el-button>
      </div>

      <el-table :data="pagedLogs" v-loading="loading" stripe style="width: 100%" empty-text="当前筛选条件下暂无审计日志">
        <el-table-column prop="log_id" label="日志追踪 UUID" width="180">
          <template #default="{ row }">
            <HashCapsule :value="row.log_id" :prefix-len="8" :suffix-len="6" label="日志 UUID" />
          </template>
        </el-table-column>
        <el-table-column prop="created_at" label="操作时间戳" width="170">
          <template #default="{ row }">
            {{ row.created_at ? row.created_at.substring(0, 19).replace('T', ' ') : '' }}
          </template>
        </el-table-column>
        <el-table-column prop="user_name" label="操作主体" width="120" />
        <el-table-column prop="operation_type" label="操作类型" width="160">
          <template #default="{ row }">
            <el-tag size="small" :type="getOpTagType(row.operation_type)">{{ formatOpType(row.operation_type) }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="target_id" label="目标编号" width="150" />
        <el-table-column prop="result" label="执行结果" width="110">
          <template #default="{ row }">
            <el-tag size="small" :type="isPositiveResult(row.result) ? 'success' : 'danger'">
              {{ formatResultText(row.result) }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="risk_level" label="风险评分评级" width="120">
          <template #default="{ row }">
            <el-tag size="small" :type="row.risk_level === 'LOW' ? 'success' : (row.risk_level === 'MEDIUM' ? 'warning' : 'danger')">
              {{ formatRiskText(row.risk_level) }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="fabric_tx_id" label="Fabric 存证 TxID" min-width="200">
          <template #default="{ row }">
            <HashCapsule :value="row.fabric_tx_id" :prefix-len="8" :suffix-len="6" label="Fabric TxID" />
          </template>
        </el-table-column>
      </el-table>

      <div v-if="filteredLogs.length > 0" class="mt-4 flex justify-between items-center flex-wrap gap-2">
        <span class="text-xs text-slate-500">共 {{ filteredLogs.length }} 条链上审计流水</span>
        <el-pagination
          v-model:current-page="currentPage"
          v-model:page-size="pageSize"
          :page-sizes="[15, 30, 50, 100]"
          layout="total, sizes, prev, pager, next"
          :total="filteredLogs.length"
          background
          small
        />
      </div>
    </el-card>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch, onMounted } from 'vue'
import HashCapsule from '@/components/common/HashCapsule.vue'
import { listAuditLogsApi } from '../../api'

const logs = ref<any[]>([])
const loading = ref(false)
const searchKeyword = ref('')
const filterOp = ref('')
const filterRisk = ref('')
const currentPage = ref(1)
const pageSize = ref(15)

const filteredLogs = computed(() => {
  const kw = searchKeyword.value.trim().toLowerCase()
  if (!kw) return logs.value
  return logs.value.filter(item =>
    `${item.log_id || ''} ${item.user_name || ''} ${item.target_id || ''} ${item.operation_type || ''}`
      .toLowerCase()
      .includes(kw)
  )
})

const pagedLogs = computed(() => {
  const start = (currentPage.value - 1) * pageSize.value
  return filteredLogs.value.slice(start, start + pageSize.value)
})

watch([searchKeyword, filterOp, filterRisk], () => {
  currentPage.value = 1
})

onMounted(() => {
  loadLogs()
})

function isPositiveResult(res?: string): boolean {
  return res === 'SUCCESS' || res === 'ALLOWED'
}

function formatResultText(res?: string): string {
  const map: Record<string, string> = {
    SUCCESS: '成功',
    ALLOWED: '已放行',
    DENIED: '已拦截',
    FAILED: '失败',
    REJECTED: '已驳回',
  }
  return map[res || ''] || res || '-'
}

function formatRiskText(level?: string): string {
  const map: Record<string, string> = {
    LOW: '低风险',
    MEDIUM: '中风险',
    HIGH: '高风险',
  }
  return map[level || ''] || level || '-'
}

function getOpTagType(op?: string): 'success' | 'warning' | 'danger' | 'info' | 'primary' {
  if (!op) return 'info'
  if (op.includes('BREAK') || op.includes('EMERGENCY')) return 'danger'
  if (op.includes('GRANT') || op.includes('UPLOAD')) return 'success'
  if (op.includes('VERIFY')) return 'warning'
  if (op.includes('LOGIN')) return 'info'
  return 'primary'
}

function formatOpType(op?: string): string {
  const map: Record<string, string> = {
    UPLOAD: '病历上链',
    GRANT: '患者授权核准',
    ACCESS: '跨院调阅',
    BREAK_GLASS: '紧急破窗调阅',
    VERIFY: '防篡改核验',
    VERIFY_RECORD: '防篡改核验',
    AUDIT_CLOSE: '监管合规裁决',
    UNRESTRICT_DOCTOR: '解除执业限制',
    FEEDBACK: '患者异议申诉',
    LOGIN: '身份认证登录',
  }
  return map[op || ''] || op || '-'
}

async function loadLogs() {
  loading.value = true
  try {
    const res: any = await listAuditLogsApi({
      operation_type: filterOp.value,
      risk_level: filterRisk.value,
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
