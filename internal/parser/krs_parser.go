package parser

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"be-lonceng_unman/internal/model"
)

// KRSParser untuk menangani parsing teks dari PDF
type KRSParser struct {
	textContent string
}

// NewKRSParser membuat instance parser baru
func NewKRSParser(text string) *KRSParser {
	return &KRSParser{textContent: text}
}

// Parse melakukan parsing teks menjadi struktur KRSResponse
func (p *KRSParser) Parse() (*model.KRSResponse, error) {
	if p.textContent == "" {
		return nil, errors.New("text content is empty")
	}

	// Ekstrak data mahasiswa
	mahasiswa, err := p.ExtractMahasiswa()
	if err != nil {
		return nil, fmt.Errorf("extract mahasiswa: %w", err)
	}

	// Ekstrak tahun ajaran dan semester
	tahunAjaran, semester, err := p.ExtractAcademicInfo()
	if err != nil {
		return nil, fmt.Errorf("extract academic info: %w", err)
	}

	// Ekstrak mata kuliah
	matkulList, totalSKS, err := p.ExtractMataKuliah()
	if err != nil {
		return nil, fmt.Errorf("extract mata kuliah: %w", err)
	}

	// Ekstrak tanggal cetak
	tanggalCetak := p.ExtractTanggalCetak()

	return &model.KRSResponse{
		Status:       "success",
		Mahasiswa:    *mahasiswa,
		TahunAjaran:  tahunAjaran,
		Semester:     semester,
		TanggalCetak: tanggalCetak,
		MataKuliah:   matkulList,
		TotalSKS:     totalSKS,
	}, nil
}

// ExtractMahasiswa mengekstrak data mahasiswa dari teks
func (p *KRSParser) ExtractMahasiswa() (*model.Mahasiswa, error) {
	// Pattern untuk mencari NPM, Nama, Program Studi
	// Contoh format: "Nama : [NAMA]" "N P M : [NPM]" "Program Studi : [PRODI]"
	npmPattern := regexp.MustCompile(`N\s*P\s*M\s*:\s*(\d{10})`)
	namaPattern := regexp.MustCompile(`Nama\s*:\s*([^\n]+?)(?:\n|N\s*P\s*M)`)
	prodiPattern := regexp.MustCompile(`Program\s*Studi\s*:\s*([^\n]+?)(?:\n|Tahun\s*Ajaran)`)

	npmMatches := npmPattern.FindStringSubmatch(p.textContent)
	namaMatches := namaPattern.FindStringSubmatch(p.textContent)
	prodiMatches := prodiPattern.FindStringSubmatch(p.textContent)

	if len(npmMatches) < 2 || len(namaMatches) < 2 || len(prodiMatches) < 2 {
		return nil, errors.New("failed to extract mahasiswa data: required fields not found")
	}

	// Validate NPM is not empty
	npm := strings.TrimSpace(npmMatches[1])
	if npm == "" {
		return nil, errors.New("NPM tidak boleh kosong")
	}

	// Validate Nama is not empty
	nama := strings.TrimSpace(namaMatches[1])
	if nama == "" {
		return nil, errors.New("Nama tidak boleh kosong")
	}

	return &model.Mahasiswa{
		NPM:          npm,
		Nama:         nama,
		ProgramStudi: strings.TrimSpace(prodiMatches[1]),
	}, nil
}

// ExtractAcademicInfo mengekstrak tahun ajaran dan semester
func (p *KRSParser) ExtractAcademicInfo() (string, string, error) {
	// Contoh format: "Tahun Ajaran : 2025/2026" "Semester : GENAP"
	tahunPattern := regexp.MustCompile(`Tahun\s*Ajaran\s*:\s*([0-9]{4}/[0-9]{4})`)
	semesterPattern := regexp.MustCompile(`Semester\s*:\s*([A-Z]+)`)

	tahunMatches := tahunPattern.FindStringSubmatch(p.textContent)
	semesterMatches := semesterPattern.FindStringSubmatch(p.textContent)

	if len(tahunMatches) < 2 || len(semesterMatches) < 2 {
		return "", "", errors.New("failed to extract academic info: tahun ajaran or semester not found")
	}

	return strings.TrimSpace(tahunMatches[1]), strings.TrimSpace(semesterMatches[1]), nil
}

// ExtractMataKuliah mengekstrak daftar mata kuliah dari teks
// Format PDF: setiap field berada pada baris terpisah (line-by-line)
// Header: No., Kode, Mata Kuliah, SKS, KELAS, DOSEN, JADWAL
// Data per mata kuliah: 8 baris (no, kode, nama, sks, kelas, dosen, jadwal_day, jadwal_time)
func (p *KRSParser) ExtractMataKuliah() ([]model.MataKuliah, int, error) {
	lines := strings.Split(p.textContent, "\n")
	var matkulList []model.MataKuliah
	var totalSKS int

	// Cari posisi header (baris yang berisi "No." atau "No")
	headerFound := false
	headerFields := []string{"No", "Kode", "Mata Kuliah", "SKS", "KELAS", "DOSEN", "JADWAL"}
	headerIndex := 0

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		// Check if this line matches first header field
		if strings.EqualFold(trimmed, "No.") || strings.EqualFold(trimmed, "No") {
			// Verify subsequent lines match header fields
			if i+6 < len(lines) {
				allMatch := true
				for j, field := range headerFields {
					nextLine := strings.TrimSpace(lines[i+j])
					if !strings.Contains(strings.ToUpper(nextLine), strings.ToUpper(field)) {
						allMatch = false
						break
					}
				}
				if allMatch {
					headerFound = true
					headerIndex = i + len(headerFields) // Start after header
					break
				}
			}
		}
	}

	if !headerFound {
		return nil, 0, errors.New("mata kuliah table not found")
	}

	// Parse data: setiap mata kuliah memiliki 8 baris
	// no, kode, nama, sks, kelas, dosen, jadwal_day, jadwal_time
	for headerIndex < len(lines) {
		// Skip empty lines
		for headerIndex < len(lines) && strings.TrimSpace(lines[headerIndex]) == "" {
			headerIndex++
		}
		if headerIndex >= len(lines) {
			break
		}

		// Check if we've hit the "Total" line or end markers
		trimmed := strings.TrimSpace(lines[headerIndex])
		if strings.EqualFold(trimmed, "Total") || strings.Contains(trimmed, "Subang") || strings.Contains(trimmed, "Mahasiswa") || strings.Contains(trimmed, "Ketua") {
			break
		}

		// Need at least 8 lines for a complete entry
		if headerIndex+7 >= len(lines) {
			break
		}

		noStr := strings.TrimSpace(lines[headerIndex])
		kode := strings.TrimSpace(lines[headerIndex+1])
		nama := strings.TrimSpace(lines[headerIndex+2])
		sksStr := strings.TrimSpace(lines[headerIndex+3])
		kelas := strings.TrimSpace(lines[headerIndex+4])
		dosen := strings.TrimSpace(lines[headerIndex+5])
		jadwalDay := strings.TrimSpace(lines[headerIndex+6])
		jadwalTime := strings.TrimSpace(lines[headerIndex+7])

		// Parse nomor
		no, err := strconv.Atoi(noStr)
		if err != nil {
			// Not a valid entry, skip
			headerIndex++
			continue
		}

		// Parse SKS - handle non-breaking spaces and other chars
		sksStr = strings.ReplaceAll(sksStr, "\u00a0", "") // Remove non-breaking spaces
		sksStr = strings.TrimSpace(sksStr)
		sks, err := strconv.Atoi(sksStr)
		if err != nil {
			// Invalid SKS, skip
			headerIndex++
			continue
		}

		totalSKS += sks

		// Combine jadwal
		jadwal := strings.TrimSpace(jadwalDay + " " + jadwalTime)

		matkul := model.MataKuliah{
			No:     no,
			Kode:   kode,
			Nama:   nama,
			SKS:    sks,
			Kelas:  kelas,
			Dosen:  dosen,
			Jadwal: jadwal,
		}
		matkulList = append(matkulList, matkul)

		headerIndex += 8 // Move to next entry
	}

	if len(matkulList) == 0 {
		return nil, 0, errors.New("Daftar mata kuliah kosong atau tidak valid")
	}

	return matkulList, totalSKS, nil
}

// ExtractTanggalCetak mengekstrak tanggal cetak KRS
func (p *KRSParser) ExtractTanggalCetak() string {
	// Format 1: "Tanggal Cetak : 07/05/2026"
	tanggalPattern1 := regexp.MustCompile(`Tanggal\s*Cetak\s*:\s*(\d{2}/\d{2}/\d{4})`)
	if matches := tanggalPattern1.FindStringSubmatch(p.textContent); len(matches) >= 2 {
		return strings.TrimSpace(matches[1])
	}

	// Format 2: ", 09 Mei 2026" (format Indonesia dari PDF nyata)
	tanggalPattern2 := regexp.MustCompile(`,\s*(\d{1,2})\s+(Januari|Februari|Maret|April|Mei|Juni|Juli|Agustus|September|Oktober|November|Desember)\s+(\d{4})`)
	if matches := tanggalPattern2.FindStringSubmatch(p.textContent); len(matches) >= 4 {
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

// atoi converts string to int (simple implementation for date parsing)
func atoi(s string) int {
	var result int
	for _, c := range s {
		if c >= '0' && c <= '9' {
			result = result*10 + int(c-'0')
		}
	}
	return result
}
