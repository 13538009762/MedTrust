<template>
  <div v-if="alert && alert.has_risk" class="infection-safety-alert" :class="[alert.risk_level.toLowerCase(), mode]">
    <!-- 右上角悬浮条幅模式 (Corner Banner) -->
    <div v-if="mode === 'corner-banner'" class="corner-banner-wrapper">
      <div class="banner-pulse-dot" />
      <div class="banner-content" @click="openGuideModal">
        <span class="banner-tag">
          <el-icon class="alert-icon"><WarningFilled /></el-icon>
          医护职业安全预警
        </span>
        <span class="banner-summary">
          检测到患者有<strong>【{{ diseaseNames }}】</strong>必定/长期携带病史！(为保证医护人员安全)
        </span>
        <el-button type="danger" size="small" round class="guide-btn" @click.stop="openGuideModal">
          查看专项防护规程
        </el-button>
      </div>
    </div>

    <!-- 右上角微标模式 (Corner Badge) -->
    <div v-else-if="mode === 'badge'" class="corner-badge-wrapper" @click="openGuideModal">
      <div class="badge-pill" :class="alert.risk_level.toLowerCase()">
        <span class="pulse-ring" />
        <el-icon class="mr-1"><WarningFilled /></el-icon>
        <span>医护安全预警: {{ diseaseNames }} (必定携带)</span>
        <span class="badge-action-hint">点击查看防护</span>
      </div>
    </div>

    <!-- 行内卡片模式 (Card) -->
    <div v-else class="card-alert-wrapper" :class="alert.risk_level.toLowerCase()">
      <div class="card-top">
        <div class="c-title">
          <el-icon class="text-xl mr-1 text-red-500"><WarningFilled /></el-icon>
          <span class="font-bold text-red-700">【医护职业安全专项警示 · 重点防护】</span>
          <el-tag :type="alert.risk_level === 'CRITICAL' ? 'danger' : 'warning'" size="small" effect="dark" class="ml-2">
            {{ alert.risk_level === 'CRITICAL' ? '极高危生物暴露风险' : '高危传染病携带' }}
          </el-tag>
        </div>
        <el-button type="danger" link size="small" @click="openGuideModal">
          查看国家卫健委规范防护指引 &gt;
        </el-button>
      </div>
      <div class="card-desc">
        {{ alert.summary }}
      </div>
      <div class="gear-chips mt-2">
        <span class="gear-lbl">接诊必须穿戴：</span>
        <el-tag
          v-for="g in (alert.protection_gear || []).slice(0, 4)"
          :key="g"
          size="small"
          type="info"
          effect="plain"
          class="mr-1"
        >
          🛡️ {{ g }}
        </el-tag>
      </div>
    </div>

    <!-- 弹窗：国家卫健委规范 · 医务人员职业安全与暴露防护规程 -->
    <el-dialog
      v-model="guideModalVisible"
      title="国家卫健委《医务人员职业暴露防护导则》· 专项安全处置指南"
      width="780px"
      append-to-body
      class="infection-guide-dialog"
    >
      <div class="guide-modal-body">
        <!-- 警报横幅 -->
        <div class="guide-banner" :class="alert.risk_level.toLowerCase()">
          <div class="gb-head">
            <el-icon class="gb-icon"><WarningFilled /></el-icon>
            <div class="gb-title">
              <span class="main-t">【{{ alert.risk_level === 'CRITICAL' ? '极高危职业暴露预警' : '高危传染性疾病预警' }}】</span>
              <span class="sub-t">严格落实标准预防、屏障防护与锐器伤防范（为保证医务人员人身安全）</span>
            </div>
          </div>
          <div class="gb-text mt-2">
            {{ alert.summary }}
          </div>
        </div>

        <!-- 检出疾病特征清单 -->
        <div class="section-title mt-4">
          <span class="st-bar" />
          <span class="st-text">检出高危病原体与临床风险特征</span>
        </div>
        <div class="disease-cards-grid">
          <div
            v-for="(d, idx) in alert.diseases"
            :key="idx"
            class="d-card"
          >
            <div class="d-header">
              <span class="d-name">{{ d.name }}</span>
              <el-tag size="small" :type="d.risk_level === 'CRITICAL' ? 'danger' : 'warning'">
                {{ d.persistence }}
              </el-tag>
            </div>
            <div class="d-row">
              <span class="lbl">传播分类：</span>
              <span>{{ d.category }}</span>
            </div>
            <div class="d-row">
              <span class="lbl">传播途径：</span>
              <span class="text-danger font-semibold">{{ d.transmission }}</span>
            </div>
            <div class="d-row">
              <span class="lbl">医护威胁：</span>
              <span class="text-danger">{{ d.threat_to_doctor }}</span>
            </div>
            <div v-if="d.record_no" class="d-row trace">
              <span class="lbl">检出溯源：</span>
              <span class="mono">{{ d.record_no }} ({{ d.hospital_name || '医疗机构' }} · {{ d.record_date }})</span>
            </div>
          </div>
        </div>

        <!-- 医护必备 PPE 装备清单 -->
        <div class="section-title mt-4">
          <span class="st-bar" />
          <span class="st-text">接诊、穿刺及查体必须穿戴之个人防护装备 (PPE)</span>
        </div>
        <div class="gear-box">
          <div
            v-for="(gear, idx) in alert.protection_gear"
            :key="idx"
            class="gear-item"
          >
            <div class="g-icon">🛡️</div>
            <div class="g-text">{{ gear }}</div>
          </div>
        </div>

        <!-- 临床防护操作细则 -->
        <div class="section-title mt-4">
          <span class="st-bar" />
          <span class="st-text">医护人员接诊与操作防护规范细则</span>
        </div>
        <ul class="precaution-list">
          <li v-for="(p, idx) in alert.precautions" :key="idx">
            <span class="num">{{ idx + 1 }}</span>
            <span class="content">{{ p }}</span>
          </li>
        </ul>

        <!-- 职业暴露紧急处置流程 (一挤二冲三消毒) -->
        <div class="section-title mt-4">
          <span class="st-bar danger" />
          <span class="st-text text-danger font-bold">意外针刺伤或破损黏膜暴露紧急处置规程 (一挤二冲三消毒)</span>
        </div>
        <div class="emergency-flow-grid">
          <div
            v-for="(step, idx) in (alert.emergency_steps || defaultEmergencySteps)"
            :key="idx"
            class="flow-step-card"
          >
            <div class="step-badge">STEP {{ idx + 1 }}</div>
            <div class="step-desc">{{ step }}</div>
          </div>
        </div>

        <!-- 检出溯源病历信息 -->
        <div v-if="alert.detected_from && alert.detected_from.length > 0" class="trace-section mt-4">
          <div class="trace-title">🔍 跨机构档案与检验检出原始凭证：</div>
          <div v-for="(src, idx) in alert.detected_from" :key="idx" class="trace-item mono text-xs">
            • {{ src }}
          </div>
        </div>
      </div>

      <template #footer>
        <div class="guide-modal-footer">
          <span class="footer-tip text-gray-500 text-xs">
            本预警基于 MedTrust 全网跨机构电子健康档案与国家卫健委规范实时计算，数据已上链存证。
          </span>
          <el-button type="primary" @click="guideModalVisible = false">
            我已熟知防护指引并穿戴完备防护装备
          </el-button>
        </div>
      </template>
    </el-dialog>
  </div>

  <!-- 无高危传染病时的安全状态展示 -->
  <div v-else-if="showSafeStatus" class="infection-safe-badge">
    <el-tag type="success" effect="light" size="small" class="safe-pill">
      <el-icon class="mr-1"><CircleCheckFilled /></el-icon>
      医护安全核验通过: 未检出高危传染病携带
    </el-tag>
  </div>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'
import { WarningFilled, CircleCheckFilled } from '@element-plus/icons-vue'
import type { InfectionRiskAlert } from '../utils/infectionGuard'

const props = withDefaults(
  defineProps<{
    alert?: InfectionRiskAlert | null
    mode?: 'corner-banner' | 'badge' | 'card'
    showSafeStatus?: boolean
  }>(),
  {
    mode: 'corner-banner',
    showSafeStatus: false
  }
)

const guideModalVisible = ref(false)

const diseaseNames = computed(() => {
  if (!props.alert?.diseases || props.alert.diseases.length === 0) return '高危传染病'
  return props.alert.diseases.map(d => {
    const mainName = d.name.split(' (')[0]
    return mainName.replace('获得性免疫缺陷综合征 / ', '')
  }).join('、')
})

const defaultEmergencySteps = [
  '挤血：从近心端向远心端轻轻挤压伤口，尽量挤出损伤处血液，严禁局部直接按压挤捏',
  '冲洗：立即使用流动的生理盐水或流动肥皂清水反复彻底冲洗伤口至少 5 分钟',
  '消毒：用 75% 医用乙醇或 0.5% 聚维酮碘消毒伤口局部，并包扎保护',
  '上报与用药：立即向院感科紧急报备，并在暴露后 2 小时内尽早服用 PEP 阻断药物'
]

function openGuideModal() {
  guideModalVisible.value = true
}

defineExpose({
  openGuideModal
})
</script>

<style scoped>
.infection-safety-alert {
  position: relative;
  z-index: 10;
  transition: all 0.3s ease;
}

/* 右上角悬浮条幅模式 */
.corner-banner-wrapper {
  display: flex;
  align-items: center;
  gap: 10px;
  background: linear-gradient(135deg, #fef2f2 0%, #fee2e2 50%, #fecaca 100%);
  border: 1.5px solid #ef4444;
  box-shadow: 0 4px 14px rgba(239, 68, 68, 0.22);
  border-radius: 8px;
  padding: 8px 14px;
  animation: pulse-glow-border 2.5s infinite ease-in-out;
  max-width: 100%;
  box-sizing: border-box;
}

.banner-pulse-dot {
  width: 10px;
  height: 10px;
  border-radius: 50%;
  background-color: #dc2626;
  box-shadow: 0 0 0 0 rgba(220, 38, 38, 0.7);
  animation: pulse-dot 1.6s infinite;
  flex-shrink: 0;
}

.banner-content {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
  cursor: pointer;
  flex: 1;
}

.banner-tag {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  background-color: #dc2626;
  color: #fff;
  font-size: 12px;
  font-weight: 700;
  padding: 2px 8px;
  border-radius: 4px;
  letter-spacing: 0.5px;
}

.banner-summary {
  font-size: 13px;
  color: #991b1b;
  font-weight: 500;
}

.banner-summary strong {
  color: #b91c1c;
  font-weight: 700;
  text-decoration: underline;
}

.guide-btn {
  font-weight: 600;
  box-shadow: 0 2px 6px rgba(220, 38, 38, 0.3);
  transition: transform 0.2s;
}

.guide-btn:hover {
  transform: translateY(-1px);
}

/* 右上角药丸微标模式 */
.corner-badge-wrapper {
  display: inline-flex;
  align-items: center;
  cursor: pointer;
  white-space: nowrap !important;
  flex-shrink: 0 !important;
}

.badge-pill {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 5px 14px;
  border-radius: 9999px;
  font-size: 12px;
  font-weight: 600;
  border: 1.5px solid transparent;
  transition: all 0.25s ease;
  position: relative;
  white-space: nowrap !important;
  flex-shrink: 0 !important;
  user-select: none;
  line-height: 1.4;
}

.badge-pill:hover {
  transform: translateY(-1px);
  filter: brightness(1.02);
}

.badge-pill.critical {
  background: linear-gradient(135deg, #fee2e2 0%, #fecaca 100%);
  color: #b91c1c;
  border-color: #ef4444;
  box-shadow: 0 2px 10px rgba(239, 68, 68, 0.25);
}

.badge-pill.high {
  background: linear-gradient(135deg, #ffedd5 0%, #fed7aa 100%);
  color: #c2410c;
  border-color: #f97316;
  box-shadow: 0 2px 10px rgba(249, 115, 22, 0.22);
}

.badge-pill.medium_high {
  background: linear-gradient(135deg, #fef9c3 0%, #fef08a 100%);
  color: #854d0e;
  border-color: #eab308;
  box-shadow: 0 2px 8px rgba(234, 179, 8, 0.2);
}

.badge-action-hint {
  margin-left: 6px;
  font-size: 11px;
  padding: 2px 8px;
  background: rgba(255, 255, 255, 0.85);
  border-radius: 10px;
  font-weight: 600;
  box-shadow: 0 1px 2px rgba(0, 0, 0, 0.05);
}

.pulse-ring {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background-color: currentColor;
  margin-right: 2px;
  display: inline-block;
  animation: pulse-ring-anim 1.8s infinite ease-in-out;
  flex-shrink: 0;
}

@keyframes pulse-ring-anim {
  0%, 100% {
    opacity: 1;
    transform: scale(1);
  }
  50% {
    opacity: 0.4;
    transform: scale(0.75);
  }
}

/* 行内卡片模式 */
.card-alert-wrapper {
  padding: 12px 16px;
  border-radius: 8px;
  border: 1.5px solid;
  margin-bottom: 12px;
}

.card-alert-wrapper.critical {
  background: #fef2f2;
  border-color: #f87171;
}

.card-alert-wrapper.high {
  background: #fff7ed;
  border-color: #fdba74;
}

.card-alert-wrapper.medium_high {
  background: #fefce8;
  border-color: #fde047;
}

.card-top {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.card-desc {
  font-size: 13px;
  color: #374151;
  margin-top: 6px;
  line-height: 1.5;
}

.gear-chips {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 4px;
}

.gear-lbl {
  font-size: 12px;
  color: #4b5563;
  font-weight: 600;
}

/* 安全状态微标 */
.safe-pill {
  display: inline-flex;
  align-items: center;
  font-weight: 500;
}

/* 弹窗样式 */
.guide-modal-body {
  max-height: 65vh;
  overflow-y: auto;
  padding-right: 6px;
}

.guide-banner {
  padding: 14px 18px;
  border-radius: 8px;
  border: 1.5px solid;
}

.guide-banner.critical {
  background: linear-gradient(135deg, #fef2f2, #fee2e2);
  border-color: #ef4444;
}

.guide-banner.high {
  background: linear-gradient(135deg, #fff7ed, #ffedd5);
  border-color: #f97316;
}

.gb-head {
  display: flex;
  align-items: flex-start;
  gap: 10px;
}

.gb-icon {
  font-size: 26px;
  color: #dc2626;
  margin-top: 2px;
}

.gb-title .main-t {
  font-size: 16px;
  font-weight: 800;
  color: #b91c1c;
  display: block;
}

.gb-title .sub-t {
  font-size: 13px;
  color: #7f1d1d;
  display: block;
  margin-top: 2px;
}

.gb-text {
  font-size: 13px;
  color: #450a0a;
  background: rgba(255, 255, 255, 0.75);
  padding: 8px 12px;
  border-radius: 6px;
  font-weight: 500;
}

.section-title {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 8px;
}

.st-bar {
  width: 4px;
  height: 16px;
  background: #3b82f6;
  border-radius: 2px;
}

.st-bar.danger {
  background: #dc2626;
}

.st-text {
  font-size: 14px;
  font-weight: 700;
  color: #1f2937;
}

.disease-cards-grid {
  display: grid;
  grid-template-columns: 1fr;
  gap: 10px;
}

.d-card {
  background: #f8fafc;
  border: 1px solid #e2e8f0;
  border-radius: 6px;
  padding: 10px 14px;
}

.d-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 6px;
}

.d-name {
  font-size: 14px;
  font-weight: 700;
  color: #0f172a;
}

.d-row {
  font-size: 12px;
  color: #475569;
  line-height: 1.6;
}

.d-row .lbl {
  font-weight: 600;
  color: #334155;
}

.d-row.trace {
  margin-top: 4px;
  padding-top: 4px;
  border-top: 1px dashed #cbd5e1;
  color: #64748b;
}

.gear-box {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(220px, 1fr));
  gap: 8px;
}

.gear-item {
  display: flex;
  align-items: center;
  gap: 8px;
  background: #eff6ff;
  border: 1px solid #bfdbfe;
  padding: 8px 12px;
  border-radius: 6px;
}

.g-icon {
  font-size: 16px;
}

.g-text {
  font-size: 12px;
  font-weight: 600;
  color: #1e40af;
}

.precaution-list {
  list-style: none;
  padding: 0;
  margin: 0;
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.precaution-list li {
  display: flex;
  align-items: flex-start;
  gap: 8px;
  font-size: 13px;
  color: #374151;
  background: #f9fafb;
  padding: 6px 10px;
  border-radius: 6px;
  border-left: 3px solid #6b7280;
}

.precaution-list .num {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 18px;
  height: 18px;
  background: #e5e7eb;
  border-radius: 50%;
  font-size: 11px;
  font-weight: 700;
  flex-shrink: 0;
  color: #4b5563;
}

.emergency-flow-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(170px, 1fr));
  gap: 8px;
}

.flow-step-card {
  background: #fef2f2;
  border: 1px solid #fecaca;
  border-radius: 6px;
  padding: 10px;
}

.step-badge {
  display: inline-block;
  font-size: 11px;
  font-weight: 800;
  color: #dc2626;
  background: #fee2e2;
  padding: 1px 6px;
  border-radius: 4px;
  margin-bottom: 4px;
}

.step-desc {
  font-size: 12px;
  color: #991b1b;
  line-height: 1.4;
  font-weight: 500;
}

.trace-section {
  background: #f1f5f9;
  border-radius: 6px;
  padding: 8px 12px;
}

.trace-title {
  font-size: 12px;
  font-weight: 600;
  color: #475569;
  margin-bottom: 4px;
}

.trace-item {
  color: #64748b;
  line-height: 1.5;
}

.guide-modal-footer {
  display: flex;
  justify-content: space-between;
  align-items: center;
  width: 100%;
}

@keyframes pulse-dot {
  0% {
    box-shadow: 0 0 0 0 rgba(220, 38, 38, 0.7);
  }
  70% {
    box-shadow: 0 0 0 8px rgba(220, 38, 38, 0);
  }
  100% {
    box-shadow: 0 0 0 0 rgba(220, 38, 38, 0);
  }
}

@keyframes pulse-glow-border {
  0%, 100% {
    box-shadow: 0 4px 14px rgba(239, 68, 68, 0.22);
    border-color: #ef4444;
  }
  50% {
    box-shadow: 0 4px 22px rgba(239, 68, 68, 0.45);
    border-color: #dc2626;
  }
}
</style>
