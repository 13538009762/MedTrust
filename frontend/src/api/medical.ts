import client from './client'
import type {
  ApiEnvelope,
  MedicalRecord,
  MedicalExamOrder,
  RecordQueryParams,
  InfectionRiskSummary,
} from '@/types'

export function listRecordsApi(params?: RecordQueryParams): Promise<ApiEnvelope<MedicalRecord[]>> {
  return client.get('/medical-records', { params })
}

export function getRecordDetailApi(id: number): Promise<ApiEnvelope<MedicalRecord>> {
  return client.get(`/medical-records/${id}`)
}

export function createRecordApi(data: FormData | Record<string, any>): Promise<ApiEnvelope<MedicalRecord>> {
  return client.post('/medical-records', data)
}

export function downloadRecordApi(id: number): Promise<Blob> {
  return client.get(`/medical-records/${id}/download`, { responseType: 'blob' })
}

export function getRecordFileApi(id: number): Promise<any> {
  return client.get(`/medical-records/${id}/file`)
}

export function previewRecordFileApi(id: number): Promise<any> {
  return client.get(`/medical-records/${id}/preview`)
}

export function getRecordInfectionRisksApi(id: number): Promise<ApiEnvelope<InfectionRiskSummary>> {
  return client.get(`/medical-records/${id}/infection-risks`)
}

export function getPatientInfectionRisksApi(patientId: number): Promise<ApiEnvelope<InfectionRiskSummary>> {
  return client.get(`/patients/${patientId}/infection-risks`)
}

export function listPatientsApi(params?: { keyword?: string }): Promise<ApiEnvelope<any[]>> {
  return client.get('/patients', { params })
}

export function createInitialEncounterApi(payload: Record<string, any>): Promise<ApiEnvelope<any>> {
  return client.post('/encounters/initial', payload)
}

export function completeEncounterFinalApi(id: number, payload: Record<string, any>): Promise<ApiEnvelope<any>> {
  return client.post(`/encounters/${id}/complete`, payload)
}

export function createEncounterOrdersApi(payload: Record<string, any>): Promise<ApiEnvelope<any>> {
  return client.post('/encounters/orders', payload)
}

export function listExamOrdersApi(params?: {
  status?: string
  hospital_id?: number
  patient_id?: number
}): Promise<ApiEnvelope<MedicalExamOrder[]>> {
  return client.get('/exam-orders', { params })
}

export function processExamOrderApi(id: number): Promise<ApiEnvelope<any>> {
  return client.post(`/exam-orders/${id}/process`)
}

export function completeExamOrderApi(id: number, data: FormData): Promise<ApiEnvelope<any>> {
  return client.post(`/exam-orders/${id}/complete`, data, {
    headers: { 'Content-Type': 'multipart/form-data' },
  })
}
