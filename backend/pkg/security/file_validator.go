package security

import (
	"errors"
	"net/http"
	"path/filepath"
	"strings"
)

var (
	// AllowedMedicalExtensions 允许的医疗影像与病历附件扩展名白名单
	AllowedMedicalExtensions = map[string]bool{
		".pdf":   true,
		".png":   true,
		".jpg":   true,
		".jpeg":  true,
		".txt":   true,
		".json":  true,
		".dcm":   true,
		".dicom": true,
	}

	// DangerousExtensions 明确禁止的危险可执行扩展名黑名单
	DangerousExtensions = map[string]bool{
		".exe": true, ".dll": true, ".so": true, ".bat": true,
		".cmd": true, ".sh": true, ".ps1": true, ".vbs": true,
		".php": true, ".jsp": true, ".asp": true, ".aspx": true,
		".jar": true, ".war": true, ".py": true, ".pl": true,
		".html": true, ".htm": true, ".svg": true, ".js": true,
	}
)

// ValidateMedicalFile 综合验证文件名、扩展名与头部 Magic Byte MIME 类型
func ValidateMedicalFile(fileName string, data []byte) error {
	if len(data) == 0 {
		return nil
	}

	ext := strings.ToLower(filepath.Ext(fileName))
	if DangerousExtensions[ext] {
		return errors.New("禁止上传可执行或脚本类危险文件")
	}

	// 扩展名白名单校验 (若文件名有扩展名)
	if ext != "" && !AllowedMedicalExtensions[ext] {
		return errors.New("不支持的文件扩展名，仅允许 PDF, PNG, JPG, JPEG, TXT, JSON, DICOM")
	}

	// 头部探测 Magic Bytes 与 MIME 类型
	if len(data) >= 2 && data[0] == 'M' && data[1] == 'Z' {
		return errors.New("检测到可执行文件 (MZ header) 签名，已被系统安全机制拦截")
	}
	if len(data) >= 4 && string(data[:4]) == "\x7fELF" {
		return errors.New("检测到 Linux ELF 可执行文件签名，已被系统安全机制拦截")
	}
	if len(data) >= 2 && string(data[:2]) == "#!" {
		return errors.New("检测到可执行脚本 (Shebang) 签名，已被系统安全机制拦截")
	}

	sniffLen := 512
	if len(data) < sniffLen {
		sniffLen = len(data)
	}
	mime := http.DetectContentType(data[:sniffLen])

	// 严禁可执行文件、DOS可执行、Shell脚本、HTML等危险类型
	dangerousMIMEs := []string{
		"application/x-dosexec",
		"application/x-executable",
		"application/x-sharedlib",
		"text/html",
		"text/javascript",
		"application/javascript",
		"application/x-php",
	}
	for _, dm := range dangerousMIMEs {
		if strings.HasPrefix(mime, dm) {
			return errors.New("检测到文件内容属于危险或可执行二进制格式，已被系统安全机制拦截")
		}
	}

	return nil
}

// ValidateImageFile 头像图片专属校验
func ValidateImageFile(fileName string, data []byte) error {
	if len(data) == 0 {
		return errors.New("头像文件不能为空")
	}

	ext := strings.ToLower(filepath.Ext(fileName))
	allowedExts := map[string]bool{
		".jpg": true, ".jpeg": true, ".png": true, ".gif": true, ".webp": true,
	}
	if !allowedExts[ext] {
		return errors.New("头像仅支持 JPG, PNG, GIF, WEBP 图片格式")
	}

	sniffLen := 512
	if len(data) < sniffLen {
		sniffLen = len(data)
	}
	mime := http.DetectContentType(data[:sniffLen])
	if !strings.HasPrefix(mime, "image/") {
		return errors.New("文件实际内容并非有效图片数据")
	}

	return nil
}
