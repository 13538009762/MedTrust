<template>
  <!-- 医技检查单智能分流派发面板 (WP-16: 从 DoctorRecordsView CREATE 路径 B 抽取)
       包含 LIS/PACS 分流说明与检查项目勾选卡 (勾选后自动生成 ORD 申请单移交医技中心)。 -->
  <div class="exam-order-dispatch-panel">
    <div class="dispatch-target-tip mb-3 mt-2">
      <el-alert
        title="医技检查单分流派发机制："
        type="info"
        :closable="false"
        description="① 血常规 / 心肌酶谱 / 肝肾生化 -> 自动派发至【临床检验医学中心·LIS任务池】(由检验科技师 tech_lab 接单)；
② 12导联心电图 / 胸部CT / 头颅MRI -> 自动派发至【放射影像与心电中心·PACS任务池】(由影像医师 tech_pacs 接单)。
提交后医技人员即可在对应工作台接单处理，当前医生亦可通过左侧「医技检查中心」协同出报告。"
        show-icon
      />
    </div>

    <el-form-item label="开具辅助检查项目 (勾选后自动生成独立申请单号 ORD... 移交医技中心)">
      <div class="exam-picker-grid">
        <el-checkbox-group :model-value="selectedExamList" class="exam-checkbox-container" @update:model-value="$emit('update:selectedExamList', $event)">
          <div class="exam-select-card" :class="{ selected: selectedExamList.includes('12导联心电图 (ECG)') }">
            <el-checkbox label="12导联心电图 (ECG)">
              <span class="exam-label-text">12导联心电图 (ECG)</span>
            </el-checkbox>
            <el-tag size="small" type="warning" effect="plain">影像心电中心</el-tag>
          </div>
          <div class="exam-select-card" :class="{ selected: selectedExamList.includes('常规全血细胞分析 (血常规)') }">
            <el-checkbox label="常规全血细胞分析 (血常规)">
              <span class="exam-label-text">常规全血细胞分析 (血常规)</span>
            </el-checkbox>
            <el-tag size="small" type="primary" effect="plain">临床检验科</el-tag>
          </div>
          <div class="exam-select-card" :class="{ selected: selectedExamList.includes('血清心肌酶谱与肌钙蛋白') }">
            <el-checkbox label="血清心肌酶谱与肌钙蛋白">
              <span class="exam-label-text">血清心肌酶谱与肌钙蛋白</span>
            </el-checkbox>
            <el-tag size="small" type="primary" effect="plain">临床检验科</el-tag>
          </div>
          <div class="exam-select-card" :class="{ selected: selectedExamList.includes('胸部高分辨率 CT 平扫') }">
            <el-checkbox label="胸部高分辨率 CT 平扫">
              <span class="exam-label-text">胸部高分辨率 CT 平扫</span>
            </el-checkbox>
            <el-tag size="small" type="warning" effect="plain">放射影像中心</el-tag>
          </div>
          <div class="exam-select-card" :class="{ selected: selectedExamList.includes('肝肾功能与电解质生化检查') }">
            <el-checkbox label="肝肾功能与电解质生化检查">
              <span class="exam-label-text">肝肾功能与电解质生化</span>
            </el-checkbox>
            <el-tag size="small" type="primary" effect="plain">临床检验科</el-tag>
          </div>
          <div class="exam-select-card" :class="{ selected: selectedExamList.includes('头颅 MRI 平扫') }">
            <el-checkbox label="头颅 MRI 平扫">
              <span class="exam-label-text">头颅 MRI 平扫</span>
            </el-checkbox>
            <el-tag size="small" type="warning" effect="plain">放射影像中心</el-tag>
          </div>
        </el-checkbox-group>
      </div>
    </el-form-item>
  </div>
</template>

<script setup lang="ts">
defineProps<{
  selectedExamList: string[]
}>()

defineEmits<{
  (e: 'update:selectedExamList', val: string[]): void
}>()
</script>
<style scoped src="../CrossQueryView.css"></style>

