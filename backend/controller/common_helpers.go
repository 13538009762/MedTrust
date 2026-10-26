package controller

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"medtrust-backend/model"
	"medtrust-backend/repository"
)

// maskPhone 手机号前三后四脱敏 (中四位掩码)
func maskPhone(phone string) string {
	runes := []rune(phone)
	if len(runes) <= 7 {
		return phone
	}
	return string(runes[:3]) + "****" + string(runes[len(runes)-4:])
}

// maskIDCard 居民身份证号前六后四脱敏 (中八位掩码)
func maskIDCard(idCard string) string {
	runes := []rune(idCard)
	if len(runes) <= 10 {
		return idCard
	}
	return string(runes[:6]) + "********" + string(runes[len(runes)-4:])
}

// parsePagination 解析标准分页入参 (page, page_size) 并计算 DB 偏移量 offset
func parsePagination(c *gin.Context, defaultPageSize int) (page int, pageSize int, offset int) {
	if defaultPageSize <= 0 {
		defaultPageSize = 20
	}
	page, _ = strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ = strconv.Atoi(c.DefaultQuery("page_size", strconv.Itoa(defaultPageSize)))

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = defaultPageSize
	}
	offset = (page - 1) * pageSize
	return page, pageSize, offset
}

// loadUsersBatch 批量拉取用户实体并建立 ID 映射表 (消除 N+1 查询)
func loadUsersBatch(ids []uint64) map[uint64]model.User {
	userMap := make(map[uint64]model.User)
	uniqueIDs := deduplicateIDs(ids)
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

// loadHospitalsBatch 批量拉取医疗机构并建立 ID 映射表
func loadHospitalsBatch(ids []uint64) map[uint64]model.Hospital {
	hospMap := make(map[uint64]model.Hospital)
	uniqueIDs := deduplicateIDs(ids)
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

// loadRecordsBatch 批量拉取病历并建立 ID 映射表
func loadRecordsBatch(ids []uint64) map[uint64]model.MedicalRecord {
	recMap := make(map[uint64]model.MedicalRecord)
	uniqueIDs := deduplicateIDs(ids)
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

// deduplicateIDs 辅助清洗并去重 ID 列表
func deduplicateIDs(ids []uint64) []uint64 {
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

// respondSuccess 统一成功结果快捷封装
func respondSuccess(c *gin.Context, message string, data interface{}) {
	c.JSON(http.StatusOK, model.Response{
		Code:    200,
		Message: message,
		Data:    data,
	})
}

// respondError 统一异常响应快捷封装
func respondError(c *gin.Context, httpStatus int, code int, message string) {
	c.JSON(httpStatus, model.Response{
		Code:    code,
		Message: message,
	})
}
