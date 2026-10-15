"""노병: the 궁병 body (hood head, faction cloth, bolt case on the back) holding a crossbow instead of a bow.

Adapted from units.archer() / blocks.bowman(): same poses and bolt case, but the weapon is a two-handed crossbow held
level like a rifle (the 궁병's bow stands upright): the stock runs along the aim, the prod lies across it, the rear hand
sits under the grip at the trigger lever, the front hand under the fore-end. The stock follows the line through both
hands, so the weapon follows the hands in every animation.

The crossbow is always spanned: the prod is bent back and the string sits on the latch behind it, a bolt lies on the
stock (idle, walk, hit, block, exhausted, attack frames 0 and 1). The shot (frame 2) pulls the trigger: the string
springs forward, the prod straightens and the bolt flies with a streak; frame 3 lowers the empty crossbow.
"""
import numpy as np

import grid3d_mounted as M

G, U, A, B, Bl, Hd = M.G, M.U, M.A, M.B, M.Bl, M.Hd
m, V = U.m, U.V

WOOD = G.BUCKLER          # pale wood: ramp keys A B C -> themeB tags
STRING, BOLT, HEAD = '#ede1bd', '#b99553', '#bacbd6'
UP = V([0, 0, 1])

U.KITS['crossbow'] = dict(U.KITS['archer'])
A.BOW['crossbow'] = dict(name='석궁', effect='streak', frames=[])  # units.bow_effects reads only the effect kind

# rear hand (grip), front hand (fore-end), model space; both at one height so the stock lies level
IDLE_REAR, IDLE_FRONT = [0, -.3, 1.16], [0, -.8, 1.16]
SHOULDERS = (V([-.4, 0, 1.42]), V([.4, 0, 1.42]))
# body lean/bob/crouch per frame; loaded=False: the string is released; flight = how far the shot bolt has flown
ATTACK = [dict(rear=[0, -.3, 1.02], front=[0, -.8, 1.02], lean=.04),                                       # low ready
          dict(rear=[0, -.32, 1.4], front=[0, -.82, 1.4], lean=.14, bob=-.04, crouch=.4),                  # raised, aiming
          dict(rear=[0, -.26, 1.42], front=[0, -.78, 1.42], lean=.22, bob=-.04, crouch=.5, flight=1.6),     # trigger pulled
          dict(rear=[0, -.3, 1.0], front=[0, -.58, .82], lean=.06, bob=-.02, loaded=False)]                # empty, lowering


def frame_of(aim):
    """(side, top): the prod direction (horizontal, across the aim) and the stock's upper face."""
    h = np.cross(aim, UP)
    h = V([1, 0, 0]) if np.linalg.norm(h) < .2 else h/np.linalg.norm(h)
    v = np.cross(h, aim)
    v = v if v[2] >= 0 else -v
    return h, v/np.linalg.norm(v)


def flying_bolt(tail, aim):
    m.rod(tail, tail + aim*.62, .03, BOLT)
    m.rod(tail + aim*.6, tail + aim*.78, .05, HEAD)


def crossbow(rear, front, loaded=True):
    """Crossbow meshes. loaded: prod bent back, string on the latch, bolt on the stock; else released and straight."""
    aim = A.unit(front - rear)
    side, top = frame_of(aim)
    lift = top*.07                                                    # rides above the hands: they hold it from below
    butt, muzzle = rear - aim*.38 + lift, front + aim*.26 + lift
    m.rod(butt, muzzle, .085, WOOD)                                   # stock
    m.ell(butt, [.11]*3, WOOD)
    trigger = rear + aim*.12 + lift
    m.rod(trigger - top*.03, trigger - top*.2 + aim*.04, .04, U.STEEL)  # trigger lever under the stock, at the rear hand
    c = muzzle - aim*.04
    tips = []
    for s in (-1, 1):
        if loaded:    # bent back: long limbs swept behind the stock's front end so the curve shows from every side
            mid, tip = c + s*side*.36 - aim*.14, c + s*side*.68 - aim*.52
        else:         # released: straight across, tips a little forward
            mid, tip = c + s*side*.36, c + s*side*.68 + aim*.04
        m.rod(c, mid, .075, U.STEEL)
        m.rod(mid, tip, .065, U.STEEL)
        tips.append(tip)
    m.ell(muzzle + aim*.06, [.07]*3, U.STEEL)                         # stirrup
    if loaded:                                                        # string on the latch, bolt on the stock
        latch = c - aim*.62
        for t in tips:
            m.rod(t, latch, .024, STRING)
        nock = latch + top*.09
        m.rod(nock, nock + aim*1.04, .032, BOLT)
        m.rod(nock + aim*1.02, nock + aim*1.2, .05, HEAD)
    else:
        m.rod(tips[0], tips[1], .024, STRING)
    return aim


def bolt_case(T):
    m.rod(T([-.21, .3, .93]), T([-.21, .3, 1.43]), .12, '#704022')    # shorter than the 궁병's quiver
    for x in (-.28, -.2, -.12):
        m.rod(T([x, .3, 1.28]), T([x, .3, 1.58]), .03, WOOD)


def elbows(rear, front):
    return [((SHOULDERS[0] + V(rear))/2 + [-.2, .08, 0]).tolist(), ((SHOULDERS[1] + V(front))/2 + [.1, 0, 0]).tolist()]


def crossbowman(row, f, kit='crossbow'):
    p = U.pose(0 if row == U.HIT else row, 0 if row == U.HIT else f)
    rear, front = IDLE_REAR, IDLE_FRONT
    k = ATTACK[f] if row == U.ATTACK else None
    if row == 1:
        p['bob'] = [-.025, .035, -.025, .035][f]
    if k:
        rear, front = k['rear'], k['front']
        p['lean'], p['bob'] = k.get('lean', 0), k.get('bob', 0)
        A.stance(p, k)
    if row == 4:
        rear, front = [-.4, -.25, 1.1], [.35, -.55, 1.0]
    p['hands'], p['elbows'] = [list(rear), list(front)], elbows(rear, front)
    if row == U.HIT:
        U.hit_lean(p, f)
    _, head = U.soldier(p, kit, weapon=False)
    T = U.transform(p)
    r, fr = T(rear), T(front)
    flight = k.get('flight') if k else None
    aim = crossbow(r, fr, loaded=flight is None and (k or {}).get('loaded', True))
    effects = []
    if flight is not None:
        tail = fr + aim*flight
        flying_bolt(tail, aim)
        effects = U.bow_effects('crossbow', 2, fr, aim, tail)
    bolt_case(T)
    return U.result(m.meshes, head, row, f, effects)


def block(f):
    k = Bl.BOW[Bl.SELECTED]['frames'][f]
    p = U.pose(0, 0)
    p['hands'], p['elbows'] = [list(k['hand']), list(k['off'])], elbows(k['hand'], k['off'])
    p['lean'], p['bob'] = k.get('lean', 0), k.get('bob', 0)
    A.stance(p, k)
    _, head = U.soldier(p, 'crossbow', weapon=False)
    T = U.transform(p)
    crossbow(T(p['hands'][0]), T(p['hands'][1]))
    bolt_case(T)
    return Bl.displace(U.result(m.meshes, head, 0, f), k.get('sway', 0), k.get('back', 0))


def make():
    def build(row, f, style=None):
        return block(f) if row == B.BLOCK else crossbowman(row, f)
    return build, Hd.heads(Hd.HOOD, Hd.HOOD_CLOTH)


M.register('crossbow', make)
