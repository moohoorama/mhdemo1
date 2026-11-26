# 테스트 기록: 20261126-proposed-damage-formula-crossbow

요청 파일: /Users/younwookim/orca/mhdemo1/test/20261126-proposed-damage-formula-crossbow/20261126-proposed-damage-formula-crossbow.md
진행 기록: /Users/younwookim/orca/mhdemo1/process/20261126-proposed-damage-formula-crossbow/progress.md

테스트 단계 진입: 2026-11-26 07:21:13 KST(Asia/Seoul, UTC+09:00)

## [damage-formula-arith] PASS

- 시작: 2026-11-26 07:21:14 KST(Asia/Seoul, UTC+09:00)
- 종료: 2026-11-26 07:21:14 KST(Asia/Seoul, UTC+09:00)
- 명령: `python3 -c 
A,D=1300,round(1120*1.05)
assert D==1176
d=A-D
assert d==124
assert abs(d*1+1200-1324)<1e-9          # 일반
assert abs(d*0.5+1200-1262)<1e-9        # 반격 제안 (규칙계수 1, 반격 -50%)
assert abs(d*1.5+1200-1386)<1e-9        # 치명 제안
assert abs(d*1.1+1200-1336.4)<1e-9      # 거리4 원사강화 +10% 제안
assert abs(1324*0.5-662)<1e-9           # 현행 반격
assert abs(1324*1.5-1986)<1e-9          # 현행 치명
assert abs(1324*1.1-1456.4)<1e-9        # 현행 원사강화
print('OK 1324 1262 1386 1336.4 | current 662 1986 1456.4 (A=1300, D=1176 assumed)')
` (cwd `/Users/younwookim/orca/mhdemo1`)
- 필수: 예
- 종료 코드: 0

```
OK 1324 1262 1386 1336.4 | current 662 1986 1456.4 (A=1300, D=1176 assumed)
```

### 2026-11-26 07:21:19 KST(Asia/Seoul, UTC+09:00)

완료: 필수 테스트 1개 모두 최근 결과 PASS. 요청 파일을 /Users/younwookim/orca/mhdemo1/result/20261126/20261126-proposed-damage-formula-crossbow/20261126-proposed-damage-formula-crossbow.md 로 옮김

