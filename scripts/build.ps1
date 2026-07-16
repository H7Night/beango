# Beango Build Script (PowerShell)
# 构建前端 + 后端，输出到 bin/ 目录

param(
    [switch]$FrontendOnly,
    [switch]$BackendOnly
)

$ErrorActionPreference = "Stop"
$root = Split-Path -Parent $PSScriptRoot

$shouldBuildFrontend = !$BackendOnly
$shouldBuildBackend = !$FrontendOnly

if ($shouldBuildFrontend) {
    Write-Host "=== 构建前端 ===" -ForegroundColor Cyan
    Push-Location "$root\beango-web"
    try {
        pnpm install --frozen-lockfile
        pnpm run build
        Write-Host "前端构建完成: beango-web/dist/" -ForegroundColor Green
    }
    finally {
        Pop-Location
    }
}

if ($shouldBuildBackend) {
    Write-Host "=== 构建后端 ===" -ForegroundColor Cyan
    Push-Location $root
    try {
        $outDir = "$root\bin"
        New-Item -ItemType Directory -Path $outDir -Force | Out-Null
        go build -o "$outDir\beango.exe" .
        Write-Host "后端构建完成: bin/beango.exe" -ForegroundColor Green
    }
    finally {
        Pop-Location
    }
}

if ($shouldBuildFrontend -and $shouldBuildBackend) {
    Write-Host "=== 全部构建完成 ===" -ForegroundColor Green
}
