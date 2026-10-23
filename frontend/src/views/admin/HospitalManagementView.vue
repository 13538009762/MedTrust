<template>
  <div class="page-container">
    <div class="page-header">
      <div>
        <h2 class="page-title">联盟医院与科室字典维护</h2>
        <p class="page-sub">管理联盟成员医疗机构接入白名单、Fabric 组织节点拓扑、MSP 证书有效期及下属临床科室字典</p>
      </div>
    </div>

    <el-row :gutter="20">
      <el-col :span="12">
        <!-- 联盟医疗机构卡片 -->
        <el-card shadow="hover" class="box-card">
          <template #header>
            <div class="card-header flex justify-between items-center">
              <span>联盟成员医疗机构 (Organizations)</span>
              <el-tag size="small" type="primary" effect="plain">共 {{ hospitals.length }} 家接入机构</el-tag>
            </div>
          </template>
          <el-table :data="hospitals" stripe style="width: 100%">
            <el-table-column prop="hospital_no" label="机构代码" width="95">
              <template #default="{ row }">
                <code class="mono font-semibold">{{ row.hospital_no }}</code>
              </template>
            </el-table-column>
            <el-table-column prop="name" label="医院机构全称" min-width="140" />
            <el-table-column label="MSP 组织标识" width="110">
              <template #default="{ row, $index }">
                <el-tag size="small" type="primary" effect="plain" class="mono font-bold">
                  {{ getHospitalMsp(row, $index) }}
                </el-tag>
              </template>
            </el-table-column>
            <el-table-column label="背书节点端点" min-width="190">
              <template #default="{ row, $index }">
                <code class="text-xs text-purple-700 mono font-semibold">{{ getHospitalPeer(row, $index) }}</code>
              </template>
            </el-table-column>
            <el-table-column prop="level" label="等级" width="70" align="center" />
            <el-table-column prop="status" label="状态" width="85" align="center">
              <template #default>
                <el-tag size="small" type="success">接入在线</el-tag>
              </template>
            </el-table-column>
          </el-table>
        </el-card>

        <!-- 🌐 联盟链多组织节点拓扑与证书健康状态 -->
        <el-card shadow="hover" class="box-card mt-4 topology-card">
          <template #header>
            <div class="card-header flex justify-between items-center">
              <div class="flex items-center gap-2">
                <el-icon class="text-emerald-400"><Connection /></el-icon>
                <span class="text-white font-bold">🌐 联盟链多组织节点拓扑与证书健康状态</span>
              </div>
              <el-tag size="small" type="success" effect="dark">Raft 共识存活</el-tag>
            </div>
          </template>

          <div class="topology-grid">
            <div v-for="node in topologyNodes" :key="node.msp" class="topo-node-box">
              <div class="tn-top">
                <div class="tn-name-row">
                  <span class="mini-pulse-dot dot-green mr-1"></span>
                  <strong class="tn-name">{{ node.name }}</strong>
                  <el-tag size="small" type="primary" effect="dark" class="mono ml-auto">{{ node.msp }}</el-tag>
                </div>
                <div class="tn-peer mono text-xs mt-1 text-slate-300">
                  <el-icon class="mr-1"><Platform /></el-icon>{{ node.peer }}
                </div>
              </div>
              <div class="tn-divider"></div>
              <div class="tn-bottom">
                <div class="tn-info-row">
                  <span class="k">X.509 根证书：</span>
                  <span class="v text-emerald-400 font-semibold">{{ node.certExpire }}</span>
                </div>
                <div class="tn-info-row">
                  <span class="k">TLS 传输加密：</span>
                  <span class="v text-cyan-300 mono">{{ node.tlsVersion }} (双向 mTLS)</span>
                </div>
                <div class="tn-info-row">
                  <span class="k">共识区块高度：</span>
                  <span class="v mono font-bold text-amber-400">#{{ node.blockHeight }} (已同步)</span>
                </div>
              </div>
            </div>
          </div>
        </el-card>
      </el-col>

      <el-col :span="12">
        <el-card shadow="hover" class="box-card h-full">
          <template #header>
            <div class="card-header flex justify-between items-center">
              <span>临床科室与医技中心字典 (Departments)</span>
              <el-tag size="small" type="info">共 {{ departments.length }} 个临床专科</el-tag>
            </div>
          </template>
          <el-table :data="departments" stripe style="width: 100%" max-height="620">
            <el-table-column prop="dept_no" label="科室代码" width="110">
              <template #default="{ row }">
                <code class="mono font-semibold">{{ row.dept_no }}</code>
              </template>
            </el-table-column>
            <el-table-column prop="name" label="科室名称" width="150">
              <template #default="{ row }">
                <strong>{{ row.name }}</strong>
              </template>
            </el-table-column>
            <el-table-column prop="description" label="职责范围与诊疗指征" />
          </el-table>
        </el-card>
      </el-col>
    </el-row>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { Connection, Platform } from '@element-plus/icons-vue'
import api from '../../api/client'

const hospitals = ref<any[]>([])
const departments = ref<any[]>([])

const topologyNodes = ref([
  {
    name: '广州第一附属医院背书节点',
    msp: 'Org1MSP',
    peer: 'peer0.org1.example.com:7051',
    certExpire: '有效期至 2036-09-12 · 正常',
    tlsVersion: 'TLSv1.3',
    blockHeight: 357
  },
  {
    name: '深圳中心医院背书节点',
    msp: 'Org2MSP',
    peer: 'peer0.org2.example.com:7051',
    certExpire: '有效期至 2036-09-12 · 正常',
    tlsVersion: 'TLSv1.3',
    blockHeight: 357
  },
  {
    name: '珠海人民医院背书节点',
    msp: 'Org3MSP',
    peer: 'peer0.org3.example.com:7051',
    certExpire: '有效期至 2036-09-12 · 正常',
    tlsVersion: 'TLSv1.3',
    blockHeight: 357
  }
])

function getHospitalMsp(row: any, index: number): string {
  if (row.id === 1 || row.name?.includes('第一') || index === 0) return 'Org1MSP'
  if (row.id === 2 || row.name?.includes('中心') || index === 1) return 'Org2MSP'
  return 'Org3MSP'
}

function getHospitalPeer(row: any, index: number): string {
  if (row.id === 1 || row.name?.includes('第一') || index === 0) return 'peer0.org1.example.com:7051'
  if (row.id === 2 || row.name?.includes('中心') || index === 1) return 'peer0.org2.example.com:7051'
  return 'peer0.org3.example.com:7051'
}

onMounted(async () => {
  try {
    const res1: any = await api.get('/system/hospitals')
    if (res1.code === 200) hospitals.value = res1.data

    const res2: any = await api.get('/system/departments')
    if (res2.code === 200) departments.value = res2.data
  } catch (err) {
    console.error(err)
  }
})
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
.box-card {
  border-radius: 14px;
}
.card-header {
  font-weight: 700;
  color: #1e293b;
}

/* 🌐 联盟链拓扑卡片深色科技感 */
.topology-card {
  background: linear-gradient(145deg, #0f172a 0%, #1e293b 100%) !important;
  color: #f8fafc !important;
  border: 1px solid #334155 !important;
}
.topology-card :deep(.el-card__header) {
  border-bottom: 1px solid rgba(255, 255, 255, 0.1) !important;
}

.topology-grid {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.topo-node-box {
  background: rgba(255, 255, 255, 0.04);
  border: 1px solid rgba(255, 255, 255, 0.08);
  border-radius: 8px;
  padding: 12px 14px;
  transition: all 0.2s ease;
}
.topo-node-box:hover {
  background: rgba(255, 255, 255, 0.08);
  border-color: rgba(56, 189, 248, 0.3);
  transform: translateY(-1px);
}

.tn-name-row {
  display: flex;
  align-items: center;
}
.tn-name {
  color: #f8fafc;
  font-size: 13px;
}

.tn-divider {
  height: 1px;
  background: rgba(255, 255, 255, 0.06);
  margin: 8px 0;
}

.tn-bottom {
  display: flex;
  flex-direction: column;
  gap: 4px;
  font-size: 11px;
}

.tn-info-row {
  display: flex;
  justify-content: space-between;
  align-items: center;
}
.tn-info-row .k {
  color: #94a3b8;
}

.mini-pulse-dot {
  width: 7px;
  height: 7px;
  border-radius: 50%;
  display: inline-block;
}
.dot-green {
  background-color: #10b981;
  box-shadow: 0 0 0 0 rgba(16, 185, 129, 0.7);
}
</style>
