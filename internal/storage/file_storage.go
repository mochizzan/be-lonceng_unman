package storage

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"
)

// FileStorage handles file operations for caching
type FileStorage struct {
	basePath string
}

// NewFileStorage creates a new file storage instance
func NewFileStorage(basePath string) *FileStorage {
	return &FileStorage{basePath: basePath}
}

// SavePDF saves PDF data to the file system
func (s *FileStorage) SavePDF(nis string, data []byte) error {
	filename := fmt.Sprintf("%s_%s.pdf", nis, time.Now().Format("20060102_150405"))
	pdfPath := filepath.Join(s.basePath, "storage", "raw", filename)

	// Create the full path if it doesn't exist
	dir := filepath.Dir(pdfPath)
	if dir != "" {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
	}

	// Save the file
	return os.WriteFile(pdfPath, data, 0644)
}

// SaveResponse saves the JSON response to the file system
func (s *FileStorage) SaveResponse(nis string, data []byte) error {
	filename := fmt.Sprintf("%s_%s.json", nis, time.Now().Format("20060102_150405"))
	responsePath := filepath.Join(s.basePath, "storage", "response", filename)

	// Create the full path if it doesn't exist
	dir := filepath.Dir(responsePath)
	if dir != "" {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
	}

	// Create the file
	return os.WriteFile(responsePath, data, 0644)
}

// SaveExtractedText saves extracted text to the file system
func (s *FileStorage) SaveExtractedText(nis string, data string) error {
	filename := fmt.Sprintf("%s_%s.txt", nis, time.Now().Format("20060102_150405"))
	extractedPath := filepath.Join(s.basePath, "storage", "extracted", filename)

	// Create the full path if it doesn't exist
	dir := filepath.Dir(extractedPath)
	if dir != "" {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
	}

	// Save the file
	return os.WriteFile(extractedPath, []byte(data), 0644)
}

// GetCachedResponse retrieves cached JSON response if it exists and is not expired
func (s *FileStorage) GetCachedResponse(nis string) ([]byte, error) {
	files, err := filepath.Glob(filepath.Join(s.basePath, "storage", "response", nis+"_*.json"))
	if err != nil {
		return nil, err
	}
	if len(files) == 0 {
		return nil, nil // No cached file found
	}

	// Sort by modification time (newest first)
	sort.Slice(files, func(i, j int) bool {
		infoI, errI := os.Stat(files[i])
		infoJ, errJ := os.Stat(files[j])
		if errI != nil || errJ != nil {
			return false
		}
		return infoI.ModTime().After(infoJ.ModTime())
	})

	// Check if the newest file is still valid (within 7 days)
	newestFile := files[0]
	info, err := os.Stat(newestFile)
	if err != nil {
		return nil, err
	}

	if time.Since(info.ModTime()) > 7*24*time.Hour {
		// File is expired, delete it
		os.Remove(newestFile)
		return nil, nil
	}

	// Read and return the file content
	return os.ReadFile(newestFile)
}

// GetCachedPDF retrieves cached PDF if it exists and is not expired
func (s *FileStorage) GetCachedPDF(nis string) ([]byte, error) {
	files, err := filepath.Glob(filepath.Join(s.basePath, "storage", "raw", nis+"_*.pdf"))
	if err != nil {
		return nil, err
	}
	if len(files) == 0 {
		return nil, nil // No cached file found
	}

	// Sort by modification time (newest first)
	sort.Slice(files, func(i, j int) bool {
		infoI, errI := os.Stat(files[i])
		infoJ, errJ := os.Stat(files[j])
		if errI != nil || errJ != nil {
			return false
		}
		return infoI.ModTime().After(infoJ.ModTime())
	})

	// Check if the newest file is still valid (within 7 days)
	newestFile := files[0]
	info, err := os.Stat(newestFile)
	if err != nil {
		return nil, err
	}

	if time.Since(info.ModTime()) > 7*24*time.Hour {
		// File is expired, delete it
		os.Remove(newestFile)
		return nil, nil
	}

	// Read and return the file content
	return os.ReadFile(newestFile)
}

// IsCacheValid checks if there's a valid cache for the given NIS
func (s *FileStorage) IsCacheValid(nis string) (bool, error) {
	// Check response cache
	cachedData, err := s.GetCachedResponse(nis)
	if err != nil {
		return false, err
	}
	return cachedData != nil, nil
}

// CleanupExpired removes expired cache files for the given NIS
func (s *FileStorage) CleanupExpired(nis string) error {
	// Clean response files
	responseFiles, err := filepath.Glob(filepath.Join(s.basePath, "storage", "response", nis+"_*.json"))
	if err != nil {
		return err
	}

	// Clean PDF files
	pdfFiles, err := filepath.Glob(filepath.Join(s.basePath, "storage", "raw", nis+"_*.pdf"))
	if err != nil {
		return err
	}

	// Clean extracted text files
	extractedFiles, err := filepath.Glob(filepath.Join(s.basePath, "storage", "extracted", nis+"_*.txt"))
	if err != nil {
		return err
	}

	now := time.Now()
	expired := now.Add(-7 * 24 * time.Hour)

	// Remove expired response files
	for _, file := range responseFiles {
		info, err := os.Stat(file)
		if err != nil {
			continue
		}
		if info.ModTime().Before(expired) {
			os.Remove(file)
		}
	}

	// Remove expired PDF files
	for _, file := range pdfFiles {
		info, err := os.Stat(file)
		if err != nil {
			continue
		}
		if info.ModTime().Before(expired) {
			os.Remove(file)
		}
	}

	// Remove expired extracted text files
	for _, file := range extractedFiles {
		info, err := os.Stat(file)
		if err != nil {
			continue
		}
		if info.ModTime().Before(expired) {
			os.Remove(file)
		}
	}

	return nil
}

// GetCacheAge returns the age of the newest cache file in hours
func (s *FileStorage) GetCacheAge(nis string) (int64, error) {
	// Check response cache for age
	files, err := filepath.Glob(filepath.Join(s.basePath, "storage", "response", nis+"_*.json"))
	if err != nil {
		return 0, err
	}
	if len(files) == 0 {
		return 0, nil // No cache found
	}

	// Find the newest file
	var newestTime time.Time
	for _, file := range files {
		info, err := os.Stat(file)
		if err != nil {
			continue
		}
		if newestTime.IsZero() || info.ModTime().After(newestTime) {
			newestTime = info.ModTime()
		}
	}

	if newestTime.IsZero() {
		return 0, nil
	}

	// Calculate age in hours
	age := time.Since(newestTime).Hours()
	return int64(age), nil
}
