# 사용자용 역사서 (20261125-campaign-chronicle)

설계 문서: `ver5/docs/system-chronicle.md`. history: `ver5/docs/history.md` 맨 위 항목.

## 한 일

- 코어: `Engine.Entered`/`Engine.Reached(from)`로 한 명령이 닿은 웨이포인트(장 시작·전투)를 판정, `State.Run/Steps`(저장 때만 세션이 적음), `ClearLine`.
- 저장소(`internal/storage/chronicle.go`): `chronicle/<줄>/NNNN.json` 지점 파일(체크섬, 한 번 쓰면 덮지 않음), `run.json`(부모 줄·갈라진 번호). 기존 수동·자동 슬롯과 완전히 별도.
- 세션(`internal/session/chronicle.go`): 지점 기록, 되돌아가 새 지점에 닿으면 새 줄로 갈라짐(원래 줄 파일 불변), `chronicle_load`, 수동·자동 저장에 줄 표시. 디버그 `jump`는 기록 안 함. 기능 이전 저장은 줄 없음 → 불러온 뒤 처음 닿은 지점부터 새 줄(앞 기록을 지어내지 않음; 소급 복원은 지점별 저장이 없어 불가능).
- GUI(`internal/gui/chronicle.go`): 역사서 창(진행 탭, 행 오른쪽에 유비 Lv·저장 날짜, 누르면 확인 후 불러오기, 휠·드래그 스크롤), 시작 화면과 게임 메뉴에 `역사서`. 디버그 조작·플래그 변경 없음, 닿은 곳만 표시. 장면 이동은 변경 없음.
- 문서: system-chronicle.md(새), README 색인, system-save.md, history.md.

## B12(서주 구원전) 질문의 답

제갈현 또는 고순을 잃으면 그 줄은 연의로만 간다(조건: 제갈현 생존 + 고순 합류 + 22장 화해 + 24장 느린 길). 역사서의 B12 지점(출전 준비)에서 다시 하면 새 줄(`진행 2`)이 갈라져 다시 구할 수 있고, 처음 줄·기록은 그대로 남는다. 과거 저장의 플래그는 고치지 않는다. 실제 이야기를 새 게임부터 걸어 B12 진입 상태를 받아 확인(`TestB12PointReplaysTheModeBranch`): 둘 다 생존+느린 길=가상, 빠른 길=연의, 제갈현 전사=연의(선택 없음), 고순 전사=연의(선택 없음), 각 재시도 뒤 받은 지점 상태 불변.

## 변경 파일

core: internal/core/engine.go, waypoint.go, types.go, chronicle_test.go(새) / storage: chronicle.go(새) / session: chronicle.go(새), session.go, chronicle_test.go(새) / gui: chronicle.go(새), uidemo_chronicle.go(새), windows.go, game.go, events.go / docs: system-chronicle.md(새), README.md, system-save.md, history.md. (작업 트리에는 이 요청과 무관한 기존 미커밋 변경이 많고 그대로 두었다.)

## 검증 (test/…/test-log.md에 시각 포함 전체 기록)

- build-vet PASS, chronicle-core PASS(전체 웨이포인트 도달·B12 분기), session-storage PASS.
- core-all 첫 실행 FAIL: `TestEngineSourceNamesNoGameContent`가 `uidemo_chronicle.go`의 "B12" 이름을 잡음 → 데모를 12번째 전투 위치로 바꿔 수정, 같은 라벨로 재시험 PASS (core·content·cli).
- gui-all 첫 실행 FAIL: `go test ./internal/gui` 전체를 한 프로세스로 돌리다 5분 만에 `signal: killed`(8GiB 맥의 메모리 부족으로 추정, 테스트 단언 실패 아님). 같은 라벨로 gui 테스트 184개를 25개씩 나눈 바이너리 실행(`gui_batches.sh`)으로 재시험 PASS. 별도로 gui-targeted(저장·창·메뉴 관련) PASS.
- 세션 테스트: 지점 기록·날짜·레벨, 슬롯과 격리, 세션 재시작, 갈라짐·원본 파일 불변, 수동 저장 불러온 뒤 갈라짐, 기능 이전 저장, jump 제외, 손상(깨짐·체크섬)·내용 변경(ChangedContent)·쓰기 불가에도 게임·저장 유지.
- 캡처(`pc/`, `phone/`): PC 1280x850과 폰 2400x1080 uiscale 130에서 창·스크롤·확인창·메뉴 확인, 역사서 지점 불러오기로 실제 출전 준비 화면 진입(pc/frame-0400.png).

## 한계

- 캡처 데모의 기록은 `core.NewAt`으로 만든 웨이포인트 상태(실제 플레이 아님)라 창 모양만 판단할 수 있다. 실제 플레이로 기록되는 것은 코어·세션 테스트가 확인.
- 기능 도입 이전 진행은 복원하지 않는다. 기록은 삭제 기능이 없어 계속 쌓인다(삭제는 사용자 승인 사안으로 두었다).
- 한 명령이 장 시작과 전투를 함께 지나면 마지막만 남기며, 현재 데이터에선 일어나지 않음을 테스트로 확인.
- 터치 입력은 기존 `scroller`/버튼을 재사용했고 실기기 터치는 시험하지 않았다. 게임 데이터가 바뀌면 이전 지점은 일반 저장처럼 확인을 거쳐야 한다.
