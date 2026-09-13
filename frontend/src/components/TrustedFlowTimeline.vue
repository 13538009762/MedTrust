<template>
  <div class="trusted-timeline-container">
    <div class="timeline-header">
      <div class="th-left">
        <span class="th-badge">⛓️ 联盟链全生命周期存证</span>
        <h3 class="th-title">医疗数据可信流转与密码学生命周期时间线 (14 关键节点)</h3>
      </div>
      <div class="th-meta" v-if="record">
        <span class="th-tag">病历单号: <strong>{{ record.record_no || 'ENC-DEMO' }}</strong></span>
        <span class="th-tag">患者: <strong>{{ record.patient_name || '患者' }}</strong></span>
        <span class="th-tag">机构: <strong>{{ record.hospital_name || '医疗机构' }}</strong></span>
      </div>
    </div>

    <div class="timeline-steps-grid">
      <div
        v-for="step in steps"
        :key="step.index"
        class="step-item-card"
        :class="{
          'step-completed': step.status === 'COMPLETED',
          'step-active': step.status === 'ACTIVE',
          'step-warning': step.status === 'WARNING',
        }"
      >
        <div class="step-card-header">
          <div class="step-index-badge">{{ String(step.index).padStart(2, '0') }}</div>
          <div class="step-title-wrap">
            <span class="step-name">{{ step.title }}</span>
            <span class="step-role-tag">{{ step.role }}</span>
          </div>
          <el-tag
            size="small"
            :type="step.status === 'COMPLETED' ? 'success' : (step.status === 'ACTIVE' ? 'primary' : 'info')"
            effect="dark"
            class="step-status-tag"
          >
            {{ step.statusText }}
          </el-tag>
        </div>

        <p class="step-desc">{{ step.desc }}</p>

        <div class="step-details-box" v-if="step.details && step.details.length">
          <div v-for="(d, di) in step.details" :key="di" class="detail-row">
            <span class="d-label">{{ d.label }}:</span>
            <code class="d-val" :title="d.val">{{ d.val }}</code>
          </div>
        </div>

        <div class="step-footer">
          <span class="sf-tech">🔒 {{ step.techSpec }}</span>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'

const props = defineProps<{
  record?: any
  tampered?: boolean
}>()

const steps = computed(() => {
  const r = props.record || {}
  const isTampered = props.tampered || r.is_tampered

  return [
    {
      index: 1,
      title: '门诊挂号建档',
      role: '患者/分诊台',
      status: 'COMPLETED',
      statusText: '已建档',
      desc: '患者实名认证建档，生成全网唯一规范临床就诊业务流水号。',
      techSpec: 'UUIDv4 + 时间戳前缀',
      details: [
        { label: '流水单号', val: r.record_no || 'ENC2026031201' },
        { label: '建档时间', val: r.created_at ? String(r.created_at).slice(0, 19) : '2026-03-12 09:15:00' },
      ]
    },
    {
      index: 2,
      title: '责任医师接诊',
      role: '经治门诊医生',
      status: 'COMPLETED',
      statusText: '已接诊',
      desc: '专科医生调阅既往档案，核实临床主诉症状与发病病程。',
      techSpec: 'RBAC 角色认证 + JWT 会话令牌',
      details: [
        { label: '接诊医师', val: r.doctor_name || '李建国 主任医师' },
        { label: '接诊科室', val: r.department_name || '心血管内科' },
      ]
    },
    {
      index: 3,
      title: 'SOAP 病历采集与体征',
      role: '责任医生/护士',
      status: 'COMPLETED',
      statusText: '已核验',
      desc: '规范采集主观病史 (S)、客观查体 (O) 与生命体征指标。',
      techSpec: '结构化医学标准字段校验',
      details: [
        { label: '基础体征', val: r.vital_signs || '血压 145/95mmHg, 脉搏 88次/分' },
        { label: '主诉症状', val: r.chief_complaint || r.symptoms || '胸骨后剧烈绞痛3小时' },
      ]
    },
    {
      index: 4,
      title: '开具医技辅助检查',
      role: '经治医生',
      status: 'COMPLETED',
      statusText: '已开立',
      desc: '根据初步拟诊下达检验化验或医学影像申请单。',
      techSpec: '检查流水号 ORD 编码生成',
      details: [
        { label: '辅助申请', val: r.exam_items || (r.need_exam ? '急诊心电图, 肌钙蛋白I, 心脏超声' : '常规临床检查') },
        { label: '初步拟诊', val: r.initial_diagnosis || '急性前壁心肌梗死待查' },
      ]
    },
    {
      index: 5,
      title: '医技科室接收与执行',
      role: '检验/影像技师',
      status: 'COMPLETED',
      statusText: '已执行',
      desc: '标本实验室化验或 PACS 影像扫描上机检查。',
      techSpec: '分布式多院区医技系统协同',
      details: [
        { label: '出具人员', val: r.exam_doctor || '检验主管技师 / 影像科医生' },
        { label: '执行时间', val: r.exam_time || '2026-03-12 09:35:00' },
      ]
    },
    {
      index: 6,
      title: '医技报告出具与归档',
      role: '医技科室',
      status: 'COMPLETED',
      statusText: '已回传',
      desc: '客观检查结果与影像结论回传并绑定至主就诊记录。',
      techSpec: '医技数据跨科室自动化流转',
      details: [
        { label: '检查结论', val: (r.exam_result && r.exam_result.length > 40) ? (r.exam_result.slice(0, 38) + '...') : (r.exam_result || 'V1-V4导联ST段抬高, 肌钙蛋白阳性') },
      ]
    },
    {
      index: 7,
      title: '最终确诊与处置方案',
      role: '经治医生',
      status: 'COMPLETED',
      statusText: '已确诊',
      desc: '责任医师结合检查结论做出明确评估 (A) 与处置医嘱 (P)。',
      techSpec: '电子病历归档闭环',
      details: [
        { label: '最终确诊', val: r.diagnosis || '急性ST段抬高型前壁心肌梗死' },
        { label: '治疗方案', val: (r.treatment_plan && r.treatment_plan.length > 40) ? (r.treatment_plan.slice(0, 38) + '...') : (r.treatment_plan || '急诊介入 PCI 手术与监护治疗') },
      ]
    },
    {
      index: 8,
      title: '规范 PDF 1.4 文档生成',
      role: '服务端引擎',
      status: 'COMPLETED',
      statusText: '已生成',
      desc: '自动化编译生成符合国家标准的合法二进制 PDF 1.4 凭证。',
      techSpec: '标准 PDF 1.4 + UniGB-UTF16-H 矢量字体',
      details: [
        { label: '文档格式', val: 'PDF-1.4 (符合 ISO 32000-1 国际标准)' },
        { label: '防伪签名', val: (r.doctor_name || '经治责任医师') + ' (CA电子签章)' },
      ]
    },
    {
      index: 9,
      title: '流式 AES-256-GCM 密文加密',
      role: '密码学中间件',
      status: 'COMPLETED',
      statusText: '密文就绪',
      desc: '对称密钥绑定 AAD (患者ID:单号) 加密，磁盘零明文落地。',
      techSpec: '国密/国际双密算法 (AES-GCM / SM4-GCM)',
      details: [
        { label: '加密模式', val: 'AES-256-GCM (认证加密流)' },
        { label: '附加认证', val: `PatientID:${r.patient_id || 4}:${r.record_no || 'ENC2026'}` },
      ]
    },
    {
      index: 10,
      title: 'IPFS 分布式节点托管',
      role: 'IPFS 守护节点',
      status: 'COMPLETED',
      statusText: '已上链寻址',
      desc: '病历密文切片托管于分布式存储网络，生成唯一内容寻址 CID。',
      techSpec: 'InterPlanetary File System (Kademlia DHT)',
      details: [
        { label: 'IPFS CID', val: (r.files && r.files[0]?.ipfs_cid) || 'QmYwAPJzv5CZsnA625s3Xf2nemtYgPpHdWEz79ojWnPbdG' },
      ]
    },
    {
      index: 11,
      title: '计算双维密码学指纹',
      role: '哈希引擎',
      status: 'COMPLETED',
      statusText: '哈希锁定',
      desc: '同时分离计算原始文件 FileHash 与结构化临床 Merkle 摘要。',
      techSpec: 'SHA-256 多维临床 Merkle Tree 构造',
      details: [
        { label: 'FileHash', val: (r.files && r.files[0]?.file_hash) || 'e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855' },
        { label: 'ClinicalHash', val: r.chain_hash || r.current_hash || 'ca978112ca1bbdcafac231b39a23dc4da78608144160670c3547f480e6085a86' },
      ]
    },
    {
      index: 12,
      title: 'Fabric 智能合约存证出块',
      role: 'Fabric Gateway',
      status: 'COMPLETED',
      statusText: '区块固化',
      desc: '提交交易至 medchannel 通道，Org1/Org2 节点背书并出块。',
      techSpec: 'Hyperledger Fabric 2.5 + Raft 共识',
      details: [
        { label: 'Fabric TxID', val: r.fabric_tx_id || '0x9f8e7d6c5b4a3210efedcba0123456789abcdef0123456789abcdef012345678' },
        { label: '区块高度', val: `#${r.block_height || 108} (medchannel 通道)` },
      ]
    },
    {
      index: 13,
      title: '跨院调阅零信任网关控制',
      role: 'AccessEngine',
      status: 'COMPLETED',
      statusText: '安全受控',
      desc: '跨机构调阅执行基于规则加权评分引擎 (RULE_ENGINE_WEIGHTED)。',
      techSpec: 'RBAC 角色 + ABAC 属性 + 规则加权风控网关',
      details: [
        { label: '决策机制', val: '患者授权 / 现场调阅密钥 / Break-Glass 破窗' },
        { label: '风控策略', val: 'RULE_ENGINE_WEIGHTED (多维规则加权)' },
      ]
    },
    {
      index: 14,
      title: '动态闭环验真与防篡改审计',
      role: 'Supervisor 监管',
      status: isTampered ? 'WARNING' : 'COMPLETED',
      statusText: isTampered ? '🚨 篡改拦截' : '🛡️ 100% 吻合',
      desc: isTampered
        ? '动态核验检测到数据库被非法修改，与 Fabric 链上固化指纹不匹配，系统即刻红标报警并阻断采信！'
        : '实时解密比对本地多维指纹与 Fabric 账本原始指纹，100% 一致，存证链闭环真实可信。',
      techSpec: '三阶段防篡改核验 (附件指纹 + 临床摘要 + 链上背书)',
      details: [
        { label: '验真判定', val: isTampered ? '🚨 TAMPER_ALERT (哈希突变)' : '✅ VERIFIED (真实完整)' },
        { label: '不可篡改', val: 'Fabric 账本不可逆，MySQL 篡改即刻显形' },
      ]
    },
  ]
})
</script>

<style scoped>
.trusted-timeline-container {
  background: #ffffff;
  border-radius: 12px;
  border: 1px solid #e2e8f0;
  padding: 20px;
  box-shadow: 0 4px 16px rgba(15, 23, 42, 0.04);
}

.timeline-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  border-bottom: 1px solid #f1f5f9;
  padding-bottom: 14px;
  margin-bottom: 18px;
  flex-wrap: wrap;
  gap: 10px;
}

.th-badge {
  display: inline-block;
  background: #eff6ff;
  color: #1d4ed8;
  font-size: 12px;
  font-weight: 600;
  padding: 2px 8px;
  border-radius: 4px;
  margin-bottom: 4px;
}

.th-title {
  margin: 0;
  font-size: 16px;
  font-weight: 700;
  color: #0f172a;
}

.th-meta {
  display: flex;
  gap: 8px;
  flex-wrap: wrap;
}

.th-tag {
  background: #f8fafc;
  border: 1px solid #e2e8f0;
  padding: 3px 8px;
  border-radius: 6px;
  font-size: 12px;
  color: #475569;
}

.th-tag strong {
  color: #1e293b;
}

.timeline-steps-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(320px, 1fr));
  gap: 14px;
}

.step-item-card {
  background: #f8fafc;
  border: 1px solid #e2e8f0;
  border-radius: 8px;
  padding: 12px 14px;
  display: flex;
  flex-direction: column;
  justify-content: space-between;
  transition: all 0.2s ease;
}

.step-item-card:hover {
  transform: translateY(-2px);
  box-shadow: 0 6px 16px rgba(0, 0, 0, 0.06);
  border-color: #cbd5e1;
}

.step-completed {
  border-left: 4px solid #10b981;
}

.step-active {
  border-left: 4px solid #3b82f6;
  background: #eff6ff;
}

.step-warning {
  border-left: 4px solid #ef4444;
  background: #fef2f2;
}

.step-card-header {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 8px;
}

.step-index-badge {
  background: #0f172a;
  color: #ffffff;
  font-weight: 800;
  font-size: 11px;
  width: 24px;
  height: 24px;
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
}

.step-completed .step-index-badge {
  background: #059669;
}

.step-warning .step-index-badge {
  background: #dc2626;
}

.step-title-wrap {
  flex: 1;
  display: flex;
  align-items: baseline;
  gap: 6px;
  overflow: hidden;
}

.step-name {
  font-size: 13px;
  font-weight: 700;
  color: #1e293b;
  white-space: nowrap;
}

.step-role-tag {
  font-size: 11px;
  color: #64748b;
  background: #e2e8f0;
  padding: 1px 5px;
  border-radius: 3px;
}

.step-desc {
  font-size: 12px;
  color: #475569;
  line-height: 1.45;
  margin: 0 0 8px 0;
}

.step-details-box {
  background: #ffffff;
  border: 1px dashed #cbd5e1;
  border-radius: 6px;
  padding: 6px 8px;
  margin-bottom: 8px;
  font-size: 11px;
}

.detail-row {
  display: flex;
  justify-content: space-between;
  gap: 8px;
  margin-bottom: 3px;
}

.detail-row:last-child {
  margin-bottom: 0;
}

.d-label {
  color: #64748b;
  flex-shrink: 0;
}

.d-val {
  color: #0f172a;
  font-family: monospace;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  max-width: 200px;
}

.step-footer {
  border-top: 1px solid #e2e8f0;
  padding-top: 6px;
}

.sf-tech {
  font-size: 11px;
  color: #0284c7;
  font-weight: 500;
}
</style>
