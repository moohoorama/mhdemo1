# 구조

범위: 패키지 배치와 의존 방향, 엔진·데이터 분리 원칙, 명령·세션 흐름, 결정적 난수, 콘텐츠 해시와 호환.
기준: 값이 다르면 `internal/*` 코드를 따른다(진입점 `cmd/srpg-gui`·`cmd/srpg-cli`, 엔진 `internal/core`, 로더 `internal/content`).

저장 형식과 호환 검사의 세부는 [system-save.md](system-save.md), 화면은 [presentation-ui.md](presentation-ui.md)·[presentation-scene.md](presentation-scene.md),
전투 규칙은 [logic-combat.md](logic-combat.md)에 있다.

## 원칙

엔진(`internal/`)에는 인물·병종·아이템·스킬·특성의 이름과 ID가 하나도 없다. 게임 내용은 모두 `assets/data/`의 데이터 묶음이다.
데이터는 이름을 자유롭게 짓고 엔진이 아는 어휘(`internal/content/vocab.go`: 특성 효과 키, 스킬 종류·모드·모양·효과, 상태, 병종 태그, 아이템 효과)만 조합한다.
새 어휘가 필요하면 엔진 코드를 고친다. 데이터 쪽 이름이 코드에 없다는 증거로 `internal/testdata/mini`(다른 게임의 최소 데이터 묶음)가 엔진·로더·저장·CLI·AI·세션 테스트를 돌린다.

- 화면 연출(장면·일기토 대본·초상·스프라이트)은 데이터 묶음 밖의 GUI 자산이다: `assets/scenes.json`, `assets/duels.json`, `assets/graphics/`, `assets/sprites/`, `assets/fonts/`.
  이것들은 콘텐츠 해시에 들어가지 않고 규칙에 영향을 주지 않는다. 일기토 대본의 결과는 규칙이 정한 결과와 같아야 한다.
- 규칙은 연출 시간에 따라 바뀌지 않는다. 화면은 이미 확정된 결과를 재생할 뿐이다.
- 데이터 묶음 위치는 `-content`(기본 `assets/data`), 화면 자산 위치는 글꼴 파일 경로 기준(`<assets>/fonts/NotoSansKR.ttf`의 상위 `assets/`)이다.

## 패키지

```
cmd/srpg-gui  Ebitengine 진입점            cmd/srpg-cli  사람용·JSON Lines 진입점
cmd/_*        확인용 임시 도구(_uidemo, _uidemo-vm 등)
internal/
  content   데이터 묶음 로더·검증, 엔진 어휘(vocab.go), 맵 문서(map_document.go)
  core      규칙 엔진: 상태, 명령 처리, 조회
  ai        적·자동 진행의 명령 선택
  session   화면 상태, 요청 처리, 저장·설정 연결
  storage   슬롯·설정 파일, 안전한 쓰기
  cli       명령 문법 분석, 텍스트·JSON 입출력
  replay    클레임 리플레이: 한 페이즈의 행동 순서·선행 관계
  sprite    .spr 로더, 태그 팔레트, 그림자, 시트 캐시
  duel      일기토 대본·시간 계산
  duelview  일기토 그리기, 말풍선 렌더러
  mapedit   맵 편집 이력과 안전 저장
  gui       Ebitengine 화면·입력·연출
  testdata/mini  엔진 테스트용 최소 데이터 묶음
```

의존은 아래에서 위로만 흐른다. 파일을 읽고 쓰는 곳과 Ebitengine을 쓰는 곳이 정해져 있다.

| 패키지 | 의존하는 내부 패키지 | Ebitengine |
|---|---|---|
| content | 없음 | 안 씀 |
| core | content | 안 씀, 파일 접근 없음 |
| ai | content, core | 안 씀 |
| storage | content, core | 안 씀 |
| replay | content, core | 안 씀 |
| session | ai, content, core, replay, storage | 안 씀 |
| cli | content, core, session | 안 씀 |
| mapedit | content | 안 씀 |
| duel | content | 안 씀 |
| sprite | 없음 | 씀(그림 조각) |
| duelview | duel, sprite | 씀 |
| gui | 위 전부(ai, cli, content, core, duel, duelview, mapedit, replay, session, sprite, storage) | 씀 |

`gui`가 `cli`를 쓰는 곳은 디버그 입력줄(백틱)의 명령 문법 분석뿐이다. Android용 시작 함수는 최상위 `mobile` 패키지에 있다([tools-build.md](tools-build.md)).

## 콘텐츠 로드

`content.Load(dir)`는 폴더 아래 모든 `.json`을 이름순으로 읽는다(`DisallowUnknownFields`).

- 최상위 파일은 `content.Data`의 한 부분씩(`game`·`classes`·`characters`·`skills`·`traits`·`equipment`·`items`·`terrain`·`script`·`mapeditor`)이고 한 부분을 여러 파일로 쪼개도 된다.
- `maps/<ID>.json`은 맵 문서 한 벌이다. 로드할 때 문서를 전투 `Stage`로 컴파일한다(대타일 판정·좌표 정규화). 형식은 [data-map.md](data-map.md)다.
  `stages/*.json`은 단순 스테이지 정의로, 테스트·사용자 묶음에서 쓴다.
- 읽은 모든 파일의 경로와 바이트로 해시(`Data.Hash`)를 만든다. 이어서 참조(병종 승급 사슬, 학습 풀)를 잇고 검증한다. 검증에 실패하면 로드가 실패한다.
- 데이터 필드의 뜻은 `assets/data/README.md`와 [data-character.md](data-character.md)·[logic-traits.md](logic-traits.md)에 있다.

## 엔진(core)

- 상태는 `core.State` 하나로 직렬화된다. 인물(`Character`, 캠페인 내내)과 전투 유닛(`Unit`, 한 전투)은 내부에서 ECS 세계(Ark)에 들어 있지만 ECS 타입은 밖으로 나오지 않는다.
- 상태를 바꾸는 길은 `Apply(Command)` 하나다. 개인 ECS 세계를 만들어 거기서 명령을 실행하고, 끝까지 성공했을 때만 상태를 교체하고 `Revision`을 올린다.
  실패하거나 패닉하면 `InternalError` 등을 돌려주고 상태는 그대로다. 결과는 `Result{Revision, Events}`다.
- 조회(읽기 전용): `Observe()`(상태 + 대사 노드 + 지도 + 유닛 뷰(능력치·특성·병법 계산 포함) + 열려 있는 학습창), `LegalActions(actor)`(가능한 명령 목록),
  `Preview(command)`(피해·확률·경험치 예측), `CharacterStats`, `AttackRange`, `Threat`, `NextTurnRecovery`, `DuelFor`, `Snapshot()`/`Restore()`.
  화면과 AI는 이 조회만 쓰고 별도 공식을 두지 않는다. 화면이 명령을 시험해 볼 때는 `Restore(Snapshot())`로 샌드박스 엔진을 만든다.
- 명령은 `Command{Kind, Actor, Target, Skill, Item, Trait, Option, X, Y, Deployment}`이고 종류는 다음과 같다.
  시나리오 `next`·`choose`·`continue`, 편성 `deploy`·`equip`·`unequip`·`deputy`·`undeputy`·`start`, 배치 `place`·`unplace`·`deployment_finish`,
  성장 `learn`·`learn_close`, 전투 `move`·`attack`·`skill`·`item`·`wait`·`end`. 단계·차례·권한 검사는 코어 안에서 한다.
- 진행 단계(`Phase`)는 `scenario` → `deployment` → `battle` → `result` → 다음 노드 또는 `complete`이다. 차례(`Turn`)는 `ally`(아군) → `support`(동맹 AI) → `enemy`다.
  규칙·성장·이벤트는 [logic-combat.md](logic-combat.md)·[logic-growth.md](logic-growth.md)·[logic-battle-event.md](logic-battle-event.md)·[logic-campaign.md](logic-campaign.md)가 기준이다.
- 난수는 `State.RNG` 하나에서 결정적으로 뽑는다. 코어와 AI는 시계·외부 난수를 쓰지 않으므로 같은 콘텐츠·시드·명령열은 같은 결과가 된다([system-save.md](system-save.md)).

## AI

`ai`는 코어가 내놓은 가능한 행동(`LegalActions`)과 평가(`Preview`)만으로 명령을 고른다. `ai.Next`는 현재 차례의 다음 한 명령, `ai.Choose(actor)`는 그 유닛의 명령,
`ai.Learning`은 학습창 선택이다. 고른 명령은 사람의 명령과 같은 `Apply`를 지나므로 같은 검사를 받는다. 정책이 바뀌면 `core.AIPolicyVersion`을 올린다.
AI는 아군 자동 진행(`자동 진행`, `Space`), 동맹 AI 유닛, 적 페이즈에 쓰인다.

## 세션

`session.Session`은 데이터 묶음, 엔진, 저장소, 설정, 화면(`title`·`playing`·`menu`·`quit`), 미저장 표시(`Dirty`)를 쥔다. 화면·CLI는 `Handle(Request)`로만 엔진을 움직인다.

- 요청(`Op`): `new`·`observe`·`slots`·`settings`·`menu`·`resume`·`title`·`quit`·`save`·`load`·`legal`·`preview`·`retry`·`command`·`ai`·`auto`.
- `command`·`ai`·`auto`·`retry`는 요청에 현재 `Revision`이 같이 와야 한다(다르면 `StaleRevision`). 전투 중 아군 차례가 아니면 사람의 `command`는 `NotYourTurn`이다
  (학습 명령은 예외). `ai`는 한 명령, `auto`는 현재 진영의 남은 차례를 한도(1000단계·10초) 안에서 끝까지 진행한다.
- 응답(`Response`)은 성공 여부·오류 코드·화면·이벤트·관찰값(·가능한 행동·예측·슬롯·설정)이다. 오류 코드는 영문(`NoGame`, `WrongPhase` 등)이고 화면이 한국어로 바꿔 보인다.
- `ResolvePhase`는 한 진영의 페이즈를 AI로 한 명령씩 모두 처리하고 행동 목록(`replay.Action`)을 돌려준다. 같은 유닛의 연속 명령은 한 행동으로 묶는다.
- 자동 저장은 `session.apply`가 명령 결과 이벤트를 보고 쓴다([system-save.md](system-save.md)).
- 설정(`storage.Settings`): 적 AI(자동/한 명령씩)·색·기록 상세·대사 지연(ms)·UI 크기·데미지 배율(60·70·80·90·100%, `damage_pct`, 0이면 기본 80). 경험치 배율(60~100%, `xp_pct`, 기본 80)도 같다. 세션이 요청마다 엔진 상태(`State.DamagePct`·`XPPct`)로 옮겨 `damage()`가 곱하므로 예측·시험 복사본도 같은 값을 쓴다. 한 번의 피해는 최소 1. 세이브와 별개 파일이다.

## 화면(gui)과 리플레이

`gui.Game`은 세션 위의 입력·그리기 계층이다. 클릭이 `core.Command`가 되고 `Game.send`가 `session.Request`로 보낸다. 요청마다 전·후 관찰값을 비교해 기록·알림·레벨업 창을 만든다.

- 플레이어의 한 명령은 행동 하나짜리 클레임 리플레이로, 적 페이즈는 `ResolvePhase`가 돌려준 행동들을 `replay.Link`로 선행 관계를 이어 동시에 재생한다.
  규칙은 한 명씩 순서대로 계산한 결과 그대로이고, 겹쳐 보이는 것은 화면뿐이다. 알고리즘은 [presentation-replay.md](presentation-replay.md)에 있다.
- 명령 도중의 임시 이동은 샌드박스 엔진에서 한다. 세션에는 확정할 때만 간다([presentation-ui.md](presentation-ui.md)).
- 고정 캔버스 화면(맵 에디터, 일기토 감상)을 뺀 화면은 UI 배율이 적용되는 한 좌표계로 그린다.
- 디버그: `-audit <폴더>`는 요청·응답을 `gui-input.jsonl`로 남기고 `F12` 화면을 저장한다. `-verify`는 입력 합성 없이 화면 상태 기계와 저장·불러오기를 점검한다.

## CLI

`cmd/srpg-cli`는 같은 세션을 텍스트로 구동한다. 사람용 모드는 명령 문법(`move 유비 3 4`, `auto`, `save 1` 등)을 받고, `-json`은 한 줄에 요청 하나·응답 하나의 JSON Lines 프로토콜이다.
플래그는 `-content`·`-data`(저장 폴더)·`-seed`·`-json`이다. 외부 AI나 스크립트가 같은 규칙으로 게임을 돌릴 때 쓴다(`tools/smoke_cli.py`).

## 콘텐츠 해시와 호환

저장·불러오기는 규칙 버전·AI 정책 버전·콘텐츠 버전·콘텐츠 해시가 현재 빌드와 모두 같을 때만 통과한다. 데이터 JSON(맵 문서 포함)이 한 글자라도 바뀌면 해시가 달라져
이전 세이브는 호환되지 않는다. 연출 자산은 해시에 들어가지 않는다. 표와 이유는 [system-save.md](system-save.md)에 있다.

## 검증

```sh
go test ./... && go vet ./...
go test -v -failfast -run TestName ./internal/core/...     # 특정 테스트
go build -o bin/srpg-cli ./cmd/srpg-cli && python3 tools/smoke_cli.py   # CLI를 JSON Lines로 끝까지 플레이
SRPG_RENDER_TEST=1 go test ./internal/gui -run TestRenderedEditorAndStatuses -count=1   # 실제 창 렌더링 검사
```

`SRPG_RENDER_TEST=1`일 때만 실제 창을 열어 화면을 `reports/`에 저장하는 렌더링 검사가 돈다.
