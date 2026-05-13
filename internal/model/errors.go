package model

import (
	"context"
	"errors"
	"fmt"
	"net/http"
)

// Custom error types for better error classification

// NISEmptyError represents an empty NIS error
type NISEmptyError struct{}

func (e *NISEmptyError) Error() string {
	return "NIS cannot be empty"
}

// NISInvalidError represents an invalid NIS format error
type NISInvalidError struct {
	Value string
}

func (e *NISInvalidError) Error() string {
	return fmt.Sprintf("NIS must be 10 digits: %s", e.Value)
}

// DataNotFoundError represents data not found error
type DataNotFoundError struct {
	Resource string
}

func (e *DataNotFoundError) Error() string {
	return fmt.Sprintf("%s not found", e.Resource)
}

// KRSEmptyError represents empty KRS data error
type KRSEmptyError struct{}

func (e *KRSEmptyError) Error() string {
	return "Daftar mata kuliah kosong atau tidak valid"
}

// PDFDownloadFailedError represents PDF download failure
type PDFDownloadFailedError struct {
	URL  string
	Err  error
	Code int // HTTP status code
}

func (e *PDFDownloadFailedError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("failed to download PDF from %s: %v", e.URL, e.Err)
	}
	return fmt.Sprintf("failed to download PDF from %s: HTTP %d", e.URL, e.Code)
}

func (e *PDFDownloadFailedError) Unwrap() error {
	return e.Err
}

// PDFExternalUnavailableError represents external PDF service unavailable
type PDFExternalUnavailableError struct {
	URL string
}

func (e *PDFExternalUnavailableError) Error() string {
	return fmt.Sprintf("external PDF service unavailable: %s", e.URL)
}

// PDFTimeoutError represents PDF request timeout
type PDFTimeoutError struct {
	URL string
}

func (e *PDFTimeoutError) Error() string {
	return fmt.Sprintf("PDF request timeout: %s", e.URL)
}

// InvalidResponseError represents invalid PDF response
type InvalidResponseError struct {
	Reason string
}

func (e *InvalidResponseError) Error() string {
	return fmt.Sprintf("invalid PDF response: %s", e.Reason)
}

// PDFParseFailedError represents PDF parsing failure
type PDFParseFailedError struct {
	Reason string
}

func (e *PDFParseFailedError) Error() string {
	return fmt.Sprintf("failed to parse PDF: %s", e.Reason)
}

// InternalServerError represents generic internal server error
type InternalServerError struct {
	Err error
}

func (e *InternalServerError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("internal server error: %v", e.Err)
	}
	return "internal server error"
}

func (e *InternalServerError) Unwrap() error {
	return e.Err
}

// CacheReadFailedError represents cache read failure
type CacheReadFailedError struct {
	Key string
	Err error
}

func (e *CacheReadFailedError) Error() string {
	return fmt.Sprintf("failed to read from cache (%s): %v", e.Key, e.Err)
}

// CacheWriteFailedError represents cache write failure
type CacheWriteFailedError struct {
	Key string
	Err error
}

func (e *CacheWriteFailedError) Error() string {
	return fmt.Sprintf("failed to write to cache (%s): %v", e.Key, e.Err)
}

// MapErrorToHTTPStatus maps custom errors to appropriate HTTP status codes and error codes
func MapErrorToHTTPStatus(err error) (int, string) {
	// Use errors.As instead of errors.Is to properly handle wrapped errors
	// errors.Is compares pointers, which fails when errors are wrapped with fmt.Errorf("%w")
	// errors.As traverses the error chain and matches the target type
	var nisEmpty *NISEmptyError
	var nisInvalid *NISInvalidError
	var dataNotFound *DataNotFoundError
	var krsEmpty *KRSEmptyError
	var pdfDownloadFailed *PDFDownloadFailedError
	var pdfExternalUnavailable *PDFExternalUnavailableError
	var pdfTimeout *PDFTimeoutError
	var invalidResp *InvalidResponseError
	var pdfParseFailed *PDFParseFailedError
	var internalErr *InternalServerError
	var cacheReadFailed *CacheReadFailedError
	var cacheWriteFailed *CacheWriteFailedError

	switch {
	case errors.As(err, &nisEmpty):
		return http.StatusBadRequest, ErrCodeNISEmpty
	case errors.As(err, &nisInvalid):
		return http.StatusBadRequest, ErrCodeNISInvalid
	case errors.As(err, &dataNotFound):
		return http.StatusNotFound, ErrCodeDataNotFound
	case errors.As(err, &krsEmpty):
		return http.StatusNotFound, ErrCodeKRSEmpty
	case errors.As(err, &pdfDownloadFailed):
		return http.StatusBadGateway, ErrCodePDFDownloadFailed
	case errors.As(err, &pdfExternalUnavailable):
		return http.StatusServiceUnavailable, ErrCodePDFExternalUnavail
	case errors.As(err, &pdfTimeout):
		return http.StatusGatewayTimeout, ErrCodePDFTimeout
	case errors.As(err, &invalidResp):
		return http.StatusBadGateway, ErrCodeInvalidResponse
	case errors.As(err, &pdfParseFailed):
		return http.StatusInternalServerError, ErrCodePDFParseFailed
	case errors.As(err, &internalErr):
		return http.StatusInternalServerError, ErrCodeInternalServer
	case errors.As(err, &cacheReadFailed):
		return http.StatusInternalServerError, ErrCodeCacheReadFailed
	case errors.As(err, &cacheWriteFailed):
		return http.StatusInternalServerError, ErrCodeCacheWriteFailed
	case errors.Is(err, context.DeadlineExceeded):
		return http.StatusGatewayTimeout, ErrCodePDFTimeout
	default:
		// Default to internal server error
		return http.StatusInternalServerError, ErrCodeInternalServer
	}
}

// Helper functions to create custom errors

func NewNISEmptyError() error {
	return &NISEmptyError{}
}

func NewNISInvalidError(value string) error {
	return &NISInvalidError{Value: value}
}

func NewDataNotFoundError(resource string) error {
	return &DataNotFoundError{Resource: resource}
}

func NewKRSEmptyError() error {
	return &KRSEmptyError{}
}

func NewPDFDownloadFailedError(url string, err error, code int) error {
	return &PDFDownloadFailedError{URL: url, Err: err, Code: code}
}

func NewPDFExternalUnavailableError(url string) error {
	return &PDFExternalUnavailableError{URL: url}
}

func NewPDFTimeoutError(url string) error {
	return &PDFTimeoutError{URL: url}
}

func NewInvalidResponseError(reason string) error {
	return &InvalidResponseError{Reason: reason}
}

func NewPDFParseFailedError(reason string) error {
	return &PDFParseFailedError{Reason: reason}
}

func NewInternalServerError(err error) error {
	return &InternalServerError{Err: err}
}

func NewCacheReadFailedError(key string, err error) error {
	return &CacheReadFailedError{Key: key, Err: err}
}

func NewCacheWriteFailedError(key string, err error) error {
	return &CacheWriteFailedError{Key: key, Err: err}
}
