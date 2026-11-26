# 연노병 NPC 계획 순서 변경

- `ver5/assets/data/classes.json`: 연노병 NPC 계획에서 저격을 49→50, 관통을 50→60, 원시를 60→70으로 옮겼다. 나머지 계획과 플레이어 학습 목록은 유지했다.
- `ver5/internal/content/trait_prerequisite_audit_test.go`: 궁병·노병·연노병의 저격 학습 목록 상속과 49·50·59·60·69·70레벨 NPC 계획을 검사한다.
- `ver5/docs/logic-traits.md`: 학습 목록과 NPC 계획을 구분해 기록했다.
- 검증: `go test ./internal/content -run 'TestRepeatingCrossbowPlanOrder|TestClassPlannedTraitPrerequisites|TestOfficerPlannedTraitPrerequisites' -count=1` 통과. 변경 파일 diff 공백 검사 통과.
- 별개인 장수·병종 중복 검사 8건은 이전 작업의 미해결 문제로 남는다. 이 작업에서 개인 특성 데이터는 건드리지 않았다.
