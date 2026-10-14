package repository

import (
	"errors"
	"medtrust-backend/model"
)

type MedicalRepository struct{}

var DefaultMedicalRepository = &MedicalRepository{}

func (r *MedicalRepository) CreateRecord(rec *model.MedicalRecord) error {
	if DB == nil {
		return errors.New("database not initialized")
	}
	return DB.Create(rec).Error
}

func (r *MedicalRepository) FindRecordByID(id uint64) (*model.MedicalRecord, error) {
	if DB == nil {
		return nil, errors.New("database not initialized")
	}
	var rec model.MedicalRecord
	if err := DB.Preload("Files").First(&rec, id).Error; err != nil {
		return nil, err
	}
	return &rec, nil
}

func (r *MedicalRepository) FindRecordByNo(recordNo string) (*model.MedicalRecord, error) {
	if DB == nil {
		return nil, errors.New("database not initialized")
	}
	var rec model.MedicalRecord
	if err := DB.Preload("Files").Where("record_no = ?", recordNo).First(&rec).Error; err != nil {
		return nil, err
	}
	return &rec, nil
}

func (r *MedicalRepository) ListRecords(patientID, doctorID, hospitalID uint64, dataType, keyword string) ([]model.MedicalRecord, error) {
	if DB == nil {
		return nil, errors.New("database not initialized")
	}
	query := DB.Model(&model.MedicalRecord{}).Preload("Files")
	if patientID > 0 {
		query = query.Where("patient_id = ?", patientID)
	}
	if doctorID > 0 {
		query = query.Where("doctor_id = ?", doctorID)
	}
	if hospitalID > 0 {
		query = query.Where("hospital_id = ?", hospitalID)
	}
	if dataType != "" {
		query = query.Where("data_type = ?", dataType)
	}
	if keyword != "" {
		query = query.Where("record_no LIKE ? OR title LIKE ? OR diagnosis LIKE ? OR chief_complaint LIKE ?",
			"%"+keyword+"%", "%"+keyword+"%", "%"+keyword+"%", "%"+keyword+"%")
	}
	var records []model.MedicalRecord
	err := query.Order("created_at DESC").Find(&records).Error
	return records, err
}

func (r *MedicalRepository) UpdateIntegrityStatus(id uint64, status string) error {
	if DB == nil {
		return errors.New("database not initialized")
	}
	return DB.Model(&model.MedicalRecord{}).Where("id = ?", id).Update("integrity_status", status).Error
}

func (r *MedicalRepository) CreateFile(file *model.MedicalFile) error {
	if DB == nil {
		return errors.New("database not initialized")
	}
	return DB.Create(file).Error
}

func (r *MedicalRepository) FindFileByID(id uint64) (*model.MedicalFile, error) {
	if DB == nil {
		return nil, errors.New("database not initialized")
	}
	var f model.MedicalFile
	if err := DB.First(&f, id).Error; err != nil {
		return nil, err
	}
	return &f, nil
}

func (r *MedicalRepository) FindFilesByRecordID(recordID uint64) ([]model.MedicalFile, error) {
	if DB == nil {
		return nil, errors.New("database not initialized")
	}
	var files []model.MedicalFile
	err := DB.Where("record_id = ?", recordID).Find(&files).Error
	return files, err
}

func (r *MedicalRepository) UpdateFile(file *model.MedicalFile) error {
	if DB == nil {
		return errors.New("database not initialized")
	}
	return DB.Save(file).Error
}
