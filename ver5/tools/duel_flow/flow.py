"""Builds duel beats (Stage.Duels[].Beats) from flow tokens; see .claude/skills/duel-authoring/SKILL.md."""
import glob,json,re,sys
ATK=re.compile(r'block|dodge|critical|^hit$|slash-through|^shoot$')
def jong(n): c=ord(n[-1]); return 0xAC00<=c<=0xD7A3 and (c-0xAC00)%28!=0
def ga(n): return n+('이' if jong(n) else '가')
def eul(n): return n+('을' if jong(n) else '를')
D={'talk':2.4,'approach':1.2,'block':1.05,'rush-block':1.1,'yielding-block':1.2,'critical-block':1.2,'dodge':1.05,'critical-dodge':1.2,'critical':1.2,'hit':1.2,'press':1.3,'close':1.1,'backstep':1.6,'retreat':3.0,'withdrawn':1.1,'draw':2.5,'win':1.6}
def B(k,side,title,speech='',dur=None): return {"Kind":k,"Title":title,"Speech":speech,"Side":side,"Duration":dur or D[k]}
def tk(side,title,text): return B('talk',side,title,text,round(min(3.6,2.0+0.045*len(text)),1))
class Duel:
    def __init__(s,names,weapons): s.n=names; s.w=weapons
    def A(s,a): return s.n[a]
    def build(s,plan):
        out=[]
        for t in plan:
            k=t[0]
            if k=='B': out.append(t[1]); continue
            if k=='T': out.append(tk(t[1],t[2],t[3])); continue
            out+=getattr(s,'t_'+k)(*t[1:])
        return out
    # a = the side that attacks in the token; b the other
    def t_probe(s,a,sp=''):  # 서로 떠 봄 (2합)
        b=1-a; n=s.n; w=s.w
        return [B('block',a,f"{ga(n[a])} {w[a]} 끝으로 떠 보자 {ga(n[b])} 가볍게 받아 넘기다",sp),
                B('block',b,f"{ga(n[b])} 곧바로 되받아 치다")]
    def t_trade(s,a,sp=''): # 맞받아침 (2합)
        b=1-a; n=s.n; w=s.w
        return [B('block',a,f"{ga(n[a])} {w[a]}을 내리치다",sp),B('block',b,f"{ga(n[b])} 맞받아 쳐 내다")]
    def t_press(s,a,sp=''): # 몰아붙임 (2합)
        b=1-a; n=s.n; w=s.w
        return [B('press',a,f"{ga(n[a])} 몰아붙이자 {ga(n[b])} 뒷걸음질 치다"),
                B('rush-block',a,f"{ga(n[a])} 거푸 {w[a]}을 휘몰아치다",sp),
                B('yielding-block',a,f"{ga(n[b])} 막아 내나 밀려나다")]
    def t_evade(s,a,sp=''): # a가 크게 휘두르고 b가 피함 (1합)
        b=1-a; n=s.n
        return [B('dodge',a,f"{ga(n[a])} 크게 휘둘렀으나 {ga(n[b])} 몸을 틀어 피하다",sp)]
    def t_counter(s,a,sp=''): # 빈틈을 파고든 반격 (1합)
        b=1-a; n=s.n
        return [B('block',a,f"{ga(n[a])} 빈틈을 파고들어 되받다",sp)]
    def t_clash(s,a,spa='',spb=''): # 정면 충돌 (2합)
        b=1-a; n=s.n
        return [B('critical-block',a,f"{ga(n[a])} 힘을 실어 내려치다",spa),B('critical-block',b,f"{ga(n[b])} 정면으로 맞받아 불꽃이 튀다",spb)]
    def t_lock(s,a,sp=''): # 힘겨루기 (1합)
        n=s.n; w=s.w
        return [B('block',a,f"{ga(n[0])} {n[1]}의 {w[1]}과 맞물려 힘겨루기를 하다",sp)]
    def t_push(s,a,sp=''): # a가 눌러 b가 밀림 (1합)
        b=1-a; n=s.n
        return [B('yielding-block',a,f"{ga(n[a])} 힘으로 누르자 {ga(n[b])} 밀려나다",sp)]
    def t_rush(s,a,sp=''): # 말을 몰아 돌진 (1합)
        n=s.n
        return [B('rush-block',a,f"{ga(n[a])} 말을 몰아 들이치다",sp)]
    def t_stag(s,a,sp=''): # 일격에 크게 밀림 (hit 1합)
        b=1-a; n=s.n
        return [B('hit',a,f"{n[a]}의 일격에 {ga(n[b])} 크게 밀리다",sp)]
    def t_edodge(s,a,sp=''): # 강공을 간신히 피함 (1합)
        b=1-a; n=s.n
        return [B('critical-dodge',a,f"{n[a]}의 일격을 {ga(n[b])} 간신히 피하다",sp)]
    def t_cb(s,a,sp=''): # 강공 막음 (1합)
        b=1-a; n=s.n
        return [B('critical-block',a,f"{ga(n[a])} 힘껏 내려치나 {ga(n[b])} 막아 내다",sp)]
    def t_kill(s,a,sp=''):
        n=s.n
        return [B('critical',a,f"{ga(n[a])} 결정타를 넣다",sp)]
    def t_gap(s,a): # 거리 벌리고 다시 마주 섬 (0합)
        n=s.n
        return [B('backstep',a,f"{ga(n[a])} 말을 물려 거리를 벌리다")]
    def t_close(s,a): return [B('close',a,f"{ga(s.n[a])} 다시 거리를 좁히다")]
def count(bs): return sum(1 for b in bs if ATK.search(b['Kind']))


t=lambda side,title,text:('T',side,title,text)
ap=lambda side,title:('B',B('approach',side,title))
def E(k,side,title,speech=''): return ('B',B(k,side,title,speech))
PL={}
def plan(duel_id,target,names,weapons,tokens): PL[duel_id]=(target,Duel(names,weapons),tokens)

def run(plans_file,write=False):
    """Loads a plans file (calls plan(...)), checks each 합 count against its target and, with write, replaces the Beats in assets/scenario/hero/stage*.json."""
    import importlib.util
    spec=importlib.util.spec_from_file_location('plans',plans_file); m=importlib.util.module_from_spec(spec); spec.loader.exec_module(m)
    bad=0
    for i,(tg,d,toks) in PL.items():
        n=count(d.build(toks)); bad+=n!=tg; print(f"{i:34s} target {tg:2d} got {n:2d}",'' if n==tg else '<<<')
    if bad or not write: return
    for f in sorted(glob.glob('assets/scenario/hero/stage*.json')):
        s=open(f).read();d=json.loads(s);nl=s.endswith(chr(10));ch=False
        for x in (d.get('Stage',d)).get('Duels') or []:
            if x['ID'] in PL: x['Beats']=PL[x['ID']][1].build(PL[x['ID']][2]);ch=True
        if ch: open(f,'w').write(json.dumps(d,ensure_ascii=False,indent=1)+(chr(10) if nl else ''))

if __name__=='__main__':
    import flow
    flow.run(sys.argv[1],'--write' in sys.argv)
