# coding: utf-8

"""Rebuild initial campaign definitions from the checked-in design, plus draft placements."""
import json,re,pathlib
root=pathlib.Path(__file__).resolve().parents[1]
s=(root/'srpg_complete_design.md').read_text()
rows=lambda prefix:[[p.strip() for p in l.split('|')[1:-1]] for l in s.splitlines() if l.startswith('| '+prefix)]
split=lambda v: [] if v=='없음' else v.split('·')
d=dict(InitialInventory={'소군량':18,'소병법단':8},Version='opening-2',Draft='대사, 선택, 고정 배치, 초기 소모품 18/8, 승리 보급 8/4, 행동XP24/병법2배/격퇴2배/필요XP100*1.035^(L-1)/승리XP없음, 초기 특성치0: 플레이 가능한 초안. 그 외 수치: 통합 기획서 2부 초안.',Classes={},Officers={},Equipment={},Traits={},Skills={},Terrain={},Items={'소군량':600,'소병법단':15,'중군량':2000},CommonLearn='속공 반격술 침착 행군 강행 군율 방패 양생 집중 협동 행운 보급 책략방어 원거리방어 철벽 불굴'.split(),Stages=[],Nodes=[])
cls=['경기병','경보병','궁병','사마','적병']
mp={'경기병':[12,.6,2,.08],'경보병':[14,.6,2,.08],'궁병':[14,.6,2,.08],'사마':[28,.65,5,.13],'적병':[14,.6,2,.09]}
for r in rows('class_'):
 name=r[0].split()[1]
 if name not in cls:continue
 weapon,armor=r[2].split('/');movement,ran,_=r[3].split(' / ')
 b,mc,rb,rc=mp[name]
 d['Classes'][name]=dict(Name=name,Family=r[1].split('/')[0],Weapon=weapon,Armor=armor,Move=int(movement),Range=int(ran.split('-')[1]),Bonus=list(map(float,r[4].split('/'))),HPFactor=int(r[5]),Skills=split(r[6]),Learn=split(r[7]),MPBase=b,MPCoeff=mc,RecoveryBase=rb,RecoveryCoeff=rc)
itemnames='철검 철창 단궁 죽선 가죽갑옷 평복 군마 군량낭 지도 약낭 병법입문 고정도'.split()
for r in rows('item_'):
 id,name=r[0].split()
 if name not in itemnames:continue
 typ=r[1];slot='weapon' if typ in ['검','창','활','부채'] else 'armor' if typ in ['갑옷','옷'] else 'aux'
 d['Equipment'][id]=dict(Name=name,Slot=slot,Type=typ,Bonus=list(map(float,r[2].split('/'))),Traits=split(r[3]),Learn=split(r[4]),Skills=split(r[5]))
lookup={e['Name']:id for id,e in d['Equipment'].items()}
profiles=[('유비',1,'경보병','인덕·격려','철검·가죽갑옷·병법입문'),('관우',1,'경기병','호걸·반격술','철창·가죽갑옷·군마'),('장비',1,'경기병','위압·맹공','철창·가죽갑옷·군마'),('간옹',2,'궁병','행운·정화','단궁·가죽갑옷·약낭'),('정원지',4,'적병','도적숙련','철검·평복·지도'),('등무',2,'적병','맹공','철검·평복·군량낭'),('장각',2,'사마','격려','죽선·평복·병법입문'),('화웅',6,'경기병','맹공·위압','철창·가죽갑옷·군마'),('호진',4,'경기병','기병숙련','철창·가죽갑옷·군마'),('여포',9,'경기병','용장·호걸','철창·가죽갑옷·군마'),('이유',7,'사마','계략','죽선·평복·병법입문'),('이각',7,'경기병','기병숙련','철창·가죽갑옷·군마')]
charrows={r[0].split()[1]:r for r in rows('character_')}
for name,level,c,t,eq in profiles:
 r=charrows[name];d['Officers'][name]=dict(ID=name,Name=name,Class=c,Stats=list(map(int,r[1].split('/'))),Level=level,Traits=split(t),Learn=split(r[4]) if name in ['유비','관우','장비','간옹'] else [],Equipment=[lookup[x] for x in split(eq)])
required=set(d['CommonLearn'])
for o in d['Officers'].values():required.update(o['Traits']+o['Learn'])
for c in d['Classes'].values():required.update(c['Learn'])
for e in d['Equipment'].values():required.update(e['Traits']+e['Learn'])
grants={'속공':['신속'],'반격술':['반격'],'원시술':['원시'],'맹공':['맹공'],'분기술':['분기'],'격려':['격려','분발'],'계략':['혼란','봉책'],'정화':['해독','정신통일','정화'],'논객':['설파','이간'],'지휘':['대분발','대방진']}
for r in rows('trait_'):
 name=r[0].split()[1]
 if name in required:d['Traits'][name]=dict(Cost=int(r[1]),Skills=grants.get(name,[]))
skills=set(sum([c['Skills'] for c in d['Classes'].values()],[])+sum([e['Skills'] for e in d['Equipment'].values()],[])+sum([t['Skills'] for t in d['Traits'].values()],[]))
kind={'물리':'physical','책략':'magic','상태':'status','회복':'heal','버프':'buff'}
effects={'신속':'speed','돌격':'charge','반격':'counter','원시':'range','소회복':'heal','격려':'morale','분발':'attack','대분발':'attack','대방진':'defense','혼란':'confusion','봉책':'seal','설파':'sealroot','이간':'confusion','해독':'curepoison','정신통일':'curemental','정화':'cure'}
for r in rows('skill_'):
 name=r[0].split()[1]
 if name not in skills:continue
 k,mode=r[1].split('/');ran,shape=r[4].split('/');lo,hi=(0,0) if ran=='자기' else (1,0) if ran=='기본사거리' else map(int,ran.split('-'))
 d['Skills'][name]=dict(Kind=kind[k],Mode=mode,Shape='all' if shape=='사거리내모든적' else 'cross' if '십자' in shape else 'single',Effect=effects.get(name,''),Condition='water' if name=='탁류' else '',Cost=int(r[2]),Hit=int(r[3].strip('%')),Min=lo,Max=hi,Coeff=float(r[5]),Duration=1 if r[6]=='당턴' else 2 if r[6] in ['2턴','다음 자기 턴 시작'] else 0)
terr={'기병':[(1.1,1),(1,2),(.8,4),(.8,3),(.9,2),(1,1),(1,1)],'보병':[(1,1),(1,1),(1.1,2),(1.1,2),(1.2,1),(1.2,1),(1.1,1)],'궁병':[(1,1),(1,1),(1,2),(1.1,2),(1.1,1),(1.1,1),(1,1)],'문관':[(1,1),(1,1),(.9,3),(1,2),(1,1),(1.1,1),(1.1,1)],'도적':[(1,1),(1.1,1),(1.2,2),(1.2,1),(.8,2),(.9,1),(1,1)]}
for f,values in terr.items():d['Terrain'][f]={c:dict(Factor=v[0],Cost=v[1]) for c,v in zip('.dsfciv',values)};d['Terrain'][f]['~']=dict(Factor=1,Cost=0)
for i,(names,limit,boss) in enumerate([(['정원지','등무','장각'],3,'장각'),(['화웅','호진'],4,'화웅'),(['여포','이유','이각'],4,'여포')]):
 w,h=18+2*i,14+2*i;tiles=[list('.'*w) for _ in range(h)]
 if i==0:
  for y in range(5,9):
   for x in range(1,4):tiles[y][x]='v'
 else:
  for y in range(4,h-2):tiles[y][w-5]='c'
  for y in range(1,4):
   for x in range(w-4,w):tiles[y][x]='i' if i==2 else 'c'
  if i==2:
   for y in range(h):
    for x in range(w-6):tiles[y][x]='d'
 for y in range(2,5):tiles[y][w//2]='f'
 for y in range(h-3,h):tiles[y][w//2]='~'
 enemies=[dict(Officer=n,X=w-4 if j==0 else w-6,Y=6+j*2) for j,n in enumerate(names)]
 if i==2:
  enemies[1].update(X=w-2,Y=h-3);enemies[2].update(X=w-2,Y=1)
 for j in range(4 if i==0 else 6):
  oid=f'B0{i+1}_troop_{j+1}';c='적병' if i==0 else '경보병' if j%2==0 else '궁병';stats=[35,45,20,40,45,40] if i==0 else [50,48,30,40,40,45] if j%2==0 else [40,43,30,52,45,40]
  d['Officers'][oid]=dict(ID=oid,Name=('황건 도적' if i==0 else '일반 부대')+str(j+1),Class=c,Stats=stats,Level=2+i*2,Traits=[],Learn=[],Equipment=[lookup['철검' if c!='궁병' else '단궁'],lookup['평복' if c=='적병' else '가죽갑옷']])
  enemies.append(dict(Officer=oid,X=w-9+j%2,Y=2+j*2))
 d['Stages'].append(dict(ID=f'B0{i+1}',Name=['황건적의 난','사수관 전투','호로관 전투'][i],Boss=boss,Width=w,Height=h,Limit=limit,Threshold=40 if i==2 else 0,Tiles=[''.join(r) for r in tiles],Allies=[dict(X=2,Y=5+j) for j in range(4)],Enemies=enemies,Reward={'소군량':8,'소병법단':4,**({'중군량':1} if i==2 else {})}))
d['Nodes']=[dict(ID='oath',Kind='dialogue',Text='유비: 백성을 지키기 위해 함께 뜻을 세웁시다. 관우·장비: 생사를 함께하겠습니다.',Next='oath_choice'),dict(ID='oath_choice',Kind='choice',Text='도원결의: 의용군을 일으킨다.',Choices={'결의':'prep1'}),dict(ID='prep1',Kind='preparation',Text='황건적의 난 — 출전 준비'),dict(ID='join',Kind='join',Officer='간옹',Event='join-간옹',Next='joined'),dict(ID='joined',Kind='dialogue',Text='간옹: 나도 백성을 위한 싸움에 힘을 보태겠소.',Next='coalition'),dict(ID='coalition',Kind='dialogue',Text='동탁 타도군 일어나다. 유비: 연합군과 함께 사수관으로 나아갑시다.',Next='prep2'),dict(ID='prep2',Kind='preparation',Text='사수관 — 출전 준비'),dict(ID='sword',Kind='grant',Item='item_024',Count=1,Event='grant-고정도',Next='chase'),dict(ID='chase',Kind='dialogue',Text='손건: 이 고정도를 받아 주십시오. 유비: 감사하오. 이제 호로관으로 추격합시다.',Next='prep3'),dict(ID='prep3',Kind='preparation',Text='호로관 — 출전 준비. 고정도는 창고에서 직접 장착한다.')]
(root/'assets/content/campaign.json').write_text(json.dumps(d,ensure_ascii=False,indent=2)+'\n')

# A low-cost spell must not be cheaper than a scholar's per-turn recovery.
for name in ['초열','탁류']:
 d['Skills'][name]['Cost']=20
d['Draft']+=' 초열/탁류 비용20: 문관 회복14 대비 자원 소모를 남기는 초안.'
# Draft HP coefficients are tuned without changing combat formulas.
for name,hp in {'경기병':3,'경보병':3,'궁병':3,'사마':3,'적병':2}.items():
 d['Classes'][name]['HPFactor']=hp
d['Items']['소군량']=220
d['Items']['중군량']=700
d['Draft']+=' 기본 병종 HP계수 3/3/3/3/2, 소군량220/중군량700: 동일 능력치 격퇴 타격 수 측정에 따른 조정.'
# Functional promotion prototypes expose only already implemented abilities.
paths={'경기병':['중기병','친위대'],'경보병':['중보병','근위대'],'궁병':['노병','연노병'],'사마':['참모','군사'],'적병':['흉적','의적']}
for base,path in paths.items():
 prev=base
 d['Classes'][base]['Tier']=0
 for tier,name in enumerate(path,1):
  r=next(r for r in rows('class_') if r[0].split()[1]==name)
  old=d['Classes'][base]; c=dict(old); c.pop('Promotes',None)
  c.update(Name=name,Tier=tier,Bonus=list(map(float,r[4].split('/'))),HPFactor=int(r[5]),Move=int(r[3].split(' / ')[0]),Range=int(r[3].split(' / ')[1].split('-')[1]),Skills=[x for x in split(r[6]) if x in d['Skills']],Learn=[x for x in split(r[7]) if x in d['Traits']],MPBase=old['MPBase']+6*tier)
  d['Classes'][name]=c
  d['Classes'][prev]['Promotes']=[name]
  prev=name
 d['Items'][f'승급:{path[0]}']=0
 d['Items'][f'승급:{path[1]}']=0
d['Stages'][1]['Duels']=[dict(ID='sishui-guan-yu',Ally='관우',Enemy='화웅',Outcome='victory',AllyResult='stay',EnemyResult='retreat',Text='관우: 내 칼을 받아라! 화웅이 물러났다.')]
d['Stages'][2]['Duels']=[dict(ID='hulao-zhang-fei',Ally='장비',Enemy='여포',Outcome='draw',AllyResult='stay',EnemyResult='stay',Text='장비와 여포의 일기토가 무승부로 끝났다.')]
d['Draft']+=' 승급 프로토타입은 계수와 기존 구현 능력만 제공. 후반 고유 병법/학습 목록 미공급. 일기토 결과는 영걸전 공략을 참고한 고정 초안.'
(root/'assets/content/campaign.json').write_text(json.dumps(d,ensure_ascii=False,indent=2)+'\n')
