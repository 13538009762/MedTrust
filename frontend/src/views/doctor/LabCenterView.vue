<template>
  <div class="lab-center-container">
    <!-- 顶部状态统计卡片 -->
    <div class="stats-grid">
      <div class="stat-card pending">
        <div class="stat-icon">📋</div>
        <div class="stat-info">
          <div class="stat-value">{{ pendingCount }}</div>
          <div class="stat-label">待处理检查单</div>
        </div>
      </div>
      <div class="stat-card processing">
        <div class="stat-icon">⏳</div>
        <div class="stat-info">
          <div class="stat-value">{{ processingCount }}</div>
          <div class="stat-label">检查进行中</div>
        </div>
      </div>
      <div class="stat-card completed">
        <div class="stat-icon">✅</div>
        <div class="stat-info">
          <div class="stat-value">{{ completedCount }}</div>
          <div class="stat-label">已出具报告回传</div>
        </div>
      </div>
    </div>

    <!-- 主卡片与标签页 -->
    <el-card class="main-card" shadow="never">
      <template #header>
        <div class="card-header">
          <div class="header-title">
            <el-icon class="title-icon"><Tickets /></el-icon>
            <span>医技辅助检查中心 (检验科 / 影像中心)</span>
            <el-tag type="info" size="small" effect="plain" class="ml-2">流程解耦·医技协同</el-tag>
          </div>
          <div class="header-actions">
            <el-button :icon="Refresh" circle @click="fetchOrders" />
          </div>
        </div>
      </template>

      <!-- 角色与本院隔离合规提示 -->
      <div class="lab-role-alert mb-3">
        <el-alert
          :title="`🛡️ 医技科室合规与数据边界：当前登录为「${auth.user?.hospital_name || '本院机构'} · ${auth.user?.department_name || '医技中心'} · ${auth.user?.real_name} (${auth.user?.title || '技师/医师'})」。依据《医疗机构临床实验室管理办法》与网络安全规范，医技检查中心仅展示并处理本院临床医生开立的检查申请单，禁止越权调取或处理外院检查。`"
          type="info"
          :closable="false"
          show-icon
        />
      </div>

      <!-- 状态筛选与本院业务标识 -->
      <div class="filter-bar">
        <el-radio-group v-model="activeTab" size="default" @change="onTabChange">
          <el-radio-button label="ALL">全部本院申请 ({{ orders.length }})</el-radio-button>
          <el-radio-button label="PENDING">待检查 ({{ pendingCount }})</el-radio-button>
          <el-radio-button label="PROCESSING">检查中 ({{ processingCount }})</el-radio-button>
          <el-radio-button label="COMPLETED">已出报告 ({{ completedCount }})</el-radio-button>
        </el-radio-group>

        <div class="filter-right-group">
          <div class="hosp-locked-badge">
            <span class="text-xs text-slate-500">业务机构：</span>
            <el-tag type="primary" size="default" effect="plain">
              🏥 {{ auth.user?.hospital_name || formatHospName(auth.user?.hospital_id) }}（本院医技工作站）
            </el-tag>
          </div>

          <el-input
            v-model="searchKeyword"
            placeholder="搜索患者姓名、单号、项目..."
            clearable
            style="width: 240px;"
            :prefix-icon="Search"
          />
        </div>
      </div>

      <!-- 检查单列表 -->
      <el-table
        v-loading="loading"
        :data="filteredOrders"
        style="width: 100%;"
        class="order-table"
        stripe
      >
        <el-table-column prop="order_no" label="申请单号" width="170">
          <template #default="{ row }">
            <span class="mono font-bold">{{ row.order_no }}</span>
          </template>
        </el-table-column>

        <el-table-column prop="record_no" label="关联就诊号" width="160">
          <template #default="{ row }">
            <span class="mono text-gray-600">{{ row.record_no || 'ENC就诊' }}</span>
          </template>
        </el-table-column>

        <el-table-column label="就诊患者" width="110">
          <template #default="{ row }">
            <span class="patient-name">{{ row.patient_name || '患者' }}</span>
          </template>
        </el-table-column>

        <el-table-column label="开单医疗机构" width="150">
          <template #default="{ row }">
            <el-tag :type="getHospTagType(row.hospital_id)" effect="plain" size="small">
              {{ row.hospital_name || formatHospName(row.hospital_id) }}
            </el-tag>
          </template>
        </el-table-column>

        <el-table-column label="派发执行科室" width="160">
          <template #default="{ row }">
            <el-tag v-if="isImageItem(row.exam_item)" type="warning" size="small">
              放射影像与心电中心
            </el-tag>
            <el-tag v-else type="primary" size="small">
              临床检验医学中心
            </el-tag>
          </template>
        </el-table-column>

        <el-table-column prop="exam_item" label="医技检查项目" min-width="180">
          <template #default="{ row }">
            <el-tag :type="getExamTagType(row.exam_item)" effect="light">
              {{ row.exam_item }}
            </el-tag>
          </template>
        </el-table-column>

        <el-table-column label="开单科室 / 医生" width="160">
          <template #default="{ row }">
            <div>{{ row.department_name || '综合门诊' }}</div>
            <div class="text-xs text-gray-500">{{ row.doctor_name }} 医师</div>
          </template>
        </el-table-column>

        <el-table-column prop="exam_reason" label="临床指征与申请目的" min-width="180" show-overflow-tooltip />

        <el-table-column prop="created_at" label="申请时间" width="150">
          <template #default="{ row }">
            {{ formatTime(row.created_at) }}
          </template>
        </el-table-column>

        <el-table-column label="出具医技 / 机构" width="180">
          <template #default="{ row }">
            <div v-if="row.technician_name">
              <span class="font-medium text-slate-800">{{ row.technician_name }}</span>
              <div class="text-xs text-gray-500">{{ row.technician_hospital_name || row.hospital_name || '医技中心' }}</div>
            </div>
            <span v-else class="text-xs text-gray-400 italic">待出具报告</span>
          </template>
        </el-table-column>

        <el-table-column prop="status" label="检查状态" width="110">
          <template #default="{ row }">
            <el-tag v-if="row.status === 'PENDING'" type="warning">待检查</el-tag>
            <el-tag v-else-if="row.status === 'PROCESSING'" type="primary">检查中</el-tag>
            <el-tag v-else-if="row.status === 'COMPLETED'" type="success">已完成</el-tag>
          </template>
        </el-table-column>

        <el-table-column label="操作" width="170" fixed="right">
          <template #default="{ row }">
            <el-button
              v-if="row.status === 'PENDING'"
              type="primary"
              size="small"
              @click="openProcessDialog(row)"
            >
              接单并出报告
            </el-button>
            <el-button
              v-else-if="row.status === 'PROCESSING'"
              type="warning"
              size="small"
              @click="openProcessDialog(row)"
            >
              录入检查报告
            </el-button>
            <el-button
              v-else
              type="info"
              size="small"
              plain
              @click="openViewDialog(row)"
            >
              查看报告单
            </el-button>
          </template>
        </el-table-column>
      </el-table>
    </el-card>

    <!-- 录入检查报告弹窗 -->
    <el-dialog
      v-model="processDialogVisible"
      title="🔬 医技辅助检查执行与报告出具"
      width="780px"
      top="4vh"
      :close-on-click-modal="false"
    >
      <div v-if="currentOrder" class="process-dialog-body">
        <!-- 患者与申请信息全景单据卡片 (解决不知道点击的是什么检验单的问题) -->
        <div class="requisition-card mb-3">
          <div class="requisition-top-bar">
            <div class="requisition-title-group">
              <span class="requisition-badge">检验/影像执行单</span>
              <span class="requisition-exam-name">{{ currentOrder.exam_item }}</span>
              <el-tag :type="getExamTagType(currentOrder.exam_item)" size="small" effect="plain" class="ml-2">
                {{ isImageItem(currentOrder.exam_item) ? '影像/心电类' : '检验生化/血常规类' }}
              </el-tag>
            </div>
            <div class="requisition-no-group">
              <span class="text-xs text-gray-500">申请单号：</span>
              <code class="order-no-code">{{ currentOrder.order_no }}</code>
            </div>
          </div>

          <div class="requisition-meta-grid">
            <div class="meta-cell">
              <span class="lbl">就诊患者：</span>
              <span class="val font-bold text-slate-800">{{ currentOrder.patient_name }}</span>
              <span v-if="currentOrder.patient_phone" class="text-xs text-gray-500 ml-1">({{ currentOrder.patient_phone }})</span>
            </div>
            <div class="meta-cell">
              <span class="lbl">开立机构：</span>
              <el-tag :type="getHospTagType(currentOrder.hospital_id)" size="small">
                {{ currentOrder.hospital_name || formatHospName(currentOrder.hospital_id) }}
              </el-tag>
            </div>
            <div class="meta-cell">
              <span class="lbl">开立医生：</span>
              <span class="val">{{ currentOrder.doctor_name || '经治责任医生' }} ({{ currentOrder.department_name || '接诊科室' }})</span>
            </div>
            <div class="meta-cell">
              <span class="lbl">就诊类型：</span>
              <el-tag size="small" effect="plain">{{ formatEncounterType(currentOrder.encounter_type) }}</el-tag>
            </div>
          </div>

          <!-- 临床关键信息背景框 (让技师明确开单背景与患者情况) -->
          <div class="clinical-context-box">
            <div class="context-row">
              <span class="context-lbl">🎯 开单目的与临床指征：</span>
              <span class="context-val highlight-reason">{{ currentOrder.exam_reason || '临床排查与专科辅助诊断' }}</span>
            </div>
            <div v-if="currentOrder.patient_initial_diagnosis" class="context-row">
              <span class="context-lbl">🩺 经治医生拟定初诊：</span>
              <span class="context-val font-semibold text-blue-700">{{ currentOrder.patient_initial_diagnosis }}</span>
            </div>
            <div v-if="currentOrder.patient_chief_complaint" class="context-row">
              <span class="context-lbl">📋 患者主诉与发病症状：</span>
              <span class="context-val">{{ currentOrder.patient_chief_complaint }}</span>
            </div>
            <div v-if="currentOrder.patient_vital_signs" class="context-row">
              <span class="context-lbl">💓 患者生命体征记录：</span>
              <span class="context-val text-xs text-gray-700">{{ currentOrder.patient_vital_signs }}</span>
            </div>
          </div>
        </div>

        <el-divider content-position="left">报告规范填报与所见描述</el-divider>

        <!-- 针对该检验项目的动态规范模板快捷载入 -->
        <div class="quick-template-bar mb-3">
          <div class="template-bar-label">
            <span class="bolt-icon">⚡</span>
            <span class="font-medium text-slate-700">载入【{{ currentOrder.exam_item }}】专科行业规范模板：</span>
          </div>
          <div class="template-buttons">
            <el-button
              v-for="(tpl, idx) in getQuickTemplates(currentOrder.exam_item)"
              :key="idx"
              size="small"
              :type="tpl.btnType"
              plain
              @click="applyQuickTemplate(tpl)"
            >
              {{ tpl.tag }} {{ tpl.title }}
            </el-button>
          </div>
        </div>

        <el-form label-position="top" class="report-form">
          <el-form-item label="测量指标明细与检查所见描述 (Findings)" required>
            <el-input
              v-model="reportForm.exam_result"
              type="textarea"
              :rows="5"
              placeholder="请输入检验检查测量数据（如各细胞分析数值、心电图各导联波形电轴、影像学密度影等）"
            />
          </el-form-item>

          <el-form-item label="检查报告诊断结论 (Impression / Conclusion)" required>
            <el-input
              v-model="reportForm.exam_conclusion"
              placeholder="例如：窦性心律，大致正常心电图；未见明显器质性病理征象"
            />
          </el-form-item>

          <el-row :gutter="16">
            <el-col :span="12">
              <el-form-item label="报告出具医师/技师姓名" required>
                <el-input v-model="reportForm.technician_name" placeholder="请输入出具人姓名" />
              </el-form-item>
            </el-col>
            <el-col :span="12">
              <el-form-item label="上传影像切片/检验报告扫描件 (支持图片/PDF在线查阅)">
                <input type="file" accept="image/*,.pdf" @change="onFileChange" class="file-input" />
                <div v-if="filePreviewUrl" class="file-preview-wrap mt-2">
                  <img :src="filePreviewUrl" class="upload-img-thumb" alt="预览" />
                  <span class="text-xs text-green-600 ml-2">已选择图片: {{ selectedFile?.name }}</span>
                </div>
              </el-form-item>
            </el-col>
          </el-row>

          <!-- 填写上传的图片是什么检查单 (支持自定义输入与专科智能快捷预设) -->
          <el-form-item label="🖼️ 上传图片/单据名称与检查类型说明 (Exam Sheet / File Title)">
            <el-input
              v-model="reportForm.file_title"
              placeholder="例如：【12导联心电图】标准心电波形报告单 或 【胸部CT平扫】横断面影像切片"
              clearable
            >
              <template #prepend>
                <span class="text-xs font-semibold text-slate-600">单据名称</span>
              </template>
            </el-input>
            <div class="quick-title-tags mt-2 flex flex-wrap gap-2 items-center">
              <span class="text-xs text-gray-500">快捷单据命名推荐：</span>
              <el-tag
                size="small"
                effect="plain"
                class="cursor-pointer hover:opacity-80"
                @click="reportForm.file_title = `【${currentOrder?.exam_item || '医技检查'}】报告扫描件`"
              >
                【{{ currentOrder?.exam_item || '检查' }}】报告扫描件
              </el-tag>
              <el-tag
                size="small"
                effect="plain"
                type="primary"
                class="cursor-pointer hover:opacity-80"
                @click="reportForm.file_title = `【${currentOrder?.exam_item || '医技检查'}】原始影像切片`"
              >
                【{{ currentOrder?.exam_item || '检查' }}】原始影像切片
              </el-tag>
              <el-tag
                size="small"
                effect="plain"
                type="success"
                class="cursor-pointer hover:opacity-80"
                @click="reportForm.file_title = `【${currentOrder?.exam_item || '检验化验'}】理化生化检验结果单`"
              >
                【{{ currentOrder?.exam_item || '检验' }}】理化生化检验单
              </el-tag>
              <el-tag
                size="small"
                effect="plain"
                type="warning"
                class="cursor-pointer hover:opacity-80"
                @click="reportForm.file_title = `【${currentOrder?.exam_item || '检查'}】标准全景波形图`"
              >
                【{{ currentOrder?.exam_item || '检查' }}】标准波形图
              </el-tag>
            </div>
          </el-form-item>
        </el-form>
      </div>

      <template #footer>
        <el-button @click="processDialogVisible = false">取消</el-button>
        <el-button
          type="primary"
          :loading="submitting"
          :disabled="!reportForm.exam_result || !reportForm.exam_conclusion"
          @click="submitReport"
        >
          ✅ 提交报告并回传至医生接诊档案
        </el-button>
      </template>
    </el-dialog>

    <!-- 查看已完成报告单弹窗 -->
    <el-dialog
      v-model="viewDialogVisible"
      title="📄 医技辅助检查回传报告单"
      width="680px"
    >
      <div v-if="viewOrder" class="view-report-body">
        <div class="report-sheet">
          <div class="sheet-title">MedTrust 医疗可信共享·医技检查回传报告</div>
          <div class="sheet-meta">
            <div><strong>申请单号：</strong>{{ viewOrder.order_no }}</div>
            <div><strong>就诊编号：</strong>{{ viewOrder.record_no }}</div>
            <div><strong>受检患者：</strong>{{ viewOrder.patient_name }}</div>
            <div><strong>开单机构：</strong>{{ viewOrder.hospital_name || formatHospName(viewOrder.hospital_id) }}</div>
            <div><strong>开单医生：</strong>{{ viewOrder.doctor_name }} ({{ viewOrder.department_name }})</div>
            <div><strong>检查项目：</strong>{{ viewOrder.exam_item }}</div>
            <div><strong>就诊类型：</strong>{{ formatEncounterType(viewOrder.encounter_type) }}</div>
            <div><strong>报告时间：</strong>{{ formatTime(viewOrder.executed_at) }}</div>
          </div>
          <div v-if="viewOrder.exam_reason" class="sheet-clinical-reason mt-2">
            <strong>送检临床指征：</strong>{{ viewOrder.exam_reason }}
          </div>
          <el-divider />
          <div class="sheet-section">
            <div class="section-lbl">【检查测量指标与所见描述】</div>
            <div class="section-val">{{ viewOrder.exam_result }}</div>
          </div>
          <div class="sheet-section highlight">
            <div class="section-lbl">【检查报告结论】</div>
            <div class="section-val bold">{{ viewOrder.exam_conclusion }}</div>
          </div>

          <!-- 附件与影像展示卡片 -->
          <div v-if="viewOrder.file_id || viewOrder.report_file_name" class="sheet-section file-section">
            <div class="section-lbl">【医技报告附件 / 原始影像存证】</div>
            <div class="file-preview-card">
              <div class="file-info-col">
                <span class="file-name font-bold">📑 {{ viewOrder.report_file_name || '检查报告影像附件' }}</span>
                <span v-if="viewOrder.ipfs_cid" class="text-xs text-gray-500 block">IPFS CID: <code>{{ viewOrder.ipfs_cid }}</code></span>
              </div>
              <div class="file-action-col">
                <el-button
                  v-if="viewOrder.file_id"
                  size="small"
                  type="primary"
                  @click="openImagePreview(`/api/v1/medical-files/${viewOrder.file_id}/view`, viewOrder.report_file_name || viewOrder.exam_item)"
                >
                  🖼️ 在线查阅影像图片
                </el-button>
                <el-button
                  v-if="viewOrder.file_id"
                  size="small"
                  type="success"
                  plain
                  @click="downloadFile(`/api/v1/medical-files/${viewOrder.file_id}/download`, viewOrder.report_file_name || '医学检查附件')"
                >
                  📥 下载原始附件
                </el-button>
              </div>
            </div>
            <!-- 如果是图片格式，直接呈现高清内联略缩图 -->
            <div v-if="viewOrder.file_id && isImageExt(viewOrder.report_file_type || viewOrder.report_file_name)" class="mt-2 text-center img-container">
              <img
                :src="`/api/v1/medical-files/${viewOrder.file_id}/view`"
                class="report-inline-img"
                @click="openImagePreview(`/api/v1/medical-files/${viewOrder.file_id}/view`, viewOrder.report_file_name || viewOrder.exam_item)"
                title="点击放大查阅影像"
              />
              <div class="text-xs text-gray-400 mt-1">（点击影像图片可全屏高清放大查阅）</div>
            </div>
          </div>

          <div class="sheet-footer">
            <span>出具人员及机构：<strong>{{ viewOrder.technician_name || '医技科室' }}</strong> <el-tag v-if="viewOrder.technician_hospital_name" size="small" type="info" class="ml-1">{{ viewOrder.technician_hospital_name }}</el-tag></span>
            <el-tag type="success" size="small">状态：已完成回传</el-tag>
          </div>
        </div>
      </div>
    </el-dialog>

    <!-- 影像大图全屏查阅弹窗 -->
    <el-dialog
      v-model="imagePreviewVisible"
      :title="`🖼️ ${previewImageTitle || '医学检验与影像检查报告大图'}`"
      width="850px"
      top="4vh"
    >
      <div class="text-center p-3">
        <img :src="previewImageUrl" style="max-width: 100%; max-height: 72vh; border-radius: 8px; box-shadow: 0 4px 16px rgba(0,0,0,0.15);" alt="影像大图" />
      </div>
      <template #footer>
        <el-button @click="imagePreviewVisible = false">关闭预览</el-button>
        <el-button type="primary" :icon="Download" @click="downloadFile(previewImageUrl, previewImageTitle)">
          下载高清影像原图
        </el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { Tickets, Refresh, Search, Download } from '@element-plus/icons-vue'
import { ElMessage } from 'element-plus'
import api from '../../api/client'
import { useAuthStore } from '../../stores/auth'

const auth = useAuthStore()

const loading = ref(false)
const orders = ref<any[]>([])
const activeTab = ref('ALL')
const searchKeyword = ref('')

const processDialogVisible = ref(false)
const currentOrder = ref<any>(null)
const submitting = ref(false)
const selectedFile = ref<File | null>(null)
const filePreviewUrl = ref('')

const viewDialogVisible = ref(false)
const viewOrder = ref<any>(null)

const imagePreviewVisible = ref(false)
const previewImageUrl = ref('')
const previewImageTitle = ref('')

const reportForm = ref({
  exam_result: '',
  exam_conclusion: '',
  technician_name: auth.user?.real_name || '检验科主管技师',
  file_title: ''
})

const pendingCount = computed(() => orders.value.filter(o => o.status === 'PENDING').length)
const processingCount = computed(() => orders.value.filter(o => o.status === 'PROCESSING').length)
const completedCount = computed(() => orders.value.filter(o => o.status === 'COMPLETED').length)

function formatHospName(hId?: number) {
  if (hId === 1) return '第一人民医院'
  if (hId === 2) return '第二人民医院'
  if (hId === 3) return '第三人民医院'
  return '医疗中心'
}

function getHospTagType(hId?: number) {
  if (hId === 1) return 'primary'
  if (hId === 2) return 'warning'
  if (hId === 3) return 'success'
  return 'info'
}

function formatEncounterType(t?: string) {
  if (t === 'OUTPATIENT') return '普通门诊'
  if (t === 'EMERGENCY') return '急救诊疗'
  if (t === 'INPATIENT') return '住院就诊'
  return t || '门诊'
}

const filteredOrders = computed(() => {
  let list = orders.value
  if (activeTab.value !== 'ALL') {
    list = list.filter(o => o.status === activeTab.value)
  }
  if (searchKeyword.value.trim()) {
    const kw = searchKeyword.value.trim().toLowerCase()
    list = list.filter(o =>
      (o.order_no && o.order_no.toLowerCase().includes(kw)) ||
      (o.patient_name && o.patient_name.toLowerCase().includes(kw)) ||
      (o.exam_item && o.exam_item.toLowerCase().includes(kw)) ||
      (o.doctor_name && o.doctor_name.toLowerCase().includes(kw))
    )
  }
  return list
})

async function fetchOrders() {
  loading.value = true
  try {
    const res: any = await api.get('/exam-orders')
    if (res.code === 200) {
      orders.value = res.data || []
    }
  } catch (err: any) {
    ElMessage.error('获取医技检查列表失败: ' + (err.message || '网络错误'))
  } finally {
    loading.value = false
  }
}

function onTabChange() {
  // Tab changed
}

function isImageItem(item?: string) {
  if (!item) return false
  return item.includes('心电图') || item.includes('CT') || item.includes('MRI') || item.includes('超声') || item.includes('放射')
}

function getExamTagType(item: string) {
  if (!item) return 'info'
  if (item.includes('心电图')) return 'danger'
  if (item.includes('血常规')) return 'primary'
  if (item.includes('CT') || item.includes('MRI')) return 'warning'
  return 'success'
}

function isImageExt(fileName?: string) {
  if (!fileName) return false
  const lower = fileName.toLowerCase()
  return lower.endsWith('.png') || lower.endsWith('.jpg') || lower.endsWith('.jpeg') || lower.endsWith('.gif') || lower.endsWith('.webp') || lower.endsWith('.svg') || lower.includes('png') || lower.includes('jpg') || lower.includes('jpeg')
}

function formatTime(t?: string) {
  if (!t) return '-'
  return t.substring(0, 16).replace('T', ' ')
}

function openProcessDialog(row: any) {
  currentOrder.value = row
  const currentHospital = auth.user?.hospital_name || formatHospName(auth.user?.hospital_id)
  const defaultTechName = auth.user?.real_name ? `${auth.user.real_name} (${currentHospital})` : '医技主管'
  const defaultFileTitle = row.report_file_name || `【${row.exam_item}】检查单据扫描件/原始影像`
  reportForm.value = {
    exam_result: row.exam_result || '',
    exam_conclusion: row.exam_conclusion || '',
    technician_name: row.technician_name || defaultTechName,
    file_title: defaultFileTitle
  }
  selectedFile.value = null
  filePreviewUrl.value = ''
  processDialogVisible.value = true
}

function openViewDialog(row: any) {
  viewOrder.value = row
  viewDialogVisible.value = true
}

function onFileChange(e: any) {
  const f = e.target.files?.[0]
  if (f) {
    selectedFile.value = f
    if (f.type.startsWith('image/')) {
      filePreviewUrl.value = URL.createObjectURL(f)
    } else {
      filePreviewUrl.value = ''
    }
    if (!reportForm.value.file_title && currentOrder.value) {
      reportForm.value.file_title = `【${currentOrder.value.exam_item}】检查单据扫描件`
    }
  } else {
    selectedFile.value = null
    filePreviewUrl.value = ''
  }
}

interface QuickTemplate {
  title: string
  tag: string
  btnType: 'success' | 'warning' | 'danger' | 'primary' | 'info'
  result: string
  conclusion: string
}

function getQuickTemplates(item: string): QuickTemplate[] {
  const norm = (item || '').toLowerCase()
  if (norm.includes('肝') || norm.includes('肾') || norm.includes('生化') || norm.includes('代谢') || norm.includes('电解质')) {
    return [
      {
        title: '生化全套指标正常 (国家标准参考基准值)',
        tag: '🟢 生理正常',
        btnType: 'success',
        result: '【肝肾功能与电解质生化全套】\n- 谷丙转氨酶(ALT): 18 U/L (参考: 9-50)\n- 谷草转氨酶(AST): 22 U/L (参考: 15-40)\n- 总胆红素(TBIL): 12.4 μmol/L (参考: 3.4-17.1)\n- 血清白蛋白(ALB): 46.2 g/L (参考: 40-55)\n- 血肌酐(Cr): 68 μmol/L (参考: 57-111)\n- 尿素氮(BUN): 4.6 mmol/L (参考: 3.2-7.1)\n- 血尿酸(UA): 290 μmol/L (参考: 208-428)\n- 钾(K+): 4.15 mmol/L (参考: 3.5-5.3)\n- 钠(Na+): 141.0 mmol/L (参考: 136-145)\n- 空腹血糖(GLU): 5.12 mmol/L (参考: 3.9-6.1)',
        conclusion: '各项肝功能、肾功能指标与电解质水盐平衡均在国家标准生理参考区间内，生化代谢未见异常。'
      },
      {
        title: '转氨酶偏高 (轻中度肝细胞受损)',
        tag: '🟡 转氨酶偏高',
        btnType: 'warning',
        result: '【肝肾功能与电解质生化全套】\n- 谷丙转氨酶(ALT): 86 U/L ↑ (参考: 9-50)\n- 谷草转氨酶(AST): 64 U/L ↑ (参考: 15-40)\n- 总胆红素(TBIL): 18.2 μmol/L (参考: 3.4-17.1)\n- 血清白蛋白(ALB): 42.1 g/L (参考: 40-55)\n- 血肌酐(Cr): 72 μmol/L (正常)\n- 尿素氮(BUN): 5.1 mmol/L (正常)\n- 钾(K+): 4.0 mmol/L，钠(Na+): 139 mmol/L',
        conclusion: '血清转氨酶轻中度升高，提示急性/亚急性肝细胞损害，建议结合病毒性肝炎、脂肪肝或药物性损伤病因复查。'
      },
      {
        title: '肾功能损害 (肌酐/尿素氮显著升高)',
        tag: '🔴 肾功异常',
        btnType: 'danger',
        result: '【肝肾功能与电解质生化全套】\n- 血肌酐(Cr): 192 μmol/L ↑ (参考: 57-111)\n- 尿素氮(BUN): 15.8 mmol/L ↑ (参考: 3.2-7.1)\n- 血尿酸(UA): 535 μmol/L ↑ (参考: 208-428)\n- 估算肾小球滤过率(eGFR): 36 mL/min/1.73m² ↓\n- 钾(K+): 5.32 mmol/L ↑ (参考: 3.5-5.3)\n- 肝功能各项测量值大致正常',
        conclusion: '血肌酐及尿素氮显著升高，伴高尿酸血症及轻度高钾倾向，提示肾小球滤过功能受损（肾功能不全），建议肾内科积极干预。'
      }
    ]
  } else if (norm.includes('心电') || norm.includes('ecg')) {
    return [
      {
        title: '12导联心电图大致正常 (窦性心律)',
        tag: '🟢 窦性心律',
        btnType: 'success',
        result: '【标准12导联静息心电图】\n- 节律: 窦性心律，心率 72 bpm\n- P波: 时限 0.08s，振幅正常\n- P-R间期: 0.16s (参考: 0.12-0.20s)\n- QRS波群: 时限 0.09s，各导联形态正常，电轴无显著偏移\n- ST-T段: ST段未见明显抬高或压低，T波形态自然直立，Q-Tc间期 412 ms',
        conclusion: '窦性心律，大致正常心电图。'
      },
      {
        title: 'ST-T缺血性压低 (下壁/前壁供血不足)',
        tag: '🟡 心肌缺血',
        btnType: 'warning',
        result: '【标准12导联静息心电图】\n- 节律: 窦性心律，心率 88 bpm\n- ST-T段: V4-V6 导联及 II、III、aVF 导联 ST 段呈水平型下移约 0.08-0.12 mV，T波低平伴倒置\n- Q-Tc间期: 442 ms',
        conclusion: '窦性心律伴下侧壁 ST-T 段缺血性改变，高度提示心肌供血不足，建议结合心肌损伤标志物随诊。'
      },
      {
        title: '窦性心动过速 / 偶发室早',
        tag: '🔴 心律失常',
        btnType: 'danger',
        result: '【标准12导联静息心电图】\n- 节律: 窦性心动过速，心率 116 bpm\n- 记录期内可见 2 次提前出现的宽大畸形 QRS 波群，代偿间歇完全\n- ST-T段轻度继发性改变',
        conclusion: '窦性心动过速伴偶发室性期前收缩（室早），建议进一步完善 24小时动态心电图 (Holter) 排查。'
      }
    ]
  } else if (norm.includes('血常规') || norm.includes('细胞') || norm.includes('血象')) {
    return [
      {
        title: '全血细胞各项指标正常 (无感染贫血)',
        tag: '🟢 细胞正常',
        btnType: 'success',
        result: '【全血细胞分析与分类】\n- 白细胞计数(WBC): 6.4 ×10^9/L (参考: 3.5-9.5)\n- 中性粒细胞百分比(NEU%): 61.2% (参考: 40-75)\n- 淋巴细胞百分比(LYM%): 29.8% (参考: 20-50)\n- 红细胞计数(RBC): 4.82 ×10^12/L (参考: 4.3-5.8)\n- 血红蛋白浓度(Hb): 148 g/L (参考: 130-175)\n- 血小板计数(PLT): 225 ×10^9/L (参考: 125-350)\n- C-反应蛋白(CRP): 2.1 mg/L (参考: 0-8)',
        conclusion: '全血细胞分析各参数均在标准参考区间内，未见明显感染、贫血或出凝血异常指征。'
      },
      {
        title: '急性细菌感染 / 炎症指标增高',
        tag: '🟡 细菌感染',
        btnType: 'warning',
        result: '【全血细胞分析与分类】\n- 白细胞计数(WBC): 14.2 ×10^9/L ↑ (参考: 3.5-9.5)\n- 中性粒细胞百分比(NEU%): 84.6% ↑ (参考: 40-75)\n- 中性粒细胞绝对值(NEU#): 12.0 ×10^9/L ↑\n- 超敏C反应蛋白(hs-CRP): 34.5 mg/L ↑ (参考: 0-5)\n- 红细胞及血小板未见明显异常',
        conclusion: '白细胞总数及中性粒细胞比例显著增高，CRP明显升高，符合急性细菌性感染或活动性炎性反应。'
      },
      {
        title: '中度小细胞低色素性贫血',
        tag: '🔴 贫血异常',
        btnType: 'danger',
        result: '【全血细胞分析与分类】\n- 红细胞计数(RBC): 3.12 ×10^12/L ↓ (参考: 4.3-5.8)\n- 血红蛋白浓度(Hb): 82 g/L ↓ (参考: 130-175)\n- 平均红细胞体积(MCV): 71.4 fL ↓ (参考: 82-100)\n- 平均红细胞血红蛋白量(MCH): 23.2 pg ↓ (参考: 27-34)\n- 白细胞与血小板大致正常',
        conclusion: '小细胞低色素性中度贫血，高度提示缺铁性贫血可能，建议进一步检查血清铁蛋白及转铁蛋白饱和度。'
      }
    ]
  } else if (norm.includes('心肌酶') || norm.includes('肌钙蛋白') || norm.includes('ctn')) {
    return [
      {
        title: '心肌损伤标志物全套阴性',
        tag: '🟢 阴性正常',
        btnType: 'success',
        result: '【心肌损伤全套化学发光检测】\n- 超敏肌钙蛋白I (hs-cTnI): < 0.012 ng/mL (阴性，参考: < 0.034)\n- 肌酸激酶同工酶 (CK-MB mass): 1.4 ng/mL (参考: 0-5.0)\n- 肌红蛋白 (Myo): 26.5 ng/mL (参考: 0-70)\n- B型脑钠肽前体 (NT-proBNP): 48 pg/mL (参考: < 125)',
        conclusion: '心肌坏死损伤标志物全套阴性，未见急性心肌梗死或急性心肌缺血坏死生物学证据。'
      },
      {
        title: '肌钙蛋白强阳性 (急性心梗指标)',
        tag: '🔴 强阳性预警',
        btnType: 'danger',
        result: '【心肌损伤全套化学发光检测】\n- 超敏肌钙蛋白I (hs-cTnI): 3.86 ng/mL ↑ (显著阳性，参考: < 0.034)\n- 肌酸激酶同工酶 (CK-MB mass): 48.2 ng/mL ↑ (参考: 0-5.0)\n- 肌红蛋白 (Myo): 185.0 ng/mL ↑ (参考: 0-70)\n- NT-proBNP: 890 pg/mL ↑',
        conclusion: '高敏肌钙蛋白I与CK-MB强阳性显著升高，高度符合急性心肌梗死(AMI)或急性冠脉综合征改变，建议心内科紧急处置。'
      }
    ]
  } else if (norm.includes('ct') || norm.includes('mri') || norm.includes('平扫') || norm.includes('影像') || norm.includes('胸部')) {
    return [
      {
        title: '平扫检查未见明确活动性病变',
        tag: '🟢 未见异常',
        btnType: 'success',
        result: '【高分辨率CT平扫检查】\n- 双侧胸廓对称，纵隔居中气管通畅。\n- 双肺野透亮度正常，双肺纹理走行清晰规整，肺野内未见明确渗出、浸润、实变或肿块结节影。\n- 肺门大小形态未见明显增大，纵隔未见明显肿大淋巴结影。\n- 心影大小处于正常生理范围，双侧胸膜腔未见明显积液征象。',
        conclusion: '胸部CT平扫未见明显活动性炎性浸润或占位性病变。'
      },
      {
        title: '双下肺野斑片状炎性渗出影 (肺炎)',
        tag: '🟡 炎性浸润',
        btnType: 'warning',
        result: '【高分辨率CT平扫检查】\n- 双侧胸廓对称，气管通畅。\n- 双侧下肺野背段及外底段见散在斑片状、磨玻璃样浅淡高密度影，边界欠清，内可见支气管充气征。\n- 纵隔淋巴结轻度反应性肿大，心影大小形态正常，未见明显胸腔积液。',
        conclusion: '双下肺野斑片状渗出性病变，考虑感染性病变（社区获得性肺炎表现可能），建议抗炎对症治疗后复查。'
      },
      {
        title: '肺部结节影 (磨玻璃结节 GGN)',
        tag: '🔴 肺部结节',
        btnType: 'danger',
        result: '【高分辨率CT平扫检查】\n- 右肺中叶外侧段近胸膜下见一结节状磨玻璃密度影，大小约为 8.5 mm × 7.2 mm，边界较清，内部密度欠均，边缘见微小分叶及轻度胸膜牵拉。\n- 其余肺野未见明显实变灶，纵隔结构清晰。',
        conclusion: '右肺中叶局灶性磨玻璃结节 (pGGN)，建议3个月后薄层低剂量CT随访或呼吸胸外科专科会诊评估。'
      }
    ]
  } else {
    return [
      {
        title: `${item} - 测量指标正常参考范围`,
        tag: '🟢 正常参考值',
        btnType: 'success',
        result: `【${item}】\n各项测量理化数据均符合国家临床检验生理参考区间，各参数值处于稳定生理中线，未检出阳性病理性改变。`,
        conclusion: '各项测量指标未见明显异常，符合健康生理参考标准。'
      },
      {
        title: `${item} - 测量指标轻度异常偏离`,
        tag: '🟡 轻度异常',
        btnType: 'warning',
        result: `【${item}】\n主要测量指标轻度偏离正常区间上限，提示机体处于早期应激、轻微炎性反应或代偿阶段。`,
        conclusion: '检查指标轻度异常，建议结合专科临床表现及随访排查。'
      },
      {
        title: `${item} - 测量指标显著异常 / 阳性发现`,
        tag: '🔴 明显异常/阳性',
        btnType: 'danger',
        result: `【${item}】\n关键理化指标显著超出参考阈值，见特征性病理改变改变与阳性指征，机体靶器官功能受累明显。`,
        conclusion: '关键指标显著异常，高度提示病理损害，需专科重点排查并跟进处置。'
      }
    ]
  }
}

function applyQuickTemplate(tpl: QuickTemplate) {
  reportForm.value.exam_result = tpl.result
  reportForm.value.exam_conclusion = tpl.conclusion
  ElMessage.success(`已载入「${tpl.title}」行业规范模板`)
}

function openImagePreview(url: string, title?: string) {
  previewImageUrl.value = url
  previewImageTitle.value = title || '医学检查图像'
  imagePreviewVisible.value = true
}

function downloadFile(url: string, name?: string) {
  const a = document.createElement('a')
  a.href = url
  a.download = name || 'medical_report_file'
  a.target = '_blank'
  document.body.appendChild(a)
  a.click()
  document.body.removeChild(a)
}

async function submitReport() {
  if (!currentOrder.value) return
  submitting.value = true
  try {
    const formData = new FormData()
    formData.append('technician_name', reportForm.value.technician_name)
    formData.append('exam_result', reportForm.value.exam_result)
    formData.append('exam_conclusion', reportForm.value.exam_conclusion)
    if (selectedFile.value) {
      formData.append('file', selectedFile.value)
      formData.append('file_title', reportForm.value.file_title || '')
    }

    const res: any = await api.post(`/exam-orders/${currentOrder.value.id}/complete`, formData, {
      headers: { 'Content-Type': 'multipart/form-data' }
    })
    if (res.code === 200) {
      ElMessage.success('报告出具成功！已自动回传至接诊医生工作台')
      processDialogVisible.value = false
      fetchOrders()
    } else {
      ElMessage.error(res.message || '提交失败')
    }
  } catch (err: any) {
    ElMessage.error('提交报告异常: ' + (err.message || '网络错误'))
  } finally {
    submitting.value = false
  }
}

onMounted(() => {
  fetchOrders()
})
</script>

<style scoped>
.lab-center-container {
  padding: 20px;
  display: flex;
  flex-direction: column;
  gap: 20px;
}

.stats-grid {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 20px;
}

.stat-card {
  background: #ffffff;
  border-radius: 12px;
  padding: 20px;
  display: flex;
  align-items: center;
  gap: 16px;
  border: 1px solid rgba(226, 232, 240, 0.8);
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.02);
}

.stat-card.pending {
  border-left: 4px solid #f59e0b;
}

.stat-card.processing {
  border-left: 4px solid #0284c7;
}

.stat-card.completed {
  border-left: 4px solid #10b981;
}

.stat-icon {
  font-size: 2.2rem;
}

.stat-value {
  font-size: 1.8rem;
  font-weight: 700;
  color: #1e293b;
  line-height: 1.2;
}

.stat-label {
  font-size: 0.88rem;
  color: #64748b;
  margin-top: 4px;
}

.main-card {
  border-radius: 12px;
  border: 1px solid rgba(226, 232, 240, 0.8);
}

.card-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.header-title {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 1.1rem;
  font-weight: 600;
  color: #1e293b;
}

.title-icon {
  font-size: 1.3rem;
  color: #0284c7;
}

.filter-bar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 16px;
  flex-wrap: wrap;
  gap: 12px;
}

.filter-right-group {
  display: flex;
  align-items: center;
  gap: 12px;
  flex-wrap: wrap;
}

.hosp-locked-badge {
  display: flex;
  align-items: center;
  gap: 4px;
}

.mono {
  font-family: 'Consolas', monospace;
  letter-spacing: 0.5px;
}

.patient-name {
  font-weight: 600;
  color: #0f172a;
}

.requisition-card {
  background: linear-gradient(135deg, #f8fafc 0%, #f1f5f9 100%);
  border: 1px solid #cbd5e1;
  border-radius: 10px;
  padding: 16px;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.03);
}

.requisition-top-bar {
  display: flex;
  justify-content: space-between;
  align-items: center;
  border-bottom: 1px solid #e2e8f0;
  padding-bottom: 10px;
  margin-bottom: 12px;
}

.requisition-title-group {
  display: flex;
  align-items: center;
  gap: 8px;
}

.requisition-badge {
  background: #0284c7;
  color: #ffffff;
  font-size: 11px;
  font-weight: 700;
  padding: 2px 8px;
  border-radius: 4px;
}

.requisition-exam-name {
  font-size: 17px;
  font-weight: 700;
  color: #0f172a;
}

.order-no-code {
  font-family: 'Consolas', monospace;
  font-weight: 700;
  background: #e2e8f0;
  padding: 2px 6px;
  border-radius: 4px;
  color: #334155;
}

.requisition-meta-grid {
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  gap: 10px 16px;
  font-size: 0.88rem;
  margin-bottom: 12px;
}

.meta-cell .lbl {
  color: #64748b;
}

.clinical-context-box {
  background: #ffffff;
  border: 1px solid #e2e8f0;
  border-left: 4px solid #3b82f6;
  border-radius: 6px;
  padding: 10px 14px;
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.context-row {
  font-size: 0.86rem;
  line-height: 1.45;
}

.context-lbl {
  font-weight: 600;
  color: #475569;
}

.context-val.highlight-reason {
  color: #1e293b;
  font-weight: 600;
}

.quick-template-bar {
  background: #f8fafc;
  border: 1px solid #e2e8f0;
  padding: 10px 14px;
  border-radius: 8px;
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.template-bar-label {
  display: flex;
  align-items: center;
  font-size: 0.85rem;
}

.bolt-icon {
  color: #eab308;
  margin-right: 4px;
  font-size: 1.1rem;
}

.template-buttons {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

.upload-img-thumb {
  width: 54px;
  height: 54px;
  object-fit: cover;
  border-radius: 6px;
  border: 1px solid #cbd5e1;
}

.file-preview-wrap {
  display: flex;
  align-items: center;
}

.file-input {
  display: block;
  font-size: 0.85rem;
  color: #475569;
}

.report-sheet {
  background: #f8fafc;
  border: 1px dashed #cbd5e1;
  padding: 20px;
  border-radius: 8px;
  font-size: 0.9rem;
}

.sheet-title {
  font-size: 1.1rem;
  font-weight: 700;
  text-align: center;
  margin-bottom: 16px;
  color: #0369a1;
}

.sheet-meta {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 8px;
  font-size: 0.85rem;
  color: #475569;
}

.sheet-clinical-reason {
  background: #f1f5f9;
  padding: 8px 12px;
  border-radius: 6px;
  font-size: 0.84rem;
  color: #334155;
}

.sheet-section {
  margin-top: 12px;
  padding: 10px;
  background: #ffffff;
  border-radius: 6px;
  border: 1px solid #e2e8f0;
}

.sheet-section.highlight {
  background: #ecfdf5;
  border-color: #a7f3d0;
}

.sheet-section.file-section {
  background: #f8fafc;
  border-color: #cbd5e1;
}

.file-preview-card {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 8px 12px;
  background: #ffffff;
  border: 1px solid #e2e8f0;
  border-radius: 6px;
  margin-top: 6px;
}

.report-inline-img {
  max-width: 100%;
  max-height: 240px;
  border-radius: 6px;
  border: 1px solid #e2e8f0;
  cursor: pointer;
  transition: transform 0.2s ease;
}

.report-inline-img:hover {
  transform: scale(1.02);
  border-color: #3b82f6;
}

.section-lbl {
  font-size: 0.85rem;
  font-weight: 600;
  color: #64748b;
  margin-bottom: 4px;
}

.section-val {
  color: #1e293b;
  line-height: 1.5;
  white-space: pre-line;
}

.section-val.bold {
  font-weight: 700;
  color: #065f46;
}

.sheet-footer {
  margin-top: 16px;
  display: flex;
  justify-content: space-between;
  align-items: center;
  font-size: 0.85rem;
  color: #64748b;
}
</style>
