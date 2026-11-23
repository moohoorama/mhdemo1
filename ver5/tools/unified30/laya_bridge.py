#!/usr/bin/env python3
"""Laya choice bridge for the 30-minute unified-model comparison (temporary tool).

Reads one JSON request per line on stdin: {"id", "state", "options": [text, ...]} and writes one JSON line per request:
{"id", "choice", "calls", "seconds"}. The state is the public battle text; each option is one legal command with the
game's own preview numbers. Options are scored by the Laya decision head in groups of --chunk (a choice question
holds at most that many options within Laya's 192-token option budget); the best option of each group moves on, until
one remains. Every chosen option therefore comes from a model call, and calls and seconds are reported per request.

Run with an isolated interpreter, since the Laya checkpoint directory is untrusted third-party code:
    <python> -I tools/unified30/laya_bridge.py --laya <laya dir> [--head <trained head .pt>]
"""
import argparse
import json
import os
import sys
import time

INSTRUCTIONS = ("Choose the legal command the army should take now in this battle to win without defeat, "
                "gain experience, and keep items and money for later battles.")
LABELS = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("--laya", required=True)
    ap.add_argument("--head", default="")
    ap.add_argument("--chunk", type=int, default=8)
    args = ap.parse_args()
    if not 2 <= args.chunk <= len(LABELS):
        raise SystemExit("chunk must be between 2 and 26")
    laya = os.path.abspath(args.laya)
    sys.path.insert(0, laya)
    import torch
    from rl_agent_api import RLAgent

    agent = RLAgent(laya, device="cpu")
    if args.head:
        head = torch.load(args.head, map_location="cpu")
        _, unexpected = agent.model.load_state_dict(head, strict=False)
        if unexpected:
            raise SystemExit("unexpected head keys: %s" % unexpected[:5])
    agent.model.eval()

    def ask(state, options):
        """Probabilities over one group of options (at most len(LABELS))."""
        labels = LABELS[:len(options)]
        q = {"type": "choice", "instructions": INSTRUCTIONS, "criteria": dict(zip(labels, options))}
        answer = agent.system_one(state, {"act": q})["answers"]["act"]
        return [answer["probabilities"][label] for label in labels]

    for line in sys.stdin:
        if not line.strip():
            continue
        req = json.loads(line)
        started = time.perf_counter()
        remaining = list(range(len(req["options"])))
        calls = 0
        error = ""
        try:
            if not remaining:
                raise ValueError("no options")
            while len(remaining) > 1:
                winners = []
                for start in range(0, len(remaining), args.chunk):
                    group = remaining[start:start + args.chunk]
                    if len(group) == 1:
                        winners.append(group[0])
                        continue
                    probs = ask(req["state"], [req["options"][i] for i in group])
                    calls += 1
                    winners.append(group[max(range(len(group)), key=lambda j: probs[j])])
                remaining = winners
            choice = remaining[0]
        except Exception as exc:  # reported to the Go side, which stops the run
            choice, error = -1, "%s: %s" % (type(exc).__name__, exc)
        print(json.dumps({"id": req["id"], "choice": choice, "calls": calls,
                          "seconds": time.perf_counter() - started, "error": error}), flush=True)


if __name__ == "__main__":
    main()
