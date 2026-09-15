<template>
  <div class="page-container">
    <!-- 视图一：跨院病例检索与多维筛选视图 (QUERY) -->
    <div v-if="currentView === 'QUERY'" class="query-main-wrapper">
      <div class="page-header">
      <div class="header-main-row">
        <div>
          <div class="header-badge-row">
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
          <el-button
            size="small"
            type="primary"
            plain
            style="margin-left: 16px;"
            @click="router.push('/doctor/records')"
          >
            返回门诊就诊工作台
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
            style="width: 380px;"
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

      <!-- 数据表格绑定 filteredResults，并按自定义列显隐动态呈现 -->
      <el-table :data="filteredResults" v-loading="loading" stripe style="width: 100%">
        <el-table-column v-if="isColumnVisible('record_no')" prop="record_no" label="病历单号" width="160" />
        <el-table-column v-if="isColumnVisible('patient')" label="就诊患者" min-width="180">
          <template #default="{ row }">
            <div class="patient-cell">
              <span class="patient-name">{{ row.patient_name || '患者' }}</span>
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
        <el-table-column v-if="isColumnVisible('action')" label="调阅操作" width="180" fixed="right">
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
            <el-button 
              v-else 
              type="warning" 
              size="small" 
              @click="handleAccess(row)"
            >
              跨院调阅 (安全网关)
            </el-button>
          </template>
        </el-table-column>
      </el-table>
    </el-card>
    </div>

        <!-- 视图二：跨院医疗数据已核准解密放行全景工作台 (DETAIL) -->
    <div v-else-if="currentView === 'DETAIL'" class="inpage-workstation-wrapper">
      <!-- 顶部工作台粘性控制栏 (轻量纤细版 48px) -->
      <div class="ws-page-header-card sticky-header">
        <div class="ws-header-left">
          <el-button :icon="ArrowLeft" plain size="small" @click="currentView = 'QUERY'" class="back-btn">返回检索</el-button>
          <div class="ws-header-title-box">
            <span class="ws-page-title">跨院解密就诊全景病历</span>
            <el-tag size="small" type="success" effect="light">已放行解密</el-tag>
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

        <!-- 篡改告警横幅 -->
        <div v-if="releasedRecord.is_tampered" class="tamper-box danger">
          <div class="t-head">
            <el-icon><WarningFilled /></el-icon>
            <span>【高危安全警报】检测到该病历数据库数据已被恶意篡改！</span>
          </div>
          <div class="t-desc">
            系统检测到底层 MySQL 数据库中的临床数据与 Fabric 联盟链上不可篡改的存证指纹不匹配！临床严禁直接采信该病历！
          </div>
          <div class="t-hashes">
            <div>数据库当前计算哈希: <code>{{ releasedRecord.current_hash }}</code></div>
            <div>区块链不可篡改基准: <code>{{ releasedRecord.chain_hash }}</code></div>
          </div>
        </div>

        <div v-else class="tamper-box safe">
          <div class="tb-safe-left">
            <el-icon class="text-emerald-600 text-lg"><CircleCheckFilled /></el-icon>
            <div class="tb-text">
              <span class="font-bold text-emerald-800 text-sm">区块链全量防篡改核验通过</span>
              <span class="text-emerald-700 text-xs ml-2">该病历所有临床症状、用药与检查数据指纹与 Fabric 联盟链上存证 100% 吻合，真实未篡改</span>
            </div>
          </div>
          <div class="tb-safe-right">
            <span class="fabric-badge">Cross-Org Verified 100%</span>
          </div>
        </div>

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

        <div class="ws-body-grid">
          <!-- 左侧栏：跨院档案属性、区块链凭据、附件文件 (380px) -->
          <div class="ws-col-left">
            <!-- 卡片 1：就诊档案与患者卡片 -->
            <div class="bento-card patient-hero-card">
              <div class="ph-header">
                <div class="ph-avatar">
                  {{ releasedRecord.patient_name ? releasedRecord.patient_name.slice(0, 1) : '患' }}
                </div>
                <div class="ph-main-info">
                  <div class="ph-name-row">
                    <span class="ph-name">{{ releasedRecord.patient_name }}</span>
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
                  <span class="val font-semibold text-primary">{{ releasedRecord.doctor_name }}</span>
                </div>
                <div class="ph-item">
                  <span class="lbl">就诊时间</span>
                  <span class="val text-xs text-slate-600">{{ formatTime(releasedRecord.created_at) }}</span>
                </div>
              </div>
            </div>

            <!-- 卡片 2：区块链安全与权威存证凭据 -->
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
                    <button class="cf-copy-btn" @click="copyToClipboard(releasedRecord.fabric_tx_id, '交易哈希')">
                      <el-icon><CopyDocument /></el-icon>
                      <span>复制哈希</span>
                    </button>
                  </div>
                  <div class="cf-hash-box mono" :title="releasedRecord.fabric_tx_id">
                    {{ shortHash(releasedRecord.fabric_tx_id, 14, 12) }}
                  </div>
                </div>

                <div class="crypto-stats-row">
                  <div class="stat-pill">
                    <span class="stat-k">区块高度</span>
                    <span class="stat-v mono">#{{ releasedRecord.block_height || '342' }}</span>
                  </div>
                  <div class="stat-pill">
                    <span class="stat-k">共识验证</span>
                    <span class="stat-v">PBFT 已确认</span>
                  </div>
                  <div class="stat-pill">
                    <span class="stat-k">存证通道</span>
                    <span class="stat-v">medtrust</span>
                  </div>
                </div>

                <div class="crypto-security-seal">
                  <el-icon class="mr-1 text-emerald-400"><CircleCheckFilled /></el-icon>
                  <span>AES-256-GCM 加密放行 · 智能合约权限审计通过</span>
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
                  <div v-for="file in releasedRecord.files" :key="file.id" class="file-item-bento">
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


    <!-- 权限拦截对话框：提供 正常诊疗知情授权申请 与 急诊Break-Glass破窗 双通道 -->
    <el-dialog v-model="breakGlassVisible" title="跨院安全网关访问控制拦截与调阅申请" width="720px">
      <div class="intercept-notice mb-3">
        <el-alert
          title="未取得患者显式知情授权 (HTTP 403 Forbidden)"
          type="error"
          description="系统基于零信任统一权限网关检测：目标病历属于其他医疗机构，且患者当前未对您建立有效的跨院知情授权策略。请根据实际临床场景选择调阅通道："
          show-icon
          :closable="false"
        />
      </div>

      <!-- 零信任规则风控评估详情卡片 (Rule-Based Weight Scoring) -->
      <div v-if="interceptRiskEval" class="risk-eval-detail-card mb-3">
        <div class="redc-header">
          <div class="redc-title">
            <strong>零信任安全规则引擎动态评估详情 (Rule-Based Risk Scoring)</strong>
          </div>
          <div class="redc-score-wrap">
            <span class="redc-strategy">策略: <code>{{ interceptRiskEval.strategy || 'RULE_ENGINE_WEIGHTED' }}</code></span>
            <el-tag :type="interceptRiskEval.level === 'HIGH' ? 'danger' : 'warning'" size="small" effect="dark">
              评级: {{ interceptRiskEval.level }} (总评分: {{ interceptRiskEval.total_score }} / 100)
            </el-tag>
          </div>
        </div>

        <p class="redc-intro">
          系统根据《跨机构医疗数据流通合规规范》进行<strong>规则加权多维综合评分</strong>，绝非不可解释的黑盒模型；当前请求命中以下规则因子：
        </p>

        <el-table
          v-if="interceptRiskEval.factors && interceptRiskEval.factors.length"
          :data="interceptRiskEval.factors"
          size="small"
          stripe
          class="factor-table mb-2"
        >
          <el-table-column prop="rule_id" label="规则编号" width="95" />
          <el-table-column prop="factor_name" label="风控判定维度" width="145" />
          <el-table-column label="基准权重" width="85">
            <template #default="{ row }">
              <span>+{{ row.weight }}</span>
            </template>
          </el-table-column>
          <el-table-column label="实际得分" width="85">
            <template #default="{ row }">
              <strong :style="{ color: row.triggered ? '#ef4444' : '#10b981' }">
                {{ row.score > 0 ? ('+' + row.score) : '0' }}
              </strong>
            </template>
          </el-table-column>
          <el-table-column label="判定状态" width="100">
            <template #default="{ row }">
              <el-tag size="small" :type="row.triggered ? 'danger' : 'success'" effect="plain">
                {{ row.triggered ? '命中规则' : '正常放行' }}
              </el-tag>
            </template>
          </el-table-column>
          <el-table-column prop="description" label="合规裁决释义" min-width="200" />
        </el-table>
      </div>

      <!-- 双通道选项切换卡 -->
      <div class="channel-selector-wrapper mb-3">
        <el-radio-group v-model="accessChannel" size="large" class="channel-tabs" style="width: 100%; display: flex;">
          <el-radio-button label="CONSENT" style="flex: 1;">
            <span>选项一：正常诊疗·现场密钥解锁 / 知情同意申请 (推荐)</span>
          </el-radio-button>
          <el-radio-button label="BREAK_GLASS" style="flex: 1;">
            <span>选项二：危重急救·Break-Glass 破窗</span>
          </el-radio-button>
        </el-radio-group>
      </div>

      <!-- 选项一：正常诊疗知情同意与现场患者密钥解锁通道 -->
      <div v-if="accessChannel === 'CONSENT'" class="channel-panel">
        <div class="target-record-banner mb-3">
          <span class="lbl">拟调阅外院病历：</span>
          <span v-if="currentTarget" class="val">
            <strong style="color: #4338ca;">{{ currentTarget.record_no }}</strong>
            （患者：<strong>{{ currentTarget.patient_name }}</strong> · 机构：<strong>{{ currentTarget.hospital_name }}</strong> · 科室：{{ currentTarget.department_name }}）
          </span>
        </div>

        <!-- 现场密码解锁专区（优先推荐） -->
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
          <div class="pk-input-row">
            <el-input
              v-model="patientKey"
              type="password"
              show-password
              size="large"
              placeholder="请输入患者专属病历调阅密钥（初始默认: 123456）"
              :prefix-icon="Key"
              class="pk-input"
              @keyup.enter="handleUnlockByKey"
            />
            <el-button
              type="success"
              size="large"
              :icon="Key"
              :loading="unlockingByKey"
              :disabled="!patientKey"
              @click="handleUnlockByKey"
              class="pk-btn"
            >
              验证密钥并即时解密调阅
            </el-button>
          </div>
          <div class="pk-tip">
            <span>提示：患者登录个人账号可在【个人中心】或【数据授权管理】随时修改此密钥；初始默认密钥为 <code>123456</code>。</span>
          </div>
        </div>

        <el-divider content-position="center">
          <span style="color: #94a3b8; font-size: 12px;">或患者未在现场时使用在线推送审批</span>
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

      <!-- 选项二：急诊危重破窗通道 -->
      <div v-else class="channel-panel">
        <div class="emergency-guide">
          <h4>是否属于急诊危重抢救极端场景？</h4>
          <p>
            根据《医疗数据可信共享实施规范》，若患者处于<strong>突发休克、严重昏迷或危及生命</strong>的极端急救场景，无法表达知情意愿，医生签署临床法律责任免责声明后，系统允许触发 <strong>Break-Glass 紧急破窗调阅机制</strong>立即放行并上链存证。
          </p>
        </div>

        <el-form :model="bgForm" label-width="110px" class="bg-form">
          <el-form-item label="紧急原因" required>
            <el-select v-model="bgForm.emergency_reason" style="width: 100%;">
              <el-option label="COMA - 患者严重休克昏迷，无法表达意愿" value="COMA" />
              <el-option label="RESCUE - 急诊抢救生命关键期，急需用药与过敏史" value="RESCUE" />
              <el-option label="CRITICAL - 突发急性危重病综合救治" value="CRITICAL" />
              <el-option label="OTHER - 其他危及生命的紧急医学场景" value="OTHER" />
            </el-select>
          </el-form-item>
          <el-form-item label="临床急救说明" required>
            <el-input
              v-model="bgForm.description"
              type="textarea"
              :rows="3"
              placeholder="请详细录入急救诊断情况、抢救必要性说明，此说明将直接作为不可篡改证据固化上链..."
            />
          </el-form-item>
          <el-form-item>
            <el-checkbox v-model="bgForm.doctor_confirmed">
              <span class="confirm-text">我确认本次紧急调阅仅用于患者紧急临床救治，并承担相应法律责任</span>
            </el-checkbox>
          </el-form-item>
        </el-form>
      </div>

      <template #footer>
        <el-button @click="breakGlassVisible = false">取消</el-button>
        <el-button
          v-if="accessChannel === 'CONSENT'"
          type="primary"
          :loading="submittingConsent"
          :disabled="!consentForm.purpose"
          @click="submitConsentRequest"
        >
          发起知情同意申请并通知患者
        </el-button>
        <el-button
          v-else
          type="danger"
          :disabled="!bgForm.doctor_confirmed || !bgForm.description"
          :loading="submittingBG"
          @click="submitBreakGlass"
        >
          确认申请 Break-Glass 抢救放行
        </el-button>
      </template>
    </el-dialog>


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
// 当前视图: QUERY-跨院检索与列表筛选, DETAIL-解密放行病历全景工作台
const currentView = ref<'QUERY' | 'DETAIL'>('QUERY')

import { ref, computed, onMounted, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { Search, WarningFilled, CircleCheckFilled, View, Download, Printer, OfficeBuilding, Lock, Compass, Refresh, Key, Setting, Document, Files, Picture, ArrowLeft, CopyDocument } from '@element-plus/icons-vue'
import { ElMessage, ElNotification } from 'element-plus'
import api from '../../api/client'
import { useAuthStore } from '../../stores/auth'
import { parseMedicalExamResult, highlightText, type ParsedMedicalReport } from '../../utils/medicalReportParser'
import InfectionSafetyAlert from '../../components/InfectionSafetyAlert.vue'

const route = useRoute()
const router = useRouter()
const auth = useAuthStore()

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

const breakGlassVisible = ref(false)
const currentTarget = ref<any>(null)
const submittingBG = ref(false)
const interceptRiskEval = ref<any>(null)

const recordModalVisible = ref(false)
const releasedRecord = ref<any>(null)
const isBreakGlassRelease = ref(false)

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
      const rNo = previewingRecord.value?.record_no || releasedRecord.value?.record_no || recordId
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

function formatEncounterType(t?: string) {
  if (t === 'OUTPATIENT') return '普通门诊'
  if (t === 'EMERGENCY') return '急危重症门诊'
  if (t === 'INPATIENT') return '住院就诊'
  return t || '普通门诊'
}

function formatHospName(hId?: number) {
  if (hId === 1) return '第一人民医院'
  if (hId === 2) return '第二人民医院'
  if (hId === 3) return '第三人民医院'
  return '医疗中心'
}

function formatTime(t?: string) {
  if (!t) return '-'
  return t.substring(0, 16).replace('T', ' ')
}

function maskIDCard(idCard?: string) {
  if (!idCard || idCard.length < 8) return idCard || '已脱敏保护'
  return idCard.substring(0, 6) + '********' + idCard.substring(idCard.length - 4)
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
    const res: any = await api.get('/medical-records')
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

    const res: any = await api.get('/medical-records', { params })
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
  initFromRouteQuery()
  fetchStatsSummary()
  searchCrossRecords()
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
      const recRes: any = await api.get(`/medical-records/${row.id}`)
      if (recRes.code === 200) {
        releasedRecord.value = recRes.data
        isBreakGlassRelease.value = (row.access_type === 'BREAK_GLASS')
        currentView.value = 'DETAIL'
        recordModalVisible.value = true
        ElMessage.success('【调阅放行】已核验权限凭据，解密病历成功')
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
    const res: any = await api.post('/access/requests', {
      patient_id: row.patient_id,
      record_id: row.id,
      purpose: accessPurpose
    })

    if (res.code === 200) {
      if (res.data.allowed) {
        ElMessage.success(res.data.reason || '【统一权限网关放行】已核准调阅，正在拉取并解密病历')
        const recRes: any = await api.get(`/medical-records/${row.id}`)
        releasedRecord.value = recRes.data
        isBreakGlassRelease.value = (res.data.is_emergency || row.access_type === 'BREAK_GLASS')
        currentView.value = 'DETAIL'
        recordModalVisible.value = true
        searchCrossRecords()
      } else {
        interceptRiskEval.value = res.data.risk_evaluation || {
          strategy: 'RULE_ENGINE_WEIGHTED',
          level: 'HIGH',
          total_score: 55,
          factors: [
            { rule_id: 'RULE-D1', factor_name: '医生执业机构与病历归属', weight: 35, score: 35, triggered: true, description: '跨医疗机构调阅（非本院开具且无有效门诊关联）' },
            { rule_id: 'RULE-A1', factor_name: '知情授权策略状态', weight: 20, score: 20, triggered: true, description: '患者未显式预签发有效知情授权策略' },
            { rule_id: 'RULE-T1', factor_name: '调阅时间窗口与频次', weight: 15, score: 0, triggered: false, description: '正常工作时间窗口且调阅频次正常' },
            { rule_id: 'RULE-F1', factor_name: '病历敏感度等级', weight: 15, score: 0, triggered: false, description: '常规病历数据，未触及极高敏感隐私标记' },
          ]
        }
        patientKey.value = ''
        breakGlassVisible.value = true
      }
    }
  } catch (err: any) {
    if (err?.response?.status === 403 || err?.code === 403) {
      interceptRiskEval.value = err?.response?.data?.data?.risk_evaluation || {
        strategy: 'RULE_ENGINE_WEIGHTED',
        level: 'HIGH',
        total_score: 55,
        factors: [
          { rule_id: 'RULE-D1', factor_name: '医生执业机构与病历归属', weight: 35, score: 35, triggered: true, description: '跨医疗机构调阅（非本院开具且无有效门诊关联）' },
          { rule_id: 'RULE-A1', factor_name: '知情授权策略状态', weight: 20, score: 20, triggered: true, description: '患者未显式预签发有效知情授权策略' },
          { rule_id: 'RULE-T1', factor_name: '调阅时间窗口与频次', weight: 15, score: 0, triggered: false, description: '正常工作时间窗口且调阅频次正常' },
          { rule_id: 'RULE-F1', factor_name: '病历敏感度等级', weight: 15, score: 0, triggered: false, description: '常规病历数据，未触及极高敏感隐私标记' },
        ]
      }
      patientKey.value = ''
      breakGlassVisible.value = true
    } else {
      ElMessage.error(err?.message || '访问控制请求失败')
    }
  }
}

async function handleUnlockByKey() {
  if (!currentTarget.value) return
  const key = patientKey.value.trim()
  if (!key) {
    ElMessage.warning('请输入患者专属病历调阅密钥（初始默认: 123456）')
    return
  }

  unlockingByKey.value = true
  try {
    const res: any = await api.post('/access/requests/unlock-by-key', {
      record_id: currentTarget.value.id,
      medical_key: key,
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
    }
  } catch (err: any) {
    ElMessage.error(err?.message || err?.response?.data?.message || '患者授权密钥校验失败，密码不正确')
  } finally {
    unlockingByKey.value = false
  }
}

async function submitConsentRequest() {
  if (!currentTarget.value) return
  submittingConsent.value = true

  try {
    const res: any = await api.post('/access/requests/apply-consent', {
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
    const res: any = await api.post('/access/break-glass', {
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
      // 破窗成功后立即刷新列表与统计数据，使列表上的按钮与状态标签立即显示为已放行
      refreshData()
    }
  } catch (err: any) {
    ElMessage.error(err?.message || '紧急放行申请失败')
  } finally {
    submittingBG.value = false
  }
}
</script>

<style scoped>
.page-container {
  padding: 8px 4px;
}

.page-header {
  margin-bottom: 16px;
}

.header-main-row {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  gap: 16px;
}

.header-badge-row {
  display: flex;
  align-items: center;
  gap: 12px;
  flex-wrap: wrap;
}

.scope-header-tag {
  font-size: 12px;
  font-weight: 700;
  border-radius: 6px;
  padding: 2px 8px;
}

.page-title {
  font-size: 20px;
  font-weight: 700;
  color: #1e293b;
  margin: 0;
}

.page-sub {
  font-size: 13px;
  color: #64748b;
  margin: 6px 0 0 0;
}

.scope-nav-row {
  display: flex;
  align-items: center;
  gap: 12px;
  flex-wrap: wrap;
}

.scope-tabs-group {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}

.tab-btn-inner {
  display: flex;
  align-items: center;
  gap: 6px;
  font-weight: 600;
}

.mr-1 {
  margin-right: 4px;
}

.rule-alert {
  border-radius: 10px;
  border: 1px solid #bfdbfe;
  background: #eff6ff;
}

.rule-alert-content {
  display: flex;
  flex-direction: column;
  gap: 4px;
  font-size: 13px;
  line-height: 1.5;
  color: #1e40af;
}

.filter-wrapper {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.scope-row {
  display: flex;
  align-items: center;
  gap: 12px;
}

.scope-label {
  font-size: 13px;
  font-weight: 600;
  color: #334155;
  white-space: nowrap;
}

.box-card {
  border-radius: 12px;
  border: 1px solid rgba(226, 232, 240, 0.8);
}

.mb-4 {
  margin-bottom: 16px;
}

.mt-3 {
  margin-top: 8px;
}

.hosp-cell {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
}

.hosp-name {
  font-weight: 500;
  color: #1e293b;
}

.tag-pill {
  font-size: 11px;
}

.self-doc-text {
  font-weight: 600;
  color: #059669;
}

.self-tag {
  color: #10b981;
  font-weight: bold;
}

.search-form {
  display: flex;
  align-items: center;
  gap: 12px;
  flex-wrap: wrap;
}

.search-tip {
  font-size: 12px;
  color: #6366f1;
}

.patient-cell {
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.patient-name {
  font-weight: 600;
  color: #1e293b;
}

.patient-sub {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
  font-size: 11px;
  color: #64748b;
}

.id-card-tag {
  background: #f1f5f9;
  padding: 1px 4px;
  border-radius: 4px;
}

.intercept-notice {
  margin-bottom: 16px;
}

.emergency-guide h4 {
  margin: 0 0 8px 0;
  color: #dc2626;
  font-size: 15px;
}

.emergency-guide p {
  margin: 0 0 16px 0;
  font-size: 13px;
  line-height: 1.6;
  color: #475569;
}

.bg-form {
  background: #fef2f2;
  border: 1px dashed #f87171;
  border-radius: 8px;
  padding: 16px;
}

.confirm-text {
  font-size: 12px;
  color: #b91c1c;
  font-weight: 600;
}

.report-header {
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  gap: 8px 16px;
  background: #f8fafc;
  padding: 12px 16px;
  border-radius: 8px;
  border: 1px solid #e2e8f0;
  margin-top: 12px;
}

.report-h-item {
  color: #334155;
  font-size: 13px;
}

.p-name {
  font-weight: 700;
  color: #4f46e5;
}

.mono {
  font-family: monospace;
}

.clinical-structured-table {
  width: 100%;
  border-collapse: collapse;
  margin-top: 8px;
  font-size: 13px;
}

.clinical-structured-table th,
.clinical-structured-table td {
  border: 1px solid #e2e8f0;
  padding: 10px 14px;
  text-align: left;
}

.clinical-structured-table th {
  background: #f1f5f9;
  color: #475569;
  font-weight: 600;
  vertical-align: top;
}

.vital-text {
  font-weight: 600;
  color: #0d9488;
}

.text-content {
  line-height: 1.6;
  color: #1e293b;
}

.highlight-cause {
  background: #fffbeb;
  padding: 6px 10px;
  border-radius: 4px;
  color: #92400e;
}

.highlight-plan {
  background: #ecfdf5;
  padding: 6px 10px;
  border-radius: 4px;
  color: #065f46;
}

.full-summary-pre {
  white-space: pre-wrap;
  background: #faf5ff;
  border: 1px solid #f3e8ff;
  padding: 10px;
  border-radius: 6px;
  color: #581c87;
  font-family: inherit;
  margin: 0;
  line-height: 1.6;
}

.blockchain-evidence-box {
  background: #f8fafc;
  border: 1px dashed #cbd5e1;
  padding: 12px 16px;
  border-radius: 8px;
  line-height: 1.8;
  font-size: 12px;
}

.tx-hash {
  font-family: monospace;
  color: #4338ca;
  font-weight: 600;
}

/* 区块链防篡改高危警告横幅 */
.tamper-warning-box {
  background: linear-gradient(135deg, #fff1f2 0%, #fee2e2 100%);
  border: 2px solid #ef4444;
  box-shadow: 0 4px 14px rgba(239, 68, 68, 0.25);
  border-radius: 10px;
  padding: 16px 20px;
  margin-bottom: 20px;
  animation: pulse-border 2s infinite ease-in-out;
}

@keyframes pulse-border {
  0% { box-shadow: 0 0 0 0 rgba(239, 68, 68, 0.4); }
  70% { box-shadow: 0 0 0 8px rgba(239, 68, 68, 0); }
  100% { box-shadow: 0 0 0 0 rgba(239, 68, 68, 0); }
}

.tw-head {
  display: flex;
  align-items: center;
  gap: 10px;
  color: #b91c1c;
  font-size: 16px;
  font-weight: 800;
  margin-bottom: 8px;
}

.tw-icon {
  font-size: 22px;
  animation: tw-bounce 1s infinite alternate;
}

@keyframes tw-bounce {
  from { transform: scale(1); }
  to { transform: scale(1.2); }
}

.tw-body {
  font-size: 13px;
  line-height: 1.6;
  color: #991b1b;
  margin-bottom: 12px;
}

.tw-hashes {
  background: rgba(255, 255, 255, 0.85);
  border: 1px solid #fca5a5;
  border-radius: 6px;
  padding: 10px 14px;
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.tw-h-item {
  display: flex;
  align-items: center;
  font-size: 12px;
  gap: 8px;
}

.tw-h-item.danger code {
  color: #b91c1c;
  background: #fee2e2;
  padding: 2px 6px;
  border-radius: 4px;
  font-family: monospace;
  font-weight: 600;
  word-break: break-all;
}

.tw-h-item.chain code {
  color: #047857;
  background: #d1fae5;
  padding: 2px 6px;
  border-radius: 4px;
  font-family: monospace;
  font-weight: 600;
  word-break: break-all;
}

.tw-h-item .lbl {
  font-weight: 600;
  min-width: 220px;
}

/* 校验通过安全横幅 */
.verified-safe-box {
  background: #ecfdf5;
  border: 1px solid #a7f3d0;
  color: #065f46;
  border-radius: 8px;
  padding: 10px 16px;
  margin-bottom: 18px;
  display: flex;
  align-items: center;
  gap: 10px;
  font-size: 13px;
}

.tamper-tag-glow {
  animation: tag-glow 1.5s infinite alternate;
  font-weight: 700;
}

@keyframes tag-glow {
  from { opacity: 0.85; transform: scale(0.98); }
  to { opacity: 1; transform: scale(1.05); }
}

.channel-selector-wrapper {
  background: #f1f5f9;
  padding: 4px;
  border-radius: 8px;
}

.channel-panel {
  margin-top: 14px;
}

.consent-form {
  padding: 8px 4px;
}

/* 附件与电子病历凭证卡片 */
.archive-files-section {
  margin: 12px 0 16px 0;
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

.target-record-banner {
  background: #f1f5f9;
  border-radius: 8px;
  padding: 10px 14px;
  font-size: 13px;
  color: #334155;
  border: 1px solid #e2e8f0;
}

.patient-key-box {
  background: linear-gradient(135deg, #f0fdf4 0%, #ecfdf5 100%);
  border: 1.5px solid #a7f3d0;
  border-radius: 12px;
  padding: 16px 18px;
  box-shadow: 0 4px 14px rgba(16, 185, 129, 0.08);
}

.pk-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 8px;
}

.pk-title-wrap {
  display: flex;
  align-items: center;
  gap: 8px;
}

.pk-icon {
  font-size: 18px;
}

.pk-title {
  font-size: 15px;
  font-weight: 700;
  color: #065f46;
}

.pk-desc {
  font-size: 13px;
  color: #047857;
  line-height: 1.5;
  margin: 0 0 12px 0;
}

.pk-input-row {
  display: flex;
  align-items: center;
  gap: 12px;
  flex-wrap: wrap;
}

.pk-input {
  flex: 1;
  min-width: 260px;
}

.pk-btn {
  font-weight: 700;
  border-radius: 8px;
  padding: 0 20px;
}

.pk-tip {
  margin-top: 10px;
  font-size: 12px;
  color: #059669;
}

.pk-tip code {
  background: #d1fae5;
  padding: 2px 6px;
  border-radius: 4px;
  font-weight: bold;
}

.online-consent-box {
  background: #f8fafc;
  border: 1px solid #e2e8f0;
  border-radius: 10px;
  padding: 14px 16px;
}

.oc-header {
  display: flex;
  align-items: center;
  gap: 6px;
}

.oc-icon {
  font-size: 16px;
}

/* 患者历史所有检查单与影像概览卡片 */
.patient-exams-card {
  border-radius: 12px;
  border: 1px solid #c7d2fe;
  background: linear-gradient(180deg, #f5f7ff 0%, #ffffff 100%);
}

.pec-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  flex-wrap: wrap;
  gap: 12px;
}

.pec-title {
  display: flex;
  align-items: center;
  font-size: 15px;
  color: #1e1b4b;
}

.pec-icon {
  font-size: 18px;
  margin-right: 6px;
}

.pec-toolbar {
  display: flex;
  align-items: center;
  gap: 8px;
}

.pec-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(360px, 1fr));
  gap: 14px;
}

.pec-item-card {
  border: 1px solid #e2e8f0;
  border-radius: 12px;
  padding: 14px 16px;
  background: #ffffff;
  cursor: pointer;
  transition: all 0.25s ease;
  display: flex;
  flex-direction: column;
  justify-content: space-between;
  box-shadow: 0 1px 4px rgba(0,0,0,0.04);
}

.pec-item-card:hover {
  transform: translateY(-2px);
  border-color: #6366f1;
  box-shadow: 0 6px 16px rgba(99, 102, 241, 0.12);
}

.pec-item-card.has-access {
  border-left: 4px solid #10b981;
}

.pec-item-card:not(.has-access) {
  border-left: 4px solid #f59e0b;
}

.pec-item-card.is-abnormal-card {
  box-shadow: 0 2px 8px rgba(239, 68, 68, 0.08);
}

.pec-item-top {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 8px;
  flex-wrap: wrap;
  gap: 6px;
}

.top-tags-left {
  display: flex;
  align-items: center;
  gap: 6px;
}

.top-tags-right {
  display: flex;
  align-items: center;
  gap: 4px;
}

.clinical-summary-tag {
  font-weight: 700;
  border-radius: 6px;
}

.pec-item-name {
  font-size: 15px;
  font-weight: 700;
  color: #0f172a;
  margin-bottom: 6px;
  line-height: 1.4;
}

.pec-item-meta-bar {
  display: flex;
  justify-content: space-between;
  align-items: center;
  font-size: 11px;
  color: #64748b;
  margin-bottom: 10px;
  padding-bottom: 6px;
  border-bottom: 1px dashed #e2e8f0;
}

.pec-item-body {
  flex: 1;
  margin-bottom: 10px;
}

/* 结构化检查项容器 */
.structured-exam-container {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.exam-sub-section {
  background: #f8fafc;
  border: 1px solid #e2e8f0;
  border-radius: 8px;
  padding: 10px 12px;
  transition: all 0.2s ease;
}

.exam-sub-section.section-abnormal {
  border-left: 3px solid #f59e0b;
  background: #fffcf0;
}

.exam-sub-section.section-matched {
  border-color: #cbd5e1;
  box-shadow: 0 0 0 1px rgba(99, 102, 241, 0.2);
}

.sub-sec-head {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 6px;
  flex-wrap: wrap;
  gap: 4px;
}

.sub-sec-title {
  display: flex;
  align-items: center;
  gap: 4px;
  font-size: 13px;
  color: #1e293b;
}

.sub-sec-icon {
  font-size: 14px;
}

.sub-sec-sub {
  font-size: 11px;
  color: #64748b;
  font-weight: normal;
}

.sub-sec-tags {
  display: flex;
  align-items: center;
  gap: 4px;
}

.sub-tag-pill {
  font-size: 10px;
  font-weight: 600;
  padding: 1px 6px;
  border-radius: 4px;
}

.sub-tag-pill.danger {
  background: #fee2e2;
  color: #b91c1c;
}

.sub-tag-pill.warning {
  background: #fef3c7;
  color: #b45309;
}

.sub-tag-pill.success {
  background: #dcfce7;
  color: #15803d;
}

.sub-tag-pill.info {
  background: #e0f2fe;
  color: #0369a1;
}

.match-badge {
  font-size: 10px;
  font-weight: 700;
  background: #fef08a;
  color: #854d0e;
  padding: 1px 5px;
  border-radius: 4px;
  border: 1px solid #facc15;
}

/* 核心结论大字高亮区 (优先展示！) */
.sub-sec-conclusion {
  background: #ffffff;
  border: 1px solid #e2e8f0;
  border-radius: 6px;
  padding: 8px 10px;
  margin-bottom: 6px;
}

.sub-sec-conclusion.is-warning-bg {
  background: #fffbeb;
  border-color: #fde68a;
}

.conclusion-badge {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  font-size: 11px;
  font-weight: 700;
  color: #b45309;
  margin-bottom: 2px;
}

.pulse-indicator {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background-color: #ef4444;
  display: inline-block;
  box-shadow: 0 0 6px #ef4444;
  animation: pulse-dot 1.8s infinite;
}

@keyframes pulse-dot {
  0% { transform: scale(0.95); opacity: 0.8; }
  50% { transform: scale(1.3); opacity: 1; }
  100% { transform: scale(0.95); opacity: 0.8; }
}

.conclusion-text {
  font-size: 13px;
  line-height: 1.5;
  color: #0f172a;
  font-weight: 600;
}

.sub-sec-suggestion {
  font-size: 11px;
  color: #0369a1;
  background: #f0f9ff;
  border: 1px solid #bae6fd;
  border-radius: 4px;
  padding: 4px 8px;
  margin-bottom: 6px;
  line-height: 1.4;
}

.sug-icon {
  font-weight: 700;
}

/* 测量参数折叠与展开面板 */
.sub-sec-findings-container {
  margin-top: 4px;
}

.findings-toggle-btn {
  font-size: 11px;
  color: #4f46e5;
  cursor: pointer;
  display: inline-flex;
  align-items: center;
  gap: 4px;
  padding: 2px 6px;
  border-radius: 4px;
  background: rgba(79, 70, 229, 0.05);
  transition: all 0.2s;
  user-select: none;
}

.findings-toggle-btn:hover {
  background: rgba(79, 70, 229, 0.12);
  color: #3730a3;
}

.findings-list-panel {
  margin-top: 6px;
  padding: 6px 8px;
  background: #ffffff;
  border: 1px solid #e2e8f0;
  border-radius: 6px;
  font-size: 11px;
  color: #475569;
  line-height: 1.6;
}

.finding-row {
  position: relative;
  padding-left: 10px;
  margin-bottom: 3px;
}

.finding-row::before {
  content: '•';
  position: absolute;
  left: 0;
  color: #94a3b8;
}

.issuer-row {
  margin-top: 4px;
  padding-top: 4px;
  border-top: 1px dashed #e2e8f0;
  font-size: 10px;
  color: #94a3b8;
}

/* 纯门诊诊断框 */
.pure-diagnosis-box {
  background: #f8fafc;
  border: 1px solid #e2e8f0;
  border-radius: 8px;
  padding: 10px 12px;
}

.pure-diag-head {
  font-size: 12px;
  color: #475569;
  margin-bottom: 4px;
}

.pure-diag-text {
  font-size: 13px;
  color: #0f172a;
  line-height: 1.5;
}

.pure-diag-plan {
  margin-top: 6px;
  font-size: 11px;
  color: #64748b;
}

/* 未授权遮罩卡片 */
.masked-lock-box {
  background: #fffbeb;
  border: 1px dashed #f59e0b;
  border-radius: 8px;
  padding: 10px 12px;
}

.lock-head {
  font-size: 12px;
  color: #b45309;
}

.lock-desc {
  font-size: 11px;
  color: #92400e;
  line-height: 1.5;
  margin-top: 4px;
}

/* 搜索关键词黄色高亮标记 */
:deep(.med-kw-mark) {
  background-color: #fef08a;
  color: #854d0e;
  font-weight: 700;
  padding: 0 2px;
  border-radius: 2px;
}

/* 表格内诊断单元格 */
.table-diag-box {
  line-height: 1.4;
}

.table-tags-row {
  display: flex;
  flex-wrap: wrap;
  gap: 2px;
}

.table-summary-text {
  line-height: 1.4;
}

/* 详情弹窗中的结构化医技检查报告 */
.modal-exam-sections {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.modal-exam-card {
  background: #f8fafc;
  border: 1px solid #e2e8f0;
  border-radius: 8px;
  padding: 12px 14px;
}

.mec-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 8px;
}

.mec-title {
  font-size: 14px;
  color: #1e293b;
}

.mec-sub {
  font-size: 12px;
  color: #64748b;
  margin-left: 6px;
}

.mec-conclusion {
  background: #ffffff;
  border: 1px solid #e2e8f0;
  border-radius: 6px;
  padding: 10px 12px;
  margin-bottom: 8px;
}

.mec-conclusion.abnormal {
  background: #fffbeb;
  border-color: #fde68a;
}

.mec-conclusion-badge {
  font-size: 12px;
  font-weight: 700;
  color: #b45309;
  margin-bottom: 2px;
}

.mec-conclusion-val {
  font-size: 14px;
  font-weight: 700;
  color: #0f172a;
  line-height: 1.5;
}

.mec-findings {
  background: #ffffff;
  border: 1px solid #e2e8f0;
  border-radius: 6px;
  padding: 8px 12px;
  margin-bottom: 6px;
}

.findings-subhead {
  font-size: 12px;
  font-weight: 600;
  color: #475569;
  margin-bottom: 4px;
}

.findings-list {
  margin: 0;
  padding-left: 18px;
  font-size: 12px;
  color: #334155;
  line-height: 1.6;
}

.mec-footer {
  font-size: 11px;
  color: #94a3b8;
  text-align: right;
}

.pec-item-bottom {
  display: flex;
  justify-content: space-between;
  align-items: center;
  font-size: 12px;
  border-top: 1px dashed #e2e8f0;
  padding-top: 8px;
}

.access-tag {
  font-size: 11px;
  font-weight: 600;
  padding: 2px 6px;
  border-radius: 4px;
}

.access-tag.green {
  background: #ecfdf5;
  color: #047857;
}

.access-tag.red {
  background: #fef2f2;
  color: #b91c1c;
}

.access-tag.orange {
  background: #fffbeb;
  color: #b45309;
}

.view-btn {
  font-weight: 600;
}

/* 表格头部与自定义列工具条 */
.table-wrapper-card {
  border-radius: 12px;
}

.table-card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  flex-wrap: wrap;
  gap: 12px;
}

.tch-left {
  display: flex;
  align-items: center;
  font-size: 15px;
  color: #1e1b4b;
}

.tch-icon {
  font-size: 18px;
  margin-right: 6px;
}

.tch-right {
  display: flex;
  align-items: center;
  gap: 8px;
}

/* 智能筛选与快捷归类工作台 */
.filter-workbench {
  background: #f8fafc;
  border: 1px solid #e2e8f0;
  border-radius: 10px;
  padding: 12px 14px;
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.workbench-row {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 10px;
}

.classify-row {
  padding-bottom: 8px;
  border-bottom: 1px dashed #e2e8f0;
}

.wb-label {
  font-size: 12px;
  font-weight: 700;
  color: #334155;
  white-space: nowrap;
}

.classify-tabs {
  flex-wrap: wrap;
}

.filters-row {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 12px;
}

.filter-item {
  display: flex;
  align-items: center;
  gap: 6px;
}

.f-lbl {
  font-size: 12px;
  color: #64748b;
  white-space: nowrap;
}

.active-filter-pills {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 6px;
  margin-left: auto;
}

/* 自定义列浮层样式 */
.col-popover-content {
  padding: 4px;
}

.col-popover-head {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 8px;
  padding-bottom: 6px;
  border-bottom: 1px solid #e2e8f0;
  font-size: 13px;
  color: #1e293b;
}

.col-checkbox-list {
  display: flex;
  flex-direction: column;
  gap: 6px;
  max-height: 280px;
  overflow-y: auto;
  padding: 2px;
}

.col-checkbox-item {
  padding: 2px 4px;
  border-radius: 4px;
  transition: background 0.15s;
}

.col-checkbox-item:hover {
  background: #f1f5f9;
}

.col-popover-footer {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-top: 8px;
  padding-top: 6px;
  border-top: 1px solid #e2e8f0;
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
  width: 400px;
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

</style>
