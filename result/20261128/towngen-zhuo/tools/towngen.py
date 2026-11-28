"""규칙으로 성·마을 장면 맵을 다시 짠다 (미리보기 루트에만 쓴다). 맵 하나씩.

1. 지킬 것: 성벽·성문, 탐험(소타일)·장면(게임 칸 → 소타일, 걸음 선분 포함)의 사람·출구·사인·시작 칸,
   사인이 가리키는 게시판·우물. 역할 건물: 출구가 잇는 실내 맵 → 관청·주점·도구점·저택, 대장간 출구 → 바깥 대장간.
   ROOT/roles/<맵>.json 에 남긴다.
2. 성 안 범위: 성벽 안쪽(성벽이 없는 쪽은 판 끝 2칸 안). 건물은 반드시 그 안에.
3. 바닥은 새로: 성문에서 직선 대로(포석 2칸), 그것과 교차하는 대로, 블록 사이 흙 골목. 옛 바닥 글자는 지운다.
4. 블록마다 구역: 관아(대로변 뒤쪽) · 상업(성문·교차로 가까운 대로변) · 부유층(뒤쪽 먼 곳) · 민가(나머지) · 밭(앞 가장자리).
   앞쪽 성벽(+u·+v)이 있으면 그 옆 블록에 중요 건물 금지. 황궁이 있으면 둘레를 비운다.
5. 블록 채우기: 건물은 길을 향한다 (+v 쪽 길: 원래 건물, +u 쪽 길: 좌우 반전 _r). 초가는 대부분 울 없이, 울타리는 한둘.
6. 검사: 지킬 칸이 안 덮이고, 출구 문 칸이 놀이 영역 안이고, 역할 건물이 다 섰는지. 어기면 골목 위치를 흔들어 다시.

  python3 towngen.py ROOT MAP REGION [seed]
"""
import glob
import json
import math
import os
import random
import sys

HERE = os.path.dirname(os.path.abspath(__file__))
VER5 = '/Users/younwookim/orca/mhdemo1/ver5'
SRC_MAPS = os.path.join(VER5, 'assets/graphics/maps')
SRC_SCN = os.path.join(VER5, 'assets/scenario/hero')
SRC_SCENES = os.path.join(VER5, 'assets/scenes')

ROLE_BY_MAP = {'S_tool_shop': 'shop', 'S_tavern': 'tavern', 'S_zhuo_office': 'yamen', 'S_xiaopei_office': 'yamen',
               'S_council': 'yamen', 'S_liu_house': 'mansion', 'S_mansion': 'mansion', 'S_weapon_shop': 'smithy'}
ROLE_KO = {'yamen': '관청', 'tavern': '주점', 'shop': '도구점', 'smithy': '대장간', 'mansion': '저택', 'palace': '황궁'}
KEEP_INSIDE = ('notice_post', 'notice_board', 'well', 'sgate_open', 'sgate_closed', 'palace_gate', 'pavilion')
TREES = ('stree_willow', 'stree_broad_0', 'stree_broad_1', 'stree_bush_0', 'stree_bush_1', 'stree_pine_0', 'stree_pine_1',
         'stree_plum_0', 'stree_plum_1', 'stree_old')


def load_meta(region):
    meta = json.load(open(os.path.join(HERE, 'reg', region, 'meta.json')))
    meta.update({k[len(region) + 1:]: v for k, v in json.load(open(os.path.join(HERE, 'lines', region, 'lines.json'))).items()})
    return meta


def tiles_rect(u0, v0, w, h):
    return {(u, v) for u in range(u0, u0 + w) for v in range(v0, v0 + h)}


def place_obj(name, bu, bv, fu, fv):
    cu, cv = bu + fu / 2, bv + fv / 2
    U, V = math.floor(cu), math.floor(cv)
    du, dv = cu - U - .5, cv - V - .5
    return [name, U, V, round(16 * (du - dv)), round(8 * (du + dv))]


# ---------------------------------------------------------------- 1. 지킬 것

def gather(map_id):
    """이 맵을 쓰는 탐험·장면에서 지킬 칸, 출구·사인, 역할"""
    keep, exits, signs, roles, uses = set(), [], [], {}, []
    for f in sorted(glob.glob(os.path.join(SRC_SCN, 'explore*.json'))):
        areas = (json.load(open(f)).get('Areas') or {})
        for k, a in areas.items():
            if a.get('Map') != map_id:
                continue
            uses.append(f'{os.path.basename(f)}/{k}')
            if 'At' in a:
                keep.add(tuple(a['At']))
            for p in (a.get('People') or {}).values():
                if 'At' in p:
                    keep.add(tuple(p['At']))
            for e in a.get('Exits') or []:
                role = ROLE_BY_MAP.get((areas.get(e.get('To')) or {}).get('Map'))
                exits.append({'file': os.path.basename(f), 'area': k, 'to': e.get('To'), 'name': e.get('Name'),
                              'at': tuple(e['At']), 'role': role})
                if role:
                    roles.setdefault(role, []).append(e.get('Name'))
                else:
                    keep.add(tuple(e['At']))
            for s in a.get('Signs') or []:
                signs.append({'file': os.path.basename(f), 'area': k, 'name': s.get('Name'), 'at': tuple(s['At'])})
                keep.add(tuple(s['At']))
    for f in sorted(glob.glob(os.path.join(SRC_SCENES, 'ch*.json'))):
        d = json.load(open(f))
        for sid, sc in (d.get('Scenes') or {}).items():
            if sc.get('Map') != map_id:
                continue
            uses.append(f'{os.path.basename(f)}/{sid}')
            pts = []

            def walk(x):
                if isinstance(x, dict):
                    for key in ('At', 'Enter', 'Move', 'To'):
                        v = x.get(key)
                        if isinstance(v, list) and len(v) == 2 and all(isinstance(c, (int, float)) for c in v):
                            pts.append(v)
                    for key in ('Via',):
                        for v in x.get(key) or []:
                            if isinstance(v, list) and len(v) == 2:
                                pts.append(v)
                    for v in x.values():
                        walk(v)
                elif isinstance(x, list):
                    for v in x:
                        walk(v)
            walk(sc)
            for i, p in enumerate(pts):        # 장면은 게임 칸: 소타일 두 배, 이어진 점 사이 선분도
                keep |= {(math.floor(2 * p[0]) + du, math.floor(2 * p[1]) + dv) for du in (0, 1) for dv in (0, 1)}
                if i and abs(pts[i - 1][0] - p[0]) + abs(pts[i - 1][1] - p[1]) < 6:
                    a, b = pts[i - 1], p
                    n = int(max(abs(a[0] - b[0]), abs(a[1] - b[1])) * 4) + 1
                    for k in range(n + 1):
                        x, y = a[0] + (b[0] - a[0]) * k / n, a[1] + (b[1] - a[1]) * k / n
                        keep.add((math.floor(2 * x), math.floor(2 * y)))
    return keep, exits, signs, roles, uses


# ---------------------------------------------------------------- 2~6. 짜기

def generate(map_id, region, seed=0, tries=40):
    meta = load_meta(region)
    src = json.load(open(os.path.join(SRC_MAPS, map_id + '.json')))
    keep, exits, signs, roles, uses = gather(map_id)
    best = None
    for grow in range(4):           # 통과하는 판이 없으면 성을 키운다 (앞쪽으로 6칸씩)
        cur = grow_map(src, 6 * grow)
        for t in range(tries):      # 검사를 통과한 것 가운데 점수가 가장 좋은 것
            res = attempt(cur, meta, region, keep, exits, signs, roles, random.Random(seed * 1000 + t))
            res['grow'] = 6 * grow
            key = (res['ok'], res['score'])
            if best is None or key > (best['ok'], best['score']):
                best = res
        if best['ok']:
            break
    return best, keep, exits, signs, roles, uses


def grow_map(src, n):
    """앞쪽(+u·+v)으로 n칸 넓힌 맵: 땅을 늘리고, 성벽 줄은 앞쪽 끝을 늘리고, 앞쪽 성벽은 n칸 밀어낸다"""
    d = json.loads(json.dumps(src))
    if not n:
        return d
    ox, oy = d['origin']
    W, H = len(d['ground'][0]), len(d['ground'])
    d['ground'] = [r + '.' * n for r in d['ground']] + ['.' * (W + n)] * n
    play_u, play_v = 2 * d['size'][0], 2 * d['size'][1]
    for w in d.get('walls', []):
        (a0, b0), (a1, b1) = w[1], w[2]
        if a0 == a1 and a0 > play_u:          # 앞쪽 성벽(u 상수)은 밀어낸다
            w[1][0] += n; w[2][0] += n
        elif b0 == b1 and b0 > play_v:
            w[1][1] += n; w[2][1] += n
        if a0 == a1:                          # v를 따라 선 줄은 앞쪽 끝을 늘린다
            w[2][1] = max(w[1][1], w[2][1]) + n if w[2][1] >= w[1][1] else w[2][1]
        else:
            w[2][0] = max(w[1][0], w[2][0]) + n if w[2][0] >= w[1][0] else w[2][0]
    d['size'] = [d['size'][0] + n // 2, d['size'][1] + n // 2]   # 사람이 걷는 범위도 성과 함께
    for o in d['objects']:                    # 앞쪽 성벽 위 성문도 같이
        if o[0].startswith('sgate') and (o[1] > play_u or o[2] > play_v):
            if o[1] > play_u:
                o[1] += n
            else:
                o[2] += n
    return d


def attempt(src, meta, region, keep, exits, signs, roles, rnd):
    d = json.loads(json.dumps(src))
    ox, oy = d['origin']
    W, H = len(d['ground'][0]), len(d['ground'])
    play = (0, 2 * d['size'][0], 0, 2 * d['size'][1])
    walls = [w for w in d.get('walls', []) if w[0] == 'swall']
    wall = set()
    for w in walls:
        (a0, b0), (a1, b1) = w[1], w[2]
        wall |= {(a, b) for a in range(min(a0, a1), max(a0, a1) + 1) for b in range(min(b0, b1), max(b0, b1) + 1)}
    # 성 안 범위
    u_lo, u_hi, v_lo, v_hi = ox + 2, ox + W - 3, oy + 2, oy + H - 3
    front_u = front_v = False
    for w in walls:
        (a0, b0), (a1, b1) = w[1], w[2]
        if a0 == a1:                      # v를 따라 선 벽: u = a0
            if a0 <= play[0]:
                u_lo = max(u_lo, a0 + 1)
            else:
                u_hi = min(u_hi, a0)
                front_u = True
        else:
            if b0 <= play[2]:
                v_lo = max(v_lo, b0 + 1)
            else:
                v_hi = min(v_hi, b0)
                front_v = True
    gates = [o for o in d['objects'] if o[0].startswith('sgate')]
    palace = [o for o in d['objects'] if o[0].startswith(('palace', 'pwall'))]
    # 지킬 물건: 성 안의 게시판·우물·성문·황궁, 성 밖 것은 모두
    inside = lambda u, v: u_lo <= u < u_hi and v_lo <= v < v_hi
    kept = [o for o in d['objects'] if not inside(o[1], o[2]) or o[0].startswith(KEEP_INSIDE) or o[0].startswith(('pwall', 'palace'))]
    hard = set(wall) | {(o[1], o[2]) for o in kept if inside(o[1], o[2])}
    for o in kept:
        if o[0] == 'palace_gate':
            hard |= tiles_rect(o[1] - 5, o[2] - 3, 10, 6)
    for u, v in list(keep):
        hard |= {(u + a, v + b) for a in (-1, 0, 1) for b in (-1, 0, 1)}
    # 성벽에서 3칸 안에는 건물·담장·나무를 두지 않는다 (길은 된다)
    nobuild = {(a + du, b + dv) for a, b in wall for du in range(-3, 4) for dv in range(-3, 4)}
    road_end_u = u_hi if front_u else ox + W        # 성벽 없는 앞쪽은 길이 판 끝까지
    road_end_v = v_hi if front_v else oy + H
    ROAD = 'i' if palace else 'd'                   # 길은 한 종류: 황성은 포석, 고을은 흙길

    # 3. 길: 성문에서 직선 대로, 교차 대로, 골목
    roads = {}                     # 칸 -> 'A'(대로) / 'L'(골목), 바닥은 둘 다 ROAD
    main_axis, main_at = 'u', (u_lo + 4)
    if gates:
        g = gates[0]
        on_v_wall = any(w[1][0] == w[2][0] == g[1] for w in walls)
        if on_v_wall:              # 성문이 u = 상수 벽에: 대로는 u 방향, v = g, g+1
            main_axis, main_at = 'u', g[2]
        else:
            main_axis, main_at = 'v', g[1]
    elif palace:
        main_axis, main_at = 'v', [o for o in palace if o[0] == 'palace_gate'][0][1] if any(o[0] == 'palace_gate' for o in palace) else (u_lo + u_hi) // 2
    jitter = rnd.choice([0, 0, -1, 1, -2, 2])
    if main_axis == 'u':
        cross_at = max(u_lo + 6, min(u_hi - 8, (play[0] + play[1]) // 2 + jitter))
    else:
        cross_at = max(v_lo + 6, min(v_hi - 8, (play[2] + play[3]) // 2 + jitter))

    def lay(axis, at, width, glyph, lo=None, hi=None):
        if axis == 'u':
            for u in range(u_lo if lo is None else lo, road_end_u if hi is None else hi):
                for k in range(width):
                    roads.setdefault((u, at + k), glyph)
        else:
            for v in range(v_lo if lo is None else lo, road_end_v if hi is None else hi):
                for k in range(width):
                    roads.setdefault((at + k, v), glyph)
    other = 'v' if main_axis == 'u' else 'u'
    lay(main_axis, main_at, 2, 'A')
    lay(other, cross_at, 2, 'A')
    if gates:                                       # 성문 밖으로도 판 끝까지 곧게
        if main_axis == 'u':
            lay('u', main_at, 2, 'A', lo=ox, hi=u_lo)
        else:
            lay('v', main_at, 2, 'A', lo=oy, hi=v_lo)
    # 골목: 대로에서 블록 깊이(7~9칸)마다
    span = {'u': (u_lo, u_hi), 'v': (v_lo, v_hi)}

    def lanes(axis_pos_lo, axis_pos_hi, avenue, step):
        out = []
        p = avenue - step
        while p > axis_pos_lo + 3:
            out.append(p)
            p -= step
        p = avenue + 2 + step - 1
        while p < axis_pos_hi - 3:
            out.append(p)
            p += step
        return out
    step = rnd.choice([10, 11, 12])
    lane_main = lanes(*span['v' if main_axis == 'u' else 'u'], main_at, step)        # 대로와 나란한 골목의 위치
    lane_cross = lanes(*span['u' if main_axis == 'u' else 'v'], cross_at, step + rnd.choice([0, 1, 2]))
    for p in lane_main:
        lay(main_axis, p, 1, 'L')
    for p in lane_cross:
        lay(other, p, 1, 'L')

    # 블록: 길로 나뉜 직사각형
    def cuts(lo, hi, lines):
        xs = sorted(set(lines))
        out, a = [], lo
        for x, w in xs:
            if x > a:
                out.append((a, x))
            a = max(a, x + w)
        if hi > a:
            out.append((a, hi))
        return out
    if main_axis == 'u':
        rows = cuts(v_lo, v_hi, [(main_at, 2)] + [(p, 1) for p in lane_main])
        cols = cuts(u_lo, u_hi, [(cross_at, 2)] + [(p, 1) for p in lane_cross])
    else:
        cols = cuts(u_lo, u_hi, [(main_at, 2)] + [(p, 1) for p in lane_main])
        rows = cuts(v_lo, v_hi, [(cross_at, 2)] + [(p, 1) for p in lane_cross])
    blocks = []
    gpos = (gates[0][1], gates[0][2]) if gates else (u_lo, v_lo)
    cross_pt = (cross_at, main_at) if main_axis == 'u' else (main_at, cross_at)
    for (a0, a1) in cols:
        for (b0, b1) in rows:
            if a1 - a0 < 3 or b1 - b0 < 3:
                continue
            road_v = roads.get((a0, b1)) or roads.get(((a0 + a1) // 2, b1))     # +v 쪽 길
            road_u = roads.get((a1, b0)) or roads.get((a1, (b0 + b1) // 2))     # +u 쪽 길
            blk = {'u0': a0, 'u1': a1, 'v0': b0, 'v1': b1, 'road_v': road_v, 'road_u': road_u,
                   'dg': abs((a0 + a1) / 2 - gpos[0]) + abs((b0 + b1) / 2 - gpos[1]),
                   'dc': abs((a0 + a1) / 2 - cross_pt[0]) + abs((b0 + b1) / 2 - cross_pt[1]),
                   'back': (a0 + a1) / 2 - u_lo + (b0 + b1) / 2 - v_lo,
                   'front_wall': (front_u and a1 >= u_hi - 1) or (front_v and b1 >= v_hi - 1),
                   'in_play': a0 < play[1] and b0 < play[3]}
            blocks.append(blk)

    # 4. 역할 건물과 구역
    need = []
    for role in ('yamen', 'smithy', 'tavern', 'shop', 'mansion'):
        if role in roles:
            need.append(role)
    for role in ('yamen', 'tavern', 'shop', 'smithy'):
        if role not in need:
            need.append(role)
    taken = set()
    objs, lines = [], []
    placed_roles = {}
    stats = {'fence': 0}

    def try_place(blk, name, encl, face, door_in_play):
        """블록 안에 길 쪽을 향해 놓기. face 'v'면 +v 길 쪽 앞줄, 'u'면 +u 길 쪽(좌우 반전)"""
        key = name + ('_r' if face == 'u' else '')
        fu, fv = meta[key]['footprint']
        pad = (1, 1, 1, 3) if encl else (0, 0, 0, 1)
        if face == 'u':
            pad = (1, 1, 3, 1) if encl else (0, 0, 1, 0)
        cand = []
        if face == 'v':
            bv = blk['v1'] - fv - (pad[3] if encl else pad[3])
            for bu in range(blk['u0'] + pad[0], blk['u1'] - fu - pad[2] + 1):
                cand.append((bu, bv))
        else:
            bu = blk['u1'] - fu - pad[2]
            for bv in range(blk['v0'] + pad[1], blk['v1'] - fv - pad[3] + 1):
                cand.append((bu, bv))
        rnd.shuffle(cand)
        cand.sort(key=lambda c: abs((c[0] if face == 'v' else c[1]) - ((blk['u0'] + blk['u1']) // 2 if face == 'v' else (blk['v0'] + blk['v1']) // 2)) // 3)
        for bu, bv in cand:
            a0, b0, a1, b1 = bu - pad[0], bv - pad[1], bu + fu + pad[2], bv + fv + pad[3]
            if a0 < blk['u0'] or b0 < blk['v0'] or a1 > blk['u1'] or b1 > blk['v1']:
                continue
            area = tiles_rect(a0, b0, a1 - a0, b1 - b0)
            ring = tiles_rect(a0 - 1, b0 - 1, a1 - a0 + 2, b1 - b0 + 2)
            if area & hard or ring & taken or area & nobuild:
                continue
            if face == 'v':
                door = (bu + fu // 2, bv + fv) if not encl else (bu + (fu - 2) // 2, bv + fv + 2)
            else:
                door = (bu + fu, bv + fv // 2) if not encl else (bu + fu + 2, bv + (fv - 2) // 2)
            if door_in_play and not (play[0] <= door[0] < play[1] and play[2] <= door[1] < play[3]):
                continue
            taken.update(area)
            objs.append(place_obj(f'{region}_{key}', bu, bv, fu, fv))
            if encl:
                fam = f'{region}_{encl}'
                L, R, B, F = bu - 1, bu + fu, bv - 1, bv + fv
                if face == 'v':
                    F = bv + fv + 1
                    lines.extend([[fam, [L, B], [R, B]], [fam, [L, B + 1], [L, F]], [fam, [R, B + 1], [R, F]]])
                    g0 = bu + (fu - (1 if encl == 'fence' else 2)) // 2
                    gw = 1 if encl == 'fence' else 2
                    if g0 - 1 >= L + 1:
                        lines.append([fam, [L + 1, F], [g0 - 1, F]])
                    if g0 + gw <= R - 1:
                        lines.append([fam, [g0 + gw, F], [R - 1, F]])
                    if encl != 'fence':
                        objs.append(place_obj(f'{region}_{"tgate" if encl == "twall" else "egate"}_u', g0, F, 2, 1))
                else:
                    R = bu + fu + 1
                    lines.extend([[fam, [L, B], [L, bv + fv]], [fam, [L + 1, B], [R, B]], [fam, [L + 1, bv + fv], [R, bv + fv]]])
                    g0 = bv + (fv - (1 if encl == 'fence' else 2)) // 2
                    gw = 1 if encl == 'fence' else 2
                    if g0 - 1 >= B + 1:
                        lines.append([fam, [R, B + 1], [R, g0 - 1]])
                    if g0 + gw <= bv + fv - 1:
                        lines.append([fam, [R, g0 + gw], [R, bv + fv - 1]])
                    if encl != 'fence':
                        objs.append(place_obj(f'{region}_{"tgate" if encl == "twall" else "egate"}_v', R, g0, 1, 2))
                if encl == 'fence':
                    stats['fence'] += 1
            hard_door = {(door[0] + a, door[1] + b) for a in (-1, 0, 1) for b in (-1, 0, 1)}
            return {'name': key, 'bu': bu, 'bv': bv, 'fu': fu, 'fv': fv, 'door': door, 'reserve': hard_door}
        return None

    def faces(blk):
        out = []
        if blk['road_v']:
            out.append('v')
        if blk['road_u']:
            out.append('u')
        return out

    def score(blk, role):
        s = 0
        av = (blk['road_v'] == 'A') + (blk['road_u'] == 'A')
        if role == 'yamen':
            s = 10 * av - blk['back'] * .4 - blk['dc'] * .2
        elif role in ('tavern', 'shop'):
            s = 10 * av - blk['dc'] * .5 - blk['dg'] * .2
        elif role == 'smithy':
            s = 8 * av - blk['dg'] * .6
        elif role == 'mansion':
            s = 3 * av - blk['back'] * .5 + blk['dg'] * .1
        if blk['front_wall'] and role in ('yamen', 'mansion', 'tavern', 'shop'):
            s -= 50
        return s

    for role in need:
        name = {'mansion': 'mansion_1'}.get(role, role)
        encl = 'twall' if role in ('yamen', 'mansion') else None
        chain = [(name, encl)] + ([(name, None)] if encl else []) + {'yamen': [('mansion_1', 'twall'), ('mansion_1', None)],
                                                                     'mansion': [('mansion_4', 'twall'), ('mansion_4', None)],
                                                                     'shop': [('mansion_4', None), ('cottage_4', None)],
                                                                     'tavern': [('mansion_4', None), ('cottage_1', None)]}.get(role, [])
        done = None
        order = sorted([b for b in blocks if faces(b)], key=lambda b: -score(b, role))
        for rank, (nm, en) in enumerate(chain):          # 제 모양으로 모든 블록을 본 뒤에야 물러선다
            for blk in order:
                for fc in faces(blk):
                    done = try_place(blk, nm, en, fc, role in roles)
                    if done:
                        break
                if done:
                    break
            if done:
                blk['zone'] = {'yamen': '관아', 'mansion': '부유층', 'smithy': '상업', 'tavern': '상업', 'shop': '상업'}[role]
                done['fallback'] = rank
                break
        if done:
            placed_roles[role] = done
            hard |= done['reserve']
    # 구역: 남은 블록에 부유층 저택(뒤쪽 큰 블록), 민가 초가, 앞 가장자리 밭
    free = [b for b in blocks if 'zone' not in b]
    free.sort(key=lambda b: b['back'])
    n_rich = max(1, len(free) // 5)
    for i, blk in enumerate(free):
        if i < n_rich and not blk['front_wall']:
            blk['zone'] = '부유층'
        elif not blk['in_play'] and (blk['u1'] >= u_hi - 1 or blk['v1'] >= v_hi - 1) and not any(b.get('zone') == '밭' for b in blocks):
            blk['zone'] = '밭'             # 성 안 밭은 앞 가장자리 한 블록만
        else:
            blk['zone'] = '민가'
    for blk in blocks:
        z = blk.get('zone')
        fcs = faces(blk) or ['v']
        if z == '부유층':
            for nm in ('mansion_2', 'mansion_3', 'mansion_4', 'mansion_1'):
                for fc in fcs:
                    try_place(blk, nm, 'twall', fc, False)
        elif z in ('민가', '상업', '관아'):
            cots = ['cottage_1', 'cottage_4', 'cottage_2', 'cottage_3', 'cottage_4', 'cottage_1', 'cottage_4']
            rnd.shuffle(cots)
            for nm in cots:
                fc = rnd.choice(fcs)
                encl = 'fence' if (stats['fence'] < 2 and rnd.random() < .12) else ('ewall' if rnd.random() < .12 else None)
                try_place(blk, nm, encl, fc, False) or try_place(blk, nm, None, fc, False)
            if z == '민가':                   # 길가 줄 뒤 안쪽: 빈 땅에 작은 초가를 한 칸씩 띄워
                for _ in range(3):
                    nm = rnd.choice(['cottage_4', 'cottage_1', 'cottage_4', 'cottage_2'])
                    key = nm + ('_r' if rnd.random() < .5 else '')
                    fu, fv = meta[key]['footprint']
                    spots = [(bu, bv) for bu in range(blk['u0'] + 1, blk['u1'] - fu) for bv in range(blk['v0'] + 1, blk['v1'] - fv)]
                    rnd.shuffle(spots)
                    for bu, bv in spots:
                        area = tiles_rect(bu - 2, bv - 2, fu + 4, fv + 4)
                        if area & hard or area & taken or area & nobuild:
                            continue
                        taken.update(tiles_rect(bu, bv, fu, fv))
                        taken.update(area)
                        objs.append(place_obj(f'{region}_{key}', bu, bv, fu, fv))
                        break
    # 밭: 땅 글자로 칠할 수 없어 지역 밭 그림을 블록 가운데
    field = 'field_paddy' if 'field_paddy' in meta else 'field_dry'
    for blk in blocks:
        if blk.get('zone') == '밭':
            for bu in range(blk['u0'], blk['u1'] - 3, 5):
                for bv in range(blk['v0'], blk['v1'] - 3, 5):
                    area = tiles_rect(bu, bv, 4, 4)
                    if not (area & hard or area & taken or area & nobuild) and sum(1 for o in objs if '_field_' in o[0]) < 2:
                        taken.update(area)
                        objs.append(place_obj(f'{region}_{field}', bu, bv, 4, 4))

    # 5. 바닥: 옛 글자를 지우고 새로 (성 안만, 성 밖은 그대로)
    g = [list(r) for r in d['ground']]
    for v in range(oy, oy + H):                     # 옛 길·바닥은 판 전체에서 지운다 (물·성벽 밑은 그대로)
        for u in range(ox, ox + W):
            if g[v - oy][u - ox] in 'di':
                g[v - oy][u - ox] = '.'
    for (u, v), c in roads.items():
        if ox <= u < ox + W and oy <= v < oy + H and g[v - oy][u - ox] == '.':
            g[v - oy][u - ox] = ROAD
    for blk in blocks:                       # 상업 블록 앞길 쪽은 흙 마당
        if blk.get('zone') == '상업':
            for u in range(blk['u0'], blk['u1']):
                for v in range(max(blk['v0'], blk['v1'] - 2), blk['v1']):
                    if (u, v) not in taken and ROAD != 'd':
                        g[v - oy][u - ox] = 'd'
    # 성문 바깥 길은 옛 글자 그대로 둔다; 성문 칸 앞은 대로와 잇기
    d['ground'] = [''.join(r) for r in g]
    # 6. 소품: 대로변 노점(상업 블록 앞), 교차로 우물(원래 우물이 없으면), 대로 버드나무
    props = []
    stalls = ['market_stall', 'market_stall_cloth', 'market_stall_jar']
    for blk in blocks:
        if blk.get('zone') == '상업' and blk['road_v'] == 'A':
            for k, u in enumerate(range(blk['u0'] + 1, blk['u1'] - 1, 4)):
                c = (u, blk['v1'] - 1)
                if c not in hard and c not in taken:
                    props.append([stalls[k % 3], c[0], c[1], 0, -8])
                    taken.add(c)
    if not any(o[0] == 'well' for o in kept):
        c = (cross_pt[0] + 3, cross_pt[1] + 3)
        if c not in hard and c not in taken:
            props.append(['well', c[0], c[1], 0, -8])
    trees = [(o[1], o[2]) for o in kept if o[0].startswith('stree')]

    def tree_ok(c):
        """나무 자리: 성 안, 성벽 3칸·건물 2칸·다른 나무 3칸 밖, 길·지킬 칸이 아님"""
        if not inside(*c) or c in nobuild or c in hard or c in roads:
            return False
        if any(abs(c[0] - a) <= 2 and abs(c[1] - b) <= 2 for a, b in taken):
            return False
        return all(max(abs(c[0] - a), abs(c[1] - b)) > 3 for a, b in trees)
    for k in range(u_lo + 4, u_hi - 3, 8) if main_axis == 'u' else range(v_lo + 4, v_hi - 3, 8):   # 대로변 버드나무
        c = (k, main_at - 1) if main_axis == 'u' else (main_at - 1, k)
        if tree_ok(c):
            props.append(['stree_willow', c[0], c[1], 0, -8])
            trees.append(c)
    for blk in blocks:                       # 민가·부유층 블록 빈 땅에 나무 하나
        if blk.get('zone') in ('민가', '부유층'):
            spots = [(u, v) for u in range(blk['u0'], blk['u1']) for v in range(blk['v0'], blk['v1'])]
            rnd.shuffle(spots)
            for c in spots:
                if tree_ok(c):
                    props.append([rnd.choice(['stree_broad_0', 'stree_broad_1', 'stree_bush_1']), c[0], c[1], 0, -8])
                    trees.append(c)
                    break
    d['objects'] = objs + props + kept
    d['walls'] = walls + lines

    # 검사
    missing = [r for r in need if r in roles and r not in placed_roles]
    covered = [c for c in keep if c in taken]
    fall = sum(p.get('fallback', 0) for p in placed_roles.values())
    inner = (u_hi - u_lo) * (v_hi - v_lo)
    score_v = -10 * len(missing) - 5 * len(covered) - 8 * fall + len(objs) + 20 * len(taken) / max(inner, 1)
    ok = not missing and not covered
    return {'ok': ok, 'score': score_v, 'map': d, 'roles': placed_roles, 'blocks': blocks, 'missing': missing,
            'covered': covered, 'fence': stats['fence'], 'main': (main_axis, main_at), 'cross': cross_at,
            'roads_kind': ROAD,
            'bounds': (u_lo, u_hi, v_lo, v_hi)}


# ---------------------------------------------------------------- 탐험·사인 고치기

def patch_scenario(root, res, exits, signs, region):
    """역할 건물 출구를 새 문 칸으로, Object를 새 건물 칸으로. 대장간은 바깥: 대장장이를 문 앞으로, 실내 지역 빼기"""
    scn = os.path.join(root, 'assets/scenario/hero')
    notes = []
    files = sorted({e['file'] for e in exits} | {s['file'] for s in signs})
    for f in files:
        path = os.path.join(scn, f)
        d = json.load(open(path))
        A = d['Areas']
        for k, a in list(A.items()):
            if k not in A or not any(e['file'] == f and e['area'] == k for e in exits):
                continue
            new_exits = []
            for e in a.get('Exits') or []:
                role = ROLE_BY_MAP.get((A.get(e.get('To')) or {}).get('Map'))
                pr = res['roles'].get(role)
                if role == 'smithy' and pr and e.get('To') in A:
                    smith_area = A[e['To']]
                    people = a.setdefault('People', {})
                    for i, (n, p) in enumerate((smith_area.get('People') or {}).items()):
                        q = dict(p)
                        q['At'] = [pr['door'][0] + (0 if i == 0 else 2), pr['door'][1] + (0 if i == 0 else 1)]
                        people[n if n not in people else n + ' (대장간)'] = q
                    notes.append(f'{f}/{k}: 대장간 바깥, 대장장이 {list(pr["door"])}')
                    del A[e['To']]
                    continue
                if pr:
                    e['At'] = list(pr['door'])
                    if 'Object' in e:
                        e['Object'] = [pr['bu'] + pr['fu'] // 2, pr['bv'] + pr['fv'] // 2]
                    notes.append(f'{f}/{k}: {e.get("Name")} 출구 → {list(pr["door"])}')
                new_exits.append(e)
            a['Exits'] = new_exits
            for s in a.get('Signs') or []:
                if s.get('Name') == '저택' and 'mansion' in res['roles']:
                    s['At'] = list(res['roles']['mansion']['door'])
        json.dump(d, open(path, 'w'), ensure_ascii=False, indent=1)
    return notes


if __name__ == '__main__':
    root, map_id, region = sys.argv[1], sys.argv[2], sys.argv[3]
    seed = int(sys.argv[4]) if len(sys.argv) > 4 else 0
    res, keep, exits, signs, roles, uses = generate(map_id, region, seed)
    json.dump(res['map'], open(os.path.join(root, 'assets/graphics/maps', map_id + '.json'), 'w'), ensure_ascii=False, indent=1)
    notes = patch_scenario(root, res, exits, signs, region)
    os.makedirs(os.path.join(root, 'roles'), exist_ok=True)
    info = {'map': map_id, 'region': region, 'uses': uses,
            'roles': {ROLE_KO[r]: names for r, names in roles.items()},
            'placed': {ROLE_KO[r]: {'name': p['name'], 'door': p['door']} for r, p in res['roles'].items()},
            'zones': [{k: b[k] for k in ('u0', 'u1', 'v0', 'v1', 'zone') if k in b} for b in res['blocks']],
            'ok': res['ok'], 'missing': res['missing'], 'covered': res['covered'], 'fenced_cottages': res['fence'],
            'main_avenue': res['main'], 'cross_avenue': res['cross'], 'bounds': res['bounds'], 'notes': notes,
            'grow': res.get('grow', 0)}
    json.dump(info, open(os.path.join(root, 'roles', map_id + '.json'), 'w'), ensure_ascii=False, indent=1)
    print(json.dumps({k: info[k] for k in ('map', 'placed', 'ok', 'missing', 'covered', 'fenced_cottages', 'grow', 'bounds')}, ensure_ascii=False))
    for n in notes:
        print(' ', n)
