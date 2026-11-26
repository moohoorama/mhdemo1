# 테스트 기록: remove-officer-class-overlaps-2

요청 파일: /Users/younwookim/orca/mhdemo1/test/remove-officer-class-overlaps-2/remove-officer-class-overlaps-2.md
진행 기록: /Users/younwookim/orca/mhdemo1/process/remove-officer-class-overlaps-2/progress.md

테스트 단계 진입: 2026-11-26 07:23:54 KST(Asia/Seoul, UTC+09:00)

## [overlap-and-prerequisites] FAIL

- 시작: 2026-11-26 07:23:54 KST(Asia/Seoul, UTC+09:00)
- 종료: 2026-11-26 07:24:00 KST(Asia/Seoul, UTC+09:00)
- 명령: `go test ./internal/content ./internal/core -run TestOfficerReformBatch/personal_traits_do_not_repeat_inherited_class_learning|TestOfficerOwnedTraitPrerequisites|TestOfficerPlannedTraitPrerequisites|TestClassPlannedTraitPrerequisites|TestResolveNPCPlannedPrerequisites|TestNPCPlannedPrerequisitesAtSpawn -count=1` (cwd `/Users/younwookim/orca/mhdemo1/ver5`)
- 필수: 예
- 종료 코드: 1

```
--- FAIL: TestOfficerOwnedTraitPrerequisites (1.35s)
    --- FAIL: TestOfficerOwnedTraitPrerequisites/liubei (0.00s)
        trait_prerequisite_audit_test.go:23: 
            	Error Trace:	/Users/younwookim/orca/mhdemo1/ver5/internal/content/trait_prerequisite_audit_test.go:23
            	Error:      	Should be true
            	Test:       	TestOfficerOwnedTraitPrerequisites/liubei
            	Messages:   	benevolence requires deputy_appoint
    --- FAIL: TestOfficerOwnedTraitPrerequisites/huangfusong (0.00s)
        trait_prerequisite_audit_test.go:23: 
            	Error Trace:	/Users/younwookim/orca/mhdemo1/ver5/internal/content/trait_prerequisite_audit_test.go:23
            	Error:      	Should be true
            	Test:       	TestOfficerOwnedTraitPrerequisites/huangfusong
            	Messages:   	firecraft requires fire
FAIL
FAIL	srpg/internal/content	4.086s
ok  	srpg/internal/core	3.314s
FAIL
```

## [overlap-and-prerequisites] PASS

- 시작: 2026-11-26 07:24:29 KST(Asia/Seoul, UTC+09:00)
- 종료: 2026-11-26 07:24:34 KST(Asia/Seoul, UTC+09:00)
- 명령: `go test ./internal/content ./internal/core -run TestOfficerReformBatch/personal_traits_do_not_repeat_inherited_class_learning|TestOfficerOwnedTraitPrerequisites|TestOfficerPlannedTraitPrerequisites|TestClassPlannedTraitPrerequisites|TestResolveNPCPlannedPrerequisites|TestNPCPlannedPrerequisitesAtSpawn -count=1` (cwd `/Users/younwookim/orca/mhdemo1/ver5`)
- 필수: 예
- 종료 코드: 0

```
ok  	srpg/internal/content	3.249s
ok  	srpg/internal/core	2.256s
```

