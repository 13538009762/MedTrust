// 病历篡改高危预警阻断提示（全局统一复用，禁止内联样式与 Emoji）
import { ElNotification, ElMessageBox } from 'element-plus'
import type { MedicalRecordRow } from '@/types/clinical'

function escapeHtml(raw: string): string {
  return raw
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;')
    .replace(/'/g, '&#39;')
}

export function useTamperWarning() {
  function promptTamperedRecordWarning(record: MedicalRecordRow) {
    if (!record || !record.is_tampered) return

    const rec = record as MedicalRecordRow & { current_hash?: string; chain_hash?: string }
    const currentHash = escapeHtml(rec.current_hash || '（哈希计算异常）')
    const chainHash = escapeHtml(rec.chain_hash || '（链上存证缺失或未完成背书锚定）')

    ElMessageBox.alert(
      `<div class="mt-tamper-modal">
        <div class="mt-tamper-modal__head">
          <div class="mt-tamper-modal__badge" aria-hidden="true">!</div>
          <div>
            <div class="mt-tamper-modal__title">高危安全警报：检测到该病历底层数据已被篡改</div>
            <div class="mt-tamper-modal__desc">
              系统执行零信任存证核验时发现：底层数据库解密后的特征指纹与 <strong>Hyperledger Fabric 联盟链存证指纹不匹配</strong>。
            </div>
          </div>
        </div>

        <div class="mt-tamper-modal__compare">
          <div class="mt-tamper-modal__compare-title">区块链分布式存证比对详情</div>
          <div class="mt-tamper-modal__row">
            <div class="mt-tamper-modal__label">数据库当前计算哈希（异常值）：</div>
            <code class="mt-tamper-modal__code mt-tamper-modal__code--danger">${currentHash}</code>
          </div>
          <div class="mt-tamper-modal__row">
            <div class="mt-tamper-modal__label">区块链账本基准哈希（Fabric 原件）：</div>
            <code class="mt-tamper-modal__code mt-tamper-modal__code--chain">${chainHash}</code>
          </div>
        </div>

        <div class="mt-tamper-modal__notice">
          <strong>临床接诊安全阻断声明：</strong><br/>
          该病历的病情描述、诱因或医嘱记录完整性已受损。<strong>临床严禁直接采信或依据该病历下达处方。</strong>
          本次异常调阅事件已实时存证并触发监管审计告警。
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
      title: '病历数据完整性校验失败',
      message: '该病历数据指纹与区块链存证基准不符，临床严禁采信。',
      type: 'error',
      duration: 8000,
      position: 'top-right'
    })
  }

  return { promptTamperedRecordWarning }
}
