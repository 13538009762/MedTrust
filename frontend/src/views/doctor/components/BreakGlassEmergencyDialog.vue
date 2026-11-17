<template>
  <!-- 极端急危重症救治 Break-Glass 紧急破窗通道说明 (WP-16)
       独立通道指引卡：当网关弹窗走 BREAK_GLASS 通道时嵌入展示的破窗协议与法律声明。
       原文案与校验规则 (≥15 字说明 + 责任确认勾选) 与 CrossQueryView 保持一致。 -->
  <div class="break-glass-protocol-card">
    <!-- 若医生处于受限状态，显示醒目 403 阻断警示 -->
    <el-alert
      v-if="restricted"
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
        根据《医疗数据可信共享实施规范》，若患者处于<strong>严重休克、深昏迷、心跳呼吸骤停</strong>等危及生命的极端场景，无法表达知情意愿，医生签署临床法律责任声明后，系统允许触发 <strong>Break-Glass 破窗机制</strong>紧急放行并同步异步上链记录生成闭环监管审计工单。
      </p>
    </div>

    <el-alert
      v-if="bgForm.description.length > 0 && bgForm.description.trim().length < 15"
      title="临床急救说明不得少于 15 字：请补充急救诊断情况、抢救必要性等关键信息，以满足监管审计留痕要求。"
      type="warning"
      show-icon
      :closable="false"
      class="mb-2"
    />

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
          placeholder="请详细录入急救诊断情况、抢救必要性说明，此说明将直接作为不可篡改证据固化上链并推送卫健监管审查... (不少于15字)"
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
</template>

<script setup lang="ts">
import type { BreakGlassFormState } from '@/types/clinical'

defineProps<{
  bgForm: BreakGlassFormState
  restricted: boolean
}>()
</script>
<style scoped src="../CrossQueryView.css"></style>

