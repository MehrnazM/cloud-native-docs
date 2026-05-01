# Cloud Native Docs

A cloud-native, microservices-based document processing system built with Go, featuring asynchronous processing, distributed tracing, and comprehensive observability.

## 🏗️ Architecture

This system consists of two main microservices:

- **API Service**: RESTful API for document management, built with Gin framework
- **Worker Service**: Asynchronous document processor with retry logic and monitoring

The services communicate via NATS JetStream for event-driven processing, with PostgreSQL as the data store.

```
┌─────────┐     ┌─────────────┐     ┌──────────────┐
│ Client  │────▶│  API (8080) │────▶│  PostgreSQL  │
└─────────┘     └─────────────┘     └──────────────┘
                       │
                       ▼
                ┌─────────────┐
                │    NATS     │
                │  JetStream  │
                └─────────────┘
                       │
                       ▼
                ┌─────────────┐     ┌──────────────┐
                │   Worker    │────▶│  PostgreSQL  │
                └─────────────┘     └──────────────┘
```

## ✨ Features

- 🚀 **RESTful API** for document creation and retrieval
- 🔐 **JWT Authentication** with bcrypt password hashing and protected endpoints
- ⚡ **Asynchronous Processing** with NATS JetStream
- 🔄 **Retry Mechanism** with configurable max retries
- 📊 **Metrics & Monitoring** with Prometheus
- 🔍 **Distributed Tracing** with OpenTelemetry and Jaeger
- 📈 **Autoscaling** with KEDA based on NATS consumer queue depth
- 🗄️ **Database Migrations** with migrate tool
- 🐳 **Containerized** with Docker and Docker Compose
- ☸️ **Kubernetes Ready** with deployment manifests
- ❤️ **Health Checks** for both API and Worker services
- 📝 **Structured Logging** with slog (JSON format)

## 🛠️ Tech Stack

- **Language**: Go 1.26+
- **Web Framework**: Gin
- **Database**: PostgreSQL 17
- **Message Broker**: NATS with JetStream
- **Authentication**: JWT (golang-jwt/jwt) + bcrypt
- **Autoscaling**: KEDA (Kubernetes Event-Driven Autoscaling)
- **Tracing**: OpenTelemetry + Jaeger
- **Metrics**: Prometheus
- **Container**: Docker & Docker Compose
- **Orchestration**: Kubernetes (minikube)
- **Migrations**: golang-migrate

## 📋 Prerequisites

- Go 1.26 or later
- Docker and Docker Compose
- (Optional) Kubernetes cluster for K8s deployment
- (Optional) kubectl for K8s management

## 🚀 Quick Start

### Using Docker Compose

1. **Clone the repository**
   ```bash
   git clone https://github.com/MehrnazM/cloud-native-docs.git
   cd cloud-native-docs
   ```

2. **Create environment file**
   ```bash
   cat > .env << EOF
   POSTGRES_USER=postgres
   POSTGRES_PASSWORD=postgres
   POSTGRES_DB=documents
   POSTGRES_HOST=localhost
   POSTGRES_PORT=5432
   NATS_URL=nats://localhost:4222
   JAEGER_COLLECTOR=localhost:4318
   SLOG_LEVEL=-4
   EOF
   ```

3. **Start all services**
   ```bash
   docker-compose up -d
   ```

4. **Verify services are running**
   ```bash
   docker-compose ps
   ```

The API will be available at `http://localhost:8080`

### Using Kubernetes

1. **Apply base configurations**
   ```bash
   kubectl apply -f k8s/base/
   ```

2. **Deploy dependencies**
   ```bash
   kubectl apply -f k8s/dependencies/
   ```

3. **Deploy API and Worker**
   ```bash
   kubectl apply -f k8s/api.yaml
   kubectl apply -f k8s/worker.yaml
   ```

4. **Deploy autoscaling**
   ```bash
   # Install KEDA (requires Helm)
   helm repo add kedacore https://kedacore.github.io/charts
   helm repo update
   helm install keda kedacore/keda --namespace keda --create-namespace

   # Apply the NATS-based scaler
   kubectl apply -f k8s/worker-keda-scaler.yaml
   ```

5. **Deploy monitoring (optional)**
   ```bash
   kubectl apply -f k8s/prometheus.yaml
   ```

## 📡 API Endpoints

### Authentication (public)

#### Register
```bash
POST /api/v1/auth/register
Content-Type: application/json

{
  "email": "user@example.com",
  "password": "yourpassword"
}
```

#### Login
```bash
POST /api/v1/auth/login
Content-Type: application/json

{
  "email": "user@example.com",
  "password": "yourpassword"
}
```

**Response:**
```json
{
  "message": "user logged in successfully",
  "data": {
    "token": "<jwt>"
  }
}
```

### Documents (protected — requires `Authorization: Bearer <token>`)

#### Create Document
```bash
POST /api/v1/documents
Authorization: Bearer <token>
Content-Type: application/json

{
  "name": "my-document"
}
```

**Response:**
```json
{
  "message": "Document created successfully",
  "data": {
    "id": "550e8400-e29b-41d4-a716-446655440000"
  }
}
```

#### Get Document by ID
```bash
GET /api/v1/documents/{id}
Authorization: Bearer <token>
```

**Response:**
```json
{
  "message": "Document retrieved successfully",
  "data": {
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "name": "my-document",
    "status": "done",
    "retryCount": 0,
    "maxRetries": 3,
    "createdAt": "2026-04-29T10:30:00Z",
    "updatedAt": "2026-04-29T10:30:05Z"
  }
}
```

### Health Check (public)
```bash
GET /api/v1/health
```

## 📁 Project Structure

```
cloud-native-docs/
├── cmd/
│   ├── api/           # API service entry point
│   └── worker/        # Worker service entry point
├── internal/
│   ├── http/              # HTTP server and routing
│   │   ├── handler/       # HTTP handlers (documents, auth, admin)
│   │   └── middleware/    # JWT auth middleware
│   ├── messaging/         # NATS JetStream client
│   ├── model/             # Domain models (Document, User)
│   ├── repository/        # Data access layer
│   ├── service/           # Business logic (document, auth)
│   └── worker/            # Document processor
├── shared/
│   ├── events/        # Event definitions
│   └── util/          # Utility functions
├── migrations/        # Database migration files
├── k8s/               # Kubernetes manifests
│   ├── base/          # ConfigMaps and Secrets
│   ├── dependencies/  # PostgreSQL, NATS, Jaeger
│   ├── worker-keda-scaler.yaml  # KEDA ScaledObject for NATS lag-based autoscaling
│   └── worker-hpa.yaml          # CPU-based HPA (reference, superseded by KEDA)
├── docker-compose.yaml
├── Dockerfile.api
├── Dockerfile.worker
└── go.mod
```

## 🔍 Observability

### Metrics (Prometheus)

The worker service exposes Prometheus metrics at `/metrics`:

- `documents_processed_total`: Total number of successfully processed documents
- `documents_failed_total`: Total number of failed documents
- `documents_in_processing`: Current number of documents being processed

### Distributed Tracing (Jaeger)

Both services emit traces to Jaeger. Access the Jaeger UI to view:
- Request traces across API and Worker
- Service dependencies
- Performance bottlenecks

### Logging

All services use structured JSON logging (slog) with configurable log levels via the `SLOG_LEVEL` environment variable.

## 📈 Autoscaling

The worker scales automatically based on NATS JetStream consumer lag using [KEDA](https://keda.sh/):

| Pending messages | Worker replicas |
|-----------------|-----------------|
| 0               | 1 (minimum)     |
| 10              | 2               |
| 30              | 3               |
| 50+             | 5 (maximum)     |

Replicas scale down after a 30-second cooldown once the queue drains. This approach is more accurate than CPU-based scaling for I/O-bound queue workers.

To check current scaling status:
```bash
kubectl get scaledobject worker-scaledobject
kubectl get hpa  # KEDA manages this automatically
```

## 🗄️ Database Migrations

Migrations are automatically applied during startup via the `migrate` service in Docker Compose, or the `migration-job` in Kubernetes.

To manually run migrations:
```bash
migrate -path ./migrations \
  -database "postgresql://postgres:postgres@localhost:5432/documents?sslmode=disable" \
  up
```

## 📊 Document Processing Flow

1. Client creates a document via POST `/api/v1/documents`
2. API service stores the document in PostgreSQL with `pending` status
3. API publishes `documents.created` event to NATS JetStream
4. Worker consumes the event and processes the document
5. Worker updates the document status to `done` or `failed`
6. If processing fails, the worker retries up to max retries (default: 3)
7. Metrics and traces are collected throughout the process

## 🔧 Configuration

Configuration is managed via environment variables:

| Variable | Description | Default |
|----------|-------------|---------|
| `POSTGRES_HOST` | PostgreSQL host | `localhost` |
| `POSTGRES_PORT` | PostgreSQL port | `5432` |
| `POSTGRES_USER` | PostgreSQL user | `postgres` |
| `POSTGRES_PASSWORD` | PostgreSQL password | `postgres` |
| `POSTGRES_DB` | Database name | `documents` |
| `NATS_URL` | NATS server URL | `nats://localhost:4222` |
| `JAEGER_COLLECTOR` | Jaeger collector endpoint | `localhost:4318` |
| `SLOG_LEVEL` | Log level (-4=DEBUG, 0=INFO, 4=WARN, 8=ERROR) | `-4` |
| `JWT_SECRET` | Secret key for signing JWTs (min 32 bytes) | required |



## 🤝 Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add some amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## 📝 License

This project is open source and available under the [MIT License](LICENSE).

## 👤 Author

**Mehrnaz M**
- GitHub: [@MehrnazM](https://github.com/MehrnazM)

## 🙏 Acknowledgments

- [Gin Web Framework](https://github.com/gin-gonic/gin)
- [NATS JetStream](https://docs.nats.io/nats-concepts/jetstream)
- [OpenTelemetry](https://opentelemetry.io/)
- [Prometheus](https://prometheus.io/)
