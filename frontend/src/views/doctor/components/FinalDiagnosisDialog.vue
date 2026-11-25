<!--
  FinalDiagnosisDialog — 终诊归档与传染病哨兵工作台弹窗
  纯代码搬移：DoctorRecordsView.vue 视图三 (FINAL) 模板，行为 / 文案 / 类名保持不变。
-->
<template>
  <div class="inpage-workstation-wrapper">
    <div class="ws-page-header-card sticky-header">
      <div class="ws-header-left">
        <el-button :icon="ArrowLeft" plain size="small" @click="$emit('back')">返回列表</el-button>
        <div class="ws-header-title-box">
          <span class="ws-page-title">就诊确诊与区块链存证</span>
          <el-tag size="small" type="warning" effect="light">终审确诊</el-tag>
          <span v-if="record" class="ws-header-sub">
            单号: <strong class="mono">{{ record.record_no }}</strong> · 患者: <strong>{{ record.patient_name }}</strong>
          </span>
        </div>
      </div>
      <div class="ws-header-right">
        <InfectionSafetyAlert
          v-if="record?.infection_alert?.has_risk"
          :alert="record.infection_alert!"
          mode="badge"
        />
      </div>
    </div>

    <div v-if="record" class="final-workstation-content mt-3">
      <div v-if="record.infection_alert?.has_risk" class="mb-3">
        <InfectionSafetyAlert :alert="record.infection_alert!" mode="corner-banner" />
      </div>

      <div class="ws-body-grid">
        <!-- 左侧栏：初诊接诊回顾与医技检查报告回传 -->
        <div class="ws-col-left">
          <!-- 初诊与接诊回顾卡片 -->
          <div class="ws-card">
            <div class="ws-card-title">
              <el-icon><Document /></el-icon>
              <span>初诊挂号与临床接诊回顾</span>
            </div>
            <div class="summary-kv-list">
              <div class="kv-item"><span class="k">就诊单号：</span><span class="v mono font-bold">{{ record.record_no }}</span></div>
              <div class="kv-item"><span class="k">就诊类型：</span><span class="v"><el-tag size="small" type="warning">{{ formatEncounterType(record.encounter_type) }}</el-tag></span></div>
              <div class="kv-item"><span class="k">就诊科室：</span><span class="v">{{ record.department_name }}</span></div>
              <div class="kv-item"><span class="k">就诊患者：</span><span class="v font-bold">{{ record.patient_name }}</span></div>
              <div class="kv-item"><span class="k">生命体征：</span><span class="v text-emerald-600 font-semibold">{{ record.vital_signs || '平稳' }}</span></div>
              <div class="kv-item"><span class="k">主诉 (S)：</span><span class="v">{{ record.chief_complaint }}</span></div>
              <div class="kv-item"><span class="k">初诊拟诊：</span><span class="v text-primary">{{ record.initial_diagnosis }}</span></div>
              <div class="kv-item" v-if="record.diagnostic_basis"><span class="k">拟诊依据：</span><span class="v text-xs text-gray-500">{{ record.diagnostic_basis }}</span></div>
            </div>
          </div>

          <!-- 医技辅助检查回传报告卡片 -->
          <div class="ws-card">
            <div class="ws-card-title flex justify-between items-center">
              <div class="flex items-center gap-2">
                <el-icon><CircleCheckFilled color="#059669" /></el-icon>
                <span>医技科室回传检查报告汇总</span>
              </div>
              <el-tag v-if="record.exam_result" size="small" type="success">报告已就绪</el-tag>
              <el-tag v-else size="small" type="info">无医技单</el-tag>
            </div>
            <div class="lab-results-display">
              <div v-if="record.exam_result" class="lab-result-text">
                {{ record.exam_result }}
              </div>
              <div v-else class="text-xs text-gray-500 italic">
                暂无开具的医技辅助检查项目，直接进行临床确诊。
              </div>
            </div>
          </div>
        </div>

        <!-- 右侧栏：最终确诊、处方医嘱与规范 SOAP 存证 (flex: 1) -->
        <div class="ws-col-main">
          <div class="ws-card">
            <div class="ws-card-title">
              <el-icon><OfficeBuilding /></el-icon>
              <span>经治医生最终确诊与处置决策</span>
            </div>

            <el-form label-position="top">
              <el-row :gutter="16">
                <el-col :span="12">
                  <el-form-item label="最终确定诊断 (Final Diagnosis)" required>
                    <el-input
                      v-model="formState.diagnosis"
                      type="textarea"
                      :autosize="{ minRows: 2, maxRows: 4 }"
                      placeholder="例如：原发性高血压病1级，伴轻度劳力型心肌供血不足"
                    />
                  </el-form-item>
                </el-col>
                <el-col :span="12">
                  <el-form-item label="病因分析与病理机制 (Etiology)">
                    <el-input
                      v-model="formState.etiology"
                      type="textarea"
                      :autosize="{ minRows: 2, maxRows: 4 }"
                      placeholder="例如：血管内皮舒缩功能紊乱，高压负荷导致心肌耗氧量增加"
                    />
                  </el-form-item>
                </el-col>
              </el-row>

              <!-- 规范处方模板录入胶囊按钮组 -->
              <div class="rx-preset-capsules mb-2">
                <span class="rx-preset-title">规范处方模板录入：</span>
                <el-button
                  v-for="p in presets"
                  :key="p.key"
                  size="small"
                  :type="p.type"
                  plain
                  round
                  class="rx-capsule-btn"
                  @click="$emit('apply-prescription-preset', p, 'FINAL')"
                >
                  {{ p.label }}
                </el-button>
              </div>

              <el-form-item label="综合处置与处方医嘱 (分区分条下达 · 支持药品关键词下拉提示)" required>
                <slot name="final-treatment">
                  <TreatmentPlanEditor
                    v-model="formState.treatment_plan"
                    @change="$emit('generate-soap-record')"
                  />
                </slot>
              </el-form-item>

              <!-- 规则辅助生成规范 SOAP -->
              <div class="soap-action-bar mb-3">
                <el-button
                  type="warning"
                  size="default"
                  @click="$emit('generate-soap-record')"
                >
                  基于临床规则辅助生成规范 SOAP 病历
                </el-button>
                <span class="text-xs text-slate-500">（自动整合主观S、客观O、评估A、计划P四维规范）</span>
              </div>

              <!-- 法律合规免责声明 -->
              <el-alert
                title="法律与医疗合规声明：临床规则辅助引擎生成内容仅供结构化整理参考，最终诊疗结论、处方权与法律责任以接诊责任医生确认为准。"
                type="info"
                show-icon
                :closable="false"
                class="mb-3"
              />

              <el-form-item label="最终结构化规范 SOAP 病历全文确认 (将进行 AES-GCM 加密与 Fabric 存证)">
                <el-input
                  v-model="formState.soap_content"
                  type="textarea"
                  :rows="7"
                  class="soap-textarea"
                />
              </el-form-item>
            </el-form>
          </div>
        </div>
      </div>

      <!-- 底部提交操作栏 -->
      <div class="ws-page-footer-card mt-3">
        <div class="ws-footer-bar">
          <el-button @click="$emit('back')">返回就诊列表</el-button>
          <el-button
            type="success"
            size="large"
            :loading="submittingFinal"
            :disabled="!formState.diagnosis || !formState.treatment_plan"
            @click="$emit('submit-final')"
          >
            确认并完成归档存证 (AES加密 + IPFS + Fabric区块链存证)
          </el-button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ArrowLeft, OfficeBuilding, Document, CircleCheckFilled } from '@element-plus/icons-vue'
import TreatmentPlanEditor from '@/components/TreatmentPlanEditor.vue'
import InfectionSafetyAlert from '@/components/InfectionSafetyAlert.vue'
import { formatEncounterType } from '@/utils/recordFormatters'
import type { MedicalRecordRow, PrescriptionPreset } from '@/types/clinical'

interface Props {
  record: MedicalRecordRow | null
  formState: { diagnosis: string; etiology: string; treatment_plan: string; soap_content: string }
  submittingFinal: boolean
  presets: PrescriptionPreset[]
}

defineProps<Props>()

defineEmits<{
  back: []
  'apply-prescription-preset': [preset: PrescriptionPreset, target: 'FINAL']
  'generate-soap-record': []
  'submit-final': []
}>()
</script>

<style scoped src="../DoctorRecordsView.css"></style>