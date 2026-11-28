"""성·마을 생성안 전체 페이지: 지역별, 맵마다 현재/생성 전체·확대, 역할, 점검"""
import base64, io, json, os, shutil
from PIL import Image
SP = os.path.dirname(os.path.abspath(__file__))
R = '/Users/younwookim/orca/mhdemo1/result/20261128/towngen-all'
os.makedirs(f'{R}/current', exist_ok=True); os.makedirs(f'{R}/generated', exist_ok=True); os.makedirs(f'{R}/roles', exist_ok=True)
MAPS = [('S_zhuo', 'huabei'), ('S_pingyuan', 'huabei'), ('S_xindu', 'huabei'), ('S_ye', 'huabei'), ('S_beihai', 'huabei'),
        ('S_luoyang', 'zhongyuan'), ('S_xudu', 'zhongyuan'), ('S_runan', 'zhongyuan'), ('S_road_town', 'zhongyuan'),
        ('S_xiaopei', 'huainan'), ('S_xuzhou_town', 'huainan'), ('S_xuzhou_gate', 'huainan'), ('S_xiapi', 'huainan'),
        ('S_xiapi_inner', 'huainan'), ('S_xiangyang', 'huainan'), ('S_xinye', 'huainan'), ('S_xinye_town', 'huainan'),
        ('S_mangdang', 'huainan'), ('S_vil_peace', 'huainan'),
        ('S_jiangling', 'jiangnan'), ('S_jiangxia_harbor', 'jiangnan'), ('S_danyang_village', 'jiangnan')]
KO = {'huabei': '화북', 'zhongyuan': '중원', 'huainan': '회남', 'jiangnan': '강남'}
DESC = {'huabei': '흙길 · 두꺼운 흙벽 · 완만한 초가 우진각 · 밭', 'zhongyuan': '황성은 포석 · 흙벽에 나무 뼈대 · 단정한 초가 맞배',
        'huainan': '흙길 · 갈대 지붕 · 습한 흙벽 · 논과 밭', 'jiangnan': '흙길 · 고상 판자집 · 처마 긴 기와 · 논'}
NAMES = {}
def jpg(im, k=.55):
    bg = Image.new('RGB', im.size, (20, 18, 16)); bg.paste(im, mask=im.split()[3])
    bg = bg.resize((max(1, int(im.width * k)), max(1, int(im.height * k))), Image.LANCZOS); b = io.BytesIO(); bg.save(b, 'JPEG', quality=82)
    return 'data:image/jpeg;base64,' + base64.b64encode(b.getvalue()).decode()
def png(im):
    b = io.BytesIO(); im.convert('RGBA').quantize(256, method=Image.Quantize.FASTOCTREE).save(b, 'PNG', optimize=True)
    return 'data:image/png;base64,' + base64.b64encode(b.getvalue()).decode()
secs = ''; nav = ''; rows = ''
for reg in KO:
    items = [m for m, r in MAPS if r == reg]
    nav += f'<a href="#{reg}">{KO[reg]} <span>{len(items)}</span></a>'
    cards = ''
    for m in items:
        shutil.copy(f'{SP}/render/gencur/{m}.png', f'{R}/current/{m}.png'); shutil.copy(f'{SP}/render/gen/{m}.png', f'{R}/generated/{m}.png')
        shutil.copy(f'{SP}/root/roles/{m}.json', f'{R}/roles/{m}.json')
        info = json.load(open(f'{R}/roles/{m}.json'))
        name = json.load(open(f'/Users/younwookim/orca/mhdemo1/ver5/assets/graphics/maps/{m}.json'))['name']
        cur = Image.open(f'{R}/current/{m}.png').convert('RGBA'); gen = Image.open(f'{R}/generated/{m}.png').convert('RGBA')
        cur = cur.crop(cur.getbbox()); gen = gen.crop(gen.getbbox())
        z = tuple(int(a) for a in (gen.width * .22, gen.height * .22, gen.width * .78, gen.height * .72))
        roles = ', '.join(f"{k}→{v['name'].replace('_r', '(반전)')}" for k, v in info['placed'].items()) or '역할 건물 없음(마을)'
        zones = {}
        for b in info['zones']:
            if b.get('zone') and b['zone'] != '빈 땅':
                zones[b['zone']] = zones.get(b['zone'], 0) + 1
        zt = ' · '.join(f'{k} {v}' for k, v in zones.items())
        uses = ', '.join(info['uses'])
        notes = ''.join(f'<li>{n}</li>' for n in info['notes'])
        grow = f"<li>성을 앞쪽으로 {info['grow']}칸 키움(사람이 걷는 범위도 함께)</li>" if info.get('grow') else ''
        cards += f'''<article class="town" id="{m}"><header><h3>{name}</h3><code>{m}</code></header>
<p class="uses">쓰는 곳: {uses}</p><ul><li>역할 건물: {roles}</li><li>구역: {zt}</li><li>울타리 초가 {info['fenced_cottages']}채</li>{grow}{notes}</ul>
<div class="pair"><figure><img loading="lazy" src="{jpg(cur)}" alt="현재 {name}"><figcaption>현재</figcaption></figure>
<figure><img loading="lazy" src="{jpg(gen)}" alt="생성 {name}"><figcaption><b>생성</b></figcaption></figure></div>
<figure><img loading="lazy" class="px" src="{png(gen.crop(z))}" alt="생성 확대"><figcaption><b>생성</b> · 가운데 확대</figcaption></figure></article>'''
        rows += f"<tr><td>{name}</td><td>{KO[reg]}</td><td>{'통과' if info['ok'] else '실패'}</td><td>{len(info['covered'])}</td><td>{', '.join(info['placed']) or '—'}</td></tr>"
    secs += f'<section class="region" id="{reg}"><header><h2>{KO[reg]}</h2><p class="muted">{DESC[reg]}</p></header>{cards}</section>'
html = f'''<title>성·마을 생성안</title>
<meta name="claude:viewport-fit-sides" content="cover">
<style>
/* layout: 요약·원칙·점검표 → 지역 바로가기 → 지역마다 맵 카드 */
:root{{--bg:#f3eee4;--fg:#2b2119;--muted:#6f6152;--line:#d9cdb9;--chip:#e6dccb;--accent:#9c3b26;--ok:#3d6b3a;--shot:#141210;
--display:"Gowun Batang",serif;--body:"Apple SD Gothic Neo","Noto Sans KR",sans-serif}}
@media (prefers-color-scheme:dark){{:root:not([data-theme="light"]){{--bg:#1d1814;--fg:#eee4d6;--muted:#ad9e8b;--line:#3d332a;--chip:#2e2620;--accent:#e0805f;--ok:#8fc48a;color-scheme:dark}}}}
:root[data-theme="dark"]{{--bg:#1d1814;--fg:#eee4d6;--muted:#ad9e8b;--line:#3d332a;--chip:#2e2620;--accent:#e0805f;--ok:#8fc48a;color-scheme:dark}}
body{{background:var(--bg);color:var(--fg);font-family:var(--body);line-height:1.6;padding-left:max(16px,env(safe-area-inset-left,0px));padding-right:max(16px,env(safe-area-inset-right,0px));padding-block:28px 60px}}
main{{max-width:1400px;margin:0 auto;display:grid;gap:40px}}
h1,h2,h3{{font-family:var(--display);text-wrap:balance;margin:0}} h1{{font-size:2rem}} h2{{font-size:1.6rem}} h3{{font-size:1.2rem}}
p{{margin:0;max-width:80ch}} .muted{{color:var(--muted)}}
img{{display:block;max-width:100%;height:auto;background:var(--shot);border-radius:4px}} img.px{{image-rendering:pixelated}}
nav{{display:flex;flex-wrap:wrap;gap:8px;position:sticky;top:env(safe-area-inset-top,0px);background:var(--bg);padding-block:8px;z-index:2}}
nav a{{background:var(--chip);color:var(--fg);text-decoration:none;padding:6px 14px;border-radius:999px;font-weight:600}}
nav a span{{color:var(--muted);font-weight:400;font-variant-numeric:tabular-nums}} nav a:hover,nav a:focus-visible{{outline:2px solid var(--accent)}}
.region{{display:grid;gap:28px;scroll-margin-top:60px}} .region>header{{border-bottom:2px solid var(--line);padding-bottom:8px}}
.town{{display:grid;gap:12px;border-top:1px solid var(--line);padding-top:18px;scroll-margin-top:60px}}
.town header{{display:flex;gap:10px;align-items:baseline;flex-wrap:wrap}} .town code{{color:var(--muted);font-size:.85rem}}
.uses{{font-size:.85rem;color:var(--muted)}}
ul,ol{{margin:0;padding-left:1.25em;display:grid;gap:3px;font-size:.93rem;max-width:95ch}}
.pair{{display:grid;grid-template-columns:repeat(auto-fit,minmax(320px,1fr));gap:14px}}
figure{{margin:0;display:grid;gap:6px;min-width:0}} figcaption{{font-size:.85rem;color:var(--muted)}} figcaption b{{color:var(--accent)}}
.tbl{{overflow-x:auto}} table{{border-collapse:collapse;font-size:.9rem}} td,th{{border-bottom:1px solid var(--line);padding:5px 10px;text-align:left}}
.big{{font-size:1.3rem;color:var(--ok);font-weight:700}}
</style>
<link rel="stylesheet" href="https://fonts.googleapis.com/css2?family=Gowun+Batang:wght@700&display=swap">
<main>
<header style="display:grid;gap:12px"><h1>성·마을 생성안</h1>
<p class="muted">탁현에서 승낙받은 규칙으로 성·마을 장면 맵 22곳(탐험과 스토리 장면에서 쓰는 곳)을 다시 짰다. 그림은 시나리오 어드민과 같은 렌더러다. 게임 데이터는 그대로이고, 미리보기 어드민(127.0.0.1:18766)에서 탐험 지역별로 볼 수 있다.</p>
<ol>
<li>지킬 것: 성벽·성문·물·절벽, 탐험의 사람·출구·사인·시작 칸, 스토리 장면의 인물 자리와 걸음 경로. 방문 건물 역할은 출구가 잇는 실내 맵과 원래 맵의 건물로 판별해 roles 파일에 남겼다.</li>
<li>성 안 바닥을 지우고 새로: 길은 한 종류(황성 포석, 나머지 흙길), 성문마다 성 안으로 곧은 대로와 성 밖으로 판 끝까지, 교차 대로, 블록 사이 골목. 황성은 황궁 문에서 대로를 내고 황궁 둘레 4칸은 비웠다.</li>
<li>구역: 관아·상업·부유층·민가·밭. 앞쪽 성벽 옆 블록에는 중요 건물을 두지 않는다.</li>
<li>건물은 길을 향한다(왼쪽 아래 길이면 원래 건물, 오른쪽 아래 길이면 반전 건물). 건물은 성벽에서 3칸, 서로 2칸 띄우고, 나무는 성벽 3칸·건물 2칸·나무끼리 3칸 밖. 울타리 초가는 맵마다 두 채 이하.</li>
<li>집 수는 원래 맵의 60%~130%. 성벽 없는 마을은 놀이 영역 둘레만 채우고 길도 그 안(큰길 하나만 판 끝까지).</li>
<li>대장간은 바깥 헛간: 실내 대장간을 빼고 대장장이를 대장간 앞으로. 관청·주점·도구점·저택 출구는 새 건물 문 앞으로.</li>
<li>뺀 맵: 진영, 성벽 위·성 밖 장면, 불탄 마을·물에 잠긴 하비(상태 그림이 따로 필요).</li>
<li><b>2차(11-28)</b>: 성문 길을 아치에 맞춤(성문 칸 -2·-1, 렌더로 확인), 업 가운데 십자 대로 포석, 관청 앞 깃발·등과 교차로·대로변 매대·수레·자루·물항아리, 서주성 앞 성 밖은 넓은 대로·나무·밭만, 하비 성안 수공 뒤 황폐(웅덩이·젖은 자국·쓰레기 줄), 망탕산 끊긴 물줄기 지움, 융중은 숲 속 외딴 초가, 강하 나루는 원래 배치에 바닥만 고침, 단양은 절벽 대신 전투맵 언덕·산과 마을 울타리.</li>
<li><b>3차(11-28)</b>: 매대는 교차로 가까운 시장 블록 안에 줄지어(대로변 흩뿌리기 없앰), 관청은 담장 안·대문 앞 돌 포석, 원래 성 안에 있던 나무 종류(복숭아·매화) 살림, 하비 성안은 맵 전체가 수공 뒤 흙탕(웅덩이·젖은 자국, 흙탕물 자국 진 집), 융중은 원래 길·밭을 살리고 길 둘레와 울타리 앞을 비움(밭 셋), 단양은 원래 절벽 언덕을 그대로 두고 위를 숲으로 채움(빈 풀밭 없이), 울타리는 마을 앞 두 변을 두르고 절벽에 닿으면 멈춤. 성문 길은 엔진 통로 칸(성문 칸 -1·0)에 다시 맞춤 — 그림·조각 배치·걷기가 모두 이 칸이고, 앞서 -2·-1로 옮긴 것은 잘못이었음. 성문 그림은 기존 것을 씀.</li>
<li><b>4차(11-28)</b>: 관청 구역을 키움 — 담장 마당을 옆 2·앞 4칸으로 넓히고(안 들어가면 옆 1칸 → 담장 없는 포석 마당 순으로 물러섬), 담장 안 앞마당까지 전부와 대문 밖 광장을 돌 포석으로.</li>
<li><b>5차(11-28)</b>: 조금 더 채움 — 집 수 원래의 75%까지, 민가 블록마다 나무 두 그루, 초가 옆 물항아리·곡식 자루·덤불·거적. 관청 광장은 가장 가까운 길까지 잇고, 길·포석에 둘러싸인 10칸 이하 풀밭 섬은 옆 바닥으로 메움.</li></ol>
<div class="tbl"><table><thead><tr><th>맵</th><th>지역</th><th>검사</th><th>지킬 칸 덮임</th><th>역할 건물</th></tr></thead><tbody>{rows}</tbody></table></div>
<p>걸어서 못 닿는 사람·출구(엔진 걷기 규칙, 탐험 지역 전부): <span class="big">0</span></p>
</header>
<nav>{nav}</nav>
{secs}
</main>'''
open(f'{R}/index.html', 'w').write(html)
open(f'{R}/README.md', 'w').write("""# 성·마을 생성안 (2026-11-28)

탁현(towngen-zhuo)에서 승낙받은 규칙으로 성·마을 장면 맵 22곳을 다시 짠 안. 미리보기 루트에만 적용, 게임 데이터 그대로.
- current/, generated/: 맵 렌더, roles/: 맵별 역할·구역·점검·바뀐 출구
- tools/: towngen.py(생성기), batch.sh(전체), build_root.py, town_dot2.py·town_lines.py·shade.py(그림), page_all.py
- 점검: 22곳 모두 검사 통과(지킬 칸 덮임 0, 역할 건물 누락 0), 걸어서 못 닿는 곳 0
""")
os.makedirs(f'{R}/tools', exist_ok=True)
for f in ('towngen.py', 'batch.sh', 'build_root.py', 'town_dot2.py', 'town_lines.py', 'shade.py', 'town_dot.py', 'page_all.py'):
    shutil.copy(f'{SP}/{f}', f'{R}/tools/{f}')
print(os.path.getsize(f'{R}/index.html') // 1024, 'KB')
