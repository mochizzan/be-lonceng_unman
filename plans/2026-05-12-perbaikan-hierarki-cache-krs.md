# Rencana Perbaikan Hierarki Cache KRS

## 1. Ringkasan Eksekutif
Tujuan dari perbaikan ini adalah untuk mengimplementasikan strategi **Multi-Level Caching (Fallthrough Cache)** pada endpoint KRS. Saat ini, sistem mengabaikan cache persisten di disk (L2) dan langsung memproses PDF jika cache in-memory (L1) kosong. Hal ini menyebabkan beban kerja yang tidak perlu pada server dan peningkatan latency bagi pengguna setelah server restart.

## 2. Arsitektur Cache yang Diusulkan

Sistem akan menggunakan hierarki berikut untuk pengambilan data:

**L1: In-Memory Cache (`CacheService`)**
- **Karakteristik:** Sangat cepat, volatile (hilang saat restart).
- **Tujuan:** Menangani request berulang dalam waktu singkat.

**L2: File System Cache (`FileStorage`)**
- **Karakteristik:** Cepat, persisten (tetap ada setelah restart).
- **Tujuan:** Menghindari pemrosesan ulang PDF yang berat setelah restart server atau setelah L1 expired.

**Source: PDF Processing (`PDFService`)**
- **Karakteristik:** Sangat lambat, resource-intensive.
- **Tujuan:** Sumber kebenaran data terakhir.

### Diagram Alur Pengambilan Data
`Request` $\rightarrow$ `Cek L1` $\rightarrow$ (Hit? $\rightarrow$ `Return`) $\rightarrow$ `Cek L2` $\rightarrow$ (Hit? $\rightarrow$ `Promosi ke L1` $\rightarrow$ `Return`) $\rightarrow$ `Proses PDF` $\rightarrow$ `Simpan ke L1 & L2` $\rightarrow$ `Return`.

## 3. Detail Implementasi Teknis

### 3.1 Modifikasi `internal/handler/krs_handler.go`

#### A. Pembaruan Fungsi `getCachedKRS`
Fungsi ini akan diubah dari sekadar wrapper `CacheService.Get` menjadi orkestrator cache.

**Logika Baru:**
1. Panggil `h.cacheService.Get(ctx, "krs:"+npm)`.
2. Jika ditemukan $\rightarrow$ Log `"L1 Cache Hit"` $\rightarrow$ Return data.
3. Jika tidak ditemukan (Miss):
    a. Panggil `h.fileStorage.GetCachedResponse(npm)`.
    b. Jika ditemukan (L2 Hit):
        i. Unmarshal data JSON dari L2 ke `model.KRSResponse`.
        ii. Simpan kembali data tersebut ke L1 menggunakan `h.cacheService.Set()` agar request berikutnya menjadi L1 Hit.
        iii. Log `"L2 Cache Hit, promoting to L1"` $\rightarrow$ Return data.
    c. Jika tidak ditemukan $\rightarrow$ Return `nil, error("full cache miss")`.

#### B. Pembaruan Fungsi `GetKRSByNIS`
Menyesuaikan penanganan error dari `getCachedKRS` untuk memicu proses PDF hanya jika terjadi "full cache miss".

#### C. Strategi Logging & Observability Mendalam
Untuk memastikan setiap masalah dapat dilacak (traceability), sistem logging akan ditingkatkan dengan standar berikut:

**1. Level Logging:**
- `DEBUG`: Detail langkah-langkah internal (misal: "Checking L1 for NPM X", "Attempting to read L2 file Y").
- `INFO`: Hasil akhir dari proses cache (L1 Hit, L2 Hit, Full Miss).
- `WARN`: Masalah yang dapat dipulihkan (misal: "L2 file found but corrupted, falling back to PDF", "L2 file expired").
- `ERROR`: Kegagalan kritis (misal: "Failed to write to L2 storage", "PDF processing failed").

**2. Detail Log yang Wajib Ada:**
Setiap log harus menyertakan: `request_id`, `npm`, dan `duration` (untuk operasi yang memakan waktu).

**3. Mapping Log Baru:**
- **L1 Hit:** `INFO: [L1 HIT] KRS data retrieved from memory | npm: %s | duration: %s`
- **L2 Hit:** `INFO: [L2 HIT] KRS data retrieved from disk, promoting to L1 | npm: %s | duration: %s`
- **L2 Corrupt/Expired:** `WARN: [L2 INVALID] Cache file found but invalid/expired, processing PDF | npm: %s | error: %s`
- **Full Miss:** `INFO: [FULL MISS] No cache found in L1/L2, processing PDF | npm: %s`
- **PDF Success:** `INFO: [PDF SUCCESS] KRS processed and cached to L1 & L2 | npm: %s | duration: %s | courses: %d`
- **PDF Failure:** `ERROR: [PDF FAILURE] Failed to process KRS PDF | npm: %s | error: %s | request_id: %s`

**4. Performance Tracking:**
Menambahkan log durasi untuk setiap tahap guna mengidentifikasi bottleneck:
- `L1 Lookup Time` $\rightarrow$ `L2 Lookup Time` $\rightarrow$ `PDF Processing Time`.

### 3.2 Penanganan Edge Cases & Risiko

| Risiko | Mitigasi |
| :--- | :--- |
| **File L2 Korup** | Implementasikan `try-catch` atau pengecekan error saat `json.Unmarshal` data dari L2. Jika korup, anggap sebagai miss dan proses ulang PDF. |
| **Inkonsistensi TTL** | Pastikan TTL di L1 (24 jam) dan L2 (7 hari) sinkron secara logika. Data L2 yang sudah expired harus dihapus oleh `FileStorage.GetCachedResponse`. |
| **Race Condition** | `CacheService` sudah menggunakan `sync.RWMutex`, sehingga aman untuk akses konkuren. |

## 4. Strategi Pengujian

### 4.1 Skenario Unit Test
1. **Skenario L1 Hit:**
   - Set data di `CacheService`.
   - Panggil `GetKRSByNIS`.
   - Verifikasi bahwa `PDFService` tidak dipanggil.
2. **Skenario L2 Hit:**
   - Kosongkan `CacheService`.
   - Simpan file JSON valid di `storage/response/`.
   - Panggil `GetKRSByNIS`.
   - Verifikasi bahwa data diambil dari file, disimpan ke L1, dan `PDFService` tidak dipanggil.
3. **Skenario Full Miss:**
   - Kosongkan L1 dan L2.
   - Panggil `GetKRSByNIS`.
   - Verifikasi bahwa `PDFService` dipanggil dan data disimpan ke L1 & L2.

### 4.2 Skenario Integration Test
- Jalankan server $\rightarrow$ Request NPM A (Full Miss).
- Restart server $\rightarrow$ Request NPM A (Harus L2 Hit).
- Request NPM A lagi (Harus L1 Hit).

## 5. Urutan Eksekusi (Step-by-Step)
1. **Tahap 1:** Implementasi logika `getCachedKRS` yang baru.
2. **Tahap 2:** Integrasi logika tersebut ke dalam `GetKRSByNIS`.
3. **Tahap 3:** Penambahan logging detail.
4. **Tahap 4:** Penulisan dan eksekusi test case.
5. **Tahap 5:** Verifikasi akhir melalui log server.
