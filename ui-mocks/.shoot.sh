#!/usr/bin/env bash
# 用无头 Chrome 在桌面宽度下给 4 个 UI 稿页面截图，验证渲染
set -euo pipefail
BASE="/Users/steven/docker/dockermanage/ui-mocks"
OUT="$BASE/screenshots"
CHROME="/Applications/Google Chrome.app/Contents/MacOS/Google Chrome"
mkdir -p "$OUT"
rm -f "$OUT"/*.png

WIDTH="${1:-1440}"
HEIGHT="${2:-900}"

for page in index containers container-detail compose; do
  "$CHROME" --headless=new --disable-gpu --hide-scrollbars \
    --virtual-time-budget=6000 \
    --window-size="${WIDTH},${HEIGHT}" \
    --screenshot="$OUT/${page}-${WIDTH}.png" \
    "file://${BASE}/${page}.html" >/dev/null 2>&1
  echo "✓ ${page}.html → ${OUT}/${page}-${WIDTH}.png"
done
