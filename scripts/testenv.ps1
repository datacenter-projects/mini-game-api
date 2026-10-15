<#
test env บน Windows ที่ใช้ Docker ไม่ได้ (เช่น Windows Server) — docs/TESTING.md
เปิด Postgres แยก instance + miniredis ตาม port ใน .env.test · ข้อมูลอยู่ใน .testenv/ (ไม่ commit)
เครื่องที่มี Docker ใช้ docker-compose.test.yml แทน

    powershell -ExecutionPolicy Bypass -File scripts\testenv.ps1 up
    powershell -ExecutionPolicy Bypass -File scripts\testenv.ps1 status
    powershell -ExecutionPolicy Bypass -File scripts\testenv.ps1 down
    powershell -ExecutionPolicy Bypass -File scripts\testenv.ps1 reset    # down + ลบข้อมูลทั้งหมด
    powershell -ExecutionPolicy Bypass -File scripts\testenv.ps1 serve    # รัน server ด้วย .env.test (Ctrl+C หยุด)
    powershell -ExecutionPolicy Bypass -File scripts\testenv.ps1 seed -count 100   # บัญชีสำหรับ load test

ต้องมี PostgreSQL binaries (initdb, pg_ctl, psql) — ใน PATH หรือตั้ง $env:PG_BIN
#>
param(
    [ValidateSet('up', 'down', 'reset', 'status', 'serve', 'seed')]
    [string]$Command = 'status',
    [Parameter(ValueFromRemainingArguments = $true)]
    [string[]]$Rest
)
$ErrorActionPreference = 'Stop'

$root = Split-Path -Parent $PSScriptRoot
$state = Join-Path $root '.testenv'
$pgData = Join-Path $state 'pgdata'
$pgLog = Join-Path $state 'postgres.log'
$redisPidFile = Join-Path $state 'miniredis.pid'
$redisExe = Join-Path $state 'miniredis.exe'

function Read-EnvFile([string]$path) {
    $vars = @{}
    foreach ($line in Get-Content $path) {
        if ($line -match '^\s*#' -or $line -notmatch '=') { continue }
        $k, $v = $line -split '=', 2
        $vars[$k.Trim()] = $v.Trim()
    }
    return $vars
}

$cfg = Read-EnvFile (Join-Path $root '.env.test')
$pgPort = [int]$cfg['DB_PORT']
$pgUser = $cfg['DB_USER']
$pgPassword = $cfg['DB_PASSWORD']
$pgDb = $cfg['DB_NAME']
$redisAddr = $cfg['REDIS_ADDR']
$redisPort = [int]($redisAddr -split ':')[-1]

function Get-PgBin {
    if ($env:PG_BIN) { return $env:PG_BIN }
    $cmd = Get-Command pg_ctl.exe -ErrorAction SilentlyContinue
    if ($cmd) { return Split-Path -Parent $cmd.Source }
    throw 'หา PostgreSQL binaries ไม่เจอ — เพิ่มใน PATH หรือตั้ง $env:PG_BIN (เช่น D:\pglocal\pgsql\bin)'
}

function Get-Listener([int]$port) {
    Get-NetTCPConnection -LocalPort $port -State Listen -ErrorAction SilentlyContinue | Select-Object -First 1
}

function Wait-Port([int]$port, [int]$seconds) {
    for ($i = 0; $i -lt $seconds * 4; $i++) {
        if (Get-Listener $port) { return }
        Start-Sleep -Milliseconds 250
    }
    throw "port $port ไม่ขึ้นภายใน $seconds วินาที"
}

function Test-PgRunning([string]$bin) {
    if (-not (Test-Path (Join-Path $pgData 'PG_VERSION'))) { return $false }
    & (Join-Path $bin 'pg_ctl.exe') status -D $pgData *> $null
    return $LASTEXITCODE -eq 0
}

function Start-Postgres {
    $bin = Get-PgBin
    if (Test-PgRunning $bin) { Write-Host "postgres: running on $pgPort"; return }
    if (Get-Listener $pgPort) { throw "port $pgPort ถูกโปรแกรมอื่นใช้อยู่ — ปิดก่อน หรือแก้ DB_PORT ใน .env.test" }

    if (-not (Test-Path (Join-Path $pgData 'PG_VERSION'))) {
        New-Item -ItemType Directory -Force $state | Out-Null
        $pwFile = Join-Path $state 'pw.tmp'
        [IO.File]::WriteAllText($pwFile, $pgPassword)
        try {
            & (Join-Path $bin 'initdb.exe') -D $pgData -U $pgUser --pwfile=$pwFile --auth=scram-sha-256 -E UTF8 --no-locale | Out-Null
            if ($LASTEXITCODE -ne 0) { throw 'initdb ล้มเหลว' }
        } finally {
            Remove-Item $pwFile -ErrorAction SilentlyContinue
        }
    }

    # ไม่ใช้ pg_ctl -w: process ลูกถือ handle ของ console ไว้ทำให้คำสั่งค้าง — เปิดแบบซ่อนแล้วรอ port แทน
    $opts = "-p $pgPort -c listen_addresses=localhost"
    Start-Process -FilePath (Join-Path $bin 'pg_ctl.exe') -WindowStyle Hidden `
        -ArgumentList @('start', '-D', "`"$pgData`"", '-l', "`"$pgLog`"", '-o', "`"$opts`"")
    Wait-Port $pgPort 30

    $env:PGPASSWORD = $pgPassword
    try {
        $psql = Join-Path $bin 'psql.exe'
        $exists = & $psql -h localhost -p $pgPort -U $pgUser -d postgres -w -tAc "SELECT 1 FROM pg_database WHERE datname = '$pgDb'"
        if ($exists -ne '1') {
            & $psql -h localhost -p $pgPort -U $pgUser -d postgres -w -c "CREATE DATABASE $pgDb" | Out-Null
        }
    } finally {
        Remove-Item Env:PGPASSWORD -ErrorAction SilentlyContinue
    }
    Write-Host "postgres: started on $pgPort (db $pgDb)"
}

function Get-RedisProcess {
    if (-not (Test-Path $redisPidFile)) { return $null }
    $id = [int](Get-Content $redisPidFile)
    return Get-Process -Id $id -ErrorAction SilentlyContinue | Where-Object { $_.ProcessName -eq 'miniredis' }
}

function Start-Redis {
    if (Get-RedisProcess) { Write-Host "redis: miniredis running on $redisPort"; return }
    if (Get-Listener $redisPort) { throw "port $redisPort ถูกโปรแกรมอื่นใช้อยู่ — ปิดก่อน หรือแก้ REDIS_ADDR ใน .env.test" }

    New-Item -ItemType Directory -Force $state | Out-Null
    Push-Location $root
    try {
        go build -o $redisExe ./scripts/miniredis
        if ($LASTEXITCODE -ne 0) { throw 'build miniredis ล้มเหลว' }
    } finally {
        Pop-Location
    }
    $p = Start-Process -FilePath $redisExe -ArgumentList @('-addr', "127.0.0.1:$redisPort") -WindowStyle Hidden -PassThru
    Set-Content -Path $redisPidFile -Value $p.Id
    Wait-Port $redisPort 15
    Write-Host "redis: miniredis started on $redisPort"
}

function Stop-All {
    $bin = Get-PgBin
    if (Test-PgRunning $bin) {
        & (Join-Path $bin 'pg_ctl.exe') stop -D $pgData -m fast | Out-Null
        Write-Host 'postgres: stopped'
    }
    $r = Get-RedisProcess
    if ($r) {
        Stop-Process -Id $r.Id -Force
        Write-Host 'redis: stopped'
    }
    Remove-Item $redisPidFile -ErrorAction SilentlyContinue
}

# Use-TestEnv ตั้ง env ของ process นี้จาก .env.test — server / script ที่รันต่อจะใช้ test env
# (godotenv/autoload ของ server ไม่ทับค่าที่ตั้งไว้แล้ว จึงไม่ไปอ่าน .env ของ dev)
function Use-TestEnv {
    foreach ($k in $cfg.Keys) { Set-Item -Path "Env:$k" -Value $cfg[$k] }
}

function Invoke-InRoot([string[]]$goArgs) {
    Use-TestEnv
    Push-Location $root
    try {
        & go @goArgs
        if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }
    } finally {
        Pop-Location
    }
}

switch ($Command) {
    'serve' { Invoke-InRoot (@('run', '.') + $Rest) }
    'seed' { Invoke-InRoot (@('run', './scripts/seed_loadtest') + $Rest) }
    'up' { Start-Postgres; Start-Redis }
    'down' { Stop-All }
    'reset' {
        Stop-All
        if (Test-Path $state) { Remove-Item $state -Recurse -Force }
        Write-Host 'testenv: data removed'
    }
    'status' {
        $bin = Get-PgBin
        $pg = if (Test-PgRunning $bin) { "running on $pgPort" } else { 'stopped' }
        $rd = if (Get-RedisProcess) { "running on $redisPort" } else { 'stopped' }
        Write-Host "postgres: $pg"
        Write-Host "redis: $rd"
    }
}
