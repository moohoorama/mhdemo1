#!/usr/bin/env python3
"""Check scene placement in ver5/assets/scenes/*.json: actors too close, off the floor, on props, outside the map.

  python3 .claude/skills/scene-staging/check_scenes.py [scene-or-node-id-prefix ...]

Plays each scene's beats in order (enter, move, exit) and checks the stage after every group: everyone on the
stage stands on open floor, away from standing props, and at least MIN_GAP cells from everyone else. Prints one
line per problem and a count; exits 1 when there are problems.
"""
import json
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parents[3] / 'ver5'
FREE_GLYPHS = '.diTWPC'  # ground, paving and the format 2 floors (assets/graphics/scene.json)
NOT_STANDING = ('floor', 'avenue', 'ash', 'carpet', 'wall', 'curtain', 'bridge', 'dais', 'throne', 'straw_mat',
                'bloody_cloth', 'broken_spear', 'fallen_flag')  # under, behind or seating the feet
MIN_GAP = 1.0   # cells between two actors' feet; closer than this the iso sprites overlap
PROP_GAP = 0.45  # cells from a standing prop's anchor


def load_map(cache, map_id):
    if map_id not in cache:
        art = json.load(open(ROOT / 'assets/graphics/maps' / f'{map_id}.json'))
        if art.get('format') == 2:  # objects stand dx, dy canvas pixels from their small tile's centre
            ground = art['ground']
            art = {'visual': [*art['origin'], len(ground[0]), len(ground)], 'small_tiles': ground,
                   'props': [[n, (U - .5) / 2 + dx / 64 + dy / 32, (V - .5) / 2 - dx / 64 + dy / 32]
                             for n, U, V, *d in art['objects'] for dx, dy in [d or (0, 0)]]}
        props = [(p[0], p[1], p[2]) for p in art.get('props', []) if len(p) == 3 and not p[0].startswith(NOT_STANDING)]
        cache[map_id] = (art, props)
    return cache[map_id]


def spot_problems(art, props, u, v):
    vx, vy, vw, vh = art['visual']
    su, sv = int(u * 2 // 1) - vx, int(v * 2 // 1) - vy
    if not (0 <= sv < vh and 0 <= su < vw):
        return ['outside the map picture']
    if art['small_tiles'][sv][su] not in FREE_GLYPHS:
        return ['not on open floor']
    for name, a, b in props:
        if ((u - a) ** 2 + (v - b) ** 2) ** .5 < PROP_GAP:
            return [f'on prop {name} ({a}, {b})']
    return []


def main():
    prefixes = sys.argv[1:]
    cache, problems = {}, []
    for path in sorted((ROOT / 'assets/scenes').glob('*.json')):
        for sid, sc in json.load(open(path))['Scenes'].items():
            if prefixes and not any(sid.startswith(p) or any(n.startswith(p) for n in sc['Nodes']) for p in prefixes):
                continue
            art, props = load_map(cache, sc['Map'])
            pos = {k: tuple(a['At']) for k, a in sc['Cast'].items() if not a.get('Off') and 'At' in a}  # Stay: from the walk
            seen = set()

            def check(when):
                names = sorted(pos)
                for n in names:
                    for p in spot_problems(art, props, *pos[n]):
                        if (n, p) not in seen:
                            seen.add((n, p))
                            problems.append(f'{sid} [{when}] {n} {pos[n]}: {p}')
                for i, a in enumerate(names):
                    for b in names[i + 1:]:
                        (ua, va), (ub, vb) = pos[a], pos[b]
                        d = ((ua - ub) ** 2 + (va - vb) ** 2) ** .5
                        if d < MIN_GAP and (a, b, pos[a], pos[b]) not in seen:
                            seen.add((a, b, pos[a], pos[b]))
                            problems.append(f'{sid} [{when}] {a}-{b}: {d:.2f} cells apart')

            check('cast')
            for node in sc['Nodes']:
                for li, line in enumerate(sc.get('Beats', {}).get(node, [])):
                    for gi, group in enumerate(line):
                        for st in group:
                            a = st.get('Actor')
                            if not a:
                                continue
                            if st.get('Enter'):
                                pos[a] = tuple(st['Enter'])
                            if st.get('Move'):
                                pos[a] = tuple(st['Move'])
                            if st.get('Exit'):
                                pos.pop(a, None)
                        check(f'{node} line {li} group {gi}')
    print('\n'.join(problems))
    print(f'{len(problems)} problems')
    sys.exit(1 if problems else 0)


if __name__ == '__main__':
    main()
