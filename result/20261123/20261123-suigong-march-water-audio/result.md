# 수공 행군·속도·물소리 수정 (2026-11-23)

커밋·푸시·stash·rm -rf 없음. 작업 트리에만 수정. 다른 수공 작업은 진행 중이 아니었다(최근 result 확인).

## 산출물
- `suigong.mp4` 1280x850, 60fps, 15.75초(945프레임), H.264+AAC (효과음 `suigong.sfx`를 `tools/sfx/dub.py --no-music`으로 얹음)
- `march_sheet.png`(프레임 2/60/120/180), `dam_sheet.png`(684~759), `ford_sheet.png`(770~870, 아래 두 장은 잘림), `frames/`(원본 PNG 23장), `waveform_analysis.txt`

## 1. 행군: 장면 엔진 걷기 방식
원인: 게임이 매 프레임 `sync`로 유닛을 규칙 상태 칸에 되돌리며 걷기를 지우고, 데모가 매 프레임 `play("walk")`를 다시 불러 걷기 모션이 첫 프레임에 머물렀다(그래서 미끄러짐).
- 조사한 경로: `scene.go updateScene`의 칸 단위 걷기(`walkTo`, `stepTime`=.28초/칸, `u.over` 이월), `sceneFootfall`(칸마다 `step`, 말 탄 쪽은 `hoof`, `sfxVolAtCell` 거리 감쇠).
- 구현: `Game.unitsHeld`(sync 중지, `view.go`·`game.go`), `marchCell`(uidemo_suigong.go)이 칸마다 `walkTo`+발소리, 속도 1칸/.28초(3.57칸/초). 행 별 시작 위상으로 발이 한꺼번에 구르지 않게 함. 걷기 모션은 배속 없이 1배. 보병 `step`, 기병 `hoof`. 이전 `beforeDraw` 훅은 불필요해져 제거. 지도 폭 40->48, 오프닝 11칸 행군. 방향 NW, 좌측·하단으로 이어지는 구도 유지(march_sheet).
- 검증: `TestSuigongMarch`가 180프레임 동안 (a) 병사 이동거리 = 3초*3.57칸/초 +-1칸, (b) 걷기 모션·NW 유지, (c) 매 프레임 새로 시작한 칸 수(소리 범위 안)와 발소리 요청 수가 정확히 일치함을 확인. 영상 오디오에 step/hoof 포함: 효과음 기록 step 1444, hoof 179회, 3초간 RMS -36~-41dB로 들어감(원래 발소리가 .05로 작게 설계됨).
- 한계: 발이 땅에 닿는 순간과 소리의 일치는 모션 프레임 단위로 직접 맞춘 것이 아니라 칸 시작에 소리를 내는 엔진 방식 그대로다. 병사가 많아 게임의 같은 소리 3틱 간격 제한으로 소리가 솎인다.

## 2. 둑 붕괴 후 2배
둑 붕괴(tBurst) 이후만 `surgeRate=2`로 재생(`SurgeT = 1.0+(t-tBurst)*2`). 물·잔해·물보라·깊이(병사 피격)·스프레이가 같은 시계라 함께 빨라진다. 다시 굽지 않았고 잔해 모양·움직임 그대로. 붕괴 전 관우 대사·카메라·흔들림·`boulder`와 버스트 `water-big`은 그대로.
| 항목 | 이전 | 새 |
|---|---|---|
| 둑 장면(버스트->컷) | 2.5초 | 1.25초 (컷 12.65초, 프레임 759) |
| 여울 장면 | 4.4초 | 2.2초 |
| 물 끝 도달=흰색 시작 | 컷+3.4 | 컷+1.6 |
| 흰색 | 1.0초 | .6초, 컷+2.2(14.85초, 프레임 891)에 완전 `#ffffff` |
| 병사 외침 | .4/.8초 | .2/.4초 |
| 플래시 / 여진 | .5 / 1.2초 | .3 / .6초 |
확인: 프레임 891, 920 전 픽셀 흰색(100%). 870은 흰 1.9%, 영상 880번 프레임 15.6%로 서서히 올라와 정지한 물 프레임 없음.

## 3. 물소리
분석(`waveform_analysis.txt`, 원본: assets/sounds): water 1.46초·water-big 1.65초는 0.4초 올라 -17dB 고원 0.3~0.4초 후 1.2초 내림하는 단일 파도, 좌우 상관 .97(점음원), 무게중심 1.1kHz. boat 2.5초는 .3~2.0초가 -37dB로 3dB 이내로 고른 이음새 없는 물결, 좌우 상관 .27, 무게중심 650Hz. 이전 방식은 water를 1.3초 간격으로 반복 재생해 1초짜리 부풀었다 꺼지는 맥박이 이어졌다(어색함의 원인).
재조합(`tools/sfx/flood.py`, 새 cue 3개 `assets/sounds/flood-*.ogg`):
- `flood-surge`(4.4초): boat 고른 구간을 .45초 낟알·.25초 등전력 교차, 시작·+-9% 속도·역재생 무작위, 좌우 따로(상관 -.03)로 낮은 물결(50~1000Hz) + water/water-big 고원 낟알 중고음(350Hz 이상). 낮은 쪽이 먼저 차오르고 중고음이 늦게 따라 가까워지는 느낌, 정점(+3.2초) 후 1초 내 소멸.
- `flood-crash`(1.5초): 앞머리가 병사를 칠 때, 30ms 안에 오르는 중음 벽 + 낮은 쿵.
- `flood-spray`(2.4초): 1.5kHz 이상 고음 낟알, 물보라.
- 배치: 버스트 `water-big`(유지), +.25초 surge, 컷+.5초 crash, 버스트+1.7초 spray(스프레이 시작과 같음). 정점이 흰색 완성(14.85초)과 일치.
- 반복 이음새 확인: 레벨 자기상관 최대 surge .02, spray .02(원본 .35, 맥박). 20ms 단위 최대 도약 surge 3.4~6.7dB로 원본 boat(6.9)과 같은 수준.
- 영상 믹스 250ms RMS: 버스트 -22dB, 둑 구간 -23~-30, crash -23, 정점 -21dB. 클리핑 없음(피크 .41). 정점이 버스트보다 크게 높지는 않고 비슷하다.
- 한계: 귀로 직접 듣지 못했다. 수치(파형·RMS·자기상관)로만 확인. 컷 직전 -30dB 정도로 살짝 낮아지는 구간(12.5~13.0초)이 있다.

## 검증
- `go vet ./internal/gui/` 통과. `go test -p 1 -count=1 -run 'TestSuigong|Surge' ./internal/gui/` 통과. 전체 테스트 안 돌림.
- 캡처: `caffeinate -u` 후 `lockf ... go run ./cmd/_uidemo-vm -ui suigong -stage B25 -frames 945` 성공(46초), 더빙 성공.

## 바꾼 파일
`ver5/internal/gui/uidemo_suigong.go`, `view.go`, `game.go`(unitsHeld), `uidemo.go`(beforeDraw 제거), `field_surge_test.go`, `tools/sfx/flood.py`(새), `assets/sounds/flood-{surge,crash,spray}.ogg`(새), `assets/sounds/README.md`. 앞선 미커밋 변경은 보존.
