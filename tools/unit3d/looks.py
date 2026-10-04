"""ver4 look candidates: four SW looks per new unit or hero; the user picks one.

Every candidate stays defined here as a backup; build.UNITS takes the chosen one
(LOOKS[key][SELECTED[key]]). Kits are registered into units.KITS under '<key>_<n>'.

  python3 tools/unit3d/look_candidates.py   -> output/ver4-look-candidates.png
"""
import heads as Hd
import units as U

WHITE, HEMP, RED, GOLD = '#ede1bd', U.HEMP, U.RED, U.GOLD
BLUE, LIGHT, STEEL, BLACK, DARK = U.BLUE, U.LIGHT_BLUE, U.STEEL, U.BLACK, U.DARK_STEEL
PURPLE = '#4b315b'
U.RAMPS.update({PURPLE: '1223'})


def kit(cloth=BLUE, tabard=LIGHT, guard=STEEL, weapon=None, **extra):
    return dict(cloth=cloth, tabard=tabard, guard=guard, shield=extra.pop('shield', False), weapon=weapon, **extra)


def foot(key, k):
    U.KITS[key] = k
    return U.foot_unit(key)


def bowman(key, k):
    U.KITS[key] = k
    return lambda row, f, style=None: U.archer(row, f, style, kit=key)


def rider(key, k, coat, weapon, style):
    U.KITS[key] = k
    return lambda row, f, style_=None: U.cavalry(row, f, style_ or style, kit=key, coat=coat, weapon=weapon)


H = Hd.heads
DARK_IRON = {d: Hd.recolor(r, Hd.DARK_IRON) for d, r in Hd.IRON.items()}
GOLD_IRON = {d: Hd.recolor(r, Hd.GOLD_IRON) for d, r in Hd.IRON.items()}
CAP = {d: Hd.flat_cap(r) for d, r in DARK_IRON.items()}
GOLD_CAP = {d: Hd.flat_cap(r) for d, r in GOLD_IRON.items()}
WHITE_HOOD = {d: Hd.recolor(r, Hd.WHITE_HOOD) for d, r in Hd.HOOD.items()}
YELLOW_HOOD = {d: Hd.recolor(r, Hd.YELLOW_HOOD) for d, r in Hd.HOOD.items()}


# Liu Bei's ears: a small skin patch beside the face where the cap's side guard was.
EARS = {
    'SW': {8: '.0j0jj0ji0j10', 9: '.0j0jj0ji0i10'},
    'S': {7: '0jjj0jjj0jjj0', 8: '0ijj0jjj0jji0'},
    'W': {7: '.0j0jjji0jk10', 8: '0jj0jjji0ik10'},
    'NW': {7: '.0j0jllllkk10'},
}


def eared(grids):
    out = {}
    for d, rows in grids.items():
        rows = list(rows)
        for y, row in sorted(EARS.get(d, {}).items()):
            rows += ['.' * 13] * (y + 1 - len(rows))
            rows[y] = row
        out[d] = rows
    return out


def bearded(grids, kind, skin='ij'):
    return {d: Hd.beard(r, skin, kind) for d, r in grids.items()}


# (title, builder, head set) per candidate
LOOKS = {
    'strategist': [
        ('두건 · 진영색 도포 · 깃털부채', foot('strategist_1', kit(robe=BLUE, guard=BLUE, weapon='fan')), H(Hd.HOOD, Hd.HOOD_CLOTH)),
        ('흰 윤건 · 진영색 도포 · 깃털부채', foot('strategist_2', kit(robe=BLUE, guard=BLUE, weapon='fan')), H(WHITE_HOOD)),
        ('검은 관모 · 진영색 도포 · 깃털부채', foot('strategist_3', kit(robe=BLUE, weapon='fan', guard=BLUE)), H(CAP)),
        ('흰 윤건 · 흰 도포 · 진영색 띠', foot('strategist_4', kit(cloth=WHITE, robe=WHITE, guard=WHITE, weapon='fan')), H(WHITE_HOOD)),
    ],
    'liubei': [
        ('금관 · 진영색 옷 · 쌍검', foot('liubei_1', kit(weapon='sword', twin=True)), H(bearded(Hd.CROWN, 'goatee'))),
        ('금관 · 흰 도포 · 쌍검', foot('liubei_2', kit(cloth=WHITE, robe=WHITE, guard=GOLD, weapon='sword', twin=True)),
         H(bearded(Hd.CROWN, 'goatee'))),
        ('금빛 투구 · 진영색 옷 · 쌍검', foot('liubei_3', kit(guard=GOLD, weapon='sword', twin=True)),
         H(bearded(GOLD_IRON, 'goatee'), Hd.PLUME)),
        ('금관 · 금빛 갑옷 · 쌍검', foot('liubei_4', kit(guard=GOLD, robe=BLUE, weapon='sword', twin=True)),
         H(bearded(Hd.CROWN, 'goatee'))),
    ],
    'liubei2': [
        ('금빛 관모 · 진영색 도포 · 철 갑옷 · 쌍검', foot('liubei_5', kit(robe=BLUE, guard=STEEL, weapon='sword', twin=True)),
         H(bearded(GOLD_CAP, 'goatee'))),
        ('금빛 관모 · 진영색 도포 · 금빛 갑옷 · 쌍검', foot('liubei_6', kit(robe=BLUE, guard=GOLD, weapon='sword', twin=True)),
         H(bearded(GOLD_CAP, 'goatee'))),
        ('금빛 관모 · 진영색 도포 · 검은 갑옷 · 쌍검', foot('liubei_7', kit(robe=BLUE, guard=DARK, weapon='sword', twin=True)),
         H(bearded(GOLD_CAP, 'goatee'))),
        ('금빛 관모(수염 없음) · 진영색 도포 · 금빛 갑옷 · 쌍검', foot('liubei_8', kit(robe=BLUE, guard=GOLD, weapon='sword', twin=True)),
         H(GOLD_CAP)),
    ],
    'liubei3': [
        ('금빛 관모 · 큰 귀 · 진영색 도포 · 금빛 갑옷 · 쌍검', foot('liubei_9', kit(robe=BLUE, guard=GOLD, weapon='sword', twin=True)),
         H(eared(GOLD_CAP))),
    ],
    'jianyong': [
        ('흰 윤건 · 진영색 옷 · 활', bowman('jianyong_1', kit()), H(WHITE_HOOD)),
        ('맨상투 · 삼베 옷 · 활', bowman('jianyong_2', kit(cloth=HEMP, guard=U.HEMP_DARK)), H(Hd.HAIR)),
        ('검은 관모 · 진영색 도포 · 활', bowman('jianyong_3', kit(robe=BLUE, guard=BLUE)), H(CAP)),
        ('흰 윤건 · 흰 도포 · 활', bowman('jianyong_4', kit(cloth=WHITE, guard=WHITE, robe=WHITE)), H(WHITE_HOOD)),
    ],
    'zhangjiao': [
        ('황건 · 삼베 도포 · 긴 수염 · 지팡이', foot('zhangjiao_1', kit(cloth=HEMP, guard=U.HEMP_DARK, robe=HEMP, weapon='staff')),
         H(bearded(Hd.TOPKNOT, 'long'), Hd.BAND)),
        ('노란 두건 · 흰 도포 · 긴 수염 · 지팡이', foot('zhangjiao_2', kit(cloth=WHITE, guard=GOLD, robe=WHITE, weapon='staff')),
         H(bearded(YELLOW_HOOD, 'long'))),
        ('황건 · 진영색 도포 · 긴 수염 · 깃털부채', foot('zhangjiao_3', kit(robe=BLUE, guard=BLUE, weapon='fan')),
         H(bearded(Hd.TOPKNOT, 'long'), Hd.BAND)),
        ('노란 두건 · 삼베 도포 · 칼', foot('zhangjiao_4', kit(cloth=HEMP, guard=U.HEMP_DARK, robe=HEMP, weapon='sword')),
         H(bearded(YELLOW_HOOD, 'long'))),
    ],
    'huaxiong': [
        ('검은 투구·갑옷 · 언월도 · 흑마', rider('huaxiong_1', kit(cloth=BLACK, guard=DARK), 'black', 'glaive', 'glaive_sweep'),
         H(Hd.ZHANGFEI_HELM if hasattr(Hd, 'ZHANGFEI_HELM') else DARK_IRON)),
        ('검은 투구 · 붉은 옷 · 언월도 · 밤색 말', rider('huaxiong_2', kit(cloth=RED, guard=DARK), 'bay', 'glaive', 'glaive_sweep'),
         H(DARK_IRON)),
        ('검은 투구 · 수염 · 창 · 흑마', rider('huaxiong_3', kit(cloth=BLACK, guard=DARK), 'black', 'spear', 'charge'),
         H(bearded(DARK_IRON, 'bristle'))),
        ('금빛 투구 · 진영색 옷 · 언월도 · 밤색 말', rider('huaxiong_4', kit(guard=GOLD), 'bay', 'glaive', 'glaive_sweep'),
         H(GOLD_IRON, Hd.PLUME)),
    ],
    'lubu': [
        ('꿩깃 금관 · 붉은 갑옷 · 방천화극 · 적토마', rider('lubu_1', kit(cloth=RED, guard=GOLD), 'red', 'halberd', 'charge'),
         H(Hd.crested(GOLD_IRON, Hd.FEATHERS), crest=Hd.FEATHER_ROWS)),
        ('꿩깃 금관 · 자주 갑옷 · 방천화극 · 적토마', rider('lubu_2', kit(cloth=PURPLE, guard=GOLD), 'red', 'halberd', 'charge'),
         H(Hd.crested(GOLD_IRON, Hd.FEATHERS), crest=Hd.FEATHER_ROWS)),
        ('꿩깃 금관 · 흰 갑옷 · 방천화극 · 백마', rider('lubu_3', kit(cloth=WHITE, guard=GOLD), 'white', 'halberd', 'charge'),
         H(Hd.crested(GOLD_IRON, Hd.FEATHERS), crest=Hd.FEATHER_ROWS)),
        ('꿩깃 금관 · 붉은 갑옷 · 방천화극 휘두르기 · 적토마', rider('lubu_4', kit(cloth=RED, guard=GOLD), 'red', 'halberd', 'swing'),
         H(Hd.crested(GOLD_IRON, Hd.FEATHERS), crest=Hd.FEATHER_ROWS)),
    ],
}
TITLES = {'liubei3': '유비', 'liubei2': '유비', 'strategist': '사마(문관)', 'liubei': '유비', 'jianyong': '간옹', 'zhangjiao': '장각', 'huaxiong': '화웅', 'lubu': '여포'}
SELECTED = {'strategist': 2, 'liubei3': 0, 'jianyong': 0, 'zhangjiao': 0, 'huaxiong': 0, 'lubu': 3}
# unit key in the sprite set -> look key (liubei went through three rounds of candidates)
CHOSEN = {'strategist': 'strategist', 'liubei': 'liubei3', 'jianyong': 'jianyong', 'zhangjiao': 'zhangjiao',
          'huaxiong': 'huaxiong', 'lubu': 'lubu'}
NAMES = {'strategist': '사마', 'liubei': '유비', 'jianyong': '간옹', 'zhangjiao': '장각', 'huaxiong': '화웅', 'lubu': '여포'}


def chosen():
    """(key, title, build, head set) of the picked looks, in build.UNITS form."""
    out = []
    for key, look in CHOSEN.items():
        _, build, head = LOOKS[look][SELECTED[look]]
        out.append((key, NAMES[key], build, head))
    return out  # key -> candidate index (user picks)
