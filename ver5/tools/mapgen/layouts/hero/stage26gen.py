#!/usr/bin/env python3
"""Chapter 26 연의 (장판파): stage26.json on the shared map maps/map_changban.json (made by the 가상 B25V work).
Run from ver5/. Rewrites stage26.json from scratch every time; the map is not touched here.

The shared map and the crossing axis are defined in southern_battles.py."""
import json
from pathlib import Path

from southern_battles import place_changban

HERO = Path('assets/scenario/hero')


def foe(c, x, y, group, policy='react', **kw):
    e = {'Character': c, 'X': x, 'Y': y, 'Group': group, 'Policy': policy}
    if c.startswith('troop_'):
        e['Shift'] = -0.12
    e.update(kw)
    return e


def spawn(c, x, y, group, policy, **kw):
    return dict(foe(c, x, y, group, policy, **kw), Side='enemy')


def say(actor, text, **kw):
    return {'Kind': 'say', 'Lines': [dict({'Actor': actor, 'Text': text}, **kw)]}


TIGER_GOAL = 74
tiger_wave = [spawn('troop_cavalry', 0, y, 'tiger', 'march', GoalArea=TIGER_GOAL, Shift=-0.19) for y in (10, 11, 12, 13)]

field = [
    # the field is packed: most stand their ground (hold) and strike whoever comes next to them, a few riders react
    foe('wenpin', 12, 4, 'field', 'hold', Rank='officer'),
    foe('troop_cavalry', 11, 3, 'field'), foe('troop_footman', 13, 5, 'field', 'hold'), foe('troop_archer', 12, 2, 'field', 'hold'),
    foe('troop_footman', 10, 6, 'field', 'hold'),
    foe('zhanghe', 17, 10, 'field', 'hold', Rank='officer'),
    foe('troop_footman', 18, 11, 'field', 'hold'), foe('troop_archer', 18, 9, 'field', 'hold'),
    foe('troop_footman', 15, 12, 'field', 'hold'),
    foe('xuchu', 22, 5, 'field', 'hold', Rank='officer'),
    foe('troop_footman', 23, 7, 'field', 'hold'), foe('troop_footman', 21, 8, 'field', 'hold'),
    foe('troop_cavalry', 16, 2, 'field'), foe('troop_footman', 13, 13, 'field', 'hold'),
]
south = [
    foe('troop_footman', 10, 18, 'south', Shift=-0.12), foe('troop_archer', 17, 18, 'south', Shift=-0.12),
    foe('troop_footman', 23, 18, 'south', Shift=-0.12),
]
tiger = [
    foe('caochun', 21, 12, 'tiger', 'march', Rank='commander', GoalArea=TIGER_GOAL),
    foe('troop_cavalry', 20, 11, 'tiger', 'march', GoalArea=TIGER_GOAL, Shift=-0.19),
    foe('troop_cavalry', 22, 11, 'tiger', 'march', GoalArea=TIGER_GOAL, Shift=-0.19),
    foe('troop_cavalry', 21, 13, 'tiger', 'march', GoalArea=TIGER_GOAL, Shift=-0.19),
]

main_cells = [(30, 6), (29, 5), (29, 6), (29, 7), (30, 5), (30, 7), (30, 4)]
stage = {
    'ID': 'B26', 'Name': '장판파 전투', 'Width': 0, 'Height': 0, 'Max': 4,
    'Required': ['zhangfei', 'zhaoyun'],
    'Victory': [{'If': ['at:zhaoyun:70'], 'Text': '조운이 다리 동쪽 진영에 닿는다'}],
    'Defeat': [{'If': ['defeated:zhaoyun'], 'Text': '조운 퇴각'}, {'If': ['defeated:zhangfei'], 'Text': '장비 퇴각'},
               {'If': ['at:enemy:70'], 'Text': '적이 진영에 선다'}],
    'TurnLimit': 20, 'Map': 'map_changban.json',
    'Allies': [{'X': 3, 'Y': 2}] + [{'X': x, 'Y': y} for x, y in main_cells],
    'Recommend': {'zhaoyun': {'X': 3, 'Y': 2}, 'zhangfei': {'X': 30, 'Y': 6}, 'liubei': {'X': 30, 'Y': 5}},
    'Groups': [{'Name': '조운', 'Area': 72, 'Required': ['zhaoyun']},
               {'Name': '본군', 'Area': 73, 'Required': ['zhangfei']}],
    'Enemies': field + south + tiger,
    'Reward': {'money': 2000, 'mirror_armor': 1},
    'Areas': [
        {'ID': 70, 'Rect': {'X': 31, 'Y': 5, 'W': 3, 'H': 3}, 'Name': '다리 동쪽 진영',
         'Description': '조운이 닿으면 승리, 적이 서면 패배'},
        {'ID': 72, 'Rect': {'X': 3, 'Y': 2, 'W': 1, 'H': 1}, 'Name': '무너진 마을', 'Description': '조운 혼자'},
        {'ID': 73, 'Rect': {'X': 29, 'Y': 4, 'W': 2, 'H': 4}, 'Name': '다리 어귀', 'Description': '본군 일곱'},
        {'ID': TIGER_GOAL, 'Rect': {'X': 24, 'Y': 5, 'W': 3, 'H': 3}, 'Name': '다리 서쪽 머리',
         'Description': '조순의 호표기가 몰려가는 곳'},
    ],
    'Events': [
        {'ID': 'opening', 'Do': [
            {'Kind': 'buff', 'Targets': ['zhaoyun'], 'Stat': 'attack', 'Value': 20},
            {'Kind': 'buff', 'Targets': ['zhaoyun'], 'Stat': 'defense', 'Value': 20},
        ]},
        {'ID': 'opening-liubei', 'If': ['!round>=2', 'alive:liubei'], 'Do': [
            say('zhangfei', '형, 자룡이 안 보여유. 조조한테 붙었다는 놈도 있던디…….'),
            say('liubei', '자룡은 나를 버릴 사람이 아니네. 다리를 지키게. 올 걸세.'),
        ]},
        {'ID': 'opening-pursuit', 'Do': [
            say('zhaoyun', '다리까지 들판 하나다. 뚫든 돌든, 이 아이는 건넨다.'),
            say('caochun', '호표기는 다리로! 유비의 목이 저 너머에 있다!')]},
        {'ID': 'tiger-riders', 'Every': 3, 'If': ['turnstart:enemy', 'round>=3', '!round>=7'], 'Do': [
            {'Kind': 'spawn', 'Spawns': tiger_wave},
            say('caochun', '호표기가 더 왔다! 다리 머리를 메워라!')]},
        {'ID': 'bridge-roar', 'If': ['at:zhaoyun:70'], 'Do': [
            {'Kind': 'cutscene', 'Steps': [
                [{'Kind': 'move', 'Unit': 'zhangfei', 'To': {'X': 28, 'Y': 6}, 'Facing': 'NW'}],
                [{'Kind': 'say', 'Unit': 'zhangfei', 'Text': '연인 장익덕이 여기 있다! 죽고 싶은 놈은 나와라!'}]]},
            {'Kind': 'spawn', 'Spawns': [spawn('xiahoujie', 25, 6, 'jie', 'hold', Facing='SE')]},
            {'Kind': 'cutscene', 'Steps': [
                [{'Kind': 'play', 'Unit': 'xiahoujie', 'Motion': 'hit'}],
                [{'Kind': 'play', 'Unit': 'xiahoujie', 'Motion': 'exhausted'}]]},
            {'Kind': 'remove', 'Targets': ['jie']},
            say('caochun', '……숲에 먼지가 인다. 매복이다. 멈춰라!')]},
    ],
    'Faction': 'wei',
    'Armies': [{'Faction': 'shu', 'Name': '유비군', 'Ally': True}, {'Faction': 'wei', 'Name': '조조군'}],
}
(HERO / 'stage26.json').write_text(json.dumps(place_changban(stage), ensure_ascii=False, indent=1) + '\n', encoding='utf-8')
