package mcpserver

import (
	"github.com/paoloandrisani/browser-mcp-extension/internal/api"

	"github.com/mark3labs/mcp-go/server"
)

// NewServer creates a fully configured MCP server.
func NewServer(h *api.Handler) *server.MCPServer {
	s := server.NewMCPServer(
		"browser-mcp-extension",
		"1.0.0",
		server.WithToolCapabilities(true),
		server.WithResourceCapabilities(true, false),
	)
	registerTools(s, h)
	registerResources(s, h)
	return s
}

// ServeStdio starts the MCP server over stdin/stdout.
func ServeStdio(s *server.MCPServer) error {
	return server.ServeStdio(s)
}
