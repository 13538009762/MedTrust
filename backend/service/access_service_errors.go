package service

import (
	"errors"
	"fmt"
)

// WP-14: service 层共享业务错误定义。
// 控制器依据错误类型映射 HTTP 状态码，service 层不感知 Gin。

var (
	// MaxAuthDays 授权有效期的统一上限（天）。所有授权建立路径（患者授权、密钥解锁、
	// 知情同意、访问策略）均不得签发超过该窗口的授权，防止客户端传 days=100000 这类
	// 无界值实现"一次授权、永久有效"。取 365 天：拒绝无界滥用，同时保留一年期授权能力。
	MaxAuthDays = 365

	// ErrRecordNotFound 病历记录不存在
	ErrRecordNotFound = errors.New("病历记录不存在")
	// ErrPatientNotFound 患者档案不存在
	ErrPatientNotFound = errors.New("患者不存在")
	// ErrMissingTarget 未指定目标病历/患者
	ErrMissingTarget = errors.New("请指定病历ID或目标患者ID")
	// ErrEmptyRecordList 申请病历列表为空
	ErrEmptyRecordList = errors.New("请至少勾选一份病历申请权限")
	// ErrRecordListMismatch 申请病历列表存在无效记录
	ErrRecordListMismatch = errors.New("申请病历列表中存在不存在的病历记录")
	// ErrRequestAlreadyHandled 申请单已被处理
	ErrRequestAlreadyHandled = errors.New("该申请已处理过")
	// ErrKeyVerifyFailed 患者授权密钥核验失败
	ErrKeyVerifyFailed = errors.New("患者授权密钥核验失败，密钥不正确！请由患者在个人中心核实或重新设置调阅密钥")
	// ErrKeyUnlockLocked 密钥解锁防暴锁定中（参数为剩余分钟数）
	ErrKeyUnlockLocked = errors.New("针对该患者的调阅密钥连续核验失败已达上限，安全防暴锁定中")
	// ErrKeyUnlockLimit 达到连续失败上限触发防暴锁定
	ErrKeyUnlockLimit = errors.New("连续密钥错误已达 5 次，已触发防暴安全保护，锁定该患者的密钥解锁通道 15 分钟")
	// ErrUnsupportedPolicyType 不支持的访问策略类型
	ErrUnsupportedPolicyType = errors.New("不支持的策略类型")
	// ErrHospitalNotFound 指定的医疗机构不存在
	ErrHospitalNotFound = errors.New("指定的医疗机构不存在")
	// ErrDoctorNotFound 指定的医生不存在或非执业医生
	ErrDoctorNotFound = errors.New("指定的医生不存在或非执业医生")
	// ErrAuthDaysExceeded 授权有效期超过统一上限
	ErrAuthDaysExceeded = fmt.Errorf("授权有效天数超过上限（最多 %d 天），请缩短授权窗口", MaxAuthDays)
	// ErrEncounterForbidden 终诊归档越权拦截（非开立该次初诊的经治医生本人）
	ErrEncounterForbidden = errors.New("越权拦截：仅开立该次初诊的经治医生本人可签署终审确诊归档")
	// ErrEncounterDoctorDisabled 执业医生账号或所属医疗机构停用/受限
	ErrEncounterDoctorDisabled = errors.New("权限拦截：当前执业账号或所属医疗机构处于停用或受限状态，禁止签署病历")
	// ErrEncounterAlreadyCompleted 就诊病历已完成终审归档并锚定上链
	ErrEncounterAlreadyCompleted = errors.New("状态冲突：该就诊病历已完成终审归档并锚定上链，严禁重复覆写")
	// ErrInvalidRestrictionTarget 解除医生限制操作目标非法（非 RESTRICTED 执业医生）
	ErrInvalidRestrictionTarget = errors.New("权限拦截：仅允许解除处于受限 (RESTRICTED) 状态的执业医生限制，禁止解封停用 (DISABLED) 或非医生账号")
)

// ClampAuthDays 将请求的授权天数规范化到 [1, MaxAuthDays]。
// 返回 days<=0 时使用的默认值同时受上限约束，避免默认值本身突破上限。
func ClampAuthDays(days int, fallback int) (int, error) {
	if days <= 0 {
		days = fallback
	}
	if days > MaxAuthDays {
		return 0, ErrAuthDaysExceeded
	}
	return days, nil
}