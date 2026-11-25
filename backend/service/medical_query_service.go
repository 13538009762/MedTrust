package service

import (
	"fmt"
	"medtrust-backend/model"
	"medtrust-backend/repository"
	"strconv"
	"strings"
	"time"
)

// MaxRecordPageSize 病历列表单页最大条数，防止客户端传 page_size=999999 拖垮库与响应体
const MaxRecordPageSize = 100

func (s *MedicalService) GetRecords(patientID, doctorID, hospitalID, excludeHospitalID, currentUserID uint64, onlyCrossAccessed bool, keyword, dataType string, page, pageSize int) ([]model.MedicalRecord, error) {
	query := repository.DefaultMedicalRepository.NewRecordQuery()

	var currentUser model.User
	var currentUserDeptName string
	if currentUserID > 0 {
		userPtr, _ := repository.DefaultUserRepository.FindByID(currentUserID)
		if userPtr != nil {
			currentUser = *userPtr
		}
		if currentUser.DepartmentID > 0 {
			deptPtr, deptErr := repository.DefaultUserRepository.FindDepartmentByID(currentUser.DepartmentID)
			if deptErr == nil && deptPtr != nil {
				currentUserDeptName = deptPtr.Name
			}
		}
	}

	// 预加载当前医生的活跃授权与破窗记录 (仅当前调用者为医生角色时)
	now := time.Now()
	var docAuths []model.Authorization
	var docEmgRecordIDs = make(map[uint64]bool)
	var privAuthMap = make(map[uint64]model.Authorization)
	var authRecordIDs []uint64
	var authPatientIDs []uint64
	var patientAuths []model.Authorization

	if currentUserID > 0 && currentUser.Role == "doctor" {
		rawDocAuths, _ := repository.DefaultAuthPolicyRepository.ListAuthorizationsForDoctor(currentUserID, currentUser.HospitalID, "ACTIVE")
		for _, a := range rawDocAuths {
			if !a.StartTime.After(now) && a.EndTime.After(now) {
				docAuths = append(docAuths, a)
			}
		}

		privAuths, _ := repository.DefaultAuthPolicyRepository.ListPrivAuthsByTime(now, now)
		for _, pa := range privAuths {
			if pa.RecordID > 0 {
				if existing, ok := privAuthMap[pa.RecordID]; !ok || repository.IsAuthorizationNewerThan(pa, existing) {
					privAuthMap[pa.RecordID] = pa
				}
			}
		}

		for _, a := range docAuths {
			if a.ScopeType == "SINGLE" && a.RecordID > 0 {
				if priv, hasPriv := privAuthMap[a.RecordID]; !hasPriv || repository.IsAuthorizationNewerThan(a, priv) {
					authRecordIDs = append(authRecordIDs, a.RecordID)
				}
			} else if a.ScopeType == "ALL" {
				authPatientIDs = append(authPatientIDs, a.PatientID)
			}
		}

		emgs, _ := repository.DefaultEmergencyRepository.ListRecentActiveEmgs(currentUserID, now.Add(-24*time.Hour))
		for _, e := range emgs {
			docEmgRecordIDs[e.RecordID] = true
			authRecordIDs = append(authRecordIDs, e.RecordID)
		}
	} else if currentUserID > 0 && currentUser.Role == "patient" {
		patientAuths, _ = repository.DefaultAuthPolicyRepository.ListPatientAuthsAtTime(currentUserID, now, now)
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
		if currentUserID > 0 && currentUser.Role == "supervisor" {
			// 监管员仅可按病历编号、患者姓名或完整手机号/身份证号检索，严禁对临床诊断明文做模糊盲注探测
			query = query.Where(
				"record_no LIKE ? OR patient_id IN (SELECT id FROM users WHERE real_name LIKE ? OR phone = ? OR id_card = ?)",
				"%"+keyword+"%", "%"+keyword+"%", keyword, keyword,
			)
		} else {
			// 支持通过病历编号、诊断、症状、检查项目、报告结论，以及患者姓名、完整手机号/身份证号精确检索（防掩码段逐位盲猜）
			query = query.Where(
				"record_no LIKE ? OR diagnosis LIKE ? OR symptoms LIKE ? OR exam_items LIKE ? OR exam_result LIKE ? OR patient_id IN (SELECT id FROM users WHERE real_name LIKE ? OR phone = ? OR id_card = ?)",
				"%"+keyword+"%", "%"+keyword+"%", "%"+keyword+"%", "%"+keyword+"%", "%"+keyword+"%", "%"+keyword+"%", keyword, keyword,
			)
		}
	}

	var list []model.MedicalRecord
	db := query.Order("id desc")
	// 分页下推到 SQL：pageSize>0 时按页取，避免把全表拉进内存后再截断。
	// pageSize<=0 表示内部调用方（AI 诊断、导出等）需要全量，此时不施加 LIMIT。
	if pageSize > 0 {
		if pageSize > MaxRecordPageSize {
			pageSize = MaxRecordPageSize
		}
		if page < 1 {
			page = 1
		}
		db = db.Offset((page - 1) * pageSize).Limit(pageSize)
	}
	if err := db.Find(&list); err != nil {
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
	if currentUser.HospitalID > 0 {
		hospIDMap[currentUser.HospitalID] = true
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
		users, usrErr := repository.DefaultUserRepository.FindByIDs(allUserIDs)
		if usrErr == nil {
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
		hosps, hospErr := repository.DefaultUserRepository.FindHospitalsByIDs(allHospIDs)
		if hospErr == nil {
			for _, h := range hosps {
				hospMap[h.ID] = h
			}
		}
	}

	// 补充外联展示信息与区块链实时动态核验及权限标记
	filtered := make([]model.MedicalRecord, 0, len(list))
	for i := range list {
		if pat, ok := userMap[list[i].PatientID]; ok {
			list[i].PatientName = pat.RealName
			list[i].PatientIDCard = pat.IDCard
			list[i].PatientPhone = pat.Phone
		}
		rawPatPhone := list[i].PatientPhone
		rawPatIDCard := list[i].PatientIDCard
		if doc, ok := userMap[list[i].DoctorID]; ok {
			list[i].DoctorName = doc.RealName
		}
		if hosp, ok := hospMap[list[i].HospitalID]; ok {
			list[i].HospitalName = hosp.Name
		}

		// 计算针对当前调用者的访问权限状态 (基于 RBAC + ABAC + 患者所有权 + PRIVATE 隐私锁 + 动态风控)
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
			} else if currentUser.Role == "doctor" {
				docHospDisabled := currentUser.HospitalID > 0 && hospMap[currentUser.HospitalID].ID > 0 && hospMap[currentUser.HospitalID].Status == 0
				recHospDisabled := list[i].HospitalID > 0 && hospMap[list[i].HospitalID].ID > 0 && hospMap[list[i].HospitalID].Status == 0

				if currentUser.Status == "DISABLED" || docHospDisabled || recHospDisabled {
					list[i].HasAccess = false
					list[i].AccessType = "UNAUTHORIZED"
				} else if list[i].DoctorID == currentUserID {
					// 2. 接诊责任医生本人病历
					list[i].HasAccess = true
					list[i].AccessType = "OWNER"
				} else if list[i].HospitalID == currentUser.HospitalID {
					// 3. 院内机构：严格尊重患者 PRIVATE 隐私锁，无隐私锁时才允许同科室或高级职称专家协同
					hasAuth := false
					isAllDocAuth := false
					priv, hasPriv := privAuthMap[list[i].ID]
					for _, a := range docAuths {
						if a.PatientID == list[i].PatientID && (a.ScopeType == "ALL" || (a.ScopeType == "SINGLE" && a.RecordID == list[i].ID)) {
							if hasPriv && !repository.IsAuthorizationNewerThan(a, priv) {
								continue
							}
							hasAuth = true
							if a.AuthTargetType == "ALL_DOCTORS" {
								isAllDocAuth = true
								break
							}
						}
					}
					isSameDept := currentUser.DepartmentID == 0 || list[i].DepartmentName == "" ||
						list[i].DepartmentName == "综合门诊" || (currentUserDeptName != "" && currentUserDeptName == list[i].DepartmentName)
					isSenior := currentUser.Title == "主任医师" || currentUser.Title == "副主任医师"

					if currentUser.Status != "RESTRICTED" && docEmgRecordIDs[list[i].ID] {
						list[i].HasAccess = true
						list[i].AccessType = "BREAK_GLASS"
					} else if !hasPriv && (isSameDept || isSenior) {
						list[i].HasAccess = true
						list[i].AccessType = "HOSPITAL"
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
				} else {
					// 4. 跨院机构医生 (若医生处于受限状态 RESTRICTED，严禁跨院调阅)
					if currentUser.Status == "RESTRICTED" {
						list[i].HasAccess = false
						list[i].AccessType = "UNAUTHORIZED"
					} else {
						hasAuth := false
						isAllDocAuth := false
						priv, hasPriv := privAuthMap[list[i].ID]
						for _, a := range docAuths {
							if a.PatientID == list[i].PatientID && (a.ScopeType == "ALL" || (a.ScopeType == "SINGLE" && a.RecordID == list[i].ID)) {
								if hasPriv && !repository.IsAuthorizationNewerThan(a, priv) {
									continue
								}
								hasAuth = true
								if a.AuthTargetType == "ALL_DOCTORS" {
									isAllDocAuth = true
									break
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
				}

				// 对齐 AccessEngine 动态多因子风控：非经治本人且非破窗通道达到 HIGH 风险时阻断明文展示
				if list[i].HasAccess && list[i].DoctorID != currentUserID && list[i].AccessType != "BREAK_GLASS" {
					if DefaultAccessEngine != nil && DefaultAccessEngine.GetRiskEngine() != nil {
						eval := DefaultAccessEngine.GetRiskEngine().Evaluate(currentUser.ID, list[i].PatientID, list[i].HospitalID != currentUser.HospitalID, true, false)
						if eval.Level == "HIGH" {
							list[i].HasAccess = false
							list[i].AccessType = "UNAUTHORIZED"
						}
					}
				}
			} else {
				list[i].HasAccess = false
				list[i].AccessType = "UNAUTHORIZED"
			}
		} else {
			list[i].HasAccess = true
		}

		// 防范关键字布尔盲注探测预言机 (V-03)：若调用者对该病历无查阅权限，仅当 keyword 命中公开编号、姓名或完整手机号/身份证号时才允许返回掩码卡片；
		// 若仅因 SQL LIKE 命中了未授权临床明文（诊断/主诉/检验），直接从结果集剥离，阻断盲猜推断
		if keyword != "" && currentUserID > 0 && !list[i].HasAccess {
			kwLower := strings.ToLower(keyword)
			matchedPublicMeta := strings.Contains(strings.ToLower(list[i].RecordNo), kwLower) ||
				strings.Contains(strings.ToLower(list[i].PatientName), kwLower) ||
				(rawPatPhone != "" && rawPatPhone == keyword) ||
				(rawPatIDCard != "" && rawPatIDCard == keyword)
			if !matchedPublicMeta {
				continue
			}
		}

		// WP-12: 列表端点不再执行 O(N) Fabric+IPFS 同步核验（每条记录一次链查询+密文拉取+解密），
		// 改为基于索引字段的 O(1) 轻量状态推导：
		//   - 未上链且流转中的就诊记录不误判为篡改（与 VerifyRecord 流转分支语义一致）
		//   - 已上链记录以持久化 IntegrityStatus 为准（由单记录核验端点回写）
		//   - 上链但从未核验过的记录按账本锚定状态给出非篡改指纹
		if r := &list[i]; r.FabricTxID == "" && (r.Status == "WAITING_EXAM" || r.Status == "PROCESSING_EXAM" || r.Status == "EXAM_COMPLETED" || r.Status == "INITIAL_DIAGNOSIS") {
			r.Verified = false
			r.IsTampered = false
			r.ChainHash = ""
			r.TamperReason = ""
		} else if r.FabricTxID == "" {
			r.Verified = false
			r.IsTampered = false
			r.ChainHash = ""
			r.TamperReason = "【区块链存证提示】该病历尚未在区块链分布式账本形成存证锚定"
		} else if r.IntegrityStatus == "TAMPERED" {
			r.Verified = false
			r.IsTampered = true
			r.TamperReason = "【区块链安全告警】该病历最近一次单记录核验检测到篡改请点击单条核验重新验真"
		} else {
			chainAnchored := r.FabricTxID != ""
			r.Verified = chainAnchored && r.IntegrityStatus != "TAMPERED"
			r.IsTampered = false
			r.TamperReason = ""
		}

		// WP-08: 列表端点无条件统一脱敏患者身份证与手机号（即使 HasAccess == true），
		// 降低批量表格接口的 PII 拖库风险；完整明文仅在单记录详情/授权通道中按需返回
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
			list[i].Files = nil
		}

		list[i].InfectionAlert = nil
		filtered = append(filtered, list[i])
	}
	list = filtered

	// WP-13 / Batch 2: 按患者维度批量聚合职业暴露传染病预警（消除原循环内 3N 次数据库往返，并按当前调用者已授权病历集合脱敏跨院未授权来源单号）
	authRecordIDsByPatient := make(map[uint64]map[uint64]bool)
	for i := range list {
		if list[i].HasAccess && list[i].PatientID > 0 {
			if authRecordIDsByPatient[list[i].PatientID] == nil {
				authRecordIDsByPatient[list[i].PatientID] = make(map[uint64]bool)
			}
			authRecordIDsByPatient[list[i].PatientID][list[i].ID] = true
		}
	}
	if len(authRecordIDsByPatient) > 0 {
		infectionSvc := GetInfectionService()
		alertCache := make(map[uint64]*model.InfectionRiskAlert, len(authRecordIDsByPatient))
		for patID, authSet := range authRecordIDsByPatient {
			if alert, err := infectionSvc.AnalyzePatientRisksWithScope(patID, authSet); err == nil && alert != nil && alert.HasRisk {
				alertCache[patID] = alert
			}
		}
		for i := range list {
			if list[i].HasAccess {
				if alert, ok := alertCache[list[i].PatientID]; ok {
					list[i].InfectionAlert = alert
				}
			}
		}
	}
	return list, nil
}

func (s *MedicalService) GetRecordByID(id uint64) (*model.MedicalRecord, error) {
	recPtr, recErr := repository.DefaultMedicalRepository.FindRecordByID(id)
	if recErr != nil {
		return nil, recErr
	}
	rec := *recPtr

	patPtr, patErr := repository.DefaultUserRepository.FindByID(rec.PatientID)
	if patErr == nil && patPtr != nil {
		rec.PatientName = patPtr.RealName
		rec.PatientIDCard = patPtr.IDCard
		rec.PatientPhone = patPtr.Phone
	}
	docPtr, docErr := repository.DefaultUserRepository.FindByID(rec.DoctorID)
	if docErr == nil && docPtr != nil {
		rec.DoctorName = docPtr.RealName
	}
	hospPtr, hospErr := repository.DefaultUserRepository.FindHospitalByIDFull(rec.HospitalID)
	if hospErr == nil && hospPtr != nil {
		rec.HospitalName = hospPtr.Name
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
	s.enrichPatientIdentityProfile(user)

	// 2. 从最近就诊记录中获取最新生命体征参数与慢性病信息
	// WP-13: 外部批量调用方请优先使用 ListPatients 的批量预加载版本，避免逐患者 N+1 查询
	var latestRec model.MedicalRecord
	recPtr, lrErr := repository.DefaultMedicalRepository.FindLatestByPatient(user.ID)
	if lrErr == nil && recPtr != nil {
		latestRec = *recPtr
	}
	if latestRec.VitalSigns != "" {
		user.LatestVitalSigns = latestRec.VitalSigns
	}

	// 3. WP-07: 仅展示真实持久化的抢救档案字段，严禁伪造 mock 过敏史/血型/紧急联系人
	if user.LatestVitalSigns == "" {
		user.LatestVitalSigns = "未登记"
	}
	if user.BloodType == "" {
		user.BloodType = "未登记"
	}
	if strings.TrimSpace(user.Allergies) == "" {
		user.Allergies = "未登记"
		user.HasAllergies = false
	} else {
		user.HasAllergies = true
	}
	if strings.TrimSpace(user.ChronicDiseases) == "" {
		user.ChronicDiseases = "未登记"
	}
	if strings.TrimSpace(user.EmergencyContact) == "" {
		user.EmergencyContact = "未登记"
	}
	if strings.TrimSpace(user.EmergencyPhone) == "" {
		user.EmergencyPhone = "未登记"
	}
}

// enrichPatientIdentityProfile WP-13: 零 SQL 身份画像推导（身份证号推性别/年龄），供批量与单查路径共用
func (s *MedicalService) enrichPatientIdentityProfile(user *model.User) {
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
}

// EnrichPatientEmergencyProfileWithVitals WP-13: 基于预加载的最新生命体征映射的零 SQL 画像增强版本
func (s *MedicalService) EnrichPatientEmergencyProfileWithVitals(user *model.User, latestVitalSigns string) {
	if user == nil || user.Role != "patient" {
		return
	}

	// 1. 零 SQL 身份画像推导
	s.enrichPatientIdentityProfile(user)

	// 2. 生命体征由批量预加载映射传入，本函数全程零 SQL
	if latestVitalSigns != "" {
		user.LatestVitalSigns = latestVitalSigns
	}

	// 3. WP-07: 仅展示真实持久化的抢救档案字段，严禁伪造 mock 过敏史/血型/紧急联系人
	if user.LatestVitalSigns == "" {
		user.LatestVitalSigns = "未登记"
	}
	if user.BloodType == "" {
		user.BloodType = "未登记"
	}
	if strings.TrimSpace(user.Allergies) == "" {
		user.Allergies = "未登记"
		user.HasAllergies = false
	} else {
		user.HasAllergies = true
	}
	if strings.TrimSpace(user.ChronicDiseases) == "" {
		user.ChronicDiseases = "未登记"
	}
	if strings.TrimSpace(user.EmergencyContact) == "" {
		user.EmergencyContact = "未登记"
	}
	if strings.TrimSpace(user.EmergencyPhone) == "" {
		user.EmergencyPhone = "未登记"
	}
}

// ListPatients 医生或系统查询患者列表，支持通过姓名、手机号、身份证号精准搜索
func (s *MedicalService) ListPatients(keyword string) ([]model.User, error) {
	users, usrErr := repository.DefaultUserRepository.ListPatients(keyword)
	if usrErr != nil {
		return nil, usrErr
	}

	// WP-13: 一次批量查询预加载每位患者最新一条含生命体征的病历，消除循环内逐患者 SQL (N+1 → 1)
	patientIDs := make([]uint64, 0, len(users))
	for i := range users {
		patientIDs = append(patientIDs, users[i].ID)
	}
	latestVitalsMap := make(map[uint64]string)
	if len(patientIDs) > 0 {
		vitalsRecs, vitalsErr := repository.DefaultMedicalRepository.FindVitalsByPatientIDs(patientIDs)
		if vitalsErr == nil {
			for _, r := range vitalsRecs {
				if _, ok := latestVitalsMap[r.PatientID]; !ok {
					latestVitalsMap[r.PatientID] = r.VitalSigns
				}
			}
		}
	}

	// WP-07: 批量患者目录仅展示身份识别画像（性别/年龄/最新生命体征等抢救索引信息）；
	// 严禁在批量列表中逐个调用 AnalyzePatientRisks 暴露 HIV/HBV/梅毒等传染病隐私（同时消除 2N 次全病历 SQL 扫描）
	for i := range users {
		s.EnrichPatientEmergencyProfileWithVitals(&users[i], latestVitalsMap[users[i].ID])
		users[i].InfectionAlert = nil
	}

	return users, nil
}
