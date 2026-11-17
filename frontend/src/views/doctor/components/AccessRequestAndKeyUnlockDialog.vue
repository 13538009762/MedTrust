<template>
  <!-- 动态访问风险评估决策报告卡与调阅控制网关 (三态流转 + 知情同意/现场密钥解锁) -->
  <el-dialog
    :model-value="visible"
    title="动态访问风险评估决策报告卡与调阅网关"
    width="780px"
    :close-on-click-modal="false"
    class="risk-report-dialog"
    @update:model-value="$emit('update:visible', $event)"
  >
    <!-- 终审决策状态机 (三态视觉呈现) -->
    <div class="decision-status-box mb-3">
      <el-alert
        v-if="riskDecisionState === 'ALLOWED'"
        title="【终审决策：ALLOWED · 零信任风控核准直接放行】"
        type="success"
        show-icon
        :closable="false"
        description="经动态规则加权评估，当前请求属于低风险范围（或同院/已获患者授权），安全网关已核准解密放行并生成 Fabric 存证凭证。您可以直接进入工作台查阅解密病历或下载红头归档凭据。"
      />
      <el-alert
        v-else-if="riskDecisionState === 'PENDING_CONFIRM'"
        title="【终审决策：PENDING_CONFIRM · 待患者在客户端二次确认放行】"
        type="warning"
        show-icon
        :closable="false"
        :description="`跨机构就诊调阅申请单已生成（单号：${riskDecisionReqNo || 'REQ20261014888'}），当前处于中风险受控状态。系统已实时推送通知，等待患者在个人端点击同意；或患者在诊室现场可输入 6 位专属调阅密钥即时解锁！`"
      />
      <el-alert
        v-else
        title="【终审决策：REJECTED / NEED_BREAK_GLASS · 高风险访问拦截 (HTTP 403 Forbidden)】"
        type="error"
        show-icon
        :closable="false"
        description="触发高风险防御策略，常规调阅申请已被系统拦截阻断！若患者处于休克昏迷或危及生命的极端急救场景，医生可启动 Break-Glass 紧急破窗调阅通道。"
      />
    </div>

    <!-- 动态风险综合评分进度条与等级标签 -->
    <div v-if="interceptRiskEval" class="risk-score-eval-card mb-3">
      <div class="rsec-header">
        <div class="rsec-title">
          <span class="dot-indicator"></span>
          <strong>动态访问风险评估决策报告 (Risk Assessment Report)</strong>
        </div>
        <div class="rsec-badges">
          <span class="strategy-tag">策略: <code>{{ interceptRiskEval.strategy || 'RULE_ENGINE_WEIGHTED' }}</code></span>
          <el-tag
            :type="interceptRiskEval.level === 'HIGH' ? 'danger' : (interceptRiskEval.level === 'MEDIUM' ? 'warning' : 'success')"
            size="small"
            effect="dark"
          >
            风控评级: {{ interceptRiskEval.level || 'MEDIUM' }}
          </el-tag>
        </div>
      </div>

      <div class="score-progress-row">
        <div class="score-label">
          <span>综合风险分值 (Risk Score)：</span>
          <strong :style="{ color: displayRiskScore >= 60 ? '#ef4444' : (displayRiskScore >= 30 ? '#f59e0b' : '#10b981') }">
            {{ displayRiskScore }} / 100
          </strong>
        </div>
        <el-progress
          :percentage="Math.min(100, displayRiskScore)"
          :status="displayRiskScore >= 60 ? 'exception' : (displayRiskScore >= 30 ? 'warning' : 'success')"
          :stroke-width="12"
          class="score-bar"
        />
      </div>

      <!-- 命中风险规则清单表格 -->
      <div class="rsec-rules-section mt-2">
        <div class="rsec-rules-title">命中风控判定规则清单 (与后端规则引擎严格对齐)：</div>
        <el-table
          :data="interceptRiskEval.factors || []"
          size="small"
          stripe
          class="factor-table mt-1"
        >
          <el-table-column prop="rule_id" label="规则编号" width="130">
            <template #default="{ row }">
              <code class="rule-code">{{ row.rule_id }}</code>
            </template>
          </el-table-column>
          <el-table-column prop="factor_name" label="风控判定维度" width="140" />
          <el-table-column label="基准权重" width="80" align="center">
            <template #default="{ row }">+{{ row.weight }}</template>
          </el-table-column>
          <el-table-column label="实际得分" width="80" align="center">
            <template #default="{ row }">
              <strong :style="{ color: row.triggered ? '#ef4444' : '#10b981' }">
                {{ row.score > 0 ? ('+' + row.score) : '0' }}
              </strong>
            </template>
          </el-table-column>
          <el-table-column label="判定状态" width="95" align="center">
            <template #default="{ row }">
              <el-tag size="small" :type="row.triggered ? 'danger' : 'success'" effect="plain">
                {{ row.triggered ? '命中风险' : '正常放行' }}
              </el-tag>
            </template>
          </el-table-column>
          <el-table-column prop="description" label="合规裁决与风险释义" min-width="190" />
        </el-table>
      </div>
    </div>

    <!-- 终审决策三态操作面板 -->
    <!-- 状态 1: ALLOWED -->
    <div v-if="riskDecisionState === 'ALLOWED'" class="state-panel-allowed mb-2">
      <div class="allowed-guide-box">
        <el-icon class="text-emerald-500 mr-2" :size="24"><CircleCheckFilled /></el-icon>
        <div>
          <strong>已成功核准当前病历的跨院/常规调阅权限！</strong>
          <div class="text-xs text-gray-500 mt-1">数据已在安全网关解密，您可以直接点击下方按钮进入工作台查阅病历详情。</div>
        </div>
      </div>
    </div>

    <!-- 状态 2: PENDING_CONFIRM 或状态 3: REJECTED -->
    <div v-else class="state-panel-flow">
      <!-- 通道选项卡 -->
      <div class="channel-selector-wrapper mb-3">
        <el-radio-group v-model="channelModel" size="large" class="channel-tabs" style="width: 100%; display: flex;">
          <el-radio-button value="CONSENT" style="flex: 1;">
            <span>选项一：正常诊疗·现场密钥解锁 / 知情同意申请 (推荐)</span>
          </el-radio-button>
          <el-radio-button value="BREAK_GLASS" style="flex: 1;">
            <span class="text-danger font-bold">选项二：危重急救·Break-Glass 破窗</span>
          </el-radio-button>
        </el-radio-group>
      </div>

      <!-- 选项一：正常诊疗知情同意与现场患者密钥解锁通道 -->
      <div v-if="channelModel === 'CONSENT'" class="channel-panel">
        <div class="target-record-banner mb-3">
          <span class="lbl">目标调阅病历：</span>
          <span v-if="currentTarget" class="val">
            <strong style="color: #4338ca;">{{ currentTarget.record_no }}</strong>
            （患者：<strong>{{ currentTarget.patient_name }}</strong> · 机构：<strong>{{ currentTarget.hospital_name || formatHospName(currentTarget.hospital_id) }}</strong> · 科室：{{ currentTarget.department_name }}）
          </span>
        </div>

        <!-- 方式一：现场密码解锁专区（优先推荐） -->
        <div class="patient-key-box mb-3">
          <div class="pk-header">
            <div class="pk-title-wrap">
              <span class="pk-title">方式一：患者现场输入专属密钥即时解锁（推荐）</span>
            </div>
            <el-tag size="small" type="success" effect="dark">现场协同</el-tag>
          </div>
          <p class="pk-desc">
            若患者本人在接诊现场，可由<strong>患者本人在下方输入其个人专属病历调阅密钥</strong>（或告知医生代为输入），系统将立即进行密码学核验并现场解密病历、存证上链，<strong>无需在线异步等待推送审批</strong>。
          </p>
          <div class="pk-input-box">
            <el-input
              v-model="patientKeyModel"
              type="password"
              show-password
              size="large"
              placeholder="请输入患者专属病历调阅密钥（初始默认: 123456）"
              :prefix-icon="Key"
              class="pk-key-input"
              @keyup.enter="$emit('unlock-by-key')"
            />
            <div class="pk-captcha-row mt-2">
              <div class="pk-captcha-wrapper">
                <el-input
                  v-model="captchaAnswerModel"
                  size="large"
                  placeholder="输入计算结果防爆破"
                  :prefix-icon="Finished"
                  class="pk-captcha-input"
                  autocomplete="off"
                  @keyup.enter="$emit('unlock-by-key')"
                />
                <div class="pk-captcha-img-container" title="点击刷新验证码" @click="$emit('fetch-captcha')">
                  <img v-if="captchaImg" :src="captchaImg" alt="验证码" class="pk-captcha-img" />
                  <span v-else class="pk-captcha-loading">加载中...</span>
                </div>
              </div>
              <el-button
                type="success"
                size="large"
                :icon="Key"
                :loading="unlockingByKey"
                :disabled="!patientKey || !captchaAnswer"
                @click="$emit('unlock-by-key')"
                class="pk-btn"
              >
                验证密钥并即时解密调阅
              </el-button>
            </div>
          </div>
          <div class="pk-tip">
            <span>提示：患者登录个人账号可在【数据授权管理】随时修改此密钥；初始默认密钥为 <code>123456</code>。</span>
          </div>
        </div>

        <el-divider content-position="center">
          <span style="color: #94a3b8; font-size: 12px;">或患者未在现场时使用在线知情推送审批</span>
        </el-divider>

        <!-- 方式二：在线知情申请推送 -->
        <div class="online-consent-box">
          <div class="oc-header mb-2">
            <strong style="color: #334155; font-size: 14px;">方式二：向患者手机/个人端发起知情同意申请 (离线推送审批)</strong>
          </div>
          <p class="oc-desc mb-2" style="font-size: 12px; color: #64748b; line-height: 1.5;">
            医生填写临床调阅目的后向患者推送在线申请，系统将即时通知患者。患者在个人端点击同意后，系统将自动为医生释放调阅权限并存证上链。
          </p>
          <el-form :model="consentForm" label-width="110px" class="consent-form">
            <el-form-item label="临床调阅目的" required>
              <el-input
                v-model="consentForm.purpose"
                type="textarea"
                :rows="2"
                placeholder="请输入临床调阅目的与会诊说明，该说明将如实同步通知患者..."
              />
            </el-form-item>
            <el-form-item label="申请授权范围" required>
              <el-radio-group v-model="consentForm.scope_type">
                <el-radio value="SINGLE">仅限调阅当前这一份病历 (推荐)</el-radio>
                <el-radio value="ALL">调阅该患者在全网的全部历史健康档案</el-radio>
              </el-radio-group>
            </el-form-item>
            <el-form-item label="希望授权期限" required>
              <el-select v-model="consentForm.days" style="width: 220px;">
                <el-option label="3 天 (急诊随访)" :value="3" />
                <el-option label="7 天 (常规门诊)" :value="7" />
                <el-option label="14 天 (两周疗程)" :value="14" />
                <el-option label="30 天 (慢病随访)" :value="30" />
                <el-option label="90 天 (长程康复)" :value="90" />
              </el-select>
            </el-form-item>
          </el-form>
        </div>
      </div>

      <!-- 选项二：急诊危重破窗通道 (Break-Glass) -->
      <div v-else class="channel-panel">
        <!-- 若医生处于受限状态，显示醒目 403 阻断警示 -->
        <el-alert
          v-if="restricted"
          title="【监管阻断 403 Forbidden】您已被监管限制跨院/紧急访问权限，严禁发起破窗！"
          type="error"
          description="依据卫健委医疗监管违规处罚条例，您的执业账号当前处于 RESTRICTED 受限状态，已被依法冻结紧急破窗调阅权。请联系医院管理员或卫健监管专员复核并解除限制后方可恢复使用。"
          show-icon
          :closable="false"
          class="mb-3"
        />

        <div class="emergency-guide">
          <h4 style="color: #dc2626;">极端急危重症救治 Break-Glass 破窗协议说明</h4>
          <p>
            根据《医疗数据可信共享实施规范》，若患者处于<strong>严重休克、深昏迷、心跳呼吸骤停</strong>等危及生命的极端场景，无法表达知情意愿，医生签署临床法律责任声明后，系统允许触发 <strong>Break-Glass 破窗机制</strong>紧急放行并同步异步上链记录生成闭环监管审计工单。
          </p>
        </div>

        <el-form :model="bgForm" label-width="110px" class="bg-form">
          <el-form-item label="紧急抢救原因" required>
            <el-select v-model="bgForm.emergency_reason" style="width: 100%;">
              <el-option label="COMA - 患者严重昏迷休克，无法表达意愿" value="COMA" />
              <el-option label="SHOCK - 多发创伤失血性休克，急需抢救手术" value="SHOCK" />
              <el-option label="EMERGENCY - 急危重症抢救生命关键期，急需用药与过敏史" value="EMERGENCY" />
              <el-option label="CRITICAL - 突发急性危重病多学科综合救治" value="CRITICAL" />
              <el-option label="OTHER - 其他危及生命的特殊医学紧急场景" value="OTHER" />
            </el-select>
          </el-form-item>
          <el-form-item label="临床急救说明" required>
            <el-input
              v-model="bgForm.description"
              type="textarea"
              :rows="3"
              placeholder="请详细录入急救诊断情况、抢救必要性说明，此说明将直接作为不可篡改证据固化上链并推送卫健监管审查..."
            />
          </el-form-item>
          <el-form-item>
            <el-checkbox v-model="bgForm.doctor_confirmed">
              <span class="confirm-text" style="color: #dc2626; font-weight: bold;">
                我确认本次紧急调阅仅用于患者紧急临床救治，陈述真实无伪造，并承担相应医疗与法律责任
              </span>
            </el-checkbox>
          </el-form-item>
        </el-form>
      </div>
    </div>

    <template #footer>
      <el-button @click="$emit('update:visible', false)">关闭</el-button>
      <template v-if="riskDecisionState === 'ALLOWED'">
        <el-button type="success" @click="$emit('enter-detail')">
          立即解密查阅完整病历
        </el-button>
      </template>
      <template v-else-if="channelModel === 'CONSENT'">
        <el-button
          type="primary"
          :loading="submittingConsent"
          :disabled="!consentForm.purpose"
          @click="$emit('submit-consent')"
        >
          发起知情同意申请并通知患者
        </el-button>
      </template>
      <template v-else>
        <el-button
          type="danger"
          :disabled="!bgForm.doctor_confirmed || !bgForm.description || restricted"
          :loading="submittingBG"
          @click="$emit('submit-break-glass')"
        >
          {{ restricted ? '权限已受限 (严禁破窗)' : '确认申请 Break-Glass 抢救放行' }}
        </el-button>
      </template>
    </template>
  </el-dialog>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { CircleCheckFilled, Key, Finished } from '@element-plus/icons-vue'
import type {
  MedicalRecordRow,
  RiskEvaluation,
  RiskDecisionState,
  AccessChannel,
  ConsentFormState,
  BreakGlassFormState,
} from '@/types/clinical'

const props = defineProps<{
  visible: boolean
  riskDecisionState: RiskDecisionState
  riskDecisionReqNo: string
  interceptRiskEval: RiskEvaluation | null
  displayRiskScore: number
  accessChannel: AccessChannel
  consentForm: ConsentFormState
  bgForm: BreakGlassFormState
  currentTarget: MedicalRecordRow | null
  patientKey: string
  captchaImg: string
  captchaAnswer: string
  captchaToken: string
  unlockingByKey: boolean
  submittingConsent: boolean
  submittingBG: boolean
  restricted: boolean
  formatHospName: (hId?: number) => string
}>()

const emit = defineEmits<{
  (e: 'update:visible', val: boolean): void
  (e: 'update:accessChannel', val: AccessChannel): void
  (e: 'update:patientKey', val: string): void
  (e: 'update:captchaAnswer', val: string): void
  (e: 'unlock-by-key'): void
  (e: 'fetch-captcha'): void
  (e: 'submit-consent'): void
  (e: 'submit-break-glass'): void
  (e: 'enter-detail'): void
}>()

const channelModel = computed({
  get: () => props.accessChannel,
  set: (val: AccessChannel) => emit('update:accessChannel', val),
})
const patientKeyModel = computed({
  get: () => props.patientKey,
  set: (val: string) => emit('update:patientKey', val),
})
const captchaAnswerModel = computed({
  get: () => props.captchaAnswer,
  set: (val: string) => emit('update:captchaAnswer', val),
})
</script>
<style scoped src="../CrossQueryView.css"></style>

