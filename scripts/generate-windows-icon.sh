#!/usr/bin/env bash
set -euo pipefail

# Generate a Windows .syso resource from Designer.ico if present.
# Tries rsrc first, then windres. Writes resources.syso in the current dir.

if [ -f Designer.ico ]; then
  echo "Designer.ico found — generating windows resource .syso"
  if command -v rsrc >/dev/null 2>&1; then
    rsrc -ico Designer.ico -o resources.syso || { echo "rsrc failed"; exit 1; }
  elif command -v windres >/dev/null 2>&1; then
    cat > _icon.rc <<'EOF'
1 ICON "Designer.ico"
EOF
    windres _icon.rc resources.syso || { echo "windres failed"; rm -f _icon.rc; exit 1; }
    rm -f _icon.rc
  else
    echo "warning: neither rsrc nor windres found; windows icon won't be embedded"
  fi
else
  echo "Designer.ico not present; skipping windows resource generation"
fi
