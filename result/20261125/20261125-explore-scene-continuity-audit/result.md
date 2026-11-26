# 탐험↔시나리오 전환 연속성 전수 조사 결과

요청: `process/20261125-explore-scene-continuity-audit/20261125-explore-scene-continuity-audit.md`
범위: 우선 **전수 조사**. 이야기·연출·기존 데이터는 하나도 바꾸지 않았다. 추가한 것은 읽기 전용 분석 도구 1개와 켜야만 도는 추적 시험 1개뿐이다.

## 1. 한눈에 보기

| 구분 | 건수 | 내용 |
|---|---|---|
| 조사한 전환 | 154건 | 탐험 노드 44개(탐험 파일 44개·지역 166·사람 841) 기준: 장면→탐험 43, 탐험→장면 74(이벤트·출진 53, 즉시 종료 11, Opening 10), 이벤트→탐험 37 |
| **확정 문제** | 6건 (4묶음) | 같은 장소에서 **페이드 없이** 인물이 제자리에서 튀거나 사라졌다 다시 들어옴 (A1~A4) |
| **시스템 공통 문제 후보** | 이벤트가 끝날 때마다 | 같은 맵에서 이벤트가 끝나 탐험으로 돌아올 때 **아무것도 안 바뀌어도** 0.5초 암전+0.5초 밝아짐 (B). 장면→탐험 같은 맵 17건도 같은 방식 |
| 정상·의도 | 106건 | 맵이 다른 장소 전환(81), `Stay`로 이어받는 이벤트(15), `Cut`(1), Opening(6), 새 인물만 입장(2), 정체 공개 교체(1) |
| 추가 확인 필요 | 42건 | 같은 맵인데 페이드 뒤에서 인물 자리가 바뀌는 장면→탐험·이벤트→탐험 (C) |

제보한 **13장 소패 관청 조운 이별**은 코드·데이터로 원인을 확정했고, 실제 게임 흐름으로 재현했다(§2).

## 2. 13장 제보 사례 재현과 원인

### 흐름 (소패 관청, 맵 `S_xiaopei_office` 한 곳)

1. `c13-x12` 조운 "이제 북평으로 돌아가려 합니다" (`story13.json:412`, 장면 `c13-xiaopei` = `assets/scenes/ch13.json:178`, 맵 `S_xiaopei_office`)
2. 탐험 노드 `c13-wait-zhaoyun` (`story13.json:418`, 파일 `explore13_farewell.json`, 같은 맵 `S_xiaopei_office`)
3. 조운에게 말을 걸면 이벤트 `c13-hear-zhaoyun` (`story13.json:598`)이 **`back` + `Leave`만 있는 노드**라 탐험이 바로 끝난다
4. 이야기는 `c13-x13` "……벌써 떠나시오?" (`story13.json:424`)로 이어진다. 이 노드는 1번과 **같은 장면 `c13-xiaopei`** 에 속한다(장면이 탐험을 가운데 끼고 두 쪽으로 나뉘어 있다)

### 무슨 일이 생기나 (실제 흐름 추적, `traces/`·`ch13-repro/trace.tsv`)

**(가) 1→2: 암전 후 밝아짐만 나온다.** `nextNode`가 다음이 탐험이라 장면을 0.5초 어둡게 하고(`scenario.go:169-170`, `backToWalk` `scenario.go:179`, `scene.go:352-356`, `fadeTime=.5` `scene.go:126`), 탐험이 `layOutExplore`로 새로 깔리며 어두운 채 열려 0.5초 동안 밝아진다(`explore.go:89,117`). 그런데 탐험의 사람 자리(`explore13_farewell.json`)는 장면과 0.35칸 안으로 똑같다. 즉 **바뀌는 것 없이 화면만 꺼졌다 켜진다.** 캡처: `ch13-repro/ch13-x12-to-walk-fade-sheet.png`.

**(나) 3→4: 페이드 없이 자리가 튄다.** 이벤트가 곧바로 탐험을 끝내고 다음 노드 `c13-x13`이 열리면 `enterScene`(`scene.go:145`) → `adoptScene`(`explore.go:146-190`)이 탐험 무대를 이어받는다. 이때 `Stay`가 아닌 배역은 `def.Cast`의 **첫 자리**로 즉시 옮겨진다(`explore.go:169`). 페이드는 없다(`s.fade=0`, 같은 맵이라 `transition.go`도 안 걸림).

| 인물 | 탐험에서 서 있던 자리 | 장면이 정한 자리 | 거리 |
|---|---|---|---|
| 유비(주인공) | (3.2, 1.2) 조운 옆으로 걸어간 자리 | (2.0, 1.0) SW (`ch13.json:183`, 배역 `Stay` 아님) | 1.2칸 |
| 조운 | (3.2, 2.2) | (3.0, 3.0) — 배역 첫 자리. `c13-x12` 비트가 (3,2)로 옮겨 둔 자리는 반영 안 됨 | 0.8칸 |
| 간옹·관우·손건·장비 | 0.2칸 어긋난 칸 중앙 | 정수 칸 | 0.3칸 |

주인공의 탐험 위치는 플레이어가 어디서 말을 걸었느냐에 따라 달라서, 배역에 `At`이 있는 한 **주인공은 늘 튄다.** 인물 방향도 장면 `Face`로 다시 지정된다(말을 건 상대를 향하던 방향이 초기값으로). 같은 패턴의 14장 캡처에서는 카메라 기준도 함께 바뀌어 화면이 한꺼번에 밀리는 것을 확인했다.

원인 요약: ① 탐험이 낀 장면을 두 쪽으로 나눠 놓고 뒤쪽 노드가 **초기 `Cast`를 다시 적용**한다 ② 앞쪽 `x12` 비트로 움직인 자리는 반영되지 않는다 ③ 탐험→장면 이어받기 경로(`adoptScene`)는 페이드를 전혀 두지 않아 순간이동이 그대로 보인다 ④ 반대쪽 장면→탐험은 늘 페이드를 둬서 같은 화면에 암전만 생긴다.

### 캡처

- `ch13-repro/ch13-farewell-actual.mp4`: `c13-xiaopei`에서 시작해 탐험→조운 대화→이어받기까지 실제 게임(`_uidemo-vm`, `-ui explore`) 흐름
- `ch13-repro/ch13-snap-before-after.png`: 이어받기 직전(왼쪽, 탐험 이름판) 직후(오른쪽, 장면) 한 프레임씩
- `ch13-repro/ch13-x12-to-walk-fade-sheet.png`: `x12`→탐험 암전·밝아짐 연속 프레임
- `ch13-repro/trace.tsv`, 비교용 프레임 번호는 데모와 추적 시험이 달라 추적 쪽은 시험 프레임 기준

## 3. 엔진이 전환을 다루는 법 (코드 근거)

| 전환 | 코드 | 페이드 | 자리 |
|---|---|---|---|
| 장면 → 탐험 노드 | `nextNode` `scenario.go:158-175`, `backToWalk` `:179-195`, `layOutExplore` `explore.go:84-143` | 장면 0.5초 암전 + 탐험 0.5초 밝아짐. 맵이 같아도 항상 | 주인공은 장면이 남긴 자리 유지(`explore.go:130-134`), 나머지는 탐험 데이터·플래그대로 새로 배치 |
| 탐험 → 같은 맵 장면(이벤트·`Leave` 뒤·출진 뒤) | `enterScene` `scene.go:137-152`, `adoptScene` `explore.go:146-190` | **없음** | 배역 `Stay`만 그 자리, 나머지는 `Cast`의 첫 `At`·`Face`로 즉시. 배역에 없는 탐험 인물은 그대로 남음. `Off` 배역은 즉시 사라짐 |
| 탐험 → 다른 맵 장면 | `screenKey` `transition.go:26-60`, `watchScreen` `:65` | 0.35초 + 0.35초 (+ 새 장면 0.5초 밝아짐) | 새 장면 첫 자리 |
| 이벤트 끝(`back`) → 탐험 | 위 `nextNode`/`backToWalk` | 0.5초 + 0.5초, 같은 맵이라도 항상 | 주인공 유지, 나머지 플래그대로 새로 배치 |
| `Opening` | `engine.go:516-519`, `enterScene` | 앞 장면이 이미 암전한 뒤 열림 | 어두운 화면에서 자리 잡음 |
| 장면 `Cut: true` | `scenes.go:101`, `transition.go:51` | 같은 맵이어도 새 장면으로 페이드 | 새 장면 자리 |
| 출입구 `Node`(건물 출입) | `explore.go` `updateExplore`의 `w.going` | 걸어 나가며 암전 뒤 이벤트 | 새 장면 |

문서 불일치: `docs/presentation-explore.md:28`은 "이벤트가 끝나면 지역을 플래그로 다시 세운다(같은 맵이면 검은 화면 없이)"라고 쓰지만 실제로는 같은 맵이어도 0.5초 암전이 있다(`captures/ch13-event-end-blink-sheet.png`). `transition.go`의 맵 전환 검정을 말한 것으로 보이나 사용자가 보는 화면은 검게 변한다. 확인이 필요하다.

## 4. 확정 문제 (페이드 없이 순간이동·사라짐)

판단 기준: 탐험의 같은 지역·같은 맵에서 이벤트·`Leave`가 장면을 열어 `adoptScene`이 돌고, `Stay`가 아닌 배역이 있어 위치·방향이 바뀌거나 탐험에 있던 사람이 `Off`로 사라졌다가 `Enter`로 다시 들어오는 경우.

### A1. 13장 소패 관청 조운 이별 (제보 사례) — §2

| 항목 | 위치 |
|---|---|
| 탐험 | `explore13_farewell.json:46` 조운, 지역 `office` |
| 이벤트 | `story13.json:598` `c13-hear-zhaoyun` (back+Leave) |
| 다음 장면 | `story13.json:424` `c13-x13`, `ch13.json:178` `c13-xiaopei` 배역 (`:183` 유비 등) |
| 증상 | 주인공·조운 자리·방향·카메라가 한 프레임에 바뀜(페이드 없음). 앞쪽 `x12`→탐험은 반대로 바뀌는 것 없는데 암전 |
| 재현 | `ch13-repro/`, `traces/` |

### A2. 14장 서주 거리 미축·미방 이야기 (`c14-e-sword`)

- 탐험 `explore14.json:128` 미축 `(8.75,5.25)SE`, `:149` 미방, 지역 `town`(`S_xuzhou_town`)
- 이벤트 `story14.json:475`(scene), 첫 줄 `:480` `c14-sw01`, 장면 `ch14.json:39` 같은 맵. 배역 유비 `(8,7)NW`, 미축 `(8,6)SE`, 미방 `(10,6)SW`가 **모두 `Stay` 아님**
- 추적: 미방 (9.8,5.8)→(10.0,6.0), 미축 (8.8,5.2)→(8.0,6.0), 주인공 (9.8,5.2)→(8.0,7.0), 페이드 0 (`traces/c14-sword.tsv`)
- 캡처 `captures/ch14-sword-snap.mp4`, `ch14-sword-snap-sheet.png`(연속 프레임 3→4에서 세 사람이 한꺼번에 이동)
- 이벤트가 끝날 때 `c14-sw-back`(`story14.json:524`)에서 미축이 다시 1.06칸 다른 자리로 놓이는 것은 페이드 뒤라 C에 둔다

### A3. 25장 소패 관청 제갈량 보고 (`c25-report-out`)

- 탐험 `explore25_report.json:175` 제갈량 `(4,6)small`, `:187` 간옹, 지역 `office`(`S_xiaopei_office`)
- 이벤트 `story25.json:527` `c25-report-out`(back+Leave only) → 첫 줄 `story25.json:58` `c25-yr-00`, 장면 `ch25.json:27` `c25-road` 같은 맵
- 배역 유비가 `Off`+`Enter (2,6)`이라 **주인공이 있던 자리에서 사라졌다 문가에서 걸어 들어온다**, 간옹 (0.8,1.8)→(1.0,3.0) 1.2칸, 제갈량 (1.8,2.8)→(2.0,3.0)
- 추적 `traces/c25-report.tsv`(f233), 캡처 `captures/ch25-report-snap.mp4`

### A4. 23·24장 융중 방문: 이미 있던 장비·관우가 사라졌다 다시 들어옴

- 23장 첫 방문 `explore23.json:329` 장비@before `(21,9)small` = (10.2,4.2)칸에 서 있음 → 이벤트 `story23.json:228`, 첫 줄 `:233` `c23-a01`, 장면 `ch23.json:217`은 장비를 `Off`+`Enter (10,3)`(0.4초 뒤)로 쓴다 → 서 있던 장비가 즉시 사라졌다가 1.3칸 옆에서 다시 걸어 들어온다. 이벤트가 끝나면 (5,3)에서 문 앞 (10.2,4.2)로 암전 뒤 순간이동한다(`traces/c23-longzhong.tsv` f268/f284, f756)
- 23장 둘째 방문 `explore23b.json:270` 관우@before, `:302` 장비@before, 이벤트 `story23.json:331` `c23-b01`, 장면 `ch23.json:250`: 같은 패턴
- 24장 셋째 방문 `explore24.json:540` 관우, `:553` 장비(`If` 없음, 늘 서 있음), 이벤트 `story24.json:732` `c24-c01`, 장면 `ch24.json:137`: 같은 패턴
- 캡처 `captures/ch23-longzhong-pop.mp4`

참고: 정체 공개(5장 주막, `c05-c2` `story05.json:553`)는 삿갓 쓴 사내 `Off`와 조조가 **같은 자리·같은 스프라이트(`caocao_hat_noncombat`)** 라 교체가 보이지 않아 정상으로 본다.

## 5. 시스템 공통 문제 후보 B: 이벤트가 끝날 때의 암전

- 같은 맵으로 돌아오는 이벤트→탐험 25건 **모두** 0.5초 암전+0.5초 밝아짐이 나오고, 그중 16건은 **인물이 하나도 안 움직이는데도** 그렇다(`Stay` 배역만 있는 1·2·3·5·13·22장 등 이야기 이벤트, 막사 출입 일부)
- 캡처: 13장 피난 상인 이야기(`captures/ch13-event-end-blink.mp4`, `ch13-event-end-blink-sheet.png`: 대사가 끝나면 거리 전체가 검게 꺼졌다 켜짐)
- 코드: `nextNode`가 다음 노드가 `back`이면 무조건 `s.leaving=true`(`scenario.go:169-170`, `backToWalk` `:179-195`). 같은 맵·`Stay` 배역만이어도 구분하지 않는다
- 사용자 기준의 "같은 장소 연속 화면에서 불필요한 페이드"에 해당하는지는 **사용자 판단이 필요하다**(요청 본문은 탐험→시나리오를 말한다). 반대 방향은 의도적으로 사람 배치를 다시 세우는 암전일 수 있다

## 6. 정상·의도된 연출 (요약)

전체 표는 `edge-classification.tsv`(분류·장·종류·파일:줄·최대 이동 칸).

- **맵이 다른 장소 전환 81건**: 건물 출입(출구 `Node`·`To`), 출진 전 장소 이동(출진 질문 뒤 행군·전투 장면), 거리→관아(13장 `c13-council`, 21장 `c21-alarm`) 등. `transition.go`가 0.35+0.35초 페이드를 건다. 같은 맵 이름만으로 연속이라 보지 않았고, 장소가 바뀌는 쪽(건물 안팎·행군)만 제외했다. 13장 `c13-council`·21장 `c21-alarm`처럼 **거리에서 말을 걸었는데 문을 지나지 않고 관아 장면으로 넘어가는** 경우는 장소 이동 연출로 본다(`back`의 `Area`가 관아).
- **`Stay`로 이어받는 이벤트 15건**: 1~3·5·13·22장 등 마을 소문·대화. 자리·방향이 그대로다. 현재 데이터가 연속성을 **대체로 잘 지키고 있다**는 증거로, 위반은 A1~A4에 모여 있다.
- **새 인물만 `Off`+`Enter`로 들어오는 이벤트**: 1장 출진(정찰병), 11장 전령 등. 탐험에 없던 사람이 걸어 들어오므로 정상.
- **`Cut: true`** 27장 강하 출진(`ch27.json`, `story27.json:177`): 새 장면으로의 페이드를 데이터가 명시.
- **같은 맵 Opening 6건**(1·3·5·19·20·27장): 앞 장면이 암전한 뒤 열려 자리 정리가 보이지 않는다. 5장 추적으로 확인(`traces/ch05-opening.tsv`: 시작 `fade=1.00`). 나머지는 같은 코드 경로(`engine.go:516-519`)로 읽었고 별도 추적은 하지 않았다. 20장 추적(`traces/ch20-opening.tsv`)은 Opening까지 가지 못하고 앞 장면에서 끝났다.

## 7. 추가 확인 필요 (C: 같은 맵, 페이드 뒤에서 자리가 바뀜 / 변화 없는 페이드)

전수 목록은 `edge-classification.tsv`에서 분류 `추가 확인 필요(...)`.

**C1. 장면→탐험, 같은 맵, 자리는 같은데 0.5+0.5초 페이드 (4건)**: 6장 `c06-b7`→`explore06`(33명 모두 0.0칸), 8장 `c08-p12`→`explore08`(0.35), 13장 `c13-x12`→`explore13_farewell`(제보), 24장 `c24-s08`→`explore24`(0.0)
**C2. 장면→탐험, 같은 맵, 페이드 뒤 자리가 크게 바뀜 (13건, 같은 간선이 앞 노드가 둘이라 중복된 것 포함)**: 7장 `c07-d5`(5.8칸), 9장 `c09-19`(4.3), 10장 `c10-19`·`c10-17`(8.8), 13장 `c13-f01b`→`explore13`(7.2), 20장 `c19-k09`(10.4)·`c20-envoy-03`(2.4), 21장 `c21-pass-04`(16.1), 22장 `c21-e02`(3.3), 24장 `c24-e29`(5.7), 27장 `c26-c07`(5.8)·`c27-a08`(5.1). 같은 자리에서 이어지는 시간인지(이튿날·이동 뒤) 이야기 맥락 확인이 필요하다. 예: 13장 `c13-farewell`은 "소패 성문 앞 · 이튿날"이라 정상 가능성이 높다
**C3. 이벤트→탐험, 같은 맵, 자리 변화 없는 페이드 (16건)**: §5와 같다
**C4. 이벤트→탐험, 같은 맵, 페이드 뒤 자리 변화 (9건)**: 3장 `c03-m-back`(2.9칸), 13장 `c13-c-back`, 14장 `c14-sw-back`, 15장 `c15-a-back`(2.9), 17장 `c17-council-back`(2.8), 18장 `c18-news-back`(4.0), 23장 `c23-alarm-back`(4.0)·`c23-visit1/2-back`(5.4~5.5)

대부분 사람이 걸어 나가는 장면 없이 자리가 바뀌므로, 건물 안 회의 뒤 흩어지는 것으로 읽을 수 있는지 사용자 판단이 필요하다.

## 8. 자동 검증 설계 제안

### 8.1 콘텐츠(정적) 시험: `internal/gui/explore_scene_continuity_test.go`

기존 `TestExploreAreasCanBeWalked`·`TestScenesData`와 같은 방식으로 `content.Load`와 `scenes.Load`를 읽어 `_contaudit`와 같은 간선을 만든다.

1. **간선 구성**: 모든 `explore` 노드마다
   - (장면→탐험) 탐험 노드 앞에서 이어지는 대사 노드(`set/grant/join/leave/branch/scene`는 건너뜀, 분기는 모두)의 장면
   - (탐험→장면) `People.Event`, `Sortie`, `Exits.Node`, `Opening`의 첫 대사 노드. 이벤트가 `back+Leave`뿐이면 탐험 노드 `After`의 첫 대사 노드(이 간선을 따로 만들어야 13·25장이 잡힌다)
   - (이벤트→탐험) 이벤트가 닿는 모든 `back`과 그 앞 마지막 대사 노드
2. **같은 장소 판정**: `scene.Map == area.Map`이고 `scene.Cut`이 아니고 `back.Area`가 다른 지역을 가리키지 않는 간선만 비교한다. 건물 출입·맵이 다른 장면·`Cut`는 자동 제외(이름이 같아서가 아니라 데이터로 판정). 같은 맵 안에서 장소를 옮기는 연출은 예외 표로 명시한다.
3. **규칙 R1(탐험→장면, 페이드 없음)**: 배역 중 탐험에 서 있는 사람(이름·`#` 규칙으로 매칭, `If`가 이벤트 `If`와 모순되는 자리는 제외)은 `Stay`이거나 `|Cast.At − 탐험 자리| ≤ 0.4칸`이고 `Face`도 같아야 한다. **주인공은 `Stay`이거나 배역에 없어야 한다**(자리를 정적으로 알 수 없으므로 좌표 비교 불가). 탐험에 서 있는 사람이 `Off`이면 위반(`Off` 후 `Enter`는 새 인물만 허용). 장면에만 있는 인물은 `Off`+`Enter`이거나 `Stay`가 아니면 위반(허공에서 나타남)
4. **규칙 R2(장면이 탐험을 끼고 이어짐)**: 같은 장면 안에서 탐험 노드를 사이에 둔 뒤쪽 노드(13장 `c13-x13`)가 첫 자리로 되돌리지 못하게, 탐험을 낀 장면은 뒤쪽 모두 `Stay`이거나 장면을 둘로 나눈 것이어야 한다
5. **규칙 R3(장면→탐험, 같은 맵)**: 비트를 끝까지 적용한 마지막 자리(`Place·Enter·Move·Exit·Face`; `Via`는 끝점만)가 탐험 자리와 0.4칸 안이면 "연속"으로 분류하고, 연속인데 페이드가 있는 간선은 엔진이 건너뛸 수 있게 되기 전까지는 **허용 목록(이유 필수)** 에 둔다. 허용 목록은 낡으면 실패(더 이상 필요 없는 항목은 지우게)해서 목록이 썩지 않게 한다
6. **허용 목록 표현**: 시험 안 `map[edgeKey]string{이유}`. 이유 문자열이 비면 실패. 이유 예: `"이튿날"`, `"일행이 흩어짐"`, `"정체 공개: 같은 자리 교체"`
7. **오탐 방지**: `Move`/`Patrol` 사람은 자리 비교에서 빼고 `Off` 규칙만 적용, `If`·`party:`·`flag:` 조건이 갈리는 변형은 모순 검사 후 남은 변형 모두와 비교, 소수점 칸 중앙 어긋남 0.35칸 허용(현재 데이터가 의도해 맞춘 값), 엑스트라 `#`는 이름 매칭, 비트가 첫 줄 앞에서 곧바로 걸어 들어오게(`Enter`) 한 인물은 R1 제외

### 8.2 실행(회귀) 시험: 13장 재현

`internal/gui/explore_test.go`의 `TestMansionDialogueReturnsToExploreWithoutErrorSound`와 같은 틀로, `NewDemo("../..", DemoOptions{Name:"explore", Stage:"start:c13-wait-zhaoyun"})` → 탐험이 깔린 뒤 주인공을 조운 곁으로 걷게 하고 `talk` 명령 → `c13-x13`이 열릴 때까지 프레임을 돌리며 **한 프레임에 배우가 0.6칸 넘게 움직이는지**, `scene.fade`가 0으로 유지되는지를 검사한다. 지금은 유비(1.2칸)·조운(0.8칸) 때문에 **실패해야 정상**이며, 고친 뒤(조운 이벤트가 장면을 `Stay`로 이어받거나 장면을 분리) 통과해야 한다. `x12`→탐험도 마찬가지로 `scene.leaving`이 켜지지 않거나 자리가 바뀌지 않는지를 같은 틀로 검사한다. 이번에 넣은 `TestTraceExploreContinuity`는 이 시험을 만들 때 그대로 쓸 수 있는 틀이다.

### 8.3 한계와 오탐

- 주인공의 탐험 위치는 플레이어가 정하므로 좌표 비교 불가 → `Stay` 규칙으로만 검증
- 플래그 분기: 모든 조건 조합을 풀지 않고 `flag:`·`!flag:` 모순만 걸러 낸 뒤 가능한 변형과 모두 비교한다. 엔진 `Visitors()`로 실제 플래그 상태를 만들어 비교하면 정확하나 시험이 느려진다
- 카메라·확대·진행 속도는 정적으로 비교하지 못한다 → 실행 시험에서 카메라 목표 점프(프레임당 이동량)와 확대 배율 변화를 따로 본다
- 페이드는 데이터에서 읽는 것이 아니라 엔진 규칙이므로, 엔진이 "같은 장소·같은 자리는 페이드 생략"을 지원하기 전에는 R3를 허용 목록으로만 관리한다

## 9. 조사 범위와 한계

- 범위: 시나리오 `hero`의 모든 챕터(`25v·26v·27v` 대안 포함), 탐험 노드 44개·지역 166·사람 841·사람 이벤트 24·출진 25·`Node` 출구 33. `assets/scenes/ch*.json` 전체.
- 비교는 **정적 분석(`_contaudit`) + 의심 사례 5건의 실제 흐름 추적·캡처**(13장 제보, 13장 이야기 끝, 14장, 23장, 25장) + Opening 2건 추적이다.
- 실제 게임으로 재현하지 못했거나 하지 않은 것: 나머지 C 항목(장면→탐험·이벤트→탐험 자리 변화) 약 40건은 데이터 값만 확인했고 캡처하지 않았다. 불러오기로 이벤트 중간에 들어가는 경로(`layOutWalkFor`), 전투 직후 탐험 시작, 상점·부대 창은 조사하지 않았다.
- 정적 분석 한계: 비트는 `Place/Enter/Move/Exit/Face` 끝점만 계산(`Via`·속도·`After` 무시), 사람 `If`는 모순만 걸러 모든 변형을 비교(`conditional` 표시), `Patrol/Wander` 사람은 자리 불확정(`walker` 표시), 주인공 자리는 불확정, 이벤트 안의 분기는 모두 따름, 카메라·확대는 보지 않음.
- 맵 이름이 같다고 연속이라 단정하지 않았고, 좌표가 다르다고 무조건 오류로 분류하지도 않았다: 의도된 장소 이동(건물 출입·출진 행군)·`Cut`·정체 공개 교체·새 인물 입장은 별도로 뺐다.
- 기준 파일: 조사 시점의 작업 트리(13·14·25장 이야기·탐험·장면 파일 다수가 미커밋 수정 상태)를 그대로 읽었다.

## 10. 실행한 명령과 검증 (그대로)

```
cd ver5
go build ./...                                                   # 성공
go vet ./cmd/_contaudit ./internal/gui                          # 성공(출력 없음)
gofmt -l cmd/_contaudit internal/gui/explore_continuity_trace_test.go   # 정리 후 출력 없음
go run ./cmd/_contaudit > result/.../transitions-full.tsv        # 653행
go run ./cmd/_contaudit -summary > result/.../transitions-summary.tsv   # 154행
go test -p 1 ./internal/gui -run 'TestTraceExploreContinuity|TestMansionDialogueReturnsToExploreWithoutErrorSound|TestExploreAreasCanBeWalked|TestHeldStoryKeepsTheWalkAway' -v
    # SKIP TestTraceExploreContinuity(환경변수 없음), PASS 나머지 3개, ok 4.9s
SRPG_TRACE_STAGE="start:c13-xiaopei,talk:조운" SRPG_TRACE_FRAMES=900 SRPG_TRACE_OUT=... go test -p 1 ./internal/gui -run TestTraceExploreContinuity -v   # 13장 추적
(같은 방법으로 c13-walk 피난 상인, c14-walk 미축, c23-walk 동자, c25-report-walk 제갈량, c19-mangdang, c05-walk-town)
lockf -k /private/tmp/claude-501/scratch/capture.lock caffeinate -d -i go run ./cmd/_uidemo-vm -ui explore -stage "..." -frames N -mp4 ...   # 5회, 직렬
```

전체 `go test ./...`는 돌리지 않았다(메모리 8GiB·직렬 원칙). 저장소에 이미 미커밋 변경이 많아 다른 시험의 결과는 이 조사와 무관하다.

## 11. 변경·생성 파일

저장소(코드·문서):
- `ver5/cmd/_contaudit/main.go` (새, 읽기 전용 임시 도구)
- `ver5/internal/gui/explore_continuity_trace_test.go` (새, `SRPG_TRACE_STAGE`가 없으면 skip)
- `ver5/docs/history.md` (이번 조사 항목 추가)

결과 폴더(`result/20261125/20261125-explore-scene-continuity-audit/`):
- `result.md`(이 파일), `edge-classification.tsv`(분류), `transitions-full.tsv`·`transitions-summary.tsv`(원자료)
- `ch13-repro/`: 영상, 전후 프레임, 암전 연속 프레임, 추적 TSV
- `captures/`: 13장 이야기 끝 암전, 14장, 23장, 25장 영상과 연속 프레임
- `traces/`: 사례별 추적 TSV

이야기·연출·데이터는 수정하지 않았다. 사용자의 후속 결정(A1~A4를 어떻게 고칠지, B의 암전을 줄일지, C를 어디까지 볼지)을 기다린다.
