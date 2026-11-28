"""시나리오에 나오는 성·마을 장면 맵을 지역 건물로 다시 깐 재설계 안 (미리보기 루트에만 쓴다).

- 옛 집·관아 → 지역 건물. 관청·저택은 기와 담장+대문, 초가는 흙담+사립문 / 울타리 / 없음을 번갈아.
- 그 맵을 쓰는 모든 탐험 지역의 사람·출구 칸은 비워 둔다 (둘레 한 칸까지).
- 놀이 영역(Size) 안에만, 길(포장)·물·다른 바닥 위에는 놓지 않는다. 흙(d)이 영역의 절반이 안 되면 흙도 길로 본다.
- 거리에 실내 대장간 출구가 있던 맵: 그 출구 자리 가까이에 바깥 대장간을 세우고, 대장장이·견습공을 그 앞으로
  옮기고, 실내 대장간 출구·지역을 뺀다 (루트의 시나리오 사본만).

  python3 redesign.py ROOT     -> ROOT/assets/graphics/maps/<id>.json, ROOT/assets/scenario/hero/explore*.json, redesign.json
"""
import glob
import json
import math
import os
import sys

HERE = os.path.dirname(os.path.abspath(__file__))
ROOT = sys.argv[1]
G = os.path.join(ROOT, 'assets', 'graphics')
SCN = os.path.join(ROOT, 'assets', 'scenario', 'hero')
SRC = '/Users/younwookim/orca/mhdemo1/ver5/assets/graphics/maps'
SRC_SCN = '/Users/younwookim/orca/mhdemo1/ver5/assets/scenario/hero'
FP = json.load(open(os.path.join(G, 'scene.json')))['footprints']

REG = {}
for r in os.listdir(os.path.join(HERE, 'reg')):
    p = os.path.join(HERE, 'reg', r, 'meta.json')
    if os.path.exists(p):
        REG[r] = json.load(open(p))
        lp = os.path.join(HERE, 'lines', r, 'lines.json')
        if os.path.exists(lp):
            REG[r].update({k[len(r) + 1:]: v for k, v in json.load(open(lp)).items()})

TOWNS = [  # 맵, 지역, 밭/논 몇 판
    ('S_zhuo', 'huabei', 1), ('S_pingyuan', 'huabei', 1), ('S_xindu', 'huabei', 1), ('S_ye', 'huabei', 0),
    ('S_beihai', 'huabei', 0),
    ('S_luoyang', 'zhongyuan', 0), ('S_xudu', 'zhongyuan', 0), ('S_runan', 'zhongyuan', 1), ('S_road_town', 'zhongyuan', 2),
    ('S_xiaopei', 'huainan', 1), ('S_xuzhou_town', 'huainan', 1), ('S_xiapi', 'huainan', 0), ('S_xiapi_inner', 'huainan', 0),
    ('S_xiangyang', 'huainan', 0), ('S_xinye', 'huainan', 1), ('S_mangdang', 'huainan', 2), ('S_vil_peace', 'huainan', 1),
    ('S_jiangling', 'jiangnan', 1), ('S_jiangxia_harbor', 'jiangnan', 1),
]
DOOR_KIND = {'S_tool_shop': 'shop', 'S_tavern': 'tavern', 'S_zhuo_office': 'yamen', 'S_xiaopei_office': 'yamen',
             'S_council': 'yamen', 'S_liu_house': 'mansion_1', 'S_mansion': 'mansion_1'}
REGION_KO = {'huabei': '화북', 'zhongyuan': '중원', 'huainan': '회남', 'jiangnan': '강남', 'bashu': '파촉'}

OLD = ('house_thatch', 'house_thatch_x', 'house_tile', 'house_tile_x', 'tavern', 'hall_u', 'hall_v')
SMALL = ('wine_jar', 'grain_sack', 'water_jar', 'bench', 'table_wine', 'straw_mat', 'lantern', 'flagstones_0',
         'flagstones_1', 'weapon_rack', 'armor_stand', 'notice_post', 'notice_board', 'stree_bush_0', 'stree_bush_1',
         'water_bucket', 'grain_cart', 'grain_cart_v', 'brazier', 'command_flag', 'taihu_rock')


def foot(name, u, v, dx=0, dy=0):
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


def cell_tiles(c, ring=1):
    """소타일 (x, y)와 둘레 ring칸 (탐험 지역의 At은 소타일 좌표)"""
    x, y = c
    return {(a, b) for a in range(x - ring, x + ring + 1) for b in range(y - ring, y + ring + 1)}


def find(u0, v0, fu, fv, pad, bounds, hard, taken, wall, ok_ground, rmax=60, accept=None):
    pu0, pv0, pu1, pv1 = pad
    for r in range(0, rmax):
        for bu in range(u0 - r, u0 + r + 1):
            for bv in range(v0 - r, v0 + r + 1):
                if max(abs(bu - u0), abs(bv - v0)) != r:
                    continue
                a0, b0, a1, b1 = bu + pu0, bv + pv0, bu + fu + pu1, bv + fv + pv1
                if a0 < bounds[0] or b0 < bounds[2] or a1 > bounds[1] or b1 > bounds[3]:
                    continue
                tiles = {(a, b) for a in range(a0, a1) for b in range(b0, b1)}
                ring = {(a, b) for a in range(a0 - 1, a1 + 1) for b in range(b0 - 1, b1 + 1)}
                near_wall = ring & wall if pad != (0, 0, 0, pad[3]) else set()   # 담장 줄만 기존 줄과 한 칸 띄운다
                if tiles & hard or ring & taken or near_wall or not all(ok_ground(a, b) for a, b in tiles):
                    continue
                if accept and not accept(bu, bv):
                    continue
                return bu, bv, tiles
    return None


def uses(map_id):
    """이 맵을 쓰는 (탐험 파일, 지역 키)"""
    out = []
    for f in sorted(glob.glob(os.path.join(SRC_SCN, 'explore*.json'))):
        d = json.load(open(f))
        for k, a in (d.get('Areas') or {}).items():
            if a.get('Map') == map_id:
                out.append((os.path.basename(f), k))
    return out


def convert(map_id, region, nfields):
    d = json.load(open(os.path.join(SRC, map_id + '.json')))
    ox, oy = d['origin']
    ground = d['ground']
    W, H = d['size']
    play = (0, 2 * W, 0, 2 * H)

    def glyph(u, v):
        r, c = v - oy, u - ox
        return ground[r][c] if 0 <= r < len(ground) and 0 <= c < len(ground[0]) else '#'

    inside = [glyph(u, v) for u in range(play[0], play[1]) for v in range(play[2], play[3])]
    soil_open = inside.count('d') >= len(inside) * .5
    ok_ground = (lambda a, b: glyph(a, b) in ('.', 'd')) if soil_open else (lambda a, b: glyph(a, b) == '.')

    wall = set()
    for w in d.get('walls', []):
        (a0, b0), (a1, b1) = w[1], w[2]
        for a in range(min(a0, a1), max(a0, a1) + 1):
            for b in range(min(b0, b1), max(b0, b1) + 1):
                wall.add((a, b))
    # 놓는 범위: 성벽이 있으면 성벽 안 전체, 없으면 땅 가장자리 2칸 안쪽 (사람이 걷는 곳은 놀이 영역이지만 성 안
    # 나머지 땅의 건물도 성의 풍경)
    g0u, g1u, g0v, g1v = ox + 2, ox + len(ground[0]) - 2, oy + 2, oy + len(ground) - 2
    if wall:
        us, vs = [a for a, b in wall], [b for a, b in wall]
        bounds = (min(us) + 1 if min(us) >= g0u - 2 else g0u, min(max(us), g1u) if max(us) > play[1] else g1u,
                  min(vs) + 1 if min(vs) >= g0v - 2 else g0v, min(max(vs), g1v) if max(vs) > play[3] else g1v)
    else:
        bounds = (g0u, g1u, g0v, g1v)
    olds = [o for o in d['objects'] if o[0] in OLD]
    keep = [o for o in d['objects'] if o[0] not in OLD]
    hard = set(wall)
    for o in keep:
        if o[0] not in SMALL:
            hard |= foot(o[0], *o[1:3], *(o[3:5] if len(o) >= 5 and not isinstance(o[3], dict) else (0, 0)))
    # 사람·출구 칸 (둘레 한 칸까지) 비우기, 실내 대장간 출구는 대장간 자리 표적으로
    used = uses(map_id)
    smith_target = None
    doors = {}            # 건물 종류 -> (표적 칸, [(파일, 지역, 출구 To)])
    for f, k in used:
        areas = json.load(open(os.path.join(SRC_SCN, f)))['Areas']
        a = areas[k]
        for e in a.get('Exits') or []:
            kind = DOOR_KIND.get((areas.get(e.get('To')) or {}).get('Map'))
            if kind:
                doors.setdefault(kind, [tuple(e['At']), []])[1].append((f, k, e['To']))
        for p in (a.get('People') or {}).values():
            if 'At' in p:
                hard |= cell_tiles(p['At'])
        for e in a.get('Exits') or []:
            if e.get('To') == 'weapons' and smith_target is None:
                smith_target = e['At']
                continue
            hard |= cell_tiles(e['At'], 0 if DOOR_KIND.get((areas.get(e.get('To')) or {}).get('Map')) else 1)   # 새 건물 문 앞으로 옮길 출구도, 건물이 못 서면 그대로 쓰도록 비워 둔다
        if 'At' in a:
            hard |= cell_tiles(a['At'])
    # 옛 건물 → 새 건물
    cot = [f'cottage_{k}' for k in (1, 2, 3, 4)]
    big = ['mansion_1', 'mansion_2', 'mansion_4', 'mansion_3']
    plan, nc, nm = [], 0, 0
    extra = ['shop'] + ([] if smith_target else ['smithy'])
    halls = [o for o in olds if o[0].startswith('hall')]
    for i, o in enumerate(sorted(olds, key=lambda o: (not o[0].startswith('hall'), o[2], o[1]))):
        n = o[0]
        if n.startswith('hall'):
            new = 'yamen' if o is halls[0] else big[nm % 4]
            nm += (o is not halls[0])
        elif n == 'tavern':
            new = 'tavern'
        elif n.startswith('house_tile') and extra:
            new = extra.pop(0)
        elif n.startswith('house_tile') and (i % 3 == 0):
            new = big[nm % 4]
            nm += 1
        else:
            new = cot[nc % 4]
            nc += 1
        plan.append((new, o))
    for kind, (at, _) in doors.items():
        for i, (new, o) in enumerate(plan):
            if new == kind:
                del plan[i]
                break
        plan.insert(0, (kind, [kind, at[0], at[1]]))
    if smith_target:
        plan.insert(0, ('smithy', ['smithy', smith_target[0], smith_target[1]]))
    plan.sort(key=lambda p: (p[0] != 'smithy' and p[0] not in doors, -REG[region][p[0]]['footprint'][0] * REG[region][p[0]]['footprint'][1]))
    fields = [k for k in ('field_paddy', 'field_dry') if k in REG[region]]
    cx, cy = (play[0] + play[1]) // 2, (play[2] + play[3]) // 2
    for k in range(nfields):
        plan.append((fields[k % len(fields)], ['field', cx + (6 if k % 2 else -6), play[3] + 2]))
    taken, out, small_hit, lines, report = set(), [], set(), [], []
    smith = None
    nfolk = 0
    moved_doors = {}
    for want, o in plan:
        if want.startswith('cottage'):
            folk_encl = ('ewall', 'fence', None)[nfolk % 3]
            nfolk += 1
        # 자리가 없으면 담장 없이, 그다음 작은 것으로 물러선다
        chain = {'yamen': [('yamen', 'twall'), ('yamen', None), ('mansion_1', 'twall'), ('mansion_1', None)],
                 'mansion_1': [('mansion_1', 'twall'), ('mansion_1', None), ('mansion_4', None)],
                 'mansion_2': [('mansion_2', 'twall'), ('mansion_2', None), ('mansion_4', None)],
                 'mansion_3': [('mansion_3', 'twall'), ('mansion_3', None), ('mansion_4', None)],
                 'mansion_4': [('mansion_4', 'twall'), ('mansion_4', None), ('cottage_4', None)],
                 'shop': [('shop', None), ('mansion_4', None), ('cottage_4', None)],
                 'tavern': [('tavern', None), ('mansion_4', None), ('cottage_1', None)]}.get(want)
        if chain is None and want.startswith('cottage'):
            chain = ([(want, folk_encl)] if folk_encl else []) + [(want, None), ('cottage_4', None)]
        if chain is None:
            chain = [(want, None)]
        best = None
        for new, encl in chain:
            fu, fv = REG[region][new]['footprint']
            u0, v0 = o[1] - fu // 2, o[2] - fv // 2
            front = 2 if new == 'smithy' else 1 if want in doors else 0
            pad = (-1, -1, 1, 3 + (1 if want in doors else 0)) if encl else (0, 0, 0, front)   # 대장간 앞 2칸은 대장장이 자리, 출구 건물 문 앞 1칸
            # 문 앞 칸(대장간은 대장장이 칸)만 놀이 영역 안이면 된다: 건물 몸체는 성 안 나머지 땅에 걸쳐도 됨
            if new == 'smithy' or want in doors:
                gw = 2
                def accept(bu, bv, fu=fu, fv=fv, encl=encl):
                    x, y = (bu + (fu - gw) // 2, bv + fv + 2) if encl else (bu + fu // 2, bv + fv)
                    return play[0] <= x < play[1] - (2 if new == 'smithy' else 0) and play[2] <= y < play[3] - (1 if new == 'smithy' else 0)
            else:
                accept = None
            best = find(u0, v0, fu, fv, pad, bounds, hard, taken, wall, ok_ground, accept=accept)   # 대장장이는 걸어가 말을 걸 수 있는 곳에
            if best:
                break
        if not best:
            report.append(f'자리 없음: {want}')
            continue
        if new != want:
            report.append(f'{want} → {new}')
        bu, bv, tiles = best
        taken |= tiles
        out.append(place_obj(f'{region}_{new}', bu, bv, fu, fv))
        if new == 'smithy':
            smith = (bu, bv, fu, fv)
        door_at = (bu + fu // 2, bv + fv)
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
                out.append(place_obj(f'{region}_{"tgate" if encl == "twall" else "egate"}_u', g0, F, 2, 1))
            door_at = (g0, F + 1)
        if want in doors and want not in moved_doors:
            moved_doors[want] = door_at
            hard |= cell_tiles(door_at)
    kept = []
    for o in keep:
        if o[0] in SMALL and foot(o[0], o[1], o[2]) & taken:
            small_hit.add(o[0])
            continue
        kept.append(o)
    d['objects'] = out + kept
    d['walls'] = d.get('walls', []) + lines
    json.dump(d, open(os.path.join(G, 'maps', map_id + '.json'), 'w'), ensure_ascii=False, indent=1)
    n_old = len(olds)
    n_new = sum(1 for o in out if not o[0].endswith(('_u', '_v')) and '_field_' not in o[0])
    report.insert(0, f'건물 {n_old} → {n_new}, 담장·울타리 줄 {len(lines)}, 뺀 소품 {sorted(small_hit) or "없음"}')
    # 출구를 새 건물 문 앞으로 (루트의 시나리오 사본)
    for kind, door in moved_doors.items():
        for f, k, to in doors[kind][1]:
            path = os.path.join(SCN, f)
            dd = json.load(open(path))
            for e in dd['Areas'][k].get('Exits') or []:
                if e.get('To') == to:
                    e['At'] = list(door)
            json.dump(dd, open(path, 'w'), ensure_ascii=False, indent=1)
        report.append(f'{kind} 출구 → {list(door)}')
    return smith, used, report


def move_smith(smith, used):
    """실내 대장간 → 바깥: 대장장이·견습공을 대장간 앞 칸으로, 거리의 대장간 출구와 대장간 지역을 뺀다"""
    bu, bv, fu, fv = smith
    front = (bu + fu // 2, bv + fv)                         # 대장간 앞 가운데 소타일
    done = []
    for f, k in used:
        path = os.path.join(SCN, f)
        d = json.load(open(path))
        A = d['Areas']
        town = A[k]
        if not any(e.get('To') == 'weapons' for e in town.get('Exits') or []) or 'weapons' not in A:
            continue
        town['Exits'] = [e for e in town['Exits'] if e.get('To') != 'weapons']
        people = town.setdefault('People', {})
        for i, (name, p) in enumerate((A['weapons'].get('People') or {}).items()):
            q = dict(p)
            q['At'] = [front[0] + (0 if i == 0 else 2), front[1] + (0 if i == 0 else 1)]
            q['Face'] = 'SW' if i == 0 else 'SE'
            people[name if name not in people else name + ' (대장간)'] = q
        del A['weapons']
        json.dump(d, open(path, 'w'), ensure_ascii=False, indent=1)
        done.append(f'{f}/{k}')
    return front, done


if __name__ == '__main__':
    summary = []
    for map_id, region, nf in TOWNS:
        smith, used, report = convert(map_id, region, nf)
        moved = None
        if smith and any(True for _ in used):
            front, files = move_smith(smith, used)
            if files:
                moved = {'front': front, 'files': files}
                report.append(f'대장간 바깥으로: 대장장이 소타일 {list(front)}, ' + ', '.join(files))
        name = json.load(open(os.path.join(SRC, map_id + '.json')))['name']
        summary.append({'map': map_id, 'name': name, 'region': region, 'uses': [f'{f}/{k}' for f, k in used],
                        'report': report, 'smith': moved})
        print(map_id, name, REGION_KO[region], '|', ' / '.join(report))
    json.dump(summary, open(os.path.join(ROOT, 'redesign.json'), 'w'), ensure_ascii=False, indent=1)
