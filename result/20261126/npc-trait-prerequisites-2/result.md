# NPC 계획 특성의 선행 보완

- 논변의 선행으로 혼란을 지정했다.
- NPC의 현재 레벨 계획과 병종 계획에 나온 특성의 선행 중, 그 장수의 최종 병종 계열 학습 가능 목록에 있는 특성을 보완한다. 재귀 선행을 처리하며, 이미 계획이나 최종 장비가 제공하면 중복하지 않는다. 적·동맹 AI와 부관에 적용하고 플레이어 장수의 지급 방식은 유지한다.
- 저레벨 곽가가 실제 혼란 책략을 사용할 수 있는지, 장비·계획 중복, 병종 계열 외 선행, 연쇄 선행과 부관·플레이어 구분을 검증했다.

## 변경 파일

- `ver5/assets/data/traits.json`: 논변의 선행 설정.
- `ver5/internal/content/spawn.go`: 계획 특성 선행 보완.
- `ver5/internal/core/engine.go`: NPC 및 부관 생성에 적용.
- `ver5/internal/content/trait_prerequisite_audit_test.go`: 기존 전 장수·전 레벨 계획 선행 검사가 실제 보완 후 특성을 검사하도록 입력 보완. 기존 선행 필수 검증은 유지.
- `ver5/internal/content/npc_prerequisites_test.go`, `ver5/internal/core/npc_prerequisites_test.go`: 선행 보완과 실제 생성 회귀 검사.
- `ver5/docs/history.md`: 변경 이력.

## 검증 및 미완료

세부 명령과 결과는 test/npc-trait-prerequisites-2/test-log.md를 참조한다. 특성 선행·계획·장비·실제 NPC 생성 검사는 통과했다. 기존 `TestOfficerReformBatch/personal_traits_do_not_repeat_inherited_class_learning`은 삭제하거나 완화하지 않았으며 기존 데이터 8건에서 실패한다: 순유 수계, 법정 지계, 유비 부관임명, 허사 화계, 유벽 지계, 황보숭 화계, 하후연 저지사격, 번궁 화계. 이 장수들의 특성 배치 변경은 이번 요청 범위에 포함하지 않았다. content 패키지는 통과했고 core 패키지는 이 중복 검사 8건 때문에 실패했다. 따라서 요청은 test 단계에 남아 있다.
