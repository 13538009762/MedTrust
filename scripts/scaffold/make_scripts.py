import os

base = r'e:\EnglishEncoding\competition\last\MedTrust'

def write_bat(filename, content):
    path = os.path.join(base, filename)
    with open(path, 'w', encoding='utf-8') as f:
        f.write(content.strip() + '\n')
    print('Wrote:', filename)

# 1. init_db.bat
write_bat('init_db.bat', """@echo off
chcp 65001 >nul
echo ==============================================================================
echo [MedTrust] 正在初始化数据库 medtrust (MySQL 8.0)...
echo ==============================================================================
python "%~dp0setup_db.py"
if %errorlevel% equ 0 (
    echo [MedTrust] 数据库 9 张核心表及预置演示数据初始化成功！
) else (
    echo [MedTrust] 数据库初始化失败，请确保本地 MySQL 服务已开启。
)
pause
""")

# 2. start_backend.bat
write_bat('start_backend.bat', """@echo off
chcp 65001 >nul
title MedTrust Go 核心业务与安全网关服务
echo ==============================================================================
echo [MedTrust] 正在启动 Go 核心业务与安全网关服务 (端口 8080)...
echo ==============================================================================
cd /d "%~dp0backend"
if exist medtrust_backend.exe (
    medtrust_backend.exe
) else (
    go run ./cmd/main.go
)
pause
""")

# 3. start_ai.bat
write_bat('start_ai.bat', """@echo off
chcp 65001 >nul
title MedTrust Python 受控 AI Agent 服务
echo ==============================================================================
echo [MedTrust] 正在启动 Python FastAPI 受控 AI Agent 独立服务 (端口 8000)...
echo ==============================================================================
cd /d "%~dp0ai_agent"
python -m uvicorn main:app --host 127.0.0.1 --port 8000
pause
""")

# 4. start_frontend.bat
write_bat('start_frontend.bat', """@echo off
chcp 65001 >nul
title MedTrust Vue3 现代化前端
echo ==============================================================================
echo [MedTrust] 正在启动 Vue3 + Vite 前端开发服务器 (端口 5173)...
echo ==============================================================================
cd /d "%~dp0frontend"
npm run dev
pause
""")

# 5. start_all.bat
write_bat('start_all.bat', """@echo off
chcp 65001 >nul
echo ==============================================================================
echo [MedTrust] 跨医院医疗数据可信共享系统 2.0 - 一键并发启动服务
echo ==============================================================================
echo 1. 正在拉起 Go 核心业务与安全网关服务 (8080 端口)...
start "MedTrust Go Backend (Port 8080)" cmd /k "%~dp0start_backend.bat"

echo 2. 正在拉起 Python FastAPI 受控 AI Agent (8000 端口)...
start "MedTrust Python AI Agent (Port 8000)" cmd /k "%~dp0start_ai.bat"

echo 3. 正在拉起 Vue 3 现代化前端 (5173 端口)...
start "MedTrust Vue3 Frontend (Port 5173)" cmd /k "%~dp0start_frontend.bat"

echo ==============================================================================
echo 所有子服务已在独立终端并发启动！
echo 正在为您打开默认浏览器进入登录演示界面...
timeout /t 3 /nobreak >nul
start http://localhost:5173/login
echo.
echo 系统登录预置账号提示：
echo • 医院A李建国医生: doc_a / 123456
echo • 医院B王明德医生: doc_b / 123456
echo • 患者张三:        pat_zhang / 123456
echo • 卫健监管专员:    supervisor / 123456
echo • 系统管理员:      admin / 123456
echo ==============================================================================
pause
""")

print("All batch scripts written successfully!")
