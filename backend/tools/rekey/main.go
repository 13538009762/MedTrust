package main

import (
	"encoding/base64"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"medtrust-backend/config"
	"medtrust-backend/model"
	"medtrust-backend/pkg/crypto"
)

func parseKey(raw string) ([]byte, error) {
	k := strings.TrimSpace(raw)
	if strings.HasPrefix(k, "base64:") {
		decoded, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(k, "base64:"))
		if err != nil {
			return nil, fmt.Errorf("base64 解码失败: %w", err)
		}
		if len(decoded) != 32 {
			return nil, fmt.Errorf("base64 解码后密钥长度为 %d 字节，必须为 32 字节", len(decoded))
		}
		return decoded, nil
	}
	if len(k) == 32 {
		return []byte(k), nil
	}
	return nil, fmt.Errorf("密钥长度为 %d 字符，必须为 32 字符明文或 'base64:...' 格式", len(k))
}

func main() {
	configPath := flag.String("config", "./config/config.yaml", "配置文件路径")
	oldKeyFlag := flag.String("old-key", "", "旧 AES 主密钥 (32 字符或 base64: 格式，若为空则从配置读取)")
	newKeyFlag := flag.String("new-key", "", "新 AES 主密钥 (必须为 32 字符或 base64: 格式)")
	dryRun := flag.Bool("dry-run", false, "演练模式（仅验证解封与重封装逻辑，不写入数据库）")
	flag.Parse()

	if *newKeyFlag == "" {
		fmt.Println("使用方法示例:")
		fmt.Println("  go run ./tools/rekey/main.go --old-key \"MedTrustMasterKeyAES256GCM2026!\" --new-key \"NewMasterKeyAES256GCM32Bytes2026\" --dry-run")
		fmt.Println("  go run ./tools/rekey/main.go --new-key \"NewMasterKeyAES256GCM32Bytes2026\"")
		log.Fatalf("错误: 必须指定 --new-key")
	}

	cfg, err := config.LoadConfig(*configPath)
	if err != nil {
		log.Fatalf("加载配置失败: %v", err)
	}

	var oldKeyBytes []byte
	if *oldKeyFlag != "" {
		oldKeyBytes, err = parseKey(*oldKeyFlag)
		if err != nil {
			log.Fatalf("解析 --old-key 失败: %v", err)
		}
	} else {
		oldKeyBytes = cfg.GetAESMasterKey()
	}

	newKeyBytes, err := parseKey(*newKeyFlag)
	if err != nil {
		log.Fatalf("解析 --new-key 失败: %v", err)
	}

	if string(oldKeyBytes) == string(newKeyBytes) {
		log.Fatalf("错误: --old-key 与 --new-key 相同，无需轮转")
	}

	// 连接数据库
	db, err := gorm.Open(mysql.Open(cfg.Database.DSN), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Warn),
	})
	if err != nil {
		log.Fatalf("连接数据库失败: %v", err)
	}

	var files []model.MedicalFile
	if err := db.Where("encrypted_dek != ''").Find(&files).Error; err != nil {
		log.Fatalf("查询 medical_files 失败: %v", err)
	}

	fmt.Printf("====================================================\n")
	fmt.Printf("MedTrust 信封加密主密钥轮转工具 (Rekey Tool)\n")
	fmt.Printf("模式: %s\n", map[bool]string{true: "【DRY-RUN 演练模式 - 不写库】", false: "【LIVE 生产更新模式】"}[*dryRun])
	fmt.Printf("共检索到 %d 条待重包装的病历文件记录\n", len(files))
	fmt.Printf("====================================================\n\n")

	successCount := 0
	failCount := 0

	for _, f := range files {
		newDEK, err := crypto.RewrapDEK(f.EncryptedDEK, oldKeyBytes, newKeyBytes)
		if err != nil {
			failCount++
			fmt.Printf("[FAIL] FileID=%d RecordID=%d Name=%s: %v\n", f.ID, f.RecordID, f.FileName, err)
			continue
		}

		if !*dryRun {
			if err := db.Model(&model.MedicalFile{}).Where("id = ?", f.ID).Update("encrypted_dek", newDEK).Error; err != nil {
				failCount++
				fmt.Printf("[FAIL] FileID=%d 更新数据库失败: %v\n", f.ID, err)
				continue
			}
		}

		successCount++
		fmt.Printf("[OK] FileID=%d RecordID=%d Name=%s -> DEK 重包装成功\n", f.ID, f.RecordID, f.FileName)
	}

	fmt.Printf("\n====================================================\n")
	fmt.Printf("处理完成! 成功: %d, 失败: %d, 模式: %v\n", successCount, failCount, *dryRun)
	if *dryRun {
		fmt.Println("提示: 当前为 dry-run 演练模式，数据库未发生任何变更。确认无误后去掉 --dry-run 执行实际写入。")
	} else if failCount == 0 {
		fmt.Println("恭喜: 所有 DEK 均已用新主密钥重封装完毕！请将新主密钥配置到 .env 的 AES_MASTER_KEY 中并重启后端服务。")
	}
	fmt.Printf("====================================================\n")

	if failCount > 0 {
		os.Exit(1)
	}
}
