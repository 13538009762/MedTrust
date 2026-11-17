// WP-16: 跨院检索表格自定义列持久化记忆 (从 CrossQueryView 迁入, 行为与提示文案保持原样)
import { ref, computed } from 'vue'
import { ElMessage } from 'element-plus'
import type { TableColumnItem } from '@/types/clinical'

const DEFAULT_COLUMNS: TableColumnItem[] = [
  { key: 'record_no', label: '病历单号', visible: true, width: '160' },
  { key: 'patient', label: '就诊患者', visible: true, minWidth: '180' },
  { key: 'hospital', label: '归属医疗机构', visible: true, minWidth: '180' },
  { key: 'doctor', label: '开具医生', visible: true, width: '130' },
  { key: 'data_type', label: '类别', visible: true, width: '105' },
  { key: 'course', label: '发病与病程', visible: true, width: '140' },
  { key: 'diagnosis', label: '检查项目与临床诊断', visible: true, minWidth: '240' },
  { key: 'tamper', label: '防篡改状态', visible: true, width: '125' },
  { key: 'auth_status', label: '调阅授权状态', visible: true, width: '145' },
  { key: 'date', label: '就诊日期', visible: true, width: '120' },
  { key: 'action', label: '调阅操作', visible: true, width: '180', fixed: 'right' },
]

const STORAGE_KEY_COLUMNS = 'medtrust_doctor_cross_columns_v1'

export function useCrossTableColumns() {
  const tableColumns = ref<TableColumnItem[]>(loadColumnsConfig())

  function loadColumnsConfig(): TableColumnItem[] {
    try {
      const saved = localStorage.getItem(STORAGE_KEY_COLUMNS)
      if (saved) {
        const parsed: Record<string, boolean> = JSON.parse(saved)
        return DEFAULT_COLUMNS.map(col => ({
          ...col,
          visible: parsed[col.key] !== undefined ? parsed[col.key] : col.visible
        }))
      }
    } catch (e) {
      console.warn('Failed to load column config from localStorage', e)
    }
    return DEFAULT_COLUMNS.map(c => ({ ...c }))
  }

  function saveColumnsConfig() {
    try {
      const map: Record<string, boolean> = {}
      tableColumns.value.forEach(col => {
        map[col.key] = col.visible
      })
      localStorage.setItem(STORAGE_KEY_COLUMNS, JSON.stringify(map))
    } catch (e) {
      console.warn('Failed to save column config to localStorage', e)
    }
  }

  function isColumnVisible(key: string): boolean {
    const found = tableColumns.value.find(c => c.key === key)
    return found ? found.visible : true
  }

  function resetColumns() {
    tableColumns.value = DEFAULT_COLUMNS.map(c => ({ ...c }))
    saveColumnsConfig()
    ElMessage.success('已恢复表格默认列显示')
  }

  function selectAllColumns() {
    tableColumns.value.forEach(c => { c.visible = true })
    saveColumnsConfig()
    ElMessage.success('已显示全部列')
  }

  const visibleColumnsCount = computed(() => {
    return tableColumns.value.filter(c => c.visible).length
  })

  return { tableColumns, saveColumnsConfig, isColumnVisible, resetColumns, selectAllColumns, visibleColumnsCount }
}