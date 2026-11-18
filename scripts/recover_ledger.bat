@echo off
chcp 65001 >nul
cd /d "%~dp0.."
echo ==============================================================================
echo [MedTrust] 联盟链账本重建与病历基准哈希一键恢复工具
echo ==============================================================================
python "%~dp0recover_ledger.py"
if %errorlevel% neq 0 (
    echo.
    echo [ERROR] 恢复或核验过程中存在异常，请查看上方日志。
    pause
    exit /b 1
)
echo.
echo [SUCCESS] 全部病历已恢复正常并完成链上一致性质检！
pause
