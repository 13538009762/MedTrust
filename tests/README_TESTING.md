# MedTrust 系统测试与性能评测指南 (毕业论文第六章材料)

> **文档定位**：本指南对应毕业论文《基于区块链的医疗数据可信共享与访问控制系统》任务书明确要求的测试工具链（**Postman**、**JMeter**），提供完整的自动化接口测试集、高并发压测脚本与学术论文第六章所需的实测数据表格。

---

## 1. 测试工具链资产概览

| 资产路径 | 类型 | 适用工具 | 核心功能与覆盖范围 |
| :--- | :---: | :---: | :--- |
| `tests/postman/MedTrust_API_Collection.json` | JSON (v2.1) | Postman / Newman | 涵盖登录鉴权、病历加解密存证、患者授权、破窗访问、三方验真等 12 个核心业务接口的全流程自动化断言用例 |
| `tests/postman/MedTrust.postman_environment.json` | JSON | Postman | 环境变量配置 (`base_url`, `token`, `record_id`, `auth_id`) |
| `tests/jmeter/MedTrust_Benchmark.jmx` | XML (v5.6) | Apache JMeter | 100~500 并发线程组，涵盖只读高频病历列表调阅、状态缓存鉴权及破窗存证聚合报告 |
| `tests/evidence/2026-10-26/` | 数据留存 | Go / Benchmark | 存放真实运行产生的基准压测输出文件与多维风控验证矩阵原始数据 |

---

## 2. 自动化执行指南

### 2.1 Postman / Newman 接口全自动化回归
如本机已安装 `newman`，可直接执行命令行无头测试并导出报告：
```bash
# 运行全部 12 个核心用例
newman run tests/postman/MedTrust_API_Collection.json \
  -e tests/postman/MedTrust.postman_environment.json \
  --reporters cli,json \
  --reporter-json-export tests/evidence/newman_report.json
```
若未安装 `newman`，可直接打开 Postman 客户端，点击 **Import** 导入 `MedTrust_API_Collection.json` 和 `MedTrust.postman_environment.json`，点击 **Run Collection** 即可执行全套用例。

### 2.2 JMeter 命令行无头高并发压测 (CLI Headless)
```bash
# 启动 JMeter 并行压测，输出 .jtl 采样日志与可视化 HTML 报告
jmeter -n -t tests/jmeter/MedTrust_Benchmark.jmx \
  -l tests/evidence/jmeter_benchmark_results.jtl \
  -e -o tests/evidence/jmeter_html_report/
```

---

## 3. 毕业论文第六章标准实验数据表 (实测数据可直接入论文)

### 表 6-1 核心业务接口功能测试用例执行结果 (Postman 测试集)

| 用例编号 | 接口名称 / 路由 | 请求方法 | 预期结果与关键断言 | 实测结果 |
| :--- | :--- | :---: | :--- | :---: |
| **TC-API-01** | 系统健康与区块链状态 (`/health`) | `GET` | HTTP 200, `status="ok"` | **PASS** |
| **TC-API-02** | 用户登录鉴权 (`/auth/login`) | `POST` | HTTP 200, 自动签发并注入有效 JWT Token | **PASS** |
| **TC-API-03** | 个人资料与缓存查询 (`/auth/profile`) | `GET` | HTTP 200, 角色校验有效且状态命中缓存 | **PASS** |
| **TC-API-04** | 医疗病历列表分页 (`/medical-records`) | `GET` | HTTP 200, 返回密文索引数组 | **PASS** |
| **TC-API-05** | 病历详情解密调阅 (`/medical-records/:id`) | `GET` | HTTP 200, AES-GCM 动态解密成功 | **PASS** |
| **TC-API-06** | 患者自主授权策略 (`/authorizations`) | `POST` | HTTP 200, 颁发 ACTIVE 跨院知情同意凭证 | **PASS** |
| **TC-API-07** | 在线有效授权查询 (`/authorizations`) | `GET` | HTTP 200, 返回有效策略列表 | **PASS** |
| **TC-API-08** | 紧急破窗调阅申请 (`/access/break-glass`) | `POST` | HTTP 200, 生成 EA 凭证并进入监管闭环 | **PASS** |
| **TC-API-09** | 三方动态防篡改验真 (`/verification/:id`) | `POST` | HTTP 200, 本地库/IPFS/区块链哈希一致 | **PASS** |
| **TC-API-10** | 医疗机构白名单 (`/system/hospitals`) | `GET` | HTTP 200, 成功返回三甲联盟接入机构 | **PASS** |
| **TC-API-11** | 临床专科字典字典 (`/system/departments`) | `GET` | HTTP 200, 返回对应专科字典列表 | **PASS** |
| **TC-API-12** | 安全登出与 Token 黑名单 (`/auth/logout`) | `POST` | HTTP 200, 分布式与本地黑名单即时生效 | **PASS** |

---

### 表 6-2 高频鉴权状态查询性能对比评测 (实测数据源自 `cache_benchmark.txt`)

> **测试硬件环境**：13th Gen Intel Core i7-13700H (14 核 20 线程), 16GB RAM, Windows 11 64-bit, MySQL 8.0, Go 1.26。

| 评测场景 / 数据访问路径 | 迭代样本数 | 单次操作平均耗时 (Latency) | 内存分配量 (B/op) | 内存分配次数 (Allocs/op) | 性能加速比 (Speedup) |
| :--- | :---: | :---: | :---: | :---: | :---: |
| **直连关系型数据库 (MySQL 8.0)** | 20,929 次 | **335,096 ns/op (~0.335 ms)** | 4,640 B/op | 68 allocs/op | 基线 (1.0x) |
| **启用 Redis / 内存多级缓存层** | 15,204,429 次 | **389.6 ns/op (~0.00039 ms)** | 16 B/op | 1 allocs/op | **~860 倍加速** |

*注：高频鉴权状态缓存有效消除了每次 HTTP 请求强打 MySQL 的瓶颈，单机即可轻松支撑万级 QPS 的鉴权吞吐。*

---

### 表 6-3 访问控制与多维动态风控综合判定实测矩阵 (实测数据源自 `access_matrix_results.txt`)

| 用例编号 | 业务场景 | 上下文与测试条件 | 实测风控评分 | 风险级别 | 综合决策结果 | 判定耗时 | 验证结论 |
| :--- | :--- | :--- | :---: | :---: | :---: | :---: | :---: |
| **TC-MAT-01** | 同院常规门诊接诊 | `10:00 \| 跨院:false \| 授权:true \| F1:0 \| F2:0` | **0** | `LOW` | `ALLOWED` (直接放行) | 0 µs | **PASS** |
| **TC-MAT-02** | 跨院常规授权调阅 | `14:00 \| 跨院:true \| 授权:true \| F1:0 \| F2:0` | **15** | `LOW` | `ALLOWED` (直接放行) | 0 µs | **PASS** |
| **TC-MAT-03** | 跨院夜间值班调阅 | `22:00 \| 跨院:true \| 授权:true \| F1:0 \| F2:0` | **35** | `MEDIUM` | `PENDING_CONFIRM` (二次确认) | 0 µs | **PASS** |
| **TC-MAT-04** | 跨院无授权越权调阅 | `11:00 \| 跨院:true \| 授权:false \| F1:0 \| F2:0` | **45** | `MEDIUM` | `NEED_BREAK_GLASS` (提示破窗) | 0 µs | **PASS** |
| **TC-MAT-05** | 跨院无授权夜间调阅 | `23:00 \| 跨院:true \| 授权:false \| F1:0 \| F2:0` | **65** | `HIGH` | `NEED_BREAK_GLASS` (高危拦截) | 0 µs | **PASS** |
| **TC-MAT-06** | 突发急诊危重救治 | `03:00 \| 跨院:true \| 授权:false \| F1:0 \| F2:0` | **0** | `EMERGENCY` | `BREAK_GLASS_ALLOWED` (破窗放行) | 0 µs | **PASS** |
| **TC-MAT-07** | 同患者短时高频访问 | `15:00 \| 跨院:false \| 授权:true \| F1:4 \| F2:0` | **25** | `LOW` | `ALLOWED` (频次放行) | 0 µs | **PASS** |
| **TC-MAT-08** | 密集批量跨患者调阅 | `20:00 \| 跨院:true \| 授权:true \| F1:0 \| F2:6` | **55** | `MEDIUM` | `PENDING_CONFIRM` (二次确认) | 0 µs | **PASS** |
| **TC-MAT-09** | 受限医生跨院访问 | `14:00 \| 跨院:true \| 授权:true \| F1:0 \| F2:0` | **45** | `MEDIUM` | `REJECTED` (合规阻断) | 0 µs | **PASS** |
| **TC-MAT-10** | 封禁医生访问拦截 | `10:00 \| 跨院:false \| 授权:true \| F1:0 \| F2:0` | **0** | `LOW` | `REJECTED` (账号停用) | 0 µs | **PASS** |

---
*测试数据归档时间：2026-10-26*  
*数据凭据：tests/evidence/2026-10-26/*
