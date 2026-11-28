import json,glob,math,sys
SP=sys.argv[1]+'/assets'
fp=json.load(open(SP+'/graphics/scene.json'))['footprints']
TOWN={'S_zhuo','S_pingyuan','S_xindu','S_ye','S_beihai','S_luoyang','S_xudu','S_runan','S_road_town','S_xiaopei','S_xuzhou_town','S_xiapi','S_xiapi_inner','S_xiangyang','S_xinye','S_mangdang','S_vil_peace','S_jiangling','S_jiangxia_harbor'}
bad=0; n=0
for f in sorted(glob.glob(SP+'/scenario/hero/explore*.json')):
    d=json.load(open(f))
    for k,a in (d.get('Areas') or {}).items():
        m=a.get('Map')
        if m not in TOWN: continue
        mp=json.load(open(SP+'/graphics/maps/'+m+'.json'))
        occ=set()
        for o in mp['objects']:
            nm=o[0]
            if nm not in fp or not nm.startswith(('huabei','zhongyuan','huainan','jiangnan')): continue
            dx,dy=(o[3],o[4]) if len(o)>=5 and not isinstance(o[3],dict) else (0,0)
            cu=o[1]+.5+(dx/16+dy/8)/2; cv=o[2]+.5+(dy/8-dx/16)/2; F=fp[nm]
            occ|={(x,y) for x in range(math.floor(cu+F[0]+1e-6),math.ceil(cu+F[0]+F[2]-1e-6)) for y in range(math.floor(cv+F[1]+1e-6),math.ceil(cv+F[1]+F[3]-1e-6))}
        for w in mp.get('walls',[]):
            if not w[0].startswith(('huabei','zhongyuan','huainan','jiangnan')): continue
            (a0,b0),(a1,b1)=w[1],w[2]
            occ|={(x,y) for x in range(min(a0,a1),max(a0,a1)+1) for y in range(min(b0,b1),max(b0,b1)+1)}
        pts=[('사람 '+nm,p['At']) for nm,p in (a.get('People') or {}).items() if 'At' in p]+[('출구 '+(e.get('Name') or ''),e['At']) for e in a.get('Exits') or []]+([('시작',a['At'])] if 'At' in a else [])
        for lab,(x,y) in pts:
            n+=1
            if (x,y) in occ:
                bad+=1; print('덮임', f.split('/')[-1], k, lab, (x,y))
print('사람·출구·시작 칸', n, '개 중 덮인 것', bad)
