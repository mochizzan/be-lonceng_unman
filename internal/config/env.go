package config

import (
	"errors"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

// Config holds all configuration for the application.
type Config struct {
	RunningPort     string
	JWTSecret       string
	APIKey          string
	PDFBaseURL      string
	AllowedPDFHosts []string
	RateLimitRPS    float64
	RateLimitBurst  int
	ServerVersion   string
	ServiceName     string
	OutputDir       string
	TrustProxy      bool
}

var cfg *Config

// LoadConfig loads configuration from environment variables and .env file.
func LoadConfig() error {
	// Load .env file from project root if present
	root := getProjectRoot()
	_ = godotenv.Load(filepath.Join(root, ".env"))

	outputDir := getEnv("OUTPUT_DIR", "output")
	// Make it absolute relative to project root
	if !filepath.IsAbs(outputDir) {
		root := getProjectRoot()
		outputDir = filepath.Join(root, outputDir)
	}

	trustProxy, _ := strconv.ParseBool(os.Getenv("TRUST_PROXY"))

	cfg = &Config{
		RunningPort:     getEnv("RUNNING_PORT", "8080"),
		JWTSecret:       getEnv("JWT_SECRET", ""),
		APIKey:          getEnv("API_KEY", ""),
		PDFBaseURL:      getEnv("PDF_BASE_URL", ""),
		AllowedPDFHosts: getAllowedPDFHosts(getEnv("PDF_BASE_URL", ""), getEnv("PDF_ALLOWED_HOSTS", "")),
		RateLimitRPS:    getEnvAsFloat("RATE_LIMIT_RPS", 10),
		RateLimitBurst:  getEnvAsInt("RATE_LIMIT_BURST", 30),
		ServerVersion:   getEnv("SERVER_VERSION", "1.0.0"),
		ServiceName:     getEnv("SERVICE_NAME", "be-lonceng_unman"),
		OutputDir:       outputDir,
		TrustProxy:      trustProxy,
	}

	// Validate required fields
	if cfg.JWTSecret == "" {
		return errors.New("JWT_SECRET is required")
	}
	if cfg.APIKey == "" {
		return errors.New("API_KEY is required")
	}
	if cfg.PDFBaseURL == "" {
		return errors.New("PDF_BASE_URL is required")
	}
	return nil
}

// getProjectRoot returns the project root directory.
// It walks up from the current working directory until it finds go.mod,
// or falls back to one level up from the current file's location.
func getProjectRoot() string {
	// Try walking up from CWD looking for go.mod
	dir, err := os.Getwd()
	if err == nil {
		for {
			if _, statErr := os.Stat(filepath.Join(dir, "go.mod")); statErr == nil {
				return dir
			}
			parent := filepath.Dir(dir)
			if parent == dir {
				break
			}
			dir = parent
		}
	}

	// Fallback: use the directory of this file (internal/config/) and go up 2 levels
	_, filename, _, _ := runtime.Caller(0)
	return filepath.Dir(filepath.Dir(filename))
}

// getEnv gets an environment variable or returns a default value.
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getEnvAsFloat gets an environment variable as float64 or returns a default value.
func getEnvAsFloat(key string, defaultValue float64) float64 {
	if value := os.Getenv(key); value != "" {
		if v, err := strconv.ParseFloat(value, 64); err == nil {
			return v
		}
	}
	return defaultValue
}

// getEnvAsInt gets an environment variable as int or returns a default value.
func getEnvAsInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if v, err := strconv.Atoi(value); err == nil {
			return v
		}
	}
	return defaultValue
}

// GetPDFBaseURL returns the base URL for PDF processing.
func GetPDFBaseURL() string {
	if cfg == nil {
		return ""
	}
	return cfg.PDFBaseURL
}

// GetRateLimitRPS returns the rate limit requests per second.
func GetRateLimitRPS() float64 {
	if cfg == nil {
		return 10
	}
	return cfg.RateLimitRPS
}

// GetRateLimitBurst returns the rate limit burst size.
func GetRateLimitBurst() int {
	if cfg == nil {
		return 30
	}
	return cfg.RateLimitBurst
}

// GetAppVersion returns the application version from environment.
func GetAppVersion() string {
	if cfg == nil {
		return "1.0.0"
	}
	if cfg.ServerVersion == "" {
		return "1.0.0"
	}
	return cfg.ServerVersion
}

// GetServiceName returns the service name from environment.
func GetServiceName() string {
	if cfg == nil {
		return "be-lonceng_unman"
	}
	if cfg.ServiceName == "" {
		return "be-lonceng_unman"
	}
	return cfg.ServiceName
}

// GetOutputDir returns the configured output directory path.
func GetOutputDir() string {
	if cfg == nil {
		return "output"
	}
	return cfg.OutputDir
}

// GetRunningPort returns the server running port from config.
func GetRunningPort() string {
	if cfg == nil {
		return "8080"
	}
	return cfg.RunningPort
}

// GetJWTSecret returns the JWT secret from config.
func GetJWTSecret() string {
	if cfg == nil {
		return ""
	}
	return cfg.JWTSecret
}

// GetAPIKey returns the API key from config.
func GetAPIKey() string {
	if cfg == nil {
		return ""
	}
	return cfg.APIKey
}

// GetTrustProxy returns whether to trust proxy headers for rate limiting.
func GetTrustProxy() bool {
	if cfg == nil {
		return false
	}
	return cfg.TrustProxy
}

// GetConfig returns the full Config struct.
func GetConfig() Config {
	if cfg == nil {
		return Config{
			RunningPort:     "8080",
			JWTSecret:       "",
			APIKey:          "",
			PDFBaseURL:      "",
			AllowedPDFHosts: []string{},
			RateLimitRPS:    10,
			RateLimitBurst:  30,
			ServerVersion:   "1.0.0",
			ServiceName:     "be-lonceng_unman",
			OutputDir:       "output",
			TrustProxy:      false,
		}
	}
	return *cfg
}

// EnsureOutputDir creates the output directory if it doesn't exist.
func EnsureOutputDir() error {
	return os.MkdirAll(GetOutputDir(), 0755)
}

// getAllowedPDFHosts extracts hostnames from a base URL and optional override env var.
// Returns hostnames only (not full URLs).
func getAllowedPDFHosts(baseURL, hostsStr string) []string {
	if hostsStr != "" {
		var hosts []string
		for _, h := range strings.Split(hostsStr, ",") {
			h = strings.TrimSpace(h)
			if h == "" {
				continue
			}
			// If someone provides a full URL, extract just the hostname
			if u, err := url.Parse(h); err == nil && u.Hostname() != "" {
				hosts = append(hosts, u.Hostname())
			} else {
				hosts = append(hosts, h)
			}
		}
		if len(hosts) > 0 {
			return hosts
		}
	}
	// Default: extract hostname from base URL
	if u, err := url.Parse(baseURL); err == nil && u.Hostname() != "" {
		return []string{u.Hostname()}
	}
	return []string{}
}

// GetAllowedPDFHosts returns a slice of allowed hostnames for PDF URLs.
// Hostnames are extracted from PDF_ALLOWED_HOSTS env var or derived from PDF_BASE_URL.
func GetAllowedPDFHosts() []string {
	if cfg == nil {
		return []string{}
	}
	return cfg.AllowedPDFHosts
}
