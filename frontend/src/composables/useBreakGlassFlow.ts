/**
 * useBreakGlassFlow — 跨院调阅网关三态流转组合式函数 (WP-16)
 *
 * 从 CrossQueryView.vue 抽取的「动态访问风险评估 + 知情同意/现场密钥解锁 + Break-Glass 破窗」
 * 全部状态与业务逻辑，视图层只负责编排与展示。文案、API 调用与交互流程保持原样。
 */
import { ref, computed, watch, type Ref } from 'vue'
import { ElMessage, ElNotification } from 'element-plus'
import {
  createAccessRequestApi,
  getCaptchaApi,
  unlockByKeyApi,
  applyConsentRequestApi,
  breakGlassAccessApi,
  getRecordDetailApi,
} from '../api'
import type {
  MedicalRecordRow,
  RiskEvaluation,
  RiskDecisionState,
  AccessChannel,
  ConsentFormState,
  BreakGlassFormState,
} from '../types/clinical'

export interface BreakGlassFlowOptions {
  /** 调阅成功（密钥解锁/破窗放行）后释放病历详情的回调 */
  onReleased: (record: MedicalRecordRow, via: 'KEY' | 'BREAK_GLASS') => void
  /** 风控核准直接放行后进入详情的回调 */
  onAllowedDirect: (record: MedicalRecordRow, isEmergency: boolean) => void
  /** 列表刷新回调（授权状态变化后） */
  onRefresh?: () => void
}

export function useBreakGlassFlow(target: Ref<MedicalRecordRow | null>, options: BreakGlassFlowOptions) {
  // ── 三态决策与风险报告 ──
  const breakGlassVisible = ref(false)
  const interceptRiskEval = ref<RiskEvaluation | null>(null)
  const riskDecisionState = ref<RiskDecisionState>('PENDING_CONFIRM')
  const riskDecisionReason = ref('')
  const riskDecisionReqNo = ref('')

  const submittingBG = ref(false)
  const submittingConsent = ref(false)

  // 动态计算风控综合总分，确保顶层展示分值与下方多因子规则得分之和严格数学对齐
  const displayRiskScore = computed(() => {
    if (!interceptRiskEval.value) return 45
    if (interceptRiskEval.value.factors && interceptRiskEval.value.factors.length > 0) {
      const sum = interceptRiskEval.value.factors.reduce((acc: number, cur: any) => acc + (cur.score || 0), 0)
      if (sum > 0) return sum
    }
    return interceptRiskEval.value.total_score ?? 45
  })

  // ── 通道与表单 ──
  const accessChannel = ref<AccessChannel>('CONSENT')
  const consentForm = ref<ConsentFormState>({
    purpose: '门诊专科联合随访评估与既往慢病复核，需调阅外院历史健康档案与用药史',
    scope_type: 'SINGLE',
    days: 7,
  })
  const bgForm = ref<BreakGlassFormState>({
    emergency_reason: 'COMA',
    description: '患者严重休克昏迷送医，急需调阅既往严重药物过敏史及基础心脑血管病史。',
    doctor_confirmed: true,
  })

  // ── 现场密钥解锁 + 防爆破验证码 ──
  const patientKey = ref('')
  const unlockingByKey = ref(false)
  const unlockCaptchaImg = ref('')
  const unlockCaptchaToken = ref('')
  const unlockCaptchaAnswer = ref('')

  async function fetchUnlockCaptcha() {
    try {
      const res: any = await getCaptchaApi()
      if (res.code === 200 && res.data) {
        unlockCaptchaImg.value = res.data.captcha_img
        unlockCaptchaToken.value = res.data.captcha_token
        unlockCaptchaAnswer.value = ''
      }
    } catch (err) {
      console.error('Failed to fetch unlock captcha', err)
    }
  }

  watch(breakGlassVisible, (val) => {
    if (val) {
      fetchUnlockCaptcha()
    }
  })

  /** 直接从列表打开破窗通道（危重急救快捷入口，不经风控评估） */
  function openBreakGlassDirect(row: MedicalRecordRow) {
    target.value = row
    accessChannel.value = 'BREAK_GLASS'
    riskDecisionState.value = 'REJECTED'
    riskDecisionReason.value = '触发高风险阻断/危重急救场景，请核验责任声明后启用 Break-Glass 紧急破窗调阅通道'
    interceptRiskEval.value = {
      strategy: 'RULE_ENGINE_WEIGHTED',
      level: 'HIGH',
      total_score: 85,
      factors: [
        { rule_id: 'RULE-D1', factor_name: '医生执业机构与病历归属', weight: 35, score: 35, triggered: true, description: '跨医疗机构调阅（非本院开具且无有效门诊关联）' },
        { rule_id: 'RULE-A1', factor_name: '知情授权策略状态', weight: 20, score: 20, triggered: true, description: '患者未显式预签发有效知情授权策略' },
        { rule_id: 'RULE-E1', factor_name: '急诊危重救治标记', weight: 30, score: 30, triggered: true, description: '危重患者急救破窗通道触发' }
      ]
    }
    breakGlassVisible.value = true
  }

  /**
   * 跨院未授权记录调阅申请：提交安全规则与风险网关评估。
   * allowed → 直接放行；PENDING_CONFIRM → 知情/密钥；REJECTED → 仅破窗。
   */
  async function submitAccessRequest(row: MedicalRecordRow, accessPurpose = '跨机构连续性病史调阅与急诊救治') {
    target.value = row
    interceptRiskEval.value = null

    try {
      const res: any = await createAccessRequestApi({
        patient_id: row.patient_id,
        record_id: row.id,
        purpose: accessPurpose
      })

      if (res.code === 200) {
        const d = res.data || {}
        if (d.allowed) {
          riskDecisionState.value = 'ALLOWED'
          riskDecisionReason.value = d.reason || '【零信任风控核准直接放行】规则评估分值合格，已自动授权并存证上链'
          ElMessage.success(riskDecisionReason.value)
          const recRes: any = await getRecordDetailApi(row.id)
          options.onAllowedDirect(recRes.data, d.is_emergency || row.access_type === 'BREAK_GLASS')
          options.onRefresh?.()
          return
        }

        // 未直接放行：判断是待患者确认还是被拒绝
        if (d.decision === 'PENDING_CONFIRM' || d.decision === 'REQUIRE_AUTHORIZATION' || !d.decision) {
          riskDecisionState.value = 'PENDING_CONFIRM'
          riskDecisionReason.value = d.reason || '中风险调阅：需要患者在移动端确认知情同意或在现场输入专属密钥'
          riskDecisionReqNo.value = d.request_no || ('REQ-' + Math.floor(100000 + Math.random() * 900000))
          accessChannel.value = 'CONSENT'
        } else {
          riskDecisionState.value = 'REJECTED'
          riskDecisionReason.value = d.reason || '高风险调阅拦截：常规申请已阻断，若属生命垂危抢救场景可走破窗通道'
          accessChannel.value = 'BREAK_GLASS'
        }

        const resEval = d.risk_evaluation || {
          strategy: 'RULE_ENGINE_WEIGHTED',
          level: (riskDecisionState.value === 'REJECTED' ? 'HIGH' : 'MEDIUM'),
          total_score: (riskDecisionState.value === 'REJECTED' ? 75 : 45),
          factors: riskDecisionState.value === 'REJECTED' ? [
            { rule_id: 'RULE-D1', factor_name: '医生执业机构与病历归属', weight: 35, score: 35, triggered: true, description: '跨医疗机构调阅（非本院开具且无有效门诊关联）' },
            { rule_id: 'RULE-A1', factor_name: '知情授权策略状态', weight: 20, score: 20, triggered: true, description: '患者未显式预签发有效知情授权策略' },
            { rule_id: 'RULE-T1', factor_name: '调阅时间窗口与频次', weight: 20, score: 20, triggered: true, description: '非工作时间/夜间跨机构调阅，触发敏感时段防泄露预警' },
            { rule_id: 'RULE-F1', factor_name: '病历敏感度等级', weight: 15, score: 0, triggered: false, description: '常规病历数据，未触及极高敏感隐私标记' },
          ] : [
            { rule_id: 'RULE-D1', factor_name: '医生执业机构与病历归属', weight: 35, score: 25, triggered: true, description: '跨医疗机构协同调阅' },
            { rule_id: 'RULE-A1', factor_name: '知情授权策略状态', weight: 20, score: 20, triggered: true, description: '需患者在线知情授权二次核准' },
            { rule_id: 'RULE-T1', factor_name: '调阅时间窗口与频次', weight: 15, score: 0, triggered: false, description: '正常工作时间窗口且调阅频次正常' },
            { rule_id: 'RULE-F1', factor_name: '病历敏感度等级', weight: 15, score: 0, triggered: false, description: '常规病历数据，未触及极高敏感隐私标记' },
          ]
        }
        if (resEval.factors && resEval.factors.length > 0) {
          const sumScore = resEval.factors.reduce((acc: number, cur: any) => acc + (cur.score || 0), 0)
          if (sumScore > 0) resEval.total_score = sumScore
        }
        interceptRiskEval.value = resEval
        patientKey.value = ''
        breakGlassVisible.value = true
      }
    } catch (err: any) {
      const isForbidden = err?.response?.status === 403 || err?.code === 403 || err?.status === 403
      if (isForbidden) {
        // 兼容 axios 拦截器已提取 data 或未提取原生 response 的双重格式
        const errData = err?.data || err?.response?.data?.data || {}
        riskDecisionState.value = 'REJECTED'
        riskDecisionReason.value = err?.message || err?.response?.data?.message || '高风险拦截：已被风控安全网关阻断 (HTTP 403 Forbidden)'
        riskDecisionReqNo.value = errData.request_no || ''
        accessChannel.value = 'BREAK_GLASS'
        const catchEval = errData.risk_evaluation || {
          strategy: 'RULE_ENGINE_WEIGHTED',
          level: 'HIGH',
          total_score: 75,
          factors: [
            { rule_id: 'RULE-D1', factor_name: '医生执业机构与病历归属', weight: 35, score: 35, triggered: true, description: '跨医疗机构调阅（非本院开具且无有效门诊关联）' },
            { rule_id: 'RULE-A1', factor_name: '知情授权策略状态', weight: 20, score: 20, triggered: true, description: '患者未显式预签发有效知情授权策略' },
            { rule_id: 'RULE-T1', factor_name: '调阅时间窗口与频次', weight: 20, score: 20, triggered: true, description: '非工作时间/夜间跨机构调阅，触发敏感时段防泄露预警' },
            { rule_id: 'RULE-F1', factor_name: '病历敏感度等级', weight: 15, score: 0, triggered: false, description: '常规病历数据，未触及极高敏感隐私标记' },
          ]
        }
        if (catchEval.factors && catchEval.factors.length > 0) {
          const sumScore = catchEval.factors.reduce((acc: number, cur: any) => acc + (cur.score || 0), 0)
          if (sumScore > 0) catchEval.total_score = sumScore
        }
        interceptRiskEval.value = catchEval
        patientKey.value = ''
        breakGlassVisible.value = true
      } else {
        ElMessage.error(err?.message || '访问控制请求失败')
      }
    }
  }

  /** 方式一：患者现场输入专属密钥即时解锁 */
  async function handleUnlockByKey() {
    if (!target.value) return
    const key = patientKey.value.trim()
    if (!key) {
      ElMessage.warning('请输入患者专属病历调阅密钥（初始默认: 123456）')
      return
    }
    if (!unlockCaptchaAnswer.value.trim()) {
      ElMessage.warning('请输入安全计算验证码以防密钥爆破')
      return
    }

    unlockingByKey.value = true
    try {
      const res: any = await unlockByKeyApi({
        record_id: target.value.id,
        medical_key: key,
        captcha_answer: unlockCaptchaAnswer.value.trim(),
        captcha_token: unlockCaptchaToken.value,
        purpose: consentForm.value.purpose || '门诊现场患者密码授权解锁调阅',
        days: consentForm.value.days || 7
      })

      if (res.code === 200) {
        ElNotification({
          title: '患者现场密钥核验通过',
          message: '患者专属授权密钥比对成功！已即时解密放行该份跨院病历并完成区块链存证。',
          type: 'success',
          duration: 5000
        })

        breakGlassVisible.value = false
        patientKey.value = ''
        options.onReleased(res.data.record, 'KEY')
        options.onRefresh?.()
      }
    } catch (err: any) {
      fetchUnlockCaptcha()
      ElMessage.error(err?.response?.data?.message || err?.message || '患者授权密钥或验证码校验失败')
    } finally {
      unlockingByKey.value = false
    }
  }

  /** 方式二：向患者手机/个人端发起在线知情同意申请 */
  async function submitConsentRequest() {
    if (!target.value) return
    submittingConsent.value = true

    try {
      const res: any = await applyConsentRequestApi({
        record_id: target.value.id,
        purpose: consentForm.value.purpose,
        scope_type: consentForm.value.scope_type,
        days: consentForm.value.days
      })

      if (res.code === 200) {
        ElNotification({
          title: '跨院知情同意申请已发起',
          message: `已向患者 ${target.value.patient_name || '患者'} 发送调阅申请（单号：${res.data?.request_no}）！系统已实时通知患者，等待患者在个人端点击同意后，将自动释放权限。`,
          type: 'success',
          duration: 7000
        })

        breakGlassVisible.value = false
      }
    } catch (err: any) {
      ElMessage.error(err?.message || '发起知情授权申请失败')
    } finally {
      submittingConsent.value = false
    }
  }

  /** 选项二：Break-Glass 危重急救破窗放行 */
  async function submitBreakGlass() {
    if (!target.value) return
    submittingBG.value = true

    try {
      const res: any = await breakGlassAccessApi({
        record_id: target.value.id,
        emergency_reason: bgForm.value.emergency_reason,
        description: bgForm.value.description,
        doctor_confirmed: bgForm.value.doctor_confirmed
      })

      if (res.code === 200) {
        ElNotification({
          title: 'Break-Glass 紧急访问已放行',
          message: `生成紧急事件单 ${res.data.event?.event_no}，已固化上链并通知卫健监管与患者！系统已签发24小时紧急调阅授权，重启或刷新免重复申请。`,
          type: 'warning',
          duration: 6000
        })

        breakGlassVisible.value = false
        options.onReleased(res.data.record, 'BREAK_GLASS')
        options.onRefresh?.()
      }
    } catch (err: any) {
      ElMessage.error(err?.message || '紧急放行申请失败')
    } finally {
      submittingBG.value = false
    }
  }

  return {
    breakGlassVisible,
    interceptRiskEval,
    riskDecisionState,
    riskDecisionReason,
    riskDecisionReqNo,
    displayRiskScore,
    submittingBG,
    submittingConsent,
    accessChannel,
    consentForm,
    bgForm,
    patientKey,
    unlockingByKey,
    unlockCaptchaImg,
    unlockCaptchaToken,
    unlockCaptchaAnswer,
    fetchUnlockCaptcha,
    openBreakGlassDirect,
    submitAccessRequest,
    handleUnlockByKey,
    submitConsentRequest,
    submitBreakGlass,
  }
}