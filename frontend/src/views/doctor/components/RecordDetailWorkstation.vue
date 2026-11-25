<template>
  <!-- 跨院/院内解密放行全景病历工作台 DETAIL 主体 (WP-16)
       从 CrossQueryView.vue 与 DoctorRecordsView.vue 高度重复的 DETAIL 区段合并抽取。
       variant='cross' 保留跨院检索文案，variant='local' 保留门诊记录文案。 -->
  <div v-if="record" class="detail-workstation-content mt-3">
    <!-- 医护职业暴露安全防范条幅 -->
    <div v-if="record.infection_alert?.has_risk" class="mb-3">
      <InfectionSafetyAlert :alert="record.infection_alert" mode="corner-banner" />
    </div>

    <!-- 篡改告警横幅 (高危防护预警卡) -->
    <div v-if="record.is_tampered" class="tamper-box danger">
      <div class="tamper-top-badge-row">
        <div class="tamper-pill-badge">
          <el-icon class="pulse-icon"><WarningFilled /></el-icon>
          <span>BLOCKCHAIN INTEGRITY ALERT / 联盟链存证指纹不匹配</span>
        </div>
        <ChainVerifyBadge status="tampered" label="已触发临床采信阻断" />
      </div>

      <div class="t-head">
        <el-icon class="t-head-icon"><WarningFilled /></el-icon>
        <span class="t-title">【高危安全警报】{{ variant === 'cross' ? '检测到该病历底层数据已被恶意篡改！' : '底层 MySQL 数据库临床数据已被恶意篡改！' }}</span>
      </div>

      <div class="t-desc">
        系统零信任存证网关检测到底层 {{ variant === 'cross' ? '' : 'MySQL ' }}数据库中的临床就诊明细特征指纹与 Hyperledger Fabric 联盟链上不可篡改的存证指纹<strong>不匹配</strong>！该病历临床真实性已丧失，<strong>临床严禁直接采信该病历或据此开具医嘱</strong>！
      </div>

      <div class="t-hashes">
        <div class="hash-row danger-hash">
          <span class="hash-label">数据库当前计算指纹 (已污染):</span>
          <code class="hash-code">{{ record.current_hash || '（计算生成中...）' }}</code>
          <el-button size="small" link type="danger" @click.stop="$emit('copy-text', record.current_hash || '', '当前计算哈希')">复制</el-button>
        </div>
        <div class="hash-row chain-hash">
          <span class="hash-label">区块链不可篡改基准 (Fabric 账本):</span>
          <code class="hash-code">{{ record.chain_hash || '（链上存证缺失 / 未通过共识锚定）' }}</code>
          <el-button v-if="record.chain_hash" size="small" link type="primary" @click.stop="$emit('copy-text', record.chain_hash, '区块链基准哈希')">复制</el-button>
        </div>
      </div>

      <div class="tamper-footer-row">
        <div class="tamper-reason">
          <el-icon class="mr-1 text-red-600"><InfoFilled /></el-icon>
          <span>{{ record.tamper_reason || '【区块链安全告警】病历关键临床数据（诱因/用药/诊断）已被非法篡改，哈希指纹校验失败！' }}</span>
        </div>
        <div class="tamper-audit-pill">
          <span>违规调阅已实时上报卫健区块链监督中心</span>
        </div>
      </div>
    </div>
    <div v-else-if="record.verified" class="tamper-box safe">
      <div class="tb-safe-left">
        <el-icon class="text-emerald-600 text-lg"><CircleCheckFilled /></el-icon>
        <div class="tb-text">
          <span class="font-bold text-emerald-800 text-sm">区块链全量防篡改核验通过：</span>
          <span class="text-emerald-700 text-xs ml-1.5">链下数据指纹与 Hyperledger Fabric 联盟链完全吻合，数据完整未篡改</span>
        </div>
      </div>
      <div class="tb-safe-right">
        <ChainVerifyBadge status="verified" :label="variant === 'cross' ? '跨院核验一致' : 'Fabric 共识已核验'" />
      </div>
    </div>
    <div v-else class="tamper-box pending">
      <div class="tb-safe-left">
        <el-icon class="text-amber-600 text-lg"><InfoFilled /></el-icon>
        <div class="tb-text">
          <span class="font-bold text-amber-800 text-sm">区块链核验状态提示：</span>
          <span class="text-amber-700 text-xs ml-1.5">{{ record.tamper_reason || '区块链网关离线或存证尚未完成，暂未执行共识比对' }}</span>
        </div>
      </div>
      <div class="tb-safe-right">
        <ChainVerifyBadge status="pending" label="待复核" />
      </div>
    </div>

    <el-alert
      v-if="record.is_tampered"
      title="【安全风控介入】虽然该病历具备调阅权限，但由于底层数据库指纹与区块链存证不匹配，系统已锁定为只读存证审计模式，临床操作已被安全防护策略阻断！"
      type="error"
      show-icon
      class="mb-3 mt-3"
      :closable="false"
    />
    <template v-else>
      <el-alert
        v-if="isBreakGlassRelease || record.access_type === 'BREAK_GLASS'"
        title="本次调阅为 Break-Glass 紧急破窗放行（24小时急救绿色通道生效中），事件单已上链存证并向卫健监管派发预警"
        type="error"
        show-icon
        class="mb-3 mt-3"
        :closable="false"
      />
      <el-alert
        v-else-if="record.doctor_id === userId"
        title="本人开具病历：您作为接诊责任医生依法享有持续随诊与病史调阅权限，系统免申请直接放行"
        type="success"
        show-icon
        class="mb-3 mt-3"
        :closable="false"
      />
      <el-alert
        v-else-if="record.hospital_id === userHospitalId"
        title="院内诊疗互通：目标病历归属本医疗机构，遵循院内就诊连续性互认规范直接放行"
        type="info"
        show-icon
        class="mb-3 mt-3"
        :closable="false"
      />
      <el-alert
        v-else
        title="跨机构授权放行：目标机构已核准知情同意放行策略，数据经跨院零信任安全网关实时解密"
        type="success"
        show-icon
        class="mb-3 mt-3"
        :closable="false"
      />
    </template>

    <div class="ws-body-grid">
      <!-- 左侧栏：跨院档案属性、区块链凭据、附件文件 (380px) -->
      <div class="ws-col-left">
        <!-- 卡片 1：就诊档案与患者卡片 -->
        <div class="bento-card patient-hero-card">
          <div class="ph-header">
            <div class="ph-avatar">
              {{ formatPatientName(record).slice(0, 1) }}
            </div>
            <div class="ph-main-info">
              <div class="ph-name-row">
                <span class="ph-name">{{ formatPatientName(record) }}</span>
                <el-tag size="small" :type="record.patient_gender === 'FEMALE' ? 'danger' : 'primary'" effect="plain">
                  {{ record.patient_gender === 'FEMALE' ? '女' : (record.patient_gender === 'MALE' ? '男' : (record.gender || '患者')) }}
                </el-tag>
                <span v-if="record.patient_age || record.age" class="ph-age">{{ record.patient_age || record.age }} 岁</span>
              </div>
              <div class="ph-meta-id mono text-xs text-gray-500">
                ID: {{ maskIDCard(record.patient_id_card) }}
              </div>
            </div>
          </div>

          <div class="ph-divider" />

          <div class="ph-details-grid">
            <div class="ph-item">
              <span class="lbl">就诊单号</span>
              <span class="val mono font-semibold">{{ record.record_no }}</span>
            </div>
            <div class="ph-item">
              <span class="lbl">就诊类型</span>
              <span class="val"><el-tag size="small" type="warning" effect="light">{{ formatEncounterType(record.encounter_type) }}</el-tag></span>
            </div>
            <div class="ph-item">
              <span class="lbl">开立机构</span>
              <span class="val font-semibold text-slate-800">{{ record.hospital_name || formatHospName(record.hospital_id) }}</span>
            </div>
            <div class="ph-item">
              <span class="lbl">就诊科室</span>
              <span class="val font-semibold text-slate-800">{{ record.department_name }}</span>
            </div>
            <div class="ph-item">
              <span class="lbl">接诊医师</span>
              <span class="val font-semibold text-primary">{{ record.doctor_name || '李建国医生' }}</span>
            </div>
            <div class="ph-item">
              <span class="lbl">流转状态</span>
              <span class="val"><el-tag size="small" type="success" effect="light">{{ formatRecordStatus(record.status) }}</el-tag></span>
            </div>
            <div class="ph-item">
              <span class="lbl">就诊时间</span>
              <span class="val text-xs text-slate-600">{{ formatTime(record.created_at) }}</span>
            </div>
          </div>
        </div>

        <!-- 卡片 2：区块链存证与信封加密数字证书 (联盟链共识确权凭据) -->
        <div class="bento-card crypto-card digital-cert-card">
          <div class="crypto-header">
            <div class="crypto-title">
              <span class="crypto-dot" />
              <span>区块链存证与信封加密凭证 (数字证书)</span>
            </div>
            <el-tag size="small" type="success" effect="dark" class="crypto-tag">联盟链共识已上链</el-tag>
          </div>

          <div class="crypto-body">
            <!-- 信封加密与密码学存证凭证表单 -->
            <div class="cert-fields-grid">
              <div class="cert-item">
                <span class="cert-k">对称加密算法：</span>
                <span class="cert-v font-bold text-emerald-600">AES-256-GCM (AAD)</span>
              </div>
              <div class="cert-item">
                <span class="cert-k">加密数据密钥 (DEK)：</span>
                <HashCapsule :value="record.encrypted_dek" :prefix-len="8" :suffix-len="8" label="加密密钥 DEK" empty-text="待同步" />
              </div>
              <div class="cert-item">
                <span class="cert-k">初始向量 (IV/Nonce)：</span>
                <span class="cert-v mono text-xs">{{ record.iv_nonce ? `${record.iv_nonce} (96-bit)` : '—' }}</span>
              </div>
              <div class="cert-item">
                <span class="cert-k">IPFS 寻址 CID：</span>
                <HashCapsule :value="getRecordCID(record)" :prefix-len="10" :suffix-len="8" label="IPFS CID" empty-text="—" />
              </div>
              <div class="cert-item">
                <span class="cert-k">数据指纹 (FileHash)：</span>
                <HashCapsule :value="getRecordFileHash(record)" :prefix-len="10" :suffix-len="8" label="FileHash" empty-text="—" />
              </div>
              <div class="cert-item">
                <span class="cert-k">联盟通道 / 智能合约：</span>
                <span class="cert-v font-semibold text-purple-700">medicalchannel / medical_contract</span>
              </div>
            </div>

            <div class="crypto-field mt-3">
              <div class="cf-head">
                <span class="cf-label">Fabric 联盟链交易哈希 (TxID)</span>
              </div>
              <div class="cf-hash-box mono">
                <HashCapsule :value="record.fabric_tx_id" :prefix-len="14" :suffix-len="12" label="交易哈希" empty-text="待同步" />
              </div>
            </div>

            <div class="crypto-stats-row mt-2">
              <div class="stat-pill">
                <span class="stat-k">区块高度</span>
                <span class="stat-v mono">{{ record.block_height ? `#${record.block_height}` : '待同步' }}</span>
              </div>
              <div class="stat-pill">
                <span class="stat-k">共识状态</span>
                <span class="stat-v">Raft (etcdraft) 已确认</span>
              </div>
              <div class="stat-pill">
                <span class="stat-k">存证通道</span>
                <span class="stat-v">medicalchannel</span>
              </div>
            </div>

            <div class="crypto-security-seal mt-2">
              <el-icon class="mr-1 text-emerald-400"><CircleCheckFilled /></el-icon>
              <span>{{ variant === 'cross' ? 'AES-256-GCM 信封加密 · IPFS 分布式寻址 · Fabric 账本防篡改' : 'AES-256-GCM 本地内存加密 + IPFS 分布式存证 + 智能合约存证' }}</span>
            </div>
          </div>
        </div>

        <!-- 卡片 3：临床归档电子凭据与附件 -->
        <MedicalRecordUploadDialog
          :record="record"
          :token="token"
          @view-image="$emit('view-image', $event)"
          @view-pdf="$emit('view-pdf', $event)"
          @download-image="$emit('download-image', $event)"
          @download-record="$emit('download-record', $event)"
          @copy-text="(t: string, l: string) => $emit('copy-text', t, l)"
        />
      </div>
      <!-- 右侧栏：结构化临床 SOAP Bento 全景面板 (flex: 1) -->
      <div class="ws-col-main">
        <!-- 维度 1: [S] 主观病史与主诉 (Subjective) -->
        <div class="bento-card soap-bento-card s-dimension">
          <div class="soap-card-head">
            <div class="soap-head-title">
              <span class="soap-dim-badge s-badge">S</span>
              <span class="soap-title-text">主观病史与患者主诉 (Subjective)</span>
            </div>
            <div class="soap-time-chips">
              <span v-if="record.onset_time" class="time-chip">
                发病时间: {{ record.onset_time }}
              </span>
              <span v-if="record.duration" class="time-chip">
                持续: {{ record.duration }}
              </span>
            </div>
          </div>

          <div class="soap-card-content">
            <div class="complaint-hero-box">
              <div class="chb-quote-mark">“</div>
              <div class="chb-text">
                {{ record.chief_complaint || record.symptoms || '患者就诊主诉' }}
              </div>
            </div>

            <div v-if="record.present_illness" class="present-illness-box mt-3">
              <div class="pib-label">现病史演进 (History of Present Illness)：</div>
              <div class="pib-content">{{ record.present_illness }}</div>
            </div>
          </div>
        </div>

        <!-- 维度 2: [O] 客观体征与医技检查 (Objective) -->
        <div class="bento-card soap-bento-card o-dimension">
          <div class="soap-card-head">
            <div class="soap-head-title">
              <span class="soap-dim-badge o-badge">O</span>
              <span class="soap-title-text">客观检查与生命体征 (Objective)</span>
            </div>
            <el-tag size="small" type="success" effect="light">实时临床体征指标</el-tag>
          </div>

          <div class="soap-card-content">
            <!-- 4 项生命体征紧凑临床条 (HIS 单行四栏，基于真实数值动态判定) -->
            <div class="vitals-sensor-grid vitals-strip-compact">
              <div class="vital-sensor-card compact-vital-item">
                <span class="vsc-name">体温 (T)</span>
                <span class="vsc-val mono">{{ parsedVitals.temperature || '未录入' }}</span>
                <span :class="['vsc-status', tempVitalBadge.level]">{{ tempVitalBadge.shortLabel }}</span>
              </div>

              <div class="vital-sensor-card compact-vital-item">
                <span class="vsc-name">血压 (BP)</span>
                <span class="vsc-val mono">{{ parsedVitals.blood_pressure || '未录入' }}</span>
                <span :class="['vsc-status', bpVitalBadge.level]">{{ bpVitalBadge.shortLabel }}</span>
              </div>

              <div class="vital-sensor-card compact-vital-item">
                <span class="vsc-name">心率 (HR)</span>
                <span class="vsc-val mono">{{ parsedVitals.heart_rate || '未录入' }}</span>
                <span :class="['vsc-status', hrVitalBadge.level]">{{ hrVitalBadge.shortLabel }}</span>
              </div>

              <div class="vital-sensor-card compact-vital-item">
                <span class="vsc-name">血氧 (SpO2)</span>
                <span class="vsc-val mono">{{ parsedVitals.spo2 || '未录入' }}</span>
                <span :class="['vsc-status', spo2VitalBadge.level]">{{ spo2VitalBadge.shortLabel }}</span>
              </div>
            </div>

            <!-- 医技辅助检查回传报告板块 -->
            <div class="exam-report-bento-section mt-3">
              <div class="erb-header">
                <div class="erb-title">
                  <el-icon color="#0284c7"><OfficeBuilding /></el-icon>
                  <span>LIS / PACS 医技检验与影像检查报告</span>
                </div>
                <span v-if="record.exam_items" class="erb-items-tag">
                  申请项目：{{ record.exam_items }}
                </span>
              </div>

              <div v-if="record.exam_result" class="erb-result-box">
                <div class="erb-report-text pre-wrap">{{ record.exam_result }}</div>
                <div v-if="record.exam_doctor" class="erb-footer">
                  <span>报告出具医师/技师：<strong>{{ record.exam_doctor }}</strong></span>
                  <span v-if="record.exam_time" class="ml-3">检验时间：{{ record.exam_time }}</span>
                </div>
              </div>
              <div v-else-if="record.need_exam || record.status === 'WAITING_EXAM' || record.status === 'PROCESSING_EXAM'" class="erb-empty-box">
                <el-icon color="#d97706" class="text-base mr-1"><InfoFilled /></el-icon>
                <span>医技检查申请已派发至检验/影像中心，正在等待 LIS/PACS 回传检查报告{{ record.exam_items ? `（申请项目：${record.exam_items}）` : '' }}。</span>
              </div>
              <div v-else class="erb-empty-box">
                <el-icon color="#10b981" class="text-base mr-1"><CircleCheckFilled /></el-icon>
                <span>{{ variant === 'cross' ? '跨院就诊专科体征明确，未开立外送医技辅助检验，医生综合体格检查直接确诊。' : '门诊专科体征典型，临床未开具外送医技辅助检验，经治医师综合体格检查直接确诊。' }}</span>
              </div>
            </div>
          </div>
        </div>

        <!-- 维度 3: [A] 综合评估与临床确诊 (Assessment) -->
        <div class="bento-card soap-bento-card a-dimension">
          <div class="soap-card-head">
            <div class="soap-head-title">
              <span class="soap-dim-badge a-badge">A</span>
              <span class="soap-title-text">综合临床评估与最终确诊 (Assessment)</span>
            </div>
            <el-tag size="small" type="primary" effect="light">ICD-10 规范确诊</el-tag>
          </div>

          <div class="soap-card-content">
            <!-- 确诊 Hero 区域 -->
            <div class="diagnosis-hero-banner">
              <div class="dhb-left">
                <div class="dhb-label">经治医师最终确诊 (Final Diagnosis)</div>
                <div class="dhb-name">{{ record.diagnosis || '待确诊' }}</div>
                <div v-if="record.initial_diagnosis" class="dhb-compare">
                  <span class="text-xs text-gray-500">初诊拟诊：</span>
                  <span class="text-xs font-semibold text-slate-700">{{ record.initial_diagnosis }}</span>
                  <span v-if="record.diagnostic_basis" class="text-xs text-gray-400 ml-2">（依据：{{ record.diagnostic_basis }}）</span>
                </div>
              </div>
              <div class="dhb-right">
                <div v-if="record.diagnosis && record.status !== 'WAITING_EXAM' && record.status !== 'PROCESSING_EXAM'" class="dhb-stamp-badge">
                  <el-icon class="mr-1 text-emerald-600"><CircleCheckFilled /></el-icon>
                  <span>确诊生效</span>
                </div>
                <el-tag v-else size="small" type="warning" effect="light">待检查后终审确诊</el-tag>
              </div>
            </div>

            <!-- 病理诱因与机制分析 -->
            <div v-if="record.etiology" class="etiology-bento-box mt-3">
              <div class="ebb-title">病理诱因与发病机制分析 (Pathogenesis & Etiology)：</div>
              <div class="ebb-content">{{ record.etiology }}</div>
            </div>
          </div>
        </div>

        <!-- 维度 4: [P] 处置、处方与综合医嘱 (Plan) -->
        <div class="bento-card soap-bento-card p-dimension">
          <div class="soap-card-head">
            <div class="soap-head-title">
              <span class="soap-dim-badge p-badge">P</span>
              <span class="soap-title-text">处方用药、临床处置与综合医嘱 (Plan)</span>
            </div>
            <el-tag v-if="parsePlanItems(record.treatment_plan).length > 0" size="small" type="warning" effect="light">已签发执行</el-tag>
            <el-tag v-else size="small" type="info" effect="light">待终审开具</el-tag>
          </div>

          <div class="soap-card-content">
            <!-- 结构化处方与用药清单 -->
            <div class="rx-orders-container">
              <div class="rx-header">
                <span class="rx-symbol">℞</span>
                <span class="rx-title">处方用药与临床方案明细</span>
              </div>

              <div v-if="parsePlanItems(record.treatment_plan).length > 0" class="rx-list">
                <div
                  v-for="item in parsePlanItems(record.treatment_plan)"
                  :key="item.id"
                  class="rx-item-card"
                  :class="{ 'is-medicine': item.isMed }"
                >
                  <div class="ric-num">{{ item.id }}</div>
                  <div class="ric-content">
                    <div class="ric-text">{{ item.text }}</div>
                  </div>
                  <div class="ric-badge">
                    <el-tag v-if="item.isMed" size="small" type="success" effect="plain">药品处方</el-tag>
                    <el-tag v-else size="small" type="info" effect="plain">临床医嘱</el-tag>
                  </div>
                </div>
              </div>
              <div v-else class="text-xs text-slate-500 p-3">
                当前尚未签发最终处方与临床处置医嘱（请待检查报告回传后在终审确诊阶段下达）。
              </div>
            </div>

            <!-- 责任医师数字签名与存证固化 -->
            <div class="doctor-signature-banner mt-3">
              <div class="dsb-left">
                <div class="dsb-doctor">
                  <span class="text-xs text-gray-500">经治执业医师：</span>
                  <strong class="text-slate-800">{{ record.doctor_name }}</strong>
                  <span v-if="record.status === 'COMPLETED' || record.fabric_tx_id" class="dsb-ca-tag">CA 电子认证已签署</span>
                </div>
                <div class="dsb-time text-xs text-gray-400 mt-1">
                  签发归档时间：{{ formatTime(record.created_at) }}
                </div>
              </div>
              <div class="dsb-right">
                <div class="dsb-hash-note">
                  <span>存证区块：<code>{{ record.block_height ? `#${record.block_height}` : '待同步' }}</code></span>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- 底部返回与操作栏 -->
    <div class="ws-page-footer-card mt-3">
      <div class="ws-footer-bar">
        <el-button @click="$emit('back')">{{ variant === 'cross' ? '返回跨院检索列表' : '返回就诊列表' }}</el-button>
        <div class="flex gap-2">
          <el-button type="primary" :icon="View" @click="$emit('view-pdf', record)">
            {{ variant === 'cross' ? '在线查阅完整 PDF 凭证' : '查阅规范红头 PDF 电子病历' }}
          </el-button>
          <el-button type="success" plain :icon="Download" @click="$emit('download-record', record.id)">
            {{ variant === 'cross' ? '下载解密病历凭据' : '下载原始解密凭据' }}
          </el-button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { WarningFilled, CircleCheckFilled, InfoFilled, View, Download, OfficeBuilding } from '@element-plus/icons-vue'
import InfectionSafetyAlert from '@/components/InfectionSafetyAlert.vue'
import HashCapsule from '@/components/common/HashCapsule.vue'
import ChainVerifyBadge from '@/components/common/ChainVerifyBadge.vue'
import MedicalRecordUploadDialog from './MedicalRecordUploadDialog.vue'
import { formatEncounterType, formatTime } from '@/utils/recordFormatters'
import {
  evaluateTemperature,
  evaluateBloodPressure,
  evaluateHeartRate,
  evaluateSpo2,
  type VitalBadge,
} from '@/utils/vitalSignsEvaluator'
import type { MedicalRecordRow, MedicalFileItem, ParsedVitals, PlanItem } from '@/types/clinical'

const props = defineProps<{
  record: MedicalRecordRow | null
  variant: 'cross' | 'local'
  userId?: number
  userHospitalId?: number
  isBreakGlassRelease?: boolean
  token: string
  formatPatientName: (rec?: MedicalRecordRow) => string
  maskIDCard: (idCard?: string) => string
  formatHospName: (hId?: number) => string
  formatRecordStatus: (st?: string) => string
}>()

defineEmits<{
  (e: 'view-image', file: MedicalFileItem): void
  (e: 'view-pdf', record: MedicalRecordRow | null): void
  (e: 'download-image', file: MedicalFileItem): void
  (e: 'download-record', recordId?: number): void
  (e: 'copy-text', text: string, label: string): void
  (e: 'back'): void
}>()

function getRecordCID(rec: MedicalRecordRow | null): string {
  if (!rec) return ''
  if (rec.ipfs_cid) return rec.ipfs_cid
  if (rec.files && rec.files.length > 0 && rec.files[0].ipfs_cid) return rec.files[0].ipfs_cid
  return ''
}

function getRecordFileHash(rec: MedicalRecordRow | null): string {
  if (!rec) return ''
  if (rec.file_hash) return rec.file_hash
  if (rec.files && rec.files.length > 0 && rec.files[0].file_hash) return rec.files[0].file_hash
  return ''
}

function parseVitals(vitalStr?: string): ParsedVitals {
  const res: ParsedVitals = {
    temperature: '',
    blood_pressure: '',
    heart_rate: '',
    spo2: '',
    raw: vitalStr || ''
  }
  if (!vitalStr) return res

  const tMatch = vitalStr.match(/(?:T|体温)[:：\s]*([0-9.]+\s*℃?)/i)
  if (tMatch) res.temperature = tMatch[1].includes('℃') ? tMatch[1] : `${tMatch[1]} ℃`

  const bpMatch = vitalStr.match(/(?:BP|血压)[:：\s]*([0-9]+\/[0-9]+\s*mmHg?)/i)
  if (bpMatch) res.blood_pressure = bpMatch[1].includes('mmHg') ? bpMatch[1] : `${bpMatch[1]} mmHg`

  const hrMatch = vitalStr.match(/(?:HR|心率|脉搏)[:：\s]*([0-9]+\s*bpm?)/i)
  if (hrMatch) res.heart_rate = hrMatch[1].includes('bpm') ? hrMatch[1] : `${hrMatch[1]} bpm`

  const spo2Match = vitalStr.match(/(?:SpO2|血氧)[:：\s]*([0-9.]+\s*%?)/i)
  if (spo2Match) res.spo2 = spo2Match[1].includes('%') ? spo2Match[1] : `${spo2Match[1]} %`

  return res
}

function toShortVitalBadge(badge: VitalBadge): { level: string; shortLabel: string } {
  if (badge.level === 'unknown') return { level: 'unknown', shortLabel: '未录入' }
  if (badge.level === 'normal') return { level: 'normal', shortLabel: '正常' }
  if (badge.level === 'critical') return { level: 'critical', shortLabel: '危急' }
  return { level: 'elevated', shortLabel: '异常' }
}

const parsedVitals = computed(() => parseVitals(props.record?.vital_signs))
const tempVitalBadge = computed(() => toShortVitalBadge(evaluateTemperature(parsedVitals.value.temperature)))
const bpVitalBadge = computed(() => toShortVitalBadge(evaluateBloodPressure(parsedVitals.value.blood_pressure)))
const hrVitalBadge = computed(() => toShortVitalBadge(evaluateHeartRate(parsedVitals.value.heart_rate)))
const spo2VitalBadge = computed(() => toShortVitalBadge(evaluateSpo2(parsedVitals.value.spo2)))

function parsePlanItems(planStr?: string): PlanItem[] {
  if (!planStr) return []
  const lines = planStr.split(/\n+/).map(l => l.trim()).filter(Boolean)
  return lines.map((line, idx) => {
    const cleaned = line.replace(/^[0-9]+[、.．]\s*/, '')
    const isMed = /[0-9]+(mg|g|ml|片|粒|包|支)|口服|静脉|外用|每日|bid|tid|qd/i.test(cleaned)
    return {
      id: idx + 1,
      text: cleaned,
      isMed
    }
  })
}
</script>
<style scoped src="@/styles/clinical-workstation.css"></style>
