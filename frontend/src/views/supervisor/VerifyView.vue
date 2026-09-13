<template>
  <div class="page-container">
    <div class="page-header">
      <div>
        <h2 class="page-title">🛡️ 医疗数据防篡改动态对比核验工作台</h2>
        <p class="page-sub">依据规划书 3.1.2 节：从 IPFS 拉取密文并在内存流式解密，重算明文及多维综合哈希，自动比对 Fabric 账本原始指纹，实现动态闭环验真</p>
      </div>
    </div>

    <!-- 选择核验记录与演练操作栏 -->
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
        <el-button type="primary" :loading="verifying" @click="triggerVerify">立即发起链上动态核验</el-button>
        <el-button type="danger" :loading="tampering" @click="simulateTamper">
          🔥 模拟真实数据库恶意篡改 (答辩攻击演练)
        </el-button>
        <el-button v-if="hasTampered" type="success" :loading="restoring" @click="restoreTamper">
          ✨ 一键恢复原始数据 (撤销篡改)
        </el-button>
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
            {{ verifyResult.verified ? 'VERIFIED · 数据真实完整 (区块链存证一致)' : 'TAMPER_ALERT · 检测到数据完整性哈希不匹配 (已被恶意篡改)' }}
          </div>
          <p class="result-desc">{{ verifyResult.message }}</p>
        </div>
      </div>

      <div class="hash-compare-box">
        <div class="hash-row">
          <div class="hash-header">
            <span class="hash-label">1. IPFS 密文解密实时计算多维 SHA-256 指纹 (H_calc):</span>
            <el-tag size="small" :type="verifyResult.verified ? 'success' : 'danger'">
              {{ verifyResult.verified ? '校验匹配' : '哈希突变' }}
            </el-tag>
          </div>
          <code class="hash-code" :class="{ 'mismatch-code': !verifyResult.verified }">{{ verifyResult.calculated_hash }}</code>
        </div>
        <div class="hash-row">
          <div class="hash-header">
            <span class="hash-label">2. Fabric 联盟链分布式账本原始固化指纹 (H_chain):</span>
            <el-tag size="small" type="info">区块高度不可篡改</el-tag>
          </div>
          <code class="hash-code">{{ verifyResult.chain_hash }}</code>
        </div>
        <div class="hash-row">
          <span class="hash-label">3. 链下 IPFS 内容寻址唯一标识 (CID):</span>
          <code class="hash-code text-muted">{{ verifyResult.cid || 'QmDefaultAttestationPayloadV2' }}</code>
        </div>
        <div class="hash-row">
          <span class="hash-label">4. 联盟链存证交易凭证 (Fabric TxID):</span>
          <code class="hash-code text-muted">{{ verifyResult.fabric_tx_id }}</code>
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
const tampering = ref(false)
const restoring = ref(false)
const hasTampered = ref(false)
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
      hasTampered.value = !res.data.verified
      if (res.data.verified) {
        ElMessage.success('核验通过：IPFS 文件指纹与 Fabric 账本凭据完全一致！')
      } else {
        ElMessage.error('核验失败：检测到病历已被篡改，系统已记录高危审计警报！')
      }
    }
  } catch (err: any) {
    ElMessage.error(err?.message || '核验请求失败')
  } finally {
    verifying.value = false
  }
}

async function simulateTamper() {
  if (!selectedRecordId.value) {
    ElMessage.warning('请先选择需要篡改演示的病历')
    return
  }
  tampering.value = true
  try {
    const res: any = await api.post(`/verification/simulate-tamper/${selectedRecordId.value}`)
    if (res.code === 200) {
      verifyResult.value = res.data
      hasTampered.value = true
      ElMessage.error('【演示攻击成功】已向数据库注入非法篡改数据！动态验真系统已秒级感知并触发高危警报！')
    }
  } catch (err: any) {
    ElMessage.error(err?.message || '模拟篡改请求失败')
  } finally {
    tampering.value = false
  }
}

async function restoreTamper() {
  if (!selectedRecordId.value) return
  restoring.value = true
  try {
    const res: any = await api.post(`/verification/restore/${selectedRecordId.value}`)
    if (res.code === 200) {
      verifyResult.value = res.data
      hasTampered.value = false
      ElMessage.success('数据已成功一键恢复！重算 SHA-256 与 Fabric 链上指纹恢复一致，绿标通过！')
    }
  } catch (err: any) {
    ElMessage.error(err?.message || '恢复请求失败')
  } finally {
    restoring.value = false
  }
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
  margin-top: 4px;
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
  flex-wrap: wrap;
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
  font-size: 17px;
  font-weight: 800;
  margin-bottom: 4px;
}
.result-desc {
  font-size: 13px;
  margin: 0;
  opacity: 0.9;
}
.hash-compare-box {
  background: #f8fafc;
  padding: 16px;
  border-radius: 10px;
  display: flex;
  flex-direction: column;
  gap: 14px;
}
.hash-row {
  display: flex;
  flex-direction: column;
  gap: 4px;
}
.hash-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
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
.mismatch-code {
  background: #fff1f2;
  border-color: #fda4af;
  color: #be123c;
  font-weight: 600;
}
.text-muted {
  color: #64748b;
}
</style>
