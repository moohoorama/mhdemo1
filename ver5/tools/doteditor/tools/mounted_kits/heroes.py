"""장수 전용 키트: 조조(친위대 + 관모), 여포(중기병 + 방천화극), 원소(친위대 + 금투구), 조운(은빛 갑옷 기병),
손견(두건 쓴 경기병), 헌제(바닥에 끌리는 도포의 서생), 원술(가는 몸에 금 문양의 도보 장수),
조운·원소의 다른 기병 계열(경기병·중기병·친위대) 시트.

Each unit reuses an existing kit and changes the head (2D grids below), the armour or the body:
- heads: the S view is slimmed by one interior column per side so the face is as narrow as the 경기병's; the iron
  helmet's 'n' highlight is merged into 'm' (the horse coat owns the key 'n' here); hair and beards use the hair keys
  7 8 9 (hair1-3; the beard is the lightest, hair1); the red plume, crown and gold trim use k l m (themeC).
- 조조·조운 (검): the same 조조 / 조운 with a straight sword instead of the spear. It reuses the 관우 mount code: units.glaive_at is
  replaced by a sword drawn from the same hand/direction/side (so the 언월도 횡베기 swing and its arc follow it) and the
  weapon reach is cut to about 2/3 of the glaive's, still longer than the rider's arm so it reads as reaching the enemy.
- 여포 (활): horse_archer's bow, quiver and shot poses on 여포's heavy armour and barding, with 여포's head.
- 조조/원소: royal_guard with its faction crest above the helmet removed (cap) or halved (원소; a third of it from the front).
- 조운: rider with the 중기병's cuirass, pauldrons, vambraces and short cloak but no barding on the horse.
- 헌제: the 서생 body with a robe that trails on the floor.
- 원술: infantry body squeezed to 78% width with gold embroidery (hem ring, medallion, collar band, dots, cuffs) and
  the tall faction crest moved over from 원소.
- 조운 경기병·친위대, 원소 경기병·중기병: the same head on the light rider, or on the 친위대 (crest dropped, his plume
  stands) / 중기병 kit, so the sheet changes with the class (Look.Sprites).
Colours (horse coat, silver steel, hair, plume) are set per tag with grid.py colors after the conversion.
"""
import numpy as np

import grid3d_mounted as M
from mounted_kits import heavy_cavalry as HC, horse_archer as HA, royal_guard as RG, scholar as SC
from story_heads import bearded as story_bearded

G, U, Hd, B, Bl = M.G, M.U, M.Hd, M.B, M.Bl
m = G.m
BLUE, LIGHT_BLUE, GOLD = U.BLUE, U.LIGHT_BLUE, U.GOLD
HAIR_KEYS = {'f': '8', 'g': '9'}


# ---- head grids ----------------------------------------------------------------------------
def slim_rows(rows, n=1):
    """Take n interior pixels off each side of every wide row (outline and shading stay), keep the width."""
    out = []
    for r in rows:
        xs = [i for i, c in enumerate(r) if c != '.']
        if not xs or xs[-1] - xs[0] < 8 + 2 * (n - 1):
            out.append(r)
            continue
        a, b = xs[0], xs[-1]
        drop = set(range(a + 2, a + 2 + n)) | set(range(b - 1 - n, b - 1))
        out.append('.' * n + ''.join(c for i, c in enumerate(r) if i not in drop) + '.' * n)
    return out


def slim(grids, n=1):
    return {d: (slim_rows(r, n) if d == 'S' else r) for d, r in grids.items()}


def bearded(grids, kind, keep=None):
    out = {}
    for d, r in grids.items():
        r = Hd.beard(r, 'ij', kind)
        out[d] = r[:keep] if keep else r
    return out


def recolor(grids, mapping):
    return {d: Hd.recolor(r, mapping) for d, r in grids.items()}


def put(grid, points, ch):
    g = [list(r) for r in grid]
    for y, x in points:
        g[y][x] = ch
    return [''.join(r) for r in g]


IRON_GOLD = {**Hd.GOLD_IRON, '5': 'm'}
GOLD_IRON = recolor(Hd.IRON, IRON_GOLD)
GOLD_CAP = {d: Hd.recolor(Hd.flat_cap(r), {'b': 'Y'}) for d, r in GOLD_IRON.items()}
CAP = {d: [(r.replace('m', 'Y') if y != 3 else r.replace('2', 'X')) for y, r in enumerate(Hd.gwanmo(rows, 4))]
       for d, rows in Hd.HAIR.items()}

CAOCAO = recolor(bearded(slim(CAP), 'bristle'), {**HAIR_KEYS, '1': '7'})
LUBU = Hd.crested(slim(GOLD_IRON), Hd.FEATHERS)
YUANSHAO = bearded(slim(GOLD_IRON), 'goatee')
SUNJIAN = recolor(bearded(slim(Hd.TOPKNOT), 'bristle'), {**HAIR_KEYS, '1': '7'})

# Zhao Yun: silver helmet with a red brow band; the plume rises from the crown and runs down the back of the head
# (a rough 2-3px feather, dark red and red alternating). Side views: T = the helmet's top outline row, band = the brow row.
SIDE = {'SW': (3, 6), 'W': (2, 5), 'NW': (2, 5)}
TUFT = [(-2, 7), (-2, 8), (-1, 6), (-1, 7), (-1, 8), (0, 6), (0, 7), (0, 8)]
RIDGE = [(1, 7), (1, 8), (1, 9), (2, 8), (2, 9), (2, 10), (3, 9), (3, 10), (3, 11), (4, 10), (4, 11), (5, 11), (6, 11)]
FRONT_BACK = {'S': ([(0, 6), (1, 6), (1, 7)] + [(y, x) for y in range(2, 6) for x in (5, 6, 7)], 5),
              'N': ([(0, 6), (1, 6), (2, 6), (1, 7)] + [(y, x) for y in range(3, 11) for x in (5, 6, 7)], 5)}


def feather(y, x):
    return 'l' if (x + y) % 2 == 0 else 'k'


def side_plume(rows, d):
    """Round the helmet's back-top corner (a square block otherwise), then paint the plume with its outline."""
    T, band = SIDE[d]
    g = [list(r) for r in rows]
    for y in range(T - 2, T):
        g[y] = ['.'] * 13
    g[T] = ['0' if c == 'b' else c for c in g[T]]
    g[T][10] = g[T][11] = '.'
    g[T + 1][11] = g[T + 1][12] = '.'
    g[T + 1][10] = '0'
    g[T + 2][12] = '.'
    g[T + 2][11] = '0'
    g[band] = ['l' if c == '3' else c for c in g[band]]
    cells = [(T + dy, x) for dy, x in TUFT + RIDGE]
    for y, x in cells:
        g[y][x] = feather(y, x)
    g[T - 2][7] = 'm'
    for y, x in cells:
        for dy, dx in ((1, 0), (-1, 0), (0, 1), (0, -1)):
            ny, nx = y + dy, x + dx
            if 0 <= ny < len(g) and 0 <= nx < 13 and g[ny][nx] == '.':
                g[ny][nx] = '0'
    return [''.join(r) for r in g]


def zhaoyun_head(grids):
    out = {}
    for d, rows in grids.items():
        if d in SIDE:
            out[d] = side_plume(rows, d)
            continue
        pts, band = FRONT_BACK[d]
        g = [list(r.replace('3', 'l') if y == band and d != 'N' else r) for y, r in enumerate(rows)]
        for y, x in pts:
            g[y][x] = feather(y, x)
        g[pts[0][0]][pts[0][1]] = 'm'
        out[d] = [''.join(r) for r in g]
    return out


ZHAOYUN = zhaoyun_head(slim(Hd.IRON))


def mianguan(grids):
    """Emperor's crown: a flat gold-trimmed board on top, bead strands (iron1) hanging at the sides (front) or the back."""
    tops = {
        'S': ['..000000000..', '.0lllllllll0.', '.0kkkkkkkkk0.', '..0ggggggg0..'],
        'SW': ['...000000000.', '..0lllllllll0', '..0kkkkkkkkk0', '...00ggggg00.'],
        'W': ['..000000000..', '.0lllllllll0.', '.0kkkkkkkkk0.', '...00gggggg00'],
        'NW': ['...000000000.', '..0lllllllll0', '..0kkkkkkkkk0', '...00ggggg00.'],
        'N': ['..000000000..', '.0lllllllll0.', '.0kkkkkkkkk0.', '..0ggggggg0..'],
    }
    beads = {'S': [(y, x) for y in range(3, 9) for x in (1, 11)],
             'SW': [(y, x) for y in range(3, 9) for x in (2, 11)],
             'W': [(y, x) for y in range(3, 9) for x in (1, 11)],
             'NW': [(y, x) for y in range(3, 8) for x in (4, 6, 8)],
             'N': [(y, x) for y in range(3, 8) for x in (4, 6, 8)]}
    return {d: put(tops[d] + rows[4:], beads[d], '5') for d, rows in grids.items()}


XIANDI = recolor(mianguan(Hd.CROWN), HAIR_KEYS)

# Yuan Shu: gold cap, a one-row chin beard, a narrow face; the non-frontal views lose two interior columns and a dome row
SHRINK = {'SW': ((5, 8), 5), 'W': ((5, 9), 4), 'NW': ((6, 9), 4), 'N': ((4, 8), 5)}


def shrink(grids):
    out = {}
    for d, rows in grids.items():
        if d not in SHRINK:
            out[d] = rows
            continue
        cols, row = SHRINK[d]
        rows = [r for i, r in enumerate(rows) if i != row]
        rows = ['.' + ''.join(c for i, c in enumerate(r) if i not in cols) + '.' for r in rows]
        out[d] = ['.' * 13] + rows
    return out


YUANSHU = bearded(shrink(slim(GOLD_CAP, 2)), 'goatee', keep=13)


# ---- bodies --------------------------------------------------------------------------------
_ROYAL_ARMOUR = RG.rider_armour


def royal_armour(crest):
    """royal_guard's armour with the faction crest above the 2D helmet dropped (crest False) or halved ('half')."""
    def armour(p):
        _ROYAL_ARMOUR(p)
        m.meshes[:] = [(v, c) for v, c in m.meshes
                       if not (np.asarray(v)[:, 2].min() > 2.45 and abs(np.asarray(v)[:, 0]).max() < .25)]
        if crest == 'half':
            T = U.transform(p)
            t = U.VIEW['t']
            k = .4 if t is not None and abs(t[1]) > .9*np.hypot(t[0], t[1]) else 1  # from the front the 2D tuft adds to it
            m.rod(T([0, .02, 2.52]), T([0, .02 + .03*k, 2.52 + .16*k]), .09, BLUE)
            m.ell(T([0, .08, 2.52 + .24*k]), [.11, .15, .07*k], LIGHT_BLUE)
            m.ell(T([0, .02, 2.5]), [.11, .12, .06], GOLD)
    return armour


def hair_tags():
    M.KEY2TAG.update({'7': 'g', '8': 'h', '9': 'i'})


def make_caocao():
    M.UNITS['royal_guard']()
    RG.rider_armour = royal_armour(False)
    hair_tags()
    return M.rider_build(RG.KIT), M.Hd.heads(CAOCAO)


BLADE, EDGE = '#8a96a6', '#c3ccd8'  # a dull grey blade (iron3/iron2 tags) with a paler edge (iron2/iron1)


def sword_at(hand, d, side):
    """Straight sword for the glaive's slot (units.glaive_at): grip, gold guard, tapering blade 1.35 long."""
    m.rod(hand - d*.15, hand + d*.12, .05, U.SHAFT)
    base = hand + d*.17
    m.rod(base - side*.2, base + side*.2, .05, GOLD)
    tip, mid = base + d*1.35, base + d*.85
    m.poly([base - side*.09, mid - side*.1, tip, mid + side*.14, base + side*.09], BLADE)
    m.poly([base, mid, tip, mid + side*.14, base + side*.09], EDGE)


def swordsman(inner):
    """Swap the glaive for the sword (shorter reach, same swing) and make the sweep the default attack."""
    U.MOUNT_REACH['glaive'] = (.45, 1.5)
    U.glaive_at = sword_at
    M.R.add_ramps({BLADE: '2334', EDGE: '3445'})
    U.WEAPON_COLORS.update({BLADE, EDGE})

    def build(row, f, style=None):
        return inner(row, f, style or 'glaive_sweep')
    return build


def make_caocao_sword():
    M.UNITS['royal_guard']()
    RG.rider_armour = royal_armour(False)
    hair_tags()
    return swordsman(M.rider_build(RG.KIT, weapon='glaive')), M.Hd.heads(CAOCAO)

def lubu_steel():
    """The halberd's blades, spear point and crescent in the hair tags (set to bright steel on the sheet): 여포's iron
    tags are his near-black armour, so steel drawn in them came out as an outline only. The plume root that used
    hair2 moves to horseMane2, the same dark red."""
    hair_tags()
    M.KEY2TAG['b'] = 'z'
    M.R.add_ramps({'#d6e7ec': '9877'})


def make_lubu():
    M.UNITS['heavy_cavalry']()
    lubu_steel()
    return M.rider_build(HC.KIT, weapon='halberd'), M.Hd.heads(LUBU, crest=Hd.FEATHER_ROWS)


def make_lubu_unarmed():
    """여포 on 적토마 with empty hands: the story scene where he has flung his halberd into the ground."""
    M.UNITS['heavy_cavalry']()
    lubu_steel()
    return M.rider_build(HC.KIT, weapon='none'), M.Hd.heads(LUBU, crest=Hd.FEATHER_ROWS)


def make_liubei_mounted():
    """유비 riding (story scenes): his scene robe (the white coat, gold shoulders, leather belt, white inner collar of
    civilian.py's chosen look, as civ3d.py dresses it) on the light rider, his black topknot under the small gold
    crown and goatee; no weapon."""
    import civilian as C
    hair_tags()
    _, walk, _ = C.LOOKS['liubei'][C.SELECTED['liubei']]
    kit = walk.kit + '_rider'
    # on horseback the long robe covers most of him: coat and robe are white (his chosen look), not the team colour
    U.KITS[kit] = dict(U.KITS[walk.kit], shoulder=U.GOLD, tabard=U.BROWN, scarf=C.WHITE, scarf_tail=.12)
    import scene_people as SP
    for part in ('cloth', 'robe', 'guard'):  # IVORY is the hero robe colour that converts to themeB (white on his sheet)
        U.KITS[kit][part] = SP.IVORY
    crown = {d: [''.join(r) for r in rows] for d, rows in C.SMALL_CROWN.items()}  # its crown row comes as a list
    head = recolor(bearded(slim(crown), 'goatee'), {**HAIR_KEYS, '1': '7'})
    return M.rider_build(kit, weapon='none'), M.Hd.heads(head)


def make_lubu_bow():
    M.UNITS['heavy_cavalry']()
    M.KEY2TAG[M.MOUTH] = 'h'
    cavalry = HA.archer_cavalry(U.cavalry)
    U.cavalry = cavalry
    block = Bl.rider(kit=HC.KIT, coat='dot', weapon='none')

    def build(row, f, style=None):
        if row == B.BLOCK:
            return block(f, Bl.SELECTED)
        return cavalry(row, f, style, kit=HC.KIT)
    return build, M.Hd.heads(LUBU, crest=Hd.FEATHER_ROWS)


def make_yuanshao():
    M.UNITS['royal_guard']()
    RG.rider_armour = royal_armour('half')
    return M.rider_build(RG.KIT), M.Hd.heads(YUANSHAO, Hd.PLUME)


SILVER_KIT = 'rider_silver'


def silver_rider():
    """The 중기병's cuirass, pauldrons, vambraces and short cloak on the light rider, without barding on the horse."""
    U.KITS[SILVER_KIT] = dict(U.KITS['rider'], shoulder=BLUE)
    soldier = U.soldier

    def silver_soldier(p, k, weapon=True):
        out = soldier(p, k, weapon)
        if k == SILVER_KIT:
            HC.rider_armour(p)
        return out

    U.soldier = silver_soldier


def make_zhaoyun():
    silver_rider()
    return M.rider_build(SILVER_KIT), M.Hd.heads(ZHAOYUN)


def make_zhaoyun_sword():
    silver_rider()
    return swordsman(M.rider_build(SILVER_KIT, weapon='glaive')), M.Hd.heads(ZHAOYUN)

def make_sunjian():
    hair_tags()
    return M.rider_build('rider'), M.Hd.heads(SUNJIAN, Hd.BAND)


def make_xiandi():
    body = SC.body

    def long_body(p):
        T = body(p)
        m.ell(T([0, -.01, .4]), [.47, .35, .46], SC.ROBE)
        m.ell(T([0, .0, .14]), [.56, .43, .2], SC.ROBE)
        m.ell(T([0, -.02, .05]), [.62, .5, .08], LIGHT_BLUE)
        return T

    SC.body = long_body
    build, _ = M.UNITS['scholar']()
    return build, M.Hd.heads(XIANDI)


YUANSHU_KIT = 'hero_yuanshu'


def make_yuanshu():
    U.KITS[YUANSHU_KIT] = dict(cloth=BLUE, tabard=LIGHT_BLUE, guard=GOLD, robe=BLUE, shield=False, weapon='sword')
    soldier = U.soldier

    def ornate_soldier(p, kit, weapon=True):
        faces, head = soldier(p, kit, weapon)
        if kit != YUANSHU_KIT:
            return faces, head
        m.meshes = [(v*[.78, 1, 1], c) for v, c in faces]
        T = U.transform(p)
        m.ell(T([0, -.01, .3]), [.32, .24, .05], GOLD)
        m.ell(T([0, -.27, 1.2]), [.1, .04, .1], GOLD)
        m.rod(T([-.1, -.27, 1.5]), T([.04, -.3, 1.08]), .045, GOLD)
        for z in (.45, .62, .8):
            for x in (-.13, .0, .13):
                m.ell(T([x, -.29, z]), [.035, .03, .035], GOLD)
        m.rod(T([0, .02, 2.2]), T([0, .08, 2.63]), .09, BLUE)
        m.ell(T([0, .12, 2.7]), [.11, .15, .14], LIGHT_BLUE)
        m.ell(T([0, .02, 2.18]), [.11, .12, .06], GOLD)
        for i in range(2):
            el, ha = T(p['elbows'][i]), T(p['hands'][i])
            m.ell(el*.35 + ha*.65, [.12, .12, .12], GOLD)
            m.ell(T([(-.32 if i == 0 else .32), 0, 1.42]), [.12, .12, .1], GOLD)
        return m.meshes, head

    U.soldier = ornate_soldier
    inner, block = U.foot_unit(YUANSHU_KIT), Bl.foot(YUANSHU_KIT)

    def build(row, f, style=None):
        return block(f, Bl.SELECTED) if row == B.BLOCK else inner(row, f, style)
    return build, M.Hd.heads(YUANSHU)


ZHANGLIAO_KIT = 'rider_zhangliao'
# 장료 (his scene sheet, civ3d.py): the dark iron helmet with the team plume and a goatee in hair2
ZHANGLIAO = recolor(story_bearded(slim({d: Hd.recolor(r, Hd.DARK_IRON) for d, r in Hd.IRON.items()}), 'goatee', 'g'),
                    {'g': '8'})


def make_zhangliao():
    """장료: his scene look on the light rider — team tunic, a navy coat over the thighs (RED -> themeB), steel
    guards, a leather belt — with a crescent glaive (언월도, as in his portrait) swung like 관우's."""
    hair_tags()
    U.KITS[ZHANGLIAO_KIT] = dict(U.KITS['rider'], robe=U.RED, tabard=U.BROWN)
    inner = M.rider_build(ZHANGLIAO_KIT, weapon='glaive')

    def build(row, f, style=None):
        return inner(row, f, style or 'glaive_sweep')
    return build, M.Hd.heads(ZHANGLIAO, Hd.PLUME)


# ---- the same heroes in the other classes of their line (경기병 -> 중기병 -> 친위대) ---------------------
def make_zhaoyun_light():
    return M.rider_build('rider'), M.Hd.heads(ZHAOYUN)


def make_zhaoyun_royal():
    M.UNITS['royal_guard']()
    RG.rider_armour = royal_armour(False)  # his own plume, no faction crest
    return M.rider_build(RG.KIT), M.Hd.heads(ZHAOYUN)


def make_yuanshao_light():
    return M.rider_build('rider'), M.Hd.heads(YUANSHAO, Hd.PLUME)


def make_yuanshao_heavy():
    M.UNITS['heavy_cavalry']()
    return M.rider_build(HC.KIT), M.Hd.heads(YUANSHAO, Hd.PLUME)


M.register('caocao', make_caocao, vertical_spear=True)
M.register('caocao_sword', make_caocao_sword)
M.register('lubu', make_lubu, vertical_spear=True)
M.register('lubu_bow', make_lubu_bow)
M.register('lubu_unarmed', make_lubu_unarmed)
M.register('liubei_mounted', make_liubei_mounted)
M.register('yuanshao', make_yuanshao, vertical_spear=True)
M.register('zhaoyun', make_zhaoyun, vertical_spear=True)
M.register('zhaoyun_sword', make_zhaoyun_sword)
M.register('sunjian', make_sunjian, vertical_spear=True)
M.register('zhangliao', make_zhangliao)
M.register('xiandi', make_xiandi)
M.register('yuanshu', make_yuanshu)
for _name, _make in (('zhaoyun_light', make_zhaoyun_light), ('zhaoyun_royal', make_zhaoyun_royal),
                     ('yuanshao_light', make_yuanshao_light), ('yuanshao_heavy', make_yuanshao_heavy)):
    M.register(_name, _make, vertical_spear=True)
