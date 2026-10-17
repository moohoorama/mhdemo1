# 시나리오·스테이지 자료 형식

범위: `script.json`의 노드, `game.json`의 `Campaign`, 맵 문서 안의 `Stage`(스폰·이벤트·일기토 기록)의 JSON 필드와 로드 검증 규칙.
기준은 `internal/content/content.go`(구조체), `internal/content/validate.go`·`battle_events.go`·`spawn.go`(검증)이고, 값이 다르면 코드와 `assets/data/`를 따른다.

동작은 [logic-campaign.md](logic-campaign.md)(노드·배치·재도전), [logic-battle-event.md](logic-battle-event.md)(이벤트·승패), [logic-duel.md](logic-duel.md)(일기토)가 설명한다.
맵 문서(`maps/<ID>.json`)의 지형·그림 영역·좌표계는 [data-map.md](data-map.md)를 본다. 이 문서는 그 안의 `Stage` 객체만 다룬다.

JSON은 `DisallowUnknownFields`로 읽으므로 이 문서에 없는 키는 로드 오류다. 필드 이름은 파스칼 케이스다. 인물·아이템·병종 이름은 `assets/data/README.md`의 ID 규칙(영어 스네이크 케이스)을 따르고, 한글은 `Name`·`Desc`·`Text`와 선택지 이름에만 쓴다.

## 1. 캠페인(`game.json`의 `Campaign`)

| 필드 | 뜻 |
|---|---|
| `Start` | 첫 노드 ID. 노드에 있어야 한다. |
| `Roster` | 처음부터 가진 인물 ID. 모두 `Playable`이어야 한다. |
| `Deployment` | 처음 출전 명단. 모두 `Playable`이어야 한다. |
| `Inventory` | 처음 인벤토리. 키는 `items.json`에 있는 아이템, 개수는 0 이상. |

값은 `assets/data/game.json`을 본다.

## 2. 노드(`script.json`)

파일은 `{"Nodes": [...]}` 하나다. 모든 노드는 같은 필드 집합을 쓰고, `Kind`에 따라 일부만 의미가 있다.

| 필드 | 쓰는 Kind | 뜻 |
|---|---|---|
| `ID` | 모두 | 비어 있지 않고 유일해야 한다. |
| `Kind` | 모두 | `dialogue`, `choice`, `preparation`, `join`, `grant` 중 하나. 다른 값은 로드 오류. |
| `Text` | `dialogue`, `choice`, `preparation` | 화면에 보이는 글. |
| `Next` | `dialogue`, `join`, `grant` | 다음 노드 ID. 비어 있지 않으면 노드에 있어야 한다. 기본값은 없다. |
| `Choices` | `choice` | 선택지 이름 → 노드 ID. 가리키는 노드가 모두 있어야 한다. |
| `Stage` | `preparation` | 출전할 스테이지 ID. 스테이지에 있어야 한다. |
| `Character` | `join` | 합류하는 인물 ID. `Playable`이어야 한다. |
| `Item` | `grant` | 주는 아이템 또는 장비 ID. 둘 중 하나에 있어야 한다. |
| `Count` | `grant` | 개수. |
| `Event` | `join`, `grant` | 한 번만 실행하기 위한 `Executed` 키. 규칙은 `join-<인물ID>`, `grant-<아이템ID>`. |

`preparation` 노드의 출전 뒤 행선지는 노드가 아니라 스테이지의 `Next`가 정한다.

```json
{"ID": "oath_choice", "Kind": "choice", "Text": "도원결의: 의용군을 일으킨다.", "Choices": {"결의": "prep1"}}
{"ID": "prep1", "Kind": "preparation", "Text": "황건적의 난 — 출전 준비", "Stage": "B01"}
{"ID": "join", "Kind": "join", "Event": "join-jianyong", "Next": "joined", "Character": "jianyong"}
{"ID": "sword", "Kind": "grant", "Item": "gujeong_blade", "Count": 1, "Event": "grant-gujeong_blade", "Next": "chase"}
```

## 3. 스테이지(`maps/<ID>.json`의 `Stage`)

`Width`·`Height`·`Tiles`는 맵 문서에서 만들어지는 값이라 `Stage` 객체에는 적지 않는다. 아래 좌표는 모두 맵 문서(월드) 좌표이고, 로드할 때 전투 영역 원점을 빼서 전투 좌표로 바뀐다([data-map.md](data-map.md)).

| 필드 | 뜻 |
|---|---|
| `ID` | 스테이지 ID. 유일해야 한다. 노드 `Stage`가 이 값을 가리킨다. |
| `Name` | 표시 이름. |
| `Boss` | 보스 인물 ID. `Goal`이 `rout`이 아니면 `Enemies`나 `Party`에 그 인물의 스폰이 있어야 한다. |
| `Limit` | 출전 인원 상한. `Allies` 칸 수가 이 값 이상이어야 한다. |
| `Threshold` | 보스 승리 기준. 보스 병력이 최대의 이 퍼센트 이하이면 승리(0이면 퇴각·전사). |
| `TurnLimit` | 0 이상. 0이면 무제한([logic-battle-event.md](logic-battle-event.md)). |
| `Allies` | 출전 장수를 놓을 수 있는 칸 `{X, Y}` 목록. |
| `Enemies` | 적 스폰 목록. 비어 있을 수 없다(맵 문서 검증). |
| `Party` | 처음부터 지도에 있는 고정 아군 스폰 목록(선택). |
| `Reward` | 승리 보상. 아이템 ID → 개수. 인벤토리로 들어간다. |
| `Duels` | 일기토 목록(4절). |
| `Events` | 전투 이벤트 목록(5절). |
| `Goal` | `"rout"`이면 적 전멸이 승리 조건(선택). |
| `Charge` | 참이면 적이 첫 턴부터 아군 쪽으로 진군한다(선택). AI 동작은 `internal/ai/ai.go`가 기준이다. |
| `Next` | 승리 뒤 갈 노드 ID. 비어 있으면 캠페인이 끝난다. 값이 있으면 노드에 있어야 한다. |
| `Faction` | 자기 `Faction`이 없는 적 스폰의 진영색. `Factions`에 있어야 한다. |

검증은 다음과 같다.

- 스폰(`Enemies`, `Party`)은 4절의 스폰 규칙을 통과해야 한다. 이름 있는 인물(템플릿이 아닌)은 두 목록을 통틀어 한 번만 나올 수 있고, 인물이 정의에 있어야 하며, 좌표가 영역 안이고, 다른 스폰·`Allies` 칸과 겹치지 않아야 한다.
- `Allies`의 칸은 영역 안이어야 하고 서로 또는 스폰과 겹치지 않아야 한다. 맵 문서 검증은 로스터의 병종 중 하나는 설 수 있는 지형이어야 한다고 더 요구한다.
- 보스 승리 스테이지(`Goal`이 `rout`이 아님)는 `Boss`가 스폰에 있어야 하고, 맵 문서 검증은 보스가 `Enemies`에 있어야 한다고 요구한다.

```json
{"ID": "B02", "Name": "사수관 전투", "Boss": "huaxiong", "Limit": 4, "TurnLimit": 30, "Threshold": 0,
 "Reward": {"bean": 8, "liquor": 4}, "Next": "sword", "Faction": "dong"}
```

## 4. 스폰(`Spawn`)

`Enemies`, `Party`, 이벤트의 `Spawns`가 같은 형식이다.

| 필드 | 뜻 |
|---|---|
| `Character` | 인물 정의 ID(필수). 템플릿이면 임시 인물이 번호를 붙여 만들어진다. |
| `X`, `Y` | 칸 좌표(필수). |
| `Level` | 0이면 인물의 레벨, 아니면 덮어쓴다. 0 이상 `Rules.MaxLevel` 이하. |
| `Class` | 병종 덮어쓰기. 인물 병종과 같은 계열(`Class.Family`, 없으면 `Terrain`)이어야 한다. |
| `Facing` | 처음 방향. `NE`, `SE`, `SW`, `NW` 중 하나. |
| `HP`, `MP` | 처음 값. 생략하면 병력은 최대, 병법은 0. 최대치로 잘리고 `HP`는 1 이상, `MP`는 0 이상이어야 한다. |
| `Traits` | 특성 목록 덮어쓰기. 생략하면 인물 정의를 쓰고, 빈 배열이면 비운다. 병종·장비가 주는 특성은 그대로 붙는다. |
| `Equipment` | 장비 목록 덮어쓰기. 생략·빈 배열의 뜻은 `Traits`와 같다. |
| `Faction` | 진영색 덮어쓰기. `Factions`에 있어야 한다. 우선순위는 스폰 > 스테이지 `Faction`(적만) > 인물의 `Look.Faction`. |
| `AI` | 아군 스폰을 `support` 차례에 자동 행동하는 AI 유닛으로 만든다. |
| `Guard` | 제자리에서 움직이지 못한다. |

`Traits`·`Equipment` 검증은 다음과 같다. 특성은 모두 정의에 있어야 한다. 장비는 모두 정의에 있고, 같은 칸(`Slot`)에 둘 이상 없으며, 무기·방어구는 결과 병종의 `Weapon`·`Armor` 형식과 맞아야 한다. 덮어쓴 특성이 `Requires`로 요구하는 선행 특성은 (덮어쓴 특성 + 장비가 주는 특성) 안에 있어야 한다. 병력·병법 최대치는 엔진이 계산하며 저장하지 않는다.

```json
{"Character": "mushun", "X": 24, "Y": 7, "Level": 1, "AI": true, "Faction": "yuan", "Facing": "NE"}
{"Character": "troop_footman", "X": 23, "Y": 7, "Level": 1, "Class": "light_infantry", "AI": true, "Faction": "yuan", "Facing": "NE"}
```

## 5. 일기토 기록(`Stage.Duels`)

| 필드 | 뜻 |
|---|---|
| `ID` | 비어 있지 않고 모든 스테이지를 통틀어 유일. `Executed["duel-<ID>"]`의 키가 된다. |
| `Ally` | 아군 인물 ID. `Playable`이거나 이 스테이지 이벤트가 만드는 아군이어야 한다. |
| `Enemy` | 적 인물 ID. `Enemies`에 있어야 한다. |
| `Outcome` | 아군 기준 결과 표시. `victory`, `defeat`, `draw` 중 하나. |
| `AllyResult`, `EnemyResult` | 각 쪽의 결과. `stay`, `retreat`, `death` 중 하나. |
| `Text` | 일기토 로그에 남는 글. |
| `AttackDown`, `DefenseDown` | 0~90. 아군 승리(`Outcome: victory`) 때 모든 살아 있는 적의 공격·방어를 이 퍼센트만큼 전투 내내 깎는다(선택). |

연출 자료(`assets/duels.json`)와 이 기록은 별개다. 이 기록이 규칙 결과를 정한다([presentation-duel.md](presentation-duel.md)).

```json
{"ID": "hulao-mushun", "Ally": "mushun", "Enemy": "lubu", "Outcome": "defeat", "AllyResult": "death",
 "EnemyResult": "stay", "Text": "목순이 여포에게 패하고 전사했다."}
```

## 6. 전투 이벤트 기록(`Stage.Events`)

| 필드 | 뜻 |
|---|---|
| `ID` | 비어 있지 않고 스테이지 안에서 유일. |
| `Trigger` | `boss-alone` 또는 `duel`. |
| `Duels` | `duel` 트리거에서 필수. 이 스테이지 `Duels`에 있는 ID만 쓴다. |
| `Spawns` | 합류시킬 아군 AI 스폰 목록. |
| `Dialogue` | 대사 묶음의 배열. 묶음은 `{Actor, Text}` 줄의 배열. |

`Spawns` 검증은 일반 스폰 규칙에 더해 다음을 요구한다. `AI`가 참이고, 좌표가 영역 안이며, 인물이 `Playable`도 `Lord`도 아니고, 이름 있는 인물은 이 스테이지의 다른 스폰·이전 이벤트 스폰·플레이어블 인물과 겹치지 않는다. `Faction`은 있다면 `Factions`에 있어야 한다.

`Dialogue` 검증은 다음과 같다. 묶음이 비어 있을 수 없고, 줄의 `Text`가 비어 있을 수 없으며, 한 묶음에 같은 `Actor`가 둘 있을 수 없다. `Actor`는 플레이어블 인물, 이 스테이지의 `Enemies`·`Party` 인물, 이 스테이지 이벤트가 만드는 인물 중 하나여야 한다.

```json
{"ID": "brothers-challenge", "Trigger": "duel", "Duels": ["hulao-mushun", "hulao-wuanguo"],
 "Dialogue": [[{"Actor": "liubei", "Text": "인중여포! 역시 강하군. 병사들의 피해가 없도록 우리들이 일기토로 상대해야겠어."}],
              [{"Actor": "zhangfei", "Text": "좋았어, 형! 그럼 나에게 맡겨주슈!"}, {"Actor": "guanyu", "Text": "제가 상대해보겠습니다, 형님"}]]}
```
