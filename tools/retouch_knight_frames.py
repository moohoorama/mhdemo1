#!/usr/bin/env python3
"""Per-frame native pixel corrections after review of all 160 v4 frames.

These are local edits, not another crop/rotation pass. Explicit masks restore
occluded armor, attach equipment and remove identified stationary cut remnants.
"""
import json
from pathlib import Path
from PIL import Image, ImageDraw

ROOT=Path(__file__).resolve().parents[1]
SOURCE=ROOT/'tools/spritetool/assets/knights-pixel-v4'
OUTPUT=ROOT/'tools/spritetool/assets/knights-pixel-v5'
DIRS=['N','NE','E','SE','S','SW','W','NW']
ACTS=['idle','walk','attack','hit','exhausted']
manifest=json.loads((SOURCE/'frames.json').read_text())
PALETTE={c:tuple(bytes.fromhex(h[1:]))+(255,) for c,h in zip('0123456789ABCDEFGHIJKLMNOPQR',manifest['palette'])}
PALETTE['.']=(0,0,0,0)

# Direction, action, 1-based frame -> list of reviewed local changes.
EDITS={}
def add(d,a,frames,op,note,**args):
    for f in frames:EDITS.setdefault((d,a,f),[]).append(dict(op=op,note=note,**args))

add('NE','walk',[1],'fill','Reconnect far arm to shoulder',polygon=[(16,23),(19,23),(19,27),(16,26)],color='7')
add('NE','walk',[1],'fill','Reconnect sword-side elbow',polygon=[(27,24),(30,25),(31,29),(28,29),(27,27)],color='8')
add('NE','walk',[1],'pixels','Shade the two repaired elbow joints',pixels=[[17,24,'8'],[18,24,'9'],[17,25,'8'],[18,25,'8'],[28,26,'9'],[28,27,'8'],[29,28,'7']])
add('SE','walk',[3],'fill','Restore shield arm across displaced shield gap',polygon=[(28,24),(31,25),(31,29),(28,29)],color='7')
add('SE','walk',[3],'pixels','Shield-arm shading',pixels=[[29,25,'9'],[30,26,'8'],[29,26,'8'],[29,27,'8'],[30,27,'7'],[29,28,'0']])
add('SE','walk',[3],'restore','Close torso cut beside sword arm',polygon=[(21,24),(24,24),(24,31),(21,31)],color='7')
add('SW','walk',[1],'fill','Restore chest exposed by shifting both arms without duplicating old shield',polygon=[(21,24),(25,24),(25,31),(21,30)],color='8')
add('SW','walk',[1],'pixels','Continue chest sash across repaired pixels',pixels=[[21,26,'C'],[21,27,'D'],[21,28,'C'],[23,25,'9'],[24,26,'9'],[24,29,'7']])
add('SW','walk',[1],'erase','Remove narrow helmet/sword sliver carried by arm mask',box=[17,17,19,24])
add('SW','walk',[1],'erase','Remove upper end of same sliver beside blade',box=[16,19,17,22])
add('SW','walk',[3],'restore','Close horizontal chest cut above shield',polygon=[(21,24),(26,24),(26,27),(22,29)],color='8')
add('W','walk',[1],'restore','Restore missing vertical chest strip',polygon=[(21,24),(24,24),(24,33),(21,33)],color='8')
add('NW','walk',[3],'restore','Restore torso exposed by moving shield',polygon=[(20,23),(24,23),(24,31),(20,30)],color='8')

add('W','walk',[1],'pixels','Remove detached sword-mask pixel',pixels=[[18,13,'.']])
add('W','walk',[1],'erase','Remove old blade sliver next to helmet',box=[18,15,20,19])
add('W','walk',[3],'erase','Remove stationary shield-bottom trace',box=[14,34,18,35])
add('W','hit',[3,4],'erase','Remove five-pixel stationary shield-bottom trace',box=[16,34,21,35])
add('S','hit',[3,4],'erase','Remove old shield stem beneath raised shield',box=[27,29,28,33])
add('S','hit',[3,4],'erase','Remove stationary shield-bottom rectangle',box=[28,33,32,34])
add('S','hit',[3],'pixels','Close rotation pinholes in shield wood',pixels=[[28,26,'G'],[28,27,'H'],[28,28,'G']])
add('SE','hit',[3,4],'erase','Remove duplicated sword-hilt pixels at original hand position',box=[17,31,21,33])
add('SE','hit',[3,4],'erase','Remove stationary shield fragments beside waist',box=[27,28,30,32])
add('SE','hit',[3,4],'pixels','Remove last shield-outline pixel',pixels=[[30,31,'.']])
add('SE','hit',[3,4],'fill','Repair hip articulation after removing duplicated hand',polygon=[(20,30),(24,29),(26,32),(21,33),(19,32)],color='7')
add('SE','hit',[3,4],'pixels','Shade hip joint',pixels=[[21,31,'8'],[22,31,'9'],[21,32,'8']])
add('SE','hit',[3],'pixels','Close two shield rotation pinholes',pixels=[[27,21,'G'],[27,23,'G']])
add('SW','hit',[3],'fill','Connect rotated chest to anchored pelvis',polygon=[(22,29),(25,28),(25,31),(22,33)],color='7')
add('SW','hit',[3],'pixels','Shade pelvis articulation',pixels=[[23,30,'8'],[23,31,'9'],[23,32,'8'],[24,30,'8']])
add('NW','walk',[3],'pixels','Remove remaining toe-mask spur',pixels=[[27,35,'.']])
add('SW','walk',[3],'erase','Remove dangling leg-mask pixels beneath stance',box=[21,37,22,39])

def apply(im,base,e):
    if e['op']=='erase':
        im.paste((0,0,0,0),tuple(e['box']))
    elif e['op']=='pixels':
        for x,y,c in e['pixels']:im.putpixel((x,y),PALETTE[c])
    else:
        mask=Image.new('1',(48,48));ImageDraw.Draw(mask).polygon([tuple(p) for p in e['polygon']],fill=1)
        for y in range(48):
            for x in range(48):
                if mask.getpixel((x,y)) and not im.getpixel((x,y))[3]:
                    p=base.getpixel((x,y)) if e['op']=='restore' else (0,0,0,0)
                    im.putpixel((x,y),p if p[3] else PALETTE[e['color']])

def main():
    OUTPUT.mkdir(parents=True,exist_ok=True);audit=[]
    for d in DIRS:
        sheet=Image.open(SOURCE/f'{d}-pixel.png').convert('RGBA');base=sheet.crop((0,0,48,48))
        for row,a in enumerate(ACTS):
            for f in range(1,5):
                box=((f-1)*48,row*48,f*48,(row+1)*48)
                im=sheet.crop(box);before=im.copy();edits=EDITS.get((d,a,f),[])
                for edit in edits:apply(im,base,edit)
                changed=sum(p!=q for p,q in zip(before.getdata(),im.getdata()))
                audit.append(dict(direction=d,animation=a,frame=f,reviewed=True,changed_pixels=changed,
                                  changes=edits or ['No local clipping, orphan pixels or broken joints found in enlarged review.']))
                sheet.paste(im,box[:2])
        sheet.save(OUTPUT/f'{d}-pixel.png')
    manifest.update(version=5,source_recipe='tools/retouch_knight_frames.py',
                    retouch_policy='Per-frame local pixel edits on v4; retained pivot, scale, palette and motion.')
    (OUTPUT/'frames.json').write_text(json.dumps(manifest,indent=2)+'\n')
    (OUTPUT/'review.json').write_text(json.dumps(audit,indent=2)+'\n')
    print(f'Reviewed {len(audit)} frames; retouched {sum(a["changed_pixels"]>0 for a in audit)}.')

if __name__=='__main__':main()
