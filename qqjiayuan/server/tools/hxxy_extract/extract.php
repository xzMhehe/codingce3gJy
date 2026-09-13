<?php
/**
 * 幻想西游数据提取脚本
 * 用法: php extract.php [输出目录]
 * 从西游一键端 PHP 源码提取 地图/NPC/刷怪/物品/装备/技能/宠物/BOSS/头衔 数据为 JSON
 * 提取的文案会过滤"诺哈"等外站广告字样
 */

$WWW = 'D:/mxz-code/github/codingce3gJy/西游一键端24082201/phpstudy_pro/WWW';
$OUT = isset($argv[1]) ? rtrim($argv[1], '/\\') : __DIR__ . '/../../internal/seed/hxxy_data';
if (!is_dir($OUT)) { mkdir($OUT, 0777, true); }

function clean($s) {
    if ($s === null) return '';
    $s = (string)$s;
    $s = str_replace('诺哈', '', $s);
    // 过滤外站链接
    $s = preg_replace('#https?://[^\s\'"<>]+#u', '', $s);
    return trim($s);
}

function saveJson($out, $name, $data, &$summary) {
    file_put_contents($out . '/' . $name . '.json', json_encode($data, JSON_UNESCAPED_UNICODE));
    $summary[] = $name . ': ' . count($data) . ' 条';
}

$summary = [];

/* ============ 1. 地图节点 xdt/*.php ============ */
$maps = [];
foreach (glob($WWW . '/fqxy/xdt/*.php') as $f) {
    if (basename($f) == 'index.php') continue;
    $c = file_get_contents($f);
    if (!preg_match("/\\\$json\s*=\s*'(.*?)';/s", $c, $m)) continue;
    $arr = json_decode($m[1], true);
    if (!is_array($arr)) continue;
    foreach ($arr as $row) {
        if (!is_array($row)) continue;
        foreach ($row as $cell) {
            if (!is_array($cell) || !isset($cell['id']) || !isset($cell['dtx'])) continue;
            $maps[] = [
                'node_id' => (int)$cell['id'],
                'dtx' => (int)$cell['dtx'],
                'dty' => (int)$cell['dty'],
                'name' => clean($cell['mz'] ?? ''),
                'desc' => clean($cell['ms'] ?? ''),
                'up' => (string)($cell['up'] ?? ''),
                'down' => (string)($cell['down'] ?? ''),
                'left' => (string)($cell['left'] ?? ''),
                'right' => (string)($cell['right'] ?? ''),
                'up_jump' => (string)($cell['up_jump'] ?? ''),
                'down_jump' => (string)($cell['down_jump'] ?? ''),
                'left_jump' => (string)($cell['left_jump'] ?? ''),
                'right_jump' => (string)($cell['right_jump'] ?? ''),
            ];
        }
    }
}
// 按 node_id 去重
$uniq = [];
foreach ($maps as $n) { $uniq[$n['node_id']] = $n; }
$maps = array_values($uniq);
usort($maps, function ($a, $b) { return $a['node_id'] - $b['node_id']; });
saveJson($OUT, 'maps', $maps, $summary);

/* ============ 2. 变量块提取工具（NPC/BOSS/宠物/头衔） ============ */
function captureVars($file, $idVar, $id, $varNames) {
    foreach ($varNames as $v) { $$v = null; }
    $$idVar = $id;
    $npcc = $id;
    $nname = null;
    try {
        set_error_handler(function ($no, $str) { throw new ErrorException($str); });
        include $file;
        restore_error_handler();
    } catch (Throwable $e) {
        restore_error_handler();
        // 部分块含动态 include 逻辑，无法静态提取，跳过
        if ($nname === null) return null;
    }
    $res = ['id' => $id];
    foreach ($varNames as $v) { $res[$v] = isset($$v) ? $$v : null; }
    return $res;
}

function extractBlocks($files, $idVar, $varNames) {
    // 建立 id -> 文件 映射
    $map = [];
    foreach ($files as $f) {
        if (!file_exists($f)) continue;
        $c = file_get_contents($f);
        if (!preg_match_all('/\$' . $idVar . '\s*==\s*(\d+)/', $c, $mm)) continue;
        foreach ($mm[1] as $id) { if (!isset($map[$id])) $map[$id] = $f; }
    }
    $out = [];
    foreach ($map as $id => $f) {
        $r = captureVars($f, $idVar, (int)$id, $varNames);
        if ($r === null) continue;
        $first = $varNames[0];
        $v = $r[$first] ?? null;
        if ($v === null || $v === '' || (is_string($v) && trim($v) === '')) continue;
        $out[] = $r;
    }
    usort($out, function ($a, $b) { return $a['id'] - $b['id']; });
    return $out;
}

/* ============ 3. NPC 属性 npc/npcxx01~14.php ============ */
$npcVarNames = ['name' => 'nname', 'level' => 'ndj', 'hp' => 'nhp', 'max_hp' => 'nmaxhp',
    'mp' => 'nmp', 'max_mp' => 'nmaxmp', 'atk' => 'ngj', 'mg' => 'nmg', 'def' => 'nfy', 'mf' => 'nmf',
    'bg' => 'nbg', 'hg' => 'nhg', 'lg' => 'nlg', 'bf' => 'nbf', 'hf' => 'nhf', 'lf' => 'nlf', 'take' => 'ntake'];
$npcFiles = [];
for ($i = 1; $i <= 20; $i++) {
    $f = sprintf($WWW . '/fqxy/npc/npcxx%02d.php', $i);
    if (file_exists($f)) $npcFiles[] = $f;
}
$npcRaw = extractBlocks($npcFiles, 'npcc', array_values($npcVarNames));
$npcs = [];
foreach ($npcRaw as $r) {
    $row = ['npc_id' => $r['id']];
    foreach ($npcVarNames as $key => $var) {
        $v = $r[$var];
        if ($key == 'name' || $key == 'take') $v = clean($v);
        else $v = (int)($v ?? 0);
        $row[$key] = $v;
    }
    $npcs[] = $row;
}
saveJson($OUT, 'npcs', $npcs, $summary);

/* ============ 4. BOSS 属性 npc/bossxx01~8.php（独立 id 空间） ============ */
$bossFiles = [];
for ($i = 1; $i <= 10; $i++) {
    $f = sprintf($WWW . '/fqxy/npc/bossxx%02d.php', $i);
    if (file_exists($f)) $bossFiles[] = $f;
}
$bossRaw = extractBlocks($bossFiles, 'npcc', array_values($npcVarNames));
$bosses = [];
foreach ($bossRaw as $r) {
    $row = ['boss_id' => $r['id']];
    foreach ($npcVarNames as $key => $var) {
        $v = $r[$var];
        if ($key == 'name' || $key == 'take') $v = clean($v);
        else $v = (int)($v ?? 0);
        $row[$key] = $v;
    }
    $bosses[] = $row;
}
saveJson($OUT, 'bosses', $bosses, $summary);

/* ============ 5. 宠物种族 cw/cwxx01.php ============ */
$petVarNames = ['name' => 'nname', 'level' => 'ndj', 'hp' => 'nhp', 'max_hp' => 'nmaxhp',
    'mp' => 'nmp', 'max_mp' => 'nmaxmp', 'atk' => 'ngj', 'mg' => 'nmg', 'def' => 'nfy', 'mf' => 'nmf',
    'bg' => 'nbg', 'hg' => 'nhg', 'lg' => 'nlg', 'bf' => 'nbf', 'hf' => 'nhf', 'lf' => 'nlf'];
$petRaw = extractBlocks([$WWW . '/fqxy/cw/cwxx01.php', $WWW . '/fqxy/cw/cwxx02.php'], 'cwid', array_values($petVarNames));
$pets = [];
foreach ($petRaw as $r) {
    $row = ['species_id' => $r['id']];
    foreach ($petVarNames as $key => $var) {
        $v = $r[$var];
        if ($key == 'name') $v = clean($v);
        else $v = (int)($v ?? 0);
        $row[$key] = $v;
    }
    $pets[] = $row;
}
saveJson($OUT, 'pets', $pets, $summary);

/* ============ 6. 头衔 wp/txxx.php ============ */
$titleRaw = extractBlocks([$WWW . '/fqxy/wp/txxx.php'], 'npcc', ['wpmz', 'wpms', 'max1', 'max2', 'max3', 'max4']);
$titles = [];
foreach ($titleRaw as $r) {
    if (clean($r['wpmz']) === '') continue;
    $titles[] = [
        'title_id' => $r['id'],
        'name' => clean($r['wpmz']),
        'desc' => clean($r['wpms']),
        'hp' => (int)($r['max1'] ?? 0),
        'atk' => (int)($r['max2'] ?? 0),
        'def' => (int)($r['max3'] ?? 0),
        'mg' => (int)($r['max4'] ?? 0),
    ];
}
saveJson($OUT, 'titles', $titles, $summary);

/* ============ 7. 技能 data/jnxx.php ============ */
$skillsRaw = include $WWW . '/fqxy/data/jnxx.php';
$skills = [];
foreach ($skillsRaw as $s) {
    $skills[] = [
        'skill_id' => (int)$s['jnid'],
        'category' => (int)$s['jnfl'],
        'name' => clean($s['jnmz']),
        'desc' => clean($s['jnms']),
        'multiplier' => (float)$s['shxs'],
    ];
}
saveJson($OUT, 'skills', $skills, $summary);

/* ============ 8. 物品/装备/任务 xyy.sql ============ */
$sqlText = file_get_contents($WWW . '/data/xyy.sql');

function parseInsertRows($sql, $table) {
    $rows = [];
    if (!preg_match_all('/INSERT INTO `' . $table . '` VALUES\s*([\s\S]*?);\s*(?:\r|\n|$)/', $sql, $ms)) return $rows;
    foreach ($ms[1] as $valuesBlob) {
        $len = strlen($valuesBlob);
        $i = 0; $fields = []; $cur = ''; $inStr = false;
        while ($i < $len) {
            $ch = $valuesBlob[$i];
            if ($inStr) {
                if ($ch == '\\') { $cur .= $ch . ($i + 1 < $len ? $valuesBlob[$i + 1] : ''); $i += 2; continue; }
                if ($ch == "'") {
                    // 可能是转义 ''
                    if ($i + 1 < $len && $valuesBlob[$i + 1] == "'") { $cur .= "''"; $i += 2; continue; }
                    $inStr = false; $cur .= $ch; $i++; continue;
                }
                $cur .= $ch; $i++; continue;
            }
            if ($ch == "'") { $inStr = true; $cur .= $ch; $i++; continue; }
            if ($ch == ',') { $fields[] = $cur; $cur = ''; $i++; continue; }
            if ($ch == '(') { $fields = []; $cur = ''; $i++; continue; }
            if ($ch == ')') {
                if (trim($cur) !== '' || count($fields) > 0) $fields[] = $cur;
                if (count($fields) > 0) $rows[] = $fields;
                $fields = []; $cur = ''; $i++; continue;
            }
            $cur .= $ch; $i++;
        }
    }
    return $rows;
}

function unquote($v) {
    $v = trim($v);
    if (strlen($v) >= 2 && $v[0] == "'" && substr($v, -1) == "'") {
        $v = substr($v, 1, -1);
        $v = str_replace(["\\'", '\\\\'], ["'", '\\'], $v);
    }
    return $v;
}
function toInt($v) { return (int)unquote($v); }

/* 物品 wpxx: id, mz, ms, fl, jd, jg, dj, zl, bd */
$items = [];
foreach (parseInsertRows($sqlText, 'wpxx') as $r) {
    if (count($r) < 9) continue;
    $items[] = [
        'item_id' => toInt($r[0]), 'name' => clean(unquote($r[1])), 'desc' => clean(unquote($r[2])),
        'category' => toInt($r[3]), 'bean_price' => toInt($r[4]), 'price' => toInt($r[5]),
        'level' => toInt($r[6]), 'weight' => toInt($r[7]), 'bind' => toInt($r[8]),
    ];
}
saveJson($OUT, 'items', $items, $summary);

/* 装备 zbxx: id, mz, ms, hp, gj, mg, fy, bg, hg, lg, bf, hf, lf, dj, zl, bd, jd, jg, pd, mp, fl */
$equips = [];
foreach (parseInsertRows($sqlText, 'zbxx') as $r) {
    if (count($r) < 21) continue;
    $equips[] = [
        'equip_id' => toInt($r[0]), 'name' => clean(unquote($r[1])), 'desc' => clean(unquote($r[2])),
        'hp' => toInt($r[3]), 'atk' => toInt($r[4]), 'mg' => toInt($r[5]), 'def' => toInt($r[6]),
        'bg' => toInt($r[7]), 'hg' => toInt($r[8]), 'lg' => toInt($r[9]),
        'bf' => toInt($r[10]), 'hf' => toInt($r[11]), 'lf' => toInt($r[12]),
        'level' => toInt($r[13]), 'weight' => toInt($r[14]), 'bind' => toInt($r[15]),
        'bean_price' => toInt($r[16]), 'price' => toInt($r[17]), 'slot' => toInt($r[18]),
        'sect' => toInt($r[19]), 'category' => toInt($r[20]),
    ];
}
saveJson($OUT, 'equips', $equips, $summary);

/* ============ 9. NPC 刷怪表 map/*.php ============ */
// dtx -> 地图文件名（来自 mapid.php）
$regionMap = [
    0 => 'xsc', 1 => 'cac', 2 => 'lg', 3 => 'hdml', 4 => 'hd', 5 => 'jsh', 6 => 'hdmlsc', 7 => 'yg',
    8 => 'ghl', 9 => 'xl', 10 => 'fcs', 11 => 'sl', 12 => 'xgt', 13 => 'pts', 14 => 'zzbl', 15 => 'zzl',
    16 => 'hy', 17 => 'dyt', 18 => 'xyt', 19 => 'bmy', 20 => 'bl', 21 => 'gjz', 22 => 'df', 23 => 'tg',
    24 => 'alg', 25 => 'bxg', 26 => 'wjg', 27 => 'ccg', 28 => 'neg', 29 => 'jsg', 30 => 'zzg', 31 => 'yh',
    32 => 'pty', 33 => 'zyt', 34 => 'ljl', 35 => 'scl', 36 => 'hsl', 37 => 'jt', 38 => 'bgd', 39 => 'pds',
    40 => 'jjg', 41 => 'yls', 42 => 'tl', 43 => 'bfg', 44 => 'xsmg', 45 => 'byzzl', 46 => 'bglm', 47 => 'wsc',
    48 => 'yc', 49 => 'bjt', 50 => 'byd', 51 => 'yld', 52 => 'lhd', 53 => 'jdd', 54 => 'jds', 55 => 'cys',
    56 => 'xlys', 57 => 'psd', 58 => 'qls', 59 => 'jjl', 60 => 'bqg', 61 => 'tfg', 62 => 'fxq', 63 => 'yhq',
    64 => 'jpf', 65 => 'tzg', 66 => 'ygd', 67 => 'zgz', 68 => 'xyj', 69 => 'dyj', 70 => 'kfgc', 71 => 'mz',
    72 => 'hz', 73 => 'gzai', 74 => 'gczc', 75 => 'vipqy', 76 => 'emgc', 77 => 'cwd', 78 => 'ttt', 79 => 'dy18',
    80 => 'bjd', 81 => 'ttsf', 82 => 'psdd', 83 => 'std', 84 => 'wdd', 85 => 'bhmj', 86 => 'wxz01',
    87 => 'vip1qy', 88 => 'vip2qy', 89 => 'vip3qy', 90 => 'vip4qy',
];
$spawns = [];
$seen = [];
foreach ($regionMap as $dtx => $base) {
    $f = $WWW . '/fqxy/map/' . $base . '.php';
    if (!file_exists($f)) continue;
    $c = file_get_contents($f);
    // 节点分支位置: $dty==N
    $branches = [];
    if (preg_match_all('/\$dty\s*==\s*[\'"]?(\d+)/', $c, $bm, PREG_OFFSET_CAPTURE)) {
        foreach ($bm[1] as $b) $branches[] = [(int)$b[0], (int)$b[1]]; // [值, 位置]
    }
    // 攻击链接: $clj[]=10; ... $npc[]=N; ... 蓝色链接名
    if (!preg_match_all('/\$clj\[\]\s*=\s*10\s*;[\s\S]{0,400}?\$npc\[\]\s*=\s*(\d+)[\s\S]{0,400}?color=blue[^>]*>([^<]*)<\/font>/u', $c, $am, PREG_OFFSET_CAPTURE)) continue;
    foreach ($am[1] as $idx => $m1) {
        $npcId = (int)$m1[0];
        if ($npcId <= 0) continue;
        $pos = $m1[1];
        $dty = 0;
        foreach ($branches as $b) { if ($b[1] < $pos) $dty = $b[0]; }
        $name = clean($am[2][$idx][0]);
        $name = preg_replace('/^(&nbsp;?|\s)+/u', '', $name);
        $name = str_replace('&nbsp;', '', $name);
        $diff = '普通';
        if (preg_match('/（(.+?)）|\((.+?)\)/u', $name, $dm)) {
            $diff = $dm[1] ?: $dm[2];
            $name = trim(preg_replace('/（(.+?)）|\((.+?)\)/u', '', $name));
        }
        $key = $dtx . '_' . $dty . '_' . $npcId . '_' . $diff;
        if (isset($seen[$key])) continue;
        $seen[$key] = 1;
        $spawns[] = ['dtx' => $dtx, 'dty' => $dty, 'npc_id' => $npcId, 'name' => $name, 'difficulty' => $diff];
    }
}
saveJson($OUT, 'spawns', $spawns, $summary);

/* ============ 摘要输出 ============ */
echo "=== 幻想西游数据提取完成 ===\n";
foreach ($summary as $s) echo $s . "\n";

// 物品分类分布（供商店/效果规则设计）
$catCount = [];
foreach ($items as $it) { $c = $it['category']; $catCount[$c] = ($catCount[$c] ?? 0) + 1; }
echo "物品分类分布: " . json_encode($catCount) . "\n";
$catCount = [];
foreach ($equips as $it) { $c = $it['slot']; $catCount[$c] = ($catCount[$c] ?? 0) + 1; }
echo "装备部位分布: " . json_encode($catCount) . "\n";
$catCount = [];
foreach ($equips as $it) { $c = $it['category']; $catCount[$c] = ($catCount[$c] ?? 0) + 1; }
echo "装备类别分布: " . json_encode($catCount) . "\n";
