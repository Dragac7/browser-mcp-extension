package api

// ExecuteFn is the type for the WebSocket JS execution callback.
type ExecuteFn func(code string) (success bool, data interface{}, errMsg string, err error)

// ExecuteFileFn is the type for file-based script execution with params and code fallback.
type ExecuteFileFn func(scriptFile, code string, params map[string]interface{}) (success bool, data interface{}, errMsg string, err error)

// ScreenshotFn is the callback to take a screenshot via the extension.
type ScreenshotFn func() (dataURL string, err error)

// TabsFn is the callback to manage browser tabs via the extension.
type TabsFn func(action string, index *int, url string) (success bool, data interface{}, errMsg string, err error)

// ScriptEntry represents a discovered automation script.
type ScriptEntry struct {
	Name string `json:"name"`
	Path string `json:"path"`
}
