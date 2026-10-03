from PIL import Image,ImageDraw
from pathlib import Path
import numpy as np,json,math
ROOT=Path(__file__).resolve().parents[1];OUT=ROOT/'tools/spritetool/assets/knights-pixel-v2';OUT.mkdir(exist_ok=True)
D=['N','NE','E','SE','S','SW','W','NW']
# Helmet body, excluding plume and shoulders, measured on the first source frame.
BOX={'N':(99,74,180,149),'NE':(98,75,179,141),'E':(98,54,176,124),'SE':(109,79,198,151),'S':(109,77,198,155),'SW':(98,75,180,151),'W':(111,91,181,153),'NW':(109,70,188,143)}
manifest={'directions':D,'animations':['idle','walk','attack','hit','exhausted'],'cell':[40,48],'pivot':[20,38],'target_head_geometric_size':10,'reference_scale':4,'frames':[]}
for d in D:
 src=Image.open(ROOT/'tools/spritetool/assets'/f'{d}-transparent.png').convert('RGBA');w,h=src.size
 b=BOX[d];template=src.crop(b)
 ref=Image.new('RGBA',(640,960));first=None;row_matches=[]
 for row in range(5):
  matches=[]
  for col in range(4):
   crop=src.crop((round(col*w/4),round(row*h/5),round(col*w/4)+280,round(row*h/5)+280))
   small=np.asarray(crop.resize((70,70),Image.Resampling.BOX)).astype(np.float32)/255
   best=None
   for scale in [.88,.94,1,1.06,1.12]:
    tw=max(4,round(template.width*scale/4));th=max(4,round(template.height*scale/4))
    t=np.asarray(template.resize((tw,th),Image.Resampling.BOX)).astype(np.float32)/255
    # Match color and alpha within upper body; image view dimensions y,x,channel,h,w.
    win=np.lib.stride_tricks.sliding_window_view(small[5:55,12:58],(th,tw),axis=(0,1)).transpose(0,1,3,4,2)
    weight=(t[:,:,3]>.7).astype(np.float32)
    score=(((win[:,:,:,:,:3]-t[None,None,:,:,:3])**2).sum(axis=-1)*weight).sum(axis=(-1,-2))/max(1,weight.sum())
    score+=((win[:,:,:,:,3]-t[None,None,:,:,3])**2).mean(axis=(-1,-2))*.3
    iy,ix=np.unravel_index(score.argmin(),score.shape);cost=float(score[iy,ix])
    if best is None or cost<best[0]:best=(cost,(ix+12)*4,(iy+5)*4,tw*4,th*4)
   cost,x,y,bw,bh=best;cx=x+bw/2;cy=y+bh/2
   factor=40/math.sqrt(bw*bh)
   matches.append((crop,cx,cy,factor,best))
  mx=float(np.median([a[1] for a in matches]));my=float(np.median([a[2] for a in matches]))
  for col,(crop,cx,cy,factor,best) in enumerate(matches):
   # Idle center fixed in both axes. Other poses retain source motion about the row median.
   dx=0 if row==0 else (cx-mx)*factor
   dy=0 if row==0 else (cy-my)*factor
   targetx=80+dx;targety=(104 if row==4 else 72)+dy
   resized=crop.resize((round(280*factor),)*2,Image.Resampling.LANCZOS)
   xoff=round(targetx-cx*factor);yoff=round(targety-cy*factor)
   cell=Image.new('RGBA',(160,192));cell.alpha_composite(resized,(xoff,yoff));ref.alpha_composite(cell,(col*160,row*192))
   manifest['frames'].append({'direction':d,'row':row,'frame':col,'helmet_box':[int(v) for v in best[1:]],'match_error':best[0],'scale':factor/4,'head_center':[targetx/4,targety/4]})
 ref.save(OUT/f'{d}-normalized-reference.png')
 print(d,'match max',round(max(f['match_error'] for f in manifest['frames'] if f['direction']==d),3),flush=True)
(OUT/'normalization.json').write_text(json.dumps(manifest,indent=2)+'\n')
