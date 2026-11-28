import base64, io, os, numpy as np
from PIL import Image
R='/Users/younwookim/orca/mhdemo1/result/20261128/region-towns-in-scene'
def uri(im):
    q=im.convert('RGBA').quantize(colors=256, method=Image.Quantize.FASTOCTREE)
    b=io.BytesIO(); q.save(b,'PNG',optimize=True); return 'data:image/png;base64,'+base64.b64encode(b.getvalue()).decode()
trim=lambda im: im.crop(im.getbbox())
P=[('S_zhuo','탁현','화북','두꺼운 흙벽 · 완만하고 투박한 초가 우진각 · 밭 · 땅을 누렇게 말림','S_zhuo'),
   ('S_luoyang','낙양','중원','흙벽에 나무 뼈대 · 단정한 초가 맞배 · 밭','S_luoyang'),
   ('S_xuzhou_town','서주','회남','갈대 지붕 · 습한 흙벽 · 논과 밭','S_xuzhou_town'),
   ('S_xiangyang','양양','회남','갈대 지붕 · 습한 흙벽 · 논과 밭','S_xiangyang'),
   ('S_demo_jiangnan','강남','강남','양양 지형을 빌림 · 고상 판자집 · 처마 긴 기와 · 논 · 짙은 풀','S_xiangyang'),
   ('S_demo_bashu','파촉','파촉','탁현 지형을 빌림 · 대나무 벽 · 조금 가파른 지붕 · 논 · 짙은 녹색 풀','S_zhuo')]
secs=nav=''
for nid,name,reg,desc,base in P:
    new=Image.open(f'{R}/new/{nid}.png').convert('RGBA'); cur=Image.open(f'{R}/current/{base}.png').convert('RGBA')
    old=Image.open(f'{R}/v1_steep/{nid}.png').convert('RGBA')
    a=np.asarray(new).astype(int); b=np.asarray(cur).astype(int)
    if nid in ('S_zhuo','S_demo_bashu','S_demo_jiangnan'):
        ys,xs=np.nonzero(a[...,3]>0); x0,x1,y0,y1=xs.min(),xs.max(),ys.min(),ys.max()
        box=(x0+(x1-x0)*.18, y0+(y1-y0)*.18, x0+(x1-x0)*.78, y0+(y1-y0)*.72)
    else:
        ys,xs=np.nonzero(np.abs(a-b).sum(-1)>60); box=(xs.min()-20,ys.min()-40,xs.max()+20,ys.max()+20)
    box=tuple(int(v) for v in box)
    if box[2]-box[0]>1100:
        c=(box[0]+box[2])//2; box=(c-550,box[1],c+550,box[3])
    label = f'현재 {name}' if base==nid else f'현재 {"양양" if base=="S_xiangyang" else "탁현"} (원본 지형)'
    nav+=f'<a href="#{nid}">{name}</a>'
    secs+=f'''<section id="{nid}" class="town"><header><h2>{name} <small>{reg}</small></h2><p class="muted">{desc}</p></header>
<div class="pair"><figure><img src="{uri(trim(cur))}" alt="{label} 전체"><figcaption>{label} · 전체</figcaption></figure>
<figure><img src="{uri(trim(new))}" alt="새 {name} 전체"><figcaption><b>새 건물 · {reg}</b> · 전체</figcaption></figure></div>
<div class="zoom"><figure><img src="{uri(cur.crop(box))}" alt="{label} 확대"><figcaption>{label} · 확대</figcaption></figure>
<figure><img src="{uri(new.crop(box))}" alt="새 {name} 확대"><figcaption><b>새 건물 · {reg}</b> · 확대 (낮은 지붕, 처마 밑 그늘)</figcaption></figure>
<details><summary>직전판(가파른 지붕, 처마 그늘 없음) 확대</summary><img src="{uri(old.crop(box))}" alt="직전판 {name} 확대"></details></div></section>'''
html=f'''<title>지역별 성 안 장면</title>
<meta name="claude:viewport-fit-sides" content="cover">
<style>
/* layout: 도시 바로가기 → 도시마다 전체 두 장(현재/새), 확대(현재/새/직전판) */
:root{{--bg:#f3eee4;--fg:#2b2119;--muted:#6f6152;--line:#d9cdb9;--chip:#e6dccb;--accent:#9c3b26;--shot:#ffffff;
--display:"Gowun Batang",serif;--body:"Apple SD Gothic Neo","Noto Sans KR",sans-serif}}
@media (prefers-color-scheme:dark){{:root:not([data-theme="light"]){{--bg:#1d1814;--fg:#eee4d6;--muted:#ad9e8b;--line:#3d332a;--chip:#2e2620;--accent:#e0805f;--shot:#0d0b09;color-scheme:dark}}}}
:root[data-theme="dark"]{{--bg:#1d1814;--fg:#eee4d6;--muted:#ad9e8b;--line:#3d332a;--chip:#2e2620;--accent:#e0805f;--shot:#0d0b09;color-scheme:dark}}
body{{background:var(--bg);color:var(--fg);font-family:var(--body);line-height:1.6;padding-left:max(16px,env(safe-area-inset-left,0px));padding-right:max(16px,env(safe-area-inset-right,0px));padding-block:28px 60px}}
main{{max-width:1400px;margin:0 auto;display:grid;gap:44px}}
h1,h2{{font-family:var(--display);text-wrap:balance;margin:0}} h1{{font-size:2rem}} h2{{font-size:1.5rem}} h2 small{{font-size:1rem;color:var(--accent);margin-left:6px}}
p{{margin:0;max-width:75ch}} .muted{{color:var(--muted)}}
img{{image-rendering:pixelated;display:block;max-width:100%;height:auto;background:var(--shot);border-radius:4px}}
nav{{display:flex;flex-wrap:wrap;gap:8px;position:sticky;top:env(safe-area-inset-top,0px);background:var(--bg);padding-block:8px;z-index:2}}
nav a{{background:var(--chip);color:var(--fg);text-decoration:none;padding:6px 14px;border-radius:999px;font-weight:600}}
nav a:hover,nav a:focus-visible{{outline:2px solid var(--accent)}}
.town{{border-top:1px solid var(--line);padding-top:20px;display:grid;gap:16px;scroll-margin-top:60px}}
.pair{{display:grid;grid-template-columns:repeat(auto-fit,minmax(320px,1fr));gap:16px}}
.zoom{{display:grid;gap:12px}}
figure{{margin:0;display:grid;gap:6px;min-width:0}} figcaption{{font-size:.88rem;color:var(--muted)}} figcaption b{{color:var(--accent)}}
details summary{{cursor:pointer;color:var(--muted);font-size:.9rem;margin-bottom:6px}}
ul{{margin:0;padding-left:1.2em;display:grid;gap:4px;max-width:80ch}}
</style>
<link rel="stylesheet" href="https://fonts.googleapis.com/css2?family=Gowun+Batang:wght@700&display=swap">
<main>
<header style="display:grid;gap:10px"><h1>지역별 성 안 장면</h1>
<p class="muted">시나리오 어드민과 같은 맵 렌더러(maplight)로 그렸다. 현재 장면 맵의 집과 관아를 지역 건물로 바꾸고, 빈 땅에 초가집과 밭·논을 조금 더 놓았다.</p>
<ul><li><b>이번 판 (11-28 피드백 반영)</b>: 지붕 경사를 기존 관청처럼 낮췄다. 기와는 칸당 18~22px에서 11~13px로, 초가는 18px에서 13px로 내렸다. 용마루 끝 장식도 낮췄다.</li>
<li><b>처마 밑 그늘</b>: 벽면에서 바깥쪽 위로 광선을 쏴서 처마나 윗지붕에 막히면 어둡게 칠한다. 해 그림자(건물끼리 가리는 그늘)도 같이 계산한다.</li>
<li>나무, 노점, 성벽, 성문은 지금 것 그대로다. 강남과 파촉은 양양과 탁현 지형을 빌렸다.</li>
<li>게임 데이터는 바꾸지 않았다. 새판은 미리보기 어드민(포트 18766)에서도 볼 수 있다.</li></ul></header>
<nav>{nav}</nav>
{secs}
</main>'''
open(f'{R}/index.html','w').write(html)
print(os.path.getsize(f'{R}/index.html')//1024,'KB')
