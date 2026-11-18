import client from './client'
import type { ApiEnvelope } from '@/types'

export function diagnosticAssistApi(data: Record<string, unknown>): Promise<ApiEnvelope<unknown>> {
  return client.post('/ai/diagnostic-assist', data)
}

