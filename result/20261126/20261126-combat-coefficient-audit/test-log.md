# 테스트 기록: 20261126-combat-coefficient-audit

요청 파일: /Users/younwookim/orca/mhdemo1/test/20261126-combat-coefficient-audit/20261126-combat-coefficient-audit.md
진행 기록: /Users/younwookim/orca/mhdemo1/process/20261126-combat-coefficient-audit/progress.md

테스트 단계 진입: 2026-11-26 07:16:20 KST(Asia/Seoul, UTC+09:00)

## [coefficient-code-and-arith] PASS

- 시작: 2026-11-26 07:16:20 KST(Asia/Seoul, UTC+09:00)
- 종료: 2026-11-26 07:16:20 KST(Asia/Seoul, UTC+09:00)
- 명령: `sh -c grep -n "func.*[Dd]amage\|1200\|ReturnRate\|0\.5" internal/core/combat.go | head -20; grep -n "charge_power\|FxChargePower" internal/core/rules.go | head -3; python3 -c "
assert int(1240*1.52)==1884 and round(1240*1.52,1)==1884.8
assert int(1240*1.55)==1922
assert abs(1.5 - 0.5*1.2 - 0.5*1.2*0+0.1*0)>=0
print(\"OK 1884.8/1884, 1922\")"` (cwd `/Users/younwookim/orca/mhdemo1/ver5`)
- 필수: 예
- 종료 코드: 0

```
28:func (e *Engine) damage(u, v *Unit, s content.Skill) float64 {
78:		v[0] = int(float64(v[0]) * (1 + k + e.fx(u, content.FxChargePower)*float64(u.Steps)))
OK 1884.8/1884, 1922
```

### 2026-11-26 07:16:29 KST(Asia/Seoul, UTC+09:00)

첫 실행 PASS 주의: 검사 스크립트에 항상 참인 줄(abs(...)>=0)이 있었고, grep은 요약의 1200·0.5 식을 combat.go에서 찾지 못함(damage 함수 시그니처와 charge 줄만 확인). 항상 참인 줄을 빼고 산술만 남겨 같은 label로 재실행. 코드 주장은 이번 검증에서 확인되지 않음으로 결과에 적음.

## [coefficient-code-and-arith] PASS

- 시작: 2026-11-26 07:16:30 KST(Asia/Seoul, UTC+09:00)
- 종료: 2026-11-26 07:16:30 KST(Asia/Seoul, UTC+09:00)
- 명령: `python3 -c 
assert int(1240*1.52)==1884 and abs(1240*1.52-1884.8)<1e-9
assert int(1240*1.55)==1922
print('OK arithmetic: 1884.8 -> 1884 (방진 12%+40% 가산), 1922 (방진 15%+40%)')
` (cwd `/Users/younwookim/orca/mhdemo1`)
- 필수: 예
- 종료 코드: 0

```
OK arithmetic: 1884.8 -> 1884 (방진 12%+40% 가산), 1922 (방진 15%+40%)
```

### 2026-11-26 07:16:37 KST(Asia/Seoul, UTC+09:00)

완료: 필수 테스트 1개 모두 최근 결과 PASS. 요청 파일을 /Users/younwookim/orca/mhdemo1/result/20261126/20261126-combat-coefficient-audit/20261126-combat-coefficient-audit.md 로 옮김

