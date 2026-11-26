#!/usr/bin/env python3
"""Print the officers' and classes' trait plans from the data as markdown tables.

Reads assets/data only, so the tables never drift from the game: each officer's own trait plan
(`Traits`, one more every Rules.TraitPlanStep levels; `TraitAt`; `Learn`) and each class's `Plan` and `Pool`.
Run from any directory: python3 tools/scenario_doc/trait_plans.py [--officers | --classes]
"""
import argparse
import json
from pathlib import Path

DATA = Path(__file__).resolve().parents[2] / "assets/data"


def load(name, key):
    return json.loads((DATA / name).read_text(encoding="utf-8"))[key]


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    group = parser.add_mutually_exclusive_group()
    group.add_argument("--officers", action="store_true", help="only the officer table")
    group.add_argument("--classes", action="store_true", help="only the class table")
    args = parser.parse_args()

    step = load("game.json", "Rules")["TraitPlanStep"]
    traits = load("traits.json", "Traits")
    classes = load("classes.json", "Classes")
    officers = load("characters.json", "Characters")
    officers.update(load("characters_story.json", "Characters"))

    def name(trait):
        return traits.get(trait, {}).get("Name", trait)

    def names(ids):
        return " · ".join(name(t) for t in ids) or "—"

    def own_plan(officer):
        return " · ".join(f"Lv{i * step + 1} {name(t)}" for i, t in enumerate(officer.get("Traits", []))) or "—"

    def at_plan(officer):
        at = officer.get("TraitAt", {})
        return " · ".join(f"Lv{lv} {name(t)}" for t, lv in sorted(at.items(), key=lambda kv: kv[1])) or "—"

    def class_plan(cls):
        return " · ".join(f"Lv{p['Level']} {name(p['Trait'])}" for p in cls.get("Plan", [])) or "—"

    if not args.classes:
        for title, playable in (("플레이어 장수", True), ("그 밖의 장수", False)):
            print(f"### {title}\n")
            print("| 장수 | 병종 | 고유 특성 계획 | `TraitAt` | `Learn` |")
            print("|---|---|---|---|---|")
            for officer in officers.values():
                if bool(officer.get("Playable")) != playable:
                    continue
                cls = classes.get(officer.get("Class", ""), {}).get("Name", officer.get("Class", ""))
                print(f"| {officer.get('Name', '')} | {cls} | {own_plan(officer)} | {at_plan(officer)} | {names(officer.get('Learn', []))} |")
            print()
    if not args.officers:
        print("### 병종\n")
        print("| 병종 | 특성 계획 (`Plan`) | 학습 풀 (`Pool`) |")
        print("|---|---|---|")
        for cls in classes.values():
            print(f"| {cls.get('Name', '')} | {class_plan(cls)} | {names(cls.get('Pool', []))} |")


if __name__ == "__main__":
    main()
