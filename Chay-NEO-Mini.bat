@echo off
setlocal
title Khoi dong NEO Mini

cd /d "%~dp0"

if not exist "be\deploy\.env" (
    echo Khong tim thay be\deploy\.env.
    echo Hay tao file nay tu be\deploy\.env.example va dat cac mat khau local truoc.
    pause
    exit /b 1
)

echo Dang kiem tra cac cong cua du an...
powershell.exe -NoProfile -ExecutionPolicy Bypass -Command "$occupied=Get-NetTCPConnection -State Listen -ErrorAction SilentlyContinue | Where-Object { $_.LocalPort -in 3000,8080,50051,50052 }; if($occupied){$occupied | Select-Object LocalPort,OwningProcess | Format-Table -AutoSize; exit 1}"
if errorlevel 1 (
    echo Du an cu van dang chay: 3000=Frontend, 8080=BFF, 50051=Order, 50052=Stock.
    echo Hay dong cac cua so dich vu cu bang Ctrl+C, sau do nhap dup file nay lai.
    echo Cot OwningProcess o tren la PID de tim tien trinh trong Task Manager.
    pause
    exit /b 1
)

echo [1/3] Dang bat PostgreSQL va NATS, cho PostgreSQL san sang va chay migration...
powershell.exe -NoProfile -ExecutionPolicy Bypass -Command "$ErrorActionPreference='Stop'; Set-Location '%~dp0be'; & '.\scripts\dev.ps1' up; if($LASTEXITCODE -ne 0){exit $LASTEXITCODE}; $deadline=(Get-Date).AddMinutes(2); $ready=$false; do {$health=docker inspect --format '{{.State.Health.Status}}' deploy-postgres-1 2>$null; $nats=Test-NetConnection 127.0.0.1 -Port 4222 -InformationLevel Quiet -WarningAction SilentlyContinue; if($health -eq 'healthy' -and $nats){$ready=$true; break}; Start-Sleep -Seconds 2} while((Get-Date) -lt $deadline); if(-not $ready){Write-Error 'PostgreSQL hoac NATS chua san sang. Hay kiem tra Docker Desktop.'; exit 1}; & '.\scripts\dev.ps1' migrate; exit $LASTEXITCODE"
if errorlevel 1 (
    echo Khoi dong ha tang hoac migration that bai. Cua so nay se dung de ban xem loi.
    pause
    exit /b 1
)

echo [2/3] Dang mo cac cua so backend...
start "NEO Mini - Stock" powershell.exe -NoExit -NoProfile -ExecutionPolicy Bypass -Command "Set-Location '%~dp0be'; & '.\scripts\dev.ps1' stock"
start "NEO Mini - Order" powershell.exe -NoExit -NoProfile -ExecutionPolicy Bypass -Command "Set-Location '%~dp0be'; & '.\scripts\dev.ps1' order"
start "NEO Mini - BFF" powershell.exe -NoExit -NoProfile -ExecutionPolicy Bypass -Command "Set-Location '%~dp0be'; & '.\scripts\dev.ps1' bff"

echo [3/3] Dang mo frontend. Lan dau co the mat them thoi gian cai Node.js 24 va dependencies.
start "NEO Mini - Frontend" powershell.exe -NoExit -NoProfile -ExecutionPolicy Bypass -Command "Set-Location '%~dp0fe'; try { $nodeExe=(npx --yes node@24 -p process.execPath 2>$null).Trim(); if($nodeExe){$env:PATH=(Split-Path -Parent $nodeExe)+';'+$env:PATH} } catch {}; node --version; if (Get-Command corepack -ErrorAction SilentlyContinue) { corepack pnpm@11.0.0 install --frozen-lockfile; if($LASTEXITCODE -eq 0){corepack pnpm@11.0.0 dev} } else { pnpm install; pnpm run dev }"

echo Da gui lenh khoi dong. Mo http://localhost:3000 sau khi frontend bao san sang.
echo Cac cua so backend/frontend can duoc giu mo trong luc su dung.
pause
