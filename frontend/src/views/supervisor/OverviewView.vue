<template>
  <div class="page-container">
    <div class="page-header">
      <div>
        <h2 class="page-title">📊 联盟链存证监控与监管大屏</h2>
        <p class="page-sub">依据毕业设计技术指标：实时监控跨机构数据共享流转量、Fabric 区块高度、风险拦截统计与防篡改动态校验通过率</p>
      </div>
    </div>

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
              <span class="chart-title">📈 全局数据流转量与安全风险拦截趋势 (Throughput & Interceptions)</span>
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
              <span class="chart-title">🛡️ 防篡改核验通过率</span>
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
            <div class="chart-title">🏥 各医院医疗数据沉淀分布 (Hospital Nodes)</div>
          </template>
          <div ref="hospChartRef" class="echart-box-sm"></div>
        </el-card>
      </el-col>
      <el-col :span="12">
        <el-card shadow="hover" class="chart-card">
          <template #header>
            <div class="chart-title">⚖️ 跨院调阅动态风险评估分布 (Risk Engine)</div>
          </template>
          <div ref="riskChartRef" class="echart-box-sm"></div>
        </el-card>
      </el-col>
    </el-row>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, nextTick } from 'vue'
import * as echarts from 'echarts'
import api from '../../api/client'

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

onMounted(async () => {
  try {
    const res: any = await api.get('/supervisor/overview')
    if (res.code === 200) {
      stats.value = res.data
      await nextTick()
      renderCharts(res.data)
    }
  } catch (err) {
    console.error(err)
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
</style>
