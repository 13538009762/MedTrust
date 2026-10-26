import client from './client'
import type {
  ApiEnvelope,
  DashboardOverview,
  EmergencyAccessEvent,
  AuditLog,
  VerificationResult,
  AuditEmergencyReq,
  LiftDoctorRestrictionReq,
} from '@/types'

export function getSupervisorOverviewApi(): Promise<ApiEnvelope<DashboardOverview>> {
  return client.get('/supervisor/overview')
}

export function getAccessHeatmapApi(): Promise<ApiEnvelope<[number, number, number][]>> {
  return client.get('/supervisor/access-heatmap')
}

export function getHospitalFlowApi(): Promise<ApiEnvelope<any[]>> {
  return client.get('/supervisor/hospital-flow')
}

export function getAuthFunnelApi(): Promise<ApiEnvelope<any>> {
  return client.get('/supervisor/auth-funnel')
}

export function listEmergencyEventsApi(status?: string): Promise<ApiEnvelope<EmergencyAccessEvent[]>> {
  const url = status ? `/supervisor/emergency-events?status=${status}` : '/supervisor/emergency-events'
  return client.get(url)
}

export function auditEmergencyEventApi(eventNo: string, data: AuditEmergencyReq): Promise<ApiEnvelope<any>> {
  return client.post(`/supervisor/emergency-events/${eventNo}/audit`, data)
}

export function liftDoctorRestrictionApi(data: LiftDoctorRestrictionReq): Promise<ApiEnvelope<any>> {
  return client.post('/supervisor/lift-doctor-restriction', data)
}

export function listBlockchainTransactionsApi(page = 1, pageSize = 30): Promise<ApiEnvelope<any>> {
  return client.get(`/blockchain/transactions?page=${page}&page_size=${pageSize}`)
}

export function listAuditLogsApi(params?: {
  operation_type?: string
  risk_level?: string
}): Promise<ApiEnvelope<AuditLog[]>> {
  return client.get('/audit-logs', { params })
}

export function verifyRecordApi(recordId: number): Promise<ApiEnvelope<VerificationResult>> {
  return client.post(`/verification/${recordId}`)
}

export function simulateTamperRecordApi(recordId: number): Promise<ApiEnvelope<VerificationResult>> {
  return client.post(`/verification/simulate-tamper/${recordId}`)
}

export function restoreRecordApi(recordId: number): Promise<ApiEnvelope<VerificationResult>> {
  return client.post(`/verification/restore/${recordId}`)
}
