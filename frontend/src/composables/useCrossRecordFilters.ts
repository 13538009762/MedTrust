// WP-16: 跨院检索快捷归类与多维细筛/分页状态 (从 CrossQueryView 迁入, 交互行为保持原样)
import { ref, computed, watch, type Ref } from 'vue'
import { ElMessage } from 'element-plus'
import { useAuthStore } from '@/stores/auth'
import type { ParsedMedicalReport } from '@/utils/medicalReportParser'
import type { MedicalRecordRow } from '@/types/clinical'
import type { RouteLocationNormalizedLoaded } from 'vue-router'

export function useCrossRecordFilters(
  results: Ref<MedicalRecordRow[]>,
  getParsedExam: (exam: MedicalRecordRow) => ParsedMedicalReport,
  searchKeyword: Ref<string>,
  route: RouteLocationNormalizedLoaded
) {
  const auth = useAuthStore()

  const quickClassifyMode = ref<'ALL' | 'ATTENTION' | 'ACCESSIBLE' | 'NEED_AUTH' | 'EXAM_ONLY'>('ALL')

  const filterAuthStatus = ref<'ALL' | 'SELF' | 'HOSPITAL' | 'AUTHORIZED' | 'PENDING' | 'BREAK_GLASS'>('ALL')
  const filterClinicalStatus = ref<'ALL' | 'ABNORMAL' | 'NORMAL'>('ALL')
  const filterTamperStatus = ref<'ALL' | 'SAFE' | 'TAMPERED'>('ALL')
  const filterTimeRange = ref<'ALL' | '3D' | '7D' | '30D' | '90D'>('ALL')

  const activeFilterCount = computed(() => {
    let count = 0
    if (quickClassifyMode.value !== 'ALL') count++
    if (filterAuthStatus.value !== 'ALL') count++
    if (filterClinicalStatus.value !== 'ALL') count++
    if (filterTamperStatus.value !== 'ALL') count++
    if (filterTimeRange.value !== 'ALL') count++
    return count
  })

  function resetAllFilters() {
    quickClassifyMode.value = 'ALL'
    filterAuthStatus.value = 'ALL'
    filterClinicalStatus.value = 'ALL'
    filterTamperStatus.value = 'ALL'
    filterTimeRange.value = 'ALL'
    ElMessage.info('已重置所有列表筛选条件')
  }

  // 组合过滤后的最终表格数据集
  const filteredResults = computed(() => {
    return results.value.filter((row: MedicalRecordRow) => {
      // 1. 快捷归类模式判断
      if (quickClassifyMode.value === 'ATTENTION') {
        const parsed = getParsedExam(row)
        if (!parsed.isAnyAbnormal && !row.is_tampered) return false
      } else if (quickClassifyMode.value === 'ACCESSIBLE') {
        const isSelf = row.doctor_id === auth.user?.id
        const isHosp = row.hospital_id === auth.user?.hospital_id
        const isAuth = row.has_access || row.access_type === 'BREAK_GLASS'
        if (!isSelf && !isHosp && !isAuth) return false
      } else if (quickClassifyMode.value === 'NEED_AUTH') {
        const isSelf = row.doctor_id === auth.user?.id
        const isHosp = row.hospital_id === auth.user?.hospital_id
        const isAuth = row.has_access || row.access_type === 'BREAK_GLASS'
        if (isSelf || isHosp || isAuth) return false
      } else if (quickClassifyMode.value === 'EXAM_ONLY') {
        const isExam = row.data_type === 'REPORT' || row.data_type === 'IMAGE' || row.need_exam || row.exam_items || row.exam_result
        if (!isExam) return false
      }

      // 2. 授权状态细筛
      if (filterAuthStatus.value === 'SELF') {
        if (row.doctor_id !== auth.user?.id) return false
      } else if (filterAuthStatus.value === 'HOSPITAL') {
        if (row.hospital_id !== auth.user?.hospital_id || row.doctor_id === auth.user?.id) return false
      } else if (filterAuthStatus.value === 'AUTHORIZED') {
        if (!row.has_access || row.hospital_id === auth.user?.hospital_id) return false
      } else if (filterAuthStatus.value === 'PENDING') {
        if (row.has_access || row.access_type === 'BREAK_GLASS' || row.hospital_id === auth.user?.hospital_id || row.doctor_id === auth.user?.id) return false
      } else if (filterAuthStatus.value === 'BREAK_GLASS') {
        if (row.access_type !== 'BREAK_GLASS') return false
      }

      // 3. 临床异常细筛
      if (filterClinicalStatus.value === 'ABNORMAL') {
        const parsed = getParsedExam(row)
        if (!parsed.isAnyAbnormal) return false
      } else if (filterClinicalStatus.value === 'NORMAL') {
        const parsed = getParsedExam(row)
        if (parsed.isAnyAbnormal) return false
      }

      // 4. 防篡改状态细筛
      if (filterTamperStatus.value === 'SAFE' && row.is_tampered) return false
      if (filterTamperStatus.value === 'TAMPERED' && !row.is_tampered) return false

      // 5. 就诊时间范围
      if (filterTimeRange.value !== 'ALL' && row.created_at) {
        try {
          const rowTime = new Date(row.created_at).getTime()
          const now = Date.now()
          const diffDays = (now - rowTime) / (1000 * 3600 * 24)
          if (filterTimeRange.value === '3D' && diffDays > 3) return false
          if (filterTimeRange.value === '7D' && diffDays > 7) return false
          if (filterTimeRange.value === '30D' && diffDays > 30) return false
          if (filterTimeRange.value === '90D' && diffDays > 90) return false
        } catch {
          // ignore date parse errors
        }
      }

      return true
    })
  })

  const currentPage = ref(1)
  const pageSize = ref(10)

  const paginatedResults = computed(() => {
    const start = (currentPage.value - 1) * pageSize.value
    return filteredResults.value.slice(start, start + pageSize.value)
  })

  watch([() => route.params.scope, quickClassifyMode, filterAuthStatus, filterClinicalStatus, filterTamperStatus, filterTimeRange, searchKeyword], () => {
    currentPage.value = 1
  })

  function handleSizeChange(val: number) {
    pageSize.value = val
    currentPage.value = 1
  }

  function handleCurrentChange(val: number) {
    currentPage.value = val
  }

  return {
    quickClassifyMode,
    filterAuthStatus,
    filterClinicalStatus,
    filterTamperStatus,
    filterTimeRange,
    activeFilterCount,
    resetAllFilters,
    filteredResults,
    currentPage,
    pageSize,
    paginatedResults,
    handleSizeChange,
    handleCurrentChange,
  }
}