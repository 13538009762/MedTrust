package repository

import (
	"errors"

	"medtrust-backend/model"
)

type HospitalRepository struct{}

var DefaultHospitalRepository = &HospitalRepository{}

func (r *HospitalRepository) FindByID(id uint64) (*model.Hospital, error) {
	if DB == nil {
		return nil, errors.New("database not initialized")
	}
	var h model.Hospital
	if err := DB.First(&h, id).Error; err != nil {
		return nil, err
	}
	return &h, nil
}

func (r *HospitalRepository) FindByHospitalNo(hospitalNo string) (*model.Hospital, error) {
	if DB == nil {
		return nil, errors.New("database not initialized")
	}
	var h model.Hospital
	if err := DB.Where("hospital_no = ?", hospitalNo).First(&h).Error; err != nil {
		return nil, err
	}
	return &h, nil
}

// FindByIDs WP-13: 批量拉取机构并建立 ID 映射 (WHERE id IN，消除列表端点 N+1)
func (r *HospitalRepository) FindByIDs(ids []uint64) (map[uint64]model.Hospital, error) {
	result := make(map[uint64]model.Hospital)
	if DB == nil || len(ids) == 0 {
		return result, nil
	}
	var list []model.Hospital
	if err := DB.Where("id IN ?", ids).Find(&list).Error; err != nil {
		return nil, err
	}
	for _, h := range list {
		result[h.ID] = h
	}
	return result, nil
}

// FindDepartmentsByIDs WP-13: 批量拉取科室并建立 ID 映射 (WHERE id IN，消除列表端点 N+1)
func (r *HospitalRepository) FindDepartmentsByIDs(ids []uint64) (map[uint64]model.Department, error) {
	result := make(map[uint64]model.Department)
	if DB == nil || len(ids) == 0 {
		return result, nil
	}
	var list []model.Department
	if err := DB.Where("id IN ?", ids).Find(&list).Error; err != nil {
		return nil, err
	}
	for _, d := range list {
		result[d.ID] = d
	}
	return result, nil
}

func (r *HospitalRepository) ListHospitals() ([]model.Hospital, error) {
	if DB == nil {
		return nil, errors.New("database not initialized")
	}
	var list []model.Hospital
	err := DB.Find(&list).Error
	return list, err
}

func (r *HospitalRepository) CreateHospital(h *model.Hospital) error {
	if DB == nil {
		return errors.New("database not initialized")
	}
	return DB.Create(h).Error
}

func (r *HospitalRepository) UpdateHospital(id uint64, updates map[string]interface{}) (*model.Hospital, error) {
	if DB == nil {
		return nil, errors.New("database not initialized")
	}
	var h model.Hospital
	if err := DB.First(&h, id).Error; err != nil {
		return nil, err
	}
	if err := DB.Model(&h).Updates(updates).Error; err != nil {
		return nil, err
	}
	if err := DB.First(&h, id).Error; err != nil {
		return nil, err
	}
	return &h, nil
}

func (r *HospitalRepository) UpdateHospitalStatus(id uint64, status int8) error {
	if DB == nil {
		return errors.New("database not initialized")
	}
	return DB.Model(&model.Hospital{}).Where("id = ?", id).Update("status", status).Error
}

func (r *HospitalRepository) FindDepartmentByID(id uint64) (*model.Department, error) {
	if DB == nil {
		return nil, errors.New("database not initialized")
	}
	var d model.Department
	if err := DB.First(&d, id).Error; err != nil {
		return nil, err
	}
	return &d, nil
}

func (r *HospitalRepository) FindDepartmentByHospAndNo(hospitalID uint64, deptNo string) (*model.Department, error) {
	if DB == nil {
		return nil, errors.New("database not initialized")
	}
	var d model.Department
	if err := DB.Where("hospital_id = ? AND dept_no = ?", hospitalID, deptNo).First(&d).Error; err != nil {
		return nil, err
	}
	return &d, nil
}

func (r *HospitalRepository) ListDepartments(hospitalID uint64) ([]model.Department, error) {
	if DB == nil {
		return nil, errors.New("database not initialized")
	}
	var list []model.Department
	query := DB.Model(&model.Department{})
	if hospitalID > 0 {
		query = query.Where("hospital_id = ?", hospitalID)
	}
	err := query.Find(&list).Error
	return list, err
}

func (r *HospitalRepository) CreateDepartment(dept *model.Department) error {
	if DB == nil {
		return errors.New("database not initialized")
	}
	return DB.Create(dept).Error
}

func (r *HospitalRepository) UpdateDepartment(id uint64, updates map[string]interface{}) (*model.Department, error) {
	if DB == nil {
		return nil, errors.New("database not initialized")
	}
	var dept model.Department
	if err := DB.First(&dept, id).Error; err != nil {
		return nil, err
	}
	if err := DB.Model(&dept).Updates(updates).Error; err != nil {
		return nil, err
	}
	if err := DB.First(&dept, id).Error; err != nil {
		return nil, err
	}
	return &dept, nil
}

func (r *HospitalRepository) CountUsersByDepartment(deptID uint64) (int64, error) {
	if DB == nil {
		return 0, errors.New("database not initialized")
	}
	var count int64
	err := DB.Model(&model.User{}).Where("department_id = ?", deptID).Count(&count).Error
	return count, err
}

func (r *HospitalRepository) CountRecordsByDepartment(deptID uint64) (int64, error) {
	if DB == nil {
		return 0, errors.New("database not initialized")
	}
	var count int64
	err := DB.Model(&model.MedicalRecord{}).Where("department_id = ?", deptID).Count(&count).Error
	return count, err
}

func (r *HospitalRepository) DeleteDepartment(id uint64) error {
	if DB == nil {
		return errors.New("database not initialized")
	}
	return DB.Delete(&model.Department{}, id).Error
}

// GetActiveFileCIDs WP-15: 查询所有非空 ipfs_cid（供清理孤儿文件用）
func (r *HospitalRepository) GetActiveFileCIDs() ([]string, error) {
	if DB == nil {
		return nil, errors.New("database not initialized")
	}
	var cids []string
	err := DB.Model(&model.MedicalFile{}).Where("ipfs_cid != ''").Pluck("ipfs_cid", &cids).Error
	return cids, err
}
