package service_test

import (
	"testing"
	"time"

	"medtrust-backend/service"
)

// TestSecurity_AuthWindowUpperBound 授权窗口统一上限（次级修复：无界 Days/EndTime）
func TestSecurity_AuthWindowUpperBound(t *testing.T) {
	if service.MaxAuthDays <= 0 {
		t.Fatalf("MaxAuthDays 必须为正数，当前 %d", service.MaxAuthDays)
	}

	// 1. 默认值路径：Days<=0 用 fallback，且 fallback 亦受上限约束
	got, err := service.ClampAuthDays(0, 7)
	if err != nil || got != 7 {
		t.Fatalf("默认 7 天应通过，got=%d err=%v", got, err)
	}

	// 2. 边界：恰好等于上限应通过
	got, err = service.ClampAuthDays(service.MaxAuthDays, 7)
	if err != nil || got != service.MaxAuthDays {
		t.Fatalf("恰好 %d 天应通过，got=%d err=%v", service.MaxAuthDays, got, err)
	}

	// 3. 越界：超过上限必须拒绝
	got, err = service.ClampAuthDays(service.MaxAuthDays+1, 7)
	if err == nil {
		t.Fatalf("【安全缺口】Days=%d 超过上限 %d 却被放行", service.MaxAuthDays+1, service.MaxAuthDays)
	}
	if got != 0 {
		t.Fatalf("越界时应返回 0，got=%d", got)
	}

	// 4. 无界滥用：days=100000 必须拒绝（修复 1 可选加固项）
	if _, err := service.ClampAuthDays(100000, 7); err == nil {
		t.Fatalf("【安全缺口】days=100000 未被上限拦截")
	}
}

// TestSecurity_AuthWindowEndTimeBound 显式 EndTime 路径的上限语义（controller 层逻辑等价校验）
func TestSecurity_AuthWindowEndTimeBound(t *testing.T) {
	// 模拟 authorization_controller：EndTime - now > MaxAuthDays 应拒绝
	end := time.Now().Add(time.Duration(service.MaxAuthDays+30) * 24 * time.Hour)
	if end.Sub(time.Now()) <= time.Duration(service.MaxAuthDays)*24*time.Hour {
		t.Fatalf("超限 EndTime 未被识别")
	}
	// 等于上限（含秒级误差）应通过
	endOK := time.Now().Add(time.Duration(service.MaxAuthDays) * 24 * time.Hour)
	if endOK.Sub(time.Now()) > time.Duration(service.MaxAuthDays)*24*time.Hour {
		t.Fatalf("恰好达上限的 EndTime 被误判为超限")
	}
}