package model

// KRSResponse adalah struktur utama response untuk endpoint KRS
type KRSResponse struct {
	Status       string       `json:"status"`
	Message      string       `json:"message,omitempty"`
	Mahasiswa    Mahasiswa    `json:"mahasiswa"`
	TahunAjaran  string       `json:"tahun_ajaran"`
	Semester     string       `json:"semester"`
	TanggalCetak string       `json:"tanggal_cetak"`
	MataKuliah   []MataKuliah `json:"mata_kuliah"`
	TotalSKS     int          `json:"total_sks"`
}

// Mahasiswa berisi data identitas mahasiswa
type Mahasiswa struct {
	NPM          string `json:"npm"`
	Nama         string `json:"nama"`
	ProgramStudi string `json:"program_studi"`
}

// MataKuliah berisi detail setiap mata kuliah di KRS
type MataKuliah struct {
	No     int    `json:"no"`
	Kode   string `json:"kode"`
	Nama   string `json:"nama"`
	SKS    int    `json:"sks"`
	Kelas  string `json:"kelas"`
	Dosen  string `json:"dosen"`
	Jadwal string `json:"jadwal"`
}
