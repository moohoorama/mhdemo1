# 삼국지 SRPG CLI

전체 설계와 최신 결정은 [삼국지 SRPG 전체 설계서](srpg_full_design.md)에 통합했다.
기본 규칙·전체 콘텐츠 표·성장/부관/일기토·기술 구조·그래픽 방향과 현재 실행 데이터를 포함한다.

Go 1.26+, Ark ECS v0.8.3. 도원결의에서 호로관까지 세 전투를 진행하는 게임 코어와 CLI.

```sh
go run ./cmd/srpg-cli
go run ./cmd/srpg-cli -json -seed 1
go run ./cmd/srpg-sim -seeds 10
go test ./...
go vet ./...
```

실행 콘텐츠는 `assets/content/campaign.json`이며 `-content`로 경로를 지정한다.
저장은 OS 사용자 설정 디렉터리의 `srpg-cli`에 기록하며 `-data`로 변경할 수 있다.
수동 슬롯은 1~10, 자동 슬롯은 auto. 설정은 저장 파일과 분리된다.

## 에셋 파이프라인

런타임 에셋은 `assets/`, 소스 artwork는 `tools/spritetool/assets/`에 둔다.
프로젝트 전용 sprite 변환은 `tools/spritetool`에 추가하지 않는다.
Sprite 생성·수정 전 `tools/spritetool/assets/SPRITE_ART_GUIDE.md`와 대상 설정/README를
먼저 읽는다. 현재 CLI에는 sprite와 spritetool이 없으며 artwork를 생성하지 않는다.

## 콘텐츠와 규칙

`srpg_complete_design.md` 1부의 확정 규칙을 우선한다. 콘텐츠 JSON의 draft 필드는
대사·배치·보급 수량·성장 경제 등 구현 초안의 출처를 기록한다. 병종 계수, 지형,
주요 적 프로필은 2부에서 가져온다. 초기 세 전투의 병종·장비·고유 특성과 학습
목록을 제공하며 후반 콘텐츠·승급·부관은 공급하지 않는다. 적 레벨은 고정이다.
`python3 tools/build_content.py`로 설계 문서와 초안 배치에서 콘텐츠를 재생성한다.
콘텐츠 변경 시 Version을 변경해야 하며 저장은 동일한 core/rules/content 버전과
콘텐츠 SHA256에서만 복원한다. 저장 버전은 1, 자동 마이그레이션은 없다.

Core는 파일과 입출력에 의존하지 않는다. 장수 성장/학습/장비와 전투 부대는 Ark
컴포넌트로 관리한다. 조회는 복사본이며 명령은 별도 world에서 적용 후 커밋한다.
실패한 명령은 revision과 RNG를 포함한 원래 상태를 보존한다. RNG는 저장 가능한
SplitMix64이고 타격은 부대 ID 순서, 명중→크리티컬→방어 순이다. AI 동점은 ID와
좌표 순서이며 판정 RNG를 사용하지 않는다. 반격 재반격은 없고 기본 반격은 없다.

## 사람용 CLI

`help`로 명령을 확인한다. 메뉴 번호 1~4는 시작/불러오기/설정/종료.
`next`, `choose 결의`, `deploy 유비 관우 장비`, `equip 유비 item_024`, `start`로 진행.
전투에서는 `move 관우 5 4`, `attack 관우 적ID`, `skill 유비 격려 관우`,
`item 유비 소군량 유비`, `learn 관우 속공`, `wait 관우`, `end`, `ai`를 사용한다.
`map`, `status`, `legal 관우`, `preview attack 관우 적ID`는 상태를 바꾸지 않는다.
`auto`는 현재 아군 진영만 AI로 진행한다. 적 AI 설정은 auto/step을 지원한다.
`save 1`, `load 1`, `slots`, `settings`, `menu`, `title`, `retry`, `quit`도 지원한다.
덮어쓰기와 미저장 진행 포기는 확인한다. 로그 상세도, 색상, 대사 표시 속도를 설정할 수 있다.

## JSON Lines 프로토콜

요청과 응답은 한 줄 JSON이다. stdout에는 JSON만 출력한다.

```json
{"id":"a","op":"new","seed":1}
{"id":"b","op":"command","revision":0,"command":{"kind":"next"}}
{"op":"observe"}
{"op":"legal","actor":"관우"}
{"op":"preview","command":{"kind":"attack","actor":"관우","target":"B01_enemy_01"}}
{"op":"save","slot":"1","overwrite":true}
{"op":"load","slot":"1","discard_current":true}
{"op":"settings","settings":{"ai":"step","color":false,"detail":true,"text_delay_ms":0}}
```

모든 command, ai, auto, retry 요청은 현재 revision을 필수 지정한다.
command 종류: next, choose(option), deploy(deployment), equip/unequip(actor,item),
start, move(actor,x,y), attack(actor,target), skill(actor,target,skill),
item(actor,target,item), learn(actor,trait), wait(actor), end, continue.
응답은 id/ok/error/events/observation을 포함한다. 오류는 안정적인 코드 접두사를 갖는다.
AI는 한 요청당 한 명령이며 진행 위치는 전장의 행동 예산에 보존된다.

## 검증과 확장

`internal/sim`은 코어의 합법 행동/예상 효과만 이용해 캠페인을 실행한다.
`go run ./cmd/srpg-sim -seeds 10`으로 고정 시드별 승패·라운드·성장을 출력한다.
GUI는 후속 작업이다. 코어 API Observe/LegalActions/Preview/Apply/Snapshot/Restore를
재사용할 수 있다. CLI의 EOF는 종료하며 현재 상태를 auto 슬롯에 저장한다.
EOF 저장 실패는 stderr와 프로세스 실패로 알린다.

전체 CLI JSON 캠페인의 실제 프로세스 검증: `go build -o bin/srpg-cli ./cmd/srpg-cli` 후
`python3 tools/smoke_cli.py`. JSON observation 내부와 core DTO는 Go 필드명
(Revision, Phase, Node 등)을 사용한다. 요청의 command 필드는 대소문자를 구분하지 않는다.


## 다음 작업: 성장 밸런싱과 그래픽 버전

최신 사용자 결정과 구현·검증 순서는
[srpg_go_core_cli_design.md §16](srpg_go_core_cli_design.md#16-최신-사용자-결정과-그래픽-버전-작업-기준-2026-10-01)에 기록했다.
병법 공격 경험치 2배, 방어 경험치 없음, 레벨 노가다 허용, 부관 경험치 85%,
일기토 생존 시 1레벨분 경험치가 최신 규칙이다. 이 변경과 GUI는 아직 구현 대기다.
그래픽 버전의 우선 완료 범위는 기존 초반 세 전투이며 25개 스테이지는 장기 설계 기준이다.

## 그래픽 앱과 원화 검수

```sh
go run ./cmd/srpg-gui
go build -o bin/srpg-gui ./cmd/srpg-gui
python3 tools/package_gui.py
```

macOS 패키지는 `bin/SRPG.app`. 부대 선택→이동/공격/병법/아이템/학습/일기토 메뉴로
조작한다. F5 저장, F9 불러오기, Tab 부대 전환, Space 추천 한 명령, Esc 메뉴.
부대의 이동·보행·공격·피격, 화살/책략 투사체, 회복/피해 숫자와 일기토 초상 창을 제공한다.
그래픽 렌더링은 저장되는 전투 판정을 바꾸지 않는다.

[그래픽·애니메이션 검수 미리보기](reports/graphics-preview.html)는 브라우저에서 직접 열 수 있다.
코어의 새 게임 전장 관측을 사용하는 시각 검수 페이지이며 캠페인을 실행하는 앱과는 구분한다.
원화 제작 방향·프롬프트는
[ART_DIRECTION.md](tools/spritetool/assets/three-kingdoms/ART_DIRECTION.md)에 기록했다.
새 런타임 원화는 `assets/graphics/`, 원본은 `tools/spritetool/assets/three-kingdoms/`.
Noto Sans KR과 OFL 라이선스는 `assets/fonts/`에 포함한다.

새 성장·승급·부관·일기토와 GUI 구현은 진행 중이다. 성장/저장/명령 재현 테스트는 통과했다.
그래픽 재구성 이후 실제 화면 입력 검증은 컴퓨터 조작 권한 및 사용 가능한 모니터가
필요해 아직 완료하지 못했다. 일부 공격 프레임의 무기 끝과 실제 크기에서의 반복 재생도
최종 시각 검수가 남아 있다.
