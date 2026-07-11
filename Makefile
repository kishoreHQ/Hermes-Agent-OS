.PHONY: test build run clean fmt vet lint help

KERNEL_DIR := kernel
BIN_DIR := bin
HERMESD := $(BIN_DIR)/hermesd

help: ## Show targets
	@grep -E '^[a-zA-Z_-]+:.*?##' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  %-12s %s\n", $$1, $$2}'

test: ## Run kernel unit tests
	cd $(KERNEL_DIR) && go test ./...

build: ## Build hermesd
	mkdir -p $(BIN_DIR)
	cd $(KERNEL_DIR) && go build -o ../$(HERMESD) ./cmd/hermesd

run: build ## Build and run hermesd
	./$(HERMESD)

fmt: ## go fmt
	cd $(KERNEL_DIR) && go fmt ./...

vet: ## go vet
	cd $(KERNEL_DIR) && go vet ./...

clean: ## Remove build artifacts
	rm -rf $(BIN_DIR)

lint: vet test ## Vet + test
