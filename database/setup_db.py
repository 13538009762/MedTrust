# setup_db.py
import pymysql

conn = pymysql.connect(
    host='127.0.0.1',
    user='root',
    password='123456',
    port=3306,
    autocommit=True
)

cursor = conn.cursor()
cursor.execute('CREATE DATABASE IF NOT EXISTS medtrust DEFAULT CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;')
cursor.execute('USE medtrust;')

tables = [
"""
CREATE TABLE IF NOT EXISTS users (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '用户主键',
  user_no VARCHAR(64) NOT NULL COMMENT '用户业务编号',
  username VARCHAR(50) NOT NULL COMMENT '登录账号名',
  password_hash VARCHAR(255) NOT NULL COMMENT 'bcrypt密码哈希',
  real_name VARCHAR(50) NOT NULL COMMENT '用户真实姓名',
  role VARCHAR(20) NOT NULL COMMENT '角色: doctor, patient, admin, supervisor',
  hospital_id BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '所属机构ID',
  department_id BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '所属科室ID',
  title VARCHAR(50) NOT NULL DEFAULT '' COMMENT '医生职称',
  phone VARCHAR(20) NOT NULL DEFAULT '' COMMENT '联系电话',
  id_card VARCHAR(20) NOT NULL DEFAULT '' COMMENT '居民身份证号',
  status VARCHAR(20) NOT NULL DEFAULT 'NORMAL' COMMENT '状态: NORMAL, RESTRICTED, DISABLED',
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (id),
  UNIQUE KEY uk_username (username),
  UNIQUE KEY uk_user_no (user_no),
  KEY idx_hosp_dept (hospital_id, department_id),
  KEY idx_phone (phone),
  KEY idx_id_card (id_card)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
""",
"""
CREATE TABLE IF NOT EXISTS hospitals (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  hospital_no VARCHAR(32) NOT NULL,
  name VARCHAR(100) NOT NULL,
  level VARCHAR(20) NOT NULL DEFAULT '三甲',
  address VARCHAR(255) NOT NULL DEFAULT '',
  status TINYINT NOT NULL DEFAULT 1,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (id),
  UNIQUE KEY uk_hospital_no (hospital_no)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
""",
"""
CREATE TABLE IF NOT EXISTS departments (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  dept_no VARCHAR(32) NOT NULL,
  hospital_id BIGINT UNSIGNED NOT NULL,
  name VARCHAR(64) NOT NULL,
  description VARCHAR(255) NOT NULL DEFAULT '',
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (id),
  KEY idx_hospital_id (hospital_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
""",
"""
CREATE TABLE IF NOT EXISTS medical_records (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  record_no VARCHAR(64) NOT NULL,
  patient_id BIGINT UNSIGNED NOT NULL,
  doctor_id BIGINT UNSIGNED NOT NULL,
  hospital_id BIGINT UNSIGNED NOT NULL,
  data_type VARCHAR(32) NOT NULL,
  onset_time VARCHAR(50) NOT NULL DEFAULT '' COMMENT '发病时间',
  duration VARCHAR(50) NOT NULL DEFAULT '' COMMENT '持续时间',
  symptoms TEXT NULL COMMENT '主要临床症状',
  etiology TEXT NULL COMMENT '病情诱因与病因分析',
  treatment_plan TEXT NULL COMMENT '解决建议与处方方案',
  vital_signs VARCHAR(255) NOT NULL DEFAULT '' COMMENT '生命体征指标',
  diagnosis TEXT NULL COMMENT '完整病历小结/诊断结论',
  fabric_tx_id VARCHAR(128) NOT NULL DEFAULT '',
  block_height BIGINT UNSIGNED NOT NULL DEFAULT 0,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (id),
  UNIQUE KEY uk_record_no (record_no),
  KEY idx_patient_id (patient_id),
  KEY idx_doctor_hosp (doctor_id, hospital_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
""",
"""
CREATE TABLE IF NOT EXISTS medical_files (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  record_id BIGINT UNSIGNED NOT NULL,
  file_name VARCHAR(255) NOT NULL,
  file_type VARCHAR(32) NOT NULL,
  file_size BIGINT UNSIGNED NOT NULL,
  ipfs_cid VARCHAR(128) NOT NULL,
  file_hash VARCHAR(64) NOT NULL,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (id),
  KEY idx_record_id (record_id),
  KEY idx_ipfs_cid (ipfs_cid)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
""",
"""
CREATE TABLE IF NOT EXISTS authorizations (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  auth_no VARCHAR(64) NOT NULL,
  patient_id BIGINT UNSIGNED NOT NULL,
  auth_target_type VARCHAR(20) NOT NULL,
  auth_target_id BIGINT UNSIGNED NOT NULL,
  scope_type VARCHAR(20) NOT NULL,
  record_id BIGINT UNSIGNED NOT NULL DEFAULT 0,
  start_time DATETIME NOT NULL,
  end_time DATETIME NOT NULL,
  status VARCHAR(20) NOT NULL DEFAULT 'ACTIVE',
  fabric_tx_id VARCHAR(128) NOT NULL DEFAULT '',
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (id),
  UNIQUE KEY uk_auth_no (auth_no),
  KEY idx_pat_status (patient_id, status)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
""",
"""
CREATE TABLE IF NOT EXISTS access_requests (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  request_no VARCHAR(64) NOT NULL,
  doctor_id BIGINT UNSIGNED NOT NULL,
  patient_id BIGINT UNSIGNED NOT NULL,
  record_id BIGINT UNSIGNED NOT NULL,
  source_hospital_id BIGINT UNSIGNED NOT NULL,
  target_hospital_id BIGINT UNSIGNED NOT NULL,
  purpose VARCHAR(255) NOT NULL,
  risk_score INT NOT NULL,
  risk_level VARCHAR(20) NOT NULL,
  decision VARCHAR(20) NOT NULL,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (id),
  UNIQUE KEY uk_request_no (request_no)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
""",
"""
CREATE TABLE IF NOT EXISTS emergency_access_events (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  event_no VARCHAR(64) NOT NULL,
  doctor_id BIGINT UNSIGNED NOT NULL,
  patient_id BIGINT UNSIGNED NOT NULL,
  record_id BIGINT UNSIGNED NOT NULL,
  source_hospital_id BIGINT UNSIGNED NOT NULL,
  target_hospital_id BIGINT UNSIGNED NOT NULL,
  emergency_reason VARCHAR(50) NOT NULL,
  description TEXT NOT NULL,
  doctor_confirmed TINYINT NOT NULL DEFAULT 1,
  patient_feedback VARCHAR(20) NOT NULL DEFAULT 'PENDING',
  patient_comment TEXT NULL,
  audit_status VARCHAR(30) NOT NULL DEFAULT 'PENDING_AUDIT',
  supervisor_id BIGINT UNSIGNED NOT NULL DEFAULT 0,
  audit_comment TEXT NULL,
  punishment VARCHAR(20) NOT NULL DEFAULT 'NONE',
  fabric_tx_id VARCHAR(128) NOT NULL DEFAULT '',
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (id),
  UNIQUE KEY uk_event_no (event_no),
  KEY idx_doc_pat (doctor_id, patient_id),
  KEY idx_audit_status (audit_status)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
""",
"""
CREATE TABLE IF NOT EXISTS audit_logs (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  log_id VARCHAR(64) NOT NULL,
  user_id BIGINT UNSIGNED NOT NULL,
  operation_type VARCHAR(50) NOT NULL,
  target_type VARCHAR(50) NOT NULL,
  target_id VARCHAR(64) NOT NULL,
  hospital_id BIGINT UNSIGNED NOT NULL,
  result VARCHAR(20) NOT NULL,
  risk_level VARCHAR(20) NOT NULL DEFAULT 'LOW',
  ip_address VARCHAR(50) NOT NULL DEFAULT '',
  fabric_tx_id VARCHAR(128) NOT NULL DEFAULT '',
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (id),
  UNIQUE KEY uk_log_id (log_id),
  KEY idx_user_op (user_id, operation_type),
  KEY idx_hospital_time (hospital_id, created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
"""
]

for t in tables:
    cursor.execute(t)

def add_column_if_not_exists(cursor, table, col, defn):
    cursor.execute("""
        SELECT COUNT(*) FROM information_schema.columns 
        WHERE table_schema='medtrust' AND table_name=%s AND column_name=%s
    """, (table, col))
    if cursor.fetchone()[0] == 0:
        cursor.execute(f"ALTER TABLE {table} ADD COLUMN {col} {defn}")
        print(f"Added column {col} to {table}")

add_column_if_not_exists(cursor, "users", "id_card", "VARCHAR(20) NOT NULL DEFAULT '' COMMENT '居民身份证号' AFTER phone")
add_column_if_not_exists(cursor, "medical_records", "onset_time", "VARCHAR(50) NOT NULL DEFAULT '' COMMENT '发病时间' AFTER data_type")
add_column_if_not_exists(cursor, "medical_records", "duration", "VARCHAR(50) NOT NULL DEFAULT '' COMMENT '持续时间' AFTER onset_time")
add_column_if_not_exists(cursor, "medical_records", "symptoms", "TEXT NULL COMMENT '主要临床症状' AFTER duration")
add_column_if_not_exists(cursor, "medical_records", "etiology", "TEXT NULL COMMENT '病情诱因与病因分析' AFTER symptoms")
add_column_if_not_exists(cursor, "medical_records", "treatment_plan", "TEXT NULL COMMENT '解决建议与处方方案' AFTER etiology")
add_column_if_not_exists(cursor, "medical_records", "vital_signs", "VARCHAR(255) NOT NULL DEFAULT '' COMMENT '生命体征指标' AFTER treatment_plan")

print('9 Tables created and schema updated successfully!')
conn.close()
