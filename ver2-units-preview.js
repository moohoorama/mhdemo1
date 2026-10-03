'use strict';

const $ = id => document.getElementById(id);
const BASE = 'tools/spritetool/assets/ver2-units/';
const UNITS = [
  {id: 'infantry', name: '경보병'}, {id: 'bandit', name: '황건 적병'}, {id: 'spearman', name: '창병'},
  {id: 'archer', name: '궁병'}, {id: 'cavalry', name: '경기병'},
  {id: 'guanyu', name: '관우'}, {id: 'zhangfei', name: '장비'},
];
const ACTIONS = {idle: '대기', walk: '걷기', attack: '공격', hit: '피격', exhausted: '탈진'};
const DIRECTION_NAMES = {N: '북', NE: '북동', E: '동', SE: '남동', S: '남', SW: '남서', W: '서', NW: '북서'};
const COMPASS = ['NW', 'N', 'NE', 'W', null, 'E', 'SW', 'S', 'SE'];
const state = {unit: 'infantry', direction: 'SW', action: 'idle', frame: 0, playing: true, elapsed: 0, last: 0, faction: 'wei'};
const units = new Map();
const recolored = new Map();
let factions = {team_keys: [], factions: []};
let directionCanvases = [], motionCanvases = [];

async function loadUnit(id) {
  if (units.has(id)) return units.get(id);
  const response = await fetch(`${BASE}${id}/frames.json`, {cache: 'no-store'});
  if (!response.ok) throw new Error(`${id}/frames.json 로드 실패 (${response.status})`);
  const meta = await response.json();
  const stamp = Date.now();
  const sheets = {};
  await Promise.all(meta.directions.map(d => new Promise((resolve, reject) => {
    const image = new Image();
    image.onload = () => resolve(sheets[d] = image);
    image.onerror = () => reject(new Error(`${id}/${d}-pixel.png 로드 실패`));
    image.src = `${BASE}${id}/${d}-pixel.png?v=${stamp}`;
  })));
  const lookup = new Map(meta.frames.map(f => [`${f.direction}/${f.animation}/${f.frame}`, f.crop]));
  const unit = {meta, sheets, lookup};
  units.set(id, unit);
  return unit;
}

const rgb = hex => [1, 3, 5].map(i => parseInt(hex.slice(i, i + 2), 16));

// Team key pixels (factions.json team_keys) take the selected faction's ramp.
function sheet(unit, direction) {
  const faction = factions.factions.find(f => f.id === state.faction);
  if (!faction || faction.ramp.join() === factions.team_keys.join()) return unit.sheets[direction];
  const key = `${state.unit}/${faction.id}/${direction}`;
  if (!recolored.has(key)) {
    const image = unit.sheets[direction];
    const canvas = document.createElement('canvas');
    canvas.width = image.width;
    canvas.height = image.height;
    const ctx = canvas.getContext('2d');
    ctx.drawImage(image, 0, 0);
    const data = ctx.getImageData(0, 0, canvas.width, canvas.height);
    const px = data.data, from = factions.team_keys.map(rgb), to = faction.ramp.map(rgb);
    for (let i = 0; i < px.length; i += 4) {
      if (px[i + 3] !== 255) continue;
      const k = from.findIndex(c => c[0] === px[i] && c[1] === px[i + 1] && c[2] === px[i + 2]);
      if (k >= 0) [px[i], px[i + 1], px[i + 2]] = to[k];
    }
    ctx.putImageData(data, 0, 0);
    recolored.set(key, canvas);
  }
  return recolored.get(key);
}

function draw(canvas, direction, action, frame, zoom, guides = $('guides').checked) {
  const unit = units.get(state.unit);
  const [w, h] = unit.meta.cell;
  const [px, py] = unit.meta.pivot;
  canvas.width = w * zoom;
  canvas.height = h * zoom;
  const ctx = canvas.getContext('2d');
  ctx.imageSmoothingEnabled = false;
  ctx.fillStyle = $('background').value;
  ctx.fillRect(0, 0, canvas.width, canvas.height);
  const crop = unit.lookup.get(`${direction}/${action}/${frame}`);
  if (crop) ctx.drawImage(sheet(unit, direction), crop[0], crop[1], crop[2], crop[3], 0, 0, w * zoom, h * zoom);
  if (!guides) return;
  ctx.strokeStyle = '#5aa9ff';
  ctx.lineWidth = 1;
  ctx.strokeRect(.5, .5, canvas.width - 1, canvas.height - 1);
  ctx.strokeStyle = '#ff6fc8';
  const cx = (px + .5) * zoom, cy = (py + .5) * zoom, r = Math.max(4, zoom * 1.5);
  ctx.beginPath();
  ctx.moveTo(cx - r, cy); ctx.lineTo(cx + r, cy);
  ctx.moveTo(cx, cy - r); ctx.lineTo(cx, cy + r);
  ctx.stroke();
}

function frameCount() {
  const unit = units.get(state.unit);
  return unit.meta.frames.filter(f => f.direction === state.direction && f.animation === state.action).length;
}

function render() {
  const unit = units.get(state.unit);
  if (!unit) return;
  draw($('main'), state.direction, state.action, state.frame, +$('zoom').value);
  directionCanvases.forEach(({canvas, direction}) => draw(canvas, direction, state.action, state.frame, 2));
  $('frame').max = frameCount() - 1;
  $('frame').value = state.frame;
  $('frameLabel').textContent = `${state.frame + 1} / ${frameCount()}`;
  $('play').textContent = state.playing ? '일시정지' : '재생';
}

function renderMotions() {
  motionCanvases.forEach(({canvas, action, frame}) => {
    draw(canvas, state.direction, action, frame, 2, false);
    canvas.parentElement.classList.toggle('selected', action === state.action && frame === state.frame);
  });
}

function renderInfo() {
  const meta = units.get(state.unit).meta;
  const unitName = UNITS.find(u => u.id === state.unit).name;
  $('info').innerHTML = `<dt>선택</dt><dd>${unitName} · ${DIRECTION_NAMES[state.direction]}(${state.direction}) · ${ACTIONS[state.action]}</dd>` +
    `<dt>셀 · 기준점</dt><dd>${meta.cell.join(' × ')} · (${meta.pivot.join(', ')})</dd>` +
    `<dt>프레임 시간</dt><dd>${[].concat(meta.frame_durations_ms[state.action]).join(' · ')} ms</dd>`;
  document.querySelectorAll('#units button').forEach(b => b.classList.toggle('active', b.dataset.id === state.unit));
  document.querySelectorAll('#actions button').forEach(b => b.classList.toggle('active', b.dataset.id === state.action));
  document.querySelectorAll('#compass button').forEach(b => b.classList.toggle('active', b.dataset.id === state.direction));
  document.querySelectorAll('#units button, #actions button, #compass button').forEach(b => b.setAttribute('aria-pressed', b.classList.contains('active')));
  directionCanvases.forEach(({canvas, direction}) => canvas.parentElement.classList.toggle('selected', direction === state.direction));
}

function refresh() {
  render();
  renderMotions();
  renderInfo();
}

function setFrame(frame) {
  state.frame = (frame + frameCount()) % frameCount();
  state.elapsed = 0;
  render();
  renderMotions();
}

function choose(changes) {
  Object.assign(state, changes);
  if (state.frame >= frameCount()) state.frame = 0;
  state.elapsed = 0;
  history.replaceState(null, '', `#${state.unit}/${state.direction}/${state.action}/${state.faction}`);
  refresh();
}

function fromHash() {
  const [unit, direction, action, faction] = location.hash.slice(1).split('/');
  if (faction) state.faction = faction;
  if (UNITS.some(u => u.id === unit)) state.unit = unit;
  if (direction in DIRECTION_NAMES) state.direction = direction;
  if (action in ACTIONS) state.action = action;
}

function button(parent, id, label, onClick) {
  const b = document.createElement('button');
  b.dataset.id = id;
  b.innerHTML = label;
  b.onclick = onClick;
  parent.append(b);
  return b;
}

function buildControls() {
  UNITS.forEach(u => button($('units'), u.id, u.name, async () => {
    try {
      await loadUnit(u.id);
      choose({unit: u.id});
    } catch (error) {
      showError(error);
    }
  }));
  Object.entries(ACTIONS).forEach(([id, name], i) => button($('actions'), id, `${i + 1}. ${name}`, () => choose({action: id})));
  factions.factions.forEach(f => $('faction').append(new Option(f.name, f.id, false, f.id === state.faction)));
  $('faction').onchange = e => choose({faction: e.target.value});
  COMPASS.forEach(d => {
    if (!d) return $('compass').append(document.createElement('div'));
    button($('compass'), d, `<span>${d}</span><span class="muted">${DIRECTION_NAMES[d]}</span>`, () => choose({direction: d}));
  });
  directionCanvases = units.get(state.unit).meta.directions.map(direction => {
    const card = document.createElement('div');
    card.className = 'card';
    card.onclick = () => choose({direction});
    const canvas = document.createElement('canvas');
    canvas.setAttribute('aria-label', `${DIRECTION_NAMES[direction]} 방향 재생`);
    card.append(canvas, `${direction} · ${DIRECTION_NAMES[direction]}`);
    $('directions').append(card);
    return {canvas, direction};
  });
  motionCanvases = [];
  Object.entries(ACTIONS).forEach(([action, name]) => {
    const label = document.createElement('div');
    label.className = 'name';
    label.textContent = name;
    $('motions').append(label);
    for (let frame = 0; frame < 4; frame++) {
      const card = document.createElement('div');
      card.className = 'card';
      card.onclick = () => {
        state.playing = false;
        choose({action, frame});
      };
      const canvas = document.createElement('canvas');
      canvas.setAttribute('aria-label', `${name} ${frame + 1}프레임`);
      card.append(canvas, `${frame + 1}`);
      $('motions').append(card);
      motionCanvases.push({canvas, action, frame});
    }
  });
  $('play').onclick = () => { state.playing = !state.playing; render(); };
  $('prev').onclick = () => { state.playing = false; setFrame(state.frame - 1); };
  $('next').onclick = () => { state.playing = false; setFrame(state.frame + 1); };
  $('frame').oninput = e => { state.playing = false; setFrame(+e.target.value); };
  ['zoom', 'background', 'guides'].forEach(id => $(id).onchange = refresh);
  document.addEventListener('keydown', e => {
    if (e.target.matches('input, select')) return;
    if (e.key === ' ') { e.preventDefault(); $('play').click(); }
    else if (e.key === 'ArrowLeft') $('prev').click();
    else if (e.key === 'ArrowRight') $('next').click();
    else if (e.key >= '1' && e.key <= '5') choose({action: Object.keys(ACTIONS)[+e.key - 1]});
  });
}

function tick(now) {
  const delta = state.last ? now - state.last : 0;
  state.last = now;
  if (state.playing && units.has(state.unit)) {
    const speed = $('speed').value;
    const ms = units.get(state.unit).meta.frame_durations_ms[state.action];
    const duration = speed === 'auto' ? (Array.isArray(ms) ? ms[state.frame] : ms) : 1000 / +speed;
    state.elapsed += delta;
    if (state.elapsed >= duration) {
      state.elapsed %= duration;
      state.frame = (state.frame + 1) % frameCount();
      render();
      renderMotions();
    }
  }
  requestAnimationFrame(tick);
}

function showError(error) {
  $('error').hidden = false;
  $('error').textContent = `${error.message} — 저장소 루트에서 'make preview-knights'로 서버를 열고 http://127.0.0.1:8765/ver2-units-preview.html 로 접속하세요.`;
}

fromHash();
fetch(`${BASE}factions.json`, {cache: 'no-store'})
  .then(r => r.ok ? r.json() : factions)
  .then(table => { factions = table; })
  .catch(() => {})
  .then(() => loadUnit(state.unit))
  .then(() => {
    buildControls();
    refresh();
    requestAnimationFrame(tick);
  })
  .catch(showError);
