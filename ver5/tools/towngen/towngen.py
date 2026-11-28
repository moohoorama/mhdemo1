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

import stage

HERE = os.path.dirname(os.path.abspath(__file__))
VER5 = '/Users/younwookim/orca/mhdemo1/ver5'
SRC_MAPS = os.path.join(VER5, 'assets/graphics/maps')
SRC_SCN = os.path.join(VER5, 'assets/scenario/hero')
SRC_SCENES = os.path.join(VER5, 'assets/scenes')
ROOT = os.path.join(HERE, 'work', 'root')

ROLE_BY_MAP = {'S_tool_shop': 'shop', 'S_tavern': 'tavern', 'S_zhuo_office': 'yamen', 'S_xiaopei_office': 'yamen',
               'S_council': 'yamen', 'S_liu_house': 'mansion', 'S_mansion': 'mansion', 'S_weapon_shop': 'smithy'}
ROLE_KO = {'yamen': '관청', 'tavern': '주점', 'shop': '도구점', 'smithy': '대장간', 'mansion': '저택', 'palace': '황궁'}
KEEP_INSIDE = ('notice_post', 'notice_board', 'well', 'sgate_open', 'sgate_closed', 'palace_gate', 'pavilion')
TREES = ('stree_willow', 'stree_broad_0', 'stree_broad_1', 'stree_bush_0', 'stree_bush_1', 'stree_pine_0', 'stree_pine_1',
         'stree_plum_0', 'stree_plum_1', 'stree_old')


# 맵별 설정 (사용자 11-28): avenue 대로 글자, flip_u 앞쪽(+u) 성벽 너머가 성 안, ruin 수공 뒤 황폐,
# drop_water 끊긴 물줄기 지우기, mode hermit(외딴 초가)·keep(원래 배치 살리고 바닥만), cliff_forest 원래 절벽 언덕을 두고 위를 숲으로,
# fence_around 마을 울타리 (hills: 절벽을 전투맵 언덕으로 — 단양에는 거절됨, 11-28)
CFG = {
    'S_ye': {'avenue': 'i'},
    'S_xuzhou_gate': {'flip_u': True, 'outside': True},
    'S_xiapi_inner': {'ruin': True},
    'S_mangdang': {'drop_water': True},
    'S_vil_peace': {'mode': 'keep', 'rename': {'tavern': 'cottage_3'}, 'thin_trees': 2},
    'S_jiangxia_harbor': {'mode': 'keep', 'erase_d': True},
    'S_danyang_village': {'mode': 'keep', 'cliff_forest': True, 'fence_around': True},
}


def load_meta(region):
    meta = json.load(open(os.path.join(os.path.dirname(os.path.abspath(ROOT)), 'reg', region, 'meta.json')))
    meta.update({k[len(region) + 1:]: v for k, v in json.load(open(os.path.join(os.path.dirname(os.path.abspath(ROOT)), 'lines', region, 'lines.json'))).items()})
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
    spots, walks, by = stage.gather_stage(SRC_SCENES, map_id)
    uses += [f'scene/{sid}' for sid in by]
    keep |= {(u + a, v + b) for u, v in spots for a in (-1, 0, 1) for b in (-1, 0, 1)} | walks
    # 선 자리 앞(화면 아래쪽) 네 칸은 비운다: 집·나무가 사람을 가린다
    keep |= {(u + a, v + b) for u, v in spots for a in range(0, 4) for b in range(0, 4) if a + b <= 4}
    return keep, exits, signs, roles, uses


# ---------------------------------------------------------------- 2~6. 짜기

def generate(map_id, region, seed=0, tries=40):
    meta = load_meta(region)
    src = json.load(open(os.path.join(SRC_MAPS, map_id + '.json')))
    cfg = CFG.get(map_id, {})
    src = prepare(src, cfg)
    keep, exits, signs, roles, uses = gather(map_id)
    if cfg.get('mode') == 'hermit':
        return hermit(src, meta, region, keep, roles, random.Random(seed)), keep, exits, signs, roles, uses
    if cfg.get('mode') == 'keep':
        return keep_layout(src, meta, region, keep, roles, random.Random(seed), cfg), keep, exits, signs, roles, uses
    best = None
    for grow in range(4):           # 통과하는 판이 없으면 성을 키운다 (앞쪽으로 6칸씩)
        cur = grow_map(src, 6 * grow)
        for t in range(tries):      # 검사를 통과한 것 가운데 점수가 가장 좋은 것
            res = attempt(cur, meta, region, keep, exits, signs, roles, random.Random(seed * 1000 + t), cfg)
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
    moved = []
    for o in d['objects']:                    # 앞쪽 성벽 위 성문도 같이 (그 성문의 출구도 나중에 같이 옮긴다)
        if o[0].startswith('sgate') and (o[1] > play_u or o[2] > play_v):
            old = (o[1], o[2])
            if o[1] > play_u:
                o[1] += n
            else:
                o[2] += n
            moved.append([list(old), [o[1] - old[0], o[2] - old[1]]])
    d['_moved_gates'] = moved
    return d


def prepare(src, cfg):
    """생성 전에 원본을 손본다: 끊긴 물줄기 지우기, 절벽을 평지로 (언덕 자리는 _hills에 남긴다)"""
    d = json.loads(json.dumps(src))
    if cfg.get('drop_water'):
        d['ground'] = [r.replace('~', '.') for r in d['ground']]
    if cfg.get('hills') and d.get('height'):
        ox, oy = d['origin']
        d['_hills'] = [[u + ox, v + oy] for v, r in enumerate(d['height']) for u, c in enumerate(r) if c != '0']
        d['height'] = ['0' * len(r) for r in d['height']]
    return d


def attempt(src, meta, region, keep, exits, signs, roles, rnd, cfg=None):
    cfg = cfg or {}
    d = json.loads(json.dumps(src))
    src_objects = d['objects']
    ox, oy = d['origin']
    W, H = len(d['ground'][0]), len(d['ground'])
    play = (0, 2 * d['size'][0], 0, 2 * d['size'][1])
    walls = [w for w in d.get('walls', []) if w[0] == 'swall']
    wall = set()
    for w in walls:
        (a0, b0), (a1, b1) = w[1], w[2]
        wall |= {(a, b) for a in range(min(a0, a1), max(a0, a1) + 1) for b in range(min(b0, b1), max(b0, b1) + 1)}
    # 성 안 범위
    u_lo, u_hi, v_lo, v_hi = ox + 3, ox + W - 3, oy + 3, oy + H - 3
    front_u = front_v = back_u = back_v = False
    mid_u, mid_v = (play[0] + play[1]) / 2, (play[2] + play[3]) / 2
    for w in walls:
        (a0, b0), (a1, b1) = w[1], w[2]
        if a0 == a1:                      # v를 따라 선 벽: u = a0
            if a0 < mid_u or cfg.get('flip_u'):
                u_lo = max(u_lo, a0 + 1); back_u = True
            else:
                u_hi = min(u_hi, a0); front_u = True
        else:
            if b0 < mid_v:
                v_lo = max(v_lo, b0 + 1); back_v = True
            else:
                v_hi = min(v_hi, b0); front_v = True
    # 황궁: 궁성 담 줄(pwall) 앞이 성 안의 시작
    pw = [o for o in d['objects'] if o[0].startswith('pwall')]
    if pw:
        v_lo = max(v_lo, max(o[2] for o in pw) + 1); back_v = True
    gates = [o for o in d['objects'] if o[0].startswith('sgate')]
    palace = [o for o in d['objects'] if o[0].startswith(('palace', 'pwall'))]
    # 지킬 물건: 성 안의 게시판·우물·성문·황궁, 성 밖 것은 모두
    inside = lambda u, v: u_lo <= u < u_hi and v_lo <= v < v_hi
    hg = d.get('height') or []
    raised = lambda u, v: bool(hg) and 0 <= v - oy < len(hg) and 0 <= u - ox < len(hg[0]) and hg[v - oy][u - ox] != '0'
    kept = [o for o in d['objects'] if not inside(o[1], o[2]) or o[0].startswith(KEEP_INSIDE) or o[0].startswith(('pwall', 'palace'))
            or raised(o[1], o[2])]
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
    road_start_u = u_lo if back_u else ox           # 뒤 성벽이 없으면 판 끝부터
    road_start_v = v_lo if back_v else oy
    # 원래 땅에서 건물·길을 둘 수 없는 칸: 물·다리·특수 바닥, 높은 땅
    hgt = d.get('height') or []
    noroad = set()
    for u, v in d.pop('_hills', []):              # 언덕·산으로 바꿀 자리 (prepare)
        hard.add((u, v)); nobuild.add((u, v)); noroad.add((u, v))
        d.setdefault('_hill_cells', []).append([u, v])
    for v in range(oy, oy + H):
        for u in range(ox, ox + W):
            c = d['ground'][v - oy][u - ox]
            hh = hgt[v - oy][u - ox] if hgt else '0'
            if c not in '.di' or hh != '0':
                hard.add((u, v)); nobuild.add((u, v)); noroad.add((u, v))
    # 황궁 둘레 4칸은 비운다
    for o in d['objects']:
        if o[0].startswith(('pwall', 'palace')):
            nobuild |= {(o[1] + du, o[2] + dv) for du in range(-4, 5) for dv in range(-4, 6)}
    ROAD = 'i' if palace else 'd'                   # 길은 한 종류: 황성은 포석, 고을은 흙길

    # 3. 길: 성문에서 직선 대로, 교차 대로, 골목
    roads = {}                     # 칸 -> 'A'(대로) / 'L'(골목), 바닥은 둘 다 ROAD
    main_axis, main_at = 'u', (u_lo + 4)
    def gate_axis(g):
        on_v_wall = any(w[1][0] == w[2][0] == g[1] for w in walls)
        return ('u', g[2] - 1) if on_v_wall else ('v', g[1] - 1)   # 통로는 성문 칸 -1, 0 (gate_passage = 그림 = 조각 배치)
    gates.sort(key=lambda g: (g[1] > mid_u or g[2] > mid_v, g[0] != 'sgate_open'))   # 뒤 성벽의 열린 성문 먼저
    pg = [o for o in palace if o[0] == 'palace_gate']
    if pg:                                          # 황성: 황궁 문에서 대로
        main_axis, main_at = 'v', pg[0][1] - 1
    elif gates:
        main_axis, main_at = gate_axis(gates[0])
    else:                                           # 성벽 없는 마을: 긴 쪽으로 가운데를 지나는 길
        if (u_hi - u_lo) >= (v_hi - v_lo):
            main_axis, main_at = 'u', int(mid_v) - 1
        else:
            main_axis, main_at = 'v', int(mid_u) - 1
    jitter = rnd.choice([0, 0, -1, 1, -2, 2])
    if main_axis == 'u':
        cross_at = max(u_lo + 6, min(u_hi - 8, (play[0] + play[1]) // 2 + jitter))
    else:
        cross_at = max(v_lo + 6, min(v_hi - 8, (play[2] + play[3]) // 2 + jitter))

    def lay(axis, at, width, glyph, lo=None, hi=None):
        if axis == 'u':
            for u in range(road_start_u if lo is None else lo, road_end_u if hi is None else hi):
                for k in range(width):
                    roads.setdefault((u, at + k), glyph)
        else:
            for v in range(road_start_v if lo is None else lo, road_end_v if hi is None else hi):
                for k in range(width):
                    roads.setdefault((at + k, v), glyph)
    other = 'v' if main_axis == 'u' else 'u'
    is_village = not walls and not palace
    vlo_u, vhi_u, vlo_v, vhi_v = play[0] - 6, play[1] + 6, play[2] - 6, play[3] + 6
    lay(main_axis, main_at, 2, 'A')
    if is_village:                                  # 마을: 판 끝까지 가는 건 큰길 하나, 나머지는 마을 둘레 안
        lay(other, cross_at, 2, 'A', lo=(vlo_u if other == 'u' else vlo_v), hi=(vhi_u if other == 'u' else vhi_v))
    else:
        lay(other, cross_at, 2, 'A')
    for g in gates:                                 # 성문마다: 성 안으로 곧게, 밖으로 판 끝까지
        ax, at = gate_axis(g)
        lay(ax, at, 2, 'A')
        if ax == 'u':
            lay('u', at, 2, 'A', lo=ox, hi=u_lo) if g[1] < u_lo else lay('u', at, 2, 'A', lo=u_hi, hi=ox + W)
        else:
            lay('v', at, 2, 'A', lo=oy, hi=v_lo) if g[2] < v_lo else lay('v', at, 2, 'A', lo=v_hi, hi=oy + H)
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
    if is_village:
        lane_main = [p for p in lane_main if (vlo_v if main_axis == 'u' else vlo_u) <= p < (vhi_v if main_axis == 'u' else vhi_u)]
        lane_cross = [p for p in lane_cross if (vlo_u if main_axis == 'u' else vlo_v) <= p < (vhi_u if main_axis == 'u' else vhi_v)]
        for p in lane_main:
            lay(main_axis, p, 1, 'L', lo=(vlo_u if main_axis == 'u' else vlo_v), hi=(vhi_u if main_axis == 'u' else vhi_v))
        for p in lane_cross:
            lay(other, p, 1, 'L', lo=(vlo_u if other == 'u' else vlo_v), hi=(vhi_u if other == 'u' else vhi_v))
    else:
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
    olds = [o for o in src_objects if o[0].startswith(('house', 'hall', 'tavern'))]
    orig = set()
    halls = [o for o in olds if o[0].startswith('hall')]
    if halls and (walls or palace):
        orig.add('yamen')
    if len(halls) > 1 or (halls and not (walls or palace)):
        orig.add('mansion')
    if any(o[0] == 'tavern' for o in olds):
        orig.add('tavern')
    need = [r for r in ('yamen', 'smithy', 'tavern', 'shop', 'mansion') if r in roles or r in orig]
    target = len(olds) if not cfg.get('ruin') else max(4, len(olds) // 2)
    taken = set()
    objs, lines = [], []
    placed_roles = {}
    stats = {'fence': 0}

    def try_place(blk, name, encl, face, door_in_play):
        """블록 안에 길 쪽을 향해 놓기. face 'v'면 +v 길 쪽 앞줄, 'u'면 +u 길 쪽(좌우 반전).
        담장 마당 여백: 옆 ms, 뒤 mb, 앞 mf칸. 관청은 넓게(옆 2·앞 5) 그리고 대문 밖 광장 2줄"""
        key = name + ('_r' if face == 'u' else '')
        fu, fv = meta[key]['footprint']
        # 관청 구역: encl 'twall'/'twall1'(담장 마당 넓게/좁게), 'yard'/'yard1'(담장 없이 포석 마당만)
        big = name.startswith('yamen') and bool(encl)
        yard = bool(encl) and encl.startswith('yard')
        ms, mb, mf = ((1 if encl.endswith('1') else 2), 1, 4) if big else (1, 1, 2)
        plaza = 1 if big else 0
        wall_kind = None if yard else (encl.rstrip('1') if encl else None)
        if encl:
            pad = (ms, mb, ms, mf + 1 + plaza) if face == 'v' else (mb, ms, mf + 1 + plaza, ms)
        else:
            pad = (0, 0, 0, 1) if face == 'v' else (0, 0, 1, 0)
        cand = []
        if face == 'v':
            bv = blk['v1'] - fv - pad[3]
            for bu in range(blk['u0'] + pad[0], blk['u1'] - fu - pad[2] + 1):
                cand.append((bu, bv))
        else:
            bu = blk['u1'] - fu - pad[2]
            for bv in range(blk['v0'] + pad[1], blk['v1'] - fv - pad[3] + 1):
                cand.append((bu, bv))
        rnd.shuffle(cand)
        cand.sort(key=lambda c: abs((c[0] if face == 'v' else c[1]) - ((blk['u0'] + blk['u1']) // 2 if face == 'v' else (blk['v0'] + blk['v1']) // 2)) // 3)
        gw = 1 if encl == 'fence' else 2
        for bu, bv in cand:
            a0, b0, a1, b1 = bu - pad[0], bv - pad[1], bu + fu + pad[2], bv + fv + pad[3]
            if a0 < blk['u0'] or b0 < blk['v0'] or a1 > blk['u1'] or b1 > blk['v1']:
                continue
            area = tiles_rect(a0, b0, a1 - a0, b1 - b0)
            ring = tiles_rect(a0 - 1, b0 - 1, a1 - a0 + 2, b1 - b0 + 2)
            if area & hard or ring & taken or area & nobuild:
                continue
            rect = None
            if encl and face == 'v':
                L, R, B, F = bu - ms, bu + fu + ms - 1, bv - mb, bv + fv + mf - 1
                g0 = bu + (fu - gw) // 2
                door = (g0, F + 1)
                rect = (L, R, B, F)
            elif encl:
                B, F, L, R = bv - ms, bv + fv + ms - 1, bu - mb, bu + fu + mf - 1
                g0 = bv + (fv - gw) // 2
                door = (R + 1, g0)
                rect = (L, R, B, F)
            elif face == 'v':
                door = (bu + fu // 2, bv + fv)
            else:
                door = (bu + fu, bv + fv // 2)
            if door_in_play and not (play[0] <= door[0] < play[1] and play[2] <= door[1] < play[3]):
                continue
            taken.update(area)
            objs.append(place_obj(f'{region}_{key}', bu, bv, fu, fv))
            if wall_kind:
                encl = wall_kind
                fam = f'{region}_{encl}'
                L, R, B, F = rect
                if face == 'v':
                    lines.extend([[fam, [L, B], [R, B]], [fam, [L, B + 1], [L, F]], [fam, [R, B + 1], [R, F]]])
                    if g0 - 1 >= L + 1:
                        lines.append([fam, [L + 1, F], [g0 - 1, F]])
                    if g0 + gw <= R - 1:
                        lines.append([fam, [g0 + gw, F], [R - 1, F]])
                    if encl != 'fence':
                        objs.append(place_obj(f'{region}_{"tgate" if encl == "twall" else "egate"}_u', g0, F, 2, 1))
                else:
                    lines.extend([[fam, [L, B], [L, F]], [fam, [L + 1, B], [R, B]], [fam, [L + 1, F], [R, F]]])
                    if g0 - 1 >= B + 1:
                        lines.append([fam, [R, B + 1], [R, g0 - 1]])
                    if g0 + gw <= F - 1:
                        lines.append([fam, [R, g0 + gw], [R, F - 1]])
                    if encl != 'fence':
                        objs.append(place_obj(f'{region}_{"tgate" if encl == "twall" else "egate"}_v', R, g0, 1, 2))
                if encl == 'fence':
                    stats['fence'] += 1
            hard_door = {(door[0] + a, door[1] + b) for a in (-1, 0, 1) for b in (-1, 0, 1)}
            return {'name': key, 'bu': bu, 'bv': bv, 'fu': fu, 'fv': fv, 'door': door, 'reserve': hard_door, 'encl': (encl if encl else None),
                    'face': face, 'rect': rect, 'gate': (g0 if encl else None), 'plaza': plaza}
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
        chain = ([(name, 'twall'), (name, 'twall1'), (name, 'yard'), (name, 'yard1')] if role == 'yamen' else [(name, encl)]) + \
                ([(name, None)] if encl else []) + {'yamen': [('mansion_1', 'twall'), ('mansion_1', None)],
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
    village = not walls and not palace
    if village:                                     # 성벽 없는 마을: 놀이 영역 둘레 6칸 안 블록만 쓴다
        for b in blocks:
            if b['u0'] > play[1] + 6 or b['v0'] > play[3] + 6 or b['u1'] < play[0] - 6 or b['v1'] < play[2] - 6:
                b.setdefault('zone', '빈 땅')
    cap = max(int(target * 1.3), 8)                 # 집 수 상한 (역할 건물 제외)
    n_stall0 = sum(1 for o in src_objects if o[0].startswith('market_stall'))
    if (walls or palace) or n_stall0:               # 시장: 성(또는 원래 매대가 있던 마을)에서 교차로 가까운 대로변 블록 하나
        cand = [b for b in blocks if 'zone' not in b and (b['road_v'] == 'A' or b['road_u'] == 'A') and b['in_play']
                and not b['front_wall'] and (b['u1'] - b['u0']) >= 6 and (b['v1'] - b['v0']) >= 5]
        if cand:
            min(cand, key=lambda b: b['dc'])['zone'] = '시장'
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
    def n_houses():
        return sum(1 for o in objs if not o[0].endswith(('_u', '_v')) and '_field_' not in o[0])
    for blk in blocks:
        if n_houses() >= cap:
            break
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
    # 집 수를 원래 맵의 80% 이상으로: 모자라면 민가 블록에 간격을 조금 좁혀 더
    def n_buildings():
        return sum(1 for o in objs if not o[0].endswith(('_u', '_v')) and '_field_' not in o[0])
    res_blocks = [b for b in blocks if b.get('zone') in ('민가', '부유층', '상업')] or blocks
    for _ in range(60):
        if n_buildings() >= int(target * .75):
            break
        blk = rnd.choice(res_blocks)
        nm = rnd.choice(['cottage_4', 'cottage_1', 'cottage_2', 'cottage_3'])
        key = nm + ('_r' if rnd.random() < .5 else '')
        fu, fv = meta[key]['footprint']
        spots = [(bu, bv) for bu in range(blk['u0'], blk['u1'] - fu + 1) for bv in range(blk['v0'], blk['v1'] - fv + 1)]
        rnd.shuffle(spots)
        for bu, bv in spots:
            area = tiles_rect(bu - 2, bv - 2, fu + 4, fv + 4)
            if area & hard or area & taken or tiles_rect(bu, bv, fu, fv) & nobuild:
                continue
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

    # 소품 (원래 맵에 있던 만큼): 관청 앞 깃발·등 한 쌍씩, 교차로·대로변 상업 구역에 매대·수레·자루·물항아리, 주점 앞 술항아리
    props = []
    used_props = set()

    def prop_ok(c, gap=2):
        if not inside(*c) or c in hard or c in taken or c in roads or c in nobuild:
            return False
        return all(max(abs(c[0] - a), abs(c[1] - b)) >= gap for a, b in used_props)

    def put(name, c, gap=2):
        if prop_ok(c, gap):
            props.append([name, c[0], c[1], 0, -8])
            used_props.add(c)
            return True
        return False
    if 'yamen' in placed_roles:
        pr = placed_roles['yamen']
        x, y = pr['door']
        flank = [((x - 2, y), (x + 3, y)), ((x - 3, y), (x + 4, y))] if not pr['name'].endswith('_r') else \
                [((x, y - 2), (x, y + 3)), ((x, y - 3), (x, y + 4))]
        for name, (a, b) in zip(('banner', 'lantern'), flank):
            put(name, a, 1); put(name, b, 1)
    town = bool(walls or palace)
    for blk in blocks:                              # 시장 구역: 매대를 줄지어, 가장자리에 수레·자루·물항아리
        if blk.get('zone') != '시장':
            continue
        n_stall = max(6 if town else 2, n_stall0)
        stalls = ['market_stall', 'market_stall_cloth', 'market_stall_jar']
        rows = range(blk['v1'] - 2, blk['v0'], -3)     # 길에 가까운 줄부터
        k = 0
        for v in rows:
            for u in range(blk['u0'] + 1, blk['u1'] - 1, 3):
                if k < n_stall and put(stalls[k % 3], (u, v), 2):
                    k += 1
        edge = [(blk['u1'] - 1, blk['v0'] + 1), (blk['u0'] + 1, blk['v0'] + 1), (blk['u1'] - 1, blk['v1'] - 4)]
        for name, c in zip(['grain_cart', 'grain_sack', 'water_jar'], edge):
            put(name, c, 2)
    if 'tavern' in placed_roles:
        x, y = placed_roles['tavern']['door']
        for c in ((x - 2, y), (x - 2, y + 1), (x + 3, y)):
            put('wine_jar', c, 1)

    # 5. 바닥: 옛 글자를 지우고 새로 (성 안만, 성 밖은 그대로)
    g = [list(r) for r in d['ground']]
    for v in range(oy, oy + H):                     # 옛 길·바닥은 판 전체에서 지운다 (물·성벽 밑은 그대로)
        for u in range(ox, ox + W):
            if g[v - oy][u - ox] in 'di':
                g[v - oy][u - ox] = '.'
    for (u, v), c in roads.items():
        if ox <= u < ox + W and oy <= v < oy + H and g[v - oy][u - ox] == '.' and (u, v) not in noroad:
            g[v - oy][u - ox] = cfg.get('avenue', ROAD) if c == 'A' else ROAD
    for blk in blocks:                       # 상업 블록 앞길 쪽은 흙 마당
        if blk.get('zone') == '상업':
            for u in range(blk['u0'], blk['u1']):
                for v in range(max(blk['v0'], blk['v1'] - 2), blk['v1']):
                    if (u, v) not in taken and ROAD != 'd':
                        g[v - oy][u - ox] = 'd'
    # 성문 바깥 길은 옛 글자 그대로 둔다; 성문 칸 앞은 대로와 잇기
    if 'yamen' in placed_roles:                     # 관청: 담장 안 앞마당까지 전부와 대문 밖 광장을 돌 포석으로
        pr = placed_roles['yamen']
        bu, bv, fu, fv = pr['bu'], pr['bv'], pr['fu'], pr['fv']
        if pr.get('rect'):
            L, R, B, F = pr['rect']
            rect = tiles_rect(L, B, R - L + 1, F - B + 1)
            g0, pz = pr['gate'], max(2, pr.get('plaza', 0))
            plaza = tiles_rect(g0 - 2, F + 1, 6, pz) if pr['face'] == 'v' else tiles_rect(R + 1, g0 - 2, pz, 6)
        else:
            rect = tiles_rect(bu - 1, bv - 1, fu + 2, fv + 3) if pr['face'] == 'v' else tiles_rect(bu - 1, bv - 1, fu + 3, fv + 2)
            plaza = set()
        for u, v in rect | plaza:
            if ox <= u < ox + W and oy <= v < oy + H and g[v - oy][u - ox] in '.d' and (u, v) not in noroad:
                g[v - oy][u - ox] = 'i'
    d['ground'] = [''.join(r) for r in g]
    # 6. 소품: 대로변 노점(상업 블록 앞), 교차로 우물(원래 우물이 없으면), 대로 버드나무
    taken |= used_props
    if not any(o[0] == 'well' for o in kept):
        c = (cross_pt[0] + 3, cross_pt[1] + 3)
        if c not in hard and c not in taken:
            props.append(['well', c[0], c[1], 0, -8])
    trees = [(o[1], o[2]) for o in kept if o[0].startswith('stree')]
    species = [o[0] for o in src_objects if o[0].startswith('stree') and inside(o[1], o[2])] or \
              ['stree_broad_0', 'stree_broad_1', 'stree_bush_1']
    blossom = [s for s in species if s.startswith(('stree_plum', 'stree_peach', 'peach'))]

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
            props.append([('stree_willow' if 'stree_willow' in species or not blossom else rnd.choice(blossom)), c[0], c[1], 0, -8])
            trees.append(c)
    for blk in blocks:                       # 민가·부유층 블록 빈 땅에 나무 두 그루까지
        if blk.get('zone') in ('민가', '부유층'):
            spots = [(u, v) for u in range(blk['u0'], blk['u1']) for v in range(blk['v0'], blk['v1'])]
            rnd.shuffle(spots)
            n = 0
            for c in spots:
                if n < 2 and tree_ok(c):
                    props.append([rnd.choice(blossom or species) if rnd.random() < .5 else rnd.choice(species), c[0], c[1], 0, -8])
                    trees.append(c)
                    n += 1
    # 초가 마당 소품: 물항아리·곡식 자루·덤불·거적을 집 옆에 하나씩 (절반쯤)
    yard_items = ['water_jar', 'grain_sack', 'stree_bush_0', 'stree_bush_1', 'straw_mat', 'water_jar']
    for o in list(objs):
        if '_cottage_' in o[0] and rnd.random() < .6:
            for du, dv in ((2, 1), (1, 2), (-2, 1), (1, -2), (2, 2)):
                if put(rnd.choice(yard_items), (o[1] + du, o[2] + dv), 2):
                    break
    # 관청 대문 밖 광장을 가장 가까운 길까지 잇고, 길·포석에 둘러싸인 작은 풀밭 섬은 메운다
    g2 = [list(r) for r in d['ground']]
    gl = lambda u, v: g2[v - oy][u - ox] if ox <= u < ox + W and oy <= v < oy + H else '#'
    if 'yamen' in placed_roles and placed_roles['yamen'].get('rect'):
        pr = placed_roles['yamen']
        L, R, B, F = pr['rect']
        g0 = pr['gate']
        for k in range(1, 7):
            row = [(u, F + 1 + k) for u in range(g0 - 2, g0 + 4)] if pr['face'] == 'v' else [(R + 1 + k, v) for v in range(g0 - 2, g0 + 4)]
            if any(gl(*c) in 'di' and c in roads for c in row):
                break
            for u, v in row:
                if gl(u, v) == '.' and (u, v) not in taken and (u, v) not in noroad:
                    g2[v - oy][u - ox] = 'i'
    anchors = {(o[1], o[2]) for o in objs + props}
    seen = set()
    for v0 in range(v_lo, v_hi):
        for u0 in range(u_lo, u_hi):
            if (u0, v0) in seen or gl(u0, v0) != '.':
                continue
            comp, q = [], [(u0, v0)]
            seen.add((u0, v0))
            while q:
                c = q.pop()
                comp.append(c)
                for du, dv in ((1, 0), (-1, 0), (0, 1), (0, -1)):
                    n = (c[0] + du, c[1] + dv)
                    if n not in seen and gl(*n) == '.' and u_lo <= n[0] < u_hi and v_lo <= n[1] < v_hi:
                        seen.add(n); q.append(n)
            if len(comp) > 10 or any(c in anchors or c in taken or c in noroad for c in comp):
                continue
            near = [gl(c[0] + du, c[1] + dv) for c in comp for du, dv in ((1, 0), (-1, 0), (0, 1), (0, -1))]
            fill = 'i' if near.count('i') > near.count('d') else 'd'
            if near.count('i') + near.count('d') >= len(comp):
                for u, v in comp:
                    g2[v - oy][u - ox] = fill
    d['ground'] = [''.join(r) for r in g2]
    swapped = []
    for o in kept:
        if o[0].startswith(('house', 'hall', 'tavern')):
            nm = 'cottage_4' if o[0].startswith('house_thatch') else 'mansion_4'
            key = nm + ('_r' if o[0].endswith('_x') or o[0] == 'hall_v' else '')
            swapped.append([f'{region}_{key}'] + o[1:])
        else:
            swapped.append(o)
    kept = swapped
    d['objects'] = objs + props + kept
    d['walls'] = walls + lines

    # 검사
    missing = [r for r in need if r in roles and r not in placed_roles]
    covered = [c for c in keep if c in taken]
    fall = sum(p.get('fallback', 0) for p in placed_roles.values())
    inner = (u_hi - u_lo) * (v_hi - v_lo)
    score_v = -10 * len(missing) - 5 * len(covered) - 8 * fall + len(objs) + 20 * len(taken) / max(inner, 1) - 2 * max(0, int(target * .75) - n_buildings())
    ok = not missing and not covered
    return {'ok': ok, 'score': score_v, 'map': d, 'roles': placed_roles, 'blocks': blocks, 'missing': missing,
            'covered': covered, 'fence': stats['fence'], 'main': (main_axis, main_at), 'cross': cross_at,
            'roads_kind': ROAD,
            'bounds': (u_lo, u_hi, v_lo, v_hi)}


# ---------------------------------------------------------------- 맵별 특수 처리

def occupied(d, meta, region):
    """물건이 덮는 칸 (지역 그림은 바닥 크기, 나머지는 한 칸)"""
    occ = set()
    for o in d['objects']:
        n = o[0]
        key = n[len(region) + 1:] if n.startswith(region + '_') else None
        if key in meta and 'footprint' in meta[key]:
            fu, fv = meta[key]['footprint']
            dx, dy = (o[3], o[4]) if len(o) >= 5 and not isinstance(o[3], dict) else (0, 0)
            cu = o[1] + .5 + (dx / 16 + dy / 8) / 2 - fu / 2
            cv = o[2] + .5 + (dy / 8 - dx / 16) / 2 - fv / 2
            occ |= tiles_rect(round(cu), round(cv), fu, fv)
        else:
            occ.add((o[1], o[2]))
    for w in d.get('walls', []):
        (a0, b0), (a1, b1) = w[1], w[2]
        occ |= {(a, b) for a in range(min(a0, a1), max(a0, a1) + 1) for b in range(min(b0, b1), max(b0, b1) + 1)}
    return occ


def scatter(d, cells, names, rnd, keep, occ, gap=3, n=None, ground_ok='.'):
    """cells 가운데 빈 곳에 names를 gap 칸씩 띄워 흩어 놓는다"""
    ox, oy = d['origin']
    cells = list(cells)
    rnd.shuffle(cells)
    placed = []
    keepr = {(u + a, v + b) for u, v in keep for a in (-1, 0, 1) for b in (-1, 0, 1)}
    for c in cells:
        if n is not None and len(placed) >= n:
            break
        u, v = c
        if not (0 <= v - oy < len(d['ground']) and 0 <= u - ox < len(d['ground'][0])):
            continue
        if d['ground'][v - oy][u - ox] not in ground_ok or c in occ or c in keepr:
            continue
        if any(max(abs(u - a), abs(v - b)) < gap for a, b in placed):
            continue
        if any(max(abs(u - a), abs(v - b)) < 2 for a, b in occ if abs(u - a) < 3 and abs(v - b) < 3):
            continue
        name = names[len(placed) % len(names)] if isinstance(names, list) else names
        d['objects'].append([name, u, v, 0, -8])
        placed.append(c)
        occ.add(c)
    return placed


def set_ground(d, cells, glyph, only='.'):
    ox, oy = d['origin']
    g = [list(r) for r in d['ground']]
    for u, v in cells:
        if 0 <= v - oy < len(g) and 0 <= u - ox < len(g[0]) and g[v - oy][u - ox] in only:
            g[v - oy][u - ox] = glyph
    d['ground'] = [''.join(r) for r in g]


def post_outside(d, res, region, meta, keep, rnd):
    """서주성 앞: 성벽 왼쪽 위(성 밖)는 넓은 대로와 나무·밭만 (여포군이 나타나는 곳)"""
    u_lo = res['bounds'][0]
    ox, oy = d['origin']
    W, H = len(d['ground'][0]), len(d['ground'])
    out_cells = {(u, v) for u in range(ox, u_lo - 1) for v in range(oy, oy + H)}
    d['objects'] = [o for o in d['objects'] if not ((o[1], o[2]) in out_cells and not o[0].startswith(('stree', 'sgate')))]
    axis, at = res['main']
    road = {(u, at + k) for u in range(ox, u_lo) for k in (-1, 0, 1, 2)}       # 성문에서 나가는 넓은 대로 (4칸)
    set_ground(d, road, 'd')
    occ = occupied(d, meta, region) | road
    field = f'{region}_field_' + ('paddy' if 'field_paddy' in meta else 'dry')
    fields = []
    for bu in range(ox + 3, u_lo - 8, 6):
        for bv in range(oy + 3, oy + H - 6, 6):
            area = tiles_rect(bu, bv, 4, 4)
            if not (area & occ) and not any((u, v) in area for u, v in keep) and len(fields) < 4 and rnd.random() < .5:
                d['objects'].append(place_obj(field, bu, bv, 4, 4))
                occ |= tiles_rect(bu - 1, bv - 1, 6, 6)
                fields.append((bu, bv))
    scatter(d, out_cells, ['stree_broad_0', 'stree_pine_0', 'stree_broad_1', 'stree_willow'], rnd, keep, occ, gap=4, n=14)
    return [f'성 밖(왼쪽 위): 넓은 대로 4칸, 밭 {len(fields)}판, 나무']


def post_ruin(d, res, region, meta, keep, rnd):
    """하비 성안 수공 직후: 맵 전체가 흙탕 바닥, 웅덩이·젖은 자국·쓰레기 줄이 고르게, 집은 흙탕물 자국 진 판(_mud)"""
    ox, oy = d['origin']
    W, H = len(d['ground'][0]), len(d['ground'])
    allc = {(u, v) for u in range(ox, ox + W) for v in range(oy, oy + H)}
    set_ground(d, allc, 'd')
    for o in d['objects']:
        n = o[0]
        if n.startswith(region + '_') and '_field_' not in n and not n.endswith(('_u', '_v')) and not n.endswith('_mud'):
            o[0] = n + '_mud'
    occ = occupied(d, meta, region)
    scatter(d, allc, ['wet_patch_0', 'wet_patch_1', 'wet_patch_2', 'wet_patch_3'], rnd, keep, occ, gap=4, n=30, ground_ok='d.')
    scatter(d, allc, ['puddle_0', 'puddle_1', 'puddle_2', 'puddle_3', 'puddle_4', 'puddle_5'], rnd, keep, occ, gap=3, n=36, ground_ok='d.')
    scatter(d, allc, ['wrack_u', 'wrack_v', 'rut_v'], rnd, keep, occ, gap=4, n=16, ground_ok='d.')
    scatter(d, allc, ['broken_spear', 'straw_mat', 'fallen_flag'], rnd, keep, occ, gap=5, n=8, ground_ok='d.')
    return ['수공 뒤: 맵 전체 흙탕 바닥, 웅덩이·젖은 자국·쓰레기 줄, 부러진 창·거적, 흙탕물 자국 진 집, 집은 절반']


def post_hills(d, region, meta, keep, rnd):
    """단양: 절벽이던 뒤쪽 높은 땅을 전투맵 언덕(게임 칸)으로 덮고 맨 뒤에 산을 세운다"""
    cells = {tuple(c) for c in d.pop('_hill_cells', [])}
    full = {(u // 2, v // 2) for u, v in cells}       # 반 칸이라도 걸치면 언덕으로 덮어 위에 풀밭이 남지 않게
    for cx, cy in sorted(full):
        m = sum(1 << i for i, (du, dv) in enumerate(((-1, 0), (0, -1), (1, 0), (0, 1))) if (cx + du, cy + dv) in full)
        d['objects'].append([f'sc_knoll_{m:02d}', 2 * cx + 1, 2 * cy + 1, 0, -8])
    return [f'절벽이던 자리를 모두 전투맵 언덕으로 ({len(full)}칸)']


def post_cliff_forest(d, region, meta, keep, rnd):
    """단양: 원래 절벽 언덕은 그대로, 위는 빈 풀밭이 보이지 않게 숲으로 (원래 나무에 더 채운다)"""
    ox, oy = d['origin']
    hg = d.get('height') or []
    top = {(u + ox, v + oy) for v, r in enumerate(hg) for u, c in enumerate(r) if c != '0'}
    occ = occupied(d, meta, region)
    before = sum(1 for o in d['objects'] if o[0].startswith('stree') and (o[1], o[2]) in top)
    placed = scatter(d, top, ['stree_pine_1', 'stree_broad_1', 'stree_pine_0', 'stree_broad_0'], rnd, keep, occ, gap=2, n=400)
    return [f'원래 절벽 언덕 그대로, 위는 숲 (나무 {before} → {before + len(placed)})']


def post_fence(d, region, meta, keep, rnd, res=None):
    """단양: 마을 둘레를 울타리로. 언덕과 겹치는 변은 안으로 당기고, 밖으로 이어지는 길·사람 자리에서만 끊는다"""
    ox, oy = d['origin']
    us, vs = [], []
    for o in d['objects']:
        if o[0].startswith(region + '_') and '_field_' not in o[0] and not o[0].endswith(('_u', '_v')):
            us.append(o[1]); vs.append(o[2])
    if not us:
        return []
    W2, H2 = 2 * d['size'][0], 2 * d['size'][1]     # 사람이 걷는 범위 바깥 3칸을 두른다
    L, R = max(ox + 1, min(min(us) - 5, -3)), min(ox + len(d['ground'][0]) - 2, max(max(us) + 5, W2 + 2))
    B, F = max(oy + 1, min(min(vs) - 5, -3)), min(oy + len(d['ground']) - 2, max(max(vs) + 5, H2 + 2))
    hill = set()
    for o in d['objects']:
        if o[0].startswith('sc_knoll'):
            hill |= {(o[1] - 1 + a, o[2] - 1 + b) for a in range(-1, 3) for b in range(-1, 3)}
    hg = d.get('height') or []                        # 절벽(높은 땅)과 그 앞 한 칸에서도 멈춘다
    for v, r in enumerate(hg):
        for u, c in enumerate(r):
            if c != '0':
                hill |= {(u + ox + a, v + oy + b) for a in (-1, 0, 1) for b in (-1, 0, 1)}
    gch = lambda u, v: d['ground'][v - oy][u - ox] if 0 <= v - oy < len(d['ground']) and 0 <= u - ox < len(d['ground'][0]) else '#'
    occ = occupied(d, meta, region)

    main = set()
    if res and res.get('main'):
        ax, at = res['main']
        main = {(u, at + k) for u in range(ox, ox + 999) for k in (0, 1)} if ax == 'u' else {(at + k, v) for v in range(oy, oy + 999) for k in (0, 1)}
    line_cells = set()

    def gap(c, out):
        """큰길이 지나는 칸이나 사람이 서는 칸만 틈 (건물 위도 틈)"""
        hit_building = any(o for o in d['objects'] if o[0].startswith(region + '_') and (o[1], o[2]) == c)
        return c in main or c in keep or hit_building or c in hill      # 언덕에 닿으면 거기서 멈춘다
    fam = f'{region}_fence'
    lines = []
    sides = [([(u, B) for u in range(L, R + 1)], (0, -1)), ([(u, F) for u in range(L, R + 1)], (0, 1)),
             ([(L, v) for v in range(B + 1, F)], (-1, 0)), ([(R, v) for v in range(B + 1, F)], (1, 0))]
    for cells, (du, dv) in sides:
        run = []
        for c in cells + [None]:
            if c is not None and not gap(c, (c[0] + du, c[1] + dv)):
                run.append(c)
                line_cells.add(c)
                continue
            if len(run) >= 1:
                lines.append([fam, list(run[0]), list(run[-1])])
            run = []
    d['walls'] = d.get('walls', []) + lines
    d['objects'] = [o for o in d['objects'] if not (o[0].startswith('stree') and (o[1], o[2]) in line_cells)]
    return [f'마을 둘레 울타리 {len(lines)}줄 (언덕에 닿으면 멈춤, 큰길·사람 자리만 틈, 줄에 걸린 나무는 치움)']


def hermit(src, meta, region, keep, roles, rnd):
    """융중: 숲 속 외딴 초가. 원래 길·밭은 살리고, 원래 집들이 있던 자리에 울타리를 치고 안에 초가·작은 밭·우물.
    길 둘레 2칸과 울타리 앞(화면 아래쪽)은 나무를 비워 내려다보기 좋게"""
    d = json.loads(json.dumps(src))
    ox, oy = d['origin']
    W, H = len(d['ground'][0]), len(d['ground'])
    houses = [o for o in d['objects'] if o[0].startswith(('house', 'hall', 'tavern'))]
    d['objects'] = [o for o in d['objects'] if not o[0].startswith(('house', 'hall', 'tavern', 'market'))]
    d['walls'] = [w for w in d.get('walls', []) if w[0] == 'swall']
    paths = {(u + ox, v + oy) for v, r in enumerate(d['ground']) for u, c in enumerate(r) if c == 'd'}
    cu = round(sum(o[1] for o in houses) / len(houses)) if houses else d['size'][0]
    cv = round(sum(o[2] for o in houses) / len(houses)) if houses else d['size'][1]
    keepr = {(u + a, v + b) for u, v in keep for a in (-1, 0, 1) for b in (-1, 0, 1)}
    fields0 = set()
    for o in d['objects']:                            # 원래 밭은 피한다 (나무는 울타리 안이면 치운다)
        if o[0].startswith('field'):
            fields0 |= tiles_rect(o[1] - 2, o[2] - 2, 5, 5)
    best = None
    fu, fv = meta['cottage_3']['footprint']
    for du in range(-10, 11):
        for dv in range(-10, 11):
            L, B = cu - 6 + du, cv - 5 + dv
            R, F = L + 12, B + 10
            house = tiles_rect(L + 2, B + 1, fu, fv)
            field = tiles_rect(L + 7, B + 5, 4, 4)
            box = tiles_rect(L, B, R - L + 1, F - B + 1)
            if house & keepr or field & keepr or house & paths or field & paths or box & fields0:
                continue
            sc = abs(du) + abs(dv)
            if best is None or sc < best[0]:
                best = (sc, L, B, R, F)
    _, L, B, R, F = best
    box = tiles_rect(L - 1, B - 1, R - L + 3, F - B + 3)
    d['objects'] = [o for o in d['objects'] if not (o[0].startswith('stree') and (o[1], o[2]) in box)]
    d['objects'].append(place_obj(f'{region}_cottage_3', L + 2, B + 1, fu, fv))
    fld = 'field_paddy' if 'field_paddy' in meta else 'field_dry'
    d['objects'].append(place_obj(f'{region}_{fld}', L + 7, B + 5, 4, 4))
    d['objects'].append(['well', L + 3, B + 7, 0, -8])
    # 울타리 문은 가장 가까운 원래 길 쪽 변의 가운데
    near = min(paths, key=lambda c: abs(c[0] - (L + R) / 2) + abs(c[1] - (B + F) / 2)) if paths else ((L + R) // 2, F + 5)
    fam = f'{region}_fence'
    if abs(near[1] - F) <= abs(near[0] - R):
        gate = ((L + R) // 2, F)
        lines = [[fam, [L, B], [R, B]], [fam, [L, B + 1], [L, F]], [fam, [R, B + 1], [R, F]],
                 [fam, [L + 1, F], [gate[0] - 1, F]], [fam, [gate[0] + 1, F], [R - 1, F]]]
        link = {(gate[0], v) for v in range(F, F + 40)}
    else:
        gate = (R, (B + F) // 2)
        lines = [[fam, [L, B], [R, B]], [fam, [L, B + 1], [L, F]], [fam, [L + 1, F], [R, F]],
                 [fam, [R, B + 1], [R, gate[1] - 1]], [fam, [R, gate[1] + 1], [R, F - 1]]]
        link = {(u, gate[1]) for u in range(R, R + 40)}
    d['walls'] += lines
    # 문에서 원래 길까지 잇는다 (닿을 때까지만)
    link_cells = []
    for c in sorted(link, key=lambda c: abs(c[0] - gate[0]) + abs(c[1] - gate[1])):
        if c in paths and link_cells:
            break
        link_cells.append(c)
    set_ground(d, link_cells, 'd')
    paths |= set(link_cells)
    more = 0                                          # 울타리 앞 트인 곳·길가에 밭 두 판 더
    for bu in range(R + 2, R + 12, 5):
        for bv in range(B, F + 8, 5):
            area = tiles_rect(bu, bv, 4, 4)
            if more < 2 and not (area & paths) and not (area & keepr) and not (area & fields0):
                d['objects'].append(place_obj(f'{region}_{fld}', bu, bv, 4, 4))
                fields0 |= tiles_rect(bu - 1, bv - 1, 6, 6)
                more += 1
    clear = {(u + a, v + b) for u, v in paths for a in range(-2, 3) for b in range(-2, 3)}
    front = tiles_rect(L - 2, B - 2, R - L + 12, F - B + 12)     # 울타리와 그 앞(화면 아래쪽)
    occ = occupied(d, meta, region) | clear | front
    allc = [(u, v) for u in range(ox + 1, ox + W - 1) for v in range(oy + 1, oy + H - 1)]
    trees = ['stree_bamboo_0', 'stree_pine_0', 'stree_bamboo_1', 'stree_pine_1', 'stree_bamboo_thin', 'stree_broad_0']
    scatter(d, allc, trees, rnd, keep, occ, gap=3, n=150)
    return {'ok': True, 'score': 0, 'map': d, 'roles': {}, 'blocks': [], 'missing': [], 'covered': [], 'fence': 1,
            'main': None, 'cross': None, 'bounds': (L, R, B, F), 'grow': 0,
            'notes_extra': ['융중: 원래 길·밭 그대로, 숲 속 울타리 안 초가·작은 밭·우물, 길 둘레와 울타리 앞은 나무를 비움']}


def keep_layout(src, meta, region, keep, roles, rnd, cfg):
    """원래 배치를 살리고 집만 지역 그림으로 (강하 나루·융중·단양). erase_d: 대각선으로 그어진 흙 띠 지우기,
    rename: 옛 그림 → 지역 그림 이름, thin_trees: 길에서 이 칸 안의 나무를 치워 덜 빽빽하게"""
    d = json.loads(json.dumps(src))
    if cfg.get('erase_d'):
        d['ground'] = [r.replace('d', '.') for r in d['ground']]
    rename = cfg.get('rename') or {}
    out = []
    for o in d['objects']:
        n = o[0]
        if n.startswith(stage.OLD_HOUSE):
            out.append([f'{region}_{old_to_region(n, rename)}'] + o[1:])
        else:
            out.append(o)
    d['objects'] = out
    notes = ['원래 배치 그대로, 집만 지역 그림']
    if cfg.get('erase_d'):
        notes.append('흙 띠 지움')
    k = cfg.get('thin_trees')
    if k:
        ox, oy = d['origin']
        road = {(u + ox, v + oy) for v, r in enumerate(d['ground']) for u, c in enumerate(r) if c == 'd'}
        near = {(u + a, v + b) for u, v in road for a in range(-k, k + 1) for b in range(-k, k + 1)}
        n0 = len(d['objects'])
        d['objects'] = [o for o in d['objects'] if not (o[0].startswith('stree') and (o[1], o[2]) in near)]
        notes.append(f'길 {k}칸 안 나무 {n0 - len(d["objects"])}그루 치움')
    return {'ok': True, 'score': 0, 'map': d, 'roles': {}, 'blocks': [], 'missing': [], 'covered': [], 'fence': 0,
            'main': None, 'cross': None, 'bounds': None, 'grow': 0, 'notes_extra': notes}


def old_to_region(n, rename=None):
    """옛 집 그림 → 비슷한 크기의 지역 그림 (오른쪽 아래를 보던 _x·hall_v는 반전판)"""
    rename = rename or {}
    base = n[:-2] if n.endswith('_x') else n
    if base in rename:
        nm = rename[base]
    elif n.startswith('house_thatch'):
        nm = 'cottage_4'
    elif n.startswith('tavern'):
        nm = 'tavern'
    elif n.startswith('hall'):
        nm = 'mansion_1'
    elif n.startswith('granary'):
        nm = 'cottage_2'
    else:
        nm = 'mansion_4'
    return nm + ('_r' if n.endswith('_x') or n == 'hall_v' else '')


# 장면이 기대던 옛 집 → 들어갈 지역 그림 후보 (앞이 큰 것부터)
ANCHOR_SWAP = {'hall': ['yamen', 'mansion_2', 'mansion_1', 'mansion_4'], 'house_tile': ['mansion_4', 'mansion_1', 'cottage_4'],
               'house_thatch': ['cottage_4', 'cottage_3', 'cottage_1'], 'tavern': ['tavern', 'cottage_1', 'cottage_4'],
               'granary': ['cottage_4', 'cottage_3'], 'house': ['cottage_4']}


def stage_pass(d, src, region, meta, res, cfg):
    """장면 무대 지키기: 장면이 기대던 옛 집을 그 자리에 되살리고, 선 자리의 사람을 앞에서 가리는 물건을 치우고,
    선 자리 둘레 한 칸을 길 바닥으로"""
    spots, walks, by = stage.gather_stage(SRC_SCENES, d.get('_id'))
    if not spots:
        return []
    sp = stage.Sprites(os.path.join(ROOT, 'assets/graphics'))
    notes = []
    role_rects = set()
    for p in (res.get('roles') or {}).values():
        role_rects |= tiles_rect(p['bu'] - 1, p['bv'] - 1, p['fu'] + 2, p['fv'] + 2)
    stand = {(u + a, v + b) for u, v in spots for a in (-1, 0, 1) for b in (-1, 0, 1)} | walks
    wall_cells = set()
    for w in d.get('walls', []):
        (a0, b0), (a1, b1) = w[1], w[2]
        wall_cells |= {(a, b) for a in range(min(a0, a1), max(a0, a1) + 1) for b in range(min(b0, b1), max(b0, b1) + 1)}
    if cfg.get('mode') != 'keep':
        back = 0
        for o in stage.anchors(src['objects'], spots, sp):
            flip = '_r' if o[0].endswith('_x') or o[0] == 'hall_v' else ''
            cands = ANCHOR_SWAP[next(k for k in ANCHOR_SWAP if o[0].startswith(k))]
            new = None
            for nm in cands:
                fu, fv = meta[nm]['footprint']
                if flip:
                    fu, fv = fv, fu
                for du, dv in ((0, 0), (-1, 0), (0, -1), (-1, -1), (-2, 0), (0, -2), (-2, -1), (-1, -2)):
                    bu, bv = o[1] - fu // 2 + du, o[2] - fv // 2 + dv
                    fp = tiles_rect(bu, bv, fu, fv)
                    ring = tiles_rect(bu - 1, bv - 1, fu + 2, fv + 2)
                    if fp & stand or ring & role_rects or ring & wall_cells:
                        continue
                    new = place_obj(f'{region}_{nm}{flip}', bu, bv, fu, fv)
                    break
                if new:
                    break
            if not new:
                continue
            keepers = []
            for x in d['objects']:
                xf = occupied({'objects': [x]}, meta, region)
                if xf & ring and not x[0].startswith(KEEP_INSIDE + ('sgate',)):
                    continue
                keepers.append(x)
            d['objects'] = keepers + [new]
            role_rects |= ring
            back += 1
        if back:
            notes.append(f'장면이 기대던 옛 집 {back}채를 그 자리에 지역 그림으로')

    def fixed(o):
        n = o[0]
        if n.startswith(('sgate', 'swall', 'palace', 'pwall')) or n.startswith(KEEP_INSIDE):
            return True
        return n.startswith(region + '_') and (o[1], o[2]) in role_rects
    idx = stage.occluders(d['objects'], spots, sp, skip=fixed)
    hid = [d['objects'][i][0] for i in idx]
    d['objects'] = [o for i, o in enumerate(d['objects']) if i not in set(idx)]
    if hid:
        notes.append(f'사람을 가리던 물건 {len(hid)}개 치움 ({", ".join(sorted(set(hid))[:6])})')
    blocked = [i for i, o in enumerate(d['objects']) if not fixed(o) and occupied({'objects': [o]}, meta, region) & (spots | walks)]
    if blocked:
        d['objects'] = [o for i, o in enumerate(d['objects']) if i not in set(blocked)]
        notes.append(f'선 자리·걷는 칸 위 물건 {len(blocked)}개 치움')
    if cfg.get('mode') != 'keep':
        ox0, oy0 = src['origin']
        old_g = lambda u, v: src['ground'][v - oy0][u - ox0] if 0 <= v - oy0 < len(src['ground']) and 0 <= u - ox0 < len(src['ground'][0]) else '.'
        for rect in stage.plazas(by):
            cells = tiles_rect(*rect)
            used = [old_g(u, v) for u, v in cells if old_g(u, v) not in '.~#']
            glyph = max(set(used), key=used.count) if used else (cfg.get('avenue') or 'd')
            occ = occupied(d, meta, region)
            set_ground(d, cells - occ, glyph)
        notes.append('장면 무대마다 사람들이 선 곳을 네모 광장으로 (옛 맵의 그 자리 바닥 글자)')
    left = stage.occluders(d['objects'], spots, sp)
    if left:
        notes.append(f'아직 가리는 것: {sorted({d["objects"][i][0] for i in left})}')
    return notes


# ---------------------------------------------------------------- 탐험·사인 고치기

def patch_scenario(root, res, exits, signs, region):
    """역할 건물 출구를 새 문 칸으로, Object를 새 건물 칸으로 (대장간 출구는 대장간 앞 → 마당 맵, smithy_yard.py)"""
    scn = os.path.join(root, 'assets/scenario/hero')
    notes = []
    moved = res['map'].pop('_moved_gates', [])
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
                if pr:
                    e['At'] = list(pr['door'])
                    if 'Object' in e:
                        e['Object'] = [pr['bu'] + pr['fu'] // 2, pr['bv'] + pr['fv'] // 2]
                    notes.append(f'{f}/{k}: {e.get("Name")} 출구 → {list(pr["door"])}')
                new_exits.append(e)
            for e in new_exits:              # 밀려난 성문의 출구
                for (gu, gv), (du, dv) in moved:
                    if abs(e['At'][0] - gu) <= 1 and abs(e['At'][1] - gv) <= 1:
                        e['At'] = [e['At'][0] + du, e['At'][1] + dv]
                        notes.append(f'{f}/{k}: {e.get("Name")} 출구 → 밀려난 성문 {e["At"]}')
            a['Exits'] = new_exits
            for s in a.get('Signs') or []:
                if s.get('Name') == '저택' and 'mansion' in res['roles']:
                    s['At'] = list(res['roles']['mansion']['door'])
        json.dump(d, open(path, 'w'), ensure_ascii=False, indent=1)
    return notes


if __name__ == '__main__':
    root, map_id, region = sys.argv[1], sys.argv[2], sys.argv[3]
    ROOT = root
    seed = int(sys.argv[4]) if len(sys.argv) > 4 else 0
    res, keep, exits, signs, roles, uses = generate(map_id, region, seed)
    cfg = CFG.get(map_id, {})
    meta = load_meta(region)
    rnd = random.Random(seed + 7)
    extra = list(res.get('notes_extra', []))
    d = res['map']
    if cfg.get('outside'):
        extra += post_outside(d, res, region, meta, keep, rnd)
    if cfg.get('ruin'):
        extra += post_ruin(d, res, region, meta, keep, rnd)
    if cfg.get('hills'):
        extra += post_hills(d, region, meta, keep, rnd)
    if cfg.get('cliff_forest'):
        extra += post_cliff_forest(d, region, meta, keep, rnd)
    if cfg.get('fence_around'):
        extra += post_fence(d, region, meta, keep, rnd, res)
    d.pop('_hill_cells', None)
    d['_id'] = map_id
    extra += stage_pass(d, json.load(open(os.path.join(SRC_MAPS, map_id + '.json'))), region, meta, res, cfg)
    d.pop('_id', None)
    json.dump(res['map'], open(os.path.join(root, 'assets/graphics/maps', map_id + '.json'), 'w'), ensure_ascii=False, indent=1)
    notes = extra + patch_scenario(root, res, exits, signs, region)
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
