# API Error Handling & Storage Cache Plan

## Executive Summary
Improve error handling and add persistent storage cache for KRS PDF endpoint:
1. Handle empty/invalid data (NPM, Nama kosong)
2. PDF download failures & endpoint unavailability
3. Add storage cache layer to prevent rate limiting
4. 7-day TTL with automatic refresh

---

## Storage Architecture

### Directory Structure
```
storage/
├── raw/                    # Raw downloaded PDF files
│   └── {nis}_{timestamp}.pdf
├── response/               # Parsed KRS JSON responses
│   └── {nis}_{timestamp}.json
└── extracted/              # Extracted text from PDF (backup)
    └── {nis}_{timestamp}.txt
```

### Filename Convention
Format: `{nis}_{YYYYMMDD_HHmmss}.{ext}`
Example: `2211700006_20260508_071530.pdf`

**Benefits:**
- Timestamp-based unique naming
- NIS clearly visible for debugging
- Date embedded in filename for easy age calculation

### Cache Logic (7-Day TTL)
```
1. Request comes in with NIS
2. Check if valid cached files exist in storage/{nis}_*.{ext}
3. If exists AND file age < 7 days → Return from cache (no download)
4. If not exists OR file age >= 7 days → Download fresh PDF
5. Save new files with current timestamp
6. Delete old files for that NIS (if any)
7. Return response
```

---

## Files to Create/Modify

### NEW: `internal/storage/file_storage.go`
Handles file storage operations:
```go
type FileStorage struct {
    basePath string
}

func (s *FileStorage) SavePDF(nis string, data []byte) error
func (s *FileStorage) SaveResponse(nis string, data []byte) error
func (s *FileStorage) GetCachedPDF(nis string) ([]byte, error)  // Returns nil if expired
func (s *FileStorage) GetCachedResponse(nis string) ([]byte, error)
func (s *FileStorage) IsCacheValid(nis string) (bool, error)  // Check 7-day TTL
func (s *FileStorage) CleanupExpired(nis string) error
func (s *FileStorage) GetCacheAge(nis string) (time.Duration, error)
```

### NEW: `internal/model/error_codes.go`
Error code constants:
```go
const (
    ErrCodeNISEmpty           = "ERR_NIS_EMPTY"
    ErrCodeNISInvalid         = "ERR_NIS_INVALID"
    ErrCodeDataNotFound       = "ERR_DATA_NOT_FOUND"
    ErrCodeKRSEmpty           = "ERR_KRS_EMPTY"
    ErrCodePDFDownloadFailed  = "ERR_PDF_DOWNLOAD_FAILED"
    ErrCodePDFExternalUnavail = "ERR_PDF_EXTERNAL_UNAVAILABLE"
    ErrCodePDFTimeout         = "ERR_PDF_TIMEOUT"
    ErrCodePDFParseFailed     = "ERR_PDF_PARSE_FAILED"
    ErrCodeInvalidResponse    = "ERR_INVALID_RESPONSE"
    ErrCodeUnauthorized       = "ERR_UNAUTHORIZED"
    ErrCodeInternalServer     = "ERR_INTERNAL_SERVER"
    ErrCodeCacheReadFailed    = "ERR_CACHE_READ_FAILED"
    ErrCodeCacheWriteFailed   = "ERR_CACHE_WRITE_FAILED"
)
```

### NEW: `internal/model/error_response.go`
Error response struct:
```go
type ErrorResponse struct {
    Status    string `json:"status"`
    Code      string `json:"code"`
    Message   string `json:"message"`
    Details   string `json:"details,omitempty"`
    Timestamp string `json:"timestamp"`
}
```

### NEW: `internal/model/response_wrapper.go`
Success response wrapper:
```go
type SuccessResponse struct {
    Status string      `json:"status"`
    Data   interface{} `json:"data"`
    Meta   ResponseMeta `json:"meta"`
}

type ResponseMeta struct {
    Timestamp  string `json:"timestamp"`
    Version     string `json:"version"`
    Cached      bool   `json:"cached"`       // true if from storage
    CacheAge    int64  `json:"cache_age_hours,omitempty"`  // hours if cached
}
```

### MODIFY: `internal/model/krs_model.go`
- `ExtractMahasiswa()` - Validate NPM & Nama not empty
- `ExtractMataKuliah()` - Better error message for empty list
- `Parse()` - Add validation for data completeness

### MODIFY: `internal/services/pdf_service.go`
- `DownloadPDF()` - Validate PDF magic bytes (`%PDF-`)
- Minimum file size check (>100 bytes)
- Return specific errors for 404, 503, timeout

### MODIFY: `internal/handler/krs_handler.go`
- Add `FileStorage` dependency
- Check cache before download
- Update cache after successful processing
- Return structured `SuccessResponse` with `meta.cached` flag

---

## Updated Flow: With Storage Cache

```
Request → JWT Auth → Validate NIS
                        ↓
              Check storage/cache
                        ↓
              ┌─────────┴─────────┐
              ↓                   ↓
         Cache valid?          Cache invalid/expired
         (age < 7 days)        (age >= 7 days or not exist)
              ↓                   ↓
         Read from storage      Download PDF from external
              ↓                   ↓
         Parse JSON              Validate PDF (magic bytes)
              ↓                   ↓
         Return response         Extract text from PDF
         (cached: true)              ↓
                                   Parse to KRS structure
                                        ↓
                                   Validate data completeness
                                        ↓
                                   Save to storage
                                   (raw PDF + JSON response)
                                        ↓
                                   Return response
                                   (cached: false)
```

---

## Error Handling Flow

```
Error during processing:
1. Download failed (404/503/timeout)
   → Return appropriate error with HTTP status
   
2. PDF invalid (not a real PDF)
   → ERR_INVALID_RESPONSE + 502
   
3. Parse failed (NPM/Nama empty)
   → ERR_DATA_NOT_FOUND + 404
   
4. MataKuliah empty
   → ERR_KRS_EMPTY + 404
   
5. Storage write failed
   → Log error but continue (non-critical)
   → Return response anyway
```

---

## API Response Examples

### Success - From Cache (200)
```json
{
    "status": "success",
    "data": {
        "mahasiswa": { "npm": "2211700006", "nama": "JOHN DOE" },
        "mata_kuliah": [...]
    },
    "meta": {
        "timestamp": "2026-05-08T07:15:30Z",
        "version": "1.0.0",
        "cached": true,
        "cache_age_hours": 12
    }
}
```

### Success - Fresh Download (200)
```json
{
    "status": "success",
    "data": { ... },
    "meta": {
        "timestamp": "2026-05-08T07:15:30Z",
        "version": "1.0.0",
        "cached": false
    }
}
```

### Error Response
```json
{
    "status": "error",
    "code": "ERR_DATA_NOT_FOUND",
    "message": "Data mahasiswa tidak ditemukan",
    "details": "NPM atau Nama kosong dalam response dari server",
    "timestamp": "2026-05-08T07:15:30Z"
}
```

---

## HTTP Status Code Mapping

| Error | HTTP Status | Error Code |
|-------|-------------|------------|
| NIS invalid | 400 | ERR_NIS_INVALID |
| Unauthorized | 401 | ERR_UNAUTHORIZED |
| NPM/Nama empty | 404 | ERR_DATA_NOT_FOUND |
| KRS empty | 404 | ERR_KRS_EMPTY |
| PDF not found (404) | 502 | ERR_PDF_DOWNLOAD_FAILED |
| External unavailable | 503 | ERR_PDF_EXTERNAL_UNAVAILABLE |
| Request timeout | 408 | ERR_PDF_TIMEOUT |
| PDF invalid/not a PDF | 502 | ERR_INVALID_RESPONSE |
| Parse failed | 500 | ERR_PDF_PARSE_FAILED |

---

## Files Summary

| File | Action | Purpose |
|------|--------|---------|
| `internal/storage/file_storage.go` | CREATE | Storage operations, 7-day TTL |
| `internal/model/error_codes.go` | CREATE | Error code constants |
| `internal/model/error_response.go` | CREATE | Error response struct |
| `internal/model/response_wrapper.go` | CREATE | Success wrapper |
| `internal/model/krs_model.go` | MODIFY | Data validation |
| `internal/services/pdf_service.go` | MODIFY | PDF validation |
| `internal/handler/krs_handler.go` | MODIFY | Storage integration, error handling |

---

## Testing Scenarios

| Scenario | Input | Expected |
|----------|-------|----------|
| First request | New NIS | Download, save, return (cached: false) |
| Cache hit | Same NIS within 7 days | Return from storage (cached: true) |
| Cache expired | Same NIS after 7 days | Download new, delete old, return |
| Invalid NIS format | `nis="1"` | 400 + ERR_NIS_INVALID |
| Student not found | Valid NIS, no data | 404 + ERR_DATA_NOT_FOUND |
| External down | 503 from endpoint | 503 + ERR_PDF_EXTERNAL_UNAVAILABLE |
| PDF timeout | 30s+ request | 408 + ERR_PDF_TIMEOUT |
| Invalid PDF | HTML error page | 502 + ERR_INVALID_RESPONSE |
