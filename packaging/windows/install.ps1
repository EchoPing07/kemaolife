#Requires -Version 5.1
<#
    科贸生活 Windows 便携包安装脚本（docs/002 §5.2「分发与安装」）。

    只做三件事：把 kemaolife-agent.exe 复制到固定目录、建开始菜单快捷方式、启动一次。
    刻意不写自启项：自启只由设置窗口「保存」时写入（该项默认勾选），保证注册表里的
    路径与 os.Executable() 一致；脚本代写会和用户后来开关设置打架。
#>
[CmdletBinding()]
param(
    [switch]$NoLaunch
)

$ErrorActionPreference = 'Stop'

$AppName     = '科贸生活'
$ExeName     = 'kemaolife-agent.exe'
$ProcessName = 'kemaolife-agent'
$InstallDir  = Join-Path $env:LOCALAPPDATA 'Programs\Kemaolife'
$DataDir     = Join-Path $env:LOCALAPPDATA 'Kemaolife'
$SrcDir      = $PSScriptRoot
$SrcExe      = Join-Path $SrcDir $ExeName

function Write-Step([string]$Message) {
    Write-Host "  $Message"
}

function Stop-RunningAgent {
    $procs = @(Get-Process -Name $ProcessName -ErrorAction SilentlyContinue)
    if ($procs.Count -eq 0) { return }
    Write-Step "检测到正在运行的 $AppName，先结束进程（运行中的 exe 无法覆盖）"
    foreach ($p in $procs) {
        try { $p.CloseMainWindow() | Out-Null } catch { }
    }
    Start-Sleep -Milliseconds 800
    $left = @(Get-Process -Name $ProcessName -ErrorAction SilentlyContinue)
    if ($left.Count -gt 0) {
        foreach ($p in $left) {
            try { Stop-Process -Id $p.Id -Force -ErrorAction Stop } catch { }
        }
        Start-Sleep -Milliseconds 500
    }
}

Write-Host ""
Write-Host "$AppName 安装" -ForegroundColor Cyan
Write-Host ""

if (-not (Test-Path -LiteralPath $SrcExe)) {
    throw "找不到 $ExeName。请把本脚本与 $ExeName 解压到同一个目录后再运行。"
}

Stop-RunningAgent

Write-Step "安装目录：$InstallDir"
if (-not (Test-Path -LiteralPath $InstallDir)) {
    New-Item -ItemType Directory -Path $InstallDir -Force | Out-Null
}

Copy-Item -LiteralPath $SrcExe -Destination (Join-Path $InstallDir $ExeName) -Force
foreach ($extra in @('LICENSE', 'INSTALL.txt')) {
    $src = Join-Path $SrcDir $extra
    if (Test-Path -LiteralPath $src) {
        Copy-Item -LiteralPath $src -Destination (Join-Path $InstallDir $extra) -Force
    }
}

$targetExe = Join-Path $InstallDir $ExeName

# 开始菜单快捷方式：放在 Programs 根下，Windows 搜索能直接搜到。
$programs = Join-Path $env:APPDATA 'Microsoft\Windows\Start Menu\Programs'
$shortcut = Join-Path $programs ($AppName + '.lnk')
if (-not (Test-Path -LiteralPath $programs)) {
    New-Item -ItemType Directory -Path $programs -Force | Out-Null
}
try {
    $shell = New-Object -ComObject WScript.Shell
    $link = $shell.CreateShortcut($shortcut)
    $link.TargetPath = $targetExe
    $link.WorkingDirectory = $InstallDir
    $link.Description = $AppName
    $icon = Join-Path $InstallDir $ExeName
    $link.IconLocation = "$icon,0"
    $link.Save()
    Write-Step "已创建开始菜单快捷方式：$shortcut"
} catch {
    Write-Warning "创建开始菜单快捷方式失败（不影响使用，可直接运行 $targetExe）：$($_.Exception.Message)"
}

Write-Host ""
Write-Host "安装完成。" -ForegroundColor Green
Write-Host ""
Write-Host "  - 程序位置：$targetExe"
Write-Host "  - 配置目录：$DataDir（学号与密码经 Windows DPAPI 加密）"
Write-Host "  - 开机自启：设置窗口里「开机登录时自启动」默认勾选，但只有点「保存」才写入注册表"
Write-Host "  - 卸载：运行 uninstall.bat 或 uninstall.ps1（会保留配置目录）"
Write-Host ""
Write-Host "提示：程序常驻系统托盘，关闭设置窗口只是隐藏；结束进程请右键托盘图标选「退出」。"
Write-Host ""

if (-not $NoLaunch) {
    Start-Process -FilePath $targetExe -WorkingDirectory $InstallDir
    Write-Step "已启动 $AppName，请看系统托盘。"
}
