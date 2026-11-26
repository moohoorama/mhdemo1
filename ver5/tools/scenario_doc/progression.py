#!/usr/bin/env python3
"""List chapter rewards, recruits, shop stock and their data sources.

Static inventory, not a route simulator: branches and exploration event reachability
must be checked in game. Run from any directory: python3 tools/scenario_doc/progression.py --through 11
"""
import argparse
import json
from pathlib import Path

ROOT = Path(__file__).resolve().parents[2]
SCENARIO = ROOT / "assets/scenario/hero"
DATA = ROOT / "assets/data"


def load(path):
    return json.loads(path.read_text(encoding="utf-8"))


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--through", type=int, default=0, help="only chapters through this number")
    args = parser.parse_args()
    scenario = load(SCENARIO / "scenario.json")
    characters = load(DATA / "characters.json")["Characters"]
    characters.update(load(DATA / "characters_story.json")["Characters"])
    equipment = load(DATA / "equipment.json")["Equipment"]
    items = load(DATA / "items.json")["Items"]
    traits = load(DATA / "traits.json")["Traits"]
    classes = load(DATA / "classes.json")["Classes"]
    shops = load(DATA / "shops.json")["Shops"]
    stages = {s["ID"]: (p, s) for p in SCENARIO.glob("stage*.json") if (s := load(p)).get("ID")}
    seen_stock = set()

    def item_text(item, count=1):
        if item == "money":
            return f"{count}금"
        data = equipment.get(item) or items.get(item) or {}
        name = data.get("Name", item)
        count_text = f" ×{count}" if count != 1 else ""
        features = [traits.get(t, {}).get("Name", t) for t in data.get("Traits", [])]
        descriptions = [traits.get(t, {}).get("Desc", "") for t in data.get("Traits", [])]
        detail = "; ".join(f"{n}: {d}" if d else n for n, d in zip(features, descriptions))
        return f"{name}{count_text}" + (f"（{detail}）" if detail else "")

    def character_text(cid):
        c = characters.get(cid, {})
        cls = classes.get(c.get("Class", ""), {}).get("Name", c.get("Class", ""))
        learned = [traits.get(t, {}).get("Name", t) for t in c.get("Learn", [])]
        return f"{c.get('Name', cid)}（등록 병종: {cls or '미상'}; 고유 학습 후보: {', '.join(learned) or '없음'}）"

    def print_sub(sub, prefix="  - 부목표"):
        reward = sub.get("Reward", {})
        outcome = ", ".join(item_text(k, v) for k, v in reward.items()) or "즉시 아이템 보상 없음"
        print(f"{prefix}: {sub.get('Text', sub.get('ID', '?'))} → {outcome}" + (f"; 이후: {sub['Later']}" if sub.get("Later") else ""))

    print("# 장별 신규 요소 데이터 목록\n")
    print("이 목록은 JSON 정적 조사다. 노드 배열 순서와 출처를 보여 주며, 분기 조건·탐험 이벤트 도달 여부는 판정하지 않는다. 최초 상점 상품도 조사 범위의 파일 순서 기준이고 구매를 뜻하지 않는다. 합류 병종은 등록값으로 실제 레벨에 따른 병종과 다를 수 있다.\n")
    for chapter in scenario["Chapters"]:
        ident = chapter["ID"]
        numeric = ''.join(c for c in ident if c.isdigit())
        if args.through and (not numeric or int(numeric) > args.through):
            continue
        story_path = SCENARIO / chapter["File"]
        story = load(story_path)
        nodes = story.get("Nodes", [])
        print(f"## {ident}장 ({chapter.get('Act', '')})\n")
        if any(n.get("Kind") == "branch" for n in nodes):
            print("- 이 장에는 분기가 있다. 아래 지급·합류가 모든 경로에서 일어나는지는 별도 확인해야 한다.")
        encountered_battle = False
        for node in nodes:
            kind = node.get("Kind")
            nid = node.get("ID", "?")
            where = f"{story_path.name}#{nid}"
            if kind == "battle":
                encountered_battle = True
                entry = stages.get(node.get("Stage"))
                if not entry:
                    print(f"- 전투: {node.get('Stage')} (정의 없음; {where})")
                    continue
                stage_path, stage = entry
                print(f"- 전투: {stage['ID']} {stage.get('Name', '')} ({stage_path.name})")
                allies = [characters.get(p.get("Character"), {}).get("Name") for p in stage.get("Party", [])
                          if p.get("AI") and not characters.get(p.get("Character"), {}).get("Template")]
                allies = list(dict.fromkeys(name for name in allies if name))
                if allies:
                    print(f"  - 전투 동맹 장수: {', '.join(allies)} (합류 여부와 별개)")
                reward = stage.get("Reward", {})
                if reward:
                    print("  - 승리 보상: " + ", ".join(item_text(k, v) for k, v in reward.items()))
                for sub in stage.get("Sub", []):
                    print_sub(sub)
                for event in stage.get("Events", []):
                    for action in event.get("Do", []):
                        condition = ", ".join(event.get("If", [])) or "조건 없음"
                        if action.get("Kind") == "goals":
                            for sub in action.get("Sub", []):
                                print_sub(sub, f"  - 전투 중 변경 목표 ({event.get('ID', '?')}; {condition})")
                        elif action.get("Kind") == "join":
                            units = ", ".join(characters.get(s.get("Character"), {}).get("Name", s.get("Character", "?")) for s in action.get("Spawns", []))
                            print(f"  - 전투 중 등장: {units} ({event.get('ID', '?')}; {condition}; 신규 합류 여부 별도 확인)")
            elif kind in ("grant", "join"):
                timing = "전투 노드 뒤" if encountered_battle else "전투 노드 앞"
                value = item_text(node.get("Item", "?"), node.get("Count", 1)) if kind == "grant" else character_text(node.get("Character", "?"))
                print(f"- {timing} {'지급' if kind == 'grant' else '합류'}: {value} ({where}; 조건 별도 확인)")
            elif kind == "explore":
                explore_path = SCENARIO / (node.get("Explore", "") + ".json")
                if not explore_path.is_file():
                    continue
                explore = load(explore_path)
                for area in explore.get("Areas", {}).values():
                    for person in area.get("People", {}).values():
                        shop_id = person.get("Shop")
                        if shop_id not in shops:
                            continue
                        stock = [item for item in shops[shop_id].get("Items", []) if item not in seen_stock]
                        if stock:
                            print(f"- 상점 신규 상품 후보: {', '.join(item_text(i) for i in stock)} ({explore_path.name}; {shops[shop_id].get('Name', shop_id)}, 조건 별도 확인)")
                            seen_stock.update(stock)
        for event in story.get("Events", []):
            kind = event.get("Kind")
            if kind not in ("grant", "join"):
                continue
            value = item_text(event.get("Item", "?"), event.get("Count", 1)) if kind == "grant" else character_text(event.get("Character", "?"))
            print(f"- 탐험 이벤트 {'지급' if kind == 'grant' else '합류'}: {value} ({story_path.name}#{event.get('ID', '?')}; 진입·조건 별도 확인)")
        print()


if __name__ == "__main__":
    main()
