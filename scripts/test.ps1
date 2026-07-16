# Beango Test Script (PowerShell)
# 测试 CLI 基本功能

$ErrorActionPreference = "Stop"
$root = Split-Path -Parent $PSScriptRoot

Write-Host "=== Beango 测试 ===" -ForegroundColor Cyan

# 1. 构建
Write-Host "`n[1/3] 构建..." -ForegroundColor Yellow
Push-Location $root
try {
    go build -o beango-test.exe .
    Write-Host "构建成功" -ForegroundColor Green
}
finally {
    Pop-Location
}

# 2. 测试 CLI help
Write-Host "`n[2/3] 测试 CLI help..." -ForegroundColor Yellow
Push-Location $root
try {
    $help = & ./beango-test.exe -h 2>&1
    if ($help -match "alipay|wechat") {
        Write-Host "CLI help 正常" -ForegroundColor Green
    } else {
        Write-Host "CLI help 异常" -ForegroundColor Red
    }
}
finally {
    Pop-Location
}

# 3. 清理
Write-Host "`n[3/3] 清理测试文件..." -ForegroundColor Yellow
Remove-Item "$root\beango-test.exe" -Force -ErrorAction SilentlyContinue

Write-Host "`n=== 测试完成 ===" -ForegroundColor Green
