# Beango Development Script (PowerShell)
# 同时启动前端开发服务器和后端服务器

param(
    [int]$BackendPort = 10777
)

$ErrorActionPreference = "Stop"
$root = Split-Path -Parent $PSScriptRoot

Write-Host "=== 启动 Beango 开发环境 ===" -ForegroundColor Cyan

# 启动后端
$backendJob = Start-Job -Name "beango-backend" -ScriptBlock {
    param($root)
    Set-Location $root
    go run . 2>&1 | ForEach-Object { "[后端] $_" }
} -ArgumentList $root

# 启动前端开发服务器
$frontendJob = Start-Job -Name "beango-frontend" -ScriptBlock {
    param($root)
    Set-Location "$root\beango-web"
    pnpm run dev 2>&1 | ForEach-Object { "[前端] $_" }
} -ArgumentList $root

Write-Host "后端运行在 http://localhost:$BackendPort" -ForegroundColor Yellow
Write-Host "前端开发服务器启动中..." -ForegroundColor Yellow
Write-Host "按 Ctrl+C 停止所有服务" -ForegroundColor Yellow

try {
    while ($true) {
        Receive-Job -Name "beango-backend", "beango-frontend"
        Start-Sleep -Seconds 2
    }
}
finally {
    Write-Host "`n正在停止服务..." -ForegroundColor Cyan
    Stop-Job -Name "beango-backend", "beango-frontend" -ErrorAction SilentlyContinue
    Remove-Job -Name "beango-backend", "beango-frontend" -ErrorAction SilentlyContinue
    Write-Host "已停止" -ForegroundColor Green
}
