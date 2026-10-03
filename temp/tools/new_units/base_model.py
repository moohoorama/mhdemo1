import numpy as np, math, json
from PIL import Image, ImageDraw
from pathlib import Path
D=Path(__file__).parent
V=lambda a:np.array(a,dtype=float)
meshes=[]
def poly(v,c): meshes.append((np.array(v),c))
def ell(p,s,c):
 p=V(p); s=V(s)
 for j in range(8):
  for i in range(12):
   pts=[]
   for a,b in [(j,i),(j,i+1),(j+1,i+1),(j+1,i)]:
    u=math.pi*a/8; t=2*math.pi*b/12
    pts.append(p+s*V([math.sin(u)*math.cos(t),math.sin(u)*math.sin(t),math.cos(u)]))
   poly(pts,c)
def rod(a,b,r,c):
 a=V(a); b=V(b); n=b-a; n/=np.linalg.norm(n); u=np.cross(n,[0,0,1])
 if np.linalg.norm(u)<.01:u=np.cross(n,[0,1,0])
 u/=np.linalg.norm(u); v=np.cross(n,u)
 for i in range(10):
  q=lambda t:r*(u*math.cos(t)+v*math.sin(t))
  t=i*math.tau/10; t2=(i+1)*math.tau/10
  poly([a+q(t),a+q(t2),b+q(t2),b+q(t)],c)
 ell(a,[r]*3,c);ell(b,[r]*3,c)
def box(p,s,c):
 p=V(p); s=V(s)/2
 pts=[p+s*V([x,y,z]) for x in [-1,1] for y in [-1,1] for z in [-1,1]]
 for ids in [[0,1,3,2],[4,6,7,5],[0,4,5,1],[2,3,7,6],[0,2,6,4],[1,5,7,3]]:poly([pts[i] for i in ids],c)
poses=[]
for row in range(5):
 for f in range(4):
  p=dict(row=row,frame=f,bob=0,lean=0,feet=[[-.25,0,.13],[.25,0,.13]],knees=[[-.25,-.04,.46],[.25,-.04,.46]],elbows=[[-.56,-.02,1.03],[.55,-.04,1.04]],hands=[[-.62,-.35,1.08],[.54,-.37,1.08]],blade=[-.05,-.32,.94],seated=False)
  if row==0:p['bob']=[0,.025,0,-.025][f]
  if row==1:
   stride=[-.3,0,.3,0][f];p['feet']=[[-.25,stride,.13+(.1 if f==1 else 0)],[.25,-stride,.13+(.1 if f==3 else 0)]];p['knees']=[[-.25,stride*.6,.5],[.25,-stride*.6,.5]];p['bob']=[0,-.04,0,-.04][f]
  if row==2:
   p['feet']=[[-.34,.23,.13],[.34,-.22,.13]]
   p['elbows'][0]=[[-.62,.08,1.65],[-.58,-.35,1.55],[-.52,-.58,1.22],[-.38,-.5,.95]][f]
   p['hands'][0]=[[-.52,.03,2.03],[-.5,-.7,1.73],[-.42,-.92,1.15],[-.27,-.75,.78]][f]
   p['blade']=[[0,.72,.65],[0,-.78,.63],[0,-.98,-.16],[.05,-.82,-.57]][f]
   p['elbows'][1]=[.49,-.22,1.13];p['hands'][1]=[.42,-.58,1.23];p['lean']=[-.02,-.08,-.15,-.13][f]
  if row==3 and f>=2:
   p['lean']=[.32,.24][f-2];p['hands'][0]=[-.6,-.1,.77];p['blade']=[-.1,-.7,.55];p['feet'][1]=[.29,.18,.13]
  if row==4:
   p['seated']=True;p['bob']=-.57+[0,-.03,.01,-.04][f];p['feet']=[[-.3,-.65,.15],[.3,-.65,.15]];p['knees']=[[-.3,-.38,.27],[.3,-.38,.27]];p['hands']=[[-.62,-.4,.78],[.56,-.32,.85]];p['blade']=[-.12,-.98,-.12]
  poses.append(p)

def model(p):
 global meshes; meshes=[]
 blue='#28486e';steel='#636971';red='#b91828';brown='#754323';skin='#efb15c'
 def T(q):
  x,y,z=q; z+=p['bob'];return V([x,y+p['lean']*(z-.45),z])
 for i in range(2):
  hip=T([(-.24 if i==0 else .24),0,.77]);k=V(p['knees'][i]);foot=V(p['feet'][i]);rod(hip,k,.16,blue);rod(k,foot,.13,steel);box(foot+[0,-.08,0],[.29,.42,.22],brown)
 ell(T([0,0,.92]),[.43,.27,.31],blue);ell(T([0,0,1.26]),[.4,.27,.38],blue)
 ell(T([0,-.035,1.3]),[.36,.275,.27],steel)
 box(T([0,0,1.01]),[.79,.56,.105],red);box(T([0,-.29,.8]),[.13,.05,.36],red)
 rod(T([0,0,1.48]),T([0,0,1.65]),.14,skin)
 ell(T([0,0,1.76]),[.29,.26,.32],skin)
 ell(T([0,.025,1.9]),[.37,.32,.32],steel)
 # back and side helmet guards, brow and central ridge
 box(T([0,.25,1.69]),[.59,.13,.34],steel)
 for x in [-.29,.29]:box(T([x,.035,1.68]),[.13,.4,.32],steel)
 box(T([0,-.29,1.81]),[.64,.09,.1],'#30343a')
 for x in [-.1,.1]:box(T([x,-.263,1.72]),[.05,.015,.055],'#211f22')
 for y,z in [(-.2,2.12),(0,2.22),(.2,2.13),(.36,2.01),(.48,1.99)]:ell(T([0,y,z]),[.09,.16,.14],red)
 for i in range(2):
  sh=T([(-.4 if i==0 else .4),0,1.42]);el=T(p['elbows'][i]);ha=T(p['hands'][i]);ell(sh,[.19,.22,.19],steel);rod(sh,el,.14,blue);rod(el,ha,.12,blue);ell(ha,[.125]*3,skin)
 hand=T(p['hands'][0]);direction=V(p['blade']);direction/=np.linalg.norm(direction)
 rod(hand-direction*.12,hand+direction*.1,.045,brown)
 side=np.cross(V([1,0,0]),direction) if p["row"]==2 else V([1,0,0]);side-=direction*np.dot(side,direction);side/=np.linalg.norm(side)
 base=hand+direction*.15;rod(base-side*.18,base+side*.18,.045,'#d5a123')
 tip=base+direction*.85;mid=base+direction*.55
 poly([base-side*.07,mid-side*.08,tip,mid+side*.13,base+side*.07],'#bad9f1')
 poly([base,mid,tip,mid+side*.13,base+side*.07],'#e5f4ff')
 # Shield normal always forward, attached to anatomical LEFT hand
 h=T(p['hands'][1]);center=h+V([0,-.09,0]);
 ell(center,[.3,.07,.39],brown);ell(center+V([0,-.07,0]),[.09,.055,.1],steel)
 return meshes

def render(p,az):
 cam=V([ -6, -6 if az=='SE' else 6,4.4]);toward=cam/np.linalg.norm(cam);right=np.cross([0,0,1],toward);right/=np.linalg.norm(right);up=np.cross(toward,right)
 im=Image.new('RGBA',(320,320),(0,0,0,0));dr=ImageDraw.Draw(im)
 faces=[]
 for verts,c in model(p):
  n=np.cross(verts[1]-verts[0],verts[2]-verts[0]);ln=np.linalg.norm(n)
  if ln<1e-8:continue
  n/=ln;light=.65+.35*abs(np.dot(n,V([-.4,-.5,.76])))
  rgb=tuple(int(int(c[i:i+2],16)*light) for i in [1,3,5])
  pts=[(160+np.dot(v,right)*105,255-np.dot(v,up)*105) for v in verts]
  faces.append((np.dot(verts.mean(axis=0),toward),pts,rgb))
 for _,pts,rgb in sorted(faces,key=lambda x:x[0]):dr.polygon(pts,fill=rgb)
 return im
