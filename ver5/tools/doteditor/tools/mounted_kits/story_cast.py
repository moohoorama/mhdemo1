"""The later story cast in their battle classes (2026-10-30): 제갈량·사마의·주유·육손 (서생 -> 참모 -> 군사),
황충 (궁병 -> 노병 -> 연노병), 여몽 (경보병 -> 중보병 -> 근위병). Sheet ids: <hero> for the first class,
<hero>_<class> for the others (Look.Sprites), so a hero only promotes along the line it has sheets for.

Each sheet is the class kit with the hero's head (story_heads.py, the same heads as their scene sheets) in the hero's
colours: the 서생 body's materials swapped (SWAP); the bodies built by units.soldier (참모, 군사, 궁병, 노병, 연노병,
the infantry) in a kit of the hero's own (KITS), the 군사 cape in CAPE. Materials beyond the five sheet groups go to
the horse tags, unused on a sheet without a horse: XTRA renders in N O P (-> horseSkin), MANE and FAN in K L M
(-> horseMane), SADDLE in Q R S (-> saddle). The team colour stays on part of the body so the side still reads in
battle (the 서생's trousers, the tunic under the robe). Colours are set per tag with grid.py colors after the
conversion (COLORS).
"""
import grid3d_mounted as M
from mounted_kits import grand_strategist as GS, repeating_crossbow as RC, scholar as SC, staff_officer as SO
from story_heads import DARK_TOPKNOT, bearded, cap, helmet

G, U, A, B, Bl, Hd, R = M.G, M.U, M.A, M.B, M.Bl, M.Hd, M.R
m = U.m

XTRA, MANE, SADDLE = '#5a3a7a', '#a0382a', '#4ab0b0'
FAN, FAN_BACK = '#e8e8f0', '#e8e8f1'  # white feathers; FAN_BACK is not a weapon colour (behind the head)
BLADE, BLADE_BACK = '#bacbd6', '#bacbd7'  # a weapon colour (drawn over the head) and the same steel behind it
EXTRA_RAMPS = {XTRA: 'NOPP', MANE: 'KLMM', SADDLE: 'QRSS', FAN: 'KLMM', FAN_BACK: 'KLMM', BLADE_BACK: '4556'}
EXTRA_KEYS = {'N': 'x', 'O': 'w', 'P': 'v', 'K': 'A', 'L': 'z', 'M': 'y', 'Q': 'D', 'R': 'C', 'S': 'B'}
HAIR_KEYS = {'f': '9', 'g': '8', 'h': '7'}  # the scene heads' hair keys -> the battle kits' hair keys
LEATHER, STEEL, TEAM, CLOTH, GOLD = U.BROWN, U.STEEL, U.LIGHT_BLUE, U.RED, U.GOLD

HEADS = {
    'zhugeliang': bearded(cap('45', 'lm'), 'goatee', 'g'),
    'simayi': bearded(cap('RS', 'RR'), 'long', 'd'),
    'luxun': helmet('bcd', 'klm'),
    'zhouyu': helmet('KLM', 'QRS'),
    'huangzhong': bearded(Hd.HAIR, 'long', 'h'),
    'lumeng': bearded(DARK_TOPKNOT, 'bristle', 'g'),
}

# 서생: body material -> the hero's
SWAP = {
    'zhugeliang': {SC.ROBE: CLOTH, SC.TROUSERS: TEAM, SC.BROWN: XTRA, SC.SCROLL: GOLD},
    'simayi': {SC.ROBE: LEATHER, SC.TROUSERS: TEAM, SC.BROWN: XTRA, SC.SCROLL: GOLD},
    'zhouyu': {SC.ROBE: CLOTH, SC.TROUSERS: TEAM, SC.BROWN: XTRA, SC.SCROLL: GOLD},
    'luxun': {SC.ROBE: CLOTH, SC.TROUSERS: TEAM, SC.BROWN: XTRA, SC.SCROLL: GOLD},
}
# the hero's own kit (units.soldier parts) for every other class
ROBED = dict(shield=False, weapon=None)
KITS = {
    'zhugeliang': dict(ROBED, cloth=TEAM, robe=CLOTH, arms=CLOTH, shoulder=CLOTH, guard=LEATHER, tabard=XTRA,
                       scarf=STEEL),
    'simayi': dict(ROBED, cloth=TEAM, robe=LEATHER, arms=LEATHER, shoulder=LEATHER, guard=STEEL, tabard=XTRA,
                   scarf=STEEL),
    'zhouyu': dict(ROBED, cloth=TEAM, robe=CLOTH, arms=CLOTH, shoulder=GOLD, guard=STEEL, tabard=LEATHER, scarf=XTRA),
    'luxun': dict(ROBED, cloth=TEAM, robe=CLOTH, arms=LEATHER, shoulder=STEEL, guard=STEEL, tabard=CLOTH, scarf=XTRA),
    'huangzhong': dict(U.KITS['archer'], guard=STEEL, shoulder=STEEL, tabard=XTRA),
    'lumeng': dict(G.LIGHT_KIT, guard=LEATHER, shoulder=LEATHER),
}
# what the 책사 holds in every class of the line (2026-10-31): 사마의 the scroll (죽간), the others HELD below
SCROLL_ONLY = {'simayi'}
CAPE = {'zhugeliang': CLOTH, 'simayi': CLOTH, 'zhouyu': XTRA, 'luxun': XTRA}


def tags(group, colors):
    return [f'{group}{i + 1}={c}' for i, c in enumerate(colors)]


HAIR = tags('hair', ['#5a4638', '#352920', '#1c1614'])
WHITE = ['#f6f2e6', '#d6cfbc', '#9a927e']
GOLDS = ['#f0d070', '#c8a040', '#806020']
SILVER = tags('iron', ['#e0e4ea', '#a0a8b4', '#5e6670'])
WHITE_FAN = tags('horseMane', ['#ffffff', '#c8ccd8', '#7a7e90'])
COLORS = {
    'zhugeliang': HAIR + tags('themeB', WHITE) + tags('themeA', ['#6a8cc8', '#3a5a96', '#1e3060'])
    + tags('iron', ['#86dcd2', '#3aa09a', '#1e6260']) + tags('themeC', GOLDS)
    + tags('horseSkin', ['#8a7aa0', '#5e4e78', '#3a2e50']) + WHITE_FAN,
    'simayi': tags('hair', ['#d6d4ce', '#9e9c96', '#605e5a']) + tags('themeB', ['#4a4a52', '#2a2a30', '#16161a'])
    + tags('themeA', ['#8a7a5a', '#5e5038', '#3a3022']) + tags('iron', ['#a4a4e4', '#6262b4', '#38387c'])
    + tags('themeC', GOLDS) + tags('horseSkin', ['#a04a5a', '#70303e', '#481e28'])
    + tags('saddle', ['#7a4466', '#58304c', '#381c30']) + WHITE_FAN,
    'zhouyu': HAIR + tags('themeB', WHITE) + tags('themeA', ['#7a5a40', '#523a28', '#30221a']) + SILVER
    + tags('themeC', GOLDS) + tags('horseSkin', ['#9a78c8', '#6a4a9a', '#40286a'])
    + tags('horseMane', ['#d0604a', '#a0382a', '#682018']) + tags('saddle', ['#8ae0dc', '#4ab0b0', '#2a7276']),
    'luxun': HAIR + tags('themeA', ['#a08058', '#74583a', '#4a3622']) + tags('themeB', ['#5a6aa8', '#34447a', '#1e284a'])
    + SILVER + tags('themeC', ['#ffe680', '#e8c030', '#a07818'])
    + tags('horseSkin', ['#7ab85a', '#4a8a3a', '#2a5a24']) + tags('horseMane', ['#d0604a', '#a0382a', '#682018']),
    # 황충: white hair and beard, the red sash in XTRA (themeB is the bow wood / the crossbow stock)
    'huangzhong': tags('hair', ['#f0eee8', '#c2c0ba', '#7c7a76'])
    + tags('horseSkin', ['#c8604a', '#963a2c', '#5e221c']),
    'lumeng': HAIR + tags('themeA', ['#9a6a42', '#6c4528', '#442a18']),
}


def swapped(build, swap):
    def recoloured(row, f, style=None):
        out = build(row, f, style)
        out['faces'] = [(v, swap.get(c, c)) for v, c in out['faces']]
        return out
    return recoloured


def setup(hero):
    R.add_ramps(EXTRA_RAMPS)
    M.KEY2TAG.update(EXTRA_KEYS, b='r', **{'7': 'g', '8': 'h', '9': 'i'})
    U.WEAPON_COLORS.update({FAN, BLADE})
    return Hd.heads({d: Hd.recolor(rows, HAIR_KEYS) for d, rows in HEADS[hero].items()})


def treasure_sword(hand, d, blade):
    """보검: gold pommel and guard, a red-bound grip, a straight blade 1 long laid across the view; returns the tip."""
    side = U.facing(d)
    m.ell(hand - d*.15, [.06]*3, GOLD)
    m.rod(hand - d*.12, hand + d*.12, .05, MANE)
    guard = hand + d*.15
    m.rod(guard - side*.17, guard + side*.17, .055, GOLD)
    base, tip = hand + d*.19, hand + d*1.19
    mid = base + (tip - base)*.82
    m.poly([base - side*.075, mid - side*.075, tip, mid + side*.075, base + side*.075], blade)
    return tip


# (prop, colour, colour behind the head) for staff_officer.build: 제갈량 the white feather fan (백우선), 주유·육손
# a treasure sword (보검)
HELD = {'zhugeliang': (SO.fan_at, FAN, FAN_BACK), 'zhouyu': (treasure_sword, BLADE, BLADE_BACK),
        'luxun': (treasure_sword, BLADE, BLADE_BACK)}


def scroll_body(kit, cape=False):
    """scholar.body's stand-in: the officer's robe kit (and the 군사 cape), so the 서생's scroll motions go on."""
    def body(p):
        U.soldier(p, kit, weapon=False)
        T = U.transform(p)
        if cape:
            GS.cape(p, T)
        return T
    return body


def scholar(hero):
    def make():
        build, _ = M.UNITS['scholar']()
        if hero in HELD:
            build = SO.build(lambda p: SC.body(p)(U.HEAD), *HELD[hero])
        return swapped(build, SWAP[hero]), setup(hero)
    return make


def staff_officer(hero):
    def make():
        M.UNITS['staff_officer']()
        U.KITS[hero] = KITS[hero]
        if hero in SCROLL_ONLY:
            build, _ = M.UNITS['scholar']()
            SC.body = scroll_body(hero)
            return swapped(build, {SC.SCROLL: GOLD}), setup(hero)
        return SO.build(SO.robed(hero), *HELD[hero]), setup(hero)
    return make


def strategist(hero):
    def make():
        build, _ = M.UNITS['grand_strategist']()
        U.KITS['grand_strategist'] = KITS[hero]
        if hero in SCROLL_ONLY:
            build, _ = M.UNITS['scholar']()
            SC.body = scroll_body('grand_strategist', cape=True)
            return swapped(build, {SC.SCROLL: GOLD, GS.BLUE: CAPE[hero]}), setup(hero)
        if hero in HELD:
            build = SO.build(lambda p: GS.dressed(p)[1], *HELD[hero])
        return swapped(build, {GS.BLUE: CAPE[hero]}), setup(hero)
    return make


def archer(hero):
    """궁병: units.archer (bow, quiver, shots) and blocks.bowman in the hero's kit."""
    def make():
        U.KITS[hero] = KITS[hero]
        block = Bl.bowman(hero)

        def build(row, f, style=None):
            return block(f, Bl.SELECTED) if row == B.BLOCK else U.archer(row, f, style, kit=hero)
        return build, setup(hero)
    return make


def crossbow(hero, kit):
    """노병 / 연노병: the class kit (its body drawn in units.KITS[kit]) in the hero's kit."""
    def make():
        build, _ = M.UNITS[kit]()
        U.KITS[kit] = KITS[hero]
        heads = setup(hero)
        M.KEY2TAG[M.MOUTH] = 'h'  # themeB is the pale wood
        return build, heads
    return make


def infantry(hero, kit, decorate):
    """경보병 / 중보병 / 근위병: units.foot_unit with grid3d's class decoration (straps and buckler, or cuirass,
    pauldrons, shield and cloak) on the hero's kit; the 중보병 and 근위병 keep their class kit's colours."""
    def make():
        name = f'{hero}_{kit}'
        U.KITS[name] = KITS[hero] if kit == 'light' else dict(G.HEAVY_KIT)
        soldier = U.soldier

        def decorated(p, k, weapon=True):
            out = soldier(p, k, weapon)
            if k == name:
                decorate(p)
            return out
        U.soldier = decorated
        A.SELECTED[name] = A.SELECTED['infantry']
        inner, block = U.foot_unit(name), Bl.foot(name)

        def build(row, f, style=None):
            return block(f, Bl.SELECTED) if row == B.BLOCK else inner(row, f, style) if style else inner(row, f)
        return build, setup(hero)
    return make


for _hero in SWAP:
    M.register(_hero, scholar(_hero))
    M.register(f'{_hero}_staff_officer', staff_officer(_hero))
    M.register(f'{_hero}_strategist', strategist(_hero))
M.register('huangzhong', archer('huangzhong'))
M.register('huangzhong_crossbow', crossbow('huangzhong', 'crossbow'))
M.register('huangzhong_repeating_crossbow', crossbow('huangzhong', RC.KIT))
M.register('lumeng', infantry('lumeng', 'light', G.decorate_light))
M.register('lumeng_heavy_infantry', infantry('lumeng', 'heavy', G.decorate_heavy))
M.register('lumeng_guards', infantry('lumeng', 'guards', G.decorate_guards))
