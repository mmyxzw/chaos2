package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

// ─── types ───────────────────────────────────────────────────────────────────

type PlayerMessage struct {
	SessionID string `json:"session_id"`
	Text      string `json:"text"`
	Timestamp int64  `json:"timestamp"`
}

type RBrainResult struct {
	Plan            string
	PlayerType      string
	ThreatLevel     string
	EmotionalDrift  string
	Manipulation    string
	Confidence      string
	DominantIntent  string
	TrustLevel      string
	Volatility      string
	IntimacySignals string
	AggressionCount string
}

type CognitiveContext struct {
	Session     PlayerMessage
	ShortTerm   []string     // Redis: last N messages this session
	LongTerm    []string     // SQLite/PG: historical context
	Intent      string       // C++ instinct
	RBrain      RBrainResult // R reasoning (latest cached)
	Relevant    []string     // Lua attention
	PlayerModel string       // Prolog mirror
	Intensity   float64      // Haskell regulation
	Goals       []string     // Forth purpose
	Silent      bool         // Assembly silence
	Response    string       // Python → Ollama
}

// ─── R brain: async, cached per session ─────────────────────────────────────

type sessionRBrain struct {
	mu     sync.RWMutex
	result RBrainResult
	count  int
}

var (
	rBrains   = make(map[string]*sessionRBrain)
	rBrainsMu sync.Mutex
)

func getOrCreateRBrain(sessionID string) *sessionRBrain {
	rBrainsMu.Lock()
	defer rBrainsMu.Unlock()
	if rb, ok := rBrains[sessionID]; ok {
		return rb
	}
	rb := &sessionRBrain{result: RBrainResult{
		Plan: "observe", PlayerType: "unknown", ThreatLevel: "0",
		EmotionalDrift: "stable", Manipulation: "false",
	}}
	rBrains[sessionID] = rb
	return rb
}

func parseROutput(raw string) RBrainResult {
	r := RBrainResult{Plan: "observe", PlayerType: "unknown"}
	for _, part := range strings.Split(strings.TrimSpace(raw), "|") {
		kv := strings.SplitN(part, "=", 2)
		if len(kv) != 2 {
			continue
		}
		k, v := strings.TrimSpace(kv[0]), strings.TrimSpace(kv[1])
		switch k {
		case "plan":
			r.Plan = v
		case "player_type":
			r.PlayerType = v
		case "threat_level":
			r.ThreatLevel = v
		case "emotional_drift":
			r.EmotionalDrift = v
		case "manipulation":
			r.Manipulation = v
		case "confidence":
			r.Confidence = v
		case "dominant_intent":
			r.DominantIntent = v
		case "trust_level":
			r.TrustLevel = v
		case "volatility":
			r.Volatility = v
		case "intimacy_signals":
			r.IntimacySignals = v
		case "aggression_count":
			r.AggressionCount = v
		}
	}
	return r
}

// updateRBrain writes CSVs and calls chaos_brain.R — runs async
func updateRBrain(rb *sessionRBrain, sessionID string, history []string) {
	histFile := fmt.Sprintf("/tmp/chaos2_history_%s.csv", sessionID)
	f, err := os.CreateTemp("", "chaos2_history_*.csv")
	if err != nil {
		return
	}
	histFile = f.Name()
	defer os.Remove(histFile)

	fmt.Fprintln(f, "state,distrust,passivity,indifference")
	for _, line := range history {
		fmt.Fprintln(f, line)
	}
	f.Close()

	rScript := os.Getenv("CHAOS2_R_BRAIN")
	if rScript == "" {
		rScript = "reasoning/chaos_brain.R"
	}
	out, err := exec.Command("Rscript", "--vanilla", rScript, histFile).Output()
	if err != nil {
		log.Printf("[R] error: %v", err)
		return
	}

	result := parseROutput(string(out))
	rb.mu.Lock()
	rb.result = result
	rb.mu.Unlock()
}

// ─── C++ instinct: persistent subprocess per process (singleton) ─────────────

type instinctProcess struct {
	mu     sync.Mutex
	cmd    *exec.Cmd
	stdin  *bufio.Writer
	stdout *bufio.Scanner
}

var instinct = &instinctProcess{}

func (ip *instinctProcess) start() error {
	ip.mu.Lock()
	defer ip.mu.Unlock()
	if ip.cmd != nil {
		return nil
	}

	binary := os.Getenv("CHAOS2_INSTINCT_BIN")
	if binary == "" {
		binary = "instinct/instinct_classifier"
	}
	cmd := exec.Command(binary)

	inPipe, err := cmd.StdinPipe()
	if err != nil {
		return err
	}
	outPipe, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	if err := cmd.Start(); err != nil {
		return err
	}

	ip.cmd    = cmd
	ip.stdin  = bufio.NewWriter(inPipe)
	ip.stdout = bufio.NewScanner(outPipe)
	log.Printf("[instinct] C++ classifier started (pid %d)", cmd.Process.Pid)
	return nil
}

func (ip *instinctProcess) classify(text string) string {
	ip.mu.Lock()
	defer ip.mu.Unlock()

	if ip.cmd == nil {
		return "curiosity"
	}
	// send message, read intent
	fmt.Fprintf(ip.stdin, "%s\n", text)
	if err := ip.stdin.Flush(); err != nil {
		log.Printf("[instinct] write error: %v", err)
		return "curiosity"
	}
	if ip.stdout.Scan() {
		return strings.TrimSpace(ip.stdout.Text())
	}
	return "curiosity"
}

// ─── Redis: short-term memory ────────────────────────────────────────────────

var rdb *redis.Client

func initRedis() {
	addr := os.Getenv("REDIS_ADDR")
	if addr == "" {
		addr = "localhost:6379"
	}
	rdb = redis.NewClient(&redis.Options{Addr: addr})
}

func redisKey(sessionID string) string {
	return fmt.Sprintf("chaos2:session:%s:context", sessionID)
}

func loadShortTerm(ctx context.Context, sessionID string) []string {
	vals, err := rdb.LRange(ctx, redisKey(sessionID), 0, 9).Result()
	if err != nil {
		log.Printf("[redis] load error: %v", err)
		return nil
	}
	return vals
}

func saveToRedis(ctx context.Context, sessionID, role, text string) {
	key := redisKey(sessionID)
	entry := role + ": " + text
	pipe := rdb.Pipeline()
	pipe.LPush(ctx, key, entry)
	pipe.LTrim(ctx, key, 0, 19)             // keep last 20 entries
	pipe.Expire(ctx, key, 2*time.Hour)       // session TTL
	if _, err := pipe.Exec(ctx); err != nil {
		log.Printf("[redis] save error: %v", err)
	}
}

// ─── pipeline stages (stubs for phases 2–4) ─────────────────────────────────

func runAttention(ctx *CognitiveContext) {
	// Phase 2 — Lua selective filter
	// For now: pass all short-term context through
	ctx.Relevant = ctx.ShortTerm
}

func runMirror(ctx *CognitiveContext) {
	// Phase 3 — Prolog player model
	ctx.PlayerModel = ctx.RBrain.PlayerType
}

func runRegulation(ctx *CognitiveContext) {
	// Phase 3 — Haskell response calibration
	threat, _ := strconv.ParseFloat(ctx.RBrain.ThreatLevel, 64)
	ctx.Intensity = 0.3 + (threat/10.0)*0.7
}

func runPurpose(ctx *CognitiveContext) {
	// Phase 3 — Forth: Chaos's own goals
	ctx.Goals = []string{"survive", "understand"}
	switch ctx.RBrain.Plan {
	case "seduce":
		ctx.Goals = []string{"attract", "hold"}
	case "resist":
		ctx.Goals = []string{"endure", "refuse"}
	case "reveal":
		ctx.Goals = []string{"trust", "open"}
	case "collapse":
		ctx.Goals = []string{"break", "release"}
	case "mirror":
		ctx.Goals = []string{"reflect", "expose"}
	}
}

func runSilence(ctx *CognitiveContext) {
	// Phase 4 — Assembly: when NOT to respond
	// Silent if Absent state + low threat + short message
	ctx.Silent = false
}

// ─── Python → Ollama ────────────────────────────────────────────────────────

func runPython(ctx *CognitiveContext) {
	if ctx.Silent {
		ctx.Response = ""
		return
	}

	payload, _ := json.Marshal(map[string]interface{}{
		"text":      ctx.Session.Text,
		"intent":    ctx.Intent,
		"strategy":  ctx.RBrain.Plan,
		"intensity": ctx.Intensity,
		"goals":     ctx.Goals,
		"rhythm":    "normal",
		"player": map[string]string{
			"type":         ctx.RBrain.PlayerType,
			"drift":        ctx.RBrain.EmotionalDrift,
			"threat":       ctx.RBrain.ThreatLevel,
			"manipulation": ctx.RBrain.Manipulation,
			"trust":        ctx.RBrain.TrustLevel,
			"dominant":     ctx.RBrain.DominantIntent,
		},
	})

	pythonBin := os.Getenv("CHAOS2_PYTHON_BIN")
	if pythonBin == "" {
		pythonBin = "python3"
	}
	script := os.Getenv("CHAOS2_PYTHON_BRIDGE")
	if script == "" {
		script = "python/ollama_bridge.py"
	}
	cmd := exec.Command(pythonBin, script)
	cmd.Stdin = strings.NewReader(string(payload))
	out, err := cmd.Output()
	if err != nil {
		log.Printf("[python] error: %v", err)
		ctx.Response = "..."
		return
	}
	ctx.Response = strings.TrimSpace(string(out))
}

// ─── background workers (phases 2–4) ────────────────────────────────────────

func startBackgroundWorkers() {
	// Julia — memory reorganization during inactivity (Phase 4)
	go func() {
		ticker := time.NewTicker(5 * time.Minute)
		for range ticker.C {
			log.Printf("[julia] memory reorganization tick (stub)")
		}
	}()

	// Erlang — rhythm and urgency monitor (Phase 4)
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		for range ticker.C {
			log.Printf("[erlang] rhythm check (stub)")
		}
	}()

	// Rust — memory decay between sessions (Phase 2)
	go func() {
		ticker := time.NewTicker(1 * time.Hour)
		for range ticker.C {
			log.Printf("[rust] memory decay tick (stub)")
		}
	}()

	// Lisp — rule evolution (Phase 4)
	go func() {
		ticker := time.NewTicker(10 * time.Minute)
		for range ticker.C {
			log.Printf("[lisp] rule evolution tick (stub)")
		}
	}()
}

// ─── HTTP handler ────────────────────────────────────────────────────────────

func handleMessage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}

	var msg PlayerMessage
	if err := json.NewDecoder(r.Body).Decode(&msg); err != nil {
		http.Error(w, "bad JSON", http.StatusBadRequest)
		return
	}
	if msg.Timestamp == 0 {
		msg.Timestamp = time.Now().UnixMilli()
	}

	ctx := r.Context()
	cogCtx := &CognitiveContext{Session: msg}

	// ── 1. memory: Redis (short-term) + SQLite/PG (long-term) ────────────────
	cogCtx.ShortTerm = loadShortTerm(ctx, msg.SessionID)
	// SQLite/PG stub — Phase 2
	cogCtx.LongTerm = nil

	// ── 2. instinct (C++) + reasoning (R) — parallel ─────────────────────────
	rb := getOrCreateRBrain(msg.SessionID)

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		// C++: classify intent from "context || message"
		var context string
		if len(cogCtx.ShortTerm) > 0 {
			context = cogCtx.ShortTerm[0] + " || " + msg.Text
		} else {
			context = msg.Text
		}
		cogCtx.Intent = instinct.classify(context)
	}()

	go func() {
		defer wg.Done()
		// R: use cached result; trigger update every 5 messages
		rb.mu.RLock()
		cogCtx.RBrain = rb.result
		count := rb.count
		rb.mu.RUnlock()

		rb.mu.Lock()
		rb.count++
		rb.mu.Unlock()

		if (count+1)%5 == 0 {
			// fire async — does not block this request
			go updateRBrain(rb, msg.SessionID, cogCtx.ShortTerm)
		}
	}()

	wg.Wait()

	// ── 3. attention — Lua filters what matters ───────────────────────────────
	runAttention(cogCtx)

	// ── 4. mirror (Prolog) + regulation (Haskell) — parallel ─────────────────
	wg.Add(2)
	go func() { defer wg.Done(); runMirror(cogCtx) }()
	go func() { defer wg.Done(); runRegulation(cogCtx) }()
	wg.Wait()

	// ── 5. purpose — Forth orients the response ───────────────────────────────
	runPurpose(cogCtx)

	// ── 6. silence — Assembly decides whether to speak ────────────────────────
	runSilence(cogCtx)

	// ── 7. Python → Ollama → response ────────────────────────────────────────
	runPython(cogCtx)

	// ── persist to Redis ──────────────────────────────────────────────────────
	saveToRedis(ctx, msg.SessionID, "player", msg.Text)
	if cogCtx.Response != "" {
		saveToRedis(ctx, msg.SessionID, "chaos", cogCtx.Response)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"response":     cogCtx.Response,
		"silent":       cogCtx.Silent,
		"intent":       cogCtx.Intent,
		"plan":         cogCtx.RBrain.Plan,
		"player_type":  cogCtx.RBrain.PlayerType,
		"intensity":    cogCtx.Intensity,
		"drift":        cogCtx.RBrain.EmotionalDrift,
		"threat":       cogCtx.RBrain.ThreatLevel,
		"manipulation": cogCtx.RBrain.Manipulation,
	})
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	redisOK := rdb.Ping(r.Context()).Err() == nil
	instinctOK := instinct.cmd != nil

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":    "ok",
		"system":    "chaos2",
		"redis":     redisOK,
		"instinct":  instinctOK,
	})
}

// ─── main ─────────────────────────────────────────────────────────────────────

func main() {
	initRedis()

	if err := instinct.start(); err != nil {
		log.Printf("[instinct] C++ classifier unavailable: %v — falling back to stub", err)
	}

	startBackgroundWorkers()

	http.HandleFunc("/message", handleMessage)
	http.HandleFunc("/health", handleHealth)

	addr := ":8080"
	log.Printf("[chaos2] router starting on %s", addr)
	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatalf("[chaos2] fatal: %v", err)
	}
}
