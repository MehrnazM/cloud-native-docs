# Cloud Native Docs

A cloud-native, microservices-based document processing system built with Go, featuring asynchronous processing, distributed tracing, dead letter queue handling, and comprehensive observability.

## 🎯 Project Focus

This project is intentionally focused on **distributed systems engineering**, not business logic. The document API is deliberately simple — the goal is to demonstrate:

- **Distributed systems design**: event-driven architecture, async processing, message guarantees
- **Reliability patterns**: retry with backoff, dead letter queues, idempotency, concurrency control
- **Observability**: structured logging, Prometheus metrics, distributed tracing with Jaeger, Grafana dashboards
- **Container orchestration**: Docker, Kubernetes (local with minikube, cloud with GKE Autopilot)
- **Cloud-native infrastructure**: managed database (Cloud SQL), container registry (Artifact Registry), cloud-native autoscaling (KEDA)

The simplicity of the API surface is a deliberate trade-off — it keeps the focus on the infrastructure and system design layers that matter most in production backend engineering.

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
- 🔄 **Retry Mechanism** with configurable max retries and exponential backoff
- ☠️ **Dead Letter Queue** for permanently failed documents with admin inspection endpoint
- 📊 **Metrics & Monitoring** with Prometheus (counters, gauges, and duration histograms)
- 📉 **Grafana Dashboards** provisioned as code for real-time system visibility
- 🔍 **Distributed Tracing** with OpenTelemetry and Jaeger (document-level span attributes)
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
- **Metrics**: Prometheus + Grafana
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

### Using Kubernetes (local — minikube)

1. **Apply base configurations**
   ```bash
   kubectl apply -f k8s/local/base/
   ```

2. **Deploy dependencies**
   ```bash
   kubectl apply -f k8s/local/dependencies/
   ```

3. **Deploy API and Worker**
   ```bash
   kubectl apply -f k8s/local/api.yaml
   kubectl apply -f k8s/local/worker.yaml
   ```

4. **Deploy autoscaling**
   ```bash
   # Install KEDA (requires Helm)
   helm repo add kedacore https://kedacore.github.io/charts
   helm repo update
   helm install keda kedacore/keda --namespace keda --create-namespace

   # Apply the NATS-based scaler
   kubectl apply -f k8s/local/worker-keda-scaler.yaml
   ```

5. **Deploy monitoring**
   ```bash
   kubectl apply -f k8s/local/grafana.yaml
   ```

### Using Kubernetes (GKE — Google Cloud)

Prerequisites: GCP project with billing enabled, `gcloud` CLI, `kubectl`, and `helm` installed.

1. **Create GKE Autopilot cluster**
   ```bash
   gcloud container clusters create-auto <cluster-name> --region=us-west1
   gcloud container clusters get-credentials <cluster-name> --region=us-west1
   ```

2. **Create Cloud SQL PostgreSQL instance** via GCP Console (SQL → Create Instance → PostgreSQL), then create the `documents` database.

3. **Run database migrations** using Cloud SQL Auth Proxy:
   ```bash
   ./cloud-sql-proxy <CONNECTION_NAME> &
   ./migrate -path ./migrations \
     -database "postgresql://postgres:<PASSWORD>@127.0.0.1:5432/documents?sslmode=disable" up
   ```

4. **Push images to Artifact Registry**
   ```bash
   gcloud artifacts repositories create cloud-native-docs --repository-format=docker --location=us-west1
   gcloud builds submit --tag us-west1-docker.pkg.dev/<PROJECT_ID>/cloud-native-docs/api:latest .
   gcloud builds submit --tag us-west1-docker.pkg.dev/<PROJECT_ID>/cloud-native-docs/worker:latest -f Dockerfile.worker .
   ```

5. **Create Kubernetes secrets**
   ```bash
   kubectl create secret generic postgres-secret \
     --from-literal=POSTGRES_USER=postgres \
     --from-literal=POSTGRES_PASSWORD=<PASSWORD> \
     --from-literal=POSTGRES_DB=documents

   kubectl create secret generic api-secret \
     --from-literal=JWT_SECRET=<JWT_SECRET>

   kubectl create secret generic cloudsql-sa-key \
     --from-file=sa-key.json=sa-key.json
   ```

6. **Install KEDA**
   ```bash
   helm repo add kedacore https://kedacore.github.io/charts
   helm repo update
   helm install keda kedacore/keda --namespace keda --create-namespace
   ```

7. **Deploy everything**
   ```bash
   kubectl apply -f k8s/GKE/base/
   kubectl apply -f k8s/GKE/dependencies/
   kubectl apply -f k8s/GKE/api.yaml
   kubectl apply -f k8s/GKE/worker.yaml
   kubectl apply -f k8s/GKE/grafana.yaml
   kubectl apply -f k8s/GKE/worker-keda-scaler.yaml
   ```

The API will be available at the LoadBalancer external IP:
```bash
kubectl get service api-svc
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

### Admin (protected — requires `Authorization: Bearer <token>`)

#### Get DLQ Messages
```bash
GET /api/v1/admin/dlq?limit=50
Authorization: Bearer <token>
```

**Response:**
```json
{
  "message": "DLQ messages retrieved successfully",
  "data": {
    "count": 1,
    "messages": [
      {
        "originalEvent": {
          "id": "550e8400-e29b-41d4-a716-446655440000",
          "name": "my-document"
        },
        "failureReason": "reached max retry",
        "retryCount": 3,
        "failedAt": "2026-05-08T10:30:00Z"
      }
    ]
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
├── k8s/
│   ├── local/                   # Minikube local development manifests
│   │   ├── base/                # ConfigMaps and Secrets
│   │   ├── dependencies/        # PostgreSQL, NATS, Jaeger, Prometheus
│   │   ├── api.yaml
│   │   ├── worker.yaml
│   │   ├── grafana.yaml
│   │   ├── worker-keda-scaler.yaml
│   │   └── worker-hpa.yaml
│   └── GKE/                     # Google Kubernetes Engine manifests
│       ├── base/                # Prometheus config
│       ├── dependencies/        # NATS, Jaeger, Prometheus, migration job
│       ├── api.yaml             # API + Cloud SQL Auth Proxy sidecar
│       ├── worker.yaml          # Worker + Cloud SQL Auth Proxy sidecar
│       ├── grafana.yaml
│       └── worker-keda-scaler.yaml
├── docker-compose.yaml
├── Dockerfile.api
├── Dockerfile.worker
└── go.mod
```

## 🔍 Observability

### Metrics (Prometheus)

The worker service exposes Prometheus metrics at `:9090/metrics`:

- `documents_processed_total`: Total successfully processed documents
- `documents_failed_total`: Total failed documents (includes DLQ moves)
- `documents_in_processing`: Current documents being processed (gauge)
- `document_processing_duration_seconds`: Processing duration histogram with `status` label (`done`/`failed`)

Query example — 95th percentile processing time:
```
histogram_quantile(0.95, rate(document_processing_duration_seconds_bucket[5m]))
```

### Dashboards (Grafana)

Grafana is deployed with a provisioned dashboard (defined as code in `k8s/grafana.yaml`) — no manual setup required. Access at port `30300`:

```bash
kubectl port-forward svc/grafana-svc 3000:3000
# or via minikube: http://$(minikube ip):30300
```

The dashboard includes panels for processed/failed document counts, in-flight documents, processing rate over time, and p95 processing duration by status.

### Distributed Tracing (Jaeger)

Both services emit traces to Jaeger. Each document trace carries span attributes for debugging individual documents:

- `document.id`: UUID of the document being processed
- `document.name`: Document name
- `document.status`: Final status (`done`, `failed`, `error`)

Failed spans are tagged with error status and the original error, making them visually distinct in the Jaeger UI.

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
3. API publishes `documents.created` event to NATS JetStream (`DOCUMENT_EVENTS` stream)
4. Worker consumes the event, acquires a DB-level lock, and processes the document
5. Worker updates the document status to `done` or `failed`
6. If processing fails, the worker increments the retry count and NAKs with backoff delay
7. Once max retries are exhausted, the document is published to the `DOCUMENT_DLQ` stream and the original message is acknowledged
8. Admins can inspect dead-lettered documents via `GET /api/v1/admin/dlq`
9. Metrics and distributed traces are collected throughout the process

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
| `DLQ_LIMIT` | Default page size for DLQ admin endpoint | `50` |



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
- [Grafana](https://grafana.com/)
- [KEDA](https://keda.sh/)
