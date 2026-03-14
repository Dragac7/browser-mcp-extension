#!/usr/bin/env bash
# wrap-scripts.sh — Wraps JS scripts for chrome.scripting.executeScript({ files }).
#
# Usage: bash wrap-scripts.sh <utils.js> <outdir> <srcdir1> [srcdir2 ...]
#
# observe.js → sync wrapper (entire file embedded as expression in __scriptResult)
# All others → async wrapper with utils prepended and params from __scriptParams

set -euo pipefail

if [ $# -lt 3 ]; then
  echo "Usage: $0 <utils.js> <outdir> <srcdir1> [srcdir2 ...]" >&2
  exit 1
fi

UTILS_FILE="$1"
OUTDIR="$2"
shift 2

if [ ! -f "$UTILS_FILE" ]; then
  echo "Error: utils file not found: $UTILS_FILE" >&2
  exit 1
fi

UTILS_CONTENT=$(cat "$UTILS_FILE")

# Process each source directory in order (later dirs overwrite earlier)
for srcdir in "$@"; do
  if [ ! -d "$srcdir" ]; then
    echo "Warning: source directory not found, skipping: $srcdir" >&2
    continue
  fi

  # Find all .js files recursively
  while IFS= read -r -d '' file; do
    # Compute relative path from srcdir
    relpath="${file#"$srcdir"/}"
    basename=$(basename "$relpath")
    outfile="$OUTDIR/$relpath"

    # Ensure output subdirectory exists
    mkdir -p "$(dirname "$outfile")"

    if [ "$basename" = "observe.js" ]; then
      # Sync wrapper: entire file content embedded as expression
      # The IIFE executes and its return value becomes data
      file_content=$(cat "$file")
      # Strip leading comment and blank lines so the IIFE is the first token
      file_content_stripped=$(echo "$file_content" | grep -v '^[[:space:]]*//' | grep -v '^[[:space:]]*$')
      cat > "$outfile" <<WRAPPER_EOF
try {
  var __obs = ${file_content_stripped}
  window.__scriptResult = { success: true, data: __obs };
} catch (e) {
  window.__scriptResult = { success: false, error: e.message };
}
WRAPPER_EOF
    else
      # Async wrapper: utils + params + script content inside async IIFE
      file_content=$(cat "$file")
      cat > "$outfile" <<WRAPPER_EOF
(async () => {
  try {
    const params = window.__scriptParams || {};
    delete window.__scriptParams;
    ${UTILS_CONTENT}
    const __result = await (async function(params) {
      ${file_content}
    })(params);
    window.__scriptResult = { success: true, data: __result };
  } catch (e) {
    window.__scriptResult = { success: false, error: e.message };
  }
})();
WRAPPER_EOF
    fi
  done < <(find "$srcdir" -name '*.js' -print0)
done

echo "Scripts wrapped to $OUTDIR"
