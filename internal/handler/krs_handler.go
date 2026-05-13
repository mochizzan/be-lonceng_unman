package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"be-lonceng_unman/internal/config"
	"be-lonceng_unman/internal/middleware"
	"be-lonceng_unman/internal/model"
	"be-lonceng_unman/internal/pkg/response"
	"be-lonceng_unman/internal/services"
	"be-lonceng_unman/internal/storage"
)

// PDFProcessor defines the interface for PDF processing operations needed by the handler.
// This allows mocking the PDF service in tests.
type PDFProcessor interface {
	ProcessKRSFromURL(ctx context.Context, pdfURL string) (*model.KRSResponse, error)
}

// KRSHandler menangani request untuk endpoint KRS
type KRSHandler struct {
	pdfService   PDFProcessor
	cacheService *services.CacheService
	fileStorage  *storage.FileStorage
	log          *slog.Logger
}

// NewKRSHandler membuat instance KRSHandler baru
func NewKRSHandler(pdfService PDFProcessor, cacheService *services.CacheService, fileStorage *storage.FileStorage, log *slog.Logger) *KRSHandler {
	return &KRSHandler{
		pdfService:   pdfService,
		cacheService: cacheService,
		fileStorage:  fileStorage,
		log:          log,
	}
}

// GetKRSByNIS menangani request untuk mendapatkan KRS berdasarkan NPM (disebut NIS di endpoint lama).
func (h *KRSHandler) GetKRSByNIS(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	startTime := time.Now()
	requestID := middleware.GetRequestID(ctx)

	// Extract NPM dari JWT claims (context key menggunakan nama "npm")
	npm, ok := middleware.GetUserNPM(ctx)
	if !ok || npm == "" {
		h.log.ErrorContext(ctx, "NPM not found in JWT claims",
			slog.String("request_id", requestID),
		)
		response.Error(w, http.StatusUnauthorized, model.ErrCodeUnauthorized, "Authentication required")
		return
	}

	// Validasi format NPM (10 digit angka)
	if err := services.ValidateNIS(npm); err != nil {
		h.log.WarnContext(ctx, "Invalid NPM format",
			slog.String("npm", npm),
			slog.String("request_id", requestID),
		)
		response.Error(w, http.StatusBadRequest, model.ErrCodeNISInvalid, "Invalid NPM format: must be 10 digits")
		return
	}

	// Cek cache (L1 & L2)
	cachedResponse, err := h.getCachedKRS(ctx, npm)
	if err == nil && cachedResponse != nil {
		// getCachedKRS sudah log L1 HIT atau L2 HIT
		response.JSON(w, http.StatusOK, cachedResponse, true)
		return
	}

	// Full miss — proses PDF
	pdfStartTime := time.Now()
	h.log.InfoContext(ctx, "[PDF PROCESS] Starting PDF processing for KRS",
		slog.String("npm", npm),
		slog.String("request_id", requestID),
	)

	pdfURL := config.GetPDFBaseURL() + "?nis=" + npm

	krsResponse, err := h.pdfService.ProcessKRSFromURL(ctx, pdfURL)
	if err != nil {
		h.log.ErrorContext(ctx, "[PDF FAILURE] Failed to process KRS PDF",
			slog.String("npm", npm),
			slog.String("error", err.Error()),
			slog.String("request_id", requestID),
		)
		statusCode, errorCode := model.MapErrorToHTTPStatus(err)
		response.Error(w, statusCode, errorCode, "Failed to process KRS: "+err.Error())
		return
	}

	// Log PDF success
	h.log.InfoContext(ctx, "[PDF SUCCESS] KRS processed and cached to L1 & L2",
		slog.String("npm", npm),
		slog.Duration("duration", time.Since(pdfStartTime)),
		slog.Int("courses", len(krsResponse.MataKuliah)),
	)

	// Simpan ke cache in-memory
	if err := h.cacheService.Set(ctx, "krs:"+npm, krsResponse, 24*time.Hour); err != nil {
		h.log.WarnContext(ctx, "[CACHE SET ERROR] Failed to set L1 cache",
			slog.String("npm", npm),
			slog.String("error", err.Error()),
			slog.String("request_id", requestID),
		)
	}

	// Simpan ke file storage (persistent cache)
	if err := h.storeKRSToFileStorage(npm, krsResponse); err != nil {
		h.log.WarnContext(ctx, "[L2 STORE ERROR] Failed to store KRS to L2 storage",
			slog.String("npm", npm),
			slog.String("error", err.Error()),
			slog.String("request_id", requestID),
		)
		// Non-critical — lanjutkan response
	}

	h.log.InfoContext(ctx, "[REQUEST COMPLETE] KRS request completed successfully",
		slog.String("npm", npm),
		slog.Duration("duration", time.Since(startTime)),
		slog.Int("mata_kuliah_count", len(krsResponse.MataKuliah)),
		slog.String("request_id", requestID),
	)

	response.JSON(w, http.StatusOK, krsResponse, false)
}

// getCachedKRS mengambil KRS dari in-memory cache (L1).
// Jika L1 miss, coba L2 (file storage). Jika L2 hit, promosi ke L1.
// Jika keduanya miss, return error untuk trigger PDF processing.
func (h *KRSHandler) getCachedKRS(ctx context.Context, npm string) (*model.KRSResponse, error) {
	// L1: Check in-memory cache
	startTime := time.Now()
	l1Data, err := h.cacheService.Get(ctx, "krs:"+npm)
	if err == nil && l1Data != nil {
		h.log.InfoContext(ctx, "[L1 HIT] KRS data retrieved from memory",
			slog.String("npm", npm),
			slog.Duration("duration", time.Since(startTime)),
		)
		return l1Data, nil
	}
	h.log.DebugContext(ctx, "[L1 MISS] No data in in-memory cache",
		slog.String("npm", npm),
		slog.Duration("duration", time.Since(startTime)),
	)

	// L2: Check file storage cache
	startTime = time.Now()
	l2Data, err := h.fileStorage.GetCachedResponse(npm)
	if err != nil {
		h.log.WarnContext(ctx, "[L2 ERROR] Error reading L2 cache file",
			slog.String("npm", npm),
			slog.String("error", err.Error()),
			slog.Duration("duration", time.Since(startTime)),
		)
		// Error saat membaca file, anggap miss dan lanjut ke PDF
		return nil, fmt.Errorf("full cache miss")
	}

	if l2Data == nil {
		h.log.InfoContext(ctx, "[FULL MISS] No cache found in L1/L2, processing PDF",
			slog.String("npm", npm),
		)
		return nil, fmt.Errorf("full cache miss")
	}

	// L2 Hit: Unmarshal dan promosi ke L1
	var l2Response model.KRSResponse
	unmarshalStartTime := time.Now()
	if err := json.Unmarshal(l2Data, &l2Response); err != nil {
		h.log.WarnContext(ctx, "[L2 INVALID] Cache file found but invalid/expired, processing PDF",
			slog.String("npm", npm),
			slog.String("error", err.Error()),
			slog.Duration("duration", time.Since(unmarshalStartTime)),
		)
		return nil, fmt.Errorf("full cache miss")
	}

	// Promosi ke L1
	setStartTime := time.Now()
	if err := h.cacheService.Set(ctx, "krs:"+npm, &l2Response, 24*time.Hour); err != nil {
		h.log.WarnContext(ctx, "[L2 PROMOTE ERROR] Failed to promote L2 to L1",
			slog.String("npm", npm),
			slog.String("error", err.Error()),
			slog.Duration("duration", time.Since(setStartTime)),
		)
		// Non-critical: tetap return data meski tidak bisa promosi ke L1
	} else {
		h.log.DebugContext(ctx, "[L2 PROMOTE SUCCESS] Data promoted from L2 to L1",
			slog.String("npm", npm),
			slog.Duration("duration", time.Since(setStartTime)),
		)
	}

	h.log.InfoContext(ctx, "[L2 HIT] KRS data retrieved from disk, promoting to L1",
		slog.String("npm", npm),
		slog.Duration("duration", time.Since(startTime)),
	)

	return &l2Response, nil
}

// storeKRSToFileStorage menyimpan data KRS ke file storage.
func (h *KRSHandler) storeKRSToFileStorage(npm string, krsResponse *model.KRSResponse) error {
	jsonData, err := json.Marshal(krsResponse)
	if err != nil {
		return fmt.Errorf("failed to marshal KRS response: %w", err)
	}
	if err := h.fileStorage.SaveResponse(npm, jsonData); err != nil {
		return fmt.Errorf("failed to save KRS response to storage: %w", err)
	}
	return nil
}
