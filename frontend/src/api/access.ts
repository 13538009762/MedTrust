import client from './client'
import type {
  ApiEnvelope,
  Authorization,
  AccessRequest,
  CreateAccessRequestReq,
  BatchAccessRequestReq,
  KeyUnlockVerifyReq,
  KeyUnlockVerifyRecordReq,
  EmergencyUnlockReq,
  EmergencySingleUnlockReq,
  PatientEmergencyFeedbackReq,
  SetRecordAccessPolicyReq,
  SetAllAccessPolicyReq,
} from '@/types'

export function listAuthorizationsApi(): Promise<ApiEnvelope<Authorization[]>> {
  return client.get('/authorizations')
}

export function createAuthorizationApi(payload: Record<string, any>): Promise<ApiEnvelope<Authorization>> {
  return client.post('/authorizations', payload)
}

export function revokeAuthorizationApi(id: number): Promise<ApiEnvelope<any>> {
  return client.delete(`/authorizations/${id}`)
}

export function revokeAllAuthorizationsApi(): Promise<ApiEnvelope<any>> {
  return client.delete('/authorizations/revoke-all')
}

export function retryRevokeApi(id: number): Promise<ApiEnvelope<any>> {
  return client.post(`/authorizations/${id}/retry-revoke`)
}

export function retryAllFailedRevokesApi(): Promise<ApiEnvelope<any>> {
  return client.post('/authorizations/retry-failed-revokes')
}

export function setRecordAccessPolicyApi(id: number, data: SetRecordAccessPolicyReq): Promise<ApiEnvelope<any>> {
  return client.post(`/medical-records/${id}/access-policy`, data)
}

export function setBatchAccessPolicyApi(data: SetAllAccessPolicyReq): Promise<ApiEnvelope<any>> {
  return client.post('/medical-records/batch-access-policy', data)
}

export function listPendingAccessRequestsApi(): Promise<ApiEnvelope<AccessRequest[]>> {
  return client.get('/access/requests/pending')
}

export function createAccessRequestApi(data: CreateAccessRequestReq): Promise<ApiEnvelope<AccessRequest>> {
  return client.post('/access/requests', data)
}

export function applyConsentRequestApi(data: Record<string, any>): Promise<ApiEnvelope<any>> {
  return client.post('/access/requests/apply-consent', data)
}

export function batchConsentRequestApi(data: BatchAccessRequestReq | Record<string, any>): Promise<ApiEnvelope<any>> {
  return client.post('/access/batch-consent', data)
}

export function approveAccessRequestApi(id: number): Promise<ApiEnvelope<any>> {
  return client.post(`/access/requests/${id}/approve`)
}

export function rejectAccessRequestApi(id: number): Promise<ApiEnvelope<any>> {
  return client.post(`/access/requests/${id}/reject`)
}

export function unlockByKeyApi(data: KeyUnlockVerifyRecordReq): Promise<ApiEnvelope<any>> {
  return client.post('/access/requests/unlock-by-key', data)
}

export function unlockPatientKeyApi(data: KeyUnlockVerifyReq): Promise<ApiEnvelope<any>> {
  return client.post('/access/unlock-patient-key', data)
}

export function breakGlassAccessApi(data: EmergencySingleUnlockReq | Record<string, any>): Promise<ApiEnvelope<any>> {
  return client.post('/access/break-glass', data)
}

export function breakGlassEmergencyBatchApi(data: EmergencyUnlockReq | Record<string, any>): Promise<ApiEnvelope<any>> {
  return client.post('/access/emergency-batch', data)
}

export function submitBreakGlassFeedbackApi(data: PatientEmergencyFeedbackReq | Record<string, any>): Promise<ApiEnvelope<any>> {
  return client.post('/access/break-glass/patient-feedback', data)
}
