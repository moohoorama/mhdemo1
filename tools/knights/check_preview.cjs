/* Metadata/render contract test. No browser or screenshot automation. */
const assert = require('node:assert/strict');
const fs = require('node:fs');
const path = require('node:path');
const vm = require('node:vm');
const root = path.resolve(__dirname, '../..');
const origin = 'http://preview.local/';
let renderNext, drawCount = 0;
const drawnImages = new Set();
const context2d = {
  clearRect() {}, beginPath() {}, moveTo() {}, lineTo() {}, stroke() {}, closePath() {}, strokeRect() {},
  drawImage(image, sx, sy, sw, sh) {
    assert(sx >= 0 && sy >= 0 && sx + sw <= image.width && sy + sh <= image.height,
      `Invalid source rectangle in ${image.src}: ${[sx, sy, sw, sh]}`);
    drawnImages.add(new URL(image.src, origin).pathname); drawCount++;
  },
};
class Element {
  constructor(tag = 'div') {
    this.tag = tag; this.children = []; this.style = {}; this.dataset = {}; this.value = '';
    this.checked = false; this.width = 48; this.height = 48;
    this.classList = {toggle() {}};
  }
  append(...children) { this.children.push(...children); }
  replaceChildren(...children) {
    this.children = children;
    if (this.tag === 'select') this.value = children[0]?.value || '';
  }
  querySelector(tag) { return this.children.find(child => child.tag === tag); }
  getContext() { return context2d; }
  getBoundingClientRect() { return {left: 0, top: 0, width: this.width, height: this.height}; }
}
const elements = {};
const html = fs.readFileSync(path.join(root, 'knight-preview.html'), 'utf8');
for (const match of html.matchAll(/<(\w+)[^>]*\bid="([^"]+)"/g)) elements[match[2]] = new Element(match[1]);
elements.map.width = 400; elements.map.height = 240;
elements.speed.value = 'auto'; elements.zoom.value = '2'; elements.guides.checked = true;
class ImageFixture {
  set src(value) {
    this._src = value;
    const bytes = fs.readFileSync(path.join(root, new URL(value, origin).pathname));
    this.width = bytes.readUInt32BE(16); this.height = bytes.readUInt32BE(20);
    queueMicrotask(() => this.onload());
  }
  get src() { return this._src; }
}
const sandbox = vm.createContext({
  document: {
    getElementById: id => elements[id],
    createElement: tag => new Element(tag),
    querySelectorAll: () => elements.directions.children,
  },
  Option: class extends Element { constructor(label, value) { super('option'); this.textContent = label; this.value = value; } },
  Image: ImageFixture, URL, location: {href: origin + 'knight-preview.html'},
  fetch: async url => ({ok: true, json: async () => JSON.parse(fs.readFileSync(path.join(root, new URL(url, origin).pathname), 'utf8'))}),
  performance: {now: () => 0}, requestAnimationFrame: callback => { renderNext = callback; },
});
async function main() {
  await vm.runInContext(fs.readFileSync(path.join(root, 'knight-preview.js'), 'utf8'), sandbox);
  assert.equal(elements.cards.children.length, 8);
  assert.equal(elements.frames.children.length, 4);
  assert.equal(elements.action.value, 'walk');
  const project = JSON.parse(fs.readFileSync(path.join(root, 'tools/knights/project.json')));
  for (const version of project.preview.versions) {
    elements.version.value = version.id;
    await elements.version.onchange();
    assert.equal(elements.error.hidden, true);
    for (const animation of Object.keys(project.preview.animations)) {
      elements.action.value = animation; elements.action.onchange();
      for (let frame = 0; frame < 4; frame++) {
        elements.frame.value = String(frame); elements.frame.oninput();
        renderNext(0);
        assert.equal(elements.frameLabel.textContent, `${frame + 1} / 4`);
        assert.equal(vm.runInContext('active.lookup.size', sandbox), 160);
      }
    }
    assert(drawnImages.has('/' + version.image), `Wrong atlas loaded for ${version.id}`);
    assert.equal(vm.runInContext('active.lookup.get("N/hit/0")', sandbox), 12);
  }
  elements.map.onclick({clientX: 200, clientY: 38});
  assert.equal(vm.runInContext('state.units.length', sandbox), 9);
  elements.remove.onclick();
  assert.equal(vm.runInContext('state.units.length', sandbox), 8);
  elements.reset.onclick();
  assert.equal(vm.runInContext('state.units.length', sandbox), 8);
  console.log(`Preview contracts passed: ${project.preview.versions.length} versions, 5 actions, 4 frames (${drawCount} draws).`);
}
main().catch(error => { console.error(error); process.exitCode = 1; });
