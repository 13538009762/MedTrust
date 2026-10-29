package security

import (
	"testing"
)

func TestValidateMedicalFile_AllowedTypes(t *testing.T) {
	// PDF magic bytes: %PDF-
	pdfBytes := append([]byte("%PDF-1.4\n"), make([]byte, 100)...)
	if err := ValidateMedicalFile("case_report.pdf", pdfBytes); err != nil {
		t.Fatalf("正常 PDF 文件应校验通过: %v", err)
	}

	// PNG magic bytes: \x89PNG\r\n\x1a\n
	pngBytes := append([]byte("\x89PNG\r\n\x1a\n"), make([]byte, 100)...)
	if err := ValidateMedicalFile("ct_scan.png", pngBytes); err != nil {
		t.Fatalf("正常 PNG 影像应校验通过: %v", err)
	}

	// Plain text
	txtBytes := []byte("临床电子病历纯文本记录")
	if err := ValidateMedicalFile("record.txt", txtBytes); err != nil {
		t.Fatalf("正常 TXT 文件应校验通过: %v", err)
	}
}

func TestValidateMedicalFile_RejectsDangerousExtensions(t *testing.T) {
	fakeData := []byte("echo 'malicious payload'")
	badFiles := []string{"malware.exe", "trojan.sh", "backdoor.php", "script.vbs", "exploit.bat"}

	for _, bf := range badFiles {
		if err := ValidateMedicalFile(bf, fakeData); err == nil {
			t.Fatalf("预期拒绝危险扩展名 %s，但未报错", bf)
		}
	}
}

func TestValidateMedicalFile_RejectsDisguisedExecutable(t *testing.T) {
	// DOS executable magic bytes: MZ...
	dosExeBytes := append([]byte("MZ\x90\x00\x03\x00\x00\x00"), make([]byte, 200)...)
	// 即使伪装成 .pdf 扩展名，头部 MIME 嗅探仍会被捕获
	if err := ValidateMedicalFile("disguised_trojan.pdf", dosExeBytes); err == nil {
		t.Fatalf("伪装成 PDF 的可执行文件应被检测拦截")
	}
}

func TestValidateImageFile(t *testing.T) {
	pngBytes := append([]byte("\x89PNG\r\n\x1a\n"), make([]byte, 100)...)
	if err := ValidateImageFile("avatar.png", pngBytes); err != nil {
		t.Fatalf("合法 PNG 头像应通过: %v", err)
	}

	fakeText := []byte("this is plain text not an image")
	if err := ValidateImageFile("avatar.jpg", fakeText); err == nil {
		t.Fatalf("非图片伪造为 JPG 应被拦截")
	}
}
