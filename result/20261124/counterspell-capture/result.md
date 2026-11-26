# 반계(spell_reflect) 화면 재현 결과

요청: 반계가 물리 반격처럼 보이고 책략 효과가 안 보인다는 증상을 실제 게임 화면에서 재현하고 캡처한다. 수정은 하지 않았다.

## 결론

- 재현됨(부분). 화면에서 보이는 것은 두 가지다.
  1. 반계된 책략은 별도의 책략 연출 없이 일반 피해 숫자로만 처리된다. 이벤트 로그에서 되돌린 공격은 `spell-hit`가 아니라 `damage`다(hit 실행: 정원지→관우 111 피해). 대조 실행(반계 특성 없이)에서는 이 111 피해가 사라져, 111은 반계로 되돌아온 것이 맞다.
  2. 반계 발동 뒤 적이 살아 있으면 일기토(관우 vs 정원지)가 이어진다. 칼 궤적과 일기토 연출이 이어져, 화면만 보면 물리 반격이나 근접 대결처럼 보일 수 있다. 캡처 데모의 적이 인접해 있어서 생긴 일이고, 실제 증상이 이 조건에서만 나는지는 확인하지 못했다.
- 빗나간 책략도 되돌아간다: 확인됨. 명중 0% 실행에서 관우의 초열은 `miss`로 끝났는데, 그 직후 정원지→관우 `miss`가 다시 나왔다. 반계 되돌림이 원래 공격 명중 여부를 보지 않고 실행된다는 뜻이다. 소스 근거: `ver5/internal/core/combat.go` 974~976행 조건(`!derived && has(q, reflect) && q.HP>0 && u.HP>0 && !confused(q)`)에는 `hit` 검사가 없다. 같은 함수의 바로 위 블록(957, 961행)은 `hit &&`를 쓴다.
  - 주의: 빗나감 실행은 명중률이 0이라 되돌린 공격도 빗나갔다. 되돌린 공격이 명중했을 때 피해가 나는지는 이 실행으로는 따로 보이지 않는다(이 점은 소스로만 확인).
- 책략 효과가 안 보인다: 화면에서는 초열 명령 글자(화면 위 "초열" 표시)와 불꽃 효과는 보이지만, 되돌린 공격에 대한 별도 불꽃 연출은 찾지 못했다. 되돌린 피해의 연출은 일반 피해 숫자뿐이다. 다만 불꽃이 어느 유닛에 찍히는지 픽셀로 가려서 확인하지는 못했다.

## 사용한 데모

- 기존 `counterspell` 데모(`ver5/internal/gui/uidemo_counterspell.go`, 다른 세션이 만든 미추적 파일)는 그대로 실행되지 않는다. `flame` 스킬이 없어서 `UnavailableSkill`로 실패한다(실제 화계 스킬 키는 `scorch`/초열). 이 파일은 건드리지 않았다.
- 새 데모를 임시 파일로 만들었다(`reflectcap_demo.go.txt`로 보관, 게임 트리에서는 제거). 데모 3종:
  - `reflectcap_hit`: 명중률 100%. 관우(ally, `fire` 특성 추가)가 적 정원지(enemy, `spell_reflect` 특성 추가)에게 `scorch`(초열) 시전. 적은 인접 칸에 배치.
  - `reflectcap_miss`: 명중률 0%, 같은 배치.
  - `reflectcap_control`: 명중률 100%, 적에게 `spell_reflect` 없음(대조).
- 명중률은 책략 명중 범위인 `Rules.SpellFloor/SpellCeil`로 고정했다(`HitFloor/HitCeil`은 물리 명중에만 쓰여 처음 시도는 효과가 없었다).
- 명령: `lockf -t 0 <lock> go run ./cmd/_uidemo-vm -ui reflectcap_<hit|miss|control> -frames 120 -every 4 -out <dir> -mp4 <dir>/<name>.mp4`
- 캡처 시점: 초열 명령은 게임 틱 20에 발행. 프레임 0~116을 4틱마다 PNG로 저장, MP4는 전체 120프레임.

## 이벤트 로그 (데모 실행 출력)

| 실행 | 이벤트 순서 |
|---|---|
| hit | damage 관우→정원지 263 / damage 정원지→관우 111 / spell-hit 관우→정원지 263 / 일기토 / 정원지 사망 |
| miss | miss 관우→정원지 / miss 정원지→관우 / 일기토 / 정원지 사망 |
| control (반계 없음) | damage 관우→정원지 263 / spell-hit 관우→정원지 263 / 정원지 사망 (111 없음, 일기토 없음) |

## 파일 (모두 절대 경로 기준 result/20261124/counterspell-capture/)

- 반계 명중: `hit/frame-0000.png` ~ `hit/frame-0116.png` (30장), `hit/reflect_hit.mp4`
- 반계 빗나감: `miss/frame-0000.png` ~ `miss/frame-0116.png` (30장), `miss/reflect_miss.mp4`
- 대조(반계 없음): `control/frame-*.png` (30장), `control/reflect_control.mp4`
- 잘라 낸 GIF: `hit_reflect_crop.gif`, `miss_reflect_crop.gif` (프레임 12~66, 전투 중앙 400x300)
- 프레임 띠: `hit_strip_f12-66.png`, `miss_strip_f12-66.png`
- 임시 데모 코드 보관: `reflectcap_demo.go.txt`
- 이전 시도 기록(다른 세션): `result-earlier-attempt.md`, 같은 폴더의 `counterspell.mp4`(261바이트, 실패한 빈 파일)

## 겪은 문제

- 디스플레이 꺼짐: 여러 번 `no monitor was found at initializeGLFW`. `UserIsActive`가 0이 되면 실패하고, `caffeinate -u -t 3`로 깨우면 다음 실행이 성공했다. 잠금 해제는 하지 않았다(해제 권한 없음). 같은 증상이 다시 나면 같은 방식으로 재시도해야 한다.
- 실행은 항상 한 번에 하나만 했다(lockf).

## 하지 않은 것

- 게임 코드·데이터 수정, 커밋, 푸시: 하지 않았다.
- 되돌린 책략이 명중했을 때 피해가 어떻게 보이는지 화면으로 확인: 명중률 100%의 반계는 피해가 났지만, 되돌린 공격의 연출은 픽셀 단위로 가려내지 않았다.
- 일기토가 없는 조건에서의 캡처: 데모의 적이 인접해 있어서 반계 후 일기토가 이어졌다. 일기토 없이 반계만 보는 데모는 만들지 않았다.
- 응징·지계 치명 등 다른 항목: 요청 범위 밖.
