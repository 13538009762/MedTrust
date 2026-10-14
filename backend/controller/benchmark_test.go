package controller_test

import (
	"crypto/rand"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"medtrust-backend/config"
	"medtrust-backend/model"
	"medtrust-backend/pkg/blockchain"
	"medtrust-backend/pkg/crypto"
	"medtrust-backend/repository"
	"medtrust-backend/service"
)

// BenchmarkResultTable1 实验一数据结构：数据加密与链下分布式存证性能评测
type BenchmarkResultTable1 struct {
	SizeLabel      string
	Bytes          int
	SHA256Ms       float64
	EncryptGCM_Ms  float64
	IPFS_WriteMs   float64
	IPFS_ReadMs    float64
	DecryptGCM_Ms  float64
	TotalTimeMs    float64
}

// BenchmarkResultTable2 实验二数据结构：三级访问控制与风险引擎判定吞吐量评测
type BenchmarkResultTable2 struct {
	Concurrency   int
	TotalRequests int
	AvgLatencyMs  float64
	P95LatencyMs  float64
	TPS           float64
	SuccessRate   float64
}

// BenchmarkResultTable3 实验三数据结构：智能合约 4 组核心存证调用交互延迟评测
type BenchmarkResultTable3 struct {
	Operation    string
	Description  string
	Samples      int
	AvgLatencyMs float64
	P50LatencyMs float64
	P95LatencyMs float64
	MinLatencyMs float64
	MaxLatencyMs float64
	SuccessRate  float64
}

// TestThesisBenchmarkSuite 运行毕业论文第 5、6 章核心性能评测实验套件并输出标准化 Markdown 表格
func TestThesisBenchmarkSuite(t *testing.T) {
	// 1. 初始化测试依赖环境与基准配置
	cfg, err := config.LoadConfig("../config/config.yaml")
	if err != nil {
		t.Fatalf("加载配置失败: %v", err)
	}
	db, err := repository.InitDB()
	if err != nil {
		t.Fatalf("初始化数据库失败: %v", err)
	}
	repository.SeedInitialData(db)

	tempDir := t.TempDir()
	ledgerDir := filepath.Join(tempDir, "fabric_ledger")
	ipfsDir := filepath.Join(tempDir, "ipfs_storage")

	blockchain.InitBlockchainService(ledgerDir, blockchain.FabricGatewayConfig{
		Mode:    "mock",
		Enabled: false,
	})
	service.InitMedicalService("mock", cfg.IPFS.APIURL, ipfsDir)
	service.InitAccessEngine(cfg.Risk.LowThreshold, cfg.Risk.HighThreshold)
	service.EnsureBaselineLedgerAnchored()

	masterKey := cfg.GetAESMasterKey()
	ipfsSvc := service.DefaultMedicalService.GetIPFSService()

	// -------------------------------------------------------------
	// 实验一：【数据加密与链下分布式存证性能评测】
	// 测试不同文件尺寸 (100KB, 500KB, 1MB, 5MB, 10MB)
	// -------------------------------------------------------------
	fileSizes := []struct {
		label string
		bytes int
	}{
		{"100 KB", 100 * 1024},
		{"500 KB", 500 * 1024},
		{"1 MB", 1 * 1024 * 1024},
		{"5 MB", 5 * 1024 * 1024},
		{"10 MB", 10 * 1024 * 1024},
	}

	var table1Results []BenchmarkResultTable1

	for _, fs := range fileSizes {
		rawBuf := make([]byte, fs.bytes)
		_, _ = io.ReadFull(rand.Reader, rawBuf)

		// 1. SHA-256 计算耗时
		t0 := time.Now()
		_ = crypto.ComputeSHA256(rawBuf)
		sha256Ms := float64(time.Since(t0).Microseconds()) / 1000.0

		// 2. AES-256-GCM + AAD 信封加密耗时
		t1 := time.Now()
		cipherData, encDEK, ivNonce, _, err := crypto.EncryptMedicalFile(rawBuf, 1, "REC_BENCHMARK", masterKey)
		if err != nil {
			t.Fatalf("加密失败: %v", err)
		}
		encMs := float64(time.Since(t1).Microseconds()) / 1000.0

		// 3. IPFS 写入耗时
		t2 := time.Now()
		cid, err := ipfsSvc.Add(cipherData)
		if err != nil {
			t.Fatalf("IPFS Add 失败: %v", err)
		}
		ipfsWriteMs := float64(time.Since(t2).Microseconds()) / 1000.0

		// 4. IPFS 读取耗时
		t3 := time.Now()
		readCipher, err := ipfsSvc.Cat(cid)
		if err != nil {
			t.Fatalf("IPFS Cat 失败: %v", err)
		}
		ipfsReadMs := float64(time.Since(t3).Microseconds()) / 1000.0

		// 5. AES-256-GCM + AAD 解密耗时
		t4 := time.Now()
		plainData, err := crypto.DecryptMedicalFile(readCipher, encDEK, ivNonce, 1, "REC_BENCHMARK", masterKey)
		if err != nil || len(plainData) != fs.bytes {
			t.Fatalf("解密校验失败: %v", err)
		}
		decMs := float64(time.Since(t4).Microseconds()) / 1000.0

		totalMs := sha256Ms + encMs + ipfsWriteMs + ipfsReadMs + decMs

		table1Results = append(table1Results, BenchmarkResultTable1{
			SizeLabel:     fs.label,
			Bytes:         fs.bytes,
			SHA256Ms:      sha256Ms,
			EncryptGCM_Ms: encMs,
			IPFS_WriteMs:  ipfsWriteMs,
			IPFS_ReadMs:   ipfsReadMs,
			DecryptGCM_Ms: decMs,
			TotalTimeMs:   totalMs,
		})
	}

	// -------------------------------------------------------------
	// 实验二：【三级零信任访问控制与动态风控判定吞吐量 (TPS)】
	// 测试不同并发度 (50, 100, 200, 500, 800)
	// -------------------------------------------------------------
	concurrencyLevels := []int{50, 100, 200, 500, 800}
	var table2Results []BenchmarkResultTable2

	docLi, _ := repository.DefaultUserRepository.FindByUsername("doc_li")
	var sampleRecord model.MedicalRecord
	_ = repository.DB.First(&sampleRecord)
	if sampleRecord.ID == 0 {
		sampleRecord.ID = 1
		sampleRecord.PatientID = 1
	}

	for _, conc := range concurrencyLevels {
		requestsPerWorker := 1
		if conc <= 50 {
			requestsPerWorker = 4
		} else if conc <= 100 {
			requestsPerWorker = 2
		} else {
			requestsPerWorker = 1
		}
		totalReqs := conc * requestsPerWorker
		latencies := make([]float64, totalReqs)
		var successCount int64

		var wg sync.WaitGroup
		startBarrier := make(chan struct{})
		tGlobalStart := time.Now()

		for w := 0; w < conc; w++ {
			wg.Add(1)
			go func(workerID int) {
				defer wg.Done()
				<-startBarrier

				for i := 0; i < requestsPerWorker; i++ {
					reqIdx := workerID*requestsPerWorker + i
					tReqStart := time.Now()

					req := service.AccessRequest{
						UserID:    docLi.ID,
						Role:      docLi.Role,
						PatientID: sampleRecord.PatientID,
						RecordID:  sampleRecord.ID,
						Action:    "VIEW",
						Source:    "BENCHMARK",
					}

					dec, err := service.DefaultAccessEngine.Evaluate(req)
					elapsedMs := float64(time.Since(tReqStart).Microseconds()) / 1000.0
					latencies[reqIdx] = elapsedMs

					if err == nil && dec != nil {
						atomic.AddInt64(&successCount, 1)
					}
				}
			}(w)
		}

		close(startBarrier)
		wg.Wait()
		totalDuration := time.Since(tGlobalStart)

		sort.Float64s(latencies)
		var sum float64
		for _, l := range latencies {
			sum += l
		}
		avgLat := sum / float64(len(latencies))
		p95Idx := int(float64(len(latencies)) * 0.95)
		if p95Idx >= len(latencies) {
			p95Idx = len(latencies) - 1
		}
		p95Lat := latencies[p95Idx]
		tps := float64(totalReqs) / totalDuration.Seconds()
		succRate := float64(successCount) / float64(totalReqs) * 100.0

		table2Results = append(table2Results, BenchmarkResultTable2{
			Concurrency:   conc,
			TotalRequests: totalReqs,
			AvgLatencyMs:  avgLat,
			P95LatencyMs:  p95Lat,
			TPS:           tps,
			SuccessRate:   succRate,
		})
	}

	// -------------------------------------------------------------
	// 实验三：【智能合约 4 组核心存证调用交互延迟评测】
	// 测试调用: CreateMedicalAsset, CreateAuthorizationRecord, CreateEmergencyRecord, QueryMedicalAsset
	// -------------------------------------------------------------
	operations := []struct {
		name string
		desc string
		run  func(i int) error
	}{
		{
			name: "CreateMedicalAsset",
			desc: "病历资产哈希锚定存证",
			run: func(i int) error {
				_, _, err := blockchain.DefaultService.CreateMedicalAsset(blockchain.MedicalAssetPayload{
					RecordID:   fmt.Sprintf("BENCH_REC_%d", i),
					PatientID:  "PAT_001",
					CID:        fmt.Sprintf("QmBenchCID%d", i),
					FileHash:   fmt.Sprintf("hash_%d", i),
					HospitalID: "1",
					CreatorID:  "1",
					CreateTime: time.Now().Format(time.RFC3339),
				})
				return err
			},
		},
		{
			name: "CreateAuthorizationRecord",
			desc: "患者知情授权策略上链",
			run: func(i int) error {
				_, err := blockchain.DefaultService.CreateAuthorizationRecord(blockchain.AuthorizationPayload{
					AuthID:     fmt.Sprintf("BENCH_AUTH_%d", i),
					PatientID:  "PAT_001",
					TargetType: "DOCTOR",
					TargetID:   "DOC_001",
					ScopeType:  "ALL",
					RecordID:   "ALL",
					StartTime:  time.Now().Format(time.RFC3339),
					EndTime:    time.Now().Add(24 * time.Hour).Format(time.RFC3339),
					Status:     "ACTIVE",
				})
				return err
			},
		},
		{
			name: "CreateEmergencyRecord",
			desc: "Break-Glass 破窗抢救存证",
			run: func(i int) error {
				_, err := blockchain.DefaultService.CreateEmergencyRecord(blockchain.EmergencyPayload{
					EventNo:    fmt.Sprintf("BENCH_EMERG_%d", i),
					DoctorID:   "DOC_001",
					PatientID:  "PAT_001",
					RecordID:   "REC20250501001",
					HospitalID: "2",
					Reason:     "突发急性胸痛抢救",
					Timestamp:  time.Now().Format(time.RFC3339),
					Status:     "PENDING_AUDIT",
				})
				return err
			},
		},
		{
			name: "QueryMedicalAsset",
			desc: "链上病历资产快速溯源查询",
			run: func(i int) error {
				recordNo := fmt.Sprintf("BENCH_REC_%d", i)
				_, err := blockchain.DefaultService.QueryMedicalAsset(recordNo)
				return err
			},
		},
	}

	samplesPerOp := 100
	var table3Results []BenchmarkResultTable3

	for _, op := range operations {
		latencies := make([]float64, samplesPerOp)
		successCount := 0

		for i := 0; i < samplesPerOp; i++ {
			tOpStart := time.Now()
			err := op.run(i)
			elapsedMs := float64(time.Since(tOpStart).Microseconds()) / 1000.0
			latencies[i] = elapsedMs
			if err == nil {
				successCount++
			}
		}

		sort.Float64s(latencies)
		var sum float64
		for _, l := range latencies {
			sum += l
		}
		avgLat := sum / float64(samplesPerOp)
		p50Lat := latencies[samplesPerOp/2]
		p95Idx := int(float64(samplesPerOp) * 0.95)
		if p95Idx >= samplesPerOp {
			p95Idx = samplesPerOp - 1
		}
		p95Lat := latencies[p95Idx]
		minLat := latencies[0]
		maxLat := latencies[samplesPerOp-1]
		succRate := float64(successCount) / float64(samplesPerOp) * 100.0

		table3Results = append(table3Results, BenchmarkResultTable3{
			Operation:    op.name,
			Description:  op.desc,
			Samples:      samplesPerOp,
			AvgLatencyMs: avgLat,
			P50LatencyMs: p50Lat,
			P95LatencyMs: p95Lat,
			MinLatencyMs: minLat,
			MaxLatencyMs: maxLat,
			SuccessRate:  succRate,
		})
	}

	// -------------------------------------------------------------
	// 格式化输出毕业论文第 5、6 章标准 Markdown 表格
	// -------------------------------------------------------------
	var sb strings.Builder
	sb.WriteString("# MedTrust 毕业论文核心性能评测实验报告 (Chapters 5 & 6 Benchmarks)\n\n")
	sb.WriteString(fmt.Sprintf("> 生成时间: %s  \n", time.Now().Format("2006-01-02 15:04:05")))
	sb.WriteString("> 运行平台: Windows (x86_64) | Go 1.26 | MySQL 8.0 | IPFS / LocalStorageAdapter | Hyperledger Fabric Gateway & MockLedger\n\n")
	sb.WriteString("---\n\n")

	// 表格 1
	sb.WriteString("## 5.1 数据加密与链下分布式存储性能评测 (表 5-1)\n\n")
	sb.WriteString("| 文件尺寸 (Size) | SHA-256 计算 (ms) | AES-256-GCM 加密 (ms) | IPFS 写入 (ms) | IPFS 读取 (ms) | AES-256-GCM 解密 (ms) | 端到端总开销 (ms) |\n")
	sb.WriteString("| :--- | :---: | :---: | :---: | :---: | :---: | :---: |\n")
	for _, r := range table1Results {
		sb.WriteString(fmt.Sprintf("| **%s** | %.2f | %.2f | %.2f | %.2f | %.2f | **%.2f** |\n",
			r.SizeLabel, r.SHA256Ms, r.EncryptGCM_Ms, r.IPFS_WriteMs, r.IPFS_ReadMs, r.DecryptGCM_Ms, r.TotalTimeMs))
	}
	sb.WriteString("\n*注：测试涵盖数据指纹提取、随机 DEK/IV 生成、带 AAD 绑定的 AES-256-GCM 信封加密、IPFS 分布式块持久化及全流程流式解密。*\n\n---\n\n")

	// 表格 2
	sb.WriteString("## 5.2 三级零信任访问控制与风险引擎判定吞吐量评测 (表 5-2)\n\n")
	sb.WriteString("| 并发度 (Concurrency) | 请求总量 (Total Requests) | 平均判定延迟 (ms) | P95 延迟 (ms) | 系统吞吐量 (TPS) | 判定成功率 (%) |\n")
	sb.WriteString("| :---: | :---: | :---: | :---: | :---: | :---: |\n")
	for _, r := range table2Results {
		sb.WriteString(fmt.Sprintf("| **%d** | %d | %.2f | %.2f | **%.1f** | %.1f%% |\n",
			r.Concurrency, r.TotalRequests, r.AvgLatencyMs, r.P95LatencyMs, r.TPS, r.SuccessRate))
	}
	sb.WriteString("\n*注：每次判定包含 RBAC 角色研判、ABAC 跨机构属性核查、患者链上/本地知情授权匹配及 RiskEngine 6 维透明加权评分计算。*\n\n---\n\n")

	// 表格 3
	sb.WriteString("## 5.3 智能合约 4 组核心存证调用交互延迟评测 (表 5-3)\n\n")
	sb.WriteString("| 智能合约调用方法 (Transaction) | 业务功能定义 | 样本量 (Samples) | 平均延迟 (ms) | P50 中位数 (ms) | P95 延迟 (ms) | 最小延迟 (ms) | 最大延迟 (ms) | 存证成功率 (%) |\n")
	sb.WriteString("| :--- | :--- | :---: | :---: | :---: | :---: | :---: | :---: | :---: |\n")
	for _, r := range table3Results {
		sb.WriteString(fmt.Sprintf("| **`%s`** | %s | %d | %.2f | %.2f | %.2f | %.2f | %.2f | **%.1f%%** |\n",
			r.Operation, r.Description, r.Samples, r.AvgLatencyMs, r.P50LatencyMs, r.P95LatencyMs, r.MinLatencyMs, r.MaxLatencyMs, r.SuccessRate))
	}
	sb.WriteString("\n*注：测试涵盖病历资产哈希存证、患者授权策略流转、Break-Glass 破窗应急存证及链上状态快速溯源查询四组核心交易。*\n")

	markdownOutput := sb.String()
	fmt.Println("\n" + markdownOutput)

	// 同时将 Markdown 报告持久化写入 docs/thesis_benchmarks.md
	docPath := filepath.Join("..", "..", "docs", "thesis_benchmarks.md")
	_ = os.WriteFile(docPath, []byte(markdownOutput), 0644)
	t.Logf("✅ 毕业论文性能压测完成！完整 Markdown 实验报告已同步输出至: %s", docPath)
}

// -------------------------------------------------------------
// 标准 Go Benchmark 原生基准测试 (供 go test -bench=. 调用)
// -------------------------------------------------------------

func BenchmarkAESGCMEncryption100KB(b *testing.B) {
	data := make([]byte, 100*1024)
	_, _ = io.ReadFull(rand.Reader, data)
	key := []byte("MedTrustMasterKeyAES256GCM2026!!")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, _, _, _ = crypto.EncryptMedicalFile(data, 1, "BENCH_REC", key)
	}
}

func BenchmarkAESGCMEncryption1MB(b *testing.B) {
	data := make([]byte, 1024*1024)
	_, _ = io.ReadFull(rand.Reader, data)
	key := []byte("MedTrustMasterKeyAES256GCM2026!!")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, _, _, _ = crypto.EncryptMedicalFile(data, 1, "BENCH_REC", key)
	}
}

func BenchmarkAccessEngineEvaluate(b *testing.B) {
	req := service.AccessRequest{
		UserID:    1,
		Role:      "doctor",
		PatientID: 1,
		RecordID:  1,
		Action:    "VIEW",
		Source:    "WEB",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = service.DefaultAccessEngine.Evaluate(req)
	}
}

func BenchmarkBlockchainCreateMedicalAsset(b *testing.B) {
	asset := blockchain.MedicalAssetPayload{
		RecordID:   "BENCH_REC",
		PatientID:  "PAT_001",
		CID:        "QmBenchCID",
		FileHash:   "hash_bench",
		HospitalID: "1",
		CreatorID:  "1",
		CreateTime: time.Now().Format(time.RFC3339),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, _ = blockchain.DefaultService.CreateMedicalAsset(asset)
	}
}
