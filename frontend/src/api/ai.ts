import client from './client'
import type { ApiEnvelope } from '@/types'

export function diagnosticAssistApi(
  data: Record<string, unknown>,
  options?: { signal?: AbortSignal }
): Promise<ApiEnvelope<unknown>> {
  return client.post('/ai/diagnostic-assist', data, {
    timeout: 60000,
    signal: options?.signal,
  })
}

