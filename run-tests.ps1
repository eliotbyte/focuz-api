# Read environment variables from .env file
$envContent = Get-Content .env | Where-Object { $_ -match '^[^#]' -and $_ -match '=' }
$envVars = @{}
foreach ($line in $envContent) {
    $parts = $line -split '=', 2
    if ($parts.Length -eq 2) {
        $envVars[$parts[0]] = $parts[1]
    }
}

# Build test image
Write-Host "Building test image..." -ForegroundColor Green
docker build --target test -t focuz-test .

# Construct DATABASE_URL from environment variables
$databaseUrl = "postgres://$($envVars['POSTGRES_USER']):$($envVars['POSTGRES_PASSWORD'])@db:5432/$($envVars['POSTGRES_DB'])?sslmode=disable"

Write-Host "Running tests..." -ForegroundColor Green
docker run --network focuz-api_focuz-network -e DATABASE_URL="$databaseUrl" focuz-test 