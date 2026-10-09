"""ver4 look candidates: four SW looks per new unit or hero; the user picks one.

Every candidate stays defined here as a backup; build.UNITS takes the chosen one
(LOOKS[key][SELECTED[key]]). Kits are registered into units.KITS under '<key>_<n>'.

  python3 tools/unit3d/look_candidates.py   -> output/ver4-look-candidates.png
"""
import attacks as A
import blocks as B
import heads as Hd
import units as U

WHITE, HEMP, RED, GOLD = '#ede1bd', U.HEMP, U.RED, U.GOLD
BLUE, LIGHT, STEEL, BLACK, DARK = U.BLUE, U.LIGHT_BLUE, U.STEEL, U.BLACK, U.DARK_STEEL
PURPLE = '#4b315b'
U.RAMPS.update({PURPLE: '1223'})


def kit(cloth=BLUE, tabard=LIGHT, guard=STEEL, weapon=None, **extra):
    return dict(cloth=cloth, tabard=tabard, guard=guard, shield=extra.pop('shield', False), weapon=weapon, **extra)


# builder -> its block motion builder (blocks.py), so build.py can render the block row
BLOCK_OF = {}


def foot(key, k):
    U.KITS[key] = k
    build = U.foot_unit(key)
    BLOCK_OF[build] = B.foot(key)
    return build


def bowman(key, k):
    U.KITS[key] = k
    build = lambda row, f, style=None: U.archer(row, f, style, kit=key)  # noqa: E731
    BLOCK_OF[build] = B.bowman(key)
    return build


def rider(key, k, coat, weapon, style):
    U.KITS[key] = k
    build = lambda row, f, style_=None: U.cavalry(row, f, style_ or style, kit=key, coat=coat, weapon=weapon)  # noqa: E731
    BLOCK_OF[build] = B.rider(kit=key, coat=coat, weapon=weapon)
    return build


H = Hd.heads
DARK_IRON = {d: Hd.recolor(r, Hd.DARK_IRON) for d, r in Hd.IRON.items()}
GOLD_IRON = {d: Hd.recolor(r, Hd.GOLD_IRON) for d, r in Hd.IRON.items()}
# black scholar's cap: small cap with pins on the topknot (user pick 4 of 4, 2026-10-04). Its ornament is
# the wearer's faction color (user, 2026-10-04): team-key pins and a team-key band around the cap's rim.
CAP = {d: [(r.replace('m', 'Y') if y != 3 else r.replace('2', 'X')) for y, r in enumerate(Hd.gwanmo(rows, 4))]
       for d, rows in Hd.HAIR.items()}
GOLD_CAP = {d: Hd.recolor(Hd.flat_cap(r), {'b': 'Y'}) for d, r in GOLD_IRON.items()}  # top knot in team color
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

# B01 and later (2026-10-04): candidates only until the user picks.
BRICK, MAROON, LEATHER = '#a14a35', '#7a2f22', '#704022'
LOOKS.update({
    'merchant': [  # 장세평 head; the merchant caravan class (non-combatant ally)
        ('말 탄 상인 · 진영색 도포 · 밤색 말', rider('merchant_1', kit(robe=BLUE, guard=BLUE), 'bay', 'none', 'charge'),
         H(bearded(CAP, 'goatee'))),
        ('말 탄 상인 · 짐 실은 말', rider('merchant_2', kit(robe=BLUE, guard=BLUE, packs=LEATHER), 'bay', 'none', 'charge'),
         H(bearded(CAP, 'goatee'))),
        ('도보 상인 · 등짐', foot('merchant_3', kit(robe=BLUE, guard=BLUE, pack=HEMP)), H(bearded(CAP, 'goatee'))),
        ('도보 상인 · 진영색 도포 · 지팡이', foot('merchant_4', kit(robe=BLUE, guard=BLUE, weapon='staff')),
         H(bearded(CAP, 'goatee'))),
    ],
    'chengyuanzhi': [
        ('황건 · 덥수룩한 수염 · 철 어깨 · 칼·방패', foot('chengyuanzhi_1', kit(cloth=HEMP, guard=STEEL, weapon='sword', shield=True)),
         H(bearded(Hd.TOPKNOT, 'bushy'), Hd.BAND)),
        ('노란 두건 · 뻣뻣한 수염 · 가죽 갑옷 · 칼', foot('chengyuanzhi_2', kit(cloth=HEMP, guard=LEATHER, weapon='sword')),
         H(bearded(YELLOW_HOOD, 'bristle'))),
        ('황건 · 검은 옷 · 뻣뻣한 수염 · 칼·방패', foot('chengyuanzhi_3', kit(cloth=BLACK, guard=DARK, weapon='sword', shield=True)),
         H(bearded(Hd.TOPKNOT, 'bristle'), Hd.BAND)),
        ('노란 두건 · 삼베 도포 · 긴 수염 · 지팡이', foot('chengyuanzhi_4', kit(cloth=HEMP, robe=HEMP, guard=U.HEMP_DARK, weapon='staff')),
         H(bearded(YELLOW_HOOD, 'long'))),
    ],
    'dengmao': [
        ('황건 · 붉은 옷 · 칼·방패', foot('dengmao_1', kit(cloth=RED, guard=U.HEMP_DARK, weapon='sword', shield=True)),
         H(Hd.TOPKNOT, Hd.BAND)),
        ('황건 · 삼베 옷 · 뻣뻣한 수염 · 칼', foot('dengmao_2', kit(cloth=HEMP, guard=U.HEMP_DARK, weapon='sword')),
         H(bearded(Hd.TOPKNOT, 'bristle'), Hd.BAND)),
        ('황건 · 삼베 옷 · 창', foot('dengmao_3', kit(cloth=HEMP, guard=U.HEMP_DARK, weapon='spear')),
         H(Hd.TOPKNOT, Hd.BAND)),
        ('노란 두건 · 가죽 갑옷 · 칼·방패', foot('dengmao_4', kit(cloth=HEMP, guard=LEATHER, weapon='sword', shield=True)),
         H(YELLOW_HOOD)),
    ],
})

# Chapter 1 officers (2026-10-05): candidates only until the user picks.
SILVER, PALE = '#c9cdd0', '#dde1e3'
WHITE_PLUME = {'b': '4', 'c': '5', 'd': '6'}
LOOKS.update({
    'zhaoyun': [
        ('은빛 투구 · 흰 전포 · 은빛 갑옷 · 창 · 백마', rider('zhaoyun_1', kit(cloth=WHITE, guard=SILVER, tabard=PALE), 'white', 'spear', 'charge'),
         H(Hd.IRON, WHITE_PLUME)),
        ('은빛 투구 · 진영색 전포 · 창 · 백마', rider('zhaoyun_2', kit(guard=SILVER), 'white', 'spear', 'charge'), H(Hd.IRON, Hd.PLUME)),
        ('흰 두건 · 흰 전포 · 창 · 백마', rider('zhaoyun_3', kit(cloth=WHITE, guard=WHITE, tabard=LIGHT), 'white', 'spear', 'charge'),
         H(WHITE_HOOD)),
        ('은빛 투구 · 흰 전포 · 창 · 백마 · 도약 찌르기', rider('zhaoyun_4', kit(cloth=WHITE, guard=SILVER, tabard=LIGHT), 'white', 'spear', 'leap'),
         H(Hd.IRON, Hd.PLUME)),
    ],
    'wenchou': [
        ('검은 투구 · 붉은 옷 · 뻣뻣한 수염 · 창 · 흑마', rider('wenchou_1', kit(cloth=RED, guard=DARK), 'black', 'spear', 'charge'),
         H(bearded(DARK_IRON, 'bristle'), Hd.PLUME)),
        ('검은 투구 · 검은 갑옷 · 덥수룩한 수염 · 창 · 흑마', rider('wenchou_2', kit(cloth=BLACK, guard=DARK), 'black', 'spear', 'swing'),
         H(bearded(DARK_IRON, 'bushy'), Hd.PLUME)),
        ('철 투구 · 진영색 옷 · 뻣뻣한 수염 · 창 · 밤색 말', rider('wenchou_3', kit(guard=STEEL), 'bay', 'spear', 'charge'),
         H(bearded(Hd.IRON, 'bristle'), Hd.PLUME)),
        ('금빛 투구 · 진영색 옷 · 검은 갑옷 · 창 · 흑마', rider('wenchou_4', kit(guard=DARK), 'black', 'spear', 'downstab'),
         H(bearded(GOLD_IRON, 'bristle'), Hd.PLUME)),
    ],
    'yanliang': [
        ('금빛 투구 · 진영색 옷 · 금빛 갑옷 · 대도 · 밤색 말', rider('yanliang_1', kit(guard=GOLD), 'bay', 'glaive', 'glaive_sweep'),
         H(bearded(GOLD_IRON, 'bristle'), Hd.PLUME)),
        ('철 투구 · 붉은 옷 · 대도 · 흑마', rider('yanliang_2', kit(cloth=RED, guard=STEEL), 'black', 'glaive', 'glaive_sweep'),
         H(bearded(Hd.IRON, 'bushy'), Hd.PLUME)),
        ('검은 투구 · 검은 갑옷 · 덥수룩한 수염 · 창 · 흑마', rider('yanliang_3', kit(cloth=BLACK, guard=DARK), 'black', 'spear', 'charge'),
         H(bearded(DARK_IRON, 'bushy'), Hd.PLUME)),
        ('금빛 투구 · 금빛 갑옷 · 대도 · 백마', rider('yanliang_4', kit(cloth=GOLD, guard=GOLD), 'white', 'glaive', 'glaive_sweep'),
         H(bearded(GOLD_IRON, 'long'), Hd.PLUME)),
    ],
    'zhanghe': [
        ('철 투구 · 진영색 옷 · 창 · 밤색 말', rider('zhanghe_1', kit(guard=STEEL), 'bay', 'spear', 'charge'), H(bearded(Hd.IRON, 'goatee'), Hd.PLUME)),
        ('철 투구 · 검은 갑옷 · 창 · 흑마', rider('zhanghe_2', kit(guard=DARK), 'black', 'spear', 'charge'), H(bearded(Hd.IRON, 'goatee'), Hd.PLUME)),
        ('은빛 투구 · 진영색 옷 · 은빛 갑옷 · 창 · 백마', rider('zhanghe_3', kit(guard=SILVER), 'white', 'spear', 'charge'),
         H(Hd.IRON, Hd.PLUME)),
        ('원뿔 투구 · 진영색 옷 · 창 · 밤색 말', rider('zhanghe_4', kit(guard=STEEL), 'bay', 'spear', 'leap'), H(bearded(Hd.CONE, 'goatee'), Hd.PLUME)),
    ],
    'yuanshao_b': [
        ('금빛 투구 · 흰 옷 · 금빛 갑옷 · 창 · 백마', rider('yuanshao_1', kit(cloth=WHITE, guard=GOLD), 'white', 'spear', 'charge'),
         H(bearded(GOLD_IRON, 'goatee'), Hd.PLUME)),
        ('금빛 투구 · 진영색 옷 · 금빛 갑옷 · 창 · 밤색 말', rider('yuanshao_2', kit(guard=GOLD), 'bay', 'spear', 'charge'),
         H(bearded(GOLD_IRON, 'goatee'), Hd.PLUME)),
        ('금빛 관모 · 흰 도포 · 금빛 어깨 · 칼 (도보)', foot('yuanshao_3', kit(cloth=WHITE, robe=WHITE, guard=GOLD, weapon='sword')),
         H(bearded(GOLD_CAP, 'goatee'))),
        ('금빛 투구 · 흰 옷 · 금빛 갑옷 · 대도 · 백마', rider('yuanshao_4', kit(cloth=WHITE, guard=GOLD), 'white', 'glaive', 'glaive_sweep'),
         H(bearded(GOLD_IRON, 'goatee'), Hd.PLUME)),
    ],
    'tianyu': [
        ('철 투구 · 진영색 옷 · 창 · 밤색 말 (젊은 기병)', rider('tianyu_1', kit(guard=STEEL), 'bay', 'spear', 'charge'), H(Hd.IRON, Hd.PLUME)),
        ('맨상투 · 진영색 옷 · 창 · 밤색 말', rider('tianyu_2', kit(guard=LIGHT), 'bay', 'spear', 'charge'), H(Hd.HAIR)),
        ('진영색 두건 · 창 · 흑마', rider('tianyu_3', kit(guard=STEEL), 'black', 'spear', 'charge'), H(Hd.HOOD, Hd.HOOD_CLOTH)),
        ('흰 두건 · 진영색 옷 · 창 · 백마', rider('tianyu_4', kit(guard=STEEL), 'white', 'spear', 'leap'), H(WHITE_HOOD)),
    ],
    'fangong': [  # martial artist (무도가): bare hands
        ('맨상투 · 삼베 옷 · 붉은 띠 · 맨손', foot('fangong_1', kit(cloth=HEMP, guard=U.HEMP_DARK, tabard=RED)), H(Hd.HAIR)),
        ('진영색 두건 · 진영색 옷 · 맨손', foot('fangong_2', kit(guard=BLUE)), H(Hd.HOOD, Hd.HOOD_CLOTH)),
        ('맨상투 · 덥수룩한 수염 · 검은 옷 · 진영색 띠 · 맨손', foot('fangong_3', kit(cloth=BLACK, guard=DARK)),
         H(bearded(Hd.HAIR, 'bushy'))),
        ('맨상투 · 흰 옷 · 붉은 띠 · 맨손', foot('fangong_4', kit(cloth=WHITE, guard=WHITE, tabard=RED)), H(Hd.HAIR)),
    ],
})

# 2026-10-05 picks and second rounds
GREEN, DARK_GREEN = U.GREEN, U.DARK_GREEN
FIX_BLUE, IVORY = '#193b65', '#ede1b8'  # fixed blue on the knight-v6 blue ramp (not the team keys); hemp's light cream
U.RAMPS.update({FIX_BLUE: '789a'})
GREEN_HOOD = {d: Hd.recolor(r, {'7': 'G', '8': 'H', '9': 'I', 'a': 'J'}) for d, r in Hd.HOOD.items()}
# Gold scholar's cap in the new shape; pins and rim in the wearer's faction color like CAP
GOLD_GWANMO = {d: [(r.replace('m', 'Y').replace('2', 'l').replace('3', 'm') if y < 3 else
                    r.replace('2', 'X') if y == 3 else r) for y, r in enumerate(Hd.gwanmo(rows, 4))]
               for d, rows in Hd.HAIR.items()}
LOOKS.update({
    'yuanshao_b': LOOKS['yuanshao_b'][:2] + [
        ('금빛 관모(새 모양) · 흰 도포 · 금빛 어깨 · 칼 (도보)', foot('yuanshao_3', kit(cloth=WHITE, robe=WHITE, guard=GOLD, weapon='sword')),
         H(bearded(GOLD_GWANMO, 'goatee')))] + LOOKS['yuanshao_b'][3:],
    'tianyu2': [  # Tian Yu later serves Wei: fixed blue and white, on foot
        ('철 투구 · 청색 옷 · 흰 띠 · 칼·방패', foot('tianyu_5', kit(cloth=FIX_BLUE, guard=STEEL, tabard=WHITE, weapon='sword', shield=True)),
         H(Hd.IRON, WHITE_PLUME)),
        ('흰 두건 · 청색 옷 · 흰 띠 · 창', foot('tianyu_6', kit(cloth=FIX_BLUE, guard=WHITE, tabard=WHITE, weapon='spear')), H(WHITE_HOOD)),
        ('맨상투 · 흰 옷 · 청색 도포 · 칼·방패', foot('tianyu_7', kit(cloth=WHITE, robe=FIX_BLUE, guard=FIX_BLUE, tabard=WHITE, weapon='sword', shield=True)),
         H(Hd.HAIR)),
        ('철 투구 · 흰 옷 · 청색 어깨·띠 · 창', foot('tianyu_8', kit(cloth=WHITE, guard=FIX_BLUE, tabard=FIX_BLUE, weapon='spear')),
         H(Hd.IRON, WHITE_PLUME)),
    ],
    'fangong2': [  # green hood, ivory scarf, bare shoulders and arms
        ('녹색 두건 · 아이보리 스카프 · 녹색 옷 · 맨팔', foot('fangong_5', kit(cloth=GREEN, guard=DARK_GREEN, tabard=DARK_GREEN, shoulder=U.SKIN,
                                                                      arms=U.SKIN, scarf=IVORY)), H(GREEN_HOOD)),
        ('녹색 두건 · 아이보리 스카프 · 짙은 녹색 옷 · 맨팔', foot('fangong_6', kit(cloth=DARK_GREEN, guard=GREEN, tabard=GREEN, shoulder=U.SKIN,
                                                                         arms=U.SKIN, scarf=IVORY)), H(GREEN_HOOD)),
        ('녹색 두건 · 긴 아이보리 스카프 · 녹색 옷 · 맨팔', foot('fangong_7', kit(cloth=GREEN, guard=DARK_GREEN, tabard=DARK_GREEN, shoulder=U.SKIN,
                                                                        arms=U.SKIN, scarf=IVORY, scarf_tail=.6)), H(GREEN_HOOD)),
        ('녹색 두건 · 아이보리 스카프 · 녹색 옷 · 붉은 띠 · 맨팔', foot('fangong_8', kit(cloth=GREEN, guard=DARK_GREEN, tabard=RED, shoulder=U.SKIN,
                                                                          arms=U.SKIN, scarf=IVORY)), H(GREEN_HOOD)),
    ],
})
A.SELECTED.update({'tianyu_6': 'thrust', 'tianyu_8': 'thrust'})
# Fan Gong's look (#4) with four bare-hand attacks (attacks.SWORD jab/hook/upper/rush)
FANGONG = dict(cloth=GREEN, guard=DARK_GREEN, tabard=RED, shoulder=U.SKIN, arms=U.SKIN, scarf=IVORY, weapon='fist')
LOOKS['fangong_attack'] = [(f'번궁 · {A.SWORD[st]["name"]}', foot(f'fangong_{st}', kit(**FANGONG)), H(GREEN_HOOD))
                           for st in ('jab', 'hook', 'upper', 'rush')]
A.SELECTED.update({f'fangong_{st}': st for st in ('jab', 'hook', 'upper', 'rush')})

# Chapter 1 minor officers (2026-10-05): class sprites with their own head, beard and colors.
FAN, SWORD = dict(weapon='fan'), dict(weapon='sword', shield=True)
LOOKS.update({
    'chunyuqiong': [
        ('철 투구 · 뻣뻣한 수염 · 창 · 밤색 말', rider('chunyuqiong_1', kit(guard=STEEL), 'bay', 'spear', 'charge'), H(bearded(Hd.IRON, 'bristle'), Hd.PLUME)),
        ('철 투구 · 붉은 옷 · 덥수룩한 수염(술꾼) · 창 · 밤색 말', rider('chunyuqiong_2', kit(cloth=RED, guard=STEEL), 'bay', 'spear', 'charge'),
         H(bearded(Hd.IRON, 'bushy'), Hd.PLUME)),
        ('금빛 투구 · 진영색 옷 · 창 · 흑마', rider('chunyuqiong_3', kit(guard=GOLD), 'black', 'spear', 'charge'), H(bearded(GOLD_IRON, 'bristle'), Hd.PLUME)),
        ('검은 투구 · 진영색 옷 · 대도 · 밤색 말', rider('chunyuqiong_4', kit(guard=DARK), 'bay', 'glaive', 'glaive_sweep'),
         H(bearded(DARK_IRON, 'bristle'), Hd.PLUME)),
    ],
    'shenpei': [
        ('검은 관모 · 염소수염 · 진영색 도포 · 부채', foot('shenpei_1', kit(robe=BLUE, guard=BLUE, **FAN)), H(bearded(CAP, 'goatee'))),
        ('검은 관모 · 긴 수염 · 흰 도포 · 부채', foot('shenpei_2', kit(cloth=WHITE, robe=WHITE, guard=BLUE, **FAN)), H(bearded(CAP, 'long'))),
        ('흰 윤건 · 진영색 도포 · 부채', foot('shenpei_3', kit(robe=BLUE, guard=BLUE, **FAN)), H(WHITE_HOOD)),
        ('금빛 관모 · 진영색 도포 · 부채', foot('shenpei_4', kit(robe=BLUE, guard=GOLD, **FAN)), H(bearded(GOLD_GWANMO, 'goatee'))),
    ],
    'tianfeng': [
        ('검은 관모 · 긴 수염 · 진영색 도포 · 부채', foot('tianfeng_1', kit(robe=BLUE, guard=BLUE, **FAN)), H(bearded(CAP, 'long'))),
        ('흰 윤건 · 긴 수염 · 흰 도포 · 부채', foot('tianfeng_2', kit(cloth=WHITE, robe=WHITE, guard=WHITE, **FAN)), H(bearded(WHITE_HOOD, 'long'))),
        ('검은 관모 · 흰 도포 · 진영색 띠 · 부채', foot('tianfeng_3', kit(cloth=WHITE, robe=WHITE, guard=BLUE, **FAN)), H(CAP)),
        ('맨상투 · 긴 수염 · 진영색 도포 · 부채', foot('tianfeng_4', kit(robe=BLUE, guard=BLUE, **FAN)), H(bearded(Hd.HAIR, 'long'))),
    ],
    'quyi': [
        ('철 투구 · 진영색 옷 · 활 (강노대장)', bowman('quyi_1', kit()), H(bearded(Hd.IRON, 'bristle'), Hd.PLUME)),
        ('검은 투구 · 검은 갑옷 · 활', bowman('quyi_2', kit(cloth=BLACK, guard=DARK)), H(bearded(DARK_IRON, 'bristle'), Hd.PLUME)),
        ('철 투구 · 진영색 옷 · 칼·방패', foot('quyi_3', kit(**SWORD)), H(bearded(Hd.IRON, 'bristle'), Hd.PLUME)),
        ('진영색 두건 · 활', bowman('quyi_4', kit(guard=DARK)), H(bearded(Hd.HOOD, 'bristle'), Hd.HOOD_CLOTH)),
    ],
    'yangang': [
        ('철 투구 · 진영색 옷 · 창 · 백마', rider('yangang_1', kit(guard=STEEL), 'white', 'spear', 'charge'), H(bearded(Hd.IRON, 'bristle'), Hd.PLUME)),
        ('은빛 투구 · 흰 옷 · 창 · 백마', rider('yangang_2', kit(cloth=WHITE, guard=SILVER), 'white', 'spear', 'charge'), H(Hd.IRON, WHITE_PLUME)),
        ('철 투구 · 진영색 옷 · 창 · 밤색 말', rider('yangang_3', kit(guard=STEEL), 'bay', 'spear', 'charge'), H(bearded(Hd.IRON, 'goatee'), Hd.PLUME)),
        ('철 투구 · 덥수룩한 수염 · 창 · 백마', rider('yangang_4', kit(guard=DARK), 'white', 'spear', 'charge'), H(bearded(Hd.IRON, 'bushy'), Hd.PLUME)),
    ],
    'gaolan': [
        ('철 투구 · 진영색 옷 · 칼·방패', foot('gaolan_1', kit(**SWORD)), H(bearded(Hd.IRON, 'goatee'), Hd.PLUME)),
        ('검은 투구 · 검은 갑옷 · 칼·방패', foot('gaolan_2', kit(cloth=BLACK, guard=DARK, **SWORD)), H(bearded(DARK_IRON, 'goatee'), Hd.PLUME)),
        ('철 투구 · 진영색 옷 · 창', foot('gaolan_3', kit(weapon='spear')), H(bearded(Hd.IRON, 'bristle'), Hd.PLUME)),
        ('금빛 투구 · 진영색 옷 · 칼·방패', foot('gaolan_4', kit(guard=GOLD, **SWORD)), H(bearded(GOLD_IRON, 'goatee'), Hd.PLUME)),
    ],
    'hanying': [
        ('진영색 두건 · 칼·방패', foot('hanying_1', kit(**SWORD)), H(Hd.HOOD, Hd.HOOD_CLOTH)),
        ('맨상투 · 삼베 옷 · 칼·방패', foot('hanying_2', kit(cloth=HEMP, guard=U.HEMP_DARK, **SWORD)), H(Hd.HAIR)),
        ('철 투구 · 진영색 옷 · 칼·방패', foot('hanying_3', kit(**SWORD)), H(bearded(Hd.IRON, 'goatee'), Hd.PLUME)),
        ('흰 두건 · 진영색 옷 · 창', foot('hanying_4', kit(weapon='spear')), H(WHITE_HOOD)),
    ],
    'guoshi': [
        ('맨상투 · 뻣뻣한 수염 · 삼베 옷 · 칼·방패', foot('guoshi_1', kit(cloth=HEMP, guard=U.HEMP_DARK, **SWORD)), H(bearded(Hd.HAIR, 'bristle'))),
        ('검은 두건 · 검은 옷 · 칼', foot('guoshi_2', kit(cloth=BLACK, guard=DARK, weapon='sword')),
         H(Hd.HOOD, {'7': '0', '8': '1', '9': '2', 'a': '3'})),
        ('진영색 두건 · 삼베 옷 · 칼', foot('guoshi_3', kit(cloth=HEMP, guard=U.HEMP_DARK, weapon='sword')), H(bearded(Hd.HOOD, 'bristle'), Hd.HOOD_CLOTH)),
        ('맨상투 · 덥수룩한 수염 · 가죽 갑옷 · 칼·방패', foot('guoshi_4', kit(cloth=HEMP, guard=LEATHER, **SWORD)), H(bearded(Hd.HAIR, 'bushy'))),
    ],
    'gengwu': [
        ('검은 관모 · 진영색 옷 · 칼·방패', foot('gengwu_1', kit(**SWORD)), H(CAP)),
        ('철 투구 · 진영색 옷 · 칼', foot('gengwu_2', kit(weapon='sword')), H(bearded(Hd.IRON, 'goatee'), Hd.PLUME)),
        ('검은 관모 · 염소수염 · 흰 도포 · 칼', foot('gengwu_3', kit(cloth=WHITE, robe=WHITE, guard=BLUE, weapon='sword')), H(bearded(CAP, 'goatee'))),
        ('철 투구 · 흰 깃털 · 진영색 옷 · 칼·방패', foot('gengwu_4', kit(**SWORD)), H(Hd.IRON, WHITE_PLUME)),
    ],
    'guanchun': [
        ('검은 관모 · 진영색 도포 · 부채', foot('guanchun_1', kit(robe=BLUE, guard=BLUE, **FAN)), H(CAP)),
        ('검은 관모 · 염소수염 · 흰 도포 · 부채', foot('guanchun_2', kit(cloth=WHITE, robe=WHITE, guard=BLUE, **FAN)), H(bearded(CAP, 'goatee'))),
        ('흰 윤건 · 진영색 도포 · 부채', foot('guanchun_3', kit(robe=BLUE, guard=BLUE, **FAN)), H(WHITE_HOOD)),
        ('검은 관모 · 긴 수염 · 진영색 도포 · 부채', foot('guanchun_4', kit(robe=BLUE, guard=BLUE, **FAN)), H(bearded(CAP, 'long'))),
        ('검은 관모 · 염소수염 · 흰 도포 · 활 (2번 차림의 궁병, 사용자 지정)', bowman('guanchun_5', kit(cloth=WHITE, robe=WHITE, guard=BLUE)),
         H(bearded(CAP, 'goatee'))),
    ],
})
A.SELECTED.update({'gaolan_3': 'thrust', 'hanying_4': 'thrust'})
TITLES = {'liubei3': '유비', 'liubei2': '유비', 'strategist': '사마(문관)', 'liubei': '유비', 'jianyong': '간옹', 'zhangjiao': '장각', 'huaxiong': '화웅', 'lubu': '여포',
          'merchant': '상단(장세평)', 'chengyuanzhi': '정원지', 'dengmao': '등무',
          'zhaoyun': '조운', 'wenchou': '문추', 'yanliang': '안량', 'zhanghe': '장합', 'yuanshao_b': '원소', 'tianyu': '전예', 'fangong': '번궁', 'tianyu2': '전예', 'fangong2': '번궁', 'fangong_attack': '번궁',
          'chunyuqiong': '순우경', 'shenpei': '심배', 'tianfeng': '전풍', 'quyi': '국의', 'yangang': '엄강', 'gaolan': '고람',
          'hanying': '한영', 'guoshi': '곽적', 'gengwu': '경무', 'guanchun': '관순'}
SELECTED = {'strategist': 2, 'liubei3': 0, 'jianyong': 0, 'zhangjiao': 0, 'huaxiong': 0, 'lubu': 3,
            'merchant': 2, 'chengyuanzhi': 3, 'dengmao': 3,
            'zhaoyun': 3, 'wenchou': 1, 'yanliang': 1, 'zhanghe': 1, 'yuanshao_b': 2, 'tianyu2': 0, 'fangong_attack': 0,
            'chunyuqiong': 3, 'yangang': 3, 'quyi': 0, 'gaolan': 1, 'shenpei': 3, 'tianfeng': 0, 'hanying': 0, 'guoshi': 0, 'gengwu': 0,
            'guanchun': 4}  # 2026-10-04/05 user picks; merchant walks with a bigger pack, both turbans yellow
# unit key in the sprite set -> look key (liubei went through three rounds of candidates)
CHOSEN = {'strategist': 'strategist', 'liubei': 'liubei3', 'jianyong': 'jianyong', 'zhangjiao': 'zhangjiao',
          'huaxiong': 'huaxiong', 'lubu': 'lubu', 'zhangshiping': 'merchant', 'chengyuanzhi': 'chengyuanzhi',
          'dengmao': 'dengmao', 'sushuang': 'merchant',
          'zhaoyun': 'zhaoyun', 'wenchou': 'wenchou', 'yanliang': 'yanliang', 'zhanghe': 'zhanghe', 'yuanshao': 'yuanshao_b', 'tianyu': 'tianyu2', 'fangong': 'fangong_attack',
          'chunyuqiong': 'chunyuqiong', 'yangang': 'yangang', 'quyi': 'quyi', 'gaolan': 'gaolan', 'shenpei': 'shenpei', 'tianfeng': 'tianfeng', 'hanying': 'hanying', 'guoshi': 'guoshi', 'gengwu': 'gengwu', 'guanchun': 'guanchun'}
# The two merchants share the walking merchant body; heads match their scene sprites.
HEADS = {'zhangshiping': H(CAP), 'sushuang': H(Hd.HAIR)}
NAMES = {'strategist': '사마', 'liubei': '유비', 'jianyong': '간옹', 'zhangjiao': '장각', 'huaxiong': '화웅', 'lubu': '여포',
         'zhangshiping': '장세평', 'chengyuanzhi': '정원지', 'dengmao': '등무', 'sushuang': '소쌍',
         'zhaoyun': '조운', 'wenchou': '문추', 'yanliang': '안량', 'zhanghe': '장합', 'yuanshao': '원소', 'tianyu': '전예', 'fangong': '번궁',
         'chunyuqiong': '순우경', 'yangang': '엄강', 'quyi': '국의', 'gaolan': '고람', 'shenpei': '심배', 'tianfeng': '전풍', 'hanying': '한영', 'guoshi': '곽적', 'gengwu': '경무', 'guanchun': '관순'}


def chosen():
    """(key, title, build, head set) of the picked looks, in build.UNITS form."""
    out = []
    for key, look in CHOSEN.items():
        _, build, head = LOOKS[look][SELECTED[look]]
        out.append((key, NAMES[key], build, HEADS.get(key, head)))
    return out  # key -> candidate index (user picks)
