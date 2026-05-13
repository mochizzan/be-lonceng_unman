# Rencana Implementasi: Workflow KRS Aman & Enkripsi Storage

Dokumen ini menjelaskan langkah-langkah teknis untuk mengintegrasikan `CHACHA_KEY` guna mengenkripsi data mahasiswa di storage dan mengoptimalkan alur pengambilan KRS dengan kebijakan *stale-check* 3 hari.

## Goal Utama
1.  **Stale-Check 3 Hari**: Jika data di storage lebih lama dari 3 hari, sistem akan melakukan crawl ulang.
2.  **Enkripsi At-Rest**: Data yang disimpan di storage dienkripsi menggunakan ChaCha20-Poly1305.
3.  **Penyimpanan Paralel**: Proses simpan ke storage dilakukan di background agar response ke user lebih cepat.

## Perubahan yang Diusulkan

### 1. Konfigurasi (Internal Config)
Memastikan `CHACHA_KEY` terbaca dengan benar dari `.env`.
- **File**: `internal/config/env.go`
- **Tugas**: Tambahkan validasi kunci (harus 32 byte).

### 2. Package Keamanan (Crypto Utility) [BARU]
Membuat fungsi utilitas untuk enkripsi yang cepat dan aman.
- **File**: `internal/pkg/crypto/crypto.go`
- **Tugas**: Implementasi fungsi `Encrypt` dan `Decrypt` menggunakan `crypto/chacha20poly1305`.

### 3. Storage Terenkripsi
Memodifikasi layer storage agar menangani enkripsi secara transparan.
- **File**: `internal/storage/file_storage.go`
- **Tugas**: 
    - Enkripsi data di `SaveResponse` sebelum ditulis ke disk.
    - Dekripsi data di `GetCachedResponse` setelah dibaca dari disk.
    - Sesuaikan logika pengecekan umur file menjadi 3 hari.

### 4. Handler Logic (Workflow Baru)
Menggabungkan semua komponen di level API.
- **File**: `internal/handler/krs_handler.go`
- **Alur**:
    1. Cek file storage (Decrypt & Check Age).
    2. Jika valid < 3 hari -> Kirim Response.
    3. Jika > 3 hari/tidak ada -> Jalankan `pdfService.ProcessKRSFromURL`.
    4. Setelah dapat data baru:
        - Kirim JSON ke user segera.
        - Jalankan `go func()` (background) untuk Encrypt & Save ke storage.

---

## Rencana Verifikasi

### Pengujian Manual
1.  Hapus folder `output/storage`.
2.  Lakukan request KRS.
3.  Cek file JSON di folder `output/storage/response/`. Pastikan isinya berantakan (terenkripsi).
4.  Lakukan request kedua; harusnya sangat cepat (Cache Hit).
5.  Ubah tanggal modifikasi file secara manual menjadi 4 hari yang lalu.
6.  Lakukan request ketiga; log harus menunjukkan "Cache stale, re-crawling".

---
**Status**: Menunggu persetujuan user untuk memulai Phase 1 & 2.
