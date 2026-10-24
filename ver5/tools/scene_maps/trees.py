"""Plant the story scene maps' trees: the 3D scene trees of tools/unit3d/scene_props3d.py (pines, broadleaves, a
willow, an old tree, bushes, blossom trees, burnt trees), at real size — a few people tall — instead of the battle
map's small tile trees. build.py and compose.py hand every tree they would have drawn to plant(): it thins them to
the big trees' spacing and keeps them off and out of the way of the people on the stage: no tree within reach of a
spot where a scene stands someone, and none in front of such a spot hiding them. The spots are the map's playing
area and every cast place and walk end of the scenes in assets/scenes that use the map (or a variant of it).
"""
import json
import math
import random
from pathlib import Path

ROOT = Path(__file__).resolve().parents[3]
SCENES = ROOT / 'ver5/assets/scenes'
TILES = ROOT / 'assets_candidate/tiles/tiles.json'

SPECIES = {
    'broad': ['stree_broad_0', 'stree_broad_1'], 'pine': ['stree_pine_0', 'stree_pine_1'], 'old': ['stree_old'],
    'willow': ['stree_willow'], 'bush': ['stree_bush_0', 'stree_bush_1'], 'dead': ['stree_dead'],
    'peach': ['peach_bloom_0', 'peach_bloom_1', 'peach_bloom_2'], 'plum': ['stree_plum_0', 'stree_plum_1'],
}
FIRST = {'plum', 'peach', 'dead'}
GAP = {'bush': .8}  # cells between two trees' feet; big trees 1.5
REACH = {'bush': .8}  # cells a tree's foot keeps from a person's
# maps that share their trees (a variant copies its base's): a spot on any of them protects all
FAMILY = [{'S_army_camp', 'S_turban_camp', 'S_army_camp_hungry'}, {'S_castle_out', 'S_castle_out_open'},
          {'S_vil_peace', 'S_vil_burnt'}, {'S_castle_town', 'S_castle_town_burnt'}, {'S_rampart', 'S_rampart_siege'}]


def family(map_id):
    return next((f for f in FAMILY if map_id in f), {map_id})


def scene_spots(map_id):
    """Where the scenes on this map (and its family) put people: cast places, entries, walk ends, patrol ends."""
    maps, spots = family(map_id), []
    for path in sorted(SCENES.glob('*.json')):
        for sc in json.load(open(path))['Scenes'].values():
            if sc['Map'] not in maps:
                continue
            spots += [tuple(a['At']) for a in sc['Cast'].values() if len(a.get('At', [])) == 2]
            for line in (l for lines in sc.get('Beats', {}).values() for l in lines):
                for st in (s for g in line for s in g):
                    for k in ('Enter', 'Move', 'Patrol'):
                        if len(st.get(k) or []) == 2:
                            spots.append(tuple(st[k]))
    return spots


def stage_spots(W, H):
    return [(u / 2, v / 2) for u in range(-1, 2 * W) for v in range(-1, 2 * H)]


def screen(u, v):
    return 32 * (u - v), 16 * (u + v)


def hides(name, tu, tv, su, sv, sizes):
    """Whether the tree's picture stands in front of a person at (su, sv) and covers their body."""
    if tu + tv <= su + sv:
        return False
    w, h, px, py = sizes[name]
    tx, ty = screen(tu, tv)
    x0, y0 = tx - px, ty - py
    bx, by = screen(su, sv)
    return x0 < bx + 9 and bx - 9 < x0 + w and y0 < by and by - 40 < y0 + h - 6


def plant(cands, map_id, W, H, seed=1, protect=True, extra=(), fill_bush=True):
    """cands: (u, v, species) where the source drew a tree. Returns the (prop, u, v) to put on the map. protect:
    keep the whole playing area clear (off for a grove, whose trees stand in it; extra then names its open spots)."""
    meta = json.load(open(TILES))['sprites']
    sizes = {n: (*meta[n]['size'], *meta[n]['pivot']) for names in SPECIES.values() for n in names if n in meta}
    spots = scene_spots(map_id) + (stage_spots(W, H) if protect else []) + list(extra)
    rnd = random.Random(f'{map_id}-{seed}')
    kept = []
    # the trees a map was made around (a garden's plums, a grove's peaches, a village's pines) take their places first
    for u, v, kind in sorted(cands, key=lambda c: (c[2] not in FIRST, round(c[0] + c[1], 3), c[0])):
        name = SPECIES[kind][rnd.randrange(len(SPECIES[kind]))]
        if name not in sizes:
            continue
        gap = GAP.get(kind, 1.5)
        if any(math.hypot(u - a, v - b) < max(gap, GAP.get(k, 1.5)) for _, a, b, k in kept):
            continue
        near = any(math.hypot(u - a, v - b) < REACH.get(kind, 1.2) for a, b in spots)
        if near or any(hides(name, u, v, a, b, sizes) for a, b in spots):
            if not fill_bush or kind == 'bush':
                continue
            name, kind = SPECIES['bush'][rnd.randrange(2)], 'bush'  # a low bush keeps the place green
            if any(math.hypot(u - a, v - b) < REACH['bush'] for a, b in spots) or \
                    any(hides(name, u, v, a, b, sizes) for a, b in spots) or \
                    any(math.hypot(u - a, v - b) < GAP['bush'] for _, a, b, _ in kept):
                continue
        kept.append((name, round(u, 2), round(v, 2), kind))
    return [(n, u, v) for n, u, v, _ in kept]
