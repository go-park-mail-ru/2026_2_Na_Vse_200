# Загружает обычные KEY=VALUE из .env без выполнения его как кода.
# .\scripts\run.ps1                      — сервер
# .\scripts\run.ps1 go run ./cmd/migrate up — SQL-миграции
param(
    [Parameter(ValueFromRemainingArguments = $true)]
    [string[]]$CommandArgs
)

$ErrorActionPreference = 'Stop'
$projectRoot = Split-Path $PSScriptRoot -Parent
$envFile = if ($env:ENV_FILE) { $env:ENV_FILE } else { '.env' }
if (-not [System.IO.Path]::IsPathRooted($envFile)) {
    $envFile = Join-Path $projectRoot $envFile
}

Push-Location -LiteralPath $projectRoot
try {
    if (Test-Path -LiteralPath $envFile) {
        $lineNumber = 0
        foreach ($line in Get-Content -LiteralPath $envFile -Encoding UTF8) {
            $lineNumber++
            if ($line -match '^\s*(#.*)?$') { continue }
            if ($line -notmatch '^\s*(?:export\s+)?([A-Za-z_][A-Za-z0-9_]*)\s*=\s*(.*?)\s*$') {
                throw "Invalid .env assignment on line $lineNumber"
            }
            $key = $matches[1]
            $value = $matches[2]
            if ($value.StartsWith('"')) {
                if ($value -notmatch '^"([^"\r\n]*)"\s*(?:#.*)?$') {
                    throw "Invalid quoted .env value on line $lineNumber"
                }
                $value = $matches[1]
            } elseif ($value.StartsWith("'")) {
                if ($value -notmatch "^'([^'\r\n]*)'\s*(?:#.*)?$") {
                    throw "Invalid quoted .env value on line $lineNumber"
                }
                $value = $matches[1]
            } else {
                $value = ($value -replace '\s+#.*$', '').TrimEnd()
            }
            [Environment]::SetEnvironmentVariable($key, $value, 'Process')
        }
    }
    if (-not $CommandArgs -or $CommandArgs.Count -eq 0) {
        $CommandArgs = @('go', 'run', './cmd/server')
    }
    $program = $CommandArgs[0]
    [string[]]$programArgs = @()
    if ($CommandArgs.Count -gt 1) {
        $programArgs = $CommandArgs[1..($CommandArgs.Count - 1)]
    }
    & $program @programArgs
    exit $LASTEXITCODE
} finally {
    Pop-Location
}