#!/usr/bin/env python3
"""Give every stretch of the story a scene: assets/scenes/chNN.json gets an empty scene (its map and nodes, no cast)
for each stretch that has none yet. Staged scenes are never touched.

  python3 ver5/tools/scene_maps/wire_story.py

A stretch runs from a `scene` node (or the first node after a battle) to the next `scene` or `battle` node. Its map
comes from the place of the last `scene` node (PLACES). A stretch whose nodes are partly in a scene already, and a
place PLACES does not know, are reported for a person to sort out. Stage the new scenes by hand afterwards
(.claude/skills/scene-staging).
"""
import json
import sys
from pathlib import Path

from scene_format import write

ROOT = Path(__file__).resolve().parents[2]
SCENARIO = ROOT / 'assets/scenario/hero'
SCENES = ROOT / 'assets/scenes'

PLACES = {
    '반동탁 연합 · 맹주의 막사': 'S_council',
    '반동탁 연합 · 맹주의 막사, 며칠 뒤 밤': 'S_council',
    '산조 · 원소의 막사': 'S_council',
    '계교 · 공손찬의 막사': 'S_tent',
    '광종 · 노식의 진영': 'S_army_camp',
    '영천 · 장사성 밖 관군 진영': 'S_army_camp',
    '영천 · 장사성 밖 황건 진영': 'S_turban_camp',
    '영천 · 장사성 성루': 'S_rampart_siege',
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
    '낙양 · 궁궐 옥좌 앞': 'S_hall',
    '신도성 안': 'S_courtyard',
}
STAGED = ('scene', 'say', 'narrate', 'choice')


def stretches(nodes):
    """(map id or None, place, [node ids]) for each stretch of a chapter."""
    out, cur, map_id, place = [], None, None, ''
    for n in nodes:
        if n['Kind'] == 'scene':
            place = n['Place']
            map_id = PLACES.get(place)
            cur = None
        elif n['Kind'] == 'battle':
            cur = None
            continue
        if n['Kind'] not in STAGED:
            continue
        if cur is None:
            cur = (map_id, place, [])
            out.append(cur)
        cur[2].append(n['ID'])
    return out


def main():
    chapters = json.load(open(SCENARIO / 'scenario.json'))['Chapters']
    added = 0
    for ch in chapters:
        path = SCENES / f"ch{ch['ID']}.json"
        scenes = json.load(open(path))['Scenes'] if path.exists() else {}
        staged = {n for s in scenes.values() for n in s['Nodes']}
        changed = False
        for map_id, place, ids in stretches(json.load(open(SCENARIO / ch['File']))['Nodes']):
            have = [n for n in ids if n in staged]
            if len(have) == len(ids):
                continue
            if have:
                print(f'{ids[0]}: nodes {[n for n in ids if n not in staged]} are not in its scene; add them by hand', file=sys.stderr)
                continue
            if map_id is None:
                print(f'{ids[0]}: no map for place {place!r}', file=sys.stderr)
                continue
            scenes[ids[0]] = {'Map': map_id, 'Nodes': ids, 'Cast': {}, 'Beats': {}}
            added += 1
            changed = True
        if changed:
            write(path, scenes)
    print(added, 'empty scenes added')


if __name__ == '__main__':
    main()
