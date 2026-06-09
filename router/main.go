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
	ChaosState  string
}

// ─── state machine ───────────────────────────────────────────────────────────

type stateValues struct {
	Passivity    int
	Distrust     int
	Indifference int
}

var stateData = map[string]stateValues{
	"Neutral":               {60, 50, 60},
	"Curious":               {40, 60, 40},
	"Confident":             {40, 25, 50},
	"Hostile":               {20, 70, 45},
	"Obsessive_Love":        {70, 40, 10},
	"Obsessive_Hate":        {10, 70, 45},
	"Obsessive_Fascination": {20, 70, 30},
	"Redemptive":            {70, 50, 50},
	"Absent":                {95,  5, 95},
}

var stateCooldowns = map[string]int{
	"Neutral": 3, "Curious": 4, "Confident": 5, "Hostile": 6,
	"Obsessive_Love": 7, "Obsessive_Hate": 7, "Obsessive_Fascination": 5,
	"Redemptive": 6, "Absent": 4,
}

// intent → current_state → next_state
var intentTransitions = map[string]map[string]string{
	"aggression": {
		"Neutral": "Hostile", "Confident": "Hostile", "Curious": "Obsessive_Hate",
		"Obsessive_Love": "Obsessive_Hate", "Redemptive": "Hostile",
		"Absent": "Hostile", "Obsessive_Fascination": "Hostile",
	},
	"curiosity": {
		"Neutral": "Curious", "Absent": "Curious", "Hostile": "Curious",
		"Obsessive_Hate": "Curious", "Obsessive_Love": "Curious",
		"Obsessive_Fascination": "Curious", "Redemptive": "Curious",
	},
	"trust": {
		"Neutral": "Confident", "Curious": "Confident", "Hostile": "Redemptive",
		"Obsessive_Hate": "Redemptive", "Obsessive_Love": "Confident",
		"Obsessive_Fascination": "Confident", "Redemptive": "Confident",
		"Absent": "Neutral",
	},
	"withdrawal": {
		"Neutral": "Absent", "Curious": "Absent", "Confident": "Absent",
		"Hostile": "Absent", "Obsessive_Love": "Absent", "Obsessive_Hate": "Absent",
		"Obsessive_Fascination": "Absent", "Redemptive": "Absent",
	},
	"philosophical": {
		"Neutral": "Curious", "Curious": "Absent", "Confident": "Curious",
		"Hostile": "Curious", "Obsessive_Love": "Obsessive_Fascination",
		"Obsessive_Hate": "Obsessive_Fascination", "Obsessive_Fascination": "Absent",
		"Redemptive": "Absent",
	},
	"intimacy": {
		"Neutral": "Obsessive_Love", "Confident": "Obsessive_Love",
		"Redemptive": "Obsessive_Love", "Curious": "Obsessive_Love",
		"Absent": "Obsessive_Love", "Obsessive_Fascination": "Obsessive_Love",
		"Hostile": "Obsessive_Hate", "Obsessive_Hate": "Obsessive_Love",
	},
	"provocation": {
		"Neutral": "Obsessive_Fascination", "Curious": "Obsessive_Fascination",
		"Absent": "Obsessive_Fascination", "Confident": "Hostile",
		"Redemptive": "Hostile", "Obsessive_Love": "Obsessive_Hate",
		"Obsessive_Fascination": "Hostile", "Hostile": "Obsessive_Hate",
	},
}

// plan blocks certain transitions
var planAllowedIntents = map[string][]string{
	"mirror":   {},
	"resist":   {"aggression"},
	"collapse": {"trust"},
	"reveal":   {"trust", "curiosity", "intimacy"},
}

func planAllows(plan, intent string) bool {
	allowed, restricted := planAllowedIntents[plan]
	if !restricted {
		return true
	}
	for _, a := range allowed {
		if a == intent {
			return true
		}
	}
	return false
}

func loadChaosState(ctx context.Context, sessionID string) (string, int) {
	stateKey := fmt.Sprintf("chaos2:session:%s:chaos_state", sessionID)
	countKey := fmt.Sprintf("chaos2:session:%s:state_count", sessionID)
	state, err := rdb.Get(ctx, stateKey).Result()
	if err != nil || state == "" {
		state = "Neutral"
	}
	countStr, _ := rdb.Get(ctx, countKey).Result()
	count, _ := strconv.Atoi(countStr)
	return state, count
}

func saveChaosState(ctx context.Context, sessionID, state string, count int) {
	stateKey := fmt.Sprintf("chaos2:session:%s:chaos_state", sessionID)
	countKey := fmt.Sprintf("chaos2:session:%s:state_count", sessionID)
	rdb.Set(ctx, stateKey, state, 24*time.Hour)
	rdb.Set(ctx, countKey, count, 24*time.Hour)
}

func transitionState(ctx context.Context, sessionID, intent, plan string) string {
	state, count := loadChaosState(ctx, sessionID)
	count++
	cooldown := stateCooldowns[state]
	if cooldown == 0 {
		cooldown = 4
	}
	newState := state
	if count >= cooldown && planAllows(plan, intent) {
		if transitions, ok := intentTransitions[intent]; ok {
			if next, ok := transitions[state]; ok {
				newState = next
				count = 0
			}
		}
	}
	saveChaosState(ctx, sessionID, newState, count)
	return newState
}

// ─── R brain daemon ──────────────────────────────────────────────────────────

type rDaemonProcess struct {
	mu     sync.Mutex
	cmd    *exec.Cmd
	stdin  *bufio.Writer
	stdout *bufio.Scanner
}

var rDaemon = &rDaemonProcess{}

func (rd *rDaemonProcess) start() error {
	rd.mu.Lock()
	defer rd.mu.Unlock()
	if rd.cmd != nil {
		return nil
	}
	script := os.Getenv("CHAOS2_R_BRAIN")
	if script == "" {
		script = "reasoning/chaos_brain.R"
	}
	cmd := exec.Command("Rscript", "--vanilla", script)
	inPipe, err := cmd.StdinPipe()
	if err != nil {
		return err
	}
	outPipe, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		return err
	}
	rd.cmd    = cmd
	rd.stdin  = bufio.NewWriter(inPipe)
	rd.stdout = bufio.NewScanner(outPipe)
	log.Printf("[R] daemon started (pid %d)", cmd.Process.Pid)
	return nil
}

var rFallback = RBrainResult{
	Plan: "observe", PlayerType: "unknown", ThreatLevel: "0",
	EmotionalDrift: "stable", Manipulation: "false", Confidence: "0.5",
}

func (rd *rDaemonProcess) query(sessionID, intent, chaosState, trustLevel string) RBrainResult {
	rd.mu.Lock()
	defer rd.mu.Unlock()
	if rd.cmd == nil {
		return rFallback
	}
	fmt.Fprintf(rd.stdin, "%s|%s|%s|%s\n", sessionID, intent, chaosState, trustLevel)
	if err := rd.stdin.Flush(); err != nil {
		log.Printf("[R] write error: %v — restarting", err)
		rd.cmd = nil
		return rFallback
	}
	if rd.stdout.Scan() {
		return parseROutput(rd.stdout.Text())
	}
	log.Printf("[R] daemon died — restarting")
	rd.cmd = nil
	return rFallback
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
		case "plan":             r.Plan = v
		case "player_type":      r.PlayerType = v
		case "threat_level":     r.ThreatLevel = v
		case "emotional_drift":  r.EmotionalDrift = v
		case "manipulation":     r.Manipulation = v
		case "confidence":       r.Confidence = v
		case "dominant_intent":  r.DominantIntent = v
		case "trust_level":      r.TrustLevel = v
		case "volatility":       r.Volatility = v
		case "intimacy_signals": r.IntimacySignals = v
		case "aggression_count": r.AggressionCount = v
		}
	}
	return r
}

func loadPlan(ctx context.Context, sessionID string) string {
	key := fmt.Sprintf("chaos2:session:%s:plan", sessionID)
	plan, err := rdb.Get(ctx, key).Result()
	if err != nil || plan == "" {
		return "observe"
	}
	return plan
}

func savePlan(ctx context.Context, sessionID, plan string) {
	key := fmt.Sprintf("chaos2:session:%s:plan", sessionID)
	rdb.Set(ctx, key, plan, 24*time.Hour)
}

func trackIntent(ctx context.Context, sessionID, intent string) {
	key := fmt.Sprintf("chaos2:session:%s:intents", sessionID)
	rdb.HIncrBy(ctx, key, intent, 1)
	rdb.Expire(ctx, key, 24*time.Hour)
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
	ts  := time.Now().Unix()
	entry := fmt.Sprintf("%d|1.0|%s: %s", ts, role, text)
	pipe := rdb.Pipeline()
	pipe.LPush(ctx, key, entry)
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
	sv := stateData[ctx.ChaosState]
	if sv.Passivity == 0 {
		sv = stateData["Neutral"]
	}
	payload, _ := json.Marshal(map[string]interface{}{
		"text":     ctx.Session.Text,
		"intent":   ctx.Intent,
		"strategy": ctx.RBrain.Plan,
		"intensity": ctx.Intensity,
		"rhythm":   ctx.Rhythm,
		"history":  ctx.ShortTerm,
		"chaos_state": ctx.ChaosState,
		"state": map[string]interface{}{
			"passivity":    sv.Passivity,
			"distrust":     sv.Distrust,
			"indifference": sv.Indifference,
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

// runForgetting applies exponential memory decay across all active sessions.
func runForgetting() {
	bin := os.Getenv("CHAOS2_FORGETTING_BIN")
	if bin == "" {
		bin = "forgetting/target/release/forgetting"
	}
	ctx := context.Background()
	keys, err := rdb.Keys(ctx, "chaos2:session:*:context").Result()
	if err != nil || len(keys) == 0 {
		return
	}
	for _, key := range keys {
		memories, err := rdb.LRange(ctx, key, 0, -1).Result()
		if err != nil || len(memories) == 0 {
			continue
		}
		input := strings.Join(memories, "\n") + "\n"
		cmd := exec.Command(bin)
		cmd.Stdin = strings.NewReader(input)
		out, err := cmd.Output()
		if err != nil {
			log.Printf("[forgetting] error on %s: %v", key, err)
			continue
		}
		survived := strings.Split(strings.TrimSpace(string(out)), "\n")
		if len(survived) == 0 || (len(survived) == 1 && survived[0] == "") {
			// all decayed — remove the key
			rdb.Del(ctx, key)
			continue
		}
		// atomically replace the list with the decayed version
		pipe := rdb.Pipeline()
		pipe.Del(ctx, key)
		for _, m := range survived {
			pipe.RPush(ctx, key, m)
		}
		pipe.Expire(ctx, key, 2*time.Hour)
		if _, err := pipe.Exec(ctx); err != nil {
			log.Printf("[forgetting] write error on %s: %v", key, err)
		}
	}
	log.Printf("[forgetting] decay applied to %d sessions", len(keys))
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
			runForgetting()
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

	// 2. instinct (C++) — classify intent
	{
		context := msg.Text
		if len(cogCtx.ShortTerm) > 0 {
			context = cogCtx.ShortTerm[0] + " || " + msg.Text
		}
		cogCtx.Intent = instinct.classify(context)
	}

	// track intent + update trust
	trackIntent(ctx, msg.SessionID, cogCtx.Intent)
	if cogCtx.Intent == "trust" {
		rdb.IncrByFloat(ctx, fmt.Sprintf("chaos2:session:%s:trust", msg.SessionID), 0.08)
	} else if cogCtx.Intent == "aggression" {
		rdb.IncrByFloat(ctx, fmt.Sprintf("chaos2:session:%s:trust", msg.SessionID), -0.05)
	}

	// state transition uses previous plan (loaded from Redis)
	prevPlan := loadPlan(ctx, msg.SessionID)
	cogCtx.ChaosState = transitionState(ctx, msg.SessionID, cogCtx.Intent, prevPlan)

	// 3. R brain daemon — receives intent + fresh chaos_state every message
	trustVal, _ := rdb.Get(ctx, fmt.Sprintf("chaos2:session:%s:trust", msg.SessionID)).Result()
	if trustVal == "" {
		trustVal = "0"
	}
	cogCtx.RBrain = rDaemon.query(msg.SessionID, cogCtx.Intent, cogCtx.ChaosState, trustVal)
	if cogCtx.RBrain.Plan == "" {
		cogCtx.RBrain.Plan = "observe"
	}
	savePlan(ctx, msg.SessionID, cogCtx.RBrain.Plan)

	// 4. attention — Lua
	runAttention(cogCtx)

	// 5. mirror (Prolog) + regulation (Haskell) + time (Erlang) — parallel
	var wg sync.WaitGroup
	wg.Add(3)
	go func() { defer wg.Done(); runMirror(cogCtx) }()
	go func() { defer wg.Done(); runRegulation(cogCtx) }()
	go func() { defer wg.Done(); runTime(cogCtx, ctx) }()
	wg.Wait()

	// 6. purpose — Forth
	runPurpose(cogCtx)

	// 7. silence — Assembly
	runSilence(cogCtx)

	// 8. Python → Ollama
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
		"chaos_state":  cogCtx.ChaosState,
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
	if err := rDaemon.start(); err != nil {
		log.Printf("[R] daemon unavailable: %v — using fallback", err)
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
