<template>
  <div class="page-container">
    <!-- 1. 规范页面顶栏 -->
    <div class="page-header">
      <div class="header-left">
        <div class="title-row">
          <h2 class="page-title">我的电子健康档案</h2>
          <el-tag size="small" type="success" effect="plain" class="page-header-tag">
            区块链全流程存证
          </el-tag>
        </div>
        <p class="page-sub">查看您在联盟链医疗机构建立的所有门诊病历、结构化临床问诊记录、检验报告与影像数据</p>
      </div>
      <div class="header-actions">
        <el-button type="success" plain :icon="Share" @click="openBatchPolicyDialog">
          批量设置共享范围
        </el-button>
        <el-button type="primary" :icon="Key" @click="$router.push('/patient/auth')">
          知情授权管理中心
        </el-button>
      </div>
    </div>

    <!-- 2. 患者自主隐私与数据主权卡片 -->
    <div class="privacy-sovereignty-card mb-4">
      <div class="psc-left">
        <div class="psc-icon-wrap">
          <el-icon :size="22"><CircleCheckFilled /></el-icon>
        </div>
        <div class="psc-content">
          <div class="psc-title-line">
            <span class="psc-title">患者自主数据主权：您对每一份就诊病历均享有 100% 的知情与共享控制权</span>
            <el-tag size="small" type="success" effect="dark" class="psc-badge">Fabric 智能合约存证</el-tag>
          </div>
          <p class="psc-desc">
            系统默认遵循隐私最小化保护原则。若您需要跨院转诊、异地就医或联合会诊，可灵活配置病历共享权限：<strong>全面公开、指定某家医院全体医生或指定单个医生专属调阅</strong>。所有授权策略均实时固化至 Fabric 联盟链分布式账本，随时支持一键恢复私密受控。
          </p>
        </div>
      </div>
      <div class="psc-right">
        <div class="psc-metric">
          <span class="metric-val">{{ records.length }}</span>
          <span class="metric-lbl">累计病历</span>
        </div>
        <div class="psc-metric-sep"></div>
        <div class="psc-metric">
          <span class="metric-val text-emerald-600">{{ publicRecordCount }}</span>
          <span class="metric-lbl">全网公开</span>
        </div>
        <div class="psc-metric-sep"></div>
        <div class="psc-metric">
          <span class="metric-val text-blue-600">{{ targetedRecordCount }}</span>
          <span class="metric-lbl">定向授权</span>
        </div>
      </div>
    </div>

    <!-- 3. 时间线就诊记录列表 -->
    <div class="timeline-wrapper">
      <el-timeline v-if="records.length">
        <el-timeline-item
          v-for="rec in records"
          :key="rec.id"
          :timestamp="rec.created_at ? rec.created_at.substring(0, 16).replace('T', ' ') : ''"
          placement="top"
          type="primary"
          class="med-timeline-node"
        >
          <div class="record-glass-card">
            <!-- 卡片头部：机构、医生、标签群 -->
            <div class="rg-head">
              <div class="rg-head-left">
                <div class="hosp-pill">
                  <el-icon class="mr-1 text-emerald-600"><OfficeBuilding /></el-icon>
                  <span>{{ rec.hospital_name || '联盟医疗中心' }}</span>
                </div>
                <span class="h-sep">·</span>
                <div class="doc-pill">
                  <el-icon class="mr-1 text-slate-500"><User /></el-icon>
                  <span>{{ formatDoctorTitle(rec.doctor_name) }}</span>
                </div>
                <el-tag size="small" effect="plain" type="info" class="encounter-type-tag">
                  {{ rec.encounter_type === 'EMERGENCY' ? '急诊就诊' : '普通门诊' }}
                </el-tag>
              </div>

              <div class="rg-head-right">
                <!-- 访问权公开状态徽章 -->
                <el-tag
                  v-if="rec.sharing_scope === 'ALL_DOCTORS'"
                  size="small"
                  type="success"
                  effect="dark"
                  class="scope-badge"
                >
                  <el-icon class="mr-1"><Share /></el-icon>
                  全体医生可见
                </el-tag>
                <el-tag
                  v-else-if="rec.sharing_scope === 'HOSPITAL'"
                  size="small"
                  type="primary"
                  effect="dark"
                  class="scope-badge"
                >
                  <el-icon class="mr-1"><OfficeBuilding /></el-icon>
                  {{ rec.sharing_summary || (rec.active_auth_target_name ? `仅限【${rec.active_auth_target_name}】` : '指定医院可见') }}
                </el-tag>
                <el-tag
                  v-else-if="rec.sharing_scope === 'DOCTOR'"
                  size="small"
                  type="warning"
                  effect="dark"
                  class="scope-badge"
                >
                  <el-icon class="mr-1"><User /></el-icon>
                  {{ rec.sharing_summary || (rec.active_auth_target_name ? `仅限【${rec.active_auth_target_name} 医生】` : '指定医生可见') }}
                </el-tag>
                <el-tag
                  v-else-if="rec.sharing_scope === 'AUTHORIZED'"
                  size="small"
                  type="primary"
                  effect="light"
                  class="scope-badge"
                >
                  <el-icon class="mr-1"><Check /></el-icon>
                  {{ rec.sharing_summary || '已定向授权' }}
                </el-tag>
                <el-tag
                  v-else
                  size="small"
                  type="info"
                  effect="plain"
                  class="scope-badge"
                >
                  <el-icon class="mr-1"><Lock /></el-icon>
                  私密受控 (需授权)
                </el-tag>

                <!-- 防篡改核验 -->
                <el-tag v-if="rec.is_tampered" size="small" type="danger" effect="dark" class="tamper-tag-glow">
                  存在篡改风险
                </el-tag>
                <el-tag v-else size="small" type="success" effect="light">
                  链上核验通过
                </el-tag>

                <!-- 数据类型与区块高度 -->
                <el-tag size="small" class="data-type-pill">{{ rec.data_type || 'EMR' }}</el-tag>
                <span class="block-pill">
                  <el-icon><Connection /></el-icon>
                  Fabric #{{ rec.block_height }}
                </span>
              </div>
            </div>

            <!-- 结构化关键信息胶囊卡 -->
            <div v-if="rec.onset_time || rec.symptoms || rec.treatment_plan || rec.vital_signs" class="clinical-structured-capsule">
              <!-- 生命体征指标栏 -->
              <div v-if="rec.vital_signs" class="vitals-row">
                <span class="capsule-label">生命体征：</span>
                <span class="vitals-val">{{ rec.vital_signs }}</span>
              </div>

              <div class="capsule-grid">
                <div class="capsule-item">
                  <span class="capsule-label">发病时间：</span>
                  <span class="capsule-val">{{ rec.onset_time || '接诊前' }}（{{ rec.duration || '急性起病' }}）</span>
                </div>
                <div v-if="rec.etiology" class="capsule-item">
                  <span class="capsule-label">病因与诱因：</span>
                  <span class="etiology-chip">{{ rec.etiology }}</span>
                </div>
              </div>

              <div v-if="rec.symptoms" class="capsule-item full-width">
                <span class="capsule-label">核心临床症状：</span>
                <span class="capsule-val">{{ rec.symptoms }}</span>
              </div>

              <div v-if="rec.treatment_plan" class="capsule-item full-width treatment-plan-row">
                <span class="capsule-label">医生处置建议：</span>
                <span class="plan-text">{{ rec.treatment_plan }}</span>
              </div>
            </div>

            <!-- 诊断结论高亮栏 -->
            <div class="diagnosis-callout">
              <span class="diag-label">病历诊断小结：</span>
              <span class="diag-value">{{ rec.diagnosis || '记录中' }}</span>
            </div>

            <!-- 底栏元数据与权限操作 -->
            <div class="rg-footer">
              <div class="meta-ids">
                <span class="meta-item">
                  <span class="meta-lbl">就诊单号:</span>
                  <code class="meta-code">{{ rec.record_no }}</code>
                </span>
                <span class="meta-dot">|</span>
                <span class="meta-item">
                  <span class="meta-lbl">存证 TxID:</span>
                  <el-tooltip :content="rec.fabric_tx_id || '-'" placement="top">
                    <code class="meta-code tx-hash" @click="copyTxId(rec.fabric_tx_id)">
                      {{ formatTxId(rec.fabric_tx_id) }}
                      <el-icon class="copy-icon"><CopyDocument /></el-icon>
                    </code>
                  </el-tooltip>
                </span>
              </div>

              <div class="meta-actions">
                <el-button
                  size="small"
                  :type="rec.sharing_scope === 'ALL_DOCTORS' ? 'success' : (rec.sharing_scope === 'HOSPITAL' || rec.sharing_scope === 'DOCTOR' || rec.sharing_scope === 'AUTHORIZED' ? 'primary' : 'info')"
                  plain
                  @click="openAccessPolicyDialog(rec)"
                >
                  <el-icon class="mr-1"><Lock /></el-icon>
                  访问权限设置
                  <span v-if="rec.sharing_scope === 'ALL_DOCTORS'" class="policy-active-text">(全网公开)</span>
                  <span v-else-if="rec.sharing_scope === 'HOSPITAL'" class="policy-active-text">(指定医院)</span>
                  <span v-else-if="rec.sharing_scope === 'DOCTOR'" class="policy-active-text">(指定医生)</span>
                </el-button>
                <el-button link type="primary" size="small" @click="viewFullDetail(rec)">
                  查看完整病历卡片
                  <el-icon class="ml-1"><ArrowRight /></el-icon>
                </el-button>
              </div>
            </div>
          </div>
        </el-timeline-item>
      </el-timeline>
      <el-empty v-else description="暂无就诊记录" />
    </div>

    <!-- 完整病历大表格对话框 -->
    <el-dialog v-model="detailVisible" title="个人就诊档案（结构化临床大表单视图）" width="760px">
      <div v-if="selectedRec" class="patient-detail-box">
        <!-- 篡改告警横幅 -->
        <div v-if="selectedRec.is_tampered" class="tamper-warning-box">
          <div class="tw-head">
            <el-icon class="tw-icon"><WarningFilled /></el-icon>
            <span class="tw-title">【高危安全警报】检测到您的病历已被数据库非法篡改！</span>
          </div>
          <div class="tw-body">
            系统检测到底层 MySQL 数据库中的临床数据与 Fabric 联盟链上不可篡改的存证指纹<strong>不匹配</strong>！该病历中的用药方案、药物过敏史或核心诊断已被黑客攻击篡改，<strong>临床严禁直接采信该病历！</strong>系统已阻断非法使用并记录高危安全审计！
          </div>
          <div class="tw-hashes">
            <div class="tw-h-item danger">
              <span class="lbl">数据库当前计算哈希 (Current Hash)：</span>
              <code>{{ selectedRec.current_hash }}</code>
            </div>
            <div class="tw-h-item chain">
              <span class="lbl">区块链不可篡改基准 (Chain Hash)：</span>
              <code>{{ selectedRec.chain_hash }}</code>
            </div>
          </div>
        </div>

        <div v-else class="verified-safe-box">
          <el-icon><CircleCheckFilled /></el-icon>
          <span><strong>区块链防篡改校验通过：</strong>该病历所有临床症状、用药方案与影像数据指纹均与 Fabric 联盟链上固化存证 100% 严格一致，数据真实完整，未遭任何篡改。</span>
        </div>

        <!-- 详情弹窗中的访问权限配置条 -->
        <div class="detail-policy-strip">
          <div class="dps-left">
            <span class="dps-label">当前调阅控制：</span>
            <el-tag
              :type="selectedRec.sharing_scope === 'ALL_DOCTORS' ? 'success' : (selectedRec.sharing_scope === 'HOSPITAL' || selectedRec.sharing_scope === 'DOCTOR' || selectedRec.sharing_scope === 'AUTHORIZED' ? 'primary' : 'info')"
              effect="dark"
              size="default"
            >
              {{ selectedRec.sharing_summary || (selectedRec.sharing_scope === 'ALL_DOCTORS' ? '全体医生可见 (全联盟免审调阅)' : '私密受控 (需知情同意申请)') }}
            </el-tag>
          </div>
          <el-button
            size="small"
            type="primary"
            @click="openAccessPolicyDialog(selectedRec)"
          >
            变更此病历访问权限
          </el-button>
        </div>

        <table class="clinical-structured-table">
          <tbody>
            <tr>
              <th width="140">就诊医疗机构</th>
              <td>{{ selectedRec.hospital_name }}（主治医生：{{ formatDoctorTitle(selectedRec.doctor_name) }}）</td>
            </tr>
            <tr>
              <th>病历编号与时间</th>
              <td>单号：{{ selectedRec.record_no }} | 就诊时间：{{ selectedRec.created_at ? selectedRec.created_at.substring(0, 16).replace('T', ' ') : '-' }}</td>
            </tr>
            <tr>
              <th>发病与病程</th>
              <td>发病时间：{{ selectedRec.onset_time || '接诊前' }} | 持续周期：{{ selectedRec.duration || '急性发作' }}</td>
            </tr>
            <tr v-if="selectedRec.vital_signs">
              <th>生命体征参数</th>
              <td><strong style="color: #0d9488;">{{ selectedRec.vital_signs }}</strong></td>
            </tr>
            <tr>
              <th>主要临床症状</th>
              <td>{{ selectedRec.symptoms || selectedRec.diagnosis }}</td>
            </tr>
            <tr>
              <th>诱发因素与病因</th>
              <td><div class="cause-box">{{ selectedRec.etiology || '无特殊诱因记录' }}</div></td>
            </tr>
            <tr>
              <th>治疗建议方案</th>
              <td><div class="plan-box">{{ selectedRec.treatment_plan || '遵医嘱随诊' }}</div></td>
            </tr>
            <tr>
              <th>完整病历小结</th>
              <td><pre class="full-pre">{{ selectedRec.diagnosis }}</pre></td>
            </tr>
          </tbody>
        </table>
      </div>
    </el-dialog>

    <!-- 单病历访问权限选择弹窗 -->
    <el-dialog
      v-model="policyDialogVisible"
      title="病历访问权限与共享范围配置"
      width="650px"
      :close-on-click-modal="false"
    >
      <div v-if="policyTargetRec" class="policy-modal-body">
        <div class="target-rec-summary">
          <div class="trs-row">
            <span class="trs-label">目标病历：</span>
            <code class="trs-code">{{ policyTargetRec.record_no }}</code>
            <el-tag size="small" type="primary" class="ml-2">{{ policyTargetRec.hospital_name }}</el-tag>
          </div>
          <div class="trs-row">
            <span class="trs-label">就诊确诊：</span>
            <strong class="text-emerald-700">{{ policyTargetRec.diagnosis ? policyTargetRec.diagnosis.substring(0, 45) : '临床门诊记录' }}</strong>
          </div>
        </div>

        <el-form label-position="top" class="mt-4">
          <el-form-item label="请选择该病历的访问权开放模式与授权范围：" required>
            <el-radio-group v-model="policyForm.policy_type" class="policy-radio-group">
              <!-- 模式 1：让所有医生都可见 -->
              <div
                class="policy-option-card"
                :class="{ active: policyForm.policy_type === 'ALL_DOCTORS', 'active-all': policyForm.policy_type === 'ALL_DOCTORS' }"
                @click="policyForm.policy_type = 'ALL_DOCTORS'"
              >
                <div class="poc-header">
                  <el-radio value="ALL_DOCTORS">
                    <strong style="font-size: 14.5px; color: #047857;">让所有医生都可见 (全联盟免审直接调阅)</strong>
                  </el-radio>
                  <el-tag size="small" type="success" effect="dark">推荐跨院就医</el-tag>
                </div>
                <p class="poc-desc">
                  向医联体内所有合作医院的执业医生全面开放调阅权限。任何外院医生在门诊随访、跨院转诊或紧急救治时，均可免去发起知情审批流程，直接查阅完整临床诊断、检验报告与影像切片。
                </p>
              </div>

              <!-- 模式 2：指定医疗机构全体成员可见 -->
              <div
                class="policy-option-card"
                :class="{ active: policyForm.policy_type === 'HOSPITAL', 'active-hospital': policyForm.policy_type === 'HOSPITAL' }"
                @click="policyForm.policy_type = 'HOSPITAL'"
              >
                <div class="poc-header">
                  <el-radio value="HOSPITAL">
                    <strong style="font-size: 14.5px; color: #1d4ed8;">指定医疗机构全体成员可见 (按医院定向授权)</strong>
                  </el-radio>
                  <el-tag size="small" type="primary" effect="dark">跨院协同</el-tag>
                </div>
                <p class="poc-desc">
                  仅向选定的某家合作医院全体执业医生开放免审调阅权限，其他医院医生无权直接查看。
                </p>
                <!-- 目标医院下拉选择框 -->
                <div v-if="policyForm.policy_type === 'HOSPITAL'" class="poc-extra-config" @click.stop>
                  <div class="pec-title">
                    <el-icon><OfficeBuilding /></el-icon>
                    <span>请选择授权开放的目标合作医院：</span>
                  </div>
                  <el-select
                    v-model="policyForm.hospital_id"
                    placeholder="请选择或输入搜索目标医院名称"
                    filterable
                    clearable
                    style="width: 100%;"
                  >
                    <el-option
                      v-for="h in hospitalsList"
                      :key="h.id"
                      :label="`${h.name} (${h.level || '三甲'} · ${h.city || '合作机构'})`"
                      :value="h.id"
                    />
                  </el-select>
                </div>
              </div>

              <!-- 模式 3：指定单个医生可见 -->
              <div
                class="policy-option-card"
                :class="{ active: policyForm.policy_type === 'DOCTOR', 'active-doctor': policyForm.policy_type === 'DOCTOR' }"
                @click="policyForm.policy_type = 'DOCTOR'"
              >
                <div class="poc-header">
                  <el-radio value="DOCTOR">
                    <strong style="font-size: 14.5px; color: #6d28d9;">指定单个医生可见 (专家/主管医生精准授权)</strong>
                  </el-radio>
                  <el-tag size="small" type="warning" effect="dark">精准受控</el-tag>
                </div>
                <p class="poc-desc">
                  仅向选定的某一位专家名医或主管医生开放专属调阅权限，其他任何医生均无权直接查阅。
                </p>
                <!-- 目标医生与医院筛选 -->
                <div v-if="policyForm.policy_type === 'DOCTOR'" class="poc-extra-config" @click.stop>
                  <div class="pec-title">
                    <el-icon><User /></el-icon>
                    <span>请选择授权开放的目标医生（支持按医院筛选联动）：</span>
                  </div>
                  <div class="pec-filters">
                    <div class="filter-item">
                      <span class="sub-label">按医院筛选：</span>
                      <el-select
                        v-model="doctorHospFilter"
                        placeholder="全部合作医院"
                        clearable
                        style="width: 100%;"
                      >
                        <el-option
                          v-for="h in hospitalsList"
                          :key="h.id"
                          :label="h.name"
                          :value="h.id"
                        />
                      </el-select>
                    </div>
                    <div class="filter-item">
                      <span class="sub-label">目标医生：</span>
                      <el-select
                        v-model="policyForm.doctor_id"
                        placeholder="选择或输入医生姓名搜索"
                        filterable
                        clearable
                        style="width: 100%;"
                      >
                        <el-option
                          v-for="doc in filteredDoctors"
                          :key="doc.id"
                          :label="`${doc.real_name}【${doc.title || '执业医师'}】· ${doc.hospital_name || ''} · ${doc.department_name || ''}`"
                          :value="doc.id"
                        >
                          <div style="display: flex; justify-content: space-between; align-items: center;">
                            <span>
                              <strong>{{ doc.real_name }}</strong>
                              <el-tag size="small" type="info" class="ml-1">{{ doc.title || '医师' }}</el-tag>
                            </span>
                            <span style="color: #94a3b8; font-size: 12px;">{{ doc.hospital_name }} - {{ doc.department_name }}</span>
                          </div>
                        </el-option>
                      </el-select>
                    </div>
                  </div>
                </div>
              </div>

              <!-- 模式 4：私密受控选项 -->
              <div
                class="policy-option-card"
                :class="{ active: policyForm.policy_type === 'PRIVATE', 'active-private': policyForm.policy_type === 'PRIVATE' }"
                @click="policyForm.policy_type = 'PRIVATE'"
              >
                <div class="poc-header">
                  <el-radio value="PRIVATE">
                    <strong style="font-size: 14.5px; color: #334155;">仅本人与原经治医生可见 (私密受控保护)</strong>
                  </el-radio>
                  <el-tag size="small" type="info">默认隐私保护</el-tag>
                </div>
                <p class="poc-desc">
                  仅限原开具接诊医生及本科室医生可见。其他任何跨机构医生如需查阅，必须通过系统向您发起在线知情同意申请并由您审批同意，或由您提供现场专属密钥。
                </p>
              </div>
            </el-radio-group>
          </el-form-item>

          <el-form-item v-if="policyForm.policy_type !== 'PRIVATE'" label="开放有效期限：">
            <el-select v-model="policyForm.days" style="width: 100%;">
              <el-option label="7 天 (临时跨院就医/单次门诊)" :value="7" />
              <el-option label="30 天 (短期随访评估)" :value="30" />
              <el-option label="90 天 (中期慢病监测)" :value="90" />
              <el-option label="365 天 (1年长期开放，推荐)" :value="365" />
              <el-option label="长期有效 (10年永久存证)" :value="3650" />
            </el-select>
          </el-form-item>
        </el-form>

        <div class="blockchain-tip-box">
          <el-icon class="mr-1"><Connection /></el-icon>
          <span>提示：点击保存后，授权策略或撤销指令将即刻打包签署并锚定至 Fabric 联盟链分布式账本中，确保合规可溯。</span>
        </div>
      </div>

      <template #footer>
        <div class="dialog-footer">
          <el-button @click="policyDialogVisible = false">取消</el-button>
          <el-button
            type="primary"
            :loading="policySubmitting"
            @click="submitAccessPolicy"
          >
            确认保存并上链存证
          </el-button>
        </div>
      </template>
    </el-dialog>

    <!-- 批量设置全部档案可见性弹窗 -->
    <el-dialog
      v-model="batchDialogVisible"
      title="批量设置健康档案访问权限与共享范围"
      width="650px"
      :close-on-click-modal="false"
    >
      <div class="batch-modal-body">
        <el-alert
          title="此操作将针对您在医联体内归档的全部门诊病历、化验报告与医学影像统一配置共享规则"
          type="warning"
          show-icon
          :closable="false"
          class="mb-3"
        />

        <el-form label-position="top">
          <el-form-item label="请选择全局访问策略与授权范围：" required>
            <el-radio-group v-model="batchForm.policy_type" class="policy-radio-group">
              <div
                class="policy-option-card"
                :class="{ active: batchForm.policy_type === 'ALL_DOCTORS', 'active-all': batchForm.policy_type === 'ALL_DOCTORS' }"
                @click="batchForm.policy_type = 'ALL_DOCTORS'"
              >
                <div class="poc-header">
                  <el-radio value="ALL_DOCTORS">
                    <strong style="font-size: 14.5px; color: #047857;">全部健康档案对所有医生可见 (全联盟免审直接调阅)</strong>
                  </el-radio>
                  <el-tag size="small" type="success" effect="dark">一键公开</el-tag>
                </div>
                <p class="poc-desc">
                  医联体全网所有执业医生均可直接调阅您的全部历史就诊与检查档案，适合多病共患或需要频繁跨院复诊的患者。
                </p>
              </div>

              <div
                class="policy-option-card"
                :class="{ active: batchForm.policy_type === 'HOSPITAL', 'active-hospital': batchForm.policy_type === 'HOSPITAL' }"
                @click="batchForm.policy_type = 'HOSPITAL'"
              >
                <div class="poc-header">
                  <el-radio value="HOSPITAL">
                    <strong style="font-size: 14.5px; color: #1d4ed8;">指定医疗机构全体成员可见 (定向医院公开全部病历)</strong>
                  </el-radio>
                  <el-tag size="small" type="primary" effect="dark">跨院协同</el-tag>
                </div>
                <p class="poc-desc">
                  向选定的某家合作医院全体执业医生开放查阅您的全部历史档案，其他医院医生仍需申请知情同意。
                </p>
                <div v-if="batchForm.policy_type === 'HOSPITAL'" class="poc-extra-config" @click.stop>
                  <div class="pec-title">
                    <el-icon><OfficeBuilding /></el-icon>
                    <span>请选择授权开放的目标合作医院：</span>
                  </div>
                  <el-select
                    v-model="batchForm.hospital_id"
                    placeholder="请选择或输入搜索目标医院名称"
                    filterable
                    clearable
                    style="width: 100%;"
                  >
                    <el-option
                      v-for="h in hospitalsList"
                      :key="h.id"
                      :label="`${h.name} (${h.level || '三甲'} · ${h.city || '合作机构'})`"
                      :value="h.id"
                    />
                  </el-select>
                </div>
              </div>

              <div
                class="policy-option-card"
                :class="{ active: batchForm.policy_type === 'DOCTOR', 'active-doctor': batchForm.policy_type === 'DOCTOR' }"
                @click="batchForm.policy_type = 'DOCTOR'"
              >
                <div class="poc-header">
                  <el-radio value="DOCTOR">
                    <strong style="font-size: 14.5px; color: #6d28d9;">指定单个医生可见 (授权专属专家查阅全部病历)</strong>
                  </el-radio>
                  <el-tag size="small" type="warning" effect="dark">专家托管</el-tag>
                </div>
                <p class="poc-desc">
                  向选定的某位主管医生或名医专家开放您全套历史病历的直接调阅权，便于专家统筹长期诊疗方案。
                </p>
                <div v-if="batchForm.policy_type === 'DOCTOR'" class="poc-extra-config" @click.stop>
                  <div class="pec-title">
                    <el-icon><User /></el-icon>
                    <span>请选择授权开放的医生：</span>
                  </div>
                  <div class="pec-filters">
                    <div class="filter-item">
                      <span class="sub-label">所属医院筛选：</span>
                      <el-select
                        v-model="batchHospFilter"
                        placeholder="全部合作医院"
                        clearable
                        style="width: 100%;"
                      >
                        <el-option
                          v-for="h in hospitalsList"
                          :key="h.id"
                          :label="h.name"
                          :value="h.id"
                        />
                      </el-select>
                    </div>
                    <div class="filter-item">
                      <span class="sub-label">选择目标医生：</span>
                      <el-select
                        v-model="batchForm.doctor_id"
                        placeholder="选择或输入医生姓名搜索"
                        filterable
                        clearable
                        style="width: 100%;"
                      >
                        <el-option
                          v-for="doc in batchFilteredDoctors"
                          :key="doc.id"
                          :label="`${doc.real_name}【${doc.title || '执业医师'}】· ${doc.hospital_name || ''} · ${doc.department_name || ''}`"
                          :value="doc.id"
                        >
                          <div style="display: flex; justify-content: space-between; align-items: center;">
                            <span>
                              <strong>{{ doc.real_name }}</strong>
                              <el-tag size="small" type="info" class="ml-1">{{ doc.title || '医师' }}</el-tag>
                            </span>
                            <span style="color: #94a3b8; font-size: 12px;">{{ doc.hospital_name }} - {{ doc.department_name }}</span>
                          </div>
                        </el-option>
                      </el-select>
                    </div>
                  </div>
                </div>
              </div>

              <div
                class="policy-option-card"
                :class="{ active: batchForm.policy_type === 'PRIVATE', 'active-private': batchForm.policy_type === 'PRIVATE' }"
                @click="batchForm.policy_type = 'PRIVATE'"
              >
                <div class="poc-header">
                  <el-radio value="PRIVATE">
                    <strong style="font-size: 14.5px; color: #334155;">全部档案恢复为私密受控 (关闭全局公开)</strong>
                  </el-radio>
                  <el-tag size="small" type="info">默认隐私保护</el-tag>
                </div>
                <p class="poc-desc">
                  撤销全局公开策略，所有跨院医生查阅均需单独发起知情同意申请。
                </p>
              </div>
            </el-radio-group>
          </el-form-item>

          <el-form-item v-if="batchForm.policy_type !== 'PRIVATE'" label="全局开放有效期限：">
            <el-select v-model="batchForm.days" style="width: 100%;">
              <el-option label="30 天" :value="30" />
              <el-option label="90 天" :value="90" />
              <el-option label="365 天 (1年，推荐)" :value="365" />
              <el-option label="长期有效 (10年)" :value="3650" />
            </el-select>
          </el-form-item>
        </el-form>
      </div>

      <template #footer>
        <div class="dialog-footer">
          <el-button @click="batchDialogVisible = false">取消</el-button>
          <el-button
            type="primary"
            :loading="batchSubmitting"
            @click="submitBatchPolicy"
          >
            提交并固化上链
          </el-button>
        </div>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import {
  WarningFilled,
  CircleCheckFilled,
  Share,
  Key,
  OfficeBuilding,
  User,
  Connection,
  Check,
  Lock,
  CopyDocument,
  ArrowRight
} from '@element-plus/icons-vue'
import { ElMessage } from 'element-plus'
import api from '../../api/client'

const records = ref<any[]>([])
const detailVisible = ref(false)
const selectedRec = ref<any>(null)

// 统计公开与定向授权数量
const publicRecordCount = computed(() => {
  return records.value.filter(r => r.sharing_scope === 'ALL_DOCTORS').length
})
const targetedRecordCount = computed(() => {
  return records.value.filter(r => r.sharing_scope === 'HOSPITAL' || r.sharing_scope === 'DOCTOR' || r.sharing_scope === 'AUTHORIZED').length
})

// 合作医院与医生列表（供灵活筛选和定向授权）
const hospitalsList = ref<any[]>([])
const doctorsList = ref<any[]>([])
const doctorHospFilter = ref<number | undefined>(undefined)
const batchHospFilter = ref<number | undefined>(undefined)

const filteredDoctors = computed(() => {
  if (!doctorHospFilter.value) {
    return doctorsList.value
  }
  return doctorsList.value.filter(d => d.hospital_id === doctorHospFilter.value)
})

const batchFilteredDoctors = computed(() => {
  if (!batchHospFilter.value) {
    return doctorsList.value
  }
  return doctorsList.value.filter(d => d.hospital_id === batchHospFilter.value)
})

// 单病历访问策略控制
const policyDialogVisible = ref(false)
const policyTargetRec = ref<any>(null)
const policyForm = ref({
  policy_type: 'ALL_DOCTORS',
  hospital_id: undefined as number | undefined,
  doctor_id: undefined as number | undefined,
  days: 365,
})
const policySubmitting = ref(false)

// 批量设置访问策略
const batchDialogVisible = ref(false)
const batchForm = ref({
  policy_type: 'ALL_DOCTORS',
  hospital_id: undefined as number | undefined,
  doctor_id: undefined as number | undefined,
  days: 365,
})
const batchSubmitting = ref(false)

async function loadHospitals() {
  try {
    const res: any = await api.get('/system/hospitals')
    if (res.code === 200) {
      hospitalsList.value = Array.isArray(res.data) ? res.data : []
    }
  } catch (err) {
    console.error('加载医疗机构列表失败', err)
  }
}

async function loadDoctors() {
  try {
    const res: any = await api.get('/system/doctors')
    if (res.code === 200) {
      doctorsList.value = Array.isArray(res.data) ? res.data : []
    }
  } catch (err) {
    console.error('加载医生列表失败', err)
  }
}

async function fetchRecords() {
  try {
    const res: any = await api.get('/medical-records')
    if (res.code === 200) {
      records.value = Array.isArray(res.data) ? res.data : (res.data?.list || [])
    }
  } catch (err) {
    console.error(err)
  }
}

onMounted(() => {
  fetchRecords()
  loadHospitals()
  loadDoctors()
})

async function viewFullDetail(rec: any) {
  try {
    const res: any = await api.get(`/medical-records/${rec.id}`)
    if (res.code === 200) {
      selectedRec.value = res.data
    } else {
      selectedRec.value = rec
    }
  } catch {
    selectedRec.value = rec
  }
  detailVisible.value = true
}

function openAccessPolicyDialog(rec: any) {
  policyTargetRec.value = rec
  doctorHospFilter.value = undefined

  let pType = 'ALL_DOCTORS'
  let hospId: number | undefined = undefined
  let docId: number | undefined = undefined

  if (rec.active_auth_target_type === 'HOSPITAL' && rec.active_auth_target_id) {
    pType = 'HOSPITAL'
    hospId = rec.active_auth_target_id
  } else if (rec.active_auth_target_type === 'DOCTOR' && rec.active_auth_target_id) {
    pType = 'DOCTOR'
    docId = rec.active_auth_target_id
    const d = doctorsList.value.find((item: any) => item.id === docId)
    if (d && d.hospital_id) {
      doctorHospFilter.value = d.hospital_id
    }
  } else if (rec.sharing_scope === 'ALL_DOCTORS') {
    pType = 'ALL_DOCTORS'
  } else if (rec.sharing_scope === 'PRIVATE') {
    pType = 'PRIVATE'
  }

  policyForm.value = {
    policy_type: pType,
    hospital_id: hospId,
    doctor_id: docId,
    days: 365,
  }
  policyDialogVisible.value = true
}

async function submitAccessPolicy() {
  if (!policyTargetRec.value) return

  let authTargetId = 0
  if (policyForm.value.policy_type === 'HOSPITAL') {
    if (!policyForm.value.hospital_id) {
      ElMessage.warning('请选择要授权的目标合作医院')
      return
    }
    authTargetId = policyForm.value.hospital_id
  } else if (policyForm.value.policy_type === 'DOCTOR') {
    if (!policyForm.value.doctor_id) {
      ElMessage.warning('请选择要授权的目标医生')
      return
    }
    authTargetId = policyForm.value.doctor_id
  }

  policySubmitting.value = true
  try {
    const res: any = await api.post(`/medical-records/${policyTargetRec.value.id}/access-policy`, {
      policy_type: policyForm.value.policy_type,
      auth_target_id: authTargetId,
      days: policyForm.value.days,
    })
    if (res.code === 200) {
      ElMessage.success(res.message || '访问权限设置成功，区块链存证已生效！')
      policyDialogVisible.value = false
      await fetchRecords()
      if (selectedRec.value && selectedRec.value.id === policyTargetRec.value.id) {
        selectedRec.value.sharing_scope = policyForm.value.policy_type
        if (res.data?.target_name) {
          selectedRec.value.active_auth_target_name = res.data.target_name
        }
      }
    } else {
      ElMessage.error(res.message || '设置失败')
    }
  } catch (err: any) {
    ElMessage.error(err.response?.data?.message || '网络请求异常，请稍后重试')
  } finally {
    policySubmitting.value = false
  }
}

function openBatchPolicyDialog() {
  batchHospFilter.value = undefined
  batchForm.value = {
    policy_type: 'ALL_DOCTORS',
    hospital_id: undefined,
    doctor_id: undefined,
    days: 365,
  }
  batchDialogVisible.value = true
}

async function submitBatchPolicy() {
  let authTargetId = 0
  if (batchForm.value.policy_type === 'HOSPITAL') {
    if (!batchForm.value.hospital_id) {
      ElMessage.warning('请选择要授权的目标合作医院')
      return
    }
    authTargetId = batchForm.value.hospital_id
  } else if (batchForm.value.policy_type === 'DOCTOR') {
    if (!batchForm.value.doctor_id) {
      ElMessage.warning('请选择要授权的目标医生')
      return
    }
    authTargetId = batchForm.value.doctor_id
  }

  batchSubmitting.value = true
  try {
    const res: any = await api.post('/medical-records/batch-access-policy', {
      policy_type: batchForm.value.policy_type,
      auth_target_id: authTargetId,
      days: batchForm.value.days,
    })
    if (res.code === 200) {
      ElMessage.success(res.message || '批量访问策略已成功更新至联盟链！')
      batchDialogVisible.value = false
      await fetchRecords()
    } else {
      ElMessage.error(res.message || '操作失败')
    }
  } catch (err: any) {
    ElMessage.error(err.response?.data?.message || '网络请求异常')
  } finally {
    batchSubmitting.value = false
  }
}

// 格式化医生职称称谓（防止"李建国医生 医生"重复）
function formatDoctorTitle(name?: string) {
  if (!name) return '执业医生'
  return name.endsWith('医生') || name.endsWith('医师') ? name : `${name} 医生`
}

// 格式化 TxID 缩略展示
function formatTxId(txId?: string) {
  if (!txId) return '-'
  if (txId.length <= 18) return txId
  return `${txId.substring(0, 10)}...${txId.substring(txId.length - 8)}`
}

// 复制 TxID 到剪贴板
function copyTxId(txId?: string) {
  if (!txId) return
  navigator.clipboard.writeText(txId).then(() => {
    ElMessage.success('区块链存证 TxID 已复制到剪贴板')
  }).catch(() => {
    ElMessage.info(`TxID: ${txId}`)
  })
}
</script>

<style scoped>
.page-container {
  max-width: 1060px;
  margin: 0 auto;
  padding-bottom: 40px;
}

/* 顶栏与标题规范 */
.page-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  margin-bottom: 20px;
  gap: 16px;
}
.header-left {
  flex: 1;
}
.title-row {
  display: flex;
  align-items: center;
  gap: 12px;
}
.page-title {
  font-size: 22px;
  font-weight: 800;
  color: #0f172a;
  margin: 0;
  letter-spacing: -0.3px;
}
.page-header-tag {
  font-weight: 600;
}
.page-sub {
  font-size: 13px;
  color: #64748b;
  margin: 6px 0 0 0;
  line-height: 1.5;
}
.header-actions {
  display: flex;
  align-items: center;
  gap: 10px;
  flex-shrink: 0;
}

/* 患者自主隐私与数据主权卡片 */
.privacy-sovereignty-card {
  background: linear-gradient(135deg, rgba(16, 185, 129, 0.08) 0%, rgba(59, 130, 246, 0.04) 100%);
  border: 1px solid rgba(16, 185, 129, 0.22);
  border-radius: 12px;
  padding: 16px 20px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 20px;
  box-shadow: 0 4px 16px rgba(16, 185, 129, 0.05);
}
.psc-left {
  display: flex;
  align-items: flex-start;
  gap: 14px;
  flex: 1;
}
.psc-icon-wrap {
  width: 40px;
  height: 40px;
  border-radius: 10px;
  background: rgba(16, 185, 129, 0.16);
  color: #059669;
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
  margin-top: 2px;
}
.psc-content {
  flex: 1;
}
.psc-title-line {
  display: flex;
  align-items: center;
  gap: 10px;
  margin-bottom: 5px;
}
.psc-title {
  font-size: 14px;
  font-weight: 700;
  color: #065f46;
}
.psc-badge {
  font-weight: 600;
}
.psc-desc {
  font-size: 12px;
  line-height: 1.6;
  color: #334155;
  margin: 0;
}
.psc-right {
  display: flex;
  align-items: center;
  gap: 16px;
  padding-left: 20px;
  border-left: 1px solid rgba(16, 185, 129, 0.2);
  flex-shrink: 0;
}
.psc-metric {
  display: flex;
  flex-direction: column;
  align-items: center;
}
.metric-val {
  font-size: 20px;
  font-weight: 800;
  color: #0f172a;
  line-height: 1.2;
}
.metric-lbl {
  font-size: 11px;
  color: #64748b;
  margin-top: 2px;
}
.psc-metric-sep {
  width: 1px;
  height: 24px;
  background: rgba(16, 185, 129, 0.2);
}

/* 时间线容器 */
.timeline-wrapper {
  background: #ffffff;
  border-radius: 14px;
  border: 1px solid #e2e8f0;
  padding: 24px 24px 10px 24px;
  box-shadow: 0 4px 20px rgba(0, 0, 0, 0.02);
}
.med-timeline-node {
  padding-bottom: 20px;
}

/* 就诊卡片主体 */
.record-glass-card {
  border-radius: 12px;
  border: 1px solid #e2e8f0;
  background: #ffffff;
  padding: 16px 20px;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.02);
  transition: all 0.2s ease;
}
.record-glass-card:hover {
  border-color: #cbd5e1;
  box-shadow: 0 6px 18px rgba(0, 0, 0, 0.04);
}

/* 卡片头部 */
.rg-head {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 12px;
}
.rg-head-left {
  display: flex;
  align-items: center;
  gap: 8px;
}
.hosp-pill {
  display: inline-flex;
  align-items: center;
  font-size: 15px;
  font-weight: 700;
  color: #0f172a;
}
.doc-pill {
  display: inline-flex;
  align-items: center;
  font-size: 14px;
  font-weight: 600;
  color: #334155;
}
.h-sep {
  color: #94a3b8;
  font-weight: bold;
}
.encounter-type-tag {
  font-weight: 600;
}
.rg-head-right {
  display: flex;
  align-items: center;
  gap: 8px;
}
.scope-badge {
  font-weight: 600;
}
.data-type-pill {
  font-weight: 600;
  background: #f1f5f9;
  color: #475569;
}
.block-pill {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  font-size: 11px;
  font-weight: 600;
  color: #059669;
  background: #ecfdf5;
  border: 1px solid #a7f3d0;
  padding: 2px 8px;
  border-radius: 10px;
  font-family: monospace;
}

/* 结构化胶囊 */
.clinical-structured-capsule {
  background: #f8fafc;
  border: 1px solid #eef2f6;
  border-radius: 10px;
  padding: 12px 16px;
  margin-bottom: 12px;
  display: flex;
  flex-direction: column;
  gap: 8px;
  font-size: 13px;
}
.vitals-row {
  display: flex;
  align-items: center;
  gap: 8px;
  padding-bottom: 8px;
  border-bottom: 1px dashed #e2e8f0;
}
.vitals-val {
  font-weight: 700;
  color: #0d9488;
  font-family: monospace;
}
.capsule-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 12px;
}
.capsule-item {
  display: flex;
  align-items: center;
  gap: 6px;
  line-height: 1.5;
}
.capsule-item.full-width {
  align-items: flex-start;
}
.capsule-label {
  font-weight: 600;
  color: #64748b;
  flex-shrink: 0;
}
.capsule-val {
  color: #1e293b;
}
.etiology-chip {
  background: #fef3c7;
  color: #92400e;
  padding: 2px 8px;
  border-radius: 4px;
  font-weight: 600;
  font-size: 12px;
}
.treatment-plan-row {
  background: rgba(16, 185, 129, 0.05);
  border-left: 3px solid #10b981;
  padding: 8px 10px;
  border-radius: 4px;
}
.plan-text {
  color: #065f46;
  font-weight: 600;
}

/* 诊断结论 */
.diagnosis-callout {
  background: #f1f5f9;
  border-left: 3px solid #3b82f6;
  padding: 8px 14px;
  border-radius: 0 6px 6px 0;
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 12px;
}
.diag-label {
  font-size: 12px;
  font-weight: 700;
  color: #1e40af;
}
.diag-value {
  font-size: 14px;
  font-weight: 700;
  color: #0f172a;
}

/* 卡片底栏 */
.rg-footer {
  display: flex;
  align-items: center;
  justify-content: space-between;
  font-size: 12px;
  color: #64748b;
  border-top: 1px dashed #e2e8f0;
  padding-top: 10px;
}
.meta-ids {
  display: flex;
  gap: 12px;
  align-items: center;
}
.meta-item {
  display: inline-flex;
  align-items: center;
  gap: 4px;
}
.meta-lbl {
  color: #94a3b8;
}
.meta-code {
  font-family: monospace;
  color: #334155;
  font-weight: 600;
}
.meta-dot {
  color: #cbd5e1;
}
.tx-hash {
  color: #2563eb;
  cursor: pointer;
  display: inline-flex;
  align-items: center;
  gap: 4px;
  transition: color 0.2s;
}
.tx-hash:hover {
  color: #1d4ed8;
  text-decoration: underline;
}
.copy-icon {
  font-size: 12px;
}
.meta-actions {
  display: flex;
  align-items: center;
  gap: 10px;
}
.policy-active-text {
  margin-left: 3px;
  font-weight: bold;
}

.detail-policy-strip {
  display: flex;
  align-items: center;
  justify-content: space-between;
  background: #f8fafc;
  border: 1px solid #e2e8f0;
  border-radius: 8px;
  padding: 10px 14px;
  margin-bottom: 16px;
}
.dps-left {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 13px;
}
.dps-label {
  font-weight: 600;
  color: #475569;
}

/* 策略选择弹窗样式 */
.policy-modal-body {
  padding: 4px 0;
}
.target-rec-summary {
  background: #f8fafc;
  border: 1px solid #e2e8f0;
  border-radius: 8px;
  padding: 10px 14px;
  display: flex;
  flex-direction: column;
  gap: 6px;
  font-size: 13px;
}
.trs-row {
  display: flex;
  align-items: center;
}
.trs-label {
  font-weight: 600;
  color: #64748b;
  min-width: 75px;
}
.trs-code {
  font-family: monospace;
  font-weight: bold;
  color: #0f172a;
}
.policy-radio-group {
  display: flex;
  flex-direction: column;
  gap: 12px;
  width: 100%;
}
.policy-option-card {
  border: 1.5px solid #e2e8f0;
  border-radius: 10px;
  padding: 12px 16px;
  cursor: pointer;
  transition: all 0.2s ease;
  background: #ffffff;
}
.policy-option-card:hover {
  border-color: #cbd5e1;
  background: #f8fafc;
}
.policy-option-card.active {
  border-color: #10b981;
  background: #ecfdf5;
  box-shadow: 0 2px 8px rgba(16, 185, 129, 0.15);
}
.policy-option-card.active.active-all {
  border-color: #10b981;
  background: #f0fdf4;
  box-shadow: 0 2px 10px rgba(16, 185, 129, 0.15);
}
.policy-option-card.active.active-hospital {
  border-color: #3b82f6;
  background: #eff6ff;
  box-shadow: 0 2px 10px rgba(59, 130, 246, 0.15);
}
.policy-option-card.active.active-doctor {
  border-color: #8b5cf6;
  background: #f5f3ff;
  box-shadow: 0 2px 10px rgba(139, 92, 246, 0.15);
}
.policy-option-card.active.active-private {
  border-color: #64748b;
  background: #f8fafc;
  box-shadow: 0 2px 10px rgba(100, 116, 139, 0.15);
}
.poc-extra-config {
  margin-top: 10px;
  padding: 12px 14px;
  background: #ffffff;
  border-radius: 8px;
  border: 1px solid #e2e8f0;
  cursor: default;
}
.pec-title {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 13px;
  font-weight: 600;
  color: #334155;
  margin-bottom: 8px;
}
.pec-filters {
  display: flex;
  gap: 12px;
}
.pec-filters .filter-item {
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: 4px;
}
.sub-label {
  font-size: 12px;
  color: #64748b;
  font-weight: 500;
}
.poc-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 6px;
}
.poc-desc {
  font-size: 12px;
  color: #64748b;
  line-height: 1.5;
  margin: 0;
  padding-left: 24px;
}
.blockchain-tip-box {
  margin-top: 14px;
  font-size: 12px;
  color: #6366f1;
  background: #eef2ff;
  border: 1px dashed #c7d2fe;
  padding: 8px 12px;
  border-radius: 6px;
}

.clinical-structured-table {
  width: 100%;
  border-collapse: collapse;
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
}
.cause-box {
  background: #fffbeb;
  padding: 6px 10px;
  border-radius: 4px;
  color: #92400e;
}
.plan-box {
  background: #ecfdf5;
  padding: 6px 10px;
  border-radius: 4px;
  color: #065f46;
}
.full-pre {
  white-space: pre-wrap;
  background: #f8fafc;
  padding: 10px;
  border-radius: 6px;
  margin: 0;
  color: #334155;
  font-family: inherit;
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
</style>
