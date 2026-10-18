# 해야 할 일: 엔진

범위: [conceptual_scenario.md](../conceptual_scenario.md) 1~20을 돌리기 위해 엔진에 더할 것(노드, 전투 이벤트, 승패, 군량 경제, 회의장). 아직 구현하지 않은 설계안이다.
현재 동작은 [logic-battle-event.md](../logic-battle-event.md), [logic-campaign.md](../logic-campaign.md), [data-scenario-node.md](../data-scenario-node.md)가 기준이다.
실제로 시작할 때는 `docs/temp/`에 plan을 쓰고, 구현이 끝난 부분은 위 문서에 현재형으로 옮긴 뒤 이 문서에서 지운다.

엔진(`internal/`)에는 인물·아이템 이름을 두지 않는다. 아래의 모든 어휘는 `assets/data/`의 데이터가 쓴다.

---

## 1. 진행 구조

```
[장면] → [장면] → … → [회의장] → [출전 편성] → [전투] → [전후 장면] → …
                        대화·매매·저장         필수/정원/증원
```

- 장면(`scene`/`say`/`narrate`): 플레이어는 대사를 넘기기만 한다. 전투 없는 스토리(2, 6, 12, 16, 24)는 스테이지가 아니라 장면 노드만으로 이루어진다.
- 회의장: 전투 직전마다 한 번. 대화, 군수물자 매매, 편성·장비, 저장, 출진(§4).
- 전투: 개전 연출 → 진영 페이즈 반복(전투 중 이벤트 최소) → 결과 → 다음 노드.
- 패배하면 출전 직전 체크포인트(플래그·군량 포함)로 돌아간다.

---

## 2. 데이터

### 2.1 노드 (장 파일)

구현된 노드(`say` `narrate` `scene` `set` `branch` `choice` `battle` `join` `grant` `end`, 조건 `flag` `alive` `dead` `party` `choice` `won`)와 플래그는 [data-scenario-node.md](../data-scenario-node.md)와 [logic-campaign.md](../logic-campaign.md)가 설명한다. 아직 없는 것:

| Kind | 필드 | 동작 | 대기 |
|---|---|---|---|
| `leave` | `Character`, `Event`, `Keep` | 이탈(12의 조운) | 아니오 |
| `camp` | `Camp`(회의장 ID) | 회의장을 연다. 출진하면 `Battle`로 간다 | 예 |

조건에 `food>=N`(군량)을 더한다. 장 사이의 분기(`Chapters`의 `Next` 조건)는 노드 조건 문법을 그대로 쓰기로 하고 아직 두지 않았다.

**플래그**: `State.Flags map[string]int`(구현됨), 키는 `flag-<주제>-<내용>`. 전투 중 정해진 플래그는 승리가 확정될 때 반영하고 패배하면 버린다(전투 중 `flag` 동작은 M3에서).
v1이 쓰는 플래그: 서복 생존, 거록 완승, 제갈현 생존, 서주 유민, 의대조 서명.

### 2.2 전투 이벤트와 트리거

구현됨: `If`/`Do` 형식, `Repeat`, 검사 시점(`settle()`마다, 최대 8번 연쇄), 조건 `round` `turnstart` `defeated` `alldefeated` `alive` `at` `allat` `adjacent` `hp` `duel` `flag` `choice` `won` `any`/`!`, 동작 `say` `spawn`(`Side`·`Group`) `remove` `flag`, 구역 `Areas`. 동작과 형식은 [logic-battle-event.md](../logic-battle-event.md) 5절과 [data-scenario-node.md](../data-scenario-node.md) 6~7절이 기준이다.

아직 없는 동작:

| Kind | 효과 | 쓰는 곳 |
|---|---|---|
| `join` | 조작 가능한 아군 편입 | 9·18의 조운 |
| `move` | 연출 이동(경로 비용 무시, 여러 부대 동시) | 3의 합류, 9의 백마의종 |
| `ai` | 정책 변경: `hold` `charge` `flee` `march`(`Goal` 지점으로) | 방화병, 피난민, 대군 |
| `defect` | 진영 전환 | 15의 송헌·후성 |
| `buff` | 버프·디버프 `Stat`, `Pct`, `Turns`(0=전투 끝까지) | 3의 손견군(HP 60%·이동력 저하) |
| `stall` | N턴 행동 불가 | 20의 허저 |
| `terrain` | `Area`의 타일을 `Tile`로 바꾼다(§2.6) | 1의 방화 |
| `objective` | 승패 조건 교체, 표시 문구 변경 | 17의 2단계 |
| `cutscene` | 단계 목록(`move`(동시 여럿) `play`(병종 동작) `face` `say` `wait`). 그룹 단위로 동시 시작, 그룹이 끝나야 다음 그룹. HP·MP는 바뀌지 않고 끝 위치만 남는다 | 9의 오프닝, 14의 하후돈 |
| `win` / `lose` | 즉시 확정 | — |

### 2.3 승패와 부목표

```json
"Victory": [{"Kind": "reach", "Unit": "liubei", "Area": 1, "FromRound": 6}],
"Defeat":  [{"Kind": "defeat", "Unit": "liubei"}, {"Kind": "occupied", "Area": 2, "By": "enemy"}],
"Sub":     [{"ID": "village-saved", "If": {"Kind": "defeatAll", "Group": "arsonists"},
             "Reward": {"skill": "…"}, "Flag": "flag-…"}]
```

- `Victory`와 `Defeat`는 목록. 하나라도 성립하면 확정, 같은 명령에서 둘 다 성립하면 패배가 우선. 지금의 `Boss`/`Threshold`/`Goal`은 이 목록의 `hp`·`annihilate` 조건으로 대체한다.
- 조건 종류: `defeat`(퇴각 또는 사망), `defeatAll`(그룹), `hp`, `reach`(`Unit` 또는 `All`: 목록 전원이 도달), `occupied`(적이 지점 점령), `survive`(N라운드), `annihilate`, `flag`.
- 한정어: `FromRound`(그 라운드부터 유효). 이벤트의 `objective` 동작이 목록을 교체해 단계 변경을 표현한다.
- **부목표 `Sub`**: 승리가 확정되는 순간 판정한다. 달성하면 보상·플래그가 붙는다. 표시는 전투 중 목표 창에 문구와 달성 여부를 한 줄씩 보인다(UI 추가는 이것 하나).
- **보상 `Reward`**: 지금의 아이템 맵에 예약 키 `provisions`(군량)를 둔다. 책략·특성 습득과 장수 합류는 부목표·이벤트 `grant`/`join`으로 준다.

### 2.4 NPC와 진영

- 진영: `ally`(플레이어), 아군 AI(`Spawn.AI`, `support` 페이즈), `enemy`. 손견군·조조군·제후군은 아군 AI로 배치하고 진영색(`Faction`)만 다르게 준다. 같은 조조군이 다른 스테이지에서 적이 되는 것은 스테이지마다 지정한다.
- 정책: `hold`(제자리, 인접한 적만 공격), `charge`, `flee`(목표로 가서 맵에서 빠짐), `march`(목표 지점으로 이동하며 교전), 기존 `Guard`.
- NPC 퇴각을 패배로 쓰려면 `Defeat`의 `defeat` 조건에 그 부대를 적는다.
- 조작 가능한 고정 출연 부대(19의 장비·관우 부대)는 스폰 `Playable`로 두고 출전 인원에 세지 않는다.

### 2.5 일기토

구조(`Stage.Duels`, `assets/duels.json`)와 결과 고정(`Outcome`, `AllyResult`/`EnemyResult`)은 지금 그대로다. 바꾸는 것은 발생 시점 하나다.

- **지금**: 한쪽이 상대를 공격할 때 그 공격이 일기토로 바뀐다(`duelFor`).
- **바꿀 것**: 지정한 두 장수가 **서로 인접한 상태가 되면** 일기토가 벌어진다. 공격 명령은 필요 없다. 이동 뒤 공격이든 대기든, 이동만 하든, 소환·합류·개전 직후에 놓였든 같다.
- 검사는 `settle()`의 첫 단계다(이벤트·승패보다 앞). 명령 하나가 끝나 아무도 동작 중이지 않을 때 `Stage.Duels`를 데이터 순서로 훑어 아직 치르지 않은 쌍이 인접해 있으면 하나를 치르고, 퇴각 같은 변화가 생겼으니 처음부터 다시 훑는다. 지나가는 칸은 세지 않고 멈춘 자리만 본다.
- 일기토를 치르면 그 차례의 주인(둘 중 자기 진영 차례인 쪽)의 행동이 끝난다. 소환·합류·개전으로 생긴 일기토는 누구의 행동도 소모하지 않는다.
- 공격 경로의 `duelFor`는 필요 없어지므로 지운다. 대신 UI가 이동 전에 알 수 있게 `DuelAt(유닛, 칸)` 질의를 둔다(그 칸에 서면 일기토가 벌어지는지. 터치 확인 탭에 쓴다).
- `Lines`를 선공자별로 둘 수 있다는 지금의 설계는 "먼저 인접한 쪽"으로 바꾼다(4의 장비/관우/유비).
- 일기토 뒤의 효과(디버프, 연쇄 일기토 등)는 일기토에 필드로 두지 않고 `duel:ID` 조건 이벤트로 붙인다. 디버프는 일반 디버프와 같은 상태이고 `buff` 동작이 `Turns`로 지속을 정한다(일기토 전용 상태는 없다).

일기토 상대를 원거리·책략으로 먼저 쓰러뜨리면 일기토는 일어나지 않는다. 막는 규칙은 두지 않는다(여포 무적과 같은 결정).

### 2.6 화염 타일

불은 새 규칙 없이 타일 두 종류로 쓴다. 타일은 지금의 `game.json` `Tiles`와 `terrain.json`에 더한다.

| 타일 | 진입 | 효과 |
|---|---|---|
| 불길 | 모든 병종 비용 0(들어갈 수 없다) | 지형 배율 없음. 성벽·물과 같은 장애물이지만 원거리 공격은 통과한다 |
| 불붙은 땅 | 들어갈 수 있다 | 모든 병종의 지형 배율 0.8. 선 채로 자기 차례가 시작되면 병력 회복 대신 최대 병력의 20%를 잃는다(`RestHP: -20`) |

- 지나가기만 하는 것은 피해가 없다. 그 칸에서 차례를 맞아야 한다. 그래서 불붙은 땅을 가로질러 가는 건 괜찮고 거기서 쉬는 건 아프다.
- 피해는 차례 시작의 지형 회복 자리(`RestHP`)에서 처리한다. `RestHP`가 음수일 때 병력을 깎고, 병력이 0이 되면 퇴각한다(1은 남기지 않는다).
- 불이 붙는 것은 `terrain` 동작이 타일을 바꾸는 것으로만 일어난다(1 영천의 방화). 바뀐 타일은 전투 상태(타일 덮어쓰기)에 저장한다. 불이 번지거나 꺼지는 규칙은 두지 않는다. 필요하면 이벤트가 다시 `terrain`으로 바꾼다.
- 타일 덮어쓰기는 `{타일, 남은 턴(0=영구)}`로 저장해 두면, 병법·점화 명령이 나중에 같은 수단을 쓸 수 있다. 처음 필요한 곳은 25 박망의 「불 놓기」(정해진 점화 지점에서 누구나 쓰는 명령)이고, 1~20에는 없다. 병법이 불을 붙이려면 불이 꺼지는 턴 수와 "타는 타일"(풀밭·숲 같은 `Tiles`의 `Burns` 지정)이 필요하다. 영구로 두면 병법 하나가 길을 막아 버린다. 1~20 구현에는 넣지 않는다.

---

## 3. 군량 경제

군량은 화폐다. 병력 회복 소비 아이템은 영걸전처럼 콩·보리·쌀, 병법치 회복은 술·특급주·빼갈이라 이름이 겹치지 않는다([data-item.md](../data-item.md)).

- `State.Provisions int`. 전투 보상, 시나리오 `grant`로 얻고 출전 증원과 매매에 쓴다.
- **출전 3단계**
  - **필수**: 반드시 나가는 장수(`Stage.Required`).
  - **정원**: 필수를 포함한 무료 인원(`Stage.Quota`, 시나리오의 "기본").
  - **증원**: 정원을 넘어 `Stage.Max`(시나리오의 "최대")까지. 한 명당 `Stage.ExtraCost`를 출진할 때 낸다. 어려우면 군량으로 병력을 늘리고, 아낄 수 있으면 아낀다.
  - 지금의 `Stage.Limit`는 `Max`로 바뀐다. 출전 불가 장수(관우 없음)는 `Stage.Barred`.
  - 4기부터 병량 소모(10 북해)는 이 증원 비용이 한다.
- **매매**: 회의장의 상점. 가격은 군량. 승급 아이템과 고유 보물은 팔 수 없다. 되파는 값은 가격의 일정 비율.
- 밸런스 수치(보상량, `ExtraCost`, 가격)는 맵 제작 때 정한다.

---

## 4. 회의장

```json
{"ID": "camp-b02", "Place": "연합 진영", "Stage": "S_camp", "Battle": "B02",
 "Brief": "굶주린 손견군의 후위를 맡는다.",
 "Talks": [{"Officer": "guanyu", "If": [], "Lines": "talk-b02-guanyu"}],
 "Shop": "shop-early"}
```

- 편성에 있는 장수가 무대에 서고, 클릭하면 짧은 대화가 나온다. 같은 장수에게 대화가 여러 개면 `If`가 맞는 마지막 것을 보여 준다. 새 대화가 있는 장수 머리 위에 "…"를 띄운다.
- 메뉴: 대화(무대), 상점, 편성·장비·부관(지금의 출전 준비), 저장, 출진(확인 후 체크포인트 저장).
- 회의장 대화는 보상을 주지 않는다. 정보와 정취, 플래그 설정만 한다.

---

## 5. 장면 연출

- 무대 맵은 `assets/scenes.json`(GUI 전용)에 둔다. 막사류·성내류·궁궐류는 하나로 합쳐 쓴다. 필요한 무대는 [graphics.md](graphics.md)에서 센다.
- 초상은 장수 ID로 찾는다. 초상이 없는 인물과 무명 화자(전령·병사·노인)는 범용 초상을 쓴다.
- 비트(`move` `face` `play` `emote` `wait` `effect` `intro`)는 `say` 노드의 `Beats`와 전투 위 `cutscene` 동작이 같이 쓴다.
- 일러스트(낙양, 젓가락)는 `narrate`/`say`에 `Illustration` 키로 건다.

---

## 6. 스테이지별 필요 기능

| 스테이지 | 새로 필요한 기능 |
|---|---|
| 1 영천 | `round`·`any`·`at` 조건, `turnstart` 조건(방화병), 아군 AI 스폰, `terrain`+화염 타일, `march`, `remove`, 부목표, 책략 보상, 서복 플래그 |
| 2 낙양 / 6 산조 (스토리) | 장면만(`scene`·`say`·`narrate`·일러스트) |
| 3 사수관 | `buff`(손견군), `move`, 일기토로 끝나는 승리, 손견 생존 아이템 보상 |
| 4 호로관 | `alldefeated`, 자동 일기토(선공자별 대사), 목순·무안국 일기토 |
| 5 변수 | `reach`(`All`), `round`, 아군 AI 도주(`flee`), 부목표(장수 전원 생존) |
| 7 신도성 | `occupied` 패배, `round` 스폰, 번궁 합류 |
| 8 거록 | `at`(길 굽이 진입), 복병, 부목표 → 플래그(거록 완승) |
| 9 계교 | `cutscene` 오프닝(동시 `move`·병종 동작·일기토 강제), `join`(조운), `flag:` 조건 이벤트로 선봉 교체, 임시 합류 |
| 10 북해 | 출전 3단계(정원 3, 최대 6), NPC `move`, 공융군 출격 |
| 11 담성 | `march` 대군, 피난민 `flee`/입성 집계, 성채 회복(기존 타일), 제갈현 부목표 |
| 12 서주 계승 / 16 허도 (스토리) | 장면만, `leave`(조운), `join`(미축·손건) |
| 13 우이 | `reach`, `round` |
| 14 소패 | `cutscene`(하후돈), 승리 `FromRound` |
| 15 하비 | `defect`, 일기토 무승부 |
| 17 서주 공방전 | `objective` 단계 변경, 출전 불가(관우) |
| 18 원소 진영 탈출 | `reach`, `join`(조운) |
| 19 고성 | 고정 출연 부대 `Playable`, 북문 함락 패배, 다중 패배 조건 |
| 20 여남 | `stall`, 정원 7 / 최대 9 |

---

## 7. 현재 코드에서 바꿀 곳

| 위치 | 지금 | 바꿀 것 |
|---|---|---|
| `internal/content/content.go` `Node`, `Stage`, `Campaign` | 문자열 필드 위주 노드, `Boss`/`Threshold`/`Goal`/`Limit` | §3의 노드·`Victory`/`Defeat`/`Sub`/`Events`, `Required`/`Quota`/`Max`/`ExtraCost` |
| `internal/content/battle_events.go` | 트리거 두 종류 검증 | 새 어휘 검증과 참조 무결성(유닛·그룹·일기토·플래그) |
| `internal/content/validate.go` | 노드 5종 | 새 노드 검증, 도달 가능성 |
| `internal/core/battle_events.go` | 스폰·대사 | 트리거·동작 전부, 깊이 제한, 반복 |
| `internal/core/engine.go` `settle()` | 보스 HP·전멸·턴 제한 | 승패 조건 목록, 부목표 판정 |
| `internal/core/formation.go` | 공격 시 일기토 | `settle()`의 자동 인접 일기토, `DuelAt` 질의 |
| `internal/core/types.go` `State` | `Inventory` `Warehouse` `Deployment` `Executed` | `Flags` `Provisions` 추가, 저장 형식 버전 상승 |
| `internal/core/combat.go` | 차례 시작 지형 회복 | 음수 `RestHP`(화염) |
| `internal/gui/scenario.go` | 대사·준비 화면 | `say` 화자 필드, 회의장, 출전 3단계 UI |
| `internal/gui/hud*.go` | 승패 조건 표시 | 부목표 한 줄 추가 |
| `assets/scenes.json` | 3개 무대 | 통합 무대 추가 |

**구현 순서**
1. 노드 구조·플래그·`Provisions`·출전 3단계 (데이터와 저장 형식)
2. 승패 조건 목록과 부목표
3. 이벤트 어휘(조건·`turnstart`)와 자동 일기토
4. 아군 AI 정책(`march`·`flee`), `terrain`과 화염 타일
5. `cutscene`, 회의장과 상점
6. 1~4스테이지를 먼저 만들어 검증하고 나머지를 이어 간다.

저장 형식이 바뀌므로 이전 세이브·리플레이는 읽지 않는다(콘텐츠 해시가 이미 호환을 끊는다).
