# 장비와 아이템

범위: 장비(`equipment.json`)와 아이템(`items.json`)의 필드, 장착 규칙, 장비가 주는 특성·기술·학습 풀, 장비 특수 효과 세 가지, 소모품·승급서의 사용 규칙, 보상·인벤토리·창고에서의 취급.
기준은 `assets/data/equipment.json`·`items.json`·`traits.json`과 `internal/content/vocab.go`, `internal/core/`이며, 값이 다르면 이 파일들을 따른다. 개별 장비·아이템 목록은 옮기지 않는다.

주머니·보상 흐름은 [logic-campaign.md](logic-campaign.md), 필드 검증 위치는 `internal/content/validate.go`, 특성 전반은 [logic-traits.md](logic-traits.md), 능력치 계산은 [logic-combat.md](logic-combat.md), 승급·성장은 [logic-growth.md](logic-growth.md)를 본다.

## 1. 장비(`Equipment`)

| 필드 | 뜻 |
|---|---|
| `Name`, `Desc` | 표시 이름과 설명. `Name`은 비어 있을 수 없다. |
| `Slot` | `weapon`, `armor`, `accessory` 중 하나. |
| `Type` | 장착 형식. 무기는 `sword spear bow fan`, 방어구는 `armor robe`, 장신구는 `any`. |
| `Bonus` | 병종 보너스에 더하는 6개 값. 순서는 병종 `Bonus`와 같아서 공격, 방어, 정신, 민첩, 사기, 병력이다. |
| `Traits` | 착용하는 동안 갖는 특성(주어진 특성, `Cost` 0). 모두 `traits.json`에 있어야 한다. |
| `Learn` | 착용하는 동안 학습 후보에 더해지는 특성. 모두 배울 수 있는 특성(`Cost >= 1`)이어야 한다. |
| `Skills` | 착용하는 동안 쓸 수 있는 기술. 모두 `skills.json`에 있어야 한다. |

```json
"fangtian_halberd": {"Name": "방천화극", "Slot": "weapon", "Type": "spear", "Bonus": [3, 0, 0, 0, 0, 0], "Traits": ["halberd_sweep"]}
```

### 장착 규칙

- 한 사람은 `Slot`마다 하나씩 장비를 가진다.
- 무기는 병종의 `Weapon`, 방어구는 병종의 `Armor`와 `Type`이 같아야 한다(`carries`). 장신구는 형식 제한이 없다. 시작 장비, 스폰의 `Equipment`, 저장 복원이 모두 같은 규칙으로 검사한다.
- 장착과 해제(`equip`, `unequip`)는 출전 준비 노드에서만 되고, 플레이어블 인물이며 전사하지 않았고 부관이 아닌 인물에게만 된다. 장착은 창고(`Warehouse`)의 장비를 쓰고, 같은 칸에 있던 장비는 창고로 돌아간다. 해제한 장비도 창고로 간다.
- 부관으로 임명되면 그 인물의 장비는 창고로 가고, 일기토에서 전사한 플레이어블 인물의 장비도 창고로 돌아간다.
- 장비의 보너스·특성·기술·학습 풀은 착용하는 동안만 적용되고, 병종이 바뀌어도 장비는 그대로 남는다. 병종 승급 때 장비 호환은 다시 검사하지 않는다.

## 2. 장비 특수 효과

장비의 `Traits`가 가리키는 특성이 효과 키를 가진다. 키는 `internal/content/vocab.go`의 어휘이고 아래 셋은 현재 장비가 쓴다. 효과는 착용자의 진영(아군·적군)을 따른다.

### retreat_sweep (요극)

착용자의 행동 도중 적 하나가 퇴각하면, 착용자가 살아 있는 한 그 퇴각한 적을 제외한 다른 모든 살아 있는 적에게 차례로 일반 공격을 한다. 대상은 착용자의 일반 공격 사거리 안에 있는 적이고 순서는 유닛 ID 순이다.

- 일반 공격이므로 명중, 치명, 2회 공격, 반격이 보통 공방과 같이 적용된다. 별도의 피해 비율은 없다.
- 추가 공격으로 적이 퇴각해도 다시 요극이 일어나지 않는다(연쇄 없음).
- 일기토 결과로 적이 퇴각한 경우에도 승자(착용자)에게 일어난다([logic-duel.md](logic-duel.md)).
- 지속 피해(독, 화상)로 퇴각한 경우는 일어나지 않는다.
- 추가 공격마다 `followup-start`, `followup-end` 이벤트가 나간다. 퇴각이 일어나면 시작에 `retreat-sweep` 이벤트가 하나 나간다.

### counter_always (확정 반격)

착용자는 사거리 안에서 받는 모든 물리 공격에 반격 자세 없이도 반격한다. 반격 조건은 반격 자세(`counter` 상태)와 같다. 즉 공격자가 착용자의 일반 공격 사거리 안에 있어야 하고, 반격 피해는 보통 타격의 `Rules.CounterShare` 비율이며 2회 공격 확률도 굴린다.

### turn_defense (차례 방어)

효과 값이 방어 강화 퍼센트이다. 유닛이 전장에 나타날 때(배치, 증원 포함)와 자기 진영 차례가 시작될 때마다 `defense` 상태를 `Rules.BuffTurns`턴 동안 그 퍼센트로 건다. 더 큰 기존 강화가 있으면 그 값을 유지한다.

세 효과를 주는 장비와 특성 이름의 연결은 `equipment.json`과 `traits.json`에서 `Traits`를 따라가면 된다. 현재는 방천화극(요극), 적토마(`move` +2와 확정 반격), 금강갑(차례 방어)을 여포가 함께 가진다.

## 3. 아이템(`Items`)

| 필드 | 뜻 |
|---|---|
| `Name`, `Desc` | 표시 이름과 설명. |
| `Effect` | `hp`, `mp`, `promote` 중 하나. 다른 값은 로드 오류. |
| `Amount` | `hp`, `mp`의 회복량. 1 이상. |
| `Class` | `promote`가 만드는 병종. 병종이 있어야 하고 `Tier`가 1 이상이어야 한다. |

```json
"bean": {"Name": "콩", "Effect": "hp", "Amount": 220}
"promote_heavy_infantry": {"Name": "승급서(중보병)", "Effect": "promote", "Class": "heavy_infantry"}
```

ID 규칙은 승급서가 `promote_<병종ID>`이다(`assets/data/README.md`).

### 사용

아이템은 전투 중 `item` 명령으로 쓴다. 성공하면 쓴 유닛의 이동과 행동이 모두 끝난다. 아군은 공용 `Inventory`에서, 적은 자기 `Items`에서 소모한다. 적의 시작 재고는 `Rules.EnemyStock`이다.

- `hp`, `mp`: 쓰는 유닛과 같은 진영이고 거리가 1 이하(자기 자신 포함)인 살아 있는 대상 하나에게 쓴다. `hp`는 `Amount`에 쓴 유닛의 회복량 보너스(`heal_bonus`)를 곱해 최대 병력까지 채운다. `mp`는 `Amount`를 그대로 최대 병법까지 채운다. 특성 `item_share`가 있으면 대상 옆의 같은 진영 유닛 한 명에게도 같은 효과가 간다. 이미 행동했으면 `ActionSpent`.
- `promote`: 자기 자신에게만, 아군만 쓴다(`InvalidTarget`). 현재 병종의 `Promotes`에 `Class`가 있어야 하고(`InvalidPromotion`), 인물 레벨이 `Rules.PromoteLevels[현재 병종 Tier]` 이상이어야 하며(`InsufficientLevel`), 승급서가 인벤토리에 있어야 한다(`MissingItem`). 쓰면 승급서 하나가 줄고 병종이 바뀌며 병력과 병법이 최대로 찬다. `promotion` 이벤트가 나간다. 전투 중 가능한 행동이라 이미 행동했으면 `ActionSpent`.
- `steal` 기술은 대상의 `Items` 하나를 30% 확률로 가져온다. 아군이 가져오면 `Inventory`에, 적이 가져오면 적 유닛의 `Items`에 들어간다.

## 4. 얻는 길

- 시작 인벤토리: `game.json`의 `Campaign.Inventory`(아이템).
- 승리 보상: 스테이지의 `Reward`(아이템, `Inventory`로).
- 노드 부여: `grant` 노드는 `Item`이 장비면 `Warehouse`로, 아이템이면 `Inventory`로 넣는다.
- 장비는 일기토 전사나 부관 임명으로 창고에 돌아오는 경우 외에 소모되지 않는다. 아이템 주머니와 창고의 키는 각각 `Items`, `Equipment`에 있어야 하며 아니면 불러오기가 `CorruptSave`로 거부한다.
