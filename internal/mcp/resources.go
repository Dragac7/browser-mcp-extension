package mcpserver

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/paoloandrisani/browser-mcp-extension/internal/api"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func registerResources(s *server.MCPServer, h *api.Handler) {
	s.AddResource(
		mcp.NewResource(
			"page://snapshot/latest",
			"Latest Page Snapshot",
			mcp.WithResourceDescription("The most recent structured observation of the browser page. Contains interactive elements with indices, visible text, and page sections."),
			mcp.WithMIMEType("application/json"),
		),
		func(ctx context.Context, req mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
			snap, err := h.GetState()
			if err != nil {
				return nil, fmt.Errorf("get state: %w", err)
			}
			if snap == nil {
				return []mcp.ResourceContents{
					mcp.TextResourceContents{
						URI:      "page://snapshot/latest",
						MIMEType: "application/json",
						Text:     `{"status":"no snapshot yet","hint":"call browser_snapshot first"}`,
					},
				}, nil
			}
			data, err := json.MarshalIndent(snap, "", "  ")
			if err != nil {
				return nil, fmt.Errorf("marshal snapshot: %w", err)
			}
			return []mcp.ResourceContents{
				mcp.TextResourceContents{
					URI:      "page://snapshot/latest",
					MIMEType: "application/json",
					Text:     string(data),
				},
			}, nil
		},
	)
}
