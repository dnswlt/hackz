# Makefile

.PHONY: all proto run setup

# Variables
PROTO_DIR := planpb
PROTO_FILES := $(wildcard $(PROTO_DIR)/*.proto)
GENERATED_DIR := $(PROTO_DIR)
SCHEMA_FILE := $(PROTO_DIR)/plan.jsonschema.json

# setup: Installs necessary Go tools
setup:
	@echo "Installing Go tools..."
	go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
	go install github.com/chrusty/protoc-gen-jsonschema/cmd/protoc-gen-jsonschema@latest

# proto: Generates Go code and JSON Schema from proto files
proto:
	@echo "Generating Go code and JSON Schema from proto files..."
	protoc \
	  --proto_path=. \
	  --go_out=paths=source_relative:. \
	  --jsonschema_out=$(GENERATED_DIR) \
	  $(PROTO_FILES)

# run: Runs the main application
run:
	@echo "Running the application..."
	go run ./cmd/planr
