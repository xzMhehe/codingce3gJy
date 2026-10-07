// askpass.exe —— 给 Windows 原生 ssh 用的 SSH_ASKPASS 程序。
//
// 背景（2026-10-07 排查结论）：
//   tools/ssh-run.js 原本用「生成一个 .sh 脚本当 SSH_ASKPASS」的办法喂密码，
//   这在 git-bash 的 MSYS2 ssh 下能工作，但在 **Windows 原生 ssh.exe**
//   （C:\Windows\System32\OpenSSH\ssh.exe）下必然失败 ——
//   Windows OpenSSH 用 CreateProcessW 启动 askpass，**只能执行 .exe**：
//     · .sh   → CreateProcessW failed error:193 / posix_spawnp: Unknown error
//     · .cmd  → ssh_askpass: pipe: Unknown error（实测同样不行）
//   结果密码根本没送出去，退化成 Permission denied (publickey,...,password)。
//
// 本程序只做一件事：把环境变量 SSH_ASKPASS_PW 的内容打印到 stdout。
// 密码走环境变量传递（askpass 继承 ssh 的环境，ssh 又继承 node 的环境），
// 这样命令行里不会出现明文密码。
//
// 编译（在 tools/ 目录下）：
//   go build -o askpass.exe askpass.go
package main

import (
	"fmt"
	"os"
)

func main() {
	// ssh 会带一个提示串作为参数（如 "root@host's password: "），这里忽略即可。
	fmt.Println(os.Getenv("SSH_ASKPASS_PW"))
}
