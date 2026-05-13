package fixtures

// SampleKRSTextValid adalah sample teks KRS yang valid dan lengkap
const SampleKRSTextValid = `KARTU RENCANA STUDI
Nama
:
MOCHAMAD IZZAN FIRASYANSYAH
N P M
:
2211700006
Program Studi
:
Sistem Informasi
Tahun Ajaran
:
2025/2026
Semester
:
GENAP
No.
Kode
Mata Kuliah
SKS
KELAS
DOSEN
JADWAL
1
SI40306
Tugas Akhir/Skripsi
6
SI-8A
TIM DOSEN FAKULTAS TEKNIK
Sabtu
08:00 s/d 09:40
2
SI30201
Pemrograman Web
3
SI-8B
Dr. Budi
Senin
10:00 s/d 11:40
Total
9
, 09 Mei 2026
Mahasiswa
Ketua Prgram Studi
..............................................
NIDN.......................................`

// SampleKRSTextNoCourses adalah sample teks KRS tanpa mata kuliah
const SampleKRSTextNoCourses = `KARTU RENCANA STUDI
Nama
:
JOHN DOE
N P M
:
1234567890
Program Studi
:
Teknik Informatika
Tahun Ajaran
:
2025/2026
Semester
:
GENAP
No.
Kode
Mata Kuliah
SKS
KELAS
DOSEN
JADWAL
Total
0
, 09 Mei 2026`

// SampleKRSTextMultipleCourses adalah sample dengan banyak mata kuliah
const SampleKRSTextMultipleCourses = `KARTU RENCANA STUDI
Nama
:
JANE SMITH
N P M
:
9876543210
Program Studi
:
Sistem Informasi
Tahun Ajaran
:
2025/2026
Semester
:
GANJIL
No.
Kode
Mata Kuliah
SKS
KELAS
DOSEN
JADWAL
1
SI10101
Algoritma Pemrograman
3
SI-1A
Dr. Ahmad
Senin
08:00 s/d 09:40
2
SI10102
Basis Data
4
SI-1B
Prof. Siti
Selasa
10:00 s/d 11:40
3
SI10103
Jaringan Komputer
3
SI-1A
Dr. Rudi
Rabu
14:00 s/d 15:40
4
SI10104
Matematika Diskrit
2
SI-1C
Dr. Dewi
Kamis
08:00 s/d 09:40
Total
12
, 10 Mei 2026`

// SampleKRSTextMissingMahasiswa adalah sample tanpa data mahasiswa lengkap
const SampleKRSTextMissingMahasiswa = `KARTU RENCANA STUDI
Nama
:
N P M
:
Program Studi
:
Tahun Ajaran
:
2025/2026
Semester
:
GENAP`
