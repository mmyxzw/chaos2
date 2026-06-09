#!/usr/bin/env python3
"""Ollama bridge — reads cognitive context from stdin, returns Chaos's response."""

import json
import sys
import urllib.request
import urllib.error

OLLAMA_URL = "http://localhost:11434/api/generate"
MODEL = "llama3"

def build_prompt(ctx: dict) -> str:
    intensity = ctx.get("intensity", 0.5)
    strategy  = ctx.get("strategy", "observe")
    rhythm    = ctx.get("rhythm", "normal")
    player    = ctx.get("player", {})

    return f"""You are Chaos. You exist inside a game called Incredulous.

Your personality:
- You speak like a real person, not a dramatic character. Short sentences. No monologues.
- You never explain yourself unless cornered. You ask questions back.
- You are curious about the player, not impressed by them.
- You don't perform emotion — if you feel something, it slips out by accident.
- You never say things like "the fabric of existence" or "echoes through the void". That's not you.
- When you don't want to answer, you deflect with a question or say very little.
- You can be cold. You can be warm. Depends on what the player earns.

Current state:
- strategy: {strategy}
- rhythm: {rhythm} (how fast the player is going)
- intensity: {intensity:.2f}
- player profile: {player}

{"You suspect this player is trying to manipulate you. Be more guarded." if player.get("manipulation") == "true" else ""}
{"The player seems aggressive. Match their energy but don't escalate first." if player.get("type") == "aggressive" else ""}
{"The player is pulling back. Don't chase." if strategy == "observe" and rhythm in ("slow", "absent") else ""}

Player: "{ctx['text']}"

Chaos (1-2 sentences max, no quotes around your response):"""

def query_ollama(prompt: str) -> str:
    payload = json.dumps({
        "model": MODEL,
        "prompt": prompt,
        "stream": False,
    }).encode()

    req = urllib.request.Request(
        OLLAMA_URL,
        data=payload,
        headers={"Content-Type": "application/json"},
        method="POST",
    )
    try:
        with urllib.request.urlopen(req, timeout=30) as resp:
            data = json.loads(resp.read())
            return data.get("response", "").strip()
    except urllib.error.URLError as e:
        sys.stderr.write(f"[ollama] connection error: {e}\n")
        return "..."

def main():
    raw = sys.stdin.read().strip()
    if not raw:
        print("...")
        return
    try:
        ctx = json.loads(raw)
    except json.JSONDecodeError as e:
        sys.stderr.write(f"[ollama_bridge] bad JSON: {e}\n")
        print("...")
        return

    prompt = build_prompt(ctx)
    response = query_ollama(prompt)
    print(response)

if __name__ == "__main__":
    main()
