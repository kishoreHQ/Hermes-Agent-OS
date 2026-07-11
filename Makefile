.PHONY: test build run serve status clean fmt vet lint help smoke

KERNEL_DIR := kernel
BIN_DIR := bin
HERMESD := $(BIN_DIR)/hermesd
ADDR ?= :8080

help: ## Show targets
	@grep -E '^[a-zA-Z_-]+:.*?##' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  %-12s %s\n", $$1, $$2}'

test: ## Run kernel unit tests
	cd $(KERNEL_DIR) && go test ./...

build: ## Build hermesd
	mkdir -p $(BIN_DIR)
	cd $(KERNEL_DIR) && go build -o ../$(HERMESD) ./cmd/hermesd

status: build ## Print hermesd status
	./$(HERMESD) status

serve: build ## Serve Host API (ADDR=:8080)
	./$(HERMESD) serve $(ADDR)

run: serve ## Alias for serve

smoke: build ## HTTP smoke against a temporary server
	@./$(HERMESD) serve 127.0.0.1:18080 & pid=$$!; \
	  sleep 0.4; \
	  curl -sf http://127.0.0.1:18080/api/v1/health | grep -q '"status":"ok"'; \
	  curl -sf -X POST http://127.0.0.1:18080/api/v1/missions \
	    -H 'Content-Type: application/json' \
	    -d '{"goal":"smoke","requiredCapabilities":["coding"]}' | grep -q '"state":"running"'; \
	  curl -sf 'http://127.0.0.1:18080/api/v1/events?since=0&format=json' | grep -q '"seq"'; \
	  kill $$pid 2>/dev/null; wait $$pid 2>/dev/null; \
	  echo "smoke ok"

fmt: ## go fmt
	cd $(KERNEL_DIR) && go fmt ./...

vet: ## go vet
	cd $(KERNEL_DIR) && go vet ./...

clean: ## Remove build artifacts
	rm -rf $(BIN_DIR)

lint: vet test ## Vet + test
