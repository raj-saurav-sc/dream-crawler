# Supports building Go services, setting up Python venv, and running tests.

APP_NAME=webcrawler-dream
GO_CMD=go
PYTHON=python3
PIP=$(PYTHON) -m pip
VENV_DIR=python-ml/venv

.PHONY: all go-build go-run py-venv py-install py-run clean docker-build docker-up docker-down test

all: go-build py-install

## Go targets ##
go-build:
	cd crawler && $(GO_CMD) build -o ../bin/crawler
	cd orchestrator && $(GO_CMD) build -o ../bin/orchestrator
	cd storage && $(GO_CMD) build -o ../bin/storage

go-run:
	cd orchestrator && $(GO_CMD) run main.go

## Python targets ##
py-venv:
	test -d $(VENV_DIR) || $(PYTHON) -m venv $(VENV_DIR)

py-install: py-venv
	$(VENV_DIR)/bin/$(PIP) install -r python-ml/requirements.txt

py-run:
	$(VENV_DIR)/bin/python python-ml/service.py

## Docker targets ##
docker-build:
	docker-compose build

docker-up:
	docker-compose up -d

docker-down:
	docker-compose down

## Tests ##
test:
	cd crawler && $(GO_CMD) test ./...
	cd orchestrator && $(GO_CMD) test ./...
	cd storage && $(GO_CMD) test ./...
	$(VENV_DIR)/bin/python -m pytest python-ml/tests

## Clean ##
clean:
	rm -rf bin/
	rm -rf $(VENV_DIR)
