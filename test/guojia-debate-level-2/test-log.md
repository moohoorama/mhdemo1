# 테스트 기록: guojia-debate-level-2

요청 파일: /Users/younwookim/orca/mhdemo1/test/guojia-debate-level-2/guojia-debate-level-2.md
진행 기록: /Users/younwookim/orca/mhdemo1/process/guojia-debate-level-2/progress.md

테스트 단계 진입: 2026-11-26 07:26:08 KST(Asia/Seoul, UTC+09:00)

## [actual-plans] FAIL

- 시작: 2026-11-26 07:26:08 KST(Asia/Seoul, UTC+09:00)
- 종료: 2026-11-26 07:26:15 KST(Asia/Seoul, UTC+09:00)
- 명령: `go test ./internal/content ./internal/core -run TestOfficerReformBatch/personal_traits_do_not_repeat_inherited_class_learning|TestOfficerOwnedTraitPrerequisites|TestOfficerPlannedTraitPrerequisites|TestClassPlannedTraitPrerequisites|TestOfficerSpellSpecialties -count=1` (cwd `/Users/younwookim/orca/mhdemo1/ver5`)
- 필수: 예
- 종료 코드: 1

```
--- FAIL: TestOfficerPlannedTraitPrerequisites (0.83s)
    --- FAIL: TestOfficerPlannedTraitPrerequisites/fangong (0.00s)
        trait_prerequisite_audit_test.go:44: level 16 class brawler trait firecraft requires fire
    --- FAIL: TestOfficerPlannedTraitPrerequisites/huangfusong (0.00s)
        trait_prerequisite_audit_test.go:44: level 16 class light_infantry trait firecraft requires fire
    --- FAIL: TestOfficerPlannedTraitPrerequisites/xusi (0.00s)
        trait_prerequisite_audit_test.go:44: level 1 class brawler trait firecraft requires fire
    --- FAIL: TestOfficerPlannedTraitPrerequisites/fazheng (0.00s)
        trait_prerequisite_audit_test.go:44: level 16 class scholar trait earthcraft requires earth
    --- FAIL: TestOfficerPlannedTraitPrerequisites/liupi (0.00s)
        trait_prerequisite_audit_test.go:44: level 16 class bandit trait earth_boost requires earth
    --- FAIL: TestOfficerPlannedTraitPrerequisites/xunyou (0.00s)
        trait_prerequisite_audit_test.go:44: level 1 class scholar trait flood requires water
    --- FAIL: TestOfficerPlannedTraitPrerequisites/liubei (0.00s)
        trait_prerequisite_audit_test.go:44: level 31 class heavy_infantry trait benevolence requires deputy_appoint
FAIL
FAIL	srpg/internal/content	3.158s
ok  	srpg/internal/core	1.928s
FAIL
```

## [guojia-and-overlap] PASS

- 시작: 2026-11-26 07:26:33 KST(Asia/Seoul, UTC+09:00)
- 종료: 2026-11-26 07:26:38 KST(Asia/Seoul, UTC+09:00)
- 명령: `go test ./internal/content ./internal/core -run TestGuojiaDebateFollowsConfusion|TestOfficerReformBatch/personal_traits_do_not_repeat_inherited_class_learning|TestOfficerOwnedTraitPrerequisites|TestClassPlannedTraitPrerequisites|TestOfficerSpellSpecialties|TestPrerequisitesGateLearning -count=1` (cwd `/Users/younwookim/orca/mhdemo1/ver5`)
- 필수: 예
- 종료 코드: 0

```
ok  	srpg/internal/content	2.517s
ok  	srpg/internal/core	2.181s
```

### 2026-11-26 07:26:55 KST(Asia/Seoul, UTC+09:00)

실패 상태: 자동 보완 철회 후 다른 장수 7명의 계획 선행 조건이 충족되지 않는다.
다음 조치: 개인 계획 시점과 병종 계획을 검토하여 전 장수 검사를 통과시킨다.

