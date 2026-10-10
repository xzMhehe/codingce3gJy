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

const IS_WIN = process.platform === 'win32';

// ★★ 2026-10-07 关键修复：Windows 上必须用 **.exe** 形式的 askpass。
//
//   背景：原来是在临时目录生成一个 .sh 脚本当 SSH_ASKPASS，这在 git-bash 里能跑
//   （MSYS2 ssh 内部有 /bin/sh），但从 PowerShell / cmd 启动时必然失败 ——
//   Windows 侧启动 askpass 走 CreateProcessW，**执行不了 .sh / .cmd / .bat**：
//     · git 自带的 MSYS2 ssh → CreateProcessW failed error:193
//                              + ssh_askpass: posix_spawnp: Unknown error
//     · Windows 原生 OpenSSH → ssh_askpass: pipe: Unknown error
//   两种表现都是密码没送出去，最终退化成 Permission denied (publickey,...,password)。
//
//   实测结论（2026-10-07）：
//     MSYS2 ssh + askpass.exe → 成功；Windows 原生 ssh 的 askpass 机制本身不可用。
//   所以 Windows 统一改用 tools/askpass.exe（Go 编译，见 askpass.go），
//   密码通过环境变量 SSH_ASKPASS_PW 传进去，命令行里不出现明文。
const ASKPASS_EXE = path.join(__dirname, 'askpass.exe');
let ASKPASS = ASKPASS_EXE;
const ASKPASS_ENV = {};

if (IS_WIN) {
  if (!fs.existsSync(ASKPASS_EXE)) {
    console.error('缺少 ' + ASKPASS_EXE);
    console.error('请先在 tools/ 目录执行: go build -o askpass.exe askpass.go');
    process.exit(2);
  }
  ASKPASS_ENV.SSH_ASKPASS_PW = PASS;
} else {
  // 非 Windows（Linux / macOS / git-bash 下跑）沿用临时 .sh 脚本
  ASKPASS = path.join(os.tmpdir(), `askpass-${process.pid}.sh`);
  fs.writeFileSync(ASKPASS, `#!/bin/sh\nprintf '%s\\n' '${PASS.replace(/'/g, "'\\''")}'\n`);
  fs.chmodSync(ASKPASS, 0o755);
}

// exe 是常驻文件，不能删；只有临时 .sh 需要清理。
const cleanup = () => {
  if (IS_WIN) return;
  try { fs.unlinkSync(ASKPASS); } catch (e) {}
};
process.on('exit', cleanup);
process.on('SIGINT', () => { cleanup(); process.exit(130); });

function baseOpts() {
  return [
    '-o', 'StrictHostKeyChecking=no',
    '-o', 'UserKnownHostsFile=/dev/null',
    '-o', 'LogLevel=ERROR',
    '-o', 'ConnectTimeout=20',
    // ★ 2026-10-10 连接阶段有超时, 但「命令执行中」没有——
    //   服务启动链(Ensure* 跨 WAN 跑 DDL, 10~40s)期间服务器忙, ssh 会话会一直挂着,
    //   node 永不退出 → deploy.sh 卡死。加心跳: 每 15s 探测一次, 4 次无响应(≈60s)断开
    '-o', 'ServerAliveInterval=15',
    '-o', 'ServerAliveCountMax=4',
    '-o', 'PreferredAuthentications=password',
    '-o', 'PubkeyAuthentication=no',
    '-o', 'NumberOfPasswordPrompts=1',
  ];
}

// ★ 2026-10-10 整体超时兜底: 心跳靠服务器响应, 但万一 sshd 卡死/网络黑洞,
//   心跳也不可靠。每条命令(exec)最多 30s、传包(put)最多 300s, 超时强杀并退出,
//   绝不让部署脚本永久挂住。
const EXEC_TIMEOUT_MS = 30_000;
const PUT_TIMEOUT_MS = 300_000;

function armTimeout(p, ms, tag) {
  const timer = setTimeout(() => {
    try {
      p.kill('SIGTERM');
      setTimeout(() => { try { p.kill('SIGKILL'); } catch (e) {} }, 3000).unref();
      console.error(`[超时] ${tag} 超过 ${ms / 1000}s 已终止(远端可能仍在执行, 可手动重跑 deploy.sh)`);
    } catch (e) {}
  }, ms);
  return timer;
}

function env() {
  return {
    ...process.env,
    SSH_ASKPASS: ASKPASS,
    SSH_ASKPASS_REQUIRE: 'force',
    DISPLAY: process.env.DISPLAY || ':0',
    ...ASKPASS_ENV,   // Windows: 把密码经 SSH_ASKPASS_PW 传给 askpass.exe
  };
}

function main() {
  const [, , mode, ...rest] = process.argv;

  if (mode === 'exec') {
    const cmd = rest.join(' ');
    const p = spawn('ssh', ['-T', ...baseOpts(), `${USER}@${HOST}`, cmd], { env: env(), stdio: ['ignore', 'pipe', 'pipe'] });
    const timer = armTimeout(p, EXEC_TIMEOUT_MS, `ssh exec: ${cmd.slice(0, 60)}`);
    p.stdout.on('data', d => process.stdout.write(d));
    p.stderr.on('data', d => { const s = d.toString(); if (!NOISE.test(s)) process.stderr.write(d); });
    p.on('close', code => { clearTimeout(timer); cleanup(); process.exit(code === 0 ? 0 : (code || 1)); });
    return;
  }

  if (mode === 'put') {
    const [local, remote] = rest;
    if (!local || !remote) { console.error('用法: node tools/ssh-run.js put <本地> <远程>'); process.exit(2); }
    if (!fs.existsSync(local)) { console.error('本地文件不存在: ' + local); process.exit(2); }
    const size = fs.statSync(local).size;
    console.log(`[scp] 上传 ${path.basename(local)} (${(size / 1048576).toFixed(1)} MB) → ${remote}`);
    const p = spawn('scp', [...baseOpts(), local, `${USER}@${HOST}:${remote}`], { env: env(), stdio: ['ignore', 'pipe', 'pipe'] });
    const timer = armTimeout(p, PUT_TIMEOUT_MS, `scp: ${path.basename(local)}`);
    p.stdout.on('data', d => process.stdout.write(d));
    p.stderr.on('data', d => {
      const s = d.toString();
      if (NOISE.test(s)) return;
      // scp 无进度条输出，只打印错误
      if (/lost connection|denied|No such|error/i.test(s)) process.stderr.write(d);
    });
    p.on('close', code => { clearTimeout(timer); cleanup(); console.log(`[scp] 完成 exit=${code}`); process.exit(code === 0 ? 0 : 1); });
    return;
  }

  console.error('用法:\n  node tools/ssh-run.js exec "命令"\n  node tools/ssh-run.js put <本地文件> <远程路径>');
  process.exit(2);
}

main();
