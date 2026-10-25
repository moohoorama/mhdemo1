# 문서 색인

문서 이름은 `<분류>-<기능>.md`다. 분류는 다음과 같다.

| 분류 | 맡는 것 |
|---|---|
| logic | 게임이 어떻게 돌아가는가(규칙과 처리 순서) |
| data | 그 위에 어떤 데이터를 태우는가(데이터 형식과 내용) |
| presentation | 화면, 연출, 그림 |
| tools | 에디터와 빌드 |
| system | 구조, 저장 |

규칙은 세 가지다.
- 분류 문서는 **현재 코드와 데이터가 하는 일만** 현재형으로 쓴다. 값이 다르면 문서 3~4줄에 적힌 기준(코드·데이터)을 따른다.
- 한 사실은 한 문서에만 쓰고 나머지는 링크한다.
- 앞으로 할 일은 `todo/`, 작업 중 plan은 `temp/`(git ignore), 결정과 이유는 [history.md](history.md)에 쓴다.

## 방향과 이야기

| 문서 | 내용 |
|---|---|
| [design.md](design.md) | 게임 방향, 성장·재미 목표, 설계 원칙 |
| [fullstory.md](fullstory.md) | 전체 이야기: 기획, 인물, 공통부 1~24, 연의·가상모드 |
| [story-writing.md](story-writing.md) | 이야기·대사 쓰는 법: 기조, 서사·대사·연출 원칙, 작업 순서 |
| [history.md](history.md) | 설계 결정과 이유 |

## logic

| 문서 | 내용 |
|---|---|
| [logic-combat.md](logic-combat.md) | 전투 흐름, 능력치, 피해·명중, 병법·MP, 버프, 지형 |
| [logic-growth.md](logic-growth.md) | 경험치, 레벨, 승급, 부관 |
| [stage_lv_balance.md](stage_lv_balance.md) | 스테이지 레벨 정책, 적 수, 재는 법 |
| [logic-traits.md](logic-traits.md) | 특성과 책략 |
| [logic-duel.md](logic-duel.md) | 일기토 규칙 |
| [logic-battle-event.md](logic-battle-event.md) | 전투 이벤트와 승패 판정 |
| [logic-campaign.md](logic-campaign.md) | 진행 구조, 출전·배치, 보상 |

## data

| 문서 | 내용 |
|---|---|
| [data-scenario-node.md](data-scenario-node.md) | 시나리오 노드·스테이지·이벤트 JSON 필드 |
| [data-map.md](data-map.md) | 맵 문서 형식과 저장 |
| [data-character.md](data-character.md) | 장수 열전 |
| [data-item.md](data-item.md) | 장비와 아이템 |
| [data-speech-style.md](data-speech-style.md) | 장수별 말투 |

## presentation

| 문서 | 내용 |
|---|---|
| [presentation-ui.md](presentation-ui.md) | 전장 HUD, 조작, 배치 화면, 화면 확인 방법 |
| [presentation-scene.md](presentation-scene.md) | 장면 연출, 초상, assets/scenes |
| [presentation-explore.md](presentation-explore.md) | 탐험: 전투 사이에 걷기·대화·이벤트·상점·출진, exploreNN.json, 형식 2 둘러보기 |
| [presentation-duel.md](presentation-duel.md) | 일기토 연출 |
| [presentation-sprite.md](presentation-sprite.md) | `.spr` 형식과 태그 색 |
| [presentation-replay.md](presentation-replay.md) | 클레임 리플레이: 규칙은 턴제, 화면은 동시 재생 |
| [ui-mockup/](ui-mockup/README.md) | 전장 UI 시안(움직이는 HTML)과 확정 항목 |

## tools / system

| 문서 | 내용 |
|---|---|
| [tools-editor.md](tools-editor.md) | 맵 에디터, 도트 에디터 |
| [tools-scene-viewer.md](tools-scene-viewer.md) | 모든 장면 맵을 같은 기준으로 보는 브라우저 뷰어 |
| [tools-build.md](tools-build.md) | 에셋 빌드 명령, Android 빌드 |
| [system-architecture.md](system-architecture.md) | 패키지 구조, 데이터 번들, 결정성 |
| [system-save.md](system-save.md) | 저장, 자동 저장, 호환 |

## 해야 할 일

| 문서 | 내용 |
|---|---|
| [todo/engine.md](todo/engine.md) | 시나리오 1~20을 위한 엔진 작업(미구현 설계) |
| [todo/graphics.md](todo/graphics.md) | 일러스트·초상·스프라이트·배경·타일·효과 |
| [todo/data.md](todo/data.md) | 시나리오·장수·일기토·아이템·맵·밸런스 데이터 |
| [todo/treasure-ideas.md](todo/treasure-ideas.md) | 보물 아이디어 참고 자료 |

정리 전 옛 문서는 `../docs_old/`에 있다.
