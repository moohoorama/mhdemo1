#!/usr/bin/env python3
"""Animate retained native pixel clusters; no scaling, filtering or new palette.

Project-specific rig: integer translations preserve helmet/eye/armor pixels.
Input v2 is immutable; output v3 can be rebuilt without image generation.
"""
import json
from pathlib import Path
import numpy as np
from PIL import Image, ImageDraw

ROOT = Path(__file__).resolve().parents[1]
SOURCE = ROOT / 'tools/spritetool/assets/knights-pixel-v2'
OUTPUT = ROOT / 'tools/spritetool/assets/knights-pixel-v3'
DIRS = ['N', 'NE', 'E', 'SE', 'S', 'SW', 'W', 'NW']
# Arm polygons include held equipment. Leg split is in screen coordinates.
RIG = {
 'N': dict(arms=[[(0,20),(18,20),(18,29),(0,29)],[(30,0),(47,0),(47,29),(29,29),(29,23),(30,23)]], leg=31, split=24, head=22, forward=(0,-2), left=0),
 'NE':dict(arms=[[(0,21),(18,21),(18,29),(0,29)],[(30,13),(47,13),(47,31),(29,31),(27,26),(30,23)]],leg=32,split=23,head=22,forward=(2,-1),left=0),
 'E': dict(arms=[[(30,20),(47,20),(47,33),(28,33),(27,28)]],leg=33,split=24,head=23,forward=(2,0),left=0,side=True),
 'SE':dict(arms=[[(0,24),(21,24),(23,27),(22,32),(0,32)],[(29,22),(47,22),(47,32),(29,32)]],leg=31,split=25,head=23,forward=(2,1),left=1),
 'S': dict(arms=[[(0,20),(18,20),(19,25),(19,32),(0,32)],[(28,24),(47,24),(47,33),(28,33)]],leg=33,split=23,head=25,forward=(0,2),left=1),
 'SW':dict(arms=[[(0,16),(19,16),(19,23),(21,25),(21,28),(0,28)],[(24,25),(31,25),(31,33),(24,33)]],leg=33,split=24,head=24,forward=(-2,1),left=1),
 'W': dict(arms=[[(0,10),(20,10),(20,20),(23,23),(23,34),(0,34)]],leg=34,split=24,head=24,forward=(-2,0),left=1,side=True),
 'NW':dict(arms=[[(0,19),(19,19),(22,24),(21,30),(0,30)]],leg=31,split=26,head=23,forward=(-2,-1),left=0,arm_side=0),
}

def blank(): return Image.new('RGBA',(48,48))
def shift(im, dx=0, dy=0):
 out=blank();out.alpha_composite(im,(int(dx),int(dy)));return out
def region(im, box):
 out=blank();out.paste(im.crop(box),box[:2]);return out
def split_mask(im, polygons):
 rest=im.copy();layers=[]
 for polygon in polygons:
  mask=Image.new('L',im.size);ImageDraw.Draw(mask).polygon(polygon,fill=255)
  layer=blank();layer.paste(rest,(0,0),mask);rest.paste((0,0,0,0),(0,0,48,48),mask);layers.append(layer)
 return rest,layers
def breathing(base, cut, lift):
 out=region(base,(0,cut,48,48))
 # Keep one seam row to avoid holes between lifted chest and stationary legs.
 if lift: out.alpha_composite(region(base,(0,cut-1,48,cut)))
 out.alpha_composite(shift(region(base,(0,0,48,cut)),0,-lift))
 return out

def walk(base, rig):
 body,arms=split_mask(base,rig['arms'])
 torso=region(body,(0,0,48,rig['leg']))
 legs=[region(body,(0,rig['leg'],rig['split'],48)),region(body,(rig['split'],rig['leg'],48,48))]
 if rig.get('side'):
  # Profile hides the far leg in the standing pose: reuse the same boot pixels.
  near=region(base,(0,rig['leg'],48,48));far=shift(near,-rig['forward'][0],-1)
  legs=[far,near] if rig['left']==0 else [near,far]
 output=[];fx,fy=rig['forward'];left=rig['left']
 for phase in range(4):
  out=blank();contact=phase in (0,2);sign=1 if phase==0 else -1
  for side,leg in enumerate(legs):
   anatomical=1 if side==left else -1
   if contact: dx,dy=sign*anatomical*fx,sign*anatomical*fy
   else:
    dx=0 if rig.get('side') else (1 if side==0 else -1)
    # Alternating passing feet: four distinct poses, including the two closes.
    dy=-1 if side==(left if phase==1 else 1-left) else 0
   # Connect the retained boot/thigh to its hip using its existing top rows.
   # This avoids detached feet when a stride moves a narrow diagonal leg.
   bounds=leg.getbbox()
   if bounds:
    joint=region(leg,(0,bounds[1],48,min(bounds[1]+3,48)))
    steps=max(abs(dx),abs(dy),1)
    for step in range(steps+1):
     out.alpha_composite(shift(joint,round(dx*step/steps),round(dy*step/steps)))
   out.alpha_composite(shift(leg,dx,dy))
  out.alpha_composite(torso)
  for side,arm in enumerate(arms):
   anatomical=1 if side==left else -1
   if len(arms)==1: anatomical=1 if rig.get('arm_side',1-left)==left else -1
   dx=-sign*anatomical*fx if contact else 0
   dy=-sign*anatomical*fy if contact else (-1 if phase==3 else 0)
   out.alpha_composite(shift(arm,dx,dy))
  # Exact same helmet pixels, protected against arm occlusion and pose drift.
  head=region(base,(19 if rig['forward'][0]<0 else 18,0,30 if rig['forward'][0]<0 else 29,rig['head']))
  out.alpha_composite(head)
  output.append(out)
 return output

def recoil(base,rig,strength):
 out=blank();fx,fy=rig['forward'];head=rig['head'];hip=rig['leg']
 # A rigid head and progressively displaced torso give a sharp backward lean.
 dx=-int(np.sign(fx))*strength
 dy=(-strength if fy>0 else strength//2 if fy<0 else -2)
 out.alpha_composite(region(base,(0,hip,48,48)))
 for y in range(hip-1,head-1,-1):
  amount=(hip-y)/(hip-head)
  offset=round(dy*amount)
  next_y=y+1+round(dy*(hip-y-1)/(hip-head))
  # Cover the scanline interval; forward scattering alone leaves empty stripes.
  for target_y in range(y+offset,max(y+offset+1,next_y)):
   out.alpha_composite(shift(region(base,(0,y,48,y+1)),round(dx*amount),target_y-y))
 out.alpha_composite(shift(region(base,(0,0,48,head)),dx,dy))
 return out

def main():
 OUTPUT.mkdir(parents=True,exist_ok=True)
 manifest=json.loads((SOURCE/'frames.json').read_text())
 manifest.update(version=3,idle_policy='one-pixel chest breathing; feet anchored',
  source_recipe='tools/animate_knight_pixels.py',
  frame_durations_ms={'idle':[400,400,400,400],'walk':[180]*4,'attack':[250]*4,'hit':[220,220,150,280],'exhausted':[180,220,180,220]},
  motion_policy='Reuse native pixel clusters with integer translation. Attacks unchanged. Four walking poses with opposing arms. Hit frames 1/2 identical; frame 3 recoil.')
 for direction in DIRS:
  sheet=Image.open(SOURCE/f'{direction}-pixel.png').convert('RGBA');rig=RIG[direction]
  idle=sheet.crop((0,0,48,48));exhausted=sheet.crop((0,192,48,240))
  rows=[[breathing(idle,rig['leg'],lift) for lift in [0,1,1,0]],walk(idle,rig),
        [sheet.crop((f*48,96,f*48+48,144)) for f in range(4)],
        [idle.copy(),idle.copy(),recoil(idle,rig,4),recoil(idle,rig,2)],
        [breathing(exhausted,exhausted.getbbox()[3]-3,lift) for lift in [0,1,0,1]]]
  result=Image.new('RGBA',(192,240))
  for r,frames in enumerate(rows):
   for f,im in enumerate(frames):result.paste(im,(f*48,r*48))
  result.save(OUTPUT/f'{direction}-pixel.png')
 for frame in manifest['frames']:
  # Previous redraw landmarks refer to v2; avoid misrepresenting them as v3.
  for key in list(frame):
   if key not in ('direction','animation','frame','index','crop','pivot'):del frame[key]
 (OUTPUT/'frames.json').write_text(json.dumps(manifest,indent=2)+'\n')
 (OUTPUT/'rig.json').write_text(json.dumps(RIG,indent=2)+'\n')
 print('Generated 8 native motion sheets with retained attack pixels.')

if __name__=='__main__':main()
