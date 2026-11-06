#Requires -Version 5.1
<#
    科贸生活 Windows 便携包卸载脚本（docs/002 §5.2「分发与安装」）。

    删进程、快捷方式、自启项与安装目录；**保留** %LOCALAPPDATA%\Kemaolife\ 配置目录，
    因为里面是用户填过的学号与校园网凭据（DPAPI 加密），删掉后重装要重填。
    想彻底清干净，按脚本最后打印的路径手动删除即可。
#>
[CmdletBinding()]
param(
    [switch]$KeepData
)

$ErrorActionPreference = 'Stop'

$AppName     = '科贸生活'
$ExeName     = 'kemaolife-agent.exe'
$ProcessName = 'kemaolife-agent'
$InstallDir  = Join-Path $env:LOCALAPPDATA 'Programs\Kemaolife'
$DataDir     = Join-Path $env:LOCALAPPDATA 'Kemaolife'
$RunValue    = 'KemaolifeAgent'
$RunKey      = 'HKCU:\Software\Microsoft\Windows\CurrentVersion\Run'

function Write-Step([string]$Message) {
    Write-Host "  $Message"
}

Write-Host ""
Write-Host "$AppName 卸载" -ForegroundColor Cyan
Write-Host ""

# 1. 结束进程：不结束就删不掉 exe，也会留下一个指向空目录的托盘图标。
$procs = @(Get-Process -Name $ProcessName -ErrorAction SilentlyContinue)
if ($procs.Count -gt 0) {
    Write-Step "结束正在运行的 $AppName"
    foreach ($p in $procs) {
        try { $p.CloseMainWindow() | Out-Null } catch { }
    }
    Start-Sleep -Milliseconds 800
    $left = @(Get-Process -Name $ProcessName -ErrorAction SilentlyContinue)
    foreach ($p in $left) {
        try { Stop-Process -Id $p.Id -Force -ErrorAction Stop } catch { }
    }
    Start-Sleep -Milliseconds 500
}

# 2. 自启项：程序被删掉后这条注册表项会变成每次登录都静默失败的死项，必须清掉。
try {
    if (Get-ItemProperty -Path $RunKey -Name $RunValue -ErrorAction SilentlyContinue) {
        Remove-ItemProperty -Path $RunKey -Name $RunValue -Force
        Write-Step "已删除开机自启项 $RunValue"
    }
} catch {
    Write-Warning "删除开机自启项失败：$($_.Exception.Message)"
}

# 3. 开始菜单快捷方式。
$shortcut = Join-Path (Join-Path $env:APPDATA 'Microsoft\Windows\Start Menu\Programs') ($AppName + '.lnk')
if (Test-Path -LiteralPath $shortcut) {
    Remove-Item -LiteralPath $shortcut -Force
    Write-Step "已删除开始菜单快捷方式"
}

# 4. 安装目录（只删本程序复制过去的那份，不动下载目录里的原始包）。
if (Test-Path -LiteralPath $InstallDir) {
    $exe = Join-Path $InstallDir $ExeName
    if (Test-Path -LiteralPath $exe) {
        try {
            Remove-Item -LiteralPath $exe -Force
        } catch {
            throw "无法删除 $exe，请确认托盘图标已退出后重试：$($_.Exception.Message)"
        }
    }
    Remove-Item -LiteralPath $InstallDir -Recurse -Force -ErrorAction SilentlyContinue
    Write-Step "已删除安装目录：$InstallDir"
}

Write-Host ""
Write-Host "卸载完成。" -ForegroundColor Green
Write-Host ""

if ($KeepData) {
    Write-Host "配置目录按当前版本策略一律保留：$DataDir"
} else {
    Write-Host "配置目录已保留：$DataDir"
    Write-Host "（含 DPAPI 加密的学号与凭据。确认不再使用时可手动删除整个目录。）"
}
Write-Host ""
