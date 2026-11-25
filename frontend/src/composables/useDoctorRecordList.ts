/**
 * useDoctorRecordList — DoctorRecordsView 列表状态、过滤、分页、格式化与详情调阅组合式函数
 */
import { ref, computed, watch } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { listRecordsApi, getRecordDetailApi, getPatientInfectionRisksApi } from '@/api'
import type { MedicalRecordRow } from '@/types/clinical'

export interface UseDoctorRecordListOptions {
  onNavigateToCreate?: () => void
  onNavigateToCrossQuery?: (idCard: string, name: string, pid: number) => void
}

export function useDoctorRecordList() {
  const router = useRouter()

  // ── 列表与检索状态 ──
  const loading = ref(false)
  const records = ref<MedicalRecordRow[]>([])
  const activeStatusFilter = ref('ALL')
  const searchKeyword = ref('')

  const waitingExamCount = computed(() =>
    records.value.filter(r => r.status === 'WAITING_EXAM' || r.status === 'PROCESSING_EXAM').length
  )
  const examCompletedCount = computed(() =>
    records.value.filter(r => r.status === 'EXAM_COMPLETED' || r.status === 'INITIAL_DIAGNOSIS').length
  )
  const completedCount = computed(() =>
    records.value.filter(r => r.status === 'COMPLETED').length
  )

  const currentPage = ref(1)
  const pageSize = ref(10)

  const filteredRecords = computed(() => {
    let list = records.value
    if (activeStatusFilter.value !== 'ALL') {
      if (activeStatusFilter.value === 'WAITING_EXAM') {
        list = list.filter((r: MedicalRecordRow) => r.status === 'WAITING_EXAM' || r.status === 'PROCESSING_EXAM')
      } else if (activeStatusFilter.value === 'EXAM_COMPLETED') {
        list = list.filter((r: MedicalRecordRow) => r.status === 'EXAM_COMPLETED' || r.status === 'INITIAL_DIAGNOSIS')
      } else {
        list = list.filter((r: MedicalRecordRow) => r.status === activeStatusFilter.value)
      }
    }
    if (searchKeyword.value.trim()) {
      const kw = searchKeyword.value.trim().toLowerCase()
      list = list.filter((r: MedicalRecordRow) =>
        (r.record_no && r.record_no.toLowerCase().includes(kw)) ||
        (r.patient_name && r.patient_name.toLowerCase().includes(kw)) ||
        (r.chief_complaint && r.chief_complaint.toLowerCase().includes(kw)) ||
        (r.initial_diagnosis && r.initial_diagnosis.toLowerCase().includes(kw)) ||
        (r.diagnosis && r.diagnosis.toLowerCase().includes(kw)) ||
        (r.exam_items && String(r.exam_items).toLowerCase().includes(kw))
      )
    }
    return list
  })

  const pagedRecords = computed(() => {
    const start = (currentPage.value - 1) * pageSize.value
    return filteredRecords.value.slice(start, start + pageSize.value)
  })

  watch([activeStatusFilter, searchKeyword], () => {
    currentPage.value = 1
  })

  function formatPatientName(rec?: MedicalRecordRow): string {
    if (!rec) return '未登记患者'
    const name = rec.patient_name || rec.patientName
    if (name && name !== '患者' && !name.startsWith('患者#')) {
      return name
    }
    const id = rec.patient_id || rec.patientId
    return id ? `患者 #${id}` : '未登记患者'
  }

  function formatRecordStatus(st?: string): string {
    if (st === 'COMPLETED' || st === '已接诊' || st === '已归档存证' || !st) return '已归档存证'
    if (st === 'WAITING_EXAM') return '待医技检查'
    if (st === 'PROCESSING_EXAM') return '检查进行中'
    if (st === 'EXAM_COMPLETED') return '检查完成/待确诊'
    if (st === 'INITIAL_DIAGNOSIS') return '初诊完成/待确诊'
    return st
  }

  // ── 数据获取 ──
  async function fetchRecords() {
    loading.value = true
    try {
      const res = await listRecordsApi()
      if (res.code === 200) {
        const allRecords = (res.data ?? []) as MedicalRecordRow[]
        const authorizedList = allRecords.filter((r) => {
          if (r.has_access === false) return false
          if (r.access_type === 'UNAUTHORIZED') return false
          if (typeof r.diagnosis === 'string' && r.diagnosis.includes('未授权掩码')) return false
          return true
        })
        records.value = authorizedList.map((r) => {
          if (!r.patient_name || r.patient_name === '患者' || r.patient_name.startsWith('患者#')) {
            r.patient_name = formatPatientName(r)
          }
          if (!r.doctor_name) {
            r.doctor_name = '经治医师'
          }
          if (r.status === '已接诊' || (!r.need_exam && r.diagnosis)) {
            r.status = 'COMPLETED'
          }
          return r
        })
      }
    } catch (err: unknown) {
      const e = err as { message?: string }
      ElMessage.error('获取就诊档案失败: ' + (e.message || '网络异常'))
    } finally {
      loading.value = false
    }
  }

  // ── 跨院跳转 ──
  function goToCrossQueryForPatient(
    selectedPatientInfo: { value: { id?: number; real_name?: string; id_card?: string } | null },
    saveEncounterDraft: () => void
  ) {
    if (!selectedPatientInfo.value) {
      ElMessage.warning('请先在左侧选择就诊患者')
      return
    }
    const idCard = selectedPatientInfo.value.id_card || selectedPatientInfo.value.real_name
    saveEncounterDraft()
    ElMessage.info({
      message: `已暂存患者【${selectedPatientInfo.value.real_name}】初诊录入内容，调阅病历后点击左上角即可返回继续编辑`,
      duration: 3500
    })
    router.push({
      path: '/doctor/cross-query',
      query: {
        id_card: idCard,
        patient_name: selectedPatientInfo.value.real_name,
        patient_id: selectedPatientInfo.value.id,
        scope: 'all',
        from: 'encounter_create'
      }
    })
  }

  // ── 详情调阅 / PDF 预览 / 链上核验 ──
  const detailRecord = ref<MedicalRecordRow | null>(null)

  async function openViewDetail(
    row: MedicalRecordRow,
    setView: (view: 'LIST' | 'CREATE' | 'FINAL' | 'DETAIL') => void,
    promptTamperedRecordWarning: (record: MedicalRecordRow) => void
  ) {
    try {
      const res = await getRecordDetailApi(row.id)
      if (res.code === 200) {
        detailRecord.value = res.data as MedicalRecordRow
        if (!detailRecord.value.patient_name || detailRecord.value.patient_name === '患者' || detailRecord.value.patient_name.startsWith('患者#')) {
          detailRecord.value.patient_name = formatPatientName(detailRecord.value)
        }
        if (!detailRecord.value.doctor_name) {
          detailRecord.value.doctor_name = '李建国医生'
        }
        if (detailRecord.value.status === '已接诊' || detailRecord.value.status === '已归档存证' || detailRecord.value.diagnosis) {
          detailRecord.value.status = 'COMPLETED'
        }
        if (!detailRecord.value.infection_alert && detailRecord.value.patient_id) {
          try {
            const riskRes = await getPatientInfectionRisksApi(detailRecord.value.patient_id)
            if (riskRes.code === 200 && riskRes.data?.has_risk) {
              detailRecord.value.infection_alert = riskRes.data as never
            }
          } catch {}
        }
        setView('DETAIL')
        if (detailRecord.value.is_tampered) {
          promptTamperedRecordWarning(detailRecord.value)
        }
      }
    } catch (err: unknown) {
      const e = err as { message?: string }
      ElMessage.error(e.message || '获取病历详情失败')
    }
  }

  return {
    // 列表状态
    loading,
    records,
    activeStatusFilter,
    searchKeyword,
    waitingExamCount,
    examCompletedCount,
    completedCount,
    currentPage,
    pageSize,
    filteredRecords,
    pagedRecords,
    // 工具函数
    formatPatientName,
    formatRecordStatus,
    fetchRecords,
    goToCrossQueryForPatient,
    // 详情
    detailRecord,
    openViewDetail,
  }
}