# 전투 이벤트와 승패

범위: 전투 중 스테이지가 거는 일회성 이벤트(증원·대사), 일기토 발동, 승리·패배 판정, 턴 제한, 이벤트 상태의 저장·복원.
기준은 `internal/core/battle_events.go`, `internal/core/engine.go`의 `settle()`이며, 값이 다르면 코드와 `assets/data/maps/<ID>.json`의 `Stage`를 따른다.

필드 이름과 검증 규칙은 [data-scenario-node.md](data-scenario-node.md), 일기토 자체의 규칙은 [logic-duel.md](logic-duel.md), 연출은 [presentation-duel.md](presentation-duel.md)·[presentation-replay.md](presentation-replay.md)를 본다.
캠페인 흐름(출전 준비·배치·재도전)은 [logic-campaign.md](logic-campaign.md)에 있다.

## 1. 진영 차례와 라운드

전투 상태의 `Turn`은 `ally`, `support`, `enemy` 중 하나이고 `Round`는 1에서 시작한다.

- `end` 명령(또는 AI의 마지막 행동)이 차례를 넘긴다. 아군 차례가 끝나면 살아 있는 아군 AI 유닛(`Unit.AI`)이 있을 때만 `support`로, 없으면 바로 `enemy`로 간다.
- `support` 다음은 `enemy`, `enemy` 다음은 `ally`이며 이때 `Round`가 1 늘어난다. 즉 한 라운드는 아군, (지원), 적군 차례를 모두 포함한다.
- 아군 AI 유닛은 `support` 차례에만 행동한다(`Unit.ActsIn`). 플레이어는 이들에게 명령할 수 없다.
- 새 차례가 시작될 때마다 `turn` 이벤트가 나온다(`Target`이 차례 이름, `Amount`가 라운드). 자동 저장은 이 이벤트를 쓴다([system-save.md](system-save.md)).

## 2. 판정 시점

`Engine.Apply`는 명령 하나를 처리한 뒤 항상 `settle()`을 부른다. 전투 단계(`Phase == "battle"`)가 아니면 아무것도 하지 않는다. 순서는 다음과 같다.

1. 턴 제한 검사.
2. `battleEvents()`: 전투 이벤트 발동.
3. 승패 검사.

따라서 이벤트는 아군·지원·적군 어느 차례의 어떤 명령 뒤에든 발동할 수 있고, 같은 명령으로 승패가 갈려도 이벤트가 먼저 처리된다.

## 3. 승리와 패배

패배 조건(하나라도 해당하면 패배, 승리보다 우선한다):

- 살아 있는 아군 유닛이 하나도 없다. 아군 AI 유닛도 아군으로 센다.
- 쓰러진(`HP <= 0`) 아군 중 군주(`Lord`) 인물이 있다.

승리 조건은 스테이지의 `Goal`로 갈린다.

- `Goal: "rout"`: 살아 있는 적 유닛이 없다.
- 그 외: 보스(`Stage.Boss`)의 `HP * 100 <= MaxHP * Threshold`. `Threshold`가 0이면 보스가 쓰러져야 하고, 40이면 보스 병력이 최대의 40% 이하일 때 승리한다. 일기토로 보스가 퇴각해 `HP`가 0이 된 경우도 이 식으로 승리가 된다.

승패가 갈리면 `Phase`가 `result`, `Result`가 `victory` 또는 `defeat`가 되고 더 이상 명령을 받지 않는다. 이후 흐름은 [logic-campaign.md](logic-campaign.md)의 이어하기·재도전을 따른다.

승리하면 `Stage.Reward`를 인벤토리에 한 번만 더한다(`Executed["reward-<스테이지ID>"]`). `victory` 이벤트의 `Target`은 스테이지 ID다.

## 4. 턴 제한

`Stage.TurnLimit`이 0이면 무제한이다. 양수이면 `Round > TurnLimit`가 되는 순간(= 마지막 라운드의 적군 차례가 끝나고 아군 차례가 새 라운드로 들어가는 시점) `settle()`이 `defeat`로 만들고 `turn-limit` 이벤트(`Amount`가 제한 값)를 낸다. 마지막 라운드 적군 차례까지는 승리할 수 있다. 이 검사가 이벤트·승패 검사보다 먼저이므로 제한을 넘긴 순간에는 이벤트도 승리도 처리되지 않는다.

## 5. 전투 이벤트(`Stage.Events`)

이벤트는 스테이지 로컬 `ID`를 가진 일회성 규칙이다. 발동하면 `Executed["battle-<스테이지ID>-<이벤트ID>"]`가 참이 되어 다시 발동하지 않는다.

### 5.1 발동 조건(`Trigger`)

| Trigger | 발동 조건 |
|---|---|
| `boss-alone` | 보스가 살아 있고, 보스 외 적 유닛이 모두 쓰러졌다. |
| `duel` | `Duels`에 적은 일기토 ID 중 하나라도 이미 치러졌다(`Executed["duel-<ID>"]`). |

조건을 만족해도 증원 자리를 못 찾으면 발동하지 않고 다음 `settle()`에서 다시 시도한다(5.2).

### 5.2 증원(`Spawns`)

`Spawns`의 유닛은 모두 아군 진영(`ally`)의 AI 유닛으로 나타난다.

- 위치는 각 스폰의 `X`,`Y`에서 맨해튼 거리가 가장 가까운 빈 칸이다. 칸마다 다른 유닛이 서 있지 않고(쓰러진 유닛은 무시), 이번 이벤트에서 이미 정해진 칸이 아니며, 스폰 병종의 이동 지형표에서 비용이 0보다 큰 지형이어야 한다. 거리가 같으면 위에서 아래, 왼쪽에서 오른쪽 순으로 먼저 걸리는 칸을 쓴다.
- 모든 스폰의 자리를 먼저 정한 뒤 한꺼번에 만든다. 하나라도 자리가 없으면 이벤트 전체가 보류되어 일부만 나타나는 일이 없다.
- 병종·레벨·방향·장비·특성·진영색 덮어쓰기는 일반 스폰과 같다(`ValidateSpawn`, [data-scenario-node.md](data-scenario-node.md)).
- 병사 템플릿은 번호가 붙은 임시 인물로, 이름 있는 장수는 자기 ID 그대로 만들어진다. 둘 다 임시 인물(`Temp`)이라 전투가 끝나면 사라진다.
- 유닛마다 `unit-arrival` 이벤트가 나오고, 그 `Unit`에 스냅샷이 실린다.

### 5.3 대사(`Dialogue`)

`Dialogue`는 대사 묶음의 순서 있는 배열이다. 묶음마다 `speech` 이벤트 하나가 나가고, 같은 묶음의 대사는 전장 말풍선으로 동시에 뜬다(표시는 연출의 몫).

- 대사 `Actor`가 전장에 없거나 쓰러져 있으면 그 줄만 빠진다. 묶음의 줄이 모두 빠지면 그 묶음은 이벤트를 내지 않는다.
- 증원 유닛이 만들어진 뒤에 대사를 걸러내므로, 증원이 합류하며 하는 대사가 가능하다.
- 증원이 없는 이벤트(`duel` 트리거 대사 등)도 같은 방식으로 대사만 낸다.

### 5.4 지킴과 AI 표시

- `Spawn.Guard`: 유닛이 자기 칸에서 이동하지 못한다(이동 범위가 비고 이동 명령이 `OutOfRange`). 공격은 사거리 안이면 한다.
- `Spawn.AI`: 아군 스폰만 의미가 있고 `support` 차례에 자동 행동한다.

## 6. 일기토 발동

일기토는 `Stage.Duels`에 적은 `Ally`/`Enemy` 쌍이 서로를 근접(맨해튼 거리 1)에서 물리 기술로 공격할 때 일어난다. 공격 방향은 어느 쪽이든 되고, 일기토가 그 공격을 대신한다. 조건은 다음과 같다.

- 두 유닛 모두 살아 있고 진영이 다르다.
- 공격이 물리(`physical`) 종류다.
- 아직 치르지 않았다(`Executed["duel-<ID>"]`).

일기토를 치르면 위 `Executed` 키가 참이 되고, `Duel`의 결과(`AllyResult`·`EnemyResult`)가 곧바로 적용되며(`stay`, `retreat`, `death`), `duel` 이벤트가 나간다. 이벤트의 `Actor`는 항상 아군 쪽이다. 이어지는 퇴각 후속 효과, 경험치, `AttackDown`/`DefenseDown` 디버프 같은 규칙은 [logic-duel.md](logic-duel.md)에 있다. `duel` 트리거 이벤트는 이 일기토가 끝난 같은 `settle()`에서 발동한다.

## 7. 저장과 복원

- `Executed`는 상태의 일부라 저장·불러오기를 거쳐도 유지된다. 이미 발동한 이벤트는 다시 발동하지 않고, 치른 일기토는 다시 일어나지 않는다.
- 이벤트로 나타난 유닛은 일반 유닛과 같이 저장된다. 불러올 때 `Restore`는 체크포인트에서 스테이지의 `Party`·`Enemies` 스폰을 다시 만들어 비교 기준으로 삼고, 전투 단계에서는 `Executed`에 이미 있는 이벤트의 `Spawns`도 기준에 더한다. 저장된 유닛이 이 기준과 `AI`/`Guard` 표시가 다르거나, `Guard` 유닛이 스폰 칸을 벗어났거나, 표시 없는 유닛이 `AI`/`Guard`를 가지면 `CorruptSave`로 거부한다.
- 기준 유닛은 모두 저장 상태에 있어야 하고 진영도 같아야 한다.
- 패배 후 재도전은 출전 준비 체크포인트로 돌아가므로 `battle-`·`duel-`·`reward-` 키가 전투 이전 상태로 돌아가 이벤트가 다시 발동 가능해진다([logic-campaign.md](logic-campaign.md)).

## 8. 현재 데이터의 사용처

`Events`는 B03(호로관)만 쓴다. `boss-alone`으로 연합군 증원과 대사, 증원 장수가 치르는 일기토 두 개 중 하나라도 끝나면 `duel` 트리거로 형제의 대사가 나온다. B01·B02는 이벤트가 없고 B01은 `Goal: "rout"`, B02·B03은 보스 승리다. 실제 값은 `assets/data/maps/B03.json`의 `Stage.Events`를 본다.
