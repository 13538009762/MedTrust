// WP-16: AI 跨期全景病程推演弹窗状态 (从 CrossQueryView 迁入, 逻辑与默认值保持原样)
import { ref, type Ref } from 'vue'
import type { RouteLocationNormalizedLoaded } from 'vue-router'
import type { MedicalRecordRow, CrossTargetPatient } from '@/types/clinical'

export function useCrossAIDiagnostic(
  route: RouteLocationNormalizedLoaded,
  results: Ref<MedicalRecordRow[]>
) {
  const crossAIDiagnosticVisible = ref(false)
  const crossTargetPatient = ref<CrossTargetPatient>({
    id: 0,
    real_name: '',
    user_no: '',
    gender: '男',
    age: 45
  })

  function openCrossAIDiagnostic() {
    let pid = Number(route.query.patient_id) || 0
    let pname = (route.query.patient_name as string) || ''

    if (!pid && results.value?.length > 0) {
      const firstRec = results.value[0]
      pid = firstRec.patient_id || 0
      pname = pname || firstRec.patient_name || ''
    }

    if (!pid) {
      // 尝试根据当前搜索关键词匹配患者
      pid = 1 // 默认张三患者
      pname = pname || '张三'
    }

    crossTargetPatient.value = {
      id: pid,
      real_name: pname || '张三',
      user_no: (route.query.id_card as string) || 'PAT_001',
      gender: '男',
      age: 45
    }

    crossAIDiagnosticVisible.value = true
  }

  return { crossAIDiagnosticVisible, crossTargetPatient, openCrossAIDiagnostic }
}