// 提取幻想西游功能NPC数据 → map_npcs.json
// 数据来源：
//   1. template/xy020.php  传送分支（$csm==N → 目的地坐标 + 注释中的NPC名与所在坐标）
//   2. npc/npc.php         功能NPC对话文本（<font color=black>）与商店链接（clj[]=246）
//   3. map/*.php + npc.php NPC图片映射（pic/npc/npcN.png 与相邻链接文字）
//   4. maps.json           坐标→地名解析（种子数据）
// 输出：server/internal/seed/hxxy_data/map_npcs.json
//   [{ dtx, dty, npc_id, name, img, dialogue, shop, teles:[{name,dtx,dty}] }]

const fs = require('fs');
const path = require('path');

const XY = 'D:/mxz-code/github/codingce3gJy/西游一键端24082201/phpstudy_pro/WWW/fqxy';
const OUT = path.join(__dirname, '../../server/internal/seed/hxxy_data/map_npcs.json');
const MAPS = path.join(__dirname, '../../server/internal/seed/hxxy_data/maps.json');
const NPCS = path.join(__dirname, '../../server/internal/seed/hxxy_data/npcs.json');

const maps = JSON.parse(fs.readFileSync(MAPS, 'utf8'));
const npcs = JSON.parse(fs.readFileSync(NPCS, 'utf8'));
const nodeName = (x, y) => {
  const n = maps.find(m => m.dtx === x && m.dty === y);
  return n ? n.name : null;
};
const npcIdByName = name => {
  const n = npcs.find(n => n.name === name);
  return n ? n.npc_id : 0;
};

// ---------- 1. xy020.php 传送分支 ----------
const xy020 = fs.readFileSync(path.join(XY, 'template/xy020.php'), 'utf8');
const teleRaw = [];
// 分支头：if($csm ==1){ 或 } elseif($csm ==66){
// 注释：//长安---泾水河--1-116 或 //东海龙宫--船夫--0-6（可能紧跟在分支行后或下一行）
const branchRe = /\}\s*elseif\(\$csm\s*==\s*(\d+)\)\s*\{([\s\S]*?)(?=\}\s*elseif\(\$csm|\}elseif\(\$csm|$)/g;
// 第一个分支 if($csm ==1){
const firstIdx = xy020.indexOf('if($csm ==1){');
let body1 = '';
if (firstIdx >= 0) {
  const m = xy020.slice(firstIdx).match(/^if\(\$csm\s*==\s*(\d+)\)\s*\{([\s\S]*?)(?=\}\s*elseif\(\$csm|\}elseif\(\$csm)/);
  if (m) body1 = m[0];
}
const bodies = [];
if (body1) {
  const mm = body1.match(/if\(\$csm\s*==\s*(\d+)\)\s*\{([\s\S]*)/);
  bodies.push([mm[1], mm[2]]);
}
let m2;
while ((m2 = branchRe.exec(xy020)) !== null) bodies.push([m2[1], m2[2]]);

for (const [csm, body] of bodies) {
  const dtxM = body.match(/\$dtx\s*=\s*(\d+)\s*;/);
  const dtyM = body.match(/\$dty\s*=\s*(\d+)\s*;/);
  if (!dtxM || !dtyM) continue;
  const cmtM = body.match(/^\s*(?:\/\/|#)\s*(.+)$/m);
  if (!cmtM) continue;
  // 注释尾部 --x-y 为NPC所在坐标
  const cm = cmtM[1].match(/^(.*?)-{2,}(\d+)-(\d+)\s*$/);
  if (!cm) continue;
  const label = cm[1].replace(/-{2,}/g, '--').trim();
  teleRaw.push({
    csm: parseInt(csm, 10),
    label,                       // 目的地描述（2段式为"目的地--NPC名"，3段式为"源地---目的地"）
    npc: label.includes('--') ? label.split('--').pop().trim() : '',
    src_x: parseInt(cm[2], 10),
    src_y: parseInt(cm[3], 10),
    dst_x: parseInt(dtxM[1], 10),
    dst_y: parseInt(dtyM[1], 10),
  });
}
console.log('xy020 传送分支:', teleRaw.length);

// ---------- 2. npc.php 对话 + 商店 ----------
const npcPhp = fs.readFileSync(path.join(XY, 'npc/npc.php'), 'utf8');
const dialogues = {}; // npcc -> {dialogue, shop}
const nre = /\}\s*elseif\s*\(\$npcc\s*==\s*(\d+)\)\s*\{([\s\S]*?)(?=\}\s*elseif\s*\(\$npcc|$)/g;
let nm;
while ((nm = nre.exec(npcPhp)) !== null) {
  const id = parseInt(nm[1], 10);
  const body = nm[2];
  const blacks = [];
  const bre = /<font color=black>([^<]+)<\/font>/g;
  let bm;
  while ((bm = bre.exec(body)) !== null) {
    const t = bm[1].trim();
    if (t) blacks.push(t);
  }
  const hasShop = /clj\[\]=246/.test(body);
  // 过滤未替换的 PHP 变量等脏文本
  const clean = blacks.filter(t => !t.includes('$')).slice(0, 2).join('');
  if (clean || hasShop) {
    dialogues[id] = { dialogue: clean, shop: hasShop };
  }
}
console.log('npc.php 对话NPC:', Object.keys(dialogues).length);

// ---------- 3. NPC图片映射（map/*.php 与 npc/npc.php） ----------
const imgMap = {}; // npc名 -> npcN.png
const imgScan = (content) => {
  // 模式：$img='pic/npc/npcN.png' 后 60 行内出现 <font color=blue>名字</font>
  const lines = content.split('\n');
  let lastImg = null;
  for (const line of lines) {
    const im = line.match(/\$img\s*=\s*'pic\/npc\/(npc\d+\.png)'/);
    if (im) { lastImg = im[1]; continue; }
    const lm = line.match(/<font color=blue>([^<]+)<\/font>/);
    if (lm && lastImg) {
      const name = lm[1].trim();
      if (name && !imgMap[name]) imgMap[name] = lastImg;
      lastImg = null;
    }
  }
};
imgScan(npcPhp);
const mapDir = path.join(XY, 'map');
for (const f of fs.readdirSync(mapDir)) {
  if (!f.endsWith('.php')) continue;
  imgScan(fs.readFileSync(path.join(mapDir, f), 'utf8'));
}
console.log('NPC图片映射:', Object.keys(imgMap).length);

// 可用NPC图片清单（pic/npc 下实际存在的文件）
const npcImgDir = path.join(XY, 'pic/npc');
const npcImgs = fs.readdirSync(npcImgDir).filter(f => /^npc\d+\.png$/.test(f)).sort();
// 主要NPC配图（手工指定，取原版风格接近的图片）
const curatedImg = {
  '村长': 'npc1.png', '张果老': 'npc2.png', '李白': 'npc3.png', '船夫': 'npc4.png',
  '张二妈': 'npc5.png', '南极仙翁': 'npc6.png', '渔夫': 'npc7.png',
};
// 其余NPC按名字哈希取一张实际存在的图，保证确定性
function imgFor(name) {
  if (curatedImg[name] && npcImgs.indexOf(curatedImg[name]) >= 0) return curatedImg[name];
  let h = 0;
  for (const ch of name) h = (h * 31 + ch.charCodeAt(0)) >>> 0;
  return npcImgs[h % npcImgs.length];
}

// ---------- 4. 组装传送NPC（按 NPC名+坐标 分组） ----------
const placements = new Map(); // key -> row
const key = (x, y, name) => x + '_' + y + '_' + name;
for (const t of teleRaw) {
  const srcName = nodeName(t.src_x, t.src_y);
  const dstName = nodeName(t.dst_x, t.dst_y);
  if (!srcName || !dstName) continue; // 坐标在提取后的地图中不存在则跳过
  const k = key(t.src_x, t.src_y, t.npc);
  if (!placements.has(k)) {
    const id = npcIdByName(t.npc);
    placements.set(k, {
      dtx: t.src_x, dty: t.src_y,
      npc_id: id, name: t.npc,
      img: imgMap[t.npc] || imgFor(t.npc),
      dialogue: (dialogues[id] && dialogues[id].dialogue) || '',
      shop: '',
      teles: [],
    });
  }
  const row = placements.get(k);
  // 跳过自引用与重复传送
  if (t.dst_x === t.src_x && t.dst_y === t.src_y) continue;
  if (!row.teles.some(x => x.dtx === t.dst_x && x.dty === t.dst_y)) {
    row.teles.push({ name: dstName, dtx: t.dst_x, dty: t.dst_y });
  }
}
console.log('传送NPC位置:', placements.size);

// ---------- 5. 补充功能NPC（村长/商店/服务设施，位置按地图节点名匹配） ----------
function addPlace(dtx, dty, name, opts = {}) {
  const id = npcIdByName(name);
  const k = key(dtx, dty, name);
  if (placements.has(k)) {
    const row = placements.get(k);
    if (opts.shop) row.shop = opts.shop;
    if (opts.dialogue) row.dialogue = opts.dialogue;
    if (opts.img && !row.img) row.img = opts.img;
    return;
  }
  placements.set(k, {
    dtx, dty, npc_id: id, name,
    img: opts.img || imgMap[name] || imgFor(name),
    dialogue: opts.dialogue || (dialogues[id] && dialogues[id].dialogue) || '',
    shop: opts.shop || '',
    teles: opts.teles || [],
  });
}

// 新手村（0区）：村长家/广场/杂货店/码头
addPlace(0, 0, '村长', { dialogue: '哎！老了，不管事了！' });
addPlace(0, 1, '张果老');
addPlace(0, 1, '李白');
addPlace(0, 5, '张二妈', { shop: 'medicine', dialogue: '开着渔村唯一一家店铺！' });

// 服务设施NPC：按节点名自动放置
const serviceNpc = [
  { kw: '药店', name: '药店老板', shop: 'medicine', dialogue: '需要疗伤的药品吗？', img: 'npc5.png' },
  { kw: '铁匠铺', name: '铁匠', shop: 'weapon', dialogue: '好铁百炼，宝刀赠英雄！', img: 'npc16.png' },
  { kw: '杂货', name: '杂货老板', shop: 'grocery', dialogue: '杂货百货，样样俱全。', img: 'npc3.png' },
  { kw: '宠物店', name: '宠物店老板', shop: 'pet', dialogue: '想拥有一只可爱的宠物吗？', img: 'npc38.png' },
  { kw: '钱庄', name: '钱庄老板', shop: 'bank', dialogue: '存取银两，安全可靠。', img: 'npc4.png' },
  { kw: '当铺', name: '当铺老板', shop: 'warehouse', dialogue: '典当寄存，童叟无欺。', img: 'npc4.png' },
  { kw: '客栈', name: '店小二', shop: 'rest', dialogue: '客官，住店休息可以恢复气血法力！', img: 'npc6.png' },
  { kw: '首饰', name: '首饰老板', shop: 'jewel', dialogue: '上好的首饰，戴上倍有面子。', img: 'npc5.png' },
];
for (const s of serviceNpc) {
  for (const n of maps) {
    if (n.name.indexOf(s.kw) >= 0) addPlace(n.dtx, n.dty, s.name, s);
  }
}

// ---------- 输出 ----------
// 清理：无传送/无商店/无对话的空放置不输出；补默认对话
const rows = [...placements.values()]
  .filter(r => r.teles.length || r.shop || r.dialogue || r.npc_id > 0)
  .map(r => {
    if (!r.dialogue) r.dialogue = r.teles.length ? '你要去哪里？' : '有何贵干？';
    return r;
  })
  .sort((a, b) => a.dtx - b.dtx || a.dty - b.dty || a.name.localeCompare(b.name));
fs.writeFileSync(OUT, JSON.stringify(rows, null, 1), 'utf8');
console.log('输出:', OUT, rows.length, '条');
const withTele = rows.filter(r => r.teles.length).length;
const withShop = rows.filter(r => r.shop).length;
const noNpc = rows.filter(r => !r.npc_id).map(r => r.name);
console.log('带传送:', withTele, ' 带商店/服务:', withShop, ' 未匹配npc_id:', [...new Set(noNpc)].join(','));
