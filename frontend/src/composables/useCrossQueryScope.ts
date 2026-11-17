// WP-16: 跨院检索分支范围元信息 (从 CrossQueryView 迁入, 文案与行为保持原样)
import { computed } from 'vue'
import type { RouteLocationNormalizedLoaded, Router } from 'vue-router'

export type CrossQueryScope = 'authorized' | 'local' | 'pending' | 'all'

export interface CrossScopeMeta {
  title: string
  sub: string
  tag: string
  tagType: 'success' | 'primary' | 'warning' | 'info'
  alertTitle: string
  alertText: string
  alertType: 'success' | 'info' | 'warning'
}

export function useCrossQueryScope(route: RouteLocationNormalizedLoaded, router: Router) {
  const currentScope = computed<CrossQueryScope>({
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

  const scopeMeta = computed<CrossScopeMeta>(() => {
    switch (currentScope.value) {
      case 'authorized':
        return {
          title: '外院已有查阅权限病例',
          sub: '展示经患者显式知情同意授权或处于 24 小时 Break-Glass 紧急破窗放行期内的外院病例，免申请直接解密查阅',
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
          tag: '全网联合档案 · 跨机构互通',
          tagType: 'info' as const,
          alertTitle: '全网联合检索与零信任访问控制说明',
          alertText: '系统已接入多医疗机构数据节点。对于本人开具与本院病例直接放行；对于外院已授权病例直接放行；对于外院未授权病例由规则风险网关提供知情申请与 Break-Glass 破窗双通道。',
          alertType: 'info' as const,
        }
    }
  })

  return { currentScope, scopeMeta }
}