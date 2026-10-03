#!/usr/bin/env python3
"""OpenCV material segmentation, masked template registration and local repair.

Analyze every frame. Repair rigid equipment only after a constrained source
match; never close all alpha contours (leg gaps and slash arcs are intentional).
"""
import json
from pathlib import Path
import cv2
import numpy as np
from rebuild_knight_motion import field, HEADS, CENTERS

ROOT=Path(__file__).resolve().parents[1]
SOURCE=ROOT/'tools/spritetool/assets/knights-pixel-v6'
REFERENCE=ROOT/'tools/spritetool/assets/knights-pixel-v2'
OUTPUT=ROOT/'tools/spritetool/assets/knights-pixel-v7'
DIRS=['N','NE','E','SE','S','SW','W','NW']
ACTS=['idle','walk','attack','hit','exhausted']
SHIELDS={'N':(12,20,19,30),'NE':(14,20,19,29),'E':(29,18,34,27),
 'SE':(28,21,36,32),'S':(27,24,35,34),'SW':(23,24,31,33),
 'W':(14,24,23,35),'NW':(11,19,22,31)}
SWORDS={'N':(29,14,34,24),'NE':(30,14,38,25),'E':(31,20,36,28),
 'SE':(18,24,25,30),'S':(13,20,18,29),'SW':(12,16,19,24),'W':(10,10,20,22)}

def material_template(frame,box,part):
    x,y,X,Y=box;roi=np.zeros((48,48),np.uint8);roi[y:Y,x:X]=255
    hsv=cv2.cvtColor(frame[:,:,:3],cv2.COLOR_BGR2HSV)
    if part=='shield':
        seed=cv2.inRange(hsv,np.array([0,70,35]),np.array([28,255,240]))
    else:
        seed=cv2.inRange(hsv,np.array([0,0,190]),np.array([179,140,255]))
    seed=cv2.bitwise_and(seed,roi);seed[frame[:,:,3]==0]=0
    n,labels,stats,_=cv2.connectedComponentsWithStats(seed,connectivity=8)
    if n<2:return None
    best=1+int(np.argmax(stats[1:,cv2.CC_STAT_AREA]));mask=(labels==best).astype(np.uint8)*255
    contours,_=cv2.findContours(mask,cv2.RETR_EXTERNAL,cv2.CHAIN_APPROX_SIMPLE)
    # Shield bosses are gray; include the brown contour's interior as one object.
    if part=='shield':cv2.drawContours(mask,contours,-1,255,cv2.FILLED)
    mask=cv2.dilate(mask,cv2.getStructuringElement(cv2.MORPH_CROSS,(3,3)))
    mask=cv2.bitwise_and(mask,roi);mask[frame[:,:,3]==0]=0
    template=frame.copy();template[mask==0]=0
    return template

def register(frame,template,angles,d,act,phase):
    x,y,w,h=cv2.boundingRect(template[:,:,3]);center=(x+w/2,y+h/2)
    target=frame.astype(np.float32)/255
    dx,dy=field(np.array([center[0]]),np.array([center[1]]),d,act,phase)
    expected=(center[0]+float(dx[0]),center[1]+float(dy[0]))
    # Include alpha in matching, so a dark backdrop is not mistaken for outlines.
    best=None
    for angle in angles:
        matrix=cv2.getRotationMatrix2D(center,angle,1)
        rotated=cv2.warpAffine(template,matrix,(48,48),flags=cv2.INTER_NEAREST)
        bx,by,bw,bh=cv2.boundingRect(rotated[:,:,3]);patch=rotated[by:by+bh,bx:bx+bw]
        if not bw or not bh:continue
        sx=max(0,int(round(expected[0]-bw/2))-2);sy=max(0,int(round(expected[1]-bh/2))-2)
        ex=min(48,sx+bw+5);ey=min(48,sy+bh+5)
        roi=target[sy:ey,sx:ex];mask=(patch[:,:,3]>0).astype(np.uint8)
        if roi.shape[0]<bh or roi.shape[1]<bw:continue
        score=cv2.matchTemplate(roi,patch.astype(np.float32)/255,cv2.TM_SQDIFF,mask=mask)
        value,_,loc,_=cv2.minMaxLoc(score)
        value=value/max(int(mask.sum())*4,1)+abs(angle)*0.0002
        if best is None or value<best[0]:
            matched=np.zeros_like(frame);matched[sy+loc[1]:sy+loc[1]+bh,sx+loc[0]:sx+loc[0]+bw]=patch
            best=(float(value),matched,dict(angle=angle,translation=[sx+loc[0]-bx,sy+loc[1]-by],bounds=[sx+loc[0],sy+loc[1],bw,bh]))
    return best

def holes(alpha):
    outside=255-alpha.copy();flood=outside.copy();cv2.floodFill(flood,None,(0,0),128)
    return ((outside==255)&(flood!=128)).astype(np.uint8)*255

def analyze(frame):
    alpha=frame[:,:,3];n,_,stats,_=cv2.connectedComponentsWithStats(alpha,connectivity=8)
    contour,_=cv2.findContours(alpha,cv2.RETR_EXTERNAL,cv2.CHAIN_APPROX_SIMPLE)
    return dict(components=n-1,areas=stats[1:,cv2.CC_STAT_AREA].tolist(),
                interior_transparent_pixels=int(np.count_nonzero(holes(alpha))),
                contour_points=sum(len(c) for c in contour))

def main():
    OUTPUT.mkdir(parents=True,exist_ok=True);manifest=json.loads((SOURCE/'frames.json').read_text())
    palette=np.array([tuple(bytes.fromhex(c[1:]))[::-1] for c in manifest['palette']],np.int16)
    report=[]
    for d in DIRS:
        sheet=cv2.imread(str(SOURCE/f'{d}-pixel.png'),cv2.IMREAD_UNCHANGED)
        ref=cv2.imread(str(REFERENCE/f'{d}-pixel.png'),cv2.IMREAD_UNCHANGED)[:48,:48]
        templates={k:material_template(ref,box,k) for k,box in [('shield',SHIELDS[d])]+([('sword',SWORDS[d])] if d in SWORDS else [])}
        for row,act in enumerate(ACTS):
            for f in range(4):
                original=sheet[row*48:(row+1)*48,f*48:(f+1)*48].copy();out=original.copy()
                record=dict(direction=d,animation=act,frame=f+1,before=analyze(original),objects=[])
                if act=='walk' or (act=='hit' and f>=2):
                    for part,template in templates.items():
                        if template is None:continue
                        found=register(original,template,[0] if act=='walk' else [-30,-20,-10,0,10,20,30],d,act,f)
                        if found is None:continue
                        error,matched,placement=found;mask=matched[:,:,3]>0
                        # Reject uncertain matches rather than invent a missing part.
                        accepted=error<0.11
                        protected=np.zeros((48,48),bool)
                        if act=='walk':
                            lo,hi,bottom=HEADS[d];protected[:bottom+1,lo:hi+1]=True
                            # No recognized sword may replace a helmet highlight.
                            if np.any(mask&protected&np.any(matched!=original,axis=2)):accepted=False
                        if accepted:out[mask]=matched[mask]
                        record['objects'].append(dict(part=part,match_error=error,accepted=accepted,**placement))
                    # Only small interior cavities in the torso; never the large
                    # gaps enclosed by sword trails or the space between feet.
                    hmask=holes(out[:,:,3]);n,labels,stats,_=cv2.connectedComponentsWithStats(hmask,connectivity=4)
                    repair=np.zeros((48,48),np.uint8)
                    for i in range(1,n):
                        x,y,w,h,area=stats[i]
                        if area<=5 and 18<=x and x+w<=31 and 23<=y and y+h<=34:repair[labels==i]=255
                    if repair.any():
                        rgb=cv2.inpaint(out[:,:,:3],repair,2,cv2.INPAINT_TELEA)
                        ys,xs=np.where(repair>0);values=rgb[ys,xs].astype(np.int16)
                        nearest=np.argmin(np.sum((values[:,None,:].astype(float)-palette[None,:,:])**2,axis=2),axis=1)
                        out[ys,xs,:3]=palette[nearest].astype(np.uint8);out[ys,xs,3]=255
                    record['cavity_repairs']=int(np.count_nonzero(repair))
                record['changed_pixels']=int(np.any(out!=original,axis=2).sum());record['after']=analyze(out)
                report.append(record);sheet[row*48:(row+1)*48,f*48:(f+1)*48]=out
        cv2.imwrite(str(OUTPUT/f'{d}-pixel.png'),sheet)
    manifest.update(version=7,source_recipe='tools/repair_knights_cv.py',cv_library=cv2.__version__,
                    experimental=True,note='OpenCV repair proposals for diagnosis. Not selected for runtime; semantic matches need visual confirmation.')
    (OUTPUT/'frames.json').write_text(json.dumps(manifest,indent=2)+'\n')
    (OUTPUT/'analysis.json').write_text(json.dumps(report,indent=2)+'\n')
    print('OpenCV',cv2.__version__,'analyzed',len(report),'frames; changed',sum(r['changed_pixels']>0 for r in report))

if __name__=='__main__':main()
