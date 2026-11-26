# 일회용 장비 소모 조건: 막힘 (blocked), 코드 변경 없음

## 현재 동작 조사
- `ver5/internal/content/content.go`의 `Equipment` 구조체에는 일회용 여부를 나타내는 필드가 없다 (Name, Desc, Slot, Type, Icon, Price, Line, Tier, Bonus, Traits, Learn, Skills, Palette, Owner).
- `assets/data/*.json`에 `낡은`/`모조` 장비나 일회용 표식이 없다.
- grep에서 나온 consumable/소모품은 전부 `Item`(승급 인수 등 소모 아이템, `combat.go`·`api.go`·`gui/party.go`)이며 장비가 아니다.
- 전투 종료·승리 시 장비를 제거하는 코드는 없다. 출진 부대를 장수별로 추적해 승리 시점에 처리하는 훅도 일회용용으로는 없다.

## 판단
일회용 장비 메커니즘이 전혀 없어, 이를 적용하려면 새 데이터 필드, 승리 정산 훅, 출진 판정, 표시까지 새 시스템을 만들어야 한다. 요청 범위상 "큰 새 시스템을 만들지 말고 blocked"로 멈춘다.

## 확인이 필요한 질문
1. 일회용 표식을 Equipment 데이터의 새 필드(예: `OneUse bool`)로 둘지, 아니면 별도 장비 id만 만들지.
2. 승리 정산 위치와 보상 순서·표시(결정 기록상 미정).
3. 증원·예비대의 "출진" 판정(미정).
4. 제거 후 원래 장비 자동 재장착 여부(미정).

테스트는 실행하지 않았다. 변경 파일 없음.
