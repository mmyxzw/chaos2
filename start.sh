#!/usr/bin/env bash
# Chaos2 — Colab startup script (idempotent)
# Usage: bash start.sh
# Expects .env in repo root with REDIS_URL, OPENAI_API_KEY (optional), etc.

set -euo pipefail

REPO="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$REPO"

log() { echo "[start] $*"; }

# ─── 1. System dependencies ──────────────────────────────────────────────────

log "Installing system dependencies..."
apt-get update -qq

PKGS=(
  curl wget git build-essential cmake
  lua5.4 liblua5.4-dev
  swi-prolog
  r-base r-base-dev
  cargo rustc
  ghc cabal-install
  gforth
  erlang
  clisp
  python3 python3-pip python3-venv
  liblapack-dev libopenblas-dev
)

for pkg in "${PKGS[@]}"; do
  dpkg -s "$pkg" &>/dev/null || apt-get install -y -qq "$pkg"
done

# Julia — install if missing
if ! command -v julia &>/dev/null; then
  log "Installing Julia..."
  JULIA_VER="1.10.3"
  wget -q "https://julialang-s3.julialang.org/bin/linux/x64/1.10/julia-${JULIA_VER}-linux-x86_64.tar.gz" -O /tmp/julia.tar.gz
  tar -xf /tmp/julia.tar.gz -C /usr/local --strip-components=1
  rm /tmp/julia.tar.gz
fi

# Go — install if missing or wrong version
if ! command -v go &>/dev/null || [[ "$(go version 2>/dev/null | awk '{print $3}')" < "go1.21" ]]; then
  log "Installing Go..."
  GO_VER="1.22.4"
  wget -q "https://go.dev/dl/go${GO_VER}.linux-amd64.tar.gz" -O /tmp/go.tar.gz
  rm -rf /usr/local/go
  tar -xf /tmp/go.tar.gz -C /usr/local
  rm /tmp/go.tar.gz
fi
export PATH="/usr/local/go/bin:$PATH"

# ─── 2. Python venv ───────────────────────────────────────────────────────────

log "Setting up Python venv..."
if [ ! -d "$REPO/.venv" ]; then
  python3 -m venv "$REPO/.venv"
fi
source "$REPO/.venv/bin/activate"
pip install -q --upgrade pip
pip install -q scikit-learn numpy redis

# ─── 3. Build C++ classifier ─────────────────────────────────────────────────

log "Building C++ instinct classifier..."
cd "$REPO/instinct"
if [ ! -f build/instinct ] || [ src/main.cpp -nt build/instinct ]; then
  mkdir -p build
  g++ -O2 -std=c++17 -o build/instinct src/main.cpp
fi
cd "$REPO"

# ─── 4. Train classifier if no model file ────────────────────────────────────

log "Checking classifier model..."
if [ ! -f "$REPO/instinct/model.bin" ]; then
  log "Training classifier (first run)..."
  cd "$REPO/instinct"
  python3 train.py
  cd "$REPO"
fi

# ─── 5. Build Rust forgetting binary ─────────────────────────────────────────

log "Building Rust forgetting module..."
cd "$REPO/forgetting"
cargo build --release -q
cd "$REPO"

# ─── 6. Build Go router ───────────────────────────────────────────────────────

log "Building Go router..."
cd "$REPO/router"
go mod tidy -q
go build -o ../chaos2-router .
cd "$REPO"

# ─── 7. Ollama ────────────────────────────────────────────────────────────────

if ! command -v ollama &>/dev/null; then
  log "Installing Ollama..."
  curl -fsSL https://ollama.ai/install.sh | sh
fi

# Start Ollama daemon if not running
if ! pgrep -x ollama &>/dev/null; then
  log "Starting Ollama daemon..."
  nohup ollama serve > /tmp/ollama.log 2>&1 &
  sleep 3
fi

# Pull model if not present
MODEL="llama3.2"
if ! ollama list 2>/dev/null | grep -q "$MODEL"; then
  log "Pulling $MODEL (this takes a while)..."
  ollama pull "$MODEL"
fi

# ─── 8. Load .env ─────────────────────────────────────────────────────────────

if [ -f "$REPO/.env" ]; then
  log "Loading .env..."
  set -a
  # shellcheck disable=SC1090
  source "$REPO/.env"
  set +a
fi

# Validate Redis
if [ -z "${REDIS_URL:-}" ] && [ -z "${REDIS_ADDR:-}" ]; then
  log "WARNING: neither REDIS_URL nor REDIS_ADDR set — using localhost:6379"
fi

# ─── 9. Start Chaos2 router ───────────────────────────────────────────────────

# Kill stale router if running
if pgrep -f chaos2-router &>/dev/null; then
  log "Stopping old router..."
  pkill -f chaos2-router || true
  sleep 1
fi

PORT="${PORT:-8080}"
log "Starting Chaos2 router on :${PORT}..."
nohup "$REPO/chaos2-router" > /tmp/chaos2.log 2>&1 &
ROUTER_PID=$!

# Health check
sleep 2
if kill -0 "$ROUTER_PID" 2>/dev/null; then
  log "Router running (PID $ROUTER_PID)"
  log "Tail logs: tail -f /tmp/chaos2.log"
  log "Endpoint: http://localhost:${PORT}/chat"
else
  log "ERROR: Router failed to start. Check /tmp/chaos2.log"
  cat /tmp/chaos2.log
  exit 1
fi
