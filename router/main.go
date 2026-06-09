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
	ShortTerm   []string
	LongTerm    []string
	Intent      string
	RBrain      RBrainResult
	Relevant    []string
	PlayerModel string
	Intensity   float64
	Goals       []string
	Rhythm      string
	Silent      bool
	Response    string
}

// ─── R brain ─────────────────────────────────────────────────────────────────

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
		case "plan":            r.Plan = v
		case "player_type":     r.PlayerType = v
		case "threat_level":    r.ThreatLevel = v
		case "emotional_drift": r.EmotionalDrift = v
		case "manipulation":    r.Manipulation = v
		case "confidence":      r.Confidence = v
		case "dominant_intent": r.DominantIntent = v
		case "trust_level":     r.TrustLevel = v
		case "volatility":      r.Volatility = v
		case "intimacy_signals":r.IntimacySignals = v
		case "aggression_count":r.AggressionCount = v
		}
	}
	return r
}

func trackIntent(ctx context.Context, sessionID, intent string) {
	key := fmt.Sprintf("chaos2:session:%s:intents", sessionID)
	rdb.HIncrBy(ctx, key, intent, 1)
	rdb.Expire(ctx, key, 24*time.Hour)
}

func updateRBrain(rb *sessionRBrain, sessionID string) {
	ctx := context.Background()

	// history CSV — stub with minimal valid rows so R doesn't bail early
	hist, err := os.CreateTemp("", "chaos2_history_*.csv")
	if err != nil {
		return
	}
	defer os.Remove(hist.Name())
	fmt.Fprintln(hist, "state,distrust,passivity,indifference")
	fmt.Fprintln(hist, "Neutral,50,60,60")
	hist.Close()

	// profile CSV from Redis intent counts
	prof, err := os.CreateTemp("", "chaos2_profile_*.csv")
	if err != nil {
		return
	}
	defer os.Remove(prof.Name())
	fmt.Fprintln(prof, "intent,count")
	intents, _ := rdb.HGetAll(ctx, fmt.Sprintf("chaos2:session:%s:intents", sessionID)).Result()
	for intent, count := range intents {
		fmt.Fprintf(prof, "%s,%s\n", intent, count)
	}
	prof.Close()

	// trust level file
	trustFile := prof.Name() + "_trust.txt"
	trustVal, _ := rdb.Get(ctx, fmt.Sprintf("chaos2:session:%s:trust", sessionID)).Result()
	if trustVal == "" {
		trustVal = "0"
	}
	os.WriteFile(trustFile, []byte(trustVal), 0644)
	defer os.Remove(trustFile)

	rScript := os.Getenv("CHAOS2_R_BRAIN")
	if rScript == "" {
		rScript = "reasoning/chaos_brain.R"
	}
	out, err := exec.Command("Rscript", "--vanilla", rScript, hist.Name(), prof.Name()).Output()
	if err != nil {
		log.Printf("[R] error: %v", err)
		return
	}
	result := parseROutput(string(out))
	rb.mu.Lock()
	rb.result = result
	rb.mu.Unlock()
	log.Printf("[R] session=%s plan=%s player=%s", sessionID, result.Plan, result.PlayerType)
}

// ─── C++ instinct ─────────────────────────────────────────────────────────────

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
	fmt.Fprintf(ip.stdin, "%s\n", text)
	if err := ip.stdin.Flush(); err != nil {
		return "curiosity"
	}
	if ip.stdout.Scan() {
		return strings.TrimSpace(ip.stdout.Text())
	}
	return "curiosity"
}

// ─── Redis ───────────────────────────────────────────────────────────────────

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
		return nil
	}
	return vals
}

func saveToRedis(ctx context.Context, sessionID, role, text string) {
	key := redisKey(sessionID)
	pipe := rdb.Pipeline()
	pipe.LPush(ctx, key, role+": "+text)
	pipe.LTrim(ctx, key, 0, 19)
	pipe.Expire(ctx, key, 2*time.Hour)
	pipe.Exec(ctx)
}

func lastTimestamp(ctx context.Context, sessionID string) int64 {
	key := fmt.Sprintf("chaos2:session:%s:last_ts", sessionID)
	val, err := rdb.Get(ctx, key).Result()
	if err != nil {
		return time.Now().UnixMilli() - 60000
	}
	ts, _ := strconv.ParseInt(val, 10, 64)
	return ts
}

func saveTimestamp(ctx context.Context, sessionID string, ts int64) {
	key := fmt.Sprintf("chaos2:session:%s:last_ts", sessionID)
	rdb.Set(ctx, key, ts, 2*time.Hour)
}

// ─── pipeline stages ─────────────────────────────────────────────────────────

// Lua — selective attention
func runAttention(ctx *CognitiveContext) {
	luaBin := os.Getenv("CHAOS2_LUA_BIN")
	if luaBin == "" {
		luaBin = "lua5.4"
	}
	cmd := exec.Command(luaBin, "attention/filter.lua")
	cmd.Stdin = strings.NewReader(ctx.Session.Text + "\n")
	out, err := cmd.Output()
	if err != nil {
		log.Printf("[attention] lua error: %v", err)
		ctx.Relevant = strings.Fields(ctx.Session.Text)
		return
	}
	ctx.Relevant = strings.Fields(strings.TrimSpace(string(out)))
}

// Prolog — mirror
func runMirror(ctx *CognitiveContext) {
	goal := fmt.Sprintf(
		"consult('mirror/player.pl'), mirror:print_model('%s'), halt.",
		ctx.Session.SessionID,
	)
	out, err := exec.Command("swipl", "-g", goal, "-t", "halt").Output()
	if err != nil {
		log.Printf("[mirror] prolog error: %v", err)
		ctx.PlayerModel = "type=unknown"
		return
	}
	ctx.PlayerModel = strings.TrimSpace(string(out))
}

// Haskell — regulation
func runRegulation(ctx *CognitiveContext) {
	bin := os.Getenv("CHAOS2_REGULATION_BIN")
	if bin == "" {
		bin = "regulation/regulation"
	}
	input := fmt.Sprintf("%s %s\n", ctx.RBrain.Plan, ctx.RBrain.ThreatLevel)
	cmd := exec.Command(bin)
	cmd.Stdin = strings.NewReader(input)
	out, err := cmd.Output()
	if err != nil {
		log.Printf("[regulation] haskell error: %v", err)
		threat, _ := strconv.ParseFloat(ctx.RBrain.ThreatLevel, 64)
		ctx.Intensity = 0.3 + (threat/10.0)*0.4
		return
	}
	intensity, err := strconv.ParseFloat(strings.TrimSpace(string(out)), 64)
	if err != nil {
		ctx.Intensity = 0.3
		return
	}
	ctx.Intensity = intensity
}

// Forth — purpose
func runPurpose(ctx *CognitiveContext) {
	plan := ctx.RBrain.Plan
	if plan == "" {
		plan = "observe"
	}
	evalExpr := fmt.Sprintf(`s" %s" evaluate-goals bye`, plan)
	out, err := exec.Command("gforth", "purpose/goals.fs", "-e", evalExpr).Output()
	if err != nil {
		log.Printf("[purpose] forth error: %v", err)
		ctx.Goals = []string{"survive", "understand"}
		return
	}
	ctx.Goals = strings.Split(strings.TrimSpace(string(out)), "\n")
}

// Erlang — time/rhythm
func runTime(ctx *CognitiveContext, redisCtx context.Context) {
	lastTs := lastTimestamp(redisCtx, ctx.Session.SessionID)
	lastSec := lastTs / 1000
	out, err := exec.Command("escript", "time/rhythm.erl",
		ctx.Session.SessionID,
		fmt.Sprintf("%d", lastSec),
	).Output()
	if err != nil {
		log.Printf("[time] erlang error: %v", err)
		ctx.Rhythm = "normal"
		return
	}
	ctx.Rhythm = strings.TrimSpace(string(out))
}

// Assembly — silence
func runSilence(ctx *CognitiveContext) {
	bin := os.Getenv("CHAOS2_SILENCE_BIN")
	if bin == "" {
		bin = "silence/silence"
	}
	input := fmt.Sprintf("%s|%s|%.2f\n", ctx.Intent, ctx.RBrain.Plan, ctx.Intensity)
	cmd := exec.Command(bin)
	cmd.Stdin = strings.NewReader(input)
	out, err := cmd.Output()
	if err != nil {
		log.Printf("[silence] asm error: %v", err)
		ctx.Silent = false
		return
	}
	ctx.Silent = strings.TrimSpace(string(out)) == "1"
}

// Python → Ollama
func runPython(ctx *CognitiveContext) {
	if ctx.Silent {
		ctx.Response = ""
		return
	}
	payload, _ := json.Marshal(map[string]interface{}{
		"text":     ctx.Session.Text,
		"intent":   ctx.Intent,
		"strategy": ctx.RBrain.Plan,
		"intensity": ctx.Intensity,
		"rhythm":   ctx.Rhythm,
		"history":  ctx.ShortTerm,
		"state": map[string]interface{}{
			"passivity":   100 - int(ctx.Intensity*100),
			"distrust":    50 + int(ctx.Intensity*30),
			"indifference": func() int {
				if ctx.RBrain.EmotionalDrift == "calming" { return 30 }
				if ctx.RBrain.EmotionalDrift == "escalating" { return 20 }
				return 60
			}(),
		},
		"player": map[string]string{
			"type":         ctx.RBrain.PlayerType,
			"drift":        ctx.RBrain.EmotionalDrift,
			"threat":       ctx.RBrain.ThreatLevel,
			"manipulation": ctx.RBrain.Manipulation,
			"trust":        ctx.RBrain.TrustLevel,
			"dominant":     ctx.RBrain.DominantIntent,
			"model":        ctx.PlayerModel,
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

// ─── background workers ───────────────────────────────────────────────────────

func startBackgroundWorkers() {
	// Julia — memory reorganization every 5 minutes
	go func() {
		ticker := time.NewTicker(5 * time.Minute)
		for range ticker.C {
			ctx := context.Background()
			keys, err := rdb.Keys(ctx, "chaos2:session:*:context").Result()
			if err != nil || len(keys) == 0 {
				continue
			}
			for _, key := range keys {
				memories, _ := rdb.LRange(ctx, key, 0, -1).Result()
				if len(memories) == 0 {
					continue
				}
				input := strings.Join(memories, "\n") + "\n"
				cmd := exec.Command("julia", "dream/reorganize.jl")
				cmd.Stdin = strings.NewReader(input)
				if out, err := cmd.Output(); err == nil {
					log.Printf("[julia] reorganized session %s: %d bytes", key, len(out))
				}
			}
		}
	}()

	// Rust — memory decay every hour
	go func() {
		ticker := time.NewTicker(1 * time.Hour)
		for range ticker.C {
			log.Printf("[rust] memory decay tick")
			// forgetting binary processes memories from stdin
			// wired to Redis in future iteration
		}
	}()

	// Lisp — rule evolution every 10 minutes
	go func() {
		ticker := time.NewTicker(10 * time.Minute)
		for range ticker.C {
			cmd := exec.Command("clisp", "development/evolve.lisp")
			cmd.Stdin = strings.NewReader("")
			if out, err := cmd.Output(); err == nil {
				log.Printf("[lisp] rules: %s", strings.TrimSpace(string(out)))
			}
		}
	}()
}

// ─── HTTP handlers ────────────────────────────────────────────────────────────

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

	// 1. memory
	cogCtx.ShortTerm = loadShortTerm(ctx, msg.SessionID)

	// 2. instinct (C++) + reasoning (R) — parallel
	rb := getOrCreateRBrain(msg.SessionID)
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		context := msg.Text
		if len(cogCtx.ShortTerm) > 0 {
			context = cogCtx.ShortTerm[0] + " || " + msg.Text
		}
		cogCtx.Intent = instinct.classify(context)
	}()
	go func() {
		defer wg.Done()
		rb.mu.RLock()
		cogCtx.RBrain = rb.result
		count := rb.count
		rb.mu.RUnlock()
		rb.mu.Lock()
		rb.count++
		rb.mu.Unlock()
		if (count+1)%5 == 0 {
			go updateRBrain(rb, msg.SessionID)
		}
	}()
	wg.Wait()

	// track intent + trust in Redis for R brain
	trackIntent(ctx, msg.SessionID, cogCtx.Intent)
	if cogCtx.Intent == "trust" {
		rdb.IncrByFloat(ctx, fmt.Sprintf("chaos2:session:%s:trust", msg.SessionID), 0.08)
	} else if cogCtx.Intent == "aggression" {
		rdb.IncrByFloat(ctx, fmt.Sprintf("chaos2:session:%s:trust", msg.SessionID), -0.05)
	}

	// 3. attention — Lua
	runAttention(cogCtx)

	// 4. mirror (Prolog) + regulation (Haskell) + time (Erlang) — parallel
	wg.Add(3)
	go func() { defer wg.Done(); runMirror(cogCtx) }()
	go func() { defer wg.Done(); runRegulation(cogCtx) }()
	go func() { defer wg.Done(); runTime(cogCtx, ctx) }()
	wg.Wait()

	// 5. purpose — Forth
	runPurpose(cogCtx)

	// 6. silence — Assembly
	runSilence(cogCtx)

	// 7. Python → Ollama
	runPython(cogCtx)

	// persist
	saveToRedis(ctx, msg.SessionID, "player", msg.Text)
	saveTimestamp(ctx, msg.SessionID, msg.Timestamp)
	if cogCtx.Response != "" {
		saveToRedis(ctx, msg.SessionID, "chaos", cogCtx.Response)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"response":     cogCtx.Response,
		"silent":       cogCtx.Silent,
		"intent":       cogCtx.Intent,
		"relevant":     cogCtx.Relevant,
		"plan":         cogCtx.RBrain.Plan,
		"player_type":  cogCtx.RBrain.PlayerType,
		"player_model": cogCtx.PlayerModel,
		"goals":        cogCtx.Goals,
		"intensity":    cogCtx.Intensity,
		"rhythm":       cogCtx.Rhythm,
		"drift":        cogCtx.RBrain.EmotionalDrift,
		"threat":       cogCtx.RBrain.ThreatLevel,
		"manipulation": cogCtx.RBrain.Manipulation,
	})
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	redisOK := rdb.Ping(r.Context()).Err() == nil
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":   "ok",
		"system":   "chaos2",
		"redis":    redisOK,
		"instinct": instinct.cmd != nil,
	})
}

// ─── main ─────────────────────────────────────────────────────────────────────

func main() {
	initRedis()
	if err := instinct.start(); err != nil {
		log.Printf("[instinct] unavailable: %v — using fallback", err)
	}
	startBackgroundWorkers()

	http.HandleFunc("/message", handleMessage)
	http.HandleFunc("/health", handleHealth)

	addr := ":8080"
	log.Printf("[chaos2] all modules connected — starting on %s", addr)
	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatalf("[chaos2] fatal: %v", err)
	}
}
