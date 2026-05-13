package model

// SuccessResponse wraps all successful API responses with a consistent format.
// Digunakan secara internal; layer presentation menggunakan pkg/response.
type SuccessResponse struct {
	Status string       `json:"status"`
	Data   interface{}  `json:"data"`
	Meta   ResponseMeta `json:"meta"`
}

// ResponseMeta contains metadata included in every API response.
type ResponseMeta struct {
	Timestamp string `json:"timestamp"`
	Version   string `json:"version"`
	Service   string `json:"service,omitempty"`
	Cached    bool   `json:"cached"`
	CacheAge  int64  `json:"cache_age_hours,omitempty"`
}
