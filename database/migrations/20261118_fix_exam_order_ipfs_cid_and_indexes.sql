-- Migration: Fix MedicalExamOrder ip_fsc_id typo to ipfs_cid & add integrity_status index
-- Safe idempotent migration matching GORM model definitions (backend/model/medical_record.go)

ALTER TABLE `medical_exam_orders`
  CHANGE COLUMN `ip_fsc_id` `ipfs_cid` VARCHAR(128) COLLATE utf8mb4_unicode_ci DEFAULT '';

CREATE INDEX `idx_medical_records_integrity_status`
  ON `medical_records` (`integrity_status`);
