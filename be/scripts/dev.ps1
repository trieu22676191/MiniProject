param(
    [Parameter(Mandatory = $true)]
    [ValidateSet('up', 'migrate', 'order', 'stock', 'bff', 'test', 'down')]
    [string]$Action
)

$ErrorActionPreference = 'Stop'
$beRoot = (Resolve-Path -LiteralPath (Join-Path $PSScriptRoot '..')).Path
$envFile = Join-Path $beRoot 'deploy/.env'
$composeFile = Join-Path $beRoot 'deploy/compose.yaml'
$env:PATH = (Join-Path (go env GOPATH) 'bin') + [IO.Path]::PathSeparator + $env:PATH

function Read-Config {
    if (-not (Test-Path -LiteralPath $envFile)) {
        throw "Missing $envFile. Copy deploy/.env.example and set passwords."
    }
    $config = @{}
    Get-Content -LiteralPath $envFile | ForEach-Object {
        if ($_ -match '^([^#=]+)=(.*)$') { $config[$Matches[1]] = $Matches[2] }
    }
    foreach ($key in @('POSTGRES_PASSWORD', 'MINI_OWNER_PASSWORD', 'MINI_APP_PASSWORD', 'MINI_RELAY_PASSWORD')) {
        if (-not $config[$key]) { throw "Missing $key in deploy/.env" }
    }
    if (-not $config['POSTGRES_HOST_PORT']) { $config['POSTGRES_HOST_PORT'] = '15432' }
    return $config
}

function Database-URL([string]$user, [string]$password, [string]$database, [string]$port) {
    $encoded = [Uri]::EscapeDataString($password)
    return 'postgres://{0}:{1}@127.0.0.1:{2}/{3}?sslmode=disable' -f $user, $encoded, $port, $database
}

function Migrate-Database([string]$database, [hashtable]$config) {
    $url = Database-URL 'mini_owner' $config['MINI_OWNER_PASSWORD'] $database $config['POSTGRES_HOST_PORT']
    & goose -dir (Join-Path $beRoot 'services/order/migrations') -table mini_order.goose_db_version postgres $url up
    if ($LASTEXITCODE -ne 0) { throw 'Order migration failed' }
    & goose -dir (Join-Path $beRoot 'services/stock/migrations') -table mini_stock.goose_db_version postgres $url up
    if ($LASTEXITCODE -ne 0) { throw 'Stock migration failed' }
}

switch ($Action) {
    'up' {
        Read-Config | Out-Null
        & docker compose --env-file $envFile -f $composeFile up -d
        if ($LASTEXITCODE -ne 0) { throw 'Docker Compose failed' }
    }
    'migrate' {
        $config = Read-Config
        Migrate-Database 'mini' $config
    }
    'stock' {
        $config = Read-Config
        $env:STOCK_DATABASE_URL = Database-URL 'mini_app' $config['MINI_APP_PASSWORD'] 'mini' $config['POSTGRES_HOST_PORT']
        $env:OUTBOX_DATABASE_URL = Database-URL 'mini_relay' $config['MINI_RELAY_PASSWORD'] 'mini' $config['POSTGRES_HOST_PORT']
        Push-Location $beRoot
        try { & go run ./services/stock/cmd/server } finally { Pop-Location; Remove-Item Env:STOCK_DATABASE_URL,Env:OUTBOX_DATABASE_URL -ErrorAction SilentlyContinue }
    }
    'order' {
        $config = Read-Config
        $env:ORDER_DATABASE_URL = Database-URL 'mini_app' $config['MINI_APP_PASSWORD'] 'mini' $config['POSTGRES_HOST_PORT']
        $env:OUTBOX_DATABASE_URL = Database-URL 'mini_relay' $config['MINI_RELAY_PASSWORD'] 'mini' $config['POSTGRES_HOST_PORT']
        Push-Location $beRoot
        try { & go run ./services/order/cmd/server } finally { Pop-Location; Remove-Item Env:ORDER_DATABASE_URL,Env:OUTBOX_DATABASE_URL -ErrorAction SilentlyContinue }
    }
    'bff' {
        Push-Location $beRoot
        try { & go run ./services/bff/cmd/server } finally { Pop-Location }
    }
    'test' {
        $config = Read-Config
        $testDB = 'mini_test_' + [guid]::NewGuid().ToString('N').Substring(0, 12)
        $env:PGPASSWORD = $config['POSTGRES_PASSWORD']
        $created = $false
        try {
            & docker exec -e PGPASSWORD deploy-postgres-1 psql -h 127.0.0.1 -U postgres -d postgres -v ON_ERROR_STOP=1 -c "CREATE DATABASE $testDB"
            if ($LASTEXITCODE -ne 0) { throw 'Create test database failed' }
            $created = $true
            $setup = @'
CREATE SCHEMA mini_order AUTHORIZATION mini_owner;
CREATE SCHEMA mini_stock AUTHORIZATION mini_owner;
GRANT USAGE ON SCHEMA mini_order, mini_stock TO mini_app, mini_relay;
ALTER DEFAULT PRIVILEGES FOR ROLE mini_owner IN SCHEMA mini_order, mini_stock
  GRANT SELECT, INSERT, UPDATE, DELETE ON TABLES TO mini_app;
ALTER DEFAULT PRIVILEGES FOR ROLE mini_owner IN SCHEMA mini_order, mini_stock
  GRANT USAGE, SELECT ON SEQUENCES TO mini_app;
'@
            & docker exec -e PGPASSWORD deploy-postgres-1 psql -h 127.0.0.1 -U postgres -d $testDB -v ON_ERROR_STOP=1 -c $setup
            if ($LASTEXITCODE -ne 0) { throw 'Configure test database failed' }
            Migrate-Database $testDB $config
            $env:MINI_TEST_OWNER_DATABASE_URL = Database-URL 'mini_owner' $config['MINI_OWNER_PASSWORD'] $testDB $config['POSTGRES_HOST_PORT']
            $env:MINI_TEST_APP_DATABASE_URL = Database-URL 'mini_app' $config['MINI_APP_PASSWORD'] $testDB $config['POSTGRES_HOST_PORT']
            Push-Location $beRoot
            try {
                & go test -tags integration ./...
                if ($LASTEXITCODE -ne 0) { throw 'Integration tests failed' }
            } finally { Pop-Location }
        } finally {
            Remove-Item Env:MINI_TEST_OWNER_DATABASE_URL -ErrorAction SilentlyContinue
            Remove-Item Env:MINI_TEST_APP_DATABASE_URL -ErrorAction SilentlyContinue
            if ($created) {
                & docker exec -e PGPASSWORD deploy-postgres-1 psql -h 127.0.0.1 -U postgres -d postgres -v ON_ERROR_STOP=1 -c "DROP DATABASE $testDB WITH (FORCE)"
            }
            Remove-Item Env:PGPASSWORD -ErrorAction SilentlyContinue
        }
    }
    'down' {
        & docker compose --env-file $envFile -f $composeFile down
        if ($LASTEXITCODE -ne 0) { throw 'Docker Compose down failed' }
    }
}
