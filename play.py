#!/usr/bin/env python3
"""
Chaos2 — terminal interface
Usage: python3 play.py
"""

import json
import sys
import uuid
import requests

# ── ANSI colors ───────────────────────────────────────────────────────────────
RESET  = "\033[0m"
BOLD   = "\033[1m"
DIM    = "\033[2m"
RED    = "\033[31m"
GREEN  = "\033[32m"
YELLOW = "\033[33m"
CYAN   = "\033[36m"
WHITE  = "\033[97m"
GRAY   = "\033[90m"

API_URL    = "http://localhost:8080/message"
SESSION_ID = str(uuid.uuid4())[:8]

PLAN_COLOR = {
    "observe":       GRAY,
    "mirror":        CYAN,
    "seduce":        YELLOW,
    "confront":      RED,
    "philosophical": GREEN,
}

def color_plan(plan):
    return PLAN_COLOR.get(plan, WHITE) + plan + RESET

def send(text):
    try:
        r = requests.post(API_URL, json={"session_id": SESSION_ID, "text": text}, timeout=30)
        r.raise_for_status()
        return r.json()
    except requests.exceptions.ConnectionError:
        print(f"\n{RED}Chaos2 não está rodando. Inicie o router primeiro.{RESET}\n")
        return None
    except Exception as e:
        print(f"\n{RED}Erro: {e}{RESET}\n")
        return None

def print_response(data):
    plan        = data.get("plan", "?")
    intent      = data.get("intent", "?")
    manipulation = data.get("manipulation", "false")
    silent      = data.get("silent", False)
    response    = data.get("response", "")
    intensity   = data.get("intensity", 0)
    rhythm      = data.get("rhythm", "?")
    threat      = data.get("threat", "0")

    # status bar
    manip_str = f"{RED}manipulação detectada{RESET}" if manipulation == "true" else ""
    threat_int = int(threat) if str(threat).isdigit() else 0
    threat_bar = ("█" * threat_int) + ("░" * (10 - threat_int))

    print()
    print(f"{DIM}{'─' * 50}{RESET}")
    print(f"  {GRAY}plano:{RESET} {color_plan(plan)}  "
          f"{GRAY}intent:{RESET} {WHITE}{intent}{RESET}  "
          f"{GRAY}ritmo:{RESET} {WHITE}{rhythm}{RESET}")
    print(f"  {GRAY}ameaça:{RESET} {RED}{threat_bar}{RESET}  "
          f"{GRAY}intensidade:{RESET} {WHITE}{intensity:.2f}{RESET}"
          + (f"  {manip_str}" if manip_str else ""))
    print(f"{DIM}{'─' * 50}{RESET}")

    if silent:
        print(f"\n  {DIM}[ Chaos permanece em silêncio ]{RESET}\n")
    elif response and response != "...":
        print(f"\n  {BOLD}{WHITE}Chaos:{RESET} {response}\n")
    else:
        print(f"\n  {DIM}[ Chaos não responde ]{RESET}\n")

def main():
    print(f"""
{BOLD}{WHITE}╔══════════════════════════════════════╗
║           C H A O S  2               ║
╚══════════════════════════════════════╝{RESET}
{DIM}sessão: {SESSION_ID}
digite {BOLD}sair{RESET}{DIM} para encerrar
{RESET}""")

    while True:
        try:
            text = input(f"{CYAN}você:{RESET} ").strip()
        except (EOFError, KeyboardInterrupt):
            print(f"\n{DIM}encerrando...{RESET}\n")
            break

        if not text:
            continue
        if text.lower() in ("sair", "exit", "quit"):
            print(f"\n{DIM}encerrando...{RESET}\n")
            break

        data = send(text)
        if data:
            print_response(data)

if __name__ == "__main__":
    main()
