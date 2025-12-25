# URL Shortener - Production-Grade Go Application

A professional URL shortener service built with Go, demonstrating advanced backend engineering concepts beyond basic CRUD operations.

## 🎯 Project Goals

This project is designed to help junior developers transition to mid/senior level by implementing production-ready backend patterns and concepts that go far beyond simple CRUD applications.

## ✨ Features

### Core Functionality
- ✅ **URL Shortening** - Generate short, unique codes for long URLs
- ✅ **Custom Aliases** - Create memorable custom short URLs
- ✅ **URL Expiration** - Set time-to-live (TTL) for temporary URLs
- ✅ **Click Analytics** - Track clicks with IP, user agent, and referrer
- ✅ **Persistent Storage** - PostgreSQL database with connection pooling
- ✅ **Health Checks** - Kubernetes-ready liveness/readiness endpoints

### Advanced Features (Implemented)
- ✅ **Layered Architecture** - Clean separation: Handler → Service → Repository → Domain
- ✅ **Repository Pattern** - Abstract data access for testability and flexibility
- ✅ **Dependency Injection** - No global state, fully testable components
- ✅ **Structured Logging** - JSON logs with request IDs for distributed tracing
- ✅ **Graceful Shutdown** - Proper signal handling and connection draining
- ✅ **Middleware Chain** - Logging, recovery, CORS, request ID, timeout
- ✅ **Database Migrations** - Version-controlled schema changes
- ✅ **Configuration Management** - Environment-based config with sensible defaults
- ✅ **Error Handling** - Consistent error responses and proper HTTP status codes

### Planned Features
- ⏳ **Redis Caching** - Cache frequently accessed URLs for performance
- ⏳ **Rate Limiting** - Token bucket algorithm to prevent abuse
- ⏳ **Metrics** - Prometheus metrics for monitoring
- ⏳ **Unit & Integration Tests** - Comprehensive test coverage
- ⏳ **API Documentation** - OpenAPI/Swagger specification

## 🏗️ Architecture

### Project Structure

```
url-shortener/
├── cmd/
│   └── server/
│       └── main.go              # Application entry point
├── internal/
│   ├── config/                  # Configuration management
│   ├── domain/                  # Domain models & business logic
│   ├── handler/http/            # HTTP handlers & middleware
│   ├── repository/              # Data access layer
│   │   └── postgres/            # PostgreSQL implementation
│   └── service/                 # Business logic layer
├── pkg/
│   ├── logger/                  # Logging utilities
│   └── validator/               # Input validation
├── migrations/                  # Database migrations
├── docker-compose.yml           # Local development environment
├── .env.example                 # Environment configuration template
└── Makefile                     # Development commands
```

### Layered Architecture

```
┌─────────────────────────────────────────┐
│         HTTP Layer (Handlers)           │  ← Presentation
├─────────────────────────────────────────┤
│       Service Layer (Business Logic)    │  ← Business Logic
├─────────────────────────────────────────┤
│    Repository Layer (Data Access)       │  ← Data Access
├─────────────────────────────────────────┤
│       Domain Layer (Models)             │  ← Domain Models
└─────────────────────────────────────────┘
```

**Benefits:**
- **Testability**: Each layer can be tested independently with mocks
- **Maintainability**: Changes in one layer don't cascade to others
- **Flexibility**: Easy to swap implementations (e.g., PostgreSQL → MongoDB)

## 🚀 Getting Started

### Prerequisites

- **Go 1.23+** - [Download](https://golang.org/dl/)
- **Docker Desktop** - [Download](https://www.docker.com/products/docker-desktop)
- **Make** (optional) - For convenience commands

### Quick Start

1. **Clone the repository**
   ```bash
   git clone <your-repo-url>
   cd url-shortener
   ```

2. **Set up environment**
   ```bash
   cp .env.example .env
   ```

3. **Start services (PostgreSQL, Redis, Prometheus)**
   ```bash
   docker-compose up -d
   ```

4. **Run database migrations**
   ```bash
   Get-Content migrations\001_initial_schema.sql | docker exec -i url-shortener-postgres psql -U urlshortener -d urlshortener
   ```

5. **Start the application**
   ```bash
   go run cmd/server/main.go
   ```

6. **Test the API**
   ```bash
   # Create a short URL
   Invoke-WebRequest -Uri "http://localhost:8080/api/v1/urls" `
     -Method POST `
     -Headers @{"Content-Type"="application/json"} `
     -Body '{"url": "https://github.com"}' `
     -UseBasicParsing
   ```

## 📚 API Documentation

### Create Short URL

**POST** `/api/v1/urls`

Create a new shortened URL.

**Request Body:**
```json
{
  "url": "https://example.com/very/long/url",
  "custom_alias": "mylink",           // Optional: custom short code
  "expires_in_hours": 24              // Optional: expiration time
}
```

**Response (201 Created):**
```json
{
  "data": {
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "short_code": "abc123",
    "short_url": "http://localhost:8080/abc123",
    "original_url": "https://example.com/very/long/url",
    "created_at": "2025-12-25T14:55:29Z",
    "expires_at": "2025-12-26T14:55:29Z"
  },
  "message": "URL created successfully"
}
```

### Redirect to Original URL

**GET** `/{shortCode}`

Redirects to the original URL and tracks analytics.

**Example:**
```bash
curl -L http://localhost:8080/abc123
# Redirects to https://example.com/very/long/url
```

### Get URL Statistics

**GET** `/api/v1/urls/{shortCode}/stats`

Retrieve analytics for a shortened URL.

**Response (200 OK):**
```json
{
  "data": {
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "short_code": "abc123",
    "original_url": "https://example.com/very/long/url",
    "clicks": 42,
    "created_at": "2025-12-25T14:55:29Z",
    "recent_clicks": [
      {
        "clicked_at": "2025-12-25T15:30:00Z",
        "country_code": "US",
        "city": "San Francisco"
      }
    ]
  }
}
```

### Health Check

**GET** `/health/live`

Liveness probe for container orchestration.

**Response (200 OK):**
```json
{
  "status": "ok",
  "time": "2025-12-25T14:55:29Z"
}
```

## 🧠 Backend Concepts Demonstrated

### 1. **Layered Architecture**
Clean separation of concerns with Handler → Service → Repository → Domain layers.

### 2. **Repository Pattern**
Abstract data access behind interfaces for testability and flexibility.

### 3. **Dependency Injection**
All dependencies passed through constructors, no global state.

### 4. **Database Connection Pooling**
Reusable connections for performance (25 max connections, 5 idle).

### 5. **Database Migrations**
Version-controlled schema changes with SQL migration files.

### 6. **Context Propagation**
`context.Context` for timeouts, cancellation, and request-scoped values.

### 7. **Structured Logging**
JSON logs with request IDs for distributed tracing and log aggregation.

### 8. **Middleware Pattern**
Composable request/response processing (logging, recovery, CORS, etc.).

### 9. **Graceful Shutdown**
Proper signal handling to drain connections before shutdown.

### 10. **Error Handling**
Custom errors, error wrapping with `fmt.Errorf`, and consistent API responses.

### 11. **Domain-Driven Design**
Business logic lives in domain models, not scattered across layers.

### 12. **Atomic Operations**
Database-level atomic increments to prevent race conditions.

### 13. **Asynchronous Processing**
Analytics tracking doesn't block redirects (goroutines).

### 14. **Configuration Management**
Environment-based config following 12-factor app principles.

## 🛠️ Development

### Useful Commands

```bash
# Start all services
docker-compose up -d

# View logs
docker-compose logs -f

# Stop all services
docker-compose down

# Run database migrations
Get-Content migrations\001_initial_schema.sql | docker exec -i url-shortener-postgres psql -U urlshortener -d urlshortener

# Access PostgreSQL shell
docker exec -it url-shortener-postgres psql -U urlshortener -d urlshortener

# Access Redis shell
docker exec -it url-shortener-redis redis-cli

# Run the application
go run cmd/server/main.go

# Build binary
go build -o bin/url-shortener cmd/server/main.go

# Run tests (when implemented)
go test -v ./...
```

### Database Schema

**URLs Table:**
```sql
CREATE TABLE urls (
    id UUID PRIMARY KEY,
    short_code VARCHAR(20) UNIQUE NOT NULL,
    original_url TEXT NOT NULL,
    custom_alias VARCHAR(50) UNIQUE,
    created_at TIMESTAMP NOT NULL,
    expires_at TIMESTAMP,
    clicks BIGINT DEFAULT 0,
    created_by VARCHAR(255),
    is_active BOOLEAN DEFAULT true
);
```

**Analytics Table:**
```sql
CREATE TABLE url_clicks (
    id BIGSERIAL PRIMARY KEY,
    url_id UUID REFERENCES urls(id),
    clicked_at TIMESTAMP NOT NULL,
    ip_address INET,
    user_agent TEXT,
    referer TEXT,
    country_code VARCHAR(2),
    city VARCHAR(100)
);
```

## 🔒 Security Considerations

- ✅ **SQL Injection Prevention** - Parameterized queries with `$1, $2` placeholders
- ✅ **Input Validation** - URL format and custom alias validation
- ✅ **Panic Recovery** - Middleware prevents server crashes
- ✅ **CORS Configuration** - Cross-origin resource sharing headers
- ⏳ **Rate Limiting** - Planned: Token bucket algorithm
- ⏳ **API Authentication** - Planned: API key authentication

## 📊 Monitoring

### Prometheus Metrics (Planned)

The application will expose metrics at `/metrics`:

- `http_requests_total` - Total HTTP requests
- `http_request_duration_seconds` - Request latency
- `url_shortener_urls_created_total` - URLs created
- `url_shortener_redirects_total` - Redirects performed
- `url_shortener_cache_hits_total` - Cache hits (when Redis is implemented)

Access Prometheus UI: http://localhost:9090

## 🎓 Learning Resources

### Go Concepts Covered

- **Structs & Methods** - Custom types with behavior
- **Interfaces** - Abstract contracts for polymorphism
- **Pointers** - Memory efficiency and nullable fields
- **Error Handling** - Explicit error returns, error wrapping
- **Goroutines** - Concurrent execution
- **Channels** - Communication between goroutines
- **Context** - Request lifecycle management
- **Defer** - Resource cleanup
- **JSON Encoding/Decoding** - API serialization

### Backend Patterns Covered

- Repository Pattern
- Service Layer Pattern
- Dependency Injection
- Middleware Pattern
- Builder Pattern
- Factory Pattern

## 🚧 Roadmap

- [ ] Redis caching layer
- [ ] Rate limiting middleware
- [ ] Unit tests with mocks
- [ ] Integration tests with testcontainers
- [ ] API documentation (Swagger/OpenAPI)
- [ ] Dockerfile for production deployment
- [ ] CI/CD pipeline (GitHub Actions)
- [ ] Performance benchmarks
- [ ] Geolocation service integration
- [ ] Custom domain support

## 📝 License

This is a learning project. Feel free to use it for educational purposes.

## 🙏 Acknowledgments

Built as a learning project to demonstrate production-grade Go development practices and advanced backend engineering concepts.

---

**Note**: This project is designed for learning and portfolio purposes. For production use, additional hardening (authentication, rate limiting, monitoring) is recommended.
