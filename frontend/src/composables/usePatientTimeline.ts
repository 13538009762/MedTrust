import { ref } from 'vue'
import { listRecordsApi } from '@/api'
import type { MedicalRecordRow } from '@/types/clinical'
import {
  unwrapTimelineRecordsResponse,
  type TimelineRecordApiResponse
} from '@/utils/timelineFormatters'

export { unwrapTimelineRecordsResponse, type TimelineRecordApiResponse }

/**
 * 患者既往病例时间线静默预拉取复用组合式函数（带竞态序号保护，默认限制 page_size=15）
 */
export function usePatientTimeline(defaultPageSize: number = 15) {
  const timelineRecords = ref<MedicalRecordRow[]>([])
  const timelineLoading = ref<boolean>(false)
  let fetchSeq = 0

  function resetTimeline(): void {
    ++fetchSeq
    timelineRecords.value = []
    timelineLoading.value = false
  }

  async function loadTimeline(patientId?: number, pageSize: number = defaultPageSize): Promise<void> {
    const seq = ++fetchSeq
    if (!patientId) {
      timelineRecords.value = []
      timelineLoading.value = false
      return
    }
    timelineLoading.value = true
    try {
      const res = (await listRecordsApi({
        patient_id: patientId,
        page: 1,
        page_size: pageSize
      })) as TimelineRecordApiResponse
      if (seq !== fetchSeq) return
      timelineRecords.value = unwrapTimelineRecordsResponse(res)
    } catch {
      if (seq !== fetchSeq) return
      timelineRecords.value = []
    } finally {
      if (seq === fetchSeq) {
        timelineLoading.value = false
      }
    }
  }

  return {
    timelineRecords,
    timelineLoading,
    loadTimeline,
    resetTimeline
  }
}
