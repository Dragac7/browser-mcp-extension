package ws

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"sync/atomic"
	"time"
	"unsafe"

	"github.com/gorilla/websocket"
)

// ReadTimeout is the maximum time to wait for a response from the extension.
// Set conservatively to cover long-running scripts (e.g. wait_for.js with 60s timeout).
const ReadTimeout = 90 * time.Second

// Message is the JSON envelope exchanged over the WebSocket.
type Message struct {
	Type       string                 `json:"type"`
	Code       string                 `json:"code,omitempty"`
	Action     string                 `json:"action,omitempty"`
	Index      *int                   `json:"index,omitempty"`
	URL        string                 `json:"url,omitempty"`
	ScriptFile string                 `json:"scriptFile,omitempty"`
	Params     map[string]interface{} `json:"params,omitempty"`
	Success    bool                   `json:"success,omitempty"`
	Data       interface{}            `json:"data,omitempty"`
	Error      string                 `json:"error,omitempty"`
	Timestamp  string                 `json:"timestamp,omitempty"`
}

// Connection manages a single WebSocket connection to the Chrome extension.
type Connection struct {
	conn *websocket.Conn
	mu   sync.Mutex
	// ready is accessed via atomic pointer operations to avoid data races
	// during reconnection cycles (StartServer goroutine vs. Ready() callers).
	ready    unsafe.Pointer // *chan struct{}
	disc     chan struct{}
	upgrader websocket.Upgrader
}

func NewConnection() *Connection {
	c := &Connection{
		disc: make(chan struct{}),
		upgrader: websocket.Upgrader{
			// Allow chrome-extension:// and any origin (required for MV3 service workers).
			CheckOrigin: func(r *http.Request) bool { return true },
		},
	}
	ch := make(chan struct{})
	atomic.StorePointer(&c.ready, unsafe.Pointer(&ch))
	return c
}

// sendLocked marshals and sends a message, then reads the response.
// Caller MUST hold c.mu. Sets a read deadline to prevent indefinite blocking.
func (c *Connection) sendLocked(msg Message) (success bool, data interface{}, errMsg string, err error) {
	if c.conn == nil {
		return false, nil, "", fmt.Errorf("no WebSocket connection")
	}
	msgJSON, err := json.Marshal(msg)
	if err != nil {
		return false, nil, "", fmt.Errorf("marshal: %w", err)
	}
	if err := c.conn.WriteMessage(websocket.TextMessage, msgJSON); err != nil {
		c.closeLocked()
		return false, nil, "", fmt.Errorf("send: %w", err)
	}
	c.conn.SetReadDeadline(time.Now().Add(ReadTimeout))
	_, response, err := c.conn.ReadMessage()
	c.conn.SetReadDeadline(time.Time{}) // reset after read
	if err != nil {
		c.closeLocked()
		return false, nil, "", fmt.Errorf("read: %w", err)
	}
	var resp Message
	if err := json.Unmarshal(response, &resp); err != nil {
		return false, nil, "", fmt.Errorf("unmarshal response: %w", err)
	}
	return resp.Success, resp.Data, resp.Error, nil
}

// closeLocked signals disconnection. Caller MUST hold c.mu.
func (c *Connection) closeLocked() {
	select {
	case <-c.disc:
	default:
		close(c.disc)
	}
}

// Execute sends JS code to the extension and waits for the result.
func (c *Connection) Execute(code string) (success bool, data interface{}, errMsg string, err error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.sendLocked(Message{Type: "execute", Code: code})
}

// ExecuteFile sends a file-based execution request with params and code fallback.
func (c *Connection) ExecuteFile(scriptFile, code string, params map[string]interface{}) (success bool, data interface{}, errMsg string, err error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.sendLocked(Message{Type: "execute", Code: code, ScriptFile: scriptFile, Params: params})
}

// Screenshot asks the extension to capture a screenshot of the active tab.
// Returns the PNG data URL (e.g. "data:image/png;base64,...").
func (c *Connection) Screenshot() (dataURL string, err error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	ok, data, errMsg, err := c.sendLocked(Message{Type: "screenshot"})
	if err != nil {
		return "", err
	}
	if !ok {
		return "", fmt.Errorf("screenshot failed: %s", errMsg)
	}
	url, ok2 := data.(string)
	if !ok2 {
		return "", fmt.Errorf("screenshot: unexpected data type %T", data)
	}
	return url, nil
}

// Tabs sends a tabs management command to the extension.
// action: "list" | "create" | "close" | "select"
// index: tab index (required for close/select; uses positional index from list)
// url: target URL (optional, used for create; must be http/https)
//
// Note: tab indices are positional within the current chrome.tabs.query({}) result.
// A race exists if tabs are opened/closed between a list call and a close/select call.
// This is a known limitation of the positional-index design.
func (c *Connection) Tabs(action string, index *int, url string) (success bool, data interface{}, errMsg string, err error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.sendLocked(Message{Type: "tabs", Action: action, Index: index, URL: url})
}

// Ready returns a channel that is closed when the extension connects.
func (c *Connection) Ready() <-chan struct{} {
	chPtr := (*chan struct{})(atomic.LoadPointer(&c.ready))
	return *chPtr
}

// StartServer starts the WebSocket server, binding to localhost only.
// If token is non-empty, the extension must provide it as a ?token= query parameter.
func (c *Connection) StartServer(port int, token string) {
	mux := http.NewServeMux()
	var connected bool
	var connMu sync.Mutex

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// Validate bearer token when configured.
		if token != "" && r.URL.Query().Get("token") != token {
			log.Println("[WS] Rejected connection: invalid or missing token")
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		connMu.Lock()
		if connected {
			connMu.Unlock()
			log.Println("[WS] Rejected duplicate extension connection (HTTP 409)")
			http.Error(w, "already connected", http.StatusConflict)
			return
		}
		connected = true
		connMu.Unlock()

		conn, err := c.upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Printf("[WS] Upgrade failed: %v", err)
			connMu.Lock()
			connected = false
			connMu.Unlock()
			return
		}
		log.Println("[WS] Extension connected")

		_, _, err = conn.ReadMessage()
		if err != nil {
			log.Printf("[WS] Handshake error: %v", err)
			conn.Close()
			connMu.Lock()
			connected = false
			connMu.Unlock()
			return
		}

		c.mu.Lock()
		c.conn = conn
		c.disc = make(chan struct{})
		c.mu.Unlock()

		// Atomically replace the ready channel and signal it.
		// Using atomic pointer swap avoids a data race between this goroutine
		// and any concurrent Ready() callers.
		newReady := make(chan struct{})
		oldPtr := atomic.SwapPointer(&c.ready, unsafe.Pointer(&newReady))
		oldCh := *(*chan struct{})(oldPtr)
		// Close the old channel only if it hasn't been closed yet.
		select {
		case <-oldCh:
		default:
			close(oldCh)
		}
		log.Println("[WS] Extension ready")

		<-c.disc
		log.Println("[WS] Extension disconnected, waiting for reconnection…")
		connMu.Lock()
		connected = false
		connMu.Unlock()
	})

	// Bind to loopback only — do not expose the WS port to the network.
	addr := fmt.Sprintf("127.0.0.1:%d", port)
	log.Printf("[WS] Server on %s (for extension)", addr)
	log.Println("[WS] Waiting for extension to connect…")

	go func() {
		if err := http.ListenAndServe(addr, mux); err != nil {
			log.Fatalf("[WS] Server failed: %v", err)
		}
	}()
}
