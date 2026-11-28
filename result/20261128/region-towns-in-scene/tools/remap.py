"""현재 장면 맵의 집·관아를 지역 건물로 바꾼다. 옛 건물 자리에서 가장 가까운 빈자리(길·물·나무·다른 건물과 안 겹침,
둘레 한 칸 띄움)를 찾아 놓고, 그 자리에 걸린 작은 소품은 뺀다.
  python3 remap.py ROOT
"""
import json, math, os, sys
ROOT = sys.argv[1]
G = os.path.join(ROOT, 'assets', 'graphics')
SRC = '/Users/younwookim/orca/mhdemo1/ver5/assets/graphics/maps'
scene = json.load(open(os.path.join(G, 'scene.json')))
FP = scene['footprints']
REG = {}
for r in os.listdir(os.path.join(os.path.dirname(os.path.abspath(__file__)), 'reg')):
    p = os.path.join(os.path.dirname(os.path.abspath(__file__)), 'reg', r, 'meta.json')
    if os.path.exists(p):
        REG[r] = json.load(open(p))
        lp = os.path.join(os.path.dirname(os.path.abspath(__file__)), 'lines', r, 'lines.json')
        if os.path.exists(lp):
            REG[r].update({k[len(r) + 1:]: v for k, v in json.load(open(lp)).items()})

OLD = ('house_thatch', 'house_thatch_x', 'house_tile', 'house_tile_x', 'tavern', 'hall_u', 'hall_v')
SMALL = ('wine_jar', 'grain_sack', 'water_jar', 'bench', 'table_wine', 'straw_mat', 'lantern', 'flagstones_0',
         'flagstones_1', 'weapon_rack', 'armor_stand', 'notice_post', 'notice_board', 'stree_bush_0', 'stree_bush_1',
         'water_bucket', 'grain_cart', 'grain_cart_v', 'brazier', 'command_flag', 'taihu_rock')

TARGETS = [
    # (새 맵 id, 이름, 원본, 지역, 막는 땅 글자, 풀 색조, 범위 [u0,u1,v0,v1])
    ('S_zhuo', '탁현(화북)', 'S_zhuo', 'huabei', 'di~', {'hue': -24, 'sat': .62, 'val': 1.0}, (0, 36, 0, 32)),
    ('S_luoyang', '낙양(중원)', 'S_luoyang', 'zhongyuan', 'i~', None, None),
    ('S_xuzhou_town', '서주(회남)', 'S_xuzhou_town', 'huainan', 'i~', None, (0, 59, 0, 51)),
    ('S_xiangyang', '양양(회남)', 'S_xiangyang', 'huainan', 'i~', None, (0, 48, 0, 40)),
    ('S_demo_jiangnan', '강남(양양 지형)', 'S_xiangyang', 'jiangnan', 'i~', {'hue': 14, 'sat': 1.3, 'val': .86}, (0, 48, 0, 40)),
    ('S_demo_bashu', '파촉(탁현 지형)', 'S_zhuo', 'bashu', 'di~', {'hue': 20, 'sat': 1.35, 'val': .78}, (0, 36, 0, 32)),
]
ZHUO_FILL = [('cottage_1', 3, 22), ('cottage_4', 6, 27), ('field', 1, 28), ('cottage_3', 16, 22), ('cottage_2', 22, 24),
             ('field', 28, 22), ('cottage_1', 30, 27), ('field', 15, 27)]
FILL = {'S_zhuo': ZHUO_FILL, 'S_demo_bashu': ZHUO_FILL,
        'S_luoyang': [('cottage_2', 50, 40), ('cottage_4', 44, 44)],
        'S_xuzhou_town': [('field', 40, 36), ('field', 46, 36), ('cottage_4', 52, 40), ('cottage_1', 30, 40)],
        'S_xiangyang': [('field', 2, 30), ('cottage_3', 40, 34)],
        'S_demo_jiangnan': [('field', 2, 30), ('cottage_3', 40, 34)]}


def foot(name, u, v, dx=0, dy=0):
    """객체가 덮는 소타일들"""
    f = FP.get(name)
    cu = u + .5 + (dx / 16 + dy / 8) / 2
    cv = v + .5 + (dy / 8 - dx / 16) / 2
    if not f:
        return {(u, v)}
    return {(a, b) for a in range(math.floor(cu + f[0] + 1e-6), math.ceil(cu + f[0] + f[2] - 1e-6))
            for b in range(math.floor(cv + f[1] + 1e-6), math.ceil(cv + f[1] + f[3] - 1e-6))}


def place_obj(name, bu, bv, fu, fv):
    cu, cv = bu + fu / 2, bv + fv / 2
    U, V = math.floor(cu), math.floor(cv)
    du, dv = cu - U - .5, cv - V - .5
    return [name, U, V, round(16 * (du - dv)), round(8 * (du + dv))]


def find(u0, v0, fu, fv, pu0, pv0, pu1, pv1, bounds, hard, taken, wall, glyph, blocked):
    """(u0, v0) 둘레로 넓혀 가며 첫 빈자리: 마당 칸이 막힌 칸·막는 땅에 안 걸리고, 둘레 한 칸이 다른 건물·줄과 안 닿음"""
    for r in range(0, 45):
        for bu in range(u0 - r, u0 + r + 1):
            for bv in range(v0 - r, v0 + r + 1):
                if max(abs(bu - u0), abs(bv - v0)) != r:
                    continue
                a0, b0, a1, b1 = bu + pu0, bv + pv0, bu + fu + pu1, bv + fv + pv1
                if a0 < bounds[0] or b0 < bounds[2] or a1 > bounds[1] or b1 > bounds[3]:
                    continue
                tiles = {(a, b) for a in range(a0, a1) for b in range(b0, b1)}
                ring = {(a, b) for a in range(a0 - 1, a1 + 1) for b in range(b0 - 1, b1 + 1)}
                if tiles & hard or ring & (taken | wall) or any(glyph(a, b) in blocked for a, b in tiles):
                    continue
                return bu, bv, tiles
    return None


def convert(new_id, title, src, region, blocked, grass, bounds):
    d = json.load(open(os.path.join(SRC, src + '.json')))
    ox, oy = d['origin']
    ground = d['ground']

    def glyph(u, v):
        r, c = v - oy, u - ox
        return ground[r][c] if 0 <= r < len(ground) and 0 <= c < len(ground[0]) else '#'

    wall = set()
    for w in d.get('walls', []):
        (a0, b0), (a1, b1) = w[1], w[2]
        for a in range(min(a0, a1), max(a0, a1) + 1):
            for b in range(min(b0, b1), max(b0, b1) + 1):
                wall.add((a, b))
    olds = [o for o in d['objects'] if o[0] in OLD]
    keep = [o for o in d['objects'] if o[0] not in OLD]
    hard = set(wall)
    for o in keep:
        if o[0] not in SMALL:
            hard |= foot(o[0], *o[1:3], *(o[3:5] if len(o) >= 5 and not isinstance(o[3], dict) else (0, 0)))
    if bounds is None:
        bounds = (ox, ox + len(ground[0]), oy, oy + len(ground))
    # 옛 건물 → 새 건물
    cot = [f'cottage_{k}' for k in (1, 2, 3, 4)]
    big = ['mansion_1', 'mansion_2', 'mansion_4', 'mansion_3']
    plan, nc, nm, extra = [], 0, 0, ['shop', 'smithy']
    halls = [o for o in olds if o[0].startswith('hall')]
    for i, o in enumerate(sorted(olds, key=lambda o: (not o[0].startswith('hall'), o[2], o[1]))):
        n = o[0]
        if n.startswith('hall'):
            new = 'yamen' if o is halls[0] else big[nm % 4]; nm += (o is not halls[0])
        elif n == 'tavern':
            new = 'tavern'
        elif n.startswith('house_tile') and extra:
            new = extra.pop(0)
        elif n.startswith('house_tile') and (i % 3 == 0):
            new = big[nm % 4]; nm += 1
        else:
            new = cot[nc % 4]; nc += 1
        plan.append((new, o))
    if 'tavern' not in [p[0] for p in plan]:
        plan.insert(0, ('tavern', plan[-1][1]))
    plan.sort(key=lambda p: -REG[region][p[0]]['footprint'][0] * REG[region][p[0]]['footprint'][1])
    fields = [k for k in ('field_dry', 'field_paddy') if k in REG[region]]
    for k, (n, u, v) in enumerate(FILL.get(new_id, [])):
        plan.append((fields[k % len(fields)] if n == 'field' else n, [n, u, v]))
    taken, out, small_hit, lines = set(), [], set(), []
    nfolk = 0
    for new, o in plan:
        fu, fv = REG[region][new]['footprint']
        if new == 'yamen' or new.startswith('mansion'):
            encl = 'twall'
        elif new.startswith('cottage'):
            encl = ('ewall', 'fence', None)[nfolk % 3]; nfolk += 1
        else:
            encl = None
        # 마당까지 친 자리: 집 앞(+v)으로 2칸, 옆·뒤로 1칸, 그 테두리에 담장. 자리가 없으면 담장 없이
        u0, v0 = o[1] - fu // 2, o[2] - fv // 2
        best = None
        for encl in ([encl, None] if encl else [None]):
            pu0, pv0, pu1, pv1 = (-1, -1, 1, 3) if encl else (0, 0, 0, 0)
            best = find(u0, v0, fu, fv, pu0, pv0, pu1, pv1, bounds, hard, taken, wall, glyph, blocked)
            if best:
                break
        if not best:
            print('  자리 없음:', new, o[:3])
            continue
        bu, bv, tiles = best
        taken |= tiles
        out.append(place_obj(f'{region}_{new}', bu, bv, fu, fv))
        if encl:
            fam = f'{region}_{encl}'
            L, R, B, F = bu - 1, bu + fu, bv - 1, bv + fv + 1
            lines += [[fam, [L, B], [R, B]], [fam, [L, B + 1], [L, F]], [fam, [R, B + 1], [R, F]]]
            gw = 1 if encl == 'fence' else 2
            g0 = bu + (fu - gw) // 2
            if g0 - 1 >= bu:
                lines.append([fam, [bu, F], [g0 - 1, F]])
            if g0 + gw <= R - 1:
                lines.append([fam, [g0 + gw, F], [R - 1, F]])
            if encl != 'fence':
                gate = 'tgate' if encl == 'twall' else 'egate'
                out.append(place_obj(f'{region}_{gate}_u', g0, F, 2, 1))
    kept = []
    for o in keep:
        if o[0] in SMALL and foot(o[0], o[1], o[2]) & taken:
            small_hit.add(o[0])
            continue
        kept.append(o)
    d['objects'] = out + kept
    d['walls'] = d.get('walls', []) + lines
    d['id'], d['name'] = new_id, title
    if grass is not None:
        d['grass'] = grass
    json.dump(d, open(os.path.join(G, 'maps', new_id + '.json'), 'w'), ensure_ascii=False, indent=1)
    print(new_id, title, len(olds), '->', len(out), '뺀 소품', sorted(small_hit))


for t in TARGETS:
    convert(*t)
