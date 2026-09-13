package pdf

import (
	"bytes"
	"fmt"
	"strings"
	"time"
	"unicode/utf16"
)

// AttestationData 医疗病历规范凭证数据模型
type AttestationData struct {
	RecordNo         string
	HospitalName     string
	EncounterType    string
	DepartmentName   string
	PatientName      string
	PatientIDCard    string
	PatientPhone     string
	DoctorName       string
	CreatedAt        time.Time
	ChiefComplaint   string
	PresentIllness   string
	OnsetTime        string
	Duration         string
	VitalSigns       string
	ExamResult       string
	InitialDiagnosis string
	DiagnosticBasis  string
	Diagnosis        string
	Etiology         string
	TreatmentPlan    string
	FabricTxID       string
	BlockHeight      uint64
	IPFSCID          string
	FileHash         string
	ClinicalHash     string
}

// encodeUTF16BEHex 将字符串转为 UniGB-UTF16-H 规范的 UTF-16BE 16 进制字符串（带 BOM <FEFF...>）
func encodeUTF16BEHex(s string) string {
	runes := []rune(s)
	u16 := utf16.Encode(runes)
	var b strings.Builder
	b.WriteString("<FEFF")
	for _, code := range u16 {
		b.WriteString(fmt.Sprintf("%04X", code))
	}
	b.WriteString(">")
	return b.String()
}

// GenerateAttestationPDF 生成符合 PDF 1.4 标准的合法二进制电子病历存证文档
func GenerateAttestationPDF(data AttestationData) ([]byte, error) {
	if data.HospitalName == "" {
		data.HospitalName = "MedTrust 医疗可信协同中心"
	}
	if data.EncounterType == "" {
		data.EncounterType = "门诊就诊"
	}
	if data.DepartmentName == "" {
		data.DepartmentName = "综合门诊"
	}
	if data.DoctorName == "" {
		data.DoctorName = "经治责任医师"
	}
	if data.PatientName == "" {
		data.PatientName = "匿名患者"
	}
	if data.CreatedAt.IsZero() {
		data.CreatedAt = time.Now()
	}

	// 构造 PDF 内容流 (Content Stream)
	var stream bytes.Buffer

	// 页面尺寸 A4: 595.28 x 841.89 pt
	// 顶部边距 y = 800, 左边距 x = 40, 可用宽度 515 pt

	// 1. 顶部 Header 装饰条与背景
	stream.WriteString("q\n")
	// 深蓝标题栏矩形 (x=40, y=780, w=515, h=42)
	stream.WriteString("0.09 0.28 0.58 rg\n")
	stream.WriteString("40 780 515 42 re f\n")
	// 浅蓝边框线
	stream.WriteString("0.2 0.45 0.75 RG 1.5 w\n")
	stream.WriteString("40 780 515 42 re S\n")
	stream.WriteString("Q\n")

	// 标题文字 (白色)
	stream.WriteString("BT\n")
	stream.WriteString("1 1 1 rg\n")
	stream.WriteString("/F2 15 Tf\n")
	stream.WriteString("55 795 Td\n")
	stream.WriteString(encodeUTF16BEHex(fmt.Sprintf("【%s】临床规范电子病历存证凭据", data.HospitalName)))
	stream.WriteString(" Tj\nET\n")

	// 页面主体外框
	stream.WriteString("q\n")
	stream.WriteString("0.85 0.88 0.92 RG 1 w\n")
	stream.WriteString("40 50 515 720 re S\n")
	stream.WriteString("Q\n")

	curY := 760

	// 辅助绘制水平分割线
	drawHLine := func(y int) {
		stream.WriteString(fmt.Sprintf("q 0.88 0.90 0.94 RG 0.75 w 45 %d m 550 %d l S Q\n", y, y))
	}

	// 辅助绘制区块小标题
	drawSectionTitle := func(title string) {
		curY -= 18
		stream.WriteString("q 0.94 0.96 0.98 rg 45 " + fmt.Sprintf("%d", curY-4) + " 505 18 re f Q\n")
		stream.WriteString("BT\n0.09 0.28 0.58 rg\n/F2 10 Tf\n")
		stream.WriteString(fmt.Sprintf("52 %d Td\n", curY))
		stream.WriteString(encodeUTF16BEHex(title))
		stream.WriteString(" Tj\nET\n")
		curY -= 8
	}

	// 辅助输出键值对单行
	drawRow2 := func(k1, v1, k2, v2 string) {
		curY -= 15
		stream.WriteString("BT\n0.2 0.2 0.2 rg\n/F2 8.5 Tf\n")
		stream.WriteString(fmt.Sprintf("52 %d Td\n", curY))
		stream.WriteString(encodeUTF16BEHex(fmt.Sprintf("%-6s %s", k1, v1)))
		stream.WriteString(" Tj\nET\n")

		if k2 != "" {
			stream.WriteString("BT\n0.2 0.2 0.2 rg\n/F2 8.5 Tf\n")
			stream.WriteString(fmt.Sprintf("310 %d Td\n", curY))
			stream.WriteString(encodeUTF16BEHex(fmt.Sprintf("%-6s %s", k2, v2)))
			stream.WriteString(" Tj\nET\n")
		}
	}

	// 辅助输出换行长文本
	drawLongText := func(label, content string) {
		if content == "" {
			content = "无特殊记录 / 暂缺"
		}
		curY -= 15
		stream.WriteString("BT\n0.15 0.15 0.15 rg\n/F2 8.5 Tf\n")
		stream.WriteString(fmt.Sprintf("52 %d Td\n", curY))
		stream.WriteString(encodeUTF16BEHex(label + ": "))
		stream.WriteString(" Tj\nET\n")

		// 字符折行 (每行约 36 个汉字 / 72 个英文字符)
		runes := []rune(content)
		maxPerLine := 36
		for i := 0; i < len(runes); i += maxPerLine {
			end := i + maxPerLine
			if end > len(runes) {
				end = len(runes)
			}
			chunk := string(runes[i:end])
			curY -= 13
			stream.WriteString("BT\n0.25 0.25 0.25 rg\n/F2 8 Tf\n")
			stream.WriteString(fmt.Sprintf("65 %d Td\n", curY))
			stream.WriteString(encodeUTF16BEHex(chunk))
			stream.WriteString(" Tj\nET\n")
		}
	}

	// 2. 基础就诊信息
	drawSectionTitle("一、就诊档案与经治机构")
	drawRow2("病历编号:", data.RecordNo, "就诊类型:", data.EncounterType)
	drawRow2("接诊科室:", data.DepartmentName, "就诊时间:", data.CreatedAt.Format("2006-01-02 15:04:05"))
	drawRow2("患者姓名:", data.PatientName, "身份证号:", data.PatientIDCard)
	drawRow2("联系电话:", data.PatientPhone, "经治医师:", data.DoctorName)
	curY -= 4
	drawHLine(curY)

	// 3. 临床病情与诊治明细
	drawSectionTitle("二、临床病情与诊治明细")
	drawRow2("发病时间:", data.OnsetTime, "持续病程:", data.Duration)
	drawLongText("主诉症状", data.ChiefComplaint)
	drawLongText("现病史", data.PresentIllness)
	drawLongText("生命体征", data.VitalSigns)
	if data.ExamResult != "" {
		drawLongText("辅助检查及医技结论", data.ExamResult)
	}
	if data.InitialDiagnosis != "" {
		drawLongText("初步诊断及依据", fmt.Sprintf("%s (%s)", data.InitialDiagnosis, data.DiagnosticBasis))
	}
	drawLongText("最终确诊", data.Diagnosis)
	if data.Etiology != "" {
		drawLongText("病因分析与诱因", data.Etiology)
	}
	drawLongText("处置与治疗方案", data.TreatmentPlan)
	curY -= 4
	drawHLine(curY)

	// 4. 区块链分布式账本与 IPFS 存证凭据
	drawSectionTitle("三、区块链分布式账本与密码学存证背书")
	stream.WriteString("BT\n0.05 0.45 0.25 rg\n/F2 8.5 Tf\n")
	stream.WriteString(fmt.Sprintf("52 %d Td\n", curY-14))
	stream.WriteString(encodeUTF16BEHex("● 底层链网络: Hyperledger Fabric 2.5 (Channel: medchannel, Chaincode: medical)"))
	stream.WriteString(" Tj\nET\n")
	curY -= 15

	drawRow2("存证交易 TxID:", data.FabricTxID, "区块高度:", fmt.Sprintf("#%d", data.BlockHeight))
	drawLongText("IPFS 密文寻址 CID", data.IPFSCID)
	drawLongText("原始附件哈希 (FileHash)", data.FileHash)
	drawLongText("临床综合摘要哈希 (ClinicalHash)", data.ClinicalHash)
	curY -= 4
	drawHLine(curY)

	// 5. 防伪声明与电子防伪签章
	drawSectionTitle("四、电子防伪与安全验真声明")
	stream.WriteString("BT\n0.4 0.4 0.4 rg\n/F2 7.5 Tf\n")
	stream.WriteString(fmt.Sprintf("52 %d Td\n", curY-12))
	stream.WriteString(encodeUTF16BEHex("1. 本凭证由 MedTrust 平台基于国密及国际密码学算法生成，具有不可篡改与防抵赖特性。"))
	stream.WriteString(" Tj\nET\n")
	stream.WriteString("BT\n0.4 0.4 0.4 rg\n/F2 7.5 Tf\n")
	stream.WriteString(fmt.Sprintf("52 %d Td\n", curY-24))
	stream.WriteString(encodeUTF16BEHex("2. 任何字段修改将导致链上 ClinicalHash 校验失败。真伪核验请登录平台安全验真中心。"))
	stream.WriteString(" Tj\nET\n")
	curY -= 26

	// 签章栏
	stream.WriteString("BT\n0.09 0.28 0.58 rg\n/F2 8.5 Tf\n")
	stream.WriteString(fmt.Sprintf("340 %d Td\n", curY-14))
	stream.WriteString(encodeUTF16BEHex(fmt.Sprintf("出具医生电子签章: %s", data.DoctorName)))
	stream.WriteString(" Tj\nET\n")

	stream.WriteString("BT\n0.4 0.4 0.4 rg\n/F2 7.5 Tf\n")
	stream.WriteString(fmt.Sprintf("340 %d Td\n", curY-26))
	stream.WriteString(encodeUTF16BEHex(fmt.Sprintf("签发时间: %s", time.Now().Format("2006-01-02 15:04:05"))))
	stream.WriteString(" Tj\nET\n")

	contentBytes := stream.Bytes()

	// 组装完整的标准 PDF 1.4 对象体系
	var pdf bytes.Buffer
	offsets := make([]int, 9) // 对象 1 到 8

	pdf.WriteString("%PDF-1.4\n%\xe2\xe3\xcf\xd3\n")

	// 1 0 obj: Catalog
	offsets[1] = pdf.Len()
	pdf.WriteString("1 0 obj\n<< /Type /Catalog /Pages 2 0 R >>\nendobj\n")

	// 2 0 obj: Pages
	offsets[2] = pdf.Len()
	pdf.WriteString("2 0 obj\n<< /Type /Pages /Kids [3 0 R] /Count 1 >>\nendobj\n")

	// 3 0 obj: Page
	offsets[3] = pdf.Len()
	pdf.WriteString("3 0 obj\n<< /Type /Page /Parent 2 0 R /MediaBox [0 0 595.28 841.89] ")
	pdf.WriteString("/Resources << /Font << /F1 4 0 R /F2 5 0 R >> /ProcSet [/PDF /Text /ImageB /ImageC /ImageI] >> ")
	pdf.WriteString("/Contents 6 0 R >>\nendobj\n")

	// 4 0 obj: Font F1 (Standard Helvetica)
	offsets[4] = pdf.Len()
	pdf.WriteString("4 0 obj\n<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica >>\nendobj\n")

	// 5 0 obj: Font F2 (STSong-Light Type0 CID Font with UniGB-UTF16-H)
	offsets[5] = pdf.Len()
	pdf.WriteString("5 0 obj\n<< /Type /Font /Subtype /Type0 /BaseFont /STSong-Light /Encoding /UniGB-UTF16-H /DescendantFonts [7 0 R] >>\nendobj\n")

	// 6 0 obj: Contents Stream
	offsets[6] = pdf.Len()
	pdf.WriteString(fmt.Sprintf("6 0 obj\n<< /Length %d >>\nstream\n", len(contentBytes)))
	pdf.Write(contentBytes)
	pdf.WriteString("\nendstream\nendobj\n")

	// 7 0 obj: Descendant CIDFont
	offsets[7] = pdf.Len()
	pdf.WriteString("7 0 obj\n<< /Type /Font /Subtype /CIDFontType2 /BaseFont /STSong-Light ")
	pdf.WriteString("/CIDSystemInfo << /Registry (Adobe) /Ordering (GB1) /Supplement 4 >> ")
	pdf.WriteString("/FontDescriptor 8 0 R /DW 1000 >>\nendobj\n")

	// 8 0 obj: FontDescriptor
	offsets[8] = pdf.Len()
	pdf.WriteString("8 0 obj\n<< /Type /FontDescriptor /FontName /STSong-Light /Flags 32 /ItalicAngle 0 ")
	pdf.WriteString("/Ascent 880 /Descent -120 /CapHeight 880 /StemV 80 >>\nendobj\n")

	// xref table
	xrefOffset := pdf.Len()
	pdf.WriteString("xref\n")
	pdf.WriteString("0 9\n")
	pdf.WriteString("0000000000 65535 f \n")
	for i := 1; i <= 8; i++ {
		pdf.WriteString(fmt.Sprintf("%010d 00000 n \n", offsets[i]))
	}

	// trailer
	pdf.WriteString("trailer\n<< /Size 9 /Root 1 0 R >>\nstartxref\n")
	pdf.WriteString(fmt.Sprintf("%d\n%%%%EOF\n", xrefOffset))

	return pdf.Bytes(), nil
}
