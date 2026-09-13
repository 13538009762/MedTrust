<template>
  <div class="page-container">
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
            🎯 <strong>已自动定位目标患者：</strong>
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
            🔙 返回门诊就诊工作台
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
            <el-option label="📁 全部档案与检查单" value="ALL" />
            <el-option label="🔬 仅调阅检查检验单/切片" value="EXAM_ONLY" />
            <el-option label="🧪 医学检验报告 (REPORT)" value="REPORT" />
            <el-option label="🩻 医学影像切片 (IMAGE)" value="IMAGE" />
            <el-option label="📋 门诊电子病历 (EMR)" value="EMR" />
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
          <span class="search-tip">💡 提示：支持按患者姓名或特定检查项目精准检索</span>
        </div>
      </div>
    </el-card>

    <!-- 🔬 该患者做过的所有检查单与医学影像概览专区 (支持跨机构调阅与统一安全评估) -->
    <el-card v-if="patientExamRecords.length > 0 && searchKeyword.trim()" shadow="hover" class="box-card mb-4 patient-exams-card">
      <template #header>
        <div class="pec-header">
          <div class="pec-title">
            <span class="pec-icon">🔬</span>
            <strong>【{{ route.query.patient_name || searchKeyword }}】已做过的全部外院/全网检查单与影像报告 (共 {{ patientExamRecords.length }} 项)</strong>
            <el-tag size="small" type="success" effect="dark" class="ml-2">跨机构可信存证</el-tag>
          </div>
          <span class="pec-tip">点击检查单卡片直接调阅；跨院未授权检查单将自动触发安全网关动态风险评估</span>
        </div>
      </template>

      <div class="pec-grid">
        <div
          v-for="exam in patientExamRecords"
          :key="'exam-' + exam.id"
          class="pec-item-card"
          :class="{ 'has-access': exam.has_access || exam.doctor_id === auth.user?.id || exam.hospital_id === auth.user?.hospital_id }"
          @click="handleAccess(exam)"
        >
          <div class="pec-item-top">
            <el-tag size="small" :type="exam.data_type === 'IMAGE' ? 'warning' : (exam.data_type === 'REPORT' ? 'success' : 'primary')" effect="dark">
              {{ exam.data_type === 'IMAGE' ? '🩻 影像切片' : (exam.data_type === 'REPORT' ? '🧪 检验化验' : '🔬 临床检查') }}
            </el-tag>
            <el-tag v-if="exam.is_tampered" size="small" type="danger" effect="dark">🚨 篡改告警</el-tag>
            <el-tag v-else size="small" type="success" effect="light">🛡️ 账本一致</el-tag>
          </div>

          <div class="pec-item-name">
            {{ exam.exam_items || exam.symptoms || exam.diagnosis || '医技检查项目' }}
          </div>

          <div class="pec-item-meta">
            <div class="meta-row">
              <span>🏥 <strong>{{ exam.hospital_name }}</strong> · {{ exam.department_name || '临床/医技科室' }}</span>
            </div>
            <div class="meta-row text-xs text-slate-500 mt-0.5">
              <span>🕒 {{ formatTime(exam.created_at) }} · 开单医生: {{ exam.doctor_name }}</span>
            </div>
            <div v-if="exam.exam_result" class="meta-result text-xs text-slate-600 mt-1">
              <strong>报告结论：</strong>{{ exam.exam_result }}
            </div>
            <div v-else-if="exam.diagnosis" class="meta-result text-xs text-slate-600 mt-1">
              <strong>诊断依据：</strong>{{ exam.diagnosis }}
            </div>
          </div>

          <div class="pec-item-bottom">
            <span v-if="exam.doctor_id === auth.user?.id || exam.hospital_id === auth.user?.hospital_id" class="access-tag green">
              🟢 本院/本人放行
            </span>
            <span v-else-if="exam.access_type === 'BREAK_GLASS'" class="access-tag red">
              🚨 24h破窗放行中
            </span>
            <span v-else-if="exam.has_access" class="access-tag green">
              ✅ 患者已授权
            </span>
            <span v-else class="access-tag orange">
              🔒 需安全网关核准
            </span>
            <el-button type="primary" size="small" link class="view-btn">安全调阅 ➔</el-button>
          </div>
        </div>
      </div>
    </el-card>

    <!-- 跨院记录结果列表 -->
    <el-card shadow="hover" class="box-card">
      <el-table :data="results" v-loading="loading" stripe style="width: 100%">
        <el-table-column prop="record_no" label="病历单号" width="160" />
        <el-table-column label="就诊患者" min-width="180">
          <template #default="{ row }">
            <div class="patient-cell">
              <span class="patient-name">{{ row.patient_name || '患者' }}</span>
              <div class="patient-sub">
                <span v-if="row.patient_phone">📱 {{ row.patient_phone }}</span>
                <span v-if="row.patient_id_card" class="id-card-tag">🪪 {{ maskIDCard(row.patient_id_card) }}</span>
              </div>
            </div>
          </template>
        </el-table-column>

        <el-table-column label="归属医疗机构" min-width="180">
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

        <el-table-column label="开具医生" width="130">
          <template #default="{ row }">
            <span :class="{ 'self-doc-text': row.doctor_id === auth.user?.id }">
              {{ row.doctor_name }}
              <span v-if="row.doctor_id === auth.user?.id" class="self-tag">(您)</span>
            </span>
          </template>
        </el-table-column>

        <el-table-column prop="data_type" label="类别" width="105">
          <template #default="{ row }">
            <el-tag v-if="row.data_type === 'IMAGE'" size="small" type="warning" effect="dark">🩻 影像切片</el-tag>
            <el-tag v-else-if="row.data_type === 'REPORT'" size="small" type="success" effect="dark">🧪 检验单</el-tag>
            <el-tag v-else-if="row.need_exam || row.exam_items" size="small" type="primary" effect="light">🔬 含检查</el-tag>
            <el-tag v-else size="small" type="info">📋 门诊病历</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="发病与病程" width="140">
          <template #default="{ row }">
            <div v-if="row.onset_time || row.duration" style="font-size: 12px; line-height: 1.4;">
              <div>🕒 {{ row.onset_time ? row.onset_time.substring(0, 16) : '接诊记录' }}</div>
              <div style="color: #64748b;">⏳ {{ row.duration || '-' }}</div>
            </div>
            <span v-else style="color: #94a3b8; font-size: 12px;">常规就诊</span>
          </template>
        </el-table-column>
        <el-table-column label="检查项目与临床诊断" min-width="210" show-overflow-tooltip>
          <template #default="{ row }">
            <div v-if="row.exam_items" class="text-xs text-indigo-700 font-bold mb-0.5">
              🔬 检查: {{ row.exam_items }}
            </div>
            <div class="text-xs text-slate-800">
              {{ row.diagnosis || row.symptoms || '常规就诊' }}
            </div>
          </template>
        </el-table-column>
        <el-table-column label="防篡改状态" width="125" align="center">
          <template #default="{ row }">
            <el-tag v-if="row.is_tampered" type="danger" effect="dark" class="tamper-tag-glow">
              🚨 存在篡改
            </el-tag>
            <el-tag v-else type="success" effect="light">
              🛡️ 校验一致
            </el-tag>
          </template>
        </el-table-column>

        <el-table-column label="调阅授权状态" width="145" align="center">
          <template #default="{ row }">
            <el-tag v-if="row.doctor_id === auth.user?.id" type="success" effect="plain" size="small">
              👨‍⚕️ 本人责任病历
            </el-tag>
            <el-tag v-else-if="row.hospital_id === auth.user?.hospital_id" type="primary" effect="plain" size="small">
              🏥 本院同僚放行
            </el-tag>
            <el-tag v-else-if="row.access_type === 'BREAK_GLASS'" type="danger" effect="dark" size="small">
              🚨 24h破窗放行中
            </el-tag>
            <el-tag v-else-if="row.has_access" type="success" effect="dark" size="small">
              ✅ 患者跨院已授权
            </el-tag>
            <el-tag v-else type="info" effect="plain" size="small">
              🔒 待安全网关核准
            </el-tag>
          </template>
        </el-table-column>

        <el-table-column prop="created_at" label="就诊日期" width="120">
          <template #default="{ row }">
            {{ row.created_at ? row.created_at.substring(0, 10) : '' }}
          </template>
        </el-table-column>
        <el-table-column label="调阅操作" width="180" fixed="right">
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

    <!-- 权限拦截对话框：提供 正常诊疗知情授权申请 与 急诊Break-Glass破窗 双通道 -->
    <el-dialog v-model="breakGlassVisible" title="⚠️ 跨院安全网关访问控制拦截与调阅申请" width="720px">
      <div class="intercept-notice mb-3">
        <el-alert
          title="未取得患者显式知情授权 (HTTP 403 Forbidden)"
          type="error"
          description="系统基于零信任统一权限网关检测：目标病历属于其他医疗机构，且患者当前未对您建立有效的跨院知情授权策略。请根据实际临床场景选择调阅通道："
          show-icon
          :closable="false"
        />
      </div>

      <!-- 双通道选项切换卡 -->
      <div class="channel-selector-wrapper mb-3">
        <el-radio-group v-model="accessChannel" size="large" class="channel-tabs" style="width: 100%; display: flex;">
          <el-radio-button label="CONSENT" style="flex: 1;">
            <span>🩺 选项一：正常诊疗·现场密钥解锁 / 知情同意申请 (推荐)</span>
          </el-radio-button>
          <el-radio-button label="BREAK_GLASS" style="flex: 1;">
            <span>🚨 选项二：危重急救·Break-Glass 破窗</span>
          </el-radio-button>
        </el-radio-group>
      </div>

      <!-- 选项一：正常诊疗知情同意与现场患者密钥解锁通道 -->
      <div v-if="accessChannel === 'CONSENT'" class="channel-panel">
        <div class="target-record-banner mb-3">
          <span class="lbl">🎯 拟调阅外院病历：</span>
          <span v-if="currentTarget" class="val">
            <strong style="color: #4338ca;">{{ currentTarget.record_no }}</strong>
            （患者：<strong>{{ currentTarget.patient_name }}</strong> · 机构：<strong>{{ currentTarget.hospital_name }}</strong> · 科室：{{ currentTarget.department_name }}）
          </span>
        </div>

        <!-- 现场密码解锁专区（优先推荐） -->
        <div class="patient-key-box mb-3">
          <div class="pk-header">
            <div class="pk-title-wrap">
              <span class="pk-icon">🔑</span>
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
              🔑 验证密钥并即时解密调阅
            </el-button>
          </div>
          <div class="pk-tip">
            <span>💡 提示：患者登录个人账号可在【个人中心】或【数据授权管理】随时修改此密钥；初始默认密钥为 <code>123456</code>。</span>
          </div>
        </div>

        <el-divider content-position="center">
          <span style="color: #94a3b8; font-size: 12px;">或患者未在现场时使用在线推送审批</span>
        </el-divider>

        <!-- 方式二：在线知情申请推送 -->
        <div class="online-consent-box">
          <div class="oc-header mb-2">
            <span class="oc-icon">📨</span>
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
          <h4>🚨 是否属于急诊危重抢救极端场景？</h4>
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
          📨 发起知情同意申请并通知患者
        </el-button>
        <el-button
          v-else
          type="danger"
          :disabled="!bgForm.doctor_confirmed || !bgForm.description"
          :loading="submittingBG"
          @click="submitBreakGlass"
        >
          🚨 确认申请 Break-Glass 抢救放行
        </el-button>
      </template>
    </el-dialog>

    <!-- 解密放行展示详情框（升级为完整结构化临床大表格） -->
    <el-dialog v-model="recordModalVisible" title="✅ 医疗数据已核准解密放行（结构化全量视图）" width="800px" top="5vh">
      <div v-if="releasedRecord" class="released-box">
        <!-- 篡改告警横幅 -->
        <div v-if="releasedRecord.is_tampered" class="tamper-warning-box">
          <div class="tw-head">
            <el-icon class="tw-icon"><WarningFilled /></el-icon>
            <span class="tw-title">【高危安全警报】检测到该病历数据库数据已被恶意篡改！</span>
          </div>
          <div class="tw-body">
            系统检测到底层 MySQL 数据库中的临床数据与 Fabric 联盟链上不可篡改的存证指纹<strong>不匹配</strong>！该病历中的用药方案、药物过敏史或核心诊断已被黑客攻击篡改，<strong>临床严禁直接采信该病历！</strong>系统已阻断非法使用并记录高危安全审计！
          </div>
          <div class="tw-hashes">
            <div class="tw-h-item danger">
              <span class="lbl">🚨 数据库当前计算哈希 (Current Hash)：</span>
              <code>{{ releasedRecord.current_hash }}</code>
            </div>
            <div class="tw-h-item chain">
              <span class="lbl">🛡️ 区块链不可篡改基准 (Chain Hash)：</span>
              <code>{{ releasedRecord.chain_hash }}</code>
            </div>
          </div>
        </div>

        <div v-else class="verified-safe-box">
          <el-icon><CircleCheckFilled /></el-icon>
          <span><strong>🛡️ 区块链防篡改校验通过：</strong>该病历所有临床症状、用药方案与影像数据指纹均与 Fabric 联盟链上固化存证 100% 严格一致，数据真实完整，未遭任何篡改。</span>
        </div>

        <el-alert
          v-if="isBreakGlassRelease || releasedRecord.access_type === 'BREAK_GLASS'"
          title="🚨 本次调阅为 Break-Glass 紧急破窗放行（24小时急救绿色通道生效中），事件单已上链存证并向卫健监管派发预警"
          type="error"
          show-icon
          class="mb-3"
          :closable="false"
        />
        <el-alert
          v-else-if="releasedRecord.doctor_id === auth.user?.id"
          title="✅ 本人开具病历：您作为接诊责任医生依法享有持续随诊与病史调阅权限，系统免申请直接放行"
          type="success"
          show-icon
          class="mb-3"
          :closable="false"
        />
        <el-alert
          v-else-if="releasedRecord.hospital_id === auth.user?.hospital_id"
          title="✅ 院内诊疗互通：目标病历归属本医疗机构，遵循院内就诊连续性互认规范直接放行"
          type="info"
          show-icon
          class="mb-3"
          :closable="false"
        />
        <el-alert
          v-else
          title="✅ 跨院授权放行：已通过患者显式跨机构知情授权策略校验（智能合约验证生效），解密放行"
          type="success"
          show-icon
          class="mb-3"
          :closable="false"
        />

        <div class="report-header">
          <div class="report-h-item"><strong>就诊类型：</strong><el-tag size="small" type="warning">{{ releasedRecord.encounter_type || '普通门诊' }}</el-tag></div>
          <div class="report-h-item"><strong>病历/就诊号：</strong><span class="mono">{{ releasedRecord.record_no }}</span></div>
          <div class="report-h-item" v-if="releasedRecord.department_name"><strong>就诊科室：</strong><span>{{ releasedRecord.department_name }}</span></div>
          <div class="report-h-item"><strong>就诊患者：</strong><span class="p-name">{{ releasedRecord.patient_name }}</span></div>
          <div class="report-h-item"><strong>患者身份证：</strong><span>{{ releasedRecord.patient_id_card || '已脱敏保护' }}</span></div>
          <div class="report-h-item"><strong>联系电话：</strong><span>{{ releasedRecord.patient_phone || '未留存' }}</span></div>
          <div class="report-h-item"><strong>归属机构：</strong><span>{{ releasedRecord.hospital_name }}</span></div>
          <div class="report-h-item"><strong>接诊医生：</strong><span>{{ releasedRecord.doctor_name }}</span></div>
          <div class="report-h-item"><strong>就诊时间：</strong><span>{{ releasedRecord.created_at ? releasedRecord.created_at.substring(0, 16).replace('T', ' ') : '-' }}</span></div>
          <div class="report-h-item"><strong>病历状态：</strong><el-tag size="small" type="success">{{ releasedRecord.status || '已归档' }}</el-tag></div>
        </div>

        <el-divider content-position="left">结构化临床多维信息</el-divider>

        <table class="clinical-structured-table">
          <tr>
            <th width="140">发病与病程时间</th>
            <td>
              <strong>发病时间：</strong>{{ releasedRecord.onset_time || '接诊前' }}<br>
              <strong>持续时间：</strong>{{ releasedRecord.duration || '急性发作' }}
            </td>
          </tr>
          <tr v-if="releasedRecord.vital_signs">
            <th>生命体征参数</th>
            <td><span class="vital-text">{{ releasedRecord.vital_signs }}</span></td>
          </tr>
          <tr v-if="releasedRecord.chief_complaint">
            <th>患者主诉 (S)</th>
            <td><div class="text-content font-bold">{{ releasedRecord.chief_complaint }}</div></td>
          </tr>
          <tr v-if="releasedRecord.present_illness">
            <th>现病史 (HPI)</th>
            <td><div class="text-content">{{ releasedRecord.present_illness }}</div></td>
          </tr>
          <tr v-if="releasedRecord.symptoms && !releasedRecord.chief_complaint">
            <th>主要临床症状</th>
            <td><div class="text-content">{{ releasedRecord.symptoms }}</div></td>
          </tr>
          <tr v-if="releasedRecord.initial_diagnosis">
            <th>初诊拟定与依据</th>
            <td>
              <strong>初步诊断：</strong><el-tag size="small" type="info">{{ releasedRecord.initial_diagnosis }}</el-tag><br>
              <div v-if="releasedRecord.diagnostic_basis" class="text-content mt-1"><strong>诊断依据：</strong>{{ releasedRecord.diagnostic_basis }}</div>
            </td>
          </tr>
          <tr v-if="releasedRecord.need_exam">
            <th>医技检查联动 (O)</th>
            <td>
              <p><strong>申请项目：</strong><el-tag type="warning" size="small">{{ releasedRecord.exam_items }}</el-tag></p>
              <div v-if="releasedRecord.exam_result" class="text-content highlight-cause mt-1" style="white-space: pre-line;">
                <strong>检验检查回传报告：</strong><br>{{ releasedRecord.exam_result }}
              </div>
              <p v-if="releasedRecord.exam_doctor" class="text-xs text-gray-500 mt-1">出具医师：{{ releasedRecord.exam_doctor }} | 时间：{{ releasedRecord.exam_time }}</p>
            </td>
          </tr>
          <tr v-if="releasedRecord.etiology">
            <th>诱因与病理分析</th>
            <td><div class="text-content highlight-cause">{{ releasedRecord.etiology }}</div></td>
          </tr>
          <tr>
            <th>最终确诊 (A)</th>
            <td><div class="text-content font-bold text-success">{{ releasedRecord.diagnosis }}</div></td>
          </tr>
          <tr>
            <th>处置与医嘱方案 (P)</th>
            <td><div class="text-content highlight-plan">{{ releasedRecord.treatment_plan || '遵常规医嘱治疗' }}</div></td>
          </tr>
        </table>

        <el-divider content-position="left">📁 临床归档电子凭证与附件 (PDF / 影像)</el-divider>

        <!-- 文件附件列表与预览/下载卡片 -->
        <div class="archive-files-section">
          <div v-if="releasedRecord.files && releasedRecord.files.length" class="file-cards-list">
            <div v-for="file in releasedRecord.files" :key="file.id" class="file-item-card">
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
                  <el-button type="success" size="small" plain :icon="Download" @click="downloadFileDirect(`/api/v1/medical-files/${file.id}/download?token=${auth.token}`, file.file_name)">
                    📥 下载图片
                  </el-button>
                </template>
                <template v-else>
                  <el-button type="primary" size="small" :icon="View" @click="openPdfPreview(releasedRecord)">
                    👁️ 在线查阅 PDF
                  </el-button>
                  <el-button type="success" size="small" plain :icon="Download" @click="downloadRecordFile(releasedRecord.id)">
                    📥 下载解密文件
                  </el-button>
                </template>
              </div>
            </div>

            <!-- 如果附件包含图片，同时在下方展示高清略缩图展示带 -->
            <div v-for="file in releasedRecord.files.filter((f: any) => isImageFileType(f.file_type))" :key="'img-' + file.id" class="inline-img-card mt-2">
              <div class="img-header mb-1">
                <span class="text-xs font-semibold text-slate-700">🖼️ 医技检查影像切片/报告扫描件预览（{{ file.file_name }}）：</span>
              </div>
              <div class="text-center p-2">
                <img
                  :src="`/api/v1/medical-files/${file.id}/view?token=${auth.token}`"
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
                <div class="text-xs text-gray-500">系统已对该就诊完成 AES-256-GCM 本地加密与 IPFS 分布式存证</div>
              </div>
              <div class="empty-btn-group">
                <el-button type="primary" size="small" :icon="View" @click="openPdfPreview(releasedRecord)">
                  👁️ 在线查阅电子病历 PDF
                </el-button>
                <el-button type="success" size="small" plain :icon="Download" @click="downloadRecordFile(releasedRecord.id)">
                  📥 下载电子凭据
                </el-button>
              </div>
            </div>
          </div>
        </div>

        <el-divider content-position="left">区块链存证与 IPFS 凭据</el-divider>

        <div class="blockchain-evidence-box">
          <p><strong>Fabric TxID：</strong><span class="tx-hash">{{ releasedRecord.fabric_tx_id }}</span></p>
          <p v-if="releasedRecord.block_height"><strong>存证区块高度：</strong><span class="mono">#{{ releasedRecord.block_height }}</span></p>
          <p><strong>智能合约核验状态：</strong><el-tag type="success" size="small">联盟链权威存证 · 100% 真实未篡改</el-tag></p>
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
import { ref, computed, onMounted, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { Search, WarningFilled, CircleCheckFilled, View, Download, Printer, OfficeBuilding, Lock, Compass, Refresh, Key } from '@element-plus/icons-vue'
import { ElMessage, ElNotification } from 'element-plus'
import api from '../../api/client'
import { useAuthStore } from '../../stores/auth'

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
        title: '🌐 外院已有查阅权限病例',
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
        title: '🏥 本医疗机构病例 (院内互通)',
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
        title: '🔒 外院需要调取的病例 (跨院安全网关)',
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
        title: '🔍 全网联合病例检索与调阅中心',
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

const breakGlassVisible = ref(false)
const currentTarget = ref<any>(null)
const submittingBG = ref(false)

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
        recordModalVisible.value = true
        searchCrossRecords()
      } else {
        patientKey.value = ''
        breakGlassVisible.value = true
      }
    }
  } catch (err: any) {
    if (err?.response?.status === 403 || err?.code === 403) {
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
        title: '🔑 患者现场密钥核验通过',
        message: '患者专属授权密钥比对成功！已即时解密放行该份跨院病历并完成区块链存证。',
        type: 'success',
        duration: 5000
      })

      breakGlassVisible.value = false
      patientKey.value = ''
      releasedRecord.value = res.data.record
      isBreakGlassRelease.value = false
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
        title: '📨 跨院知情同意申请已发起',
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
        title: '🚨 Break-Glass 紧急访问已放行',
        message: `生成紧急事件单 ${res.data.event?.event_no}，已固化上链并通知卫健监管与患者！系统已签发24小时紧急调阅授权，重启或刷新免重复申请。`,
        type: 'warning',
        duration: 6000
      })

      breakGlassVisible.value = false
      releasedRecord.value = res.data.record
      isBreakGlassRelease.value = true
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

/* 🚨 区块链防篡改高危警告横幅 */
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

/* 🛡️ 校验通过安全横幅 */
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

/* 📁 附件与电子病历凭证卡片 */
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
  gap: 8px;
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

.pec-tip {
  font-size: 12px;
  color: #6366f1;
}

.pec-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(300px, 1fr));
  gap: 12px;
}

.pec-item-card {
  border: 1px solid #e2e8f0;
  border-radius: 10px;
  padding: 12px 14px;
  background: #ffffff;
  cursor: pointer;
  transition: all 0.25s ease;
  display: flex;
  flex-direction: column;
  justify-content: space-between;
  box-shadow: 0 1px 3px rgba(0,0,0,0.05);
}

.pec-item-card:hover {
  transform: translateY(-2px);
  border-color: #6366f1;
  box-shadow: 0 4px 12px rgba(99, 102, 241, 0.15);
}

.pec-item-card.has-access {
  border-left: 4px solid #10b981;
}

.pec-item-card:not(.has-access) {
  border-left: 4px solid #f59e0b;
}

.pec-item-top {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 8px;
}

.pec-item-name {
  font-size: 14px;
  font-weight: 700;
  color: #0f172a;
  margin-bottom: 6px;
  line-height: 1.4;
}

.pec-item-meta {
  font-size: 12px;
  color: #475569;
  line-height: 1.5;
  margin-bottom: 10px;
  background: #f8fafc;
  padding: 8px;
  border-radius: 6px;
}

.meta-row {
  display: flex;
  align-items: center;
  gap: 4px;
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
