# 시나리오 웹 어드민 + 경량 맵 렌더러 (2026-11-24)

## 만든 것
| 경로 | 내용 |
|---|---|
| `ver5/internal/maplight/` | ebiten 없는 순수 Go 맵 렌더러. `scenemap.Build` 계획을 `image.RGBA`에 합성(지형·벽·소품·장식·홍수·그림자·풀 색조). 지도 전체 캔버스를 한 장으로 그림 |
| `ver5/internal/scenegraph/` | 시작 노드부터 첫 전투까지의 그래프(구역·출구·이벤트·출진·전투)와 검증(없는 노드/구역/지도, 닫힌 길, 플래그 세움·읽음 불일치, 걸을 수 없는 칸, 도달 못 한 이벤트) |
| `ver5/cmd/scenario-admin/` | gin 서버 + React(htm) 웹 어드민. `/api/graph`, `/api/vertex`, `/api/map.png`, `POST /api/reload` |
| `ver5/internal/gui/uidemo_mapcanvas.go` | 비교 기준용 ebiten 캔버스 캡처 데모 (`-ui mapcanvas -stage S_zhuo`) |
| `ver5/cmd/_maplight/` | 임시 비교 도구 (렌더 + 기준 대비 차이 이미지) |

## 실행
```
cd ver5 && go run ./cmd/scenario-admin -addr 127.0.0.1:18765   # http://127.0.0.1:18765
```
(18080 포트는 ssh 터널이 쓰고 있어 피함. React·htm은 unpkg CDN에서 받음.)

## 경량 렌더 vs ebiten (탁군 S_zhuo, 1760×1072)
- 크기·정렬 일치. 평균 차이 3.58, 차이 큰 픽셀 1.65%.
- 남은 차이: 나무 잎 흔들림 프레임, 낮 조명(shadeScenery)의 미세한 색 차이. 사람·셰이더 효과는 없음.
- 이미지: `light_S_zhuo.png`(경량), `ebiten_S_zhuo.png`(기준), `light_S_zhuo.png.diff.png`(빨강=큰 차이)

## 어드민 (c01-walk → B01)
14곳·23길, 검증 오류 0·경고 0. 구역을 누르면 지도 전체 + 출구·사람·표지 표시(걸을 수 없는 칸은 빨강), 이벤트는 대사·세우는 플래그, 전투는 승패 조건.

## 알아둘 것
- `go.mod`/`go.sum`에 gin v1.9.1과 의존성이 추가됨.
- 정적 검증만 함. 전투 이긴 것으로 치고 엔진을 돌려 보는 검증(autoplay 활용)은 아직.
- 지도 렌더는 눈·안개 등 `Snow` 구역 미지원, 애니메이션 첫 프레임.
- 커밋하지 않음.

## 2차: 개선안 1~5 + 스테이지 점프 반영
- 화면: 왼쪽 목차(장·전투 점프, 선택한 장 아래에 따라가기 단계), 가운데 그래프(클러스터), 오른쪽 고정 상세 패널.
- 대사: 초상 + 한글 이름 + 말풍선, 지문은 이탤릭, 혼잣말은 말풍선 목록. 지도 위 표시에 마우스를 올리면 대사 첫 줄 툴팁.
- 이름: 사람은 한글 이름·도트 스프라이트(`/api/sprite`)·초상(`/api/portrait`), 이벤트는 장면 제목 + "누구에게 말 걸면", 조건은 "「…」 이후"처럼 한글 풀이. id는 작게 병기.
- 그래프: 탐험 구역을 허브 + 가지 클러스터로 묶고, 이벤트 열(레인), 이어지는 이야기 열로 배치(`scenegraph/layout.go`). 복귀선은 선택 시에만. 확대/축소/맞춤.
- 따라가기(`scenegraph/follow.go`): 플래그와 출구 조건대로 플레이 순서를 시뮬레이션해 번호를 붙임. 못 가는 이벤트는 경고, 출진이 안 열리면 진행 불가 오류.
- 점프: `/api/outline`(장·전투), `?start=&battles=N&assume=auto`. "이후 펼치기"로 다음 전투 뒤까지, 장 점프 시 앞 장에서 세운 플래그는 가정으로 이어받음.
- 전투가 세우는 플래그(전투 이벤트·부목표)도 세우는 곳으로 인정.

## 3차: 시나리오 규칙 (go test로 고정)
- `internal/scenegraph/rules.go`: exit-return(오류), silent-npc·named-mute·sortie-count·whereabouts(경고, 소재 기록 모순은 오류).
- 데이터: `ExploreExit.OneWay`, `Explore.Whereabouts`, `Campaign.CoreAllies`(scenario.json: 관우·장비·제갈량·조운·간옹·미축·손건).
- 수정: 24·25장 신야성 전용 주점·도구점(`xinye_tavern`, `xinye_tools`), 11장 평원 출구 OneWay.
- 테스트: 출구 돌아옴 0건 강제, Whereabouts 빈 곳은 `testdata/whereabouts_baseline.txt`(89건)에서 줄어들기만 허용.
- `make scenario-admin`.

## 4차: 규모·인원 규칙
- `ExploreArea.Scale`(hamlet 3~5, village 6~9, city 10~15, metropolis 16~22, camp 8~16), `PopulationNote`(범위를 벗어난 이유).
- 100칸 이상 지도는 Scale 필수, 지도 칸 수와 2단계 이상 어긋나면 경고. 1~21장은 기존 인원에 맞춰 단계를 정했고(기준 장), 22장 이후는 지도 크기로 정함.
- 현재 23건: 22장 이후 18건(양양성 8, 신야성 7, 강하·장판·한진·강릉 등), 앞 장 5건(연합 진영 3, 공손찬 진영, 망탕산 산기슭). 기준선 `testdata/population_baseline.txt`.
