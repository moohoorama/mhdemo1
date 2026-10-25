#!/usr/bin/env python3
"""Story scenes on whole cells, drawn and edited as ASCII.

    python3 tools/scene_maps/scene_ascii.py whole [chNN ...]       # move every person onto whole cells
    python3 tools/scene_maps/scene_ascii.py dump [chNN ...]        # write assets/scenes/ascii/<scene>.txt
    python3 tools/scene_maps/scene_ascii.py load <scene> [...]     # read edited pictures back into the cast's spots

A picture is the scene's opening stage, one game cell to three characters, rows along v and columns along u, the
north corner at the top left:

    # c03-31  S_council  origin -1 0
    ###############################
    ###Yua...Tbl...Tbl...Tbl...###
    ###...Cao...Kon......Sun...###

`...` open floor, `###` wall or other ground nobody stands on, `~~~` water, `===` bridge, `^^^` cliff, blank past
the map's picture. Title-case codes are named officers (`Cao` = 조조), lower-case ones extras (`in1` = 병사#1);
the legend under the picture says who each is. Map objects show as title-case codes too (`Tbl` table, `Brz`
brazier, `Tnt` tent); they belong to the map and are not moved by `load`. Off-stage cast and the walks of each
line are listed as comments. `load` sets each listed person's `At` to the cell their code stands on.

`whole` rounds every person's spot (cast, entrances, walks, patrols) and the camera to the nearest whole cell;
when that cell is not open floor or someone already stands there it takes the nearest free whole cell. Walks that
leave the stage and entrances that walk on are only rounded. Effects and scene props stay where they are: they
sit on map objects (a brazier's fire, a table).
"""
import json
import math
import re
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parents[2]
SCENES = ROOT / 'assets/scenes'
ASCII = SCENES / 'ascii'
sys.path.insert(0, str(Path(__file__).resolve().parent))
from scene_format import write as write_formatted  # noqa: E402

FREE = '.diTWPC'
NOT_STANDING = ('floor', 'avenue', 'ash', 'carpet', 'wall', 'curtain', 'bridge', 'dais', 'throne', 'straw_mat',
                'bloody_cloth', 'broken_spear', 'fallen_flag')  # as check_scenes.py: under, behind or seating the feet
WALLISH = ('wall', 'swall', 'sgate', 'gate', 'sfence', 'screen', 'curtain')
PROP_GAP = .45
OBJECT = {'table_wine': 'Tbl', 'desk': 'Dsk', 'map_table': 'Map', 'bench': 'Bch', 'bed': 'Bed', 'throne_u': 'Thr',
          'dais': 'Dai', 'brazier': 'Brz', 'campfire': 'Fir', 'candle_stand': 'Cdl', 'lantern': 'Lnt',
          'banner': 'Bnr', 'banner_yellow': 'Bnr', 'command_flag': 'Flg', 'command_flag_yellow': 'Flg',
          'tent': 'Tnt', 'tent_round': 'Tnt', 'weapon_rack': 'Wpn', 'armor_stand': 'Arm', 'pillar_red': 'Pil',
          'pillar_wood': 'Pil', 'well': 'Wel', 'water_jar': 'Jar', 'wine_jar': 'Jar', 'water_bucket': 'Bkt',
          'grain_sack': 'Sck', 'cauldron': 'Cld', 'soup_stove': 'Stv', 'counter': 'Ctr', 'notice_board': 'Ntc',
          'notice_post': 'Ntc', 'tavern': 'Tav', 'burnt_tavern': 'Tav', 'pavilion': 'Pav', 'watchtower': 'Twr',
          'shield_wall': 'Shd', 'antler_barrier': 'Abr', 'altar': 'Alt', 'medicine_pot': 'Pot', 'taihu_rock': 'Rck',
          'milestone': 'Mil', 'straw_mat': 'Mat', 'peach_tree': 'Tre', 'stree_bush_0': 'Bsh', 'stree_bush_1': 'Bsh'}
OBJECT_PREFIX = (('house', 'Hse'), ('hall', 'Hal'), ('market_stall', 'Stl'), ('peach_bloom', 'Tre'),
                 ('stree', 'Tre'))
OFFICER = {'zhangfei': 'Zha', 'zhangliang': 'ZhL', 'zhangbao': 'ZhB', 'zhangshiping': 'ZSP', 'zhangjiao': 'ZhJ',
           'sunjian': 'Sun', 'sunce': 'SCe', 'xiahoudun': 'XHD', 'xiahouyuan': 'XHY', 'yuanshao': 'Yua',
           'yuanshu': 'YSh', 'huangfusong': 'HFS', 'huanggai': 'HGa', 'liubei': 'Liu', 'guanyu': 'Gua',
           'caocao': 'Cao', 'gongsunzan': 'GSZ', 'kongrong': 'Kon', 'taoqian': 'Tao', 'dongzhuo': 'Don',
           'lvbu': 'LvB', 'xiandi': 'Xia', 'luzhi': 'LuZ', 'zoujing': 'Zou'}
EXTRA = {'light_infantry': 'in', 'infantry_noncombat': 'in', 'light_cavalry': 'cv', 'light_cavalry_white': 'cv',
         'commoner_man': 'mn', 'commoner_woman': 'wm', 'commoner_elder': 'el', 'wounded_noncombat': 'wd',
         'eunuch_noncombat': 'eu', 'archer': 'ar', 'bandit': 'bd', 'stretcher_noncombat': 'st'}

whole = lambda x: float(x).is_integer()
rnd = lambda p: [math.floor(p[0] + .5), math.floor(p[1] + .5)]
cell_of = lambda u, v: (math.ceil(u - .5), math.ceil(v - .5))  # an object on a cell's edge (a wall at -.5) is outside


# --- maps ---------------------------------------------------------------------------------------------------

_maps = {}


def load_map(map_id):
    """The map's small-tile glyph lattice, its standing objects (name, u, v) and its wall cells."""
    if map_id in _maps:
        return _maps[map_id]
    art = json.load(open(ROOT / 'assets/graphics/maps' / f'{map_id}.json'))
    walls = set()
    if art.get('format') == 2:  # objects stand dx, dy canvas pixels from their small tile's centre
        ground = art['ground']
        ox, oy = art['origin']
        objects = [[n, (U - .5) / 2 + dx / 64 + dy / 32, (V - .5) / 2 - dx / 64 + dy / 32]
                   for n, U, V, *d in art['objects'] for dx, dy in [d or (0, 0)]]
        for _, (u0, v0), (u1, v1) in art.get('walls', []):
            for su in range(min(u0, u1), max(u0, u1) + 1):
                for sv in range(min(v0, v1), max(v0, v1) + 1):
                    walls.add((su // 2, sv // 2))
        m = {'visual': [ox, oy, len(ground[0]), len(ground)], 'small': ground, 'size': art['size']}
    else:
        objects = [p for p in art.get('props', []) if len(p) == 3]
        m = {'visual': art['visual'], 'small': art['small_tiles'], 'size': art['size']}
    m['objects'] = [(n, u, v) for n, u, v in objects]
    m['standing'] = [(n, u, v) for n, u, v in m['objects'] if not n.startswith(NOT_STANDING)]
    m['wall_lines'] = walls  # format 2 wall lines: nobody stands there
    m['walls'] = walls | {cell_of(u, v) for n, u, v in m['objects'] if n.startswith(WALLISH)}
    _maps[map_id] = m
    return m


def glyph(m, u, v):
    vx, vy, vw, vh = m['visual']
    su, sv = int(u * 2 // 1) - vx, int(v * 2 // 1) - vy
    if not (0 <= sv < vh and 0 <= su < vw):
        return None
    return m['small'][sv][su]


def open_cell(m, u, v):
    g = glyph(m, u, v)
    if g is None or g not in FREE or (u, v) in m['wall_lines']:
        return False
    return all(((u - a) ** 2 + (v - b) ** 2) ** .5 >= PROP_GAP for _, a, b in m['standing'])


# --- files --------------------------------------------------------------------------------------------------

def chapters(args):
    names = [a for a in args if re.fullmatch(r'ch\d\d', a)]
    return [p for p in sorted(SCENES.glob('ch*.json')) if not names or p.stem in names]


def save(path, book, text):
    """Writes a chapter in the layout it came in: scene_format's or plain indented JSON."""
    if json.dumps(json.loads(text), ensure_ascii=False, indent=1) + '\n' == text:
        path.write_text(json.dumps(book, ensure_ascii=False, indent=1) + '\n')
    else:
        write_formatted(path, book['Scenes'])


def steps(sc):
    for st in sc.get('Ambient', []):
        yield None, st
    for node in sc['Nodes']:
        for line in sc.get('Beats', {}).get(node, []):
            for group in line:
                for st in group:
                    yield node, st


# --- whole ----------------------------------------------------------------------------------------------------

def nearest(m, p, taken):
    cells = sorted(((u, v) for u in range(math.floor(p[0]) - 5, math.floor(p[0]) + 7)
                    for v in range(math.floor(p[1]) - 5, math.floor(p[1]) + 7)),
                   key=lambda c: ((c[0] - p[0]) ** 2 + (c[1] - p[1]) ** 2, c))
    for c in cells:
        if c not in taken and open_cell(m, *c):
            return list(c)
    return rnd(p)


def make_whole(sc):
    m = load_map(sc['Map'])
    pos, moved = {}, []

    def place(who, p, what):
        new = nearest(m, p, {tuple(q) for k, q in pos.items() if k != who})
        if new != rnd(p):
            moved.append(f'{who} {what} {p} -> {new}')
        return new

    for who, a in sc['Cast'].items():
        if 'At' not in a:
            continue
        if a.get('Off'):
            a['At'] = rnd(a['At'])
        else:
            a['At'] = place(who, a['At'], 'cast')
            pos[who] = tuple(a['At'])
    for _, st in steps(sc):
        who = st.get('Actor')
        if 'CamAt' in st:
            st['CamAt'] = rnd(st['CamAt'])
        if 'Enter' in st:
            if 'Move' in st:
                st['Enter'] = rnd(st['Enter'])
            else:
                st['Enter'] = place(who, st['Enter'], 'enter')
                pos[who] = tuple(st['Enter'])
        if 'Move' in st:
            if st.get('Exit'):
                st['Move'] = rnd(st['Move'])
            else:
                st['Move'] = place(who, st['Move'], 'move')
                pos[who] = tuple(st['Move'])
        if st.get('Exit'):
            pos.pop(who, None)
        if 'Patrol' in st:
            st['Patrol'] = place(who, st['Patrol'], 'patrol')
    return moved


def cmd_whole(args):
    for path in chapters(args):
        text = path.read_text()
        book = json.loads(text)
        for sid, sc in book['Scenes'].items():
            for line in make_whole(sc):
                print(f'{sid}: {line}')
        save(path, book, text)


# --- codes ----------------------------------------------------------------------------------------------------

def characters():
    out = {}
    for f in ('characters.json', 'characters_story.json'):
        for cid, c in json.load(open(ROOT / 'assets/data' / f))['Characters'].items():
            out.setdefault(c['Name'], cid)
    return out


def codes(sc, names):
    """A unique three-letter code for each cast member: officers title-case, extras lower-case."""
    out, used = {}, set()
    for key, a in sc['Cast'].items():
        base = re.split('[#@]', key)[0]
        cid = names.get(base)
        if cid and '#' not in key:
            want = [OFFICER.get(cid, cid[:3].title()), cid[0].upper() + cid[-2:]] + [cid[:2].title() + str(n) for n in range(1, 10)]
        else:
            stem = EXTRA.get(a.get('Sprite', ''), (a.get('Sprite') or cid or 'xx')[:2].lower())
            want = [f'{stem}{i}' for i in range(1, 10)] + [f'{stem[0]}{i:02d}' for i in range(10, 100)]
        code = next(c for c in want if c not in used)
        used.add(code)
        out[key] = code
    return out


def object_code(name):
    if name in OBJECT:
        return OBJECT[name]
    for prefix, code in OBJECT_PREFIX:
        if name.startswith(prefix):
            return code
    return None


# --- dump -----------------------------------------------------------------------------------------------------

def picture(sid, sc, chapter, names):
    m = load_map(sc['Map'])
    code = codes(sc, names)
    stage = {k: tuple(a['At']) for k, a in sc['Cast'].items() if 'At' in a and not a.get('Off')}
    for k, p in stage.items():
        if not (whole(p[0]) and whole(p[1])):
            raise SystemExit(f'{sid}: {k} stands at {p}, not on a whole cell (run whole first)')
    W, H = m['size']
    near = [c for c in m['walls'] if -1 <= c[0] <= W and -1 <= c[1] <= H]  # the room's own walls, not the town's
    us = [0, W - 1] + [int(p[0]) for p in stage.values()] + [c[0] for c in near]
    vs = [0, H - 1] + [int(p[1]) for p in stage.values()] + [c[1] for c in near]
    u0, u1, v0, v1 = min(us), max(us), min(vs), max(vs)
    things = {}
    for n, u, v in m['objects']:
        c = object_code(n)
        cell = cell_of(u, v)
        if c and (cell not in things or things[cell] in ('Tre', 'Bsh')):
            things[cell] = c
    at = {p: code[k] for k, p in stage.items()}
    lines = [f'# {sid}  {sc["Map"]}  {chapter}  origin {u0} {v0}']
    for v in range(v0, v1 + 1):
        row = ''
        for u in range(u0, u1 + 1):
            g = glyph(m, u, v)
            if (u, v) in at:
                row += at[(u, v)]
            elif (u, v) in things:
                row += things[(u, v)]
            elif (u, v) in m['walls'] or g == 'c':
                row += '###'
            elif g is None:
                row += '   '
            elif g == '~':
                row += '~~~'
            elif g == 'b':
                row += '==='
            elif g == 'h':
                row += '^^^'
            elif g in FREE:
                row += '...'
            else:
                row += '###'
        lines.append(row.rstrip())
    lines.append('')
    for k, a in sc['Cast'].items():
        extra = []
        if a.get('Sprite'):
            extra.append(a['Sprite'])
        if a.get('Pose'):
            extra.append(a['Pose'])
        if a.get('Off'):
            extra.append(f'off stage, enters later (cell {a["At"][0]} {a["At"][1]})' if 'At' in a else 'off stage')
        if 'At' not in a:
            extra.append('stands where the walk left them')
        lines.append(f'{code[k]} {k}' + (f'  ({", ".join(extra)})' if extra else ''))
    walks = []
    for node, st in steps(sc):
        who = st.get('Actor')
        if who and ('Move' in st or 'Enter' in st):
            p = st.get('Move') or st.get('Enter')
            how = ' and leaves' if st.get('Exit') else ''
            walks.append(f'# {node or "opening"}: {code.get(who, who)} -> {p[0]} {p[1]}{how}')
    if walks:
        lines.append('')
        lines += walks
    return '\n'.join(lines) + '\n'


def cmd_dump(args):
    names = characters()
    ASCII.mkdir(exist_ok=True)
    for path in chapters(args):
        for sid, sc in json.loads(path.read_text())['Scenes'].items():
            (ASCII / f'{sid}.txt').write_text(picture(sid, sc, path.stem, names))
            print(sid)


# --- load -----------------------------------------------------------------------------------------------------

def cmd_load(args):
    names = characters()
    wanted = set(args)
    for path in chapters([]):
        text = path.read_text()
        book = json.loads(text)
        changed = False
        for sid, sc in book['Scenes'].items():
            if sid not in wanted:
                continue
            pic = (ASCII / f'{sid}.txt').read_text().split('\n')
            head = re.search(r'origin (-?\d+) (-?\d+)', pic[0])
            u0, v0 = int(head[1]), int(head[2])
            rows = pic[1:pic.index('')]
            code = codes(sc, names)
            spots = {}
            for r, row in enumerate(rows):
                for c in range(0, len(row), 3):
                    spots.setdefault(row[c:c + 3], []).append([u0 + c // 3, v0 + r])
            for key, a in sc['Cast'].items():
                if a.get('Off') or 'At' not in a:
                    continue
                found = spots.get(code[key], [])
                if len(found) != 1:
                    raise SystemExit(f'{sid}: {code[key]} ({key}) is in the picture {len(found)} times')
                if a['At'] != found[0]:
                    print(f'{sid}: {key} {a["At"]} -> {found[0]}')
                    a['At'], changed = found[0], True
            wanted.discard(sid)
        if changed:
            save(path, book, text)
    if wanted:
        raise SystemExit(f'no such scene: {", ".join(sorted(wanted))}')


if __name__ == '__main__':
    if len(sys.argv) < 2 or sys.argv[1] not in ('whole', 'dump', 'load'):
        raise SystemExit(__doc__)
    {'whole': cmd_whole, 'dump': cmd_dump, 'load': cmd_load}[sys.argv[1]](sys.argv[2:])
