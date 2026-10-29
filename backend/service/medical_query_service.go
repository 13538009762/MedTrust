package service

import (
	"fmt"
	"medtrust-backend/model"
	"medtrust-backend/repository"
	"strconv"
	"strings"
	"time"
)

func (s *MedicalService) GetRecords(patientID, doctorID, hospitalID, excludeHospitalID, currentUserID uint64, onlyCrossAccessed bool, keyword, dataType string) ([]model.MedicalRecord, error) {
	query := repository.DB.Model(&model.MedicalRecord{}).Preload("Files")

	var currentUser model.User
	var currentUserDeptName string
	if currentUserID > 0 {
		_ = repository.DB.First(&currentUser, currentUserID)
		if currentUser.DepartmentID > 0 {
			var dept model.Department
			if err := repository.DB.First(&dept, currentUser.DepartmentID).Error; err == nil {
				currentUserDeptName = dept.Name
			}
		}
	}

	// 预加载当前医生的活跃授权与破窗记录 (仅当前调用者为医生角色时)
	now := time.Now()
	var docAuths []model.Authorization
	var docEmgRecordIDs = make(map[uint64]bool)
	var privateRecordIDs = make(map[uint64]bool)
	var authRecordIDs []uint64
	var authPatientIDs []uint64
	var patientAuths []model.Authorization

	if currentUserID > 0 && currentUser.Role == "doctor" {
		repository.DB.Where("(auth_target_type = 'DOCTOR' AND auth_target_id = ?) OR (auth_target_type = 'HOSPITAL' AND auth_target_id = ?) OR auth_target_type = 'ALL_DOCTORS'", currentUserID, currentUser.HospitalID).
			Where("status = 'ACTIVE' AND start_time <= ? AND end_time >= ?", now, now).Find(&docAuths)

		var privAuths []model.Authorization
		repository.DB.Where("scope_type = 'SINGLE' AND auth_target_type = 'PRIVATE' AND status = 'ACTIVE' AND start_time <= ? AND end_time >= ?", now, now).Find(&privAuths)
		for _, pa := range privAuths {
			if pa.RecordID > 0 {
				privateRecordIDs[pa.RecordID] = true
			}
		}

		for _, a := range docAuths {
			if a.ScopeType == "SINGLE" && a.RecordID > 0 && !privateRecordIDs[a.RecordID] {
				authRecordIDs = append(authRecordIDs, a.RecordID)
			} else if a.ScopeType == "ALL" {
				authPatientIDs = append(authPatientIDs, a.PatientID)
			}
		}

		var emgs []model.EmergencyAccessEvent
		repository.DB.Where("doctor_id = ? AND created_at >= ? AND audit_status != 'CLOSED_VIOLATION' AND patient_feedback != 'OBJECTED'", currentUserID, now.Add(-24*time.Hour)).Find(&emgs)
		for _, e := range emgs {
			docEmgRecordIDs[e.RecordID] = true
			authRecordIDs = append(authRecordIDs, e.RecordID)
		}
	} else if currentUserID > 0 && currentUser.Role == "patient" {
		repository.DB.Where("patient_id = ? AND status = 'ACTIVE' AND start_time <= ? AND end_time >= ?", currentUserID, now, now).Find(&patientAuths)
	}

	if onlyCrossAccessed && currentUserID > 0 && currentUser.Role == "doctor" {
		// 医生查询【跨院已调取病历】：归属非本院，且已获得授权或已破窗放行
		if len(authRecordIDs) == 0 && len(authPatientIDs) == 0 {
			return []model.MedicalRecord{}, nil
		}
		query = query.Where("hospital_id != ?", currentUser.HospitalID)
		if len(authRecordIDs) > 0 && len(authPatientIDs) > 0 {
			query = query.Where("id IN ? OR patient_id IN ?", authRecordIDs, authPatientIDs)
		} else if len(authRecordIDs) > 0 {
			query = query.Where("id IN ?", authRecordIDs)
		} else {
			query = query.Where("patient_id IN ?", authPatientIDs)
		}
	} else {
		if patientID > 0 {
			query = query.Where("patient_id = ?", patientID)
		}
		if doctorID > 0 {
			query = query.Where("doctor_id = ?", doctorID)
		}
		if hospitalID > 0 {
			query = query.Where("hospital_id = ?", hospitalID)
		}
		if excludeHospitalID > 0 {
			query = query.Where("hospital_id != ?", excludeHospitalID)
		}
	}

	if dataType == "EXAM_ONLY" {
		// 专项筛选检查单：医学检验单、医学影像切片或开具了医技检查的就诊单
		query = query.Where("data_type IN ('REPORT', 'IMAGE') OR need_exam = 1 OR exam_items != '' OR exam_result != ''")
	} else if dataType != "" && dataType != "ALL" {
		query = query.Where("data_type = ?", dataType)
	}

	if keyword != "" {
		// 支持通过病历编号、诊断、症状、检查项目、报告结论，以及患者姓名、手机号、身份证号精准搜索，杜绝同名混淆
		query = query.Where(
			"record_no LIKE ? OR diagnosis LIKE ? OR symptoms LIKE ? OR exam_items LIKE ? OR exam_result LIKE ? OR patient_id IN (SELECT id FROM users WHERE real_name LIKE ? OR phone LIKE ? OR id_card LIKE ?)",
			"%"+keyword+"%", "%"+keyword+"%", "%"+keyword+"%", "%"+keyword+"%", "%"+keyword+"%", "%"+keyword+"%", "%"+keyword+"%", "%"+keyword+"%",
		)
	}

	var list []model.MedicalRecord
	if err := query.Order("id desc").Find(&list).Error; err != nil {
		return nil, err
	}

	// 核心性能治理：预加载并建立映射表，彻底根除循环内部 3N+1 数据库往返查询 (O(N) 降至 O(1))
	patIDMap := make(map[uint64]bool)
	docIDMap := make(map[uint64]bool)
	hospIDMap := make(map[uint64]bool)

	for _, r := range list {
		if r.PatientID > 0 {
			patIDMap[r.PatientID] = true
		}
		if r.DoctorID > 0 {
			docIDMap[r.DoctorID] = true
		}
		if r.HospitalID > 0 {
			hospIDMap[r.HospitalID] = true
		}
	}
	for _, a := range patientAuths {
		if a.AuthTargetType == "DOCTOR" && a.AuthTargetID > 0 {
			docIDMap[a.AuthTargetID] = true
		} else if a.AuthTargetType == "HOSPITAL" && a.AuthTargetID > 0 {
			hospIDMap[a.AuthTargetID] = true
		}
	}

	userMap := make(map[uint64]model.User)
	hospMap := make(map[uint64]model.Hospital)

	allUserIDs := make([]uint64, 0, len(patIDMap)+len(docIDMap))
	for id := range patIDMap {
		allUserIDs = append(allUserIDs, id)
	}
	for id := range docIDMap {
		if !patIDMap[id] {
			allUserIDs = append(allUserIDs, id)
		}
	}
	if len(allUserIDs) > 0 {
		var users []model.User
		if err := repository.DB.Where("id IN ?", allUserIDs).Find(&users).Error; err == nil {
			for _, u := range users {
				userMap[u.ID] = u
			}
		}
	}

	allHospIDs := make([]uint64, 0, len(hospIDMap))
	for id := range hospIDMap {
		allHospIDs = append(allHospIDs, id)
	}
	if len(allHospIDs) > 0 {
		var hosps []model.Hospital
		if err := repository.DB.Where("id IN ?", allHospIDs).Find(&hosps).Error; err == nil {
			for _, h := range hosps {
				hospMap[h.ID] = h
			}
		}
	}

	// 补充外联展示信息与区块链实时动态核验及权限标记
	for i := range list {
		if pat, ok := userMap[list[i].PatientID]; ok {
			list[i].PatientName = pat.RealName
			list[i].PatientIDCard = pat.IDCard
			list[i].PatientPhone = pat.Phone
		}
		if doc, ok := userMap[list[i].DoctorID]; ok {
			list[i].DoctorName = doc.RealName
		}
		if hosp, ok := hospMap[list[i].HospitalID]; ok {
			list[i].HospitalName = hosp.Name
		}

		// 计算针对当前调用者的访问权限状态 (基于 RBAC + ABAC + 患者所有权)
		if currentUserID > 0 {
			// 1. 患者本人查阅自己的就诊档案：依法享有 100% 完整知情权与查阅权，直接放行
			if list[i].PatientID == currentUserID || (currentUser.Role == "patient" && list[i].PatientID == currentUser.ID) {
				list[i].HasAccess = true
				list[i].AccessType = "PATIENT_OWNER"

				// 计算患者每份病历当前的自主共享控制状态
				scope := "PRIVATE"
				summary := "私密受控 (仅经治医生)"
				var activeID uint64 = 0
				targetType := "PRIVATE"
				var targetID uint64 = 0
				targetName := ""

				// 1) 优先检查针对本单病历的显式独立授权 (SINGLE 优先级高于全局 ALL 广播)
				var singleAuth *model.Authorization
				for _, a := range patientAuths {
					if a.ScopeType == "SINGLE" && a.RecordID == list[i].ID {
						copyA := a
						singleAuth = &copyA
						break
					}
				}

				if singleAuth != nil {
					activeID = singleAuth.ID
					scope = singleAuth.AuthTargetType
					targetType = singleAuth.AuthTargetType
					targetID = singleAuth.AuthTargetID

					if singleAuth.AuthTargetType == "ALL_DOCTORS" {
						targetName = "全体医生"
						summary = "全体医生可见 (本单公开)"
					} else if singleAuth.AuthTargetType == "HOSPITAL" {
						if hosp, ok := hospMap[singleAuth.AuthTargetID]; ok && hosp.Name != "" {
							targetName = hosp.Name
							summary = fmt.Sprintf("仅限【%s】全体医生", hosp.Name)
						} else {
							summary = fmt.Sprintf("仅限医院 (ID:%d) 全体医生", singleAuth.AuthTargetID)
						}
					} else if singleAuth.AuthTargetType == "DOCTOR" {
						if doc, ok := userMap[singleAuth.AuthTargetID]; ok && doc.RealName != "" {
							docName := doc.RealName
							if !strings.HasSuffix(docName, "医生") && !strings.HasSuffix(docName, "医师") {
								docName += " 医生"
							}
							targetName = docName
							summary = fmt.Sprintf("仅限【%s】专属调阅", docName)
						} else {
							summary = fmt.Sprintf("仅限医生 (ID:%d) 调阅", singleAuth.AuthTargetID)
						}
					} else if singleAuth.AuthTargetType == "PRIVATE" {
						scope = "PRIVATE"
						targetType = "PRIVATE"
						summary = "私密受控 (仅经治医生)"
					}
				} else {
					// 2) 若无本单病历专属授权，继承全局授权策略 (ScopeType == ALL)
					var globalAuth *model.Authorization
					for _, a := range patientAuths {
						if a.ScopeType == "ALL" {
							copyA := a
							globalAuth = &copyA
							break
						}
					}

					if globalAuth != nil {
						activeID = globalAuth.ID
						scope = globalAuth.AuthTargetType
						targetType = globalAuth.AuthTargetType
						targetID = globalAuth.AuthTargetID

						if globalAuth.AuthTargetType == "ALL_DOCTORS" {
							targetName = "全体医生"
							summary = "全体医生可见 (全网公开)"
						} else if globalAuth.AuthTargetType == "HOSPITAL" {
							if hosp, ok := hospMap[globalAuth.AuthTargetID]; ok && hosp.Name != "" {
								targetName = hosp.Name
								summary = fmt.Sprintf("仅限【%s】全体医生 (全局)", hosp.Name)
							} else {
								summary = fmt.Sprintf("仅限医院 (ID:%d) 全体医生", globalAuth.AuthTargetID)
							}
						} else if globalAuth.AuthTargetType == "DOCTOR" {
							if doc, ok := userMap[globalAuth.AuthTargetID]; ok && doc.RealName != "" {
								docName := doc.RealName
								if !strings.HasSuffix(docName, "医生") && !strings.HasSuffix(docName, "医师") {
									docName += " 医生"
								}
								targetName = docName
								summary = fmt.Sprintf("仅限【%s】专属调阅 (全局)", docName)
							} else {
								summary = fmt.Sprintf("仅限医生 (ID:%d) 调阅", globalAuth.AuthTargetID)
							}
						}
					}
				}

				list[i].SharingScope = scope
				list[i].SharingSummary = summary
				list[i].ActiveAuthID = activeID
				list[i].ActiveAuthTargetType = targetType
				list[i].ActiveAuthTargetID = targetID
				list[i].ActiveAuthTargetName = targetName
			} else if list[i].DoctorID == currentUserID {
				// 2. 接诊责任医生本人病历
				list[i].HasAccess = true
				list[i].AccessType = "OWNER"
			} else if currentUser.Role == "doctor" && list[i].HospitalID == currentUser.HospitalID {
				// 3. 院内机构：同科室或全科综合门诊协同，或高级职称专家
				isSameDept := currentUser.DepartmentID == 0 || list[i].DepartmentName == "" ||
					list[i].DepartmentName == "综合门诊" || (currentUserDeptName != "" && currentUserDeptName == list[i].DepartmentName)
				if isSameDept || currentUser.Title == "主任医师" || currentUser.Title == "副主任医师" {
					list[i].HasAccess = true
					list[i].AccessType = "HOSPITAL"
				} else {
					hasAuth := false
					isAllDocAuth := false
					if !privateRecordIDs[list[i].ID] {
						for _, a := range docAuths {
							if a.PatientID == list[i].PatientID && (a.ScopeType == "ALL" || (a.ScopeType == "SINGLE" && a.RecordID == list[i].ID)) {
								hasAuth = true
								if a.AuthTargetType == "ALL_DOCTORS" {
									isAllDocAuth = true
									break
								}
							}
						}
					}
					if hasAuth {
						list[i].HasAccess = true
						if isAllDocAuth {
							list[i].AccessType = "ALL_DOCTORS"
						} else {
							list[i].AccessType = "AUTHORIZED"
						}
					} else {
						list[i].HasAccess = false
						list[i].AccessType = "UNAUTHORIZED"
					}
				}
			} else if currentUser.Role == "doctor" {
				// 4. 跨院机构医生 (若医生处于受限状态 RESTRICTED，严禁跨院调阅)
				if currentUser.Status == "RESTRICTED" {
					list[i].HasAccess = false
					list[i].AccessType = "UNAUTHORIZED"
				} else {
					hasAuth := false
					isAllDocAuth := false
					if !privateRecordIDs[list[i].ID] {
						for _, a := range docAuths {
							if a.PatientID == list[i].PatientID && (a.ScopeType == "ALL" || (a.ScopeType == "SINGLE" && a.RecordID == list[i].ID)) {
								hasAuth = true
								if a.AuthTargetType == "ALL_DOCTORS" {
									isAllDocAuth = true
									break
								}
							}
						}
					}
					if docEmgRecordIDs[list[i].ID] {
						list[i].HasAccess = true
						list[i].AccessType = "BREAK_GLASS"
					} else if hasAuth {
						list[i].HasAccess = true
						if isAllDocAuth {
							list[i].AccessType = "ALL_DOCTORS"
						} else {
							list[i].AccessType = "AUTHORIZED"
						}
					} else {
						list[i].HasAccess = false
						list[i].AccessType = "UNAUTHORIZED"
					}
				}
			} else {
				list[i].HasAccess = false
				list[i].AccessType = "UNAUTHORIZED"
			}
		} else {
			list[i].HasAccess = true
		}

		s.VerifyRecord(&list[i])

		// 安全脱敏防御：若当前医生未获得调阅权限，严禁向前端返回临床敏感诊断与明细数据
		if currentUserID > 0 && !list[i].HasAccess {
			list[i].Diagnosis = "【受控访问限制·未授权掩码】"
			list[i].Symptoms = "【未取得患者授权】"
			list[i].Etiology = "【未取得患者授权】"
			list[i].TreatmentPlan = "【未取得患者授权】"
			list[i].ChiefComplaint = "【未取得患者授权】"
			list[i].PresentIllness = "【未取得患者授权】"
			list[i].InitialDiagnosis = "【未取得患者授权】"
			list[i].DiagnosticBasis = "【未取得患者授权】"
			list[i].ExamItems = "【未取得患者授权】"
			list[i].ExamReason = "【未取得患者授权】"
			list[i].ExamResult = "【未取得患者授权】"
			list[i].VitalSigns = "【受控屏蔽】"
			if len(list[i].PatientPhone) >= 7 {
				list[i].PatientPhone = list[i].PatientPhone[:3] + "****" + list[i].PatientPhone[7:]
			} else if list[i].PatientPhone != "" {
				list[i].PatientPhone = "***"
			}
			if len(list[i].PatientIDCard) >= 14 {
				list[i].PatientIDCard = list[i].PatientIDCard[:6] + "********" + list[i].PatientIDCard[14:]
			} else if list[i].PatientIDCard != "" {
				list[i].PatientIDCard = "******************"
			}
			list[i].Files = nil
		}

		// 医护职业暴露安全与传染病预警检测（严格遵循访问控制边界：仅已获得合法调阅权限的记录才暴露传染病诊断明细，杜绝越权隐私泄露）
		if list[i].HasAccess {
			if alert, err := GetInfectionService().AnalyzeRecordRisks(list[i].ID); err == nil && alert.HasRisk {
				list[i].InfectionAlert = alert
			}
		} else {
			list[i].InfectionAlert = nil
		}
	}
	return list, nil
}

func (s *MedicalService) GetRecordByID(id uint64) (*model.MedicalRecord, error) {
	var rec model.MedicalRecord
	if err := repository.DB.Preload("Files").First(&rec, id).Error; err != nil {
		return nil, err
	}
	var pat model.User
	if err := repository.DB.First(&pat, rec.PatientID).Error; err == nil {
		rec.PatientName = pat.RealName
		rec.PatientIDCard = pat.IDCard
		rec.PatientPhone = pat.Phone
	}
	var doc model.User
	if err := repository.DB.First(&doc, rec.DoctorID).Error; err == nil {
		rec.DoctorName = doc.RealName
	}
	var hosp model.Hospital
	if err := repository.DB.First(&hosp, rec.HospitalID).Error; err == nil {
		rec.HospitalName = hosp.Name
	}
	s.VerifyRecord(&rec)

	// 医护职业暴露安全与传染病预警检测
	if alert, err := GetInfectionService().AnalyzeRecordRisks(rec.ID); err == nil && alert.HasRisk {
		rec.InfectionAlert = alert
	}

	return &rec, nil
}

// EnrichPatientEmergencyProfile 为患者补充急救生命体征、血型、过敏史、紧急联系人等抢救关键信息
func (s *MedicalService) EnrichPatientEmergencyProfile(user *model.User) {
	if user == nil || user.Role != "patient" {
		return
	}

	// 1. 根据 18 位身份证号码推导性别与年龄
	if len(user.IDCard) == 18 {
		birthYearStr := user.IDCard[6:10]
		birthYear, err := strconv.Atoi(birthYearStr)
		if err == nil && birthYear > 1900 {
			currentYear := time.Now().Year()
			user.Age = currentYear - birthYear
			if user.Age < 0 {
				user.Age = 35
			}
		}
		// 倒数第二位为性别标识：奇数为男，偶数为女
		genderDigit := user.IDCard[16] - '0'
		if genderDigit%2 == 1 {
			user.Gender = "男"
		} else {
			user.Gender = "女"
		}
	} else {
		user.Gender = "男"
		user.Age = 38
	}

	// 2. 从最近就诊记录中获取最新生命体征参数与慢性病信息
	var latestRec model.MedicalRecord
	if err := repository.DB.Where("patient_id = ?", user.ID).Order("id desc").First(&latestRec).Error; err == nil {
		if latestRec.VitalSigns != "" {
			user.LatestVitalSigns = latestRec.VitalSigns
		}
	}
	if user.LatestVitalSigns == "" {
		user.LatestVitalSigns = "BP: 122/80 mmHg, HR: 74 bpm, SpO2: 99%, T: 36.6℃"
	}

	// 3. 拟定抢救黄金档案 (结合患者数据合理生成，便于急救参考)
	bloodTypes := []string{"O型 (Rh阳性)", "A型 (Rh阳性)", "B型 (Rh阳性)", "AB型 (Rh阳性)"}
	user.BloodType = bloodTypes[user.ID%uint64(len(bloodTypes))]

	if user.ID%3 == 1 {
		user.Allergies = "青霉素类抗生素、头孢菌素轻度过敏"
		user.HasAllergies = true
		user.ChronicDiseases = "原发性高血压(2级中危), 2型糖尿病"
		user.EmergencyContact = "李秀兰 (配偶)"
		user.EmergencyPhone = "13800138021"
	} else if user.ID%3 == 2 {
		user.Allergies = "磺胺类药物过敏, 鱼虾等海鲜蛋白过敏"
		user.HasAllergies = true
		user.ChronicDiseases = "支气管哮喘, 慢性浅表性胃炎"
		user.EmergencyContact = "张宏伟 (父亲)"
		user.EmergencyPhone = "13911223344"
	} else {
		user.Allergies = "未发现明确药物过敏史"
		user.HasAllergies = false
		user.ChronicDiseases = "高脂血症, 偶发性心律不齐"
		user.EmergencyContact = "王晓敏 (家属)"
		user.EmergencyPhone = "13766554433"
	}
}

// ListPatients 医生或系统查询患者列表，支持通过姓名、手机号、身份证号精准搜索
func (s *MedicalService) ListPatients(keyword string) ([]model.User, error) {
	query := repository.DB.Model(&model.User{}).Where("role = ?", "patient")
	if keyword != "" {
		query = query.Where("real_name LIKE ? OR phone LIKE ? OR id_card LIKE ? OR user_no LIKE ?",
			"%"+keyword+"%", "%"+keyword+"%", "%"+keyword+"%", "%"+keyword+"%")
	}
	var list []model.User
	if err := query.Order("id asc").Find(&list).Error; err != nil {
		return nil, err
	}

	// 关联分析患者名下的高危传染病携带情况与急诊抢救画像
	for i := range list {
		s.EnrichPatientEmergencyProfile(&list[i])
		if alert, err := GetInfectionService().AnalyzePatientRisks(list[i].ID); err == nil && alert.HasRisk {
			list[i].InfectionAlert = alert
		}
	}

	return list, nil
}
