<template>
  <el-dialog
    v-model="visible"
    title="🎓 MedTrust 毕业设计答辩演示向导 (8 大核心场景演练指南)"
    width="920px"
    top="4vh"
    class="defense-guide-modal"
    destroy-on-close
  >
    <div class="guide-intro-banner">
      <div class="gib-left">
        <h4>基于区块链的跨机构医疗数据可信共享平台 · 答辩全流程推演导航</h4>
        <p>
          本向导严格按照毕业设计任务书指标编排。涵盖 Fabric 2.5 真实联盟链底座、密文 IPFS 托管、规范 SOAP 接诊、医技协同、零信任规则风控网关、Break-Glass 破窗急救、黑客篡改攻击演示与规范 PDF 1.4 凭证验真。
        </p>
      </div>
      <div class="gib-right">
        <el-tag size="large" type="success" effect="dark">8 大核心场景全覆盖</el-tag>
      </div>
    </div>

    <div class="scenarios-list">
      <div
        v-for="s in scenarios"
        :key="s.id"
        class="scenario-card"
        :class="{ 'current-active': currentTab === s.id }"
      >
        <div class="sc-header">
          <div class="sc-badge-index">场景 {{ s.id }}</div>
          <div class="sc-title-wrap">
            <h4 class="sc-name">{{ s.name }}</h4>
            <div class="sc-tags">
              <el-tag size="small" type="info">{{ s.roleName }} ({{ s.username }})</el-tag>
              <el-tag size="small" :type="s.tagType" effect="plain">{{ s.tagText }}</el-tag>
            </div>
          </div>
        </div>

        <p class="sc-desc"><strong>【业务背景】</strong>{{ s.desc }}</p>
        <div class="sc-points">
          <strong>【答辩考核要点】</strong>
          <ul>
            <li v-for="(p, pi) in s.points" :key="pi">{{ p }}</li>
          </ul>
        </div>

        <div class="sc-speech-box">
          <div class="speech-header">
            <span>🗣️ 答辩讲解词提示 (推荐表述):</span>
            <el-button link type="primary" size="small" @click="copySpeech(s.speech)">复制讲解词</el-button>
          </div>
          <p class="speech-content">{{ s.speech }}</p>
        </div>

        <div class="sc-action-row">
          <span class="target-path-info">目标路径: <code>{{ s.route }}</code></span>
          <el-button
            type="primary"
            size="default"
            :loading="switchingUser === s.username"
            @click="jumpToScenario(s)"
          >
            🚀 登录为【{{ s.roleName }}】并立即跳转 ➔
          </el-button>
        </div>
      </div>
    </div>

    <template #footer>
      <div class="modal-footer-content">
        <span class="mf-tip">💡 提示：所有测试账号密码统一为 <code>123456</code>；点击跳转将自动完成会话切换，无需手动注销。</span>
        <el-button @click="visible = false">关闭向导</el-button>
      </div>
    </template>
  </el-dialog>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { useAuthStore } from '../stores/auth'
import api from '../api/client'

const props = defineProps<{
  modelValue: boolean
}>()

const emit = defineEmits(['update:modelValue'])

const router = useRouter()
const auth = useAuthStore()

const visible = computed({
  get: () => props.modelValue,
  set: (val) => emit('update:modelValue', val)
})

const currentTab = ref(1)
const switchingUser = ref('')

const scenarios = [
  {
    id: 1,
    name: '双机构 Fabric 联盟链与 IPFS 底座监控',
    roleName: '监管专员',
    username: 'supervisor',
    route: '/supervisor/overview',
    tagType: 'success',
    tagText: '联盟链底座',
    desc: '展示系统底层真实接入的 Hyperledger Fabric 2.5 网络（medchannel 通道、Org1/Org2 双组织共识出块）与 IPFS 分布式节点运行指标。',
    points: [
      '底层非 Mock 伪装，已接入真实 Orderer 与 Peer0 节点 gRPC 通信',
      '实时展示区块高度、联盟链交易总数、动态验真率及全网流转大屏',
      '监管大屏底部配备【实时全网安全事件与存证流】'
    ],
    speech: '尊敬的各位答辩老师，系统底层采用 Hyperledger Fabric 2.5 联盟链与 IPFS 分布式文件系统双轨架构。屏幕上方绿标展示 medchannel 通道与 Org1/Org2 节点正常出块，全网每一笔调阅与验真事件均不可篡改地记录于监管事件流。'
  },
  {
    id: 2,
    name: '规范门诊接诊、开具医技检查与密文上链',
    roleName: '第一医院心内科医生',
    username: 'doc_a',
    route: '/doctor/records',
    tagType: 'primary',
    tagText: '临床建档上链',
    desc: '接诊患者张三，录入主观病史与客观体征，开具辅助检查申请单，由密码学引擎执行 AES-256-GCM 流式加密并存证至联盟链。',
    points: [
      '遵循国家临床标准 SOAP 规范结构化录入',
      '支持门诊、急诊、住院多类型，且自动开立医技检查单 (ORD...)',
      '无磁盘明文泄露，密文直传 IPFS 并生成真实 Fabric TxID'
    ],
    speech: '这里演示医生工作台：接诊患者后，系统按 SOAP 临床规范采集主诉与体征并开具检查单。数据在前端进行 AES-256-GCM 认证加密后推送 IPFS，智能合约同步写入分布式世界状态并返回区块链交易编号。'
  },
  {
    id: 3,
    name: '医技科室出具报告、影像归档与自动回写',
    roleName: '第一医院影像技师',
    username: 'tech_pacs_a',
    route: '/doctor/lab-center',
    tagType: 'warning',
    tagText: '多院区医技协同',
    desc: '影像科技师接收检查单，上传 DICOM/医学影像附件，录入检查结果与客观结论，自动汇总回写主就诊记录。',
    points: [
      '多院区检验科与影像科工作台协同排队处理',
      '支持医学影像切片与化验单据上传并计算附件 SHA-256 FileHash',
      '检查完成自动联动更新就诊记录状态，实现全流程业务闭环'
    ],
    speech: '现在切换至医技科室：影像医生调取待检清单并上传影像切片与诊断报告。系统自动计算附件物理哈希 FileHash 并回传门诊就诊记录，支撑经治医生下达最终确诊。'
  },
  {
    id: 4,
    name: '跨院病历调阅与基于规则的风控评估引擎拦截',
    roleName: '第二医院骨科医生',
    username: 'doc_b',
    route: '/doctor/query/pending',
    tagType: 'danger',
    tagText: '零信任规则风控',
    desc: '第二医院医生跨机构检索患者张三在第一医院的既往病历，触发基于规则的风控评估引擎（RULE_ENGINE_WEIGHTED）精准拦截。',
    points: [
      '绝非黑盒虚假 AI/ML，明确展示规则引擎 5 大可解释规则因子（跨机构+15，未授权+30，夜间访问+20，高频访问+25等）',
      '拦截弹窗展示透明的因子扣分权重、实际得分与触发释义',
      '未获授权敏感临床字段严密脱敏（*** 掩码保护）'
    ],
    speech: '这是核心安全机制：外院医生试图直接调阅患者病历时，零信任安全网关与规则引擎介入。我们坚决摒弃虚假的黑盒机器学习，采用完全透明的加权规则引擎，精准判定未授权与跨机构加分，拦截未授权访问。'
  },
  {
    id: 5,
    name: '患者在线知情授权与现场调阅密钥秒级解锁',
    roleName: '患者张三',
    username: 'pat_zhang',
    route: '/patient/auth',
    tagType: 'success',
    tagText: '患者自主授权',
    desc: '演示双通道授权：① 患者登录个人端在线点击同意授权；② 患者在接诊现场输入专属调阅密钥（123456），医生端秒级解密放行。',
    points: [
      '符合《个人信息保护法》与患者知情同意最高合规原则',
      '现场密码核验走单向密码学哈希比对并上链存证',
      '授权生效后外院医生刷新即可直接查看完整病历'
    ],
    speech: '患者自主授权中心赋予患者对个人数据的绝对控制权。系统支持在线知情审批与诊室现场密钥秒级解锁双模式，既满足法律知情同意要求，又保障门诊就医时效。'
  },
  {
    id: 6,
    name: '急诊危重 Break-Glass 破窗抢救放行与监管闭环',
    roleName: '第二医院医生',
    username: 'doc_b',
    route: '/doctor/query/pending',
    tagType: 'danger',
    tagText: 'Break-Glass 破窗',
    desc: '患者严重昏迷休克无法授权时，医生签署法律免责声明触发 Break-Glass 紧急破窗；监管专员事后审核，若违规即刻封禁，并可一键解除限制。',
    points: [
      '生命第一原则：急救场景无需等待授权，24 小时急救窗口直接放行',
      '破窗事件全量固化上链不可抵赖（CreateEmergencyRecord）',
      '监管审核台支持违规惩戒（RESTRICTED）与演示复原【一键解除限制】'
    ],
    speech: 'Break-Glass 紧急破窗机制是医疗核心特性。在休克急救极端场景下，医生签署声明即可即刻解锁过敏史与急救病史。系统全链存证并推送至监管专员审核台，形成事中放行、事后审计追责闭环。'
  },
  {
    id: 7,
    name: '黑客攻击数据库篡改与三阶段区块链验真报警',
    roleName: '监管专员',
    username: 'supervisor',
    route: '/supervisor/verify',
    tagType: 'danger',
    tagText: '防篡改攻防演练',
    desc: '点击【模拟真实数据库恶意篡改】，直接修改 MySQL 底层临床诊断与用药；点击动态验真，系统三阶段验真即刻亮起红标警报并展示哈希不匹配！一键恢复后恢复绿标。',
    points: [
      '阶段一：附件指纹验真 (FileHash)' ,
      '阶段二：结构化临床数据 Merkle 综合摘要验真 (ClinicalHash)',
      '阶段三：联盟链身份与智能合约背书核验 (TxID & BlockHeight)',
      '配备【查看区块链原始存证】弹窗，展示链上真实 JSON'
    ],
    speech: '这是整个答辩最精彩的攻防演练：我们模拟黑客攻击后台数据库篡改处方。在安全验真工作台发起动态核验，系统解密重算哈希，与 Fabric 链上原始存证比对，瞬间亮起高危红标！点击【查看区块链原始存证】可查验真实的链上不可篡改凭证。'
  },
  {
    id: 8,
    name: '规范合规电子病历 PDF 1.4 文档出具与离线验真',
    roleName: '第一医院医生',
    username: 'doc_a',
    route: '/doctor/records',
    tagType: 'primary',
    tagText: '标准 PDF 1.4',
    desc: '点击病历列表中【下载规范归档 PDF】，系统生成符合 ISO 32000-1 标准的合法二进制 PDF 1.4 文档，包含医院题头、SOAP 病情、Fabric TxID、CID 与防伪签章。',
    points: [
      '编写真实二进制 PDF 生成器，绝非假借 .pdf 后缀的纯文本',
      '包含国标 UniGB-UTF16-H 矢量中文与标准 xref 交叉引用表',
      '可使用 Adobe Acrobat、Edge、Chrome 正常渲染打开并打印'
    ],
    speech: '最后是合规出证：系统生成符合 PDF 1.4 标准的合法二进制电子病历存证文档，内嵌区块链交易编号、区块高度与防伪指纹，真正实现了医疗数据可信跨机构流通与国家卫健委规范归档。'
  }
]

async function jumpToScenario(s: any) {
  switchingUser.value = s.username
  try {
    if (auth.user?.username !== s.username) {
      const res: any = await api.post('/auth/login', {
        username: s.username,
        password: '123456'
      })
      if (res.code === 200 && res.data?.token) {
        auth.setAuth(res.data.token, res.data.user)
        ElMessage.success(`已成功切换为【${s.roleName} (${s.username})】`)
      } else {
        throw new Error(res.message || '登录切换失败')
      }
    }
    visible.value = false
    router.push(s.route)
  } catch (err: any) {
    ElMessage.error(err?.message || '账号切换失败')
  } finally {
    switchingUser.value = ''
  }
}

function copySpeech(text: string) {
  navigator.clipboard.writeText(text).then(() => {
    ElMessage.success('讲解词已复制到剪贴板！')
  }).catch(() => {
    ElMessage.warning('复制失败，请手动划选复制')
  })
}
</script>

<style scoped>
.guide-intro-banner {
  background: linear-gradient(135deg, #1e3a8a 0%, #3b82f6 100%);
  color: #ffffff;
  padding: 16px 20px;
  border-radius: 8px;
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 20px;
  box-shadow: 0 4px 14px rgba(37, 99, 235, 0.2);
}

.gib-left h4 {
  margin: 0 0 6px 0;
  font-size: 16px;
  font-weight: 700;
}

.gib-left p {
  margin: 0;
  font-size: 12px;
  opacity: 0.9;
  line-height: 1.5;
  max-width: 650px;
}

.scenarios-list {
  display: flex;
  flex-direction: column;
  gap: 16px;
  max-height: 62vh;
  overflow-y: auto;
  padding-right: 6px;
}

.scenario-card {
  background: #f8fafc;
  border: 1px solid #e2e8f0;
  border-radius: 10px;
  padding: 16px;
  transition: all 0.2s ease;
}

.scenario-card:hover {
  border-color: #3b82f6;
  box-shadow: 0 4px 16px rgba(59, 130, 246, 0.08);
}

.sc-header {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-bottom: 10px;
}

.sc-badge-index {
  background: #1e3a8a;
  color: #ffffff;
  font-size: 12px;
  font-weight: 800;
  padding: 4px 10px;
  border-radius: 6px;
  flex-shrink: 0;
}

.sc-title-wrap {
  flex: 1;
  display: flex;
  justify-content: space-between;
  align-items: center;
  flex-wrap: wrap;
  gap: 8px;
}

.sc-name {
  margin: 0;
  font-size: 15px;
  font-weight: 700;
  color: #0f172a;
}

.sc-tags {
  display: flex;
  gap: 6px;
}

.sc-desc {
  margin: 0 0 8px 0;
  font-size: 13px;
  color: #334155;
  line-height: 1.5;
}

.sc-points {
  background: #ffffff;
  border: 1px solid #e2e8f0;
  border-radius: 6px;
  padding: 8px 12px;
  margin-bottom: 10px;
  font-size: 12px;
  color: #475569;
}

.sc-points strong {
  color: #1e293b;
  display: block;
  margin-bottom: 4px;
}

.sc-points ul {
  margin: 0;
  padding-left: 18px;
}

.sc-points li {
  margin-bottom: 2px;
}

.sc-speech-box {
  background: #f0fdf4;
  border: 1px dashed #86efac;
  border-radius: 6px;
  padding: 10px 12px;
  margin-bottom: 12px;
}

.speech-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 4px;
  font-size: 12px;
  font-weight: 700;
  color: #166534;
}

.speech-content {
  margin: 0;
  font-size: 12px;
  color: #15803d;
  line-height: 1.5;
  font-style: italic;
}

.sc-action-row {
  display: flex;
  justify-content: space-between;
  align-items: center;
  border-top: 1px solid #e2e8f0;
  padding-top: 10px;
}

.target-path-info {
  font-size: 12px;
  color: #64748b;
}

.target-path-info code {
  background: #e2e8f0;
  padding: 2px 6px;
  border-radius: 4px;
  color: #0f172a;
  font-weight: 600;
}

.modal-footer-content {
  display: flex;
  justify-content: space-between;
  align-items: center;
  width: 100%;
}

.mf-tip {
  font-size: 12px;
  color: #64748b;
}
</style>
