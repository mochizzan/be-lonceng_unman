# Full Audit & Remediation Plan for `be-lonceng_unman`

> Generated: 2026-05-09 | Based on recommendations from code review

---

## Table of Contents
1. [Executive Summary](#1-executive-summary)
2. [Project Structure Audit](#2-project-structure-audit)
3. [Issue #1: Standardise Error Handling](#3-issue-standardise-error-handling)
4. [Issue #2: Fix Test Failures](#4-issue-fix-test-failures)
5. [Issue #3: Remove Dead Code](#5-issue-remove-dead-code)
6. [Issue #4: Cache Static Configuration](#6-issue-cache-static-configuration)
7. [Issue #5: Fix GetAllowedPDFHosts](#7-issue-fix-getallowedpdfhosts)
8. [Issue #6: Add Request-ID Propagation](#8-issue-add-request-id-propagation)
9. [Issue #7: Mock External HTTP Calls in Tests](#9-issue-mock-external-http-calls-in-tests)
10. [Issue #8: Document Architecture Boundaries](#10-issue-document-architecture-boundaries)
11. [Issue #9: Review Rate-Limit Trust-Proxy Logic](#11-issue-review-rate-limit-trust-proxy-logic)
12. [Issue #10: Add Graceful Shutdown](#12-issue-add-graceful-shutdown)
13. [Implementation Priority Order](#14-implementation-priority-order)

---

## 1. Executive Summary

This plan addresses 10 recommendation items from the code review, broken down into **30+ specific, file-level changes** with exact line references. Each item includes the current problem, the target state, and acceptance criteria.

---

## 2. Project Structure Audit

### Current File Inventory

| File | Lines | Role | Issues Found |
|------|-------|------|-------------|
| `cmd/api/main.go` | 99 | Bootstrap | Hardcoded env reads, no context cancellation propagation, trustProxy hardcoded |
| `internal/handler/krs_handler.go` | 212 | HTTP handler | Fat handler, raw error strings, no request ID |
| `internal/handler/krs_handler_test.go` | 347 | Handler tests | Wrong context key, integration tests not mocked |
| `internal/handler/cek_response.go` | 34 | Health check | Inconsistent response format (uses `map[string]interface{}`) |
| `internal/middleware/jwt_auth.go` | 122 | JWT auth | `respondUnauthorized` uses raw map, not `ErrorResponse` |
| `internal/middleware/jwt_context_key.go` | 14 | Context key | OK — already uses typed key |
| `internal/middleware/api_key.go` | 46 | API key | `os.Getenv` called on every request (lines 28, 34) |
| `internal/middleware/rate_limit.go` | 130 | Rate limiting | Raw JSON error response, no graceful stop, trustProxy concerns |
| `internal/middleware/set_middleware.go` | 12 | Middleware chain | OK |
| `internal/config/env.go` | 192 | Config loader | `GetAllowedPDFHosts` returns URL not hostname (line 182) |
| `internal/config/output.go` | 1 | (empty) | Dead file — delete |
| `internal/services/pdf_service.go` | 214 | PDF processing | `URLValidator` dead code (lines 37-72), plain errors not typed |
| `internal/services/pdf_service_test.go` | 366 | PDF tests | All integration — no mocking |
| `internal/services/krs_service_test.go` | 173 | Parser/cache tests | OK |
| `internal/services/pdf_integration_test.go` | 82 | Integration test | Relies on external service |
| `internal/services/pdf_debug_test.go` | 62 | Debug test | Relies on external service |
| `internal/services/cache_service.go` | 117 | In-memory cache | Cleanup goroutine has no stop signal |
| `internal/parser/krs_parser.go` | 239 | KRS parser | OK — already separated from model |
| `internal/model/errors.go` | 220 | Custom errors + mapping | `MapErrorToHTTPStatus` correct but unused by service layer |
| `internal/model/error_response.go` | 23 | Error DTO | OK |
| `internal/model/error_codes.go` | 19 | Error constants | OK |
| `internal/model/krs_model.go` | 32 | KRS entities | OK |
| `internal/model/response_wrapper.go` | 25 | Response DTO | OK |
| `internal/model/cek_response.go` | 10 | Health DTO | Inconsistent with ErrorResponse |
| `internal/storage/file_storage.go` | 246 | File storage | OK |
| `internal/routes/routes.go` | 25 | Route wiring | OK |

---

## 3. Issue #1: Standardise Error Handling

### 3.1 Problem
Middleware error responses use inconsistent formats:
- `jwt_auth.go:114-121` → `map[string]string` (raw map)
- `api_key.go:15-40` → `model.CekResponse` (wrong type)
- `rate_limit.go:99-100` → raw JSON string literal
- `krs_handler.go:194-211` → `model.ErrorResponse` (correct, but manually constructed)

Service layer returns plain `errors.New()` / `fmt.Errorf()` instead of custom error types, so `MapErrorToHTTPStatus()` falls through to default `500` for most errors.

### 3.2 Changes Required

#### 3.2.1 `internal/middleware/jwt_auth.go` (lines 113-121)
- **Current:** `respondUnauthorized` writes `map[string]string`
- **Target:** Return `model.ErrorResponse` with proper `Status`, `Code`, `Message`, `Details`, `Timestamp`
- **New helper:** Create a `respondError(w, status, code, message)` helper in this file (or shared)

#### 3.2.2 `internal/middleware/api_key.go` (lines 1-46)
- **Current:** Uses `model.CekResponse` with `os.Getenv` on every request
- **Target:** 
  - Accept API key as constructor parameter (dependency injection)
  - Return `model.ErrorResponse` format on failure
  - Remove `os.Getenv` calls from request path

#### 3.2.3 `internal/middleware/rate_limit.go` (lines 97-101)
- **Current:** Writes raw JSON string `{"status":"error","message":"rate limit exceeded"}`
- **Target:** Use `model.ErrorResponse` with proper JSON encoding via `json.NewEncoder`

#### 3.2.4 `internal/services/pdf_service.go` (lines 74-213)
- **Current:** Returns `errors.New()` and `fmt.Errorf()` — no typed errors
- **Target:** Wrap errors with custom types from `model/errors.go`:
  - Line 81 (HTTPS check) → `NewInvalidResponseError("only HTTPS URLs are allowed")`
  - Line 86 (hostname check) → `NewInvalidResponseError("URL must have a hostname")`
  - Line 98 (host not allowed) → `NewInvalidResponseError("URL host not allowed: ...")`
  - Line 108 (request error) → `NewPDFDownloadFailedError(url, err, 0)`
  - Line 113 (status code) → `NewPDFDownloadFailedError(url, nil, resp.StatusCode)`
  - Line 119 (content type) → `NewInvalidResponseError("invalid content type: not a PDF")`
  - Line 129 (too small) → `NewInvalidResponseError("PDF file too small")`
  - Line 134 (magic bytes) → `NewInvalidResponseError("invalid PDF file: missing PDF header")`
  - Line 148 (reader error) → `NewPDFParseFailedError(...)`
  - Line 170 (no text) → `NewInvalidResponseError("no text extracted from PDF")`

#### 3.2.5 `internal/handler/krs_handler.go` (lines 76-86)
- **Current:** Error mapping works but underlying errors aren't typed
- **Target:** Once service layer returns typed errors, `mapErrorToHTTPStatus` at line 158-161 will correctly match via `errors.Is()`

### 3.3 Acceptance Criteria
- All HTTP error responses use `model.ErrorResponse` JSON format
- `MapErrorToHTTPStatus` returns correct codes for all custom error types
- No raw `map[string]string` or string-literal JSON in error responses
- Unit test covers each error type → status code mapping

---

## 4. Issue #2: Fix Test Failures

### 4.1 Problem A: Wrong Context Key in Handler Tests
**File:** `internal/handler/krs_handler_test.go`
- **Line 211:** `context.WithValue(context.Background(), "user_npm", "123")` — uses raw string `"user_npm"`
- **Line 238:** `context.WithValue(context.Background(), "user_npm", "")` — same
- **Line 263:** `context.WithValue(context.Background(), "user_npm", "2211700006")` — same

The actual handler at `krs_handler.go:42` calls `middleware.GetUserNPM(ctx)` which uses the unexported typed key `userNPMKey` from `jwt_context_key.go:7`. Raw string values won't match.

### 4.1.1 Fix
- **File:** `internal/middleware/jwt_context_key.go`
  - Export a helper: `func SetUserNPM(ctx context.Context, npm string) context.Context`
  - Add at line 13:
    ```go
    func SetUserNPM(ctx context.Context, npm string) context.Context {
        return context.WithValue(ctx, userNPMKey, npm)
    }
    ```
- **File:** `internal/handler/krs_handler_test.go`
  - Line 211: Change `"user_npm"` to use `middleware.SetUserNPM(context.Background(), "123")`
  - Line 238: Change `"user_npm"` to use `middleware.SetUserNPM(context.Background(), "")`
  - Line 263: Change `"user_npm"` to use `middleware.SetUserNPM(context.Background(), "2211700006")`

### 4.2 Problem B: ValidateNIS Not Using Custom Errors
**File:** `internal/services/pdf_service.go` lines 201-213
- **Current:** Returns `errors.New("NIS cannot be empty")` and `errors.New("NIS must be 10 digits")`
- **Impact:** `MapErrorToHTTPStatus` checks for `NISEmptyError{}` and `NISInvalidError{}` types — these will never match

### 4.2.1 Fix
- **File:** `internal/services/pdf_service.go`
  - Line 203: `return errors.New("NIS cannot be empty")` → `return model.NewNISEmptyError()`
  - Line 209: `return errors.New("NIS must be 10 digits")` → `return model.NewNISInvalidError(nis)`
- **File:** `internal/handler/krs_handler.go`
  - Line 50-53: Remove the separate NIS empty check (lines 43-47 already handle it via JWT). Let `ValidateNIS` return `NISEmptyError` which maps to `400` via `MapErrorToHTTPStatus`. OR keep the 401 for empty NIS from JWT — clarify intent:
    - If NIS is empty string from JWT → 401 Unauthorized (current behavior, acceptable)
    - If NIS is malformed (not 10 digits) → 400 Bad Request via `NISInvalidError`

### 4.3 Acceptance Criteria
- All handler tests pass with `go test ./internal/handler/...`
- `ValidateNIS` returns typed errors that `MapErrorToHTTPStatus` can match
- Test covers: empty NIS → correct status code, invalid format → 400

---

## 5. Issue #3: Remove Dead Code

### 5.1 `URLValidator` — Dead Struct in `internal/services/pdf_service.go` (lines 37-72)
- **Lines 37-43:** `URLValidator` struct definition — never instantiated anywhere
- **Lines 45-72:** `URLValidator.Validate()` method — never called
- **Action:** Delete lines 37-72 entirely
- **Note:** The URL validation logic in `DownloadPDF()` (lines 80-99) already performs the same checks inline — this is the correct location

### 5.2 `internal/config/output.go` — Empty File
- **Current:** Only contains `package config` (1 line)
- **Action:** Delete the file entirely
- **Note:** If `go build` requires the file to exist for the package, keep it but add a comment. But since `env.go` is in the same `config` package and already defines everything, deletion is safe.

### 5.3 Acceptance Criteria
- `go build ./...` succeeds without the deleted code
- No unused import or variable warnings
- `go vet ./...` passes cleanly

---

## 6. Issue #4: Cache Static Configuration

### 6.1 Problem
- `internal/middleware/api_key.go:28` — `os.Getenv("API_KEY")` called on every request
- `internal/middleware/api_key.go:19,34` — `os.Getenv("SERVER_VERSION")` called on every request
- `internal/handler/krs_handler.go:177` — `config.GetAppVersion()` reads from global `cfg` (OK, but cfg could be cached better)
- `internal/handler/krs_handler.go:178` — `config.GetServiceName()` same

### 6.2 Changes Required

#### 6.2.1 `internal/middleware/api_key.go` — Full Refactor
- **Current:** Reads `API_KEY` and `SERVER_VERSION` from `os.Getenv` per-request
- **Target:** Accept config values via constructor
  ```go
  type APIKeyConfig struct {
      APIKeyValue   string
      ServerVersion string
  }
  
  func NewAPIKeyMiddleware(cfg APIKeyConfig, next http.Handler) http.Handler {
      // Use cfg.APIKeyValue and cfg.ServerVersion
  }
  ```
- **Update `routes.go`** to pass config when wiring middleware
- **Update `main.go`** to read env once at startup and pass to constructor

#### 6.2.2 `internal/config/env.go` — Cache All Values
- Already caches in `cfg` struct — the `Get*()` functions return from `cfg`
- **Issue:** `GetAllowedPDFHosts()` at line 182 calls `os.Getenv("PDF_ALLOWED_HOSTS")` directly instead of using a cached field
- **Fix:** Add `AllowedPDFHosts` field to `Config` struct, populate in `LoadConfig()`, and have `GetAllowedPDFHosts()` return `cfg.AllowedPDFHosts`

#### 6.2.3 `cmd/api/main.go` — Centralize Env Reads
- Line 35: `os.Getenv("RUNNING_PORT")` → use `config.GetRunningPort()` (add this function)
- Line 48: `os.Getenv("JWT_SECRET")` → use `config.GetJWTSecret()` (add this function)

### 6.3 Acceptance Criteria
- Zero `os.Getenv` calls in handler, middleware, or service code
- All config values read once at startup via `config.LoadConfig()`
- Tests can inject mock config values

---

## 7. Issue #5: Fix GetAllowedPDFHosts

### 7.1 Problem
**File:** `internal/config/env.go` line 182
```go
return []string{cfg.PDFBaseURL} // We'll extract host from PDFBaseURL? For simplicity, use the same default.
```
- `cfg.PDFBaseURL` is `"https://elearning.universitasmandiri.ac.id/admin/cetak/krs_pdf.php"` (a full URL)
- This is returned as an "allowed host" but `DownloadPDF()` at `pdf_service.go:92` compares it with `u.Hostname()` which returns just `"elearning.universitasmandiri.ac.id"`
- **Result:** Host comparison will NEVER match when using the default config value

### 7.2 Fix
- **File:** `internal/config/env.go` line 182
  - Extract hostname from `cfg.PDFBaseURL` using `net/url.Parse()`:
    ```go
    u, err := url.Parse(cfg.PDFBaseURL)
    if err == nil && u.Hostname() != "" {
        return []string{u.Hostname()}
    }
    return []string{"elearning.universitasmandiri.ac.id"}
    ```
- **File:** `internal/config/env.go` line 176-191
  - Also parse `PDF_ALLOWED_HOSTS` env var values — they should be hostnames only, but add validation to strip URLs if someone provides a full URL

### 7.3 Acceptance Criteria
- `GetAllowedPDFHosts()` returns `["elearning.universitasmandiri.ac.id"]` (hostname only)
- Domain whitelist matching in `DownloadPDF()` correctly validates
- Test covers: default config → hostname extracted, custom env → hostnames returned

---

## 8. Issue #6: Add Request-ID Propagation in Logs

### 8.1 Problem
- No request correlation ID exists
- Logs cannot be traced across middleware → handler → service layers for a single request
- `production_readiness_remediation_plan.md` line 489: "ada request identifier di log jika nanti ditambahkan"

### 8.2 Changes Required

#### 8.2.1 New file: `internal/middleware/request_id.go`
```go
package middleware

import (
    "context"
    "crypto/rand"
    "encoding/hex"
    "net/http"
)

type contextKeyRequestID struct{}

func generateRequestID() string {
    b := make([]byte, 16)
    rand.Read(b)
    return hex.EncodeToString(b)
}

func RequestIDMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        requestID := r.Header.Get("X-Request-Id")
        if requestID == "" {
            requestID = generateRequestID()
        }
        ctx := context.WithValue(r.Context(), contextKeyRequestID{}, requestID)
        w.Header().Set("X-Request-Id", requestID)
        next.ServeHTTP(w, r.WithContext(ctx))
    })
}

func GetRequestID(ctx context.Context) string {
    if id, ok := ctx.Value(contextKeyRequestID{}).(string); ok {
        return id
    }
    return ""
}
```

#### 8.2.2 `internal/middleware/jwt_auth.go`
- Add `requestID` to all log calls: `slog.String("request_id", GetRequestID(ctx))`

#### 8.2.3 `internal/handler/krs_handler.go`
- Add `requestID` to all `h.log.*Context(ctx, ...)` calls
- Add request ID to both success and error response metadata (optional, or just in headers)

#### 8.2.4 `internal/middleware/rate_limit.go`
- Add request ID to rate-limit-exceeded log

#### 8.2.5 `cmd/api/main.go` and `internal/routes/routes.go`
- Add `RequestIDMiddleware` to the middleware chain, applied before auth

### 8.3 Acceptance Criteria
- Every log line for a request includes `"request_id": "..."`
- Response includes `X-Request-Id` header
- Request ID propagates through all middleware and handler layers

---

## 9. Issue #7: Mock External HTTP Calls in Tests

### 9.1 Problem
**Files affected:**
- `internal/services/pdf_service_test.go` — lines 56-112: 3 integration tests hitting real external URLs
- `internal/services/pdf_integration_test.go` — entire file is integration test
- `internal/services/pdf_debug_test.go` — entire file hits external service
- `internal/handler/krs_handler_test.go` — lines 255-282: `TestGetKRSByNIS_Integration_ValidNIS`

### 9.2 Changes Required

#### 9.2.1 Make `PDFService` accept custom HTTP client
**File:** `internal/services/pdf_service.go`
- Modify `PDFService` struct to accept `*http.Client` in constructor:
  ```go
  func NewPDFService(client *http.Client) *PDFService {
      if client == nil {
          client = &http.Client{Timeout: 30 * time.Second}
      }
      return &PDFService{client: client}
  }
  ```
- Update `NewPDFService()` (no-arg version) to call `NewPDFService(nil)` for backward compatibility

#### 9.2.2 Create mock HTTP transport
**File:** `internal/services/pdf_service_test.go` (or new file `internal/services/mock_test.go`)
```go
type mockTransport struct {
    response *http.Response
    err      error
}

func (m *mockTransport) RoundTrip(req *http.Request) (*http.Response, error) {
    if m.err != nil {
        return nil, m.err
    }
    return m.response, nil
}

func newMockPDFService(response *http.Response, err error) *services.PDFService {
    client := &http.Client{Transport: &mockTransport{response: response, err: err}}
    return services.NewPDFService(client)
}
```

#### 9.2.3 Convert integration tests to unit tests
**File:** `internal/services/pdf_service_test.go`
- `TestDownloadPDF_Integration_ValidPDF` → Use mock with valid PDF bytes
- `TestDownloadPDF_Integration_Non200Status` → Use mock returning 404
- `TestDownloadPDF_Integration_ContextTimeout` → Use mock with slow handler + short timeout
- `TestExtractTextFromPDF_Integration_RealPDF` → Provide test PDF bytes directly (skip download)
- `TestProcessKRSFromURL_Integration_Success` → Use mock

**File:** `internal/handler/krs_handler_test.go`
- `TestGetKRSByNIS_Integration_ValidNIS` → Mock PDFService to return predefined KRSResponse

#### 9.2.4 Integration tests: rename and tag
- Keep integration tests but rename to `TestDownloadPDF_Integration_*` pattern with `//go:build integration` build tag
- Or move to separate `integration_test.go` file

### 9.3 Acceptance Criteria
- `go test ./... -short` passes without network access
- Integration tests are clearly separated and skipped by default
- All existing test scenarios are covered by mocked unit tests

---

## 10. Issue #8: Document Architecture Boundaries

### 10.1 Problem
- No README or architecture diagram explaining clean-architecture boundaries
- New contributors won't understand where to put code
- `production_readiness_remediation_plan.md` section 5 describes target architecture but it's not enforced

### 10.2 Deliverable
**File:** `ARCHITECTURE.md` (root directory)

Content should include:
1. **Folder structure diagram** (Mermaid)
2. **Layer responsibilities:**
   - `cmd/api/` — Bootstrap only: config, logger, server start
   - `internal/config/` — Env loading, typed config, validation
   - `internal/handler/` — HTTP request/response only, thin orchestration
   - `internal/middleware/` — Auth, rate-limit, request-id, CORS
   - `internal/services/` — Business logic, orchestration, external calls
   - `internal/parser/` — Text/PDF parsing logic
   - `internal/model/` — DTOs, entities, error types, error codes
   - `internal/storage/` — File/object storage abstraction
   - `internal/routes/` — Route wiring only
3. **Dependency rules:** Arrows showing which layers can import which
4. **What goes where:** Decision tree for new developers

### 10.3 Acceptance Criteria
- `ARCHITECTURE.md` exists in repo root
- Contains Mermaid diagram of target architecture
- Contains import dependency rules
- Linked from `README.md` or `CONTRIBUTING.md`

---

## 11. Issue #9: Review Rate-Limit Trust-Proxy Logic

### 11.1 Problem
**File:** `cmd/api/main.go` line 58
```go
rateLimit := middleware.NewRateLimit(rate.Limit(rps), burst, log, true, context.Background())
```
- `trustProxy` is hardcoded to `true`
- If service is NOT behind a reverse proxy, clients can spoof `X-Forwarded-For` headers to bypass rate limiting
- No validation that the header value is from a trusted source

### 11.2 Changes Required

#### 11.2.1 `internal/middleware/rate_limit.go`
- Add IP validation for forwarded headers:
  - Parse IP from `X-Forwarded-For` and validate it's a valid IP format
  - Reject obviously spoofed values (e.g., private IPs from external headers)
  - Add a comment documenting the trust assumption

#### 11.2.2 `internal/config/env.go`
- Add `TRUST_PROXY` env var (default `false` for safety)
- Add `GetTrustProxy()` function

#### 11.2.3 `cmd/api/main.go`
- Line 58: Replace hardcoded `true` with `config.GetTrustProxy()`

#### 11.2.4 `internal/middleware/rate_limit.go` — IP extraction refactor
- Extract the IP extraction logic (duplicated in `getLimiter` and `Handle`) into a single `getClientIP(r *http.Request) string` method
- Add validation:
  ```go
  func isValidPublicIP(ip string) bool {
      parsed := net.ParseIP(ip)
      if parsed == nil {
          return false
      }
      // Reject private/reserved ranges if trustProxy is false
      return !isPrivateIP(parsed)
  }
  ```

### 11.3 Acceptance Criteria
- `trustProxy` is configurable via environment variable
- When `trustProxy=false`, only `r.RemoteAddr` is used
- When `trustProxy=true`, forwarded headers are validated for format
- Documentation in code explains the security implications

---

## 12. Issue #10: Add Graceful Shutdown

### 12.1 Problem
**File:** `cmd/api/main.go`
- Graceful shutdown exists (lines 84-97) but doesn't cancel background goroutines
- `RateLimit.cleanup()` goroutine (rate_limit.go:108-129) runs forever — no stop signal even on shutdown
- `CacheService.cleanup()` goroutine (cache_service.go:102-116) runs forever — no stop signal
- Server context is `context.Background()` — never cancelled

### 12.2 Changes Required

#### 12.2.1 `cmd/api/main.go`
- Create a root context with cancel:
  ```go
  ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
  defer stop()
  ```
- Pass `ctx` to `NewRateLimit` instead of `context.Background()`
- Pass `ctx` to `NewCacheService` (new parameter)
- After `srv.Shutdown(ctx)`, call `stop()` to cancel all goroutines

#### 12.2.2 `internal/services/cache_service.go`
- Add `ctx context.Context` parameter to `NewCacheService(ctx context.Context)`
- Modify `cleanup()` to listen on `ctx.Done()`:
  ```go
  func (c *CacheService) cleanup() {
      ticker := time.NewTicker(1 * time.Hour)
      defer ticker.Stop()
      for {
          select {
          case <-c.ctx.Done():
              return
          case <-ticker.C:
              // existing cleanup logic
          }
      }
  }
  ```

#### 12.2.3 `internal/middleware/rate_limit.go`
- Already has `ctx.Done()` handling at line 114 — just needs the correct context from main

### 12.3 Acceptance Criteria
- All background goroutines stop within the shutdown timeout (30s)
- No goroutine leaks detected via `go test -race`
- Server exits cleanly on SIGTERM/SIGINT

---

## 14. Implementation Priority Order

### Phase 1: Critical Fixes (Security & Correctness)
| # | Task | Files | Est. Effort |
|---|------|-------|-------------|
| 1 | Fix `GetAllowedPDFHosts` — hostname extraction bug | `internal/config/env.go:182` | 15 min |
| 2 | Fix test context key — use typed key helper | `internal/middleware/jwt_context_key.go`, `internal/handler/krs_handler_test.go` | 30 min |
| 3 | Convert service errors to custom error types | `internal/services/pdf_service.go`, `internal/model/errors.go` | 1 hr |
| 4 | Standardise middleware error responses to `ErrorResponse` | `internal/middleware/jwt_auth.go`, `api_key.go`, `rate_limit.go` | 1 hr |

### Phase 2: Configuration & Architecture
| # | Task | Files | Est. Effort |
|---|------|-------|-------------|
| 5 | Cache config — remove per-request `os.Getenv` | `internal/middleware/api_key.go`, `internal/config/env.go`, `cmd/api/main.go` | 1 hr |
| 6 | Add request-ID middleware | New: `internal/middleware/request_id.go` + all log sites | 1.5 hr |
| 7 | Fix trust-proxy config | `internal/config/env.go`, `cmd/api/main.go`, `internal/middleware/rate_limit.go` | 45 min |
| 8 | Add graceful shutdown with context cancellation | `cmd/api/main.go`, `internal/services/cache_service.go` | 45 min |

### Phase 3: Testing & Cleanup
| # | Task | Files | Est. Effort |
|---|------|-------|-------------|
| 9 | Mock external HTTP calls in tests | `internal/services/pdf_service.go`, `pdf_service_test.go`, `krs_handler_test.go` | 2 hr |
| 10 | Remove dead code (`URLValidator`, `output.go`) | `internal/services/pdf_service.go`, `internal/config/output.go` | 15 min |
| 11 | Write `ARCHITECTURE.md` | New: `ARCHITECTURE.md` | 1 hr |

### Phase 4: Verification
| # | Task | Command |
|---|------|---------|
| 12 | All tests pass | `go test ./... -short -race` |
| 13 | No vet warnings | `go vet ./...` |
| 14 | Build succeeds | `go build ./...` |

---

## Appendix: File Change Summary

| File | Action | Lines Affected |
|------|--------|---------------|
| `internal/config/env.go` | Modify | 176-191 (GetAllowedPDFHosts), add GetTrustProxy, add AllowedPDFHosts field |
| `internal/config/output.go` | **DELETE** | Entire file |
| `internal/handler/krs_handler.go` | Modify | 42-47 (context key), 50-53 (error type), 76-86 (error mapping), 158-161 (mapErrorToHTTPStatus), 193-211 (respondError) |
| `internal/handler/krs_handler_test.go` | Modify | 211, 238, 263 (context key), 255-282 (mock integration test) |
| `internal/middleware/api_key.go` | Refactor | Entire file — constructor injection |
| `internal/middleware/jwt_auth.go` | Modify | 113-121 (respondUnauthorized → ErrorResponse) |
| `internal/middleware/jwt_context_key.go` | Modify | Add SetUserNPM helper |
| `internal/middleware/rate_limit.go` | Modify | 97-101 (error format), extract getClientIP, trustProxy config |
| `internal/services/pdf_service.go` | Modify | 37-72 (delete URLValidator), 74-213 (typed errors), constructor injection |
| `internal/services/pdf_service_test.go` | Modify | 56-112, 133-159, 165-203 (mock integration tests) |
| `internal/services/cache_service.go` | Modify | 30-39 (accept context), 102-116 (context-aware cleanup) |
| `internal/handler/cek_response.go` | Modify | 24-30 (consistent response format) |
| `cmd/api/main.go` | Modify | 24-97 (root context, config injection, graceful shutdown) |
| `internal/routes/routes.go` | Modify | 11-24 (add request-id middleware, pass config) |
| `ARCHITECTURE.md` | **CREATE** | New documentation file |
| `internal/middleware/request_id.go` | **CREATE** | New middleware file |