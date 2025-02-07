# MetadataTool Documentation

## Overview
MetadataTool is a comprehensive solution for managing music metadata, providing AI-powered enrichment, DDEX compliance, and efficient storage management.

## Quick Start
```bash
# Clone the repository
git clone https://github.com/yourusername/metadatatool.git

# Install dependencies
go mod download

# Run the application
go run cmd/metadatatool/main.go
```

## Architecture
The application follows clean architecture principles with the following layers:
- Domain (core business logic)
- Use Cases (application logic)
- Repositories (data access)
- Handlers (HTTP/transport layer)

## Core Features
1. **Metadata Management**
   - Track metadata CRUD operations
   - Batch processing
   - Version control

2. **AI Integration**
   - Automatic metadata enrichment
   - Confidence scoring
   - Batch processing

3. **Storage Management**
   - S3-compatible storage
   - File versioning
   - Cleanup management

4. **DDEX Integration**
   - ERN 4.3 support
   - Validation
   - Import/Export

## API Reference
### Track Management
```graphql
type Track {
  id: ID!
  title: String!
  artist: String!
  # ... other fields
}

type Query {
  track(id: ID!): Track
  tracks(filter: TrackFilter): [Track!]!
}

type Mutation {
  createTrack(input: CreateTrackInput!): Track!
  updateTrack(id: ID!, input: UpdateTrackInput!): Track!
  deleteTrack(id: ID!): Boolean!
}
```

### File Management
```graphql
type File {
  id: ID!
  name: String!
  size: Int!
  url: String!
}

type Mutation {
  uploadFile(file: Upload!): File!
  deleteFile(id: ID!): Boolean!
}
```

## Configuration
```yaml
server:
  port: 8080
  timeout: 30s

database:
  host: localhost
  port: 5432
  name: metadatatool

ai:
  provider: openai
  model: gpt-4
  maxTokens: 1000

storage:
  provider: s3
  bucket: metadatatool
  region: us-west-2
```

## Development
### Prerequisites
- Go 1.21+
- PostgreSQL 14+
- Redis 6+
- Node.js 18+ (for frontend)

### Local Setup
1. Copy `.env.example` to `.env`
2. Configure environment variables
3. Run database migrations
4. Start the development server

### Testing
```bash
# Run unit tests
go test ./...

# Run integration tests
go test -tags=integration ./...

# Run with coverage
go test -cover ./...
```

## Deployment
### Docker
```bash
# Build image
docker build -t metadatatool .

# Run container
docker run -p 8080:8080 metadatatool
```

### Kubernetes
```bash
# Apply configurations
kubectl apply -f k8s/

# Check status
kubectl get pods -n metadatatool
```

## Troubleshooting
### Common Issues
1. **Database Connection**
   - Check credentials
   - Verify network access
   - Check PostgreSQL logs

2. **Storage Issues**
   - Verify S3 credentials
   - Check bucket permissions
   - Validate file paths

3. **AI Service**
   - Check API keys
   - Verify rate limits
   - Monitor response times

## Contributing
1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Submit a pull request

## License
MIT License - see LICENSE file for details 