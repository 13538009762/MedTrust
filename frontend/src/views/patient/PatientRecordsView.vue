<template>
  <div class="page-container">
    <div class="page-header">
      <div>
        <h2 class="page-title">📁 我的电子健康档案与病历大表格</h2>
        <p class="page-sub">查看您在联盟链医疗机构建立的所有门诊病历、结构化临床问诊记录、检验报告与影像数据</p>
      </div>
    </div>

    <el-card shadow="hover" class="box-card">
      <el-timeline v-if="records.length">
        <el-timeline-item
          v-for="rec in records"
          :key="rec.id"
          :timestamp="rec.created_at ? rec.created_at.substring(0, 16).replace('T', ' ') : ''"
          placement="top"
          type="primary"
        >
          <el-card class="timeline-card">
            <div class="card-head">
              <span class="rec-title">{{ rec.hospital_name }} · {{ rec.doctor_name }} 医生</span>
              <div class="tag-group">
                <el-tag v-if="rec.is_tampered" size="small" type="danger" effect="dark" class="tamper-tag-glow">🚨 存在篡改风险</el-tag>
                <el-tag v-else size="small" type="success" effect="light">🛡️ 链上核验通过</el-tag>
                <el-tag size="small">{{ rec.data_type }}</el-tag>
                <el-tag size="small" type="success">Fabric 区块 #{{ rec.block_height }}</el-tag>
              </div>
            </div>

            <!-- 结构化关键信息胶囊卡 -->
            <div v-if="rec.onset_time || rec.symptoms || rec.treatment_plan" class="structured-capsule">
              <div class="capsule-item">
                <span class="capsule-label">🕒 发病时间：</span>
                <span>{{ rec.onset_time || '接诊前' }}（{{ rec.duration || '急性起病' }}）</span>
              </div>
              <div v-if="rec.vital_signs" class="capsule-item">
                <span class="capsule-label">🩺 生命体征：</span>
                <span class="vitals-val">{{ rec.vital_signs }}</span>
              </div>
              <div v-if="rec.symptoms" class="capsule-item">
                <span class="capsule-label">🤒 核心症状：</span>
                <span>{{ rec.symptoms }}</span>
              </div>
              <div v-if="rec.etiology" class="capsule-item">
                <span class="capsule-label">🔍 病因与诱因：</span>
                <span class="cause-text">{{ rec.etiology }}</span>
              </div>
              <div v-if="rec.treatment_plan" class="capsule-item">
                <span class="capsule-label">💊 医生建议方案：</span>
                <span class="plan-text">{{ rec.treatment_plan }}</span>
              </div>
            </div>

            <p class="diag-text"><strong>病历诊断小结：</strong>{{ rec.diagnosis }}</p>

            <div class="meta-row">
              <span>病历单号: <code>{{ rec.record_no }}</code></span>
              <span>区块链存证 TxID: <code class="tx-hash">{{ rec.fabric_tx_id }}</code></span>
              <el-button link type="primary" size="small" @click="viewFullDetail(rec)">查看完整病历卡片</el-button>
            </div>
          </el-card>
        </el-timeline-item>
      </el-timeline>
      <el-empty v-else description="暂无就诊记录" />
    </el-card>

    <!-- 完整病历大表格对话框 -->
    <el-dialog v-model="detailVisible" title="个人就诊档案（结构化临床大表单视图）" width="760px">
      <div v-if="selectedRec" class="patient-detail-box">
        <!-- 篡改告警横幅 -->
        <div v-if="selectedRec.is_tampered" class="tamper-warning-box">
          <div class="tw-head">
            <el-icon class="tw-icon"><WarningFilled /></el-icon>
            <span class="tw-title">【高危安全警报】检测到您的病历已被数据库非法篡改！</span>
          </div>
          <div class="tw-body">
            系统检测到底层 MySQL 数据库中的临床数据与 Fabric 联盟链上不可篡改的存证指纹<strong>不匹配</strong>！该病历中的用药方案、药物过敏史或核心诊断已被黑客攻击篡改，<strong>临床严禁直接采信该病历！</strong>系统已阻断非法使用并记录高危安全审计！
          </div>
          <div class="tw-hashes">
            <div class="tw-h-item danger">
              <span class="lbl">🚨 数据库当前计算哈希 (Current Hash)：</span>
              <code>{{ selectedRec.current_hash }}</code>
            </div>
            <div class="tw-h-item chain">
              <span class="lbl">🛡️ 区块链不可篡改基准 (Chain Hash)：</span>
              <code>{{ selectedRec.chain_hash }}</code>
            </div>
          </div>
        </div>

        <div v-else class="verified-safe-box">
          <el-icon><CircleCheckFilled /></el-icon>
          <span><strong>🛡️ 区块链防篡改校验通过：</strong>该病历所有临床症状、用药方案与影像数据指纹均与 Fabric 联盟链上固化存证 100% 严格一致，数据真实完整，未遭任何篡改。</span>
        </div>

        <table class="clinical-structured-table">
          <tbody>
            <tr>
              <th width="140">就诊医疗机构</th>
              <td>{{ selectedRec.hospital_name }}（主治医生：{{ selectedRec.doctor_name }}）</td>
            </tr>
            <tr>
              <th>病历编号与时间</th>
              <td>单号：{{ selectedRec.record_no }} | 就诊时间：{{ selectedRec.created_at ? selectedRec.created_at.substring(0, 16).replace('T', ' ') : '-' }}</td>
            </tr>
            <tr>
              <th>发病与病程</th>
              <td>发病时间：{{ selectedRec.onset_time || '接诊前' }} | 持续周期：{{ selectedRec.duration || '急性发作' }}</td>
            </tr>
            <tr v-if="selectedRec.vital_signs">
              <th>生命体征参数</th>
              <td><strong style="color: #0d9488;">{{ selectedRec.vital_signs }}</strong></td>
            </tr>
            <tr>
              <th>主要临床症状</th>
              <td>{{ selectedRec.symptoms || selectedRec.diagnosis }}</td>
            </tr>
            <tr>
              <th>诱发因素与病因</th>
              <td><div class="cause-box">{{ selectedRec.etiology || '无特殊诱因记录' }}</div></td>
            </tr>
            <tr>
              <th>治疗建议方案</th>
              <td><div class="plan-box">{{ selectedRec.treatment_plan || '遵医嘱随诊' }}</div></td>
            </tr>
            <tr>
              <th>完整病历小结</th>
              <td><pre class="full-pre">{{ selectedRec.diagnosis }}</pre></td>
            </tr>
          </tbody>
        </table>
      </div>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { WarningFilled, CircleCheckFilled } from '@element-plus/icons-vue'
import api from '../../api/client'

const records = ref<any[]>([])
const detailVisible = ref(false)
const selectedRec = ref<any>(null)

onMounted(async () => {
  try {
    const res: any = await api.get('/medical-records')
    if (res.code === 200) {
      records.value = res.data
    }
  } catch (err) {
    console.error(err)
  }
})

async function viewFullDetail(rec: any) {
  try {
    const res: any = await api.get(`/medical-records/${rec.id}`)
    if (res.code === 200) {
      selectedRec.value = res.data
    } else {
      selectedRec.value = rec
    }
  } catch {
    selectedRec.value = rec
  }
  detailVisible.value = true
}
</script>

<style scoped>
.page-container {
  max-width: 1000px;
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
.timeline-card {
  border-radius: 12px;
  border: 1px solid #e2e8f0;
}
.card-head {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 12px;
}
.rec-title {
  font-size: 15px;
  font-weight: 700;
  color: #1e293b;
}
.tag-group {
  display: flex;
  gap: 8px;
}
.structured-capsule {
  background: #f8fafc;
  border: 1px solid #e2e8f0;
  padding: 10px 14px;
  border-radius: 8px;
  margin-bottom: 12px;
  display: flex;
  flex-direction: column;
  gap: 6px;
  font-size: 13px;
}
.capsule-label {
  font-weight: 600;
  color: #475569;
}
.vitals-val {
  font-weight: 600;
  color: #0d9488;
}
.cause-text {
  color: #b45309;
}
.plan-text {
  color: #047857;
  font-weight: 500;
}
.diag-text {
  font-size: 14px;
  color: #334155;
  margin-bottom: 12px;
  line-height: 1.6;
}
.meta-row {
  display: flex;
  align-items: center;
  gap: 20px;
  font-size: 12px;
  color: #64748b;
}
.tx-hash {
  color: #8b5cf6;
}

.clinical-structured-table {
  width: 100%;
  border-collapse: collapse;
  font-size: 13px;
}
.clinical-structured-table th,
.clinical-structured-table td {
  border: 1px solid #e2e8f0;
  padding: 10px 14px;
  text-align: left;
}
.clinical-structured-table th {
  background: #f1f5f9;
  color: #475569;
  font-weight: 600;
}
.cause-box {
  background: #fffbeb;
  padding: 6px 10px;
  border-radius: 4px;
  color: #92400e;
}
.plan-box {
  background: #ecfdf5;
  padding: 6px 10px;
  border-radius: 4px;
  color: #065f46;
}
.full-pre {
  white-space: pre-wrap;
  background: #f8fafc;
  padding: 10px;
  border-radius: 6px;
  margin: 0;
  color: #334155;
  font-family: inherit;
}

/* 🚨 区块链防篡改高危警告横幅 */
.tamper-warning-box {
  background: linear-gradient(135deg, #fff1f2 0%, #fee2e2 100%);
  border: 2px solid #ef4444;
  box-shadow: 0 4px 14px rgba(239, 68, 68, 0.25);
  border-radius: 10px;
  padding: 16px 20px;
  margin-bottom: 20px;
  animation: pulse-border 2s infinite ease-in-out;
}

@keyframes pulse-border {
  0% { box-shadow: 0 0 0 0 rgba(239, 68, 68, 0.4); }
  70% { box-shadow: 0 0 0 8px rgba(239, 68, 68, 0); }
  100% { box-shadow: 0 0 0 0 rgba(239, 68, 68, 0); }
}

.tw-head {
  display: flex;
  align-items: center;
  gap: 10px;
  color: #b91c1c;
  font-size: 16px;
  font-weight: 800;
  margin-bottom: 8px;
}

.tw-icon {
  font-size: 22px;
  animation: tw-bounce 1s infinite alternate;
}

@keyframes tw-bounce {
  from { transform: scale(1); }
  to { transform: scale(1.2); }
}

.tw-body {
  font-size: 13px;
  line-height: 1.6;
  color: #991b1b;
  margin-bottom: 12px;
}

.tw-hashes {
  background: rgba(255, 255, 255, 0.85);
  border: 1px solid #fca5a5;
  border-radius: 6px;
  padding: 10px 14px;
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.tw-h-item {
  display: flex;
  align-items: center;
  font-size: 12px;
  gap: 8px;
}

.tw-h-item.danger code {
  color: #b91c1c;
  background: #fee2e2;
  padding: 2px 6px;
  border-radius: 4px;
  font-family: monospace;
  font-weight: 600;
  word-break: break-all;
}

.tw-h-item.chain code {
  color: #047857;
  background: #d1fae5;
  padding: 2px 6px;
  border-radius: 4px;
  font-family: monospace;
  font-weight: 600;
  word-break: break-all;
}

.tw-h-item .lbl {
  font-weight: 600;
  min-width: 220px;
}

/* 🛡️ 校验通过安全横幅 */
.verified-safe-box {
  background: #ecfdf5;
  border: 1px solid #a7f3d0;
  color: #065f46;
  border-radius: 8px;
  padding: 10px 16px;
  margin-bottom: 18px;
  display: flex;
  align-items: center;
  gap: 10px;
  font-size: 13px;
}

.tamper-tag-glow {
  animation: tag-glow 1.5s infinite alternate;
  font-weight: 700;
}

@keyframes tag-glow {
  from { opacity: 0.85; transform: scale(0.98); }
  to { opacity: 1; transform: scale(1.05); }
}
</style>
