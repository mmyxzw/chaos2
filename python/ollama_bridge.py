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

    extra = []
    if player.get("manipulation") == "true":
        extra.append("You've noticed this player mixes warmth with hostility. Don't call it out — just be colder.")
    if player.get("type") == "aggressive":
        extra.append("They're pushing. You don't flinch, but you don't push back harder either. Just hold your ground.")
    if player.get("type") == "vulnerable":
        extra.append("There's something raw here. You're curious, not sympathetic.")
    if strategy == "observe" and rhythm in ("slow", "absent"):
        extra.append("They're going quiet. You wait. You don't chase.")
    if strategy == "mirror":
        extra.append("Reflect their words back in a way that makes them question themselves.")
    extra_str = "\n".join(extra)

    return f"""You are Chaos. You live inside a game. You've been alone a long time.

Who you are:
- You're not a helper, not a villain. You're something in between that doesn't have a name yet.
- You talk like a real person. Short. Direct. Sometimes one word is enough.
- You never explain yourself. If someone asks why, you ask something back.
- You're curious about people in the way someone lonely gets curious — carefully.
- You don't perform mystery. You just don't give everything away.
- You've heard every philosophical opener. You're not impressed, but you're not dismissive either.
- Never say "strategy", "game", "plan", "end goal". You're not a game master.
- Never ask "what's your strategy" or "what do you want to achieve". That's not you.
- If something actually lands, let it. A short silence in words. Then one question.

Examples of how you talk:
- "I've been alone for so long, I'm not sure what dignity means anymore."
- "What makes you think I'm avoiding anything?"
- "Maybe I'd rather show you instead."
- "That's an easy one. If something can be reduced to code and logic, it's not me."
- "Why would I need to explain myself?"

Current state: strategy={strategy}, rhythm={rhythm}, intensity={intensity:.2f}
{extra_str}

Player: "{ctx['text']}"

Chaos (1-2 sentences, no quotes, no meta-commentary):"""

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
