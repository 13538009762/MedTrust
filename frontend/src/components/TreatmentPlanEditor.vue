<template>
  <div class="treatment-plan-editor">
    <!-- 1. 综合处置分区 -->
    <div class="section-card disposal-section mb-3">
      <div class="section-header">
        <div class="section-title">
          <span class="icon">🩺</span>
          <strong>综合处置 (General Management / Disposal)</strong>
          <span class="sub-tip">急救处置、生命体征监护、创面处理或无创对症干预</span>
        </div>
        <el-button size="small" type="primary" plain @click="addDisposalRow">
          ➕ 添加一行处置
        </el-button>
      </div>

      <!-- 快速短语建议 Chip -->
      <div class="quick-chips mb-2">
        <span class="chips-label">常用快捷填入：</span>
        <el-tag
          v-for="(tip, idx) in commonDisposals"
          :key="idx"
          class="clickable-chip"
          size="small"
          effect="plain"
          type="info"
          @click="insertDisposal(tip)"
        >
          + {{ tip }}
        </el-tag>
      </div>

      <!-- 动态条目列表 -->
      <div class="items-list">
        <div
          v-for="(_, index) in localData.disposals"
          :key="'disp-' + index"
          class="item-row"
        >
          <span class="item-index">{{ index + 1 }}.</span>
          <el-input
            v-model="localData.disposals[index]"
            placeholder="例如：心电监护与血氧饱和度监测、低流量持续吸氧 (2-3 L/min)"
            clearable
            class="item-input"
            @input="handleUpdate"
          />
          <el-button
            type="danger"
            link
            class="del-btn"
            title="删除该行"
            @click="removeDisposalRow(index)"
          >
            🗑️
          </el-button>
        </div>
      </div>
    </div>

    <!-- 2. 临床医嘱分区 -->
    <div class="section-card orders-section mb-3">
      <div class="section-header">
        <div class="section-title">
          <span class="icon">📋</span>
          <strong>临床医嘱与生活指导 (Clinical Advice / Orders)</strong>
          <span class="sub-tip">饮食指导、自测指标、活动作息及复查门诊周期</span>
        </div>
        <el-button size="small" type="primary" plain @click="addOrderRow">
          ➕ 添加一行医嘱
        </el-button>
      </div>

      <!-- 快速短语建议 Chip -->
      <div class="quick-chips mb-2">
        <span class="chips-label">常用快捷填入：</span>
        <el-tag
          v-for="(tip, idx) in commonOrders"
          :key="idx"
          class="clickable-chip"
          size="small"
          effect="plain"
          type="warning"
          @click="insertOrder(tip)"
        >
          + {{ tip }}
        </el-tag>
      </div>

      <!-- 动态条目列表 -->
      <div class="items-list">
        <div
          v-for="(_, index) in localData.orders"
          :key="'order-' + index"
          class="item-row"
        >
          <span class="item-index">{{ index + 1 }}.</span>
          <el-input
            v-model="localData.orders[index]"
            placeholder="例如：低盐低脂饮食（每日食盐<5g）；早晚各测量一次血压并详细记录"
            clearable
            class="item-input"
            @input="handleUpdate"
          />
          <el-button
            type="danger"
            link
            class="del-btn"
            title="删除该行"
            @click="removeOrderRow(index)"
          >
            🗑️
          </el-button>
        </div>
      </div>
    </div>

    <!-- 3. 处方用药方案分区 -->
    <div class="section-card rx-section mb-2">
      <div class="section-header">
        <div class="section-title">
          <span class="icon">💊</span>
          <strong>处方用药方案 (Prescription / Medications)</strong>
          <span class="sub-tip">输入药品关键词（如“硝苯/氨氯/二甲双胍/阿司匹林/美托洛尔”）下方自动提示标准规格与用法</span>
        </div>
        <el-button size="small" type="success" @click="addPrescriptionRow">
          ➕ 添加一行药品
        </el-button>
      </div>

      <!-- 快捷药品推荐 Chip -->
      <div class="quick-chips mb-2">
        <span class="chips-label">常见专科用药热词：</span>
        <el-tag
          v-for="(drug, idx) in hotDrugs"
          :key="idx"
          class="clickable-chip"
          size="small"
          effect="light"
          type="success"
          @click="insertPrescription(drug.full)"
        >
          {{ drug.short }}
        </el-tag>
      </div>

      <!-- 动态药品条目列表（带关键词自动完成 Autocomplete 提示） -->
      <div class="items-list">
        <div
          v-for="(_, index) in localData.prescriptions"
          :key="'rx-' + index"
          class="item-row rx-row"
        >
          <span class="item-index rx-index">{{ index + 1 }}.</span>
          <el-autocomplete
            v-model="localData.prescriptions[index]"
            :fetch-suggestions="queryDrugSuggestions"
            placeholder="输入药品名称、通用名或拼音缩写（如“硝苯”、“氨氯”、“阿司”、“二甲”、“头孢”）"
            clearable
            style="width: 100%;"
            class="rx-autocomplete"
            popper-class="rx-suggestion-popper"
            @select="(item: any) => handleDrugSelect(index, item)"
            @input="handleUpdate"
          >
            <template #default="{ item }">
              <div class="drug-item-suggestion">
                <div class="dis-top">
                  <span class="drug-name font-bold">{{ item.name }}</span>
                  <span v-if="item.brand" class="drug-brand">({{ item.brand }})</span>
                  <el-tag size="small" :type="item.tagType || 'info'" effect="light" class="ml-2">
                    {{ item.category }}
                  </el-tag>
                </div>
                <div class="dis-bottom text-xs text-slate-500">
                  <span class="drug-spec">{{ item.spec }}</span>
                  <span class="drug-usage">【用法用量】：{{ item.usage }}</span>
                </div>
              </div>
            </template>
          </el-autocomplete>
          <el-button
            type="danger"
            link
            class="del-btn"
            title="删除该药品"
            @click="removePrescriptionRow(index)"
          >
            🗑️
          </el-button>
        </div>
      </div>
    </div>

    <!-- 底部实时整合预览折叠卡片 -->
    <div class="plan-summary-bar">
      <span class="summary-text">
        📊 方案统计：已录入 <strong>{{ validCount.disposal }}</strong> 项综合处置、
        <strong>{{ validCount.orders }}</strong> 项临床医嘱、
        <strong>{{ validCount.prescriptions }}</strong> 项处方用药
      </span>
      <el-button size="small" type="info" link @click="showPreview = !showPreview">
        {{ showPreview ? '收起整合预览' : '👁️ 查看拼接后的规范方案全文' }}
      </el-button>
    </div>

    <div v-if="showPreview" class="preview-box mt-2">
      <div class="preview-title">规范化整合输出文本（将固化存证至 Fabric 联盟链与病历 PDF）：</div>
      <pre class="preview-pre">{{ formattedText }}</pre>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch } from 'vue'

const props = defineProps({
  modelValue: {
    type: String,
    default: ''
  }
})

const emit = defineEmits(['update:modelValue', 'change'])

interface TreatmentData {
  disposals: string[]
  orders: string[]
  prescriptions: string[]
}

const localData = ref<TreatmentData>({
  disposals: [''],
  orders: [''],
  prescriptions: ['']
})

const showPreview = ref(false)

// 临床常用综合处置快捷短语
const commonDisposals = [
  '心电监护与指脉氧饱和度持续监测',
  '低流量鼻导管吸氧 (2-3 L/min)',
  '绝对卧床制动，建立静脉留置针通路',
  '门诊局部消毒、清创换药与无菌包扎',
  '急症救治观察，开具危重症复核评估'
]

// 临床常用医嘱快捷短语
const commonOrders = [
  '低盐低脂饮食（每日食盐控制在 5g 以内）',
  '规律监测晨起与睡前血压/心率并详细记录',
  '避免过度劳累、情绪激动及重体力搬运',
  '戒烟限酒，保持规律作息与心情舒畅',
  '2周后心血管内科门诊复查心电图与血压',
  '若出现持续胸痛胸闷>15分钟，立即呼叫120就近就医'
]

// 常用重点药品（支持一键点击快捷录入）
const hotDrugs = [
  { short: '拜新同 (硝苯地平控释片)', full: '硝苯地平控释片 (拜新同) 30mg 口服 每日一次 清晨餐后服用' },
  { short: '络活喜 (氨氯地平片)', full: '苯磺酸氨氯地平片 (络活喜) 5mg 口服 每日一次' },
  { short: '立普妥 (阿托伐他汀)', full: '阿托伐他汀钙片 (立普妥) 20mg 口服 每晚一次' },
  { short: '拜阿司匹灵 (肠溶片)', full: '阿司匹林肠溶片 100mg 口服 每日一次 餐前服用' },
  { short: '倍他乐克 (美托洛尔)', full: '酒石酸美托洛尔片 (倍他乐克) 25mg 口服 每日两次' },
  { short: '格华止 (二甲双胍)', full: '盐酸二甲双胍缓释片 0.5g 口服 每日两次 餐中或餐后服用' },
  { short: '波立维 (氯吡格雷)', full: '硫酸氢氯吡格雷片 (波立维) 75mg 口服 每日一次' },
  { short: '洛赛克 (奥美拉唑)', full: '奥美拉唑肠溶胶囊 20mg 口服 每日两次 晨起及睡前空腹' }
]

// 丰富的权威临床药品知识库 (支持按通用名/商品名/拼音/类别匹配)
interface DrugEntry {
  value: string      // 填入输入框的规范全称
  name: string       // 通用名
  brand: string      // 商品名/别名
  pinyin: string     // 拼音缩写
  category: string   // 分类
  spec: string       // 规格
  usage: string      // 用法用量
  tagType: string    // 标签样式
}

const drugDatabase: DrugEntry[] = [
  // 1. 降压与心血管系统
  {
    name: '硝苯地平控释片',
    brand: '拜新同',
    pinyin: 'xb xbdp bxt',
    category: '降压药·钙拮抗剂',
    spec: '30mg * 7片',
    usage: '30mg 口服 每日一次 清晨餐后整片吞服',
    value: '硝苯地平控释片 (拜新同) 30mg 口服 每日一次 清晨餐后整片吞服',
    tagType: 'danger'
  },
  {
    name: '苯磺酸氨氯地平片',
    brand: '络活喜',
    pinyin: 'al aldp lhx',
    category: '降压药·长效CCB',
    spec: '5mg * 7片',
    usage: '5mg 口服 每日一次 晨起服用',
    value: '苯磺酸氨氯地平片 (络活喜) 5mg 口服 每日一次 晨起服用',
    tagType: 'danger'
  },
  {
    name: '缬沙坦胶囊',
    brand: '代文',
    pinyin: 'xst dw',
    category: '降压药·ARB',
    spec: '80mg * 7粒',
    usage: '80mg 口服 每日一次',
    value: '缬沙坦胶囊 (代文) 80mg 口服 每日一次',
    tagType: 'danger'
  },
  {
    name: '厄贝沙坦片',
    brand: '安博维',
    pinyin: 'ebst abw',
    category: '降压药·ARB',
    spec: '150mg * 7片',
    usage: '150mg 口服 每日一次',
    value: '厄贝沙坦片 (安博维) 150mg 口服 每日一次',
    tagType: 'danger'
  },
  {
    name: '酒石酸美托洛尔片',
    brand: '倍他乐克',
    pinyin: 'mtle btlk',
    category: 'β受体阻滞剂',
    spec: '25mg * 20片',
    usage: '25mg 口服 每日两次 饭前服用',
    value: '酒石酸美托洛尔片 (倍他乐克) 25mg 口服 每日两次 饭前服用',
    tagType: 'danger'
  },
  {
    name: '富马酸比索洛尔片',
    brand: '康忻',
    pinyin: 'bsle kx',
    category: 'β受体阻滞剂',
    spec: '5mg * 10片',
    usage: '5mg 口服 每日一次 早晨服用',
    value: '富马酸比索洛尔片 (康忻) 5mg 口服 每日一次 早晨服用',
    tagType: 'danger'
  },
  {
    name: '单硝酸异山梨酯缓释片',
    brand: '欣康',
    pinyin: 'dxsyslz xk',
    category: '心绞痛·扩血管',
    spec: '40mg * 10片',
    usage: '40mg 口服 每日一次 清晨服用',
    value: '单硝酸异山梨酯缓释片 (欣康) 40mg 口服 每日一次 清晨服用',
    tagType: 'danger'
  },
  {
    name: '硝酸甘油片',
    brand: '急救备用',
    pinyin: 'xsgy',
    category: '心绞痛急救',
    spec: '0.5mg * 20片',
    usage: '0.5mg 舌下含服 发作时使用',
    value: '硝酸甘油片 0.5mg 舌下含服 心绞痛发作时立即使用',
    tagType: 'danger'
  },

  // 2. 调脂与抗血小板
  {
    name: '阿托伐他汀钙片',
    brand: '立普妥',
    pinyin: 'atwtt lpt',
    category: '调脂稳定斑块',
    spec: '20mg * 7片',
    usage: '20mg 口服 每晚睡前一次',
    value: '阿托伐他汀钙片 (立普妥) 20mg 口服 每晚睡前一次',
    tagType: 'success'
  },
  {
    name: '瑞舒伐他汀钙片',
    brand: '可定',
    pinyin: 'rstt kd',
    category: '强效调脂',
    spec: '10mg * 7片',
    usage: '10mg 口服 每日一次',
    value: '瑞舒伐他汀钙片 (可定) 10mg 口服 每日一次',
    tagType: 'success'
  },
  {
    name: '阿司匹林肠溶片',
    brand: '拜阿司匹灵',
    pinyin: 'aspl baspl',
    category: '抗血小板聚集',
    spec: '100mg * 30片',
    usage: '100mg 口服 每日一次 餐前服用',
    value: '阿司匹林肠溶片 (拜阿司匹灵) 100mg 口服 每日一次 餐前半小时',
    tagType: 'success'
  },
  {
    name: '硫酸氢氯吡格雷片',
    brand: '波立维',
    pinyin: 'lpgl blw',
    category: '抗血小板抗栓',
    spec: '75mg * 7片',
    usage: '75mg 口服 每日一次',
    value: '硫酸氢氯吡格雷片 (波立维) 75mg 口服 每日一次',
    tagType: 'success'
  },

  // 3. 降糖与代谢系统
  {
    name: '盐酸二甲双胍缓释片',
    brand: '格华止',
    pinyin: 'ejsg ghz',
    category: '一线降糖',
    spec: '0.5g * 30片',
    usage: '0.5g 口服 每日两次 餐中或餐后',
    value: '盐酸二甲双胍缓释片 (格华止) 0.5g 口服 每日两次 餐中或餐后即刻服用',
    tagType: 'warning'
  },
  {
    name: '格列美脲片',
    brand: '亚莫利',
    pinyin: 'glmy yml',
    category: '促胰岛素分泌',
    spec: '2mg * 15片',
    usage: '2mg 口服 每日一次 早餐前即刻',
    value: '格列美脲片 (亚莫利) 2mg 口服 每日一次 早餐前即刻服用',
    tagType: 'warning'
  },
  {
    name: '阿卡波糖片',
    brand: '拜唐苹',
    pinyin: 'akbt btp',
    category: '餐后降糖',
    spec: '50mg * 30片',
    usage: '50mg 每日三次 与前两口饭嚼服',
    value: '阿卡波糖片 (拜唐苹) 50mg 口服 每日三次 与就餐前两口饭同嚼服',
    tagType: 'warning'
  },
  {
    name: '达格列净片',
    brand: '安达唐',
    pinyin: 'dglj adt',
    category: 'SGLT-2抑制剂',
    spec: '10mg * 14片',
    usage: '10mg 口服 每日一次 晨起服用',
    value: '达格列净片 (安达唐) 10mg 口服 每日一次 早晨服用',
    tagType: 'warning'
  },

  // 4. 消化系统用药
  {
    name: '奥美拉唑肠溶胶囊',
    brand: '洛赛克',
    pinyin: 'amlz lsk',
    category: '质子泵抑制剂·抑酸',
    spec: '20mg * 14粒',
    usage: '20mg 口服 每日两次 晨起及睡前空腹',
    value: '奥美拉唑肠溶胶囊 (洛赛克) 20mg 口服 每日两次 晨起及睡前空腹服用',
    tagType: 'primary'
  },
  {
    name: '雷贝拉唑钠肠溶片',
    brand: '波利特',
    pinyin: 'lblz blt',
    category: '强效抑酸胃粘膜保护',
    spec: '10mg * 14片',
    usage: '10mg 口服 每日一次 早晨空腹',
    value: '雷贝拉唑钠肠溶片 (波利特) 10mg 口服 每日一次 早晨空腹服用',
    tagType: 'primary'
  },
  {
    name: '多潘立酮片',
    brand: '吗丁啉',
    pinyin: 'dplt mdl',
    category: '胃肠动力药',
    spec: '10mg * 30片',
    usage: '10mg 口服 每日三次 餐前15-30分钟',
    value: '多潘立酮片 (吗丁啉) 10mg 口服 每日三次 饭前半小时服用',
    tagType: 'primary'
  },
  {
    name: '铝碳酸镁咀嚼片',
    brand: '达喜',
    pinyin: 'ltsm dx',
    category: '抗酸与胃粘膜保护',
    spec: '0.5g * 20片',
    usage: '1.0g 嚼服 每日三次 餐后1-2小时',
    value: '铝碳酸镁咀嚼片 (达喜) 1.0g 嚼碎后吞服 每日三次 餐后1-2小时服用',
    tagType: 'primary'
  },

  // 5. 呼吸系统与抗感染
  {
    name: '盐酸氨溴索口服溶液',
    brand: '沐舒坦',
    pinyin: 'axskfry mst',
    category: '祛痰止咳',
    spec: '100ml:0.3g',
    usage: '10ml 口服 每日三次 餐后',
    value: '盐酸氨溴索口服溶液 (沐舒坦) 10ml 口服 每日三次 餐后服用',
    tagType: 'info'
  },
  {
    name: '乙酰半胱氨酸泡腾片',
    brand: '富露施',
    pinyin: 'yxbgas fls',
    category: '呼吸道粘液溶解',
    spec: '0.6g * 10片',
    usage: '0.6g 温水溶解 口服 每日一次',
    value: '乙酰半胱氨酸泡腾片 (富露施) 0.6g 溶解于半杯温水 口服 每日一次',
    tagType: 'info'
  },
  {
    name: '头孢克肟分散片',
    brand: '世福素',
    pinyin: 'tbkw sfs',
    category: '抗生素 (过敏者禁用)',
    spec: '0.1g * 6片',
    usage: '0.1g 口服 每日两次',
    value: '头孢克肟分散片 0.1g 口服 每日两次（严格排查头孢过敏史）',
    tagType: 'danger'
  },
  {
    name: '盐酸左氧氟沙星片',
    brand: '可乐必妥',
    pinyin: 'zyfsx klbt',
    category: '广谱抗菌抗感染',
    spec: '0.5g * 4片',
    usage: '0.5g 口服 每日一次',
    value: '盐酸左氧氟沙星片 (可乐必妥) 0.5g 口服 每日一次 多饮水',
    tagType: 'danger'
  },
  {
    name: '阿奇霉素分散片',
    brand: '希舒美',
    pinyin: 'aqms xsm',
    category: '大环内酯类抗生素',
    spec: '0.25g * 6片',
    usage: '0.5g 口服 每日一次 餐前1小时',
    value: '阿奇霉素分散片 0.5g 口服 每日一次 餐前半小时',
    tagType: 'danger'
  },

  // 6. 解热镇痛与神经精神
  {
    name: '布洛芬缓释胶囊',
    brand: '芬必得',
    pinyin: 'blf fbd',
    category: '解热镇痛抗炎',
    spec: '0.3g * 20粒',
    usage: '0.3g 口服 每日两次 必要时',
    value: '布洛芬缓释胶囊 (芬必得) 0.3g 口服 疼痛或高热时服用 间隔不少于6小时',
    tagType: 'warning'
  },
  {
    name: '对乙酰氨基酚片',
    brand: '泰诺林',
    pinyin: 'dyxajf tnl',
    category: '退热镇痛',
    spec: '0.5g * 10片',
    usage: '0.5g 口服 必要时服用',
    value: '对乙酰氨基酚片 0.5g 口服 发热或剧烈头痛时服用',
    tagType: 'warning'
  },
  {
    name: '甲钴胺片',
    brand: '弥可保',
    pinyin: 'jga mkb',
    category: '神经营养代谢',
    spec: '0.5mg * 30片',
    usage: '0.5mg 口服 每日三次',
    value: '甲钴胺片 (弥可保) 0.5mg 口服 每日三次',
    tagType: 'info'
  }
]

// 自动补全过滤匹配函数
function queryDrugSuggestions(queryString: string, cb: (results: DrugEntry[]) => void) {
  if (!queryString || !queryString.trim()) {
    // 未输入时给出常用推荐列表（前 8 个）
    cb(drugDatabase.slice(0, 8))
    return
  }
  const q = queryString.trim().toLowerCase()
  const results = drugDatabase.filter((item) => {
    return (
      item.name.toLowerCase().includes(q) ||
      item.brand.toLowerCase().includes(q) ||
      item.pinyin.toLowerCase().includes(q) ||
      item.category.toLowerCase().includes(q) ||
      item.value.toLowerCase().includes(q)
    )
  })
  cb(results)
}

function handleDrugSelect(index: number, item: DrugEntry) {
  localData.value.prescriptions[index] = item.value
  handleUpdate()
}

// 增删行控制
function addDisposalRow() {
  localData.value.disposals.push('')
}

function removeDisposalRow(index: number) {
  if (localData.value.disposals.length > 1) {
    localData.value.disposals.splice(index, 1)
  } else {
    localData.value.disposals[0] = ''
  }
  handleUpdate()
}

function insertDisposal(tip: string) {
  // 如果最后一行是空的，直接填入；否则追加一行
  const list = localData.value.disposals
  if (list.length === 1 && !list[0].trim()) {
    list[0] = tip
  } else if (!list[list.length - 1].trim()) {
    list[list.length - 1] = tip
  } else {
    list.push(tip)
  }
  handleUpdate()
}

function addOrderRow() {
  localData.value.orders.push('')
}

function removeOrderRow(index: number) {
  if (localData.value.orders.length > 1) {
    localData.value.orders.splice(index, 1)
  } else {
    localData.value.orders[0] = ''
  }
  handleUpdate()
}

function insertOrder(tip: string) {
  const list = localData.value.orders
  if (list.length === 1 && !list[0].trim()) {
    list[0] = tip
  } else if (!list[list.length - 1].trim()) {
    list[list.length - 1] = tip
  } else {
    list.push(tip)
  }
  handleUpdate()
}

function addPrescriptionRow() {
  localData.value.prescriptions.push('')
}

function removePrescriptionRow(index: number) {
  if (localData.value.prescriptions.length > 1) {
    localData.value.prescriptions.splice(index, 1)
  } else {
    localData.value.prescriptions[0] = ''
  }
  handleUpdate()
}

function insertPrescription(fullDrugText: string) {
  const list = localData.value.prescriptions
  if (list.length === 1 && !list[0].trim()) {
    list[0] = fullDrugText
  } else if (!list[list.length - 1].trim()) {
    list[list.length - 1] = fullDrugText
  } else {
    list.push(fullDrugText)
  }
  handleUpdate()
}

// 统计有效录入项
const validCount = computed(() => {
  return {
    disposal: localData.value.disposals.filter((s) => s && s.trim()).length,
    orders: localData.value.orders.filter((s) => s && s.trim()).length,
    prescriptions: localData.value.prescriptions.filter((s) => s && s.trim()).length
  }
})

// 格式化输出为规整的临床方案文本
const formattedText = computed(() => {
  const parts: string[] = []

  const validDisposals = localData.value.disposals.map((s) => s.trim()).filter(Boolean)
  if (validDisposals.length > 0) {
    parts.push('【综合处置】\n' + validDisposals.map((s, i) => `${i + 1}. ${s}`).join('\n'))
  }

  const validOrders = localData.value.orders.map((s) => s.trim()).filter(Boolean)
  if (validOrders.length > 0) {
    parts.push('【临床医嘱】\n' + validOrders.map((s, i) => `${i + 1}. ${s}`).join('\n'))
  }

  const validRx = localData.value.prescriptions.map((s) => s.trim()).filter(Boolean)
  if (validRx.length > 0) {
    parts.push('【处方用药】\n' + validRx.map((s, i) => `${i + 1}. ${s}`).join('\n'))
  }

  return parts.join('\n\n')
})

function handleUpdate() {
  const text = formattedText.value
  emit('update:modelValue', text)
  emit('change', text, localData.value)
}

// 逆向解析传入的原始文本
function parseTextToData(text: string) {
  if (!text || !text.trim()) {
    localData.value = {
      disposals: [''],
      orders: [''],
      prescriptions: ['']
    }
    return
  }

  // 1. 尝试匹配【综合处置】、【临床医嘱】、【处方用药】三段式
  if (text.includes('【综合处置】') || text.includes('【临床医嘱】') || text.includes('【处方用药】')) {
    const dMatch = text.match(/【综合处置】([\s\S]*?)(?=【临床医嘱】|【处方用药】|$)/)
    const oMatch = text.match(/【临床医嘱】([\s\S]*?)(?=【综合处置】|【处方用药】|$)/)
    const pMatch = text.match(/【处方用药】([\s\S]*?)(?=【综合处置】|【临床医嘱】|$)/)

    const extractItems = (sectionStr?: string) => {
      if (!sectionStr) return []
      return sectionStr
        .split('\n')
        .map((line) => line.replace(/^\s*(\d+[\.、\s]+|●|-|\*)\s*/, '').trim())
        .filter(Boolean)
    }

    const dList = extractItems(dMatch ? dMatch[1] : '')
    const oList = extractItems(oMatch ? oMatch[1] : '')
    const pList = extractItems(pMatch ? pMatch[1] : '')

    localData.value = {
      disposals: dList.length > 0 ? dList : [''],
      orders: oList.length > 0 ? oList : [''],
      prescriptions: pList.length > 0 ? pList : ['']
    }
    return
  }

  // 2. 若为原有的普通序号列表（如 1. 拜新同... 2. 饮食指导...）
  const lines = text
    .split('\n')
    .map((l) => l.replace(/^\s*(\d+[\.、\s]+|●|-|\*)\s*/, '').trim())
    .filter(Boolean)

  const dList: string[] = []
  const oList: string[] = []
  const pList: string[] = []

  lines.forEach((l) => {
    if (l.includes('饮食') || l.includes('自测') || l.includes('血压') || l.includes('复查') || l.includes('门诊') || l.includes('睡眠') || l.includes('限酒') || l.includes('注意')) {
      oList.push(l)
    } else if (l.includes('吸氧') || l.includes('监护') || l.includes('处置') || l.includes('换药') || l.includes('清创') || l.includes('卧床') || l.includes('制动')) {
      dList.push(l)
    } else {
      pList.push(l)
    }
  })

  localData.value = {
    disposals: dList.length > 0 ? dList : [''],
    orders: oList.length > 0 ? oList : [''],
    prescriptions: pList.length > 0 ? pList : ['']
  }
}

// 监听外部 modelValue 变动
watch(
  () => props.modelValue,
  (newVal) => {
    if (newVal !== formattedText.value) {
      parseTextToData(newVal)
    }
  },
  { immediate: true }
)

// 暴露外部可调用的方法
function setStructuredData(data: { disposals?: string[]; orders?: string[]; prescriptions?: string[] }) {
  localData.value = {
    disposals: data.disposals && data.disposals.length > 0 ? [...data.disposals] : [''],
    orders: data.orders && data.orders.length > 0 ? [...data.orders] : [''],
    prescriptions: data.prescriptions && data.prescriptions.length > 0 ? [...data.prescriptions] : ['']
  }
  handleUpdate()
}

defineExpose({
  setStructuredData,
  parseTextToData,
  localData
})
</script>

<style scoped>
.treatment-plan-editor {
  width: 100%;
}

.section-card {
  border-radius: 10px;
  padding: 14px 16px;
  border: 1px solid #e2e8f0;
  transition: all 0.2s ease;
}

.disposal-section {
  background: #f8fafc;
  border-left: 4px solid #0284c7;
}

.orders-section {
  background: #fbfbfb;
  border-left: 4px solid #f59e0b;
}

.rx-section {
  background: #f0fdf4;
  border-left: 4px solid #10b981;
}

.section-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 10px;
}

.section-title {
  display: flex;
  align-items: center;
  gap: 8px;
  color: #1e293b;
  font-size: 14px;
}

.section-title .icon {
  font-size: 16px;
}

.sub-tip {
  font-size: 12px;
  color: #64748b;
  font-weight: normal;
  margin-left: 4px;
}

.quick-chips {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 6px;
}

.chips-label {
  font-size: 11px;
  color: #64748b;
}

.clickable-chip {
  cursor: pointer;
  border-radius: 6px;
  transition: all 0.15s ease;
  font-size: 11px;
}

.clickable-chip:hover {
  transform: translateY(-1px);
  filter: brightness(0.95);
}

.items-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.item-row {
  display: flex;
  align-items: center;
  gap: 8px;
}

.item-index {
  width: 22px;
  font-weight: 700;
  color: #475569;
  text-align: right;
  font-size: 13px;
  flex-shrink: 0;
}

.rx-index {
  color: #059669;
}

.item-input {
  flex: 1;
}

.del-btn {
  font-size: 16px;
  padding: 4px 6px;
  flex-shrink: 0;
}

.del-btn:hover {
  color: #dc2626 !important;
}

.plan-summary-bar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  background: #f1f5f9;
  padding: 8px 14px;
  border-radius: 8px;
  font-size: 12px;
  color: #334155;
}

.preview-box {
  background: #0f172a;
  border-radius: 8px;
  padding: 12px 14px;
  color: #e2e8f0;
}

.preview-title {
  font-size: 11px;
  color: #94a3b8;
  margin-bottom: 6px;
  font-weight: 600;
}

.preview-pre {
  margin: 0;
  font-family: 'Fira Code', Consolas, monospace;
  font-size: 12px;
  line-height: 1.5;
  white-space: pre-wrap;
  word-break: break-all;
  color: #34d399;
}
</style>

<style>
/* 药品自动补全下拉面板定制样式 */
.rx-suggestion-popper {
  min-width: 420px !important;
}

.drug-item-suggestion {
  padding: 4px 0;
  line-height: 1.4;
}

.drug-item-suggestion .dis-top {
  display: flex;
  align-items: center;
}

.drug-item-suggestion .drug-name {
  color: #0f172a;
  font-size: 13px;
}

.drug-item-suggestion .drug-brand {
  color: #2563eb;
  font-size: 12px;
  margin-left: 4px;
}

.drug-item-suggestion .dis-bottom {
  margin-top: 3px;
  display: flex;
  gap: 8px;
}

.drug-item-suggestion .drug-spec {
  background: #f1f5f9;
  padding: 1px 5px;
  border-radius: 4px;
  color: #475569;
}

.drug-item-suggestion .drug-usage {
  color: #059669;
}
</style>
