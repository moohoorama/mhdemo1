#!/usr/bin/env python3
"""Apply the compact Xiangyang siege layout to the authored stage."""

import json
from itertools import combinations

from southern_battles import HERO


def position(unit, cell, facing):
    return dict(unit, X=cell[0], Y=cell[1], Facing=facing)


def camp_buffs():
    flags = [f'flag:flag-b27v-camp-{camp}' for camp in ('w1', 'w2', 's1', 's2')]
    stats = ('morale', 'agility', 'defense', 'attack', 'mind')
    for losses, count in enumerate((1, 2, 3, 5), start=1):
        # At least one camp in every subset of this size must have fallen.
        remaining = len(flags) - losses + 1
        conditions = [
            group[0] if remaining == 1 else 'any:[' + ', '.join(group) + ']'
            for group in combinations(flags, remaining)
        ]
        yield dict(ID=f'cao-camp-bonus-{losses}', If=conditions, Do=[
            dict(Kind='buff', Targets=['cao-west', 'caoren', 'castle'],
                 Stat=stat, Value=20)
            for stat in stats[:count]
        ])


def place(stage, changban):
    changban_units = changban['Enemies'] + [
        unit for event in changban['Events'] for action in event['Do']
        for unit in action.get('Spawns', [])
    ]
    officers = [unit for unit in changban_units if unit['Rank'] != 'troop'
                and unit['Character'] != 'caoren']
    enemies = stage['Enemies']
    front = {'caochun', 'xuchu', 'xuhuang', 'zhangliao'}
    officers.sort(key=lambda unit: unit['Character'] not in front)
    cells = [(3,8),(3,11),(3,15),(3,18)] + [
        (x,y) for y in (6,8,10,12,14,16,18,20,22) for x in (1,2)
    ]
    western = []
    for officer, cell in zip(officers, cells):
        unit = dict(Character=officer['Character'], Rank=officer['Rank'],
                    Shift=-0.19, Group='cao-west', Facing='SE',
                    Policy='react' if officer['Character'] in front else 'hold',
                    X=cell[0], Y=cell[1])
        western.append(unit)
    assert len(western) == len(officers)
    for index, cell in enumerate(((3,7),(3,10),(3,13),(3,16),(3,19),(3,21))):
        character = 'troop_footman' if index % 2 == 0 else 'troop_archer'
        western.append(dict(Character=character, X=cell[0], Y=cell[1],
                            Group='cao-west', Rank='recruit', Shift=-0.12,
                            Facing='SE', Policy='react' if index < 4 else 'hold'))

    fleet = [u for u in enemies if u['Group'] in ('cai', 'cai-fleet')]
    fleet_cells = [(21,5),(21,4),(21,6),(18,6),(19,5),(20,4),(20,6),
                   (22,4),(22,6),(17,7),(18,7),(19,7),(22,5)]
    fleet = [dict(position(unit, cell, 'SE'), HPPct=68, Shift=-0.15)
             for unit, cell in zip(fleet, fleet_cells, strict=True)]

    caoren = next(u for u in enemies if u['Character'] == 'caoren')
    garrison = [position(caoren, (17,14), 'NE')]
    for character, cell in (('troop_footman',(16,11)), ('troop_footman',(18,11)),
                            ('troop_archer',(16,13)), ('troop_archer',(18,13))):
        garrison.append(dict(Character=character, X=cell[0], Y=cell[1], Rank='recruit',
                             Group='castle', Policy='hold', Facing='NE'))

    party = [u for u in stage['Party'] if u['Faction'] != 'shu']
    boats = [(31,7),(25,4),(25,6),(26,4),(27,5),(28,6),(27,4),(28,5),
             (27,6),(29,4),(29,5),(30,8),(31,8),(32,9),(30,9),(31,10)]
    stage['Party'] = [dict(position(unit, cell, 'NW'),
                           Policy='hold' if unit['Character'] == 'zhouyu' else 'assault')
                      for unit, cell in zip(party, boats, strict=True)]
    camps = ((6,9),(6,17),(14,23),(21,23))
    defensive_camps = [(5,9),(7,9),(5,17),(7,17),(13,23),(15,23),(20,23),(22,23)]
    for i,(x,y) in enumerate(defensive_camps):
        stage['Party'].append(dict(Character='troop_footman' if i % 2 == 0 else 'troop_archer',
                                   X=x,Y=y, AI=True, Rank='troop', Facing='NW' if i < 4 else 'NE',
                                   Group=f'camp-defenders-{i//2+1}', Policy='hold', Faction='shu'))
    stage['Allies'] = [dict(X=x,Y=y) for x,y in camps + (
        (5,10),(7,10),(5,18),(7,18),(14,24),(21,24),
    )]
    stage['Enemies'] = fleet + garrison + western

    areas = {a['ID']:a for a in stage['Areas']}
    for key, cell in ((60,(17,13)),(61,(13,13)),(67,(17,9))):
        areas[key]['Rect'].update(X=cell[0],Y=cell[1])
    areas[62]['Rect'] = dict(X=16,Y=6,W=4,H=3)
    areas[62]['Name'] = '북동쪽 수문 앞'
    areas[62]['Description'] = '채모 수군이 지키는 물길'
    areas[67]['Name'] = '북동쪽 수문'
    areas[67]['Description'] = '성으로 이어지는 열린 문'
    naval_win = ['alldefeated:cai-fleet', 'alldefeated:cai']
    city_win = naval_win + ['any:[at:zhouyu:60, at:navy:60, at:liuqi-fleet:60]']
    stage['Victory'] = [{'If':city_win, 'Text':'채모 수군을 물리치고 양양성을 점령한다'}]
    stage['TurnLimit'] = 22
    stage['Sub'] = [sub for sub in stage['Sub'] if sub['ID'] != 'caimao']
    stage['Reward']['treasure_sword'] = 1
    stage['Defeat'][1] = dict(If=['defeated:zhouyu'], Text='주유 퇴각')
    stage['Events'] = [e for e in stage['Events']
                       if e['ID'] not in ('cao-waves', 'liuqi-calls', 'camps-lost')
                       and not e['ID'].startswith('cao-camp-bonus-')]
    stage['Events'].extend(camp_buffs())
    for event in stage['Events']:
        match event['ID']:
            case 'navy-turns':
                event['If'] = naval_win
                order = next(a for a in event['Do'] if a['Kind'] == 'ai')
                order.update(GoalArea=60, Targets=['zhouyu','navy','liuqi-fleet'])
            case 'caocao-10':
                event['If'] = ['round>=5','turnstart:enemy']
                event['Do'] = [act for act in event['Do'] if act['Kind'] != 'spawn']
            case 'xiangyang-falls':
                event['If'] = city_win
    return stage


if __name__ == '__main__':
    stage = HERO / 'stage27v.json'
    changban = json.loads((HERO / 'stage25v.json').read_text(encoding='utf-8'))
    updated = place(json.loads(stage.read_text(encoding='utf-8')), changban)
    stage.write_text(json.dumps(updated, ensure_ascii=False, indent=1) + '\n', encoding='utf-8')
