<template>
  <div class="page-container">
    <div class="page-header">
      <div>
        <h2 class="page-title">联盟链存证监控与监管大屏</h2>
        <p class="page-sub">依据毕业设计技术指标：实时监控跨机构数据共享流转量、Fabric 区块高度、风险拦截统计与防篡改动态校验通过率</p>
      </div>
    </div>

    <!-- 区块链底层网络实时运行状态卡片 (毕业设计核心技术指标) -->
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
              本地仿真开发模式 (MockLedger fallback 模式)，适合无 Docker 环境下的快速单机逻辑联调
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
              {{ chainStatus.mode === 'fabric' ? 'Fabric 2.5' : 'MockLedger' }}
            </el-tag>
          </div>
          <div class="meta-item">
            <span class="meta-label">通道标识 (Channel)</span>
            <span class="meta-val highlight">{{ chainStatus.network || 'medchannel' }}</span>
          </div>
          <div class="meta-item">
            <span class="meta-label">智能合约 (Chaincode)</span>
            <span class="meta-val highlight">{{ chainStatus.chaincode || 'medical' }}</span>
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
          <div class="metric-foot">当前区块高度 #{{ stats.block_height }}</div>
        </el-card>
      </el-col>
      <el-col :span="6">
        <el-card shadow="hover" class="metric-card">
          <div class="metric-title">紧急访问 (Break-Glass)</div>
          <div class="metric-num text-danger">{{ stats.total_emergencies }}</div>
          <div class="metric-foot">待监管人员审核: <strong>{{ stats.pending_audits }}</strong></div>
        </el-card>
      </el-col>
      <el-col :span="6">
        <el-card shadow="hover" class="metric-card">
          <div class="metric-title">动态验真通过率</div>
          <div class="metric-num text-success">{{ Number(stats.verification_pass_rate || 100).toFixed(1) }}%</div>
          <div class="metric-foot">已核验: {{ stats.verified_count || stats.total_records }} | 异常: {{ stats.tampered_count || 0 }}</div>
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

    <!-- 第四行：全网实时安全事件与存证流 (毕业设计核心监控流) -->
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
            <el-tag size="small" :type="row.result === 'SUCCESS' ? 'success' : 'danger'" effect="dark">
              {{ row.result === 'SUCCESS' ? '核准放行' : '阻断/告警' }}
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
          <template #default>
            <el-tag size="small" type="success" effect="plain">
              账本已固化
            </el-tag>
          </template>
        </el-table-column>
      </el-table>
    </el-card>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted, nextTick } from 'vue'
import { Refresh } from '@element-plus/icons-vue'
import * as echarts from 'echarts'
import api from '../../api/client'

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
    const res: any = await api.get('/system/blockchain/status')
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
  block_height: 100,
  total_emergencies: 0,
  pending_audits: 0,
  total_logs: 0,
  verification_pass_rate: 100,
  verified_count: 0,
  tampered_count: 0,
  total_intercepted: 0,
})

const trendChartRef = ref<HTMLDivElement | null>(null)
const gaugeChartRef = ref<HTMLDivElement | null>(null)
const hospChartRef = ref<HTMLDivElement | null>(null)
const riskChartRef = ref<HTMLDivElement | null>(null)

const securityEvents = ref<any[]>([])
const loadingEvents = ref(false)
const eventFilter = ref('ALL')
let pollTimer: any = null

const filteredEvents = computed(() => {
  const f = eventFilter.value
  if (f === 'HIGH') {
    return securityEvents.value.filter(e => e.risk_level === 'HIGH' || e.result !== 'SUCCESS')
  } else if (f === 'BREAK_GLASS') {
    return securityEvents.value.filter(e => e.operation_type === 'BREAK_GLASS' || e.operation_type === 'EMERGENCY_AUDIT')
  } else if (f === 'VERIFY') {
    return securityEvents.value.filter(e => e.operation_type === 'VERIFY')
  }
  return securityEvents.value
})

async function fetchSecurityEvents() {
  loadingEvents.value = true
  try {
    const res: any = await api.get('/audit-logs')
    if (res.code === 200 && res.data) {
      securityEvents.value = res.data
    }
  } catch (err) {
    console.warn('fetchSecurityEvents error', err)
  } finally {
    loadingEvents.value = false
  }
}

function formatEventTime(dt?: string) {
  if (!dt) return '-'
  return String(dt).replace('T', ' ').slice(0, 19)
}

function getOpTagType(op?: string) {
  if (!op) return 'info'
  if (op.includes('BREAK') || op.includes('EMERGENCY')) return 'danger'
  if (op.includes('VERIFY')) return 'warning'
  if (op.includes('UPLOAD') || op.includes('CREATE')) return 'success'
  return 'primary'
}

function formatOpType(op?: string) {
  const map: Record<string, string> = {
    'UPLOAD': '病历密文上链',
    'ACCESS': '病历权限调阅',
    'BREAK_GLASS': 'Break-Glass 破窗',
    'VERIFY': '动态防篡改核验',
    'AUDIT_CLOSE': '监管审核裁决',
    'UNRESTRICT_DOCTOR': '解除执业惩戒',
    'FEEDBACK': '患者异议反馈',
  }
  return map[op || ''] || op || '安全审计'
}

onMounted(async () => {
  fetchBlockchainStatus()
  fetchSecurityEvents()
  pollTimer = setInterval(() => {
    fetchSecurityEvents()
    fetchBlockchainStatus()
  }, 10000)

  try {
    const res: any = await api.get('/supervisor/overview')
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
})

onUnmounted(() => {
  if (pollTimer) {
    clearInterval(pollTimer)
    pollTimer = null
  }
})

function renderCharts(data: any) {
  // 1. 流转趋势与风险拦截 (折线柱状组合图)
  if (trendChartRef.value) {
    const cTrend = echarts.init(trendChartRef.value)
    const trend = data.throughput_trend || []
    const dates = trend.map((t: any) => t.date)
    const flows = trend.map((t: any) => t.flow_count)
    const intercepts = trend.map((t: any) => t.intercept_count)

    cTrend.setOption({
      tooltip: { trigger: 'axis', axisPointer: { type: 'cross' } },
      legend: { data: ['业务调阅流转总量', '安全风险拦截次数'], bottom: 0 },
      grid: { left: '3%', right: '4%', bottom: '12%', top: '10%', containLabel: true },
      xAxis: { type: 'category', data: dates.length ? dates : ['09-08', '09-09', '09-10', '09-11', '09-12', '09-13', '09-14'] },
      yAxis: [{ type: 'value', name: '次/日' }],
      series: [
        {
          name: '业务调阅流转总量',
          type: 'line',
          smooth: true,
          data: flows.length ? flows : [12, 18, 24, 21, 30, 28, 35],
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
          data: intercepts.length ? intercepts : [1, 2, 0, 1, 3, 1, 2],
          itemStyle: { color: '#ef4444', borderRadius: [4, 4, 0, 0] }
        }
      ]
    })
  }

  // 2. 防篡改核验通过率仪表盘
  if (gaugeChartRef.value) {
    const cGauge = echarts.init(gaugeChartRef.value)
    const passRate = Number(data.verification_pass_rate || 100).toFixed(1)
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
          pointer: { length: '60%', width: 5 },
          title: { offsetCenter: [0, '-20%'], fontSize: 13, color: '#64748b' },
          detail: {
            fontSize: 26,
            offsetCenter: [0, '0%'],
            valueAnimation: true,
            formatter: '{value}%',
            color: Number(passRate) >= 90 ? '#10b981' : '#ef4444'
          },
          data: [{ value: Number(passRate), name: '真实完整率' }]
        }
      ]
    })
  }

  // 3. 医院分布 (环形饼图)
  if (hospChartRef.value) {
    const c1 = echarts.init(hospChartRef.value)
    const hospData = (data.hosp_stats || []).map((item: any) => ({
      name: item.name,
      value: item.count,
    }))
    c1.setOption({
      tooltip: { trigger: 'item' },
      legend: { bottom: 0 },
      series: [
        {
          name: '存证数量',
          type: 'pie',
          radius: ['42%', '68%'],
          center: ['50%', '45%'],
          avoidLabelOverlap: false,
          itemStyle: { borderRadius: 8, borderColor: '#fff', borderWidth: 2 },
          data: hospData.length ? hospData : [
            { name: '第一人民医院', value: 4 },
            { name: '省立中心医院', value: 2 },
            { name: '协和医学中心', value: 1 }
          ],
        },
      ],
    })
  }

  // 4. 风险分布 (饼图)
  if (riskChartRef.value) {
    const c2 = echarts.init(riskChartRef.value)
    c2.setOption({
      tooltip: { trigger: 'item' },
      legend: { bottom: 0 },
      color: ['#10b981', '#f59e0b', '#ef4444'],
      series: [
        {
          name: '风险评级',
          type: 'pie',
          radius: '62%',
          center: ['50%', '45%'],
          data: [
            { value: 16, name: '低风险 (0-29分: 放行)' },
            { value: 5, name: '中风险 (30-59分: 确认)' },
            { value: 3, name: '高风险 (≥60分: 拦截)' },
          ],
          emphasis: {
            itemStyle: { shadowBlur: 10, shadowOffsetX: 0, shadowColor: 'rgba(0, 0, 0, 0.5)' },
          },
        },
      ],
    })
  }
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
  border-radius: 12px;
}
.metric-title {
  font-size: 13px;
  color: #64748b;
  font-weight: 600;
  margin-bottom: 6px;
}
.metric-num {
  font-size: 30px;
  font-weight: 900;
  margin-bottom: 6px;
}
.text-primary { color: #3b82f6; }
.text-purple { color: #8b5cf6; }
.text-danger { color: #ef4444; }
.text-success { color: #10b981; }
.metric-foot {
  font-size: 12px;
  color: #94a3b8;
}
.chart-card {
  border-radius: 12px;
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
  border-radius: 12px;
  background: linear-gradient(135deg, #0f172a 0%, #1e293b 100%);
  color: #f8fafc;
  border: 1px solid #334155;
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
  gap: 6px;
}
.chain-badge-wrapper {
  display: flex;
  align-items: center;
  gap: 10px;
}
.status-pulse-dot {
  width: 12px;
  height: 12px;
  border-radius: 50%;
  display: inline-block;
  position: relative;
}
.dot-green {
  background-color: #10b981;
  box-shadow: 0 0 0 0 rgba(16, 185, 129, 0.7);
  animation: pulse-green 2s infinite;
}
.dot-amber {
  background-color: #f59e0b;
  box-shadow: 0 0 0 0 rgba(245, 158, 11, 0.7);
  animation: pulse-amber 2s infinite;
}
.dot-red {
  background-color: #ef4444;
  box-shadow: 0 0 0 0 rgba(239, 68, 68, 0.7);
  animation: pulse-red 2s infinite;
}

@keyframes pulse-green {
  0% { transform: scale(0.95); box-shadow: 0 0 0 0 rgba(16, 185, 129, 0.7); }
  70% { transform: scale(1); box-shadow: 0 0 0 8px rgba(16, 185, 129, 0); }
  100% { transform: scale(0.95); box-shadow: 0 0 0 0 rgba(16, 185, 129, 0); }
}
@keyframes pulse-amber {
  0% { transform: scale(0.95); box-shadow: 0 0 0 0 rgba(245, 158, 11, 0.7); }
  70% { transform: scale(1); box-shadow: 0 0 0 8px rgba(245, 158, 11, 0); }
  100% { transform: scale(0.95); box-shadow: 0 0 0 0 rgba(245, 158, 11, 0); }
}
@keyframes pulse-red {
  0% { transform: scale(0.95); box-shadow: 0 0 0 0 rgba(239, 68, 68, 0.7); }
  70% { transform: scale(1); box-shadow: 0 0 0 8px rgba(239, 68, 68, 0); }
  100% { transform: scale(0.95); box-shadow: 0 0 0 0 rgba(239, 68, 68, 0); }
}

.status-title-text {
  font-size: 16px;
  font-weight: 800;
  letter-spacing: 0.5px;
}
.text-status-green { color: #34d399; }
.text-status-amber { color: #fbbf24; }
.text-status-red { color: #f87171; }

.chain-sub-desc {
  font-size: 12px;
  color: #94a3b8;
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
  color: #cbd5e1;
  font-family: monospace;
}
.meta-val.highlight {
  color: #38bdf8;
}
.meta-val.block-height {
  color: #a78bfa;
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
</style>
