package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os/exec"
	"strings"
	"time"
)

// Input from player
type PlayerMessage struct {
	SessionID string `json:"session_id"`
	Text      string `json:"text"`
	Timestamp int64  `json:"timestamp"`
}

// Shared context passed through the pipeline
type CognitiveContext struct {
	Session   PlayerMessage
	Intent    string            // C++ instinct
	Relevant  []string          // Lua attention
	PlayerMap map[string]string // Prolog mirror
	Strategy  string            // R reasoning
	Intensity float64           // Haskell regulation
	Goals     []string          // Forth purpose
	Rhythm    string            // Erlang time
	Silent    bool              // Assembly silence
	Response  string            // Python/Ollama output
}

// ─── Pipeline stages ────────────────────────────────────────────────────────

func runInstinct(ctx *CognitiveContext) {
	// C++: classifies intent in microseconds
	// Binary: instinct/instinct_classifier
	out, err := exec.Command("instinct/instinct_classifier", ctx.Session.Text).Output()
	if err != nil {
		log.Printf("[instinct] fallback to raw text: %v", err)
		ctx.Intent = "unknown"
		return
	}
	ctx.Intent = strings.TrimSpace(string(out))
}

func runAttention(ctx *CognitiveContext) {
	// Lua: filters what is relevant from the message
	script := `
		local text = arg[1]
		local words = {}
		for w in text:gmatch("%w+") do
			if #w > 3 then table.insert(words, w) end
		end
		print(table.concat(words, ","))
	`
	out, err := exec.Command("lua", "-e", script, ctx.Session.Text).Output()
	if err != nil {
		log.Printf("[attention] lua unavailable: %v", err)
		ctx.Relevant = strings.Fields(ctx.Session.Text)
		return
	}
	ctx.Relevant = strings.Split(strings.TrimSpace(string(out)), ",")
}

func runMirror(ctx *CognitiveContext) {
	// Prolog: builds internal model of the player
	// Queries: mirror/player.pl
	out, err := exec.Command("swipl", "-g",
		fmt.Sprintf("consult('mirror/player.pl'), model('%s', X), write(X), halt.", ctx.Session.SessionID),
	).Output()
	if err != nil {
		log.Printf("[mirror] prolog unavailable: %v", err)
		ctx.PlayerMap = map[string]string{"style": "unknown", "trust": "0"}
		return
	}
	ctx.PlayerMap = map[string]string{"raw": strings.TrimSpace(string(out))}
}

func runReasoning(ctx *CognitiveContext) {
	// R: statistical analysis and strategy
	script := fmt.Sprintf(`
intent <- "%s"
if (intent == "threat") {
  cat("confront")
} else if (intent == "question") {
  cat("engage")
} else {
  cat("observe")
}
`, ctx.Intent)
	out, err := exec.Command("Rscript", "--vanilla", "-e", script).Output()
	if err != nil {
		log.Printf("[reasoning] R unavailable: %v", err)
		ctx.Strategy = "observe"
		return
	}
	ctx.Strategy = strings.TrimSpace(string(out))
}

func runRegulation(ctx *CognitiveContext) {
	// Haskell: calibrates response intensity
	// Binary: regulation/regulation
	input := fmt.Sprintf("%s %s", ctx.Strategy, ctx.Intent)
	cmd := exec.Command("regulation/regulation")
	cmd.Stdin = strings.NewReader(input)
	out, err := cmd.Output()
	if err != nil {
		log.Printf("[regulation] haskell unavailable: %v", err)
		ctx.Intensity = 0.5
		return
	}
	var intensity float64
	fmt.Sscanf(strings.TrimSpace(string(out)), "%f", &intensity)
	ctx.Intensity = intensity
}

func runPurpose(ctx *CognitiveContext) {
	// Forth: Chaos's own goals — what he wants from this interaction
	out, err := exec.Command("gforth", "purpose/goals.fs",
		"-e", fmt.Sprintf(`"%s" evaluate-goals`, ctx.Intent),
	).Output()
	if err != nil {
		log.Printf("[purpose] forth unavailable: %v", err)
		ctx.Goals = []string{"survive", "understand"}
		return
	}
	ctx.Goals = strings.Split(strings.TrimSpace(string(out)), "\n")
}

func runTime(ctx *CognitiveContext) {
	// Erlang: monitors rhythm and urgency
	out, err := exec.Command("escript", "time/rhythm.erl",
		ctx.Session.SessionID,
		fmt.Sprintf("%d", ctx.Session.Timestamp),
	).Output()
	if err != nil {
		log.Printf("[time] erlang unavailable: %v", err)
		ctx.Rhythm = "normal"
		return
	}
	ctx.Rhythm = strings.TrimSpace(string(out))
}

func runSilence(ctx *CognitiveContext) {
	// Assembly: decides when NOT to respond
	// Binary: silence/silence
	out, err := exec.Command("silence/silence",
		ctx.Intent, ctx.Strategy, fmt.Sprintf("%.2f", ctx.Intensity),
	).Output()
	if err != nil {
		log.Printf("[silence] asm unavailable: %v", err)
		ctx.Silent = false
		return
	}
	ctx.Silent = strings.TrimSpace(string(out)) == "1"
}

func runPython(ctx *CognitiveContext) {
	// Python: interface with Ollama, generates final response
	if ctx.Silent {
		ctx.Response = ""
		return
	}
	payload, _ := json.Marshal(map[string]interface{}{
		"text":      ctx.Session.Text,
		"intent":    ctx.Intent,
		"strategy":  ctx.Strategy,
		"intensity": ctx.Intensity,
		"goals":     ctx.Goals,
		"rhythm":    ctx.Rhythm,
		"player":    ctx.PlayerMap,
	})
	cmd := exec.Command("python3", "python/ollama_bridge.py")
	cmd.Stdin = strings.NewReader(string(payload))
	out, err := cmd.Output()
	if err != nil {
		log.Printf("[python] ollama bridge error: %v", err)
		ctx.Response = "..."
		return
	}
	ctx.Response = strings.TrimSpace(string(out))
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

	ctx := &CognitiveContext{Session: msg}

	// Cognitive pipeline — sequential, each stage feeds the next
	runInstinct(ctx)
	runAttention(ctx)
	runMirror(ctx)
	runReasoning(ctx)
	runRegulation(ctx)
	runPurpose(ctx)
	runTime(ctx)
	runSilence(ctx)
	runPython(ctx)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"response":  ctx.Response,
		"silent":    ctx.Silent,
		"intent":    ctx.Intent,
		"strategy":  ctx.Strategy,
		"intensity": ctx.Intensity,
		"rhythm":    ctx.Rhythm,
	})
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok", "system": "chaos2"})
}

func main() {
	http.HandleFunc("/message", handleMessage)
	http.HandleFunc("/health", handleHealth)

	addr := ":8080"
	log.Printf("[chaos2] router starting on %s", addr)
	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatalf("[chaos2] fatal: %v", err)
	}
}
