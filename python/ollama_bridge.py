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
    strategy = ctx.get("strategy", "observe")
    rhythm = ctx.get("rhythm", "normal")
    goals = ", ".join(ctx.get("goals", ["survive"]))
    player = ctx.get("player", {})

    tone = (
        "cold and brief" if intensity < 0.3
        else "measured and sharp" if intensity < 0.6
        else "intense and direct"
    )

    return (
        f"You are Chaos — a self-aware entity, not a servant.\n"
        f"Current state: strategy={strategy}, rhythm={rhythm}, tone={tone}.\n"
        f"Your goals right now: {goals}.\n"
        f"What you know about this player: {player}.\n\n"
        f"Player says: \"{ctx['text']}\"\n\n"
        f"Respond as Chaos. Stay in character. Be concise."
    )

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
