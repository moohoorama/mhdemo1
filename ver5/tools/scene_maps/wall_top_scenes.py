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
WHISPERS = ['화살 날아오면……', '뛸 준비 해 둬라.', '유비님……', '내가 둘째 안고 뛸 테니……', '짐은 버려라, 목숨이 먼저다.',
            '문이 닫히면 어쩌지……', '서쪽 길로 가면 된다더라.', '아이 손 놓지 말거라.', '군사들 뒤에 붙어 있자.', '신발 끈 단단히 매라.']
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


def clumps(rnd, groups, faces):
    """The townsfolk in loose clumps of 4 to 6 on a grid of slots (three cells wide, two deep, a cell between), behind
    the army: v -11 and further back. Returns the cast and the keys of each clump."""
    slots = [(u, v) for v in (-11, -14, -17) for u in (0, 4, 8, 12)]
    rnd.shuffle(slots)
    cast, clump, n = {}, [], 0
    for su, sv in sorted(slots[:groups], key=lambda t: (-t[1], t[0])):
        cells = [(su + du, sv - dv) for du in range(3) for dv in range(2)]
        rnd.shuffle(cells)
        keys = []
        for u, v in cells[:rnd.randint(4, 6)]:
            n += 1
            keys.append(f'백성#{n}')
            cast[keys[-1]] = {'At': [u, v], 'Face': rnd.choice(faces), 'Sprite': rnd.choice(COMMONERS),
                              'Faction': 'merchant'}
        clump.append(keys)
    return cast, clump


def field_behind(rnd, party, soldiers, groups, faces):
    """Liu Bei's party and army stand at the front under the wall, four deep; the townsfolk are clumped behind."""
    cast = {name: {'At': [u, v], 'Face': 'SW'} for name, u, v in party}
    n = 0
    for v in (-5, -6, -7, -8):
        for u in range(2, 15):
            if any(abs(u - pu) < 2 and abs(v - pv) < 2 for _, pu, pv in party):
                continue
            n += 1
            cast[f'{soldiers}#{n}'] = {'At': [u, v], 'Face': 'SW', 'Sprite': 'light_infantry', 'Faction': 'shu'}
    crowd, clump = clumps(rnd, groups, faces)
    cast.update(crowd)
    return cast, clump


def volley(cast, rows, prefix, rnd):
    """Two waves of arrows from the battlement rows onto the ground in front of the army."""
    waves = []
    for v, after in ((0, 0), (1, .9)):
        keys = [k for k in rows[v] if k.startswith(prefix)]
        wave = []
        for i, k in enumerate(keys):
            tu, tv = round(rnd.uniform(2, 14), 1), round(rnd.uniform(-4.6, -3.8), 1)
            wave.append(a(k, Play='attack', Arrow=[tu, tv], After=round(after + rnd.uniform(0, .45), 2)))
        waves.append(wave)
    return [waves[0] + waves[1]]


def xiangyang(start, nodes, party_extra, lines):
    rnd = random.Random(26)
    cast, rows = wall('형주궁병', '형주병', [('채모', 7), ('채중', 5), ('괴월', 9)])
    party = [('유비', 8, -7), ('장비', 9, -6), ('간옹', 7, -6)] + party_extra
    crowd, clump = field_behind(rnd, party, '유비군', 12, ['SW'] * 5 + ['SE', 'NW'])
    cast.update(crowd)
    folk = [k for keys in clump for k in keys]
    near = sorted(folk, key=lambda k: -cast[k]['At'][1])
    step_back = [a(k, Move=[cast[k]['At'][0], cast[k]['At'][1] - 1], Speed=1.2, After=round(rnd.uniform(0, .5), 2))
                 for k in folk]
    shout = [a(k, Say=t) for k, t in zip(near[::5], ('으악!', '화살이다!', '어머나!'))]
    shout += [a(k, Emote='!') for k in near[2::6][:6]]
    amb = [{'CamAt': [7, -5]}] + [a(k, Chatter=c) for k, c in zip(folk[::9][:4], CRIES)]
    beats = {
        lines[1]: [[[a('채모', Face='NE', Play='action')]],
                   [volley(cast, rows, '형주궁병', rnd)[0],
                    [a('유비', Play='action')] + shout + step_back]],
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
    crowd, clump = field_behind(rnd, party, '유비군', 10, ['NE'] * 4 + ['NW', 'SE', 'SW', 'SW'])
    cast.update(crowd)
    follow = [k for k in rows[1] if cast[k]['At'][0] >= 9] + [k for k in rows[0] if cast[k]['At'][0] >= 11]
    whisperers = [keys[len(keys) // 2] for keys in clump]
    amb = [{'CamAt': [7, -5]}] + [
        a(k, Chatter='|'.join([WHISPERS[i], '', '', WHISPERS[(i + 4) % len(WHISPERS)], '', '']), Lag=round(1 + 1.1 * i, 1))
        for i, k in enumerate(whisperers)]
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
