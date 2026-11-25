<template>
  <div class="page-container">
    <div class="page-header">
      <div>
        <h2 class="page-title">联盟链存证监控与监管大屏</h2>
        <p class="page-sub">国家健康医疗大数据互联互通合规监管中枢 · 实时监控跨机构数据流转合规性、Fabric 联盟链区块高度与防篡改审计动态</p>
      </div>
    </div>

    <!-- 区块链底层网络实时运行状态卡片 (核心技术基础设施指标) -->
    <el-card shadow="hover" class="blockchain-status-card mb-4">
      <div class="chain-status-container">
        <div class="chain-status-left">
          <div class="chain-badge-wrapper">
            <span
              class="status-pulse-dot"
              :class="{
                'dot-green': chainStatus.connected && chainStatus.mode === 'fabric',
                'dot-amber': chainStatus.mode === 'mock',
                'dot-red': chainStatus.mode === 'fabric' && !chainStatus.connected
              }"
            ></span>
            <span
              class="status-title-text"
              :class="{
                'text-status-green': chainStatus.connected && chainStatus.mode === 'fabric',
                'text-status-amber': chainStatus.mode === 'mock',
                'text-status-red': chainStatus.mode === 'fabric' && !chainStatus.connected
              }"
            >
              {{ chainStatus.message || (chainStatus.connected ? 'Fabric Network Running' : 'Fabric Unavailable') }}
            </span>
          </div>
          <div class="chain-sub-desc">
            <template v-if="chainStatus.mode === 'fabric' && chainStatus.connected">
              Hyperledger Fabric 2.5 联盟链网络正常运行，经由 gRPC Gateway 安全互联，Raft 共识持续出块中
            </template>
            <template v-else-if="chainStatus.mode === 'mock'">
              沙箱隔离演练节点 (Sandbox Ledger Fallback) · 处于轻量级快速仿真校验与安全沙盒模式
            </template>
            <template v-else>
              底层 Fabric 联盟链服务未就绪，系统已触发安全熔断保护，拒绝伪造上链存证
            </template>
          </div>
        </div>

        <div class="chain-status-meta">
          <div class="meta-item">
            <span class="meta-label">运行模式 (Mode)</span>
            <el-tag
              :type="chainStatus.mode === 'fabric' ? (chainStatus.connected ? 'success' : 'danger') : 'warning'"
              size="small"
              effect="dark"
            >
              {{ chainStatus.mode === 'fabric' ? 'Fabric 2.5' : 'Sandbox 沙箱节点' }}
            </el-tag>
          </div>
          <div class="meta-item">
            <span class="meta-label">通道标识 (Channel)</span>
            <span class="meta-val highlight mono">{{ (chainStatus.network === 'medchannel' || !chainStatus.network) ? 'medicalchannel' : chainStatus.network }}</span>
          </div>
          <div class="meta-item">
            <span class="meta-label">智能合约 (Chaincode)</span>
            <span class="meta-val highlight mono">{{ (chainStatus.chaincode === 'medical' || !chainStatus.chaincode) ? 'medical_contract' : chainStatus.chaincode }}</span>
          </div>
          <div class="meta-item">
            <span class="meta-label">接入 Peer 节点</span>
            <span class="meta-val" :title="chainStatus.peer">{{ chainStatus.peer || 'peer0.org1.example.com' }}</span>
          </div>
          <div class="meta-item">
            <span class="meta-label">最新区块高度</span>
            <span class="meta-val block-height">#{{ chainStatus.latest_block ?? stats.block_height }}</span>
          </div>
          <div class="meta-item action-box">
            <el-button
              type="primary"
              size="small"
              plain
              :icon="Refresh"
              :loading="refreshingStatus"
              @click="fetchBlockchainStatus"
            >
              刷新状态
            </el-button>
          </div>
        </div>
      </div>
    </el-card>

    <!-- 顶部四项核心指标卡片 -->
    <el-row :gutter="16" class="metric-row">
      <el-col :span="6">
        <el-card shadow="hover" class="metric-card">
          <div class="metric-title">全网病历存证总量</div>
          <div class="metric-num text-primary">{{ stats.total_records }}</div>
          <div class="metric-foot">链下 IPFS 密文 + 链上存证</div>
        </el-card>
      </el-col>
      <el-col :span="6">
        <el-card shadow="hover" class="metric-card">
          <div class="metric-title">Fabric 账本交易总数</div>
          <div class="metric-num text-purple">{{ stats.total_tx }}</div>
          <div class="metric-foot">当前区块高度 {{ stats.block_height ? `#${stats.block_height}` : '待同步' }}</div>
        </el-card>
      </el-col>
      <el-col :span="6">
        <el-card
          shadow="hover"
          class="metric-card cursor-pointer"
          title="点击前往紧急访问事件审核台"
          @click="router.push('/supervisor/emergency')"
        >
          <div class="metric-title">紧急访问 (Break-Glass)</div>
          <div class="metric-num text-danger">{{ stats.total_emergencies }}</div>
          <div class="metric-foot">待监管人员审核: <strong>{{ stats.pending_audits }}</strong> →</div>
        </el-card>
      </el-col>
      <el-col :span="6">
        <el-card shadow="hover" class="metric-card">
          <div class="metric-title">动态验真通过率</div>
          <div class="metric-num text-success">{{ Number(stats.verification_pass_rate ?? 0).toFixed(1) }}%</div>
          <div class="metric-foot">已核验: {{ stats.verified_count ?? stats.total_records }} | 异常: {{ stats.tampered_count ?? 0 }}</div>
        </el-card>
      </el-col>
    </el-row>

    <!-- 第二行：流转趋势与验真率仪表盘 -->
    <el-row :gutter="20" class="mt-4">
      <el-col :span="16">
        <el-card shadow="hover" class="chart-card">
          <template #header>
            <div class="chart-header-row">
              <span class="chart-title">全局数据流转量与安全风险拦截趋势 (Throughput & Interceptions)</span>
              <el-tag size="small" type="info">最近7天监控流水</el-tag>
            </div>
          </template>
          <div ref="trendChartRef" class="echart-box"></div>
        </el-card>
      </el-col>
      <el-col :span="8">
        <el-card shadow="hover" class="chart-card">
          <template #header>
            <div class="chart-header-row">
              <span class="chart-title">防篡改核验通过率</span>
              <el-tag size="small" :type="stats.tampered_count > 0 ? 'danger' : 'success'">
                {{ stats.tampered_count > 0 ? '检测到篡改告警' : '全量真实完整' }}
              </el-tag>
            </div>
          </template>
          <div ref="gaugeChartRef" class="echart-box"></div>
        </el-card>
      </el-col>
    </el-row>

    <!-- 第三行：机构分布与风险等级分布 -->
    <el-row :gutter="20" class="mt-4">
      <el-col :span="12">
        <el-card shadow="hover" class="chart-card">
          <template #header>
            <div class="chart-title">各医院医疗数据沉淀分布 (Hospital Nodes)</div>
          </template>
          <div ref="hospChartRef" class="echart-box-sm"></div>
        </el-card>
      </el-col>
      <el-col :span="12">
        <el-card shadow="hover" class="chart-card">
          <template #header>
            <div class="chart-title">跨院调阅动态风险评估分布 (Risk Engine)</div>
          </template>
          <div ref="riskChartRef" class="echart-box-sm"></div>
        </el-card>
      </el-col>
    </el-row>

    <!-- 新增行 1：跨院调阅行为时空热力分布 & 跨院医疗数据流向桑基图 -->
    <el-row :gutter="20" class="mt-4">
      <el-col :span="12">
        <el-card shadow="hover" class="chart-card">
          <template #header>
            <div class="chart-header-row">
              <span class="chart-title">跨院调阅行为时空热力图 (24h × 7天 时段异常基线)</span>
              <el-tag size="small" type="primary">行为基线审计</el-tag>
            </div>
          </template>
          <div ref="heatmapChartRef" class="echart-box"></div>
        </el-card>
      </el-col>
      <el-col :span="12">
        <el-card shadow="hover" class="chart-card">
          <template #header>
            <div class="chart-header-row">
              <span class="chart-title">跨机构医疗数据共享流动拓扑 (Cross-Hospital Data Flow)</span>
              <el-tag size="small" type="success">Sankey 拓扑</el-tag>
            </div>
          </template>
          <div ref="sankeyChartRef" class="echart-box"></div>
        </el-card>
      </el-col>
    </el-row>

    <!-- 新增行 2：患者自主知情授权闭环漏斗 & 零信任安全合规效能说明 -->
    <el-row :gutter="20" class="mt-4">
      <el-col :span="12">
        <el-card shadow="hover" class="chart-card">
          <template #header>
            <div class="chart-header-row">
              <span class="chart-title">患者知情授权全流程转化漏斗 (Consent Lifecycle Funnel)</span>
              <el-tag size="small" type="warning">知情同意闭环</el-tag>
            </div>
          </template>
          <div ref="funnelChartRef" class="echart-box" style="height: 280px; min-height: 280px;"></div>
        </el-card>
      </el-col>
      <el-col :span="12">
        <el-card shadow="hover" class="chart-card compliance-card">
          <template #header>
            <div class="chart-header-row">
              <span class="chart-title">零信任数据主权与闭环监管合规效能</span>
              <el-tag size="small" type="success" effect="dark">GDPR / 个人信息保护法</el-tag>
            </div>
          </template>
          <div class="compliance-content">
            <div class="comp-stat-grid">
              <div class="comp-stat-item">
                <div class="comp-stat-val text-primary mono">{{ funnelStats.total }}</div>
                <div class="comp-stat-lbl">医生申请总数</div>
              </div>
              <div class="comp-stat-item">
                <div class="comp-stat-val text-purple mono">{{ funnelStats.handled }}</div>
                <div class="comp-stat-lbl">风控初筛通过</div>
              </div>
              <div class="comp-stat-item">
                <div class="comp-stat-val text-success mono">{{ funnelStats.approved }}</div>
                <div class="comp-stat-lbl">患者知情核准</div>
              </div>
              <div class="comp-stat-item">
                <div class="comp-stat-val text-amber mono">{{ funnelStats.active }}</div>
                <div class="comp-stat-lbl">链上授权固化</div>
              </div>
            </div>
            <div class="comp-rate-bar-box mt-3">
              <div class="comp-rate-row">
                <span>风控初筛通过率（通过 / 申请总数）：</span>
                <strong class="mono">{{ funnelStats.total ? ((funnelStats.handled / funnelStats.total) * 100).toFixed(1) : '0.0' }}%</strong>
              </div>
              <el-progress :percentage="funnelStats.total ? Math.min(100, Math.round((funnelStats.handled / funnelStats.total) * 100)) : 0" :show-text="false" status="success" :stroke-width="10" />
            </div>
            <div class="comp-rate-bar-box mt-3">
              <div class="comp-rate-row">
                <span>患者知情核准率（核准 / 初筛通过）：</span>
                <strong class="mono">{{ funnelStats.handled ? ((funnelStats.approved / funnelStats.handled) * 100).toFixed(1) : '0.0' }}%</strong>
              </div>
              <el-progress :percentage="funnelStats.handled ? Math.min(100, Math.round((funnelStats.approved / funnelStats.handled) * 100)) : 0" :show-text="false" color="#8b5cf6" :stroke-width="10" />
            </div>
            <div class="comp-summary-note mt-3">
              <el-icon class="mr-1 text-emerald-600"><CircleCheckFilled /></el-icon>
              <span>数据主权指标：经患者自主授权激活后，密文授权凭证全生命周期锚定 Fabric 联盟链通道，支持患者随时撤销作废。</span>
            </div>
          </div>
        </el-card>
      </el-col>
    </el-row>

    <!-- 第四行：全网实时安全事件与存证流 (全域安全与联盟链存证审计流) -->
    <el-card shadow="hover" class="chart-card mt-4">
      <template #header>
        <div class="chart-header-row stream-header-wrap">
          <div class="stream-title-box">
            <span class="chart-title">全网实时安全事件与不可篡改审计流 (Real-Time Security Event Stream)</span>
            <span class="stream-live-badge">
              <span class="slb-dot"></span>
              LIVE 持续监听
            </span>
          </div>
          <div class="stream-filter-box">
            <el-radio-group v-model="eventFilter" size="small">
              <el-radio-button label="ALL">全部事件 ({{ securityEvents.length }})</el-radio-button>
              <el-radio-button label="HIGH">高危事件</el-radio-button>
              <el-radio-button label="BREAK_GLASS">破窗调阅</el-radio-button>
              <el-radio-button label="VERIFY">验真审计</el-radio-button>
            </el-radio-group>
            <el-button size="small" type="primary" plain :icon="Refresh" :loading="loadingEvents" @click="fetchSecurityEvents">
              刷新流水
            </el-button>
          </div>
        </div>
      </template>

      <el-table :data="filteredEvents" stripe style="width: 100%" max-height="340" v-loading="loadingEvents">
        <el-table-column prop="id" label="事件ID" width="80" />
        <el-table-column label="发生时间" width="165">
          <template #default="{ row }">
            <span>{{ formatEventTime(row.created_at) }}</span>
          </template>
        </el-table-column>
        <el-table-column label="操作主体" width="140">
          <template #default="{ row }">
            <strong>{{ row.user_name || ('用户#' + row.user_id) }}</strong>
          </template>
        </el-table-column>
        <el-table-column label="医疗机构" width="150">
          <template #default="{ row }">
            <span>{{ row.hospital_name || '联盟机构' }}</span>
          </template>
        </el-table-column>
        <el-table-column label="操作类型" width="130">
          <template #default="{ row }">
            <el-tag size="small" :type="getOpTagType(row.operation_type)">
              {{ formatOpType(row.operation_type) }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="target_id" label="目标病历/事件" min-width="160" />
        <el-table-column label="系统决策结果" width="110">
          <template #default="{ row }">
            <el-tag size="small" :type="row.result === 'SUCCESS' || row.result === 'ALLOWED' ? 'success' : 'danger'" effect="dark">
              {{ row.result === 'SUCCESS' || row.result === 'ALLOWED' ? '核准放行' : '阻断/告警' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column label="风险等级" width="95">
          <template #default="{ row }">
            <el-tag size="small" :type="row.risk_level === 'HIGH' ? 'danger' : (row.risk_level === 'MEDIUM' ? 'warning' : 'info')">
              {{ row.risk_level }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column label="Fabric 存证状态" width="125">
          <template #default="{ row }">
            <el-tag v-if="row.fabric_tx_id" size="small" type="success" effect="plain">
              账本已固化
            </el-tag>
            <el-tag v-else size="small" type="info" effect="plain">
              待同步上链
            </el-tag>
          </template>
        </el-table-column>
      </el-table>
    </el-card>

    <!-- 第五行：联盟链实时交易账本 (blockchain_transactions) 核心共识明细 -->
    <el-card shadow="hover" class="chart-card mt-4">
      <template #header>
        <div class="chart-header-row stream-header-wrap">
          <div class="stream-title-box">
            <span class="chart-title">联盟链实时交易账本与共识日志 (Blockchain Ledger Transactions)</span>
            <el-tag size="small" type="success" effect="dark">Fabric 共识已锚定</el-tag>
          </div>
          <div class="stream-filter-box">
            <el-button size="small" type="primary" plain :icon="Refresh" :loading="loadingTx" @click="fetchBlockchainTransactions">
              刷新账本交易
            </el-button>
          </div>
        </div>
      </template>

      <el-table :data="blockchainTransactions" stripe style="width: 100%" max-height="360" v-loading="loadingTx">
        <el-table-column prop="id" label="序号" width="70" align="center" />
        <el-table-column label="Fabric 交易哈希 (TxID)" min-width="220">
          <template #default="{ row }">
            <code class="tx-hash-mono font-bold" :title="row.tx_id">{{ formatTxId(row.tx_id) }}</code>
          </template>
        </el-table-column>
        <el-table-column prop="block_height" label="区块高度" width="110" align="center">
          <template #default="{ row }">
            <el-tag size="small" type="primary" effect="plain" class="mono font-semibold">
              {{ row.block_height ? `#${row.block_height}` : '待同步' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column label="通道 / 智能合约" width="200">
          <template #default="{ row }">
            <span class="text-xs font-semibold text-purple-700 mono">{{ (row.channel_name === 'medchannel' || !row.channel_name) ? 'medicalchannel' : row.channel_name }}</span>
            <span class="text-xs text-slate-400"> / </span>
            <span class="text-xs font-semibold text-indigo-600 mono">{{ (row.chaincode_name === 'medical' || !row.chaincode_name) ? 'medical_contract' : row.chaincode_name }}</span>
          </template>
        </el-table-column>
        <el-table-column label="智能合约操作方法" width="170">
          <template #default="{ row }">
            <el-tag size="small" :type="getTxFunctionTagType(row.function_name || row.business_type)" effect="light">
              {{ row.function_name || row.business_type || 'InvokeContract' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column label="存证负载摘要 (Payload Hash)" width="180">
          <template #default="{ row }">
            <code class="text-xs mono text-slate-600" :title="row.payload_hash">{{ formatPayloadHash(row.payload_hash) }}</code>
          </template>
        </el-table-column>
        <el-table-column label="共识背书状态" width="120" align="center">
          <template #default="{ row }">
            <el-tag size="small" :type="row.status === 'VALID' || row.status === 'CONFIRMED' || row.status === 'SUCCESS' || !row.status ? 'success' : 'danger'" effect="dark">
              {{ row.status || 'CONFIRMED' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column label="共识确认时间" width="170">
          <template #default="{ row }">
            <span class="text-xs text-slate-600">{{ formatEventTime(row.created_at) }}</span>
          </template>
        </el-table-column>
      </el-table>
    </el-card>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted, nextTick } from 'vue'
import { useRouter } from 'vue-router'
import { Refresh } from '@element-plus/icons-vue'
import * as echarts from 'echarts'

const router = useRouter()
import {
  getBlockchainStatusApi,
  listAuditLogsApi,
  listBlockchainTransactionsApi,
  getSupervisorOverviewApi,
  getAccessHeatmapApi,
  getHospitalFlowApi,
  getAuthFunnelApi
} from '../../api'

interface BlockchainStatus {
  mode: string
  connected: boolean
  network: string
  chaincode: string
  peer: string
  latest_block: number
  message: string
}

const chainStatus = ref<BlockchainStatus>({
  mode: 'fabric',
  connected: false,
  network: 'medchannel',
  chaincode: 'medical',
  peer: 'peer0.org1.example.com',
  latest_block: 0,
  message: '正在检测底层区块链状态...',
})
const refreshingStatus = ref(false)

async function fetchBlockchainStatus() {
  refreshingStatus.value = true
  try {
    const res: any = await getBlockchainStatusApi()
    const st = (res && res.data !== undefined) ? res.data : res
    if (st && st.mode) {
      chainStatus.value = st
      if (st.latest_block !== undefined && st.latest_block > 0) {
        stats.value.block_height = st.latest_block
      }
    }
  } catch (e) {
    chainStatus.value = {
      mode: 'fabric',
      connected: false,
      network: 'medchannel',
      chaincode: 'medical',
      peer: 'peer0.org1.example.com',
      latest_block: stats.value.block_height || 0,
      message: 'Fabric Unavailable: 网络连接断开',
    }
  } finally {
    refreshingStatus.value = false
  }
}

const stats = ref<any>({
  total_records: 0,
  total_tx: 0,
  block_height: 0,
  total_emergencies: 0,
  pending_audits: 0,
  total_logs: 0,
  verification_pass_rate: 0,
  verified_count: 0,
  tampered_count: 0,
  total_intercepted: 0,
})

const trendChartRef = ref<HTMLDivElement | null>(null)
const gaugeChartRef = ref<HTMLDivElement | null>(null)
const hospChartRef = ref<HTMLDivElement | null>(null)
const riskChartRef = ref<HTMLDivElement | null>(null)

// 专项三新增高级监管分析图表 Refs
const heatmapChartRef = ref<HTMLDivElement | null>(null)
const sankeyChartRef = ref<HTMLDivElement | null>(null)
const funnelChartRef = ref<HTMLDivElement | null>(null)

const funnelStats = ref({
  total: 0,
  handled: 0,
  approved: 0,
  active: 0,
})

const chartInstances: echarts.ECharts[] = []

function getOrInitChart(el: HTMLElement): echarts.ECharts {
  const existing = echarts.getInstanceByDom(el)
  if (existing) return existing
  const created = echarts.init(el)
  chartInstances.push(created)
  return created
}

const securityEvents = ref<any[]>([])
const loadingEvents = ref(false)
const eventFilter = ref('ALL')
let pollTimer: ReturnType<typeof setInterval> | null = null

const filteredEvents = computed(() => {
  const f = eventFilter.value
  if (f === 'HIGH') {
    return securityEvents.value.filter(e => e.risk_level === 'HIGH' || (e.result !== 'SUCCESS' && e.result !== 'ALLOWED'))
  } else if (f === 'BREAK_GLASS') {
    return securityEvents.value.filter(e => e.operation_type === 'BREAK_GLASS' || e.operation_type === 'EMERGENCY_AUDIT')
  } else if (f === 'VERIFY') {
    return securityEvents.value.filter(e => e.operation_type === 'VERIFY')
  }
  return securityEvents.value
})

async function fetchSecurityEvents(silent = false) {
  if (!silent) loadingEvents.value = true
  try {
    const res: any = await listAuditLogsApi()
    if (res.code === 200 && Array.isArray(res.data)) {
      securityEvents.value = res.data
    } else if (!silent) {
      securityEvents.value = []
    }
  } catch (err) {
    console.warn('fetchSecurityEvents error', err)
    if (!silent) securityEvents.value = []
  } finally {
    if (!silent) loadingEvents.value = false
  }
}

function formatEventTime(dt?: string) {
  if (!dt) return '-'
  return String(dt).replace('T', ' ').slice(0, 19)
}

function getOpTagType(op?: string) {
  if (!op) return 'info'
  if (op.includes('BREAK') || op.includes('EMERGENCY')) return 'danger'
  if (op.includes('GRANT') || op.includes('UPLOAD')) return 'success'
  if (op.includes('VERIFY')) return 'warning'
  if (op.includes('ACCESS')) return 'primary'
  if (op.includes('LOGIN')) return 'info'
  return 'primary'
}

function formatOpType(op?: string) {
  const map: Record<string, string> = {
    'UPLOAD': '病历加密上链 (UPLOAD)',
    'GRANT': '跨院授权核准 (GRANT)',
    'ACCESS': '病历权限调阅 (ACCESS)',
    'BREAK_GLASS': '紧急破窗调阅 (BREAK_GLASS)',
    'VERIFY': '动态验真比对 (VERIFY)',
    'AUDIT_CLOSE': '监管合规裁决 (DECIDE)',
    'UNRESTRICT_DOCTOR': '解除执业惩戒 (RESTORE)',
    'FEEDBACK': '患者异议申诉 (FEEDBACK)',
    'LOGIN': '安全身份认证 (LOGIN)'
  }
  return map[op || ''] || op || '系统安全审计'
}

const blockchainTransactions = ref<any[]>([])
const loadingTx = ref(false)

async function fetchBlockchainTransactions(silent = false) {
  if (!silent) loadingTx.value = true
  try {
    const res: any = await listBlockchainTransactionsApi(1, 30)
    if (res.code === 200 && res.data) {
      blockchainTransactions.value = Array.isArray(res.data) ? res.data : (res.data.items || res.data.list || [])
    }
  } catch (err) {
    console.warn('fetchBlockchainTransactions fallback error', err)
    if (!silent) blockchainTransactions.value = []
  } finally {
    if (!silent) loadingTx.value = false
  }
}

function formatTxId(txId?: string) {
  if (!txId) return '—'
  const clean = String(txId).replace(/^0x/i, '')
  if (clean.length <= 20) return clean
  return `${clean.slice(0, 10)}...${clean.slice(-8)}`
}

function formatPayloadHash(hash?: string) {
  if (!hash) return '—'
  if (hash.length <= 16) return hash
  return `${hash.slice(0, 8)}...${hash.slice(-6)}`
}

function getTxFunctionTagType(fn?: string) {
  if (!fn) return 'info'
  const f = fn.toLowerCase()
  if (f.includes('break') || f.includes('emergency')) return 'danger'
  if (f.includes('upload') || f.includes('create')) return 'success'
  if (f.includes('grant') || f.includes('consent') || f.includes('approve')) return 'primary'
  if (f.includes('verify')) return 'warning'
  return 'info'
}

onMounted(async () => {
  fetchBlockchainStatus()
  fetchSecurityEvents()
  fetchBlockchainTransactions()
  // 静默后台轮询且在标签页不可见时暂停，避免每 10 秒闪现表格 Loading 遮罩打断阅读
  pollTimer = setInterval(() => {
    if (document.hidden) return
    fetchSecurityEvents(true)
    fetchBlockchainStatus()
    fetchBlockchainTransactions(true)
  }, 10000)

  try {
    const res: any = await getSupervisorOverviewApi()
    if (res.code === 200) {
      stats.value = res.data
      if (chainStatus.value.latest_block > 0) {
        stats.value.block_height = chainStatus.value.latest_block
      }
      await nextTick()
      renderCharts(res.data)
    }
  } catch (err) {
    console.error(err)
  }

  // 加载专项三新增的高级可视化图表（时空热力图、跨院桑基图、授权闭环漏斗）
  await loadAdvancedCharts()

  window.addEventListener('resize', handleWindowResize)
})

function handleWindowResize() {
  chartInstances.forEach(c => {
    try {
      c && c.resize()
    } catch {}
  })
}

onUnmounted(() => {
  if (pollTimer) {
    clearInterval(pollTimer)
    pollTimer = null
  }
  window.removeEventListener('resize', handleWindowResize)
  chartInstances.forEach(c => {
    try {
      c && c.dispose()
    } catch {}
  })
})

function renderCharts(data: any) {
  // 1. 流转趋势与风险拦截 (折线柱状组合图，严格展示后端真实统计)
  if (trendChartRef.value) {
    const cTrend = getOrInitChart(trendChartRef.value)
    const trend = Array.isArray(data?.throughput_trend) ? data.throughput_trend : []
    const dates = trend.map((t: any) => t.date)
    const flows = trend.map((t: any) => t.flow_count)
    const intercepts = trend.map((t: any) => t.intercept_count)

    cTrend.setOption({
      tooltip: { trigger: 'axis', axisPointer: { type: 'cross' } },
      legend: { data: ['业务调阅流转总量', '安全风险拦截次数'], bottom: 0 },
      grid: { left: '3%', right: '4%', bottom: '12%', top: '10%', containLabel: true },
      xAxis: { type: 'category', data: dates },
      yAxis: [{ type: 'value', name: '次/日', minInterval: 1 }],
      graphic: dates.length === 0 ? [{ type: 'text', left: 'center', top: 'middle', style: { text: '暂无趋势统计数据', fill: '#94a3b8', fontSize: 13 } }] : [],
      series: [
        {
          name: '业务调阅流转总量',
          type: 'line',
          smooth: true,
          data: flows,
          itemStyle: { color: '#3b82f6' },
          areaStyle: {
            color: new echarts.graphic.LinearGradient(0, 0, 0, 1, [
              { offset: 0, color: 'rgba(59,130,246,0.35)' },
              { offset: 1, color: 'rgba(59,130,246,0.02)' }
            ])
          }
        },
        {
          name: '安全风险拦截次数',
          type: 'bar',
          barWidth: '24%',
          data: intercepts,
          itemStyle: { color: '#ef4444', borderRadius: [4, 4, 0, 0] }
        }
      ]
    }, true)
  }

  // 2. 防篡改核验通过率仪表盘（动态读取后端 verification_pass_rate）
  if (gaugeChartRef.value) {
    const cGauge = getOrInitChart(gaugeChartRef.value)
    const rawRate = Number(data?.verification_pass_rate ?? stats.value.verification_pass_rate ?? 0)
    const passRate = Number((Number.isFinite(rawRate) ? rawRate : 0).toFixed(1))
    cGauge.setOption({
      series: [
        {
          type: 'gauge',
          startAngle: 180,
          endAngle: 0,
          center: ['50%', '75%'],
          radius: '100%',
          min: 0,
          max: 100,
          splitNumber: 5,
          axisLine: {
            lineStyle: {
              width: 16,
              color: [
                [0.7, '#ef4444'],
                [0.9, '#f59e0b'],
                [1, '#10b981']
              ]
            }
          },
          pointer: { length: '55%', width: 5, itemStyle: { color: '#0284c7' } },
          title: { offsetCenter: [0, '35%'], fontSize: 13, color: '#64748b' },
          detail: {
            fontSize: 26,
            offsetCenter: [0, '65%'],
            valueAnimation: true,
            formatter: `${passRate}%`,
            color: passRate >= 90 ? '#10b981' : (passRate >= 70 ? '#f59e0b' : '#ef4444')
          },
          data: [{ value: passRate, name: '真实完整率' }]
        }
      ]
    }, true)
  }

  // 3. 医院分布 (环形饼图)
  if (hospChartRef.value) {
    const c1 = getOrInitChart(hospChartRef.value)
    const hospData = (Array.isArray(data?.hosp_stats) ? data.hosp_stats : []).map((item: any) => ({
      name: item.name,
      value: item.count,
    }))
    c1.setOption({
      tooltip: { trigger: 'item' },
      legend: { bottom: 0 },
      graphic: hospData.length === 0 ? [{ type: 'text', left: 'center', top: 'middle', style: { text: '暂无机构分布数据', fill: '#94a3b8', fontSize: 13 } }] : [],
      series: [
        {
          name: '存证数量',
          type: 'pie',
          radius: ['42%', '68%'],
          center: ['50%', '45%'],
          avoidLabelOverlap: true,
          itemStyle: { borderRadius: 8, borderColor: '#fff', borderWidth: 2 },
          data: hospData,
        },
      ],
    }, true)
  }

  // 4. 风险分布 (饼图，绑定后端真实 risk_stats)
  if (riskChartRef.value) {
    const c2 = getOrInitChart(riskChartRef.value)
    const riskLabelMap: Record<string, string> = {
      LOW: '低风险 (0-29分: 放行)',
      MEDIUM: '中风险 (30-59分: 确认)',
      HIGH: '高风险 (≥60分: 拦截)',
      EMERGENCY: '紧急破窗 (EMERGENCY)',
    }
    const riskStatsRaw = Array.isArray(data?.risk_stats) ? data.risk_stats : []
    const riskData = riskStatsRaw
      .filter((item: any) => item && item.count > 0)
      .map((item: any) => ({
        name: riskLabelMap[String(item.risk_level || '').toUpperCase()] || item.risk_level || '未知',
        value: item.count,
      }))
    c2.setOption({
      tooltip: { trigger: 'item' },
      legend: { bottom: 0 },
      color: ['#10b981', '#f59e0b', '#ef4444', '#8b5cf6'],
      graphic: riskData.length === 0 ? [{ type: 'text', left: 'center', top: 'middle', style: { text: '暂无风险分布数据', fill: '#94a3b8', fontSize: 13 } }] : [],
      series: [
        {
          name: '风险评级',
          type: 'pie',
          radius: '62%',
          center: ['50%', '45%'],
          data: riskData,
          emphasis: {
            itemStyle: { shadowBlur: 10, shadowOffsetX: 0, shadowColor: 'rgba(0, 0, 0, 0.5)' },
          },
        },
      ],
    }, true)
  }
}

// ── 专项三：高级可视化图表加载与渲染 ───────────────────────
async function loadAdvancedCharts() {
  try {
    const [heatRes, flowRes, funnelRes]: any = await Promise.all([
      getAccessHeatmapApi().catch(() => ({ code: 200, data: [] as any })),
      getHospitalFlowApi().catch(() => ({ code: 200, data: [] as any })),
      getAuthFunnelApi().catch(() => ({ code: 200, data: null as any })),
    ])

    await nextTick()

    if (heatRes && heatRes.data !== undefined) {
      renderHeatmapChart(heatRes.data)
    }
    if (flowRes && flowRes.data !== undefined) {
      renderSankeyChart(flowRes.data)
    }
    if (funnelRes && funnelRes.data) {
      funnelStats.value = funnelRes.data
      renderFunnelChart(funnelRes.data)
    }
  } catch (err) {
    console.warn('loadAdvancedCharts error', err)
  }
}

// 1. 跨院调阅行为时空热力图 (24h × 7天 真实审计频次)
function renderHeatmapChart(data: [number, number, number][]) {
  if (!heatmapChartRef.value) return
  const chart = getOrInitChart(heatmapChartRef.value)
  const hours = Array.from({ length: 24 }, (_, i) => `${i}:00`)
  const days = ['周一', '周二', '周三', '周四', '周五', '周六', '周日']
  const validData = Array.isArray(data) ? data : []
  const maxVal = validData.length > 0 ? Math.max(...validData.map((d: any) => d[2] || 0), 1) : 1

  chart.setOption({
    tooltip: {
      position: 'top',
      formatter: (params: any) => {
        const d = params.data
        return `${days[d[1]]} ${hours[d[0]]}<br/>跨院调阅频次: <b>${d[2]} 次</b>`
      }
    },
    grid: { height: '65%', top: '8%', bottom: '20%', left: '8%', right: '5%' },
    xAxis: { type: 'category', data: hours, splitArea: { show: true }, axisLabel: { interval: 2, fontSize: 10 } },
    yAxis: { type: 'category', data: days, splitArea: { show: true }, axisLabel: { fontSize: 11 } },
    visualMap: {
      min: 0,
      max: maxVal,
      calculable: true,
      orient: 'horizontal',
      left: 'center',
      bottom: '1%',
      itemWidth: 12,
      itemHeight: 120,
      inRange: {
        color: ['#f8fafc', '#bfdbfe', '#3b82f6', '#1d4ed8', '#7f1d1d']
      }
    },
    series: [{
      name: '时空调阅热度',
      type: 'heatmap',
      data: validData,
      label: { show: false },
      emphasis: {
        itemStyle: { shadowBlur: 10, shadowColor: 'rgba(0, 0, 0, 0.5)' }
      }
    }]
  }, true)
}

// 2. 跨机构医疗数据流向拓扑 (Sankey 桑基图)
function renderSankeyChart(flowList: any[]) {
  if (!sankeyChartRef.value) return
  const chart = getOrInitChart(sankeyChartRef.value)
  const flows = Array.isArray(flowList) ? flowList.filter((f: any) => f && f.source && f.target && f.source !== f.target && f.value > 0) : []

  const nodeSet = new Set<string>()
  flows.forEach((f: any) => {
    nodeSet.add(f.source)
    nodeSet.add(f.target)
  })
  const nodes = Array.from(nodeSet).map(name => ({ name }))

  chart.setOption({
    tooltip: {
      trigger: 'item',
      triggerOn: 'mousemove',
      formatter: (params: any) => {
        if (params.dataType === 'edge') {
          return `数据调阅流向: <b>${params.data.source}</b> → <b>${params.data.target}</b><br/>共识上链记录: <b>${params.data.value} 份</b>`
        }
        return `医疗机构节点: <b>${params.name}</b>`
      }
    },
    graphic: flows.length === 0 ? [{ type: 'text', left: 'center', top: 'middle', style: { text: '暂无跨院数据流向记录', fill: '#94a3b8', fontSize: 13 } }] : [],
    series: flows.length > 0 ? [{
      type: 'sankey',
      layout: 'none',
      emphasis: { focus: 'adjacency' },
      data: nodes,
      links: flows,
      top: '12%',
      bottom: '12%',
      left: '4%',
      right: '16%',
      nodeWidth: 16,
      nodeGap: 16,
      itemStyle: { borderWidth: 1, borderColor: '#cbd5e1' },
      lineStyle: { color: 'gradient', curveness: 0.5, opacity: 0.45 },
      label: { color: '#0f172a', fontSize: 11, fontWeight: 'bold' }
    }] : []
  }, true)
}

// 3. 患者自主知情授权闭环漏斗 (Funnel 漏斗图，严格反映后端真实数据)
function renderFunnelChart(funnelData: any) {
  if (!funnelChartRef.value) return
  const chart = getOrInitChart(funnelChartRef.value)
  const f = {
    total: Number(funnelData?.total || 0),
    handled: Number(funnelData?.handled || 0),
    approved: Number(funnelData?.approved || 0),
    active: Number(funnelData?.active || 0),
  }
  funnelStats.value = f

  chart.setOption({
    tooltip: {
      trigger: 'item',
      formatter: '{a} <br/>{b} : <b>{c} 笔</b>'
    },
    legend: {
      data: ['发起跨院申请', '风控初筛通过', '患者知情核准', '链上授权固化'],
      bottom: 0,
      itemWidth: 12,
      itemHeight: 12,
      textStyle: { fontSize: 11 }
    },
    series: [{
      name: '知情授权闭环漏斗',
      type: 'funnel',
      left: '10%',
      top: 15,
      bottom: 35,
      width: '80%',
      min: 0,
      max: f.total > 0 ? f.total : 1,
      minSize: '18%',
      maxSize: '100%',
      sort: 'descending',
      gap: 3,
      label: {
        show: true,
        position: 'inside',
        formatter: '{b}\n{c} 笔',
        fontSize: 11,
        color: '#ffffff',
        fontWeight: 'bold'
      },
      labelLine: { length: 10, lineStyle: { width: 1, type: 'solid' } },
      itemStyle: { borderColor: '#fff', borderWidth: 1.5 },
      data: [
        { value: f.total, name: '发起跨院申请', itemStyle: { color: '#3b82f6' } },
        { value: f.handled, name: '风控初筛通过', itemStyle: { color: '#8b5cf6' } },
        { value: f.approved, name: '患者知情核准', itemStyle: { color: '#10b981' } },
        { value: f.active, name: '链上授权固化', itemStyle: { color: '#f59e0b' } },
      ]
    }]
  }, true)
}
</script>

<style scoped>
.page-container {
  max-width: 1350px;
  margin: 0 auto;
}
.page-header {
  margin-bottom: 18px;
}
.page-title {
  font-size: 22px;
  font-weight: 800;
  color: #0f172a;
}
.page-sub {
  font-size: 13px;
  color: #64748b;
  margin-top: 4px;
}
.metric-row {
  margin-bottom: 16px;
}
.metric-card {
  border-radius: 8px;
}
.metric-title {
  font-size: 13px;
  color: #64748b;
  font-weight: 600;
  margin-bottom: 6px;
}
.metric-num {
  font-size: 28px;
  font-weight: 800;
  margin-bottom: 6px;
}
.text-primary { color: #0284c7; }
.text-purple { color: #0369a1; }
.text-danger { color: #dc2626; }
.text-success { color: #059669; }
.metric-foot {
  font-size: 12px;
  color: #64748b;
}
.chart-card {
  border-radius: 8px;
}
.chart-header-row {
  display: flex;
  justify-content: space-between;
  align-items: center;
}
.chart-title {
  font-weight: 700;
  color: #1e293b;
  font-size: 14px;
}
.echart-box {
  width: 100%;
  height: 290px;
}
.echart-box-sm {
  width: 100%;
  height: 250px;
}
.mt-4 {
  margin-top: 16px;
}

.blockchain-status-card {
  border-radius: 6px;
  background: #ffffff;
  color: #0f172a;
  border: 1px solid #e2e8f0;
  border-left: 4px solid var(--mt-primary, #0284c7);
}
.mb-4 {
  margin-bottom: 16px;
}
.chain-status-container {
  display: flex;
  justify-content: space-between;
  align-items: center;
  flex-wrap: wrap;
  gap: 16px;
  padding: 4px 6px;
}
.chain-status-left {
  display: flex;
  flex-direction: column;
  gap: 4px;
}
.chain-badge-wrapper {
  display: flex;
  align-items: center;
  gap: 10px;
}
.status-pulse-dot {
  width: 10px;
  height: 10px;
  border-radius: 50%;
  display: inline-block;
  position: relative;
}
.dot-green {
  background-color: #059669;
}
.dot-amber {
  background-color: #d97706;
}
.dot-red {
  background-color: #dc2626;
}

.status-title-text {
  font-size: 15px;
  font-weight: 700;
  letter-spacing: 0.3px;
}
.text-status-green { color: #059669; }
.text-status-amber { color: #d97706; }
.text-status-red { color: #dc2626; }

.chain-sub-desc {
  font-size: 12px;
  color: #64748b;
}
.chain-status-meta {
  display: flex;
  align-items: center;
  gap: 20px;
  flex-wrap: wrap;
}
.meta-item {
  display: flex;
  flex-direction: column;
  gap: 4px;
}
.meta-label {
  font-size: 11px;
  color: #64748b;
  text-transform: uppercase;
  letter-spacing: 0.5px;
}
.meta-val {
  font-size: 13px;
  font-weight: 600;
  color: #1e293b;
  font-family: var(--mt-font-mono, monospace);
}
.meta-val.highlight {
  color: #0284c7;
}
.meta-val.block-height {
  color: #0369a1;
  font-size: 15px;
}
.action-box {
  align-self: center;
}

.stream-header-wrap {
  display: flex;
  justify-content: space-between;
  align-items: center;
  flex-wrap: wrap;
  gap: 10px;
}
.stream-title-box {
  display: flex;
  align-items: center;
  gap: 8px;
}
.stream-icon {
  font-size: 16px;
}
.stream-live-badge {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  background: #ecfdf5;
  color: #059669;
  font-size: 11px;
  font-weight: 700;
  padding: 2px 8px;
  border-radius: 9999px;
  border: 1px solid #a7f3d0;
}
.slb-dot {
  width: 7px;
  height: 7px;
  border-radius: 50%;
  background: #10b981;
  animation: pulse-green 1.8s infinite;
}
.stream-filter-box {
  display: flex;
  align-items: center;
  gap: 10px;
}

/* 专项三新增样式 */
.compliance-card {
  min-height: 350px;
}
.compliance-content {
  padding: 8px 4px;
}
.comp-stat-grid {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 10px;
  background: #f8fafc;
  padding: 12px;
  border-radius: 10px;
  border: 1px solid #e2e8f0;
}
.comp-stat-item {
  text-align: center;
}
.comp-stat-val {
  font-size: 22px;
  font-weight: 900;
  line-height: 1.2;
}
.comp-stat-lbl {
  font-size: 11px;
  color: #64748b;
  margin-top: 4px;
  font-weight: 600;
}
.comp-rate-bar-box {
  background: #ffffff;
  padding: 10px 14px;
  border-radius: 8px;
  border: 1px solid #f1f5f9;
}
.comp-rate-row {
  display: flex;
  justify-content: space-between;
  font-size: 12px;
  color: #334155;
  margin-bottom: 6px;
}
.comp-summary-note {
  display: flex;
  align-items: flex-start;
  gap: 6px;
  font-size: 12px;
  color: #047857;
  background: #ecfdf5;
  padding: 10px 12px;
  border-radius: 8px;
  border: 1px solid #a7f3d0;
  line-height: 1.5;
}
.text-amber { color: #f59e0b; }

@keyframes pulse-green {
  0%, 100% {
    opacity: 1;
    transform: scale(1);
  }
  50% {
    opacity: 0.45;
    transform: scale(1.25);
  }
}
</style>
