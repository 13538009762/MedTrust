<template>
  <!-- PDF 电子病历规范预览弹窗 (WP-16: 从两视图几乎完全相同的 PDF 预览弹窗合并抽取) -->
  <el-dialog
    :model-value="visible"
    width="min(1040px, 94vw)"
    top="2vh"
    :close-on-click-modal="false"
    class="pdf-preview-dialog"
    @update:model-value="$emit('update:visible', $event)"
  >
    <template #header>
      <div class="pdf-dialog-header">
        <div class="pdf-header-title">
          <el-icon class="pdf-header-icon"><Document /></el-icon>
          <span class="pdf-header-text">临床就诊电子病历归档凭证</span>
          <el-tag v-if="record?.is_tampered" size="small" type="danger" effect="dark" class="tamper-tag-glow">篡改告警 · 校验失败</el-tag>
          <el-tag v-else size="small" type="primary" effect="plain" class="pdf-header-tag">PDF 规范视图</el-tag>
        </div>
        <div class="pdf-header-actions no-print">
          <el-button
            type="primary"
            size="small"
            :icon="Printer"
            class="pdf-action-btn pdf-btn-print"
            @click="printPdfSheet"
          >
            打印 / 另存为 PDF 文件
          </el-button>
          <el-button
            type="success"
            plain
            size="small"
            :icon="Download"
            class="pdf-action-btn pdf-btn-download"
            @click="$emit('download-record', record!.id)"
          >
            下载规范归档 PDF
          </el-button>
          <el-button
            type="warning"
            plain
            size="small"
            :icon="CircleCheckFilled"
            class="pdf-action-btn pdf-btn-verify"
            @click="$emit('verify-integrity', record!.id)"
          >
            链上完整性核验
          </el-button>
        </div>
      </div>
    </template>

    <div v-if="record" id="emr-print-container" class="emr-sheet-wrapper">
      <!-- 存证规范与安全防护状态栏 -->
      <div class="emr-action-bar no-print">
        <div class="emr-tip-tag">
          <el-tag v-if="record?.is_tampered" type="danger" effect="dark" class="action-tag tamper-tag-glow">
            底层数据已被篡改 · 链上存证核验失败 (不可采信)
          </el-tag>
          <el-tag v-else type="success" effect="dark" class="action-tag">密文解密验证通过</el-tag>
          <el-tag type="info" class="action-tag">国家卫健委《电子病历应用规范》甲级存证标准</el-tag>
        </div>
        <div v-if="record?.infection_alert?.has_risk" class="emr-infection-slot">
          <InfectionSafetyAlert :alert="record.infection_alert" mode="badge" />
        </div>
      </div>

      <!-- 正式红头病历文档纸张 -->
      <div class="emr-sheet-paper">
        <!-- 右上角国家卫健委医护安全防护警示水印/标识 -->
        <div v-if="record.infection_alert?.has_risk" class="emr-infection-watermark">
          【国家卫健委医护职业安全重点警示：高危传染病携带者 · 接诊请穿戴双层手套与防护装备】
        </div>
        <!-- 红头医院名称与标题 -->
        <div class="emr-header">
          <div class="emr-hospital-name">{{ record.hospital_name || formatHospName(record.hospital_id) }}</div>
          <div class="emr-doc-title">门 诊 / 临 床 就 诊 归 档 记 录 单</div>
          <div class="emr-sub-title">（MedTrust 医疗可信共享联盟·Hyperledger Fabric 区块链存证凭证）</div>
          <div class="emr-red-line-double">
            <div class="line-thick"></div>
            <div class="line-thin"></div>
          </div>
        </div>

        <!-- 条形码与凭证流水号 -->
        <div class="emr-meta-banner">
          <div class="meta-left">
            <div>就诊编号：<span class="mono bold">{{ record.record_no }}</span></div>
            <div>就诊科室：{{ record.department_name }}</div>
            <div>就诊类型：{{ formatEncounterType(record.encounter_type) }}</div>
          </div>
          <div class="meta-barcode-box">
            <svg class="barcode-svg" :viewBox="`0 0 ${getBarcodeData(record.record_no).totalWidth} 34`" height="34" preserveAspectRatio="none">
              <rect
                v-for="(bar, idx) in getBarcodeData(record.record_no).bars"
                :key="idx"
                :x="bar.x"
                y="0"
                :width="bar.w"
                height="34"
                fill="#0f172a"
              />
            </svg>
            <div class="barcode-text mono">{{ record.record_no }}</div>
          </div>
          <div class="meta-right">
            <div>就诊时间：{{ formatTime(record.created_at) }}</div>
            <div>责任医师：<strong>{{ record.doctor_name }}</strong></div>
            <div>归档状态：
              <el-tag v-if="record.is_tampered" type="danger" size="small">篡改告警</el-tag>
              <el-tag v-else-if="record.fabric_tx_id" type="success" size="small">已上链存证</el-tag>
              <el-tag v-else type="info" size="small">待同步上链</el-tag>
            </div>
          </div>
        </div>

        <!-- 患者基本信息表格 -->
        <table class="emr-patient-table">
          <tr>
            <th width="90">患者姓名</th>
            <td width="150"><strong>{{ record.patient_name || '—' }}</strong></td>
            <th width="90">身份证号</th>
            <td width="220"><span class="mono">{{ maskIDCard(record.patient_id_card) }}</span></td>
            <th width="80">联系电话</th>
            <td><span class="mono">{{ record.patient_phone || '未登记' }}</span></td>
          </tr>
        </table>

        <!-- 临床 SOAP 详细内容 -->
        <div class="emr-soap-section">
          <div class="soap-block">
            <div class="soap-title">【S - Subjective 主观病史采集】</div>
            <div class="soap-row"><strong>● 患者主诉：</strong>{{ record.chief_complaint || record.symptoms }}</div>
            <div class="soap-row"><strong>● 现病史：</strong>{{ record.present_illness || '患者因主诉症状就诊，发病过程如上所述。' }}</div>
            <div class="soap-row"><strong>● 发病时间：</strong>{{ formatOnsetTime(record.onsetTime || record.onset_time) }}<span v-if="record.duration">（病程/持续时间：{{ cleanDurationText(record.duration) }}）</span></div>
          </div>

          <div class="soap-block">
            <div class="soap-title">【O - Objective 客观检查与测量】</div>
            <div class="soap-row"><strong>● 基础生命体征：</strong><span class="vitals-highlight">{{ record.vital_signs || '生命体征平稳' }}</span></div>
            <div class="soap-row"><strong>● 辅助检查申请：</strong>{{ record.need_exam ? (record.exam_items || '已开具辅助检查') : '未开具医技检查' }}</div>
            <div v-if="record.exam_result" class="soap-exam-result-box">
              <div class="soap-exam-head">医技科室回传检查报告明细：</div>
              <pre class="soap-exam-content">{{ record.exam_result }}</pre>
              <div v-if="record.exam_doctor" class="soap-exam-foot">
                出具技师/医生：{{ record.exam_doctor }} | 时间：{{ record.exam_time }}
              </div>
            </div>
          </div>

          <div class="soap-block">
            <div class="soap-title">【A - Assessment 综合评估与确诊】</div>
            <div class="soap-row"><strong>● 临床初步诊断：</strong>{{ record.initial_diagnosis || '-' }} <span v-if="record.diagnostic_basis" class="text-xs text-gray-500">（依据：{{ record.diagnostic_basis }}）</span></div>
            <div class="soap-row final-diagnosis">
              <strong>● 经治医师最终确诊：</strong>
              <span class="font-bold text-success">{{ record.diagnosis }}</span>
            </div>
            <div class="soap-row">
              <strong>● 病理诱因与机制：</strong>{{ getDisplayEtiology(record) }}
            </div>
          </div>

          <div class="soap-block">
            <div class="soap-title">【P - Plan 综合处置与医嘱方案】</div>
            <div class="soap-row plan-box">
              <pre class="soap-plan-content">{{ record.treatment_plan || '遵常规临床医嘱随访' }}</pre>
            </div>
          </div>
        </div>

        <!-- 底端医师电子签名与区块链防伪存证图章 -->
        <div class="emr-footer-box">
          <div class="footer-crypto-evidence">
            <div><strong>Hyperledger Fabric TxID：</strong><code class="tx-hash">{{ formatTxId(record.fabric_tx_id) }}</code></div>
            <div class="crypto-sub-line">
              <span><strong>联盟链通道：</strong><span class="mono text-purple-700">medicalchannel</span></span>
              <span class="ml-2"><strong>智能合约：</strong><span class="mono text-indigo-700">medical_contract</span></span>
              <span class="ml-2"><strong>区块高度：</strong><span class="mono text-emerald-700 font-bold">{{ record.block_height ? `#${record.block_height}` : '待同步' }}</span></span>
            </div>
            <div><strong>底层加密算法：</strong><span class="mono text-slate-700">AES-256-GCM (AAD 关联认证数据绑定)</span></div>
            <div v-if="record.files && record.files.length">
              <strong>IPFS 分布式存证 CID：</strong><code class="mono" style="word-break: break-all;">{{ record.files[0].ipfs_cid || '—' }}</code>
            </div>
            <div v-if="record.files && record.files.length">
              <strong>原始数据 SHA-256 指纹：</strong><code class="mono" style="word-break: break-all;">{{ record.files[0].file_hash || '—' }}</code>
            </div>
            <div class="text-xs text-emerald-700 mt-1">本病历经 Hyperledger Fabric 真实存证，数字签名抗抵赖、防篡改</div>
          </div>

          <div class="footer-seal-area">
            <!-- 防伪公章样式 -->
            <div class="emr-stamp">
              <div class="stamp-inner">
                <div class="stamp-star">*</div>
                <div class="stamp-text-top">{{ record.hospital_name || '医疗机构' }}</div>
                <div class="stamp-text-center">区块链存证专章</div>
                <div class="stamp-text-bottom">链上存证核验凭证</div>
              </div>
            </div>

            <!-- 医师签字 -->
            <div class="doctor-signature-box">
              <span class="sig-lbl">经治医师签字：</span>
              <span class="sig-name">{{ record.doctor_name }} (印)</span>
              <div class="sig-time text-xs">{{ formatTime(record.created_at) }}</div>
            </div>
          </div>
        </div>
      </div>
    </div>
  </el-dialog>
</template>

<script setup lang="ts">
import { Document, Printer, Download, CircleCheckFilled } from '@element-plus/icons-vue'
import InfectionSafetyAlert from '@/components/InfectionSafetyAlert.vue'
import {
  formatEncounterType,
  formatTime,
  formatTxId,
  cleanDurationText,
  formatOnsetTime,
  getDisplayEtiology,
  getBarcodeData,
} from '@/utils/recordFormatters'
import type { MedicalRecordRow } from '@/types/clinical'

defineProps<{
  visible: boolean
  record: MedicalRecordRow | null
  formatHospName: (hId?: number) => string
  maskIDCard: (idCard?: string) => string
}>()

defineEmits<{
  (e: 'update:visible', val: boolean): void
  (e: 'download-record', recordId: number): void
  (e: 'verify-integrity', recordId: number): void
}>()

function printPdfSheet() {
  document.body.classList.add('printing-emr')
  const cleanup = () => {
    document.body.classList.remove('printing-emr')
    window.removeEventListener('afterprint', cleanup)
  }
  window.addEventListener('afterprint', cleanup)
  try {
    window.print()
  } finally {
    setTimeout(cleanup, 1000)
  }
}
</script>

