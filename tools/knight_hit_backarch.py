#!/usr/bin/env python3
"""Apply selected backward-arch reaction using retained native pixel parts."""
import json
from pathlib import Path
from PIL import Image
from animate_knight_pixels import RIG, DIRS, blank, region, shift, split_mask

ROOT=Path(__file__).resolve().parents[1]
SOURCE=ROOT/'tools/spritetool/assets/knights-pixel-v3'
OUTPUT=ROOT/'tools/spritetool/assets/knights-pixel-v4'

def pose(base,rig,strength):
    hip=rig['leg'];fx,fy=rig['forward']
    body,arms=split_mask(base,rig['arms'])
    angle=(24 if fx>0 else -24 if fx<0 else 0)*strength
    dy=round((-2 if fy>0 else 1 if fy<0 else 0)*strength)
    # Keep every lower-body pixel at its original location.
    feet=region(base,(0,hip,48,48))
    upper=region(body,(0,0,48,hip))
    upper=upper.rotate(angle,Image.Resampling.NEAREST,center=(24,hip))
    upper=shift(upper,0,dy)
    out=blank()
    # Original pelvis forms the articulation overlap, not a stretched torso.
    out.alpha_composite(region(body,(0,hip-3,48,hip)))
    out.alpha_composite(upper)
    for i,arm in enumerate(arms):
        arm=region(arm,(0,0,48,hip))
        # Open both arms as a passive reaction; no slash or attack effect.
        side=-1 if (arm.getbbox() or (24,))[0]<22 else 1
        moved=arm.rotate(angle+side*8*strength,Image.Resampling.NEAREST,center=(24,hip))
        out.alpha_composite(shift(moved,round(side*strength),dy-round(strength)))
    # A planted stance is an invariant, including the original boot silhouettes.
    out.paste(feet.crop((0,hip,48,48)),(0,hip))
    return out

def main():
    OUTPUT.mkdir(parents=True,exist_ok=True)
    manifest=json.loads((SOURCE/'frames.json').read_text())
    manifest.update(version=4,source_recipe='tools/knight_hit_backarch.py',
        hit_policy='Selected candidate 1: backward arch, planted boots, rigid upper-body rotation, arms open; neutral/neutral/impact/recovery.')
    for d in DIRS:
        im=Image.open(SOURCE/f'{d}-pixel.png').convert('RGBA')
        base=im.crop((0,0,48,48));frames=[base,base,pose(base,RIG[d],1),pose(base,RIG[d],0.5)]
        for f,frame in enumerate(frames): im.paste(frame,(f*48,144))
        im.save(OUTPUT/f'{d}-pixel.png')
    (OUTPUT/'frames.json').write_text(json.dumps(manifest,indent=2)+'\n')
    print('Wrote 8 backward-arch hit animations; retained all other rows.')

if __name__=='__main__':main()
