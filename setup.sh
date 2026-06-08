#!/bin/bash
# Chaos2 — first-time setup
# Run once from the chaos2 directory: bash setup.sh

set -e
CHAOS2_DIR="$(cd "$(dirname "$0")" && pwd)"

echo ""
echo "=== Chaos2 setup ==="
echo ""

# ── Python venv ───────────────────────────────────────────────────────────────
echo "[1/4] Python venv..."
python3 -m venv "$CHAOS2_DIR/.venv"
"$CHAOS2_DIR/.venv/bin/pip" install --quiet --upgrade pip
"$CHAOS2_DIR/.venv/bin/pip" install --quiet -r "$CHAOS2_DIR/requirements.txt"
echo "      venv ready: .venv/"

# ── Go dependencies ───────────────────────────────────────────────────────────
echo "[2/4] Go dependencies..."
cd "$CHAOS2_DIR"
go mod tidy
echo "      go.mod ok"

# ── C++ classifier ────────────────────────────────────────────────────────────
echo "[3/4] C++ classifier..."
if [ -f "$CHAOS2_DIR/instinct/model_weights.h" ]; then
    make -C "$CHAOS2_DIR/instinct/" --silent
    echo "      instinct_classifier built"
else
    echo "      SKIP — model_weights.h not found yet"
    echo "      To generate it, run from your chaos1 directory:"
    echo "        source $CHAOS2_DIR/.venv/bin/activate"
    echo "        python3 $CHAOS2_DIR/instinct/export_weights.py"
    echo "      Then run: make -C $CHAOS2_DIR/instinct/"
fi

# ── .env ─────────────────────────────────────────────────────────────────────
echo "[4/4] .env..."
if [ ! -f "$CHAOS2_DIR/.env" ]; then
    cat > "$CHAOS2_DIR/.env" <<EOF
REDIS_ADDR=localhost:6379
CHAOS2_PYTHON_BIN=$CHAOS2_DIR/.venv/bin/python3
CHAOS2_PYTHON_BRIDGE=$CHAOS2_DIR/python/ollama_bridge.py
CHAOS2_INSTINCT_BIN=$CHAOS2_DIR/instinct/instinct_classifier
CHAOS2_R_BRAIN=$CHAOS2_DIR/reasoning/chaos_brain.R
EOF
    echo "      .env created"
else
    echo "      .env already exists, skipped"
fi

echo ""
echo "=== Done ==="
echo ""
echo "Next steps:"
echo "  1. Start Redis:  redis-server"
echo "  2. Start Ollama: ollama serve  (+ ollama pull llama3.2)"
echo "  3. Run Chaos2:   source .env && go run router/main.go"
echo ""
