param(
    [string]$Backend = ""
)

$choices = @{
    "1" = @{Name="Go";     Cmd="go run ./backends/go";              Req="go"}
    "2" = @{Name="Node";   Cmd="node backends/node/dev.js";         Req="node"}
    "3" = @{Name="Python"; Cmd="python backends/python/server.py";  Req="python"}
}

if (-not $Backend) {
    Write-Host "=== Qunchat 后端选择 ===" -ForegroundColor Cyan
    Write-Host ""
    foreach ($key in $choices.Keys | Sort-Object) {
        $c = $choices[$key]
        Write-Host "  $key. $($c.Name)"
    }
    Write-Host ""
    $Backend = Read-Host "输入编号 (默认 1)"
    if (-not $Backend) { $Backend = "1" }
}

$choice = $choices[$Backend]
if (-not $choice) {
    Write-Error "无效选项: $Backend"
    exit 1
}

if (-not (Get-Command $choice.Req -ErrorAction SilentlyContinue)) {
    Write-Error "未找到 $($choice.Req)，请先安装 $($choice.Name) 运行时"
    exit 1
}

$projectRoot = Split-Path -Parent $PSCommandPath
Push-Location $projectRoot

Write-Host "启动 $($choice.Name) 后端 (http://localhost:$($env:PORT -or 8080))..." -ForegroundColor Green
switch ($choice.Name) {
    "Go"     { go run ./backends/go }
    "Node"   { node backends/node/dev.js }
    "Python" { python backends/python/server.py }
}

Pop-Location
