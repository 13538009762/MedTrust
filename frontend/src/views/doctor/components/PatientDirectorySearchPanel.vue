<template>
  <!-- 检索与分支快捷导航区 (WP-16: 从 CrossQueryView 抽取) -->
  <el-card shadow="hover" class="box-card mb-4">
    <div class="filter-wrapper">
      <!-- 分支目录标签卡片 -->
      <div class="scope-nav-row">
        <span class="scope-label">病例目录：</span>
        <el-radio-group :model-value="currentScope" size="default" class="scope-tabs-group" @update:model-value="emitScope">
          <el-radio-button value="authorized">
            <span class="tab-btn-inner">
              <el-icon class="mr-1"><CircleCheckFilled /></el-icon>
              <span>外院已有查阅权限 ({{ statsCount.authorized }})</span>
            </span>
          </el-radio-button>
          <el-radio-button value="local">
            <span class="tab-btn-inner">
              <el-icon class="mr-1"><OfficeBuilding /></el-icon>
              <span>本院的 ({{ statsCount.local }})</span>
            </span>
          </el-radio-button>
          <el-radio-button value="pending">
            <span class="tab-btn-inner">
              <el-icon class="mr-1"><Lock /></el-icon>
              <span>外院需要调取的 ({{ statsCount.pending }})</span>
            </span>
          </el-radio-button>
          <el-radio-button value="all">
            <span class="tab-btn-inner">
              <el-icon class="mr-1"><Compass /></el-icon>
              <span>全网联合检索 ({{ statsCount.all }})</span>
            </span>
          </el-radio-button>
        </el-radio-group>
      </div>

      <div class="search-form mt-3">
        <el-select
          v-if="currentScope === 'all'"
          v-model="hospitalModel"
          placeholder="筛选特定机构"
          style="width: 180px;"
          @change="emitSearch"
        >
          <el-option label="全网所有医疗机构" :value="0" />
          <el-option label="第一人民医院 (Hosp A)" :value="1" />
          <el-option label="省立中心医院 (Hosp B)" :value="2" />
          <el-option label="协和医学中心 (Hosp C)" :value="3" />
        </el-select>

        <!-- 检查单与档案类别精准筛选 -->
        <el-select
          v-model="dataTypeModel"
          placeholder="档案/检查单类型"
          style="width: 200px;"
          @change="emitSearch"
        >
          <el-option label="全部档案与检查单" value="ALL" />
          <el-option label="仅调阅检查检验单/切片" value="EXAM_ONLY" />
          <el-option label="医学检验报告 (REPORT)" value="REPORT" />
          <el-option label="医学影像切片 (IMAGE)" value="IMAGE" />
          <el-option label="门诊电子病历 (EMR)" value="EMR" />
        </el-select>

        <el-input
          v-model="keywordModel"
          placeholder="输入患者姓名、手机号、身份证号、检查项目(如心电图/CT)检索..."
          class="cross-search-input"
          clearable
          @keyup.enter="emitSearch"
          @clear="emitSearch"
        >
          <template #prefix><el-icon><Search /></el-icon></template>
        </el-input>
        <el-button type="primary" :icon="Search" :loading="loading" @click="emitSearch">精准检索</el-button>
        <span class="search-tip">提示：支持按患者姓名或特定检查项目精准检索</span>
      </div>
    </div>
  </el-card>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { Search, CircleCheckFilled, OfficeBuilding, Lock, Compass } from '@element-plus/icons-vue'

type QueryScope = 'authorized' | 'local' | 'pending' | 'all'

const props = defineProps<{
  currentScope: QueryScope
  statsCount: { authorized: number; local: number; pending: number; all: number }
  selectedHospital: number
  selectedDataType: string
  searchKeyword: string
  loading: boolean
}>()

const emit = defineEmits<{
  (e: 'update:currentScope', val: QueryScope): void
  (e: 'update:selectedHospital', val: number): void
  (e: 'update:selectedDataType', val: string): void
  (e: 'update:searchKeyword', val: string): void
  (e: 'search'): void
}>()

const hospitalModel = computed({
  get: () => props.selectedHospital,
  set: (val: number) => emit('update:selectedHospital', val),
})
const dataTypeModel = computed({
  get: () => props.selectedDataType,
  set: (val: string) => emit('update:selectedDataType', val),
})
const keywordModel = computed({
  get: () => props.searchKeyword,
  set: (val: string) => emit('update:searchKeyword', val),
})

function emitScope(val: QueryScope | string | number | undefined) {
  emit('update:currentScope', val as QueryScope)
}

function emitSearch() {
  emit('search')
}
</script>
<style scoped src="../CrossQueryView.css"></style>

