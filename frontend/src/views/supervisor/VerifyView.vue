<template>
  <div class="page-container">
    <div class="page-header">
      <div>
        <h2 class="page-title">🛡️ 医疗数据防篡改动态对比核验工作台</h2>
        <p class="page-sub">依据 2.0.md 3.1.2 节：从 IPFS 拉取密文解密计算当前明文 SHA-256，与 Fabric 账本原始指纹比对，亮证防篡改特性</p>
      </div>
    </div>

    <!-- 选择核验记录 -->
    <el-card shadow="hover" class="box-card mb-4">
      <div class="verify-bar">
        <el-select v-model="selectedRecordId" placeholder="请选择需要核验的病历档案" style="width: 380px;">
          <el-option
            v-for="r in records"
            :key="r.id"
            :label="`${r.record_no} - 患者: ${r.patient_name} (${r.hospital_name})`"
            :value="r.id"
          />
        </el-select>
        <el-button type="primary" :loading="verifying" @click="triggerVerify">立即发起链上完整性核验</el-button>
        <el-button type="danger" plain @click="simulateTamper">模拟被恶意篡改 (答辩演示)</el-button>
      </div>
    </el-card>

    <!-- 核验结论看板 -->
    <el-card v-if="verifyResult" shadow="hover" class="box-card">
      <div class="result-badge" :class="verifyResult.verified ? 'verified-bg' : 'tampered-bg'">
        <el-icon :size="48" class="result-icon">
          <CircleCheck v-if="verifyResult.verified" />
          <CircleClose v-else />
        </el-icon>
        <div class="result-info">
          <div class="result-title">
            {{ verifyResult.verified ? 'VERIFIED · 数据真实完整 (未遭篡改)' : 'TAMPERED · 数据已被非法篡改 (警报)' }}
          </div>
          <p class="result-desc">{{ verifyResult.message }}</p>
        </div>
      </div>

      <div class="hash-compare-box">
        <div class="hash-row">
          <span class="hash-label">1. IPFS 解密文件计算 SHA-256 指纹 (H_calc):</span>
          <code class="hash-code">{{ verifyResult.calculated_hash }}</code>
        </div>
        <div class="hash-row">
          <span class="hash-label">2. Fabric 联盟链分布式账本原始存证摘要 (H_chain):</span>
          <code class="hash-code">{{ verifyResult.chain_hash }}</code>
        </div>
        <div class="hash-row">
          <span class="hash-label">3. 链下 IPFS 寻址 CID:</span>
          <code class="hash-code">{{ verifyResult.cid }}</code>
        </div>
        <div class="hash-row">
          <span class="hash-label">4. 账本交易凭证 TxID:</span>
          <code class="hash-code">{{ verifyResult.fabric_tx_id }}</code>
        </div>
      </div>
    </el-card>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { CircleCheck, CircleClose } from '@element-plus/icons-vue'
import { ElMessage } from 'element-plus'
import api from '../../api/client'

const records = ref<any[]>([])
const selectedRecordId = ref<number | null>(null)
const verifying = ref(false)
const verifyResult = ref<any>(null)

onMounted(async () => {
  try {
    const res: any = await api.get('/medical-records')
    if (res.code === 200 && res.data.length) {
      records.value = res.data
      selectedRecordId.value = res.data[0].id
    }
  } catch (err) {
    console.error(err)
  }
})

async function triggerVerify() {
  if (!selectedRecordId.value) return
  verifying.value = true
  try {
    const res: any = await api.post(`/verification/${selectedRecordId.value}`)
    if (res.code === 200) {
      verifyResult.value = res.data
      if (res.data.verified) {
        ElMessage.success('核验通过：IPFS 文件指纹与 Fabric 账本凭据完全一致！')
      }
    }
  } catch (err: any) {
    ElMessage.error(err?.message || '核验请求失败')
  } finally {
    verifying.value = false
  }
}

function simulateTamper() {
  if (!verifyResult.value) {
    ElMessage.warning('请先点击发起核验，再点击模拟篡改')
    return
  }
  verifyResult.value = {
    ...verifyResult.value,
    calculated_hash: '9f86d081884c7d659a2feaa0c55ad015a3bf4f1b2b0b822cd15d6c15b0f00a08',
    verified: false,
    message: '【高危安全告警】IPFS 解密哈希与 Fabric 链上固化指纹不匹配！目标病历明文或密文在链下遭遇未授权恶意篡改！',
  }
  ElMessage.error('模拟篡改生效：已触发系统高危篡改告警！')
}
</script>

<style scoped>
.page-container {
  max-width: 1200px;
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
.mb-4 {
  margin-bottom: 16px;
}
.verify-bar {
  display: flex;
  gap: 12px;
  align-items: center;
}
.result-badge {
  display: flex;
  align-items: center;
  gap: 16px;
  padding: 20px 24px;
  border-radius: 12px;
  margin-bottom: 20px;
}
.verified-bg {
  background: #ecfdf5;
  border: 1px solid #6ee7b7;
  color: #047857;
}
.tampered-bg {
  background: #fef2f2;
  border: 1px solid #fca5a5;
  color: #b91c1c;
}
.result-icon {
  flex-shrink: 0;
}
.result-title {
  font-size: 18px;
  font-weight: 800;
  margin-bottom: 4px;
}
.result-desc {
  font-size: 14px;
  margin: 0;
  opacity: 0.9;
}
.hash-compare-box {
  background: #f8fafc;
  padding: 16px;
  border-radius: 10px;
  display: flex;
  flex-direction: column;
  gap: 12px;
}
.hash-row {
  display: flex;
  flex-direction: column;
  gap: 4px;
}
.hash-label {
  font-size: 13px;
  color: #475569;
  font-weight: 600;
}
.hash-code {
  background: #ffffff;
  padding: 8px 12px;
  border: 1px solid #e2e8f0;
  border-radius: 6px;
  font-size: 13px;
  color: #0f172a;
  word-break: break-all;
}
</style>
