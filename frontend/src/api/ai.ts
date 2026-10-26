import client from './client'
import type { ApiEnvelope } from '@/types'

export function diagnosticAssistApi(data: Record<string, any>): Promise<ApiEnvelope<any>> {
  return client.post('/ai/diagnostic-assist', data)
}

export function soapSummaryApi(data: Record<string, any>): Promise<ApiEnvelope<any>> {
  return client.post('/ai/soap-summary', data)
}

export function conflictDetectApi(data: Record<string, any>): Promise<ApiEnvelope<any>> {
  return client.post('/ai/conflict-detect', data)
}
