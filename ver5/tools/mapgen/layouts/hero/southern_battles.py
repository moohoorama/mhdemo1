#!/usr/bin/env python3
"""Build the Changban and Red Cliffs battle terrain. Run from ver5."""

import json
import math
import sys
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parents[2]))
from mapgen import make

HERO = Path('assets/scenario/hero')
LAYOUTS = Path(__file__).resolve().parent


def write_map(name, rows, description):
    (LAYOUTS / f'{name}.txt').write_text(f'# {description}\n' + '\n'.join(
        ' '.join(row) for row in rows) + '\n')
    (HERO / 'maps' / f'{name}.json').write_text(json.dumps(
        make(rows), ensure_ascii=False, indent=1) + '\n')


def changban_rows():
    width, depth = 26, 36
    rows = []
    for y in range(depth):
        row = []
        for x in range(width):
            middle = 26 + round(2 * math.sin((x - 17) / 5))
            if middle - 1 <= y <= middle + 2:
                cell = 'w--'
            elif 14 <= x <= 20 and 11 <= y <= 33 or 3 <= x <= 7 and 13 <= y <= 26:
                cell = 'l--'
            elif y < 8 and x >= 15:
                cell = 'dv-' if (x, y) in ((19, 2), (20, 5), (23, 4)) else 'd--'
            elif y in range(29, 34) and 13 <= x <= 23:
                cell = 'd--'
            elif x < 3 and 9 <= y <= 26 or x > 22 and 13 <= y <= 24:
                cell = 'u--'
            elif (x * 11 + y * 7) % 19 < 4:
                cell = 'gt-'
            else:
                cell = 'g--'
            row.append(cell)
        rows.append(row)
    for y in range(25, 29):
        rows[y][17] = 'wb-'
    for x, y, tile in ((23, 14, 'd-r'), (24, 15, 'd--'), (25, 15, 'dv-'),
                       (24, 16, 'l--'), (23, 17, 'l--'), (22, 18, 'l--'),
                       (17, 31, 'dk-'), (15, 32, 'dk-'), (21, 32, 'dk-'),
                       (20, 3, 'd-r'), (21, 7, 'd-r'), (23, 6, 'd-s'),
                       (18, 22, 'g--'), (16, 23, 'l--')):
        rows[y][x] = tile
    return rows


def redcliff_rows():
    width, depth = 40, 26
    rows = []
    for y in range(depth):
        row = []
        west_bank = 8 + y // 6 + (y % 7 == 6)
        east_bank = 27 + (y % 9 in (0, 1, 8))
        south_bank = 18 + (y % 4 == 3)
        for x in range(width):
            on_mainland = west_bank <= x <= east_bank and y <= south_bank
            on_jiangxia = east_bank + 4 <= x <= 38 and 4 <= y <= 18 - (x > 36)
            if not (on_mainland or on_jiangxia):
                cell = 'w--'
            elif y <= 2 and (x < 14 or x > 22):
                cell = 'u--'
            elif on_mainland and ((17 <= x <= 19 and y <= 16) or (13 <= x <= 24 and 11 <= y <= 13)):
                cell = 'l--'
            elif y >= 15 or 33 <= x <= 36 or (x * 11 + y * 7) % 19 < 4:
                cell = 'd--'
            else:
                cell = 'g--'
            row.append(cell)
        rows.append(row)
    for x, y in ((12, 11), (14, 15), (16, 16), (21, 14), (24, 12), (24, 16)):
        rows[y][x] = 'dk-'
    for x, y in ((33, 9), (36, 8), (37, 11)):
        rows[y][x] = 'go-'
    return rows


def place_changban(stage):
    """Place the romance stage on the shared bridge axis."""
    facings = {'SE': 'SW', 'SW': 'NW', 'NW': 'NE', 'NE': 'SE'}

    def turn(value):
        if isinstance(value, list):
            for item in value:
                turn(item)
        elif isinstance(value, dict):
            if 'X' in value and 'Y' in value:
                x, y = value['X'], value['Y']
                if 'W' in value:
                    value['X'], value['Y'] = 24 - y - value['H'], x
                    value['W'], value['H'] = value['H'], value['W']
                else:
                    value['X'], value['Y'] = 23 - y, x
            if value.get('Facing') in facings:
                value['Facing'] = facings[value['Facing']]
            for item in value.values():
                turn(item)
    turn(stage)
    areas = {area['ID']: area for area in stage['Areas']}
    areas[74]['Rect'].update(X=13, Y=22, W=3, H=3)
    areas[70]['Name'] = '다리 건너 진영'
    areas[74]['Name'] = '장판교 앞'
    stage['Victory'][0]['Text'] = '조운이 다리 건너 진영에 닿는다'
    for enemy in stage['Enemies']:
        if enemy['Group'] == 'field' and enemy['Policy'] == 'react':
            enemy['Group'] = 'scouts'
    areas[72]['Rect'].update(X=24, Y=15)
    stage['Allies'][0].update(X=24, Y=15)
    stage['Allies'] = [stage['Allies'][0], {'X': 17, 'Y': 30},
                       {'X': 18, 'Y': 30}, {'X': 16, 'Y': 30}]
    stage['Recommend']['liubei'].update(X=18, Y=30)
    areas[73]['Description'] = '장비와 본군 두 명'
    for enemy in stage['Enemies']:
        if (enemy['X'], enemy['Y']) in ((18, 22), (16, 23)):
            enemy['Y'] -= 3
    stage['Recommend']['zhaoyun'].update(X=24, Y=15)
    for event in stage['Events']:
        if event['ID'] == 'bridge-roar':
            spawn = next(action['Spawns'][0] for action in event['Do'] if action['Kind'] == 'spawn')
            spawn.update(X=17, Y=24)
    return stage


def redcliff_stage():
    def point(x, y):
        return dict(X=x, Y=y)

    def area(key, name, x, y, width, height, description):
        return dict(ID=key, Name=name, Rect=dict(X=x, Y=y, W=width, H=height),
                    Description=description)

    def spawn(character, x, y, group, policy, facing, rank='troop', shift=0, **extra):
        return dict(Character=character, X=x, Y=y, Rank=rank, Group=group,
                    Shift=shift, Policy=policy, Facing=facing, **extra)

    def say(*lines):
        return dict(Kind='say', Lines=[dict(Actor=actor, Text=text) for actor, text in lines])

    def focus(x, y):
        return dict(Kind='cutscene', Steps=[[dict(Kind='look', To=point(x, y), Seconds=0.35)]])

    def flame(key):
        return dict(Kind='terrain', Area=key, State='F', Turns=4)

    fire_targets = ['cao-fleet', 'cao-land']
    return {
        'ID': 'B27R', 'Name': '적벽 전투', 'Map': 'map_redcliff.json', 'Boats': True, 'Night': True,
        'Max': 7, 'Required': ['liubei', 'zhugeliang'], 'TurnLimit': 22, 'Faction': 'wei',
        'Allies': [point(x, y) for x, y in ((34, 10), (35, 10), (34, 11), (35, 11), (36, 11),
                                          (35, 12), (36, 12))],
        'Recommend': {'liubei': point(34, 10), 'zhugeliang': point(35, 10)},
        'Enemies': [
            # The routed army's debuff must survive Cao Cao's automatic command buffs.
            spawn('caocao', 22, 16, 'cao-land', 'hold', 'NW', 'boss',
                  Traits=['versatile', 'civil_martial', 'ambition', 'feint']),
            spawn('caoren', 16, 13, 'cao-land', 'hold', 'SW', 'commander'),
            spawn('xuchu', 20, 15, 'cao-land', 'hold', 'SW', 'officer'),
            *[spawn('troop_footman', x, y, 'cao-land', 'hold', 'SW')
              for x, y in ((12, 11), (14, 15), (16, 16), (20, 16), (21, 14), (23, 13), (24, 16))],
            *[spawn('troop_archer', x, y, 'cao-land', 'hold', 'SW')
              for x, y in ((15, 10), (17, 11), (21, 11), (24, 12), (23, 15))],
            *[spawn('troop_cavalry', x, y, 'cao-land', 'hold', 'NE')
              for x, y in ((16, 8), (17, 7), (19, 8), (21, 9))],
            spawn('caimao', 5, 11, 'cao-fleet', 'hold', 'SW', 'officer', -0.15),
            *[spawn('troop_footman', x, y, 'cao-fleet', 'hold', 'SW', 'recruit', -0.08)
              for x, y in ((7, 16), (6, 16), (5, 17), (5, 15), (7, 13), (3, 9), (6, 9))],
            *[spawn('troop_archer', x, y, 'cao-fleet', 'hold', 'SW', 'recruit', -0.08)
              for x, y in ((8, 15), (6, 14), (5, 13), (4, 11), (5, 9), (4, 8))],
            spawn('zhangyun', 30, 10, 'zhang-fleet', 'hold', 'SE', 'officer', -0.07, Faction='jing'),
            *[spawn('troop_footman', x, y, 'zhang-fleet', 'hold', 'SE', shift=-0.07, Faction='jing')
              for x, y in ((30, 8), (30, 12), (29, 11), (29, 13))],
            *[spawn('troop_archer', x, y, 'zhang-fleet', 'hold', 'SE', shift=-0.07, Faction='jing')
              for x, y in ((29, 8), (29, 10), (28, 12))],
        ],
        'Victory': [{'If': ['defeated:caocao'], 'Text': '조조군을 북쪽으로 물리친다'}],
        'Defeat': [{'If': ['defeated:liubei'], 'Text': '유비 퇴각'}],
        'Sub': [{'ID': 'zhangyun', 'Text': '장윤 수군을 물리친다', 'If': ['defeated:zhangyun'],
                 'Flag': 'flag-b27r-zhangyun', 'Reward': {'copper_spear': 1, 'rice': 1}}],
        'Areas': [
            area(81, '북쪽 퇴로', 14, 0, 10, 2, '조조 본대의 퇴로'),
            area(82, '선봉 함선', 5, 15, 4, 3, '황개의 화선이 들이받는 곳'),
            area(83, '서쪽 선단', 3, 8, 5, 7, '바람을 타고 불길이 번지는 조조군 수군'),
            area(84, '강변 진채', 10, 14, 7, 4, '오군이 불을 옮겨 붙이는 육상 진채'),
            area(85, '서북쪽 진채', 10, 9, 6, 4, '강변에서 불길이 번지는 조조군 진채'),
            area(86, '서쪽 물길', 1, 0, 7, 2, '조조 수군의 북쪽 퇴로'),
            area(87, '강하 앞 물길', 28, 6, 5, 13, '장윤이 가로막은 물길'),
            area(88, '강하 북쪽 물길', 29, 0, 4, 2, '장윤 수군이 물러나는 곳'),
        ],
        'Lights': [point(x, y) for x, y in ((5, 11), (8, 15), (22, 16), (23, 13), (33, 10), (36, 8),
                                          (6, 21), (13, 23), (30, 10))],
        'Party': [
            spawn('liuqi', 34, 13, 'jiangxia', 'hold', 'NW', 'officer', AI=True, Faction='liuqi'),
            spawn('huanggai', 7, 21, 'wu-fleet', 'hold', 'NE', 'officer', AI=True, Faction='wu'),
            spawn('zhouyu', 12, 23, 'wu-fleet', 'hold', 'NW', 'commander', AI=True, Faction='wu'),
            spawn('chengpu', 9, 23, 'wu-fleet', 'hold', 'NE', 'officer', AI=True, Faction='wu'),
            *[spawn('troop_footman', x, y, 'wu-fleet', 'hold', 'NE', AI=True, Faction='wu')
              for x, y in ((4, 21), (7, 23), (10, 24), (11, 22), (13, 23), (15, 23))],
            *[spawn('troop_archer', x, y, 'wu-fleet', 'hold', 'NE', AI=True, Faction='wu')
              for x, y in ((5, 22), (8, 22), (10, 21), (11, 24), (13, 22), (14, 24))],
        ],
        'Armies': [{'Faction': 'shu', 'Name': '유비군', 'Ally': True},
                   {'Faction': 'wu', 'Name': '강동 수군', 'Ally': True},
                   {'Faction': 'liuqi', 'Name': '강하군', 'Ally': True},
                   {'Faction': 'wei', 'Name': '조조군'},
                   {'Faction': 'jing', 'Name': '양양 수군'}],
        'Events': [
            {'ID': 'fire-ships', 'Do': [
                focus(7, 19),
                say(('caimao', '황개의 배다. 항복하러 오는군. 길을 열어라.')),
                dict(Kind='cutscene', Steps=[
                    [dict(Kind='move', Unit='huanggai', To=point(7, 18), Facing='NE')],
                    [dict(Kind='say', Unit='huanggai', Text='불을 붙이시오!')],
                ]),
                flame(82),
                dict(Kind='damage', Targets=['cao-fleet@area:82'], Value=35),
                say(('caimao', '불이다! 물을 퍼라! 옆 배에 옮겨붙기 전에 꺼라!')),
                dict(Kind='spawn', Spawns=[
                    spawn('ganning', 4, 18, 'wu-fleet', 'hold', 'NE', 'officer', AI=True, Faction='wu'),
                    spawn('handang', 10, 19, 'wu-fleet', 'hold', 'NE', 'officer', AI=True, Faction='wu'),
                ]),
                say(('ganning', '불 붙은 적선부터 쳐라!'), ('handang', '강가 진채에도 불을 놓으시오!')),
                flame(84),
                focus(5, 11),
                flame(83),
                flame(85),
                say(('caimao', '바람이다! 동남풍이 불길을 밀고 온다!'),
                    ('caoren', '강변의 불이 위쪽 진채까지 옮았습니다!'),
                    ('caocao', '모두 당장 물러나라! 북쪽으로 가라. 목숨부터 보존해라!')),
                dict(Kind='damage', Targets=['cao-fleet'], Value=30),
                dict(Kind='buff', Targets=fire_targets, Stat='confusion', Turns=3),
                *[dict(Kind='buff', Targets=fire_targets, Stat=stat, Value=-20)
                  for stat in ('attack', 'defense', 'mind', 'morale', 'agility')],
                dict(Kind='buff', Targets=fire_targets, Stat='speed', Value=-2),
                dict(Kind='buff', Targets=fire_targets, Stat='range', Value=-1),
                dict(Kind='ai', Targets=['cao-fleet'], Policy='flee', GoalArea=86),
                dict(Kind='ai', Targets=['cao-land'], Policy='flee', GoalArea=81),
                dict(Kind='ai', Targets=['wu-fleet'], Policy='assault'),
                say(('zhouyu', '조조의 배가 흐트러졌소. 수군은 전진하시오!')),
                focus(30, 10),
                say(('zhangyun', '우리 배에는 불이 닿지 않았다! 유비가 건너지 못하게 막아라!'),
                    ('zhugeliang', '장윤의 배만 건너면 적의 북쪽 길에 닿습니다. 서두르십시오.')),
            ]},
            {'ID': 'jiangxia-advances', 'If': ['at:player:87'], 'Do': [
                dict(Kind='ai', Targets=['liuqi'], Policy='assault'),
            ]},
            {'ID': 'jiangxia-crosses', 'If': ['defeated:zhangyun'], 'Do': [
                dict(Kind='ai', Targets=['zhang-fleet'], Policy='flee', GoalArea=88),
                say(('liubei', '장윤의 배가 돌아선다! 물길을 건너 조조군을 쳐라!')),
            ]},
        ],
    }


def main():
    write_map('map_changban', changban_rows(), '장판파 · 굽은 강과 남쪽 둑 (B26, B25V)')
    write_map('map_redcliff', redcliff_rows(), '적벽 · 강동 수군과 강하의 북쪽 퇴로 (B27R)')
    (HERO / 'stage27r.json').write_text(json.dumps(redcliff_stage(), ensure_ascii=False, indent=1) + '\n')


if __name__ == '__main__':
    main()
