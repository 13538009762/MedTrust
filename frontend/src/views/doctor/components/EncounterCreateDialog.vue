<!--
  EncounterCreateDialog — 接诊初诊工作台模板 (WP-16)
  从 DoctorRecordsView.vue L30–507 提取，行为 / 文案 / 类名保持不变。
  Props: initForm / vitals / directForm 为响应式对象（属性双向可变），
  m 与 examList 经 defineModel 双向同步，医生输入实时回写工作流状态。
-->
<template>
  <div class="inpage-workstation-wrapper">
    <!-- ===== 顶部标题栏 ===== -->
    <div class="ws-page-header-card sticky-header mb-4">
      <div class="ws-header-left">
        <el-button :icon="ArrowLeft" plain size="small" @click="$emit('cancel')">返回列表</el-button>
        <div class="ws-header-title-box">
          <span class="ws-page-title">新建临床就诊录入</span>
          <el-tag size="small" type="primary" effect="light">初诊接诊</el-tag>
          <span class="ws-header-sub">接诊医师：{{ doctorInfo }}</span>
        </div>
      </div>
      <div class="ws-header-right">
        <InfectionSafetyAlert ref="localInfectionAlertRef" v-if="selPid && ia?.has_risk" :alert="ia!" mode="badge" />
      </div>
    </div>

    <div v-if="selPid && ia?.has_risk" class="mb-3 mt-3">
      <InfectionSafetyAlert :alert="ia!" mode="corner-banner" />
    </div>

    <!-- ===== 状态一：尚未选择患者 ===== -->
    <div v-if="!selPid" class="select-patient-first-container mt-3">
      <el-card shadow="never" class="select-patient-card">
        <template #header>
          <div class="sp-card-header">
            <div class="sp-title-badge"><span class="step-badge">1</span><span class="sp-title">接诊首要步骤：请选择本次接诊患者</span></div>
            <el-tag type="warning" effect="plain">身份挂号核验</el-tag>
          </div>
        </template>
        <div class="sp-card-body">
          <el-form label-position="top" @submit.prevent>
            <el-form-item label="选择挂号就诊患者" required class="mb-3">
              <el-select :model-value="selPid" placeholder="请搜索或选择本次接诊患者（支持患者姓名、档案卡号检索）..."
                filterable clearable default-first-option size="large" style="width: 100%;"
                no-match-text="未检索到匹配患者" no-data-text="暂无可选就诊患者"
                :filter-method="handlePatientFilter"
                @visible-change="handlePatientDropdownVisible"
                @update:model-value="(pid: number | undefined) => emit('onSelectPatient', pid)">
                <el-option v-for="p in filteredPatientList" :key="p.id"
                  :label="`${p.real_name || p.username} (卡号: ${p.user_no}${idCard(p.id_card) ? ' · ' + maskID(p.id_card) : ''})`"
                  :value="p.id" />
              </el-select>
            </el-form-item>
          </el-form>
          <div class="sp-quick-chips mb-4">
            <span class="chip-label">快捷接诊候选患者：</span>
            <el-button v-for="cand in patientList" :key="cand.id" size="small" plain class="sp-chip-btn"
              @click="emit('onPickQuickPatient', cand.id)">
              <el-icon class="mr-1"><User /></el-icon><span>{{ cand.real_name || cand.username }}</span>
              <span class="chip-sub">({{ cand.gender === 'FEMALE' || cand.gender === '女' ? '女' : '男' }} · {{ cand.age || 40 }}岁)</span>
            </el-button>
          </div>
          <div class="sp-empty-guide">
            <div class="sp-guide-icon"><el-icon :size="56" color="#059669"><UserFilled /></el-icon></div>
            <h3>请先选定本次接诊的就诊患者</h3>
            <p>为了保障医疗数据合规性与患者隐私，必须首先指定接诊患者。选定后，系统将自动核验其身份档案、既往过敏筛查与传染病预警，并立即展开完整的临床接诊与规范 SOAP 病历录入工作台。</p>
            <div class="sp-guide-checklist">
              <div class="check-item"><el-icon color="#059669"><CircleCheckFilled /></el-icon> 自动核验患者电子健康档案与实名医保卡</div>
              <div class="check-item"><el-icon color="#059669"><CircleCheckFilled /></el-icon> 实时筛查既往药物过敏史与医护传染病暴露预警</div>
              <div class="check-item"><el-icon color="#059669"><CircleCheckFilled /></el-icon> 展开临床主诉采集、体征测量、诊断处方与检查单派发</div>
            </div>
          </div>
        </div>
      </el-card>
    </div>

    <!-- ===== 状态二：已选定患者（左右联动分栏：左为患者过敏画像与历史时间线，右为临床录入表单） ===== -->
    <template v-else>
    <el-form label-position="top" class="encounter-form mt-3">
      <div class="ws-body-grid encounter-create-grid">
        <div class="ws-col-left encounter-create-left">
          <!-- Step 1: 患者实名画像、血型与过敏筛查 -->
          <div class="ws-card">
            <div class="ws-card-title">
              <span class="step-badge">1</span><span>已核验就诊患者画像</span>
              <el-button size="small" type="primary" link style="margin-left: auto;" @click="$emit('changePatient')">重新选人 / 切换</el-button>
            </div>
            <div class="patient-quick-card">
              <div class="patient-profile-top">
                <div class="ppt-avatar">{{ (pi.real_name || pi.username || '患').slice(0, 1) }}</div>
                <div class="ppt-meta">
                  <div class="ppt-name-row">
                    <span class="ppt-name">{{ pi.real_name || pi.username }}</span>
                    <el-tag size="small" :type="pi.gender === 'FEMALE' || pi.gender === '女' ? 'danger' : 'primary'" effect="plain">
                      {{ pi.gender === 'FEMALE' || pi.gender === '女' ? '女' : (pi.gender ? '男' : '性别未登记') }} / {{ pi.age ? `${pi.age}岁` : '年龄未登记' }}
                    </el-tag>
                  </div>
                  <div class="ppt-no mono">卡号: {{ pi.user_no }}</div>
                </div>
              </div>
              <div class="patient-info-rows mt-2">
                <div class="p-row"><span class="p-label">公民身份证号</span><span class="p-val mono">{{ maskID(pi.id_card) }}</span></div>
                <div class="p-row">
                  <span class="p-label">血型归属</span>
                  <span class="p-val">
                    <el-tag size="small" :type="pi.blood_type && pi.blood_type !== '未登记' ? 'primary' : 'info'" effect="plain">
                      {{ pi.blood_type || '未登记' }}
                    </el-tag>
                  </span>
                </div>
                <div class="p-row">
                  <span class="p-label">既往过敏筛查</span>
                  <span class="p-val">
                    <el-tag v-if="patientHasAllergy" type="danger" size="small" effect="light">{{ pi.allergies }}</el-tag>
                    <el-tag v-else type="info" size="small" effect="light">{{ pi.allergies || '未登记药物过敏' }}</el-tag>
                  </span>
                </div>
                <div class="p-row">
                  <span class="p-label">既往慢病史</span>
                  <span class="p-val">{{ pi.chronic_diseases || '未登记' }}</span>
                </div>
              </div>
              <div v-if="ia?.has_risk" class="patient-infection-alert-box">
                <div class="pia-head">
                  <div class="pia-title"><el-icon class="pia-icon"><WarningFilled /></el-icon><span>医护安全防护专项预警</span></div>
                  <el-button type="danger" link size="small" class="pia-guide-link" @click="handleOpenInfectionGuide">防护规程 ></el-button>
                </div>
                <div class="pia-body">
                  <div class="pia-disease"><span class="pia-label">确诊携带：</span><strong class="pia-disease-name">{{ alertDiseasesName }}</strong></div>
                  <div class="pia-notice">须严格落实标准屏障防护，防止接诊锐器伤及职业暴露</div>
                  <div v-if="ia.protection_gear?.length" class="pia-gears">
                    <span class="pia-gear-item" v-for="gear in ia.protection_gear.slice(0, 3)" :key="gear">{{ gear }}</span>
                  </div>
                </div>
              </div>
              <div class="patient-cross-action">
                <el-button type="warning" size="default" :icon="Search" class="cross-patient-query-btn" @click="$emit('crossQueryPatient')">跨机构调取历史病历 (身份证自动联查)</el-button>
              </div>
            </div>
          </div>

          <!-- 左侧联动面板：既往病例纵向时间线 -->
          <div class="ws-card encounter-timeline-card">
            <div class="ws-card-title">
              <span class="step-badge is-timeline"><el-icon><Clock /></el-icon></span>
              <span>既往病例时间线</span>
              <span class="tl-head-hint">辅助核查历史病因与症状演变</span>
            </div>
            <PatientClinicalTimeline
              :records="timelineRecords"
              :loading="timelineLoading"
              :compact="true"
              max-height="none"
              :allow-adopt="true"
              :current-hospital-id="currentHospitalId"
              empty-description="该患者暂无已归档的历史就诊记录"
              @adopt-history="handleAdoptTimeline"
              @view-detail="openTimelineRecordPreview"
            />
          </div>
        </div>

        <!-- 右侧栏：临床就诊录入工作流表单 -->
        <div class="ws-col-main encounter-create-main">
          <div class="encounter-top-meta-grid">
            <!-- Step 2 -->
            <div class="ws-card">
              <div class="ws-card-title"><span class="step-badge">2</span><span>就诊基本信息与挂号分诊</span></div>
              <el-row :gutter="12">
                <el-col :span="12">
                  <el-form-item label="就诊科室" required>
                    <el-select v-model="initForm.department_name" style="width: 100%;">
                      <el-option label="心血管内科" value="心血管内科" />
                      <el-option label="呼吸与危重症医学科" value="呼吸与危重症医学科" />
                      <el-option label="消化内科" value="消化内科" />
                      <el-option label="神经内科" value="神经内科" />
                      <el-option label="急诊重症科" value="急诊重症科" />
                      <el-option label="全科综合门诊" value="全科综合门诊" />
                    </el-select>
                  </el-form-item>
                </el-col>
                <el-col :span="12">
                  <el-form-item label="就诊类型" required>
                    <el-select v-model="initForm.encounter_type" style="width: 100%;">
                      <el-option label="普通门诊" value="OUTPATIENT" />
                      <el-option label="急诊" value="EMERGENCY" />
                      <el-option label="住院" value="INPATIENT" />
                    </el-select>
                  </el-form-item>
                </el-col>
              </el-row>
              <el-form-item label="发病时间">
                <el-date-picker v-model="initForm.onset_time" type="datetime" placeholder="选择发病时间" format="YYYY-MM-DD HH:mm:ss" value-format="YYYY-MM-DD HH:mm:ss" style="width: 100%;" />
              </el-form-item>
              <el-form-item label="症状持续时间">
                <el-input v-model="initForm.duration" placeholder="例如：阵发性反复发作 3 天，活动后加重" />
              </el-form-item>
            </div>

            <!-- Step 3 -->
            <div class="ws-card">
              <div class="ws-card-title flex justify-between items-center">
                <div><span class="step-badge">3</span><span>生命体征参数 (Vital Signs)</span></div>
                <el-button size="small" type="primary" link @click="$emit('quickFillVitals')">填入标准参考体征</el-button>
              </div>
              <el-row :gutter="10">
                <el-col :span="12">
                  <el-form-item label="体温 (T)">
                    <el-input v-model="vitals.temperature" placeholder="36.5"><template #suffix><span class="unit-text">℃</span></template></el-input>
                    <div class="vital-ref-indicator mt-1"><span :class="['vital-badge-tag', tempBadge.level]">{{ tempBadge.label }}</span></div>
                  </el-form-item>
                </el-col>
                <el-col :span="12">
                  <el-form-item label="血压 (BP)">
                    <el-input v-model="vitals.blood_pressure" placeholder="125/80"><template #suffix><span class="unit-text">mmHg</span></template></el-input>
                    <div class="vital-ref-indicator mt-1"><span :class="['vital-badge-tag', bpBadge.level]">{{ bpBadge.label }}</span></div>
                  </el-form-item>
                </el-col>
                <el-col :span="12">
                  <el-form-item label="心率/脉搏 (HR)">
                    <el-input v-model="vitals.heart_rate" placeholder="78"><template #suffix><span class="unit-text">bpm</span></template></el-input>
                    <div class="vital-ref-indicator mt-1"><span :class="['vital-badge-tag', hrBadge.level]">{{ hrBadge.label }}</span></div>
                  </el-form-item>
                </el-col>
                <el-col :span="12">
                  <el-form-item label="血氧 (SpO2)">
                    <el-input v-model="vitals.spo2" placeholder="98"><template #suffix><span class="unit-text">%</span></template></el-input>
                    <div class="vital-ref-indicator mt-1"><span :class="['vital-badge-tag', spo2Badge.level]">{{ spo2Badge.label }}</span></div>
                  </el-form-item>
                </el-col>
              </el-row>
            </div>
          </div>

          <!-- Step 4 -->
          <div class="ws-card">
            <div class="ws-card-title">
              <span class="step-badge">4</span><span>初诊信息采集 (S + O 初步)</span>
              <el-button size="small" class="ml-auto ai-diagnose-capsule-btn" @click="$emit('openAiDiagnosticAssist')">
                <el-icon class="ai-spark-icon"><DocumentChecked /></el-icon><span>跨院既往病史 SOAP 汇总与隐匿线索</span>
              </el-button>
            </div>
            <div class="symptom-input-header-row">
              <span class="symptom-req-label">患者主诉 (Chief Complaint) <span class="text-danger">*</span></span>
              <span class="symptom-ai-tip text-muted">输入患者主要症状后，可点击右侧按钮调阅已授权跨期病历提取 SOAP 摘要与隐匿线索</span>
            </div>
            <div class="symptom-input-with-action mb-3">
              <el-input v-model="initForm.chief_complaint" placeholder="简明记录患者就诊主要症状和持续时间，例如：胸闷心悸伴气促2天，劳累后加剧" class="symptom-flex-input" />
            </div>
            <el-form-item label="现病史与诱因 (Present Illness)">
              <el-input v-model="initForm.present_illness" type="textarea" :rows="2" placeholder="详细记录起病情况、主要症状特点、伴随症状、诱因及既往诊治经过" />
            </el-form-item>
          </div>

          <!-- Step 5: 临床决策 -->
          <div class="ws-card">
            <div class="ws-card-title"><span class="step-badge">5</span><span>临床决策与诊疗路径选择 (Clinical Decision)</span></div>
            <div class="decision-mode-bento mb-3">
              <div
                class="decision-option-card"
                role="button"
                tabindex="0"
                :class="{ active: m === 'DIRECT' }"
                @click="m = 'DIRECT'; $emit('generateDirectSoap')"
                @keydown.enter.space.prevent="m = 'DIRECT'; $emit('generateDirectSoap')"
              >
                <div class="doc-radio-circle"><div v-if="m === 'DIRECT'" class="doc-radio-inner" /></div>
                <div class="doc-content">
                  <div class="doc-title-row">
                    <span class="doc-badge direct">路径 A</span><span class="doc-title">临床体征明确 · 直接下达确诊与处方</span>
                    <el-tag size="small" type="success" effect="light" class="ml-auto">一步存证闭环</el-tag>
                  </div>
                  <div class="doc-desc">适用于门诊常规、轻症或体征典型患者。直接下达确诊与处方方案，同步完成 AES-256 加密与 Fabric 联盟链存证，无需开立辅助检查。</div>
                </div>
              </div>
              <div
                class="decision-option-card"
                role="button"
                tabindex="0"
                :class="{ active: m === 'EXAM' }"
                @click="m = 'EXAM'"
                @keydown.enter.space.prevent="m = 'EXAM'"
              >
                <div class="doc-radio-circle"><div v-if="m === 'EXAM'" class="doc-radio-inner" /></div>
                <div class="doc-content">
                  <div class="doc-title-row">
                    <span class="doc-badge exam">路径 B</span><span class="doc-title">病情待查 · 开立医技辅助检查单</span>
                    <el-tag size="small" type="primary" effect="light" class="ml-auto">科室协同流转</el-tag>
                  </div>
                  <div class="doc-desc">适用于初诊疑难待查。系统自动生成 ORD 检查申请单派发至检验科/影像中心，回传报告后再终审确诊。</div>
                </div>
              </div>
            </div>

            <div v-if="m === 'DIRECT'" class="direct-decision-panel">
              <el-row :gutter="16">
                <el-col :span="12">
                  <el-form-item label="经治医生最终确诊 (Final Diagnosis)" required>
                    <el-input v-model="directForm.diagnosis" type="textarea" :autosize="{ minRows: 2, maxRows: 4 }" placeholder="例如：原发性高血压病1级 / 上呼吸道感染 / 慢性胃炎" />
                  </el-form-item>
                </el-col>
                <el-col :span="12">
                  <el-form-item label="病理诱因与发病机制 (Etiology)">
                    <el-input v-model="directForm.etiology" type="textarea" :autosize="{ minRows: 2, maxRows: 4 }" placeholder="例如：气候变化受凉致黏膜屏障受损 / 劳累诱发交感神经兴奋" />
                  </el-form-item>
                </el-col>
              </el-row>
              <div class="rx-preset-capsules mb-2">
                <span class="rx-preset-title">规范处方模板录入：</span>
                <el-button v-for="p in presets" :key="p.key" size="small" :type="p.type" plain round class="rx-capsule-btn"
                  @click="$emit('applyPrescriptionPreset', p, 'DIRECT')">{{ p.label }}</el-button>
              </div>
              <el-form-item label="综合处置、医嘱与处方用药方案 (分区分条下达 · 支持药品关键词下拉提示)" required>
                <TreatmentPlanEditor
                  v-model="directForm.treatment_plan"
                  @change="$emit('generateDirectSoap')"
                />
              </el-form-item>
              <div class="soap-quick-action mb-2" style="display: flex; gap: 10px; align-items: center;">
                <el-button size="small" type="primary" plain @click="$emit('generateDirectSoap')">临床规则引擎生成规范 SOAP 病历结构</el-button>
                <el-button size="small" type="info" link @click="$emit('fillDirectPreset')">载入常规高血压门诊处方示例</el-button>
              </div>
              <el-form-item v-if="directForm.soap_content" label="规范临床 SOAP 病历全文确认 (将进行 AES-256-GCM 加密与 Fabric 区块链存证)">
                <el-input v-model="directForm.soap_content" type="textarea" :rows="6" class="soap-textarea" />
              </el-form-item>
            </div>

            <div v-else class="exam-decision-panel">
              <el-row :gutter="16">
                <el-col :span="12">
                  <el-form-item label="初步诊断 (Initial Diagnosis)" required>
                    <el-input v-model="initForm.initial_diagnosis" type="textarea" :autosize="{ minRows: 2, maxRows: 4 }" placeholder="例如：原发性高血压病1级待查；心律失常待排" />
                  </el-form-item>
                </el-col>
                <el-col :span="12">
                  <el-form-item label="接诊依据与拟诊分析">
                    <el-input v-model="initForm.diagnostic_basis" type="textarea" :autosize="{ minRows: 2, maxRows: 4 }" placeholder="例如：患者血压145/95mmHg，活动后胸闷心悸，需辅助检查确证" />
                  </el-form-item>
                </el-col>
              </el-row>
              <div class="dispatch-target-tip mb-3 mt-2">
                <el-alert title="医技检查单分流派发机制：" type="info" :closable="false"
                  description="① 血常规 / 心肌酶谱 / 肝肾生化 -> 自动派发至【临床检验医学中心·LIS任务池】(由检验科技师 tech_lab 接单)；
② 12导联心电图 / 胸部CT / 头颅MRI -> 自动派发至【放射影像与心电中心·PACS任务池】(由影像医师 tech_pacs 接单)。
提交后医技人员即可在对应工作台接单处理，当前医生亦可通过左侧「医技检查中心」协同出报告。" show-icon />
              </div>
              <el-form-item label="开具辅助检查项目 (勾选后自动生成独立申请单号 ORD... 移交医技中心)">
                <div class="exam-picker-grid">
                  <el-checkbox-group v-model="examList" class="exam-checkbox-container">
                    <div class="exam-select-card" :class="{ selected: examList.includes('12导联心电图 (ECG)') }">
                      <el-checkbox label="12导联心电图 (ECG)"><span class="exam-label-text">12导联心电图 (ECG)</span></el-checkbox><el-tag size="small" type="warning" effect="plain">影像心电中心</el-tag>
                    </div>
                    <div class="exam-select-card" :class="{ selected: examList.includes('常规全血细胞分析 (血常规)') }">
                      <el-checkbox label="常规全血细胞分析 (血常规)"><span class="exam-label-text">常规全血细胞分析 (血常规)</span></el-checkbox><el-tag size="small" type="primary" effect="plain">临床检验科</el-tag>
                    </div>
                    <div class="exam-select-card" :class="{ selected: examList.includes('血清心肌酶谱与肌钙蛋白') }">
                      <el-checkbox label="血清心肌酶谱与肌钙蛋白"><span class="exam-label-text">血清心肌酶谱与肌钙蛋白</span></el-checkbox><el-tag size="small" type="primary" effect="plain">临床检验科</el-tag>
                    </div>
                    <div class="exam-select-card" :class="{ selected: examList.includes('胸部高分辨率 CT 平扫') }">
                      <el-checkbox label="胸部高分辨率 CT 平扫"><span class="exam-label-text">胸部高分辨率 CT 平扫</span></el-checkbox><el-tag size="small" type="warning" effect="plain">放射影像中心</el-tag>
                    </div>
                    <div class="exam-select-card" :class="{ selected: examList.includes('肝肾功能与电解质生化检查') }">
                      <el-checkbox label="肝肾功能与电解质生化检查"><span class="exam-label-text">肝肾功能与电解质生化</span></el-checkbox><el-tag size="small" type="primary" effect="plain">临床检验科</el-tag>
                    </div>
                    <div class="exam-select-card" :class="{ selected: examList.includes('头颅 MRI 平扫') }">
                      <el-checkbox label="头颅 MRI 平扫"><span class="exam-label-text">头颅 MRI 平扫</span></el-checkbox><el-tag size="small" type="warning" effect="plain">放射影像中心</el-tag>
                    </div>
                  </el-checkbox-group>
                </div>
              </el-form-item>
              <el-form-item label="检查原因及临床指征">
                <el-input v-model="initForm.exam_reason" type="textarea" :autosize="{ minRows: 2, maxRows: 4 }" placeholder="说明开具该检查的目的，例如：排查急性心肌缺血病变及全身炎性指标" />
              </el-form-item>
            </div>
          </div>

          <!-- Chain Security Preview (置于右栏末尾，保持左栏专注患者时间线) -->
          <div class="ws-card chain-security-preview-card">
            <div class="csp-header">
              <div class="csp-title"><el-icon class="csp-icon"><Lock /></el-icon><span>链上存证与信封加密就绪预览</span></div>
              <span class="mini-pulse-dot dot-green"></span>
            </div>
            <div class="csp-body csp-body-inline">
              <div class="csp-row"><span class="csp-k">加密引擎：</span><span class="csp-v mono">AES-256-GCM (AAD 强绑定)</span></div>
              <div class="csp-row"><span class="csp-k">目标通道：</span><span class="csp-v mono">medicalchannel</span></div>
              <div class="csp-row"><span class="csp-k">智能合约：</span><span class="csp-v mono">medical_contract</span></div>
              <div class="csp-row"><span class="csp-k">共识节点：</span><span class="csp-v"><span class="csp-status-online">Raft (etcdraft) 在线</span></span></div>
              <div class="csp-row"><span class="csp-k">去中心存储：</span><span class="csp-v mono">IPFS 分布式节点寻址</span></div>
            </div>
            <div class="csp-footer"><span class="csp-tip">就诊初审提交后，系统将自动对 S-O-A-P 病历实施信封加密并上链广播存证。</span></div>
          </div>
        </div>
      </div>
    </el-form>

    <!-- 历史病历节点只读对照弹窗（避免退出当前初诊录入表单） -->
    <el-dialog
      v-model="previewVisible"
      title="既往历史病历详情对照（只读参考）"
      width="min(840px, 92vw)"
      align-center
      append-to-body
    >
      <div v-if="previewRecord" class="tl-preview-scroll-body">
        <div class="tl-preview-head">
          <div>
            <div class="tl-preview-title">
              {{ cleanClinicalText(previewRecord.diagnosis || previewRecord.initial_diagnosis) || '待确诊记录' }}
            </div>
            <div class="tl-preview-sub">
              <span>{{ formatTime(previewRecord.created_at) }}</span>
              <span>·</span>
              <span>{{ formatEncounterType(previewRecord.encounter_type) }}</span>
              <span>·</span>
              <span>{{ previewRecord.hospital_name || '既往就诊机构' }} / {{ previewRecord.department_name || '接诊科室' }}</span>
            </div>
          </div>
          <ChainVerifyBadge :status="resolveChainVerifyStatus(previewRecord)" />
        </div>

        <el-descriptions
          :column="1"
          border
          size="default"
          label-width="144px"
          class="mt-3 tl-preview-descriptions"
        >
          <el-descriptions-item label="主诉与现象">
            {{ cleanClinicalText(previewRecord.chief_complaint || previewRecord.symptoms) || '未登记' }}
          </el-descriptions-item>
          <el-descriptions-item label="现病史摘要">
            {{ cleanClinicalText(previewRecord.present_illness) || '未登记' }}
          </el-descriptions-item>
          <el-descriptions-item label="病因与诊断依据">
            {{ cleanClinicalText(previewRecord.etiology || previewRecord.diagnostic_basis) || '未记录明确病因' }}
          </el-descriptions-item>
          <el-descriptions-item label="客观检查与结果">
            {{ [previewRecord.exam_items, cleanClinicalText(previewRecord.exam_result)].filter(Boolean).join(' ｜ ') || '无检查记录' }}
          </el-descriptions-item>
          <el-descriptions-item label="既往处置方案">
            <div style="white-space: pre-wrap;">{{ cleanClinicalText(previewRecord.treatment_plan) || '未记录处置处方' }}</div>
          </el-descriptions-item>
          <el-descriptions-item label="链上存证指纹">
            <div style="display: flex; flex-wrap: wrap; gap: 8px;">
              <HashCapsule :value="previewRecord.fabric_tx_id" label="TxID" empty-text="待上链" />
              <HashCapsule :value="previewRecord.data_hash" label="DataHash" empty-text="—" />
              <HashCapsule :value="previewRecord.ipfs_cid" label="IPFS CID" empty-text="—" />
            </div>
          </el-descriptions-item>
        </el-descriptions>
      </div>
      <template #footer>
        <el-button @click="previewVisible = false">关闭</el-button>
        <el-button
          v-if="previewRecord && !previewRecord.is_tampered && isRecordAccessible(previewRecord)"
          type="primary"
          @click="adoptFromPreviewDialog"
        >
          引用摘要至现病史
        </el-button>
      </template>
    </el-dialog>

    <!-- 底部提交操作栏 -->
    <div class="ws-page-footer-card mt-4">
      <div class="ws-footer-bar">
        <div class="ws-footer-left">
          <span class="ws-footer-crypto-badge"><span class="mini-pulse-dot dot-green"></span>AES-256-GCM (AAD绑定) + Fabric (medicalchannel) 存证就绪</span>
          <span v-if="m === 'DIRECT'" class="ws-footer-stat-tip">就诊模式：确诊处方一体化上链 · 即时固化</span>
          <span v-else class="ws-footer-stat-tip">多科室协同：已开立 {{ examList.length }} 项医技申请</span>
        </div>
        <div class="ws-footer-right">
          <el-button @click="$emit('cancel')">取消退出</el-button>
          <el-button v-if="m === 'DIRECT'" type="success" size="default" :loading="submitting"
            :disabled="!selPid || !initForm.chief_complaint || !directForm.diagnosis || !directForm.treatment_plan"
            @click="$emit('submitDirect')"
          >确认最终确诊并直接归档 (加密存证闭环)</el-button>
          <el-button v-else type="primary" size="default" :loading="submitting"
            :disabled="!selPid || !initForm.chief_complaint || !initForm.initial_diagnosis || examList.length === 0"
            @click="$emit('submitExam')"
          >提交初诊并派发检查申请单 (移交医技科室)</el-button>
        </div>
      </div>
    </div>
    </template>
  </div>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { ElMessage } from 'element-plus'
import { ArrowLeft, WarningFilled, User, UserFilled, CircleCheckFilled, Search, Lock, DocumentChecked, Clock } from '@element-plus/icons-vue'
import InfectionSafetyAlert from '../../../components/InfectionSafetyAlert.vue'
import TreatmentPlanEditor from '@/components/TreatmentPlanEditor.vue'
import PatientClinicalTimeline from '@/components/clinical/PatientClinicalTimeline.vue'
import ChainVerifyBadge from '@/components/common/ChainVerifyBadge.vue'
import HashCapsule from '@/components/common/HashCapsule.vue'
import { hasRealAllergy } from '@/utils/emergencyPatientUtils'
import { filterPatientOptions } from '@/utils/patientSearchUtils'
import {
  evaluateTemperature,
  evaluateBloodPressure,
  evaluateHeartRate,
  evaluateSpo2,
} from '@/utils/vitalSignsEvaluator'
import {
  buildHistoryClueSnippet,
  cleanClinicalText,
  formatEncounterType,
  formatTime,
  isRecordAccessible,
  resolveChainVerifyStatus
} from '@/utils/timelineFormatters'
import type { PatientOption, PrescriptionPreset, EncounterInitForm, VitalSigns, DiagnosisFormState, EncounterMode, MedicalRecordRow, InfectionAlert } from '@/types/clinical'

interface Props {
  patientList: PatientOption[]
  selPid?: number
  si: PatientOption | null
  ia: InfectionAlert | null
  initForm: EncounterInitForm
  vitals: VitalSigns
  directForm: DiagnosisFormState
  presets: PrescriptionPreset[]
  submitting: boolean
  doctorInfo: string
  timelineRecords?: MedicalRecordRow[]
  timelineLoading?: boolean
  currentHospitalId?: number
}

const props = withDefaults(defineProps<Props>(), {
  patientList: () => [],
  selPid: undefined,
  si: null,
  ia: null,
  presets: () => [],
  submitting: false,
  doctorInfo: '',
  timelineRecords: () => [],
  timelineLoading: false,
  currentHospitalId: undefined
})

// 双向绑定：临床决策路径与已勾选医技检查项目
const m = defineModel<EncounterMode>('m', { required: true })
const examList = defineModel<string[]>('examList', { required: true })

const emit = defineEmits<{
  cancel: []
  changePatient: []
  openInfectionGuide: []
  crossQueryPatient: []
  quickFillVitals: []
  openAiDiagnosticAssist: []
  generateDirectSoap: []
  applyPrescriptionPreset: [preset: PrescriptionPreset, target: string]
  fillDirectPreset: []
  submitDirect: []
  submitExam: []
  onSelectPatient: [pid: number | undefined]
  onPickQuickPatient: [pid: number]
}>()

const selPid = computed(() => props.selPid)
const pi = computed(() => props.si ?? ({ real_name: '', username: '' } as PatientOption))
const ia = computed(() => props.ia)
const patientHasAllergy = computed(() => hasRealAllergy(pi.value))

const patientQuery = ref('')
const filteredPatientList = computed(() => filterPatientOptions(props.patientList, patientQuery.value))

function handlePatientFilter(query: string) {
  patientQuery.value = query
}

function handlePatientDropdownVisible(visible: boolean) {
  if (!visible) {
    patientQuery.value = ''
  }
}

const previewVisible = ref(false)
const previewRecord = ref<MedicalRecordRow | null>(null)

function handleAdoptTimeline(snippet: string) {
  if (!snippet) return
  const current = props.initForm.present_illness?.trim() || ''
  if (current.includes(snippet)) {
    ElMessage.info('该历史病例摘要已在现病史栏中，无需重复引用')
    return
  }
  props.initForm.present_illness = current
    ? `${current}\n【既往病史参考】${snippet}`
    : `【既往病史参考】${snippet}`
  ElMessage.success('已将历史病例摘要引入右侧「现病史与诱因」')
}

function openTimelineRecordPreview(rec: MedicalRecordRow) {
  previewRecord.value = rec
  previewVisible.value = true
}

function adoptFromPreviewDialog() {
  if (!previewRecord.value) return
  const snippet = buildHistoryClueSnippet(previewRecord.value, previewRecord.value.hospital_name)
  handleAdoptTimeline(snippet)
  previewVisible.value = false
}

// Computed helpers
const alertDiseasesName = computed(() => {
  const a = props.ia
  if (!a) return ''
  const diseases = (a.diseases as Array<{ name: string }>) ?? undefined
  return Array.isArray(diseases) ? diseases.map((d) => d.name).join('、') : (a.summary || '')
})

const localInfectionAlertRef = ref<{ openGuideModal?: () => void } | null>(null)
function handleOpenInfectionGuide() {
  if (localInfectionAlertRef.value?.openGuideModal) {
    localInfectionAlertRef.value.openGuideModal()
  } else {
    emit('openInfectionGuide')
  }
}

const tempBadge = computed(() => evaluateTemperature(props.vitals.temperature))
const bpBadge = computed(() => evaluateBloodPressure(props.vitals.blood_pressure))
const hrBadge = computed(() => evaluateHeartRate(props.vitals.heart_rate))
const spo2Badge = computed(() => evaluateSpo2(props.vitals.spo2))

function idCard(v?: string): boolean { return !!v && v.length >= 8 }
function maskID(idCard?: string): string {
  if (!idCard || idCard.length < 8) return '已脱敏保护'
  return idCard.substring(0, 6) + '********' + idCard.substring(idCard.length - 4)
}
</script>

<style scoped src="../DoctorRecordsView.css"></style>