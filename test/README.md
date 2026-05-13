# 📁 Test Directory - be-lonceng_unman

Direktori ini berisi semua test yang terstruktur dan rapi untuk audit menyeluruh.

## Struktur

```
test/
├── README.md                  # File ini - panduan test
├── unit/                      # Unit tests
│   ├── parser_test.go         # Test parser KRS
│   ├── validator_test.go      # Test validasi NIS/NPM
│   ├── cache_test.go          # Test cache service
│   └── error_test.go          # Test error mapping
├── integration/               # Integration tests
│   ├── pdf_flow_test.go       # Test PDF download -> parse flow
│   └── cache_flow_test.go     # Test cache hit/miss flow
├── e2e/                       # End-to-end tests
│   ├── auth_flow_test.go      # Test autentikasi penuh
│   ├── rate_limit_test.go     # Test rate limiting
│   └── krs_flow_test.go       # Test KRS request-response penuh
├── mocks/                     # Mock objects
│   └── mock_pdf_service.go    # Mock PDF service
└── fixtures/                  # Test data/fixtures
    └── sample_krs_text.go     # Sample KRS text untuk testing
```

## Cara Menjalankan

```bash
# Semua test
go test -v ./test/...

# Hanya unit test
go test -v ./test/unit/...

# Hanya integration test
go test -v ./test/integration/...

# Hanya e2e test (server harus running)
go test -v ./test/e2e/...

# Dengan coverage
go test -v -cover ./test/...

# Short mode (skip integration)
go test -v -short ./test/...
```
