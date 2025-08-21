# 🌌 Web Crawler That Dreams

[![Go Version](https://img.shields.io/badge/Go-1.23+-00ADD8.svg?style=for-the-badge&logo=go)](https://golang.org/)
[![Python Version](https://img.shields.io/badge/Python-3.9+-3776AB.svg?style=for-the-badge&logo=python)](https://www.python.org/)
[![License](https://img.shields.io/badge/License-MIT-green.svg?style=for-the-badge)](https://opensource.org/licenses/MIT)

An experimental **AI-powered web crawler** that ingests web content, processes it, and generates *dreamlike associations*. This project combines **Go** for high-performance crawling and orchestration with **Python** for AI/ML-powered content enrichment and dream generation.

## 🎯 Vision

A web crawler that not only indexes and extracts content but also **"dreams"** about the data: generating associations, analogies, and surreal insights by combining classical crawling with AI/ML imagination layers.

## 🏗️ Architecture

### Core Components

1. **Crawling & Scraping Layer (Go)**
   - Built in **Go** for high performance, concurrency, and efficient network I/O
   - Uses worker pools and rate-limiting
   - Responsible for fetching HTML, JSON, PDFs, etc.
   - Outputs raw content streams to Kafka

2. **Content Processing & Normalization (Go)**
   - **Content Processor**: Cleans, normalizes, and enriches raw content
   - Extracts metadata, processes content chunks, analyzes dream hints
   - Publishes clean content for ML processing

3. **Dreaming Engine (Python/AI)**
   - Runs on Python for ML ecosystem (sentence-transformers, ctransformers)
   - **Semantic Embeddings**: Convert text to vector representations
   - **Associative Memory**: Store in vector DB (Qdrant)
   - **Dream Generator**: Uses generative models (LLM) to create surreal insights

4. **Orchestration Layer (Go)**
   - Job scheduler & orchestrator
   - Kafka used for event-driven messaging
   - Each stage publishes/subscribes to events

5. **Storage & Indexing**
   - **Vector DB (Qdrant)**: semantic search and dream storage
   - **PostgreSQL**: metadata, logs, crawl jobs
   - **Object storage**: raw snapshots (configurable)

6. **API Layer**
   - **Go REST API** (port 8080): high-perf endpoints for crawling, searching, metadata
   - **Python FastAPI** (port 8001): ML results, semantic search, dream generation
   - Unified via API gateway pattern

7. **Observability & Control Plane**
   - Health check endpoints on all services
   - Structured logging throughout
   - Metrics collection points

## 📂 Project Structure

```
web-crawler-that-dreams/
├── go-backend/                    # Go microservices
│   ├── cmd/                       # Service entry points
│   │   ├── crawler/              # Web crawler service
│   │   ├── content-processor/    # Content cleaning & enrichment
│   │   ├── orchestrator/         # Job orchestration
│   │   ├── indexer/              # Content indexing
│   │   └── api/                  # REST API service
│   ├── pkg/                      # Shared packages
│   │   └── model/                # Data models & types
│   ├── internal/                  # Internal packages
│   └── Dockerfile                 # Multi-service container
├── py-ml-service/                 # Python ML services
│   ├── api.py                    # FastAPI ML endpoints
│   ├── dream_processor.py        # Dream generation service
│   ├── narrative.py              # LLM narrative generator
│   ├── vector_store.py           # Qdrant vector store integration
│   └── requirements.txt          # Python dependencies
├── deployments/                   # Deployment configurations
├── scripts/                       # Helper scripts
├── shared/                        # Shared resources
├── docker-compose.yml            # Local development environment
├── Makefile                      # Unified command interface
└── README.md                     # This file
```

## 🚀 Getting Started

### Prerequisites
- Go >= 1.23
- Python >= 3.9
- Docker & Docker Compose
- Make

### Local Development with Docker (Recommended)

This is the easiest way to get all services running together.

```bash
# Build and start all services in the background
make up

# View logs for all services
make logs

# Stop and remove all services
make down
```

### Manual Build & Run

If you prefer to run services manually on your host machine:

```bash
# 1. Build all Go binaries into the ./bin directory
make build

# 2. Install Python dependencies into a virtual environment
make py-install

# 3. Run the services (in separate terminals)
make run-go-crawler
make run-go-content-processor
make run-go-api
make run-py-service
```

## 🔧 Service Details

### Go Services

- **Crawler** (`./bin/crawler`): Web crawling with rate limiting and robots.txt support
- **Content Processor** (`./bin/content-processor`): Content cleaning and enrichment
- **API** (`./bin/api`): REST API for crawling, search, and metadata
- **Orchestrator** (`./bin/orchestrator`): Job scheduling and orchestration
- **Indexer** (`./bin/indexer`): Content indexing and storage

### Python Services

- **ML API** (`python api.py`): FastAPI service for ML operations
- **Dream Processor** (`python dream_processor.py`): Kafka consumer for dream generation
- **Vector Store** (`vector_store.py`): Qdrant integration for embeddings

## 📡 API Endpoints

### Go REST API (Port 8080)

- `GET /health` - Health check
- `POST /crawl` - Create crawl job
- `GET /crawl/{id}` - Get crawl job details
- `GET /search` - Search documents
- `GET /search/semantic` - Semantic search
- `GET /search/dreams` - Search dreams
- `GET /documents/{id}` - Get document
- `GET /stats` - System statistics

### Python ML API (Port 8001)

- `GET /health` - Health check
- `POST /embed` - Generate text embeddings
- `POST /search/semantic` - Semantic search
- `POST /search/dreams` - Dream search
- `POST /dream` - Generate dream narrative
- `GET /dreams/{id}/similar` - Find similar dreams
- `GET /stats/vector-store` - Vector store statistics

## 🔄 Data Flow

1. **Crawler** → Kafka (`raw.content`) → **Content Processor**
2. **Content Processor** → Kafka (`clean.content`) → **Dream Processor**
3. **Dream Processor** → **Vector Store** (Qdrant) + **PostgreSQL**
4. **APIs** serve data from both storage layers

## 🧪 Testing

Run all Go and Python tests with a single command:

```bash
make test
```

## 📊 Monitoring

- Health check endpoints on all services
- Structured logging with consistent format
- Kafka topic monitoring
- Vector store statistics

## 🔮 Roadmap

- [x] Go-based concurrent crawler
- [x] Python ML enrichment service
- [x] Content processing pipeline
- [x] Vector database integration
- [x] REST API endpoints
- [x] FastAPI ML service
- [x] Docker Compose setup
- [ ] Kubernetes deployment
- [ ] CI/CD with GitHub Actions
- [ ] Advanced monitoring (Prometheus/Grafana)
- [ ] Authentication & authorization
- [ ] Rate limiting & API quotas

## 🌐 Kafka Topics

- `raw.content` - Raw crawled content
- `clean.content` - Processed and cleaned content
- `dream.outputs` - Generated dream narratives
- `crawl.jobs` - Crawl job management
- `crawl.results` - Crawl completion events

## 🗄️ Database Schema

### PostgreSQL Tables
- `crawl_jobs` - Crawling job metadata
- `documents` - Crawled document metadata
- `dreams` - Generated dream outputs
- `crawl_stats` - Crawling statistics

### Qdrant Collections
- `documents` - Document embeddings for semantic search
- `dreams` - Dream narrative embeddings

## 🐳 Docker Services

- **Zookeeper** - Kafka coordination
- **Kafka** - Message broker
- **Qdrant** - Vector database
- **PostgreSQL** - Metadata storage
- **Crawler** - Web crawling service
- **Content Processor** - Content processing service
- **API** - Go REST API
- **Dream Processor** - Dream generation service
- **ML API** - Python FastAPI service

## 📝 Configuration

Environment variables can be set in `docker-compose.yml` or passed directly to services:

- `KAFKA_BROKER` - Kafka broker address
- `QDRANT_HOST` - Qdrant host address
- `POSTGRES_HOST` - PostgreSQL host address
- `MODEL_PATH` - Path to LLM model file

## 🤝 Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests if applicable
5. Submit a pull request

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 🙏 Acknowledgments

- Built with Go and Python
- Uses Qdrant for vector search
- Kafka for event streaming
- PostgreSQL for metadata storage
- FastAPI for Python ML endpoints