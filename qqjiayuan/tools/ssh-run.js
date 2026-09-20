#!/usr/bin/env node
/**
 * ssh-run.js —— 用密码登录远程服务器执行命令 / 传文件
 * （Windows git-bash 没有 sshpass，这个脚本是它的替代品）
 *
 * 用法：
 *   node tools/ssh-run.js exec "命令"
 *   node tools/ssh-run.js put  本地文件  远程路径
 *
 * 环境变量（都有默认值，可用 SSH_HOST/SSH_USER/SSH_PASS 覆盖）：
 *
 * ★ 关键坑（踩过，别改成喂 stdin 那种写法）：
 *   ssh 在**没有 tty** 时不会从 stdin 读密码，而是走 SSH_ASKPASS 机制
 *   —— 也就是说必须设置 SSH_ASKPASS 指向一个「打印密码」的脚本，
 *   并且 SSH_ASKPASS_REQUIRE=force + 有 DISPLAY，ssh 才会去调用它。
 *   单纯 spawn('ssh') 然后 p.stdin.write(password) 是完全无效的，
 *   表现为「直接 Permission denied」，看起来像密码错，其实根本没送出去。
 */
const { spawn } = require('child_process');
const fs = require('fs');
const os = require('os');
const path = require('path');

const HOST = process.env.SSH_HOST || '39.105.151.141';
const USER = process.env.SSH_USER || 'root';
const PASS = process.env.SSH_PASS || 'Admin123mzd..';

const NOISE = /post-quantum|decrypt later|may need to be upgraded|openssh\.com\/pq/i;

// 生成临时 askpass 脚本（打印密码）。进程退出时删掉。
const ASKPASS = path.join(os.tmpdir(), `askpass-${process.pid}.sh`);
fs.writeFileSync(ASKPASS, `#!/bin/sh\nprintf '%s\\n' '${PASS.replace(/'/g, "'\\''")}'\n`);
fs.chmodSync(ASKPASS, 0o755);
const cleanup = () => { try { fs.unlinkSync(ASKPASS); } catch (e) {} };
process.on('exit', cleanup);
process.on('SIGINT', () => { cleanup(); process.exit(130); });

function baseOpts() {
  return [
    '-o', 'StrictHostKeyChecking=no',
    '-o', 'UserKnownHostsFile=/dev/null',
    '-o', 'LogLevel=ERROR',
    '-o', 'ConnectTimeout=20',
    '-o', 'PreferredAuthentications=password',
    '-o', 'PubkeyAuthentication=no',
    '-o', 'NumberOfPasswordPrompts=1',
  ];
}

function env() {
  return { ...process.env, SSH_ASKPASS: ASKPASS, SSH_ASKPASS_REQUIRE: 'force', DISPLAY: process.env.DISPLAY || ':0' };
}

function main() {
  const [, , mode, ...rest] = process.argv;

  if (mode === 'exec') {
    const cmd = rest.join(' ');
    const p = spawn('ssh', ['-T', ...baseOpts(), `${USER}@${HOST}`, cmd], { env: env(), stdio: ['ignore', 'pipe', 'pipe'] });
    p.stdout.on('data', d => process.stdout.write(d));
    p.stderr.on('data', d => { const s = d.toString(); if (!NOISE.test(s)) process.stderr.write(d); });
    p.on('close', code => { cleanup(); process.exit(code === 0 ? 0 : (code || 1)); });
    return;
  }

  if (mode === 'put') {
    const [local, remote] = rest;
    if (!local || !remote) { console.error('用法: node tools/ssh-run.js put <本地> <远程>'); process.exit(2); }
    if (!fs.existsSync(local)) { console.error('本地文件不存在: ' + local); process.exit(2); }
    const size = fs.statSync(local).size;
    console.log(`[scp] 上传 ${path.basename(local)} (${(size / 1048576).toFixed(1)} MB) → ${remote}`);
    const p = spawn('scp', [...baseOpts(), local, `${USER}@${HOST}:${remote}`], { env: env(), stdio: ['ignore', 'pipe', 'pipe'] });
    p.stdout.on('data', d => process.stdout.write(d));
    p.stderr.on('data', d => {
      const s = d.toString();
      if (NOISE.test(s)) return;
      // scp 无进度条输出，只打印错误
      if (/lost connection|denied|No such|error/i.test(s)) process.stderr.write(d);
    });
    p.on('close', code => { cleanup(); console.log(`[scp] 完成 exit=${code}`); process.exit(code === 0 ? 0 : 1); });
    return;
  }

  console.error('用法:\n  node tools/ssh-run.js exec "命令"\n  node tools/ssh-run.js put <本地文件> <远程路径>');
  process.exit(2);
}

main();
