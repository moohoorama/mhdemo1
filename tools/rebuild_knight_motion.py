#!/usr/bin/env python3
"""Continuous upper-body motion and rigid boots connected with knee joints.

Inverse sampling keeps chest and arms together rather than pasting cut limbs.
Head patterns stay fixed during gait; boots retain their native pixel shapes.
"""
import json
from pathlib import Path
import numpy as np
from PIL import Image, ImageDraw, ImageFilter
from animate_knight_pixels import DIRS, RIG

ROOT=Path(__file__).resolve().parents[1]
SOURCE=ROOT/'tools/spritetool/assets/knights-pixel-v2'
KEEP=ROOT/'tools/spritetool/assets/knights-pixel-v5'
OUTPUT=ROOT/'tools/spritetool/assets/knights-pixel-v6'

def smooth(v):
    v=np.clip(v,0,1);return v*v*(3-2*v)

CENTERS={'N':24,'NE':24,'E':25,'SE':24,'S':23,'SW':24,'W':25,'NW':24}
HEADS={'N':(17,30,22),'NE':(14,30,23),'E':(14,32,24),'SE':(14,31,24),
       'S':(15,30,26),'SW':(19,33,25),'W':(20,38,25),'NW':(18,34,24)}
BOOTS={
 'N':[(18,31,24,38),(24,31,30,38)],
 'NE':[(17,30,23,36),(23,32,31,40)],
 'E':[(22,32,31,40)],
 'SE':[(16,32,24,39),(24,31,29,37)],
 'S':[(18,32,23,40),(24,32,30,40)],
 'SW':[(17,31,24,37),(24,33,30,40)],
 'W':[(22,33,31,41)],
 'NW':[(19,32,26,39),(26,30,34,36)],
}
ARM_WEIGHTS={}
for direction in DIRS:
    weight=np.zeros((48,48),dtype=float)
    for polygon in RIG[direction]['arms']:
        mask=Image.new('L',(48,48));ImageDraw.Draw(mask).polygon(polygon,fill=255)
        plane=np.array(mask.filter(ImageFilter.GaussianBlur(1.2)),dtype=float)/255
        side=0 if np.mean([p[0] for p in polygon])<CENTERS[direction] else 1
        weight+=plane*(1 if side==RIG[direction]['left'] else -1)
    ARM_WEIGHTS[direction]=weight

def weight_at(a,x,y):
    x=np.clip(x,0,47);y=np.clip(y,0,47)
    ix=np.floor(x).astype(int);iy=np.floor(y).astype(int)
    jx=np.minimum(ix+1,47);jy=np.minimum(iy+1,47);wx=x-ix;wy=y-iy
    return a[iy,ix]*(1-wx)*(1-wy)+a[iy,jx]*wx*(1-wy)+a[jy,ix]*(1-wx)*wy+a[jy,jx]*wx*wy

def field(x,y,d,action,phase):
    rig=RIG[d];center=CENTERS[d];fx,fy=rig['forward']
    if action=='walk':
        # Side weights fade continuously into the chest, so wrists never detach.
        arm=weight_at(ARM_WEIGHTS[d],x,y)
        legside=np.tanh((center-x)/1.7)*(1 if rig['left']==0 else -1)
        contact=phase in (0,2);sign=1 if phase==0 else -1
        leg=np.zeros_like(x)  # rigid boot templates and drawn knee joints below
        armfade=1-smooth((y-29)/5)
        lo,hi,bottom=HEADS[d]
        head=smooth((x-lo+2)/2)*smooth((hi+2-x)/2)*(1-smooth((y-bottom)/3))
        arm*=armfade*(1-head)
        if contact:
            dx=sign*legside*fx*leg-sign*arm*fx
            dy=sign*legside*fy*leg-sign*arm*fy
        else:
            dx=np.tanh((center-x)/2)*0.8*leg
            moving=(1+legside*(1 if phase==1 else -1))/2
            dy=-moving*leg-(0.7 if phase==3 else 0)*np.abs(arm)
        return dx,dy
    if action=='hit':
        amount=1 if phase==2 else .5
        angle=np.deg2rad((24 if fx>0 else -24 if fx<0 else 0)*amount)
        px,py=center,32
        # Image coordinates: positive visual lean left for east-facing characters.
        rx=np.cos(angle)*(x-px)+np.sin(angle)*(y-py)+px
        ry=-np.sin(angle)*(x-px)+np.cos(angle)*(y-py)+py
        dy0=(-1 if fy>0 else 1 if fy<0 else 0)*amount
        anchored=1-smooth((y-28)/7)
        dx=(rx-x)*anchored;dy=(ry-y+dy0)*anchored
        side=np.tanh((x-center)/5)
        spread=smooth((np.abs(x-center)-4)/5)*(1-smooth((y-27)/7))
        dx+=side*spread*amount
        dy-=spread*amount
        return dx,dy
    if action=='exhausted':
        amount=1 if phase in (1,3) else 0
        # Same complete seated drawing, a short chest breath tapering to ground.
        return np.zeros_like(x),-amount*(1-smooth((y-32)/6))
    return np.zeros_like(x),np.zeros_like(y)

def deform(im,d,action,phase):
    src=np.asarray(im);yy,xx=np.mgrid[:48,:48].astype(float)
    sx=xx.copy();sy=yy.copy()
    for _ in range(32):
        dx,dy=field(sx,sy,d,action,phase)
        # Damp the inverse solve at narrow ankle transition zones.
        sx=.4*sx+.6*(xx-dx);sy=.4*sy+.6*(yy-dy)
    ix=np.rint(sx).astype(int);iy=np.rint(sy).astype(int)
    valid=(ix>=0)&(ix<48)&(iy>=0)&(iy<48)
    out=np.zeros((48,48,4),dtype=np.uint8);out[valid]=src[iy[valid],ix[valid]]
    return Image.fromarray(out)

def gait(base,d,phase):
    body=base.copy();boxes=BOOTS[d]
    pieces=[]
    for side,box in enumerate(boxes):
        piece=Image.new('RGBA',(48,48));piece.paste(base.crop(box),box[:2])
        if d=='S' and side==1:
            # The shield overlaps the leg bounding box; it is not part of the
            # boot. Keep its bottom border attached to the continuously moved arm.
            piece.paste((0,0,0,0),(28,32,30,34))
        for y in range(box[1],box[3]):
            for x in range(box[0],box[2]):
                if piece.getpixel((x,y))[3]:body.putpixel((x,y),(0,0,0,0))
        pieces.append(piece)
    if d=='SW':
        # This boot's outline crosses the two template boxes; include its tip
        # in the moving near boot, never leave it at the standing position.
        pieces[1].putpixel((23,37),base.getpixel((23,37)))
        body.putpixel((23,37),(0,0,0,0))
    centers=[((b[0]+b[2]-1)//2,b[1]) for b in boxes]
    if len(pieces)==1:
        near=pieces[0];far=Image.new('RGBA',(48,48));far.alpha_composite(near,(-2,-1))
        pieces=[far,near];centers=[(centers[0][0]-2,centers[0][1]-1),centers[0]]
    out=Image.new('RGBA',(48,48));rig=RIG[d];fx,fy=rig['forward'];layers=[]
    palette=json.loads((SOURCE/'frames.json').read_text())['palette']
    colors=[tuple(bytes.fromhex(c[1:]))+(255,) for c in palette]
    for side,(piece,(cx,cy)) in enumerate(zip(pieces,centers)):
        sign=1 if side==rig['left'] else -1
        if phase in (0,2):
            sign*=1 if phase==0 else -1;dx=sign*fx;dy=sign*fy
        else:
            dx=(1 if side==0 else -1) if len(boxes)>1 else 0
            dy=-1 if side==(rig['left'] if phase==1 else 1-rig['left']) else 0
        layer=Image.new('RGBA',(48,48));draw=ImageDraw.Draw(layer)
        hip=(cx,29 if d not in ('N','S') else 30)
        knee=(cx+dx,cy+dy+2)
        draw.line([hip,knee],fill=colors[0],width=5)
        draw.line([(hip[0],hip[1]),knee],fill=colors[8],width=3)
        draw.line([(hip[0]-1,hip[1]),(knee[0]-1,knee[1])],fill=colors[9],width=1)
        layer.alpha_composite(piece,(dx,dy));layers.append((cy+dy,layer))
    for _,layer in sorted(layers,key=lambda v:v[0]):out.alpha_composite(layer)
    out.alpha_composite(deform(body,d,'walk',phase))
    return out

def main():
    OUTPUT.mkdir(parents=True,exist_ok=True)
    manifest=json.loads((KEEP/'frames.json').read_text())
    for d in DIRS:
        original=Image.open(SOURCE/f'{d}-pixel.png').convert('RGBA')
        result=Image.open(KEEP/f'{d}-pixel.png').convert('RGBA')
        standing=original.crop((0,0,48,48));seated=original.crop((0,192,48,240))
        for f in range(4):
            result.paste(gait(standing,d,f),(f*48,48))
            result.paste(standing if f<2 else deform(standing,d,'hit',f),(f*48,144))
            result.paste(deform(seated,d,'exhausted',f),(f*48,192))
        result.save(OUTPUT/f'{d}-pixel.png')
    manifest.update(version=6,source_recipe='tools/rebuild_knight_motion.py',
        motion_policy='Continuous upper-body motion with joint blending and rigid boot pixels connected by drawn knees. Idle and attack retained.')
    (OUTPUT/'frames.json').write_text(json.dumps(manifest,indent=2)+'\n')
    print('Rebuilt walk/hit/exhausted from complete standing/seated references.')

if __name__=='__main__':main()
