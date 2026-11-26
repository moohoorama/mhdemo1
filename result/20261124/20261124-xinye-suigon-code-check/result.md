# 신야 수공 본편 적용 여부 확인 결과

## 결론

코드는 커밋되어 있다. 하지만 시나리오 데이터와 에셋은 아직 커밋되지 않았다. 따라서 "적용됨"은 작업 트리 기준으로만 맞고, 커밋 기준으로는 완전하지 않다.

## 근거

- 참조 보고서 `result/20261124/xinye-suigon-code-check/summary.md`: `story25`의 `c25-battle`이 `B25`를 가리키고, `stage25`의 수공 이벤트가 surge와 조홍을 지정한다. GUI 연출(`field_surge_show`, cutscene)이 있고, ch25 두 둑 장면은 `S_baihe_dike`를 쓴다. 관련 테스트 `TestSuigong|TestSurge` 통과.
- 직접 확인 (repo-root 경로 기준):
  - 수공 연출 코드는 커밋 `a880fcf`에 있다(`field_surge.go`, `field_surge_show.go`, `field_surge_test.go`, `uidemo_suigong.go`).
  - `stage25.json`(HEAD)에는 수공 이벤트가 없다. 작업 트리에서만 수정돼 있다(`M`).
  - `ch25.json`도 작업 트리에서만 수정돼 있다(`M`).
  - 수공 에셋 폴더 `ver5/assets/graphics/effects/flood/`는 커밋되지 않았다(`??`).

## 확인하지 않은 것

- 이번 확인에서 새 GUI 캡처나 전투 전체 플레이는 하지 않았다.
- 커밋 기준의 본편 적용 여부는 미확인이다.

## 다음 단계

시나리오 데이터(`stage25.json`, `ch25.json`)와 `flood/` 에셋을 검토해 커밋해야 본편에 적용된 상태가 된다. 커밋 여부는 사용자 승인이 필요하다.
