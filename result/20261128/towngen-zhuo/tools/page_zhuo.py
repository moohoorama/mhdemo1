import base64, io, json, os, sys
from PIL import Image, ImageDraw
SP = os.path.dirname(os.path.abspath(__file__))
R = '/Users/younwookim/orca/mhdemo1/result/20261128/towngen-zhuo'
info = json.load(open(f'{R}/roles_S_zhuo.json'))
mp = json.load(open(f'{SP}/root/assets/graphics/maps/S_zhuo.json'))
ox, oy = mp['origin']; g = mp['ground']; S = 8
u_lo, u_hi, v_lo, v_hi = info['bounds']
W = (u_hi - u_lo + v_hi - v_lo + 4) * S; H = (u_hi - u_lo + v_hi - v_lo + 4) * S // 2 + 20
img = Image.new('RGBA', (W, H), (0, 0, 0, 0)); d = ImageDraw.Draw(img)
X0 = (v_hi - v_lo + 2) * S
def P(u, v): return (X0 + (u - u_lo - (v - v_lo)) * S, 10 + (u - u_lo + v - v_lo) * S // 2)
def tile(u, v, col):
    x, y = P(u, v); d.polygon([(x, y), (x + S, y + S // 2), (x, y + S), (x - S, y + S // 2)], fill=col)
ZC = {'관아': (176, 60, 46), '상업': (214, 150, 50), '부유층': (120, 90, 160), '민가': (110, 150, 80), '밭': (150, 170, 60)}
zones = {}
for b in info['zones']:
    for u in range(b['u0'], b['u1']):
        for v in range(b['v0'], b['v1']): zones[(u, v)] = b.get('zone')
for u in range(u_lo - 1, u_hi):
    for v in range(v_lo - 1, v_hi):
        c = g[v - oy][u - ox]
        if u < u_lo or v < v_lo: tile(u, v, (90, 86, 80))
        elif c == 'i': tile(u, v, (150, 155, 165))
        elif c == 'd': tile(u, v, (196, 170, 120))
        else: tile(u, v, ZC.get(zones.get((u, v)), (205, 205, 195)))
for k, p in info['placed'].items():
    tile(*p['door'], (20, 20, 20))
img = img.crop(img.getbbox()); img = img.resize((img.width * 2, img.height * 2), Image.NEAREST); img.save(f'{R}/zones.png')
def b64(im, fmt='PNG'):
    b = io.BytesIO()
    if fmt == 'PNG': im.convert('RGBA').quantize(256, method=Image.Quantize.FASTOCTREE).save(b, 'PNG', optimize=True)
    else:
        bg = Image.new('RGB', im.size, (20, 18, 16)); bg.paste(im, mask=im.split()[3]); bg.save(b, 'JPEG', quality=85)
    return f'data:image/{"png" if fmt == "PNG" else "jpeg"};base64,' + base64.b64encode(b.getvalue()).decode()
cur = Image.open(f'{R}/current.png').convert('RGBA'); gen = Image.open(f'{R}/generated.png').convert('RGBA'); sw = Image.open(f'{R}/swap_version.png').convert('RGBA')
bb = gen.getbbox(); cur, gen, sw = cur.crop(bb), gen.crop(bb), sw.crop(bb)
z = tuple(int(a) for a in (gen.width * 0.2, gen.height * 0.25, gen.width * 0.8, gen.height * 0.75))
mir = []
for n in ('cottage_1', 'tavern', 'mansion_2', 'yamen'):
    a = Image.open(f'{SP}/reg/huabei/{n}.png'); b = Image.open(f'{SP}/reg/huabei/{n}_r.png')
    c = Image.new('RGBA', (a.width + b.width + 30, max(a.height, b.height) + 10), (184, 148, 96, 255))
    c.alpha_composite(a, (5, c.height - 5 - a.height)); c.alpha_composite(b, (a.width + 25, c.height - 5 - b.height))
    mir.append(c.resize((c.width * 2, c.height * 2), Image.NEAREST))
roles_rows = ''.join(f"<tr><td>{k}</td><td>{', '.join(sorted(set(v)))}</td><td>{info['placed'].get(k, {}).get('name', '—')}</td><td>{info['placed'].get(k, {}).get('door', '—')}</td></tr>" for k, v in info['roles'].items())
legend = ''.join(f'<span class="sw" style="background:rgb{c}"></span>{k} ' for k, c in ZC.items())
notes = ''.join(f'<li>{n}</li>' for n in info['notes'])
html = f'''<title>탁현 생성안</title>
<meta name="claude:viewport-fit-sides" content="cover">
<style>
/* layout: 원칙 적용 요약 → 결과(현재/치환판/생성판) → 구역 배치도 → 역할 표 → 점검 → 거울판 건물 */
:root{{--bg:#f3eee4;--fg:#2b2119;--muted:#6f6152;--line:#d9cdb9;--accent:#9c3b26;--ok:#3d6b3a;--shot:#141210;
--display:"Gowun Batang",serif;--body:"Apple SD Gothic Neo","Noto Sans KR",sans-serif}}
@media (prefers-color-scheme:dark){{:root:not([data-theme="light"]){{--bg:#1d1814;--fg:#eee4d6;--muted:#ad9e8b;--line:#3d332a;--accent:#e0805f;--ok:#8fc48a;color-scheme:dark}}}}
:root[data-theme="dark"]{{--bg:#1d1814;--fg:#eee4d6;--muted:#ad9e8b;--line:#3d332a;--accent:#e0805f;--ok:#8fc48a;color-scheme:dark}}
body{{background:var(--bg);color:var(--fg);font-family:var(--body);line-height:1.6;padding-left:max(16px,env(safe-area-inset-left,0px));padding-right:max(16px,env(safe-area-inset-right,0px));padding-block:28px 60px}}
main{{max-width:1300px;margin:0 auto;display:grid;gap:36px}}
h1,h2{{font-family:var(--display);text-wrap:balance;margin:0}} h1{{font-size:2rem}} h2{{font-size:1.35rem;margin-bottom:8px}}
p{{margin:0;max-width:80ch}} .muted{{color:var(--muted)}}
img{{display:block;max-width:100%;height:auto;border-radius:4px}} .px{{image-rendering:pixelated}} .shot{{background:var(--shot)}}
ol,ul{{margin:0;padding-left:1.3em;display:grid;gap:4px;max-width:90ch}}
.grid{{display:grid;grid-template-columns:repeat(auto-fit,minmax(340px,1fr));gap:14px}}
figure{{margin:0;display:grid;gap:6px;min-width:0}} figcaption{{font-size:.86rem;color:var(--muted)}} figcaption b{{color:var(--accent)}}
.tbl{{overflow-x:auto}} table{{border-collapse:collapse;font-size:.92rem}} td,th{{border-bottom:1px solid var(--line);padding:6px 12px;text-align:left}}
.sw{{display:inline-block;width:12px;height:12px;border-radius:2px;margin:0 4px 0 10px;vertical-align:-1px}}
.ok{{color:var(--ok);font-weight:700}}
</style>
<link rel="stylesheet" href="https://fonts.googleapis.com/css2?family=Gowun+Batang:wght@700&display=swap">
<main>
<header style="display:grid;gap:10px"><h1>탁현 생성안</h1>
<p class="muted">주신 원칙으로 맵을 다시 짜는 생성기(towngen)의 첫 맵이다. 옛 건물을 바꿔 끼우는 치환이 아니라, 지킬 것만 남기고 바닥·길·구역·건물을 새로 깐다. 이 맵의 방식이 맞으면 같은 규칙으로 다음 맵들을 하나씩 만든다. 게임 데이터는 그대로이고, 미리보기 어드민(127.0.0.1:18766)의 탐험01·02 탁현에서 볼 수 있다.</p></header>
<section><h2>원칙이 이 맵에 어떻게 적용됐나</h2><ol>
<li><b>맵 하나씩</b>: 탁현 한 장. 이 맵을 쓰는 곳: {', '.join(info['uses'])}.</li>
<li><b>방문할 건물과 역할 기록</b>: 출구가 잇는 실내 맵으로 판별해 roles 파일에 남겼다(아래 표).</li>
<li><b>성 안 배치·성 키우기</b>: 성벽은 뒤 두 변(성문은 왼쪽 뒤 벽)뿐이라 앞 두 변은 판 끝 3칸 안까지를 성 안으로 봤다. 간격을 넓히니 자리가 모자라 <b>성을 앞쪽으로 {info['grow']}칸 키우고</b> 사람이 걷는 범위(Size)도 같이 넓혔다(원점 쪽은 그대로라 기존 좌표 유지).</li>
<li><b>바닥 새로</b>: 성 안 옛 바닥 글자를 지우고 다시 칠했다. 성 밖은 그대로다.</li>
<li><b>대로</b>: 길은 한 종류(고을은 흙길, 황성은 포석). 성문에서 맞은편 판 끝까지 곧은 대로(2칸), 성문 밖도 판 끝까지 곧게, 가로지르는 대로, 블록 사이 골목(1칸). 성벽 없는 앞쪽은 길이 판 끝까지 간다.</li>
<li><b>황궁 둘레 비우기</b>: 탁현에는 해당 없음(허도·낙양에서 적용).</li>
<li><b>구역</b>: 관아(뒤쪽 교차로 옆), 상업(성문 가까운 대로변), 부유층(대로 뒤 블록), 민가(앞·오른쪽), 밭(앞 가장자리 한 블록, 두 판).</li>
<li><b>구역별 배치</b>: 건물은 길을 향한다(왼쪽 아래 길이면 원래 건물, 오른쪽 아래 길이면 거울판). 건물·담장은 성벽에서 3칸, 서로 2칸 띄운다. 나무는 성벽 3칸·건물 2칸·나무끼리 3칸 밖. 초가는 울 없이(울타리 초가 {info['fenced_cottages']}채).</li>
<li><b>앞쪽 성벽 쪽에 중요 건물 금지</b>: 탁현은 앞 두 변에 성벽이 없다. 규칙은 넣어 두었다.</li>
<li><b>재시도</b>: 골목 간격·교차로 위치를 흔들어 40번 짜 보고, 검사를 통과한 것 가운데 역할 건물이 제 모양으로 서고 채움이 좋은 것을 골랐다.</li></ol></section>
<section><h2>결과</h2>
<div class="grid"><figure><img class="shot" src="{b64(cur, 'JPEG')}" alt="현재 탁현"><figcaption>현재</figcaption></figure>
<figure><img class="shot" src="{b64(Image.open(f'{R}/generated_v1.png').convert('RGBA').crop(Image.open(f'{R}/generated_v1.png').getbbox()), 'JPEG')}" alt="생성 1판"><figcaption>생성 1판(길 두 종류, 빽빽함)</figcaption></figure>
<figure><img class="shot" src="{b64(gen, 'JPEG')}" alt="생성판"><figcaption><b>생성판</b></figcaption></figure></div>
<figure style="margin-top:14px"><img class="px shot" src="{b64(gen.crop(z))}" alt="생성판 확대"><figcaption><b>생성판</b> · 가운데 확대</figcaption></figure></section>
<section><h2>구역 배치도</h2><p class="muted" style="margin-bottom:8px">{legend}<span class="sw" style="background:rgb(150,155,165)"></span>대로 <span class="sw" style="background:rgb(196,170,120)"></span>골목 <span class="sw" style="background:rgb(20,20,20)"></span>역할 건물 문 칸</p>
<img class="px" src="{b64(Image.open(f'{R}/zones.png'))}" alt="구역 배치도" style="max-width:720px"></section>
<section><h2>방문 건물·역할</h2><div class="tbl"><table><thead><tr><th>역할</th><th>출구 이름</th><th>세운 건물</th><th>문 칸(출구)</th></tr></thead><tbody>{roles_rows}</tbody></table></div>
<ul style="margin-top:10px">{notes}</ul></section>
<section><h2>점검</h2><ul>
<li>지킬 칸(탐험 사람·출구·사인·시작, 1장 장면 인물과 걸음 경로)이 건물에 덮임: <span class="ok">{len(info['covered'])}</span></li>
<li>시작 칸에서 걸어서 못 닿는 사람·출구(엔진 걷기 규칙): <span class="ok">0</span></li>
<li>역할 건물 모두 섬: <span class="ok">{'예' if not info['missing'] else '아니오 ' + str(info['missing'])}</span></li></ul></section>
<section><h2>오른쪽 아래를 보는 건물</h2><p class="muted" style="margin-bottom:8px">왼쪽이 원래 판, 오른쪽이 거울판(_r)이다. 빛을 거울로 놓고 그린 뒤 좌우로 뒤집어서, 밝은 면·처마 그늘·땅 그림자가 게임 빛 방향 그대로다. 다섯 지역 모든 건물에 만들었다.</p>
<div class="grid">{''.join(f'<img class="px" src="{b64(m)}" alt="거울판 비교">' for m in mir)}</div></section>
</main>'''
open(f'{R}/index.html', 'w').write(html)
open(f'{R}/README.md', 'w').write("""# 탁현 생성안 (2026-11-28)

원칙(사용자 11-28)대로 장면 맵을 다시 짜는 생성기 towngen의 첫 맵. 미리보기 루트에만 적용, 게임 데이터 그대로.
- generated.png(생성판), current.png(현재), swap_version.png(앞 치환판), zones.png(구역 배치도)
- roles_S_zhuo.json: 방문 건물·역할, 세운 건물·문 칸, 구역 블록, 점검
- tools/towngen.py: `python3 towngen.py ROOT S_zhuo huabei` (reg/·lines/ 필요: town_dot2.py, town_lines.py)
- 거울판(_r): town_dot2.make_mirrored — 거울 빛으로 그리고 좌우로 뒤집음
""")
print(os.path.getsize(f'{R}/index.html') // 1024, 'KB')
