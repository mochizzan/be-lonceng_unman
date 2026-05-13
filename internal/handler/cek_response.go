package handler

import (
	"net/http"
	"time"

	"be-lonceng_unman/internal/config"
	"be-lonceng_unman/internal/pkg/response"
)

// CekHandler menangani health check endpoint
func CekHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		response.Error(w, http.StatusMethodNotAllowed, "ERR_METHOD_NOT_ALLOWED", "method not allowed")
		return
	}

	response.JSON(w, http.StatusOK, map[string]string{
		"message":   "Server is running",
		"service":   config.GetServiceName(),
		"version":   config.GetAppVersion(),
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	}, false)
}
