.PHONY: install dev test lint format type-check clean proto build docs

# Development setup
install:
	pip install -e .

dev:
	pip install -e ".[dev]"

# Testing
test:
	pytest

test-cov:
	pytest --cov=plugin --cov-report=html --cov-report=term

test-verbose:
	pytest -v

# Code quality
lint:
	flake8 plugin/ tests/

format:
	black plugin/ tests/
	isort plugin/ tests/

type-check:
	mypy plugin/

# Protobuf generation
proto:
	cd plugin/proto && python -m grpc_tools.protoc --python_out=. --proto_path=. *.proto
	# Fix relative imports in generated files
	sed -i 's/^import \([^.]\)/from . import \1/' plugin/proto/*_pb2.py

# Build and distribution
build:
	python -m build

clean:
	rm -rf build/
	rm -rf dist/
	rm -rf *.egg-info/
	rm -rf .pytest_cache/
	rm -rf htmlcov/
	find . -type d -name __pycache__ -delete
	find . -type f -name "*.pyc" -delete

# Development servers
serve:
	uvicorn plugin.server:app --host 0.0.0.0 --port 8000

serve-dev:
	uvicorn plugin.server:app --reload --host 0.0.0.0 --port 8000

# Plugin execution
run-plugin:
	python main.py

# Full validation
validate: lint type-check test

# Setup pre-commit hooks
hooks:
	pre-commit install

# Documentation
docs:
	@echo "Documentation available in README.md"
	@echo "API docs available at http://localhost:8000/docs when server is running"
