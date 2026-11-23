# Delcom Blog — Backend REST API (Golang Gin)

Backend REST API untuk aplikasi **Delcom Blog** dengan arsitektur **Clean Architecture & Feature-First**, autentikasi JWT multi-tier (`guest`, `member`, `admin`), kalkulasi *reading time*, generator slug unik, AI Content Generator terintegrasi Search Engine, Swagger UI interaktif, dan 100% statement test coverage.

## Menjalankan Backend
```bash
psql -U postgres -c "CREATE DATABASE pabwe_p9_blog;"
go run cmd/server/main.go
```
- REST API: `http://localhost:8081`
- Swagger UI: `http://localhost:8081/swagger/index.html`

## Pengujian Unit (100% Coverage)
```bash
go test -v -cover ./internal/...
```

## Build Binary
```bash
go build -o server.exe cmd/server/main.go
.\server.exe
```
