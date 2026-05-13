# Observability Standards

This document defines the minimum observability requirements for the `be-lonceng_unman` service to ensure operational visibility in production.

## 1. Structured Logging
The application uses `log/slog` for structured JSON logging.

### Log Levels
- **INFO**: General operational events (e.g., "Server started", "KRS request completed").
- **WARN**: Non-critical issues that may require attention (e.g., "Invalid NIS format", "Cache write failed").
- **ERROR**: Critical failures that impact request completion (e.g., "Failed to process KRS from PDF", "Configuration load failure").

### Mandatory Log Fields
Every log entry should include:
- `time`: RFC3339 timestamp.
- `level`: Log level.
- `msg`: Descriptive message.
- `nis`: The student identifier (when applicable).
- `duration`: Request processing time (for completion logs).
- `error`: Detailed error message (for ERROR/WARN levels).

## 2. Health Checks
The service must provide a health check endpoint for orchestrators.

- **Endpoint**: `/health`
- **Method**: `GET`
- **Response**: `200 OK` with JSON body `{"status": "UP"}`.
- **Checks**:
    - Basic process availability.
    - Configuration validity.
    - Connectivity to critical dependencies (e.g., Cache service).

## 3. Metrics (Recommended)
For production scaling, the following metrics should be exported (e.g., via Prometheus):
- **Request Rate**: Total requests per second.
- **Error Rate**: Percentage of non-2xx responses.
- **Latency**: P50, P95, and P99 response times.
- **Cache Hit Ratio**: Ratio of cache hits vs misses.
- **PDF Processing Time**: Time taken to download and parse PDFs.

## 4. Tracing (Recommended)
Implement distributed tracing (e.g., OpenTelemetry) to track requests across:
`Frontend` $\rightarrow$ `API Gateway` $\rightarrow$ `be-lonceng_unman` $\rightarrow$ `LMS PDF Endpoint`.
