package model

// ErrorResponse represents a structured error response
type ErrorResponse struct {
	Status    string `json:"status"`
	Code      string `json:"code"`
	Message   string `json:"message"`
	Details   string `json:"details,omitempty"`
	Timestamp string `json:"timestamp"`
}

// ErrorDetail holds details for error responses
type ErrorDetail struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
}

// Error returns the error details with timestamp
func (e *ErrorResponse) Error() string {
	return e.Message
}
