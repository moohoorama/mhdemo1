#!/usr/bin/env python3
"""Build the Changban, Red Cliffs and Huarong battle terrain. Run from ver5."""

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
    width, depth = 26, 21
    rows = []
    for y in range(depth):
        row = []
        for x in range(width):
            shore = 5 + round(1.6 * math.cos((x - 4) / 4))
            if y <= shore:
                if 19 <= x <= 23 and y >= 3:
                    cell = 'l--'
                elif y <= 2 or y <= 3 and 7 <= x <= 15:
                    cell = 'h--'
                else:
                    cell = 'd--'
            elif 8 <= x <= 16 and abs(y - (11 - (x - 8) // 5)) <= (1 if 10 <= x <= 14 else 0):
                cell = 'd--' if x in (9, 15) else 's--'
            else:
                cell = 'w--'
            row.append(cell)
        rows.append(row)
    for x in range(19, 24):
        rows[5][x] = 'l--'
    for x, y, tile in ((4, 5, 'dkF'), (7, 6, 'dkF'), (11, 5, 'dkF'), (14, 4, 'dkF'),
                       (17, 4, 'dkF'), (24, 5, 'dkF'), (25, 5, 'd-e'), (9, 4, 'd-e')):
        rows[y][x] = tile
    return rows


def huarong_rows():
    width, depth = 32, 22
    rows = []
    for y in range(depth):
        row = []
        for x in range(width):
            lane = 8 + round(2.5 * math.sin((x - 6) / 6)) + x // 12
            if x >= 29 + (y % 7) // 3:
                cell = 'w--'
            elif x <= 5 and abs(y - 7) > 2:
                cell = 'h--' if abs(y - 7) < 6 and x > 1 else 'n--'
            elif x >= 24:
                cell = 'd--' if 8 <= y <= 17 else 'g--'
            elif abs(y - lane) <= 1:
                cell = 'l--'
            elif 6 <= x <= 10 and 3 <= y <= 11 or 14 <= x <= 21 and 14 <= y <= 16:
                cell = 'g--' if (x * 7 + y * 3) % 5 else 'gt-'
            elif y <= 2 or y >= 18:
                cell = 'u--' if (x + y) % 3 else 'gt-'
            elif abs(y - lane) <= 3 or abs(y - 14) <= 1:
                cell = 's--' if (x * 3 + y * 7) % 5 else 'st-'
            else:
                cell = 'w--' if (x + y) % 9 < 6 else 's--'
            row.append(cell)
        rows.append(row)
    for x in range(6):
        rows[7][x] = 'l--'
        rows[8][x] = 'l--'
    for x, y, tile in ((28, 8, 'dkF'), (28, 14, 'dkF'), (27, 17, 'dkF'),
                       (26, 7, 'd-e'), (25, 15, 'd-e'), (17, 11, 'd-w'), (20, 12, 'd-w')):
        rows[y][x] = tile
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


def place_huarong(stage):
    """Stage both groups and Cao Cao's retreat between the landing and the mountain pass."""
    areas = {area['ID']: area for area in stage['Areas']}
    for key, rect in {
        1:(24, 10, 3, 4), 2:(6, 5, 2, 3), 30:(0, 6, 2, 4),
        31:(8, 5, 5, 6), 32:(15, 7, 9, 7),
    }.items():
        areas[key]['Rect'] = dict(zip(('X','Y','W','H'),rect))
    areas[30]['Description'] = '산목 쪽 끝. 조조가 닿으면 빠져나간다'
    for unit, (x, y) in zip(stage['Allies'], (
        (25,11), (26,12), (24,10), (25,12), (24,12), (26,13), (6,6), (7,6), (6,7),
    ), strict=True):
        unit.update(X=x,Y=y)
    stage['Recommend'] = {'liubei': {'X': 25, 'Y': 11}, 'zhangfei': {'X': 6, 'Y': 6}}
    stage['Lights'] = [dict(X=x,Y=y) for x,y in ((27,12),(25,14),(7,7),(1,7))]
    for unit, (x, y) in zip(stage['Enemies'], (
        (15,10), (14,10), (15,9), (19,11),
        (12,9), (13,9), (13,10), (14,9), (16,10),
        (17,9), (16,9), (17,10), (18,10), (17,11),
        (21,11), (20,11), (18,11), (20,12),
    ), strict=True):
        unit.update(X=x,Y=y,Facing='NW')
    for event in stage['Events']:
        if event['ID'] == 'ambush':
            event['Do'][0]['Steps'][0][0]['Facing'] = 'SE'
    return stage


def redcliff_stage():
    def spawn(character, x, y, rank='troop', group=None, shift=0):
        if group is None:
            group = 'west-flotilla' if x < 14 else 'east-flotilla'
        return dict(Character=character, X=x, Y=y, Rank=rank, Group=group,
                    Shift=shift, HPPct=65 if rank == 'troop' else 85,
                    Policy='hold' if group in ('west-flotilla', 'landing') else 'react',
                    Facing='SW')

    return {
        'ID':'B27R', 'Name':'적벽 전투', 'Map':'map_redcliff.json', 'Boats':True, 'Night':True,
        'Max':8, 'Required':['liubei'], 'TurnLimit':16, 'Faction':'wei',
        'Allies':[dict(X=x,Y=y) for x,y in ((17,15),(18,15),(16,16),(19,16),(15,16),(17,17),(20,15),(19,17))],
        'Enemies':[
            spawn('caimao', 20, 7, 'commander', shift=-0.12), spawn('zhangyun', 7, 9, 'officer', shift=-0.12),
            spawn('troop_archer',5,10,shift=-0.12), spawn('troop_footman',8,12,shift=-0.12),
            spawn('troop_footman',7,13,shift=-0.12), spawn('troop_archer',17,9,shift=-0.12),
            spawn('troop_footman',19,10,shift=-0.12),
            spawn('troop_archer',22,7,shift=-0.12), spawn('troop_footman',23,10,shift=-0.12),
            spawn('troop_footman',23,5,group='landing',shift=-0.12),
        ],
        'Victory':[{'If':['defeated:caimao','at:liubei:81'], 'Text':'채모의 배를 물리고 유비가 오림 나루에 닿는다'}],
        'Defeat':[{'If':['defeated:liubei'],'Text':'유비 퇴각'}],
        'Areas':[{'ID':81,'Rect':{'X':19,'Y':4,'W':3,'H':2},'Name':'오림 나루',
                  'Description':'채모를 물리친 뒤 유비가 닿으면 승리'}],
        'Lights':[{'X':19,'Y':5},{'X':23,'Y':5},{'X':17,'Y':15}],
        'Events': [
            {'ID': 'liuqi-pursuit', 'If': ['defeated:caimao', '!defeated:liuqi'],
             'Do': [
                 {'Kind': 'say', 'Lines': [
                     {'Actor': 'liuqi', 'Text': '채모! 아버님의 부고를 감춘 값을……!'},
                 ]},
                 {'Kind': 'cutscene', 'Steps': [
                     [{'Kind': 'face', 'Unit': 'liuqi', 'Facing': 'NE'}],
                     [{'Kind': 'play', 'Unit': 'liuqi', 'Motion': 'hit'}],
                 ]},
                 {'Kind': 'remove', 'Targets': ['liuqi']},
             ]},
        ],
        'Party':[
            dict(Character='liuqi', X=20, Y=14, AI=True, Policy='hold',
                 Faction='liuqi', Rank='officer', Facing='NE'),
            dict(Character='troop_archer', X=21, Y=15, AI=True, Policy='react',
                 Faction='shu', Rank='troop', Facing='NE'),
        ],
        'Armies':[{'Faction':'shu','Name':'유비군','Ally':True},{'Faction':'wei','Name':'양양 수군'},
                  {'Faction':'liuqi','Name':'강하 수군','Ally':True}],
    }


def main():
    write_map('map_changban', changban_rows(), '장판파 · 굽은 강과 남쪽 둑 (B26, B25V)')
    write_map('map_redcliff', redcliff_rows(), '적벽 · 오림 앞 강과 상륙지 (B27R)')
    write_map('map27', huarong_rows(), '화용도 · 오림에서 산목까지 이어지는 늪길 (B27)')
    (HERO / 'stage27r.json').write_text(json.dumps(redcliff_stage(), ensure_ascii=False, indent=1) + '\n')


if __name__ == '__main__':
    main()
