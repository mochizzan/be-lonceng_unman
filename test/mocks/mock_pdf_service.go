package mocks

import (
	"context"

	"be-lonceng_unman/internal/model"
)

// MockPDFService adalah mock implementation dari PDFProcessor interface
// untuk digunakan dalam unit testing
type MockPDFService struct {
	ProcessResult *model.KRSResponse
	ProcessErr    error
	DownloadErr   error
	ExtractErr    error
}

// ProcessKRSFromURL mock implementation
func (m *MockPDFService) ProcessKRSFromURL(ctx context.Context, pdfURL string) (*model.KRSResponse, error) {
	if m.ProcessErr != nil {
		return nil, m.ProcessErr
	}
	return m.ProcessResult, nil
}

// NewMockPDFServiceSuccess membuat mock yang selalu sukses
func NewMockPDFServiceSuccess(npm, nama, prodi string) *MockPDFService {
	return &MockPDFService{
		ProcessResult: &model.KRSResponse{
			Status: "success",
			Mahasiswa: model.Mahasiswa{
				NPM:          npm,
				Nama:         nama,
				ProgramStudi: prodi,
			},
			TahunAjaran:  "2025/2026",
			Semester:     "GENAP",
			TanggalCetak: "09/05/2026",
			MataKuliah: []model.MataKuliah{
				{No: 1, Kode: "SI40306", Nama: "Tugas Akhir/Skripsi", SKS: 6, Kelas: "SI-8A", Dosen: "TIM DOSEN", Jadwal: "Sabtu 08:00 s/d 09:40"},
				{No: 2, Kode: "SI30201", Nama: "Pemrograman Web", SKS: 3, Kelas: "SI-8B", Dosen: "Dr. Budi", Jadwal: "Senin 10:00 s/d 11:40"},
			},
			TotalSKS: 9,
		},
	}
}

// NewMockPDFServiceError membuat mock yang selalu error
func NewMockPDFServiceError(err error) *MockPDFService {
	return &MockPDFService{
		ProcessErr: err,
	}
}

// NewMockPDFServiceEmptyData membuat mock yang return data kosong
func NewMockPDFServiceEmptyData() *MockPDFService {
	return &MockPDFService{
		ProcessResult: &model.KRSResponse{
			Status:     "success",
			Mahasiswa:  model.Mahasiswa{NPM: "2211700006", Nama: ""},
			MataKuliah: []model.MataKuliah{},
		},
	}
}
