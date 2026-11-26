# 탐험·시나리오 전환 연속성 수정 결과

설계: `design.md`. 사례별 분류(C 42건): `case-classification-C.tsv`. 영상·시트: `captures/` (수정 후 `ch13-farewell-after.mp4`, `ch13-after-sheet-x12-walk-x13.png`; 수정 전 비교 `ch13-before.mp4`, `ch13-before-fade-sheet.png`).

## 한 일
- 엔진(`ver5/internal/gui/explore.go`, `scene.go`, `scenario.go`):
  - 탐험→같은 맵 장면(`adoptScene`): 멀리 있는 배역은 순간이동 대신 걸어서 자리 잡고(첫 줄은 도착 뒤), 0.4칸 안이면 그대로 둠, 탐험이 낀 장면은 앞 노드 비트가 정한 자리(`settledCast`)를 기준으로 함. 13장 조운이 (3,3)으로 되돌아가던 문제 해결.
  - 장면/이벤트→탐험: 같은 맵·시간·사람·자리·밝은 화면이면(`seamless`/`walkSeamless`) 암전 없이 이어받고 카메라·확대만 서서히 맞춤. 사람이 흩어지거나 새로 서거나 맵·시간이 바뀌면 기존 0.5+0.5초 암전 유지.
  - 탐험 복귀 시 주인공 방향만 달라도 이어받아 저장에 반영(기존 시험이 드러낸 가려져 있던 문제).
- 데이터: 13장 `c13-xiaopei` 유비 `Stay`; 25장 `c25-road` 유비 Off+Enter 제거; 23장 `c23-visit1/2`, 24장 `c24-cottage3` 장비·관우 Off+Enter 제거(Stay+Move+Face).
- 문서: `docs/presentation-explore.md`, `docs/presentation-scene.md`, `docs/history.md`. `scenes.Actor.Stay` 주석(At 병용 허용).
- 시험: `internal/gui/explore_continuity_test.go`, `explore_scene_continuity_test.go` 새로 작성. 기존 `TestMansionDialogueReturnsToExploreWithoutErrorSound`는 걷기 자리 잡기를 기다리도록 수정(의도된 동작 변경).

## 사례 결과
- 확정 6건(A1~A4): 모두 해결, 실제 흐름 시험(13·14·23×2·25장) 통과. 24장 셋째 방문은 23장과 같은 데이터 수정+전 장 데이터 시험으로 확인(24장 흐름 재생은 안 함).
- 추가 확인 42건: 해결 18(seamless; 실제 재생 확인은 13장 3건·5장 1건, 나머지는 같은 코드 경로, 6장은 나레이션 마지막 줄을 헤드리스로 못 넘겨 미재생), 정상(의도) 24(맵·시간 경과, 회의 뒤 해산, 8장 현병 추가, 24장 장면 일행이 탐험에 없음). 판단 불가 항목은 없으나 C2·C4 판정은 대사 한 줄과 데이터로 한 것이라 이야기 전체를 읽은 확인은 아님.
- 유지된 의도 전환: 다른 맵 출구(페이드), 자리 재배치 이벤트(14장 칼 이야기 끝 암전), `Cut`, 5장 정체 공개 교체(같은 자리·같은 스프라이트 규칙으로 허용).

## 검증 (실제 결과)
- 수정 전: 새 시험이 13장(암전 38프레임·6명 튐·주인공 1.25칸), 14·25·23·23b에서 실패함을 확인.
- `go vet ./internal/gui ./internal/scenes`: PASS
- 새·관련 시험 묶음: PASS (RF label continuity)
- `go test -p 1 ./internal/gui -run 'Explore|Scene|Story|Walk|Opening|Mansion|Held|Continuity|Scenario|Cast|Stage|Dialogue|Farewell|Chapter'`: PASS (RF label gui-subset)
- `go test ./internal/core`: ok (46s). 중간에 `internal/gui` 전체를 한 번에 돌린 것은 10분 제한에 걸려 실패(느린 `TestWaypointFlagsReachable`에서 시간 초과, 이 변경과 무관한 전체 실행 시간 문제). 전체 gui 통과는 확인하지 못함.
- 게임 캡처: 13장 `_uidemo-vm` 1100프레임; 시트에서 x12→탐험→x13 구간에 암전·튐 없음. 다른 사례와 정상 전환은 영상 대신 프레임 단위 시험으로 확인.

## 한계
- 장면 끝(나레이션) 헤드리스 재생 불가 구간, 6장 등 일부 C1 개별 재생 미확인.
- 걸어서 자리 잡는 동안 대사가 잠시 기다린다(새 동작, 연출 검토 필요하면 알려 주세요).
- 저장소에 다른 세션의 미커밋 변경이 많음. 내 변경만 건드림, 커밋 안 함.
