# 🔍 Audit Findings & Remediation Plan

**Tanggal Audit:** 2026-05-09  
**Auditor:** OWL (Automated Code Audit)  
**Scope:** End-to-end workflow analysis, unit testing, integration testing, security testing

---

## Executive Summary

Audit menyeluruh telah dilakukan terhadap `be-lonceng_unman` KRS extraction service. Sistem secara umum berfungsi dengan baik, namun ditemukan **5 masalah** yang perlu diperbaiki untuk production readiness.

| ID | Severity | Status | Deskripsi |
|----|----------|--------|-----------|
| F-01 | 🔴 HIGH | Open | Error wrapping memecah error type mapping |
| F-02 | 🟡 MEDIUM | Open | `tanggal_cetak` tidak ter-extract dari PDF nyata |
| F-03 | 🟡 MEDIUM | Open | Duplicate mock transport di pdf_service_test.go |
| F-04 | 🟢 LOW | Open | `Meta.Version` kosong di error responses |
| F-05 | 🟢 LOW | Open | Tidak ada test untuk middleware chain lengkap |

---

## F-01: 🔴 HIGH - Error Wrapping Memecah Error Type Mapping

### Lokasi
- [`internal/services/pdf_service.go:147`](internal/services/pdf_service.go:147) - `ProcessKRSFromURL` menggunakan `fmt.Errorf("...: %w", err)`
- [`internal/model/errors.go:127-159`](internal/model/errors.go:127) - `MapErrorToHTTPStatus` menggunakan `errors.Is`

### Masalah
Ketika error di-wrap dengan `fmt.Errorf("download PDF: %w", err)`, `errors.Is()` tidak dapat mengenali tipe custom struct pointer (`&InvalidResponseError{}`) karena `errors.Is` membandingkan pointer, bukan nilai struct.

```go
// pdf_service.go - error di-wrap
return nil, fmt.Errorf("download PDF: %w", err)

// errors.go - errors.Is gagal match untuk wrapped custom struct pointer
case errors.Is(err, &InvalidResponseError{}): // TIDAK AKAN MATCH jika wrapped
```

### Dampak
- Error dari PDF service yang di-wrap akan selalu fallback ke `ERR_INTERNAL_SERVER` (500)
- Seharusnya `InvalidResponseError` → `ERR_INVALID_RESPONSE` (502)
- Seharusnya `PDFDownloadFailedError` → `ERR_PDF_DOWNLOAD_FAILED` (502)

### Remediasi
Implementasikan `Unwrap()` method pada semua custom error types, atau gunakan `errors.As()`:

```go
// Opsi A: Gunakan errors.As di MapErrorToHTTPStatus
func MapErrorToHTTPStatus(err error) (int, string) {
    var nisEmpty *NISEmptyError
    var nisInvalid *NISInvalidError
    var invalidResp *InvalidResponseError
    // ... dst
    
    switch {
    case errors.As(err, &nisEmpty):
        return http.StatusBadRequest, ErrCodeNISEmpty
    case errors.As(err, &nisInvalid):
        return http.StatusBadRequest, ErrCodeNISInvalid
    case errors.As(err, &invalidResp):
        return http.StatusBadGateway, ErrCodeInvalidResponse
    // ... dst
    }
}
```

---

## F-02: 🟡 MEDIUM - `tanggal_cetak` Tidak Ter-Extract dari PDF Nyata

### Lokasi
- [`internal/parser/krs_parser.go:229-238`](internal/parser/krs_parser.go:229) - `ExtractTanggalCetak`

### Masalah
Regex pattern `Tanggal\s*Cetak\s*:\s*(\d{2}/\d{2}/\d{4})` tidak cocok dengan format tanggal di PDF nyata. Dari hasil debug test, format tanggal di PDF adalah:

```
Total
0
, 09 Mei 2026
```

Tanggal cetak menggunakan format `DD MMMM YYYY` (Indonesia) tanpa prefix "Tanggal Cetak :".

### Dampak
- Field `tanggal_cetak` selalu kosong (`""`) di response
- Data tidak lengkap untuk kebutuhan reporting

### Remediasi
Update regex untuk mendukung format tanggal Indonesia:

```go
func (p *KRSParser) ExtractTanggalCetak() string {
    // Format 1: "Tanggal Cetak : 07/05/2026"
    pattern1 := regexp.MustCompile(`Tanggal\s*Cetak\s*:\s*(\d{2}/\d{2}/\d{4})`)
    if matches := pattern1.FindStringSubmatch(p.textContent); len(matches) >= 2 {
        return strings.TrimSpace(matches[1])
    }
    
    // Format 2: ", 09 Mei 2026" (format Indonesia dari PDF nyata)
    pattern2 := regexp.MustCompile(`,\s*(\d{1,2})\s+(Januari|Februari|Maret|April|Mei|Juni|Juli|Agustus|September|Oktober|November|Desember)\s+(\d{4})`)
    if matches := pattern2.FindStringSubmatch(p.textContent); len(matches) >= 4 {
        // Convert to DD/MM/YYYY
        monthMap := map[string]string{
            "Januari": "01", "Februari": "02", "Maret": "03", "April": "04",
            "Mei": "05", "Juni": "06", "Juli": "07", "Agustus": "08",
            "September": "09", "Oktober": "10", "November": "11", "Desember": "12",
        }
        day := fmt.Sprintf("%02d", atoi(matches[1]))
        month := monthMap[matches[2]]
        return fmt.Sprintf("%s/%s/%s", day, month, matches[3])
    }
    
    return ""
}
```

---

## F-03: 🟡 MEDIUM - Duplicate Mock Transport di pdf_service_test.go

### Lokasi
- [`internal/services/pdf_service_test.go:59-72`](internal/services/pdf_service_test.go:59) - `mockPDFTransport`
- [`internal/services/pdf_service_test.go:200-213`](internal/services/pdf_service_test.go:200) - `mockPDFTransport2` (duplikat identik)

### Masalah
Terdapat dua set mock transport yang identik (`mockPDFTransport`/`mockPDFTransport2` dan `mockSlowTransport`/`mockSlowTransport2`) dengan test cases yang juga identik. Ini menambah ~130 baris kode redundan tanpa nilai tambah.

### Dampak
- Code bloat, maintenance overhead
- Test execution time meningkat tanpa alasan
- Membingungkan developer yang membaca test

### Remediasi
Hapus set kedua (mockPDFTransport2, mockSlowTransport2) dan test cases yang terkait. Satu set sudah cukup untuk coverage.

---

## F-04: 🟢 LOW - `Meta.Version` Kosong di Error Responses

### Lokasi
- [`internal/middleware/jwt_auth.go:117-141`](internal/middleware/jwt_auth.go:117) - `respondError`
- [`internal/middleware/rate_limit.go:96`](internal/middleware/rate_limit.go:96) - `respondError`

### Masalah
Error response dari middleware (JWT, Rate Limit) tidak mengisi `Meta.Version` dan `Meta.Service`:

```json
{
  "status": "success",
  "data": {"status": "error", "code": "ERR_UNAUTHORIZED", ...},
  "meta": {"timestamp": "...", "version": "", "cached": false}
}
```

Sementara success response dari handler mengisi dengan benar:

```json
{
  "status": "success",
  "data": {...},
  "meta": {"timestamp": "...", "version": "0.0.1", "service": "be-lonceng_unman", "cached": false}
}
```

### Dampak
- Inkonsistensi response format
- Client yang bergantung pada `meta.version` akan mendapat nilai kosong di error cases

### Remediasi
Update `respondError` di middleware untuk mengisi version dan service:

```go
// middleware/jwt_auth.go
response := model.SuccessResponse{
    Status: "success",
    Data:   model.ErrorResponse{...},
    Meta: model.ResponseMeta{
        Timestamp: time.Now().UTC().Format(time.RFC3339),
        Version:   config.GetAppVersion(),
        Service:   config.GetServiceName(),
    },
}
```

---

## F-05: 🟢 LOW - Tidak Ada Test untuk Middleware Chain Lengkap

### Lokasi
- [`internal/routes/routes.go:12-34`](internal/routes/routes.go:12) - `SetupRoutes`

### Masalah
Tidak ada test yang memverifikasi middleware chain secara terintegrasi:
- API Key → Request ID → Rate Limit → JWT → Handler

Test yang ada hanya menguji masing-masing komponen secara terpisah.

### Dampak
- Regresi pada middleware order tidak terdeteksi
- Tidak ada jaminan bahwa chain bekerja benar secara bersamaan

### Remediasi
Buat integration test untuk full middleware chain (sudah dibuat di `test/e2e/auth_flow_test.go`).

---

## Test Results Summary

| Test Suite | Total | Pass | Fail | Skip |
|------------|-------|------|------|------|
| Unit Tests (short) | 45 | 45 | 0 | 0 |
| Integration Tests | 7 | 6 | 1 | 0 |
| E2E Curl Tests | 6 | 6 | 0 | 0 |
| **TOTAL** | **58** | **57** | **1** | **0** |

### Failed Tests
1. `TestPDFEndpoint_Integration` - PDF NIS `1234567890` tidak memiliki data mata kuliah (bukan bug sistem, tapi test data issue)

---

## Rekomendasi Prioritas

1. **Segera (P0):** Fix F-01 - Error mapping yang salah bisa menyebabkan 500 alih-alih 502/404
2. **Minggu ini (P1):** Fix F-02 - Data tanggal cetak harus lengkap
3. **Sprint ini (P2):** Fix F-03 - Bersihkan duplicate test code
4. **Backlog (P3):** Fix F-04, F-05 - Konsistensi dan coverage
