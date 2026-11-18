#!/usr/bin/env python3
"""Chapter 25 연의 (불타는 신야): map25 layout, maps/map25.json and stage25.json. Run from ver5/.
Rewrites its outputs from scratch every time."""
import json
import subprocess
from pathlib import Path

HERO = Path('assets/scenario/hero')
W, H = 26, 24

# ---------------------------------------------------------------- map
g = [['g--'] * W for _ in range(H)]


def put(x, y, c):
    if 0 <= x < W and 0 <= y < H:
        g[y][x] = c


# the flooded 백하 runs from the east edge (y 11) down to the south edge (x 13), two cells of water with a
# wide fringe of shallows on both banks
for y in range(9, H):
    cx = 25 - (y - 11)
    for x in range(cx - 4, cx + 4):
        if 0 <= x < W:
            put(x, y, 's--')
for y in range(11, H):
    cx = 25 - (y - 11)
    put(cx, y, 'w--')
    put(cx - 1, y, 'w--')
# 신야성 NE, ringed by burning fences and houses
put(21, 3, 'go-')
for x, y in ((19, 1), (20, 1), (21, 1), (22, 1), (23, 1), (19, 2), (23, 2), (19, 3), (23, 3), (19, 4), (23, 4),
             (19, 5), (20, 5), (22, 5), (23, 5)):
    put(x, y, 'gzF')
for x, y in ((21, 2), (20, 2), (22, 2), (20, 3), (22, 3), (20, 4), (21, 4), (22, 4)):
    put(x, y, 'i--')
put(21, 5, 'i--')
for x, y in ((17, 1), (24, 7), (17, 6), (25, 2)):
    put(x, y, 'gvF')
for x, y in ((16, 3), (18, 7), (24, 0), (25, 5)):
    put(x, y, 'g-e')
# the road to 완 runs from the castle's south gate (21, 5) west and north-west to the map's NW corner
road = [(21, 6), (20, 6), (19, 6), (18, 6)] + [(x, 6) for x in range(17, 9, -1)] + \
       [(9, 6), (9, 5), (8, 5), (7, 5), (7, 4), (6, 4), (5, 4), (5, 3), (4, 3), (3, 3), (3, 2), (2, 2), (2, 1),
        (1, 1), (1, 0), (0, 0)]
for x, y in road:
    put(x, y, 'l--')
# puddles left by the night's rain and the burst dike: slow for horses
for x, y in ((12, 9), (13, 9), (12, 10), (8, 11), (9, 11), (15, 13), (16, 13), (6, 14), (7, 14), (11, 15),
             (10, 8), (14, 7), (4, 9), (5, 9), (3, 12), (9, 17), (10, 17), (5, 18), (14, 11), (17, 10),
             (6, 7), (11, 4), (12, 3), (8, 2)):
    put(x, y, 'g-w')
# trees and a low rise
for x, y in ((2, 7), (1, 10), (0, 13), (3, 16), (1, 20), (6, 22), (9, 21), (13, 1), (11, 0), (4, 5), (7, 0),
             (0, 4), (15, 16), (12, 19)):
    put(x, y, 'gt-')
for x, y in ((4, 15), (5, 15), (5, 16), (17, 15), (18, 15)):
    put(x, y, 'u--')

layout = Path('tools/mapgen/layouts/hero/map25.txt')
layout.write_text(
    '# map25 신야 전투 (26x24, dawn). North = row 0. 신야성 (21, 3) NE ringed by burning fences,\n'
    '# the road to 완 from the south gate (21, 5) west to the NW corner, puddles over the field,\n'
    '# the flooded 백하 SE (water on a diagonal from (25, 11) to (13, 23), shallows on both banks).\n'
    + '\n'.join(' '.join(r) for r in g) + '\n', encoding='utf-8')
subprocess.run(['python3', 'tools/mapgen/mapgen.py', str(layout), str(HERO / 'maps/map25.json')], check=True)


# ---------------------------------------------------------------- stage
def foe(c, x, y, group, **kw):
    e = {'Character': c, 'X': x, 'Y': y, 'Policy': 'flee', 'GoalArea': 20, 'HPPct': 70}
    if group:
        e['Group'] = group
    if c.startswith('troop_'):
        e['Shift'] = -0.07
    e.update(kw)
    return e


def say(actor, text, **kw):
    return {'Kind': 'say', 'Lines': [dict({'Actor': actor, 'Text': text}, **kw)]}


def debuff(targets, turns=None):
    out = []
    for stat in ('attack', 'defense', 'mind', 'agility', 'morale'):
        b = {'Kind': 'buff', 'Targets': targets, 'Stat': stat, 'Value': -30}
        if turns:
            b['Turns'] = turns
        out.append(b)
    return out


enemies = [
    foe('caoren', 16, 7, 'rout', Rank='boss'),
    foe('caohong', 14, 7, 'rout', Rank='officer'),
    foe('xuchu', 17, 8, None, Rank='troop', HPPct=60),
]
for x, y in ((13, 6), (15, 6), (12, 8), (14, 8), (15, 9), (13, 9), (11, 9), (16, 9), (12, 10), (10, 7), (11, 10), (14, 10)):
    enemies.append(foe('troop_footman', x, y, 'rout'))
for x, y in ((18, 8), (18, 10), (17, 10), (19, 9), (16, 11), (19, 7)):
    enemies.append(foe('troop_cavalry', x, y, 'rearguard', Shift=-0.12))

main_cells = [(2, 19), (3, 19), (2, 20), (3, 20), (4, 19), (4, 20)]
guan_cells = [(16, 17), (17, 17), (16, 18), (15, 17)]
stage = {
    'ID': 'B25', 'Name': '신야 전투', 'Width': 0, 'Height': 0, 'Max': 10,
    'Required': ['liubei', 'guanyu', 'mizhu'],
    'Victory': [{'If': ['defeated:caoren'], 'Text': '조인을 물리치거나 북서쪽 끝으로 쫓아낸다'}],
    'Defeat': [{'If': ['defeated:guanyu'], 'Text': '관우 퇴각'}, {'If': ['defeated:mizhu'], 'Text': '미축 퇴각'}],
    'Sub': [{'ID': 'caoren', 'Text': '조인이 북서쪽 끝에 닿기 전에 물리친다', 'If': ['defeated:caoren', '!fled:caoren'],
             'Flag': 'flag-b25-caoren', 'Reward': {'golden_breastplate': 1}}],
    'TurnLimit': 0, 'Map': 'map25.json',
    'Allies': [{'X': x, 'Y': y} for x, y in main_cells + guan_cells],
    'Recommend': {'liubei': {'X': 2, 'Y': 19}, 'guanyu': {'X': 16, 'Y': 17}, 'mizhu': {'X': 15, 'Y': 17}},
    'Groups': [{'Name': '본대', 'Area': 1, 'Required': ['liubei']},
               {'Name': '둑의 군사', 'Area': 2, 'Required': ['guanyu', 'mizhu']}],
    'Enemies': enemies,
    'Reward': {'money': 2000},
    'Duels': [{
        'ID': 'xinye-zhangfei-xuchu', 'Ally': 'zhangfei', 'Enemy': 'xuchu', 'Outcome': 'victory', 'AllyResult': 'stay',
        'EnemyResult': 'retreat', 'Text': '허저가 장비와 세 합을 겨루다 불과 물에 지친 군사를 끌고 물러났다.',
        'Beats': [
            {'Kind': 'talk', 'Title': '장비가 허저를 막아서다', 'Speech': '거기 뚱보! 박망에서 못 본 얼굴이구먼. 한판 붙자!',
             'Side': 0, 'Duration': 2.8},
            {'Kind': 'talk', 'Title': '허저가 받다', 'Speech': '장익덕이냐. 뒤를 막는 김에 네 목도 가져가마!', 'Side': 1,
             'Duration': 2.6},
            {'Kind': 'approach', 'Title': '두 장수가 물웅덩이를 차며 맞붙다', 'Speech': '', 'Side': 0, 'Duration': 1.2},
            {'Kind': 'block', 'Title': '허저가 장비의 사모를 받다', 'Speech': '', 'Side': 0, 'Duration': 1.05},
            {'Kind': 'block', 'Title': '허저가 칼을 내리치다', 'Speech': '무겁구먼!', 'Side': 1, 'Duration': 1.05},
            {'Kind': 'yielding-block', 'Title': '허저가 장비의 사모에 밀리다', 'Speech': '', 'Side': 0, 'Duration': 1.2},
            {'Kind': 'talk', 'Title': '허저가 물러서다', 'Speech': '……오늘은 불과 물이 네 편이다. 다음엔 맨땅에서 보자!',
             'Side': 1, 'Duration': 3.0},
            {'Kind': 'retreat', 'Title': '허저가 북서쪽으로 물러나다', 'Speech': '', 'Side': 1, 'Duration': 3.0},
            {'Kind': 'withdrawn', 'Title': '허저가 퇴각하다', 'Speech': '', 'Side': 1, 'Duration': 1.1}]}],
    'Areas': [
        {'ID': 1, 'Rect': {'X': 2, 'Y': 19, 'W': 3, 'H': 2}, 'Name': '본대 자리', 'Description': '남서쪽. 유비와 다섯'},
        {'ID': 2, 'Rect': {'X': 15, 'Y': 17, 'W': 3, 'H': 2}, 'Name': '여울 가', 'Description': '남동쪽 백하 여울 가. 관우·미축과 둘'},
        {'ID': 20, 'Rect': {'X': 0, 'Y': 0, 'W': 4, 'H': 2}, 'Name': '완으로 가는 길',
         'Description': '북서쪽 끝. 조인군이 닿으면 빠져나간다'},
        {'ID': 21, 'Rect': {'X': 0, 'Y': 8, 'W': 26, 'H': 8}, 'Name': '들판 가운데',
         'Description': '아군이 들어서면 허저와 기병이 돌아서서 막는다'},
    ],
    'Events': [
        {'ID': 'routed', 'Do': debuff(['rout', 'rearguard']) + debuff(['xuchu'], 1) + [
            {'Kind': 'buff', 'Targets': ['rout'], 'Stat': 'speed', 'Value': -2},
            say('caoren', '불에 물까지……. 완으로 물러난다. 대오는 버려라, 살아서 가는 것이 먼저다!'),
            say('jianyong', '현덕, 그냥 둬도 달아날 군사일세.'),
            say('liubei', '달아나게 두면 열흘 뒤에 다시 오네. 조인만은 잡아야 하네.'),
            say('mizhu', '운장, 여울이 얕은 데는 저쪽입니다. 자루를 쌓으며 보아 두었습니다.'),
            say('guanyu', '앞장서시오, 자중.')]},
        {'ID': 'rearguard-turns', 'If': ['at:player:21'], 'Do': [
            {'Kind': 'ai', 'Targets': ['xuchu', 'rearguard'], 'Policy': 'react'},
            say('xuchu', '자효는 먼저 가시오! 뒤는 이 허저가 막소!')]},
        {'ID': 'caoren-gone', 'If': ['fled:caoren'], 'Do': [
            say('caoren', '신야에서 두 번이나……. 유비, 이 빚은 잊지 않는다!', Offstage=True)]},
    ],
    'Faction': 'wei',
    'Armies': [{'Faction': 'shu', 'Name': '유비군', 'Ally': True}, {'Faction': 'wei', 'Name': '조조군'}],
}
(HERO / 'stage25.json').write_text(json.dumps(stage, ensure_ascii=False, indent=1) + '\n', encoding='utf-8')
