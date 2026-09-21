# build.ps1 — 编译 WSL 托盘工具为单文件 exe
# 用法:在 PowerShell 里执行  .\build.ps1
# 产物: .\dist\WslToggle.exe  (复制给别人直接双击即可)

$ErrorActionPreference = "Stop"
Set-Location -Path $PSScriptRoot

Write-Host "==> 整理 Go 依赖 ..."
go mod tidy

Write-Host "==> 生成多尺寸应用图标 ..."
go run generate_icon.go icon.go
go run github.com/akavel/rsrc@v0.10.2 -ico app.ico -o rsrc_windows_amd64.syso

Write-Host "==> 编译(隐藏控制台 + 瘦身)..."
$env:GOOS = "windows"
$env:GOARCH = "amd64"
$env:CGO_ENABLED = "0"

New-Item -ItemType Directory -Force -Path ".\dist" | Out-Null

go build -trimpath -ldflags "-s -w -H windowsgui" -o ".\dist\WslToggle.exe" .

if (Test-Path ".\dist\WslToggle.exe") {
    $sz = (Get-Item ".\dist\WslToggle.exe").Length
    Write-Host ("==> 成功! WslToggle.exe 体积 {0:N1} MB" -f ($sz / 1MB))
    Write-Host "==> 位置: $((Resolve-Path '.\dist\WslToggle.exe').Path)"
} else {
    Write-Host "!! 编译失败,未生成 exe"
}
