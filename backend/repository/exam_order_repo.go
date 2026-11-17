package repository

import (
	"errors"

	"medtrust-backend/model"
)

// ExamOrderRepository WP-15: 医技检查单数据访问收口
// 将 service 层（encounter_service / ai_diagnostic_service）中直接使用的
// repository.DB.* GORM 查询构建器封装为 repository 方法，领域逻辑不再依赖 GORM。
type ExamOrderRepository struct{}

var DefaultExamOrderRepository = &ExamOrderRepository{}

func (r *ExamOrderRepository) db() error {
	if DB == nil {
		return errors.New("database not initialized")
	}
	return nil
}

// CreateExamOrder 创建单个检查单
func (r *ExamOrderRepository) CreateExamOrder(order *model.MedicalExamOrder) error {
	if err := r.db(); err != nil {
		return err
	}
	return DB.Create(order).Error
}

// ListExamOrdersByFilter 检查中心列表查询：按状态/医院/患者过滤，ID 倒序
func (r *ExamOrderRepository) ListExamOrdersByFilter(status string, hospitalID, patientID uint64) ([]model.MedicalExamOrder, error) {
	if err := r.db(); err != nil {
		return nil, err
	}
	query := DB.Model(&model.MedicalExamOrder{})
	if status != "" && status != "ALL" {
		query = query.Where("status = ?", status)
	}
	if hospitalID > 0 {
		query = query.Where("hospital_id = ?", hospitalID)
	}
	if patientID > 0 {
		query = query.Where("patient_id = ?", patientID)
	}
	var list []model.MedicalExamOrder
	err := query.Order("id desc").Find(&list).Error
	return list, err
}

// SaveExamOrder 整体保存检查单更新
func (r *ExamOrderRepository) SaveExamOrder(order *model.MedicalExamOrder) error {
	if err := r.db(); err != nil {
		return err
	}
	return DB.Save(order).Error
}

// FindExamOrdersByRecordID 查询就诊记录下全部检查单
func (r *ExamOrderRepository) FindExamOrdersByRecordID(recordID uint64) ([]model.MedicalExamOrder, error) {
	if err := r.db(); err != nil {
		return nil, err
	}
	var orders []model.MedicalExamOrder
	err := DB.Where("record_id = ?", recordID).Find(&orders).Error
	return orders, err
}

// FindExamOrdersByPatientID 按患者查询全部检查单（ID 正序，用于病历时序轴）
func (r *ExamOrderRepository) FindExamOrdersByPatientID(patientID uint64) ([]model.MedicalExamOrder, error) {
	if err := r.db(); err != nil {
		return nil, err
	}
	var orders []model.MedicalExamOrder
	err := DB.Where("patient_id = ?", patientID).Order("id asc").Find(&orders).Error
	return orders, err
}

// FindFileIDsByCIDs 按 IPFS CID 批量查询附件 ID 映射
func (r *ExamOrderRepository) FindFileIDsByCIDs(cids []string) (map[string]uint64, error) {
	result := make(map[string]uint64)
	if err := r.db(); err != nil {
		return result, err
	}
	if len(cids) == 0 {
		return result, nil
	}
	var files []model.MedicalFile
	if err := DB.Where("ipfs_cid IN ?", cids).Find(&files).Error; err != nil {
		return result, err
	}
	for _, mf := range files {
		result[mf.IPFSCID] = mf.ID
	}
	return result, nil
}

// UpdateRecordStatusIf 条件更新就诊记录状态（仅当当前状态匹配时生效）
func (r *ExamOrderRepository) UpdateRecordStatusIf(recordID uint64, fromStatus, toStatus string) error {
	if err := r.db(); err != nil {
		return err
	}
	return DB.Model(&model.MedicalRecord{}).
		Where("id = ? AND status = ?", recordID, fromStatus).
		Update("status", toStatus).Error
}

// --- 医技人员展示名清洗所需的关联查询 ---

// FindUserHospitalName 返回技师所属医院名称（任一环节缺失返回空串）
func (r *ExamOrderRepository) FindUserHospitalName(userID uint64) string {
	if err := r.db(); err != nil {
		return ""
	}
	var u model.User
	if err := DB.First(&u, userID).Error; err != nil || u.HospitalID == 0 {
		return ""
	}
	var h model.Hospital
	if err := DB.First(&h, u.HospitalID).Error; err != nil {
		return ""
	}
	return h.Name
}

// SaveRecord 整体保存就诊记录更新
func (r *ExamOrderRepository) SaveRecord(rec *model.MedicalRecord) error {
	if err := r.db(); err != nil {
		return err
	}
	return DB.Save(rec).Error
}