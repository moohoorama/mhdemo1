"""Project unchanged 3D mesh frames with quantized shading, outlines, and world-space trails."""
import json,math
from pathlib import Path
import numpy as np
from PIL import Image,ImageDraw,ImageFilter
ROOT=Path(__file__).resolve().parent.parent
raw=[json.loads((ROOT/f'dist/mesh-{i}.json').read_text()) for i in range(5)]
trails={}
for frame in (7,8,9):
 strips=[]
 for j in range(max(6,frame-2),frame):
  a=np.array(next(v for v,c in raw[2][j] if c=='#bad9f1'));b=np.array(next(v for v,c in raw[2][j+1] if c=='#bad9f1'))
  # Blade tip and inner blade, exactly in the moving blade plane.
  basea=(a[0]+a[-1])/2;baseb=(b[0]+b[-1])/2
  strips.append([a[2].tolist(),b[2].tolist(),(baseb+(b[2]-baseb)*.3).tolist(),(basea+(a[2]-basea)*.3).tolist()])
 trails[str(frame)]=strips
(ROOT/'dist/trails.json').write_text(json.dumps(trails))
prepared=[]
for row in raw:
 out=[]
 for fs in row:
  faces=[]
  for vv,c in fs:
   v=np.array(vv);n=np.cross(v[1]-v[0],v[2]-v[0]);ln=np.linalg.norm(n)
   if ln<1e-8:continue
   light=.45+.55*abs(np.dot(n/ln,[-.4,-.5,.76]));light=round(light*4)/4
   base={'#636971':'#666b75','#28486e':'#315c8c','#b91828':'#ec172d','#754323':'#9e582c'}.get(c,c)
   rgb=tuple(min(255,round(int(base[i:i+2],16)*light)) for i in (1,3,5))
   faces.append((v,v.mean(axis=0),rgb,False))
  out.append(faces)
 prepared.append(out)
angles={'N':math.pi/2,'NE':math.pi*.75,'E':math.pi,'SE':-math.pi*.75,'S':-math.pi/2,'SW':-math.pi/4,'W':0,'NW':math.pi/4}
for name,az in angles.items():
 t=np.array([math.cos(az)*math.cos(.48),math.sin(az)*math.cos(.48),math.sin(.48)]);r=np.array([-math.sin(az),math.cos(az),0]);u=np.cross(t,r)
 sheet=Image.new('RGBA',(128*16,128*5))
 for row in range(5):
  for f in range(16):
   im=Image.new('RGBA',(128,128));draw=ImageDraw.Draw(im)
   faces=list(prepared[row][f])
   if row==2:
    for strip in trails.get(str(f),[]):
     v=np.array(strip);faces.append((v,v.mean(axis=0),(147,188,248,125),True))
   for v,center,c,is_trail in sorted(faces,key=lambda x:np.dot(x[1],t)):
    pts=[(64+np.dot(p-[0,0,1],r)*34.6,65.28-np.dot(p-[0,0,1],u)*34.6) for p in v]
    if is_trail:
     layer=Image.new('RGBA',im.size);ImageDraw.Draw(layer).polygon(pts,fill=c);im.alpha_composite(layer);draw=ImageDraw.Draw(im)
    else:draw.polygon(pts,fill=c)
   a=im.getchannel('A');border=a.point(lambda x:255 if x>180 else 0).filter(ImageFilter.MaxFilter(3));bg=Image.new('RGBA',im.size,(13,17,25,0));bg.putalpha(border);bg.alpha_composite(im)
   sheet.alpha_composite(bg,(f*128,row*128))
 sheet.save(ROOT/f'dist/sprites/{name}.png')
 print(name,flush=True)
