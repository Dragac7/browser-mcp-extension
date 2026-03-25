<!-- SACRED DOCUMENT — DO NOT MODIFY except for checkmarks ([ ] → [x]) and review findings. -->
<!-- You MUST NEVER alter, revert, or delete files outside the scope of this plan. -->
<!-- Plans in docs/plans/ are PERMANENT artifacts. There are ZERO exceptions. -->

# File Upload via DataTransfer API

## User Story 1: Self-contained upload test page

Replace the scraped Ashby SPA (non-functional locally) with a self-contained HTML test fixture that exercises all upload scenarios the e2e tests need.

**Acceptance criteria:**
- [x] `test/e2e/testdata/pages/upload.html` renders standalone (no external CDN/JS deps)
- [x] Contains: single file `<input type="file">`, multiple file `<input type="file" multiple>`, accept-restricted input (`accept=".pdf,.txt"`), drag-and-drop zone
- [x] Each input displays chosen filename(s) on change via JS `change` event listener
- [x] A status `<div>` shows upload results for verification by e2e tests

### Task 1.1: Create upload test page

**Action 1** — `test/e2e/testdata/pages/upload.html` — **replace**

```html
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <title>E2E Upload Page</title>
</head>
<body>
    <h1>File Upload Test</h1>

    <!-- Single file input -->
    <label for="single-file">Single file:</label>
    <input type="file" id="single-file" aria-label="Single file">
    <p id="single-status">No file selected</p>

    <!-- Multiple file input -->
    <label for="multi-file">Multiple files:</label>
    <input type="file" id="multi-file" multiple aria-label="Multiple files">
    <p id="multi-status">No files selected</p>

    <!-- Accept-restricted input -->
    <label for="restricted-file">PDF/TXT only:</label>
    <input type="file" id="restricted-file" accept=".pdf,.txt" aria-label="PDF or TXT only">
    <p id="restricted-status">No file selected</p>

    <!-- Drag-and-drop zone -->
    <div id="drop-zone" style="border:2px dashed #999;padding:40px;text-align:center;margin:20px 0;"
         aria-label="Drop zone">
        Drop files here
    </div>
    <p id="drop-status">No files dropped</p>

    <script>
        function showFiles(input, statusId) {
            var el = document.getElementById(statusId);
            var names = Array.from(input.files).map(function(f) { return f.name + '(' + f.size + 'b)'; });
            el.textContent = names.length > 0 ? 'Selected: ' + names.join(', ') : 'No file selected';
        }

        document.getElementById('single-file').addEventListener('change', function() {
            showFiles(this, 'single-status');
        });
        document.getElementById('multi-file').addEventListener('change', function() {
            showFiles(this, 'multi-status');
        });
        document.getElementById('restricted-file').addEventListener('change', function() {
            showFiles(this, 'restricted-status');
        });

        var dropZone = document.getElementById('drop-zone');
        dropZone.addEventListener('dragover', function(e) { e.preventDefault(); });
        dropZone.addEventListener('drop', function(e) {
            e.preventDefault();
            var names = Array.from(e.dataTransfer.files).map(function(f) { return f.name; });
            document.getElementById('drop-status').textContent = names.length > 0
                ? 'Dropped: ' + names.join(', ')
                : 'No files dropped';
        });
    </script>
</body>
</html>
```

**DoD:**
- [x] Page renders standalone in browser
- [x] All four inputs visible, JS event listeners fire on file selection/drop

---

## User Story 2: Upload JS script (DataTransfer API)

Create `upload.js` that uses the DataTransfer API to programmatically set files on `<input type="file">` elements. The MCP caller provides base64-encoded file content + filename; the script creates `File` objects and attaches them via `DataTransfer`.

**Why:** Browser security prevents setting `.value` or `.files` on file inputs directly. DataTransfer API is the only non-debugger approach that works in Chromium's MAIN world.

**Acceptance criteria:**
- [x] Accepts `elementIndex` (required), `files` array (required) where each entry has `name` (string) and `content` (base64-encoded string), optional `mimeType` (string, default `application/octet-stream`)
- [x] Creates `File` objects from decoded base64, sets them on the input via `DataTransfer`, dispatches `input` + `change` events with bubbles so React/framework handlers fire
- [x] Returns descriptive status string on success, `Error: ...` string on failure
- [x] Validates: element exists, element is `<input type="file">`, files array non-empty, base64 decoding succeeds, individual file size ≤ 10 MB

### Task 2.1: Create upload.js script

**Action 1** — `resources/js_scripts/upload.js` — **create**

```javascript
// upload.js — Set files on <input type="file"> via DataTransfer API.
//
// Params:
//   elementIndex — numeric index from the last observe.js snapshot
//   files        — array of {name: string, content: string (base64), mimeType?: string}
//
// Returns a status string on success.

const elements = window.__observedElements;
if (!elements || !Array.isArray(elements)) {
  return 'Error: No observed elements found. Run observe.js first.';
}

const idx = params.elementIndex;
if (idx === undefined || idx === null) {
  return 'Error: elementIndex is required';
}
if (idx < 0 || idx >= elements.length) {
  return `Error: elementIndex ${idx} out of range (0-${elements.length - 1})`;
}

const el = elements[idx];
if (!el) {
  return `Error: Element at index ${idx} is no longer in the DOM`;
}
if (!document.contains(el)) {
  return `Error: Element at index ${idx} has been removed from the DOM`;
}
if (el.tagName.toLowerCase() !== 'input' || el.type !== 'file') {
  return `Error: Element at index ${idx} is not a file input (found <${el.tagName.toLowerCase()} type="${el.type || ''}">)`;
}

const filesParam = params.files;
if (!Array.isArray(filesParam) || filesParam.length === 0) {
  return 'Error: "files" must be a non-empty array of {name, content, mimeType?}';
}

// Fail fast: check multiple attribute before processing files
if (filesParam.length > 1 && !el.multiple) {
  return `Error: Element at index ${idx} does not have the "multiple" attribute but ${filesParam.length} files were provided`;
}

const MAX_BASE64_LENGTH = 14 * 1024 * 1024; // ~10 MB decoded
const dt = new DataTransfer();
const fileNames = [];

for (let i = 0; i < filesParam.length; i++) {
  const f = filesParam[i];
  if (!f.name || typeof f.name !== 'string') {
    return `Error: files[${i}].name is required and must be a string`;
  }
  if (!f.content || typeof f.content !== 'string') {
    return `Error: files[${i}].content is required and must be a base64 string`;
  }

  // Guard: reject individual files > 10 MB (base64 ≈ 1.37× raw size, so 10 MB raw ≈ 13.7 MB base64)
  if (f.content.length > MAX_BASE64_LENGTH) {
    return `Error: files[${i}].content exceeds 10 MB limit (${Math.round(f.content.length / 1024 / 1024)} MB base64)`;
  }

  let bytes;
  try {
    const binary = atob(f.content);
    bytes = Uint8Array.from(binary, c => c.charCodeAt(0));
  } catch (e) {
    return `Error: files[${i}].content is not valid base64: ${e.message}`;
  }

  const mimeType = f.mimeType || 'application/octet-stream';
  const file = new File([bytes], f.name, { type: mimeType });
  dt.items.add(file);
  fileNames.push(f.name);
}

el.files = dt.files;

// Dispatch events so frameworks (React, Vue, etc.) detect the change
el.dispatchEvent(new Event('input', { bubbles: true }));
el.dispatchEvent(new Event('change', { bubbles: true }));

const desc = el.getAttribute('aria-label') || el.id || 'file input';
return `Uploaded ${fileNames.length} file(s) [${fileNames.join(', ')}] to element #${idx} (${desc})`;
```

**DoD:**
- [x] Script follows existing patterns (element lookup via `__observedElements`, params validation, status string return)
- [x] DataTransfer + File construction + event dispatch implemented

---

## User Story 3: MCP tool `browser_upload`

Wire the upload script into the MCP tool layer so AI agents can call `browser_upload`.

**Acceptance criteria:**
- [x] `browser_upload` MCP tool registered with `elementIndex` (required number), `files` (required array)
- [x] Handler validates inputs, delegates to `h.ExecuteScriptAndObserve("upload.js", params)`
- [x] Returns `{success, data, error, snapshot}` consistent with other interaction tools
- [x] Unit tests for `handleUpload` handler

### Task 3.1: Register tool and add handler

**Action 1** — `internal/mcp/tools.go` — **modify**: add tool registration after `browser_fill_form` (after line 81)

```go
s.AddTool(mcp.NewTool("browser_upload",
    mcp.WithDescription("Upload file(s) to a <input type=\"file\"> element by its index from the last browser_snapshot. Files are provided as base64-encoded content."),
    mcp.WithNumber("elementIndex", mcp.Required(), mcp.Description("Element index of the file input from the last browser_snapshot")),
    mcp.WithArray("files", mcp.Required(), mcp.Description("Array of {name: string, content: string (base64), mimeType?: string}")),
), handleUpload(h))
```

**Action 2** — `internal/mcp/tools.go` — **modify**: add `handleUpload` function (after `handleFillForm`, around line 346)

```go
func handleUpload(h *api.Handler) func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := req.GetArguments()
		idxF, ok := args["elementIndex"].(float64)
		if !ok {
			return mcp.NewToolResultError("elementIndex is required"), nil
		}
		files, ok := args["files"].([]interface{})
		if !ok || len(files) == 0 {
			return mcp.NewToolResultError("files must be a non-empty array"), nil
		}
		ok2, data, errMsg, snap, err := h.ExecuteScriptAndObserve("upload.js", map[string]interface{}{
			"elementIndex": int(idxF),
			"files":        files,
		})
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("upload failed: %v", err)), nil
		}
		return toolResult(map[string]interface{}{"success": ok2, "data": data, "error": errMsg, "snapshot": snap}), nil
	}
}
```

**DoD:**
- [x] `upload.js` will appear in `browser_list_scripts` — this is consistent with other dedicated tools that also have underlying scripts (e.g., `fill_form.js`, `select_option.js`)
- [x] `handleUpload` follows `handleFillForm` / `handleSelectOption` pattern exactly

### Task 3.2: Update tool count assertion and architecture doc

**Action 1** — `internal/mcp/server_test.go` — **modify**: update `TestNewServerRegisters18Tools` → `TestNewServerRegisters19Tools`, change expected count from 18 to 19

**Action 2** — `docs/ARCHITECTURE.md` — **modify**: update line 32 `tools.go` description from `(18 tools)` to `(19 tools)`

**DoD:**
- [x] Test name and assertion reflect 19 tools
- [x] Architecture doc reflects 19 tools

### Task 3.3: Unit tests for handleUpload

**File**: `internal/mcp/tools_upload_test.go` — **create** in `package mcpserver` (internal test file, NOT `mcpserver_test`)

Using `package mcpserver` gives direct access to the unexported `handleUpload` function. This avoids the complexity of testing through MCP server dispatch (which requires stdio transport).

Setup: build an `*api.Handler` using the same pattern as `buildTestHandler` in `server_test.go` — create `observation.Store`, stub function types, temp scripts dir. For `TestHandleUpload_Success`: create dummy `upload.js` and `observe.js` files in the temp scripts dir so `ExecuteScript` file reads succeed. For `TestHandleUpload_ExecuteError`: provide an `executeFile` stub that returns an error. Call `handleUpload(h)` directly to get the handler func, then invoke it with a crafted `mcp.CallToolRequest`.

| Test | Verifies |
|------|----------|
| `TestHandleUpload_MissingElementIndex` | Missing elementIndex → tool result IsError, text contains "elementIndex is required" |
| `TestHandleUpload_MissingFiles` | Missing files → tool result IsError, text contains "files must be a non-empty array" |
| `TestHandleUpload_EmptyFiles` | Empty files array → tool result IsError, text contains "files must be a non-empty array" |
| `TestHandleUpload_Success` | Valid elementIndex + files + dummy scripts → returns non-error result |
| `TestHandleUpload_ExecuteError` | `executeFile` stub returns error → tool result IsError, text contains "upload failed" |

**DoD:**
- [x] All unit tests pass
- [x] Handler arg validation + success + error execution paths covered

---

## User Story 4: E2E tests for file upload

**Acceptance criteria:**
- [x] E2E tests cover: single file upload, multiple file upload, invalid element index, non-file-input element, invalid base64 content
- [x] Tests use the new `upload.html` test page
- [x] Tests verify both tool result and page state (via snapshot visible text)

### Task 4.1: Create upload e2e tests

**File**: `test/e2e/upload_test.go`

| Test | Verifies | Notes |
|------|----------|-------|
| `TestUpload_SingleFile` | Single file → input.files set, page shows "Selected: test.txt(11b)" | base64 of "hello world" (11 bytes) |
| `TestUpload_MultipleFiles` | Two files on `multiple` input → page shows both filenames with sizes | Use multi-file input |
| `TestUpload_InvalidIndex` | elementIndex 9999 → data contains "out of range" | |
| `TestUpload_NonFileInput` | elementIndex pointing to a non-file element → data contains "not a file input" | Use the `<h1>` or `<label>` element |
| `TestUpload_InvalidBase64` | Malformed base64 → data contains "not valid base64" | |
| `TestUpload_EmptyFilesArray` | Empty files array → tool error | Uses `callToolExpectError` |
| `TestUpload_MultipleFilesOnSingleInput` | 2 files on single (non-multiple) input → data contains "multiple" error | |
| `TestUpload_AcceptRestrictedInput` | Upload .txt file to accept=".pdf,.txt" input → succeeds, page shows filename | DataTransfer API does not enforce accept client-side; verifies no unexpected rejection |

**DoD:**
- [x] All e2e tests pass
- [x] Tests cover happy path + error cases

---

## Cleanup: Remove POC files

After all user stories are implemented, remove the proof-of-concept files that were created during validation:

- [x] Delete `test/e2e/testdata/pages/upload_poc.html`
- [x] Delete `test/e2e/upload_poc_test.go`

Note: `resources/js_scripts/upload.js` is overwritten by Task 2.1 (not a POC leftover).

---

## Quality Gates (run after ALL user stories implemented AND POC cleanup done)

Run from module root (`/Users/paoloandrisani/Personal/automation-ext/browser-mcp-extension/`):

- [x] `go vet ./...` — no warnings/errors
- [x] `gofmt -l .` — no output
- [x] `go test ./...` — all unit tests pass (including updated 19-tool count)
- [x] `make sync-scripts` — upload.js wrapped into `extension/scripts/`
- [x] E2E tests: `cd test/e2e && go test -v -count=1 -timeout 10m -p 1 ./...` — all pass
