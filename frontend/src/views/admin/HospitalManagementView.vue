<template>
  <div class="page-container">
    <div class="page-header">
      <div>
        <h2 class="page-title">🏥 联盟医院与科室字典维护</h2>
        <p class="page-sub">管理多组织接入白名单、医院等级及下属临床科室字典</p>
      </div>
    </div>

    <el-row :gutter="20">
      <el-col :span="12">
        <el-card shadow="hover" class="box-card">
          <template #header>
            <div class="card-header">
              <span>联盟成员医疗机构 (Organizations)</span>
            </div>
          </template>
          <el-table :data="hospitals" stripe style="width: 100%">
            <el-table-column prop="hospital_no" label="机构代码" width="120" />
            <el-table-column prop="name" label="机构全称" />
            <el-table-column prop="level" label="等级" width="90" />
            <el-table-column prop="status" label="服务状态" width="100">
              <template #default>
                <el-tag size="small" type="success">接入运行</el-tag>
              </template>
            </el-table-column>
          </el-table>
        </el-card>
      </el-col>

      <el-col :span="12">
        <el-card shadow="hover" class="box-card">
          <template #header>
            <div class="card-header">
              <span>科室字典 (Departments)</span>
            </div>
          </template>
          <el-table :data="departments" stripe style="width: 100%">
            <el-table-column prop="dept_no" label="科室代码" width="120" />
            <el-table-column prop="name" label="科室名称" width="140" />
            <el-table-column prop="description" label="职责范围" />
          </el-table>
        </el-card>
      </el-col>
    </el-row>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import api from '../../api/client'

const hospitals = ref([])
const departments = ref([])

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
</style>
