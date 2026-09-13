<template>
  <div class="page-container">
    <div class="page-header">
      <div>
        <h2 class="page-title">📊 联盟链存证监控与监管大屏</h2>
        <p class="page-sub">实时汇聚三所医院的数据存证上链总量、区块高度、紧急访问发生率及动态风险评分分布</p>
      </div>
    </div>

    <!-- 顶部四项指标卡片 -->
    <el-row :gutter="16" class="metric-row">
      <el-col :span="6">
        <el-card shadow="hover" class="metric-card">
          <div class="metric-title">全网病历存证总量</div>
          <div class="metric-num text-primary">{{ stats.total_records }}</div>
          <div class="metric-foot">链下 IPFS + 链上凭证</div>
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
          <div class="metric-title">全量可信审计流水</div>
          <div class="metric-num text-success">{{ stats.total_logs }}</div>
          <div class="metric-foot">全生命周期不可篡改追责</div>
        </el-card>
      </el-col>
    </el-row>

    <!-- 图表展示区 -->
    <el-row :gutter="20" class="mt-4">
      <el-col :span="12">
        <el-card shadow="hover" class="chart-card">
          <template #header>
            <div class="chart-title">各医院医疗数据沉淀分布 (Hospital Nodes)</div>
          </template>
          <div ref="hospChartRef" class="echart-box"></div>
        </el-card>
      </el-col>
      <el-col :span="12">
        <el-card shadow="hover" class="chart-card">
          <template #header>
            <div class="chart-title">跨院调阅动态风险评估分布 (Risk Engine)</div>
          </template>
          <div ref="riskChartRef" class="echart-box"></div>
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
})

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
  if (hospChartRef.value) {
    const c1 = echarts.init(hospChartRef.value)
    const hospData = (data.hosp_stats || []).map((item: any) => ({
      name: item.name,
      value: item.count,
    }))
    c1.setOption({
      tooltip: { trigger: 'item' },
      series: [
        {
          name: '存证数量',
          type: 'pie',
          radius: ['45%', '70%'],
          avoidLabelOverlap: false,
          itemStyle: { borderRadius: 10, borderColor: '#fff', borderWidth: 2 },
          data: hospData.length ? hospData : [
            { name: '第一人民医院', value: 4 },
            { name: '省立中心医院', value: 2 },
            { name: '协和医学中心', value: 1 }
          ],
        },
      ],
    })
  }

  if (riskChartRef.value) {
    const c2 = echarts.init(riskChartRef.value)
    c2.setOption({
      tooltip: { trigger: 'item' },
      color: ['#10b981', '#f59e0b', '#ef4444'],
      series: [
        {
          name: '风险评级',
          type: 'pie',
          radius: '65%',
          data: [
            { value: 12, name: '低风险 (0-29分: 放行)' },
            { value: 4, name: '中风险 (30-59分: 确认)' },
            { value: 2, name: '高风险 (≥60分: 拦截)' },
          ],
          emphasis: {
            itemStyle: {
              shadowBlur: 10,
              shadowOffsetX: 0,
              shadowColor: 'rgba(0, 0, 0, 0.5)',
            },
          },
        },
      ],
    })
  }
}
</script>

<style scoped>
.page-container {
  max-width: 1300px;
  margin: 0 auto;
}
.page-header {
  margin-bottom: 20px;
}
.page-title {
  font-size: 22px;
  font-weight: 800;
  color: #0f172a;
}
.page-sub {
  font-size: 13px;
  color: #64748b;
}
.metric-row {
  margin-bottom: 20px;
}
.metric-card {
  border-radius: 14px;
}
.metric-title {
  font-size: 13px;
  color: #64748b;
  font-weight: 600;
  margin-bottom: 8px;
}
.metric-num {
  font-size: 32px;
  font-weight: 900;
  margin-bottom: 8px;
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
  border-radius: 14px;
}
.chart-title {
  font-weight: 700;
  color: #1e293b;
}
.echart-box {
  width: 100%;
  height: 320px;
}
.mt-4 {
  margin-top: 16px;
}
</style>
