'use strict';
const titles=['뒤로 젖힘','옆으로 비틀림','무릎 꺾임','몸이 들리는 반동'];
const descriptions=['발은 버티고, 머리와 가슴이 뒤로 크게 젖혀집니다.','충격에 상체가 옆으로 돌아가며 균형을 잃습니다.','복부를 맞아 몸을 접고 무릎이 꺾입니다.','강한 충격으로 두 발이 잠깐 뜨며 뒤로 밀립니다.'];
const grid=document.getElementById('grid'),play=document.getElementById('play');
let frame=0,playing=true,last=0,sheet;
titles.forEach((title,row)=>{const article=document.createElement('article');article.innerHTML=`<h2>${row+1}. ${title}</h2><p>${descriptions[row]}</p><canvas class="stage" width="256" height="256" aria-label="후보 ${row+1} 애니메이션"></canvas><div class="strip"></div>`;grid.append(article);for(let f=0;f<4;f++){const b=document.createElement('button');b.innerHTML=`<canvas width="128" height="128"></canvas><span>${f+1}</span>`;b.setAttribute('aria-label',`후보 ${row+1} 프레임 ${f+1}`);b.onclick=()=>{frame=f;playing=false;play.textContent='재생';};article.querySelector('.strip').append(b);}});
function draw(canvas,row,f){const c=canvas.getContext('2d');c.imageSmoothingEnabled=false;c.clearRect(0,0,canvas.width,canvas.height);const w=sheet.width/4,h=sheet.height/4;c.drawImage(sheet,f*w,row*h,w,h,0,0,canvas.width,canvas.height);}
function render(t){const durations=[380,280,220,360];if(playing&&t-last>=durations[frame]*Number(document.getElementById('speed').value)){frame=(frame+1)%4;last=t;}document.getElementById('frame').textContent=(frame+1)+' / 4';grid.querySelectorAll('article').forEach((article,row)=>{draw(article.querySelector('.stage'),row,frame);article.querySelectorAll('.strip button').forEach((b,f)=>{draw(b.querySelector('canvas'),row,f);b.classList.toggle('active',f===frame);});});requestAnimationFrame(render);}
play.onclick=()=>{playing=!playing;last=performance.now();play.textContent=playing?'일시정지':'재생';};
document.getElementById('impact').onclick=()=>{frame=2;playing=false;play.textContent='재생';};
sheet=new Image();sheet.onload=()=>requestAnimationFrame(render);sheet.onerror=()=>{const e=document.getElementById('error');e.hidden=false;e.textContent='후보 이미지를 불러오지 못했습니다.';};sheet.src='tools/spritetool/assets/knight-hit-candidates/SW-hit-options.png';
