# 궁기병 최소 사거리와 근접사격 특성 - 결과

## 변경 파일 (ver5/)
- assets/data/classes.json: 궁기병·노기병 MinRange 1 -> 2 (최대 3 유지), 두 병종 풀에 close_shot 추가
- assets/data/traits.json: close_shot(근접사격, Cost 800, Effects close_shot:1)
- internal/content/vocab.go: FxCloseShot 등록
- internal/core/rules.go: rangedBand에서 close_shot이 있으면 원거리 병종 최소 거리 1
- internal/core/close_shot_test.go (신규), internal/content/class_pools_test.go (풀 기대값)
- docs/logic-combat.md, docs/logic-traits.md, docs/history.md
- 설계: design.md (같은 폴더)

## 설계 요약
모든 기본 공격 사거리는 rangedBand 한 곳을 지나므로(공격·반격·협동·AI 합법 행동·이동 후 사격 칸·GUI 사거리 표시) 그곳 한 줄로 적용. 원거리 판정은 병종 기준이라 거리 1에서도 원거리 피해·반격·명중·애니메이션 그대로. 특성은 traits()가 주장+생존 부관 id를 집합으로 합치므로 양쪽 보유 시 중복 없음, 부관 사망/해임 시 즉시 소멸. 근접 병종은 rangedBand의 원거리 분기에 들지 않아 사격 능력이 생기지 않음. 책략(skillBand)·기사(mounted_range) 경로는 불변. 노병·발석거 등은 부관 공유로 적용.

## 테스트 (RF run-test, 라벨별, 모두 ver5에서 실행)
- core-closeshot: go test -p 1 -v -run CloseShot ./internal/core PASS (특성 없음/주장/부관/중복, 사망·해임, 궁기병·노기병·노병·발석거·궁병 vs 경보병·창병·경기병, 거리 1·최소·최대·밖, 실제 공격·반격, AI LegalActions, 비용 800·학습 풀)
- build: go build ./... PASS
- vet: go vet ./internal/core ./internal/content ./internal/ai ./internal/gui PASS
- content: go test -p 1 ./internal/content PASS
- core-all: go test -p 1 ./internal/core PASS (93.9s)
- ai: go test -p 1 ./internal/ai PASS
- gui-unit: go test -p 1 -run 'TestQuickAttack|Look|Shoot|Art' ./internal/gui PASS (재실행)
- gui-targeted: 같은 명령 PASS

## 실패와 수정
- build/vet/content 첫 실행: 종료 코드 127. 원인: run-test에 명령을 한 문자열로 넘긴 내 호출 실수. 수정: argv로 분리해 같은 라벨 재실행, PASS.
- gui-unit 첫 실행(go test -p 1 ./internal/gui 전체): signal: killed(349s, 8GiB 메모리로 추정). 코드 실패 아님. 수정: 사거리·사격 관련 테스트만 -run으로 재실행해 PASS. gui 패키지 전체 테스트는 이 환경에서 끝까지 확인하지 못함.

## 확인하지 못한 것
- 실제 게임 화면 캡처(사거리 표시·인접 사격): `go run ./cmd/_uidemo-vm -ui ranges -frames 30 -at 10 -out ...` 한 번 시도, 오류 `ui: no monitor was found at initializeGLFW` (exit status 1). 재시도하지 않음. 미검증. GUI 사거리 표시는 AttackRange(rangeFor)를 쓰는 기존 경로를 그대로 쓰며, 엔진 테스트로 대역 값만 확인함.
- gui 패키지 전체 테스트(위).
- 밸런스: 자동전투/스테이지 밸런스 영향은 측정하지 않음(궁기병 계열 최소 사거리 상향의 영향 가능성).
- 기존 캐릭터 데이터에 이미 close_shot을 가진 장수는 없음(신규 특성). 기사(mounted_range) 기병은 근접사격 대상이 아님(의도적으로 불변).
