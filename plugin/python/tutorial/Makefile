.PHONY: install proto test clean

# Install dependencies
install:
	pip install -r requirements.txt

# Generate protobuf Python files
proto:
	python3 -m grpc_tools.protoc --python_out=proto --proto_path=proto proto/tx.proto

# Run the RPC test (requires running Canopy node with plugin)
test:
	python3 rpc_test.py

# Clean generated files
clean:
	find . -type d -name __pycache__ -delete
	find . -type f -name "*.pyc" -delete
