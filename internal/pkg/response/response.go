// Package response menyediakan helper terpusat untuk mengirim JSON response
// yang konsisten di seluruh aplikasi.
package response

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"be-lonceng_unman/internal/config"
)

// Meta berisi metadata standar yang disertakan dalam setiap response.
type Meta struct {
	Timestamp string `json:"timestamp"`
	Version   string `json:"version"`
	Service   string `json:"service,omitempty"`
	Cached    bool   `json:"cached"`
}

// Envelope adalah struktur JSON yang membungkus semua response API.
type Envelope struct {
	Status string `json:"status"`
	Data   any    `json:"data"`
	Meta   Meta   `json:"meta"`
}

// ErrorData adalah isi field "data" pada response error.
type ErrorData struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
}

// buildMeta membangun objek Meta dengan nilai dari config.
func buildMeta() Meta {
	return Meta{
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Version:   config.GetAppVersion(),
		Service:   config.GetServiceName(),
	}
}

// JSON mengirim response sukses dengan format baku.
// Parameter cached mengontrol apakah data berasal dari cache.
func JSON(w http.ResponseWriter, statusCode int, data any, cached bool) {
	meta := buildMeta()
	meta.Cached = cached

	env := Envelope{
		Status: "success",
		Data:   data,
		Meta:   meta,
	}

	w.Header().Set("Content-Type", "application/json")
	if statusCode == http.StatusOK {
		w.Header().Set("Cache-Control", "private, max-age=3600")
	}
	w.WriteHeader(statusCode)

	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(env); err != nil {
		slog.Error("response.JSON: gagal encode JSON", slog.String("error", err.Error()))
	}
}

// Error mengirim response error dengan format baku.
// Field "status" di root adalah "error" sehingga client dapat membedakannya dari response sukses.
func Error(w http.ResponseWriter, statusCode int, code string, message string) {
	env := Envelope{
		Status: "error",
		Data: ErrorData{
			Code:    code,
			Message: message,
		},
		Meta: buildMeta(),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(env); err != nil {
		slog.Error("response.Error: gagal encode JSON", slog.String("error", err.Error()))
	}
}
