import numpy as np, math, copy, json
from pathlib import Path
from PIL import Image,ImageDraw
import base_model as m
D=Path(__file__).parent
# Reuse the same articulated soldier, without its sword/shield attachment block.
src=(D/'base_model.py').read_text(); body=src[src.index('def model(p):'):src.index(" hand=T(p['hands'][0]);")]+ '\n return meshes\n'
exec(body,m.__dict__)
V=m.V

def soldier(p):
 fs=m.model(p)
 # Round boot geometry in place of the two six-face boxes.
 # Identify their leather faces, replace with compact ellipsoids.
 m.meshes=[(v,c) for v,c in fs if c!='#754323']
 for foot in p['feet']:m.ell(V(foot)+[0,-.035,0],[.135,.20,.105],'#754323')
 return m.meshes

def bow(h,nock,lowered=False):
 axis=V([0,-.5,.866]) if lowered else V([0,0,1])
 pts=[h+V([0,-.18*math.sin(math.pi*t),0])+axis*.67*(1-2*t) for t in np.linspace(0,1,13)]
 for a,b in zip(pts,pts[1:]):m.rod(a,b,.025,'#976039')
 for end in [pts[0],pts[-1]]:m.rod(end,nock,.008,'#ede1bd')

def archer(row,f):
 p=copy.deepcopy(m.poses[row*4+f]);p['hands']=[[.05,-.22,1.18],[.12,-.62,.97]];p['elbows']=[[-.42,-.02,1.18],[.42,-.24,1.13]]
 if row==1:
  p['bob']=[-.025,.035,-.025,.035][f]
 if row==2:
  p['lean']=0;p['hands'][1]=[.20,-.88,1.55];p['elbows'][1]=[.34,-.42,1.5]
  p['hands'][0]=[[-.08,-.35,1.55],[-.12,.12,1.55],[-.2,.22,1.57],[-.35,-.02,1.2]][f]
  p['elbows'][0]=[-.58,.25,1.5]
 if row==4:
  p['hands']=[[-.4,-.25,1.1],[.35,-.55,1.0]]
 soldier(p)
 def T(q):
  x,y,z=q;z+=p['bob'];return V([x,y+p['lean']*(z-.45),z])
 h=T(p['hands'][1]);pull=T(p['hands'][0])
 # Bowstring is drawn back along the forward firing axis, no sword/shield.
 nock=pull if row!=2 or f<2 else h+V([0,.08,0])
 bow(h,nock,lowered=row!=2)
 if row!=2 or f<3:
  a=nock if row!=2 or f<2 else h+V([0,-.6,0]);direction=V([0,-.866,-.5]) if row!=2 else V([0,-1,0]);b=a+direction*1.05;m.rod(a,b,.014,'#b99553');m.poly([b+[0,-.13,0],b+[-.055,0,0],b+[.055,0,0]],'#bacbd6')
  for dx in [-.06,.06]:m.poly([a,a+[dx,-.1,0],a+[0,-.22,0]],'#ede1b8')
 # Quiver fixed to back.
 m.rod(T([-.21,.3,.93]),T([-.21,.3,1.53]),.12,'#704022')
 for x in [-.28,-.2,-.12]:m.rod(T([x,.3,1.3]),T([x,.3,1.82]),.015,'#d7bc7c')
 return m.meshes

def cavalry(row,f):
 p=copy.deepcopy(m.poses[row*4+f]);p['bob']=0;p['lean']=0
 p['feet']=[[-.49,.05,.03],[.49,.05,.03]];p['knees']=[[-.53,-.27,.45],[.53,-.27,.45]]
 p['hands'][1]=[.25,-.48,1.02];p['elbows'][1]=[.43,-.15,1.18]
 if row==3:p['lean']=[0,0,.24,.16][f]
 if row==2:
  p['hands'][0]=[[-.48,.12,1.17],[-.48,-.48,1.15],[-.48,-.98,1.13],[-.48,-.38,1.15]][f];p['elbows'][0]=[-.58,-.1,1.2];p['lean']=[.06,-.1,-.24,-.06][f]
 if row==1:p['lean']=-.18
 if row==4:
  p['lean']=[-.52,-.60,-.54,-.62][f];p['bob']=[0,-.035,0,-.035][f];p['hands'][0]=[-.48,-.38,.85]
 fs=soldier(p);bounce=[.08,0,.14,.03][f] if row==1 else 0
 off=V([0,.12,1.2+bounce]);m.meshes=[(v*.8+off,c) for v,c in fs]
 hand=V(p['hands'][0]);hand[1]+=p['lean']*(hand[2]-.45);hand=hand*.8+off
 dv=V([0,-1,0]) if row==2 else V([0,-.35,.94]);dv/=np.linalg.norm(dv)
 m.rod(hand-dv*.5,hand+dv*1.45,.025,'#966131');tip=hand+dv*1.68;base=hand+dv*1.42
 side=V([.07,0,0]);m.poly([base-side,tip,base+side,base-dv*.06],'#d6e7ec')
 # Horse: four jointed legs, barrel, haunches, neck, muzzle, ears, tack.
 drop=0
 def H(q):
  v=V(q);v[2]+=bounce
  if row==4 and v[1]<-.7:
   v[2]-=.25;v[1]-=.08
  return v
 chest='#825032';dark='#352820'
 m.ell(H([0,0,1.13]),[.43,.79,.44],chest);m.ell(H([0,.53,1.12]),[.44,.4,.42],chest)
 m.rod(H([0,-.56,1.2]),H([0,-.9,1.83]),.24,chest)
 m.ell(H([0,-1.04,1.86]),[.21,.35,.25],chest);m.ell(H([0,-1.32,1.69]),[.20,.27,.17],'#9a6845')
 for x in [-.13,.13]:m.ell(H([x,-.93,2.12]),[.055,.1,.17],chest);m.ell(H([x*1.5,-1.13,1.92]),[.032,.034,.032],'#15191b')
 for y,z in [(-.64,1.53),(-.76,1.73),(-.89,1.98)]:m.ell(H([0,y+.13,z]),[.12,.15,.2],dark)
 for i,(x,y) in enumerate([(-.3,-.5),(.3,-.5),(-.3,.55),(.3,.55)]):
  phase=f*math.pi/2+(0 if i<2 else math.pi)+(.28 if i%2 else 0);stride=.43*math.cos(phase) if row==1 else 0;lift=(.12+max(0,math.sin(phase))*.28 if f==2 else max(0,math.sin(phase))*.24) if row==1 else 0
  hip=H([x,y,1.12]);k=H([x,y+stride*.5,.58]);ft=V([x,y+stride,.11+lift]);
  if row==1:k[1]-=.15*math.sin(phase);k[2]+=.08
  m.rod(hip,k,.10,chest);m.rod(k,ft,.065,chest);m.ell(ft,[.1,.14,.10],dark)
 m.rod(H([0,.73,1.3]),H([0,1.02,.60]),.09,dark)
 m.ell(H([0,.06,1.52]),[.47,.38,.07],'#a91e2c')
 for x in [-.22,.22]:m.rod(H([x,-1.27,1.74]),V([x,-.25,1.98]),.017,'#38261d')
 if drop:
  # Lower rider with horse, maintaining saddle contact.
  m.meshes=[(v-[0,0,drop] if j<len(fs) else v,c) for j,(v,c) in enumerate(m.meshes)]
 return m.meshes

def render(fs,scale):
 az=-math.pi/4;el=.48;t=V([math.cos(az)*math.cos(el),math.sin(az)*math.cos(el),math.sin(el)]);r=V([-math.sin(az),math.cos(az),0]);u=np.cross(t,r)
 im=Image.new('RGBA',(320,320));draw=ImageDraw.Draw(im)
 for v,c in sorted(fs,key=lambda f:np.dot(f[0].mean(axis=0),t)):
  n=np.cross(v[1]-v[0],v[2]-v[0]);ln=np.linalg.norm(n)
  if ln<1e-9:continue
  light=.65+.35*abs(np.dot(n/ln,[-.4,-.5,.76]));rgb=tuple(round(int(c[i:i+2],16)*light) for i in [1,3,5])
  draw.polygon([(160+np.dot(q,r)*scale,265-np.dot(q,u)*scale) for q in v],fill=rgb)
 return im
for name,fn,scale in [('archer',archer,88),('cavalry',cavalry,60)]:
 sheet=Image.new('RGBA',(1280,1600));data=[]
 for row in range(5):
  for f in range(4):
   fs=fn(row,f)
   if row==3 and f>=2:
    strength=1 if f==2 else .45
    revised=[]
    for v,c in fs:
     v=v.copy()
     if name=='cavalry':
      pivot=V([0,.55,.12]);a=-.65*strength;rot=V([[1,0,0],[0,math.cos(a),-math.sin(a)],[0,math.sin(a),math.cos(a)]])
      v=(v-pivot)@rot.T+pivot;v[:,1]+=.35*strength
     else:
      v[:,1]+=.45*strength+np.maximum(v[:,2]-.75,0)*.3*strength
     revised.append((v,c))
    fs=revised
   sheet.alpha_composite(render(fs,scale),(f*320,row*320));data.append([[v.round(4).tolist(),c] for v,c in fs])
 sheet.save(D/f'{name}-SW-3d.png');(D/f'{name}-poses.json').write_text(json.dumps(data,separators=(',',':')))
 print(name,'20 poses rendered')
