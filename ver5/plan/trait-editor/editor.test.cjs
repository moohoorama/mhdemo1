'use strict';

const assert = require('node:assert/strict');
const fs = require('node:fs');
const path = require('node:path');
const vm = require('node:vm');
const { test } = require('node:test');

const html = fs.readFileSync(path.resolve(__dirname, '../../../_newfeature-editor.html'), 'utf8');
const seed = JSON.parse(html.match(/<script id="seed-data" type="application\/json">([\s\S]*?)<\/script>/)[1]);
const code = html.match(/<script id="editor-model">([\s\S]*?)<\/script>/)[1];
const context = vm.createContext({});
vm.runInContext(code + '\nglobalThis.Model = EditorModel;', context);
const model = context.Model;
const trait = (data, name) => data.traits.find(item => item.name === name);
const owner = (data, name) => data.owners.find(item => item.name === name);

test('embedded draft preserves the current document and learning assignments', () => {
  model.validate(seed);
  assert.equal(seed.traits.length, 146);
  assert.equal(seed.owners.filter(item => item.type === 'class').length, 25);
  assert.equal(seed.owners.filter(item => item.type === 'officer').length, 61);
  const markdown = fs.readFileSync(path.resolve(__dirname, '../../../_newfeature.md'), 'utf8');
  for (const item of seed.traits) assert.ok(markdown.includes('| ' + item.name + ' |'), item.name);
  for (const item of seed.owners) {
    const names = item.traits.map(id => model.trait(seed, id).name).join(', ') || '미정';
    assert.ok(markdown.includes('| ' + item.name + ' | ' + names + ' |'), item.name);
  }
});

test('renaming updates references and prose without renaming longer trait names', () => {
  const data = model.clone(seed);
  const root = trait(data, '기합');
  model.rename(data, root.id, '전의');
  assert.equal(trait(data, '기합확산').requires[0], root.id);
  assert.match(trait(data, '기합확산').description, /전의이 자신/);
  assert.ok(owner(data, '무도가').traits.includes(root.id));
  for (const name of ['보법(가칭)', '기술[1]+', 'a.b?', 'x|y', '검\\술']) {
    root.name = name;
    root.description = name + '을 배운다';
    model.rename(data, root.id, '검증용 이름');
    assert.equal(root.description, '검증용 이름을 배운다');
  }
  assert.throws(() => model.rename(data, root.id, '필살'), /같은 이름/);
});

test('name replacement can leave free text unchanged', () => {
  const data = model.clone(seed);
  const root = trait(data, '기합');
  const description = trait(data, '기합확산').description;
  model.rename(data, root.id, '전의', false);
  assert.equal(trait(data, '기합확산').description, description);
  assert.equal(trait(data, '기합확산').requires[0], root.id);
});

test('cyclic prerequisites are rejected without keeping the invalid edit', () => {
  const data = model.clone(seed);
  const root = trait(data, '군율');
  const child = trait(data, '엄명');
  assert.throws(() => model.setRequires(data, root.id, [child.id]), /순환/);
  assert.equal(root.requires.length, 0);
  assert.throws(() => model.setRequires(data, root.id, [root.id]), /순환/);
  assert.equal(root.requires.length, 0);
});

test('assignment changes reverse ownership and the live prerequisite graph', () => {
  const data = model.clone(seed);
  const officer = owner(data, '유비');
  const root = trait(data, '군율');
  const child = trait(data, '엄명');
  model.assign(data, officer.id, child.id, true);
  assert.ok(model.ownersOf(data, child.id).some(item => item.id === officer.id));
  let graph = model.graph(data, officer.id, false);
  assert.ok(graph.nodes.some(node => node.id === root.id && node.source === 'missing'));
  assert.ok(graph.edges.some(edge => edge.from === root.id && edge.to === child.id));
  model.assign(data, officer.id, root.id, true);
  graph = model.graph(data, officer.id, false);
  assert.ok(graph.nodes.some(node => node.id === root.id && node.source === 'own'));
  model.assign(data, officer.id, child.id, false);
  assert.ok(!model.ownersOf(data, child.id).some(item => item.id === officer.id));
});

test('class preview includes inherited learning pools without changing direct ownership', () => {
  const data = model.clone(seed);
  const officer = owner(data, '유비');
  const guards = owner(data, '근위병');
  assert.equal(Array.from(model.classChain(data, guards.id), item => item.name).join(','), '경보병,중보병,근위병');
  const graph = model.graph(data, officer.id, true, guards.id);
  const ironwall = trait(data, '철벽');
  assert.ok(graph.nodes.some(node => node.id === ironwall.id && node.source === 'class'));
  assert.ok(!model.ownersOf(data, ironwall.id).some(item => item.id === officer.id));
});

test('deleting removes prerequisite and ownership references', () => {
  const data = model.clone(seed);
  const root = trait(data, '군율');
  model.remove(data, root.id);
  assert.ok(!trait(data, '엄명').requires.includes(root.id));
  assert.ok(!owner(data, '간옹').traits.includes(root.id));
  model.validate(data);
});

test('Markdown export uses edited names, descriptions, prerequisites and inverse ownership', () => {
  const data = model.clone(seed);
  const root = trait(data, '기합');
  model.rename(data, root.id, '전의');
  root.description = '첫 줄 | 둘째 항목\n다음 줄';
  model.assign(data, owner(data, '유비').id, root.id, true);
  const markdown = model.markdown(data);
  assert.ok(markdown.includes('| 전의 |'));
  assert.ok(!markdown.includes('| 기합 |'));
  assert.ok(markdown.includes('첫 줄 \\| 둘째 항목<br>다음 줄'));
  assert.ok(markdown.includes('병종: 무도가 / 인물: 유비'));
  assert.ok(markdown.includes('| 유비 | 양생, 의협, 인덕, 용인술, 보급, 전의 |'));
});

test('invalid imports are rejected before replacing a valid draft', () => {
  const missing = model.clone(seed);
  missing.traits[0].requires = ['missing'];
  assert.throws(() => model.validate(missing), /존재하지 않는/);
  const cycle = model.clone(seed);
  owner(cycle, '경보병').parents = [owner(cycle, '근위병').id];
  assert.throws(() => model.validate(cycle), /승급 경로가 순환/);
  const kind = model.clone(seed);
  kind.traits[0].activation = '사용';
  assert.throws(() => model.validate(kind), /특기는 자동/);
});
