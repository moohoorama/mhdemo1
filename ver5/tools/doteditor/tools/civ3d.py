#!/usr/bin/env python3
"""Non-combat (<id>_noncombat) dot sheets from the 3D civilian models of tools/unit3d (scripts DO generate the pixels).

  python3 civ3d.py caocao sunjian ...        # build units/<key>_noncombat.yaml; --list shows the keys
  python3 civ3d.py --list

Per unit: `grid.py new light_infantry <id> --noncombat`, the tag colours of the battle sheet (BATTLE) copied in, the
3D scene model rendered for NW/SW (idle, two walk steps, one action = the held salute), keys mapped to tags, the
grid written and `grid.py import` run. Head pixels (2D heads of tools/unit3d/heads.py) and body pixels are mapped
separately because hair and leather share render keys. The 3D walk swings the arms against the legs (civilian.walk_arms).
Add a look: a `scene(...)` entry in LOOKS (kit parts and head like tools/unit3d/civilian.py), its battle sheet in
BATTLE (whose tag colours it takes) and a NAME. Then `python3 ../spritebuild/build.py units/<id>_noncombat.yaml`
and set `Look.Scene` in assets/data/characters.json (docs/presentation-sprite.md). A mounted class without a scene
sprite of its own stands in `Scene` of its class (classes.json).
"""
import re
import subprocess
import sys
from pathlib import Path

HERE = Path(__file__).resolve().parent
DE = HERE.parent
sys.path.insert(0, str(DE.parents[2] / 'tools/unit3d'))
sys.path.append(str(HERE))  # after unit3d: both folders have a render.py
import build as B  # noqa: E402
import civilian as C  # noqa: E402
import dots  # noqa: E402
import heads as Hd  # noqa: E402
import looks as L  # noqa: E402
import render as R  # noqa: E402
import units as U  # noqa: E402

W, H = 56, 46
LEATHER = U.BROWN
HAIR = [f'hair{i + 1}={c}' for i, c in enumerate(['#5a4638', '#352920', '#1c1614'])]  # base sheet hair is maroon


def scene(key, n, head, **kit):
    walk = C.walker(key, n, **kit)
    build = C.scene_unit(walk, walk.kit)
    build.kit = walk.kit
    return build, head


def chosen(key, look):
    build, head = [e for e in C.chosen() if e[0] == key][0][2:]
    build.kit = C.LOOKS[look][C.SELECTED[look]][1].kit
    return build, head


LOOKS = {
    'caocao': chosen('caocao', 'caocao3'),
    'sunjian': scene('sunjian', 1, Hd.heads(L.bearded(Hd.TOPKNOT, 'bristle'), Hd.BAND), cloth=U.BLUE, guard=LEATHER, tabard=LEATHER),
    'yuanshao': scene('yuanshao', 9, Hd.heads(L.bearded(L.GOLD_IRON, 'goatee'), Hd.PLUME), cloth=U.BLUE, robe=U.BLUE, guard=U.GOLD, tabard=U.GOLD),
    'zhaoyun': scene('zhaoyun', 9, Hd.heads(Hd.IRON), cloth=U.BLUE, guard=C.STEEL),
    'dongzhuo': scene('dongzhuo', 9, Hd.heads(L.bearded(L.CAP, 'bushy')), cloth=LEATHER, robe=LEATHER, guard=U.GOLD, tabard=U.GOLD, belly=True),
    'infantry': scene('infantry', 1, Hd.heads(L.CAP), robe=U.BLUE, guard=U.BLUE),
}
BATTLE = {'caocao': 'caocao', 'sunjian': 'sunjian', 'yuanshao': 'yuanshao', 'zhaoyun': 'zhaoyun', 'dongzhuo': 'light_cavalry', 'infantry': 'light_infantry'}
NAME = {'caocao': '조조', 'sunjian': '손견', 'yuanshao': '원소', 'zhaoyun': '조운', 'dongzhuo': '동탁', 'infantry': '일반보병'}
EXTRA_COLORS = {'dongzhuo': HAIR + ['themeA1=#7a5a9a', 'themeA2=#523470', 'themeA3=#2e1a44'], 'infantry': HAIR}
# (3D animation, direction, 3D frame) -> (sheet animation, frame); walk steps are the two stride frames
FRAMES = [('idle', 'NW', 0, 'idle', 0), ('idle', 'SW', 0, 'idle', 0),
          ('walk', 'NW', 0, 'walk', 0), ('walk', 'NW', 2, 'walk', 1), ('walk', 'SW', 0, 'walk', 0), ('walk', 'SW', 2, 'walk', 1),
          ('salute', 'NW', 2, 'action', 0), ('salute', 'SW', 2, 'action', 0)]
BODY = {'0': '#', '1': 'c', '2': 'c', '3': 'b', '4': 'b', '5': 'a', '6': 'a', 'i': 'd', 'j': 'e', 'f': 'o', 'g': 'n', 'h': 'm',
        'k': 'u', 'l': 't', 'm': 's', 'W': 'l', 'X': 'l', 'Y': 'k', 'Z': 'j', 'A': 'r', 'B': 'q', 'C': 'p', 'b': 'r', 'c': 'q', 'd': 'p'}
HEADMAP = dict(BODY, f='i', g='h', h='g')  # head hair dark/mid/light -> hair3/2/1
LETTERS = 'abcdefghijklmnopqrstuvwxyzABCD#EFG'  # tag order of tag_colors


def nearest(key, colors):
    """Tag whose colour is closest to an unmapped render key (never faction, horse or effect tags); reported."""
    rgb = R.COLOR[key]
    return LETTERS[min((i for i in range(len(colors)) if LETTERS[i] in 'abcdefghimnopqrstu'),
                       key=lambda i: sum((a - int(colors[i][1 + 2*j:3 + 2*j], 16))**2 for j, a in enumerate(rgb)))]


def grid(*args):
    subprocess.run(['python3', 'grid.py', *args], cwd=HERE, check=True, capture_output=True)


def make(unit):
    uid = f'{unit}_noncombat'
    build, head = LOOKS[unit]
    grid('new', 'light_infantry', uid, NAME[unit], '--noncombat')
    doc = dots.with_alpha(dots.load(uid))
    doc['tag_colors'] = list(dots.load(BATTLE[unit])['tag_colors'])
    dots.save(doc)
    if unit in EXTRA_COLORS:
        grid('colors', uid, *EXTRA_COLORS[unit])
    grid('export', uid, '--force')
    colors = dots.load(uid)['tag_colors']
    blank = Hd.heads({d: ['.'*13]*12 for d in Hd.IRON})
    captured, to_img = [], Hd.to_img
    Hd.to_img = lambda g: (captured.append([r[:] for r in g]), to_img(g))[1]
    unmapped = {}

    def render(anim, d, n, heads):
        captured.clear()
        B.frame(build, heads, B.SPEC, d, C.ANIMATIONS.index(anim), n, lift=0)
        return captured[-1]

    ground = {d: max(y for y, r in enumerate(render('idle', d, 0, head)) for c in r if c != '.') for d in ('NW', 'SW')}
    for anim, d, n, out_anim, out_n in FRAMES:
        full, body = render(anim, d, n, head), render(anim, d, n, blank)
        dx, dy = 28 - B.SPEC['pivot'][0], 38 - ground[d]
        out = [['.']*W for _ in range(H)]
        for y, row in enumerate(full):
            for x, c in enumerate(row):
                if c == '.':
                    continue
                table = BODY if body[y][x] == c else HEADMAP
                tag = table.get(c)
                if tag is None:
                    tag = unmapped[c, table is HEADMAP] = nearest(c, colors)
                if 0 <= x + dx < W and 0 <= y + dy < H:
                    out[y + dy][x + dx] = tag
        path = DE / f'heroes/{uid}/grid/{out_anim}/{d}-{out_n}.txt'
        lines = path.read_text().split('\n')
        for i, line in enumerate(lines):
            if re.match(r'^\d\d\|', line):
                lines[i] = line[:3] + ''.join(out[int(line[:2])])
        path.write_text('\n'.join(lines))
    Hd.to_img = to_img
    grid('import', uid)
    print(unit, 'done', f'(unmapped render keys, nearest tag used: {unmapped})' if unmapped else '', flush=True)


if __name__ == '__main__':
    if '--list' in sys.argv or len(sys.argv) < 2:
        sys.exit('keys: ' + ' '.join(LOOKS))
    for key in sys.argv[1:]:
        make(key)
