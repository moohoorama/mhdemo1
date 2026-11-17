# 음악 목록과 출처

게임이 트는 곡은 `themes/`의 Ogg Vorbis 96 kbps다. 원본(마스터)은 `tools/music/sources/`에 두고 git에 보관한다. 원본은 게임에 들어가지 않으며, 지금 쓰지 않는 곡도 지우지 않는다.

- 어디서 트는지: `music.json`(기본 곡, 장면 맵별 `Maps`, 전투별 `Stages`)과 장면 파일(`assets/scenes/chNN.json`)의 `"Music"`.
- 인코딩: `python3 tools/music/encode.py --force`. `music.json`·장면이 부르는 곡만 −16 LUFS(true peak −1.5 dB)로 맞춰 ogg를 만들고, 부르지 않는 ogg는 `themes/`에서 지운다. 원본은 남는다.
- 새 곡 넣기: 원본을 `tools/music/sources/<이름>.<확장자>`로 두고, `music.json`이나 장면에서 `<이름>`을 부른 뒤 인코딩한다. 이 표에 한 줄을 더한다.

## 쓰는 곡

| 이름 | 어떤 음악 | 곡명 · 만든 이 | 출처 · 라이선스 | 쓰는 곳 |
|---|---|---|---|---|
| imperial-china-cinematic | 웅장한 관현악 주제곡 | Imperial China Cinematic · Shane Ivers | silvermansound.com · CC BY 4.0 (표기 필요, 아래) | 타이틀 |
| china-trailer | 짧고 힘찬 예고편풍 | China Trailer · VitalSource | Pixabay · Pixabay Content License | 도원결의 (c01-e-oath) |
| the-march | 행진하는 북과 관악 | 出征 (The March) · shanghaistoneman | Pixabay (AI 제작) · Pixabay Content License | 출진 준비 |
| mountains-rivers | 넓고 느린 산수풍 | Mountains And Rivers Have Dreams · kaazoom | Pixabay (AI 제작) · Pixabay Content License | 세계 지도 |
| golden-bamboo | 가볍고 평화로운 피리·현 | Breezes Through The Golden Bamboo · kaazoom | Pixabay (AI 제작) · Pixabay Content License | 탐험, 평화로운 고을 |
| quiet-water | 잔잔한 실내 | By The Quiet Water · kaazoom | Pixabay (AI 제작) · Pixabay Content License | 주막, 상점 |
| mystical-china | 어둡고 신비로운 | Mystical China · MusicInMedia | Pixabay · Pixabay Content License | 음울한 고을(길가 마을, 북해, 소패), 간옹의 급보 (c18-news) |
| empty-mountain | 무게 있는 관청 | Echoes of the Empty Mountain · OpenMindAudio | Pixabay · Pixabay Content License | 저택, 회의, 관청, 업 관청 |
| azure-cloud | 쓸쓸한 대화 | Ascent To The Azure Cloud · kaazoom | Pixabay (AI 제작) · Pixabay Content License | 관청의 슬픈 이야기 (c07-council, c13-xiaopei) |
| forbidden-city | 도성 거리 | Tử Cấm Thành (Track 02) · HungTran90 | Pixabay · Pixabay Content License | 낙양 거리 |
| echoes-empire | 궁궐의 위엄 | Echoes of the Empire · djovan | Pixabay (AI 제작) · Pixabay Content License | 궁궐, 허도 궁문 앞 |
| empress-joy | 밝은 궁중풍 | The Empress's Joy · djovan | Pixabay (AI 제작) · Pixabay Content License | 영천 전투 (B02) |
| chinese-dynasty | 무거운 전투 | Chinese Dynasty · Roman_Sol | Pixabay · Pixabay Content License | 어려운 전투 (B05, B07, B10, B12, B15) |
| chinese-traditional | 기본 전투 | Chinese Traditional · moodmode | Pixabay · Pixabay Content License | 그 밖의 전투 |
| war-drums-duel | 북 위주의 격렬한 일기토 | 일기토 (war drums) · 직접 생성 | ACE-Step v1 (2026-11-10) · Apache 2.0 모델, 생성물 자체 소유 | 일기토 |
| painted-pavilion | 밝고 흐뭇한 | The Painted Pavilion · kaazoom | Pixabay (AI 제작) · Pixabay Content License | 승리, 일이 잘 풀릴 때 (c08-p1) |
| emerald-peaks | 고쟁·얼후의 평온한 아침 | Dawn Over The Emerald Peaks · kaazoom | Pixabay (AI 제작) · Pixabay Content License | 유비의 거처, 복숭아 숲 (`S_liu_house`, `S_peach_grove`) |
| sad-melody | 슬픈 동양풍 선율 | Sad, melancolic, oriental melody · fidelfortune | Pixabay · Pixabay Content License | 슬픈 장면 (c03-start, c08-start) |

## Suno로 만든 곡

2026-11-15 Suno 무료 플랜(v6-mini)으로 생성. 무료 플랜 생성물은 **개인·비상업 용도만** 된다. 게임을 공개·판매하려면 유료 플랜에서 다시 만들어야 한다. 원본은 M4A(무료 플랜은 WAV 불가).

| 이름 | 어떤 음악 | Suno 제목 · 길이 | 쓰는 곳 |
|---|---|---|---|
| suno-story | emerald-peaks 분위기를 옮긴 고쟁·비파·얼후의 평온한 곡 | [채택] 스토리 대화곡 · 3:20 | 스토리 대화 기본 (`Story`) |
| suno-journey | 디즈·고쟁의 떠돌이 여정 | 02 여정 길 · 0:21 | 길 (`S_road`, `S_field_road`) |
| suno-farewell | 얼후 독주의 이별·죽음 | 03a 이별 죽음 · 3:18 | 도겸의 죽음 (c14-death), 우이 퇴각 (c14-retreat), 옥대를 넘긴 뒤 (c17-death) |
| suno-defeat | 낮은 현·장송 북의 패배 | 03b 패배 · 3:10 | 패배 화면 (`Defeat`) |
| suno-intrigue | 피치카토·고금의 조용한 음모 | 04 긴장 음모 · 2:49 | 첩자 (c15-spy), 백문루 (c16-baimen), 동승의 밀모 (c17-dong1, c17-dong2), 의대조 (c17-edict) |
| suno-boss-battle | 150 BPM 북·쇄납의 강적 전투 | 06a 대장 전투 · 3:03 | 호로관 (B06), 하비 (B16) |
| suno-last-stand | 다급한 현 반복의 불리한 방어전 | 06c 불리한 방어전 · 3:08 | 우이 (B14), 서주성 (B18), 연주 (B19) |

## 원본만 보관하는 곡 (지금 쓰지 않음)

| 이름 | 어떤 음악 | 출처 |
|---|---|---|
| 01-village … 06-victory | 마을·관청·음모·전투·다급함·환희 여섯 테마 (GM 음색 MIDI) | `tools/music/compose.py`로 직접 작곡·렌더, 음색 TimGM6mb (GPL-2.0), `tools/music/README.md` |
| sipmyeon-maebok | 십면매복 MIDI 렌더 | `tools/music/sipmyeon.py`로 직접 렌더 |
| gosan-yusu, maehwa-samnong, yangchun-baekseol | 고산유수·매화삼농·양춘백설 렌더 | musics.zip 렌더, 출처 기록 없음 |

## 표기 의무

CC BY 4.0 곡은 게임을 보이거나 배포하는 곳에 다음을 적어야 한다. 아직 크레딧 화면이 없으니 생기면 넣는다.
"Music: Imperial China Cinematic by Shane Ivers - https://www.silvermansound.com"
