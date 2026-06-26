.PHONY: install proto test clean

VENV_DIR := .venv
VENV_PY := $(VENV_DIR)/bin/python
# blspy only ships wheels for CPython <= 3.12, so prefer a compatible interpreter and fall back to python3
PYTHON := $(shell command -v python3.12 || command -v python3.11 || command -v python3.10 || command -v python3)

# Create the venv and install tutorial dependencies (protobuf, blspy)
$(VENV_PY):
	$(PYTHON) -m venv $(VENV_DIR)
	$(VENV_DIR)/bin/pip install --quiet --upgrade pip
	$(VENV_DIR)/bin/pip install -r requirements.txt

# Install dependencies into an isolated virtual environment
install: $(VENV_PY)

# Generate protobuf Python files
proto:
	python3 -m grpc_tools.protoc --python_out=proto --proto_path=proto proto/tx.proto

# Run the integration tests (requires a running Canopy node with the python plugin).
# This runs the transaction tests AND the custom RPC endpoints test
# (/v1/query/faucets, /v1/query/rewards) — the latter needs the plugin's RPC server on port 50010.
test: install
	$(VENV_PY) rpc_test.py && $(VENV_PY) rpc_test.py custom

# Clean generated files and the virtual environment
clean:
	rm -rf $(VENV_DIR)
	find . -type d -name __pycache__ -delete
	find . -type f -name "*.pyc" -delete
