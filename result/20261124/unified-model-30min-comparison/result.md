# 단일 모델 30분 비교: A 통합 수치 모델 / B 라야 (B 미실행)

## 완료 상태

- **A (기존 Go 수치 모델 확장, 통합 단일 모델): 완료.** 30분 학습·개발, 미사용 평가 시드 8개 평가까지 수행했다.
- **B (라야 단일 체크포인트): 미실행.** 라야 가중치는 내려받았고(scratch, 807 MB) 브리지 코드를 작성했으나, 실행 환경을 만드는 중 디스크가 100%(여유 약 284 MiB)가 되어 설치를 중단했다. 측정값은 없다. 아래 "미완료" 참고.

## A 결과 (매우 어려움, 재도전 없음, 평가 시드 240001-240008)

| 항목 | 값 |
|---|---|
| 학습 예산 | 30분 벽시계 (설정 이후부터, warm·교사 모방·강화학습·개발 포함). 실제 1788.5 s |
| 학습 캠페인 수 | 1120 (warm 8 + 교사 모방 8 + 강화학습) |
| 개발 최고 | 평균 4.25 승 (개발 4판, 초기 3.00), 개발 시드 230001-230004, 체크포인트 episode 1088 |
| 평가 (준비 학습) | 8 시드 평균 2.25, 중앙값 2.5, 최소 0, 최대 5, 완주 0. 최고 기록: 5승 후 B06 첫 패배 (seed 240005) |
| 평가 (준비 규칙) | 8 시드 평균 2.50, 중앙값 3.0, 최소 0, 최대 5, 완주 0. 최고 기록: 5승 후 B06 첫 패배 (seed 240005) |
| 참고 (기존 규칙 AI, 같은 시드) | 평균 5.62, 중앙값 5, 최고 8승 (B09 첫 패배, seed 240001) |

시드별 결과(첫 패배 전까지 승리 수 / 첫 패배 전투):
- 준비 학습: 0/B01, 2/B03, 4/B05, 0/B01, 5/B06, 4/B05, 3/B04, 0/B01
- 준비 규칙: 0/B01, 2/B03, 4/B05, 0/B01, 5/B06, 4/B05, 5/B06, 0/B01

전투별 턴, 장수별 획득 경험치(엔진 원 단위), 부목표 달성 수·보상, 돈·소모품 변화는 `approach-a/eval/eval-*.json`의 `runs[].battles[]`에 기록되어 있다. 경험치·보상은 엔진의 전투 결산(Merit)에서 읽었고, 돈·인벤토리 변화는 준비 전후 구간으로 나눠 기록했다(전투 후 구간은 다음 전투 시작 직전까지, 스토리·탐험·상점 포함).

## 해석 범위

- 8 시드, 30분 예산, 한 번 학습한 체크포인트에 대한 비교다. 최고 기록(5승)만으로 안정적 실력이라고 볼 수 없다.
- 학습 모델은 규칙 AI보다 약하다(평균 2.3~2.5 대 5.6). 이 예산에서 통합 수치 모델은 기존 규칙 AI를 넘지 못했다.
- 개발 평가는 시드 4개라 체크포인트 선택도 잡음이 많다.
- 준비 규칙 평가는 학습한 준비망을 빼고 규칙 준비를 쓴 것이다. B와 같은 규칙 집합으로 맞춘 비교용이며, 통합 모델 본래 구성은 첫째 행이다.

## 규칙 기반으로 남은 기능 (A에서 동일 적용, B에도 같은 규칙 집합을 적용할 계획이었음)

- 탐험 이동·상점 구매·보급: 기존 `Searcher` 유전자(genome) 규칙.
- 이야기 선택: 기존 `ChooseStory` 규칙.
- 기본 편성 준비(`prepareBase`)와 학습 창 처리: 기존 규칙 (준비 규칙 평가에서는 준비망도 규칙으로 바꿈).
- A 학습 모델이 배운 것: 전투 중 각 명령 선택(수치 신경망, 32 은닉 2층), 준비 학습 평가 행만 준비망 사용.
- 교사 모방 단계는 규칙 탐색 AI의 전투 선택을 모방한 것이다(전투별 전문가 없음, 단일 가중치).

## 입력과 정책 조건

- 전투 입력은 기존 `BattleInputs`(합법 명령 후보와 공개 상태 특징)를 그대로 사용했다. 숨은 복병·미래 사건·난수 노출은 기존 코드 범위를 확인하지 않았다(신규 감사 없음, "확인 안 됨").
- 전용 전문가 가중치 없음: 학습 모델 `StageModels` 비어 있음(평가 명령이 확인).

## B 미완료 사유와 범위

- 라야(`convaiinnovations/laya`)는 ModernBERT-large 기반 결정 헤드 모델(약 4억 2천만 파라미터)이다. 로컬에 pip 실행 환경이 없어 `/opt/homebrew/bin/python3.12`로 venv를 만들었고, `torch` 설치 중 디스크가 가득 차 `torch._strobelight` 모듈 누락으로 실패했다. 전역 pyenv Python은 SSL이 없어 pip이 동작하지 않았다.
- 완료된 것: 라야 파일 내려받기(모델 가중치, config, 토크나이저, rl_agent_api.py, rl_common.py), Go 쪽 상태 텍스트·선택지 텍스트 작성, `_unifiedlearn export`(교사 결정을 텍스트 행으로 내보내기), 파이썬 브리지(`tools/unified30/laya_bridge.py`)와 헤드 미세조정(`laya_train.py`). 두 파이썬 스크립트는 실행하지 않았다.
- 미실행: 라야 추론 지연 측정, 헤드 학습 30분, 개발 평가, 캠페인 평가. 따라서 B의 최고 기록·평균·중앙값은 없다.
- 예상 비용(설계상): 라야는 CPU에서 결정당 여러 번 호출(8개 묶음 토너먼트)이 필요해 한 캠페인이 수십 분 이상 걸릴 것으로 보인다. 측정 전이므로 추정일 뿐이다.

## 재현 명령

```
# 스냅샷: 작업 트리 상태를 scratchpad/frozen/ver5 에 복사(기준 HEAD ae5f7fb는 탐험 대본 검증에 실패해 쓰지 않음)
cd <scratch>/frozen/ver5
<scratch>/frozen/unifiedlearn train -content assets/data -budget 30m -warm 8 -warm-epochs 20 -bc 8 -bc-epochs 2 -collect 8 -par 4 -dev-games 4 -seed 220001 -dev-seed 230001 -bc-seed 225001 -battle-temperature 0.25 -hidden 32 -init-seed 7 -model <result>/approach-a/train/model -out <result>/approach-a/train
<scratch>/frozen/unifiedlearn-eval eval -content assets/data -model <result>/approach-a/train/model.best.json -prep learned -eval 8 -eval-seed 240001 -par 4 -out <result>/approach-a/eval/eval-learned.json
<scratch>/frozen/unifiedlearn-eval eval -content assets/data -model <result>/approach-a/train/model.best.json -prep rule -eval 8 -eval-seed 240001 -par 4 -out <result>/approach-a/eval/eval-rule-prep.json
<scratch>/frozen/autoplay run -content assets/data -player rule -seeds 240001,...,240008 -tries 1 -par 4
```
`approach-a/train/run-train.sh`, `approach-a/eval/run-eval.sh` 참고.

## 해시

- 콘텐츠 해시(`assets/data`+`assets/scenario`, 스냅샷): `8e1f8c056dd262f9cff2d480057f951e2dfd00801bc8aaf830c25f994082c25f`
- 기존 `assets/autoplay/battle.json` 콘텐츠 해시 `e3dff54b...`는 현재 데이터와 달라 사용하지 않았다(해시 변경·우회 없음).
- 체크포인트 `approach-a/train/model.best.json` sha256 `5ee15eef63ae69e6a2244ab69604824e1a291cc97a969380c85a422ccbae9685`
- `model.latest.json` sha256 `5887b72f5a89a152e74dc8e1199d1ea8fb1f6b270f877b0185947e7896a6e256` (결과 폴더에 있으나 커밋하지 않음)
- 바이너리: `unifiedlearn` `14a2bb88...`, `unifiedlearn-eval` `7ecb85cf...`, `autoplay` `df9d56d1...` (scratch/frozen, 커밋하지 않음)

## 시드

- 학습: 220001-220008(warm), 220009 이후(에피소드). 교사 모방: 225001-225008. 개발: 230001-230004. 평가(미사용): 240001-240008.
- 개발·평가·학습 시드는 서로 겹치지 않는다.

## 시간 사용

- A 학습: 1788.5 s(약 29.8분) 벽시계, 1120 캠페인, 병렬 4.
- A 평가: 준비 학습 5 s, 준비 규칙 20 s, 기준 규칙 AI 약 10 s 수준(벽시계 27 s 합계, `approach-a/eval/eval.log`).
- 설치·스냅샷·빌드·다운로드 시간은 측정 예산에 넣지 않음(별도 기록 없음).
- B: 설치 실패 전까지 약 10분 소요, 학습·평가 시간 없음.

## 오류와 주의

1. 기준 HEAD(`ae5f7fb`)의 탐험 대본 검증이 실패("explore23 town 장터 상인 ... says nothing"). 작업 트리는 다른 세션의 미커밋 변경이 고쳐 놓아 정상이다. 그래서 작업 트리 상태를 스냅샷으로 얼렸다. 다른 세션 변경의 영향을 받지 않도록 같은 스냅샷을 A·기준 평가에 모두 썼다.
2. `_unifiedlearn` 학습 바이너리는 평가 플래그(`-prep`, `-policy`) 추가 전에 빌드됐다. 학습 코드 경로는 같다(바이너리 해시 기록). 평가 바이너리(`unifiedlearn-eval`)에서 평가했다.
3. 대기 모니터(pgrep 패턴이 자기 자신에 매칭)가 종료되지 않아 셸 프로세스를 직접 종료했다. 결과에 영향 없음.
4. 디스크 100%: 스크래치 `frozen/ver5`(29 MB), `snap`(636 MB, 사용 안 함), `venvb`(530 MB, 부분 설치 실패), `laya`(807 MB), 기존 `venv`(24 MB)가 있다. 삭제는 하지 않았다(rm -rf 필요, 사용자 승인 없음). 정리는 사용자 승인 후 필요.

## 미완료 항목

- B 전체 (설치·추론 지연 측정·헤드 학습 30분·개발 평가·캠페인 8 시드).
- B 정책 감사: 라야 입력이 공개 정보만 쓰는지 검증 안 함.
- A의 공개 정보 감사(숨은 복병 노출 여부)는 기존 특징 코드를 그대로 쓴 것 외 검증하지 않음.
- 체크포인트 `latest` 커밋 제외.
