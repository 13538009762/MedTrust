import { describe, it, expect } from 'vitest'
import type { PatientOption } from '../../types/clinical'
import { filterPatientOptions } from '../patientSearchUtils'

const SAMPLE_PATIENTS: PatientOption[] = [
  {
    id: 4,
    user_no: 'PAT_0001',
    username: 'pat_zhang',
    real_name: '张三',
    id_card: '110101198805122315',
    phone: '13900000001'
  },
  {
    id: 26,
    user_no: 'PAT_0009',
    username: 'pat_zhouhq',
    real_name: '周华强',
    id_card: '110101198409121234',
    phone: '13900000009'
  },
  {
    id: 1207,
    user_no: 'PAT_20261096f019',
    username: 'lisi',
    real_name: '李四',
    id_card: '440307200412211035',
    phone: '13636597852'
  }
]

describe('filterPatientOptions unit tests', () => {
  it('returns all patients when query is empty or whitespace', () => {
    expect(filterPatientOptions(SAMPLE_PATIENTS, '')).toEqual(SAMPLE_PATIENTS)
    expect(filterPatientOptions(SAMPLE_PATIENTS, '   ')).toEqual(SAMPLE_PATIENTS)
  })

  it('matches by real name, card number, username, id card, and phone case-insensitively', () => {
    expect(filterPatientOptions(SAMPLE_PATIENTS, '周华强').map((p) => p.id)).toEqual([26])
    expect(filterPatientOptions(SAMPLE_PATIENTS, 'pat_0009').map((p) => p.id)).toEqual([26])
    expect(filterPatientOptions(SAMPLE_PATIENTS, 'PAT_ZHANG').map((p) => p.id)).toEqual([4])
    expect(filterPatientOptions(SAMPLE_PATIENTS, '19840912').map((p) => p.id)).toEqual([26])
    expect(filterPatientOptions(SAMPLE_PATIENTS, '13636597852').map((p) => p.id)).toEqual([1207])
  })

  it('returns empty array when no patient matches query', () => {
    expect(filterPatientOptions(SAMPLE_PATIENTS, '不存在患者')).toEqual([])
  })
})
