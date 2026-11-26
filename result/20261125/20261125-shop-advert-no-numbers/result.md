# 결과: 새 상품 안내 대사에서 숫자 제거

## 변경 파일
- ver5/assets/scenario/hero/explore03.json, explore08.json, explore14.json, explore19_ye.json: 점원 `Talk` 문구(말풍선 `Chatter`는 원래 숫자 없음)
- ver5/internal/scenegraph/rules.go: `shop-notice`의 "값·회복량이 대사에 있어야 함" 검사를 삭제하고 "말풍선·설명에 상품 수치 금지"(`productNumber`)로 교체. 새 물건 이름이 말풍선·설명에 있는지, 위치·시선·재방문 상점 규칙은 그대로
- ver5/internal/scenegraph/graph_test.go: 가격/회복량 일치 사례를 숫자 금지 사례(아라비아·띄어쓴 숫자·말풍선·한글 수사 "이천사백 냥"·능력치)로 교체, "한 급 위/한 줌/만만치 않소"는 통과 확인, 상점 가격을 바꿔도 같은 대사가 통과하는 시험 추가
- ver5/internal/core/shop_advert_test.go: 값 포함 요구 삭제, 실제 데이터의 점원 말풍선·설명에 숫자가 없음을 확인(이름 포함·구매 가능 검증은 유지)
- ver5/docs/presentation-explore.md, ver5/docs/history.md
- result/20261125/20261125-shop-advert-no-numbers/shop-advert-talk-drafts-no-numbers.html: 숫자 없는 A/B/C 판본(옛 `shop-advert-talk-drafts.html`의 B·C판에는 숫자가 있어 이 새 판으로 대체, 옛 파일은 건드리지 않음)
- 상품 가격·효과 데이터(shops/items/equipment)와 상점 UI는 변경 없음. 커밋·푸시 없음.

## 문구 전후 (A판 임시 적용, 말풍선은 변경 없음)
- 콩: "콩 한 줌이면 지친 장수도 병력이 400은 돌아오오. 값은 200이오." -> "콩 한 줌이면 지친 장수도 병력이 거뜬히 돌아오오. 싸고 흔한 병량이지."
- 보리: "보리는 병력이 700이나 돌아오니 콩보다 든든하오. 값은 400이오." -> "보리는 콩보다 병력을 훨씬 많이 채워 주니 든든하오."
- 술: "술은 병법치를 30 채워 주오. 최대치를 넘어서도 오르지. 값은 300이오." -> "술은 병법치를 채워 주오. 최대치를 넘어서도 오르지."
- 특급주: "특급주는 병법치를 60 채우오. 흔한 술이 아니라 값은 600이오." -> "특급주는 술보다 병법치를 훨씬 많이 채우오. 흔한 술이 아니라 값도 만만치 않소."
- 만능약: "...해로운 상태는 이 약 하나로 다 푸오. 값은 300이오." -> 뒤 문장 삭제
- 중급 인수: "...쓸 때 고르시오. 값은 2000이오." -> 값 문장 삭제
- 빼갈: "병법치를 120이나 채우니 ... 딱이지. 값은 1200이오." -> "병법치를 크게 채우니 큰 책략을 앞둔 장수에게 딱이지."
- 철검: "...한 급 위인 칼이지. 값은 2400이오." -> "구리검보다 한 급 위인 칼이지."
- 철창: "철창도 구리창보다 한 급 위요. 값은 같은 2400이오." -> "철창도 구리창보다 한 급 위요."
- 대궁: "...시위가 굵은 큰 활이오. 값은 2400이오." -> "...시위가 굵은 큰 활이오."
- 가죽장갑: "...천장갑보다 한 급 위, 값은 2400이오." -> "...천장갑보다 한 급 위요."
- 구리갑옷: "...가죽갑옷보다 한 급 위, 값은 2000이오." -> "...가죽갑옷보다 한 급 위요."
- 가죽옷: "...무명옷보다 한 급 위, 값은 1200이오." -> "...무명옷보다 한 급 위요."
- 업 대장간 첫 줄(병주에서 좋은 쇠가…)은 원래 숫자 없음.
전수 확인: 새 상품 안내는 03 낙양 주점, 08 평원 도구점, 14 서주 도구점·주점, 19 업 대장간 5곳뿐(이전 조사). 모두 안내 유지.

## 시험 (requestflow run-test, cwd ver5, 상세 시각은 test/기록의 test-log.md)
1. `go test -p 1 -count=1 ./internal/scenegraph` [scenegraph-shop]: 처음 FAIL -> 수정 후 PASS
2. `go test -p 1 -count=1 -run 'TestNewStockIsToldOutdoorsAndSoldInside|TestAShopSeenAgainIsNotCriedAgain' ./internal/core` [core-advert]: PASS
3. `go test -p 1 -count=1 ./internal/content` [content-stock]: PASS
4. `go test -p 1 -count=1 -run 'Shop|Explore|Notice|Talk' ./internal/core` [core-shop-explore]: PASS
5. `go build ./...` [go-build]: PASS

## 실패와 수정
- [scenegraph-shop] 실패: 정규식 단위에 `치`를 넣어 특급주 문구의 "만만치"(관용 표현)를 수치로 오탐. 단위를 `냥|푼|퍼센트|%`로 좁히고 "만만치 않소"를 통과 사례로 시험에 추가, 같은 라벨로 재시험 PASS.
- 한계: "천 냥 빚" 같은 관용구도 숫자+냥이라 안내 문구에서는 거부된다(상품 안내에는 쓰지 않음).

## 캡처: 미확인
`go run ./cmd/_uidemo-vm -ui explore -root . -stage "start:c08-walk,exit:거리로,talk:도구점 점원" -frames 900 -at 300,600,880` 1회 실행, `ui: no monitor was found at initializeGLFW`로 실패(이전 작업과 같은 환경 문제). 재시도하지 않음. 문구는 자동 시험이 실제 데이터로 검증.

## 남은 사용자 선택
- 말을 걸었을 때 문구는 숫자 없는 A/B/C 판본(shop-advert-talk-drafts-no-numbers.html) 중 선택 대기(A 임시 적용). 어느 판이든 숫자는 시험이 막는다.
