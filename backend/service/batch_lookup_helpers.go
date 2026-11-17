package service

import (
	"medtrust-backend/model"
	"medtrust-backend/repository"
)

// batch_lookup_helpers.go WP-13: service 层批量预加载辅助函数
// 消除列表端点逐条 FindByID 造成的 SQL N+1，统一改为 WHERE id IN 批量查询 + 内存映射组装。

// dedupeIDsService 清洗并去重 ID 列表 (过滤 0 值)
func dedupeIDsService(ids []uint64) []uint64 {
	seen := make(map[uint64]bool, len(ids))
	result := make([]uint64, 0, len(ids))
	for _, id := range ids {
		if id > 0 && !seen[id] {
			seen[id] = true
			result = append(result, id)
		}
	}
	return result
}

// loadUsersBatchService 批量拉取用户并建立 ID 映射表
func loadUsersBatchService(ids []uint64) map[uint64]model.User {
	userMap := make(map[uint64]model.User)
	uniqueIDs := dedupeIDsService(ids)
	if len(uniqueIDs) == 0 {
		return userMap
	}
	if users, err := repository.DefaultUserRepository.FindByIDs(uniqueIDs); err == nil {
		for _, u := range users {
			userMap[u.ID] = u
		}
	}
	return userMap
}

// loadHospitalsBatchService 批量拉取医疗机构并建立 ID 映射表
func loadHospitalsBatchService(ids []uint64) map[uint64]model.Hospital {
	hospMap := make(map[uint64]model.Hospital)
	uniqueIDs := dedupeIDsService(ids)
	if len(uniqueIDs) == 0 {
		return hospMap
	}
	if hosps, err := repository.DefaultUserRepository.FindHospitalsByIDs(uniqueIDs); err == nil {
		for _, h := range hosps {
			hospMap[h.ID] = h
		}
	}
	return hospMap
}

// loadRecordsBatchService 批量拉取病历并建立 ID 映射表
func loadRecordsBatchService(ids []uint64) map[uint64]model.MedicalRecord {
	recMap := make(map[uint64]model.MedicalRecord)
	uniqueIDs := dedupeIDsService(ids)
	if len(uniqueIDs) == 0 {
		return recMap
	}
	if recs, err := repository.DefaultMedicalRepository.FindRecordsByIDs(uniqueIDs); err == nil {
		for _, r := range recs {
			recMap[r.ID] = r
		}
	}
	return recMap
}

// maskPhoneService 手机号前三后四脱敏 (与 controller 层 maskPhone 保持一致)
func maskPhoneService(phone string) string {
	runes := []rune(phone)
	if len(runes) <= 7 {
		return phone
	}
	return string(runes[:3]) + "****" + string(runes[len(runes)-4:])
}

// maskIDCardService 居民身份证号前六后四脱敏 (与 controller 层 maskIDCard 保持一致)
func maskIDCardService(idCard string) string {
	runes := []rune(idCard)
	if len(runes) <= 10 {
		return idCard
	}
	return string(runes[:6]) + "********" + string(runes[len(runes)-4:])
}