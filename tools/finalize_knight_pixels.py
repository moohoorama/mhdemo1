#!/usr/bin/env python3
"""Normalize ImageGen redraws into native pixel artwork. Requires Pillow and numpy.

Character-specific helmet landmarks belong here, not in the generic spritetool.
The original/redrawn artwork is retained; this recipe writes new native sheets.
"""
import json
import math
from pathlib import Path
from collections import deque
import numpy as np
from PIL import Image

ROOT = Path(__file__).resolve().parents[1]
SOURCE = ROOT / 'tools/spritetool/assets/knights-pixel-v2'
DIRS = ['N', 'NE', 'E', 'SE', 'S', 'SW', 'W', 'NW']
ANIMATIONS = ['idle', 'walk', 'attack', 'hit', 'exhausted']
# First-frame helmet body landmarks, excluding plume, in 1024x1536 redraws.
BOX = {'N': (82, 83, 169, 149), 'NE': (83, 83, 162, 145),
       'E': (84, 86, 161, 157), 'SE': (91, 95, 177, 159),
       'S': (89, 96, 175, 164), 'SW': (88, 99, 164, 166),
       'W': (95, 97, 161, 158), 'NW': (94, 107, 173, 169)}
COLORS = ['0b101b','222b38','3f4a56','637280','93a3af','c4d2d9','f1f6f5',
          '102441','193b65','245784','3974a0',
          '650d24','ab1633','ed2342','ff5362',
          '3d211d','663720','96522b','c4783c','e2a061',
          'ad6318','e69d23','ffd052','ffe89a',
          '247ec5','49adf0','8cd5ff','d4f3ff']
PALETTE = np.array([[int(c[i:i+2],16) for i in (0,2,4)] for c in COLORS], dtype=np.float32)

def match(crop, template):
    small = np.asarray(crop.resize((64,77),Image.Resampling.BOX)).astype(np.float32)/255
    best = None
    for scale in [.9,.95,1,1.05,1.1]:
        tw,th = round(template.width*scale/4),round(template.height*scale/4)
        t = np.asarray(template.resize((tw,th),Image.Resampling.BOX)).astype(np.float32)/255
        win = np.lib.stride_tricks.sliding_window_view(small[8:60,10:57],(th,tw),axis=(0,1)).transpose(0,1,3,4,2)
        weight = (t[:,:,3]>.7).astype(np.float32)
        score = (((win[:,:,:,:,:3]-t[None,None,:,:,:3])**2).sum(axis=-1)*weight).sum(axis=(-1,-2))/max(1,weight.sum())
        score += ((win[:,:,:,:,3]-t[None,None,:,:,3])**2).mean(axis=(-1,-2))*.3
        iy,ix = np.unravel_index(score.argmin(),score.shape)
        result = (float(score[iy,ix]),float((ix+10)*4),float((iy+8)*4),tw*4,th*4)
        if best is None or result[0]<best[0]: best=result
    return best

def foot(crop):
    a=np.array(crop).astype(float);r,g,b,alpha=[a[:,:,i] for i in range(4)]
    yy=np.indices(r.shape)[0]
    mask=(alpha>180)&(r>g*1.2)&(g>b*1.15)&(r>45)&(yy>crop.height*.57)
    ys=np.where(mask)[0]
    if len(ys): return float(np.quantile(ys,.99))+3
    ys=np.where(alpha>180)[0]
    return float(np.quantile(ys,.995))+1

def crisp(im):
    a=np.array(im)
    colors=a[:,:,:3].astype(np.float32)
    distance=((colors[:,:,None,:]-PALETTE[None,None,:,:])**2).sum(axis=-1)
    a[:,:,:3]=PALETTE[distance.argmin(axis=-1)].astype(np.uint8)
    a[:,:,3]=np.where(a[:,:,3]>=128,255,0)
    # Remove disconnected tiny debris; keep sword trails and connected outlines.
    seen=set()
    for y,x in zip(*np.where(a[:,:,3]>0)):
        if (y,x) in seen: continue
        q=deque([(y,x)]);seen.add((y,x));component=[]
        while q:
            cy,cx=q.popleft();component.append((cy,cx))
            for dy in (-1,0,1):
                for dx in (-1,0,1):
                    ny,nx=cy+dy,cx+dx
                    if 0<=ny<48 and 0<=nx<48 and a[ny,nx,3] and (ny,nx) not in seen:
                        seen.add((ny,nx));q.append((ny,nx))
        if len(component)<5:
            for cy,cx in component:a[cy,cx,3]=0
    a[a[:,:,3]==0]=0
    return Image.fromarray(a)

def main():
    data={'version':2,'directions':DIRS,'animations':ANIMATIONS,'cell':[48,48],
          'pivot':[24,38],'target_head_geometric_size':10,'palette':['#'+c for c in COLORS],
          'idle_policy':'four pixel-identical frames','frames':[]}
    for di,d in enumerate(DIRS):
        src=Image.open(SOURCE/f'{d}-redrawn.png').convert('RGBA')
        assert src.size==(1024,1536), (d,src.size)
        template=src.crop(BOX[d]);sheet=Image.new('RGBA',(192,240))
        for row,action in enumerate(ANIMATIONS):
            group=[]
            for col in range(4):
                x=col*256;y=round(row*1536/5);y2=round((row+1)*1536/5)
                crop=src.crop((x,y,x+256,y2));error,hx,hy,hw,hh=match(crop,template)
                # The raised hand occludes E's first attack helmet; use the reviewed landmark.
                if d=='E' and row==2 and col==0:
                    hx,hy,hw,hh=100,68,80,72
                scale=10/math.sqrt(hw*hh);cx=hx+hw/2;cy=hy+hh/2
                group.append((crop,cx,cy,scale,foot(crop),error,hw,hh))
            centerx=float(np.median([g[1] for g in group]))
            ground=float(np.median([(g[4]-g[2])*g[3] for g in group]))
            centery=float(np.median([g[2] for g in group]))
            idle=None
            for col,(crop,cx,cy,scale,bottom,error,hw,hh) in enumerate(group):
                targetx=24 if row==0 else 24+(cx-centerx)*scale
                targety=38-ground+(0 if row==0 else (cy-centery)*scale)
                resized=crop.resize((round(crop.width*scale),round(crop.height*scale)),Image.Resampling.BOX)
                frame=Image.new('RGBA',(48,48))
                frame.alpha_composite(resized,(round(targetx-cx*scale),round(targety-cy*scale)))
                frame=crisp(frame)
                if row==0:
                    if idle is None:idle=frame.copy()
                    frame=idle.copy()
                sheet.alpha_composite(frame,(col*48,row*48))
                data['frames'].append({'direction':d,'animation':action,'frame':col,
                    'index':di*20+row*4+col,'crop':[col*48,row*48,48,48],
                    'helmet_dimensions':[round(hw*scale,3),round(hh*scale,3)],
                    'head_match_error':error,'scale':scale})
        sheet.save(SOURCE/f'{d}-pixel.png')
        print(d, 'native pixel sheet ready',flush=True)
    (SOURCE/'frames.json').write_text(json.dumps(data,indent=2)+'\n')

if __name__=='__main__':main()
