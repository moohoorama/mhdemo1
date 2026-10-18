# 저장과 불러오기

범위: 수동 슬롯, 회전 자동 저장, 불러오기, 재도전 체크포인트, 호환 검사, 저장 파일 위치.
기준: 값이 다르면 `internal/storage/storage.go`, `internal/session/session.go`, `internal/core/api.go`(`Restore`)·`core/types.go`(`State`, 버전 상수)를 따른다.

저장하는 것은 규칙 엔진의 상태(`core.State`)뿐이다. 카메라·창·연출·선택 상태는 저장하지 않고 불러온 뒤 화면이 상태에서 다시 만든다.
설정(AI 진행·기록 상세·대사 속도·UI 크기·데미지 배율)은 `settings.json`에 따로 있고 세이브에 섞이지 않는다.
화면의 저장·불러오기 창은 [presentation-ui.md](presentation-ui.md), 구조는 [system-architecture.md](system-architecture.md)에 있다.

## 슬롯

| 슬롯 | 쓰는 때 | 파일 |
|---|---|---|
| 수동 `1`~`10` | 플레이어가 저장을 누를 때 | `slot-1.json` ~ `slot-10.json` |
| 자동 `auto1`~`auto4` | 아래 시점에 순서대로 | `slot-auto1.json` ~ `slot-auto4.json` |

- 수동 저장은 수동 슬롯에만 된다(자동 슬롯에 저장을 요청하면 `InvalidSlot`). 불러오기는 열 네 슬롯 모두 된다.
- 이미 있는 수동 슬롯에 저장하면 먼저 확인한다(세션이 `OverwriteRequired`를 돌려주고, 화면은 `기존 저장을 덮어쓰시겠습니까?`를 묻는다).
- 저장은 시나리오·배치·전투·결과·완료 어느 단계에서든 된다. 불러오면 저장한 단계(대사 노드, 배치 중, 전투 턴 중간)로 돌아간다.
  대사 타이핑 위치처럼 연출 상태는 저장하지 않으므로 대사는 처음부터 다시 나온다.
- 화면에서 `F5`/`F9`는 마지막으로 쓴 수동 슬롯(처음에는 1)에 저장·불러오기를 한다.
- 옛 단일 자동 슬롯 `auto` 파일이 남아 있으면 읽을 수는 있으나(`slots` 목록에 나온다) 새로 쓰지 않으며 화면 목록에도 없다.

## 파일 형식과 쓰기

슬롯 파일은 봉투 JSON이다: `Sequence`(자동 저장 순번, 수동은 생략), `Format`(1), `Checksum`(상태 JSON의 SHA-256), `State`.

- 쓰기는 같은 폴더의 임시 파일에 쓰고 `fsync`한 뒤 이름을 바꾸는 방식이라 도중에 끊겨도 기존 파일이 남는다. 폴더는 권한 0700으로 만든다.
- 읽을 때 JSON이 깨졌거나 체크섬이 다르면 `CorruptSave`, `Format`이 1이 아니면 `IncompatibleSave`다.
- 불러오기 창에서 읽을 수 없는 슬롯은 `읽을 수 없음`으로 보인다.

## 자동 저장

- 시점: 전투 시작(배치를 마치는 `deployment_finish`)과 매 아군 새 턴 시작(`turn` 이벤트의 대상이 `ally`)이다. 둘 다 전투 단계일 때만 쓴다.
  이동·공격·적군·동맹군 차례 도중에는 쓰지 않으므로 최근 네 턴의 시작 상태가 남는다. 적 페이즈를 AI가 이어서 처리하다 새 아군 턴이 열릴 때도 같은 경로로 쓴다.
- 회전: 네 슬롯의 가장 큰 `Sequence`를 찾아 다음 순번을 쓴다. 슬롯은 `Sequence % 4`로 정해져 auto1 → auto2 → auto3 → auto4 → auto1이다.
  순번이 파일에 들어 있으므로 게임을 다시 켜거나 과거 저장을 불러온 뒤에도 순서가 이어진다. 한 번의 쓰기에 상태와 순번이 같이 들어간다.
- 자동 저장은 `Dirty`(미저장 진행) 표시를 바꾸지 않는다. 쓰기에 실패하면 `자동 저장 실패: …` 이벤트가 로그에 남고 게임은 계속된다.
- 불러오기 창은 자동 저장 네 개의 장·전투 이름·턴·진영·시각을 보여 준다.

## 불러오기

`load` 요청은 다음 순서로 처리한다(`session.go`).

1. 미저장 진행(`Dirty`)이 있고 버리겠다는 표시(`discard_current`)가 없으면 `DiscardRequired`를 돌려준다. 화면은 `저장하지 않은 진행을 버리시겠습니까?`를 묻는다.
2. 파일을 읽고 체크섬을 검증한다.
3. `core.Restore`가 호환 검사와 구조 검증을 하고 새 엔진을 만든다. 실패하면 현재 엔진은 그대로다.
4. 성공하면 `Dirty`를 끄고 플레이 화면으로 간다. 화면은 유닛·필드·로그를 비우고 상태에서 다시 만든다.

`Dirty`는 새 게임·명령·재도전으로 켜지고 저장·불러오기·시작 메뉴 이동으로 꺼진다. `title`·`quit`·`new`·`load`가 이 확인을 거친다.

## 호환 검사

`Restore`는 다음이 현재 빌드와 모두 같아야 통과한다. 하나라도 다르면 `IncompatibleSave`(`호환되지 않는 저장입니다`)다.

| 항목 | 값(지금) | 정의 위치 |
|---|---|---|
| 상태 형식 `Format` | 2 | `core/engine.go` |
| 코어 `Core` | `2` | `core.CoreVersion` |
| 규칙 `Rules` | `rules-13` | `core.RulesVersion` |
| AI 정책 `AI` | `greedy-8` | `core.AIPolicyVersion` |
| 난이도 | `normal` | 고정 |
| 콘텐츠 버전 `Content` | `ver5-1` | `assets/data/game.json`의 `Version` |
| 콘텐츠 해시 `ContentHash` | SHA-256 | `content.Load` |

- 콘텐츠 해시는 `assets/data/` 아래 모든 `.json`(맵 문서 포함)을 경로 순으로 정렬해 파일 경로와 바이트를 이어 해시한 값이다. 글자 하나만 바뀌어도, 맵 에디터로 저장해도
  이전 세이브는 모두 호환되지 않는다. `assets/scenes.json`·그림·스프라이트는 해시에 들어가지 않으므로 바꿔도 세이브는 유효하다.
- 규칙 계산이 달라지는 코드 변경은 `RulesVersion`을 올린다. AI 선택이 달라지면 `AIPolicyVersion`을 올린다. 올리지 않으면 같은 세이브가 다른 결과로 이어진다.
- 호환이 맞아도 `Restore`는 내용을 검증한다: 단계·노드 종류, 인물·병종·장비·특성 참조, 레벨·경험치 범위, 출전 명단, 유닛 위치와 중복, 고정 배치,
  승패 조건에 필요한 보스, 체크포인트의 모양. 어긋나면 `CorruptSave`·`CorruptCheckpoint`(`저장 파일이 손상되었습니다`)다.

## 재도전 체크포인트

- 출전 준비에서 `start`를 처리할 때 그 시점의 상태(일시 유닛·적을 뺀 인물, 소지품, 창고, 난수 상태)를 `State.Checkpoint`에 한 벌 넣는다.
  체크포인트는 상태 안에 들어 있어서 배치·전투·결과 단계의 모든 저장(자동 저장 포함)에 함께 저장되고, 이 세 단계에서 체크포인트가 없으면 `CorruptSave`다.
- 패배 결과(`result`, `defeat`)에서 `retry`를 받으면 체크포인트로 엔진을 되돌린다. 화면은 출전 준비 노드로 돌아가며 `출전 준비부터 재도전`이다.
  소비한 아이템·성장·난수까지 그 시점으로 돌아가고 일부만 유지하지 않는다. 난수 상태(`RNG`)도 체크포인트에 있어서 같은 명령을 내리면 같은 결과가 나온다.
- 승리 뒤 다음 노드로 넘어가면(`continue`) 체크포인트를 버리고 다음 출전 준비에서 새로 만든다. 체크포인트 안의 체크포인트는 없다.

## 난수와 결정성

세이브가 상태를 정확히 되살릴 수 있는 것은 엔진이 결정적이기 때문이다. 난수는 `State.RNG` 하나이고 매번 상수를 더해 섞는(SplitMix64 꼴) 함수로만 뽑는다.
새 게임의 시드(`Seed`)가 `RNG`의 처음 값이며 GUI·CLI의 `-seed`(기본 1)로 정한다. 코어와 AI는 시계나 외부 난수를 쓰지 않는다.
같은 콘텐츠·같은 시드·같은 명령열은 같은 상태로 이어지고, `save` 뒤 `load`한 상태는 저장 전 관찰값과 같다(`tools/smoke_cli.py`가 이를 확인한다).

## 저장 위치

- 데스크톱 GUI와 CLI의 기본 폴더는 `os.UserConfigDir()/srpg-cli`다(macOS `~/Library/Application Support/srpg-cli`, Linux `~/.config/srpg-cli`). `-data <폴더>`로 바꾼다.
  GUI와 CLI가 같은 폴더를 쓰므로 한쪽에서 만든 세이브를 다른 쪽에서 불러올 수 있다. 폴더에는 슬롯 파일과 `settings.json`이 있다.
- Android는 앱 내부 저장소의 `saves/` 폴더이고 에셋과 따로 있다. 자세한 것은 [tools-build.md](tools-build.md)에 있다.
- 맵 에디터의 시험 전투는 임시 폴더를 저장 위치로 쓰므로 위 슬롯을 건드리지 않는다([tools-editor.md](tools-editor.md)).

## CLI

수동 저장 `save 1`~`save 10`, 불러오기 `load 1`~`load 10`·`load auto1`~`load auto4`, 목록 `slots`, 재도전 `retry`다. 덮어쓰기와 미저장 진행은 대화형으로 묻는다.
JSON 모드는 대신 요청의 `overwrite`·`discard_current` 필드로 뜻을 밝히고, 명령 요청은 현재 `revision`을 함께 보내야 한다(어긋나면 `StaleRevision`).
