package service_test

import (
	"fmt"
	"testing"

	"medtrust-backend/config"
	"medtrust-backend/repository"
	"medtrust-backend/service"
)

func TestInfectionRiskEngine(t *testing.T) {
	// 初始化配置与数据库
	_, err := config.LoadConfig("../config/config.yaml")
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	_, err = repository.InitDB()
	if err != nil {
		t.Fatalf("Failed to init DB: %v", err)
	}

	engine := service.GetInfectionService()

	// 1. 测试周华强 (PAT_0009, id: 26) - 艾滋病 HIV
	alertHIV, err := engine.AnalyzePatientRisks(26)
	if err != nil {
		t.Fatalf("AnalyzePatientRisks(26) failed: %v", err)
	}
	fmt.Printf("[TEST HIV] HasRisk: %v, RiskLevel: %s, Diseases: %d\n", alertHIV.HasRisk, alertHIV.RiskLevel, len(alertHIV.Diseases))
	if !alertHIV.HasRisk || alertHIV.RiskLevel != "CRITICAL" {
		t.Errorf("Expected CRITICAL HIV alert, got: %+v", alertHIV)
	}
	if len(alertHIV.ProtectionGear) == 0 {
		t.Errorf("Expected protection gear items, got 0")
	}

	// 2. 测试林秀芝 (PAT_0010, id: 27) - 活动性肺结核 TB
	alertTB, err := engine.AnalyzePatientRisks(27)
	if err != nil {
		t.Fatalf("AnalyzePatientRisks(27) failed: %v", err)
	}
	fmt.Printf("[TEST TB] HasRisk: %v, RiskLevel: %s, Diseases: %d\n", alertTB.HasRisk, alertTB.RiskLevel, len(alertTB.Diseases))
	if !alertTB.HasRisk || alertTB.RiskLevel != "HIGH" {
		t.Errorf("Expected HIGH TB alert, got: %+v", alertTB)
	}

	// 3. 测试张三 (PAT_0001, id: 4) - 慢性乙肝 HBV
	alertHBV, err := engine.AnalyzePatientRisks(4)
	if err != nil {
		t.Fatalf("AnalyzePatientRisks(4) failed: %v", err)
	}
	fmt.Printf("[TEST HBV] HasRisk: %v, RiskLevel: %s, Diseases: %d\n", alertHBV.HasRisk, alertHBV.RiskLevel, len(alertHBV.Diseases))
	if !alertHBV.HasRisk {
		t.Errorf("Expected HBV alert for Zhang San, got: %+v", alertHBV)
	}

	// 4. 测试李四 (PAT_0002, id: 5) - 无高危传染病
	alertSafe, err := engine.AnalyzePatientRisks(5)
	if err != nil {
		t.Fatalf("AnalyzePatientRisks(5) failed: %v", err)
	}
	fmt.Printf("[TEST SAFE] HasRisk: %v, RiskLevel: %s\n", alertSafe.HasRisk, alertSafe.RiskLevel)
	if alertSafe.HasRisk {
		t.Errorf("Expected SAFE status for Li Si, got risk: %+v", alertSafe)
	}
}
