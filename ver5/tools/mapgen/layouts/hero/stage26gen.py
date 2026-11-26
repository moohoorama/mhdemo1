#!/usr/bin/env python3
"""Generate the romance Changban stage on the shared map. Run from ver5/."""

import json
from pathlib import Path

HERO = Path("assets/scenario/hero")
CAMP = 70
BRIDGEHEAD = 74


def point(x, y):
    return {"X": x, "Y": y}


def area(number, x, y, w, h, name, description):
    return {"ID": number, "Rect": dict(point(x, y), W=w, H=h),
            "Name": name, "Description": description}


def unit(character, x, y, group, policy, facing, **fields):
    return dict(point(x, y), Character=character, Group=group,
                Policy=policy, Facing=facing, **fields)


def move(actor, x, y, facing):
    return {"Kind": "move", "Unit": actor, "To": point(x, y), "Facing": facing}


def line(actor, text):
    return {"Kind": "say", "Unit": actor, "Text": text}


def play(actor, motion):
    return {"Kind": "play", "Unit": actor, "Motion": motion}


def scene(*beats):
    return {"Kind": "cutscene", "Steps": [list(beat) for beat in beats]}


def foes(group, entries, policy, facing, **fields):
    return [
        unit(c, x, y, group, policy, facing,
             **({"Shift": -0.12} if c.startswith("troop_") else {"Rank": "officer"}),
             **fields)
        for c, x, y in entries
    ]


# The pursuers already face the loose pockets of troops. Contact follows from
# their positions and ordinary movement rather than a timer or a stat bonus.
enemies = foes("wenpin-force", [
    ("wenpin", 19, 12), ("troop_footman", 18, 13),
    ("troop_cavalry", 20, 11), ("troop_footman", 17, 10),
], "assault", "SW")
enemies += foes("zhanghe-force", [
    ("zhanghe", 13, 17), ("troop_footman", 12, 18),
    ("troop_footman", 11, 15), ("troop_footman", 10, 13),
], "march", "SE", GoalArea=75)
enemies += foes("xuchu-force", [
    ("xuchu", 10, 19), ("troop_footman", 11, 20), ("troop_footman", 13, 16),
    ("troop_cavalry", 16, 16),
], "march", "SW", GoalArea=75)
enemies += foes("archers", [
    ("troop_archer", 21, 12), ("troop_archer", 14, 18),
    ("troop_archer", 5, 17),
], "hold", "SW")
enemies += foes("rear", [
    ("troop_footman", 5, 10), ("troop_footman", 5, 23),
], "march", "SE", GoalArea=75)
enemies += [
    unit("caochun", 11, 17, "tiger", "march", "SE", GoalArea=BRIDGEHEAD, Rank="commander"),
    *[unit("troop_cavalry", x, y, "tiger", "march", "SE", GoalArea=BRIDGEHEAD,
           Shift=-0.19) for x, y in ((12, 16), (12, 17), (10, 17))],
    unit("xiahouen", 23, 17, "xiahouen", "react", "SE", Rank="officer"),
]

remnants = [
    unit("troop_footman", x, y, "remnants", "hold", facing,
         Rank="troop", AI=True, HPPct=66, Faction="shu")
    for x, y, facing in (
        (16, 12, "SE"), (19, 15, "NE"), (19, 19, "NW"),
        (12, 14, "NE"), (14, 16, "SW"), (15, 22, "NE"),
    )
]
tiger_wave = [
    unit("troop_cavalry", x, 0, "tiger", "march", "SW", GoalArea=BRIDGEHEAD,
         Shift=-0.19, Side="enemy") for x in (13, 12, 11, 10)
]
bridge_force = [
    unit("xiahoudun", 17, 23, "bridge-command", "hold", "SW", Rank="commander", Side="enemy"),
    unit("troop_footman", 16, 24, "bridge-foot", "hold", "SW", Rank="troop", Side="enemy"),
    unit("troop_cavalry", 18, 24, "bridge-horse", "hold", "SW", Rank="troop", Side="enemy"),
    unit("troop_footman", 19, 23, "bridge-spotter", "hold", "SW", Rank="troop", Side="enemy"),
]

duel = json.loads((HERO / "stage25v.json").read_text(encoding="utf-8"))["Duels"][0]
duel["ID"] = "changban-zhaoyun-xiahouen"

stage = {
    "ID": "B26", "Name": "장판파 전투", "Width": 0, "Height": 0, "Max": 4,
    "Required": ["zhangfei", "zhaoyun"], "Excluded": ["liubei"], "StrictVictory": True,
    "Victory": [{"If": ["at:zhaoyun:70", "at:zhangfei:70"],
                 "Text": "조운과 장비가 다리 뒤 진영에 닿는다"}],
    "Defeat": [
        {"If": ["defeated:zhaoyun"], "Text": "조운 퇴각"},
        {"If": ["defeated:zhangfei"], "Text": "장비 퇴각"},
        {"If": ["at:enemy:70"], "Text": "적이 후방 진영에 닿는다"},
    ],
    "Sub": [
        {"ID": "crossing", "If": ["at:zhaoyun:70"], "Text": "조운이 다리를 건넌다"},
        {"ID": "qinggang", "If": ["duel:changban-zhaoyun-xiahouen"],
         "Text": "조운이 일기토로 하후은을 벤다", "Reward": {"qinggang_sword": 1}},
    ],
    "TurnLimit": 20, "Map": "map_changban.json",
    "Allies": [point(24, 15), point(17, 30), point(18, 30), point(16, 30)],
    "Recommend": {"zhaoyun": point(24, 15), "zhangfei": point(17, 30)},
    "Groups": [
        {"Name": "조운", "Area": 72, "Required": ["zhaoyun"]},
        {"Name": "장판교", "Area": 73, "Required": ["zhangfei"]},
    ],
    "Enemies": enemies, "Party": remnants, "Duels": [duel],
    "Reward": {"money": 2000, "mirror_armor": 1},
    "Areas": [
        area(CAMP, 16, 31, 3, 3, "다리 뒤 진영", "조운과 장비가 철수할 곳"),
        area(72, 24, 15, 1, 1, "무너진 마을", "아두를 구한 조운의 출발점"),
        area(73, 16, 29, 4, 2, "다리 어귀", "장비와 지원 병력의 출발점"),
        area(BRIDGEHEAD, 13, 22, 3, 3, "장판교 앞", "추격군이 모이는 곳"),
        area(75, 10, 17, 4, 4, "들판 서쪽", "조조군이 흩어진 적을 에워싸는 곳"),
    ],
    "Events": [
        {"ID": "opening", "Do": [
            scene((move("zhangfei", 17, 27, "NE"),),
                  (line("zhangfei", "이 장익덕이 여기 있는 한, 아무도 이 다리를 못 건너유!"),),
                  (line("zhaoyun", "공자를 모셨다. 곧장 다리로 간다!"),),
                  (line("caochun", "호표기는 다리로! 유비군이 빠져나가기 전에 막아라!"),)),
            {"Kind": "lockmove", "Targets": ["zhangfei"], "Value": 1},
        ]},
        {"ID": "tiger-riders", "Every": 3,
         "If": ["turnstart:enemy", "round>=3", "!round>=7"],
         "Do": [{"Kind": "spawn", "Spawns": tiger_wave}]},
        {"ID": "tiger-crossing", "If": ["turnstart:enemy", "round>=5"], "Do": [
            {"Kind": "ai", "Targets": ["tiger"], "Policy": "march", "GoalArea": CAMP},
            {"Kind": "ai", "Targets": ["zhanghe-force", "xuchu-force", "rear"], "Policy": "assault"},
        ]},
        {"ID": "bridge-roar",
         "If": ["at:zhaoyun:70", "alive:zhangfei", "!at:enemy:70"],
         "Do": [
             scene((move("enemy", 13, 22, "SW"),)),
             {"Kind": "spawn", "Spawns": bridge_force},
             scene(
                 (move("bridge-foot", 17, 26, "SW"), move("bridge-horse", 17, 25, "SW")),
                 (play("zhangfei", "attack"), play("bridge-foot", "hit"), play("bridge-horse", "hit")),
                 ({"Kind": "fall", "Unit": "bridge-foot"}, {"Kind": "fall", "Unit": "bridge-horse"}),
                 (line("xiahoudun", "길을 비켜라! 내가 뚫겠다."),),
                 (move("xiahoudun", 17, 26, "SW"),),
                 (play("xiahoudun", "attack"), play("zhangfei", "block")),
                 (play("xiahoudun", "attack"), play("zhangfei", "block")),
                 (play("zhangfei", "attack"), play("xiahoudun", "hit")),
                 (move("xiahoudun", 17, 23, "SW"),),
                 (line("zhangfei", "난 다리만 지킬 테니, 내 걱정 말고 천천히들 가슈!"),),
                 (line("xiahoudun", "제길, 저놈은 지치지도 않는군."),),
                 ({"Kind": "dust", "To": point(18, 29), "Seconds": 1.2},),
                 (line("bridge-spotter", "장군, 다리 뒤에 먼지가 일어납니다. 복병 아닙니까?"),),
                 (line("xiahoudun", "여기서 지체할 수 없다. 길을 돌아 강릉으로 간다!"),),
                 (move("xiahoudun", 13, 21, "NW"), move("bridge-spotter", 14, 21, "NW")),
                 (move("zhangfei", 17, 32, "SW"),),
             ),
             {"Kind": "remove", "Targets": ["enemy"]},
             {"Kind": "lockmove", "Targets": ["zhangfei"]},
         ]},
    ],
    "Faction": "wei",
    "Armies": [{"Faction": "shu", "Name": "유비군", "Ally": True},
               {"Faction": "wei", "Name": "조조군"}],
}

(HERO / "stage26.json").write_text(
    json.dumps(stage, ensure_ascii=False, indent=1) + "\n", encoding="utf-8")
