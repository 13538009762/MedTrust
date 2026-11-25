package service

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"medtrust-backend/config"
	"medtrust-backend/model"
	"medtrust-backend/pkg/blockchain"
	"medtrust-backend/pkg/crypto"
	"medtrust-backend/pkg/pdf"
	"medtrust-backend/repository"
	"strings"
	"time"
)

type UploadRecordParams struct {
	DoctorID         uint64
	PatientID        uint64
	DataType         string
	OnsetTime        string
	Duration         string
	Symptoms         string
	Etiology         string
	TreatmentPlan    string
	VitalSigns       string
	Diagnosis        string
	FileName         string
	FileType         string
	FileData         []byte
	EncounterType    string
	DepartmentName   string
	Status           string
	ChiefComplaint   string
	PresentIllness   string
	InitialDiagnosis string
	DiagnosticBasis  string
	NeedExam         bool
	ExamItems        string
	ExamReason       string
	ExamResult       string
	ExamDoctor       string
	ExamTime         string
	ClientIP         string
	IdempotencyKey   string
}

// UploadRecord 录入就诊事件并在本地完成 AES-256-GCM 加密，存入 IPFS，上链锚定存证 (具备幂等拦截与 DB.Transaction 强原子事务保护)
func (s *MedicalService) UploadRecord(p UploadRecordParams) (*model.MedicalRecord, error) {
	// 0. 幂等性校验拦截 (严格绑定 doctor_id + idempotency_key，防止跨医生碰撞回放)
	if p.IdempotencyKey != "" {
		existing, err := repository.DefaultMedicalRepository.FindByDoctorAndIdempotencyKeyWithFiles(p.DoctorID, p.IdempotencyKey)
		if err == nil && existing.ID > 0 {
			log.Printf("[MedicalService] ⚠️ 触发幂等拦截: DoctorID=%d, IdempotencyKey=%s, 复用既有就诊记录 ID=%d, RecordNo=%s", p.DoctorID, p.IdempotencyKey, existing.ID, existing.RecordNo)
			return existing, nil
		}
	}

	doc, err := repository.DefaultUserRepository.FindByID(p.DoctorID)
	if err != nil {
		return nil, fmt.Errorf("医生不存在: %w", err)
	}

	// 1. 生成就诊档案业务编号
	randBytes := make([]byte, 4)
	_, _ = rand.Read(randBytes)
	recordNo := fmt.Sprintf("ENC%s%s", time.Now().Format("20060102"), hex.EncodeToString(randBytes))

	// 字段兜底兼容
	if p.Symptoms == "" && p.ChiefComplaint != "" {
		p.Symptoms = p.ChiefComplaint
	}
	if p.ChiefComplaint == "" && p.Symptoms != "" {
		p.ChiefComplaint = p.Symptoms
	}
	if p.DataType == "" {
		p.DataType = "EMR"
	}
	if p.EncounterType == "" {
		p.EncounterType = "OUTPATIENT"
	}
	if p.DepartmentName == "" {
		p.DepartmentName = "综合门诊"
	}
	if p.Status == "" {
		p.Status = "COMPLETED"
	}

	// 若未单独上传附件，系统自动为本次就诊生成规范电子病历归档切片凭证 (符合 PDF 1.4 标准)
	if len(p.FileData) == 0 {
		patName := "陈晨"
		patient, err := repository.DefaultUserRepository.FindByID(p.PatientID)
		if err == nil && patient.RealName != "" {
			patName = patient.RealName
		}
		hospName := doc.HospitalName
		if hospName == "" && doc.HospitalID > 0 {
			if hosp, err := repository.DefaultHospitalRepository.FindByID(doc.HospitalID); err == nil && hosp.Name != "" {
				hospName = hosp.Name
			}
		}
		pdfBytes, _ := pdf.GenerateAttestationPDF(pdf.AttestationData{
			RecordNo:         recordNo,
			HospitalName:     hospName,
			EncounterType:    p.EncounterType,
			DepartmentName:   p.DepartmentName,
			PatientName:      patName,
			DoctorName:       doc.RealName,
			CreatedAt:        time.Now(),
			ChiefComplaint:   p.ChiefComplaint,
			PresentIllness:   p.PresentIllness,
			OnsetTime:        p.OnsetTime,
			Duration:         p.Duration,
			VitalSigns:       p.VitalSigns,
			ExamResult:       p.ExamResult,
			InitialDiagnosis: p.InitialDiagnosis,
			DiagnosticBasis:  p.DiagnosticBasis,
			Diagnosis:        p.Diagnosis,
			Etiology:         p.Etiology,
			TreatmentPlan:    p.TreatmentPlan,
		})
		p.FileData = pdfBytes
		p.FileName = fmt.Sprintf("临床就诊规范归档凭据_%s.pdf", recordNo)
		p.FileType = "pdf"
	}

	// 2. 使用 AES-256-GCM 信封加密原始明文，绑定 AAD (patient:ID|record:NO)，生成 SHA-256 指纹
	masterKey := config.AppConfig.GetAESMasterKey()
	cipherData, encDEK, ivNonce, fileHash, err := crypto.EncryptMedicalFile(p.FileData, p.PatientID, recordNo, masterKey)
	if err != nil {
		return nil, fmt.Errorf("AES-256-GCM 加密失败: %w", err)
	}
	clinicalHash := ComputeRecordHashWithFileHash(&model.MedicalRecord{
		RecordNo:         recordNo,
		DataType:         p.DataType,
		EncounterType:    p.EncounterType,
		DepartmentName:   p.DepartmentName,
		OnsetTime:        p.OnsetTime,
		Duration:         p.Duration,
		Symptoms:         p.Symptoms,
		Etiology:         p.Etiology,
		TreatmentPlan:    p.TreatmentPlan,
		Diagnosis:        p.Diagnosis,
		ChiefComplaint:   p.ChiefComplaint,
		PresentIllness:   p.PresentIllness,
		VitalSigns:       p.VitalSigns,
		InitialDiagnosis: p.InitialDiagnosis,
		DiagnosticBasis:  p.DiagnosticBasis,
		ExamItems:        p.ExamItems,
		ExamReason:       p.ExamReason,
		ExamResult:       p.ExamResult,
	}, fileHash)

	// 3. 密文存储至 IPFS 节点获取唯一 CID (严禁明文磁盘落地)
	cid, err := s.ipfsService.Add(cipherData)
	if err != nil {
		return nil, fmt.Errorf("IPFS 存储失败: %w", err)
	}

	// 4. 联盟链交易锚定: 提交 MedicalAsset 智能合约存证
	txID, height, err := blockchain.DefaultService.CommitAsset("MEDICAL_RECORD", recordNo, map[string]interface{}{
		"record_no":      recordNo,
		"patient_id":     p.PatientID,
		"doctor_id":      p.DoctorID,
		"hospital_id":    doc.HospitalID,
		"cid":            cid,
		"file_hash":      fileHash,
		"clinical_hash":  clinicalHash,
		"data_type":      p.DataType,
		"encounter_type": p.EncounterType,
		"create_time":    time.Now().Format(time.RFC3339),
	})
	if err != nil {
		return nil, fmt.Errorf("区块链存证失败: %w", err)
	}
	// 上链成功但后续 DB 落库可能失败：注册一个回滚钩子，DB 失败时清理 IPFS 孤儿密文。
	// 说明：链上存证按设计不可撤销，此处只保证"无 DB 记录却留密文"的存储泄漏被消除。
	dbCommitted := false
	defer func() {
		if !dbCommitted {
			if delErr := s.ipfsService.Delete(cid); delErr != nil {
				log.Printf("[UploadCleanup] ⚠️ 上传失败后清理 IPFS 孤儿密文失败 cid=%s err=%v（需人工核查存储泄漏）", cid, delErr)
			} else {
				log.Printf("[UploadCleanup] 已清理 IPFS 孤儿密文 cid=%s（DB 落库失败，链上锚点保留待对账）", cid)
			}
		}
	}()

	// 5. 写入 MySQL 数据库
	record := model.MedicalRecord{
		RecordNo:         recordNo,
		PatientID:        p.PatientID,
		DoctorID:         p.DoctorID,
		HospitalID:       doc.HospitalID,
		DataType:         p.DataType,
		OnsetTime:        p.OnsetTime,
		Duration:         p.Duration,
		Symptoms:         p.Symptoms,
		Etiology:         p.Etiology,
		TreatmentPlan:    p.TreatmentPlan,
		VitalSigns:       p.VitalSigns,
		Diagnosis:        p.Diagnosis,
		FabricTxID:       txID,
		BlockHeight:      height,
		EncounterType:    p.EncounterType,
		DepartmentName:   p.DepartmentName,
		Status:           p.Status,
		ChiefComplaint:   p.ChiefComplaint,
		PresentIllness:   p.PresentIllness,
		InitialDiagnosis: p.InitialDiagnosis,
		DiagnosticBasis:  p.DiagnosticBasis,
		NeedExam:         p.NeedExam,
		ExamItems:        p.ExamItems,
		ExamReason:       p.ExamReason,
		ExamResult:       p.ExamResult,
		ExamDoctor:       p.ExamDoctor,
		ExamTime:         p.ExamTime,
		IdempotencyKey:   p.IdempotencyKey,
		SyncStatus:       SyncStatusCompleted,
		CreatedAt:        time.Now(),
	}

	fileRecord := model.MedicalFile{
		FileName:     p.FileName,
		FileType:     p.FileType,
		FileSize:     uint64(len(p.FileData)),
		IPFSCID:      cid,
		FileHash:     fileHash,
		EncryptedDEK: encDEK,
		IVNonce:      ivNonce,
		CreatedAt:    time.Now(),
	}

	// 5. 使用关系型数据库事务 (ACID) 强原子落库，确保病历主表与附件表原子一致，任何失败全量回滚
	err = repository.Transaction(func(tx *repository.TxRunner) error {
		if err := tx.Create(&record); err != nil {
			return fmt.Errorf("创建病历数据库记录失败: %w", err)
		}
		fileRecord.RecordID = record.ID
		if err := tx.Create(&fileRecord); err != nil {
			return fmt.Errorf("保存病历附件记录失败: %w", err)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	dbCommitted = true

	record.Files = []model.MedicalFile{fileRecord}
	// 严格遵循链下密文存储规范，服务端本地磁盘零明文落地，数据均以 AES-256-GCM 密文托管于 IPFS
	uploadIP := p.ClientIP
	if uploadIP == "" {
		uploadIP = "SYSTEM_INTERNAL"
	}
	DefaultAuditService.Log(p.DoctorID, "UPLOAD", "RECORD", recordNo, doc.HospitalID, "SUCCESS", "LOW", uploadIP)
	return &record, nil
}

// ComputeRecordHash 计算结构化病历与影像哈希综合指纹，任何字段被篡改均会导致哈希不符
type clinicalHashPayload struct {
	RecordNo         string `json:"record_no"`
	DataType         string `json:"data_type"`
	EncounterType    string `json:"encounter_type"`
	DepartmentName   string `json:"department_name"`
	OnsetTime        string `json:"onset_time"`
	Duration         string `json:"duration"`
	Symptoms         string `json:"symptoms"`
	Etiology         string `json:"etiology"`
	TreatmentPlan    string `json:"treatment_plan"`
	Diagnosis        string `json:"diagnosis"`
	ChiefComplaint   string `json:"chief_complaint"`
	PresentIllness   string `json:"present_illness"`
	VitalSigns       string `json:"vital_signs"`
	InitialDiagnosis string `json:"initial_diagnosis"`
	DiagnosticBasis  string `json:"diagnostic_basis"`
	ExamItems        string `json:"exam_items"`
	ExamReason       string `json:"exam_reason"`
	ExamResult       string `json:"exam_result"`
	FileHash         string `json:"file_hash"`
}

// ComputeRecordHashWithFileHash 基于全量临床结构化字段与指定附件 SHA-256 计算规范 JSON 指纹，
// 防止分隔符注入并确保任何临床明细或检査所见被篡改均导致哈希失配。
func ComputeRecordHashWithFileHash(r *model.MedicalRecord, fileHash string) string {
	if r == nil {
		return crypto.CalculateSHA256(nil)
	}
	payload := clinicalHashPayload{
		RecordNo:         r.RecordNo,
		DataType:         r.DataType,
		EncounterType:    r.EncounterType,
		DepartmentName:   r.DepartmentName,
		OnsetTime:        r.OnsetTime,
		Duration:         r.Duration,
		Symptoms:         r.Symptoms,
		Etiology:         r.Etiology,
		TreatmentPlan:    r.TreatmentPlan,
		Diagnosis:        r.Diagnosis,
		ChiefComplaint:   r.ChiefComplaint,
		PresentIllness:   r.PresentIllness,
		VitalSigns:       r.VitalSigns,
		InitialDiagnosis: r.InitialDiagnosis,
		DiagnosticBasis:  r.DiagnosticBasis,
		ExamItems:        r.ExamItems,
		ExamReason:       r.ExamReason,
		ExamResult:       r.ExamResult,
		FileHash:         fileHash,
	}
	raw, _ := json.Marshal(payload)
	return crypto.CalculateSHA256(raw)
}

// ComputeRecordHash 计算结构化病历与主归档附件的综合指纹，任何临床或文件字段被篡改均会导致哈希不符
func ComputeRecordHash(r *model.MedicalRecord) string {
	if r == nil {
		return crypto.CalculateSHA256(nil)
	}
	if len(r.Files) == 0 && r.ID > 0 {
		if files, err := repository.DefaultMedicalRepository.FindFilesByRecordID(r.ID); err == nil {
			r.Files = files
		}
	}

	fileHash := ""
	if primary := SelectPrimaryMedicalFile(r); primary != nil {
		fileHash = primary.FileHash
	}
	return ComputeRecordHashWithFileHash(r, fileHash)
}

// DecryptFileAndComputeHash 从 IPFS 拉取密文并在内存中执行 AES-256-GCM 验签解密，返回解密后明文、重算的 SHA-256 及错误信息
func (s *MedicalService) DecryptFileAndComputeHash(rec *model.MedicalRecord, file *model.MedicalFile) ([]byte, string, error) {
	if file == nil || file.IPFSCID == "" {
		return nil, "", fmt.Errorf("附件 CID 为空")
	}
	ipfsSvc := s.GetIPFSService()
	cipherPack, err := ipfsSvc.Cat(file.IPFSCID)
	if err != nil || len(cipherPack) == 0 {
		return nil, "", fmt.Errorf("从 IPFS 拉取密文失败 (CID=%s): %v", file.IPFSCID, err)
	}

	var plainData []byte
	var decErr error
	if file.EncryptedDEK != "" && file.IVNonce != "" {
		masterKey := config.AppConfig.GetAESMasterKey()
		plainData, decErr = crypto.DecryptMedicalFile(cipherPack, file.EncryptedDEK, file.IVNonce, rec.PatientID, rec.RecordNo, masterKey)
	} else {
		plainData, decErr = crypto.DecryptRecordFile(cipherPack, rec.PatientID, rec.RecordNo)
		if decErr != nil {
			plainData, decErr = crypto.DecryptRecordFile(cipherPack, rec.PatientID, "")
		}
		// 非生产环境下向下兼容未配置独立 AES_MASTER_KEY 前生成的旧派生密文
		if decErr != nil && !config.AppConfig.IsProduction() {
			plainData, decErr = crypto.DecryptRecordFileWithSecret(cipherPack, rec.PatientID, rec.RecordNo, crypto.DefaultMasterSecret)
			if decErr != nil {
				plainData, decErr = crypto.DecryptRecordFileWithSecret(cipherPack, rec.PatientID, "", crypto.DefaultMasterSecret)
			}
		}
	}
	if decErr != nil || len(plainData) == 0 {
		return nil, "", fmt.Errorf("AES-256-GCM 认证标签或 AAD 校验失败: %v", decErr)
	}
	return plainData, crypto.ComputeSHA256(plainData), nil
}

// SelectPrimaryMedicalFile 选择用于链上锚定核验的主归档附件
func SelectPrimaryMedicalFile(r *model.MedicalRecord) *model.MedicalFile {
	if r == nil || len(r.Files) == 0 {
		return nil
	}
	target := &r.Files[len(r.Files)-1]
	for i := len(r.Files) - 1; i >= 0; i-- {
		if strings.Contains(r.Files[i].FileName, "归档凭据") || strings.Contains(r.Files[i].FileName, r.RecordNo) {
			target = &r.Files[i]
			break
		}
	}
	return target
}

// MatchRecordChainFileHash 统一校验主归档附件或同病历多附件与链上登记 file_hash 的一致性。
// 当提供 decryptedHash 时，首先要求其与 primaryFile.FileHash（若非空）一致；在主附件通过校验的前提下，
// 若链上 file_hash 锚定于同病历的其他历史附件，也视为多附件归档场景一致。
func MatchRecordChainFileHash(files []model.MedicalFile, primaryFile *model.MedicalFile, decryptedHash, chainFileHash string) (string, bool) {
	if primaryFile == nil {
		return "", chainFileHash == ""
	}
	localHash := primaryFile.FileHash
	if decryptedHash != "" {
		if primaryFile.FileHash == "" || decryptedHash != primaryFile.FileHash {
			return decryptedHash, false
		}
		localHash = decryptedHash
	}
	if chainFileHash == "" || localHash == chainFileHash {
		return localHash, true
	}
	if len(files) > 1 {
		for _, f := range files {
			if f.FileHash == chainFileHash {
				return f.FileHash, true
			}
		}
	}
	return localHash, false
}

// ShortHashPrefix 安全截取哈希前 16 位用于告警展示，避免短串切片越界 panic。
func ShortHashPrefix(h string) string {
	if len(h) <= 16 {
		return h
	}
	return h[:16]
}
