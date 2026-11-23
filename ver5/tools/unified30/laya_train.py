#!/usr/bin/env python3
"""Fine-tune the Laya decision head on teacher battle decisions for a fixed time (temporary tool).

The Laya encoder stays frozen; the head, type embedding and scorer are trained with cross-entropy over each group of
options that contains the teacher's choice. Input rows come from `_unifiedlearn export` (one JSON per line:
state, options, label). Writes the trainable weights only (not the encoder) to --out.

    <python> -I tools/unified30/laya_train.py --laya <laya dir> --data decisions.jsonl --out head.pt --minutes 25
"""
import argparse
import json
import os
import random
import sys
import time

INSTRUCTIONS = ("Choose the legal command the army should take now in this battle to win without defeat, "
                "gain experience, and keep items and money for later battles.")
LABELS = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"


def groups(row, size):
    """Option groups that contain the teacher's choice: (state, options, index of the choice in the group)."""
    n, label = len(row["options"]), row["label"]
    for start in range(0, n, size):
        idx = list(range(start, min(n, start + size)))
        if label in idx:
            yield row["state"], [row["options"][i] for i in idx], idx.index(label)


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("--laya", required=True)
    ap.add_argument("--data", required=True)
    ap.add_argument("--out", required=True)
    ap.add_argument("--minutes", type=float, default=25.0)
    ap.add_argument("--lr", type=float, default=1e-4)
    ap.add_argument("--chunk", type=int, default=8)
    ap.add_argument("--accum", type=int, default=8)
    ap.add_argument("--seed", type=int, default=1)
    args = ap.parse_args()
    if not 2 <= args.chunk <= len(LABELS):
        raise SystemExit("chunk must be between 2 and 26")
    laya = os.path.abspath(args.laya)
    sys.path.insert(0, laya)
    import torch
    from rl_agent_api import RLAgent
    from rl_common import build_sequence, collate_items, QTYPES

    torch.manual_seed(args.seed)
    started = time.perf_counter()
    agent = RLAgent(laya, device="cpu")
    model, tok, cfg = agent.model, agent.tok, agent.cfg
    for p in model.encoder.parameters():
        p.requires_grad_(False)
    params = [p for name, p in model.named_parameters() if not name.startswith("encoder.")]
    for p in params:
        p.requires_grad_(True)
    opt = torch.optim.AdamW(params, lr=args.lr, weight_decay=0.01)

    examples = []
    with open(args.data) as f:
        for line in f:
            examples.extend(groups(json.loads(line), args.chunk))
    random.Random(args.seed).shuffle(examples)
    print("examples %d setup %.1fs" % (len(examples), time.perf_counter() - started), flush=True)

    model.train()
    model.encoder.eval()
    qtype = QTYPES["choice"]
    deadline = time.perf_counter() + args.minutes * 60
    seen, steps, running = 0, 0, 0.0
    opt.zero_grad()
    for state, opts, label in examples:
        if time.perf_counter() > deadline:
            break
        q = {"t": "choice", "ins": INSTRUCTIONS, "crit": dict(zip(LABELS, opts))}
        seq, markers = build_sequence(tok, state, q, cfg["max_len"], cfg["head_max_len"])
        if len(markers) != len(opts):
            seen += 1
            continue
        item = {"ids": seq, "markers": markers, "qtype": qtype, "target": [0.0] * len(markers), "label": label,
                "episode": 0, "ep_step": 0, "ep_len": 1, "src": "train"}
        b = collate_items([[item]], tok.pad_token_id)
        logits, _ = model(b["input_ids"], b["attention_mask"], b["marker_pos"], b["marker_mask"], b["qtype"])
        loss = torch.nn.functional.cross_entropy(logits[:, :len(opts)], torch.tensor([label]))
        (loss / args.accum).backward()
        running += float(loss)
        seen += 1
        if seen % args.accum == 0:
            torch.nn.utils.clip_grad_norm_(params, 1.0)
            opt.step()
            opt.zero_grad()
            steps += 1
        if seen % (args.accum * 25) == 0:
            print("seen %d steps %d mean_loss %.4f elapsed %.0fs" % (seen, steps, running / (args.accum * 25),
                                                                   time.perf_counter() - started), flush=True)
            running = 0.0
    if seen % args.accum:
        opt.step()
        steps += 1
    trainable = {name: p.detach().clone() for name, p in model.named_parameters() if not name.startswith("encoder.")}
    torch.save(trainable, args.out)
    print("done seen %d optimizer_steps %d elapsed %.0fs saved %s" % (seen, steps, time.perf_counter() - started, args.out),
          flush=True)


if __name__ == "__main__":
    main()
