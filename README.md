# Metadata Tool

A robust metadata management tool for audio tracks, featuring AI-powered metadata enrichment, DDEX ERN 4.3 compliance, and comprehensive track management capabilities.

## Features

- 🎵 Audio track metadata management
- 🤖 AI-powered metadata enrichment
- 📊 DDEX ERN 4.3 export and validation
- 🔐 Role-based access control
- 🚀 High-performance caching
- 📈 Prometheus metrics and monitoring
- 🔍 Error tracking with Sentry

## Tech Stack

- **Backend**: Go 1.21+
- **Database**: PostgreSQL
- **Cache**: Redis
- **Storage**: S3-compatible storage
- **AI**: OpenAI GPT integration
- **Monitoring**: Prometheus + Grafana
- **Error Tracking**: Sentry

## Getting Started

### Prerequisites

- Go 1.21 or higher
- PostgreSQL 13+
- Redis 6+
- S3-compatible storage (AWS S3, MinIO, etc.)
- OpenAI API key (for AI features)

### Environment Variables

Copy `.env.example` to `.env` and configure:

```bash
# Server
SERVER_PORT=8080
ENVIRONMENT=development
LOG_LEVEL=info

# Database
DB_HOST=localhost
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=your_password
DB_NAME=metadatatool
DB_SSLMODE=disable

# Redis
REDIS_ENABLED=true
REDIS_HOST=localhost
REDIS_PORT=6379
REDIS_PASSWORD=
REDIS_DB=0

# Auth
JWT_SECRET=your_jwt_secret
ACCESS_TOKEN_EXPIRY=15m
REFRESH_TOKEN_EXPIRY=7d

# AI
AI_API_KEY=your_openai_api_key
AI_MODEL_NAME=gpt-4
AI_MODEL_VERSION=latest

# Storage
STORAGE_PROVIDER=s3
STORAGE_REGION=us-east-1
STORAGE_BUCKET=metadatatool
STORAGE_ACCESS_KEY=your_access_key
STORAGE_SECRET_KEY=your_secret_key
```

### Installation

1. Clone the repository:
```bash
git clone https://github.com/yourusername/metadatatool.git
cd metadatatool
```

2. Install dependencies:
```bash
go mod download
```

3. Run database migrations:
```bash
go run cmd/migrate/main.go up
```

4. Start the server:
```bash
go run cmd/api/main.go
```

### Docker Deployment

Build and run with Docker Compose:

```bash
docker-compose up -d
```

For monitoring stack:

```bash
docker-compose -f docker-compose.monitoring.yml up -d
```

## API Documentation

API documentation is available at `/swagger/index.html` when running in development mode.

## Contributing

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add some amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details. 