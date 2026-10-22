#!/usr/bin/env python3
"""Non-combat (<id>_noncombat) dot sheets from the 3D civilian models of tools/unit3d (scripts DO generate the pixels).

  python3 civ3d.py caocao sunjian ...        # build units/<key>_noncombat.yaml; --list shows the keys
  python3 civ3d.py --list
  python3 civ3d.py --poses liubei guanyu     # add the scene poses (sit …) to existing sheets, other rows untouched

People and poses beyond standing (officials, eunuch, wounded, helper pair, stretcher, prone crossbowman; sit,
throne, bow, prostrate, lie, sitground) come from tools/unit3d/scene_people.py.

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
import scene_people as SP  # noqa: E402

W, H = 56, 46
LEATHER = U.BROWN
HAIR = [f'hair{i + 1}={c}' for i, c in enumerate(['#5a4638', '#352920', '#1c1614'])]  # base sheet hair is maroon


def scene(key, n, head, **kit):
    walk = C.walker(key, n, **kit)
    build = SP.posed(walk, walk.kit)  # the civilian rows, then the scene poses (sit …)
    build.kit = walk.kit
    return build, head


def chosen(key, look):
    """The user's civilian pick (civilian.LOOKS / SELECTED) with the scene poses."""
    _, walk, head = C.LOOKS[look][C.SELECTED[look]]
    kit = walk.kit
    if key in SP.POSE_KITS:  # the sheet colours the robe in other tags than the civilian kit
        kit = f'{kit}_pose'
        U.KITS[kit] = dict(U.KITS[walk.kit], **SP.POSE_KITS[key])
    build = SP.posed(walk, kit)
    build.kit = kit
    return build, head


LOOKS = {
    'caocao': chosen('caocao', 'caocao3'),
    'sunjian': scene('sunjian', 1, Hd.heads(L.bearded(Hd.TOPKNOT, 'bristle'), Hd.BAND), cloth=U.BLUE, guard=LEATHER, tabard=LEATHER),
    'yuanshao': scene('yuanshao', 9, Hd.heads(L.bearded(L.GOLD_IRON, 'goatee'), Hd.PLUME), cloth=U.BLUE, robe=U.BLUE, guard=U.GOLD, tabard=U.GOLD),
    'zhaoyun': scene('zhaoyun', 9, Hd.heads(Hd.IRON), cloth=U.BLUE, guard=C.STEEL),
    'dongzhuo': scene('dongzhuo', 9, Hd.heads(L.bearded(L.CAP, 'bushy')), cloth=LEATHER, robe=LEATHER, guard=U.GOLD, tabard=U.GOLD, belly=True),
    'infantry': scene('infantry', 1, Hd.heads(L.CAP), robe=U.BLUE, guard=U.BLUE),
}
for _key in ('yuanshu', 'gongsunzan', 'liubei', 'guanyu', 'zhangfei'):  # the user's civilian picks (civilian.SELECTED)
    LOOKS[_key] = chosen(_key, _key)
LOOKS.update(SP.LOOKS)
BATTLE = {'caocao': 'caocao', 'sunjian': 'sunjian', 'yuanshao': 'yuanshao', 'zhaoyun': 'zhaoyun', 'dongzhuo': 'light_cavalry', 'infantry': 'light_infantry',
          'yuanshu': 'yuanshu', 'gongsunzan': 'light_cavalry', 'liubei': 'liubei', 'guanyu': 'guanyu', 'zhangfei': 'zhangfei'}
BATTLE.update({k: 'light_infantry' for k in SP.LOOKS})
NAME = {'caocao': '조조', 'sunjian': '손견', 'yuanshao': '원소', 'zhaoyun': '조운', 'dongzhuo': '동탁', 'infantry': '일반보병',
        'yuanshu': '원술', 'gongsunzan': '공손찬', 'liubei': '유비', 'guanyu': '관우', 'zhangfei': '장비', **SP.NAMES}
EXTRA_COLORS = {'dongzhuo': HAIR + ['themeA1=#7a5a9a', 'themeA2=#523470', 'themeA3=#2e1a44'], 'infantry': HAIR,
                **{k: HAIR for k in SP.LOOKS}}
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


def head_box(rows):
    """(top row, centre column) of a frame's head: the first 4 rows of drawn pixels."""
    top = next(y for y, r in enumerate(rows) if any(c != '.' for c in r))
    xs = [x for r in rows[top:top + 4] for x, c in enumerate(r) if c != '.']
    return top, round(sum(xs) / len(xs))


def sheet_head(uid, d, out, under):
    """Put the sheet's own idle head (12 rows, 15 columns) over a pose frame's rendered head. under: the frame drawn
    without its head, so the rendered head's pixels (and only those) go back to whatever lies behind them, and the
    new head lands on the old one's top even when a chair back rises above it."""
    tags = dots.frame(dots.load(uid), 'idle', d, 0)['tags']
    idle = [''.join(dots.TAG_CHARS[t - 1] if t else '.' for t in r) for r in tags]
    spots = [(y, x) for y in range(H) for x in range(W) if out[y][x] != under[y][x]]
    if not spots:  # hidden behind something
        return
    ty, cx = head_box(idle)
    ny = min(y for y, _ in spots)
    top = [x for y, x in spots if y < ny + 4]
    nx = round(sum(top) / len(top))
    for y, x in spots:
        out[y][x] = under[y][x]
    for y in range(12):
        for x in range(-7, 8):
            c = idle[ty + y][cx + x] if 0 <= ty + y < len(idle) and 0 <= cx + x < len(idle[0]) else '.'
            if c != '.' and 0 <= ny + y < H and 0 <= nx + x < W:
                out[ny + y][nx + x] = c


def to_tags(full, body, dx, dy, colors, unmapped, posed_unit):
    """A rendered frame as tag characters, moved by (dx, dy): head pixels (where it differs from the headless body
    render) by HEADMAP, the rest by BODY. posed_unit: that unit's pose key tags come first."""
    out = [['.']*W for _ in range(H)]
    for y, row in enumerate(full):
        for x, c in enumerate(row):
            if c == '.':
                continue
            table = BODY if body[y][x] == c else HEADMAP
            tag = SP.POSE_KEY_TAGS.get(posed_unit, {}).get(c) or table.get(c)
            if tag is None:
                tag = unmapped[c, table is HEADMAP] = nearest(c, colors)
            if 0 <= x + dx < W and 0 <= y + dy < H:
                out[y + dy][x + dx] = tag
    return out


def make(unit, only_poses=False):
    """The whole sheet from the 3D look, or with only_poses just the scene poses added to an existing sheet."""
    uid = f'{unit}_noncombat'
    build, head = LOOKS[unit]
    poses = SP.SHEET_POSES.get(unit) or SP.OFFICER_POSES.get(unit, [])
    if not only_poses:
        grid('new', 'light_infantry', uid, NAME[unit], '--noncombat')
        doc = dots.with_alpha(dots.load(uid))
        doc['tag_colors'] = list(dots.load(BATTLE[unit])['tag_colors'])
        dots.save(doc)
        if unit in EXTRA_COLORS:
            grid('colors', uid, *EXTRA_COLORS[unit])
    if poses:
        grid('poses', uid, *poses)
    if unit in SP.POSE_COLORS:
        grid('colors', uid, *SP.POSE_COLORS[unit])
    if only_poses:
        for pose in poses:
            grid('export', uid, '--anim', pose, '--force')
    else:
        grid('export', uid, '--force')
    colors = dots.load(uid)['tag_colors']
    rows = getattr(build, 'rows', C.ANIMATIONS)
    draw = getattr(build, 'frame', B.frame)
    frames = [] if only_poses else list(FRAMES)
    frames += [(pose, d, 0, pose, 0) for pose in poses for d in ('NW', 'SW')]
    blank = Hd.heads({d: ['.'*13]*12 for d in Hd.IRON})
    captured, to_img = [], Hd.to_img
    Hd.to_img = lambda g: (captured.append([r[:] for r in g]), to_img(g))[1]
    unmapped = {}

    def render(anim, d, n, heads):
        captured.clear()
        draw(build, heads, B.SPEC, d, rows.index(anim), n, lift=0)
        return captured[-1]

    ground = {d: max(y for y, r in enumerate(render('idle', d, 0, head)) for c in r if c != '.') for d in ('NW', 'SW')}
    for anim, d, n, out_anim, out_n in frames:
        own = SP.turned(head) if anim in SP.TURNED_HEAD else head
        full, body = render(anim, d, n, own), render(anim, d, n, blank)
        dx, dy = 28 - B.SPEC['pivot'][0], 38 - ground[d]
        out, under = to_tags(full, body, dx, dy, colors, unmapped, unit if out_anim in poses else None), None
        if out_anim in poses and unit in SP.POSE_SHEET_HEAD:
            under = to_tags(body, body, dx, dy, colors, unmapped, unit)
            sheet_head(uid, d, out, under)
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
    only = '--poses' in sys.argv
    for key in [a for a in sys.argv[1:] if not a.startswith('--')]:
        make(key, only_poses=only)
