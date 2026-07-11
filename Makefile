.PHONY: test build run serve status clean fmt vet lint help smoke prove-h4 prove-h5 conform e2e bench ui-install ui-build ui-dev ui-typecheck dev

KERNEL_DIR := kernel
UI_DIR := mission-control
BIN_DIR := bin
HERMESD := $(BIN_DIR)/hermesd
ADDR ?= :8080

help: ## Show targets
	@grep -E '^[a-zA-Z_-]+:.*?##' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  %-14s %s\n", $$1, $$2}'

test: ## Run kernel unit tests
	cd $(KERNEL_DIR) && go test ./...

build: ## Build hermesd
	mkdir -p $(BIN_DIR)
	cd $(KERNEL_DIR) && go build -o ../$(HERMESD) ./cmd/hermesd

status: build ## Print hermesd status
	./$(HERMESD) status

serve: build ## Serve Host API (+ SPA if mission-control/dist exists)
	./$(HERMESD) serve $(ADDR)

run: serve ## Alias for serve

ui-install: ## npm install Mission Control deps
	cd $(UI_DIR) && npm install

ui-build: ## Build Mission Control into mission-control/dist
	cd $(UI_DIR) && npm run build

ui-dev: ## Vite dev server (proxies /api → :8080)
	cd $(UI_DIR) && npm run dev -- --host 127.0.0.1 --port 5173

ui-typecheck: ## Typecheck Mission Control
	cd $(UI_DIR) && npm run typecheck

dev: ## Print dual-terminal dev instructions
	@echo "Terminal 1: make serve"
	@echo "Terminal 2: make ui-dev"
	@echo "Open http://127.0.0.1:5173  (API via proxy → :8080)"
	@echo "Or: make ui-build && make serve  → http://127.0.0.1:8080"

smoke: build ## HTTP smoke against a temporary server
	@./$(HERMESD) serve 127.0.0.1:18080 & pid=$$!; \
	  sleep 0.4; \
	  curl -sf http://127.0.0.1:18080/api/v1/health | grep -q '"status":"ok"'; \
	  curl -sf -X POST http://127.0.0.1:18080/api/v1/missions \
	    -H 'Content-Type: application/json' \
	    -d '{"goal":"smoke","requiredCapabilities":["coding","tools"]}' | tee /tmp/hermes-smoke.json | grep -q '"state":"succeeded"'; \
	  grep -q '"providerId":"provider.example.echo"' /tmp/hermes-smoke.json; \
	  curl -sf 'http://127.0.0.1:18080/api/v1/events?since=0&format=json' | grep -q 'route.decided'; \
	  curl -sf 'http://127.0.0.1:18080/api/v1/memory/search' | grep -q 'mem_'; \
	  curl -sf 'http://127.0.0.1:18080/api/v1/credentials' | grep -q 'cred_'; \
	  kill $$pid 2>/dev/null; wait $$pid 2>/dev/null; \
	  echo "smoke ok"

prove-h4: build ## H4 interchangeability proof (2×2 provider×runtime)
	./$(HERMESD) prove-h4

prove-h5: build ## H5 production hardening proof
	./$(HERMESD) prove-h5

conform: build ## AESP hermes-core conformance claim
	./$(HERMESD) conform

conform-full: build ## AESP hermes-agent-os profile
	./$(HERMESD) conform full

e2e: build ## Full Host HTTP e2e (starts temp hermesd if needed)
	bash scripts/e2e-host.sh

bench: ## Go benchmarks for mission path
	cd $(KERNEL_DIR) && go test ./pkg/perf/ -bench=. -benchmem -count=1

fmt: ## go fmt
	cd $(KERNEL_DIR) && go fmt ./...

vet: ## go vet
	cd $(KERNEL_DIR) && go vet ./...

clean: ## Remove build artifacts
	rm -rf $(BIN_DIR) $(UI_DIR)/dist

lint: vet test ## Vet + test
