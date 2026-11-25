import { describe, it, expect, vi, beforeEach } from 'vitest'
import { ref } from 'vue'

const {
  getPatientInfectionRisksApiMock,
  listPatientsApiMock,
  loadTimelineMock,
  resetTimelineMock,
  elMessageSuccessMock,
} = vi.hoisted(() => ({
  getPatientInfectionRisksApiMock: vi.fn(),
  listPatientsApiMock: vi.fn(),
  loadTimelineMock: vi.fn(),
  resetTimelineMock: vi.fn(),
  elMessageSuccessMock: vi.fn(),
}))

vi.mock('element-plus', () => ({
  ElMessage: {
    success: elMessageSuccessMock,
    info: vi.fn(),
    warning: vi.fn(),
    error: vi.fn(),
  },
  ElMessageBox: {
    confirm: vi.fn(),
    alert: vi.fn(),
  },
}))

vi.mock('../../api', () => ({
  createInitialEncounterApi: vi.fn(),
  completeEncounterFinalApi: vi.fn(),
  getPatientInfectionRisksApi: getPatientInfectionRisksApiMock,
  listPatientsApi: listPatientsApiMock,
}))

vi.mock('../usePatientTimeline', () => ({
  usePatientTimeline: () => ({
    timelineRecords: ref([]),
    timelineLoading: ref(false),
    loadTimeline: loadTimelineMock,
    resetTimeline: resetTimelineMock,
  }),
}))

vi.mock('../../stores/auth', () => ({
  useAuthStore: () => ({
    user: {
      id: 1,
      real_name: '李建国医生',
      hospital_name: '第一人民医院',
      department_name: '心血管内科',
    },
    token: 'test-token',
  }),
}))

import { useEncounterWorkflow } from '../useEncounterWorkflow'

describe('useEncounterWorkflow patient selection regression tests', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    const mem = new Map<string, string>()
    vi.stubGlobal('sessionStorage', {
      getItem: (k: string) => mem.get(k) ?? null,
      setItem: (k: string, v: string) => mem.set(k, v),
      removeItem: (k: string) => mem.delete(k),
      clear: () => mem.clear(),
    })
  })

  it('sets initForm.patient_id when selecting a patient via search dropdown', async () => {
    getPatientInfectionRisksApiMock.mockResolvedValue({
      code: 200,
      data: { has_risk: true, summary: 'HIV' },
    })

    const wf = useEncounterWorkflow({
      setView: vi.fn(),
      refreshRecords: vi.fn(),
    })
    wf.patientList.value = [
      { id: 26, user_no: 'PAT_0009', real_name: '周华强', username: 'pat_zhouhq' },
    ]

    expect(wf.initForm.value.patient_id).toBeUndefined()

    await wf.onPatientSelect(26)

    expect(wf.initForm.value.patient_id).toBe(26)
    expect(wf.selectedPatientInfo.value?.real_name).toBe('周华强')
    expect(wf.selectedPatientInfectionAlert.value).toEqual({ has_risk: true, summary: 'HIV' })
    expect(loadTimelineMock).toHaveBeenCalledWith(26)
  })

  it('clears initForm.patient_id and infection state when patient selection is cleared', async () => {
    getPatientInfectionRisksApiMock.mockResolvedValue({
      code: 200,
      data: { has_risk: false },
    })

    const wf = useEncounterWorkflow({
      setView: vi.fn(),
      refreshRecords: vi.fn(),
    })
    wf.patientList.value = [
      { id: 4, user_no: 'PAT_0001', real_name: '张三', username: 'pat_zhang' },
    ]

    await wf.onPatientSelect(4)
    expect(wf.initForm.value.patient_id).toBe(4)

    await wf.onPatientSelect(undefined)
    expect(wf.initForm.value.patient_id).toBeUndefined()
    expect(wf.selectedPatientInfo.value).toBeNull()
    expect(wf.selectedPatientInfectionAlert.value).toBeNull()
    expect(resetTimelineMock).toHaveBeenCalled()
  })

  it('ignores stale infection risk response if doctor switches/clears patient mid-flight', async () => {
    let resolveRisk!: (val: { code: number; data: { has_risk: boolean; summary?: string } }) => void
    getPatientInfectionRisksApiMock.mockImplementation(
      () =>
        new Promise((resolve) => {
          resolveRisk = resolve
        })
    )

    const wf = useEncounterWorkflow({
      setView: vi.fn(),
      refreshRecords: vi.fn(),
    })
    wf.patientList.value = [
      { id: 26, user_no: 'PAT_0009', real_name: '周华强', username: 'pat_zhouhq' },
    ]

    const pendingSelect = wf.onPatientSelect(26)
    wf.changeSelectedPatient()
    resolveRisk({ code: 200, data: { has_risk: true, summary: 'HIV' } })
    await pendingSelect

    expect(wf.initForm.value.patient_id).toBeUndefined()
    expect(wf.selectedPatientInfectionAlert.value).toBeNull()
  })

  it('does not auto-fill fake symptoms on select and resets clinical fields when switching patients', async () => {
    getPatientInfectionRisksApiMock.mockResolvedValue({
      code: 200,
      data: { has_risk: false },
    })

    const wf = useEncounterWorkflow({
      setView: vi.fn(),
      refreshRecords: vi.fn(),
    })
    wf.patientList.value = [
      { id: 4, user_no: 'PAT_0001', real_name: '张三', username: 'pat_zhang' },
      { id: 5, user_no: 'PAT_0002', real_name: '李四', username: 'pat_li' },
    ]

    await wf.onPatientSelect(4)
    expect(wf.initForm.value.chief_complaint).toBe('')
    expect(wf.directForm.value.diagnosis).toBe('')

    wf.initForm.value.chief_complaint = '患者A专属主诉'
    wf.directForm.value.diagnosis = '患者A专属确诊'
    wf.vitals.value.blood_pressure = '150/95'

    wf.changeSelectedPatient()
    expect(wf.initForm.value.chief_complaint).toBe('')
    expect(wf.directForm.value.diagnosis).toBe('')
    expect(wf.vitals.value.blood_pressure).toBe('')
  })
})
