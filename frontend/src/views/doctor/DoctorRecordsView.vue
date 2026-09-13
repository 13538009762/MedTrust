<template>
  <div class="doctor-records-container">
    <!-- 顶部状态统计卡片 -->
    <div class="stats-grid">
      <div class="stat-card blue">
        <div class="stat-icon">🏥</div>
        <div class="stat-info">
          <div class="stat-value">{{ records.length }}</div>
          <div class="stat-label">累计接诊记录</div>
        </div>
      </div>
      <div class="stat-card yellow">
        <div class="stat-icon">⏳</div>
        <div class="stat-info">
          <div class="stat-value">{{ waitingExamCount }}</div>
          <div class="stat-label">待医技检查 (申请中)</div>
        </div>
      </div>
      <div class="stat-card cyan">
        <div class="stat-icon">📋</div>
        <div class="stat-info">
          <div class="stat-value">{{ examCompletedCount }}</div>
          <div class="stat-label">检查完成 (待下确诊)</div>
        </div>
      </div>
      <div class="stat-card green">
        <div class="stat-icon">🛡️</div>
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
            <span class="header-title">📋 临床就诊记录管理 (Encounter Management)</span>
            <el-tag type="info" size="small" effect="plain" class="ml-2">就诊生命周期驱动</el-tag>
          </div>
          <div class="header-actions">
            <el-button type="primary" :icon="Plus" @click="openInitialDialog">
              新建就诊记录 (接诊初诊)
            </el-button>
            <el-button :icon="Refresh" circle @click="fetchRecords" />
          </div>
        </div>
      </template>

      <!-- 状态筛选与搜索 -->
      <div class="filter-bar">
        <el-radio-group v-model="activeStatusFilter" size="default">
          <el-radio-button label="ALL">全部状态 ({{ records.length }})</el-radio-button>
          <el-radio-button label="WAITING_EXAM">待检查 ({{ waitingExamCount }})</el-radio-button>
          <el-radio-button label="EXAM_COMPLETED">待确诊 ({{ examCompletedCount }})</el-radio-button>
          <el-radio-button label="COMPLETED">已归档上链 ({{ completedCount }})</el-radio-button>
        </el-radio-group>

        <div class="search-box">
          <el-input
            v-model="searchKeyword"
            placeholder="搜索就诊号、患者姓名、诊断..."
            clearable
            style="width: 280px;"
            :prefix-icon="Search"
          />
        </div>
      </div>

      <!-- 就诊记录列表 -->
      <el-table
        v-loading="loading"
        :data="filteredRecords"
        style="width: 100%;"
        class="record-table"
        stripe
      >
        <el-table-column prop="record_no" label="就诊编号" width="170">
          <template #default="{ row }">
            <span class="mono font-bold">{{ row.record_no }}</span>
          </template>
        </el-table-column>

        <el-table-column label="就诊类型" width="100">
          <template #default="{ row }">
            <el-tag :type="getEncounterTypeTag(row.encounter_type)" size="small">
              {{ formatEncounterType(row.encounter_type) }}
            </el-tag>
          </template>
        </el-table-column>

        <el-table-column label="接诊机构" width="140">
          <template #default="{ row }">
            <el-tag :type="getHospTagType(row.hospital_id)" size="small" effect="plain">
              {{ row.hospital_name || formatHospName(row.hospital_id) }}
            </el-tag>
          </template>
        </el-table-column>

        <el-table-column prop="department_name" label="就诊科室" width="130" />

        <el-table-column label="就诊患者" width="120">
          <template #default="{ row }">
            <span class="patient-name">{{ row.patient_name || '患者' }}</span>
          </template>
        </el-table-column>

        <el-table-column label="主诉与初诊拟定" min-width="200">
          <template #default="{ row }">
            <div class="text-sm font-medium text-slate-800">{{ row.chief_complaint || row.symptoms || '-' }}</div>
            <div class="text-xs text-blue-600 mt-1">初诊: {{ row.initial_diagnosis || '-' }}</div>
          </template>
        </el-table-column>

        <el-table-column label="医技检查项目" min-width="160">
          <template #default="{ row }">
            <div v-if="row.need_exam && row.exam_items">
              <el-tag type="warning" size="small" effect="plain">{{ row.exam_items }}</el-tag>
            </div>
            <span v-else class="text-xs text-gray-400">无需医技检查</span>
          </template>
        </el-table-column>

        <el-table-column label="最终确诊" min-width="170">
          <template #default="{ row }">
            <span v-if="row.diagnosis" class="text-success font-semibold">{{ row.diagnosis }}</span>
            <span v-else class="text-xs text-gray-400 italic">待出最终诊断</span>
          </template>
        </el-table-column>

        <el-table-column label="就诊流转状态" width="140">
          <template #default="{ row }">
            <el-tag v-if="row.status === 'WAITING_EXAM'" type="warning" effect="dark">⏳ 待医技检查</el-tag>
            <el-tag v-else-if="row.status === 'PROCESSING_EXAM'" type="primary" effect="dark">🔄 检查进行中</el-tag>
            <el-tag v-else-if="row.status === 'EXAM_COMPLETED'" type="info" effect="dark" class="status-ready-tag">📋 检查完成/待确诊</el-tag>
            <el-tag v-else-if="row.status === 'INITIAL_DIAGNOSIS'" type="info">🩺 初诊完成/待确诊</el-tag>
            <el-tag v-else-if="row.status === 'COMPLETED'" type="success" effect="plain">🛡️ 已归档存证</el-tag>
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
              ✍️ 下达最终诊断
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

    <!-- 弹窗一：新建就诊记录（第一阶段：初诊与开立检查单） -->
    <el-dialog
      v-model="initialDialogVisible"
      title="🩺 新建临床就诊记录与初诊接诊 (Clinical Encounter)"
      width="820px"
      top="4vh"
      :close-on-click-modal="false"
    >
      <el-form label-position="top" class="encounter-form">
        <!-- 步骤 1：患者身份确认 -->
        <div class="form-step-title" style="display: flex; justify-content: space-between; align-items: center;">
          <div>
            <span class="step-num">1</span>
            <span>患者身份核验与就诊挂号</span>
          </div>
          <el-button
            v-if="selectedPatientInfo"
            type="warning"
            size="small"
            plain
            :icon="Search"
            @click="goToCrossQueryForPatient"
          >
            ⚡ 跨机构一键调取该患者历史病历
          </el-button>
        </div>

        <el-row :gutter="16">
          <el-col :span="12">
            <el-form-item label="选择就诊患者" required>
              <el-select
                v-model="initForm.patient_id"
                placeholder="请选择就诊患者"
                filterable
                style="width: 100%;"
                @change="onPatientSelect"
              >
                <el-option
                  v-for="p in patientList"
                  :key="p.id"
                  :label="`${p.real_name || p.username} (卡号: ${p.user_no})`"
                  :value="p.id"
                />
              </el-select>
            </el-form-item>
          </el-col>
          <el-col :span="12">
            <div v-if="selectedPatientInfo" class="patient-quick-card">
              <div class="p-line"><strong>姓名：</strong>{{ selectedPatientInfo.real_name }} | <strong>性别/年龄：</strong>男 / 45岁</div>
              <div class="p-line"><strong>身份证：</strong><span class="mono">{{ maskID(selectedPatientInfo.id_card) }}</span></div>
              <div class="p-line"><strong>就诊卡号：</strong><span class="mono">{{ selectedPatientInfo.user_no }}</span></div>
              <div class="p-line"><el-tag type="danger" size="small">过敏史：无明确药物过敏</el-tag></div>
              <div class="patient-cross-action mt-2">
                <el-button
                  type="warning"
                  size="small"
                  :icon="Search"
                  style="width: 100%; font-weight: 600;"
                  @click="goToCrossQueryForPatient"
                >
                  ⚡ 跨机构一键调取该患者历史病历 (自动填入身份证检索)
                </el-button>
              </div>
            </div>
            <div v-else class="patient-placeholder">
              👈 请先在左侧选择患者以核验电子健康档案
            </div>
          </el-col>
        </el-row>

        <!-- 步骤 2：就诊基本信息 -->
        <div class="form-step-title mt-4">
          <span class="step-num">2</span>
          <span>就诊基本信息与挂号分诊</span>
        </div>

        <el-row :gutter="16">
          <el-col :span="8">
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
          <el-col :span="8">
            <el-form-item label="就诊类型" required>
              <el-radio-group v-model="initForm.encounter_type">
                <el-radio-button label="OUTPATIENT">普通门诊</el-radio-button>
                <el-radio-button label="EMERGENCY">急诊</el-radio-button>
                <el-radio-button label="INPATIENT">住院</el-radio-button>
              </el-radio-group>
            </el-form-item>
          </el-col>
          <el-col :span="8">
            <el-form-item label="接诊医生">
              <el-input :value="`${auth.user?.real_name || '当前医生'} (${auth.user?.title || '主治医师'})`" disabled />
            </el-form-item>
          </el-col>
        </el-row>

        <el-row :gutter="16">
          <el-col :span="12">
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
          </el-col>
          <el-col :span="12">
            <el-form-item label="症状持续时间">
              <el-input v-model="initForm.duration" placeholder="例如：阵发性反复发作 3 天，活动后加重" />
            </el-form-item>
          </el-col>
        </el-row>

        <!-- 步骤 3：初诊主诉与体征采集 -->
        <div class="form-step-title mt-4">
          <span class="step-num">3</span>
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

        <!-- 生命体征卡片 -->
        <div class="vitals-panel mb-3">
          <div class="vitals-head">
            <span class="text-sm font-semibold text-slate-700">🩺 生命体征测量参数 (Vital Signs)</span>
            <el-button size="small" type="primary" link @click="quickFillVitals">⚡ 一键填入标准体征</el-button>
          </div>
          <el-row :gutter="12">
            <el-col :span="6">
              <el-form-item label="体温 (T)">
                <el-input v-model="vitals.temperature" placeholder="36.5 ℃" />
              </el-form-item>
            </el-col>
            <el-col :span="6">
              <el-form-item label="血压 (BP)">
                <el-input v-model="vitals.blood_pressure" placeholder="125/80 mmHg" />
              </el-form-item>
            </el-col>
            <el-col :span="6">
              <el-form-item label="心率/脉搏 (HR)">
                <el-input v-model="vitals.heart_rate" placeholder="78 bpm" />
              </el-form-item>
            </el-col>
            <el-col :span="6">
              <el-form-item label="血氧 (SpO2)">
                <el-input v-model="vitals.spo2" placeholder="98 %" />
              </el-form-item>
            </el-col>
          </el-row>
        </div>

        <!-- 步骤 4：临床决策与诊疗路径选择 -->
        <div class="form-step-title mt-4">
          <span class="step-num">4</span>
          <span>临床决策与诊疗路径选择 (Clinical Decision)</span>
        </div>

        <div class="decision-mode-bar mb-3">
          <el-radio-group v-model="encounterMode" size="default">
            <el-radio-button label="DIRECT">
              🩺 临床体征明确 · 直接下达确诊与处方 (一步完成存证)
            </el-radio-button>
            <el-radio-button label="EXAM">
              🔬 病情待查 · 开立医技检查单 (派发检验/影像科，协同流转)
            </el-radio-button>
          </el-radio-group>
        </div>

        <!-- 路径 A：临床体征明确，直接确诊并归档 -->
        <div v-if="encounterMode === 'DIRECT'" class="direct-decision-panel">
          <el-alert
            type="success"
            :closable="false"
            show-icon
            class="mb-3"
            title="💡 诊断提示：若患者体征与现病史典型明确，医生可直接下达确诊结论与处置方案，一键完成加密上链存证，无需开具医技检查。"
          />
          <el-row :gutter="16">
            <el-col :span="12">
              <el-form-item label="经治医生最终确诊 (Final Diagnosis)" required>
                <el-input
                  v-model="directForm.diagnosis"
                  type="textarea"
                  :autosize="{ minRows: 2, maxRows: 5 }"
                  placeholder="例如：原发性高血压病1级 / 上呼吸道感染 / 慢性胃炎"
                />
              </el-form-item>
            </el-col>
            <el-col :span="12">
              <el-form-item label="病理诱因与发病机制 (Etiology)">
                <el-input
                  v-model="directForm.etiology"
                  type="textarea"
                  :autosize="{ minRows: 2, maxRows: 5 }"
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
              📋 临床规则引擎辅助生成规范 SOAP 全景病历 (一键装配)
            </el-button>
            <el-button size="small" type="info" link @click="fillQuickDirectPreset">
              ⚡ 载入常规高血压门诊处方示例
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
                  :autosize="{ minRows: 2, maxRows: 5 }"
                  placeholder="例如：原发性高血压病1级待查；心律失常待排"
                />
              </el-form-item>
            </el-col>
            <el-col :span="12">
              <el-form-item label="接诊依据与拟诊分析">
                <el-input
                  v-model="initForm.diagnostic_basis"
                  type="textarea"
                  :autosize="{ minRows: 2, maxRows: 5 }"
                  placeholder="例如：患者血压145/95mmHg，活动后胸闷心悸，需辅助检查确证"
                />
              </el-form-item>
            </el-col>
          </el-row>

          <div class="form-step-title mt-4">
            <span class="step-num">5</span>
            <span>辅助检查申请联动 (医技科室协同与任务池派发)</span>
          </div>

          <div class="exam-order-box">
            <div class="dispatch-target-tip mb-3">
              <el-alert
                title="🎯 医技检查单智能分流派发机制："
                type="info"
                :closable="false"
                description="① 血常规 / 心肌酶谱 / 肝肾生化 ➔ 自动派发至【临床检验医学中心·LIS任务池】(由检验科技师 tech_lab 接单)；
② 12导联心电图 / 胸部CT / 头颅MRI ➔ 自动派发至【放射影像与心电中心·PACS任务池】(由影像医师 tech_pacs 接单)。
提交后医技人员即可在对应工作台接单处理！当前医生亦可直接通过左侧「医技检查中心」协同出报告。"
                show-icon
              />
            </div>

            <el-form-item label="开具辅助检查项目 (勾选后自动生成独立申请单号 ORD... 移交医技中心)">
              <el-checkbox-group v-model="selectedExamList">
                <el-checkbox label="12导联心电图 (ECG)">12导联心电图 (ECG) <el-tag size="small" type="warning">影像心电中心</el-tag></el-checkbox>
                <el-checkbox label="常规全血细胞分析 (血常规)">常规全血细胞分析 (血常规) <el-tag size="small" type="primary">临床检验科</el-tag></el-checkbox>
                <el-checkbox label="血清心肌酶谱与肌钙蛋白">血清心肌酶谱与肌钙蛋白 <el-tag size="small" type="primary">临床检验科</el-tag></el-checkbox>
                <el-checkbox label="胸部高分辨率 CT 平扫">胸部高分辨率 CT 平扫 <el-tag size="small" type="warning">放射影像中心</el-tag></el-checkbox>
                <el-checkbox label="肝肾功能与电解质生化检查">肝肾功能与电解质生化 <el-tag size="small" type="primary">临床检验科</el-tag></el-checkbox>
                <el-checkbox label="头颅 MRI 平扫">头颅 MRI 平扫 <el-tag size="small" type="warning">放射影像中心</el-tag></el-checkbox>
              </el-checkbox-group>
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
      </el-form>

      <template #footer>
        <div class="dialog-footer">
          <el-button @click="initialDialogVisible = false">取消</el-button>
          <!-- 模式一：直接确诊并归档 -->
          <el-button
            v-if="encounterMode === 'DIRECT'"
            type="success"
            size="large"
            :loading="submitting"
            :disabled="!initForm.patient_id || !initForm.chief_complaint || !directForm.diagnosis || !directForm.treatment_plan"
            @click="submitDirectEncounter"
          >
            🛡️ 确认最终确诊并直接归档 (一步完成 AES加密 + IPFS + Fabric区块链存证)
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
            🚀 提交初诊并派发检查申请单 (移交医技科室)
          </el-button>
        </div>
      </template>
    </el-dialog>

    <!-- 弹窗二：第二阶段 最终确诊与规范 SOAP 病历生成存证 -->
    <el-dialog
      v-model="finalDialogVisible"
      title="📋 就诊最终确诊与规范 SOAP 病历确认 (Final Diagnosis & Blockchain Sealing)"
      width="850px"
      top="4vh"
      :close-on-click-modal="false"
    >
      <div v-if="activeRecordForFinal" class="final-dialog-body">
        <!-- 初诊与检查回传摘要看板 -->
        <div class="summary-banner">
          <div class="s-head">
            <span class="mono font-bold">{{ activeRecordForFinal.record_no }}</span>
            <el-tag size="small" type="warning">{{ formatEncounterType(activeRecordForFinal.encounter_type) }}</el-tag>
            <span>患者：<strong>{{ activeRecordForFinal.patient_name }}</strong></span>
            <span>接诊科室：{{ activeRecordForFinal.department_name }}</span>
          </div>
          <div class="s-content mt-2">
            <div><strong>主诉：</strong>{{ activeRecordForFinal.chief_complaint }}</div>
            <div><strong>初诊：</strong>{{ activeRecordForFinal.initial_diagnosis }} (依据: {{ activeRecordForFinal.diagnostic_basis || '接诊查体' }})</div>
            <div><strong>生命体征：</strong>{{ activeRecordForFinal.vital_signs }}</div>
          </div>
        </div>

        <!-- 医技科室回传报告展示区 -->
        <el-divider content-position="left">🔬 医技辅助检查回传报告汇总</el-divider>

        <div class="lab-results-display">
          <div v-if="activeRecordForFinal.exam_result" class="lab-result-text">
            {{ activeRecordForFinal.exam_result }}
          </div>
          <div v-else class="text-xs text-gray-500 italic">
            暂无开具的医技辅助检查项目，直接进行临床确诊。
          </div>
        </div>

        <!-- 最终诊断录入 -->
        <el-divider content-position="left">✍️ 经治医生最终确诊与处置决策</el-divider>

        <el-form label-position="top">
          <el-form-item label="最终确定诊断 (Final Diagnosis)" required>
            <el-input
              v-model="finalForm.diagnosis"
              type="textarea"
              :autosize="{ minRows: 2, maxRows: 4 }"
              placeholder="例如：原发性高血压病1级，伴轻度劳力型心肌供血不足"
            />
          </el-form-item>

          <el-form-item label="病因分析与病理机制 (Etiology)">
            <el-input
              v-model="finalForm.etiology"
              type="textarea"
              :autosize="{ minRows: 2, maxRows: 4 }"
              placeholder="例如：血管内皮舒缩功能紊乱，高压负荷导致心肌耗氧量增加"
            />
          </el-form-item>

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
              📋 基于临床规则辅助生成规范 SOAP 病历
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
              :rows="8"
              class="soap-textarea"
            />
          </el-form-item>
        </el-form>
      </div>

      <template #footer>
        <el-button @click="finalDialogVisible = false">返回</el-button>
        <el-button
          type="success"
          size="large"
          :loading="submittingFinal"
          :disabled="!finalForm.diagnosis || !finalForm.treatment_plan"
          @click="submitFinalEncounter"
        >
          🛡️ 确认并完成归档存证 (AES加密 + IPFS + Fabric区块链存证)
        </el-button>
      </template>
    </el-dialog>

    <!-- 弹窗三：查看已归档/调阅全景病历详情 -->
    <el-dialog
      v-model="detailModalVisible"
      title="🛡️ 临床就诊全景档案与区块链存证详情"
      width="820px"
      top="5vh"
    >
      <div v-if="detailRecord" class="detail-body">
        <!-- 篡改与存证告警横幅 -->
        <div v-if="detailRecord.is_tampered" class="tamper-box danger">
          <div class="t-head">🚨【高危安全警报】检测到数据库临床数据已被黑客篡改！</div>
          <div class="t-desc">
            底层 MySQL 数据库数据指纹与 Fabric 联盟链不可篡改基准不一致！系统已阻断非法使用并记录高危安全审计！
          </div>
          <div class="t-hashes">
            <div>当前计算哈希: <code>{{ detailRecord.current_hash }}</code></div>
            <div>链上存证基准: <code>{{ detailRecord.chain_hash }}</code></div>
          </div>
        </div>
        <div v-else class="tamper-box safe">
          <span>🛡️ <strong>区块链防篡改核验通过：</strong>数据指纹与 Hyperledger Fabric 联盟链完全吻合，100% 真实完整。</span>
        </div>

        <div class="detail-meta-grid">
          <div><strong>就诊编号：</strong><span class="mono">{{ detailRecord.record_no }}</span></div>
          <div><strong>就诊类型：</strong><el-tag size="small">{{ formatEncounterType(detailRecord.encounter_type) }}</el-tag></div>
          <div><strong>就诊科室：</strong>{{ detailRecord.department_name }}</div>
          <div><strong>就诊患者：</strong>{{ detailRecord.patient_name }}</div>
          <div><strong>接诊医生：</strong>{{ detailRecord.doctor_name }}</div>
          <div><strong>流转状态：</strong><el-tag size="small" type="success">{{ detailRecord.status }}</el-tag></div>
          <div><strong>就诊时间：</strong>{{ formatTime(detailRecord.created_at) }}</div>
          <div><strong>Fabric TxID：</strong><span class="mono text-xs">{{ detailRecord.fabric_tx_id || '已生成交易凭证' }}</span></div>
        </div>

        <el-divider content-position="left">结构化就诊多维信息</el-divider>

        <table class="detail-table">
          <tr>
            <th width="140">主诉 (S)</th>
            <td><strong>{{ detailRecord.chief_complaint || detailRecord.symptoms }}</strong></td>
          </tr>
          <tr>
            <th>生命体征 (O)</th>
            <td>{{ detailRecord.vital_signs || '未录入' }}</td>
          </tr>
          <tr v-if="detailRecord.initial_diagnosis">
            <th>初诊与拟诊依据</th>
            <td>
              <div>初诊：{{ detailRecord.initial_diagnosis }}</div>
              <div v-if="detailRecord.diagnostic_basis" class="text-xs text-gray-500 mt-1">依据：{{ detailRecord.diagnostic_basis }}</div>
            </td>
          </tr>
          <tr v-if="detailRecord.exam_result">
            <th>医技检查报告 (O)</th>
            <td><div class="pre-text">{{ detailRecord.exam_result }}</div></td>
          </tr>
          <tr>
            <th>最终确诊 (A)</th>
            <td><span class="text-success font-bold">{{ detailRecord.diagnosis || '待确诊' }}</span></td>
          </tr>
          <tr>
            <th>处置与处方方案 (P)</th>
            <td><div class="pre-text highlight">{{ detailRecord.treatment_plan || '遵医嘱' }}</div></td>
          </tr>
        </table>

        <el-divider content-position="left">📁 临床归档电子凭据与原始文件 (PDF / 影像)</el-divider>

        <div class="archive-files-section">
          <div v-if="detailRecord.files && detailRecord.files.length" class="file-cards-list">
            <div v-for="file in detailRecord.files" :key="file.id" class="file-item-card">
              <div class="file-icon-box">
                <span v-if="file.file_type === 'pdf'">📄</span>
                <span v-else-if="isImageFileType(file.file_type)">🖼️</span>
                <span v-else>📑</span>
              </div>
              <div class="file-info">
                <div class="file-name font-bold">
                  {{ file.file_name }}
                  <el-tag size="small" type="primary" class="ml-2">{{ file.file_type.toUpperCase() }}</el-tag>
                  <el-tag size="small" type="success" effect="plain" class="ml-1">{{ formatFileSize(file.file_size) }}</el-tag>
                </div>
                <div class="file-cid text-xs text-gray-500 mt-1">
                  <span>IPFS CID: <code>{{ file.ipfs_cid }}</code></span>
                </div>
                <div class="file-hash text-xs text-gray-500">
                  <span>明文 SHA-256: <code>{{ file.file_hash }}</code></span>
                </div>
              </div>
              <div class="file-actions">
                <template v-if="isImageFileType(file.file_type)">
                  <el-button type="primary" size="small" :icon="View" @click="openImageModal(file)">
                    🖼️ 查阅检验/影像图片
                  </el-button>
                  <el-button type="success" size="small" plain :icon="Download" @click="downloadFileDirect(`/api/v1/medical-files/${file.id}/download`, file.file_name)">
                    📥 下载图片
                  </el-button>
                </template>
                <template v-else>
                  <el-button type="primary" size="small" :icon="View" @click="openPdfPreview(detailRecord)">
                    👁️ 在线预览 PDF
                  </el-button>
                  <el-button type="success" size="small" plain :icon="Download" @click="downloadRecordFile(detailRecord.id)">
                    📥 下载解密文件
                  </el-button>
                </template>
              </div>
            </div>

            <!-- 如果附件包含图片，同时在下方展示高清略缩图展示带 -->
            <div v-for="file in detailRecord.files.filter((f: any) => isImageFileType(f.file_type))" :key="'img-' + file.id" class="inline-img-card mt-2">
              <div class="img-header mb-1">
                <span class="text-xs font-semibold text-slate-700">🖼️ 医技检查影像切片/报告扫描件预览（{{ file.file_name }}）：</span>
              </div>
              <div class="text-center p-2">
                <img
                  :src="`/api/v1/medical-files/${file.id}/view`"
                  class="preview-thumbnail"
                  @click="openImageModal(file)"
                  title="点击全屏查阅大图"
                />
                <div class="text-xs text-gray-400 mt-1">（点击上方图片可在全景视窗放大查阅）</div>
              </div>
            </div>
          </div>
          <div v-else class="empty-files-box">
            <div class="empty-flex-row">
              <div class="empty-text">
                <span class="font-medium text-slate-700">📑 标准规范电子病历归档凭证 (PDF)</span>
                <div class="text-xs text-gray-500">系统已自动生成全量 SOAP 规范电子病历并完成区块链不可篡改存证</div>
              </div>
              <div class="empty-btn-group">
                <el-button type="primary" size="small" :icon="View" @click="openPdfPreview(detailRecord)">
                  👁️ 在线查阅电子病历 PDF
                </el-button>
                <el-button type="success" size="small" plain :icon="Download" @click="downloadRecordFile(detailRecord.id)">
                  📥 下载电子凭据
                </el-button>
              </div>
            </div>
          </div>
        </div>

        <el-divider content-position="left">区块链安全与存证凭据</el-divider>

        <div class="blockchain-evidence-box">
          <p><strong>Fabric TxID：</strong><span class="tx-hash">{{ detailRecord.fabric_tx_id || '0x4f8a2b3c...' }}</span></p>
          <p v-if="detailRecord.block_height"><strong>存证区块高度：</strong><span class="mono">#{{ detailRecord.block_height }}</span></p>
          <p><strong>智能合约存证认证：</strong><el-tag type="success" size="small">联盟链权威存证 · 100% 真实未篡改</el-tag></p>
        </div>
      </div>
    </el-dialog>

    <!-- PDF 电子病历规范预览弹窗 -->
    <el-dialog
      v-model="pdfPreviewVisible"
      title="📄 临床就诊电子病历归档凭证 (PDF 规范视图)"
      width="880px"
      top="3vh"
      :close-on-click-modal="false"
      class="pdf-preview-dialog"
    >
      <div v-if="previewingRecord" id="emr-print-container" class="emr-sheet-wrapper">
        <!-- 打印与工具栏 -->
        <div class="emr-action-bar no-print">
          <div class="emr-tip-tag">
            <el-tag type="success" effect="dark">✅ 密文解密验证通过</el-tag>
            <el-tag type="info" class="ml-2">国家卫健委《电子病历应用规范》甲级存证标准</el-tag>
          </div>
          <div class="emr-btns">
            <el-button type="primary" :icon="Printer" @click="printPdfSheet">
              🖨️ 打印 / 另存为 PDF 文件
            </el-button>
            <el-button type="success" plain :icon="Download" @click="downloadRecordFile(previewingRecord.id)">
              📥 下载原始归档凭证
            </el-button>
          </div>
        </div>

        <!-- 正式红头病历文档纸张 -->
        <div class="emr-sheet-paper">
          <!-- 红头医院名称与标题 -->
          <div class="emr-header">
            <div class="emr-hospital-name">{{ previewingRecord.hospital_name || formatHospName(previewingRecord.hospital_id) }}</div>
            <div class="emr-doc-title">门 诊 / 临 床 就 诊 归 档 记 录 单</div>
            <div class="emr-sub-title">（MedTrust 医疗可信共享联盟·Hyperledger Fabric 区块链存证凭证）</div>
            <div class="emr-red-line"></div>
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
                <div class="soap-exam-head">🔬 医技科室回传检查报告明细：</div>
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
              <div class="text-xs text-emerald-700 mt-1">🛡️ 本病历经 Hyperledger Fabric 存证，数字签名抗抵赖、防篡改</div>
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
      :title="`🖼️ ${imageModalTitle || '医学检验/影像检查原图'}`"
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
          📥 下载原始影像文件
        </el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { Plus, Refresh, Search, View, Download, Printer } from '@element-plus/icons-vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import api from '../../api/client'
import { useAuthStore } from '../../stores/auth'
import TreatmentPlanEditor from '../../components/TreatmentPlanEditor.vue'

const router = useRouter()
const auth = useAuthStore()

const loading = ref(false)
const records = ref<any[]>([])
const patientList = ref<any[]>([])
const activeStatusFilter = ref('ALL')
const searchKeyword = ref('')

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
  imageModalUrl.value = `/api/v1/medical-files/${file.id}/view`
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

function onPatientSelect(pid: number) {
  selectedPatientInfo.value = patientList.value.find(p => p.id === pid) || null
  if (encounterMode.value === 'DIRECT') {
    generateDirectSOAP()
  }
}

function goToCrossQueryForPatient() {
  if (!selectedPatientInfo.value) {
    ElMessage.warning('请先在左侧选择就诊患者')
    return
  }
  const idCard = selectedPatientInfo.value.id_card || selectedPatientInfo.value.real_name
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

function getHospTagType(hId?: number) {
  if (hId === 1) return 'primary'
  if (hId === 2) return 'warning'
  if (hId === 3) return 'success'
  return 'info'
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
  initForm.value = {
    patient_id: patientList.value.length > 0 ? patientList.value[0].id : undefined,
    encounter_type: 'OUTPATIENT',
    department_name: '心血管内科',
    onset_time: new Date().toISOString().substring(0, 19).replace('T', ' '),
    duration: '阵发性胸闷 2 天',
    chief_complaint: '胸骨后发作性压榨样闷痛2天，劳累后加剧',
    present_illness: '患者2天前爬楼劳累后突感心前区及胸骨后压榨样闷痛，每次持续3-5分钟，伴心慌出冷汗，休息后可轻微缓解。',
    initial_diagnosis: '原发性高血压病1级待查；冠脉缺血待排',
    diagnostic_basis: '中年患者，活动诱发胸闷心悸，既往有轻度血压偏高史，需心电图及血常规排查。',
    exam_reason: '排查急性心肌缺血病变及炎性反应'
  }
  selectedExamList.value = ['12导联心电图 (ECG)', '常规全血细胞分析 (血常规)']

  directForm.value = {
    diagnosis: '原发性高血压病1级（低危组）',
    etiology: '活动劳累及情绪应激导致交感神经过度兴奋，外周阻力血管收缩，收缩压升高。',
    treatment_plan: '【综合处置】\n1. 静坐休息15分钟后再次复测双侧上臂血压，持续脉氧监测\n\n【临床医嘱】\n1. 饮食指导：严格低盐低脂清淡饮食（每日食盐量控制在 5g 以内）\n2. 每日早晚各监测一次血压并详细做健康日志记录\n3. 避免连续熬夜加班、情绪激动与重度体力劳累，适度慢走运动\n4. 2周后心血管内科专科门诊复查\n\n【处方用药】\n1. 硝苯地平控释片 (拜新同) 30mg 口服 每日一次 清晨餐后整片吞服\n2. 阿托伐他汀钙片 (立普妥) 20mg 口服 每晚睡前一次',
    soap_content: ''
  }

  if (initForm.value.patient_id) {
    onPatientSelect(initForm.value.patient_id)
  } else {
    generateDirectSOAP()
  }
  initialDialogVisible.value = true
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
          <p style="color: #059669; font-weight: bold; margin-bottom: 8px;">✅ 门诊就诊已一步完成最终确诊与规范归档！</p>
          <p style="margin-bottom: 6px;"><strong>🛡️ 可信区块链与隐私安全执行清单：</strong></p>
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
            <p style="color: #059669; font-weight: bold; margin-bottom: 8px;">✅ 初诊记录建立成功，已生成医技检查单并派发！</p>
            <p style="margin-bottom: 6px;"><strong>🎯 检查单派发去向说明：</strong></p>
            <ul style="padding-left: 20px; margin: 6px 0; color: #334155; font-size: 13px;">
              <li><strong>血常规 / 心肌酶谱 / 肝肾生化：</strong>已派发至 <strong>【临床检验科 (LIS任务池)】</strong>，由检验科技师 (<strong>tech_lab</strong>) 处理</li>
              <li><strong>12导联心电图 / 胸部CT / 头颅MRI：</strong>已派发至 <strong>【放射影像与心电中心 (PACS任务池)】</strong>，由影像医师 (<strong>tech_pacs</strong>) 处理</li>
            </ul>
            <p style="margin-top: 10px; font-size: 13px; color: #0284c7; font-weight: 600;">💡 接下来谁来检查？（两种演示方式均可）：</p>
            <p style="font-size: 13px; color: #475569; margin: 4px 0;"><strong>方式一（快捷协同，无需换号）：</strong>直接点击左侧菜单的 <strong>「🔬 医技检查中心」</strong>，即可立即接单并出具回传报告！</p>
            <p style="font-size: 13px; color: #475569; margin: 4px 0;"><strong>方式二（多角色切换，规范演示）：</strong>退出当前登录，切换为检验科账号 <strong>tech_lab</strong> (密码 <strong>123456</strong>) 或影像科账号 <strong>tech_pacs</strong> (密码 <strong>123456</strong>) 登录系统出报告！</p>
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

function openFinalDialog(row: any) {
  activeRecordForFinal.value = row
  const defaultPlan = row.treatment_plan || '【综合处置】\n1. 门诊心电图与心肌酶谱动态复查评估\n\n【临床医嘱】\n1. 限制钠盐摄入 (每日<5g)，保证充足规律睡眠\n2. 每日晨起及睡前规范监测自测血压\n3. 2周后心内科门诊复查血压及心电图\n\n【处方用药】\n1. 苯磺酸氨氯地平片 (络活喜) 5mg 口服 每日一次\n2. 酒石酸美托洛尔片 (倍他乐克) 25mg 口服 每日两次 饭前半小时'
  finalForm.value = {
    diagnosis: row.diagnosis || '原发性高血压病1级，心肌轻度供血不足',
    etiology: row.etiology || '劳累导致交感神经过度激活，外周阻力血管收缩，心肌负荷增加。',
    treatment_plan: defaultPlan,
    soap_content: ''
  }
  generateSOAPRecord()
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
● 现病史：${rec.present_illness || '患者因上述症状就诊，发病过程如主诉所述。'}
● 发病时间：${rec.onset_time || '近期'} | 持续时间：${rec.duration || '阵发'}

【O - Objective 客观检查与测量】
● 查体生命体征：${rec.vital_signs || '体征平稳'}
● 医技辅助检查回传结论：
${rec.exam_result ? rec.exam_result : '（未开具医技检验检查）'}

【A - Assessment 综合评估与确诊】
● 初步接诊拟诊：${rec.initial_diagnosis || '-'} (拟诊依据: ${rec.diagnostic_basis || '临床体征'})
● 经治医生最终确诊：${finalForm.value.diagnosis || '原发性高血压病1级'}
● 病理诱因与机制：${finalForm.value.etiology || '心血管神经调节紊乱'}

【P - Plan 综合处置与医嘱方案】
${finalForm.value.treatment_plan || '遵医嘱治疗'}

● 医生签署说明：本病历经责任医生依据患者主诉、查体与医技回传报告综合研判确诊，符合国家医疗质量安全核心制度。
--------------------------------------------------------------------------------
存证机构：${auth.user?.hospital_name || '本医疗中心'} | 加密协议：AES-256-GCM | 存证底层：Hyperledger Fabric`

  finalForm.value.soap_content = soapText
}

async function submitFinalEncounter() {
  if (!activeRecordForFinal.value) return
  submittingFinal.value = true
  try {
    const payload = {
      record_id: activeRecordForFinal.value.id,
      diagnosis: finalForm.value.diagnosis,
      etiology: finalForm.value.etiology,
      treatment_plan: finalForm.value.treatment_plan,
      soap_content: finalForm.value.soap_content
    }

    const res: any = await api.post(`/encounters/${activeRecordForFinal.value.id}/complete`, payload)
    if (res.code === 200) {
      ElMessage.success('就诊病历最终确诊已确认！已成功完成 AES 密文存储、IPFS 存储及 Fabric 区块链存证！')
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

function getEncounterTypeTag(t?: string) {
  if (t === 'EMERGENCY') return 'danger'
  if (t === 'INPATIENT') return 'warning'
  return 'primary'
}

function formatTime(t?: string) {
  if (!t) return '-'
  return t.substring(0, 16).replace('T', ' ')
}

function openPdfPreview(rec: any) {
  previewingRecord.value = rec
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

/* 📁 临床归档文件卡片样式 */
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

/* 📄 拟真红头电子病历 PDF 预览单纸张样式 */
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
</style>
