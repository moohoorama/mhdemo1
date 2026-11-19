#!/usr/bin/env python3
"""Stage the three gate scenes on the wall top: 양양 (c26-start, c25v-gate) on S_wall_top, 강릉 (c26v-start) on its
left-right mirror S_wall_top_flip. Rewrites those scenes in assets/scenes/ch26.json, ch25v.json and ch26v.json and
leaves every other scene, node and line alone.

  python3 ver5/tools/scene_maps/wall_top_scenes.py

The wall walk is u 0..13, v 0..4 (S_wall_top, wall_top.py) and runs on past both ends; the garrison stands on it facing
out (NE, -v), the officers behind their archers. Outside, upper right, the crowd stands from v -5 back to v -14 with
Liu Bei's party at the front (v -7) facing the wall (SW). Everything is laid out for S_wall_top; for S_wall_top_flip
u and v swap in every cell and every Face turns SE <-> SW, NE <-> NW.
"""
import json
import random
import sys
from pathlib import Path

sys.path.insert(0, str(Path(__file__).parent))
import scene_format  # noqa: E402

SCENES = Path(__file__).resolve().parents[2] / 'assets/scenes'
TURN = {'SE': 'SW', 'SW': 'SE', 'NE': 'NW', 'NW': 'NE'}
CRIES = ['문을 열어 주시오…', '아이가 굶고 있소', '제발…', '살려 주시오']
HUNGER = ['굶은 지 사흘이오', '문을 열어 주시오', '아이가 우오…', '물 한 모금만…']
COMMONERS = ['commoner_man', 'commoner_woman', 'commoner_elder', 'commoner_man', 'commoner_woman', 'commoner_oldwoman']


def a(actor, **k):
    return {'Actor': actor, **k}


def turn(x):
    """The same staging on the mirrored map: cells, faces and every point a step names, u and v swapped."""
    if isinstance(x, dict):
        out = {}
        for k, v in x.items():
            if k in ('At', 'Move', 'Enter', 'CamAt', 'Arrow'):
                out[k] = [v[1], v[0]]
            elif k == 'Via':
                out[k] = [[p[1], p[0]] for p in v]
            elif k in ('Face', 'Look'):
                out[k] = TURN[v]
            else:
                out[k] = turn(v)
        return out
    if isinstance(x, list):
        return [turn(i) for i in x]
    return x


def wall(prefix, spears, officers):
    """The garrison: archers on the battlement rows, spearmen on the inner rows, officers (with named guards) between."""
    cast, rows = {}, {}
    n = 0

    def add(sprite, u, v, face='NE'):
        nonlocal n
        n += 1
        key = f'{prefix}#{n}' if sprite == 'archer' else f'{spears}#{n}'
        cast[key] = {'At': [u, v], 'Face': face, 'Sprite': sprite, 'Faction': 'jing'}
        rows.setdefault(v, []).append(key)

    for u in range(14):
        add('archer', u, 0)
    for u in range(14):
        if not 6 <= u <= 8:
            add('archer', u, 1)
    for u in (0, 1, 2, 3, 11, 12, 13):
        add('archer', u, 2)
    for u in (0, 1, 2, 3, 11, 12, 13):
        add('light_infantry', u, 3)
    for u in range(0, 14, 2):
        add('light_infantry', u, 4)
    for u in [u for u in range(4, 11) if u not in dict(officers).values()][:4]:
        n += 1
        cast[f'{spears}#{n}'] = {'At': [u, 2], 'Face': 'NE', 'Sprite': 'guards', 'Faction': 'jing'}
    for name, u in officers:
        cast[name] = {'At': [u, 2], 'Face': 'NE', 'Hold': True}
    return cast, rows


def field(rnd, party, soldiers, chatter):
    """The crowd outside and Liu Bei's party at its front."""
    cast, keep = {}, set()
    for (name, u, v) in party:
        cast[name] = {'At': [u, v], 'Face': 'SW'}
        keep.add((u, v))
    n = 0
    for v in range(-5, -15, -1):
        for u in range(0, 16):
            if (u, v) in keep or any(abs(u - pu) < 2 and abs(v - pv) < 2 for _, pu, pv in party):
                continue
            if v == -8 and 5 <= u <= 11 or v == -9 and 5 <= u <= 11:
                continue  # the escort's two rows
            if rnd.random() < (.45 if v >= -6 else .62):
                n += 1
                cast[f'백성#{n}'] = {'At': [u, v], 'Face': rnd.choice(['SW'] * 7 + ['SE', 'NW', 'SE']),
                                     'Sprite': rnd.choice(COMMONERS), 'Faction': 'merchant'}
    for i, (u, v) in enumerate([(u, v) for v in (-8, -9) for u in range(5, 12)], 1):
        cast[f'{soldiers}#{i}'] = {'At': [u, v], 'Face': 'SW', 'Sprite': 'light_infantry', 'Faction': 'shu'}
    return cast


def field_behind(rnd, party, soldiers):
    """강릉: the army stands at the front under the wall, four deep, and the townsfolk huddle behind it."""
    cast = {name: {'At': [u, v], 'Face': 'SW'} for name, u, v in party}
    n = 0
    for v in (-5, -6, -7, -8):
        for u in range(2, 15):
            if any(abs(u - pu) < 2 and abs(v - pv) < 2 for _, pu, pv in party):
                continue
            n += 1
            cast[f'{soldiers}#{n}'] = {'At': [u, v], 'Face': 'SW', 'Sprite': 'light_infantry', 'Faction': 'shu'}
    n = 0
    for v in range(-11, -17, -1):
        for u in range(0, 16):
            if rnd.random() < .72:
                n += 1
                cast[f'백성#{n}'] = {'At': [u, v], 'Face': rnd.choice(['SW'] * 5 + ['SE', 'NW', 'NE']),
                                     'Sprite': rnd.choice(COMMONERS), 'Faction': 'merchant'}
    return cast


def frontline(cast, depth=2.5):
    return [k for k, x in cast.items() if k.startswith('백성#') and x['At'][1] >= -5 - depth]


def volley(cast, rows, prefix, rnd):
    """Two waves of arrows from the battlement rows onto the ground in front of the crowd."""
    waves = []
    for v, after in ((0, 0), (1, .9)):
        keys = [k for k in rows[v] if k.startswith(prefix)]
        wave = []
        for i, k in enumerate(keys):
            tu, tv = round(rnd.uniform(1.5, 14), 1), round(rnd.uniform(-6.4, -5.4), 1)
            wave.append(a(k, Play='attack', Arrow=[tu, tv], After=round(after + rnd.uniform(0, .45), 2)))
        waves.append(wave)
    return [waves[0] + waves[1]]


def xiangyang(start, nodes, party_extra, lines):
    rnd = random.Random(26)
    cast, rows = wall('형주궁병', '형주병', [('채모', 7), ('채중', 5), ('괴월', 9)])
    party = [('유비', 8, -7), ('장비', 9, -6), ('간옹', 7, -6)] + party_extra
    crowd = field(rnd, party, '유비군', None)
    cast.update(crowd)
    front = sorted(frontline(cast), key=lambda k: cast[k]['At'][0])
    run = [k for i, k in enumerate(front) if i % 2 == 0]
    scatter = [a(k, Move=[17 + i % 5, -5 - i // 5], Speed=2, After=round(rnd.uniform(.2, 1.0), 2))
               for i, k in enumerate(run)]
    scatter += [a(k, Emote='!') for k in front[1::4]]
    chat = [k for k in cast if k.startswith('백성#')][::19][:4]
    amb = [{'CamAt': [7, -3]}] + [a(k, Chatter=c) for k, c in zip(chat, CRIES)]
    beats = {
        lines[1]: [[[a('채모', Face='NE', Play='action')]],
                   [volley(cast, rows, '형주궁병', rnd)[0],
                    [a('유비', Play='action')] + scatter]],
        lines[4]: [[[a('간옹', Face='SE')]]],
        lines[5]: [[[a('유비', Face='SE')]]],
    }
    return cast, amb, beats


def build():
    ch26 = json.load(open(SCENES / 'ch26.json'))['Scenes']
    ch25v = json.load(open(SCENES / 'ch25v.json'))['Scenes']
    ch26v = json.load(open(SCENES / 'ch26v.json'))['Scenes']

    for scenes, sid, extra, pre in ((ch26, 'c26-start', [], 'c26-a0'), (ch25v, 'c25v-gate', [('서서', 8, -5)], 'c25v-g0')):
        cast, amb, beats = xiangyang(sid, scenes[sid]['Nodes'], extra, [f'{pre}{i}' for i in range(1, 7)])
        if sid == 'c25v-gate':
            beats[f'{pre}3'] = [[[a('장비', Emote='!')]]]
        s = scenes[sid]
        s.update({'Map': 'S_wall_top', 'Ambient': amb, 'Cast': cast, 'Beats': beats})

    # 강릉: the same wall top mirrored; 장윤 and 곽준 on the wall, the archers follow 장윤 out along it
    rnd = random.Random(27)
    cast, rows = wall('강릉궁병', '강릉병', [('장윤', 6), ('곽준', 8)])
    party = [('유비', 8, -7), ('장비', 9, -6), ('간옹', 7, -6), ('서서', 8, -5)]
    cast.update(field_behind(rnd, party, '유비군'))
    follow = [k for k in rows[1] if cast[k]['At'][0] >= 9] + [k for k in rows[0] if cast[k]['At'][0] >= 11]
    chat = [k for k in cast if k.startswith('백성#')][::19][:4]
    amb = [{'CamAt': [7, -3]}] + [a(k, Chatter=c) for k, c in zip(chat, HUNGER)]
    beats = {
        'c26v-a03': [[[a('곽준', Face='NW')]]],
        'c26v-a05': [[[a('장윤', Face='SE')]]],
        'c26v-a06': [[[a('장윤', Emote='…')]]],
        'c26v-a07': [[[a(k, Emote='!') for k in follow[:4]]]],
        'c26v-a08': [[], [[a('장윤', Move=[16, 2], Exit=True)] +
                          [a(k, Move=[16, cast[k]['At'][1]], Exit=True, After=round(.3 + .08 * i, 2)) for i, k in enumerate(follow)]]],
        'c26v-a09': [[[a('곽준', Face='NE')]]],
        'c26v-a10': [[[a('유비', Play='action')]]],
    }
    s = ch26v['c26v-start']
    s.update({'Map': 'S_wall_top_flip', 'Ambient': turn(amb), 'Cast': turn(cast), 'Beats': turn(beats)})

    for name, scenes in (('ch26', ch26), ('ch25v', ch25v), ('ch26v', ch26v)):
        scene_format.write(SCENES / f'{name}.json', scenes)


if __name__ == '__main__':
    build()
