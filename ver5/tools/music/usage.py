#!/usr/bin/env python3
"""Counts how often each theme plays: per chapter (scenes, explore areas, battles) and in total.

Resolution follows internal/gui/sound.go musicFor: a scene's Music, else Maps[map], else Story;
an explore area's Maps[map], else Explore; a battle's Stages[id], else Battle.
Usage: python3 tools/music/usage.py [--chapter 12] [--track suno-story]
"""
import argparse
import glob
import json
import os
import re
from collections import Counter, defaultdict

ROOT = os.path.join(os.path.dirname(os.path.abspath(__file__)), "..", "..", "assets")
HERO = os.path.join(ROOT, "scenario", "hero")


def load(path):
    with open(path, encoding="utf-8") as f:
        return json.load(f)


def chapter_of(path):
    return re.match(r"[a-z_]*?(\d+[a-z]*)", os.path.basename(path)).group(1)


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("--chapter")
    ap.add_argument("--track")
    args = ap.parse_args()

    m = load(os.path.join(ROOT, "music", "music.json"))
    maps, stages = m.get("Maps", {}), m.get("Stages", {})
    uses = defaultdict(Counter)  # chapter -> (kind, track) -> count
    total = defaultdict(Counter)  # track -> kind -> count
    how = defaultdict(Counter)  # track -> scene/map/default -> count

    def add(ch, kind, track, src):
        uses[ch][(kind, track)] += 1
        total[track][kind] += 1
        how[track][src] += 1

    for f in sorted(glob.glob(os.path.join(ROOT, "scenes", "ch*.json"))):
        ch = chapter_of(f)
        for s in load(f)["Scenes"].values():
            if s.get("Music"):
                add(ch, "scene", s["Music"], "scene")
            elif maps.get(s.get("Map")):
                add(ch, "scene", maps[s["Map"]], "map")
            else:
                add(ch, "scene", m["Story"], "default")

    for f in sorted(glob.glob(os.path.join(HERO, "explore*.json"))):
        ch = chapter_of(f)
        areas = load(f).get("Areas", [])
        for a in areas.values() if isinstance(areas, dict) else areas:
            mp = a.get("Map") if isinstance(a, dict) else a
            if maps.get(mp):
                add(ch, "explore", maps[mp], "map")
            else:
                add(ch, "explore", m["Explore"], "default")

    for f in sorted(glob.glob(os.path.join(HERO, "stage*.json"))):
        ch = chapter_of(f)
        sid = "B" + os.path.basename(f)[5:-5].upper()
        if stages.get(sid):
            add(ch, "battle", stages[sid], "map")
        else:
            add(ch, "battle", m["Battle"], "default")

    def key(c):
        return (int(re.match(r"\d+", c).group()), c)

    print("== 장별 (scene=장면, explore=탐험 구역, battle=전투) ==")
    for ch in sorted(uses, key=key):
        if args.chapter and ch != args.chapter.lstrip("0"):
            if ch.lstrip("0") != args.chapter.lstrip("0"):
                continue
        parts = []
        for (kind, track), n in sorted(uses[ch].items(), key=lambda kv: (kv[0][0], -kv[1], kv[0][1])):
            if args.track and track != args.track:
                continue
            parts.append(f"{kind}:{track}×{n}")
        if parts:
            print(f"ch{ch}: " + "  ".join(parts))

    for title, kinds in (("탐험·시나리오(장면+탐험)", ("scene", "explore")), ("전투", ("battle",))):
        sub = {t: sum(k[x] for x in kinds) for t, k in total.items()}
        sub = {t: n for t, n in sub.items() if n}
        print(f"\n== 곡별 합계: {title} ({sum(sub.values())}곳, {len(sub)}곡) ==")
        for track, n in sorted(sub.items(), key=lambda kv: -kv[1]):
            if args.track and track != args.track:
                continue
            print(f"{n:4d}  {track}")


if __name__ == "__main__":
    main()
