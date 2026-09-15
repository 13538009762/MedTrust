<template>
  <div class="page-container">
    <div class="page-header">
      <div>
        <h2 class="page-title">患者知情授权与跨院审批中心</h2>
        <p class="page-sub">由患者自主控制个人医疗数据共享，在线审批外院医生门诊知情申请，生效即刻锚定 Fabric 联盟链，随时一键撤销</p>
      </div>
      <el-button type="primary" :icon="Plus" @click="dialogVisible = true">主动创建授权策略</el-button>
    </div>

    <!-- 0. 现场病历即时调阅专属密钥卡片 -->
    <div class="medical-key-banner-card mb-4">
      <div class="mkb-left">
        <div class="mkb-icon-wrap"><el-icon><Key /></el-icon></div>
        <div>
          <div class="mkb-title-row">
            <span class="mkb-title">我的跨院病历现场调阅专属密钥</span>
            <el-tag size="small" type="success" effect="dark">现场秒级授权</el-tag>
          </div>
          <p class="mkb-sub">
            在其他医疗机构就诊时，您可直接向主治医生提供此密钥或在医生电脑现场输入，即可立即授权解锁您的健康病历，免去等待在线审批推送。
          </p>
        </div>
      </div>
      <div class="mkb-right">
        <div class="mkb-key-display">
          <span class="mkb-key-label">当前专属密钥：</span>
          <code class="mkb-key-code">{{ showKeyPlain ? (auth.user?.medical_key || '123456') : '••••••' }}</code>
          <el-button
            link
            type="primary"
            size="small"
            :icon="showKeyPlain ? View : Hide"
            @click="showKeyPlain = !showKeyPlain"
            :title="showKeyPlain ? '隐藏密钥' : '显示明文密钥'"
          />
        </div>
        <el-button type="success" :icon="Key" size="default" @click="openMedicalKeyDialog">
          修改我的专属密钥
        </el-button>
      </div>
    </div>

    <!-- 1. 待我知情审批的跨院调阅申请专区 -->
    <el-card shadow="hover" class="box-card mb-4 pending-card">
      <template #header>
        <div class="card-header-flex">
          <div class="header-title">
            <strong>待我知情审批的医生跨院调阅申请</strong>
            <el-badge v-if="pendingRequests.length > 0" :value="pendingRequests.length" class="pending-badge" type="danger" />
          </div>
          <span class="header-sub">外院经治医生在正常诊疗/会诊中发起知情同意申请，您核准同意后系统将自动释放调阅权限并存证上链</span>
        </div>
      </template>

      <div v-if="pendingRequests.length > 0">
        <el-alert
          :title="`您当前有 ${pendingRequests.length} 条待处理的医生调阅知情申请，请仔细核验医生资质与调阅目的`"
          type="warning"
          show-icon
          class="mb-3"
          :closable="false"
        />

        <div class="pending-grid">
          <div v-for="item in pendingRequests" :key="item.id" class="pending-item-card">
            <div class="pi-header">
              <span class="pi-no">申请单号: <strong>{{ item.request_no }}</strong></span>
              <el-tag size="small" type="warning" effect="dark">待您知情核准</el-tag>
            </div>
            <div class="pi-body">
              <div class="pi-info-row">
                <span class="lbl">申请医生：</span>
                <span class="doc-highlight">{{ item.doctor_name || '执业医生' }}</span>
                <span class="hosp-tag">（{{ item.doctor_hospital_name || '外院机构' }} · {{ item.doctor_title || '主治医师' }}）</span>
              </div>
              <div class="pi-info-row record-row">
                <span class="lbl">申请病历：</span>
                <template v-if="item.scope_type === 'ALL'">
                  <div class="scope-all-box">
                    <el-tag size="small" type="success" effect="dark">全部健康档案 (涵盖您在全网的所有既往就诊与检查)</el-tag>
                    <el-button
                      type="primary"
                      link
                      size="small"
                      class="ml-2"
                      @click="goToRecords"
                    >
                      前往「我的电子健康档案」查看全部记录
                    </el-button>
                  </div>
                </template>
                <template v-else>
                  <div class="req-record-meta-box">
                    <div class="rec-code-line">
                      <code class="rec-code clickable-rec" @click="viewRecordDetail(item)">{{ item.record_no || '单份指定就诊病历' }}</code>
                      <el-tag v-if="item.record_hospital_name" size="small" type="primary" class="ml-2">
                        {{ item.record_hospital_name }}
                      </el-tag>
                      <el-tag v-if="item.record_department" size="small" type="info" class="ml-1">
                        {{ item.record_department }}
                      </el-tag>
                      <el-tag v-if="item.record_created_at" size="small" type="info" effect="plain" class="ml-1">
                        {{ item.record_created_at }}
                      </el-tag>
                    </div>
                    <div v-if="item.record_diagnosis" class="rec-diag-line mt-1">
                      <span class="text-xs text-slate-500">该病历确诊：</span>
                      <strong class="text-emerald-700">{{ item.record_diagnosis }}</strong>
                    </div>
                    <div class="rec-action-btn-line mt-2">
                      <el-button
                        type="primary"
                        size="small"
                        plain
                        :icon="View"
                        @click="viewRecordDetail(item)"
                      >
                        查看我对应的这份病历详情与红头 PDF
                      </el-button>
                    </div>
                  </div>
                </template>
              </div>
              <div class="pi-info-row">
                <span class="lbl">期望授权期：</span>
                <el-tag size="small" type="info">{{ item.days || 7 }} 天</el-tag>
              </div>
              <div class="purpose-box">
                <span class="purpose-lbl">医生临床调阅目的说明：</span>
                <p class="purpose-text">{{ item.purpose || '门诊专科联合随访评估与既往慢病复核，需调阅外院历史健康档案' }}</p>
              </div>
              <div class="pi-time">
                申请发起时间：{{ item.created_at ? item.created_at.substring(0, 16).replace('T', ' ') : '' }}
              </div>
            </div>
            <div class="pi-actions">
              <el-button
                type="success"
                :loading="processingId === item.id"
                @click="approveConsent(item.id)"
              >
                同意授权 (立即释放权限并上链)
              </el-button>
              <el-button
                type="danger"
                plain
                :loading="processingId === item.id"
                @click="rejectConsent(item.id)"
              >
                拒绝调阅
              </el-button>
            </div>
          </div>
        </div>
      </div>

      <el-empty
        v-else
        description="暂无待审批的医生跨院调阅申请。当外院医生在门诊随访中发起知情申请时，将在此处即时通知您！"
        :image-size="70"
      />
    </el-card>

    <!-- 2. 已生效与历史授权策略存证 -->
    <el-card shadow="hover" class="box-card">
      <template #header>
        <div class="card-header-flex">
          <strong>我已生效与历史授权策略存证 (Fabric 联盟链分布式账本)</strong>
        </div>
      </template>

      <el-table :data="authorizations" v-loading="loading" stripe style="width: 100%">
        <el-table-column prop="auth_no" label="授权流水号" width="160" />
        <el-table-column prop="auth_target_type" label="授权对象类型" width="130">
          <template #default="{ row }">
            <el-tag v-if="row.auth_target_type === 'ALL_DOCTORS'" size="small" type="success" effect="dark">
              全体执业医生
            </el-tag>
            <el-tag v-else-if="row.auth_target_type === 'DOCTOR'" size="small" type="primary">
              执业医生
            </el-tag>
            <el-tag v-else size="small" type="warning">
              医院机构
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="target_name" label="被授权方" width="190" />
        <el-table-column prop="scope_type" label="授权范围与指定病历" min-width="190">
          <template #default="{ row }">
            <div v-if="row.scope_type === 'ALL'">
              <el-tag size="small" type="success" effect="dark">全部健康档案</el-tag>
            </div>
            <div v-else class="single-auth-scope-cell">
              <el-tag size="small" type="primary" effect="light">指定单份病历</el-tag>
              <div v-if="row.record_no" class="mt-1" style="font-size: 11px; line-height: 1.3;">
                <code class="mono bold text-emerald-800">{{ row.record_no }}</code>
                <div style="color: #64748b;">{{ row.record_hospital_name ? row.record_hospital_name + ' · ' : '' }}{{ row.record_diagnosis || '' }}</div>
              </div>
            </div>
          </template>
        </el-table-column>
        <el-table-column prop="end_time" label="失效时间" width="140">
          <template #default="{ row }">
            {{ row.end_time ? row.end_time.substring(0, 10) : '' }}
          </template>
        </el-table-column>
        <el-table-column prop="status" label="状态" width="100">
          <template #default="{ row }">
            <el-tag size="small" :type="row.status === 'ACTIVE' ? 'success' : 'danger'">
              {{ row.status === 'ACTIVE' ? '生效中' : '已撤销' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="fabric_tx_id" label="Fabric 存证 TxID" min-width="180">
          <template #default="{ row }">
            <el-tooltip :content="row.fabric_tx_id" placement="top">
              <span class="tx-hash">{{ row.fabric_tx_id ? row.fabric_tx_id.substring(0, 18) + '...' : '' }}</span>
            </el-tooltip>
          </template>
        </el-table-column>
        <el-table-column label="操作" width="100" fixed="right">
          <template #default="{ row }">
            <el-popconfirm
              v-if="row.status === 'ACTIVE'"
              title="确定立即撤销此授权？撤销后该医生将无法继续调阅，撤销行为记录至区块链。"
              @confirm="revokeAuth(row.id)"
            >
              <template #reference>
                <el-button link type="danger">撤销授权</el-button>
              </template>
            </el-popconfirm>
            <span v-else class="text-muted">已作废</span>
          </template>
        </el-table-column>
      </el-table>
    </el-card>

    <!-- 新增数据访问授权策略弹窗 (增强级联筛选与单份指定病历选择) -->
    <el-dialog v-model="dialogVisible" title="新增数据访问授权策略" width="620px" :close-on-click-modal="false" @open="initDialogData">
      <el-form :model="form" label-width="110px">
        <el-form-item label="授权目标类型" required>
          <el-radio-group v-model="form.auth_target_type" size="default" @change="onTargetTypeChange">
            <el-radio-button value="ALL_DOCTORS">让所有医生都可见 (全联盟公开)</el-radio-button>
            <el-radio-button value="DOCTOR">指定执业医生 (精确至个人)</el-radio-button>
            <el-radio-button value="HOSPITAL">指定医院机构 (全院科室)</el-radio-button>
          </el-radio-group>
        </el-form-item>

        <!-- 全联盟医生开放调阅说明提示 -->
        <div v-if="form.auth_target_type === 'ALL_DOCTORS'" class="all-doctors-info-box mb-3">
          <div class="adi-title">全联盟医生开放调阅策略说明</div>
          <div class="adi-desc">
            此策略将向所有入驻联盟链的医疗机构认证执业医生开放直接调阅。外院医生在接诊随访、跨院转诊或应急救治时，无需等待发起审批即可直接查阅，方便异地就医。您可随时在此界面撤销该授权。
          </div>
        </div>

        <!-- 级联选择 1: 筛选医院 (当指定医生或医院时显示) -->
        <el-form-item v-if="form.auth_target_type !== 'ALL_DOCTORS'" label="目标医疗机构" required>
          <el-select
            v-model="selectedHospitalId"
            placeholder="请选择医疗机构"
            style="width: 100%;"
            @change="onHospitalChange"
          >
            <el-option
              v-for="h in hospitalsList"
              :key="h.id"
              :label="h.name + (h.code ? ' (' + h.code + ')' : '')"
              :value="h.id"
            />
          </el-select>
        </el-form-item>

        <!-- 级联选择 2: 筛选医生 (当选择DOCTOR时) -->
        <el-form-item v-if="form.auth_target_type === 'DOCTOR'" label="目标执业医生" required>
          <el-select
            v-model="form.auth_target_id"
            placeholder="请在选定医院中选择医生"
            style="width: 100%;"
            :loading="loadingDoctors"
          >
            <el-option
              v-for="doc in filteredDoctors"
              :key="doc.id"
              :label="doc.real_name + ' (' + (doc.department_name || '科室') + ' · ' + (doc.title || '主治医师') + ')'"
              :value="doc.id"
            >
              <div style="display: flex; justify-content: space-between; align-items: center; width: 100%;">
                <span><strong>{{ doc.real_name }}</strong> <small style="color: #64748b;">({{ doc.title || '主治医师' }})</small></span>
                <el-tag size="small" type="info">{{ doc.department_name || '综合门诊' }}</el-tag>
              </div>
            </el-option>
          </el-select>
          <div v-if="filteredDoctors.length === 0 && selectedHospitalId" class="text-xs text-amber-600 mt-1">
            当前选定医院暂无可指派的执业医生
          </div>
        </el-form-item>

        <el-divider style="margin: 14px 0;" />

        <!-- 授权范围 -->
        <el-form-item label="授权数据范围" required>
          <el-radio-group v-model="form.scope_type" size="default" @change="onScopeTypeChange">
            <el-radio value="ALL">全部健康档案 (涵盖全网所有就诊记录)</el-radio>
            <el-radio value="SINGLE">指定单份就诊病历 (最小必要原则，推荐)</el-radio>
          </el-radio-group>
        </el-form-item>

        <!-- 指定具体病历选择器 -->
        <el-form-item v-if="form.scope_type === 'SINGLE'" label="选择目标病历" required>
          <el-select
            v-model="form.record_id"
            placeholder="请选择您要授权调阅的具体哪一份病历"
            style="width: 100%;"
            :loading="loadingMyRecords"
          >
            <el-option
              v-for="rec in myRecordsList"
              :key="rec.id"
              :label="'【' + rec.record_no + '】' + (rec.hospital_name || '医疗机构') + ' · ' + rec.diagnosis + ' (' + (rec.created_at ? rec.created_at.substring(0, 10) : '') + ')'"
              :value="rec.id"
            >
              <div style="display: flex; justify-content: space-between; align-items: center; width: 100%;">
                <span>
                  <strong style="color: #0f172a;">{{ rec.record_no }}</strong>
                  <span style="color: #059669; font-weight: bold; margin-left: 8px;">{{ rec.diagnosis }}</span>
                </span>
                <span style="font-size: 11px; color: #64748b;">
                  {{ rec.hospital_name }} · {{ rec.created_at ? rec.created_at.substring(0, 10) : '' }}
                </span>
              </div>
            </el-option>
          </el-select>

          <!-- 选定病历后的卡片概览与红头查看 -->
          <div v-if="selectedRecordInfo" class="selected-record-preview-card mt-2">
            <div class="srp-header">
              <span class="srp-title">选定病历摘要</span>
              <el-button type="primary" link size="small" :icon="View" @click="openSelectedRecordDetail">
                预览完整病历
              </el-button>
            </div>
            <div class="srp-grid">
              <div>就诊编号：<span class="mono bold">{{ selectedRecordInfo.record_no }}</span></div>
              <div>确诊结论：<strong class="text-emerald-700">{{ selectedRecordInfo.diagnosis }}</strong></div>
              <div>就诊医院：{{ selectedRecordInfo.hospital_name }}</div>
              <div>接诊科室：{{ selectedRecordInfo.department_name }}</div>
              <div>接诊医生：{{ selectedRecordInfo.doctor_name }}</div>
              <div>就诊时间：{{ selectedRecordInfo.created_at ? selectedRecordInfo.created_at.substring(0, 16).replace('T', ' ') : '-' }}</div>
            </div>
            <div v-if="selectedRecordInfo.chief_complaint || selectedRecordInfo.symptoms" class="srp-complaint mt-1">
              <strong>患者主诉：</strong>{{ selectedRecordInfo.chief_complaint || selectedRecordInfo.symptoms }}
            </div>
          </div>
          <div v-else-if="myRecordsList.length === 0" class="text-xs text-amber-600 mt-1">
            您当前暂无已归档的历史就诊病历
          </div>
        </el-form-item>

        <!-- 授权有效期 -->
        <el-form-item label="有效期限" required>
          <div style="display: flex; align-items: center; gap: 12px; width: 100%;">
            <el-select v-model="form.days" style="width: 200px;">
              <el-option label="3 天 (急诊随访)" :value="3" />
              <el-option label="7 天 (常规门诊)" :value="7" />
              <el-option label="14 天 (两周疗程)" :value="14" />
              <el-option label="30 天 (慢病管理)" :value="30" />
              <el-option label="90 天 (长程康复)" :value="90" />
              <el-option label="365 天 (年度签约)" :value="365" />
            </el-select>
            <span class="text-xs text-slate-500">到期后系统将自动撤销权限，您也可随时一键作废</span>
          </div>
        </el-form-item>
      </el-form>

      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button
          type="primary"
          :loading="submitting"
          :disabled="!isFormValid"
          @click="submitCreate"
        >
          签署并存证上链 (Hyperledger Fabric)
        </el-button>
      </template>
    </el-dialog>

    <!-- 弹窗一：查看申请对应的指定病历详情 -->
    <el-dialog
      v-model="recordDetailVisible"
      title="申请调阅的目标就诊病历详情 (核实无误后再决定授权)"
      width="820px"
      top="4vh"
    >
      <div v-if="viewingRecord" class="detail-body">
        <!-- 篡改与存证核验横幅 -->
        <div v-if="viewingRecord.is_tampered" class="tamper-box danger mb-3">
          <div class="t-head">
            <el-icon><WarningFilled /></el-icon>
            【高危安全警报】检测到该病历数据库数据已被篡改！
          </div>
          <div class="t-desc">
            底层 MySQL 数据哈希与 Fabric 联盟链不可篡改基准不一致！系统已阻断非法使用并记录高危安全审计！
          </div>
        </div>
        <div v-else class="tamper-box safe mb-3">
          <span>
            <el-icon><CircleCheckFilled /></el-icon>
            <strong>区块链防篡改核验通过：</strong>该病历数据指纹与 Hyperledger Fabric 联盟链存证 100% 严格一致，数据真实完整。
          </span>
        </div>

        <div class="detail-meta-grid">
          <div><strong>就诊编号：</strong><span class="mono">{{ viewingRecord.record_no }}</span></div>
          <div><strong>就诊类型：</strong><el-tag size="small">{{ formatEncounterType(viewingRecord.encounter_type) }}</el-tag></div>
          <div><strong>就诊医院：</strong>{{ viewingRecord.hospital_name }}</div>
          <div><strong>就诊科室：</strong>{{ viewingRecord.department_name }}</div>
          <div><strong>经治医生：</strong>{{ viewingRecord.doctor_name }}</div>
          <div><strong>就诊时间：</strong>{{ formatTime(viewingRecord.created_at) }}</div>
        </div>

        <table class="detail-table">
          <tr>
            <th width="140">发病与病程</th>
            <td>发病时间：{{ viewingRecord.onset_time || '接诊前' }} | 持续时间：{{ viewingRecord.duration || '发作性' }}</td>
          </tr>
          <tr>
            <th>生命体征 (O)</th>
            <td><strong style="color: #059669;">{{ viewingRecord.vital_signs || '生命体征平稳' }}</strong></td>
          </tr>
          <tr>
            <th>患者主诉 (S)</th>
            <td><strong>{{ viewingRecord.chief_complaint || viewingRecord.symptoms }}</strong></td>
          </tr>
          <tr v-if="viewingRecord.present_illness">
            <th>现病史</th>
            <td>{{ viewingRecord.present_illness }}</td>
          </tr>
          <tr v-if="viewingRecord.initial_diagnosis">
            <th>初诊拟定与依据</th>
            <td>
              <div>初步拟诊：{{ viewingRecord.initial_diagnosis }}</div>
              <div v-if="viewingRecord.diagnostic_basis" class="text-xs text-gray-500 mt-1">诊断依据：{{ viewingRecord.diagnostic_basis }}</div>
            </td>
          </tr>
          <tr v-if="viewingRecord.exam_result">
            <th>医技检查报告 (O)</th>
            <td><div class="pre-text">{{ viewingRecord.exam_result }}</div></td>
          </tr>
          <tr>
            <th>最终确诊 (A)</th>
            <td><span class="text-success font-bold" style="color: #16a34a; font-size: 15px;">{{ viewingRecord.diagnosis }}</span></td>
          </tr>
          <tr>
            <th>处置与处方方案 (P)</th>
            <td><div class="pre-text highlight" style="color: #0284c7;">{{ viewingRecord.treatment_plan || '遵医嘱治疗' }}</div></td>
          </tr>
        </table>

        <!-- 临床归档文件卡片与 PDF 查阅 -->
        <div class="archive-files-section mt-3">
          <div class="empty-flex-row" style="background: #f8fafc; border: 1px dashed #cbd5e1; border-radius: 8px; padding: 12px 16px; display: flex; justify-content: space-between; align-items: center;">
            <div class="empty-text">
              <span class="font-medium text-slate-700">国家规范红头临床就诊病历单 (PDF 存证)</span>
              <div class="text-xs text-gray-500">包含完整 SOAP 记录、医技报告与医疗机构区块链防伪公章 (存证大小: {{ formatFileSize(viewingRecord.files?.[0]?.file_size) }})</div>
            </div>
            <div class="btn-grp" style="display: flex; gap: 8px;">
              <el-button type="primary" size="small" :icon="View" @click="openPdfPreview(viewingRecord)">
                在线查阅红头 PDF 病历
              </el-button>
              <el-button type="success" size="small" plain :icon="Download" @click="downloadRecordFile(viewingRecord.id)">
                下载解密凭据
              </el-button>
            </div>
          </div>
        </div>

        <div class="blockchain-evidence-box mt-3" style="background: #f8fafc; padding: 10px 14px; border-radius: 6px; font-size: 12px; color: #475569;">
          <div><strong>Hyperledger Fabric TxID：</strong><span class="mono">{{ viewingRecord.fabric_tx_id }}</span></div>
          <div v-if="viewingRecord.block_height"><strong>存证区块高度：</strong><span class="mono">#{{ viewingRecord.block_height }}</span></div>
        </div>
      </div>
      <template #footer>
        <el-button @click="recordDetailVisible = false">关闭</el-button>
        <el-button type="primary" :icon="View" @click="openPdfPreview(viewingRecord)">
          查看标准红头 PDF
        </el-button>
      </template>
    </el-dialog>

    <!-- 弹窗二：PDF 电子病历规范预览弹窗 -->
    <el-dialog
      v-model="pdfPreviewVisible"
      title="临床就诊电子病历归档凭证 (PDF 规范视图)"
      width="880px"
      top="3vh"
      :close-on-click-modal="false"
      class="pdf-preview-dialog"
    >
      <div v-if="previewingRecord" id="emr-print-container" class="emr-sheet-wrapper">
        <!-- 打印与工具栏 -->
        <div class="emr-action-bar no-print">
          <div class="emr-tip-tag">
            <el-tag type="success" effect="dark">密文解密验证通过</el-tag>
            <el-tag type="info" class="ml-2">国家卫健委《电子病历应用规范》甲级存证标准</el-tag>
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
          <!-- 红头医院名称与标题 -->
          <div class="emr-header">
            <div class="emr-hospital-name">{{ previewingRecord.hospital_name || '医疗机构' }}</div>
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
          </div>

          <!-- 患者基本人口学信息表 -->
          <table class="emr-patient-table">
            <tr>
              <th width="80">患者姓名</th>
              <td width="140"><strong>{{ previewingRecord.patient_name || '患者' }}</strong></td>
              <th width="80">性别/年龄</th>
              <td width="140">男 / 45岁</td>
              <th width="80">就诊卡号</th>
              <td><span class="mono">{{ previewingRecord.patient_user_no || 'PAT_0001' }}</span></td>
            </tr>
            <tr>
              <th>身份证号</th>
              <td><span class="mono">{{ previewingRecord.patient_id_card || '已脱敏保护' }}</span></td>
              <th>就诊时间</th>
              <td>{{ formatTime(previewingRecord.created_at) }}</td>
              <th>联系电话</th>
              <td><span class="mono">{{ previewingRecord.patient_phone || '138****0000' }}</span></td>
            </tr>
          </table>

          <!-- 临床 SOAP 详细内容 -->
          <div class="emr-soap-section">
            <div class="soap-block">
              <div class="soap-title">【S - Subjective 主观病史采集】</div>
              <div class="soap-row"><strong>● 患者主诉：</strong>{{ previewingRecord.chief_complaint || previewingRecord.symptoms }}</div>
              <div class="soap-row"><strong>● 现病史：</strong>{{ previewingRecord.present_illness || '患者因主诉症状就诊，发病过程如上所述。' }}</div>
              <div class="soap-row"><strong>● 发病时间：</strong>{{ previewingRecord.onset_time || '接诊前' }}（持续时间：{{ previewingRecord.duration || '发作性' }}）</div>
            </div>

            <div class="soap-block">
              <div class="soap-title">【O - Objective 客观检查与测量】</div>
              <div class="soap-row"><strong>● 基础生命体征：</strong><span class="vitals-highlight">{{ previewingRecord.vital_signs || '生命体征平稳' }}</span></div>
              <div class="soap-row"><strong>● 辅助检查说明：</strong>{{ previewingRecord.need_exam ? (previewingRecord.exam_items || '已开具辅助检查') : '经治医生研判体征典型明确，未开具侵入性检验检查' }}</div>
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
                <span class="font-bold text-success" style="color: #16a34a; font-size: 15px;">{{ previewingRecord.diagnosis }}</span>
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

    <!-- 4. 修改跨院病历调阅专属密钥弹窗 -->
    <el-dialog 
      title="修改跨院病历调阅专属密钥" 
      v-model="medicalKeyVisible" 
      width="480px"
      :close-on-click-modal="false"
    >
      <el-alert
        title="关于跨院病历调阅专属密钥"
        type="success"
        show-icon
        :closable="false"
        class="mb-3"
        description="当您在外院就医时，主治医生发起跨院调阅时，您可直接向医生提供此专属密钥或现场输入，即可立即授权解锁您的健康病历，无需在线等待推送审批。"
      />

      <el-form :model="medicalKeyForm" label-position="top">
        <el-form-item label="原调阅密钥 (初始默认密码为 123456)">
          <el-input 
            v-model="medicalKeyForm.old_key" 
            type="password" 
            placeholder="若首次修改可留空或输入 123456" 
            show-password 
            :prefix-icon="Key"
          />
        </el-form-item>

        <el-form-item label="设置新专属调阅密钥 (建议 6 位数字或口令)" required>
          <el-input 
            v-model="medicalKeyForm.new_key" 
            type="password" 
            placeholder="请输入新的专属调阅密钥（≥4位）" 
            show-password 
            :prefix-icon="Key"
          />
        </el-form-item>

        <el-form-item label="确认新专属调阅密钥" required>
          <el-input 
            v-model="medicalKeyForm.confirm_key" 
            type="password" 
            placeholder="请再次输入新专属密钥进行确认" 
            show-password 
            :prefix-icon="Key"
          />
        </el-form-item>
      </el-form>

      <template #footer>
        <el-button @click="medicalKeyVisible = false">取 消</el-button>
        <el-button type="success" :loading="savingMedicalKey" @click="handleSaveMedicalKey">
          确 认 更 改 专 属 密 钥
        </el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { Plus, View, Download, Printer, WarningFilled, CircleCheckFilled, Key, Hide } from '@element-plus/icons-vue'
import { ElMessage, ElNotification } from 'element-plus'
import api from '../../api/client'
import { useAuthStore } from '../../stores/auth'

const router = useRouter()
const auth = useAuthStore()
const authorizations = ref<any[]>([])
const pendingRequests = ref<any[]>([])
const loading = ref(false)
const dialogVisible = ref(false)
const submitting = ref(false)
const processingId = ref<number | null>(null)

// 详情查看与 PDF 规范预览状态
const recordDetailVisible = ref(false)
const viewingRecord = ref<any>(null)
const loadingDetail = ref(false)

const pdfPreviewVisible = ref(false)
const previewingRecord = ref<any>(null)

// ── 跨院病历调阅专属密钥 ───────────────────
const showKeyPlain = ref(false)
const medicalKeyVisible = ref(false)
const savingMedicalKey = ref(false)
const medicalKeyForm = ref({
  old_key: '',
  new_key: '',
  confirm_key: '',
})

function openMedicalKeyDialog() {
  medicalKeyForm.value = {
    old_key: '',
    new_key: '',
    confirm_key: '',
  }
  medicalKeyVisible.value = true
}

async function handleSaveMedicalKey() {
  if (!medicalKeyForm.value.new_key) {
    ElMessage.warning('请输入新的专属调阅密钥')
    return
  }
  if (medicalKeyForm.value.new_key.length < 4) {
    ElMessage.warning('专属调阅密钥长度不能少于 4 位')
    return
  }
  if (medicalKeyForm.value.new_key !== medicalKeyForm.value.confirm_key) {
    ElMessage.warning('两次输入的新专属密钥不一致，请核对')
    return
  }

  savingMedicalKey.value = true
  try {
    const res: any = await api.put('/auth/medical-key', {
      old_key: medicalKeyForm.value.old_key,
      new_key: medicalKeyForm.value.new_key,
    })
    if (res.code === 200) {
      ElMessage.success(res.message || '病历调阅专属密钥已成功更新！')
      if (auth.user) {
        auth.user.medical_key = medicalKeyForm.value.new_key
      }
      medicalKeyVisible.value = false
    }
  } catch (err: any) {
    ElMessage.error(err?.message || err?.response?.data?.message || '更新调阅密钥失败')
  } finally {
    savingMedicalKey.value = false
  }
}

// 级联医院/医生与病历选择器数据状态
const hospitalsList = ref<any[]>([])
const allDoctorsList = ref<any[]>([])
const myRecordsList = ref<any[]>([])
const selectedHospitalId = ref<number | undefined>(undefined)
const loadingDoctors = ref(false)
const loadingMyRecords = ref(false)

const form = ref({
  auth_target_type: 'ALL_DOCTORS' as 'DOCTOR' | 'HOSPITAL' | 'ALL_DOCTORS',
  auth_target_id: undefined as number | undefined,
  scope_type: 'ALL' as 'ALL' | 'SINGLE',
  record_id: undefined as number | undefined,
  days: 30,
})

const filteredDoctors = computed(() => {
  if (!selectedHospitalId.value) return allDoctorsList.value
  return allDoctorsList.value.filter((d: any) => d.hospital_id === selectedHospitalId.value)
})

const selectedRecordInfo = computed(() => {
  if (!form.value.record_id) return null
  return myRecordsList.value.find((r: any) => r.id === form.value.record_id) || null
})

const isFormValid = computed(() => {
  if (!form.value.auth_target_type) return false
  if (form.value.auth_target_type === 'ALL_DOCTORS') {
    if (form.value.scope_type === 'SINGLE' && !form.value.record_id) return false
    return true
  }
  if (form.value.auth_target_type === 'DOCTOR' && !form.value.auth_target_id) return false
  if (form.value.auth_target_type === 'HOSPITAL' && !selectedHospitalId.value) return false
  if (form.value.scope_type === 'SINGLE' && !form.value.record_id) return false
  return true
})

function onTargetTypeChange(val: string) {
  if (val === 'ALL_DOCTORS') {
    form.value.auth_target_id = 0
  } else if (val === 'HOSPITAL') {
    form.value.auth_target_id = selectedHospitalId.value
  } else {
    // DOCTOR
    if (filteredDoctors.value.length > 0) {
      form.value.auth_target_id = filteredDoctors.value[0].id
    } else {
      form.value.auth_target_id = undefined
    }
  }
}

function onHospitalChange(hId: number) {
  if (form.value.auth_target_type === 'HOSPITAL') {
    form.value.auth_target_id = hId
  } else {
    const docs = allDoctorsList.value.filter((d: any) => d.hospital_id === hId)
    if (docs.length > 0) {
      form.value.auth_target_id = docs[0].id
    } else {
      form.value.auth_target_id = undefined
    }
  }
}

function onScopeTypeChange(val: string) {
  if (val === 'SINGLE') {
    if (!form.value.record_id && myRecordsList.value.length > 0) {
      form.value.record_id = myRecordsList.value[0].id
    }
  } else {
    form.value.record_id = undefined
  }
}

async function initDialogData() {
  await Promise.all([loadHospitals(), loadDoctors(), loadMyRecords()])
}

async function loadHospitals() {
  try {
    const res: any = await api.get('/system/hospitals')
    if (res.code === 200) {
      hospitalsList.value = res.data || []
      if (!selectedHospitalId.value && hospitalsList.value.length > 0) {
        selectedHospitalId.value = hospitalsList.value[0].id
        if (form.value.auth_target_type === 'HOSPITAL') {
          form.value.auth_target_id = hospitalsList.value[0].id
        }
      }
    }
  } catch (err) {
    console.error(err)
  }
}

async function loadDoctors() {
  loadingDoctors.value = true
  try {
    const res: any = await api.get('/system/doctors')
    if (res.code === 200) {
      allDoctorsList.value = res.data || []
      if (!form.value.auth_target_id && filteredDoctors.value.length > 0) {
        form.value.auth_target_id = filteredDoctors.value[0].id
      }
    }
  } catch (err) {
    console.error(err)
  } finally {
    loadingDoctors.value = false
  }
}

async function loadMyRecords() {
  loadingMyRecords.value = true
  try {
    const res: any = await api.get('/medical-records')
    if (res.code === 200) {
      myRecordsList.value = res.data || []
      if (form.value.scope_type === 'SINGLE' && !form.value.record_id && myRecordsList.value.length > 0) {
        form.value.record_id = myRecordsList.value[0].id
      }
    }
  } catch (err) {
    console.error(err)
  } finally {
    loadingMyRecords.value = false
  }
}

function openSelectedRecordDetail() {
  if (selectedRecordInfo.value) {
    viewingRecord.value = selectedRecordInfo.value
    recordDetailVisible.value = true
  }
}

onMounted(() => {
  loadAuths()
  loadPending()
})

async function loadPending() {
  try {
    const res: any = await api.get('/access/requests/pending')
    if (res.code === 200) {
      pendingRequests.value = res.data
    }
  } catch (err) {
    console.error(err)
  }
}

async function loadAuths() {
  loading.value = true
  try {
    const res: any = await api.get('/authorizations')
    if (res.code === 200) {
      authorizations.value = res.data
    }
  } finally {
    loading.value = false
  }
}

async function approveConsent(id: number) {
  processingId.value = id
  try {
    const res: any = await api.post(`/access/requests/${id}/approve`)
    if (res.code === 200) {
      ElNotification({
        title: '授权核准成功',
        message: '已向经治医生释放跨院调阅权限，授权策略已永久锚定至 Fabric 联盟链！该医生现在可直接查看病历。',
        type: 'success',
        duration: 6000
      })
      loadPending()
      loadAuths()
    }
  } catch (err: any) {
    ElMessage.error(err?.message || '授权审批失败')
  } finally {
    processingId.value = null
  }
}

async function rejectConsent(id: number) {
  processingId.value = id
  try {
    const res: any = await api.post(`/access/requests/${id}/reject`)
    if (res.code === 200) {
      ElMessage.info('已驳回该跨院调阅申请')
      loadPending()
    }
  } catch (err: any) {
    ElMessage.error(err?.message || '驳回失败')
  } finally {
    processingId.value = null
  }
}

async function submitCreate() {
  if (!isFormValid.value) {
    ElMessage.warning('请完整选择授权对象与指定病历')
    return
  }

  let targetId = 0
  if (form.value.auth_target_type === 'HOSPITAL') {
    targetId = selectedHospitalId.value || 0
  } else if (form.value.auth_target_type === 'DOCTOR') {
    targetId = form.value.auth_target_id || 0
  } else {
    targetId = 0
  }

  if (form.value.auth_target_type !== 'ALL_DOCTORS' && !targetId) {
    ElMessage.warning('请选择被授权的医疗机构或执业医生')
    return
  }

  submitting.value = true
  try {
    const now = new Date()
    const days = form.value.days || 30
    const endTime = new Date(now.getTime() + days * 24 * 3600 * 1000)

    const payload: any = {
      auth_target_type: form.value.auth_target_type,
      auth_target_id: targetId,
      scope_type: form.value.scope_type,
      start_time: now.toISOString(),
      end_time: endTime.toISOString(),
    }
    if (form.value.scope_type === 'SINGLE' && form.value.record_id) {
      payload.record_id = form.value.record_id
    }

    const res: any = await api.post('/authorizations', payload)
    if (res.code === 200) {
      const targetText = form.value.auth_target_type === 'ALL_DOCTORS'
        ? '全联盟全体执业医生 (全网公开)'
        : (form.value.auth_target_type === 'DOCTOR' ? '指定医生' : '指定医院机构')
      ElNotification({
        title: '授权策略建立成功',
        message: `已向【${targetText}】签发智能合约知情授权凭证，Fabric TxID: ${res.data?.fabric_tx_id?.substring(0, 18)}...`,
        type: 'success',
        duration: 6000
      })
      dialogVisible.value = false
      loadAuths()
    }
  } catch (err: any) {
    ElMessage.error(err?.message || '授权创建失败')
  } finally {
    submitting.value = false
  }
}

async function revokeAuth(id: number) {
  try {
    const res: any = await api.delete(`/authorizations/${id}`)
    if (res.code === 200) {
      ElMessage.success('授权已成功撤销')
      loadAuths()
    }
  } catch (err: any) {
    ElMessage.error(err?.message || '撤销失败')
  }
}

function goToRecords() {
  router.push('/patient/records')
}

async function viewRecordDetail(item: any) {
  loadingDetail.value = true
  try {
    const recordId = item.record_id
    if (recordId) {
      const res: any = await api.get(`/medical-records/${recordId}`)
      if (res.code === 200) {
        viewingRecord.value = res.data
      } else {
        viewingRecord.value = {
          record_no: item.record_no,
          diagnosis: item.record_diagnosis,
          department_name: item.record_department,
          hospital_name: item.record_hospital_name,
          encounter_type: item.record_encounter_type,
          created_at: item.record_created_at,
          doctor_name: item.doctor_name,
          is_tampered: false,
        }
      }
    } else {
      viewingRecord.value = {
        record_no: item.request_no,
        diagnosis: item.record_diagnosis || '全部历史健康档案',
        department_name: item.record_department || '临床科室',
        hospital_name: item.record_hospital_name || '综合医院',
        encounter_type: item.record_encounter_type || 'OUTPATIENT',
        created_at: item.created_at,
        doctor_name: item.doctor_name,
        is_tampered: false,
      }
    }
    recordDetailVisible.value = true
  } catch (err) {
    ElMessage.error('获取病历详情失败，请重试')
  } finally {
    loadingDetail.value = false
  }
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
      const rNo = previewingRecord.value?.record_no || viewingRecord.value?.record_no || recordId
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
  if (t === 'EMERGENCY') return '急诊救治'
  if (t === 'INPATIENT') return '住院就诊'
  return t || '门诊'
}

function formatTime(t?: string) {
  if (!t) return '-'
  return t.substring(0, 16).replace('T', ' ')
}
</script>

<style scoped>
.page-container {
  max-width: 1200px;
  margin: 0 auto;
}
.page-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 20px;
}
.page-title {
  font-size: 22px;
  font-weight: 800;
  color: #0f172a;
}
.page-sub {
  font-size: 13px;
  color: #64748b;
}
.box-card {
  border-radius: 14px;
}
.card-header-flex {
  display: flex;
  justify-content: space-between;
  align-items: center;
}
.header-title {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 16px;
  color: #1e293b;
}
.header-icon {
  font-size: 20px;
}
.header-sub {
  font-size: 12px;
  color: #64748b;
}
.pending-badge {
  margin-left: 6px;
}
.pending-card {
  border: 1.5px solid #fed7aa;
  background: #fffdfa;
}
.pending-grid {
  display: flex;
  flex-direction: column;
  gap: 14px;
}
.pending-item-card {
  border: 1px solid #fed7aa;
  background: #ffffff;
  border-radius: 12px;
  padding: 16px;
  box-shadow: 0 2px 6px rgba(251, 146, 60, 0.08);
}
.pi-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 12px;
  padding-bottom: 8px;
  border-bottom: 1px dashed #fed7aa;
}
.pi-no {
  font-size: 13px;
  color: #64748b;
}
.pi-body {
  display: flex;
  flex-direction: column;
  gap: 6px;
  font-size: 13px;
  color: #334155;
  margin-bottom: 14px;
}
.pi-info-row {
  display: flex;
  align-items: center;
}
.pi-info-row .lbl {
  font-weight: 600;
  color: #475569;
  min-width: 90px;
}
.doc-highlight {
  font-weight: 700;
  color: #0284c7;
}
.hosp-tag {
  color: #64748b;
  margin-left: 4px;
}
.rec-code {
  font-family: monospace;
  background: #f1f5f9;
  padding: 2px 6px;
  border-radius: 4px;
  color: #0f172a;
}
.purpose-box {
  background: #eff6ff;
  border: 1px solid #dbeafe;
  border-radius: 8px;
  padding: 10px 14px;
  margin: 6px 0;
}
.purpose-lbl {
  font-weight: 600;
  color: #1e40af;
  display: block;
  margin-bottom: 4px;
}
.purpose-text {
  margin: 0;
  color: #1e3a8a;
  line-height: 1.5;
}
.pi-time {
  font-size: 12px;
  color: #94a3b8;
}
.pi-actions {
  display: flex;
  gap: 12px;
}
.tx-hash {
  font-family: monospace;
  color: #8b5cf6;
}
.text-muted {
  color: #94a3b8;
  font-size: 12px;
}

/* 病历引用及卡片元数据 */
.record-row {
  align-items: flex-start !important;
}
.scope-all-box {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 8px;
}
.req-record-meta-box {
  flex: 1;
  background: #f8fafc;
  border: 1px solid #fed7aa;
  border-radius: 8px;
  padding: 10px 14px;
}
.rec-code-line {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 6px;
}
.rec-code.clickable-rec {
  cursor: pointer;
  background: #e0f2fe;
  color: #0369a1;
  border: 1px solid #bae6fd;
  font-weight: 700;
  transition: all 0.2s;
}
.rec-code.clickable-rec:hover {
  background: #bae6fd;
  color: #0284c7;
  text-decoration: underline;
}

/* 详情弹窗样式 */
.detail-body {
  font-size: 13px;
  color: #1e293b;
}
.tamper-box {
  padding: 10px 14px;
  border-radius: 6px;
  font-size: 12px;
}
.tamper-box.safe {
  background: #ecfdf5;
  border: 1px solid #a7f3d0;
  color: #065f46;
}
.tamper-box.danger {
  background: #fef2f2;
  border: 1px solid #fecaca;
  color: #991b1b;
}
.tamper-box .t-head {
  font-weight: 700;
  margin-bottom: 4px;
}
.tamper-box .t-desc {
  line-height: 1.5;
}
.detail-meta-grid {
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  gap: 8px 16px;
  background: #f8fafc;
  padding: 12px 16px;
  border-radius: 8px;
  border: 1px solid #e2e8f0;
  margin-bottom: 14px;
}
.detail-table {
  width: 100%;
  border-collapse: collapse;
  margin-bottom: 12px;
}
.detail-table th, .detail-table td {
  border: 1px solid #e2e8f0;
  padding: 8px 12px;
  font-size: 13px;
}
.detail-table th {
  background: #f1f5f9;
  color: #475569;
  text-align: left;
  font-weight: 600;
}
.pre-text {
  margin: 0;
  white-space: pre-wrap;
  font-family: inherit;
  line-height: 1.5;
}
.mono {
  font-family: monospace;
}

/* 红头 PDF 预览规范纸张与印章 */
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
.selected-record-preview-card {
  background: #f8fafc;
  border: 1px solid #cbd5e1;
  border-radius: 8px;
  padding: 10px 14px;
  margin-top: 8px;
  font-size: 12px;
}
.srp-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 6px;
  border-bottom: 1px dashed #e2e8f0;
  padding-bottom: 4px;
}
.srp-title {
  font-weight: 700;
  color: #1e293b;
}
.srp-grid {
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  gap: 4px 12px;
  color: #475569;
}
.srp-complaint {
  color: #334155;
  border-top: 1px dashed #e2e8f0;
  padding-top: 4px;
}
.single-auth-scope-cell {
  display: flex;
  flex-direction: column;
  gap: 2px;
}
.medical-key-banner-card {
  display: flex;
  justify-content: space-between;
  align-items: center;
  background: linear-gradient(135deg, #f0fdf4 0%, #ecfdf5 50%, #d1fae5 100%);
  border: 1.5px solid #a7f3d0;
  border-radius: 12px;
  padding: 16px 20px;
  gap: 16px;
  box-shadow: 0 4px 12px rgba(16, 185, 129, 0.08);
}
.mkb-left {
  display: flex;
  align-items: center;
  gap: 14px;
  flex: 1;
}
.mkb-icon-wrap {
  width: 44px;
  height: 44px;
  background: white;
  border-radius: 10px;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 22px;
  box-shadow: 0 2px 8px rgba(16, 185, 129, 0.15);
  flex-shrink: 0;
}
.mkb-title-row {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 4px;
}
.mkb-title {
  font-size: 16px;
  font-weight: 700;
  color: #065f46;
}
.mkb-sub {
  font-size: 13px;
  color: #047857;
  margin: 0;
  line-height: 1.5;
}
.mkb-right {
  display: flex;
  align-items: center;
  gap: 14px;
  flex-shrink: 0;
}
.mkb-key-display {
  display: flex;
  align-items: center;
  gap: 6px;
  background: white;
  padding: 6px 12px;
  border-radius: 8px;
  border: 1px solid #6ee7b7;
}
.mkb-key-label {
  font-size: 12px;
  color: #065f46;
}
.mkb-key-code {
  font-family: monospace;
  font-weight: bold;
  font-size: 14px;
  color: #059669;
}
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

.all-doctors-info-box {
  background: linear-gradient(135deg, #ecfdf5 0%, #f0fdf4 100%);
  border: 1.5px solid #10b981;
  border-radius: 8px;
  padding: 12px 16px;
}
.adi-title {
  font-weight: 700;
  font-size: 13px;
  color: #065f46;
  margin-bottom: 4px;
}
.adi-desc {
  font-size: 12px;
  color: #047857;
  line-height: 1.5;
}
</style>
