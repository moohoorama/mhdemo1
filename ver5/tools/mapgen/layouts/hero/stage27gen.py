#!/usr/bin/env python3
"""Chapter 27 연의 (화용도): map27 layout, maps/map27.json and stage27.json. Run from ver5/.
Rewrites its outputs from scratch every time."""
import json
import subprocess
from pathlib import Path

HERO = Path('assets/scenario/hero')
W, H = 28, 26

# ---------------------------------------------------------------- map
g = [['g--'] * W for _ in range(H)]


def put(x, y, c):
    if 0 <= x < W and 0 <= y < H:
        g[y][x] = c


# NW: hills and mountains close in on the way out to 화용도, a pass three cells wide along the diagonal
for y in range(H):
    for x in range(W):
        if x + y <= 17:
            put(x, y, 'n--' if abs(x - y) > 2 else ('u--' if abs(x - y) == 2 else 'g--'))
for i in range(0, 10):
    put(i, i, 'l--')
    put(i + 1, i, 'l--')
for x, y in ((3, 0), (0, 3), (5, 2), (2, 5), (7, 4), (4, 7)):
    put(x, y, 'gt-')
# the reed marsh in the middle: shallows with reeds, mud with standing water
for y in range(6, 23):
    for x in range(6, 24):
        if x + y > 17 and x + y < 40:
            put(x, y, 's--' if (x * 7 + y * 3) % 5 else 'st-')
for x, y in ((10, 12), (11, 12), (12, 15), (13, 15), (15, 10), (16, 10), (9, 16), (14, 18), (18, 13),
             (19, 14), (17, 18), (12, 9), (20, 17), (8, 13)):
    put(x, y, 'd-w')
# a causeway of firm ground wanders through the marsh from 오림 to the pass
cause = [(21, 20), (20, 20), (20, 19), (19, 19), (19, 18), (18, 18), (18, 17), (17, 17), (17, 16), (16, 16),
         (16, 15), (15, 15), (15, 14), (14, 14), (14, 13), (13, 13), (13, 12), (12, 12), (12, 11), (11, 11),
         (11, 10), (10, 10)]
for x, y in cause:
    put(x, y, 'l--')
# SE: the river and the burning 오림 stockade on its bank
for y in range(H):
    for x in range(W):
        if x + y >= 44:
            put(x, y, 'w--')
for x, y in ((24, 19), (25, 18), (26, 17), (23, 20), (22, 21), (27, 16), (21, 22), (20, 23)):
    put(x, y, 'gkF')
for x, y in ((23, 18), (24, 17), (22, 19), (25, 15), (19, 22), (18, 24), (26, 14)):
    put(x, y, 'g-e')
for x, y in ((24, 9), (26, 11), (22, 6), (8, 23), (5, 21), (3, 19), (2, 22), (25, 5), (20, 3)):
    put(x, y, 'gt-')

layout = Path('tools/mapgen/layouts/hero/map27.txt')
layout.write_text(
    '# map27 화용도 전투 (28x26, night). North = row 0. The way out to 화용도 is the NW corner: a pass three\n'
    '# cells wide along the diagonal between hills and mountains. The middle is reed marsh (shallows, reeds,\n'
    '# mud with standing water) with a causeway of firm ground from 오림 (SE) to the pass. SE corner: the river\n'
    '# and the burning stockade on its bank.\n'
    + '\n'.join(' '.join(r) for r in g) + '\n', encoding='utf-8')
subprocess.run(['python3', 'tools/mapgen/mapgen.py', str(layout), str(HERO / 'maps/map27.json')], check=True)


# ---------------------------------------------------------------- stage
def foe(c, x, y, group, hp=60, **kw):
    e = {'Character': c, 'X': x, 'Y': y, 'Policy': 'flee', 'GoalArea': 30, 'HPPct': hp}
    if group:
        e['Group'] = group
    if c.startswith('troop_'):
        e['Shift'] = -0.07
    e.update(kw)
    return e


def say(actor, text, **kw):
    return {'Kind': 'say', 'Lines': [dict({'Actor': actor, 'Text': text}, **kw)]}


enemies = [
    foe('caocao', 15, 13, None, Rank='boss'),
    foe('xuchu', 14, 12, None, Rank='officer'),
    foe('zhangliao', 16, 12, None, Rank='officer'),
    foe('xuhuang', 17, 15, None, Rank='officer'),
]
for x, y in ((13, 11), (14, 11), (12, 13), (13, 14), (16, 14), (15, 16), (17, 13), (14, 15), (18, 14), (16, 17)):
    enemies.append(foe('troop_footman', x, y, 'foot'))
for x, y in ((18, 16), (19, 16), (17, 16), (19, 15)):
    enemies.append(foe('troop_cavalry', x, y, 'horse'))

ambush_cells = [(6, 6), (7, 6), (6, 7)]
main_cells = [(19, 20), (20, 21), (18, 20), (19, 21), (18, 21), (20, 22)]
stage = {
    'ID': 'B27', 'Name': '화용도 전투', 'Width': 0, 'Height': 0, 'Max': 9,
    'Required': ['liubei', 'zhangfei'],
    'Victory': [{'If': ['defeated:caocao'], 'Text': '조조를 물리치거나 화용도로 쫓아낸다'}],
    'Defeat': [{'If': ['defeated:zhangfei'], 'Text': '장비 퇴각'}],
    'Sub': [{'ID': 'caocao', 'Text': '조조가 빠져나가기 전에 물리친다', 'If': ['defeated:caocao', '!fled:caocao'],
             'Flag': 'flag-b27-caocao', 'Reward': {'mengde_book': 1, 'yitian_sword': 1}}],
    'TurnLimit': 0, 'Map': 'map27.json', 'Night': True,
    'Lights': [{'X': 23, 'Y': 18}, {'X': 22, 'Y': 19}, {'X': 19, 'Y': 22}, {'X': 6, 'Y': 6}, {'X': 1, 'Y': 1}],
    'Allies': [{'X': x, 'Y': y} for x, y in main_cells + ambush_cells],
    'Recommend': {'liubei': {'X': 19, 'Y': 20}, 'zhangfei': {'X': 6, 'Y': 6}},
    'Groups': [{'Name': '본군', 'Area': 1, 'Required': ['liubei']},
               {'Name': '복병', 'Area': 2, 'Required': ['zhangfei']}],
    'Enemies': enemies,
    'Reward': {'money': 2500},
    'Duels': [
        {'ID': 'huarong-zhangfei-xuchu', 'Ally': 'zhangfei', 'Enemy': 'xuchu', 'Outcome': 'draw', 'AllyResult': 'stay',
         'EnemyResult': 'stay', 'Text': '장비와 허저가 진흙 속에서 다섯 합을 겨뤘으나 말이 빠져 갈라섰다.',
         'Beats': [
             {'Kind': 'talk', 'Title': '허저가 길을 막다', 'Speech': '승상께는 한 발짝도 못 간다!', 'Side': 1, 'Duration': 2.4},
             {'Kind': 'talk', 'Title': '장비가 받다', 'Speech': '신야에서 내뺀 뚱보구먼! 오늘은 맨땅이 아니어도 봐줄 거 없슈!',
              'Side': 0, 'Duration': 3.0},
             {'Kind': 'approach', 'Title': '두 장수가 진흙을 차며 맞붙다', 'Speech': '', 'Side': 0, 'Duration': 1.2},
             {'Kind': 'block', 'Title': '허저가 사모를 받다', 'Speech': '', 'Side': 0, 'Duration': 1.05},
             {'Kind': 'block', 'Title': '허저가 칼을 내리치다', 'Speech': '', 'Side': 1, 'Duration': 1.05},
             {'Kind': 'rush-block', 'Title': '장비가 몰아치다', 'Speech': '으라앗!', 'Side': 0, 'Duration': 1.1},
             {'Kind': 'block', 'Title': '허저가 버티다', 'Speech': '', 'Side': 1, 'Duration': 1.05},
             {'Kind': 'talk', 'Title': '말이 진흙에 빠지다', 'Speech': '에잇, 말굽이 안 빠지는구먼!', 'Side': 0,
              'Duration': 2.4},
             {'Kind': 'talk', 'Title': '허저가 물러서다', 'Speech': '……승부는 다음이다!', 'Side': 1, 'Duration': 2.2},
             {'Kind': 'backstep', 'Title': '두 장수가 갈라서다', 'Speech': '', 'Side': 0, 'Duration': 1.2},
             {'Kind': 'draw', 'Title': '무승부', 'Speech': '', 'Side': 0, 'Duration': 1.0}]},
        {'ID': 'huarong-zhaoyun-xuhuang', 'Ally': 'zhaoyun', 'Enemy': 'xuhuang', 'Outcome': 'victory',
         'AllyResult': 'stay', 'EnemyResult': 'retreat', 'Text': '서황이 조운과 세 합을 겨룬 뒤 조조를 지키러 물러났다.',
         'Beats': [
             {'Kind': 'talk', 'Title': '서황이 막아서다', 'Speech': '하동의 서공명이다. 여기서부터는 못 지나간다!', 'Side': 1,
              'Duration': 2.6},
             {'Kind': 'talk', 'Title': '조운이 답하다', 'Speech': '상산의 조자룡이오. 길을 비키시오!', 'Side': 0, 'Duration': 2.4},
             {'Kind': 'approach', 'Title': '맞붙다', 'Speech': '', 'Side': 0, 'Duration': 1.2},
             {'Kind': 'block', 'Title': '서황이 창을 받다', 'Speech': '', 'Side': 0, 'Duration': 1.05},
             {'Kind': 'block', 'Title': '서황이 도끼를 휘두르다', 'Speech': '', 'Side': 1, 'Duration': 1.05},
             {'Kind': 'yielding-block', 'Title': '서황이 밀리다', 'Speech': '', 'Side': 0, 'Duration': 1.2},
             {'Kind': 'talk', 'Title': '서황이 등을 돌리다', 'Speech': '……승상을 모셔라!', 'Side': 1, 'Duration': 2.2},
             {'Kind': 'retreat', 'Title': '서황이 물러나다', 'Speech': '', 'Side': 1, 'Duration': 3.0},
             {'Kind': 'withdrawn', 'Title': '서황이 퇴각하다', 'Speech': '', 'Side': 1, 'Duration': 1.1}]},
    ],
    'Areas': [
        {'ID': 1, 'Rect': {'X': 18, 'Y': 20, 'W': 3, 'H': 3}, 'Name': '오림 쪽 강가', 'Description': '유비의 본군'},
        {'ID': 2, 'Rect': {'X': 6, 'Y': 6, 'W': 2, 'H': 2}, 'Name': '늪 어귀', 'Description': '장비의 복병'},
        {'ID': 30, 'Rect': {'X': 0, 'Y': 0, 'W': 2, 'H': 2}, 'Name': '화용도로 가는 길',
         'Description': '북서쪽 끝. 조조가 닿으면 빠져나간다'},
        {'ID': 31, 'Rect': {'X': 7, 'Y': 7, 'W': 6, 'H': 6}, 'Name': '늪 어귀 앞',
         'Description': '조조가 들어서면 장비가 길을 막고 나선다'},
        {'ID': 32, 'Rect': {'X': 10, 'Y': 14, 'W': 14, 'H': 6}, 'Name': '늪 한가운데',
         'Description': '본군이 들어서면 서황과 기병이 돌아선다'},
    ],
    'Events': [
        {'ID': 'opening', 'Do': [
            {'Kind': 'buff', 'Targets': ['foot', 'horse'], 'Stat': 'morale', 'Value': -30},
            say('caocao', '하하하! 주유도 제갈량도 아직 멀었다. 이 길목에 군사 한 무리만 숨겨 두었어도…….'),
            say('liubei', '익덕이 늪 어귀를 막고 있네. 늪을 질러 가세!')]},
        {'ID': 'ambush', 'If': ['any:[at:caocao:31, round>=3]', 'turnstart:enemy'], 'Do': [
            {'Kind': 'cutscene', 'Steps': [[{'Kind': 'face', 'Unit': 'zhangfei', 'Facing': 'SE'}]]},
            say('zhangfei', '연인 장익덕이 여기서 기다렸다! 조조, 어디로 가려는 거여!'),
            say('caocao', '……또 장비냐!'),
            say('xuchu', '승상, 먼저 가십시오. 저놈은 제가 막습니다.'),
            {'Kind': 'ai', 'Targets': ['xuchu', 'zhangliao'], 'Policy': 'assault'}]},
        {'ID': 'rearguard', 'If': ['at:player:32', 'round>=2'], 'Do': [
            {'Kind': 'ai', 'Targets': ['xuhuang', 'horse'], 'Policy': 'assault'},
            say('xuhuang', '뒤가 붙었다! 기병은 나를 따르라. 승상께서 늪을 벗어나실 때까지 막는다!')]},
    ],
    'Faction': 'wei',
    'Armies': [{'Faction': 'shu', 'Name': '유비군', 'Ally': True}, {'Faction': 'wei', 'Name': '조조군'}],
}
(HERO / 'stage27.json').write_text(json.dumps(stage, ensure_ascii=False, indent=1) + '\n', encoding='utf-8')
