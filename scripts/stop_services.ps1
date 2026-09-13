# stop_services.ps1 - Stop MedTrust microservices cleanly
[Console]::OutputEncoding = [System.Text.Encoding]::UTF8

$ports = @(8080, 8000, 5173, 8090)
$names = @{
    8080 = "Go 后端 API 网关"
    8000 = "Python AI Agent 服务"
    5173 = "Vue 3 前端服务"
    8090 = "区块链攻防演示控制台"
}

Write-Host "==============================================================================" -ForegroundColor Cyan
Write-Host "[MedTrust] 正在扫描并终止运行中的微服务..." -ForegroundColor Cyan
Write-Host "==============================================================================" -ForegroundColor Cyan

foreach ($port in $ports) {
    $conns = Get-NetTCPConnection -LocalPort $port -State Listen -ErrorAction SilentlyContinue
    if ($conns) {
        foreach ($conn in $conns) {
            $procId = $conn.OwningProcess
            if ($procId -gt 0 -and $procId -ne $PID) {
                try {
                    $p = Get-Process -Id $procId -ErrorAction SilentlyContinue
                    if ($p) {
                        $pName = $p.ProcessName
                        Stop-Process -Id $procId -Force -ErrorAction SilentlyContinue
                        Write-Host "[√ 已停止] $($names[$port]) - 端口: $port | PID: $procId ($pName)" -ForegroundColor Green
                    }
                } catch {
                    Write-Host "[!] 终止 PID $procId 失败: $_" -ForegroundColor Yellow
                }
            }
        }
    } else {
        Write-Host "[○ 未运行] $($names[$port]) - 端口: $port" -ForegroundColor Gray
    }
}

# Clean residual medtrust_backend.exe
$backendProcs = Get-Process -Name "medtrust_backend" -ErrorAction SilentlyContinue
if ($backendProcs) {
    $backendProcs | Stop-Process -Force -ErrorAction SilentlyContinue
    Write-Host "[√ 已清理] 终止残留 medtrust_backend.exe 进程" -ForegroundColor Green
}

Write-Host ""
Write-Host "==============================================================================" -ForegroundColor Cyan
Write-Host "[MedTrust] 所有关联服务 (8080, 8000, 5173, 8090) 已全部安全停止！" -ForegroundColor Green
Write-Host "==============================================================================" -ForegroundColor Cyan
