#!/usr/bin/env python3
"""Stages chapter 14 from 여포's arrival to 원술's hall (assets/scenes/ch14.json, beside the deathbed scene staged on
its own) and chapter 15 (assets/scenes/ch15.json), from the tables below. The tables still name the scenes and lines
by the ids they had in the cmd/_scene15 preview (c15-...); RENAME turns those of chapter 14 into its own ids.

  python3 ver5/tools/scene_maps/ch14_15_scenes.py
"""
import json
import re
from pathlib import Path

from scene_format import write

ASSETS = Path(__file__).resolve().parents[2] / 'assets'
story = [n for ch in ('14', '15') for n in json.loads((ASSETS / f'scenario/hero/story{ch}.json').read_text())['Nodes']]
# the preview's ids of the scenes that are in chapter 14 now
RENAME = {'c15-gate': 'c14-gate', 'c15-hall': 'c14-hall', 'c15-envoy': 'c14-envoy', 'c15-yuanshu': 'c14-yuanshu'}
for p in ('lg', 'lh', 'e', 'y'):
    RENAME[f'c15-{p}'] = f'c14-{p}'


def nodes(first):
    """The scene's nodes: from its scene node up to the next scene node or the end."""
    ids = [n['ID'] for n in story]
    i = ids.index(first)
    out = [first]
    for n in story[i + 1:]:
        if n['Kind'] not in ('say', 'narrate'):
            break
        out.append(n['ID'])
    return out


def g(*steps):
    return list(steps)


S = {}
RETREAT_SHU, RETREAT_YUAN = [], []  # filled once the armies are laid out (section 5)
EVERYONE = ['관우', '미축', '미방', '진등', '장비', '조표', '간옹']  # the hall's officers, who bow down to the edict
FACE_NE = [{'Actor': k, 'Face': 'NE'} for k in ['유비'] + EVERYONE]  # toward the envoy, whoever speaks
BOW_NE = FACE_NE + [{'Actor': k, 'Pose': 'prostrate'} for k in ['유비'] + EVERYONE]

# 0. 도겸의 임종 (c14-death) moved into the game: assets/scenes/ch14.json

# 0b. 서주성 앞 (S_xuzhou_gate, 13장의 맵): 여포가 패잔병을 끌고 와 몸을 맡긴다; the party comes out of the gate, 여포
# rides in down the north-west road with 장료, 진궁 on foot beside them
S['c15-gate'] = {
    'Map': 'S_xuzhou_gate',
    'Cast': {
        '수문장': {'At': [11, 5], 'Face': 'NW', 'Off': True},
        '서주병#1': {'At': [12, 5], 'Face': 'NW', 'Sprite': 'light_infantry', 'Faction': 'han'},
        '서주병#2': {'At': [12, 7], 'Face': 'NW', 'Sprite': 'light_infantry', 'Faction': 'han'},
        '유비': {'At': [9, 6], 'Face': 'NW', 'Off': True},
        '관우': {'At': [10, 5], 'Face': 'NW', 'Off': True},
        '장비': {'At': [10, 7], 'Face': 'NW', 'Off': True},
        '진등': {'At': [11, 4], 'Face': 'NW', 'Off': True},
        '미축': {'At': [11, 8], 'Face': 'NW', 'Off': True},
        '여포': {'At': [5, 6], 'Face': 'SE', 'Sprite': 'lubu_unarmed', 'Off': True},
        '장료': {'At': [4, 5], 'Face': 'SE', 'Sprite': 'zhangliao', 'Faction': 'dong', 'Off': True},
        '진궁': {'At': [5, 7], 'Face': 'SE', 'Off': True},
        '기병#1': {'At': [3, 6], 'Face': 'SE', 'Sprite': 'light_cavalry', 'Faction': 'dong', 'Off': True},
        '기병#2': {'At': [2, 5], 'Face': 'SE', 'Sprite': 'light_cavalry', 'Faction': 'dong', 'Off': True},
        '기병#3': {'At': [2, 7], 'Face': 'SE', 'Sprite': 'light_cavalry', 'Faction': 'dong', 'Off': True},
    },
    'Ambient': [{'CamAt': [8, 7]}],
    'Beats': {
        'c15-lg01': [[g({'Actor': '수문장', 'Enter': [13, 6], 'Move': [11, 5], 'Speed': 1.8}), g({'Actor': '수문장', 'Face': 'NW'})]],
        'c15-lg02': [[g({'Actor': '유비', 'Enter': [13, 6], 'Move': [9, 6]},
                       {'Actor': '관우', 'Enter': [13, 6], 'Move': [10, 5], 'After': .3},
                       {'Actor': '장비', 'Enter': [13, 6], 'Move': [10, 7], 'After': .5},
                       {'Actor': '진등', 'Enter': [13, 6], 'Move': [11, 4], 'After': .8},
                       {'Actor': '미축', 'Enter': [13, 6], 'Move': [11, 8], 'After': 1},
                       {'Actor': '여포', 'Enter': [-1, 6], 'Move': [5, 6]},
                       {'Actor': '장료', 'Enter': [-1, 5], 'Move': [4, 5], 'After': .3},
                       {'Actor': '진궁', 'Enter': [-1, 7], 'Move': [5, 7], 'After': .4},
                       *[{'Actor': f'기병#{i}', 'Enter': [-1, 6], 'Move': p, 'After': .5 + .3 * i}
                         for i, p in ((1, [3, 6]), (2, [2, 5]), (3, [2, 7]))]),
                     g(*[{'Actor': k, 'Face': 'NW'} for k in ('유비', '관우', '장비', '진등', '미축')],
                       *[{'Actor': k, 'Face': 'SE'} for k in ('여포', '장료', '진궁', '기병#1', '기병#2', '기병#3')])]],
        'c15-lg04': [[g({'Actor': '장비', 'Emote': '!'})]],
        'c15-lg07': [[g({'Actor': '진궁', 'Move': [6, 7]}), g({'Actor': '진궁', 'Play': 'action'})]],
        'c15-lg08': [[g({'Actor': '여포', 'Move': [7, 6]})]],
        'c15-lg09': [[g({'Actor': '장비', 'Play': 'action'}, {'Actor': '관우', 'Move': [9, 7]}), g({'Actor': '관우', 'Face': 'NE'})]],
    },
}

# 0c. 서주 관청: whether to take 여포 in; then he is called in and sent to 소패
S['c15-hall'] = {
    'Map': 'S_xuzhou_hall',
    'Cast': {
        '유비': {'At': [4, 1], 'Face': 'SW'},
        '관우': {'At': [2, 2], 'Face': 'SE'},
        '미축': {'At': [2, 3], 'Face': 'SE'},
        '미방': {'At': [2, 4], 'Face': 'SE'},
        '진등': {'At': [2, 5], 'Face': 'SE'},
        '장비': {'At': [6, 2], 'Face': 'NW'},
        '조표': {'At': [6, 3], 'Face': 'NW'},
        '간옹': {'At': [6, 4], 'Face': 'NW'},
        '여포': {'At': [4, 4], 'Face': 'NE', 'Off': True},
        '진궁': {'At': [4, 5], 'Face': 'NE', 'Off': True},
    },
    'Beats': {
        'c15-lh02': [[g({'Actor': '장비', 'Play': 'action'})]],
        'c15-lh07': [[g({'Actor': '조표', 'Emote': '…'})]],
        'c15-lh08': [[g({'Actor': '여포', 'Enter': [4, 9], 'Move': [4, 4]}, {'Actor': '진궁', 'Enter': [4, 9], 'Move': [4, 5], 'After': .4}),
                     g({'Actor': '여포', 'Face': 'NE'}, {'Actor': '진궁', 'Face': 'NE'})]],
        'c15-lh09': [[g({'Actor': '여포', 'Play': 'action'})]],
        'c15-lh10': [[g({'Actor': '진궁', 'Pose': 'bow'}, {'Wait': .8}),
                     g({'Actor': '여포', 'Move': [4, 10], 'Exit': True}, {'Actor': '진궁', 'Pose': 'idle'}),
                     g({'Actor': '진궁', 'Move': [4, 10], 'Exit': True})]],
    },
}

# 1. 하비 관청: the edict
S['c15-envoy'] = {
    'Map': 'S_xuzhou_hall',
    'Cast': {
        '유비': {'At': [4, 1], 'Face': 'SW', 'Hold': True},
        '관우': {'At': [2, 2], 'Face': 'SE'},
        '미축': {'At': [2, 3], 'Face': 'SE'},
        '미방': {'At': [2, 4], 'Face': 'SE'},
        '장비': {'At': [6, 2], 'Face': 'NW'},
        '조표': {'At': [6, 3], 'Face': 'NW'},
        '간옹': {'At': [6, 4], 'Face': 'NW'},
        '진등': {'At': [2, 5], 'Face': 'SE'},
        '조정 사자': {'At': [4, 4], 'Face': 'NE', 'Off': True},
    },
    'Beats': {
        # the envoy takes the seat of honour, Liu Bei comes down onto the carpet, and the hall bows to the edict
        'c15-e00': [[g({'Actor': '조정 사자', 'Enter': [4, 9], 'Move': [4, 7]},
                       {'Actor': '유비', 'Via': [[3, 1], [3, 4]], 'Move': [4, 4]}),
                     g({'Actor': '조정 사자', 'Via': [[3, 7], [3, 1]], 'Move': [4, 1]}),
                     g({'Actor': '조정 사자', 'Face': 'SW'}, *BOW_NE)]],
        'c15-e01': [[g(*FACE_NE)]],
        'c15-e02': [[g(*FACE_NE)]],
        'c15-e02b': [[g(*FACE_NE)]],
        # the envoy leaves; only then does the hall rise, and Liu Bei go back to his seat
        'c15-e03': [[g({'Actor': '조정 사자', 'Play': 'action'}, {'Wait': .6}),
                     g({'Actor': '조정 사자', 'Via': [[3, 1], [3, 9]], 'Move': [4, 10], 'Exit': True}),
                     g(*[{'Actor': k, 'Pose': 'idle'} for k in ['유비'] + EVERYONE], {'Wait': .4}),
                     g({'Actor': '유비', 'Via': [[3, 4], [3, 1]], 'Move': [4, 1]}),
                     g({'Actor': '유비', 'Face': 'SW'})]],
        'c15-e04': [[g({'Actor': '장비', 'Play': 'action'})]],
        'c15-e06': [[g({'Actor': '유비', 'Play': 'action'})]],
        # 진등 steps out of the file toward the seat to argue for stalling; 유비 refuses, and he goes back
        'c15-e06a': [[g({'Actor': '진등', 'Move': [3, 4]}), g({'Actor': '진등', 'Face': 'NE'})]],
        'c15-e06c': [[g({'Actor': '유비', 'Emote': '!'})]],
        'c15-e06d': [[g({'Actor': '진등', 'Play': 'action'})]],
        'c15-e06g': [[g({'Actor': '진등', 'Emote': '…'}), g({'Actor': '진등', 'Move': [2, 5]}), g({'Actor': '진등', 'Face': 'SE'})]],
        'c15-e08': [[g({'Actor': '조표', 'Emote': '…'}), g({'Actor': '조표', 'Play': 'action'})]],
    },
}

# 2. 수춘 대전: 원술 on his throne, the whole court in files facing each other across the carpet: the civil officials
# (SE) show their faces, the generals (NW) their backs (tools/scene_maps/grand_halls.py)
RANKS = ['official_purple_noncombat', 'official_crimson_noncombat', 'official_crimson_noncombat',
         'official_blue_noncombat', 'official_green_noncombat', 'official_noncombat']  # nearest the seat first


def court(prefix, faction, sprites, face, cols, rows, pose='', skip=()):
    out, i = {}, 0
    for r, v in enumerate(rows):
        for u in cols:
            if (u, v) in skip:
                continue
            i += 1
            a = {'At': [u, v], 'Face': face, 'Sprite': sprites[r % len(sprites)], 'Faction': faction}
            if pose:
                a['Pose'] = pose
            out[f'{prefix}#{i}'] = a
    return out


S['c15-yuanshu'] = {
    'Map': 'S_shouchun_hall',
    'Cast': {
        '원술': {'At': [6, 1], 'Face': 'SW', 'Hold': True, 'Pose': 'sit'},
        '양홍': {'At': [4, 3], 'Face': 'SE'},
        '기령': {'At': [8, 3], 'Face': 'NW'},
        **court('문관', 'yuan', RANKS, 'SE', [2, 3, 4], range(3, 9), skip={(4, 3)}),
        **court('무관', 'yuan', ['infantry_noncombat'], 'NW', [8, 9, 10], range(3, 9), skip={(8, 3)}),
        **{f'호위#{i}': {'At': at, 'Face': f, 'Sprite': 'heavy_infantry', 'Faction': 'yuan'}
           for i, (at, f) in enumerate([([4, 1], 'SE'), ([8, 1], 'NW'), ([5, 11], 'NE'), ([7, 11], 'NE')], 1)},
    },
    'Ambient': [{'CamAt': [6, 3]}],
    'Beats': {
        'c15-y01': [[g({'Actor': '원술', 'Play': 'action', 'Shake': .25})]],
        'c15-y02': [[g({'Actor': '양홍', 'Emote': '!'})]],
        'c15-y04': [[g({'Actor': '양홍', 'Move': [5, 3]}, {'CamAt': [6, 6]}), g({'Actor': '양홍', 'Face': 'NE'})]],
        'c15-y05': [[g({'CamAt': [6, 3]})]],
        'c15-y06': [[g({'Actor': '기령', 'Move': [7, 3]}), g({'Actor': '기령', 'Face': 'NE'}, {'Actor': '기령', 'Play': 'action'})]],
    },
}

# 3. 광릉 진영: grain handed out, then the two brothers at the camp's mouth
S['c15-grain'] = {
    'Map': 'S_gongsun_camp',
    'Time': 'dusk',
    'Cast': {
        '미축': {'At': [1, 5], 'Face': 'SW'},
        '병사#1': {'At': [1, 6], 'Face': 'NE', 'Sprite': 'light_infantry', 'Faction': 'shu'},
        '병사#2': {'At': [1, 7], 'Face': 'NE', 'Sprite': 'light_infantry', 'Faction': 'shu'},
        '병사#3': {'At': [1, 8], 'Face': 'NE', 'Sprite': 'light_infantry', 'Faction': 'shu'},
        '병사#4': {'At': [2, 8], 'Face': 'NW', 'Sprite': 'light_infantry', 'Faction': 'shu'},
        '미방': {'At': [12, 6], 'Face': 'NW', 'Off': True},
    },
    'Ambient': [{'Effect': 'smallfire', 'At': [7, 10]}, {'CamAt': [3, 6]},
                {'Actor': '병사#4', 'Chatter': '배고파…|언제 먹나|꼬르륵'}],
    'Beats': {
        'c15-g01': [[g({'Actor': '병사#1', 'Move': [2, 6]}, {'Actor': '병사#2', 'Move': [1, 6], 'After': .3},
                      {'Actor': '병사#3', 'Move': [1, 7], 'After': .6}),
                     g({'Actor': '병사#1', 'Move': [5, 6], 'Exit': True}, {'Actor': '미축', 'Play': 'action'})]],
        'c15-g02': [[g({'Actor': '미방', 'Enter': [12, 6], 'Move': [4, 6]}, {'Cam': '미방'}),
                     g({'Actor': '미축', 'Move': [3, 5]}, {'CamAt': [4, 6]}),
                     g({'Actor': '미축', 'Face': 'SE'}, {'Actor': '미방', 'Face': 'NW'})]],
        'c15-g08': [[g({'Actor': '미축', 'Play': 'action'})]],
        'c15-g09': [[g({'Actor': '미축', 'Move': [1, 5]}, {'Actor': '미축', 'Face': 'SW', 'After': 2}),
                     g({'Actor': '미방', 'Face': 'SE'}, {'Actor': '미방', 'Emote': '…'})]],
    },
}

# 4. 광릉 군막: the spy
S['c15-spy'] = {
    'Map': 'S_tent_mid',
    'Cast': {
        '유비': {'At': [3, 1], 'Face': 'SW'},
        '미축': {'At': [2, 2], 'Face': 'SE'},
        '미방': {'At': [2, 3], 'Face': 'NE'},
        '장비': {'At': [4, 2], 'Face': 'NW'},
        '관우': {'At': [4, 3], 'Face': 'NW', 'Off': True},
        '첩자#1': {'At': [5, 3], 'Face': 'NW', 'Sprite': 'commoner_man', 'Faction': 'yuan', 'Off': True},
    },
    'Beats': {
        'c15-s02': [[g({'Actor': '첩자#1', 'Enter': [7, 3], 'Move': [5, 3]}, {'Actor': '관우', 'Enter': [7, 3], 'Move': [4, 3], 'After': .5}),
                     g({'Actor': '첩자#1', 'Face': 'NW'}, {'Actor': '관우', 'Face': 'SE'}),
                     g({'Actor': '첩자#1', 'Play': 'hit', 'Shake': .2})]],
        'c15-s03': [[g({'Actor': '첩자#1', 'Pose': 'exhausted'}), g({'Actor': '관우', 'Move': [3, 3]}, {'Actor': '미방', 'Emote': '!'}),
                     g({'Actor': '관우', 'Face': 'NW'}, {'Actor': '미방', 'Face': 'SE'}, {'Actor': '관우', 'Play': 'action', 'Shake': .3})]],
        'c15-s04': [[g({'Actor': '장비', 'Move': [4, 3]}, {'Actor': '미축', 'Look': 'SE'}),
                     g({'Actor': '장비', 'Face': 'NW'})]],
        'c15-s09': [[g({'Actor': '미축', 'Face': 'NE'}, {'Actor': '미축', 'Pose': 'bow'})]],
        'c15-s10': [[g({'Actor': '미방', 'Face': 'NE'}, {'Actor': '미방', 'Pose': 'bow'})]],
        'c15-s12': [[g({'Actor': '미축', 'Pose': 'idle'}, {'Actor': '미방', 'Pose': 'idle'})]],
        'c15-s13': [[g({'Actor': '관우', 'Look': 'SE'}, {'Actor': '장비', 'Look': 'SE'}, {'Actor': '유비', 'Emote': '!', 'Shake': .4})]],
    },
}

# 5. 광릉 들판 (S_long_field, tools/scene_maps/long_field.py): the brothers and 기령 face off at the head of the field,
# 여포's halberd flies in and drives its blades into the ground between them, 여포 rides in with 장료 behind him on his
# left; both sides send their men back, and 유비, 여포 and 기령 ride south-west down the road side by side, talking,
# 관우, 장비 and 장료 behind, to the clearing where the road ends, 150 cells from the halberd (a player who reads fast
# is led on there at a gallop: "나를 따라와 보시오"). 여포 takes his bow in a fade, the others draw aside, and he
# shoots back north-east along the road; the camera follows the arrow there.
HALBERD = [5.5, 9]
FAR = 159              # the clearing at the road's end (long_field.py END): 150 cells from the halberd; the ride ends there
LEADERS = {'유비': 4, '여포': 6, '기령': 8}      # u, abreast
FOLLOW = {'관우': 4, '장비': 5, '장료': 7}       # u, two cells behind
START = 13
ASIDE = {'유비': [3, FAR], '관우': [2, FAR - 1], '장비': [3, FAR - 2],   # where they stand aside for the shot
         '기령': [9, FAR], '장료': [10, FAR - 1]}


def ride_to(v0, speed, free):
    out = [{'Actor': k, 'Move': [u, v0], 'Speed': speed, 'Free': free} for k, u in LEADERS.items()]
    return out + [{'Actor': k, 'Move': [u, v0 - 2], 'Speed': speed, 'Free': free} for k, u in FOLLOW.items()]


S['c15-clash'] = {
    'Map': 'S_long_field',
    'Backdrop': 'mountains',
    'Cast': {
        '유비': {'At': [2, 9], 'Face': 'SE', 'Sprite': 'liubei_mounted'},
        '관우': {'At': [3, 8], 'Face': 'SE', 'Sprite': 'guanyu'},
        '장비': {'At': [3, 10], 'Face': 'SE', 'Sprite': 'zhangfei'},
        '기령': {'At': [9, 9], 'Face': 'NW', 'Sprite': 'light_cavalry'},
        '여포': {'At': [6, 12], 'Face': 'NE', 'Sprite': 'lubu_unarmed', 'Off': True},
        '장료': {'At': [5, 13], 'Face': 'NE', 'Sprite': 'light_cavalry', 'Faction': 'dong', 'Off': True},
    },
    'Ambient': [{'CamAt': [6, 9]}],
    'Beats': {
        'c15-c01': [[g({'Actor': '기령', 'Play': 'attack'}, {'Actor': '원술군#5', 'Say': '와아아!'},
                       {'Actor': '원술군#12', 'Say': '짓밟아라!', 'After': .3})]],
        'c15-c02': [[g({'Actor': '장비', 'Play': 'attack'})]],
        # they charge at each other; the beats wait for the halberd, and where it lands the riders pull up
        'c15-c04': [[g({'Actor': '관우', 'Move': [6, 8], 'Speed': .6, 'Free': True},
                       {'Actor': '장비', 'Move': [6, 10], 'Speed': .6, 'Free': True},
                       {'Actor': '기령', 'Move': [5, 9], 'Speed': .6, 'Free': True},
                       {'Effect': 'halberd_throw', 'At': HALBERD, 'After': .25}),
                     g({'Actor': '관우', 'Emote': '!'}, {'Actor': '장비', 'Emote': '!'}, {'Actor': '기령', 'Emote': '!'},
                       {'Actor': '유비', 'Emote': '!'}, {'Wait': .7}),
                     g({'Actor': '여포', 'Enter': [6, 24], 'Move': [6, 12], 'Speed': 2},
                       {'Actor': '장료', 'Enter': [5, 25], 'Move': [5, 13], 'Speed': 2, 'After': .25}, {'CamAt': [6, 11]}),
                     g({'Actor': '여포', 'Face': 'NE'}, {'Actor': '장료', 'Face': 'NE'})]],
        'c15-c05': [[g({'Actor': '기령', 'Play': 'hit'})]],
        'c15-c06': [[g({'Actor': '장비', 'Play': 'attack'})]],
        'c15-c09': [[g({'Actor': '유비', 'Move': [3, 9]}), g({'Actor': '유비', 'Face': 'NE'}, {'Actor': '관우', 'Face': 'SW'})]],
        'c15-r01': [[g(*ride_to(START, 1, False)),
                     g(*ride_to(FAR, 1, True), {'Follow': '여포'})]],
        # a player who read fast is not there yet: 여포 leads them on to where the ride ends
        'c15-r12': [[g({'Actor': '여포', 'Say': '나를 따라와 보시오.', 'IfWalking': True}, *ride_to(FAR, 2, False))]],
        # 여포 takes his bow in the dark of a fade, 150 cells on; he turns back north-east and shoots at the halberd
        'c15-p11': [[g({'Time': 'day'}, {'Actor': '여포', 'Sprite': 'lubu_bow'},
                       *[{'Actor': k, 'Place': [u, FAR]} for k, u in LEADERS.items()],
                       *[{'Actor': k, 'Place': [u, FAR - 2]} for k, u in FOLLOW.items()],
                       *[{'Actor': k, 'Face': 'NE'} for k in [*LEADERS, *FOLLOW]]),
                     g({'Wait': .3}),
                     # the others draw aside out of the arrow's way, left and right; 여포 stands alone on the road
                     g(*[{'Actor': k, 'Move': at} for k, at in ASIDE.items()]),
                     g(*[{'Actor': k, 'Face': 'NE'} for k in ASIDE], {'Wait': .3}),
                     g({'Actor': '여포', 'Shoot': {'Time': 2.5, 'Peak': 160, 'Say': '어디……'}}),
                     g({'Actor': '기령', 'Emote': '…'}, {'Actor': '유비', 'Emote': '!'}, {'Actor': '장비', 'Emote': '!'}, {'Wait': .8})]],
        'c15-p13': [[g({'Actor': '기령', 'Move': [9, FAR - 30], 'Speed': 1.5, 'Exit': True})]],
        'c15-p18': [[g({'Actor': '장비', 'Play': 'attack', 'Shake': .3})]],
        'c15-p19': [[g({'Actor': '관우', 'Look': 'SE'})]],
        'c15-p21': [[g({'Actor': '여포', 'Play': 'block'})]],
    },
}

# the two armies fill the field behind their generals to the screen's edges: horsemen in front, foot behind
_bush = set()


def _army(prefix, faction, face, cols, rows):
    out, i = {}, 0
    for c, u in enumerate(cols):
        for v in rows:
            if (u, v) in _bush:
                continue
            i += 1
            out[f'{prefix}#{i}'] = {'At': [u, v], 'Face': face, 'Faction': faction,
                                    'Sprite': 'light_cavalry' if c == 0 else 'light_infantry'}
    return out


SHU = _army('병사', 'shu', 'SE', [1, 0, -1, -2], range(4, 15))
YUAN = _army('원술군', 'yuan', 'NW', [11, 12, 13, 14], range(4, 15))
S['c15-clash']['Cast'].update(SHU)
S['c15-clash']['Cast'].update(YUAN)
for i, (k, a) in enumerate(SHU.items()):
    RETREAT_SHU.append({'Actor': k, 'Move': [a['At'][0] - 9, a['At'][1]], 'Exit': True})  # all at once, ranks kept
for i, (k, a) in enumerate(YUAN.items()):
    RETREAT_YUAN.append({'Actor': k, 'Move': [a['At'][0] + 9, a['At'][1]], 'Exit': True})
# the brothers' men turn and walk back off the field after 관우's order; 기령's after his
S['c15-clash']['Beats']['c15-c11'] = [[RETREAT_SHU]]
S['c15-clash']['Beats']['c15-c13'] = [[RETREAT_YUAN]]


def renamed(x):
    """x with the chapter 14 ids of the preview turned into the chapter's own (c15-e01 -> c14-e01, not c15-end)."""
    t = json.dumps(x, ensure_ascii=False)
    t = re.sub(r'"(c15-(?:lg|lh|e|y)\d+[a-z]?)"', lambda m: '"c14-' + m.group(1)[4:] + '"', t)
    for a, b in RENAME.items():
        if not a[-1].isalpha() or a in ('c15-gate', 'c15-hall', 'c15-envoy', 'c15-yuanshu'):
            t = t.replace(f'"{a}"', f'"{b}"')
    return json.loads(t)


S = renamed(S)
for sid, sc in S.items():
    sc['Nodes'] = nodes(sid)
ch14 = json.loads((ASSETS / 'scenes/ch14.json').read_text())['Scenes']
ch14 = {k: v for k, v in ch14.items() if not k.startswith('c14-') or k == 'c14-death'}
ch14.update({k: v for k, v in S.items() if k.startswith('c14-')})
write(ASSETS / 'scenes/ch14.json', ch14)
write(ASSETS / 'scenes/ch15.json', {k: v for k, v in S.items() if k.startswith('c15-')})
print({k: len(v['Nodes']) for k, v in S.items()})
