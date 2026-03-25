# Project: browser-mcp-extension

Browser automation via MCP (Model Context Protocol) and HTTP API, powered by a Chrome Extension (MV3) + Go binary.

## Tech Stack

| Layer | Technology | Version |
|---|---|---|
| Backend | Go | 1.24+ (toolchain 1.25.7) |
| WebSocket | gorilla/websocket | 1.5.3 |
| MCP | mark3labs/mcp-go | 0.44.0 |
| Extension | Chrome Manifest V3 | Vanilla JS |
| E2E tests | playwright-go + mcp-go client | Separate module (`test/e2e/`) |
| CI/CD | GitHub Actions | SHA-pinned actions |

## Configuration

All configuration via environment variables. See `env.example` for defaults.

| Variable | Default | Description |
|---|---|---|
| `WS_PORT` | `9001` | WebSocket port for the Chrome extension |
| `WS_TOKEN` | _(empty)_ | Optional bearer token for WS auth |
| `HTTP_PORT` | `9082` | HTTP API port (loopback only) |
| `JS_SCRIPTS_PATH` | `./resources/js_scripts` | JS automation scripts directory |
| `OBSERVATIONS_DIR` | _(empty)_ | Disk persistence for snapshots (empty = in-memory only) |

## Build & Run

| Command | Description |
|---|---|
| `make build` | Compile `browser-cmd` binary |
| `make sync-scripts` | Wrap JS scripts + utils.js into `extension/scripts/` |
| `make serve` | HTTP+WS server (sync-scripts + build, then serve) |
| `make mcp` | MCP server on stdio (build, then run with `--mcp`) |
| `make test` | Unit tests (`go test ./...`) |
| `make vet` | `go vet ./...` |
| `make e2e` | E2E tests (requires Chrome + display) |
| `make clean` | Remove binary, observations, wrapped scripts |

## Quality Gates

- Lint: `go vet ./...`
- Format: `gofmt -l .`
- Unit tests: `go test ./...`
- E2E tests: `cd test/e2e && go test -v -count=1 -timeout 10m -p 1 ./...`

## Dependencies

- **No database** — this project is stateless. Observation store is in-memory with optional disk persistence.
- **No Docker Compose** — no infrastructure dependencies.
- **Chrome or Chromium** required at runtime for the extension.

## Release

GitHub Actions workflow (`.github/workflows/release.yml`) triggers on `v*` tags:
- Runs tests
- Builds `darwin-arm64` binary
- Packages Chrome extension as zip
- Creates GitHub Release with both artifacts
