// 从原版 fqxy/map/*.php 提取每个节点($dty 分段)的 NPC 链接（clj=7 走 NPC 页面）放置数据
// 输出 hxxy_mapnpcs_extract.json: [{dtx,dty,npc_id,name}]
const fs = require('fs')
const path = require('path')

const mapDir = 'd:/mxz-code/github/codingce3gJy/西游一键端24082201/phpstudy_pro/WWW/fqxy/map'
const files = {
  xsc: 0, cac: 1, lg: 2, hdml: 3, hd: 4, jsh: 5, hdmlsc: 6, yg: 7, ghl: 8, xl: 9,
  fcs: 10, sl: 11, xgt: 12, pts: 13, zzbl: 14, zzl: 15, hy: 16, dyt: 17, xyt: 18,
  bmy: 19, bl: 20, gjz: 21, df: 22, tg: 23, alg: 24, bxg: 25, wjg: 26, ccg: 27,
  neg: 28, jsg: 29, zzg: 30, yh: 31, pty: 32, zyt: 33, ljl: 34, scl: 35, hsl: 36,
  jt: 37, bgd: 38, pds: 39, jjg: 40, yls: 41, tl: 42, bfg: 43, xsmg: 44, byzzl: 45,
  bglm: 46, wsc: 47, yc: 48, bjt: 49, byd: 50, yld: 51, lhd: 52, jdd: 53, jds: 54,
  cys: 55, xlys: 56, psd: 57, qls: 58, jjl: 59, bqg: 60, tfg: 61, fxq: 62, yhq: 63,
  jpf: 64, tzg: 65, ygd: 66, zgz: 67, xyj: 68, dyj: 69, kfgc: 70,
  mz: 71, hz: 72, gzai: 73, gczc: 74, vipqy: 75, emgc: 76, cwd: 77,
  bjd: 80, ttsf: 81, psdd: 82, std: 83, wdd: 84, bhmj: 85, wxz01: 86,
  vip1qy: 87, vip2qy: 88, vip3qy: 89, vip4qy: 90,
}

const out = []
let stats = { files: 0, sections: 0, links: 0 }

for (const [name, dtx] of Object.entries(files)) {
  const fp = path.join(mapDir, name + '.php')
  if (!fs.existsSync(fp)) { console.log('MISS', name); continue }
  const src = fs.readFileSync(fp, 'utf8')
  stats.files++
  // 按 $dty==N 分段切
  const segRe = /(if|elseif)\s*\(\s*\$dty\s*==\s*(\d+)\s*\)\s*\{/g
  const marks = []
  let m
  while ((m = segRe.exec(src))) marks.push({ dty: parseInt(m[2]), start: m.index + m[0].length })
  for (let i = 0; i < marks.length; i++) {
    const end = i + 1 < marks.length ? src.lastIndexOf('} elseif', marks[i + 1].start) : src.length
    const seg = src.slice(marks[i].start, end > marks[i].start ? end : src.length)
    stats.sections++
    // 依序扫描 clj/npc/echo 三元组
    const lineRe = /\$clj\[\]=(\d+);|\$npc\[\]=(\d+);|<font color=(blue|red)>([^<]+)<\/font>/g
    let curClj = 0, curNpc = 0
    let lm
    while ((lm = lineRe.exec(seg))) {
      if (lm[1] !== undefined) { curClj = parseInt(lm[1]); continue }
      if (lm[2] !== undefined) { curNpc = parseInt(lm[2]); continue }
      // echo 文本
      if (curClj === 7 && curNpc > 0) {
        const txt = lm[4].trim()
        if (txt && txt !== '刷新' && txt !== '查看地图') {
          out.push({ dtx, dty: marks[i].dty, npc_id: curNpc, name: txt })
          stats.links++
        }
      }
      curClj = 0; curNpc = 0
    }
  }
}

fs.writeFileSync('d:/mxz-code/github/codingce3gJy/qqjiayuan/server/internal/seed/hxxy_data/mapnpcs_extract.json', JSON.stringify(out, null, 1), 'utf8')
console.log('done', JSON.stringify(stats), 'rows', out.length)
