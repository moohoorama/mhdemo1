# 이벤트 CG 프롬프트

출력은 1672×941 PNG(16:9)다. 시도한 다른 화풍(수묵·코에이·만화·카툰·유화 등)의 결과 이미지는 이 폴더에 남아 있지만 프롬프트는 버렸다.

## 확정 화풍: 아마추어 웹툰 · 진중한 색 · 성인 이목구비 (2026-10-18)

이벤트 CG는 이 화풍으로 그린다. 기준 이미지는 `heroes-chopstick-webtoon-amateur-somber-mature.png`이고, 새 장면은 이 이미지를 참조로 넘겨 화풍만 따른다(장면·인물은 베끼지 않는다). 수묵 담채 초상과는 별개다.

기준 이미지를 만든 프롬프트(참조: `heroes-chopstick-webtoon-amateur-somber.png`):

```text
Redraw the reference image keeping everything exactly the same (composition, poses, costumes, somber muted palette, simple amateur webtoon line style, flat coloring, level of detail) except for the facial proportions of both men. Do NOT add wrinkles, age lines or gray hair; the skin stays smooth and simple. Instead redraw the faces with mature adult proportions instead of a youthful idealized look: smaller and narrower eyes set lower on the face with less white showing, lower and heavier brows closer to the eyes, a longer nose and a longer space between nose and mouth, a wider flatter nose, thinner and wider mouth, a longer and squarer face with a stronger heavier jaw, smaller forehead-to-chin ratio shifted so the lower face is larger, slightly lower-set ears, thicker neck. Simplify the eyes to small simple shapes rather than large expressive anime eyes. Keep their expressions (Liu Bei worried with a sweat drop, Cao Cao smirking). Wide 16:9, no text, no watermark, no border.
```

이 이미지에 이르는 순서(앞 단계 결과를 다음 단계의 참조로 넘겼다):

1. `heroes-chopstick-webtoon-amateur.png` — 막 배우기 시작한 학생의 웹툰으로 단순화(떨리는 선, 평면 채색, 배경 거의 없음)
2. `...-amateur-somber.png` — 색만 진중하게(저채도 짙은 남색·먹색 하늘, 이끼색·검정 도포, 번개만 밝게)
3. `...-amateur-somber-mature.png` — 주름·흰 수염 없이 비율로 성숙하게(작고 좁은 눈, 낮고 굵은 눈썹, 긴 코와 인중, 길고 각진 턱). 주름을 넣은 `-older`는 버렸다.

새 장면에 쓰는 틀(예: `luoyang-bloodbath-webtoon-amateur-somber.png`):

```text
Use the reference image ONLY for the drawing style: a simple amateur webtoon line style with flat coloring, a somber muted dark palette, simple unfussy backgrounds, and mature adult facial proportions (small narrow eyes, heavy brows, long nose, long square jaw, no wrinkles). Do NOT copy its scene or characters. Draw a new scene: <장면과 구도를 구체적으로>. Faces small and simple. Wide 16:9, no text, no watermark, no border.
```

- 구도는 앞서 쓴 것과 겹치지 않게 새로 잡는다(낙양: 지붕 위 부감).
- 선을 지나치게 굵게 하지 않는다(굵게 한 `amateur2`는 거절). 장식·반사·등불·원경 건물은 넣지 않는다.

## 만든 장면

- `luoyang-bloodbath-webtoon-amateur-somber.png` (시나리오 2 낙양): 지붕 위 부감으로 본 궁궐 마당. 계단 아래 쓰러진 하진, 계단을 오르는 원소군, 위쪽 문으로 들어오는 서량 기병, 타오르는 지붕.
- `heroes-chopstick-webtoon-amateur-somber-mature.png` (시나리오 16 허도, 영웅론): 조조의 매화 정원 정자, 번개, 떨어진 젓가락을 줍는 유비와 술잔을 든 조조. 화풍 기준 이미지.
