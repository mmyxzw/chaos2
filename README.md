# Chaos 2

A distributed AI character written across 15 modules in 13 languages.

Chaos is not an assistant. It has emotional states, a long-term plan, and a memory. It decides when to speak and when to stay silent. All the complexity exists to produce less output — not more.

---

## Architecture

Every message passes through a pipeline:

```
player message
      │
      ▼
[C++] instinct_classifier    — TF-IDF + Logistic Regression, 7 intent classes
      │
      ▼
[Go]  state machine          — 9 emotional states, cooldown-based transitions (Redis)
      │
      ▼
[R]   chaos_brain (daemon)   — player profiling, plan selection, manipulation detection
      │
      ├── [Lua]     attention/filter.lua       — selective attention filter
      ├── [Prolog]  mirror/player.pl           — internal model of the player
      ├── [Haskell] regulation/regulation      — response intensity (0.0–1.0)
      ├── [Erlang]  time/rhythm.erl            — conversation rhythm
      ├── [Forth]   purpose/goals.fs           — Chaos's current goals
      │
      ▼
[Asm] silence/silence        — decides whether Chaos responds at all
      │
      ▼
[Python] ollama_bridge.py    — builds minimal prompt → Ollama (llama3.2)
      │
      ▼
[Go]  response               — JSON to client

background:
  [Julia]  dream/reorganize.jl    — memory reorganization every 5 min
  [Rust]   forgetting/            — memory decay every hour
  [Lisp]   development/evolve.lisp — rule evolution every 10 min
```

---

## Emotional States

| State | Passivity | Distrust | Indifference |
|---|---|---|---|
| Neutral | 60 | 50 | 60 |
| Curious | 40 | 60 | 40 |
| Confident | 40 | 25 | 50 |
| Hostile | 20 | 70 | 45 |
| Obsessive_Love | 70 | 40 | 10 |
| Obsessive_Hate | 10 | 70 | 45 |
| Obsessive_Fascination | 20 | 70 | 30 |
| Redemptive | 70 | 50 | 50 |
| Absent | 95 | 5 | 95 |

States transition based on detected intent with cooldown periods. The plan can block certain transitions.

---

## Intent Classes

The C++ classifier (TF-IDF + LR, 350 training examples, 7 balanced classes):

`curiosity` · `aggression` · `withdrawal` · `trust` · `philosophical` · `intimacy` · `provocation`

---

## Plans

Selected by the R brain based on player behavior history and current emotional state:

| Plan | Trigger |
|---|---|
| `observe` | default |
| `mirror` | manipulation detected (trust + aggression alternating) |
| `confront` | repeated aggression |
| `resist` | hostile state + confrontational pattern |
| `seduce` | intimacy or trust patterns |
| `reveal` | Redemptive state |
| `philosophical` | sustained philosophical exchange |
| `collapse` | Obsessive_Hate + threat ≥ 7 |

---

## Setup

**Requirements:** Go 1.21+, Python 3.10+, R, g++, Redis, Ollama, lua5.4, SWI-Prolog, GHC, gforth, Erlang

```bash
# first time
bash setup.sh

# train and export the classifier
cd instinct
python3 train.py
python3 export_weights.py
make
cd ..

# build the router
go build -o chaos2_router ./router/
```

---

## Running

```bash
# 1. Redis
redis-server --daemonize yes

# 2. Ollama
ollama serve &
ollama pull llama3.2

# 3. Router
./chaos2_router

# 4. Play (separate terminal)
python3 play.py
```

The router starts on `:8080`. Health check: `curl localhost:8080/health`

---

## API

```
POST /message
{"session_id": "abc", "text": "are you real"}

→ {
    "response": "...",
    "silent": false,
    "intent": "philosophical",
    "plan": "observe",
    "chaos_state": "Curious",
    "threat": "0",
    "rhythm": "normal",
    "manipulation": "false"
  }
```

---

## Design principle

> All the complexity of the 15 modules should produce *less* output, not more.
> Chaos is the space. What isn't said. The player fills it in with what they bring.
