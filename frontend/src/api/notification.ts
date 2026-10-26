import client from './client'
import type { ApiEnvelope, NotificationItem } from '@/types'

export function listNotificationsApi(): Promise<ApiEnvelope<NotificationItem[]>> {
  return client.get('/notifications')
}

export function confirmAccessRequestApi(reqNoOrId: string | number): Promise<ApiEnvelope<any>> {
  return client.post(`/access-requests/${reqNoOrId}/confirm`)
}
