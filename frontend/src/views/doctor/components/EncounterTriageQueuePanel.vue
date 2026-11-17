<template>
  <!-- 视图一：就诊记录列表与统计看板 (LIST) — WP-16: 从 DoctorRecordsView 抽取 -->
  <div class="records-list-wrapper">
    <!-- 顶部状态统计卡片 -->
    <div class="stats-grid">
      <div class="stat-card blue">
        <div class="stat-icon"><el-icon><OfficeBuilding /></el-icon></div>
        <div class="stat-info">
          <div class="stat-value">{{ records.length }}</div>
          <div class="stat-label">累计接诊记录</div>
        </div>
      </div>
      <div class="stat-card yellow">
        <div class="stat-icon"><el-icon><Timer /></el-icon></div>
        <div class="stat-info">
          <div class="stat-value">{{ waitingExamCount }}</div>
          <div class="stat-label">待医技检查 (申请中)</div>
        </div>
      </div>
      <div class="stat-card cyan">
        <div class="stat-icon"><el-icon><Document /></el-icon></div>
        <div class="stat-info">
          <div class="stat-value">{{ examCompletedCount }}</div>
          <div class="stat-label">检查完成 (待下确诊)</div>
        </div>
      </div>
      <div class="stat-card green">
        <div class="stat-icon"><el-icon><CircleCheckFilled /></el-icon></div>
        <div class="stat-info">
          <div class="stat-value">{{ completedCount }}</div>
          <div class="stat-label">已存证归档病历</div>
        </div>
      </div>
    </div>

    <!-- 主卡片 -->
    <el-card class="main-card" shadow="never">
      <template #header>
        <div class="card-header">
          <div class="header-left">
            <span class="header-title">临床就诊记录管理 (Encounter Management)</span>
            <el-tag type="info" size="small" effect="plain" class="ml-2">就诊生命周期驱动</el-tag>
          </div>
          <div class="header-actions">
            <el-button
              v-if="hasSavedDraft"
              type="warning"
              plain
              :icon="Document"
              @click="$emit('restore-draft')"
            >
              恢复未完成草稿【{{ draftPatientName }}】
            </el-button>
            <el-button type="danger" plain @click="$router.push('/doctor/patient-search')">
              <el-icon style="margin-right: 4px;"><FirstAidKit /></el-icon>
              患者检索与急救调阅
            </el-button>
            <el-button type="primary" :icon="Plus" @click="$emit('create-encounter')">
              新建就诊记录 (接诊初诊)
            </el-button>
            <el-button :icon="Refresh" circle @click="$emit('refresh')" />
          </div>
        </div>
      </template>

      <!-- 搜索与状态过滤器 -->
      <div class="table-toolbar">
        <el-radio-group v-model="statusFilterModel" size="small">
          <el-radio-button label="ALL">全部 ({{ records.length }})</el-radio-button>
          <el-radio-button label="WAITING_EXAM">待医技检查 ({{ waitingExamCount }})</el-radio-button>
          <el-radio-button label="EXAM_COMPLETED">待下最终确诊 ({{ examCompletedCount }})</el-radio-button>
          <el-radio-button label="COMPLETED">已归档存证 ({{ completedCount }})</el-radio-button>
        </el-radio-group>

        <el-input
          v-model="searchKeywordModel"
          placeholder="检索患者姓名、就诊单号、初诊/确诊或检查项目..."
          style="width: 320px;"
          clearable
          size="small"
        >
          <template #prefix><el-icon><Search /></el-icon></template>
        </el-input>
      </div>

      <!-- 病历表格 -->
      <el-table
        :data="pagedRecords"
        v-loading="loading"
        stripe
        class="doctor-records-table is-timeline-clickable"
        style="width: 100%"
        @row-click="handleTableRowClick"
      >
        <el-table-column prop="record_no" label="就诊编号" width="190">
          <template #default="{ row }">
            <span class="mono font-bold" style="white-space: nowrap;">{{ row.record_no }}</span>
          </template>
        </el-table-column>

        <el-table-column label="就诊患者" width="165">
          <template #default="{ row }">
            <div style="display: flex; align-items: center; gap: 4px;">
              <el-tooltip content="点击打开右侧抽屉查看该患者既往病例时间线" placement="top">
                <button
                  type="button"
                  class="patient-timeline-link"
                  @click.stop="openPatientTimelineDrawer(row)"
                >
                  <el-icon class="ptl-icon"><Clock /></el-icon>
                  <span class="font-bold">{{ formatPatientName(row) }}</span>
                </button>
              </el-tooltip>
              <el-tooltip
                v-if="row.infection_alert?.has_risk"
                :content="`医护安全预警：该患者有传染病携带史（${row.infection_alert.summary}）`"
                placement="top"
              >
                <el-tag size="small" type="danger" effect="dark" style="font-size: 10px; padding: 0 4px; height: 18px; line-height: 16px; cursor: pointer;" @click.stop="$emit('view-detail', row)">
                  防护预警
                </el-tag>
              </el-tooltip>
            </div>
          </template>
        </el-table-column>

        <el-table-column prop="encounter_type" label="就诊类型" width="110">
          <template #default="{ row }">
            <el-tag size="small" :type="row.encounter_type === 'EMERGENCY' ? 'danger' : 'info'">
              {{ formatEncounterType(row.encounter_type) }}
            </el-tag>
          </template>
        </el-table-column>

        <el-table-column prop="department_name" label="就诊科室" width="130" />

        <el-table-column label="临床初步诊断 / 确诊结论" min-width="200">
          <template #default="{ row }">
            <div v-if="row.diagnosis" class="text-success font-bold">
              {{ row.diagnosis }}
            </div>
            <div v-else-if="row.initial_diagnosis" class="text-primary">
              <span class="text-xs text-gray-500">初诊拟诊:</span> {{ row.initial_diagnosis }}
            </div>
            <div v-else class="gray-400-text">
              {{ row.chief_complaint || '记录中' }}
            </div>
          </template>
        </el-table-column>

        <el-table-column label="医技检查项目" min-width="170">
          <template #default="{ row }">
            <template v-if="row.need_exam || row.exam_items">
              <el-tag size="small" type="warning">{{ row.exam_items || '辅助检查' }}</el-tag>
            </template>
            <span v-else class="text-gray-400 text-xs">无医技检查 (直接确诊)</span>
          </template>
        </el-table-column>

        <el-table-column label="就诊流转状态" width="140">
          <template #default="{ row }">
            <ChainVerifyBadge v-if="row.is_tampered" status="tampered" label="存在篡改" />
            <el-tag v-else-if="row.status === 'WAITING_EXAM'" type="warning" effect="dark">待医技检查</el-tag>
            <el-tag v-else-if="row.status === 'PROCESSING_EXAM'" type="primary" effect="dark">检查进行中</el-tag>
            <el-tag v-else-if="row.status === 'EXAM_COMPLETED'" type="info" effect="dark" class="status-ready-tag">检查完成/待确诊</el-tag>
            <el-tag v-else-if="row.status === 'INITIAL_DIAGNOSIS'" type="info">初诊完成/待确诊</el-tag>
            <el-tag v-else-if="row.status === 'COMPLETED' || row.status === '已接诊' || row.status === '已归档存证' || row.diagnosis" type="success" effect="plain">已归档存证</el-tag>
            <el-tag v-else size="small">{{ row.status }}</el-tag>
          </template>
        </el-table-column>

        <el-table-column prop="created_at" label="就诊时间" width="170">
          <template #default="{ row }">
            <span style="white-space: nowrap;">{{ formatTime(row.created_at) }}</span>
          </template>
        </el-table-column>

        <el-table-column label="操作" width="180" :fixed="isWideScreen ? 'right' : false">
          <template #default="{ row }">
            <!-- 检查回传完成，等待医生出最终诊断 -->
            <el-button
              v-if="row.status === 'EXAM_COMPLETED' || row.status === 'INITIAL_DIAGNOSIS'"
              type="danger"
              size="small"
              @click="$emit('open-final', row)"
            >
              下达最终诊断
            </el-button>

            <!-- 检查进行中 -->
            <el-button
              v-else-if="row.status === 'WAITING_EXAM' || row.status === 'PROCESSING_EXAM'"
              type="primary"
              size="small"
              plain
              @click="$emit('view-detail', row)"
            >
              查看检查进度
            </el-button>

            <!-- 已归档存证 -->
            <el-button
              v-else
              type="success"
              size="small"
              plain
              @click="$emit('view-detail', row)"
            >
              调阅全景病历
            </el-button>
          </template>
        </el-table-column>
      </el-table>

      <!-- 表格分页控制 (一页最多显示10行) -->
      <div class="table-pagination-wrapper">
        <span class="pagination-info text-muted">
          共 {{ filteredRecords.length }} 条就诊记录，当前第 {{ currentPage }} / {{ Math.max(1, Math.ceil(filteredRecords.length / pageSize)) }} 页
        </span>
        <el-pagination
          v-model:current-page="pageModel"
          v-model:page-size="sizeModel"
          :page-sizes="[10, 20, 50]"
          :total="filteredRecords.length"
          layout="total, sizes, prev, pager, next, jumper"
          background
          size="default"
        />
      </div>
    </el-card>

    <!-- 右侧病例时间线抽屉（点击候诊患者时非破坏式预览既往就诊史） -->
    <el-drawer
      v-model="timelineDrawerVisible"
      direction="rtl"
      size="540px"
      append-to-body
      destroy-on-close
    >
      <template #header>
        <div class="timeline-drawer-head">
          <div class="tdh-title-row">
            <span class="tdh-title">
              {{ activeDrawerRecord ? formatPatientName(activeDrawerRecord) : '患者' }} · 既往病例时间线
            </span>
            <el-tag
              v-if="activeDrawerRecord?.infection_alert?.has_risk"
              size="small"
              type="danger"
              effect="dark"
            >
              防护预警
            </el-tag>
          </div>
          <div class="tdh-sub">
            预载近 15 条跨院及本院归档病历，按时间倒序展示病因、症状现象、诊断与链上核验指纹
          </div>
        </div>
      </template>

      <div class="timeline-drawer-body">
        <div class="tdh-toolbar">
          <span v-if="activeDrawerRecord?.record_no" class="mono tdh-current-no">
            当前候诊单号：{{ activeDrawerRecord.record_no }}
          </span>
          <el-button
            size="small"
            plain
            :icon="Refresh"
            :loading="drawerLoading"
            @click="refreshDrawerTimeline"
          >
            刷新时间线
          </el-button>
        </div>

        <PatientClinicalTimeline
          :records="drawerRecords"
          :loading="drawerLoading"
          :compact="false"
          :allow-adopt="false"
          :allow-request-access="false"
          :current-hospital-id="auth.user?.hospital_id"
          empty-description="该患者暂无可展示的既往病例时间线记录"
          @view-detail="handleDrawerViewDetail"
        />
      </div>
    </el-drawer>
  </div>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { Plus, Refresh, Search, Document, OfficeBuilding, Timer, CircleCheckFilled, FirstAidKit, Clock } from '@element-plus/icons-vue'
import ChainVerifyBadge from '@/components/common/ChainVerifyBadge.vue'
import PatientClinicalTimeline from '@/components/clinical/PatientClinicalTimeline.vue'
import { usePatientTimeline } from '@/composables/usePatientTimeline'
import { useAuthStore } from '@/stores/auth'
import { formatEncounterType, formatTime } from '@/utils/recordFormatters'
import type { MedicalRecordRow } from '@/types/clinical'

const props = defineProps<{
  records: MedicalRecordRow[]
  filteredRecords: MedicalRecordRow[]
  pagedRecords: MedicalRecordRow[]
  loading: boolean
  activeStatusFilter: string
  searchKeyword: string
  currentPage: number
  pageSize: number
  waitingExamCount: number
  examCompletedCount: number
  completedCount: number
  isWideScreen: boolean
  hasSavedDraft: boolean
  draftPatientName: string
  formatPatientName: (rec?: MedicalRecordRow) => string
}>()

const emit = defineEmits<{
  (e: 'update:activeStatusFilter', val: string): void
  (e: 'update:searchKeyword', val: string): void
  (e: 'update:currentPage', val: number): void
  (e: 'update:pageSize', val: number): void
  (e: 'create-encounter'): void
  (e: 'restore-draft'): void
  (e: 'refresh'): void
  (e: 'open-final', row: MedicalRecordRow): void
  (e: 'view-detail', row: MedicalRecordRow): void
}>()

const statusFilterModel = computed({
  get: () => props.activeStatusFilter,
  set: (val: string) => emit('update:activeStatusFilter', val),
})
const searchKeywordModel = computed({
  get: () => props.searchKeyword,
  set: (val: string) => emit('update:searchKeyword', val),
})
const pageModel = computed({
  get: () => props.currentPage,
  set: (val: number) => emit('update:currentPage', val),
})
const sizeModel = computed({
  get: () => props.pageSize,
  set: (val: number) => emit('update:pageSize', val),
})

const auth = useAuthStore()
const timelineDrawerVisible = ref(false)
const activeDrawerRecord = ref<MedicalRecordRow | null>(null)
const {
  timelineRecords: drawerRecords,
  timelineLoading: drawerLoading,
  loadTimeline: loadDrawerTimeline
} = usePatientTimeline(15)

function openPatientTimelineDrawer(row: MedicalRecordRow) {
  activeDrawerRecord.value = row
  timelineDrawerVisible.value = true
  void loadDrawerTimeline(row.patient_id)
}

function refreshDrawerTimeline() {
  if (activeDrawerRecord.value?.patient_id) {
    void loadDrawerTimeline(activeDrawerRecord.value.patient_id)
  }
}

function handleTableRowClick(row: MedicalRecordRow, _column: unknown, event: Event) {
  const target = event?.target as HTMLElement | null
  if (target?.closest('button, a, .el-button, .el-tag')) return
  openPatientTimelineDrawer(row)
}

function handleDrawerViewDetail(rec: MedicalRecordRow) {
  timelineDrawerVisible.value = false
  emit('view-detail', rec)
}
</script>
<style scoped src="../DoctorRecordsView.css"></style>
<style scoped>
.patient-timeline-link {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  border: 1px solid transparent;
  background: transparent;
  padding: 2px 6px;
  border-radius: var(--mt-radius-xs, 4px);
  color: var(--mt-primary, #0284c7);
  cursor: pointer;
  transition: background-color 0.15s ease, border-color 0.15s ease;
}

.patient-timeline-link:hover {
  background: var(--mt-primary-soft, #f0f9ff);
  border-color: var(--mt-primary-border, #bae6fd);
}

.ptl-icon {
  font-size: 13px;
  flex-shrink: 0;
}

.is-timeline-clickable :deep(.el-table__row) {
  cursor: pointer;
}

.timeline-drawer-head {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.tdh-title-row {
  display: flex;
  align-items: center;
  gap: 8px;
}

.tdh-title {
  font-size: 16px;
  font-weight: 700;
  color: var(--mt-text-primary, #0f172a);
}

.tdh-sub {
  font-size: 12px;
  color: var(--mt-text-secondary, #64748b);
  line-height: 1.45;
}

.timeline-drawer-body {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.tdh-toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  padding: 8px 10px;
  background: var(--mt-bg-canvas, #f8fafc);
  border: 1px solid var(--mt-border-base, #e2e8f0);
  border-radius: var(--mt-radius-sm, 6px);
}

.tdh-current-no {
  font-size: 12px;
  color: var(--mt-text-secondary, #64748b);
}
</style>
