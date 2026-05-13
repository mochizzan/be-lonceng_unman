package unit_test

import (
	"testing"

	"be-lonceng_unman/internal/model"
	"be-lonceng_unman/internal/parser"
	"be-lonceng_unman/test/fixtures"

	"github.com/stretchr/testify/assert"
)

// =============================================================================
// Parser - ExtractMahasiswa
// =============================================================================

func TestParser_ExtractMahasiswa_Valid(t *testing.T) {
	p := parser.NewKRSParser(fixtures.SampleKRSTextValid)
	mhs, err := p.ExtractMahasiswa()
	assert.NoError(t, err)
	assert.Equal(t, "2211700006", mhs.NPM)
	assert.Equal(t, "MOCHAMAD IZZAN FIRASYANSYAH", mhs.Nama)
	assert.Equal(t, "Sistem Informasi", mhs.ProgramStudi)
}

func TestParser_ExtractMahasiswa_MissingNPM(t *testing.T) {
	text := "Nama : JOHN DOE\nProgram Studi : Teknik Informatika\nTahun Ajaran : 2025/2026"
	p := parser.NewKRSParser(text)
	mhs, err := p.ExtractMahasiswa()
	assert.Error(t, err)
	assert.Nil(t, mhs)
	assert.Contains(t, err.Error(), "failed to extract mahasiswa data")
}

func TestParser_ExtractMahasiswa_MissingNama(t *testing.T) {
	text := "N P M : 2211700006\nProgram Studi : Teknik Informatika\nTahun Ajaran : 2025/2026"
	p := parser.NewKRSParser(text)
	mhs, err := p.ExtractMahasiswa()
	assert.Error(t, err)
	assert.Nil(t, mhs)
}

func TestParser_ExtractMahasiswa_MissingProdi(t *testing.T) {
	text := "Nama : JOHN DOE\nN P M : 2211700006\nTahun Ajaran : 2025/2026"
	p := parser.NewKRSParser(text)
	mhs, err := p.ExtractMahasiswa()
	assert.Error(t, err)
	assert.Nil(t, mhs)
}

// =============================================================================
// Parser - ExtractAcademicInfo
// =============================================================================

func TestParser_ExtractAcademicInfo_Valid(t *testing.T) {
	p := parser.NewKRSParser(fixtures.SampleKRSTextValid)
	tahun, semester, err := p.ExtractAcademicInfo()
	assert.NoError(t, err)
	assert.Equal(t, "2025/2026", tahun)
	assert.Equal(t, "GENAP", semester)
}

func TestParser_ExtractAcademicInfo_MissingTahun(t *testing.T) {
	text := "Semester : GENAP\nNama : TEST"
	p := parser.NewKRSParser(text)
	_, _, err := p.ExtractAcademicInfo()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "academic info")
}

func TestParser_ExtractAcademicInfo_MissingSemester(t *testing.T) {
	text := "Tahun Ajaran : 2025/2026\nNama : TEST"
	p := parser.NewKRSParser(text)
	_, _, err := p.ExtractAcademicInfo()
	assert.Error(t, err)
}

// =============================================================================
// Parser - ExtractMataKuliah
// =============================================================================

func TestParser_ExtractMataKuliah_Valid(t *testing.T) {
	p := parser.NewKRSParser(fixtures.SampleKRSTextValid)
	list, totalSKS, err := p.ExtractMataKuliah()
	assert.NoError(t, err)
	assert.Len(t, list, 2)
	assert.Equal(t, 9, totalSKS)
	assert.Equal(t, 1, list[0].No)
	assert.Equal(t, "SI40306", list[0].Kode)
	assert.Equal(t, "Tugas Akhir/Skripsi", list[0].Nama)
	assert.Equal(t, 6, list[0].SKS)
	assert.Equal(t, "Sabtu 08:00 s/d 09:40", list[0].Jadwal)
}

func TestParser_ExtractMataKuliah_Multiple(t *testing.T) {
	p := parser.NewKRSParser(fixtures.SampleKRSTextMultipleCourses)
	list, totalSKS, err := p.ExtractMataKuliah()
	assert.NoError(t, err)
	assert.Len(t, list, 4)
	assert.Equal(t, 12, totalSKS)
}

func TestParser_ExtractMataKuliah_Empty(t *testing.T) {
	p := parser.NewKRSParser(fixtures.SampleKRSTextNoCourses)
	list, _, err := p.ExtractMataKuliah()
	assert.Error(t, err)
	assert.Nil(t, list)
	assert.Contains(t, err.Error(), "Daftar mata kuliah kosong")
}

func TestParser_ExtractMataKuliah_NoHeader(t *testing.T) {
	text := "Nama : TEST\nN P M : 2211700006"
	p := parser.NewKRSParser(text)
	list, _, err := p.ExtractMataKuliah()
	assert.Error(t, err)
	assert.Nil(t, list)
	assert.Contains(t, err.Error(), "table not found")
}

// =============================================================================
// Parser - ExtractTanggalCetak
// NOTE: F-02 Bug - Regex tidak match format tanggal Indonesia dari PDF nyata
// Format di PDF: ", 09 Mei 2026" bukan "Tanggal Cetak : 09/05/2026"
// =============================================================================

func TestParser_ExtractTanggalCetak_NotExtracted(t *testing.T) {
	// F-02 Fix: Regex sekarang mendukung format tanggal Indonesia ", 09 Mei 2026"
	p := parser.NewKRSParser(fixtures.SampleKRSTextValid)
	tanggal := p.ExtractTanggalCetak()
	assert.Equal(t, "09/05/2026", tanggal) // Fixed: now extracts Indonesian format
}

func TestParser_ExtractTanggalCetak_WithExplicitLabel(t *testing.T) {
	// Format dengan label "Tanggal Cetak :" seharusnya bekerja
	text := "Nama : TEST\nTanggal Cetak : 07/05/2026"
	p := parser.NewKRSParser(text)
	tanggal := p.ExtractTanggalCetak()
	assert.Equal(t, "07/05/2026", tanggal)
}

func TestParser_ExtractTanggalCetak_Missing(t *testing.T) {
	text := "Nama : TEST\nN P M : 2211700006"
	p := parser.NewKRSParser(text)
	tanggal := p.ExtractTanggalCetak()
	assert.Equal(t, "", tanggal)
}

// =============================================================================
// Parser - Parse (Full)
// =============================================================================

func TestParser_Parse_FullValid(t *testing.T) {
	p := parser.NewKRSParser(fixtures.SampleKRSTextValid)
	result, err := p.Parse()
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "success", result.Status)
	assert.Equal(t, "2211700006", result.Mahasiswa.NPM)
	assert.Equal(t, "MOCHAMAD IZZAN FIRASYANSYAH", result.Mahasiswa.Nama)
	assert.Equal(t, "Sistem Informasi", result.Mahasiswa.ProgramStudi)
	assert.Equal(t, "2025/2026", result.TahunAjaran)
	assert.Equal(t, "GENAP", result.Semester)
	assert.Len(t, result.MataKuliah, 2)
	assert.Equal(t, 9, result.TotalSKS)
}

func TestParser_Parse_EmptyText(t *testing.T) {
	p := parser.NewKRSParser("")
	_, err := p.Parse()
	assert.Error(t, err)
}

// =============================================================================
// Model - Error Mapping
// NOTE: F-01 Bug - errors.Is tidak bekerja untuk custom struct pointer
// yang di-wrap dengan fmt.Errorf("...: %w", err). Beberapa test di bawah
// mendokumentasikan behavior aktual vs expected.
// =============================================================================

func TestErrorMapping_NISEmpty(t *testing.T) {
	err := model.NewNISEmptyError()
	status, code := model.MapErrorToHTTPStatus(err)
	assert.Equal(t, 400, status)
	assert.Equal(t, model.ErrCodeNISEmpty, code)
}

func TestErrorMapping_NISInvalid(t *testing.T) {
	err := model.NewNISInvalidError("abc")
	status, code := model.MapErrorToHTTPStatus(err)
	// F-01 Fix: errors.As now correctly maps NISInvalidError to 400
	assert.Equal(t, 400, status)
	assert.Equal(t, model.ErrCodeNISInvalid, code)
}

func TestErrorMapping_DataNotFound(t *testing.T) {
	err := model.NewDataNotFoundError("mahasiswa")
	status, code := model.MapErrorToHTTPStatus(err)
	// F-01 Fix: errors.As now correctly maps DataNotFoundError to 404
	assert.Equal(t, 404, status)
	assert.Equal(t, model.ErrCodeDataNotFound, code)
}

func TestErrorMapping_KRSEmpty(t *testing.T) {
	err := model.NewKRSEmptyError()
	status, code := model.MapErrorToHTTPStatus(err)
	assert.Equal(t, 404, status)
	assert.Equal(t, model.ErrCodeKRSEmpty, code)
}

func TestErrorMapping_PDFDownloadFailed(t *testing.T) {
	err := model.NewPDFDownloadFailedError("http://test", nil, 500)
	status, code := model.MapErrorToHTTPStatus(err)
	// F-01 Fix: errors.As now correctly maps PDFDownloadFailedError to 502
	assert.Equal(t, 502, status)
	assert.Equal(t, model.ErrCodePDFDownloadFailed, code)
}

func TestErrorMapping_InvalidResponse(t *testing.T) {
	err := model.NewInvalidResponseError("bad content")
	status, code := model.MapErrorToHTTPStatus(err)
	// F-01 Fix: errors.As now correctly maps InvalidResponseError to 502
	assert.Equal(t, 502, status)
	assert.Equal(t, model.ErrCodeInvalidResponse, code)
}

func TestErrorMapping_InternalServer(t *testing.T) {
	err := model.NewInternalServerError(nil)
	status, code := model.MapErrorToHTTPStatus(err)
	assert.Equal(t, 500, status)
	assert.Equal(t, model.ErrCodeInternalServer, code)
}

func TestErrorMapping_UnknownError(t *testing.T) {
	err := assert.AnError
	status, code := model.MapErrorToHTTPStatus(err)
	assert.Equal(t, 500, status)
	assert.Equal(t, model.ErrCodeInternalServer, code)
}
