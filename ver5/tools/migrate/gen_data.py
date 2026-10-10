#!/usr/bin/env python3
"""Throwaway generator: ver4 campaign.json + docs/trait-design.md -> ver5/assets/data/*.json."""
import json, os, sys
ROOT = os.path.abspath(os.path.join(os.path.dirname(__file__), "..", ".."))
OLD = os.path.join(ROOT, "..", "ver4", "assets", "content", "campaign.json")
FAC = os.path.join(ROOT, "..", "tools", "spritetool", "assets", "ver2-units", "factions.json")
OUT = os.path.join(ROOT, "assets", "data")
old = json.load(open(OLD))
def dump(name, obj):
    p = os.path.join(OUT, name)
    os.makedirs(os.path.dirname(p), exist_ok=True)
    with open(p, "w") as f:
        json.dump(obj, f, ensure_ascii=False, indent=1); f.write("\n")

# ---------------- game.json ----------------
rules = {
 "DamageCurve": [[.5,.8],[1,1],[2,1.2]], "HitCurve": [[0.3333,30],[.5,60],[1,80],[2,100]],
 "StatNumerator": 400, "StatDenominator": 140, "StatLevelBase": 10,
 "BasicAttack": {"Name":"공격","Kind":"physical","Mode":"any","Shape":"single","Hit":90,"Min":1,"Coeff":1},
 "HitFloor":30,"HitCeil":100,"SpellFloor":5,"SpellCeil":100,"MagicPower":2,"MagicGuard":.5,
 "Crit":{"Base":10,"Per":30,"Max":70},"CritMultiplier":1.5,"Double":{"Base":10,"Per":30,"Max":70},
 "CounterShare":.7,"PoisonPercent":5,"BurnPercent":4,"BuffPercent":20,"BuffTurns":2,
 "ExpBase":100,"ExpGrowth":1.035,"ExpAction":24,"ExpSpell":2,"ExpDefeat":2,"ExpLevelStep":.12,
 "ExpMinFactor":.25,"ExpMaxFactor":2,"ExpSupportPercent":16,"PointsPerLevel":100,"MaxLevel":99,
 "DeputyShare":85,"DeputyWeight":2,"DeputyDivisor":300,"DeputyMinTier":1,"PromoteLevels":[15,30],
 "EnemyStock":{"small_food":1},"EdgeTile":"~"}
tiles = {
 ".":{"Name":"평지"}, "d":{"Name":"황무지"}, "~":{"Name":"물","Water":True}, "f":{"Name":"숲","Ambush":True},
 "s":{"Name":"산"}, "v":{"Name":"마을","RestHP":8,"RestMP":2}, "c":{"Name":"성벽"},
 "i":{"Name":"성내","Castle":True,"RestMP":2}, "b":{"Name":"다리"}, "g":{"Name":"성문"},
 "k":{"Name":"주둔지","RestHP":8,"RestMP":2}}
fnames = {"wei":"위","shu":"촉","wu":"오","turban":"황건적","dong":"동탁·여포","gongsun":"공손찬","yuan":"원소·원술","barbarian":"이민족","merchant":"상단"}
factions = {}
for f in json.load(open(FAC))["factions"]:
    r = f["ramp"]; factions[f["id"]] = {"Name": f["name"], "Colors": [r[3], r[2], r[0]]}
campaign = {"Start":"oath","Roster":["liubei","guanyu","zhangfei"],"Deployment":["liubei","guanyu","zhangfei"],
            "Inventory":{"small_food":18,"small_scroll":8}}
dump("game.json", {"Title":"삼국지 SRPG","Version":"ver5-1","Rules":rules,"Campaign":campaign,"Factions":factions,"Tiles":tiles})

# ---------------- terrain.json ----------------
gmap = {"기병":"cavalry","보병":"infantry","궁병":"archer","문관":"scholar","도적":"bandit"}
dump("terrain.json", {"Terrain": {gmap[k]: v for k, v in old["Terrain"].items()}})

# ---------------- classes.json ----------------
def cls(name, desc, terrain, attack, weapon, armor, move, rng, minr, bonus, mpb, mpc, rb, rc, tier, sprite,
        grants=(), skills=(), pool=(), tags=(), promotes=()):
    d = {"Name":name,"Desc":desc,"Terrain":terrain,"Attack":attack,"Weapon":weapon,"Armor":armor,"Move":move,"Range":rng,
         "Bonus":bonus,"HPFactor":3,"MPBase":mpb,"MPCoeff":mpc,"RecoveryBase":rb,"RecoveryCoeff":rc,"Tier":tier,"Sprite":sprite}
    if minr: d["MinRange"] = minr
    for k, v in (("Grants",grants),("Skills",skills),("Pool",pool),("Tags",tags),("Promotes",promotes)):
        if v: d[k] = list(v)
    return d
H = (14, 8, 4, 9, 8, 14)
classes = {
 "light_infantry": cls("경보병","기본 보병. 검과 갑옷으로 전선을 지킨다.","infantry","melee","sword","armor",4,1,0,[12,10,5,7,8,18],14,.6,2,.08,0,"infantry",pool=["counter_art","composure","rough_terrain"],promotes=["heavy_infantry"]),
 "heavy_infantry": cls("중보병","두꺼운 갑옷의 보병.","infantry","melee","sword","armor",4,1,0,[15,14,6,8,10,23],20,.6,2,.08,1,"infantry",pool=["ironwall","missile_immunity","sweep_art"],promotes=["guards"]),
 "guards": cls("근위대","주군을 지키는 정예 보병.","infantry","melee","sword","armor",4,1,0,[18,19,8,10,13,29],26,.6,2,.08,2,"infantry",pool=["castle_defense","diamond","vitality"]),
 "light_cavalry": cls("경기병","기본 기병. 빠르게 달려 적진을 흔든다.","cavalry","melee","spear","armor",5,1,0,[14,8,4,9,8,14],12,.6,2,.08,0,"cavalry",pool=["dash","white_horse","charge"],tags=["mounted","light"],promotes=["heavy_cavalry"]),
 "heavy_cavalry": cls("중기병","갑옷을 두른 기병.","cavalry","melee","spear","armor",5,1,0,[18,10,6,10,10,18],18,.6,2,.08,1,"cavalry",pool=["gale","sweep_art","ironwall"],tags=["mounted"],promotes=["royal_guard"]),
 "royal_guard": cls("친위대","군주 직속의 정예 기병.","cavalry","melee","spear","armor",6,1,0,[22,15,7,12,12,22],24,.6,2,.08,2,"cavalry",pool=["deadly","siegecraft","diamond"],tags=["mounted"]),
 "archer": cls("궁병","활로 멀리서 적을 쏜다.","archer","ranged","bow","armor",4,3,2,[11,6,5,10,9,14],14,.6,2,.08,0,"archer",pool=["venom","far_sight","rough_terrain"],promotes=["crossbow"]),
 "crossbow": cls("노병","강한 쇠뇌를 쏘는 궁병.","archer","ranged","bow","armor",4,4,2,[15,8,6,12,11,18],20,.6,2,.08,1,"archer",pool=["pierce","return_fire","divine_archer"],promotes=["repeating_crossbow"]),
 "repeating_crossbow": cls("연노병","연발 쇠뇌를 쏘는 정예 궁병.","archer","ranged","bow","armor",4,4,2,[22,10,7,15,14,20],26,.6,2,.08,2,"archer",pool=["volley","cross_strike","drain"]),
 "scholar": cls("사마","화계·수계·구휼을 쓰는 문관.","scholar","melee","fan","robe",4,1,0,[6,4,12,8,9,11],28,.65,5,.13,0,"strategist",grants=["fire","water","relief"],pool=["fire_boost","water_boost","focus"],promotes=["staff_officer","sorcerer"]),
 "staff_officer": cls("참모","공격 책략과 방해 책략에 능한 참모.","scholar","melee","fan","robe",4,1,0,[7,5,18,10,12,14],34,.65,5,.13,1,"strategist",grants=["fire","water","earth","hinder","stratagem"],pool=["fire_spread","water_spread","earth_boost","feint","fire_god"],promotes=["strategist"]),
 "strategist": cls("군사","모든 공격·방해 책략을 쓰는 군사.","scholar","melee","fan","robe",4,1,0,[8,7,25,12,16,17],40,.65,5,.13,2,"strategist",grants=["fire","water","earth","hinder","stratagem"],pool=["earth_spread","hinder_boost","stratagem_boost","chain","half_cost"]),
 "sorcerer": cls("방술사","구휼과 독려로 아군을 돕는 술사.","scholar","melee","fan","robe",4,1,0,[6,5,16,9,11,13],34,.65,5,.13,1,"strategist",grants=["relief","exhort"],pool=["relief_boost","relief_spread","medicine","cunning"],promotes=["mystic"]),
 "mystic": cls("선술사","상급 회복과 버프를 쓰는 선술사.","scholar","melee","fan","robe",4,1,0,[7,6,22,11,14,16],40,.65,5,.13,2,"strategist",grants=["relief","exhort"],pool=["exhort_boost","eight_formation","lasting","deep_scheme"]),
 "bandit": cls("적병","재물을 노리는 무뢰배.","bandit","melee","sword","robe",4,1,0,[10,5,4,10,15,24],14,.6,2,.09,0,"infantry",skills=["steal"],pool=["venom","rough_terrain","ambush_crit"],promotes=["ruffian"]),
 "ruffian": cls("흉적","거칠고 날랜 도적.","bandit","melee","sword","robe",5,1,0,[13,7,5,13,20,29],20,.6,2,.09,1,"infantry",skills=["steal"],pool=["drain","dash","fortune"],promotes=["outlaw"]),
 "outlaw": cls("의적","의기를 아는 도적 두목.","bandit","melee","sword","robe",5,1,0,[16,10,7,16,25,33],26,.6,2,.09,2,"infantry",skills=["steal"],pool=["assist","free_item","charge"]),
}
dump("classes.json", {"Classes": classes})

# ---------------- traits.json ----------------
T = {}
def t(id, name, cost, desc, fx=None, req=()):
    d = {"Name":name,"Desc":desc,"Cost":cost}
    if req: d["Requires"] = list(req)
    if fx: d["Effects"] = fx
    T[id] = d
# physical
t("valor","용장",900,"자신보다 무력이 낮은 대상을 공격하면 반드시 치명타.",{"crit_vs_weaker":1})
t("relentless","연전",900,"2회 공격 확률이 100%.",{"double_always":1})
t("warlord","신장",1500,"자신보다 무력이 낮은 대상을 공격하면 반드시 치명타, 2회 공격 확률 100%.",{"crit_vs_weaker":1,"double_always":1},["valor","relentless"])
t("battle_fury","투신",1400,"자신보다 무력이 낮은 적에게 공격받고 그 적이 공격범위 안이면 반드시 반격(2회).",{"counter_force":1})
t("assist","협동",700,"자기 차례에 공격범위 안의 적이 아군의 물리 공격을 받으면 같은 대상을 50% 피해로 추가 공격.",{"assist":.5})
t("ambush_crit","기습",400,"숲에 있으면 물리 공격이 반드시 치명타.",{"crit_ambush":1})
t("volley","난사",1200,"난사 병법 사용 가능. 공격범위 안의 모든 적을 쏜다(원거리 전용).")
t("cross_strike","사방격",600,"물리 공격이 십자 범위가 된다. 주변 칸의 적은 75% 피해.",{"splash_cross":.75})
t("square_strike","팔방격",1000,"물리 공격이 3×3 범위가 된다. 주변 칸의 적은 75% 피해.",{"splash_square":.75},["cross_strike"])
t("siegecraft","공성",700,"대상의 지형 방어 보정을 무시한다.",{"ignore_terrain":1})
t("resolve_strike","회심",0,"일반 공격과 공격 책략이 반드시 치명타. 옥쇄 전용.",{"crit_always":1,"spell_crit":1})
t("peerless","국사무쌍",1500,"병법을 써도 행동을 소모하지 않는다.",{"free_skill":1})
t("free_item","부호",900,"소비 아이템을 써도 행동을 소모하지 않는다.",{"free_item":1})
t("half_cost","백출",800,"병법치 소모가 절반.",{"cost_half":1})
t("deadly","필살",1500,"물리 공격이 반드시 치명타.",{"crit_always":1})
t("pierce","찔러공격",600,"대상 뒤의 적도 75% 피해를 받는다.",{"pierce":.75})
t("venom","독공",300,"물리 공격이 명중하면 독을 건다.",{"poison_on_hit":1})
t("drain","심공",500,"준 피해의 25%만큼 병력을 회복한다.",{"life_steal":.25})
t("composure","침착",300,"자기 차례가 돌아오면 모든 상태이상이 풀린다.",{"cleanse":1})
t("intimidate","위압",0,"물리 공격이 빗나가도 피해의 절반이 들어간다.",{"miss_half":.5})
t("counter_art","반격술",300,"반격 병법 사용 가능.")
t("sweep_art","무쌍술",800,"무쌍 병법 사용 가능. 공격범위 안의 모든 적을 벤다.")
# defense
t("ironwall","철벽",500,"한 번의 피격에서 최대 병력 10%를 넘는 피해는 절반만 받는다.",{"bulwark":.1})
t("diamond","금강",1000,"치명타로 판정된 공격을 막아낸다.",{"crit_immune":1})
t("missile_immunity","간접공격면역",700,"적의 원거리 공격이 명중하지 않는다.",{"ranged_immune":1})
t("return_fire","응사",700,"적의 물리 공격이 빗나가면 70% 피해로 되받아친다.",{"riposte":.7})
t("spell_reflect","반계",1200,"공격 책략을 받으면 같은 책략을 시전자에게 되돌린다.",{"reflect":1})
t("castle_defense","수성",700,"성에 있으면 매 턴 방어력 버프를 건다.",{"fortify":20})
# movement
t("dash","질주",300,"이동력 +1.",{"move":1})
t("gale","질풍",700,"이동력 +2(질주와 합쳐 +3).",{"move":2},["dash"])
t("rough_terrain","험지이동",400,"지형에 따른 이동력 저하가 없다.",{"ignore_move_cost":1})
t("charge","돌격",400,"적의 견제(ZOC)에 막히지 않고 이동한다.",{"ignore_zoc":1})
t("swiftness","신속",500,"매 턴 자신에게 이동력 +2 버프를 건다.",{"haste":2})
t("far_sight","원시",400,"매 턴 자신에게 사거리 +1 버프를 건다.",{"far_sight":1})
t("white_horse","백마",500,"기병이면 사거리 1~2의 원거리 공격이 가능. 경기병은 이동력 +1.",{"mounted_range":2,"light_move":1})
# spell aids
t("cunning","귀모",600,"책략 사거리 +3.",{"spell_range":3})
t("feint","허실",800,"자신보다 지력이 낮은 부대에게 모든 책략이 반드시 성공.",{"spell_sure":1})
t("deep_scheme","심모",1200,"성공한 책략이 반드시 치명타.",{"spell_crit":1})
t("chain","연환",1200,"공격 책략이 인접한 무작위 한 명에게 한 번 더 적용된다.",{"chain":1})
t("eight_formation","팔진도",1000,"팔진도 병법 사용 가능. 아군 한 명의 단일 책략이 3×3 범위가 된다.")
t("fire_god","화신",1500,"화염계 책략이 반드시 성공하고 반드시 치명타.",{"sure:fire":1,"crit:fire":1},["fire"])
# status defense
t("insight","간파",1000,"자신보다 지력이 낮은 시전자의 책략이 듣지 않는다.",{"dispel":1})
t("discipline","군율",500,"해로운 상태이상 책략에 걸리지 않는다.",{"status_immune":1})
# support
t("medicine","의술",500,"모든 병력 회복량 +100%.",{"heal_bonus":1})
t("benevolence","인덕",700,"자기 차례 시작에 인접 아군이 최대 병력의 10%를 회복한다.",{"aura_heal":.1})
t("employment","용인술",1000,"부관 능력치를 항상 110%로 반영한다.",{"deputy_rate":1.1},["benevolence"])
t("supply","보급",500,"쓴 아이템이 인접 아군 한 명에게도 적용된다.",{"item_share":1})
t("focus","집중",400,"병법치 회복량이 2배.",{"mana_regen":1})
t("blood_pay","고육",1000,"병법치가 모자라면 모자란 양의 10배만큼 병력으로 대신 낸다.",{"blood_cost":10})
t("vitality","양생",500,"자기 차례 시작에 최대 병력의 10%를 회복한다.",{"regen":.1})
t("lasting","장구",600,"자신이 건 2턴 이상 버프의 지속이 1턴 늘어난다.",{"buff_extend":1})
t("fortune","호운",600,"전투 중 처음 쓰러질 때 병력 1로 버틴다.",{"last_stand":1})
t("divine_archer","신궁",1000,"원거리 공격의 최소 사거리가 없다.",{"no_min_range":1})
# spell families (roots given only)
for id, name, desc in [("fire","화계","화염계 책략을 쓴다."),("water","수계","수계 책략을 쓴다."),("earth","지계","토지계 책략을 쓴다."),
                       ("relief","구휼","회복계 책략을 쓴다."),("exhort","독려","버프계 책략을 쓴다."),
                       ("hinder","견제","디버프계 책략을 쓴다."),("stratagem","계략","방해계 책략을 쓴다.")]:
    t(id, name, 0, desc+" 병종·장비가 주는 특성.")
for fam, nm, c1, c2 in [("fire","화계",500,700),("water","수계",500,700),("earth","지계",500,700),("relief","구휼",500,700)]:
    t(fam+"_boost", nm+"강화", c1, nm+" 책략의 강한 갈래를 연다.", None, [fam])
    t(fam+"_spread", nm+"확산", c2, nm+" 책략의 범위 갈래를 연다. 강화와 함께 있으면 최상위 책략이 열린다.", None, [fam])
for fam, nm in [("exhort","독려"),("hinder","견제"),("stratagem","계략")]:
    t(fam+"_boost", nm+"강화", 700, nm+" 계통의 상위 책략을 연다.", None, [fam])
dump("traits.json", {"Traits": T})

# ---------------- skills.json ----------------
S = {}
def sk(id, name, desc, kind, mode, shape, effect="", elem="", cond="", cost=0, hit=100, mn=0, mx=0, dur=0, length=0, coeff=0, req=(), item=""):
    d = {"Name":name,"Desc":desc,"Kind":kind,"Mode":mode,"Shape":shape,"Cost":cost,"Hit":hit}
    for k, v in (("Effect",effect),("Element",elem),("Condition",cond),("Item",item)):
        if v: d[k] = v
    for k, v in (("Min",mn),("Max",mx),("Duration",dur),("Length",length),("Coeff",coeff)):
        if v: d[k] = v
    if req: d["Requires"] = list(req)
    S[id] = d
sk("sweep","무쌍","공격범위 안의 모든 적을 벤다.","physical","melee","all",cost=54,hit=90,mn=1,coeff=1,req=["sweep_art"])
sk("barrage","난사","공격범위 안의 모든 적을 쏜다.","physical","ranged","all",cost=40,hit=90,mn=1,coeff=.5,req=["volley"])
sk("counter","반격","다음 적 차례까지 반격 태세를 취한다.","buff","self","single","counter",cost=18,dur=2,req=["counter_art"])
sk("steal","탈취","적이 가진 아이템을 빼앗는다.","physical","melee","single","steal",cost=24,hit=90,mn=1,coeff=.7,item="small_food")
sk("eightfold","팔진도","아군 한 명의 단일 책략을 3×3 범위로 넓힌다.","buff","any","single","eightfold",cost=40,mx=3,dur=2,req=["eight_formation"])
fams = [("fire","화","fire","", [("scorch","초열","화염으로 적을 태운다."),("blaze","업화","초열보다 피해 1.5배."),("fire_array","화진","십자 범위 화염."),("inferno","폭염","일자 6칸 화염."),("fire_dragon","화룡","십자 범위 화염.")]),
        ("water","수","water","water", [("torrent","탁류","탁류로 적을 휩쓴다."),("rapids","격류","탁류보다 피해 1.5배."),("water_array","수진","십자 범위 물."),("tsunami","해일","멀리까지 닿는 거센 파도."),("water_dragon","수룡","십자 범위 물.")]),
        ("earth","지","earth","", [("rockfall","낙석","바위를 굴려 적을 친다."),("landslide","산사태","낙석보다 피해 1.5배."),("earthquake","지진","십자 범위 지진."),("boulder","거암","낙석보다 피해 2배."),("earth_dragon","지룡","십자 범위 대지.")])]
for root, short, elem, cond, names in fams:
    b, s_ = root+"_boost", root+"_spread"
    (i1,n1,d1),(i2,n2,d2),(i3,n3,d3),(i4,n4,d4),(i5,n5,d5) = names
    big = {"fire":dict(c=1.2,mx=2,ln=6), "water":dict(c=1.5,mx=7), "earth":dict(c=2,mx=2)}[root]
    sk(i1,n1,d1,"magic","any","single",elem=elem,cond=cond,cost=60,hit=110,mn=1,mx=4,coeff=1,req=[root])
    sk(i2,n2,d2,"magic","any","single",elem=elem,cond=cond,cost=80,hit=110,mn=1,mx=3,coeff=1.5,req=[root,b])
    sk(i3,n3,d3,"magic","any","cross",elem=elem,cond=cond,cost=80,hit=110,mn=1,mx=4,coeff=.7,req=[root,s_])
    if root == "fire":
        sk(i4,n4,d4,"magic","any","line",elem=elem,cond=cond,cost=100,hit=110,mn=1,mx=2,coeff=1.2,length=6,req=[root,b,s_])
    else:
        sk(i4,n4,d4,"magic","any","single",elem=elem,cond=cond,cost=100,hit=110,mn=1,mx=big["mx"],coeff=big["c"],req=[root,b,s_])
    sk(i5,n5,d5,"magic","any","cross",elem=elem,cond=cond,cost=100,hit=110,mn=1,mx=3,coeff=1,req=[root,b,s_])
R, RB, RS = ["relief"], ["relief","relief_boost"], ["relief","relief_spread"]
RA = ["relief","relief_boost","relief_spread"]
sk("minor_supply","소보급","아군 한 명의 병력을 회복한다.","heal","any","single","heal",cost=24,hit=95,mx=3,coeff=3,req=R)
sk("petition","헌책","병법치 24를 써서 아군 한 명의 병법치를 24 올린다.","heal","any","single","refill",cost=24,mx=3,coeff=24,req=R)
sk("medium_supply","중보급","소보급의 2배 회복.","heal","any","single","heal",cost=48,hit=95,mx=3,coeff=6,req=RB)
sk("awaken","각성","아군 한 명의 모든 상태이상을 푼다.","heal","any","single","cure",cost=36,mx=3,req=RB)
sk("reinforce","원군","십자 범위 아군을 소보급만큼 회복한다.","heal","any","cross","heal",cost=72,hit=95,mx=3,coeff=3,req=RS)
sk("grand_supply","대보급","소보급의 3배 회복.","heal","any","single","heal",cost=72,hit=95,mx=3,coeff=9,req=RA)
sk("grand_reinforce","대원군","3×3 범위 아군을 중보급만큼 회복한다.","heal","any","square","heal",cost=120,hit=95,mx=3,coeff=6,req=RA)
sk("transport","수송","멀리 있는 아군도 중보급만큼 회복한다.","heal","any","single","heal",cost=60,hit=95,mx=99,coeff=6,req=RA)
sk("counsel","간언","병법치 48을 써서 아군 한 명의 병법치를 48 올린다.","heal","any","single","refill",cost=48,mx=3,coeff=48,req=RA)
E, EB = ["exhort"], ["exhort","exhort_boost"]
sk("swift_wit","기민","십자 범위 아군의 민첩 +20%.","buff","any","cross","agility",cost=48,hit=95,mx=3,dur=2,req=E)
sk("uplift","고양","십자 범위 아군의 사기 +20%.","buff","any","cross","morale",cost=48,hit=95,mx=3,dur=2,req=E)
sk("rouse","분기","아군 한 명의 공격력 +20%.","buff","any","single","attack",cost=24,hit=95,mx=3,dur=2,req=E)
sk("harden","견고","아군 한 명의 방어력 +20%.","buff","any","single","defense",cost=24,hit=95,mx=3,dur=2,req=E)
sk("inspire","고무","십자 범위 아군의 공격력 +20%.","buff","any","cross","attack",cost=60,hit=90,mx=3,dur=2,req=EB)
sk("fortify_line","강진","십자 범위 아군의 방어력 +20%.","buff","any","cross","defense",cost=60,hit=90,mx=3,dur=2,req=EB)
sk("forced_march","강행","아군 한 명의 이동력 +2.","buff","any","single","speed",cost=36,hit=95,mx=3,dur=2,coeff=2,req=EB)
H1, HB = ["hinder"], ["hinder","hinder_boost"]
sk("slow","둔화","십자 범위 적의 민첩 -20%.","status","any","cross","agility",cost=48,hit=75,mn=1,mx=3,dur=2,req=H1)
sk("unrest","동요","십자 범위 적의 사기 -20%.","status","any","cross","morale",cost=48,hit=75,mn=1,mx=3,dur=2,req=H1)
sk("weaken","쇠약","적 한 명의 공격력 -20%.","status","any","single","attack",cost=30,hit=75,mn=1,mx=3,dur=2,req=H1)
sk("enfeeble","약화","적 한 명의 방어력 -20%.","status","any","single","defense",cost=30,hit=75,mn=1,mx=3,dur=2,req=H1)
sk("shrink","위축","십자 범위 적의 공격력 -20%.","status","any","cross","attack",cost=60,hit=70,mn=1,mx=3,dur=2,req=HB)
sk("collapse","와해","십자 범위 적의 방어력 -20%.","status","any","cross","defense",cost=60,hit=70,mn=1,mx=3,dur=2,req=HB)
G, GB = ["stratagem"], ["stratagem","stratagem_boost"]
sk("seal_spell","봉책","적 한 명의 책략을 막는다.","status","any","single","seal",cost=36,hit=75,mn=1,mx=3,dur=2,req=G)
sk("poison_mist","독연","적 한 명에게 독을 건다.","status","any","single","poison",cost=36,hit=75,mn=1,mx=3,dur=3,req=G)
sk("confuse","혼란","적 한 명을 혼란에 빠뜨린다.","status","any","single","confusion",cost=36,hit=70,mn=1,mx=3,dur=2,req=G)
sk("root_spell","봉쇄","적 한 명의 이동을 막는다.","status","any","single","root",cost=54,hit=75,mn=1,mx=3,dur=2,req=GB)
sk("disarm_spell","봉격","적 한 명의 공격을 막는다.","status","any","single","disarm",cost=54,hit=75,mn=1,mx=3,dur=2,req=GB)
sk("great_confusion","대혼란","십자 범위 적을 혼란에 빠뜨린다.","status","any","cross","confusion",cost=72,hit=65,mn=1,mx=3,dur=2,req=GB)
sk("poison_fog","독무","십자 범위 적에게 독을 건다.","status","any","cross","poison",cost=72,hit=65,mn=1,mx=3,dur=3,req=GB)
dump("skills.json", {"Skills": S})

# ---------------- equipment.json / items.json ----------------
def eq(name, desc, slot, typ, bonus, traits=(), learn=()):
    d = {"Name":name,"Desc":desc,"Slot":slot,"Type":typ,"Bonus":bonus}
    if traits: d["Traits"] = list(traits)
    if learn: d["Learn"] = list(learn)
    return d
equipment = {
 "iron_sword": eq("철검","평범한 쇠 검.","weapon","sword",[1,0,0,0,0,0]),
 "iron_spear": eq("철창","평범한 쇠 창.","weapon","spear",[1,0,0,0,0,0]),
 "short_bow": eq("단궁","짧은 활.","weapon","bow",[1,0,0,0,0,0]),
 "bamboo_fan": eq("죽선","대나무 부채.","weapon","fan",[0,0,1,0,0,0]),
 "gujeong_blade": eq("고정도","싸움터에서 수습한 명검. 위압을 준다.","weapon","sword",[3,0,0,0,0,0],["intimidate"]),
 "leather_armor": eq("가죽갑옷","질긴 가죽 갑옷.","armor","armor",[0,1,0,0,0,1]),
 "plain_clothes": eq("평복","평범한 옷.","armor","robe",[0,1,0,0,0,0]),
 "war_horse": eq("군마","잘 길들인 말.","accessory","any",[0,0,0,1,0,0],learn=["dash"]),
 "ration_bag": eq("군량낭","군량을 담은 주머니.","accessory","any",[0,0,0,0,0,2],learn=["supply"]),
 "route_map": eq("지도","지형을 적은 지도.","accessory","any",[0,0,0,1,0,0],learn=["rough_terrain"]),
 "medicine_pouch": eq("약낭","약을 담은 주머니. 구휼 책략을 쓰게 한다.","accessory","any",[0,0,1,0,0,0],["relief"],["medicine"]),
 "tactics_primer": eq("병법입문","병법 입문서. 독려 책략을 쓰게 한다.","accessory","any",[0,0,1,0,0,0],["exhort"],["composure","swiftness"]),
 "imperial_seal": eq("옥쇄","전국새. 반드시 치명타를 낸다.","accessory","any",[0,0,0,0,0,0],["resolve_strike"]),
}
dump("equipment.json", {"Equipment": equipment})
items = {
 "small_food": {"Name":"소군량","Desc":"병력을 220 회복한다.","Effect":"hp","Amount":220},
 "medium_food": {"Name":"중군량","Desc":"병력을 700 회복한다.","Effect":"hp","Amount":700},
 "small_scroll": {"Name":"소병법단","Desc":"병법치를 15 회복한다.","Effect":"mp","Amount":15},
}
promo = [("heavy_cavalry","중기병"),("royal_guard","친위대"),("heavy_infantry","중보병"),("guards","근위대"),("crossbow","노병"),
         ("repeating_crossbow","연노병"),("staff_officer","참모"),("strategist","군사"),("sorcerer","방술사"),("mystic","선술사"),
         ("ruffian","흉적"),("outlaw","의적")]
for c, n in promo:
    items["promote_"+c] = {"Name":"승급서("+n+")","Desc":n+"(으)로 승급한다.","Effect":"promote","Class":c}
dump("items.json", {"Items": items})

# ---------------- characters.json ----------------
def ch(name, cls_, stats, lvl, face, portrait=None, faction=None, sprite=None, scene=None, palette=None,
       traits=(), learn=(), equip=(), lord=False, playable=False, template=False):
    d = {"Name":name,"Class":cls_,"Stats":stats,"Level":lvl}
    if lord: d["Lord"] = True
    if playable: d["Playable"] = True
    if template: d["Template"] = True
    d["Traits"] = list(traits); d["Learn"] = list(learn); d["Equipment"] = list(equip)
    look = {}
    if sprite: look["Sprite"] = sprite
    if scene: look["Scene"] = scene
    if portrait: look["Portrait"] = portrait
    if faction: look["Faction"] = faction
    if palette: look["Palette"] = palette
    d["Look"] = look
    return d
def tri(l, m, d): return [l, m, d]
SW, SP, BW, FN = ["iron_sword","leather_armor"], ["iron_spear","leather_armor"], ["short_bow","leather_armor"], ["bamboo_fan","plain_clothes"]
C = {
 "liubei": ch("유비","light_infantry",[91,76,82,75,87,99],1,0,"liubei","shu","liubei","liubei_noncombat",
    learn=["benevolence","employment","supply","peerless","free_item"],equip=["iron_sword","leather_armor","tactics_primer"],lord=True,playable=True),
 "guanyu": ch("관우","light_cavalry",[100,98,79,83,85,92],1,0,"guanyu","shu","guanyu","guanyu_noncombat",
    learn=["valor","relentless","warlord","deadly","counter_art"],equip=["iron_spear","leather_armor","war_horse"],playable=True),
 "zhangfei": ch("장비","light_cavalry",[83,99,42,92,72,74],1,0,"zhangfei","shu","zhangfei","zhangfei_noncombat",
    learn=["battle_fury","sweep_art","relentless","cross_strike","square_strike"],equip=["iron_spear","leather_armor","war_horse"],playable=True),
 "jianyong": ch("간옹","archer",[36,42,76,66,94,96],2,0,"jianyong","shu",
    learn=["free_item","supply","medicine","discipline","insight"],equip=["short_bow","leather_armor","medicine_pouch"],playable=True),
 "chengyuanzhi": ch("정원지","bandit",[58,66,30,58,47,42],4,0,"chengyuanzhi","turban",equip=["iron_sword","plain_clothes","route_map"]),
 "dengmao": ch("등무","bandit",[49,62,25,57,45,39],2,0,"dengmao","turban",equip=["iron_sword","plain_clothes","ration_bag"]),
 "zhangjiao": ch("장각","scholar",[82,25,94,74,84,93],2,0,"zhangjiao","turban",palette={"themeA":tri("#f4dc70","#e0c030","#9a7a10"),"hair":"#d8d8d8"},
    traits=["exhort"],equip=["bamboo_fan","plain_clothes","tactics_primer"]),
 "huaxiong": ch("화웅","light_cavalry",[82,92,39,78,60,53],6,0,"huaxiong","dong","cavalry",palette={"iron":tri("#8a8a94","#55555e","#2e2e36"),"horseSkin":tri("#55555e","#33333a","#1c1c22")},
    traits=["intimidate"],equip=["iron_spear","leather_armor","war_horse"]),
 "huzhen": ch("호진","light_cavalry",[66,72,48,66,51,48],4,0,"huzhen","dong",palette={"horseSkin":tri("#b8b0a0","#8a8272","#5c5648")},
    equip=["iron_spear","leather_armor","war_horse"]),
 "lubu": ch("여포","light_cavalry",[95,100,34,99,76,42],9,0,"lubu","dong","cavalry",palette={"iron":tri("#6a6a78","#3a3a46","#1a1a22"),
    "themeA":tri("#d04848","#9a2424","#5a1010"),"horseSkin":tri("#e06050","#b03020","#741a10"),"horseMane":tri("#901818","#601010","#300808")},
    traits=["valor"],equip=["iron_spear","leather_armor","war_horse"]),
 "liru": ch("이유","scholar",[61,25,93,65,57,43],7,0,"liru","dong",palette={"themeA":tri("#9a8ab8","#6a5a8a","#3a2e58")},
    traits=["stratagem"],equip=["bamboo_fan","plain_clothes","tactics_primer"]),
 "lijue": ch("이각","light_cavalry",[71,78,52,69,50,45],7,0,"lijue","dong",palette={"horseSkin":tri("#a07850","#785430","#4c3418")},
    equip=["iron_spear","leather_armor","war_horse"]),
 # cast known from docs/officers.md (levels are placeholders)
 "sunqian": ch("손건","scholar",[54,32,82,65,85,92],3,0,"sunqian","shu",equip=FN),
 "zhaoyun": ch("조운","light_cavalry",[92,97,78,96,90,90],8,0,"zhaoyun","gongsun",palette={"horseSkin":tri("#ffffff","#e8e8ec","#b0b0b8")},equip=SP),
 "gongsunzan": ch("공손찬","light_cavalry",[89,86,63,86,72,79],8,0,"gongsunzan","gongsun",palette={"horseSkin":tri("#ffffff","#e8e8ec","#b0b0b8")},equip=SP),
 "yangang": ch("엄강","light_cavalry",[77,80,48,76,61,59],6,0,"yangang","gongsun",equip=SP),
 "yuanshao": ch("원소","light_cavalry",[86,69,74,67,81,91],10,0,"yuanshao","yuan",palette={"themeA":tri("#f0d860","#d0b030","#8a7410")},equip=SP),
 "wenchou": ch("문추","light_cavalry",[86,95,35,88,63,59],10,0,"wenchou","yuan",equip=SP),
 "yanliang": ch("안량","light_cavalry",[87,94,40,84,67,62],10,0,"yanliang","yuan",equip=SP),
 "zhanghe": ch("장합","light_cavalry",[91,90,77,91,80,78],11,0,"zhanghe","yuan",equip=SP),
 "chunyuqiong": ch("순우경","light_infantry",[73,78,47,65,44,51],11,0,"chunyuqiong","yuan",equip=SW),
 "gaolan": ch("고람","light_infantry",[85,86,65,79,68,66],10,0,"gaolan","yuan",equip=SW),
 "tianfeng": ch("전풍","scholar",[82,29,96,68,70,77],9,0,"tianfeng","yuan",equip=FN),
 "shenpei": ch("심배","light_infantry",[80,59,83,65,62,61],9,0,"shenpei","yuan",equip=SW),
 "yuanshu": ch("원술","light_infantry",[65,68,58,59,63,49],8,0,"yuanshu","yuan",equip=SW),
 "caocao": ch("조조","light_infantry",[96,72,94,82,88,96],14,0,"caocao","wei",equip=SW),
 "dongzhuo": ch("동탁","light_cavalry",[77,84,68,59,46,38],12,0,"dongzhuo","dong",palette={"themeA":tri("#7a5a9a","#523470","#2e1a44"),"horseSkin":tri("#55555e","#33333a","#1c1c22")},equip=SP),
 # soldier templates
 "troop_turban": ch("황건 도적","bandit",[35,45,20,40,45,40],2,0,"turban_soldier",template=True,equip=["iron_sword","plain_clothes"]),
 "troop_footman": ch("일반 부대","light_infantry",[50,48,30,40,40,45],4,0,template=True,equip=SW),
 "troop_archer": ch("일반 부대","archer",[40,43,30,52,45,40],4,0,template=True,equip=BW),
}
dump("characters.json", {"Characters": C})

# ---------------- stages ----------------
cmap = {"유비":"liubei","관우":"guanyu","장비":"zhangfei","간옹":"jianyong","정원지":"chengyuanzhi","등무":"dengmao","장각":"zhangjiao","화웅":"huaxiong",
        "호진":"huzhen","여포":"lubu","이유":"liru","이각":"lijue"}
imap = {"소군량":"small_food","소병법단":"small_scroll","중군량":"medium_food"}
troop = {"B01":("troop_turban",2),"B02":None,"B03":None}
stage_faction = {"B01":"turban","B02":"dong","B03":"dong"}
stage_next = {"B01":"join","B02":"sword"}
for s in old["Stages"]:
    sid = s["ID"]; o = dict(s)
    en = []
    for e in s["Enemies"]:
        n = e["Officer"]
        if n in cmap:
            en.append({"Character":cmap[n],"X":e["X"],"Y":e["Y"]})
        else:
            tpl = "troop_turban" if sid == "B01" else ("troop_archer" if old["Officers"][n]["Class"] == "궁병" else "troop_footman")
            sp = {"Character":tpl,"X":e["X"],"Y":e["Y"]}
            lvl = old["Officers"][n]["Level"]
            if lvl != C[tpl]["Level"]: sp["Level"] = lvl
            en.append(sp)
    o["Enemies"] = en
    o["Boss"] = cmap[s["Boss"]]
    o["Reward"] = {imap[k]: v for k, v in s["Reward"].items()}
    o["Duels"] = [dict(d, Ally=cmap[d["Ally"]], Enemy=cmap[d["Enemy"]]) for d in s.get("Duels", [])]
    o["Faction"] = stage_faction[sid]
    if sid in stage_next: o["Next"] = stage_next[sid]
    dump("stages/%s.json" % sid, o)

# ---------------- script.json ----------------
nodes = []
stage_of = {"prep1":"B01","prep2":"B02","prep3":"B03"}
for n in old["Nodes"]:
    d = {k: v for k, v in n.items() if k != "Officer"}
    if n.get("Officer"): d["Character"] = cmap[n["Officer"]]
    if n["ID"] in stage_of: d["Stage"] = stage_of[n["ID"]]
    if n["Kind"] == "grant": d["Item"] = "gujeong_blade"
    if n.get("Event") == "join-간옹": d["Event"] = "join-jianyong"
    if n.get("Event") == "grant-고정도": d["Event"] = "grant-gujeong_blade"
    nodes.append(d)
dump("script.json", {"Nodes": nodes})
print("ok")
