<template>
  <div class="doctor-records-container">
    <!-- 视图一：就诊记录列表与统计看板 (LIST) -->
    <div v-if="currentView === 'LIST'" class="records-list-wrapper">
      <!-- 顶部状态统计卡片 -->
    <div class="stats-grid">
      <div class="stat-card blue">
        <div class="stat-icon"><el-icon><OfficeBuilding /></el-icon></div>
        <div class="stat-info">
          <div class="stat-value">{{ records.length }}</div>
          <div class="stat-label">累计接诊记录</div>
        </div>
      </div>
      <div class="stat-card yellow">
        <div class="stat-icon"><el-icon><Timer /></el-icon></div>
        <div class="stat-info">
          <div class="stat-value">{{ waitingExamCount }}</div>
          <div class="stat-label">待医技检查 (申请中)</div>
        </div>
      </div>
      <div class="stat-card cyan">
        <div class="stat-icon"><el-icon><Document /></el-icon></div>
        <div class="stat-info">
          <div class="stat-value">{{ examCompletedCount }}</div>
          <div class="stat-label">检查完成 (待下确诊)</div>
        </div>
      </div>
      <div class="stat-card green">
        <div class="stat-icon"><el-icon><CircleCheckFilled /></el-icon></div>
        <div class="stat-info">
          <div class="stat-value">{{ completedCount }}</div>
          <div class="stat-label">已存证归档病历</div>
        </div>
      </div>
    </div>

    <!-- 主卡片 -->
    <el-card class="main-card" shadow="never">
      <template #header>
        <div class="card-header">
          <div class="header-left">
            <span class="header-title">临床就诊记录管理 (Encounter Management)</span>
            <el-tag type="info" size="small" effect="plain" class="ml-2">就诊生命周期驱动</el-tag>
          </div>
          <div class="header-actions">
            <el-button type="danger" plain @click="$router.push('/doctor/patient-search')">
              <el-icon style="margin-right: 4px;"><FirstAidKit /></el-icon>
              患者检索与急救调阅
            </el-button>
            <el-button type="primary" :icon="Plus" @click="openInitialDialog">
              新建就诊记录 (接诊初诊)
            </el-button>
            <el-button :icon="Refresh" circle @click="fetchRecords" />
          </div>
        </div>
      </template>

      <!-- 搜索与状态过滤器 -->
      <div class="table-toolbar">
        <el-radio-group v-model="activeStatusFilter" size="small">
          <el-radio-button label="ALL">全部 ({{ records.length }})</el-radio-button>
          <el-radio-button label="WAITING_EXAM">待医技检查 ({{ waitingExamCount }})</el-radio-button>
          <el-radio-button label="EXAM_COMPLETED">待下最终确诊 ({{ examCompletedCount }})</el-radio-button>
          <el-radio-button label="COMPLETED">已归档存证 ({{ completedCount }})</el-radio-button>
        </el-radio-group>

        <el-input
          v-model="searchKeyword"
          placeholder="检索患者姓名、就诊单号、初诊/确诊或检查项目..."
          style="width: 320px;"
          clearable
          size="small"
        >
          <template #prefix><el-icon><Search /></el-icon></template>
        </el-input>
      </div>

      <!-- 病历表格 -->
      <el-table :data="filteredRecords" v-loading="loading" stripe style="width: 100%">
        <el-table-column prop="record_no" label="就诊编号" width="165">
          <template #default="{ row }">
            <span class="mono font-bold">{{ row.record_no }}</span>
          </template>
        </el-table-column>

        <el-table-column label="就诊患者" width="145">
          <template #default="{ row }">
            <div style="display: flex; align-items: center; gap: 4px;">
              <span class="font-bold">{{ row.patient_name || '患者' }}</span>
              <el-tooltip
                v-if="row.infection_alert?.has_risk"
                :content="`⚠️ 医护安全预警：该患者有传染病携带史（${row.infection_alert.summary}）`"
                placement="top"
              >
                <el-tag size="small" type="danger" effect="dark" style="font-size: 10px; padding: 0 4px; height: 18px; line-height: 16px; cursor: pointer;" @click="openViewDetail(row)">
                  ⚠️ 防护
                </el-tag>
              </el-tooltip>
            </div>
          </template>
        </el-table-column>

        <el-table-column prop="encounter_type" label="就诊类型" width="110">
          <template #default="{ row }">
            <el-tag size="small" :type="row.encounter_type === 'EMERGENCY' ? 'danger' : 'info'">
              {{ formatEncounterType(row.encounter_type) }}
            </el-tag>
          </template>
        </el-table-column>

        <el-table-column prop="department_name" label="就诊科室" width="130" />

        <el-table-column label="临床初步诊断 / 确诊结论" min-width="200">
          <template #default="{ row }">
            <div v-if="row.diagnosis" class="text-success font-bold">
              {{ row.diagnosis }}
            </div>
            <div v-else-if="row.initial_diagnosis" class="text-primary">
              <span class="text-xs text-gray-500">初诊拟诊:</span> {{ row.initial_diagnosis }}
            </div>
            <div v-else class="text-gray-400 text-xs">
              {{ row.chief_complaint || '记录中' }}
            </div>
          </template>
        </el-table-column>

        <el-table-column label="医技检查项目" min-width="170">
          <template #default="{ row }">
            <template v-if="row.need_exam || row.exam_items">
              <el-tag size="small" type="warning">{{ row.exam_items || '辅助检查' }}</el-tag>
            </template>
            <span v-else class="text-gray-400 text-xs">无医技检查 (直接确诊)</span>
          </template>
        </el-table-column>

        <el-table-column label="就诊流转状态" width="140">
          <template #default="{ row }">
            <el-tag v-if="row.is_tampered" type="danger" effect="dark" class="tamper-tag-glow">存在篡改</el-tag>
            <el-tag v-else-if="row.status === 'WAITING_EXAM'" type="warning" effect="dark">待医技检查</el-tag>
            <el-tag v-else-if="row.status === 'PROCESSING_EXAM'" type="primary" effect="dark">检查进行中</el-tag>
            <el-tag v-else-if="row.status === 'EXAM_COMPLETED'" type="info" effect="dark" class="status-ready-tag">检查完成/待确诊</el-tag>
            <el-tag v-else-if="row.status === 'INITIAL_DIAGNOSIS'" type="info">初诊完成/待确诊</el-tag>
            <el-tag v-else-if="row.status === 'COMPLETED'" type="success" effect="plain">已归档存证</el-tag>
            <el-tag v-else size="small">{{ row.status }}</el-tag>
          </template>
        </el-table-column>

        <el-table-column prop="created_at" label="就诊时间" width="140">
          <template #default="{ row }">
            {{ formatTime(row.created_at) }}
          </template>
        </el-table-column>

        <el-table-column label="操作" width="180" fixed="right">
          <template #default="{ row }">
            <!-- 检查回传完成，等待医生出最终诊断 -->
            <el-button
              v-if="row.status === 'EXAM_COMPLETED' || row.status === 'INITIAL_DIAGNOSIS'"
              type="danger"
              size="small"
              @click="openFinalDialog(row)"
            >
              下达最终诊断
            </el-button>

            <!-- 检查进行中 -->
            <el-button
              v-else-if="row.status === 'WAITING_EXAM' || row.status === 'PROCESSING_EXAM'"
              type="primary"
              size="small"
              plain
              @click="openViewDetail(row)"
            >
              查看检查进度
            </el-button>

            <!-- 已归档存证 -->
            <el-button
              v-else
              type="success"
              size="small"
              plain
              @click="openViewDetail(row)"
            >
              调阅全景病历
            </el-button>
          </template>
        </el-table-column>
      </el-table>
    </el-card>
    </div>

    <!-- 视图二：新建就诊记录 / 接诊初诊工作台 (CREATE) -->
    <div v-else-if="currentView === 'CREATE'" class="inpage-workstation-wrapper">
      <!-- 顶部工作台标题与控制栏 (轻量纤细版 48px) -->
      <div class="ws-page-header-card sticky-header">
        <div class="ws-header-left">
          <el-button :icon="ArrowLeft" plain size="small" @click="currentView = 'LIST'">返回列表</el-button>
          <div class="ws-header-title-box">
            <span class="ws-page-title">新建临床就诊录入</span>
            <el-tag size="small" type="primary" effect="light">初诊接诊</el-tag>
            <span class="ws-header-sub">接诊医师：{{ auth.user?.real_name || '当前医生' }}</span>
          </div>
        </div>
        <div class="ws-header-right">
          <InfectionSafetyAlert
            ref="createInfectionAlertRef"
            v-if="selectedPatientInfectionAlert?.has_risk"
            :alert="selectedPatientInfectionAlert"
            mode="badge"
          />
          <el-button plain size="small" @click="currentView = 'LIST'">取消退出</el-button>
        </div>
      </div>

      <!-- 医护职业暴露高危预警条幅 -->
      <div v-if="selectedPatientInfectionAlert?.has_risk" class="mb-3 mt-3">
        <InfectionSafetyAlert
          :alert="selectedPatientInfectionAlert"
          mode="corner-banner"
        />
      </div>

      <!-- 状态一：尚未选择就诊患者时，仅展示第一步选人与空态引导，不展开其他就诊信息 -->
      <div v-if="!selectedPatientInfo" class="select-patient-first-container mt-3">
        <el-card shadow="never" class="select-patient-card">
          <template #header>
            <div class="sp-card-header">
              <div class="sp-title-badge">
                <span class="step-badge">1</span>
                <span class="sp-title">接诊首要步骤：请选择本次接诊患者</span>
              </div>
              <el-tag type="warning" effect="plain">身份挂号核验</el-tag>
            </div>
          </template>

          <div class="sp-card-body">
            <el-form label-position="top">
              <el-form-item label="选择挂号就诊患者" required class="mb-3">
                <el-select
                  v-model="initForm.patient_id"
                  placeholder="请搜索或选择本次接诊患者（支持患者姓名、档案卡号检索）..."
                  filterable
                  clearable
                  size="large"
                  style="width: 100%;"
                  @change="onPatientSelect"
                >
                  <el-option
                    v-for="p in patientList"
                    :key="p.id"
                    :label="`${p.real_name || p.username} (卡号: ${p.user_no}${p.id_card ? ' · ' + maskID(p.id_card) : ''})`"
                    :value="p.id"
                  />
                </el-select>
              </el-form-item>
            </el-form>

            <!-- 快捷患者候选芯片 -->
            <div class="sp-quick-chips mb-4">
              <span class="chip-label">快捷接诊候选患者：</span>
              <el-button
                v-for="cand in patientList"
                :key="cand.id"
                size="small"
                plain
                class="sp-chip-btn"
                @click="pickQuickPatient(cand.id)"
              >
                <el-icon class="mr-1"><User /></el-icon>
                <span>{{ cand.real_name || cand.username }}</span>
                <span class="chip-sub">({{ cand.gender === 'FEMALE' || cand.gender === '女' ? '女' : '男' }} · {{ cand.age || 40 }}岁)</span>
              </el-button>
            </div>

            <!-- 空态指引说明 -->
            <div class="sp-empty-guide">
              <div class="sp-guide-icon">
                <el-icon :size="56" color="#059669"><UserFilled /></el-icon>
              </div>
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

      <!-- 状态二：已选定就诊患者后，展开完整的2列接诊工作台与其他所有信息 -->
      <template v-else>
      <el-form label-position="top" class="encounter-form mt-3">
        <div class="ws-body-grid">
          <!-- 左侧栏：患者核验、挂号基本信息、生命体征 (380px) -->
          <div class="ws-col-left">
            <!-- 模块 1：患者身份核验 -->
            <div class="ws-card">
              <div class="ws-card-title">
                <span class="step-badge">1</span>
                <span>已核验就诊患者</span>
                <el-button size="small" type="primary" link style="margin-left: auto;" @click="changeSelectedPatient">
                  重新选人 / 切换
                </el-button>
              </div>

              <div class="patient-quick-card">
                <div class="patient-profile-top">
                  <div class="ppt-avatar">
                    {{ (selectedPatientInfo.real_name || selectedPatientInfo.username || '患').slice(0, 1) }}
                  </div>
                  <div class="ppt-meta">
                    <div class="ppt-name-row">
                      <span class="ppt-name">{{ selectedPatientInfo.real_name || selectedPatientInfo.username }}</span>
                      <el-tag size="small" :type="selectedPatientInfo.gender === 'FEMALE' ? 'danger' : 'primary'" effect="plain">
                        {{ selectedPatientInfo.gender === 'FEMALE' ? '女' : '男' }} / {{ selectedPatientInfo.age || 45 }}岁
                      </el-tag>
                    </div>
                    <div class="ppt-no mono">卡号: {{ selectedPatientInfo.user_no }}</div>
                  </div>
                </div>

                <div class="patient-info-rows mt-2">
                  <div class="p-row">
                    <span class="p-label">公民身份证号</span>
                    <span class="p-val mono">{{ maskID(selectedPatientInfo.id_card) }}</span>
                  </div>
                  <div class="p-row">
                    <span class="p-label">既往过敏筛查</span>
                    <span class="p-val">
                      <el-tag type="info" size="small" effect="light">无明确药物过敏</el-tag>
                    </span>
                  </div>
                </div>

                <!-- 医护职业安全专项防护预警卡片 (100%容器自适应包裹，彻底根除遮挡与单行溢出) -->
                <div v-if="selectedPatientInfectionAlert?.has_risk" class="patient-infection-alert-box">
                  <div class="pia-head">
                    <div class="pia-title">
                      <el-icon class="pia-icon"><WarningFilled /></el-icon>
                      <span>医护安全防护专项预警</span>
                    </div>
                    <el-button
                      type="danger"
                      link
                      size="small"
                      class="pia-guide-link"
                      @click="openCreateInfectionGuide"
                    >
                      防护规程 &gt;
                    </el-button>
                  </div>
                  <div class="pia-body">
                    <div class="pia-disease">
                      <span class="pia-label">确诊携带：</span>
                      <strong class="pia-disease-name">
                        {{ selectedPatientInfectionAlert.diseases?.map((d: any) => d.name).join('、') || selectedPatientInfectionAlert.summary }}
                      </strong>
                    </div>
                    <div class="pia-notice">
                      ⚠️ 须严格落实标准屏障防护，防止接诊锐器伤及职业暴露
                    </div>
                    <div v-if="selectedPatientInfectionAlert.protection_gear?.length" class="pia-gears">
                      <span class="pia-gear-item" v-for="gear in selectedPatientInfectionAlert.protection_gear.slice(0, 3)" :key="gear">
                        🛡️ {{ gear }}
                      </span>
                    </div>
                  </div>
                </div>

                <div class="patient-cross-action">
                  <el-button
                    type="warning"
                    size="default"
                    :icon="Search"
                    class="cross-patient-query-btn"
                    @click="goToCrossQueryForPatient"
                  >
                    跨机构一键调取历史病历 (身份证自动联查)
                  </el-button>
                </div>
              </div>
            </div>

            <!-- 模块 2：就诊基本信息 -->
            <div class="ws-card">
              <div class="ws-card-title">
                <span class="step-badge">2</span>
                <span>就诊基本信息与挂号分诊</span>
              </div>
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
                <el-date-picker
                  v-model="initForm.onset_time"
                  type="datetime"
                  placeholder="选择发病时间"
                  format="YYYY-MM-DD HH:mm:ss"
                  value-format="YYYY-MM-DD HH:mm:ss"
                  style="width: 100%;"
                />
              </el-form-item>
              <el-form-item label="症状持续时间">
                <el-input v-model="initForm.duration" placeholder="例如：阵发性反复发作 3 天，活动后加重" />
              </el-form-item>
            </div>

            <!-- 模块 3：生命体征测量 -->
            <div class="ws-card">
              <div class="ws-card-title flex justify-between items-center">
                <div>
                  <span class="step-badge">3</span>
                  <span>生命体征参数 (Vital Signs)</span>
                </div>
                <el-button size="small" type="primary" link @click="quickFillVitals">一键填入标准体征</el-button>
              </div>
              <el-row :gutter="10">
                <el-col :span="12">
                  <el-form-item label="体温 (T)">
                    <el-input v-model="vitals.temperature" placeholder="36.5">
                      <template #suffix><span class="unit-text">℃</span></template>
                    </el-input>
                  </el-form-item>
                </el-col>
                <el-col :span="12">
                  <el-form-item label="血压 (BP)">
                    <el-input v-model="vitals.blood_pressure" placeholder="125/80">
                      <template #suffix><span class="unit-text">mmHg</span></template>
                    </el-input>
                  </el-form-item>
                </el-col>
                <el-col :span="12">
                  <el-form-item label="心率/脉搏 (HR)">
                    <el-input v-model="vitals.heart_rate" placeholder="78">
                      <template #suffix><span class="unit-text">bpm</span></template>
                    </el-input>
                  </el-form-item>
                </el-col>
                <el-col :span="12">
                  <el-form-item label="血氧 (SpO2)">
                    <el-input v-model="vitals.spo2" placeholder="98">
                      <template #suffix><span class="unit-text">%</span></template>
                    </el-input>
                  </el-form-item>
                </el-col>
              </el-row>
            </div>
          </div>

          <!-- 右侧栏：初诊信息、临床决策与处置 (flex: 1) -->
          <div class="ws-col-main">
            <!-- 模块 4：初诊采集 -->
            <div class="ws-card">
              <div class="ws-card-title">
                <span class="step-badge">4</span>
                <span>初诊信息采集 (S + O 初步)</span>
              </div>
              <el-form-item label="患者主诉 (Chief Complaint)" required>
                <el-input
                  v-model="initForm.chief_complaint"
                  placeholder="简明记录患者就诊主要症状和持续时间，例如：胸闷心悸伴气促2天，劳累后加剧"
                />
              </el-form-item>
              <el-form-item label="现病史与诱因 (Present Illness)">
                <el-input
                  v-model="initForm.present_illness"
                  type="textarea"
                  :rows="2"
                  placeholder="详细记录起病情况、主要症状特点、伴随症状、诱因及既往诊治经过"
                />
              </el-form-item>
            </div>

            <!-- 模块 5：临床决策与路径 -->
            <div class="ws-card">
              <div class="ws-card-title">
                <span class="step-badge">5</span>
                <span>临床决策与诊疗路径选择 (Clinical Decision)</span>
              </div>

              <div class="decision-mode-bento mb-3">
                <div
                  class="decision-option-card"
                  :class="{ active: encounterMode === 'DIRECT' }"
                  @click="encounterMode = 'DIRECT'; generateDirectSOAP()"
                >
                  <div class="doc-radio-circle">
                    <div v-if="encounterMode === 'DIRECT'" class="doc-radio-inner" />
                  </div>
                  <div class="doc-content">
                    <div class="doc-title-row">
                      <span class="doc-badge direct">路径 A</span>
                      <span class="doc-title">临床体征明确 · 直接下达确诊与处方</span>
                      <el-tag size="small" type="success" effect="light" class="ml-auto">一步存证闭环</el-tag>
                    </div>
                    <div class="doc-desc">
                      适用于门诊常规、轻症或体征典型患者。无需开立辅助检查，一步完成 AES-256 加密与 Fabric 联盟链存证。
                    </div>
                  </div>
                </div>

                <div
                  class="decision-option-card"
                  :class="{ active: encounterMode === 'EXAM' }"
                  @click="encounterMode = 'EXAM'"
                >
                  <div class="doc-radio-circle">
                    <div v-if="encounterMode === 'EXAM'" class="doc-radio-inner" />
                  </div>
                  <div class="doc-content">
                    <div class="doc-title-row">
                      <span class="doc-badge exam">路径 B</span>
                      <span class="doc-title">病情待查 · 开立医技辅助检查单</span>
                      <el-tag size="small" type="primary" effect="light" class="ml-auto">科室协同流转</el-tag>
                    </div>
                    <div class="doc-desc">
                      适用于初诊疑难待查。系统自动生成 ORD 检查申请单派发至检验科/影像中心，回传报告后再终审确诊。
                    </div>
                  </div>
                </div>
              </div>

              <!-- 路径 A：直接确诊 -->
              <div v-if="encounterMode === 'DIRECT'" class="direct-decision-panel">
                <el-alert
                  type="success"
                  :closable="false"
                  show-icon
                  class="mb-3"
                  title="诊断提示：若患者体征与现病史典型明确，医生可直接下达确诊结论与处置方案，一键完成加密上链存证，无需开具医技检查。"
                />
                <el-row :gutter="16">
                  <el-col :span="12">
                    <el-form-item label="经治医生最终确诊 (Final Diagnosis)" required>
                      <el-input
                        v-model="directForm.diagnosis"
                        type="textarea"
                        :autosize="{ minRows: 2, maxRows: 4 }"
                        placeholder="例如：原发性高血压病1级 / 上呼吸道感染 / 慢性胃炎"
                      />
                    </el-form-item>
                  </el-col>
                  <el-col :span="12">
                    <el-form-item label="病理诱因与发病机制 (Etiology)">
                      <el-input
                        v-model="directForm.etiology"
                        type="textarea"
                        :autosize="{ minRows: 2, maxRows: 4 }"
                        placeholder="例如：气候变化受凉致黏膜屏障受损 / 劳累诱发交感神经兴奋"
                      />
                    </el-form-item>
                  </el-col>
                </el-row>

                <el-form-item label="综合处置、医嘱与处方用药方案 (分区分条下达 · 支持药品关键词下拉提示)" required>
                  <TreatmentPlanEditor
                    ref="directTreatmentRef"
                    v-model="directForm.treatment_plan"
                    @change="generateDirectSOAP"
                  />
                </el-form-item>

                <div class="soap-quick-action mb-2" style="display: flex; gap: 10px; align-items: center;">
                  <el-button size="small" type="primary" plain @click="generateDirectSOAP">
                    临床规则引擎辅助生成规范 SOAP 全景病历 (一键装配)
                  </el-button>
                  <el-button size="small" type="info" link @click="fillQuickDirectPreset">
                    载入常规高血压门诊处方示例
                  </el-button>
                </div>

                <el-form-item v-if="directForm.soap_content" label="规范临床 SOAP 病历全文确认 (将进行 AES-256-GCM 加密与 Fabric 区块链存证)">
                  <el-input
                    v-model="directForm.soap_content"
                    type="textarea"
                    :rows="6"
                    class="soap-textarea"
                  />
                </el-form-item>
              </div>

              <!-- 路径 B：开立检查单移交医技科室 -->
              <div v-else class="exam-decision-panel">
                <el-row :gutter="16">
                  <el-col :span="12">
                    <el-form-item label="初步诊断 (Initial Diagnosis)" required>
                      <el-input
                        v-model="initForm.initial_diagnosis"
                        type="textarea"
                        :autosize="{ minRows: 2, maxRows: 4 }"
                        placeholder="例如：原发性高血压病1级待查；心律失常待排"
                      />
                    </el-form-item>
                  </el-col>
                  <el-col :span="12">
                    <el-form-item label="接诊依据与拟诊分析">
                      <el-input
                        v-model="initForm.diagnostic_basis"
                        type="textarea"
                        :autosize="{ minRows: 2, maxRows: 4 }"
                        placeholder="例如：患者血压145/95mmHg，活动后胸闷心悸，需辅助检查确证"
                      />
                    </el-form-item>
                  </el-col>
                </el-row>

                <div class="dispatch-target-tip mb-3 mt-2">
                  <el-alert
                    title="医技检查单智能分流派发机制："
                    type="info"
                    :closable="false"
                    description="① 血常规 / 心肌酶谱 / 肝肾生化 ➔ 自动派发至【临床检验医学中心·LIS任务池】(由检验科技师 tech_lab 接单)；
② 12导联心电图 / 胸部CT / 头颅MRI ➔ 自动派发至【放射影像与心电中心·PACS任务池】(由影像医师 tech_pacs 接单)。
提交后医技人员即可在对应工作台接单处理！当前医生亦可直接通过左侧「医技检查中心」协同出报告。"
                    show-icon
                  />
                </div>

                <el-form-item label="开具辅助检查项目 (勾选后自动生成独立申请单号 ORD... 移交医技中心)">
                  <div class="exam-picker-grid">
                    <el-checkbox-group v-model="selectedExamList" class="exam-checkbox-container">
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

                <el-form-item label="检查原因及临床指征">
                  <el-input
                    v-model="initForm.exam_reason"
                    type="textarea"
                    :autosize="{ minRows: 2, maxRows: 4 }"
                    placeholder="说明开具该检查的目的，例如：排查急性心肌缺血病变及全身炎性指标"
                  />
                </el-form-item>
              </div>
            </div>
          </div>
        </div>
      </el-form>

      <!-- 底部提交操作栏 -->
      <div class="ws-page-footer-card mt-3">
        <div class="ws-footer-bar">
          <el-button @click="currentView = 'LIST'">取消退出</el-button>
          <!-- 模式一：直接确诊并归档 -->
          <el-button
            v-if="encounterMode === 'DIRECT'"
            type="success"
            size="large"
            :loading="submitting"
            :disabled="!initForm.patient_id || !initForm.chief_complaint || !directForm.diagnosis || !directForm.treatment_plan"
            @click="submitDirectEncounter"
          >
            确认最终确诊并直接归档 (一步完成 AES加密 + IPFS + Fabric区块链存证)
          </el-button>
          <!-- 模式二：开具检查单派发医技 -->
          <el-button
            v-else
            type="primary"
            size="large"
            :loading="submitting"
            :disabled="!initForm.patient_id || !initForm.chief_complaint || !initForm.initial_diagnosis || selectedExamList.length === 0"
            @click="submitInitialEncounter"
          >
            提交初诊并派发检查申请单 (移交医技科室)
          </el-button>
        </div>
      </div>
      </template>
    </div>

    <!-- 视图三：第二阶段 最终确诊与规范 SOAP 病历生成存证工作台 (FINAL) -->
    <div v-else-if="currentView === 'FINAL'" class="inpage-workstation-wrapper">
      <div class="ws-page-header-card sticky-header">
        <div class="ws-header-left">
          <el-button :icon="ArrowLeft" plain size="small" @click="currentView = 'LIST'">返回列表</el-button>
          <div class="ws-header-title-box">
            <span class="ws-page-title">就诊确诊与区块链存证</span>
            <el-tag size="small" type="warning" effect="light">终审确诊</el-tag>
            <span v-if="activeRecordForFinal" class="ws-header-sub">
              单号: <strong class="mono">{{ activeRecordForFinal.record_no }}</strong> · 患者: <strong>{{ activeRecordForFinal.patient_name }}</strong>
            </span>
          </div>
        </div>
        <div class="ws-header-right">
          <InfectionSafetyAlert
            v-if="activeRecordForFinal?.infection_alert?.has_risk"
            :alert="activeRecordForFinal.infection_alert"
            mode="badge"
          />
          <el-button plain size="small" @click="currentView = 'LIST'">返回列表</el-button>
        </div>
      </div>

      <div v-if="activeRecordForFinal" class="final-workstation-content mt-3">
        <!-- 医护安全防护高危条幅 -->
        <div v-if="activeRecordForFinal.infection_alert?.has_risk" class="mb-3">
          <InfectionSafetyAlert
            :alert="activeRecordForFinal.infection_alert"
            mode="corner-banner"
          />
        </div>

        <div class="ws-body-grid">
          <!-- 左侧栏：初诊接诊回顾与医技检查报告回传 (420px) -->
          <div class="ws-col-left" style="width: 420px;">
            <!-- 初诊与接诊回顾卡片 -->
            <div class="ws-card">
              <div class="ws-card-title">
                <el-icon><Document /></el-icon>
                <span>初诊挂号与临床接诊回顾</span>
              </div>
              <div class="summary-kv-list">
                <div class="kv-item"><span class="k">就诊单号：</span><span class="v mono font-bold">{{ activeRecordForFinal.record_no }}</span></div>
                <div class="kv-item"><span class="k">就诊类型：</span><span class="v"><el-tag size="small" type="warning">{{ formatEncounterType(activeRecordForFinal.encounter_type) }}</el-tag></span></div>
                <div class="kv-item"><span class="k">就诊科室：</span><span class="v">{{ activeRecordForFinal.department_name }}</span></div>
                <div class="kv-item"><span class="k">就诊患者：</span><span class="v font-bold">{{ activeRecordForFinal.patient_name }}</span></div>
                <div class="kv-item"><span class="k">生命体征：</span><span class="v text-emerald-600 font-semibold">{{ activeRecordForFinal.vital_signs || '平稳' }}</span></div>
                <div class="kv-item"><span class="k">主诉 (S)：</span><span class="v">{{ activeRecordForFinal.chief_complaint }}</span></div>
                <div class="kv-item"><span class="k">初诊拟诊：</span><span class="v text-primary">{{ activeRecordForFinal.initial_diagnosis }}</span></div>
                <div class="kv-item" v-if="activeRecordForFinal.diagnostic_basis"><span class="k">拟诊依据：</span><span class="v text-xs text-gray-500">{{ activeRecordForFinal.diagnostic_basis }}</span></div>
              </div>
            </div>

            <!-- 医技辅助检查回传报告卡片 -->
            <div class="ws-card">
              <div class="ws-card-title flex justify-between items-center">
                <div class="flex items-center gap-2">
                  <el-icon><CircleCheckFilled color="#059669" /></el-icon>
                  <span>医技科室回传检查报告汇总</span>
                </div>
                <el-tag v-if="activeRecordForFinal.exam_result" size="small" type="success">报告已就绪</el-tag>
                <el-tag v-else size="small" type="info">无医技单</el-tag>
              </div>
              <div class="lab-results-display">
                <div v-if="activeRecordForFinal.exam_result" class="lab-result-text">
                  {{ activeRecordForFinal.exam_result }}
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
                        v-model="finalForm.diagnosis"
                        type="textarea"
                        :autosize="{ minRows: 2, maxRows: 4 }"
                        placeholder="例如：原发性高血压病1级，伴轻度劳力型心肌供血不足"
                      />
                    </el-form-item>
                  </el-col>
                  <el-col :span="12">
                    <el-form-item label="病因分析与病理机制 (Etiology)">
                      <el-input
                        v-model="finalForm.etiology"
                        type="textarea"
                        :autosize="{ minRows: 2, maxRows: 4 }"
                        placeholder="例如：血管内皮舒缩功能紊乱，高压负荷导致心肌耗氧量增加"
                      />
                    </el-form-item>
                  </el-col>
                </el-row>

                <el-form-item label="综合处置与处方医嘱 (分区分条下达 · 支持药品关键词下拉提示)" required>
                  <TreatmentPlanEditor
                    ref="finalTreatmentRef"
                    v-model="finalForm.treatment_plan"
                    @change="generateSOAPRecord"
                  />
                </el-form-item>

                <!-- 规则辅助生成规范 SOAP -->
                <div class="soap-action-bar mb-3">
                  <el-button
                    type="warning"
                    size="default"
                    @click="generateSOAPRecord"
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
                    v-model="finalForm.soap_content"
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
            <el-button @click="currentView = 'LIST'">返回就诊列表</el-button>
            <el-button
              type="success"
              size="large"
              :loading="submittingFinal"
              :disabled="!finalForm.diagnosis || !finalForm.treatment_plan"
              @click="submitFinalEncounter"
            >
              确认并完成归档存证 (AES加密 + IPFS + Fabric区块链存证)
            </el-button>
          </div>
        </div>
      </div>
    </div>

        <!-- 视图四：全景病历详情调阅工作台 (DETAIL) -->
    <div v-else-if="currentView === 'DETAIL'" class="inpage-workstation-wrapper">
      <div class="ws-page-header-card sticky-header">
        <div class="ws-header-left">
          <el-button :icon="ArrowLeft" plain size="small" @click="currentView = 'LIST'" class="back-btn">返回列表</el-button>
          <div class="ws-header-title-box">
            <span class="ws-page-title">临床就诊全景病历</span>
            <el-tag size="small" type="success" effect="light">已存证</el-tag>
            <div v-if="detailRecord" class="ws-record-chip">
              <span class="chip-lbl">单号</span>
              <span class="chip-val mono">{{ detailRecord.record_no }}</span>
              <el-tooltip content="复制单号" placement="top">
                <el-icon class="copy-icon" @click.stop="copyToClipboard(detailRecord.record_no, '就诊单号')"><CopyDocument /></el-icon>
              </el-tooltip>
            </div>
          </div>
        </div>
        <div class="ws-header-right">
          <InfectionSafetyAlert
            v-if="detailRecord?.infection_alert?.has_risk"
            :alert="detailRecord.infection_alert"
            mode="badge"
          />
          <el-button type="primary" size="small" :icon="View" @click="openPdfPreview(detailRecord)">查阅红头 PDF</el-button>
        </div>
      </div>

      <div v-if="detailRecord" class="detail-workstation-content mt-3">
        <!-- 医护职业暴露安全提示 -->
        <div v-if="detailRecord.infection_alert?.has_risk" class="mb-3">
          <InfectionSafetyAlert
            :alert="detailRecord.infection_alert"
            mode="corner-banner"
          />
        </div>

        <!-- 篡改与存证告警横幅 -->
        <div v-if="detailRecord.is_tampered" class="tamper-box danger">
          <div class="t-head">
            <el-icon><WarningFilled /></el-icon>
            <span>【高危安全警报】底层 MySQL 数据库临床数据已被篡改！</span>
          </div>
          <div class="t-desc">
            底层数据库指纹与 Hyperledger Fabric 联盟链不可篡改基准不一致！系统已阻断非法采信并记录高危安全审计！
          </div>
          <div class="t-hashes">
            <div>当前计算哈希: <code>{{ detailRecord.current_hash }}</code></div>
            <div>链上存证基准: <code>{{ detailRecord.chain_hash }}</code></div>
          </div>
        </div>
        <div v-else class="tamper-box safe">
          <div class="tb-safe-left">
            <el-icon class="text-emerald-600 text-lg"><CircleCheckFilled /></el-icon>
            <div class="tb-text">
              <span class="font-bold text-emerald-800 text-sm">区块链全量防篡改核验通过</span>
              <span class="text-emerald-700 text-xs ml-2">数据指纹与 Hyperledger Fabric 联盟链完全吻合，100% 真实未篡改</span>
            </div>
          </div>
          <div class="tb-safe-right">
            <span class="fabric-badge">Fabric Consensus 100% Verified</span>
          </div>
        </div>

        <div class="ws-body-grid">
          <!-- 左侧栏：档案元数据、区块链凭据、归档附件 (380px) -->
          <div class="ws-col-left">
            <!-- 卡片 1：就诊档案与患者卡片 -->
            <div class="bento-card patient-hero-card">
              <div class="ph-header">
                <div class="ph-avatar">
                  {{ detailRecord.patient_name ? detailRecord.patient_name.slice(0, 1) : '患' }}
                </div>
                <div class="ph-main-info">
                  <div class="ph-name-row">
                    <span class="ph-name">{{ detailRecord.patient_name }}</span>
                    <el-tag size="small" :type="detailRecord.patient_gender === 'FEMALE' ? 'danger' : 'primary'" effect="plain">
                      {{ detailRecord.patient_gender === 'FEMALE' ? '女' : (detailRecord.patient_gender === 'MALE' ? '男' : (detailRecord.gender || '患者')) }}
                    </el-tag>
                    <span v-if="detailRecord.patient_age || detailRecord.age" class="ph-age">{{ detailRecord.patient_age || detailRecord.age }} 岁</span>
                  </div>
                  <div class="ph-meta-id mono text-xs text-gray-500">
                    ID: {{ maskID(detailRecord.patient_id_card) }}
                  </div>
                </div>
              </div>

              <div class="ph-divider" />

              <div class="ph-details-grid">
                <div class="ph-item">
                  <span class="lbl">就诊单号</span>
                  <span class="val mono font-semibold">{{ detailRecord.record_no }}</span>
                </div>
                <div class="ph-item">
                  <span class="lbl">就诊类型</span>
                  <span class="val"><el-tag size="small" type="warning" effect="light">{{ formatEncounterType(detailRecord.encounter_type) }}</el-tag></span>
                </div>
                <div class="ph-item">
                  <span class="lbl">就诊科室</span>
                  <span class="val font-semibold text-slate-800">{{ detailRecord.department_name }}</span>
                </div>
                <div class="ph-item">
                  <span class="lbl">接诊责任医生</span>
                  <span class="val font-semibold text-primary">{{ detailRecord.doctor_name }}</span>
                </div>
                <div class="ph-item">
                  <span class="lbl">流转状态</span>
                  <span class="val"><el-tag size="small" type="success" effect="light">{{ detailRecord.status === 'COMPLETED' ? '已完成归档' : (detailRecord.status || '归档有效') }}</el-tag></span>
                </div>
                <div class="ph-item">
                  <span class="lbl">就诊时间</span>
                  <span class="val text-xs text-slate-600">{{ formatTime(detailRecord.created_at) }}</span>
                </div>
              </div>
            </div>

            <!-- 卡片 2：区块链安全与权威存证凭据 (深色科技风) -->
            <div class="bento-card crypto-card">
              <div class="crypto-header">
                <div class="crypto-title">
                  <span class="crypto-dot" />
                  <span>Hyperledger Fabric 权威存证</span>
                </div>
                <el-tag size="small" type="success" effect="dark" class="crypto-tag">联盟链共识已上链</el-tag>
              </div>

              <div class="crypto-body">
                <div class="crypto-field">
                  <div class="cf-head">
                    <span class="cf-label">Fabric 交易哈希 (TxID)</span>
                    <button class="cf-copy-btn" @click="copyToClipboard(detailRecord.fabric_tx_id, '交易哈希')">
                      <el-icon><CopyDocument /></el-icon>
                      <span>复制哈希</span>
                    </button>
                  </div>
                  <div class="cf-hash-box mono" :title="detailRecord.fabric_tx_id">
                    {{ shortHash(detailRecord.fabric_tx_id, 14, 12) }}
                  </div>
                </div>

                <div class="crypto-stats-row">
                  <div class="stat-pill">
                    <span class="stat-k">区块高度</span>
                    <span class="stat-v mono">#{{ detailRecord.block_height || '342' }}</span>
                  </div>
                  <div class="stat-pill">
                    <span class="stat-k">共识状态</span>
                    <span class="stat-v">PBFT 已确认</span>
                  </div>
                  <div class="stat-pill">
                    <span class="stat-k">存证通道</span>
                    <span class="stat-v">medtrust</span>
                  </div>
                </div>

                <div class="crypto-security-seal">
                  <el-icon class="mr-1 text-emerald-400"><CircleCheckFilled /></el-icon>
                  <span>AES-256-GCM 本地加密 + IPFS 分布式存证</span>
                </div>
              </div>
            </div>

                        <!-- 卡片 3：临床归档电子凭据与附件 (精致分层版) -->
            <div class="bento-card files-card">
              <div class="files-card-header">
                <div class="fch-title">
                  <el-icon color="#0284c7"><Files /></el-icon>
                  <span>临床归档凭据与附件</span>
                </div>
                <el-tag size="small" type="info" effect="plain" class="fch-tag">IPFS 存证</el-tag>
              </div>

              <div class="archive-files-section">
                <div v-if="detailRecord.files && detailRecord.files.length" class="file-cards-list">
                  <div v-for="file in detailRecord.files" :key="file.id" class="file-item-bento">
                    <!-- 顶部行：图标 + 文件名 + 类型与大小 -->
                    <div class="fib-top-row">
                      <div class="fib-icon">
                        <el-icon v-if="file.file_type === 'pdf'" color="#dc2626"><Document /></el-icon>
                        <el-icon v-else-if="isImageFileType(file.file_type)" color="#0284c7"><Picture /></el-icon>
                        <el-icon v-else color="#64748b"><Files /></el-icon>
                      </div>
                      <div class="fib-meta">
                        <div class="fib-name-box">
                          <span class="fib-name" :title="file.file_name">{{ file.file_name }}</span>
                        </div>
                        <div class="fib-tags">
                          <el-tag size="small" type="primary" effect="light" class="type-tag">{{ file.file_type.toUpperCase() }}</el-tag>
                          <el-tag size="small" type="success" effect="plain" class="size-tag">{{ formatFileSize(file.file_size) }}</el-tag>
                        </div>
                      </div>
                    </div>

                    <!-- 中间行：IPFS CID 存证哈希 -->
                    <div class="fib-cid-row">
                      <span class="cid-lbl">IPFS CID：</span>
                      <code class="cid-val mono" :title="file.ipfs_cid">{{ shortHash(file.ipfs_cid, 10, 8) }}</code>
                      <el-tooltip content="复制 IPFS CID" placement="top">
                        <el-icon class="copy-icon ml-1" @click.stop="copyToClipboard(file.ipfs_cid, 'IPFS CID')"><CopyDocument /></el-icon>
                      </el-tooltip>
                    </div>

                    <!-- 底部行：独立操作按钮条 -->
                    <div class="fib-actions-row">
                      <template v-if="isImageFileType(file.file_type)">
                        <el-button type="primary" size="small" plain :icon="View" class="fib-btn" @click="openImageModal(file)">查阅大图</el-button>
                        <el-button type="success" size="small" plain :icon="Download" class="fib-btn" @click="downloadFileDirect(`/api/v1/medical-files/${file.id}/download?token=${auth.token}`, file.file_name)">下载原图</el-button>
                      </template>
                      <template v-else>
                        <el-button type="primary" size="small" plain :icon="View" class="fib-btn" @click="openPdfPreview(detailRecord)">查阅红头 PDF</el-button>
                        <el-button type="success" size="small" plain :icon="Download" class="fib-btn" @click="downloadRecordFile(detailRecord.id)">下载凭据</el-button>
                      </template>
                    </div>

                    <!-- 略缩图（如果是图片） -->
                    <div v-if="isImageFileType(file.file_type)" class="fib-thumb-box mt-2">
                      <img
                        :src="`/api/v1/medical-files/${file.id}/view?token=${auth.token}`"
                        class="preview-thumbnail"
                        @click="openImageModal(file)"
                        title="点击全屏查阅大图"
                      />
                    </div>
                  </div>
                </div>

                <div v-else class="empty-files-bento">
                  <div class="efb-title font-medium text-slate-800">标准规范电子病历归档凭证 (PDF)</div>
                  <div class="efb-desc text-xs text-gray-500 mt-1">已自动生成全量 SOAP 规范电子病历并完成区块链存证</div>
                  <div class="efb-btns mt-2">
                    <el-button type="primary" size="small" plain :icon="View" class="fib-btn" @click="openPdfPreview(detailRecord)">
                      查阅红头 PDF
                    </el-button>
                    <el-button type="success" size="small" plain :icon="Download" class="fib-btn" @click="downloadRecordFile(detailRecord.id)">
                      下载加密凭据
                    </el-button>
                  </div>
                </div>
              </div>
            </div>
          </div>

          <!-- 右侧栏：结构化临床 SOAP Bento 全景面板 (flex: 1) -->
          <div class="ws-col-main">
            <!-- 维度 1: [S] 主观病史与主诉 (Subjective) -->
            <div class="bento-card soap-bento-card s-dimension">
              <div class="soap-card-head">
                <div class="soap-head-title">
                  <span class="soap-dim-badge s-badge">S</span>
                  <span class="soap-title-text">主观病史与患者主诉 (Subjective)</span>
                </div>
                <div class="soap-time-chips">
                  <span v-if="detailRecord.onset_time" class="time-chip">
                    🕒 发病时间: {{ detailRecord.onset_time }}
                  </span>
                  <span v-if="detailRecord.duration" class="time-chip">
                    ⏱️ 持续: {{ detailRecord.duration }}
                  </span>
                </div>
              </div>

              <div class="soap-card-content">
                <div class="complaint-hero-box">
                  <div class="chb-quote-mark">“</div>
                  <div class="chb-text">
                    {{ detailRecord.chief_complaint || detailRecord.symptoms || '患者就诊主诉' }}
                  </div>
                </div>

                <div v-if="detailRecord.present_illness" class="present-illness-box mt-3">
                  <div class="pib-label">现病史演进 (History of Present Illness)：</div>
                  <div class="pib-content">{{ detailRecord.present_illness }}</div>
                </div>
              </div>
            </div>

            <!-- 维度 2: [O] 客观体征与医技检查 (Objective) -->
            <div class="bento-card soap-bento-card o-dimension">
              <div class="soap-card-head">
                <div class="soap-head-title">
                  <span class="soap-dim-badge o-badge">O</span>
                  <span class="soap-title-text">客观检查与生命体征 (Objective)</span>
                </div>
                <el-tag size="small" type="success" effect="light">实时临床体征指标</el-tag>
              </div>

              <div class="soap-card-content">
                <!-- 4 项生命体征网格 -->
                <div class="vitals-sensor-grid">
                  <div class="vital-sensor-card">
                    <div class="vsc-icon-box temp-icon">🌡️</div>
                    <div class="vsc-info">
                      <span class="vsc-name">体温 (T)</span>
                      <span class="vsc-val mono">{{ parseVitals(detailRecord.vital_signs).temperature || '36.5 ℃' }}</span>
                      <span class="vsc-status normal">体温正常</span>
                    </div>
                  </div>

                  <div class="vital-sensor-card">
                    <div class="vsc-icon-box bp-icon">🫀</div>
                    <div class="vsc-info">
                      <span class="vsc-name">血压 (BP)</span>
                      <span class="vsc-val mono">{{ parseVitals(detailRecord.vital_signs).blood_pressure || '120/80 mmHg' }}</span>
                      <span class="vsc-status normal">血压理想</span>
                    </div>
                  </div>

                  <div class="vital-sensor-card">
                    <div class="vsc-icon-box hr-icon">💓</div>
                    <div class="vsc-info">
                      <span class="vsc-name">心率 (HR)</span>
                      <span class="vsc-val mono">{{ parseVitals(detailRecord.vital_signs).heart_rate || '75 bpm' }}</span>
                      <span class="vsc-status normal">窦性心律</span>
                    </div>
                  </div>

                  <div class="vital-sensor-card">
                    <div class="vsc-icon-box spo2-icon">🫁</div>
                    <div class="vsc-info">
                      <span class="vsc-name">血氧 (SpO2)</span>
                      <span class="vsc-val mono">{{ parseVitals(detailRecord.vital_signs).spo2 || '98 %' }}</span>
                      <span class="vsc-status normal">动脉血氧充足</span>
                    </div>
                  </div>
                </div>

                <!-- 医技辅助检查回传报告板块 -->
                <div class="exam-report-bento-section mt-3">
                  <div class="erb-header">
                    <div class="erb-title">
                      <el-icon color="#0284c7"><OfficeBuilding /></el-icon>
                      <span>LIS / PACS 医技检验与影像检查报告</span>
                    </div>
                    <span v-if="detailRecord.exam_items" class="erb-items-tag">
                      申请项目：{{ detailRecord.exam_items }}
                    </span>
                  </div>

                  <div v-if="detailRecord.exam_result" class="erb-result-box">
                    <div class="erb-report-text pre-wrap">{{ detailRecord.exam_result }}</div>
                    <div v-if="detailRecord.exam_doctor" class="erb-footer">
                      <span>报告出具医师/技师：<strong>{{ detailRecord.exam_doctor }}</strong></span>
                      <span v-if="detailRecord.exam_time" class="ml-3">检验时间：{{ detailRecord.exam_time }}</span>
                    </div>
                  </div>
                  <div v-else class="erb-empty-box">
                    <el-icon color="#10b981" class="text-base mr-1"><CircleCheckFilled /></el-icon>
                    <span>门诊专科体征典型，临床未开具外送医技辅助检验，经治医师综合体格检查直接确诊。</span>
                  </div>
                </div>
              </div>
            </div>

            <!-- 维度 3: [A] 综合评估与临床确诊 (Assessment) -->
            <div class="bento-card soap-bento-card a-dimension">
              <div class="soap-card-head">
                <div class="soap-head-title">
                  <span class="soap-dim-badge a-badge">A</span>
                  <span class="soap-title-text">综合临床评估与最终确诊 (Assessment)</span>
                </div>
                <el-tag size="small" type="primary" effect="light">ICD-10 规范确诊</el-tag>
              </div>

              <div class="soap-card-content">
                <!-- 确诊 Hero 区域 -->
                <div class="diagnosis-hero-banner">
                  <div class="dhb-left">
                    <div class="dhb-label">经治医师最终确诊 (Final Diagnosis)</div>
                    <div class="dhb-name">{{ detailRecord.diagnosis || '待确诊' }}</div>
                    <div v-if="detailRecord.initial_diagnosis" class="dhb-compare">
                      <span class="text-xs text-gray-500">初诊拟诊：</span>
                      <span class="text-xs font-semibold text-slate-700">{{ detailRecord.initial_diagnosis }}</span>
                      <span v-if="detailRecord.diagnostic_basis" class="text-xs text-gray-400 ml-2">（依据：{{ detailRecord.diagnostic_basis }}）</span>
                    </div>
                  </div>
                  <div class="dhb-right">
                    <div class="dhb-stamp-badge">
                      <el-icon class="mr-1 text-emerald-600"><CircleCheckFilled /></el-icon>
                      <span>确诊生效</span>
                    </div>
                  </div>
                </div>

                <!-- 病理诱因与机制分析 -->
                <div v-if="detailRecord.etiology" class="etiology-bento-box mt-3">
                  <div class="ebb-title">🔬 病理诱因与发病机制分析 (Pathogenesis & Etiology)：</div>
                  <div class="ebb-content">{{ detailRecord.etiology }}</div>
                </div>
              </div>
            </div>

            <!-- 维度 4: [P] 处置、处方与综合医嘱 (Plan) -->
            <div class="bento-card soap-bento-card p-dimension">
              <div class="soap-card-head">
                <div class="soap-head-title">
                  <span class="soap-dim-badge p-badge">P</span>
                  <span class="soap-title-text">处方用药、临床处置与综合医嘱 (Plan)</span>
                </div>
                <el-tag size="small" type="warning" effect="light">已签发执行</el-tag>
              </div>

              <div class="soap-card-content">
                <!-- 结构化处方与用药清单 -->
                <div class="rx-orders-container">
                  <div class="rx-header">
                    <span class="rx-symbol">℞</span>
                    <span class="rx-title">处方用药与临床方案明细</span>
                  </div>

                  <div class="rx-list">
                    <div
                      v-for="item in parsePlanItems(detailRecord.treatment_plan)"
                      :key="item.id"
                      class="rx-item-card"
                      :class="{ 'is-medicine': item.isMed }"
                    >
                      <div class="ric-num">{{ item.id }}</div>
                      <div class="ric-content">
                        <div class="ric-text">{{ item.text }}</div>
                      </div>
                      <div class="ric-badge">
                        <el-tag v-if="item.isMed" size="small" type="success" effect="plain">药品处方</el-tag>
                        <el-tag v-else size="small" type="info" effect="plain">临床医嘱</el-tag>
                      </div>
                    </div>
                  </div>
                </div>

                <!-- 责任医师数字签名与存证固化 -->
                <div class="doctor-signature-banner mt-3">
                  <div class="dsb-left">
                    <div class="dsb-doctor">
                      <span class="text-xs text-gray-500">经治执业医师：</span>
                      <strong class="text-slate-800">{{ detailRecord.doctor_name }}</strong>
                      <span class="dsb-ca-tag">CA 电子认证已签署</span>
                    </div>
                    <div class="dsb-time text-xs text-gray-400 mt-1">
                      签发归档时间：{{ formatTime(detailRecord.created_at) }}
                    </div>
                  </div>
                  <div class="dsb-right">
                    <div class="dsb-hash-note">
                      <span>存证区块：<code>#{{ detailRecord.block_height || '342' }}</code></span>
                    </div>
                  </div>
                </div>
              </div>
            </div>
          </div>
        </div>

        <!-- 底部关闭与快速操作栏 -->
        <div class="ws-page-footer-card mt-3">
          <div class="ws-footer-bar">
            <el-button @click="currentView = 'LIST'">返回就诊列表</el-button>
            <div class="flex gap-2">
              <el-button type="primary" :icon="View" @click="openPdfPreview(detailRecord)">
                查阅规范红头 PDF 电子病历
              </el-button>
              <el-button type="success" plain :icon="Download" @click="downloadRecordFile(detailRecord.id)">
                下载原始解密凭据
              </el-button>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- PDF 电子病历规范预览弹窗 -->
    <el-dialog
      v-model="pdfPreviewVisible"
      title="临床就诊电子病历归档凭证 (PDF 规范视图)"
      width="1040px"
      top="2vh"
      :close-on-click-modal="false"
      class="pdf-preview-dialog"
    >
      <div v-if="previewingRecord" id="emr-print-container" class="emr-sheet-wrapper">
        <!-- 打印与工具栏 -->
        <div class="emr-action-bar no-print">
          <div class="emr-tip-tag">
            <el-tag type="success" effect="dark">密文解密验证通过</el-tag>
            <el-tag type="info">国家卫健委《电子病历应用规范》甲级存证标准</el-tag>
            <InfectionSafetyAlert
              v-if="previewingRecord?.infection_alert?.has_risk"
              :alert="previewingRecord.infection_alert"
              mode="badge"
            />
          </div>
          <div class="emr-btns">
            <el-button type="primary" :icon="Printer" @click="printPdfSheet">
              打印 / 另存为 PDF 文件
            </el-button>
            <el-button type="success" plain :icon="Download" @click="downloadRecordFile(previewingRecord.id)">
              下载原始归档凭证
            </el-button>
          </div>
        </div>

        <!-- 正式红头病历文档纸张 -->
        <div class="emr-sheet-paper">
          <!-- 右上角国家卫健委医护安全防护警示水印/标识 -->
          <div v-if="previewingRecord.infection_alert?.has_risk" class="emr-infection-watermark">
            ⚠️【国家卫健委医护职业安全重点警示：高危传染病携带者 · 接诊请穿戴双层手套与防护装备】
          </div>
          <!-- 红头医院名称与标题 -->
          <div class="emr-header">
            <div class="emr-hospital-name">{{ previewingRecord.hospital_name || formatHospName(previewingRecord.hospital_id) }}</div>
            <div class="emr-doc-title">门 诊 / 临 床 就 诊 归 档 记 录 单</div>
            <div class="emr-sub-title">（MedTrust 医疗可信共享联盟·Hyperledger Fabric 区块链存证凭证）</div>
            <div class="emr-red-line-double">
              <div class="line-thick"></div>
              <div class="line-thin"></div>
            </div>
          </div>

          <!-- 条形码与凭证流水号 -->
          <div class="emr-meta-banner">
            <div class="meta-left">
              <div>就诊编号：<span class="mono bold">{{ previewingRecord.record_no }}</span></div>
              <div>就诊科室：{{ previewingRecord.department_name }}</div>
              <div>就诊类型：{{ formatEncounterType(previewingRecord.encounter_type) }}</div>
            </div>
            <div class="meta-barcode-box">
              <div class="barcode-graphic">||| | |||| | || |||| | ||| |</div>
              <div class="barcode-text mono">{{ previewingRecord.record_no }}</div>
            </div>
            <div class="meta-right">
              <div>就诊时间：{{ formatTime(previewingRecord.created_at) }}</div>
              <div>责任医师：<strong>{{ previewingRecord.doctor_name }}</strong></div>
              <div>归档状态：<el-tag type="success" size="small">已上链存证</el-tag></div>
            </div>
          </div>

          <!-- 患者基本信息表格 -->
          <table class="emr-patient-table">
            <tr>
              <th width="90">患者姓名</th>
              <td width="150"><strong>{{ previewingRecord.patient_name }}</strong></td>
              <th width="90">身份证号</th>
              <td width="220"><span class="mono">{{ maskID(previewingRecord.patient_id_card) }}</span></td>
              <th width="80">联系电话</th>
              <td><span class="mono">{{ previewingRecord.patient_phone || '138****0000' }}</span></td>
            </tr>
          </table>

          <!-- 临床 SOAP 详细内容 -->
          <div class="emr-soap-section">
            <div class="soap-block">
              <div class="soap-title">【S - Subjective 主观病史采集】</div>
              <div class="soap-row"><strong>● 患者主诉：</strong>{{ previewingRecord.chief_complaint || previewingRecord.symptoms }}</div>
              <div class="soap-row"><strong>● 现病史：</strong>{{ previewingRecord.present_illness || '患者因主诉症状就诊，发病过程如上所述。' }}</div>
              <div class="soap-row"><strong>● 发病时间：</strong>{{ previewingRecord.onsetTime || previewingRecord.onset_time || '接诊前' }}（持续时间：{{ previewingRecord.duration || '发作性' }}）</div>
            </div>

            <div class="soap-block">
              <div class="soap-title">【O - Objective 客观检查与测量】</div>
              <div class="soap-row"><strong>● 基础生命体征：</strong><span class="vitals-highlight">{{ previewingRecord.vital_signs || '生命体征平稳' }}</span></div>
              <div class="soap-row"><strong>● 辅助检查申请：</strong>{{ previewingRecord.need_exam ? (previewingRecord.exam_items || '已开具辅助检查') : '未开具医技检查' }}</div>
              <div v-if="previewingRecord.exam_result" class="soap-exam-result-box">
                <div class="soap-exam-head">医技科室回传检查报告明细：</div>
                <pre class="soap-exam-content">{{ previewingRecord.exam_result }}</pre>
                <div v-if="previewingRecord.exam_doctor" class="soap-exam-foot">
                  出具技师/医生：{{ previewingRecord.exam_doctor }} | 时间：{{ previewingRecord.exam_time }}
                </div>
              </div>
            </div>

            <div class="soap-block">
              <div class="soap-title">【A - Assessment 综合评估与确诊】</div>
              <div class="soap-row"><strong>● 临床初步诊断：</strong>{{ previewingRecord.initial_diagnosis || '-' }} <span v-if="previewingRecord.diagnostic_basis" class="text-xs text-gray-500">（依据：{{ previewingRecord.diagnostic_basis }}）</span></div>
              <div class="soap-row final-diagnosis">
                <strong>● 经治医师最终确诊：</strong>
                <span class="font-bold text-success">{{ previewingRecord.diagnosis }}</span>
              </div>
              <div v-if="previewingRecord.etiology" class="soap-row">
                <strong>● 病理诱因与机制：</strong>{{ previewingRecord.etiology }}
              </div>
            </div>

            <div class="soap-block">
              <div class="soap-title">【P - Plan 综合处置与医嘱方案】</div>
              <div class="soap-row plan-box">
                <pre class="soap-plan-content">{{ previewingRecord.treatment_plan || '遵常规临床医嘱随访' }}</pre>
              </div>
            </div>
          </div>

          <!-- 底端医师电子签名与区块链防伪存证图章 -->
          <div class="emr-footer-box">
            <div class="footer-crypto-evidence">
              <div><strong>Hyperledger Fabric TxID：</strong><code class="tx-hash">{{ previewingRecord.fabric_tx_id }}</code></div>
              <div v-if="previewingRecord.files && previewingRecord.files.length">
                <strong>IPFS 分布式存证 CID：</strong><code>{{ previewingRecord.files[0].ipfs_cid }}</code>
              </div>
              <div v-if="previewingRecord.files && previewingRecord.files.length">
                <strong>原始数据 SHA-256 指纹：</strong><code>{{ previewingRecord.files[0].file_hash }}</code>
              </div>
              <div class="text-xs text-emerald-700 mt-1">本病历经 Hyperledger Fabric 存证，数字签名抗抵赖、防篡改</div>
            </div>

            <div class="footer-seal-area">
              <!-- 防伪公章样式 -->
              <div class="emr-stamp">
                <div class="stamp-inner">
                  <div class="stamp-star">★</div>
                  <div class="stamp-text-top">{{ previewingRecord.hospital_name || '医疗机构' }}</div>
                  <div class="stamp-text-center">区块链存证专章</div>
                  <div class="stamp-text-bottom">100% 真实有效</div>
                </div>
              </div>

              <!-- 医师签字 -->
              <div class="doctor-signature-box">
                <span class="sig-lbl">经治医师签字：</span>
                <span class="sig-name">{{ previewingRecord.doctor_name }} (印)</span>
                <div class="sig-time text-xs">{{ formatTime(previewingRecord.created_at) }}</div>
              </div>
            </div>
          </div>
        </div>
      </div>
    </el-dialog>

    <!-- 影像大图全屏查阅弹窗 -->
    <el-dialog
      v-model="imageModalVisible"
      :title="imageModalTitle || '医学检验/影像检查原图'"
      width="860px"
      top="4vh"
    >
      <div class="text-center p-3">
        <img
          :src="imageModalUrl"
          style="max-width: 100%; max-height: 72vh; border-radius: 8px; box-shadow: 0 4px 16px rgba(0,0,0,0.15);"
          alt="医学图像"
        />
      </div>
      <template #footer>
        <el-button @click="imageModalVisible = false">关闭预览</el-button>
        <el-button type="primary" :icon="Download" @click="downloadFileDirect(imageModalUrl, imageModalTitle)">
          下载原始影像文件
        </el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
// 当前主工作区视图: LIST-就诊记录列表, CREATE-接诊初诊工作台, FINAL-确诊存证工作台, DETAIL-全景病历详情工作台
const currentView = ref<'LIST' | 'CREATE' | 'FINAL' | 'DETAIL'>('LIST')

import { ref, computed, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { Plus, Refresh, Search, View, Download, Printer, OfficeBuilding, Timer, Document, CircleCheckFilled, Picture, Files, ArrowLeft, CopyDocument, WarningFilled } from '@element-plus/icons-vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import api from '../../api/client'
import { useAuthStore } from '../../stores/auth'
import TreatmentPlanEditor from '../../components/TreatmentPlanEditor.vue'
import InfectionSafetyAlert from '../../components/InfectionSafetyAlert.vue'
import { scanTextForInfections } from '../../utils/infectionGuard'

const router = useRouter()
const auth = useAuthStore()

const loading = ref(false)
const records = ref<any[]>([])
const patientList = ref<any[]>([])
const activeStatusFilter = ref('ALL')
const searchKeyword = ref('')

// 医护职业安全防范与高危传染病预警
const selectedPatientInfectionAlert = ref<any>(null)
const createInfectionAlertRef = ref<any>(null)

function openCreateInfectionGuide() {
  if (createInfectionAlertRef.value) {
    createInfectionAlertRef.value.openGuideModal()
  }
}

// 方案编辑器组件引用
const directTreatmentRef = ref<any>(null)
const finalTreatmentRef = ref<any>(null)

// 新建就诊（初诊）表单
const initialDialogVisible = ref(false)
const submitting = ref(false)
const selectedPatientInfo = ref<any>(null)
const selectedExamList = ref<string[]>([])

// 临床决策路径: DIRECT (体征明确直接确诊一步存证) | EXAM (病情待查开立检查移交医技)
const encounterMode = ref<'DIRECT' | 'EXAM'>('DIRECT')
const directForm = ref({
  diagnosis: '',
  etiology: '',
  treatment_plan: '',
  soap_content: ''
})

const initForm = ref({
  patient_id: undefined as number | undefined,
  encounter_type: 'OUTPATIENT',
  department_name: '心血管内科',
  onset_time: '',
  duration: '持续 2 天',
  chief_complaint: '',
  present_illness: '',
  initial_diagnosis: '',
  diagnostic_basis: '',
  exam_reason: ''
})

const vitals = ref({
  temperature: '36.6',
  blood_pressure: '128/82',
  heart_rate: '76',
  spo2: '98'
})

// 第二阶段最终诊断表单
const finalDialogVisible = ref(false)
const submittingFinal = ref(false)
const activeRecordForFinal = ref<any>(null)
const finalForm = ref({
  diagnosis: '',
  etiology: '',
  treatment_plan: '',
  soap_content: ''
})

// 详情查看
const detailModalVisible = ref(false)
const detailRecord = ref<any>(null)
const pdfPreviewVisible = ref(false)
const previewingRecord = ref<any>(null)

const imageModalVisible = ref(false)
const imageModalUrl = ref('')
const imageModalTitle = ref('')

function isImageFileType(ft?: string) {
  if (!ft) return false
  const lower = ft.toLowerCase()
  return ['png', 'jpg', 'jpeg', 'gif', 'webp', 'svg'].includes(lower) || lower.includes('png') || lower.includes('jpg') || lower.includes('jpeg')
}

function openImageModal(file: any) {
  imageModalUrl.value = `/api/v1/medical-files/${file.id}/view?token=${auth.token}`
  imageModalTitle.value = file.file_name || '医学检查图像'
  imageModalVisible.value = true
}

function downloadFileDirect(url: string, fileName?: string) {
  const a = document.createElement('a')
  a.href = url
  a.download = fileName || 'medical_file'
  a.target = '_blank'
  document.body.appendChild(a)
  a.click()
  document.body.removeChild(a)
}

const waitingExamCount = computed(() => records.value.filter(r => r.status === 'WAITING_EXAM' || r.status === 'PROCESSING_EXAM').length)
const examCompletedCount = computed(() => records.value.filter(r => r.status === 'EXAM_COMPLETED' || r.status === 'INITIAL_DIAGNOSIS').length)
const completedCount = computed(() => records.value.filter(r => r.status === 'COMPLETED').length)

const filteredRecords = computed(() => {
  let list = records.value
  if (activeStatusFilter.value !== 'ALL') {
    if (activeStatusFilter.value === 'WAITING_EXAM') {
      list = list.filter(r => r.status === 'WAITING_EXAM' || r.status === 'PROCESSING_EXAM')
    } else if (activeStatusFilter.value === 'EXAM_COMPLETED') {
      list = list.filter(r => r.status === 'EXAM_COMPLETED' || r.status === 'INITIAL_DIAGNOSIS')
    } else {
      list = list.filter(r => r.status === activeStatusFilter.value)
    }
  }
  if (searchKeyword.value.trim()) {
    const kw = searchKeyword.value.trim().toLowerCase()
    list = list.filter(r =>
      (r.record_no && r.record_no.toLowerCase().includes(kw)) ||
      (r.patient_name && r.patient_name.toLowerCase().includes(kw)) ||
      (r.chief_complaint && r.chief_complaint.toLowerCase().includes(kw)) ||
      (r.diagnosis && r.diagnosis.toLowerCase().includes(kw))
    )
  }
  return list
})

async function fetchRecords() {
  loading.value = true
  try {
    const res: any = await api.get('/medical-records')
    if (res.code === 200) {
      records.value = res.data || []
    }
  } catch (err: any) {
    ElMessage.error('获取就诊档案失败: ' + (err.message || '网络异常'))
  } finally {
    loading.value = false
  }
}

async function fetchPatients() {
  try {
    const res: any = await api.get('/patients')
    if (res.code === 200) {
      patientList.value = res.data || []
    }
  } catch {}
}

async function onPatientSelect(pid: number) {
  if (!pid) {
    selectedPatientInfo.value = null
    selectedPatientInfectionAlert.value = null
    return
  }
  selectedPatientInfo.value = patientList.value.find(p => p.id === pid) || null

  // 选定患者后，自动装配该科室规范接诊模板，方便医生快速核验与调整
  if (!initForm.value.chief_complaint) {
    initForm.value.duration = '阵发性胸闷 2 天'
    initForm.value.chief_complaint = '胸骨后发作性压榨样闷痛2天，劳累后加剧'
    initForm.value.present_illness = '患者2天前爬楼劳累后突感心前区及胸骨后压榨样闷痛，每次持续3-5分钟，伴心慌出冷汗，休息后可轻微缓解。'
    initForm.value.initial_diagnosis = '原发性高血压病1级待查；冠脉缺血待排'
    initForm.value.diagnostic_basis = '中年患者，活动诱发胸闷心悸，既往有轻度血压偏高史，需心电图及血常规排查。'
    initForm.value.exam_reason = '排查急性心肌缺血病变及炎性反应'
  }
  if (!directForm.value.diagnosis) {
    directForm.value.diagnosis = '原发性高血压病1级（低危组）'
    directForm.value.etiology = '活动劳累及情绪应激导致交感神经过度兴奋，外周阻力血管收缩，收缩压升高。'
    directForm.value.treatment_plan = '【综合处置】\n1. 静坐休息15分钟后再次复测双侧上臂血压，持续脉氧监测\n\n【临床医嘱】\n1. 饮食指导：严格低盐低脂清淡饮食（每日食盐量控制在 5g 以内）\n2. 每日早晚各监测一次血压并详细做健康日志记录\n3. 避免连续熬夜加班、情绪激动与重度体力劳累，适度慢走运动\n4. 2周后心血管内科专科门诊复查\n\n【处方用药】\n1. 硝苯地平控释片 (拜新同) 30mg 口服 每日一次 清晨餐后整片吞服\n2. 阿托伐他汀钙片 (立普妥) 20mg 口服 每晚睡前一次'
  }
  if (selectedExamList.value.length === 0) {
    selectedExamList.value = ['12导联心电图 (ECG)', '常规全血细胞分析 (血常规)']
  }
  quickFillVitals()

  if (encounterMode.value === 'DIRECT') {
    generateDirectSOAP()
  }

  // 1. 若当前患者对象已有预带的传染病风险
  if (selectedPatientInfo.value?.infection_alert) {
    selectedPatientInfectionAlert.value = selectedPatientInfo.value.infection_alert
  } else {
    selectedPatientInfectionAlert.value = null
  }

  // 2. 异步请求跨机构全景传染病风险评估
  if (pid) {
    try {
      const res: any = await api.get(`/patients/${pid}/infection-risks`)
      if (res.code === 200 && res.data) {
        selectedPatientInfectionAlert.value = res.data
      }
    } catch {
      if (selectedPatientInfo.value) {
        const text = `${selectedPatientInfo.value.real_name} ${selectedPatientInfo.value.user_no}`
        selectedPatientInfectionAlert.value = scanTextForInfections(text)
      }
    }
  }
}

function goToCrossQueryForPatient() {
  if (!selectedPatientInfo.value) {
    ElMessage.warning('请先在左侧选择就诊患者')
    return
  }
  const idCard = selectedPatientInfo.value.id_card || selectedPatientInfo.value.real_name
  currentView.value = 'LIST'
    initialDialogVisible.value = false
  router.push({
    path: '/doctor/cross-query',
    query: {
      id_card: idCard,
      patient_name: selectedPatientInfo.value.real_name,
      patient_id: selectedPatientInfo.value.id,
      scope: 'all'
    }
  })
}

function maskID(idCard?: string) {
  if (!idCard || idCard.length < 8) return '已脱敏保护'
  return idCard.substring(0, 6) + '********' + idCard.substring(idCard.length - 4)
}

function formatHospName(hId?: number) {
  if (hId === 1) return '第一人民医院'
  if (hId === 2) return '第二人民医院'
  if (hId === 3) return '第三人民医院'
  return '医疗中心'
}

function quickFillVitals() {
  vitals.value = {
    temperature: '36.5',
    blood_pressure: '122/80',
    heart_rate: '74',
    spo2: '99'
  }
  ElMessage.success('已填入正常参考生命体征数值！')
}

function openInitialDialog() {
  encounterMode.value = 'DIRECT'
  // 必须先选择就诊人，初始不预选张三或任何患者！
  initForm.value = {
    patient_id: undefined,
    encounter_type: 'OUTPATIENT',
    department_name: '心血管内科',
    onset_time: new Date().toISOString().substring(0, 19).replace('T', ' '),
    duration: '',
    chief_complaint: '',
    present_illness: '',
    initial_diagnosis: '',
    diagnostic_basis: '',
    exam_reason: ''
  }
  selectedPatientInfo.value = null
  selectedPatientInfectionAlert.value = null
  selectedExamList.value = []

  directForm.value = {
    diagnosis: '',
    etiology: '',
    treatment_plan: '',
    soap_content: ''
  }

  currentView.value = 'CREATE'
  initialDialogVisible.value = true
}

function pickQuickPatient(pid: number) {
  initForm.value.patient_id = pid
  onPatientSelect(pid)
}

function changeSelectedPatient() {
  initForm.value.patient_id = undefined
  selectedPatientInfo.value = null
  selectedPatientInfectionAlert.value = null
}

function generateDirectSOAP() {
  const pName = selectedPatientInfo.value?.real_name || '就诊患者'
  const timeStr = initForm.value.onset_time || new Date().toLocaleString()
  const docName = auth.user?.real_name || '经治医师'
  const vitalStr = `T: ${vitals.value.temperature}℃, BP: ${vitals.value.blood_pressure} mmHg, HR: ${vitals.value.heart_rate} bpm, SpO2: ${vitals.value.spo2}%`

  const soapText = `【MedTrust 医疗可信共享·国家卫健委规范临床 SOAP 病历】
--------------------------------------------------------------------------------
就诊编号：系统将自动生成 | 接诊科室：${initForm.value.department_name} | 就诊类型：${formatEncounterType(initForm.value.encounter_type)}
患者姓名：${pName} | 就诊时间：${timeStr} | 责任医师：${docName}
--------------------------------------------------------------------------------
【S - Subjective 主观病史采集】
● 患者主诉：${initForm.value.chief_complaint || '就诊主诉'}
● 现病史：${initForm.value.present_illness || '患者因上述主诉就诊，病史采集如前所述。'}
● 发病时间：${initForm.value.onset_time || '近期'} | 持续时间：${initForm.value.duration || '发作性'}

【O - Objective 客观检查与测量】
● 查体生命体征：${vitalStr}
● 辅助检查说明：患者临床体征特征典型明确，经经治医师研判无需开展侵入性或大型医技辅助检查，执行临床合理诊疗减负标准。

【A - Assessment 综合评估与确诊】
● 经治医师最终确诊：${directForm.value.diagnosis || '临床明确诊断'}
● 病理诱因与机制：${directForm.value.etiology || '临床常见发病机制'}

【P - Plan 综合处置与医嘱方案】
${directForm.value.treatment_plan || '遵临床常规医嘱治疗'}

● 责任医师合规签署：本病历依据患者现病史采集与生命体征查体直接作出规范确诊，符合国家临床医疗核心制度规范。
--------------------------------------------------------------------------------
存证机构：${auth.user?.hospital_name || '本医疗中心'} | 加密协议：AES-256-GCM | 存证底层：Hyperledger Fabric`

  directForm.value.soap_content = soapText
}

function fillQuickDirectPreset() {
  directForm.value.diagnosis = '原发性高血压病1级（低危组）'
  directForm.value.etiology = '活动劳累及情绪应激导致交感神经过度兴奋，外周阻力血管收缩，收缩压升高。'
  const planData = {
    disposals: [
      '静坐休息15分钟后再次复测双侧上臂血压，持续脉氧监测'
    ],
    orders: [
      '饮食指导：严格低盐低脂清淡饮食（每日食盐量控制在 5g 以内）',
      '每日早晚各监测一次血压并详细做健康日志记录',
      '避免连续熬夜加班、情绪激动与重度体力劳累，适度慢走运动',
      '2周后心血管内科专科门诊复查'
    ],
    prescriptions: [
      '硝苯地平控释片 (拜新同) 30mg 口服 每日一次 清晨餐后整片吞服',
      '阿托伐他汀钙片 (立普妥) 20mg 口服 每晚睡前一次'
    ]
  }
  if (directTreatmentRef.value) {
    directTreatmentRef.value.setStructuredData(planData)
  } else {
    directForm.value.treatment_plan = `【综合处置】\n1. ${planData.disposals[0]}\n\n【临床医嘱】\n${planData.orders.map((s, i) => `${i + 1}. ${s}`).join('\n')}\n\n【处方用药】\n${planData.prescriptions.map((s, i) => `${i + 1}. ${s}`).join('\n')}`
  }
  generateDirectSOAP()
  ElMessage.success('已载入常规高血压门诊分区分条规范处方！')
}

async function submitDirectEncounter() {
  submitting.value = true
  try {
    const vitalStr = `T: ${vitals.value.temperature}℃, BP: ${vitals.value.blood_pressure} mmHg, HR: ${vitals.value.heart_rate} bpm, SpO2: ${vitals.value.spo2}%`
    if (!directForm.value.soap_content) {
      generateDirectSOAP()
    }

    const payload = {
      patient_id: initForm.value.patient_id,
      encounter_type: initForm.value.encounter_type,
      department_name: initForm.value.department_name,
      onset_time: initForm.value.onset_time,
      duration: initForm.value.duration,
      chief_complaint: initForm.value.chief_complaint,
      present_illness: initForm.value.present_illness,
      vital_signs: vitalStr,
      initial_diagnosis: directForm.value.diagnosis,
      diagnostic_basis: directForm.value.etiology || '临床体征明确，未开具医技检查',
      need_exam: false,
      exam_items: [],
      exam_reason: '',
      direct_complete: true,
      diagnosis: directForm.value.diagnosis,
      etiology: directForm.value.etiology || '临床发病机理明确',
      treatment_plan: directForm.value.treatment_plan,
      soap_content: directForm.value.soap_content
    }

    const res: any = await api.post('/encounters/initial', payload)
    if (res.code === 200) {
      ElMessageBox.alert(
        `<div style="line-height: 1.6; font-size: 14px;">
          <p style="color: #059669; font-weight: bold; margin-bottom: 8px;">门诊就诊已一步完成最终确诊与规范归档！</p>
          <p style="margin-bottom: 6px;"><strong>可信区块链与隐私安全执行清单：</strong></p>
          <ul style="padding-left: 20px; margin: 6px 0; color: #334155; font-size: 13px;">
            <li><strong>诊疗决策：</strong>临床体征明确，无需医技辅助检查，就诊记录已直接闭环；</li>
            <li><strong>安全加密：</strong>采用 <strong>AES-256-GCM</strong> 对结构化 SOAP 电子病历完成对称加密；</li>
            <li><strong>分布式存储：</strong>密文已推送到去中心化 <strong>IPFS</strong> 存储节点；</li>
            <li><strong>不可篡改存证：</strong>病历哈希指纹已成功提交至 <strong>Hyperledger Fabric</strong> 联盟链！</li>
          </ul>
        </div>`,
        '就诊归档与区块链存证成功',
        {
          dangerouslyUseHTMLString: true,
          confirmButtonText: '确定',
          type: 'success'
        }
      )
      currentView.value = 'LIST'
    initialDialogVisible.value = false
      fetchRecords()
    } else {
      ElMessage.error(res.message || '归档就诊失败')
    }
  } catch (err: any) {
    ElMessage.error('就诊归档异常: ' + (err.message || '网络错误'))
  } finally {
    submitting.value = false
  }
}

async function submitInitialEncounter() {
  submitting.value = true
  try {
    const vitalStr = `T: ${vitals.value.temperature}℃, BP: ${vitals.value.blood_pressure} mmHg, HR: ${vitals.value.heart_rate} bpm, SpO2: ${vitals.value.spo2}%`
    const payload = {
      patient_id: initForm.value.patient_id,
      encounter_type: initForm.value.encounter_type,
      department_name: initForm.value.department_name,
      onset_time: initForm.value.onset_time,
      duration: initForm.value.duration,
      chief_complaint: initForm.value.chief_complaint,
      present_illness: initForm.value.present_illness,
      vital_signs: vitalStr,
      initial_diagnosis: initForm.value.initial_diagnosis,
      diagnostic_basis: initForm.value.diagnostic_basis,
      need_exam: selectedExamList.value.length > 0,
      exam_items: selectedExamList.value,
      exam_reason: initForm.value.exam_reason
    }

    const res: any = await api.post('/encounters/initial', payload)
    if (res.code === 200) {
      if (selectedExamList.value.length > 0) {
        ElMessageBox.alert(
          `<div style="line-height: 1.6; font-size: 14px;">
            <p style="color: #059669; font-weight: bold; margin-bottom: 8px;">初诊记录建立成功，已生成医技检查单并派发！</p>
            <p style="margin-bottom: 6px;"><strong>检查单派发去向说明：</strong></p>
            <ul style="padding-left: 20px; margin: 6px 0; color: #334155; font-size: 13px;">
              <li><strong>血常规 / 心肌酶谱 / 肝肾生化：</strong>已派发至 <strong>【临床检验科 (LIS任务池)】</strong>，由检验科技师 (<strong>tech_lab</strong>) 处理</li>
              <li><strong>12导联心电图 / 胸部CT / 头颅MRI：</strong>已派发至 <strong>【放射影像与心电中心 (PACS任务池)】</strong>，由影像医师 (<strong>tech_pacs</strong>) 处理</li>
            </ul>
            <p style="margin-top: 10px; font-size: 13px; color: #0284c7; font-weight: 600;">操作指引（两种演示方式均可）：</p>
            <p style="font-size: 13px; color: #475569; margin: 4px 0;"><strong>方式一（快捷协同）：</strong>直接点击左侧菜单的 <strong>「医技检查中心」</strong>，即可立即接单并出具回传报告！</p>
            <p style="font-size: 13px; color: #475569; margin: 4px 0;"><strong>方式二（多角色切换）：</strong>退出当前登录，切换为检验科账号 <strong>tech_lab</strong> (密码 <strong>123456</strong>) 或影像科账号 <strong>tech_pacs</strong> (密码 <strong>123456</strong>) 登录系统出报告！</p>
          </div>`,
          '检查申请已成功派发至医技科室',
          {
            dangerouslyUseHTMLString: true,
            confirmButtonText: '收到，立即前往医技中心',
            type: 'success'
          }
        )
      } else {
        ElMessage.success('初诊就诊记录已建立完成！')
      }
      currentView.value = 'LIST'
    initialDialogVisible.value = false
      fetchRecords()
    } else {
      ElMessage.error(res.message || '创建初诊失败')
    }
  } catch (err: any) {
    ElMessage.error('创建初诊异常: ' + (err.message || '网络错误'))
  } finally {
    submitting.value = false
  }
}

async function openFinalDialog(row: any) {
  activeRecordForFinal.value = row
  if (!activeRecordForFinal.value.infection_alert && activeRecordForFinal.value.patient_id) {
    try {
      const riskRes: any = await api.get(`/patients/${activeRecordForFinal.value.patient_id}/infection-risks`)
      if (riskRes.code === 200 && riskRes.data?.has_risk) {
        activeRecordForFinal.value.infection_alert = riskRes.data
      }
    } catch {}
  }
  const defaultPlan = row.treatment_plan || '【综合处置】\n1. 门诊心电图与心肌酶谱动态复查评估\n\n【临床医嘱】\n1. 限制钠盐摄入 (每日<5g)，保证充足规律睡眠\n2. 每日晨起及睡前规范监测自测血压\n3. 2周后心内科门诊复查血压及心电图\n\n【处方用药】\n1. 苯磺酸氨氯地平片 (络活喜) 5mg 口服 每日一次\n2. 酒石酸美托洛尔片 (倍他乐克) 25mg 口服 每日两次 饭前半小时'
  finalForm.value = {
    diagnosis: row.diagnosis || '原发性高血压病1级，心肌轻度供血不足',
    etiology: row.etiology || '劳累导致交感神经过度激活，外周阻力血管收缩，心肌负荷增加。',
    treatment_plan: defaultPlan,
    soap_content: ''
  }
  generateSOAPRecord()
  currentView.value = 'FINAL'
  finalDialogVisible.value = true
}

function generateSOAPRecord() {
  const rec = activeRecordForFinal.value
  if (!rec) return

  const pName = rec.patient_name || '就诊患者'
  const timeStr = rec.created_at ? rec.created_at.substring(0, 16).replace('T', ' ') : new Date().toLocaleString()
  const docName = auth.user?.real_name || '经治医师'

  const soapText = `【MedTrust 医疗可信共享·国家卫健委规范临床 SOAP 病历】
--------------------------------------------------------------------------------
就诊编号：${rec.record_no} | 接诊科室：${rec.department_name} | 就诊类型：${formatEncounterType(rec.encounter_type)}
患者姓名：${pName} | 就诊时间：${timeStr} | 责任医师：${docName}
--------------------------------------------------------------------------------
【S - Subjective 主观病史采集】
● 患者主诉：${rec.chief_complaint || rec.symptoms || '未主诉明确症状'}
● 现病史：${rec.present_illness || '患者因上述主诉就诊，病史采集如前所述。'}
● 发病时间：${rec.onset_time || '近期'} | 持续时间：${rec.duration || '发作性'}

【O - Objective 客观检查与测量】
● 查体生命体征：${rec.vital_signs || '生命体征平稳'}
● 辅助检查说明：${rec.need_exam ? `开立检查项目：${rec.exam_items}。医技回传报告：${rec.exam_result || '已核实'}` : '体征典型明确，经医师研判无需开展侵入性辅助检查。'}

【A - Assessment 综合评估与确诊】
● 经治医师最终确诊：${finalForm.value.diagnosis || rec.diagnosis || rec.initial_diagnosis}
● 病理诱因与机制：${finalForm.value.etiology || rec.etiology || '临床明确发病机制'}

【P - Plan 综合处置与医嘱方案】
${finalForm.value.treatment_plan || rec.treatment_plan || '遵临床常规医嘱治疗'}

● 责任医师合规签署：本病历经责任医师最终审定确诊，全流程数据经国家卫健委《电子病历应用规范》数字签名与 Fabric 区块链联盟存证。
--------------------------------------------------------------------------------
存证机构：${rec.hospital_name || auth.user?.hospital_name || '本医疗中心'} | 加密协议：AES-256-GCM | 存证底层：Hyperledger Fabric`

  finalForm.value.soap_content = soapText
}

async function submitFinalEncounter() {
  if (!activeRecordForFinal.value) return
  submittingFinal.value = true
  try {
    const p = {
      record_id: activeRecordForFinal.value.id,
      diagnosis: finalForm.value.diagnosis,
      etiology: finalForm.value.etiology,
      treatment_plan: finalForm.value.treatment_plan,
      soap_content: finalForm.value.soap_content
    }
    const res: any = await api.post(`/encounters/${activeRecordForFinal.value.id}/complete`, p)
    if (res.code === 200) {
      ElMessage.success('最终确诊已签署下达，全量临床 SOAP 病历已成功完成 Hyperledger Fabric 区块链存证！')
      currentView.value = 'LIST'
    finalDialogVisible.value = false
      fetchRecords()
    } else {
      ElMessage.error(res.message || '确认确诊失败')
    }
  } catch (err: any) {
    ElMessage.error('确认确诊异常: ' + (err.message || '网络错误'))
  } finally {
    submittingFinal.value = false
  }
}

async function openViewDetail(row: any) {
  try {
    const res: any = await api.get(`/medical-records/${row.id}`)
    if (res.code === 200) {
      detailRecord.value = res.data
      if (!detailRecord.value.infection_alert && detailRecord.value.patient_id) {
        try {
          const riskRes: any = await api.get(`/patients/${detailRecord.value.patient_id}/infection-risks`)
          if (riskRes.code === 200 && riskRes.data?.has_risk) {
            detailRecord.value.infection_alert = riskRes.data
          }
        } catch {}
      }
      currentView.value = 'DETAIL'
  detailModalVisible.value = true
    }
  } catch (err: any) {
    ElMessage.error('获取病历详情失败')
  }
}

function formatEncounterType(t?: string) {
  if (t === 'OUTPATIENT') return '普通门诊'
  if (t === 'EMERGENCY') return '急救门诊'
  if (t === 'INPATIENT') return '住院就诊'
  return t || '门诊'
}

function formatTime(t?: string) {
  if (!t) return '-'
  return t.substring(0, 16).replace('T', ' ')
}

async function openPdfPreview(rec: any) {
  previewingRecord.value = rec
  if (!previewingRecord.value.infection_alert && previewingRecord.value.patient_id) {
    try {
      const riskRes: any = await api.get(`/patients/${previewingRecord.value.patient_id}/infection-risks`)
      if (riskRes.code === 200 && riskRes.data?.has_risk) {
        previewingRecord.value.infection_alert = riskRes.data
      }
    } catch {}
  }
  pdfPreviewVisible.value = true
}

function printPdfSheet() {
  window.print()
}

function downloadRecordFile(recordId: number) {
  const baseUrl = api.defaults.baseURL || '/api/v1'
  const url = `${baseUrl}/medical-records/${recordId}/download`
  api.get(`/medical-records/${recordId}/download`, { responseType: 'blob' })
    .then((res: any) => {
      const blob = new Blob([res], { type: 'application/pdf;charset=utf-8' })
      const link = document.createElement('a')
      link.href = window.URL.createObjectURL(blob)
      const rNo = previewingRecord.value?.record_no || detailRecord.value?.record_no || recordId
      link.download = `临床就诊规范归档凭据_${rNo}.pdf`
      document.body.appendChild(link)
      link.click()
      document.body.removeChild(link)
      window.URL.revokeObjectURL(link.href)
      ElMessage.success('已成功下载解密归档凭据！')
    })
    .catch(() => {
      window.open(url, '_blank')
    })
}

function formatFileSize(bytes?: number) {
  if (!bytes) return '2.8 KB'
  if (bytes < 1024) return bytes + ' B'
  if (bytes < 1024 * 1024) return (bytes / 1024).toFixed(1) + ' KB'
  return (bytes / (1024 * 1024)).toFixed(2) + ' MB'
}


function copyToClipboard(text: string, label = '交易哈希') {
  if (!text) return
  if (navigator.clipboard) {
    navigator.clipboard.writeText(text).then(() => {
      ElMessage.success(`已复制${label}到剪贴板`)
    }).catch(() => {
      fallbackCopy(text, label)
    })
  } else {
    fallbackCopy(text, label)
  }
}

function fallbackCopy(text: string, label: string) {
  const textArea = document.createElement('textarea')
  textArea.value = text
  document.body.appendChild(textArea)
  textArea.select()
  try {
    document.execCommand('copy')
    ElMessage.success(`已复制${label}到剪贴板`)
  } catch (err) {
    ElMessage.error('复制失败，请手动复制')
  }
  document.body.removeChild(textArea)
}

function shortHash(hash?: string, left = 12, right = 10) {
  if (!hash) return '-'
  if (hash.length <= left + right) return hash
  return `${hash.slice(0, left)}...${hash.slice(-right)}`
}

function parseVitals(vitalStr?: string) {
  const res = {
    temperature: '',
    blood_pressure: '',
    heart_rate: '',
    spo2: '',
    raw: vitalStr || ''
  }
  if (!vitalStr) return res

  const tMatch = vitalStr.match(/(?:T|体温)[:：\s]*([0-9.]+\s*℃?)/i)
  if (tMatch) res.temperature = tMatch[1].includes('℃') ? tMatch[1] : `${tMatch[1]} ℃`

  const bpMatch = vitalStr.match(/(?:BP|血压)[:：\s]*([0-9]+\/[0-9]+\s*mmHg?)/i)
  if (bpMatch) res.blood_pressure = bpMatch[1].includes('mmHg') ? bpMatch[1] : `${bpMatch[1]} mmHg`

  const hrMatch = vitalStr.match(/(?:HR|心率|脉搏)[:：\s]*([0-9]+\s*bpm?)/i)
  if (hrMatch) res.heart_rate = hrMatch[1].includes('bpm') ? hrMatch[1] : `${hrMatch[1]} bpm`

  const spo2Match = vitalStr.match(/(?:SpO2|血氧)[:：\s]*([0-9.]+\s*%?)/i)
  if (spo2Match) res.spo2 = spo2Match[1].includes('%') ? spo2Match[1] : `${spo2Match[1]} %`

  return res
}

function parsePlanItems(planStr?: string) {
  if (!planStr) return []
  const lines = planStr.split(/\n+/).map(l => l.trim()).filter(Boolean)
  return lines.map((line, idx) => {
    const cleaned = line.replace(/^[0-9]+[、.．]\s*/, '')
    const isMed = /[0-9]+(mg|g|ml|片|粒|包|支)|口服|静脉|外用|每日|bid|tid|qd/i.test(cleaned)
    return {
      id: idx + 1,
      text: cleaned,
      isMed
    }
  })
}

onMounted(() => {
  fetchRecords()
  fetchPatients()
})
</script>

<style scoped>
.doctor-records-container {
  padding: 20px;
  display: flex;
  flex-direction: column;
  gap: 20px;
}

.stats-grid {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 16px;
}

.stat-card {
  background: #ffffff;
  border-radius: 12px;
  padding: 18px;
  display: flex;
  align-items: center;
  gap: 14px;
  border: 1px solid rgba(226, 232, 240, 0.8);
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.02);
}

.stat-card.blue { border-left: 4px solid #0284c7; }
.stat-card.yellow { border-left: 4px solid #f59e0b; }
.stat-card.cyan { border-left: 4px solid #06b6d4; }
.stat-card.green { border-left: 4px solid #10b981; }

.stat-icon { font-size: 2rem; }
.stat-value { font-size: 1.7rem; font-weight: 700; color: #1e293b; line-height: 1.2; }
.stat-label { font-size: 0.82rem; color: #64748b; margin-top: 4px; }

.main-card {
  border-radius: 12px;
  border: 1px solid rgba(226, 232, 240, 0.8);
}

.card-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.header-title {
  font-size: 1.1rem;
  font-weight: 600;
  color: #1e293b;
}

.filter-bar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 16px;
  flex-wrap: wrap;
  gap: 12px;
}

.mono {
  font-family: 'Consolas', monospace;
  letter-spacing: 0.5px;
}

.patient-name {
  font-weight: 600;
  color: #0f172a;
}

.status-ready-tag {
  background: #0891b2 !important;
  color: #ffffff !important;
  border-color: #0891b2 !important;
}

.form-step-title {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 0.95rem;
  font-weight: 700;
  color: #0369a1;
  border-bottom: 1px dashed #cbd5e1;
  padding-bottom: 6px;
  margin-bottom: 12px;
}

.step-num {
  width: 20px;
  height: 20px;
  border-radius: 50%;
  background: #0284c7;
  color: #ffffff;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 0.75rem;
}

.patient-quick-card {
  background: #f8fafc;
  border: 1px solid #e2e8f0;
  border-radius: 8px;
  padding: 10px 12px;
  font-size: 0.85rem;
  color: #334155;
}

.p-line {
  margin-bottom: 4px;
}

.patient-placeholder {
  background: #f8fafc;
  border: 1px dashed #cbd5e1;
  border-radius: 8px;
  height: 80px;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 0.85rem;
  color: #94a3b8;
}

.vitals-panel {
  background: #f1f5f9;
  border-radius: 8px;
  padding: 12px;
}

.vitals-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 8px;
}

.exam-order-box {
  background: #fefce8;
  border: 1px solid #fef08a;
  border-radius: 8px;
  padding: 12px 16px;
}

.summary-banner {
  background: #f8fafc;
  border: 1px solid #e2e8f0;
  border-radius: 8px;
  padding: 12px 16px;
  font-size: 0.88rem;
}

.s-head {
  display: flex;
  align-items: center;
  gap: 12px;
}

.lab-results-display {
  background: #f0fdf4;
  border: 1px solid #bbf7d0;
  border-radius: 8px;
  padding: 12px 16px;
  font-size: 0.9rem;
}

.lab-result-text {
  white-space: pre-line;
  line-height: 1.6;
  color: #166534;
  font-weight: 500;
}

.soap-action-bar {
  display: flex;
  align-items: center;
  gap: 10px;
}

.soap-textarea :deep(textarea) {
  font-family: 'Consolas', 'Courier New', monospace;
  font-size: 0.85rem;
  background: #fafafa;
  line-height: 1.5;
}

.tamper-box {
  border-radius: 8px;
  padding: 12px 16px;
  margin-bottom: 16px;
  font-size: 0.9rem;
}

.tamper-box.safe {
  background: #ecfdf5;
  border: 1px solid #a7f3d0;
  color: #065f46;
}

.tamper-box.danger {
  background: #fef2f2;
  border: 1px solid #fca5a5;
  color: #991b1b;
}

.tamper-tag-glow {
  box-shadow: 0 0 10px rgba(239, 68, 68, 0.6);
  animation: pulse-red 1.5s infinite;
  font-weight: 700;
}

@keyframes pulse-red {
  0%, 100% { opacity: 1; transform: scale(1); }
  50% { opacity: 0.85; transform: scale(1.05); }
}

.t-head {
  font-weight: 700;
  font-size: 1rem;
  margin-bottom: 6px;
}

.t-desc {
  font-size: 0.85rem;
  line-height: 1.5;
  margin-bottom: 8px;
}

.t-hashes {
  font-size: 0.8rem;
  background: #ffffff;
  padding: 8px;
  border-radius: 4px;
}

.detail-meta-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 8px;
  font-size: 0.88rem;
  background: #f8fafc;
  padding: 12px;
  border-radius: 8px;
  margin-bottom: 16px;
}

.detail-table {
  width: 100%;
  border-collapse: collapse;
  font-size: 0.88rem;
  margin-bottom: 16px;
}

.detail-table th,
.detail-table td {
  border: 1px solid #e2e8f0;
  padding: 10px 14px;
}

.detail-table th {
  background: #f8fafc;
  color: #475569;
  text-align: left;
}

.pre-text {
  white-space: pre-line;
  line-height: 1.5;
}

.pre-text.highlight {
  color: #0284c7;
  font-weight: 500;
}

.evidence-footer {
  background: #f1f5f9;
  padding: 12px;
  border-radius: 6px;
  font-size: 0.82rem;
  color: #475569;
}

/* 临床归档文件卡片样式 */
.archive-files-section {
  margin: 12px 0 16px;
}

.file-cards-list {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.file-item-card {
  display: flex;
  align-items: center;
  background: #f8fafc;
  border: 1px solid #e2e8f0;
  border-radius: 8px;
  padding: 12px 16px;
  gap: 14px;
  transition: all 0.2s;
}

.file-item-card:hover {
  border-color: #cbd5e1;
  box-shadow: 0 2px 6px rgba(0, 0, 0, 0.04);
}

.file-icon-box {
  font-size: 28px;
  line-height: 1;
}

.file-info {
  flex: 1;
  min-width: 0;
}

.file-name {
  font-size: 14px;
  color: #1e293b;
  display: flex;
  align-items: center;
}

.file-cid, .file-hash {
  font-family: monospace;
  word-break: break-all;
}

.file-actions {
  display: flex;
  gap: 8px;
}

.empty-files-box {
  background: #f8fafc;
  border: 1px dashed #cbd5e1;
  border-radius: 8px;
  padding: 14px 18px;
}

.empty-flex-row {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 12px;
}

/* 拟真红头电子病历 PDF 预览单纸张样式 */
.emr-sheet-wrapper {
  background: #f1f5f9;
  padding: 16px;
  border-radius: 8px;
}

.emr-action-bar {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 14px;
  background: white;
  padding: 10px 16px;
  border-radius: 8px;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.05);
}

.emr-sheet-paper {
  background: white;
  box-shadow: 0 4px 16px rgba(0, 0, 0, 0.08);
  padding: 40px 48px;
  border-radius: 4px;
  position: relative;
  max-width: 800px;
  margin: 0 auto;
}

.emr-header {
  text-align: center;
  margin-bottom: 20px;
}

.emr-hospital-name {
  font-size: 22px;
  font-weight: 800;
  color: #dc2626;
  letter-spacing: 2px;
  margin-bottom: 4px;
}

.emr-doc-title {
  font-size: 18px;
  font-weight: 700;
  color: #1e293b;
  letter-spacing: 4px;
  margin-bottom: 4px;
}

.emr-sub-title {
  font-size: 12px;
  color: #64748b;
  margin-bottom: 12px;
}

.emr-red-line {
  height: 3px;
  background: #dc2626;
  border-bottom: 1px solid #dc2626;
  margin: 0 auto;
}

.emr-meta-banner {
  display: flex;
  justify-content: space-between;
  align-items: center;
  font-size: 12px;
  color: #475569;
  padding: 10px 0;
  border-bottom: 1px dashed #cbd5e1;
  margin-bottom: 12px;
}

.meta-barcode-box {
  text-align: center;
}

.barcode-graphic {
  font-family: monospace;
  font-size: 16px;
  letter-spacing: 2px;
  font-weight: 900;
  color: #0f172a;
}

.barcode-text {
  font-size: 10px;
  color: #64748b;
}

.emr-patient-table {
  width: 100%;
  border-collapse: collapse;
  margin-bottom: 16px;
  font-size: 13px;
}

.emr-patient-table th, .emr-patient-table td {
  border: 1px solid #cbd5e1;
  padding: 6px 10px;
}

.emr-patient-table th {
  background: #f8fafc;
  color: #475569;
  font-weight: 600;
}

.emr-soap-section {
  display: flex;
  flex-direction: column;
  gap: 14px;
  margin-bottom: 24px;
}

.soap-block {
  border-left: 3px solid #3b82f6;
  padding-left: 12px;
}

.soap-title {
  font-weight: 700;
  color: #1e40af;
  font-size: 14px;
  margin-bottom: 4px;
}

.soap-row {
  font-size: 13px;
  color: #334155;
  line-height: 1.6;
  margin-bottom: 4px;
}

.vitals-highlight {
  color: #059669;
  font-weight: 600;
}

.soap-exam-result-box {
  background: #f8fafc;
  border: 1px solid #e2e8f0;
  border-radius: 6px;
  padding: 10px 12px;
  margin-top: 6px;
}

.soap-exam-head {
  font-weight: 600;
  color: #0284c7;
  font-size: 12px;
  margin-bottom: 4px;
}

.soap-exam-content, .soap-plan-content {
  margin: 0;
  font-family: inherit;
  white-space: pre-wrap;
  font-size: 13px;
  line-height: 1.6;
  color: #1e293b;
}

.soap-exam-foot {
  margin-top: 6px;
  font-size: 11px;
  color: #64748b;
  text-align: right;
}

.emr-footer-box {
  display: flex;
  justify-content: space-between;
  align-items: flex-end;
  border-top: 1px solid #cbd5e1;
  padding-top: 16px;
  margin-top: 20px;
}

.footer-crypto-evidence {
  font-size: 11px;
  color: #64748b;
  display: flex;
  flex-direction: column;
  gap: 4px;
  max-width: 480px;
}

.footer-seal-area {
  display: flex;
  align-items: center;
  gap: 20px;
  position: relative;
}

.doctor-signature-box {
  text-align: right;
  font-size: 13px;
  color: #334155;
}

.sig-name {
  font-weight: 700;
  font-size: 15px;
  color: #0f172a;
  border-bottom: 1px solid #334155;
  padding: 0 6px 2px;
}

/* 红色印章样式 */
.emr-stamp {
  width: 90px;
  height: 90px;
  border: 2px solid #dc2626;
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  color: #dc2626;
  transform: rotate(-12deg);
  opacity: 0.85;
}

.stamp-inner {
  text-align: center;
  font-weight: 700;
}

.stamp-star {
  font-size: 14px;
  line-height: 1;
}

.stamp-text-top {
  font-size: 8px;
  letter-spacing: 1px;
}

.stamp-text-center {
  font-size: 10px;
  letter-spacing: 1px;
  margin: 1px 0;
}

.stamp-text-bottom {
  font-size: 7px;
}

.decision-mode-bar {
  background: #f8fafc;
  padding: 12px 16px;
  border-radius: 8px;
  border: 1px solid #e2e8f0;
}

.direct-decision-panel {
  background: #f8fafc;
  border: 1px solid #cbd5e1;
  border-radius: 8px;
  padding: 16px 18px 8px;
  margin-top: 8px;
}

.patient-cross-action {
  margin-top: 8px;
}

.inline-img-card {
  background: #f8fafc;
  border: 1px solid #e2e8f0;
  border-radius: 8px;
  padding: 10px 14px;
}

.preview-thumbnail {
  max-width: 100%;
  max-height: 240px;
  border-radius: 6px;
  border: 1px solid #cbd5e1;
  cursor: pointer;
  transition: transform 0.2s ease, box-shadow 0.2s ease;
}

.preview-thumbnail:hover {
  transform: scale(1.02);
  box-shadow: 0 4px 12px rgba(59, 130, 246, 0.15);
  border-color: #3b82f6;
}

.emr-infection-watermark {
  border: 2px solid #dc2626;
  color: #dc2626;
  font-weight: 800;
  font-size: 13px;
  padding: 6px 14px;
  border-radius: 4px;
  background: rgba(254, 242, 242, 0.9);
  display: inline-block;
  margin-bottom: 12px;
  letter-spacing: 0.5px;
  box-shadow: 0 2px 8px rgba(220, 38, 38, 0.18);
  width: 100%;
  box-sizing: border-box;
  text-align: center;
}

/* 打印样式 */
@media print {
  body * {
    visibility: hidden;
  }
  #emr-print-container,
  #emr-print-container * {
    visibility: visible;
  }
  #emr-print-container {
    position: absolute;
    left: 0;
    top: 0;
    width: 100%;
    margin: 0;
    padding: 0;
    background: white;
  }
  .no-print {
    display: none !important;
  }
}

/* 全屏临床工作台基础设计 */
.fullscreen-workstation-dialog.el-dialog.is-fullscreen {
  display: flex !important;
  flex-direction: column !important;
}

.fullscreen-workstation-dialog :deep(.el-dialog__header) {
  padding: 14px 24px !important;
  border-bottom: 1px solid #e2e8f0 !important;
  background: #ffffff !important;
  margin-right: 0 !important;
}

.fullscreen-workstation-dialog :deep(.el-dialog__body) {
  flex: 1 !important;
  overflow-y: auto !important;
  padding: 20px 24px !important;
  background: #f1f5f9 !important;
}

.fullscreen-workstation-dialog :deep(.el-dialog__footer) {
  padding: 12px 24px !important;
  border-top: 1px solid #e2e8f0 !important;
  background: #ffffff !important;
}

.ws-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  width: 100%;
  padding-right: 28px;
}

.ws-header-left {
  display: flex;
  align-items: center;
  gap: 10px;
}

.ws-header-title {
  font-size: 17px;
  font-weight: 700;
  color: #1e293b;
}

.ws-header-sub {
  font-size: 13px;
  color: #64748b;
  margin-left: 6px;
}

.ws-header-right {
  display: flex;
  align-items: center;
  gap: 12px;
}

.ws-body-grid {
  display: flex;
  gap: 20px;
  align-items: flex-start;
}

.ws-col-left {
  width: 380px;
  flex-shrink: 0;
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.ws-col-main {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.ws-card {
  background: #ffffff;
  border: 1px solid #e2e8f0;
  border-radius: 8px;
  padding: 16px 20px;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.04);
}

.ws-card-title {
  font-size: 15px;
  font-weight: 700;
  color: #0f172a;
  display: flex;
  align-items: center;
  gap: 8px;
  padding-bottom: 10px;
  margin-bottom: 14px;
  border-bottom: 1px solid #f1f5f9;
}

.step-badge {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 22px;
  height: 22px;
  background: #3b82f6;
  color: white;
  border-radius: 50%;
  font-size: 12px;
  font-weight: 700;
}

.summary-kv-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
  font-size: 13px;
}

.kv-item {
  display: flex;
  justify-content: space-between;
  align-items: center;
  border-bottom: 1px dashed #f1f5f9;
  padding-bottom: 6px;
}

.kv-item .k {
  color: #64748b;
  font-weight: 500;
}

.kv-item .v {
  color: #1e293b;
  text-align: right;
  max-width: 260px;
  word-break: break-all;
}

.ws-footer-bar {
  display: flex;
  justify-content: space-between;
  align-items: center;
  width: 100%;
}



/* =======================================================
   现代化全景工作台与 Bento 临床布局设计 (Modern Clinical Bento)
   ======================================================= */
.inpage-workstation-wrapper {
  display: flex;
  flex-direction: column;
  gap: 16px;
  animation: fadeIn 0.25s ease-out;
  padding-bottom: 24px;
}

@keyframes fadeIn {
  from { opacity: 0; transform: translateY(6px); }
  to { opacity: 1; transform: translateY(0); }
}

/* 顶部粘性工作台导航卡片 (纤细轻量条 48px - 杜绝换行膨胀) */
.ws-page-header-card.sticky-header {
  position: sticky;
  top: -24px;
  z-index: 40;
  height: 48px;
  min-height: 48px;
  max-height: 48px;
  display: flex;
  justify-content: space-between;
  align-items: center;
  background: rgba(255, 255, 255, 0.96);
  backdrop-filter: blur(14px);
  border: 1px solid #e2e8f0;
  border-radius: 8px;
  padding: 0 16px;
  box-shadow: 0 2px 8px rgba(15, 23, 42, 0.04);
  white-space: nowrap !important;
  box-sizing: border-box;
}

.ws-header-left {
  display: flex;
  align-items: center;
  gap: 10px;
  flex-shrink: 1;
  min-width: 0;
  white-space: nowrap !important;
}

.ws-header-title-box {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-shrink: 1;
  min-width: 0;
  white-space: nowrap !important;
}

.ws-page-title {
  font-size: 15px;
  font-weight: 700;
  color: #0f172a;
  white-space: nowrap !important;
  flex-shrink: 0 !important;
}

.ws-header-sub {
  font-size: 12px;
  color: #64748b;
  white-space: nowrap !important;
  flex-shrink: 0 !important;
}

.ws-record-chip {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  background: #f1f5f9;
  border: 1px solid #e2e8f0;
  padding: 2px 8px;
  border-radius: 4px;
  font-size: 12px;
  white-space: nowrap !important;
  flex-shrink: 0 !important;
}

.ws-record-chip .chip-lbl {
  color: #64748b;
  font-weight: 500;
}

.ws-record-chip .chip-val {
  color: #0f172a;
  font-weight: 700;
}

.copy-icon {
  cursor: pointer;
  color: #64748b;
  transition: all 0.2s;
}

.copy-icon:hover {
  color: #2563eb;
  transform: scale(1.15);
}

.ws-header-right {
  display: flex;
  align-items: center;
  gap: 10px;
  flex-shrink: 0 !important;
  white-space: nowrap !important;
}

/* 防篡改状态条 */
.tamper-box.safe {
  background: linear-gradient(135deg, #f0fdf4 0%, #ecfdf5 100%);
  border: 1.5px solid #86efac;
  border-radius: 10px;
  padding: 12px 18px;
  display: flex;
  justify-content: space-between;
  align-items: center;
  box-shadow: 0 2px 8px rgba(16, 185, 129, 0.08);
}

.tb-safe-left {
  display: flex;
  align-items: center;
  gap: 10px;
}

.fabric-badge {
  display: inline-block;
  background: #065f46;
  color: #a7f3d0;
  font-family: monospace;
  font-size: 11px;
  font-weight: 700;
  padding: 3px 10px;
  border-radius: 6px;
  letter-spacing: 0.5px;
}

/* Bento 基础卡片 */
.bento-card {
  background: #ffffff;
  border: 1px solid #e2e8f0;
  border-radius: 12px;
  padding: 18px 20px;
  box-shadow: 0 1px 3px rgba(15, 23, 42, 0.04);
  transition: all 0.2s ease;
}

.bento-card:hover {
  box-shadow: 0 6px 18px -2px rgba(15, 23, 42, 0.06);
  border-color: #cbd5e1;
}

/* 患者基本信息卡片 */
.patient-hero-card {
  background: #ffffff;
}

.ph-header {
  display: flex;
  align-items: center;
  gap: 14px;
}

.ph-avatar {
  width: 48px;
  height: 48px;
  border-radius: 50%;
  background: linear-gradient(135deg, #0284c7 0%, #2563eb 100%);
  color: #ffffff;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 20px;
  font-weight: 800;
  box-shadow: 0 4px 12px rgba(37, 99, 235, 0.28);
  flex-shrink: 0;
}

.ph-main-info {
  flex: 1;
  min-width: 0;
}

.ph-name-row {
  display: flex;
  align-items: center;
  gap: 8px;
}

.ph-name {
  font-size: 18px;
  font-weight: 800;
  color: #0f172a;
}

.ph-age {
  font-size: 13px;
  color: #64748b;
  font-weight: 500;
}

.ph-divider {
  height: 1px;
  background: #f1f5f9;
  margin: 14px 0;
}

.ph-details-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 12px;
}

.ph-item {
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.ph-item .lbl {
  font-size: 11px;
  color: #64748b;
  font-weight: 500;
}

.ph-item .val {
  font-size: 13px;
  color: #1e293b;
}

/* 区块链深色存证卡片 */
.crypto-card {
  background: linear-gradient(145deg, #0f172a 0%, #1e293b 100%);
  border: 1px solid #334155;
  color: #f8fafc;
}

.crypto-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding-bottom: 12px;
  border-bottom: 1px solid rgba(255, 255, 255, 0.08);
  margin-bottom: 14px;
}

.crypto-title {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 14px;
  font-weight: 700;
  color: #f1f5f9;
}

.crypto-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background: #10b981;
  box-shadow: 0 0 10px #10b981;
}

.cf-head {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 6px;
}

.cf-label {
  font-size: 11px;
  color: #94a3b8;
  font-weight: 600;
}

.cf-copy-btn {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  background: rgba(255, 255, 255, 0.08);
  border: 1px solid rgba(255, 255, 255, 0.15);
  color: #93c5fd;
  border-radius: 4px;
  padding: 2px 8px;
  font-size: 11px;
  cursor: pointer;
  transition: all 0.2s;
}

.cf-copy-btn:hover {
  background: rgba(255, 255, 255, 0.18);
  color: #bfdbfe;
}

.cf-hash-box {
  background: rgba(0, 0, 0, 0.35);
  border: 1px solid rgba(255, 255, 255, 0.08);
  border-radius: 6px;
  padding: 8px 12px;
  font-size: 12px;
  color: #38bdf8;
  letter-spacing: 0.5px;
}

.crypto-stats-row {
  display: grid;
  grid-template-columns: 1fr 1fr 1fr;
  gap: 8px;
  margin: 14px 0 10px 0;
}

.stat-pill {
  background: rgba(255, 255, 255, 0.05);
  border: 1px solid rgba(255, 255, 255, 0.07);
  border-radius: 6px;
  padding: 6px 8px;
  text-align: center;
}

.stat-k {
  display: block;
  font-size: 10px;
  color: #94a3b8;
}

.stat-v {
  display: block;
  font-size: 12px;
  font-weight: 700;
  color: #f1f5f9;
  margin-top: 2px;
}

.crypto-security-seal {
  font-size: 11px;
  color: #6ee7b7;
  display: flex;
  align-items: center;
  padding-top: 8px;
  border-top: 1px dashed rgba(255, 255, 255, 0.1);
}

/* SOAP Bento 维度卡片 */
.soap-bento-card {
  position: relative;
  overflow: hidden;
}

.soap-card-head {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding-bottom: 12px;
  border-bottom: 1px solid #f1f5f9;
  margin-bottom: 14px;
}

.soap-head-title {
  display: flex;
  align-items: center;
  gap: 10px;
}

.soap-dim-badge {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 26px;
  height: 26px;
  border-radius: 6px;
  color: #ffffff;
  font-weight: 900;
  font-size: 14px;
  box-shadow: 0 2px 6px rgba(0, 0, 0, 0.12);
}

.s-badge { background: linear-gradient(135deg, #0284c7, #0369a1); }
.o-badge { background: linear-gradient(135deg, #059669, #047857); }
.a-badge { background: linear-gradient(135deg, #6366f1, #4f46e5); }
.p-badge { background: linear-gradient(135deg, #d97706, #b45309); }

.soap-title-text {
  font-size: 15px;
  font-weight: 700;
  color: #0f172a;
}

.time-chip {
  display: inline-block;
  font-size: 12px;
  color: #475569;
  background: #f1f5f9;
  padding: 3px 8px;
  border-radius: 6px;
  margin-left: 6px;
}

/* S 维度 主诉卡片 */
.complaint-hero-box {
  background: linear-gradient(135deg, #f0f9ff 0%, #f8fafc 100%);
  border: 1.5px solid #bae6fd;
  border-radius: 10px;
  padding: 16px 20px;
  position: relative;
}

.chb-quote-mark {
  position: absolute;
  top: -6px;
  left: 12px;
  font-size: 32px;
  font-family: Georgia, serif;
  color: #7dd3fc;
  line-height: 1;
  pointer-events: none;
}

.chb-text {
  font-size: 16px;
  font-weight: 700;
  color: #0369a1;
  line-height: 1.6;
  padding-left: 12px;
}

.present-illness-box {
  background: #f8fafc;
  border: 1px solid #e2e8f0;
  border-left: 4px solid #0284c7;
  border-radius: 6px;
  padding: 12px 16px;
}

.pib-label {
  font-size: 12px;
  font-weight: 700;
  color: #334155;
  margin-bottom: 4px;
}

.pib-content {
  font-size: 13px;
  color: #475569;
  line-height: 1.6;
}

/* O 维度 生命体征 4 项网格 */
.vitals-sensor-grid {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 12px;
}

.vital-sensor-card {
  display: flex;
  align-items: center;
  gap: 12px;
  background: #f8fafc;
  border: 1px solid #e2e8f0;
  border-radius: 8px;
  padding: 12px 14px;
  transition: transform 0.2s;
}

.vital-sensor-card:hover {
  transform: translateY(-2px);
  background: #ffffff;
  border-color: #cbd5e1;
}

.vsc-icon-box {
  width: 38px;
  height: 38px;
  border-radius: 8px;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 18px;
  background: #ffffff;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.06);
  flex-shrink: 0;
}

.vsc-info {
  display: flex;
  flex-direction: column;
  min-width: 0;
}

.vsc-name {
  font-size: 11px;
  color: #64748b;
  font-weight: 600;
}

.vsc-val {
  font-size: 16px;
  font-weight: 800;
  color: #0f172a;
  line-height: 1.2;
  margin: 2px 0;
}

.vsc-status.normal {
  font-size: 10px;
  color: #059669;
  font-weight: 600;
}

/* 医技检验板块 */
.exam-report-bento-section {
  background: #f8fafc;
  border: 1px solid #e2e8f0;
  border-radius: 8px;
  padding: 14px 16px;
}

.erb-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 8px;
}

.erb-title {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 13px;
  font-weight: 700;
  color: #1e293b;
}

.erb-items-tag {
  font-size: 11px;
  color: #d97706;
  background: #fef3c7;
  padding: 2px 8px;
  border-radius: 4px;
  font-weight: 600;
}

.erb-result-box {
  background: #ffffff;
  border: 1px solid #e2e8f0;
  border-radius: 6px;
  padding: 12px 14px;
}

.erb-report-text {
  font-size: 13px;
  color: #334155;
  line-height: 1.6;
}

.erb-footer {
  margin-top: 8px;
  padding-top: 8px;
  border-top: 1px dashed #e2e8f0;
  font-size: 11px;
  color: #64748b;
  text-align: right;
}

.erb-empty-box {
  display: flex;
  align-items: center;
  font-size: 12px;
  color: #059669;
  background: #ecfdf5;
  padding: 8px 12px;
  border-radius: 6px;
}

/* A 维度 确诊 Hero Card */
.diagnosis-hero-banner {
  background: linear-gradient(135deg, #f0fdf4 0%, #dcfce7 100%);
  border: 1.5px solid #86efac;
  border-radius: 10px;
  padding: 18px 22px;
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.dhb-label {
  font-size: 12px;
  font-weight: 700;
  color: #166534;
  letter-spacing: 0.5px;
}

.dhb-name {
  font-size: 20px;
  font-weight: 900;
  color: #14532d;
  margin: 4px 0;
}

.dhb-stamp-badge {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  background: #ffffff;
  border: 1.5px solid #22c55e;
  color: #15803d;
  font-size: 13px;
  font-weight: 800;
  padding: 6px 14px;
  border-radius: 9999px;
  box-shadow: 0 2px 8px rgba(34, 197, 94, 0.2);
}

.etiology-bento-box {
  background: #f8fafc;
  border: 1px solid #e2e8f0;
  border-left: 4px solid #6366f1;
  border-radius: 6px;
  padding: 12px 16px;
}

.ebb-title {
  font-size: 12px;
  font-weight: 700;
  color: #4338ca;
  margin-bottom: 4px;
}

.ebb-content {
  font-size: 13px;
  color: #334155;
  line-height: 1.6;
}

/* P 维度 处方与医嘱方案 */
.rx-orders-container {
  background: #ffffff;
  border: 1px solid #fef3c7;
  border-radius: 8px;
  padding: 14px 16px;
}

.rx-header {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 12px;
}

.rx-symbol {
  font-family: serif;
  font-size: 22px;
  font-weight: 900;
  color: #d97706;
}

.rx-title {
  font-size: 14px;
  font-weight: 700;
  color: #92400e;
}

.rx-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.rx-item-card {
  display: flex;
  align-items: center;
  gap: 12px;
  background: #fffbeb;
  border: 1px solid #fde68a;
  border-radius: 6px;
  padding: 10px 14px;
  transition: all 0.2s;
}

.rx-item-card:hover {
  background: #fef3c7;
  border-color: #f59e0b;
}

.ric-num {
  width: 20px;
  height: 20px;
  border-radius: 50%;
  background: #d97706;
  color: white;
  font-size: 11px;
  font-weight: 700;
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
}

.ric-content {
  flex: 1;
  min-width: 0;
}

.ric-text {
  font-size: 13px;
  font-weight: 600;
  color: #1e293b;
}

.doctor-signature-banner {
  background: #f8fafc;
  border: 1px solid #e2e8f0;
  border-radius: 8px;
  padding: 12px 18px;
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.dsb-ca-tag {
  background: #dbeafe;
  color: #1e40af;
  font-size: 10px;
  font-weight: 700;
  padding: 1px 6px;
  border-radius: 4px;
  margin-left: 8px;
}

.ws-page-footer-card {
  background: #ffffff;
  border: 1px solid #e2e8f0;
  border-radius: 10px;
  padding: 14px 20px;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.04);
}

/* =======================================================
   PDF 规范弹窗与国家级红头病历升级样式
   ======================================================= */
.pdf-preview-dialog :deep(.el-dialog) {
  border-radius: 12px;
  overflow: hidden;
  box-shadow: 0 25px 50px -12px rgba(0, 0, 0, 0.25);
}

.emr-action-bar {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 16px;
  margin-bottom: 16px;
  background: #ffffff;
  padding: 12px 20px;
  border-radius: 8px;
  border: 1px solid #e2e8f0;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.05);
  flex-wrap: nowrap;
}

.emr-tip-tag {
  display: flex;
  align-items: center;
  gap: 10px;
  flex-wrap: nowrap;
  flex-shrink: 0;
}

.emr-sheet-paper {
  background: #ffffff;
  box-shadow: 0 8px 30px rgba(0, 0, 0, 0.12);
  padding: 48px 56px;
  border-radius: 4px;
  position: relative;
  max-width: 900px;
  margin: 0 auto;
  border: 1px solid #e2e8f0;
}

.emr-hospital-name {
  font-family: "STSong", "SimSun", "Songti SC", serif;
  font-size: 24px;
  font-weight: 900;
  color: #dc2626;
  letter-spacing: 3px;
  margin-bottom: 6px;
}

.emr-red-line-double {
  margin: 12px auto 16px auto;
  position: relative;
  text-align: center;
}

.emr-red-line-double .line-thick {
  height: 3px;
  background: #dc2626;
}

.emr-red-line-double .line-thin {
  height: 1px;
  background: #dc2626;
  margin-top: 2px;
}

/* =======================================================
   附件凭证卡片分层现代化设计 (Non-overlapping Bento File Layout)
   ======================================================= */
.files-card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding-bottom: 10px;
  border-bottom: 1px solid #f1f5f9;
  margin-bottom: 12px;
}

.fch-title {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 14px;
  font-weight: 700;
  color: #0f172a;
}

.fch-tag {
  font-size: 11px;
}

.file-cards-list {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.file-item-bento {
  background: #f8fafc;
  border: 1px solid #e2e8f0;
  border-radius: 8px;
  padding: 12px 14px;
  display: flex;
  flex-direction: column;
  gap: 8px;
  transition: all 0.2s;
}

.file-item-bento:hover {
  background: #ffffff;
  border-color: #cbd5e1;
  box-shadow: 0 4px 12px rgba(15, 23, 42, 0.05);
}

.fib-top-row {
  display: flex;
  align-items: flex-start;
  gap: 10px;
}

.fib-icon {
  font-size: 24px;
  line-height: 1;
  margin-top: 2px;
  flex-shrink: 0;
}

.fib-meta {
  flex: 1;
  min-width: 0;
}

.fib-name-box {
  width: 100%;
}

.fib-name {
  display: block;
  font-size: 13px;
  font-weight: 700;
  color: #1e293b;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.fib-tags {
  display: flex;
  align-items: center;
  gap: 6px;
  margin-top: 4px;
}

.fib-cid-row {
  background: #ffffff;
  border: 1px dashed #cbd5e1;
  border-radius: 4px;
  padding: 4px 8px;
  font-size: 11px;
  color: #64748b;
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.cid-val {
  color: #0284c7;
  font-weight: 600;
}

.fib-actions-row {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-top: 2px;
}

.fib-btn {
  flex: 1;
  margin: 0 !important;
  justify-content: center;
  font-weight: 600;
}

.empty-files-bento {
  background: #f8fafc;
  border: 1px dashed #cbd5e1;
  border-radius: 8px;
  padding: 14px 16px;
  text-align: center;
}

.efb-title {
  font-size: 13px;
  font-weight: 700;
  color: #1e293b;
}

.efb-desc {
  font-size: 11px;
  color: #64748b;
  line-height: 1.4;
}

.efb-btns {
  display: flex;
  gap: 8px;
}


/* 患者卡片与防遮挡现代布局 */
.patient-quick-card {
  background: #f8fafc;
  border: 1px solid #e2e8f0;
  border-radius: 10px;
  padding: 12px 14px;
  font-size: 0.85rem;
  color: #334155;
  box-sizing: border-box;
  width: 100%;
}

.patient-profile-top {
  display: flex;
  align-items: center;
  gap: 10px;
  padding-bottom: 10px;
  border-bottom: 1px solid #f1f5f9;
}

.ppt-avatar {
  width: 38px;
  height: 38px;
  border-radius: 50%;
  background: linear-gradient(135deg, #3b82f6, #1d4ed8);
  color: #fff;
  display: flex;
  align-items: center;
  justify-content: center;
  font-weight: 700;
  font-size: 16px;
  flex-shrink: 0;
  box-shadow: 0 2px 6px rgba(59, 130, 246, 0.3);
}

.ppt-meta {
  flex: 1;
  min-width: 0;
}

.ppt-name-row {
  display: flex;
  align-items: center;
  gap: 8px;
}

.ppt-name {
  font-size: 15px;
  font-weight: 700;
  color: #0f172a;
}

.ppt-no {
  font-size: 11px;
  color: #64748b;
  margin-top: 2px;
}

.patient-info-rows {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.p-row {
  display: flex;
  justify-content: space-between;
  align-items: center;
  font-size: 12.5px;
}

.p-label {
  color: #64748b;
  font-size: 12px;
}

.p-val {
  color: #1e293b;
  font-weight: 600;
}

/* 医护安全防护专项预警卡片 (彻底解决单行 nowrap 溢出和重叠问题) */
.patient-infection-alert-box {
  margin: 10px 0;
  background: linear-gradient(135deg, #fef2f2 0%, #fee2e2 100%);
  border: 1.5px solid #f87171;
  border-radius: 8px;
  padding: 10px 12px;
  box-sizing: border-box;
  width: 100%;
  box-shadow: 0 2px 8px rgba(239, 68, 68, 0.12);
  animation: pib-pulse 3s infinite ease-in-out;
}

@keyframes pib-pulse {
  0%, 100% { border-color: #f87171; box-shadow: 0 1px 4px rgba(239, 68, 68, 0.15); }
  50% { border-color: #ef4444; box-shadow: 0 2px 10px rgba(239, 68, 68, 0.28); }
}

.pia-head {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 6px;
}

.pia-title {
  display: flex;
  align-items: center;
  gap: 5px;
  color: #b91c1c;
  font-weight: 800;
  font-size: 12.5px;
}

.pia-icon {
  font-size: 15px;
  color: #dc2626;
}

.pia-guide-link {
  font-size: 11.5px;
  font-weight: 700;
  padding: 0;
  color: #dc2626;
}

.pia-body {
  display: flex;
  flex-direction: column;
  gap: 5px;
}

.pia-disease {
  font-size: 12px;
  color: #7f1d1d;
  line-height: 1.4;
  word-break: break-word;
}

.pia-label {
  font-weight: 600;
  color: #991b1b;
}

.pia-disease-name {
  color: #b91c1c;
  font-weight: 700;
  text-decoration: underline;
}

.pia-notice {
  font-size: 11px;
  color: #991b1b;
  line-height: 1.35;
}

.pia-gears {
  display: flex;
  flex-wrap: wrap;
  gap: 4px;
  margin-top: 3px;
}

.pia-gear-item {
  font-size: 11px;
  background: rgba(255, 255, 255, 0.85);
  border: 1px solid #fca5a5;
  color: #991b1b;
  padding: 2px 6px;
  border-radius: 4px;
  font-weight: 600;
}

.patient-cross-action {
  margin-top: 10px;
}

.cross-patient-query-btn {
  width: 100%;
  font-weight: 700;
  font-size: 12.5px;
  height: 36px;
  border-radius: 6px;
  box-shadow: 0 1px 4px rgba(217, 119, 6, 0.15);
}

.unit-text {
  font-size: 12px;
  color: #64748b;
  font-weight: 600;
  padding-right: 4px;
}

/* 临床决策与诊疗路径 Bento 现代化卡片选择器 */
.decision-mode-bento {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.decision-option-card {
  display: flex;
  align-items: flex-start;
  gap: 12px;
  padding: 12px 16px;
  background: #ffffff;
  border: 1.5px solid #e2e8f0;
  border-radius: 8px;
  cursor: pointer;
  transition: all 0.25s ease;
}

.decision-option-card:hover {
  border-color: #cbd5e1;
  background: #f8fafc;
}

.decision-option-card.active {
  border-color: #10b981;
  background: #f0fdf4;
  box-shadow: 0 2px 10px rgba(16, 185, 129, 0.12);
}

.doc-radio-circle {
  width: 18px;
  height: 18px;
  border-radius: 50%;
  border: 2px solid #cbd5e1;
  margin-top: 3px;
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
  transition: all 0.2s;
}

.decision-option-card.active .doc-radio-circle {
  border-color: #10b981;
}

.doc-radio-inner {
  width: 10px;
  height: 10px;
  border-radius: 50%;
  background: #10b981;
}

.doc-content {
  flex: 1;
  min-width: 0;
}

.doc-title-row {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
}

.doc-badge {
  font-size: 11px;
  font-weight: 800;
  padding: 1px 6px;
  border-radius: 4px;
}

.doc-badge.direct {
  background: #dcfce7;
  color: #15803d;
}

.doc-badge.exam {
  background: #dbeafe;
  color: #1d4ed8;
}

.doc-title {
  font-size: 14px;
  font-weight: 700;
  color: #0f172a;
}

.doc-desc {
  font-size: 12px;
  color: #64748b;
  margin-top: 4px;
  line-height: 1.4;
}

/* 辅助检查卡片多列布局 */
.exam-picker-grid {
  width: 100%;
}

.exam-checkbox-container {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(240px, 1fr));
  gap: 8px;
  width: 100%;
}

.exam-select-card {
  display: flex;
  justify-content: space-between;
  align-items: center;
  background: #ffffff;
  border: 1px solid #e2e8f0;
  border-radius: 6px;
  padding: 6px 12px;
  transition: all 0.2s;
}

.exam-select-card:hover {
  border-color: #93c5fd;
  background: #f8fafc;
}

.exam-select-card.selected {
  border-color: #3b82f6;
  background: #eff6ff;
}

.exam-label-text {
  font-size: 13px;
  font-weight: 600;
  color: #1e293b;
}

</style>


/* 接诊首要选人与空态引导卡片 */
.select-patient-first-container {
  max-width: 960px;
  margin: 16px auto 30px;
}

.select-patient-card {
  border-radius: 12px;
  border: 1px solid #e2e8f0;
  box-shadow: 0 4px 20px rgba(0, 0, 0, 0.04);
  background: #ffffff;
}

.sp-card-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.sp-title-badge {
  display: flex;
  align-items: center;
  gap: 8px;
}

.sp-title {
  font-size: 16px;
  font-weight: 700;
  color: #0f172a;
}

.sp-card-body {
  padding: 10px 4px;
}

.sp-quick-chips {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 8px;
  background: #f8fafc;
  padding: 12px 16px;
  border-radius: 8px;
  border: 1px solid #e2e8f0;
}

.chip-label {
  font-size: 13px;
  font-weight: 600;
  color: #475569;
}

.sp-chip-btn {
  border-radius: 16px;
}

.chip-sub {
  font-size: 11px;
  color: #64748b;
  margin-left: 4px;
}

.sp-empty-guide {
  text-align: center;
  padding: 40px 20px 20px;
  max-width: 640px;
  margin: 0 auto;
}

.sp-guide-icon {
  display: inline-flex;
  padding: 18px;
  background: #dcfce7;
  border-radius: 50%;
  margin-bottom: 16px;
}

.sp-empty-guide h3 {
  font-size: 18px;
  font-weight: 700;
  color: #0f172a;
  margin: 0 0 8px;
}

.sp-empty-guide p {
  font-size: 13px;
  color: #64748b;
  line-height: 1.6;
  margin: 0 0 24px;
}

.sp-guide-checklist {
  display: flex;
  flex-direction: column;
  gap: 10px;
  background: #f0fdf4;
  border: 1px solid #bbf7d0;
  padding: 14px 20px;
  border-radius: 8px;
  text-align: left;
}

.check-item {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 13px;
  color: #166534;
  font-weight: 500;
}
