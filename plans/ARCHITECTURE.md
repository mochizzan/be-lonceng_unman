# Architecture Documentation

This document describes the architecture of the BE Lonceng Unman microservice - a KRS (Kartu Rencana Studi) retrieval system for Universitas Mandiri.

## Overview

The application is a Go microservice that retrieves student KRS data by downloading and parsing PDF documents from an external e-learning platform. It follows Clean Architecture principles with clear separation of concerns.

## Architecture Layers

### 1. Entry Point (`cmd/api/`)

- **`main.go`**: Application entry point. Responsible for:
  - Loading configuration from environment variables via `config.LoadConfig()`
  - Initializing all services with dependency injection
  - Setting up graceful shutdown using `signal.NotifyContext`
  - Creating and wiring all components together

### 2. Configuration (`internal/config/`)

- **`env.go`**: Centralized configuration management
  - `Config` struct holds all application settings
  - `LoadConfig()` reads from environment variables and `.env` file
  - Helper functions (`GetRunningPort`, `GetJWTSecret`, `GetTrustProxy`, etc.) provide type-safe access
  - `GetAllowedPDFHosts()` validates and extracts hostnames from URLs
  - All config is loaded once at startup - no `os.Getenv` in business logic

### 3. HTTP Handler (`internal/handler/`)

- **`krs_handler.go`**: Handles KRS retrieval requests
  - `KRSHandler` struct with dependency injection via `PDFProcessor` interface
  - `GetKRSByNIS()` - main endpoint handler:
    1. Extracts NPM from JWT context
    2. Validates NPM format
    3. Checks cache for existing data
    4. Downloads and processes PDF on cache miss
    5. Stores result in cache and file storage
  - `respondJSON()` / `respondError()` - standardized response formatting
  - All responses wrapped in `SuccessResponse` for consistent format

- **`cek_response.go`**: Health check endpoint
  - Returns service status, version, and timestamp

### 4. Middleware (`internal/middleware/`)

- **`jwt_auth.go`**: JWT authentication middleware
  - Validates Bearer tokens using HS256
  - Extracts NPM from JWT claims into request context
  - Returns standardized error responses

- **`api_key.go`**: API key authentication middleware
  - Validates API key from header
  - Uses constructor injection for config

- **`rate_limit.go`**: Rate limiting middleware
  - Uses `golang.org/x/time/rate` for token bucket algorithm
  - Supports trust-proxy configuration for load balancer scenarios
  - Returns standardized error responses

- **`request_id.go`**: Request correlation middleware
  - Generates unique `X-Request-Id` for each request
  - Propagates through context for logging

- **`jwt_context_key.go`**: Type-safe context keys
  - Prevents context key collisions
  - `SetUserNPM()` / `GetUserNPM()` helpers

### 5. Services (`internal/services/`)

- **`pdf_service.go`**: PDF processing service
  - `NewPDFService(client *http.Client)` - accepts custom HTTP client for testability
  - `DownloadPDF()` - downloads PDF with:
    - URL validation (HTTPS only)
    - Domain whitelist enforcement
    - Content-Type validation
    - Size validation (>100 bytes)
    - PDF magic bytes verification
  - `ExtractTextFromPDF()` - text extraction using `ledongthuc/pdf`
  - `ProcessKRSFromURL()` - orchestrates download → extract → parse
  - `ValidateNIS()` - NIS format validation (10 digits)

- **`cache_service.go`**: In-memory caching service
  - TTL-based cache with automatic cleanup goroutine
  - Accepts `context.Context` for graceful shutdown
  - Thread-safe with `sync.RWMutex`

### 6. Model (`internal/model/`)

- **`errors.go`**: Typed error definitions
  - `NISEmptyError`, `NISInvalidError`, `PDFDownloadFailedError`, `InvalidResponseError`
  - Each error type maps to specific HTTP status codes via `MapErrorToHTTPStatus()`

- **`error_response.go`**: Error response structure
  - `ErrorResponse` with status, code, message, details, timestamp

- **`response_wrapper.go`**: Standardized response format
  - `SuccessResponse` wrapping data with metadata
  - `ResponseMeta` with timestamp, version, service name, cache status

- **`krs_model.go`**: KRS data structures
  - `KRSResponse`, `Mahasiswa`, `MataKuliah`

- **`error_codes.go`**: Error code constants

### 7. Parser (`internal/parser/`)

- **`krs_parser.go`**: Parses extracted text into structured KRS data
  - `KRSParser` with `ExtractMahasiswa()`, `ExtractAcademicInfo()`, `ExtractMataKuliah()`

### 8. Storage (`internal/storage/`)

- **`file_storage.go`**: File-based persistence for KRS responses
  - JSON serialization with organized directory structure

## Request Flow

```
Client Request
    │
    ▼
┌─────────────────────────────────┐
│  RequestIDMiddleware            │  ← Generates X-Request-Id
├─────────────────────────────────┤
│  RateLimitMiddleware            │  ← Token bucket rate limiting
├─────────────────────────────────┤
│  APIKeyMiddleware               │  ← API key validation
├─────────────────────────────────┤
│  JWTAuthMiddleware              │  ← JWT validation + NPM extraction
├─────────────────────────────────┤
│  KRSHandler.GetKRSByNIS()       │
│  ├── Validate NPM from JWT      │
│  ├── Check cache                │
│  ├── Download PDF (PDFService)  │
│  │   ├── Validate URL/HTTPS     │
│  │   ├── Check domain whitelist │
│  │   ├── Download with timeout  │
│  │   └── Validate response      │
│  ├── Extract text (PDF parser)  │
│  ├── Parse KRS data             │
│  ├── Store in cache             │
│  └── Store in file storage      │
├─────────────────────────────────┤
│  Response (SuccessResponse)     │  ← Standardized JSON format
└─────────────────────────────────┘
```

## Key Design Decisions

### Dependency Injection
- All services receive dependencies through constructors
- `PDFService` accepts `*http.Client` for mock testing
- `KRSHandler` depends on `PDFProcessor` interface for testability
- `CacheService` accepts `context.Context` for lifecycle management

### Error Handling
- Custom error types in `internal/model/errors.go`
- `MapErrorToHTTPStatus()` provides consistent HTTP status mapping
- All errors wrapped in `SuccessResponse` for consistent client experience

### Configuration
- All configuration loaded at startup from environment
- No `os.Getenv` calls in business logic
- `Config` struct provides type-safe access to all settings

### Observability
- `X-Request-Id` propagated through all log entries
- Structured logging with `slog` (JSON format)
- Request duration tracking in handler

### Graceful Shutdown
- Root context with `signal.NotifyContext` for SIGTERM/SIGINT
- Cache cleanup goroutine respects context cancellation
- HTTP server shutdown with 30-second timeout

## Testing Strategy

- **Unit tests**: Use mock HTTP transports (`httptest.NewServer`, custom `RoundTripper`)
- **Integration tests**: Marked with `testing.Short()` skip, require real external service
- **Handler tests**: Use `PDFProcessor` interface for mock injection
- **All tests**: `-short` flag for CI/CD compatibility