# 지난 이야기 이름 반영 (20261125-chronicle-name)

## 한 일
사용자에게 보이는 `역사서`를 `지난 이야기`로 변경. 창 상단 안내를 "고르면 지나온 시점부터 다시 시작하며, 지금까지의 진행은 그대로 남습니다."로 바꿈(확인창 문구는 이미 "…에서 다시 시작합니다").
변경 파일(ver5): internal/gui/game.go(게임 메뉴), windows.go(창 제목·시작 화면 항목), chronicle.go, uidemo_chronicle.go, internal/session/chronicle.go(기록 실패 로그), internal/storage/chronicle.go(주석), docs/system-chronicle.md, system-save.md, README.md, history.md(새 항목).
저장 디렉터리 `chronicle/`, 데이터 형식, 코드 식별자, overlay 이름은 그대로라 기존 기록 호환.

## 검증
- build: `go build ./...` + `go vet ./internal/gui ./internal/session ./internal/storage` PASS
- chronicle-tests: `go test -p 1 ./internal/session ./internal/storage -run "Chronicle|B12"` 및 `./internal/gui -run "Chronicle|Menu|Title|Window"` PASS
(상세: test-log.md)

## 한계
- PC/모바일 게임 캡처는 못 함: 이 에이전트 환경에서 `_uidemo-vm`이 `no monitor was found at initializeGLFW`로 실패(모니터 없음). 화면 확인은 미실시이며 다음 명령으로 재현 가능: `go run ./cmd/_uidemo-vm -ui chronicle -frames 320 -at 20,100,215,305 -out <dir>` (폰: `-size 2400x1080 -uiscale 130`).
- 새 안내문이 기존 창 폭에 들어가는지는 시각 확인 전이다(길이 약 +4자).
- 이전 요청 결과 문서(campaign-chronicle/result.md)의 `역사서` 표기는 기록이므로 그대로 둠.
