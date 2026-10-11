# 일기토 연출과 감상 모드

일기토는 이야기와 장수 성격에 맞춘 짧은 전투극이다. 시간에 따른 상태를 `internal/duel`에서 계산하고 `internal/duelview`가 기존 .spr와 수묵 초상으로 그린다. 기본 재생은 기존보다 정확히 2배 빠르며 본편·감상 모드에 함께 적용한다. 이름은 굵게, 대사는 보통 굵기로 구별하며 속말은 괄호로 표시한다. 일반 감상 화면에는 초상·흰색 PNG 말풍선·건너뛰기만 표시한다. 말풍선은 모서리·꼬리 크기를 유지하는 10조각 공통 렌더러다.

본편 전투와 시작 메뉴의 **일기토**는 같은 JSON·렌더러를 사용한다. 감상 모드는 캠페인 명령을 실행하지 않고 기존 진행·저장·난수를 유지한다. 목록에서 승부를 고르면 바로 재생하고, 끝나거나 건너뛰면 목록으로 돌아온다. `Esc`로 목록에서 시작 메뉴로 돌아간다. 독립 `cmd/srpg-duel`은 본편으로 통합했다.

## 이야기 데이터

`assets/duels.json`의 키는 스테이지 일기토 ID다. `Outcome`은 맵과 일치하고 `Beats`는 `Kind`·저작용 `Title`·`Speech`·`Side`·`Duration`으로 구성한다. `Supports`는 합류할 두 장수의 콘텐츠 ID다. 0=주역 아군, 1=적군, 2·3=지원군이다. 지원군은 합류 전 보이지 않으며 이후 대사와 공방에 참여한다. `block`·`dodge`·`yielding-block`·`critical-dodge`의 대사는 방어자에게, `critical-block`의 대사는 공격자에게 표시한다.

| ID | 전개 |
|---|---|
| B01-duel-1 | 등무가 당당히 중앙에 나왔다가 장비에게 한 합에 패한다. |
| B01-duel-2 | 정원지가 돌진하고 관우가 한 칼에 벤다. |
| sishui-guan-yu | 화웅의 연타와 강타를 관우가 쉽게 막고, 직선으로 도망치는 화웅을 추격해 벤다. |
| hulao-zhang-fei | 격렬한 공방에도 장비가 밀리지 않아 여포가 의아해한다. 관우·유비가 합류한다. |
| hulao-guan-yu | 첫 합부터 강공. 여포가 막고 놀란 뒤 반격하나, 다시 들어오는 강공은 피해낸다. 장비·유비가 합류한다. |
| hulao-liu-bei | 유비가 방어할 때 밀리며 회피로 버틴다. 여포가 질기다고 생각한다. 관우·장비가 합류한다. |

여포의 세 승부는 합류한 두 형제도 공격하고, 여포가 셋을 감당하기 어렵다고 판단해 직선 퇴각한다. 맵 결과는 아군 유지·적 퇴각이다. 승리 판정이지만 사망·탈진 자세로 끝내지 않는다.

## 동작과 본편 연결

기본 `duel.New`의 일곱 결말과 공통 공방은 참고 예제이며 필수 전개가 아니다. 필요한 동작이 없으면 타임라인과 렌더러를 확장한다. 양측 탈진인 `rest`를 여유로운 장수에게 쓰지 않는다.

- `critical-block`: 큰 섬광이 있는 강공을 막는다. 방어자는 피격·밀림·흔들림을 보이지 않는다.
- `critical-dodge`: 강공의 섬광은 있지만 회피하므로 접촉 불꽃·피격이 없다.
- `yielding-block`: 공격을 막아도 지정된 방어자가 뒤로 밀려 다음 공방에 그 위치를 유지한다.
- `reinforcements`: 두 지원군이 화면 좌하단과 우하단에서 각각 진입하고 기존 상대와 함께 적을 세 방향으로 둘러싼다.
- `withdraw`: 지정한 장수만 상대를 보며 뒤로 물러난다.
- `straight-chase`: 직선 도주와 추격 후 `backhit`으로 베기를 연결한다.
- `retreat` → `withdrawn`: 다른 장수는 자리를 지키고 퇴각자만 본진 대열 입구까지 직선 이동한다. 병사 그림과 겹치기 전에 멈춘다.
- `crossing-charge`: 참고 예제의 접촉·교차·방향 전환이다.

본편은 엔진이 확정한 `DuelID`·`Outcome`을 리플레이 장벽에서 재생한다. 종료·건너뛰기 후 전투맵에서 사망·퇴각 등의 후속 이벤트를 이어간다. 실제 참가자의 병종·진영색·외형을 사용한다. 지원군은 이야기 데이터가 지정한 인물이다. 연출은 규칙 상태와 난수를 변경하지 않으며 화면 없는 시뮬레이터도 같은 맵 결과를 따른다. 소리와 새 전용 스프라이트는 아직 없다.

## 실행과 검증

```sh
go run ./cmd/srpg-gui
go run ./cmd/srpg-gui -duel hulao-guan-yu
go run ./cmd/srpg-gui -duel hulao-liu-bei -duel-debug
go run ./cmd/srpg-gui -duel hulao-zhang-fei -duel-capture reports/duel-gallery
go test ./internal/duel ./internal/core ./internal/gui ./internal/content
go vet ./internal/duel ./internal/duelview ./internal/gui ./cmd/srpg-gui
SRPG_RENDER_TEST=1 go test ./internal/gui -run '^TestRenderedCampaignDuels$' -count=1 -v
SRPG_RENDER_TEST=1 go test ./internal/gui -run '^TestRenderedDuelGalleryMenu$' -count=1 -v
```

명시적 `-duel-debug`는 저작용 상황 설명과 `Space` 정지, `R` 처음, `←/→` 장면 탐색을 표시한다. 일반 감상에서는 `Enter`·`Space`·`Esc`가 건너뛰기다. 본편 일기토의 `F`는 기존 배속 조작이다. `-duel-capture`는 각 장면을 PNG로 저장한 뒤 종료한다.

프로젝트 제작 스킬은 `.agents/skills/duel-authoring/SKILL.md`다. 검증은 첫 타격·강공 방어·회피·지원군 진입·퇴각 경로·후속 결과와 감상 모드의 캠페인 상태 불변을 확인한다. 실제 본편 발동·종료·건너뛰기와 2배 해상도 렌더링도 검사한다.
