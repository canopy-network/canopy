.PHONY: venv install dev test lint format type-check clean proto build docs

# Virtual environment
VENV_DIR := .venv
PYTHON := $(VENV_DIR)/bin/python3
PIP := $(VENV_DIR)/bin/pip

# Create virtual environment
venv:
	@if [ ! -d "$(VENV_DIR)" ]; then \
		echo "Creating virtual environment..."; \
		python3 -m venv $(VENV_DIR); \
	fi

# Development setup
install: venv
	$(PIP) install -e .

dev: venv
	$(PIP) install -e ".[dev]"

# Testing
test: venv
	$(VENV_DIR)/bin/pytest

test-cov: venv
	$(VENV_DIR)/bin/pytest --cov=plugin --cov-report=html --cov-report=term

test-verbose: venv
	$(VENV_DIR)/bin/pytest -v

# Code quality
lint: venv
	$(VENV_DIR)/bin/flake8 plugin/ tests/

format: venv
	$(VENV_DIR)/bin/black plugin/ tests/
	$(VENV_DIR)/bin/isort plugin/ tests/

type-check: venv
	$(VENV_DIR)/bin/mypy plugin/

# Protobuf generation
proto: venv
	cd contract/proto && $(PYTHON) -m grpc_tools.protoc --python_out=. --proto_path=. *.proto
	# Fix relative imports in generated files
	sed -i 's/^import \([^.]\)/from . import \1/' contract/proto/*_pb2.py

# Build and distribution
build: venv
	$(PYTHON) -m build

clean:
	rm -rf build/
	rm -rf dist/
	rm -rf *.egg-info/
	rm -rf .pytest_cache/
	rm -rf htmlcov/
	find . -type d -name __pycache__ -delete
	find . -type f -name "*.pyc" -delete

# Development servers
serve: venv
	$(VENV_DIR)/bin/uvicorn plugin.server:app --host 0.0.0.0 --port 8000

serve-dev: venv
	$(VENV_DIR)/bin/uvicorn plugin.server:app --reload --host 0.0.0.0 --port 8000

# Plugin execution
run-plugin: venv
	$(PYTHON) main.py

# Full validation
validate: venv lint type-check test

# Setup pre-commit hooks
hooks:
	pre-commit install

# Documentation
docs:
	@echo "Documentation available in README.md"
	@echo "API docs available at http://localhost:8000/docs when server is running"
