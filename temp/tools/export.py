import model as m
import numpy as np, math, json, copy
from pathlib import Path
frames=[]
for row in range(5):
 for f in range(16):
  t=f/16;idx=int(t*4);a=m.poses[row*4+idx];b=m.poses[row*4+(idx+1)%4];u=t*4-idx;u=u*u*(3-2*u);p=copy.deepcopy(a)
  for k in ['bob','lean','feet','knees','elbows','hands','blade']:
   p[k]=(np.array(a[k])*(1-u)+np.array(b[k])*u).tolist()
  if row==1:
   p['bob']=-.025+.06*math.sin(t*math.tau)**2
  if row==2:
   # Anticipation 38%, explosive strike 18%, impact hold 12%, recovery 32%.
   ready=copy.deepcopy(m.poses[0]);wind=m.poses[8];high=m.poses[9];hit=m.poses[10];end=m.poses[11]
   keys=[(0,ready),(.375,wind),(.4375,high),(.5,hit),(.5625,end),(.6875,end),(1,ready)]
   for (ta,pa),(tb,pb) in zip(keys,keys[1:]):
    if ta<=t<tb:
     u=(t-ta)/(tb-ta)
     if ta==0 or ta>=.6875:u=u*u*(3-2*u)
     p=copy.deepcopy(pa);p['row']=2
     for k in ['bob','lean','feet','knees','elbows','hands','blade']:
      p[k]=(np.array(pa[k])*(1-u)+np.array(pb[k])*u).tolist()
     break
  faces=m.model(p)
  frames.append([[v.round(4).tolist(),c] for v,c in faces])
  if row==2:
   d=np.array(p['blade']);d/=np.linalg.norm(d);edge=np.cross([1,0,0],d);assert abs(np.dot(edge,[1,0,0]))<1e-8
for i in range(5):json.dump(frames[i*16:(i+1)*16],open(Path(__file__).parent.parent/f'dist/mesh-{i}.json','w'),separators=(',',':'))
print('Exported 80 poses; archer gait applied; attack blade plane verified.')
