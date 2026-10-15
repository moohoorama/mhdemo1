"""전차대: one bay horse pulling a faction-blue cart with two spoked wheels, a hooded 창병 standing on it with a spear.

The rider is the 창병 body (light kit without buckler) scaled like the 경기병's rider; the spear grip, the idle slant and
the thrust follow the 경기병 (attacks.SPEAR['charge']); the horse follows units.cavalry (breath, trot, drooping head).
"""
import math

import numpy as np

import grid3d_mounted as M

G, U, A, B, Bl, Hd = M.G, M.U, M.A, M.B, M.Bl, M.Hd
m, V = U.m, U.V

WOOD, WOOD_DARK, SHAFT = '#704022', '#38261d', U.SHAFT
Y_SHIFT = -.7                  # centre the long model (horse + cart) on the pivot
CART_Y, WHEEL_Z = 2.05, .55    # cart centre behind the horse, wheel axle height
RIDER_OFF = V([0, 2.0, .66])   # rider origin on the cart bed

U.KITS['chariot'] = dict(G.LIGHT_KIT, weapon=None)
_soldier = U.soldier


def soldier(p, k, weapon=True):
    out = _soldier(p, k, weapon)
    if k == 'chariot':
        G.decorate_light(p, buckler=False)
    return out


U.soldier = soldier


def parts(fn):
    m.meshes = []
    fn()
    return m.meshes


def lift(meshes, dz):
    return [(v + V([0, 0, dz]), c) for v, c in meshes]


def wheel(cx, spin, r=.55):
    """Wheel in the y-z plane at x=cx: rim ring, six spokes, hub. spin > 0 rolls the wheel forward (-y)."""
    n = 16
    pts = [V([cx, CART_Y + r*math.cos(2*math.pi*i/n), WHEEL_Z + r*math.sin(2*math.pi*i/n)]) for i in range(n)]
    for a, b in zip(pts, pts[1:] + pts[:1]):
        m.rod(a, b, .06, U.BROWN)
    hub = V([cx, CART_Y, WHEEL_Z])
    for i in range(6):
        t = 2*math.pi*i/6 + .26 + spin
        m.rod(hub, hub + V([0, (r - .04)*math.cos(t), (r - .04)*math.sin(t)]), .035, SHAFT)
    m.ell(hub, [.11, .11, .11], U.BROWN)


def cart_body():
    cy = CART_Y
    m.rod([-.9, cy, WHEEL_Z], [.9, cy, WHEEL_Z], .05, WOOD_DARK)
    m.box([0, cy, .66], [1.32, 1.5, .12], WOOD)                    # bed
    for x in (-.64, .64):
        m.box([x, cy, .98], [.08, 1.5, .52], U.BLUE)               # faction cloth side panels
        m.box([x, cy, 1.26], [.1, 1.5, .06], U.LIGHT_BLUE)         # light trim along the top
    m.box([0, cy - .72, .98], [1.32, .08, .52], U.BLUE)            # front panel
    m.box([0, cy - .72, 1.26], [1.32, .1, .06], U.LIGHT_BLUE)
    m.box([0, cy - .78, .42], [.5, .08, .22], WOOD_DARK)           # footboard under the front panel


def horse(row, f, bounce, nod, bob):
    """units.cavalry's horse without saddle and rider; bob lifts the draught poles' cart end with the cart."""
    chest_c, muzzle, dark = U.COATS['dot']

    def H(q):
        v = V(q)
        v[2] += bounce
        if v[1] < -.6:
            v[2] += nod*(-v[1] - .6)/.7
            if row == 3:      # hit: the head tosses up
                v[2] += [0, .05, .2, .1][f]*(-v[1] - .6)/.7
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
            lift_ = .12 + max(0, math.sin(phase))*.28 if f == 2 else max(0, math.sin(phase))*.24
        else:
            lift_ = 0
        hip, knee, foot = H([x, y, 1.12]), H([x, y + stride*.5, .58]), V([x, y + stride, .11 + lift_])
        if row == 1:
            knee[1] -= .15*math.sin(phase)
            knee[2] += .08
        m.rod(hip, knee, .10, chest_c)
        m.rod(knee, foot, .065, chest_c)
        m.ell(foot, [.1, .14, .10], dark)
    sway = [0, .12, 0, -.12][f] if row == 0 else 0
    m.rod(H([0, .73, 1.3]), H([sway, 1.02, .60]), .09, dark)       # tail
    m.ell(H([0, -.42, 1.55]), [.5, .12, .1], U.BLUE)               # collar cloth in faction colour
    for x in [-.22, .22]:
        m.rod(H([x, -1.27, 1.74]), V([x, -.25, 1.98]), .017, '#38261d')  # reins
    for x in (-.4, .4):                                             # draught poles: collar to cart front
        m.rod(H([x*.8, -.3, 1.0]), [x, CART_Y - .7, .74 + bob], .075, SHAFT)
    m.rod(H([-.5, -.42, 1.52]), H([.5, -.42, 1.52]), .055, WOOD_DARK)  # yoke over the withers


# thrust: the 경기병's charge (hand, direction) in the 창병's stance, raised so the spear clears the horse's head;
# the cart lurches forward with the blow
ATTACK = [dict(hand=[-.48, .3, 1.2], blade=[0, -1, .12], lean=.1),
          dict(hand=[-.45, .6, 1.3], blade=[0, -1, .2], lean=.3, bob=-.12, crouch=1),
          dict(hand=[-.48, -1.1, 1.45], blade=[0, -1, .34], lean=-.42, bob=-.08, lunge=1.2),
          dict(hand=[-.48, -.8, 1.4], blade=[0, -1, .26], lean=-.24, bob=-.06, lunge=1)]
# facing the camera (S) or away from it (N) the thrust is pitched down / up so the spear stays vertical on screen
ATTACK_PITCH = {'charge_s': -.45, 'charge_n': .45}
LURCH = [0, 0, -.3, -.18]
HIT_PUSH = [0, .04, .1, .05]
SEATED_BLADE = V([-.12, -.25, .95])


def attack_frames(style):
    """ATTACK, or its pitched variant for style 'charge_s' / 'charge_n' (strike and follow-through frames only)."""
    if style not in ATTACK_PITCH:
        return ATTACK
    return ATTACK[:2] + [dict(k, blade=[0, -.9, ATTACK_PITCH[style] + k['blade'][2]]) for k in ATTACK[2:]]


def rider_pose(row, f, attack=ATTACK):
    """(pose, spear direction or None, weapon grip y shift) for the standing rider."""
    if row == U.ATTACK:
        p = U.sword_pose(attack[f])
    elif row == B.BLOCK:
        k = Bl.MOUNT[Bl.SELECTED]['frames'][f]
        p = U.pose(0, 0)
        p['hands'][0] = list(k['hand'])
        p['elbows'][0] = ((U.SHOULDER + V(k['hand']))/2 + [-.18, 0, -.06]).tolist()
        p['lean'], p['bob'] = float(k.get('lean', 0)), float(k.get('bob', 0))
    elif row == U.HIT:
        p = U.pose(3, f)
        U.hit_lean(p, f)
    elif row == 4:
        p = U.pose(4, f)
    else:
        p = U.pose(0, 0 if row == 1 else f)
    p['hands'][1], p['elbows'][1] = [.4, -.6, 1.02], [.5, -.2, 1.15]
    return p


def chariot(row, f, style=None):
    attack = attack_frames(style)
    walk_bob = [.04, 0, .07, .015][f] if row == 1 else 0
    bounce = [.08, 0, .14, .03][f] if row == 1 else 0
    nod = [0, -.04, -.08, -.04][f] if row == 0 else 0
    spin = f*math.pi/6 if row == 1 else 0            # 30 degrees a frame: four frames loop on the six-spoke wheel
    shift = LURCH[f] if row == U.ATTACK else HIT_PUSH[f] if row == U.HIT else 0
    if row == B.BLOCK:
        shift = Bl.MOUNT[Bl.SELECTED]['frames'][f].get('shift', 0)

    horse_m = parts(lambda: horse(row, f, bounce, nod, walk_bob))
    cart_m = lift(parts(cart_body), walk_bob) + parts(lambda: [wheel(x, spin) for x in (-.82, .82)])

    p = rider_pose(row, f, attack)
    T = U.transform(p)
    off = RIDER_OFF + V([0, 0, walk_bob])
    rider_m, head = U.soldier(p, 'chariot', weapon=False)
    rider_m = [(v*.8 + off, c) for v, c in rider_m]
    head = head*.8 + off
    hand = T(p['hands'][0])*.8 + off
    attacking = row == U.ATTACK
    dv = A.unit(attack[f]['blade']) if attacking else A.unit(SEATED_BLADE if row == 4 else
                                                              Bl.MOUNT[Bl.SELECTED]['frames'][f]['dir'] if row == B.BLOCK
                                                              else [0, -.35, .94])
    if attacking or row == B.BLOCK:
        side = A.unit(np.cross(dv, [0, 0, 1]))*.1 if abs(dv[2]) < .9 else V([.1, 0, 0])
    else:
        side = U.facing(dv)*.1
    base, tip = hand + dv*1.42, hand + dv*1.75

    def spear():
        m.rod(hand - dv*(.25 if row == 4 else .5), hand + dv*1.45, .045, SHAFT)
        m.poly([base - side, tip, base + side, base - dv*.06], '#d6e7ec')

    spear_m = parts(spear)
    s = V([0, Y_SHIFT + shift, 0])
    faces = [(v + s, c) for v, c in horse_m + cart_m + rider_m + spear_m]
    effects = []
    if attacking and f >= 2:
        effects = A.pierce(tip + s, dv, hand + s, alpha=1 if f == 2 else .6, burst=f == 2, reach=1.5)
    res = U.result(faces, head + s, row, f, effects)
    if row == B.BLOCK:
        res['face'] = 'normal'
    return res


def make():
    return (lambda row, f, style=None: chariot(row, f, style)), Hd.heads(Hd.HOOD, Hd.HOOD_CLOTH)


M.register('chariot', make, vertical_spear=True)
