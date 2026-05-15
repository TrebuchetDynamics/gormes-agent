#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
OUT="${1:-$ROOT_DIR/webpages/docs/assets/gormes-tui-demo.gif}"
WIDTH=1280
HEIGHT=720
FONT="DejaVu-Sans-Mono"
FONT_BOLD="DejaVu-Sans-Mono-Bold"

if ! command -v convert >/dev/null 2>&1; then
  echo "ImageMagick convert is required to render the README demo GIF." >&2
  exit 1
fi

TMP_DIR="$(mktemp -d)"
trap 'rm -rf "$TMP_DIR"' EXIT

escape_draw_text() {
  local value="$1"
  value="${value//\\/\\\\}"
  value="${value//\'/\\\'}"
  printf '%s' "$value"
}

draw_text() {
  local x="$1"
  local y="$2"
  local size="$3"
  local color="$4"
  local font="$5"
  local text="$6"
  cmd+=(-font "$font" -pointsize "$size" -fill "$color" -draw "text $x,$y '$(escape_draw_text "$text")'")
}

draw_chip() {
  local x="$1"
  local y="$2"
  local w="$3"
  local label="$4"
  local stroke="$5"
  local fill="$6"
  cmd+=(-fill "$fill" -stroke "$stroke" -strokewidth 1 -draw "roundrectangle $x,$y $((x + w)),$((y + 34)) 14,14")
  draw_text "$((x + 16))" "$((y + 23))" 17 "#e5f2ff" "$FONT_BOLD" "$label"
}

line_color() {
  local line="$1"
  case "$line" in
    '$ '*|'> '*) printf '#7dd3fc' ;;
    PASS*|READY*|OK*) printf '#86efac' ;;
    WARN*) printf '#fde68a' ;;
    tool*|engine*|selector*|backend*|memory*|route*|workspace*|provider*|model*|secret*|config*|session*|channel*) printf '#c4b5fd' ;;
    result*|assistant*|gateway*|saved*|reply*) printf '#f8fafc' ;;
    *) printf '#cbd5e1' ;;
  esac
}

draw_terminal_lines() {
  local x="$1"
  local y="$2"
  local gap="$3"
  local -n lines_ref="$4"
  local line
  for line in "${lines_ref[@]}"; do
    draw_text "$x" "$y" 20 "$(line_color "$line")" "$FONT" "$line"
    y=$((y + gap))
  done
}

draw_bullets() {
  local x="$1"
  local y="$2"
  local -n bullets_ref="$3"
  local bullet
  for bullet in "${bullets_ref[@]}"; do
    cmd+=(-fill "#38bdf8" -stroke none -draw "circle $x,$((y - 7)) $((x + 5)),$((y - 7))")
    draw_text "$((x + 18))" "$y" 18 "#dbeafe" "$FONT" "$bullet"
    y=$((y + 42))
  done
}

render_frame() {
  local idx="$1"
  local eyebrow="$2"
  local headline="$3"
  local panel_title="$4"
  local accent="$5"
  local progress="$6"
  local -n left_ref="$7"
  local -n right_ref="$8"
  local frame
  frame="$(printf '%s/frame_%02d.png' "$TMP_DIR" "$idx")"

  cmd=(convert -size "${WIDTH}x${HEIGHT}" "gradient:#07111f-#111827")
  cmd+=(-fill "#08111f" -draw "rectangle 0,0 ${WIDTH},${HEIGHT}")
  cmd+=(-fill "#0f2133" -draw "rectangle 0,0 ${WIDTH},108")
  cmd+=(-fill "#102a3f" -draw "rectangle 0,108 ${WIDTH},110")
  cmd+=(-fill "#0b1626" -draw "rectangle 0,610 ${WIDTH},${HEIGHT}")

  for x in 0 80 160 240 320 400 480 560 640 720 800 880 960 1040 1120 1200 1280; do
    cmd+=(-stroke "#10243a" -strokewidth 1 -draw "line $x,110 $x,610")
  done
  for y in 150 190 230 270 310 350 390 430 470 510 550 590; do
    cmd+=(-stroke "#10243a" -strokewidth 1 -draw "line 0,$y ${WIDTH},$y")
  done

  draw_text 56 68 42 "#f8fafc" "$FONT_BOLD" "GORMES"
  draw_text 276 67 21 "#93c5fd" "$FONT" "$eyebrow"
  draw_text 56 140 34 "#f8fafc" "$FONT_BOLD" "$headline"

  draw_chip 980 38 96 "Go" "#38bdf8" "#0b2033"
  draw_chip 1092 38 136 "Termux" "#22c55e" "#0b241b"
  draw_chip 76 628 178 "No Python" "#38bdf8" "#0b2033"
  draw_chip 270 628 160 "No Docker" "#818cf8" "#151735"
  draw_chip 446 628 210 "SQLite state" "#22c55e" "#0b241b"
  draw_chip 672 628 190 "web_* tools" "#f59e0b" "#251a08"

  cmd+=(-fill "#07131f" -stroke "#36516a" -strokewidth 2 -draw "roundrectangle 56,176 842,590 20,20")
  cmd+=(-fill "#13253a" -stroke none -draw "roundrectangle 56,176 842,226 20,20")
  cmd+=(-fill "#ef4444" -draw "circle 86,202 94,202")
  cmd+=(-fill "#f59e0b" -draw "circle 114,202 122,202")
  cmd+=(-fill "#22c55e" -draw "circle 142,202 150,202")
  draw_text 184 209 18 "#dbeafe" "$FONT_BOLD" "$panel_title"
  draw_terminal_lines 84 264 38 left_ref

  cmd+=(-fill "#0b1728" -stroke "$accent" -strokewidth 2 -draw "roundrectangle 880,176 1224,590 20,20")
  draw_text 914 226 24 "#f8fafc" "$FONT_BOLD" "User benefit"
  draw_bullets 914 282 right_ref

  cmd+=(-fill "#172235" -stroke none -draw "roundrectangle 76,680 1204,696 8,8")
  local filled=$((76 + (1128 * progress / 100)))
  cmd+=(-fill "$accent" -stroke none -draw "roundrectangle 76,680 $filled,696 8,8")
  draw_text 76 668 16 "#94a3b8" "$FONT" "install"
  draw_text 232 668 16 "#94a3b8" "$FONT" "setup"
  draw_text 414 668 16 "#94a3b8" "$FONT" "provider"
  draw_text 604 668 16 "#94a3b8" "$FONT" "first turn"
  draw_text 824 668 16 "#94a3b8" "$FONT" "tools"
  draw_text 1040 668 16 "#94a3b8" "$FONT" "gateway"

  cmd+=("$frame")
  "${cmd[@]}"
}

frame01_left=(
  '$ curl -fsSL <release install.sh> | sh'
  'READY fetched verified release binary'
  'READY installed to ~/.local/bin/gormes'
  'READY PATH hint written for this shell'
  '> next: gormes setup'
)
frame01_right=(
  'One command start'
  'No pip or venv'
  'User-scoped install'
  'Termux uses same path'
)

frame02_left=(
  '$ gormes setup'
  'Quick setup - provider, model, and messaging'
  'Full setup - configure everything'
  'Terminal target: local TUI and chat queries'
  'config: ~/.gormes/config.toml'
  '> next: gormes setup provider'
)
frame02_right=(
  'Canonical setup'
  'Explains missing deps'
  'Creates local config'
  'Direct shortcuts later'
)

frame03_left=(
  '$ gormes setup provider'
  'provider: OpenAI / Anthropic / Groq / Ollama'
  'model: choose default for this profile'
  'secret: saved to ~/.gormes/.env'
  'config: active profile updated'
  '> next: gormes chat'
)
frame03_right=(
  'Provider in minutes'
  'Secrets stay local'
  'Profiles are explicit'
  'No config guessing'
)

frame04_left=(
  '$ gormes chat'
  'session: created in ~/.gormes/sessions.db'
  'provider: configured profile'
  'assistant: ready to work in this directory'
  '> user: help me understand this repo'
  'reply: I can read files, run tools, and report back.'
)
frame04_right=(
  'First real answer'
  'Credentials verified'
  'Session persisted'
  'Terminal-native loop'
)

frame05_left=(
  '$ cd ~/projects/app'
  '$ gormes chat -q "summarize this repo"'
  'tool: list_files, ripgrep, read_file'
  'result: README, tests, build scripts detected'
  'assistant: next action plan ready'
)
frame05_right=(
  'Useful immediately'
  'Works in real repos'
  'PC-like shell flow'
  'No IDE required'
)

frame06_left=(
  '$ gormes chat -q "extract setup docs"'
  '> url: https://example.test/install'
  'tool: web_extract'
  'engine: goscrapling static fetch'
  'result: install steps + links returned'
  'assistant: source-backed summary ready'
)
frame06_right=(
  'web_* in the loop'
  'Local fetch fallback'
  'Private URL guards'
  'No hosted scraper needed'
)

frame07_left=(
  '$ pkg install git golang tmux openssh curl jq sqlite'
  '$ curl -fsSL <release install.sh> | sh'
  '$ gormes setup'
  '$ tmux new -s gormes'
  'READY same agent from Android'
)
frame07_right=(
  'Phone setup path'
  'No-root Android'
  'tmux persistence'
  'SSH for heavy work'
)

frame08_left=(
  '$ gormes setup gateway'
  'channel: Telegram / Discord / Slack'
  '$ gormes gateway'
  'gateway: terminal session stays in control'
  'memory: same sessions and provider profile'
)
frame08_right=(
  'CLI first'
  'Chat when ready'
  'Same local config'
  'One operator loop'
)

render_frame 1 "install" "Install once. No Python ceremony." "release installer" "#38bdf8" 12 frame01_left frame01_right
render_frame 2 "setup" "Let Gormes guide first-run configuration" "gormes setup" "#22c55e" 26 frame02_left frame02_right
render_frame 3 "provider setup" "Connect the model you actually use" "gormes setup provider" "#818cf8" 41 frame03_left frame03_right
render_frame 4 "first chat" "Start the useful terminal loop" "provider-backed chat" "#f59e0b" 55 frame04_left frame04_right
render_frame 5 "project work" "Point it at a repo and ask for help" "repo-aware first task" "#06b6d4" 69 frame05_left frame05_right
render_frame 6 "web tools" "Pull setup facts from the web when needed" "web_extract + goscrapling" "#a78bfa" 81 frame06_left frame06_right
render_frame 7 "Termux" "Use the same setup flow on Android" "Termux setup" "#22c55e" 92 frame07_left frame07_right
render_frame 8 "gateway" "Keep the terminal flow. Add chat later." "gateway setup" "#facc15" 100 frame08_left frame08_right

mkdir -p "$(dirname "$OUT")"
convert -delay 115 -loop 0 "$TMP_DIR"/frame_*.png -layers OptimizeTransparency -colors 256 "$OUT"
echo "rendered $OUT"
