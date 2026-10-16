"""군사: the top civil rank. The 참모 body (ankle-length faction robe, light sash) under a tall black-lacquer 진현관 with
a gold plaque and two cap ribbons, a white-edged faction cape (披風) from the shoulders, and a gold 여의 (ruyi) in
the right hand instead of the 참모's fan.

Head: the 참모 cap head (looks.CAP) with its 4 cap rows replaced by 8 crown rows (crest=4 keeps the face anchor);
side views lean the crown back over the nape, front views flare its top. Cape: mantle over the shoulders and a back
panel whose hem swings back a little in walk and hit; seated (exhausted) it stops at the hips and spreads on the
ground behind. Ruyi: gold rod bent forward at the top, a cloud head of three lobes laid out across the view
(units.facing) so it shows its width from every side.

Attack (formation casting): ruyi raised (0), lifted high (1), thrust forward and down at the ground (2), held (3);
frames 2 and 3 draw a formation sigil on the ground in front (two rings), linked to the ruyi head by a line in
frame 2. Raised (0, 1) and guarding, the ruyi uses GOLD_BACK so it never covers the face. Block: blocks.SWORD 'guard' poses with the ruyi held across.

Tags: crown and ribbons 1 2 3 -> iron3/iron2 (black lacquer), cape trim 5 6 -> iron1, gold (plaque, ribbon tips,
ruyi) k l m -> themeC, hair 7 8 9 -> hair1-3, robe, cape, band -> faction.
"""
import numpy as np

import grid3d_mounted as M

G, U, A, B, Bl, Hd, R = M.G, M.U, M.A, M.B, M.Bl, M.Hd, M.R
m, V = U.m, U.V
looks = B.looks

BLUE, LIGHT_BLUE, GOLD = U.BLUE, U.LIGHT_BLUE, U.GOLD
TRIM = '#ede1bd'           # ramp 5566
GOLD_BACK = '#d5a124'      # the same gold, not a weapon colour: stays behind the stamped head (raised, guarding)
UP = V([0, 0, 1])
OUT = V([-.35, 0, 0])      # tips the held ruyi away from the body

# ---- head ----------------------------------------------------------------------------------
HAT = {
    'S': ['.00000000000.', '.03222222110.', '..032222110..', '..032mml110..',
          '..032lll110..', '...0322110...', '...0322110...', '.00YXXXXXX00.'],
    'SW': ['.......00000.', '.....0032210.', '....03222110.', '....0mml110..',
           '...03lll110..', '...0322110...', '...0322110...', '..00YXXXXX00.'],
    'W': ['........00000', '.......032210', '......0322110', '.....0mml110.',
          '.....0lll110.', '....0322110..', '....0322110..', '...00YXXXXX00'],
    'NW': ['.......00000.', '.....0032210.', '....03222110.', '....0322110..',
           '...03222110..', '...0322110...', '...0322110...', '..00YXXXXX00.'],
    'N': ['.00000000000.', '.03222222110.', '..032222110..', '..032222110..',
          '..032222110..', '...0322110...', '...0322110...', '.00YXXXXXX00.'],
}
# (columns, first row, last row) of the two dark cap ribbons hanging behind; the last row is a gold tip
RIBBONS = {'S': ((), 0, 0), 'SW': ((12,), 9, 15), 'W': ((12,), 9, 15), 'NW': ((8, 10), 9, 16), 'N': ((4, 8), 9, 16)}


def crown(d):
    hair = [r.translate(str.maketrans({'f': '8', 'g': '7'})) for r in looks.CAP[d][4:]]
    rows = HAT[d] + hair
    cols, y0, y1 = RIBBONS[d]
    rows += ['.' * 13] * max(0, y1 + 1 - len(rows))
    g = [list(r) for r in rows]
    for x in cols:
        for y in range(y0, y1 + 1):
            g[y][x] = 'l' if y == y1 else '1'
    return [''.join(r) for r in g]


HEADS = Hd.heads({d: crown(d) for d in HAT}, crest=4)

# ---- body ----------------------------------------------------------------------------------
KIT = dict(cloth=BLUE, tabard=LIGHT_BLUE, guard=BLUE, shield=False, weapon=None, robe=BLUE)


def obox(c, x, y, z, color):
    """Box with edge vectors x, y, z (full lengths) centred on c."""
    pts = [c + (sx*x + sy*y + sz*z)/2 for sx in (-1, 1) for sy in (-1, 1) for sz in (-1, 1)]
    for ids in [[0, 1, 3, 2], [4, 6, 7, 5], [0, 4, 5, 1], [2, 3, 7, 6], [0, 2, 6, 4], [1, 5, 7, 3]]:
        m.poly([pts[i] for i in ids], color)


def panel(top, bottom, width):
    """Cape panel from the top edge centre to the bottom edge centre: faction cloth, white side edges and hem."""
    across = V([width, 0, 0])
    c, z = (top + bottom)/2, top - bottom
    obox(c, across, V([0, .07, 0]), z, BLUE)
    for s in (-1, 1):
        obox(c + across*s/2, V([.06, 0, 0]), V([0, .09, 0]), z, TRIM)
    obox(bottom, across + V([.02, 0, 0]), V([0, .09, 0]), V([0, 0, .07]), TRIM)


def cape(p, T, swing=0.):
    m.ell(T([0, .03, 1.36]), [.47, .32, .2], BLUE)                 # mantle over the shoulders
    for s in (-1, 1):
        m.box(T([s*.2, -.27, 1.27]), [.07, .05, .3], TRIM)         # open front edges over the chest
    top = T([0, .27, 1.4])
    if p.get('seated'):                                            # stops at the hips, the rest lies behind
        hip = T([0, .3, .8])
        panel(top, hip, .84)
        ground = V([0, hip[1] + .5, .06])
        obox((hip + ground)/2, V([.84, 0, 0]), V([0, 0, .07]), ground - hip, BLUE)
        obox(ground, V([.86, 0, 0]), V([0, .09, 0]), V([0, 0, .07]), TRIM)
    else:
        panel(top, T([0, .3 + .1 + swing, .42]), .84)


def ruyi(hand, d, gold=GOLD):
    """Gold rod bent forward at the top, cloud head of three lobes across the view (~6 px wide)."""
    side = U.facing(d)
    fwd = np.cross(side, d)
    fwd = fwd if fwd[1] <= 0 else -fwd                             # the head tips away from the body
    a, b = hand - d*.2, hand + d*.28
    c = b + d*.2 + fwd*.08
    m.rod(a, b, .045, gold)
    m.rod(b, c, .045, gold)
    up = A.unit(d + fwd*.5)
    head = c + up*.12
    m.ell(head + up*.04, [.15]*3, gold)
    for s in (-1, 1):
        m.ell(head - up*.05 + side*s*.17, [.1]*3, gold)
    m.ell(a, [.06]*3, gold)
    return head


def dressed(p, swing=0.):
    _, head = U.soldier(p, 'grand_strategist', weapon=False)
    T = U.transform(p)
    cape(p, T, swing)
    return T, head


def holding(p, block=False, swing=0.):
    T, head = dressed(p, swing)
    if block:     # held across in front of the chin: kept behind the face
        ruyi(T(p['hands'][0]), A.unit(V(p['blade'])), GOLD_BACK)
    elif p.get('seated'):  # propped up beside the knee (lying forward it would leave the cell)
        ruyi(T(p['hands'][0]), A.unit(V([-.3, -.55, .6])))
    else:
        ruyi(T(p['hands'][0]), A.unit(V(p['blade']) + OUT))
    return head


# attack: hand, ruyi direction, body
ATTACK = [dict(hand=[-.45, -.42, 1.32], dir=[-.1, -.25, 1], lean=.02),
          dict(hand=[-.6, -.2, 1.62], dir=[-.25, .15, 1], lean=-.03, crouch=.2, bob=-.02),
          dict(hand=[-.3, -.86, 1.28], dir=[0, -.75, -.55], lean=.18, lunge=.5, bob=-.04),
          dict(hand=[-.32, -.76, 1.18], dir=[0, -.7, -.65], lean=.12, lunge=.35, bob=-.02)]
SHOULDER = V([-.4, 0, 1.42])
SIGIL = V([0, -.85, .02])  # centre of the ground sigil, in front of the feet


def sigil(alpha, grow=1.):
    # two thin rings only: at 1x any glyph lines inside fill the ellipse into a puddle
    return [dict(kind='ring', center=SIGIL, axis=UP, radius=.62*grow, alpha=alpha, layer='under'),
            dict(kind='ring', center=SIGIL, axis=UP, radius=.3*grow, alpha=alpha, layer='under')]


def casting(f):
    k = ATTACK[f]
    p = U.pose(0, 0)
    p['row'] = U.ATTACK
    p['lean'], p['bob'] = k['lean'], k.get('bob', 0)
    A.stance(p, k)
    p['hands'][0] = list(k['hand'])
    p['elbows'][0] = ((SHOULDER + V(k['hand']))/2 + [-.16, .06, -.08]).tolist()
    T, head = dressed(p, swing=.08 if f >= 2 else 0)
    tip = ruyi(T(k['hand']), A.unit(V(k['dir'])), GOLD_BACK if f < 2 else GOLD)
    effects = []
    if f == 2:
        effects = sigil(1.) + [dict(kind='lines', segments=[(tip, SIGIL)], alpha=.7, layer='over')]
    elif f == 3:
        effects = sigil(.55, 1.06)
    return U.result(m.meshes, head, U.ATTACK, f, effects)


def grand(row, f):
    if row == U.ATTACK:
        return casting(f)
    if row == U.HIT:
        p = U.hit_pose(f, U.pose(0, 0))
        swing = .25*p['lean']
    else:
        p = U.pose(row, f)
        swing = .06*(f % 2) if row == 1 else 0.
    head = holding(p, swing=swing)  # before reading m.meshes: units.soldier rebinds the list
    return U.result(m.meshes, head, row, f)


def block(f):
    k = Bl.SWORD[Bl.SELECTED]['frames'][f]
    # the guard is held at chest height, not the sword's chin height: across the chin it read as a gold beard
    head = holding(Bl.foot_pose(dict(k, hand=list(V(k['hand']) + [0, -.08, -.24]))), block=True)
    return Bl.displace(U.result(m.meshes, head, 0, f), k.get('sway', 0), k.get('back', 0))


def make():
    U.KITS['grand_strategist'] = KIT
    R.add_ramps({TRIM: '5566', GOLD_BACK: 'kllm'})
    M.KEY2TAG.update({'7': 'g', '8': 'h', '9': 'i', 'p': 'F', 'q': 'E'})  # hair keys; tired-face sweat -> effects
    U.WEAPON_COLORS.add(GOLD)  # the ruyi (the only 3D gold) is drawn over the stamped head where it passes in front

    def build(row, f, style=None):
        return block(f) if row == B.BLOCK else grand(row, f)
    return build, HEADS


M.register('grand_strategist', make)
