"""방술사: the 사마 (strategist) robe turned Taoist priest — 화양건 cap, ankle-length faction robe with wide hanging
sleeves, gourd on the left hip, horsetail whisk (불진) in the right hand instead of the fan.

The whisk is a short wooden handle with a white tuft of strands; the tuft follows its own direction per pose (hanging
down at rest, trailing behind when raised, lashing forward on the strike). Attack: raise the whisk overhead, then sweep
it forward. The robe skirt follows the feet (walk), or drapes over the lap when seated (exhausted).

Tags: hat themeA (black), hair hair1-3, whisk tuft and jade iron (white), whisk handle themeC (wood),
gourd themeB (tan), robe/sleeves/brim faction. Every patch is applied in make().
"""
import math

import numpy as np

import grid3d_mounted as M

G, U, A, B, Bl, Hd, R = M.G, M.U, M.A, M.B, M.Bl, M.Hd, M.R
m, V = U.m, U.V

TUFT, TUFT_SHADE, TUFT_LIGHT = '#ede1be', '#c4d2da', '#f1f6f4'
HANDLE, GOURD = '#a0703b', '#b07a41'
SHOULDER = V([-.4, 0, 1.42])
IDLE_HANDS = [[-.5, -.4, 1.18], [.14, -.42, 1.06]]
IDLE_ELBOWS = [[-.64, -.08, 1.04], [.52, -.14, 1.04]]
DOWN = V([-.1, 0, -1.])
# attack: hand, handle direction (up), tuft direction, body keys
ATTACK = [dict(hand=[-.5, -.3, 1.32], up=[-.2, -.1, 1], tuft=[-.1, .1, -1], lean=.02),
          dict(hand=[-.42, .08, 2.0], up=[0, .45, 1], tuft=[0, 1, -.35], lean=.12, crouch=.3),
          dict(hand=[-.45, -.8, 1.3], up=[0, -1, .15], tuft=[0, -.25, 1], lean=-.22, bob=-.04, lunge=.7),
          dict(hand=[-.42, -.65, .98], up=[0, -.7, -.5], tuft=[0, -1, -.25], lean=-.12, bob=-.02, lunge=.5)]
# head: 화양건 on the topknot face; grids are 17 wide (2 spare columns per side), so the anchor moves 2 right
HY = {
    'S': ['...00000000000...',
          '..0222333322220..',
          '..0111111111110..',
          '..0221222221220..',
          '...02122212110...',
          '...02125212110...',
          '...0ZYYYYYYYX0...'],
    'SW': ['.....00000000000.',
           '....0222333222220',
           '...0111111111110.',
           '...0221222211110.',
           '...02122211110...',
           '...02152211110...',
           '...0ZYYYYYYXX0...'],
    'W': ['..........00000..',
          '......00002222220',
          '...0222222222210.',
          '...0312222221110.',
          '...0212222211110.',
          '...0212222211110.',
          '...0ZYYYYYYYYXX0.'],
    'NW': ['.....00000000000.',
           '....0222333222220',
           '...0111111111110.',
           '...0221222211110.',
           '...02122211110...',
           '...02122211110...',
           '...0YYYYYYYXX0...'],
    'N': ['...00000000000...',
          '..0222333322220..',
          '..0111111111110..',
          '..0221222221220..',
          '...02122212110...',
          '...02122212110...',
          '...0YYYYYYYYX0...'],
}
HAT_KEYS = {'1': 'f', '2': 'g', '3': 'h'}   # black cap -> themeA (o n m)
HAIR_KEYS = {'f': 'x', 'g': 'y'}            # topknot hair -> hair3, hair2


def heads():
    grids = {}
    for d, hat in HY.items():
        face = [Hd.recolor(['..' + r + '..'], HAIR_KEYS)[0] for r in Hd.HAIR[d][5:]]
        grids[d] = Hd.recolor(hat, HAT_KEYS) + face
        assert all(len(r) == 17 for r in grids[d]), d
    return dict(grids=grids, skin='ij', lid='i', anchor=(Hd.ANCHOR[0] + 2, Hd.ANCHOR[1] + 2))


def unit(v):
    v = V(v)
    return v/np.linalg.norm(v)


def frustum(top, rt, bot, rb, color, n=14):
    """Elliptic frustum between two horizontal rings (rx, ry), with a bottom cap."""
    def ring(c, r, a):
        return c + V([r[0]*math.cos(a), r[1]*math.sin(a), 0])
    angs = np.linspace(0, 2*math.pi, n + 1)
    for a0, a1 in zip(angs, angs[1:]):
        m.poly([ring(top, rt, a0), ring(top, rt, a1), ring(bot, rb, a1), ring(bot, rb, a0)], color)
    m.poly([ring(bot, rb, a) for a in angs[:-1]], color)


def cone(a, b, ra, rb, color, n=10):
    d = unit(b - a)
    u = np.cross(d, [0, 0, 1])
    u = u/np.linalg.norm(u) if np.linalg.norm(u) > .01 else V([1, 0, 0])
    w = np.cross(d, u)
    angs = np.linspace(0, 2*math.pi, n + 1)
    for t0, t1 in zip(angs, angs[1:]):
        def q(r, t):
            return r*(u*math.cos(t) + w*math.sin(t))
        m.poly([a + q(ra, t0), a + q(ra, t1), b + q(rb, t1), b + q(rb, t0)], color)


def robe(p, T, seated):
    if seated:  # lap: cloth over the thighs to the knees, hem falling from the knees to the feet
        hips = (T([-.24, 0, .77]) + T([.24, 0, .77]))/2
        knees = (V(p['knees'][0]) + V(p['knees'][1]))/2
        feet = (V(p['feet'][0]) + V(p['feet'][1]))/2
        m.ell(hips + V([0, .02, .05]), [.46, .34, .26], U.BLUE)
        m.ell((hips + knees)/2 + V([0, 0, .02]), [.44, .36, .17], U.BLUE)
        frustum(knees + V([0, 0, .1]), (.4, .14), V([feet[0], feet[1] + .1, .08]), (.44, .14), U.BLUE)
        return
    fl, fr = V(p['feet'][0]), V(p['feet'][1])
    spread = abs(fl[1] - fr[1])
    hem = .34 if spread > .1 else .24  # walking: the hem rises so the striding feet show
    frustum(T([0, 0, .98]), (.42, .3), V([0, (fl[1] + fr[1])/2 + .02, hem]), (.5, .36 + .4*spread), U.BLUE)


def sleeves(p, T):
    for i in range(2):
        el, ha = V(T(p['elbows'][i])), V(T(p['hands'][i]))
        cone(el, ha + (ha - el)*.05, .15, .23, U.BLUE)
        m.ell(el*.4 + ha*.6 + V([0, .03, -.2]), [.16, .17, .27], U.BLUE)
        m.ell(ha + (ha - el)*.06, [.21]*3, U.LIGHT_BLUE)


def gourd(T):
    g0 = T([.56, -.2, .72])
    m.ell(g0, [.15, .15, .16], GOURD)
    m.ell(g0 + V([0, 0, .2]), [.1, .1, .1], GOURD)


def whisk(hand, up, tuft):
    """Handle from the hand along `up`, tuft strands from its end along `tuft`, spreading and ragged at the end."""
    up, td = unit(up), unit(tuft)
    top = hand + up*.45
    m.rod(hand - up*.12, top, .06, HANDLE)
    s1 = np.cross(td, [0, 1, 0])
    s1 = unit(s1) if np.linalg.norm(s1) > .1 else V([1, 0, 0])
    s2 = np.cross(td, s1)
    root = top + td*.08
    for (a, b, ln), col in zip([(-.02, -.06, .86), (-.16, .02, .78), (.04, .06, .7), (-.1, -.1, .74)],
                               [TUFT, TUFT_SHADE, TUFT, TUFT_LIGHT]):
        end = root + td*ln + s1*(a - .04) + s2*b
        mid = root*.5 + end*.5
        m.rod(root, mid, .075, col)
        m.rod(mid, end, .06, col)
    return top


def body(p, row, f, up, tuft, effects=()):
    seated = row == 4
    _, head = U.soldier(p, 'sorcerer', weapon=False)
    T = U.transform(p)
    for i in range(2):  # dark trousers below the hem (themeA, like the shoes): the stride reads when walking
        knee, foot = V(p['knees'][i]), V(p['feet'][i])
        m.rod(knee, foot, .135, U.BROWN)
    robe(p, T, seated)
    sleeves(p, T)
    gourd(T)
    whisk(V(T(p['hands'][0])), up, tuft)
    return U.result(m.meshes, head, row, f, effects)


def elbow_for(hand):
    return ((SHOULDER + V(hand))/2 + [-.15, 0, -.05]).tolist()


def attack_pose(k):
    p = U.pose(0, 0)
    p['row'] = U.ATTACK
    p['hands'] = [list(k['hand']), list(IDLE_HANDS[1])]
    p['elbows'] = [elbow_for(k['hand']), list(IDLE_ELBOWS[1])]
    p['lean'], p['bob'] = k.get('lean', 0), k.get('bob', 0)
    A.stance(p, k)
    return p


def lerp(a, b, t):
    return {key: np.array(a.get(key, 0), float)*(1 - t) + np.array(b.get(key, 0), float)*t
            for key in ('hand', 'up', 'tuft', 'lean', 'bob', 'crouch', 'lunge')}


def edge(k):
    """(inner, outer) points of the whisk for the swing trail: handle end and the tuft's tip."""
    p = attack_pose(k)
    hand = U.transform(p)(p['hands'][0])
    top = hand + unit(k['up'])*.45
    return top, top + unit(k['tuft'])*.8*A.TRAIL_REACH


def attack_effects(f):
    if f < 2:
        return []
    keys = [lerp(ATTACK[1], ATTACK[2], t) for t in np.linspace(0, 1, 7)]
    if f == 3:
        keys = keys[-4:] + [lerp(ATTACK[2], ATTACK[3], t) for t in np.linspace(0, 1, 3)][1:]
    return [A.arc([edge(k) for k in keys], alpha=1 if f == 2 else .45)]


def sorcerer(row, f):
    if row == U.ATTACK:
        k = ATTACK[f]
        return body(attack_pose(k), row, f, k['up'], k['tuft'], attack_effects(f))
    if row == B.BLOCK:  # guard: whisk handle held across before the face, tuft hanging
        k = Bl.SWORD[Bl.SELECTED]['frames'][f]
        p = Bl.foot_pose(k)
        res = body(p, 0, f, k['blade'], [-.2, 0, -1])
        return Bl.displace(res, k.get('sway', 0), k.get('back', 0))
    p = U.pose(row, f) if row != U.HIT else U.pose(0, 0)
    if row in (0, 1):
        p['hands'], p['elbows'] = [list(h) for h in IDLE_HANDS], [list(e) for e in IDLE_ELBOWS]
        return body(p, row, f, [-.3, -.25, 1], DOWN)
    if row == U.HIT:
        p['hands'], p['elbows'] = [list(h) for h in IDLE_HANDS], [list(e) for e in IDLE_ELBOWS]
        p = U.hit_pose(f, p)
        return body(p, row, f, [-.3, -.1, 1], DOWN)
    # exhausted (seated): whisk held up beside the knee, tuft drooping to the ground (a forward whisk leaves the cell)
    p['hands'][0], p['elbows'][0] = [-.58, -.3, 1.02], [-.66, -.05, 1.12]
    return body(p, row, f, [-.1, -.1, 1], [-.3, 0, -1])


def make():
    U.KITS['sorcerer'] = dict(cloth=U.BLUE, tabard=U.LIGHT_BLUE, guard=U.BLUE, shoulder=U.BLUE, shield=False, weapon=None,
                              robe=U.BLUE)
    R.add_ramps({TUFT: '4556', TUFT_SHADE: '4455', TUFT_LIGHT: '5666', HANDLE: 'kllm', GOURD: 'ABCC'})
    R.COLOR.update({'x': (42, 30, 25), 'y': (71, 52, 42)})
    M.KEY2TAG.update(x='i', y='h', q='j', p='k')  # hair keys; the tired face's sweat drop -> faction light
    U.WEAPON_COLORS.update({TUFT, TUFT_SHADE, TUFT_LIGHT, HANDLE})

    def build(row, f, style=None):
        return sorcerer(row, f)
    return build, heads()


M.register('sorcerer', make)
