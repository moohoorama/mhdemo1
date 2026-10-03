'use strict';

const $ = id => document.getElementById(id);
const map = $('map');
const ctx = map.getContext('2d');
const detail = $('detail').getContext('2d');
const aliases = {guard: 'hit', sit: 'exhausted'};
const state = {units: [], selected: 0, playing: true, frame: 0, last: 0, action: 'walk', direction: 'SW'};
const versions = new Map();
let project, current, active, terrain;
let directionCards = [], frameCards = [];

async function json(path) {
  const response = await fetch(path, {cache: 'no-store'});
  if (!response.ok) throw new Error(`${path} 로드 실패 (${response.status})`);
  return response.json();
}

function image(path) {
  return new Promise((resolve, reject) => {
    const result = new Image();
    result.onload = () => resolve(result);
    result.onerror = () => reject(new Error(`${path} 로드 실패`));
    result.src = path;
  });
}

async function loadVersion(version) {
  if (versions.has(version.id)) return versions.get(version.id);
  const metadata = await json(version.metadata);
  const url = version.image
    ? new URL(version.image, location.href)
    : new URL(metadata.sheet, new URL(version.metadata, location.href));
  url.searchParams.set('revision', metadata.asset_revision || String(metadata.version));
  const atlas = await image(url.href);
  const lookup = new Map();
  for (const frame of metadata.frames) {
    const animation = aliases[frame.animation] || frame.animation;
    lookup.set(`${frame.direction}/${animation}/${frame.frame}`, frame.index);
  }
  const result = {metadata, atlas, lookup};
  versions.set(version.id, result);
  return result;
}

function framesFor(animation = state.action) {
  return current.metadata.frames.filter(f => f.direction === current.metadata.directions[0] && f.animation === animation).length;
}

function sprite(context, direction, frame, x, y) {
  const index = active.lookup.get(`${direction}/${state.action}/${frame}`);
  if (index === undefined) return;
  const [width, height] = active.metadata.cell;
  const [pivotX, pivotY] = active.metadata.pivot;
  const [viewX, viewY] = current.metadata.pivot;
  const columns = active.metadata.columns;
  context.imageSmoothingEnabled = false;
  context.drawImage(active.atlas, index % columns * width, Math.floor(index / columns) * height,
    width, height, x + viewX - pivotX, y + viewY - pivotY, width, height);
}

function tile(u, v) { return {x: 200 + (u - v) * 16, y: 30 + (u + v) * 8}; }
function mark(context, x, y) {
  context.strokeStyle = '#fb87bd';
  context.lineWidth = 1;
  context.beginPath();
  context.moveTo(x - 3, y + .5); context.lineTo(x + 4, y + .5);
  context.moveTo(x + .5, y - 3); context.lineTo(x + .5, y + 4);
  context.stroke();
}
function diamond(point, color) {
  ctx.strokeStyle = color;
  ctx.beginPath();
  ctx.moveTo(point.x, point.y); ctx.lineTo(point.x + 16, point.y + 8);
  ctx.lineTo(point.x, point.y + 16); ctx.lineTo(point.x - 16, point.y + 8);
  ctx.closePath(); ctx.stroke();
}
function sync() {
  if (state.units[state.selected]) state.direction = state.units[state.selected].direction;
  $('selected').textContent = `${state.direction} · ${project.preview.animations[state.action].label}`;
  $('status').textContent = `${state.units.length}명 배치`;
  document.querySelectorAll('[data-dir]').forEach(button => button.classList.toggle('active', button.dataset.dir === state.direction));
}
function setDirection(direction) {
  state.direction = direction;
  if (state.units[state.selected]) state.units[state.selected].direction = direction;
  sync();
}
function reset() {
  state.units = current.metadata.directions.map((direction, i) => ({u: 2 + i % 4 * 2, v: 2 + Math.floor(i / 4) * 4, direction}));
  state.selected = Math.max(0, current.metadata.directions.indexOf('SW'));
  sync();
}
function selectFrame(frame) {
  state.playing = false; state.frame = frame;
  $('play').textContent = '재생';
}
function card(label, click) {
  const button = document.createElement('button');
  button.className = 'card';
  const canvas = document.createElement('canvas');
  [canvas.width, canvas.height] = current.metadata.cell;
  const span = document.createElement('span'); span.textContent = label;
  button.append(canvas, span); button.onclick = click;
  return button;
}
function makeFrameCards() {
  $('frames').replaceChildren();
  frameCards = Array.from({length: framesFor()}, (_, frame) => {
    const label = project.preview.animations[state.action].phases[frame] || '';
    const button = card(`${frame + 1} · ${label}`, () => selectFrame(frame));
    $('frames').append(button); return button;
  });
  $('frame').max = framesFor() - 1;
}
function drawSmall(canvas, direction, frame) {
  const context = canvas.getContext('2d');
  context.clearRect(0, 0, canvas.width, canvas.height);
  sprite(context, direction, frame, 0, 0);
  if ($('guides').checked) mark(context, ...current.metadata.pivot);
}
function frameDuration() {
  if ($('speed').value !== 'auto') return 1000 / Number($('speed').value);
  const timing = active.metadata.frame_durations_ms || current.metadata.frame_durations_ms;
  return timing?.[state.action]?.[state.frame] || 250;
}
function render(now) {
  if (state.playing && now - state.last >= frameDuration()) {
    state.frame = (state.frame + 1) % framesFor(); state.last = now;
  }
  $('frame').value = state.frame;
  $('frameLabel').textContent = `${state.frame + 1} / ${framesFor()}`;
  ctx.clearRect(0, 0, map.width, map.height); ctx.imageSmoothingEnabled = false;
  for (let sum = 0; sum <= 18; sum++) for (let u = 0; u < 10; u++) {
    const v = sum - u;
    if (v < 0 || v >= 10) continue;
    const point = tile(u, v);
    for (const [dx, dy] of [[0, 0], [-8, 4], [8, 4], [0, 8]]) {
      ctx.drawImage(terrain, 112, 16, 16, 8, point.x + dx - 8, point.y + dy, 16, 8);
    }
    if ($('guides').checked) diamond(point, '#334d4270');
  }
  const selected = state.units[state.selected];
  if (selected) diamond(tile(selected.u, selected.v), '#92d3ff');
  const [pivotX, pivotY] = current.metadata.pivot;
  [...state.units].sort((a, b) => a.u + a.v - b.u - b.v).forEach(unit => {
    const point = tile(unit.u, unit.v);
    sprite(ctx, unit.direction, state.frame, point.x - pivotX, point.y + 8 - pivotY);
    if ($('guides').checked) mark(ctx, point.x, point.y + 8);
  });
  drawSmall($('detail'), state.direction, state.frame);
  if ($('guides').checked) {
    detail.strokeStyle = '#629acb';
    detail.strokeRect(.5, .5, $('detail').width - 1, $('detail').height - 1);
  }
  directionCards.forEach((button, i) => {
    const direction = current.metadata.directions[i];
    drawSmall(button.querySelector('canvas'), direction, state.frame);
    button.classList.toggle('selected', direction === state.direction);
  });
  frameCards.forEach((button, frame) => {
    drawSmall(button.querySelector('canvas'), state.direction, frame);
    button.classList.toggle('selected', frame === state.frame);
  });
  requestAnimationFrame(render);
}
function showError(error) {
  $('error').hidden = false; $('error').textContent = error.message;
}
function bindControls() {
  map.onclick = event => {
    const rect = map.getBoundingClientRect();
    const x = (event.clientX - rect.left) * 400 / rect.width;
    const y = (event.clientY - rect.top) * 240 / rect.height;
    const u = Math.round(((y - 38) / 8 + (x - 200) / 16) / 2);
    const v = Math.round(((y - 38) / 8 - (x - 200) / 16) / 2);
    if (u < 0 || u > 9 || v < 0 || v > 9) return;
    const point = tile(u, v);
    if (Math.abs(x - point.x) / 16 + Math.abs(y - (point.y + 8)) / 8 > 1) return;
    const found = state.units.findIndex(unit => unit.u === u && unit.v === v);
    if (found >= 0) state.selected = found;
    else { state.units.push({u, v, direction: state.direction}); state.selected = state.units.length - 1; }
    sync();
  };
  $('action').onchange = () => { state.action = $('action').value; state.frame = 0; state.last = performance.now(); makeFrameCards(); sync(); };
  $('frame').oninput = () => selectFrame(Number($('frame').value));
  $('play').onclick = () => { state.playing = !state.playing; state.last = performance.now(); $('play').textContent = state.playing ? '일시정지' : '재생'; };
  $('background').onchange = () => { $('detail').style.background = $('background').value; };
  $('remove').onclick = () => { if (state.units[state.selected]) state.units.splice(state.selected, 1); state.selected = Math.min(state.selected, state.units.length - 1); sync(); };
  $('reset').onclick = reset;
  $('zoom').onchange = () => { map.style.width = `${400 * Number($('zoom').value)}px`; map.style.height = `${240 * Number($('zoom').value)}px`; };
  $('zoom').onchange();
  $('version').onchange = async () => {
    const version = project.preview.versions.find(version => version.id === $('version').value);
    $('version').disabled = true;
    try {
      active = await loadVersion(version); state.frame = 0; state.last = performance.now(); $('error').hidden = true;
    } catch (error) { showError(error); }
    finally { $('version').disabled = false; }
  };
}
async function main() {
  project = await json('tools/knights/project.json');
  [current, terrain] = await Promise.all([loadVersion(project.preview.versions[0]), image('assets/terrain.png')]);
  active = current;
  $('version').replaceChildren(...project.preview.versions.map(v => new Option(v.label, v.id)));
  $('action').replaceChildren(...current.metadata.animations.map(a => new Option(project.preview.animations[a].label, a)));
  $('action').value = state.action;
  [$('detail').width, $('detail').height] = current.metadata.cell;
  directionCards = current.metadata.directions.map(direction => {
    const button = document.createElement('button'); button.textContent = direction; button.dataset.dir = direction;
    button.onclick = () => setDirection(direction); $('directions').append(button);
    const preview = card(direction, () => setDirection(direction)); $('cards').append(preview); return preview;
  });
  makeFrameCards(); bindControls(); reset(); requestAnimationFrame(render);
}
main().catch(error => { showError(error); $('status').textContent = '이미지 로드 실패'; });
