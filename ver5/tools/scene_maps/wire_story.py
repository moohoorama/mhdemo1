#!/usr/bin/env python3
"""Put a scene map behind every dialogue of the scenario: assets/scenes.json "scenes" gets one entry per node.

  python3 ver5/tools/scene_maps/wire_story.py

A `scene` node names the place the story is in now; every node after it (until the next `scene` node) is staged on
the map PLACES gives for that place. Each entry has the map and a cast: the characters who speak in that stretch of
the story, standing in a row on the map's emptiest ground near its middle, facing each other (the same cast and
places on every node of the stretch, so nobody jumps). Beats are left to the authors; an entry's own beats are
kept. A place that is not in PLACES is reported.
"""
import json
import re
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parents[2]
SCENARIO = ROOT / 'assets/scenario/hero'
SCENES = ROOT / 'assets/scenes.json'

PLACES = {
    '반동탁 연합 · 맹주의 막사': 'S_tent',
    '반동탁 연합 · 맹주의 막사, 며칠 뒤 밤': 'S_tent',
    '산조 · 원소의 막사': 'S_tent',
    '계교 · 공손찬의 막사': 'S_tent',
    '광종 · 노식의 진영': 'S_army_camp',
    '영천 · 장사성 밖 관군 진영': 'S_army_camp',
    '하북 · 관군 진영': 'S_army_camp',
    '호로관 · 연합 진영': 'S_army_camp',
    '호로관 · 연합 진영 저녁': 'S_army_camp',
    '낙양 동쪽 · 조조의 영채': 'S_army_camp',
    '변수 동쪽 · 연합 진영 근처': 'S_army_camp',
    '계교 · 원소의 본진': 'S_army_camp',
    '계교 · 공손찬의 진영': 'S_army_camp',
    '영천 · 장사성 북쪽 언덕': 'S_hill',
    '산조 · 진영 밖 언덕': 'S_hill',
    '영천 · 장사 외곽 마을': 'S_vil_peace',
    '낙양으로 가는 길': 'S_road',
    '하북으로 가는 길': 'S_road',
    '사수관 · 동쪽 들판': 'S_road',
    '거록 · 들판': 'S_road',
    '낙양 · 궁궐 옥좌 앞': 'S_palace',
    '신도성 안': 'S_courtyard',
}
STAGED = ('scene', 'say', 'narrate', 'choice')
NOT_STANDING = ('floor', 'avenue', 'ash', 'carpet', 'wall', 'curtain', 'pillar', 'screen', 'bridge')  # props under or beside the feet
FREE_GLYPHS = '.di'
MAX_CAST = 6


def load_names():
    """id -> (display name, sprite to pick or ''): a template character (a soldier, a messenger) shares its name
    with others, so a scene must name the sprite it stands as."""
    classes = json.load(open(ROOT / 'assets/data/classes.json'))
    classes = classes.get('Classes', classes)
    names = {}
    for f in ('characters.json', 'characters_story.json'):
        d = json.load(open(ROOT / 'assets/data' / f))
        for k, v in d.get('Characters', d).items():
            look = v.get('Look', {})
            sprite = look.get('Scene') or look.get('Sprite') or classes[v['Class']]['Sprite'] if v.get('Template') else ''
            names[k] = (v['Name'], sprite)
    return names


def stage_spot(map_id, cache={}):
    """The point of the map's open ground, near its middle, that is farthest from every standing prop."""
    if map_id not in cache:
        art = json.load(open(ROOT / 'assets/graphics/maps' / f'{map_id}.json'))
        W, H = art['size']
        vx, vy, vw, vh = art['visual']
        small = art['small_tiles']
        props = [(p[1], p[2]) for p in art['props'] if not p[0].startswith(NOT_STANDING) and len(p) == 3]
        best = None
        for i in range(W * 4):
            for j in range(H * 4):
                u, v = i / 4, j / 4
                su, sv = int(u * 2) - vx, int(v * 2) - vy
                if not (0 <= sv < vh and 0 <= su < vw) or small[sv][su] not in FREE_GLYPHS:
                    continue
                gap = min([((u - a) ** 2 + (v - b) ** 2) ** .5 for a, b in props] or [9])
                score = min(gap, 2.4) - .12 * (((u - W / 2 + .5) ** 2 + (v - H / 2 + .5) ** 2) ** .5)
                if best is None or score > best[0]:
                    best = (score, u, v)
        cache[map_id] = (best[1], best[2]) if best else (W / 2 - .5, H / 2 - .5)
    return cache[map_id]


def cast_of(map_id, speakers, names):
    """name -> [u, v, facing]: a row along the u axis, the left half facing the right half."""
    ids = list(dict.fromkeys(speakers))[:MAX_CAST]
    if not ids:
        return {}
    cu, cv = stage_spot(map_id)
    cast = {}
    for i, id in enumerate(ids):
        u = cu + (i - (len(ids) - 1) / 2) * .8
        face = 'SW' if len(ids) == 1 else ('SE' if i < len(ids) / 2 else 'NW')
        name, sprite = names[id]
        cast[name] = [round(u, 2), round(cv, 2), face] + (['', sprite] if sprite else [])
    return cast


def main():
    chapters = json.load(open(SCENARIO / 'scenario.json'))['Chapters']
    old = json.load(open(SCENES))['scenes']
    names = load_names()
    entries, unknown = {}, set()
    segment, map_id, speakers = [], None, []

    def close():
        for node_id in segment:
            entry = dict(old.get(node_id, {}))
            entry['map'] = map_id
            entry['cast'] = cast_of(map_id, speakers, names)
            entry.setdefault('beats', [])
            entries[node_id] = entry

    for ch in chapters:
        for node in json.load(open(SCENARIO / ch['File']))['Nodes']:
            if node['Kind'] == 'scene':
                if segment and map_id:
                    close()
                segment, speakers = [], []
                map_id = PLACES.get(node['Place'])
                if map_id is None:
                    unknown.add(node['Place'])
            elif node['Kind'] == 'battle':
                if segment and map_id:
                    close()
                segment, speakers = [], []
                continue
            if node['Kind'] in STAGED and map_id:
                segment.append(node['ID'])
                if node['Kind'] == 'say':
                    speakers += [x for x in node['Speaker'].split('·') if x in names]
    if segment and map_id:
        close()
    text = open(SCENES, encoding='utf-8').read()
    body = '{\n' + ',\n'.join(f'  {json.dumps(k, ensure_ascii=False)}: {json.dumps(v, ensure_ascii=False)}' for k, v in entries.items()) + '\n }'
    text, n = re.subn(r'"scenes": \{.*\}\s*\n\}\s*$', '"scenes": ' + body + '\n}\n', text, flags=re.S)
    assert n == 1, 'no "scenes" object at the end of scenes.json'
    open(SCENES, 'w', encoding='utf-8').write(text)
    json.loads(text)
    print(len(entries), 'nodes staged')
    for p in sorted(unknown):
        print('no map for place:', p, file=sys.stderr)


if __name__ == '__main__':
    main()
