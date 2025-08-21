# Makefile for the Web Crawler That Dreams project

# --- Variables ---
GO_CMD=go
PYTHON=python3
PIP=$(PYTHON) -m pip

# Service Directories
GO_ROOT=go-backend
PY_ROOT=py-ml-service
VENV_DIR=$(PY_ROOT)/.venv

# Binaries
BINS=crawler indexer orchestrator
GO_BINS=$(patsubst %,bin/%,$(BINS))

.DEFAULT_GOAL := help

# --- Docker Commands ---
up: ## Build and start all services with Docker Compose
	docker-compose up --build -d

down: ## Stop and remove all Docker Compose services
	docker-compose down

logs: ## Follow logs from all running services
	docker-compose logs -f

# --- Build Commands ---
build: go-build py-install ## Build all binaries and install Python dependencies

setup: download-model ## Download required ML models

download-model:
	@mkdir -p models
	@echo "Downloading TinyLlama model (if it doesn't exist)..."
	@test -f models/tinyllama-1.1b-chat-v1.0.Q4_K_M.gguf || \
		wget -q --show-progress -O models/tinyllama-1.1b-chat-v1.0.Q4_K_M.gguf https://huggingface.co/TheBloke/TinyLlama-1.1B-Chat-v1.0-GGUF/resolve/main/tinyllama-1.1b-chat-v1.0.Q4_K_M.gguf

go-build: $(GO_BINS) ## Build all Go binaries into ./bin

# This rule builds a binary like 'bin/crawler' from a source like './go-backend/cmd/crawler/'
bin/%:
	@echo "Building Go binary: $@"
	@mkdir -p bin
	$(GO_CMD) build -v -o $@ ./$(GO_ROOT)/cmd/$*

py-install: $(VENV_DIR)/touchfile ## Install Python dependencies into a virtualenv
	@echo "Python dependencies are up to date."

$(VENV_DIR)/touchfile: $(PY_ROOT)/requirements.txt
	test -d $(VENV_DIR) || $(PYTHON) -m venv $(VENV_DIR)
	$(VENV_DIR)/bin/$(PIP) install -r $(PY_ROOT)/requirements.txt
	touch $@

# --- Test Commands ---
test: test-go test-py ## Run all tests

test-go: ## Run Go unit tests
	cd $(GO_ROOT) && $(GO_CMD) test ./...

test-integration: ## Run Go integration tests (requires network)
	cd $(GO_ROOT) && $(GO_CMD) test -v -tags=integration ./...

test-py: ## Run Python unit tests
	$(VENV_DIR)/bin/python -m pytest $(PY_ROOT)/

# --- Cleanup ---
clean: ## Remove built artifacts and virtual environment
	rm -rf bin
	rm -rf $(VENV_DIR)
	@echo "Cleanup complete."

# --- Help ---
.PHONY: help
help: ## Display this help screen
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

# --- Manual Run Commands ---
run-py-dream-processor: py-install ## Run the Python dream processor service manually
	$(VENV_DIR)/bin/python $(PY_ROOT)/dream_processor.py
