# Suno 새 BGM 11곡 (2026-11-26)

목적: 너무 자주 나오는 곡(empty-mountain 71, quiet-water 67, suno-story 66, golden-bamboo 60, chinese-traditional 전투 19)을 나누고,
도주·추격·습격·전사/상실/굴욕 장면용 곡을 새로 만든다. 장르(중국 전통 악기 RPG BGM, 인스트루멘털, 루프)는 기존과 같게, 느낌은 완전히 다르게.

Suno 무료 v6-mini, Advanced 모드, Lyrics 비움(인스트루멘털). 한 번 생성에 2곡. 하루 약 5번 생성 후 "Out of songs"(11-26 확인).

## 만든 것 (11-26, 4번 = 8곡)
| 제목 | 대체할 곳 | 스타일 프롬프트 |
|---|---|---|
| 11 탐험 추수 마을 (2:57, 3:00) | golden-bamboo | 얀친 아르페지오 + 샤오 플루트, 108 BPM 핸드 드럼·나무 딱딱이, 해맑은 민속 춤곡 |
| 12 관청 회의 (2:50, 3:03) | empty-mountain | 72 BPM, 사원 종·낮은 현 드론, 생(笙)·구친, 절제된 긴장 |
| 13 주막 저녁 (3:06, 3:08) | quiet-water | 84 BPM 느슨한 스윙, 구친·쉰(훈), 나무 블록, 아늑한 작은 방 |
| 14 밤 대화 (3:10, 3:19) | suno-story | 62 BPM, 얼후 솔로·비파 트레몰로, 현 패드, 모닥불 곁 대화 |

## 11-27 추가 생성: 16 기병 돌격, 17 혼전 난전, 18 도주, 19 추격, 20 습격 (각 2곡, 아래 프롬프트 그대로). 21 전사 상실 굴욕(3:44, 3:46), 22 공성 중전투(92 BPM, 대형 징·저음 북·낮은 합창 패드·디즈+저음 얼후, 쒀나·비파·행군 북 제외; 생성 중). 계획한 곡은 모두 만들었다

## 남은 것 (아래 표 중 21번만 남음)
제목 접두 번호를 이어 쓴다. 공통 꼬리말: `instrumental, seamless loopable arrangement`.

| 제목 | 용도 | 스타일 프롬프트 초안 |
|---|---|---|
| 16 기병 돌격 | 전투, 기본 전투 분산 | Chinese traditional RPG cavalry charge battle music, instrumental; galloping 160 BPM taiko and hooves-like hand drums, staccato erhu runs, urgent suona calls, bold and exhilarating, loopable |
| 17 혼전 난전 | 전투, 기본 전투 분산 | Chinese traditional RPG chaotic melee battle music, instrumental; frenetic pipa tremolo and rapid guzheng runs over irregular taiko hits and clashing gongs, 135 BPM, gritty and relentless, loopable |
| 18 도주 | 도망 장면(c18-flight 등) | Chinese traditional RPG escape chase music, instrumental; breathless 150 BPM, fast erhu ostinato and a pounding frame-drum heartbeat, strings rising without resolution, panicked and desperate, loopable |
| 19 추격 | 쫓기는 장면 | Chinese traditional RPG pursuit music, instrumental; relentless 128 BPM galloping low taiko, ominous low brass pulses and pizzicato strings tightening, a hunter closing in, dark and sustained tension, loopable |
| 20 습격 | 복병·기습 직후 | Chinese traditional RPG ambush music, instrumental; sudden explosive taiko hits and sharp pizzicato stabs separated by silent gaps, 170 BPM bursts, suona screams, shock and chaos, loopable |
| 21 전사 상실 굴욕 | 전사·상실·굴욕 직후(c07-scene-2, c25-start 등) | Chinese traditional RPG grief and humiliation music, instrumental; very slow 48 BPM, low string drone and single guqin notes with long silences, a broken solo erhu and a muffled funeral drum, bleak, heavy, bitter, loopable |

## 받기와 연결
- 무료 계정은 다운로드 7회(평생)를 이미 써서 새 곡은 Suno 안에서 듣기만 가능하다. 받으려면 Pro(월 ₩12,000대, 연 결제 시 ₩8,400/월 표시, 20곡 다운로드/월, 상업 권리) 필요. 구독은 사용자가 정한다.
- 받으면 `ver5/tools/music/sources/suno-<이름>.m4a`에 두고 `assets/music/music.json`·장면 `Music`에 연결, CREDITS.md에 한 줄 추가, `python3 tools/music/encode.py --force`.

## 폐기
- 15 야전 행군(2곡): 사용자 "기존 전투곡과 완전 똑같다"(11-26)로 Suno 휴지통에 보냄. 전투곡 3곡 목표는 16 기병 돌격·17 혼전에 하나를 더해야 한다. 기존 chinese-traditional과 겹치지 않게 템포·편성·분위기를 크게 틀 것(쒀나·비파·행군 북 조합 피하기).
