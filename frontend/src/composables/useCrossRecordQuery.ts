// WP-16: 跨院检索数据获取与报告结构化解析缓存 (从 CrossQueryView 迁入, 接口调用与文案保持原样)
import { ref, watch, onMounted, onBeforeUnmount, type Ref } from 'vue'
import { ElNotification } from 'element-plus'
import { listRecordsApi } from '../api'
import { useAuthStore } from '../stores/auth'
import { parseMedicalExamResult, type ParsedMedicalReport } from '../utils/medicalReportParser'
import type { MedicalRecordRow } from '../types/clinical'
import type { CrossQueryScope } from './useCrossQueryScope'
import type { RouteLocationNormalizedLoaded } from 'vue-router'

interface StatsCounts {
  authorized: number
  local: number
  pending: number
  all: number
}

interface StatsCacheEntry {
  key: string
  counts: StatsCounts
  expiresAt: number
}

// 4 个范围菜单切换时共享短期汇总统计缓存，避免每次切页都双倍请求全量病历与徽标归零闪烁
const STATS_CACHE_TTL_MS = 5_000
let statsSummaryCache: StatsCacheEntry | null = null

function computeStatsCounts(
  allList: MedicalRecordRow[],
  currentHospId: number,
  currentDocId: number
): StatsCounts {
  return {
    all: allList.length,
    local: allList.filter((r) => r.hospital_id === currentHospId).length,
    authorized: allList.filter((r) => r.hospital_id !== currentHospId && (r.has_access || r.access_type === 'BREAK_GLASS')).length,
    pending: allList.filter((r) => r.hospital_id !== currentHospId && !r.has_access && r.access_type !== 'BREAK_GLASS' && r.doctor_id !== currentDocId).length,
  }
}

function filterRowsByScope(
  rows: MedicalRecordRow[],
  scope: CrossQueryScope,
  currentHospId: number,
  currentDocId: number
): MedicalRecordRow[] {
  if (scope === 'authorized') {
    return rows.filter((row) => row.hospital_id !== currentHospId && (row.has_access || row.access_type === 'BREAK_GLASS'))
  }
  if (scope === 'pending') {
    return rows.filter((row) => row.hospital_id !== currentHospId && !row.has_access && row.access_type !== 'BREAK_GLASS' && row.doctor_id !== currentDocId)
  }
  if (scope === 'local') {
    return rows.filter((row) => row.hospital_id === currentHospId)
  }
  return rows
}

export interface UseCrossRecordQueryOptions {
  waitForEnterTransition?: () => Promise<void>
}

export function useCrossRecordQuery(
  route: RouteLocationNormalizedLoaded,
  currentScope: Ref<CrossQueryScope>,
  options?: UseCrossRecordQueryOptions
) {
  const auth = useAuthStore()
  const mountedPath = route.path
  const cacheKey = `${auth.user?.id || 0}_${auth.user?.hospital_id || 1}`
  const cachedEntry = (statsSummaryCache && statsSummaryCache.key === cacheKey && Date.now() < statsSummaryCache.expiresAt)
    ? statsSummaryCache
    : null

  const statsCount = ref<StatsCounts>(
    cachedEntry
      ? { ...cachedEntry.counts }
      : { authorized: 0, local: 0, pending: 0, all: 0 }
  )

  const selectedHospital = ref(0)
  const selectedDataType = ref('ALL')
  const searchKeyword = ref('')
  const results = ref<MedicalRecordRow[]>([])
  const loading = ref(false)

  let isUnmounted = false
  let searchSeq = 0

  // 解析结果缓存，避免重复计算
  const parsedExamCache = new Map<string, ParsedMedicalReport>()

  function getParsedExam(exam: MedicalRecordRow): ParsedMedicalReport {
    if (!exam) return { hasItems: false, sections: [], summaryTags: [], isAnyAbnormal: false, hasKeywordMatch: false }
    const cacheKey = `${exam.id}_${exam.exam_result || ''}_${exam.diagnosis || ''}_${searchKeyword.value}`
    if (parsedExamCache.has(cacheKey)) {
      return parsedExamCache.get(cacheKey)!
    }
    const text = exam.exam_result || exam.diagnosis || exam.symptoms || ''
    const parsed = parseMedicalExamResult(text, searchKeyword.value)
    parsedExamCache.set(cacheKey, parsed)
    return parsed
  }

  function getExamTableSummary(row: MedicalRecordRow): string {
    const parsed = getParsedExam(row)
    if (parsed.hasItems && parsed.sections.length > 0) {
      return parsed.sections.map(s => `${s.title}: ${s.conclusion}`).join('； ')
    }
    return row.diagnosis || row.symptoms || row.exam_result || '常规就诊记录'
  }

  function initFromRouteQuery() {
    if (route.query.id_card) {
      searchKeyword.value = String(route.query.id_card)
      selectedHospital.value = 0
      if (route.query.patient_name) {
        ElNotification({
          title: '已自动填入患者身份证号',
          message: `已自动载入患者【${route.query.patient_name}】身份证号，正在检索全网既往就诊记录...`,
          type: 'info',
          duration: 4000
        })
      }
    } else if (route.query.keyword) {
      searchKeyword.value = String(route.query.keyword)
    }
  }

  function writeStatsCache(counts: StatsCounts) {
    statsCount.value = counts
    statsSummaryCache = {
      key: `${auth.user?.id || 0}_${auth.user?.hospital_id || 1}`,
      counts: { ...counts },
      expiresAt: Date.now() + STATS_CACHE_TTL_MS,
    }
  }

  async function fetchStatsSummary(force = false) {
    if (isUnmounted) return
    const key = `${auth.user?.id || 0}_${auth.user?.hospital_id || 1}`
    if (!force && statsSummaryCache && statsSummaryCache.key === key && Date.now() < statsSummaryCache.expiresAt) {
      statsCount.value = { ...statsSummaryCache.counts }
      return
    }
    try {
      const currentHospId = auth.user?.hospital_id || 1
      const currentDocId = auth.user?.id || 0
      const res = await listRecordsApi()
      if (isUnmounted) return
      if (res.code === 200 && res.data) {
        const counts = computeStatsCounts(res.data || [], currentHospId, currentDocId)
        writeStatsCache(counts)
      }
    } catch (err) {
      console.warn('fetchStatsSummary error', err)
    }
  }

  async function searchCrossRecords(gatePromise?: Promise<void>) {
    const seq = ++searchSeq
    let settled = !gatePromise
    if (settled) {
      loading.value = true
    } else {
      void gatePromise!.then(() => {
        settled = true
        if (!isUnmounted && seq === searchSeq && loadingAfterGatePending) {
          loading.value = true
        }
      })
    }
    let loadingAfterGatePending = !settled
    parsedExamCache.clear()
    try {
      const scope = currentScope.value
      const currentHospId = auth.user?.hospital_id || 1
      const currentDocId = auth.user?.id || 0
      const params: Record<string, unknown> = {
        keyword: searchKeyword.value,
        data_type: selectedDataType.value
      }

      if (scope === 'authorized') {
        params.exclude_hospital_id = currentHospId
      } else if (scope === 'local') {
        params.hospital_id = currentHospId
      } else if (scope === 'pending') {
        params.exclude_hospital_id = currentHospId
      } else if (scope === 'all' && selectedHospital.value > 0) {
        params.hospital_id = selectedHospital.value
      }

      const res = await listRecordsApi(params)
      const isFullUnfilteredQuery =
        scope === 'all' &&
        selectedHospital.value === 0 &&
        selectedDataType.value === 'ALL' &&
        !searchKeyword.value.trim()

      // 若仍在执行 250ms 入场动画，等待入场动画结束后再将数据写入表格，避免中途重排掉帧
      if (gatePromise) {
        await gatePromise
      }
      loadingAfterGatePending = false
      if (isUnmounted || seq !== searchSeq) return

      if (res.code === 200) {
        const rawList = (res.data || []) as MedicalRecordRow[]
        const filtered = filterRowsByScope(rawList, scope, currentHospId, currentDocId)
        if (isFullUnfilteredQuery) {
          writeStatsCache(computeStatsCounts(rawList, currentHospId, currentDocId))
        } else if (selectedHospital.value === 0 && selectedDataType.value === 'ALL' && !searchKeyword.value.trim()) {
          // 同步当前范围的真实最新计数，避免新建病历切入时缓存未过期导致数量短暂不一致
          const nextCounts: StatsCounts = { ...statsCount.value, [scope]: filtered.length }
          writeStatsCache(nextCounts)
        }
        results.value = filtered
      }
    } catch (err) {
      loadingAfterGatePending = false
      console.error(err)
    } finally {
      loadingAfterGatePending = false
      if (!isUnmounted && seq === searchSeq) {
        loading.value = false
      }
    }
  }

  function refreshData() {
    statsSummaryCache = null
    fetchStatsSummary(true)
    searchCrossRecords()
  }

  // 路由驱动的自动重检索：离开期间旧组件仍停留 250ms 播完退场动画，仅当仍停留在同一挂载路径时响应变化
  watch(() => route.params.scope, () => {
    if (isUnmounted || route.path !== mountedPath) return
    searchCrossRecords()
    fetchStatsSummary()
  })

  watch(() => route.query, () => {
    if (isUnmounted || route.path !== mountedPath) return
    initFromRouteQuery()
    searchCrossRecords()
    fetchStatsSummary()
  })

  initFromRouteQuery()

  onMounted(() => {
    const enterGate = options?.waitForEnterTransition?.()
    const isFullUnfilteredAll =
      currentScope.value === 'all' &&
      selectedHospital.value === 0 &&
      selectedDataType.value === 'ALL' &&
      !searchKeyword.value.trim()

    // 在挂载首帧即并发发起网络请求，与 250ms CSS 入场动画重叠执行；结果在入场完成后一次性落表
    void searchCrossRecords(enterGate)
    if (!isFullUnfilteredAll) {
      void Promise.resolve(enterGate).then(() => {
        if (!isUnmounted) {
          void fetchStatsSummary(false)
        }
      })
    }
  })

  onBeforeUnmount(() => {
    isUnmounted = true
    searchSeq++
  })

  return {
    statsCount,
    selectedHospital,
    selectedDataType,
    searchKeyword,
    results,
    loading,
    getParsedExam,
    getExamTableSummary,
    initFromRouteQuery,
    fetchStatsSummary,
    searchCrossRecords,
    refreshData,
  }
}