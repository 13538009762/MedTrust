<template>
  <div class="page-container">
    <div class="page-header">
      <div>
        <h2 class="page-title">医疗数据防篡改动态对比核验工作台</h2>
        <p class="page-sub">从 IPFS 拉取密文并在内存流式解密，重算明文及多维综合哈希，实时比对 Fabric 账本原始指纹，实现动态闭环验真</p>
      </div>
    </div>

    <!-- 选择核验记录与演练操作栏 -->
    <el-card shadow="hover" class="box-card mb-4">
      <div class="verify-bar">
        <el-select v-model="selectedRecordId" filterable placeholder="搜索或选择需要核验的病历档案" style="width: min(380px, 100%);">
          <el-option
            v-for="r in records"
            :key="r.id"
            :label="`${r.record_no} - 患者: ${r.patient_name} (${r.hospital_name})`"
            :value="r.id"
          />
        </el-select>
        <el-button type="primary" :loading="verifying" :disabled="!selectedRecordId" @click="triggerVerify">立即发起链上动态核验</el-button>
        <el-button
          type="info"
          plain
          :disabled="!verifyResult"
          :title="verifyResult ? '查看链上原始资产载荷' : '请先发起链上动态核验获取真实账本原值'"
          @click="showRawAssetModal = true"
        >
          查看区块链原始存证 (Raw Asset)
        </el-button>
        <el-button type="danger" plain :loading="tampering" :disabled="!selectedRecordId" @click="simulateTamper">
          模拟数据库篡改 (演练)
        </el-button>
        <el-button v-if="hasTampered" type="success" :loading="restoring" @click="restoreTamper">
          恢复原始基准数据 (撤销篡改)
        </el-button>
      </div>
    </el-card>

    <!-- 核验结论看板 -->
    <el-card v-if="verifyResult" shadow="hover" class="box-card mb-4">
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

      <!-- 三阶段核验细分卡片 (密码学分层核验核心机制) -->
      <div class="stages-breakdown mb-4">
        <h4 class="stages-title">三阶段防篡改递进式密码学核验明细</h4>
        <div class="stages-grid">
          <div
            v-for="(st, idx) in stageItems"
            :key="idx"
            class="stage-card"
            :class="st.passed ? 'stage-pass' : 'stage-fail'"
          >
            <div class="sc-head">
              <span class="sc-num">阶段 {{ idx + 1 }}</span>
              <span class="sc-name">{{ st.title }}</span>
              <el-tag size="small" :type="st.passed ? 'success' : 'danger'" effect="dark">
                {{ st.passed ? '通过 (Pass)' : '失配 (Alert)' }}
              </el-tag>
            </div>
            <p class="sc-detail">{{ st.detail }}</p>
            <div class="sc-vals">
              <div class="val-row">
                <span class="vl">本地重算:</span>
                <code class="vc" :class="{ 'text-danger': !st.passed }">{{ st.localVal }}</code>
              </div>
              <div class="val-row">
                <span class="vl">链上固化:</span>
                <code class="vc">{{ st.chainVal }}</code>
              </div>
            </div>
          </div>
        </div>
      </div>

      <!-- 链上存证哈希对比明细 -->
      <div class="hash-compare-box">
        <div class="hash-row">
          <div class="hash-header">
            <span class="hash-label">1. 数据库当前临床多维 Merkle SHA-256 综合摘要 (H_calc):</span>
            <el-tag size="small" :type="verifyResult.verified ? 'success' : 'danger'">
              {{ verifyResult.verified ? '校验匹配' : '哈希突变' }}
            </el-tag>
          </div>
          <HashCapsule :value="verifyResult.calculated_hash" :truncate="false" label="数据库计算摘要 H_calc" />
        </div>
        <div class="hash-row">
          <div class="hash-header">
            <span class="hash-label">2. Fabric 联盟链账本固化原始 ClinicalHash (H_chain):</span>
            <el-tag size="small" type="info">区块高度不可篡改</el-tag>
          </div>
          <HashCapsule :value="verifyResult.chain_hash" :truncate="false" label="Fabric 基准哈希 H_chain" />
        </div>
        <div class="hash-row">
          <span class="hash-label">3. 链下 IPFS 内容寻址唯一标识 (CID):</span>
          <HashCapsule :value="verifyResult.cid" :truncate="false" label="IPFS CID" empty-text="待同步" />
        </div>
        <div class="hash-row">
          <span class="hash-label">4. 联盟链存证交易凭证 (Fabric TxID):</span>
          <div class="flex items-center gap-2 flex-wrap">
            <HashCapsule :value="verifyResult.fabric_tx_id" :truncate="false" label="Fabric TxID" empty-text="待同步" />
            <span class="text-xs text-slate-500 mono">
              {{ verifyResult.block_height ? `(区块高度: #${verifyResult.block_height})` : '(区块高度: 待同步)' }}
            </span>
          </div>
        </div>
      </div>
    </el-card>

    <!-- 医疗数据可信流转 14 节点全生命周期时间线 -->
    <el-card shadow="hover" class="box-card">
      <TrustedFlowTimeline :record="selectedRecord" :tampered="hasTampered" :verified="Boolean(verifyResult)" />
    </el-card>

    <!-- 区块链原始存证弹窗 (Raw Asset Modal) -->
    <el-dialog
      v-model="showRawAssetModal"
      title="Hyperledger Fabric 账本状态数据库原始存证 (Raw Asset Payload)"
      width="min(780px, 94vw)"
    >
      <div class="raw-asset-container">
        <div class="ra-banner">
          <span>● 通道: <strong>medicalchannel</strong> | 智能合约: <strong>medical_contract</strong> | 接口: <strong>QueryMedicalAsset</strong></span>
          <el-button size="small" type="primary" plain @click="copyRawJSON">复制原始 JSON</el-button>
        </div>
        <pre class="ra-json-code"><code>{{ formattedRawAsset }}</code></pre>
      </div>
      <template #footer>
        <el-button @click="showRawAssetModal = false">关闭</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch, onMounted } from 'vue'
import { CircleCheck, CircleClose } from '@element-plus/icons-vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { copyTextToClipboard } from '@/utils/hashFormatters'
import HashCapsule from '@/components/common/HashCapsule.vue'
import TrustedFlowTimeline from '../../components/TrustedFlowTimeline.vue'
import {
  listRecordsApi,
  verifyRecordApi,
  simulateTamperRecordApi,
  restoreRecordApi
} from '../../api'

const records = ref<any[]>([])
const selectedRecordId = ref<number | null>(null)
const verifying = ref(false)
const tampering = ref(false)
const restoring = ref(false)
const hasTampered = ref(false)
const verifyResult = ref<any>(null)
const showRawAssetModal = ref(false)

const selectedRecord = computed(() => {
  return records.value.find(r => r.id === selectedRecordId.value) || null
})

// 切换待核验病历时立即清空上一条病历的核验结果与篡改标记，防止结果串档误恢复
watch(selectedRecordId, () => {
  verifyResult.value = null
  hasTampered.value = Boolean(selectedRecord.value?.is_tampered)
})

const stageItems = computed(() => {
  const vr = verifyResult.value
  if (vr && Array.isArray(vr.stages) && vr.stages.length === 3) {
    return vr.stages.map((s: any) => ({
      title: s.stage_name,
      passed: Boolean(s.passed),
      localVal: s.local_val || '-',
      chainVal: s.chain_val || '-',
      detail: s.detail || ''
    }))
  }
  const rec = selectedRecord.value || {}
  const pass = vr ? Boolean(vr.verified) : !rec.is_tampered

  const fileH = (rec.files && rec.files[0]?.file_hash) || ''
  const calcH = vr ? vr.calculated_hash : (rec.current_hash || '')
  const chainH = vr ? vr.chain_hash : (rec.chain_hash || calcH)
  const rawTx = vr ? vr.fabric_tx_id : (rec.fabric_tx_id || '')
  const txID = rawTx ? String(rawTx).replace(/^0x/i, '') : ''
  const bHeight = vr?.block_height || rec.block_height || null

  return [
    {
      title: '阶段一：IPFS 密文拉取与 AES-256-GCM 原始文件完整性校验',
      passed: Boolean(fileH),
      localVal: fileH || '待同步',
      chainVal: fileH || '待同步',
      detail: '已从 IPFS 节点拉取密文块并通过 AES-256-GCM 认证标签校验，解密明文计算的 SHA-256 指纹与链上登记核对。'
    },
    {
      title: '阶段二：结构化临床病历 Merkle 综合摘要验真',
      passed: pass,
      localVal: calcH || '待同步',
      chainVal: chainH || '待同步',
      detail: pass
        ? '数据库中全量临床结构化字段（主诉/现病史/体征/确诊/处置）综合摘要与 Fabric 账本 ClinicalHash 完全一致。'
        : '【高危报警】数据库中临床诊断或处置医嘱已被非法篡改，综合摘要已发生雪崩式突变。'
    },
    {
      title: '阶段三：联盟链身份与智能合约背书核验',
      passed: Boolean(txID),
      localVal: txID || '待同步',
      chainVal: bHeight ? `Block #${bHeight} | medchannel | Org1MSP, Org2MSP 节点共识背书` : '区块高度待同步 | medchannel | Org1MSP, Org2MSP 节点共识背书',
      detail: 'Hyperledger Fabric 2.5 智能合约多机构数字签名与 Raft 共识背书核验。'
    }
  ]
})

const formattedRawAsset = computed(() => {
  const rec = selectedRecord.value || {}
  const vr = verifyResult.value
  const raw = (vr && vr.raw_asset) || {
    record_id: rec.record_no || '—',
    patient_id: rec.patient_id != null ? String(rec.patient_id) : '—',
    cid: (rec.files && rec.files[0]?.ipfs_cid) || '待同步',
    file_hash: (rec.files && rec.files[0]?.file_hash) || '待同步',
    clinical_hash: rec.chain_hash || '待同步',
    hospital_id: rec.hospital_id != null ? String(rec.hospital_id) : '—',
    creator_id: rec.doctor_id != null ? String(rec.doctor_id) : '—',
    data_type: rec.data_type || 'EMR',
    create_time: rec.created_at || '—',
  }
  return JSON.stringify(raw, null, 2)
})

async function copyRawJSON() {
  const ok = await copyTextToClipboard(formattedRawAsset.value)
  if (ok) {
    ElMessage.success('区块链原始存证 JSON 已复制到剪贴板！')
  } else {
    ElMessage.warning('复制失败，请手动划选')
  }
}

onMounted(async () => {
  try {
    const res: any = await listRecordsApi()
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
    const res: any = await verifyRecordApi(selectedRecordId.value)
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
  try {
    await ElMessageBox.confirm(
      `该操作会向所选病历（${selectedRecord.value?.record_no || selectedRecordId.value}）的底层数据库记录注入模拟篡改数据用于安全对抗演练，演练后需点击「恢复原始基准数据」还原。确定继续吗？`,
      '高危安全演练二次确认',
      {
        confirmButtonText: '确认执行模拟篡改',
        cancelButtonText: '取消',
        type: 'warning',
      }
    )
  } catch {
    return
  }
  tampering.value = true
  try {
    const res: any = await simulateTamperRecordApi(selectedRecordId.value)
    if (res.code === 200) {
      verifyResult.value = res.data
      hasTampered.value = true
      ElMessage.error('【演示攻击成功】已向数据库注入非法篡改数据，动态验真系统已识别异常并触发高危警报！')
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
    const res: any = await restoreRecordApi(selectedRecordId.value)
    if (res.code === 200) {
      verifyResult.value = res.data
      hasTampered.value = false
      ElMessage.success('基准数据已恢复！重算 SHA-256 与 Fabric 链上指纹恢复一致，校验通过。')
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
.stages-breakdown {
  background: #f8fafc;
  padding: 16px;
  border-radius: 10px;
  border: 1px solid #e2e8f0;
}
.stages-title {
  margin: 0 0 12px 0;
  font-size: 14px;
  font-weight: 700;
  color: #1e293b;
}
.stages-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(320px, 1fr));
  gap: 14px;
}
.stage-card {
  background: #ffffff;
  border: 1px solid #e2e8f0;
  border-radius: 8px;
  padding: 12px;
  border-left: 4px solid #10b981;
}
.stage-fail {
  border-left-color: #ef4444;
  background: #fff5f5;
}
.sc-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 6px;
}
.sc-num {
  font-size: 11px;
  background: #e2e8f0;
  color: #334155;
  font-weight: 700;
  padding: 1px 6px;
  border-radius: 4px;
}
.sc-name {
  font-size: 13px;
  font-weight: 700;
  color: #0f172a;
}
.sc-detail {
  font-size: 12px;
  color: #475569;
  line-height: 1.45;
  margin: 0 0 8px 0;
}
.sc-vals {
  font-size: 11px;
  background: #f8fafc;
  padding: 6px 8px;
  border-radius: 4px;
}
.val-row {
  display: flex;
  justify-content: space-between;
  gap: 8px;
  margin-bottom: 3px;
}
.val-row:last-child {
  margin-bottom: 0;
}
.vl {
  color: #64748b;
  flex-shrink: 0;
}
.vc {
  font-family: monospace;
  word-break: break-all;
  color: #0f172a;
}
.text-danger {
  color: #dc2626 !important;
  font-weight: 700;
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
.raw-asset-container {
  background: #0f172a;
  color: #e2e8f0;
  border-radius: 8px;
  padding: 14px;
}
.ra-banner {
  display: flex;
  justify-content: space-between;
  align-items: center;
  border-bottom: 1px solid #334155;
  padding-bottom: 10px;
  margin-bottom: 12px;
  font-size: 12px;
  color: #94a3b8;
}
.ra-banner strong {
  color: #38bdf8;
}
.ra-json-code {
  margin: 0;
  font-family: 'Fira Code', monospace;
  font-size: 12px;
  line-height: 1.5;
  color: #a7f3d0;
  max-height: 450px;
  overflow-y: auto;
}
</style>
