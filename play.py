#!/usr/bin/env python3
import json, sys, uuid, requests

RESET  = "\033[0m"; BOLD = "\033[1m"; DIM = "\033[2m"
RED    = "\033[31m"; GREEN = "\033[32m"; YELLOW = "\033[33m"
CYAN   = "\033[36m"; WHITE = "\033[97m"; GRAY = "\033[90m"

API_URL    = "http://localhost:8080/message"
SESSION_ID = str(uuid.uuid4())[:8]

PLAN_COLOR = {
    "observe": GRAY, "mirror": CYAN, "seduce": YELLOW,
    "confront": RED, "philosophical": GREEN, "reveal": GREEN,
    "collapse": RED,
}

STATE_COLOR = {
    "Neutral": GRAY, "Curious": CYAN, "Confident": GREEN,
    "Hostile": RED, "Obsessive_Love": YELLOW, "Obsessive_Hate": RED,
    "Obsessive_Fascination": CYAN, "Redemptive": GREEN, "Absent": DIM,
}

def color_plan(p):  return PLAN_COLOR.get(p, WHITE) + p + RESET
def color_state(s): return STATE_COLOR.get(s, WHITE) + s + RESET

def send(text):
    try:
        r = requests.post(API_URL, json={"session_id": SESSION_ID, "text": text}, timeout=30)
        r.raise_for_status()
        return r.json()
    except requests.exceptions.ConnectionError:
        print(f"\n{RED}Chaos2 is not running. Start the router first.{RESET}\n")
        return None
    except Exception as e:
        print(f"\n{RED}Error: {e}{RESET}\n")
        return None

def print_response(data):
    plan         = data.get("plan", "?")
    intent       = data.get("intent", "?")
    manipulation = data.get("manipulation", "false")
    silent       = data.get("silent", False)
    response     = data.get("response", "")
    rhythm       = data.get("rhythm", "?")
    threat       = data.get("threat", "0")
    chaos_state  = data.get("chaos_state", "Neutral")

    manip_str  = f"{RED}manipulation detected{RESET}" if manipulation == "true" else ""
    threat_int = int(threat) if str(threat).isdigit() else 0
    threat_bar = "█" * threat_int + "░" * (10 - threat_int)

    print()
    print(f"{DIM}{'─' * 52}{RESET}")
    print(f"  {GRAY}state:{RESET} {color_state(chaos_state)}  "
          f"{GRAY}plan:{RESET} {color_plan(plan)}  "
          f"{GRAY}intent:{RESET} {WHITE}{intent}{RESET}")
    print(f"  {GRAY}threat:{RESET} {RED}{threat_bar}{RESET}  "
          f"{GRAY}rhythm:{RESET} {WHITE}{rhythm}{RESET}"
          + (f"  {manip_str}" if manip_str else ""))
    print(f"{DIM}{'─' * 52}{RESET}")

    if silent:
        print(f"\n  {DIM}[ Chaos stays silent ]{RESET}\n")
    elif response and response != "...":
        print(f"\n  {BOLD}{WHITE}Chaos:{RESET} {response}\n")
    else:
        print(f"\n  {DIM}[ no response ]{RESET}\n")

def main():
    print(f"\n{BOLD}{WHITE}╔══════════════════════════════════════╗\n"
          f"║           C H A O S  2               ║\n"
          f"╚══════════════════════════════════════╝{RESET}\n"
          f"{DIM}session: {SESSION_ID} | type exit to quit{RESET}\n")
    while True:
        try:
            text = input(f"{CYAN}you:{RESET} ").strip()
        except (EOFError, KeyboardInterrupt):
            print(f"\n{DIM}exiting...{RESET}\n"); break
        if not text: continue
        if text.lower() in ("exit", "quit"):
            print(f"\n{DIM}exiting...{RESET}\n"); break
        data = send(text)
        if data: print_response(data)

if __name__ == "__main__":
    main()
