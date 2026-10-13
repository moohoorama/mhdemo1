from pathlib import Path
import sys
sys.path.insert(0,str(Path(__file__).resolve().parents[3]/'tools'))
import dots,render
from PIL import Image
hero=dots.load_working('archer'); base=dots.with_alpha(dots.load('infantry'))
for d in hero['unit']['rows']:
    for n in range(4):
        pics=[render.grid_lines(render.on_bg(render.image(doc,dots.frame(doc,'block',d,n)),10,doc['unit']['pivot']),10) for doc in (hero,base)]
        out=Image.new('RGBA',(1130,460),(30,32,34,255))
        out.paste(pics[0],(0,0)); out.paste(pics[1],(570,0))
        render.save(out,hero,f'frame-block-{d}-{n}.png')
    render.save(render.block(hero,['idle','block'],[d],6),hero,f'sheet-idle_block-{d}.png')
render.save(render.block(hero,['block'],hero['unit']['rows'],5),hero,'sheet-block-N_NE_E_SE_S_SW_W_NW.png')
for group in ('faction','themeA','themeB','themeC','skin','iron'):
    a=render.block(hero,['block'],hero['unit']['rows'],4)
    b=render.block(hero,['block'],hero['unit']['rows'],4,render.shifted(hero,group,150))
    out=Image.new('RGBA',(a.width*2+16,a.height),(90,20,20,255))
    out.paste(a,(0,0)); out.paste(b,(a.width+16,0))
    render.save(out,hero,f'recolor-{group}-block.png')
