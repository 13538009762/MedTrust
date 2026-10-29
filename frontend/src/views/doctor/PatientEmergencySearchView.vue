<template>
  <div class="patient-emergency-container">
    <!-- 顶部检索区与快速定位 -->
    <div class="search-header-card">
      <div class="header-content">
        <div class="title-row">
          <div class="title-badge">
            <el-icon class="title-icon"><FirstAidKit /></el-icon>
            <h2>患者全景检索与急救调阅中心</h2>
          </div>
          <el-tag type="danger" effect="dark" class="emergency-mode-tag">
            <span class="live-pulse"></span> 临床急救响应就绪
          </el-tag>
        </div>
        <p class="subtitle">
          面向跨机构临床协同与急诊抢救：快速定位患者全景档案，秒级调阅致命药物过敏与生命体征，全览已授权就诊记录，支持一键申请全部权限、分批申请及急救破窗绿色通道。
        </p>

        <!-- 检索栏与候选患者芯片 -->
        <div class="search-controls-row">
          <div class="search-input-group">
            <el-input
              v-model="searchKeyword"
              placeholder="输入患者姓名、身份证号、手机号或患者编号检索..."
              size="large"
              clearable
              :prefix-icon="Search"
              class="patient-search-input"
              @keyup.enter="handleSearch"
            >
              <template #append>
                <el-button type="primary" :loading="searching" @click="handleSearch">
                  定位患者
                </el-button>
              </template>
            </el-input>
          </div>
        </div>
      </div>
    </div>

    <!-- 若未选定患者时的空态指引 -->
    <div v-if="!selectedPatient && !searching" class="empty-patient-guide">
      <el-empty description="输入患者姓名、身份证号、手机号或患者编号定位患者">
        <template #image>
          <div class="empty-icon-wrap">
            <el-icon :size="64" color="#059669"><Search /></el-icon>
          </div>
        </template>
        <div class="guide-features">
          <div class="guide-item">
            <el-icon class="g-icon green"><CircleCheck /></el-icon>
            <span>秒级穿透全网医联体已授权病历</span>
          </div>
          <div class="guide-item">
            <el-icon class="g-icon red"><WarningFilled /></el-icon>
            <span>抢救核心：ABO血型、药物过敏与生命体征速查</span>
          </div>
          <div class="guide-item">
            <el-icon class="g-icon blue"><Key /></el-icon>
            <span>一键申请全量知情同意或分批勾选申请</span>
          </div>
          <div class="guide-item">
            <el-icon class="g-icon red"><FirstAidKit /></el-icon>
            <span>极危重症急救绿色通道（Break-Glass 存证上链）</span>
          </div>
        </div>
      </el-empty>
    </div>

    <!-- 当已选中目标患者时：展示 3 层完整界面 -->
    <div v-else-if="selectedPatient" class="patient-main-content">
      <!-- ================= 界面上层：病人基础信息与急救生命画像 ================= -->
      <div class="tier-upper-patient-profile">
        <!-- 患者身份总览条 -->
        <div class="profile-hero-strip">
          <div class="profile-avatar-box">
            <div class="avatar-circle" :class="selectedPatient.gender === '女' ? 'female' : 'male'">
              <span class="avatar-char">{{ selectedPatient.real_name ? selectedPatient.real_name.charAt(0) : '患' }}</span>
            </div>
          </div>

          <div class="profile-main-meta">
            <div class="name-row">
              <span class="patient-name">{{ selectedPatient.real_name }}</span>
              <el-tag :type="selectedPatient.gender === '女' ? 'danger' : 'primary'" effect="plain" class="meta-tag">
                {{ selectedPatient.gender || '男' }}
              </el-tag>
              <el-tag type="info" effect="plain" class="meta-tag">
                {{ selectedPatient.age }} 岁
              </el-tag>
              <el-tag type="success" effect="plain" class="meta-tag">
                医保参保正常
              </el-tag>
              <span class="patient-no-badge">ID: {{ selectedPatient.user_no || `PAT_${selectedPatient.id}` }}</span>
            </div>

            <div class="identity-info-row">
              <span class="info-item">
                <el-icon><Postcard /></el-icon>
                <span>身份证号：</span>
                <code class="code-val">{{ selectedPatient.id_card }}</code>
                <el-button link type="primary" size="small" @click="copyText(selectedPatient.id_card, '身份证号')">复制</el-button>
              </span>
              <span class="info-sep">|</span>
              <span class="info-item">
                <el-icon><Phone /></el-icon>
                <span>本人电话：</span>
                <span class="val">{{ selectedPatient.phone }}</span>
                <el-button link type="primary" size="small" @click="copyText(selectedPatient.phone, '联系电话')">复制</el-button>
              </span>
              <span class="info-sep">|</span>
              <span class="info-item">
                <el-icon><Lock /></el-icon>
                <span>授权密钥状态：</span>
                <el-tag size="small" type="success">已由患者预设生效 (支持现场核验)</el-tag>
              </span>
            </div>
          </div>

          <div class="profile-actions-right">
            <el-button size="small" :icon="Refresh" @click="refreshPatientRecords" :loading="loadingRecords">
              刷新病历
            </el-button>
          </div>
        </div>

        <!-- 急救黄金决策 Bento 矩阵 (6大维度生命保障) -->
        <div class="emergency-bento-grid">
          <!-- 1. ABO / Rh 血型 (特大醒目红标) -->
          <div class="bento-card blood-card">
            <div class="bento-header">
              <span class="bento-icon blood"><el-icon><FirstAidKit /></el-icon></span>
              <span class="bento-title">ABO / Rh 紧急血型</span>
            </div>
            <div class="bento-body">
              <div class="blood-type-display">
                <span class="blood-big">{{ selectedPatient.blood_type || 'O型 (Rh阳性)' }}</span>
              </div>
              <p class="bento-note">急救输血配型核心依据 · 链上历史验血存证</p>
            </div>
          </div>

          <!-- 2. 致命药物过敏史 (高危预警) -->
          <div class="bento-card allergy-card" :class="hasRealAllergy(selectedPatient) ? 'has-danger' : 'is-safe'">
            <div class="bento-header">
              <span class="bento-icon allergy" :class="hasRealAllergy(selectedPatient) ? 'danger' : 'safe'">
                <el-icon><Warning v-if="hasRealAllergy(selectedPatient)" /><CircleCheckFilled v-else /></el-icon>
              </span>
              <span class="bento-title">用药禁忌与致命过敏史</span>
              <span v-if="hasRealAllergy(selectedPatient)" class="danger-pill">严密防范</span>
              <span v-else class="safe-pill">未见药物过敏</span>
            </div>
            <div class="bento-body">
              <div v-if="hasRealAllergy(selectedPatient)" class="allergy-content-box">
                <div class="allergy-text font-bold">
                  ⚠️ {{ selectedPatient.allergies }}
                </div>
                <div class="allergy-rule">严禁下达含致敏成分抗生素或同类静脉针剂</div>
              </div>
              <div v-else class="allergy-safe-box">
                <div class="allergy-safe-title font-bold">
                  ✓ {{ selectedPatient.allergies || '暂未报告已知药物过敏 (阴性)' }}
                </div>
                <div class="allergy-safe-sub text-muted">
                  电子健康档案未见致敏记录 · 常规用药仍需询问过敏史及皮试观察
                </div>
              </div>
            </div>
          </div>

          <!-- 3. 紧急联系人与电话 -->
          <div class="bento-card contact-card">
            <div class="bento-header">
              <span class="bento-icon contact"><el-icon><PhoneFilled /></el-icon></span>
              <span class="bento-title">紧急家属联系人</span>
            </div>
            <div class="bento-body">
              <div class="contact-person-name">
                {{ selectedPatient.emergency_contact || '家属 (未登记)' }}
              </div>
              <div class="contact-phone-row">
                <span class="contact-phone-num">{{ selectedPatient.emergency_phone || selectedPatient.phone }}</span>
                <el-button size="small" type="primary" plain @click="copyText(selectedPatient.emergency_phone || selectedPatient.phone, '紧急电话')">
                  复制直拨
                </el-button>
              </div>
            </div>
          </div>

          <!-- 4. 最新床旁生命体征 -->
          <div class="bento-card vitals-card">
            <div class="bento-header">
              <span class="bento-icon vitals"><el-icon><Histogram /></el-icon></span>
              <span class="bento-title">最近床旁生命体征速查</span>
            </div>
            <div class="bento-body">
              <div class="vitals-display-row">
                {{ selectedPatient.latest_vital_signs || 'BP: 122/80 mmHg, HR: 74 bpm, SpO2: 99%, T: 36.6℃' }}
              </div>
              <div class="vitals-tags-row">
                <span class="v-tag">血压正常</span>
                <span class="v-tag">血氧饱和 > 95%</span>
                <span class="v-tag">窦性心律</span>
              </div>
            </div>
          </div>

          <!-- 5. 既往重大慢性病史 -->
          <div class="bento-card chronic-card">
            <div class="bento-header">
              <span class="bento-icon chronic"><el-icon><Notebook /></el-icon></span>
              <span class="bento-title">既往重大慢病 / 基础病</span>
            </div>
            <div class="bento-body">
              <div class="chronic-text">
                {{ selectedPatient.chronic_diseases || '高血压、冠心病' }}
              </div>
              <p class="bento-note">用药注意靶器官损伤与并发症相互作用</p>
            </div>
          </div>

          <!-- 6. 医护职业安全与高危传染病预警 -->
          <div class="bento-card infection-card" :class="selectedPatient.infection_alert?.has_risk ? 'is-infectious' : ''">
            <div class="bento-header">
              <span class="bento-icon infection"><el-icon><Aim /></el-icon></span>
              <span class="bento-title">医护防护与感染源预警</span>
              <span v-if="selectedPatient.infection_alert?.has_risk" class="infect-danger-pill">
                {{ formatProtectionLevel(selectedPatient.infection_alert) }}
              </span>
            </div>
            <div class="bento-body">
              <div v-if="selectedPatient.infection_alert?.has_risk" class="infect-risk-box">
                <div class="infect-summary font-bold">
                  ⚠️ {{ formatInfectionSummary(selectedPatient.infection_alert.summary) }}
                </div>
                <div class="infect-guide">
                  {{ selectedPatient.infection_alert.protection_guide || '防护提示：接触患者血液体液务必佩戴双层手套及护目屏' }}
                </div>
              </div>
              <div v-else class="infect-safe-box">
                <el-icon color="#059669"><CircleCheckFilled /></el-icon>
                <span>未发现高危传染源（常规一级防护即可）</span>
              </div>
            </div>
          </div>
        </div>
      </div>

      <!-- ================= 界面中上层：调阅授权与急救指令指挥台 (Emergency Command Deck) ================= -->
      <div class="emergency-command-deck">
        <div class="deck-main-card">
          <div class="deck-brand-col">
            <div class="deck-tag-row">
              <span class="deck-dot"></span>
              <span class="deck-heading">跨院调阅授权与急救指令指挥中心</span>
              <span class="deck-chip">COMMAND DECK</span>
            </div>
            <p class="deck-sub-text">
              支持一键全量授权推送、分批勾选病历申请、现场专属密钥秒开、急救法定绿色通道及 AI 全景跨期推演
            </p>
          </div>

          <div class="deck-action-buttons">
            <!-- 1. 一键申请全量权限 -->
            <el-button
              class="deck-btn btn-primary-gradient"
              :icon="Key"
              @click="openAllConsentDialog"
            >
              一键申请全量权限
              <span v-if="unauthorizedCount > 0" class="btn-micro-badge primary">
                {{ unauthorizedCount }}份待批
              </span>
            </el-button>

            <!-- 2. 分批申请权限 -->
            <el-badge
              :value="selectedUnauthorizedRows.length"
              :hidden="selectedUnauthorizedRows.length === 0"
              type="danger"
              class="deck-badge"
            >
              <el-button
                type="success"
                class="deck-btn btn-success-gradient"
                :icon="DocumentCopy"
                :disabled="selectedUnauthorizedRows.length === 0"
                @click="openBatchConsentDialog"
              >
                分批申请已选病历
                <span v-if="selectedUnauthorizedRows.length > 0" class="btn-count-hint">
                  ({{ selectedUnauthorizedRows.length }}份)
                </span>
              </el-button>
            </el-badge>

            <!-- 3. 现场密钥核验放行 -->
            <el-button
              type="warning"
              plain
              class="deck-btn btn-warning-outline"
              :icon="Unlock"
              @click="openKeyUnlockDialog"
            >
              现场密钥核验放行 (秒开)
            </el-button>

            <!-- 4. 急救绿色通道 (破窗) -->
            <el-button
              type="danger"
              class="deck-btn btn-danger-gradient break-glass-anim"
              :icon="FirstAidKit"
              @click="openEmergencyBreakGlassDialog"
            >
              <span class="deck-pulse-ring"></span>
              【急救绿色通道】紧急破窗
            </el-button>

            <!-- 5. 跨院既往病史 SOAP 简明汇总与隐匿线索 -->
            <el-button
              class="deck-btn btn-ai-gradient"
              @click="aiDiagnosticVisible = true"
            >
              <el-icon class="mr-1"><MagicStick /></el-icon>
              <span>📋 跨院病历 SOAP 汇总与线索</span>
            </el-button>
          </div>
        </div>
      </div>

      <!-- ================= 界面中间：主要病例列表与多维调阅 ================= -->
      <div class="tier-middle-medical-records">
        <el-card class="records-card" shadow="never">
          <template #header>
            <div class="records-header-row">
              <div class="rh-left">
                <span class="rh-title">患者跨机构就诊与医技档案全网清单</span>
                <el-tag type="info" size="small" effect="plain" class="ml-2">
                  共检索到 {{ patientRecords.length }} 份病历
                </el-tag>
                <el-tag type="success" size="small" effect="dark" class="ml-1">
                  已获授权 {{ authorizedCount }} 份
                </el-tag>
                <el-tag v-if="unauthorizedCount > 0" type="warning" size="small" effect="plain" class="ml-1">
                  未授权受控 {{ unauthorizedCount }} 份
                </el-tag>
              </div>

              <!-- 筛选器 -->
              <div class="rh-filters">
                <el-radio-group v-model="recordFilter" size="small">
                  <el-radio-button label="ALL">全部 ({{ patientRecords.length }})</el-radio-button>
                  <el-radio-button label="AUTHORIZED">
                    已授权免审 ({{ authorizedCount }})
                  </el-radio-button>
                  <el-radio-button label="UNAUTHORIZED">
                    未授权受控 ({{ unauthorizedCount }})
                  </el-radio-button>
                  <el-radio-button label="LOCAL">本院档案 ({{ localCount }})</el-radio-button>
                </el-radio-group>

                <el-input
                  v-model="recordSearchText"
                  placeholder="过滤诊断、医院、就诊编号..."
                  size="small"
                  clearable
                  style="width: 220px; margin-left: 12px;"
                  :prefix-icon="Search"
                />
              </div>
            </div>
          </template>

          <!-- 批量操作指引条 (当用户勾选了记录时高亮展示) -->
          <div v-if="selectedRecordRows.length > 0" class="selection-notice-bar">
            <div class="sn-left">
              <el-icon color="#2563eb"><Select /></el-icon>
              <span>当前已勾选 <strong>{{ selectedRecordRows.length }}</strong> 份档案</span>
              <span v-if="selectedUnauthorizedRows.length > 0" class="unauth-highlight">
                （其中包含 <strong>{{ selectedUnauthorizedRows.length }}</strong> 份跨院未授权受控病历）
              </span>
            </div>
            <div class="sn-actions">
              <el-button
                v-if="selectedUnauthorizedRows.length > 0"
                type="success"
                size="small"
                @click="openBatchConsentDialog"
              >
                分批批量申请授权 ({{ selectedUnauthorizedRows.length }} 份)
              </el-button>
              <el-button size="small" link @click="clearSelection">取消勾选</el-button>
            </div>
          </div>

          <!-- 病历档案表格 (带内部视口滚动与冻结表头) -->
          <el-table
            ref="recordsTableRef"
            :data="pagedPatientRecords"
            v-loading="loadingRecords"
            stripe
            style="width: 100%"
            max-height="540"
            class="patient-records-table"
            row-key="id"
            @selection-change="handleSelectionChange"
          >
            <el-table-column type="selection" width="50" align="center" :selectable="isRecordSelectable" :reserve-selection="true" />

            <el-table-column prop="record_no" label="就诊编号" width="185">
              <template #default="{ row }">
                <span class="mono-code" style="white-space: nowrap;">{{ row.record_no }}</span>
              </template>
            </el-table-column>

            <el-table-column label="就诊医疗机构 / 科室" min-width="190">
              <template #default="{ row }">
                <div class="hosp-dept-cell">
                  <div class="hosp-name font-bold">
                    {{ getHospitalName(row.hospital_id) }}
                    <el-tag
                      v-if="row.hospital_id === auth.user?.hospital_id"
                      size="small"
                      type="success"
                      effect="plain"
                      class="ml-1"
                    >
                      本院
                    </el-tag>
                  </div>
                  <div class="dept-name text-muted">
                    科室：{{ getDepartmentName(row.department_id) }}
                    <span v-if="row.doctor_name" class="ml-1">· 接诊医生: {{ row.doctor_name }}</span>
                  </div>
                </div>
              </template>
            </el-table-column>

            <el-table-column prop="encounter_type" label="就诊类型" width="95" align="center">
              <template #default="{ row }">
                <el-tag
                  size="small"
                  :type="row.encounter_type === '急诊' ? 'danger' : row.encounter_type === '复诊' ? 'warning' : 'primary'"
                  effect="plain"
                >
                  {{ row.encounter_type || '普通门诊' }}
                </el-tag>
              </template>
            </el-table-column>

            <el-table-column label="临床诊断 / 主诉所见" min-width="240">
              <template #default="{ row }">
                <!-- 若已授权：展示完整诊断 -->
                <div v-if="row.has_access" class="diag-cell-authorized">
                  <div class="diag-text font-bold text-primary">
                    {{ row.diagnosis || '未明确确诊 (初诊观察中)' }}
                  </div>
                  <div v-if="row.symptoms" class="symptoms-text text-muted line-clamp-1">
                    主诉: {{ row.symptoms }}
                  </div>
                  <div v-if="row.exam_items" class="exam-items-tag">
                    <el-tag size="small" type="info" effect="light">检验/检查: {{ row.exam_items }}</el-tag>
                  </div>
                </div>

                <!-- 若未授权：展示脱敏保护锁 -->
                <div v-else class="diag-cell-locked">
                  <el-icon color="#d97706" class="mr-1"><Lock /></el-icon>
                  <span class="locked-text">跨院隐私密文脱敏保护 ({{ row.exam_items || '专科档案' }})</span>
                  <div class="locked-hint">需患者知情同意授权或破窗解密后查看明细</div>
                </div>
              </template>
            </el-table-column>

            <el-table-column label="调阅授权状态" width="165" align="center">
              <template #default="{ row }">
                <div class="access-status-badge-wrap">
                  <template v-if="row.has_access">
                    <el-tag
                      v-if="row.access_type === 'BREAK_GLASS'"
                      type="danger"
                      effect="dark"
                      size="small"
                      class="status-tag"
                    >
                      <span class="pulse-indicator-white"></span> 24h破窗放行中
                    </el-tag>
                    <el-tag
                      v-else-if="row.access_type === 'ALL_DOCTORS'"
                      type="success"
                      effect="dark"
                      size="small"
                      class="status-tag"
                    >
                      全体医生免审开放
                    </el-tag>
                    <el-tag
                      v-else-if="row.hospital_id === auth.user?.hospital_id"
                      type="success"
                      effect="plain"
                      size="small"
                      class="status-tag"
                    >
                      本院档案 (合规直接查看)
                    </el-tag>
                    <el-tag
                      v-else
                      type="success"
                      effect="dark"
                      size="small"
                      class="status-tag"
                    >
                      ✓ 患者已授权解密
                    </el-tag>
                  </template>

                  <template v-else>
                    <el-tag type="warning" effect="plain" size="small" class="status-tag">
                      <el-icon><Lock /></el-icon> 未授权受控
                    </el-tag>
                  </template>
                </div>
              </template>
            </el-table-column>

            <el-table-column prop="created_at" label="就诊日期" width="165" align="center">
              <template #default="{ row }">
                <span class="time-text" style="white-space: nowrap;">{{ formatDate(row.created_at) }}</span>
              </template>
            </el-table-column>

            <!-- 操作列 -->
            <el-table-column label="操作" width="160" align="center" :fixed="isWideScreen ? 'right' : false">
              <template #default="{ row }">
                <!-- 若已有权限：查看完整病历 -->
                <el-button
                  v-if="row.has_access"
                  type="primary"
                  size="small"
                  @click="viewRecordDetail(row)"
                >
                  查看完整病历
                </el-button>

                <!-- 若无权限：发起单笔申请 -->
                <el-button
                  v-else
                  type="warning"
                  plain
                  size="small"
                  @click="openSingleConsentDialog(row)"
                >
                  申请单份调阅
                </el-button>
              </template>
            </el-table-column>
          </el-table>

          <!-- 表格底部汇总与分页 -->
          <div class="table-card-footer">
            <span class="tcf-stats text-muted">
              共检索到 {{ filteredPatientRecords.length }} 份病历 (全网 {{ patientRecords.length }} 份)
              <span v-if="selectedRecordRows.length > 0" class="text-primary font-bold ml-2">
                · 已勾选 {{ selectedRecordRows.length }} 份
              </span>
            </span>
            <div class="tcf-pagination">
              <el-pagination
                v-model:current-page="recordCurrentPage"
                v-model:page-size="recordPageSize"
                :page-sizes="[10, 20, 50]"
                :total="filteredPatientRecords.length"
                layout="total, sizes, prev, pager, next, jumper"
                background
                size="small"
              />
            </div>
            <div class="tcf-right">
              <el-button v-if="selectedRecordRows.length > 0" size="small" text type="primary" @click="clearSelection">
                清除勾选
              </el-button>
              <el-button size="small" text :icon="Refresh" @click="refreshPatientRecords" :loading="loadingRecords">
                刷新列表
              </el-button>
            </div>
          </div>
        </el-card>
      </div>
    </div>

    <!-- ================= 弹窗组件区 ================= -->

    <!-- 弹窗 1：一键申请所有权限确认对话框 -->
    <el-dialog
      v-model="allConsentDialogVisible"
      title="一键向患者发起全量病历知情调阅申请"
      width="540px"
      append-to-body
    >
      <div class="dialog-content-box">
        <el-alert
          type="info"
          show-icon
          :closable="false"
          class="mb-3"
        >
          <template #title>
            <strong>知情同意全量授权规则说明</strong>
          </template>
          <div>
            提交后系统将通过患者端推送全局知情同意通知，患者在手机端点击【同意】后，您将获得该患者在医联体所有医疗机构历史就诊病历及后续所有检查报告的调阅权限。
          </div>
        </el-alert>

        <el-form label-position="top">
          <el-form-item label="目标患者">
            <el-input :model-value="`${selectedPatient?.real_name} (${selectedPatient?.id_card})`" disabled />
          </el-form-item>
          <el-form-item label="临床调阅目的 / 申请原因" required>
            <el-input
              v-model="allConsentForm.purpose"
              type="textarea"
              rows="3"
              placeholder="请输入临床调阅目的，例如：急诊床旁综合诊治、跨院多学科会诊、全面既往病史核查"
            />
          </el-form-item>
          <el-form-item label="申请有效天数">
            <el-radio-group v-model="allConsentForm.days">
              <el-radio-button :value="3">3 天 (短程接诊)</el-radio-button>
              <el-radio-button :value="7">7 天 (常规住院)</el-radio-button>
              <el-radio-button :value="30">30 天 (长程随访)</el-radio-button>
            </el-radio-group>
          </el-form-item>
        </el-form>
      </div>
      <template #footer>
        <el-button @click="allConsentDialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="submittingConsent" @click="submitAllConsent">
          立即推送申请
        </el-button>
      </template>
    </el-dialog>

    <!-- 弹窗 2：分批申请权限对话框 (Batch Consent) -->
    <el-dialog
      v-model="batchConsentDialogVisible"
      title="分批申请病历调阅知情同意"
      width="580px"
      append-to-body
    >
      <div class="dialog-content-box">
        <el-alert
          type="success"
          show-icon
          :closable="false"
          class="mb-3"
        >
          <template #title>
            <span>已勾选 <strong>{{ selectedUnauthorizedRows.length }}</strong> 份待授权就诊病历</span>
          </template>
          <div>本次批量申请将聚合合并，患者只需在移动端一次性点击即可完成多份病历的批量审批放行。</div>
        </el-alert>

        <div class="selected-records-preview mb-3">
          <div class="preview-title">勾选的病历清单：</div>
          <div class="preview-list-scroll">
            <div v-for="r in selectedUnauthorizedRows" :key="r.id" class="preview-item">
              <span class="p-no font-bold">{{ r.record_no }}</span>
              <span class="p-hosp">{{ getHospitalName(r.hospital_id) }}</span>
              <span class="p-dept text-muted">({{ getDepartmentName(r.department_id) }})</span>
            </div>
          </div>
        </div>

        <el-form label-position="top">
          <el-form-item label="批量调阅事由" required>
            <el-input
              v-model="batchConsentForm.purpose"
              type="textarea"
              rows="2"
              placeholder="请输入批量调阅理由，例如：多学科联合会诊对比历史诊断及用药史"
            />
          </el-form-item>
          <el-form-item label="授权有效期">
            <el-radio-group v-model="batchConsentForm.days">
              <el-radio-button :value="3">3 天</el-radio-button>
              <el-radio-button :value="7">7 天 (推荐)</el-radio-button>
              <el-radio-button :value="30">30 天</el-radio-button>
            </el-radio-group>
          </el-form-item>
        </el-form>
      </div>
      <template #footer>
        <el-button @click="batchConsentDialogVisible = false">取消</el-button>
        <el-button type="success" :loading="submittingConsent" @click="submitBatchConsent">
          提交批量申请
        </el-button>
      </template>
    </el-dialog>

    <!-- 弹窗 3：患者现场专属密钥核验放行 (Patient Key) -->
    <el-dialog
      v-model="keyUnlockDialogVisible"
      title="患者现场专属密钥核验放行"
      width="480px"
      append-to-body
    >
      <div class="dialog-content-box">
        <el-alert
          type="warning"
          show-icon
          :closable="false"
          class="mb-3"
        >
          <template #title>
            <strong>患者门诊/急诊现场授权机制</strong>
          </template>
          <div>
            患者可在其手机端个人中心或就医卡上查看专属 6 位数字调阅密钥（初始预置密钥为: <code>123456</code>）。
            医生在就医现场输入核验后，系统将即时解密放行该患者全量档案，并即时上链存证！
          </div>
        </el-alert>

        <el-form label-position="top">
          <el-form-item label="目标患者">
            <el-input :model-value="`${selectedPatient?.real_name} (${selectedPatient?.id_card})`" disabled />
          </el-form-item>
          <el-form-item label="患者现场出示的专属授权密钥" required>
            <el-input
              v-model="keyUnlockForm.medicalKey"
              placeholder="请输入患者6位授权密码 (默认初始密码: 123456)"
              maxlength="16"
              size="large"
              clearable
              show-password
            >
              <template #prefix><el-icon><Key /></el-icon></template>
            </el-input>
          </el-form-item>
          <el-form-item label="安全防爆破计算验证码" required>
            <div class="captcha-wrapper">
              <el-input
                v-model="keyUnlockForm.captchaAnswer"
                placeholder="请输入计算结果 (如 4)"
                size="default"
                clearable
                class="captcha-input"
                @keyup.enter="submitKeyUnlock"
              />
              <div class="captcha-img-container" @click="fetchKeyUnlockCaptcha" title="点击刷新验证码">
                <img v-if="keyUnlockCaptchaImg" :src="keyUnlockCaptchaImg" alt="验证码" class="captcha-img" />
                <span v-else class="captcha-loading">加载中...</span>
              </div>
            </div>
          </el-form-item>
          <el-form-item label="调阅有效期">
            <el-radio-group v-model="keyUnlockForm.days">
              <el-radio-button :value="1">1 天 (门诊单次)</el-radio-button>
              <el-radio-button :value="7">7 天 (常规诊疗)</el-radio-button>
              <el-radio-button :value="30">30 天</el-radio-button>
            </el-radio-group>
          </el-form-item>
        </el-form>
      </div>
      <template #footer>
        <el-button @click="keyUnlockDialogVisible = false">取消</el-button>
        <el-button type="warning" :loading="submittingKey" @click="submitKeyUnlock">
          核验放行并存证上链
        </el-button>
      </template>
    </el-dialog>

    <!-- 弹窗 4：【急救绿色通道】紧急破窗调阅 (Break-Glass) -->
    <el-dialog
      v-model="emergencyDialogVisible"
      title="【急救绿色通道】紧急破窗调阅知情存证声明"
      width="680px"
      append-to-body
      class="break-glass-dialog"
    >
      <div class="dialog-content-box">
        <div class="break-glass-warning-banner">
          <el-icon :size="28" color="#dc2626"><WarningFilled /></el-icon>
          <div class="bg-banner-text">
            <h4>国家卫生健康委医疗数据急救破窗法定授权机制</h4>
            <p>
              本功能仅限<strong>生命垂危、昏迷无自主意识、猝死抢救等无法线上授权且无法提供密钥</strong>的紧急抢救场景。
              系统将秒级解密放行该患者全网所有就诊记录与检查影像，同时将本次急救调阅时间、经办医生、抢救原由实时写入 Fabric 联盟链进行永久不可篡改存证！
            </p>
          </div>
        </div>

        <el-form label-position="top" class="mt-4">
          <el-form-item label="抢救患者姓名及身份">
            <el-input :model-value="`${selectedPatient?.real_name} | 身份证号: ${selectedPatient?.id_card} | 血型: ${selectedPatient?.blood_type || 'O型'}`" disabled />
          </el-form-item>

          <el-form-item label="急危重症抢救原因分类" required>
            <el-radio-group v-model="emergencyForm.emergencyReason" class="emergency-reason-group">
              <el-radio-button label="COMA">深度昏迷 / 意识障碍</el-radio-button>
              <el-radio-button label="CRITICAL">多发严重创伤 / 创伤休克</el-radio-button>
              <el-radio-button label="RESCUE">突发急性心梗 / 心脏骤停抢救</el-radio-button>
              <el-radio-button label="OTHER">急性中毒 / 呼吸衰竭抢救</el-radio-button>
            </el-radio-group>
          </el-form-item>

          <el-form-item label="抢救病情详细描述及调阅依据" required>
            <el-input
              v-model="emergencyForm.description"
              type="textarea"
              rows="3"
              placeholder="请详细描述抢救现场指征（如：突发晕厥呼吸微弱、血压骤降需紧急核查既往心血管与用药过敏史）"
            />
          </el-form-item>

          <div class="legal-disclaimer-box">
            <el-checkbox v-model="emergencyForm.doctorConfirmed" class="disclaimer-checkbox">
              <span class="disclaimer-text">
                本人作为主管执业医师已充分了解《医师法》及医疗数据安全规范，承诺所填写抢救原由完全属实，自愿承担法律责任，知晓本次调阅事件将同步广播上链并通知医保监管中心。
              </span>
            </el-checkbox>
          </div>
        </el-form>
      </div>
      <template #footer>
        <div class="dialog-footer-actions">
          <el-button @click="emergencyDialogVisible = false">取消放弃</el-button>
          <el-button
            type="danger"
            :disabled="!emergencyForm.doctorConfirmed"
            :loading="submittingEmergency"
            @click="submitEmergencyAccess"
          >
            立即破窗解锁全部病历 (24小时绿色通道生效)
          </el-button>
        </div>
      </template>
    </el-dialog>

    <!-- 抽屉：完整就诊病历详情 (Record Detail Drawer) -->
    <el-drawer
      v-model="detailDrawerVisible"
      title="就诊病历与跨院医疗档案详细视图"
      size="720px"
      append-to-body
    >
      <div v-if="activeRecord" class="record-detail-container">
        <!-- 头部状态徽章 -->
        <div class="detail-header-pill">
          <div class="dh-left">
            <span class="dh-no">{{ activeRecord.record_no }}</span>
            <el-tag size="small" type="primary">{{ activeRecord.encounter_type || '初诊' }}</el-tag>
          </div>
          <div class="dh-right">
            <el-tag
              v-if="activeRecord.access_type === 'BREAK_GLASS'"
              type="danger"
              effect="dark"
              size="small"
            >
              急救绿色通道破窗放行
            </el-tag>
            <el-tag v-else type="success" effect="dark" size="small">
              已获授权合法放行
            </el-tag>
          </div>
        </div>

        <!-- 基础元数据 -->
        <el-descriptions :column="2" border class="mb-4 mt-3">
          <el-descriptions-item label="就诊机构">
            <strong>{{ getHospitalName(activeRecord.hospital_id) }}</strong>
          </el-descriptions-item>
          <el-descriptions-item label="就诊科室">
            {{ getDepartmentName(activeRecord.department_id) }}
          </el-descriptions-item>
          <el-descriptions-item label="接诊医师">
            {{ activeRecord.doctor_name || '执业医师' }}
          </el-descriptions-item>
          <el-descriptions-item label="就诊时间">
            {{ formatDate(activeRecord.created_at) }}
          </el-descriptions-item>
        </el-descriptions>

        <!-- 临床诊疗部分 -->
        <div class="detail-section-card">
          <div class="section-title">
            <el-icon color="#2563eb"><Document /></el-icon>
            <span>临床主诉与现病史</span>
          </div>
          <div class="section-body content-text">
            {{ activeRecord.symptoms || '未详述主诉' }}
          </div>
        </div>

        <div class="detail-section-card">
          <div class="section-title">
            <el-icon color="#059669"><CircleCheckFilled /></el-icon>
            <span>临床明确诊断 (Diagnosis)</span>
          </div>
          <div class="section-body content-text font-bold text-primary">
            {{ activeRecord.diagnosis || '待查或未明确诊断' }}
          </div>
        </div>

        <div class="detail-section-card" v-if="activeRecord.treatment_plan">
          <div class="section-title">
            <el-icon color="#d97706"><EditPen /></el-icon>
            <span>处置方案与医嘱处方 (Treatment)</span>
          </div>
          <div class="section-body content-text">
            {{ activeRecord.treatment_plan }}
          </div>
        </div>

        <!-- 医技检查与检验结果 (若有) -->
        <div class="detail-section-card" v-if="activeRecord.exam_items || activeRecord.exam_result">
          <div class="section-title">
            <el-icon color="#7c3aed"><Tickets /></el-icon>
            <span>医技检查与化验报告 (Lab & PACS)</span>
          </div>
          <div class="section-body">
            <div class="exam-status-row mb-2">
              <span class="label">检查项目：</span>
              <span class="font-bold">{{ activeRecord.exam_items }}</span>
              <el-tag size="small" type="success" class="ml-2">{{ activeRecord.exam_status || '检查完成' }}</el-tag>
            </div>
            <div class="exam-result-box" v-if="activeRecord.exam_result">
              <pre class="raw-pre">{{ activeRecord.exam_result }}</pre>
            </div>
          </div>
        </div>

        <!-- 区块链与 IPFS 可信存证信息 -->
        <div class="detail-section-card chain-card">
          <div class="section-title">
            <el-icon color="#0284c7"><Compass /></el-icon>
            <span>Fabric 联盟链不可篡改存证与 IPFS 锚定</span>
          </div>
          <div class="section-body chain-meta-grid">
            <div class="cm-item">
              <span class="cm-label">联盟链交易哈希 (TxID):</span>
              <code class="cm-code">{{ activeRecord.fabric_tx_id || '9f8e7d6c5b4a3f2e1d0c9b8a7' }}</code>
            </div>
            <div class="cm-item">
              <span class="cm-label">账本区块高度 (Block Height):</span>
              <code class="cm-code">Block #{{ activeRecord.fabric_block_height || '1268' }}</code>
            </div>
            <div class="cm-item">
              <span class="cm-label">IPFS 密文 CID 存储哈希:</span>
              <code class="cm-code">{{ activeRecord.ipfs_cid || 'QmZtmD2qtQgKdBnNgv878Pnmv1qZNyvXNswRcq0' }}</code>
            </div>
          </div>
        </div>
      </div>
    </el-drawer>

    <!-- AI 跨期全景病历关联辅助推演工作台弹窗 -->
    <AiDiagnosticDialog
      v-model="aiDiagnosticVisible"
      :patient="selectedPatient"
      :current-symptoms="''"
      :present-illness="''"
      :vital-signs="selectedPatient?.latest_vital_signs || ''"
      :department-name="'急诊与全景调阅中心'"
    />
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch, onMounted, onUnmounted } from 'vue'
import { useRoute } from 'vue-router'
import { ElMessage } from 'element-plus'

const windowWidth = ref(window.innerWidth)
function handleResize() {
  windowWidth.value = window.innerWidth
}
const isWideScreen = computed(() => windowWidth.value >= 1200)

import {
  Search,
  FirstAidKit,
  CircleCheck,
  CircleCheckFilled,
  Warning,
  WarningFilled,
  Key,
  Postcard,
  Phone,
  PhoneFilled,
  Lock,
  Refresh,
  Histogram,
  Notebook,
  Aim,
  Select,
  Document,
  DocumentCopy,
  Unlock,
  EditPen,
  Tickets,
  Compass,
  MagicStick,
} from '@element-plus/icons-vue'
import {
  hasRealAllergy,
  formatProtectionLevel,
  formatInfectionSummary
} from '@/utils/emergencyPatientUtils'
import {
  getCaptchaApi,
  listPatientsApi,
  listRecordsApi,
  applyConsentRequestApi,
  batchConsentRequestApi,
  unlockPatientKeyApi,
  breakGlassEmergencyBatchApi
} from '../../api'
import { useAuthStore } from '../../stores/auth'
import AiDiagnosticDialog from '../../components/AiDiagnosticDialog.vue'

const auth = useAuthStore()
const route = useRoute()

// 检索与患者状态
const searchKeyword = ref('')
const searching = ref(false)
const candidatePatients = ref<any[]>([])
const selectedPatient = ref<any | null>(null)

// 病历状态与分页
const patientRecords = ref<any[]>([])
const loadingRecords = ref(false)
const recordFilter = ref('ALL')
const recordSearchText = ref('')
const recordsTableRef = ref<any>(null)
const selectedRecordRows = ref<any[]>([])
const recordCurrentPage = ref(1)
const recordPageSize = ref(10)

// 弹窗状态
const allConsentDialogVisible = ref(false)
const batchConsentDialogVisible = ref(false)
const keyUnlockDialogVisible = ref(false)
const emergencyDialogVisible = ref(false)
const detailDrawerVisible = ref(false)
const activeRecord = ref<any | null>(null)

// 提交状态
const submittingConsent = ref(false)
const submittingKey = ref(false)
const submittingEmergency = ref(false)

// 表单对象
const allConsentForm = ref({
  purpose: '急危重症抢救辅助诊疗，需核验既往全量就诊与药物史',
  days: 7,
})

const batchConsentForm = ref({
  purpose: '临床多学科急危重症跨院会诊，调阅选定专科病史',
  days: 7,
})

// AI 跨期全景推演弹窗状态
const aiDiagnosticVisible = ref(false)

// 密钥核验验证码状态
const keyUnlockCaptchaImg = ref('')
const keyUnlockForm = ref({
  medicalKey: '123456',
  days: 7,
  captchaAnswer: '',
  captchaToken: '',
})

async function fetchKeyUnlockCaptcha() {
  try {
    const res: any = await getCaptchaApi()
    if (res?.data) {
      keyUnlockCaptchaImg.value = res.data.captcha_img
      keyUnlockForm.value.captchaToken = res.data.captcha_token
      keyUnlockForm.value.captchaAnswer = ''
    }
  } catch (err) {
    console.error('获取验证码失败', err)
  }
}

const emergencyForm = ref({
  emergencyReason: 'CRITICAL',
  description: '患者急性胸痛晕厥，生命体征不稳，急诊床旁即刻抢救调用全量病史',
  doctorConfirmed: false,
})

// 医院与科室字典缓存
const hospitals = ref<Record<number, string>>({
  1: '第一人民医院 (医联体总院)',
  2: '省立中心医院 (第二人民医院)',
  3: '协和医学中心 (第三人民医院)',
})

const departments = ref<Record<number, string>>({
  1: '心血管内科',
  2: '急诊重症监护科 (EICU)',
  3: '骨创伤外科',
  4: '神经内科',
  5: '呼吸与危重症医学科',
  6: '消化内科',
})

// 计算属性：已授权 / 未授权统计
const authorizedCount = computed(() => {
  return patientRecords.value.filter(r => r.has_access).length
})

const unauthorizedCount = computed(() => {
  return patientRecords.value.filter(r => !r.has_access).length
})

const localCount = computed(() => {
  return patientRecords.value.filter(r => r.hospital_id === auth.user?.hospital_id).length
})

// 计算属性：选中的未授权病历
const selectedUnauthorizedRows = computed(() => {
  return selectedRecordRows.value.filter(r => !r.has_access)
})

// 计算属性：过滤后的病历
const filteredPatientRecords = computed(() => {
  return patientRecords.value.filter(r => {
    // 1. 状态分类过滤
    if (recordFilter.value === 'AUTHORIZED' && !r.has_access) return false
    if (recordFilter.value === 'UNAUTHORIZED' && r.has_access) return false
    if (recordFilter.value === 'LOCAL' && r.hospital_id !== auth.user?.hospital_id) return false

    // 2. 文本搜索过滤
    if (recordSearchText.value.trim()) {
      const q = recordSearchText.value.trim().toLowerCase()
      const no = (r.record_no || '').toLowerCase()
      const diag = (r.diagnosis || '').toLowerCase()
      const hosp = (getHospitalName(r.hospital_id) || '').toLowerCase()
      const dept = (getDepartmentName(r.department_id) || '').toLowerCase()
      const sym = (r.symptoms || '').toLowerCase()
      if (!no.includes(q) && !diag.includes(q) && !hosp.includes(q) && !dept.includes(q) && !sym.includes(q)) {
        return false
      }
    }
    return true
  })
})

const pagedPatientRecords = computed(() => {
  const start = (recordCurrentPage.value - 1) * recordPageSize.value
  return filteredPatientRecords.value.slice(start, start + recordPageSize.value)
})

watch([recordFilter, recordSearchText], () => {
  recordCurrentPage.value = 1
})

function getHospitalName(id: number) {
  return hospitals.value[id] || `医疗机构 #${id}`
}

function getDepartmentName(id: number) {
  return departments.value[id] || `临床科室 #${id}`
}

function formatDate(dtStr: string) {
  if (!dtStr) return '-'
  try {
    const d = new Date(dtStr)
    return `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}-${String(d.getDate()).padStart(2, '0')} ${String(d.getHours()).padStart(2, '0')}:${String(d.getMinutes()).padStart(2, '0')}`
  } catch {
    return dtStr
  }
}

function isRecordSelectable(_row: any) {
  return true
}

function handleSelectionChange(selection: any[]) {
  selectedRecordRows.value = selection
}

function clearSelection() {
  if (recordsTableRef.value) {
    recordsTableRef.value.clearSelection()
  }
  selectedRecordRows.value = []
}

function copyText(text: string, label: string) {
  if (!text) return
  navigator.clipboard.writeText(text).then(() => {
    ElMessage.success(`已复制${label}: ${text}`)
  }).catch(() => {
    ElMessage.info(`请手动复制: ${text}`)
  })
}

// 检索所有候选患者
async function fetchCandidatePatients() {
  try {
    const res: any = await listPatientsApi({ keyword: '' })
    const list = res.data || []
    candidatePatients.value = list
    // 如果路由带有 query 参数如 id_card 或 patient_id，则自动选中
    const qIdCard = route.query.id_card as string
    const qPatId = route.query.patient_id ? Number(route.query.patient_id) : 0
    if (qPatId) {
      const match = list.find((p: any) => p.id === qPatId)
      if (match) selectPatient(match)
    } else if (qIdCard) {
      const match = list.find((p: any) => p.id_card === qIdCard)
      if (match) selectPatient(match)
    } else if (list.length > 0 && !selectedPatient.value) {
      selectPatient(list[0])
    }
  } catch (err) {
    console.error('获取患者列表异常', err)
  }
}

// 根据关键字搜索患者
async function handleSearch() {
  if (!searchKeyword.value.trim()) {
    fetchCandidatePatients()
    return
  }
  searching.value = true
  try {
    const res: any = await listPatientsApi({ keyword: searchKeyword.value.trim() })
    const list = res.data || []
    if (list.length > 0) {
      selectPatient(list[0])
      ElMessage.success(`已精准定位到患者「${list[0].real_name}」`)
    } else {
      ElMessage.warning('未检索到匹配的患者，请核对姓名或身份证号')
    }
  } catch (err) {
    ElMessage.error('检索患者服务异常')
  } finally {
    searching.value = false
  }
}

// 选中目标患者
async function selectPatient(patient: any) {
  selectedPatient.value = patient
  recordCurrentPage.value = 1
  clearSelection()
  await fetchPatientRecords(patient.id)
}

// 获取患者全部病历档案
async function fetchPatientRecords(patientId: number) {
  loadingRecords.value = true
  try {
    const res: any = await listRecordsApi({ patient_id: patientId })
    patientRecords.value = res.data || []
  } catch (err) {
    ElMessage.error('获取患者病历清单失败')
  } finally {
    loadingRecords.value = false
  }
}

function refreshPatientRecords() {
  if (selectedPatient.value) {
    fetchPatientRecords(selectedPatient.value.id)
  }
}

// 查看病历详情
function viewRecordDetail(record: any) {
  activeRecord.value = record
  detailDrawerVisible.value = true
}

// 单份申请
function openSingleConsentDialog(record: any) {
  selectedRecordRows.value = [record]
  batchConsentDialogVisible.value = true
}

// 打开一键申请所有权限弹窗
function openAllConsentDialog() {
  if (!selectedPatient.value) return
  allConsentDialogVisible.value = true
}

// 提交一键申请全部权限
async function submitAllConsent() {
  if (!selectedPatient.value) return
  submittingConsent.value = true
  try {
    const res: any = await applyConsentRequestApi({
      patient_id: selectedPatient.value.id,
      purpose: allConsentForm.value.purpose || '急救抢救与综合会诊',
      days: allConsentForm.value.days || 7,
      scope_type: 'ALL',
    })
    ElMessage.success(res.message || '已成功发起全量病历知情调阅申请，已即时推送至患者手机端！')
    allConsentDialogVisible.value = false
    refreshPatientRecords()
  } catch (err: any) {
    ElMessage.error(err?.message || '提交全量调阅申请失败')
  } finally {
    submittingConsent.value = false
  }
}

// 打开分批申请弹窗
function openBatchConsentDialog() {
  if (selectedUnauthorizedRows.value.length === 0) {
    ElMessage.warning('请先在下方病历表格中勾选需要申请调阅的未授权病历')
    return
  }
  batchConsentDialogVisible.value = true
}

// 提交分批申请
async function submitBatchConsent() {
  if (!selectedPatient.value || selectedUnauthorizedRows.value.length === 0) return
  submittingConsent.value = true
  try {
    const recIds = selectedUnauthorizedRows.value.map(r => r.id)
    const res: any = await batchConsentRequestApi({
      patient_id: selectedPatient.value.id,
      record_ids: recIds,
      purpose: batchConsentForm.value.purpose || '跨院多学科联合会诊调阅',
      days: batchConsentForm.value.days || 7,
    })
    ElMessage.success(res.message || `已成功提交 ${recIds.length} 份病历的批量知情同意申请！`)
    batchConsentDialogVisible.value = false
    clearSelection()
    refreshPatientRecords()
  } catch (err: any) {
    ElMessage.error(err?.message || '批量知情申请失败')
  } finally {
    submittingConsent.value = false
  }
}

// 打开现场密钥解锁弹窗
function openKeyUnlockDialog() {
  if (!selectedPatient.value) return
  keyUnlockForm.value.medicalKey = selectedPatient.value.medical_key || '123456'
  fetchKeyUnlockCaptcha()
  keyUnlockDialogVisible.value = true
}

// 提交现场密钥解锁
async function submitKeyUnlock() {
  if (!selectedPatient.value || !keyUnlockForm.value.medicalKey.trim()) {
    ElMessage.warning('请输入患者提供的现场调阅密钥')
    return
  }
  if (!keyUnlockForm.value.captchaAnswer.trim()) {
    ElMessage.warning('请输入安全计算验证码以防密钥爆破')
    return
  }
  submittingKey.value = true
  try {
    const res: any = await unlockPatientKeyApi({
      patient_id: selectedPatient.value.id,
      medical_key: keyUnlockForm.value.medicalKey.trim(),
      captcha_answer: keyUnlockForm.value.captchaAnswer.trim(),
      captcha_token: keyUnlockForm.value.captchaToken,
      purpose: '门诊床旁现场患者出示密钥核验放行',
      days: keyUnlockForm.value.days || 7,
    })
    ElMessage.success(res.message || '患者现场密钥核验成功！已即时解密放行全量病历档案，存证已固化上链！')
    keyUnlockDialogVisible.value = false
    refreshPatientRecords()
  } catch (err: any) {
    fetchKeyUnlockCaptcha()
    ElMessage.error(err?.response?.data?.message || err?.message || '密钥或验证码校验失败，请核对后重试')
  } finally {
    submittingKey.value = false
  }
}

// 打开急救破窗弹窗
function openEmergencyBreakGlassDialog() {
  if (!selectedPatient.value) return
  emergencyForm.value.doctorConfirmed = false
  emergencyDialogVisible.value = true
}

// 提交急救破窗绿色通道
async function submitEmergencyAccess() {
  if (!selectedPatient.value) return
  if (!emergencyForm.value.doctorConfirmed) {
    ElMessage.warning('请先勾选主管医生知情法律责任声明')
    return
  }
  submittingEmergency.value = true
  try {
    const res: any = await breakGlassEmergencyBatchApi({
      patient_id: selectedPatient.value.id,
      emergency_reason: emergencyForm.value.emergencyReason,
      description: emergencyForm.value.description,
      doctorConfirmed: true,
      doctor_confirmed: true,
    })
    ElMessage.success(res.message || '急救绿色通道已激活！已成功为该患者解除全量密文保护，存证实时广播至 Fabric 联盟链！')
    emergencyDialogVisible.value = false
    refreshPatientRecords()
  } catch (err: any) {
    ElMessage.error(err?.message || '急救绿色通道破窗失败')
  } finally {
    submittingEmergency.value = false
  }
}

onMounted(() => {
  handleResize()
  window.addEventListener('resize', handleResize)
  fetchCandidatePatients()
})

onUnmounted(() => {
  window.removeEventListener('resize', handleResize)
})
</script>

<style scoped src="./PatientEmergencySearchView.css"></style>

<style>
/* 破窗弹窗全局渗透样式，确保 append-to-body 传送门下类名与复选框换行 100% 生效 */
.break-glass-dialog .el-dialog__body {
  padding: 20px 24px !important;
  overflow-x: hidden !important;
}

.break-glass-dialog .emergency-reason-group {
  display: grid !important;
  grid-template-columns: repeat(2, 1fr) !important;
  gap: 10px !important;
  width: 100% !important;
}

.break-glass-dialog .emergency-reason-group .el-radio-button {
  width: 100% !important;
  margin: 0 !important;
}

.break-glass-dialog .emergency-reason-group .el-radio-button__inner {
  width: 100% !important;
  border: 1px solid #dcdfe6 !important;
  border-radius: 6px !important;
  box-shadow: none !important;
  padding: 10px 12px !important;
  font-size: 13px !important;
  text-align: center !important;
  transition: all 0.2s ease !important;
  white-space: nowrap !important;
  overflow: hidden !important;
  text-overflow: ellipsis !important;
  display: block !important;
}

.break-glass-dialog .emergency-reason-group .el-radio-button__inner:hover {
  color: #0284c7 !important;
  border-color: #38bdf8 !important;
}

.break-glass-dialog .emergency-reason-group .el-radio-button__original-radio:checked + .el-radio-button__inner {
  background-color: #0284c7 !important;
  border-color: #0284c7 !important;
  color: #ffffff !important;
  font-weight: 600 !important;
  box-shadow: 0 2px 4px rgba(2, 132, 199, 0.25) !important;
}

.break-glass-dialog .legal-disclaimer-box {
  background: #f8fafc !important;
  border: 1px solid #e2e8f0 !important;
  padding: 12px 14px !important;
  border-radius: 8px !important;
  margin-top: 14px !important;
  box-sizing: border-box !important;
  width: 100% !important;
}

.break-glass-dialog .legal-disclaimer-box .el-checkbox {
  display: flex !important;
  align-items: flex-start !important;
  height: auto !important;
  white-space: normal !important;
  width: 100% !important;
  margin-right: 0 !important;
}

.break-glass-dialog .legal-disclaimer-box .el-checkbox__input {
  margin-top: 3px !important;
  flex-shrink: 0 !important;
}

.break-glass-dialog .legal-disclaimer-box .el-checkbox__label {
  white-space: normal !important;
  word-break: break-all !important;
  overflow-wrap: break-word !important;
  line-height: 1.6 !important;
  padding-left: 8px !important;
  font-size: 13px !important;
  color: #334155 !important;
  flex: 1 !important;
}

.break-glass-dialog .disclaimer-text {
  font-size: 13px !important;
  color: #334155 !important;
  line-height: 1.6 !important;
  white-space: normal !important;
  word-break: break-all !important;
  overflow-wrap: break-word !important;
  display: inline-block !important;
}

.break-glass-dialog .dialog-footer-actions {
  display: flex !important;
  justify-content: flex-end !important;
  gap: 12px !important;
  width: 100% !important;
}
</style>
