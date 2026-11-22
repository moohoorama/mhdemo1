#!/usr/bin/env python3
"""Apply the Changban pursuit and evacuation to the authored alternate stage."""

import json

from southern_battles import HERO


def spawn(character, cell, group, *, policy='march', goal=55, rank='troop'):
    unit = dict(Character=character, X=cell[0], Y=cell[1], Group=group,
                Policy=policy, Rank=rank,
                Shift=-0.12 if rank == 'troop' else -0.07, Facing='SW')
    if policy == 'march':
        unit['GoalArea'] = goal
    return unit


def formation(members, cells, group, *, goal):
    return [spawn(name, cell, group, goal=goal, rank=rank)
            for (name, rank), cell in zip(members, cells, strict=True)]


def wave(name, number, units):
    return dict(ID=name, If=[f'round>={number}', 'turnstart:enemy'],
                Do=[dict(Kind='spawn', Spawns=[dict(unit, Side='enemy') for unit in units])])


def place(stage):
    rider = ('troop_cavalry', 'troop')
    horse_archer = ('troop_horse_archer', 'troop')
    foot = ('troop_footman', 'troop')
    bow = ('troop_archer', 'troop')
    van = formation(
        [('caochun','commander'), ('xiahouyuan','officer')]
        + [rider, horse_archer] * 8 + [rider],
        [(12,9),(21,8)] + [(x,y) for y in (5,7) for x in range(6,22,2)] + [(22,6)],
        'vanguard', goal=56,
    )
    van.append(spawn('xiahouen', (16,11), 'duelist', policy='react', rank='officer'))
    second = formation(
        [('xiahoudun','commander'), ('caoren','officer'), ('caohong','officer')]
        + [foot, bow, rider] * 5 + [foot, bow],
        [(8,4),(14,4),(21,4)] + [(x,3) for x in range(5,22)],
        'second-line', goal=55,
    )
    commanders = [
        ('caocao','boss'), ('xuhuang','officer'), ('xuchu','officer'),
        ('zhanghe','officer'), ('yuejin','officer'), ('yujin','officer'),
        ('zhangliao','officer'), ('chengyu','officer'), ('jiaxu','officer'),
        ('xunyou','officer'), ('liuye','officer'), ('hanhao','officer'),
        ('caoxiu','officer'), ('lidian','officer'), ('zhuling','officer'),
        ('wenpin_foot','officer'),
    ]
    main = formation(
        commanders + [foot,bow] * 10 + [rider] * 4,
        [(x,2) for x in range(5,21)] + [(x,y) for y in (0,1) for x in range(1,25,2)],
        'main-army', goal=54,
    )
    stage['Enemies'] = van
    for refugee, (x,y) in zip(stage['Party'][:6],
                               ((16,20),(17,20),(18,20),(16,21),(17,21),(18,21)),
                               strict=True):
        refugee.update(X=x, Y=y)
    conditions = [f'fled:ref{i}' for i in range(1,7)] + ['at:liubei:57']
    stage['Victory'] = [{'If': conditions, 'Text': '피난민 여섯 무리와 유비가 철수한다'}]
    losses = [d for d in stage['Defeat'] if 'defeated:zhangfei' not in ''.join(d['If'])]
    stage['Defeat'] = losses
    stage['Areas'] = [a for a in stage['Areas'] if a['ID'] != 52]
    for number, name, rectangle in (
        (54,'조조 본군 집결지',(4,8,19,3)),
        (55,'중군 집결지',(5,11,19,3)),
        (56,'기병 선봉 진격로',(10,15,12,4)),
        (57,'철수 지점',(16,31,5,3)),
    ):
        areas = [a for a in stage['Areas'] if a['ID'] == number]
        area = dict(ID=number, Name=name, Rect=dict(zip(('X','Y','W','H'),rectangle)))
        if number == 57:
            area['Description'] = '피난민을 모두 보낸 뒤 유비가 여기에 닿으면 승리'
        if areas:
            areas[0].update(area)
        else:
            stage['Areas'].append(area)

    opening = next(event for event in stage['Events'] if event['ID'] == 'opening')
    opening['Do'] = [action for action in opening['Do']
                     if not any('여울' in line['Text'] for line in action.get('Lines',[]))]
    cut = next(event for event in stage['Events'] if event['ID'] == 'bridge-cut')
    cut['If'] = conditions
    stage['Events'] = [opening, wave('second-line-arrives',3,second),
                       wave('main-army-arrives',5,main), cut]
    return stage


if __name__ == '__main__':
    path = HERO / 'stage25v.json'
    stage = place(json.loads(path.read_text(encoding='utf-8')))
    path.write_text(json.dumps(stage, ensure_ascii=False, indent=1) + '\n', encoding='utf-8')
