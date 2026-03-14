-include .env
export

BINARY  := browser-cmd
GOFLAGS := -trimpath

# ── Server settings ───────────────────────────────────────────────────
API_PORT ?= 9082

.PHONY: all build serve mcp test vet clean help sync-scripts e2e

all: serve

help:
	@echo "browser-mcp-extension commands:"
	@echo ""
	@echo "  Development:"
	@echo "    make build         - Build the browser-cmd binary"
	@echo "    make sync-scripts  - Wrap JS scripts into extension/scripts/"
	@echo "    make serve         - Start HTTP API + WebSocket server (WS :9001, HTTP :$(API_PORT))"
	@echo "    make mcp           - Start MCP server on stdio"
	@echo "    make test          - Run all unit tests"
	@echo "    make vet           - Run go vet"
	@echo "    make e2e           - Run E2E tests (requires Chrome + display server)"
	@echo "    make clean         - Remove binary, observations, and wrapped scripts"

# ═══════════════════════════════════════════════════════════════════════
# Build / Serve
# ═══════════════════════════════════════════════════════════════════════

build:
	go build $(GOFLAGS) -o $(BINARY) .
	@echo "Built: $(BINARY)"

sync-scripts:
	bash extension/wrap-scripts.sh \
		resources/js_lib/utils.js \
		extension/scripts \
		resources/js_scripts

serve: sync-scripts build
	@echo "Starting browser API server (WS :9001, HTTP :$(API_PORT))…"
	HTTP_PORT=$(API_PORT) ./$(BINARY)

mcp: build
	./$(BINARY) --mcp

# ═══════════════════════════════════════════════════════════════════════
# Quality
# ═══════════════════════════════════════════════════════════════════════

test:
	go test ./...

vet:
	go vet ./...

e2e: sync-scripts
	cd test/e2e && go test -v -count=1 -timeout 10m -p 1 ./...

clean:
	rm -f $(BINARY)
	rm -rf observations
	rm -rf extension/scripts
	@echo "Cleaned."
