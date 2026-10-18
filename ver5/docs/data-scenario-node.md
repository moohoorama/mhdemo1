# 시나리오·스테이지 자료 형식

범위: 시나리오 폴더(`assets/scenario/`)의 시나리오 파일(`Campaign`·`Chapters`), 장 파일의 노드, 스테이지 파일의 `Stage`(스폰·이벤트·일기토 기록)의 JSON 필드와 로드 검증 규칙.
기준은 `internal/content/content.go`(구조체), `internal/content/scenario.go`(로더), `internal/content/validate.go`·`battle_events.go`·`spawn.go`(검증)이고, 값이 다르면 코드와 `assets/scenario/`를 따른다.

동작은 [logic-campaign.md](logic-campaign.md)(노드·배치·재도전), [logic-battle-event.md](logic-battle-event.md)(이벤트·승패), [logic-duel.md](logic-duel.md)(일기토)가 설명한다.
맵 문서(`maps/mapNN.json`)의 지형·그림 영역·좌표계는 [data-map.md](data-map.md)를 본다. 이 문서는 스테이지 파일의 `Stage` 객체만 다룬다.

JSON은 `DisallowUnknownFields`로 읽으므로 이 문서에 없는 키는 로드 오류다. 필드 이름은 파스칼 케이스다. 인물·아이템·병종 이름은 `assets/data/README.md`의 ID 규칙(영어 스네이크 케이스)을 따르고, 한글은 `Name`·`Desc`·`Text`와 선택지 이름에만 쓴다.

## 0. 파일 구성

```
assets/data/                 공용 데이터 묶음(인물·병종·아이템·규칙…)
assets/scenario/
  scenario.json              {"Scenarios":[{"ID","Name","Path"}]} — 게임은 첫 항목을 연다
  <시나리오>/
    scenario.json            시작 상태(Campaign 필드)와 Chapters
    storyNN.json             장 하나의 노드 배열 {"Nodes":[…]}
    stageNN.json             스테이지 규칙·배치 (Map이 맵 파일을 가리킴, 없으면 Tiles를 직접 가짐)
    maps/mapNN.json          지형 문서
```

- 로더(`content.Load(<data 폴더>)`)는 데이터 묶음과, 그 옆(`../scenario`) 또는 안(`scenario/`)의 시나리오 폴더를 함께 읽는다. 해시에는 두 쪽의 모든 파일이 들어간다.
- 장(`Chapters`)은 `{ID, File}`의 배열이고 순서가 진행 순서다. 장 파일의 노드는 파일 안 배열 순서가 의미를 가진다.
- 스테이지 파일은 `stage*.json`이면 모두 읽는다. 스테이지 ID는 시나리오를 통틀어 유일해야 한다.

## 1. 캠페인(시나리오 파일)

| 필드 | 뜻 |
|---|---|
| `Start` | 첫 노드 ID. 노드에 있어야 한다. |
| `Roster` | 처음부터 가진 인물 ID. 모두 `Playable`이어야 한다. |
| `Deployment` | 처음 출전 명단. 모두 `Playable`이어야 한다. |
| `Inventory` | 처음 인벤토리. 키는 `items.json`에 있는 아이템, 개수는 0 이상. |
| `Chapters` | `[{"ID","File"}]`. 장 ID는 유일, 파일은 비어 있지 않은 노드 배열. |

## 2. 노드(장 파일)

파일은 `{"Nodes": [...]}` 하나다. 모든 노드는 같은 필드 집합을 쓰고, `Kind`에 따라 일부만 의미가 있다.

| 필드 | 쓰는 Kind | 뜻 |
|---|---|---|
| `ID` | 모두 | 비어 있지 않고 유일해야 한다. |
| `Kind` | 모두 | `say` `narrate` `choice` `battle` `scene` `set` `branch` `join` `grant` `end` 중 하나. 다른 값은 로드 오류. |
| `Text` | `say` `narrate` `choice` `battle` | 화면에 보이는 글. `say`·`narrate`는 비어 있을 수 없다. |
| `Speaker` | `say` | 말하는 인물 ID. 여럿이 함께 말하면 `·`로 잇는다. 모두 인물이어야 한다. `narrate`에는 쓰지 않는다. |
| `Place` | `scene` | 이야기가 지금 있는 장소. 비어 있을 수 없고 화면 배경 위에 표시된다. |
| `Next` | `say` `narrate` `battle` `scene` `set` `join` `grant` | 다음 노드 ID. 비어 있지 않으면 노드에 있어야 한다. 비어 있으면 아래의 기본 순서를 따른다. |
| `Choices` | `choice` | 선택지 이름 → 노드 ID. 가리키는 노드가 모두 있어야 한다. |
| `Cases` | `branch` | `[{If, Next}]`. `If`는 조건 문자열 배열(비어 있을 수 없음), `Next`는 있는 노드. 위에서부터 보아 처음 참인 칸으로 간다. |
| `Else` | `branch` | 어느 칸도 참이 아닐 때 갈 노드(없으면 기본 순서). |
| `Flags` | `set` | 플래그 이름 → 값(정수). 비어 있을 수 없다. |
| `Stage` | `battle` | 출전할 스테이지 ID. 스테이지에 있어야 한다. |
| `Character` | `join` | 합류하는 인물 ID. `Playable`이어야 한다. |
| `Item` | `grant` | 주는 아이템 또는 장비 ID. 둘 중 하나에 있어야 한다. |
| `Count` | `grant` | 개수. |
| `Event` | `join`, `grant` | 한 번만 실행하기 위한 `Executed` 키. 규칙은 `join-<인물ID>`, `grant-<아이템ID>`. |

조건 문자열(`If`): `flag:키`(값이 1 이상) `flag:키>=N` `alive:인물` `dead:인물` `party:인물` `choice:노드=옵션` `won:스테이지`, 앞에 `!`을 붙이면 부정이다. 참조는 모두 로드 때 검증한다. 멈추지 않는 노드(`scene` `set` `branch` `join` `grant`)만으로 고리를 만들면 로드 오류다.

다음 노드는 노드의 `Next`가 정하고, 비어 있으면 같은 장 파일의 다음 노드, 장의 마지막이면 다음 장의 첫 노드, 마지막 장의 끝이면 캠페인이 끝난다. `battle` 노드의 `Next`는 그 스테이지에서 이긴 뒤의 행선지다.

```json
{"ID": "oath_choice", "Kind": "choice", "Text": "도원결의: 의용군을 일으킨다.", "Choices": {"결의": "prep1"}}
{"ID": "prep1", "Kind": "battle", "Text": "황건적의 난 — 출전 준비", "Stage": "B01"}
{"ID": "oath", "Kind": "say", "Speaker": "guanyu·zhangfei", "Text": "생사를 함께하겠습니다.", "Next": "oath_choice"}
{"ID": "m1", "Kind": "set", "Flags": {"flag-oath-sworn": 1}}
{"ID": "m2", "Kind": "branch", "Cases": [{"If": ["flag-oath-sworn"], "Next": "m3"}], "Else": "m4"}
{"ID": "join", "Kind": "join", "Event": "join-jianyong", "Next": "joined", "Character": "jianyong"}
{"ID": "sword", "Kind": "grant", "Item": "gujeong_blade", "Count": 1, "Event": "grant-gujeong_blade", "Next": "chase"}
```

## 3. 스테이지(`stageNN.json`)

`Map`(맵 파일 이름, `maps/` 아래)이 있으면 `Width`·`Height`·`Tiles`는 맵 문서에서 만들어지는 값이라 적지 않는다. `Map`이 없는 스테이지(테스트 픽스처)는 `Tiles`를 직접 적는다. 아래 좌표는 모두 맵 문서(월드) 좌표이고, 로드할 때 전투 영역 원점을 빼서 전투 좌표로 바뀐다([data-map.md](data-map.md)).

| 필드 | 뜻 |
|---|---|
| `ID` | 스테이지 ID. 유일해야 한다. 노드 `Stage`가 이 값을 가리킨다. |
| `Name` | 표시 이름. |
| `Max` | 출전 인원 상한. 1 이상이고 `Allies`(배치 칸)가 이 값 이상이어야 한다. |
| `Required` | 반드시 출전해야 하는 인물 ID(선택). 플레이어블이고 중복이 없어야 하며 `Max` 이하여야 한다. |
| `Victory` | 승리 목표 `[{If, Text}]`. 하나라도 있어야 하고 `Text`는 필수. 하나의 `If`가 모두 참이면 승리([logic-battle-event.md](logic-battle-event.md) 3절). |
| `Defeat` | 추가 패배 목표 `[{If, Text}]`(선택, `Text`는 선택). 군주의 퇴각과 아군 전멸은 늘 패배다. |
| `Sub` | 부목표 `[{ID, Text, If, Flag, Reward}]`(선택). `ID`는 유일, `Text`·`If` 필수, `Reward`는 아이템 ID → 1 이상 개수. |
| `TurnLimit` | 0 이상. 0이면 무제한([logic-battle-event.md](logic-battle-event.md)). |
| `Allies` | 출전 장수를 놓을 수 있는 칸 `{X, Y}` 목록. |
| `Enemies` | 적 스폰 목록. 비어 있을 수 없다(맵 문서 검증). |
| `Party` | 처음부터 지도에 있는 고정 아군 스폰 목록(선택). |
| `Reward` | 승리 보상. 아이템 ID → 개수. 인벤토리로 들어간다. |
| `Duels` | 일기토 목록(5절). |
| `Areas` | 번호 붙은 구역 목록(6절). |
| `Events` | 전투 이벤트 목록(7절). |
| `Faction` | 자기 `Faction`이 없는 적 스폰의 진영색. `Factions`에 있어야 한다. |

검증은 다음과 같다.

- 스폰(`Enemies`, `Party`)은 4절의 스폰 규칙을 통과해야 한다. 이름 있는 인물(템플릿이 아닌)은 두 목록을 통틀어 한 번만 나올 수 있고, 인물이 정의에 있어야 하며, 좌표가 영역 안이고, 다른 스폰·`Allies` 칸과 겹치지 않아야 한다.
- `Allies`의 칸은 영역 안이어야 하고 서로 또는 스폰과 겹치지 않아야 한다. 맵 문서 검증은 로스터의 병종 중 하나는 설 수 있는 지형이어야 한다고 더 요구한다.
- `Victory`·`Defeat`·`Sub`의 조건은 전투 이벤트 조건(logic-battle-event.md 5.1절)이고 참조(유닛·그룹·구역·일기토)는 로드 때 검증한다. 대상 `enemy`·`ally`는 그 진영 전체이고, 스폰의 `Group` 이름으로 `ally`·`enemy`·`support`는 쓸 수 없다.

```json
{"ID": "B02", "Name": "사수관 전투", "Max": 4, "TurnLimit": 30,
 "Victory": [{"If": ["defeated:huaxiong"], "Text": "화웅 격파"}],
 "Sub": [{"ID": "duels", "Text": "두 일기토에서 모두 이긴다", "If": ["duel:a", "duel:b"], "Flag": "flag-b02-duels", "Reward": {"liquor": 1}}],
 "Reward": {"bean": 8, "liquor": 4}, "Faction": "dong", "Map": "map02.json"}
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
| `Policy` | 행동 정책 `hold` `assault` `march` `flee`(선택). 뜻은 logic-battle-event.md 5.4절. |
| `Goal` | `march`·`flee`의 목표 칸 `{X, Y}`(맵 문서 좌표, 로드할 때 원점을 뺀다). 다른 정책에는 쓰지 않는다. |
| `Group` | 같은 이름을 가진 스폰을 이벤트 조건·`remove`가 한꺼번에 가리킨다(이름 있는 인물은 인물 ID로도 가리킬 수 있다). |
| `Side` | 이벤트가 만드는 스폰만: `ally`(기본, AI 아군) 또는 `enemy`. |
| `DuelFirst` | AI가 아직 안 치른 일기토 상대를 먼저 찾는다. 이동 범위 안에 상대와 맞닿을 칸이 있으면 그 칸으로 먼저 가고, 상대를 맞닿기 전에는 멀리서 쏘거나 치지 않는다. |

`Traits`·`Equipment` 검증은 다음과 같다. 특성은 모두 정의에 있어야 한다. 장비는 모두 정의에 있고, 같은 칸(`Slot`)에 둘 이상 없으며, 무기·방어구는 결과 병종의 `Weapon`·`Armor` 형식과 맞아야 한다. 덮어쓴 특성이 `Requires`로 요구하는 선행 특성은 (덮어쓴 특성 + 장비가 주는 특성) 안에 있어야 한다. 병력·병법 최대치는 엔진이 계산하며 저장하지 않는다.

```json
{"Character": "mushun", "X": 24, "Y": 7, "Level": 1, "AI": true, "Faction": "yuan", "Facing": "NE"}
{"Character": "troop_footman", "X": 23, "Y": 7, "Level": 1, "Class": "light_infantry", "AI": true, "Faction": "yuan", "Facing": "NE"}
```

## 5. 일기토 기록(`Stage.Duels`)

| 필드 | 뜻 |
|---|---|
| `ID` | 비어 있지 않고 모든 스테이지를 통틀어 유일. `Executed["duel-<ID>"]`의 키가 된다. |
| `Ally` | 아군 인물 ID. `Playable`이거나 `Party`에 있거나 이 스테이지 이벤트가 만드는 아군이어야 한다. |
| `Enemy` | 적 인물 ID. `Enemies`에 있거나 이 스테이지 이벤트가 `Side: "enemy"`로 만드는 인물이어야 한다. |
| `Outcome` | 아군 기준 결과 표시. `victory`, `defeat`, `draw` 중 하나. |
| `AllyResult`, `EnemyResult` | 각 쪽의 결과. `stay`, `retreat`, `death` 중 하나. |
| `Text` | 일기토 로그에 남는 글. |
| `Supports` | 연출에서 도중에 합류하는 두 장수의 인물 ID(선택). 둘이어야 한다. |
| `Beats` | 연출 장면 목록(선택). 없으면 기본 타임라인이 `Outcome`대로 보인다. |
| `Variants` | `[{First, Beats}]`(선택). `First`(먼저 이동해 맞닿은 장수)에 따라 `Beats` 대신 쓰는 연출. `First`는 유일하고 비어 있지 않다. |

`Beats`의 장면은 `Kind`(동작)·`Title`(쓰는 사람을 위한 메모)·`Speech`(대사)·`Side`(0 아군, 1 적, 2·3 지원군)·`Duration`(초)이다. 동작 어휘와 규칙은 `internal/duel`의 `Script.Validate`가 로드 때 검증한다(GUI가 자산을 읽을 때, 그리고 테스트에서). 마지막 장면이 `Outcome`과 어긋나면 오류다. 연출은 규칙 결과를 바꾸지 않는다([presentation-duel.md](presentation-duel.md)).

```json
{"ID": "hulao-mushun", "Ally": "mushun", "Enemy": "lubu", "Outcome": "defeat", "AllyResult": "death",
 "EnemyResult": "stay", "Text": "목순이 여포에게 패하고 전사했다.",
 "Beats": [{"Kind": "talk", "Title": "도발", "Speech": "네가 여포냐?", "Side": 0, "Duration": 3.2}, {"Kind": "win", "Side": 1, "Duration": 1}]}
```

## 6. 구역(`Stage.Areas`)

| 필드 | 뜻 |
|---|---|
| `ID` | 1 이상의 정수, 스테이지 안에서 유일. 이벤트·조건이 이 번호로 가리킨다. |
| `Rect` | `{X, Y, W, H}` 맵 문서(월드) 좌표. 폭·높이는 1 이상이고 전투 영역과 조금이라도 겹쳐야 한다. 로드할 때 전투 영역 원점을 뺀다. |
| `Name` | 비어 있을 수 없다. 사람이 보는 이름. |
| `Description` | 사람이 보는 설명(선택). |

```json
{"ID": 3, "Rect": {"X": 20, "Y": 8, "W": 3, "H": 3}, "Name": "마을 입구", "Description": "방화병이 불을 붙이러 오는 점화 지점"}
```

## 7. 전투 이벤트 기록(`Stage.Events`)

| 필드 | 뜻 |
|---|---|
| `ID` | 비어 있지 않고 스테이지 안에서 유일. |
| `Repeat` | 참이면 조건이 거짓에서 참이 될 때마다 실행(선택). `spawn`은 쓸 수 없다. |
| `If` | 조건 문자열 배열(선택). 비어 있으면 전투가 시작되는 첫 `settle()`에 실행된다. 조건은 [logic-battle-event.md](logic-battle-event.md) 5.1. |
| `Do` | 동작 배열(비어 있을 수 없음). `say`(`Lines`) `spawn`(`Spawns`) `remove`(`Targets`) `join`(`Spawns`) `ai`(`Targets`·`Policy`·`Goal`) `terrain`(`Area`·`Tile`·`Turns`) `buff`(`Targets`·`Stat`·`Value`·`Turns`) `move`(`Moves`) `cutscene`(`Steps`) `duel`(`Duel`) `flag`(`Flags`). 모르는 `Kind`는 로드 오류. |

`spawn`의 스폰은 일반 스폰 규칙에 더해 다음을 요구한다. `Side`는 비어 있거나 `ally`/`enemy`이고, `ally`이면 `AI`가 참이며, 좌표가 영역 안이고, 인물이 `Playable`도 `Lord`도 아니고, 이름 있는 인물은 이 스테이지의 다른 스폰·이전 이벤트 스폰·플레이어블 인물과 겹치지 않는다. `Faction`은 있다면 `Factions`에 있어야 한다. 스폰의 `Group`은 조건과 `remove`가 가리킬 수 있는 이름이 된다.

`say` 검증은 다음과 같다. `Lines`가 비어 있을 수 없고, 줄의 `Text`가 비어 있을 수 없으며, 한 줄에 같은 `Actor`가 둘 있을 수 없다. `Actor`는 플레이어블 인물, 이 스테이지의 `Enemies`·`Party` 인물, 이 스테이지 이벤트가 만드는 인물 중 하나여야 한다. `remove`·`ai`의 `Targets`는 유닛 ID나 그룹이어야 하고, `ai`의 `Policy`는 필수이며 `Goal`은 정책에 맞아야 하고, `join`은 반복 이벤트에 쓸 수 없고 스폰이 `Playable`이며 군주가 아니고 `AI`·`Side`·`Policy`가 없고 맵 안이며 이 스테이지의 다른 스폰에 없는 인물이어야 한다. `buff`는 `Stat`이 `attack` `defense` `mind` `morale` `agility` `speed` `range` 중 하나, `Value`가 0이 아닌 -100~100, `Turns`가 0~99이고 대상이 있어야 한다. `move`의 `Moves`는 비어 있을 수 없고 `Unit`은 유닛·그룹, `To`는 맵 안이어야 한다. `cutscene`의 `Steps`는 비어 있지 않은 묶음들이고, 단계는 `move`(`Unit`·`To`) `play`(`Unit`·`Motion`) `face`(`Unit`·`Facing`: NE SE SW NW) `say`(`Unit`·`Text`) `wait`(`Seconds` 0 초과 30 이하)이며 `Unit`은 유닛·그룹이다. `duel`의 `Duel`은 이 스테이지의 일기토 ID여야 한다. `terrain`의 `Area`는 이 스테이지의 구역이고 `Tile`은 `game.json`의 `Tiles`에 있으며 `Turns`는 0 이상이어야 하고, `flag`의 `Flags`는 비어 있을 수 없다.

```json
{"ID": "brothers-challenge", "If": ["any:[duel:hulao-mushun, duel:hulao-wuanguo]"],
 "Do": [{"Kind": "say", "Lines": [{"Actor": "liubei", "Text": "인중여포! 역시 강하군."}]},
        {"Kind": "say", "Lines": [{"Actor": "zhangfei", "Text": "좋았어, 형!"}, {"Actor": "guanyu", "Text": "제가 상대해보겠습니다."}]}]}
{"ID": "coalition-arrival", "If": ["alive:lubu", "alldefeated:guard"],
 "Do": [{"Kind": "spawn", "Spawns": [{"Character": "mushun", "X": 24, "Y": 7, "AI": true, "DuelFirst": true, "Faction": "yuan"}]},
        {"Kind": "say", "Lines": [{"Actor": "mushun", "Text": "지금이 기회다!"}]}]}
