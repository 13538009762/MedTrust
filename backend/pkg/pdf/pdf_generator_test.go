package pdf

import (
	"os"
	"testing"
	"time"
)

func TestGenerateAttestationPDF(t *testing.T) {
	data := AttestationData{
		RecordNo:         "ENC202603129999",
		HospitalName:     "华东第一人民医院",
		EncounterType:    "专家门诊",
		DepartmentName:   "心血管内科",
		PatientName:      "张三",
		PatientIDCard:    "310101199001011234",
		PatientPhone:     "13800138000",
		DoctorName:       "李建国主任医师",
		CreatedAt:        time.Now(),
		ChiefComplaint:   "持续胸痛3小时，伴大汗淋漓",
		PresentIllness:   "患者3小时前突发胸骨后剧烈绞痛，持续不缓解",
		OnsetTime:        "2026-03-12 08:30",
		Duration:         "3小时",
		VitalSigns:       "血压 145/95mmHg, 脉搏 88次/分, 呼吸 20次/分",
		ExamResult:       "心电图提示V1-V4导联ST段抬高0.3mV，肌钙蛋白I阳性",
		InitialDiagnosis: "急性前壁心肌梗死",
		DiagnosticBasis:  "典型胸痛症状伴心电图缺血性改变及心肌标志物升高",
		Diagnosis:        "急性ST段抬高型前壁心肌梗死",
		Etiology:         "冠状动脉粥样硬化伴斑块破裂及急性血栓闭塞",
		TreatmentPlan:    "急诊行经皮冠状动脉介入术(PCI)，予阿司匹林、氯吡格雷双联抗血小板",
		FabricTxID:       "0x9f8e7d6c5b4a3210efedcba0123456789abcdef0123456789abcdef012345678",
		BlockHeight:      108,
		IPFSCID:          "QmYwAPJzv5CZsnA625s3Xf2nemtYgPpHdWEz79ojWnPbdG",
		FileHash:         "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
		ClinicalHash:     "ca978112ca1bbdcafac231b39a23dc4da78608144160670c3547f480e6085a86",
	}

	pdfBytes, err := GenerateAttestationPDF(data)
	if err != nil {
		t.Fatalf("GenerateAttestationPDF failed: %v", err)
	}

	if len(pdfBytes) < 500 {
		t.Fatalf("PDF output too small: %d bytes", len(pdfBytes))
	}

	_ = os.WriteFile("sample_attestation.pdf", pdfBytes, 0644)
	t.Logf("Generated valid PDF: %d bytes", len(pdfBytes))
}
