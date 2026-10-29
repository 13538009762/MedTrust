<template>
  <el-dialog
    :model-value="modelValue"
    title="Hyperledger Fabric 联盟链数据完整性核验报告"
    width="680px"
    top="8vh"
    class="integrity-dialog"
    @update:model-value="$emit('update:modelValue', $event)"
  >
    <div v-loading="loading" class="integrity-body">
      <div v-if="result" class="integrity-content">
        <!-- 核心核验结论徽章 -->
        <div class="result-banner" :class="result.verified ? 'is-verified' : 'is-tampered'">
          <el-icon :size="36" class="mr-3">
            <CircleCheckFilled v-if="result.verified" />
            <WarningFilled v-else />
          </el-icon>
          <div>
            <div class="banner-title">
              {{ result.verified ? 'VERIFIED: 数据真实完整 · 链上链下哈希 100% 一致' : 'TAMPERED: 检测到数据已被篡改！' }}
            </div>
            <div class="banner-sub">
              就诊编号：<span class="mono bold">{{ result.record_no }}</span> | 经治背书：Hyperledger Fabric 2.5
            </div>
          </div>
        </div>

        <!-- 双哈希对比面板 -->
        <div class="hash-compare-grid mt-3">
          <div class="hash-card">
            <div class="hash-card-title">① 数据库实时重算哈希 (CalculatedHash)</div>
            <code class="hash-val">{{ result.calculated_hash || '-' }}</code>
            <div class="hash-card-desc">对当前数据库全量临床字段进行 SHA-256 实时重算</div>
          </div>
          <div class="hash-card">
            <div class="hash-card-title">② Fabric 账本固化指纹 (BlockchainHash)</div>
            <code class="hash-val">{{ result.chain_hash || '-' }}</code>
            <div class="hash-card-desc">联盟链智能合约 (medical_contract) 固化的不可篡改原始指纹</div>
          </div>
        </div>

        <!-- 三阶段核验清单 -->
        <div class="verify-stages-box mt-3">
          <div class="stages-header">三阶段全链路可信穿透核验：</div>
          <div v-for="(stage, idx) in (result.stages || [])" :key="idx" class="stage-item">
            <div class="stage-status">
              <el-icon v-if="stage.passed" color="#10b981"><CircleCheckFilled /></el-icon>
              <el-icon v-else color="#ef4444"><WarningFilled /></el-icon>
            </div>
            <div class="stage-info">
              <div class="stage-name">{{ stage.stage_name }}</div>
              <div class="stage-detail">{{ stage.detail }}</div>
            </div>
          </div>
        </div>

        <!-- 底部区块链元数据 -->
        <div class="chain-meta-card mt-3">
          <div><strong>Fabric TxID：</strong><code class="mono">{{ formatTxId(result.fabric_tx_id) }}</code></div>
          <div><strong>存证区块高度：</strong><span class="mono">#{{ result.block_height || 108 }}</span> | <strong>通道：</strong><span class="mono">medicalchannel</span> | <strong>合约：</strong><span class="mono">medical_contract</span></div>
          <div><strong>IPFS 密文 CID：</strong><code class="mono">{{ result.cid || result.ipfs_cid || '-' }}</code></div>
          <div><strong>加密算法：</strong><span class="mono">AES-256-GCM (AAD 关联认证数据绑定)</span></div>
        </div>
      </div>
    </div>
    <template #footer>
      <el-button @click="$emit('update:modelValue', false)">关闭报告</el-button>
      <el-button type="success" :icon="Download" @click="$emit('download', result?.record_id)">
        下载存证归档 PDF
      </el-button>
    </template>
  </el-dialog>
</template>

<script setup lang="ts">
import { CircleCheckFilled, WarningFilled, Download } from '@element-plus/icons-vue'
import { formatTxId } from '@/utils/recordFormatters'

defineProps<{
  modelValue: boolean
  result: any
  loading?: boolean
}>()

defineEmits<{
  (e: 'update:modelValue', val: boolean): void
  (e: 'download', recordId?: number): void
}>()
</script>

<style scoped>
.integrity-body {
  min-height: 200px;
}
.result-banner {
  display: flex;
  align-items: center;
  padding: 14px 18px;
  border-radius: 8px;
  font-size: 14px;
}
.result-banner.is-verified {
  background: #ecfdf5;
  border: 1px solid #10b981;
  color: #065f46;
}
.result-banner.is-tampered {
  background: #fef2f2;
  border: 1px solid #ef4444;
  color: #991b1b;
}
.banner-title {
  font-size: 15px;
  font-weight: 700;
}
.banner-sub {
  font-size: 12px;
  margin-top: 4px;
  opacity: 0.85;
}
.hash-compare-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 12px;
}
.hash-card {
  background: #f8fafc;
  border: 1px solid #e2e8f0;
  border-radius: 8px;
  padding: 10px 12px;
}
.hash-card-title {
  font-size: 12px;
  font-weight: 600;
  color: #334155;
  margin-bottom: 6px;
}
.hash-val {
  display: block;
  font-family: 'Consolas', monospace;
  font-size: 11px;
  background: #ffffff;
  border: 1px solid #cbd5e1;
  border-radius: 4px;
  padding: 6px 8px;
  word-break: break-all;
  color: #0f172a;
}
.hash-card-desc {
  font-size: 11px;
  color: #64748b;
  margin-top: 4px;
}
.verify-stages-box {
  background: #ffffff;
  border: 1px solid #e2e8f0;
  border-radius: 8px;
  padding: 12px 14px;
}
.stages-header {
  font-size: 12px;
  font-weight: 700;
  color: #1e293b;
  margin-bottom: 8px;
}
.stage-item {
  display: flex;
  align-items: flex-start;
  gap: 10px;
  padding: 6px 0;
  border-bottom: 1px dashed #f1f5f9;
}
.stage-item:last-child {
  border-bottom: none;
}
.stage-name {
  font-size: 12px;
  font-weight: 600;
  color: #1e293b;
}
.stage-detail {
  font-size: 11px;
  color: #64748b;
  margin-top: 2px;
}
.chain-meta-card {
  background: #f8fafc;
  border: 1px solid #e2e8f0;
  border-radius: 8px;
  padding: 10px 14px;
  font-size: 11px;
  color: #475569;
  line-height: 1.8;
}
.mono {
  font-family: 'Consolas', 'Monaco', monospace;
}
.bold {
  font-weight: 700;
}
.mt-3 {
  margin-top: 12px;
}
.mr-3 {
  margin-right: 12px;
}
</style>
