package repository

import (
	"errors"
	"fmt"
	"gorm.io/gorm"
	"medtrust-backend/model"
	"time"
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

// FindRecordSimpleByID WP-15: 按主键查单条病历（不带 Preload，供模拟篡改等场景复用）
func (r *MedicalRepository) FindRecordSimpleByID(id uint64) (*model.MedicalRecord, error) {
	if DB == nil {
		return nil, errors.New("database not initialized")
	}
	var rec model.MedicalRecord
	if err := DB.First(&rec, id).Error; err != nil {
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

// UpdateMedicalRecord WP-15: 整体保存医疗记录（Save），供 SimulateTamper / RestoreTamperedRecord 调用
func (r *MedicalRepository) UpdateMedicalRecord(rec *model.MedicalRecord) error {
	if DB == nil {
		return errors.New("database not initialized")
	}
	return DB.Save(rec).Error
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

func (r *MedicalRepository) FindRecordsByIDs(ids []uint64) ([]model.MedicalRecord, error) {
	if DB == nil {
		return nil, errors.New("database not initialized")
	}
	var recs []model.MedicalRecord
	err := DB.Where("id IN ?", ids).Find(&recs).Error
	return recs, err
}

func (r *MedicalRepository) FindExamOrderByID(id uint64) (*model.MedicalExamOrder, error) {
	if DB == nil {
		return nil, errors.New("database not initialized")
	}
	var order model.MedicalExamOrder
	if err := DB.First(&order, id).Error; err != nil {
		return nil, err
	}
	return &order, nil
}

func (r *MedicalRepository) CountTotal() int64 {
	if DB == nil {
		return 0
	}
	var count int64
	DB.Model(&model.MedicalRecord{}).Count(&count)
	return count
}

// CountIntegrityStatus WP-12: 按持久化 IntegrityStatus 统计记录数 (索引化 COUNT，替代总览端点全表 Fabric 核验循环)
func (r *MedicalRepository) CountIntegrityStatus(status string) int64 {
	if DB == nil {
		return 0
	}
	var count int64
	DB.Model(&model.MedicalRecord{}).Where("integrity_status = ?", status).Count(&count)
	return count
}

func (r *MedicalRepository) ListAllRecords() ([]model.MedicalRecord, error) {
	if DB == nil {
		return nil, errors.New("database not initialized")
	}
	var recs []model.MedicalRecord
	err := DB.Find(&recs).Error
	return recs, err
}

func (r *MedicalRepository) GetPatientRecordIdentifiers(patientID uint64) (recordNos []string, recordIDs []uint64) {
	if DB == nil {
		return nil, nil
	}
	DB.Model(&model.MedicalRecord{}).Where("patient_id = ?", patientID).Pluck("record_no", &recordNos)
	DB.Model(&model.MedicalRecord{}).Where("patient_id = ?", patientID).Pluck("id", &recordIDs)
	return recordNos, recordIDs
}

func (r *MedicalRepository) FindRecordsByPatientAndIDs(patientID uint64, recordIDs []uint64) ([]model.MedicalRecord, error) {
	if DB == nil {
		return nil, errors.New("database not initialized")
	}
	var records []model.MedicalRecord
	err := DB.Where("patient_id = ? AND id IN ?", patientID, recordIDs).Find(&records).Error
	return records, err
}

func (r *MedicalRepository) FindRecordsByPatient(patientID uint64) ([]model.MedicalRecord, error) {
	if DB == nil {
		return nil, errors.New("database not initialized")
	}
	var records []model.MedicalRecord
	err := DB.Where("patient_id = ?", patientID).Find(&records).Error
	return records, err
}

func (r *MedicalRepository) FindPatientRecord(recordID, patientID uint64) (*model.MedicalRecord, error) {
	if DB == nil {
		return nil, errors.New("database not initialized")
	}
	var rec model.MedicalRecord
	if err := DB.Where("id = ? AND patient_id = ?", recordID, patientID).First(&rec).Error; err != nil {
		return nil, err
	}
	return &rec, nil
}

// RecordByNo WP-12: 仅按 record_no 查单条（不带 Preload，供 VerifyBaselineLedgerAnchored 用）
func (r *MedicalRepository) RecordByNo(recordNo string) (*model.MedicalRecord, error) {
	if DB == nil {
		return nil, errors.New("database not initialized")
	}
	var rec model.MedicalRecord
	if err := DB.Where("record_no = ?", recordNo).First(&rec).Error; err != nil {
		return nil, err
	}
	return &rec, nil
}

// HasRecordByRecordNo WP-15: 按 record_no 查是否存在（返回 bool + error，无 Preload）
func (r *MedicalRepository) HasRecordByRecordNo(recordNo string) (bool, error) {
	if DB == nil {
		return false, nil
	}
	var count int64
	DB.Model(&model.MedicalRecord{}).Where("record_no = ?", recordNo).Count(&count)
	return count > 0, nil
}

// FindByIdempotencyKeyWithFiles WP-15: 按幂等键查询病历（含 Files），供 UploadRecord 幂等拦截用
func (r *MedicalRepository) FindByIdempotencyKeyWithFiles(key string) (*model.MedicalRecord, error) {
	if DB == nil {
		return nil, errors.New("database not initialized")
	}
	var rec model.MedicalRecord
	if err := DB.Where("idempotency_key = ?", key).Preload("Files").First(&rec).Error; err != nil {
		return nil, err
	}
	return &rec, nil
}

// UpdateRecordAttachments WP-15: 更新记录关联的文件附件（含创建新文件 if missing）
func (r *MedicalRepository) UpdateRecordAttachments(recordID uint64, cid, encDEK, ivNonce, fileHash string) error {
	if DB == nil {
		return errors.New("database not initialized")
	}
	var existingFile model.MedicalFile
	if err := DB.Where("record_id = ?", recordID).First(&existingFile).Error; err != nil {
		// 不存在则新建
		newFile := model.MedicalFile{
			RecordID:     recordID,
			FileName:     fmt.Sprintf("规范诊疗记录_%d.pdf", recordID),
			FileType:     "pdf",
			FileSize:     uint64(len(fileHash)),
			IPFSCID:      cid,
			EncryptedDEK: encDEK,
			IVNonce:      ivNonce,
			FileHash:     fileHash,
			CreatedAt:    time.Now(),
		}
		return DB.Create(&newFile).Error
	}
	return DB.Model(&existingFile).Updates(map[string]interface{}{
		"ipfs_cid":      cid,
		"encrypted_dek": encDEK,
		"iv_nonce":      ivNonce,
		"file_hash":     fileHash,
	}).Error
}

// ExistsByRecordNo WP-15: 按 record_no 查是否存在（返回 bool）
func (r *MedicalRepository) ExistsByRecordNo(recordNo string) bool {
	if DB == nil {
		return false
	}
	var count int64
	DB.Model(&model.MedicalRecord{}).Where("record_no = ?", recordNo).Count(&count)
	return count > 0
}

// ListAllRecordsPreloaded WP-15: 列出所有病历并预加载 Files（供 EnsureBaselineLedgerAnchored 用）
func (r *MedicalRepository) ListAllRecordsPreloaded() ([]model.MedicalRecord, error) {
	if DB == nil {
		return nil, errors.New("database not initialized")
	}
	var records []model.MedicalRecord
	err := DB.Preload("Files").Find(&records).Error
	return records, err
}

// RecordByNoSimple WP-15: 按 record_no 简单查询（无 Preload），供 EnsureBaselineLedgerAnchored 判断记录是否存在
func (r *MedicalRepository) RecordByNoSimple(recordNo string) (*model.MedicalRecord, error) {
	if DB == nil {
		return nil, errors.New("database not initialized")
	}
	var rec model.MedicalRecord
	if err := DB.Where("record_no = ?", recordNo).First(&rec).Error; err != nil {
		return nil, err
	}
	return &rec, nil
}

// UpsertRecordFile WP-15: 原子更新已有文件或者新建，供 EnsureBaselineLedgerAnchored 文件分支调用
func (r *MedicalRepository) UpsertRecordFile(recordID uint64, fileName string, fileType string, fileSize uint64, ipfsCID, encDEK, ivNonce, fileHash string) error {
	if DB == nil {
		return errors.New("database not initialized")
	}
	var existing model.MedicalFile
	if err := DB.Where("record_id = ?", recordID).First(&existing).Error; err != nil {
		newFile := model.MedicalFile{
			RecordID:     recordID,
			FileName:     fileName,
			FileType:     fileType,
			FileSize:     fileSize,
			IPFSCID:      ipfsCID,
			EncryptedDEK: encDEK,
			IVNonce:      ivNonce,
			FileHash:     fileHash,
			CreatedAt:    time.Now(),
		}
		return DB.Create(&newFile).Error
	}
	return DB.Model(&existing).Updates(map[string]interface{}{
		"ipfs_cid":      ipfsCID,
		"encrypted_dek": encDEK,
		"iv_nonce":      ivNonce,
		"file_hash":     fileHash,
	}).Error
}

// UpdateMedicalRecordFields WP-15: 按主键批量更新医疗记录指定字段
func (r *MedicalRepository) UpdateMedicalRecordFields(id uint64, updates map[string]interface{}) error {
	if DB == nil {
		return errors.New("database not initialized")
	}
	return DB.Model(&model.MedicalRecord{}).Where("id = ?", id).Updates(updates).Error
}

// CountMedicalRecordsByNo WP-15: 按 record_no 统计存在性
func (r *MedicalRepository) CountMedicalRecordsByNo(recordNo string) int64 {
	if DB == nil {
		return 0
	}
	var count int64
	DB.Model(&model.MedicalRecord{}).Where("record_no = ?", recordNo).Count(&count)
	return count
}

// FindLatestByPatient 取患者最新一条记录（含 Files）
func (r *MedicalRepository) FindLatestByPatient(patientID uint64) (*model.MedicalRecord, error) {
	if DB == nil {
		return nil, errors.New("database not initialized")
	}
	var rec model.MedicalRecord
	if err := DB.Where("patient_id = ?", patientID).Order("id desc").Preload("Files").First(&rec).Error; err != nil {
		return nil, err
	}
	return &rec, nil
}

// FindVitalsByPatientIDs 批量查询指定患者在有生命体征的最新病历
func (r *MedicalRepository) FindVitalsByPatientIDs(patientIDs []uint64) ([]model.MedicalRecord, error) {
	if DB == nil {
		return nil, errors.New("database not initialized")
	}
	var recs []model.MedicalRecord
	err := DB.Select("id, patient_id, vital_signs").
		Where("patient_id IN ? AND vital_signs != ''", patientIDs).
		Order("id desc").Find(&recs).Error
	return recs, err
}

// CountRecentActiveBreakGlassWP15 WP-15: 统计某医生对某病历在窗口期内有效紧急破窗次数
func (r *MedicalRepository) CountRecentActiveBreakGlass(doctorID, recordID uint64, windowStart time.Time) int64 {
	if DB == nil {
		return 0
	}
	var count int64
	DB.Model(&model.EmergencyAccessEvent{}).Where(
		"doctor_id = ? AND record_id = ? AND created_at >= ? AND audit_status != 'CLOSED_VIOLATION' AND patient_feedback != 'OBJECTED'",
		doctorID, recordID, windowStart,
	).Count(&count)
	return count
}

// RecordQuery 返回 *gorm.DB 查询构建器（用于 GetRecords 复杂条件链式拼接）
type RecordQuery struct {
	q *gorm.DB
}

func (r *RecordQuery) Where(query interface{}, args ...interface{}) *RecordQuery {
	r.q = r.q.Where(query, args...)
	return r
}

func (r *RecordQuery) Order(query string) *RecordQuery {
	r.q = r.q.Order(query)
	return r
}

// Offset 设置分页起始偏移，配合 Limit 实现分页下推
func (r *RecordQuery) Offset(offset int) *RecordQuery {
	r.q = r.q.Offset(offset)
	return r
}

// Limit 限制单页返回条数
func (r *RecordQuery) Limit(limit int) *RecordQuery {
	r.q = r.q.Limit(limit)
	return r
}

func (r *RecordQuery) Find(dst interface{}) error {
	return r.q.Find(dst).Error
}

// NewRecordQuery 创建一个带 Preload 的医疗记录查询构建器
func (r *MedicalRepository) NewRecordQuery() *RecordQuery {
	return &RecordQuery{q: DB.Model(&model.MedicalRecord{}).Preload("Files")}
}
