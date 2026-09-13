@echo off
chcp 65001 >nul
echo ==============================================================================
echo [MedTrust] 正在初始化数据库 medtrust (MySQL 8.0)...
echo ==============================================================================
python "%~dp0database\setup_db.py"
if %errorlevel% neq 0 (
    echo [MedTrust] 数据库建表失败，请确保本地 MySQL 服务已开启。
    pause
    exit /b 1
)

echo [MedTrust] 正在写入预置演示数据 (医院/医生/患者/电子病历/授权策略)...
python "%~dp0database\seed_db.py"
if %errorlevel% equ 0 (
    echo [MedTrust] 数据库 9 张核心表及预置演示数据初始化成功！
) else (
    echo [MedTrust] 数据预置失败。
)
pause
