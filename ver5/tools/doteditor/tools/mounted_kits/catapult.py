"""발석거: a small wheeled beam-sling stone thrower, no crew. Wood (leather/themeA ramp) with faction-colour cloth wrapped
around the arm, uprights and skid ends, iron fittings on the axle and counterweight, a stone in the pan.

Model space: front (the throwing direction) is -y, the engine stands on z = 0. The arm turns about the axle in the y-z
plane: phi = 0 long end straight up, 90 forward horizontal, 180 straight down, 210 down at the rear (cocked), 385 swung
over the top and up-forward (release), 418 = 58 stopped by the padded front bar. Everything is scaled by S so the
engine stands about as tall as a foot soldier.

The attack is a lever throw: cocked (bucket low behind the axle), the arm whipping up through 45 degrees behind, just
past vertical (the stone leaves; the game's flying stone takes over from the bucket), then slammed into the stop bar
with a small rebound and a shudder. The trail is a crescent along the circle the bucket sweeps about the axle.
"""
import math

import numpy as np

import grid3d_mounted as M

G, U, A, B, Hd = M.G, M.U, M.A, M.B, M.Hd
m, V = U.m, U.V

WOOD, WOOD_DARK, ROPE = U.BROWN, '#38261d', G.BUCKLER  # '#38261d' reins ramp: outline + dark leather; BUCKLER: pale wood (themeB)
S = .85
P = V([0, 0, 2.0])               # arm axle
LONG, SHORT = 1.45, .62          # arm lengths either side of the axle
WHEEL_R, WHEEL_X, AXLE_Y = .4, .68, .62
SKID_X = .42
DRUM = V([0, .85, .8])
STOP_Y, STOP_Z, STOP_X = -.7, 2.44, .3   # front stop bar: posts on the skids, a padded crossbar the arm slams into
COCKED, SWING, RELEASE, STOPPED = 205, 315, 375, 415   # attack arm angles (stopped = 418 against the bar, rebounded 3)


def parts(fn):
    m.meshes = []
    fn()
    return m.meshes


def arm_dir(phi):
    a = math.radians(phi)
    return V([0, -math.sin(a), math.cos(a)])


def wheel(cx, cy, spin, r=WHEEL_R):
    """Spoked wheel in the y-z plane at x=cx (rim ring, six spokes, hub). spin > 0 rolls it forward (-y)."""
    n = 14
    pts = [V([cx, cy + r*math.cos(2*math.pi*i/n), r + r*math.sin(2*math.pi*i/n)]) for i in range(n)]
    for a, b in zip(pts, pts[1:] + pts[:1]):
        m.rod(a, b, .055, WOOD)
    hub = V([cx, cy, r])
    for i in range(6):
        t = 2*math.pi*i/6 + .26 + spin
        m.rod(hub, hub + V([0, (r - .04)*math.cos(t), (r - .04)*math.sin(t)]), .03, WOOD_DARK)
    m.ell(hub, [.1, .1, .1], U.STEEL)


def leg_point(sx, sy, t):
    """Point on an A-frame leg: t = 0 at the foot, 1 at the apex."""
    return V([sx*.42, sy*.5, .6]) * (1 - t) + V([sx*.16, 0, 1.95]) * t


def band(a, b, t, length, r, color):
    """A cloth wrap: a short thick cylinder around the beam a-b, centred at fraction t."""
    a, b = V(a), V(b)
    d = (b - a)/np.linalg.norm(b - a)
    c = a + (b - a)*t
    m.rod(c - d*length/2, c + d*length/2, r, color)


def body(phi, spin=0, sway=0, stone='pan', rope=True, sag=0):
    u = arm_dir(phi)
    end, weight = P + u*LONG, P - u*SHORT
    for x in (-SKID_X, SKID_X):                                  # skids, faction cloth on their ends
        m.box([x, 0, .52], [.14, 2.0, .14], WOOD)
        for y in (-1.0, 1.0):
            m.box([x, y, .52], [.18, .14, .18], U.BLUE)
    for y in (-.65, .65):                                         # cross beams
        m.box([0, y, .52], [.9, .12, .12], WOOD)
    for y in (-AXLE_Y, AXLE_Y):                                   # wheel axles and the four wheels
        m.rod([-WHEEL_X, y, WHEEL_R], [WHEEL_X, y, WHEEL_R], .045, WOOD_DARK)
        for x in (-WHEEL_X, WHEEL_X):
            wheel(x, y, spin)
    for sx in (-1, 1):                                            # A-frame legs with cloth wraps
        for sy in (-1, 1):
            m.rod(leg_point(sx, sy, 0), leg_point(sx, sy, 1), .06, WOOD)
            band(leg_point(sx, sy, 0), leg_point(sx, sy, 1), .32, .26, .092, U.BLUE)
            band(leg_point(sx, sy, 0), leg_point(sx, sy, 1), .7, .14, .08, U.LIGHT_BLUE)
    for sy in (-1, 1):
        m.rod(leg_point(-1, sy, .5), leg_point(1, sy, .5), .04, WOOD_DARK)
    m.rod([-.3, 0, 2.0], [.3, 0, 2.0], .055, U.STEEL)             # axle with iron caps
    for x in (-.32, .32):
        m.ell([x, 0, 2.0], [.09, .09, .09], U.STEEL)
    m.rod(weight, end, .07, WOOD)                                 # the arm
    for d, half, r, color in ((.45, .1, .115, U.BLUE), (.64, .04, .1, U.LIGHT_BLUE), (.82, .1, .115, U.BLUE),
                              (1.0, .04, .1, U.LIGHT_BLUE), (1.19, .1, .115, U.BLUE)):  # cloth wraps with wood showing between
        m.rod(P + u*(d - half), P + u*(d + half), r, color)
    m.box(weight, [.42, .34, .36], WOOD)                          # counterweight box with an iron band and a cloth wrap
    m.box(weight + V([0, 0, .13]), [.44, .36, .06], U.STEEL)
    m.box(weight + V([0, 0, -.08]), [.44, .36, .09], U.BLUE)
    m.ell(end, [.2, .2, .06], WOOD_DARK)                          # pan
    if stone == 'pan':
        m.ell(end + V([0, 0, .15]), [.17]*3, U.STEEL)
    elif stone == 'ground':
        m.ell([-.15, -1.2, .17], [.17]*3, U.STEEL)
    for x in (-STOP_X, STOP_X):                                   # stop posts and the padded crossbar
        m.rod([x, STOP_Y, .52], [x, STOP_Y, STOP_Z], .05, WOOD)
    m.rod([-STOP_X - .05, STOP_Y, STOP_Z], [STOP_X + .05, STOP_Y, STOP_Z], .05, WOOD_DARK)
    m.rod([-.14, STOP_Y, STOP_Z], [.14, STOP_Y, STOP_Z], .085, U.BLUE)
    for x in (-.3, .3):                                           # winch: posts, drum with a cloth wrap, crank
        m.rod([x, DRUM[1], .55], [x, DRUM[1], DRUM[2]], .05, WOOD)
    m.rod(DRUM + V([-.3, 0, 0]), DRUM + V([.3, 0, 0]), .12, WOOD_DARK)
    m.ell(DRUM, [.16, .13, .13], U.BLUE)
    m.rod(DRUM + V([.3, 0, 0]), DRUM + V([.5, 0, .2]), .03, U.STEEL)
    if rope:                                                      # winch rope to the arm end (sags when slack)
        mid = (end + DRUM)/2 + V([0, 0, -sag])
        m.rod(end, mid, .02, ROPE)
        m.rod(mid, DRUM, .02, ROPE)
    top = weight + V([0, 0, .2])                                  # cloth streamers fluttering from the weight and the apex
    m.rod(top, top + V([sway, .12, -.5]), .05, U.LIGHT_BLUE)
    m.rod([0, 0, 2.05], [sway*.8, -.1, 1.6], .045, U.LIGHT_BLUE)
    return end


# per row and frame: arm angle, roll/pitch of the whole frame, shift, wheel spin, stone, rope, slack.
# At rest the long arm hangs forward-down (phi ~ 118) so the lever reads from every side; the attack winches it back
# (205), holds, whips it over the top (385) and swings through (470 = 110) back to the rest pose.
def pose(row, f):
    ph = f*math.pi/2
    p = dict(phi=118, roll=0, pitch=0, shift=V([0, 0, 0]), spin=0, sway=.12*math.sin(ph), stone='pan', rope=False, sag=0)
    if row == 0:                                                  # idle: the arm and the streamers creak a little
        p.update(phi=118 + 2*math.sin(ph), sway=.16*math.sin(ph))
    elif row == 1:                                                # walk: pushed forward, wheels turn, the frame rocks
        p.update(phi=124 + 2*math.sin(ph), spin=f*math.pi/6, shift=V([0, 0, [0, .03, 0, .03][f]]), sway=.2*math.sin(ph))
    elif row == 2:                                                # attack: cocked, whip up, release, slam into the bar
        p.update(phi=[COCKED, SWING, RELEASE, STOPPED][f], sway=[.05, -.15, -.3, .3][f], rope=f == 0,
                 stone='pan' if f < 2 else 'none')
        if f == 0:                                                # held down by the winch: the frame squats a little
            p.update(shift=V([0, 0, -.02]), pitch=.03)
        if f == 3:                                                # the blow on the bar: the frame bucks forward
            p.update(pitch=-.06, roll=.02, shift=V([0, -.05, .02]))
    elif row == 3:                                                # hit: shudders sideways, a wheel lifts, the cloth snaps
        p.update(phi=118 + [0, 4, -5, 2][f], roll=[0, .03, .13, .05][f], shift=V([[0, .05, -.04, .02][f], [0, .06, .12, .06][f], 0]),
                 sway=[0, .3, -.35, .15][f])
    elif row == 4:                                                # exhausted: arm droops, frame leans, sling slack, stone fell
        p.update(phi=[150, 153, 150, 155][f], pitch=.09, roll=.06, shift=V([0, 0, -.04 + [0, -.01, 0, -.01][f]]), stone='ground',
                 rope=True, sag=.3, sway=.05*math.sin(ph))
    else:                                                         # block: arm lowered across, frame planted
        p.update(phi=[112, 100, 92, 100][f], pitch=[0, .02, .04, .02][f], shift=V([0, [0, 0, .1, .05][f], 0]), sway=[.05, .02, 0, .02][f])
    return p


def transform(p):
    c, s, z0 = math.cos(p['roll']), math.sin(p['roll']), .4*S
    cp, sp = math.cos(p['pitch']), math.sin(p['pitch'])

    def tf(v):
        v = np.asarray(v, float)*S
        x, y, z = v[..., 0], v[..., 1], v[..., 2] - z0
        x, z = c*x + s*z, -s*x + c*z
        y, z = cp*y - sp*z, sp*y + cp*z
        return np.stack([x, y, z + z0], axis=-1) + p['shift']*S
    return tf


def sweep(a, b, alpha, yaw=0, n=12):
    """Crescent along the bucket's circle from arm angle a to b: thin at the tail, full at the bucket. Seen from the
    front or the back that circle is edge-on (a straight line on screen), so yaw (degrees) turns the whole circle
    about the upright through the axle, which shows it as an ellipse, and slides it back so it ends on the bucket."""
    k = math.radians(yaw)
    turn = lambda q: P + V([-(q - P)[1]*math.sin(k), (q - P)[1]*math.cos(k), (q - P)[2]])
    pairs = []
    for i in range(n + 1):
        t = i / n
        u = arm_dir(a + (b - a)*t)
        pairs.append((turn(P + u*(LONG - .08 - .5*t)), turn(P + u*(LONG + .18))))
    back = P + arm_dir(b)*(LONG + .18) - pairs[-1][1]
    pairs = [(i + back, o + back) for i, o in pairs]
    return dict(kind='arc', pairs=pairs, alpha=alpha, layer='under')


def effects_for(row, f, end, tf, style=None):
    if row != 2 or f == 0:
        return []
    yaw = {'charge_n': 55, 'charge_s': 55}.get(style, 0)   # N and S views (M.FACING_STYLE)
    arc = {1: sweep(COCKED + 25, SWING, .85, yaw), 2: sweep(SWING - 35, RELEASE, 1, yaw),
           3: sweep(RELEASE, STOPPED + 3, .6, yaw)}[f]
    arc['pairs'] = [(tf(i), tf(o)) for i, o in arc['pairs']]
    if f < 3:
        return [arc]
    hit = V([0, STOP_Y, STOP_Z])                                  # the slam: a short ring where the arm meets the bar
    return [arc, dict(kind='ring', center=tf(hit), axis=V([0, 1, 0]), radius=.32*S, alpha=.8, layer='over')]


def catapult(row, f, style=None):
    p = pose(row, f)
    tf = transform(p)
    end = V([0, 0, 0])

    def draw():
        nonlocal end
        end = body(p['phi'], p['spin'], p['sway'], p['stone'], p['rope'], p['sag'])

    faces = [(tf(v), c) for v, c in parts(draw)]
    return U.result(faces, tf(V([0, 0, 2.3])), row, f, effects_for(row, f, end, tf, style))


def make():
    return catapult, Hd.heads({d: ['.'*13]*12 for d in Hd.IRON})  # body only: no head anywhere


M.register('catapult', make, vertical_spear=True)  # no spear: only the N/S facing style for the attack trail
