Write-Host "Starting test environment..." -ForegroundColor Green
docker-compose -f docker-compose.test.yml down -v
docker-compose -f docker-compose.test.yml up --build --abort-on-container-exit

Write-Host "Cleaning up test environment..." -ForegroundColor Green
docker-compose -f docker-compose.test.yml down -v 