# 캠페인 진행

범위: 시나리오 노드를 따라가는 캠페인 흐름, 출전 준비·부관·배치, 전투 결과 이후의 이어하기, 재도전 체크포인트, 인벤토리·창고, 보상, 자동 저장 시점.
기준은 `internal/core/engine.go`·`deployment.go`·`formation.go`·`api.go`, `internal/session/session.go`, `assets/scenario/<시나리오>/scenario.json`(`Campaign`·`Chapters`)과 장 파일(`storyNN.json`)이며, 값이 다르면 코드와 이 데이터를 따른다.

노드·스테이지의 JSON 필드는 [data-scenario-node.md](data-scenario-node.md), 전투 중 이벤트와 승패는 [logic-battle-event.md](logic-battle-event.md), 아이템·장비는 [data-item.md](data-item.md), 저장 파일 형식은 [system-save.md](system-save.md), 성장(경험치·특성 학습)은 [logic-growth.md](logic-growth.md)를 본다.
아직 구현하지 않은 진행 구조(회의장, 장면 노드 등)는 [todo/engine.md](todo/engine.md)에 있다.

## 1. 단계

상태의 `Phase`는 다음 다섯 값이다.

| Phase | 의미 |
|---|---|
| `scenario` | 현재 `Node`(대화·선택·출전 준비)를 진행한다. |
| `deployment` | 전투 지도에서 출전 장수를 배치한다. |
| `battle` | 진영 차례 전투. |
| `result` | 전투 결과(`Result`가 `victory` 또는 `defeat`)를 보여 준다. |
| `complete` | 캠페인이 끝났다. |

새 게임은 `Campaign.Start` 노드에서 `scenario` 단계로 시작한다. 히어로 시나리오는 유비만 로스터로 시작하고, 관우·장비는 1장 도원결의 이벤트 끝의 `join` 노드로 합류한다. 시작 노드가 이야기의 뒤쪽이면 그 앞에서 합류하는 인물(걷기 이벤트의 `join` 포함)을 로스터에 미리 넣는다(`joinBefore`). 시작 로스터(`Campaign.Roster`)는 모두 인물로 만들어지고, 시작 출전 명단은 `Campaign.Deployment`, 시작 인벤토리는 `Campaign.Inventory`, 시작 창고는 `Campaign.Warehouse`다. 값은 시나리오 파일(`assets/scenario/<시나리오>/scenario.json`)을 본다.

**장면 이동**(시작 메뉴, 디버그 모드가 아니어도 나온다. 세션 `jump` 요청에 `node`·`difficulty`·`flags`·`level`)은 지점(`Data.Waypoints`: 각 장의 첫 노드와 각 `battle` 노드)을 고르면 새로 시작과 같은 난이도 창을 띄우고, 고른 난이도로 그 지점에서 새 게임을 연다(`core.NewAtWith`). 창 왼쪽이 지점 목록, 오른쪽이 설정이다. 설정은 둘이다.

- **아군 레벨**(`core.Start.Level`): 기본은 다음 전투의 단계 레벨. 값을 정하면 모든 장수를 그 레벨까지 올린다(승급·특성치 포함. 이미 더 높은 장수는 내리지 않는다).
- **앞선 전투 결과**(`core.Start.Flags`): 전투가 뒤에 남기는 플래그(`Data.StoryFlags`: 부목표의 `Flag`. 줄마다 전투·부목표와 그 플래그가 뒤에 바꾸는 일 `Later`를 보인다) 가운데 켠 것만 값 1로 적는다. 고른 지점보다 뒤 전투의 플래그는 쓰이지 않는다. 켜지 않으면 부목표 없이 이긴 것으로 친다.

지점 앞의 이야기는 분기를 따라가며 지나간다: `branch`는 그때의 플래그·파티로 고른 갈래로, `Next`가 뒤쪽을 가리키면 그 사이 노드는 건너뛴다(`choice`·`end`와 뒤로 가는 `Next`는 다음 노드로 읽어 나간다). 그래서 고순은 "고순이 끝까지 살아남는다"를 켰을 때만 합류해 있다. 고른 지점이 안 가는 갈래 위에 있어도 그 지점에서 시작한다.

- 그 앞의 이야기는 기본 순서대로 지나간 것으로 친다: 장 노드를 순서대로 보며 `set`의 플래그, `join`·`grant`(한 번 실행 키 포함), `explore`를 지나면 그 장의 `Events` 안 `set`·`grant`, `battle`을 지나면 승리(`won-`)와 승리 보상(`reward-`)을 적는다. 분기·선택·부목표는 기본 길(조건 거짓)이다.
- 모든 인물을 다음 전투의 스테이지 레벨(`StageLevel(NextBattleOrder(지점))`, 정규 난이도에서 1 + round(1.5 × (전투 순번 − 1)))까지 올리고, 그 레벨에 맞춰 자기 승급 길을 따른다(`joinAt`). 늘어난 특성치로 배울 것이 있으면 학습 창을 연다.
- 인벤토리는 모든 아이템을 9개 이상, 창고는 아무도 차지 않은 모든 장비를 1개 이상 갖는다. `Owner`가 있는 장비 가운데 그 지점까지 이야기가 준 것(채우기 전에 창고에 든 것)만 주인에게 채운다(`Engine.OwnerPiece`, 그 병종이 못 드는 장비는 건너뜀). 아직 받지 않은 주인 장비(2장 화계서를 1·2장에서)는 창고에 남는다.

## 2. 노드 종류

노드는 `Kind`로 동작이 갈린다. 노드로 들어올 때(`enter`) 엔진이 하는 일은 다음과 같다.

| Kind | 들어올 때 | 넘어가는 법 |
|---|---|---|
| `say` | 멈춘다. | `next` 명령으로 `Next`(없으면 기본 순서)로 간다. `Speaker`(인물 ID, 여럿이면 `·`)의 한 줄. |
| `narrate` | 멈춘다. | `next`. 화자 없는 해설 한 줄. |
| `choice` | 멈춘다. | `choose` 명령에 `Option`(선택지 이름)을 주면 `Choices`가 가리키는 노드로 가고, 고른 선택지를 `Choices[노드]`에 기록한다. 없는 이름이면 `InvalidOption`. |
| `battle` | `Stage`에 적은 스테이지를 현재 스테이지로 정하고 멈춘다. | 출전 준비 명령(3절)과 `start`. 이긴 뒤에는 이 노드의 `Next`(기본 순서)로 간다. |
| `scene` | `Place`를 `State.Place`에 적는다. | 멈추지 않고 이어진다. |
| `set` | `Flags`의 값을 `State.Flags`에 적는다. | 멈추지 않고 이어진다. |
| `branch` | `Cases`를 순서대로 보아 `If` 조건이 모두 참인 첫 칸의 `Next`로 가고, 없으면 `Else`, 그것도 없으면 기본 순서. | 멈추지 않는다. |
| `end` | 캠페인을 끝낸다(`complete`). | — |
| `join` | 처음 한 번만 `Character`를 인물로 만들고 출전 명단 끝에 더한다. | 멈추지 않고 `Next`로 이어진다. |
| `grant` | 처음 한 번만 `Item`을 `Count`개 준다. | 멈추지 않고 `Next`로 이어진다. |

- `join`과 `grant`는 `Event` 키(`Executed`)로 한 번만 실행된다. 이미 실행된 키면 건너뛰고 `Next`로 간다. 실행하면 `join` 또는 `grant` 이벤트가 나간다.
- 분기 조건 문자열은 `flag:키`(값 1 이상), `flag:키>=N`, `alive:인물`, `dead:인물`, `party:인물`(살아 있는 플레이어블 로스터), `choice:노드=옵션`, `won:스테이지`이고 앞에 `!`를 붙이면 부정이다. 모두 참이어야 참(AND)이다.
- 멈추지 않는 노드로만 이루어진 고리(`branch` 포함)는 로드 오류다.
- `grant`의 `Item`이 장비 ID이면 창고(`Warehouse`)에, 아니면 인벤토리(`Inventory`)에 들어간다.
- `Next`가 비어 있으면 같은 장 파일의 다음 노드로, 장의 마지막이면 다음 장의 첫 노드로 간다. 마지막 장의 끝이면 더 갈 노드가 없다(`After`). `battle` 노드는 `Next`가 승리한 뒤의 행선지이고, 없으면 같은 규칙이다.
- 장 안의 노드 순서가 기본 진행 순서다. 분기(`choice`)나 `Next`가 가리키는 노드는 순서와 무관하다.

학습 대기열(`Learning`)에 장수가 있는 동안에는 `learn`·`learn_close` 외의 모든 명령이 `LearningPending`으로 거부된다. 대기열 밖에서도 `CanLearn`이 참이면(탐험·출전 준비) `learn`으로 바로 배운다.

## 3. 출전 준비

`battle` 노드에서만 다음 명령이 가능하다(그 밖에는 `WrongPhase`).

- `deploy`: 출전 명단을 정한다. 1명 이상 `Stage.Max`명 이하이고, `Stage.Required` 중 출전할 수 있는 인물을 모두 포함하고, 중복이 없고, 플레이어블 인물이며, 전사하지 않았고, 부관이 아니어야 한다. 군주(`Lord`) 인물이 로스터에 있으면 반드시 포함해야 한다. 어기면 `InvalidDeployment`.
- `equip`·`unequip`: 창고와 장수의 장비 칸 사이에서 장비를 옮긴다([data-item.md](data-item.md)).
- `deputy`·`undeputy`: 부관 임명·해임. 임명하면 부관의 장비가 창고로 가고, 부관은 출전 명단에서 빠진다. 부관의 효과는 [logic-traits.md](logic-traits.md)를 본다.
- `start`: 출전한다(4절).

## 4. 출전과 배치

`start`는 다음을 한 번에 한다.

1. 임시 인물(적, 병사, 아군 AI)을 모두 지운다. 이름 있는 인물의 상태는 유지한다.
2. 이 시점의 상태를 체크포인트로 저장한다(`State.Checkpoint`, 6절).
3. 단계를 `deployment`, 차례를 `ally`, 라운드를 1로 하고 결과를 지운다.
4. 스테이지의 `Party`(고정 아군)와 `Enemies`를 지도에 놓는다. `Party`에 이름 있는 장수가 있는데 그가 출전 명단에도 있으면 `InvalidDeployment`.

출전 명단의 장수는 아직 지도에 없다. 플레이어가 `Stage.Allies`의 칸에 한 명씩 놓는다.

`Stage.Forced`(강제 출전)이면 `start`가 이어서 `DeploymentPlan`대로 장수를 놓고 `deployment_finish`까지 처리한다. 출전 준비(장비)는 그대로 있고 배치 화면만 없다. 이미 전투가 시작된 뒤의 `DeploymentPlan`은 빈 명령열이다.

- `place`(`Actor`, `X`, `Y`): 출전 명단의 장수를 놓는다. 칸은 `Allies`에 있고, 비어 있고, 그 장수의 병종이 설 수 있는 지형(비용 > 0)이어야 한다. 스테이지가 출전 무리(`Groups`)를 두었으면 무리의 `Required` 장수는 그 무리 구역 안의 칸이어야 한다(`reposition`의 교환도 두 장수 모두 이 규칙을 지켜야 한다). 배치 화면은 그 장수가 설 수 있는 칸만 밝히고, 목록에 장수가 선 무리의 이름을 보인다. 아니면 `InvalidPlacement`. 이미 놓였으면 `AlreadyPlaced`. 놓인 장수는 병력 최대, 병법 0으로 시작한다.
- `unplace`: 놓은 장수를 거둔다. 고정 아군은 대상이 아니다(출전 명단에 없으므로 `InvalidActor`).
- `deployment_finish`: 출전 명단 전원이 놓였고 살아 있어야 한다(아니면 `DeploymentIncomplete`). 단계를 `battle`로 바꾸고 `deployment_finish` 이벤트를 낸다.
- `DeploymentPlan`: 남은 장수에게 칸을 최대한 겹치지 않게 배정하는 명령열을 만든다(자동 진행용). `Stage.Recommend`에 권장 칸이 있는 장수는 그 칸에 먼저 세우고(비어 있고 설 수 있을 때), 나머지는 `Allies` 순서대로 채운다. 무리의 필수 장수는 제 무리 칸 안에서만 맞춘다(먼저 자리를 차지한 다른 장수를 다른 칸으로 옮겨서라도). 자리가 모자라면 `NoDeploymentSpace`.

`Allies` 칸 수는 `Max` 이상이어야 로드된다([data-scenario-node.md](data-scenario-node.md)). 스테이지의 `Required`(필수 출전)에 든 인물은 출전 준비에 들어서면 출전 명단에 자동으로 들어가고(전사했거나 부관이면 건너뜀) 명단에서 뺄 수 없다. 합류한 장수가 `Max`보다 많으면 필수 인물과 군주를 남기고 레벨이 높은 순으로 `Max`명까지 기본 출전 명단을 채우며(`fitSortie`), 플레이어가 거기서 바꾼다. `deploy`·`start`는 명단이 `Max`를 넘거나 필수 인물이 빠지면 `InvalidDeployment`다. 스테이지의 `Reserve`(복병, [data-scenario-node.md](data-scenario-node.md) 6.2절)에 든 인물도 출전 준비에 들어서면 명단에 자동으로 들어가고(로스터에 없거나 전사했거나 부관이면 건너뜀) 뺄 수 없지만 `Max`에는 세지 않는다. `deploy`는 복병을 빼고 보내도 엔진이 다시 붙이고, 복병은 `place`(`InvalidPlacement`)·`DeploymentPlan`·`placements`·`deploymentReady`에서 제외되어 처음 맵에 서지 않는다. 이벤트 `join`이 부르면 출전 준비 그대로(레벨·병종·장비) 나타난다.

## 5. 전투 뒤

- 승리하면 `Phase`가 `result`가 되고 `continue` 명령을 기다린다. `continue`는 전투 상태(유닛, 임시 인물, 체크포인트, 학습 창, 라운드)를 지운다. 그다음 출전 노드(`battle`)의 다음 노드(`After`)가 없으면 `complete` 단계가 되고, 있으면 `scenario` 단계에서 그 노드로 가서 `enter`를 돌린다. 이때 전사한 인물은 출전 명단에서 빠진다.
- 패배하면 `Phase`가 `result`이고 `continue`는 거부된다. `Retry`만 가능하다(6절).
- 전사(`Dead`)는 일기토 결과 `death`로만 생긴다([logic-duel.md](logic-duel.md)). 전사한 장수의 장비는 창고로 돌아가고 부관은 풀린다. 전투에서 병력이 0이 된 일반 퇴각은 전사가 아니라 다음 출전에 다시 나온다.
- 승리 보상(`Stage.Reward`)은 인벤토리에 더해진다. 7절을 본다.

현재 데이터의 흐름은 시나리오 파일의 `Chapters`와 장 파일의 노드를 따라가면 알 수 있다. 마지막 장의 마지막 노드 뒤에서 캠페인이 끝난다.

## 6. 재도전 체크포인트

`start` 때 저장한 체크포인트는 `scenario` 단계, 출전 준비 노드, 같은 스테이지 상태다. 인벤토리, 창고, 장비, 부관, 출전 명단, `Executed`가 모두 출전 직전 값이다.

- `Retry`는 `Phase == "result"`, `Result == "defeat"`, 체크포인트가 있을 때만 된다(아니면 `WrongPhase`). 체크포인트를 복원하고 `Revision`만 현재보다 1 늘린다.
- 복원되면 출전 준비 화면으로 돌아가므로 편성·장비를 바꿔 다시 출전할 수 있다. 그 전투에서 번 경험치와 쓴 아이템, 일어난 이벤트·일기토는 모두 되돌아간다.
- 체크포인트는 중첩되지 않는다. 체크포인트 안의 체크포인트는 `CorruptCheckpoint`로 거부된다.
- 세션에서는 `retry` 명령이 `StaleRevision` 검사를 거치고 상태를 변경됨(`Dirty`)으로 표시한다. 재도전 자체는 자동 저장을 쓰지 않는다.

## 7. 인벤토리와 창고

두 주머니는 서로 다른 종류를 담는다.

- `Inventory`: 아이템(`items.json`) ID별 개수. 아군이 쓰는 소모품과 인수는 모두 이 공용 주머니에서 쓴다. 적과 동맹군(`AI` 아군)은 자기 `Items`를 따로 가진다(`Rules.EnemyStock`, `Rules.AllyStock`).
- `Warehouse`: 장비(`equipment.json`) ID별 개수. 장착하지 않은 장비를 둔다.

보상과 부여는 다음과 같이 들어간다.

- `Stage.Reward`는 모두 `Inventory`에 더해진다. 키는 아이템 ID여야 하며, 로드 검증은 이 키를 검사하지 않지만 장비 ID 등 잘못된 키가 들어가면 그 상태를 `Restore`할 때 `CorruptSave`가 된다.
- `grant` 노드는 2절대로 아이템·장비에 따라 주머니를 나눈다.
- 보상은 승리한 스테이지당 한 번이다(`Executed["reward-<스테이지ID>"]`).

`Restore`는 `Inventory`의 키가 `Items`에, `Warehouse`의 키가 `Equipment`에 있고 개수가 0 이상인지 확인한다.

## 8. 자동 저장

세션(`internal/session`)이 탐험의 `sortie` 명령이 성공하면 그 명령 직전 상태를 자동 저장한다. 또 `Apply`의 결과 이벤트를 보고 자동 저장한다. `deployment_finish` 이벤트 또는 `turn` 이벤트 중 아군 차례 시작(`Target == "ally"`)이 있고, 그때 `Phase`가 `battle`일 때 한 번 저장한다. 이동·공격·지원·적군 차례에는 저장하지 않는다.

시나리오는 `auto-scenario1`~`auto-scenario10`, 전투는 `auto-battle1`~`auto-battle10`을 각자 순환한다. 다음 슬롯은 해당 종류의 파일 순번(`sequence`) 최댓값으로 정하므로 게임을 다시 실행하거나 옛 저장을 불러와도 순서가 이어진다. 수동 슬롯 `1`~`10`과 독립이다. 쓰기에 실패하면 `autosave-failed` 이벤트가 결과에 덧붙고 회전은 변하지 않는다. 파일 형식과 불러오기는 [system-save.md](system-save.md)가 기준이다.
