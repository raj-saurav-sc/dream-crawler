# 🌌 Web Crawler That Dreams

[![Go Version](https://img.shields.io/badge/Go-1.22+-00ADD8.svg?style=for-the-badge&logo=go)](https://golang.org/)
[![Python Version](https://img.shields.io/badge/Python-3.9+-3776AB.svg?style=for-the-badge&logo=python)](https://www.python.org/)
[![License](https://img.shields.io/badge/License-MIT-green.svg?style=for-the-badge)](https://opensource.org/licenses/MIT)

An experimental **AI-powered web crawler** that ingests web content, processes it, and generates *dreamlike associations*. This project combines **Go** for high-performance crawling and orchestration with **Python** for AI/ML-powered content enrichment.

---

## 📂 Project Structure

The repository is a monorepo containing multiple services and shared resources:

```
web-crawler-that-dreams/
├── go-backend/       # Go microservices for crawling, orchestration, etc.
├── py-ml-service/    # Python microservice for ML/AI tasks (embeddings, LLM calls)
├── deployments/      # Docker, Kubernetes, and Helm configurations
├── scripts/          # Helper scripts for development (linting, migrations)
├── shared/           # Shared resources like Protobufs, OpenAPI specs, and configs
├── bin/              # Compiled application binaries (auto-generated)
├── docker-compose.yml # Local development environment orchestration
├── Makefile          # Unified command interface for building, testing, and running
└── README.md         # This file
```

---

## 🚀 Getting Started

### Prerequisites
- Go >= 1.22
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
make run-go-orchestrator
make run-py-service
```

---

## 🧪 Testing

Run all Go and Python tests with a single command:

```bash
make test
```

---

## 🔮 Roadmap

- [x] Go-based concurrent crawler
- [x] Python ML enrichment service
- [ ] Integration via gRPC/REST
- [ ] Kubernetes deployment
- [ ] CI/CD with GitHub Actions
- [ ] Scalable vector search (Qdrant)