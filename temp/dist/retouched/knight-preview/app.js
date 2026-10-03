'use strict';
const $=id=>document.getElementById(id), dirs=['N','NE','E','SE','S','SW','W','NW'];
const names=['대기','걷기','공격','방어 추정','앉기'];
let units=[],selected=0,playing=true,frame=0,last=0,action=0,direction=4,atlas,terrain;
const map=$('map'),ctx=map.getContext('2d'),detail=$('detail').getContext('2d');
function tile(u,v){return{x:200+(u-v)*16,y:30+(u+v)*8};}
function sprite(context,index,x,y){context.imageSmoothingEnabled=false;context.drawImage(atlas,index%16*40,Math.floor(index/16)*48,40,48,x,y,40,48);}
function mark(context,x,y){context.strokeStyle='#fb87bd';context.lineWidth=1;context.beginPath();context.moveTo(x-3,y+.5);context.lineTo(x+4,y+.5);context.moveTo(x+.5,y-3);context.lineTo(x+.5,y+4);context.stroke();}
function diamond(p,color){ctx.strokeStyle=color;ctx.beginPath();ctx.moveTo(p.x,p.y);ctx.lineTo(p.x+16,p.y+8);ctx.lineTo(p.x,p.y+16);ctx.lineTo(p.x-16,p.y+8);ctx.closePath();ctx.stroke();}
function reset(){units=dirs.map((_,i)=>({u:2+(i%4)*2,v:2+Math.floor(i/4)*4,d:i}));selected=4;direction=4;sync();}
function sync(){if(units[selected])direction=units[selected].d;$('selected').textContent=dirs[direction]+' · '+names[action];$('status').textContent=units.length+'명 배치';document.querySelectorAll('[data-dir]').forEach(b=>b.classList.toggle('active',+b.dataset.dir===direction));}
function setDirection(d){direction=d;if(units[selected])units[selected].d=d;sync();}
dirs.forEach((d,i)=>{const b=document.createElement('button');b.textContent=d;b.dataset.dir=i;b.onclick=()=>setDirection(i);$('directions').append(b);const card=document.createElement('button');card.className='card';card.onclick=()=>setDirection(i);card.innerHTML='<canvas width="40" height="48"></canvas><span>'+d+'</span>';$('cards').append(card);});
for(let i=0;i<4;i++){const card=document.createElement('button');card.className='card';card.innerHTML='<canvas width="40" height="48"></canvas><span>프레임 '+(i+1)+'</span>';card.onclick=()=>{playing=false;frame=i;$('play').textContent='재생';};$('frames').append(card);}
function drawSmall(canvas,d,f){const c=canvas.getContext('2d');c.clearRect(0,0,40,48);sprite(c,d*20+action*4+f,0,0);if($('guides').checked)mark(c,20,38);}
function render(now){if(playing&&now-last>=1000/+$('speed').value){frame=(frame+1)%4;last=now;}$('frame').value=frame;$('frameLabel').textContent=(frame+1)+' / 4';ctx.clearRect(0,0,400,240);ctx.imageSmoothingEnabled=false;
for(let sum=0;sum<=18;sum++)for(let u=0;u<10;u++){const v=sum-u;if(v<0||v>=10)continue;const p=tile(u,v);for(const [dx,dy]of [[0,0],[-8,4],[8,4],[0,8]])ctx.drawImage(terrain,112,16,16,8,p.x+dx-8,p.y+dy,16,8);if($('guides').checked)diamond(p,'#334d4270');}
if(units[selected])diamond(tile(units[selected].u,units[selected].v),'#92d3ff');
units.map((unit,i)=>({...unit,i})).sort((a,b)=>(a.u+a.v)-(b.u+b.v)).forEach(unit=>{const p=tile(unit.u,unit.v);sprite(ctx,unit.d*20+action*4+frame,p.x-20,p.y+8-38);if($('guides').checked)mark(ctx,p.x,p.y+8);});
detail.clearRect(0,0,40,48);sprite(detail,direction*20+action*4+frame,0,0);if($('guides').checked){detail.strokeStyle='#629acb';detail.strokeRect(.5,.5,39,47);mark(detail,20,38);}
document.querySelectorAll('#cards .card').forEach((card,d)=>{drawSmall(card.querySelector('canvas'),d,frame);card.classList.toggle('selected',d===direction);});
document.querySelectorAll('#frames .card').forEach((card,f)=>{drawSmall(card.querySelector('canvas'),direction,f);card.classList.toggle('selected',f===frame);});requestAnimationFrame(render);}
map.onclick=e=>{if(!atlas)return;const r=map.getBoundingClientRect(),x=(e.clientX-r.left)*400/r.width,y=(e.clientY-r.top)*240/r.height;const u=Math.round(((y-38)/8+(x-200)/16)/2),v=Math.round(((y-38)/8-(x-200)/16)/2);if(u<0||u>9||v<0||v>9)return;const p=tile(u,v);if(Math.abs(x-p.x)/16+Math.abs(y-(p.y+8))/8>1)return;const found=units.findIndex(a=>a.u===u&&a.v===v);if(found>=0)selected=found;else{units.push({u,v,d:direction});selected=units.length-1;}sync();};
$('action').onchange=()=>{action=+$('action').value;frame=0;last=performance.now();sync();};$('play').onclick=()=>{playing=!playing;$('play').textContent=playing?'일시정지':'재생';};$('frame').oninput=()=>{playing=false;frame=+$('frame').value;$('play').textContent='재생';};$('background').onchange=()=>{$('detail').style.background=$('background').value;};$('remove').onclick=()=>{if(units[selected])units.splice(selected,1);selected=Math.min(selected,units.length-1);sync();};$('reset').onclick=reset;function zoom(){map.style.width=400*+$('zoom').value+'px';map.style.height=240*+$('zoom').value+'px';}$('zoom').onchange=zoom;zoom();
function load(src){return new Promise((resolve,reject)=>{const i=new Image();i.onload=()=>resolve(i);i.onerror=()=>reject(Error(src+' 로드 실패'));i.src=src;});}
Promise.all([load('assets/knights.png'),load('assets/terrain.png')]).then(images=>{[atlas,terrain]=images;reset();requestAnimationFrame(render);}).catch(error=>{$('error').hidden=false;$('error').textContent=error.message;$('status').textContent='이미지 로드 실패';});
