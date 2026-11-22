#!/usr/bin/env python3
"""Write assets/scenes/ch21.json, the staging of chapter 21 (여남): the office meeting, the two alarms, the north
gate opened to Cao Cao's men at night, and the road south after the battle.

  python3 ver5/tools/scene_maps/ch21_scenes.py
"""
import json
import sys
from pathlib import Path

sys.path.insert(0, str(Path(__file__).parent))
import scene_format  # noqa: E402

ROOT = Path(__file__).resolve().parents[2]
story = json.load(open(ROOT / 'assets/scenario/hero/story21.json'))['Nodes']


def stretch(first):
    """The scene node and the dialogue nodes after it, up to the next scene or battle."""
    out, on = [], False
    for n in story:
        if n['ID'] == first:
            on = True
        elif on and n['Kind'] in ('scene', 'battle'):
            break
        if on and n['Kind'] in ('scene', 'say', 'narrate'):
            out.append(n['ID'])
    return out


def a(actor, **k):
    return {'Actor': actor, **k}


DOOR = [7, 4]   # the office's entrance, off its floor
office1 = {
    'Map': 'S_zhuo_office', 'Nodes': stretch('c21-runan'),
    'Cast': {
        '유벽': {'At': [4, 2], 'Face': 'SW'}, '공도': {'At': [5, 2], 'Face': 'SW'},
        '유비': {'At': DOOR, 'Face': 'NW', 'Off': True}, '관우': {'At': DOOR, 'Face': 'NW', 'Off': True},
        '장비': {'At': DOOR, 'Face': 'NW', 'Off': True}, '간옹': {'At': DOOR, 'Face': 'NW', 'Off': True},
        '손건': {'At': DOOR, 'Face': 'NW', 'Off': True},
        '전령': {'At': DOOR, 'Face': 'NW', 'Off': True, 'Sprite': 'light_infantry', 'Faction': 'shu'},
    },
    'Beats': {
        'c21-r01': [[[{'CamAt': [4, 3]}, {'Zoom': 3},
                      a('관우', Enter=DOOR, Move=[3, 4]), a('유비', Enter=DOOR, Move=[4, 4], After=.5),
                      a('장비', Enter=DOOR, Move=[5, 4], After=1.0),
                      a('간옹', Enter=DOOR, Via=[[6, 4], [6, 5]], Move=[3, 5], After=1.5),
                      a('손건', Enter=DOOR, Via=[[6, 4], [6, 5]], Move=[5, 5], After=2.0)],
                     [a('유비', Face='NE'), a('관우', Face='NE'), a('장비', Face='NE'), a('간옹', Face='NE'), a('손건', Face='NE')]]],
        'c21-r04': [[[a('장비', Face='NW'), a('장비', Emote='?')]]],
        'c21-r05': [[[a('유벽', Face='SE'), a('장비', Face='NE')]]],
        'c21-r08': [[[a('전령', Enter=DOOR, Move=[6, 4], Speed=2)],
                     [a('유비', Face='SE'), a('관우', Face='SE'), a('장비', Face='SE'), a('유벽', Emote='!'), a('공도', Emote='!')]]],
        'c21-r09': [[], [[a('전령', Move=DOOR, Speed=1.4, Exit=True)]]],
        'c21-r10': [[[a('간옹', Emote='…')]]],
        'c21-r13': [[[a('유비', Face='NE')]]],
        'c21-r16': [[[a('손건', Face='NW'), a('유비', Face='SE')]]],
        'c21-r21': [[[a('장비', Face='SW')]], [[a('손건', Via=[[6, 5], [6, 4]], Move=DOOR, Exit=True)]]],
        'c21-r22': [[[a('유벽', Face='SE')]]],
        'c21-r23': [[], [[a('공도', Via=[[5, 3], [6, 3]], Move=DOOR, Exit=True)]]],
    },
}

office2 = {
    'Map': 'S_zhuo_office', 'Time': 'dusk', 'Nodes': stretch('c21-alarm'),
    'Cast': {
        '유벽': {'At': [4, 2], 'Face': 'SW'}, '유비': {'At': [2, 2], 'Face': 'SE'},
        '관우': {'At': [2, 3], 'Face': 'SE'}, '장비': {'At': [5, 2], 'Face': 'SW'},
        '간옹': {'At': [2, 4], 'Face': 'NE'}, '조운': {'At': [4, 5], 'Face': 'NE'},
        '전령': {'At': DOOR, 'Face': 'NW', 'Off': True, 'Sprite': 'light_infantry', 'Faction': 'shu'},
    },
    'Beats': {
        'c21-a01': [[[{'CamAt': [4, 3]}, {'Zoom': 3}, {'Wait': .6}], [a('전령', Enter=DOOR, Move=[5, 4], Speed=2)],
                     [a('유비', Face='SE'), a('유벽', Face='SE'), a('장비', Emote='!')]],
                    [[a('전령', Move=DOOR, Speed=1.6, Exit=True)]]],
        'c21-a03': [[[a('전령', Enter=DOOR, Move=[5, 4], Speed=2.2)], [a('유벽', Emote='!'), a('유비', Emote='!')]]],
        'c21-a04': [[[a('유벽', Move=[4, 3])]]],
        'c21-a07': [[[a('관우', Face='NW'), a('장비', Face='NW')]]],
        'c21-a08': [[], [[a('장비', Via=[[5, 3], [6, 3]], Move=DOOR, Speed=1.4, Exit=True),
                          a('전령', Move=DOOR, Speed=1.4, Exit=True, After=.4)]]],
        'c21-a09': [[], [[a('관우', Via=[[2, 4], [3, 4]], Move=DOOR, Speed=1.2, Exit=True)]]],
        'c21-a10': [[[a('조운', Via=[[3, 5], [3, 3]], Move=[3, 3])], [a('조운', Face='NW'), a('유비', Face='SE')]]],
    },
}

GATE = 8   # the north gate's passage, cell u (v = -1)
wei = {'Faction': 'wei', 'Off': True, 'Face': 'SW'}
gate = {
    'Map': 'S_runan', 'Time': 'night', 'Nodes': stretch('c21-gate'), 'Ambient': [{'CamAt': [8, 3]}],
    'Cast': {
        '유비': {'At': [8, 5], 'Face': 'NE'}, '조운': {'At': [9, 5], 'Face': 'NE'}, '유벽': {'At': [7, 5], 'Face': 'NE'},
        '간옹': {'At': [7, 6], 'Face': 'NE'}, '고순': {'At': [9, 6], 'Face': 'NE', 'If': ['party:gaoshun']},
        '수문장': {'At': [7, 1], 'Face': 'SW', 'Faction': 'shu'},
        '문지기#1': {'At': [9, 1], 'Face': 'SW', 'Sprite': 'light_infantry', 'Faction': 'shu'},
        '악진': {'At': [GATE, -3], 'Sprite': 'chariot', **wei},
        '고람': {'At': [GATE, -3], **wei},
        '조조군#1': {'At': [GATE, -4], 'Sprite': 'light_cavalry', **wei},
        '조조군#2': {'At': [GATE, -5], 'Sprite': 'light_cavalry', **wei},
    },
    'Beats': {
        'c21-g01': [[[{'Zoom': 2}, a('수문장', Emote='!')]]],
        'c21-g03': [[[a('수문장', Face='NE'), a('문지기#1', Face='NE'),
                      a('악진', Enter=[GATE, -3], Move=[GATE, 1], Speed=1.8),
                      a('조조군#1', Enter=[GATE, -4], Via=[[GATE, 0]], Move=[9, 0], Speed=1.8, After=.5),
                      a('조조군#2', Enter=[GATE, -5], Via=[[GATE, 0]], Move=[7, 0], Speed=1.8, After=1.0)],
                     [a('수문장', Emote='!'), a('유비', Emote='!')],
                     [a('수문장', Move=[6, 4], Speed=2.2), a('문지기#1', Move=[10, 3], Speed=2.2)],
                     [a('수문장', Face='NE'), a('문지기#1', Face='NE')]]],
        'c21-g05': [[[a('고람', Enter=[GATE, -3], Move=[GATE, 0], Speed=1.2)]]],
        'c21-g08': [[[a('간옹', Emote='!')]]],
        'c21-g10': [[[a('악진', Play='attack')]]],
        'c21-g11': [[[a('유벽', Face='SE')]]],
        'c21-g12': [[[a('조운', Via=[[9, 4]], Move=[8, 4])], [a('조운', Face='NE')]],
                    [[a('유벽', Face='SW'), a('유비', Face='SW'), a('간옹', Face='SW')]]],
    },
}

alive = ['flag:flag-b21-liupi']
south = {
    'Map': 'S_road', 'Time': 'dawn', 'Nodes': stretch('c21-south'), 'Ambient': [{'CamAt': [4, 2]}],
    'Cast': {
        '유비': {'At': [4, 2], 'Face': 'SE'}, '관우': {'At': [6, 2], 'Face': 'NW'}, '장비': {'At': [6, 3], 'Face': 'NW'},
        '조운': {'At': [3, 1], 'Face': 'SE'}, '간옹': {'At': [3, 3], 'Face': 'SE'}, '손건': {'At': [2, 2], 'Face': 'SE'},
        '유벽': {'At': [9, 3], 'Face': 'NW', 'If': alive},
    },
    'Beats': {
        'c21-s01': [[[a('관우', Emote='…')]]],
        'c21-s03': [[[a('장비', Face='SE')]]],
        'c21-p01': [[[a('유벽', Move=[7, 4])], [a('유비', Face='SE'), a('장비', Face='SE')]]],
        'c21-d01': [[[a('유비', Face='SE'), a('유비', Emote='…')]]],
        'c21-e01': [[[a('손건', Via=[[2, 1]], Move=[4, 1])], [a('손건', Face='SW'), a('유비', Face='NE')]]],
        'c21-e02': [[], [[a('유비', Move=[-3, 2], Exit=True), a('손건', Move=[-3, 1], Exit=True, After=.2),
                          a('조운', Move=[-3, 1], Exit=True, After=.5), a('간옹', Move=[-3, 3], Exit=True, After=.4),
                          a('장비', Move=[-3, 3], Exit=True, After=.9), a('관우', Move=[-3, 2], Exit=True, After=1.2),
                          a('유벽', Move=[-3, 3], Exit=True, After=1.6)]]],
    },
}

scene_format.write(ROOT / 'assets/scenes/ch21.json',
                   {'c21-runan': office1, 'c21-alarm': office2, 'c21-gate': gate, 'c21-south': south})
print('ch21: 4 scenes')
