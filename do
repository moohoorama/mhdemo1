던게 뭐뭐뭐였어?

  Listed 1 directory, ran 8 shell commands

요청 파일 폴더 requests/는 없습니다. 작업 기록은 process/, test/, result/에 있고, 가장 최근 작업은 11월 25~26일 병종·특성 개편입니다.

최근 작업 (11/25~11/26, 대부분 완료)
- 병종 특성 개편: 병종별 특성 풀, 진영별 특성 재조정, 최종안 제안 (result/20261126/class-trait-final-proposal 계열). 특성 이름 정정도 포함 (ironwall-name-correction, trait-name-defense-revision).
- 연노병(반복 쇠뇌): 특성 순서, 장거리 사격, 특성 조회 (repeating-crossbow-*). 밀집사격과 장거리 사격은 아직 논의 중인 제안입니다.
- 전투 계산식: 통합 설계 문서 result/20261126/combat-class-overhaul-consolidated/plan.md. 확정안과 미확정 후보를 구분해 두었고, 게임 코드는 바꾸지 않았습니다. 쇠뇌 피해 공식 제안(proposed-damage-formula-crossbow)도 여기에 속합니다.
- 레벨 50 비교: 방비·중갑·돌진 조합을 공식으로 계산한 예측 (level50-formation-simulation, guard-offense-level50).
- 장수 계획 검사: 개인 계획과 병종 계획의 중복, 선행 조건 검사 (remove-officer-class-overlaps-2, trait-prereq-count-review, progression-inventory-all-20261126).

실패 상태로 남은 것 (사용자 결정 대기)
- guojia-debate-level-2: 곽가 논변을 40레벨로 옮겼지만, 다른 장수 7명의 계획 선행 조건이 맞지 않아 실패. 자동 보완을 철회한 뒤 계획 시점과 병종 계획을 다시 검토해야 합니다.
- npc-trait-prerequisites-2: 현재 데이터에 고유·병종 특성 중복 8건이 있어 전체 검사가 실패. 장수별 특성 배치 수정안을 정해야 합니다.
- trait-prerequisite-correction-proposal: 같은 중복 8건을 복구된 테스트가 잡아냄. 요청 범위는 수정안 제시까지였습니다 (하후연 저지 사격, 허사 화계 등).
- traits-xiaopei-fixes-2: B15 GUI 캡처가 모니터를 인식하지 못해 슬롯 9 실전 검증을 못 했습니다.

상태가 불분명한 것
- process/20261125-waypoint-groups, process/20261125-explore-scene-continuity-audit, process/20261125-explore-scene-audit-*: 상태 기록이 없어 중단된 것으로 보입니다.

git 상태
- 마지막 커밋은 fa57e1d feat(scenario-admin)입니다. 최근 커밋 중 병종 특성 반영은 9a3bd59이고, 그 뒤 49dc757부터 수전·25장 맵과 GUI 수정이 이어졌습니다.
- 작업 트리에는 수정 136개, 삭제 7개, 미추적 170개가 쌓여 있습니다. ver5/assets/data/ 6개 파일(classes, traits, characters 등)과 ver5/internal/core, content, gui Go 코드가 크게 바뀌어 있어, 병종·특성 개편과 11/24~25 스크롤·수전 작업이 커밋 없이 섞여 있을 가능성이 높습니다. 어느 변경이 어느 작업인지는 이번에 분리해 보지 않았습니다.

다음에 할 일
1. 중복 8건의 수정안(장수별 특성 배치)을 결정해 주시면 데이터를 고치고 전체 검사를 다시 돌리겠습니다.
2. 곽가 논변 건은 40레벨 배치를 확정해 주시면 선행 검사를 맞추겠습니다.
3. 미커밋 변경을 작업별로 나눠 커밋할지 정해 주세요. 커밋은 요청이 있을 때만 하겠습니다.

✻ Brewed for 1m 8s · done 오후 9:12
