"""흉적: the 적병 body (hemp sleeveless top, bare arms, rope belt, faction headband) with a tiger pelt over the left
shoulder, a thick shaggy beard up to the band, and a broad heavy chopper (대도) swung two-handed overhead.

Everything that touches shared modules (units.KITS / WEAPON_DRAW / soldier, attacks.SWORD, heads.expression,
the converter's KEY2TAG) is patched inside make(), so importing this module changes nothing for other units.
Tags: hemp -> themeC, leather/rope and pelt stripes -> themeA, pelt -> themeB, hair and beard -> hair,
blade -> iron (edge iron1, body iron2, spine iron3), headband -> faction.
"""
import numpy as np

import grid3d_mounted as M

G, U, A, B, Bl, Hd = M.G, M.U, M.A, M.B, M.Bl, M.Hd
m, V = U.m, U.V

HILT = '#976039'                                             # weapon leather (ramp fghh -> themeA)
BLADE_EDGE, BLADE_MID, BLADE_SPINE = '#e5f4ff', '#9fb0bd', '#5a6470'
PELT, PELT_LIGHT, STRIPE = '#c07a2a', '#e0a050', '#2a1a14'
RAMPS = {BLADE_MID: '3445', BLADE_SPINE: '2223', PELT: 'kkll', PELT_LIGHT: 'klmm', STRIPE: 'ffff'}
KIT = dict(cloth=U.HEMP, tabard=U.BROWN, guard=U.HEMP_DARK, shoulder=U.SKIN, arms=U.SKIN, shield=False, weapon='dadao')
# render keys -> tags for this unit: hemp S T U, pelt k l m, head hair 7 8 9, sweat drop q p
KEYS = dict(S='s', T='t', U='u', k='q', l='p', m='p', **{'7': 'i', '8': 'h', '9': 'g'}, q='a', p='b')
HAIR_KEYS = {'f': '8', 'g': '9', '1': '7'}                    # head grid: hair / beard -> the hair render keys

# Heavy overhead chop: higher wind-up with the blade hanging far behind, a deeper lunge, both hands on the grip
# from the wind-up on (two_hand), and a lower follow-through than the 적병's 내려베기.
CHOP = dict(name='대도 내려찍기', effect='arc', frames=[
    dict(hand=[-.45, .05, 1.85], elbow=[-.6, .05, 1.58], blade=[-.1, .5, .86], lean=.1, two_hand=True),
    dict(hand=[-.4, .25, 2.12], elbow=[-.6, .15, 1.78], blade=[-.2, .9, .4], lean=.32, bob=-.14, crouch=1.2,
         two_hand=True),
    dict(hand=[-.25, -.95, .95], elbow=[-.42, -.55, 1.2], blade=[0, -.6, -.8], lean=-.52, bob=-.16, lunge=1.25,
         two_hand=True),
    dict(hand=[-.28, -.8, .72], elbow=[-.45, -.5, 1.0], blade=[.05, -.3, -.95], lean=-.36, bob=-.13, lunge=1.2,
         two_hand=True)],
    via=[dict(hand=[-.35, -.1, 2.15], blade=[-.05, .1, 1]), dict(hand=[-.3, -.6, 1.85], blade=[0, -.6, .8]),
         dict(hand=[-.27, -.9, 1.35], blade=[0, -1, .05])])
# Facing the camera or away from it (S, N) the straight chop swings in the view plane and the blade and trail shrink to
# a line: chop diagonally instead, from over the weapon shoulder down across to the off side.
CHOP_DIAGONAL = dict(name='대도 비껴 내려찍기', effect='arc', frames=[
    dict(hand=[-.6, .05, 1.85], blade=[-.6, .1, .8], lean=.1, two_hand=True),
    dict(hand=[-.7, .1, 2.05], blade=[-.8, .15, .55], lean=.28, bob=-.14, crouch=1.2, two_hand=True),
    dict(hand=[.15, -.7, .95], blade=[.7, -.2, -.7], lean=-.5, bob=-.16, lunge=1.25, two_hand=True),
    dict(hand=[.2, -.6, .75], blade=[.6, -.1, -.8], lean=-.34, bob=-.13, lunge=1.2, two_hand=True)],
    via=[dict(hand=[-.45, -.25, 2.05], blade=[-.25, -.1, 1]), dict(hand=[-.05, -.5, 1.75], blade=[.5, -.3, .8]),
         dict(hand=[.15, -.65, 1.3], blade=[.9, -.3, .15])])

# Block: the 적병's 막아서기 beat, but the heavy blade is raised slanting across the body (the level guard showed it
# edge-on in N, NE and W) with the off hand braced against its back.
GUARD = [dict(hand=[-.45, -.45, 1.4], blade=[.75, -.15, .65], lean=.02),
         dict(hand=[-.42, -.5, 1.45], blade=[.75, -.1, .68], crouch=.6, lean=.06, bob=-.06),
         dict(hand=[-.38, -.42, 1.47], blade=[.7, -.05, .72], crouch=.8, lean=.22, bob=-.08, back=.12),
         dict(hand=[-.45, -.45, 1.35], blade=[.7, -.2, .68], crouch=.3, lean=.1)]
for _k in GUARD:
    _k['off'] = (V(_k['hand']) + A.unit(_k['blade'])*.5 + V([0, -.06, -.04])).tolist()


def dadao(p, T):
    """Broad heavy chopper: long two-hand grip, straight dark spine, mid-steel body, bright edge, clipped tip."""
    hand, d, _ = U.grip(p, T)
    side = U.facing(d)  # broad face to the camera in every row, the attack included
    side = side - d*np.dot(side, d)
    side = side/np.linalg.norm(side)
    m.rod(hand - d*.3, hand + d*.1, .055, HILT)
    m.ell(hand - d*.33, [.07]*3, HILT)
    m.ell(hand + d*.14, [.15, .15, .06], HILT)
    b = hand + d*.18
    L = .78
    pt = lambda t, s: b + d*L*t + side*s  # noqa: E731
    edge = [(0, .17), (.45, .26), (.75, .29), (.92, .23), (1.0, .07)]
    inner = [(t, w - .09) for t, w in edge[:-1]] + [(.97, .03)]
    m.poly([pt(0, .02), pt(.96, .02)] + [pt(t, w) for t, w in reversed(inner)], BLADE_MID)
    for (t0, w0), (t1, w1), (u0, v0), (u1, v1) in zip(edge, edge[1:], inner, inner[1:]):
        m.poly([pt(u0, v0), pt(t0, w0), pt(t1, w1), pt(u1, v1)], BLADE_EDGE)
    m.poly([pt(0, -.08), pt(.96, -.08), pt(1.0, .07), pt(.96, .02), pt(0, .02)], BLADE_SPINE)


def open_chest(T):
    m.ell(T([0, -.25, 1.36]), [.1, .05, .12], U.SKIN)


def pelt(T):
    """Tiger hide over the left (+x) shoulder: shoulder cap, back drape, a band across the chest, a hanging paw."""
    m.ell(T([.36, .02, 1.52]), [.25, .31, .14], PELT)
    for z, x, w in [(1.48, .2, .64), (1.26, .12, .7), (1.04, .04, .66), (.84, -.02, .56)]:
        m.box(T([x, .3, z]), [w, .09, .24], PELT)
    m.rod(T([.32, -.22, 1.48]), T([-.28, -.26, .98]), .13, PELT)
    m.rod(T([.48, -.16, 1.42]), T([.5, -.2, 1.02]), .08, PELT)
    m.ell(T([.5, -.21, .98]), [.07, .07, .06], PELT_LIGHT)
    for z, x, w in [(1.4, .3, .26), (1.17, .02, .3), (1.0, .32, .24), (.82, -.08, .26)]:
        m.box(T([x, .37, z]), [w, .05, .09], STRIPE)
    for t in (.3, .7):
        c = np.array(T([.32, -.22, 1.48]))*(1 - t) + np.array(T([-.28, -.26, .98]))*t
        m.box(c + V([0, -.1, 0]), [.06, .05, .2], STRIPE)
    for z in (1.32, 1.16):
        m.box(T([.5, -.27, z]), [.12, .03, .035], STRIPE)


def full_beard(rows, skin='ij'):
    """Thick shaggy beard: whole lower face, sideburns up to the band, two jagged rows under the chin."""
    g = [list(r) for r in rows]
    eyes = Hd.find_eyes(g, skin)
    if not eyes:
        return rows
    w = len(g[0])
    ey = eyes[0][0]
    eye_x = {x for _, x in eyes}
    profile = len(eyes) == 1
    chin = max(y for y, row in enumerate(g) if any(c in skin for c in row))
    for y in range(ey - 1, ey + 2):
        xs = [x for x, c in enumerate(g[y]) if c in skin]
        if not xs:
            continue
        for x in ([xs[-1]] if profile else [xs[0], xs[-1]]):
            if x - 1 not in eye_x and x + 1 not in eye_x:  # keep the eyes findable for the expressions
                g[y][x] = 'B'
    for y in range(ey + 2, chin + 1):
        for x in range(w):
            if g[y][x] in skin:
                g[y][x] = 'B'
    cols = [x for x, c in enumerate(g[chin]) if c == 'B']
    left, right = min(cols), max(cols)
    start = chin + 1
    g += [['.'] * w for _ in range(max(0, start + 3 - len(g)))]
    for x in range(left - 1, right + 2):
        g[start][x] = 'B'
    for x in range(left, right + 1):
        if (x - left) % 2 == 0:
            g[start + 1][x] = 'B'
    Hd.outline_beard(g, start)
    hair = [[c == 'B' for c in row] for row in g]
    for y in range(len(g)):
        for x in range(w):
            if hair[y][x]:
                top = y == 0 or not hair[y-1][x]
                bottom = y + 1 >= len(g) or not hair[y+1][x]
                g[y][x] = '1' if bottom else 'g' if (top and (x + y) % 2) or (x + 2*y) % 5 == 0 else 'f'
    return [''.join(r) for r in g]


def heads():
    grids = {d: Hd.recolor(full_beard(r), HAIR_KEYS) for d, r in Hd.TOPKNOT.items()}
    return Hd.heads(grids, Hd.BAND)


def make():
    U.RAMPS.update(RAMPS)
    B.R.add_ramps(RAMPS)
    U.WEAPON_COLORS |= {BLADE_MID, BLADE_SPINE}
    U.WEAPON_DRAW['dadao'] = dadao
    U.WEAPONS['dadao'] = (.4, 1.0)
    U.KITS['ruffian'] = KIT
    A.SWORD['ruffian_chop'] = CHOP
    A.SWORD['ruffian_chop_diagonal'] = CHOP_DIAGONAL
    A.SELECTED['ruffian'] = 'ruffian_chop'
    M.KEY2TAG.update(KEYS)

    sword_pose = U.sword_pose

    def two_hand_pose(k):
        p = sword_pose(k)
        if k.get('two_hand'):  # off hand just behind the weapon hand on the long grip
            d = A.unit(p['blade'])
            off = V(p['hands'][0]) - d*.2 + V([.06, 0, 0])
            p['hands'][1] = off.tolist()
            p['elbows'][1] = ((V([.4, 0, 1.42]) + off)/2 + V([.15, .05, -.05])).tolist()
        return p

    U.sword_pose = two_hand_pose

    soldier = U.soldier

    def dressed(p, k, weapon=True):
        out = soldier(p, k, weapon)
        if k == 'ruffian':
            T = U.transform(p)
            open_chest(T)
            pelt(T)
        return m.meshes, out[1]

    U.soldier = dressed

    expression = Hd.expression

    def bearded_expression(head, kind, skin, lid):
        """The beard hides the mouth row: open a 1px dark mouth in it (2px when hurt)."""
        out = expression(head, kind, skin, lid)
        if kind == 'normal':
            return out
        eyes = Hd.find_eyes([list(r) for r in head], skin)
        if not eyes:
            return out
        g = [list(r) for r in out]
        y, x = eyes[0][0] + 3, round(sum(x for _, x in eyes)/len(eyes)) - (len(eyes) == 1)
        for yy in range(y, y + (2 if kind == 'hurt' else 1)):
            if yy < len(g) and g[yy][x] in '789':
                g[yy][x] = '0'
        return [''.join(r) for r in g]

    Hd.expression = bearded_expression

    foot = U.foot_unit('ruffian')

    def block(f):
        k = GUARD[f]
        faces, head = U.soldier(Bl.foot_pose(k), 'ruffian')
        return Bl.displace(U.result(faces, head, 0, f), k.get('sway', 0), k.get('back', 0))

    def build(row, f, style=None):
        if row == B.BLOCK:
            return block(f)
        t = U.VIEW['t']
        if row == U.ATTACK and style is None and t is not None and abs(t[0]) < .1:  # S or N view
            style = 'ruffian_chop_diagonal'
        return foot(row, f, style=style) if style else foot(row, f)
    return build, heads()


M.register('ruffian', make)
