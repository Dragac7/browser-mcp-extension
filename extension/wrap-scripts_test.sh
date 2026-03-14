#!/usr/bin/env bash
# Tests for wrap-scripts.sh
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
WRAP_SCRIPT="$SCRIPT_DIR/wrap-scripts.sh"
TMPDIR=$(mktemp -d)
trap 'rm -rf "$TMPDIR"' EXIT

PASS=0
FAIL=0

assert_contains() {
  local file="$1" pattern="$2" test_name="$3"
  if grep -q "$pattern" "$file" 2>/dev/null; then
    echo "PASS: $test_name"
    PASS=$((PASS + 1))
  else
    echo "FAIL: $test_name — expected '$pattern' in $file"
    FAIL=$((FAIL + 1))
  fi
}

assert_not_contains() {
  local file="$1" pattern="$2" test_name="$3"
  if ! grep -q "$pattern" "$file" 2>/dev/null; then
    echo "PASS: $test_name"
    PASS=$((PASS + 1))
  else
    echo "FAIL: $test_name — did not expect '$pattern' in $file"
    FAIL=$((FAIL + 1))
  fi
}

assert_file_exists() {
  local file="$1" test_name="$2"
  if [ -f "$file" ]; then
    echo "PASS: $test_name"
    PASS=$((PASS + 1))
  else
    echo "FAIL: $test_name — file not found: $file"
    FAIL=$((FAIL + 1))
  fi
}

# Setup: create test fixtures
UTILS="$TMPDIR/utils.js"
echo "const wait = (ms) => new Promise(r => setTimeout(r, ms));" > "$UTILS"

SRC1="$TMPDIR/src1"
SRC2="$TMPDIR/src2"
OUTDIR="$TMPDIR/out"
mkdir -p "$SRC1/instagram" "$SRC2/instagram"

# observe.js — IIFE returning a value
cat > "$SRC1/observe.js" <<'EOF'
(function() { return JSON.stringify({url: "test"}); })();
EOF

# interact.js — bare statements with return
cat > "$SRC1/interact.js" <<'EOF'
console.log('interact');
return 'done';
EOF

# instagram subdirectory script
cat > "$SRC1/instagram/click-post.js" <<'EOF'
console.log('click');
return 'clicked';
EOF

# Same file in src2 (should overwrite src1's version)
cat > "$SRC2/interact.js" <<'EOF'
console.log('interact v2');
return 'done v2';
EOF

# ── test_wrap_observe ──────────────────────────────────────────────
bash "$WRAP_SCRIPT" "$UTILS" "$OUTDIR" "$SRC1"

assert_contains "$OUTDIR/observe.js" "try {" "test_wrap_observe: starts with try"
assert_contains "$OUTDIR/observe.js" "window.__scriptResult" "test_wrap_observe: has __scriptResult"
assert_not_contains "$OUTDIR/observe.js" "__scriptParams" "test_wrap_observe: no params wrapper"
assert_not_contains "$OUTDIR/observe.js" "const wait" "test_wrap_observe: no utils"

# ── test_wrap_interact ─────────────────────────────────────────────
assert_contains "$OUTDIR/interact.js" "window.__scriptParams" "test_wrap_interact: has __scriptParams"
assert_contains "$OUTDIR/interact.js" "const wait" "test_wrap_interact: has utils content"
assert_contains "$OUTDIR/interact.js" "(async () =>" "test_wrap_interact: has async IIFE"

# ── test_preserves_subdir ──────────────────────────────────────────
assert_file_exists "$OUTDIR/instagram/click-post.js" "test_preserves_subdir: instagram/click-post.js exists"

# ── test_overwrite_later_dir ───────────────────────────────────────
# Re-run with both source dirs (src2 should overwrite src1's interact.js)
bash "$WRAP_SCRIPT" "$UTILS" "$OUTDIR" "$SRC1" "$SRC2"
assert_contains "$OUTDIR/interact.js" "interact v2" "test_overwrite_later_dir: src2 content wins"

# ── Summary ────────────────────────────────────────────────────────
echo ""
echo "Results: $PASS passed, $FAIL failed"
if [ "$FAIL" -gt 0 ]; then
  exit 1
fi
