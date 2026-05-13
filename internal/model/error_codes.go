package model

// Error codes for the application
const (
	ErrCodeNISEmpty           = "ERR_NIS_EMPTY"
	ErrCodeNISInvalid         = "ERR_NIS_INVALID"
	ErrCodeDataNotFound       = "ERR_DATA_NOT_FOUND"
	ErrCodeKRSEmpty           = "ERR_KRS_EMPTY"
	ErrCodePDFDownloadFailed  = "ERR_PDF_DOWNLOAD_FAILED"
	ErrCodePDFExternalUnavail = "ERR_PDF_EXTERNAL_UNAVAILABLE"
	ErrCodePDFTimeout         = "ERR_PDF_TIMEOUT"
	ErrCodePDFParseFailed     = "ERR_PDF_PARSE_FAILED"
	ErrCodeInvalidResponse    = "ERR_INVALID_RESPONSE"
	ErrCodeUnauthorized       = "ERR_UNAUTHORIZED"
	ErrCodeInternalServer     = "ERR_INTERNAL_SERVER"
	ErrCodeCacheReadFailed    = "ERR_CACHE_READ_FAILED"
	ErrCodeCacheWriteFailed   = "ERR_CACHE_WRITE_FAILED"
	ErrCodeRateLimit          = "ERR_RATE_LIMIT"
	ErrCodeMethodNotAllowed   = "ERR_METHOD_NOT_ALLOWED"
)
