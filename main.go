package main

import (
	"fmt"
	"github.com/paoloandrisani/browser-mcp-extension/internal/api"
	"github.com/paoloandrisani/browser-mcp-extension/internal/config"
	mcpserver "github.com/paoloandrisani/browser-mcp-extension/internal/mcp"
	"github.com/paoloandrisani/browser-mcp-extension/internal/observation"
	"github.com/paoloandrisani/browser-mcp-extension/internal/ws"
	"log"
	"net/http"
	"os"
	"path/filepath"
)

func main() {
	log.SetFlags(log.Ltime)

	if len(os.Args) >= 2 && os.Args[1] == "--mcp" {
		if err := runMCP(); err != nil {
			log.Fatalf("[MCP] ✗ %v", err)
		}
		return
	}

	if err := runServe(); err != nil {
		log.Fatalf("[API] ✗ %v", err)
	}
}

func setup() (*api.Handler, *config.Config, *observation.Store, error) {
	cfg, err := config.NewConfig().Load()
	if err != nil {
		return nil, nil, nil, fmt.Errorf("load config: %w", err)
	}

	store, err := observation.NewStore(observationsDir())
	if err != nil {
		return nil, nil, nil, fmt.Errorf("init observation store: %w", err)
	}

	libPath := filepath.Join(filepath.Dir(cfg.JSScriptsPath), "js_lib", "utils.js")
	libData, err := os.ReadFile(libPath)
	libCode := ""
	if err != nil {
		log.Printf("[API] ⚠ Failed to load utils library: %v", err)
	} else {
		libCode = string(libData)
	}

	wsConn := ws.NewConnection()
	wsConn.StartServer(cfg.WSPort, cfg.WSToken)

	handler := api.NewHandler(store, wsConn.Execute, wsConn.ExecuteFile, wsConn.Screenshot, wsConn.Tabs, cfg.JSScriptsPath, libCode)
	return handler, cfg, store, nil
}

func runServe() error {
	handler, _, store, err := setup()
	if err != nil {
		return err
	}

	apiMux := http.NewServeMux()
	handler.RegisterRoutes(apiMux)

	httpPort := envOrDefault("HTTP_PORT", "9082")
	// Bind to loopback only — do not expose the HTTP API to the network.
	httpAddr := fmt.Sprintf("127.0.0.1:%s", httpPort)
	log.Printf("[API] HTTP server on %s", httpAddr)
	if dir := store.Dir(); dir != "" {
		log.Printf("[API] Observations dir: %s", dir)
	} else {
		log.Println("[API] Observations: in-memory only (set OBSERVATIONS_DIR to persist)")
	}
	return http.ListenAndServe(httpAddr, apiMux)
}

func runMCP() error {
	log.SetOutput(os.Stderr)
	handler, _, store, err := setup()
	if err != nil {
		return err
	}
	if dir := store.Dir(); dir != "" {
		log.Printf("[MCP] Observations dir: %s", dir)
	} else {
		log.Println("[MCP] Observations: in-memory only (set OBSERVATIONS_DIR to persist)")
	}
	log.Println("[MCP] Starting MCP server on stdio…")
	s := mcpserver.NewServer(handler)
	return mcpserver.ServeStdio(s)
}

func observationsDir() string {
	return os.Getenv("OBSERVATIONS_DIR")
}

func envOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
