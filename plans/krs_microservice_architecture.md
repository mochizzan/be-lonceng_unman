# KRS PDF Processing Microservice Architecture

## Overview
Sistem microservice untuk memproses KRS (Kartu Rencana Studi) dalam format PDF dari endpoint eksternal, mengekstrak teks secara presisi, memetakan ke struktur JSON yang terdefinisi, dan menyediakan endpoint yang aman dan efisien untuk frontend.

## System Architecture

```mermaid
flowchart TD
    A[Frontend] -->|GET /krs
    (with JWT Auth)| B[API Gateway
be-lonceng_unman]
    B -->|Middleware: Rate Limit| C[KRS Handler
/krs endpoint]
    C -->|Check Cache| D[Cache Layer
Redis/In-Memory]
    D -->|Cache Hit| C
    D -->|Cache Miss| E[PDF Processor Service]
    E -->|Download PDF| F[External PDF Source
https://elearning.universitasmandiri.ac.id]
    F -->|Return PDF| E
    E -->|Extract & Parse| E
    E -->|Store Result| D
    C -->|Response JSON| A
```

## Components

### 1. Security & Authorization
- **Authentication**: JWT token validation
- **Authorization**: Extract NPM from JWT claims (bukan dari query parameter)
- **API Key**: Untuk penggunaan internal antar-service

### 2. New Endpoint
- **Path**: `/krs`
- **Method**: GET
- **Headers**: `Authorization: Bearer <JWT_TOKEN>`
- **Response**: JSON dengan struktur KRS yang sudah diparsing
- **Middleware**: Rate limiting (menggunakan middleware existing)

### 2. Cache Layer
- **Storage**: Redis atau in-memory cache
- **Key**: `krs:{npm}`
- **TTL**: 24 jam (dapat dikonfigurasi)
- **Invalidation**: Saat mahasiswa melakukan perubahan KRS

### 3. PDF Processing Flow
1. Cek cache untuk data KRS berdasarkan NPM
2. Jika cache hit, return data dari cache
3. Jika cache miss:
   - Download PDF dari endpoint eksternal
   - Ekstrak teks dari PDF menggunakan library yang tepat
   - Parsing teks untuk mengidentifikasi struktur KRS
   - Mapping data ke struktur JSON yang terdefinisi
   - Simpan hasil ke cache
4. Return struktur JSON ke frontend

### 4. PDF Processing Details
- **Library**: github.com/ledongthuc/pdf atau pdftotext (poppler-utils)
- **Format**: Mempertahankan layout tabel untuk parsing yang akurat
- **Error Handling**: Deteksi PDF corrupt, format tidak didukung, ekstraksi gagal

### 5. Data Model
```go
package model

type KRSResponse struct {
    Status      string      `json:"status"`
    Message     string      `json:"message,omitempty"`
    Mahasiswa   Mahasiswa   `json:"mahasiswa"`
    TahunAjaran string      `json:"tahun_ajaran"`
    Semester    string      `json:"semester"`
    TanggalCetak string      `json:"tanggal_cetak"`
    MataKuliah  []MataKuliah `json:"mata_kuliah"`
    TotalSKS    int         `json:"total_sks"`
}

type Mahasiswa struct {
    NPM          string `json:"npm"`
    Nama         string `json:"nama"`
    ProgramStudi string `json:"program_studi"`
}

type MataKuliah struct {
    No        int    `json:"no"`
    Kode      string `json:"kode"`
    Nama      string `json:"nama"`
    SKS       int    `json:"sks"`
    Kelas     string `json:"kelas"`
    Dosen     string `json:"dosen"`
    Jadwal    string `json:"jadwal"`
}
```

## Implementation Plan

### Phase 1: Setup & Library Selection
1. Install dan evaluasi library PDF text extraction (ledongthuc/pdf vs pdftotext)
2. Implement PDF downloader service dengan timeout dan error handling
3. Analisis struktur teks KRS secara mendetail dari contoh PDF
4. Implementasikan strategi caching (Redis atau in-memory)

### Phase 2: Security & PDF Processing
1. Implementasikan JWT authentication untuk endpoint /krs
2. Implementasikan PDF text extraction dengan library yang dipilih
3. Buat pattern matching yang presisi untuk semua field KRS
4. Implementasikan validasi data dan error handling

### Phase 3: Data Mapping & Caching
1. Definisikan struktur data KRSResponse berdasarkan field yang telah ditentukan
2. Implementasikan logic parsing teks yang presisi untuk setiap field
3. Implementasikan caching layer dengan TTL yang sesuai
4. Implementasikan cache invalidation strategy

### Phase 4: Endpoint Implementation
1. Buat handler untuk endpoint `/krs` dengan JWT auth
2. Integrasikan dengan middleware rate limiting
3. Implementasikan error handling yang komprehensif
4. Testing endpoint dengan berbagai skenario (cache hit/miss, error cases)

## Error Handling
- **Authentication**: JWT validation failed, token expired
- **Authorization**: NPM mismatch between token and requested data
- **Input Validation**: NPM format validation
- **PDF Download**: Timeout, 404, network errors, invalid PDF URL
- **PDF Processing**: File corrupt, format tidak didukung, ekstraksi gagal
- **Data Parsing**: Format KRS tidak sesuai, field yang hilang atau tidak valid
- **Rate Limiting**: Rate limit exceeded
- **Cache**: Cache connection errors, serialization errors
- **Data Mapping**: Error mapping teks ke struktur JSON

## Security Considerations
- **JWT Validation**: Pastikan token valid dan belum expired
- **Data Isolation**: Pastikan mahasiswa hanya bisa mengakses KRS miliknya sendiri
- **Rate Limiting**: Mencegah brute force attacks
- **HTTPS**: Enforce HTTPS untuk semua komunikasi
- **Input Sanitization**: Mencegah injection attacks

## Testing Strategy
1. Unit test untuk fungsi parsing teks
2. Integration test untuk flow end-to-end
3. Test dengan berbagai format KRS
4. Load testing untuk rate limiting