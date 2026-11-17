<template>
  <!-- 跨院记录结果列表与多维筛选工作台 (WP-16: 从 CrossQueryView 抽取) -->
  <el-card shadow="hover" class="box-card table-wrapper-card">
    <template #header>
      <div class="table-card-header">
        <div class="tch-left">
          <strong>跨机构就诊与医技病历列表</strong>
          <el-tag size="small" type="info" class="ml-2">
            当前展示 {{ filteredResults.length }} / {{ results.length }} 项
          </el-tag>
        </div>

        <!-- 右侧：自定义列按钮与清空筛选 -->
        <div class="tch-right">
          <el-button
            v-if="activeFilterCount > 0"
            size="small"
            type="danger"
            plain
            @click="$emit('reset-filters')"
          >
            清除筛选 ({{ activeFilterCount }})
          </el-button>

          <!-- 自定义列浮层面板 (Popover) -->
          <el-popover
            placement="bottom-end"
            :width="270"
            trigger="click"
            popper-class="column-setting-popover"
          >
            <template #reference>
              <el-button size="small" :icon="Setting" plain>
                自定义列 ({{ visibleColumnsCount }}/{{ tableColumns.length }})
              </el-button>
            </template>

            <div class="col-popover-content">
              <div class="col-popover-head">
                <strong>自定义表格显示列</strong>
                <span class="text-xs text-slate-500">（已实时本地保存）</span>
              </div>
              <div class="col-checkbox-list">
                <div
                  v-for="col in tableColumns"
                  :key="col.key"
                  class="col-checkbox-item"
                >
                  <el-checkbox
                    v-model="col.visible"
                    @change="$emit('save-columns')"
                  >
                    <span>{{ col.label }}</span>
                  </el-checkbox>
                </div>
              </div>
              <div class="col-popover-footer">
                <el-button size="small" link type="primary" @click="$emit('select-all-columns')">全选</el-button>
                <el-button size="small" link type="info" @click="$emit('reset-columns')">恢复默认</el-button>
              </div>
            </div>
          </el-popover>
        </div>
      </div>
    </template>

    <!-- 智能筛选与快捷归类工作台 -->
    <div class="filter-workbench mb-3">
      <!-- 第 1 栏：快捷归类场景模式 -->
      <div class="workbench-row classify-row">
        <span class="wb-label">快捷归类：</span>
        <el-radio-group v-model="classifyModel" size="small" class="classify-tabs">
          <el-radio-button label="ALL">
            <span>全部档案 ({{ results.length }})</span>
          </el-radio-button>
          <el-radio-button label="ATTENTION">
            <span>重点关注 (异常/告警)</span>
          </el-radio-button>
          <el-radio-button label="ACCESSIBLE">
            <span>立即可查 (已放行)</span>
          </el-radio-button>
          <el-radio-button label="NEED_AUTH">
            <span>待安全网关核准</span>
          </el-radio-button>
          <el-radio-button label="EXAM_ONLY">
            <span>仅医技检查/切片</span>
          </el-radio-button>
        </el-radio-group>
      </div>

      <!-- 第 2 栏：多维度细筛下拉项组合 -->
      <div class="workbench-row filters-row">
        <div class="filter-item">
          <span class="f-lbl">调阅权限:</span>
          <el-select :model-value="filterAuthStatus" size="small" style="width: 145px;" @update:model-value="$emit('update:filterAuthStatus', $event)">
            <el-option label="全部状态" value="ALL" />
            <el-option label="本人责任病历" value="SELF" />
            <el-option label="本院同僚放行" value="HOSPITAL" />
            <el-option label="跨院已授权" value="AUTHORIZED" />
            <el-option label="待网关核准" value="PENDING" />
            <el-option label="24h破窗放行" value="BREAK_GLASS" />
          </el-select>
        </div>

        <div class="filter-item">
          <span class="f-lbl">临床指标:</span>
          <el-select :model-value="filterClinicalStatus" size="small" style="width: 130px;" @update:model-value="$emit('update:filterClinicalStatus', $event)">
            <el-option label="全部指标" value="ALL" />
            <el-option label="仅看阳性/异常" value="ABNORMAL" />
            <el-option label="仅看大致正常" value="NORMAL" />
          </el-select>
        </div>

        <div class="filter-item">
          <span class="f-lbl">防篡改状态:</span>
          <el-select :model-value="filterTamperStatus" size="small" style="width: 125px;" @update:model-value="$emit('update:filterTamperStatus', $event)">
            <el-option label="全部校验" value="ALL" />
            <el-option label="账本一致" value="SAFE" />
            <el-option label="存在篡改" value="TAMPERED" />
          </el-select>
        </div>

        <div class="filter-item">
          <span class="f-lbl">就诊时间:</span>
          <el-select :model-value="filterTimeRange" size="small" style="width: 120px;" @update:model-value="$emit('update:filterTimeRange', $event)">
            <el-option label="全部时间" value="ALL" />
            <el-option label="近 3 天" value="3D" />
            <el-option label="近 1 周" value="7D" />
            <el-option label="近 1 个月" value="30D" />
            <el-option label="近 3 个月" value="90D" />
          </el-select>
        </div>

        <!-- 活动标签展示 -->
        <div v-if="activeFilterCount > 0" class="active-filter-pills">
          <el-tag
            v-if="quickClassifyMode !== 'ALL'"
            size="small"
            closable
            type="danger"
            @close="classifyModel = 'ALL'"
          >
            归类: {{ quickClassifyMode === 'ATTENTION' ? '需重点关注' : (quickClassifyMode === 'ACCESSIBLE' ? '立即可查阅' : (quickClassifyMode === 'NEED_AUTH' ? '待核准' : '仅医技单')) }}
          </el-tag>
          <el-tag
            v-if="filterAuthStatus !== 'ALL'"
            size="small"
            closable
            type="warning"
            @close="$emit('update:filterAuthStatus', 'ALL')"
          >
            权限: {{ filterAuthStatus }}
          </el-tag>
          <el-tag
            v-if="filterClinicalStatus !== 'ALL'"
            size="small"
            closable
            type="primary"
            @close="$emit('update:filterClinicalStatus', 'ALL')"
          >
            指标: {{ filterClinicalStatus === 'ABNORMAL' ? '仅阳性/异常' : '仅正常' }}
          </el-tag>
          <el-tag
            v-if="filterTimeRange !== 'ALL'"
            size="small"
            closable
            type="info"
            @close="$emit('update:filterTimeRange', 'ALL')"
          >
            时间: {{ filterTimeRange }}
          </el-tag>
        </div>
      </div>
    </div>

    <!-- 数据表格绑定 paginatedResults，并按自定义列显隐动态呈现 -->
    <el-table
      :data="paginatedResults"
      v-loading="loading"
      stripe
      style="width: 100%"
      :max-height="tableMaxHeight"
      class="cross-query-table"
    >
      <el-table-column v-if="isColumnVisible('record_no')" prop="record_no" label="病历单号" width="160" />
      <el-table-column v-if="isColumnVisible('patient')" label="就诊患者" min-width="180">
        <template #default="{ row }">
          <div class="patient-cell">
            <span class="patient-name font-bold">{{ formatPatientName(row) }}</span>
            <div class="patient-sub">
              <span v-if="row.patient_phone">{{ row.patient_phone }}</span>
              <span v-if="row.patient_id_card" class="id-card-tag">{{ maskIDCard(row.patient_id_card) }}</span>
            </div>
          </div>
        </template>
      </el-table-column>

      <el-table-column v-if="isColumnVisible('hospital')" label="归属医疗机构" min-width="180">
        <template #default="{ row }">
          <div class="hosp-cell">
            <span class="hosp-name">{{ row.hospital_name }}</span>
            <el-tag
              v-if="row.doctor_id === userId"
              size="small"
              type="success"
              effect="light"
              class="tag-pill"
            >
              本人开具
            </el-tag>
            <el-tag
              v-else-if="row.hospital_id === userHospitalId"
              size="small"
              type="primary"
              effect="light"
              class="tag-pill"
            >
              本院同僚
            </el-tag>
            <el-tag
              v-else
              size="small"
              type="warning"
              effect="plain"
              class="tag-pill"
            >
              跨院机构
            </el-tag>
          </div>
        </template>
      </el-table-column>

      <el-table-column v-if="isColumnVisible('doctor')" label="开具医生" width="130">
        <template #default="{ row }">
          <span :class="{ 'self-doc-text': row.doctor_id === userId }">
            {{ row.doctor_name }}
            <span v-if="row.doctor_id === userId" class="self-tag">(您)</span>
          </span>
        </template>
      </el-table-column>

      <el-table-column v-if="isColumnVisible('data_type')" prop="data_type" label="类别" width="105">
        <template #default="{ row }">
          <el-tag v-if="row.data_type === 'IMAGE'" size="small" type="warning" effect="dark">影像切片</el-tag>
          <el-tag v-else-if="row.data_type === 'REPORT'" size="small" type="success" effect="dark">检验单</el-tag>
          <el-tag v-else-if="row.need_exam || row.exam_items" size="small" type="primary" effect="light">含检查</el-tag>
          <el-tag v-else size="small" type="info">门诊病历</el-tag>
        </template>
      </el-table-column>
      <el-table-column v-if="isColumnVisible('course')" label="发病与病程" width="140">
        <template #default="{ row }">
          <div v-if="row.onset_time || row.duration" style="font-size: 12px; line-height: 1.4;">
            <div>{{ row.onset_time ? row.onset_time.substring(0, 16) : '接诊记录' }}</div>
            <div style="color: #64748b;">{{ row.duration || '-' }}</div>
          </div>
          <span v-else style="color: #94a3b8; font-size: 12px;">常规就诊</span>
        </template>
      </el-table-column>
      <el-table-column v-if="isColumnVisible('diagnosis')" label="检查项目与临床诊断" min-width="240">
        <template #default="{ row }">
          <div v-if="row.exam_items" class="text-xs text-indigo-700 font-bold mb-1">
            检查: <span v-html="highlightText(row.exam_items, searchKeyword)"></span>
          </div>
          <div v-if="row.has_access || row.doctor_id === userId || row.hospital_id === userHospitalId" class="table-diag-box">
            <div v-if="getParsedExam(row).summaryTags.length" class="table-tags-row mb-1">
              <el-tag
                v-for="t in getParsedExam(row).summaryTags.slice(0, 2)"
                :key="t.text"
                size="small"
                :type="t.type"
                effect="light"
                class="mr-1"
              >
                {{ t.text }}
              </el-tag>
            </div>
            <div class="text-xs text-slate-800 table-summary-text" v-html="highlightText(getExamTableSummary(row), searchKeyword)"></div>
          </div>
          <div v-else class="text-xs text-amber-700 font-medium">
            敏感诊疗数据已受掩码保护 (点击【跨院调阅】核准)
          </div>
        </template>
      </el-table-column>
      <el-table-column v-if="isColumnVisible('tamper')" label="防篡改状态" width="125" align="center">
        <template #default="{ row }">
          <el-tag v-if="row.is_tampered" type="danger" effect="dark" class="tamper-tag-glow">
            存在篡改
          </el-tag>
          <el-tag v-else type="success" effect="light">
            校验一致
          </el-tag>
        </template>
      </el-table-column>

      <el-table-column v-if="isColumnVisible('auth_status')" label="调阅授权状态" width="145" align="center">
        <template #default="{ row }">
          <el-tag v-if="row.doctor_id === userId" type="success" effect="plain" size="small">
            本人责任病历
          </el-tag>
          <el-tag v-else-if="row.hospital_id === userHospitalId" type="primary" effect="plain" size="small">
            本院同僚放行
          </el-tag>
          <el-tag v-else-if="row.access_type === 'ALL_DOCTORS'" type="success" effect="dark" size="small">
            全网公开 (全体可见)
          </el-tag>
          <el-tag v-else-if="row.access_type === 'BREAK_GLASS'" type="danger" effect="dark" size="small">
            24h破窗放行中
          </el-tag>
          <el-tag v-else-if="row.has_access" type="success" effect="dark" size="small">
            患者跨院已授权
          </el-tag>
          <el-tag v-else type="info" effect="plain" size="small">
            待安全网关核准
          </el-tag>
        </template>
      </el-table-column>

      <el-table-column v-if="isColumnVisible('date')" prop="created_at" label="就诊日期" width="120">
        <template #default="{ row }">
          {{ row.created_at ? row.created_at.substring(0, 10) : '' }}
        </template>
      </el-table-column>
      <el-table-column v-if="isColumnVisible('action')" label="调阅操作" width="270" :fixed="isWideScreen ? 'right' : false">
        <template #default="{ row }">
          <el-button
            v-if="row.doctor_id === userId"
            type="success"
            size="small"
            @click="$emit('access', row)"
          >
            直接查看 (免申请)
          </el-button>
          <el-button
            v-else-if="row.hospital_id === userHospitalId"
            type="primary"
            size="small"
            @click="$emit('access', row)"
          >
            院内调阅 (直接放行)
          </el-button>
          <el-button
            v-else-if="row.access_type === 'ALL_DOCTORS'"
            type="success"
            size="small"
            @click="$emit('access', row)"
          >
            直接查看 (全体可见)
          </el-button>
          <el-button
            v-else-if="row.access_type === 'BREAK_GLASS'"
            type="danger"
            size="small"
            @click="$emit('access', row)"
          >
            直接查看 (破窗放行)
          </el-button>
          <el-button
            v-else-if="row.has_access"
            type="success"
            size="small"
            @click="$emit('access', row)"
          >
            直接查看 (已获授权)
          </el-button>
          <div v-else class="action-btn-group" style="display: flex; gap: 6px; white-space: nowrap;">
            <el-button
              type="warning"
              size="small"
              @click="$emit('access', row)"
            >
              跨院调阅 (安全网关)
            </el-button>
            <el-button
              type="danger"
              size="small"
              plain
              @click="$emit('break-glass', row)"
              title="患者危重抢救时快速触发 Break-Glass 破窗"
            >
              紧急访问 (破窗)
            </el-button>
          </div>
        </template>
      </el-table-column>
    </el-table>

    <!-- 表格底部：分页与快捷跳转栏 -->
    <div class="table-pagination-wrapper">
      <div class="tp-info text-xs text-slate-500">
        共 <strong class="text-slate-800">{{ filteredResults.length }}</strong> 条跨机构病历记录
        <span v-if="filteredResults.length !== results.length"> (已过滤筛选)</span>
      </div>
      <el-pagination
        v-model:current-page="pageModel"
        v-model:page-size="sizeModel"
        :page-sizes="[10, 15, 20, 50, 100]"
        :total="filteredResults.length"
        layout="sizes, prev, pager, next, jumper"
        background
        size="small"
        @size-change="$emit('size-change', $event)"
        @current-change="$emit('current-change', $event)"
      />
    </div>
  </el-card>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { Setting } from '@element-plus/icons-vue'
import { highlightText } from '@/utils/medicalReportParser'
import type { ParsedMedicalReport } from '@/utils/medicalReportParser'
import type { MedicalRecordRow, TableColumnItem } from '@/types/clinical'

type ClassifyMode = 'ALL' | 'ATTENTION' | 'ACCESSIBLE' | 'NEED_AUTH' | 'EXAM_ONLY'

const props = defineProps<{
  results: MedicalRecordRow[]
  filteredResults: MedicalRecordRow[]
  paginatedResults: MedicalRecordRow[]
  loading: boolean
  tableMaxHeight: string
  isWideScreen: boolean
  userId: number | undefined
  userHospitalId: number | undefined
  searchKeyword: string
  quickClassifyMode: ClassifyMode
  filterAuthStatus: string
  filterClinicalStatus: string
  filterTamperStatus: string
  filterTimeRange: string
  activeFilterCount: number
  tableColumns: TableColumnItem[]
  visibleColumnsCount: number
  currentPage: number
  pageSize: number
  formatPatientName: (rec?: any) => string
  maskIDCard: (idCard?: string) => string
  getParsedExam: (exam: any) => ParsedMedicalReport
  getExamTableSummary: (row: any) => string
  isColumnVisible: (key: string) => boolean
}>()

const emit = defineEmits<{
  (e: 'update:quickClassifyMode', val: ClassifyMode): void
  (e: 'update:filterAuthStatus', val: string): void
  (e: 'update:filterClinicalStatus', val: string): void
  (e: 'update:filterTamperStatus', val: string): void
  (e: 'update:filterTimeRange', val: string): void
  (e: 'update:currentPage', val: number): void
  (e: 'update:pageSize', val: number): void
  (e: 'access', row: MedicalRecordRow): void
  (e: 'break-glass', row: MedicalRecordRow): void
  (e: 'reset-filters'): void
  (e: 'save-columns'): void
  (e: 'select-all-columns'): void
  (e: 'reset-columns'): void
  (e: 'size-change', val: number): void
  (e: 'current-change', val: number): void
}>()

const classifyModel = computed({
  get: () => props.quickClassifyMode,
  set: (val: ClassifyMode) => emit('update:quickClassifyMode', val),
})
const pageModel = computed({
  get: () => props.currentPage,
  set: (val: number) => emit('update:currentPage', val),
})
const sizeModel = computed({
  get: () => props.pageSize,
  set: (val: number) => emit('update:pageSize', val),
})
</script>
<style scoped src="../CrossQueryView.css"></style>

