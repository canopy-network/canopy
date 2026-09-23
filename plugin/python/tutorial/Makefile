.PHONY: install proto test clean

# Virtual environment (avoids PEP 668 "externally-managed-environment" errors)
VENV_DIR := .venv
PYTHON := $(VENV_DIR)/bin/python3
PIP := $(VENV_DIR)/bin/pip

# Create the virtual environment if it doesn't already exist
$(VENV_DIR):
	python3 -m venv $(VENV_DIR)

# Install dependencies into the virtual environment
install: $(VENV_DIR)
	$(PIP) install --upgrade pip
	$(PIP) install -r requirements.txt

# Generate protobuf Python files
proto: $(VENV_DIR)
	$(PYTHON) -m grpc_tools.protoc --python_out=proto --proto_path=proto proto/tx.proto

# Run the RPC test (requires running Canopy node with plugin)
test: $(VENV_DIR)
	$(PYTHON) rpc_test.py

# Clean generated files and the virtual environment
clean:
	rm -rf $(VENV_DIR)
	find . -type d -name __pycache__ -delete
	find . -type f -name "*.pyc" -delete
