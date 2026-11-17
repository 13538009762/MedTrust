/**
 * useEncounterWorkflow — 接诊初诊/最终确诊工作流组合式函数 (WP-16)
 *
 * 从 DoctorRecordsView.vue 抽取的接诊草稿管理、初诊表单、生命体征、
 * SOAP 规则生成与提交逻辑。文案、API 调用与交互流程保持原样。
 */
import { ref, watch } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { createInitialEncounterApi, completeEncounterFinalApi, getPatientInfectionRisksApi, listPatientsApi } from '../api'
import { usePatientTimeline } from './usePatientTimeline'
import { formatEncounterType } from '../utils/recordFormatters'
import { scanTextForInfections } from '../utils/infectionGuard'
import { useAuthStore } from '../stores/auth'
import type {
  PatientOption,
  InfectionAlert,
  EncounterMode,
  EncounterInitForm,
  VitalSigns,
  DiagnosisFormState,
  MedicalRecordRow,
} from '../types/clinical'

const DRAFT_STORAGE_KEY = 'medtrust_encounter_draft'

export interface EncounterWorkflowCallbacks {
  /** 视图切换（由宿主视图持有 currentView） */
  setView: (view: 'LIST' | 'CREATE' | 'FINAL' | 'DETAIL') => void
  /** 提交成功后刷新列表 */
  refreshRecords: () => void
}

export function useEncounterWorkflow(callbacks: EncounterWorkflowCallbacks) {
  const auth = useAuthStore()

  // ── 草稿状态 ──
  const draftPatientName = ref('')
  const hasSavedDraft = ref(false)

  function updateDraftState() {
    try {
      const draftStr = sessionStorage.getItem(DRAFT_STORAGE_KEY)
      if (draftStr) {
        const d = JSON.parse(draftStr)
        if (d && d.selectedPatientInfo) {
          draftPatientName.value = d.selectedPatientInfo.real_name || d.selectedPatientInfo.username || '就诊患者'
          hasSavedDraft.value = true
          return
        }
      }
    } catch {}
    draftPatientName.value = ''
    hasSavedDraft.value = false
  }

  function saveEncounterDraft() {
    const draft = {
      selectedPatientInfo: selectedPatientInfo.value,
      selectedPatientInfectionAlert: selectedPatientInfectionAlert.value,
      initForm: { ...initForm.value },
      vitals: { ...vitals.value },
      encounterMode: encounterMode.value,
      directForm: { ...directForm.value },
      selectedExamList: [...selectedExamList.value],
      timestamp: Date.now()
    }
    sessionStorage.setItem(DRAFT_STORAGE_KEY, JSON.stringify(draft))
    updateDraftState()
  }

  function clearEncounterDraft() {
    sessionStorage.removeItem(DRAFT_STORAGE_KEY)
    draftPatientName.value = ''
    hasSavedDraft.value = false
  }

  function checkAndRestoreDraft(quiet = false): boolean {
    const draftStr = sessionStorage.getItem(DRAFT_STORAGE_KEY)
    if (!draftStr) return false
    try {
      const draft = JSON.parse(draftStr)
      if (draft && draft.selectedPatientInfo) {
        selectedPatientInfo.value = draft.selectedPatientInfo
        selectedPatientInfectionAlert.value = draft.selectedPatientInfectionAlert || null
        if (draft.selectedPatientInfo.id) {
          void fetchPatientTimeline(draft.selectedPatientInfo.id)
        }
        if (draft.initForm) initForm.value = { ...initForm.value, ...draft.initForm }
        if (draft.vitals) vitals.value = { ...vitals.value, ...draft.vitals }
        if (draft.encounterMode) encounterMode.value = draft.encounterMode
        if (draft.directForm) directForm.value = { ...directForm.value, ...draft.directForm }
        if (draft.selectedExamList) selectedExamList.value = draft.selectedExamList || []
        callbacks.setView('CREATE')
        updateDraftState()
        if (!quiet) {
          ElMessage.success({
            message: `已自动恢复患者【${draft.selectedPatientInfo.real_name || '患者'}】就诊录入草稿，可继续填写！`,
            duration: 3500
          })
        }
        return true
      }
    } catch (e) {
      console.warn('Failed to parse encounter draft', e)
    }
    return false
  }

  // ── 表单状态 ──
  const selectedPatientInfo = ref<PatientOption | null>(null)
  const selectedPatientInfectionAlert = ref<InfectionAlert | null>(null)
  const patientList = ref<PatientOption[]>([])
  const {
    timelineRecords: patientTimelineRecords,
    timelineLoading: patientTimelineLoading,
    loadTimeline: fetchPatientTimeline,
    resetTimeline: resetPatientTimeline
  } = usePatientTimeline(15)
  const selectedExamList = ref<string[]>([])
  const submitting = ref(false)
  const submittingFinal = ref(false)
  const encounterMode = ref<EncounterMode>('DIRECT')

  const initForm = ref<EncounterInitForm>({
    patient_id: undefined,
    encounter_type: 'OUTPATIENT',
    department_name: '心血管内科',
    onset_time: '',
    duration: '持续 2 天',
    chief_complaint: '',
    present_illness: '',
    initial_diagnosis: '',
    diagnostic_basis: '',
    exam_reason: ''
  })

  watch(() => initForm.value.department_name, (dept) => {
    if (dept && dept.includes('急诊')) {
      initForm.value.encounter_type = 'EMERGENCY'
    } else if (dept) {
      initForm.value.encounter_type = 'OUTPATIENT'
    }
  })

  const vitals = ref<VitalSigns>({
    temperature: '36.6',
    blood_pressure: '128/82',
    heart_rate: '76',
    spo2: '98'
  })

  const directForm = ref<DiagnosisFormState>({
    diagnosis: '',
    etiology: '',
    treatment_plan: '',
    soap_content: ''
  })

  const finalForm = ref<DiagnosisFormState>({
    diagnosis: '',
    etiology: '',
    treatment_plan: '',
    soap_content: ''
  })
  const activeRecordForFinal = ref<MedicalRecordRow | null>(null)

  // ── 患者选择与传染病预警 ──
  async function fetchPatients() {
    try {
      const res: any = await listPatientsApi()
      if (res.code === 200) {
        patientList.value = res.data || []
      }
    } catch {}
  }

  async function onPatientSelect(pid?: number | null) {
    if (!pid) {
      initForm.value.patient_id = undefined
      selectedPatientInfo.value = null
      selectedPatientInfectionAlert.value = null
      resetPatientTimeline()
      return
    }
    // 同步绑定患者 ID，确保无论通过顶部搜索下拉还是快捷候选切换都能立即进入接诊工作台
    initForm.value.patient_id = pid
    selectedPatientInfo.value = patientList.value.find(p => p.id === pid) || null
    void fetchPatientTimeline(pid)

    // 选定患者后，自动装配该科室规范接诊模板，方便医生快速核验与调整
    if (!initForm.value.chief_complaint) {
      initForm.value.duration = '持续 2 天'
      initForm.value.chief_complaint = '胸骨后发作性压榨样闷痛2天，劳累后加剧'
      initForm.value.present_illness = '患者2天前爬楼劳累后突感心前区及胸骨后压榨样闷痛，每次持续3-5分钟，伴心慌出冷汗，休息后可轻微缓解。'
      initForm.value.initial_diagnosis = '原发性高血压病1级待查；冠脉缺血待排'
      initForm.value.diagnostic_basis = '中年患者，活动诱发胸闷心悸，既往有轻度血压偏高史，需心电图及血常规排查。'
      initForm.value.exam_reason = '排查急性心肌缺血病变及炎性反应'
    }
    if (!directForm.value.diagnosis) {
      directForm.value.diagnosis = '原发性高血压病1级（低危组）'
      directForm.value.etiology = '活动劳累及情绪应激导致交感神经过度兴奋，外周阻力血管收缩，收缩压升高。'
      directForm.value.treatment_plan = '【综合处置】\n1. 静坐休息15分钟后再次复测双侧上臂血压，持续脉氧监测\n\n【临床医嘱】\n1. 饮食指导：严格低盐低脂清淡饮食（每日食盐量控制在 5g 以内）\n2. 每日早晚各监测一次血压并详细做健康日志记录\n3. 避免连续熬夜加班、情绪激动与重度体力劳累，适度慢走运动\n4. 2周后心血管内科专科门诊复查\n\n【处方用药】\n1. 硝苯地平控释片 (拜新同) 30mg 口服 每日一次 清晨餐后整片吞服\n2. 阿托伐他汀钙片 (立普妥) 20mg 口服 每晚睡前一次'
    }
    if (selectedExamList.value.length === 0) {
      selectedExamList.value = ['12导联心电图 (ECG)', '常规全血细胞分析 (血常规)']
    }
    quickFillVitals()

    if (encounterMode.value === 'DIRECT') {
      generateDirectSOAP()
    }

    // 1. 若当前患者对象已有预带的传染病风险
    if (selectedPatientInfo.value?.infection_alert) {
      selectedPatientInfectionAlert.value = selectedPatientInfo.value.infection_alert
    } else {
      selectedPatientInfectionAlert.value = null
    }

    // 2. 异步请求跨机构全景传染病风险评估
    if (pid) {
      try {
        const res: any = await getPatientInfectionRisksApi(pid)
        if (initForm.value.patient_id !== pid) return
        if (res.code === 200 && res.data) {
          selectedPatientInfectionAlert.value = res.data
        }
      } catch {
        if (initForm.value.patient_id !== pid) return
        if (selectedPatientInfo.value) {
          const text = `${selectedPatientInfo.value.real_name} ${selectedPatientInfo.value.user_no}`
          selectedPatientInfectionAlert.value = scanTextForInfections(text)
        }
      }
    }
  }

  function pickQuickPatient(pid: number) {
    void onPatientSelect(pid)
  }

  function changeSelectedPatient() {
    initForm.value.patient_id = undefined
    selectedPatientInfo.value = null
    selectedPatientInfectionAlert.value = null
    resetPatientTimeline()
  }

  // ── 生命体征与 SOAP 规则生成 ──
  function quickFillVitals() {
    vitals.value = {
      temperature: '36.5',
      blood_pressure: '122/80',
      heart_rate: '74',
      spo2: '99'
    }
    ElMessage.success('已填入正常参考生命体征数值！')
  }

  function formatVitalSignsText() {
    const parts = []
    const v = vitals.value as any
    if (v?.sbp && v?.dbp) parts.push(`血压 ${v.sbp}/${v.dbp} mmHg`)
    else if (v?.blood_pressure) parts.push(`血压 ${v.blood_pressure}`)
    if (v?.hr || v?.heart_rate) parts.push(`心率 ${v.hr || v.heart_rate} bpm`)
    if (v?.temp || v?.temperature) parts.push(`体温 ${v.temp || v.temperature} ℃`)
    if (v?.spo2) parts.push(`血氧 ${v.spo2} %`)
    return parts.join('; ')
  }

  function generateDirectSOAP() {
    const pName = selectedPatientInfo.value?.real_name || '就诊患者'
    const timeStr = initForm.value.onset_time || new Date().toLocaleString()
    const docName = auth.user?.real_name || '经治医师'
    const vitalStr = `T: ${vitals.value.temperature}℃, BP: ${vitals.value.blood_pressure} mmHg, HR: ${vitals.value.heart_rate} bpm, SpO2: ${vitals.value.spo2}%`

    const soapText = `【MedTrust 医疗可信共享·国家卫健委规范临床 SOAP 病历】
--------------------------------------------------------------------------------
就诊编号：系统将自动生成 | 接诊科室：${initForm.value.department_name} | 就诊类型：${formatEncounterType(initForm.value.encounter_type)}
患者姓名：${pName} | 就诊时间：${timeStr} | 责任医师：${docName}
--------------------------------------------------------------------------------
【S - Subjective 主观病史采集】
● 患者主诉：${initForm.value.chief_complaint || '就诊主诉'}
● 现病史：${initForm.value.present_illness || '患者因上述主诉就诊，病史采集如前所述。'}
● 发病时间：${initForm.value.onset_time || '近期'} | 持续时间：${initForm.value.duration || '发作性'}

【O - Objective 客观检查与测量】
● 查体生命体征：${vitalStr}
● 辅助检查说明：患者临床体征特征典型明确，经经治医师研判无需开展侵入性或大型医技辅助检查，执行临床合理诊疗减负标准。

【A - Assessment 综合评估与确诊】
● 经治医师最终确诊：${directForm.value.diagnosis || '临床明确诊断'}
● 病理诱因与机制：${directForm.value.etiology || '临床常见发病机制'}

【P - Plan 综合处置与医嘱方案】
${directForm.value.treatment_plan || '遵临床常规医嘱治疗'}

● 责任医师合规签署：本病历依据患者现病史采集与生命体征查体直接作出规范确诊，符合国家临床医疗核心制度规范。
--------------------------------------------------------------------------------
存证机构：${auth.user?.hospital_name || '本医疗中心'} | 加密协议：AES-256-GCM | 存证底层：Hyperledger Fabric`

    directForm.value.soap_content = soapText
  }

  function generateSOAPRecord() {
    const rec = activeRecordForFinal.value
    if (!rec) return

    const pName = rec.patient_name || '就诊患者'
    const timeStr = rec.created_at ? rec.created_at.substring(0, 16).replace('T', ' ') : new Date().toLocaleString()
    const docName = auth.user?.real_name || '经治医师'

    const soapText = `【MedTrust 医疗可信共享·国家卫健委规范临床 SOAP 病历】
--------------------------------------------------------------------------------
就诊编号：${rec.record_no} | 接诊科室：${rec.department_name} | 就诊类型：${formatEncounterType(rec.encounter_type)}
患者姓名：${pName} | 就诊时间：${timeStr} | 责任医师：${docName}
--------------------------------------------------------------------------------
【S - Subjective 主观病史采集】
● 患者主诉：${rec.chief_complaint || rec.symptoms || '未主诉明确症状'}
● 现病史：${rec.present_illness || '患者因上述主诉就诊，病史采集如前所述。'}
● 发病时间：${rec.onset_time || '近期'} | 持续时间：${rec.duration || '发作性'}

【O - Objective 客观检查与测量】
● 查体生命体征：${rec.vital_signs || '生命体征平稳'}
● 辅助检查说明：${rec.need_exam ? `开立检查项目：${rec.exam_items}。医技回传报告：${rec.exam_result || '已核实'}` : '体征典型明确，经医师研判无需开展侵入性辅助检查。'}

【A - Assessment 综合评估与确诊】
● 经治医师最终确诊：${finalForm.value.diagnosis || rec.diagnosis || rec.initial_diagnosis}
● 病理诱因与机制：${finalForm.value.etiology || rec.etiology || '临床明确发病机制'}

【P - Plan 综合处置与医嘱方案】
${finalForm.value.treatment_plan || rec.treatment_plan || '遵临床常规医嘱治疗'}

● 责任医师合规签署：本病历经责任医师最终审定确诊，全流程数据经国家卫健委《电子病历应用规范》数字签名与 Fabric 区块链联盟存证。
--------------------------------------------------------------------------------
存证机构：${rec.hospital_name || auth.user?.hospital_name || '本医疗中心'} | 加密协议：AES-256-GCM | 存证底层：Hyperledger Fabric`

    finalForm.value.soap_content = soapText
  }

  // ── 接诊流程控制 ──
  function openInitialDialog() {
    if (patientList.value.length === 0) {
      void fetchPatients()
    }
    updateDraftState()
    if (hasSavedDraft.value) {
      ElMessageBox.confirm(
        `检测到您有一份尚未完成的接诊录入草稿【${draftPatientName.value}】，是否恢复继续填写？`,
        '发现未完成草稿',
        {
          confirmButtonText: '恢复继续填写',
          cancelButtonText: '放弃草稿并新建',
          type: 'info'
        }
      ).then(() => {
        checkAndRestoreDraft(false)
      }).catch(() => {
        clearEncounterDraft()
        startNewBlankEncounter()
      })
      return
    }
    startNewBlankEncounter()
  }

  function startNewBlankEncounter() {
    encounterMode.value = 'DIRECT'
    const defaultDept = auth.user?.department_name || '心血管内科'
    const isEmergency = defaultDept.includes('急诊')
    // 必须先选择就诊人，初始不预选张三或任何患者！
    initForm.value = {
      patient_id: undefined,
      encounter_type: isEmergency ? 'EMERGENCY' : 'OUTPATIENT',
      department_name: defaultDept,
      onset_time: new Date().toISOString().substring(0, 19).replace('T', ' '),
      duration: '',
      chief_complaint: '',
      present_illness: '',
      initial_diagnosis: '',
      diagnostic_basis: '',
      exam_reason: ''
    }
    selectedPatientInfo.value = null
    selectedPatientInfectionAlert.value = null
    resetPatientTimeline()
    selectedExamList.value = []

    directForm.value = {
      diagnosis: '',
      etiology: '',
      treatment_plan: '',
      soap_content: ''
    }

    callbacks.setView('CREATE')
  }

  function handleCancelEncounter() {
    if (selectedPatientInfo.value || initForm.value.chief_complaint) {
      ElMessageBox.confirm(
        '确定退出当前接诊录入吗？已填写的草稿仍会临时保留，可随时返回继续编辑。',
        '退出确认',
        {
          confirmButtonText: '退出并保留草稿',
          cancelButtonText: '清空并放弃草稿',
          distinguishCancelAndClose: true,
          type: 'warning'
        }
      ).then(() => {
        saveEncounterDraft()
        callbacks.setView('LIST')
        ElMessage.info('已保留当前录入草稿')
      }).catch((action) => {
        if (action === 'cancel') {
          clearEncounterDraft()
          callbacks.setView('LIST')
          ElMessage.info('已清空未完成草稿')
        }
      })
    } else {
      callbacks.setView('LIST')
    }
  }

  // ── 提交：路径 A 直接确诊归档 ──
  async function submitDirectEncounter() {
    submitting.value = true
    try {
      const vitalStr = `T: ${vitals.value.temperature}℃, BP: ${vitals.value.blood_pressure} mmHg, HR: ${vitals.value.heart_rate} bpm, SpO2: ${vitals.value.spo2}%`
      if (!directForm.value.soap_content) {
        generateDirectSOAP()
      }

      const payload = {
        patient_id: initForm.value.patient_id,
        encounter_type: initForm.value.encounter_type,
        department_name: initForm.value.department_name,
        onset_time: initForm.value.onset_time,
        duration: initForm.value.duration,
        chief_complaint: initForm.value.chief_complaint,
        present_illness: initForm.value.present_illness,
        vital_signs: vitalStr,
        initial_diagnosis: directForm.value.diagnosis,
        diagnostic_basis: directForm.value.etiology || '临床体征明确，未开具医技检查',
        need_exam: false,
        exam_items: [],
        exam_reason: '',
        direct_complete: true,
        diagnosis: directForm.value.diagnosis,
        etiology: directForm.value.etiology || '临床发病机理明确',
        treatment_plan: directForm.value.treatment_plan,
        soap_content: directForm.value.soap_content
      }

      const res: any = await createInitialEncounterApi(payload)
      if (res.code === 200) {
        ElMessageBox.alert(
          `<div style="line-height: 1.6; font-size: 14px;">
          <p style="color: #059669; font-weight: bold; margin-bottom: 8px;">门诊就诊已一步完成最终确诊与规范归档！</p>
          <p style="margin-bottom: 6px;"><strong>可信区块链与隐私安全执行清单：</strong></p>
          <ul style="padding-left: 20px; margin: 6px 0; color: #334155; font-size: 13px;">
            <li><strong>诊疗决策：</strong>临床体征明确，无需医技辅助检查，就诊记录已直接闭环；</li>
            <li><strong>安全加密：</strong>采用 <strong>AES-256-GCM</strong> 对结构化 SOAP 电子病历完成对称加密；</li>
            <li><strong>分布式存储：</strong>密文已推送到去中心化 <strong>IPFS</strong> 存储节点；</li>
            <li><strong>不可篡改存证：</strong>病历哈希指纹已成功提交至 <strong>Hyperledger Fabric</strong> 联盟链！</li>
          </ul>
        </div>`,
          '就诊归档与区块链存证成功',
          {
            dangerouslyUseHTMLString: true,
            confirmButtonText: '确定',
            type: 'success'
          }
        )
        clearEncounterDraft()
        callbacks.setView('LIST')
        callbacks.refreshRecords()
      } else {
        ElMessage.error(res.message || '归档就诊失败')
      }
    } catch (err: any) {
      ElMessage.error('就诊归档异常: ' + (err.message || '网络错误'))
    } finally {
      submitting.value = false
    }
  }

  // ── 提交：路径 B 开立医技检查单派发 ──
  async function submitInitialEncounter() {
    submitting.value = true
    try {
      const vitalStr = `T: ${vitals.value.temperature}℃, BP: ${vitals.value.blood_pressure} mmHg, HR: ${vitals.value.heart_rate} bpm, SpO2: ${vitals.value.spo2}%`
      const payload = {
        patient_id: initForm.value.patient_id,
        encounter_type: initForm.value.encounter_type,
        department_name: initForm.value.department_name,
        onset_time: initForm.value.onset_time,
        duration: initForm.value.duration,
        chief_complaint: initForm.value.chief_complaint,
        present_illness: initForm.value.present_illness,
        vital_signs: vitalStr,
        initial_diagnosis: initForm.value.initial_diagnosis,
        diagnostic_basis: initForm.value.diagnostic_basis,
        need_exam: selectedExamList.value.length > 0,
        exam_items: selectedExamList.value,
        exam_reason: initForm.value.exam_reason
      }

      const res: any = await createInitialEncounterApi(payload)
      if (res.code === 200) {
        if (selectedExamList.value.length > 0) {
          ElMessageBox.alert(
            `<div style="line-height: 1.6; font-size: 14px;">
            <p style="color: #059669; font-weight: bold; margin-bottom: 8px;">初诊记录建立成功，已生成医技检查单并派发！</p>
            <p style="margin-bottom: 6px;"><strong>检查单派发去向说明：</strong></p>
            <ul style="padding-left: 20px; margin: 6px 0; color: #334155; font-size: 13px;">
              <li><strong>血常规 / 心肌酶谱 / 肝肾生化：</strong>已派发至 <strong>【临床检验科 (LIS任务池)】</strong>，由检验科技师 (<strong>tech_lab</strong>) 处理</li>
              <li><strong>12导联心电图 / 胸部CT / 头颅MRI：</strong>已派发至 <strong>【放射影像与心电中心 (PACS任务池)】</strong>，由影像医师 (<strong>tech_pacs</strong>) 处理</li>
            </ul>
            <p style="margin-top: 10px; font-size: 13px; color: #0284c7; font-weight: 600;">操作指引（两种演示方式均可）：</p>
            <p style="font-size: 13px; color: #475569; margin: 4px 0;"><strong>方式一（快捷协同）：</strong>直接点击左侧菜单的 <strong>「医技检查中心」</strong>，即可立即接单并出具回传报告！</p>
            <p style="font-size: 13px; color: #475569; margin: 4px 0;"><strong>方式二（多角色切换）：</strong>退出当前登录，切换为检验科账号 <strong>tech_lab</strong> (密码 <strong>123456</strong>) 或影像科账号 <strong>tech_pacs</strong> (密码 <strong>123456</strong>) 登录系统出报告！</p>
          </div>`,
            '检查申请已成功派发至医技科室',
            {
              dangerouslyUseHTMLString: true,
              confirmButtonText: '收到，立即前往医技中心',
              type: 'success'
            }
          )
        } else {
          ElMessage.success('初诊就诊记录已建立完成！')
        }
        clearEncounterDraft()
        callbacks.setView('LIST')
        callbacks.refreshRecords()
      } else {
        ElMessage.error(res.message || '创建初诊失败')
      }
    } catch (err: any) {
      ElMessage.error('创建初诊异常: ' + (err.message || '网络错误'))
    } finally {
      submitting.value = false
    }
  }

  // ── 最终确诊阶段 ──
  async function openFinalDialog(row: MedicalRecordRow) {
    activeRecordForFinal.value = row
    if (!activeRecordForFinal.value.infection_alert && activeRecordForFinal.value.patient_id) {
      try {
        const riskRes: any = await getPatientInfectionRisksApi(activeRecordForFinal.value.patient_id)
        if (riskRes.code === 200 && riskRes.data?.has_risk) {
          activeRecordForFinal.value.infection_alert = riskRes.data
        }
      } catch {}
    }
    const defaultPlan = row.treatment_plan || '【综合处置】\n1. 门诊心电图与心肌酶谱动态复查评估\n\n【临床医嘱】\n1. 限制钠盐摄入 (每日<5g)，保证充足规律睡眠\n2. 每日晨起及睡前规范监测自测血压\n3. 2周后心内科门诊复查血压及心电图\n\n【处方用药】\n1. 苯磺酸氨氯地平片 (络活喜) 5mg 口服 每日一次\n2. 酒石酸美托洛尔片 (倍他乐克) 25mg 口服 每日两次 饭前半小时'
    finalForm.value = {
      diagnosis: row.diagnosis || '原发性高血压病1级，心肌轻度供血不足',
      etiology: row.etiology || '劳累导致交感神经过度激活，外周阻力血管收缩，心肌负荷增加。',
      treatment_plan: defaultPlan,
      soap_content: ''
    }
    generateSOAPRecord()
    callbacks.setView('FINAL')
  }

  async function submitFinalEncounter() {
    if (!activeRecordForFinal.value) return
    submittingFinal.value = true
    try {
      const p = {
        record_id: activeRecordForFinal.value.id,
        diagnosis: finalForm.value.diagnosis,
        etiology: finalForm.value.etiology,
        treatment_plan: finalForm.value.treatment_plan,
        soap_content: finalForm.value.soap_content
      }
      const res: any = await completeEncounterFinalApi(activeRecordForFinal.value.id, p)
      if (res.code === 200) {
        ElMessage.success('最终确诊已签署下达，全量临床 SOAP 病历已成功完成 Hyperledger Fabric 区块链存证！')
        callbacks.setView('LIST')
        callbacks.refreshRecords()
      } else {
        ElMessage.error(res.message || '确认确诊失败')
      }
    } catch (err: any) {
      ElMessage.error('确认确诊异常: ' + (err.message || '网络错误'))
    } finally {
      submittingFinal.value = false
    }
  }

  return {
    // 草稿
    draftPatientName,
    hasSavedDraft,
    updateDraftState,
    saveEncounterDraft,
    clearEncounterDraft,
    checkAndRestoreDraft,
    // 患者与表单
    selectedPatientInfo,
    selectedPatientInfectionAlert,
    patientList,
    patientTimelineRecords,
    patientTimelineLoading,
    fetchPatientTimeline,
    selectedExamList,
    submitting,
    submittingFinal,
    encounterMode,
    initForm,
    vitals,
    directForm,
    finalForm,
    activeRecordForFinal,
    // 患者选择
    fetchPatients,
    onPatientSelect,
    pickQuickPatient,
    changeSelectedPatient,
    // SOAP
    quickFillVitals,
    formatVitalSignsText,
    generateDirectSOAP,
    generateSOAPRecord,
    // 流程
    openInitialDialog,
    startNewBlankEncounter,
    handleCancelEncounter,
    submitDirectEncounter,
    submitInitialEncounter,
    openFinalDialog,
    submitFinalEncounter,
  }
}