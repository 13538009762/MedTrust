package service

import (
	"fmt"
	"log"
	"medtrust-backend/config"
	"medtrust-backend/model"
	"medtrust-backend/pkg/blockchain"
	"medtrust-backend/pkg/crypto"
	"medtrust-backend/repository"
	"strings"
	"time"
)

// VerifyRecord 对病历执行区块链分布式防篡改核验（含 IPFS 密文拉取 + AES-256-GCM 解密验签 + 结构化综合哈希比对）
func (s *MedicalService) VerifyRecord(r *model.MedicalRecord) {
	if len(r.Files) == 0 && r.ID > 0 && repository.DB != nil {
		var files []model.MedicalFile
		if err := repository.DB.Where("record_id = ?", r.ID).Order("id asc").Find(&files).Error; err == nil {
			r.Files = files
		}
	}

	if blockchain.DefaultService == nil {
		currentHash := ComputeRecordHash(r)
		r.CurrentHash = currentHash
		r.Verified = false
		r.IsTampered = true
		r.ChainHash = ""
		r.TamperReason = "【区块链安全告警】区块链存证服务未就绪，无法比对账本指纹"
		return
	}

	chainData, exists := blockchain.DefaultService.QueryAsset(r.RecordNo)
	if !exists || chainData == nil {
		currentHash := ComputeRecordHash(r)
		r.CurrentHash = currentHash
		r.Verified = false
		r.IsTampered = true
		r.ChainHash = ""
		r.TamperReason = "【区块链安全告警】该病历在区块链分布式账本中存证缺失，未通过可信背书核验！"
		return
	}

	chainHash := ""
	if h, ok := chainData["clinical_hash"]; ok && fmt.Sprintf("%v", h) != "" && fmt.Sprintf("%v", h) != "<nil>" {
		chainHash = fmt.Sprintf("%v", h)
	} else if h, ok := chainData["file_hash"]; ok && fmt.Sprintf("%v", h) != "" && fmt.Sprintf("%v", h) != "<nil>" {
		chainHash = fmt.Sprintf("%v", h)
	}
	r.ChainHash = chainHash

	if chainHash == "" {
		currentHash := ComputeRecordHash(r)
		r.CurrentHash = currentHash
		r.Verified = false
		r.IsTampered = true
		r.TamperReason = "【区块链安全告警】链上存证数据关键指纹为空，未能通过可信背书校验！"
		return
	}

	// 阶段 0：若存在关联 IPFS 密文附件，真实拉取 IPFS 密文块并校验 AES-256-GCM AuthTag 与明文 SHA-256
	chainFileHash := ""
	if fh, ok := chainData["file_hash"]; ok && fmt.Sprintf("%v", fh) != "" && fmt.Sprintf("%v", fh) != "<nil>" {
		chainFileHash = fmt.Sprintf("%v", fh)
	}
	if primaryFile := SelectPrimaryMedicalFile(r); primaryFile != nil && primaryFile.IPFSCID != "" {
		_, decryptedFileHash, decErr := s.DecryptFileAndComputeHash(r, primaryFile)
		if decErr != nil {
			r.CurrentHash = ComputeRecordHash(r)
			r.IsTampered = true
			r.Verified = false
			r.TamperReason = fmt.Sprintf("【IPFS 密文安全告警】链下 IPFS 密文块拉取或 AES-256-GCM 认证标签校验失败 (%v)！密文可能已被篡改或替换！", decErr)
			return
		}
		expectedFH := chainFileHash
		if expectedFH == "" {
			expectedFH = primaryFile.FileHash
		}
		if expectedFH != "" && decryptedFileHash != expectedFH {
			r.CurrentHash = ComputeRecordHash(r)
			r.IsTampered = true
			r.Verified = false
			r.TamperReason = fmt.Sprintf("【IPFS 密文安全告警】IPFS 解密明文 SHA-256 (%s...) 与链上登记文件指纹 (%s...) 不匹配！", decryptedFileHash[:16], expectedFH[:16])
			return
		}
	}

	// 动态智能比对：针对就诊流程中的临床多维结构化数据与附件进行严密防篡改核验
	matched := false
	matchedHash := ""

	// 1. 首先以当前病历所有临床字段+主附件指纹计算综合哈希
	defaultHash := ComputeRecordHash(r)
	if defaultHash == chainHash {
		matched = true
		matchedHash = defaultHash
	} else if len(r.Files) > 1 {
		// 2. 若存在多个流转阶段附件，检查是否与某一历史归档阶段关联哈希一致
		for _, f := range r.Files {
			raw := fmt.Sprintf("%s|%s|%s|%s|%s|%s|%s|%s|%s",
				r.RecordNo, r.DataType, r.OnsetTime, r.Duration, r.Symptoms, r.Etiology, r.TreatmentPlan, r.Diagnosis, f.FileHash)
			calc := crypto.CalculateSHA256([]byte(raw))
			if calc == chainHash {
				matched = true
				matchedHash = calc
				break
			}
		}
	}

	if matchedHash == "" {
		matchedHash = defaultHash
	}
	r.CurrentHash = matchedHash

	// 严密安全校验：凡是计算哈希与链上不可篡改指纹不相等的，绝对判定为篡改！
	if !matched {
		r.IsTampered = true
		r.Verified = false
		subMatched := matchedHash
		if len(subMatched) > 16 {
			subMatched = subMatched[:16]
		}
		subChain := chainHash
		if len(subChain) > 16 {
			subChain = subChain[:16]
		}
		r.TamperReason = fmt.Sprintf("【区块链安全告警】病历关键临床数据（诱因/过敏史/治疗方案/诊断）或文件指纹已被恶意篡改！当前哈希: %s..., 链上不可篡改凭证: %s...", subMatched, subChain)
	} else {
		r.IsTampered = false
		r.Verified = true
	}
}

// EnsureBaselineLedgerAnchored 初始化或补充链上初始资产存证，确保种子病历在数据库被攻击篡改前已将基准哈希固化在 Fabric 账本中
func EnsureBaselineLedgerAnchored() {
	if repository.DB == nil || blockchain.DefaultService == nil {
		return
	}

	// 1. 自动注入与更新 3 份标准种子病历 (REC20250501001, REC20250502002, REC20250503003)
	seedRecordDefinitions := []struct {
		RecordNo       string
		PatientUserNo  string
		DoctorUserNo   string
		HospitalID     uint64
		DepartmentName string
		DataType       string
		EncounterType  string
		OnsetTime      string
		Duration       string
		Symptoms       string
		ChiefComplaint string
		Etiology       string
		Diagnosis      string
		TreatmentPlan  string
		FileContent    string
	}{
		{
			RecordNo:       "REC20250501001",
			PatientUserNo:  "pat_zhang",
			DoctorUserNo:   "doc_li",
			HospitalID:     1,
			DepartmentName: "心内科",
			DataType:       "EMR",
			EncounterType:  "OUTPATIENT",
			OnsetTime:      "2025-04-10",
			Duration:       "2周",
			Symptoms:       "反复心前区闷痛伴胸闷",
			ChiefComplaint: "活动后胸痛胸闷6月，加重1周",
			Etiology:       "劳累及情绪激动诱发",
			Diagnosis:      "冠状动脉粥样硬化性心脏病 劳力性心绞痛",
			TreatmentPlan:  "单硝酸异山梨酯缓释片 30mg qd, 阿托伐他汀 20mg qn",
			FileContent:    "广州第一附属医院 门诊电子病历 [REC20250501001]\n患者: 张三\n经治医生: 李建国 (主任医师)\n诊断: 冠状动脉粥样硬化性心脏病 劳力性心绞痛\n医嘱: 规范抗心肌缺血与降脂稳定斑块治疗。",
		},
		{
			RecordNo:       "REC20250502002",
			PatientUserNo:  "pat_zhang",
			DoctorUserNo:   "doc_wang",
			HospitalID:     2,
			DepartmentName: "心内科",
			DataType:       "EMR",
			EncounterType:  "OUTPATIENT",
			OnsetTime:      "2025-04-15",
			Duration:       "1个月",
			Symptoms:       "间歇性头晕、颈项发胀伴睡眠不佳",
			ChiefComplaint: "血压升高伴头胀2周，最高达 180/110 mmHg",
			Etiology:       "高盐饮食及工作熬夜",
			Diagnosis:      "原发性高血压 3级 (很高危组)",
			TreatmentPlan:  "苯磺酸氨氯地平片 5mg qd, 替米沙坦片 40mg qd",
			FileContent:    "深圳中心医院 门诊电子病历 [REC20250502002]\n患者: 张三\n经治医生: 王明德 (主治医师)\n诊断: 原发性高血压 3级 (很高危组)\n医嘱: 规律服用降压药，严格监测晨起与晚间血压。",
		},
		{
			RecordNo:       "REC20250503003",
			PatientUserNo:  "pat_zhang",
			DoctorUserNo:   "doc_chen",
			HospitalID:     2,
			DepartmentName: "急诊科",
			DataType:       "EMR",
			EncounterType:  "EMERGENCY",
			OnsetTime:      "2025-05-03 08:30",
			Duration:       "2小时",
			Symptoms:       "突发剧烈胸骨后压榨样疼痛，向左肩放射，面色苍白大汗",
			ChiefComplaint: "突发持续剧烈胸痛2小时伴大汗",
			Etiology:       "急性心肌缺血坏死",
			Diagnosis:      "急性广泛前壁心肌梗死 (AMI) Killip II级",
			TreatmentPlan:  "急诊冠脉造影(CAG)及经皮冠状动脉介入治疗(PCI)，植入DES支架1枚",
			FileContent:    "深圳中心医院 急诊抢救记录 [REC20250503003]\n患者: 张三\n经治医生: 陈建明 (主治医师)\n诊断: 急性广泛前壁心肌梗死 (AMI) Killip II级\n抢救处置: 紧急开通绿色通道行急诊PCI开通闭塞前降支。",
		},
	}

	masterKey := config.AppConfig.GetAESMasterKey()
	ipfsSvc := DefaultMedicalService.GetIPFSService()

	for _, seed := range seedRecordDefinitions {
		var pat, doc model.User
		if err := repository.DB.Where("username = ? OR user_no = ?", seed.PatientUserNo, seed.PatientUserNo).First(&pat).Error; err != nil {
			continue
		}
		if err := repository.DB.Where("username = ? OR user_no = ?", seed.DoctorUserNo, seed.DoctorUserNo).First(&doc).Error; err != nil {
			continue
		}

		var existing model.MedicalRecord
		recFound := (repository.DB.Where("record_no = ?", seed.RecordNo).First(&existing).Error == nil)

		fileData := []byte(seed.FileContent)
		expectedFileHash := crypto.ComputeSHA256(fileData)
		expectedClinicalHash := crypto.ComputeSHA256([]byte(fmt.Sprintf("%s|%s|%s|%s|%s|%s|%s|%s|%s",
			seed.RecordNo, seed.DataType, seed.OnsetTime, seed.Duration, seed.Symptoms, seed.Etiology, seed.TreatmentPlan, seed.Diagnosis, expectedFileHash)))

		// 幂等检查：若数据库记录、IPFS 密文块以及链上存证均已完整且一致，避免每次重启重新生成随机 CID 覆盖链上存证
		if recFound && existing.FabricTxID != "" {
			var existingFile model.MedicalFile
			if repository.DB.Where("record_id = ?", existing.ID).First(&existingFile).Error == nil && existingFile.IPFSCID != "" {
				if _, decFH, decErr := DefaultMedicalService.DecryptFileAndComputeHash(&existing, &existingFile); decErr == nil && decFH == expectedFileHash {
					if chainData, ok := blockchain.DefaultService.QueryAsset(seed.RecordNo); ok && chainData != nil {
						if fmt.Sprintf("%v", chainData["clinical_hash"]) == expectedClinicalHash && ComputeRecordHash(&existing) == expectedClinicalHash {
							log.Printf("[EnsureBaselineLedgerAnchored] ✅ 种子病历 %s 已处于完整一致状态 (CID=%s, TxID=%s)，跳过重复加密上链", seed.RecordNo, existingFile.IPFSCID, existing.FabricTxID)
							continue
						}
					}
				}
			}
		}

		cipherData, encDEK, ivNonce, fileHash, err := crypto.EncryptMedicalFile(fileData, pat.ID, seed.RecordNo, masterKey)
		if err != nil {
			log.Printf("[EnsureBaselineLedgerAnchored] 加密种子病历失败 %s: %v", seed.RecordNo, err)
			continue
		}

		cid, err := ipfsSvc.Add(cipherData)
		if err != nil {
			log.Printf("[EnsureBaselineLedgerAnchored] 存储种子病历至IPFS失败 %s: %v", seed.RecordNo, err)
			continue
		}

		clinicalHash := crypto.ComputeSHA256([]byte(fmt.Sprintf("%s|%s|%s|%s|%s|%s|%s|%s|%s",
			seed.RecordNo, seed.DataType, seed.OnsetTime, seed.Duration, seed.Symptoms, seed.Etiology, seed.TreatmentPlan, seed.Diagnosis, fileHash)))

		txID, height, commitErr := blockchain.DefaultService.CommitAsset("MEDICAL_RECORD", seed.RecordNo, map[string]interface{}{
			"record_no":      seed.RecordNo,
			"patient_id":     pat.ID,
			"doctor_id":      doc.ID,
			"hospital_id":    seed.HospitalID,
			"cid":            cid,
			"file_hash":      fileHash,
			"clinical_hash":  clinicalHash,
			"data_type":      seed.DataType,
			"encounter_type": seed.EncounterType,
			"create_time":    time.Now().Format(time.RFC3339),
		})
		// 若真实 Fabric 链上资产已存在导致 CommitAsset 报错，保留既有 TxID 与区块高度，防止抹空
		if (commitErr != nil || txID == "") && recFound && existing.FabricTxID != "" {
			txID = existing.FabricTxID
			height = existing.BlockHeight
		}

		if !recFound {
			record := model.MedicalRecord{
				RecordNo:       seed.RecordNo,
				PatientID:      pat.ID,
				DoctorID:       doc.ID,
				HospitalID:     seed.HospitalID,
				DataType:       seed.DataType,
				EncounterType:  seed.EncounterType,
				DepartmentName: seed.DepartmentName,
				OnsetTime:      seed.OnsetTime,
				Duration:       seed.Duration,
				Symptoms:       seed.Symptoms,
				ChiefComplaint: seed.ChiefComplaint,
				Etiology:       seed.Etiology,
				Diagnosis:      seed.Diagnosis,
				TreatmentPlan:  seed.TreatmentPlan,
				FabricTxID:     txID,
				BlockHeight:    height,
				Status:         "COMPLETED",
				SyncStatus:     SyncStatusCompleted,
				CreatedAt:      time.Now(),
			}
			repository.DB.Create(&record)
			fileRec := model.MedicalFile{
				RecordID:     record.ID,
				FileName:     fmt.Sprintf("规范诊疗记录_%s.pdf", seed.RecordNo),
				FileType:     "pdf",
				FileSize:     uint64(len(fileData)),
				IPFSCID:      cid,
				EncryptedDEK: encDEK,
				IVNonce:      ivNonce,
				FileHash:     fileHash,
				CreatedAt:    time.Now(),
			}
			repository.DB.Create(&fileRec)
		} else {
			repository.DB.Model(&existing).Updates(map[string]interface{}{
				"fabric_tx_id":    txID,
				"block_height":    height,
				"diagnosis":       seed.Diagnosis,
				"symptoms":        seed.Symptoms,
				"chief_complaint": seed.ChiefComplaint,
				"etiology":        seed.Etiology,
				"treatment_plan":  seed.TreatmentPlan,
				"onset_time":      seed.OnsetTime,
				"duration":        seed.Duration,
				"encounter_type":  seed.EncounterType,
				"department_name": seed.DepartmentName,
			})
			var existingFile model.MedicalFile
			if repository.DB.Where("record_id = ?", existing.ID).First(&existingFile).Error == nil {
				repository.DB.Model(&existingFile).Updates(map[string]interface{}{
					"ipfs_cid":      cid,
					"encrypted_dek": encDEK,
					"iv_nonce":      ivNonce,
					"file_hash":     fileHash,
				})
			} else {
				fileRec := model.MedicalFile{
					RecordID:     existing.ID,
					FileName:     fmt.Sprintf("规范诊疗记录_%s.pdf", seed.RecordNo),
					FileType:     "pdf",
					FileSize:     uint64(len(fileData)),
					IPFSCID:      cid,
					EncryptedDEK: encDEK,
					IVNonce:      ivNonce,
					FileHash:     fileHash,
					CreatedAt:    time.Now(),
				}
				repository.DB.Create(&fileRec)
			}
		}
		log.Printf("[EnsureBaselineLedgerAnchored] ✅ 种子病历 %s 已真实加密注入 IPFS 并锚定账本: TxID=%s, Height=%d", seed.RecordNo, txID, height)
	}

	// 2. 检查其余动态上传的病历并确保上链锚定
	var list []model.MedicalRecord
	if err := repository.DB.Preload("Files").Find(&list).Error; err != nil {
		return
	}

	for _, rec := range list {
		if strings.HasPrefix(rec.RecordNo, "REC202505") {
			continue // 已在上方处理
		}
		fileHash := ""
		cid := ""
		if len(rec.Files) > 0 {
			fileHash = rec.Files[0].FileHash
			cid = rec.Files[0].IPFSCID
		}
		clinicalHash := ComputeRecordHash(&rec)

		chainData, exists := blockchain.DefaultService.QueryAsset(rec.RecordNo)
		needCommit := !exists
		if exists && chainData != nil {
			currChainHash := fmt.Sprintf("%v", chainData["file_hash"])
			if currChainHash != "" && fileHash != "" && currChainHash == fileHash && currChainHash != clinicalHash {
				needCommit = true
			}
		}

		if needCommit {
			txID, height, err := blockchain.DefaultService.CommitAsset("MEDICAL_RECORD", rec.RecordNo, map[string]interface{}{
				"record_no":     rec.RecordNo,
				"patient_id":    rec.PatientID,
				"doctor_id":     rec.DoctorID,
				"hospital_id":   rec.HospitalID,
				"cid":           cid,
				"file_hash":     fileHash,
				"clinical_hash": clinicalHash,
				"data_type":     rec.DataType,
				"create_time":   rec.CreatedAt.Format(time.RFC3339),
			})
			if err == nil {
				repository.DB.Model(&model.MedicalRecord{}).Where("id = ?", rec.ID).Updates(map[string]interface{}{
					"fabric_tx_id": txID,
					"block_height": height,
				})
			}
		}
	}
}
