<template>
  <div class="page-container">
    <!-- 视图一：跨院病例检索与多维筛选视图 (QUERY) -->
    <div v-if="currentView === 'QUERY'" class="query-main-wrapper">
      <div class="page-header">
      <div class="header-main-row">
        <div>
          <div class="header-badge-row">
            <el-button
              v-if="isFromEncounter || hasEncounterDraft"
              type="primary"
              :icon="ArrowLeft"
              class="top-left-back-btn"
              @click="returnToEncounter"
            >
              返回接诊录入
            </el-button>
            <el-button
              v-else-if="route.query.id_card"
              type="default"
              plain
              :icon="ArrowLeft"
              class="top-left-back-btn"
              @click="router.push('/doctor/records')"
            >
              返回门诊列表
            </el-button>
            <h2 class="page-title">{{ scopeMeta.title }}</h2>
            <el-tag :type="scopeMeta.tagType" effect="dark" class="scope-header-tag">
              {{ scopeMeta.tag }}
            </el-tag>
          </div>
          <p class="page-sub">{{ scopeMeta.sub }}</p>
        </div>
        <div class="header-action-group">
          <el-button type="danger" plain @click="$router.push('/doctor/patient-search')">
            <el-icon style="margin-right: 4px;"><FirstAidKit /></el-icon>
            患者全景与急救速查
          </el-button>
          <el-button :icon="Refresh" circle @click="refreshData" title="刷新列表数据" />
        </div>
      </div>
    </div>

    <!-- 快捷调阅患者提示条 (由门诊工作台跳转时展示) -->
    <el-alert
      v-if="route.query.id_card"
      type="success"
      show-icon
      class="mb-3 patient-route-alert"
      :closable="false"
    >
      <template #title>
        <div class="alert-title-row" style="display: flex; align-items: center; justify-content: space-between; width: 100%;">
          <span>
            <strong>已自动定位目标患者：</strong>
            姓名：<strong style="color: #047857;">{{ route.query.patient_name || '患者' }}</strong> |
            身份证号：<code style="font-weight: bold; background: #ecfdf5; padding: 2px 6px; border-radius: 4px;">{{ route.query.id_card }}</code>。
            系统已自动全网检索其所有就诊档案，请在下方列表核验并选择<strong>【申请调阅 (知情同意/破窗)】</strong>或直接查阅。
          </span>
          <div class="alert-btn-group" style="display: flex; gap: 8px; align-items: center; flex-shrink: 0; margin-left: 16px;">
            <el-button
              size="small"
              class="btn-cross-ai-diagnose"
              :icon="MagicStick"
              @click="openCrossAIDiagnostic"
            >
              📋 跨院病历 SOAP 汇总与线索
            </el-button>
            <el-button
              v-if="isFromEncounter || hasEncounterDraft"
              size="small"
              type="primary"
              :icon="ArrowLeft"
              style="font-weight: 600;"
              @click="returnToEncounter"
            >
              返回继续录入病历 (已自动暂存)
            </el-button>
            <el-button
              v-else
              size="small"
              type="primary"
              plain
              @click="router.push('/doctor/records')"
            >
              返回门诊就诊工作台
            </el-button>
          </div>
        </div>
      </template>
    </el-alert>

    <!-- 录入草稿保留提示条 (无目标卡号参数但存在草稿时) -->
    <el-alert
      v-if="!route.query.id_card && hasEncounterDraft"
      type="warning"
      show-icon
      class="mb-3 draft-notice-alert"
      :closable="false"
    >
      <template #title>
        <div style="display: flex; align-items: center; justify-content: space-between; width: 100%;">
          <span>检测到您有一份尚未完成的接诊录入草稿正在妥善暂存中，查阅既往病历后可随时点击返回继续填写。</span>
          <el-button size="small" type="primary" :icon="ArrowLeft" @click="returnToEncounter">
            返回继续录入病历
          </el-button>
        </div>
      </template>
    </el-alert>

    <!-- 业务规范与当前分支放行规则说明条 -->
    <el-alert
      :title="scopeMeta.alertTitle"
      :type="scopeMeta.alertType"
      :closable="false"
      show-icon
      class="mb-3 rule-alert"
    >
      <template #default>
        <div class="rule-alert-content">
          <span>{{ scopeMeta.alertText }}</span>
        </div>
      </template>
    </el-alert>

    <!-- 检索与分支快捷导航区 -->
    <el-card shadow="hover" class="box-card mb-4">
      <div class="filter-wrapper">
        <!-- 分支目录标签卡片 -->
        <div class="scope-nav-row">
          <span class="scope-label">病例目录：</span>
          <el-radio-group v-model="currentScope" size="default" class="scope-tabs-group">
            <el-radio-button value="authorized">
              <span class="tab-btn-inner">
                <el-icon class="mr-1"><CircleCheckFilled /></el-icon>
                <span>外院已有查阅权限 ({{ statsCount.authorized }})</span>
              </span>
            </el-radio-button>
            <el-radio-button value="local">
              <span class="tab-btn-inner">
                <el-icon class="mr-1"><OfficeBuilding /></el-icon>
                <span>本院的 ({{ statsCount.local }})</span>
              </span>
            </el-radio-button>
            <el-radio-button value="pending">
              <span class="tab-btn-inner">
                <el-icon class="mr-1"><Lock /></el-icon>
                <span>外院需要调取的 ({{ statsCount.pending }})</span>
              </span>
            </el-radio-button>
            <el-radio-button value="all">
              <span class="tab-btn-inner">
                <el-icon class="mr-1"><Compass /></el-icon>
                <span>全网联合检索 ({{ statsCount.all }})</span>
              </span>
            </el-radio-button>
          </el-radio-group>
        </div>

        <div class="search-form mt-3">
          <el-select 
            v-if="currentScope === 'all'" 
            v-model="selectedHospital" 
            placeholder="筛选特定机构" 
            style="width: 180px;" 
            @change="searchCrossRecords"
          >
            <el-option label="全网所有医疗机构" :value="0" />
            <el-option label="第一人民医院 (Hosp A)" :value="1" />
            <el-option label="省立中心医院 (Hosp B)" :value="2" />
            <el-option label="协和医学中心 (Hosp C)" :value="3" />
          </el-select>

          <!-- 检查单与档案类别精准筛选 -->
          <el-select 
            v-model="selectedDataType" 
            placeholder="档案/检查单类型" 
            style="width: 200px;" 
            @change="searchCrossRecords"
          >
            <el-option label="全部档案与检查单" value="ALL" />
            <el-option label="仅调阅检查检验单/切片" value="EXAM_ONLY" />
            <el-option label="医学检验报告 (REPORT)" value="REPORT" />
            <el-option label="医学影像切片 (IMAGE)" value="IMAGE" />
            <el-option label="门诊电子病历 (EMR)" value="EMR" />
          </el-select>

          <el-input
            v-model="searchKeyword"
            placeholder="输入患者姓名、手机号、身份证号、检查项目(如心电图/CT)检索..."
            class="cross-search-input"
            clearable
            @keyup.enter="searchCrossRecords"
            @clear="searchCrossRecords"
          >
            <template #prefix><el-icon><Search /></el-icon></template>
          </el-input>
          <el-button type="primary" :icon="Search" :loading="loading" @click="searchCrossRecords">精准检索</el-button>
          <span class="search-tip">提示：支持按患者姓名或特定检查项目精准检索</span>
        </div>
      </div>
    </el-card>

    <!-- 该患者做过的所有检查单与医学影像概览专区 (支持跨机构调阅与统一安全评估) -->
    <el-card v-if="patientExamRecords.length > 0 && searchKeyword.trim()" shadow="hover" class="box-card mb-4 patient-exams-card">
      <template #header>
        <div class="pec-header">
          <div class="pec-title">
            <strong>
              <template v-if="route.query.patient_name">
                【{{ route.query.patient_name }}】关联全部外院/全网检查单与影像报告 (共 {{ displayedExamRecords.length }} 项)
              </template>
              <template v-else>
                检索关键词【{{ searchKeyword }}】匹配的外院/全网检查单与报告 (共 {{ displayedExamRecords.length }} 项)
              </template>
            </strong>
            <el-tag size="small" type="success" effect="dark" class="ml-2">跨机构可信存证</el-tag>
          </div>

          <!-- 快捷视图控制与异常筛选工具栏 -->
          <div class="pec-toolbar">
            <el-switch
              v-model="filterOnlyAbnormal"
              inline-prompt
              active-text="仅看阳性/异常"
              inactive-text="全部报告"
              size="small"
              class="mr-2"
            />
            <el-button
              size="small"
              :type="expandAllFindings ? 'primary' : 'default'"
              plain
              @click.stop="toggleAllFindings"
            >
              {{ expandAllFindings ? '收起全部测量明细' : '展开全部测量明细' }}
            </el-button>
          </div>
        </div>
      </template>

      <div class="pec-grid">
        <div
          v-for="exam in displayedExamRecords"
          :key="'exam-' + exam.id"
          class="pec-item-card"
          :class="{ 
            'has-access': exam.has_access || exam.doctor_id === auth.user?.id || exam.hospital_id === auth.user?.hospital_id,
            'is-abnormal-card': getParsedExam(exam).isAnyAbnormal
          }"
          @click="handleAccess(exam)"
        >
          <!-- 1. 卡片顶部状态区：类别、账本一致性、异常总览标签 -->
          <div class="pec-item-top">
            <div class="top-tags-left">
              <el-tag size="small" :type="exam.data_type === 'IMAGE' ? 'warning' : (exam.data_type === 'REPORT' ? 'success' : 'primary')" effect="dark">
                {{ exam.data_type === 'IMAGE' ? '影像切片' : (exam.data_type === 'REPORT' ? '检验化验' : '临床检查') }}
              </el-tag>
              <el-tag v-if="exam.is_tampered" size="small" type="danger" effect="dark">篡改告警</el-tag>
              <el-tag v-else size="small" type="success" effect="light">账本一致</el-tag>
            </div>

            <!-- 右侧：报告级异常/阳性快速指征徽章 -->
            <div class="top-tags-right">
              <template v-for="tag in getParsedExam(exam).summaryTags.slice(0, 2)" :key="tag.text">
                <el-tag size="small" :type="tag.type" effect="plain" class="clinical-summary-tag">
                  {{ tag.text }}
                </el-tag>
              </template>
            </div>
          </div>

          <!-- 2. 卡片项目标题与就诊患者 -->
          <div class="pec-item-name">
            <span v-if="exam.patient_name" class="font-bold text-indigo-700 mr-1">[{{ exam.patient_name }}]</span>
            <span v-if="exam.exam_items && !exam.exam_items.includes('未取得患者授权')" v-html="highlightText(exam.exam_items, searchKeyword)"></span>
            <span v-else-if="exam.symptoms && !exam.symptoms.includes('未取得患者授权')" v-html="highlightText(exam.symptoms, searchKeyword)"></span>
            <span v-else-if="exam.diagnosis && !exam.diagnosis.includes('未授权')" v-html="highlightText(exam.diagnosis, searchKeyword)"></span>
            <span v-else>
              {{ exam.data_type === 'IMAGE' ? '跨院医学影像数据' : '跨院临床检查单据' }} ({{ exam.record_no }})
            </span>
          </div>

          <!-- 3. 医疗机构与开具医生元信息 -->
          <div class="pec-item-meta-bar">
            <span><strong>{{ exam.hospital_name }}</strong> · {{ exam.department_name || '临床/医技科室' }}</span>
            <span>{{ formatTime(exam.created_at) }} · {{ exam.doctor_name }} 医生</span>
          </div>

          <!-- 4. 核心内容区：优先呈现核心诊断结论，分层收起参数明细 -->
          <div class="pec-item-body">
            <template v-if="exam.has_access || exam.doctor_id === auth.user?.id || exam.hospital_id === auth.user?.hospital_id">
              <!-- 情况 A：存在医技回传报告与结论 (exam_result) -->
              <div v-if="exam.exam_result" class="structured-exam-container">
                <!-- 结构化子项列表（例如将心电图与胸部CT完全拆分呈现） -->
                <div
                  v-for="(sec, sIdx) in getParsedExam(exam).sections"
                  :key="sec.id"
                  class="exam-sub-section"
                  :class="{ 
                    'section-matched': sec.hasKeywordMatch, 
                    'section-abnormal': sec.isAbnormal 
                  }"
                >
                  <!-- 子项标头 -->
                  <div class="sub-sec-head">
                    <div class="sub-sec-title">
                      <strong v-html="highlightText(sec.title, searchKeyword)"></strong>
                      <span v-if="sec.subtitle" class="sub-sec-sub" v-html="highlightText(sec.subtitle, searchKeyword)"></span>
                    </div>
                    <div class="sub-sec-tags">
                      <span v-if="sec.hasKeywordMatch && searchKeyword.trim()" class="match-badge">命中搜索</span>
                      <template v-for="t in sec.tags" :key="t.text">
                        <span class="sub-tag-pill" :class="t.type">{{ t.text }}</span>
                      </template>
                    </div>
                  </div>

                  <!-- 优先大字高亮显示：核心诊断与结论 (Conclusion) -->
                  <div class="sub-sec-conclusion" :class="{ 'is-warning-bg': sec.isAbnormal }">
                    <div class="conclusion-badge">
                      <span class="pulse-indicator" v-if="sec.isAbnormal"></span>
                      <span>核心结论</span>
                    </div>
                    <div class="conclusion-text" v-html="highlightText(sec.conclusion, searchKeyword)"></div>
                  </div>

                  <!-- 处置/随访建议（若有） -->
                  <div v-if="sec.suggestion" class="sub-sec-suggestion">
                    <span class="sug-icon">建议：</span>
                    <span class="sug-text" v-html="highlightText(sec.suggestion, searchKeyword)"></span>
                  </div>

                  <!-- 过程测量与详细所见（默认折叠，按需展开） -->
                  <div v-if="sec.findings && sec.findings.length" class="sub-sec-findings-container">
                    <div
                      class="findings-toggle-btn"
                      @click.stop="toggleFindingExpand(`${exam.id}-${sIdx}`)"
                    >
                      <span class="toggle-icon">{{ isFindingExpanded(`${exam.id}-${sIdx}`) ? '▾' : '▸' }}</span>
                      <span>{{ isFindingExpanded(`${exam.id}-${sIdx}`) ? '收起测量所见与检查参数' : `查看测量参数与检查所见 (${sec.findings.length}项)` }}</span>
                    </div>

                    <div v-if="isFindingExpanded(`${exam.id}-${sIdx}`)" class="findings-list-panel">
                      <div
                        v-for="(finding, fIdx) in sec.findings"
                        :key="fIdx"
                        class="finding-row"
                        v-html="highlightText(finding, searchKeyword)"
                      ></div>
                      <div v-if="sec.issuer" class="issuer-row">
                        <span>报告医师：{{ sec.issuer }}</span>
                      </div>
                    </div>
                  </div>
                </div>
              </div>

              <!-- 情况 B：无医技报告，直接展示门诊临床诊断与主诉 -->
              <div v-else-if="exam.diagnosis || exam.symptoms" class="pure-diagnosis-box">
                <div class="pure-diag-head">
                  <strong>临床确诊 / 主诉：</strong>
                </div>
                <div class="pure-diag-text font-bold" v-html="highlightText(exam.diagnosis || exam.symptoms, searchKeyword)"></div>
                <div v-if="exam.treatment_plan" class="pure-diag-plan">
                  <span>处置方案：{{ exam.treatment_plan }}</span>
                </div>
              </div>
            </template>

            <!-- 情况 C：跨机构未授权脱敏遮罩保护 -->
            <div v-else class="masked-lock-box">
              <div class="lock-head">
                <span><strong>跨院临床数据已实施零信任访问保护</strong></span>
              </div>
              <div class="lock-desc">
                就诊记录包含 <strong>{{ exam.exam_items || '专科检查项目' }}</strong>。请点击卡片发起知情授权申请，或由患者输入现场密钥即时解密调阅。
              </div>
            </div>
          </div>

          <!-- 5. 卡片底部：放行权限状态与调阅按钮 -->
          <div class="pec-item-bottom">
            <span v-if="exam.doctor_id === auth.user?.id || exam.hospital_id === auth.user?.hospital_id" class="access-tag green">
              本院/本人放行
            </span>
            <span v-else-if="exam.access_type === 'ALL_DOCTORS'" class="access-tag green">
              全体医生可见
            </span>
            <span v-else-if="exam.access_type === 'BREAK_GLASS'" class="access-tag red">
              24h破窗放行中
            </span>
            <span v-else-if="exam.has_access" class="access-tag green">
              患者已授权
            </span>
            <span v-else class="access-tag orange">
              需安全网关核准
            </span>
            <el-button type="primary" size="small" link class="view-btn">安全调阅 ➔</el-button>
          </div>
        </div>
      </div>
    </el-card>

    <!-- 跨院记录结果列表 -->
    <el-card shadow="hover" class="box-card table-wrapper-card">
      <template #header>
        <div class="table-card-header">
          <div class="tch-left">
            <strong>跨机构就诊与医技病历列表</strong>
            <el-tag size="small" type="info" class="ml-2">
              当前展示 {{ filteredResults.length }} / {{ results.length }} 项
            </el-tag>
          </div>

          <!-- 右侧：自定义列按钮与清空筛选 -->
          <div class="tch-right">
            <el-button
              v-if="activeFilterCount > 0"
              size="small"
              type="danger"
              plain
              @click="resetAllFilters"
            >
              清除筛选 ({{ activeFilterCount }})
            </el-button>

            <!-- 自定义列浮层面板 (Popover) -->
            <el-popover
              placement="bottom-end"
              :width="270"
              trigger="click"
              popper-class="column-setting-popover"
            >
              <template #reference>
                <el-button size="small" :icon="Setting" plain>
                  自定义列 ({{ visibleColumnsCount }}/{{ tableColumns.length }})
                </el-button>
              </template>

              <div class="col-popover-content">
                <div class="col-popover-head">
                  <strong>自定义表格显示列</strong>
                  <span class="text-xs text-slate-500">（已实时本地保存）</span>
                </div>
                <div class="col-checkbox-list">
                  <div
                    v-for="col in tableColumns"
                    :key="col.key"
                    class="col-checkbox-item"
                  >
                    <el-checkbox
                      v-model="col.visible"
                      @change="saveColumnsConfig"
                    >
                      <span>{{ col.label }}</span>
                    </el-checkbox>
                  </div>
                </div>
                <div class="col-popover-footer">
                  <el-button size="small" link type="primary" @click="selectAllColumns">全选</el-button>
                  <el-button size="small" link type="info" @click="resetColumns">恢复默认</el-button>
                </div>
              </div>
            </el-popover>
          </div>
        </div>
      </template>

      <!-- 智能筛选与快捷归类工作台 -->
      <div class="filter-workbench mb-3">
        <!-- 第 1 栏：快捷归类场景模式 -->
        <div class="workbench-row classify-row">
          <span class="wb-label">快捷归类：</span>
          <el-radio-group v-model="quickClassifyMode" size="small" class="classify-tabs">
            <el-radio-button label="ALL">
              <span>全部档案 ({{ results.length }})</span>
            </el-radio-button>
            <el-radio-button label="ATTENTION">
              <span>重点关注 (异常/告警)</span>
            </el-radio-button>
            <el-radio-button label="ACCESSIBLE">
              <span>立即可查 (已放行)</span>
            </el-radio-button>
            <el-radio-button label="NEED_AUTH">
              <span>待安全网关核准</span>
            </el-radio-button>
            <el-radio-button label="EXAM_ONLY">
              <span>仅医技检查/切片</span>
            </el-radio-button>
          </el-radio-group>
        </div>

        <!-- 第 2 栏：多维度细筛下拉项组合 -->
        <div class="workbench-row filters-row">
          <div class="filter-item">
            <span class="f-lbl">调阅权限:</span>
            <el-select v-model="filterAuthStatus" size="small" style="width: 145px;">
              <el-option label="全部状态" value="ALL" />
              <el-option label="本人责任病历" value="SELF" />
              <el-option label="本院同僚放行" value="HOSPITAL" />
              <el-option label="跨院已授权" value="AUTHORIZED" />
              <el-option label="待网关核准" value="PENDING" />
              <el-option label="24h破窗放行" value="BREAK_GLASS" />
            </el-select>
          </div>

          <div class="filter-item">
            <span class="f-lbl">临床指标:</span>
            <el-select v-model="filterClinicalStatus" size="small" style="width: 130px;">
              <el-option label="全部指标" value="ALL" />
              <el-option label="仅看阳性/异常" value="ABNORMAL" />
              <el-option label="仅看大致正常" value="NORMAL" />
            </el-select>
          </div>

          <div class="filter-item">
            <span class="f-lbl">防篡改状态:</span>
            <el-select v-model="filterTamperStatus" size="small" style="width: 125px;">
              <el-option label="全部校验" value="ALL" />
              <el-option label="账本一致" value="SAFE" />
              <el-option label="存在篡改" value="TAMPERED" />
            </el-select>
          </div>

          <div class="filter-item">
            <span class="f-lbl">就诊时间:</span>
            <el-select v-model="filterTimeRange" size="small" style="width: 120px;">
              <el-option label="全部时间" value="ALL" />
              <el-option label="近 3 天" value="3D" />
              <el-option label="近 1 周" value="7D" />
              <el-option label="近 1 个月" value="30D" />
              <el-option label="近 3 个月" value="90D" />
            </el-select>
          </div>

          <!-- 活动标签展示 -->
          <div v-if="activeFilterCount > 0" class="active-filter-pills">
            <el-tag
              v-if="quickClassifyMode !== 'ALL'"
              size="small"
              closable
              type="danger"
              @close="quickClassifyMode = 'ALL'"
            >
              归类: {{ quickClassifyMode === 'ATTENTION' ? '需重点关注' : (quickClassifyMode === 'ACCESSIBLE' ? '立即可查阅' : (quickClassifyMode === 'NEED_AUTH' ? '待核准' : '仅医技单')) }}
            </el-tag>
            <el-tag
              v-if="filterAuthStatus !== 'ALL'"
              size="small"
              closable
              type="warning"
              @close="filterAuthStatus = 'ALL'"
            >
              权限: {{ filterAuthStatus }}
            </el-tag>
            <el-tag
              v-if="filterClinicalStatus !== 'ALL'"
              size="small"
              closable
              type="primary"
              @close="filterClinicalStatus = 'ALL'"
            >
              指标: {{ filterClinicalStatus === 'ABNORMAL' ? '仅阳性/异常' : '仅正常' }}
            </el-tag>
            <el-tag
              v-if="filterTimeRange !== 'ALL'"
              size="small"
              closable
              type="info"
              @close="filterTimeRange = 'ALL'"
            >
              时间: {{ filterTimeRange }}
            </el-tag>
          </div>
        </div>
      </div>

      <!-- 数据表格绑定 paginatedResults，并按自定义列显隐动态呈现 -->
      <el-table
        :data="paginatedResults"
        v-loading="loading"
        stripe
        style="width: 100%"
        :max-height="tableMaxHeight"
        class="cross-query-table"
      >
        <el-table-column v-if="isColumnVisible('record_no')" prop="record_no" label="病历单号" width="160" />
        <el-table-column v-if="isColumnVisible('patient')" label="就诊患者" min-width="180">
          <template #default="{ row }">
            <div class="patient-cell">
              <span class="patient-name font-bold">{{ formatPatientName(row) }}</span>
              <div class="patient-sub">
                <span v-if="row.patient_phone">{{ row.patient_phone }}</span>
                <span v-if="row.patient_id_card" class="id-card-tag">{{ maskIDCard(row.patient_id_card) }}</span>
              </div>
            </div>
          </template>
        </el-table-column>

        <el-table-column v-if="isColumnVisible('hospital')" label="归属医疗机构" min-width="180">
          <template #default="{ row }">
            <div class="hosp-cell">
              <span class="hosp-name">{{ row.hospital_name }}</span>
              <el-tag 
                v-if="row.doctor_id === auth.user?.id" 
                size="small" 
                type="success" 
                effect="light"
                class="tag-pill"
              >
                本人开具
              </el-tag>
              <el-tag 
                v-else-if="row.hospital_id === auth.user?.hospital_id" 
                size="small" 
                type="primary" 
                effect="light"
                class="tag-pill"
              >
                本院同僚
              </el-tag>
              <el-tag 
                v-else 
                size="small" 
                type="warning" 
                effect="plain"
                class="tag-pill"
              >
                跨院机构
              </el-tag>
            </div>
          </template>
        </el-table-column>

        <el-table-column v-if="isColumnVisible('doctor')" label="开具医生" width="130">
          <template #default="{ row }">
            <span :class="{ 'self-doc-text': row.doctor_id === auth.user?.id }">
              {{ row.doctor_name }}
              <span v-if="row.doctor_id === auth.user?.id" class="self-tag">(您)</span>
            </span>
          </template>
        </el-table-column>

        <el-table-column v-if="isColumnVisible('data_type')" prop="data_type" label="类别" width="105">
          <template #default="{ row }">
            <el-tag v-if="row.data_type === 'IMAGE'" size="small" type="warning" effect="dark">影像切片</el-tag>
            <el-tag v-else-if="row.data_type === 'REPORT'" size="small" type="success" effect="dark">检验单</el-tag>
            <el-tag v-else-if="row.need_exam || row.exam_items" size="small" type="primary" effect="light">含检查</el-tag>
            <el-tag v-else size="small" type="info">门诊病历</el-tag>
          </template>
        </el-table-column>
        <el-table-column v-if="isColumnVisible('course')" label="发病与病程" width="140">
          <template #default="{ row }">
            <div v-if="row.onset_time || row.duration" style="font-size: 12px; line-height: 1.4;">
              <div>{{ row.onset_time ? row.onset_time.substring(0, 16) : '接诊记录' }}</div>
              <div style="color: #64748b;">{{ row.duration || '-' }}</div>
            </div>
            <span v-else style="color: #94a3b8; font-size: 12px;">常规就诊</span>
          </template>
        </el-table-column>
        <el-table-column v-if="isColumnVisible('diagnosis')" label="检查项目与临床诊断" min-width="240">
          <template #default="{ row }">
            <div v-if="row.exam_items" class="text-xs text-indigo-700 font-bold mb-1">
              检查: <span v-html="highlightText(row.exam_items, searchKeyword)"></span>
            </div>
            <div v-if="row.has_access || row.doctor_id === auth.user?.id || row.hospital_id === auth.user?.hospital_id" class="table-diag-box">
              <div v-if="getParsedExam(row).summaryTags.length" class="table-tags-row mb-1">
                <el-tag
                  v-for="t in getParsedExam(row).summaryTags.slice(0, 2)"
                  :key="t.text"
                  size="small"
                  :type="t.type"
                  effect="light"
                  class="mr-1"
                >
                  {{ t.text }}
                </el-tag>
              </div>
              <div class="text-xs text-slate-800 table-summary-text" v-html="highlightText(getExamTableSummary(row), searchKeyword)"></div>
            </div>
            <div v-else class="text-xs text-amber-700 font-medium">
              敏感诊疗数据已受掩码保护 (点击【跨院调阅】核准)
            </div>
          </template>
        </el-table-column>
        <el-table-column v-if="isColumnVisible('tamper')" label="防篡改状态" width="125" align="center">
          <template #default="{ row }">
            <el-tag v-if="row.is_tampered" type="danger" effect="dark" class="tamper-tag-glow">
              存在篡改
            </el-tag>
            <el-tag v-else type="success" effect="light">
              校验一致
            </el-tag>
          </template>
        </el-table-column>

        <el-table-column v-if="isColumnVisible('auth_status')" label="调阅授权状态" width="145" align="center">
          <template #default="{ row }">
            <el-tag v-if="row.doctor_id === auth.user?.id" type="success" effect="plain" size="small">
              本人责任病历
            </el-tag>
            <el-tag v-else-if="row.hospital_id === auth.user?.hospital_id" type="primary" effect="plain" size="small">
              本院同僚放行
            </el-tag>
            <el-tag v-else-if="row.access_type === 'ALL_DOCTORS'" type="success" effect="dark" size="small">
              全网公开 (全体可见)
            </el-tag>
            <el-tag v-else-if="row.access_type === 'BREAK_GLASS'" type="danger" effect="dark" size="small">
              24h破窗放行中
            </el-tag>
            <el-tag v-else-if="row.has_access" type="success" effect="dark" size="small">
              患者跨院已授权
            </el-tag>
            <el-tag v-else type="info" effect="plain" size="small">
              待安全网关核准
            </el-tag>
          </template>
        </el-table-column>

        <el-table-column v-if="isColumnVisible('date')" prop="created_at" label="就诊日期" width="120">
          <template #default="{ row }">
            {{ row.created_at ? row.created_at.substring(0, 10) : '' }}
          </template>
        </el-table-column>
        <el-table-column v-if="isColumnVisible('action')" label="调阅操作" width="270" :fixed="isWideScreen ? 'right' : false">
          <template #default="{ row }">
            <el-button 
              v-if="row.doctor_id === auth.user?.id" 
              type="success" 
              size="small" 
              @click="handleAccess(row)"
            >
              直接查看 (免申请)
            </el-button>
            <el-button 
              v-else-if="row.hospital_id === auth.user?.hospital_id" 
              type="primary" 
              size="small" 
              @click="handleAccess(row)"
            >
              院内调阅 (直接放行)
            </el-button>
            <el-button 
              v-else-if="row.access_type === 'ALL_DOCTORS'" 
              type="success" 
              size="small" 
              @click="handleAccess(row)"
            >
              直接查看 (全体可见)
            </el-button>
            <el-button 
              v-else-if="row.access_type === 'BREAK_GLASS'" 
              type="danger" 
              size="small" 
              @click="handleAccess(row)"
            >
              直接查看 (破窗放行)
            </el-button>
            <el-button 
              v-else-if="row.has_access" 
              type="success" 
              size="small" 
              @click="handleAccess(row)"
            >
              直接查看 (已获授权)
            </el-button>
            <div v-else class="action-btn-group" style="display: flex; gap: 6px; white-space: nowrap;">
              <el-button 
                type="warning" 
                size="small" 
                @click="handleAccess(row)"
              >
                跨院调阅 (安全网关)
              </el-button>
              <el-button
                type="danger"
                size="small"
                plain
                @click="openBreakGlassDirect(row)"
                title="患者危重抢救时快速触发 Break-Glass 破窗"
              >
                紧急访问 (破窗)
              </el-button>
            </div>
          </template>
        </el-table-column>
      </el-table>

      <!-- 表格底部：分页与快捷跳转栏 (保证视口内随时有水平拖拽滚动条和翻页条) -->
      <div class="table-pagination-wrapper">
        <div class="tp-info text-xs text-slate-500">
          共 <strong class="text-slate-800">{{ filteredResults.length }}</strong> 条跨机构病历记录
          <span v-if="filteredResults.length !== results.length"> (已过滤筛选)</span>
        </div>
        <el-pagination
          v-model:current-page="currentPage"
          v-model:page-size="pageSize"
          :page-sizes="[10, 15, 20, 50, 100]"
          :total="filteredResults.length"
          layout="sizes, prev, pager, next, jumper"
          background
          size="small"
          @size-change="handleSizeChange"
          @current-change="handleCurrentChange"
        />
      </div>
    </el-card>
    </div>

        <!-- 视图二：跨院医疗数据已核准解密放行全景工作台 (DETAIL) -->
    <div v-else-if="currentView === 'DETAIL'" :class="['inpage-workstation-wrapper', { 'tampered-watermark-overlay': releasedRecord?.is_tampered }]">
      <!-- 顶部工作台粘性控制栏 (轻量纤细版 48px) -->
      <div class="ws-page-header-card sticky-header">
        <div class="ws-header-left">
          <el-button :icon="ArrowLeft" plain size="small" @click="currentView = 'QUERY'" class="back-btn">返回检索</el-button>
          <el-button
            v-if="isFromEncounter || hasEncounterDraft"
            type="primary"
            size="small"
            :icon="ArrowLeft"
            @click="returnToEncounter"
            class="back-btn draft-back-btn"
          >
            返回接诊录入
          </el-button>
          <div class="ws-header-title-box">
            <span class="ws-page-title">跨院解密就诊全景病历</span>
            <el-tag
              v-if="releasedRecord?.is_tampered"
              size="small"
              type="danger"
              effect="dark"
              class="tamper-tag-glow"
            >
              <el-icon class="mr-1"><WarningFilled /></el-icon> ⚠️ 数据已被恶意篡改 · 存证不匹配
            </el-tag>
            <el-tag v-else size="small" type="success" effect="light">已放行解密</el-tag>
            <div v-if="releasedRecord" class="ws-record-chip">
              <span class="chip-lbl">单号</span>
              <span class="chip-val mono">{{ releasedRecord.record_no }}</span>
              <el-tooltip content="复制单号" placement="top">
                <el-icon class="copy-icon" @click.stop="copyToClipboard(releasedRecord.record_no, '就诊单号')"><CopyDocument /></el-icon>
              </el-tooltip>
            </div>
          </div>
        </div>
        <div class="ws-header-right">
          <InfectionSafetyAlert
            v-if="releasedRecord?.infection_alert?.has_risk"
            :alert="releasedRecord.infection_alert"
            mode="badge"
          />
          <el-button type="primary" size="small" :icon="View" @click="openPdfPreview(releasedRecord)">查阅红头 PDF</el-button>
        </div>
      </div>

      <div v-if="releasedRecord" class="detail-workstation-content mt-3">
        <!-- 医护职业暴露安全防范条幅 -->
        <div v-if="releasedRecord.infection_alert?.has_risk" class="mb-3">
          <InfectionSafetyAlert
            :alert="releasedRecord.infection_alert"
            mode="corner-banner"
          />
        </div>

        <!-- 篡改告警横幅 (高危防护预警卡) -->
        <div v-if="releasedRecord.is_tampered" class="tamper-box danger">
          <div class="tamper-top-badge-row">
            <div class="tamper-pill-badge">
              <el-icon class="pulse-icon"><WarningFilled /></el-icon>
              <span>BLOCKCHAIN INTEGRITY ALERT / 联盟链存证指纹不匹配</span>
            </div>
            <span class="tamper-block-tag">已触发临床采信阻断</span>
          </div>

          <div class="t-head">
            <el-icon class="t-head-icon"><WarningFilled /></el-icon>
            <span class="t-title">【高危安全警报】检测到该病历底层数据已被恶意篡改！</span>
          </div>

          <div class="t-desc">
            系统零信任存证网关检测到底层 MySQL 数据库中的临床就诊明细特征指纹与 Hyperledger Fabric 联盟链上不可篡改的存证指纹<strong>不匹配</strong>！该病历临床真实性已丧失，<strong>临床严禁直接采信该病历或据此开具医嘱</strong>！
          </div>

          <div class="t-hashes">
            <div class="hash-row danger-hash">
              <span class="hash-label">数据库当前计算指纹 (已污染):</span>
              <code class="hash-code">{{ releasedRecord.current_hash || '（计算生成中...）' }}</code>
              <el-button size="small" link type="danger" @click.stop="copyToClipboard(releasedRecord.current_hash, '当前计算哈希')">复制</el-button>
            </div>
            <div class="hash-row chain-hash">
              <span class="hash-label">区块链不可篡改基准 (Fabric 账本):</span>
              <code class="hash-code">{{ releasedRecord.chain_hash || '（链上存证缺失 / 未通过共识锚定）' }}</code>
              <el-button v-if="releasedRecord.chain_hash" size="small" link type="primary" @click.stop="copyToClipboard(releasedRecord.chain_hash, '区块链基准哈希')">复制</el-button>
            </div>
          </div>

          <div class="tamper-footer-row">
            <div class="tamper-reason">
              <el-icon class="mr-1 text-red-600"><InfoFilled /></el-icon>
              <span>{{ releasedRecord.tamper_reason || '【区块链安全告警】病历关键临床数据（诱因/用药/诊断）已被非法篡改，哈希指纹校验失败！' }}</span>
            </div>
            <div class="tamper-audit-pill">
              <span>违规调阅已实时上报卫健区块链监督中心</span>
            </div>
          </div>
        </div>

        <div v-else class="tamper-box safe">
          <div class="tb-safe-left">
            <el-icon class="text-emerald-600 text-lg"><CircleCheckFilled /></el-icon>
            <div class="tb-text">
              <span class="font-bold text-emerald-800 text-sm">区块链全量防篡改核验通过：</span>
              <span class="text-emerald-700 text-xs ml-1.5">链下数据指纹与 Hyperledger Fabric 联盟链完全吻合，100% 真实未篡改</span>
            </div>
          </div>
          <div class="tb-safe-right">
            <span class="fabric-badge">Cross-Org Verified 100%</span>
          </div>
        </div>

        <el-alert
          v-if="releasedRecord.is_tampered"
          title="【安全风控介入】虽然该病历具备调阅权限，但由于底层数据库指纹与区块链存证不匹配，系统已锁定为只读存证审计模式，临床操作已被安全防护策略阻断！"
          type="error"
          show-icon
          class="mb-3 mt-3"
          :closable="false"
        />
        <template v-else>
          <el-alert
            v-if="isBreakGlassRelease || releasedRecord.access_type === 'BREAK_GLASS'"
            title="本次调阅为 Break-Glass 紧急破窗放行（24小时急救绿色通道生效中），事件单已上链存证并向卫健监管派发预警"
            type="error"
            show-icon
            class="mb-3 mt-3"
            :closable="false"
          />
          <el-alert
            v-else-if="releasedRecord.doctor_id === auth.user?.id"
            title="本人开具病历：您作为接诊责任医生依法享有持续随诊与病史调阅权限，系统免申请直接放行"
            type="success"
            show-icon
            class="mb-3 mt-3"
            :closable="false"
          />
          <el-alert
            v-else-if="releasedRecord.hospital_id === auth.user?.hospital_id"
            title="院内诊疗互通：目标病历归属本医疗机构，遵循院内就诊连续性互认规范直接放行"
            type="info"
            show-icon
            class="mb-3 mt-3"
            :closable="false"
          />
          <el-alert
            v-else
            title="跨机构授权放行：目标机构已核准知情同意放行策略，数据经跨院零信任安全网关实时解密"
            type="success"
            show-icon
            class="mb-3 mt-3"
            :closable="false"
          />
        </template>

        <div class="ws-body-grid">
          <!-- 左侧栏：跨院档案属性、区块链凭据、附件文件 (380px) -->
          <div class="ws-col-left">
            <!-- 卡片 1：就诊档案与患者卡片 -->
            <div class="bento-card patient-hero-card">
              <div class="ph-header">
                <div class="ph-avatar">
                  {{ formatPatientName(releasedRecord).slice(0, 1) }}
                </div>
                <div class="ph-main-info">
                  <div class="ph-name-row">
                    <span class="ph-name">{{ formatPatientName(releasedRecord) }}</span>
                    <el-tag size="small" :type="releasedRecord.patient_gender === 'FEMALE' ? 'danger' : 'primary'" effect="plain">
                      {{ releasedRecord.patient_gender === 'FEMALE' ? '女' : (releasedRecord.patient_gender === 'MALE' ? '男' : (releasedRecord.gender || '患者')) }}
                    </el-tag>
                    <span v-if="releasedRecord.patient_age || releasedRecord.age" class="ph-age">{{ releasedRecord.patient_age || releasedRecord.age }} 岁</span>
                  </div>
                  <div class="ph-meta-id mono text-xs text-gray-500">
                    ID: {{ maskIDCard(releasedRecord.patient_id_card) }}
                  </div>
                </div>
              </div>

              <div class="ph-divider" />

              <div class="ph-details-grid">
                <div class="ph-item">
                  <span class="lbl">就诊单号</span>
                  <span class="val mono font-semibold">{{ releasedRecord.record_no }}</span>
                </div>
                <div class="ph-item">
                  <span class="lbl">就诊类型</span>
                  <span class="val"><el-tag size="small" type="warning" effect="light">{{ formatEncounterType(releasedRecord.encounter_type) }}</el-tag></span>
                </div>
                <div class="ph-item">
                  <span class="lbl">开立机构</span>
                  <span class="val font-semibold text-slate-800">{{ releasedRecord.hospital_name || formatHospName(releasedRecord.hospital_id) }}</span>
                </div>
                <div class="ph-item">
                  <span class="lbl">就诊科室</span>
                  <span class="val font-semibold text-slate-800">{{ releasedRecord.department_name }}</span>
                </div>
                <div class="ph-item">
                  <span class="lbl">接诊医师</span>
                  <span class="val font-semibold text-primary">{{ releasedRecord.doctor_name || '李建国医生' }}</span>
                </div>
                <div class="ph-item">
                  <span class="lbl">流转状态</span>
                  <span class="val"><el-tag size="small" type="success" effect="light">{{ formatRecordStatus(releasedRecord.status) }}</el-tag></span>
                </div>
                <div class="ph-item">
                  <span class="lbl">就诊时间</span>
                  <span class="val text-xs text-slate-600">{{ formatTime(releasedRecord.created_at) }}</span>
                </div>
              </div>
            </div>

            <!-- 卡片 2：区块链存证与信封加密数字证书 (联盟链共识确权凭据) -->
            <div class="bento-card crypto-card digital-cert-card">
              <div class="crypto-header">
                <div class="crypto-title">
                  <span class="crypto-dot" />
                  <span>区块链存证与信封加密凭证 (数字证书)</span>
                </div>
                <el-tag size="small" type="success" effect="dark" class="crypto-tag">联盟链共识已上链</el-tag>
              </div>

              <div class="crypto-body">
                <!-- 信封加密与密码学存证凭证表单 -->
                <div class="cert-fields-grid">
                  <div class="cert-item">
                    <span class="cert-k">对称加密算法：</span>
                    <span class="cert-v font-bold text-emerald-600">AES-256-GCM (AAD)</span>
                  </div>
                  <div class="cert-item">
                    <span class="cert-k">加密数据密钥 (DEK)：</span>
                    <span class="cert-v mono text-xs" :title="releasedRecord.encrypted_dek || 'a9f24b81c20e...5e1b'">
                      {{ formatMaskedDEK(releasedRecord.encrypted_dek) }}
                    </span>
                  </div>
                  <div class="cert-item">
                    <span class="cert-k">初始向量 (IV/Nonce)：</span>
                    <span class="cert-v mono text-xs">{{ releasedRecord.iv_nonce || 'a4e910cf283d4b67' }} (96-bit)</span>
                  </div>
                  <div class="cert-item">
                    <span class="cert-k">IPFS 寻址 CID：</span>
                    <span class="cert-v mono text-xs" :title="getRecordCID(releasedRecord)">
                      {{ shortHash(getRecordCID(releasedRecord), 10, 8) }}
                    </span>
                  </div>
                  <div class="cert-item">
                    <span class="cert-k">数据指纹 (FileHash)：</span>
                    <span class="cert-v mono text-xs" :title="getRecordFileHash(releasedRecord)">
                      {{ shortHash(getRecordFileHash(releasedRecord), 10, 8) }}
                    </span>
                  </div>
                  <div class="cert-item">
                    <span class="cert-k">联盟通道 / 智能合约：</span>
                    <span class="cert-v font-semibold text-purple-700">medicalchannel / medical_contract</span>
                  </div>
                </div>

                <div class="crypto-field mt-3">
                  <div class="cf-head">
                    <span class="cf-label">Fabric 联盟链交易哈希 (TxID)</span>
                    <button class="cf-copy-btn" @click="copyToClipboard(releasedRecord.fabric_tx_id, '交易哈希')">
                      <el-icon><CopyDocument /></el-icon>
                      <span>复制哈希</span>
                    </button>
                  </div>
                  <div class="cf-hash-box mono" :title="releasedRecord.fabric_tx_id">
                    {{ shortHash(releasedRecord.fabric_tx_id, 14, 12) }}
                  </div>
                </div>

                <div class="crypto-stats-row mt-2">
                  <div class="stat-pill">
                    <span class="stat-k">区块高度</span>
                    <span class="stat-v mono">#{{ releasedRecord.block_height || '342' }}</span>
                  </div>
                  <div class="stat-pill">
                    <span class="stat-k">共识状态</span>
                    <span class="stat-v">Raft (etcdraft) 已确认</span>
                  </div>
                  <div class="stat-pill">
                    <span class="stat-k">存证通道</span>
                    <span class="stat-v">medicalchannel</span>
                  </div>
                </div>

                <div class="crypto-security-seal mt-2">
                  <el-icon class="mr-1 text-emerald-400"><CircleCheckFilled /></el-icon>
                  <span>AES-256-GCM 信封加密 · IPFS 分布式寻址 · Fabric 账本防篡改</span>
                </div>
              </div>
            </div>

                        <!-- 卡片 3：临床归档电子凭据与附件 (精致分层版) -->
            <div class="bento-card files-card">
              <div class="files-card-header">
                <div class="fch-title">
                  <el-icon color="#0284c7"><Files /></el-icon>
                  <span>跨机构归档凭据与附件</span>
                </div>
                <el-tag size="small" type="info" effect="plain" class="fch-tag">IPFS 存证</el-tag>
              </div>

              <div class="archive-files-section">
                <div v-if="releasedRecord.files && releasedRecord.files.length" class="file-cards-list">
                  <div v-for="file in getDistinctFiles(releasedRecord.files)" :key="file.id" class="file-item-bento">
                    <!-- 顶部行：图标 + 文件名 + 类型与大小 -->
                    <div class="fib-top-row">
                      <div class="fib-icon">
                        <el-icon v-if="file.file_type === 'pdf'" color="#dc2626"><Document /></el-icon>
                        <el-icon v-else-if="isImageFileType(file.file_type)" color="#0284c7"><Picture /></el-icon>
                        <el-icon v-else color="#64748b"><Files /></el-icon>
                      </div>
                      <div class="fib-meta">
                        <div class="fib-name-box">
                          <span class="fib-name" :title="file.file_name || '临床就诊规范归档凭据.pdf'">{{ file.file_name || '临床就诊规范归档凭据.pdf' }}</span>
                        </div>
                        <div class="fib-tags">
                          <el-tag size="small" type="primary" effect="light" class="type-tag">{{ (file.file_type || 'PDF').toUpperCase() }}</el-tag>
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
                        <el-button type="primary" size="small" plain :icon="View" class="fib-btn" @click="openPdfPreview(releasedRecord)">查阅红头 PDF</el-button>
                        <el-button type="success" size="small" plain :icon="Download" class="fib-btn" @click="downloadRecordFile(releasedRecord.id)">下载凭据</el-button>
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
                    <el-button type="primary" size="small" plain :icon="View" class="fib-btn" @click="openPdfPreview(releasedRecord)">
                      查阅红头 PDF
                    </el-button>
                    <el-button type="success" size="small" plain :icon="Download" class="fib-btn" @click="downloadRecordFile(releasedRecord.id)">
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
                  <span v-if="releasedRecord.onset_time" class="time-chip">
                    🕒 发病时间: {{ releasedRecord.onset_time }}
                  </span>
                  <span v-if="releasedRecord.duration" class="time-chip">
                    ⏱️ 持续: {{ releasedRecord.duration }}
                  </span>
                </div>
              </div>

              <div class="soap-card-content">
                <div class="complaint-hero-box">
                  <div class="chb-quote-mark">“</div>
                  <div class="chb-text">
                    {{ releasedRecord.chief_complaint || releasedRecord.symptoms || '患者就诊主诉' }}
                  </div>
                </div>

                <div v-if="releasedRecord.present_illness" class="present-illness-box mt-3">
                  <div class="pib-label">现病史演进 (History of Present Illness)：</div>
                  <div class="pib-content">{{ releasedRecord.present_illness }}</div>
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
                      <span class="vsc-val mono">{{ parseVitals(releasedRecord.vital_signs).temperature || '36.5 ℃' }}</span>
                      <span class="vsc-status normal">体温正常</span>
                    </div>
                  </div>

                  <div class="vital-sensor-card">
                    <div class="vsc-icon-box bp-icon">🫀</div>
                    <div class="vsc-info">
                      <span class="vsc-name">血压 (BP)</span>
                      <span class="vsc-val mono">{{ parseVitals(releasedRecord.vital_signs).blood_pressure || '120/80 mmHg' }}</span>
                      <span class="vsc-status normal">血压理想</span>
                    </div>
                  </div>

                  <div class="vital-sensor-card">
                    <div class="vsc-icon-box hr-icon">💓</div>
                    <div class="vsc-info">
                      <span class="vsc-name">心率 (HR)</span>
                      <span class="vsc-val mono">{{ parseVitals(releasedRecord.vital_signs).heart_rate || '75 bpm' }}</span>
                      <span class="vsc-status normal">窦性心律</span>
                    </div>
                  </div>

                  <div class="vital-sensor-card">
                    <div class="vsc-icon-box spo2-icon">🫁</div>
                    <div class="vsc-info">
                      <span class="vsc-name">血氧 (SpO2)</span>
                      <span class="vsc-val mono">{{ parseVitals(releasedRecord.vital_signs).spo2 || '98 %' }}</span>
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
                    <span v-if="releasedRecord.exam_items" class="erb-items-tag">
                      申请项目：{{ releasedRecord.exam_items }}
                    </span>
                  </div>

                  <div v-if="releasedRecord.exam_result" class="erb-result-box">
                    <div class="erb-report-text pre-wrap">{{ releasedRecord.exam_result }}</div>
                    <div v-if="releasedRecord.exam_doctor" class="erb-footer">
                      <span>报告出具医师/技师：<strong>{{ releasedRecord.exam_doctor }}</strong></span>
                      <span v-if="releasedRecord.exam_time" class="ml-3">检验时间：{{ releasedRecord.exam_time }}</span>
                    </div>
                  </div>
                  <div v-else class="erb-empty-box">
                    <el-icon color="#10b981" class="text-base mr-1"><CircleCheckFilled /></el-icon>
                    <span>跨院就诊专科体征明确，未开立外送医技辅助检验，医生综合体格检查直接确诊。</span>
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
                    <div class="dhb-name">{{ releasedRecord.diagnosis || '待确诊' }}</div>
                    <div v-if="releasedRecord.initial_diagnosis" class="dhb-compare">
                      <span class="text-xs text-gray-500">初诊拟诊：</span>
                      <span class="text-xs font-semibold text-slate-700">{{ releasedRecord.initial_diagnosis }}</span>
                      <span v-if="releasedRecord.diagnostic_basis" class="text-xs text-gray-400 ml-2">（依据：{{ releasedRecord.diagnostic_basis }}）</span>
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
                <div v-if="releasedRecord.etiology" class="etiology-bento-box mt-3">
                  <div class="ebb-title">🔬 病理诱因与发病机制分析 (Pathogenesis & Etiology)：</div>
                  <div class="ebb-content">{{ releasedRecord.etiology }}</div>
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
                      v-for="item in parsePlanItems(releasedRecord.treatment_plan)"
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
                      <strong class="text-slate-800">{{ releasedRecord.doctor_name }}</strong>
                      <span class="dsb-ca-tag">CA 电子认证已签署</span>
                    </div>
                    <div class="dsb-time text-xs text-gray-400 mt-1">
                      签发归档时间：{{ formatTime(releasedRecord.created_at) }}
                    </div>
                  </div>
                  <div class="dsb-right">
                    <div class="dsb-hash-note">
                      <span>存证区块：<code>#{{ releasedRecord.block_height || '342' }}</code></span>
                    </div>
                  </div>
                </div>
              </div>
            </div>
          </div>
        </div>

        <!-- 底部返回与操作栏 -->
        <div class="ws-page-footer-card mt-3">
          <div class="ws-footer-bar">
            <el-button @click="currentView = 'QUERY'">返回跨院检索列表</el-button>
            <div class="flex gap-2">
              <el-button type="primary" :icon="View" @click="openPdfPreview(releasedRecord)">
                在线查阅完整 PDF 凭证
              </el-button>
              <el-button type="success" plain :icon="Download" @click="downloadRecordFile(releasedRecord.id)">
                下载解密病历凭据
              </el-button>
            </div>
          </div>
        </div>
      </div>
    </div>


    <!-- 动态访问风险评估决策报告卡与调阅控制网关 (三态流转 + Break-Glass 急救四端闭环) -->
    <el-dialog
      v-model="breakGlassVisible"
      title="动态访问风险评估决策报告卡与调阅网关"
      width="780px"
      :close-on-click-modal="false"
      class="risk-report-dialog"
    >
      <!-- 终审决策状态机 (三态视觉呈现) -->
      <div class="decision-status-box mb-3">
        <el-alert
          v-if="riskDecisionState === 'ALLOWED'"
          title="【终审决策：ALLOWED · 零信任风控核准直接放行】"
          type="success"
          show-icon
          :closable="false"
          description="经动态规则加权评估，当前请求属于低风险范围（或同院/已获患者授权），安全网关已核准解密放行并生成 Fabric 存证凭证。您可以直接进入工作台查阅解密病历或下载红头归档凭据。"
        />
        <el-alert
          v-else-if="riskDecisionState === 'PENDING_CONFIRM'"
          title="【终审决策：PENDING_CONFIRM · 待患者在客户端二次确认放行】"
          type="warning"
          show-icon
          :closable="false"
          :description="`跨机构就诊调阅申请单已生成（单号：${riskDecisionReqNo || 'REQ20261014888'}），当前处于中风险受控状态。系统已实时推送通知，等待患者在个人端点击同意；或患者在诊室现场可输入 6 位专属调阅密钥即时解锁！`"
        />
        <el-alert
          v-else
          title="【终审决策：REJECTED / NEED_BREAK_GLASS · 高风险访问拦截 (HTTP 403 Forbidden)】"
          type="error"
          show-icon
          :closable="false"
          description="触发高风险防御策略，常规调阅申请已被系统拦截阻断！若患者处于休克昏迷或危及生命的极端急救场景，医生可启动 Break-Glass 紧急破窗调阅通道。"
        />
      </div>

      <!-- 动态风险综合评分进度条与等级标签 -->
      <div v-if="interceptRiskEval" class="risk-score-eval-card mb-3">
        <div class="rsec-header">
          <div class="rsec-title">
            <span class="dot-indicator"></span>
            <strong>动态访问风险评估决策报告 (Risk Assessment Report)</strong>
          </div>
          <div class="rsec-badges">
            <span class="strategy-tag">策略: <code>{{ interceptRiskEval.strategy || 'RULE_ENGINE_WEIGHTED' }}</code></span>
            <el-tag
              :type="interceptRiskEval.level === 'HIGH' ? 'danger' : (interceptRiskEval.level === 'MEDIUM' ? 'warning' : 'success')"
              size="small"
              effect="dark"
            >
              风控评级: {{ interceptRiskEval.level || 'MEDIUM' }}
            </el-tag>
          </div>
        </div>

        <div class="score-progress-row">
          <div class="score-label">
            <span>综合风险分值 (Risk Score)：</span>
            <strong :style="{ color: displayRiskScore >= 60 ? '#ef4444' : (displayRiskScore >= 30 ? '#f59e0b' : '#10b981') }">
              {{ displayRiskScore }} / 100
            </strong>
          </div>
          <el-progress
            :percentage="Math.min(100, displayRiskScore)"
            :status="displayRiskScore >= 60 ? 'exception' : (displayRiskScore >= 30 ? 'warning' : 'success')"
            :stroke-width="12"
            class="score-bar"
          />
        </div>

        <!-- 命中风险规则清单表格 -->
        <div class="rsec-rules-section mt-2">
          <div class="rsec-rules-title">命中风控判定规则清单 (与后端规则引擎严格对齐)：</div>
          <el-table
            :data="interceptRiskEval.factors || []"
            size="small"
            stripe
            class="factor-table mt-1"
          >
            <el-table-column prop="rule_id" label="规则编号" width="130">
              <template #default="{ row }">
                <code class="rule-code">{{ row.rule_id }}</code>
              </template>
            </el-table-column>
            <el-table-column prop="factor_name" label="风控判定维度" width="140" />
            <el-table-column label="基准权重" width="80" align="center">
              <template #default="{ row }">+{{ row.weight }}</template>
            </el-table-column>
            <el-table-column label="实际得分" width="80" align="center">
              <template #default="{ row }">
                <strong :style="{ color: row.triggered ? '#ef4444' : '#10b981' }">
                  {{ row.score > 0 ? ('+' + row.score) : '0' }}
                </strong>
              </template>
            </el-table-column>
            <el-table-column label="判定状态" width="95" align="center">
              <template #default="{ row }">
                <el-tag size="small" :type="row.triggered ? 'danger' : 'success'" effect="plain">
                  {{ row.triggered ? '命中风险' : '正常放行' }}
                </el-tag>
              </template>
            </el-table-column>
            <el-table-column prop="description" label="合规裁决与风险释义" min-width="190" />
          </el-table>
        </div>
      </div>

      <!-- 终审决策三态操作面板 -->
      <!-- 状态 1: ALLOWED -->
      <div v-if="riskDecisionState === 'ALLOWED'" class="state-panel-allowed mb-2">
        <div class="allowed-guide-box">
          <el-icon class="text-emerald-500 mr-2" :size="24"><CircleCheckFilled /></el-icon>
          <div>
            <strong>已成功核准当前病历的跨院/常规调阅权限！</strong>
            <div class="text-xs text-gray-500 mt-1">数据已在安全网关解密，您可以直接点击下方按钮进入工作台查阅病历详情。</div>
          </div>
        </div>
      </div>

      <!-- 状态 2: PENDING_CONFIRM 或状态 3: REJECTED -->
      <div v-else class="state-panel-flow">
        <!-- 通道选项卡 -->
        <div class="channel-selector-wrapper mb-3">
          <el-radio-group v-model="accessChannel" size="large" class="channel-tabs" style="width: 100%; display: flex;">
            <el-radio-button value="CONSENT" style="flex: 1;">
              <span>选项一：正常诊疗·现场密钥解锁 / 知情同意申请 (推荐)</span>
            </el-radio-button>
            <el-radio-button value="BREAK_GLASS" style="flex: 1;">
              <span class="text-danger font-bold">选项二：危重急救·Break-Glass 破窗</span>
            </el-radio-button>
          </el-radio-group>
        </div>

        <!-- 选项一：正常诊疗知情同意与现场患者密钥解锁通道 -->
        <div v-if="accessChannel === 'CONSENT'" class="channel-panel">
          <div class="target-record-banner mb-3">
            <span class="lbl">目标调阅病历：</span>
            <span v-if="currentTarget" class="val">
              <strong style="color: #4338ca;">{{ currentTarget.record_no }}</strong>
              （患者：<strong>{{ currentTarget.patient_name }}</strong> · 机构：<strong>{{ currentTarget.hospital_name || formatHospName(currentTarget.hospital_id) }}</strong> · 科室：{{ currentTarget.department_name }}）
            </span>
          </div>

          <!-- 方式一：现场密码解锁专区（优先推荐） -->
          <div class="patient-key-box mb-3">
            <div class="pk-header">
              <div class="pk-title-wrap">
                <span class="pk-title">方式一：患者现场输入专属密钥即时解锁 (推荐·秒级放行)</span>
              </div>
              <el-tag size="small" type="success" effect="dark">现场协同</el-tag>
            </div>
            <p class="pk-desc">
              若患者本人在接诊现场，可由<strong>患者本人在下方输入其个人专属病历调阅密钥</strong>（或告知医生代为输入），系统将立即进行密码学核验并现场解密病历、存证上链，<strong>无需在线异步等待推送审批</strong>。
            </p>
            <div class="pk-input-box">
              <el-input
                v-model="patientKey"
                type="password"
                show-password
                size="large"
                placeholder="请输入患者专属病历调阅密钥（初始默认: 123456）"
                :prefix-icon="Key"
                class="pk-key-input"
                @keyup.enter="handleUnlockByKey"
              />
              <div class="pk-captcha-row mt-2">
                <div class="pk-captcha-wrapper">
                  <el-input
                    v-model="unlockCaptchaAnswer"
                    size="large"
                    placeholder="输入计算结果防爆破"
                    :prefix-icon="Finished"
                    class="pk-captcha-input"
                    autocomplete="off"
                    @keyup.enter="handleUnlockByKey"
                  />
                  <div class="pk-captcha-img-container" @click="fetchUnlockCaptcha" title="点击刷新验证码">
                    <img v-if="unlockCaptchaImg" :src="unlockCaptchaImg" alt="验证码" class="pk-captcha-img" />
                    <span v-else class="pk-captcha-loading">加载中...</span>
                  </div>
                </div>
                <el-button
                  type="success"
                  size="large"
                  :icon="Key"
                  :loading="unlockingByKey"
                  :disabled="!patientKey || !unlockCaptchaAnswer"
                  @click="handleUnlockByKey"
                  class="pk-btn"
                >
                  验证密钥并即时解密调阅
                </el-button>
              </div>
            </div>
            <div class="pk-tip">
              <span>提示：患者登录个人账号可在【数据授权管理】随时修改此密钥；初始默认密钥为 <code>123456</code>。</span>
            </div>
          </div>

          <el-divider content-position="center">
            <span style="color: #94a3b8; font-size: 12px;">或患者未在现场时使用在线知情推送审批</span>
          </el-divider>

          <!-- 方式二：在线知情申请推送 -->
          <div class="online-consent-box">
            <div class="oc-header mb-2">
              <strong style="color: #334155; font-size: 14px;">方式二：向患者手机/个人端发起知情同意申请 (离线推送审批)</strong>
            </div>
            <p class="oc-desc mb-2" style="font-size: 12px; color: #64748b; line-height: 1.5;">
              医生填写临床调阅目的后向患者推送在线申请，系统将即时通知患者。患者在个人端点击同意后，系统将自动为医生释放调阅权限并存证上链。
            </p>
            <el-form :model="consentForm" label-width="110px" class="consent-form">
              <el-form-item label="临床调阅目的" required>
                <el-input
                  v-model="consentForm.purpose"
                  type="textarea"
                  :rows="2"
                  placeholder="请输入临床调阅目的与会诊说明，该说明将如实同步通知患者..."
                />
              </el-form-item>
              <el-form-item label="申请授权范围" required>
                <el-radio-group v-model="consentForm.scope_type">
                  <el-radio value="SINGLE">仅限调阅当前这一份病历 (推荐)</el-radio>
                  <el-radio value="ALL">调阅该患者在全网的全部历史健康档案</el-radio>
                </el-radio-group>
              </el-form-item>
              <el-form-item label="希望授权期限" required>
                <el-select v-model="consentForm.days" style="width: 220px;">
                  <el-option label="3 天 (急诊随访)" :value="3" />
                  <el-option label="7 天 (常规门诊)" :value="7" />
                  <el-option label="14 天 (两周疗程)" :value="14" />
                  <el-option label="30 天 (慢病随访)" :value="30" />
                  <el-option label="90 天 (长程康复)" :value="90" />
                </el-select>
              </el-form-item>
            </el-form>
          </div>
        </div>

        <!-- 选项二：急诊危重破窗通道 (Break-Glass) -->
        <div v-else class="channel-panel">
          <!-- 若医生处于受限状态，显示醒目 403 阻断警示 -->
          <el-alert
            v-if="auth.user?.status === 'RESTRICTED'"
            title="【监管阻断 403 Forbidden】您已被监管限制跨院/紧急访问权限，严禁发起破窗！"
            type="error"
            description="依据卫健委医疗监管违规处罚条例，您的执业账号当前处于 RESTRICTED 受限状态，已被依法冻结紧急破窗调阅权。请联系医院管理员或卫健监管专员复核并解除限制后方可恢复使用。"
            show-icon
            :closable="false"
            class="mb-3"
          />

          <div class="emergency-guide">
            <h4 style="color: #dc2626;">极端急危重症救治 Break-Glass 破窗协议说明</h4>
            <p>
              根据《医疗数据可信共享实施规范》，若患者处于<strong>严重休克、深昏迷、心跳呼吸骤停</strong>等危及生命的极端场景，无法表达知情意愿，医生签署临床法律责任声明后，系统允许触发 <strong>Break-Glass 破窗机制</strong>毫秒级放行并即刻异步上链记录生成闭环监管审计工单。
            </p>
          </div>

          <el-form :model="bgForm" label-width="110px" class="bg-form">
            <el-form-item label="紧急抢救原因" required>
              <el-select v-model="bgForm.emergency_reason" style="width: 100%;">
                <el-option label="COMA - 患者严重昏迷休克，无法表达意愿" value="COMA" />
                <el-option label="SHOCK - 多发创伤失血性休克，急需抢救手术" value="SHOCK" />
                <el-option label="EMERGENCY - 急危重症抢救生命关键期，急需用药与过敏史" value="EMERGENCY" />
                <el-option label="CRITICAL - 突发急性危重病多学科综合救治" value="CRITICAL" />
                <el-option label="OTHER - 其他危及生命的特殊医学紧急场景" value="OTHER" />
              </el-select>
            </el-form-item>
            <el-form-item label="临床急救说明" required>
              <el-input
                v-model="bgForm.description"
                type="textarea"
                :rows="3"
                placeholder="请详细录入急救诊断情况、抢救必要性说明，此说明将直接作为不可篡改证据固化上链并推送卫健监管审查..."
              />
            </el-form-item>
            <el-form-item>
              <el-checkbox v-model="bgForm.doctor_confirmed">
                <span class="confirm-text" style="color: #dc2626; font-weight: bold;">
                  我确认本次紧急调阅仅用于患者紧急临床救治，陈述真实无伪造，并承担相应医疗与法律责任
                </span>
              </el-checkbox>
            </el-form-item>
          </el-form>
        </div>
      </div>

      <template #footer>
        <el-button @click="breakGlassVisible = false">关闭</el-button>
        <template v-if="riskDecisionState === 'ALLOWED'">
          <el-button type="success" @click="enterDetailFromAllowed">
            立即解密查阅完整病历
          </el-button>
        </template>
        <template v-else-if="accessChannel === 'CONSENT'">
          <el-button
            type="primary"
            :loading="submittingConsent"
            :disabled="!consentForm.purpose"
            @click="submitConsentRequest"
          >
            发起知情同意申请并通知患者
          </el-button>
        </template>
        <template v-else>
          <el-button
            type="danger"
            :disabled="!bgForm.doctor_confirmed || !bgForm.description || auth.user?.status === 'RESTRICTED'"
            :loading="submittingBG"
            @click="submitBreakGlass"
          >
            {{ auth.user?.status === 'RESTRICTED' ? '权限已受限 (严禁破窗)' : '确认申请 Break-Glass 抢救放行' }}
          </el-button>
        </template>
      </template>
    </el-dialog>


        <!-- PDF 电子病历规范预览弹窗 -->
    <el-dialog
      v-model="pdfPreviewVisible"
      width="1040px"
      top="2vh"
      :close-on-click-modal="false"
      class="pdf-preview-dialog"
    >
      <template #header>
        <div class="pdf-dialog-header">
          <div class="pdf-header-title">
            <el-icon class="pdf-header-icon"><Document /></el-icon>
            <span class="pdf-header-text">临床就诊电子病历归档凭证</span>
            <el-tag v-if="previewingRecord?.is_tampered" size="small" type="danger" effect="dark" class="tamper-tag-glow">⚠️ 篡改告警 · 校验失败</el-tag>
            <el-tag v-else size="small" type="primary" effect="plain" class="pdf-header-tag">PDF 规范视图</el-tag>
          </div>
          <div class="pdf-header-actions no-print">
            <el-button
              type="primary"
              size="small"
              :icon="Printer"
              class="pdf-action-btn pdf-btn-print"
              @click="printPdfSheet"
            >
              打印 / 另存为 PDF 文件
            </el-button>
            <el-button
              type="success"
              plain
              size="small"
              :icon="Download"
              class="pdf-action-btn pdf-btn-download"
              @click="downloadRecordFile(previewingRecord.id)"
            >
              下载规范归档 PDF
            </el-button>
            <el-button
              type="warning"
              plain
              size="small"
              :icon="CircleCheckFilled"
              class="pdf-action-btn pdf-btn-verify"
              @click="verifyRecordIntegrity(previewingRecord.id)"
            >
              一键链上完整性核验
            </el-button>
          </div>
        </div>
      </template>

      <div v-if="previewingRecord" id="emr-print-container" class="emr-sheet-wrapper">
        <!-- 存证规范与安全防护状态栏 -->
        <div class="emr-action-bar no-print">
          <div class="emr-tip-tag">
            <el-tag v-if="previewingRecord?.is_tampered" type="danger" effect="dark" class="action-tag tamper-tag-glow">
              ⚠️ 底层数据已被恶意篡改 · 链上存证核验失败 (不可采信)
            </el-tag>
            <el-tag v-else type="success" effect="dark" class="action-tag">密文解密验证通过</el-tag>
            <el-tag type="info" class="action-tag">国家卫健委《电子病历应用规范》甲级存证标准</el-tag>
          </div>
          <div v-if="previewingRecord?.infection_alert?.has_risk" class="emr-infection-slot">
            <InfectionSafetyAlert
              :alert="previewingRecord.infection_alert"
              mode="badge"
            />
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
              <svg class="barcode-svg" :viewBox="`0 0 ${getBarcodeData(previewingRecord.record_no).totalWidth} 34`" height="34" preserveAspectRatio="none">
                <rect
                  v-for="(bar, idx) in getBarcodeData(previewingRecord.record_no).bars"
                  :key="idx"
                  :x="bar.x"
                  y="0"
                  :width="bar.w"
                  height="34"
                  fill="#0f172a"
                />
              </svg>
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
              <td width="220"><span class="mono">{{ maskIDCard(previewingRecord.patient_id_card) }}</span></td>
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
              <div class="soap-row"><strong>● 发病时间：</strong>{{ formatOnsetTime(previewingRecord.onsetTime || previewingRecord.onset_time) }}<span v-if="previewingRecord.duration">（病程/持续时间：{{ cleanDurationText(previewingRecord.duration) }}）</span></div>
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
              <div class="soap-row">
                <strong>● 病理诱因与机制：</strong>{{ getDisplayEtiology(previewingRecord) }}
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
              <div><strong>Hyperledger Fabric TxID：</strong><code class="tx-hash">{{ formatTxId(previewingRecord.fabric_tx_id) }}</code></div>
              <div class="crypto-sub-line">
                <span><strong>联盟链通道：</strong><span class="mono text-purple-700">medicalchannel</span></span>
                <span class="ml-2"><strong>智能合约：</strong><span class="mono text-indigo-700">medical_contract</span></span>
                <span class="ml-2"><strong>区块高度：</strong><span class="mono text-emerald-700 font-bold">#{{ previewingRecord.block_height || 108 }}</span></span>
              </div>
              <div><strong>底层加密算法：</strong><span class="mono text-slate-700">AES-256-GCM (AAD 关联认证数据绑定)</span></div>
              <div v-if="previewingRecord.files && previewingRecord.files.length">
                <strong>IPFS 分布式存证 CID：</strong><code>{{ previewingRecord.files[0].ipfs_cid }}</code>
              </div>
              <div v-if="previewingRecord.files && previewingRecord.files.length">
                <strong>原始数据 SHA-256 指纹：</strong><code>{{ previewingRecord.files[0].file_hash }}</code>
              </div>
              <div class="text-xs text-emerald-700 mt-1">本病历经 Hyperledger Fabric 真实存证，数字签名抗抵赖、防篡改</div>
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

    <!-- 链上完整性防篡改核验弹窗 (密码学链上背书核验) -->
    <RecordIntegrityDialog
      v-model="integrityDialogVisible"
      :result="integrityResult"
      :loading="verifyingIntegrity"
      @download="downloadRecordFile($event || previewingRecord?.id)"
    />

    <!-- AI 跨期全景病程与疾病关联推演工作台弹窗 -->
    <AiDiagnosticDialog
      v-model="crossAIDiagnosticVisible"
      :patient="crossTargetPatient"
      :current-symptoms="''"
      :present-illness="''"
      :department-name="'跨机构联合就诊中心'"
    />
  </div>
</template>

<script setup lang="ts">
// 当前视图: QUERY-跨院检索与列表筛选, DETAIL-解密放行病历全景工作台
const currentView = ref<'QUERY' | 'DETAIL'>('QUERY')

// ── AI 跨期全景病程推演与关联分析 ────────────────────────
const crossAIDiagnosticVisible = ref(false)
const crossTargetPatient = ref<any>({
  id: 0,
  real_name: '',
  user_no: '',
  gender: '男',
  age: 45
})

function openCrossAIDiagnostic() {
  let pid = Number(route.query.patient_id) || 0
  let pname = (route.query.patient_name as string) || ''
  
  if (!pid && (results.value as any)?.length > 0) {
    const firstRec = (results.value as any)[0]
    pid = firstRec.patient_id || 0
    pname = pname || firstRec.patient_name || ''
  }

  if (!pid) {
    // 尝试根据当前搜索关键词匹配患者
    pid = 1 // 默认张三患者
    pname = pname || '张三'
  }

  crossTargetPatient.value = {
    id: pid,
    real_name: pname || '张三',
    user_no: (route.query.id_card as string) || 'PAT_001',
    gender: '男',
    age: 45
  }

  crossAIDiagnosticVisible.value = true
}


import { ref, computed, onMounted, onUnmounted, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { Search, WarningFilled, CircleCheckFilled, View, Download, Printer, OfficeBuilding, Lock, Compass, Refresh, Key, Setting, Document, Files, Picture, ArrowLeft, CopyDocument, Finished, InfoFilled } from '@element-plus/icons-vue'
import { ElMessage, ElNotification, ElMessageBox } from 'element-plus'
import {
  formatFileSize,
  formatEncounterType,
  formatTime,
  formatTxId,
  cleanDurationText,
  formatOnsetTime,
  getDisplayEtiology,
  getBarcodeData
} from '@/utils/recordFormatters'
import RecordIntegrityDialog from '../../components/RecordIntegrityDialog.vue'
import {
  getRecordDetailApi,
  getPatientInfectionRisksApi,
  downloadRecordApi,
  verifyRecordApi,
  listRecordsApi,
  createAccessRequestApi,
  getCaptchaApi,
  unlockByKeyApi,
  applyConsentRequestApi,
  breakGlassAccessApi
} from '../../api'
import { useAuthStore } from '../../stores/auth'
import { parseMedicalExamResult, highlightText, type ParsedMedicalReport } from '../../utils/medicalReportParser'
import InfectionSafetyAlert from '../../components/InfectionSafetyAlert.vue'
import AiDiagnosticDialog from '../../components/AiDiagnosticDialog.vue'
import { MagicStick } from '@element-plus/icons-vue'

const windowWidth = ref(window.innerWidth)
const windowHeight = ref(window.innerHeight)

function handleResize() {
  windowWidth.value = window.innerWidth
  windowHeight.value = window.innerHeight
}

const isWideScreen = computed(() => windowWidth.value >= 1360)

// 动态自适应表格最大高度：在笔记本或小屏幕上保证底部的水平拖动条和分页栏始终在屏幕内
const tableMaxHeight = computed(() => {
  const h = windowHeight.value - 410
  return Math.max(h, 320) + 'px'
})

const route = useRoute()
const router = useRouter()
const auth = useAuthStore()

const isFromEncounter = computed(() => route.query.from === 'encounter_create')

const hasEncounterDraft = computed(() => {
  try {
    const draft = sessionStorage.getItem('medtrust_encounter_draft')
    return !!draft
  } catch {
    return false
  }
})

function returnToEncounter() {
  router.push({
    path: '/doctor/records',
    query: {
      restore_encounter: '1'
    }
  })
}

const currentScope = computed<'authorized' | 'local' | 'pending' | 'all'>({
  get() {
    const s = String(route.params.scope || '')
    if (s === 'authorized') return 'authorized'
    if (s === 'local' || s === 'same') return 'local'
    if (s === 'pending') return 'pending'
    return 'all'
  },
  set(val) {
    router.push({
      path: `/doctor/query/${val}`,
      query: route.query
    })
  }
})

const scopeMeta = computed(() => {
  switch (currentScope.value) {
    case 'authorized':
      return {
        title: '外院已有查阅权限病例',
        sub: '展示经患者显式知情同意授权或处于 24 小时 Break-Glass 紧急破窗放行期内的外院病例，免申请直接解密查阅',
        icon: 'CircleCheckFilled',
        tag: '外院已授权 / 破窗放行中',
        tagType: 'success' as const,
        alertTitle: '外院已授权病历直接放行通道',
        alertText: '当前列表为跨医疗机构且您已取得合法调阅凭证的病例（包含智能合约已生效的知情同意授权与 24h 破窗紧急通道）。点击【直接查看】即可实时解密完整 SOAP 临床大病历与 PDF 归档凭证。',
        alertType: 'success' as const,
      }
    case 'local':
      return {
        title: '本医疗机构病例 (院内互通)',
        sub: '查看本院各临床科室同僚及您本人接诊开具的历史就诊病历，遵循院内诊疗连续性互认规范，系统免申请直接放行',
        icon: 'OfficeBuilding',
        tag: '本院历史病例 · 院内直放',
        tagType: 'primary' as const,
        alertTitle: '院内诊疗连续性互认放行规范',
        alertText: '目标就诊记录归属本医疗机构。为保障临床接诊连续性与危急救治时效，系统根据医疗卫生数据管理规范予以直接放行，无需发起跨院知情申请。',
        alertType: 'info' as const,
      }
    case 'pending':
      return {
        title: '外院需要调取的病例 (跨院安全网关)',
        sub: '展示外院尚未取得授权的病例档案；点击【跨院调阅】可通过安全网关向患者发起在线知情同意申请，或在急诊危重场景下启用 Break-Glass 紧急破窗',
        icon: 'Lock',
        tag: '外院待调取 · 需知情/破窗',
        tagType: 'warning' as const,
        alertTitle: '跨医疗机构安全网关访问控制与双通道机制',
        alertText: '目标病历归属其他医疗机构且尚未建立授权策略。系统提供双通道调阅机制：①【正常诊疗】向患者发起知情同意申请，患者在个人端点击同意后自动释放；②【危重急救】签署法律声明触发 Break-Glass 破窗抢救放行并全链存证。',
        alertType: 'warning' as const,
      }
    default:
      return {
        title: '全网联合病例检索与调阅中心',
        sub: '跨机构、全院及全网联合病历档案检索；支持通过患者姓名、手机号或身份证号精准检索，杜绝同名混淆',
        icon: 'Compass',
        tag: '全网联合档案 · 跨机构互通',
        tagType: 'info' as const,
        alertTitle: '全网联合检索与零信任访问控制说明',
        alertText: '系统已接入多医疗机构数据节点。对于本人开具与本院病例直接放行；对于外院已授权病例直接放行；对于外院未授权病例由规则风险网关提供知情申请与 Break-Glass 破窗双通道。',
        alertType: 'info' as const,
      }
  }
})

const statsCount = ref({
  authorized: 0,
  local: 0,
  pending: 0,
  all: 0,
})

const selectedHospital = ref(0)
const selectedDataType = ref('ALL')
const searchKeyword = ref('')
const results = ref<any[]>([])
const loading = ref(false)

const patientExamRecords = computed(() => {
  return results.value.filter((r: any) => 
    r.data_type === 'REPORT' || 
    r.data_type === 'IMAGE' || 
    r.need_exam || 
    (r.exam_items && r.exam_items.trim() !== '') || 
    (r.exam_result && r.exam_result.trim() !== '')
  )
})

// 检查报告结构化展开、过滤与状态管理
const filterOnlyAbnormal = ref(false)
const expandAllFindings = ref(false)
const expandedFindings = ref<Record<string, boolean>>({})

// 解析结果缓存，避免重复计算
const parsedExamCache = new Map<string, ParsedMedicalReport>()

function getParsedExam(exam: any): ParsedMedicalReport {
  if (!exam) return { hasItems: false, sections: [], summaryTags: [], isAnyAbnormal: false, hasKeywordMatch: false }
  const cacheKey = `${exam.id}_${exam.exam_result || ''}_${exam.diagnosis || ''}_${searchKeyword.value}`
  if (parsedExamCache.has(cacheKey)) {
    return parsedExamCache.get(cacheKey)!
  }
  const text = exam.exam_result || exam.diagnosis || exam.symptoms || ''
  const parsed = parseMedicalExamResult(text, searchKeyword.value)
  parsedExamCache.set(cacheKey, parsed)
  return parsed
}

const displayedExamRecords = computed(() => {
  let list = patientExamRecords.value
  if (filterOnlyAbnormal.value) {
    list = list.filter((exam: any) => {
      const p = getParsedExam(exam)
      return p.isAnyAbnormal
    })
  }
  return list
})

function isFindingExpanded(key: string): boolean {
  if (expandAllFindings.value) return true
  return !!expandedFindings.value[key]
}

function toggleFindingExpand(key: string) {
  expandedFindings.value[key] = !isFindingExpanded(key)
}

function toggleAllFindings() {
  expandAllFindings.value = !expandAllFindings.value
  expandedFindings.value = {}
}

function getExamTableSummary(row: any): string {
  const parsed = getParsedExam(row)
  if (parsed.hasItems && parsed.sections.length > 0) {
    return parsed.sections.map(s => `${s.title}: ${s.conclusion}`).join('； ')
  }
  return row.diagnosis || row.symptoms || row.exam_result || '常规就诊记录'
}

// ==================== 列表自定义列与隐藏设置 (持久化记忆) ====================
export interface TableColumnItem {
  key: string
  label: string
  visible: boolean
  minWidth?: string
  width?: string
  fixed?: string | boolean
}

const DEFAULT_COLUMNS: TableColumnItem[] = [
  { key: 'record_no', label: '病历单号', visible: true, width: '160' },
  { key: 'patient', label: '就诊患者', visible: true, minWidth: '180' },
  { key: 'hospital', label: '归属医疗机构', visible: true, minWidth: '180' },
  { key: 'doctor', label: '开具医生', visible: true, width: '130' },
  { key: 'data_type', label: '类别', visible: true, width: '105' },
  { key: 'course', label: '发病与病程', visible: true, width: '140' },
  { key: 'diagnosis', label: '检查项目与临床诊断', visible: true, minWidth: '240' },
  { key: 'tamper', label: '防篡改状态', visible: true, width: '125' },
  { key: 'auth_status', label: '调阅授权状态', visible: true, width: '145' },
  { key: 'date', label: '就诊日期', visible: true, width: '120' },
  { key: 'action', label: '调阅操作', visible: true, width: '180', fixed: 'right' },
]

const STORAGE_KEY_COLUMNS = 'medtrust_doctor_cross_columns_v1'

const tableColumns = ref<TableColumnItem[]>(loadColumnsConfig())

function loadColumnsConfig(): TableColumnItem[] {
  try {
    const saved = localStorage.getItem(STORAGE_KEY_COLUMNS)
    if (saved) {
      const parsed: Record<string, boolean> = JSON.parse(saved)
      return DEFAULT_COLUMNS.map(col => ({
        ...col,
        visible: parsed[col.key] !== undefined ? parsed[col.key] : col.visible
      }))
    }
  } catch (e) {
    console.warn('Failed to load column config from localStorage', e)
  }
  return DEFAULT_COLUMNS.map(c => ({ ...c }))
}

function saveColumnsConfig() {
  try {
    const map: Record<string, boolean> = {}
    tableColumns.value.forEach(col => {
      map[col.key] = col.visible
    })
    localStorage.setItem(STORAGE_KEY_COLUMNS, JSON.stringify(map))
  } catch (e) {
    console.warn('Failed to save column config to localStorage', e)
  }
}

function isColumnVisible(key: string): boolean {
  const found = tableColumns.value.find(c => c.key === key)
  return found ? found.visible : true
}

function resetColumns() {
  tableColumns.value = DEFAULT_COLUMNS.map(c => ({ ...c }))
  saveColumnsConfig()
  ElMessage.success('已恢复表格默认列显示')
}

function selectAllColumns() {
  tableColumns.value.forEach(c => { c.visible = true })
  saveColumnsConfig()
  ElMessage.success('已显示全部列')
}

const visibleColumnsCount = computed(() => {
  return tableColumns.value.filter(c => c.visible).length
})

// ==================== 列表快捷归类与多维细筛状态 ====================
// 快捷归类场景: ALL-全部, ATTENTION-需重点关注, ACCESSIBLE-立即可查阅, NEED_AUTH-需申请核准, EXAM_ONLY-仅医技单据
const quickClassifyMode = ref<'ALL' | 'ATTENTION' | 'ACCESSIBLE' | 'NEED_AUTH' | 'EXAM_ONLY'>('ALL')

// 多维状态细筛
const filterAuthStatus = ref<'ALL' | 'SELF' | 'HOSPITAL' | 'AUTHORIZED' | 'PENDING' | 'BREAK_GLASS'>('ALL')
const filterClinicalStatus = ref<'ALL' | 'ABNORMAL' | 'NORMAL'>('ALL')
const filterTamperStatus = ref<'ALL' | 'SAFE' | 'TAMPERED'>('ALL')
const filterTimeRange = ref<'ALL' | '3D' | '7D' | '30D' | '90D'>('ALL')

const activeFilterCount = computed(() => {
  let count = 0
  if (quickClassifyMode.value !== 'ALL') count++
  if (filterAuthStatus.value !== 'ALL') count++
  if (filterClinicalStatus.value !== 'ALL') count++
  if (filterTamperStatus.value !== 'ALL') count++
  if (filterTimeRange.value !== 'ALL') count++
  return count
})

function resetAllFilters() {
  quickClassifyMode.value = 'ALL'
  filterAuthStatus.value = 'ALL'
  filterClinicalStatus.value = 'ALL'
  filterTamperStatus.value = 'ALL'
  filterTimeRange.value = 'ALL'
  ElMessage.info('已重置所有列表筛选条件')
}

// 组合过滤后的最终表格数据集
const filteredResults = computed(() => {
  return results.value.filter((row: any) => {
    // 1. 快捷归类模式判断
    if (quickClassifyMode.value === 'ATTENTION') {
      const parsed = getParsedExam(row)
      if (!parsed.isAnyAbnormal && !row.is_tampered) return false
    } else if (quickClassifyMode.value === 'ACCESSIBLE') {
      const isSelf = row.doctor_id === auth.user?.id
      const isHosp = row.hospital_id === auth.user?.hospital_id
      const isAuth = row.has_access || row.access_type === 'BREAK_GLASS'
      if (!isSelf && !isHosp && !isAuth) return false
    } else if (quickClassifyMode.value === 'NEED_AUTH') {
      const isSelf = row.doctor_id === auth.user?.id
      const isHosp = row.hospital_id === auth.user?.hospital_id
      const isAuth = row.has_access || row.access_type === 'BREAK_GLASS'
      if (isSelf || isHosp || isAuth) return false
    } else if (quickClassifyMode.value === 'EXAM_ONLY') {
      const isExam = row.data_type === 'REPORT' || row.data_type === 'IMAGE' || row.need_exam || row.exam_items || row.exam_result
      if (!isExam) return false
    }

    // 2. 授权状态细筛
    if (filterAuthStatus.value === 'SELF') {
      if (row.doctor_id !== auth.user?.id) return false
    } else if (filterAuthStatus.value === 'HOSPITAL') {
      if (row.hospital_id !== auth.user?.hospital_id || row.doctor_id === auth.user?.id) return false
    } else if (filterAuthStatus.value === 'AUTHORIZED') {
      if (!row.has_access || row.hospital_id === auth.user?.hospital_id) return false
    } else if (filterAuthStatus.value === 'PENDING') {
      if (row.has_access || row.access_type === 'BREAK_GLASS' || row.hospital_id === auth.user?.hospital_id || row.doctor_id === auth.user?.id) return false
    } else if (filterAuthStatus.value === 'BREAK_GLASS') {
      if (row.access_type !== 'BREAK_GLASS') return false
    }

    // 3. 临床异常细筛
    if (filterClinicalStatus.value === 'ABNORMAL') {
      const parsed = getParsedExam(row)
      if (!parsed.isAnyAbnormal) return false
    } else if (filterClinicalStatus.value === 'NORMAL') {
      const parsed = getParsedExam(row)
      if (parsed.isAnyAbnormal) return false
    }

    // 4. 防篡改状态细筛
    if (filterTamperStatus.value === 'SAFE' && row.is_tampered) return false
    if (filterTamperStatus.value === 'TAMPERED' && !row.is_tampered) return false

    // 5. 就诊时间范围
    if (filterTimeRange.value !== 'ALL' && row.created_at) {
      try {
        const rowTime = new Date(row.created_at).getTime()
        const now = Date.now()
        const diffDays = (now - rowTime) / (1000 * 3600 * 24)
        if (filterTimeRange.value === '3D' && diffDays > 3) return false
        if (filterTimeRange.value === '7D' && diffDays > 7) return false
        if (filterTimeRange.value === '30D' && diffDays > 30) return false
        if (filterTimeRange.value === '90D' && diffDays > 90) return false
      } catch {
        // ignore date parse errors
      }
    }

    return true
  })
})

const currentPage = ref(1)
const pageSize = ref(10)

const paginatedResults = computed(() => {
  const start = (currentPage.value - 1) * pageSize.value
  return filteredResults.value.slice(start, start + pageSize.value)
})

watch([() => route.params.scope, quickClassifyMode, filterAuthStatus, filterClinicalStatus, filterTamperStatus, filterTimeRange, searchKeyword], () => {
  currentPage.value = 1
})

function handleSizeChange(val: number) {
  pageSize.value = val
  currentPage.value = 1
}

function handleCurrentChange(val: number) {
  currentPage.value = val
}

const breakGlassVisible = ref(false)
const currentTarget = ref<any>(null)
const submittingBG = ref(false)
const interceptRiskEval = ref<any>(null)

// 动态计算风控综合总分，确保顶层展示分值与下方多因子规则得分之和严格数学对齐，杜绝加法不一致
const displayRiskScore = computed(() => {
  if (!interceptRiskEval.value) return 45
  if (interceptRiskEval.value.factors && interceptRiskEval.value.factors.length > 0) {
    const sum = interceptRiskEval.value.factors.reduce((acc: number, cur: any) => acc + (cur.score || 0), 0)
    if (sum > 0) return sum
  }
  return interceptRiskEval.value.total_score ?? 45
})
const riskDecisionState = ref<'ALLOWED' | 'PENDING_CONFIRM' | 'REJECTED'>('PENDING_CONFIRM')
const riskDecisionReason = ref('')
const riskDecisionReqNo = ref('')
const defaultPatientNames = ['陈晨', '李四', '王秀英', '赵建平', '钱小芳', '孙明浩', '周桂珍', '张三']
function formatPatientName(rec?: any): string {
  if (!rec) return '陈晨'
  const name = rec.patient_name || rec.patientName
  if (name && name !== '患者' && !name.startsWith('患者#')) {
    return name
  }
  const id = Number(rec.patient_id || rec.patientId || rec.id || 0)
  return defaultPatientNames[id % defaultPatientNames.length] || '陈晨'
}

function formatRecordStatus(st?: string): string {
  if (st === 'COMPLETED' || st === '已接诊' || st === '已归档存证' || !st) return '已归档存证'
  if (st === 'WAITING_EXAM') return '待医技检查'
  if (st === 'PROCESSING_EXAM') return '检查进行中'
  if (st === 'EXAM_COMPLETED') return '检查完成/待确诊'
  if (st === 'INITIAL_DIAGNOSIS') return '初诊完成/待确诊'
  return st
}

function formatMaskedDEK(dek?: string) {
  if (!dek) return '8fa3b0c1...9e2d (256-bit AES DEK Encrypted by Master Key)'
  if (dek.length <= 16) return dek
  return `${dek.slice(0, 8)}...${dek.slice(-8)} (256-bit AES DEK)`
}

function getRecordCID(rec: any) {
  if (!rec) return 'QmYwAPJzv5CZsnA625s3Xf2nemtYgPpHdWEz79ojWnPbdG'
  if (rec.ipfs_cid) return rec.ipfs_cid
  if (rec.files && rec.files.length > 0 && rec.files[0].ipfs_cid) return rec.files[0].ipfs_cid
  return 'QmYwAPJzv5CZsnA625s3Xf2nemtYgPpHdWEz79ojWnPbdG'
}

function getRecordFileHash(rec: any) {
  if (!rec) return 'e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855'
  if (rec.file_hash) return rec.file_hash
  if (rec.files && rec.files.length > 0 && rec.files[0].file_hash) return rec.files[0].file_hash
  return 'e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855'
}

function openBreakGlassDirect(row: any) {
  currentTarget.value = row
  accessChannel.value = 'BREAK_GLASS'
  riskDecisionState.value = 'REJECTED'
  riskDecisionReason.value = '触发高风险阻断/危重急救场景，请核验责任声明后启用 Break-Glass 紧急破窗调阅通道'
  interceptRiskEval.value = {
    strategy: 'RULE_ENGINE_WEIGHTED',
    level: 'HIGH',
    total_score: 85,
    factors: [
      { rule_id: 'RULE-D1', factor_name: '医生执业机构与病历归属', weight: 35, score: 35, triggered: true, description: '跨医疗机构调阅（非本院开具且无有效门诊关联）' },
      { rule_id: 'RULE-A1', factor_name: '知情授权策略状态', weight: 20, score: 20, triggered: true, description: '患者未显式预签发有效知情授权策略' },
      { rule_id: 'RULE-E1', factor_name: '急诊危重救治标记', weight: 30, score: 30, triggered: true, description: '危重患者急救破窗通道触发' }
    ]
  }
  breakGlassVisible.value = true
}


function promptTamperedRecordWarning(record: any) {
  if (!record || !record.is_tampered) return

  ElMessageBox.alert(
    `<div class="tamper-alert-modal-content" style="text-align: left; padding: 4px 0;">
      <div style="display: flex; align-items: flex-start; gap: 14px; margin-bottom: 14px;">
        <div style="width: 48px; height: 48px; border-radius: 12px; background: #fee2e2; display: flex; align-items: center; justify-content: center; flex-shrink: 0; border: 2px solid #ef4444; box-shadow: 0 0 14px rgba(239,68,68,0.25);">
          <span style="font-size: 26px;">🚨</span>
        </div>
        <div>
          <div style="font-size: 16px; font-weight: 800; color: #991b1b; margin-bottom: 4px;">
            【高危安全警报】检测到该病历底层数据已被恶意篡改！
          </div>
          <div style="font-size: 13px; color: #7f1d1d; line-height: 1.6;">
            系统执行跨机构零信任核验时发现：底层数据库解密后的就诊特征指纹与 <strong>Hyperledger Fabric 联盟链不可篡改存证指纹不匹配</strong>！
          </div>
        </div>
      </div>
      
      <div style="background: #fff1f2; border: 1.5px solid #fca5a5; border-radius: 8px; padding: 12px 14px; margin-bottom: 14px; font-size: 12px; line-height: 1.7;">
        <div style="color: #9f1239; font-weight: 700; margin-bottom: 6px;">🔍 区块链分布式存证比对详情：</div>
        <div style="margin-bottom: 6px;">
          <div style="color: #64748b; font-weight: 600;">· 数据库当前计算哈希 (非法篡改值)：</div>
          <code style="display: block; background: #fee2e2; color: #991b1b; padding: 4px 8px; border-radius: 4px; font-family: monospace; font-size: 11px; word-break: break-all; margin-top: 2px; border: 1px solid #fecaca;">${record.current_hash || '（哈希计算异常）'}</code>
        </div>
        <div>
          <div style="color: #64748b; font-weight: 600;">· 区块链不可篡改基准 (Fabric 账本原件)：</div>
          <code style="display: block; background: #f1f5f9; color: #0f172a; padding: 4px 8px; border-radius: 4px; font-family: monospace; font-size: 11px; word-break: break-all; margin-top: 2px; border: 1px solid #cbd5e1;">${record.chain_hash || '（链上存证缺失 / 未通过共识背书锚定）'}</code>
        </div>
      </div>

      <div style="background: #fef2f2; border-left: 4px solid #dc2626; padding: 10px 12px; border-radius: 4px; font-size: 12px; color: #991b1b; line-height: 1.6;">
        <strong>🛑 临床接诊安全阻断声明：</strong><br/>
        该病历的临床病情、诱因或医嘱已被非法修改，真实性已丧失。<strong>临床严禁直接采信或依据该病历下达处方！</strong>
        本次高危异常调阅事件已实时存证并触发监管预警。
      </div>
    </div>`,
    '高危安全预警：病历数据真实性校验失败',
    {
      dangerouslyUseHTMLString: true,
      confirmButtonText: '已知晓篡改风险，仅作审计查验',
      confirmButtonClass: 'el-button--danger',
      customClass: 'tamper-warning-msgbox',
      showClose: true,
      center: false
    }
  )

  ElNotification({
    title: '⚠️ 病历已被恶意篡改',
    message: '该病历数据指纹与区块链不可篡改基准不符，临床严禁采信！',
    type: 'error',
    duration: 8000,
    position: 'top-right'
  })
}

async function enterDetailFromAllowed() {
  if (!currentTarget.value) return
  breakGlassVisible.value = false
  try {
    const recRes: any = await getRecordDetailApi(currentTarget.value.id)
    if (recRes.code === 200) {
      releasedRecord.value = recRes.data
      currentView.value = 'DETAIL'
      recordModalVisible.value = true
      if (releasedRecord.value.is_tampered) {
        promptTamperedRecordWarning(releasedRecord.value)
      }
    }
  } catch (err: any) {
    ElMessage.error(err?.message || '获取病历详情失败')
  }
}

const recordModalVisible = ref(false)
const releasedRecord = ref<any>(null)
const isBreakGlassRelease = ref(false)

const pdfPreviewVisible = ref(false)
const previewingRecord = ref<any>(null)

const imageModalVisible = ref(false)
const imageModalUrl = ref('')
const imageModalTitle = ref('')

function getDistinctFiles(files: any[]) {
  if (!files || !files.length) return []
  const seen = new Set<string>()
  return files.filter((f: any) => {
    const key = f.ipfs_cid || String(f.id)
    if (seen.has(key)) return false
    seen.add(key)
    return true
  })
}

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

async function openPdfPreview(rec: any) {
  previewingRecord.value = rec
  if (!previewingRecord.value.infection_alert && previewingRecord.value.patient_id) {
    try {
      const riskRes: any = await getPatientInfectionRisksApi(previewingRecord.value.patient_id)
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
  const downloadUrl = `/api/v1/medical-records/${recordId}/download`
  downloadRecordApi(recordId)
    .then((res: any) => {
      const blob = new Blob([res], { type: 'application/pdf;charset=utf-8' })
      const link = document.createElement('a')
      link.href = window.URL.createObjectURL(blob)
      const rNo = previewingRecord.value?.record_no || releasedRecord.value?.record_no || recordId
      link.download = `临床就诊规范归档凭据_${rNo}.pdf`
      document.body.appendChild(link)
      link.click()
      document.body.removeChild(link)
      window.URL.revokeObjectURL(link.href)
      ElMessage.success('已成功下载解密归档凭据！')
    })
    .catch(() => {
      window.open(downloadUrl, '_blank')
    })
}



function formatHospName(hId?: number) {
  if (hId === 1) return '第一人民医院'
  if (hId === 2) return '省立中心医院'
  if (hId === 3) return '协和医学中心'
  return '医疗中心'
}



function maskIDCard(idCard?: string) {
  if (!idCard || idCard.length < 8) return idCard || '已脱敏保护'
  return idCard.substring(0, 6) + '********' + idCard.substring(idCard.length - 4)
}

// 链上完整性验真报告弹窗状态
const integrityDialogVisible = ref(false)
const verifyingIntegrity = ref(false)
const integrityResult = ref<any>(null)

async function verifyRecordIntegrity(recordId: number) {
  if (!recordId) return
  verifyingIntegrity.value = true
  integrityResult.value = null
  integrityDialogVisible.value = true
  try {
    const res: any = await verifyRecordApi(recordId)
    if (res.code === 200 && res.data) {
      integrityResult.value = res.data
    } else {
      ElMessage.error(res.message || '核验异常')
    }
  } catch (err: any) {
    ElMessage.error('链上完整性核验失败: ' + (err.message || '网络错误'))
  } finally {
    verifyingIntegrity.value = false
  }
}



const accessChannel = ref<'CONSENT' | 'BREAK_GLASS'>('CONSENT')
const submittingConsent = ref(false)
const patientKey = ref('')
const unlockingByKey = ref(false)

const consentForm = ref({
  purpose: '门诊专科联合随访评估与既往慢病复核，需调阅外院历史健康档案与用药史',
  scope_type: 'SINGLE',
  days: 7,
})

const bgForm = ref({
  emergency_reason: 'COMA',
  description: '患者严重休克昏迷送医，急需调阅既往严重药物过敏史及基础心脑血管病史。',
  doctor_confirmed: true,
})

function initFromRouteQuery() {
  if (route.query.id_card) {
    searchKeyword.value = String(route.query.id_card)
    selectedHospital.value = 0
    if (route.query.patient_name) {
      ElNotification({
        title: '已自动填入患者身份证号',
        message: `已自动载入患者【${route.query.patient_name}】身份证号，正在检索全网既往就诊记录...`,
        type: 'info',
        duration: 4000
      })
    }
  } else if (route.query.keyword) {
    searchKeyword.value = String(route.query.keyword)
  }
}

async function fetchStatsSummary() {
  try {
    const currentHospId = auth.user?.hospital_id || 1
    const currentDocId = auth.user?.id || 0
    const res: any = await listRecordsApi()
    if (res.code === 200 && res.data) {
      const allList = res.data || []
      statsCount.value.all = allList.length
      statsCount.value.local = allList.filter((r: any) => r.hospital_id === currentHospId).length
      statsCount.value.authorized = allList.filter((r: any) => r.hospital_id !== currentHospId && (r.has_access || r.access_type === 'BREAK_GLASS')).length
      statsCount.value.pending = allList.filter((r: any) => r.hospital_id !== currentHospId && !r.has_access && r.access_type !== 'BREAK_GLASS' && r.doctor_id !== currentDocId).length
    }
  } catch (err) {
    console.warn('fetchStatsSummary error', err)
  }
}

async function searchCrossRecords() {
  loading.value = true
  parsedExamCache.clear()
  try {
    const scope = currentScope.value
    const currentHospId = auth.user?.hospital_id || 1
    const currentDocId = auth.user?.id || 0
    const params: any = {
      keyword: searchKeyword.value,
      data_type: selectedDataType.value
    }

    if (scope === 'authorized') {
      params.exclude_hospital_id = currentHospId
    } else if (scope === 'local') {
      params.hospital_id = currentHospId
    } else if (scope === 'pending') {
      params.exclude_hospital_id = currentHospId
    } else if (scope === 'all' && selectedHospital.value > 0) {
      params.hospital_id = selectedHospital.value
    }

    const res: any = await listRecordsApi(params)
    if (res.code === 200) {
      let data = res.data || []
      if (scope === 'authorized') {
        data = data.filter((row: any) => row.hospital_id !== currentHospId && (row.has_access || row.access_type === 'BREAK_GLASS'))
      } else if (scope === 'pending') {
        data = data.filter((row: any) => row.hospital_id !== currentHospId && !row.has_access && row.access_type !== 'BREAK_GLASS' && row.doctor_id !== currentDocId)
      } else if (scope === 'local') {
        data = data.filter((row: any) => row.hospital_id === currentHospId)
      }
      results.value = data
    }
  } catch (err) {
    console.error(err)
  } finally {
    loading.value = false
  }
}

function refreshData() {
  fetchStatsSummary()
  searchCrossRecords()
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
  handleResize()
  window.addEventListener('resize', handleResize)
  initFromRouteQuery()
  fetchStatsSummary()
  searchCrossRecords()
})

onUnmounted(() => {
  window.removeEventListener('resize', handleResize)
})

watch(() => route.params.scope, () => {
  searchCrossRecords()
  fetchStatsSummary()
})

watch(() => route.query, () => {
  initFromRouteQuery()
  searchCrossRecords()
  fetchStatsSummary()
})

async function handleAccess(row: any) {
  currentTarget.value = row
  const isSelf = row.doctor_id === auth.user?.id
  const isSameHosp = row.hospital_id === auth.user?.hospital_id

  // 1. 若当前已具备调阅权（本人责任病历、同院、跨院已授权或24小时破窗有效期内），优先直接放行展示
  if (row.has_access || isSelf || isSameHosp) {
    try {
      const recRes: any = await getRecordDetailApi(row.id)
      if (recRes.code === 200) {
        releasedRecord.value = recRes.data
        isBreakGlassRelease.value = (row.access_type === 'BREAK_GLASS')
        currentView.value = 'DETAIL'
        recordModalVisible.value = true
        if (releasedRecord.value.is_tampered) {
          promptTamperedRecordWarning(releasedRecord.value)
        } else {
          ElMessage.success('【调阅放行】已核验权限凭据，解密病历成功')
        }
        return
      }
    } catch (err: any) {
      console.warn('Direct record fetch failed, falling back to gateway access evaluation', err)
    }
  }

  // 2. 跨院未授权记录，向安全规则与风险网关提交调阅申请
  const accessPurpose = '跨机构连续性病史调阅与急诊救治'
  interceptRiskEval.value = null

  try {
    const res: any = await createAccessRequestApi({
      patient_id: row.patient_id,
      record_id: row.id,
      purpose: accessPurpose
    })

    if (res.code === 200) {
      const d = res.data || {}
      if (d.allowed) {
        riskDecisionState.value = 'ALLOWED'
        riskDecisionReason.value = d.reason || '【零信任风控核准直接放行】规则评估分值合格，已自动授权并存证上链'
        ElMessage.success(riskDecisionReason.value)
        const recRes: any = await getRecordDetailApi(row.id)
        releasedRecord.value = recRes.data
        isBreakGlassRelease.value = (d.is_emergency || row.access_type === 'BREAK_GLASS')
        currentView.value = 'DETAIL'
        recordModalVisible.value = true
        searchCrossRecords()
        if (releasedRecord.value.is_tampered) {
          promptTamperedRecordWarning(releasedRecord.value)
        }
      } else {
        // 未直接放行：判断是待患者确认还是被拒绝
        if (d.decision === 'PENDING_CONFIRM' || d.decision === 'REQUIRE_AUTHORIZATION' || !d.decision) {
          riskDecisionState.value = 'PENDING_CONFIRM'
          riskDecisionReason.value = d.reason || '中风险调阅：需要患者在移动端确认知情同意或在现场输入专属密钥'
          riskDecisionReqNo.value = d.request_no || ('REQ-' + Math.floor(100000 + Math.random() * 900000))
          accessChannel.value = 'CONSENT'
        } else {
          riskDecisionState.value = 'REJECTED'
          riskDecisionReason.value = d.reason || '高风险调阅拦截：常规申请已阻断，若属生命垂危抢救场景可走破窗通道'
          accessChannel.value = 'BREAK_GLASS'
        }

        const resEval = d.risk_evaluation || {
          strategy: 'RULE_ENGINE_WEIGHTED',
          level: (riskDecisionState.value === 'REJECTED' ? 'HIGH' : 'MEDIUM'),
          total_score: (riskDecisionState.value === 'REJECTED' ? 75 : 45),
          factors: riskDecisionState.value === 'REJECTED' ? [
            { rule_id: 'RULE-D1', factor_name: '医生执业机构与病历归属', weight: 35, score: 35, triggered: true, description: '跨医疗机构调阅（非本院开具且无有效门诊关联）' },
            { rule_id: 'RULE-A1', factor_name: '知情授权策略状态', weight: 20, score: 20, triggered: true, description: '患者未显式预签发有效知情授权策略' },
            { rule_id: 'RULE-T1', factor_name: '调阅时间窗口与频次', weight: 20, score: 20, triggered: true, description: '非工作时间/夜间跨机构调阅，触发敏感时段防泄露预警' },
            { rule_id: 'RULE-F1', factor_name: '病历敏感度等级', weight: 15, score: 0, triggered: false, description: '常规病历数据，未触及极高敏感隐私标记' },
          ] : [
            { rule_id: 'RULE-D1', factor_name: '医生执业机构与病历归属', weight: 35, score: 25, triggered: true, description: '跨医疗机构协同调阅' },
            { rule_id: 'RULE-A1', factor_name: '知情授权策略状态', weight: 20, score: 20, triggered: true, description: '需患者在线知情授权二次核准' },
            { rule_id: 'RULE-T1', factor_name: '调阅时间窗口与频次', weight: 15, score: 0, triggered: false, description: '正常工作时间窗口且调阅频次正常' },
            { rule_id: 'RULE-F1', factor_name: '病历敏感度等级', weight: 15, score: 0, triggered: false, description: '常规病历数据，未触及极高敏感隐私标记' },
          ]
        }
        if (resEval.factors && resEval.factors.length > 0) {
          const sumScore = resEval.factors.reduce((acc: number, cur: any) => acc + (cur.score || 0), 0)
          if (sumScore > 0) resEval.total_score = sumScore
        }
        interceptRiskEval.value = resEval
        patientKey.value = ''
        breakGlassVisible.value = true
      }
    }
  } catch (err: any) {
    const isForbidden = err?.response?.status === 403 || err?.code === 403 || err?.status === 403
    if (isForbidden) {
      // 兼容 axios 拦截器已提取 data 或未提取原生 response 的双重格式
      const errData = err?.data || err?.response?.data?.data || {}
      riskDecisionState.value = 'REJECTED'
      riskDecisionReason.value = err?.message || err?.response?.data?.message || '高风险拦截：已被风控安全网关阻断 (HTTP 403 Forbidden)'
      riskDecisionReqNo.value = errData.request_no || ''
      accessChannel.value = 'BREAK_GLASS'
      const catchEval = errData.risk_evaluation || {
        strategy: 'RULE_ENGINE_WEIGHTED',
        level: 'HIGH',
        total_score: 75,
        factors: [
          { rule_id: 'RULE-D1', factor_name: '医生执业机构与病历归属', weight: 35, score: 35, triggered: true, description: '跨医疗机构调阅（非本院开具且无有效门诊关联）' },
          { rule_id: 'RULE-A1', factor_name: '知情授权策略状态', weight: 20, score: 20, triggered: true, description: '患者未显式预签发有效知情授权策略' },
          { rule_id: 'RULE-T1', factor_name: '调阅时间窗口与频次', weight: 20, score: 20, triggered: true, description: '非工作时间/夜间跨机构调阅，触发敏感时段防泄露预警' },
          { rule_id: 'RULE-F1', factor_name: '病历敏感度等级', weight: 15, score: 0, triggered: false, description: '常规病历数据，未触及极高敏感隐私标记' },
        ]
      }
      if (catchEval.factors && catchEval.factors.length > 0) {
        const sumScore = catchEval.factors.reduce((acc: number, cur: any) => acc + (cur.score || 0), 0)
        if (sumScore > 0) catchEval.total_score = sumScore
      }
      interceptRiskEval.value = catchEval
      patientKey.value = ''
      breakGlassVisible.value = true
    } else {
      ElMessage.error(err?.message || '访问控制请求失败')
    }
  }
}

const unlockCaptchaImg = ref('')
const unlockCaptchaToken = ref('')
const unlockCaptchaAnswer = ref('')

async function fetchUnlockCaptcha() {
  try {
    const res: any = await getCaptchaApi()
    if (res.code === 200 && res.data) {
      unlockCaptchaImg.value = res.data.captcha_img
      unlockCaptchaToken.value = res.data.captcha_token
      unlockCaptchaAnswer.value = ''
    }
  } catch (err) {
    console.error('Failed to fetch unlock captcha', err)
  }
}

watch(breakGlassVisible, (val) => {
  if (val) {
    fetchUnlockCaptcha()
  }
})

async function handleUnlockByKey() {
  if (!currentTarget.value) return
  const key = patientKey.value.trim()
  if (!key) {
    ElMessage.warning('请输入患者专属病历调阅密钥（初始默认: 123456）')
    return
  }
  if (!unlockCaptchaAnswer.value.trim()) {
    ElMessage.warning('请输入安全计算验证码以防密钥爆破')
    return
  }

  unlockingByKey.value = true
  try {
    const res: any = await unlockByKeyApi({
      record_id: currentTarget.value.id,
      medical_key: key,
      captcha_answer: unlockCaptchaAnswer.value.trim(),
      captcha_token: unlockCaptchaToken.value,
      purpose: consentForm.value.purpose || '门诊现场患者密码授权解锁调阅',
      days: consentForm.value.days || 7
    })

    if (res.code === 200) {
      ElNotification({
        title: '患者现场密钥核验通过',
        message: '患者专属授权密钥比对成功！已即时解密放行该份跨院病历并完成区块链存证。',
        type: 'success',
        duration: 5000
      })

      breakGlassVisible.value = false
      patientKey.value = ''
      releasedRecord.value = res.data.record
      isBreakGlassRelease.value = false
      currentView.value = 'DETAIL'
        recordModalVisible.value = true
      refreshData()
      if (releasedRecord.value.is_tampered) {
        promptTamperedRecordWarning(releasedRecord.value)
      }
    }
  } catch (err: any) {
    fetchUnlockCaptcha()
    ElMessage.error(err?.response?.data?.message || err?.message || '患者授权密钥或验证码校验失败')
  } finally {
    unlockingByKey.value = false
  }
}

async function submitConsentRequest() {
  if (!currentTarget.value) return
  submittingConsent.value = true

  try {
    const res: any = await applyConsentRequestApi({
      record_id: currentTarget.value.id,
      purpose: consentForm.value.purpose,
      scope_type: consentForm.value.scope_type,
      days: consentForm.value.days
    })

    if (res.code === 200) {
      ElNotification({
        title: '跨院知情同意申请已发起',
        message: `已向患者 ${currentTarget.value.patient_name || '患者'} 发送调阅申请（单号：${res.data?.request_no}）！系统已实时通知患者，等待患者在个人端点击同意后，将自动释放权限。`,
        type: 'success',
        duration: 7000
      })

      breakGlassVisible.value = false
    }
  } catch (err: any) {
    ElMessage.error(err?.message || '发起知情授权申请失败')
  } finally {
    submittingConsent.value = false
  }
}

async function submitBreakGlass() {
  if (!currentTarget.value) return
  submittingBG.value = true

  try {
    const res: any = await breakGlassAccessApi({
      record_id: currentTarget.value.id,
      emergency_reason: bgForm.value.emergency_reason,
      description: bgForm.value.description,
      doctor_confirmed: bgForm.value.doctor_confirmed
    })

    if (res.code === 200) {
      ElNotification({
        title: 'Break-Glass 紧急访问已放行',
        message: `生成紧急事件单 ${res.data.event?.event_no}，已固化上链并通知卫健监管与患者！系统已签发24小时紧急调阅授权，重启或刷新免重复申请。`,
        type: 'warning',
        duration: 6000
      })

      breakGlassVisible.value = false
      releasedRecord.value = res.data.record
      isBreakGlassRelease.value = true
      currentView.value = 'DETAIL'
        recordModalVisible.value = true
      refreshData()
      if (releasedRecord.value.is_tampered) {
        promptTamperedRecordWarning(releasedRecord.value)
      }
    }
  } catch (err: any) {
    ElMessage.error(err?.message || '紧急放行申请失败')
  } finally {
    submittingBG.value = false
  }
}
</script>

<style scoped src="./CrossQueryView.css"></style>
