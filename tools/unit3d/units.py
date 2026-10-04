"""ver2 unit meshes built from the legacy articulated soldier (base_model.py).

Bodies, weapons and horse follow base_model.model() and the legacy
new_units/build.py poses; attacks use the keyframes in attacks.py. Heads are
left out: each unit function returns a dict with the faces, the model-space head
centre (a hand-drawn 2D head is stamped there), the facial expression and effect
geometry. Thin parts (spear shaft, bow) are thickened to survive at 48px.
"""
import copy
import math
import numpy as np
import attacks as A
import base_model as m

V = m.V
BLUE, LIGHT_BLUE, STEEL, BROWN, SKIN = '#28486e', '#3974a0', '#636971', '#754323', '#efb15c'
GOLD, HEMP, HEMP_DARK = '#d5a123', '#cdb88a', '#8a7650'
GREEN, DARK_GREEN, BLACK, DARK_STEEL, RED = '#2f6b3a', '#1f4a2e', '#2a2d36', '#30343a', '#b91828'
BLACK_STEEL = '#34383f'  # Zhang Fei's armor: dark, but with all four light steps
RAMPS = {HEMP: 'UTSS', HEMP_DARK: 'UUTT', '#976039': 'fghh', '#ede1bd': '5566',
         '#b99553': 'ghkk', '#ede1b8': 'TSSS', '#bacbd6': '4556', '#704022': 'fggh', '#d7bc7c': 'kllm',
         GREEN: 'GHIJ', DARK_GREEN: 'GGHI', BLACK: '1234', BLACK_STEEL: '1234', '#2b2b33': '1234', '#3d3d48': '2334',
         '#15151a': '0011'}
HEAD = [0, .02, 1.84]
SHOULDER = V([-.4, 0, 1.42])  # weapon-arm shoulder
POSE_KEYS = ['bob', 'lean', 'feet', 'knees', 'elbows', 'hands', 'blade']
ATTACK, HIT = 2, 3

# Team-colored materials: BLUE (tunic, hood), LIGHT_BLUE (sash) and the saddle cloth render with the
# faction key ramp (render.RAMPS, factions.TEAM_KEYS). Hero robes (Guan Yu green, Zhang Fei black) stay fixed.
KITS = {
    'infantry': dict(cloth=BLUE, tabard=LIGHT_BLUE, guard=STEEL, shield=True, weapon='sword'),
    'bandit': dict(cloth=HEMP, tabard=LIGHT_BLUE, guard=HEMP_DARK, shield=True, weapon='sword'),
    'spearman': dict(cloth=BLUE, tabard=LIGHT_BLUE, guard=STEEL, shield=False, weapon='spear'),
    'guanyu': dict(cloth=GREEN, tabard=LIGHT_BLUE, guard=DARK_GREEN, shield=False, weapon=None),
    'archer': dict(cloth=BLUE, tabard=LIGHT_BLUE, guard=STEEL, shield=False, weapon=None),
    'rider': dict(cloth=BLUE, tabard=LIGHT_BLUE, guard=STEEL, shield=False, weapon=None),
    'zhangfei': dict(cloth=BLUE, tabard=RED, guard=BLACK_STEEL, shield=False, weapon=None),  # team color, red trim
}
# (inner, tip) distance from the hand along the weapon, for trails and thrust bursts.
WEAPONS = {'sword': (.45, 1.0), 'spear': (.9, 1.75), 'glaive': (1.15, 1.95), 'fan': (.25, .7), 'staff': (.9, 1.35)}


def result(faces, head, row, f, effects=()):
    face = 'tired' if row == 4 else 'hurt' if row == HIT and f >= 2 else 'normal'
    return dict(faces=faces, head=np.asarray(head), face=face, effects=list(effects))


# Hit reaction, pose only (color flashes and particles are added later as effects).
# The blow lands on the third frame: the upper body reels back and the arms drop.
HIT_STEPS = [dict(lean=0), dict(lean=.04), dict(lean=.3, bob=-.03, arms=.5), dict(lean=.14, arms=.3)]
LOOSE_ARMS = dict(hands=[[-.55, .05, .85], [.55, .05, .85]], elbows=[[-.58, .02, 1.1], [.58, .02, 1.1]],
                  blade=[-.1, -.2, -.9])


def hit_lean(p, f):
    p['lean'], p['bob'] = HIT_STEPS[f]['lean'], HIT_STEPS[f].get('bob', 0)


def transform(p):
    def T(q):
        x, y, z = q
        z += p['bob']
        return V([x, y + p['lean']*(z - .45), z])
    return T


def lerp_pose(a, b, t):
    p = copy.deepcopy(a)
    for k in POSE_KEYS:
        p[k] = (np.array(a[k])*(1 - t) + np.array(b[k])*t).tolist()
    return p


def lerp_key(a, b, t):
    keys = set(a) | set(b)
    return {k: (np.array(a.get(k, 0), float)*(1 - t) + np.array(b.get(k, 0), float)*t) for k in keys}


def soldier(p, kit, weapon=True):
    """Headless soldier; returns (faces, head centre)."""
    m.meshes = []
    k = KITS[kit]
    cloth, guard = k['cloth'], k['guard']
    T = transform(p)
    for i in range(2):
        hip = T([(-.24 if i == 0 else .24), 0, .77])
        knee, foot = V(p['knees'][i]), V(p['feet'][i])
        m.rod(hip, knee, .16, cloth)
        m.rod(knee, foot, .13, guard)
        m.ell(foot + [0, -.035, 0], [.135, .2, .105], BROWN)
    if k.get('robe'):  # long robe over the thighs; shins and feet show below
        m.ell(T([0, -.01, .66]), [.42, .31, .42], k['robe'])
    m.ell(T([0, 0, .92]), [.43, .27, .31], cloth)
    m.ell(T([0, 0, 1.26]), [.4, .27, .38], cloth)
    m.ell(T([0, -.035, 1.3]), [.36, .275, .27], guard)
    if k.get('belly'):  # stout figure (Dong Zhuo): a round belly over the sash
        m.ell(T([0, -.14, 1.0]), [.62, .5, .46], k.get('robe') or cloth)
    if k.get('pack'):  # merchant's bundle on the back
        m.box(T([0, .4, 1.3]), [.72, .42, .86], k['pack'])  # tall enough to show over the shoulders
        for z in (1.08, 1.5):
            m.box(T([0, .4, z]), [.75, .45, .06], BROWN)
    m.box(T([0, 0, 1.01]), [.79, .56, .105], k['tabard'])
    m.box(T([0, -.29, .8]), [.13, .05, .36], k['tabard'])
    m.rod(T([0, 0, 1.48]), T([0, 0, 1.68]), .14, SKIN)
    for i in range(2):
        sh, el, ha = T([(-.4 if i == 0 else .4), 0, 1.42]), T(p['elbows'][i]), T(p['hands'][i])
        m.ell(sh, [.19, .22, .19], guard)
        m.rod(sh, el, .14, cloth)
        m.rod(el, ha, .12, cloth)
        m.ell(ha, [.125]*3, SKIN)
    if weapon and k['weapon']:
        WEAPON_DRAW[k['weapon']](p, T)
        if k.get('twin'):  # second sword in the off hand, mirrored
            q = copy.deepcopy(p)
            q['hands'][0] = p['hands'][1]
            # held out to the side so it clears the body; mirrors the main blade while attacking
            q['blade'] = [-p['blade'][0] + .3, p['blade'][1], p['blade'][2]] if p['row'] == ATTACK else [.8, -.3, .5]
            WEAPON_DRAW[k['weapon']](q, T)
        if k['shield']:
            center = T(p['hands'][1]) + V([0, -.09, 0])
            m.ell(center, [.3, .07, .39], BROWN)
            m.ell(center + V([0, -.07, 0]), [.09, .055, .1], STEEL)
    return m.meshes, T(HEAD)


def grip(p, T):
    """Weapon hand, direction and the flat side of the weapon head."""
    hand = T(p['hands'][0])
    d = A.unit(p['blade'])
    if p['row'] != ATTACK:
        side = V([1, 0, 0])
    else:
        side = np.cross(V([1, 0, 0]), d) if abs(d[0]) < .9 else V([0, 0, 1])
    side -= d*np.dot(side, d)
    return hand, d, side/np.linalg.norm(side)


def sword(p, T):
    hand, d, side = grip(p, T)
    m.rod(hand - d*.12, hand + d*.1, .045, BROWN)
    base = hand + d*.15
    m.rod(base - side*.18, base + side*.18, .045, GOLD)
    tip, mid = base + d*.85, base + d*.55
    m.poly([base - side*.07, mid - side*.08, tip, mid + side*.13, base + side*.07], '#bad9f1')
    m.poly([base, mid, tip, mid + side*.13, base + side*.07], '#e5f4ff')


def spear(p, T):
    hand, d, side = grip(p, T)
    m.rod(hand - d*.95, hand + d*1.45, .045, '#966131')
    base = hand + d*1.42
    m.poly([base - side*.1, hand + d*1.75, base + side*.1, base - d*.06], '#d6e7ec')
    m.ell(base - d*.06, [.07]*3, RED)


def glaive(p, T):
    glaive_at(*grip(p, T))


def glaive_at(hand, d, side):
    """Green Dragon crescent blade: long shaft, broad curved head, gold collar, red tassel."""
    m.rod(hand - d*.8, hand + d*1.2, .05, BROWN)
    b = hand + d*1.15
    ts = np.linspace(0, 1, 7)
    bulge = [math.sin(math.pi*t) for t in ts]
    inner = [b + d*.95*t + side*(.12*s - .05*(1 - t)) for t, s in zip(ts, bulge)]
    edge = [b + d*.95*t + side*.28*s for t, s in zip(ts, bulge)]
    outer = [b + d*.95*t + side*.36*s for t, s in zip(ts, bulge)]
    strip(inner, edge, '#bad9f1')
    strip(edge, outer, '#e5f4ff')
    m.rod(b - d*.08, b + d*.04, .075, GOLD)
    m.ell(b - d*.12 - side*.08, [.08]*3, RED)


def strip(left, right, color):
    """Concave blade outline as convex quads (the rasterizer fills convex polygons)."""
    for i in range(len(left) - 1):
        m.poly([left[i], left[i+1], right[i+1], right[i]], color)


def fan(p, T):
    """Feather fan: short handle, broad convex fan head (strategists)."""
    hand, d, side = grip(p, T)
    m.rod(hand - d*.08, hand + d*.22, .04, BROWN)
    base = hand + d*.2
    pts = [base] + [base + (d*math.cos(a) + side*math.sin(a))*.62 for a in np.linspace(-.8, .8, 7)]
    m.poly(pts, '#ede1bd')
    m.ell(base, [.06]*3, GOLD)


def staff(p, T):
    """Taoist staff: long shaft with a gold ring head."""
    hand, d, side = grip(p, T)
    m.rod(hand - d*.7, hand + d*1.05, .045, '#966131')
    head = hand + d*1.22
    for a in np.linspace(0, 2*math.pi, 7)[:-1]:
        m.ell(head + (d*math.cos(a) + side*math.sin(a))*.15, [.05]*3, GOLD)


WEAPON_DRAW = {'sword': sword, 'spear': spear, 'glaive': glaive, 'fan': fan, 'staff': staff}


def weapon_edge(p, weapon, reach=1.0):
    """(inner, tip) of the weapon head in model space; reach stretches the tip."""
    inner, tip = WEAPONS[weapon]
    hand = transform(p)(p['hands'][0])
    d = A.unit(p['blade'])
    return hand + d*inner, hand + d*tip*reach


def pose(row, f):
    return copy.deepcopy(m.poses[row*4 + f])


def sword_pose(k):
    p = pose(0, 0)
    p['row'] = ATTACK
    p['hands'][0] = list(k['hand'])
    p['elbows'][0] = list(k.get('elbow', (SHOULDER + V(k['hand']))/2 + [-.15, 0, -.05]))
    p['blade'] = list(k['blade'])
    p['lean'], p['bob'] = float(k.get('lean', 0)), float(k.get('bob', 0))
    A.stance(p, k)
    return p


def sword_effects(style, f, weapon):
    spec = A.SWORD[style]
    if f < 2:
        return []
    if spec['effect'] == 'arc':
        keys = A.sample(A.swing_path(spec), 4, lerp_key)
        if f == 3:
            keys = keys[-6:] + A.sample([spec['frames'][2], spec['frames'][3]], 3, lerp_key)[1:]
        reach = A.TRAIL_REACH if weapon == 'sword' else 1.1
        pairs = [weapon_edge(sword_pose(k), weapon, reach) for k in keys]
        return [A.arc(pairs, alpha=1 if f == 2 else .45)]
    p = sword_pose(spec['frames'][f])
    _, tip = weapon_edge(p, weapon)
    return A.pierce(tip, p['blade'], transform(p)(p['hands'][0]), alpha=1 if f == 2 else .6, burst=f == 2)


def hit_pose(f, base):
    p = copy.deepcopy(base)
    p['row'] = HIT
    hit_lean(p, f)
    k = HIT_STEPS[f].get('arms', 0)
    for key, target in LOOSE_ARMS.items():
        p[key] = (np.array(p[key])*(1 - k) + np.array(target)*k).tolist()
    return p


def foot_unit(kit):
    def build(row, f, style=None):
        style = style or A.SELECTED.get(kit, 'sweep')
        effects = []
        if row == ATTACK:
            p = sword_pose(A.SWORD[style]['frames'][f])
            effects = sword_effects(style, f, KITS[kit]['weapon']) if KITS[kit]['weapon'] else []
        elif row == HIT:
            p = hit_pose(f, pose(0, 0))
        else:
            p = pose(row, f)
        faces, head = soldier(p, kit)
        return result(faces, head, row, f, effects)
    return build


infantry = foot_unit('infantry')
bandit = foot_unit('bandit')
spearman = foot_unit('spearman')


def bow(h, nock, axis, belly, bend=1.0, width=.045):
    pts = [h + belly*.18*bend*math.sin(math.pi*t) + axis*.67*(1 - 2*t) for t in np.linspace(0, 1, 13)]
    for a, b in zip(pts, pts[1:]):
        m.rod(a, b, width, '#976039')
    for end in [pts[0], pts[-1]]:
        m.rod(end, nock, .02, '#ede1bd')


def arrow(a, d):
    b = a + d*1.05
    m.rod(a, b, .03, '#b99553')
    m.poly([b + d*.13, b + [-.07, 0, 0], b + [.07, 0, 0]], '#bacbd6')
    for dx in [-.06, .06]:
        m.poly([a, a + [dx, 0, 0] + d*.1, a + d*.22], '#ede1b8')


def bow_effects(style, f, h, aim, tail):
    kind = A.BOW[style]['effect']
    side = A.unit(np.cross(aim, [0, 0, 1]))
    up = np.cross(side, aim)
    if f == 2:
        lines = dict(kind='lines', alpha=.8, layer='under',
                     segments=[(h + aim*.3 + o, tail - aim*.1 + o) for o in (up*.16, -up*.16, side*.18)])
        if kind == 'ghost':
            return [lines, dict(kind='ghost', layer='under', alphas=[.25, .45, .7],
                                segments=[(tail - aim*s, tail - aim*s + aim*1.05) for s in (1.5, 1.0, .5)])]
        out = [lines, dict(kind='wedge', start=h + aim*.2, end=tail, width=.34, alpha=1, layer='under')]
        if kind == 'ring':
            out.append(dict(kind='ring', center=h + aim*.25, axis=aim, radius=.3, alpha=1, layer='over'))
        return out
    if f == 3 and kind == 'ring':
        return [dict(kind='ring', center=h + aim*.45, axis=aim, radius=.45, alpha=.45, layer='over')]
    return []


def archer(row, f, style=None, kit='archer'):
    style = style or A.SELECTED.get(kit, A.SELECTED['archer'])
    p = pose(0 if row == HIT else row, 0 if row == HIT else f)
    p['hands'] = [[.05, -.22, 1.18], [.12, -.62, .97]]
    p['elbows'] = [[-.42, -.02, 1.18], [.42, -.24, 1.13]]
    k = A.BOW[style]['frames'][f] if row == ATTACK else None
    if row == 1:
        p['bob'] = [-.025, .035, -.025, .035][f]
    if k:
        p['hands'] = [k['draw'], k['bow']]
        p['elbows'] = [(V([-.4, 0, 1.42]) + V(k['draw']))/2 + [-.22, .08, 0],
                       (V([.4, 0, 1.42]) + V(k['bow']))/2 + [.12, 0, 0]]
        p['lean'], p['bob'] = k.get('lean', 0), k.get('bob', 0)
        A.stance(p, k)
    if row == 4:
        p['hands'] = [[-.4, -.25, 1.1], [.35, -.55, 1.0]]
    if row == HIT:
        hit_lean(p, f)
    _, head = soldier(p, kit, weapon=False)
    T = transform(p)
    h, pull = T(p['hands'][1]), T(p['hands'][0])
    effects = []
    if k:
        aim = A.unit(k['aim'])
        axis = -A.unit(np.cross([1, 0, 0], aim))
        flight = k.get('flight')
        nock = pull if flight is None and f < 3 else h - aim*.06
        bow(h, nock, axis, aim, k.get('bend', 1))
        if f < 3:
            tail = nock if flight is None else h + aim*flight
            arrow(tail, aim)
            effects = bow_effects(style, f, h, aim, tail)
        else:
            effects = bow_effects(style, f, h, aim, h)
    else:
        bow(h, pull, V([0, -.5, .866]), V([0, -1, 0]))
        arrow(pull, V([0, -.866, -.5]))
    m.rod(T([-.21, .3, .93]), T([-.21, .3, 1.53]), .12, '#704022')
    for x in [-.28, -.2, -.12]:
        m.rod(T([x, .3, 1.3]), T([x, .3, 1.82]), .03, '#d7bc7c')
    return result(m.meshes, head, row, f, effects)


COATS = {'bay': ('#825032', '#9a6845', '#352820'), 'black': ('#2b2b33', '#3d3d48', '#15151a'),
         'red': ('#7a2f22', '#a14a35', '#352820'),  # red: Red Hare
         'white': ('#c9cdd0', '#dde1e3', '#5a5f66')}
RAMPS.update({'#7a2f22': 'bKLM', '#a14a35': 'KLLM', '#c9cdd0': '3455', '#dde1e3': '4556', '#5a5f66': '1223'})
MOUNT_REACH = {'none': (.9, 1.75), 'spear': (.9, 1.75), 'snake': (.9, 1.75), 'glaive': (1.15, 1.95), 'halberd': (.9, 1.75)}
RIDER_OFFSET = V([0, .12, 1.2])


def rider_pose(row, f, k=None):
    p = pose(row, f)
    p['bob'] = [0, .06, .1, .06][f] if row == 0 else 0
    p['lean'] = 0
    p['feet'] = [[-.49, .05, .03], [.49, .05, .03]]
    p['knees'] = [[-.53, -.27, .45], [.53, -.27, .45]]
    p['hands'][1] = [.25, -.48, 1.02]
    p['elbows'][1] = [.43, -.15, 1.18]
    if row == HIT:
        hit_lean(p, f)
    if k is not None:
        p['hands'][0] = list(k['hand'])
        p['elbows'][0] = ((SHOULDER + V(k['hand']))/2 + [-.18, 0, -.06]).tolist()
        p['lean'], p['bob'] = float(k.get('lean', 0)), float(k.get('bob', 0))
    if row == 1:
        p['lean'] = -.18
    if row == 4:
        p['lean'] = [-.52, -.60, -.54, -.62][f]
        p['bob'] = [0, -.035, 0, -.035][f]
        p['hands'][0] = [-.48, -.38, .85]
    return p


def horse_move(rear, shift):
    """Pitch the whole mount about the hind hooves (+ rear lifts the forehand), then slide."""
    a = -float(rear)
    rot = V([[1, 0, 0], [0, math.cos(a), -math.sin(a)], [0, math.sin(a), math.cos(a)]])
    pivot, d = V([0, .55, .12]), V([0, float(shift), 0])
    return lambda v: ((np.asarray(v) - pivot) @ rot.T + pivot) + d


def spear_points(k, reach=1.0, weapon='spear'):
    """(hand, inner, tip) of the mounted weapon in world space for an attack keyframe."""
    p = rider_pose(ATTACK, 0, k)
    hand = transform(p)(p['hands'][0])*.8 + RIDER_OFFSET
    d = A.unit(k['dir'])
    inner, tip = MOUNT_REACH[weapon]
    move = horse_move(k.get('rear', 0), k.get('shift', 0))
    return move(hand), move(hand + d*inner), move(hand + d*tip*reach)


def spear_effects(style, f, weapon):
    spec = A.SPEAR[style]
    frames = spec['frames']
    if f < 2:
        return []
    if spec['effect'] == 'arc':
        keys = A.sample(A.swing_path(spec), 4, lerp_key)
        if f == 3:
            keys = keys[-6:] + A.sample([frames[2], frames[3]], 3, lerp_key)[1:]
        pts = [spear_points(k, 1.15 if weapon != 'glaive' else 1.1, weapon) for k in keys]
        return [A.arc([(i, t) for _, i, t in pts], alpha=1 if f == 2 else .45)]
    hand, _, tip = spear_points(frames[f], weapon=weapon)
    k = frames[f]
    turn = horse_move(k.get('rear', 0), 0)
    d = turn(A.unit(k['dir'])) - turn(V([0, 0, 0]))
    return A.pierce(tip, d, hand, alpha=1 if f == 2 else .6, burst=f == 2, reach=1.5)


def cavalry(row, f, style=None, kit='rider', coat='bay', weapon='spear'):
    """Rider and horse from legacy new_units/build.py, with idle breathing.
    weapon: 'spear', 'snake' (Zhang Fei's serpent spear) or 'glaive' (Guan Yu's crescent blade)."""
    style = style or A.SELECTED['cavalry']
    k = A.SPEAR[style]['frames'][f] if row == ATTACK else None
    p = rider_pose(row, f, k)
    fs, head = soldier(p, kit, weapon=False)
    bounce = [.08, 0, .14, .03][f] if row == 1 else 0
    off = V([0, .12, 1.2 + bounce])
    m.meshes = [(v*.8 + off, c) for v, c in fs]
    head = head*.8 + off
    hand = transform(p)(p['hands'][0])*.8 + off
    dv = A.unit(k['dir']) if k else V([0, -.35, .94])/np.linalg.norm([0, -.35, .94])
    side = A.unit(np.cross(dv, [0, 0, 1]))*.1 if abs(dv[2]) < .9 else V([.1, 0, 0])
    if weapon == 'glaive':
        glaive_at(hand, dv, side*10)
    elif weapon != 'none':  # 'none': an unarmed rider (merchant)
        m.rod(hand - dv*.5, hand + dv*1.45, .045, '#966131')
    tip, base = hand + dv*1.75, hand + dv*1.42
    if weapon == 'snake':
        ts = np.linspace(0, 1, 8)
        center = [base + dv*.55*t + side*.6*math.sin(3*math.pi*t) for t in ts]
        strip([c - side*.9*(1 - t) for c, t in zip(center, ts)],
              [c + side*.9*(1 - t) for c, t in zip(center, ts)], '#d6e7ec')
    elif weapon in ('spear', 'halberd'):
        m.poly([base - side, tip, base + side, base - dv*.06], '#d6e7ec')
    if weapon == 'halberd':  # Sky Piercer: crescent blade on one side of the spear head
        ts = np.linspace(0, 1, 6)
        bulge = [math.sin(math.pi*t) for t in ts]
        inner = [base - dv*.25 + dv*.45*t + side*(1.2 + .8*b) for t, b in zip(ts, bulge)]
        outer = [base - dv*.25 + dv*.45*t + side*(1.2 + 2.6*b) for t, b in zip(ts, bulge)]
        strip(inner, outer, '#d6e7ec')
        m.rod(base - dv*.3, base - dv*.18, .06, GOLD)
        m.ell(base - dv*.36 - side*.6, [.07]*3, RED)
    chest_c, muzzle, dark = COATS[coat]
    nod = [0, -.04, -.08, -.04][f] if row == 0 else 0

    def H(q):
        v = V(q)
        v[2] += bounce
        if v[1] < -.6:
            v[2] += nod*(-v[1] - .6)/.7
        if row == 4 and v[1] < -.7:
            v[2] -= .25
            v[1] -= .08
        return v

    breath = [0, .012, .024, .012][f] if row == 0 else 0
    m.ell(H([0, 0, 1.13]), [.43 + breath, .79, .44 + breath], chest_c)
    m.ell(H([0, .53, 1.12]), [.44, .4, .42], chest_c)
    m.rod(H([0, -.56, 1.2]), H([0, -.9, 1.83]), .24, chest_c)
    m.ell(H([0, -1.04, 1.86]), [.21, .35, .25], chest_c)
    m.ell(H([0, -1.32, 1.69]), [.20, .27, .17], muzzle)
    for x in [-.13, .13]:
        m.ell(H([x, -.93, 2.12]), [.055, .1, .17], chest_c)
        m.ell(H([x*1.5, -1.13, 1.92]), [.032, .034, .032], '#15191b')
    for y, z in [(-.64, 1.53), (-.76, 1.73), (-.89, 1.98)]:
        m.ell(H([0, y + .13, z]), [.12, .15, .2], dark)
    for i, (x, y) in enumerate([(-.3, -.5), (.3, -.5), (-.3, .55), (.3, .55)]):
        phase = f*math.pi/2 + (0 if i < 2 else math.pi) + (.28 if i % 2 else 0)
        stride = .43*math.cos(phase) if row == 1 else 0
        if row == 1:
            lift = .12 + max(0, math.sin(phase))*.28 if f == 2 else max(0, math.sin(phase))*.24
        else:
            lift = 0
        hip, knee, foot = H([x, y, 1.12]), H([x, y + stride*.5, .58]), V([x, y + stride, .11 + lift])
        if row == 1:
            knee[1] -= .15*math.sin(phase)
            knee[2] += .08
        m.rod(hip, knee, .10, chest_c)
        m.rod(knee, foot, .065, chest_c)
        m.ell(foot, [.1, .14, .10], dark)
    sway = [0, .12, 0, -.12][f] if row == 0 else 0
    m.rod(H([0, .73, 1.3]), H([sway, 1.02, .60]), .09, dark)
    m.ell(H([0, .06, 1.52]), [.47, .38, .07], '#a91e2c')
    if KITS[kit].get('packs'):  # saddle bags behind the rider
        for x in [-.47, .47]:
            m.box(H([x, .42, 1.3]), [.16, .42, .34], KITS[kit]['packs'])
    for x in [-.22, .22]:
        m.rod(H([x, -1.27, 1.74]), V([x, -.25, 1.98]), .017, '#38261d')
    fs, effects = m.meshes, []
    if k:
        move = horse_move(k.get('rear', 0), k.get('shift', 0))
        fs = [(move(v), c) for v, c in fs]
        head = move(head)
        effects = spear_effects(style, f, weapon) if weapon != 'none' else []
    return result(fs, head, row, f, effects)


def zhangfei(row, f, style=None):
    return cavalry(row, f, style or A.SELECTED['zhangfei'], kit='zhangfei', coat='black', weapon='snake')


def guanyu(row, f, style=None):
    return cavalry(row, f, style or A.SELECTED['guanyu'], kit='guanyu', coat='red', weapon='glaive')
