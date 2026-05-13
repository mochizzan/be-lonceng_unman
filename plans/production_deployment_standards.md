# Production Deployment Standards

This document outlines the standards and requirements for deploying the `be-lonceng_unman` service to a production environment.

## 1. Environment Configuration
All configuration must be managed via environment variables. Use `.env.example` as a template.

### Required Variables
- `PORT`: Port the server listens on (default: 8080).
- `JWT_SECRET`: Strong secret key for JWT signing.
- `API_KEY`: Secret key for API key authentication.
- `SERVICE_NAME`: Name of the service (e.g., `be-lonceng_unman`).
- `APP_VERSION`: Semantic version of the application.
- `RATE_LIMIT_RPS`: Requests per second limit.
- `RATE_LIMIT_BURST`: Burst capacity for rate limiting.
- `ALLOWED_PDF_HOSTS`: Comma-separated list of allowed PDF domains.

## 2. Deployment Strategy
### Containerization (Docker)
The application should be deployed as a container.
- **Base Image**: Use a multi-stage build with `golang:alpine` for building and `alpine:latest` for the final image to minimize attack surface and image size.
- **User**: Run the application as a non-root user.
- **Health Checks**: Implement a `/health` endpoint for orchestrator (K8s/Docker Swarm) health checks.

### Resource Limits
- **CPU**: Set requests and limits based on load testing.
- **Memory**: Set limits to prevent OOM kills; monitor memory usage of the PDF parsing process.

## 3. Security Standards
- **TLS/SSL**: All traffic must be encrypted via HTTPS. Terminate TLS at the Load Balancer or Ingress Controller.
- **Secrets Management**: Do not commit `.env` files. Use Secret Management systems (e.g., Kubernetes Secrets, AWS Secrets Manager, HashiCorp Vault).
- **Network**: Restrict access to the application port to only the Load Balancer/Ingress.

## 4. Scalability & Availability
- **Horizontal Scaling**: The service is designed to be stateless. Scale horizontally by adding more replicas.
- **Caching**: The current `CacheService` is in-memory. For multi-replica deployments, transition to a distributed cache like Redis.
- **Graceful Shutdown**: The server implements graceful shutdown (SIGTERM/SIGINT) to ensure in-flight requests are completed.

## 5. Observability
- **Logging**: Use structured JSON logging (implemented via `slog`).
- **Monitoring**: Export metrics (e.g., Prometheus) for request rates, error rates, and latency.
- **Tracing**: Implement distributed tracing (e.g., OpenTelemetry) for complex request flows.
