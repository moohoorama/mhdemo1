# 결과: 새 상품 야외 점원 안내 + 요청 처리 절차 변경

요청: `20261125-shop-advert-workflow` (완료 시 `<이 폴더>/20261125-shop-advert-workflow.md`로 이동)
진행 기록: `process/20261125-shop-advert-workflow/progress.md` (1분마다 실제 시각) · 테스트 기록: `test-log.md`(이 폴더에 복사본) · 연결: `LINKS.md`
설계: `design.md` · 전수 조사표: `survey.md` · 전후 캡처: `captures.html` · 문구 판본: `shop-advert-talk-drafts.html`

## 1. 새 상품 야외 점원

**원인**: 새 상품을 손 표시(`Shop.Mark`)로 가렸고 업 대장간 하나뿐이었다. 03 주점·08 도구점·14 도구점·주점의 광고 점원(`점원#1`)은 가게 안쪽에 있었고 `Talk`가 없었다. 03 주점은 특급주를, 19 대장간은 6종 중 3종만 외쳤고 말을 걸어도 새 물건 설명이 없었다. 손 표시는 재방문 상점(서주성 14·18장, 도구점 8·9·11장)도 가르지 못한다. 상세는 `survey.md`.

**전수(상점 사람 69곳)**: 정상 62(새 상품 없음, 점원 없음) / 문제→수정 5 / 추가 확인 2(1장 시작 재고). 새 상품은 데이터 비교로 계산했다: 03 주점(콩·보리·술·특급주), 08 도구점(만능약), 14 도구점(중급 인수)·주점(빼갈), 19 업 대장간(철검·철창·대궁·가죽장갑·구리갑옷·가죽옷). 이전 장에서 팔던 물건을 다시 홍보하는 곳은 없었다.

**수정**
- `content.Data.NewStock`(새 `stock.go`): 앞선 탐험의 어느 상점도 안 판 물건을 장·노드 순서로 계산(연의/가상 갈래 분리, 1장은 시작 재고). `Shop.Mark` 삭제.
- `scenegraph` `shop-notice` 규칙 재작성: 문 6칸 안·건물 앞·문이나 길(SW)을 보는 점원, 말풍선·대화가 새 물건 이름을 모두 말하고 값(회복량)이 데이터와 맞는지, 가게 안/이미 판 물건 점원 금지.
- 안내 판 빨강도 NewStock으로(`Engine.ExploreName`).
- 데이터: `explore03·08·14·19_ye.json` 입구 앞 점원 5명(말풍선은 사용자 예시 말투), 안쪽 `점원#1` 삭제. `shops.json` Mark 삭제.
- 문서: `presentation-explore.md`(새 상품 안내 절), `data-item.md`, `AGENTS.md`, `history.md`.

**문구 선택 대기**: 말을 걸었을 때의 설명 문구는 사용자 선택이 필요해 `shop-advert-talk-drafts.html`에 3판(A 입담/B 담백/C 고을 맥락)을 준비했고, **A판을 임시로 넣어 둠**. 고르시면 `adverts` 문구만 바꾸면 되고 시험이 이름·값 일치를 지킨다.

**수정 전 실패 / 후 통과**: 새 규칙을 실제 데이터에 걸었을 때 수정 전 `TestShopsWithNewStockAreAnnouncedNearTheirDoor`가 6건 실패(03·08·14·14·19×2, 내용은 작업 폴더 기록 `before-shop-issues` 요약: 입구 근처 점원 없음 4건, 19 대화·말풍선에 새 물건 없음 2건) → 수정 후 통과. 새 core 시험(`shop_advert_test.go`)은 수정 후 데이터로만 돌렸다(수정 전 데이터로의 재현은 하지 않음).

**캡처**: `captures.html`(수정 전 HEAD 데이터를 `-root`로 가리켜 같은 방식으로 촬영). 문 앞 점원 말풍선·말 걸기·빨강 안내 판, 시선(SW, 화면 쪽) 확인. 데모가 대화를 자동으로 넘겨 일부 프레임은 글이 덜 나온 상태다.

## 2. 요청 처리 절차

- `tools/request_mcp/requestflow.py`(새, stdlib): 원자적 claim(rename)·무덮어쓰기 이동, 요청별 `progress.md`에 분리 프로세스(heartbeat)가 1분마다 실제 시각 기록(작업자 보고가 오래되면 그 사실을 적음), `begin-test`→`run-test`(시각·명령·종료코드·출력 기록)→`complete`(필수 시험 최근 결과 전부 PASS일 때만 result로), `fail`·`recover`.
- `tools/request_mcp/server.py`: 같은 모듈 사용, `set_status/begin_test/run_test/note_test/complete_request/fail_request` 도구 추가, 대기 중 1분마다 `recover`.
- `~/.claude/skills/request-watcher/SKILL.md` 전면 갱신(단계·우선순위·커밋 금지·승인 규칙 유지). `AGENTS.md` 요청 단락 갱신.
- 자동 시험 `test_requestflow.py` 14개(임시 디렉터리, 실제 heartbeat 프로세스): 이동 경쟁, 이름 충돌, 1분(1초로 축소) 실제 시각 기록, 요청 간 분리, test 중 지속·완료 시 종료, 실패 시 거부 후 수정 재시험 통과, 시험 없음 거부, 비코드 사유, 재시작 recover, failed 시 정지.
- **이 요청 자체가 새 절차로 처리됨**: 요청 md는 process → test(08:27) → result. heartbeat는 08:02:57부터 기록했다(요청 수신 약 08:00~08:02는 이 처리기가 접수한 뒤라 기록이 없음).

## 변경 파일
tools/request_mcp/{requestflow.py,server.py,test_requestflow.py}, ~/.claude/skills/request-watcher/SKILL.md, AGENTS.md,
ver5/internal/content/{stock.go,stock_test.go,explore.go}, ver5/internal/core/{explore.go,shop_advert_test.go}, ver5/internal/gui/explore.go,
ver5/internal/scenegraph/{rules.go,graph_test.go}, ver5/assets/data/shops.json, ver5/assets/scenario/hero/explore03·08·14·19_ye.json,
ver5/docs/{presentation-explore.md,data-item.md,history.md}. (작업 트리에는 다른 세션의 미커밋 변경이 많이 섞여 있고 건드리지 않았다. 커밋 없음.)

## 검증 (모두 실제 실행, 상세 시각은 test-log.md)
requestflow-unit 14개 PASS · go build · go vet · go test content / scenegraph / core 전체 / gui `-run Explore|Shop|Guide|Walk` / cli+session 모두 PASS (content·scenegraph·build·vet은 정리 뒤 재실행해 최근 결과 PASS).
실패·수정 기록: 개발 중 requestflow 시험에서 claim 경쟁(6명이 모두 성공)과 중복 heartbeat 판정 시간차, run-test 인자 파싱 오류 3건을 발견해 고치고 14개 통과로 재시험했다(시험 기록 이전의 개발 단계).

## 남은 한계
- 이미 떠 있는 다른 세션의 옛 request-watch MCP 서버(프로세스 4개)는 재시작 전까지 옛 동작(이동만, heartbeat 없음)이다. 이 파일을 쓰는 새 세션부터 적용; 옛 서버가 가져간 요청은 `requestflow.py adopt <name>`.
- 대화 문구는 임시 A판(사용자 선택 대기). 1장 시작 재고는 안내 대상이 아니라고 정했다(바꾸려면 알려 주세요).
- 19장 철 계열은 업에서만 살 수 있다(이후 대장간 없음): 별개 설계 문제로 남김.
- go test ./... 전체와 gui 전체는 돌리지 않았다(8GiB 맥, 관련 패키지만).
