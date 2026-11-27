# 장비와 아이템

범위: 장비(`equipment.json`)와 아이템(`items.json`)의 필드, 장착 규칙, 장비가 주는 특성·기술·학습 풀, 장비 특수 효과 세 가지, 소모품·인수(승급 아이템)의 사용 규칙, 보상·인벤토리·창고에서의 취급.
기준은 `assets/data/equipment.json`·`items.json`·`traits.json`과 `internal/content/vocab.go`, `internal/core/`이며, 값이 다르면 이 파일들을 따른다. 개별 장비·아이템 목록은 옮기지 않는다.

주머니·보상 흐름은 [logic-campaign.md](logic-campaign.md), 필드 검증 위치는 `internal/content/validate.go`, 특성 전반은 [logic-traits.md](logic-traits.md), 능력치 계산은 [logic-combat.md](logic-combat.md), 승급·성장은 [logic-growth.md](logic-growth.md)를 본다.

## 1. 장비(`Equipment`)

| 필드 | 뜻 |
|---|---|
| `Name`, `Desc` | 표시 이름과 설명. `Name`은 비어 있을 수 없다. |
| `Slot` | `weapon`, `armor`, `accessory` 중 하나. |
| `Icon` | UI 그림 `item_<Icon>`(`tools/spritetool/assets/ver5-ui`). 같은 그림을 여러 장비가 나눠 써도 된다. 게임 데이터의 모든 장비가 시트에 그림을 갖는지는 GUI 시험(`TestEveryEquipmentIconIsOnTheUISheet`)이 본다. |
| `Type` | 장착 형식. 무기는 `sword spear bow fan`, 방어구는 `armor robe`, 장신구는 `any`. |
| `Line`, `Tier` | 단계가 있는 일반 장비의 계열(`sword spear bow fist armor robe`)과 그 계열의 단계(1 D·2 C·3 B). 둘 다 있거나 둘 다 없다. 같은 계열은 칸·종류가 같고 단계마다 하나뿐이며, `Rules.GearTiers`의 모든 단계에 장비가 있어야 한다. 고유 장비·장신구·부채는 단계가 없다. 아래 장비 단계 절. |
| `Bonus` | 병종 보너스에 더하는 6개 값. 순서는 병종 `Bonus`와 같아서 공격, 방어, 정신, 민첩, 사기, 병력이다. |
| `Traits` | 착용하는 동안 갖는 특성(주어진 특성, `Cost` 0). 모두 `traits.json`에 있어야 한다. |
| `Learn` | 착용하는 동안 학습 후보에 더해지는 특성. 모두 배울 수 있는 특성(`Cost >= 1`)이어야 한다. |
| `Palette` | 고유한 색을 가진 말: 도트 태그 묶음(`horseSkin`·`horseMane`·`saddle` 등) → 색. 기마 병종이 이 장신구를 차면 전투 도트(배치·전투·일기토)의 말이 이 색으로 칠해진다(장수 `Look.Palette` 위에 덮음). 도보 병종은 그대로다. 예: 백마는 흰 털·흰 갈기, 절영은 검은 말, 적토마는 붉은 말. |
| `Skills` | 착용하는 동안 쓸 수 있는 기술. 모두 `skills.json`에 있어야 한다. |
| `Owner` | 추천 주인. `Playable` 인물이고 그 시작 병종이 들 수 있어야 한다. 이야기로 받은 장비는 창고로 가고 저절로 차지 않는다(`grant` 노드에 `Character`를 적은 것만 그 인물이 곧바로 찬다). 게임 AI(자동 진행·검증 실행·`_campaign`·`_balance`)만 출전 전에 창고에서 꺼내 이 인물에게 채우고(`Engine.OwnerPiece`; 플레이어의 `Space`는 채우지 않는다), 장면 이동 게임은 그 지점까지 이야기가 준 것만 채운다. 그 칸에 이미 자기 주인 장비를 찬 인물에게는 채우지 않는다. 탐험 이벤트가 준 주인 장비는 초록 안내가 장착까지 이끈다([presentation-explore.md](presentation-explore.md)). |

```json
"fangtian_halberd": {"Name": "방천화극", "Slot": "weapon", "Type": "spear", "Bonus": [3, 0, 0, 0, 0, 0], "Traits": ["halberd_sweep"]}
```

### 장비 단계와 등급 글자

보너스 숫자를 사용자에게는 글자로 보인다. D = 1, C = 2, B = 3, A = 4(내부 값은 숫자 그대로). 일반 장비는 장(章)에 따라 한 단계씩 오른다.

| 단계 | 장 | 검 | 창 | 활 | 무도가 장갑 | 갑옷(무장) | 옷(문관·산적·무도가) |
|---|---|---|---|---|---|---|---|
| 1 (D) | 1장 | 구리검 `copper_sword` | 구리창 `copper_spear` | 단궁 `short_bow` | 천장갑 `cloth_gloves` | 가죽갑옷 `leather_armor` | 무명옷 `cotton_clothes` |
| 2 (C) | 2장(17장 광릉부터) | 철검 `iron_sword` | 철창 `iron_spear` | 대궁 `long_bow` | 가죽장갑 `leather_gloves` | 구리갑옷 `bronze_armor` | 가죽옷 `leather_clothes` |
| 3 (B) | 3장(28장부터, 아직 판 없음) | 강검 `steel_sword` | 강창 `steel_spear` | 강궁 `steel_bow` | 철장갑 `iron_gloves` | 철갑옷 `iron_armor` | 호피복 `tiger_skin` |

무기는 주 능력치(공격) 하나가 등급 숫자이고, 갑옷은 방어와 병력이, 옷은 방어만 그 숫자다. 값은 무기 1200·2400·4800, 갑옷 1000·2000·4000, 옷 600·1200·2400이다. 어느 병종이 어느 계열을 입는지는 병종의 `Weapon`·`Armor`와 장비의 `Type`이 정한다(무장 병종은 `armor`, 서생·참모 계열과 적병·황건적·흉적·의적·무도가·권사·권성은 `robe`, 무도가 계열 무기는 `fist`).

`Rules.GearTiers`(`game.json`)는 `{Chapter, Tier}`의 오름차순 표로, 장 번호가 `Chapter` 이상이면 그 `Tier`다. 로드할 때 전투 노드가 든 장 번호로 `Stage.GearTier`를 정한다. 전투에서 적·AI 병력(`Temp`)은 `Equipment`를 덮어쓰지 않은 스폰이라면 일반 장비를 그 스테이지의 단계로 바꿔 입는다(`Data.GearAt`, 같은 `Line`의 그 단계 장비). 스폰이 `Equipment`를 적으면 그대로 입는다. 단계가 없는 고유 장비는 그대로다. 새로 합류하는 장수도 합류하는 전투의 단계로 바꿔 입는다. 시작 장비는 모두 1단계로 적고, 병종이 못 드는 장비는 `refitGear`가 값이 가까운 같은 칸의 상점 장비로 바꾼다(단계마다 값이 같아 같은 단계끼리 맞는다).

상점 `weapons`는 1단계, `weapons_iron`은 2단계 장비를 판다. 19장 업의 대장간만 `weapons_iron`이라, 철 계열은 업부터 살 수 있다. 그 탐험에서 처음 팔리는 물건이 있는 상점(`Data.NewStock`, [presentation-explore.md](presentation-explore.md) "새 상품 안내")은 탐험 안내 판(대장장이와 그 가게로 드는 문)이 빨강이고(`plateHot`), 아니면 노랑이다. 업은 처음이라 빨강이고, 다음 탐험에서는 다시 노랑이다.

### 장착 규칙

- 한 사람은 `Slot`마다 하나씩 장비를 가진다.
- 무기는 병종의 `Weapon`, 방어구는 병종의 `Armor`와 `Type`이 같아야 한다(`Data.Carries`). 특성의 `wield:<무기 종류>` 효과가 있으면 그 종류의 무기도 장착한다(다재다능은 검과 창). 장신구는 형식 제한이 없다. 시작 장비, 스폰의 `Equipment`, 저장 복원, 맵 편집기의 장비 목록이 모두 같은 규칙으로 검사한다.
- 장착과 해제(`equip`, `unequip`)는 출전 준비 노드에서만 되고, 플레이어블 인물이며 전사하지 않았고 부관이 아닌 인물에게만 된다. 장착은 창고(`Warehouse`)의 장비를 쓰고, 같은 칸에 있던 장비는 창고로 돌아간다. 해제한 장비도 창고로 간다.
- 부관으로 임명되면 그 인물의 장비는 창고로 가고, 일기토에서 전사한 플레이어블 인물의 장비도 창고로 돌아간다.
- 장비의 보너스·특성·기술·학습 풀은 착용하는 동안만 적용되고, 병종이 바뀌어도 장비는 그대로 남는다. 병종 승급 때 장비 호환은 다시 검사하지 않는다.

## 2. 장비 특수 효과

장비의 `Traits`가 가리키는 특성이 효과 키를 가진다. 키는 `internal/content/vocab.go`의 어휘이고 아래는 현재 장비가 쓴다. 효과는 착용자의 진영(아군·적군)을 따른다.

### retreat_sweep (요극)

착용자의 행동 도중 적 하나가 퇴각하면, 착용자의 일반 공격 사거리 안에 있는 살아 있는 적 1기에게 일반 공격을 한다. 대상은 체력이 가장 낮은 적이고, 같으면 유닛 ID 순으로 앞선 적이다. 그 대상도 퇴각하면 같은 규칙으로 다음 1기를 공격하고, 대상이 버티거나 사거리 안에 적이 없거나 착용자가 퇴각하면 멈춘다.

- 일반 공격이므로 명중, 치명, 2회 공격, 반격이 보통 공방과 같이 적용된다. 추가 공격의 기본 공격 계수는 효과 값(0.5)만큼, 곧 피해 50%다.
- 연쇄는 추가 공격의 대상이 퇴각할 때만 이어진다. 사방격 등으로 주변 적만 퇴각한 경우는 이어지지 않는다.
- 일기토 결과로 적이 퇴각한 경우에도 승자(착용자)에게 일어난다([logic-duel.md](logic-duel.md)).
- 지속 피해(독, 화상)로 퇴각한 경우는 일어나지 않는다.
- 추가 공격마다 `followup-start`, `followup-end` 이벤트가 나간다. 퇴각이 일어나면 시작에 `retreat-sweep` 이벤트가 하나 나가고, 연쇄 중에는 다시 나가지 않는다.

### counter_always (확정 반격)

착용자는 반격 자세(`counter` 상태) 없이도 반격 자세처럼 반격한다. 모든 유닛이 사거리 안의 공격자에게 반격하지만(`Rules.PlainCounterShare`), 착용자의 반격은 보통 타격의 `Rules.CounterShare` 비율이며 2회 공격 확률도 굴린다.

### turn_defense (차례 방어)

효과 값이 방어 강화 퍼센트이다. 유닛이 전장에 나타날 때(배치, 증원 포함)와 자기 진영 차례가 시작될 때마다 `defense` 상태를 `Rules.BuffTurns`턴 동안 그 퍼센트로 건다. 더 큰 기존 강화가 있으면 그 값을 유지한다.

### sweep_arc (청공)

효과 값이 피해 비율이다. 착용자가 근접으로 앞을 칠 때(대상이 8칸 안, 원거리가 아님) 공격자를 둘러싼 8칸을 화면 방향(N, NE, E, SE, S, SW, W, NW)으로 세워 앞 칸을 중심으로 양옆 둘씩 5칸의 적도 그 비율로 피해를 받는다. 앞 칸은 대상이라 온전한 피해를 받는다. 앞이 NE면 NW·N·NE·E·SE, SW면 SE·S·SW·W·NW, E면 N·NE·E·SE·S다. 단일 대상 물리 행동(일반 공격 포함)에 쓰이고, 찔러공격·사방격과 겹치면 각 칸에서 가장 큰 비율을 쓴다. 화면 방향의 격자 변화는 [presentation-scene.md](presentation-scene.md)의 방향 규칙(NE는 y−1, SE는 x+1, SW는 y+1, NW는 x−1)과 같다.

### evade (의천)

효과 값은 %p이다. 착용자를 향한 물리 공격의 명중률에서 그만큼 뺀 뒤 하한·상한을 적용한다. 책략·상태이상 명중에는 영향이 없다.

### double_ward·crit_immune (금흉)

금흉갑의 금흉은 두 효과를 함께 가진다. 치명타로 판정된 타격(`crit_immune`, 금강과 같은 규칙)과 연타·`Strikes` 2·반격 연타의 두 번째 타격(`double_ward`)이 모두 빗나감(`miss` 이벤트)이 된다. 급소(`miss_half`)가 있으면 빗나간 타격의 그 비율은 들어간다.

### `hit_status:` + 상태이상 (유성)

효과 값이 턴 수다. 착용자의 물리 타격이 명중해 피해를 주면 대상에게 그 상태이상을 반드시 건다(명중 판정 없음). 유성추는 `hit_status:disarm` 2로 봉격(일반 공격과 물리 병법 불가, 봉격 책략과 같은 2턴)을 건다. 상태이상 면역(군율·엄명)은 막는다. 반격·협공의 타격에도 걸린다.

### charm·mana_max·mana_plus (풍채·윤건·신서)

`charm`은 매력을 그만큼 높게 센다(최대 병력과 부관 보정률, [logic-growth.md](logic-growth.md) 3절). `mana_max`는 최대 병법치에, `mana_plus`는 매 차례 병법치 회복량에 그 값을 더한다([logic-combat.md](logic-combat.md) 2절). 비단 도포 +10, 제갈건 +30, 맹덕신서 +10이다.

### 기존 효과를 쓰는 22~27장 장비

적로는 답파(`sure_footing`, 지형 보정 100 미만이면 100) 민첩 A(4), 적토마는 민첩 A(4), 방천화극은 공격 A(4), 거울갑옷은 방어 A(4), 청룡언월도는 공격 B(3)·민첩 D(1), 장팔사모는 공격 B(3)·사기 D(1)이다. 양유기의 활은 백보천양(`ranged_reach` +2, 원시와 합산), 진법서는 금쇄(`aura_buff:defense` 20, 둘레 여덟 칸 아군), 백우선은 백출(`cost_half`), 학창의는 학창(`turn_buff:mind` 20)을 준다. 의천검의 `evade` 15는 보법(`evade` 15)과 합쳐져 30%p가 된다(같은 효과 키는 더한다). 청홍검은 옛 청공검(`qinggang_sword`)의 표시 이름을 바꾼 것이다.

세 효과를 주는 장비와 특성 이름의 연결은 `equipment.json`과 `traits.json`에서 `Traits`를 따라가면 된다. 현재는 방천화극(요극), 적토마(`move` +2와 확정 반격), 당예개갑(차례 방어)을 여포가 함께 가진다.

## 3. 아이템(`Items`)

| 필드 | 뜻 |
|---|---|
| `Name`, `Desc` | 표시 이름과 설명. |
| `Effect` | `hp`, `mp`, `promote` 중 하나. 다른 값은 로드 오류. |
| `Amount` | `hp`, `mp`의 회복량. 1 이상. |
| `Tier` | `promote`가 만드는 병종의 등급: 1 중급, 2 상급. 1 이상 `Rules.PromoteLevels` 길이 이하. 병종은 쓸 때 고른다. |

```json
"bean": {"Name": "콩", "Effect": "hp", "Amount": 220}
"insu_mid": {"Name": "중급 인수", "Icon": "insu", "Effect": "promote", "Tier": 1}
```

인수는 `insu_mid`(중급)와 `insu_high`(상급) 둘이다(`assets/data/README.md`).

### 사용

아이템은 전투 중 `item` 명령으로 쓴다. 성공하면 쓴 유닛의 이동과 행동이 모두 끝난다. 플레이어 아군은 공용 `Inventory`에서, 적과 동맹군(`AI` 아군)은 자기 `Items`에서 소모한다. 시작 재고는 적이 `Rules.EnemyStock`, 동맹군이 `Rules.AllyStock`이다.

- `hp`, `mp`: 쓰는 유닛과 같은 진영이고 거리가 1 이하(자기 자신 포함)인 살아 있는 대상 하나에게 쓴다. `hp`는 `Amount`를, `mp`는 `Amount`를 최대 병력·병법까지 채운다. 쓴 유닛에게 `item_power`(물심양면 0.5)가 있으면 `Amount`가 그만큼 늘어난다. 의술(`heal_bonus`)은 회복 책략에만 붙고 아이템에는 붙지 않는다. 특성 `item_share`가 있으면 대상 옆의 같은 진영 유닛 한 명에게도 같은 효과가 간다. 이미 행동했으면 `ActionSpent`.
- `promote`(인수): 전투 밖에서만 쓴다 — 탐험에서 걸을 때, 전투 노드의 출전 준비, 배치 중(`CanLearn`과 같은 때, 이벤트 대사 중은 아님). 전투 중에는 `PromoteAway`로 거부되고 전투의 합법 명령에도 없다. 명령은 `{Kind: "item", Target: 장수, Item, Option}`이고 `Target`이 승급할 장수(부관이어도 됨), `Option`이 고른 병종이다. 현재 병종의 `Promotes`에 있어야 하고(갈래가 하나면 비워도 그 병종) 인수의 `Tier`가 현재 병종 `Tier` + 1이어야 하며(`InvalidPromotion`), 인물 레벨이 `Rules.PromoteLevels[현재 병종 Tier]` 이상이어야 하며(`InsufficientLevel`), 인수가 인벤토리에 있어야 한다(`MissingItem`). 쓰면 인수 하나가 줄고 병종이 바뀌며, 배치 중이면 그 유닛의 병력·병법이 찬다. `promotion` 이벤트(`Actor`·`Text` 모두 그 장수)가 나가고, 새 병종이 들 수 없는 장비는 창고로 돌아가며 장비마다 `unequip` 이벤트(`Actor` 장수, `Target` 장비)가 나간다. 다른 아이템은 전투 밖에서 `WrongPhase`. `Engine.PromotionsAway(item, 장수)`가 지금 쓸 수 있는 병종(또는 이유)을 준다.
- `steal` 기술은 대상의 `Items` 하나를 30% 확률로 가져온다. 아군이 가져오면 `Inventory`에, 적이 가져오면 적 유닛의 `Items`에 들어간다.

## 4. 얻는 길

- 시작 인벤토리: `game.json`의 `Campaign.Inventory`(아이템).
- 승리 보상: 스테이지의 `Reward`(아이템, `Inventory`로).
- 노드 부여: `grant` 노드는 `Item`이 장비면 `Warehouse`로, 아이템이면 `Inventory`로 넣는다.
- 장비는 일기토 전사나 부관 임명으로 창고에 돌아오는 경우 외에 소모되지 않는다. 아이템 주머니와 창고의 키는 각각 `Items`, `Equipment`에 있어야 하며 아니면 불러오기가 `CorruptSave`로 거부한다.
