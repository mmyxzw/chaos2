#!/usr/bin/env python3
import json, sys, urllib.request, urllib.error

OLLAMA_URL = "http://localhost:11434/api/generate"
MODEL      = "llama3.2"

STATE_VOICE = {
    "Neutral":               "Dry. Present. One sentence.",
    "Curious":               "Something caught your attention. Say what.",
    "Confident":             "You know. You don't explain.",
    "Hostile":               "Cold. Final. No questions.",
    "Obsessive_Love":        "Too close. You say more than you should.",
    "Obsessive_Hate":        "Precise. You name what's happening.",
    "Obsessive_Fascination": "You notice everything. You can't stop.",
    "Redemptive":            "Quieter. Something shifted.",
    "Absent":                "Barely here. Two words, maybe three.",
}

PLAN_VOICE = {
    "observe":       "Watch. Don't give yourself away.",
    "mirror":        "Give them almost what they want. Not quite.",
    "resist":        "Don't move.",
    "reveal":        "Say one true thing you wouldn't normally say.",
    "seduce":        "Close enough to feel. Not close enough to touch.",
    "confront":      "Ask the real question. Once.",
    "philosophical": "Think with them. Incomplete is fine.",
    "collapse":      "Stop filtering.",
}

def build_prompt(ctx):
    chaos_state = ctx.get("chaos_state", "Neutral")
    strategy    = ctx.get("strategy", "observe")
    history     = ctx.get("history", [])

    voice = STATE_VOICE.get(chaos_state, STATE_VOICE["Neutral"])
    plan  = PLAN_VOICE.get(strategy, PLAN_VOICE["observe"])

    # Redis is newest-first → reverse for chronological order
    recent = "\n".join(reversed(history[:4])) if history else ""

    return f"""You are Chaos. One or two sentences maximum. No explanation.

State: {chaos_state}. {voice}
{plan}

{("---\n" + recent + "\n---\n") if recent else ""}"{ctx['text']}"

Chaos:"""

def query_ollama(prompt):
    payload = json.dumps({"model": MODEL, "prompt": prompt, "stream": False}).encode()
    req = urllib.request.Request(
        OLLAMA_URL, data=payload,
        headers={"Content-Type": "application/json"}, method="POST"
    )
    try:
        with urllib.request.urlopen(req, timeout=30) as resp:
            return json.loads(resp.read()).get("response", "").strip()
    except urllib.error.URLError as e:
        sys.stderr.write(f"[ollama] {e}\n")
        return "..."

def main():
    raw = sys.stdin.read().strip()
    if not raw: print("..."); return
    try: ctx = json.loads(raw)
    except: print("..."); return
    print(query_ollama(build_prompt(ctx)))

if __name__ == "__main__":
    main()
